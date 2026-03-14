package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Strategy constants for conflict resolution.
const (
	StrategyOurs   = "ours"   // Keep production version
	StrategyTheirs = "theirs" // Use development version
	StrategyManual = "manual" // Require manual review
)

// Config holds the full DeepDiff DB configuration.
type Config struct {
	Prod               DBConfig                 `yaml:"prod"`
	Dev                DBConfig                 `yaml:"dev"`
	Ignore             IgnoreConfig             `yaml:"ignore"`
	Output             OutputConfig             `yaml:"output"`
	Migration          MigrationConfig          `yaml:"migration"`
	ConflictResolution ConflictResolutionConfig `yaml:"conflict_resolution"`
	Performance        PerformanceConfig        `yaml:"performance"`
}

// PerformanceConfig controls resource usage during large-dataset operations.
type PerformanceConfig struct {
	// HashBatchSize is the number of rows fetched per keyset-paginated query during
	// table hashing. Set to 0 to disable batching (load all rows in one query).
	// Default: 10000.
	HashBatchSize int `yaml:"hash_batch_size"`

	// MaxParallelTables is the maximum number of tables hashed concurrently.
	// Default: 1 (sequential, safe for all environments).
	MaxParallelTables int `yaml:"max_parallel_tables"`
}

// DBConfig represents connection details for a single database.
type DBConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// IgnoreConfig defines tables/columns to skip.
type IgnoreConfig struct {
	Tables  []string `yaml:"tables"`
	Columns []string `yaml:"columns"`
}

// OutputConfig defines the output directory for reports and packs.
type OutputConfig struct {
	Dir string `yaml:"dir"`
}

// MigrationConfig defines safety and behavior settings for schema migrations.
type MigrationConfig struct {
	// AllowDropColumn controls whether DROP COLUMN statements are uncommented.
	// When false (default), DROP COLUMN statements are commented out for safety.
	AllowDropColumn bool `yaml:"allow_drop_column"`

	// AllowDropTable controls whether DROP TABLE statements are uncommented.
	// When false (default), DROP TABLE statements are commented out for safety.
	AllowDropTable bool `yaml:"allow_drop_table"`

	// AllowDropIndex controls whether DROP INDEX statements are uncommented.
	// When false (default), DROP INDEX statements are commented out for safety.
	AllowDropIndex bool `yaml:"allow_drop_index"`

	// AllowDropForeignKey controls whether DROP FOREIGN KEY statements are uncommented.
	// When false (default), DROP FOREIGN KEY statements are commented out for safety.
	AllowDropForeignKey bool `yaml:"allow_drop_foreign_key"`

	// AllowModifyPrimaryKey controls whether primary key modification statements are uncommented.
	// When false (default), PRIMARY KEY modification statements are commented out for safety.
	AllowModifyPrimaryKey bool `yaml:"allow_modify_primary_key"`

	// ConfirmDestructive requires manual confirmation for destructive operations.
	// When true, destructive operations include additional warnings.
	ConfirmDestructive bool `yaml:"confirm_destructive"`
}

// ConflictResolutionConfig defines how conflicts between prod and dev databases are resolved.
type ConflictResolutionConfig struct {
	// DefaultStrategy is the global default strategy for conflict resolution.
	// Valid values: "ours" (keep prod), "theirs" (use dev), "manual" (require review).
	// Defaults to "manual" if not specified.
	DefaultStrategy string `yaml:"default_strategy"`

	// Strategies defines per-table strategy overrides.
	// These take precedence over the DefaultStrategy.
	Strategies []TableStrategy `yaml:"strategies"`
}

// TableStrategy defines a conflict resolution strategy for a specific table.
type TableStrategy struct {
	// Table is the name of the table this strategy applies to.
	Table string `yaml:"table"`

	// Strategy is the resolution strategy for this table.
	// Valid values: "ours" (keep prod), "theirs" (use dev), "manual" (require review).
	Strategy string `yaml:"strategy"`
}

