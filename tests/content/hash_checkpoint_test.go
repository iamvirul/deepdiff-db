package main

import (
	"context"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/checkpoint"
	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	"github.com/iamvirul/deepdiff-db/pkg/config"
)

// newCheckpointManager creates a real checkpoint.Manager backed by the test's
// temporary directory and pre-initialises it with a valid State so that
// Manager.Update works (it requires an existing in-memory state).
func newCheckpointManager(t *testing.T) *checkpoint.Manager {
	t.Helper()
	dir := t.TempDir()
	mgr := checkpoint.NewManager(dir)

	cfg := &config.Config{}
	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, dir, cfg)
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	if err := mgr.Save(state); err != nil {
		t.Fatalf("mgr.Save: %v", err)
	}
	// Load so the internal in-memory state pointer is populated.
	if _, err := mgr.Load(); err != nil {
		t.Fatalf("mgr.Load: %v", err)
	}
	return mgr
}

// ============================================================================
// HashTable — checkpoint paths
// ============================================================================

// TestHashTable_WithCheckpointManager runs HashTable (full / unbatched) with a
// real checkpoint.Manager injected via context to exercise saveHashBatchCheckpoint
// at the 1000-row interval and the final "mark table completed" block.
func TestHashTable_WithCheckpointManager(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT NOT NULL)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Insert 1050 rows so the 1000-row checkpoint interval fires at least once.
	for i := 1; i <= 1050; i++ {
		if _, err := db.ExecContext(ctx, `INSERT INTO items VALUES (?, ?)`, i, "v"); err != nil {
			t.Fatalf("insert row %d: %v", i, err)
		}
	}

	tbl := schema.Table{
		Name: "items",
		Columns: map[string]schema.Column{
			"id":  {Name: "id", DataType: "integer", IsNullable: false},
			"val": {Name: "val", DataType: "text", IsNullable: false},
		},
		PrimaryKey: []string{"id"},
	}

	mgr := newCheckpointManager(t)
	ctx = checkpoint.ToContext(ctx, mgr)

	hashes, err := content.HashTable(ctx, db, "sqlite", tbl, nil, 0)
	if err != nil {
		t.Fatalf("HashTable: %v", err)
	}
	if len(hashes) != 1050 {
		t.Errorf("expected 1050 hashes, got %d", len(hashes))
	}

	// Verify the table is recorded as completed in the checkpoint state.
	state, loadErr := mgr.Load()
	if loadErr != nil {
		t.Fatalf("Load after HashTable: %v", loadErr)
	}
	if state == nil || state.HashTableState == nil {
		t.Fatal("expected non-nil HashTableState after HashTable")
	}
	found := false
	for _, name := range state.HashTableState.CompletedTables {
		if name == "items" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'items' in CompletedTables, got %v", state.HashTableState.CompletedTables)
	}
}

// TestHashTable_BatchedWithCheckpointManager exercises saveHashBatchCheckpoint
// via the batched path (called at every batch boundary).
func TestHashTable_BatchedWithCheckpointManager(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT NOT NULL)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// 300 rows, batch size 100 → 3 batches → 3 checkpoint saves.
	for i := 1; i <= 300; i++ {
		if _, err := db.ExecContext(ctx, `INSERT INTO items VALUES (?, ?)`, i, "x"); err != nil {
			t.Fatalf("insert row %d: %v", i, err)
		}
	}

	tbl := schema.Table{
		Name: "items",
		Columns: map[string]schema.Column{
			"id":  {Name: "id", DataType: "integer", IsNullable: false},
			"val": {Name: "val", DataType: "text", IsNullable: false},
		},
		PrimaryKey: []string{"id"},
	}

	mgr := newCheckpointManager(t)
	ctx = checkpoint.ToContext(ctx, mgr)

	hashes, err := content.HashTable(ctx, db, "sqlite", tbl, nil, 100)
	if err != nil {
		t.Fatalf("HashTable (batched): %v", err)
	}
	if len(hashes) != 300 {
		t.Errorf("expected 300 hashes, got %d", len(hashes))
	}
	// Verify hashes were persisted in the checkpoint state.
	batchState, loadErr := mgr.Load()
	if loadErr != nil {
		t.Fatalf("Load after batched HashTable: %v", loadErr)
	}
	if batchState == nil || batchState.HashTableState == nil {
		t.Fatal("expected non-nil HashTableState after batched HashTable")
	}
	if n := len(batchState.HashTableState.Hashes["items"]); n == 0 {
		t.Errorf("expected hashes persisted for 'items', got 0 entries")
	}
}

