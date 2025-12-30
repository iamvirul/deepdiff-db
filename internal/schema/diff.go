package schema

import (
	"sort"
	"strings"
)

// TableDiff describes differences for a single table.
type TableDiff struct {
	Name                string           `json:"table"`
	Table               string           `json:"-"` // Deprecated: use Name
	MissingInProd       bool             `json:"missing_in_prod,omitempty"`
	MissingInDev        bool             `json:"missing_in_dev,omitempty"`
	ColumnDiffs         []ColumnDiff     `json:"column_diffs,omitempty"`
	AddedColumns        []Column         `json:"added_columns,omitempty"`
	RemovedColumns      []Column         `json:"removed_columns,omitempty"`
	ModifiedColumns     []ColumnDiff     `json:"modified_columns,omitempty"`
	AddedIndexes        []Index          `json:"added_indexes,omitempty"`
	RemovedIndexes      []Index          `json:"removed_indexes,omitempty"`
	ModifiedIndexes     []IndexDiff      `json:"modified_indexes,omitempty"`
	AddedForeignKeys    []ForeignKey     `json:"added_foreign_keys,omitempty"`
	RemovedForeignKeys  []ForeignKey     `json:"removed_foreign_keys,omitempty"`
	ModifiedForeignKeys []ForeignKeyDiff `json:"modified_foreign_keys,omitempty"`
	PrimaryKeyDiff      *PrimaryKeyDiff  `json:"primary_key_diff,omitempty"`
	ExtraInProd         []string         `json:"extra_in_prod,omitempty"`
	ExtraInDev          []string         `json:"extra_in_dev,omitempty"`
	OnlyInProd          bool             `json:"only_in_prod,omitempty"`
	OnlyInDev           bool             `json:"only_in_dev,omitempty"`
	HasDifferences      bool             `json:"has_differences"`
}

// ColumnDiff captures mismatches for a column across prod/dev.
type ColumnDiff struct {
	Column           string  `json:"column"`
	MissingInProd    bool    `json:"missing_in_prod,omitempty"`
	MissingInDev     bool    `json:"missing_in_dev,omitempty"`
	TypeMismatch     bool    `json:"type_mismatch,omitempty"`
	ProdType         string  `json:"prod_type,omitempty"`
	DevType          string  `json:"dev_type,omitempty"`
	NullableMismatch bool    `json:"nullable_mismatch,omitempty"`
	ProdNullable     *bool   `json:"prod_nullable,omitempty"`
	DevNullable      *bool   `json:"dev_nullable,omitempty"`
	DefaultMismatch  bool    `json:"default_mismatch,omitempty"`
	ProdDefault      *string `json:"prod_default,omitempty"`
	DevDefault       *string `json:"dev_default,omitempty"`
}

// IndexDiff captures mismatches for an index across prod/dev.
type IndexDiff struct {
	Name          string   `json:"index_name"`
	MissingInProd bool     `json:"missing_in_prod,omitempty"`
	MissingInDev  bool     `json:"missing_in_dev,omitempty"`
	ColumnsDiffer bool     `json:"columns_differ,omitempty"`
	ProdColumns   []string `json:"prod_columns,omitempty"`
	DevColumns    []string `json:"dev_columns,omitempty"`
	UniqueDiffers bool     `json:"unique_differs,omitempty"`
	ProdUnique    *bool    `json:"prod_unique,omitempty"`
	DevUnique     *bool    `json:"dev_unique,omitempty"`
}

// ForeignKeyDiff captures mismatches for a foreign key across prod/dev.
type ForeignKeyDiff struct {
	Name                    string   `json:"fk_name"`
	MissingInProd           bool     `json:"missing_in_prod,omitempty"`
	MissingInDev            bool     `json:"missing_in_dev,omitempty"`
	ColumnsDiffer           bool     `json:"columns_differ,omitempty"`
	ProdColumns             []string `json:"prod_columns,omitempty"`
	DevColumns              []string `json:"dev_columns,omitempty"`
	ReferencedTableDiffers  bool     `json:"referenced_table_differs,omitempty"`
	ProdReferencedTable     string   `json:"prod_referenced_table,omitempty"`
	DevReferencedTable      string   `json:"dev_referenced_table,omitempty"`
	ReferencedColumnsDiffer bool     `json:"referenced_columns_differ,omitempty"`
	ProdReferencedColumns   []string `json:"prod_referenced_columns,omitempty"`
	DevReferencedColumns    []string `json:"dev_referenced_columns,omitempty"`
	OnDeleteDiffers         bool     `json:"on_delete_differs,omitempty"`
	ProdOnDelete            string   `json:"prod_on_delete,omitempty"`
	DevOnDelete             string   `json:"dev_on_delete,omitempty"`
	OnUpdateDiffers         bool     `json:"on_update_differs,omitempty"`
	ProdOnUpdate            string   `json:"prod_on_update,omitempty"`
	DevOnUpdate             string   `json:"dev_on_update,omitempty"`
}