// Load reads configuration from the YAML file at path, unmarshals it into a Config,
// validates the resulting configuration, and ensures Output.Dir defaults to "./diff-output"
// when not specified.
//
// If the file cannot be read the returned error is wrapped with the prefix "read config",
// and if the YAML cannot be parsed the error is wrapped with the prefix "parse config".
// Validation errors from Config.Validate are returned as-is.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.Output.Dir == "" {
		cfg.Output.Dir = "./diff-output"
	}

	// Default conflict resolution strategy to "manual" if not specified
	if cfg.ConflictResolution.DefaultStrategy == "" {
		cfg.ConflictResolution.DefaultStrategy = StrategyManual
	}

	// Apply performance defaults
	if cfg.Performance.HashBatchSize == 0 {
		cfg.Performance.HashBatchSize = 10000
	}
	if cfg.Performance.MaxParallelTables == 0 {
		cfg.Performance.MaxParallelTables = 1
	}

	return &cfg, nil
}

// Validate performs minimal sanity checks.
func (c *Config) Validate() error {
	if err := c.Prod.validate("prod"); err != nil {
		return err
	}
	if err := c.Dev.validate("dev"); err != nil {
		return err
	}
	if err := c.ConflictResolution.validate(); err != nil {
		return err
	}
	if err := c.Performance.validate(); err != nil {
		return err
	}
	return nil
}

func (p *PerformanceConfig) validate() error {
	if p.HashBatchSize < 0 {
		return fmt.Errorf("performance.hash_batch_size must be >= 0")
	}
	if p.MaxParallelTables < 0 {
		return fmt.Errorf("performance.max_parallel_tables must be >= 0")
	}
	return nil
}

func (c *DBConfig) validate(prefix string) error {
	if c.Driver == "" {
		return fmt.Errorf("%s.driver is required", prefix)
	}
	if c.Database == "" {
		return fmt.Errorf("%s.database is required", prefix)
	}
	if c.Port == 0 && c.Driver != "sqlite" && c.Driver != "mssql" {
		return fmt.Errorf("%s.port is required", prefix)
	}
	switch c.Driver {
	case "mysql", "postgres", "postgresql", "sqlite", "mssql":
	default:
		return fmt.Errorf("%s.driver unsupported: %s", prefix, c.Driver)
	}
	return nil
}

// ErrInvalidConfig indicates missing or invalid fields.
var ErrInvalidConfig = errors.New("invalid configuration")

// isValidStrategy checks if the given strategy is valid.
func isValidStrategy(strategy string) bool {
	switch strategy {
	case StrategyOurs, StrategyTheirs, StrategyManual, "":
		return true
	default:
		return false
	}
}

// validate checks that the conflict resolution configuration is valid.
func (c *ConflictResolutionConfig) validate() error {
	// Validate default strategy if specified
	if c.DefaultStrategy != "" && !isValidStrategy(c.DefaultStrategy) {
		return fmt.Errorf("conflict_resolution.default_strategy: invalid value %q (must be %q, %q, or %q)",
			c.DefaultStrategy, StrategyOurs, StrategyTheirs, StrategyManual)
	}

	// Validate per-table strategies
	for i, ts := range c.Strategies {
		if ts.Table == "" {
			return fmt.Errorf("conflict_resolution.strategies[%d].table: table name is required", i)
		}
		if ts.Strategy == "" {
			return fmt.Errorf("conflict_resolution.strategies[%d].strategy: strategy is required for table %q", i, ts.Table)
		}
		if !isValidStrategy(ts.Strategy) {
			return fmt.Errorf("conflict_resolution.strategies[%d].strategy: invalid value %q for table %q (must be %q, %q, or %q)",
				i, ts.Strategy, ts.Table, StrategyOurs, StrategyTheirs, StrategyManual)
		}
	}

	return nil
}

// GetStrategyForTable returns the conflict resolution strategy for the given table.
// It first checks for a table-specific override, then falls back to the default strategy.
func (c *ConflictResolutionConfig) GetStrategyForTable(tableName string) string {
	// Check for table-specific override
	for _, ts := range c.Strategies {
		if ts.Table == tableName {
			return ts.Strategy
		}
	}
	// Fall back to default strategy
	return c.DefaultStrategy
}
