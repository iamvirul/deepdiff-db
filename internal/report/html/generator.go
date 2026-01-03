package html

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// Generator handles HTML report generation.
type Generator struct {
	options  *ReportOptions
	template *template.Template
}

// NewGenerator creates a new HTML report generator with the given options.
func NewGenerator(opts *ReportOptions) *Generator {
	if opts == nil {
		opts = DefaultReportOptions()
	}

	g := &Generator{
		options: opts,
	}

	// Parse the embedded template
	tmpl, err := template.New("report").Funcs(templateFuncs()).Parse(reportTemplate)
	if err != nil {
		// This should never happen with a valid template
		panic(fmt.Sprintf("failed to parse HTML template: %v", err))
	}
	g.template = tmpl

	return g
}

// templateFuncs returns custom template functions.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05 MST")
		},
		"statusClass": func(status string) string {
			switch status {
			case "OK":
				return "status-ok"
			case "DRIFT":
				return "status-warning"
			default:
				return "status-info"
			}
		},
		"resolutionClass": func(resolution string) string {
			switch resolution {
			case "keep_prod":
				return "resolution-ours"
			case "use_dev":
				return "resolution-theirs"
			case "pending":
				return "resolution-pending"
			default:
				return ""
			}
		},
		"changeTypeClass": func(changeType string) string {
			switch changeType {
			case "added", "added_table":
				return "change-added"
			case "removed", "removed_table":
				return "change-removed"
			case "modified", "type_change", "nullable_change":
				return "change-modified"
			default:
				return ""
			}
		},
	}
}

