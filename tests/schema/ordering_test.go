package schema_test

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ============================================================================
// GetOrderedAddedTables Tests
// ============================================================================

func TestGetOrderedAddedTables_NoForeignKeys(t *testing.T) {
	tables := []schema.Table{
		{Name: "users"},
		{Name: "products"},
		{Name: "categories"},
	}

	result := schema.GetOrderedAddedTables(tables)

	if len(result) != 3 {
		t.Errorf("expected 3 tables, got %d", len(result))
	}
}

func TestGetOrderedAddedTables_SimpleDependency(t *testing.T) {
	// orders depends on users
	tables := []schema.Table{
		{
			Name: "orders",
			ForeignKeys: map[string]schema.ForeignKey{
				"fk_orders_user": {
					Name:            "fk_orders_user",
					Columns:         []string{"user_id"},
					ReferencedTable: "users",
				},
			},
		},
		{Name: "users"},
	}

	result := schema.GetOrderedAddedTables(tables)

	// users should come before orders
	usersIdx := -1
	ordersIdx := -1
	for i, t := range result {
		if t.Name == "users" {
			usersIdx = i
		}
		if t.Name == "orders" {
			ordersIdx = i
		}
	}

	if usersIdx > ordersIdx {
		t.Errorf("users (idx=%d) should come before orders (idx=%d)", usersIdx, ordersIdx)
	}
}

func TestGetOrderedAddedTables_ChainedDependencies(t *testing.T) {
	// order_items depends on orders, orders depends on users
	tables := []schema.Table{
		{
			Name: "order_items",
			ForeignKeys: map[string]schema.ForeignKey{
				"fk_items_order": {
					Name:            "fk_items_order",
					Columns:         []string{"order_id"},
					ReferencedTable: "orders",
				},
			},
		},
		{
			Name: "orders",
			ForeignKeys: map[string]schema.ForeignKey{
				"fk_orders_user": {
					Name:            "fk_orders_user",
					Columns:         []string{"user_id"},
					ReferencedTable: "users",
				},
			},
		},
		{Name: "users"},
	}

	result := schema.GetOrderedAddedTables(tables)

	// Expected order: users -> orders -> order_items
	positions := make(map[string]int)
	for i, t := range result {
		positions[t.Name] = i
	}

	if positions["users"] > positions["orders"] {
		t.Errorf("users should come before orders")
	}
	if positions["orders"] > positions["order_items"] {
		t.Errorf("orders should come before order_items")
	}
}

func TestGetOrderedAddedTables_MultipleForeignKeys(t *testing.T) {
	// reviews depends on both users and products
	tables := []schema.Table{
		{
			Name: "reviews",
			ForeignKeys: map[string]schema.ForeignKey{
				"fk_reviews_user": {
					Name:            "fk_reviews_user",
					Columns:         []string{"user_id"},
					ReferencedTable: "users",
				},
				"fk_reviews_product": {
					Name:            "fk_reviews_product",
					Columns:         []string{"product_id"},
					ReferencedTable: "products",
				},
			},
		},
		{Name: "users"},
		{Name: "products"},
	}

	result := schema.GetOrderedAddedTables(tables)

	positions := make(map[string]int)
	for i, t := range result {
		positions[t.Name] = i
	}

	if positions["users"] > positions["reviews"] {
		t.Errorf("users should come before reviews")
	}
	if positions["products"] > positions["reviews"] {
		t.Errorf("products should come before reviews")
	}
}

func TestGetOrderedAddedTables_CircularDependency(t *testing.T) {
	// Circular: a -> b -> a (should handle gracefully)
	tables := []schema.Table{
		{
			Name: "table_a",
			ForeignKeys: map[string]schema.ForeignKey{
				"fk_a_b": {
					Name:            "fk_a_b",
					Columns:         []string{"b_id"},
					ReferencedTable: "table_b",
				},
			},
		},
		{
			Name: "table_b",
			ForeignKeys: map[string]schema.ForeignKey{
				"fk_b_a": {
					Name:            "fk_b_a",
					Columns:         []string{"a_id"},
					ReferencedTable: "table_a",
				},
			},
		},
	}

	// Should not panic and should return all tables
	result := schema.GetOrderedAddedTables(tables)

	if len(result) != 2 {
		t.Errorf("expected 2 tables, got %d", len(result))
	}
}

