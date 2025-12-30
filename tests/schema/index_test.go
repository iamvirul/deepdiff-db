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
