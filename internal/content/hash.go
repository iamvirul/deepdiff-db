package content

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/iamvirul/deepdiff-db/internal/checkpoint"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	"github.com/iamvirul/deepdiff-db/pkg/logger"
	"github.com/iamvirul/deepdiff-db/pkg/progress"
)

// HashTable streams table rows and returns a map of PK composite key -> row hash.
//
// When batchSize > 0 the table is processed using keyset-paginated queries
// (LIMIT batchSize per round-trip). This bounds per-batch heap allocations so
// the Go GC can reclaim intermediate row data between pages. The accumulated
// result map is still O(n) in the number of rows, which is unavoidable for an
// in-memory diff; for tables that genuinely exhaust available memory consider
// reducing batchSize or increasing the host's RAM.
//
// When batchSize <= 0 the original single-query behaviour is used (full
// backward compatibility).
//
// HashTable requires the table to have a primary key; primary key columns are
// always included first (in the order specified by tbl.PrimaryKey). The optional
// ignoreFn may exclude non-primary-key columns; if no columns remain to hash
// after applying ignoreFn, HashTable returns an error.
//
// The returned map uses a composite key formed by joining primary key column
// values with "|" and maps that key to the hex-encoded SHA-256 hash of the
// row's "col=value" representation.
func HashTable(ctx context.Context, db *sql.DB, driver string, tbl schema.Table, ignoreFn func(table, column string) bool, batchSize int) (map[string]string, error) {
	log := logger.FromContext(ctx).WithTable(tbl.Name)
	progressMgr := progress.FromContext(ctx)

	if len(tbl.PrimaryKey) == 0 {
		return nil, fmt.Errorf("table %s has no primary key", tbl.Name)
	}

	cols := orderedColumns(tbl, ignoreFn)
	if len(cols) == 0 {
		return nil, fmt.Errorf("no columns to hash for table %s", tbl.Name)
	}

	rowCount, err := getRowCount(ctx, db, driver, tbl.Name)
	if err != nil {
		log.Warn("could not get row count for progress tracking", logger.FieldError, err.Error())
		rowCount = 0
	}

	log.Debug("starting table hash", logger.FieldRowCount, rowCount, "columns", len(cols), "batch_size", batchSize)

	result := make(map[string]string)

	// Progress bar shown for tables whose row count meets the threshold.
	const progressThreshold = 10000
	var bar *progress.Bar
	if progressMgr != nil && rowCount >= progressThreshold {
		bar = progressMgr.StartBar(ctx, fmt.Sprintf("Hashing %s", tbl.Name), rowCount)
		defer func() {
			_ = bar.Finish()
		}()
	}

	checkpointMgr := checkpoint.FromContext(ctx)

	if batchSize > 0 {
		if err := hashTableBatched(ctx, db, driver, tbl, cols, batchSize, bar, checkpointMgr, log, result); err != nil {
			return nil, err
		}
	} else {
		if err := hashTableFull(ctx, db, driver, tbl, cols, bar, checkpointMgr, log, result); err != nil {
			return nil, err
		}
	}

	log.Info("table hashing complete",
		logger.FieldRowCount, int64(len(result)),
		"unique_keys", len(result))

	// Mark table as completed in checkpoint.
	if checkpointMgr != nil {
		if err := checkpointMgr.Update(func(s *checkpoint.State) error {
			if s.HashTableState == nil {
				s.HashTableState = &checkpoint.HashTableState{
					Hashes:          make(map[string]map[string]string),
					CompletedTables: []string{},
				}
			}
			found := false
			for _, t := range s.HashTableState.CompletedTables {
				if t == tbl.Name {
					found = true
					break
				}
			}
			if !found {
				s.HashTableState.CompletedTables = append(s.HashTableState.CompletedTables, tbl.Name)
			}
			if s.HashTableState.Hashes[tbl.Name] == nil {
				s.HashTableState.Hashes[tbl.Name] = make(map[string]string)
			}
			for k, v := range result {
				s.HashTableState.Hashes[tbl.Name][k] = v
			}
			s.HashTableState.CurrentTable = ""
			s.HashTableState.CurrentRowCount = 0
			return nil
		}); err != nil {
			log.Warn("failed to save final checkpoint", logger.FieldError, err.Error())
		}
	}

	return result, nil
}

