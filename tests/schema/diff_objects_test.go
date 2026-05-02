package schema_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func emptyDiffResult() schema.DiffResult {
	return schema.DiffResult{
		Tables: []schema.TableDiff{},
	}
}

// ── omitempty: new slices must be absent from JSON when empty ─────────────────

func TestDiffResult_OmitemptySlices_AbsentWhenEmpty(t *testing.T) {
	result := emptyDiffResult()

	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	s := string(b)

	for _, key := range []string{
		"added_views", "removed_views", "modified_views",
		"added_routines", "removed_routines", "modified_routines",
		"added_triggers", "removed_triggers", "modified_triggers",
		"added_sequences", "removed_sequences", "modified_sequences",
	} {
		if strings.Contains(s, `"`+key+`"`) {
			t.Errorf("expected %q to be omitted from JSON when nil, but found it in: %s", key, s)
		}
	}
}

// ── omitempty: new slices must appear in JSON when populated ──────────────────

func TestDiffResult_OmitemptySlices_PresentWhenPopulated(t *testing.T) {
	trueBool := true

	result := schema.DiffResult{
		Tables: []schema.TableDiff{},
		AddedViews: []schema.View{
			{Name: "v_active", Definition: "SELECT 1", IsMaterialized: false},
		},
		RemovedViews: []schema.View{{Name: "v_old"}},
		ModifiedViews: []schema.ViewDiff{
			{Name: "v_summary", DefinitionDiffers: true, ProdDefinition: "SELECT 1", DevDefinition: "SELECT 2"},
		},
		AddedRoutines: []schema.Routine{
			{Name: "fn_calc", Kind: "FUNCTION", Definition: "BEGIN RETURN 1; END"},
		},
		RemovedRoutines: []string{"fn_legacy"},
		ModifiedRoutines: []schema.RoutineDiff{
			{Name: "fn_price", DefinitionDiffers: true},
		},
		AddedTriggers: []schema.Trigger{
			{Name: "trg_audit", Table: "orders", Timing: "AFTER", Event: "INSERT", Definition: "BEGIN END"},
		},
		RemovedTriggers: []string{"trg_old"},
		ModifiedTriggers: []schema.TriggerDiff{
			{Name: "trg_sync", EventDiffers: true, ProdEvent: "INSERT", DevEvent: "UPDATE"},
		},
		AddedSequences: []schema.Sequence{
			{Name: "seq_order_id", StartValue: 1, Increment: 1},
		},
		RemovedSequences: []string{"seq_legacy"},
		ModifiedSequences: []schema.SequenceDiff{
			{Name: "seq_user_id", IncrementDiffers: true, ProdIncrement: 1, DevIncrement: 2,
				ProdCycle: &trueBool, DevCycle: boolPtr(false)},
		},
	}

	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	s := string(b)

	for _, key := range []string{
		"added_views", "removed_views", "modified_views",
		"added_routines", "removed_routines", "modified_routines",
		"added_triggers", "removed_triggers", "modified_triggers",
		"added_sequences", "removed_sequences", "modified_sequences",
	} {
		if !strings.Contains(s, `"`+key+`"`) {
			t.Errorf("expected %q to appear in JSON when populated, but it is missing from: %s", key, s)
		}
	}
}

// ── HasDrift with new object types ────────────────────────────────────────────

func TestHasDrift_FalseWhenNoChanges(t *testing.T) {
	result := schema.DiffResult{Tables: []schema.TableDiff{}}
	if result.HasDrift() {
		t.Error("HasDrift() should be false for an empty DiffResult")
	}
}

func TestHasDrift_TrueForAddedView(t *testing.T) {
	result := schema.DiffResult{
		AddedViews: []schema.View{{Name: "v_new", Definition: "SELECT 1"}},
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when AddedViews is non-empty")
	}
}

func TestHasDrift_TrueForRemovedView(t *testing.T) {
	result := schema.DiffResult{RemovedViews: []schema.View{{Name: "v_old"}}}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when RemovedViews is non-empty")
	}
}

func TestHasDrift_TrueForModifiedView(t *testing.T) {
	result := schema.DiffResult{
		ModifiedViews: []schema.ViewDiff{{Name: "v_summary", DefinitionDiffers: true}},
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when ModifiedViews is non-empty")
	}
}

func TestHasDrift_TrueForAddedRoutine(t *testing.T) {
	result := schema.DiffResult{
		AddedRoutines: []schema.Routine{{Name: "fn_new", Kind: "FUNCTION"}},
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when AddedRoutines is non-empty")
	}
}

func TestHasDrift_TrueForRemovedRoutine(t *testing.T) {
	result := schema.DiffResult{RemovedRoutines: []string{"fn_old"}}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when RemovedRoutines is non-empty")
	}
}

