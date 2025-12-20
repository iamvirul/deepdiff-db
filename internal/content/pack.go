package content

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// GeneratePack builds a SQL migration pack for applying data diffs to prod.
// GeneratePack builds a SQL migration script that applies the provided data diff to a production database,
// using the development database as the source of truth for inserted and updated rows.
// 
// The generated script contains:
// 1. ALTER TABLE statements to add columns missing in prod (from schemaDiff)
// 2. Transactional statements and, for MySQL, temporarily disables and re-enables foreign key checks
// 3. For each table with changes, rows identified by primary keys are deleted (for removed and updated entries)
//    and inserted (for added and updated entries) with column values taken from the development database.
// Only columns that exist in both prod and dev schemas are included in INSERT statements to handle schema drift.
// The script is written to outDir/migration_pack.sql and the returned string is the written file path.
// 
// Errors are returned if a table lacks a primary key, if row fetching or WHERE-clause construction fails,
// or if writing the output file fails.
func GeneratePack(ctx context.Context, prodDriver string, devDB *sql.DB, devDatabase string, prodSchema, devSchema *schema.Schema, schemaDiff schema.DiffResult, diff DataDiff, ignoreFn func(table, column string) bool, outDir string) (string, error) {
	var stmts []string
	stmts = append(stmts, "BEGIN;")
	
	// Disable foreign key checks for MySQL to allow out-of-order operations
	if prodDriver == "mysql" {
		stmts = append(stmts, "SET FOREIGN_KEY_CHECKS = 0;")
	}

	// Generate ALTER TABLE statements for columns missing in prod
	for _, td := range schemaDiff.Tables {
		if !td.HasDifferences {
			continue
		}
		// Only process tables that exist in both schemas (skip tables that only exist in dev)
		_, prodOK := prodSchema.Tables[td.Table]
		devTbl, devOK := devSchema.Tables[td.Table]
		if !prodOK || !devOK {
			continue
		}

		// Track columns that need to be added
		var columnsToAdd []string
		
		// Add ALTER TABLE statements for columns missing in prod
		for _, cd := range td.ColumnDiffs {
			if cd.MissingInProd {
				// Skip if column should be ignored
				if ignoreFn != nil && ignoreFn(td.Table, cd.Column) {
					continue
				}
				// Get column definition from dev schema
				devCol, ok := devTbl.Columns[cd.Column]
				if !ok {
					continue
				}
				// Get full column type definition from dev database (includes length, precision, etc.)
				fullType, err := getFullColumnType(ctx, devDB, prodDriver, devDatabase, td.Table, cd.Column)
				if err != nil {
					return "", fmt.Errorf("get column type for %s.%s: %w", td.Table, cd.Column, err)
				}
				// Build ALTER TABLE ADD COLUMN statement with full type definition
				alterStmt := buildAlterTableAddColumn(prodDriver, td.Table, cd.Column, fullType, devCol.IsNullable)
				stmts = append(stmts, alterStmt)
				columnsToAdd = append(columnsToAdd, cd.Column)
			}
		}
	}

	// Track which tables need UPDATE statements for new columns (after INSERT)
	tableColumnsToUpdate := make(map[string][]string)
	for _, td := range schemaDiff.Tables {
		if !td.HasDifferences {
			continue
		}
		_, prodOK := prodSchema.Tables[td.Table]
		devTbl, devOK := devSchema.Tables[td.Table]
		if !prodOK || !devOK {
			continue
		}
		var columnsToAdd []string
		for _, cd := range td.ColumnDiffs {
			if cd.MissingInProd {
				if ignoreFn != nil && ignoreFn(td.Table, cd.Column) {
					continue
				}
				if _, exists := devTbl.Columns[cd.Column]; exists {
					columnsToAdd = append(columnsToAdd, cd.Column)
				}
			}
		}
		if len(columnsToAdd) > 0 {
			tableColumnsToUpdate[td.Table] = columnsToAdd
		}
	}

	for _, td := range diff.Tables {
		if len(td.Added)+len(td.Removed)+len(td.Updated) == 0 {
			continue
		}
		devTbl, ok := devSchema.Tables[td.Table]
		if !ok {
			continue
		}
		prodTbl, ok := prodSchema.Tables[td.Table]
		if !ok {
			// Skip tables that don't exist in prod
			continue
		}
		pk := devTbl.PrimaryKey
		if len(pk) == 0 {
			return "", fmt.Errorf("table %s lacks primary key; cannot build pack", devTbl.Name)
		}
		// Only include columns that exist in both schemas
		cols := orderedColumnsIntersection(devTbl, prodTbl, ignoreFn)

		// Deletes for removed/updated
		for _, key := range append(td.Removed, td.Updated...) {
			where, err := keyToWhere(prodDriver, pk, key)
			if err != nil {
				return "", fmt.Errorf("table %s delete where: %w", devTbl.Name, err)
			}
			stmts = append(stmts, fmt.Sprintf("DELETE FROM %s WHERE %s;", quoteIdent(prodDriver, devTbl.Name), where))
		}

		// Inserts (for added) and re-inserts (for updated)
		for _, key := range append(td.Added, td.Updated...) {
			rowVals, err := fetchRow(ctx, devDB, prodDriver, devTbl, cols, pk, key)
			if err != nil {
				return "", fmt.Errorf("fetch row %s.%s: %w", devTbl.Name, key, err)
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
				quoteIdent(prodDriver, devTbl.Name),
				strings.Join(colsQuoted, ", "),
				strings.Join(valLiterals, ", "),
			))
		}
	}

	// Generate UPDATE statements for new columns AFTER INSERT
	// (INSERT statements don't include new columns, so we need to update them after insertion)
	for tableName, columnsToAdd := range tableColumnsToUpdate {
		devTbl, ok := devSchema.Tables[tableName]
		if !ok {
			continue
		}
		// Track which rows to skip (only Removed rows, since they won't exist after deletion)
		rowsToSkipUpdate := make(map[string]bool)
		for _, dataDiffTable := range diff.Tables {
			if dataDiffTable.Table == tableName {
				// Only skip Removed rows (they won't exist after deletion)
				for _, key := range dataDiffTable.Removed {
					rowsToSkipUpdate[key] = true
				}
			}
		}
		
		updateStmts, err := generateUpdateStatementsForNewColumns(ctx, devDB, prodDriver, tableName, devTbl, columnsToAdd, rowsToSkipUpdate, ignoreFn)
		if err != nil {
			return "", fmt.Errorf("generate update statements for %s: %w", tableName, err)
		}
		stmts = append(stmts, updateStmts...)
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

// getFullColumnType queries the dev database to get the complete column type definition
// including length, precision, and scale. This preserves the exact column definition from dev.
func getFullColumnType(ctx context.Context, devDB *sql.DB, driver, database, tableName, columnName string) (string, error) {
	driver = strings.ToLower(driver)
	
	switch driver {
	case "mysql":
		// Query information_schema to get COLUMN_TYPE which includes full definition
		var fullType string
		err := devDB.QueryRowContext(ctx, `
			SELECT COLUMN_TYPE
			FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = ?
			  AND TABLE_NAME = ?
			  AND COLUMN_NAME = ?
		`, database, tableName, columnName).Scan(&fullType)
		if err != nil {
			return "", fmt.Errorf("query column type: %w", err)
		}
		return fullType, nil
		
	case "postgres", "postgresql":
		// Query information_schema to get full type definition
		var fullType string
		err := devDB.QueryRowContext(ctx, `
			SELECT 
				CASE 
					WHEN c.data_type = 'ARRAY' THEN c.udt_name || '[]'
					WHEN c.character_maximum_length IS NOT NULL THEN 
						c.udt_name || '(' || c.character_maximum_length || ')'
					WHEN c.numeric_precision IS NOT NULL AND c.numeric_scale IS NOT NULL THEN
						c.udt_name || '(' || c.numeric_precision || ',' || c.numeric_scale || ')'
					WHEN c.numeric_precision IS NOT NULL THEN
						c.udt_name || '(' || c.numeric_precision || ')'
					ELSE c.udt_name
				END AS full_type
			FROM information_schema.columns c
			WHERE c.table_schema = current_schema()
			  AND c.table_name = $1
			  AND c.column_name = $2
		`, tableName, columnName).Scan(&fullType)
		if err != nil {
			return "", fmt.Errorf("query column type: %w", err)
		}
		return fullType, nil
		
	case "sqlite":
		// SQLite PRAGMA table_info returns the full type definition
		// Note: PRAGMA must be called directly, not in a subquery
		rows, err := devDB.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", quoteIdent(driver, tableName)))
		if err != nil {
			return "", fmt.Errorf("query table info: %w", err)
		}
		defer rows.Close()
		
		for rows.Next() {
			var (
				cid       int
				name      string
				ctype     string
				notnull   int
				dfltValue any
				pk        int
			)
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				return "", fmt.Errorf("scan column info: %w", err)
			}
			if name == columnName {
				return ctype, nil
			}
		}
		if err := rows.Err(); err != nil {
			return "", fmt.Errorf("iterate columns: %w", err)
		}
		return "", fmt.Errorf("column %s not found in table %s", columnName, tableName)
		
	default:
		return "", fmt.Errorf("unsupported driver: %s", driver)
	}
}

