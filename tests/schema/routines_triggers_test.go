package schema_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// ── diffRoutines ──────────────────────────────────────────────────────────────

func TestDiffRoutines_AddedRoutine(t *testing.T) {
	prod := &schema.Schema{Tables: map[string]schema.Table{}}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_new": {Name: "fn_new", Kind: "FUNCTION", Definition: "BEGIN RETURN 1; END"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.AddedRoutines) != 1 {
		t.Fatalf("expected 1 added routine, got %d", len(result.AddedRoutines))
	}
	if result.AddedRoutines[0].Name != "fn_new" {
		t.Errorf("expected fn_new, got %s", result.AddedRoutines[0].Name)
	}
	if len(result.RemovedRoutines) != 0 || len(result.ModifiedRoutines) != 0 {
		t.Error("expected no removed or modified routines")
	}
}

func TestDiffRoutines_RemovedRoutine(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_old": {Name: "fn_old", Kind: "PROCEDURE", Definition: "BEGIN END"},
		},
	}
	dev := &schema.Schema{Tables: map[string]schema.Table{}}
	result := schema.DiffSchemas(prod, dev)
	if len(result.RemovedRoutines) != 1 {
		t.Fatalf("expected 1 removed routine, got %d", len(result.RemovedRoutines))
	}
	if result.RemovedRoutines[0] != "fn_old" {
		t.Errorf("expected fn_old, got %s", result.RemovedRoutines[0])
	}
}

func TestDiffRoutines_IdenticalRoutine_NoDiff(t *testing.T) {
	r := schema.Routine{Name: "fn_same", Kind: "FUNCTION", Definition: "BEGIN RETURN 42; END", ReturnType: "INT", Language: "sql"}
	prod := &schema.Schema{Tables: map[string]schema.Table{}, Routines: map[string]schema.Routine{"fn_same": r}}
	dev := &schema.Schema{Tables: map[string]schema.Table{}, Routines: map[string]schema.Routine{"fn_same": r}}
	result := schema.DiffSchemas(prod, dev)
	if len(result.AddedRoutines) != 0 || len(result.RemovedRoutines) != 0 || len(result.ModifiedRoutines) != 0 {
		t.Error("expected no diff for identical routines")
	}
}

func TestDiffRoutines_DefinitionDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_calc": {Name: "fn_calc", Kind: "FUNCTION", Definition: "BEGIN RETURN 1; END"},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_calc": {Name: "fn_calc", Kind: "FUNCTION", Definition: "BEGIN RETURN 2; END"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedRoutines) != 1 {
		t.Fatalf("expected 1 modified routine, got %d", len(result.ModifiedRoutines))
	}
	rd := result.ModifiedRoutines[0]
	if !rd.DefinitionDiffers {
		t.Error("DefinitionDiffers should be true")
	}
	if rd.ProdDefinition != "BEGIN RETURN 1; END" || rd.DevDefinition != "BEGIN RETURN 2; END" {
		t.Errorf("unexpected definitions: prod=%q dev=%q", rd.ProdDefinition, rd.DevDefinition)
	}
}

func TestDiffRoutines_KindDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_x": {Name: "fn_x", Kind: "FUNCTION", Definition: "BEGIN END"},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_x": {Name: "fn_x", Kind: "PROCEDURE", Definition: "BEGIN END"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedRoutines) != 1 {
		t.Fatalf("expected 1 modified routine, got %d", len(result.ModifiedRoutines))
	}
	rd := result.ModifiedRoutines[0]
	if !rd.KindDiffers {
		t.Error("KindDiffers should be true")
	}
	if rd.ProdKind != "FUNCTION" || rd.DevKind != "PROCEDURE" {
		t.Errorf("unexpected kinds: prod=%q dev=%q", rd.ProdKind, rd.DevKind)
	}
}

func TestDiffRoutines_ReturnTypeDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_price": {Name: "fn_price", Kind: "FUNCTION", Definition: "BEGIN RETURN 0; END", ReturnType: "INT"},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_price": {Name: "fn_price", Kind: "FUNCTION", Definition: "BEGIN RETURN 0; END", ReturnType: "DECIMAL(10,2)"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedRoutines) != 1 {
		t.Fatalf("expected 1 modified routine, got %d", len(result.ModifiedRoutines))
	}
	rd := result.ModifiedRoutines[0]
	if !rd.ReturnTypeDiffers {
		t.Error("ReturnTypeDiffers should be true")
	}
}

