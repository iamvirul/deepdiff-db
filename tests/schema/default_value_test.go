package schema_test

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

func TestDefaultValue_Detection(t *testing.T) {
	tests := []struct {
		name     string
		prod     *schema.Schema
		dev      *schema.Schema
		wantDiff bool
	}{
		{
			name: "DEFAULT value changed",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"status": {
								Name:         "status",
								DataType:     "varchar(20)",
								IsNullable:   false,
								DefaultValue: stringPtr("'active'"),
							},
						},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"status": {
								Name:         "status",
								DataType:     "varchar(20)",
								IsNullable:   false,
								DefaultValue: stringPtr("'pending'"),
							},
						},
					},
				},
			},
			wantDiff: true,
		},
		{
			name: "DEFAULT value added",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"count": {
								Name:         "count",
								DataType:     "int",
								IsNullable:   false,
								DefaultValue: nil,
							},
						},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"count": {
								Name:         "count",
								DataType:     "int",
								IsNullable:   false,
								DefaultValue: stringPtr("0"),
							},
						},
					},
				},
			},
			wantDiff: true,
		},
		{
			name: "DEFAULT value removed",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"count": {
								Name:         "count",
								DataType:     "int",
								IsNullable:   false,
								DefaultValue: stringPtr("0"),
							},
						},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"count": {
								Name:         "count",
								DataType:     "int",
								IsNullable:   false,
								DefaultValue: nil,
							},
						},
					},
				},
			},
			wantDiff: true,
		},
		{
			name: "DEFAULT value unchanged",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"status": {
								Name:         "status",
								DataType:     "varchar(20)",
								IsNullable:   false,
								DefaultValue: stringPtr("'active'"),
							},
						},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"status": {
								Name:         "status",
								DataType:     "varchar(20)",
								IsNullable:   false,
								DefaultValue: stringPtr("'active'"),
							},
						},
					},
				},
			},
			wantDiff: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := schema.DiffSchemas(tt.prod, tt.dev)
			gotDiff := result.HasDrift()

			if gotDiff != tt.wantDiff {
				t.Errorf("HasDrift() = %v, want %v", gotDiff, tt.wantDiff)
			}

			if tt.wantDiff && len(result.Tables) > 0 {
				tableDiff := result.Tables[0]
				if len(tableDiff.ModifiedColumns) == 0 {
					t.Errorf("Expected modified columns, got none")
				} else {
					colDiff := tableDiff.ModifiedColumns[0]
					if !colDiff.DefaultMismatch {
						t.Errorf("Expected DefaultMismatch to be true")
					}
				}
			}
		})
	}
}

func TestDefaultValue_MySQL_Migration(t *testing.T) {
	tests := []struct {
		name     string
		colDiff  schema.ColumnDiff
		wantSQL  string
	}{
		{
			name: "Add DEFAULT value",
			colDiff: schema.ColumnDiff{
				Column:          "status",
				DefaultMismatch: true,
				TypeMismatch:    false,
				DevType:         "varchar(20)",
				ProdType:        "varchar(20)",
				DevNullable:     boolPtr(false),
				ProdDefault:     nil,
				DevDefault:      stringPtr("'active'"),
			},
			wantSQL: "ALTER TABLE `users` MODIFY COLUMN `status` varchar(20) NOT NULL DEFAULT 'active';",
		},
		{
			name: "Remove DEFAULT value",
			colDiff: schema.ColumnDiff{
				Column:          "status",
				DefaultMismatch: true,
				TypeMismatch:    false,
				DevType:         "varchar(20)",
				ProdType:        "varchar(20)",
				DevNullable:     boolPtr(false),
				ProdDefault:     stringPtr("'active'"),
				DevDefault:      nil,
			},
			wantSQL: "ALTER TABLE `users` MODIFY COLUMN `status` varchar(20) NOT NULL;",
		},
		{
			name: "Change DEFAULT value",
			colDiff: schema.ColumnDiff{
				Column:          "count",
				DefaultMismatch: true,
				TypeMismatch:    false,
				DevType:         "int",
				ProdType:        "int",
				DevNullable:     boolPtr(false),
				ProdDefault:     stringPtr("0"),
				DevDefault:      stringPtr("100"),
			},
			wantSQL: "ALTER TABLE `users` MODIFY COLUMN `count` int NOT NULL DEFAULT 100;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:            "users",
						HasDifferences:  true,
						ModifiedColumns: []schema.ColumnDiff{tt.colDiff},
					},
				},
			}

			sql, err := schema.GenerateMigration(diff, "mysql", nil)
			if err != nil {
				t.Fatalf("GenerateMigration() error = %v", err)
			}

			if !strings.Contains(sql, tt.wantSQL) {
				t.Errorf("Generated SQL does not contain expected statement.\nWant: %s\nGot: %s", tt.wantSQL, sql)
			}
		})
	}
}