func TestHasDrift_TrueForModifiedRoutine(t *testing.T) {
	result := schema.DiffResult{
		ModifiedRoutines: []schema.RoutineDiff{{Name: "fn_price", DefinitionDiffers: true}},
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when ModifiedRoutines is non-empty")
	}
}

func TestHasDrift_TrueForAddedTrigger(t *testing.T) {
	result := schema.DiffResult{
		AddedTriggers: []schema.Trigger{{Name: "trg_new", Table: "orders"}},
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when AddedTriggers is non-empty")
	}
}

func TestHasDrift_TrueForRemovedTrigger(t *testing.T) {
	result := schema.DiffResult{RemovedTriggers: []string{"trg_old"}}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when RemovedTriggers is non-empty")
	}
}

func TestHasDrift_TrueForModifiedTrigger(t *testing.T) {
	result := schema.DiffResult{
		ModifiedTriggers: []schema.TriggerDiff{{Name: "trg_sync", EventDiffers: true}},
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when ModifiedTriggers is non-empty")
	}
}

func TestHasDrift_TrueForAddedSequence(t *testing.T) {
	result := schema.DiffResult{
		AddedSequences: []schema.Sequence{{Name: "seq_new", StartValue: 1, Increment: 1}},
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when AddedSequences is non-empty")
	}
}

func TestHasDrift_TrueForRemovedSequence(t *testing.T) {
	result := schema.DiffResult{RemovedSequences: []string{"seq_old"}}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when RemovedSequences is non-empty")
	}
}

func TestHasDrift_TrueForModifiedSequence(t *testing.T) {
	result := schema.DiffResult{
		ModifiedSequences: []schema.SequenceDiff{{Name: "seq_user_id", IncrementDiffers: true}},
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when ModifiedSequences is non-empty")
	}
}

// ── round-trip: JSON marshal → unmarshal preserves values ────────────────────

func TestDiffResult_JSONRoundTrip_ViewDiff(t *testing.T) {
	original := schema.DiffResult{
		ModifiedViews: []schema.ViewDiff{
			{
				Name:              "v_summary",
				DefinitionDiffers: true,
				ProdDefinition:    "SELECT 1",
				DevDefinition:     "SELECT 2",
				ProdIsMaterialized: boolPtr(false),
				DevIsMaterialized:  boolPtr(true),
			},
		},
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got schema.DiffResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.ModifiedViews) != 1 {
		t.Fatalf("expected 1 ModifiedView, got %d", len(got.ModifiedViews))
	}
	vd := got.ModifiedViews[0]
	if vd.Name != "v_summary" {
		t.Errorf("Name: want %q, got %q", "v_summary", vd.Name)
	}
	if !vd.DefinitionDiffers {
		t.Error("DefinitionDiffers should be true")
	}
	if vd.ProdDefinition != "SELECT 1" {
		t.Errorf("ProdDefinition: want %q, got %q", "SELECT 1", vd.ProdDefinition)
	}
	if vd.DevDefinition != "SELECT 2" {
		t.Errorf("DevDefinition: want %q, got %q", "SELECT 2", vd.DevDefinition)
	}
}

func TestDiffResult_JSONRoundTrip_SequenceDiff(t *testing.T) {
	original := schema.DiffResult{
		ModifiedSequences: []schema.SequenceDiff{
			{
				Name:              "seq_order_id",
				IncrementDiffers:  true,
				ProdIncrement:     1,
				DevIncrement:      5,
				MaxValueDiffers:   true,
				ProdMaxValue:      9223372036854775807,
				DevMaxValue:       1000000,
				CycleDiffers:      true,
				ProdCycle:         boolPtr(false),
				DevCycle:          boolPtr(true),
			},
		},
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got schema.DiffResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.ModifiedSequences) != 1 {
		t.Fatalf("expected 1 ModifiedSequence, got %d", len(got.ModifiedSequences))
	}
	sd := got.ModifiedSequences[0]
	if sd.ProdIncrement != 1 || sd.DevIncrement != 5 {
		t.Errorf("Increment: want prod=1 dev=5, got prod=%d dev=%d", sd.ProdIncrement, sd.DevIncrement)
	}
	if sd.ProdMaxValue != 9223372036854775807 || sd.DevMaxValue != 1000000 {
		t.Errorf("MaxValue round-trip failed: prod=%d dev=%d", sd.ProdMaxValue, sd.DevMaxValue)
	}
	if sd.ProdCycle == nil || *sd.ProdCycle != false {
		t.Error("ProdCycle should be false")
	}
	if sd.DevCycle == nil || *sd.DevCycle != true {
		t.Error("DevCycle should be true")
	}
}
