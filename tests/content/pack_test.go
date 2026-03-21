package main

import (
	"github.com/iamvirul/deepdiff-db/internal/content"

	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

func TestGeneratePack(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create dev database with test data
	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(1, 'Alice', 'alice@example.com'),
		(2, 'Bob', 'bob@example.com'),
		(3, 'Charlie', 'charlie@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "integer", IsNullable: false},
					"name":  {Name: "name", DataType: "text", IsNullable: false},
					"email": {Name: "email", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	// For unit tests, prodSchema matches devSchema
	prodSchema := devSchema

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "users",
				Added:   []string{"3"},
				Removed: []string{"1"},
				Updated: []string{"2"},
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("content.content.GeneratePack failed: %v", err)
	}

	// Verify pack file exists
	if _, err := os.Stat(packPath); os.IsNotExist(err) {
		t.Fatalf("pack file does not exist: %s", packPath)
	}

	// Read and verify pack content
	content, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(content)
	if !strings.Contains(sqlText, "BEGIN;") {
		t.Error("pack should start with BEGIN;")
	}
	if !strings.Contains(sqlText, "COMMIT;") {
		t.Error("pack should end with COMMIT;")
	}
	if !strings.Contains(sqlText, "DELETE FROM") {
		t.Error("pack should contain DELETE statements")
	}
	if !strings.Contains(sqlText, "INSERT INTO") {
		t.Error("pack should contain INSERT statements")
	}

	// Verify it's a transaction
	lines := strings.Split(strings.TrimSpace(sqlText), "\n")
	if !strings.Contains(lines[0], "BEGIN") {
		t.Error("first statement should be BEGIN")
	}
	if !strings.Contains(lines[len(lines)-1], "COMMIT") {
		t.Error("last statement should be COMMIT")
	}
}

func TestGeneratePack_NoChanges(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{},
	}

	// For unit tests, prodSchema matches devSchema
	prodSchema := devSchema

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "users"}, // No changes
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("content.content.GeneratePack failed: %v", err)
	}

	// Read pack content
	content, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(content)
	// Should only have BEGIN and COMMIT
	lines := strings.Split(strings.TrimSpace(sqlText), "\n")
	if len(lines) != 2 {
		t.Errorf("expected only BEGIN and COMMIT, got %d lines", len(lines))
	}
}

func TestGeneratePack_WithIgnore(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			updated_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, updated_at) VALUES
		(1, 'Alice', '2024-01-01')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "integer", IsNullable: false},
					"name":       {Name: "name", DataType: "text", IsNullable: false},
					"updated_at": {Name: "updated_at", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	// For unit tests, prodSchema matches devSchema
	prodSchema := devSchema

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1"},
			},
		},
	}

	ignoreFn := content.IgnoreMatcher([]string{"*.updated_at"})
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, ignoreFn, tmpDir)
	if err != nil {
		t.Fatalf("content.content.GeneratePack failed: %v", err)
	}

	content, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(content)
	// Should not include updated_at in INSERT
	if strings.Contains(sqlText, "updated_at") {
		t.Error("pack should not include ignored columns")
	}
}

func TestGeneratePack_NoPrimaryKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name:       "users",
				Columns:    map[string]schema.Column{},
				PrimaryKey: []string{}, // No primary key
			},
		},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "users", Added: []string{"1"}},
		},
	}

	prodSchema := devSchema
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	_, err = content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err == nil {
		t.Error("expected error for table without primary key")
	}
}

func TestGeneratePack_TableNotFound(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{Table: "nonexistent", Added: []string{"1"}},
		},
	}

	// content.content.GeneratePack skips tables not in schema, so it should succeed but produce empty pack
	prodSchema := devSchema
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("content.content.GeneratePack should not error for nonexistent table (it skips it): %v", err)
	}

	// Verify pack only has BEGIN and COMMIT
	content, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(content)
	// Should only have BEGIN and COMMIT, no actual statements
	lines := strings.Split(strings.TrimSpace(sqlText), "\n")
	if len(lines) > 2 {
		t.Errorf("expected only BEGIN and COMMIT, got %d lines", len(lines))
	}
}

