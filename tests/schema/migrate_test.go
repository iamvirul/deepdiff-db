package schema_test

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

func TestGenerateMigration_MySQL(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "new_table",
				Columns: map[string]schema.Column{
					"id":   {Name: "id", DataType: "INT", IsNullable: false},
					"name": {Name: "name", DataType: "VARCHAR(100)", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
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

	// Check CREATE TABLE statement is generated (not commented)
	if !strings.Contains(sql, "CREATE TABLE `new_table`") {
		t.Error("Expected CREATE TABLE statement")
	}
	if !strings.Contains(sql, "`id` INT NOT NULL") {
		t.Error("Expected id column in CREATE TABLE")
	}
	if !strings.Contains(sql, "`name` VARCHAR(100)") {
		t.Error("Expected name column in CREATE TABLE")
	}
	if !strings.Contains(sql, "PRIMARY KEY (`id`)") {
		t.Error("Expected PRIMARY KEY in CREATE TABLE")
	}
	if !strings.Contains(sql, "ENGINE=InnoDB") {
		t.Error("Expected MySQL ENGINE option")
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

	_, err := schema.GenerateMigration(diff, "db2", nil)
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
		name           string
		colDiff        schema.ColumnDiff
		wantContains   []string
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
				AddedTables: []schema.Table{
					{
						Name: "new_table1",
						Columns: map[string]schema.Column{
							"id": {Name: "id", DataType: "INT", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
					{
						Name: "new_table2",
						Columns: map[string]schema.Column{
							"id": {Name: "id", DataType: "INT", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			wantContains: []string{
				"CREATE TABLE `new_table1`",
				"CREATE TABLE `new_table2`",
				"-- CREATE TABLES (present in dev but not in prod)",
				"BEGIN;",
				"COMMIT;",
			},
		},
		{
			name: "Both added and removed tables",
			diff: schema.DiffResult{
				AddedTables: []schema.Table{
					{
						Name: "new_table",
						Columns: map[string]schema.Column{
							"id": {Name: "id", DataType: "INT", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
				RemovedTables: []string{"old_table"},
			},
			wantContains: []string{
				"-- DROP TABLE `old_table`;",
				"CREATE TABLE `new_table`",
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

// ============================================================================
// CREATE TABLE Tests
// ============================================================================

func TestGenerateCreateTable_MySQL(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "users",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "INT", IsNullable: false},
					"email":      {Name: "email", DataType: "VARCHAR(255)", IsNullable: false},
					"name":       {Name: "name", DataType: "VARCHAR(100)", IsNullable: true},
					"created_at": {Name: "created_at", DataType: "TIMESTAMP", IsNullable: false, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE `users`") {
		t.Error("Expected CREATE TABLE statement")
	}
	if !strings.Contains(sql, "`id` INT NOT NULL") {
		t.Error("Expected id column definition")
	}
	if !strings.Contains(sql, "`email` VARCHAR(255) NOT NULL") {
		t.Error("Expected email column definition")
	}
	if !strings.Contains(sql, "PRIMARY KEY (`id`)") {
		t.Error("Expected PRIMARY KEY constraint")
	}
	if !strings.Contains(sql, "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4") {
		t.Error("Expected MySQL table options")
	}
}

func TestGenerateCreateTable_PostgreSQL(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "products",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "SERIAL", IsNullable: false},
					"name":  {Name: "name", DataType: "VARCHAR(200)", IsNullable: false},
					"price": {Name: "price", DataType: "DECIMAL(10,2)", IsNullable: false, DefaultValue: strPtr("0.00")},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE \"products\"") {
		t.Error("Expected CREATE TABLE with double quotes")
	}
	if !strings.Contains(sql, "\"id\" SERIAL NOT NULL") {
		t.Error("Expected id column definition")
	}
	if !strings.Contains(sql, "PRIMARY KEY (\"id\")") {
		t.Error("Expected PRIMARY KEY constraint")
	}
	if strings.Contains(sql, "ENGINE=") {
		t.Error("PostgreSQL should not have ENGINE option")
	}
}

func TestGenerateCreateTable_SQLite(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "settings",
				Columns: map[string]schema.Column{
					"key":   {Name: "key", DataType: "TEXT", IsNullable: false},
					"value": {Name: "value", DataType: "TEXT", IsNullable: true},
				},
				PrimaryKey: []string{"key"},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE \"settings\"") {
		t.Error("Expected CREATE TABLE with double quotes")
	}
	if !strings.Contains(sql, "\"key\" TEXT NOT NULL") {
		t.Error("Expected key column definition")
	}
	if !strings.Contains(sql, "PRIMARY KEY (\"key\")") {
		t.Error("Expected PRIMARY KEY constraint")
	}
}

func TestGenerateCreateTable_WithIndexes(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "orders",
				Columns: map[string]schema.Column{
					"id":          {Name: "id", DataType: "INT", IsNullable: false},
					"customer_id": {Name: "customer_id", DataType: "INT", IsNullable: false},
					"status":      {Name: "status", DataType: "VARCHAR(50)", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
				Indexes: map[string]schema.Index{
					"idx_orders_customer": {Name: "idx_orders_customer", Columns: []string{"customer_id"}, IsUnique: false},
					"idx_orders_status":   {Name: "idx_orders_status", Columns: []string{"status"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE `orders`") {
		t.Error("Expected CREATE TABLE statement")
	}
	if !strings.Contains(sql, "CREATE INDEX `idx_orders_customer` ON `orders`") {
		t.Error("Expected CREATE INDEX for customer_id")
	}
	if !strings.Contains(sql, "CREATE INDEX `idx_orders_status` ON `orders`") {
		t.Error("Expected CREATE INDEX for status")
	}
}

func TestGenerateCreateTable_CompositePrimaryKey(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "order_items",
				Columns: map[string]schema.Column{
					"order_id":   {Name: "order_id", DataType: "INT", IsNullable: false},
					"product_id": {Name: "product_id", DataType: "INT", IsNullable: false},
					"quantity":   {Name: "quantity", DataType: "INT", IsNullable: false, DefaultValue: strPtr("1")},
				},
				PrimaryKey: []string{"order_id", "product_id"},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "PRIMARY KEY (`order_id`, `product_id`)") {
		t.Error("Expected composite PRIMARY KEY")
	}
}

func TestGenerateCreateTable_WithDefaultValues(t *testing.T) {
	defaultStatus := "active"
	defaultCount := "0"

	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "items",
				Columns: map[string]schema.Column{
					"id":     {Name: "id", DataType: "INT", IsNullable: false},
					"status": {Name: "status", DataType: "VARCHAR(20)", IsNullable: false, DefaultValue: &defaultStatus},
					"count":  {Name: "count", DataType: "INT", IsNullable: false, DefaultValue: &defaultCount},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "DEFAULT 'active'") {
		t.Error("Expected quoted string default")
	}
	if !strings.Contains(sql, "DEFAULT 0") {
		t.Error("Expected unquoted numeric default")
	}
}

func TestGenerateCreateTable_NoPrimaryKey(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "logs",
				Columns: map[string]schema.Column{
					"message":    {Name: "message", DataType: "TEXT", IsNullable: true},
					"created_at": {Name: "created_at", DataType: "TIMESTAMP", IsNullable: false},
				},
				PrimaryKey: nil,
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE `logs`") {
		t.Error("Expected CREATE TABLE statement")
	}
	if strings.Contains(sql, "PRIMARY KEY") {
		t.Error("Should not have PRIMARY KEY when not defined")
	}
}

// ============================================================================
// DROP TABLE Tests
// ============================================================================

func TestGenerateDropTable_MySQL_Commented(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"old_table"},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "-- DROP TABLE `old_table`;") {
		t.Error("Expected commented DROP TABLE statement")
	}
	if !strings.Contains(sql, "-- WARNING: These operations will delete data!") {
		t.Error("Expected warning message")
	}
}

func TestGenerateDropTable_MySQL_Enabled(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"old_table"},
	}

	opts := &schema.MigrationOptions{AllowDropTable: true}
	sql, err := schema.GenerateMigration(diff, "mysql", opts)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if strings.Contains(sql, "-- DROP TABLE `old_table`;") {
		t.Error("DROP TABLE should not be commented when AllowDropTable=true")
	}
	if !strings.Contains(sql, "DROP TABLE `old_table`;") {
		t.Error("Expected uncommented DROP TABLE statement")
	}
}

func TestGenerateDropTable_PostgreSQL(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"legacy_data"},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "-- DROP TABLE \"legacy_data\";") {
		t.Error("Expected commented DROP TABLE with double quotes")
	}
}

func TestGenerateDropTable_SQLite(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"temp_data"},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "-- DROP TABLE \"temp_data\";") {
		t.Error("Expected commented DROP TABLE with double quotes")
	}
}

func TestGenerateDropTable_MultipleTables(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTables: []string{"old_table1", "old_table2", "old_table3"},
	}

	opts := &schema.MigrationOptions{AllowDropTable: true}
	sql, err := schema.GenerateMigration(diff, "mysql", opts)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "DROP TABLE `old_table1`;") {
		t.Error("Expected DROP TABLE for old_table1")
	}
	if !strings.Contains(sql, "DROP TABLE `old_table2`;") {
		t.Error("Expected DROP TABLE for old_table2")
	}
	if !strings.Contains(sql, "DROP TABLE `old_table3`;") {
		t.Error("Expected DROP TABLE for old_table3")
	}
}

func TestGenerateMigration_CreateAndDropTables(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "new_users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "INT", IsNullable: false},
					"email": {Name: "email", DataType: "VARCHAR(255)", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
		RemovedTables: []string{"old_users"},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE `new_users`") {
		t.Error("Expected CREATE TABLE statement")
	}
	if !strings.Contains(sql, "-- DROP TABLE `old_users`;") {
		t.Error("Expected commented DROP TABLE statement")
	}

	dropPos := strings.Index(sql, "DROP TABLE")
	createPos := strings.Index(sql, "CREATE TABLE")
	if dropPos > createPos {
		t.Error("DROP TABLE should appear before CREATE TABLE")
	}
}

func strPtr(s string) *string {
	return &s
}

// ---------------------------------------------------------------------------
// MSSQL-specific migration tests
// ---------------------------------------------------------------------------

func TestGenerateMigration_MSSQL(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "orders",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "INT", IsNullable: false},
					"customer":   {Name: "customer", DataType: "NVARCHAR(100)", IsNullable: false},
					"total":      {Name: "total", DataType: "DECIMAL(10,2)", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
		RemovedTables: []string{"legacy_log"},
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedColumns: []schema.Column{
					{Name: "phone", DataType: "NVARCHAR(20)", IsNullable: true},
					{Name: "score", DataType: "INT", IsNullable: false},
				},
				RemovedColumns: []schema.Column{
					{Name: "old_code", DataType: "INT", IsNullable: true},
				},
				ModifiedColumns: []schema.ColumnDiff{
					{
						Column:           "name",
						TypeMismatch:     true,
						DevType:          "NVARCHAR(200)",
						ProdType:         "NVARCHAR(100)",
						NullableMismatch: false,
						DevNullable:      boolPtr(false),
						ProdNullable:     boolPtr(false),
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mssql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// MSSQL transaction syntax
	if !strings.Contains(sql, "BEGIN;") {
		t.Error("Expected migration to start with BEGIN;")
	}
	if !strings.Contains(sql, "COMMIT;") {
		t.Error("Expected migration to end with COMMIT;")
	}

	// CREATE TABLE uses square-bracket quoting
	if !strings.Contains(sql, "CREATE TABLE [orders]") {
		t.Errorf("Expected CREATE TABLE [orders], got:\n%s", sql)
	}
	if !strings.Contains(sql, "[id] INT NOT NULL") {
		t.Error("Expected [id] column with square-bracket quoting")
	}
	if !strings.Contains(sql, "PRIMARY KEY ([id])") {
		t.Error("Expected PRIMARY KEY with square-bracket quoting")
	}

	// DROP TABLE is commented
	if !strings.Contains(sql, "-- DROP TABLE [legacy_log];") {
		t.Errorf("Expected commented DROP TABLE [legacy_log], got:\n%s", sql)
	}

	// ADD column (no COLUMN keyword for MSSQL)
	if !strings.Contains(sql, "ALTER TABLE [users] ADD [phone] NVARCHAR(20) NULL;") {
		t.Errorf("Expected ADD without COLUMN keyword, got:\n%s", sql)
	}
	// NOT NULL without default → commented with guidance
	if !strings.Contains(sql, "-- ALTER TABLE [users] ADD [score] INT NOT NULL;") {
		t.Errorf("Expected commented NOT NULL ADD, got:\n%s", sql)
	}

	// DROP COLUMN is commented
	if !strings.Contains(sql, "-- ALTER TABLE [users] DROP COLUMN [old_code];") {
		t.Errorf("Expected commented DROP COLUMN [old_code], got:\n%s", sql)
	}

	// MODIFY uses ALTER COLUMN (not MODIFY COLUMN)
	if !strings.Contains(sql, "ALTER TABLE [users] ALTER COLUMN [name] NVARCHAR(200) NOT NULL;") {
		t.Errorf("Expected ALTER COLUMN syntax, got:\n%s", sql)
	}
}

func TestQuoteIdentifier_MSSQL(t *testing.T) {
	tests := []struct {
		name       string
		diff       schema.DiffResult
		wantHas    []string
		wantNotHas []string
	}{
		{
			name: "simple identifier gets square brackets",
			diff: schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:           "users",
						HasDifferences: true,
						AddedColumns: []schema.Column{
							{Name: "email", DataType: "NVARCHAR(255)", IsNullable: true},
						},
					},
				},
			},
			wantHas: []string{
				"ALTER TABLE [users] ADD [email] NVARCHAR(255) NULL;",
			},
			wantNotHas: []string{"`users`", `"users"`},
		},
		{
			name: "drop column uses square brackets",
			diff: schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:           "products",
						HasDifferences: true,
						RemovedColumns: []schema.Column{
							{Name: "old_price", DataType: "DECIMAL(10,2)", IsNullable: true},
						},
					},
				},
			},
			wantHas: []string{
				"-- ALTER TABLE [products] DROP COLUMN [old_price];",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := schema.GenerateMigration(tt.diff, "mssql", nil)
			if err != nil {
				t.Fatalf("GenerateMigration error: %v", err)
			}
			for _, want := range tt.wantHas {
				if !strings.Contains(sql, want) {
					t.Errorf("expected %q in output, got:\n%s", want, sql)
				}
			}
			for _, notWant := range tt.wantNotHas {
				if strings.Contains(sql, notWant) {
					t.Errorf("unexpected %q in output, got:\n%s", notWant, sql)
				}
			}
		})
	}
}

