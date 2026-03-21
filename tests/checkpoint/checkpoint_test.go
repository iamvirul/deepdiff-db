package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/internal/checkpoint"
	"github.com/iamvirul/deepdiff-db/pkg/config"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	expectedPath := filepath.Join(tmpDir, checkpoint.CheckpointFileName)
	if mgr.Path() != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, mgr.Path())
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	cfg := &config.Config{
		Prod: config.DBConfig{
			Driver:   "sqlite",
			Database: "/tmp/prod.db",
		},
		Dev: config.DBConfig{
			Driver:   "sqlite",
			Database: "/tmp/dev.db",
		},
	}

	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, tmpDir, cfg)
	if err != nil {
		t.Fatalf("NewState failed: %v", err)
	}

	// Save checkpoint
	if err := mgr.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if !mgr.HasCheckpoint() {
		t.Error("HasCheckpoint returned false after Save")
	}

	// Load checkpoint
	loadedState, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loadedState == nil {
		t.Fatal("Load returned nil state")
	}

	if loadedState.Operation != checkpoint.OperationTypeHashTable {
		t.Errorf("expected operation %s, got %s", checkpoint.OperationTypeHashTable, loadedState.Operation)
	}

	if loadedState.OutputDir != tmpDir {
		t.Errorf("expected output dir %s, got %s", tmpDir, loadedState.OutputDir)
	}

	if loadedState.Version != checkpoint.CurrentVersion {
		t.Errorf("expected version %s, got %s", checkpoint.CurrentVersion, loadedState.Version)
	}
}

