package content

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteReports writes the given data diff and conflicts into outDir as JSON files and produces a summary.txt using an empty schema status, zero tables scanned, and no migration pack.
// It creates content_diff.json, conflicts.json, and summary.txt in outDir and returns any error encountered.
func WriteReports(diff DataDiff, conflicts Conflicts, outDir string) error {
	return WriteReportsWithInfo(diff, conflicts, outDir, "", 0, "")
}

// WriteReportsWithInfo creates the output directory (if needed) and writes the data diff,
// conflicts, and a human-readable summary to the specified output directory.
//
// The function writes three files into outDir:
//   - content_diff.json: the provided DataDiff serialized as indented JSON,
//   - conflicts.json: the provided Conflicts serialized as indented JSON,
//   - summary.txt: a textual summary that may include schemaStatus, tablesScanned,
//     aggregated counts of added/removed/updated rows, conflicts count, and migrationPack.
//
// It returns an error if directory creation or any file write fails.
func WriteReportsWithInfo(diff DataDiff, conflicts Conflicts, outDir string, schemaStatus string, tablesScanned int, migrationPack string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}
	if err := writeJSON(diff, filepath.Join(outDir, "content_diff.json")); err != nil {
		return err
	}
	if err := writeConflictsJSON(conflicts, filepath.Join(outDir, "conflicts.json")); err != nil {
		return err
	}
	if err := writeSummary(diff, conflicts, filepath.Join(outDir, "summary.txt"), schemaStatus, tablesScanned, migrationPack); err != nil {
		return err
	}
	return nil
}

// writeJSON writes the given DataDiff as pretty-printed JSON to the provided file path.
// It returns an error if marshaling the data or writing the file fails.
func writeJSON(diff DataDiff, path string) error {
	data, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal content diff: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write content diff json: %w", err)
	}
	return nil
}

// writeConflictsJSON marshals the provided conflicts to indented JSON and writes the result to path.
// It returns an error that wraps any failure to marshal the conflicts or to write the file.
func writeConflictsJSON(conflicts Conflicts, path string) error {
	data, err := json.MarshalIndent(conflicts, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal conflicts: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write conflicts json: %w", err)
	}
	return nil
}

// writeSummary writes a textual summary of the provided DataDiff and Conflicts to the file at path.
// The summary includes optional schema status, number of tables scanned, totals of added/removed/updated
// rows across all tables, the count of conflicts, and the base name of the migrationPack if provided.
// It returns an error if writing the summary file fails.
func writeSummary(diff DataDiff, conflicts Conflicts, path string, schemaStatus string, tablesScanned int, migrationPack string) error {
	var b strings.Builder
	totalAdded, totalRemoved, totalUpdated := 0, 0, 0

	for _, t := range diff.Tables {
		totalAdded += len(t.Added)
		totalRemoved += len(t.Removed)
		totalUpdated += len(t.Updated)
	}

	// Match README format
	if schemaStatus != "" {
		fmt.Fprintf(&b, "Schema: %s\n", schemaStatus)
	}
	if tablesScanned > 0 {
		fmt.Fprintf(&b, "Tables scanned: %d\n", tablesScanned)
	}
	if totalAdded > 0 {
		fmt.Fprintf(&b, "Added rows: %d\n", totalAdded)
	}
	if totalRemoved > 0 {
		fmt.Fprintf(&b, "Removed rows: %d\n", totalRemoved)
	}
	if totalUpdated > 0 {
		fmt.Fprintf(&b, "Updated rows: %d\n", totalUpdated)
	}
	if len(conflicts.Conflicts) > 0 {
		fmt.Fprintf(&b, "Conflicts: %d\n", len(conflicts.Conflicts))
	}
	if migrationPack != "" {
		fmt.Fprintf(&b, "Migration pack: %s\n", filepath.Base(migrationPack))
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}

// ResolutionInfo contains resolution information to include in reports.
type ResolutionInfo struct {
	TotalConflicts  int            `json:"total_conflicts"`
	ResolvedCount   int            `json:"resolved_count"`
	UnresolvedCount int            `json:"unresolved_count"`
	ByDecision      map[string]int `json:"by_decision"`
	ByTable         map[string]int `json:"by_table"`
}

// WriteReportsWithResolutions writes reports that include resolution information.
// This extends WriteReportsWithInfo by adding resolution statistics to the summary
// and generating an optional resolutions_summary.json file.
func WriteReportsWithResolutions(diff DataDiff, conflicts Conflicts, outDir string,
	schemaStatus string, tablesScanned int, migrationPack string, resInfo *ResolutionInfo) error {

	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}
	if err := writeJSON(diff, filepath.Join(outDir, "content_diff.json")); err != nil {
		return err
	}
	if err := writeConflictsJSON(conflicts, filepath.Join(outDir, "conflicts.json")); err != nil {
		return err
	}
	if err := writeSummaryWithResolutions(diff, conflicts, filepath.Join(outDir, "summary.txt"),
		schemaStatus, tablesScanned, migrationPack, resInfo); err != nil {
		return err
	}

	// Write resolutions_summary.json if resolution info is provided
	if resInfo != nil && resInfo.TotalConflicts > 0 {
		if err := writeResolutionSummaryJSON(resInfo, filepath.Join(outDir, "resolutions_summary.json")); err != nil {
			return err
		}
	}

	return nil
}

