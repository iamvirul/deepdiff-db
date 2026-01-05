package content

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/iamvirul/deepdiff-db/internal/schema"
	"github.com/iamvirul/deepdiff-db/pkg/logger"
	"github.com/iamvirul/deepdiff-db/pkg/progress"
)

// HashTable streams table rows and returns a map of PK composite key -> row hash.
// HashTable streams all rows of the specified table and returns a map keyed by the table's composite
// primary key to a SHA-256 hash of each row's column values.
//
// HashTable requires the table to have a primary key; primary key columns are always included first
// (in the order specified by tbl.PrimaryKey). The optional ignoreFn may exclude non-primary-key
// columns; if no columns remain to hash after applying ignoreFn, HashTable returns an error.
// The SELECT is constructed with driver-appropriate identifier quoting and ordered by the primary key.
// The returned map uses a composite key formed by joining primary key column values with "|" and maps
// that key to the hex-encoded SHA-256 hash of the row's "col=value" representation.
//
// On failure HashTable returns an error for missing primary keys, no selectable columns, query or scan
// errors, iteration errors, or when a primary key column is not present among the selected columns.
func HashTable(ctx context.Context, db *sql.DB, driver string, tbl schema.Table, ignoreFn func(table, column string) bool) (map[string]string, error) {
	// Get logger and progress manager from context
	log := logger.FromContext(ctx).WithTable(tbl.Name)
	progressMgr := progress.FromContext(ctx)

	if len(tbl.PrimaryKey) == 0 {
		return nil, fmt.Errorf("table %s has no primary key", tbl.Name)
	}

	cols := orderedColumns(tbl, ignoreFn)
	if len(cols) == 0 {
		return nil, fmt.Errorf("no columns to hash for table %s", tbl.Name)
	}

	// Get row count for progress bar
	rowCount, err := getRowCount(ctx, db, driver, tbl.Name)
	if err != nil {
		log.Warn("could not get row count for progress tracking", logger.FieldError, err.Error())
		rowCount = 0
	}

	log.Debug("starting table hash", logger.FieldRowCount, rowCount, "columns", len(cols))

	query := buildSelect(driver, tbl.Name, cols, tbl.PrimaryKey)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query rows: %w", err)
	}
	defer rows.Close()

	dest := make([]any, len(cols))
	for i := range dest {
		var holder any
		dest[i] = &holder
	}

	result := make(map[string]string)

	// Create progress bar if row count is significant (threshold: 10,000)
	const progressThreshold = 10000
	var bar *progress.Bar
	if progressMgr != nil && rowCount >= progressThreshold {
		bar = progressMgr.StartBar(ctx, fmt.Sprintf("Hashing %s", tbl.Name), rowCount)
		defer bar.Finish()
	}

	rowsProcessed := int64(0)
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		// deref values
		values := make([]any, len(cols))
		for i, v := range dest {
			values[i] = *(v.(*any))
		}

		key, err := buildKey(cols, values, tbl.PrimaryKey)
		if err != nil {
			return nil, err
		}

		hash := hashRow(cols, values)
		result[key] = hash

		rowsProcessed++
		if bar != nil {
			bar.Add(1)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	log.Info("table hashing complete",
		logger.FieldRowCount, rowsProcessed,
		"unique_keys", len(result))

	return result, nil
}

// getRowCount returns the approximate row count for a table.
// Returns 0 if count cannot be determined.
func getRowCount(ctx context.Context, db *sql.DB, driver, table string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteIdent(driver, table))
	var count int64
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// orderedColumns returns an ordered list of column names for hashing.
// Primary key columns from tbl.PrimaryKey are always included first in their original order.
// Non-primary columns from tbl.Columns follow, sorted alphabetically, excluding any for which
// ignoreFn(tbl.Name, column) returns true. If ignoreFn is nil, no non-primary columns are excluded.
func orderedColumns(tbl schema.Table, ignoreFn func(table, column string) bool) []string {
	var cols []string
	// Ensure primary keys first (and always included)
	cols = append(cols, tbl.PrimaryKey...)
	var nonPK []string
	for name := range tbl.Columns {
		if contains(tbl.PrimaryKey, name) {
			continue
		}
		if ignoreFn != nil && ignoreFn(tbl.Name, name) {
			continue
		}
		nonPK = append(nonPK, name)
	}
	sort.Strings(nonPK)
	cols = append(cols, nonPK...)
	return cols
}

// contains reports whether the slice contains the given string.
func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// buildSelect constructs a SELECT query for the given table that selects the provided columns and orders results by the given primary key columns.
// Identifiers are quoted using driver-specific quoting; the result has the form: "SELECT <cols> FROM <table> ORDER BY <pk>".
func buildSelect(driver, table string, cols []string, pk []string) string {
	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = quoteIdent(driver, c)
	}
	quotedPK := make([]string, len(pk))
	for i, c := range pk {
		quotedPK[i] = quoteIdent(driver, c)
	}
	return fmt.Sprintf("SELECT %s FROM %s ORDER BY %s",
		strings.Join(quotedCols, ", "),
		quoteIdent(driver, table),
		strings.Join(quotedPK, ", "),
	)
}

// quoteIdent returns the SQL identifier quoted according to the target driver.
// For "mysql" it wraps the identifier in backticks (`ident`); for "postgres" and
// "postgresql" it wraps the identifier in double quotes ("ident"); for any other
// driver it returns the identifier unchanged.
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

// buildKey builds a composite key string from the values of the specified primary key columns by joining them with "|".
// cols is the ordered list of selected column names, values holds the scanned row values corresponding to cols, and pk lists the primary key column names in their key order.
// It returns the composite key or an error if any primary key column is not present among cols.
func buildKey(cols []string, values []any, pk []string) (string, error) {
	index := make(map[string]int, len(cols))
	for i, c := range cols {
		index[c] = i
	}
	var parts []string
	for _, k := range pk {
		pos, ok := index[k]
		if !ok {
			return "", fmt.Errorf("primary key column %s not selected", k)
		}
		parts = append(parts, fmt.Sprint(values[pos]))
	}
	return strings.Join(parts, "|"), nil
}

// hashRow computes the SHA-256 hash of a row and returns it as a hex-encoded string.
// It produces a deterministic text representation by writing each column name followed by `=` and its value on separate lines, then hashing that representation.
func hashRow(cols []string, values []any) string {
	var b strings.Builder
	for i, col := range cols {
		fmt.Fprintf(&b, "%s=%v\n", col, values[i])
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}