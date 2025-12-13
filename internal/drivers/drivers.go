package drivers

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/config"
)

// Open returns a ready-to-use *sql.DB for the given database config and driver.
// It validates the driver, builds a DSN, and pings the database to ensure connectivity.
func Open(ctx context.Context, cfg config.DBConfig) (*sql.DB, error) {
	driverName, dsn, err := BuildDSN(cfg)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", driverName, err)
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping %s: %w", driverName, err)
	}

	return db, nil
}

// BuildDSN builds a DSN string for the given database config.
// Exported for testing purposes.
func BuildDSN(cfg config.DBConfig) (string, string, error) {
	switch cfg.Driver {
	case "mysql":
		return "mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
			url.QueryEscape(cfg.User),
			url.QueryEscape(cfg.Password),
			net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port)),
			cfg.Database,
		), nil
	case "postgres", "postgresql":
		// pgx DSN format
		return "pgx", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host,
			cfg.Port,
			cfg.User,
			cfg.Password,
			cfg.Database,
		), nil
	case "sqlite":
		// Database is treated as the filepath for sqlite.
		if cfg.Database == "" {
			return "", "", fmt.Errorf("sqlite database path is required")
		}
		return "sqlite", cfg.Database, nil
	default:
		return "", "", fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}
}