// generateUpdateStatementsForNewColumns generates UPDATE statements to set values for newly added columns
// from the dev database. It updates existing rows (rows that exist in both prod and dev) with the
// values from dev for the newly added columns. Rows in skipKeys will be skipped (they'll be updated via DELETE/INSERT).
func generateUpdateStatementsForNewColumns(ctx context.Context, devDB *sql.DB, driver, tableName string, devTbl schema.Table, newColumns []string, skipKeys map[string]bool, ignoreFn func(table, column string) bool) ([]string, error) {
	if len(devTbl.PrimaryKey) == 0 {
		return nil, fmt.Errorf("table %s lacks primary key", tableName)
	}
	
	var stmts []string
	
	// Get all rows from dev database for this table
	// We need to update all existing rows in prod with the new column values from dev
	// Build column list: primary keys + new columns (to fetch values)
	allCols := append([]string{}, devTbl.PrimaryKey...)
	for _, newCol := range newColumns {
		// Check if column exists in dev table
		if _, exists := devTbl.Columns[newCol]; exists {
			allCols = append(allCols, newCol)
		}
	}
	
	if len(allCols) == 0 {
		return nil, fmt.Errorf("no columns to select for table %s", tableName)
	}
	
	// Build SELECT query using quoteIdent from hash.go (same package)
	quotedCols := make([]string, len(allCols))
	for i, c := range allCols {
		quotedCols[i] = quoteIdent(driver, c)
	}
	quotedPK := make([]string, len(devTbl.PrimaryKey))
	for i, c := range devTbl.PrimaryKey {
		quotedPK[i] = quoteIdent(driver, c)
	}
	query := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s",
		strings.Join(quotedCols, ", "),
		quoteIdent(driver, tableName),
		strings.Join(quotedPK, ", "),
	)
	rows, err := devDB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query dev rows: %w", err)
	}
	defer rows.Close()
	
	dest := make([]any, len(allCols))
	for i := range dest {
		var holder any
		dest[i] = &holder
	}
	
	// Build index map for column positions
	colIndex := make(map[string]int)
	for i, col := range allCols {
		colIndex[col] = i
	}
	
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		
		// Dereference values
		values := make([]any, len(allCols))
		for i, v := range dest {
			values[i] = *(v.(*any))
		}
		
		// Build primary key for WHERE clause using buildKey from hash.go
		key, err := buildKey(allCols, values, devTbl.PrimaryKey)
		if err != nil {
			return nil, fmt.Errorf("build key: %w", err)
		}
		
		// Skip rows that will be updated via DELETE/INSERT
		if skipKeys[key] {
			continue
		}
		
		// Build SET clause with only the new columns
		var setParts []string
		for _, newCol := range newColumns {
			idx, ok := colIndex[newCol]
			if !ok {
				continue
			}
			val := values[idx]
			// Skip NULL values - only update if there's an actual value
			if val != nil {
				setParts = append(setParts, fmt.Sprintf("%s = %s", quoteIdent(driver, newCol), literal(val)))
			}
		}
		
		if len(setParts) == 0 {
			continue
		}
		
		// Build WHERE clause from primary key
		where, err := keyToWhere(driver, devTbl.PrimaryKey, key)
		if err != nil {
			return nil, fmt.Errorf("build where clause: %w", err)
		}
		
		// Generate UPDATE statement
		updateStmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s;",
			quoteIdent(driver, tableName),
			strings.Join(setParts, ", "),
			where,
		)
		stmts = append(stmts, updateStmt)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	
	return stmts, nil
}

