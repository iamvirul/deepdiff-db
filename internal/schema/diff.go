package schema

import (
	"sort"
	"strings"
)

// TableDiff describes differences for a single table.
type TableDiff struct {
	Table          string       `json:"table"`
	MissingInProd  bool         `json:"missing_in_prod,omitempty"`
	MissingInDev   bool         `json:"missing_in_dev,omitempty"`
	ColumnDiffs    []ColumnDiff `json:"column_diffs,omitempty"`
	ExtraInProd    []string     `json:"extra_in_prod,omitempty"`
	ExtraInDev     []string     `json:"extra_in_dev,omitempty"`
	OnlyInProd     bool         `json:"only_in_prod,omitempty"`
	OnlyInDev      bool         `json:"only_in_dev,omitempty"`
	HasDifferences bool         `json:"has_differences"`
}

// ColumnDiff captures mismatches for a column across prod/dev.
type ColumnDiff struct {
	Column          string `json:"column"`
	MissingInProd   bool   `json:"missing_in_prod,omitempty"`
	MissingInDev    bool   `json:"missing_in_dev,omitempty"`
	TypeMismatch    bool   `json:"type_mismatch,omitempty"`
	ProdType        string `json:"prod_type,omitempty"`
	DevType         string `json:"dev_type,omitempty"`
	NullableMismatch bool  `json:"nullable_mismatch,omitempty"`
	ProdNullable     *bool `json:"prod_nullable,omitempty"`
	DevNullable      *bool `json:"dev_nullable,omitempty"`
}

// DiffResult aggregates all table diffs.
type DiffResult struct {
	Tables []TableDiff `json:"tables"`
}

// DiffSchemas compares two Schema values and returns a DiffResult that describes table- and column-level differences between them.
// It aggregates table names from both schemas, processes tables in sorted order, marks tables present only on one side (setting MissingInProd/MissingInDev and OnlyInProd/OnlyInDev) and computes column-level differences for tables present in both, setting HasDifferences when any mismatches are found.
func DiffSchemas(prod, dev *Schema) DiffResult {
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

		td := TableDiff{Table: tbl}
		if prodOK && !devOK {
			td.MissingInDev = true
			td.OnlyInProd = true
			td.HasDifferences = true
			result.Tables = append(result.Tables, td)
			continue
		}
		if !prodOK && devOK {
			td.MissingInProd = true
			td.OnlyInDev = true
			td.HasDifferences = true
			result.Tables = append(result.Tables, td)
			continue
		}

		td.ColumnDiffs = diffColumns(p.Columns, d.Columns)
		td.HasDifferences = len(td.ColumnDiffs) > 0
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

		if stringsDiffer(p.DataType, d.DataType) {
			cd.TypeMismatch = true
			cd.ProdType = p.DataType
			cd.DevType = d.DataType
		}
		if p.IsNullable != d.IsNullable {
			cd.NullableMismatch = true
			cd.ProdNullable = &p.IsNullable
			cd.DevNullable = &d.IsNullable
		}

		if cd.TypeMismatch || cd.NullableMismatch {
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