// PrimaryKeyDiff captures mismatches for primary keys across prod/dev.
type PrimaryKeyDiff struct {
	ProdColumns []string `json:"prod_columns,omitempty"`
	DevColumns  []string `json:"dev_columns,omitempty"`
}

// DiffResult aggregates all table diffs.
type DiffResult struct {
	Tables        []TableDiff `json:"tables"`
	AddedTables   []Table     `json:"added_tables,omitempty"`   // Full table definitions for CREATE TABLE
	RemovedTables []string    `json:"removed_tables,omitempty"` // Table names for DROP TABLE
}

// DiffSchemas compares two Schema values and returns a DiffResult that describes table- and column-level differences between them.
// It aggregates table names from both schemas, processes tables in sorted order, marks tables present only on one side (setting MissingInProd/MissingInDev and OnlyInProd/OnlyInDev) and computes column-level differences for tables present in both, setting HasDifferences when any mismatches are found.
func DiffSchemas(prod, dev *Schema) DiffResult {
	if prod == nil {
		prod = &Schema{}
	}
	if dev == nil {
		dev = &Schema{}
	}
	result := DiffResult{}

	seen := make(map[string]struct{})
	for name := range prod.Tables {
		seen[name] = struct{}{}
	}
	for name := range dev.Tables {
		seen[name] = struct{}{}
	}

	var tableNames []string
	for name := range seen {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	for _, tbl := range tableNames {
		p, prodOK := prod.Tables[tbl]
		d, devOK := dev.Tables[tbl]

		td := TableDiff{Name: tbl, Table: tbl}
		if prodOK && !devOK {
			td.MissingInDev = true
			td.OnlyInProd = true
			td.HasDifferences = true
			result.Tables = append(result.Tables, td)
			result.RemovedTables = append(result.RemovedTables, tbl)
			continue
		}
		if !prodOK && devOK {
			td.MissingInProd = true
			td.OnlyInDev = true
			td.HasDifferences = true
			result.Tables = append(result.Tables, td)
			result.AddedTables = append(result.AddedTables, d) // Store full table definition
			continue
		}

		// Process column differences
		td.ColumnDiffs = diffColumns(p.Columns, d.Columns)
		td.AddedColumns, td.RemovedColumns, td.ModifiedColumns = categorizeColumnDiffs(td.ColumnDiffs, d.Columns, p.Columns)

		// Process index differences
		td.AddedIndexes, td.RemovedIndexes, td.ModifiedIndexes = diffIndexes(p.Indexes, d.Indexes)

		// Process foreign key differences
		td.AddedForeignKeys, td.RemovedForeignKeys, td.ModifiedForeignKeys = diffForeignKeys(p.ForeignKeys, d.ForeignKeys)

		// Process primary key differences
		td.PrimaryKeyDiff = diffPrimaryKey(p.PrimaryKey, d.PrimaryKey)

		td.HasDifferences = len(td.ColumnDiffs) > 0 ||
			len(td.AddedIndexes) > 0 || len(td.RemovedIndexes) > 0 || len(td.ModifiedIndexes) > 0 ||
			len(td.AddedForeignKeys) > 0 || len(td.RemovedForeignKeys) > 0 || len(td.ModifiedForeignKeys) > 0 ||
			td.PrimaryKeyDiff != nil
		result.Tables = append(result.Tables, td)
	}

	return result
}

// HasDrift indicates whether any differences were found.
func (d DiffResult) HasDrift() bool {
	for _, t := range d.Tables {
		if t.HasDifferences {
			return true
		}
	}
	return false
}

// diffColumns computes column-level differences between prodCols and devCols.
// It returns a slice of ColumnDiff describing columns that are missing on one side,
// have differing type strings (normalized for comparison), or have differing nullability.
// Each ColumnDiff includes the column name and populated fields for observed types
// and nullable values where applicable. The returned diffs are ordered by column name.
func diffColumns(prodCols, devCols map[string]Column) []ColumnDiff {
	seen := make(map[string]struct{})
	for name := range prodCols {
		seen[name] = struct{}{}
	}
	for name := range devCols {
		seen[name] = struct{}{}
	}

	var names []string
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	var diffs []ColumnDiff
	for _, col := range names {
		p, pOK := prodCols[col]
		d, dOK := devCols[col]

		cd := ColumnDiff{Column: col}

		if pOK && !dOK {
			cd.MissingInDev = true
			cd.ProdType = p.DataType
			cd.ProdNullable = &p.IsNullable
			diffs = append(diffs, cd)
			continue
		}
		if !pOK && dOK {
			cd.MissingInProd = true
			cd.DevType = d.DataType
			cd.DevNullable = &d.IsNullable
			diffs = append(diffs, cd)
			continue
		}

		// Check for type mismatch
		if stringsDiffer(p.DataType, d.DataType) {
			cd.TypeMismatch = true
			cd.ProdType = p.DataType
			cd.DevType = d.DataType
		}

		// Check for nullable mismatch
		if p.IsNullable != d.IsNullable {
			cd.NullableMismatch = true
		}

		// Check for default value mismatch
		if defaultsDiffer(p.DefaultValue, d.DefaultValue) {
			cd.DefaultMismatch = true
		}

		// If there's any mismatch, populate all fields needed for migration generation
		if cd.TypeMismatch || cd.NullableMismatch || cd.DefaultMismatch {
			cd.ProdNullable = &p.IsNullable
			cd.DevNullable = &d.IsNullable
			cd.ProdDefault = p.DefaultValue
			cd.DevDefault = d.DefaultValue
			if cd.ProdType == "" {
				cd.ProdType = p.DataType
			}
			if cd.DevType == "" {
				cd.DevType = d.DataType
			}
			diffs = append(diffs, cd)
		}
	}

	return diffs
}

// stringsDiffer reports whether two type strings differ after normalization.
// The comparison ignores surrounding whitespace and letter case.
func stringsDiffer(a, b string) bool {
	return normalizeType(a) != normalizeType(b)
}

// defaultsDiffer reports whether two default values differ.
// It handles nil pointers (no default) and string values, with normalization.
func defaultsDiffer(a, b *string) bool {
	// Both nil - no difference
	if a == nil && b == nil {
		return false
	}
	// One nil, one not - difference
	if a == nil || b == nil {
		return true
	}
	// Both have values - compare normalized versions
	return normalizeDefault(*a) != normalizeDefault(*b)
}

// normalizeDefault normalizes a default value string for comparison.
// It trims whitespace and handles common variations in default value representation.
func normalizeDefault(val string) string {
	normalized := strings.TrimSpace(val)
	// Remove surrounding quotes if present (handle both single and double quotes)
	if len(normalized) >= 2 &&
		((strings.HasPrefix(normalized, "'") && strings.HasSuffix(normalized, "'")) ||
			(strings.HasPrefix(normalized, "\"") && strings.HasSuffix(normalized, "\""))) {
		normalized = normalized[1 : len(normalized)-1]
	}
	return normalized
}

// normalizeType trims leading and trailing whitespace from t and converts it to lower-case.
// The result is suitable for case- and space-insensitive type comparisons.
func normalizeType(t string) string {
	return strings.TrimSpace(strings.ToLower(t))
}

// NormalizeType normalizes a data type string for comparison.
// NormalizeType returns a normalized form of the type string by trimming whitespace and converting it to lower case.
func NormalizeType(t string) string {
	return normalizeType(t)
}

// categorizeColumnDiffs separates column diffs into added, removed, and modified columns.
// It returns three slices: added columns (from dev), removed columns (from prod), and modified column diffs.
func categorizeColumnDiffs(diffs []ColumnDiff, devCols, prodCols map[string]Column) (added []Column, removed []Column, modified []ColumnDiff) {
	for _, cd := range diffs {
		if cd.MissingInProd {
			// Column exists in dev but not in prod - it should be added
			if col, ok := devCols[cd.Column]; ok {
				added = append(added, col)
			}
		} else if cd.MissingInDev {
			// Column exists in prod but not in dev - it should be removed
			if col, ok := prodCols[cd.Column]; ok {
				removed = append(removed, col)
			}
		} else if cd.TypeMismatch || cd.NullableMismatch || cd.DefaultMismatch {
			// Column exists in both but differs - it should be modified
			modified = append(modified, cd)
		}
	}
	return
}

// diffIndexes compares indexes between prod and dev tables.
// It returns three slices: added indexes (in dev but not prod), removed indexes (in prod but not dev),
// and modified indexes (exist in both but differ in columns or uniqueness).
func diffIndexes(prodIndexes, devIndexes map[string]Index) (added []Index, removed []Index, modified []IndexDiff) {
	if prodIndexes == nil {
		prodIndexes = make(map[string]Index)
	}
	if devIndexes == nil {
		devIndexes = make(map[string]Index)
	}

	// Collect all index names
	seen := make(map[string]struct{})
	for name := range prodIndexes {
		seen[name] = struct{}{}
	}
	for name := range devIndexes {
		seen[name] = struct{}{}
	}

	var names []string
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		prodIdx, prodOK := prodIndexes[name]
		devIdx, devOK := devIndexes[name]

		if prodOK && !devOK {
			// Index exists in prod but not in dev - should be removed
			removed = append(removed, prodIdx)
			continue
		}

		if !prodOK && devOK {
			// Index exists in dev but not in prod - should be added
			added = append(added, devIdx)
			continue
		}

		// Index exists in both - check for differences
		diff := IndexDiff{Name: name}
		hasDiff := false

		// Compare columns (order matters)
		if !slicesEqual(prodIdx.Columns, devIdx.Columns) {
			diff.ColumnsDiffer = true
			diff.ProdColumns = prodIdx.Columns
			diff.DevColumns = devIdx.Columns
			hasDiff = true
		}

		// Compare uniqueness
		if prodIdx.IsUnique != devIdx.IsUnique {
			diff.UniqueDiffers = true
			diff.ProdUnique = &prodIdx.IsUnique
			diff.DevUnique = &devIdx.IsUnique
			hasDiff = true
		}

		if hasDiff {
			modified = append(modified, diff)
		}
	}

	return
}

