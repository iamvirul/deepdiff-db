package schema

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ============================================================================
// Foreign Key Diff Tests
// ============================================================================

func TestDiffForeignKeys_NoForeignKeys(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {Name: "users", Columns: map[string]schema.Column{}},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {Name: "users", Columns: map[string]schema.Column{}},
		},
	}

	diff := schema.DiffSchemas(prod, dev)

	for _, td := range diff.Tables {
		if td.Name == "users" {
			if len(td.AddedForeignKeys) != 0 {
				t.Errorf("expected no added foreign keys, got %d", len(td.AddedForeignKeys))
			}
			if len(td.RemovedForeignKeys) != 0 {
				t.Errorf("expected no removed foreign keys, got %d", len(td.RemovedForeignKeys))
			}
		}
	}
}

func TestDiffForeignKeys_AddedForeignKey(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:        "orders",
				Columns:     map[string]schema.Column{"id": {Name: "id"}, "user_id": {Name: "user_id"}},
				ForeignKeys: map[string]schema.ForeignKey{},
			},
			"users": {Name: "users", Columns: map[string]schema.Column{"id": {Name: "id"}}},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:    "orders",
				Columns: map[string]schema.Column{"id": {Name: "id"}, "user_id": {Name: "user_id"}},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:              "fk_orders_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
						OnUpdate:          "NO ACTION",
					},
				},
			},
			"users": {Name: "users", Columns: map[string]schema.Column{"id": {Name: "id"}}},
		},
	}

	diff := schema.DiffSchemas(prod, dev)

	for _, td := range diff.Tables {
		if td.Name == "orders" {
			if len(td.AddedForeignKeys) != 1 {
				t.Fatalf("expected 1 added foreign key, got %d", len(td.AddedForeignKeys))
			}
			if td.AddedForeignKeys[0].Name != "fk_orders_user" {
				t.Errorf("expected fk_orders_user, got %s", td.AddedForeignKeys[0].Name)
			}
			if !td.HasDifferences {
				t.Error("expected HasDifferences to be true")
			}
		}
	}
}

func TestDiffForeignKeys_RemovedForeignKey(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:    "orders",
				Columns: map[string]schema.Column{"id": {Name: "id"}, "user_id": {Name: "user_id"}},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:              "fk_orders_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
					},
				},
			},
			"users": {Name: "users", Columns: map[string]schema.Column{"id": {Name: "id"}}},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:        "orders",
				Columns:     map[string]schema.Column{"id": {Name: "id"}, "user_id": {Name: "user_id"}},
				ForeignKeys: map[string]schema.ForeignKey{},
			},
			"users": {Name: "users", Columns: map[string]schema.Column{"id": {Name: "id"}}},
		},
	}

	diff := schema.DiffSchemas(prod, dev)

	for _, td := range diff.Tables {
		if td.Name == "orders" {
			if len(td.RemovedForeignKeys) != 1 {
				t.Fatalf("expected 1 removed foreign key, got %d", len(td.RemovedForeignKeys))
			}
			if td.RemovedForeignKeys[0].Name != "fk_orders_user" {
				t.Errorf("expected fk_orders_user, got %s", td.RemovedForeignKeys[0].Name)
			}
		}
	}
}

func TestDiffForeignKeys_ModifiedForeignKey_OnDelete(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:    "orders",
				Columns: map[string]schema.Column{"id": {Name: "id"}, "user_id": {Name: "user_id"}},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:              "fk_orders_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "NO ACTION",
					},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name:    "orders",
				Columns: map[string]schema.Column{"id": {Name: "id"}, "user_id": {Name: "user_id"}},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_orders_user": {
						Name:              "fk_orders_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
					},
				},
			},
		},
	}

	diff := schema.DiffSchemas(prod, dev)

	for _, td := range diff.Tables {
		if td.Name == "orders" {
			if len(td.ModifiedForeignKeys) != 1 {
				t.Fatalf("expected 1 modified foreign key, got %d", len(td.ModifiedForeignKeys))
			}
			if !td.ModifiedForeignKeys[0].OnDeleteDiffers {
				t.Error("expected OnDeleteDiffers to be true")
			}
		}
	}
}

// ============================================================================
// Foreign Key Migration Generation Tests
// ============================================================================

func TestGenerateAddForeignKey_MySQL(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				AddedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_orders_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
						OnUpdate:          "NO ACTION",
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "ADD CONSTRAINT `fk_orders_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)") {
		t.Error("Expected ADD FOREIGN KEY statement")
	}
	if !strings.Contains(sql, "ON DELETE CASCADE") {
		t.Error("Expected ON DELETE CASCADE")
	}
}

