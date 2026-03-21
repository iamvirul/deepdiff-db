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
							"id":    {Name: "id", DataType: "int", IsNullable: false},
							"name":  {Name: "name", DataType: "varchar", IsNullable: false},
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
							"id":  {Name: "id", DataType: "int", IsNullable: false},
							"age": {Name: "age", DataType: "int", IsNullable: false},
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
							"id":  {Name: "id", DataType: "int", IsNullable: false},
							"age": {Name: "age", DataType: "varchar", IsNullable: false},
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

// ============================================================================
// diffIndexes coverage — exercised via DiffSchemas
// ============================================================================

func TestDiffSchemas_IndexDiffs(t *testing.T) {
	prodIndexes := map[string]schema.Index{
		"idx_email":    {Name: "idx_email", Columns: []string{"email"}, IsUnique: true},
		"idx_name":     {Name: "idx_name", Columns: []string{"name"}, IsUnique: false},
		"idx_prod_only": {Name: "idx_prod_only", Columns: []string{"age"}, IsUnique: false},
	}
	devIndexes := map[string]schema.Index{
		"idx_email":   {Name: "idx_email", Columns: []string{"email", "created_at"}, IsUnique: true}, // columns changed
		"idx_name":    {Name: "idx_name", Columns: []string{"name"}, IsUnique: true},                 // unique changed
		"idx_dev_only": {Name: "idx_dev_only", Columns: []string{"status"}, IsUnique: false},
	}

	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: prodIndexes,
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:    "users",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				Indexes: devIndexes,
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	if len(result.Tables) != 1 {
		t.Fatalf("expected 1 table diff, got %d", len(result.Tables))
	}

	td := result.Tables[0]
	if !td.HasDifferences {
		t.Error("expected differences")
	}

	// Added: idx_dev_only
	if len(td.AddedIndexes) != 1 {
		t.Errorf("expected 1 added index, got %d", len(td.AddedIndexes))
	} else if td.AddedIndexes[0].Name != "idx_dev_only" {
		t.Errorf("expected idx_dev_only to be added, got %s", td.AddedIndexes[0].Name)
	}

	// Removed: idx_prod_only
	if len(td.RemovedIndexes) != 1 {
		t.Errorf("expected 1 removed index, got %d", len(td.RemovedIndexes))
	} else if td.RemovedIndexes[0].Name != "idx_prod_only" {
		t.Errorf("expected idx_prod_only to be removed, got %s", td.RemovedIndexes[0].Name)
	}

	// Modified: idx_email (columns differ) and idx_name (unique differs)
	if len(td.ModifiedIndexes) != 2 {
		t.Errorf("expected 2 modified indexes, got %d", len(td.ModifiedIndexes))
	}

	for _, idx := range td.ModifiedIndexes {
		switch idx.Name {
		case "idx_email":
			if !idx.ColumnsDiffer {
				t.Error("expected idx_email columns to differ")
			}
		case "idx_name":
			if !idx.UniqueDiffers {
				t.Error("expected idx_name unique to differ")
			}
		default:
			t.Errorf("unexpected modified index: %s", idx.Name)
		}
	}
}

// ============================================================================
// diffForeignKeys coverage — exercised via DiffSchemas
// ============================================================================

