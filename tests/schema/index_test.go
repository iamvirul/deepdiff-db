package schema_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

// TestDiffIndexes_AddedIndex tests detection of indexes present in dev but not prod.
func TestDiffIndexes_AddedIndex(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_email": {
						Name:     "idx_users_email",
						Columns:  []string{"email"},
						IsUnique: false,
					},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	if !result.HasDrift() {
		t.Error("expected HasDrift to return true")
	}

	if len(result.Tables) != 1 {
		t.Fatalf("expected 1 table diff, got %d", len(result.Tables))
	}

	td := result.Tables[0]
	if len(td.AddedIndexes) != 1 {
		t.Fatalf("expected 1 added index, got %d", len(td.AddedIndexes))
	}

	addedIdx := td.AddedIndexes[0]
	if addedIdx.Name != "idx_users_email" {
		t.Errorf("expected index name 'idx_users_email', got '%s'", addedIdx.Name)
	}
	if len(addedIdx.Columns) != 1 || addedIdx.Columns[0] != "email" {
		t.Errorf("expected columns [email], got %v", addedIdx.Columns)
	}
	if addedIdx.IsUnique {
		t.Error("expected IsUnique to be false")
	}
}

// TestDiffIndexes_RemovedIndex tests detection of indexes present in prod but not dev.
func TestDiffIndexes_RemovedIndex(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_name": {
						Name:     "idx_users_name",
						Columns:  []string{"name"},
						IsUnique: true,
					},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	if !result.HasDrift() {
		t.Error("expected HasDrift to return true")
	}

	td := result.Tables[0]
	if len(td.RemovedIndexes) != 1 {
		t.Fatalf("expected 1 removed index, got %d", len(td.RemovedIndexes))
	}

	removedIdx := td.RemovedIndexes[0]
	if removedIdx.Name != "idx_users_name" {
		t.Errorf("expected index name 'idx_users_name', got '%s'", removedIdx.Name)
	}
	if !removedIdx.IsUnique {
		t.Error("expected IsUnique to be true")
	}
}

// TestDiffIndexes_ModifiedIndex tests detection of indexes that differ in columns.
func TestDiffIndexes_ModifiedIndex(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_composite": {
						Name:     "idx_users_composite",
						Columns:  []string{"first_name", "last_name"},
						IsUnique: false,
					},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_composite": {
						Name:     "idx_users_composite",
						Columns:  []string{"last_name", "first_name"}, // Different order
						IsUnique: false,
					},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	if !result.HasDrift() {
		t.Error("expected HasDrift to return true")
	}

	td := result.Tables[0]
	if len(td.ModifiedIndexes) != 1 {
		t.Fatalf("expected 1 modified index, got %d", len(td.ModifiedIndexes))
	}

	modifiedIdx := td.ModifiedIndexes[0]
	if modifiedIdx.Name != "idx_users_composite" {
		t.Errorf("expected index name 'idx_users_composite', got '%s'", modifiedIdx.Name)
	}
	if !modifiedIdx.ColumnsDiffer {
		t.Error("expected ColumnsDiffer to be true")
	}
}

// TestDiffIndexes_UniqueDiffers tests detection of indexes that differ in uniqueness.
func TestDiffIndexes_UniqueDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_email": {
						Name:     "idx_users_email",
						Columns:  []string{"email"},
						IsUnique: false,
					},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_email": {
						Name:     "idx_users_email",
						Columns:  []string{"email"},
						IsUnique: true, // Changed to unique
					},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	if !result.HasDrift() {
		t.Error("expected HasDrift to return true")
	}

	td := result.Tables[0]
	if len(td.ModifiedIndexes) != 1 {
		t.Fatalf("expected 1 modified index, got %d", len(td.ModifiedIndexes))
	}

	modifiedIdx := td.ModifiedIndexes[0]
	if !modifiedIdx.UniqueDiffers {
		t.Error("expected UniqueDiffers to be true")
	}
	if modifiedIdx.ProdUnique == nil || *modifiedIdx.ProdUnique != false {
		t.Error("expected ProdUnique to be false")
	}
	if modifiedIdx.DevUnique == nil || *modifiedIdx.DevUnique != true {
		t.Error("expected DevUnique to be true")
	}
}

