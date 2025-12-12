package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteReports writes both JSON and text summaries to the output directory.
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

func boolString(v *bool) string {
	if v == nil {
		return "unknown"
	}
	if *v {
		return "true"
	}
	return "false"
}
