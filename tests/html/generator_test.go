package html_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
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
		nil, nil, nil, nil, nil,
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
		schemaDiff, nil, nil, nil, nil,
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
		nil, dataDiff, nil, nil, nil,
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
		nil, nil, conflicts, nil, nil,
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
		nil, nil, nil, resInfo, nil,
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
		nil, nil, nil, nil, nil,
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
		schemaDiff, nil, nil, nil, nil,
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
		nil, dataDiff, nil, nil, nil,
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
		nil, dataDiff, nil, nil, nil,
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
		nil, dataDiff, nil, nil, nil,
		"", "", 1, opts,
	)

	td := data.TableDiffs[0]
	if len(td.AddedKeys) != 0 {
		t.Errorf("expected no added keys when IncludeDetailedKeys=false, got %d", len(td.AddedKeys))
	}
}

// ============================================================================
// buildSchemaChanges — FK modifications with OnDelete / OnUpdate
// ============================================================================

func TestBuildReportData_WithFKModifications(t *testing.T) {
	schemaDiff := &schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				ModifiedForeignKeys: []schema.ForeignKeyDiff{
					{
						Name:            "fk_on_delete",
						OnDeleteDiffers: true,
						ProdOnDelete:    "CASCADE",
						DevOnDelete:     "RESTRICT",
					},
					{
						Name:            "fk_on_update",
						OnUpdateDiffers: true,
						ProdOnUpdate:    "SET NULL",
						DevOnUpdate:     "CASCADE",
					},
					{
						Name:            "fk_both",
						OnDeleteDiffers: true,
						ProdOnDelete:    "CASCADE",
						DevOnDelete:     "SET NULL",
						OnUpdateDiffers: true,
						ProdOnUpdate:    "NO ACTION",
						DevOnUpdate:     "RESTRICT",
					},
				},
				RemovedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_removed",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
					},
				},
			},
		},
	}

	data := htmlreport.BuildReportData(
		"prod", "dev",
		schemaDiff, nil, nil, nil, nil,
		"", "", 1, nil,
	)

	if !data.HasSchemaDiff {
		t.Error("expected HasSchemaDiff")
	}
	if len(data.SchemaChanges) != 1 {
		t.Fatalf("expected 1 schema change, got %d", len(data.SchemaChanges))
	}

	fkChanges := data.SchemaChanges[0].ForeignKeyChanges
	if len(fkChanges) != 4 {
		t.Errorf("expected 4 FK changes (3 modified + 1 removed), got %d", len(fkChanges))
	}
}

// ============================================================================
// buildSummary — conflict path with nil resInfo (PendingConflicts fallback)
// ============================================================================

func TestBuildReportData_ConflictsWithoutResolutionInfo(t *testing.T) {
	conflicts := &content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "1", ProdHash: "abc", DevHash: "def"},
			{Table: "users", Key: "2", ProdHash: "ghi", DevHash: "jkl"},
		},
	}

	// Explicitly pass nil for resInfo — exercises the else branch in buildSummary
	data := htmlreport.BuildReportData(
		"prod", "dev",
		nil, nil, conflicts, nil, nil,
		"", "", 1, nil,
	)

	if data.Summary.TotalConflicts != 2 {
		t.Errorf("expected TotalConflicts=2, got %d", data.Summary.TotalConflicts)
	}
	// When resInfo is nil, PendingConflicts should equal total conflicts
	if data.Summary.PendingConflicts != 2 {
		t.Errorf("expected PendingConflicts=2 (fallback from nil resInfo), got %d", data.Summary.PendingConflicts)
	}
	if data.Summary.ResolvedConflicts != 0 {
		t.Errorf("expected ResolvedConflicts=0, got %d", data.Summary.ResolvedConflicts)
	}
}

// ============================================================================
// buildReportData_WithResolutions — exercises buildResolutionBreakdown
// ============================================================================

