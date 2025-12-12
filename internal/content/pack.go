package content

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// GeneratePack builds a SQL migration pack for applying data diffs to prod.
// It uses the dev database as the source of truth for INSERT/UPDATE rows.
func GeneratePack(ctx context.Context, prodDriver string, devDB *sql.DB, devSchema *schema.Schema, diff DataDiff, ignoreFn func(table, column string) bool, outDir string) (string, error) {
	var stmts []string
	stmts = append(stmts, "BEGIN;")

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

	stmts = append(stmts, "COMMIT;")

	packPath := filepath.Join(outDir, "migration_pack.sql")
	if err := writeFile(packPath, strings.Join(stmts, "\n")); err != nil {
		return "", err
	}
	return packPath, nil
}

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

func splitKey(key string, expected int) ([]string, error) {
	parts := strings.Split(key, "|")
	if len(parts) != expected {
		return nil, fmt.Errorf("unexpected key format %q", key)
	}
	return parts, nil
}

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

func quoteIdents(driver string, cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = quoteIdent(driver, c)
	}
	return out
}

func literal(v any) string {
	switch val := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return "'" + escape(string(val)) + "'"
	case string:
		return "'" + escape(val) + "'"
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func escape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
