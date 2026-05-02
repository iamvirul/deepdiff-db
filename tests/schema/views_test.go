package schema_test

// Tests for Phase 2: Views — diffViews logic, migration generation, and SQLite introspection.

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ── diffViews unit tests ──────────────────────────────────────────────────────

func TestDiffViews_EmptyBothSides(t *testing.T) {
	result := schema.DiffSchemas(
		&schema.Schema{Tables: map[string]schema.Table{}, Views: map[string]schema.View{}},
		&schema.Schema{Tables: map[string]schema.Table{}, Views: map[string]schema.View{}},
	)
	if len(result.AddedViews) != 0 || len(result.RemovedViews) != 0 || len(result.ModifiedViews) != 0 {
		t.Errorf("expected no view diffs, got added=%d removed=%d modified=%d",
			len(result.AddedViews), len(result.RemovedViews), len(result.ModifiedViews))
	}
}

func TestDiffViews_AddedView(t *testing.T) {
	prod := &schema.Schema{Tables: map[string]schema.Table{}, Views: map[string]schema.View{}}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views: map[string]schema.View{
			"v_active": {Name: "v_active", Definition: "SELECT id FROM users WHERE active = 1"},
		},
	}
	result := schema.DiffSchemas(prod, dev)

	if len(result.AddedViews) != 1 {
		t.Fatalf("expected 1 AddedView, got %d", len(result.AddedViews))
	}
	if result.AddedViews[0].Name != "v_active" {
		t.Errorf("AddedViews[0].Name = %q, want %q", result.AddedViews[0].Name, "v_active")
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when views added")
	}
}

func TestDiffViews_RemovedView(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views: map[string]schema.View{
			"v_legacy": {Name: "v_legacy", Definition: "SELECT * FROM old_table"},
		},
	}
	dev := &schema.Schema{Tables: map[string]schema.Table{}, Views: map[string]schema.View{}}
	result := schema.DiffSchemas(prod, dev)

	if len(result.RemovedViews) != 1 || result.RemovedViews[0] != "v_legacy" {
		t.Errorf("expected RemovedViews=[v_legacy], got %v", result.RemovedViews)
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true when views removed")
	}
}

func TestDiffViews_IdenticalViews_NoDiff(t *testing.T) {
	def := "SELECT id, name FROM users WHERE active = 1"
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views:  map[string]schema.View{"v_active": {Name: "v_active", Definition: def}},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views:  map[string]schema.View{"v_active": {Name: "v_active", Definition: def}},
	}
	result := schema.DiffSchemas(prod, dev)

	if len(result.ModifiedViews) != 0 {
		t.Errorf("expected no modified views for identical definitions, got %v", result.ModifiedViews)
	}
}

func TestDiffViews_NormalizationIgnoresWhitespace(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views: map[string]schema.View{
			"v_users": {Name: "v_users", Definition: "SELECT  id  FROM   users"},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views: map[string]schema.View{
			"v_users": {Name: "v_users", Definition: "select id from users"},
		},
	}
	result := schema.DiffSchemas(prod, dev)

	if len(result.ModifiedViews) != 0 {
		t.Errorf("expected no diff after normalization, got modified=%v", result.ModifiedViews)
	}
}

func TestDiffViews_DefinitionDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views: map[string]schema.View{
			"v_summary": {Name: "v_summary", Definition: "SELECT COUNT(*) FROM orders"},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views: map[string]schema.View{
			"v_summary": {Name: "v_summary", Definition: "SELECT COUNT(*), SUM(total) FROM orders"},
		},
	}
	result := schema.DiffSchemas(prod, dev)

	if len(result.ModifiedViews) != 1 {
		t.Fatalf("expected 1 ModifiedView, got %d", len(result.ModifiedViews))
	}
	vd := result.ModifiedViews[0]
	if !vd.DefinitionDiffers {
		t.Error("DefinitionDiffers should be true")
	}
	if vd.ProdDefinition != "SELECT COUNT(*) FROM orders" {
		t.Errorf("ProdDefinition: got %q", vd.ProdDefinition)
	}
	if vd.DevDefinition != "SELECT COUNT(*), SUM(total) FROM orders" {
		t.Errorf("DevDefinition: got %q", vd.DevDefinition)
	}
}

func TestDiffViews_MaterializedDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views: map[string]schema.View{
			"v_mat": {Name: "v_mat", Definition: "SELECT id FROM users", IsMaterialized: false},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views: map[string]schema.View{
			"v_mat": {Name: "v_mat", Definition: "SELECT id FROM users", IsMaterialized: true},
		},
	}
	result := schema.DiffSchemas(prod, dev)

	if len(result.ModifiedViews) != 1 {
		t.Fatalf("expected 1 ModifiedView, got %d", len(result.ModifiedViews))
	}
	vd := result.ModifiedViews[0]
	if !vd.IsMaterializedDiffers {
		t.Error("IsMaterializedDiffers should be true")
	}
	if vd.ProdIsMaterialized == nil || *vd.ProdIsMaterialized != false {
		t.Error("ProdIsMaterialized should be false")
	}
	if vd.DevIsMaterialized == nil || *vd.DevIsMaterialized != true {
		t.Error("DevIsMaterialized should be true")
	}
}

