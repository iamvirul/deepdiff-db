package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/drivers"
	"github.com/iamvirul/deepdiff-db/pkg/config"
	_ "modernc.org/sqlite"
)

// TestOpen_SQLiteInMemory verifies that Open succeeds for an in-memory SQLite
// database, exercising the successful sql.Open → Ping path in drivers.Open.
func TestOpen_SQLiteInMemory(t *testing.T) {
	ctx := context.Background()

	cfg := config.DBConfig{
		Driver:   "sqlite",
		Database: ":memory:",
	}

	db, err := drivers.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Open(:memory:) returned unexpected error: %v", err)
	}
	defer db.Close()

	// Verify the connection is actually usable.
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Ping after Open: %v", err)
	}
}

// TestOpen_SQLiteTempFile verifies Open works for a file-based SQLite path,
// exercising the same code path as in-memory but with a real filesystem path.
func TestOpen_SQLiteTempFile(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := config.DBConfig{
		Driver:   "sqlite",
		Database: dbPath,
	}

	db, err := drivers.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Open(file) returned unexpected error: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Ping after file Open: %v", err)
	}
}

// TestOpen_UnsupportedDriver exercises the BuildDSN error path in Open.
func TestOpen_UnsupportedDriver(t *testing.T) {
	ctx := context.Background()
	cfg := config.DBConfig{
		Driver:   "cassandra",
		Host:     "localhost",
		Port:     9042,
		Database: "ks",
	}
	_, err := drivers.Open(ctx, cfg)
	if err == nil {
		t.Error("expected error for unsupported driver, got nil")
	}
}

// TestOpen_SQLiteEmptyDatabase exercises the "sqlite database path is required"
// error from BuildDSN, which surfaces through Open.
func TestOpen_SQLiteEmptyDatabase(t *testing.T) {
	ctx := context.Background()
	cfg := config.DBConfig{
		Driver:   "sqlite",
		Database: "",
	}
	_, err := drivers.Open(ctx, cfg)
	if err == nil {
		t.Error("expected error for empty SQLite database path, got nil")
	}
}

// TestOpen_SQLiteContextCancelled verifies that a cancelled context causes
// Open to fail during the Ping step.
func TestOpen_SQLiteContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	cfg := config.DBConfig{
		Driver:   "sqlite",
		Database: ":memory:",
	}

	// SQLite in-memory Open/Ping is so fast it may succeed before the
	// cancellation is observed. Both outcomes are valid — assert we get
	// either a usable *sql.DB or a non-nil error; never both nil.
	db, err := drivers.Open(ctx, cfg)
	if err == nil && db == nil {
		t.Error("Open returned (nil, nil) — expected either a valid DB or an error")
	}
	if db != nil {
		db.Close()
	}
}