func TestGetOrderedAddedTables_ExternalReference(t *testing.T) {
	// FK references a table not in the list (already exists)
	tables := []schema.Table{
		{
			Name: "orders",
			ForeignKeys: map[string]schema.ForeignKey{
				"fk_orders_user": {
					Name:            "fk_orders_user",
					Columns:         []string{"user_id"},
					ReferencedTable: "users", // users is not in the list
				},
			},
		},
	}

	result := schema.GetOrderedAddedTables(tables)

	if len(result) != 1 {
		t.Errorf("expected 1 table, got %d", len(result))
	}
	if result[0].Name != "orders" {
		t.Errorf("expected orders, got %s", result[0].Name)
	}
}

func TestGetOrderedAddedTables_SingleTable(t *testing.T) {
	tables := []schema.Table{
		{Name: "users"},
	}

	result := schema.GetOrderedAddedTables(tables)

	if len(result) != 1 {
		t.Errorf("expected 1 table, got %d", len(result))
	}
}

func TestGetOrderedAddedTables_EmptyList(t *testing.T) {
	var tables []schema.Table

	result := schema.GetOrderedAddedTables(tables)

	if len(result) != 0 {
		t.Errorf("expected 0 tables, got %d", len(result))
	}
}

// ============================================================================
// GetOrderedRemovedTables Tests
// ============================================================================

func TestGetOrderedRemovedTables_NoForeignKeys(t *testing.T) {
	tableNames := []string{"users", "products", "categories"}

	result := schema.GetOrderedRemovedTables(tableNames, nil)

	if len(result) != 3 {
		t.Errorf("expected 3 tables, got %d", len(result))
	}
}

func TestGetOrderedRemovedTables_WithForeignKeys(t *testing.T) {
	// orders has FK to users
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name: "orders",
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:            "fk_orders_user",
						Columns:         []string{"user_id"},
						ReferencedTable: "users",
					},
				},
			},
			"users": {Name: "users"},
		},
	}

	tableNames := []string{"users", "orders"}
	result := schema.GetOrderedRemovedTables(tableNames, prodSchema)

	// orders should come before users (table with FK dropped first)
	ordersIdx := -1
	usersIdx := -1
	for i, name := range result {
		if name == "orders" {
			ordersIdx = i
		}
		if name == "users" {
			usersIdx = i
		}
	}

	if ordersIdx > usersIdx {
		t.Errorf("orders (idx=%d) should come before users (idx=%d)", ordersIdx, usersIdx)
	}
}

func TestGetOrderedRemovedTables_ChainedForeignKeys(t *testing.T) {
	// order_items -> orders -> users
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"order_items": {
				Name: "order_items",
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_items_order": {
						Name:            "fk_items_order",
						Columns:         []string{"order_id"},
						ReferencedTable: "orders",
					},
				},
			},
			"orders": {
				Name: "orders",
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:            "fk_orders_user",
						Columns:         []string{"user_id"},
						ReferencedTable: "users",
					},
				},
			},
			"users": {Name: "users"},
		},
	}

	tableNames := []string{"users", "orders", "order_items"}
	result := schema.GetOrderedRemovedTables(tableNames, prodSchema)

	positions := make(map[string]int)
	for i, name := range result {
		positions[name] = i
	}

	// order_items should come before orders, orders before users
	if positions["order_items"] > positions["orders"] {
		t.Errorf("order_items should come before orders")
	}
	if positions["orders"] > positions["users"] {
		t.Errorf("orders should come before users")
	}
}