func TestDiffRoutines_ParametersDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_add": {
				Name:       "fn_add",
				Kind:       "FUNCTION",
				Definition: "BEGIN RETURN a + b; END",
				Parameters: []schema.RoutineParameter{
					{Name: "a", DataType: "INT", Mode: "IN"},
					{Name: "b", DataType: "INT", Mode: "IN"},
				},
			},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_add": {
				Name:       "fn_add",
				Kind:       "FUNCTION",
				Definition: "BEGIN RETURN a + b; END",
				Parameters: []schema.RoutineParameter{
					{Name: "a", DataType: "INT", Mode: "IN"},
					{Name: "b", DataType: "BIGINT", Mode: "IN"},
				},
			},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedRoutines) != 1 {
		t.Fatalf("expected 1 modified routine, got %d", len(result.ModifiedRoutines))
	}
	if !result.ModifiedRoutines[0].ParametersDiffers {
		t.Error("ParametersDiffers should be true")
	}
}

func TestDiffRoutines_NormalizedDefinition_WhitespaceDifference(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_ws": {Name: "fn_ws", Kind: "FUNCTION", Definition: "BEGIN  RETURN  1;  END"},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_ws": {Name: "fn_ws", Kind: "FUNCTION", Definition: "BEGIN RETURN 1; END"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedRoutines) != 0 {
		t.Error("whitespace-only definition differences should not produce a diff")
	}
}

func TestDiffRoutines_SortOrder(t *testing.T) {
	prod := &schema.Schema{Tables: map[string]schema.Table{}}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_z": {Name: "fn_z", Kind: "FUNCTION", Definition: "BEGIN END"},
			"fn_a": {Name: "fn_a", Kind: "FUNCTION", Definition: "BEGIN END"},
			"fn_m": {Name: "fn_m", Kind: "FUNCTION", Definition: "BEGIN END"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.AddedRoutines) != 3 {
		t.Fatalf("expected 3 added routines, got %d", len(result.AddedRoutines))
	}
	if result.AddedRoutines[0].Name != "fn_a" || result.AddedRoutines[1].Name != "fn_m" || result.AddedRoutines[2].Name != "fn_z" {
		t.Errorf("added routines not sorted: got %s, %s, %s",
			result.AddedRoutines[0].Name, result.AddedRoutines[1].Name, result.AddedRoutines[2].Name)
	}
}

// ── diffTriggers ──────────────────────────────────────────────────────────────

func TestDiffTriggers_AddedTrigger(t *testing.T) {
	prod := &schema.Schema{Tables: map[string]schema.Table{}}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_audit": {Name: "trg_audit", Table: "orders", Timing: "AFTER", Event: "INSERT",
				Definition: "CREATE TRIGGER trg_audit AFTER INSERT ON orders FOR EACH ROW BEGIN END", ForEachRow: true},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.AddedTriggers) != 1 {
		t.Fatalf("expected 1 added trigger, got %d", len(result.AddedTriggers))
	}
	if result.AddedTriggers[0].Name != "trg_audit" {
		t.Errorf("expected trg_audit, got %s", result.AddedTriggers[0].Name)
	}
}

func TestDiffTriggers_RemovedTrigger(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_old": {Name: "trg_old", Table: "users", Timing: "BEFORE", Event: "DELETE",
				Definition: "CREATE TRIGGER trg_old BEFORE DELETE ON users BEGIN END"},
		},
	}
	dev := &schema.Schema{Tables: map[string]schema.Table{}}
	result := schema.DiffSchemas(prod, dev)
	if len(result.RemovedTriggers) != 1 {
		t.Fatalf("expected 1 removed trigger, got %d", len(result.RemovedTriggers))
	}
	if result.RemovedTriggers[0] != "trg_old" {
		t.Errorf("expected trg_old, got %s", result.RemovedTriggers[0])
	}
}

