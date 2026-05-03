package schema_test

// Tests for Phase 4: Sequences — diffSequences logic and writeText output.
// Also covers utility functions containsString and stripMySQLSchemaPrefix
// (exported via public wrappers is not available, so we exercise them
// indirectly through DiffSchemas / WriteReports which call them internally).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ── diffSequences — via DiffSchemas ──────────────────────────────────────────

func makeSeqSchema(seqs map[string]schema.Sequence) *schema.Schema {
	return &schema.Schema{
		Tables:    map[string]schema.Table{},
		Sequences: seqs,
	}
}

func TestDiffSequences_BothEmpty(t *testing.T) {
	result := schema.DiffSchemas(makeSeqSchema(nil), makeSeqSchema(nil))
	if len(result.AddedSequences) != 0 || len(result.RemovedSequences) != 0 || len(result.ModifiedSequences) != 0 {
		t.Errorf("expected no sequence diffs, got added=%d removed=%d modified=%d",
			len(result.AddedSequences), len(result.RemovedSequences), len(result.ModifiedSequences))
	}
}

func TestDiffSequences_Added(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{})
	dev := makeSeqSchema(map[string]schema.Sequence{
		"seq_order_id": {Name: "seq_order_id", StartValue: 1, Increment: 1, MinValue: 1, MaxValue: 9999999},
	})
	result := schema.DiffSchemas(prod, dev)
	if len(result.AddedSequences) != 1 {
		t.Fatalf("expected 1 added sequence, got %d", len(result.AddedSequences))
	}
	if result.AddedSequences[0].Name != "seq_order_id" {
		t.Errorf("expected seq_order_id, got %s", result.AddedSequences[0].Name)
	}
	if len(result.RemovedSequences) != 0 || len(result.ModifiedSequences) != 0 {
		t.Error("expected no removed/modified sequences")
	}
}

func TestDiffSequences_Removed(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{
		"seq_legacy": {Name: "seq_legacy", StartValue: 100, Increment: 10},
	})
	dev := makeSeqSchema(map[string]schema.Sequence{})
	result := schema.DiffSchemas(prod, dev)
	if len(result.RemovedSequences) != 1 {
		t.Fatalf("expected 1 removed sequence, got %d", len(result.RemovedSequences))
	}
	if result.RemovedSequences[0] != "seq_legacy" {
		t.Errorf("expected seq_legacy, got %s", result.RemovedSequences[0])
	}
}

func TestDiffSequences_Identical(t *testing.T) {
	seq := schema.Sequence{Name: "seq_users", StartValue: 1, Increment: 1, MinValue: 1, MaxValue: 2147483647, CacheSize: 1, Cycle: false}
	prod := makeSeqSchema(map[string]schema.Sequence{"seq_users": seq})
	dev := makeSeqSchema(map[string]schema.Sequence{"seq_users": seq})
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedSequences) != 0 {
		t.Errorf("expected no modified sequences for identical sequences, got %d", len(result.ModifiedSequences))
	}
}

func TestDiffSequences_ModifiedStartValue(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{
		"seq_a": {Name: "seq_a", StartValue: 1},
	})
	dev := makeSeqSchema(map[string]schema.Sequence{
		"seq_a": {Name: "seq_a", StartValue: 100},
	})
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedSequences) != 1 {
		t.Fatalf("expected 1 modified sequence, got %d", len(result.ModifiedSequences))
	}
	sd := result.ModifiedSequences[0]
	if !sd.StartValueDiffers {
		t.Error("expected StartValueDiffers=true")
	}
	if sd.ProdStartValue != 1 || sd.DevStartValue != 100 {
		t.Errorf("expected prod=1 dev=100, got prod=%d dev=%d", sd.ProdStartValue, sd.DevStartValue)
	}
}

func TestDiffSequences_ModifiedIncrement(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{"seq_b": {Name: "seq_b", Increment: 1}})
	dev := makeSeqSchema(map[string]schema.Sequence{"seq_b": {Name: "seq_b", Increment: 5}})
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedSequences) != 1 || !result.ModifiedSequences[0].IncrementDiffers {
		t.Error("expected IncrementDiffers=true")
	}
	sd := result.ModifiedSequences[0]
	if sd.ProdIncrement != 1 || sd.DevIncrement != 5 {
		t.Errorf("expected prod=1 dev=5, got prod=%d dev=%d", sd.ProdIncrement, sd.DevIncrement)
	}
}