// buildAlterTableAddColumn generates an ALTER TABLE ADD COLUMN statement for the given column.
// It uses the full type definition from the dev database to preserve exact column definitions.
func buildAlterTableAddColumn(driver, tableName, columnName, fullType string, isNullable bool) string {
	colName := quoteIdent(driver, columnName)
	tableNameQuoted := quoteIdent(driver, tableName)
	
	// Build column definition with the full type from dev database
	colDef := colName + " " + fullType
	if !isNullable {
		colDef += " NOT NULL"
	}
	
	// Driver-specific handling
	switch driver {
	case "mysql":
		// MySQL: ALTER TABLE table ADD COLUMN col TYPE [NOT NULL]
		return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", tableNameQuoted, colDef)
	case "postgres", "postgresql":
		// PostgreSQL: ALTER TABLE table ADD COLUMN col TYPE [NOT NULL]
		return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", tableNameQuoted, colDef)
	case "sqlite":
		// SQLite: ALTER TABLE table ADD COLUMN col TYPE [NOT NULL]
		// Note: SQLite has limited ALTER TABLE support, but ADD COLUMN is supported
		return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", tableNameQuoted, colDef)
	default:
		// Generic SQL
		return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", tableNameQuoted, colDef)
	}
}

// orderedColumnsIntersection returns an ordered list of column names that exist in both devTbl and prodTbl.
// Primary key columns are always included first in their original order.
// Non-primary columns that exist in both tables follow, sorted alphabetically, excluding any for which
// ignoreFn(table, column) returns true. If ignoreFn is nil, no non-primary columns are excluded.
func orderedColumnsIntersection(devTbl, prodTbl schema.Table, ignoreFn func(table, column string) bool) []string {
	var cols []string
	// Ensure primary keys first (and always included)
	cols = append(cols, devTbl.PrimaryKey...)
	
	var nonPK []string
	// Only include columns that exist in both schemas
	for name := range devTbl.Columns {
		// Skip primary key columns (already added)
		if contains(devTbl.PrimaryKey, name) {
			continue
		}
		// Skip if column doesn't exist in prod schema
		if _, exists := prodTbl.Columns[name]; !exists {
			continue
		}
		// Skip if ignored
		if ignoreFn != nil && ignoreFn(devTbl.Name, name) {
			continue
		}
		nonPK = append(nonPK, name)
	}
	sort.Strings(nonPK)
	cols = append(cols, nonPK...)
	return cols
}

// writeFile writes content to the file at path using file mode 0644.
// It returns any error encountered while writing the file.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
