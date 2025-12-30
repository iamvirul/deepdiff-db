package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteReports writes JSON and human-readable schema diff reports to outDir.
// It ensures the output directory exists, then writes schema_diff.json (JSON)
// and schema_diff.txt (text). An error is returned if any step fails.
func WriteReports(result DiffResult, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}
	if err := writeJSON(result, filepath.Join(outDir, "schema_diff.json")); err != nil {
		return err
	}
	if err := writeText(result, filepath.Join(outDir, "schema_diff.txt")); err != nil {
		return err
	}
	return nil
}

// writeJSON marshals result to indented JSON and writes it to the given path.
// It returns an error if marshaling or writing fails; the returned error wraps the underlying cause.
func writeJSON(result DiffResult, path string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema diff json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write schema diff json: %w", err)
	}
	return nil
}

// writeText writes a human-readable schema diff for result to the file at path.
// It emits per-table sections for tables that have differences and lists column-level
// changes (missing in prod/dev, type mismatches, and nullable mismatches). If no
// differences are found it writes "Schema: OK (no differences)".
// It returns an error if the report cannot be written to the provided path.
func writeText(result DiffResult, path string) error {
	var b strings.Builder

	for _, td := range result.Tables {
		if !td.HasDifferences {
			continue
		}
		fmt.Fprintf(&b, "Table: %s\n", td.Table)
		if td.OnlyInProd {
			fmt.Fprintf(&b, "  - present in prod, missing in dev\n")
		}
		if td.OnlyInDev {
			fmt.Fprintf(&b, "  - present in dev, missing in prod\n")
		}
		for _, cd := range td.ColumnDiffs {
			if cd.MissingInProd {
				fmt.Fprintf(&b, "  - column %s missing in prod (dev type=%s nullable=%v)\n", cd.Column, cd.DevType, boolString(cd.DevNullable))
			} else if cd.MissingInDev {
				fmt.Fprintf(&b, "  - column %s missing in dev (prod type=%s nullable=%v)\n", cd.Column, cd.ProdType, boolString(cd.ProdNullable))
			} else {
				if cd.TypeMismatch {
					fmt.Fprintf(&b, "  - column %s type mismatch prod=%s dev=%s\n", cd.Column, cd.ProdType, cd.DevType)
				}
				if cd.NullableMismatch {
					fmt.Fprintf(&b, "  - column %s nullable mismatch prod=%v dev=%v\n", cd.Column, boolString(cd.ProdNullable), boolString(cd.DevNullable))
				}
			}
		}
		// Index differences
		for _, idx := range td.AddedIndexes {
			fmt.Fprintf(&b, "  - index %s missing in prod (columns=%v unique=%v)\n", idx.Name, idx.Columns, idx.IsUnique)
		}
		for _, idx := range td.RemovedIndexes {
			fmt.Fprintf(&b, "  - index %s missing in dev (columns=%v unique=%v)\n", idx.Name, idx.Columns, idx.IsUnique)
		}
		for _, idxDiff := range td.ModifiedIndexes {
			if idxDiff.ColumnsDiffer {
				fmt.Fprintf(&b, "  - index %s columns differ prod=%v dev=%v\n", idxDiff.Name, idxDiff.ProdColumns, idxDiff.DevColumns)
			}
			if idxDiff.UniqueDiffers {
				fmt.Fprintf(&b, "  - index %s uniqueness differs prod=%v dev=%v\n", idxDiff.Name, boolString(idxDiff.ProdUnique), boolString(idxDiff.DevUnique))
			}
		}
		b.WriteString("\n")
	}

	if b.Len() == 0 {
		b.WriteString("Schema: OK (no differences)\n")
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write schema diff text: %w", err)
	}
	return nil
}

// boolString returns a human-readable string for a *bool.
// It returns "unknown" when the pointer is nil, "true" when it points to true, and "false" when it points to false.
func boolString(v *bool) string {
	if v == nil {
		return "unknown"
	}
	if *v {
		return "true"
	}
	return "false"
}
