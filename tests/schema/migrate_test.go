package schema_test

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
						DevNullable:      boolPtr(false),
						ProdNullable:     boolPtr(false),
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
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
	if !strings.Contains(sql, "ALTER TABLE `users` MODIFY COLUMN `age` INT NOT NULL") {
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

	sql, err := schema.GenerateMigration(diff, "postgresql", nil)
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

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
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

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
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

	_, err := schema.GenerateMigration(diff, "oracle", nil)
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

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
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

			sql, err := schema.GenerateMigration(diff, "mysql", nil)
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

			sql, err := schema.GenerateMigration(diff, "postgresql", nil)
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			if !strings.Contains(sql, tt.want) {
				t.Errorf("Expected SQL to contain %q, but got:\n%s", tt.want, sql)
			}
		})
	}
}

func TestGenerateMigration_MySQL_NullableHandling(t *testing.T) {
	tests := []struct {
		name         string
		colDiff      schema.ColumnDiff
		wantContains string
		wantError    bool
	}{
		{
			name: "DevNullable is true",
			colDiff: schema.ColumnDiff{
				Column:       "test_col",
				TypeMismatch: true,
				DevType:      "VARCHAR(255)",
				ProdType:     "TEXT",
				DevNullable:  boolPtr(true),
				ProdNullable: boolPtr(false),
			},
			wantContains: "MODIFY COLUMN `test_col` VARCHAR(255) NULL",
			wantError:    false,
		},
		{
			name: "DevNullable is false",
			colDiff: schema.ColumnDiff{
				Column:       "test_col",
				TypeMismatch: true,
				DevType:      "VARCHAR(255)",
				ProdType:     "TEXT",
				DevNullable:  boolPtr(false),
				ProdNullable: boolPtr(true),
			},
			wantContains: "MODIFY COLUMN `test_col` VARCHAR(255) NOT NULL",
			wantError:    false,
		},
		{
			name: "DevNullable is nil, fallback to ProdNullable true",
			colDiff: schema.ColumnDiff{
				Column:       "test_col",
				TypeMismatch: true,
				DevType:      "VARCHAR(255)",
				ProdType:     "TEXT",
				DevNullable:  nil,
				ProdNullable: boolPtr(true),
			},
			wantContains: "MODIFY COLUMN `test_col` VARCHAR(255) NULL",
			wantError:    false,
		},
		{
			name: "DevNullable is nil, fallback to ProdNullable false",
			colDiff: schema.ColumnDiff{
				Column:       "test_col",
				TypeMismatch: true,
				DevType:      "VARCHAR(255)",
				ProdType:     "TEXT",
				DevNullable:  nil,
				ProdNullable: boolPtr(false),
			},
			wantContains: "MODIFY COLUMN `test_col` VARCHAR(255) NOT NULL",
			wantError:    false,
		},
		{
			name: "Both DevNullable and ProdNullable are nil - should error",
			colDiff: schema.ColumnDiff{
				Column:       "test_col",
				TypeMismatch: true,
				DevType:      "VARCHAR(255)",
				ProdType:     "TEXT",
				DevNullable:  nil,
				ProdNullable: nil,
			},
			wantContains: "",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:            "test_table",
						HasDifferences:  true,
						ModifiedColumns: []schema.ColumnDiff{tt.colDiff},
					},
				},
			}

			sql, err := schema.GenerateMigration(diff, "mysql", nil)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !strings.Contains(sql, tt.wantContains) {
				t.Errorf("Expected SQL to contain %q, but got:\n%s", tt.wantContains, sql)
			}
		})
	}
}

