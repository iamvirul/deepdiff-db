package drivers

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/config"
	"github.com/iamvirul/deepdiff-db/pkg/errors"
)

// Open returns a ready-to-use *sql.DB for the given database config and driver.
// Open creates and returns a ready-to-use *sql.DB for the given DBConfig.
// It validates the driver and DSN, opens the database driver, sets connection pool
// parameters (connection max lifetime 5 minutes, max open connections 10, max idle connections 5),
// and verifies connectivity by pinging the database with a 5-second timeout.
// If the ping fails the opened connection is closed and an error containing the driver name is returned.
// Open uses retry logic for transient connection errors.
func Open(ctx context.Context, cfg config.DBConfig) (*sql.DB, error) {
	driverName, dsn, err := BuildDSN(cfg)
	if err != nil {
		return nil, err
	}

	var db *sql.DB
	
	// Retry connection with exponential backoff
	retryCfg := errors.DefaultRetryConfig()
	err = errors.Retry(ctx, retryCfg, func() error {
		var openErr error
		db, openErr = sql.Open(driverName, dsn)
		if openErr != nil {
			return errors.Wrap(openErr, errors.ErrConnectionFailed, fmt.Sprintf("open %s connection", driverName))
		}

		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if pingErr := db.PingContext(pingCtx); pingErr != nil {
			_ = db.Close()
			return errors.Wrap(pingErr, errors.ErrConnectionFailed, fmt.Sprintf("ping %s database", driverName))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return db, nil
}

// BuildDSN builds a DSN string for the given database config.
// BuildDSN constructs a driver name and DSN string from cfg for use with sql.Open.
//
// For MySQL it returns driver "mysql" and a DSN that includes parseTime=true, UTF-8 mb4 charset,
// and utf8mb4_unicode_ci collation; user and password are URL-escaped and host/port are joined.
// For PostgreSQL it returns driver "pgx" and a lib/pq-like DSN with sslmode=disable.
// For MSSQL it returns driver "sqlserver" (the registered name for go-mssqldb) and a URL-form DSN.
// For SQLite it returns driver "sqlite" and treats cfg.Database as the file path.
//
// It returns the chosen driver name, the corresponding DSN string, and an error if the driver
// is unsupported or if SQLite is selected but cfg.Database is empty.
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
	case "mssql":
		// go-mssqldb registers under the "sqlserver" driver name.
		// URL DSN: sqlserver://user:password@host:port?database=db
		// Omit the port segment when cfg.Port == 0 so MSSQL defaults to 1433.
		host := cfg.Host
		if cfg.Port != 0 {
			host = net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
		}
		dsn := fmt.Sprintf("sqlserver://%s:%s@%s?database=%s",
			url.QueryEscape(cfg.User),
			url.QueryEscape(cfg.Password),
			host,
			url.QueryEscape(cfg.Database),
		)
		return "sqlserver", dsn, nil
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