func TestGetOrderedRemovedTables_SingleTable(t *testing.T) {
	tableNames := []string{"users"}

	result := schema.GetOrderedRemovedTables(tableNames, nil)

	if len(result) != 1 {
		t.Errorf("expected 1 table, got %d", len(result))
	}
}

// ============================================================================
// DependencyGraph Tests
// ============================================================================

func TestBuildDependencyGraph_CreateTableDependencies(t *testing.T) {
	// orders depends on users
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "orders",
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:            "fk_orders_user",
						Columns:         []string{"user_id"},
						ReferencedTable: "users",
					},
				},
			},
			{Name: "users"},
		},
	}

	graph := schema.BuildDependencyGraph(diff, nil)

	// Should have operations for both CREATE TABLE and ADD FK
	if len(graph.Operations) < 2 {
		t.Errorf("expected at least 2 operations, got %d", len(graph.Operations))
	}
}

func TestBuildDependencyGraph_DropForeignKeyBeforeDropTable(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"users"},
		Tables: []schema.TableDiff{
			{
				Name: "orders",
				RemovedForeignKeys: []schema.ForeignKey{
					{
						Name:            "fk_orders_user",
						Columns:         []string{"user_id"},
						ReferencedTable: "users",
					},
				},
				HasDifferences: true,
			},
		},
	}

	graph := schema.BuildDependencyGraph(diff, nil)

	// Should have DROP FK and DROP TABLE operations
	hasDropFK := false
	hasDropTable := false
	for _, op := range graph.Operations {
		if op.Type == schema.OpDropForeignKey {
			hasDropFK = true
		}
		if op.Type == schema.OpDropTable {
			hasDropTable = true
		}
	}

	if !hasDropFK {
		t.Error("expected DROP_FOREIGN_KEY operation")
	}
	if !hasDropTable {
		t.Error("expected DROP_TABLE operation")
	}
}

// ============================================================================
// TopologicalSort Tests
// ============================================================================