func TestGenerateMigration_PostgreSQL_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		colDiff      schema.ColumnDiff
		wantContains []string
		wantNotContain string
	}{
		{
			name: "Only TypeMismatch, no nullable change",
			colDiff: schema.ColumnDiff{
				Column:           "description",
				TypeMismatch:     true,
				DevType:          "TEXT",
				ProdType:         "VARCHAR(255)",
				NullableMismatch: false,
			},
			wantContains: []string{
				"ALTER TABLE \"test_table\" ALTER COLUMN \"description\" TYPE TEXT;",
			},
			wantNotContain: "NOT NULL",
		},
		{
			name: "Only NullableMismatch, no type change",
			colDiff: schema.ColumnDiff{
				Column:           "status",
				TypeMismatch:     false,
				NullableMismatch: true,
				DevNullable:      boolPtr(true),
				ProdNullable:     boolPtr(false),
			},
			wantContains: []string{
				"ALTER TABLE \"test_table\" ALTER COLUMN \"status\" DROP NOT NULL;",
			},
			wantNotContain: "TYPE",
		},
		{
			name: "NullableMismatch but DevNullable is nil - should skip nullable change",
			colDiff: schema.ColumnDiff{
				Column:           "price",
				TypeMismatch:     true,
				DevType:          "DECIMAL(10,2)",
				ProdType:         "NUMERIC",
				NullableMismatch: true,
				DevNullable:      nil,
				ProdNullable:     boolPtr(false),
			},
			wantContains: []string{
				"ALTER TABLE \"test_table\" ALTER COLUMN \"price\" TYPE DECIMAL(10,2);",
			},
			wantNotContain: "NOT NULL",
		},
		{
			name: "Set NOT NULL (DevNullable false)",
			colDiff: schema.ColumnDiff{
				Column:           "email",
				TypeMismatch:     false,
				NullableMismatch: true,
				DevNullable:      boolPtr(false),
				ProdNullable:     boolPtr(true),
			},
			wantContains: []string{
				"ALTER TABLE \"test_table\" ALTER COLUMN \"email\" SET NOT NULL;",
			},
			wantNotContain: "TYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:            "test_table",
						HasDifferences:  true,
						ModifiedColumns: []schema.ColumnDiff{tt.colDiff},
					},
				},
			}

			sql, err := schema.GenerateMigration(diff, "postgresql", nil)
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(sql, want) {
					t.Errorf("Expected SQL to contain %q, but got:\n%s", want, sql)
				}
			}

			if tt.wantNotContain != "" && strings.Contains(sql, tt.wantNotContain) {
				t.Errorf("Expected SQL to NOT contain %q, but got:\n%s", tt.wantNotContain, sql)
			}
		})
	}
}

func TestGenerateMigration_OnlyTableOperations(t *testing.T) {
	tests := []struct {
		name         string
		diff         schema.DiffResult
		wantContains []string
	}{
		{
			name: "Only removed tables",
			diff: schema.DiffResult{
				RemovedTables: []string{"old_table1", "old_table2"},
			},
			wantContains: []string{
				"-- DROP TABLE `old_table1`;",
				"-- DROP TABLE `old_table2`;",
				"-- DROP TABLES (present in prod but not in dev)",
				"BEGIN;",
				"COMMIT;",
			},
		},
		{
			name: "Only added tables",
			diff: schema.DiffResult{
				AddedTables: []string{"new_table1", "new_table2"},
			},
			wantContains: []string{
				"-- CREATE TABLE `new_table1` (...);",
				"-- CREATE TABLE `new_table2` (...);",
				"-- CREATE TABLES (present in dev but not in prod)",
				"BEGIN;",
				"COMMIT;",
			},
		},
		{
			name: "Both added and removed tables",
			diff: schema.DiffResult{
				AddedTables:   []string{"new_table"},
				RemovedTables: []string{"old_table"},
			},
			wantContains: []string{
				"-- DROP TABLE `old_table`;",
				"-- CREATE TABLE `new_table` (...);",
				"BEGIN;",
				"COMMIT;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := schema.GenerateMigration(tt.diff, "mysql", nil)
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(sql, want) {
					t.Errorf("Expected SQL to contain %q, but got:\n%s", want, sql)
				}
			}
		})
	}
}

func TestGenerateMigration_SQLite_AddColumnEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		column       schema.Column
		wantContains string
		description  string
	}{
		{
			name: "Nullable column - should work",
			column: schema.Column{
				Name:       "optional_field",
				DataType:   "TEXT",
				IsNullable: true,
			},
			wantContains: "ALTER TABLE \"test_table\" ADD COLUMN \"optional_field\" TEXT;",
			description:  "SQLite supports adding nullable columns",
		},
		{
			name: "NOT NULL column - should be commented",
			column: schema.Column{
				Name:       "required_field",
				DataType:   "INTEGER",
				IsNullable: false,
			},
			wantContains: "-- ALTER TABLE \"test_table\" ADD COLUMN \"required_field\" INTEGER NOT NULL; -- SQLite limitation",
			description:  "SQLite cannot add NOT NULL columns without default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:           "test_table",
						HasDifferences: true,
						AddedColumns:   []schema.Column{tt.column},
					},
				},
			}

			sql, err := schema.GenerateMigration(diff, "sqlite", nil)
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			if !strings.Contains(sql, tt.wantContains) {
				t.Errorf("%s\nExpected SQL to contain:\n%q\n\nBut got:\n%s", tt.description, tt.wantContains, sql)
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

			sql, err := schema.GenerateMigration(diff, "sqlite", nil)
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			if !strings.Contains(sql, tt.want) {
				t.Errorf("Expected SQL to contain %q, but got:\n%s", tt.want, sql)
			}
		})
	}
}

