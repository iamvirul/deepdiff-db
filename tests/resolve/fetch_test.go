package main

import (
	"context"
	"database/sql"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

// openFetchMemDB opens an in-memory SQLite database and registers cleanup.
func openFetchMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// mustExec executes a statement and fails the test on error.
func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// usersSchemaTable returns a schema.Table for a simple users table.
func usersSchemaTable() schema.Table {
	return schema.Table{
		Name: "users",
		Columns: map[string]schema.Column{
			"id":    {Name: "id", DataType: "integer", IsNullable: false},
			"name":  {Name: "name", DataType: "text", IsNullable: false},
			"email": {Name: "email", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}
}

// usersSchema returns a schema.Schema wrapping a single users table.
func usersSchema(tbl schema.Table) *schema.Schema {
	return &schema.Schema{
		Tables: map[string]schema.Table{
			tbl.Name: tbl,
		},
	}
}

// ============================================================================
// FetchConflictRows
// ============================================================================

func TestFetchConflictRows_HappyPath(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	mustExec(t, prodDB, `INSERT INTO users VALUES (1, 'Alice', 'alice@prod.com')`)

	mustExec(t, devDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	mustExec(t, devDB, `INSERT INTO users VALUES (1, 'Alice', 'alice@dev.com')`)

	tbl := usersSchemaTable()
	prodSch := usersSchema(tbl)
	devSch := usersSchema(tbl)

	conflict := content.Conflict{Table: "users", Key: "1"}
	prod, dev, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "sqlite", prodSch, devSch, conflict)
	if err != nil {
		t.Fatalf("FetchConflictRows: %v", err)
	}
	if prod == nil {
		t.Fatal("expected non-nil prod row")
	}
	if dev == nil {
		t.Fatal("expected non-nil dev row")
	}
	if len(prod.Columns) == 0 {
		t.Error("expected columns in prod row")
	}
	if len(dev.Columns) == 0 {
		t.Error("expected columns in dev row")
	}
}

func TestFetchConflictRows_RowMissingInProd(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	// No rows inserted into prod.

	mustExec(t, devDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	mustExec(t, devDB, `INSERT INTO users VALUES (42, 'Bob', 'bob@dev.com')`)

	tbl := usersSchemaTable()
	prodSch := usersSchema(tbl)
	devSch := usersSchema(tbl)

	conflict := content.Conflict{Table: "users", Key: "42"}
	prod, dev, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "sqlite", prodSch, devSch, conflict)
	if err != nil {
		t.Fatalf("FetchConflictRows: %v", err)
	}
	// prod row does not exist — should be nil (no-rows is swallowed).
	if prod != nil {
		t.Errorf("expected nil prod row when key absent, got %+v", prod)
	}
	if dev == nil {
		t.Fatal("expected non-nil dev row")
	}
}

func TestFetchConflictRows_RowMissingInDev(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	mustExec(t, prodDB, `INSERT INTO users VALUES (7, 'Carol', 'carol@prod.com')`)

	mustExec(t, devDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	// No rows in dev.

	tbl := usersSchemaTable()
	prodSch := usersSchema(tbl)
	devSch := usersSchema(tbl)

	conflict := content.Conflict{Table: "users", Key: "7"}
	prod, dev, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "sqlite", prodSch, devSch, conflict)
	if err != nil {
		t.Fatalf("FetchConflictRows: %v", err)
	}
	if prod == nil {
		t.Fatal("expected non-nil prod row")
	}
	if dev != nil {
		t.Errorf("expected nil dev row when key absent, got %+v", dev)
	}
}

func TestFetchConflictRows_TableNotInEitherSchema(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	emptySch := &schema.Schema{Tables: map[string]schema.Table{}}
	conflict := content.Conflict{Table: "nonexistent", Key: "1"}

	_, _, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "sqlite", emptySch, emptySch, conflict)
	if err == nil {
		t.Error("expected error when table missing from both schemas")
	}
}

func TestFetchConflictRows_TableOnlyInProdSchema(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	mustExec(t, prodDB, `INSERT INTO users VALUES (1, 'Prod-only', 'x@prod.com')`)

	tbl := usersSchemaTable()
	prodSch := usersSchema(tbl)
	devSch := &schema.Schema{Tables: map[string]schema.Table{}} // dev has no table

	conflict := content.Conflict{Table: "users", Key: "1"}
	prod, dev, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "sqlite", prodSch, devSch, conflict)
	if err != nil {
		t.Fatalf("FetchConflictRows: %v", err)
	}
	if prod == nil {
		t.Fatal("expected non-nil prod row")
	}
	if dev != nil {
		t.Errorf("expected nil dev row when table absent from dev schema, got %+v", dev)
	}
}

func TestFetchConflictRows_CompositePrimaryKey(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE order_items (order_id INTEGER NOT NULL, item_id INTEGER NOT NULL, qty INTEGER, PRIMARY KEY (order_id, item_id))`)
	mustExec(t, prodDB, `INSERT INTO order_items VALUES (10, 20, 3)`)

	mustExec(t, devDB, `CREATE TABLE order_items (order_id INTEGER NOT NULL, item_id INTEGER NOT NULL, qty INTEGER, PRIMARY KEY (order_id, item_id))`)
	mustExec(t, devDB, `INSERT INTO order_items VALUES (10, 20, 5)`)

	tbl := schema.Table{
		Name: "order_items",
		Columns: map[string]schema.Column{
			"order_id": {Name: "order_id", DataType: "integer", IsNullable: false},
			"item_id":  {Name: "item_id", DataType: "integer", IsNullable: false},
			"qty":      {Name: "qty", DataType: "integer", IsNullable: true},
		},
		PrimaryKey: []string{"order_id", "item_id"},
	}
	sch := &schema.Schema{Tables: map[string]schema.Table{"order_items": tbl}}

	conflict := content.Conflict{Table: "order_items", Key: "10|20"}
	prod, dev, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "sqlite", sch, sch, conflict)
	if err != nil {
		t.Fatalf("FetchConflictRows composite key: %v", err)
	}
	if prod == nil || dev == nil {
		t.Fatal("expected both prod and dev rows for composite key")
	}
}

// ============================================================================
// CompareRows
// ============================================================================

func TestCompareRows_BothNil(t *testing.T) {
	diffs := resolve.CompareRows(nil, nil)
	if diffs != nil {
		t.Errorf("expected nil diffs for two nil rows, got %v", diffs)
	}
}

func TestCompareRows_ProdNil(t *testing.T) {
	dev := &resolve.RowData{
		Columns: []string{"id", "name"},
		Values:  map[string]any{"id": int64(1), "name": "Alice"},
	}
	diffs := resolve.CompareRows(nil, dev)
	if len(diffs) == 0 {
		t.Error("expected diffs when prod is nil")
	}
	for _, d := range diffs {
		if !d.Differs {
			t.Errorf("column %q should differ when prod is nil", d.Column)
		}
	}
}

func TestCompareRows_DevNil(t *testing.T) {
	prod := &resolve.RowData{
		Columns: []string{"id", "name"},
		Values:  map[string]any{"id": int64(1), "name": "Alice"},
	}
	diffs := resolve.CompareRows(prod, nil)
	if len(diffs) == 0 {
		t.Error("expected diffs when dev is nil")
	}
}

func TestCompareRows_IdenticalRows(t *testing.T) {
	row := &resolve.RowData{
		Columns: []string{"id", "name"},
		Values:  map[string]any{"id": int64(1), "name": "Alice"},
	}
	diffs := resolve.CompareRows(row, row)
	for _, d := range diffs {
		if d.Differs {
			t.Errorf("column %q should not differ for identical rows", d.Column)
		}
	}
}

func TestCompareRows_DifferentValues(t *testing.T) {
	prod := &resolve.RowData{
		Columns: []string{"id", "name"},
		Values:  map[string]any{"id": int64(1), "name": "Alice"},
	}
	dev := &resolve.RowData{
		Columns: []string{"id", "name"},
		Values:  map[string]any{"id": int64(1), "name": "Bob"},
	}
	diffs := resolve.CompareRows(prod, dev)
	found := false
	for _, d := range diffs {
		if d.Column == "name" && d.Differs {
			found = true
		}
		if d.Column == "id" && d.Differs {
			t.Error("id column should not differ")
		}
	}
	if !found {
		t.Error("expected name column to differ")
	}
}

func TestCompareRows_ExtraColumnInDev(t *testing.T) {
	prod := &resolve.RowData{
		Columns: []string{"id"},
		Values:  map[string]any{"id": int64(1)},
	}
	dev := &resolve.RowData{
		Columns: []string{"id", "extra"},
		Values:  map[string]any{"id": int64(1), "extra": "new"},
	}
	diffs := resolve.CompareRows(prod, dev)
	found := false
	for _, d := range diffs {
		if d.Column == "extra" {
			found = true
			if !d.Differs {
				t.Error("extra column should differ since prod lacks it")
			}
		}
	}
	if !found {
		t.Error("expected extra column in diffs")
	}
}

// ============================================================================
// FormatValue
// ============================================================================

func TestFormatValue_Nil(t *testing.T) {
	got := resolve.FormatValue(nil)
	if got != "NULL" {
		t.Errorf("expected 'NULL', got %q", got)
	}
}

func TestFormatValue_Bool(t *testing.T) {
	if resolve.FormatValue(true) != "true" {
		t.Error("expected 'true' for bool true")
	}
	if resolve.FormatValue(false) != "false" {
		t.Error("expected 'false' for bool false")
	}
}

func TestFormatValue_ByteSlice(t *testing.T) {
	got := resolve.FormatValue([]byte("hello"))
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestFormatValue_Int(t *testing.T) {
	got := resolve.FormatValue(int64(42))
	if got != "42" {
		t.Errorf("expected '42', got %q", got)
	}
}

func TestFormatValue_String(t *testing.T) {
	got := resolve.FormatValue("world")
	if got != "world" {
		t.Errorf("expected 'world', got %q", got)
	}
}

// ============================================================================
// splitKey (tested indirectly via FetchConflictRows, but also directly via
// a bad composite key to confirm the error path)
// ============================================================================

func TestFetchConflictRows_BadCompositeKey(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE order_items (order_id INTEGER NOT NULL, item_id INTEGER NOT NULL, PRIMARY KEY (order_id, item_id))`)
	mustExec(t, devDB, `CREATE TABLE order_items (order_id INTEGER NOT NULL, item_id INTEGER NOT NULL, PRIMARY KEY (order_id, item_id))`)

	tbl := schema.Table{
		Name: "order_items",
		Columns: map[string]schema.Column{
			"order_id": {Name: "order_id", DataType: "integer", IsNullable: false},
			"item_id":  {Name: "item_id", DataType: "integer", IsNullable: false},
		},
		PrimaryKey: []string{"order_id", "item_id"},
	}
	sch := &schema.Schema{Tables: map[string]schema.Table{"order_items": tbl}}

	// Key with only 1 part for a 2-column PK — should error.
	conflict := content.Conflict{Table: "order_items", Key: "10"} // missing second part
	_, _, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "sqlite", sch, sch, conflict)
	if err == nil {
		t.Error("expected error for malformed composite key")
	}
}

