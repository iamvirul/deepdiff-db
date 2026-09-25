package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
	"github.com/iamvirul/deepdiff-db/internal/schema"
)

func TestFetchRowDiffReport_UpdatedAndRemovedAndAdded(t *testing.T) {
	ctx := context.Background()
	prodDB := openFetchMemDB(t)
	devDB := openFetchMemDB(t)

	mustExec(t, prodDB, `CREATE TABLE items (id INTEGER PRIMARY KEY, title TEXT, price REAL)`)
	mustExec(t, prodDB, `INSERT INTO items VALUES (1, 'Book', 10.5), (2, 'Pen', 1.5), (3, 'Notebook', 4.0)`)

	mustExec(t, devDB, `CREATE TABLE items (id INTEGER PRIMARY KEY, title TEXT, price REAL)`)
	mustExec(t, devDB, `INSERT INTO items VALUES (1, 'Book', 12.0), (2, 'Pen', 1.5), (4, 'Eraser', 0.5)`)

	prodSch := &schema.Schema{
		Tables: map[string]schema.Table{
			"items": {
				Name: "items",
				Columns: map[string]schema.Column{
					"id":    {Name: "id"},
					"title": {Name: "title"},
					"price": {Name: "price"},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}
	devSch := &schema.Schema{
		Tables: map[string]schema.Table{
			"items": {
				Name: "items",
				Columns: map[string]schema.Column{
					"id":    {Name: "id"},
					"title": {Name: "title"},
					"price": {Name: "price"},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	dataDiff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "items",
				Updated: []string{"1"},
				Removed: []string{"3"},
				Added:   []string{"4"},
			},
		},
	}
	conflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "items", Key: "1"},
		},
	}

	// 1. Test status "all"
	opts := resolve.RowDiffOptions{StatusFilter: "all"}
	report, err := resolve.FetchRowDiffReport(ctx, prodDB, devDB, "sqlite", "sqlite", prodSch, devSch, dataDiff, conflicts, opts)
	if err != nil {
		t.Fatalf("FetchRowDiffReport: %v", err)
	}

	if len(report.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(report.Tables))
	}
	if len(report.Tables[0].Rows) != 3 {
		t.Fatalf("expected 3 row diffs, got %d", len(report.Tables[0].Rows))
	}

	// Check updated row (key 1)
	row1 := report.Tables[0].Rows[0]
	if row1.Key != "1" || row1.Status != resolve.StatusUpdated {
		t.Errorf("expected row 1 to be updated, got %+v", row1)
	}
	if len(row1.DiffColumns) != 1 || row1.DiffColumns[0] != "price" {
		t.Errorf("expected price to be differing column for row 1, got %v", row1.DiffColumns)
	}

	// Check removed row (key 3)
	row3 := report.Tables[0].Rows[1]
	if row3.Key != "3" || row3.Status != resolve.StatusRemoved {
		t.Errorf("expected row 3 to be removed, got %+v", row3)
	}

	// Check added row (key 4)
	row4 := report.Tables[0].Rows[2]
	if row4.Key != "4" || row4.Status != resolve.StatusAdded {
		t.Errorf("expected row 4 to be added, got %+v", row4)
	}

	// 2. Test status "updated" only
	optsUpdated := resolve.RowDiffOptions{StatusFilter: "updated"}
	reportUpdated, err := resolve.FetchRowDiffReport(ctx, prodDB, devDB, "sqlite", "sqlite", prodSch, devSch, dataDiff, conflicts, optsUpdated)
	if err != nil {
		t.Fatalf("FetchRowDiffReport (updated): %v", err)
	}
	if len(reportUpdated.Tables[0].Rows) != 1 || reportUpdated.Tables[0].Rows[0].Key != "1" {
		t.Errorf("expected only key 1 for updated filter, got %d rows", len(reportUpdated.Tables[0].Rows))
	}

	// 3. Test key filter
	optsKey := resolve.RowDiffOptions{StatusFilter: "all", KeyFilter: "3"}
	reportKey, err := resolve.FetchRowDiffReport(ctx, prodDB, devDB, "sqlite", "sqlite", prodSch, devSch, dataDiff, conflicts, optsKey)
	if err != nil {
		t.Fatalf("FetchRowDiffReport (key filter): %v", err)
	}
	if len(reportKey.Tables[0].Rows) != 1 || reportKey.Tables[0].Rows[0].Key != "3" {
		t.Errorf("expected only key 3 for key filter, got %d rows", len(reportKey.Tables[0].Rows))
	}
}

func TestWriteRowDiffReport_AndFormat(t *testing.T) {
	report := resolve.RowDiffReport{
		Tables: []resolve.TableRowDiff{
			{
				Table: "customers",
				Rows: []resolve.RowDiff{
					{
						Table:  "customers",
						Key:    "4",
						Status: resolve.StatusUpdated,
						Columns: []resolve.ColumnDiff{
							{Column: "id", ProdVal: 4, DevVal: 4, Differs: false},
							{Column: "email", ProdVal: "dan@example.com", DevVal: "dan+new@example.com", Differs: true},
						},
						DiffColumns: []string{"email"},
					},
				},
			},
		},
	}

	tmpDir, err := os.MkdirTemp("", "rowdiff_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := resolve.WriteRowDiffReport(report, tmpDir); err != nil {
		t.Fatalf("WriteRowDiffReport: %v", err)
	}

	jsonPath := filepath.Join(tmpDir, "row_diff.json")
	if _, err := os.Stat(jsonPath); err != nil {
		t.Errorf("row_diff.json was not created: %v", err)
	}

	txtPath := filepath.Join(tmpDir, "row_diff.txt")
	txtData, err := os.ReadFile(txtPath)
	if err != nil {
		t.Fatalf("read row_diff.txt: %v", err)
	}

	txt := string(txtData)
	if !strings.Contains(txt, "Table: customers | Key: 4 (updated)") {
		t.Errorf("row_diff.txt missing table/key header: %s", txt)
	}
	if !strings.Contains(txt, "dan+new@example.com") {
		t.Errorf("row_diff.txt missing updated value: %s", txt)
	}
}
