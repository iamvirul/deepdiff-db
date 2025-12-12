package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// LoadSchema collects table/column metadata for the given driver.
func LoadSchema(ctx context.Context, db *sql.DB, driver string, database string, ignoreTables []string) (*Schema, error) {
	driver = strings.ToLower(driver)
	ignore := make(map[string]struct{}, len(ignoreTables))
	for _, t := range ignoreTables {
		ignore[strings.ToLower(t)] = struct{}{}
	}

	s := &Schema{Tables: make(map[string]Table)}
	switch driver {
	case "mysql":
		rows, err := db.QueryContext(ctx, `
			SELECT table_name, column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_schema = ?
			ORDER BY table_name, ordinal_position
		`, database)
		if err != nil {
			return nil, fmt.Errorf("mysql columns: %w", err)
		}
		defer rows.Close()
		if err := scanColumns(rows, s, ignore); err != nil {
			return nil, err
		}
	case "postgres", "postgresql":
		rows, err := db.QueryContext(ctx, `
			SELECT table_name, column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			ORDER BY table_name, ordinal_position
		`)
		if err != nil {
			return nil, fmt.Errorf("postgres columns: %w", err)
		}
		defer rows.Close()
		if err := scanColumns(rows, s, ignore); err != nil {
			return nil, err
		}
	case "sqlite":
		tables, err := listSqliteTables(ctx, db, ignore)
		if err != nil {
			return nil, err
		}
		for _, tbl := range tables {
			cols, err := listSqliteColumns(ctx, db, tbl)
			if err != nil {
				return nil, err
			}
			s.Tables[tbl] = Table{
				Name:    tbl,
				Columns: cols,
			}
		}
	default:
		return nil, fmt.Errorf("schema load unsupported driver: %s", driver)
	}
	return s, nil
}

func scanColumns(rows *sql.Rows, s *Schema, ignore map[string]struct{}) error {
	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable); err != nil {
			return fmt.Errorf("scan columns: %w", err)
		}

		if _, skip := ignore[strings.ToLower(tableName)]; skip {
			continue
		}

		tbl, ok := s.Tables[tableName]
		if !ok {
			tbl = Table{
				Name:    tableName,
				Columns: make(map[string]Column),
			}
		}
		tbl.Columns[columnName] = Column{
			Name:       columnName,
			DataType:   strings.ToLower(dataType),
			IsNullable: strings.EqualFold(isNullable, "YES") || isNullable == "1" || strings.EqualFold(isNullable, "true"),
		}
		s.Tables[tableName] = tbl
	}
	return rows.Err()
}

func listSqliteTables(ctx context.Context, db *sql.DB, ignore map[string]struct{}) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT name
		FROM sqlite_master
		WHERE type='table'
		  AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("sqlite tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan sqlite table: %w", err)
		}
		if _, skip := ignore[strings.ToLower(name)]; skip {
			continue
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite tables: %w", err)
	}
	return tables, nil
}

func listSqliteColumns(ctx context.Context, db *sql.DB, table string) (map[string]Column, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return nil, fmt.Errorf("sqlite pragma table_info: %w", err)
	}
	defer rows.Close()

	cols := make(map[string]Column)
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
			return nil, fmt.Errorf("scan sqlite column: %w", err)
		}
		cols[name] = Column{
			Name:       name,
			DataType:   strings.ToLower(ctype),
			IsNullable: notnull == 0,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite columns: %w", err)
	}
	return cols, nil
}