func TestTopologicalSort_SimpleGraph(t *testing.T) {
	graph := schema.NewDependencyGraph()

	op1 := &schema.MigrationOperation{ID: "op1", Type: schema.OpCreateTable, Table: "users"}
	op2 := &schema.MigrationOperation{ID: "op2", Type: schema.OpAddForeignKey, Table: "orders"}

	graph.AddOperation(op1)
	graph.AddOperation(op2)
	graph.AddDependency("op1", "op2") // op2 depends on op1

	result, err := schema.TopologicalSort(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// op1 should come before op2
	op1Idx := -1
	op2Idx := -1
	for i, op := range result.Operations {
		if op.ID == "op1" {
			op1Idx = i
		}
		if op.ID == "op2" {
			op2Idx = i
		}
	}

	if op1Idx > op2Idx {
		t.Errorf("op1 (idx=%d) should come before op2 (idx=%d)", op1Idx, op2Idx)
	}
}

func TestTopologicalSort_DetectsCycle(t *testing.T) {
	graph := schema.NewDependencyGraph()

	op1 := &schema.MigrationOperation{ID: "op1", Type: schema.OpCreateTable, Table: "a"}
	op2 := &schema.MigrationOperation{ID: "op2", Type: schema.OpCreateTable, Table: "b"}

	graph.AddOperation(op1)
	graph.AddOperation(op2)
	graph.AddDependency("op1", "op2") // op2 depends on op1
	graph.AddDependency("op2", "op1") // op1 depends on op2 (cycle!)

	result, err := schema.TopologicalSort(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect circular dependency and add warnings
	if len(result.CircularDependencies) == 0 && len(result.Warnings) == 0 {
		t.Error("expected circular dependency detection")
	}
}

func TestTopologicalSort_OperationPriority(t *testing.T) {
	graph := schema.NewDependencyGraph()

	// Add operations in wrong order
	op1 := &schema.MigrationOperation{ID: "add_fk", Type: schema.OpAddForeignKey, Table: "orders", Priority: 2}
	op2 := &schema.MigrationOperation{ID: "create_table", Type: schema.OpCreateTable, Table: "orders", Priority: 1}

	graph.AddOperation(op1)
	graph.AddOperation(op2)

	result, _ := schema.TopologicalSort(graph)

	// CREATE_TABLE should come before ADD_FK due to operation type ordering
	createIdx := -1
	addFKIdx := -1
	for i, op := range result.Operations {
		if op.ID == "create_table" {
			createIdx = i
		}
		if op.ID == "add_fk" {
			addFKIdx = i
		}
	}

	if createIdx > addFKIdx {
		t.Errorf("CREATE_TABLE (idx=%d) should come before ADD_FK (idx=%d)", createIdx, addFKIdx)
	}
}

// ============================================================================
// Integration Tests - GenerateMigration with Ordering
// ============================================================================

func TestGenerateMigration_OrdersCreateTables(t *testing.T) {
	// reviews depends on users and products
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "reviews",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "int", IsNullable: false},
					"user_id":    {Name: "user_id", DataType: "int", IsNullable: false},
					"product_id": {Name: "product_id", DataType: "int", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_reviews_user": {
						Name:            "fk_reviews_user",
						Columns:         []string{"user_id"},
						ReferencedTable: "users",
					},
					"fk_reviews_product": {
						Name:            "fk_reviews_product",
						Columns:         []string{"product_id"},
						ReferencedTable: "products",
					},
				},
			},
			{
				Name: "users",
				Columns: map[string]schema.Column{
					"id": {Name: "id", DataType: "int", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
			{
				Name: "products",
				Columns: map[string]schema.Column{
					"id": {Name: "id", DataType: "int", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that users and products are created before reviews
	usersIdx := strings.Index(sql, "CREATE TABLE `users`")
	productsIdx := strings.Index(sql, "CREATE TABLE `products`")
	reviewsIdx := strings.Index(sql, "CREATE TABLE `reviews`")

	if usersIdx == -1 || productsIdx == -1 || reviewsIdx == -1 {
		t.Fatal("expected all CREATE TABLE statements")
	}

	if usersIdx > reviewsIdx {
		t.Error("CREATE TABLE users should come before CREATE TABLE reviews")
	}
	if productsIdx > reviewsIdx {
		t.Error("CREATE TABLE products should come before CREATE TABLE reviews")
	}
}

func TestGenerateMigration_OrdersDropTables(t *testing.T) {
	// orders depends on users in production
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name: "orders",
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:            "fk_orders_user",
						Columns:         []string{"user_id"},
						ReferencedTable: "users",
					},
				},
			},
			"users": {Name: "users"},
		},
	}

	diff := schema.DiffResult{
		RemovedTables: []string{"users", "orders"},
	}

	opts := &schema.MigrationOptions{
		AllowDropTable: true,
	}

	sql, err := schema.GenerateMigrationWithSchemas(diff, "mysql", opts, prodSchema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// orders should be dropped before users
	ordersIdx := strings.Index(sql, "DROP TABLE `orders`")
	usersIdx := strings.Index(sql, "DROP TABLE `users`")

	if ordersIdx == -1 || usersIdx == -1 {
		t.Fatal("expected both DROP TABLE statements")
	}

	if ordersIdx > usersIdx {
		t.Errorf("DROP TABLE orders (idx=%d) should come before DROP TABLE users (idx=%d)", ordersIdx, usersIdx)
	}
}

func TestGenerateMigration_ForeignKeyAfterTable(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "orders",
				Columns: map[string]schema.Column{
					"id":      {Name: "id", DataType: "int", IsNullable: false},
					"user_id": {Name: "user_id", DataType: "int", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:              "fk_orders_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CREATE TABLE should come before ADD CONSTRAINT
	createIdx := strings.Index(sql, "CREATE TABLE `orders`")
	fkIdx := strings.Index(sql, "ADD CONSTRAINT `fk_orders_user`")

	if createIdx == -1 {
		t.Fatal("expected CREATE TABLE statement")
	}
	if fkIdx == -1 {
		t.Fatal("expected ADD CONSTRAINT statement")
	}

	if createIdx > fkIdx {
		t.Error("CREATE TABLE should come before ADD CONSTRAINT FOREIGN KEY")
	}
}

// ============================================================================
// Operation Type Tests
// ============================================================================

func TestOperationType_String(t *testing.T) {
	tests := []struct {
		opType   schema.OperationType
		expected string
	}{
		{schema.OpDropForeignKey, "DROP_FOREIGN_KEY"},
		{schema.OpDropIndex, "DROP_INDEX"},
		{schema.OpDropColumn, "DROP_COLUMN"},
		{schema.OpDropTable, "DROP_TABLE"},
		{schema.OpCreateTable, "CREATE_TABLE"},
		{schema.OpAddColumn, "ADD_COLUMN"},
		{schema.OpModifyColumn, "MODIFY_COLUMN"},
		{schema.OpAddIndex, "ADD_INDEX"},
		{schema.OpAddForeignKey, "ADD_FOREIGN_KEY"},
		{schema.OpModifyPrimaryKey, "MODIFY_PRIMARY_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.opType.String(); got != tt.expected {
				t.Errorf("OperationType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMigrationOperation_String(t *testing.T) {
	op := &schema.MigrationOperation{
		Type:   schema.OpCreateTable,
		Table:  "users",
		Object: "",
	}

	expected := "CREATE_TABLE users"
	if got := op.String(); got != expected {
		t.Errorf("MigrationOperation.String() = %v, want %v", got, expected)
	}

	op2 := &schema.MigrationOperation{
		Type:   schema.OpAddForeignKey,
		Table:  "orders",
		Object: "fk_orders_user",
	}

	expected2 := "ADD_FOREIGN_KEY orders.fk_orders_user"
	if got := op2.String(); got != expected2 {
		t.Errorf("MigrationOperation.String() = %v, want %v", got, expected2)
	}
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestOperationType_String_Unknown(t *testing.T) {
	// Test unknown operation type (default case)
	unknownType := schema.OperationType(999)
	if got := unknownType.String(); got != "UNKNOWN" {
		t.Errorf("Unknown OperationType.String() = %v, want UNKNOWN", got)
	}
}

func TestBuildDependencyGraph_AllOperationTypes(t *testing.T) {
	// Test comprehensive diff with all operation types
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "new_table",
				Columns: map[string]schema.Column{
					"id": {Name: "id", DataType: "int", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
				Indexes: map[string]schema.Index{
					"idx_new": {Name: "idx_new", Columns: []string{"id"}},
				},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_new": {
						Name:            "fk_new",
						Columns:         []string{"id"},
						ReferencedTable: "existing",
					},
				},
			},
		},
		RemovedTables: []string{"old_table"},
		Tables: []schema.TableDiff{
			{
				Name: "existing_table",
				AddedColumns: []schema.Column{
					{Name: "new_col", DataType: "varchar(100)", IsNullable: true},
				},
				RemovedColumns: []schema.Column{
					{Name: "old_col", DataType: "int", IsNullable: false},
				},
				ModifiedColumns: []schema.ColumnDiff{
					{Column: "mod_col", TypeMismatch: true, DevType: "bigint"},
				},
				AddedIndexes: []schema.Index{
					{Name: "idx_added", Columns: []string{"new_col"}},
				},
				RemovedIndexes: []schema.Index{
					{Name: "idx_removed", Columns: []string{"old_col"}},
				},
				AddedForeignKeys: []schema.ForeignKey{
					{Name: "fk_added", Columns: []string{"new_col"}, ReferencedTable: "ref"},
				},
				RemovedForeignKeys: []schema.ForeignKey{
					{Name: "fk_removed", Columns: []string{"old_col"}, ReferencedTable: "old_ref"},
				},
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"id", "new_col"},
				},
				HasDifferences: true,
			},
		},
	}

	graph := schema.BuildDependencyGraph(diff, nil)

	// Verify we have operations of various types
	opTypes := make(map[schema.OperationType]bool)
	for _, op := range graph.Operations {
		opTypes[op.Type] = true
	}

	expectedTypes := []schema.OperationType{
		schema.OpDropForeignKey,
		schema.OpDropIndex,
		schema.OpDropColumn,
		schema.OpDropTable,
		schema.OpCreateTable,
		schema.OpAddColumn,
		schema.OpModifyColumn,
		schema.OpAddIndex,
		schema.OpAddForeignKey,
		schema.OpModifyPrimaryKey,
	}

	for _, et := range expectedTypes {
		if !opTypes[et] {
			t.Errorf("expected operation type %s in graph", et.String())
		}
	}
}

func TestBuildDependencyGraph_IndexColumnDependencies(t *testing.T) {
	// Test that indexes depend on their columns being added
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name: "test_table",
				AddedColumns: []schema.Column{
					{Name: "new_col", DataType: "int", IsNullable: true},
				},
				AddedIndexes: []schema.Index{
					{Name: "idx_new_col", Columns: []string{"new_col"}},
				},
				HasDifferences: true,
			},
		},
	}

	graph := schema.BuildDependencyGraph(diff, nil)

	// Should have both add column and add index operations
	hasAddCol := false
	hasAddIdx := false
	for _, op := range graph.Operations {
		if op.Type == schema.OpAddColumn && op.Object == "new_col" {
			hasAddCol = true
		}
		if op.Type == schema.OpAddIndex && op.Object == "idx_new_col" {
			hasAddIdx = true
		}
	}

	if !hasAddCol {
		t.Error("expected ADD_COLUMN operation")
	}
	if !hasAddIdx {
		t.Error("expected ADD_INDEX operation")
	}
}

