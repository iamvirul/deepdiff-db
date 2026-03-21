package html

import (
	"time"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ReportData contains all data needed for the HTML report.
type ReportData struct {
	// Metadata
	GeneratedAt   time.Time `json:"generated_at"`
	Version       string    `json:"version"`
	ProdDB        string    `json:"prod_db"`
	DevDB         string    `json:"dev_db"`
	MigrationPack string    `json:"migration_pack,omitempty"`

	// Summary statistics
	Summary ReportSummary `json:"summary"`

	// Schema information
	SchemaDiff    *schema.DiffResult    `json:"schema_diff,omitempty"`
	SchemaChanges []SchemaChangeDisplay `json:"schema_changes,omitempty"`
	HasSchemaDiff bool                  `json:"has_schema_diff"`

	// Data differences
	DataDiff    *content.DataDiff  `json:"data_diff,omitempty"`
	TableDiffs  []TableDiffDisplay `json:"table_diffs,omitempty"`
	HasDataDiff bool               `json:"has_data_diff"`

	// Conflicts
	Conflicts     *content.Conflicts `json:"conflicts,omitempty"`
	ConflictItems []ConflictDisplay  `json:"conflict_items,omitempty"`
	HasConflicts  bool               `json:"has_conflicts"`

	// Resolution info
	ResolutionInfo      *content.ResolutionInfo `json:"resolution_info,omitempty"`
	HasResolutions      bool                    `json:"has_resolutions"`
	ResolutionBreakdown *ResolutionBreakdown    `json:"resolution_breakdown,omitempty"`

	// SQL Migration
	MigrationSQL string `json:"migration_sql,omitempty"`
	HasMigration bool   `json:"has_migration"`
}

// ReportSummary contains aggregate statistics.
type ReportSummary struct {
	SchemaStatus      string `json:"schema_status"`
	TablesScanned     int    `json:"tables_scanned"`
	TablesWithChanges int    `json:"tables_with_changes"`
	AddedRows         int    `json:"added_rows"`
	RemovedRows       int    `json:"removed_rows"`
	UpdatedRows       int    `json:"updated_rows"`
	TotalConflicts    int    `json:"total_conflicts"`
	ResolvedConflicts int    `json:"resolved_conflicts"`
	PendingConflicts  int    `json:"pending_conflicts"`
}

// SchemaChangeDisplay represents a schema change for display.
type SchemaChangeDisplay struct {
	Table             string                    `json:"table"`
	ChangeType        string                    `json:"change_type"` // "added_table", "removed_table", "modified"
	Description       string                    `json:"description"`
	ColumnChanges     []ColumnChangeDisplay     `json:"column_changes,omitempty"`
	IndexChanges      []IndexChangeDisplay      `json:"index_changes,omitempty"`
	ForeignKeyChanges []ForeignKeyChangeDisplay `json:"foreign_key_changes,omitempty"`
}

// ColumnChangeDisplay represents a column change for display.
type ColumnChangeDisplay struct {
	Column        string `json:"column"`
	ChangeType    string `json:"change_type"` // "added", "removed", "type_change", "nullable_change"
	ProdValue     string `json:"prod_value,omitempty"`
	DevValue      string `json:"dev_value,omitempty"`
	Description   string `json:"description"`
	IsDestructive bool   `json:"is_destructive"`
}

// IndexChangeDisplay represents an index change for display.
type IndexChangeDisplay struct {
	Name        string   `json:"name"`
	ChangeType  string   `json:"change_type"` // "added", "removed", "modified"
	Columns     []string `json:"columns,omitempty"`
	ProdColumns []string `json:"prod_columns,omitempty"`
	DevColumns  []string `json:"dev_columns,omitempty"`
	IsUnique    bool     `json:"is_unique"`
	Description string   `json:"description"`
}

// ForeignKeyChangeDisplay represents a foreign key change for display.
type ForeignKeyChangeDisplay struct {
	Name          string   `json:"name"`
	ChangeType    string   `json:"change_type"` // "added", "removed", "modified"
	Columns       []string `json:"columns,omitempty"`
	RefTable      string   `json:"ref_table"`
	RefColumns    []string `json:"ref_columns,omitempty"`
	OnDelete      string   `json:"on_delete,omitempty"`
	OnUpdate      string   `json:"on_update,omitempty"`
	ProdOnDelete  string   `json:"prod_on_delete,omitempty"`
	ProdOnUpdate  string   `json:"prod_on_update,omitempty"`
	DevOnDelete   string   `json:"dev_on_delete,omitempty"`
	DevOnUpdate   string   `json:"dev_on_update,omitempty"`
	Description   string   `json:"description"`
	IsDestructive bool     `json:"is_destructive"`
}

// TableDiffDisplay represents a table's data differences for display.
type TableDiffDisplay struct {
	Table        string   `json:"table"`
	AddedCount   int      `json:"added_count"`
	RemovedCount int      `json:"removed_count"`
	UpdatedCount int      `json:"updated_count"`
	AddedKeys    []string `json:"added_keys,omitempty"`
	RemovedKeys  []string `json:"removed_keys,omitempty"`
	UpdatedKeys  []string `json:"updated_keys,omitempty"`
	HasChanges   bool     `json:"has_changes"`
}

// ConflictDisplay represents a conflict for display.
type ConflictDisplay struct {
	Table      string `json:"table"`
	Key        string `json:"key"`
	ProdHash   string `json:"prod_hash"`
	DevHash    string `json:"dev_hash"`
	Resolution string `json:"resolution,omitempty"` // "keep_prod", "use_dev", "pending"
	Strategy   string `json:"strategy,omitempty"`   // "ours", "theirs", "manual"
	Decision   string `json:"decision,omitempty"`   // "keep_prod", "use_dev", "pending"
	IsResolved bool   `json:"is_resolved"`
}

// TableStrategyDisplay shows the resolution strategy for a table.
type TableStrategyDisplay struct {
	Table         string `json:"table"`
	Strategy      string `json:"strategy"` // "ours", "theirs", "manual"
	ConflictCount int    `json:"conflict_count"`
	ResolvedCount int    `json:"resolved_count"`
	PendingCount  int    `json:"pending_count"`
}

// ResolutionBreakdown shows resolution statistics by type.
type ResolutionBreakdown struct {
	TotalConflicts     int                    `json:"total_conflicts"`
	AutoResolvedOurs   int                    `json:"auto_resolved_ours"`   // keep_prod
	AutoResolvedTheirs int                    `json:"auto_resolved_theirs"` // use_dev
	PendingManual      int                    `json:"pending_manual"`
	TableStrategies    []TableStrategyDisplay `json:"table_strategies,omitempty"`
}

// ReportOptions configures HTML report generation.
type ReportOptions struct {
	// Include full SQL migration in report
	IncludeMigrationSQL bool

	// Include detailed row keys (may be large)
	IncludeDetailedKeys bool

	// Maximum number of keys to display per table
	MaxKeysPerTable int

	// Enable PDF export button (requires browser print)
	EnablePDFExport bool
}

// DefaultReportOptions returns sensible defaults for report generation.
func DefaultReportOptions() *ReportOptions {
	return &ReportOptions{
		IncludeMigrationSQL: true,
		IncludeDetailedKeys: true,
		MaxKeysPerTable:     100,
		EnablePDFExport:     true,
	}
}
