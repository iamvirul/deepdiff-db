package main

import (
	"github.com/iamvirul/deepdiff-db/internal/drivers"

	"context"
	"testing"

	"github.com/iamvirul/deepdiff-db/pkg/config"
)

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.DBConfig
		wantError bool
		validate  func(*testing.T, string, string)
	}{
		{
			name: "mysql",
			cfg: config.DBConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     3306,
				User:     "root",
				Password: "password",
				Database: "testdb",
			},
			wantError: false,
			validate: func(t *testing.T, driver, dsn string) {
				if driver != "mysql" {
					t.Errorf("expected driver 'mysql', got %q", driver)
				}
				if dsn == "" {
					t.Error("DSN should not be empty")
				}
				// MySQL DSN should contain connection details
				if !contains(dsn, "testdb") {
					t.Error("DSN should contain database name")
				}
			},
		},
		{
			name: "postgres",
			cfg: config.DBConfig{
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				Database: "testdb",
			},
			wantError: false,
			validate: func(t *testing.T, driver, dsn string) {
				if driver != "pgx" {
					t.Errorf("expected driver 'pgx', got %q", driver)
				}
				if dsn == "" {
					t.Error("DSN should not be empty")
				}
			},
		},
		{
			name: "postgresql alias",
			cfg: config.DBConfig{
				Driver:   "postgresql",
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				Database: "testdb",
			},
			wantError: false,
			validate: func(t *testing.T, driver, dsn string) {
				if driver != "pgx" {
					t.Errorf("expected driver 'pgx', got %q", driver)
				}
			},
		},
		{
			name: "sqlite",
			cfg: config.DBConfig{
				Driver:   "sqlite",
				Database: "/tmp/test.db",
			},
			wantError: false,
			validate: func(t *testing.T, driver, dsn string) {
				if driver != "sqlite" {
					t.Errorf("expected driver 'sqlite', got %q", driver)
				}
				if dsn != "/tmp/test.db" {
					t.Errorf("expected DSN '/tmp/test.db', got %q", dsn)
				}
			},
		},
		{
			name: "sqlite empty database",
			cfg: config.DBConfig{
				Driver:   "sqlite",
				Database: "",
			},
			wantError: true,
		},
		{
			name: "unsupported driver",
			cfg: config.DBConfig{
				Driver:   "db2",
				Host:     "localhost",
				Port:     50000,
				Database: "testdb",
			},
			wantError: true,
		},
		{
			name: "mysql with special characters in password",
			cfg: config.DBConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     3306,
				User:     "user",
				Password: "p@ssw0rd!@#",
				Database: "testdb",
			},
			wantError: false,
			validate: func(t *testing.T, driver, dsn string) {
				if driver != "mysql" {
					t.Errorf("expected driver 'mysql', got %q", driver)
				}
				// DSN should be properly escaped
				if dsn == "" {
					t.Error("DSN should not be empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, dsn, err := drivers.BuildDSN(tt.cfg)
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, driver, dsn)
			}
		})
	}
}

func TestOpen_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	cfg := config.DBConfig{
		Driver:   "invalid",
		Database: "test",
		Port:     3306,
	}

	_, err := drivers.Open(ctx, cfg)
	if err == nil {
		t.Error("expected error for invalid driver")
	}
}

