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
	"github.com/testcontainers/testcontainers-go/modules/mssql"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
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

			pHashes, err := content.HashTable(ctx, prodDB, "mysql", prodTable, ignoreColumn, 0)
			if err != nil {
				t.Fatalf("failed to hash prod table %s: %v", name, err)
			}

			dHashes, err := content.HashTable(ctx, devDB, "mysql", devTable, ignoreColumn, 0)
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

			pHashes, err := content.HashTable(ctx, prodDB, "mysql", prodTable, ignoreColumn, 0)
			if err != nil {
				t.Fatalf("failed to hash prod table %s: %v", name, err)
			}

			dHashes, err := content.HashTable(ctx, devDB, "mysql", devTable, ignoreColumn, 0)
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

			pHashes, err := content.HashTable(ctx, prodDB, "mysql", prodTable, ignoreColumn, 0)
			if err != nil {
				t.Fatalf("failed to hash prod table %s: %v", name, err)
			}

			dHashes, err := content.HashTable(ctx, devDB, "mysql", devTable, ignoreColumn, 0)
			if err != nil {
				t.Fatalf("failed to hash dev table %s: %v", name, err)
			}

			prodHashes[name] = pHashes
			devHashes[name] = dHashes
		}

		dataDiff, _ := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

		schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
		packPath, err := content.GeneratePack(ctx, "mysql", devDB, "dev_db", prodSchema, devSchema, schemaDiff, dataDiff, ignoreColumn, outputDir)
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
		prodHashesAfter, err := content.HashTable(ctx, prodDB, "mysql", prodSchema.Tables["users"], ignoreColumn, 0)
		if err != nil {
			t.Fatalf("failed to hash prod users after apply: %v", err)
		}

		devHashesAfter, err := content.HashTable(ctx, devDB, "mysql", devSchema.Tables["users"], ignoreColumn, 0)
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

		pHashes, err := content.HashTable(ctx, prodDB, "postgres", prodTable, ignoreColumn, 0)
		if err != nil {
			t.Fatalf("failed to hash prod table %s: %v", name, err)
		}

		dHashes, err := content.HashTable(ctx, devDB, "postgres", devTable, ignoreColumn, 0)
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
	prodHashesAfter, err := content.HashTable(ctx, prodDB, "postgres", prodSchema.Tables["products"], ignoreColumn, 0)
	if err != nil {
		t.Fatalf("failed to hash prod products after apply: %v", err)
	}

	devHashesAfter, err := content.HashTable(ctx, devDB, "postgres", devSchema.Tables["products"], ignoreColumn, 0)
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

