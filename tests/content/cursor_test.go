package main

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/content"
)

func TestBuildCursorQuery_MSSQL_FirstPage(t *testing.T) {
	cols := []string{"id", "name", "email"}
	pk := []string{"id"}

	query := content.BuildCursorQuery("mssql", "users", cols, pk, 500, nil)

	// Must use OFFSET/FETCH — not LIMIT
	if !strings.Contains(query, "OFFSET 0 ROWS FETCH NEXT 500 ROWS ONLY") {
		t.Errorf("MSSQL first page must use OFFSET/FETCH, got: %s", query)
	}
	if strings.Contains(query, "LIMIT") {
		t.Errorf("MSSQL must not use LIMIT, got: %s", query)
	}
	// ORDER BY must precede OFFSET/FETCH
	orderPos := strings.Index(query, "ORDER BY")
	offsetPos := strings.Index(query, "OFFSET")
	if orderPos < 0 || offsetPos < 0 || orderPos > offsetPos {
		t.Errorf("ORDER BY must appear before OFFSET, got: %s", query)
	}
	// Square-bracket quoting
	if !strings.Contains(query, "[users]") {
		t.Errorf("MSSQL should use square-bracket quoting for table, got: %s", query)
	}
	if !strings.Contains(query, "[id]") {
		t.Errorf("MSSQL should use square-bracket quoting for PK, got: %s", query)
	}
}

func TestBuildCursorQuery_MSSQL_SubsequentPage(t *testing.T) {
	cols := []string{"id", "name"}
	pk := []string{"id"}
	lastPKValues := []any{42}

	query := content.BuildCursorQuery("mssql", "orders", cols, pk, 100, lastPKValues)

	// Must have a WHERE clause for cursor pagination
	if !strings.Contains(query, "WHERE") {
		t.Errorf("subsequent page must have WHERE clause, got: %s", query)
	}
	// Still uses OFFSET/FETCH
	if !strings.Contains(query, "OFFSET 0 ROWS FETCH NEXT 100 ROWS ONLY") {
		t.Errorf("subsequent page must still use OFFSET/FETCH, got: %s", query)
	}
	if strings.Contains(query, "LIMIT") {
		t.Errorf("MSSQL must not use LIMIT, got: %s", query)
	}
}

func TestBuildCursorQuery_MSSQL_CompositePK(t *testing.T) {
	cols := []string{"tenant_id", "id", "value"}
	pk := []string{"tenant_id", "id"}

	// First page — no previous values
	query := content.BuildCursorQuery("mssql", "events", cols, pk, 200, nil)

	if !strings.Contains(query, "OFFSET 0 ROWS FETCH NEXT 200 ROWS ONLY") {
		t.Errorf("composite PK first page must use OFFSET/FETCH, got: %s", query)
	}

	// Second page — with previous values
	lastPKValues := []any{"tenant-1", 99}
	query2 := content.BuildCursorQuery("mssql", "events", cols, pk, 200, lastPKValues)

	if !strings.Contains(query2, "WHERE") {
		t.Errorf("composite PK subsequent page must have WHERE clause, got: %s", query2)
	}
	if !strings.Contains(query2, "OFFSET 0 ROWS FETCH NEXT 200 ROWS ONLY") {
		t.Errorf("composite PK subsequent page must use OFFSET/FETCH, got: %s", query2)
	}
}

func TestBuildCursorQuery_MSSQL_VsMySQLDifference(t *testing.T) {
	cols := []string{"id"}
	pk := []string{"id"}

	mssqlQuery := content.BuildCursorQuery("mssql", "t", cols, pk, 50, nil)
	mysqlQuery := content.BuildCursorQuery("mysql", "t", cols, pk, 50, nil)

	if strings.Contains(mssqlQuery, "LIMIT") {
		t.Errorf("MSSQL should not use LIMIT, got: %s", mssqlQuery)
	}
	if !strings.Contains(mysqlQuery, "LIMIT 50") {
		t.Errorf("MySQL should use LIMIT, got: %s", mysqlQuery)
	}
	if !strings.Contains(mssqlQuery, "OFFSET 0 ROWS FETCH NEXT 50 ROWS ONLY") {
		t.Errorf("MSSQL should use OFFSET/FETCH, got: %s", mssqlQuery)
	}
}

func TestBuildCursorQuery_Oracle_FirstPage(t *testing.T) {
	cols := []string{"id", "name", "email"}
	pk := []string{"id"}

	query := content.BuildCursorQuery("oracle", "USERS", cols, pk, 500, nil)

	// Oracle 12c+ uses OFFSET/FETCH — not LIMIT
	if !strings.Contains(query, "OFFSET 0 ROWS FETCH NEXT 500 ROWS ONLY") {
		t.Errorf("Oracle first page must use OFFSET/FETCH, got: %s", query)
	}
	if strings.Contains(query, "LIMIT") {
		t.Errorf("Oracle must not use LIMIT, got: %s", query)
	}
	// ORDER BY must precede OFFSET/FETCH
	orderPos := strings.Index(query, "ORDER BY")
	offsetPos := strings.Index(query, "OFFSET")
	if orderPos < 0 || offsetPos < 0 || orderPos > offsetPos {
		t.Errorf("ORDER BY must appear before OFFSET, got: %s", query)
	}
	// Double-quote identifier quoting
	if !strings.Contains(query, `"USERS"`) {
		t.Errorf("Oracle should use double-quote identifiers for table, got: %s", query)
	}
}

func TestBuildCursorQuery_Oracle_SubsequentPage(t *testing.T) {
	cols := []string{"id", "status"}
	pk := []string{"id"}
	lastPKValues := []any{100}

	query := content.BuildCursorQuery("oracle", "ORDERS", cols, pk, 1000, lastPKValues)

	if !strings.Contains(query, "WHERE") {
		t.Errorf("Oracle subsequent page must have WHERE clause, got: %s", query)
	}
	if !strings.Contains(query, "OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY") {
		t.Errorf("Oracle subsequent page must use OFFSET/FETCH, got: %s", query)
	}
	if strings.Contains(query, "LIMIT") {
		t.Errorf("Oracle must not use LIMIT, got: %s", query)
	}
}
