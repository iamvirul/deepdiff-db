package resolve

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// RowDiffStatus represents the state of a row difference.
type RowDiffStatus string

const (
	// StatusUpdated indicates the row exists in both prod and dev with differing values.
	StatusUpdated RowDiffStatus = "updated"
	// StatusAdded indicates the row exists only in development.
	StatusAdded RowDiffStatus = "added"
	// StatusRemoved indicates the row exists only in production.
	StatusRemoved RowDiffStatus = "removed"
)

// RowDiff represents the column-level differences for a single row.
type RowDiff struct {
	Table       string        `json:"table"`
	Key         string        `json:"key"`
	Status      RowDiffStatus `json:"status"`
	Columns     []ColumnDiff  `json:"columns"`
	DiffColumns []string      `json:"differing_columns,omitempty"`
}

// TableRowDiff represents all row differences for a specific table.
type TableRowDiff struct {
	Table string    `json:"table"`
	Rows  []RowDiff `json:"rows"`
}

// RowDiffReport contains all row differences grouped by table.
type RowDiffReport struct {
	Tables []TableRowDiff `json:"tables"`
}

// HasChanges reports whether any row differences exist in the report.
func (r RowDiffReport) HasChanges() bool {
	for _, t := range r.Tables {
		if len(t.Rows) > 0 {
			return true
		}
	}
	return false
}

// TotalRows returns the total count of differing rows in the report.
func (r RowDiffReport) TotalRows() int {
	count := 0
	for _, t := range r.Tables {
		count += len(t.Rows)
	}
	return count
}

// RowDiffOptions controls which row diffs are fetched.
type RowDiffOptions struct {
	TableFilter  string // optional filter for a specific table
	KeyFilter    string // optional filter for a specific primary key
	StatusFilter string // "updated", "added", "removed", or "all" (default: "updated")
	Limit        int    // max rows to inspect per table (0 = unlimited)
}

// FetchRowDiffReport fetches column-level row data and differences for conflicting / changed rows.
// It supports cross-engine setups where table or column casing differs between prod and dev.
func FetchRowDiffReport(
	ctx context.Context,
	prodDB, devDB *sql.DB,
	prodDriver, devDriver string,
	prodSchema, devSchema *schema.Schema,
	dataDiff content.DataDiff,
	conflicts content.Conflicts,
	opts RowDiffOptions,
) (RowDiffReport, error) {
	report := RowDiffReport{}
	statusFilter := strings.ToLower(strings.TrimSpace(opts.StatusFilter))
	if statusFilter == "" {
		statusFilter = string(StatusUpdated)
	}

	includeUpdated := statusFilter == "all" || statusFilter == string(StatusUpdated)
	includeAdded := statusFilter == "all" || statusFilter == string(StatusAdded)
	includeRemoved := statusFilter == "all" || statusFilter == string(StatusRemoved)

	// Direct single-key lookup optimization when table and key are specified
	if opts.TableFilter != "" && opts.KeyFilter != "" && len(dataDiff.Tables) == 0 {
		conflict := content.Conflict{Table: opts.TableFilter, Key: opts.KeyFilter}
		prod, dev, err := FetchConflictRowsCrossEngine(
			ctx, prodDB, devDB, prodDriver, devDriver, prodSchema, devSchema, conflict,
		)
		if err != nil {
			return report, fmt.Errorf("fetch row data for %s key %s: %w", opts.TableFilter, opts.KeyFilter, err)
		}
		if prod != nil || dev != nil {
			diffs := CompareRows(prod, dev)
			var diffCols []string
			for _, cd := range diffs {
				if cd.Differs {
					diffCols = append(diffCols, cd.Column)
				}
			}
			st := StatusUpdated
			if prod == nil {
				st = StatusAdded
			} else if dev == nil {
				st = StatusRemoved
			}
			report.Tables = append(report.Tables, TableRowDiff{
				Table: opts.TableFilter,
				Rows: []RowDiff{
					{
						Table:       opts.TableFilter,
						Key:         opts.KeyFilter,
						Status:      st,
						Columns:     diffs,
						DiffColumns: diffCols,
					},
				},
			})
		}
		return report, nil
	}

	for _, td := range dataDiff.Tables {
		if opts.TableFilter != "" && schema.CanonicalIdent(td.Table) != schema.CanonicalIdent(opts.TableFilter) {
			continue
		}

		tableRows := TableRowDiff{Table: td.Table}
		count := 0

		// Helper to fetch and append a row diff
		fetchRow := func(key string, status RowDiffStatus) error {
			if opts.KeyFilter != "" && key != opts.KeyFilter {
				return nil
			}
			if opts.Limit > 0 && count >= opts.Limit {
				return nil
			}

			conflict := content.Conflict{Table: td.Table, Key: key}
			prod, dev, err := FetchConflictRowsCrossEngine(
				ctx, prodDB, devDB, prodDriver, devDriver, prodSchema, devSchema, conflict,
			)
			if err != nil {
				return fmt.Errorf("fetch row %s.%s: %w", td.Table, key, err)
			}

			diffs := CompareRows(prod, dev)
			var diffCols []string
			for _, cd := range diffs {
				if cd.Differs {
					diffCols = append(diffCols, cd.Column)
				}
			}

			tableRows.Rows = append(tableRows.Rows, RowDiff{
				Table:       td.Table,
				Key:         key,
				Status:      status,
				Columns:     diffs,
				DiffColumns: diffCols,
			})
			count++
			return nil
		}

		if includeUpdated {
			for _, k := range td.Updated {
				if err := fetchRow(k, StatusUpdated); err != nil {
					return report, err
				}
				if opts.Limit > 0 && count >= opts.Limit {
					break
				}
			}
		}

		if includeRemoved {
			for _, k := range td.Removed {
				if err := fetchRow(k, StatusRemoved); err != nil {
					return report, err
				}
				if opts.Limit > 0 && count >= opts.Limit {
					break
				}
			}
		}

		if includeAdded {
			for _, k := range td.Added {
				if err := fetchRow(k, StatusAdded); err != nil {
					return report, err
				}
				if opts.Limit > 0 && count >= opts.Limit {
					break
				}
			}
		}

		if len(tableRows.Rows) > 0 {
			report.Tables = append(report.Tables, tableRows)
		}
	}

	return report, nil
}

