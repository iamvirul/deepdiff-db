package content

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// GeneratePack builds a SQL migration pack for applying data diffs to prod.
// GeneratePack builds a SQL migration script that applies the provided data diff to a production database,
// using the development database as the source of truth for inserted and updated rows.
// 
// The generated script contains transactional statements and, for MySQL, temporarily disables and
// re-enables foreign key checks. For each table with changes, rows identified by primary keys are
// deleted (for removed and updated entries) and inserted (for added and updated entries) with column
// values taken from the development database. The script is written to outDir/migration_pack.sql and
// the returned string is the written file path.
// 
// Errors are returned if a table lacks a primary key, if row fetching or WHERE-clause construction fails,
// or if writing the output file fails.
func GeneratePack(ctx context.Context, prodDriver string, devDB *sql.DB, devSchema *schema.Schema, diff DataDiff, ignoreFn func(table, column string) bool, outDir string) (string, error) {
	var stmts []string
	stmts = append(stmts, "BEGIN;")
	
	// Disable foreign key checks for MySQL to allow out-of-order operations
	if prodDriver == "mysql" {
		stmts = append(stmts, "SET FOREIGN_KEY_CHECKS = 0;")
	}

	for _, td := range diff.Tables {
		if len(td.Added)+len(td.Removed)+len(td.Updated) == 0 {
			continue
		}
		tbl, ok := devSchema.Tables[td.Table]
		if !ok {
			continue
		}
		pk := tbl.PrimaryKey
		if len(pk) == 0 {
			return "", fmt.Errorf("table %s lacks primary key; cannot build pack", tbl.Name)
		}
		cols := orderedColumns(tbl, ignoreFn)

		// Deletes for removed/updated
		for _, key := range append(td.Removed, td.Updated...) {
			where, err := keyToWhere(prodDriver, pk, key)
			if err != nil {
				return "", fmt.Errorf("table %s delete where: %w", tbl.Name, err)
			}
			stmts = append(stmts, fmt.Sprintf("DELETE FROM %s WHERE %s;", quoteIdent(prodDriver, tbl.Name), where))
		}

		// Inserts (for added) and re-inserts (for updated)
		for _, key := range append(td.Added, td.Updated...) {
			rowVals, err := fetchRow(ctx, devDB, prodDriver, tbl, cols, pk, key)
			if err != nil {
				return "", fmt.Errorf("fetch row %s.%s: %w", tbl.Name, key, err)
			}
			valLiterals := make([]string, len(cols))
			for i, v := range rowVals {
				valLiterals[i] = literal(v)
			}
			colsQuoted := make([]string, len(cols))
			for i, c := range cols {
				colsQuoted[i] = quoteIdent(prodDriver, c)
			}
			stmts = append(stmts, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
				quoteIdent(prodDriver, tbl.Name),
				strings.Join(colsQuoted, ", "),
				strings.Join(valLiterals, ", "),
			))
		}
	}

	// Re-enable foreign key checks for MySQL
	if prodDriver == "mysql" {
		stmts = append(stmts, "SET FOREIGN_KEY_CHECKS = 1;")
	}

	stmts = append(stmts, "COMMIT;")

	packPath := filepath.Join(outDir, "migration_pack.sql")
	if err := writeFile(packPath, strings.Join(stmts, "\n")); err != nil {
		return "", err
	}
	return packPath, nil
}

// fetchRow retrieves the row from tbl identified by the composite primary key encoded in key
// and returns the column values for cols in the same order.
//
// The key is a '|'‑separated composite key whose parts must match the order and count of pk.
// Returns an error if the key cannot be split, the query fails, or scanning the result fails.
func fetchRow(ctx context.Context, db *sql.DB, driver string, tbl schema.Table, cols, pk []string, key string) ([]any, error) {
	keyParts, err := splitKey(key, len(pk))
	if err != nil {
		return nil, err
	}

	where := make([]string, len(pk))
	for i, col := range pk {
		where[i] = fmt.Sprintf("%s = %s", quoteIdent(driver, col), literal(keyParts[i]))
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s",
		strings.Join(quoteIdents(driver, cols), ", "),
		quoteIdent(driver, tbl.Name),
		strings.Join(where, " AND "),
	)

	row := db.QueryRowContext(ctx, query)
	dest := make([]any, len(cols))
	for i := range dest {
		var holder any
		dest[i] = &holder
	}
	if err := row.Scan(dest...); err != nil {
		return nil, fmt.Errorf("scan row: %w", err)
	}
	values := make([]any, len(cols))
	for i, v := range dest {
		values[i] = *(v.(*any))
	}
	return values, nil
}