// TestIntegration_MSSQL_FullWorkflow tests the complete schema + content diff workflow
// against two real MSSQL 2022 containers. The test verifies that:
//   - Schema is loaded correctly (columns, data types, PKs, indexes, FKs)
//   - Row hashing via keyset pagination produces the correct diff
//   - A migration pack with MSSQL-compatible SQL is generated
//
// The MSSQL containers use SA authentication with a strong password to satisfy
// SQL Server's password complexity policy. ACCEPT_EULA=Y is required by Microsoft.
func TestIntegration_MSSQL_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MSSQL integration test in -short mode")
	}

	ctx := context.Background()

	const mssqlImage = "mcr.microsoft.com/mssql/server:2022-latest"
	const saPassword = "StrongP@ss1word!"

	mssqlWait := wait.ForLog("SQL Server is now ready for client connections").
		WithOccurrence(1).
		WithStartupTimeout(120 * time.Second)

	// Start prod container.
	prodContainer, err := mssql.Run(ctx, mssqlImage,
		mssql.WithAcceptEULA(),
		mssql.WithPassword(saPassword),
		testcontainers.WithWaitStrategy(mssqlWait),
	)
	if err != nil {
		t.Fatalf("failed to start prod MSSQL container: %v", err)
	}
	defer func() {
		if err := prodContainer.Terminate(ctx); err != nil {
			t.Logf("warn: failed to terminate prod container: %v", err)
		}
	}()

	// Start dev container.
	devContainer, err := mssql.Run(ctx, mssqlImage,
		mssql.WithAcceptEULA(),
		mssql.WithPassword(saPassword),
		testcontainers.WithWaitStrategy(mssqlWait),
	)
	if err != nil {
		t.Fatalf("failed to start dev MSSQL container: %v", err)
	}
	defer func() {
		if err := devContainer.Terminate(ctx); err != nil {
			t.Logf("warn: failed to terminate dev container: %v", err)
		}
	}()

	prodPort, err := prodContainer.MappedPort(ctx, "1433")
	if err != nil {
		t.Fatalf("get prod port: %v", err)
	}
	devPort, err := devContainer.MappedPort(ctx, "1433")
	if err != nil {
		t.Fatalf("get dev port: %v", err)
	}

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	makeCfg := func(port int, database string) config.DBConfig {
		return config.DBConfig{
			Driver:   "mssql",
			Host:     "localhost",
			Port:     port,
			User:     "sa",
			Password: saPassword,
			Database: database,
		}
	}

	// Connect with retries — MSSQL can be slow even after the log appears.
	connectWithRetry := func(cfg config.DBConfig, name string) *sql.DB {
		t.Helper()
		var (
			db  *sql.DB
			err error
		)
		for i := 0; i < 10; i++ {
			db, err = drivers.Open(ctx, cfg)
			if err == nil {
				return db
			}
			time.Sleep(3 * time.Second)
		}
		t.Fatalf("failed to connect to %s MSSQL after retries: %v", name, err)
		return nil
	}

	// Bootstrap: connect to master only to create dedicated test databases.
	// Using master directly would pick up system tables (spt_values, spt_fallback_usg,
	// etc.) that have no primary key and cause HashTable to fail.
	const testDB = "deepdifftest"
	for _, side := range []struct {
		port int
		name string
	}{
		{prodPort.Int(), "prod"},
		{devPort.Int(), "dev"},
	} {
		bootstrap := connectWithRetry(makeCfg(side.port, "master"), side.name+"-bootstrap")
		if _, err := bootstrap.ExecContext(ctx, "CREATE DATABASE "+testDB); err != nil {
			bootstrap.Close()
			t.Fatalf("[%s] create database: %v", side.name, err)
		}
		bootstrap.Close()
	}

	prodCfg := makeCfg(prodPort.Int(), testDB)
	devCfg := makeCfg(devPort.Int(), testDB)

	prodDB := connectWithRetry(prodCfg, "prod")
	defer prodDB.Close()

	devDB := connectWithRetry(devCfg, "dev")
	defer devDB.Close()

	// ---------- Schema setup ----------

	prodSetup := []string{
		`CREATE TABLE users (
			id    INT           NOT NULL,
			name  NVARCHAR(100) NOT NULL,
			email NVARCHAR(100) NULL,
			PRIMARY KEY (id)
		)`,
		`CREATE TABLE orders (
			id      INT            NOT NULL,
			user_id INT            NOT NULL,
			amount  DECIMAL(10,2)  NOT NULL,
			PRIMARY KEY (id),
			CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users (id)
		)`,
		`CREATE INDEX idx_orders_user ON orders (user_id)`,
		`INSERT INTO users  (id, name, email) VALUES (1,'Alice','alice@example.com'),(2,'Bob','bob@example.com'),(3,'Charlie','charlie@example.com')`,
		`INSERT INTO orders (id, user_id, amount) VALUES (1,1,100.50),(2,2,200.75)`,
	}
	devSetup := []string{
		`CREATE TABLE users (
			id    INT           NOT NULL,
			name  NVARCHAR(100) NOT NULL,
			email NVARCHAR(100) NULL,
			PRIMARY KEY (id)
		)`,
		`CREATE TABLE orders (
			id      INT            NOT NULL,
			user_id INT            NOT NULL,
			amount  DECIMAL(10,2)  NOT NULL,
			PRIMARY KEY (id),
			CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users (id)
		)`,
		`CREATE INDEX idx_orders_user ON orders (user_id)`,
		// Bob updated, Charlie unchanged, David new
		`INSERT INTO users  (id, name, email) VALUES (2,'Bob Updated','bob.new@example.com'),(3,'Charlie','charlie@example.com'),(4,'David','david@example.com')`,
		// order 2 updated amount, order 3+4 new
		`INSERT INTO orders (id, user_id, amount) VALUES (2,2,250.00),(3,3,150.25),(4,4,300.00)`,
	}

	execStmts := func(db *sql.DB, label string, stmts []string) {
		t.Helper()
		for _, stmt := range stmts {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				t.Fatalf("[%s] exec failed: %v\nSQL: %s", label, err, stmt)
			}
		}
	}
	execStmts(prodDB, "prod", prodSetup)
	execStmts(devDB, "dev", devSetup)

	// ---------- Test: SchemaDiff ----------
	t.Run("SchemaDiff", func(t *testing.T) {
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "mssql", testDB, nil)
		if err != nil {
			t.Fatalf("load prod schema: %v", err)
		}
		devSchema, err := schema.LoadSchema(ctx, devDB, "mssql", testDB, nil)
		if err != nil {
			t.Fatalf("load dev schema: %v", err)
		}

		if _, ok := prodSchema.Tables["users"]; !ok {
			t.Fatal("prod schema missing 'users' table")
		}
		if _, ok := prodSchema.Tables["orders"]; !ok {
			t.Fatal("prod schema missing 'orders' table")
		}

		// Both databases have identical schemas — no drift expected.
		diff := schema.DiffSchemas(prodSchema, devSchema)
		if diff.HasDrift() {
			t.Errorf("unexpected schema drift: %+v", diff)
		}

		// Primary keys should be detected.
		for _, tbl := range []string{"users", "orders"} {
			if len(prodSchema.Tables[tbl].PrimaryKey) == 0 {
				t.Errorf("table %s has no primary key in prod schema", tbl)
			}
		}

		// Index on orders should be detected.
		if _, ok := prodSchema.Tables["orders"].Indexes["idx_orders_user"]; !ok {
			t.Error("expected index 'idx_orders_user' on orders table")
		}

		// FK should be detected.
		if _, ok := prodSchema.Tables["orders"].ForeignKeys["fk_orders_user"]; !ok {
			t.Error("expected FK 'fk_orders_user' on orders table")
		}

		if err := schema.WriteReports(diff, outputDir); err != nil {
			t.Fatalf("write schema reports: %v", err)
		}
	})

	// ---------- Test: ContentDiff ----------
	t.Run("ContentDiff", func(t *testing.T) {
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "mssql", testDB, nil)
		if err != nil {
			t.Fatalf("load prod schema: %v", err)
		}
		devSchema, err := schema.LoadSchema(ctx, devDB, "mssql", testDB, nil)
		if err != nil {
			t.Fatalf("load dev schema: %v", err)
		}

		ignoreFn := content.IgnoreMatcher(nil)
		prodHashes := make(map[string]map[string]string)
		devHashes := make(map[string]map[string]string)

		for name, pt := range prodSchema.Tables {
			dt, ok := devSchema.Tables[name]
			if !ok {
				continue
			}
			ph, err := content.HashTable(ctx, prodDB, "mssql", pt, ignoreFn, 0)
			if err != nil {
				t.Fatalf("hash prod %s: %v", name, err)
			}
			dh, err := content.HashTable(ctx, devDB, "mssql", dt, ignoreFn, 0)
			if err != nil {
				t.Fatalf("hash dev %s: %v", name, err)
			}
			prodHashes[name] = ph
			devHashes[name] = dh
		}

		dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)
		if !dataDiff.HasChanges() {
			t.Error("expected data changes to be detected")
		}

		tablesScanned := 0
		for name := range prodSchema.Tables {
			if _, ok := devSchema.Tables[name]; ok {
				tablesScanned++
			}
		}
		if err := content.WriteReportsWithInfo(dataDiff, conflicts, outputDir, "OK", tablesScanned, ""); err != nil {
			t.Fatalf("write content reports: %v", err)
		}

		// Verify content_diff.json exists and is valid.
		contentJSONPath := filepath.Join(outputDir, "content_diff.json")
		if _, err := os.Stat(contentJSONPath); os.IsNotExist(err) {
			t.Fatal("content_diff.json was not created")
		}
	})

	// ---------- Test: GeneratePack ----------
	t.Run("GeneratePack", func(t *testing.T) {
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "mssql", testDB, nil)
		if err != nil {
			t.Fatalf("load prod schema: %v", err)
		}
		devSchema, err := schema.LoadSchema(ctx, devDB, "mssql", testDB, nil)
		if err != nil {
			t.Fatalf("load dev schema: %v", err)
		}

		ignoreFn := content.IgnoreMatcher(nil)
		prodHashes := make(map[string]map[string]string)
		devHashes := make(map[string]map[string]string)
		for name, pt := range prodSchema.Tables {
			dt, ok := devSchema.Tables[name]
			if !ok {
				continue
			}
			ph, err := content.HashTable(ctx, prodDB, "mssql", pt, ignoreFn, 0)
			if err != nil {
				t.Fatalf("hash prod %s: %v", name, err)
			}
			dh, err := content.HashTable(ctx, devDB, "mssql", dt, ignoreFn, 0)
			if err != nil {
				t.Fatalf("hash dev %s: %v", name, err)
			}
			prodHashes[name] = ph
			devHashes[name] = dh
		}

		schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
		dataDiff, _ := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

		packPath, err := content.GeneratePack(ctx, "mssql", devDB, testDB, prodSchema, devSchema, schemaDiff, dataDiff, ignoreFn, outputDir)
		if err != nil {
			t.Fatalf("generate pack: %v", err)
		}

		packContent, err := os.ReadFile(packPath)
		if err != nil {
			t.Fatalf("read pack: %v", err)
		}

		packStr := string(packContent)
		// MSSQL uses BEGIN TRANSACTION / COMMIT TRANSACTION (not bare BEGIN/COMMIT).
		if !strings.Contains(packStr, "BEGIN TRANSACTION;") {
			t.Errorf("pack should contain BEGIN TRANSACTION;, got:\n%s", packStr)
		}
		if !strings.Contains(packStr, "COMMIT TRANSACTION;") {
			t.Errorf("pack should contain COMMIT TRANSACTION;, got:\n%s", packStr)
		}
		// MSSQL uses square-bracket quoting.
		if !strings.Contains(packStr, "[users]") && !strings.Contains(packStr, "[orders]") {
			t.Error("pack should contain MSSQL-quoted table names")
		}
	})
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

		pHashes, err := content.HashTable(ctx, prodDB, "sqlite", prodTable, ignoreColumn, 0)
		if err != nil {
			t.Fatalf("failed to hash prod table %s: %v", name, err)
		}

		dHashes, err := content.HashTable(ctx, devDB, "sqlite", devTable, ignoreColumn, 0)
		if err != nil {
			t.Fatalf("failed to hash dev table %s: %v", name, err)
		}

		prodHashes[name] = pHashes
		devHashes[name] = dHashes
	}

	dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	// Generate pack
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


