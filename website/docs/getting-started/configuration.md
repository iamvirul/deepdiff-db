---
sidebar_position: 3
---

# Configuration

DeepDiff DB is configured through a YAML file. By convention it is named `deepdiffdb.config.yaml` and lives in the working directory, but you can point any command at a different file with the `--config` flag.

## Full Annotated Example

```yaml
# ─────────────────────────────────────────────
# Production database connection (read-only)
# ─────────────────────────────────────────────
prod:
  driver: "postgres"       # mysql | postgres | postgresql | sqlite | mssql | oracle
  host: "127.0.0.1"
  port: 5432
  user: "postgres"
  password: "secret"
  database: "prod_db"

# ─────────────────────────────────────────────
# Development database connection (read-only)
# ─────────────────────────────────────────────
dev:
  driver: "postgres"
  host: "127.0.0.1"
  port: 5432
  user: "postgres"
  password: "secret"
  database: "dev_db"

# ─────────────────────────────────────────────
# Tables and columns to exclude from all diffs
# ─────────────────────────────────────────────
ignore:
  tables:
    - "audit_logs"         # exact table name
    - "sessions"
  columns:
    - "*.updated_at"       # wildcard: ignore updated_at on every table
    - "*.created_at"
    - "users.last_login"   # table-qualified: ignore only on users table

# ─────────────────────────────────────────────
# Where to write reports and migration files
# ─────────────────────────────────────────────
output:
  dir: "./diff-output"     # created automatically if it does not exist

# ─────────────────────────────────────────────
# Safety controls for schema migration
# All default to false. Set to true to allow.
# ─────────────────────────────────────────────
migration:
  allow_drop_column: false
  allow_drop_table: false
  allow_drop_index: false
  allow_drop_foreign_key: false
  allow_modify_primary_key: false

# ─────────────────────────────────────────────
# Conflict resolution
# ─────────────────────────────────────────────
conflict_resolution:
  # Default strategy for every table: ours | theirs | manual
  #   ours   — keep production values
  #   theirs — use development values
  #   manual — require interactive decision
  default_strategy: "manual"

  # Per-table overrides (applied before default_strategy)
  strategies:
    - table: "feature_flags"
      strategy: "theirs"   # always accept dev version of feature flags
    - table: "config"
      strategy: "ours"     # never overwrite production config rows

# ─────────────────────────────────────────────
# Performance tuning (v0.7+)
# ─────────────────────────────────────────────
performance:
  # Rows per keyset-paginated query during table hashing.
  # 0 = load entire table in one query (pre-v0.7 behaviour).
  hash_batch_size: 10000

  # Maximum tables hashed concurrently.
  max_parallel_tables: 1
```

## Connection Options

| Field | Required | Description |
|---|---|---|
| `driver` | Yes | Database driver identifier (see table below) |
| `host` | Yes (except SQLite) | Hostname or IP of the database server |
| `port` | No | Port number. Defaults: MySQL 3306, PostgreSQL 5432, MSSQL 1433, Oracle 1521 |
| `user` | Yes (except SQLite) | Database username |
| `password` | Yes (except SQLite) | Database password |
| `database` | Yes | Database name. For Oracle: the service name (e.g. `XEPDB1`) |

## Driver Connection Snippets

### MySQL

```yaml
prod:
  driver: "mysql"
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "secret"
  database: "prod_db"
```

### PostgreSQL

```yaml
prod:
  driver: "postgres"
  host: "127.0.0.1"
  port: 5432
  user: "postgres"
  password: "secret"
  database: "prod_db"
```

Both `postgres` and `postgresql` are accepted as the driver value.

### SQLite

For SQLite the `host`, `port`, `user`, and `password` fields are ignored. Set `database` to the file path.

```yaml
prod:
  driver: "sqlite"
  database: "/var/data/prod.db"

dev:
  driver: "sqlite"
  database: "/var/data/dev.db"
```

### Microsoft SQL Server (v0.8+)

Port defaults to 1433 when omitted.

```yaml
prod:
  driver: "mssql"
  host: "127.0.0.1"
  port: 1433
  user: "sa"
  password: "StrongP@ss1word!"
  database: "prod_db"
```

### Oracle (v0.9+)

Port defaults to 1521 when omitted. Set `database` to the Oracle service name.

```yaml
prod:
  driver: "oracle"
  host: "127.0.0.1"
  port: 1521
  user: "system"
  password: "secret"
  database: "XEPDB1"
```

## Ignore Patterns

The `ignore.columns` list supports two forms:

- `*.column_name` — ignores that column on every table
- `table_name.column_name` — ignores that column on a specific table only

Ignored columns are excluded from row hashing, so changes to those columns will not appear as conflicts or data differences.

## Migration Safety Controls

By default all destructive DDL operations are blocked. When `schema-migrate` or `gen-pack` encounters a blocked operation it emits a warning and comments out the statement in the generated SQL. Set the relevant flag to `true` to allow the operation to be generated as executable SQL.

| Key | Controls |
|---|---|
| `allow_drop_column` | `DROP COLUMN` / `DROP COLUMN` equivalent per driver |
| `allow_drop_table` | `DROP TABLE` |
| `allow_drop_index` | `DROP INDEX` |
| `allow_drop_foreign_key` | `DROP FOREIGN KEY` / `DROP CONSTRAINT` |
| `allow_modify_primary_key` | `DROP PRIMARY KEY` + `ADD PRIMARY KEY` |

## Conflict Resolution Strategies

| Strategy | Behaviour |
|---|---|
| `ours` | Keep the production row value; discard the dev change |
| `theirs` | Replace the production row value with the dev value |
| `manual` | Flag for interactive resolution via `resolve-conflicts` |

## Performance Options

| Key | Default | Description |
|---|---|---|
| `performance.hash_batch_size` | `10000` | Rows per keyset-paginated query. Set to `0` to load the whole table in one query. |
| `performance.max_parallel_tables` | `1` | Maximum tables hashed concurrently. Increase on multi-core hosts with fast storage. |

These values can also be overridden at runtime with `--batch-size` and `--parallel` flags on the `diff` and `gen-pack` commands.
