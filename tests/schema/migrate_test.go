package main

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

func TestGenerateMigration_MySQL(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables:   []string{"new_table"},
		RemovedTables: []string{"old_table"},
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedColumns: []schema.Column{
					{Name: "email", DataType: "VARCHAR(255)", IsNullable: false},
					{Name: "status", DataType: "VARCHAR(50)", IsNullable: true},
				},
				RemovedColumns: []schema.Column{
					{Name: "legacy_field", DataType: "INT", IsNullable: true},
				},
				ModifiedColumns: []schema.ColumnDiff{
					{
						Column:           "age",
						TypeMismatch:     true,
						DevType:          "INT",
						ProdType:         "SMALLINT",
						NullableMismatch: false,
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql")
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Check transaction wrapping
	if !strings.Contains(sql, "BEGIN;") {
		t.Error("Expected migration to start with BEGIN;")
	}
	if !strings.Contains(sql, "COMMIT;") {
		t.Error("Expected migration to end with COMMIT;")
	}

	// Check added columns
	if !strings.Contains(sql, "ALTER TABLE `users` ADD COLUMN `email` VARCHAR(255) NOT NULL;") {
		t.Error("Expected ADD COLUMN statement for email")
	}
	if !strings.Contains(sql, "ALTER TABLE `users` ADD COLUMN `status` VARCHAR(50) NULL;") {
		t.Error("Expected ADD COLUMN statement for status")
	}

	// Check dropped columns are commented
	if !strings.Contains(sql, "-- ALTER TABLE `users` DROP COLUMN `legacy_field`;") {
		t.Error("Expected commented DROP COLUMN statement for legacy_field")
	}

	// Check modified columns
	if !strings.Contains(sql, "ALTER TABLE `users` MODIFY COLUMN `age` INT") {
		t.Error("Expected MODIFY COLUMN statement for age")
	}

	// Check table operations are commented
	if !strings.Contains(sql, "-- DROP TABLE `old_table`;") {
		t.Error("Expected commented DROP TABLE statement")
	}
	if !strings.Contains(sql, "-- CREATE TABLE `new_table`") {
		t.Error("Expected commented CREATE TABLE statement")
	}
}

func TestGenerateMigration_PostgreSQL(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "products",
				HasDifferences: true,
				AddedColumns: []schema.Column{
					{Name: "price", DataType: "DECIMAL(10,2)", IsNullable: false},
				},
				ModifiedColumns: []schema.ColumnDiff{
					{
						Column:           "description",
						TypeMismatch:     true,
						DevType:          "TEXT",
						ProdType:         "VARCHAR(255)",
						NullableMismatch: true,
						DevNullable:      boolPtr(true),
						ProdNullable:     boolPtr(false),
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Check added column
	if !strings.Contains(sql, "ALTER TABLE \"products\" ADD COLUMN \"price\" DECIMAL(10,2) NOT NULL;") {
		t.Error("Expected ADD COLUMN statement for price")
	}

	// Check type change
	if !strings.Contains(sql, "ALTER TABLE \"products\" ALTER COLUMN \"description\" TYPE TEXT;") {
		t.Error("Expected ALTER COLUMN TYPE statement")
	}

	// Check nullable change
	if !strings.Contains(sql, "ALTER TABLE \"products\" ALTER COLUMN \"description\" DROP NOT NULL;") {
		t.Error("Expected ALTER COLUMN DROP NOT NULL statement")
	}
}

func TestGenerateMigration_SQLite(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "items",
				HasDifferences: true,
				AddedColumns: []schema.Column{
					{Name: "category", DataType: "TEXT", IsNullable: true},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite")
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Check added column
	if !strings.Contains(sql, "ALTER TABLE \"items\" ADD COLUMN \"category\" TEXT;") {
		t.Error("Expected ADD COLUMN statement for category")
	}
}

func TestGenerateMigration_NoDifferences(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "unchanged_table",
				HasDifferences: false,
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql")
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Should only contain transaction and header
	if !strings.Contains(sql, "BEGIN;") {
		t.Error("Expected BEGIN;")
	}
	if !strings.Contains(sql, "COMMIT;") {
		t.Error("Expected COMMIT;")
	}

	// Should not contain any ALTER TABLE statements
	if strings.Contains(sql, "ALTER TABLE") {
		t.Error("Expected no ALTER TABLE statements for unchanged table")
	}
}

func TestGenerateMigration_UnsupportedDriver(t *testing.T) {
	diff := schema.DiffResult{}

	_, err := schema.GenerateMigration(diff, "oracle")
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
	if !strings.Contains(err.Error(), "unsupported driver") {
		t.Errorf("Expected 'unsupported driver' error, got: %v", err)
	}
}

func TestGenerateMigration_SQLiteLimitations(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "test_table",
				HasDifferences: true,
				AddedColumns: []schema.Column{
					{Name: "required_field", DataType: "TEXT", IsNullable: false},
				},
				RemovedColumns: []schema.Column{
					{Name: "old_field", DataType: "TEXT", IsNullable: true},
				},
				ModifiedColumns: []schema.ColumnDiff{
					{
						Column:       "status",
						TypeMismatch: true,
						DevType:      "INTEGER",
						ProdType:     "TEXT",
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite")
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Check that NOT NULL column is commented with limitation note
	if !strings.Contains(sql, "SQLite limitation") {
		t.Error("Expected SQLite limitation comment for NOT NULL column")
	}

	// Check that DROP COLUMN shows limitation
	if !strings.Contains(sql, "SQLite does not support DROP COLUMN") {
		t.Error("Expected SQLite DROP COLUMN limitation comment")
	}

	// Check that MODIFY COLUMN shows limitation
	if !strings.Contains(sql, "SQLite does not support MODIFY COLUMN") {
		t.Error("Expected SQLite MODIFY COLUMN limitation comment")
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}

func TestQuoteIdentifier_MySQL(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{
			name:       "simple identifier",
			identifier: "users",
			want:       "`users`",
		},
		{
			name:       "identifier with backtick",
			identifier: "user`table",
			want:       "`user``table`",
		},
		{
			name:       "schema-qualified name",
			identifier: "myschema.users",
			want:       "`myschema`.`users`",
		},
		{
			name:       "schema-qualified with quotes",
			identifier: "my`schema.user`table",
			want:       "`my``schema`.`user``table`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Access the internal quoteIdentifier by generating a migration
			// and checking the output contains properly quoted identifiers
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:           tt.identifier,
						HasDifferences: true,
						AddedColumns: []schema.Column{
							{Name: "test_col", DataType: "VARCHAR(255)", IsNullable: true},
						},
					},
				},
			}

			sql, err := schema.GenerateMigration(diff, "mysql")
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			if !strings.Contains(sql, tt.want) {
				t.Errorf("Expected SQL to contain %q, but got:\n%s", tt.want, sql)
			}
		})
	}
}

func TestQuoteIdentifier_PostgreSQL(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{
			name:       "simple identifier",
			identifier: "users",
			want:       "\"users\"",
		},
		{
			name:       "identifier with double quote",
			identifier: "user\"table",
			want:       "\"user\"\"table\"",
		},
		{
			name:       "schema-qualified name",
			identifier: "myschema.users",
			want:       "\"myschema\".\"users\"",
		},
		{
			name:       "schema-qualified with quotes",
			identifier: "my\"schema.user\"table",
			want:       "\"my\"\"schema\".\"user\"\"table\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:           tt.identifier,
						HasDifferences: true,
						AddedColumns: []schema.Column{
							{Name: "test_col", DataType: "VARCHAR(255)", IsNullable: true},
						},
					},
				},
			}

			sql, err := schema.GenerateMigration(diff, "postgresql")
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			if !strings.Contains(sql, tt.want) {
				t.Errorf("Expected SQL to contain %q, but got:\n%s", tt.want, sql)
			}
		})
	}
}

func TestQuoteIdentifier_SQLite(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{
			name:       "simple identifier",
			identifier: "users",
			want:       "\"users\"",
		},
		{
			name:       "identifier with double quote",
			identifier: "user\"table",
			want:       "\"user\"\"table\"",
		},
		{
			name:       "schema-qualified name",
			identifier: "main.users",
			want:       "\"main\".\"users\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:           tt.identifier,
						HasDifferences: true,
						AddedColumns: []schema.Column{
							{Name: "test_col", DataType: "TEXT", IsNullable: true},
						},
					},
				},
			}

			sql, err := schema.GenerateMigration(diff, "sqlite")
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			if !strings.Contains(sql, tt.want) {
				t.Errorf("Expected SQL to contain %q, but got:\n%s", tt.want, sql)
			}
		})
	}
}