func TestGenerateMigration_MSSQL_ModifyColumn(t *testing.T) {
	tests := []struct {
		name    string
		colDiff schema.ColumnDiff
		wantHas []string
	}{
		{
			name: "type change only",
			colDiff: schema.ColumnDiff{
				Column:       "description",
				TypeMismatch: true,
				DevType:      "NVARCHAR(MAX)",
				ProdType:     "VARCHAR(255)",
				DevNullable:  boolPtr(true),
				ProdNullable: boolPtr(false),
			},
			wantHas: []string{
				"ALTER TABLE [t] ALTER COLUMN [description] NVARCHAR(MAX) NULL;",
			},
		},
		{
			name: "nullable change only",
			colDiff: schema.ColumnDiff{
				Column:           "active",
				NullableMismatch: true,
				DevNullable:      boolPtr(false),
				ProdNullable:     boolPtr(true),
				DevType:          "BIT",
			},
			wantHas: []string{
				"ALTER TABLE [t] ALTER COLUMN [active] BIT NOT NULL;",
			},
		},
		{
			name: "default change emits comment",
			colDiff: schema.ColumnDiff{
				Column:          "status",
				DefaultMismatch: true,
				DevDefault:      strPtr("'active'"),
			},
			wantHas: []string{
				"-- MSSQL: to change DEFAULT on t.status",
				"-- ALTER TABLE [t] ADD DEFAULT 'active' FOR [status];",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:            "t",
						HasDifferences:  true,
						ModifiedColumns: []schema.ColumnDiff{tt.colDiff},
					},
				},
			}
			sql, err := schema.GenerateMigration(diff, "mssql", nil)
			if err != nil {
				t.Fatalf("GenerateMigration error: %v", err)
			}
			for _, want := range tt.wantHas {
				if !strings.Contains(sql, want) {
					t.Errorf("expected %q in output, got:\n%s", want, sql)
				}
			}
		})
	}
}