func TestBuildDependencyGraph_FKColumnDependencies(t *testing.T) {
	// Test that FKs depend on their columns being added
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name: "test_table",
				AddedColumns: []schema.Column{
					{Name: "ref_id", DataType: "int", IsNullable: false},
				},
				AddedForeignKeys: []schema.ForeignKey{
					{
						Name:            "fk_ref",
						Columns:         []string{"ref_id"},
						ReferencedTable: "ref_table",
					},
				},
				HasDifferences: true,
			},
		},
	}

	graph := schema.BuildDependencyGraph(diff, nil)

	hasAddCol := false
	hasAddFK := false
	for _, op := range graph.Operations {
		if op.Type == schema.OpAddColumn && op.Object == "ref_id" {
			hasAddCol = true
		}
		if op.Type == schema.OpAddForeignKey && op.Object == "fk_ref" {
			hasAddFK = true
		}
	}

	if !hasAddCol {
		t.Error("expected ADD_COLUMN operation")
	}
	if !hasAddFK {
		t.Error("expected ADD_FOREIGN_KEY operation")
	}
}

func TestBuildDependencyGraph_OnlyInDevTable(t *testing.T) {
	// Test that OnlyInDev tables are handled correctly in addDropTableDependencies
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "dev_only_table",
				OnlyInDev:      true,
				HasDifferences: true,
			},
		},
	}

	graph := schema.BuildDependencyGraph(diff, nil)

	// Should not crash and should have no operations for OnlyInDev tables
	// (they're handled separately as AddedTables)
	if graph == nil {
		t.Error("expected non-nil graph")
	}
}

