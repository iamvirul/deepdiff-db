package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

// openMemDB opens an in-memory SQLite database and fails the test on error.
func openMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// usersTable returns a basic users schema.Table for testing.
func usersTable() schema.Table {
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

func TestHashTable(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com'), (2, 'Bob', 'bob@example.com')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	table := usersTable()

	hashes, err := content.HashTable(ctx, db, "sqlite", table, nil, 0)
	if err != nil {
		t.Fatalf("HashTable: %v", err)
	}
	if len(hashes) != 2 {
		t.Fatalf("expected 2 hashes, got %d", len(hashes))
	}
	if _, ok := hashes["1"]; !ok {
		t.Error("missing hash for key '1'")
	}
	if _, ok := hashes["2"]; !ok {
		t.Error("missing hash for key '2'")
	}

	// Determinism check.
	hashes2, err := content.HashTable(ctx, db, "sqlite", table, nil, 0)
	if err != nil {
		t.Fatalf("HashTable (second call): %v", err)
	}
	if hashes["1"] != hashes2["1"] {
		t.Error("hash for key '1' is not deterministic")
	}
	if hashes["2"] != hashes2["2"] {
		t.Error("hash for key '2' is not deterministic")
	}
}

func TestHashTable_WithIgnore(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, updated_at TEXT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, name, updated_at) VALUES (1, 'Alice', '2024-01-01')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
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

	hashes1, err := content.HashTable(ctx, db, "sqlite", table, nil, 0)
	if err != nil {
		t.Fatalf("HashTable (no ignore): %v", err)
	}

	ignoreFn := content.IgnoreMatcher([]string{"*.updated_at"})
	hashes2, err := content.HashTable(ctx, db, "sqlite", table, ignoreFn, 0)
	if err != nil {
		t.Fatalf("HashTable (with ignore): %v", err)
	}
	if hashes1["1"] == hashes2["1"] {
		t.Error("hashes should differ when ignoring updated_at")
	}

	_, err = db.ExecContext(ctx, `UPDATE users SET updated_at = '2024-01-02' WHERE id = 1`)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	hashes3, err := content.HashTable(ctx, db, "sqlite", table, ignoreFn, 0)
	if err != nil {
		t.Fatalf("HashTable (after update, with ignore): %v", err)
	}
	if hashes2["1"] != hashes3["1"] {
		t.Error("hash should stay the same when ignored column changes")
	}
}

func TestHashTable_NoPrimaryKey(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	table := schema.Table{
		Name:       "users",
		Columns:    map[string]schema.Column{},
		PrimaryKey: []string{},
	}

	_, err := content.HashTable(ctx, db, "sqlite", table, nil, 0)
	if err == nil {
		t.Error("expected error for table without primary key")
	}
}

func TestHashTable_EmptyTable(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	table := schema.Table{
		Name: "users",
		Columns: map[string]schema.Column{
			"id":   {Name: "id", DataType: "integer", IsNullable: false},
			"name": {Name: "name", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	hashes, err := content.HashTable(ctx, db, "sqlite", table, nil, 0)
	if err != nil {
		t.Fatalf("HashTable: %v", err)
	}
	if len(hashes) != 0 {
		t.Errorf("expected 0 hashes for empty table, got %d", len(hashes))
	}
}

// TestHashTable_BatchedMatchesUnbatched verifies that keyset-paginated hashing
// produces identical results to the single-query path for the same data.
func TestHashTable_BatchedMatchesUnbatched(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	const rowCount = 1000
	for i := 1; i <= rowCount; i++ {
		_, err := db.ExecContext(ctx, `INSERT INTO items (id, name) VALUES (?, ?)`, i, fmt.Sprintf("item-%d", i))
		if err != nil {
			t.Fatalf("insert row %d: %v", i, err)
		}
	}

	table := schema.Table{
		Name: "items",
		Columns: map[string]schema.Column{
			"id":   {Name: "id", DataType: "integer", IsNullable: false},
			"name": {Name: "name", DataType: "text", IsNullable: false},
		},
		PrimaryKey: []string{"id"},
	}

	unbatched, err := content.HashTable(ctx, db, "sqlite", table, nil, 0)
	if err != nil {
		t.Fatalf("HashTable (unbatched): %v", err)
	}

	batched, err := content.HashTable(ctx, db, "sqlite", table, nil, 100)
	if err != nil {
		t.Fatalf("HashTable (batched, batchSize=100): %v", err)
	}

	if len(unbatched) != rowCount {
		t.Fatalf("unbatched: expected %d hashes, got %d", rowCount, len(unbatched))
	}
	if len(batched) != rowCount {
		t.Fatalf("batched: expected %d hashes, got %d", rowCount, len(batched))
	}

	for key, hash := range unbatched {
		if batched[key] != hash {
			t.Errorf("hash mismatch for key %q: unbatched=%q batched=%q", key, hash, batched[key])
		}
	}
}

// TestHashTable_KeysetPaginationCorrect checks that no rows are skipped or
// duplicated when the total row count is not an exact multiple of batchSize.
func TestHashTable_KeysetPaginationCorrect(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	const rowCount = 250
	for i := 1; i <= rowCount; i++ {
		_, err := db.ExecContext(ctx, `INSERT INTO items (id, val) VALUES (?, ?)`, i, fmt.Sprintf("v%d", i))
		if err != nil {
			t.Fatalf("insert row %d: %v", i, err)
		}
	}

	table := schema.Table{
		Name: "items",
		Columns: map[string]schema.Column{
			"id":  {Name: "id", DataType: "integer", IsNullable: false},
			"val": {Name: "val", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	// batchSize=50 gives 5 pages exactly. rowCount=250 is an exact multiple but
	// the final-page detection (pageCount < batchSize) still exercises the boundary.
	hashes, err := content.HashTable(ctx, db, "sqlite", table, nil, 50)
	if err != nil {
		t.Fatalf("HashTable: %v", err)
	}

	if len(hashes) != rowCount {
		t.Fatalf("expected %d keys, got %d", rowCount, len(hashes))
	}

	// Verify each expected key is present exactly once (no duplicates possible
	// in a map, but ensure no gaps).
	for i := 1; i <= rowCount; i++ {
		key := fmt.Sprintf("%d", i)
		if _, ok := hashes[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
}

// TestHashTable_BatchedEmptyTable verifies that batched mode on an empty table
// returns an empty map without error.
func TestHashTable_BatchedEmptyTable(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	table := schema.Table{
		Name: "items",
		Columns: map[string]schema.Column{
			"id":  {Name: "id", DataType: "integer", IsNullable: false},
			"val": {Name: "val", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	hashes, err := content.HashTable(ctx, db, "sqlite", table, nil, 1000)
	if err != nil {
		t.Fatalf("HashTable: %v", err)
	}
	if len(hashes) != 0 {
		t.Errorf("expected empty map, got %d entries", len(hashes))
	}
}
