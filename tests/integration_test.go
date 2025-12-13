package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/drivers"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	"github.com/iamvirul/deepdiff-db/pkg/config"
	_ "modernc.org/sqlite"
)

// Integration test that tests the full workflow:
// 1. Create two databases (prod and dev)
// 2. Load schemas
// 3. Hash tables
// 4. Build diff
// 5. Generate pack
// 6. Apply pack
func TestIntegration_FullWorkflow(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create prod database
	prodDB, err := sql.Open("sqlite", filepath.Join(tmpDir, "prod.db"))
	if err != nil {
		t.Fatalf("failed to open prod database: %v", err)
	}
	defer prodDB.Close()

	_, err = prodDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create prod table: %v", err)
	}

	_, err = prodDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(1, 'Alice', 'alice@example.com'),
		(2, 'Bob', 'bob@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert prod data: %v", err)
	}

	// Create dev database with different data
	devDB, err := sql.Open("sqlite", filepath.Join(tmpDir, "dev.db"))
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create dev table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(2, 'Bob Updated', 'bob.new@example.com'),
		(3, 'Charlie', 'charlie@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert dev data: %v", err)
	}

	// Load schemas
	prodSchema, err := schema.LoadSchema(ctx, prodDB, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load prod schema: %v", err)
	}

	devSchema, err := schema.LoadSchema(ctx, devDB, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load dev schema: %v", err)
	}

	// Check schema diff
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	if schemaDiff.HasDrift() {
		t.Error("schemas should match")
	}

	// Hash tables
	ignoreColumn := content.IgnoreMatcher(nil)
	prodHashes := make(map[string]map[string]string)
	devHashes := make(map[string]map[string]string)

	for name, prodTable := range prodSchema.Tables {
		devTable, ok := devSchema.Tables[name]
		if !ok {
			continue
		}

		pHashes, err := content.HashTable(ctx, prodDB, "sqlite", prodTable, ignoreColumn)
		if err != nil {
			t.Fatalf("failed to hash prod table %s: %v", name, err)
		}

		dHashes, err := content.HashTable(ctx, devDB, "sqlite", devTable, ignoreColumn)
		if err != nil {
			t.Fatalf("failed to hash dev table %s: %v", name, err)
		}

		prodHashes[name] = pHashes
		devHashes[name] = dHashes
	}

	// Build data diff
	dataDiff, _ := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	if !dataDiff.HasChanges() {
		t.Error("expected data changes")
	}

	// Verify diff contents
	var usersDiff *content.TableDataDiff
	for i := range dataDiff.Tables {
		if dataDiff.Tables[i].Table == "users" {
			usersDiff = &dataDiff.Tables[i]
			break
		}
	}

	if usersDiff == nil {
		t.Fatal("users table diff not found")
	}

	// Should have: removed key "1", updated key "2", added key "3"
	if len(usersDiff.Removed) != 1 || usersDiff.Removed[0] != "1" {
		t.Errorf("expected removed key '1', got %v", usersDiff.Removed)
	}
	if len(usersDiff.Updated) != 1 || usersDiff.Updated[0] != "2" {
		t.Errorf("expected updated key '2', got %v", usersDiff.Updated)
	}
	if len(usersDiff.Added) != 1 || usersDiff.Added[0] != "3" {
		t.Errorf("expected added key '3', got %v", usersDiff.Added)
	}

	// Generate pack
	outDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, devSchema, dataDiff, ignoreColumn, outDir)
	if err != nil {
		t.Fatalf("failed to generate pack: %v", err)
	}

	if _, err := os.Stat(packPath); os.IsNotExist(err) {
		t.Fatalf("pack file does not exist: %s", packPath)
	}

	// Read pack and strip BEGIN/COMMIT for SQLite (since ApplyPack wraps in transaction)
	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack: %v", err)
	}
	
	// Remove BEGIN and COMMIT lines for SQLite compatibility
	packStr := string(packContent)
	packStr = strings.ReplaceAll(packStr, "BEGIN;\n", "")
	packStr = strings.ReplaceAll(packStr, "\nCOMMIT;", "")
	packStr = strings.TrimSpace(packStr)
	
	// Write modified pack
	packPathModified := filepath.Join(tmpDir, "migration_pack_modified.sql")
	if err := os.WriteFile(packPathModified, []byte(packStr), 0o644); err != nil {
		t.Fatalf("failed to write modified pack: %v", err)
	}

	// Apply pack to prod (dry-run first)
	if err := content.ApplyPack(ctx, prodDB, packPathModified, true); err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	// Apply pack to prod
	if err := content.ApplyPack(ctx, prodDB, packPathModified, false); err != nil {
		t.Fatalf("apply pack failed: %v", err)
	}

	// Verify prod database matches dev
	prodHashesAfter, err := content.HashTable(ctx, prodDB, "sqlite", prodSchema.Tables["users"], ignoreColumn)
	if err != nil {
		t.Fatalf("failed to hash prod table after apply: %v", err)
	}

	devHashesAfter, err := content.HashTable(ctx, devDB, "sqlite", devSchema.Tables["users"], ignoreColumn)
	if err != nil {
		t.Fatalf("failed to hash dev table: %v", err)
	}

	// Compare hashes
	for key, devHash := range devHashesAfter {
		prodHash, ok := prodHashesAfter[key]
		if !ok {
			t.Errorf("key %s missing in prod after apply", key)
			continue
		}
		if prodHash != devHash {
			t.Errorf("hash mismatch for key %s: prod=%s dev=%s", key, prodHash, devHash)
		}
	}

	for key := range prodHashesAfter {
		if _, ok := devHashesAfter[key]; !ok {
			t.Errorf("key %s in prod but not in dev after apply", key)
		}
	}
}

// Test integration with config loading
func TestIntegration_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	configPath := filepath.Join(tmpDir, "test.yaml")
	configContent := `
prod:
  driver: "sqlite"
  host: ""
  port: 1
  user: ""
  password: ""
  database: "` + filepath.Join(tmpDir, "prod.db") + `"

dev:
  driver: "sqlite"
  host: ""
  port: 1
  user: ""
  password: ""
  database: "` + filepath.Join(tmpDir, "dev.db") + `"

ignore:
  tables: []
  columns: []

output:
  dir: "` + filepath.Join(tmpDir, "output") + `"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Create databases
	ctx := context.Background()

	prodDB, err := drivers.Open(ctx, cfg.Prod)
	if err != nil {
		t.Fatalf("failed to open prod database: %v", err)
	}
	defer prodDB.Close()

	_, err = prodDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create prod table: %v", err)
	}

	devDB, err := drivers.Open(ctx, cfg.Dev)
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create dev table: %v", err)
	}

	// Verify connections work
	if err := prodDB.PingContext(ctx); err != nil {
		t.Fatalf("prod database ping failed: %v", err)
	}

	if err := devDB.PingContext(ctx); err != nil {
		t.Fatalf("dev database ping failed: %v", err)
	}
}

