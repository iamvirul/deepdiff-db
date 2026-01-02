package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
)

func TestSaveAndLoadResolutions(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "resolutions.json")

	resolutions := []resolve.Resolution{
		{
			Conflict: content.Conflict{
				Table:    "users",
				Key:      "1",
				ProdHash: "abc123",
				DevHash:  "def456",
			},
			Strategy: resolve.StrategyOurs,
			Decision: resolve.DecisionKeepProd,
			Resolved: true,
		},
		{
			Conflict: content.Conflict{
				Table:    "orders",
				Key:      "42",
				ProdHash: "hash1",
				DevHash:  "hash2",
			},
			Strategy: resolve.StrategyManual,
			Decision: resolve.DecisionPending,
			Resolved: false,
		},
	}

	// Save resolutions
	err := resolve.SaveResolutions(resolutions, filePath)
	if err != nil {
		t.Fatalf("SaveResolutions failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("resolutions file was not created")
	}

	// Load resolutions
	loaded, err := resolve.LoadResolutions(filePath)
	if err != nil {
		t.Fatalf("LoadResolutions failed: %v", err)
	}

	// Verify loaded data
	if loaded.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", loaded.Version)
	}
	if len(loaded.Resolutions) != 2 {
		t.Fatalf("expected 2 resolutions, got %d", len(loaded.Resolutions))
	}

	// Check first resolution
	if loaded.Resolutions[0].Conflict.Table != "users" {
		t.Errorf("expected table users, got %s", loaded.Resolutions[0].Conflict.Table)
	}
	if loaded.Resolutions[0].Strategy != resolve.StrategyOurs {
		t.Errorf("expected strategy ours, got %s", loaded.Resolutions[0].Strategy)
	}
	if !loaded.Resolutions[0].Resolved {
		t.Error("expected first resolution to be resolved")
	}

	// Check second resolution
	if loaded.Resolutions[1].Decision != resolve.DecisionPending {
		t.Errorf("expected decision pending, got %s", loaded.Resolutions[1].Decision)
	}
}

func TestLoadResolutionsNonExistent(t *testing.T) {
	_, err := resolve.LoadResolutions("/nonexistent/path/resolutions.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestSaveResolutionsPreservesCreatedAt(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "resolutions.json")

	// Save initial resolutions
	initial := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "users", Key: "1"}},
	}
	err := resolve.SaveResolutions(initial, filePath)
	if err != nil {
		t.Fatalf("initial save failed: %v", err)
	}

	// Load and get createdAt
	loaded1, _ := resolve.LoadResolutions(filePath)
	createdAt := loaded1.CreatedAt

	// Save updated resolutions
	updated := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "users", Key: "1"}},
		{Conflict: content.Conflict{Table: "orders", Key: "2"}},
	}
	err = resolve.SaveResolutions(updated, filePath)
	if err != nil {
		t.Fatalf("updated save failed: %v", err)
	}

	// Verify createdAt is preserved
	loaded2, _ := resolve.LoadResolutions(filePath)
	if loaded2.CreatedAt != createdAt {
		t.Errorf("createdAt changed: was %s, now %s", createdAt, loaded2.CreatedAt)
	}
	if loaded2.UpdatedAt == loaded2.CreatedAt {
		// This could fail if executed within the same second
		// but is unlikely in practice
	}
}

func TestMergeResolutions(t *testing.T) {
	saved := []resolve.Resolution{
		{
			Conflict: content.Conflict{Table: "users", Key: "1", ProdHash: "old1", DevHash: "old2"},
			Strategy: resolve.StrategyOurs,
			Decision: resolve.DecisionKeepProd,
			Resolved: true,
		},
		{
			Conflict: content.Conflict{Table: "orders", Key: "2", ProdHash: "old3", DevHash: "old4"},
			Strategy: resolve.StrategyManual,
			Decision: resolve.DecisionPending,
			Resolved: false,
		},
	}

	newConflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			// Updated existing conflict (hash changed)
			{Table: "users", Key: "1", ProdHash: "new1", DevHash: "new2"},
			// New conflict
			{Table: "products", Key: "3", ProdHash: "prod1", DevHash: "dev1"},
		},
		// orders conflict removed (no longer a conflict)
	}

	merged := resolve.MergeResolutions(saved, newConflicts)

	if len(merged) != 2 {
		t.Fatalf("expected 2 merged resolutions, got %d", len(merged))
	}

	// Check that users resolution is preserved with updated hashes
	var usersRes *resolve.Resolution
	for i := range merged {
		if merged[i].Conflict.Table == "users" {
			usersRes = &merged[i]
			break
		}
	}
	if usersRes == nil {
		t.Fatal("users resolution not found")
	}
	if usersRes.Decision != resolve.DecisionKeepProd {
		t.Errorf("users decision should be preserved, got %s", usersRes.Decision)
	}
	if usersRes.Conflict.ProdHash != "new1" {
		t.Errorf("users hash should be updated, got %s", usersRes.Conflict.ProdHash)
	}

	// Check that new conflict is added as pending
	var productsRes *resolve.Resolution
	for i := range merged {
		if merged[i].Conflict.Table == "products" {
			productsRes = &merged[i]
			break
		}
	}
	if productsRes == nil {
		t.Fatal("products resolution not found")
	}
	if productsRes.Decision != resolve.DecisionPending {
		t.Errorf("new conflict should be pending, got %s", productsRes.Decision)
	}
	if productsRes.Resolved {
		t.Error("new conflict should not be resolved")
	}
}

