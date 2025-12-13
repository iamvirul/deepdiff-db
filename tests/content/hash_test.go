package main

import (
	"github.com/iamvirul/deepdiff-db/internal/content"

	"context"
	"database/sql"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

func TestHashTable(t *testing.T) {
	ctx := context.Background()

	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(1, 'Alice', 'alice@example.com'),
		(2, 'Bob', 'bob@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	table := schema.Table{
		Name: "users",
		Columns: map[string]schema.Column{
			"id":    {Name: "id", DataType: "integer", IsNullable: false},
			"name":  {Name: "name", DataType: "text", IsNullable: false},
			"email": {Name: "email", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	// Test hashing without ignore function
	hashes, err := content.HashTable(ctx, db, "sqlite", table, nil)
	if err != nil {
		t.Fatalf("content.content.HashTable failed: %v", err)
	}

	if len(hashes) != 2 {
		t.Fatalf("expected 2 hashes, got %d", len(hashes))
	}

	// Verify keys exist
	if _, ok := hashes["1"]; !ok {
		t.Error("missing hash for key '1'")
	}
	if _, ok := hashes["2"]; !ok {
		t.Error("missing hash for key '2'")
	}

	// Verify hashes are consistent
	hashes2, err := content.HashTable(ctx, db, "sqlite", table, nil)
	if err != nil {
		t.Fatalf("content.content.HashTable failed on second call: %v", err)
	}

	if hashes["1"] != hashes2["1"] {
		t.Error("hash for key '1' is not consistent")
	}
	if hashes["2"] != hashes2["2"] {
		t.Error("hash for key '2' is not consistent")
	}
}

func TestHashTable_WithIgnore(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			updated_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, name, updated_at) VALUES
		(1, 'Alice', '2024-01-01')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	table := schema.Table{
		Name: "users",
		Columns: map[string]schema.Column{
			"id":         {Name: "id", DataType: "integer", IsNullable: false},
			"name":       {Name: "name", DataType: "text", IsNullable: false},
			"updated_at": {Name: "updated_at", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	// Hash without ignoring updated_at
	hashes1, err := content.HashTable(ctx, db, "sqlite", table, nil)
	if err != nil {
		t.Fatalf("content.content.HashTable failed: %v", err)
	}

	// Hash with ignoring updated_at
	ignoreFn := content.IgnoreMatcher([]string{"*.updated_at"})
	hashes2, err := content.HashTable(ctx, db, "sqlite", table, ignoreFn)
	if err != nil {
		t.Fatalf("content.content.HashTable failed: %v", err)
	}

	// Hashes should be different because updated_at is included in first but not second
	if hashes1["1"] == hashes2["1"] {
		t.Error("hashes should differ when ignoring columns")
	}

	// Now update updated_at and hash again with ignore - hash should remain same
	_, err = db.ExecContext(ctx, `UPDATE users SET updated_at = '2024-01-02' WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	hashes3, err := content.HashTable(ctx, db, "sqlite", table, ignoreFn)
	if err != nil {
		t.Fatalf("content.content.HashTable failed: %v", err)
	}

	// Hash should be same because updated_at is ignored
	if hashes2["1"] != hashes3["1"] {
		t.Error("hash should remain same when ignored column changes")
	}
}

func TestHashTable_NoPrimaryKey(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	table := schema.Table{
		Name:    "users",
		Columns: map[string]schema.Column{},
		PrimaryKey: []string{}, // No primary key
	}

	_, err = content.HashTable(ctx, db, "sqlite", table, nil)
	if err == nil {
		t.Error("expected error for table without primary key")
	}
}

func TestHashTable_EmptyTable(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	table := schema.Table{
		Name: "users",
		Columns: map[string]schema.Column{
			"id":   {Name: "id", DataType: "integer", IsNullable: false},
			"name": {Name: "name", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	hashes, err := content.HashTable(ctx, db, "sqlite", table, nil)
	if err != nil {
		t.Fatalf("content.content.HashTable failed: %v", err)
	}

	if len(hashes) != 0 {
		t.Errorf("expected 0 hashes for empty table, got %d", len(hashes))
	}
}

// 

// 

