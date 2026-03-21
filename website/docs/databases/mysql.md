---
sidebar_position: 1
---

# MySQL

DeepDiff DB supports MySQL 5.7 and MySQL 8.x as both production and development database targets.

## Requirements

- MySQL 5.7+ or MySQL 8.x
- The database user must have `SELECT` privileges on the target database and `SHOW FULL TABLES` / `INFORMATION_SCHEMA` read access

## Connection Configuration

```yaml
prod:
  driver: "mysql"
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "secret"
  database: "prod_db"

dev:
  driver: "mysql"
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "secret"
  database: "dev_db"
```

The `port` field defaults to `3306` when omitted.

## Go Driver

DeepDiff DB uses [`github.com/go-sql-driver/mysql`](https://github.com/go-sql-driver/mysql) internally. No separate MySQL client libraries are required — the driver is statically compiled into the binary.

## Identifier Quoting

MySQL identifiers (table and column names) are quoted with backticks:

```sql
ALTER TABLE `orders` MODIFY COLUMN `status` VARCHAR(50) NOT NULL;
CREATE INDEX `idx_orders_status` ON `orders` (`status`);
```

## Foreign Key Handling During Pack Apply

When applying a migration pack to a MySQL database, DeepDiff DB temporarily disables foreign key checks to allow rows to be inserted/deleted in any order without constraint violations:

```sql
SET FOREIGN_KEY_CHECKS = 0;
-- ... data changes ...
SET FOREIGN_KEY_CHECKS = 1;
```

This is safe because the migration pack is applied inside a single transaction and FK integrity is checked again when `FOREIGN_KEY_CHECKS` is re-enabled.

## Schema Introspection

MySQL schema is introspected via `INFORMATION_SCHEMA.COLUMNS`, `INFORMATION_SCHEMA.STATISTICS`, and `INFORMATION_SCHEMA.KEY_COLUMN_USAGE` / `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS`.

## Limitations

- MySQL 5.7 has limited support for generated/computed columns — these may not be fully represented in the schema diff.
- `JSON` column type differences are detected at the type level only; DeepDiff DB does not perform deep JSON comparison.
