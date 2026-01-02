// Package cli provides interactive CLI utilities for user input and display.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
)

// Display handles formatted output for conflict resolution.
type Display struct {
	writer io.Writer
}

// NewDisplay creates a new Display using stdout.
func NewDisplay() *Display {
	return &Display{writer: os.Stdout}
}

// NewDisplayWithWriter creates a Display with custom output.
func NewDisplayWithWriter(w io.Writer) *Display {
	return &Display{writer: w}
}

const (
	lineWidth  = 70
	lineSingle = "─"
	lineDouble = "━"
)

// PrintHeader prints a section header.
func (d *Display) PrintHeader(title string) {
	line := strings.Repeat(lineDouble, lineWidth)
	fmt.Fprintln(d.writer, line)
	fmt.Fprintln(d.writer, title)
	fmt.Fprintln(d.writer, line)
}

// PrintSubHeader prints a sub-section header.
func (d *Display) PrintSubHeader(title string) {
	line := strings.Repeat(lineSingle, lineWidth)
	fmt.Fprintln(d.writer, line)
	fmt.Fprintln(d.writer, title)
	fmt.Fprintln(d.writer, line)
}

// PrintLine prints a separator line.
func (d *Display) PrintLine() {
	fmt.Fprintln(d.writer, strings.Repeat(lineSingle, lineWidth))
}

// PrintDoubleLine prints a double separator line.
func (d *Display) PrintDoubleLine() {
	fmt.Fprintln(d.writer, strings.Repeat(lineDouble, lineWidth))
}

// PrintProgress displays the current progress.
func (d *Display) PrintProgress(current, total int, table, key string) {
	d.PrintDoubleLine()
	fmt.Fprintf(d.writer, "Conflict %d of %d | Table: %s | Key: %s\n", current, total, table, key)
	d.PrintDoubleLine()
}

// PrintConflictComparison displays a side-by-side comparison of prod and dev values.
func (d *Display) PrintConflictComparison(prod, dev *resolve.RowData, diffs []resolve.ColumnDiff) {
	fmt.Fprintln(d.writer)

	// Calculate column widths
	colWidth := 20
	valWidth := 22

	// Print header
	fmt.Fprintf(d.writer, "  %-*s | %-*s | %-*s\n",
		colWidth, "Column",
		valWidth, "Production",
		valWidth, "Development")
	fmt.Fprintf(d.writer, "  %s+%s+%s\n",
		strings.Repeat("-", colWidth+1),
		strings.Repeat("-", valWidth+2),
		strings.Repeat("-", valWidth+2))

	// Print each column
	for _, diff := range diffs {
		prodStr := formatValueForDisplay(diff.ProdVal, valWidth)
		devStr := formatValueForDisplay(diff.DevVal, valWidth)

		marker := ""
		if diff.Differs {
			marker = " *"
		}

		fmt.Fprintf(d.writer, "  %-*s | %-*s | %-*s%s\n",
			colWidth, truncate(diff.Column, colWidth),
			valWidth, prodStr,
			valWidth, devStr,
			marker)
	}

	fmt.Fprintln(d.writer)
	if hasDifferences(diffs) {
		fmt.Fprintln(d.writer, "  * indicates columns that differ")
	}
	d.PrintDoubleLine()
	fmt.Fprintln(d.writer)
}

// PrintSummary displays the resolution summary.
func (d *Display) PrintSummary(summary resolve.ResolutionSummary, outputPath string) {
	d.PrintDoubleLine()
	fmt.Fprintln(d.writer, "Resolution Summary")
	d.PrintDoubleLine()

	fmt.Fprintf(d.writer, "  %-25s %d\n", "Total conflicts:", summary.TotalConflicts)
	fmt.Fprintf(d.writer, "  %-25s %d\n", "Resolved (keep prod):", summary.ByDecision[resolve.DecisionKeepProd])
	fmt.Fprintf(d.writer, "  %-25s %d\n", "Resolved (use dev):", summary.ByDecision[resolve.DecisionUseDev])
	fmt.Fprintf(d.writer, "  %-25s %d\n", "Pending (manual):", summary.UnresolvedCount)

	d.PrintDoubleLine()
	if outputPath != "" {
		fmt.Fprintf(d.writer, "  Resolutions saved to: %s\n", outputPath)
		d.PrintDoubleLine()
	}
	fmt.Fprintln(d.writer)
}

// PrintTableSummary displays a summary grouped by table.
func (d *Display) PrintTableSummary(summary resolve.ResolutionSummary) {
	if len(summary.ByTable) == 0 {
		return
	}

	fmt.Fprintln(d.writer, "\n  By Table:")
	for table, count := range summary.ByTable {
		fmt.Fprintf(d.writer, "    %-30s %d conflicts\n", table, count)
	}
}

// PrintNoConflicts displays a message when there are no conflicts.
func (d *Display) PrintNoConflicts() {
	fmt.Fprintln(d.writer, "No conflicts to resolve.")
}

// PrintAllResolved displays a message when all conflicts are resolved.
func (d *Display) PrintAllResolved() {
	fmt.Fprintln(d.writer, "All conflicts have been resolved.")
}

// PrintError displays an error message.
func (d *Display) PrintError(msg string) {
	fmt.Fprintf(d.writer, "Error: %s\n", msg)
}

// PrintWarning displays a warning message.
func (d *Display) PrintWarning(msg string) {
	fmt.Fprintf(d.writer, "Warning: %s\n", msg)
}

// PrintInfo displays an info message.
func (d *Display) PrintInfo(msg string) {
	fmt.Fprintln(d.writer, msg)
}

// PrintSaving displays a saving message.
func (d *Display) PrintSaving(path string) {
	fmt.Fprintf(d.writer, "Saving resolutions to %s...\n", path)
}

// PrintSaved displays a saved confirmation.
func (d *Display) PrintSaved(path string) {
	fmt.Fprintf(d.writer, "Resolutions saved to %s\n", path)
}

// PrintSkipped displays the number of skipped conflicts.
func (d *Display) PrintSkipped(count int) {
	if count > 0 {
		fmt.Fprintf(d.writer, "Skipped %d conflict(s) (left pending)\n", count)
	}
}

// PrintBulkApplied displays a message about bulk resolution.
func (d *Display) PrintBulkApplied(strategy string, count int, scope string) {
	fmt.Fprintf(d.writer, "Applied \"%s\" to %d conflict(s) in %s\n", strategy, count, scope)
}

// PrintWelcome displays the welcome message.
func (d *Display) PrintWelcome(totalConflicts, pendingConflicts int) {
	d.PrintDoubleLine()
	fmt.Fprintln(d.writer, "DeepDiff DB - Interactive Conflict Resolution")
	d.PrintDoubleLine()
	fmt.Fprintf(d.writer, "  Total conflicts: %d\n", totalConflicts)
	fmt.Fprintf(d.writer, "  Pending review:  %d\n", pendingConflicts)
	d.PrintLine()
	fmt.Fprintln(d.writer)
}

// formatValueForDisplay formats a value for display, truncating if necessary.
func formatValueForDisplay(v any, maxWidth int) string {
	str := resolve.FormatValue(v)
	return truncate(str, maxWidth)
}

// truncate truncates a string to maxWidth, adding "..." if truncated.
func truncate(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	return s[:maxWidth-3] + "..."
}

// hasDifferences checks if any columns differ.
func hasDifferences(diffs []resolve.ColumnDiff) bool {
	for _, diff := range diffs {
		if diff.Differs {
			return true
		}
	}
	return false
}
