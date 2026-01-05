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
	"github.com/iamvirul/deepdiff-db/pkg/logger"
	"github.com/iamvirul/deepdiff-db/pkg/progress"
)

const (
	// updateBatchSize is the number of rows to batch together in a single UPDATE statement
	updateBatchSize = 1000
	// progressLogThreshold is the minimum number of rows before progress logging is enabled
	progressLogThreshold = 10000
	// queryPageSize is the number of rows to fetch per database query to avoid memory issues on large tables
	queryPageSize = 10000
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
	// Get logger and progress manager from context
	log := logger.FromContext(ctx).WithOperation("generate_pack")
	progressMgr := progress.FromContext(ctx)

	// Count total changes for logging and progress
	totalChanges := 0
	totalOperations := 0
	for _, td := range diff.Tables {
		totalChanges += len(td.Added) + len(td.Removed) + len(td.Updated)
		// Count operations: updated rows require 2 operations (delete + insert)
		totalOperations += len(td.Added) + len(td.Removed) + (len(td.Updated) * 2)
	}
	log.Info("generating migration pack",
		"tables_with_changes", len(diff.Tables),
		"total_row_changes", totalChanges)

	// Create progress bar if we have many operations (threshold: 10,000)
	const progressThreshold = 10000
	var bar *progress.Bar
	if progressMgr != nil && totalOperations >= progressThreshold {
		bar = progressMgr.StartBar(ctx, "Generating pack", int64(totalOperations))
		defer bar.Finish()
	}

	var stmts []string
	stmts = append(stmts, "BEGIN;")

	// Disable foreign key checks for MySQL to allow out-of-order operations
	if prodDriver == "mysql" {
		stmts = append(stmts, "SET FOREIGN_KEY_CHECKS = 0;")
		log.Debug("disabled foreign key checks for MySQL")
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
			if bar != nil {
				bar.Add(1)
			}
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
			if bar != nil {
				bar.Add(1)
			}
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
		
		updateStmts, err := generateUpdateStatementsForNewColumns(ctx, log, devDB, prodDriver, tableName, devTbl, columnsToAdd, rowsToSkipUpdate, ignoreFn)
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
		log.Error("failed to write migration pack", logger.FieldError, err.Error(), "path", packPath)
		return "", err
	}

	log.Info("migration pack generated successfully",
		"path", packPath,
		"total_statements", len(stmts))

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
// For PostgreSQL, the database parameter is used as the schema name. If empty, the function
// queries information_schema to find the table's schema, defaulting to "public" if not found.
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
		// For PostgreSQL, determine the schema to use
		// Priority: 1) explicit database parameter (used as schema), 2) query table's actual schema, 3) default to "public"
		var schemaName string
		if database != "" {
			schemaName = database
		} else {
			// Query to find which schema contains this table
			err := devDB.QueryRowContext(ctx, `
				SELECT table_schema
				FROM information_schema.tables
				WHERE table_name = $1
				ORDER BY table_schema = 'public' DESC, table_schema
				LIMIT 1
			`, tableName).Scan(&schemaName)
			if err != nil {
				return "", fmt.Errorf("could not determine schema for table %s: %w (hint: provide explicit schema via database parameter)", tableName, err)
			}
		}
		
		// Query information_schema to get full type definition using explicit schema
		// Note: Integer types (integer, bigint, smallint, serial types) have numeric_precision
		// but don't accept precision notation, so we exclude precision/scale for these types.
		var fullType string
		err := devDB.QueryRowContext(ctx, `
			SELECT 
				CASE 
					WHEN c.data_type = 'ARRAY' THEN c.udt_name || '[]'
					WHEN c.character_maximum_length IS NOT NULL THEN 
						c.udt_name || '(' || c.character_maximum_length || ')'
					WHEN c.numeric_precision IS NOT NULL AND c.numeric_scale IS NOT NULL 
						AND c.data_type NOT IN ('integer', 'bigint', 'smallint', 'serial', 'bigserial', 'smallserial') THEN
						c.udt_name || '(' || c.numeric_precision || ',' || c.numeric_scale || ')'
					WHEN c.numeric_precision IS NOT NULL 
						AND c.data_type NOT IN ('integer', 'bigint', 'smallint', 'serial', 'bigserial', 'smallserial') THEN
						c.udt_name || '(' || c.numeric_precision || ')'
					ELSE c.udt_name
				END AS full_type
			FROM information_schema.columns c
			WHERE c.table_schema = $1
			  AND c.table_name = $2
			  AND c.column_name = $3
		`, schemaName, tableName, columnName).Scan(&fullType)
		if err != nil {
			return "", fmt.Errorf("query column type for schema %s, table %s, column %s: %w", schemaName, tableName, columnName, err)
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

// generateUpdateStatementsForNewColumns generates batched UPDATE statements to set values for newly added columns
// from the dev database. It uses CASE expressions to batch multiple rows into single UPDATE statements for efficiency.
// Rows in skipKeys will be skipped (they'll be updated via DELETE/INSERT).
func generateUpdateStatementsForNewColumns(ctx context.Context, log *logger.Logger, devDB *sql.DB, driver, tableName string, devTbl schema.Table, newColumns []string, skipKeys map[string]bool, ignoreFn func(table, column string) bool) ([]string, error) {
	log = log.WithTable(tableName)
	if len(devTbl.PrimaryKey) == 0 {
		return nil, fmt.Errorf("table %s lacks primary key", tableName)
	}
	
	// Get all rows from dev database for this table
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
	
	// Build index map for column positions
	colIndex := make(map[string]int)
	for i, col := range allCols {
		colIndex[col] = i
	}
	
	// Batch data structures
	type batchRow struct {
		key      string
		pkValues []any
		values   map[string]any // column name -> value
	}
	
	var batch []batchRow
	var stmts []string
	rowCount := 0
	totalRows := 0
	
	// Helper to build cursor-based WHERE clause for pagination
	buildCursorWhere := func(lastPKValues []any) string {
		if len(lastPKValues) == 0 {
			return ""
		}
		// For composite keys, use lexicographic comparison: (pk1 > v1) OR (pk1 = v1 AND pk2 > v2) OR ...
		// For single key, just: pk1 > v1
		var conditions []string
		for i := 0; i < len(devTbl.PrimaryKey); i++ {
			var parts []string
			for j := 0; j < i; j++ {
				parts = append(parts, fmt.Sprintf("%s = %s", quoteIdent(driver, devTbl.PrimaryKey[j]), literal(lastPKValues[j])))
			}
			parts = append(parts, fmt.Sprintf("%s > %s", quoteIdent(driver, devTbl.PrimaryKey[i]), literal(lastPKValues[i])))
			conditions = append(conditions, "("+strings.Join(parts, " AND ")+")")
		}
		return "WHERE " + strings.Join(conditions, " OR ")
	}
	
	// Helper to flush current batch
	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		
		// Build CASE expressions for each new column
		var setParts []string
		for _, newCol := range newColumns {
			if _, exists := devTbl.Columns[newCol]; !exists {
				continue
			}
			
			// Build CASE expression: CASE WHEN pk_match THEN val1 WHEN pk_match2 THEN val2 ... ELSE col END
			var caseParts []string
			for _, row := range batch {
				// Build primary key match condition
				var pkConditions []string
				for i, pkCol := range devTbl.PrimaryKey {
					pkConditions = append(pkConditions, fmt.Sprintf("%s = %s", quoteIdent(driver, pkCol), literal(row.pkValues[i])))
				}
				pkMatch := strings.Join(pkConditions, " AND ")
				
				// Get value for this column
				val := row.values[newCol]
				caseParts = append(caseParts, fmt.Sprintf("WHEN %s THEN %s", pkMatch, literal(val)))
			}
			
			// Complete CASE expression: CASE ... ELSE col END
			caseExpr := fmt.Sprintf("CASE %s ELSE %s END", strings.Join(caseParts, " "), quoteIdent(driver, newCol))
			setParts = append(setParts, fmt.Sprintf("%s = %s", quoteIdent(driver, newCol), caseExpr))
		}
		
		if len(setParts) == 0 {
			return nil
		}
		
		// Build WHERE clause with all primary keys in batch
		var whereParts []string
		for _, row := range batch {
			var pkConditions []string
			for i, pkCol := range devTbl.PrimaryKey {
				pkConditions = append(pkConditions, fmt.Sprintf("%s = %s", quoteIdent(driver, pkCol), literal(row.pkValues[i])))
			}
			whereParts = append(whereParts, "("+strings.Join(pkConditions, " AND ")+")")
		}
		whereClause := strings.Join(whereParts, " OR ")
		
		// Generate batched UPDATE statement
		updateStmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s;",
			quoteIdent(driver, tableName),
			strings.Join(setParts, ", "),
			whereClause,
		)
		stmts = append(stmts, updateStmt)
		
		// Log progress for large datasets
		if totalRows >= progressLogThreshold {
			log.Debug("generated batch for update statements",
				"batch_number", len(stmts),
				"rows_processed", totalRows)
		}
		
		batch = batch[:0] // Clear batch but keep capacity
		rowCount = 0
		return nil
	}
	
	// Process rows using cursor-based pagination to avoid loading all rows into memory
	quotedCols := make([]string, len(allCols))
	for i, c := range allCols {
		quotedCols[i] = quoteIdent(driver, c)
	}
	quotedPK := make([]string, len(devTbl.PrimaryKey))
	for i, c := range devTbl.PrimaryKey {
		quotedPK[i] = quoteIdent(driver, c)
	}
	
	dest := make([]any, len(allCols))
	for i := range dest {
		var holder any
		dest[i] = &holder
	}
	
	var lastPKValues []any // Track last primary key for cursor-based pagination
	
	for {
		// Build query with cursor-based pagination
		baseQuery := fmt.Sprintf("SELECT %s FROM %s",
			strings.Join(quotedCols, ", "),
			quoteIdent(driver, tableName),
		)
		orderBy := " ORDER BY " + strings.Join(quotedPK, ", ")
		limit := fmt.Sprintf(" LIMIT %d", queryPageSize)
		
		var query string
		if len(lastPKValues) > 0 {
			cursorWhere := buildCursorWhere(lastPKValues)
			query = baseQuery + " " + cursorWhere + orderBy + limit
		} else {
			query = baseQuery + orderBy + limit
		}
		
		rows, err := devDB.QueryContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("query dev rows: %w", err)
		}
		
		pageRowCount := 0
		hasMoreRows := false
		
		for rows.Next() {
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan row: %w", err)
			}
			
			// Dereference values
			values := make([]any, len(allCols))
			for i, v := range dest {
				values[i] = *(v.(*any))
			}
			
			// Build primary key
			key, err := buildKey(allCols, values, devTbl.PrimaryKey)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("build key: %w", err)
			}
			
			// Skip rows that will be updated via DELETE/INSERT
			if skipKeys[key] {
				continue
			}
			
			// Extract PK values
			pkValues := make([]any, len(devTbl.PrimaryKey))
			for i, pkCol := range devTbl.PrimaryKey {
				idx := colIndex[pkCol]
				pkValues[i] = values[idx]
			}
			
			// Extract values for new columns
			rowValues := make(map[string]any)
			for _, newCol := range newColumns {
				idx, ok := colIndex[newCol]
				if !ok {
					continue
				}
				rowValues[newCol] = values[idx]
			}
			
			// Add to batch
			batch = append(batch, batchRow{
				key:      key,
				pkValues: pkValues,
				values:   rowValues,
			})
			rowCount++
			totalRows++
			pageRowCount++
			
			// Update last PK values for cursor
			lastPKValues = make([]any, len(pkValues))
			copy(lastPKValues, pkValues)
			
			// Flush batch when it reaches the batch size
			if rowCount >= updateBatchSize {
				if err := flushBatch(); err != nil {
					rows.Close()
					return nil, err
				}
			}
		}
		
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate rows: %w", err)
		}
		rows.Close()
		
		// If we got fewer rows than the page size, we've reached the end
		if pageRowCount < queryPageSize {
			break
		}
		
		// If we got exactly the page size, there might be more rows
		hasMoreRows = pageRowCount == queryPageSize
		if !hasMoreRows {
			break
		}
	}
	
	// Flush remaining batch
	if err := flushBatch(); err != nil {
		return nil, err
	}
	
	// Final progress log if we processed many rows
	if totalRows >= progressLogThreshold {
		log.Info("completed update statements for new columns",
			logger.FieldRowCount, totalRows,
			"batches", len(stmts))
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
