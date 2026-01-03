package html_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/internal/content"
	htmlreport "github.com/iamvirul/deepdiff-db/internal/report/html"
	"github.com/iamvirul/deepdiff-db/internal/schema"
)

func TestNewGenerator(t *testing.T) {
	// Test with nil options (should use defaults)
	g := htmlreport.NewGenerator(nil)
	if g == nil {
		t.Fatal("expected generator to be created with nil options")
	}

	// Test with custom options
	opts := &htmlreport.ReportOptions{
		IncludeMigrationSQL: false,
		IncludeDetailedKeys: false,
		MaxKeysPerTable:     50,
		EnablePDFExport:     false,
	}
	g = htmlreport.NewGenerator(opts)
	if g == nil {
		t.Fatal("expected generator to be created with custom options")
	}
}

func TestDefaultReportOptions(t *testing.T) {
	opts := htmlreport.DefaultReportOptions()
	if opts == nil {
		t.Fatal("expected default options to be non-nil")
	}
	if !opts.IncludeMigrationSQL {
		t.Error("expected IncludeMigrationSQL to be true by default")
	}
	if !opts.IncludeDetailedKeys {
		t.Error("expected IncludeDetailedKeys to be true by default")
	}
	if opts.MaxKeysPerTable != 100 {
		t.Errorf("expected MaxKeysPerTable to be 100, got %d", opts.MaxKeysPerTable)
	}
	if !opts.EnablePDFExport {
		t.Error("expected EnablePDFExport to be true by default")
	}
}

func TestBuildReportData_Empty(t *testing.T) {
	data := htmlreport.BuildReportData(
		"localhost:3306/prod",
		"localhost:3306/dev",
		nil, nil, nil, nil,
		"", "", 0, nil,
	)

	if data == nil {
		t.Fatal("expected report data to be created")
	}
	if data.ProdDB != "localhost:3306/prod" {
		t.Errorf("expected ProdDB 'localhost:3306/prod', got '%s'", data.ProdDB)
	}
	if data.DevDB != "localhost:3306/dev" {
		t.Errorf("expected DevDB 'localhost:3306/dev', got '%s'", data.DevDB)
	}
	if data.Summary.SchemaStatus != "OK" {
		t.Errorf("expected SchemaStatus 'OK', got '%s'", data.Summary.SchemaStatus)
	}
}

func TestBuildReportData_WithSchemaDiff(t *testing.T) {
	schemaDiff := &schema.DiffResult{
		AddedTables: []schema.Table{
			{Name: "new_table"},
		},
		RemovedTables: []string{"old_table"},
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedColumns: []schema.Column{
					{Name: "email", DataType: "varchar(255)", IsNullable: false},
				},
			},
		},
	}

	data := htmlreport.BuildReportData(
		"localhost:3306/prod",
		"localhost:3306/dev",
		schemaDiff, nil, nil, nil,
		"", "", 5, nil,
	)

	if !data.HasSchemaDiff {
		t.Error("expected HasSchemaDiff to be true")
	}
	if data.Summary.SchemaStatus != "DRIFT" {
		t.Errorf("expected SchemaStatus 'DRIFT', got '%s'", data.Summary.SchemaStatus)
	}
	if len(data.SchemaChanges) != 3 {
		t.Errorf("expected 3 schema changes, got %d", len(data.SchemaChanges))
	}
}

func TestBuildReportData_WithDataDiff(t *testing.T) {
	dataDiff := &content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "users",
				Added:   []string{"1", "2", "3"},
				Removed: []string{"4"},
				Updated: []string{"5", "6"},
			},
		},
	}

	data := htmlreport.BuildReportData(
		"localhost:3306/prod",
		"localhost:3306/dev",
		nil, dataDiff, nil, nil,
		"", "", 5, nil,
	)

	if !data.HasDataDiff {
		t.Error("expected HasDataDiff to be true")
	}
	if data.Summary.AddedRows != 3 {
		t.Errorf("expected 3 added rows, got %d", data.Summary.AddedRows)
	}
	if data.Summary.RemovedRows != 1 {
		t.Errorf("expected 1 removed row, got %d", data.Summary.RemovedRows)
	}
	if data.Summary.UpdatedRows != 2 {
		t.Errorf("expected 2 updated rows, got %d", data.Summary.UpdatedRows)
	}
}

func TestBuildReportData_WithConflicts(t *testing.T) {
	conflicts := &content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "1", ProdHash: "abc123", DevHash: "def456"},
			{Table: "users", Key: "2", ProdHash: "ghi789", DevHash: "jkl012"},
		},
	}

	data := htmlreport.BuildReportData(
		"localhost:3306/prod",
		"localhost:3306/dev",
		nil, nil, conflicts, nil,
		"", "", 5, nil,
	)

	if !data.HasConflicts {
		t.Error("expected HasConflicts to be true")
	}
	if data.Summary.TotalConflicts != 2 {
		t.Errorf("expected 2 conflicts, got %d", data.Summary.TotalConflicts)
	}
	if len(data.ConflictItems) != 2 {
		t.Errorf("expected 2 conflict items, got %d", len(data.ConflictItems))
	}
}

