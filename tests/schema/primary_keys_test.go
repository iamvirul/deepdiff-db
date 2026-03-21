package schema_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

// pkTestCounter generates unique DB names so each test gets an isolated shared-cache DB.
var pkTestCounter uint64

// openPKTestDB opens an in-memory SQLite database that supports concurrent statements.
// SQLite's shared-cache mode with a unique name lets us run a second query while
// the first result-set is still open (which CheckPrimaryKeys does internally via PRAGMA).
func openPKTestDB(t *testing.T) *sql.DB {
	t.Helper()
	id := atomic.AddUint64(&pkTestCounter, 1)
	dsn := fmt.Sprintf("file:pktest%d?mode=memory&cache=shared", id)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	// Allow multiple open connections so concurrent queries work.
	db.SetMaxOpenConns(5)
	t.Cleanup(func() { db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// CheckPrimaryKeys – unsupported driver
// ---------------------------------------------------------------------------

func TestCheckPrimaryKeys_UnsupportedDriver(t *testing.T) {
	db := openPKTestDB(t)
	_, err := schema.CheckPrimaryKeys(context.Background(), db, "baddriver", "", nil)
	if err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
}

// ---------------------------------------------------------------------------
// CheckPrimaryKeys – SQLite: all tables have primary keys
// ---------------------------------------------------------------------------

func TestCheckPrimaryKeys_SQLite_AllHavePK(t *testing.T) {
	ctx := context.Background()
	db := openPKTestDB(t)

	exec(t, db, `CREATE TABLE users    (id INTEGER PRIMARY KEY, name TEXT)`)
	exec(t, db, `CREATE TABLE products (id INTEGER PRIMARY KEY, title TEXT)`)

	missing, err := schema.CheckPrimaryKeys(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("CheckPrimaryKeys() error = %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("expected no tables missing PK, got %v", missing)
	}
}

// ---------------------------------------------------------------------------
// CheckPrimaryKeys – SQLite: one table missing PK
// ---------------------------------------------------------------------------

func TestCheckPrimaryKeys_SQLite_OneMissingPK(t *testing.T) {
	ctx := context.Background()
	db := openPKTestDB(t)

	exec(t, db, `CREATE TABLE with_pk    (id INTEGER PRIMARY KEY, name TEXT)`)
	exec(t, db, `CREATE TABLE without_pk (name TEXT, value INTEGER)`)

	missing, err := schema.CheckPrimaryKeys(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("CheckPrimaryKeys() error = %v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("expected 1 table missing PK, got %v", missing)
	}
	if missing[0] != "without_pk" {
		t.Errorf("missing[0] = %q, want %q", missing[0], "without_pk")
	}
}

// ---------------------------------------------------------------------------
// CheckPrimaryKeys – SQLite: multiple tables missing PK
// ---------------------------------------------------------------------------

func TestCheckPrimaryKeys_SQLite_MultipleMissingPK(t *testing.T) {
	ctx := context.Background()
	db := openPKTestDB(t)

	exec(t, db, `CREATE TABLE has_pk  (id INTEGER PRIMARY KEY)`)
	exec(t, db, `CREATE TABLE nopk_a  (col1 TEXT, col2 INTEGER)`)
	exec(t, db, `CREATE TABLE nopk_b  (x REAL)`)

	missing, err := schema.CheckPrimaryKeys(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("CheckPrimaryKeys() error = %v", err)
	}
	if len(missing) != 2 {
		t.Errorf("expected 2 tables missing PK, got %v", missing)
	}
}

// ---------------------------------------------------------------------------
// CheckPrimaryKeys – SQLite: ignore tables (case-insensitive)
// ---------------------------------------------------------------------------

func TestCheckPrimaryKeys_SQLite_IgnoreTables(t *testing.T) {
	ctx := context.Background()
	db := openPKTestDB(t)

	exec(t, db, `CREATE TABLE tracked   (id INTEGER PRIMARY KEY)`)
	exec(t, db, `CREATE TABLE ignored   (no_pk TEXT)`)
	exec(t, db, `CREATE TABLE MixedIgn  (also_no_pk TEXT)`)

	// Ignore both tables that lack PKs
	missing, err := schema.CheckPrimaryKeys(ctx, db, "sqlite", "", []string{"IGNORED", "mixedign"})
	if err != nil {
		t.Fatalf("CheckPrimaryKeys() error = %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("expected 0 missing after ignoring, got %v", missing)
	}
}

// ---------------------------------------------------------------------------
// CheckPrimaryKeys – SQLite: composite primary key counts as having PK
// ---------------------------------------------------------------------------

func TestCheckPrimaryKeys_SQLite_CompositePKCountsAsHavingPK(t *testing.T) {
	ctx := context.Background()
	db := openPKTestDB(t)

	exec(t, db, `
		CREATE TABLE composite_pk (
			a INTEGER NOT NULL,
			b INTEGER NOT NULL,
			PRIMARY KEY (a, b)
		)
	`)

	missing, err := schema.CheckPrimaryKeys(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("CheckPrimaryKeys() error = %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("composite PK table should not appear as missing, got %v", missing)
	}
}

// ---------------------------------------------------------------------------
// CheckPrimaryKeys – SQLite: empty database returns empty slice
// ---------------------------------------------------------------------------

func TestCheckPrimaryKeys_SQLite_EmptyDatabase(t *testing.T) {
	ctx := context.Background()
	db := openPKTestDB(t)

	missing, err := schema.CheckPrimaryKeys(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("CheckPrimaryKeys() error = %v", err)
	}
	if missing != nil && len(missing) != 0 {
		t.Errorf("expected nil or empty slice, got %v", missing)
	}
}

// ---------------------------------------------------------------------------
// CheckPrimaryKeys – SQLite: driver name normalised to lower-case
// ---------------------------------------------------------------------------

func TestCheckPrimaryKeys_SQLite_DriverCaseNormalised(t *testing.T) {
	ctx := context.Background()
	db := openPKTestDB(t)

	exec(t, db, `CREATE TABLE t (id INTEGER PRIMARY KEY)`)

	for _, drv := range []string{"SQLITE", "Sqlite", "SQLite"} {
		_, err := schema.CheckPrimaryKeys(ctx, db, drv, "", nil)
		if err != nil {
			t.Errorf("driver %q: unexpected error: %v", drv, err)
		}
	}
}

// ---------------------------------------------------------------------------
// CheckPrimaryKeys – SQLite: ignore list partial – only some tables ignored
// ---------------------------------------------------------------------------

func TestCheckPrimaryKeys_SQLite_PartialIgnore(t *testing.T) {
	ctx := context.Background()
	db := openPKTestDB(t)

	exec(t, db, `CREATE TABLE pk_table    (id INTEGER PRIMARY KEY)`)
	exec(t, db, `CREATE TABLE nopk_keep   (val TEXT)`)
	exec(t, db, `CREATE TABLE nopk_ignore (val TEXT)`)

	missing, err := schema.CheckPrimaryKeys(ctx, db, "sqlite", "", []string{"nopk_ignore"})
	if err != nil {
		t.Fatalf("CheckPrimaryKeys() error = %v", err)
	}
	if len(missing) != 1 || missing[0] != "nopk_keep" {
		t.Errorf("expected [nopk_keep], got %v", missing)
	}
}
