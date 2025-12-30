package schema_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

func TestWriteReports(t *testing.T) {
	tmpDir := t.TempDir()

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				Table:          "users",
				HasDifferences: true,
				OnlyInProd:     true,
				MissingInDev:   true,
			},
			{
				Name:           "posts",
				Table:          "posts",
				HasDifferences: true,
				ColumnDiffs: []schema.ColumnDiff{
					{
						Column:        "title",
						TypeMismatch:   true,
						ProdType:       "varchar",
						DevType:        "text",
					},
				},
			},
			{
				Name:           "comments",
				Table:          "comments",
				HasDifferences: false,
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("schema.schema.WriteReports failed: %v", err)
	}

	// Check JSON file
	jsonPath := filepath.Join(tmpDir, "schema_diff.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("JSON file was not created")
	}

	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	if !strings.Contains(string(jsonContent), "users") {
		t.Error("JSON should contain 'users'")
	}
	if !strings.Contains(string(jsonContent), "posts") {
		t.Error("JSON should contain 'posts'")
	}

	// Check text file
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	if _, err := os.Stat(textPath); os.IsNotExist(err) {
		t.Fatal("Text file was not created")
	}

	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "users") {
		t.Error("Text should contain 'users'")
	}
	if !strings.Contains(textStr, "posts") {
		t.Error("Text should contain 'posts'")
	}
	// Comments table should not appear (no differences)
	if strings.Contains(textStr, "comments") {
		t.Error("Text should not contain 'comments' (no differences)")
	}
}

func TestWriteReports_NoDifferences(t *testing.T) {
	tmpDir := t.TempDir()

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				Table:          "users",
				HasDifferences: false,
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("schema.schema.WriteReports failed: %v", err)
	}

	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "Schema: OK") {
		t.Error("Text should indicate no differences")
	}
}

func TestWriteReports_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "nested", "output", "dir")

	result := schema.DiffResult{
		Tables: []schema.TableDiff{},
	}

	if err := schema.WriteReports(result, outDir); err != nil {
		t.Fatalf("schema.schema.WriteReports failed: %v", err)
	}

	// Directory should be created
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		t.Fatal("Output directory was not created")
	}

	// Files should exist
	jsonPath := filepath.Join(outDir, "schema_diff.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("JSON file was not created")
	}
}

// ============================================================================
// Index Report Tests
// ============================================================================

func TestWriteReports_AddedIndexes(t *testing.T) {
	tmpDir := t.TempDir()

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				Table:          "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: false},
					{Name: "idx_users_name", Columns: []string{"first_name", "last_name"}, IsUnique: true},
				},
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports failed: %v", err)
	}

	// Check text file
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "idx_users_email") {
		t.Error("Text should contain 'idx_users_email'")
	}
	if !strings.Contains(textStr, "idx_users_name") {
		t.Error("Text should contain 'idx_users_name'")
	}
	if !strings.Contains(textStr, "missing in prod") {
		t.Error("Text should indicate indexes missing in prod")
	}
	if !strings.Contains(textStr, "[email]") {
		t.Error("Text should contain email column")
	}
	if !strings.Contains(textStr, "[first_name last_name]") {
		t.Error("Text should contain composite column names")
	}
	if !strings.Contains(textStr, "unique=true") {
		t.Error("Text should indicate unique index")
	}

	// Check JSON file
	jsonPath := filepath.Join(tmpDir, "schema_diff.json")
	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	jsonStr := string(jsonContent)
	if !strings.Contains(jsonStr, `"added_indexes"`) {
		t.Error("JSON should contain 'added_indexes'")
	}
	if !strings.Contains(jsonStr, `"idx_users_email"`) {
		t.Error("JSON should contain 'idx_users_email'")
	}
}

func TestWriteReports_RemovedIndexes(t *testing.T) {
	tmpDir := t.TempDir()

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "products",
				Table:          "products",
				HasDifferences: true,
				RemovedIndexes: []schema.Index{
					{Name: "idx_products_old", Columns: []string{"old_field"}, IsUnique: false},
					{Name: "idx_products_legacy", Columns: []string{"legacy_id"}, IsUnique: true},
				},
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports failed: %v", err)
	}

	// Check text file
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "idx_products_old") {
		t.Error("Text should contain 'idx_products_old'")
	}
	if !strings.Contains(textStr, "idx_products_legacy") {
		t.Error("Text should contain 'idx_products_legacy'")
	}
	if !strings.Contains(textStr, "missing in dev") {
		t.Error("Text should indicate indexes missing in dev")
	}

	// Check JSON file
	jsonPath := filepath.Join(tmpDir, "schema_diff.json")
	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	jsonStr := string(jsonContent)
	if !strings.Contains(jsonStr, `"removed_indexes"`) {
		t.Error("JSON should contain 'removed_indexes'")
	}
}

func TestWriteReports_ModifiedIndexes_ColumnsDiffer(t *testing.T) {
	tmpDir := t.TempDir()

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				Table:          "orders",
				HasDifferences: true,
				ModifiedIndexes: []schema.IndexDiff{
					{
						Name:          "idx_orders_composite",
						ColumnsDiffer: true,
						ProdColumns:   []string{"col_a", "col_b"},
						DevColumns:    []string{"col_b", "col_a"},
					},
				},
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports failed: %v", err)
	}

	// Check text file
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "idx_orders_composite") {
		t.Error("Text should contain 'idx_orders_composite'")
	}
	if !strings.Contains(textStr, "columns differ") {
		t.Error("Text should indicate columns differ")
	}
	if !strings.Contains(textStr, "prod=[col_a col_b]") {
		t.Error("Text should contain prod columns")
	}
	if !strings.Contains(textStr, "dev=[col_b col_a]") {
		t.Error("Text should contain dev columns")
	}

	// Check JSON file
	jsonPath := filepath.Join(tmpDir, "schema_diff.json")
	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	jsonStr := string(jsonContent)
	if !strings.Contains(jsonStr, `"modified_indexes"`) {
		t.Error("JSON should contain 'modified_indexes'")
	}
	if !strings.Contains(jsonStr, `"columns_differ": true`) {
		t.Error("JSON should contain 'columns_differ: true'")
	}
}

