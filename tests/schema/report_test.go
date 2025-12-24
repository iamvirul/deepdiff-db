package main

import (
	"github.com/iamvirul/deepdiff-db/internal/schema"

	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReports(t *testing.T) {
	tmpDir := t.TempDir()

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				Table:          "users",
				HasDifferences: true,
				OnlyInProd:     true,
				MissingInDev:   true,
			},
			{
				Name:           "posts",
				Table:          "posts",
				HasDifferences: true,
				ColumnDiffs: []schema.ColumnDiff{
					{
						Column:        "title",
						TypeMismatch:   true,
						ProdType:       "varchar",
						DevType:        "text",
					},
				},
			},
			{
				Name:           "comments",
				Table:          "comments",
				HasDifferences: false,
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("schema.schema.WriteReports failed: %v", err)
	}

	// Check JSON file
	jsonPath := filepath.Join(tmpDir, "schema_diff.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("JSON file was not created")
	}

	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	if !strings.Contains(string(jsonContent), "users") {
		t.Error("JSON should contain 'users'")
	}
	if !strings.Contains(string(jsonContent), "posts") {
		t.Error("JSON should contain 'posts'")
	}

	// Check text file
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	if _, err := os.Stat(textPath); os.IsNotExist(err) {
		t.Fatal("Text file was not created")
	}

	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "users") {
		t.Error("Text should contain 'users'")
	}
	if !strings.Contains(textStr, "posts") {
		t.Error("Text should contain 'posts'")
	}
	// Comments table should not appear (no differences)
	if strings.Contains(textStr, "comments") {
		t.Error("Text should not contain 'comments' (no differences)")
	}
}

func TestWriteReports_NoDifferences(t *testing.T) {
	tmpDir := t.TempDir()

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				Table:          "users",
				HasDifferences: false,
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("schema.schema.WriteReports failed: %v", err)
	}

	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "Schema: OK") {
		t.Error("Text should indicate no differences")
	}
}

func TestWriteReports_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "nested", "output", "dir")

	result := schema.DiffResult{
		Tables: []schema.TableDiff{},
	}

	if err := schema.WriteReports(result, outDir); err != nil {
		t.Fatalf("schema.schema.WriteReports failed: %v", err)
	}

	// Directory should be created
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		t.Fatal("Output directory was not created")
	}

	// Files should exist
	jsonPath := filepath.Join(outDir, "schema_diff.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("JSON file was not created")
	}
}

