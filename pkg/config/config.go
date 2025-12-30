package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the full DeepDiff DB configuration.
type Config struct {
	Prod      DBConfig        `yaml:"prod"`
	Dev       DBConfig        `yaml:"dev"`
	Ignore    IgnoreConfig    `yaml:"ignore"`
	Output    OutputConfig    `yaml:"output"`
	Migration MigrationConfig `yaml:"migration"`
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

	// ConfirmDestructive requires manual confirmation for destructive operations.
	// When true, destructive operations include additional warnings.
	ConfirmDestructive bool `yaml:"confirm_destructive"`
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
	return nil
}

func (c *DBConfig) validate(prefix string) error {
	if c.Driver == "" {
		return fmt.Errorf("%s.driver is required", prefix)
	}
	if c.Database == "" {
		return fmt.Errorf("%s.database is required", prefix)
	}
	if c.Port == 0 && c.Driver != "sqlite" {
		return fmt.Errorf("%s.port is required", prefix)
	}
	switch c.Driver {
	case "mysql", "postgres", "postgresql", "sqlite":
	default:
		return fmt.Errorf("%s.driver unsupported: %s", prefix, c.Driver)
	}
	return nil
}

// ErrInvalidConfig indicates missing or invalid fields.
var ErrInvalidConfig = errors.New("invalid configuration")