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
)

// HashTable streams table rows and returns a map of PK composite key -> row hash.
// Ignores columns according to ignoreFn. Primary key columns must be present.
func HashTable(ctx context.Context, db *sql.DB, driver string, tbl schema.Table, ignoreFn func(table, column string) bool) (map[string]string, error) {
	if len(tbl.PrimaryKey) == 0 {
		return nil, fmt.Errorf("table %s has no primary key", tbl.Name)
	}

	cols := orderedColumns(tbl, ignoreFn)
	if len(cols) == 0 {
		return nil, fmt.Errorf("no columns to hash for table %s", tbl.Name)
	}

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
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return result, nil
}

func orderedColumns(tbl schema.Table, ignoreFn func(table, column string) bool) []string {
	var cols []string
	// Ensure primary keys first (and always included)
	for _, pk := range tbl.PrimaryKey {
		cols = append(cols, pk)
	}
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

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

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

func hashRow(cols []string, values []any) string {
	var b strings.Builder
	for i, col := range cols {
		fmt.Fprintf(&b, "%s=%v\n", col, values[i])
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