// writeSummaryWithResolutions writes a summary that includes resolution statistics.
func writeSummaryWithResolutions(diff DataDiff, conflicts Conflicts, path string,
	schemaStatus string, tablesScanned int, migrationPack string, resInfo *ResolutionInfo) error {

	var b strings.Builder
	totalAdded, totalRemoved, totalUpdated := 0, 0, 0

	for _, t := range diff.Tables {
		totalAdded += len(t.Added)
		totalRemoved += len(t.Removed)
		totalUpdated += len(t.Updated)
	}

	// Match README format
	if schemaStatus != "" {
		fmt.Fprintf(&b, "Schema: %s\n", schemaStatus)
	}
	if tablesScanned > 0 {
		fmt.Fprintf(&b, "Tables scanned: %d\n", tablesScanned)
	}
	if totalAdded > 0 {
		fmt.Fprintf(&b, "Added rows: %d\n", totalAdded)
	}
	if totalRemoved > 0 {
		fmt.Fprintf(&b, "Removed rows: %d\n", totalRemoved)
	}
	if totalUpdated > 0 {
		fmt.Fprintf(&b, "Updated rows: %d\n", totalUpdated)
	}

	// Conflict and resolution information
	if len(conflicts.Conflicts) > 0 {
		fmt.Fprintf(&b, "Conflicts: %d\n", len(conflicts.Conflicts))
	}

	// Add resolution statistics if available
	if resInfo != nil && resInfo.TotalConflicts > 0 {
		fmt.Fprintf(&b, "\nResolution Summary:\n")
		fmt.Fprintf(&b, "  Auto-resolved: %d\n", resInfo.ResolvedCount)
		fmt.Fprintf(&b, "  Pending review: %d\n", resInfo.UnresolvedCount)

		// Show breakdown by decision
		if len(resInfo.ByDecision) > 0 {
			fmt.Fprintf(&b, "\nBy Decision:\n")
			// Sort decisions for consistent output
			decisions := make([]string, 0, len(resInfo.ByDecision))
			for d := range resInfo.ByDecision {
				decisions = append(decisions, d)
			}
			sort.Strings(decisions)
			for _, d := range decisions {
				fmt.Fprintf(&b, "  %s: %d\n", formatDecision(d), resInfo.ByDecision[d])
			}
		}

		// Show breakdown by table
		if len(resInfo.ByTable) > 0 {
			fmt.Fprintf(&b, "\nBy Table:\n")
			// Sort tables for consistent output
			tables := make([]string, 0, len(resInfo.ByTable))
			for t := range resInfo.ByTable {
				tables = append(tables, t)
			}
			sort.Strings(tables)
			for _, t := range tables {
				fmt.Fprintf(&b, "  %s: %d\n", t, resInfo.ByTable[t])
			}
		}
	}

	if migrationPack != "" {
		fmt.Fprintf(&b, "\nMigration pack: %s\n", filepath.Base(migrationPack))
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}

// formatDecision formats a decision string for display.
func formatDecision(decision string) string {
	switch decision {
	case "keep_prod":
		return "Keep production (ours)"
	case "use_dev":
		return "Use development (theirs)"
	case "pending":
		return "Pending manual review"
	default:
		return decision
	}
}

// writeResolutionSummaryJSON writes detailed resolution statistics to a JSON file.
func writeResolutionSummaryJSON(resInfo *ResolutionInfo, path string) error {
	data, err := json.MarshalIndent(resInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal resolution summary: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write resolution summary json: %w", err)
	}
	return nil
}

// BuildResolutionInfo creates a ResolutionInfo from resolution counts.
// This is a helper to convert internal resolution data to the report format.
func BuildResolutionInfo(totalConflicts, resolvedCount, unresolvedCount int,
	byDecision map[string]int, byTable map[string]int) *ResolutionInfo {
	return &ResolutionInfo{
		TotalConflicts:  totalConflicts,
		ResolvedCount:   resolvedCount,
		UnresolvedCount: unresolvedCount,
		ByDecision:      byDecision,
		ByTable:         byTable,
	}
}
