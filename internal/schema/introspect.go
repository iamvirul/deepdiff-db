package schema

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/iamvirul/deepdiff-db/pkg/logger"
	"github.com/iamvirul/deepdiff-db/pkg/progress"
)

// LoadSchema loads table and column metadata for the specified SQL driver into a Schema,
// building Table entries with Columns and ordered PrimaryKey values and respecting the provided ignoreTables (case-insensitive).
// Supported drivers: "mysql", "postgres"/"postgresql", and "sqlite"; the database parameter is used for MySQL queries.
// Returns an error if the driver is unsupported or if any database query or row scanning fails.
func LoadSchema(ctx context.Context, db *sql.DB, driver string, database string, ignoreTables []string) (*Schema, error) {
	// Get logger and progress manager from context
	log := logger.FromContext(ctx).WithDatabase(driver, database)
	progressMgr := progress.FromContext(ctx)

	log.Info("loading database schema", "ignored_tables", len(ignoreTables))

	driver = strings.ToLower(driver)
	ignore := make(map[string]struct{}, len(ignoreTables))
	for _, t := range ignoreTables {
		ignore[strings.ToLower(t)] = struct{}{}
	}

	s := &Schema{Tables: make(map[string]Table)}
	switch driver {
	case "mysql":
		log.Debug("querying MySQL information_schema for columns and primary keys")
		rows, err := db.QueryContext(ctx, `
			SELECT c.table_name,
			       c.column_name,
			       c.column_type,
			       c.is_nullable,
			       c.column_default,
			       CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN kcu.ordinal_position ELSE NULL END AS pk_ordinal
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
		log.Debug("loaded table and column metadata", "tables", len(s.Tables))
	case "postgres", "postgresql":
		log.Debug("querying PostgreSQL information_schema for columns and primary keys")
		rows, err := db.QueryContext(ctx, `
			SELECT c.table_name,
			       c.column_name,
			       COALESCE(c.udt_name, c.data_type) AS data_type,
			       c.is_nullable,
			       c.column_default,
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
		log.Debug("loaded table and column metadata", "tables", len(s.Tables))
	case "sqlite":
		log.Debug("querying SQLite schema metadata")
		tables, err := listSqliteTables(ctx, db, ignore)
		if err != nil {
			return nil, err
		}
		log.Debug("found tables", "count", len(tables))

		// Create progress bar for many tables (threshold: 10)
		const progressThreshold = 10
		var bar *progress.Bar
		if progressMgr != nil && len(tables) >= progressThreshold {
			bar = progressMgr.StartBar(ctx, "Loading schema", int64(len(tables)))
			defer func() {
				_ = bar.Finish() // Ignore error - bar is finishing anyway
			}()
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
			if bar != nil {
				_ = bar.Add(1) // Ignore error - progress update
			}
		}
		log.Debug("loaded table and column metadata", "tables", len(s.Tables))
	case "mssql":
		log.Debug("querying MSSQL information_schema for columns and primary keys")
		rows, err := db.QueryContext(ctx, `
			SELECT
				c.TABLE_NAME,
				c.COLUMN_NAME,
				CASE
					WHEN c.CHARACTER_MAXIMUM_LENGTH IS NOT NULL
						AND c.DATA_TYPE IN ('char','nchar','varchar','nvarchar','binary','varbinary')
					THEN c.DATA_TYPE + '(' +
						CASE WHEN c.CHARACTER_MAXIMUM_LENGTH = -1 THEN 'max'
						     ELSE CAST(c.CHARACTER_MAXIMUM_LENGTH AS NVARCHAR(10))
						END + ')'
					WHEN c.NUMERIC_PRECISION IS NOT NULL AND c.NUMERIC_SCALE IS NOT NULL
						AND c.DATA_TYPE IN ('decimal','numeric')
					THEN c.DATA_TYPE + '(' + CAST(c.NUMERIC_PRECISION AS NVARCHAR(10))
					     + ',' + CAST(c.NUMERIC_SCALE AS NVARCHAR(10)) + ')'
					ELSE c.DATA_TYPE
				END AS data_type,
				c.IS_NULLABLE,
				c.COLUMN_DEFAULT,
				kcu.ORDINAL_POSITION AS pk_ordinal
			FROM INFORMATION_SCHEMA.COLUMNS c
			LEFT JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
				ON kcu.TABLE_SCHEMA = c.TABLE_SCHEMA
				AND kcu.TABLE_NAME  = c.TABLE_NAME
				AND kcu.COLUMN_NAME = c.COLUMN_NAME
				AND EXISTS (
					SELECT 1 FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
					WHERE tc.TABLE_SCHEMA     = kcu.TABLE_SCHEMA
					  AND tc.TABLE_NAME       = kcu.TABLE_NAME
					  AND tc.CONSTRAINT_NAME  = kcu.CONSTRAINT_NAME
					  AND tc.CONSTRAINT_TYPE  = 'PRIMARY KEY'
				)
			WHERE c.TABLE_SCHEMA = SCHEMA_NAME()
			ORDER BY c.TABLE_NAME, c.ORDINAL_POSITION
		`)
		if err != nil {
			return nil, fmt.Errorf("mssql columns: %w", err)
		}
		defer rows.Close()
		if err := scanColumns(rows, s, ignore); err != nil {
			return nil, err
		}
		log.Debug("loaded table and column metadata", "tables", len(s.Tables))
	case "oracle":
		// Oracle stores all unquoted identifiers in UPPERCASE.
		// USER_TAB_COLUMNS covers the current connected user's tables.
		// NULLABLE = 'Y' means the column is nullable.
		log.Debug("querying Oracle USER_TAB_COLUMNS for columns and primary keys")
		rows, err := db.QueryContext(ctx, `
			SELECT
				c.TABLE_NAME,
				c.COLUMN_NAME,
				CASE
					WHEN c.DATA_TYPE IN ('VARCHAR2','CHAR','NVARCHAR2','NCHAR','RAW') THEN
						LOWER(c.DATA_TYPE) || '(' || c.CHAR_LENGTH || ')'
					WHEN c.DATA_TYPE = 'NUMBER' AND c.DATA_PRECISION IS NOT NULL AND c.DATA_SCALE IS NOT NULL THEN
						'number(' || c.DATA_PRECISION || ',' || c.DATA_SCALE || ')'
					WHEN c.DATA_TYPE = 'NUMBER' AND c.DATA_PRECISION IS NOT NULL THEN
						'number(' || c.DATA_PRECISION || ')'
					WHEN c.DATA_TYPE = 'FLOAT' AND c.DATA_PRECISION IS NOT NULL THEN
						'float(' || c.DATA_PRECISION || ')'
					ELSE LOWER(c.DATA_TYPE)
				END AS data_type,
				CASE WHEN c.NULLABLE = 'Y' THEN 'YES' ELSE 'NO' END AS is_nullable,
				c.DATA_DEFAULT,
				cc.POSITION AS pk_ordinal
			FROM USER_TAB_COLUMNS c
			LEFT JOIN USER_CONSTRAINTS pk
				ON pk.TABLE_NAME = c.TABLE_NAME
				AND pk.CONSTRAINT_TYPE = 'P'
			LEFT JOIN USER_CONS_COLUMNS cc
				ON cc.CONSTRAINT_NAME = pk.CONSTRAINT_NAME
				AND cc.TABLE_NAME = c.TABLE_NAME
				AND cc.COLUMN_NAME = c.COLUMN_NAME
			ORDER BY c.TABLE_NAME, c.COLUMN_ID
		`)
		if err != nil {
			return nil, fmt.Errorf("oracle columns: %w", err)
		}
		defer rows.Close()
		if err := scanColumns(rows, s, ignore); err != nil {
			return nil, err
		}
		log.Debug("loaded table and column metadata", "tables", len(s.Tables))
	default:
		return nil, fmt.Errorf("schema load unsupported driver: %s", driver)
	}

	// Load indexes for all tables
	log.Debug("loading indexes")
	if err := loadIndexes(ctx, db, driver, database, s); err != nil {
		return nil, fmt.Errorf("load indexes: %w", err)
	}

	// Load foreign keys for all tables
	log.Debug("loading foreign keys")
	if err := loadForeignKeys(ctx, db, driver, database, s); err != nil {
		return nil, fmt.Errorf("load foreign keys: %w", err)
	}

	// Calculate total columns
	totalColumns := 0
	for _, tbl := range s.Tables {
		totalColumns += len(tbl.Columns)
	}

	log.Info("schema loaded successfully",
		"tables", len(s.Tables),
		"columns", totalColumns)

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
		var columnDefault sql.NullString
		var pkOrdinal sql.NullInt64
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &columnDefault, &pkOrdinal); err != nil {
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

		// Handle default value - convert sql.NullString to *string
		var defaultVal *string
		if columnDefault.Valid {
			defaultVal = &columnDefault.String
		}

		tbl.Columns[columnName] = Column{
			Name:         columnName,
			DataType:     strings.ToLower(dataType),
			IsNullable:   strings.EqualFold(isNullable, "YES") || isNullable == "1" || strings.EqualFold(isNullable, "true"),
			DefaultValue: defaultVal,
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

		// Handle default value - convert any to *string
		var defaultVal *string
		if dfltValue != nil {
			if s, ok := dfltValue.(string); ok && s != "" {
				defaultVal = &s
			} else if b, ok := dfltValue.([]byte); ok && len(b) > 0 {
				str := string(b)
				defaultVal = &str
			}
		}

		cols[name] = Column{
			Name:         name,
			DataType:     strings.ToLower(ctype),
			IsNullable:   notnull == 0,
			DefaultValue: defaultVal,
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

// loadIndexes queries the database for index metadata and populates the Indexes map for each table in the schema.
// It dispatches to driver-specific index loading functions based on the driver parameter.
// Primary key indexes are excluded as they are already handled separately.
func loadIndexes(ctx context.Context, db *sql.DB, driver string, database string, s *Schema) error {
	driver = strings.ToLower(driver)

	switch driver {
	case "mysql":
		return loadMySQLIndexes(ctx, db, database, s)
	case "postgres", "postgresql":
		return loadPostgreSQLIndexes(ctx, db, s)
	case "sqlite":
		return loadSQLiteIndexes(ctx, db, s)
	case "mssql":
		return loadMSSQLIndexes(ctx, db, s)
	case "oracle":
		return loadOracleIndexes(ctx, db, s)
	default:
		return fmt.Errorf("index introspection unsupported for driver: %s", driver)
	}
}

// loadMySQLIndexes queries MySQL's information_schema.statistics to load index metadata.
// It populates the Indexes map for each table in the schema, excluding PRIMARY indexes.
func loadMySQLIndexes(ctx context.Context, db *sql.DB, database string, s *Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, index_name, column_name, seq_in_index, non_unique
		FROM information_schema.statistics
		WHERE table_schema = ? AND index_name != 'PRIMARY'
		ORDER BY table_name, index_name, seq_in_index
	`, database)
	if err != nil {
		return fmt.Errorf("mysql indexes: %w", err)
	}
	defer rows.Close()

	// Temporary structure to accumulate columns per index
	type indexEntry struct {
		tableName  string
		indexName  string
		columnName string
		seqInIndex int
		nonUnique  int
	}

	var entries []indexEntry
	for rows.Next() {
		var e indexEntry
		if err := rows.Scan(&e.tableName, &e.indexName, &e.columnName, &e.seqInIndex, &e.nonUnique); err != nil {
			return fmt.Errorf("scan mysql index: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Group entries by table and index, then populate schema
	indexMap := make(map[string]map[string]*Index) // table -> indexName -> Index
	for _, e := range entries {
		if _, ok := s.Tables[e.tableName]; !ok {
			continue // Skip indexes for tables not in schema (e.g., ignored tables)
		}
		if indexMap[e.tableName] == nil {
			indexMap[e.tableName] = make(map[string]*Index)
		}
		if indexMap[e.tableName][e.indexName] == nil {
			indexMap[e.tableName][e.indexName] = &Index{
				Name:     e.indexName,
				Columns:  []string{},
				IsUnique: e.nonUnique == 0,
			}
		}
		indexMap[e.tableName][e.indexName].Columns = append(indexMap[e.tableName][e.indexName].Columns, e.columnName)
	}

	// Assign indexes to tables
	for tableName, indexes := range indexMap {
		t := s.Tables[tableName]
		if t.Indexes == nil {
			t.Indexes = make(map[string]Index)
		}
		for indexName, idx := range indexes {
			t.Indexes[indexName] = *idx
		}
		s.Tables[tableName] = t
	}

	return nil
}

// loadPostgreSQLIndexes queries PostgreSQL's system catalogs to load index metadata.
// It populates the Indexes map for each table in the schema, excluding primary key indexes.
func loadPostgreSQLIndexes(ctx context.Context, db *sql.DB, s *Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
			t.relname AS table_name,
			i.relname AS index_name,
			a.attname AS column_name,
			array_position(ix.indkey, a.attnum) AS column_position,
			ix.indisunique AS is_unique
		FROM pg_class t
		JOIN pg_index ix ON ix.indrelid = t.oid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = current_schema()
		  AND NOT ix.indisprimary
		ORDER BY t.relname, i.relname, array_position(ix.indkey, a.attnum)
	`)
	if err != nil {
		return fmt.Errorf("postgres indexes: %w", err)
	}
	defer rows.Close()

	// Temporary structure to accumulate columns per index
	type indexEntry struct {
		tableName  string
		indexName  string
		columnName string
		position   int
		isUnique   bool
	}

	var entries []indexEntry
	for rows.Next() {
		var e indexEntry
		if err := rows.Scan(&e.tableName, &e.indexName, &e.columnName, &e.position, &e.isUnique); err != nil {
			return fmt.Errorf("scan postgres index: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Group entries by table and index, then populate schema
	indexMap := make(map[string]map[string]*Index) // table -> indexName -> Index
	for _, e := range entries {
		if _, ok := s.Tables[e.tableName]; !ok {
			continue // Skip indexes for tables not in schema (e.g., ignored tables)
		}
		if indexMap[e.tableName] == nil {
			indexMap[e.tableName] = make(map[string]*Index)
		}
		if indexMap[e.tableName][e.indexName] == nil {
			indexMap[e.tableName][e.indexName] = &Index{
				Name:     e.indexName,
				Columns:  []string{},
				IsUnique: e.isUnique,
			}
		}
		indexMap[e.tableName][e.indexName].Columns = append(indexMap[e.tableName][e.indexName].Columns, e.columnName)
	}

	// Assign indexes to tables
	for tableName, indexes := range indexMap {
		t := s.Tables[tableName]
		if t.Indexes == nil {
			t.Indexes = make(map[string]Index)
		}
		for indexName, idx := range indexes {
			t.Indexes[indexName] = *idx
		}
		s.Tables[tableName] = t
	}

	return nil
}

// loadSQLiteIndexes uses PRAGMA statements to load index metadata for SQLite.
// It populates the Indexes map for each table in the schema, excluding primary key
// and auto-generated indexes (prefixed with 'sqlite_autoindex').
func loadSQLiteIndexes(ctx context.Context, db *sql.DB, s *Schema) error {
	for tableName, table := range s.Tables {
		if table.Indexes == nil {
			table.Indexes = make(map[string]Index)
		}

		// Get index list for this table
		indexRows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%s);", tableName))
		if err != nil {
			return fmt.Errorf("sqlite index_list: %w", err)
		}

		type indexMeta struct {
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		}
		var indexes []indexMeta

		for indexRows.Next() {
			var idx indexMeta
			if err := indexRows.Scan(&idx.seq, &idx.name, &idx.unique, &idx.origin, &idx.partial); err != nil {
				indexRows.Close()
				return fmt.Errorf("scan sqlite index_list: %w", err)
			}
			// Skip primary key indexes (origin = 'pk') and auto-generated indexes
			if idx.origin == "pk" || strings.HasPrefix(idx.name, "sqlite_autoindex") {
				continue
			}
			indexes = append(indexes, idx)
		}
		indexRows.Close()
		if err := indexRows.Err(); err != nil {
			return err
		}

		// For each index, get column info
		for _, idx := range indexes {
			infoRows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info(%s);", idx.name))
			if err != nil {
				return fmt.Errorf("sqlite index_info: %w", err)
			}

			var columns []string
			for infoRows.Next() {
				var seqno, cid int
				var name string
				if err := infoRows.Scan(&seqno, &cid, &name); err != nil {
					infoRows.Close()
					return fmt.Errorf("scan sqlite index_info: %w", err)
				}
				columns = append(columns, name)
			}
			infoRows.Close()
			if err := infoRows.Err(); err != nil {
				return err
			}

			table.Indexes[idx.name] = Index{
				Name:     idx.name,
				Columns:  columns,
				IsUnique: idx.unique == 1,
			}
		}

		s.Tables[tableName] = table
	}
	return nil
}

// loadForeignKeys queries the database for foreign key metadata and populates the ForeignKeys map for each table in the schema.
// It dispatches to driver-specific foreign key loading functions based on the driver parameter.
func loadForeignKeys(ctx context.Context, db *sql.DB, driver string, database string, s *Schema) error {
	driver = strings.ToLower(driver)

	switch driver {
	case "mysql":
		return loadMySQLForeignKeys(ctx, db, database, s)
	case "postgres", "postgresql":
		return loadPostgreSQLForeignKeys(ctx, db, s)
	case "sqlite":
		return loadSQLiteForeignKeys(ctx, db, s)
	case "mssql":
		return loadMSSQLForeignKeys(ctx, db, s)
	case "oracle":
		return loadOracleForeignKeys(ctx, db, s)
	default:
		return fmt.Errorf("foreign key introspection unsupported for driver: %s", driver)
	}
}

// loadMySQLForeignKeys queries MySQL's information_schema to load foreign key metadata.
// It populates the ForeignKeys map for each table in the schema.
func loadMySQLForeignKeys(ctx context.Context, db *sql.DB, database string, s *Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
			kcu.table_name,
			kcu.constraint_name,
			kcu.column_name,
			kcu.ordinal_position,
			kcu.referenced_table_name,
			kcu.referenced_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.referential_constraints rc
			ON rc.constraint_schema = kcu.constraint_schema
			AND rc.constraint_name = kcu.constraint_name
		WHERE kcu.table_schema = ?
			AND kcu.referenced_table_name IS NOT NULL
		ORDER BY kcu.table_name, kcu.constraint_name, kcu.ordinal_position
	`, database)
	if err != nil {
		return fmt.Errorf("mysql foreign keys: %w", err)
	}
	defer rows.Close()

	// Temporary structure to accumulate columns per foreign key
	type fkEntry struct {
		tableName        string
		constraintName   string
		columnName       string
		ordinalPosition  int
		referencedTable  string
		referencedColumn string
		deleteRule       string
		updateRule       string
	}

	var entries []fkEntry
	for rows.Next() {
		var e fkEntry
		if err := rows.Scan(&e.tableName, &e.constraintName, &e.columnName, &e.ordinalPosition,
			&e.referencedTable, &e.referencedColumn, &e.deleteRule, &e.updateRule); err != nil {
			return fmt.Errorf("scan mysql foreign key: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Group entries by table and constraint, then populate schema
	fkMap := make(map[string]map[string]*ForeignKey) // table -> constraintName -> ForeignKey
	for _, e := range entries {
		if _, ok := s.Tables[e.tableName]; !ok {
			continue // Skip foreign keys for tables not in schema (e.g., ignored tables)
		}
		if fkMap[e.tableName] == nil {
			fkMap[e.tableName] = make(map[string]*ForeignKey)
		}
		if fkMap[e.tableName][e.constraintName] == nil {
			fkMap[e.tableName][e.constraintName] = &ForeignKey{
				Name:              e.constraintName,
				Columns:           []string{},
				ReferencedTable:   e.referencedTable,
				ReferencedColumns: []string{},
				OnDelete:          e.deleteRule,
				OnUpdate:          e.updateRule,
			}
		}
		fkMap[e.tableName][e.constraintName].Columns = append(fkMap[e.tableName][e.constraintName].Columns, e.columnName)
		fkMap[e.tableName][e.constraintName].ReferencedColumns = append(fkMap[e.tableName][e.constraintName].ReferencedColumns, e.referencedColumn)
	}

	// Assign foreign keys to tables
	for tableName, fks := range fkMap {
		t := s.Tables[tableName]
		if t.ForeignKeys == nil {
			t.ForeignKeys = make(map[string]ForeignKey)
		}
		for fkName, fk := range fks {
			t.ForeignKeys[fkName] = *fk
		}
		s.Tables[tableName] = t
	}

	return nil
}

// loadPostgreSQLForeignKeys queries PostgreSQL's system catalogs to load foreign key metadata.
// It populates the ForeignKeys map for each table in the schema.
func loadPostgreSQLForeignKeys(ctx context.Context, db *sql.DB, s *Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
			tc.table_name,
			tc.constraint_name,
			kcu.column_name,
			kcu.ordinal_position,
			ccu.table_name AS referenced_table_name,
			ccu.column_name AS referenced_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = current_schema()
		ORDER BY tc.table_name, tc.constraint_name, kcu.ordinal_position
	`)
	if err != nil {
		return fmt.Errorf("postgres foreign keys: %w", err)
	}
	defer rows.Close()

	// Temporary structure to accumulate columns per foreign key
	type fkEntry struct {
		tableName        string
		constraintName   string
		columnName       string
		ordinalPosition  int
		referencedTable  string
		referencedColumn string
		deleteRule       string
		updateRule       string
	}

	var entries []fkEntry
	for rows.Next() {
		var e fkEntry
		if err := rows.Scan(&e.tableName, &e.constraintName, &e.columnName, &e.ordinalPosition,
			&e.referencedTable, &e.referencedColumn, &e.deleteRule, &e.updateRule); err != nil {
			return fmt.Errorf("scan postgres foreign key: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Group entries by table and constraint, then populate schema
	fkMap := make(map[string]map[string]*ForeignKey) // table -> constraintName -> ForeignKey
	for _, e := range entries {
		if _, ok := s.Tables[e.tableName]; !ok {
			continue // Skip foreign keys for tables not in schema (e.g., ignored tables)
		}
		if fkMap[e.tableName] == nil {
			fkMap[e.tableName] = make(map[string]*ForeignKey)
		}
		if fkMap[e.tableName][e.constraintName] == nil {
			fkMap[e.tableName][e.constraintName] = &ForeignKey{
				Name:              e.constraintName,
				Columns:           []string{},
				ReferencedTable:   e.referencedTable,
				ReferencedColumns: []string{},
				OnDelete:          e.deleteRule,
				OnUpdate:          e.updateRule,
			}
		}
		// Avoid duplicate columns (PostgreSQL query may return duplicates for composite keys)
		if !containsString(fkMap[e.tableName][e.constraintName].Columns, e.columnName) {
			fkMap[e.tableName][e.constraintName].Columns = append(fkMap[e.tableName][e.constraintName].Columns, e.columnName)
		}
		if !containsString(fkMap[e.tableName][e.constraintName].ReferencedColumns, e.referencedColumn) {
			fkMap[e.tableName][e.constraintName].ReferencedColumns = append(fkMap[e.tableName][e.constraintName].ReferencedColumns, e.referencedColumn)
		}
	}

	// Assign foreign keys to tables
	for tableName, fks := range fkMap {
		t := s.Tables[tableName]
		if t.ForeignKeys == nil {
			t.ForeignKeys = make(map[string]ForeignKey)
		}
		for fkName, fk := range fks {
			t.ForeignKeys[fkName] = *fk
		}
		s.Tables[tableName] = t
	}

	return nil
}

// loadSQLiteForeignKeys uses PRAGMA statements to load foreign key metadata for SQLite.
// It populates the ForeignKeys map for each table in the schema.
func loadSQLiteForeignKeys(ctx context.Context, db *sql.DB, s *Schema) error {
	for tableName, table := range s.Tables {
		if table.ForeignKeys == nil {
			table.ForeignKeys = make(map[string]ForeignKey)
		}

		// Get foreign key list for this table
		fkRows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%s);", tableName))
		if err != nil {
			return fmt.Errorf("sqlite foreign_key_list: %w", err)
		}

		// Group by id (foreign key id) - SQLite assigns same id to columns of composite FK
		type fkMeta struct {
			id       int
			seq      int
			table    string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		}
		fkByID := make(map[int][]fkMeta)

		for fkRows.Next() {
			var fk fkMeta
			if err := fkRows.Scan(&fk.id, &fk.seq, &fk.table, &fk.from, &fk.to, &fk.onUpdate, &fk.onDelete, &fk.match); err != nil {
				fkRows.Close()
				return fmt.Errorf("scan sqlite foreign_key_list: %w", err)
			}
			fkByID[fk.id] = append(fkByID[fk.id], fk)
		}
		fkRows.Close()
		if err := fkRows.Err(); err != nil {
			return err
		}

		// Build ForeignKey entries
		for id, fkMetas := range fkByID {
			// Sort by seq to get correct column order
			sort.Slice(fkMetas, func(i, j int) bool { return fkMetas[i].seq < fkMetas[j].seq })

			var columns, refColumns []string
			var refTable, onUpdate, onDelete string
			for _, m := range fkMetas {
				columns = append(columns, m.from)
				refColumns = append(refColumns, m.to)
				refTable = m.table
				onUpdate = m.onUpdate
				onDelete = m.onDelete
			}

			// SQLite doesn't name FK constraints, so generate a name
			fkName := fmt.Sprintf("fk_%s_%d", tableName, id)
			table.ForeignKeys[fkName] = ForeignKey{
				Name:              fkName,
				Columns:           columns,
				ReferencedTable:   refTable,
				ReferencedColumns: refColumns,
				OnDelete:          onDelete,
				OnUpdate:          onUpdate,
			}
		}

		s.Tables[tableName] = table
	}
	return nil
}

// containsString checks if a slice contains a string
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// loadMSSQLIndexes queries sys.indexes and sys.index_columns to load non-primary-key
// index metadata for all tables in the schema.
func loadMSSQLIndexes(ctx context.Context, db *sql.DB, s *Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
			t.name          AS table_name,
			i.name          AS index_name,
			c.name          AS column_name,
			ic.key_ordinal  AS seq_in_index,
			CAST(i.is_unique AS INT) AS is_unique
		FROM sys.indexes i
		JOIN sys.tables  t  ON t.object_id = i.object_id
		JOIN sys.index_columns ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id
		JOIN sys.columns c  ON c.object_id = i.object_id AND c.column_id = ic.column_id
		WHERE i.is_primary_key = 0
		  AND i.type_desc <> 'HEAP'
		  AND SCHEMA_NAME(t.schema_id) = SCHEMA_NAME()
		  AND ic.is_included_column = 0
		ORDER BY t.name, i.name, ic.key_ordinal
	`)
	if err != nil {
		return fmt.Errorf("mssql indexes: %w", err)
	}
	defer rows.Close()

	type indexEntry struct {
		tableName  string
		indexName  string
		columnName string
		seqInIndex int
		isUnique   int
	}
	var entries []indexEntry
	for rows.Next() {
		var e indexEntry
		if err := rows.Scan(&e.tableName, &e.indexName, &e.columnName, &e.seqInIndex, &e.isUnique); err != nil {
			return fmt.Errorf("scan mssql index: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	indexMap := make(map[string]map[string]*Index)
	for _, e := range entries {
		if _, ok := s.Tables[e.tableName]; !ok {
			continue
		}
		if indexMap[e.tableName] == nil {
			indexMap[e.tableName] = make(map[string]*Index)
		}
		if indexMap[e.tableName][e.indexName] == nil {
			indexMap[e.tableName][e.indexName] = &Index{
				Name:     e.indexName,
				Columns:  []string{},
				IsUnique: e.isUnique == 1,
			}
		}
		indexMap[e.tableName][e.indexName].Columns = append(indexMap[e.tableName][e.indexName].Columns, e.columnName)
	}

	for tableName, indexes := range indexMap {
		t := s.Tables[tableName]
		if t.Indexes == nil {
			t.Indexes = make(map[string]Index)
		}
		for indexName, idx := range indexes {
			t.Indexes[indexName] = *idx
		}
		s.Tables[tableName] = t
	}
	return nil
}

// loadMSSQLForeignKeys queries INFORMATION_SCHEMA views to load foreign key metadata
// for all tables in the schema.
func loadMSSQLForeignKeys(ctx context.Context, db *sql.DB, s *Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
			fk_cols.TABLE_NAME        AS table_name,
			rc.CONSTRAINT_NAME        AS constraint_name,
			fk_cols.COLUMN_NAME       AS column_name,
			fk_cols.ORDINAL_POSITION  AS ordinal_position,
			pk_cols.TABLE_NAME        AS referenced_table_name,
			pk_cols.COLUMN_NAME       AS referenced_column_name,
			rc.DELETE_RULE            AS delete_rule,
			rc.UPDATE_RULE            AS update_rule
		FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS rc
		JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE fk_cols
			ON fk_cols.CONSTRAINT_NAME   = rc.CONSTRAINT_NAME
			AND fk_cols.TABLE_SCHEMA     = rc.CONSTRAINT_SCHEMA
		JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE pk_cols
			ON pk_cols.CONSTRAINT_NAME   = rc.UNIQUE_CONSTRAINT_NAME
			AND pk_cols.TABLE_SCHEMA     = rc.UNIQUE_CONSTRAINT_SCHEMA
			AND pk_cols.ORDINAL_POSITION = fk_cols.ORDINAL_POSITION
		WHERE rc.CONSTRAINT_SCHEMA = SCHEMA_NAME()
		ORDER BY fk_cols.TABLE_NAME, rc.CONSTRAINT_NAME, fk_cols.ORDINAL_POSITION
	`)
	if err != nil {
		return fmt.Errorf("mssql foreign keys: %w", err)
	}
	defer rows.Close()

	type fkEntry struct {
		tableName        string
		constraintName   string
		columnName       string
		ordinalPosition  int
		referencedTable  string
		referencedColumn string
		deleteRule       string
		updateRule       string
	}
	var entries []fkEntry
	for rows.Next() {
		var e fkEntry
		if err := rows.Scan(&e.tableName, &e.constraintName, &e.columnName, &e.ordinalPosition,
			&e.referencedTable, &e.referencedColumn, &e.deleteRule, &e.updateRule); err != nil {
			return fmt.Errorf("scan mssql foreign key: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	fkMap := make(map[string]map[string]*ForeignKey)
	for _, e := range entries {
		if _, ok := s.Tables[e.tableName]; !ok {
			continue
		}
		if fkMap[e.tableName] == nil {
			fkMap[e.tableName] = make(map[string]*ForeignKey)
		}
		if fkMap[e.tableName][e.constraintName] == nil {
			fkMap[e.tableName][e.constraintName] = &ForeignKey{
				Name:              e.constraintName,
				Columns:           []string{},
				ReferencedTable:   e.referencedTable,
				ReferencedColumns: []string{},
				OnDelete:          e.deleteRule,
				OnUpdate:          e.updateRule,
			}
		}
		fkMap[e.tableName][e.constraintName].Columns = append(fkMap[e.tableName][e.constraintName].Columns, e.columnName)
		fkMap[e.tableName][e.constraintName].ReferencedColumns = append(fkMap[e.tableName][e.constraintName].ReferencedColumns, e.referencedColumn)
	}

	for tableName, fks := range fkMap {
		t := s.Tables[tableName]
		if t.ForeignKeys == nil {
			t.ForeignKeys = make(map[string]ForeignKey)
		}
		for fkName, fk := range fks {
			t.ForeignKeys[fkName] = *fk
		}
		s.Tables[tableName] = t
	}
	return nil
}

// loadOracleIndexes queries USER_INDEXES and USER_IND_COLUMNS to load non-primary-key
// index metadata for all tables visible to the current Oracle user.
func loadOracleIndexes(ctx context.Context, db *sql.DB, s *Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
			uic.TABLE_NAME,
			ui.INDEX_NAME,
			uic.COLUMN_NAME,
			uic.COLUMN_POSITION,
			CASE WHEN ui.UNIQUENESS = 'UNIQUE' THEN 1 ELSE 0 END AS is_unique
		FROM USER_INDEXES ui
		JOIN USER_IND_COLUMNS uic ON ui.INDEX_NAME = uic.INDEX_NAME
		WHERE ui.INDEX_TYPE = 'NORMAL'
		  AND NOT EXISTS (
		      SELECT 1 FROM USER_CONSTRAINTS uc
		      WHERE uc.INDEX_NAME = ui.INDEX_NAME
		        AND uc.CONSTRAINT_TYPE = 'P'
		  )
		ORDER BY uic.TABLE_NAME, ui.INDEX_NAME, uic.COLUMN_POSITION
	`)
	if err != nil {
		return fmt.Errorf("oracle indexes: %w", err)
	}
	defer rows.Close()

	type indexEntry struct {
		tableName  string
		indexName  string
		columnName string
		seqInIndex int
		isUnique   int
	}
	var entries []indexEntry
	for rows.Next() {
		var e indexEntry
		if err := rows.Scan(&e.tableName, &e.indexName, &e.columnName, &e.seqInIndex, &e.isUnique); err != nil {
			return fmt.Errorf("scan oracle index: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	indexMap := make(map[string]map[string]*Index)
	for _, e := range entries {
		if _, ok := s.Tables[e.tableName]; !ok {
			continue
		}
		if indexMap[e.tableName] == nil {
			indexMap[e.tableName] = make(map[string]*Index)
		}
		if indexMap[e.tableName][e.indexName] == nil {
			indexMap[e.tableName][e.indexName] = &Index{
				Name:     e.indexName,
				Columns:  []string{},
				IsUnique: e.isUnique == 1,
			}
		}
		indexMap[e.tableName][e.indexName].Columns = append(indexMap[e.tableName][e.indexName].Columns, e.columnName)
	}

	for tableName, indexes := range indexMap {
		t := s.Tables[tableName]
		if t.Indexes == nil {
			t.Indexes = make(map[string]Index)
		}
		for indexName, idx := range indexes {
			t.Indexes[indexName] = *idx
		}
		s.Tables[tableName] = t
	}
	return nil
}

// loadOracleForeignKeys queries USER_CONSTRAINTS and USER_CONS_COLUMNS to load foreign key
// metadata for all tables visible to the current Oracle user.
// Oracle does not support ON UPDATE CASCADE — UpdateRule is always "NO ACTION".
func loadOracleForeignKeys(ctx context.Context, db *sql.DB, s *Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
			fk.TABLE_NAME,
			fk.CONSTRAINT_NAME,
			fk_cols.COLUMN_NAME,
			fk_cols.POSITION,
			pk.TABLE_NAME   AS ref_table,
			pk_cols.COLUMN_NAME AS ref_column,
			fk.DELETE_RULE
		FROM USER_CONSTRAINTS fk
		JOIN USER_CONS_COLUMNS fk_cols
			ON fk_cols.CONSTRAINT_NAME = fk.CONSTRAINT_NAME
		JOIN USER_CONSTRAINTS pk
			ON pk.CONSTRAINT_NAME = fk.R_CONSTRAINT_NAME
		JOIN USER_CONS_COLUMNS pk_cols
			ON pk_cols.CONSTRAINT_NAME = pk.CONSTRAINT_NAME
			AND pk_cols.POSITION = fk_cols.POSITION
		WHERE fk.CONSTRAINT_TYPE = 'R'
		ORDER BY fk.TABLE_NAME, fk.CONSTRAINT_NAME, fk_cols.POSITION
	`)
	if err != nil {
		return fmt.Errorf("oracle foreign keys: %w", err)
	}
	defer rows.Close()

	type fkEntry struct {
		tableName        string
		constraintName   string
		columnName       string
		position         int
		referencedTable  string
		referencedColumn string
		deleteRule       string
	}
	var entries []fkEntry
	for rows.Next() {
		var e fkEntry
		if err := rows.Scan(&e.tableName, &e.constraintName, &e.columnName, &e.position,
			&e.referencedTable, &e.referencedColumn, &e.deleteRule); err != nil {
			return fmt.Errorf("scan oracle foreign key: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	fkMap := make(map[string]map[string]*ForeignKey)
	for _, e := range entries {
		if _, ok := s.Tables[e.tableName]; !ok {
			continue
		}
		if fkMap[e.tableName] == nil {
			fkMap[e.tableName] = make(map[string]*ForeignKey)
		}
		if fkMap[e.tableName][e.constraintName] == nil {
			fkMap[e.tableName][e.constraintName] = &ForeignKey{
				Name:              e.constraintName,
				Columns:           []string{},
				ReferencedTable:   e.referencedTable,
				ReferencedColumns: []string{},
				OnDelete:          e.deleteRule,
				OnUpdate:          "NO ACTION", // Oracle does not support ON UPDATE CASCADE
			}
		}
		fkMap[e.tableName][e.constraintName].Columns = append(fkMap[e.tableName][e.constraintName].Columns, e.columnName)
		fkMap[e.tableName][e.constraintName].ReferencedColumns = append(fkMap[e.tableName][e.constraintName].ReferencedColumns, e.referencedColumn)
	}

	for tableName, fks := range fkMap {
		t := s.Tables[tableName]
		if t.ForeignKeys == nil {
			t.ForeignKeys = make(map[string]ForeignKey)
		}
		for fkName, fk := range fks {
			t.ForeignKeys[fkName] = *fk
		}
		s.Tables[tableName] = t
	}
	return nil
}