// slicesEqual compares two string slices for equality (same length and elements in same order).
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// diffForeignKeys compares foreign keys between prod and dev tables.
// It returns three slices: added foreign keys (in dev but not prod), removed foreign keys (in prod but not dev),
// and modified foreign keys (exist in both but differ in definition).
func diffForeignKeys(prodFKs, devFKs map[string]ForeignKey) (added []ForeignKey, removed []ForeignKey, modified []ForeignKeyDiff) {
	if prodFKs == nil {
		prodFKs = make(map[string]ForeignKey)
	}
	if devFKs == nil {
		devFKs = make(map[string]ForeignKey)
	}

	// Collect all foreign key names
	seen := make(map[string]struct{})
	for name := range prodFKs {
		seen[name] = struct{}{}
	}
	for name := range devFKs {
		seen[name] = struct{}{}
	}

	var names []string
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		prodFK, prodOK := prodFKs[name]
		devFK, devOK := devFKs[name]

		if prodOK && !devOK {
			// Foreign key exists in prod but not in dev - should be removed
			removed = append(removed, prodFK)
			continue
		}

		if !prodOK && devOK {
			// Foreign key exists in dev but not in prod - should be added
			added = append(added, devFK)
			continue
		}

		// Foreign key exists in both - check for differences
		diff := ForeignKeyDiff{Name: name}
		hasDiff := false

		// Compare source columns
		if !slicesEqual(prodFK.Columns, devFK.Columns) {
			diff.ColumnsDiffer = true
			diff.ProdColumns = prodFK.Columns
			diff.DevColumns = devFK.Columns
			hasDiff = true
		}

		// Compare referenced table
		if prodFK.ReferencedTable != devFK.ReferencedTable {
			diff.ReferencedTableDiffers = true
			diff.ProdReferencedTable = prodFK.ReferencedTable
			diff.DevReferencedTable = devFK.ReferencedTable
			hasDiff = true
		}

		// Compare referenced columns
		if !slicesEqual(prodFK.ReferencedColumns, devFK.ReferencedColumns) {
			diff.ReferencedColumnsDiffer = true
			diff.ProdReferencedColumns = prodFK.ReferencedColumns
			diff.DevReferencedColumns = devFK.ReferencedColumns
			hasDiff = true
		}

		// Compare ON DELETE action
		if normalizeAction(prodFK.OnDelete) != normalizeAction(devFK.OnDelete) {
			diff.OnDeleteDiffers = true
			diff.ProdOnDelete = prodFK.OnDelete
			diff.DevOnDelete = devFK.OnDelete
			hasDiff = true
		}

		// Compare ON UPDATE action
		if normalizeAction(prodFK.OnUpdate) != normalizeAction(devFK.OnUpdate) {
			diff.OnUpdateDiffers = true
			diff.ProdOnUpdate = prodFK.OnUpdate
			diff.DevOnUpdate = devFK.OnUpdate
			hasDiff = true
		}

		if hasDiff {
			modified = append(modified, diff)
		}
	}

	return
}

// normalizeAction normalizes FK action strings for comparison.
// Handles variations like "NO ACTION" vs "RESTRICT" (which are equivalent in some databases).
func normalizeAction(action string) string {
	action = strings.TrimSpace(strings.ToUpper(action))
	// Treat empty string and "NO ACTION" as equivalent
	if action == "" || action == "NO ACTION" {
		return "NO ACTION"
	}
	return action
}

// diffPrimaryKey compares primary key columns between prod and dev.
// Returns a PrimaryKeyDiff if columns differ, nil otherwise.
func diffPrimaryKey(prodPK, devPK []string) *PrimaryKeyDiff {
	if slicesEqual(prodPK, devPK) {
		return nil
	}
	return &PrimaryKeyDiff{
		ProdColumns: prodPK,
		DevColumns:  devPK,
	}
}