func TestGenerateAddForeignKey_PostgreSQL(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				AddedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_orders_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "SET NULL",
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, `ADD CONSTRAINT "fk_orders_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id")`) {
		t.Error("Expected ADD FOREIGN KEY statement with double quotes")
	}
	if !strings.Contains(sql, "ON DELETE SET NULL") {
		t.Error("Expected ON DELETE SET NULL")
	}
}

func TestGenerateDropForeignKey_MySQL_Commented(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				RemovedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_orders_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "-- ALTER TABLE `orders` DROP FOREIGN KEY `fk_orders_user`;") {
		t.Error("Expected commented DROP FOREIGN KEY statement")
	}
}

func TestGenerateDropForeignKey_MySQL_Enabled(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				RemovedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_orders_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowDropForeignKey: true}
	sql, err := schema.GenerateMigration(diff, "mysql", opts)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if strings.Contains(sql, "-- ALTER TABLE `orders` DROP FOREIGN KEY") {
		t.Error("DROP FOREIGN KEY should not be commented when AllowDropForeignKey=true")
	}
	if !strings.Contains(sql, "ALTER TABLE `orders` DROP FOREIGN KEY `fk_orders_user`;") {
		t.Error("Expected uncommented DROP FOREIGN KEY statement")
	}
}

func TestGenerateDropForeignKey_PostgreSQL(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "orders",
				HasDifferences: true,
				RemovedForeignKeys: []schema.ForeignKey{
					{
						Name: "fk_orders_user",
					},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowDropForeignKey: true}
	sql, err := schema.GenerateMigration(diff, "postgres", opts)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, `ALTER TABLE "orders" DROP CONSTRAINT "fk_orders_user";`) {
		t.Error("Expected DROP CONSTRAINT statement for PostgreSQL")
	}
}

func TestGenerateAddForeignKey_CompositeForeignKey(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "order_items",
				HasDifferences: true,
				AddedForeignKeys: []schema.ForeignKey{
					{
						Name:              "fk_order_items_order",
						Columns:           []string{"order_id", "product_id"},
						ReferencedTable:   "order_products",
						ReferencedColumns: []string{"order_id", "product_id"},
						OnDelete:          "CASCADE",
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "FOREIGN KEY (`order_id`, `product_id`)") {
		t.Error("Expected composite foreign key columns")
	}
	if !strings.Contains(sql, "REFERENCES `order_products` (`order_id`, `product_id`)") {
		t.Error("Expected composite referenced columns")
	}
}

// ============================================================================
// Primary Key Diff Tests
// ============================================================================

func TestDiffPrimaryKey_NoDifference(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {Name: "users", PrimaryKey: []string{"id"}},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {Name: "users", PrimaryKey: []string{"id"}},
		},
	}

	diff := schema.DiffSchemas(prod, dev)

	for _, td := range diff.Tables {
		if td.Name == "users" {
			if td.PrimaryKeyDiff != nil {
				t.Error("expected no primary key diff")
			}
		}
	}
}

func TestDiffPrimaryKey_ColumnAdded(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"order_items": {Name: "order_items", PrimaryKey: []string{"id"}},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"order_items": {Name: "order_items", PrimaryKey: []string{"order_id", "product_id"}},
		},
	}

	diff := schema.DiffSchemas(prod, dev)

	for _, td := range diff.Tables {
		if td.Name == "order_items" {
			if td.PrimaryKeyDiff == nil {
				t.Fatal("expected primary key diff")
			}
			if len(td.PrimaryKeyDiff.ProdColumns) != 1 {
				t.Errorf("expected 1 prod PK column, got %d", len(td.PrimaryKeyDiff.ProdColumns))
			}
			if len(td.PrimaryKeyDiff.DevColumns) != 2 {
				t.Errorf("expected 2 dev PK columns, got %d", len(td.PrimaryKeyDiff.DevColumns))
			}
			if !td.HasDifferences {
				t.Error("expected HasDifferences to be true")
			}
		}
	}
}

func TestDiffPrimaryKey_ColumnOrderChanged(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{
			"items": {Name: "items", PrimaryKey: []string{"a", "b"}},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{
			"items": {Name: "items", PrimaryKey: []string{"b", "a"}},
		},
	}

	diff := schema.DiffSchemas(prod, dev)

	for _, td := range diff.Tables {
		if td.Name == "items" {
			if td.PrimaryKeyDiff == nil {
				t.Fatal("expected primary key diff due to order change")
			}
		}
	}
}

// ============================================================================
// Primary Key Migration Generation Tests
// ============================================================================