// GenerateReport creates the HTML report file.
func (g *Generator) GenerateReport(data *ReportData, outPath string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Create output file
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := g.template.Execute(f, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

// BuildReportData constructs the ReportData from individual components.
func BuildReportData(
	prodDB, devDB string,
	schemaDiff *schema.DiffResult,
	dataDiff *content.DataDiff,
	conflicts *content.Conflicts,
	resInfo *content.ResolutionInfo,
	migrationSQL string,
	migrationPack string,
	tablesScanned int,
	opts *ReportOptions,
) *ReportData {
	if opts == nil {
		opts = DefaultReportOptions()
	}

	data := &ReportData{
		GeneratedAt:   time.Now(),
		Version:       "v0.5",
		ProdDB:        prodDB,
		DevDB:         devDB,
		MigrationPack: migrationPack,
	}

	// Build summary
	data.Summary = buildSummary(schemaDiff, dataDiff, conflicts, resInfo, tablesScanned)

	// Process schema diff
	if schemaDiff != nil {
		data.SchemaDiff = schemaDiff
		data.HasSchemaDiff = schemaDiff.HasDrift()
		data.SchemaChanges = buildSchemaChanges(schemaDiff)
	}

	// Process data diff
	if dataDiff != nil {
		data.DataDiff = dataDiff
		data.HasDataDiff = dataDiff.HasChanges()
		data.TableDiffs = buildTableDiffs(dataDiff, opts)
	}

	// Process conflicts
	if conflicts != nil && conflicts.HasConflicts() {
		data.Conflicts = conflicts
		data.HasConflicts = true
		data.ConflictItems = buildConflictItems(conflicts, resInfo)
	}

	// Process resolutions
	if resInfo != nil && resInfo.TotalConflicts > 0 {
		data.ResolutionInfo = resInfo
		data.HasResolutions = true
	}

	// Include migration SQL if requested
	if opts.IncludeMigrationSQL && migrationSQL != "" {
		data.MigrationSQL = migrationSQL
		data.HasMigration = true
	}

	return data
}

// buildSummary constructs the summary statistics.
func buildSummary(
	schemaDiff *schema.DiffResult,
	dataDiff *content.DataDiff,
	conflicts *content.Conflicts,
	resInfo *content.ResolutionInfo,
	tablesScanned int,
) ReportSummary {
	summary := ReportSummary{
		TablesScanned: tablesScanned,
	}

	// Schema status
	if schemaDiff == nil || !schemaDiff.HasDrift() {
		summary.SchemaStatus = "OK"
	} else {
		summary.SchemaStatus = "DRIFT"
	}

	// Data diff stats
	if dataDiff != nil {
		for _, t := range dataDiff.Tables {
			summary.AddedRows += len(t.Added)
			summary.RemovedRows += len(t.Removed)
			summary.UpdatedRows += len(t.Updated)
			if len(t.Added) > 0 || len(t.Removed) > 0 || len(t.Updated) > 0 {
				summary.TablesWithChanges++
			}
		}
	}

	// Conflict stats
	if conflicts != nil {
		summary.TotalConflicts = len(conflicts.Conflicts)
	}

	// Resolution stats
	if resInfo != nil {
		summary.ResolvedConflicts = resInfo.ResolvedCount
		summary.PendingConflicts = resInfo.UnresolvedCount
	} else if conflicts != nil {
		summary.PendingConflicts = len(conflicts.Conflicts)
	}

	return summary
}

// buildSchemaChanges converts schema diff to display format.
func buildSchemaChanges(schemaDiff *schema.DiffResult) []SchemaChangeDisplay {
	var changes []SchemaChangeDisplay

	// Added tables
	for _, table := range schemaDiff.AddedTables {
		changes = append(changes, SchemaChangeDisplay{
			Table:       table.Name,
			ChangeType:  "added_table",
			Description: fmt.Sprintf("Table '%s' exists in dev but not in prod", table.Name),
		})
	}

	// Removed tables
	for _, table := range schemaDiff.RemovedTables {
		changes = append(changes, SchemaChangeDisplay{
			Table:       table,
			ChangeType:  "removed_table",
			Description: fmt.Sprintf("Table '%s' exists in prod but not in dev", table),
		})
	}

	// Table-level changes
	for _, td := range schemaDiff.Tables {
		if !td.HasDifferences {
			continue
		}

		change := SchemaChangeDisplay{
			Table:       td.Name,
			ChangeType:  "modified",
			Description: "Table has column or index changes",
		}

		// Column changes
		for _, col := range td.AddedColumns {
			change.ColumnChanges = append(change.ColumnChanges, ColumnChangeDisplay{
				Column:      col.Name,
				ChangeType:  "added",
				DevValue:    formatColumnDef(col),
				Description: fmt.Sprintf("Column added: %s %s", col.Name, col.DataType),
			})
		}

		for _, col := range td.RemovedColumns {
			change.ColumnChanges = append(change.ColumnChanges, ColumnChangeDisplay{
				Column:       col.Name,
				ChangeType:   "removed",
				ProdValue:    formatColumnDef(col),
				Description:  fmt.Sprintf("Column removed: %s", col.Name),
				IsDestructive: true,
			})
		}

		for _, mod := range td.ModifiedColumns {
			cc := ColumnChangeDisplay{
				Column:      mod.Column,
				ProdValue:   formatColumnDiffProd(mod),
				DevValue:    formatColumnDiffDev(mod),
			}
			if mod.TypeMismatch {
				cc.ChangeType = "type_change"
				cc.Description = fmt.Sprintf("Type changed: %s -> %s", mod.ProdType, mod.DevType)
			}
			if mod.NullableMismatch {
				if cc.ChangeType != "" {
					cc.Description += "; "
				}
				cc.ChangeType = "nullable_change"
				cc.Description += fmt.Sprintf("Nullable changed: %v -> %v", boolStr(mod.ProdNullable), boolStr(mod.DevNullable))
			}
			change.ColumnChanges = append(change.ColumnChanges, cc)
		}

		// Index changes
		for _, idx := range td.AddedIndexes {
			change.IndexChanges = append(change.IndexChanges, IndexChangeDisplay{
				Name:        idx.Name,
				ChangeType:  "added",
				Columns:     idx.Columns,
				IsUnique:    idx.IsUnique,
				Description: fmt.Sprintf("Index added on columns: %v", idx.Columns),
			})
		}

		for _, idx := range td.RemovedIndexes {
			change.IndexChanges = append(change.IndexChanges, IndexChangeDisplay{
				Name:        idx.Name,
				ChangeType:  "removed",
				Columns:     idx.Columns,
				IsUnique:    idx.IsUnique,
				Description: fmt.Sprintf("Index removed: %s", idx.Name),
			})
		}

		for _, idx := range td.ModifiedIndexes {
			change.IndexChanges = append(change.IndexChanges, IndexChangeDisplay{
				Name:        idx.Name,
				ChangeType:  "modified",
				ProdColumns: idx.ProdColumns,
				DevColumns:  idx.DevColumns,
				Description: fmt.Sprintf("Index modified: columns changed from %v to %v", idx.ProdColumns, idx.DevColumns),
			})
		}

		if len(change.ColumnChanges) > 0 || len(change.IndexChanges) > 0 {
			changes = append(changes, change)
		}
	}

	return changes
}

// buildTableDiffs converts data diff to display format.
func buildTableDiffs(dataDiff *content.DataDiff, opts *ReportOptions) []TableDiffDisplay {
	var diffs []TableDiffDisplay

	for _, t := range dataDiff.Tables {
		td := TableDiffDisplay{
			Table:        t.Table,
			AddedCount:   len(t.Added),
			RemovedCount: len(t.Removed),
			UpdatedCount: len(t.Updated),
			HasChanges:   len(t.Added) > 0 || len(t.Removed) > 0 || len(t.Updated) > 0,
		}

		if opts.IncludeDetailedKeys && td.HasChanges {
			maxKeys := opts.MaxKeysPerTable

			td.AddedKeys = limitKeys(t.Added, maxKeys)
			td.RemovedKeys = limitKeys(t.Removed, maxKeys)
			td.UpdatedKeys = limitKeys(t.Updated, maxKeys)
		}

		diffs = append(diffs, td)
	}

	// Sort by table name
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Table < diffs[j].Table
	})

	return diffs
}