func TestDiffSequences_ModifiedMinValue(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{"seq_c": {Name: "seq_c", MinValue: 1}})
	dev := makeSeqSchema(map[string]schema.Sequence{"seq_c": {Name: "seq_c", MinValue: 100}})
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedSequences) != 1 || !result.ModifiedSequences[0].MinValueDiffers {
		t.Error("expected MinValueDiffers=true")
	}
}

func TestDiffSequences_ModifiedMaxValue(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{"seq_d": {Name: "seq_d", MaxValue: 9999}})
	dev := makeSeqSchema(map[string]schema.Sequence{"seq_d": {Name: "seq_d", MaxValue: 99999}})
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedSequences) != 1 || !result.ModifiedSequences[0].MaxValueDiffers {
		t.Error("expected MaxValueDiffers=true")
	}
	sd := result.ModifiedSequences[0]
	if sd.ProdMaxValue != 9999 || sd.DevMaxValue != 99999 {
		t.Errorf("expected prod=9999 dev=99999, got prod=%d dev=%d", sd.ProdMaxValue, sd.DevMaxValue)
	}
}

func TestDiffSequences_ModifiedCacheSize(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{"seq_e": {Name: "seq_e", CacheSize: 1}})
	dev := makeSeqSchema(map[string]schema.Sequence{"seq_e": {Name: "seq_e", CacheSize: 20}})
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedSequences) != 1 || !result.ModifiedSequences[0].CacheSizeDiffers {
		t.Error("expected CacheSizeDiffers=true")
	}
	sd := result.ModifiedSequences[0]
	if sd.ProdCacheSize != 1 || sd.DevCacheSize != 20 {
		t.Errorf("expected prod=1 dev=20, got prod=%d dev=%d", sd.ProdCacheSize, sd.DevCacheSize)
	}
}

func TestDiffSequences_ModifiedCycle(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{"seq_f": {Name: "seq_f", Cycle: false}})
	dev := makeSeqSchema(map[string]schema.Sequence{"seq_f": {Name: "seq_f", Cycle: true}})
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedSequences) != 1 || !result.ModifiedSequences[0].CycleDiffers {
		t.Error("expected CycleDiffers=true")
	}
	sd := result.ModifiedSequences[0]
	if sd.ProdCycle == nil || *sd.ProdCycle != false {
		t.Error("expected ProdCycle=false")
	}
	if sd.DevCycle == nil || *sd.DevCycle != true {
		t.Error("expected DevCycle=true")
	}
}

func TestDiffSequences_ModifiedMultipleFields(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{
		"seq_multi": {Name: "seq_multi", StartValue: 1, Increment: 1, MinValue: 1, MaxValue: 1000, CacheSize: 1, Cycle: false},
	})
	dev := makeSeqSchema(map[string]schema.Sequence{
		"seq_multi": {Name: "seq_multi", StartValue: 10, Increment: 5, MinValue: 10, MaxValue: 9999, CacheSize: 50, Cycle: true},
	})
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedSequences) != 1 {
		t.Fatalf("expected 1 modified sequence, got %d", len(result.ModifiedSequences))
	}
	sd := result.ModifiedSequences[0]
	if !sd.StartValueDiffers {
		t.Error("expected StartValueDiffers")
	}
	if !sd.IncrementDiffers {
		t.Error("expected IncrementDiffers")
	}
	if !sd.MinValueDiffers {
		t.Error("expected MinValueDiffers")
	}
	if !sd.MaxValueDiffers {
		t.Error("expected MaxValueDiffers")
	}
	if !sd.CacheSizeDiffers {
		t.Error("expected CacheSizeDiffers")
	}
	if !sd.CycleDiffers {
		t.Error("expected CycleDiffers")
	}
}

