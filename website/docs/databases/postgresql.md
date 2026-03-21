---
sidebar_position: 2
---

# PostgreSQL

DeepDiff DB supports PostgreSQL 12 and later as both production and development database targets.

## Requirements

- PostgreSQL 12+
- The database user must have `SELECT` on all target tables and `SELECT` on `information_schema` and `pg_catalog` views

## Connection Configuration

```yaml
prod:
  driver: "postgres"
  host: "127.0.0.1"
  port: 5432
  user: "postgres"
  password: "secret"
  database: "prod_db"

dev:
  driver: "postgres"
  host: "127.0.0.1"
  port: 5432
  user: "postgres"
  password: "secret"
  database: "dev_db"
```

Both `postgres` and `postgresql` are accepted as the `driver` value. The `port` field defaults to `5432` when omitted.

## Go Driver

DeepDiff DB uses [`github.com/lib/pq`](https://github.com/lib/pq) internally. No separate PostgreSQL client libraries are required.

## Identifier Quoting

PostgreSQL identifiers are quoted with double-quotes:

```sql
ALTER TABLE "orders" ADD COLUMN "shipped_at" TIMESTAMP NULL;
CREATE INDEX "idx_orders_status" ON "orders" ("status");
```

## Schema-Qualified Names

DeepDiff DB operates on the `public` schema by default. If your tables live in a different schema, include the schema name in the `database` field is not the right approach — connect directly to the database and ensure the `search_path` is set correctly for the connecting user.

## Column Type Handling

PostgreSQL uses verbose type names. DeepDiff DB normalises common aliases:

- `character varying(n)` is treated as `VARCHAR(n)`
- `integer` is treated as `INT4`
- `boolean` is treated as `BOOL`

Type differences are compared after normalisation, so cosmetic alias differences do not generate false positives.

## Foreign Key Handling During Pack Apply

PostgreSQL does not have a global FK check toggle equivalent to MySQL's `SET FOREIGN_KEY_CHECKS`. DeepDiff DB relies on the transaction-level `DEFERRABLE INITIALLY DEFERRED` capability where possible, or generates statements in dependency order (parent tables before child tables for inserts; child tables before parent tables for deletes).

## Schema Introspection

PostgreSQL schema is introspected via `information_schema.columns`, `pg_indexes`, and `information_schema.referential_constraints` / `information_schema.key_column_usage`.

## Limitations

- Table inheritance (`INHERITS`) is not currently supported.
- Partitioned tables may produce unexpected results — test with `check` before running a full diff.
- `GENERATED ALWAYS AS` computed columns are detected but not included in `ALTER TABLE` generation.