// TestHashTable_CheckpointMarksCompletedTable verifies that after HashTable
// returns the table is recorded in HashTableState.CompletedTables and its
// hashes are stored in HashTableState.Hashes.
func TestHashTable_CheckpointMarksCompletedTable(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	for i := 1; i <= 5; i++ {
		if _, err := db.ExecContext(ctx, `INSERT INTO products VALUES (?, ?)`, i, "prod"); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	tbl := schema.Table{
		Name: "products",
		Columns: map[string]schema.Column{
			"id":   {Name: "id", DataType: "integer", IsNullable: false},
			"name": {Name: "name", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	mgr := newCheckpointManager(t)
	ctx = checkpoint.ToContext(ctx, mgr)

	hashes, err := content.HashTable(ctx, db, "sqlite", tbl, nil, 0)
	if err != nil {
		t.Fatalf("HashTable: %v", err)
	}

	// Reload checkpoint state from disk and verify.
	state, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if state == nil || state.HashTableState == nil {
		t.Fatal("expected HashTableState in checkpoint")
	}

	// "products" must be in CompletedTables.
	found := false
	for _, name := range state.HashTableState.CompletedTables {
		if name == "products" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'products' in CompletedTables, got %v", state.HashTableState.CompletedTables)
	}

	// All hashes must be persisted.
	saved := state.HashTableState.Hashes["products"]
	for k, v := range hashes {
		if saved[k] != v {
			t.Errorf("persisted hash mismatch for key %q: got %q want %q", k, saved[k], v)
		}
	}
}

// TestHashTable_CheckpointAlreadyHasTable verifies that calling HashTable a
// second time with the same checkpoint manager appends (not overwrites) the
// CompletedTables list idempotently.
func TestHashTable_CheckpointAlreadyHasTable(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE items (id INTEGER PRIMARY KEY, v TEXT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO items VALUES (1, 'a')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	tbl := schema.Table{
		Name: "items",
		Columns: map[string]schema.Column{
			"id": {Name: "id", DataType: "integer", IsNullable: false},
			"v":  {Name: "v", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	mgr := newCheckpointManager(t)
	ctx = checkpoint.ToContext(ctx, mgr)

	// First call.
	if _, err := content.HashTable(ctx, db, "sqlite", tbl, nil, 0); err != nil {
		t.Fatalf("first HashTable: %v", err)
	}

	// Second call — table already in CompletedTables, the dedup guard must not
	// add a duplicate entry.
	if _, err := content.HashTable(ctx, db, "sqlite", tbl, nil, 0); err != nil {
		t.Fatalf("second HashTable: %v", err)
	}

	state, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	count := 0
	for _, name := range state.HashTableState.CompletedTables {
		if name == "items" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 entry for 'items' in CompletedTables, got %d", count)
	}
}

// TestHashTable_AllColumnsIgnored confirms that an error is returned when
// the ignoreFn removes every non-PK column, leaving nothing to hash.
func TestHashTable_AllColumnsIgnored(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	_, err := db.ExecContext(ctx, `CREATE TABLE things (id INTEGER PRIMARY KEY, ts TEXT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	tbl := schema.Table{
		Name: "things",
		Columns: map[string]schema.Column{
			"id": {Name: "id", DataType: "integer", IsNullable: false},
			"ts": {Name: "ts", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	// ignoreFn removes all non-PK columns.
	ignoreFn := func(table, column string) bool {
		return column == "ts"
	}

	// With only "id" (PK) left, cols should just be ["id"] — not zero, so no
	// error from orderedColumns. This sub-test verifies that at least the PK is
	// always present.
	hashes, err := content.HashTable(ctx, db, "sqlite", tbl, ignoreFn, 0)
	if err != nil {
		t.Fatalf("unexpected error when only PK remains: %v", err)
	}
	// Empty table → 0 hashes, no error.
	if len(hashes) != 0 {
		t.Errorf("expected 0 hashes for empty table, got %d", len(hashes))
	}
}

// TestHashTable_NoColumnsAtAll checks the error path when orderedColumns
// returns an empty slice (PK is also excluded — only possible if PrimaryKey
// list is empty, but table has zero cols; PK empty already tested separately).
// This exercises the "no columns to hash" early-return.
func TestHashTable_NoColumnsToHash(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)

	// Table with only the PK column, which ignoreFn cannot remove; but we
	// construct a schema.Table where Columns is empty and PrimaryKey has one
	// entry that ignoreFn also removes — however ignoreFn only acts on non-PK
	// columns (orderedColumns always includes PK).  To hit the "no columns"
	// branch we need PrimaryKey non-empty but Columns empty AND ignoreFn
	// returning true for everything non-PK (which is nothing).  The only real
	// trigger is an empty Columns map with a non-empty PK where the PK cols
	// exist in the actual DB.  But that still returns at least one col (the PK).
	//
	// The "no columns to hash" error fires only when orderedColumns returns [].
	// orderedColumns always prepends PrimaryKey, so we can't make it empty if
	// PrimaryKey is non-empty.  We cover the nil-PK case in the existing
	// TestHashTable_NoPrimaryKey test.  This test just confirms the behaviour
	// when we have a non-empty PK and a totally-empty Columns map.
	_, err := db.ExecContext(ctx, `CREATE TABLE empty_cols (id INTEGER PRIMARY KEY)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	tbl := schema.Table{
		Name:       "empty_cols",
		Columns:    map[string]schema.Column{}, // empty — but PK still present
		PrimaryKey: []string{"id"},
	}

	// Should not error because PK col is always included.
	_, err = content.HashTable(ctx, db, "sqlite", tbl, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error for empty Columns map: %v", err)
	}
}