// WriteRowDiffReport writes the row diff report to row_diff.json and row_diff.txt in outDir.
func WriteRowDiffReport(report RowDiffReport, outDir string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}

	// 1. JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal row diff json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "row_diff.json"), data, 0o600); err != nil {
		return fmt.Errorf("write row_diff.json: %w", err)
	}

	// 2. Text
	txt := FormatRowDiffText(report)
	if err := os.WriteFile(filepath.Join(outDir, "row_diff.txt"), []byte(txt), 0o600); err != nil {
		return fmt.Errorf("write row_diff.txt: %w", err)
	}

	return nil
}

// FormatRowDiffText returns a human-readable text representation of row differences.
func FormatRowDiffText(report RowDiffReport) string {
	var b strings.Builder
	if !report.HasChanges() {
		b.WriteString("No row-level differences found.\n")
		return b.String()
	}

	colWidth := 20
	valWidth := 24

	for _, trd := range report.Tables {
		for _, rd := range trd.Rows {
			b.WriteString(strings.Repeat("=", 75) + "\n")
			fmt.Fprintf(&b, "Table: %s | Key: %s (%s)\n", rd.Table, rd.Key, rd.Status)
			b.WriteString(strings.Repeat("=", 75) + "\n")
			fmt.Fprintf(&b, "  %-*s | %-*s | %-*s\n",
				colWidth, "Column",
				valWidth, "Production",
				valWidth, "Development")
			fmt.Fprintf(&b, "  %s+%s+%s\n",
				strings.Repeat("-", colWidth+1),
				strings.Repeat("-", valWidth+2),
				strings.Repeat("-", valWidth+2))

			for _, cd := range rd.Columns {
				prodStr := FormatValue(cd.ProdVal)
				devStr := FormatValue(cd.DevVal)
				marker := ""
				if cd.Differs {
					marker = " *"
				}
				fmt.Fprintf(&b, "  %-*s | %-*s | %-*s%s\n",
					colWidth, cd.Column,
					valWidth, prodStr,
					valWidth, devStr,
					marker)
			}

			b.WriteString("\n  * indicates columns that differ\n")
			b.WriteString(strings.Repeat("=", 75) + "\n\n")
		}
	}

	return b.String()
}
