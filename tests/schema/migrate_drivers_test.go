package schema_test

// Coverage tests targeting driver-specific branches in:
//   - generateModifyColumn  (postgres nullable/default branches; mssql; oracle)
//   - generateModifyPrimaryKey (postgres; sqlite)
//   - generateDropForeignKey   (postgres; sqlite; default)
//   - generateDropIndex        (postgres/sqlite path; default/unknown driver)

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func boolRef(b bool) *bool { return &b }
func strRef(s string) *string { return &s }

func singleModifiedColumnDiff(t *testing.T, diff schema.DiffResult, driver string) string {
	t.Helper()
	sql, err := schema.GenerateMigration(diff, driver, nil)
	if err != nil {
		t.Fatalf("GenerateMigration(%s) error: %v", driver, err)
	}
	return sql
}

func modifyColDiff(table string, colDiff schema.ColumnDiff) schema.DiffResult {
	return schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            table,
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}
}

// ── generateModifyColumn — PostgreSQL branches ────────────────────────────────

func TestGenerateMigration_Postgres_ModifyColumn_NullableDropNotNull(t *testing.T) {
	// DevNullable=true → DROP NOT NULL
	diff := modifyColDiff("users", schema.ColumnDiff{
		Column:           "email",
		NullableMismatch: true,
		DevNullable:      boolRef(true),
		ProdNullable:     boolRef(false),
	})
	sql := singleModifiedColumnDiff(t, diff, "postgresql")
	if !strings.Contains(sql, "DROP NOT NULL") {
		t.Errorf("expected DROP NOT NULL for DevNullable=true, got:\n%s", sql)
	}
}

func TestGenerateMigration_Postgres_ModifyColumn_NullableSetNotNull(t *testing.T) {
	// DevNullable=false → SET NOT NULL
	diff := modifyColDiff("users", schema.ColumnDiff{
		Column:           "email",
		NullableMismatch: true,
		DevNullable:      boolRef(false),
		ProdNullable:     boolRef(true),
	})
	sql := singleModifiedColumnDiff(t, diff, "postgres")
	if !strings.Contains(sql, "SET NOT NULL") {
		t.Errorf("expected SET NOT NULL for DevNullable=false, got:\n%s", sql)
	}
}

func TestGenerateMigration_Postgres_ModifyColumn_DefaultMismatch_AddDefault(t *testing.T) {
	diff := modifyColDiff("products", schema.ColumnDiff{
		Column:          "discount",
		DefaultMismatch: true,
		DevDefault:      strRef("0.00"),
		TypeMismatch:    false,
	})
	sql := singleModifiedColumnDiff(t, diff, "postgresql")
	if !strings.Contains(sql, "SET DEFAULT 0.00") {
		t.Errorf("expected SET DEFAULT, got:\n%s", sql)
	}
}

func TestGenerateMigration_Postgres_ModifyColumn_DefaultMismatch_DropDefault(t *testing.T) {
	// DevDefault nil → DROP DEFAULT
	diff := modifyColDiff("products", schema.ColumnDiff{
		Column:          "discount",
		DefaultMismatch: true,
		DevDefault:      nil,
		TypeMismatch:    false,
	})
	sql := singleModifiedColumnDiff(t, diff, "postgresql")
	if !strings.Contains(sql, "DROP DEFAULT") {
		t.Errorf("expected DROP DEFAULT when DevDefault=nil, got:\n%s", sql)
	}
}

func TestGenerateMigration_Postgres_ModifyColumn_TypeAndNullable(t *testing.T) {
	diff := modifyColDiff("orders", schema.ColumnDiff{
		Column:           "amount",
		TypeMismatch:     true,
		ProdType:         "DECIMAL(10,2)",
		DevType:          "DECIMAL(12,2)",
		NullableMismatch: true,
		DevNullable:      boolRef(true),
	})
	sql := singleModifiedColumnDiff(t, diff, "postgresql")
	if !strings.Contains(sql, "ALTER COLUMN") {
		t.Errorf("expected ALTER COLUMN, got:\n%s", sql)
	}
	if !strings.Contains(sql, "DECIMAL(12,2)") {
		t.Errorf("expected type change, got:\n%s", sql)
	}
	if !strings.Contains(sql, "DROP NOT NULL") {
		t.Errorf("expected DROP NOT NULL, got:\n%s", sql)
	}
}