func TestDiffSchemas_ForeignKeyDiffs(t *testing.T) {
	prodFKs := map[string]schema.ForeignKey{
		"fk_prod_only": {
			Name:              "fk_prod_only",
			Columns:           []string{"user_id"},
			ReferencedTable:   "users",
			ReferencedColumns: []string{"id"},
		},
		"fk_columns_change": {
			Name:              "fk_columns_change",
			Columns:           []string{"a_id"},
			ReferencedTable:   "a",
			ReferencedColumns: []string{"id"},
		},
		"fk_ref_table_change": {
			Name:              "fk_ref_table_change",
			Columns:           []string{"x_id"},
			ReferencedTable:   "table_x",
			ReferencedColumns: []string{"id"},
		},
		"fk_ref_cols_change": {
			Name:              "fk_ref_cols_change",
			Columns:           []string{"y_id"},
			ReferencedTable:   "y",
			ReferencedColumns: []string{"id"},
		},
		"fk_on_delete_change": {
			Name:              "fk_on_delete_change",
			Columns:           []string{"z_id"},
			ReferencedTable:   "z",
			ReferencedColumns: []string{"id"},
			OnDelete:          "CASCADE",
		},
		"fk_on_update_change": {
			Name:              "fk_on_update_change",
			Columns:           []string{"w_id"},
			ReferencedTable:   "w",
			ReferencedColumns: []string{"id"},
			OnUpdate:          "SET NULL",
		},
	}
	devFKs := map[string]schema.ForeignKey{
		"fk_dev_only": {
			Name:              "fk_dev_only",
			Columns:           []string{"product_id"},
			ReferencedTable:   "products",
			ReferencedColumns: []string{"id"},
		},
		"fk_columns_change": {
			Name:              "fk_columns_change",
			Columns:           []string{"b_id"}, // columns changed
			ReferencedTable:   "a",
			ReferencedColumns: []string{"id"},
		},
		"fk_ref_table_change": {
			Name:              "fk_ref_table_change",
			Columns:           []string{"x_id"},
			ReferencedTable:   "table_y", // ref table changed
			ReferencedColumns: []string{"id"},
		},
		"fk_ref_cols_change": {
			Name:              "fk_ref_cols_change",
			Columns:           []string{"y_id"},
			ReferencedTable:   "y",
			ReferencedColumns: []string{"uuid"}, // ref columns changed
		},
		"fk_on_delete_change": {
			Name:              "fk_on_delete_change",
			Columns:           []string{"z_id"},
			ReferencedTable:   "z",
			ReferencedColumns: []string{"id"},
			OnDelete:          "RESTRICT", // on delete changed
		},
		"fk_on_update_change": {
			Name:              "fk_on_update_change",
			Columns:           []string{"w_id"},
			ReferencedTable:   "w",
			ReferencedColumns: []string{"id"},
			OnUpdate:          "CASCADE", // on update changed
		},
	}

	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:        "orders",
				Columns:     map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				ForeignKeys: prodFKs,
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:        "orders",
				Columns:     map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				ForeignKeys: devFKs,
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)

	if len(result.Tables) != 1 {
		t.Fatalf("expected 1 table diff, got %d", len(result.Tables))
	}
	td := result.Tables[0]
	if !td.HasDifferences {
		t.Error("expected differences")
	}

	// Added
	if len(td.AddedForeignKeys) != 1 || td.AddedForeignKeys[0].Name != "fk_dev_only" {
		t.Errorf("expected fk_dev_only to be added, got %v", td.AddedForeignKeys)
	}

	// Removed
	if len(td.RemovedForeignKeys) != 1 || td.RemovedForeignKeys[0].Name != "fk_prod_only" {
		t.Errorf("expected fk_prod_only to be removed, got %v", td.RemovedForeignKeys)
	}

	// Modified: 5 FKs should be modified (columns, ref table, ref cols, on delete, on update)
	if len(td.ModifiedForeignKeys) != 5 {
		t.Errorf("expected 5 modified FKs, got %d", len(td.ModifiedForeignKeys))
	}

	for _, fkDiff := range td.ModifiedForeignKeys {
		switch fkDiff.Name {
		case "fk_columns_change":
			if !fkDiff.ColumnsDiffer {
				t.Errorf("expected fk_columns_change columns to differ")
			}
		case "fk_ref_table_change":
			if !fkDiff.ReferencedTableDiffers {
				t.Errorf("expected fk_ref_table_change referenced table to differ")
			}
		case "fk_ref_cols_change":
			if !fkDiff.ReferencedColumnsDiffer {
				t.Errorf("expected fk_ref_cols_change referenced columns to differ")
			}
		case "fk_on_delete_change":
			if !fkDiff.OnDeleteDiffers {
				t.Errorf("expected fk_on_delete_change ON DELETE to differ")
			}
		case "fk_on_update_change":
			if !fkDiff.OnUpdateDiffers {
				t.Errorf("expected fk_on_update_change ON UPDATE to differ")
			}
		}
	}
}