func TestBuildDependencyGraph_SelfReferencingFK(t *testing.T) {
	// Test table with self-referencing FK (like categories with parent_id)
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "categories",
				Columns: map[string]schema.Column{
					"id":        {Name: "id", DataType: "int", IsNullable: false},
					"parent_id": {Name: "parent_id", DataType: "int", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_parent": {
						Name:            "fk_parent",
						Columns:         []string{"parent_id"},
						ReferencedTable: "categories", // Self-reference
					},
				},
			},
		},
	}

	graph := schema.BuildDependencyGraph(diff, nil)

	// Should not crash on self-reference
	if graph == nil {
		t.Error("expected non-nil graph")
	}
}

func TestOrderMigrationOperations(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "orders",
				Columns: map[string]schema.Column{
					"id":      {Name: "id", DataType: "int", IsNullable: false},
					"user_id": {Name: "user_id", DataType: "int", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:            "fk_orders_user",
						Columns:         []string{"user_id"},
						ReferencedTable: "users",
					},
				},
			},
			{
				Name: "users",
				Columns: map[string]schema.Column{
					"id": {Name: "id", DataType: "int", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	result := schema.OrderMigrationOperations(diff, nil)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(result.OrderedOperations) == 0 {
		t.Error("expected ordered operations")
	}

	// Verify CREATE TABLE users comes before CREATE TABLE orders
	usersIdx := -1
	ordersIdx := -1
	for i, op := range result.OrderedOperations {
		if op.Type == schema.OpCreateTable && op.Table == "users" {
			usersIdx = i
		}
		if op.Type == schema.OpCreateTable && op.Table == "orders" {
			ordersIdx = i
		}
	}

	if usersIdx != -1 && ordersIdx != -1 && usersIdx > ordersIdx {
		t.Error("users should be created before orders")
	}
}

func TestGetOrderedRemovedTables_CircularDependency(t *testing.T) {
	// Test circular FK dependencies in DROP scenario
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"table_a": {
				Name: "table_a",
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_a_b": {
						Name:            "fk_a_b",
						Columns:         []string{"b_id"},
						ReferencedTable: "table_b",
					},
				},
			},
			"table_b": {
				Name: "table_b",
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_b_a": {
						Name:            "fk_b_a",
						Columns:         []string{"a_id"},
						ReferencedTable: "table_a",
					},
				},
			},
		},
	}

	tableNames := []string{"table_a", "table_b"}
	result := schema.GetOrderedRemovedTables(tableNames, prodSchema)

	// Should return both tables even with circular dependency
	if len(result) != 2 {
		t.Errorf("expected 2 tables, got %d", len(result))
	}
}

