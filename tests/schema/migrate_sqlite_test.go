package schema_test

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// containsUncommented reports whether sql contains substr on a non-commented line.
// This prevents false-positives where strings.Contains matches "-- DROP TABLE ..."
// even though the destructive statement is still commented out.
func containsUncommented(sql, substr string) bool {
	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		if strings.Contains(trimmed, substr) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// GenerateMigration – unsupported driver
// ---------------------------------------------------------------------------

func TestGenerateMigration_UnsupportedDriver(t *testing.T) {
	diff := schema.DiffResult{}
	_, err := schema.GenerateMigration(diff, "baddriver", nil)
	if err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported driver") {
		t.Errorf("error = %q, want to contain 'unsupported driver'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – nil opts → safe defaults applied
// ---------------------------------------------------------------------------

func TestGenerateMigration_NilOpts_WrapsInTransaction(t *testing.T) {
	diff := schema.DiffResult{}
	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}
	if !strings.Contains(sql, "BEGIN;") {
		t.Error("expected BEGIN; in migration script")
	}
	if !strings.Contains(sql, "COMMIT;") {
		t.Error("expected COMMIT; in migration script")
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – CREATE TABLE for added tables
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddedTable_SQLite(t *testing.T) {
	tbl := schema.Table{
		Name: "users",
		Columns: map[string]schema.Column{
			"id":   {Name: "id", DataType: "integer", IsNullable: false},
			"name": {Name: "name", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}

	diff := schema.DiffResult{
		AddedTables: []schema.Table{tbl},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE") {
		t.Error("expected CREATE TABLE statement")
	}
	if !strings.Contains(sql, `"users"`) {
		t.Error("expected quoted table name 'users'")
	}
	if !strings.Contains(sql, "PRIMARY KEY") {
		t.Error("expected PRIMARY KEY in CREATE TABLE")
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – DROP TABLE commented out by default
// ---------------------------------------------------------------------------

func TestGenerateMigration_RemovedTable_CommentedByDefault(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"old_table"},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "-- DROP TABLE") {
		t.Error("DROP TABLE should be commented out by default")
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – DROP TABLE uncommented when AllowDropTable=true
// ---------------------------------------------------------------------------

func TestGenerateMigration_RemovedTable_UncommentedWhenAllowed(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"old_table"},
	}

	opts := &schema.MigrationOptions{AllowDropTable: true}
	sql, err := schema.GenerateMigration(diff, "sqlite", opts)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !containsUncommented(sql, `DROP TABLE "old_table"`) {
		t.Errorf("expected uncommented DROP TABLE, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – ADD COLUMN for existing table (nullable)
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddColumn_SQLite_Nullable_New(t *testing.T) {
	col := schema.Column{Name: "email", DataType: "text", IsNullable: true}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedColumns:   []schema.Column{col},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// Nullable column – no NOT NULL constraint in SQLite ADD COLUMN
	if !strings.Contains(sql, `ALTER TABLE "users" ADD COLUMN "email" text`) {
		t.Errorf("expected ADD COLUMN for nullable, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – ADD COLUMN NOT NULL without default → commented
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddColumn_SQLite_NotNullNoDefault_Commented(t *testing.T) {
	col := schema.Column{Name: "score", DataType: "integer", IsNullable: false, DefaultValue: nil}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "games",
				HasDifferences: true,
				AddedColumns:   []schema.Column{col},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// SQLite limitation: commented out
	if !strings.Contains(sql, "-- ALTER TABLE") {
		t.Errorf("expected commented-out ADD COLUMN for NOT NULL without default, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – DROP COLUMN commented by default
// ---------------------------------------------------------------------------

func TestGenerateMigration_DropColumn_CommentedByDefault(t *testing.T) {
	col := schema.Column{Name: "old_col", DataType: "text"}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "users",
				HasDifferences:  true,
				RemovedColumns:  []schema.Column{col},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// SQLite cannot execute DROP COLUMN — generator emits a comment instead.
	if !strings.Contains(sql, "SQLite does not support DROP COLUMN") {
		t.Errorf("expected commented DROP COLUMN notice for SQLite, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – MODIFY COLUMN SQLite → comment only
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyColumn_SQLite(t *testing.T) {
	nullable := true
	colDiff := schema.ColumnDiff{
		Column:          "status",
		TypeMismatch:    true,
		DevType:         "varchar(50)",
		ProdType:        "varchar(20)",
		DevNullable:     &nullable,
		NullableMismatch: false,
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "orders",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// SQLite MODIFY COLUMN → comment only
	if !strings.Contains(sql, "SQLite does not support MODIFY COLUMN") {
		t.Errorf("expected SQLite MODIFY COLUMN comment, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – CREATE INDEX on new table
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddedTable_WithIndex_SQLite(t *testing.T) {
	tbl := schema.Table{
		Name: "articles",
		Columns: map[string]schema.Column{
			"id":    {Name: "id", DataType: "integer", IsNullable: false},
			"title": {Name: "title", DataType: "text", IsNullable: false},
		},
		PrimaryKey: []string{"id"},
		Indexes: map[string]schema.Index{
			"idx_title": {Name: "idx_title", Columns: []string{"title"}, IsUnique: false},
		},
	}

	diff := schema.DiffResult{AddedTables: []schema.Table{tbl}}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "CREATE INDEX") {
		t.Errorf("expected CREATE INDEX statement, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – CREATE UNIQUE INDEX on new table
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddedTable_WithUniqueIndex_SQLite(t *testing.T) {
	tbl := schema.Table{
		Name: "slugs",
		Columns: map[string]schema.Column{
			"id":   {Name: "id", DataType: "integer", IsNullable: false},
			"slug": {Name: "slug", DataType: "text", IsNullable: false},
		},
		PrimaryKey: []string{"id"},
		Indexes: map[string]schema.Index{
			"uidx_slug": {Name: "uidx_slug", Columns: []string{"slug"}, IsUnique: true},
		},
	}

	diff := schema.DiffResult{AddedTables: []schema.Table{tbl}}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "CREATE UNIQUE INDEX") {
		t.Errorf("expected CREATE UNIQUE INDEX statement, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – ADD INDEX on modified table
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddedIndex_OnExistingTable(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_email", Columns: []string{"email"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, `CREATE INDEX "idx_email"`) {
		t.Errorf("expected CREATE INDEX for added index, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – DROP INDEX commented by default
// ---------------------------------------------------------------------------

func TestGenerateMigration_DroppedIndex_CommentedByDefault(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_old", Columns: []string{"old_col"}},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "-- DROP INDEX") {
		t.Errorf("expected commented DROP INDEX, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – DROP INDEX uncommented when allowed
// ---------------------------------------------------------------------------

func TestGenerateMigration_DroppedIndex_UncommentedWhenAllowed(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_old", Columns: []string{"old_col"}},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowDropIndex: true}
	sql, err := schema.GenerateMigration(diff, "sqlite", opts)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !containsUncommented(sql, `DROP INDEX "idx_old"`) {
		t.Errorf("expected uncommented DROP INDEX, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – Modified indexes → comment block
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifiedIndexes_CommentBlock(t *testing.T) {
	prodUnique := false
	devUnique := true

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				ModifiedIndexes: []schema.IndexDiff{
					{
						Name:          "idx_email",
						UniqueDiffers: true,
						ProdUnique:    &prodUnique,
						DevUnique:     &devUnique,
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "MODIFIED INDEXES") {
		t.Errorf("expected MODIFIED INDEXES comment block, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – ADD FOREIGN KEY on new table → SQLite limitation comment
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddedTable_WithFK_SQLite(t *testing.T) {
	tbl := schema.Table{
		Name: "comments",
		Columns: map[string]schema.Column{
			"id":      {Name: "id", DataType: "integer", IsNullable: false},
			"post_id": {Name: "post_id", DataType: "integer", IsNullable: false},
		},
		PrimaryKey: []string{"id"},
		ForeignKeys: map[string]schema.ForeignKey{
			"fk_post": {
				Name:              "fk_post",
				Columns:           []string{"post_id"},
				ReferencedTable:   "posts",
				ReferencedColumns: []string{"id"},
			},
		},
	}

	diff := schema.DiffResult{AddedTables: []schema.Table{tbl}}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// SQLite emits a comment about the FK limitation
	if !strings.Contains(sql, "SQLite limitation") {
		t.Errorf("expected SQLite FK limitation comment, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – ADD FOREIGN KEY on existing table
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddedForeignKey_OnExistingTable(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "posts",
				HasDifferences: true,
				AddedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_author",
						Columns:           []string{"author_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "FOREIGN KEY") {
		t.Errorf("expected FOREIGN KEY statement for mysql, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – DROP FOREIGN KEY commented by default
// ---------------------------------------------------------------------------

func TestGenerateMigration_DroppedForeignKey_CommentedByDefault(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "posts",
				HasDifferences: true,
				RemovedForeignKeys: []schema.ForeignKey{
					{Name: "fk_old", Columns: []string{"old_id"}, ReferencedTable: "olds", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "-- ALTER TABLE") {
		t.Errorf("expected commented DROP FOREIGN KEY, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – PRIMARY KEY modification for SQLite → comment block
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyPrimaryKey_SQLite(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "items",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"id", "tenant_id"},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "PRIMARY KEY MODIFICATION") {
		t.Errorf("expected PRIMARY KEY MODIFICATION section, got:\n%s", sql)
	}
	if !strings.Contains(sql, "SQLite limitation") {
		t.Errorf("expected SQLite limitation comment for PK modify, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – PRIMARY KEY modification for MySQL (commented by default)
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyPrimaryKey_MySQL_CommentedByDefault(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "items",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"id", "tenant_id"},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "-- ALTER TABLE") {
		t.Errorf("expected commented-out PK modification, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – tables with OnlyInProd/OnlyInDev are skipped
// ---------------------------------------------------------------------------

func TestGenerateMigration_SkipsOnlyInProdAndOnlyInDev(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{Name: "only_prod", HasDifferences: true, OnlyInProd: true},
			{Name: "only_dev", HasDifferences: true, OnlyInDev: true},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// These tables should not produce any ALTER TABLE or CREATE TABLE
	if strings.Contains(sql, "only_prod") || strings.Contains(sql, "only_dev") {
		t.Errorf("tables OnlyInProd/OnlyInDev should not appear in migration, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – tables with HasDifferences=false are skipped
// ---------------------------------------------------------------------------

func TestGenerateMigration_SkipsTablesWithNoDifferences(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{Name: "unchanged", HasDifferences: false},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if strings.Contains(sql, "unchanged") {
		t.Errorf("unchanged table should not appear in migration, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – ConfirmDestructive adds extra warning lines
// ---------------------------------------------------------------------------

func TestGenerateMigration_ConfirmDestructive_AddsWarnings(t *testing.T) {
	diff := schema.DiffResult{RemovedTables: []string{"dropped"}}

	opts := &schema.MigrationOptions{ConfirmDestructive: true, AllowDropTable: false}
	sql, err := schema.GenerateMigration(diff, "mysql", opts)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "Review carefully") {
		t.Errorf("expected 'Review carefully' warning, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – mysql driver produces backtick-quoted identifiers
// ---------------------------------------------------------------------------

func TestGenerateMigration_MySQL_BacktickQuoting(t *testing.T) {
	tbl := schema.Table{
		Name: "my_table",
		Columns: map[string]schema.Column{
			"id": {Name: "id", DataType: "int", IsNullable: false},
		},
		PrimaryKey: []string{"id"},
	}
	diff := schema.DiffResult{AddedTables: []schema.Table{tbl}}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "`my_table`") {
		t.Errorf("expected backtick-quoted table name for mysql, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – postgres driver uses double-quote identifiers
// ---------------------------------------------------------------------------

func TestGenerateMigration_Postgres_DoubleQuoteIdentifiers(t *testing.T) {
	tbl := schema.Table{
		Name: "my_table",
		Columns: map[string]schema.Column{
			"id": {Name: "id", DataType: "serial", IsNullable: false},
		},
		PrimaryKey: []string{"id"},
	}
	diff := schema.DiffResult{AddedTables: []schema.Table{tbl}}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, `"my_table"`) {
		t.Errorf("expected double-quoted table name for postgres, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigrationWithSchemas – nil prodSchema accepted
// ---------------------------------------------------------------------------

func TestGenerateMigrationWithSchemas_NilProdSchema(t *testing.T) {
	diff := schema.DiffResult{}
	sql, err := schema.GenerateMigrationWithSchemas(diff, "sqlite", nil, nil)
	if err != nil {
		t.Fatalf("GenerateMigrationWithSchemas() error = %v", err)
	}
	if !strings.Contains(sql, "BEGIN;") {
		t.Error("expected BEGIN; in migration output")
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – modified FK diff comment block
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifiedForeignKeys_CommentBlock(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				ModifiedForeignKeys: []schema.ForeignKeyDiff{
					{
						Name:                   "fk_customer",
						ColumnsDiffer:          true,
						ProdColumns:            []string{"customer_id"},
						DevColumns:             []string{"account_id"},
						ReferencedTableDiffers: true,
						ProdReferencedTable:    "customers",
						DevReferencedTable:     "accounts",
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "MODIFIED FOREIGN KEYS") {
		t.Errorf("expected MODIFIED FOREIGN KEYS section, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – modified FK: ON DELETE and ON UPDATE differ
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifiedForeignKeys_OnDeleteOnUpdateDiff(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "invoices",
				HasDifferences: true,
				ModifiedForeignKeys: []schema.ForeignKeyDiff{
					{
						Name:            "fk_inv",
						OnDeleteDiffers: true,
						ProdOnDelete:    "NO ACTION",
						DevOnDelete:     "CASCADE",
						OnUpdateDiffers: true,
						ProdOnUpdate:    "NO ACTION",
						DevOnUpdate:     "SET NULL",
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "ON DELETE") {
		t.Errorf("expected ON DELETE diff comment, got:\n%s", sql)
	}
	if !strings.Contains(sql, "ON UPDATE") {
		t.Errorf("expected ON UPDATE diff comment, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – modified indexes: columns differ
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifiedIndexes_ColumnsDiffer(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "search",
				HasDifferences: true,
				ModifiedIndexes: []schema.IndexDiff{
					{
						Name:         "idx_q",
						ColumnsDiffer: true,
						ProdColumns:  []string{"q"},
						DevColumns:   []string{"q", "lang"},
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "Columns:") {
		t.Errorf("expected Columns diff comment, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – ADD COLUMN with DEFAULT value
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddColumn_WithDefault_SQLite(t *testing.T) {
	defVal := "'pending'"
	col := schema.Column{Name: "status", DataType: "text", IsNullable: true, DefaultValue: &defVal}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "tasks",
				HasDifferences: true,
				AddedColumns:   []schema.Column{col},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "DEFAULT") {
		t.Errorf("expected DEFAULT clause in ADD COLUMN, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – mysql: MODIFY COLUMN with DevNullable only
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyColumn_MySQL_DevNullableOnly(t *testing.T) {
	nullable := true
	colDiff := schema.ColumnDiff{
		Column:       "bio",
		TypeMismatch: true,
		DevType:      "text",
		ProdType:     "varchar(255)",
		DevNullable:  &nullable,
		ProdNullable: nil,
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "profiles",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "MODIFY COLUMN") {
		t.Errorf("expected MODIFY COLUMN for mysql, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – mysql: MODIFY COLUMN with ProdNullable fallback
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyColumn_MySQL_ProdNullableFallback(t *testing.T) {
	notNull := false
	colDiff := schema.ColumnDiff{
		Column:       "bio",
		TypeMismatch: true,
		DevType:      "text",
		ProdType:     "varchar(255)",
		DevNullable:  nil,
		ProdNullable: &notNull,
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "profiles",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "NOT NULL") {
		t.Errorf("expected NOT NULL from ProdNullable fallback, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – mysql: MODIFY COLUMN with default mismatch (DevDefault set)
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyColumn_MySQL_DefaultMismatch_DevDefault(t *testing.T) {
	nullable := true
	devDef := "0"
	colDiff := schema.ColumnDiff{
		Column:          "count",
		TypeMismatch:    false,
		DevType:         "integer",
		ProdType:        "integer",
		DevNullable:     &nullable,
		DefaultMismatch: true,
		DevDefault:      &devDef,
		ProdDefault:     nil,
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "stats",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "DEFAULT 0") {
		t.Errorf("expected DEFAULT 0 from DevDefault, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – postgres: ALTER COLUMN TYPE + SET NOT NULL
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyColumn_Postgres_TypeAndNullable(t *testing.T) {
	notNull := false
	colDiff := schema.ColumnDiff{
		Column:           "age",
		TypeMismatch:     true,
		DevType:          "bigint",
		ProdType:         "integer",
		NullableMismatch: true,
		DevNullable:      &notNull,
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "people",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "ALTER COLUMN") {
		t.Errorf("expected ALTER COLUMN for postgres, got:\n%s", sql)
	}
	if !strings.Contains(sql, "SET NOT NULL") {
		t.Errorf("expected SET NOT NULL for postgres, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – postgres: ALTER COLUMN DROP DEFAULT
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyColumn_Postgres_DropDefault(t *testing.T) {
	colDiff := schema.ColumnDiff{
		Column:          "sku",
		DefaultMismatch: true,
		DevDefault:      nil, // remove default
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "products",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "DROP DEFAULT") {
		t.Errorf("expected DROP DEFAULT for postgres, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – postgres: ALTER COLUMN SET DEFAULT
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyColumn_Postgres_SetDefault(t *testing.T) {
	defVal := "now()"
	colDiff := schema.ColumnDiff{
		Column:          "created_at",
		DefaultMismatch: true,
		DevDefault:      &defVal,
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "events",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "SET DEFAULT") {
		t.Errorf("expected SET DEFAULT for postgres, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// GenerateMigration – postgres DROP NULLABLE: DevNullable=true → DROP NOT NULL
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyColumn_Postgres_DropNotNull(t *testing.T) {
	nullable := true
	colDiff := schema.ColumnDiff{
		Column:           "middle_name",
		NullableMismatch: true,
		DevNullable:      &nullable,
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "persons",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "DROP NOT NULL") {
		t.Errorf("expected DROP NOT NULL for postgres nullable=true, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// generateColumnDefinition – needsQuoting exercised via CREATE TABLE
// Each DEFAULT value type exercises a branch in needsQuoting.
// ---------------------------------------------------------------------------

func TestGenerateMigration_CreateTable_DefaultValues_Quoting(t *testing.T) {
	strDefault := "active"
	numDefault := "0"
	nullDefault := "NULL"
	trueDefault := "TRUE"
	falseDefault := "FALSE"
	funcDefault := "now()"
	curDefault := "CURRENT_TIMESTAMP"

	tbl := schema.Table{
		Name: "quoting_test",
		Columns: map[string]schema.Column{
			"str_col":  {Name: "str_col", DataType: "text", IsNullable: true, DefaultValue: &strDefault},
			"num_col":  {Name: "num_col", DataType: "integer", IsNullable: true, DefaultValue: &numDefault},
			"null_col": {Name: "null_col", DataType: "text", IsNullable: true, DefaultValue: &nullDefault},
			"true_col": {Name: "true_col", DataType: "integer", IsNullable: true, DefaultValue: &trueDefault},
			"fls_col":  {Name: "fls_col", DataType: "integer", IsNullable: true, DefaultValue: &falseDefault},
			"func_col": {Name: "func_col", DataType: "text", IsNullable: true, DefaultValue: &funcDefault},
			"cur_col":  {Name: "cur_col", DataType: "text", IsNullable: true, DefaultValue: &curDefault},
		},
	}
	diff := schema.DiffResult{AddedTables: []schema.Table{tbl}}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// String value should be quoted
	if !strings.Contains(sql, "DEFAULT 'active'") {
		t.Errorf("expected string default to be quoted, got:\n%s", sql)
	}
	// Numeric should NOT be quoted
	if !strings.Contains(sql, "DEFAULT 0") {
		t.Errorf("expected numeric default not quoted, got:\n%s", sql)
	}
	// NULL should NOT be quoted
	if !strings.Contains(sql, "DEFAULT NULL") {
		t.Errorf("expected NULL default not quoted, got:\n%s", sql)
	}
	// Function call should NOT be quoted
	if !strings.Contains(sql, "DEFAULT now()") {
		t.Errorf("expected function call default not quoted, got:\n%s", sql)
	}
	// CURRENT_TIMESTAMP should NOT be quoted
	if !strings.Contains(sql, "DEFAULT CURRENT_TIMESTAMP") {
		t.Errorf("expected CURRENT_TIMESTAMP not quoted, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// generateDropColumn – all supported drivers
// ---------------------------------------------------------------------------

func TestGenerateMigration_DropColumn_AllDrivers(t *testing.T) {
	col := schema.Column{Name: "old_col", DataType: "text"}
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "tbl",
				HasDifferences:  true,
				RemovedColumns:  []schema.Column{col},
			},
		},
	}

	for _, tt := range []struct {
		driver       string
		wantContains string
	}{
		{"mysql", "DROP COLUMN"},
		{"postgres", "DROP COLUMN"},
		{"postgresql", "DROP COLUMN"},
		{"sqlite", "SQLite does not support DROP COLUMN"},
	} {
		t.Run(tt.driver, func(t *testing.T) {
			opts := &schema.MigrationOptions{AllowDropColumn: true}
			got, err := schema.GenerateMigration(diff, tt.driver, opts)
			if err != nil {
				t.Fatalf("GenerateMigration() error = %v", err)
			}
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("driver=%s: want %q in output, got:\n%s", tt.driver, tt.wantContains, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// generateDropForeignKey – sqlite and default
// ---------------------------------------------------------------------------

func TestGenerateMigration_DropForeignKey_AllDrivers(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "tbl",
				HasDifferences: true,
				RemovedForeignKeys: []schema.ForeignKey{
					{Name: "fk_x", Columns: []string{"x_id"}, ReferencedTable: "x", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}

	tests := []struct {
		driver       string
		allowDrop    bool
		wantContains string
	}{
		{"mysql", true, "DROP FOREIGN KEY"},
		{"postgres", true, "DROP CONSTRAINT"},
		{"sqlite", true, "SQLite limitation"},
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			opts := &schema.MigrationOptions{AllowDropForeignKey: tt.allowDrop}
			got, err := schema.GenerateMigration(diff, tt.driver, opts)
			if err != nil {
				t.Fatalf("driver=%s GenerateMigration() error = %v", tt.driver, err)
			}
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("driver=%s: want %q, got:\n%s", tt.driver, tt.wantContains, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// generateAddForeignKey – ON DELETE/ON UPDATE clauses (non-NO ACTION)
// ---------------------------------------------------------------------------

func TestGenerateMigration_AddForeignKey_WithActions(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "posts",
				HasDifferences: true,
				AddedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
						OnUpdate:          "SET NULL",
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "ON DELETE CASCADE") {
		t.Errorf("expected ON DELETE CASCADE, got:\n%s", sql)
	}
	if !strings.Contains(sql, "ON UPDATE SET NULL") {
		t.Errorf("expected ON UPDATE SET NULL, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// generateModifyPrimaryKey – postgres: allowModify=true
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyPrimaryKey_Postgres_Allowed(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "items",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"id", "tenant_id"},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowModifyPrimaryKey: true}
	sql, err := schema.GenerateMigration(diff, "postgres", opts)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "DROP CONSTRAINT") {
		t.Errorf("expected DROP CONSTRAINT for postgres PK modify, got:\n%s", sql)
	}
	if !strings.Contains(sql, "ADD PRIMARY KEY") {
		t.Errorf("expected ADD PRIMARY KEY for postgres PK modify, got:\n%s", sql)
	}
}

// ---------------------------------------------------------------------------
// generateModifyPrimaryKey – mysql: allowModify=true
// ---------------------------------------------------------------------------

func TestGenerateMigration_ModifyPrimaryKey_MySQL_Allowed(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "items",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"id", "tenant_id"},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowModifyPrimaryKey: true}
	sql, err := schema.GenerateMigration(diff, "mysql", opts)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	if !strings.Contains(sql, "DROP PRIMARY KEY") {
		t.Errorf("expected DROP PRIMARY KEY for mysql allowed, got:\n%s", sql)
	}
}