// ============================================================================
// quoteIdent — exercised via FetchConflictRows with mysql driver identifier
// quoting; we also test the SQLite default path (no quoting) indirectly above.
// Direct coverage of the helper comes from round-tripping through the query.
// ============================================================================

func TestFetchConflictRows_MySQLQuoting_IdentifiersInQuery(t *testing.T) {
	// SQLite accepts backtick-quoted identifiers too, so we can use driver="mysql"
	// to exercise the MySQL quoteIdent branch without a real MySQL server.
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	mustExec(t, prodDB, `INSERT INTO users VALUES (5, 'Dave', 'dave@prod.com')`)
	mustExec(t, devDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	mustExec(t, devDB, `INSERT INTO users VALUES (5, 'Dave', 'dave@dev.com')`)

	tbl := usersSchemaTable()
	sch := usersSchema(tbl)
	conflict := content.Conflict{Table: "users", Key: "5"}

	prod, dev, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "mysql", sch, sch, conflict)
	if err != nil {
		t.Fatalf("FetchConflictRows (mysql quoting): %v", err)
	}
	if prod == nil || dev == nil {
		t.Fatal("expected both rows with mysql quoting")
	}
}

func TestFetchConflictRows_PostgresQuoting(t *testing.T) {
	// SQLite also accepts double-quote identifiers, so postgres quoting path
	// is exercisable via SQLite.
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	mustExec(t, prodDB, `INSERT INTO users VALUES (9, 'Eve', 'eve@prod.com')`)
	mustExec(t, devDB, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	mustExec(t, devDB, `INSERT INTO users VALUES (9, 'Eve', 'eve@dev.com')`)

	tbl := usersSchemaTable()
	sch := usersSchema(tbl)
	conflict := content.Conflict{Table: "users", Key: "9"}

	prod, dev, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "postgres", sch, sch, conflict)
	if err != nil {
		t.Fatalf("FetchConflictRows (postgres quoting): %v", err)
	}
	if prod == nil || dev == nil {
		t.Fatal("expected both rows with postgres quoting")
	}
}

