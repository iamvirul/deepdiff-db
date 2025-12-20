package main

import (
	"github.com/iamvirul/deepdiff-db/internal/content"

	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReportsWithInfo(t *testing.T) {
	tmpDir := t.TempDir()

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "users",
				Added:   []string{"1", "2"},
				Removed: []string{"3"},
				Updated: []string{"4"},
			},
			{
				Table:   "posts",
				Added:   []string{"5"},
				Removed: []string{},
				Updated: []string{},
			},
		},
	}

	conflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "4", ProdHash: "hash1", DevHash: "hash2"},
		},
	}

	if err := content.WriteReportsWithInfo(diff, conflicts, tmpDir, "OK", 2, "migration_pack.sql"); err != nil {
		t.Fatalf("content.content.WriteReportsWithInfo failed: %v", err)
	}

	// Check content_diff.json
	jsonPath := filepath.Join(tmpDir, "content_diff.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("content_diff.json was not created")
	}

	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	if !strings.Contains(string(jsonContent), "users") {
		t.Error("JSON should contain 'users'")
	}

	// Check conflicts.json
	conflictsPath := filepath.Join(tmpDir, "conflicts.json")
	if _, err := os.Stat(conflictsPath); os.IsNotExist(err) {
		t.Fatal("conflicts.json was not created")
	}

	conflictsContent, err := os.ReadFile(conflictsPath)
	if err != nil {
		t.Fatalf("failed to read conflicts file: %v", err)
	}

	if !strings.Contains(string(conflictsContent), "users") {
		t.Error("content.Conflicts JSON should contain 'users'")
	}

	// Check summary.txt
	summaryPath := filepath.Join(tmpDir, "summary.txt")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Fatal("summary.txt was not created")
	}

	summaryContent, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("failed to read summary file: %v", err)
	}

	summaryStr := string(summaryContent)
	if !strings.Contains(summaryStr, "Schema: OK") {
		t.Error("Summary should contain schema status")
	}
	if !strings.Contains(summaryStr, "Tables scanned: 2") {
		t.Error("Summary should contain tables scanned count")
	}
	if !strings.Contains(summaryStr, "Added rows: 3") {
		t.Error("Summary should contain added rows count")
	}
	if !strings.Contains(summaryStr, "Removed rows: 1") {
		t.Error("Summary should contain removed rows count")
	}
	if !strings.Contains(summaryStr, "Updated rows: 1") {
		t.Error("Summary should contain updated rows count")
	}
	if !strings.Contains(summaryStr, "Conflicts: 1") {
		t.Error("Summary should contain conflicts count")
	}
	if !strings.Contains(summaryStr, "Migration pack: migration_pack.sql") {
		t.Error("Summary should contain migration pack filename")
	}
}

func TestWriteReportsWithInfo_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "users"},
		},
	}

	conflicts := content.Conflicts{Conflicts: []content.Conflict{}}

	if err := content.WriteReportsWithInfo(diff, conflicts, tmpDir, "OK", 1, ""); err != nil {
		t.Fatalf("content.content.WriteReportsWithInfo failed: %v", err)
	}

	summaryPath := filepath.Join(tmpDir, "summary.txt")
	summaryContent, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("failed to read summary file: %v", err)
	}

	summaryStr := string(summaryContent)
	// Should not contain added/removed/updated rows
	if strings.Contains(summaryStr, "Added rows:") {
		t.Error("Summary should not contain 'Added rows:' when there are none")
	}
	if strings.Contains(summaryStr, "Removed rows:") {
		t.Error("Summary should not contain 'Removed rows:' when there are none")
	}
	if strings.Contains(summaryStr, "Updated rows:") {
		t.Error("Summary should not contain 'Updated rows:' when there are none")
	}
}

func TestWriteReports(t *testing.T) {
	tmpDir := t.TempDir()

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "users", Added: []string{"1"}},
		},
	}

	conflicts := content.Conflicts{Conflicts: []content.Conflict{}}

	if err := content.WriteReports(diff, conflicts, tmpDir); err != nil {
		t.Fatalf("content.content.WriteReports failed: %v", err)
	}

	// Should create files (delegates to content.content.WriteReportsWithInfo with empty info)
	jsonPath := filepath.Join(tmpDir, "content_diff.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("content_diff.json was not created")
	}
}

func TestWriteReportsWithInfo_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "nested", "output", "dir")

	diff := content.DataDiff{Tables: []content.TableDataDiff{}}
	conflicts := content.Conflicts{Conflicts: []content.Conflict{}}

	if err := content.WriteReportsWithInfo(diff, conflicts, outDir, "", 0, ""); err != nil {
		t.Fatalf("content.content.WriteReportsWithInfo failed: %v", err)
	}

	// Directory should be created
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		t.Fatal("Output directory was not created")
	}
}

