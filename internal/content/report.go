package content

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteReports writes data diff in JSON and appends a summary to summary.txt.
func WriteReports(diff DataDiff, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}
	if err := writeJSON(diff, filepath.Join(outDir, "content_diff.json")); err != nil {
		return err
	}
	if err := writeSummary(diff, filepath.Join(outDir, "summary.txt")); err != nil {
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

func writeSummary(diff DataDiff, path string) error {
	var b strings.Builder
	totalAdded, totalRemoved, totalUpdated := 0, 0, 0

	for _, t := range diff.Tables {
		totalAdded += len(t.Added)
		totalRemoved += len(t.Removed)
		totalUpdated += len(t.Updated)
	}

	fmt.Fprintf(&b, "Data Diff Summary\n")
	fmt.Fprintf(&b, "=================\n")
	fmt.Fprintf(&b, "Added rows   : %d\n", totalAdded)
	fmt.Fprintf(&b, "Removed rows : %d\n", totalRemoved)
	fmt.Fprintf(&b, "Updated rows : %d\n", totalUpdated)

	for _, t := range diff.Tables {
		if len(t.Added)+len(t.Removed)+len(t.Updated) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\nTable: %s\n", t.Table)
		if len(t.Added) > 0 {
			fmt.Fprintf(&b, "  Added keys   (%d): %v\n", len(t.Added), t.Added)
		}
		if len(t.Removed) > 0 {
			fmt.Fprintf(&b, "  Removed keys (%d): %v\n", len(t.Removed), t.Removed)
		}
		if len(t.Updated) > 0 {
			fmt.Fprintf(&b, "  Updated keys (%d): %v\n", len(t.Updated), t.Updated)
		}
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}
