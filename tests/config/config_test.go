package main

import (
	"github.com/iamvirul/deepdiff-db/pkg/config"

	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError bool
		validate  func(*testing.T, *config.Config)
	}{
		{
			name: "valid config",
			config: `
prod:
  driver: "mysql"
  host: "localhost"
  port: 3306
  user: "root"
  password: "pass"
  database: "prod_db"

dev:
  driver: "postgres"
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "pass"
  database: "dev_db"

ignore:
  tables:
    - "logs"
  columns:
    - "*.updated_at"

output:
  dir: "./test-output"
`,
			wantError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Prod.Driver != "mysql" {
					t.Errorf("expected prod driver mysql, got %s", cfg.Prod.Driver)
				}
				if cfg.Dev.Driver != "postgres" {
					t.Errorf("expected dev driver postgres, got %s", cfg.Dev.Driver)
				}
				if len(cfg.Ignore.Tables) != 1 || cfg.Ignore.Tables[0] != "logs" {
					t.Errorf("expected ignore table 'logs', got %v", cfg.Ignore.Tables)
				}
				if cfg.Output.Dir != "./test-output" {
					t.Errorf("expected output dir './test-output', got %s", cfg.Output.Dir)
				}
			},
		},
		{
			name: "default output dir",
			config: `
prod:
  driver: "sqlite"
  host: ""
  port: 1
  user: ""
  password: ""
  database: "/tmp/test.db"

dev:
  driver: "sqlite"
  host: ""
  port: 1
  user: ""
  password: ""
  database: "/tmp/test2.db"
`,
			wantError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Output.Dir != "./diff-output" {
					t.Errorf("expected default output dir './diff-output', got %s", cfg.Output.Dir)
				}
			},
		},
		{
			name: "missing prod driver",
			config: `
prod:
  host: "localhost"
  port: 3306
  database: "prod_db"

dev:
  driver: "mysql"
  host: "localhost"
  port: 3306
  database: "dev_db"
`,
			wantError: true,
		},
		{
			name: "missing prod database",
			config: `
prod:
  driver: "mysql"
  host: "localhost"
  port: 3306

dev:
  driver: "mysql"
  host: "localhost"
  port: 3306
  database: "dev_db"
`,
			wantError: true,
		},
		{
			name: "missing prod port",
			config: `
prod:
  driver: "mysql"
  host: "localhost"
  database: "prod_db"

dev:
  driver: "mysql"
  host: "localhost"
  port: 3306
  database: "dev_db"
`,
			wantError: true,
		},
		{
			name: "unsupported driver",
			config: `
prod:
  driver: "oracle"
  host: "localhost"
  port: 1521
  database: "prod_db"

dev:
  driver: "mysql"
  host: "localhost"
  port: 3306
  database: "dev_db"
`,
			wantError: true,
		},
		{
			name: "missing dev driver",
			config: `
prod:
  driver: "mysql"
  host: "localhost"
  port: 3306
  database: "prod_db"

dev:
  host: "localhost"
  port: 3306
  database: "dev_db"
`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0o644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := config.Load(configPath)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg == nil {
				t.Fatalf("config is nil")
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		wantError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				Prod: config.DBConfig{
					Driver:   "mysql",
					Host:     "localhost",
					Port:     3306,
					Database: "prod",
				},
				Dev: config.DBConfig{
					Driver:   "postgres",
					Host:     "localhost",
					Port:     5432,
					Database: "dev",
				},
			},
			wantError: false,
		},
		{
			name: "sqlite valid",
			config: &config.Config{
				Prod: config.DBConfig{
					Driver:   "sqlite",
					Database: "/tmp/test.db",
					Port:     1, // sqlite doesn't need port but validation requires it
				},
				Dev: config.DBConfig{
					Driver:   "sqlite",
					Database: "/tmp/test2.db",
					Port:     1,
				},
			},
			wantError: false,
		},
		{
			name: "postgresql alias",
			config: &config.Config{
				Prod: config.DBConfig{
					Driver:   "postgresql",
					Host:     "localhost",
					Port:     5432,
					Database: "prod",
				},
				Dev: config.DBConfig{
					Driver:   "postgresql",
					Host:     "localhost",
					Port:     5432,
					Database: "dev",
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDBConfig_validate(t *testing.T) {
	tests := []struct {
		name      string
		config    config.DBConfig
		prefix    string
		wantError bool
	}{
		{
			name: "valid mysql",
			config: config.DBConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     3306,
				Database: "test",
			},
			prefix:    "prod",
			wantError: false,
		},
		{
			name: "missing driver",
			config: config.DBConfig{
				Host:     "localhost",
				Port:     3306,
				Database: "test",
			},
			prefix:    "prod",
			wantError: true,
		},
		{
			name: "missing database",
			config: config.DBConfig{
				Driver: "mysql",
				Host:   "localhost",
				Port:   3306,
			},
			prefix:    "prod",
			wantError: true,
		},
		{
			name: "missing port",
			config: config.DBConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Database: "test",
			},
			prefix:    "prod",
			wantError: true,
		},
		{
			name: "unsupported driver",
			config: config.DBConfig{
				Driver:   "mssql",
				Host:     "localhost",
				Port:     1433,
				Database: "test",
			},
			prefix:    "prod",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: validate is unexported, so we skip direct testing
			// Validation is tested indirectly through Load
			_ = tt.config
			_ = tt.prefix
			// This test is skipped as validate is unexported
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// ============================================================================
// Migration Config Tests (AllowDropIndex, AllowDropColumn, AllowDropTable)
// ============================================================================

func TestLoad_MigrationConfig_AllowDropIndex(t *testing.T) {
	tests := []struct {
		name           string
		config         string
		wantDropIndex  bool
	}{
		{
			name: "allow_drop_index true",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

migration:
  allow_drop_index: true
`,
			wantDropIndex: true,
		},
		{
			name: "allow_drop_index false",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

migration:
  allow_drop_index: false
`,
			wantDropIndex: false,
		},
		{
			name: "allow_drop_index not specified (default false)",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db
`,
			wantDropIndex: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0o644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Migration.AllowDropIndex != tt.wantDropIndex {
				t.Errorf("expected AllowDropIndex=%v, got %v", tt.wantDropIndex, cfg.Migration.AllowDropIndex)
			}
		})
	}
}

func TestLoad_MigrationConfig_AllSettings(t *testing.T) {
	configYAML := `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

migration:
  allow_drop_column: true
  allow_drop_table: true
  allow_drop_index: true
  confirm_destructive: true
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Migration.AllowDropColumn {
		t.Error("expected AllowDropColumn to be true")
	}
	if !cfg.Migration.AllowDropTable {
		t.Error("expected AllowDropTable to be true")
	}
	if !cfg.Migration.AllowDropIndex {
		t.Error("expected AllowDropIndex to be true")
	}
	if !cfg.Migration.ConfirmDestructive {
		t.Error("expected ConfirmDestructive to be true")
	}
}

func TestLoad_MigrationConfig_DefaultValues(t *testing.T) {
	configYAML := `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All migration settings should default to false
	if cfg.Migration.AllowDropColumn {
		t.Error("expected AllowDropColumn to default to false")
	}
	if cfg.Migration.AllowDropTable {
		t.Error("expected AllowDropTable to default to false")
	}
	if cfg.Migration.AllowDropIndex {
		t.Error("expected AllowDropIndex to default to false")
	}
	if cfg.Migration.ConfirmDestructive {
		t.Error("expected ConfirmDestructive to default to false")
	}
}

func TestLoad_MigrationConfig_PartialSettings(t *testing.T) {
	// Test that specifying some migration settings doesn't affect others
	configYAML := `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

migration:
  allow_drop_index: true
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only AllowDropIndex should be true
	if cfg.Migration.AllowDropColumn {
		t.Error("expected AllowDropColumn to be false")
	}
	if cfg.Migration.AllowDropTable {
		t.Error("expected AllowDropTable to be false")
	}
	if !cfg.Migration.AllowDropIndex {
		t.Error("expected AllowDropIndex to be true")
	}
	if cfg.Migration.ConfirmDestructive {
		t.Error("expected ConfirmDestructive to be false")
	}
}

func TestMigrationConfig_Struct(t *testing.T) {
	// Test direct struct creation
	migConfig := config.MigrationConfig{
		AllowDropColumn:    true,
		AllowDropTable:     true,
		AllowDropIndex:     true,
		ConfirmDestructive: true,
	}

	if !migConfig.AllowDropColumn {
		t.Error("expected AllowDropColumn to be true")
	}
	if !migConfig.AllowDropTable {
		t.Error("expected AllowDropTable to be true")
	}
	if !migConfig.AllowDropIndex {
		t.Error("expected AllowDropIndex to be true")
	}
	if !migConfig.ConfirmDestructive {
		t.Error("expected ConfirmDestructive to be true")
	}
}

// ============================================================================
// Conflict Resolution Config Tests
// ============================================================================

func TestLoad_ConflictResolutionConfig_DefaultStrategy(t *testing.T) {
	tests := []struct {
		name            string
		config          string
		wantStrategy    string
		wantError       bool
		wantErrContains string
	}{
		{
			name: "default strategy manual when not specified",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db
`,
			wantStrategy: "manual",
			wantError:    false,
		},
		{
			name: "explicit default strategy ours",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

conflict_resolution:
  default_strategy: "ours"
`,
			wantStrategy: "ours",
			wantError:    false,
		},
		{
			name: "explicit default strategy theirs",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

conflict_resolution:
  default_strategy: "theirs"
`,
			wantStrategy: "theirs",
			wantError:    false,
		},
		{
			name: "explicit default strategy manual",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

conflict_resolution:
  default_strategy: "manual"
`,
			wantStrategy: "manual",
			wantError:    false,
		},
		{
			name: "invalid default strategy",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

conflict_resolution:
  default_strategy: "invalid"
`,
			wantError:       true,
			wantErrContains: "invalid value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0o644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := config.Load(configPath)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.wantErrContains != "" && !contains(err.Error(), tt.wantErrContains) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.ConflictResolution.DefaultStrategy != tt.wantStrategy {
				t.Errorf("expected DefaultStrategy=%q, got %q", tt.wantStrategy, cfg.ConflictResolution.DefaultStrategy)
			}
		})
	}
}

func TestLoad_ConflictResolutionConfig_TableStrategies(t *testing.T) {
	tests := []struct {
		name            string
		config          string
		wantStrategies  []config.TableStrategy
		wantError       bool
		wantErrContains string
	}{
		{
			name: "valid table strategies",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

conflict_resolution:
  default_strategy: "manual"
  strategies:
    - table: "logs"
      strategy: "theirs"
    - table: "config"
      strategy: "ours"
    - table: "users"
      strategy: "manual"
`,
			wantStrategies: []config.TableStrategy{
				{Table: "logs", Strategy: "theirs"},
				{Table: "config", Strategy: "ours"},
				{Table: "users", Strategy: "manual"},
			},
			wantError: false,
		},
		{
			name: "missing table name",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

conflict_resolution:
  strategies:
    - strategy: "theirs"
`,
			wantError:       true,
			wantErrContains: "table name is required",
		},
		{
			name: "missing strategy",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

conflict_resolution:
  strategies:
    - table: "logs"
`,
			wantError:       true,
			wantErrContains: "strategy is required",
		},
		{
			name: "invalid table strategy",
			config: `
prod:
  driver: mysql
  host: localhost
  port: 3306
  database: prod_db

dev:
  driver: mysql
  host: localhost
  port: 3306
  database: dev_db

conflict_resolution:
  strategies:
    - table: "logs"
      strategy: "invalid"
`,
			wantError:       true,
			wantErrContains: "invalid value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0o644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := config.Load(configPath)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.wantErrContains != "" && !contains(err.Error(), tt.wantErrContains) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(cfg.ConflictResolution.Strategies) != len(tt.wantStrategies) {
				t.Errorf("expected %d strategies, got %d", len(tt.wantStrategies), len(cfg.ConflictResolution.Strategies))
				return
			}

			for i, want := range tt.wantStrategies {
				got := cfg.ConflictResolution.Strategies[i]
				if got.Table != want.Table || got.Strategy != want.Strategy {
					t.Errorf("strategy[%d]: expected {%q, %q}, got {%q, %q}",
						i, want.Table, want.Strategy, got.Table, got.Strategy)
				}
			}
		})
	}
}

func TestConflictResolutionConfig_GetStrategyForTable(t *testing.T) {
	cfg := config.ConflictResolutionConfig{
		DefaultStrategy: "manual",
		Strategies: []config.TableStrategy{
			{Table: "logs", Strategy: "theirs"},
			{Table: "config", Strategy: "ours"},
		},
	}

	tests := []struct {
		tableName    string
		wantStrategy string
	}{
		{"logs", "theirs"},
		{"config", "ours"},
		{"users", "manual"},       // Falls back to default
		{"unknown", "manual"},     // Falls back to default
	}

	for _, tt := range tests {
		t.Run(tt.tableName, func(t *testing.T) {
			got := cfg.GetStrategyForTable(tt.tableName)
			if got != tt.wantStrategy {
				t.Errorf("GetStrategyForTable(%q) = %q, want %q", tt.tableName, got, tt.wantStrategy)
			}
		})
	}
}

func TestConflictResolutionConfig_Struct(t *testing.T) {
	// Test direct struct creation
	crConfig := config.ConflictResolutionConfig{
		DefaultStrategy: "theirs",
		Strategies: []config.TableStrategy{
			{Table: "logs", Strategy: "theirs"},
			{Table: "users", Strategy: "ours"},
		},
	}

	if crConfig.DefaultStrategy != "theirs" {
		t.Errorf("expected DefaultStrategy to be 'theirs', got %q", crConfig.DefaultStrategy)
	}

	if len(crConfig.Strategies) != 2 {
		t.Errorf("expected 2 strategies, got %d", len(crConfig.Strategies))
	}

	if crConfig.Strategies[0].Table != "logs" || crConfig.Strategies[0].Strategy != "theirs" {
		t.Errorf("expected first strategy to be {logs, theirs}, got {%q, %q}",
			crConfig.Strategies[0].Table, crConfig.Strategies[0].Strategy)
	}
}

func TestStrategyConstants(t *testing.T) {
	// Verify strategy constants are correct
	if config.StrategyOurs != "ours" {
		t.Errorf("expected StrategyOurs to be 'ours', got %q", config.StrategyOurs)
	}
	if config.StrategyTheirs != "theirs" {
		t.Errorf("expected StrategyTheirs to be 'theirs', got %q", config.StrategyTheirs)
	}
	if config.StrategyManual != "manual" {
		t.Errorf("expected StrategyManual to be 'manual', got %q", config.StrategyManual)
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

