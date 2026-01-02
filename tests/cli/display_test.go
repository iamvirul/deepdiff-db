package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/cli"
	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
)

func TestDisplayPrintHeader(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintHeader("Test Header")

	result := output.String()
	if !strings.Contains(result, "Test Header") {
		t.Error("header text not found")
	}
	// Should have double lines
	if !strings.Contains(result, "━") {
		t.Error("double line separator not found")
	}
}

func TestDisplayPrintSubHeader(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintSubHeader("Sub Header")

	result := output.String()
	if !strings.Contains(result, "Sub Header") {
		t.Error("sub header text not found")
	}
	// Should have single lines
	if !strings.Contains(result, "─") {
		t.Error("single line separator not found")
	}
}

func TestDisplayPrintProgress(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintProgress(3, 10, "users", "42")

	result := output.String()
	if !strings.Contains(result, "3 of 10") {
		t.Error("progress numbers not found")
	}
	if !strings.Contains(result, "users") {
		t.Error("table name not found")
	}
	if !strings.Contains(result, "42") {
		t.Error("key not found")
	}
}

func TestDisplayPrintConflictComparison(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	prod := &resolve.RowData{
		Columns: []string{"id", "name", "tier"},
		Values:  map[string]any{"id": 1, "name": "Alice", "tier": "silver"},
	}
	dev := &resolve.RowData{
		Columns: []string{"id", "name", "tier"},
		Values:  map[string]any{"id": 1, "name": "Alice", "tier": "gold"},
	}

	diffs := []resolve.ColumnDiff{
		{Column: "id", ProdVal: 1, DevVal: 1, Differs: false},
		{Column: "name", ProdVal: "Alice", DevVal: "Alice", Differs: false},
		{Column: "tier", ProdVal: "silver", DevVal: "gold", Differs: true},
	}

	d.PrintConflictComparison(prod, dev, diffs)

	result := output.String()

	// Check headers
	if !strings.Contains(result, "Column") {
		t.Error("Column header not found")
	}
	if !strings.Contains(result, "Production") {
		t.Error("Production header not found")
	}
	if !strings.Contains(result, "Development") {
		t.Error("Development header not found")
	}

	// Check values
	if !strings.Contains(result, "silver") {
		t.Error("prod tier value not found")
	}
	if !strings.Contains(result, "gold") {
		t.Error("dev tier value not found")
	}

	// Check difference marker
	if !strings.Contains(result, "*") {
		t.Error("difference marker not found")
	}
}

func TestDisplayPrintSummary(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	summary := resolve.ResolutionSummary{
		TotalConflicts:  10,
		ResolvedCount:   7,
		UnresolvedCount: 3,
		ByDecision: map[resolve.Decision]int{
			resolve.DecisionKeepProd: 4,
			resolve.DecisionUseDev:   3,
		},
		ByTable: map[string]int{
			"users":  5,
			"orders": 5,
		},
	}

	d.PrintSummary(summary, "output/resolutions.json")

	result := output.String()

	if !strings.Contains(result, "10") {
		t.Error("total conflicts not found")
	}
	if !strings.Contains(result, "4") {
		t.Error("keep prod count not found")
	}
	if !strings.Contains(result, "3") {
		t.Error("use dev count not found")
	}
	if !strings.Contains(result, "output/resolutions.json") {
		t.Error("output path not found")
	}
}

func TestDisplayPrintSummaryNoPath(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	summary := resolve.ResolutionSummary{
		TotalConflicts: 5,
	}

	d.PrintSummary(summary, "")

	result := output.String()

	// Should not have "saved to" message
	if strings.Contains(result, "saved to") {
		t.Error("should not show 'saved to' when path is empty")
	}
}

func TestDisplayPrintWelcome(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintWelcome(14, 3)

	result := output.String()

	if !strings.Contains(result, "DeepDiff DB") {
		t.Error("app name not found")
	}
	if !strings.Contains(result, "14") {
		t.Error("total conflicts not found")
	}
	if !strings.Contains(result, "3") {
		t.Error("pending count not found")
	}
}

func TestDisplayPrintNoConflicts(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintNoConflicts()

	if !strings.Contains(output.String(), "No conflicts") {
		t.Error("no conflicts message not found")
	}
}

func TestDisplayPrintAllResolved(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintAllResolved()

	if !strings.Contains(output.String(), "resolved") {
		t.Error("all resolved message not found")
	}
}

func TestDisplayPrintError(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintError("something went wrong")

	result := output.String()
	if !strings.Contains(result, "Error") {
		t.Error("Error prefix not found")
	}
	if !strings.Contains(result, "something went wrong") {
		t.Error("error message not found")
	}
}

func TestDisplayPrintWarning(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintWarning("be careful")

	result := output.String()
	if !strings.Contains(result, "Warning") {
		t.Error("Warning prefix not found")
	}
	if !strings.Contains(result, "be careful") {
		t.Error("warning message not found")
	}
}

func TestDisplayPrintSkipped(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintSkipped(5)

	result := output.String()
	if !strings.Contains(result, "5") {
		t.Error("skip count not found")
	}
	if !strings.Contains(result, "pending") {
		t.Error("pending status not mentioned")
	}
}

func TestDisplayPrintSkippedZero(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintSkipped(0)

	// Should not print anything for zero skipped
	if output.Len() > 0 {
		t.Error("should not print anything for zero skipped")
	}
}

func TestDisplayPrintBulkApplied(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	d.PrintBulkApplied("ours", 5, "all remaining conflicts")

	result := output.String()
	if !strings.Contains(result, "ours") {
		t.Error("strategy not found")
	}
	if !strings.Contains(result, "5") {
		t.Error("count not found")
	}
	if !strings.Contains(result, "all remaining") {
		t.Error("scope not found")
	}
}

func TestDisplayPrintTableSummary(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	summary := resolve.ResolutionSummary{
		ByTable: map[string]int{
			"users":    5,
			"products": 3,
		},
	}

	d.PrintTableSummary(summary)

	result := output.String()
	if !strings.Contains(result, "users") {
		t.Error("users table not found")
	}
	if !strings.Contains(result, "products") {
		t.Error("products table not found")
	}
}

func TestDisplayPrintTableSummaryEmpty(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	summary := resolve.ResolutionSummary{
		ByTable: map[string]int{},
	}

	d.PrintTableSummary(summary)

	// Should not print anything for empty table summary
	if output.Len() > 0 {
		t.Error("should not print anything for empty table summary")
	}
}

func TestDisplayComparisonWithNullValues(t *testing.T) {
	output := &bytes.Buffer{}
	d := cli.NewDisplayWithWriter(output)

	prod := &resolve.RowData{
		Columns: []string{"id", "email"},
		Values:  map[string]any{"id": 1, "email": nil},
	}
	dev := &resolve.RowData{
		Columns: []string{"id", "email"},
		Values:  map[string]any{"id": 1, "email": "test@example.com"},
	}

	diffs := []resolve.ColumnDiff{
		{Column: "id", ProdVal: 1, DevVal: 1, Differs: false},
		{Column: "email", ProdVal: nil, DevVal: "test@example.com", Differs: true},
	}

	d.PrintConflictComparison(prod, dev, diffs)

	result := output.String()
	if !strings.Contains(result, "NULL") {
		t.Error("NULL value not displayed")
	}
}