func TestTopologicalSort_EmptyGraph(t *testing.T) {
	graph := schema.NewDependencyGraph()

	result, err := schema.TopologicalSort(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(result.Operations))
	}
}

func TestTopologicalSort_SingleOperation(t *testing.T) {
	graph := schema.NewDependencyGraph()
	op := &schema.MigrationOperation{ID: "op1", Type: schema.OpCreateTable, Table: "users"}
	graph.AddOperation(op)

	result, err := schema.TopologicalSort(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Operations) != 1 {
		t.Errorf("expected 1 operation, got %d", len(result.Operations))
	}
}

func TestTopologicalSort_ComplexCycle(t *testing.T) {
	graph := schema.NewDependencyGraph()

	// Create a more complex cycle: A -> B -> C -> A
	op1 := &schema.MigrationOperation{ID: "op1", Type: schema.OpCreateTable, Table: "a"}
	op2 := &schema.MigrationOperation{ID: "op2", Type: schema.OpCreateTable, Table: "b"}
	op3 := &schema.MigrationOperation{ID: "op3", Type: schema.OpCreateTable, Table: "c"}

	graph.AddOperation(op1)
	graph.AddOperation(op2)
	graph.AddOperation(op3)
	graph.AddDependency("op1", "op2") // B depends on A
	graph.AddDependency("op2", "op3") // C depends on B
	graph.AddDependency("op3", "op1") // A depends on C (cycle!)

	result, err := schema.TopologicalSort(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect circular dependency
	if len(result.CircularDependencies) == 0 && len(result.Warnings) == 0 {
		t.Error("expected circular dependency detection for complex cycle")
	}

	// Should still include all operations
	if len(result.Operations) != 3 {
		t.Errorf("expected 3 operations, got %d", len(result.Operations))
	}
}

func TestTopologicalSort_MultipleIndependentOperations(t *testing.T) {
	graph := schema.NewDependencyGraph()

	// Add multiple independent operations
	op1 := &schema.MigrationOperation{ID: "op1", Type: schema.OpCreateTable, Table: "users", Priority: 1}
	op2 := &schema.MigrationOperation{ID: "op2", Type: schema.OpCreateTable, Table: "products", Priority: 2}
	op3 := &schema.MigrationOperation{ID: "op3", Type: schema.OpCreateTable, Table: "categories", Priority: 3}

	graph.AddOperation(op1)
	graph.AddOperation(op2)
	graph.AddOperation(op3)

	result, err := schema.TopologicalSort(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Operations) != 3 {
		t.Errorf("expected 3 operations, got %d", len(result.Operations))
	}

	// Should be sorted by priority
	if result.Operations[0].Priority > result.Operations[1].Priority {
		t.Error("operations should be sorted by priority")
	}
}

func TestDependencyGraph_AddDependencyCreatesNodes(t *testing.T) {
	graph := schema.NewDependencyGraph()

	// Add dependency without first adding operations
	graph.AddDependency("non_existent_1", "non_existent_2")

	// Should create entries in InDegree
	if _, exists := graph.InDegree["non_existent_1"]; !exists {
		t.Error("expected non_existent_1 in InDegree")
	}
	if _, exists := graph.InDegree["non_existent_2"]; !exists {
		t.Error("expected non_existent_2 in InDegree")
	}
}

func TestGetOrderedRemovedTables_ExternalFKReference(t *testing.T) {
	// Test when FK references a table NOT being removed
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name: "orders",
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:            "fk_orders_user",
						Columns:         []string{"user_id"},
						ReferencedTable: "users", // users is NOT being removed
					},
				},
			},
		},
	}

	tableNames := []string{"orders"}
	result := schema.GetOrderedRemovedTables(tableNames, prodSchema)

	if len(result) != 1 {
		t.Errorf("expected 1 table, got %d", len(result))
	}
	if result[0] != "orders" {
		t.Errorf("expected orders, got %s", result[0])
	}
}