// buildConflictItems converts conflicts to display format.
func buildConflictItems(conflicts *content.Conflicts, resInfo *content.ResolutionInfo) []ConflictDisplay {
	var items []ConflictDisplay

	for _, c := range conflicts.Conflicts {
		item := ConflictDisplay{
			Table:    c.Table,
			Key:      c.Key,
			ProdHash: truncateHash(c.ProdHash),
			DevHash:  truncateHash(c.DevHash),
		}

		// If we have resolution info, try to determine the status
		if resInfo != nil {
			// We can't directly map individual conflicts to resolutions here,
			// but we can show the general status
			if resInfo.ResolvedCount > 0 {
				item.IsResolved = true
			}
		}

		items = append(items, item)
	}

	return items
}

// Helper functions

func formatColumnDef(col schema.Column) string {
	nullStr := "NOT NULL"
	if col.IsNullable {
		nullStr = "NULL"
	}
	return fmt.Sprintf("%s %s", col.DataType, nullStr)
}

func formatColumnDiffProd(mod schema.ColumnDiff) string {
	nullStr := "NOT NULL"
	if mod.ProdNullable != nil && *mod.ProdNullable {
		nullStr = "NULL"
	}
	return fmt.Sprintf("%s %s", mod.ProdType, nullStr)
}

func formatColumnDiffDev(mod schema.ColumnDiff) string {
	nullStr := "NOT NULL"
	if mod.DevNullable != nil && *mod.DevNullable {
		nullStr = "NULL"
	}
	return fmt.Sprintf("%s %s", mod.DevType, nullStr)
}

func boolStr(b *bool) string {
	if b == nil {
		return "unknown"
	}
	if *b {
		return "true"
	}
	return "false"
}

func limitKeys(keys []string, max int) []string {
	if len(keys) <= max {
		return keys
	}
	return keys[:max]
}

func truncateHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12] + "..."
}