// hashTableBatched processes rows using keyset pagination (LIMIT batchSize per
// query). Between batches the Go runtime GC is hinted to reclaim per-batch
// allocations. Memory statistics are logged at debug level after each batch.
func hashTableBatched(
	ctx context.Context,
	db *sql.DB,
	driver string,
	tbl schema.Table,
	cols []string,
	batchSize int,
	bar *progress.Bar,
	checkpointMgr *checkpoint.Manager,
	log *logger.Logger,
	result map[string]string,
) error {
	dest := make([]any, len(cols))
	for i := range dest {
		var holder any
		dest[i] = &holder
	}

	var lastPKValues []any
	batchNum := 0

	for {
		query := BuildCursorQuery(driver, tbl.Name, cols, tbl.PrimaryKey, batchSize, lastPKValues)

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return fmt.Errorf("query rows (batch %d): %w", batchNum+1, err)
		}

		pageCount := 0
		for rows.Next() {
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				return fmt.Errorf("scan row: %w", err)
			}

			values := make([]any, len(cols))
			for i, v := range dest {
				values[i] = *(v.(*any))
			}

			key, err := buildKey(cols, values, tbl.PrimaryKey)
			if err != nil {
				rows.Close()
				return err
			}
			result[key] = hashRow(cols, values)

			// Track last PK values for the next cursor page.
			pkVals := make([]any, len(tbl.PrimaryKey))
			for i, pkCol := range tbl.PrimaryKey {
				for ci, c := range cols {
					if c == pkCol {
						pkVals[i] = values[ci]
						break
					}
				}
			}
			lastPKValues = pkVals

			pageCount++
			if bar != nil {
				_ = bar.Add(1)
			}
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate rows: %w", err)
		}
		rows.Close()

		batchNum++

		// Checkpoint and memory stats at every batch boundary.
		saveHashBatchCheckpoint(checkpointMgr, log, tbl, result, int64(len(result)))
		logBatchMemory(log, tbl.Name, batchNum, len(result))

		// Hint the GC so per-batch allocations can be reclaimed before the next page.
		runtime.GC()

		// Fewer rows than batchSize means this was the last page.
		if pageCount < batchSize {
			break
		}
	}

	return nil
}

// hashTableFull is the original single-query path, kept for batchSize <= 0.
func hashTableFull(
	ctx context.Context,
	db *sql.DB,
	driver string,
	tbl schema.Table,
	cols []string,
	bar *progress.Bar,
	checkpointMgr *checkpoint.Manager,
	log *logger.Logger,
	result map[string]string,
) error {
	query := buildSelect(driver, tbl.Name, cols, tbl.PrimaryKey)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query rows: %w", err)
	}
	defer rows.Close()

	dest := make([]any, len(cols))
	for i := range dest {
		var holder any
		dest[i] = &holder
	}

	rowsProcessed := int64(0)
	const checkpointInterval = 1000

	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}

		values := make([]any, len(cols))
		for i, v := range dest {
			values[i] = *(v.(*any))
		}

		key, err := buildKey(cols, values, tbl.PrimaryKey)
		if err != nil {
			return err
		}
		result[key] = hashRow(cols, values)

		rowsProcessed++
		if bar != nil {
			_ = bar.Add(1)
		}

		if checkpointMgr != nil && rowsProcessed%checkpointInterval == 0 {
			saveHashBatchCheckpoint(checkpointMgr, log, tbl, result, rowsProcessed)
		}
	}

	return rows.Err()
}

// saveHashBatchCheckpoint persists a partial snapshot of result to the checkpoint.
func saveHashBatchCheckpoint(mgr *checkpoint.Manager, log *logger.Logger, tbl schema.Table, result map[string]string, rowsProcessed int64) {
	if mgr == nil {
		return
	}
	if err := mgr.Update(func(s *checkpoint.State) error {
		if s.HashTableState == nil {
			s.HashTableState = &checkpoint.HashTableState{
				Hashes: make(map[string]map[string]string),
			}
		}
		s.HashTableState.CurrentTable = tbl.Name
		s.HashTableState.CurrentRowCount = rowsProcessed
		if s.HashTableState.Hashes[tbl.Name] == nil {
			s.HashTableState.Hashes[tbl.Name] = make(map[string]string)
		}
		for k, v := range result {
			s.HashTableState.Hashes[tbl.Name][k] = v
		}
		return nil
	}); err != nil {
		log.Warn("failed to save batch checkpoint", logger.FieldError, err.Error())
	}
}

// logBatchMemory emits a debug log with current heap allocation statistics.
func logBatchMemory(log *logger.Logger, table string, batchNum, totalRows int) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	allocMB := float64(ms.Alloc) / (1024 * 1024)
	log.Debug("batch complete",
		"table", table,
		"batch", batchNum,
		"total_rows_hashed", totalRows,
		"alloc_mb", fmt.Sprintf("%.1f", allocMB),
	)
}

// getRowCount returns the approximate row count for a table.
// Returns 0 if count cannot be determined.
func getRowCount(ctx context.Context, db *sql.DB, driver, table string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteIdent(driver, table)) // #nosec G201 -- table name sourced from schema introspection and passed through quoteIdent
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
// MySQL uses backticks, PostgreSQL/SQLite/Oracle use double-quotes, MSSQL uses square brackets.
// Unknown drivers are returned unquoted.
func quoteIdent(driver, ident string) string {
	switch driver {
	case "mysql":
		return "`" + ident + "`"
	case "postgres", "postgresql", "oracle":
		return `"` + ident + `"`
	case "mssql":
		return "[" + ident + "]"
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
