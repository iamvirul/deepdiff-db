package main

import (
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
)

func TestCompareRowsBothNil(t *testing.T) {
	diffs := resolve.CompareRows(nil, nil)
	if diffs != nil {
		t.Error("expected nil for both nil inputs")
	}
}

func TestCompareRowsOneNil(t *testing.T) {
	prod := &resolve.RowData{
		Columns: []string{"id", "name"},
		Values:  map[string]any{"id": 1, "name": "test"},
	}

	diffs := resolve.CompareRows(prod, nil)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}

	// All should differ since dev is nil
	for _, diff := range diffs {
		if !diff.Differs {
			t.Errorf("column %s should differ when dev is nil", diff.Column)
		}
		if diff.DevVal != nil {
			t.Errorf("dev value should be nil for column %s", diff.Column)
		}
	}
}

func TestCompareRowsIdentical(t *testing.T) {
	prod := &resolve.RowData{
		Columns: []string{"id", "name"},
		Values:  map[string]any{"id": 1, "name": "test"},
	}
	dev := &resolve.RowData{
		Columns: []string{"id", "name"},
		Values:  map[string]any{"id": 1, "name": "test"},
	}

	diffs := resolve.CompareRows(prod, dev)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}

	for _, diff := range diffs {
		if diff.Differs {
			t.Errorf("column %s should not differ for identical rows", diff.Column)
		}
	}
}

func TestCompareRowsDifferent(t *testing.T) {
	prod := &resolve.RowData{
		Columns: []string{"id", "name", "tier"},
		Values:  map[string]any{"id": 1, "name": "Alice", "tier": "silver"},
	}
	dev := &resolve.RowData{
		Columns: []string{"id", "name", "tier"},
		Values:  map[string]any{"id": 1, "name": "Alice", "tier": "gold"},
	}

	diffs := resolve.CompareRows(prod, dev)

	// Find the tier diff
	var tierDiff *resolve.ColumnDiff
	for i := range diffs {
		if diffs[i].Column == "tier" {
			tierDiff = &diffs[i]
			break
		}
	}

	if tierDiff == nil {
		t.Fatal("tier diff not found")
	}
	if !tierDiff.Differs {
		t.Error("tier should differ")
	}
	if tierDiff.ProdVal != "silver" {
		t.Errorf("expected prod tier 'silver', got %v", tierDiff.ProdVal)
	}
	if tierDiff.DevVal != "gold" {
		t.Errorf("expected dev tier 'gold', got %v", tierDiff.DevVal)
	}

	// id and name should not differ
	for _, diff := range diffs {
		if diff.Column == "id" || diff.Column == "name" {
			if diff.Differs {
				t.Errorf("column %s should not differ", diff.Column)
			}
		}
	}
}

func TestCompareRowsDifferentColumns(t *testing.T) {
	// Dev has an extra column
	prod := &resolve.RowData{
		Columns: []string{"id", "name"},
		Values:  map[string]any{"id": 1, "name": "Alice"},
	}
	dev := &resolve.RowData{
		Columns: []string{"id", "name", "email"},
		Values:  map[string]any{"id": 1, "name": "Alice", "email": "alice@example.com"},
	}

	diffs := resolve.CompareRows(prod, dev)

	// Should have 3 columns (union)
	if len(diffs) != 3 {
		t.Fatalf("expected 3 diffs, got %d", len(diffs))
	}

	// Find email diff
	var emailDiff *resolve.ColumnDiff
	for i := range diffs {
		if diffs[i].Column == "email" {
			emailDiff = &diffs[i]
			break
		}
	}

	if emailDiff == nil {
		t.Fatal("email diff not found")
	}
	if !emailDiff.Differs {
		t.Error("email should differ (prod has nil)")
	}
	if emailDiff.ProdVal != nil {
		t.Errorf("expected prod email nil, got %v", emailDiff.ProdVal)
	}
}

func TestFormatValueNil(t *testing.T) {
	result := resolve.FormatValue(nil)
	if result != "NULL" {
		t.Errorf("expected 'NULL' for nil, got %s", result)
	}
}

func TestFormatValueBytes(t *testing.T) {
	result := resolve.FormatValue([]byte("hello"))
	if result != "hello" {
		t.Errorf("expected 'hello', got %s", result)
	}
}

func TestFormatValueBool(t *testing.T) {
	if resolve.FormatValue(true) != "true" {
		t.Error("expected 'true' for true")
	}
	if resolve.FormatValue(false) != "false" {
		t.Error("expected 'false' for false")
	}
}

func TestFormatValueNumber(t *testing.T) {
	if resolve.FormatValue(42) != "42" {
		t.Errorf("expected '42', got %s", resolve.FormatValue(42))
	}
	if resolve.FormatValue(3.14) != "3.14" {
		t.Errorf("expected '3.14', got %s", resolve.FormatValue(3.14))
	}
}

func TestFormatValueString(t *testing.T) {
	result := resolve.FormatValue("test string")
	if result != "test string" {
		t.Errorf("expected 'test string', got %s", result)
	}
}

func TestCompareRowsByteValues(t *testing.T) {
	// Test that byte slices are compared correctly
	prod := &resolve.RowData{
		Columns: []string{"data"},
		Values:  map[string]any{"data": []byte("hello")},
	}
	dev := &resolve.RowData{
		Columns: []string{"data"},
		Values:  map[string]any{"data": []byte("hello")},
	}

	diffs := resolve.CompareRows(prod, dev)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Differs {
		t.Error("identical byte slices should not differ")
	}
}

func TestCompareRowsByteValuesDifferent(t *testing.T) {
	prod := &resolve.RowData{
		Columns: []string{"data"},
		Values:  map[string]any{"data": []byte("hello")},
	}
	dev := &resolve.RowData{
		Columns: []string{"data"},
		Values:  map[string]any{"data": []byte("world")},
	}

	diffs := resolve.CompareRows(prod, dev)
	if !diffs[0].Differs {
		t.Error("different byte slices should differ")
	}
}

func TestCompareRowsMixedTypes(t *testing.T) {
	// Compare byte slice with string
	prod := &resolve.RowData{
		Columns: []string{"data"},
		Values:  map[string]any{"data": []byte("hello")},
	}
	dev := &resolve.RowData{
		Columns: []string{"data"},
		Values:  map[string]any{"data": "hello"},
	}

	diffs := resolve.CompareRows(prod, dev)
	// Should not differ because byte slice "hello" == string "hello"
	if diffs[0].Differs {
		t.Error("byte slice and equivalent string should not differ")
	}
}
