package content

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteReports writes data diff in JSON and appends a summary to summary.txt.
func WriteReports(diff DataDiff, conflicts Conflicts, outDir string) error {
	return WriteReportsWithInfo(diff, conflicts, outDir, "", 0, "")
}

// WriteReportsWithInfo writes data diff in JSON and creates a summary.txt with schema info.
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