// TestDiffIndexes_NoChanges tests that identical indexes don't show differences.
func TestDiffIndexes_NoChanges(t *testing.T) {
	idxDef := schema.Index{
		Name:     "idx_users_email",
		Columns:  []string{"email"},
		IsUnique: true,
	}
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{"idx_users_email": idxDef},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{"idx_users_email": idxDef},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	if result.HasDrift() {
		t.Error("expected HasDrift to return false for identical schemas")
	}
}

// TestGenerateCreateIndex_MySQL tests CREATE INDEX generation for MySQL.
func TestGenerateCreateIndex_MySQL(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "CREATE INDEX `idx_users_email` ON `users` (`email`);") {
		t.Errorf("expected CREATE INDEX statement, got:\n%s", sql)
	}
}

// TestGenerateCreateIndex_PostgreSQL tests CREATE INDEX generation for PostgreSQL.
func TestGenerateCreateIndex_PostgreSQL(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `CREATE INDEX "idx_users_email" ON "users" ("email");`) {
		t.Errorf("expected CREATE INDEX statement, got:\n%s", sql)
	}
}

// TestGenerateCreateIndex_SQLite tests CREATE INDEX generation for SQLite.
func TestGenerateCreateIndex_SQLite(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, `CREATE INDEX "idx_users_email" ON "users" ("email");`) {
		t.Errorf("expected CREATE INDEX statement, got:\n%s", sql)
	}
}

// TestGenerateCreateUniqueIndex tests CREATE UNIQUE INDEX generation.
func TestGenerateCreateUniqueIndex(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: true},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "CREATE UNIQUE INDEX `idx_users_email` ON `users` (`email`);") {
		t.Errorf("expected CREATE UNIQUE INDEX statement, got:\n%s", sql)
	}
}

// TestGenerateCreateIndex_Composite tests CREATE INDEX for multi-column indexes.
func TestGenerateCreateIndex_Composite(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_users_name", Columns: []string{"first_name", "last_name"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "CREATE INDEX `idx_users_name` ON `users` (`first_name`, `last_name`);") {
		t.Errorf("expected CREATE INDEX with multiple columns, got:\n%s", sql)
	}
}

// TestGenerateDropIndex_MySQL tests DROP INDEX generation for MySQL.
func TestGenerateDropIndex_MySQL(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_users_old", Columns: []string{"old_col"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DROP INDEX should be commented out by default
	if !strings.Contains(sql, "-- DROP INDEX `idx_users_old` ON `users`;") {
		t.Errorf("expected commented DROP INDEX statement, got:\n%s", sql)
	}
}

// TestGenerateDropIndex_PostgreSQL tests DROP INDEX generation for PostgreSQL.
func TestGenerateDropIndex_PostgreSQL(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_users_old", Columns: []string{"old_col"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DROP INDEX should be commented out by default
	if !strings.Contains(sql, `-- DROP INDEX "idx_users_old";`) {
		t.Errorf("expected commented DROP INDEX statement, got:\n%s", sql)
	}
}