func TestGeneratePack_MySQLDriver(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(1, 'Alice', 'alice@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "integer", IsNullable: false},
					"name":  {Name: "name", DataType: "text", IsNullable: false},
					"email": {Name: "email", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	prodSchema := devSchema

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1"},
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "mysql", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack failed: %v", err)
	}

	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(packContent)
	// MySQL should have FOREIGN_KEY_CHECKS statements
	if !strings.Contains(sqlText, "SET FOREIGN_KEY_CHECKS = 0;") {
		t.Error("MySQL pack should disable foreign key checks")
	}
	if !strings.Contains(sqlText, "SET FOREIGN_KEY_CHECKS = 1;") {
		t.Error("MySQL pack should re-enable foreign key checks")
	}
	// MySQL uses backticks for identifiers
	if !strings.Contains(sqlText, "`users`") {
		t.Error("MySQL pack should use backticks for table names")
	}
}

func TestGeneratePack_CompositePrimaryKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE orders (
			user_id INTEGER,
			order_id INTEGER,
			amount REAL,
			PRIMARY KEY (user_id, order_id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO orders (user_id, order_id, amount) VALUES
		(1, 100, 50.0),
		(1, 101, 75.0)
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"orders": {
				Name: "orders",
				Columns: map[string]schema.Column{
					"user_id":  {Name: "user_id", DataType: "integer", IsNullable: false},
					"order_id": {Name: "order_id", DataType: "integer", IsNullable: false},
					"amount":   {Name: "amount", DataType: "real", IsNullable: true},
				},
				PrimaryKey: []string{"user_id", "order_id"},
			},
		},
	}

	prodSchema := devSchema

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "orders",
				Added:   []string{"1|100"},
				Removed: []string{"1|101"},
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack failed: %v", err)
	}

	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(packContent)
	// Should have DELETE with composite key WHERE clause
	if !strings.Contains(sqlText, "DELETE FROM") {
		t.Error("pack should contain DELETE statement")
	}
	// Should have INSERT with composite key
	if !strings.Contains(sqlText, "INSERT INTO") {
		t.Error("pack should contain INSERT statement")
	}
}

func TestGeneratePack_SchemaDiffWithNewColumns(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			age INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email, age) VALUES
		(1, 'Alice', 'alice@example.com', 30)
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "integer", IsNullable: false},
					"name":  {Name: "name", DataType: "text", IsNullable: false},
					"email": {Name: "email", DataType: "text", IsNullable: true},
					"age":   {Name: "age", DataType: "integer", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	// Prod schema missing 'age' column
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "integer", IsNullable: false},
					"name":  {Name: "name", DataType: "text", IsNullable: false},
					"email": {Name: "email", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1"},
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack failed: %v", err)
	}

	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(packContent)
	// Should have ALTER TABLE to add the missing column
	if !strings.Contains(sqlText, "ALTER TABLE") {
		t.Error("pack should contain ALTER TABLE statement for new column")
	}
	if !strings.Contains(sqlText, "age") {
		t.Error("pack should contain the new column name")
	}
}

func TestGeneratePack_InvalidKeyFormat(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":   {Name: "id", DataType: "integer", IsNullable: false},
					"name": {Name: "name", DataType: "text", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	prodSchema := devSchema

	// Invalid key format (composite key format for single key)
	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1|2"}, // Invalid: single PK but composite format
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	_, err = content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err == nil {
		t.Error("expected error for invalid key format")
	}
}

func TestGeneratePack_PostgreSQLDriver(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name) VALUES (1, 'Alice')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":   {Name: "id", DataType: "integer", IsNullable: false},
					"name": {Name: "name", DataType: "text", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	prodSchema := devSchema

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1"},
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "postgres", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack failed: %v", err)
	}

	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(packContent)
	// PostgreSQL uses double quotes for identifiers
	if !strings.Contains(sqlText, `"users"`) {
		t.Error("PostgreSQL pack should use double quotes for table names")
	}
	// PostgreSQL should not have FOREIGN_KEY_CHECKS
	if strings.Contains(sqlText, "FOREIGN_KEY_CHECKS") {
		t.Error("PostgreSQL pack should not contain FOREIGN_KEY_CHECKS")
	}
}

