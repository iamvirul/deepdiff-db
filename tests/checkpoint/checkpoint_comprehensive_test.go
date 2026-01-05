package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/checkpoint"
	"github.com/iamvirul/deepdiff-db/pkg/config"
)

func TestManager_Save_ErrorPaths(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	// Test nil state
	err := mgr.Save(nil)
	if err == nil {
		t.Error("expected error for nil state")
	}

	// Test with invalid directory (read-only)
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o555); err != nil {
		t.Fatalf("failed to create read-only dir: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755) // Restore for cleanup

	readOnlyMgr := checkpoint.NewManager(readOnlyDir)
	cfg := &config.Config{
		Prod: config.DBConfig{Driver: "sqlite", Database: "/tmp/prod.db"},
		Dev:  config.DBConfig{Driver: "sqlite", Database: "/tmp/dev.db"},
	}
	state, _ := checkpoint.NewState(checkpoint.OperationTypeHashTable, readOnlyDir, cfg)
	
	// This might fail on some systems, but should handle gracefully
	_ = readOnlyMgr.Save(state)
}

func TestManager_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	// Write invalid JSON
	checkpointPath := mgr.Path()
	if err := os.MkdirAll(filepath.Dir(checkpointPath), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(checkpointPath, []byte("invalid json"), 0o644); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}

	state, err := mgr.Load()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if state != nil {
		t.Error("expected nil state for invalid JSON")
	}
}

func TestManager_Load_WrongVersion(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	// Create checkpoint with wrong version
	checkpointPath := mgr.Path()
	if err := os.MkdirAll(filepath.Dir(checkpointPath), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	invalidState := `{
		"version": "0.0.0",
		"operation": "hash_table",
		"output_dir": "` + tmpDir + `"
	}`

	if err := os.WriteFile(checkpointPath, []byte(invalidState), 0o644); err != nil {
		t.Fatalf("failed to write checkpoint: %v", err)
	}

	state, err := mgr.Load()
	if err == nil {
		t.Error("expected error for wrong version")
	}
	if state != nil {
		t.Error("expected nil state for wrong version")
	}
}

func TestManager_Update(t *testing.T) {
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

	// Update the state
	err = mgr.Update(func(s *checkpoint.State) error {
		if s.HashTableState == nil {
			s.HashTableState = &checkpoint.HashTableState{}
		}
		s.HashTableState.CompletedTables = []string{"table1"}
		return nil
	})

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update was saved
	loaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.HashTableState == nil {
		t.Fatal("expected HashTableState to be set")
	}
	if len(loaded.HashTableState.CompletedTables) != 1 {
		t.Errorf("expected 1 completed table, got %d", len(loaded.HashTableState.CompletedTables))
	}
}

func TestManager_Update_NoState(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	err := mgr.Update(func(s *checkpoint.State) error {
		return nil
	})

	if err == nil {
		t.Error("expected error when no state is loaded")
	}
}

func TestManager_Update_ErrorInFunction(t *testing.T) {
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

	updateErr := mgr.Update(func(s *checkpoint.State) error {
		return os.ErrPermission
	})

	if updateErr != os.ErrPermission {
		t.Errorf("expected update error, got: %v", updateErr)
	}
}

func TestManager_Delete(t *testing.T) {
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
		t.Error("expected checkpoint to exist")
	}

	if err := mgr.Delete(); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if mgr.HasCheckpoint() {
		t.Error("expected checkpoint to be deleted")
	}
}

func TestManager_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	// Delete when no checkpoint exists should not error
	if err := mgr.Delete(); err != nil {
		t.Errorf("Delete of non-existent checkpoint should not error, got: %v", err)
	}
}

func TestManager_HasCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	if mgr.HasCheckpoint() {
		t.Error("expected no checkpoint initially")
	}

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
		t.Error("expected checkpoint to exist after Save")
	}
}

func TestToContext_FromContext(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	ctx := context.Background()
	ctx = checkpoint.ToContext(ctx, mgr)

	retrieved := checkpoint.FromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected non-nil manager from context")
	}

	if retrieved.Path() != mgr.Path() {
		t.Errorf("expected same path, got: %s vs %s", retrieved.Path(), mgr.Path())
	}
}

func TestToContext_NilContext(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := checkpoint.NewManager(tmpDir)

	ctx := checkpoint.ToContext(nil, mgr)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	retrieved := checkpoint.FromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected manager from context")
	}
}

func TestFromContext_NilContext(t *testing.T) {
	mgr := checkpoint.FromContext(nil)
	if mgr != nil {
		t.Error("expected nil manager from nil context")
	}
}

func TestFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	mgr := checkpoint.FromContext(ctx)
	if mgr != nil {
		t.Error("expected nil manager for wrong type in context")
	}
}

func TestManager_Save_AtomicWrite(t *testing.T) {
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

	// Save should use atomic write (temp file then rename)
	if err := mgr.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Temp file should not exist
	tmpPath := mgr.Path() + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("expected temp file to be cleaned up")
	}

	// Checkpoint file should exist
	if !mgr.HasCheckpoint() {
		t.Error("expected checkpoint file to exist")
	}
}