// TestGenerateDropIndex_AllowDropIndex tests that AllowDropIndex uncomments DROP INDEX.
func TestGenerateDropIndex_AllowDropIndex(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_users_old", Columns: []string{"old_col"}, IsUnique: false},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowDropIndex: true}
	sql, err := schema.GenerateMigration(diff, "mysql", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DROP INDEX should NOT be commented out when AllowDropIndex is true
	if !strings.Contains(sql, "DROP INDEX `idx_users_old` ON `users`;") {
		t.Errorf("expected uncommented DROP INDEX statement, got:\n%s", sql)
	}
	// Make sure it's not the commented version
	if strings.Contains(sql, "-- DROP INDEX `idx_users_old` ON `users`;") {
		t.Errorf("expected DROP INDEX to not be commented out, got:\n%s", sql)
	}
}

// TestLoadSchema_SQLite_Indexes tests index introspection for SQLite.
func TestLoadSchema_SQLite_Indexes(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Create table with indexes
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT,
			name TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.ExecContext(ctx, `CREATE INDEX idx_users_email ON users(email)`)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	_, err = db.ExecContext(ctx, `CREATE UNIQUE INDEX idx_users_name ON users(name)`)
	if err != nil {
		t.Fatalf("failed to create unique index: %v", err)
	}

	// Load schema
	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	usersTable, ok := s.Tables["users"]
	if !ok {
		t.Fatal("users table not found")
	}

	if len(usersTable.Indexes) != 2 {
		t.Errorf("expected 2 indexes, got %d", len(usersTable.Indexes))
	}

	emailIdx, ok := usersTable.Indexes["idx_users_email"]
	if !ok {
		t.Error("idx_users_email not found")
	} else {
		if emailIdx.IsUnique {
			t.Error("expected idx_users_email to not be unique")
		}
		if len(emailIdx.Columns) != 1 || emailIdx.Columns[0] != "email" {
			t.Errorf("expected columns [email], got %v", emailIdx.Columns)
		}
	}

	nameIdx, ok := usersTable.Indexes["idx_users_name"]
	if !ok {
		t.Error("idx_users_name not found")
	} else {
		if !nameIdx.IsUnique {
			t.Error("expected idx_users_name to be unique")
		}
	}
}

// TestLoadSchema_SQLite_CompositeIndex tests composite index introspection for SQLite.
func TestLoadSchema_SQLite_CompositeIndex(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Create table with composite index
	_, err = db.ExecContext(ctx, `
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY,
			customer_id INTEGER,
			product_id INTEGER,
			quantity INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.ExecContext(ctx, `CREATE INDEX idx_orders_customer_product ON orders(customer_id, product_id)`)
	if err != nil {
		t.Fatalf("failed to create composite index: %v", err)
	}

	// Load schema
	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	ordersTable := s.Tables["orders"]
	idx, ok := ordersTable.Indexes["idx_orders_customer_product"]
	if !ok {
		t.Fatal("idx_orders_customer_product not found")
	}

	if len(idx.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(idx.Columns))
	}
	if idx.Columns[0] != "customer_id" || idx.Columns[1] != "product_id" {
		t.Errorf("expected columns [customer_id, product_id], got %v", idx.Columns)
	}
}

// TestLoadSchema_SQLite_SkipPrimaryKeyIndex tests that primary key indexes are skipped.
func TestLoadSchema_SQLite_SkipPrimaryKeyIndex(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Create table with only a primary key (no user-created indexes)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE items (
			id INTEGER PRIMARY KEY,
			name TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Load schema
	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	itemsTable := s.Tables["items"]
	if len(itemsTable.Indexes) != 0 {
		t.Errorf("expected 0 indexes (PK should be skipped), got %d: %v", len(itemsTable.Indexes), itemsTable.Indexes)
	}
}

// ============================================================================
// Additional Index Diff Tests
// ============================================================================

// TestDiffIndexes_NilIndexes tests handling of nil index maps.
func TestDiffIndexes_NilIndexes(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: nil, // nil indexes
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_email": {Name: "idx_users_email", Columns: []string{"email"}, IsUnique: false},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	if !result.HasDrift() {
		t.Error("expected HasDrift to return true")
	}

	td := result.Tables[0]
	if len(td.AddedIndexes) != 1 {
		t.Errorf("expected 1 added index, got %d", len(td.AddedIndexes))
	}
}

// TestDiffIndexes_BothNilIndexes tests handling when both schemas have nil indexes.
func TestDiffIndexes_BothNilIndexes(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: nil,
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: nil,
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	if result.HasDrift() {
		t.Error("expected HasDrift to return false for identical schemas with nil indexes")
	}
}

// TestDiffIndexes_MultipleAddedIndexes tests detection of multiple added indexes.
func TestDiffIndexes_MultipleAddedIndexes(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_email":    {Name: "idx_users_email", Columns: []string{"email"}, IsUnique: true},
					"idx_users_name":     {Name: "idx_users_name", Columns: []string{"name"}, IsUnique: false},
					"idx_users_fullname": {Name: "idx_users_fullname", Columns: []string{"first_name", "last_name"}, IsUnique: false},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	td := result.Tables[0]
	if len(td.AddedIndexes) != 3 {
		t.Errorf("expected 3 added indexes, got %d", len(td.AddedIndexes))
	}
}

// TestDiffIndexes_MultipleRemovedIndexes tests detection of multiple removed indexes.
func TestDiffIndexes_MultipleRemovedIndexes(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_old_1": {Name: "idx_old_1", Columns: []string{"col1"}, IsUnique: false},
					"idx_old_2": {Name: "idx_old_2", Columns: []string{"col2"}, IsUnique: false},
					"idx_old_3": {Name: "idx_old_3", Columns: []string{"col3"}, IsUnique: true},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	td := result.Tables[0]
	if len(td.RemovedIndexes) != 3 {
		t.Errorf("expected 3 removed indexes, got %d", len(td.RemovedIndexes))
	}
}

// TestDiffIndexes_MixedChanges tests detection of added, removed, and modified indexes together.
func TestDiffIndexes_MixedChanges(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_to_remove": {Name: "idx_to_remove", Columns: []string{"old_col"}, IsUnique: false},
					"idx_to_modify": {Name: "idx_to_modify", Columns: []string{"col_a", "col_b"}, IsUnique: false},
					"idx_unchanged": {Name: "idx_unchanged", Columns: []string{"status"}, IsUnique: false},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_to_add":    {Name: "idx_to_add", Columns: []string{"new_col"}, IsUnique: true},
					"idx_to_modify": {Name: "idx_to_modify", Columns: []string{"col_b", "col_a"}, IsUnique: false}, // Changed order
					"idx_unchanged": {Name: "idx_unchanged", Columns: []string{"status"}, IsUnique: false},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	td := result.Tables[0]
	if len(td.AddedIndexes) != 1 {
		t.Errorf("expected 1 added index, got %d", len(td.AddedIndexes))
	}
	if len(td.RemovedIndexes) != 1 {
		t.Errorf("expected 1 removed index, got %d", len(td.RemovedIndexes))
	}
	if len(td.ModifiedIndexes) != 1 {
		t.Errorf("expected 1 modified index, got %d", len(td.ModifiedIndexes))
	}
}

// TestDiffIndexes_MultipleTablesWithIndexes tests index diff across multiple tables.
func TestDiffIndexes_MultipleTablesWithIndexes(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_old": {Name: "idx_users_old", Columns: []string{"old"}, IsUnique: false},
				},
			},
			"products": {
				Name:    "products",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_products_sku": {Name: "idx_products_sku", Columns: []string{"sku"}, IsUnique: true},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_new": {Name: "idx_users_new", Columns: []string{"new"}, IsUnique: false},
				},
			},
			"products": {
				Name:    "products",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_products_sku":  {Name: "idx_products_sku", Columns: []string{"sku"}, IsUnique: true},
					"idx_products_name": {Name: "idx_products_name", Columns: []string{"name"}, IsUnique: false},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	// Find users table diff
	var usersDiff, productsDiff *schema.TableDiff
	for i := range result.Tables {
		if result.Tables[i].Name == "users" {
			usersDiff = &result.Tables[i]
		}
		if result.Tables[i].Name == "products" {
			productsDiff = &result.Tables[i]
		}
	}

	if usersDiff == nil {
		t.Fatal("users table diff not found")
	}
	if len(usersDiff.AddedIndexes) != 1 || len(usersDiff.RemovedIndexes) != 1 {
		t.Errorf("users: expected 1 added and 1 removed, got %d added, %d removed",
			len(usersDiff.AddedIndexes), len(usersDiff.RemovedIndexes))
	}

	if productsDiff == nil {
		t.Fatal("products table diff not found")
	}
	if len(productsDiff.AddedIndexes) != 1 || len(productsDiff.RemovedIndexes) != 0 {
		t.Errorf("products: expected 1 added and 0 removed, got %d added, %d removed",
			len(productsDiff.AddedIndexes), len(productsDiff.RemovedIndexes))
	}
}

// TestDiffIndexes_ColumnsAndUniquenessBothDiffer tests when both columns and uniqueness differ.
func TestDiffIndexes_ColumnsAndUniquenessBothDiffer(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_email": {Name: "idx_users_email", Columns: []string{"email"}, IsUnique: false},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: map[string]schema.Index{
					"idx_users_email": {Name: "idx_users_email", Columns: []string{"email", "domain"}, IsUnique: true},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	td := result.Tables[0]
	if len(td.ModifiedIndexes) != 1 {
		t.Fatalf("expected 1 modified index, got %d", len(td.ModifiedIndexes))
	}

	modified := td.ModifiedIndexes[0]
	if !modified.ColumnsDiffer {
		t.Error("expected ColumnsDiffer to be true")
	}
	if !modified.UniqueDiffers {
		t.Error("expected UniqueDiffers to be true")
	}
}

// ============================================================================
// Additional Index Migration Tests
// ============================================================================

// TestGenerateDropIndex_SQLite tests DROP INDEX generation for SQLite.
func TestGenerateDropIndex_SQLite(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_users_old", Columns: []string{"old_col"}, IsUnique: false},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowDropIndex: true}
	sql, err := schema.GenerateMigration(diff, "sqlite", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SQLite uses DROP INDEX without table reference (like PostgreSQL)
	if !strings.Contains(sql, `DROP INDEX "idx_users_old";`) {
		t.Errorf("expected DROP INDEX statement without table reference, got:\n%s", sql)
	}
}

// TestGenerateMigration_MultipleIndexOperations tests multiple index operations in one migration.
func TestGenerateMigration_MultipleIndexOperations(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_new_1", Columns: []string{"col1"}, IsUnique: false},
					{Name: "idx_new_2", Columns: []string{"col2"}, IsUnique: true},
				},
				RemovedIndexes: []schema.Index{
					{Name: "idx_old_1", Columns: []string{"old1"}, IsUnique: false},
					{Name: "idx_old_2", Columns: []string{"old2"}, IsUnique: false},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowDropIndex: true}
	sql, err := schema.GenerateMigration(diff, "mysql", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check all CREATE INDEX statements
	if !strings.Contains(sql, "CREATE INDEX `idx_new_1` ON `users` (`col1`);") {
		t.Error("expected CREATE INDEX for idx_new_1")
	}
	if !strings.Contains(sql, "CREATE UNIQUE INDEX `idx_new_2` ON `users` (`col2`);") {
		t.Error("expected CREATE UNIQUE INDEX for idx_new_2")
	}

	// Check all DROP INDEX statements
	if !strings.Contains(sql, "DROP INDEX `idx_old_1` ON `users`;") {
		t.Error("expected DROP INDEX for idx_old_1")
	}
	if !strings.Contains(sql, "DROP INDEX `idx_old_2` ON `users`;") {
		t.Error("expected DROP INDEX for idx_old_2")
	}
}

// TestGenerateMigration_ModifiedIndexesOutput tests that modified indexes produce comments.
func TestGenerateMigration_ModifiedIndexesOutput(t *testing.T) {
	boolTrue := true
	boolFalse := false

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				ModifiedIndexes: []schema.IndexDiff{
					{
						Name:          "idx_modified",
						ColumnsDiffer: true,
						ProdColumns:   []string{"col_a", "col_b"},
						DevColumns:    []string{"col_b", "col_a"},
					},
					{
						Name:          "idx_unique_change",
						UniqueDiffers: true,
						ProdUnique:    &boolFalse,
						DevUnique:     &boolTrue,
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check for modified index comments
	if !strings.Contains(sql, "MODIFIED INDEXES") {
		t.Error("expected MODIFIED INDEXES section header")
	}
	if !strings.Contains(sql, "Index idx_modified differs") {
		t.Error("expected comment about idx_modified")
	}
	if !strings.Contains(sql, "Index idx_unique_change differs") {
		t.Error("expected comment about idx_unique_change")
	}
	if !strings.Contains(sql, "Columns:") {
		t.Error("expected Columns difference output")
	}
	if !strings.Contains(sql, "Unique:") {
		t.Error("expected Unique difference output")
	}
}

// TestGenerateMigration_IndexWithSpecialCharacters tests index names with special characters.
func TestGenerateMigration_IndexWithSpecialCharacters(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "my_table",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_my_table_col", Columns: []string{"my_column"}, IsUnique: false},
				},
			},
		},
	}

	// MySQL
	sqlMySQL, _ := schema.GenerateMigration(diff, "mysql", nil)
	if !strings.Contains(sqlMySQL, "CREATE INDEX `idx_my_table_col` ON `my_table` (`my_column`);") {
		t.Errorf("MySQL: expected properly quoted identifiers, got:\n%s", sqlMySQL)
	}

	// PostgreSQL
	sqlPG, _ := schema.GenerateMigration(diff, "postgres", nil)
	if !strings.Contains(sqlPG, `CREATE INDEX "idx_my_table_col" ON "my_table" ("my_column");`) {
		t.Errorf("PostgreSQL: expected properly quoted identifiers, got:\n%s", sqlPG)
	}
}

// TestGenerateMigration_IndexOnlyChanges tests migration with only index changes (no column changes).
func TestGenerateMigration_IndexOnlyChanges(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_new", Columns: []string{"col"}, IsUnique: false},
				},
				// No column changes
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "CREATE INDEX") {
		t.Error("expected CREATE INDEX statement")
	}
	if !strings.Contains(sql, "Table: users") {
		t.Error("expected table header")
	}
}

// TestGenerateMigration_DropIndexWarnings tests that DROP INDEX includes proper warnings.
func TestGenerateMigration_DropIndexWarnings(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_old", Columns: []string{"col"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sql, "DROP INDEXES") {
		t.Error("expected DROP INDEXES section header")
	}
	if !strings.Contains(sql, "WARNING: Dropping indexes may impact query performance") {
		t.Error("expected warning about dropping indexes")
	}
}

// TestGenerateMigration_CompositeIndexThreeColumns tests composite index with 3+ columns.
func TestGenerateMigration_CompositeIndexThreeColumns(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_orders_composite", Columns: []string{"user_id", "product_id", "order_date"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "CREATE INDEX `idx_orders_composite` ON `orders` (`user_id`, `product_id`, `order_date`);"
	if !strings.Contains(sql, expected) {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, sql)
	}
}

// ============================================================================
// Additional SQLite Introspection Tests
// ============================================================================

// TestLoadSchema_SQLite_MultipleTablesWithIndexes tests loading indexes across multiple tables.
func TestLoadSchema_SQLite_MultipleTablesWithIndexes(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Create multiple tables with indexes
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT,
			name TEXT
		);
		CREATE INDEX idx_users_email ON users(email);

		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			sku TEXT,
			name TEXT
		);
		CREATE UNIQUE INDEX idx_products_sku ON products(sku);
		CREATE INDEX idx_products_name ON products(name);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	// Check users table
	usersTable := s.Tables["users"]
	if len(usersTable.Indexes) != 1 {
		t.Errorf("users: expected 1 index, got %d", len(usersTable.Indexes))
	}

	// Check products table
	productsTable := s.Tables["products"]
	if len(productsTable.Indexes) != 2 {
		t.Errorf("products: expected 2 indexes, got %d", len(productsTable.Indexes))
	}

	skuIdx := productsTable.Indexes["idx_products_sku"]
	if !skuIdx.IsUnique {
		t.Error("expected idx_products_sku to be unique")
	}
}

// TestLoadSchema_SQLite_NoIndexes tests loading schema with no indexes.
func TestLoadSchema_SQLite_NoIndexes(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	usersTable := s.Tables["users"]
	if usersTable.Indexes == nil {
		t.Error("expected Indexes to be initialized (not nil)")
	}
	if len(usersTable.Indexes) != 0 {
		t.Errorf("expected 0 indexes, got %d", len(usersTable.Indexes))
	}
}

// TestLoadSchema_SQLite_IgnoredTableIndexes tests that indexes on ignored tables are not loaded.
func TestLoadSchema_SQLite_IgnoredTableIndexes(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT
		);
		CREATE INDEX idx_users_email ON users(email);

		CREATE TABLE logs (
			id INTEGER PRIMARY KEY,
			message TEXT
		);
		CREATE INDEX idx_logs_message ON logs(message);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// Load schema ignoring logs table
	s, err := schema.LoadSchema(ctx, db, "sqlite", "", []string{"logs"})
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	if _, exists := s.Tables["logs"]; exists {
		t.Error("logs table should be ignored")
	}

	usersTable := s.Tables["users"]
	if len(usersTable.Indexes) != 1 {
		t.Errorf("expected 1 index on users, got %d", len(usersTable.Indexes))
	}
}

// TestLoadSchema_SQLite_UniqueConstraintIndex tests that UNIQUE constraint indexes are loaded.
func TestLoadSchema_SQLite_UniqueConstraintIndex(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Create table with explicit UNIQUE INDEX (not constraint)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT
		);
		CREATE UNIQUE INDEX idx_users_email_unique ON users(email);
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	usersTable := s.Tables["users"]
	idx, exists := usersTable.Indexes["idx_users_email_unique"]
	if !exists {
		t.Fatal("idx_users_email_unique not found")
	}
	if !idx.IsUnique {
		t.Error("expected idx_users_email_unique to be unique")
	}
}

// ============================================================================
// Index and Column Combined Tests
// ============================================================================

// TestDiffSchemas_IndexAndColumnChanges tests diff with both index and column changes.
func TestDiffSchemas_IndexAndColumnChanges(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "int"},
					"email": {Name: "email", DataType: "varchar(100)"},
				},
				Indexes: map[string]schema.Index{
					"idx_old": {Name: "idx_old", Columns: []string{"email"}, IsUnique: false},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "int"},
					"email": {Name: "email", DataType: "varchar(255)"}, // Type change
					"name":  {Name: "name", DataType: "varchar(50)"},   // New column
				},
				Indexes: map[string]schema.Index{
					"idx_new": {Name: "idx_new", Columns: []string{"name"}, IsUnique: false},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	td := result.Tables[0]

	// Check column changes
	if len(td.AddedColumns) != 1 {
		t.Errorf("expected 1 added column, got %d", len(td.AddedColumns))
	}
	if len(td.ModifiedColumns) != 1 {
		t.Errorf("expected 1 modified column, got %d", len(td.ModifiedColumns))
	}

	// Check index changes
	if len(td.AddedIndexes) != 1 {
		t.Errorf("expected 1 added index, got %d", len(td.AddedIndexes))
	}
	if len(td.RemovedIndexes) != 1 {
		t.Errorf("expected 1 removed index, got %d", len(td.RemovedIndexes))
	}
}

// TestGenerateMigration_ColumnAndIndexChanges tests migration with both column and index changes.
func TestGenerateMigration_ColumnAndIndexChanges(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedColumns: []schema.Column{
					{Name: "status", DataType: "varchar(20)", IsNullable: true},
				},
				AddedIndexes: []schema.Index{
					{Name: "idx_users_status", Columns: []string{"status"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check column ADD appears before index CREATE
	addColumnPos := strings.Index(sql, "ADD COLUMN")
	createIndexPos := strings.Index(sql, "CREATE INDEX")

	if addColumnPos == -1 {
		t.Error("expected ADD COLUMN statement")
	}
	if createIndexPos == -1 {
		t.Error("expected CREATE INDEX statement")
	}
	if addColumnPos > createIndexPos {
		t.Error("ADD COLUMN should appear before CREATE INDEX (dependency order)")
	}
}