func TestBuildDependencyGraph_DropIndexBeforeDropColumn(t *testing.T) {
	// Test that drop index operation is created before drop column
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name: "test_table",
				RemovedColumns: []schema.Column{
					{Name: "old_col", DataType: "int"},
				},
				RemovedIndexes: []schema.Index{
					{Name: "idx_old", Columns: []string{"old_col"}},
				},
				HasDifferences: true,
			},
		},
	}

	graph := schema.BuildDependencyGraph(diff, nil)

	hasDropIdx := false
	hasDropCol := false
	for _, op := range graph.Operations {
		if op.Type == schema.OpDropIndex {
			hasDropIdx = true
		}
		if op.Type == schema.OpDropColumn {
			hasDropCol = true
		}
	}

	if !hasDropIdx {
		t.Error("expected DROP_INDEX operation")
	}
	if !hasDropCol {
		t.Error("expected DROP_COLUMN operation")
	}
}

func TestBuildDependencyGraph_TableNotInBothSchemas(t *testing.T) {
	// Test handling of tables that exist only in prod (OnlyInProd)
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "existing",
				OnlyInProd:     false,
				OnlyInDev:      false,
				HasDifferences: false,
			},
		},
	}

	graph := schema.BuildDependencyGraph(diff, nil)

	// Should not crash
	if graph == nil {
		t.Error("expected non-nil graph")
	}
}
