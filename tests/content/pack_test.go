package main

import (
	"github.com/iamvirul/deepdiff-db/internal/content"

	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

func TestGeneratePack(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create dev database with test data
	devDB, err := sql.Open("sqlite", ":memory:")
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
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(1, 'Alice', 'alice@example.com'),
		(2, 'Bob', 'bob@example.com'),
		(3, 'Charlie', 'charlie@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "integer", IsNullable: false},
					"name":  {Name: "name", DataType: "text", IsNullable: false},
					"email": {Name: "email", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "users",
				Added:   []string{"3"},
				Removed: []string{"1"},
				Updated: []string{"2"},
			},
		},
	}

	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, devSchema, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("content.content.GeneratePack failed: %v", err)
	}

	// Verify pack file exists
	if _, err := os.Stat(packPath); os.IsNotExist(err) {
		t.Fatalf("pack file does not exist: %s", packPath)
	}

	// Read and verify pack content
	content, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(content)
	if !strings.Contains(sqlText, "BEGIN;") {
		t.Error("pack should start with BEGIN;")
	}
	if !strings.Contains(sqlText, "COMMIT;") {
		t.Error("pack should end with COMMIT;")
	}
	if !strings.Contains(sqlText, "DELETE FROM") {
		t.Error("pack should contain DELETE statements")
	}
	if !strings.Contains(sqlText, "INSERT INTO") {
		t.Error("pack should contain INSERT statements")
	}

	// Verify it's a transaction
	lines := strings.Split(strings.TrimSpace(sqlText), "\n")
	if !strings.Contains(lines[0], "BEGIN") {
		t.Error("first statement should be BEGIN")
	}
	if !strings.Contains(lines[len(lines)-1], "COMMIT") {
		t.Error("last statement should be COMMIT")
	}
}

func TestGeneratePack_NoChanges(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "users"}, // No changes
		},
	}

	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, devSchema, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("content.content.GeneratePack failed: %v", err)
	}

	// Read pack content
	content, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(content)
	// Should only have BEGIN and COMMIT
	lines := strings.Split(strings.TrimSpace(sqlText), "\n")
	if len(lines) != 2 {
		t.Errorf("expected only BEGIN and COMMIT, got %d lines", len(lines))
	}
}

func TestGeneratePack_WithIgnore(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			updated_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, updated_at) VALUES
		(1, 'Alice', '2024-01-01')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "integer", IsNullable: false},
					"name":       {Name: "name", DataType: "text", IsNullable: false},
					"updated_at": {Name: "updated_at", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1"},
			},
		},
	}

	ignoreFn := content.IgnoreMatcher([]string{"*.updated_at"})
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, devSchema, diff, ignoreFn, tmpDir)
	if err != nil {
		t.Fatalf("content.content.GeneratePack failed: %v", err)
	}

	content, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(content)
	// Should not include updated_at in INSERT
	if strings.Contains(sqlText, "updated_at") {
		t.Error("pack should not include ignored columns")
	}
}









func TestGeneratePack_NoPrimaryKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:       "users",
				Columns:   map[string]schema.Column{},
				PrimaryKey: []string{}, // No primary key
			},
		},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "users", Added: []string{"1"}},
		},
	}

	_, err = content.GeneratePack(ctx, "sqlite", devDB, devSchema, diff, nil, tmpDir)
	if err == nil {
		t.Error("expected error for table without primary key")
	}
}

func TestGeneratePack_TableNotFound(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "nonexistent", Added: []string{"1"}},
		},
	}

	// content.content.GeneratePack skips tables not in schema, so it should succeed but produce empty pack
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, devSchema, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("content.content.GeneratePack should not error for nonexistent table (it skips it): %v", err)
	}

	// Verify pack only has BEGIN and COMMIT
	content, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(content)
	// Should only have BEGIN and COMMIT, no actual statements
	lines := strings.Split(strings.TrimSpace(sqlText), "\n")
	if len(lines) > 2 {
		t.Errorf("expected only BEGIN and COMMIT, got %d lines", len(lines))
	}
}

