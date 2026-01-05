package schema_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestIntrospect_MySQL_DefaultValues tests DEFAULT value introspection for MySQL
func TestIntrospect_MySQL_DefaultValues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start MySQL container
	container, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("test_db"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start MySQL container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	// Get connection details
	port, err := container.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatalf("failed to get port: %v", err)
	}

	connStr := "testuser:testpass@tcp(localhost:" + port.Port() + ")/test_db?parseTime=true"
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Wait for database to be ready
	time.Sleep(2 * time.Second)
	for i := 0; i < 5; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}

	// Create test table with various DEFAULT values
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_defaults (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			status VARCHAR(20) DEFAULT 'active',
			count INT DEFAULT 0,
			score DECIMAL(10,2) DEFAULT 0.00,
			is_enabled TINYINT(1) DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			nullable_with_default VARCHAR(50) DEFAULT 'default_value',
			no_default VARCHAR(50)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Load schema
	s, err := schema.LoadSchema(ctx, db, "mysql", "test_db", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	// Verify table exists
	table, ok := s.Tables["test_defaults"]
	if !ok {
		t.Fatal("test_defaults table not found in schema")
	}

	// Test cases for each column
	// Note: MySQL returns DEFAULT values from INFORMATION_SCHEMA without quotes for strings
	tests := []struct {
		columnName       string
		expectDefault    bool
		expectedDefault  string
		dataTypeContains string
	}{
		{"status", true, "active", "varchar"},
		{"count", true, "0", "int"},
		{"score", true, "0.00", "decimal"},
		{"is_enabled", true, "1", "tinyint"},
		{"created_at", true, "CURRENT_TIMESTAMP", "timestamp"},
		{"nullable_with_default", true, "default_value", "varchar"},
		{"no_default", false, "", "varchar"},
	}

	for _, tt := range tests {
		t.Run(tt.columnName, func(t *testing.T) {
			col, ok := table.Columns[tt.columnName]
			if !ok {
				t.Fatalf("column %s not found", tt.columnName)
			}

			// Check data type includes size specification
			if !strings.Contains(col.DataType, tt.dataTypeContains) {
				t.Errorf("DataType = %q, should contain %q", col.DataType, tt.dataTypeContains)
			}

			// Check DEFAULT value
			if tt.expectDefault {
				if col.DefaultValue == nil {
					t.Errorf("expected DefaultValue to be set, got nil")
				} else if *col.DefaultValue != tt.expectedDefault {
					t.Errorf("DefaultValue = %q, want %q", *col.DefaultValue, tt.expectedDefault)
				}
			} else {
				if col.DefaultValue != nil {
					t.Errorf("expected DefaultValue to be nil, got %q", *col.DefaultValue)
				}
			}
		})
	}

	// Verify column_type is captured (not just data_type)
	statusCol := table.Columns["status"]
	if !strings.Contains(statusCol.DataType, "20") {
		t.Errorf("status column should include size specification, got %q", statusCol.DataType)
	}
}

// TestIntrospect_PostgreSQL_DefaultValues tests DEFAULT value introspection for PostgreSQL
func TestIntrospect_PostgreSQL_DefaultValues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start PostgreSQL container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	// Get connection details
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get port: %v", err)
	}

	connStr := "postgres://testuser:testpass@localhost:" + port.Port() + "/test_db?sslmode=disable"
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Wait for database to be ready
	time.Sleep(2 * time.Second)
	for i := 0; i < 5; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}

	// Create test table with various DEFAULT values
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_defaults (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			status VARCHAR(20) DEFAULT 'active',
			count INTEGER DEFAULT 0,
			score NUMERIC(10,2) DEFAULT 0.00,
			is_enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			nullable_with_default TEXT DEFAULT 'default_value',
			no_default TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Load schema
	s, err := schema.LoadSchema(ctx, db, "postgres", "test_db", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	// Verify table exists
	table, ok := s.Tables["test_defaults"]
	if !ok {
		t.Fatal("test_defaults table not found in schema")
	}

	// Test cases for each column
	tests := []struct {
		columnName    string
		expectDefault bool
		// PostgreSQL may return defaults with ::type casts
	}{
		{"status", true},
		{"count", true},
		{"score", true},
		{"is_enabled", true},
		{"created_at", true},
		{"nullable_with_default", true},
		{"no_default", false},
	}

	for _, tt := range tests {
		t.Run(tt.columnName, func(t *testing.T) {
			col, ok := table.Columns[tt.columnName]
			if !ok {
				t.Fatalf("column %s not found", tt.columnName)
			}

			// Check DEFAULT value presence
			if tt.expectDefault {
				if col.DefaultValue == nil {
					t.Errorf("expected DefaultValue to be set, got nil")
				}
			} else {
				if col.DefaultValue != nil {
					t.Errorf("expected DefaultValue to be nil, got %q", *col.DefaultValue)
				}
			}
		})
	}
}

// TestIntrospect_MySQL_ColumnTypes tests MySQL column_type introspection
func TestIntrospect_MySQL_ColumnTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	container, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("test_db"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start MySQL container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	port, err := container.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatalf("failed to get port: %v", err)
	}

	connStr := "testuser:testpass@tcp(localhost:" + port.Port() + ")/test_db"
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	time.Sleep(2 * time.Second)
	for i := 0; i < 5; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}

	// Create table with specific column types
	_, err = db.ExecContext(ctx, `
		CREATE TABLE type_test (
			id INT PRIMARY KEY,
			email VARCHAR(100),
			username VARCHAR(255),
			age BIGINT,
			active TINYINT(1),
			description TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Load schema
	s, err := schema.LoadSchema(ctx, db, "mysql", "test_db", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	table := s.Tables["type_test"]

	// Verify column_type includes size specifications
	tests := []struct {
		columnName   string
		expectedType string
	}{
		{"email", "varchar(100)"},
		{"username", "varchar(255)"},
		{"age", "bigint"},
		{"active", "tinyint(1)"},
		{"description", "text"},
	}

		for _, tt := range tests {
		t.Run(tt.columnName, func(t *testing.T) {
			col := table.Columns[tt.columnName]
			if col.DataType != tt.expectedType {
				t.Errorf("DataType = %q, want %q", col.DataType, tt.expectedType)
			}
		})
	}
}

// TestIntrospect_PostgreSQL_CompositeForeignKey tests composite foreign keys
// This exercises the containsString function used to avoid duplicate columns
func TestIntrospect_PostgreSQL_CompositeForeignKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start PostgreSQL container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	// Get connection details
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get port: %v", err)
	}

	connStr := "postgres://testuser:testpass@localhost:" + port.Port() + "/test_db?sslmode=disable"
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Wait for database to be ready
	time.Sleep(2 * time.Second)
	for i := 0; i < 5; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		if i < 4 {
			time.Sleep(time.Second)
		}
	}

	// Create parent table with composite primary key
	_, err = db.ExecContext(ctx, `
		CREATE TABLE parent (
			tenant_id INTEGER,
			user_id INTEGER,
			name TEXT,
			PRIMARY KEY (tenant_id, user_id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create parent table: %v", err)
	}

	// Create child table with composite foreign key
	// This will exercise containsString when processing the FK columns
	_, err = db.ExecContext(ctx, `
		CREATE TABLE child (
			id SERIAL PRIMARY KEY,
			tenant_id INTEGER,
			user_id INTEGER,
			data TEXT,
			FOREIGN KEY (tenant_id, user_id) REFERENCES parent(tenant_id, user_id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create child table: %v", err)
	}

	// Load schema - this should exercise containsString
	s, err := schema.LoadSchema(ctx, db, "postgres", "test_db", nil)
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	// Verify foreign key was loaded correctly
	childTable, ok := s.Tables["child"]
	if !ok {
		t.Fatal("child table not found")
	}

	if len(childTable.ForeignKeys) == 0 {
		t.Fatal("expected foreign key to be loaded")
	}

	// Check that composite FK columns are correct
	var fk schema.ForeignKey
	var found bool
	fk, found = childTable.ForeignKeys["child_tenant_id_user_id_fkey"]
	if !found {
		// Try to find any FK
		for _, f := range childTable.ForeignKeys {
			fk = f
			found = true
			break
		}
	}

	if found {
		if len(fk.Columns) != 2 {
			t.Errorf("expected 2 FK columns, got %d", len(fk.Columns))
		}
		if len(fk.ReferencedColumns) != 2 {
			t.Errorf("expected 2 referenced columns, got %d", len(fk.ReferencedColumns))
		}
		// Verify no duplicates (containsString should prevent this)
		colSet := make(map[string]bool)
		for _, col := range fk.Columns {
			if colSet[col] {
				t.Errorf("duplicate column in FK: %s", col)
			}
			colSet[col] = true
		}
	}
}
