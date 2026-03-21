package schema

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"sync"
	"testing"
)

// oracleFakeDriver is a query-aware fake SQL driver for testing Oracle introspection paths
// in introspect.go without requiring a live Oracle instance.
//
// It routes responses based on Oracle catalog table names present in the SQL:
//   - USER_TAB_COLUMNS  → column/PK metadata rows
//   - USER_INDEXES      → index metadata rows
//   - USER_CONSTRAINTS  → foreign key metadata rows (default)

type oracleFakeDriver struct{}
type oracleFakeConn struct{}
type oracleFakeStmt struct{ query string }
type oracleFakeResult struct{}

type oracleFakeRows struct {
	cols []string
	data [][]driver.Value
	pos  int
}

func (d *oracleFakeDriver) Open(_ string) (driver.Conn, error) { return &oracleFakeConn{}, nil }

func (c *oracleFakeConn) Prepare(query string) (driver.Stmt, error) {
	return &oracleFakeStmt{query: query}, nil
}
func (c *oracleFakeConn) Close() error              { return nil }
func (c *oracleFakeConn) Begin() (driver.Tx, error) { return nil, nil }

func (s *oracleFakeStmt) Close() error  { return nil }
func (s *oracleFakeStmt) NumInput() int { return -1 }
func (s *oracleFakeStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &oracleFakeResult{}, nil
}

func (s *oracleFakeStmt) Query(_ []driver.Value) (driver.Rows, error) {
	switch {
	case strings.Contains(s.query, "USER_TAB_COLUMNS"):
		// Simulates USER_TAB_COLUMNS for columns + PK data.
		// Columns: TABLE_NAME, COLUMN_NAME, data_type, is_nullable, DATA_DEFAULT, pk_ordinal
		return &oracleFakeRows{
			cols: []string{"TABLE_NAME", "COLUMN_NAME", "data_type", "is_nullable", "DATA_DEFAULT", "pk_ordinal"},
			data: [][]driver.Value{
				// CUSTOMERS.ID — PK, NOT NULL, NUMBER(10)
				{"CUSTOMERS", "ID", "number(10)", "NO", nil, int64(1)},
				// CUSTOMERS.EMAIL — nullable, no default, not PK
				{"CUSTOMERS", "EMAIL", "varchar2(100)", "YES", nil, nil},
				// CUSTOMERS.STATUS — has a default value
				{"CUSTOMERS", "STATUS", "varchar2(20)", "YES", driver.Value("ACTIVE"), nil},
			},
		}, nil

	case strings.Contains(s.query, "USER_INDEXES"):
		// Simulates USER_INDEXES / USER_IND_COLUMNS for non-PK indexes.
		// Columns: TABLE_NAME, INDEX_NAME, COLUMN_NAME, COLUMN_POSITION, is_unique
		return &oracleFakeRows{
			cols: []string{"TABLE_NAME", "INDEX_NAME", "COLUMN_NAME", "COLUMN_POSITION", "is_unique"},
			data: [][]driver.Value{
				// unique index on CUSTOMERS.EMAIL
				{"CUSTOMERS", "IDX_CUST_EMAIL", "EMAIL", int64(1), int64(1)},
				// multi-column non-unique index on CUSTOMERS (STATUS, EMAIL)
				{"CUSTOMERS", "IDX_CUST_STATUS", "STATUS", int64(1), int64(0)},
				{"CUSTOMERS", "IDX_CUST_STATUS", "EMAIL", int64(2), int64(0)},
				// row for a table NOT in the schema — exercises the `continue` branch
				{"UNKNOWN_TABLE", "IDX_UNKNOWN", "COL", int64(1), int64(0)},
			},
		}, nil

	default:
		// USER_CONSTRAINTS query for foreign keys.
		// Columns: TABLE_NAME, CONSTRAINT_NAME, COLUMN_NAME, POSITION, ref_table, ref_column, DELETE_RULE
		return &oracleFakeRows{
			cols: []string{"TABLE_NAME", "CONSTRAINT_NAME", "COLUMN_NAME", "POSITION",
				"ref_table", "ref_column", "DELETE_RULE"},
			data: [][]driver.Value{
				// FK on CUSTOMERS (table IS in schema) — exercises fkMap inner loop bodies
				{"CUSTOMERS", "FK_CUST_PARENT", "STATUS", int64(1), "CUSTOMERS", "ID", "CASCADE"},
				// second column on same FK constraint — exercises the Columns/ReferencedColumns append paths
				{"CUSTOMERS", "FK_CUST_PARENT", "EMAIL", int64(2), "CUSTOMERS", "EMAIL", "CASCADE"},
				// row for a table NOT in schema — exercises the `continue` branch
				{"UNKNOWN_TABLE", "FK_UNKNOWN", "SOME_COL", int64(1), "CUSTOMERS", "ID", "NO ACTION"},
			},
		}, nil
	}
}

