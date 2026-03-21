---
sidebar_position: 5
---

# Oracle Database

Oracle Database support was added in **v0.9**. DeepDiff DB supports Oracle 12c and later (including Oracle 19c and Oracle 21c XE).

## Requirements

- Oracle Database 12c or later (12c is required for `OFFSET/FETCH` pagination)
- The database user must have `SELECT` on target tables and `SELECT` on `ALL_TAB_COLUMNS`, `ALL_INDEXES`, `ALL_IND_COLUMNS`, `ALL_CONSTRAINTS`, and `ALL_CONS_COLUMNS`
- No Oracle Instant Client or native Oracle libraries required — the Go driver is pure Go

## Connection Configuration

Set `database` to the Oracle **service name** (not the SID). For Oracle XE the default service name is `XEPDB1`.

```yaml
prod:
  driver: "oracle"
  host: "127.0.0.1"
  port: 1521
  user: "system"
  password: "secret"
  database: "XEPDB1"

dev:
  driver: "oracle"
  host: "127.0.0.1"
  port: 1522
  user: "system"
  password: "secret"
  database: "XEPDB1"
```

The `port` field is optional and defaults to `1521` when omitted.

## Go Driver

DeepDiff DB uses [`github.com/sijms/go-ora/v2`](https://github.com/sijms/go-ora) — a pure Go Oracle driver that communicates directly over the Oracle Net protocol. No Oracle Instant Client, OCI libraries, or `LD_LIBRARY_PATH` configuration is required.

The DSN format used internally is:

```
oracle://user:password@host:port/service_name
```

## Identifier Quoting

Oracle identifiers are quoted with double-quotes (same as PostgreSQL and the SQL standard):

```sql
ALTER TABLE "orders" ADD "shipped_at" TIMESTAMP NULL;
DROP INDEX "idx_orders_status";
```

## DDL Syntax Differences

Oracle uses different DDL syntax from MySQL and PostgreSQL:

| Operation | Oracle syntax |
|---|---|
| Add column | `ALTER TABLE "t" ADD "col" TYPE` (no `COLUMN` keyword) |
| Modify column | `ALTER TABLE "t" MODIFY "col" TYPE` (not `ALTER COLUMN`) |
| Drop column | `ALTER TABLE "t" DROP COLUMN "col"` |
| Drop index | `DROP INDEX "idx_name"` (standalone — no table name) |
| Drop FK | `ALTER TABLE "t" DROP CONSTRAINT "fk_name"` |
| Auto-increment | `"id" NUMBER GENERATED ALWAYS AS IDENTITY` |

## Pagination

Oracle 12c+ supports standard SQL row-limiting syntax. DeepDiff DB uses it for keyset pagination:

```sql
SELECT * FROM "table"
WHERE "pk" > :lastVal
ORDER BY "pk"
OFFSET 0 ROWS FETCH NEXT 10000 ROWS ONLY
```

## Foreign Key Handling During Pack Apply

Oracle does not have a global FK disable command. DeepDiff DB disables FK constraints per-table using:

```sql
ALTER TABLE "table_name" DISABLE CONSTRAINT "fk_name";
-- ... data changes ...
ALTER TABLE "table_name" ENABLE CONSTRAINT "fk_name";
```

## Schema Introspection

Oracle schema is introspected via the `ALL_*` catalog views (not `USER_*`) to support cross-schema access:

- `ALL_TAB_COLUMNS` — column metadata
- `ALL_INDEXES` + `ALL_IND_COLUMNS` — index metadata
- `ALL_CONSTRAINTS` + `ALL_CONS_COLUMNS` — PK, FK, and unique constraints

## Supported Oracle Data Types

| Oracle type | Notes |
|---|---|
| `NUMBER(p,s)` | Numeric with precision and scale |
| `VARCHAR2(n)` | Variable-length string |
| `CHAR(n)` | Fixed-length string |
| `DATE` | Date and time (Oracle DATE includes time component) |
| `TIMESTAMP` | High-precision timestamp |
| `CLOB` | Character large object |
| `BLOB` | Binary large object |
| `FLOAT` | Floating-point number |

## Limitations

- Requires Oracle 12c or later for `OFFSET/FETCH` pagination. Oracle 11g and earlier are not supported.
- `CLOB` and `BLOB` columns are included in row hashing as string values; very large LOB values may affect performance.
- Oracle `ROWID` pseudo-column is not used as a primary key substitute — tables must have an explicit primary key or be added to `ignore.tables`.
- Special characters in Oracle passwords (e.g., `@`, `/`, `#`) must be handled carefully. If the password contains special characters, test connectivity with `deepdiffdb check` first; an `ORA-01017` error indicates an incorrect password or DSN encoding issue.
