package schema_test

// Tests for MigrationOptions — new Phase 1 fields (AllowDropView/Routine/Trigger/Sequence)
// and the behaviour of GenerateMigration when a DiffResult carries the new object types.
//
// Phase 1 adds the options and the diff structs only; the migration generator does not yet
// emit SQL for views/routines/triggers/sequences (that is Phase 2-4 work).  These tests
// verify:
//   - new AllowDrop* fields default to false (zero value)
//   - GenerateMigration does not error when DiffResult contains new object types
//   - existing table-level output is unaffected when new object types are also present
//   - AllowDropView/Routine/Trigger/Sequence can be set to true without breaking anything

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ── zero-value defaults ───────────────────────────────────────────────────────

func TestMigrationOptions_NewFields_DefaultFalse(t *testing.T) {
	opts := schema.MigrationOptions{}

	if opts.AllowDropView {
		t.Error("AllowDropView should default to false")
	}
	if opts.AllowDropRoutine {
		t.Error("AllowDropRoutine should default to false")
	}
	if opts.AllowDropTrigger {
		t.Error("AllowDropTrigger should default to false")
	}
	if opts.AllowDropSequence {
		t.Error("AllowDropSequence should default to false")
	}
}

func TestMigrationOptions_NewFields_CanBeSetTrue(t *testing.T) {
	opts := schema.MigrationOptions{
		AllowDropView:     true,
		AllowDropRoutine:  true,
		AllowDropTrigger:  true,
		AllowDropSequence: true,
	}

	if !opts.AllowDropView {
		t.Error("AllowDropView should be true")
	}
	if !opts.AllowDropRoutine {
		t.Error("AllowDropRoutine should be true")
	}
	if !opts.AllowDropTrigger {
		t.Error("AllowDropTrigger should be true")
	}
	if !opts.AllowDropSequence {
		t.Error("AllowDropSequence should be true")
	}
}

// ── GenerateMigration with new object types ───────────────────────────────────

// A DiffResult that contains only view/routine/trigger/sequence changes (no table
// changes) should produce a valid, empty-body transaction — not an error.
func TestGenerateMigration_NewObjectTypesOnly_NoError(t *testing.T) {
	diff := schema.DiffResult{
		AddedViews:   []schema.View{{Name: "v_active", Definition: "SELECT 1"}},
		RemovedViews: []schema.View{{Name: "v_old"}},
		ModifiedViews: []schema.ViewDiff{
			{Name: "v_summary", DefinitionDiffers: true},
		},
		AddedRoutines:   []schema.Routine{{Name: "fn_calc", Kind: "FUNCTION", Definition: "BEGIN RETURN 1 END"}},
		RemovedRoutines: []string{"fn_legacy"},
		ModifiedRoutines: []schema.RoutineDiff{
			{Name: "fn_price", DefinitionDiffers: true},
		},
		AddedTriggers:   []schema.Trigger{{Name: "trg_audit", Table: "orders", Timing: "AFTER", Event: "INSERT", Definition: "BEGIN INSERT INTO audit_log(action) VALUES('INSERT'); END"}},
		RemovedTriggers: []string{"trg_old"},
		ModifiedTriggers: []schema.TriggerDiff{
			{Name: "trg_sync", EventDiffers: true},
		},
		AddedSequences:   []schema.Sequence{{Name: "seq_order_id", StartValue: 1, Increment: 1}},
		RemovedSequences: []string{"seq_legacy"},
		ModifiedSequences: []schema.SequenceDiff{
			{Name: "seq_user_id", IncrementDiffers: true},
		},
	}

	for _, driver := range []string{"sqlite", "mysql", "postgres", "mssql"} {
		t.Run(driver, func(t *testing.T) {
			sql, err := schema.GenerateMigration(diff, driver, nil)
			if err != nil {
				t.Fatalf("GenerateMigration(%s) unexpected error: %v", driver, err)
			}
			if !strings.Contains(sql, "BEGIN;") {
				t.Errorf("GenerateMigration(%s): expected BEGIN; in output", driver)
			}
			if !strings.Contains(sql, "COMMIT;") {
				t.Errorf("GenerateMigration(%s): expected COMMIT; in output", driver)
			}
		})
	}
}

// AllowDrop* = true should not cause an error even though the generator has no
// implementation for these types yet.
func TestGenerateMigration_AllowDropNewTypes_NoError(t *testing.T) {
	diff := schema.DiffResult{
		RemovedViews:    []schema.View{{Name: "v_old"}},
		RemovedRoutines: []string{"fn_old"},
		RemovedTriggers: []string{"trg_old"},
		RemovedSequences: []string{"seq_old"},
	}

	opts := &schema.MigrationOptions{
		AllowDropView:     true,
		AllowDropRoutine:  true,
		AllowDropTrigger:  true,
		AllowDropSequence: true,
	}

	_, err := schema.GenerateMigration(diff, "sqlite", opts)
	if err != nil {
		t.Fatalf("GenerateMigration with AllowDrop* = true returned unexpected error: %v", err)
	}
}

// Existing table-level output must be unaffected when new object types are also
// present in the DiffResult.
func TestGenerateMigration_TableOutputUnaffectedByNewObjectTypes(t *testing.T) {
	diff := schema.DiffResult{
		// A plain table addition
		AddedTables: []schema.Table{
			{
				Name: "orders",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "INTEGER", IsNullable: false},
					"total": {Name: "total", DataType: "REAL", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
		// Plus new-type noise that should not affect table output
		AddedViews:      []schema.View{{Name: "v_orders", Definition: "SELECT * FROM orders"}},
		RemovedRoutines: []string{"fn_old"},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error: %v", err)
	}
	if !strings.Contains(sql, "CREATE TABLE") {
		t.Error("expected CREATE TABLE statement for added table")
	}
	if !strings.Contains(sql, "orders") {
		t.Error("expected table name 'orders' in output")
	}
}

// With both table drops and new-type data, AllowDropTable=true still produces
// an uncommented DROP TABLE statement.
func TestGenerateMigration_AllowDropTable_WithNewTypesPresent(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"legacy_tbl"},
		RemovedViews:  []schema.View{{Name: "v_old"}},
	}

	opts := &schema.MigrationOptions{
		AllowDropTable:    true,
		AllowDropView:     true,
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", opts)
	if err != nil {
		t.Fatalf("GenerateMigration() error: %v", err)
	}
	if !containsUncommented(sql, "DROP TABLE") {
		t.Error("expected uncommented DROP TABLE when AllowDropTable=true")
	}
}