func TestDefaultValue_PostgreSQL_Migration(t *testing.T) {
	tests := []struct {
		name     string
		colDiff  schema.ColumnDiff
		wantSQL  string
	}{
		{
			name: "Add DEFAULT value",
			colDiff: schema.ColumnDiff{
				Column:          "status",
				DefaultMismatch: true,
				TypeMismatch:    false,
				DevType:         "varchar(20)",
				ProdType:        "varchar(20)",
				DevNullable:     boolPtr(false),
				ProdDefault:     nil,
				DevDefault:      stringPtr("'active'"),
			},
			wantSQL: `ALTER TABLE "users" ALTER COLUMN "status" SET DEFAULT 'active';`,
		},
		{
			name: "Remove DEFAULT value",
			colDiff: schema.ColumnDiff{
				Column:          "status",
				DefaultMismatch: true,
				TypeMismatch:    false,
				DevType:         "varchar(20)",
				ProdType:        "varchar(20)",
				DevNullable:     boolPtr(false),
				ProdDefault:     stringPtr("'active'"),
				DevDefault:      nil,
			},
			wantSQL: `ALTER TABLE "users" ALTER COLUMN "status" DROP DEFAULT;`,
		},
		{
			name: "Change DEFAULT value",
			colDiff: schema.ColumnDiff{
				Column:          "count",
				DefaultMismatch: true,
				TypeMismatch:    false,
				DevType:         "int",
				ProdType:        "int",
				DevNullable:     boolPtr(false),
				ProdDefault:     stringPtr("0"),
				DevDefault:      stringPtr("100"),
			},
			wantSQL: `ALTER TABLE "users" ALTER COLUMN "count" SET DEFAULT 100;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:            "users",
						HasDifferences:  true,
						ModifiedColumns: []schema.ColumnDiff{tt.colDiff},
					},
				},
			}

			sql, err := schema.GenerateMigration(diff, "postgresql", nil)
			if err != nil {
				t.Fatalf("GenerateMigration() error = %v", err)
			}

			if !strings.Contains(sql, tt.wantSQL) {
				t.Errorf("Generated SQL does not contain expected statement.\nWant: %s\nGot: %s", tt.wantSQL, sql)
			}
		})
	}
}

func TestDefaultValue_AddColumn(t *testing.T) {
	tests := []struct {
		name    string
		driver  string
		column  schema.Column
		wantSQL string
	}{
		{
			name:   "MySQL - Add column with DEFAULT",
			driver: "mysql",
			column: schema.Column{
				Name:         "created_at",
				DataType:     "timestamp",
				IsNullable:   false,
				DefaultValue: stringPtr("CURRENT_TIMESTAMP"),
			},
			wantSQL: "ADD COLUMN `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP",
		},
		{
			name:   "PostgreSQL - Add column with DEFAULT",
			driver: "postgresql",
			column: schema.Column{
				Name:         "is_active",
				DataType:     "boolean",
				IsNullable:   false,
				DefaultValue: stringPtr("true"),
			},
			wantSQL: `ADD COLUMN "is_active" boolean NOT NULL DEFAULT true`,
		},
		{
			name:   "SQLite - Add column with DEFAULT and NOT NULL",
			driver: "sqlite",
			column: schema.Column{
				Name:         "score",
				DataType:     "integer",
				IsNullable:   false,
				DefaultValue: stringPtr("0"),
			},
			wantSQL: `ADD COLUMN "score" integer NOT NULL DEFAULT 0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := schema.DiffResult{
				Tables: []schema.TableDiff{
					{
						Name:           "users",
						HasDifferences: true,
						AddedColumns:   []schema.Column{tt.column},
					},
				},
			}

			sql, err := schema.GenerateMigration(diff, tt.driver, nil)
			if err != nil {
				t.Fatalf("GenerateMigration() error = %v", err)
			}

			if !strings.Contains(sql, tt.wantSQL) {
				t.Errorf("Generated SQL does not contain expected statement.\nWant substring: %s\nGot: %s", tt.wantSQL, sql)
			}
		})
	}
}

// Helper function for test pointers
func stringPtr(s string) *string {
	return &s
}