func TestDiffTriggers_IdenticalTrigger_NoDiff(t *testing.T) {
	trg := schema.Trigger{
		Name: "trg_same", Table: "orders", Timing: "AFTER", Event: "INSERT",
		Definition: "CREATE TRIGGER trg_same AFTER INSERT ON orders FOR EACH ROW BEGIN END",
		ForEachRow: true,
	}
	prod := &schema.Schema{Tables: map[string]schema.Table{}, Triggers: map[string]schema.Trigger{"trg_same": trg}}
	dev := &schema.Schema{Tables: map[string]schema.Table{}, Triggers: map[string]schema.Trigger{"trg_same": trg}}
	result := schema.DiffSchemas(prod, dev)
	if len(result.AddedTriggers) != 0 || len(result.RemovedTriggers) != 0 || len(result.ModifiedTriggers) != 0 {
		t.Error("expected no diff for identical triggers")
	}
}

func TestDiffTriggers_TimingDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_t": {Name: "trg_t", Table: "orders", Timing: "BEFORE", Event: "INSERT",
				Definition: "CREATE TRIGGER trg_t BEFORE INSERT ON orders BEGIN END"},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_t": {Name: "trg_t", Table: "orders", Timing: "AFTER", Event: "INSERT",
				Definition: "CREATE TRIGGER trg_t AFTER INSERT ON orders BEGIN END"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedTriggers) != 1 {
		t.Fatalf("expected 1 modified trigger, got %d", len(result.ModifiedTriggers))
	}
	td := result.ModifiedTriggers[0]
	if !td.TimingDiffers {
		t.Error("TimingDiffers should be true")
	}
	if td.ProdTiming != "BEFORE" || td.DevTiming != "AFTER" {
		t.Errorf("unexpected timing: prod=%q dev=%q", td.ProdTiming, td.DevTiming)
	}
}

func TestDiffTriggers_EventDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_e": {Name: "trg_e", Table: "orders", Timing: "AFTER", Event: "INSERT",
				Definition: "CREATE TRIGGER trg_e AFTER INSERT ON orders BEGIN END"},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_e": {Name: "trg_e", Table: "orders", Timing: "AFTER", Event: "UPDATE",
				Definition: "CREATE TRIGGER trg_e AFTER UPDATE ON orders BEGIN END"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedTriggers) != 1 {
		t.Fatalf("expected 1 modified trigger, got %d", len(result.ModifiedTriggers))
	}
	td := result.ModifiedTriggers[0]
	if !td.EventDiffers {
		t.Error("EventDiffers should be true")
	}
	if td.ProdEvent != "INSERT" || td.DevEvent != "UPDATE" {
		t.Errorf("unexpected events: prod=%q dev=%q", td.ProdEvent, td.DevEvent)
	}
}

func TestDiffTriggers_DefinitionDiffers(t *testing.T) {
	prod := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_d": {Name: "trg_d", Table: "orders", Timing: "AFTER", Event: "INSERT",
				Definition: "CREATE TRIGGER trg_d AFTER INSERT ON orders BEGIN SELECT 1; END"},
		},
	}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_d": {Name: "trg_d", Table: "orders", Timing: "AFTER", Event: "INSERT",
				Definition: "CREATE TRIGGER trg_d AFTER INSERT ON orders BEGIN SELECT 2; END"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.ModifiedTriggers) != 1 {
		t.Fatalf("expected 1 modified trigger, got %d", len(result.ModifiedTriggers))
	}
	if !result.ModifiedTriggers[0].DefinitionDiffers {
		t.Error("DefinitionDiffers should be true")
	}
}

func TestDiffTriggers_SortOrder(t *testing.T) {
	prod := &schema.Schema{Tables: map[string]schema.Table{}}
	dev := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_z": {Name: "trg_z", Table: "t", Timing: "AFTER", Event: "INSERT",
				Definition: "CREATE TRIGGER trg_z AFTER INSERT ON t BEGIN END"},
			"trg_a": {Name: "trg_a", Table: "t", Timing: "AFTER", Event: "INSERT",
				Definition: "CREATE TRIGGER trg_a AFTER INSERT ON t BEGIN END"},
		},
	}
	result := schema.DiffSchemas(prod, dev)
	if len(result.AddedTriggers) != 2 {
		t.Fatalf("expected 2 added triggers, got %d", len(result.AddedTriggers))
	}
	if result.AddedTriggers[0].Name != "trg_a" || result.AddedTriggers[1].Name != "trg_z" {
		t.Errorf("added triggers not sorted: got %s, %s", result.AddedTriggers[0].Name, result.AddedTriggers[1].Name)
	}
}