func TestGenerateModifyPrimaryKey_MySQL_Commented(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "order_items",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"order_id", "product_id"},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "-- PRIMARY KEY MODIFICATION") {
		t.Error("Expected PRIMARY KEY MODIFICATION section")
	}
	if !strings.Contains(sql, "-- ALTER TABLE `order_items` DROP PRIMARY KEY;") {
		t.Error("Expected commented DROP PRIMARY KEY")
	}
	if !strings.Contains(sql, "-- ALTER TABLE `order_items` ADD PRIMARY KEY (`order_id`, `product_id`);") {
		t.Error("Expected commented ADD PRIMARY KEY")
	}
}

func TestGenerateModifyPrimaryKey_MySQL_Enabled(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "order_items",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"order_id", "product_id"},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowModifyPrimaryKey: true}
	sql, err := schema.GenerateMigration(diff, "mysql", opts)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "ALTER TABLE `order_items` DROP PRIMARY KEY;") {
		t.Error("Expected uncommented DROP PRIMARY KEY")
	}
	if !strings.Contains(sql, "ALTER TABLE `order_items` ADD PRIMARY KEY (`order_id`, `product_id`);") {
		t.Error("Expected uncommented ADD PRIMARY KEY")
	}
}

func TestGenerateModifyPrimaryKey_PostgreSQL(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "items",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"item_id"},
				},
			},
		},
	}

	opts := &schema.MigrationOptions{AllowModifyPrimaryKey: true}
	sql, err := schema.GenerateMigration(diff, "postgres", opts)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, `ALTER TABLE "items" DROP CONSTRAINT items_pkey;`) {
		t.Error("Expected DROP CONSTRAINT for PostgreSQL")
	}
	if !strings.Contains(sql, `ALTER TABLE "items" ADD PRIMARY KEY ("item_id");`) {
		t.Error("Expected ADD PRIMARY KEY for PostgreSQL")
	}
}

func TestGenerateModifyPrimaryKey_SQLite(t *testing.T) {
	diff := schema.DiffResult{
		Tables: []schema.TableDiff{
			{
				Name:           "items",
				HasDifferences: true,
				PrimaryKeyDiff: &schema.PrimaryKeyDiff{
					ProdColumns: []string{"id"},
					DevColumns:  []string{"item_id"},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	if !strings.Contains(sql, "-- SQLite limitation: Cannot modify primary key") {
		t.Error("Expected SQLite limitation message")
	}
	if !strings.Contains(sql, "-- Manual table recreation required:") {
		t.Error("Expected manual migration instructions")
	}
}

// ============================================================================
// FK Generation for New Tables Tests
// ============================================================================

func TestGenerateMigration_NewTableWithForeignKeys(t *testing.T) {
	diff := schema.DiffResult{
		AddedTables: []schema.Table{
			{
				Name: "reviews",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "int", IsNullable: false},
					"user_id":    {Name: "user_id", DataType: "int", IsNullable: false},
					"product_id": {Name: "product_id", DataType: "int", IsNullable: false},
					"rating":     {Name: "rating", DataType: "int", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
				ForeignKeys: map[string]schema.ForeignKey{
					"fk_reviews_user": {
						Name:              "fk_reviews_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
					},
					"fk_reviews_product": {
						Name:              "fk_reviews_product",
						Columns:           []string{"product_id"},
						ReferencedTable:   "products",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
					},
				},
			},
		},
	}

	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Check that CREATE TABLE is generated
	if !strings.Contains(sql, "CREATE TABLE `reviews`") {
		t.Error("Expected CREATE TABLE statement")
	}

	// Check that FK statements are generated for the new table
	if !strings.Contains(sql, "ADD CONSTRAINT `fk_reviews_user`") {
		t.Errorf("Expected ADD CONSTRAINT for fk_reviews_user, got:\n%s", sql)
	}
	if !strings.Contains(sql, "FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)") {
		t.Error("Expected FOREIGN KEY clause for user_id")
	}
	if !strings.Contains(sql, "ADD CONSTRAINT `fk_reviews_product`") {
		t.Errorf("Expected ADD CONSTRAINT for fk_reviews_product, got:\n%s", sql)
	}
	if !strings.Contains(sql, "FOREIGN KEY (`product_id`) REFERENCES `products` (`id`)") {
		t.Error("Expected FOREIGN KEY clause for product_id")
	}
}

// ============================================================================
// Config Tests for New Options
// ============================================================================

func TestMigrationConfig_ForeignKeyOptions(t *testing.T) {
	opts := &schema.MigrationOptions{
		AllowDropForeignKey:   true,
		AllowModifyPrimaryKey: true,
	}

	if !opts.AllowDropForeignKey {
		t.Error("expected AllowDropForeignKey to be true")
	}
	if !opts.AllowModifyPrimaryKey {
		t.Error("expected AllowModifyPrimaryKey to be true")
	}
}
