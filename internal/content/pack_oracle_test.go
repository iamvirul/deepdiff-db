package content

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"sync"
	"testing"
)

// fakeOracleDriver is a minimal database/sql/driver implementation that returns
// a single hard-coded row for any query. It is used to test Oracle-specific paths
// in getFullColumnType and buildAlterTableAddColumn without a live Oracle instance.

type fakeOracleDriver struct{}
type fakeOracleConn struct{}
type fakeOracleStmt struct{}
type fakeOracleResult struct{}
type fakeOracleRows struct {
	cols []string
	row  []driver.Value
	done bool
}

func (d *fakeOracleDriver) Open(_ string) (driver.Conn, error) { return &fakeOracleConn{}, nil }
func (c *fakeOracleConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeOracleStmt{}, nil
}
func (c *fakeOracleConn) Close() error                        { return nil }
func (c *fakeOracleConn) Begin() (driver.Tx, error)           { return nil, nil }
func (s *fakeOracleStmt) Close() error                        { return nil }
func (s *fakeOracleStmt) NumInput() int                       { return -1 }
func (s *fakeOracleStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &fakeOracleResult{}, nil
}
func (s *fakeOracleStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &fakeOracleRows{
		cols: []string{"full_type"},
		row:  []driver.Value{"VARCHAR2(255)"},
	}, nil
}
func (r *fakeOracleResult) LastInsertId() (int64, error) { return 0, nil }
func (r *fakeOracleResult) RowsAffected() (int64, error) { return 1, nil }
func (r *fakeOracleRows) Columns() []string              { return r.cols }
func (r *fakeOracleRows) Close() error                   { return nil }
func (r *fakeOracleRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	copy(dest, r.row)
	return nil
}

var registerFakeOracleOnce sync.Once

func openFakeOracleDB(t *testing.T) *sql.DB {
	t.Helper()
	registerFakeOracleOnce.Do(func() {
		sql.Register("oracle-fake", &fakeOracleDriver{})
	})
	db, err := sql.Open("oracle-fake", "test")
	if err != nil {
		t.Fatalf("open fake oracle db: %v", err)
	}
	return db
}

// TestBuildAlterTableAddColumn_Oracle covers the "oracle" case in buildAlterTableAddColumn
// (internal/content/pack.go lines 906-908): Oracle uses ADD without the COLUMN keyword.
func TestBuildAlterTableAddColumn_Oracle(t *testing.T) {
	tests := []struct {
		name       string
		tableName  string
		columnName string
		fullType   string
		isNullable bool
		wantHas    []string
		wantNot    []string
	}{
		{
			name:       "nullable column — ADD without COLUMN keyword",
			tableName:  "ORDERS",
			columnName: "NOTES",
			fullType:   "VARCHAR2(500)",
			isNullable: true,
			wantHas:    []string{`ALTER TABLE "ORDERS" ADD "NOTES" VARCHAR2(500);`},
			wantNot:    []string{"ADD COLUMN"},
		},
		{
			name:       "NOT NULL column",
			tableName:  "PRODUCTS",
			columnName: "SKU",
			fullType:   "VARCHAR2(50)",
			isNullable: false,
			wantHas:    []string{`ALTER TABLE "PRODUCTS" ADD "SKU" VARCHAR2(50) NOT NULL;`},
			wantNot:    []string{"ADD COLUMN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAlterTableAddColumn("oracle", tt.tableName, tt.columnName, tt.fullType, tt.isNullable)
			for _, want := range tt.wantHas {
				if !strings.Contains(got, want) {
					t.Errorf("expected %q in output; got: %s", want, got)
				}
			}
			for _, notWant := range tt.wantNot {
				if strings.Contains(got, notWant) {
					t.Errorf("did not expect %q in output; got: %s", notWant, got)
				}
			}
		})
	}
}

// TestGetFullColumnType_Oracle covers the "oracle" case in getFullColumnType
// (internal/content/pack.go lines 621-649).
// The fake driver returns "VARCHAR2(255)" for any query, simulating USER_TAB_COLUMNS.
func TestGetFullColumnType_Oracle(t *testing.T) {
	db := openFakeOracleDB(t)
	defer db.Close()

	ctx := context.Background()

	got, err := getFullColumnType(ctx, db, "oracle", "", "CUSTOMERS", "EMAIL")
	if err != nil {
		t.Fatalf("getFullColumnType oracle: %v", err)
	}
	// Oracle result is lower-cased (line 649: return strings.ToLower(fullType), nil)
	if got != "varchar2(255)" {
		t.Errorf("expected varchar2(255), got %q", got)
	}
}