func TestSaveNilState(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	err := mgr.Save(nil)
	if err == nil {
		t.Error("Save with nil state should return error")
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	state, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load of non-existent checkpoint should not error, got: %v", err)
	}

	if state != nil {
		t.Error("Load of non-existent checkpoint should return nil state")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	// Write invalid JSON
	checkpointPath := mgr.Path()
	if err := os.MkdirAll(filepath.Dir(checkpointPath), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(checkpointPath, []byte("invalid json"), 0o644); err != nil {
		t.Fatalf("failed to write invalid checkpoint: %v", err)
	}

	_, err := mgr.Load()
	if err == nil {
		t.Error("Load of invalid JSON should return error")
	}
}

func TestLoadInvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	// Write checkpoint with wrong version
	checkpointPath := mgr.Path()
	if err := os.MkdirAll(filepath.Dir(checkpointPath), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	invalidState := `{
		"version": "2.0",
		"operation": "hash_table",
		"created_at": "2024-01-01T00:00:00Z",
		"last_updated": "2024-01-01T00:00:00Z",
		"config_hash": "abc123",
		"output_dir": "` + tmpDir + `"
	}`

	if err := os.WriteFile(checkpointPath, []byte(invalidState), 0o644); err != nil {
		t.Fatalf("failed to write invalid checkpoint: %v", err)
	}

	_, err := mgr.Load()
	if err == nil {
		t.Error("Load of invalid version should return error")
	}
}

func TestUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, tmpDir, cfg)
	if err != nil {
		t.Fatalf("NewState failed: %v", err)
	}

	if err := mgr.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Update state
	err = mgr.Update(func(s *checkpoint.State) error {
		s.HashTableState = &checkpoint.HashTableState{
			CompletedTables: []string{"users"},
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	loadedState, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loadedState.HashTableState == nil {
		t.Fatal("HashTableState should not be nil after update")
	}

	if len(loadedState.HashTableState.CompletedTables) != 1 {
		t.Errorf("expected 1 completed table, got %d", len(loadedState.HashTableState.CompletedTables))
	}
}

func TestUpdateWithoutLoadedState(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	err := mgr.Update(func(s *checkpoint.State) error {
		return nil
	})

	if err == nil {
		t.Error("Update without loaded state should return error")
	}
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, tmpDir, cfg)
	if err != nil {
		t.Fatalf("NewState failed: %v", err)
	}

	if err := mgr.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !mgr.HasCheckpoint() {
		t.Error("HasCheckpoint should return true before delete")
	}

	if err := mgr.Delete(); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if mgr.HasCheckpoint() {
		t.Error("HasCheckpoint should return false after delete")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	// Deleting non-existent checkpoint should not error
	if err := mgr.Delete(); err != nil {
		t.Errorf("Delete of non-existent checkpoint should not error, got: %v", err)
	}
}

func TestContext(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	ctx := context.Background()
	ctx = checkpoint.ToContext(ctx, mgr)

	retrieved := checkpoint.FromContext(ctx)
	if retrieved == nil {
		t.Fatal("FromContext returned nil")
	}

	if retrieved.Path() != mgr.Path() {
		t.Error("retrieved manager has different path")
	}
}

func TestFromContextNil(t *testing.T) {
	ctx := context.Background()
	if checkpoint.FromContext(ctx) != nil {
		t.Error("FromContext with context without manager should return nil")
	}
}

func TestStateComputeConfigHash(t *testing.T) {
	cfg1 := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	hash1, err := checkpoint.ComputeConfigHash(cfg1)
	if err != nil {
		t.Fatalf("ComputeConfigHash failed: %v", err)
	}

	if hash1 == "" {
		t.Error("hash should not be empty")
	}

	// Same config should produce same hash
	hash2, err := checkpoint.ComputeConfigHash(cfg1)
	if err != nil {
		t.Fatalf("ComputeConfigHash failed: %v", err)
	}

	if hash1 != hash2 {
		t.Error("same config should produce same hash")
	}

	// Different config should produce different hash
	cfg2 := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod2.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	hash3, err := checkpoint.ComputeConfigHash(cfg2)
	if err != nil {
		t.Fatalf("ComputeConfigHash failed: %v", err)
	}

	if hash1 == hash3 {
		t.Error("different config should produce different hash")
	}
}

func TestNewState(t *testing.T) {
	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	state, err := checkpoint.NewState(checkpoint.OperationTypeGeneratePack, "/tmp/output", cfg)
	if err != nil {
		t.Fatalf("NewState failed: %v", err)
	}

	if state.Operation != checkpoint.OperationTypeGeneratePack {
		t.Errorf("expected operation %s, got %s", checkpoint.OperationTypeGeneratePack, state.Operation)
	}

	if state.OutputDir != "/tmp/output" {
		t.Errorf("expected output dir /tmp/output, got %s", state.OutputDir)
	}

	if state.Version != checkpoint.CurrentVersion {
		t.Errorf("expected version %s, got %s", checkpoint.CurrentVersion, state.Version)
	}

	if state.ConfigHash == "" {
		t.Error("ConfigHash should not be empty")
	}
}

func TestValidateConfigHash(t *testing.T) {
	cfg1 := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, "/tmp", cfg1)
	if err != nil {
		t.Fatalf("NewState failed: %v", err)
	}

	// Same config should validate
	if err := state.ValidateConfigHash(cfg1); err != nil {
		t.Errorf("ValidateConfigHash with same config should not error: %v", err)
	}

	// Different config should fail
	cfg2 := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod2.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	if err := state.ValidateConfigHash(cfg2); err == nil {
		t.Error("ValidateConfigHash with different config should error")
	}
}

func TestIsExpired(t *testing.T) {
	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, "/tmp", cfg)
	if err != nil {
		t.Fatalf("NewState failed: %v", err)
	}

	// Should not be expired (just created)
	if state.IsExpired(24 * time.Hour) {
		t.Error("newly created state should not be expired")
	}

	// Make it expired
	state.LastUpdated = time.Now().Add(-25 * time.Hour)
	if !state.IsExpired(24 * time.Hour) {
		t.Error("old state should be expired")
	}

	// Test default expiration (24 hours)
	state.LastUpdated = time.Now().Add(-25 * time.Hour)
	if !state.IsExpired(0) {
		t.Error("old state should be expired with default expiration")
	}
}

func TestValidate(t *testing.T) {
	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, "/tmp", cfg)
	if err != nil {
		t.Fatalf("NewState failed: %v", err)
	}

	// Valid state should pass
	opts := checkpoint.ResumeOptions{
		MaxAge:         24 * time.Hour,
		ValidateConfig: false,
	}

	if err := checkpoint.Validate(state, cfg, opts); err != nil {
		t.Errorf("Validate should pass for valid state: %v", err)
	}

	// Expired state should fail
	state.LastUpdated = time.Now().Add(-25 * time.Hour)
	if err := checkpoint.Validate(state, cfg, opts); err == nil {
		t.Error("Validate should fail for expired state")
	}

	// Config validation
	opts.ValidateConfig = true
	state.LastUpdated = time.Now() // Reset to not expired
	if err := checkpoint.Validate(state, cfg, opts); err != nil {
		t.Errorf("Validate with config validation should pass for matching config: %v", err)
	}

	cfg2 := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod2.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	if err := checkpoint.Validate(state, cfg2, opts); err == nil {
		t.Error("Validate should fail for mismatched config")
	}
}