func TestGeneratePack_WithNewColumnsAndUpdates(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			age INTEGER,
			created_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert multiple rows to test batching
	for i := 1; i <= 5; i++ {
		_, err = devDB.ExecContext(ctx, `
			INSERT INTO users (id, name, email, age, created_at) VALUES (?, ?, ?, ?, ?)
		`, i, fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@example.com", i), 20+i, "2024-01-01")
		if err != nil {
			t.Fatalf("failed to insert data: %v", err)
		}
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "integer", IsNullable: false},
					"name":       {Name: "name", DataType: "text", IsNullable: false},
					"email":      {Name: "email", DataType: "text", IsNullable: true},
					"age":        {Name: "age", DataType: "integer", IsNullable: true},
					"created_at": {Name: "created_at", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	// Prod schema missing 'age' and 'created_at' columns
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "integer", IsNullable: false},
					"name":  {Name: "name", DataType: "text", IsNullable: false},
					"email": {Name: "email", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	// Data diff with existing rows (not removed) - these will get UPDATE statements for new columns
	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "users",
				Added:   []string{"1", "2"}, // New rows
				Updated: []string{"3"},      // Updated row - will get UPDATE for new columns
				// Note: Removed rows won't get UPDATE statements
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack failed: %v", err)
	}

	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(packContent)
	// Should have ALTER TABLE statements
	if !strings.Contains(sqlText, "ALTER TABLE") {
		t.Error("pack should contain ALTER TABLE statements for new columns")
	}
	// Should have UPDATE statements for new columns (for existing rows)
	if !strings.Contains(sqlText, "UPDATE") {
		t.Error("pack should contain UPDATE statements for new columns")
	}
	// Should have INSERT statements
	if !strings.Contains(sqlText, "INSERT INTO") {
		t.Error("pack should contain INSERT statements")
	}
}

func TestGeneratePack_WithIgnoredNewColumns(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			updated_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email, updated_at) VALUES
		(1, 'Alice', 'alice@example.com', '2024-01-01')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "integer", IsNullable: false},
					"name":       {Name: "name", DataType: "text", IsNullable: false},
					"email":      {Name: "email", DataType: "text", IsNullable: true},
					"updated_at": {Name: "updated_at", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	// Prod schema missing 'updated_at' column
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "integer", IsNullable: false},
					"name":  {Name: "name", DataType: "text", IsNullable: false},
					"email": {Name: "email", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1"},
			},
		},
	}

	ignoreFn := content.IgnoreMatcher([]string{"*.updated_at"})
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, ignoreFn, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack failed: %v", err)
	}

	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(packContent)
	// Should not have ALTER TABLE for ignored column
	if strings.Contains(sqlText, "ALTER TABLE") && strings.Contains(sqlText, "updated_at") {
		t.Error("pack should not contain ALTER TABLE for ignored column")
	}
	// Should not have UPDATE for ignored column
	if strings.Contains(sqlText, "UPDATE") && strings.Contains(sqlText, "updated_at") {
		t.Error("pack should not contain UPDATE for ignored column")
	}
}