// ── SQLite trigger introspection ──────────────────────────────────────────────

func newSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteTriggers_LoadsTriggers(t *testing.T) {
	db := newSQLiteDB(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `CREATE TABLE orders (id INTEGER PRIMARY KEY, total REAL)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = db.ExecContext(ctx, `
		CREATE TRIGGER trg_audit AFTER INSERT ON orders
		FOR EACH ROW BEGIN SELECT 1; END
	`)
	if err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	trg, ok := s.Triggers["trg_audit"]
	if !ok {
		t.Fatal("trg_audit not found in schema")
	}
	if trg.Table != "orders" {
		t.Errorf("Table: want %q, got %q", "orders", trg.Table)
	}
	if trg.Timing != "AFTER" {
		t.Errorf("Timing: want AFTER, got %q", trg.Timing)
	}
	if trg.Event != "INSERT" {
		t.Errorf("Event: want INSERT, got %q", trg.Event)
	}
	if !trg.ForEachRow {
		t.Error("ForEachRow should be true")
	}
	if trg.Definition == "" {
		t.Error("Definition should not be empty")
	}
}

func TestSQLiteTriggers_BeforeDelete(t *testing.T) {
	db := newSQLiteDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER trg_before_del BEFORE DELETE ON users BEGIN SELECT 1; END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	trg, ok := s.Triggers["trg_before_del"]
	if !ok {
		t.Fatal("trg_before_del not found")
	}
	if trg.Timing != "BEFORE" {
		t.Errorf("Timing: want BEFORE, got %q", trg.Timing)
	}
	if trg.Event != "DELETE" {
		t.Errorf("Event: want DELETE, got %q", trg.Event)
	}
	if trg.ForEachRow {
		t.Error("ForEachRow should be false (no FOR EACH ROW clause)")
	}
}

func TestSQLiteTriggers_IgnoreTrigger(t *testing.T) {
	db := newSQLiteDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE t (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER trg_keep AFTER INSERT ON t BEGIN SELECT 1; END`); err != nil {
		t.Fatalf("create trg_keep: %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER trg_skip AFTER INSERT ON t BEGIN SELECT 2; END`); err != nil {
		t.Fatalf("create trg_skip: %v", err)
	}

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil,
		schema.LoadSchemaOptions{IgnoreTriggers: []string{"trg_skip"}})
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	if _, found := s.Triggers["trg_skip"]; found {
		t.Error("trg_skip should have been ignored")
	}
	if _, found := s.Triggers["trg_keep"]; !found {
		t.Error("trg_keep should be present")
	}
}

func TestSQLiteTriggers_CaseInsensitiveIgnore(t *testing.T) {
	db := newSQLiteDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE t (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER TRG_UPPER AFTER INSERT ON t BEGIN SELECT 1; END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil,
		schema.LoadSchemaOptions{IgnoreTriggers: []string{"trg_upper"}})
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	if _, found := s.Triggers["TRG_UPPER"]; found {
		t.Error("TRG_UPPER should have been ignored (case-insensitive match)")
	}
}

func TestSQLiteTriggers_NoTriggers_EmptyMap(t *testing.T) {
	db := newSQLiteDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE t (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	s, err := schema.LoadSchema(ctx, db, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}
	if len(s.Triggers) != 0 {
		t.Errorf("expected empty triggers map, got %d entries", len(s.Triggers))
	}
}

// ── migration generation: routines ───────────────────────────────────────────

func TestGenerateMigration_AddedRoutine_UsesDefinition(t *testing.T) {
	def := "CREATE FUNCTION fn_calc() RETURNS INT BEGIN RETURN 42; END"
	diff := schema.DiffResult{
		AddedRoutines: []schema.Routine{
			{Name: "fn_calc", Kind: "FUNCTION", Definition: def},
		},
	}
	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}
	if !strings.Contains(sql, def) {
		t.Errorf("expected routine definition in output, got:\n%s", sql)
	}
}

