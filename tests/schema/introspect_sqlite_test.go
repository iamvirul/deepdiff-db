package schema_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

// openInMemorySQLite opens an in-memory SQLite database and returns it ready for use.
func openInMemorySQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// exec is a helper to run DDL/DML and fail fast on error.
func exec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – unsupported driver
// ---------------------------------------------------------------------------

func TestLoadSchema_UnsupportedDriver_ReturnsError(t *testing.T) {
	db := openInMemorySQLite(t)
	_, err := schema.LoadSchema(context.Background(), db, "baddriver", "", nil)
	if err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: empty database
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_EmptyDatabase(t *testing.T) {
	db := openInMemorySQLite(t)
	s, err := schema.LoadSchema(context.Background(), db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}
	if len(s.Tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(s.Tables))
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: single table with all column types
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_SingleTable(t *testing.T) {
	ctx := context.Background()
	db := openInMemorySQLite(t)

	exec(t, db, `
		CREATE TABLE products (
			id      INTEGER PRIMARY KEY,
			name    TEXT    NOT NULL,
			price   REAL    DEFAULT 9.99,
			stock   INTEGER DEFAULT 0,
			active  INTEGER DEFAULT 1,
			notes   TEXT
		)
	`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	tbl, ok := s.Tables["products"]
	if !ok {
		t.Fatal("expected 'products' table in schema")
	}

	// Column count
	if len(tbl.Columns) != 6 {
		t.Errorf("expected 6 columns, got %d", len(tbl.Columns))
	}

	// Primary key
	if len(tbl.PrimaryKey) != 1 || tbl.PrimaryKey[0] != "id" {
		t.Errorf("expected PrimaryKey = [id], got %v", tbl.PrimaryKey)
	}

	// Nullable: notes has no NOT NULL constraint → should be nullable
	notes, ok := tbl.Columns["notes"]
	if !ok {
		t.Fatal("notes column not found")
	}
	if !notes.IsNullable {
		t.Error("notes should be nullable")
	}

	// Not-null: name is NOT NULL
	name, ok := tbl.Columns["name"]
	if !ok {
		t.Fatal("name column not found")
	}
	if name.IsNullable {
		t.Error("name should not be nullable")
	}

	// Default value: price
	price, ok := tbl.Columns["price"]
	if !ok {
		t.Fatal("price column not found")
	}
	if price.DefaultValue == nil {
		t.Error("price should have a default value")
	} else if *price.DefaultValue != "9.99" {
		t.Errorf("price default = %q, want %q", *price.DefaultValue, "9.99")
	}

	// DataType should be lower-cased
	if price.DataType != "real" {
		t.Errorf("price DataType = %q, want %q", price.DataType, "real")
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: composite primary key
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_CompositePrimaryKey(t *testing.T) {
	ctx := context.Background()
	db := openInMemorySQLite(t)

	exec(t, db, `
		CREATE TABLE order_items (
			order_id   INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			qty        INTEGER NOT NULL DEFAULT 1,
			PRIMARY KEY (order_id, product_id)
		)
	`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	tbl, ok := s.Tables["order_items"]
	if !ok {
		t.Fatal("expected 'order_items' table")
	}

	if len(tbl.PrimaryKey) != 2 {
		t.Fatalf("expected 2-column PK, got %v", tbl.PrimaryKey)
	}
	if tbl.PrimaryKey[0] != "order_id" || tbl.PrimaryKey[1] != "product_id" {
		t.Errorf("PrimaryKey = %v, want [order_id product_id]", tbl.PrimaryKey)
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: multiple tables
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_MultipleTables(t *testing.T) {
	ctx := context.Background()
	db := openInMemorySQLite(t)

	exec(t, db, `CREATE TABLE alpha (id INTEGER PRIMARY KEY, val TEXT)`)
	exec(t, db, `CREATE TABLE beta  (id INTEGER PRIMARY KEY, num INTEGER)`)
	exec(t, db, `CREATE TABLE gamma (id INTEGER PRIMARY KEY, ts  TEXT)`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	if len(s.Tables) != 3 {
		t.Errorf("expected 3 tables, got %d", len(s.Tables))
	}
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if _, ok := s.Tables[name]; !ok {
			t.Errorf("table %q not found in schema", name)
		}
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: ignore tables (case-insensitive)
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_IgnoreTables(t *testing.T) {
	ctx := context.Background()
	db := openInMemorySQLite(t)

	exec(t, db, `CREATE TABLE keep_me   (id INTEGER PRIMARY KEY)`)
	exec(t, db, `CREATE TABLE ignore_me (id INTEGER PRIMARY KEY)`)
	exec(t, db, `CREATE TABLE MixedCase (id INTEGER PRIMARY KEY)`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", []string{"IGNORE_ME", "mixedcase"})
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	if len(s.Tables) != 1 {
		t.Errorf("expected 1 table after ignore, got %d: %v", len(s.Tables), tableNames(s))
	}
	if _, ok := s.Tables["keep_me"]; !ok {
		t.Error("keep_me should be present")
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: indexes loaded (simple variant)
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_IndexesLoaded(t *testing.T) {
	ctx := context.Background()
	db := openInMemorySQLite(t)

	exec(t, db, `
		CREATE TABLE events (
			id        INTEGER PRIMARY KEY,
			user_id   INTEGER NOT NULL,
			event_at  TEXT    NOT NULL
		)
	`)
	exec(t, db, `CREATE INDEX idx_events_user ON events (user_id)`)
	exec(t, db, `CREATE UNIQUE INDEX uidx_events_at ON events (event_at)`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	tbl := s.Tables["events"]
	if tbl.Indexes == nil {
		t.Fatal("expected indexes to be loaded")
	}

	idx, ok := tbl.Indexes["idx_events_user"]
	if !ok {
		t.Fatal("idx_events_user not found")
	}
	if idx.IsUnique {
		t.Error("idx_events_user should not be unique")
	}
	if len(idx.Columns) != 1 || idx.Columns[0] != "user_id" {
		t.Errorf("idx_events_user columns = %v, want [user_id]", idx.Columns)
	}

	uidx, ok := tbl.Indexes["uidx_events_at"]
	if !ok {
		t.Fatal("uidx_events_at not found")
	}
	if !uidx.IsUnique {
		t.Error("uidx_events_at should be unique")
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: foreign keys loaded
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_ForeignKeys(t *testing.T) {
	ctx := context.Background()
	db := openInMemorySQLite(t)

	exec(t, db, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	exec(t, db, `
		CREATE TABLE posts (
			id      INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE
		)
	`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	posts := s.Tables["posts"]
	if len(posts.ForeignKeys) == 0 {
		t.Fatal("expected foreign keys on posts table")
	}

	// Find the FK (SQLite names them fk_posts_0, fk_posts_1, etc.)
	var fk schema.ForeignKey
	for _, v := range posts.ForeignKeys {
		fk = v
		break
	}

	if fk.ReferencedTable != "users" {
		t.Errorf("FK ReferencedTable = %q, want %q", fk.ReferencedTable, "users")
	}
	if len(fk.Columns) != 1 || fk.Columns[0] != "user_id" {
		t.Errorf("FK Columns = %v, want [user_id]", fk.Columns)
	}
	if len(fk.ReferencedColumns) != 1 || fk.ReferencedColumns[0] != "id" {
		t.Errorf("FK ReferencedColumns = %v, want [id]", fk.ReferencedColumns)
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: composite foreign key
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_CompositeForeignKey(t *testing.T) {
	ctx := context.Background()
	db := openInMemorySQLite(t)

	exec(t, db, `
		CREATE TABLE parent (
			a INTEGER NOT NULL,
			b INTEGER NOT NULL,
			PRIMARY KEY (a, b)
		)
	`)
	exec(t, db, `
		CREATE TABLE child (
			id INTEGER PRIMARY KEY,
			pa INTEGER NOT NULL,
			pb INTEGER NOT NULL,
			FOREIGN KEY (pa, pb) REFERENCES parent(a, b)
		)
	`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	child := s.Tables["child"]
	if len(child.ForeignKeys) == 0 {
		t.Fatal("expected composite FK on child table")
	}

	var fk schema.ForeignKey
	for _, v := range child.ForeignKeys {
		fk = v
		break
	}

	if len(fk.Columns) != 2 {
		t.Errorf("expected 2 FK columns, got %v", fk.Columns)
	}
	if len(fk.ReferencedColumns) != 2 {
		t.Errorf("expected 2 referenced columns, got %v", fk.ReferencedColumns)
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: table with no primary key
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_NoPrimaryKey(t *testing.T) {
	ctx := context.Background()
	db := openInMemorySQLite(t)

	exec(t, db, `
		CREATE TABLE log_entries (
			message TEXT,
			created TEXT
		)
	`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	tbl := s.Tables["log_entries"]
	if len(tbl.PrimaryKey) != 0 {
		t.Errorf("expected empty PrimaryKey, got %v", tbl.PrimaryKey)
	}
}

// ---------------------------------------------------------------------------
// LoadSchema – SQLite: driver name is case-insensitive
// ---------------------------------------------------------------------------

func TestLoadSchema_SQLite_DriverCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	db := openInMemorySQLite(t)
	exec(t, db, `CREATE TABLE t (id INTEGER PRIMARY KEY)`)

	for _, drv := range []string{"SQLITE", "Sqlite", "SQLite", "sqlite"} {
		s, err := schema.LoadSchema(ctx, db, drv, "", nil)
		if err != nil {
			t.Errorf("driver %q: unexpected error: %v", drv, err)
			continue
		}
		if _, ok := s.Tables["t"]; !ok {
			t.Errorf("driver %q: table 't' not found", drv)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func tableNames(s *schema.Schema) []string {
	names := make([]string, 0, len(s.Tables))
	for n := range s.Tables {
		names = append(names, n)
	}
	return names
}
