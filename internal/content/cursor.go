package content

import (
	"fmt"
	"strings"
)

// BuildCursorQuery constructs a keyset-paginated SELECT statement.
//
// On the first page, pass nil for lastPKValues. On subsequent pages, pass the
// primary key column values from the last row of the previous page.
//
// Returns the query string and positional argument slice. The query uses ? as
// the placeholder for all drivers; callers that need driver-specific placeholders
// (e.g. PostgreSQL $1) should use database/sql's parameter binding — but note
// that for large-table cursor pagination the values are inlined via literal()
// rather than bound, matching the existing pattern in pack.go.
//
// The generated ORDER BY clause uses the primary key columns in the supplied order,
// ensuring a stable sort for consistent keyset pagination across pages.
func BuildCursorQuery(driver, table string, cols, pk []string, batchSize int, lastPKValues []any) string {
	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = quoteIdent(driver, c)
	}
	quotedPK := make([]string, len(pk))
	for i, c := range pk {
		quotedPK[i] = quoteIdent(driver, c)
	}

	base := fmt.Sprintf("SELECT %s FROM %s",
		strings.Join(quotedCols, ", "),
		quoteIdent(driver, table),
	)
	orderBy := " ORDER BY " + strings.Join(quotedPK, ", ")

	// MSSQL does not support LIMIT — it uses the ANSI SQL:2008 OFFSET/FETCH syntax.
	// The paging clause must come after ORDER BY.
	if driver == "mssql" {
		// "OFFSET 0 ROWS FETCH NEXT n ROWS ONLY" replaces "LIMIT n".
		mssqlPage := fmt.Sprintf(" OFFSET 0 ROWS FETCH NEXT %d ROWS ONLY", batchSize)
		if len(lastPKValues) == 0 {
			return base + orderBy + mssqlPage
		}
		return base + " " + buildCursorWhere(driver, pk, lastPKValues) + orderBy + mssqlPage
	}

	limit := fmt.Sprintf(" LIMIT %d", batchSize)
	if len(lastPKValues) == 0 {
		return base + orderBy + limit
	}
	return base + " " + buildCursorWhere(driver, pk, lastPKValues) + orderBy + limit
}

// buildCursorWhere returns a WHERE clause for keyset pagination over the given
// primary key columns and their last-seen values.
//
// For a single-column PK it produces:   WHERE pk > lastVal
// For a composite PK (pk1, pk2, …) it produces a lexicographic comparison:
//
//	WHERE (pk1 > v1) OR (pk1 = v1 AND pk2 > v2) OR …
//
// Column identifiers are quoted for the target driver; values are inlined via
// literal() to match the existing pack.go approach of building SQL strings
// without bound parameters for cursor queries.
func buildCursorWhere(driver string, pk []string, lastPKValues []any) string {
	var conditions []string
	for i := range pk {
		var parts []string
		// Equality predicates for all preceding PK columns
		for j := 0; j < i; j++ {
			parts = append(parts, fmt.Sprintf("%s = %s", quoteIdent(driver, pk[j]), literal(lastPKValues[j])))
		}
		// Strictly-greater predicate for column i
		parts = append(parts, fmt.Sprintf("%s > %s", quoteIdent(driver, pk[i]), literal(lastPKValues[i])))
		conditions = append(conditions, "("+strings.Join(parts, " AND ")+")")
	}
	return "WHERE " + strings.Join(conditions, " OR ")
}
