package schema_test

import (
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

func TestDiffSchemas(t *testing.T) {
	tests := []struct {
		name     string
		prod     *schema.Schema
		dev      *schema.Schema
		validate func(*testing.T, schema.DiffResult)
	}{
		{
			name: "identical schemas",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":   {Name: "id", DataType: "int", IsNullable: false},
							"name": {Name: "name", DataType: "varchar", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":   {Name: "id", DataType: "int", IsNullable: false},
							"name": {Name: "name", DataType: "varchar", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			validate: func(t *testing.T, result schema.DiffResult) {
				if len(result.Tables) != 1 {
					t.Fatalf("expected 1 table, got %d", len(result.Tables))
				}
				if result.Tables[0].HasDifferences {
					t.Error("expected no differences")
				}
			},
		},
		{
			name: "table only in prod",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id": {Name: "id", DataType: "int", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{},
			},
			validate: func(t *testing.T, result schema.DiffResult) {
				if len(result.Tables) != 1 {
					t.Fatalf("expected 1 table, got %d", len(result.Tables))
				}
				td := result.Tables[0]
				if !td.HasDifferences {
					t.Error("expected differences")
				}
				if !td.OnlyInProd || !td.MissingInDev {
					t.Error("expected table to be only in prod")
				}
			},
		},
		{
			name: "table only in dev",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id": {Name: "id", DataType: "int", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			validate: func(t *testing.T, result schema.DiffResult) {
				if len(result.Tables) != 1 {
					t.Fatalf("expected 1 table, got %d", len(result.Tables))
				}
				td := result.Tables[0]
				if !td.HasDifferences {
					t.Error("expected differences")
				}
				if !td.OnlyInDev || !td.MissingInProd {
					t.Error("expected table to be only in dev")
				}
			},
		},
		{
			name: "column missing in dev",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":   {Name: "id", DataType: "int", IsNullable: false},
							"name": {Name: "name", DataType: "varchar", IsNullable: false},
							"email": {Name: "email", DataType: "varchar", IsNullable: true},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":   {Name: "id", DataType: "int", IsNullable: false},
							"name": {Name: "name", DataType: "varchar", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			validate: func(t *testing.T, result schema.DiffResult) {
				if len(result.Tables) != 1 {
					t.Fatalf("expected 1 table, got %d", len(result.Tables))
				}
				td := result.Tables[0]
				if !td.HasDifferences {
					t.Error("expected differences")
				}
				if len(td.ColumnDiffs) != 1 {
					t.Fatalf("expected 1 column diff, got %d", len(td.ColumnDiffs))
				}
				cd := td.ColumnDiffs[0]
				if cd.Column != "email" {
					t.Errorf("expected column 'email', got %s", cd.Column)
				}
				if !cd.MissingInDev {
					t.Error("expected column to be missing in dev")
				}
			},
		},
		{
			name: "column missing in prod",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":   {Name: "id", DataType: "int", IsNullable: false},
							"name": {Name: "name", DataType: "varchar", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":    {Name: "id", DataType: "int", IsNullable: false},
							"name":  {Name: "name", DataType: "varchar", IsNullable: false},
							"email": {Name: "email", DataType: "varchar", IsNullable: true},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			validate: func(t *testing.T, result schema.DiffResult) {
				if len(result.Tables) != 1 {
					t.Fatalf("expected 1 table, got %d", len(result.Tables))
				}
				td := result.Tables[0]
				if !td.HasDifferences {
					t.Error("expected differences")
				}
				if len(td.ColumnDiffs) != 1 {
					t.Fatalf("expected 1 column diff, got %d", len(td.ColumnDiffs))
				}
				cd := td.ColumnDiffs[0]
				if cd.Column != "email" {
					t.Errorf("expected column 'email', got %s", cd.Column)
				}
				if !cd.MissingInProd {
					t.Error("expected column to be missing in prod")
				}
			},
		},
		{
			name: "type mismatch",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":   {Name: "id", DataType: "int", IsNullable: false},
							"age":  {Name: "age", DataType: "int", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":   {Name: "id", DataType: "int", IsNullable: false},
							"age":  {Name: "age", DataType: "varchar", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			validate: func(t *testing.T, result schema.DiffResult) {
				if len(result.Tables) != 1 {
					t.Fatalf("expected 1 table, got %d", len(result.Tables))
				}
				td := result.Tables[0]
				if !td.HasDifferences {
					t.Error("expected differences")
				}
				if len(td.ColumnDiffs) != 1 {
					t.Fatalf("expected 1 column diff, got %d", len(td.ColumnDiffs))
				}
				cd := td.ColumnDiffs[0]
				if !cd.TypeMismatch {
					t.Error("expected type mismatch")
				}
				if cd.ProdType != "int" || cd.DevType != "varchar" {
					t.Errorf("expected prod=int dev=varchar, got prod=%s dev=%s", cd.ProdType, cd.DevType)
				}
			},
		},
		{
			name: "nullable mismatch",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":   {Name: "id", DataType: "int", IsNullable: false},
							"name": {Name: "name", DataType: "varchar", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id":   {Name: "id", DataType: "int", IsNullable: false},
							"name": {Name: "name", DataType: "varchar", IsNullable: true},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			validate: func(t *testing.T, result schema.DiffResult) {
				if len(result.Tables) != 1 {
					t.Fatalf("expected 1 table, got %d", len(result.Tables))
				}
				td := result.Tables[0]
				if !td.HasDifferences {
					t.Error("expected differences")
				}
				if len(td.ColumnDiffs) != 1 {
					t.Fatalf("expected 1 column diff, got %d", len(td.ColumnDiffs))
				}
				cd := td.ColumnDiffs[0]
				if !cd.NullableMismatch {
					t.Error("expected nullable mismatch")
				}
				if cd.ProdNullable == nil || *cd.ProdNullable != false {
					t.Error("expected prod nullable to be false")
				}
				if cd.DevNullable == nil || *cd.DevNullable != true {
					t.Error("expected dev nullable to be true")
				}
			},
		},
		{
			name: "multiple tables with various differences",
			prod: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id": {Name: "id", DataType: "int", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
					"posts": {
						Name: "posts",
						Columns: map[string]schema.Column{
							"id": {Name: "id", DataType: "int", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			dev: &schema.Schema{
				Tables: map[string]schema.Table{
					"users": {
						Name: "users",
						Columns: map[string]schema.Column{
							"id": {Name: "id", DataType: "int", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
					"comments": {
						Name: "comments",
						Columns: map[string]schema.Column{
							"id": {Name: "id", DataType: "int", IsNullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
			},
			validate: func(t *testing.T, result schema.DiffResult) {
				if len(result.Tables) != 3 {
					t.Fatalf("expected 3 tables, got %d", len(result.Tables))
				}
				// Should have posts only in prod, comments only in dev, users identical
				foundPosts := false
				foundComments := false
				foundUsers := false
				for _, td := range result.Tables {
					switch td.Table {
					case "posts":
						foundPosts = true
						if !td.OnlyInProd {
							t.Error("posts should be only in prod")
						}
					case "comments":
						foundComments = true
						if !td.OnlyInDev {
							t.Error("comments should be only in dev")
						}
					case "users":
						foundUsers = true
						if td.HasDifferences {
							t.Error("users should have no differences")
						}
					}
				}
				if !foundPosts || !foundComments || !foundUsers {
					t.Error("missing expected tables in diff result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := schema.DiffSchemas(tt.prod, tt.dev)
			tt.validate(t, result)
		})
	}
}

func TestHasDrift(t *testing.T) {
	tests := []struct {
		name     string
		result   schema.DiffResult
		expected bool
	}{
		{
			name: "no drift",
			result: schema.DiffResult{
				Tables: []schema.TableDiff{
					{Name: "users", Table: "users", HasDifferences: false},
				},
			},
			expected: false,
		},
		{
			name: "has drift",
			result: schema.DiffResult{
				Tables: []schema.TableDiff{
					{Name: "users", Table: "users", HasDifferences: true},
				},
			},
			expected: true,
		},
		{
			name: "mixed",
			result: schema.DiffResult{
				Tables: []schema.TableDiff{
					{Name: "users", Table: "users", HasDifferences: false},
					{Name: "posts", Table: "posts", HasDifferences: true},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasDrift(); got != tt.expected {
				t.Errorf("HasDrift() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"INT", "int"},
		{"  VARCHAR  ", "varchar"},
		{"INTEGER", "integer"},
		{"text", "text"},
		{"  TEXT  ", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := schema.NormalizeType(tt.input)
			if got != tt.expected {
				t.Errorf("schema.NormalizeType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