func TestGenerateMigration_MSSQL_DropIndex(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "products",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_products_sku", Columns: []string{"sku"}, IsUnique: false},
					{Name: "uq_products_code", Columns: []string{"code"}, IsUnique: true},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mssql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}

	// MSSQL DROP INDEX requires table name: DROP INDEX idx ON table
	if !strings.Contains(sql, "DROP INDEX [idx_products_sku] ON [products];") {
		t.Errorf("expected DROP INDEX with table name, got:\n%s", sql)
	}
	if !strings.Contains(sql, "DROP INDEX [uq_products_code] ON [products];") {
		t.Errorf("expected DROP INDEX for unique index, got:\n%s", sql)
	}
}

func TestGenerateMigration_MSSQL_DropForeignKey(t *testing.T) {
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

	sql, err := schema.GenerateMigration(diff, "mssql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}

	// MSSQL uses DROP CONSTRAINT (not DROP FOREIGN KEY)
	if !strings.Contains(sql, "ALTER TABLE [orders] DROP CONSTRAINT [fk_orders_customer];") {
		t.Errorf("expected DROP CONSTRAINT syntax, got:\n%s", sql)
	}
	if strings.Contains(sql, "ALTER TABLE [orders] DROP FOREIGN KEY") {
		t.Errorf("MSSQL should not use DROP FOREIGN KEY, got:\n%s", sql)
	}
}

func TestGenerateMigration_MSSQL_ModifyPrimaryKey(t *testing.T) {
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

	cfg := &schema.MigrationOptions{AllowModifyPrimaryKey: true}
	sql, err := schema.GenerateMigration(diff, "mssql", cfg)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}

	// Drop uses PK_tablename convention
	if !strings.Contains(sql, "ALTER TABLE [items] DROP CONSTRAINT PK_items;") {
		t.Errorf("expected DROP CONSTRAINT PK_items, got:\n%s", sql)
	}
	// Add new PK
	if !strings.Contains(sql, "ALTER TABLE [items] ADD PRIMARY KEY ([id], [tenant_id]);") {
		t.Errorf("expected ADD PRIMARY KEY with new columns, got:\n%s", sql)
	}
	// Should include note about constraint name
	if !strings.Contains(sql, "-- NOTE: Replace 'PK_tablename'") {
		t.Errorf("expected placeholder note, got:\n%s", sql)
	}
}