func TestWriteReports_ModifiedIndexes_UniqueDiffers(t *testing.T) {
	tmpDir := t.TempDir()

	boolTrue := true
	boolFalse := false

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				Table:          "users",
				HasDifferences: true,
				ModifiedIndexes: []schema.IndexDiff{
					{
						Name:          "idx_users_email",
						UniqueDiffers: true,
						ProdUnique:    &boolFalse,
						DevUnique:     &boolTrue,
					},
				},
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports failed: %v", err)
	}

	// Check text file
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "idx_users_email") {
		t.Error("Text should contain 'idx_users_email'")
	}
	if !strings.Contains(textStr, "uniqueness differs") {
		t.Error("Text should indicate uniqueness differs")
	}
	if !strings.Contains(textStr, "prod=false") {
		t.Error("Text should contain prod=false")
	}
	if !strings.Contains(textStr, "dev=true") {
		t.Error("Text should contain dev=true")
	}
}

func TestWriteReports_MixedColumnAndIndexChanges(t *testing.T) {
	tmpDir := t.TempDir()

	boolTrue := true

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				Table:          "users",
				HasDifferences: true,
				ColumnDiffs: []schema.ColumnDiff{
					{
						Column:       "email",
						TypeMismatch: true,
						ProdType:     "varchar(100)",
						DevType:      "varchar(255)",
					},
				},
				AddedIndexes: []schema.Index{
					{Name: "idx_users_new", Columns: []string{"new_col"}, IsUnique: false},
				},
				RemovedIndexes: []schema.Index{
					{Name: "idx_users_old", Columns: []string{"old_col"}, IsUnique: false},
				},
				ModifiedIndexes: []schema.IndexDiff{
					{
						Name:          "idx_users_email",
						UniqueDiffers: true,
						ProdUnique:    &boolTrue,
						DevUnique:     &boolTrue,
					},
				},
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports failed: %v", err)
	}

	// Check text file contains all types of changes
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "type mismatch") {
		t.Error("Text should contain column type mismatch")
	}
	if !strings.Contains(textStr, "idx_users_new") {
		t.Error("Text should contain added index")
	}
	if !strings.Contains(textStr, "idx_users_old") {
		t.Error("Text should contain removed index")
	}
	if !strings.Contains(textStr, "idx_users_email") {
		t.Error("Text should contain modified index")
	}
}

func TestWriteReports_MultipleTablesWithIndexes(t *testing.T) {
	tmpDir := t.TempDir()

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				Table:          "users",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_users_email", Columns: []string{"email"}, IsUnique: true},
				},
			},
			{
				Name:           "products",
				Table:          "products",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_products_sku", Columns: []string{"sku"}, IsUnique: true},
				},
				RemovedIndexes: []schema.Index{
					{Name: "idx_products_old", Columns: []string{"old"}, IsUnique: false},
				},
			},
			{
				Name:           "orders",
				Table:          "orders",
				HasDifferences: false, // No changes
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports failed: %v", err)
	}

	// Check text file
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "Table: users") {
		t.Error("Text should contain users table")
	}
	if !strings.Contains(textStr, "Table: products") {
		t.Error("Text should contain products table")
	}
	if strings.Contains(textStr, "Table: orders") {
		t.Error("Text should not contain orders table (no changes)")
	}
	if !strings.Contains(textStr, "idx_users_email") {
		t.Error("Text should contain users index")
	}
	if !strings.Contains(textStr, "idx_products_sku") {
		t.Error("Text should contain products added index")
	}
	if !strings.Contains(textStr, "idx_products_old") {
		t.Error("Text should contain products removed index")
	}
}

func TestWriteReports_IndexOnly_NoDifferences(t *testing.T) {
	tmpDir := t.TempDir()

	// All tables with no index differences
	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "users",
				Table:          "users",
				HasDifferences: false,
				AddedIndexes:   []schema.Index{},
				RemovedIndexes: []schema.Index{},
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports failed: %v", err)
	}

	// Check text file shows no differences
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "Schema: OK") {
		t.Error("Text should indicate no differences")
	}
}

func TestWriteReports_CompositeIndexes(t *testing.T) {
	tmpDir := t.TempDir()

	result := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				Table:          "orders",
				HasDifferences: true,
				AddedIndexes: []schema.Index{
					{Name: "idx_orders_composite", Columns: []string{"user_id", "product_id", "order_date"}, IsUnique: false},
				},
			},
		},
	}

	if err := schema.WriteReports(result, tmpDir); err != nil {
		t.Fatalf("WriteReports failed: %v", err)
	}

	// Check text file
	textPath := filepath.Join(tmpDir, "schema_diff.txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}

	textStr := string(textContent)
	if !strings.Contains(textStr, "[user_id product_id order_date]") {
		t.Error("Text should contain all composite index columns")
	}
}