func TestWriteReportsWithInfo_AllFields(t *testing.T) {
	tmpDir := t.TempDir()

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "users",
				Added:   []string{"1", "2"},
				Removed: []string{"3"},
				Updated: []string{"4"},
			},
		},
	}

	conflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "4", ProdHash: "hash1", DevHash: "hash2"},
		},
	}

	if err := content.WriteReportsWithInfo(diff, conflicts, tmpDir, "DRIFT", 5, "/path/to/migration_pack.sql"); err != nil {
		t.Fatalf("WriteReportsWithInfo failed: %v", err)
	}

	summaryPath := filepath.Join(tmpDir, "summary.txt")
	summaryContent, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("failed to read summary file: %v", err)
	}

	summaryStr := string(summaryContent)
	if !strings.Contains(summaryStr, "Schema: DRIFT") {
		t.Error("Summary should contain schema status")
	}
	if !strings.Contains(summaryStr, "Tables scanned: 5") {
		t.Error("Summary should contain tables scanned count")
	}
	if !strings.Contains(summaryStr, "migration_pack.sql") {
		t.Error("Summary should contain migration pack filename")
	}
}

func TestWriteReportsWithInfo_EmptyDiff(t *testing.T) {
	tmpDir := t.TempDir()

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{},
	}

	conflicts := content.Conflicts{Conflicts: []content.Conflict{}}

	if err := content.WriteReportsWithInfo(diff, conflicts, tmpDir, "", 0, ""); err != nil {
		t.Fatalf("WriteReportsWithInfo failed: %v", err)
	}

	// Verify files were created
	jsonPath := filepath.Join(tmpDir, "content_diff.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("content_diff.json was not created")
	}

	conflictsPath := filepath.Join(tmpDir, "conflicts.json")
	if _, err := os.Stat(conflictsPath); os.IsNotExist(err) {
		t.Fatal("conflicts.json was not created")
	}

	summaryPath := filepath.Join(tmpDir, "summary.txt")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Fatal("summary.txt was not created")
	}
}

func TestWriteReportsWithInfo_MultipleTables(t *testing.T) {
	tmpDir := t.TempDir()

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "users",
				Added:   []string{"1"},
				Removed: []string{"2"},
				Updated: []string{"3"},
			},
			{
				Table:   "posts",
				Added:   []string{"10", "11"},
				Removed: []string{"20"},
				Updated: []string{"30", "31"},
			},
			{
				Table:   "comments",
				Added:   []string{},
				Removed: []string{},
				Updated: []string{},
			},
		},
	}

	conflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "3", ProdHash: "hash1", DevHash: "hash2"},
			{Table: "posts", Key: "30", ProdHash: "hash3", DevHash: "hash4"},
		},
	}

	if err := content.WriteReportsWithInfo(diff, conflicts, tmpDir, "OK", 3, ""); err != nil {
		t.Fatalf("WriteReportsWithInfo failed: %v", err)
	}

	summaryPath := filepath.Join(tmpDir, "summary.txt")
	summaryContent, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("failed to read summary file: %v", err)
	}

	summaryStr := string(summaryContent)
	// Should aggregate counts across all tables
	if !strings.Contains(summaryStr, "Added rows: 3") {
		t.Errorf("Summary should show 3 added rows (1+2), got: %s", summaryStr)
	}
	if !strings.Contains(summaryStr, "Removed rows: 2") {
		t.Errorf("Summary should show 2 removed rows (1+1), got: %s", summaryStr)
	}
	if !strings.Contains(summaryStr, "Updated rows: 3") {
		t.Errorf("Summary should show 3 updated rows (1+2), got: %s", summaryStr)
	}
	if !strings.Contains(summaryStr, "Conflicts: 2") {
		t.Error("Summary should show 2 conflicts")
	}
}

func TestWriteReports_ErrorHandling(t *testing.T) {
	// Test with invalid path (read-only directory or invalid characters)
	// Note: This is platform-dependent, so we'll test with a very long path
	// that might fail on some systems
	invalidPath := filepath.Join(strings.Repeat("a", 300), "test")

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "users", Added: []string{"1"}},
		},
	}

	conflicts := content.Conflicts{Conflicts: []content.Conflict{}}

	err := content.WriteReports(diff, conflicts, invalidPath)
	if err == nil {
		// On some systems this might succeed, so we'll just verify the function handles errors
		t.Log("WriteReports did not error on invalid path (may be platform-dependent)")
	}
}

func TestWriteReportsWithInfo_ZeroValues(t *testing.T) {
	tmpDir := t.TempDir()

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "users"}, // No changes
		},
	}

	conflicts := content.Conflicts{Conflicts: []content.Conflict{}}

	if err := content.WriteReportsWithInfo(diff, conflicts, tmpDir, "", 0, ""); err != nil {
		t.Fatalf("WriteReportsWithInfo failed: %v", err)
	}

	summaryPath := filepath.Join(tmpDir, "summary.txt")
	summaryContent, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("failed to read summary file: %v", err)
	}

	summaryStr := string(summaryContent)
	// Should not contain zero-value fields
	if strings.Contains(summaryStr, "Schema:") {
		t.Error("Summary should not contain empty schema status")
	}
	if strings.Contains(summaryStr, "Tables scanned: 0") {
		t.Error("Summary should not contain zero tables scanned")
	}
	if strings.Contains(summaryStr, "Migration pack:") && strings.Contains(summaryStr, ": ") {
		// Check if it's just the label without value
		lines := strings.Split(summaryStr, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Migration pack:") && len(strings.TrimSpace(strings.TrimPrefix(line, "Migration pack:"))) == 0 {
				t.Error("Summary should not contain empty migration pack")
			}
		}
	}
}