// ============================================================================
// quoteLiteral — single-quote escaping tested via rows with apostrophes
// ============================================================================

func TestFetchConflictRows_ValueWithSingleQuote(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	// PK value contains a single quote — exercises quoteLiteral escaping.
	mustExec(t, prodDB, `CREATE TABLE tags (tag TEXT PRIMARY KEY, color TEXT)`)
	mustExec(t, prodDB, `INSERT INTO tags VALUES ("it's-a-tag", 'red')`)

	mustExec(t, devDB, `CREATE TABLE tags (tag TEXT PRIMARY KEY, color TEXT)`)
	mustExec(t, devDB, `INSERT INTO tags VALUES ("it's-a-tag", 'blue')`)

	tbl := schema.Table{
		Name: "tags",
		Columns: map[string]schema.Column{
			"tag":   {Name: "tag", DataType: "text", IsNullable: false},
			"color": {Name: "color", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"tag"},
	}
	sch := &schema.Schema{Tables: map[string]schema.Table{"tags": tbl}}
	conflict := content.Conflict{Table: "tags", Key: "it's-a-tag"}

	prod, dev, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "sqlite", sch, sch, conflict)
	if err != nil {
		t.Fatalf("FetchConflictRows with single-quote PK: %v", err)
	}
	if prod == nil || dev == nil {
		t.Fatal("expected both rows for single-quote PK value")
	}
}

