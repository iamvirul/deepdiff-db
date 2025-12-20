package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/drivers"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	"github.com/iamvirul/deepdiff-db/pkg/config"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5"
)

// TestIntegration_MySQL_FullWorkflow tests the complete workflow with MySQL databases
func TestIntegration_MySQL_FullWorkflow(t *testing.T) {
	ctx := context.Background()

	// Start MySQL containers for prod and dev
	prodContainer, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("prod_db"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start prod MySQL container: %v", err)
	}
	defer func() {
		if err := prodContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate prod container: %v", err)
		}
	}()

	devContainer, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("dev_db"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start dev MySQL container: %v", err)
	}
	defer func() {
		if err := devContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate dev container: %v", err)
		}
	}()

	// Get connection strings - use localhost for port-mapped containers
	prodPort, err := prodContainer.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatalf("failed to get prod port: %v", err)
	}

	devPort, err := devContainer.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatalf("failed to get dev port: %v", err)
	}

	prodHost := "localhost"
	devHost := "localhost"

	// Create config
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	configPath := filepath.Join(tmpDir, "test.yaml")

	cfg := &config.Config{
		Prod: config.DBConfig{
			Driver:   "mysql",
			Host:     prodHost,
			Port:     prodPort.Int(),
			User:     "testuser",
			Password: "testpass",
			Database: "prod_db",
		},
		Dev: config.DBConfig{
			Driver:   "mysql",
			Host:     devHost,
			Port:     devPort.Int(),
			User:     "testuser",
			Password: "testpass",
			Database: "dev_db",
		},
		Ignore: config.IgnoreConfig{
			Tables:  []string{},
			Columns: []string{"*.updated_at"},
		},
		Output: config.OutputConfig{
			Dir: outputDir,
		},
	}

	// Write config file
	configYAML := fmt.Sprintf(`prod:
  driver: "mysql"
  host: "%s"
  port: %d
  user: "testuser"
  password: "testpass"
  database: "prod_db"

dev:
  driver: "mysql"
  host: "%s"
  port: %d
  user: "testuser"
  password: "testpass"
  database: "dev_db"

ignore:
  tables: []
  columns:
    - "*.updated_at"

output:
  dir: "%s"
`, prodHost, prodPort.Int(), devHost, devPort.Int(), outputDir)

	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Wait a bit for containers to be fully ready
	time.Sleep(2 * time.Second)

	// Connect to databases with retry
	var prodDB, devDB *sql.DB
	for i := 0; i < 5; i++ {
		prodDB, err = drivers.Open(ctx, cfg.Prod)
		if err == nil {
			break
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		t.Fatalf("failed to connect to prod after retries: %v", err)
	}
	defer prodDB.Close()

	for i := 0; i < 5; i++ {
		devDB, err = drivers.Open(ctx, cfg.Dev)
		if err == nil {
			break
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		t.Fatalf("failed to connect to dev after retries: %v", err)
	}
	defer devDB.Close()

	// Setup prod database
	_, err = prodDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create prod users table: %v", err)
	}

	_, err = prodDB.ExecContext(ctx, `
		CREATE TABLE orders (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create prod orders table: %v", err)
	}

	_, err = prodDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(1, 'Alice', 'alice@example.com'),
		(2, 'Bob', 'bob@example.com'),
		(3, 'Charlie', 'charlie@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert prod users: %v", err)
	}

	_, err = prodDB.ExecContext(ctx, `
		INSERT INTO orders (id, user_id, amount) VALUES
		(1, 1, 100.50),
		(2, 2, 200.75)
	`)
	if err != nil {
		t.Fatalf("failed to insert prod orders: %v", err)
	}

	// Setup dev database (with different data)
	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create dev users table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE orders (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create dev orders table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(2, 'Bob Updated', 'bob.new@example.com'),
		(3, 'Charlie', 'charlie@example.com'),
		(4, 'David', 'david@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert dev users: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO orders (id, user_id, amount) VALUES
		(2, 2, 250.00),
		(3, 3, 150.25),
		(4, 4, 300.00)
	`)
	if err != nil {
		t.Fatalf("failed to insert dev orders: %v", err)
	}

	// Test 1: Schema Diff
	t.Run("SchemaDiff", func(t *testing.T) {
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "mysql", "prod_db", cfg.Ignore.Tables)
		if err != nil {
			t.Fatalf("failed to load prod schema: %v", err)
		}

		devSchema, err := schema.LoadSchema(ctx, devDB, "mysql", "dev_db", cfg.Ignore.Tables)
		if err != nil {
			t.Fatalf("failed to load dev schema: %v", err)
		}

		schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
		if err := schema.WriteReports(schemaDiff, outputDir); err != nil {
			t.Fatalf("failed to write schema reports: %v", err)
		}

		// Verify schema diff files exist
		schemaJSONPath := filepath.Join(outputDir, "schema_diff.json")
		schemaTxtPath := filepath.Join(outputDir, "schema_diff.txt")

		if _, err := os.Stat(schemaJSONPath); os.IsNotExist(err) {
			t.Fatal("schema_diff.json was not created")
		}
		if _, err := os.Stat(schemaTxtPath); os.IsNotExist(err) {
			t.Fatal("schema_diff.txt was not created")
		}

		// Verify schemas match (should have no drift)
		if schemaDiff.HasDrift() {
			t.Error("expected no schema drift")
		}
	})

	// Test 2: Full Diff (Schema + Data)
	t.Run("FullDiff", func(t *testing.T) {
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "mysql", "prod_db", cfg.Ignore.Tables)
		if err != nil {
			t.Fatalf("failed to load prod schema: %v", err)
		}

		devSchema, err := schema.LoadSchema(ctx, devDB, "mysql", "dev_db", cfg.Ignore.Tables)
		if err != nil {
			t.Fatalf("failed to load dev schema: %v", err)
		}

		ignoreColumn := content.IgnoreMatcher(cfg.Ignore.Columns)
		prodHashes := make(map[string]map[string]string)
		devHashes := make(map[string]map[string]string)

		for name, prodTable := range prodSchema.Tables {
			devTable, ok := devSchema.Tables[name]
			if !ok {
				continue
			}

			pHashes, err := content.HashTable(ctx, prodDB, "mysql", prodTable, ignoreColumn)
			if err != nil {
				t.Fatalf("failed to hash prod table %s: %v", name, err)
			}

			dHashes, err := content.HashTable(ctx, devDB, "mysql", devTable, ignoreColumn)
			if err != nil {
				t.Fatalf("failed to hash dev table %s: %v", name, err)
			}

			prodHashes[name] = pHashes
			devHashes[name] = dHashes
		}

		dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

		tablesScanned := 0
		for name := range prodSchema.Tables {
			if _, ok := devSchema.Tables[name]; ok {
				tablesScanned++
			}
		}

		if err := content.WriteReportsWithInfo(dataDiff, conflicts, outputDir, "OK", tablesScanned, ""); err != nil {
			t.Fatalf("failed to write content reports: %v", err)
		}

		// Verify all report files exist
		contentJSONPath := filepath.Join(outputDir, "content_diff.json")
		conflictsJSONPath := filepath.Join(outputDir, "conflicts.json")
		summaryTxtPath := filepath.Join(outputDir, "summary.txt")

		if _, err := os.Stat(contentJSONPath); os.IsNotExist(err) {
			t.Fatal("content_diff.json was not created")
		}
		if _, err := os.Stat(conflictsJSONPath); os.IsNotExist(err) {
			t.Fatal("conflicts.json was not created")
		}
		if _, err := os.Stat(summaryTxtPath); os.IsNotExist(err) {
			t.Fatal("summary.txt was not created")
		}

		// Verify diff detected changes
		if !dataDiff.HasChanges() {
			t.Error("expected data changes to be detected")
		}

		// Verify summary content
		summaryContent, err := os.ReadFile(summaryTxtPath)
		if err != nil {
			t.Fatalf("failed to read summary: %v", err)
		}

		summaryStr := string(summaryContent)
		if !strings.Contains(summaryStr, "Schema: OK") {
			t.Error("summary should contain schema status")
		}
		if !strings.Contains(summaryStr, "Tables scanned:") {
			t.Error("summary should contain tables scanned count")
		}

		// Verify content diff JSON structure
		contentJSON, err := os.ReadFile(contentJSONPath)
		if err != nil {
			t.Fatalf("failed to read content diff JSON: %v", err)
		}

		var diffData content.DataDiff
		if err := json.Unmarshal(contentJSON, &diffData); err != nil {
			t.Fatalf("failed to parse content diff JSON: %v", err)
		}

		if len(diffData.Tables) == 0 {
			t.Error("expected at least one table in diff")
		}
	})

	// Test 3: Generate Migration Pack
	t.Run("GeneratePack", func(t *testing.T) {
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "mysql", "prod_db", cfg.Ignore.Tables)
		if err != nil {
			t.Fatalf("failed to load prod schema: %v", err)
		}

		devSchema, err := schema.LoadSchema(ctx, devDB, "mysql", "dev_db", cfg.Ignore.Tables)
		if err != nil {
			t.Fatalf("failed to load dev schema: %v", err)
		}

		ignoreColumn := content.IgnoreMatcher(cfg.Ignore.Columns)
		prodHashes := make(map[string]map[string]string)
		devHashes := make(map[string]map[string]string)

		for name, prodTable := range prodSchema.Tables {
			devTable, ok := devSchema.Tables[name]
			if !ok {
				continue
			}

			pHashes, err := content.HashTable(ctx, prodDB, "mysql", prodTable, ignoreColumn)
			if err != nil {
				t.Fatalf("failed to hash prod table %s: %v", name, err)
			}

			dHashes, err := content.HashTable(ctx, devDB, "mysql", devTable, ignoreColumn)
			if err != nil {
				t.Fatalf("failed to hash dev table %s: %v", name, err)
			}

			prodHashes[name] = pHashes
			devHashes[name] = dHashes
		}

		dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

		schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
		packPath, err := content.GeneratePack(ctx, "mysql", devDB, "dev_db", prodSchema, devSchema, schemaDiff, dataDiff, ignoreColumn, outputDir)
		if err != nil {
			t.Fatalf("failed to generate pack: %v", err)
		}

		// Verify pack file exists
		if _, err := os.Stat(packPath); os.IsNotExist(err) {
			t.Fatalf("migration pack file does not exist: %s", packPath)
		}

		// Verify pack content
		packContent, err := os.ReadFile(packPath)
		if err != nil {
			t.Fatalf("failed to read pack: %v", err)
		}

		packStr := string(packContent)
		if !strings.Contains(packStr, "BEGIN;") {
			t.Error("pack should start with BEGIN;")
		}
		if !strings.Contains(packStr, "COMMIT;") {
			t.Error("pack should end with COMMIT;")
		}
		if !strings.Contains(packStr, "DELETE FROM") {
			t.Error("pack should contain DELETE statements")
		}
		if !strings.Contains(packStr, "INSERT INTO") {
			t.Error("pack should contain INSERT statements")
		}

		// Verify conflicts are detected
		if conflicts.HasConflicts() {
			// Verify conflicts.json exists and has content
			conflictsJSON, err := os.ReadFile(filepath.Join(outputDir, "conflicts.json"))
			if err != nil {
				t.Fatalf("failed to read conflicts.json: %v", err)
			}

			var conflictsData content.Conflicts
			if err := json.Unmarshal(conflictsJSON, &conflictsData); err != nil {
				t.Fatalf("failed to parse conflicts JSON: %v", err)
			}

			if len(conflictsData.Conflicts) == 0 {
				t.Error("expected conflicts to be present")
			}
		}
	})

	// Test 4: Apply Migration Pack
	t.Run("ApplyPack", func(t *testing.T) {
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "mysql", "prod_db", cfg.Ignore.Tables)
		if err != nil {
			t.Fatalf("failed to load prod schema: %v", err)
		}

		devSchema, err := schema.LoadSchema(ctx, devDB, "mysql", "dev_db", cfg.Ignore.Tables)
		if err != nil {
			t.Fatalf("failed to load dev schema: %v", err)
		}

		ignoreColumn := content.IgnoreMatcher(cfg.Ignore.Columns)
		prodHashes := make(map[string]map[string]string)
		devHashes := make(map[string]map[string]string)

		for name, prodTable := range prodSchema.Tables {
			devTable, ok := devSchema.Tables[name]
			if !ok {
				continue
			}

			pHashes, err := content.HashTable(ctx, prodDB, "mysql", prodTable, ignoreColumn)
			if err != nil {
				t.Fatalf("failed to hash prod table %s: %v", name, err)
			}

			dHashes, err := content.HashTable(ctx, devDB, "mysql", devTable, ignoreColumn)
			if err != nil {
				t.Fatalf("failed to hash dev table %s: %v", name, err)
			}

			prodHashes[name] = pHashes
			devHashes[name] = dHashes
		}

		dataDiff, _ := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

		packPath, err := content.GeneratePack(ctx, "mysql", devDB, prodSchema, devSchema, dataDiff, ignoreColumn, outputDir)
		if err != nil {
			t.Fatalf("failed to generate pack: %v", err)
		}

		// Read pack and strip BEGIN/COMMIT for MySQL (since ApplyPack wraps in transaction)
		packContent, err := os.ReadFile(packPath)
		if err != nil {
			t.Fatalf("failed to read pack: %v", err)
		}

		packStr := string(packContent)
		packStr = strings.ReplaceAll(packStr, "BEGIN;\n", "")
		packStr = strings.ReplaceAll(packStr, "\nCOMMIT;", "")
		packStr = strings.TrimSpace(packStr)

		packPathModified := filepath.Join(tmpDir, "migration_pack_modified.sql")
		if err := os.WriteFile(packPathModified, []byte(packStr), 0o644); err != nil {
			t.Fatalf("failed to write modified pack: %v", err)
		}

		// Test dry-run first
		if err := content.ApplyPack(ctx, prodDB, packPathModified, true); err != nil {
			t.Fatalf("dry-run failed: %v", err)
		}

		// Apply pack
		if err := content.ApplyPack(ctx, prodDB, packPathModified, false); err != nil {
			t.Fatalf("apply pack failed: %v", err)
		}

		// Verify prod database matches dev after migration
		prodHashesAfter, err := content.HashTable(ctx, prodDB, "mysql", prodSchema.Tables["users"], ignoreColumn)
		if err != nil {
			t.Fatalf("failed to hash prod users after apply: %v", err)
		}

		devHashesAfter, err := content.HashTable(ctx, devDB, "mysql", devSchema.Tables["users"], ignoreColumn)
		if err != nil {
			t.Fatalf("failed to hash dev users: %v", err)
		}

		// Compare hashes - prod should match dev after migration
		for key, devHash := range devHashesAfter {
			prodHash, ok := prodHashesAfter[key]
			if !ok {
				t.Errorf("key %s missing in prod after apply", key)
				continue
			}
			if prodHash != devHash {
				t.Errorf("hash mismatch for key %s: prod=%s dev=%s", key, prodHash, devHash)
			}
		}

		// Verify all keys in prod exist in dev
		for key := range prodHashesAfter {
			if _, ok := devHashesAfter[key]; !ok {
				t.Errorf("key %s in prod but not in dev after apply", key)
			}
		}
	})
}

