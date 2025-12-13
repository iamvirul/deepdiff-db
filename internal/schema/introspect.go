package schema

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// LoadSchema loads table and column metadata for the specified SQL driver into a Schema,
// building Table entries with Columns and ordered PrimaryKey values and respecting the provided ignoreTables (case-insensitive).
// Supported drivers: "mysql", "postgres"/"postgresql", and "sqlite"; the database parameter is used for MySQL queries.
// Returns an error if the driver is unsupported or if any database query or row scanning fails.
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
			SELECT c.table_name,
			       c.column_name,
			       c.data_type,
			       c.is_nullable,
			       kcu.ordinal_position AS pk_ordinal
			FROM information_schema.columns c
			LEFT JOIN information_schema.key_column_usage kcu
			  ON kcu.table_schema = c.table_schema
			 AND kcu.table_name = c.table_name
			 AND kcu.column_name = c.column_name
			LEFT JOIN information_schema.table_constraints tc
			  ON tc.table_schema = c.table_schema
			 AND tc.table_name = c.table_name
			 AND tc.constraint_name = kcu.constraint_name
			 AND tc.constraint_type = 'PRIMARY KEY'
			WHERE c.table_schema = ?
			ORDER BY c.table_name, c.ordinal_position
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
			SELECT c.table_name,
			       c.column_name,
			       COALESCE(c.udt_name, c.data_type) AS data_type,
			       c.is_nullable,
			       pk.ordinal_position AS pk_ordinal
			FROM information_schema.columns c
			LEFT JOIN (
			  SELECT kc.table_name, kc.column_name, kc.ordinal_position
			  FROM information_schema.table_constraints tc
			  JOIN information_schema.key_column_usage kc
			    ON kc.constraint_name = tc.constraint_name
			   AND kc.table_schema = tc.table_schema
			   AND kc.table_name = tc.table_name
			  WHERE tc.constraint_type = 'PRIMARY KEY'
			    AND tc.table_schema = current_schema()
			) pk
			  ON pk.table_name = c.table_name
			 AND pk.column_name = c.column_name
			WHERE c.table_schema = current_schema()
			ORDER BY c.table_name, c.ordinal_position
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
			cols, pk, err := listSqliteColumns(ctx, db, tbl)
			if err != nil {
				return nil, err
			}
			s.Tables[tbl] = Table{
				Name:       tbl,
				Columns:    cols,
				PrimaryKey: pk,
			}
		}
	default:
		return nil, fmt.Errorf("schema load unsupported driver: %s", driver)
	}
	return s, nil
}

// scanColumns reads column metadata from rows and populates s.Tables with Table and Column entries.
// It skips tables present in ignore (keys compared case-insensitively), normalizes data types to lower case,
// interprets common nullable representations for Column.IsNullable, and collects primary key columns with
// their ordinal positions. After scanning, primary key columns are ordered by ordinal and assigned to each
// table's PrimaryKey slice. Returns any scan or iteration error encountered.
func scanColumns(rows *sql.Rows, s *Schema, ignore map[string]struct{}) error {
	type pkEntry struct {
		table string
		col   string
		pos   int
	}
	var pks []pkEntry

	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		var pkOrdinal sql.NullInt64
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &pkOrdinal); err != nil {
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
		if pkOrdinal.Valid {
			pks = append(pks, pkEntry{table: tableName, col: columnName, pos: int(pkOrdinal.Int64)})
		}
		s.Tables[tableName] = tbl
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// finalize primary keys ordered by ordinal
	sort.Slice(pks, func(i, j int) bool {
		if pks[i].table == pks[j].table {
			return pks[i].pos < pks[j].pos
		}
		return pks[i].table < pks[j].table
	})
	for _, pk := range pks {
		t := s.Tables[pk.table]
		t.PrimaryKey = append(t.PrimaryKey, pk.col)
		s.Tables[pk.table] = t
	}

	return nil
}

// listSqliteTables returns a sorted list of non-system table names from the SQLite database,
// excluding any names whose lowercase form appears as a key in the provided ignore map.
// It queries sqlite_master for entries of type "table" that do not start with "sqlite_".
// An error is returned if the query, row scan, or iteration fails.
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

// listSqliteColumns returns column metadata and the ordered primary-key column names for the named SQLite table.
// The first return value is a map from column name to Column (DataType is lowercased; IsNullable is true when the column is not marked NOT NULL).
// The second return value is a slice of primary key column names ordered by their PK ordinal as reported by PRAGMA table_info.
// A non-nil error is returned if the PRAGMA query fails or if rows cannot be scanned or iterated.
func listSqliteColumns(ctx context.Context, db *sql.DB, table string) (map[string]Column, []string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return nil, nil, fmt.Errorf("sqlite pragma table_info: %w", err)
	}
	defer rows.Close()

	cols := make(map[string]Column)
	var pkOrdered []struct {
		name string
		pos  int
	}
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
			return nil, nil, fmt.Errorf("scan sqlite column: %w", err)
		}
		cols[name] = Column{
			Name:       name,
			DataType:   strings.ToLower(ctype),
			IsNullable: notnull == 0,
		}
		if pk > 0 {
			pkOrdered = append(pkOrdered, struct {
				name string
				pos  int
			}{name: name, pos: pk})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate sqlite columns: %w", err)
	}
	sort.Slice(pkOrdered, func(i, j int) bool { return pkOrdered[i].pos < pkOrdered[j].pos })

	var pk []string
	for _, p := range pkOrdered {
		pk = append(pk, p.name)
	}

	return cols, pk, nil
}