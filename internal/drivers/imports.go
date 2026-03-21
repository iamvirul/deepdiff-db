package drivers

import (
	// Register MySQL driver.
	_ "github.com/go-sql-driver/mysql"
	// Register PostgreSQL driver.
	_ "github.com/jackc/pgx/v5/stdlib"
	// Register MSSQL driver.
	_ "github.com/microsoft/go-mssqldb"
	// Register Oracle driver.
	_ "github.com/sijms/go-ora/v2"
	// Register SQLite driver.
	_ "modernc.org/sqlite"
)