func TestDiffSequences_MultipleAddedRemovedModified(t *testing.T) {
	prod := makeSeqSchema(map[string]schema.Sequence{
		"seq_keep":   {Name: "seq_keep", Increment: 1},
		"seq_remove": {Name: "seq_remove", Increment: 1},
		"seq_modify": {Name: "seq_modify", StartValue: 1},
	})
	dev := makeSeqSchema(map[string]schema.Sequence{
		"seq_keep":   {Name: "seq_keep", Increment: 1},
		"seq_add":    {Name: "seq_add", Increment: 10},
		"seq_modify": {Name: "seq_modify", StartValue: 999},
	})
	result := schema.DiffSchemas(prod, dev)
	if len(result.AddedSequences) != 1 || result.AddedSequences[0].Name != "seq_add" {
		t.Errorf("expected 1 added seq_add, got %v", result.AddedSequences)
	}
	if len(result.RemovedSequences) != 1 || result.RemovedSequences[0] != "seq_remove" {
		t.Errorf("expected 1 removed seq_remove, got %v", result.RemovedSequences)
	}
	if len(result.ModifiedSequences) != 1 || result.ModifiedSequences[0].Name != "seq_modify" {
		t.Errorf("expected 1 modified seq_modify, got %v", result.ModifiedSequences)
	}
}

// ── writeText — views/routines/triggers/sequences sections ───────────────────

func TestWriteText_ViewsSection(t *testing.T) {
	tmpDir := t.TempDir()
	result := schema.DiffResult{
		AddedViews: []schema.View{
			{Name: "v_new_report", Definition: "SELECT 1", IsMaterialized: false},
		},
		RemovedViews: []schema.View{
			{Name: "v_legacy_summary", Definition: "SELECT 2"},
		},
		ModifiedViews: []schema.ViewDiff{
			{Name: "v_active_orders", DefinitionDiffers: true},
			{Name: "v_mat_orders", IsMaterializedDiffers: true},
		},
	}
	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports: %v", err)
	}
	txt := readTextReport(t, tmpDir)
	assertContains(t, txt, "View: v_new_report [added]")
	assertContains(t, txt, "View: v_legacy_summary [removed]")
	assertContains(t, txt, "View: v_active_orders [modified]")
	assertContains(t, txt, "definition differs")
	assertContains(t, txt, "View: v_mat_orders [modified]")
	assertContains(t, txt, "materialized differs")
}

func TestWriteText_RoutinesSection(t *testing.T) {
	tmpDir := t.TempDir()
	result := schema.DiffResult{
		AddedRoutines: []schema.Routine{
			{Name: "fn_format_price", Kind: "FUNCTION"},
			{Name: "sp_cleanup", Kind: "PROCEDURE"},
		},
		RemovedRoutines: []string{"fn_old_calc"},
		ModifiedRoutines: []schema.RoutineDiff{
			{Name: "fn_calculate_total", DefinitionDiffers: true},
			{Name: "fn_tier_check", KindDiffers: true, ProdKind: "PROCEDURE", DevKind: "FUNCTION"},
			{Name: "fn_price", ReturnTypeDiffers: true, ProdReturnType: "DECIMAL(10,2)", DevReturnType: "DECIMAL(12,2)"},
			{Name: "fn_lang", LanguageDiffers: true, ProdLanguage: "SQL", DevLanguage: "PLPGSQL"},
			{Name: "fn_params", ParametersDiffers: true},
		},
	}
	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports: %v", err)
	}
	txt := readTextReport(t, tmpDir)
	assertContains(t, txt, "Routine (FUNCTION): fn_format_price [added]")
	assertContains(t, txt, "Routine (PROCEDURE): sp_cleanup [added]")
	assertContains(t, txt, "Routine: fn_old_calc [removed]")
	assertContains(t, txt, "Routine: fn_calculate_total [modified]")
	assertContains(t, txt, "definition differs")
	assertContains(t, txt, "Routine: fn_tier_check [modified]")
	assertContains(t, txt, "kind differs: prod=PROCEDURE dev=FUNCTION")
	assertContains(t, txt, "Routine: fn_price [modified]")
	assertContains(t, txt, "return type differs: prod=DECIMAL(10,2) dev=DECIMAL(12,2)")
	assertContains(t, txt, "Routine: fn_lang [modified]")
	assertContains(t, txt, "language differs: prod=SQL dev=PLPGSQL")
	assertContains(t, txt, "Routine: fn_params [modified]")
	assertContains(t, txt, "parameters differ")
}

