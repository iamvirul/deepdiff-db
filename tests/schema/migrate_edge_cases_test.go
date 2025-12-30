package schema_test

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// TestGenerateMigration_ModifyColumn_NilNullable tests error handling when nullable is nil
func TestGenerateMigration_ModifyColumn_NilNullable(t *testing.T) {
	// This tests the error path in generateModifyColumn when both DevNullable and ProdNullable are nil
	colDiff := schema.ColumnDiff{
		Column:       "test_col",
		TypeMismatch: true,
		DevType:      "varchar(100)",
		ProdType:     "varchar(50)",
		DevNullable:  nil, // Both nil to trigger error
		ProdNullable: nil,
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "test_table",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	// Should return an error for MySQL since it requires nullable info
	_, err := schema.GenerateMigration(diff, "mysql", nil)
	if err == nil {
		t.Error("expected error when both DevNullable and ProdNullable are nil for MySQL, got nil")
	}

	expectedErrMsg := "cannot determine NULL/NOT NULL constraint"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("error message = %q, should contain %q", err.Error(), expectedErrMsg)
	}
}

// TestGenerateMigration_AddColumn_SQLite_NotNullWithDefault tests SQLite NOT NULL with DEFAULT
func TestGenerateMigration_AddColumn_SQLite_NotNullWithDefault(t *testing.T) {
	// SQLite allows NOT NULL when DEFAULT is provided
	col := schema.Column{
		Name:         "new_col",
		DataType:     "integer",
		IsNullable:   false,
		DefaultValue: stringPtr("0"),
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "test_table",
				HasDifferences: true,
				AddedColumns:   []schema.Column{col},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// Should generate ADD COLUMN with NOT NULL and DEFAULT
	expectedSQL := `ADD COLUMN "new_col" integer NOT NULL DEFAULT 0`
	if !strings.Contains(sql, expectedSQL) {
		t.Errorf("Generated SQL does not contain expected statement.\nWant: %s\nGot: %s", expectedSQL, sql)
	}
}

// TestGenerateMigration_AddColumn_SQLite_NotNullWithoutDefault tests SQLite limitation
func TestGenerateMigration_AddColumn_SQLite_NotNullWithoutDefault(t *testing.T) {
	// SQLite does NOT allow NOT NULL without DEFAULT
	col := schema.Column{
		Name:         "new_col",
		DataType:     "text",
		IsNullable:   false,
		DefaultValue: nil, // No default
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "test_table",
				HasDifferences: true,
				AddedColumns:   []schema.Column{col},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// Should be commented out with limitation note
	expectedComment := "-- ALTER TABLE"
	expectedLimitation := "SQLite limitation: cannot add NOT NULL without default"
	if !strings.Contains(sql, expectedComment) || !strings.Contains(sql, expectedLimitation) {
		t.Errorf("Generated SQL should contain commented statement with limitation note.\nGot: %s", sql)
	}
}

// TestGenerateMigration_AddColumn_SQLite_Nullable tests SQLite nullable column
func TestGenerateMigration_AddColumn_SQLite_Nullable(t *testing.T) {
	col := schema.Column{
		Name:         "new_col",
		DataType:     "text",
		IsNullable:   true,
		DefaultValue: nil,
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "test_table",
				HasDifferences: true,
				AddedColumns:   []schema.Column{col},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// Should generate ADD COLUMN without NOT NULL
	expectedSQL := `ADD COLUMN "new_col" text;`
	if !strings.Contains(sql, expectedSQL) {
		t.Errorf("Generated SQL does not contain expected statement.\nWant: %s\nGot: %s", expectedSQL, sql)
	}
}

// TestGenerateMigration_ModifyColumn_DefaultMismatch_Remove tests removing DEFAULT value
func TestGenerateMigration_ModifyColumn_DefaultMismatch_Remove(t *testing.T) {
	colDiff := schema.ColumnDiff{
		Column:          "status",
		DefaultMismatch: true,
		TypeMismatch:    false,
		DevType:         "varchar(20)",
		ProdType:        "varchar(20)",
		DevNullable:     boolPtr(false),
		ProdNullable:    boolPtr(false),
		ProdDefault:     stringPtr("'active'"),
		DevDefault:      nil, // Remove default
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "users",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	// Test MySQL
	sqlMySQL, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("MySQL GenerateMigration() error = %v", err)
	}

	// MySQL MODIFY should NOT include DEFAULT clause when removing
	expectedMySQL := "MODIFY COLUMN `status` varchar(20) NOT NULL;"
	if !strings.Contains(sqlMySQL, expectedMySQL) {
		t.Errorf("MySQL: expected %q in output, got:\n%s", expectedMySQL, sqlMySQL)
	}

	// Test PostgreSQL
	sqlPG, err := schema.GenerateMigration(diff, "postgresql", nil)
	if err != nil {
		t.Fatalf("PostgreSQL GenerateMigration() error = %v", err)
	}

	// PostgreSQL should have DROP DEFAULT statement
	expectedPG := `ALTER TABLE "users" ALTER COLUMN "status" DROP DEFAULT;`
	if !strings.Contains(sqlPG, expectedPG) {
		t.Errorf("PostgreSQL: expected %q in output, got:\n%s", expectedPG, sqlPG)
	}
}

// TestGenerateMigration_ModifyColumn_DefaultOnly tests changing only DEFAULT value
func TestGenerateMigration_ModifyColumn_DefaultOnly(t *testing.T) {
	colDiff := schema.ColumnDiff{
		Column:           "count",
		DefaultMismatch:  true,
		TypeMismatch:     false,
		NullableMismatch: false,
		DevType:          "int",
		ProdType:         "int",
		DevNullable:      boolPtr(false),
		ProdNullable:     boolPtr(false),
		ProdDefault:      stringPtr("0"),
		DevDefault:       stringPtr("100"),
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "users",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	// Test PostgreSQL (uses separate ALTER statements)
	sqlPG, err := schema.GenerateMigration(diff, "postgresql", nil)
	if err != nil {
		t.Fatalf("PostgreSQL GenerateMigration() error = %v", err)
	}

	// Should only have SET DEFAULT, not TYPE or nullable changes
	expectedPG := `ALTER TABLE "users" ALTER COLUMN "count" SET DEFAULT 100;`
	if !strings.Contains(sqlPG, expectedPG) {
		t.Errorf("PostgreSQL: expected %q in output, got:\n%s", expectedPG, sqlPG)
	}

	// Should NOT contain TYPE or NOT NULL changes
	if strings.Contains(sqlPG, "ALTER COLUMN \"count\" TYPE") {
		t.Error("PostgreSQL: should not contain TYPE change when only default changed")
	}
	if strings.Contains(sqlPG, "SET NOT NULL") || strings.Contains(sqlPG, "DROP NOT NULL") {
		t.Error("PostgreSQL: should not contain nullable change when only default changed")
	}
}

// TestGenerateMigration_ModifyColumn_NoDefaultMismatch_KeepExisting tests keeping existing DEFAULT
func TestGenerateMigration_ModifyColumn_NoDefaultMismatch_KeepExisting(t *testing.T) {
	// When there's no default mismatch, the DEFAULT should be preserved in the output
	colDiff := schema.ColumnDiff{
		Column:           "status",
		DefaultMismatch:  false, // No mismatch
		TypeMismatch:     true,
		NullableMismatch: false,
		DevType:          "varchar(50)", // Type change only
		ProdType:         "varchar(20)",
		DevNullable:      boolPtr(false),
		ProdNullable:     boolPtr(false),
		DevDefault:       stringPtr("'active'"), // Both have same default
		ProdDefault:      stringPtr("'active'"),
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "users",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	// Test MySQL
	sqlMySQL, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("MySQL GenerateMigration() error = %v", err)
	}

	// Should include the DEFAULT in MySQL MODIFY even though it's not changing
	expectedMySQL := "DEFAULT 'active'"
	if !strings.Contains(sqlMySQL, expectedMySQL) {
		t.Errorf("MySQL: expected %q in output to preserve default, got:\n%s", expectedMySQL, sqlMySQL)
	}
}

// TestGenerateMigration_PostgreSQL_MultipleChanges tests PostgreSQL with type, nullable, and default changes
func TestGenerateMigration_PostgreSQL_MultipleChanges(t *testing.T) {
	colDiff := schema.ColumnDiff{
		Column:           "status",
		DefaultMismatch:  true,
		TypeMismatch:     true,
		NullableMismatch: true,
		DevType:          "varchar(100)",
		ProdType:         "varchar(20)",
		DevNullable:      boolPtr(true), // Change to nullable
		ProdNullable:     boolPtr(false),
		DevDefault:       stringPtr("'pending'"),
		ProdDefault:      stringPtr("'active'"),
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "users",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgresql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// Should have all three ALTER statements
	expectedStatements := []string{
		`ALTER TABLE "users" ALTER COLUMN "status" TYPE varchar(100);`,
		`ALTER TABLE "users" ALTER COLUMN "status" DROP NOT NULL;`,
		`ALTER TABLE "users" ALTER COLUMN "status" SET DEFAULT 'pending';`,
	}

	for _, expected := range expectedStatements {
		if !strings.Contains(sql, expected) {
			t.Errorf("PostgreSQL: expected %q in output, got:\n%s", expected, sql)
		}
	}
}

// TestGenerateMigration_PostgreSQL_SetNotNull tests PostgreSQL SET NOT NULL
func TestGenerateMigration_PostgreSQL_SetNotNull(t *testing.T) {
	colDiff := schema.ColumnDiff{
		Column:           "email",
		NullableMismatch: true,
		TypeMismatch:     false,
		DevType:          "varchar(100)",
		ProdType:         "varchar(100)",
		DevNullable:      boolPtr(false), // Change to NOT NULL
		ProdNullable:     boolPtr(true),
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "users",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgresql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	expectedSQL := `ALTER TABLE "users" ALTER COLUMN "email" SET NOT NULL;`
	if !strings.Contains(sql, expectedSQL) {
		t.Errorf("PostgreSQL: expected %q in output, got:\n%s", expectedSQL, sql)
	}
}

// TestGenerateMigration_SQLite_ModifyColumn tests SQLite limitation message
func TestGenerateMigration_SQLite_ModifyColumn(t *testing.T) {
	colDiff := schema.ColumnDiff{
		Column:       "status",
		TypeMismatch: true,
		DevType:      "text",
		ProdType:     "varchar(20)",
		DevNullable:  boolPtr(false),
	}

	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:            "users",
				HasDifferences:  true,
				ModifiedColumns: []schema.ColumnDiff{colDiff},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration() error = %v", err)
	}

	// SQLite should return comment about manual migration
	expectedMsg := "SQLite does not support MODIFY COLUMN"
	if !strings.Contains(sql, expectedMsg) {
		t.Errorf("SQLite: expected limitation message %q in output, got:\n%s", expectedMsg, sql)
	}
}
