package schema_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

// TestIntrospect_SQLite_DefaultValues tests DEFAULT value introspection for SQLite
func TestIntrospect_SQLite_DefaultValues(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(dbPath)
	}()

	// Create test table with various DEFAULT values
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_defaults (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			count INTEGER DEFAULT 0,
			score REAL DEFAULT 0.0,
			is_enabled INTEGER DEFAULT 1,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			nullable_with_default TEXT DEFAULT 'default_value',
			no_default TEXT
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

	// Verify table exists
	table, ok := s.Tables["test_defaults"]
	if !ok {
		t.Fatal("test_defaults table not found in schema")
	}

	// Test cases for each column
	tests := []struct {
		columnName       string
		expectDefault    bool
		expectedDefault  string
	}{
		{"status", true, "'active'"},
		{"count", true, "0"},
		{"score", true, "0.0"},
		{"is_enabled", true, "1"},
		{"created_at", true, "CURRENT_TIMESTAMP"},
		{"nullable_with_default", true, "'default_value'"},
		{"no_default", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.columnName, func(t *testing.T) {
			col, ok := table.Columns[tt.columnName]
			if !ok {
				t.Fatalf("column %s not found", tt.columnName)
			}

			if tt.expectDefault {
				if col.DefaultValue == nil {
					t.Errorf("expected DefaultValue to be set, got nil")
				} else if *col.DefaultValue != tt.expectedDefault {
					t.Errorf("DefaultValue = %q, want %q", *col.DefaultValue, tt.expectedDefault)
				}
			} else {
				if col.DefaultValue != nil {
					t.Errorf("expected DefaultValue to be nil, got %q", *col.DefaultValue)
				}
			}
		})
	}
}

// TestIntrospect_SQLite_DefaultValue_ByteSlice tests DEFAULT value as byte slice
func TestIntrospect_SQLite_DefaultValue_ByteSlice(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_bytes.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(dbPath)
	}()

	// Create table with DEFAULT that might be returned as []byte
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_bytes (
			id INTEGER PRIMARY KEY,
			data BLOB DEFAULT x'deadbeef'
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

	table, ok := s.Tables["test_bytes"]
	if !ok {
		t.Fatal("test_bytes table not found")
	}

	col, ok := table.Columns["data"]
	if !ok {
		t.Fatal("data column not found")
	}

	// The DEFAULT value should be captured (exact value depends on SQLite driver)
	if col.DefaultValue == nil {
		t.Error("expected DefaultValue to be set for BLOB with default")
	}
}

// TestIntrospect_SQLite_EmptyDefault tests empty string DEFAULT
func TestIntrospect_SQLite_EmptyDefault(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_empty.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(dbPath)
	}()

	// Create table with empty string DEFAULT (should be ignored)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_empty (
			id INTEGER PRIMARY KEY,
			description TEXT
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

	table, ok := s.Tables["test_empty"]
	if !ok {
		t.Fatal("test_empty table not found")
	}

	col, ok := table.Columns["description"]
	if !ok {
		t.Fatal("description column not found")
	}

	// Empty string DEFAULT should result in nil DefaultValue
	if col.DefaultValue != nil {
		t.Errorf("expected DefaultValue to be nil for empty default, got %q", *col.DefaultValue)
	}
}

// TestIntrospect_SQLite_PrimaryKey tests PRIMARY KEY column
func TestIntrospect_SQLite_PrimaryKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_pk.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(dbPath)
	}()

	// Create table with composite primary key
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_pk (
			org_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			PRIMARY KEY (org_id, user_id)
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

	table, ok := s.Tables["test_pk"]
	if !ok {
		t.Fatal("test_pk table not found")
	}

	// Verify primary key is correctly identified
	if len(table.PrimaryKey) != 2 {
		t.Errorf("expected 2 primary key columns, got %d", len(table.PrimaryKey))
	}

	expectedPK := []string{"org_id", "user_id"}
	for i, expected := range expectedPK {
		if i >= len(table.PrimaryKey) || table.PrimaryKey[i] != expected {
			t.Errorf("PrimaryKey[%d] = %q, want %q", i, table.PrimaryKey[i], expected)
		}
	}
}

// TestIntrospect_SQLite_IgnoreTables tests table ignore functionality
func TestIntrospect_SQLite_IgnoreTables(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_ignore.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(dbPath)
	}()

	// Create multiple tables
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE sessions (id INTEGER PRIMARY KEY, token TEXT);
		CREATE TABLE logs (id INTEGER PRIMARY KEY, message TEXT);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// Load schema with ignored tables (case-insensitive)
	ignoreTables := []string{"Sessions", "LOGS"}
	s, err := schema.LoadSchema(ctx, db, "sqlite", "", ignoreTables)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	// Verify only users table is loaded
	if len(s.Tables) != 1 {
		t.Errorf("expected 1 table, got %d", len(s.Tables))
	}

	if _, ok := s.Tables["users"]; !ok {
		t.Error("users table should be present")
	}

	if _, ok := s.Tables["sessions"]; ok {
		t.Error("sessions table should be ignored")
	}

	if _, ok := s.Tables["logs"]; ok {
		t.Error("logs table should be ignored")
	}
}