// ── generateModifyColumn — MSSQL branches ────────────────────────────────────

func TestGenerateMigration_MSSQL_ModifyColumn_TypeChange(t *testing.T) {
	diff := modifyColDiff("users", schema.ColumnDiff{
		Column:       "name",
		TypeMismatch: true,
		ProdType:     "NVARCHAR(100)",
		DevType:      "NVARCHAR(255)",
		DevNullable:  boolRef(true),
	})
	sql := singleModifiedColumnDiff(t, diff, "mssql")
	if !strings.Contains(sql, "ALTER COLUMN [name] NVARCHAR(255)") {
		t.Errorf("expected MSSQL ALTER COLUMN with new type, got:\n%s", sql)
	}
}

func TestGenerateMigration_MSSQL_ModifyColumn_NullableChange(t *testing.T) {
	diff := modifyColDiff("orders", schema.ColumnDiff{
		Column:           "notes",
		NullableMismatch: true,
		DevNullable:      boolRef(false), // NOT NULL
		ProdNullable:     boolRef(true),
		DevType:          "NVARCHAR(MAX)",
		ProdType:         "NVARCHAR(MAX)",
	})
	sql := singleModifiedColumnDiff(t, diff, "mssql")
	if !strings.Contains(sql, "NOT NULL") {
		t.Errorf("expected NOT NULL in MSSQL modify, got:\n%s", sql)
	}
}

func TestGenerateMigration_MSSQL_ModifyColumn_DefaultMismatch_WithDevDefault(t *testing.T) {
	diff := modifyColDiff("products", schema.ColumnDiff{
		Column:          "status",
		DefaultMismatch: true,
		DevDefault:      strRef("'active'"),
		DevNullable:     boolRef(false),
		DevType:         "NVARCHAR(20)",
	})
	sql := singleModifiedColumnDiff(t, diff, "mssql")
	// MSSQL emits a comment for DEFAULT changes
	if !strings.Contains(sql, "MSSQL: to change DEFAULT") {
		t.Errorf("expected MSSQL DEFAULT comment, got:\n%s", sql)
	}
	if !strings.Contains(sql, "'active'") {
		t.Errorf("expected default value in MSSQL comment, got:\n%s", sql)
	}
}

func TestGenerateMigration_MSSQL_ModifyColumn_DefaultMismatch_NilDevDefault(t *testing.T) {
	diff := modifyColDiff("products", schema.ColumnDiff{
		Column:          "status",
		DefaultMismatch: true,
		DevDefault:      nil, // removing default
		DevNullable:     boolRef(false),
		DevType:         "NVARCHAR(20)",
	})
	sql := singleModifiedColumnDiff(t, diff, "mssql")
	if !strings.Contains(sql, "MSSQL: to change DEFAULT") {
		t.Errorf("expected MSSQL DEFAULT comment for nil DevDefault, got:\n%s", sql)
	}
}

func TestGenerateMigration_MSSQL_ModifyColumn_NilNullable_FallsBackToNotNull(t *testing.T) {
	// When both DevNullable and ProdNullable are nil in MSSQL path,
	// the code defaults to NOT NULL (no error like mysql).
	diff := modifyColDiff("users", schema.ColumnDiff{
		Column:       "email",
		TypeMismatch: true,
		ProdType:     "NVARCHAR(100)",
		DevType:      "NVARCHAR(255)",
		DevNullable:  nil,
		ProdNullable: nil,
	})
	sql := singleModifiedColumnDiff(t, diff, "mssql")
	if !strings.Contains(sql, "NOT NULL") {
		t.Errorf("expected NOT NULL fallback for nil nullable in mssql, got:\n%s", sql)
	}
}

// ── generateModifyColumn — Oracle branches ───────────────────────────────────