func TestBuildReportData_WithResolutions(t *testing.T) {
	resInfo := &content.ResolutionInfo{
		TotalConflicts:  10,
		ResolvedCount:   7,
		UnresolvedCount: 3,
		ByDecision: map[string]int{
			"keep_prod": 5,
			"use_dev":   2,
		},
		ByTable: map[string]int{
			"users":  5,
			"orders": 5,
		},
	}

	data := htmlreport.BuildReportData(
		"localhost:3306/prod",
		"localhost:3306/dev",
		nil, nil, nil, resInfo,
		"", "", 5, nil,
	)

	if !data.HasResolutions {
		t.Error("expected HasResolutions to be true")
	}
	if data.Summary.ResolvedConflicts != 7 {
		t.Errorf("expected 7 resolved conflicts, got %d", data.Summary.ResolvedConflicts)
	}
	if data.Summary.PendingConflicts != 3 {
		t.Errorf("expected 3 pending conflicts, got %d", data.Summary.PendingConflicts)
	}
}

func TestBuildReportData_WithMigrationSQL(t *testing.T) {
	migrationSQL := `BEGIN;
INSERT INTO users (id, name) VALUES (1, 'John');
COMMIT;`

	data := htmlreport.BuildReportData(
		"localhost:3306/prod",
		"localhost:3306/dev",
		nil, nil, nil, nil,
		migrationSQL, "migration_pack.sql", 5, nil,
	)

	if !data.HasMigration {
		t.Error("expected HasMigration to be true")
	}
	if data.MigrationSQL != migrationSQL {
		t.Error("expected MigrationSQL to match input")
	}
	if data.MigrationPack != "migration_pack.sql" {
		t.Errorf("expected MigrationPack 'migration_pack.sql', got '%s'", data.MigrationPack)
	}
}

func TestGenerateReport(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "report.html")

	data := &htmlreport.ReportData{
		GeneratedAt:   time.Now(),
		Version:       "v0.5-test",
		ProdDB:        "localhost:3306/prod_db",
		DevDB:         "localhost:3306/dev_db",
		MigrationPack: "migration_pack.sql",
		Summary: htmlreport.ReportSummary{
			SchemaStatus:      "OK",
			TablesScanned:     10,
			TablesWithChanges: 3,
			AddedRows:         15,
			RemovedRows:       2,
			UpdatedRows:       8,
			TotalConflicts:    5,
			ResolvedConflicts: 3,
			PendingConflicts:  2,
		},
		HasSchemaDiff: false,
		HasDataDiff:   true,
		HasConflicts:  true,
		TableDiffs: []htmlreport.TableDiffDisplay{
			{
				Table:        "users",
				AddedCount:   5,
				RemovedCount: 1,
				UpdatedCount: 3,
				AddedKeys:    []string{"1", "2", "3", "4", "5"},
				RemovedKeys:  []string{"10"},
				UpdatedKeys:  []string{"6", "7", "8"},
				HasChanges:   true,
			},
		},
		ConflictItems: []htmlreport.ConflictDisplay{
			{
				Table:      "users",
				Key:        "100",
				ProdHash:   "abc123def...",
				DevHash:    "xyz789ghi...",
				Resolution: "pending",
				IsResolved: false,
			},
		},
		MigrationSQL: "BEGIN;\nINSERT INTO users VALUES (1, 'test');\nCOMMIT;",
		HasMigration: true,
	}

	generator := htmlreport.NewGenerator(nil)
	err := generator.GenerateReport(data, outPath)
	if err != nil {
		t.Fatalf("failed to generate report: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatal("expected report file to exist")
	}

	// Read and verify content
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}

	html := string(content)

	// Check for key elements
	checks := []string{
		"<!DOCTYPE html>",
		"Database Diff Report",
		"localhost:3306/prod_db",
		"localhost:3306/dev_db",
		"v0.5-test",
		"Schema",
		"Data",
		"Conflicts",
		"Migration",
		"users",
		"Export PDF",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain '%s'", check)
		}
	}
}

func TestGenerateReport_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "subdir", "nested", "report.html")

	data := &htmlreport.ReportData{
		GeneratedAt: time.Now(),
		Version:     "v0.5",
		ProdDB:      "prod",
		DevDB:       "dev",
		Summary: htmlreport.ReportSummary{
			SchemaStatus: "OK",
		},
	}

	generator := htmlreport.NewGenerator(nil)
	err := generator.GenerateReport(data, outPath)
	if err != nil {
		t.Fatalf("failed to generate report in nested directory: %v", err)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatal("expected report file to exist in nested directory")
	}
}