func (r *oracleFakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r *oracleFakeResult) RowsAffected() (int64, error) { return 1, nil }
func (r *oracleFakeRows) Columns() []string              { return r.cols }
func (r *oracleFakeRows) Close() error                   { return nil }
func (r *oracleFakeRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

var registerOracleFakeOnce sync.Once

func openOracleFakeDB(t *testing.T) *sql.DB {
	t.Helper()
	registerOracleFakeOnce.Do(func() {
		sql.Register("oracle-schema-fake", &oracleFakeDriver{})
	})
	db, err := sql.Open("oracle-schema-fake", "test")
	if err != nil {
		t.Fatalf("open oracle fake db: %v", err)
	}
	return db
}

// TestLoad_Oracle_ColumnIntrospection exercises the Oracle case in Load (lines 174-214),
// loadOracleIndexes (lines 1113-1186), and loadOracleForeignKeys (lines 1188-1270).
func TestLoad_Oracle_ColumnIntrospection(t *testing.T) {
	db := openOracleFakeDB(t)
	defer db.Close()

	ctx := context.Background()

	s, err := LoadSchema(ctx, db, "oracle", "", nil)
	if err != nil {
		t.Fatalf("Load oracle: %v", err)
	}

	// Verify the CUSTOMERS table was loaded
	tbl, ok := s.Tables["CUSTOMERS"]
	if !ok {
		t.Fatalf("expected CUSTOMERS table in schema; got tables: %v", tableNames(s))
	}

	// Verify columns
	if _, ok := tbl.Columns["ID"]; !ok {
		t.Errorf("expected ID column in CUSTOMERS")
	}
	if _, ok := tbl.Columns["EMAIL"]; !ok {
		t.Errorf("expected EMAIL column in CUSTOMERS")
	}
	if _, ok := tbl.Columns["STATUS"]; !ok {
		t.Errorf("expected STATUS column in CUSTOMERS")
	}

	// ID should be NOT NULL, EMAIL should be nullable
	if tbl.Columns["ID"].IsNullable {
		t.Errorf("expected ID to be NOT NULL")
	}
	if !tbl.Columns["EMAIL"].IsNullable {
		t.Errorf("expected EMAIL to be nullable")
	}

	// STATUS should have a default value
	if tbl.Columns["STATUS"].DefaultValue == nil || *tbl.Columns["STATUS"].DefaultValue != "ACTIVE" {
		t.Errorf("expected STATUS default='ACTIVE', got %v", tbl.Columns["STATUS"].DefaultValue)
	}

	// ID should be the primary key
	if len(tbl.PrimaryKey) == 0 || tbl.PrimaryKey[0] != "ID" {
		t.Errorf("expected PrimaryKey=[ID], got %v", tbl.PrimaryKey)
	}

	// Indexes: IDX_CUST_EMAIL (unique), IDX_CUST_STATUS (non-unique, 2 columns)
	if _, ok := tbl.Indexes["IDX_CUST_EMAIL"]; !ok {
		t.Errorf("expected IDX_CUST_EMAIL index")
	}
	emailIdx := tbl.Indexes["IDX_CUST_EMAIL"]
	if !emailIdx.IsUnique {
		t.Errorf("expected IDX_CUST_EMAIL to be unique")
	}

	statusIdx, ok := tbl.Indexes["IDX_CUST_STATUS"]
	if !ok {
		t.Errorf("expected IDX_CUST_STATUS index")
	}
	if len(statusIdx.Columns) != 2 {
		t.Errorf("expected IDX_CUST_STATUS to have 2 columns, got %d", len(statusIdx.Columns))
	}
}

func tableNames(s *Schema) []string {
	names := make([]string, 0, len(s.Tables))
	for k := range s.Tables {
		names = append(names, k)
	}
	return names
}
