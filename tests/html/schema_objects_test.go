package html_test

// Coverage tests for buildViewChanges, buildRoutineChanges, buildTriggerChanges,
// buildSequenceChanges, and the variadic add templateFunc in the HTML report generator.

import (
	"strings"
	"testing"

	htmlreport "github.com/iamvirul/deepdiff-db/internal/report/html"
	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func emptyDiff() *schema.DiffResult {
	return &schema.DiffResult{}
}

func buildWith(diff *schema.DiffResult) *htmlreport.ReportData {
	return htmlreport.BuildReportData("prod", "dev", diff, nil, nil, nil, nil, "", "", 0, nil)
}

// ── buildViewChanges ─────────────────────────────────────────────────────────

func TestBuildViewChanges_Added(t *testing.T) {
	diff := &schema.DiffResult{
		AddedViews: []schema.View{
			{Name: "v_new_orders", IsMaterialized: false},
			{Name: "v_mat_view", IsMaterialized: true},
		},
	}
	data := buildWith(diff)
	if !data.HasViewChanges {
		t.Fatal("expected HasViewChanges=true")
	}
	if len(data.ViewChanges) != 2 {
		t.Fatalf("expected 2 view changes, got %d", len(data.ViewChanges))
	}
	for _, vc := range data.ViewChanges {
		if vc.ChangeType != "added" {
			t.Errorf("expected added, got %s", vc.ChangeType)
		}
		if !strings.Contains(vc.Description, vc.Name) {
			t.Errorf("description should contain view name, got %q", vc.Description)
		}
		if !strings.Contains(vc.Description, "dev but not in prod") {
			t.Errorf("description should indicate dev-only, got %q", vc.Description)
		}
	}
	// Verify IsMaterialized is propagated
	for _, vc := range data.ViewChanges {
		if vc.Name == "v_mat_view" && !vc.IsMaterialized {
			t.Error("expected IsMaterialized=true for v_mat_view")
		}
		if vc.Name == "v_new_orders" && vc.IsMaterialized {
			t.Error("expected IsMaterialized=false for v_new_orders")
		}
	}
}

func TestBuildViewChanges_Removed(t *testing.T) {
	diff := &schema.DiffResult{
		RemovedViews: []schema.View{
			{Name: "v_legacy_report", IsMaterialized: false},
		},
	}
	data := buildWith(diff)
	if len(data.ViewChanges) != 1 {
		t.Fatalf("expected 1 view change, got %d", len(data.ViewChanges))
	}
	vc := data.ViewChanges[0]
	if vc.ChangeType != "removed" {
		t.Errorf("expected removed, got %s", vc.ChangeType)
	}
	if !strings.Contains(vc.Description, "prod but not in dev") {
		t.Errorf("description should indicate prod-only, got %q", vc.Description)
	}
}

func TestBuildViewChanges_ModifiedDefinitionOnly(t *testing.T) {
	diff := &schema.DiffResult{
		ModifiedViews: []schema.ViewDiff{
			{Name: "v_active_orders", DefinitionDiffers: true, IsMaterializedDiffers: false},
		},
	}
	data := buildWith(diff)
	if len(data.ViewChanges) != 1 {
		t.Fatalf("expected 1 view change, got %d", len(data.ViewChanges))
	}
	vc := data.ViewChanges[0]
	if vc.ChangeType != "modified" {
		t.Errorf("expected modified, got %s", vc.ChangeType)
	}
	if !strings.Contains(vc.Description, "View definition changed") {
		t.Errorf("expected 'View definition changed' in description, got %q", vc.Description)
	}
	if strings.Contains(vc.Description, "materialization") {
		t.Errorf("should not mention materialization when IsMaterializedDiffers=false, got %q", vc.Description)
	}
}

func TestBuildViewChanges_ModifiedWithMaterializationChange(t *testing.T) {
	diff := &schema.DiffResult{
		ModifiedViews: []schema.ViewDiff{
			{Name: "v_mat_view", DefinitionDiffers: true, IsMaterializedDiffers: true},
		},
	}
	data := buildWith(diff)
	if len(data.ViewChanges) != 1 {
		t.Fatalf("expected 1 view change, got %d", len(data.ViewChanges))
	}
	vc := data.ViewChanges[0]
	if !strings.Contains(vc.Description, "materialization type differs") {
		t.Errorf("expected materialization mention, got %q", vc.Description)
	}
}

func TestBuildViewChanges_SummaryCounts(t *testing.T) {
	diff := &schema.DiffResult{
		AddedViews:    []schema.View{{Name: "v1"}, {Name: "v2"}},
		RemovedViews:  []schema.View{{Name: "v3"}},
		ModifiedViews: []schema.ViewDiff{{Name: "v4", DefinitionDiffers: true}},
	}
	data := buildWith(diff)
	if data.Summary.ViewsChanged != 4 {
		t.Errorf("expected ViewsChanged=4, got %d", data.Summary.ViewsChanged)
	}
}

func TestBuildViewChanges_Empty(t *testing.T) {
	data := buildWith(emptyDiff())
	if data.HasViewChanges {
		t.Error("expected HasViewChanges=false for empty diff")
	}
	if len(data.ViewChanges) != 0 {
		t.Errorf("expected no view changes, got %d", len(data.ViewChanges))
	}
}

// ── buildRoutineChanges ──────────────────────────────────────────────────────

func TestBuildRoutineChanges_Added(t *testing.T) {
	diff := &schema.DiffResult{
		AddedRoutines: []schema.Routine{
			{Name: "fn_format_price", Kind: "FUNCTION"},
			{Name: "sp_cleanup_jobs", Kind: "PROCEDURE"},
		},
	}
	data := buildWith(diff)
	if !data.HasRoutineChanges {
		t.Fatal("expected HasRoutineChanges=true")
	}
	if len(data.RoutineChanges) != 2 {
		t.Fatalf("expected 2 routine changes, got %d", len(data.RoutineChanges))
	}
	for _, rc := range data.RoutineChanges {
		if rc.ChangeType != "added" {
			t.Errorf("expected added, got %s", rc.ChangeType)
		}
		if rc.IsDestructive {
			t.Error("added routine should not be destructive")
		}
	}
	// Verify kind is propagated
	for _, rc := range data.RoutineChanges {
		if rc.Name == "fn_format_price" && rc.Kind != "FUNCTION" {
			t.Errorf("expected FUNCTION kind, got %s", rc.Kind)
		}
		if rc.Name == "sp_cleanup_jobs" && rc.Kind != "PROCEDURE" {
			t.Errorf("expected PROCEDURE kind, got %s", rc.Kind)
		}
	}
}

func TestBuildRoutineChanges_Removed(t *testing.T) {
	diff := &schema.DiffResult{
		RemovedRoutines: []string{"fn_old_calc", "sp_legacy"},
	}
	data := buildWith(diff)
	if len(data.RoutineChanges) != 2 {
		t.Fatalf("expected 2 routine changes, got %d", len(data.RoutineChanges))
	}
	for _, rc := range data.RoutineChanges {
		if rc.ChangeType != "removed" {
			t.Errorf("expected removed, got %s", rc.ChangeType)
		}
		if !rc.IsDestructive {
			t.Error("removed routine should be destructive")
		}
	}
}

func TestBuildRoutineChanges_ModifiedAllFlags(t *testing.T) {
	diff := &schema.DiffResult{
		ModifiedRoutines: []schema.RoutineDiff{
			{
				Name:              "fn_everything",
				DefinitionDiffers: true,
				KindDiffers:       true, ProdKind: "PROCEDURE", DevKind: "FUNCTION",
				ReturnTypeDiffers: true, ProdReturnType: "INT", DevReturnType: "BIGINT",
				LanguageDiffers:   true, ProdLanguage: "SQL", DevLanguage: "PLPGSQL",
				ParametersDiffers: true,
			},
		},
	}
	data := buildWith(diff)
	if len(data.RoutineChanges) != 1 {
		t.Fatalf("expected 1 routine change, got %d", len(data.RoutineChanges))
	}
	rc := data.RoutineChanges[0]
	if rc.ChangeType != "modified" {
		t.Errorf("expected modified, got %s", rc.ChangeType)
	}
	checks := []string{
		"definition changed",
		"kind: PROCEDURE -> FUNCTION",
		"return type: INT -> BIGINT",
		"language: SQL -> PLPGSQL",
		"parameters changed",
	}
	for _, check := range checks {
		if !strings.Contains(rc.Description, check) {
			t.Errorf("expected %q in description, got %q", check, rc.Description)
		}
	}
}

func TestBuildRoutineChanges_ModifiedNoFlags_FallbackDescription(t *testing.T) {
	diff := &schema.DiffResult{
		ModifiedRoutines: []schema.RoutineDiff{
			{Name: "fn_mystery"}, // no flags set
		},
	}
	data := buildWith(diff)
	if len(data.RoutineChanges) != 1 {
		t.Fatalf("expected 1 routine change, got %d", len(data.RoutineChanges))
	}
	rc := data.RoutineChanges[0]
	if rc.Description != "Routine modified" {
		t.Errorf("expected fallback description 'Routine modified', got %q", rc.Description)
	}
}

func TestBuildRoutineChanges_SummaryCounts(t *testing.T) {
	diff := &schema.DiffResult{
		AddedRoutines:    []schema.Routine{{Name: "fn_a"}, {Name: "fn_b"}},
		RemovedRoutines:  []string{"fn_c"},
		ModifiedRoutines: []schema.RoutineDiff{{Name: "fn_d", DefinitionDiffers: true}},
	}
	data := buildWith(diff)
	if data.Summary.RoutinesChanged != 4 {
		t.Errorf("expected RoutinesChanged=4, got %d", data.Summary.RoutinesChanged)
	}
}

// ── buildTriggerChanges ──────────────────────────────────────────────────────

func TestBuildTriggerChanges_Added(t *testing.T) {
	diff := &schema.DiffResult{
		AddedTriggers: []schema.Trigger{
			{Name: "trg_products_updated_at", Table: "products"},
			{Name: "trg_orders_notify", Table: "orders"},
		},
	}
	data := buildWith(diff)
	if !data.HasTriggerChanges {
		t.Fatal("expected HasTriggerChanges=true")
	}
	if len(data.TriggerChanges) != 2 {
		t.Fatalf("expected 2 trigger changes, got %d", len(data.TriggerChanges))
	}
	for _, tc := range data.TriggerChanges {
		if tc.ChangeType != "added" {
			t.Errorf("expected added, got %s", tc.ChangeType)
		}
		if !strings.Contains(tc.Description, tc.Table) {
			t.Errorf("description should contain table name, got %q", tc.Description)
		}
	}
}

func TestBuildTriggerChanges_Removed(t *testing.T) {
	diff := &schema.DiffResult{
		RemovedTriggers: []string{"trg_legacy_log"},
	}
	data := buildWith(diff)
	if len(data.TriggerChanges) != 1 {
		t.Fatalf("expected 1 trigger change, got %d", len(data.TriggerChanges))
	}
	tc := data.TriggerChanges[0]
	if tc.ChangeType != "removed" {
		t.Errorf("expected removed, got %s", tc.ChangeType)
	}
	if !tc.IsDestructive {
		t.Error("removed trigger should be destructive")
	}
}

func TestBuildTriggerChanges_ModifiedAllFlags(t *testing.T) {
	diff := &schema.DiffResult{
		ModifiedTriggers: []schema.TriggerDiff{
			{
				Name:              "trg_orders_audit",
				TimingDiffers:     true, ProdTiming: "AFTER", DevTiming: "BEFORE",
				EventDiffers:      true, ProdEvent: "INSERT", DevEvent: "UPDATE",
				DefinitionDiffers: true,
			},
		},
	}
	data := buildWith(diff)
	if len(data.TriggerChanges) != 1 {
		t.Fatalf("expected 1 trigger change, got %d", len(data.TriggerChanges))
	}
	tc := data.TriggerChanges[0]
	if tc.ChangeType != "modified" {
		t.Errorf("expected modified, got %s", tc.ChangeType)
	}
	checks := []string{
		"timing: AFTER -> BEFORE",
		"event: INSERT -> UPDATE",
		"definition changed",
	}
	for _, check := range checks {
		if !strings.Contains(tc.Description, check) {
			t.Errorf("expected %q in description, got %q", check, tc.Description)
		}
	}
}

func TestBuildTriggerChanges_ModifiedNoFlags_FallbackDescription(t *testing.T) {
	diff := &schema.DiffResult{
		ModifiedTriggers: []schema.TriggerDiff{{Name: "trg_mystery"}},
	}
	data := buildWith(diff)
	if len(data.TriggerChanges) != 1 {
		t.Fatalf("expected 1 trigger change, got %d", len(data.TriggerChanges))
	}
	if data.TriggerChanges[0].Description != "Trigger modified" {
		t.Errorf("expected fallback 'Trigger modified', got %q", data.TriggerChanges[0].Description)
	}
}

func TestBuildTriggerChanges_SummaryCounts(t *testing.T) {
	diff := &schema.DiffResult{
		AddedTriggers:    []schema.Trigger{{Name: "t1", Table: "a"}, {Name: "t2", Table: "b"}},
		RemovedTriggers:  []string{"t3"},
		ModifiedTriggers: []schema.TriggerDiff{{Name: "t4", DefinitionDiffers: true}},
	}
	data := buildWith(diff)
	if data.Summary.TriggersChanged != 4 {
		t.Errorf("expected TriggersChanged=4, got %d", data.Summary.TriggersChanged)
	}
}

// ── buildSequenceChanges ─────────────────────────────────────────────────────

func TestBuildSequenceChanges_Added(t *testing.T) {
	diff := &schema.DiffResult{
		AddedSequences: []schema.Sequence{
			{Name: "seq_order_id"},
			{Name: "seq_invoice_num"},
		},
	}
	data := buildWith(diff)
	if !data.HasSequenceChanges {
		t.Fatal("expected HasSequenceChanges=true")
	}
	if len(data.SequenceChanges) != 2 {
		t.Fatalf("expected 2 sequence changes, got %d", len(data.SequenceChanges))
	}
	for _, sc := range data.SequenceChanges {
		if sc.ChangeType != "added" {
			t.Errorf("expected added, got %s", sc.ChangeType)
		}
	}
}

func TestBuildSequenceChanges_Removed(t *testing.T) {
	diff := &schema.DiffResult{
		RemovedSequences: []string{"seq_legacy"},
	}
	data := buildWith(diff)
	if len(data.SequenceChanges) != 1 {
		t.Fatalf("expected 1 sequence change, got %d", len(data.SequenceChanges))
	}
	sc := data.SequenceChanges[0]
	if sc.ChangeType != "removed" {
		t.Errorf("expected removed, got %s", sc.ChangeType)
	}
	if !strings.Contains(sc.Description, "prod but not in dev") {
		t.Errorf("description should indicate prod-only, got %q", sc.Description)
	}
}

func TestBuildSequenceChanges_ModifiedAllNumericFields(t *testing.T) {
	diff := &schema.DiffResult{
		ModifiedSequences: []schema.SequenceDiff{
			{
				Name:              "seq_users",
				StartValueDiffers: true, ProdStartValue: 1, DevStartValue: 1000,
				IncrementDiffers: true, ProdIncrement: 1, DevIncrement: 10,
				MinValueDiffers:  true, ProdMinValue: 1, DevMinValue: 100,
				MaxValueDiffers:  true, ProdMaxValue: 9999, DevMaxValue: 99999,
				CacheSizeDiffers: true, ProdCacheSize: 1, DevCacheSize: 50,
			},
		},
	}
	data := buildWith(diff)
	if len(data.SequenceChanges) != 1 {
		t.Fatalf("expected 1 sequence change, got %d", len(data.SequenceChanges))
	}
	sc := data.SequenceChanges[0]
	if sc.ChangeType != "modified" {
		t.Errorf("expected modified, got %s", sc.ChangeType)
	}
	checks := []string{
		"start value: 1 -> 1000",
		"increment: 1 -> 10",
		"min value: 1 -> 100",
		"max value: 9999 -> 99999",
		"cache size: 1 -> 50",
	}
	for _, check := range checks {
		if !strings.Contains(sc.Description, check) {
			t.Errorf("expected %q in description, got %q", check, sc.Description)
		}
	}
}

func TestBuildSequenceChanges_ModifiedCycle_TrueToFalse(t *testing.T) {
	boolTrue := true
	boolFalse := false
	diff := &schema.DiffResult{
		ModifiedSequences: []schema.SequenceDiff{
			{Name: "seq_cycled", CycleDiffers: true, ProdCycle: &boolTrue, DevCycle: &boolFalse},
		},
	}
	data := buildWith(diff)
	sc := data.SequenceChanges[0]
	if !strings.Contains(sc.Description, "cycle: true -> false") {
		t.Errorf("expected 'cycle: true -> false', got %q", sc.Description)
	}
}

func TestBuildSequenceChanges_ModifiedCycle_NilPointers(t *testing.T) {
	diff := &schema.DiffResult{
		ModifiedSequences: []schema.SequenceDiff{
			{Name: "seq_nil_cycle", CycleDiffers: true, ProdCycle: nil, DevCycle: nil},
		},
	}
	data := buildWith(diff)
	sc := data.SequenceChanges[0]
	// Both nil → both render as "false"
	if !strings.Contains(sc.Description, "cycle: false -> false") {
		t.Errorf("expected 'cycle: false -> false' for nil pointers, got %q", sc.Description)
	}
}

func TestBuildSequenceChanges_ModifiedNoFlags_FallbackDescription(t *testing.T) {
	diff := &schema.DiffResult{
		ModifiedSequences: []schema.SequenceDiff{{Name: "seq_mystery"}},
	}
	data := buildWith(diff)
	if len(data.SequenceChanges) != 1 {
		t.Fatalf("expected 1 sequence change, got %d", len(data.SequenceChanges))
	}
	if data.SequenceChanges[0].Description != "Sequence modified" {
		t.Errorf("expected fallback 'Sequence modified', got %q", data.SequenceChanges[0].Description)
	}
}

func TestBuildSequenceChanges_SummaryCounts(t *testing.T) {
	diff := &schema.DiffResult{
		AddedSequences:    []schema.Sequence{{Name: "s1"}, {Name: "s2"}},
		RemovedSequences:  []string{"s3", "s4"},
		ModifiedSequences: []schema.SequenceDiff{{Name: "s5", StartValueDiffers: true}},
	}
	data := buildWith(diff)
	if data.Summary.SequencesChanged != 5 {
		t.Errorf("expected SequencesChanged=5, got %d", data.Summary.SequencesChanged)
	}
}

// ── templateFuncs variadic add ───────────────────────────────────────────────
// The schema tab badge calls add with 5 arguments: len(SchemaChanges) +
// ViewsChanged + RoutinesChanged + TriggersChanged + SequencesChanged.
// We verify this via a full GenerateReport call with all change types populated.

func TestGenerateReport_VariadicAddInSchemaBadge(t *testing.T) {
	diff := &schema.DiffResult{
		AddedTables: []schema.Table{{Name: "new_tbl"}},
		AddedViews:  []schema.View{{Name: "v_new"}},
		AddedRoutines: []schema.Routine{{Name: "fn_new", Kind: "FUNCTION"}},
		AddedTriggers: []schema.Trigger{{Name: "trg_new", Table: "new_tbl"}},
		AddedSequences: []schema.Sequence{{Name: "seq_new"}},
	}
	data := htmlreport.BuildReportData("prod", "dev", diff, nil, nil, nil, nil, "", "", 1, nil)

	g := htmlreport.NewGenerator(nil)
	outPath := t.TempDir() + "/report.html"
	if err := g.GenerateReport(data, outPath); err != nil {
		t.Fatalf("GenerateReport failed with variadic add: %v", err)
	}
}

// ── all object types together in one report ──────────────────────────────────

func TestBuildReportData_AllSchemaObjectTypes(t *testing.T) {
	boolFalse := false
	boolTrue := true
	diff := &schema.DiffResult{
		// Tables
		AddedTables:   []schema.Table{{Name: "feature_flags"}},
		RemovedTables: []string{"legacy_settings"},
		Tables: []schema.TableDiff{
			{
				Name:           "customers",
				HasDifferences: true,
				AddedColumns:   []schema.Column{{Name: "tier", DataType: "VARCHAR(20)"}},
			},
		},
		// Views
		AddedViews:   []schema.View{{Name: "v_customer_stats"}},
		RemovedViews: []schema.View{{Name: "v_customer_summary"}},
		ModifiedViews: []schema.ViewDiff{
			{Name: "v_active_orders", DefinitionDiffers: true, IsMaterializedDiffers: true},
		},
		// Routines
		AddedRoutines:   []schema.Routine{{Name: "fn_format_price", Kind: "FUNCTION"}},
		RemovedRoutines: []string{"fn_old_calc"},
		ModifiedRoutines: []schema.RoutineDiff{
			{Name: "fn_get_tier", DefinitionDiffers: true, ReturnTypeDiffers: true,
				ProdReturnType: "VARCHAR(10)", DevReturnType: "VARCHAR(20)"},
		},
		// Triggers
		AddedTriggers:   []schema.Trigger{{Name: "trg_updated_at", Table: "products"}},
		RemovedTriggers: []string{"trg_legacy"},
		ModifiedTriggers: []schema.TriggerDiff{
			{Name: "trg_audit", TimingDiffers: true, ProdTiming: "BEFORE", DevTiming: "AFTER"},
		},
		// Sequences
		AddedSequences:   []schema.Sequence{{Name: "seq_orders"}},
		RemovedSequences: []string{"seq_old"},
		ModifiedSequences: []schema.SequenceDiff{
			{Name: "seq_users", CycleDiffers: true, ProdCycle: &boolFalse, DevCycle: &boolTrue},
		},
	}

	data := buildWith(diff)

	if !data.HasViewChanges {
		t.Error("expected HasViewChanges=true")
	}
	if !data.HasRoutineChanges {
		t.Error("expected HasRoutineChanges=true")
	}
	if !data.HasTriggerChanges {
		t.Error("expected HasTriggerChanges=true")
	}
	if !data.HasSequenceChanges {
		t.Error("expected HasSequenceChanges=true")
	}
	if !data.HasSchemaDiff {
		t.Error("expected HasSchemaDiff=true")
	}

	// Verify summary counts
	if data.Summary.ViewsChanged != 3 {
		t.Errorf("expected ViewsChanged=3, got %d", data.Summary.ViewsChanged)
	}
	if data.Summary.RoutinesChanged != 3 {
		t.Errorf("expected RoutinesChanged=3, got %d", data.Summary.RoutinesChanged)
	}
	if data.Summary.TriggersChanged != 3 {
		t.Errorf("expected TriggersChanged=3, got %d", data.Summary.TriggersChanged)
	}
	if data.Summary.SequencesChanged != 3 {
		t.Errorf("expected SequencesChanged=3, got %d", data.Summary.SequencesChanged)
	}

	// Verify full report renders without error
	g := htmlreport.NewGenerator(nil)
	outPath := t.TempDir() + "/full_report.html"
	if err := g.GenerateReport(data, outPath); err != nil {
		t.Fatalf("GenerateReport with all object types failed: %v", err)
	}
}