func TestGenerateMigration_RemovedRoutine_CommentedByDefault(t *testing.T) {
	diff := schema.DiffResult{
		RemovedRoutines: []string{"fn_old"},
	}
	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}
	if !strings.Contains(sql, "-- ") {
		t.Error("removed routine should be commented out by default")
	}
	if !strings.Contains(sql, "fn_old") {
		t.Error("removed routine name should appear in output")
	}
}

func TestGenerateMigration_RemovedRoutine_UncommentedWhenAllowed(t *testing.T) {
	diff := schema.DiffResult{
		RemovedRoutines: []string{"fn_old"},
	}
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{},
		Routines: map[string]schema.Routine{
			"fn_old": {Name: "fn_old", Kind: "FUNCTION"},
		},
	}
	opts := &schema.MigrationOptions{AllowDropRoutine: true}
	sql, err := schema.GenerateMigrationWithSchemas(diff, "mysql", opts, prodSchema)
	if err != nil {
		t.Fatalf("GenerateMigrationWithSchemas error: %v", err)
	}
	if !containsUncommented(sql, "DROP FUNCTION") {
		t.Errorf("expected uncommented DROP FUNCTION when AllowDropRoutine=true, got:\n%s", sql)
	}
}

func TestGenerateMigration_ModifiedRoutine_CommentOnly(t *testing.T) {
	diff := schema.DiffResult{
		ModifiedRoutines: []schema.RoutineDiff{
			{Name: "fn_price", DefinitionDiffers: true, ProdDefinition: "BEGIN RETURN 1; END", DevDefinition: "BEGIN RETURN 2; END"},
		},
	}
	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}
	if !strings.Contains(sql, "fn_price") {
		t.Error("modified routine name should appear in output")
	}
	// Modified routines only produce comments — no executable DROP/CREATE
	if strings.Contains(sql, "DROP FUNCTION fn_price") || strings.Contains(sql, "CREATE FUNCTION fn_price") {
		t.Error("modified routine should not produce executable DROP/CREATE statements")
	}
}

// ── migration generation: triggers ───────────────────────────────────────────

func TestGenerateMigration_AddedTrigger_UsesDefinition(t *testing.T) {
	def := "CREATE TRIGGER trg_audit AFTER INSERT ON orders FOR EACH ROW BEGIN SELECT 1; END"
	diff := schema.DiffResult{
		AddedTriggers: []schema.Trigger{
			{Name: "trg_audit", Table: "orders", Timing: "AFTER", Event: "INSERT", Definition: def, ForEachRow: true},
		},
	}
	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}
	if !strings.Contains(sql, def) {
		t.Errorf("expected trigger definition in output, got:\n%s", sql)
	}
}

func TestGenerateMigration_AddedTrigger_EmptyDefinition_Error(t *testing.T) {
	diff := schema.DiffResult{
		AddedTriggers: []schema.Trigger{
			{Name: "trg_bad", Table: "orders", Timing: "AFTER", Event: "INSERT"},
		},
	}
	_, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err == nil {
		t.Error("expected error for trigger with empty Definition, got nil")
	}
}

func TestGenerateMigration_RemovedTrigger_CommentedByDefault(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTriggers: []string{"trg_old"},
	}
	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}
	if !strings.Contains(sql, "-- ") {
		t.Error("removed trigger should be commented out by default")
	}
	if !strings.Contains(sql, "trg_old") {
		t.Error("removed trigger name should appear in output")
	}
}

func TestGenerateMigration_RemovedTrigger_UncommentedWhenAllowed(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTriggers: []string{"trg_old"},
	}
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_old": {Name: "trg_old", Table: "orders"},
		},
	}
	opts := &schema.MigrationOptions{AllowDropTrigger: true}
	sql, err := schema.GenerateMigrationWithSchemas(diff, "mysql", opts, prodSchema)
	if err != nil {
		t.Fatalf("GenerateMigrationWithSchemas error: %v", err)
	}
	if !containsUncommented(sql, "DROP TRIGGER") {
		t.Errorf("expected uncommented DROP TRIGGER when AllowDropTrigger=true, got:\n%s", sql)
	}
}

