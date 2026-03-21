---
sidebar_position: 4
---

# Microsoft SQL Server

Microsoft SQL Server support was added in **v0.8**. DeepDiff DB supports SQL Server 2017 and later, including Azure SQL Database.

## Requirements

- SQL Server 2017+ or Azure SQL Database
- The database user must have `SELECT` on all target tables and `SELECT` on `INFORMATION_SCHEMA`, `sys.tables`, `sys.indexes`, `sys.index_columns`, and `sys.key_constraints`

## Connection Configuration

```yaml
prod:
  driver: "mssql"
  host: "127.0.0.1"
  port: 1433
  user: "sa"
  password: "StrongP@ss1word!"
  database: "prod_db"

dev:
  driver: "mssql"
  host: "127.0.0.1"
  port: 1434
  user: "sa"
  password: "StrongP@ss1word!"
  database: "dev_db"
```

The `port` field is optional and defaults to `1433` when omitted.

## Go Driver

DeepDiff DB uses [`github.com/microsoft/go-mssqldb`](https://github.com/microsoft/go-mssqldb) with the `sqlserver://` DSN format. No ODBC drivers or native SQL Server client libraries are required.

## Identifier Quoting

SQL Server identifiers are quoted with square brackets, with `]]` used to escape a literal `]` character:

```sql
ALTER TABLE [orders] ADD [shipped_at] DATETIME NULL;
CREATE INDEX [idx_orders_status] ON [orders] ([status]);
```

## DDL Syntax Differences

SQL Server uses slightly different ALTER TABLE syntax:

| Operation | SQL Server syntax |
|---|---|
| Add column | `ALTER TABLE [t] ADD [col] TYPE` (no `COLUMN` keyword) |
| Modify column | `ALTER TABLE [t] ALTER COLUMN [col] TYPE` |
| Drop column | `ALTER TABLE [t] DROP COLUMN [col]` |
| Drop index | `DROP INDEX [idx] ON [table]` |
| Drop FK | `ALTER TABLE [t] DROP CONSTRAINT [fk_name]` |

## Pagination

SQL Server does not support `LIMIT / OFFSET`. DeepDiff DB uses the SQL Server standard pagination syntax for keyset queries:

```sql
SELECT ... FROM [table]
WHERE [pk] > @lastVal
ORDER BY [pk]
OFFSET 0 ROWS FETCH NEXT 10000 ROWS ONLY;
```

## Foreign Key Handling During Pack Apply

DeepDiff DB uses `sp_msforeachtable` to disable and re-enable all FK constraints on the target database during pack application:

```sql
EXEC sp_msforeachtable 'ALTER TABLE ? NOCHECK CONSTRAINT ALL';
-- ... data changes ...
EXEC sp_msforeachtable 'ALTER TABLE ? WITH CHECK CHECK CONSTRAINT ALL';
```

## Column Type Handling

SQL Server stores character column lengths in bytes for `nchar` and `nvarchar` (because they are UTF-16, 2 bytes per character). DeepDiff DB applies a half-length correction when reading these types so that `nvarchar(255)` in the catalog is correctly reported as `NVARCHAR(255)` (characters) rather than `NVARCHAR(510)`.

## Schema Introspection

SQL Server schema is introspected via `INFORMATION_SCHEMA.COLUMNS`, `sys.indexes`, `sys.index_columns`, `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS`, and `sys.key_constraints`.

## Limitations

- SQL Server `ROWVERSION` / `TIMESTAMP` columns are excluded from row hashing (their value changes on every update and would produce false positives).
- Temporal tables (system-versioned) are not currently supported.
- Named instances (e.g. `host\SQLEXPRESS`) are not tested; use the host + explicit port form instead.