func TestGenerateMigration_Oracle_ModifyColumn_TypeChange(t *testing.T) {
	diff := modifyColDiff("employees", schema.ColumnDiff{
		Column:       "salary",
		TypeMismatch: true,
		ProdType:     "NUMBER(10,2)",
		DevType:      "NUMBER(15,2)",
		DevNullable:  boolRef(false),
	})
	sql := singleModifiedColumnDiff(t, diff, "oracle")
	if !strings.Contains(sql, `MODIFY "salary" NUMBER(15,2)`) {
		t.Errorf("expected Oracle MODIFY with new type, got:\n%s", sql)
	}
}

func TestGenerateMigration_Oracle_ModifyColumn_NullableChange(t *testing.T) {
	diff := modifyColDiff("orders", schema.ColumnDiff{
		Column:           "notes",
		NullableMismatch: true,
		DevNullable:      boolRef(true), // NULL
		ProdNullable:     boolRef(false),
		DevType:          "VARCHAR2(4000)",
		ProdType:         "VARCHAR2(4000)",
	})
	sql := singleModifiedColumnDiff(t, diff, "oracle")
	if !strings.Contains(sql, "NULL") {
		t.Errorf("expected NULL in Oracle modify, got:\n%s", sql)
	}
}

func TestGenerateMigration_Oracle_ModifyColumn_DefaultMismatch_WithDevDefault(t *testing.T) {
	diff := modifyColDiff("products", schema.ColumnDiff{
		Column:          "status",
		DefaultMismatch: true,
		DevDefault:      strRef("'active'"),
		DevNullable:     boolRef(false),
		DevType:         "VARCHAR2(20)",
	})
	sql := singleModifiedColumnDiff(t, diff, "oracle")
	if !strings.Contains(sql, "Oracle: to change DEFAULT") {
		t.Errorf("expected Oracle DEFAULT comment, got:\n%s", sql)
	}
	if !strings.Contains(sql, "MODIFY") && !strings.Contains(sql, "'active'") {
		t.Errorf("expected Oracle DEFAULT modify with value, got:\n%s", sql)
	}
}

func TestGenerateMigration_Oracle_ModifyColumn_DefaultMismatch_NilDevDefault(t *testing.T) {
	diff := modifyColDiff("products", schema.ColumnDiff{
		Column:          "status",
		DefaultMismatch: true,
		DevDefault:      nil,
		DevNullable:     boolRef(false),
		DevType:         "VARCHAR2(20)",
	})
	sql := singleModifiedColumnDiff(t, diff, "oracle")
	if !strings.Contains(sql, "Oracle: to change DEFAULT") {
		t.Errorf("expected Oracle DEFAULT comment for nil DevDefault, got:\n%s", sql)
	}
}

// ── generateModifyPrimaryKey — PostgreSQL ────────────────────────────────────

func TestGenerateMigration_Postgres_ModifyPrimaryKey_Allowed(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"id", "tenant_id"},
				},
			},
		},
	}
	cfg := &schema.MigrationOptions{AllowModifyPrimaryKey: true}
	sql, err := schema.GenerateMigration(diff, "postgresql", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "DROP CONSTRAINT") {
		t.Errorf("expected DROP CONSTRAINT for postgres PK modification, got:\n%s", sql)
	}
	if !strings.Contains(sql, "ADD PRIMARY KEY") {
		t.Errorf("expected ADD PRIMARY KEY for postgres PK modification, got:\n%s", sql)
	}
}

func TestGenerateMigration_Postgres_ModifyPrimaryKey_Blocked(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"id", "tenant_id"},
				},
			},
		},
	}
	// AllowModifyPrimaryKey = false (default)
	sql, err := schema.GenerateMigration(diff, "postgresql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Statements should be commented out
	if strings.Contains(sql, "DROP CONSTRAINT") && !strings.Contains(sql, "--") {
		t.Errorf("expected commented-out DROP CONSTRAINT, got:\n%s", sql)
	}
}

func TestGenerateMigration_Postgres_ModifyPrimaryKey_NoteInOutput(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"uuid"},
				},
			},
		},
	}
	cfg := &schema.MigrationOptions{AllowModifyPrimaryKey: true}
	sql, err := schema.GenerateMigration(diff, "postgres", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// PostgreSQL path emits a NOTE about replacing constraint name
	if !strings.Contains(sql, "NOTE") {
		t.Errorf("expected NOTE about constraint name replacement, got:\n%s", sql)
	}
}

