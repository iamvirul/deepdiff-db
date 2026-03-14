package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// CheckPrimaryKeys returns any tables that lack a primary key for the given driver.
// CheckPrimaryKeys identifies tables that do not have a primary key for the given driver and database/schema.
// It supports "mysql", "postgres"/"postgresql", and "sqlite". Tables listed in ignoreTables are skipped
// (match is case-insensitive).
//
// It returns a slice of table names that are missing a primary key (after applying ignores). An error is
// returned for unsupported drivers or when any database query, scan, or iteration fails.
func CheckPrimaryKeys(ctx context.Context, db *sql.DB, driver string, database string, ignoreTables []string) ([]string, error) {
	driver = strings.ToLower(driver)
	ignore := make(map[string]struct{}, len(ignoreTables))
	for _, t := range ignoreTables {
		ignore[strings.ToLower(t)] = struct{}{}
	}

	var rows *sql.Rows
	var err error

	switch driver {
	case "mysql":
		rows, err = db.QueryContext(ctx, `
			SELECT t.table_name
			FROM information_schema.tables t
			WHERE t.table_schema = ?
			  AND t.table_type = 'BASE TABLE'
			  AND NOT EXISTS (
			    SELECT 1 FROM information_schema.table_constraints tc
			    WHERE tc.table_schema = t.table_schema
			      AND tc.table_name = t.table_name
			      AND tc.constraint_type = 'PRIMARY KEY'
			  )
		`, database)
	case "postgres", "postgresql":
		rows, err = db.QueryContext(ctx, `
			SELECT c.relname AS table_name
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = current_schema()
			  AND c.relkind = 'r'
			  AND NOT EXISTS (
			    SELECT 1 FROM pg_index i
			    WHERE i.indrelid = c.oid
			      AND i.indisprimary
			  )
		`)
	case "sqlite":
		rows, err = db.QueryContext(ctx, `
			SELECT name AS table_name
			FROM sqlite_master
			WHERE type='table'
			  AND name NOT LIKE 'sqlite_%'
		`)
	case "mssql":
		// Return all user tables in the current schema that lack a primary key constraint.
		rows, err = db.QueryContext(ctx, `
			SELECT t.name AS table_name
			FROM sys.tables t
			WHERE SCHEMA_NAME(t.schema_id) = SCHEMA_NAME()
			  AND NOT EXISTS (
			    SELECT 1 FROM sys.key_constraints kc
			    WHERE kc.parent_object_id = t.object_id
			      AND kc.type = 'PK'
			  )
		`)
	default:
		return nil, fmt.Errorf("pk check unsupported for driver: %s", driver)
	}

	if err != nil {
		return nil, fmt.Errorf("query primary keys: %w", err)
	}
	defer rows.Close()

	var missing []string

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}
		if _, skip := ignore[strings.ToLower(table)]; skip {
			continue
		}

		if driver == "sqlite" {
			hasPK, err := sqliteTableHasPK(ctx, db, table)
			if err != nil {
				return nil, err
			}
			if !hasPK {
				missing = append(missing, table)
			}
			continue
		}

		missing = append(missing, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tables: %w", err)
	}

	return missing, nil
}

// sqliteTableHasPK reports whether the named SQLite table has a primary key.
// It inspects the table's column metadata (PRAGMA table_info) and returns `true` if any column has `pk > 0`.
// Returns `false` if no primary key column is found. An error is returned if the metadata query or row scanning fails.
func sqliteTableHasPK(ctx context.Context, db *sql.DB, table string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return false, fmt.Errorf("sqlite pragma table_info: %w", err)
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
			return false, fmt.Errorf("scan sqlite pragma: %w", err)
		}
		if pk > 0 {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate sqlite pragma: %w", err)
	}
	return false, nil
}
