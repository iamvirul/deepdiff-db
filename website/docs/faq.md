---
sidebar_position: 9
---

# FAQ

## Does DeepDiff DB modify the production database directly?

No. The `diff`, `schema-diff`, and `gen-pack` commands are **read-only** — they only read from both databases and write reports to the local filesystem. The production database is only written to when you explicitly run `deepdiffdb apply` with a reviewed migration pack.

## What databases are supported?

As of v0.9, DeepDiff DB supports:
- **MySQL** 5.7+
- **PostgreSQL** 12+
- **SQLite** 3 (any version supported by `modernc.org/sqlite`)
- **Microsoft SQL Server** 2017+ (including Azure SQL Database) — added in v0.8
- **Oracle Database** 12c+ — added in v0.9

## Does it support schema-only migration (without data)?

Yes. Use `schema-migrate` to generate a schema-only DDL migration script:

```bash
deepdiffdb schema-migrate --config deepdiffdb.config.yaml
```

This does not touch data at all. Alternatively, `gen-pack` generates a combined schema + data migration pack.

## How does it handle large tables?

DeepDiff DB uses keyset-paginated batch hashing (v0.7+). Each page fetches a bounded number of rows using a cursor query (`WHERE pk > lastVal LIMIT N`), hashes the page, then discards it before fetching the next page. Memory usage is O(batch\_size) — flat regardless of total row count.

Tune with `--batch-size` and `--parallel`:

```bash
deepdiffdb diff --config deepdiffdb.config.yaml --batch-size 5000 --parallel 4
```

See [Streaming Large Datasets](/docs/features/streaming) for details.

## Is it safe to run in production?

Yes, for the diff phase. `diff`, `schema-diff`, and `gen-pack` only read from the production database and write to the local filesystem. They never write to, lock, or modify the production database.

`apply` writes to the production database, but only when you explicitly invoke it with a reviewed migration pack file. Even then, all writes are wrapped in a single transaction that rolls back automatically on any error.

## What is a conflict?

A conflict is a row where the primary key exists in **both** the production and development databases, but the column values differ. This means both sides have "valid" versions of the row and you need to decide which one to keep. See [Conflict Resolution](/docs/features/conflict-resolution) for details.

## Does DeepDiff DB require Docker?

No. The binary is fully standalone with no runtime dependencies. Docker is only needed if you want to run the MySQL, PostgreSQL, MSSQL, or Oracle sample projects (which use Docker Compose to spin up test databases). For SQLite workflows and for connecting to existing databases, Docker is not needed at all.

## What versions of each database are tested?

| Database | Minimum | Tested versions |
|---|---|---|
| MySQL | 5.7 | 5.7, 8.0, 8.4 |
| PostgreSQL | 12 | 13, 14, 15, 16 |
| SQLite | 3 | Any (modernc.org/sqlite is self-contained) |
| Microsoft SQL Server | 2017 | 2019, 2022, Azure SQL |
| Oracle | 12c | 19c, 21c XE |

## Can I use it without a config file?

Not currently. A `deepdiffdb.config.yaml` (or file specified with `--config`) is required for all commands. The config file defines the database connections and behavior settings that all commands depend on.

## How do I ignore tables or columns?

Add them to the `ignore` section of the config:

```yaml
ignore:
  tables:
    - "audit_logs"
    - "sessions"
  columns:
    - "*.updated_at"        # all tables
    - "users.last_login"    # specific table
```

## Does it work with read replicas?

Yes. The production connection can point to a read replica for the diff phase (since diff is read-only). For `apply`, point the production connection at the primary/master instance.

## What happens if the apply is interrupted mid-way?

The apply is wrapped in a single database transaction. If the process is killed or the connection drops mid-apply, the database transaction is either rolled back by the database (if the connection drop is detected) or left open until the session timeout. In either case, production data is not partially modified.

For large packs, use `--resume` to restart apply from the last checkpoint rather than from the beginning.

## Can I generate a migration pack without running a diff first?

Yes. `gen-pack` performs its own internal diff — you do not need to run `diff` separately before `gen-pack`. Running `diff` first is optional and useful if you want to inspect the diff output before committing to pack generation.