// splitKey splits a composite key string on the '|' separator and returns the parts.
// It returns an error if the resulting number of parts does not match expected.
func splitKey(key string, expected int) ([]string, error) {
	parts := strings.Split(key, "|")
	if len(parts) != expected {
		return nil, fmt.Errorf("unexpected key format %q", key)
	}
	return parts, nil
}

// keyToWhere converts a composite primary key string into a SQL WHERE clause.
// It returns a string of equality comparisons for each primary key column to
// its corresponding value, joined with " AND ". Column identifiers are quoted
// for the specified driver and values are formatted as SQL literals. If the
// composite key does not contain the expected number of parts, an error is returned.
func keyToWhere(driver string, pk []string, key string) (string, error) {
	parts, err := splitKey(key, len(pk))
	if err != nil {
		return "", err
	}
	clauses := make([]string, len(pk))
	for i, col := range pk {
		clauses[i] = fmt.Sprintf("%s = %s", quoteIdent(driver, col), literal(parts[i]))
	}
	return strings.Join(clauses, " AND "), nil
}

// quoteIdents returns a slice of column identifiers quoted for the specified SQL driver, preserving input order and length.
func quoteIdents(driver string, cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = quoteIdent(driver, c)
	}
	return out
}

// literal converts a Go value to an SQL literal suitable for embedding in a query.
// It returns:
//
// - "NULL" for nil.
// - "TRUE" or "FALSE" for booleans.
// - A quoted timestamp in the format 'YYYY-MM-DD HH:MM:SS' for time.Time values or
//   for strings/byte slices that successfully parse as known timestamp formats
//   (RFC3339, common MySQL/UTC layouts, and several common variants).
// - A single-quoted, SQL-escaped string for all other values (single quotes doubled).
//
// The function attempts multiple timestamp layouts when given strings or []byte;
// if parsing succeeds it formats the time as 'YYYY-MM-DD HH:MM:SS', otherwise it
// escapes and quotes the textual representation produced by fmt.Sprintf("%v", v).
func literal(v any) string {
	switch val := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		// Try to parse as timestamp first
		if t, err := time.Parse(time.RFC3339, string(val)); err == nil {
			return "'" + t.Format("2006-01-02 15:04:05") + "'"
		}
		// Try MySQL timestamp format
		if t, err := time.Parse("2006-01-02 15:04:05", string(val)); err == nil {
			return "'" + t.Format("2006-01-02 15:04:05") + "'"
		}
		return "'" + escape(string(val)) + "'"
	case string:
		// Try to parse as timestamp first
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return "'" + t.Format("2006-01-02 15:04:05") + "'"
		}
		// Try MySQL timestamp format
		if t, err := time.Parse("2006-01-02 15:04:05", val); err == nil {
			return "'" + t.Format("2006-01-02 15:04:05") + "'"
		}
		// Try to parse common timestamp formats
		for _, layout := range []string{
			"2006-01-02 15:04:05.000000",
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05",
		} {
			if t, err := time.Parse(layout, val); err == nil {
				return "'" + t.Format("2006-01-02 15:04:05") + "'"
			}
		}
		return "'" + escape(val) + "'"
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	case time.Time:
		// Format timestamp for SQL: 'YYYY-MM-DD HH:MM:SS'
		return "'" + val.Format("2006-01-02 15:04:05") + "'"
	default:
		// Check if it's a time.Time wrapped in interface{}
		if t, ok := val.(time.Time); ok {
			return "'" + t.Format("2006-01-02 15:04:05") + "'"
		}
		// For other types, format as string and escape if needed
		str := fmt.Sprintf("%v", val)
		// If it looks like a timestamp string, try to parse it
		if strings.Contains(str, ":") && (strings.Contains(str, "-") || strings.Contains(str, "/")) {
			for _, layout := range []string{
				time.RFC3339,
				"2006-01-02 15:04:05.000000",
				"2006-01-02T15:04:05Z07:00",
				"2006-01-02 15:04:05",
			} {
				if t, err := time.Parse(layout, str); err == nil {
					return "'" + t.Format("2006-01-02 15:04:05") + "'"
				}
			}
		}
		return "'" + escape(str) + "'"
	}
}

// escape returns s with each single quote doubled so it can be embedded in a SQL string literal.
// It performs only the quote-doubling and does not add surrounding quotes.
func escape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// writeFile writes content to the file at path using file mode 0644.
// It returns any error encountered while writing the file.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}