// ── generateModifyPrimaryKey — SQLite ────────────────────────────────────────

func TestGenerateMigration_SQLite_ModifyPrimaryKey_RequiresRecreation(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "items",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"uuid"},
				},
			},
		},
	}
	cfg := &schema.MigrationOptions{AllowModifyPrimaryKey: true}
	sql, err := schema.GenerateMigration(diff, "sqlite", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "SQLite limitation") {
		t.Errorf("expected SQLite limitation comment, got:\n%s", sql)
	}
	if !strings.Contains(sql, "table recreation") {
		t.Errorf("expected table recreation mention, got:\n%s", sql)
	}
}

// ── generateDropForeignKey — PostgreSQL ──────────────────────────────────────

func TestGenerateMigration_Postgres_DropForeignKey(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				RemovedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_orders_customer",
						Columns:           []string{"customer_id"},
						ReferencedTable:   "customers",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
	}
	cfg := &schema.MigrationOptions{AllowDropForeignKey: true}
	sql, err := schema.GenerateMigration(diff, "postgresql", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// PostgreSQL uses DROP CONSTRAINT (not DROP FOREIGN KEY like MySQL)
	if !strings.Contains(sql, "DROP CONSTRAINT") {
		t.Errorf("expected PostgreSQL DROP CONSTRAINT, got:\n%s", sql)
	}
	// MySQL-style DROP FOREIGN KEY syntax should not appear (only in comments is fine)
	if strings.Contains(sql, "DROP FOREIGN KEY `") {
		t.Errorf("PostgreSQL should not use MySQL DROP FOREIGN KEY backtick syntax, got:\n%s", sql)
	}
}

func TestGenerateMigration_Postgres_DropForeignKey_AliasDriver(t *testing.T) {
	// Both "postgres" and "postgresql" should produce the same output
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "line_items",
				HasDifferences: true,
				RemovedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_line_items_order",
						Columns:           []string{"order_id"},
						ReferencedTable:   "orders",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
	}
	cfg := &schema.MigrationOptions{AllowDropForeignKey: true}
	sqlPG, err := schema.GenerateMigration(diff, "postgres", cfg)
	if err != nil {
		t.Fatalf("unexpected error (postgres): %v", err)
	}
	sqlPQL, err := schema.GenerateMigration(diff, "postgresql", cfg)
	if err != nil {
		t.Fatalf("unexpected error (postgresql): %v", err)
	}
	// Both should use DROP CONSTRAINT (PostgreSQL style, not MySQL DROP FOREIGN KEY)
	for _, sql := range []string{sqlPG, sqlPQL} {
		if !strings.Contains(sql, "DROP CONSTRAINT") {
			t.Errorf("expected DROP CONSTRAINT, got:\n%s", sql)
		}
		if strings.Contains(sql, "DROP FOREIGN KEY `") {
			t.Errorf("should not use MySQL DROP FOREIGN KEY syntax, got:\n%s", sql)
		}
	}
}

// ── generateDropIndex — postgres/sqlite/unknown driver ────────────────────────

func TestGenerateMigration_Postgres_DropIndex(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: false},
				},
			},
		},
	}
	cfg := &schema.MigrationOptions{AllowDropIndex: true}
	sql, err := schema.GenerateMigration(diff, "postgres", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Postgres DROP INDEX does not include ON table (unlike MySQL/MSSQL)
	if !strings.Contains(sql, `DROP INDEX "idx_users_email";`) {
		t.Errorf("expected postgres DROP INDEX without ON clause, got:\n%s", sql)
	}
	if strings.Contains(sql, " ON ") {
		t.Errorf("postgres DROP INDEX should not contain ON, got:\n%s", sql)
	}
}

func TestGenerateMigration_SQLite_DropIndex(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "products",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_products_sku", Columns: []string{"sku"}, IsUnique: true},
				},
			},
		},
	}
	cfg := &schema.MigrationOptions{AllowDropIndex: true}
	sql, err := schema.GenerateMigration(diff, "sqlite", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// SQLite DROP INDEX also does not include ON table
	if !strings.Contains(sql, "DROP INDEX") {
		t.Errorf("expected DROP INDEX for sqlite, got:\n%s", sql)
	}
}