func TestGenerateMigration_DropColumnWithConfig(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				RemovedColumns: []schema.Column{
					{Name: "deprecated_field", DataType: "VARCHAR(255)", IsNullable: true},
					{Name: "old_column", DataType: "INT", IsNullable: false},
				},
			},
		},
	}

	tests := []struct {
		name           string
		driver         string
		opts           *schema.MigrationOptions
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:   "MySQL - AllowDropColumn=false (default, commented)",
			driver: "mysql",
			opts:   nil, // Uses safe defaults
			wantContains: []string{
				"-- DROP COLUMNS (present in prod but not in dev)",
				"-- WARNING: These operations will delete data!",
				"-- IMPORTANT: Review carefully before executing!",
				"-- ALTER TABLE `users` DROP COLUMN `deprecated_field`;",
				"-- ALTER TABLE `users` DROP COLUMN `old_column`;",
			},
			wantNotContain: nil, // The wantContains checks are sufficient
		},
		{
			name:   "MySQL - AllowDropColumn=true (uncommented)",
			driver: "mysql",
			opts: &schema.MigrationOptions{
				AllowDropColumn:    true,
				AllowDropTable:     false,
				ConfirmDestructive: true,
			},
			wantContains: []string{
				"-- DROP COLUMNS (present in prod but not in dev)",
				"-- WARNING: These operations will delete data!",
				"-- IMPORTANT: Review carefully before executing!",
				"ALTER TABLE `users` DROP COLUMN `deprecated_field`;",
				"ALTER TABLE `users` DROP COLUMN `old_column`;",
			},
			wantNotContain: []string{
				"-- ALTER TABLE `users` DROP COLUMN `deprecated_field`;",
			},
		},
		{
			name:   "PostgreSQL - AllowDropColumn=true",
			driver: "postgresql",
			opts: &schema.MigrationOptions{
				AllowDropColumn:    true,
				ConfirmDestructive: false, // No extra warnings
			},
			wantContains: []string{
				"ALTER TABLE \"users\" DROP COLUMN \"deprecated_field\";",
				"ALTER TABLE \"users\" DROP COLUMN \"old_column\";",
			},
			wantNotContain: []string{
				"-- IMPORTANT: Review carefully before executing!",
			},
		},
		{
			name:   "SQLite - AllowDropColumn=true (but still has limitation)",
			driver: "sqlite",
			opts: &schema.MigrationOptions{
				AllowDropColumn: true,
			},
			wantContains: []string{
				"-- SQLite does not support DROP COLUMN directly",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := schema.GenerateMigration(diff, tt.driver, tt.opts)
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(sql, want) {
					t.Errorf("Expected SQL to contain %q, but got:\n%s", want, sql)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(sql, notWant) {
					t.Errorf("Expected SQL to NOT contain %q, but got:\n%s", notWant, sql)
				}
			}
		})
	}
}

func TestGenerateMigration_DropTableWithConfig(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"old_table", "deprecated_table"},
	}

	tests := []struct {
		name           string
		driver         string
		opts           *schema.MigrationOptions
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:   "AllowDropTable=false (default, commented)",
			driver: "mysql",
			opts:   nil,
			wantContains: []string{
				"-- DROP TABLES (present in prod but not in dev)",
				"-- WARNING: These operations will delete data!",
				"-- DROP TABLE `old_table`;",
				"-- DROP TABLE `deprecated_table`;",
			},
			wantNotContain: nil, // The wantContains checks are sufficient
		},
		{
			name:   "AllowDropTable=true (uncommented)",
			driver: "postgresql",
			opts: &schema.MigrationOptions{
				AllowDropTable:     true,
				ConfirmDestructive: true,
			},
			wantContains: []string{
				"-- DROP TABLES (present in prod but not in dev)",
				"-- WARNING: These operations will delete data!",
				"-- IMPORTANT: Review carefully before executing!",
				"DROP TABLE \"old_table\";",
				"DROP TABLE \"deprecated_table\";",
			},
			wantNotContain: []string{
				"-- DROP TABLE \"old_table\";",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := schema.GenerateMigration(diff, tt.driver, tt.opts)
			if err != nil {
				t.Fatalf("GenerateMigration failed: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(sql, want) {
					t.Errorf("Expected SQL to contain %q, but got:\n%s", want, sql)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(sql, notWant) {
					t.Errorf("Expected SQL to NOT contain %q, but got:\n%s", notWant, sql)
				}
			}
		})
	}
}