func TestGeneratePack_WithVariousDataTypes(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE test_types (
			id INTEGER PRIMARY KEY,
			name TEXT,
			age INTEGER,
			price REAL,
			is_active INTEGER,
			created_at TEXT,
			data BLOB
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert data with various types including timestamps
	_, err = devDB.ExecContext(ctx, `
		INSERT INTO test_types (id, name, age, price, is_active, created_at, data) VALUES
		(1, 'Test', 25, 99.99, 1, '2024-01-01 12:00:00', 'binary data'),
		(2, 'Test2', 30, 199.50, 0, '2024-01-02T15:04:05Z', 'more data')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"test_types": {
				Name: "test_types",
				Columns: map[string]schema.Column{
					"id":         {Name: "id", DataType: "integer", IsNullable: false},
					"name":       {Name: "name", DataType: "text", IsNullable: true},
					"age":        {Name: "age", DataType: "integer", IsNullable: true},
					"price":      {Name: "price", DataType: "real", IsNullable: true},
					"is_active":  {Name: "is_active", DataType: "integer", IsNullable: true},
					"created_at": {Name: "created_at", DataType: "text", IsNullable: true},
					"data":       {Name: "data", DataType: "blob", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	prodSchema := devSchema

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "test_types",
				Added: []string{"1", "2"},
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack failed: %v", err)
	}

	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(packContent)
	// Should contain INSERT statements with various data types
	if !strings.Contains(sqlText, "INSERT INTO") {
		t.Error("pack should contain INSERT statements")
	}
	// Should handle various data types properly
	if !strings.Contains(sqlText, "test_types") {
		t.Error("pack should reference test_types table")
	}
}

func TestGeneratePack_ErrorPaths(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":   {Name: "id", DataType: "integer", IsNullable: false},
					"name": {Name: "name", DataType: "text", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	prodSchema := devSchema

	// Test with invalid key format (wrong number of parts for composite key)
	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1|2|3"}, // Invalid: single PK but 3 parts
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	_, err = content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err == nil {
		t.Error("expected error for invalid key format")
	}
}

func TestGeneratePack_WithColumnNotFoundInSchema(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert data so fetchRow can succeed
	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "integer", IsNullable: false},
					"name":  {Name: "name", DataType: "text", IsNullable: false},
					"email": {Name: "email", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	// Prod schema has a column diff for a column that doesn't exist in dev schema
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":   {Name: "id", DataType: "integer", IsNullable: false},
					"name": {Name: "name", DataType: "text", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1"},
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	// This should succeed - columns not in dev schema are skipped
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack should handle missing columns gracefully: %v", err)
	}

	// Verify pack was created
	if _, err := os.Stat(packPath); os.IsNotExist(err) {
		t.Fatal("pack file should be created")
	}
}

func TestGeneratePack_UnsupportedDriver(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":   {Name: "id", DataType: "integer", IsNullable: false},
					"name": {Name: "name", DataType: "text", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id": {Name: "id", DataType: "integer", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1"},
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	// Test with unsupported driver - should error when trying to get column type
	_, err = content.GeneratePack(ctx, "unsupported", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err == nil {
		t.Error("expected error for unsupported driver")
	}
	if !strings.Contains(err.Error(), "unsupported driver") {
		t.Errorf("error should mention unsupported driver, got: %v", err)
	}
}

func TestGeneratePack_BuildAlterTableSQLite(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			age INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert data so fetchRow can succeed
	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, age) VALUES (1, 'Alice', 25)
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":   {Name: "id", DataType: "integer", IsNullable: false},
					"name": {Name: "name", DataType: "text", IsNullable: false},
					"age":  {Name: "age", DataType: "integer", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":   {Name: "id", DataType: "integer", IsNullable: false},
					"name": {Name: "name", DataType: "text", IsNullable: false},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1"},
			},
		},
	}

	// Test with SQLite driver - this should work
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack failed for sqlite: %v", err)
	}

	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}

	sqlText := string(packContent)
	if !strings.Contains(sqlText, "ALTER TABLE") {
		t.Error("pack should contain ALTER TABLE")
	}
	if !strings.Contains(sqlText, "age") {
		t.Error("pack should contain the new column name 'age'")
	}
	// Verify the ALTER TABLE statement format for SQLite
	if !strings.Contains(sqlText, "ALTER TABLE") || !strings.Contains(sqlText, "ADD COLUMN") {
		t.Error("pack should contain properly formatted ALTER TABLE ADD COLUMN statement")
	}
}

func TestGeneratePack_MSSQLDriver(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Use SQLite as the backing store — pack.go reads data via the DB handle
	// regardless of the driver label. The driver label only affects SQL generation
	// (quoting, transaction syntax, FK control).
	devDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(1, 'Alice', 'alice@example.com'),
		(2, 'Bob', 'bob@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {
				Name: "users",
				Columns: map[string]schema.Column{
					"id":    {Name: "id", DataType: "integer", IsNullable: false},
					"name":  {Name: "name", DataType: "text", IsNullable: false},
					"email": {Name: "email", DataType: "text", IsNullable: true},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}
	prodSchema := devSchema

	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table: "users",
				Added: []string{"1", "2"},
			},
		},
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "mssql", devDB, "", prodSchema, devSchema, schemaDiff, diff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack with mssql driver failed: %v", err)
	}

	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack file: %v", err)
	}
	sqlText := string(packContent)

	// MSSQL uses BEGIN TRANSACTION / COMMIT TRANSACTION (not BEGIN / COMMIT)
	if !strings.Contains(sqlText, "BEGIN TRANSACTION;") {
		t.Errorf("MSSQL pack should use BEGIN TRANSACTION;, got:\n%s", sqlText)
	}
	if !strings.Contains(sqlText, "COMMIT TRANSACTION;") {
		t.Errorf("MSSQL pack should use COMMIT TRANSACTION;, got:\n%s", sqlText)
	}
	if strings.Contains(sqlText, "\nBEGIN;\n") {
		t.Errorf("MSSQL pack must not use plain BEGIN;, got:\n%s", sqlText)
	}

	// MSSQL FK control uses sp_msforeachtable
	if !strings.Contains(sqlText, "sp_msforeachtable") {
		t.Errorf("MSSQL pack should use sp_msforeachtable for FK control, got:\n%s", sqlText)
	}
	if !strings.Contains(sqlText, "NOCHECK CONSTRAINT ALL") {
		t.Errorf("MSSQL pack should disable constraints with NOCHECK CONSTRAINT ALL, got:\n%s", sqlText)
	}
	if !strings.Contains(sqlText, "WITH CHECK CHECK CONSTRAINT ALL") {
		t.Errorf("MSSQL pack should re-enable constraints with WITH CHECK CHECK CONSTRAINT ALL, got:\n%s", sqlText)
	}

	// Must not contain MySQL/PostgreSQL FK syntax
	if strings.Contains(sqlText, "FOREIGN_KEY_CHECKS") {
		t.Errorf("MSSQL pack should not contain FOREIGN_KEY_CHECKS, got:\n%s", sqlText)
	}

	// MSSQL uses square-bracket quoting for table names
	if !strings.Contains(sqlText, "[users]") {
		t.Errorf("MSSQL pack should use square-bracket quoting, got:\n%s", sqlText)
	}

	// Verify INSERT statements are present
	if !strings.Contains(sqlText, "INSERT INTO") {
		t.Errorf("pack should contain INSERT statements, got:\n%s", sqlText)
	}
}

// TestGeneratePack_OracleNoData verifies Oracle-specific FK comment stmts are emitted
// when GeneratePack is called with an oracle driver but no data changes.
// The devDB is never touched when diff and schemaDiff are empty, so nil is safe.
func TestGeneratePack_OracleNoData(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	emptySchema := &schema.Schema{Tables: map[string]schema.Table{}}
	emptyDiff := content.DataDiff{}
	schemaDiff := schema.DiffSchemas(emptySchema, emptySchema)

	packPath, err := content.GeneratePack(ctx, "oracle", nil, "", emptySchema, emptySchema, schemaDiff, emptyDiff, nil, tmpDir)
	if err != nil {
		t.Fatalf("GeneratePack oracle no-data: %v", err)
	}

	sqlBytes, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("read pack: %v", err)
	}
	sqlText := string(sqlBytes)

	// Lines 105-110: Oracle FK disable comment
	if !strings.Contains(sqlText, "-- Oracle: disable all FK constraints") {
		t.Errorf("expected Oracle FK disable comment, got:\n%s", sqlText)
	}
	// Lines 306-308: Oracle FK re-enable comment
	if !strings.Contains(sqlText, "-- Oracle: re-enable all FK constraints") {
		t.Errorf("expected Oracle FK re-enable comment, got:\n%s", sqlText)
	}
	// Oracle BEGIN is plain (not BEGIN TRANSACTION like MSSQL)
	if !strings.Contains(sqlText, "BEGIN;") {
		t.Errorf("expected BEGIN; for oracle, got:\n%s", sqlText)
	}
}
