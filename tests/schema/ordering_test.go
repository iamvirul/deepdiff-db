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