func TestSchemaChangeDisplay(t *testing.T) {
	schemaDiff := &schema.DiffResult{
		AddedTables: []schema.Table{
			{Name: "new_table"},
		},
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedColumns: []schema.Column{
					{Name: "email", DataType: "varchar(255)", IsNullable: false},
				},
				RemovedColumns: []schema.Column{
					{Name: "old_field", DataType: "text", IsNullable: true},
				},
				ModifiedColumns: []schema.ColumnDiff{
					{
						Column:           "name",
						TypeMismatch:     true,
						ProdType:         "varchar(100)",
						DevType:          "varchar(255)",
						NullableMismatch: true,
						ProdNullable:     boolPtr(false),
						DevNullable:      boolPtr(true),
					},
				},
				AddedIndexes: []schema.Index{
					{Name: "idx_email", Columns: []string{"email"}, IsUnique: true},
				},
				RemovedIndexes: []schema.Index{
					{Name: "idx_old", Columns: []string{"old_field"}, IsUnique: false},
				},
			},
		},
	}

	data := htmlreport.BuildReportData(
		"prod", "dev",
		schemaDiff, nil, nil, nil,
		"", "", 5, nil,
	)

	// Check schema changes were built correctly
	if len(data.SchemaChanges) != 2 {
		t.Fatalf("expected 2 schema changes (added table + modified), got %d", len(data.SchemaChanges))
	}

	// Find the modified table change
	var modifiedChange *htmlreport.SchemaChangeDisplay
	for i := range data.SchemaChanges {
		if data.SchemaChanges[i].Table == "users" {
			modifiedChange = &data.SchemaChanges[i]
			break
		}
	}

	if modifiedChange == nil {
		t.Fatal("expected to find 'users' table in schema changes")
	}

	if len(modifiedChange.ColumnChanges) != 3 {
		t.Errorf("expected 3 column changes, got %d", len(modifiedChange.ColumnChanges))
	}

	if len(modifiedChange.IndexChanges) != 2 {
		t.Errorf("expected 2 index changes, got %d", len(modifiedChange.IndexChanges))
	}
}

func TestTableDiffDisplay_KeyLimiting(t *testing.T) {
	dataDiff := &content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "large_table",
				Added:   generateKeys(150),
				Removed: generateKeys(50),
				Updated: generateKeys(200),
			},
		},
	}

	// With default options (MaxKeysPerTable = 100)
	data := htmlreport.BuildReportData(
		"prod", "dev",
		nil, dataDiff, nil, nil,
		"", "", 1, nil,
	)

	if len(data.TableDiffs) != 1 {
		t.Fatalf("expected 1 table diff, got %d", len(data.TableDiffs))
	}

	td := data.TableDiffs[0]
	if len(td.AddedKeys) != 100 {
		t.Errorf("expected 100 added keys (limited), got %d", len(td.AddedKeys))
	}
	if len(td.RemovedKeys) != 50 {
		t.Errorf("expected 50 removed keys (not limited), got %d", len(td.RemovedKeys))
	}
	if len(td.UpdatedKeys) != 100 {
		t.Errorf("expected 100 updated keys (limited), got %d", len(td.UpdatedKeys))
	}

	// With custom options
	opts := &htmlreport.ReportOptions{
		IncludeDetailedKeys: true,
		MaxKeysPerTable:     25,
	}
	data = htmlreport.BuildReportData(
		"prod", "dev",
		nil, dataDiff, nil, nil,
		"", "", 1, opts,
	)

	td = data.TableDiffs[0]
	if len(td.AddedKeys) != 25 {
		t.Errorf("expected 25 added keys (custom limit), got %d", len(td.AddedKeys))
	}
}

func TestTableDiffDisplay_NoDetailedKeys(t *testing.T) {
	dataDiff := &content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "users",
				Added:   []string{"1", "2"},
				Removed: []string{"3"},
				Updated: []string{"4"},
			},
		},
	}

	opts := &htmlreport.ReportOptions{
		IncludeDetailedKeys: false,
	}
	data := htmlreport.BuildReportData(
		"prod", "dev",
		nil, dataDiff, nil, nil,
		"", "", 1, opts,
	)

	td := data.TableDiffs[0]
	if len(td.AddedKeys) != 0 {
		t.Errorf("expected no added keys when IncludeDetailedKeys=false, got %d", len(td.AddedKeys))
	}
}

// Helper functions

func boolPtr(b bool) *bool {
	return &b
}

func generateKeys(n int) []string {
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = string(rune('a' + i%26))
	}
	return keys
}