func TestGenerateCreateTable_MSSQL(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "sessions",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "UNIQUEIDENTIFIER", IsNullable: false},
					"user_id":    {Name: "user_id", DataType: "INT", IsNullable: false},
					"created_at": {Name: "created_at", DataType: "DATETIME2", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
				Indexes: map[string]schema.Index{
					"idx_sessions_user": {Name: "idx_sessions_user", Columns: []string{"user_id"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mssql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}

	// MSSQL CREATE TABLE syntax
	if !strings.Contains(sql, "CREATE TABLE [sessions]") {
		t.Errorf("expected CREATE TABLE [sessions], got:\n%s", sql)
	}
	if !strings.Contains(sql, "[id] UNIQUEIDENTIFIER NOT NULL") {
		t.Errorf("expected [id] column, got:\n%s", sql)
	}
	// No ENGINE=InnoDB for MSSQL
	if strings.Contains(sql, "ENGINE=InnoDB") {
		t.Errorf("MSSQL should not include ENGINE=InnoDB, got:\n%s", sql)
	}
	// Index creation uses square brackets
	if !strings.Contains(sql, "CREATE INDEX [idx_sessions_user] ON [sessions] ([user_id])") {
		t.Errorf("expected CREATE INDEX with square brackets, got:\n%s", sql)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Oracle-specific migration generation tests
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateMigration_Oracle_AddDropColumn(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				HasDifferences: true,
				AddedColumns: []schema.Column{
					{Name: "phone", DataType: "VARCHAR2(30)", IsNullable: true},
				},
				RemovedColumns: []schema.Column{
					{Name: "old_code", DataType: "VARCHAR2(10)", IsNullable: true},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "oracle", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Oracle ADD column: no COLUMN keyword
	if !strings.Contains(sql, `ALTER TABLE "users" ADD "phone" VARCHAR2(30) NULL;`) {
		t.Errorf("expected Oracle ADD without COLUMN keyword, got:\n%s", sql)
	}

	// Oracle DROP COLUMN: requires COLUMN keyword
	if !strings.Contains(sql, `-- ALTER TABLE "users" DROP COLUMN "old_code";`) {
		t.Errorf("expected commented DROP COLUMN with COLUMN keyword, got:\n%s", sql)
	}

	// Must not use MSSQL/MySQL quoting
	if strings.Contains(sql, "[users]") || strings.Contains(sql, "`users`") {
		t.Errorf("Oracle should use double-quote identifiers, got:\n%s", sql)
	}
}

func TestGenerateMigration_Oracle_ModifyColumn(t *testing.T) {
	tests := []struct {
		name    string
		colDiff schema.ColumnDiff
		wantHas []string
	}{
		{
			name: "type change uses MODIFY",
			colDiff: schema.ColumnDiff{
				Column:       "description",
				TypeMismatch: true,
				DevType:      "VARCHAR2(500)",
				ProdType:     "VARCHAR2(200)",
				DevNullable:  boolPtr(true),
				ProdNullable: boolPtr(true),
			},
			wantHas: []string{
				`ALTER TABLE "t" MODIFY "description" VARCHAR2(500) NULL;`,
			},
		},
		{
			name: "nullable change uses MODIFY",
			colDiff: schema.ColumnDiff{
				Column:           "active",
				NullableMismatch: true,
				DevNullable:      boolPtr(false),
				ProdNullable:     boolPtr(true),
				DevType:          "NUMBER(1)",
			},
			wantHas: []string{
				`ALTER TABLE "t" MODIFY "active" NUMBER(1) NOT NULL;`,
			},
		},
		{
			name: "default change emits Oracle-specific comment",
			colDiff: schema.ColumnDiff{
				Column:          "status",
				DefaultMismatch: true,
				DevDefault:      strPtr("'active'"),
			},
			wantHas: []string{
				`-- Oracle: to change DEFAULT on t.status, use MODIFY with the new default.`,
				`ALTER TABLE "t" MODIFY "status" DEFAULT 'active';`,
			},
		},
		{
			// Covers the `if dataType == ""` fallback to ProdType branch (line 488-490)
			name: "type mismatch with empty DevType falls back to ProdType",
			colDiff: schema.ColumnDiff{
				Column:       "code",
				TypeMismatch: true,
				DevType:      "",              // empty — must fall back to ProdType
				ProdType:     "VARCHAR2(100)",
				DevNullable:  boolPtr(true),
				ProdNullable: boolPtr(true),
			},
			wantHas: []string{
				`ALTER TABLE "t" MODIFY "code" VARCHAR2(100) NULL;`,
			},
		},
		{
			// Covers the `else if colDiff.ProdNullable != nil` branch (line 494-496)
			name: "nullable mismatch with nil DevNullable uses ProdNullable",
			colDiff: schema.ColumnDiff{
				Column:           "active",
				NullableMismatch: true,
				DevType:          "NUMBER(1)",
				DevNullable:      nil,            // not set
				ProdNullable:     boolPtr(false),  // falls through to ProdNullable
			},
			wantHas: []string{
				`ALTER TABLE "t" MODIFY "active" NUMBER(1) NOT NULL;`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:            "t",
						HasDifferences:  true,
						ModifiedColumns: []schema.ColumnDiff{tt.colDiff},
					},
				},
			}
			sql, err := schema.GenerateMigration(diff, "oracle", nil)
			if err != nil {
				t.Fatalf("GenerateMigration error: %v", err)
			}
			for _, want := range tt.wantHas {
				if !strings.Contains(sql, want) {
					t.Errorf("expected %q in output, got:\n%s", want, sql)
				}
			}
		})
	}
}

func TestGenerateMigration_Oracle_DropIndex(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "products",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_products_sku", Columns: []string{"sku"}, IsUnique: false},
					{Name: "uq_products_code", Columns: []string{"code"}, IsUnique: true},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "oracle", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Oracle DROP INDEX: no table name (unlike MSSQL which requires ON table)
	if !strings.Contains(sql, `DROP INDEX "idx_products_sku";`) {
		t.Errorf("expected Oracle DROP INDEX without table name, got:\n%s", sql)
	}
	if !strings.Contains(sql, `DROP INDEX "uq_products_code";`) {
		t.Errorf("expected Oracle DROP INDEX for unique index, got:\n%s", sql)
	}
	// Must not include ON clause
	if strings.Contains(sql, `ON "products"`) {
		t.Errorf("Oracle DROP INDEX must not include ON table, got:\n%s", sql)
	}
}

func TestGenerateMigration_Oracle_DropForeignKey(t *testing.T) {
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

	sql, err := schema.GenerateMigration(diff, "oracle", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Oracle uses DROP CONSTRAINT (same as MSSQL, not DROP FOREIGN KEY like MySQL)
	if !strings.Contains(sql, `ALTER TABLE "orders" DROP CONSTRAINT "fk_orders_customer";`) {
		t.Errorf("expected DROP CONSTRAINT syntax for Oracle, got:\n%s", sql)
	}
	// MySQL-style DROP FOREIGN KEY uses backtick quoting — must not appear for Oracle
	if strings.Contains(sql, "DROP FOREIGN KEY `") {
		t.Errorf("Oracle should not use MySQL DROP FOREIGN KEY syntax, got:\n%s", sql)
	}
}

func TestGenerateMigration_Oracle_ModifyPrimaryKey(t *testing.T) {
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

	cfg := &schema.MigrationOptions{AllowModifyPrimaryKey: true}
	sql, err := schema.GenerateMigration(diff, "oracle", cfg)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Oracle: DROP PRIMARY KEY (no constraint name needed)
	if !strings.Contains(sql, `ALTER TABLE "items" DROP PRIMARY KEY;`) {
		t.Errorf("expected Oracle DROP PRIMARY KEY, got:\n%s", sql)
	}
	// Add new composite PK
	if !strings.Contains(sql, `ALTER TABLE "items" ADD PRIMARY KEY ("id", "tenant_id");`) {
		t.Errorf("expected ADD PRIMARY KEY with new columns, got:\n%s", sql)
	}
}

func TestGenerateCreateTable_Oracle(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "sessions",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "NUMBER", IsNullable: false},
					"user_id":    {Name: "user_id", DataType: "NUMBER", IsNullable: false},
					"created_at": {Name: "created_at", DataType: "TIMESTAMP", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
				Indexes: map[string]schema.Index{
					"idx_sessions_user": {Name: "idx_sessions_user", Columns: []string{"user_id"}, IsUnique: false},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "oracle", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Oracle CREATE TABLE with double-quote identifiers
	if !strings.Contains(sql, `CREATE TABLE "sessions"`) {
		t.Errorf("expected CREATE TABLE with double-quote identifiers, got:\n%s", sql)
	}
	if !strings.Contains(sql, `"id" NUMBER NOT NULL`) {
		t.Errorf("expected \"id\" column definition, got:\n%s", sql)
	}
	// No ENGINE=InnoDB for Oracle
	if strings.Contains(sql, "ENGINE=InnoDB") {
		t.Errorf("Oracle should not include ENGINE=InnoDB, got:\n%s", sql)
	}
	// Index creation with double-quote identifiers
	if !strings.Contains(sql, `CREATE INDEX "idx_sessions_user" ON "sessions" ("user_id")`) {
		t.Errorf("expected CREATE INDEX with double-quote identifiers, got:\n%s", sql)
	}
}

func TestGenerateMigration_Oracle_ModifyColumn_NilDefault(t *testing.T) {
	// When DevDefault is nil and DefaultMismatch is true, Oracle should emit MODIFY col DEFAULT NULL
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "t",
				HasDifferences: true,
				ModifiedColumns: []schema.ColumnDiff{
					{
						Column:          "status",
						DefaultMismatch: true,
						DevDefault:      nil, // removing the default
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "oracle", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, `-- Oracle: to change DEFAULT on t.status, use MODIFY with the new default.`) {
		t.Errorf("expected Oracle DEFAULT comment, got:\n%s", sql)
	}
	if !strings.Contains(sql, `ALTER TABLE "t" MODIFY "status" DEFAULT NULL;`) {
		t.Errorf("expected Oracle MODIFY DEFAULT NULL when DevDefault is nil, got:\n%s", sql)
	}
}

func TestGenerateMigration_Oracle_ModifyPrimaryKey_Blocked(t *testing.T) {
	// When AllowModifyPrimaryKey is false, Oracle statements should be commented out
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

	// nil options → AllowModifyPrimaryKey defaults to false
	sql, err := schema.GenerateMigration(diff, "oracle", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Both statements must be commented out
	if !strings.Contains(sql, `-- ALTER TABLE "items" DROP PRIMARY KEY;`) {
		t.Errorf("Oracle DROP PRIMARY KEY should be commented when not allowed, got:\n%s", sql)
	}
	if !strings.Contains(sql, `-- ALTER TABLE "items" ADD PRIMARY KEY ("id", "tenant_id");`) {
		t.Errorf("Oracle ADD PRIMARY KEY should be commented when not allowed, got:\n%s", sql)
	}
	// Must not have uncommented DROP/ADD
	if strings.Contains(sql, "\nALTER TABLE \"items\" DROP PRIMARY KEY;") {
		t.Errorf("Oracle DROP PRIMARY KEY must not be uncommented when blocked, got:\n%s", sql)
	}
}