// ============================================================================
// valuesEqual — via CompareRows with various type combinations
// ============================================================================

func TestCompareRows_ByteSliceValues(t *testing.T) {
	// valuesEqual has special handling for []byte — test that path.
	prod := &resolve.RowData{
		Columns: []string{"data"},
		Values:  map[string]any{"data": []byte("hello")},
	}
	dev := &resolve.RowData{
		Columns: []string{"data"},
		Values:  map[string]any{"data": []byte("hello")},
	}
	diffs := resolve.CompareRows(prod, dev)
	for _, d := range diffs {
		if d.Column == "data" && d.Differs {
			t.Error("identical byte slices should not differ")
		}
	}

	// Now with differing byte slices.
	dev.Values["data"] = []byte("world")
	diffs = resolve.CompareRows(prod, dev)
	for _, d := range diffs {
		if d.Column == "data" && !d.Differs {
			t.Error("differing byte slices should differ")
		}
	}
}

func TestCompareRows_NilValues(t *testing.T) {
	prod := &resolve.RowData{
		Columns: []string{"notes"},
		Values:  map[string]any{"notes": nil},
	}
	dev := &resolve.RowData{
		Columns: []string{"notes"},
		Values:  map[string]any{"notes": nil},
	}
	diffs := resolve.CompareRows(prod, dev)
	for _, d := range diffs {
		if d.Column == "notes" && d.Differs {
			t.Error("nil == nil should not differ")
		}
	}
}

// ============================================================================
// isNoRowsError — indirectly tested via FetchConflictRows missing rows above;
// also test directly that a non-sql.ErrNoRows error is not masked.
// ============================================================================

func TestFetchConflictRows_NoPrimaryKey(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE nopk (name TEXT)`)
	mustExec(t, prodDB, `INSERT INTO nopk VALUES ('Alice')`)
	mustExec(t, devDB, `CREATE TABLE nopk (name TEXT)`)

	// Table with no primary key — fetchRowData should error.
	tbl := schema.Table{
		Name: "nopk",
		Columns: map[string]schema.Column{
			"name": {Name: "name", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{}, // no PK
	}
	sch := &schema.Schema{Tables: map[string]schema.Table{"nopk": tbl}}
	conflict := content.Conflict{Table: "nopk", Key: "Alice"}

	_, _, err := resolve.FetchConflictRows(ctx, prodDB, devDB, "sqlite", sch, sch, conflict)
	if err == nil {
		t.Error("expected error for table with no primary key")
	}
}
