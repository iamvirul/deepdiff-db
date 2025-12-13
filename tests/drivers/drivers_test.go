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
				Driver:   "oracle",
				Host:     "localhost",
				Port:     1521,
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

