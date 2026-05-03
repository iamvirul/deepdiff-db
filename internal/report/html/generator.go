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
	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
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
	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Create output file with restricted permissions (0600)
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
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
	resolutions []resolve.Resolution,
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
		data.ViewChanges = buildViewChanges(schemaDiff)
		data.HasViewChanges = len(data.ViewChanges) > 0
		data.RoutineChanges = buildRoutineChanges(schemaDiff)
		data.HasRoutineChanges = len(data.RoutineChanges) > 0
		data.TriggerChanges = buildTriggerChanges(schemaDiff)
		data.HasTriggerChanges = len(data.TriggerChanges) > 0
		data.SequenceChanges = buildSequenceChanges(schemaDiff)
		data.HasSequenceChanges = len(data.SequenceChanges) > 0
	}

	// Process data diff
	if dataDiff != nil {
		data.DataDiff = dataDiff
		data.HasDataDiff = dataDiff.HasChanges()
		data.TableDiffs = buildTableDiffs(dataDiff, opts)
	}

	// Process conflicts with resolution details
	if conflicts != nil && conflicts.HasConflicts() {
		data.Conflicts = conflicts
		data.HasConflicts = true
		data.ConflictItems = buildConflictItemsWithResolutions(conflicts, resolutions)
	}

	// Process resolutions and build breakdown
	if resInfo != nil && resInfo.TotalConflicts > 0 {
		data.ResolutionInfo = resInfo
		data.HasResolutions = true
	}

	// Build resolution breakdown from resolutions slice
	if len(resolutions) > 0 {
		data.ResolutionBreakdown = buildResolutionBreakdown(resolutions)
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

	if schemaDiff != nil {
		summary.ViewsChanged = len(schemaDiff.AddedViews) + len(schemaDiff.RemovedViews) + len(schemaDiff.ModifiedViews)
		summary.RoutinesChanged = len(schemaDiff.AddedRoutines) + len(schemaDiff.RemovedRoutines) + len(schemaDiff.ModifiedRoutines)
		summary.TriggersChanged = len(schemaDiff.AddedTriggers) + len(schemaDiff.RemovedTriggers) + len(schemaDiff.ModifiedTriggers)
		summary.SequencesChanged = len(schemaDiff.AddedSequences) + len(schemaDiff.RemovedSequences) + len(schemaDiff.ModifiedSequences)
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
				Column:        col.Name,
				ChangeType:    "removed",
				ProdValue:     formatColumnDef(col),
				Description:   fmt.Sprintf("Column removed: %s", col.Name),
				IsDestructive: true,
			})
		}

		for _, mod := range td.ModifiedColumns {
			cc := ColumnChangeDisplay{
				Column:    mod.Column,
				ProdValue: formatColumnDiffProd(mod),
				DevValue:  formatColumnDiffDev(mod),
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

		// Foreign key changes
		for _, fk := range td.AddedForeignKeys {
			change.ForeignKeyChanges = append(change.ForeignKeyChanges, ForeignKeyChangeDisplay{
				Name:        fk.Name,
				ChangeType:  "added",
				Columns:     fk.Columns,
				RefTable:    fk.ReferencedTable,
				RefColumns:  fk.ReferencedColumns,
				OnDelete:    fk.OnDelete,
				OnUpdate:    fk.OnUpdate,
				Description: fmt.Sprintf("FK added: %s -> %s(%s)", fk.Name, fk.ReferencedTable, strings.Join(fk.ReferencedColumns, ", ")),
			})
		}

		for _, fk := range td.RemovedForeignKeys {
			change.ForeignKeyChanges = append(change.ForeignKeyChanges, ForeignKeyChangeDisplay{
				Name:          fk.Name,
				ChangeType:    "removed",
				Columns:       fk.Columns,
				RefTable:      fk.ReferencedTable,
				RefColumns:    fk.ReferencedColumns,
				OnDelete:      fk.OnDelete,
				OnUpdate:      fk.OnUpdate,
				Description:   fmt.Sprintf("FK removed: %s", fk.Name),
				IsDestructive: true,
			})
		}

		for _, fkDiff := range td.ModifiedForeignKeys {
			desc := fmt.Sprintf("FK modified: %s", fkDiff.Name)
			if fkDiff.OnDeleteDiffers {
				desc += fmt.Sprintf(" (ON DELETE: %s -> %s)", fkDiff.ProdOnDelete, fkDiff.DevOnDelete)
			}
			if fkDiff.OnUpdateDiffers {
				desc += fmt.Sprintf(" (ON UPDATE: %s -> %s)", fkDiff.ProdOnUpdate, fkDiff.DevOnUpdate)
			}
			change.ForeignKeyChanges = append(change.ForeignKeyChanges, ForeignKeyChangeDisplay{
				Name:         fkDiff.Name,
				ChangeType:   "modified",
				ProdOnDelete: fkDiff.ProdOnDelete,
				ProdOnUpdate: fkDiff.ProdOnUpdate,
				DevOnDelete:  fkDiff.DevOnDelete,
				DevOnUpdate:  fkDiff.DevOnUpdate,
				Description:  desc,
			})
		}

		if len(change.ColumnChanges) > 0 || len(change.IndexChanges) > 0 || len(change.ForeignKeyChanges) > 0 {
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

// buildConflictItemsWithResolutions converts conflicts to display format with resolution details.
func buildConflictItemsWithResolutions(conflicts *content.Conflicts, resolutions []resolve.Resolution) []ConflictDisplay {
	var items []ConflictDisplay

	// Build a map of resolutions by table+key for quick lookup
	resolutionMap := make(map[string]resolve.Resolution)
	for _, r := range resolutions {
		key := r.Conflict.Table + ":" + r.Conflict.Key
		resolutionMap[key] = r
	}

	for _, c := range conflicts.Conflicts {
		item := ConflictDisplay{
			Table:    c.Table,
			Key:      c.Key,
			ProdHash: truncateHash(c.ProdHash),
			DevHash:  truncateHash(c.DevHash),
		}

		// Look up resolution for this conflict
		key := c.Table + ":" + c.Key
		if res, ok := resolutionMap[key]; ok {
			item.Strategy = string(res.Strategy)
			item.Decision = string(res.Decision)
			item.Resolution = string(res.Decision)
			item.IsResolved = res.Resolved
		}

		items = append(items, item)
	}

	return items
}

// buildResolutionBreakdown creates resolution statistics from resolutions.
func buildResolutionBreakdown(resolutions []resolve.Resolution) *ResolutionBreakdown {
	breakdown := &ResolutionBreakdown{
		TotalConflicts: len(resolutions),
	}

	// Count by decision type
	tableStats := make(map[string]*TableStrategyDisplay)

	for _, r := range resolutions {
		switch r.Decision {
		case resolve.DecisionKeepProd:
			breakdown.AutoResolvedOurs++
		case resolve.DecisionUseDev:
			breakdown.AutoResolvedTheirs++
		case resolve.DecisionPending:
			breakdown.PendingManual++
		}

		// Build per-table stats
		ts, ok := tableStats[r.Conflict.Table]
		if !ok {
			ts = &TableStrategyDisplay{
				Table:    r.Conflict.Table,
				Strategy: string(r.Strategy),
			}
			tableStats[r.Conflict.Table] = ts
		}
		ts.ConflictCount++
		if r.Resolved {
			ts.ResolvedCount++
		} else {
			ts.PendingCount++
		}
	}

	// Convert map to slice and sort by table name
	for _, ts := range tableStats {
		breakdown.TableStrategies = append(breakdown.TableStrategies, *ts)
	}
	sort.Slice(breakdown.TableStrategies, func(i, j int) bool {
		return breakdown.TableStrategies[i].Table < breakdown.TableStrategies[j].Table
	})

	return breakdown
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

// buildViewChanges converts view diff to display format.
func buildViewChanges(schemaDiff *schema.DiffResult) []ViewChangeDisplay {
	var changes []ViewChangeDisplay
	for _, v := range schemaDiff.AddedViews {
		changes = append(changes, ViewChangeDisplay{
			Name:           v.Name,
			ChangeType:     "added",
			IsMaterialized: v.IsMaterialized,
			Description:    fmt.Sprintf("View '%s' exists in dev but not in prod", v.Name),
		})
	}
	for _, v := range schemaDiff.RemovedViews {
		changes = append(changes, ViewChangeDisplay{
			Name:           v.Name,
			ChangeType:     "removed",
			IsMaterialized: v.IsMaterialized,
			Description:    fmt.Sprintf("View '%s' exists in prod but not in dev", v.Name),
		})
	}
	for _, vd := range schemaDiff.ModifiedViews {
		desc := "View definition changed"
		if vd.IsMaterializedDiffers {
			desc += "; materialization type differs"
		}
		changes = append(changes, ViewChangeDisplay{
			Name:        vd.Name,
			ChangeType:  "modified",
			Description: desc,
		})
	}
	return changes
}

// buildRoutineChanges converts routine diff to display format.
func buildRoutineChanges(schemaDiff *schema.DiffResult) []RoutineChangeDisplay {
	var changes []RoutineChangeDisplay
	for _, r := range schemaDiff.AddedRoutines {
		changes = append(changes, RoutineChangeDisplay{
			Name:        r.Name,
			Kind:        r.Kind,
			ChangeType:  "added",
			Description: fmt.Sprintf("%s '%s' exists in dev but not in prod", r.Kind, r.Name),
		})
	}
	for _, name := range schemaDiff.RemovedRoutines {
		changes = append(changes, RoutineChangeDisplay{
			Name:          name,
			ChangeType:    "removed",
			Description:   fmt.Sprintf("Routine '%s' exists in prod but not in dev", name),
			IsDestructive: true,
		})
	}
	for _, rd := range schemaDiff.ModifiedRoutines {
		var parts []string
		if rd.DefinitionDiffers {
			parts = append(parts, "definition changed")
		}
		if rd.KindDiffers {
			parts = append(parts, fmt.Sprintf("kind: %s -> %s", rd.ProdKind, rd.DevKind))
		}
		if rd.ReturnTypeDiffers {
			parts = append(parts, fmt.Sprintf("return type: %s -> %s", rd.ProdReturnType, rd.DevReturnType))
		}
		if rd.LanguageDiffers {
			parts = append(parts, fmt.Sprintf("language: %s -> %s", rd.ProdLanguage, rd.DevLanguage))
		}
		if rd.ParametersDiffers {
			parts = append(parts, "parameters changed")
		}
		desc := strings.Join(parts, "; ")
		if desc == "" {
			desc = "Routine modified"
		}
		changes = append(changes, RoutineChangeDisplay{
			Name:        rd.Name,
			Kind:        rd.ProdKind,
			ChangeType:  "modified",
			Description: desc,
		})
	}
	return changes
}

// buildTriggerChanges converts trigger diff to display format.
func buildTriggerChanges(schemaDiff *schema.DiffResult) []TriggerChangeDisplay {
	var changes []TriggerChangeDisplay
	for _, t := range schemaDiff.AddedTriggers {
		changes = append(changes, TriggerChangeDisplay{
			Name:        t.Name,
			Table:       t.Table,
			ChangeType:  "added",
			Description: fmt.Sprintf("Trigger '%s' on table '%s' exists in dev but not in prod", t.Name, t.Table),
		})
	}
	for _, name := range schemaDiff.RemovedTriggers {
		changes = append(changes, TriggerChangeDisplay{
			Name:          name,
			ChangeType:    "removed",
			Description:   fmt.Sprintf("Trigger '%s' exists in prod but not in dev", name),
			IsDestructive: true,
		})
	}
	for _, td := range schemaDiff.ModifiedTriggers {
		var parts []string
		if td.TimingDiffers {
			parts = append(parts, fmt.Sprintf("timing: %s -> %s", td.ProdTiming, td.DevTiming))
		}
		if td.EventDiffers {
			parts = append(parts, fmt.Sprintf("event: %s -> %s", td.ProdEvent, td.DevEvent))
		}
		if td.DefinitionDiffers {
			parts = append(parts, "definition changed")
		}
		desc := strings.Join(parts, "; ")
		if desc == "" {
			desc = "Trigger modified"
		}
		changes = append(changes, TriggerChangeDisplay{
			Name:        td.Name,
			ChangeType:  "modified",
			Description: desc,
		})
	}
	return changes
}

// buildSequenceChanges converts sequence diff to display format.
func buildSequenceChanges(schemaDiff *schema.DiffResult) []SequenceChangeDisplay {
	var changes []SequenceChangeDisplay
	for _, seq := range schemaDiff.AddedSequences {
		changes = append(changes, SequenceChangeDisplay{
			Name:        seq.Name,
			ChangeType:  "added",
			Description: fmt.Sprintf("Sequence '%s' exists in dev but not in prod", seq.Name),
		})
	}
	for _, name := range schemaDiff.RemovedSequences {
		changes = append(changes, SequenceChangeDisplay{
			Name:        name,
			ChangeType:  "removed",
			Description: fmt.Sprintf("Sequence '%s' exists in prod but not in dev", name),
		})
	}
	for _, sd := range schemaDiff.ModifiedSequences {
		var parts []string
		if sd.StartValueDiffers {
			parts = append(parts, fmt.Sprintf("start value: %d -> %d", sd.ProdStartValue, sd.DevStartValue))
		}
		if sd.IncrementDiffers {
			parts = append(parts, fmt.Sprintf("increment: %d -> %d", sd.ProdIncrement, sd.DevIncrement))
		}
		if sd.MinValueDiffers {
			parts = append(parts, fmt.Sprintf("min value: %d -> %d", sd.ProdMinValue, sd.DevMinValue))
		}
		if sd.MaxValueDiffers {
			parts = append(parts, fmt.Sprintf("max value: %d -> %d", sd.ProdMaxValue, sd.DevMaxValue))
		}
		if sd.CacheSizeDiffers {
			parts = append(parts, fmt.Sprintf("cache size: %d -> %d", sd.ProdCacheSize, sd.DevCacheSize))
		}
		if sd.CycleDiffers {
			prodCycle := "false"
			if sd.ProdCycle != nil && *sd.ProdCycle {
				prodCycle = "true"
			}
			devCycle := "false"
			if sd.DevCycle != nil && *sd.DevCycle {
				devCycle = "true"
			}
			parts = append(parts, fmt.Sprintf("cycle: %s -> %s", prodCycle, devCycle))
		}
		desc := strings.Join(parts, "; ")
		if desc == "" {
			desc = "Sequence modified"
		}
		changes = append(changes, SequenceChangeDisplay{
			Name:        sd.Name,
			ChangeType:  "modified",
			Description: desc,
		})
	}
	return changes
}