// TestDiffSchemas_FKNormalizeAction verifies that "" and "NO ACTION" compare as equal.
func TestDiffSchemas_FKNormalizeAction_NoActionEquivalence(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:    "orders",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_test": {
						Name:              "fk_test",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "",         // empty
						OnUpdate:          "NO ACTION", // equivalent to empty
					},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:    "orders",
				Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_test": {
						Name:              "fk_test",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "NO ACTION", // equivalent to empty
						OnUpdate:          "",          // empty
					},
				},
			},
		},
	}

	result := schema.DiffSchemas(prod, dev)
	if len(result.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(result.Tables))
	}

	td := result.Tables[0]
	// "" vs "NO ACTION" should NOT be a difference
	if td.HasDifferences {
		t.Error("expected NO ACTION and empty string to be treated as equivalent")
	}
	if len(td.ModifiedForeignKeys) != 0 {
		t.Errorf("expected no modified FKs, got %d", len(td.ModifiedForeignKeys))
	}
}

// ============================================================================
// defaultsDiffer and normalizeDefault — exercised via DiffSchemas column diffs
// ============================================================================

func TestDiffSchemas_DefaultValueDiffs(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name        string
		prodDefault *string
		devDefault  *string
		wantDiff    bool
	}{
		{
			name:        "both nil — no diff",
			prodDefault: nil,
			devDefault:  nil,
			wantDiff:    false,
		},
		{
			name:        "prod nil dev has value — diff",
			prodDefault: nil,
			devDefault:  strPtr("0"),
			wantDiff:    true,
		},
		{
			name:        "dev nil prod has value — diff",
			prodDefault: strPtr("0"),
			devDefault:  nil,
			wantDiff:    true,
		},
		{
			name:        "same value — no diff",
			prodDefault: strPtr("hello"),
			devDefault:  strPtr("hello"),
			wantDiff:    false,
		},
		{
			name:        "single-quoted vs unquoted — same after normalization",
			prodDefault: strPtr("'hello'"),
			devDefault:  strPtr("hello"),
			wantDiff:    false,
		},
		{
			name:        "double-quoted vs unquoted — same after normalization",
			prodDefault: strPtr(`"world"`),
			devDefault:  strPtr("world"),
			wantDiff:    false,
		},
		{
			name:        "different values — diff",
			prodDefault: strPtr("0"),
			devDefault:  strPtr("1"),
			wantDiff:    true,
		},
		{
			name:        "whitespace trimmed — no diff",
			prodDefault: strPtr("  0  "),
			devDefault:  strPtr("0"),
			wantDiff:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prod := &schema.Schema{
				Tables: map[string]schema.Table{
					"t": {
						Name: "t",
						Columns: map[string]schema.Column{
							"col": {Name: "col", DataType: "varchar", DefaultValue: tt.prodDefault},
						},
					},
				},
			}
			dev := &schema.Schema{
				Tables: map[string]schema.Table{
					"t": {
						Name: "t",
						Columns: map[string]schema.Column{
							"col": {Name: "col", DataType: "varchar", DefaultValue: tt.devDefault},
						},
					},
				},
			}

			result := schema.DiffSchemas(prod, dev)
			td := result.Tables[0]

			hasDiff := td.HasDifferences
			if hasDiff != tt.wantDiff {
				t.Errorf("wantDiff=%v but HasDifferences=%v", tt.wantDiff, hasDiff)
			}
			if tt.wantDiff {
				found := false
				for _, cd := range td.ColumnDiffs {
					if cd.DefaultMismatch {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected a DefaultMismatch column diff")
				}
			}
		})
	}
}

// ============================================================================
// diffPrimaryKey — exercised via DiffSchemas
// ============================================================================

