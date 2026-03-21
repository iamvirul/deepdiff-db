// Package resolve provides the core resolution engine for applying conflict
// resolution strategies to detected conflicts between production and development databases.
package resolve

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// RowData represents a single row's column values for display.
type RowData struct {
	// Columns is the ordered list of column names.
	Columns []string
	// Values maps column names to their values.
	Values map[string]any
}

// ColumnDiff represents a difference in a single column between prod and dev.
type ColumnDiff struct {
	Column  string
	ProdVal any
	DevVal  any
	Differs bool
}

// FetchConflictRows fetches both production and development row data for a conflict.
// Returns nil for a row if it doesn't exist in that database.
func FetchConflictRows(
	ctx context.Context,
	prodDB, devDB *sql.DB,
	driver string,
	prodSchema, devSchema *schema.Schema,
	conflict content.Conflict,
) (prod, dev *RowData, err error) {
	// Get table schema from both databases
	prodTable, prodExists := prodSchema.Tables[conflict.Table]
	devTable, devExists := devSchema.Tables[conflict.Table]

	if !prodExists && !devExists {
		return nil, nil, fmt.Errorf("table %s not found in either database", conflict.Table)
	}

	// Use dev table schema as reference (it should have all columns)
	var refTable schema.Table
	if devExists {
		refTable = devTable
	} else {
		refTable = prodTable
	}

	// Get ordered column list
	columns := getOrderedColumns(refTable)

	// Fetch from production
	if prodExists {
		prod, err = fetchRowData(ctx, prodDB, driver, prodTable, columns, conflict.Key)
		if err != nil && !isNoRowsError(err) {
			return nil, nil, fmt.Errorf("fetch prod row: %w", err)
		}
	}

	// Fetch from development
	if devExists {
		dev, err = fetchRowData(ctx, devDB, driver, devTable, columns, conflict.Key)
		if err != nil && !isNoRowsError(err) {
			return nil, nil, fmt.Errorf("fetch dev row: %w", err)
		}
	}

	return prod, dev, nil
}

// fetchRowData fetches a single row from the database.
func fetchRowData(
	ctx context.Context,
	db *sql.DB,
	driver string,
	table schema.Table,
	columns []string,
	key string,
) (*RowData, error) {
	pk := table.PrimaryKey
	if len(pk) == 0 {
		return nil, fmt.Errorf("table %s has no primary key", table.Name)
	}

	// Parse the composite key
	keyParts, err := splitKey(key, len(pk))
	if err != nil {
		return nil, err
	}

	// Build WHERE clause
	whereParts := make([]string, len(pk))
	for i, col := range pk {
		whereParts[i] = fmt.Sprintf("%s = %s", quoteIdent(driver, col), quoteLiteral(keyParts[i]))
	}

	// Build query
	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = quoteIdent(driver, col)
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", // #nosec G201 -- identifiers sourced from schema introspection and passed through quoteIdent; values are parameterised separately
		strings.Join(quotedCols, ", "),
		quoteIdent(driver, table.Name),
		strings.Join(whereParts, " AND "),
	)

	// Execute query
	row := db.QueryRowContext(ctx, query)
	dest := make([]any, len(columns))
	for i := range dest {
		var holder any
		dest[i] = &holder
	}

	if err := row.Scan(dest...); err != nil {
		return nil, err
	}

	// Build result
	values := make(map[string]any)
	for i, col := range columns {
		values[col] = *(dest[i].(*any))
	}

	return &RowData{
		Columns: columns,
		Values:  values,
	}, nil
}

// CompareRows compares two rows and returns the differences.
// Returns a list of column differences, with Differs=true for columns that don't match.
func CompareRows(prod, dev *RowData) []ColumnDiff {
	if prod == nil && dev == nil {
		return nil
	}

	// Collect all columns from both rows
	colSet := make(map[string]bool)
	if prod != nil {
		for _, col := range prod.Columns {
			colSet[col] = true
		}
	}
	if dev != nil {
		for _, col := range dev.Columns {
			colSet[col] = true
		}
	}

	// Sort columns for consistent output
	columns := make([]string, 0, len(colSet))
	for col := range colSet {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	// Compare each column
	diffs := make([]ColumnDiff, 0, len(columns))
	for _, col := range columns {
		var prodVal, devVal any
		if prod != nil {
			prodVal = prod.Values[col]
		}
		if dev != nil {
			devVal = dev.Values[col]
		}

		differs := !valuesEqual(prodVal, devVal)
		diffs = append(diffs, ColumnDiff{
			Column:  col,
			ProdVal: prodVal,
			DevVal:  devVal,
			Differs: differs,
		})
	}

	return diffs
}

// getOrderedColumns returns column names in a consistent order.
// Primary key columns come first, followed by other columns sorted alphabetically.
func getOrderedColumns(table schema.Table) []string {
	pkSet := make(map[string]bool)
	for _, pk := range table.PrimaryKey {
		pkSet[pk] = true
	}

	// Start with primary key columns
	columns := make([]string, 0, len(table.Columns))
	columns = append(columns, table.PrimaryKey...)

	// Add non-PK columns sorted alphabetically
	nonPK := make([]string, 0)
	for name := range table.Columns {
		if !pkSet[name] {
			nonPK = append(nonPK, name)
		}
	}
	sort.Strings(nonPK)
	columns = append(columns, nonPK...)

	return columns
}

// splitKey splits a composite key string on the '|' separator.
func splitKey(key string, expected int) ([]string, error) {
	parts := strings.Split(key, "|")
	if len(parts) != expected {
		return nil, fmt.Errorf("unexpected key format %q: expected %d parts, got %d", key, expected, len(parts))
	}
	return parts, nil
}

// quoteIdent quotes an identifier for the specified SQL driver.
func quoteIdent(driver, ident string) string {
	switch driver {
	case "mysql":
		return "`" + ident + "`"
	case "postgres", "postgresql":
		return `"` + ident + `"`
	default:
		return ident
	}
}

// quoteLiteral quotes a string value for use in SQL.
func quoteLiteral(s string) string {
	escaped := strings.ReplaceAll(s, "'", "''")
	return "'" + escaped + "'"
}

// isNoRowsError checks if an error is sql.ErrNoRows.
func isNoRowsError(err error) bool {
	return err == sql.ErrNoRows
}

// valuesEqual compares two values for equality.
// Handles byte slices and nil values.
func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle byte slices
	aBytes, aOK := a.([]byte)
	bBytes, bOK := b.([]byte)
	if aOK && bOK {
		return string(aBytes) == string(bBytes)
	}
	if aOK {
		return string(aBytes) == fmt.Sprintf("%v", b)
	}
	if bOK {
		return fmt.Sprintf("%v", a) == string(bBytes)
	}

	// Default comparison
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// FormatValue formats a value for display.
func FormatValue(v any) string {
	if v == nil {
		return "NULL"
	}

	switch val := v.(type) {
	case []byte:
		return string(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}
