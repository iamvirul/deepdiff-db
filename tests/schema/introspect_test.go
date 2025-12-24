package schema_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"

	_ "modernc.org/sqlite"
)

func TestLoadSchema_SQLite(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create test tables
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			age INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		CREATE TABLE posts (
			post_id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create posts table: %v", err)
	}

	schema, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("schema.schema.LoadSchema failed: %v", err)
	}

	if len(schema.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(schema.Tables))
	}

	// Check users table
	usersTable, ok := schema.Tables["users"]
	if !ok {
		t.Fatal("users table not found")
	}

	if len(usersTable.PrimaryKey) != 1 || usersTable.PrimaryKey[0] != "id" {
		t.Errorf("expected primary key ['id'], got %v", usersTable.PrimaryKey)
	}

	if len(usersTable.Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(usersTable.Columns))
	}

	// Check columns
	if col, ok := usersTable.Columns["id"]; ok {
		if col.DataType != "integer" {
			t.Errorf("expected id data type 'integer', got %q", col.DataType)
		}
		// Note: SQLite INTEGER PRIMARY KEY can be nullable in some contexts
		// We'll just check that the column exists
	} else {
		t.Error("id column not found")
	}

	if col, ok := usersTable.Columns["name"]; ok {
		if col.DataType != "text" {
			t.Errorf("expected name data type 'text', got %q", col.DataType)
		}
		if col.IsNullable {
			t.Error("name should not be nullable")
		}
	} else {
		t.Error("name column not found")
	}

	if col, ok := usersTable.Columns["email"]; ok {
		if !col.IsNullable {
			t.Error("email should be nullable")
		}
	} else {
		t.Error("email column not found")
	}

	// Check posts table
	postsTable, ok := schema.Tables["posts"]
	if !ok {
		t.Fatal("posts table not found")
	}

	if len(postsTable.PrimaryKey) != 1 || postsTable.PrimaryKey[0] != "post_id" {
		t.Errorf("expected primary key ['post_id'], got %v", postsTable.PrimaryKey)
	}
}

func TestLoadSchema_SQLite_WithIgnore(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		CREATE TABLE logs (
			id INTEGER PRIMARY KEY,
			message TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create logs table: %v", err)
	}

	ignoreTables := []string{"logs"}
	schema, err := schema.LoadSchema(ctx, db, "sqlite", "", ignoreTables)
	if err != nil {
		t.Fatalf("schema.schema.LoadSchema failed: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table (logs ignored), got %d", len(schema.Tables))
	}

	if _, ok := schema.Tables["logs"]; ok {
		t.Error("logs table should be ignored")
	}

	if _, ok := schema.Tables["users"]; !ok {
		t.Error("users table should not be ignored")
	}
}

func TestLoadSchema_SQLite_NoPrimaryKey(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create table without primary key
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER,
			name TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	schema, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("schema.schema.LoadSchema failed: %v", err)
	}

	usersTable, ok := schema.Tables["users"]
	if !ok {
		t.Fatal("users table not found")
	}

	if len(usersTable.PrimaryKey) != 0 {
		t.Errorf("expected no primary key, got %v", usersTable.PrimaryKey)
	}
}

func TestCheckPrimaryKeys_SQLite(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create table with primary key
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Create table without primary key
	_, err = db.ExecContext(ctx, `
		CREATE TABLE logs (
			id INTEGER,
			message TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create logs table: %v", err)
	}

	missing, err := schema.CheckPrimaryKeys(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("schema.schema.CheckPrimaryKeys failed: %v", err)
	}

	if len(missing) < 1 {
		t.Fatalf("expected at least 1 table missing primary key, got %d", len(missing))
	}

	// Check that logs is in the missing list
	foundLogs := false
	for _, table := range missing {
		if table == "logs" {
			foundLogs = true
			break
		}
	}
	if !foundLogs {
		t.Errorf("expected 'logs' to be in missing primary key list, got %v", missing)
	}
}

func TestCheckPrimaryKeys_SQLite_WithIgnore(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE logs (
			id INTEGER,
			message TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	ignoreTables := []string{"logs"}
	missing, err := schema.CheckPrimaryKeys(ctx, db, "sqlite", "", ignoreTables)
	if err != nil {
		t.Fatalf("schema.schema.CheckPrimaryKeys failed: %v", err)
	}

	if len(missing) != 0 {
		t.Errorf("expected 0 tables missing primary key (logs ignored), got %d", len(missing))
	}
}

func TestLoadSchema_UnsupportedDriver(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	_, err = schema.LoadSchema(ctx, db, "oracle", "", nil)
	if err == nil {
		t.Error("expected error for unsupported driver")
	}
}

func TestCheckPrimaryKeys_UnsupportedDriver(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	_, err = schema.CheckPrimaryKeys(ctx, db, "oracle", "", nil)
	if err == nil {
		t.Error("expected error for unsupported driver")
	}
}