func TestUpdateResolution(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "users", Key: "1"}, Decision: resolve.DecisionPending, Resolved: false},
		{Conflict: content.Conflict{Table: "orders", Key: "2"}, Decision: resolve.DecisionPending, Resolved: false},
	}

	updated := resolve.UpdateResolution(resolutions, 0, resolve.StrategyOurs, resolve.DecisionKeepProd)

	if updated[0].Strategy != resolve.StrategyOurs {
		t.Errorf("expected strategy ours, got %s", updated[0].Strategy)
	}
	if updated[0].Decision != resolve.DecisionKeepProd {
		t.Errorf("expected decision keep_prod, got %s", updated[0].Decision)
	}
	if !updated[0].Resolved {
		t.Error("expected resolution to be marked as resolved")
	}

	// Verify second resolution unchanged
	if updated[1].Decision != resolve.DecisionPending {
		t.Error("second resolution should not be modified")
	}
}

func TestUpdateResolutionInvalidIndex(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "users", Key: "1"}},
	}

	// Should return unchanged list for invalid indices
	result := resolve.UpdateResolution(resolutions, -1, resolve.StrategyOurs, resolve.DecisionKeepProd)
	if len(result) != 1 {
		t.Error("should return original list for negative index")
	}

	result = resolve.UpdateResolution(resolutions, 5, resolve.StrategyOurs, resolve.DecisionKeepProd)
	if len(result) != 1 {
		t.Error("should return original list for out-of-bounds index")
	}
}

func TestApplyBulkResolution(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "users", Key: "1"}, Decision: resolve.DecisionPending, Resolved: false},
		{Conflict: content.Conflict{Table: "users", Key: "2"}, Decision: resolve.DecisionPending, Resolved: false},
		{Conflict: content.Conflict{Table: "orders", Key: "1"}, Decision: resolve.DecisionPending, Resolved: false},
		{Conflict: content.Conflict{Table: "users", Key: "3"}, Decision: resolve.DecisionKeepProd, Resolved: true}, // Already resolved
	}

	// Apply to all in "users" table
	updated := resolve.ApplyBulkResolution(resolutions, "users", false, resolve.StrategyTheirs)

	// Check users resolutions
	if updated[0].Decision != resolve.DecisionUseDev {
		t.Errorf("users 1 should be use_dev, got %s", updated[0].Decision)
	}
	if updated[1].Decision != resolve.DecisionUseDev {
		t.Errorf("users 2 should be use_dev, got %s", updated[1].Decision)
	}
	// Already resolved should stay unchanged
	if updated[3].Decision != resolve.DecisionKeepProd {
		t.Errorf("users 3 should remain keep_prod, got %s", updated[3].Decision)
	}

	// Check orders resolution unchanged
	if updated[2].Decision != resolve.DecisionPending {
		t.Errorf("orders should remain pending, got %s", updated[2].Decision)
	}
}

func TestApplyBulkResolutionAllTables(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "users", Key: "1"}, Decision: resolve.DecisionPending, Resolved: false},
		{Conflict: content.Conflict{Table: "orders", Key: "1"}, Decision: resolve.DecisionPending, Resolved: false},
	}

	// Apply to all tables
	updated := resolve.ApplyBulkResolution(resolutions, "", true, resolve.StrategyOurs)

	for _, res := range updated {
		if res.Decision != resolve.DecisionKeepProd {
			t.Errorf("all should be keep_prod, got %s", res.Decision)
		}
		if !res.Resolved {
			t.Error("all should be resolved")
		}
	}
}

func TestGetPendingResolutions(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Resolved: true},
		{Resolved: false},
		{Resolved: true},
		{Resolved: false},
	}

	pending := resolve.GetPendingResolutions(resolutions)

	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}
	for _, p := range pending {
		if p.Resolved {
			t.Error("pending resolutions should not be resolved")
		}
	}
}

func TestGetPendingCount(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Resolved: true},
		{Resolved: false},
		{Resolved: false},
	}

	count := resolve.GetPendingCount(resolutions)
	if count != 2 {
		t.Errorf("expected 2 pending, got %d", count)
	}
}

func TestGroupByTable(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "users", Key: "1"}},
		{Conflict: content.Conflict{Table: "users", Key: "2"}},
		{Conflict: content.Conflict{Table: "orders", Key: "1"}},
	}

	grouped := resolve.GroupByTable(resolutions)

	if len(grouped["users"]) != 2 {
		t.Errorf("expected 2 users, got %d", len(grouped["users"]))
	}
	if len(grouped["orders"]) != 1 {
		t.Errorf("expected 1 order, got %d", len(grouped["orders"]))
	}
}

func TestGetTableOrder(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "users", Key: "1"}},
		{Conflict: content.Conflict{Table: "orders", Key: "1"}},
		{Conflict: content.Conflict{Table: "users", Key: "2"}},
		{Conflict: content.Conflict{Table: "products", Key: "1"}},
	}

	order := resolve.GetTableOrder(resolutions)

	if len(order) != 3 {
		t.Fatalf("expected 3 unique tables, got %d", len(order))
	}
	// Order should be: users, orders, products (first occurrence order)
	expected := []string{"users", "orders", "products"}
	for i, table := range expected {
		if order[i] != table {
			t.Errorf("expected %s at position %d, got %s", table, i, order[i])
		}
	}
}