func TestValidateNilState(t *testing.T) {
	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	opts := checkpoint.ResumeOptions{
		MaxAge:         24 * time.Hour,
		ValidateConfig: false,
	}

	if err := checkpoint.Validate(nil, cfg, opts); err == nil {
		t.Error("Validate with nil state should return error")
	}
}

func TestGetResumeInfo(t *testing.T) {
	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}

	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, "/tmp/output", cfg)
	if err != nil {
		t.Fatalf("NewState failed: %v", err)
	}

	info := checkpoint.GetResumeInfo(state)
	if info == nil {
		t.Fatal("GetResumeInfo returned nil")
	}

	if info.Operation != checkpoint.OperationTypeHashTable {
		t.Errorf("expected operation %s, got %s", checkpoint.OperationTypeHashTable, info.Operation)
	}

	if info.OutputDir != "/tmp/output" {
		t.Errorf("expected output dir /tmp/output, got %s", info.OutputDir)
	}

	// Test nil state
	if checkpoint.GetResumeInfo(nil) != nil {
		t.Error("GetResumeInfo with nil state should return nil")
	}
}

func TestHashTableState(t *testing.T) {
	state := &checkpoint.HashTableState{
		CompletedTables: []string{"users", "posts"},
		CurrentTable:    "comments",
		CurrentRowCount: 100,
		Hashes: map[string]map[string]string{
			"users": {
				"1": "hash1",
				"2": "hash2",
			},
		},
	}

	if len(state.CompletedTables) != 2 {
		t.Errorf("expected 2 completed tables, got %d", len(state.CompletedTables))
	}

	if state.CurrentTable != "comments" {
		t.Errorf("expected current table 'comments', got %s", state.CurrentTable)
	}

	if state.CurrentRowCount != 100 {
		t.Errorf("expected current row count 100, got %d", state.CurrentRowCount)
	}
}

func TestGeneratePackState(t *testing.T) {
	state := &checkpoint.GeneratePackState{
		CompletedTables: []string{"users"},
		CurrentTable:    "posts",
		Statements:      []string{"INSERT INTO users ...", "UPDATE users ..."},
	}

	if len(state.CompletedTables) != 1 {
		t.Errorf("expected 1 completed table, got %d", len(state.CompletedTables))
	}

	if state.CurrentTable != "posts" {
		t.Errorf("expected current table 'posts', got %s", state.CurrentTable)
	}

	if len(state.Statements) != 2 {
		t.Errorf("expected 2 statements, got %d", len(state.Statements))
	}
}

func TestSave_MkdirError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on Windows")
	}
	tmpParent := t.TempDir()
	// Read-only parent so MkdirAll cannot create a new subdirectory inside it.
	if err := os.Chmod(tmpParent, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(tmpParent, 0o750) }) //nolint:errcheck

	mgr := checkpoint.NewManager(filepath.Join(tmpParent, "newsubdir"))
	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}
	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, tmpParent, cfg)
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	if err := mgr.Save(state); err == nil {
		t.Error("Save should fail when MkdirAll cannot create directory")
	}
}

func TestSave_WriteFileError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on Windows")
	}
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}
	state, err := checkpoint.NewState(checkpoint.OperationTypeHashTable, tmpDir, cfg)
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	// Make dir read-only: MkdirAll succeeds (dir exists) but WriteFile fails.
	if err := os.Chmod(tmpDir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(tmpDir, 0o750) }) //nolint:errcheck

	mgr := checkpoint.NewManager(tmpDir)
	if err := mgr.Save(state); err == nil {
		t.Error("Save should fail when WriteFile cannot write to directory")
	}
}

func TestApplyPackState(t *testing.T) {
	state := &checkpoint.ApplyPackState{
		ExecutedStatements: 50,
		TotalStatements:     100,
		PackPath:            "/tmp/pack.sql",
	}

	if state.ExecutedStatements != 50 {
		t.Errorf("expected 50 executed statements, got %d", state.ExecutedStatements)
	}

	if state.TotalStatements != 100 {
		t.Errorf("expected 100 total statements, got %d", state.TotalStatements)
	}

	if state.PackPath != "/tmp/pack.sql" {
		t.Errorf("expected pack path /tmp/pack.sql, got %s", state.PackPath)
	}
}