func TestWriteText_TriggersSection(t *testing.T) {
	tmpDir := t.TempDir()
	result := schema.DiffResult{
		AddedTriggers: []schema.Trigger{
			{Name: "trg_products_updated_at", Table: "products", Timing: "BEFORE", Event: "UPDATE"},
		},
		RemovedTriggers: []string{"trg_legacy_audit"},
		ModifiedTriggers: []schema.TriggerDiff{
			{Name: "trg_orders_audit", DefinitionDiffers: true},
			{Name: "trg_timing_change", TimingDiffers: true, ProdTiming: "AFTER", DevTiming: "BEFORE"},
			{Name: "trg_event_change", EventDiffers: true, ProdEvent: "INSERT", DevEvent: "UPDATE"},
		},
	}
	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports: %v", err)
	}
	txt := readTextReport(t, tmpDir)
	assertContains(t, txt, "Trigger: trg_products_updated_at (table: products) [added]")
	assertContains(t, txt, "Trigger: trg_legacy_audit [removed]")
	assertContains(t, txt, "Trigger: trg_orders_audit [modified]")
	assertContains(t, txt, "definition differs")
	assertContains(t, txt, "Trigger: trg_timing_change [modified]")
	assertContains(t, txt, "timing differs: prod=AFTER dev=BEFORE")
	assertContains(t, txt, "Trigger: trg_event_change [modified]")
	assertContains(t, txt, "event differs: prod=INSERT dev=UPDATE")
}

func TestWriteText_SequencesSection(t *testing.T) {
	tmpDir := t.TempDir()
	boolTrue := true
	boolFalse := false
	result := schema.DiffResult{
		AddedSequences: []schema.Sequence{
			{Name: "seq_new_orders", StartValue: 1, Increment: 1},
		},
		RemovedSequences: []string{"seq_legacy_ids"},
		ModifiedSequences: []schema.SequenceDiff{
			{
				Name:              "seq_users",
				StartValueDiffers: true, ProdStartValue: 1, DevStartValue: 1000,
				IncrementDiffers: true, ProdIncrement: 1, DevIncrement: 5,
				MinValueDiffers: true, ProdMinValue: 1, DevMinValue: 10,
				MaxValueDiffers: true, ProdMaxValue: 9999, DevMaxValue: 99999,
				CacheSizeDiffers: true, ProdCacheSize: 1, DevCacheSize: 20,
				CycleDiffers: true, ProdCycle: &boolFalse, DevCycle: &boolTrue,
			},
		},
	}
	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports: %v", err)
	}
	txt := readTextReport(t, tmpDir)
	assertContains(t, txt, "Sequence: seq_new_orders [added]")
	assertContains(t, txt, "Sequence: seq_legacy_ids [removed]")
	assertContains(t, txt, "Sequence: seq_users [modified]")
	assertContains(t, txt, "start value differs: prod=1 dev=1000")
	assertContains(t, txt, "increment differs: prod=1 dev=5")
	assertContains(t, txt, "min value differs: prod=1 dev=10")
	assertContains(t, txt, "max value differs: prod=9999 dev=99999")
	assertContains(t, txt, "cache size differs: prod=1 dev=20")
	assertContains(t, txt, "cycle differs:")
}

func TestWriteText_AllObjectTypesNoTableChanges(t *testing.T) {
	// Verify "Schema: OK" is NOT written when there are only view/routine/trigger changes
	tmpDir := t.TempDir()
	result := schema.DiffResult{
		AddedViews: []schema.View{{Name: "v_new", Definition: "SELECT 1"}},
	}
	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports: %v", err)
	}
	txt := readTextReport(t, tmpDir)
	if strings.Contains(txt, "Schema: OK") {
		t.Error("should not write 'Schema: OK' when there are view changes")
	}
	assertContains(t, txt, "View: v_new [added]")
}

func TestWriteText_OKWhenNothingChanged(t *testing.T) {
	tmpDir := t.TempDir()
	result := schema.DiffResult{}
	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports: %v", err)
	}
	txt := readTextReport(t, tmpDir)
	assertContains(t, txt, "Schema: OK (no differences)")
}

// ── helpers ──────────────────────────────────────────────────────────────────

func readTextReport(t *testing.T, dir string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, "schema_diff.txt"))
	if err != nil {
		t.Fatalf("read schema_diff.txt: %v", err)
	}
	return string(content)
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected text to contain %q\ngot:\n%s", needle, haystack)
	}
}