// TestIntegration_Oracle_FullWorkflow tests the complete workflow with Oracle XE 21c databases.
// It uses gvenzl/oracle-xe:21-slim-faststart via the generic testcontainers API (no official Oracle module).
// Oracle startup takes ~2-3 minutes; the test waits on "DATABASE IS READY TO USE!" in logs.
func TestIntegration_Oracle_FullWorkflow(t *testing.T) {
	ctx := context.Background()

	const (
		oraclePassword = "StrongP@ss1word!" // satisfies Oracle complexity requirements
		oracleService  = "XEPDB1"           // Oracle 21c XE pluggable database
		oracleUser     = "system"
		oraclePort     = "1521/tcp"
	)

	// Start prod Oracle container
	prodReq := testcontainers.ContainerRequest{
		Image: "gvenzl/oracle-xe:21-slim-faststart",
		Env: map[string]string{
			"ORACLE_PASSWORD": oraclePassword,
		},
		ExposedPorts: []string{oraclePort},
		WaitingFor:   wait.ForLog("DATABASE IS READY TO USE!").WithStartupTimeout(5 * time.Minute),
	}
	prodContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: prodReq,
		Started:          true,
	})
	if err != nil {
		t.Skipf("Oracle prod container unavailable (Docker or image issue): %v", err)
	}
	defer func() {
		if err := prodContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate Oracle prod container: %v", err)
		}
	}()

	// Start dev Oracle container
	devReq := testcontainers.ContainerRequest{
		Image: "gvenzl/oracle-xe:21-slim-faststart",
		Env: map[string]string{
			"ORACLE_PASSWORD": oraclePassword,
		},
		ExposedPorts: []string{oraclePort},
		WaitingFor:   wait.ForLog("DATABASE IS READY TO USE!").WithStartupTimeout(5 * time.Minute),
	}
	devContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: devReq,
		Started:          true,
	})
	if err != nil {
		t.Skipf("Oracle dev container unavailable (Docker or image issue): %v", err)
	}
	defer func() {
		if err := devContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate Oracle dev container: %v", err)
		}
	}()

	// Resolve host/port for each container
	prodHost, err := prodContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Oracle prod container host: %v", err)
	}
	prodMappedPort, err := prodContainer.MappedPort(ctx, oraclePort)
	if err != nil {
		t.Fatalf("Oracle prod container port: %v", err)
	}

	devHost, err := devContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Oracle dev container host: %v", err)
	}
	devMappedPort, err := devContainer.MappedPort(ctx, oraclePort)
	if err != nil {
		t.Fatalf("Oracle dev container port: %v", err)
	}

	prodDSN := fmt.Sprintf("oracle://%s:%s@%s:%s/%s", oracleUser, oraclePassword, prodHost, prodMappedPort.Port(), oracleService)
	devDSN := fmt.Sprintf("oracle://%s:%s@%s:%s/%s", oracleUser, oraclePassword, devHost, devMappedPort.Port(), oracleService)

	prodDB, err := sql.Open("oracle", prodDSN)
	if err != nil {
		t.Fatalf("open Oracle prod DB: %v", err)
	}
	defer prodDB.Close()

	devDB, err := sql.Open("oracle", devDSN)
	if err != nil {
		t.Fatalf("open Oracle dev DB: %v", err)
	}
	defer devDB.Close()

	// Seed prod: users table + orders table with FK, data
	prodSetup := []string{
		`CREATE TABLE USERS (
			ID       NUMBER(10) NOT NULL,
			USERNAME VARCHAR2(100) NOT NULL,
			EMAIL    VARCHAR2(255) NOT NULL,
			STATUS   VARCHAR2(20) DEFAULT 'active',
			CONSTRAINT PK_USERS PRIMARY KEY (ID)
		)`,
		`CREATE TABLE ORDERS (
			ID          NUMBER(10) NOT NULL,
			USER_ID     NUMBER(10) NOT NULL,
			AMOUNT      NUMBER(10,2) NOT NULL,
			DESCRIPTION VARCHAR2(500),
			CONSTRAINT PK_ORDERS PRIMARY KEY (ID),
			CONSTRAINT FK_ORDERS_USERS FOREIGN KEY (USER_ID) REFERENCES USERS(ID)
		)`,
		`CREATE INDEX IDX_ORDERS_USER_ID ON ORDERS(USER_ID)`,
		`INSERT INTO USERS (ID, USERNAME, EMAIL) VALUES (1, 'alice', 'alice@example.com')`,
		`INSERT INTO USERS (ID, USERNAME, EMAIL) VALUES (2, 'bob', 'bob@example.com')`,
		`INSERT INTO ORDERS (ID, USER_ID, AMOUNT, DESCRIPTION) VALUES (1, 1, 99.99, 'Order A')`,
		`INSERT INTO ORDERS (ID, USER_ID, AMOUNT, DESCRIPTION) VALUES (2, 2, 49.99, 'Order B')`,
	}
	for _, stmt := range prodSetup {
		if _, err := prodDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("Oracle prod setup %q: %v", stmt[:40], err)
		}
	}

	// Seed dev: same tables with schema drift (extra column, new user, modified order)
	devSetup := []string{
		`CREATE TABLE USERS (
			ID         NUMBER(10) NOT NULL,
			USERNAME   VARCHAR2(100) NOT NULL,
			EMAIL      VARCHAR2(255) NOT NULL,
			STATUS     VARCHAR2(20) DEFAULT 'active',
			CREATED_AT DATE DEFAULT SYSDATE,
			CONSTRAINT PK_USERS PRIMARY KEY (ID)
		)`,
		`CREATE TABLE ORDERS (
			ID          NUMBER(10) NOT NULL,
			USER_ID     NUMBER(10) NOT NULL,
			AMOUNT      NUMBER(10,2) NOT NULL,
			DESCRIPTION VARCHAR2(500),
			CONSTRAINT PK_ORDERS PRIMARY KEY (ID),
			CONSTRAINT FK_ORDERS_USERS FOREIGN KEY (USER_ID) REFERENCES USERS(ID)
		)`,
		`CREATE INDEX IDX_ORDERS_USER_ID ON ORDERS(USER_ID)`,
		`INSERT INTO USERS (ID, USERNAME, EMAIL, CREATED_AT) VALUES (1, 'alice', 'alice@example.com', SYSDATE)`,
		`INSERT INTO USERS (ID, USERNAME, EMAIL, CREATED_AT) VALUES (2, 'bob', 'bob@example.com', SYSDATE)`,
		`INSERT INTO USERS (ID, USERNAME, EMAIL, CREATED_AT) VALUES (3, 'carol', 'carol@example.com', SYSDATE)`,
		`INSERT INTO ORDERS (ID, USER_ID, AMOUNT, DESCRIPTION) VALUES (1, 1, 149.99, 'Order A updated')`,
		`INSERT INTO ORDERS (ID, USER_ID, AMOUNT, DESCRIPTION) VALUES (2, 2, 49.99, 'Order B')`,
	}
	for _, stmt := range devSetup {
		if _, err := devDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("Oracle dev setup %q: %v", stmt[:40], err)
		}
	}

	outputDir := t.TempDir()

	prodCfg := config.DBConfig{Driver: "oracle", Host: prodHost, Port: prodMappedPort.Int(), User: oracleUser, Password: oraclePassword, Database: oracleService}
	devCfg := config.DBConfig{Driver: "oracle", Host: devHost, Port: devMappedPort.Int(), User: oracleUser, Password: oraclePassword, Database: oracleService}

	ignoreColumn := func(_, _ string) bool { return false }
	ignoreTables := []string{}

	t.Run("SchemaDiff", func(t *testing.T) {
		_ = prodCfg
		_ = devCfg
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "oracle", oracleService, ignoreTables)
		if err != nil {
			t.Fatalf("load prod schema: %v", err)
		}
		devSchema, err := schema.LoadSchema(ctx, devDB, "oracle", oracleService, ignoreTables)
		if err != nil {
			t.Fatalf("load dev schema: %v", err)
		}

		schemaDiff := schema.DiffSchemas(prodSchema, devSchema)

		if err := schema.WriteReports(schemaDiff, outputDir); err != nil {
			t.Fatalf("write schema reports: %v", err)
		}

		// USERS table should show CREATED_AT as missing in prod
		var usersDiff *schema.TableDiff
		for i := range schemaDiff.Tables {
			if schemaDiff.Tables[i].Table == "USERS" {
				usersDiff = &schemaDiff.Tables[i]
				break
			}
		}
		if usersDiff == nil {
			t.Fatal("expected USERS table in schema diff")
		}
		if !usersDiff.HasDifferences {
			t.Error("expected USERS table to have schema differences")
		}
	})

	t.Run("ContentDiff", func(t *testing.T) {
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "oracle", oracleService, ignoreTables)
		if err != nil {
			t.Fatalf("load prod schema: %v", err)
		}
		devSchema, err := schema.LoadSchema(ctx, devDB, "oracle", oracleService, ignoreTables)
		if err != nil {
			t.Fatalf("load dev schema: %v", err)
		}

		for _, tbl := range prodSchema.Tables {
			ph, err := content.HashTable(ctx, prodDB, "oracle", tbl, ignoreColumn, 0)
			if err != nil {
				t.Fatalf("hash prod table %s: %v", tbl.Name, err)
			}
			devTbl, ok := devSchema.Tables[tbl.Name]
			if !ok {
				continue
			}
			dh, err := content.HashTable(ctx, devDB, "oracle", devTbl, ignoreColumn, 0)
			if err != nil {
				t.Fatalf("hash dev table %s: %v", devTbl.Name, err)
			}
			_ = content.DiffHashes(tbl.Name, ph, dh)
		}
	})

	t.Run("GeneratePack", func(t *testing.T) {
		prodSchema, err := schema.LoadSchema(ctx, prodDB, "oracle", oracleService, ignoreTables)
		if err != nil {
			t.Fatalf("load prod schema: %v", err)
		}
		devSchema, err := schema.LoadSchema(ctx, devDB, "oracle", oracleService, ignoreTables)
		if err != nil {
			t.Fatalf("load dev schema: %v", err)
		}
		schemaDiff := schema.DiffSchemas(prodSchema, devSchema)

		var tableDiffs []content.TableDiff
		for _, tbl := range prodSchema.Tables {
			devTbl, ok := devSchema.Tables[tbl.Name]
			if !ok {
				continue
			}
			ph, err := content.HashTable(ctx, prodDB, "oracle", tbl, ignoreColumn, 0)
			if err != nil {
				t.Fatalf("hash prod %s: %v", tbl.Name, err)
			}
			dh, err := content.HashTable(ctx, devDB, "oracle", devTbl, ignoreColumn, 0)
			if err != nil {
				t.Fatalf("hash dev %s: %v", devTbl.Name, err)
			}
			tableDiffs = append(tableDiffs, content.DiffHashes(tbl.Name, ph, dh))
		}
		dataDiff := content.DataDiff{Tables: tableDiffs}

		_, err = content.GeneratePack(ctx, devDB, prodDB, "oracle", oracleService, dataDiff, schemaDiff, devSchema, prodSchema, outputDir, ignoreColumn, nil, nil)
		if err != nil {
			t.Fatalf("generate pack: %v", err)
		}

		packPath := filepath.Join(outputDir, "migration_pack.sql")
		if _, statErr := os.Stat(packPath); os.IsNotExist(statErr) {
			t.Fatal("migration_pack.sql not created")
		}
	})
}