func TestDiffSchemas_PrimaryKeyDiffs(t *testing.T) {
	t.Run("no pk diff when same", func(t *testing.T) {
		prod := &schema.Schema{
			Tables: map[string]schema.Table{
				"t": {Name: "t", Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}}, PrimaryKey: []string{"id"}},
			},
		}
		dev := &schema.Schema{
			Tables: map[string]schema.Table{
				"t": {Name: "t", Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}}, PrimaryKey: []string{"id"}},
			},
		}
		result := schema.DiffSchemas(prod, dev)
		if result.Tables[0].PrimaryKeyDiff != nil {
			t.Error("expected no PrimaryKeyDiff when PKs are identical")
		}
	})

	t.Run("pk diff detected", func(t *testing.T) {
		prod := &schema.Schema{
			Tables: map[string]schema.Table{
				"t": {Name: "t", Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}}, PrimaryKey: []string{"id"}},
			},
		}
		dev := &schema.Schema{
			Tables: map[string]schema.Table{
				"t": {Name: "t", Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}}, PrimaryKey: []string{"id", "name"}},
			},
		}
		result := schema.DiffSchemas(prod, dev)
		td := result.Tables[0]
		if td.PrimaryKeyDiff == nil {
			t.Fatal("expected PrimaryKeyDiff when PKs differ")
		}
		if len(td.PrimaryKeyDiff.ProdColumns) != 1 || len(td.PrimaryKeyDiff.DevColumns) != 2 {
			t.Errorf("unexpected PrimaryKeyDiff: prod=%v dev=%v", td.PrimaryKeyDiff.ProdColumns, td.PrimaryKeyDiff.DevColumns)
		}
		if !td.HasDifferences {
			t.Error("expected HasDifferences when PK differs")
		}
	})

	t.Run("nil pk in both — no diff", func(t *testing.T) {
		prod := &schema.Schema{
			Tables: map[string]schema.Table{
				"t": {Name: "t", Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}}, PrimaryKey: nil},
			},
		}
		dev := &schema.Schema{
			Tables: map[string]schema.Table{
				"t": {Name: "t", Columns: map[string]schema.Column{"id": {Name: "id", DataType: "int"}}, PrimaryKey: nil},
			},
		}
		result := schema.DiffSchemas(prod, dev)
		if result.Tables[0].PrimaryKeyDiff != nil {
			t.Error("expected nil PrimaryKeyDiff when both PKs are nil")
		}
	})
}

// ============================================================================
// DiffSchemas with nil inputs
// ============================================================================

func TestDiffSchemas_NilInputs(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		result := schema.DiffSchemas(nil, nil)
		if len(result.Tables) != 0 {
			t.Errorf("expected 0 tables, got %d", len(result.Tables))
		}
	})

	t.Run("nil prod", func(t *testing.T) {
		dev := &schema.Schema{
			Tables: map[string]schema.Table{
				"users": {Name: "users"},
			},
		}
		result := schema.DiffSchemas(nil, dev)
		if len(result.Tables) != 1 {
			t.Fatalf("expected 1 table, got %d", len(result.Tables))
		}
		if !result.Tables[0].OnlyInDev {
			t.Error("expected table to be OnlyInDev when prod is nil")
		}
	})

	t.Run("nil dev", func(t *testing.T) {
		prod := &schema.Schema{
			Tables: map[string]schema.Table{
				"users": {Name: "users"},
			},
		}
		result := schema.DiffSchemas(prod, nil)
		if len(result.Tables) != 1 {
			t.Fatalf("expected 1 table, got %d", len(result.Tables))
		}
		if !result.Tables[0].OnlyInProd {
			t.Error("expected table to be OnlyInProd when dev is nil")
		}
	})
}

// ============================================================================
// HasDrift with empty tables list
// ============================================================================

func TestHasDrift_EmptyTables(t *testing.T) {
	result := schema.DiffResult{}
	if result.HasDrift() {
		t.Error("empty DiffResult should not indicate drift")
	}
}