// TestIntegration_PostgreSQL_FullWorkflow tests the complete workflow with PostgreSQL databases
func TestIntegration_PostgreSQL_FullWorkflow(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL containers
	prodContainer, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("prod_db"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start prod PostgreSQL container: %v", err)
	}
	defer func() {
		if err := prodContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate prod container: %v", err)
		}
	}()

	devContainer, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("dev_db"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start dev PostgreSQL container: %v", err)
	}
	defer func() {
		if err := devContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate dev container: %v", err)
		}
	}()

	// Get connection strings - use localhost for port-mapped containers
	prodPort, err := prodContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get prod port: %v", err)
	}

	devPort, err := devContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get dev port: %v", err)
	}

	prodHost := "localhost"
	devHost := "localhost"

	// Create config
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	cfg := &config.Config{
		Prod: config.DBConfig{
			Driver:   "postgres",
			Host:     prodHost,
			Port:     prodPort.Int(),
			User:     "testuser",
			Password: "testpass",
			Database: "prod_db",
		},
		Dev: config.DBConfig{
			Driver:   "postgres",
			Host:     devHost,
			Port:     devPort.Int(),
			User:     "testuser",
			Password: "testpass",
			Database: "dev_db",
		},
		Ignore: config.IgnoreConfig{
			Tables:  []string{},
			Columns: []string{},
		},
		Output: config.OutputConfig{
			Dir: outputDir,
		},
	}

	// Wait a bit for containers to be fully ready
	time.Sleep(2 * time.Second)

	// Connect to databases with retry
	var prodDB, devDB *sql.DB
	for i := 0; i < 5; i++ {
		prodDB, err = drivers.Open(ctx, cfg.Prod)
		if err == nil {
			break
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		t.Fatalf("failed to connect to prod after retries: %v", err)
	}
	defer prodDB.Close()

	for i := 0; i < 5; i++ {
		devDB, err = drivers.Open(ctx, cfg.Dev)
		if err == nil {
			break
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		t.Fatalf("failed to connect to dev after retries: %v", err)
	}
	defer devDB.Close()

	// Setup prod database
	_, err = prodDB.ExecContext(ctx, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			stock INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create prod products table: %v", err)
	}

	_, err = prodDB.ExecContext(ctx, `
		INSERT INTO products (id, name, price, stock) VALUES
		(1, 'Product A', 10.99, 100),
		(2, 'Product B', 20.50, 50),
		(3, 'Product C', 15.75, 75)
	`)
	if err != nil {
		t.Fatalf("failed to insert prod products: %v", err)
	}

	// Setup dev database with different data
	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			stock INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create dev products table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO products (id, name, price, stock) VALUES
		(2, 'Product B Updated', 25.00, 60),
		(3, 'Product C', 15.75, 75),
		(4, 'Product D', 30.00, 40)
	`)
	if err != nil {
		t.Fatalf("failed to insert dev products: %v", err)
	}

	// Run full workflow
	prodSchema, err := schema.LoadSchema(ctx, prodDB, "postgres", "prod_db", cfg.Ignore.Tables)
	if err != nil {
		t.Fatalf("failed to load prod schema: %v", err)
	}

	devSchema, err := schema.LoadSchema(ctx, devDB, "postgres", "dev_db", cfg.Ignore.Tables)
	if err != nil {
		t.Fatalf("failed to load dev schema: %v", err)
	}

	// Schema diff
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	if err := schema.WriteReports(schemaDiff, outputDir); err != nil {
		t.Fatalf("failed to write schema reports: %v", err)
	}

	// Data diff
	ignoreColumn := content.IgnoreMatcher(cfg.Ignore.Columns)
	prodHashes := make(map[string]map[string]string)
	devHashes := make(map[string]map[string]string)

	for name, prodTable := range prodSchema.Tables {
		devTable, ok := devSchema.Tables[name]
		if !ok {
			continue
		}

		pHashes, err := content.HashTable(ctx, prodDB, "postgres", prodTable, ignoreColumn)
		if err != nil {
			t.Fatalf("failed to hash prod table %s: %v", name, err)
		}

		dHashes, err := content.HashTable(ctx, devDB, "postgres", devTable, ignoreColumn)
		if err != nil {
			t.Fatalf("failed to hash dev table %s: %v", name, err)
		}

		prodHashes[name] = pHashes
		devHashes[name] = dHashes
	}

	dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	tablesScanned := 0
	for name := range prodSchema.Tables {
		if _, ok := devSchema.Tables[name]; ok {
			tablesScanned++
		}
	}

	// Generate pack
	packPath, err := content.GeneratePack(ctx, "postgres", devDB, "dev_db", prodSchema, devSchema, schemaDiff, dataDiff, ignoreColumn, outputDir)
	if err != nil {
		t.Fatalf("failed to generate pack: %v", err)
	}

	// Write reports
	if err := content.WriteReportsWithInfo(dataDiff, conflicts, outputDir, "OK", tablesScanned, packPath); err != nil {
		t.Fatalf("failed to write reports: %v", err)
	}

	// Verify all report files exist
	requiredFiles := []string{
		"schema_diff.json",
		"schema_diff.txt",
		"content_diff.json",
		"conflicts.json",
		"summary.txt",
		"migration_pack.sql",
	}

	for _, filename := range requiredFiles {
		filePath := filepath.Join(outputDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("required file %s was not created", filename)
		}
	}

	// Verify summary contains migration pack reference
	summaryContent, err := os.ReadFile(filepath.Join(outputDir, "summary.txt"))
	if err != nil {
		t.Fatalf("failed to read summary: %v", err)
	}

	if !strings.Contains(string(summaryContent), "Migration pack:") {
		t.Error("summary should contain migration pack reference")
	}

	// Apply pack (strip BEGIN/COMMIT for PostgreSQL)
	packContent, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("failed to read pack: %v", err)
	}

	packStr := string(packContent)
	packStr = strings.ReplaceAll(packStr, "BEGIN;\n", "")
	packStr = strings.ReplaceAll(packStr, "\nCOMMIT;", "")
	packStr = strings.TrimSpace(packStr)

	packPathModified := filepath.Join(tmpDir, "migration_pack_modified.sql")
	if err := os.WriteFile(packPathModified, []byte(packStr), 0o644); err != nil {
		t.Fatalf("failed to write modified pack: %v", err)
	}

	// Apply pack
	if err := content.ApplyPack(ctx, prodDB, packPathModified, false); err != nil {
		t.Fatalf("apply pack failed: %v", err)
	}

	// Verify data matches after migration
	prodHashesAfter, err := content.HashTable(ctx, prodDB, "postgres", prodSchema.Tables["products"], ignoreColumn)
	if err != nil {
		t.Fatalf("failed to hash prod products after apply: %v", err)
	}

	devHashesAfter, err := content.HashTable(ctx, devDB, "postgres", devSchema.Tables["products"], ignoreColumn)
	if err != nil {
		t.Fatalf("failed to hash dev products: %v", err)
	}

	// Verify all keys match
	if len(prodHashesAfter) != len(devHashesAfter) {
		t.Errorf("hash count mismatch: prod=%d dev=%d", len(prodHashesAfter), len(devHashesAfter))
	}

	for key, devHash := range devHashesAfter {
		prodHash, ok := prodHashesAfter[key]
		if !ok {
			t.Errorf("key %s missing in prod after apply", key)
			continue
		}
		if prodHash != devHash {
			t.Errorf("hash mismatch for key %s: prod=%s dev=%s", key, prodHash, devHash)
		}
	}
}

// TestIntegration_AllReportsGenerated verifies all report files are created correctly
func TestIntegration_AllReportsGenerated(t *testing.T) {
	ctx := context.Background()

	// Use SQLite for faster tests
	tmpDir := t.TempDir()
	prodDBPath := filepath.Join(tmpDir, "prod.db")
	devDBPath := filepath.Join(tmpDir, "dev.db")
	outputDir := filepath.Join(tmpDir, "output")

	prodDB, err := sql.Open("sqlite", prodDBPath)
	if err != nil {
		t.Fatalf("failed to open prod database: %v", err)
	}
	defer prodDB.Close()

	devDB, err := sql.Open("sqlite", devDBPath)
	if err != nil {
		t.Fatalf("failed to open dev database: %v", err)
	}
	defer devDB.Close()

	// Setup databases
	_, err = prodDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create prod table: %v", err)
	}

	_, err = prodDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(1, 'Alice', 'alice@example.com'),
		(2, 'Bob', 'bob@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert prod data: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create dev table: %v", err)
	}

	_, err = devDB.ExecContext(ctx, `
		INSERT INTO users (id, name, email) VALUES
		(2, 'Bob Updated', 'bob.new@example.com'),
		(3, 'Charlie', 'charlie@example.com')
	`)
	if err != nil {
		t.Fatalf("failed to insert dev data: %v", err)
	}

	// Load schemas
	prodSchema, err := schema.LoadSchema(ctx, prodDB, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load prod schema: %v", err)
	}

	devSchema, err := schema.LoadSchema(ctx, devDB, "sqlite", "", nil)
	if err != nil {
		t.Fatalf("failed to load dev schema: %v", err)
	}

	// Generate schema diff reports
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	if err := schema.WriteReports(schemaDiff, outputDir); err != nil {
		t.Fatalf("failed to write schema reports: %v", err)
	}

	// Generate data diff
	ignoreColumn := content.IgnoreMatcher(nil)
	prodHashes := make(map[string]map[string]string)
	devHashes := make(map[string]map[string]string)

	for name, prodTable := range prodSchema.Tables {
		devTable, ok := devSchema.Tables[name]
		if !ok {
			continue
		}

		pHashes, err := content.HashTable(ctx, prodDB, "sqlite", prodTable, ignoreColumn)
		if err != nil {
			t.Fatalf("failed to hash prod table %s: %v", name, err)
		}

		dHashes, err := content.HashTable(ctx, devDB, "sqlite", devTable, ignoreColumn)
		if err != nil {
			t.Fatalf("failed to hash dev table %s: %v", name, err)
		}

		prodHashes[name] = pHashes
		devHashes[name] = dHashes
	}

	dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	// Generate pack
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	packPath, err := content.GeneratePack(ctx, "sqlite", devDB, "", prodSchema, devSchema, schemaDiff, dataDiff, ignoreColumn, outputDir)
	if err != nil {
		t.Fatalf("failed to generate pack: %v", err)
	}

	tablesScanned := 0
	for name := range prodSchema.Tables {
		if _, ok := devSchema.Tables[name]; ok {
			tablesScanned++
		}
	}

	// Write all reports
	if err := content.WriteReportsWithInfo(dataDiff, conflicts, outputDir, "OK", tablesScanned, packPath); err != nil {
		t.Fatalf("failed to write reports: %v", err)
	}

	// Verify all required files exist
	requiredFiles := map[string]func(string) error{
		"schema_diff.json": func(path string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var result schema.DiffResult
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return nil
		},
		"schema_diff.txt": func(path string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if len(data) == 0 {
				return fmt.Errorf("file is empty")
			}
			return nil
		},
		"content_diff.json": func(path string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var result content.DataDiff
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return nil
		},
		"conflicts.json": func(path string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var result content.Conflicts
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return nil
		},
		"summary.txt": func(path string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)
			if !strings.Contains(content, "Schema:") {
				return fmt.Errorf("summary missing schema status")
			}
			if !strings.Contains(content, "Tables scanned:") {
				return fmt.Errorf("summary missing tables scanned")
			}
			return nil
		},
		"migration_pack.sql": func(path string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)
			if !strings.Contains(content, "BEGIN;") {
				return fmt.Errorf("pack missing BEGIN")
			}
			if !strings.Contains(content, "COMMIT;") {
				return fmt.Errorf("pack missing COMMIT")
			}
			return nil
		},
	}

	for filename, validator := range requiredFiles {
		filePath := filepath.Join(outputDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("required file %s was not created", filename)
			continue
		}

		if err := validator(filePath); err != nil {
			t.Errorf("file %s validation failed: %v", filename, err)
		}
	}

	// Verify report contents are meaningful
	schemaJSON, _ := os.ReadFile(filepath.Join(outputDir, "schema_diff.json"))
	contentJSON, _ := os.ReadFile(filepath.Join(outputDir, "content_diff.json"))
	summaryTxt, _ := os.ReadFile(filepath.Join(outputDir, "summary.txt"))

	if len(schemaJSON) == 0 {
		t.Error("schema_diff.json is empty")
	}
	if len(contentJSON) == 0 {
		t.Error("content_diff.json is empty")
	}
	if len(summaryTxt) == 0 {
		t.Error("summary.txt is empty")
	}

	// Verify summary contains expected information
	summaryStr := string(summaryTxt)
	expectedFields := []string{
		"Schema:",
		"Tables scanned:",
		"Added rows:",
		"Removed rows:",
		"Updated rows:",
		"Migration pack:",
	}

	for _, field := range expectedFields {
		if !strings.Contains(summaryStr, field) {
			t.Errorf("summary missing field: %s", field)
		}
	}
}