func TestBuildReportData_WithResolutionsSlice(t *testing.T) {
	conflicts := &content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "1", ProdHash: "aaa", DevHash: "bbb"},
			{Table: "users", Key: "2", ProdHash: "ccc", DevHash: "ddd"},
			{Table: "orders", Key: "1", ProdHash: "eee", DevHash: "fff"},
		},
	}

	resolutions := []resolve.Resolution{
		{
			Conflict: content.Conflict{Table: "users", Key: "1"},
			Strategy: resolve.StrategyOurs,
			Decision: resolve.DecisionKeepProd,
			Resolved: true,
		},
		{
			Conflict: content.Conflict{Table: "users", Key: "2"},
			Strategy: resolve.StrategyTheirs,
			Decision: resolve.DecisionUseDev,
			Resolved: true,
		},
		{
			Conflict: content.Conflict{Table: "orders", Key: "1"},
			Strategy: resolve.StrategyManual,
			Decision: resolve.DecisionPending,
			Resolved: false,
		},
	}

	data := htmlreport.BuildReportData(
		"prod", "dev",
		nil, nil, conflicts, nil, resolutions,
		"", "", 3, nil,
	)

	if !data.HasResolutions {
		t.Error("expected HasResolutions when resolutions slice is non-empty")
	}
	if data.ResolutionBreakdown == nil {
		t.Fatal("expected ResolutionBreakdown to be populated")
	}

	bd := data.ResolutionBreakdown
	if bd.TotalConflicts != 3 {
		t.Errorf("expected TotalConflicts=3, got %d", bd.TotalConflicts)
	}
	if bd.AutoResolvedOurs != 1 {
		t.Errorf("expected AutoResolvedOurs=1, got %d", bd.AutoResolvedOurs)
	}
	if bd.AutoResolvedTheirs != 1 {
		t.Errorf("expected AutoResolvedTheirs=1, got %d", bd.AutoResolvedTheirs)
	}
	if bd.PendingManual != 1 {
		t.Errorf("expected PendingManual=1, got %d", bd.PendingManual)
	}
	// Verify TableStrategies are sorted
	if len(bd.TableStrategies) != 2 {
		t.Errorf("expected 2 table strategies (users, orders), got %d", len(bd.TableStrategies))
	}
	if bd.TableStrategies[0].Table != "orders" {
		t.Errorf("expected first table to be 'orders' (sorted), got %s", bd.TableStrategies[0].Table)
	}

	// Also test that conflict items are resolved correctly from the resolution map
	if len(data.ConflictItems) != 3 {
		t.Errorf("expected 3 conflict items, got %d", len(data.ConflictItems))
	}
	for _, item := range data.ConflictItems {
		if item.Table == "users" && item.Key == "1" {
			if !item.IsResolved {
				t.Error("expected users:1 to be resolved")
			}
			if item.Decision != string(resolve.DecisionKeepProd) {
				t.Errorf("expected keep_prod decision, got %s", item.Decision)
			}
		}
	}
}

// ============================================================================
// GenerateReport error path — directory creation failure
// ============================================================================

func TestGenerateReport_InvalidPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping path error test in short mode")
	}

	// Use a path with a null byte which is always invalid on all platforms.
	// filepath.Dir on it still produces something, but os.MkdirAll will fail.
	invalidPath := "/tmp/\x00invalid/report.html"

	data := &htmlreport.ReportData{
		GeneratedAt: time.Now(),
		Version:     "v0.5",
		ProdDB:      "prod",
		DevDB:       "dev",
		Summary:     htmlreport.ReportSummary{SchemaStatus: "OK"},
	}

	generator := htmlreport.NewGenerator(nil)
	err := generator.GenerateReport(data, invalidPath)
	if err == nil {
		t.Error("expected error for invalid output path")
	}
}

// ============================================================================
// buildSummary — empty dataDiff and nil conflicts branches
// ============================================================================

func TestBuildReportData_SummaryWithNilDataDiffAndConflicts(t *testing.T) {
	data := htmlreport.BuildReportData(
		"prod", "dev",
		nil, nil, nil, nil, nil,
		"", "", 5, nil,
	)

	if data.Summary.AddedRows != 0 {
		t.Errorf("expected 0 added rows, got %d", data.Summary.AddedRows)
	}
	if data.Summary.TotalConflicts != 0 {
		t.Errorf("expected 0 total conflicts, got %d", data.Summary.TotalConflicts)
	}
	if data.Summary.PendingConflicts != 0 {
		t.Errorf("expected 0 pending conflicts, got %d", data.Summary.PendingConflicts)
	}
	if data.Summary.TablesScanned != 5 {
		t.Errorf("expected TablesScanned=5, got %d", data.Summary.TablesScanned)
	}
}

// ============================================================================
// buildSchemaChanges — table with differences but no column/index/FK changes
// (HasDifferences=true but all change slices are empty — should not appear)
// ============================================================================

func TestBuildReportData_SchemaChangeTableWithNoSubChanges(t *testing.T) {
	schemaDiff := &schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				// No added/removed/modified columns, indexes, or FKs
				// This exercises the guard: only append if there are sub-changes
			},
		},
	}

	data := htmlreport.BuildReportData(
		"prod", "dev",
		schemaDiff, nil, nil, nil, nil,
		"", "", 1, nil,
	)

	// HasSchemaDiff should be true (HasDrift returns true)
	if !data.HasSchemaDiff {
		t.Error("expected HasSchemaDiff=true")
	}
	// But no SchemaChanges because the table has no sub-changes to display
	if len(data.SchemaChanges) != 0 {
		t.Errorf("expected 0 SchemaChanges when table has no column/index/FK changes, got %d", len(data.SchemaChanges))
	}
}

// ============================================================================
// MigrationSQL not included when IncludeMigrationSQL is false
// ============================================================================

func TestBuildReportData_MigrationSQLNotIncluded(t *testing.T) {
	opts := &htmlreport.ReportOptions{
		IncludeMigrationSQL: false,
		IncludeDetailedKeys: false,
		MaxKeysPerTable:     10,
	}

	data := htmlreport.BuildReportData(
		"prod", "dev",
		nil, nil, nil, nil, nil,
		"SELECT 1", "", 0, opts,
	)

	if data.HasMigration {
		t.Error("expected HasMigration=false when IncludeMigrationSQL is false")
	}
	if data.MigrationSQL != "" {
		t.Errorf("expected empty MigrationSQL, got %q", data.MigrationSQL)
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
