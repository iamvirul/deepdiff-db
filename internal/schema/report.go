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
	if err := os.MkdirAll(outDir, 0o750); err != nil {
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
	if err := os.WriteFile(path, data, 0o600); err != nil {
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

	// Views
	for _, v := range result.AddedViews {
		fmt.Fprintf(&b, "View: %s [added]\n", v.Name)
	}
	for _, v := range result.RemovedViews {
		fmt.Fprintf(&b, "View: %s [removed]\n", v.Name)
	}
	for _, vd := range result.ModifiedViews {
		fmt.Fprintf(&b, "View: %s [modified]\n", vd.Name)
		if vd.DefinitionDiffers {
			fmt.Fprintf(&b, "  - definition differs\n")
		}
		if vd.IsMaterializedDiffers {
			fmt.Fprintf(&b, "  - materialized differs\n")
		}
	}

	// Routines
	for _, r := range result.AddedRoutines {
		fmt.Fprintf(&b, "Routine (%s): %s [added]\n", r.Kind, r.Name)
	}
	for _, name := range result.RemovedRoutines {
		fmt.Fprintf(&b, "Routine: %s [removed]\n", name)
	}
	for _, rd := range result.ModifiedRoutines {
		fmt.Fprintf(&b, "Routine: %s [modified]\n", rd.Name)
		if rd.DefinitionDiffers {
			fmt.Fprintf(&b, "  - definition differs\n")
		}
		if rd.KindDiffers {
			fmt.Fprintf(&b, "  - kind differs: prod=%s dev=%s\n", rd.ProdKind, rd.DevKind)
		}
		if rd.ReturnTypeDiffers {
			fmt.Fprintf(&b, "  - return type differs: prod=%s dev=%s\n", rd.ProdReturnType, rd.DevReturnType)
		}
		if rd.LanguageDiffers {
			fmt.Fprintf(&b, "  - language differs: prod=%s dev=%s\n", rd.ProdLanguage, rd.DevLanguage)
		}
		if rd.ParametersDiffers {
			fmt.Fprintf(&b, "  - parameters differ\n")
		}
	}

	// Triggers
	for _, t := range result.AddedTriggers {
		fmt.Fprintf(&b, "Trigger: %s (table: %s) [added]\n", t.Name, t.Table)
	}
	for _, name := range result.RemovedTriggers {
		fmt.Fprintf(&b, "Trigger: %s [removed]\n", name)
	}
	for _, td := range result.ModifiedTriggers {
		fmt.Fprintf(&b, "Trigger: %s [modified]\n", td.Name)
		if td.TimingDiffers {
			fmt.Fprintf(&b, "  - timing differs: prod=%s dev=%s\n", td.ProdTiming, td.DevTiming)
		}
		if td.EventDiffers {
			fmt.Fprintf(&b, "  - event differs: prod=%s dev=%s\n", td.ProdEvent, td.DevEvent)
		}
		if td.DefinitionDiffers {
			fmt.Fprintf(&b, "  - definition differs\n")
		}
	}

	// Sequences
	for _, seq := range result.AddedSequences {
		fmt.Fprintf(&b, "Sequence: %s [added]\n", seq.Name)
	}
	for _, name := range result.RemovedSequences {
		fmt.Fprintf(&b, "Sequence: %s [removed]\n", name)
	}
	for _, sd := range result.ModifiedSequences {
		fmt.Fprintf(&b, "Sequence: %s [modified]\n", sd.Name)
		if sd.StartValueDiffers {
			fmt.Fprintf(&b, "  - start value differs: prod=%d dev=%d\n", sd.ProdStartValue, sd.DevStartValue)
		}
		if sd.IncrementDiffers {
			fmt.Fprintf(&b, "  - increment differs: prod=%d dev=%d\n", sd.ProdIncrement, sd.DevIncrement)
		}
		if sd.MinValueDiffers {
			fmt.Fprintf(&b, "  - min value differs: prod=%d dev=%d\n", sd.ProdMinValue, sd.DevMinValue)
		}
		if sd.MaxValueDiffers {
			fmt.Fprintf(&b, "  - max value differs: prod=%d dev=%d\n", sd.ProdMaxValue, sd.DevMaxValue)
		}
		if sd.CacheSizeDiffers {
			fmt.Fprintf(&b, "  - cache size differs: prod=%d dev=%d\n", sd.ProdCacheSize, sd.DevCacheSize)
		}
		if sd.CycleDiffers {
			fmt.Fprintf(&b, "  - cycle differs: prod=%v dev=%v\n", boolString(sd.ProdCycle), boolString(sd.DevCycle))
		}
	}

	if b.Len() == 0 {
		b.WriteString("Schema: OK (no differences)\n")
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
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