// Note: We can't easily test successful Open() without actual database connections
// Those would be integration tests that require running database servers

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBuildDSN_MSSQL(t *testing.T) {
	tests := []struct {
		name       string
		cfg        config.DBConfig
		wantDriver string
		wantDSNHas []string
		wantDSNNot []string
	}{
		{
			name: "mssql with port",
			cfg: config.DBConfig{
				Driver:   "mssql",
				Host:     "sqlserver.example.com",
				Port:     1433,
				User:     "sa",
				Password: "StrongP@ss1word!",
				Database: "mydb",
			},
			wantDriver: "sqlserver",
			wantDSNHas: []string{
				"sqlserver://",
				"sqlserver.example.com:1433",
				"database=mydb",
			},
		},
		{
			name: "mssql without port defaults to no port in DSN",
			cfg: config.DBConfig{
				Driver:   "mssql",
				Host:     "sqlserver.example.com",
				Port:     0,
				User:     "sa",
				Password: "pass",
				Database: "mydb",
			},
			wantDriver: "sqlserver",
			wantDSNHas: []string{
				"sqlserver://",
				"sqlserver.example.com",
				"database=mydb",
			},
			// Port 0 means omit port — no colon+number after host
			wantDSNNot: []string{":0"},
		},
		{
			name: "mssql URL-encodes credentials with special chars",
			cfg: config.DBConfig{
				Driver:   "mssql",
				Host:     "localhost",
				Port:     1433,
				User:     "user@domain",
				Password: "p@ss w0rd+1",
				Database: "test db",
			},
			wantDriver: "sqlserver",
			wantDSNHas: []string{
				"sqlserver://",
				"localhost:1433",
			},
			// Raw special chars must not appear unencoded in the DSN
			wantDSNNot: []string{"p@ss w0rd+1", " "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, dsn, err := drivers.BuildDSN(tt.cfg)
			if err != nil {
				t.Fatalf("BuildDSN returned unexpected error: %v", err)
			}
			if driver != tt.wantDriver {
				t.Errorf("driver = %q, want %q", driver, tt.wantDriver)
			}
			for _, want := range tt.wantDSNHas {
				if !containsMiddle(dsn, want) {
					t.Errorf("DSN %q does not contain %q", dsn, want)
				}
			}
			for _, notWant := range tt.wantDSNNot {
				if containsMiddle(dsn, notWant) {
					t.Errorf("DSN %q should not contain %q", dsn, notWant)
				}
			}
		})
	}
}

func TestBuildDSN_Oracle(t *testing.T) {
	tests := []struct {
		name       string
		cfg        config.DBConfig
		wantDriver string
		wantDSNHas []string
		wantDSNNot []string
	}{
		{
			name: "oracle with port",
			cfg: config.DBConfig{
				Driver:   "oracle",
				Host:     "oracle.example.com",
				Port:     1521,
				User:     "appuser",
				Password: "Secret1234",
				Database: "XEPDB1",
			},
			wantDriver: "oracle",
			wantDSNHas: []string{
				"oracle://",
				"oracle.example.com:1521",
				"/XEPDB1",
			},
		},
		{
			name: "oracle without port defaults to 1521",
			cfg: config.DBConfig{
				Driver:   "oracle",
				Host:     "oracle.example.com",
				Port:     0,
				User:     "appuser",
				Password: "pass",
				Database: "ORCL",
			},
			wantDriver: "oracle",
			wantDSNHas: []string{
				"oracle://",
				"oracle.example.com:1521",
				"/ORCL",
			},
			// Port 0 must not appear literally in the DSN
			wantDSNNot: []string{":0"},
		},
		{
			name: "oracle URL-encodes credentials with special chars",
			cfg: config.DBConfig{
				Driver:   "oracle",
				Host:     "localhost",
				Port:     1521,
				User:     "user@domain",
				Password: "p@ss w0rd+1",
				Database: "XEPDB1",
			},
			wantDriver: "oracle",
			wantDSNHas: []string{
				"oracle://",
				"localhost:1521",
			},
			// Raw special chars must not appear unencoded
			wantDSNNot: []string{"p@ss w0rd+1", " "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, dsn, err := drivers.BuildDSN(tt.cfg)
			if err != nil {
				t.Fatalf("BuildDSN returned unexpected error: %v", err)
			}
			if driver != tt.wantDriver {
				t.Errorf("driver = %q, want %q", driver, tt.wantDriver)
			}
			for _, want := range tt.wantDSNHas {
				if !containsMiddle(dsn, want) {
					t.Errorf("DSN %q does not contain %q", dsn, want)
				}
			}
			for _, notWant := range tt.wantDSNNot {
				if containsMiddle(dsn, notWant) {
					t.Errorf("DSN %q should not contain %q", dsn, notWant)
				}
			}
		})
	}
}