func TestDiffViews_DeterministicOrder(t *testing.T) {
	prod := &schema.Schema{Tables: map[string]schema.Table{}, Views: map[string]schema.View{}}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Views: map[string]schema.View{
			"v_z": {Name: "v_z", Definition: "SELECT 3"},
			"v_a": {Name: "v_a", Definition: "SELECT 1"},
			"v_m": {Name: "v_m", Definition: "SELECT 2"},
		},
	}
	result := schema.DiffSchemas(prod, dev)

	if len(result.AddedViews) != 3 {
		t.Fatalf("expected 3 AddedViews, got %d", len(result.AddedViews))
	}
	names := []string{result.AddedViews[0].Name, result.AddedViews[1].Name, result.AddedViews[2].Name}
	if names[0] != "v_a" || names[1] != "v_m" || names[2] != "v_z" {
		t.Errorf("AddedViews not sorted alphabetically: %v", names)
	}
}

// ── Migration generation for views ───────────────────────────────────────────

func TestGenerateMigration_AddView_SQLite(t *testing.T) {
	diff := schema.DiffResult{
		AddedViews: []schema.View{
			{Name: "v_active", Definition: "SELECT id FROM users WHERE active = 1"},
		},
	}
	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "CREATE VIEW IF NOT EXISTS") {
		t.Errorf("SQLite should use CREATE VIEW IF NOT EXISTS, got:\n%s", sql)
	}
	if !strings.Contains(sql, "v_active") {
		t.Errorf("expected view name in output")
	}
}

func TestGenerateMigration_AddView_MySQL(t *testing.T) {
	diff := schema.DiffResult{
		AddedViews: []schema.View{
			{Name: "v_active", Definition: "SELECT id FROM users WHERE active = 1"},
		},
	}
	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "CREATE VIEW") {
		t.Errorf("expected CREATE VIEW in MySQL output")
	}
	// MySQL must NOT use IF NOT EXISTS
	if strings.Contains(sql, "IF NOT EXISTS") {
		t.Errorf("MySQL should not use IF NOT EXISTS for views")
	}
}

func TestGenerateMigration_AddView_PostgreSQL_Materialized(t *testing.T) {
	diff := schema.DiffResult{
		AddedViews: []schema.View{
			{Name: "v_mat", Definition: "SELECT id FROM users", IsMaterialized: true},
		},
	}
	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "CREATE MATERIALIZED VIEW") {
		t.Errorf("PostgreSQL should emit CREATE MATERIALIZED VIEW, got:\n%s", sql)
	}
}

func TestGenerateMigration_DropView_CommentedByDefault(t *testing.T) {
	diff := schema.DiffResult{RemovedViews: []string{"v_old"}}
	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if containsUncommented(sql, "DROP VIEW") {
		t.Error("DROP VIEW should be commented out by default")
	}
	if !strings.Contains(sql, "DROP VIEW") {
		t.Error("DROP VIEW should appear (commented) in output")
	}
}

func TestGenerateMigration_DropView_UncommentedWhenAllowed(t *testing.T) {
	diff := schema.DiffResult{RemovedViews: []string{"v_old"}}
	opts := &schema.MigrationOptions{AllowDropView: true}
	sql, err := schema.GenerateMigration(diff, "sqlite", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsUncommented(sql, "DROP VIEW") {
		t.Error("DROP VIEW should be uncommented when AllowDropView=true")
	}
}

func TestGenerateMigration_ModifiedView_PostgreSQL_CreateOrReplace(t *testing.T) {
	diff := schema.DiffResult{
		ModifiedViews: []schema.ViewDiff{
			{
				Name:              "v_summary",
				DefinitionDiffers: true,
				ProdDefinition:    "SELECT COUNT(*) FROM orders",
				DevDefinition:     "SELECT COUNT(*), SUM(total) FROM orders",
			},
		},
	}
	sql, err := schema.GenerateMigration(diff, "postgres", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "CREATE OR REPLACE VIEW") {
		t.Errorf("PostgreSQL modified view should use CREATE OR REPLACE VIEW, got:\n%s", sql)
	}
}

func TestGenerateMigration_ModifiedView_MySQL_DropCreate(t *testing.T) {
	diff := schema.DiffResult{
		ModifiedViews: []schema.ViewDiff{
			{
				Name:              "v_summary",
				DefinitionDiffers: true,
				ProdDefinition:    "SELECT COUNT(*) FROM orders",
				DevDefinition:     "SELECT COUNT(*), SUM(total) FROM orders",
			},
		},
	}
	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "DROP VIEW") {
		t.Errorf("MySQL modified view should include DROP VIEW")
	}
	if !strings.Contains(sql, "CREATE VIEW") {
		t.Errorf("MySQL modified view should include CREATE VIEW")
	}
}

// ── SQLite introspection (real DB) ────────────────────────────────────────────

func openSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite3: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestLoadSchema_SQLite_LoadsViews(t *testing.T) {
	db := openSQLiteDB(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, active INTEGER)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = db.ExecContext(ctx, `CREATE VIEW v_active AS SELECT id, name FROM users WHERE active = 1`)
	if err != nil {
		t.Fatalf("create view: %v", err)
	}
	_, err = db.ExecContext(ctx, `CREATE VIEW v_all AS SELECT id, name FROM users`)
	if err != nil {
		t.Fatalf("create view v_all: %v", err)
	}

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	if len(s.Views) != 2 {
		t.Errorf("expected 2 views, got %d: %v", len(s.Views), s.Views)
	}
	if _, ok := s.Views["v_active"]; !ok {
		t.Error("v_active not found in Views")
	}
	if _, ok := s.Views["v_all"]; !ok {
		t.Error("v_all not found in Views")
	}
	if s.Views["v_active"].IsMaterialized {
		t.Error("SQLite views should never be materialized")
	}
}

func TestLoadSchema_SQLite_IgnoreViews(t *testing.T) {
	db := openSQLiteDB(t)
	ctx := context.Background()

	_, _ = db.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`)
	_, _ = db.ExecContext(ctx, `CREATE VIEW v_keep AS SELECT id FROM users`)
	_, _ = db.ExecContext(ctx, `CREATE VIEW v_ignore AS SELECT name FROM users`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil,
		schema.LoadSchemaOptions{IgnoreViews: []string{"v_ignore"}})
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	if _, ok := s.Views["v_ignore"]; ok {
		t.Error("v_ignore should have been excluded by IgnoreViews")
	}
	if _, ok := s.Views["v_keep"]; !ok {
		t.Error("v_keep should still be present")
	}
}

func TestLoadSchema_SQLite_IgnoreViews_CaseInsensitive(t *testing.T) {
	db := openSQLiteDB(t)
	ctx := context.Background()

	_, _ = db.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY)`)
	_, _ = db.ExecContext(ctx, `CREATE VIEW V_Secret AS SELECT id FROM users`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil,
		schema.LoadSchemaOptions{IgnoreViews: []string{"v_secret"}}) // lowercase
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	if _, ok := s.Views["V_Secret"]; ok {
		t.Error("V_Secret should be excluded by case-insensitive ignore")
	}
}

func TestLoadSchema_SQLite_NoViews(t *testing.T) {
	db := openSQLiteDB(t)
	ctx := context.Background()

	_, _ = db.ExecContext(ctx, `CREATE TABLE items (id INTEGER PRIMARY KEY)`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	if len(s.Views) != 0 {
		t.Errorf("expected 0 views when none created, got %d", len(s.Views))
	}
}

func TestLoadSchema_SQLite_ViewDefinitionStored(t *testing.T) {
	db := openSQLiteDB(t)
	ctx := context.Background()

	_, _ = db.ExecContext(ctx, `CREATE TABLE orders (id INTEGER PRIMARY KEY, total REAL)`)
	_, _ = db.ExecContext(ctx, `CREATE VIEW v_orders AS SELECT id, total FROM orders WHERE total > 0`)

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	v, ok := s.Views["v_orders"]
	if !ok {
		t.Fatal("v_orders not found")
	}
	if !strings.Contains(strings.ToLower(v.Definition), "select") {
		t.Errorf("view definition should contain SELECT, got: %q", v.Definition)
	}
}

// ── Full diff pipeline: introspect → diff → migrate ─────────────────��────────

func TestViewPipeline_SQLite_EndToEnd(t *testing.T) {
	ctx := context.Background()

	// Prod DB: has v_active
	prodDB := openSQLiteDB(t)
	_, _ = prodDB.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, active INTEGER)`)
	_, _ = prodDB.ExecContext(ctx, `CREATE VIEW v_active AS SELECT id, name FROM users WHERE active = 1`)

	// Dev DB: v_active changed definition, plus new v_all
	devDB := openSQLiteDB(t)
	_, _ = devDB.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, active INTEGER)`)
	_, _ = devDB.ExecContext(ctx, `CREATE VIEW v_active AS SELECT id, name, active FROM users WHERE active = 1`)
	_, _ = devDB.ExecContext(ctx, `CREATE VIEW v_all AS SELECT id, name FROM users`)

	prodSchema, err := schema.LoadSchema(ctx, prodDB, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("prod LoadSchema: %v", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("dev LoadSchema: %v", err)
	}

	result := schema.DiffSchemas(prodSchema, devSchema)

	if len(result.AddedViews) != 1 || result.AddedViews[0].Name != "v_all" {
		t.Errorf("expected AddedViews=[v_all], got %v", result.AddedViews)
	}
	if len(result.ModifiedViews) != 1 || result.ModifiedViews[0].Name != "v_active" {
		t.Errorf("expected ModifiedViews=[v_active], got %v", result.ModifiedViews)
	}
	if !result.HasDrift() {
		t.Error("HasDrift() should be true")
	}

	// Migration should include both a CREATE VIEW IF NOT EXISTS and a modified view block
	migration, err := schema.GenerateMigration(result, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	if !strings.Contains(migration, "v_all") {
		t.Error("migration should reference added view v_all")
	}
	if !strings.Contains(migration, "v_active") {
		t.Error("migration should reference modified view v_active")
	}
}