func TestGenerateMigration_RemovedTrigger_PostgreSQL_NeedsProdSchema(t *testing.T) {
	diff := schema.DiffResult{
		RemovedTriggers: []string{"trg_pg"},
	}
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{},
		Triggers: map[string]schema.Trigger{
			"trg_pg": {Name: "trg_pg", Table: "orders"},
		},
	}
	opts := &schema.MigrationOptions{AllowDropTrigger: true}
	sql, err := schema.GenerateMigrationWithSchemas(diff, "postgres", opts, prodSchema)
	if err != nil {
		t.Fatalf("GenerateMigrationWithSchemas error: %v", err)
	}
	if !strings.Contains(sql, "ON") {
		t.Errorf("PostgreSQL DROP TRIGGER should include ON <table>, got:\n%s", sql)
	}
}

func TestGenerateMigration_ModifiedTrigger_CommentOnly(t *testing.T) {
	diff := schema.DiffResult{
		ModifiedTriggers: []schema.TriggerDiff{
			{Name: "trg_sync", EventDiffers: true, ProdEvent: "INSERT", DevEvent: "UPDATE"},
		},
	}
	sql, err := schema.GenerateMigration(diff, "mysql", nil)
	if err != nil {
		t.Fatalf("GenerateMigration error: %v", err)
	}
	if !strings.Contains(sql, "trg_sync") {
		t.Error("modified trigger name should appear in output")
	}
	if containsUncommented(sql, "DROP TRIGGER") {
		t.Error("modified trigger should not produce executable DROP TRIGGER")
	}
}

// ── end-to-end pipeline: introspect → diff → migrate ─────────────────────────

func TestEndToEnd_RoutinesAndTriggers_SQLite(t *testing.T) {
	// prod: has trg_legacy (to be removed), no trg_new
	prodDB := newSQLiteDB(t)
	devDB := newSQLiteDB(t)
	ctx := context.Background()

	for _, db := range []*sql.DB{prodDB, devDB} {
		if _, err := db.ExecContext(ctx, `CREATE TABLE orders (id INTEGER PRIMARY KEY, total REAL)`); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	// prod has trg_legacy
	if _, err := prodDB.ExecContext(ctx, `
		CREATE TRIGGER trg_legacy AFTER INSERT ON orders FOR EACH ROW BEGIN SELECT 1; END
	`); err != nil {
		t.Fatalf("prod create trigger: %v", err)
	}
	// dev has trg_new
	if _, err := devDB.ExecContext(ctx, `
		CREATE TRIGGER trg_new AFTER UPDATE ON orders FOR EACH ROW BEGIN SELECT 2; END
	`); err != nil {
		t.Fatalf("dev create trigger: %v", err)
	}

	prodSchema, err := schema.LoadSchema(ctx, prodDB, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema prod: %v", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("LoadSchema dev: %v", err)
	}

	diff := schema.DiffSchemas(prodSchema, devSchema)

	if len(diff.AddedTriggers) != 1 || diff.AddedTriggers[0].Name != "trg_new" {
		t.Errorf("expected trg_new in AddedTriggers, got %v", diff.AddedTriggers)
	}
	if len(diff.RemovedTriggers) != 1 || diff.RemovedTriggers[0] != "trg_legacy" {
		t.Errorf("expected trg_legacy in RemovedTriggers, got %v", diff.RemovedTriggers)
	}

	sql, err := schema.GenerateMigration(diff, "sqlite", nil)
	if err != nil {
		t.Fatalf("GenerateMigration: %v", err)
	}
	if !strings.Contains(sql, "trg_new") {
		t.Error("migration should reference trg_new")
	}
	if !strings.Contains(sql, "trg_legacy") {
		t.Error("migration should reference trg_legacy")
	}
	if !strings.Contains(sql, "BEGIN;") || !strings.Contains(sql, "COMMIT;") {
		t.Error("migration should be wrapped in a transaction")
	}
}
