package content

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
//  - content_diff.json: the provided DataDiff serialized as indented JSON,
//  - conflicts.json: the provided Conflicts serialized as indented JSON,
//  - summary.txt: a textual summary that may include schemaStatus, tablesScanned,
//    aggregated counts of added/removed/updated rows, conflicts count, and migrationPack.
//
// It returns an error if directory creation or any file write fails.
func WriteReportsWithInfo(diff DataDiff, conflicts Conflicts, outDir string, schemaStatus string, tablesScanned int, migrationPack string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
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

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}