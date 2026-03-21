---
sidebar_position: 2
---

# Quick Start

This guide walks through the standard DeepDiff DB workflow from first config to applying a migration. It assumes you have already [installed](/docs/getting-started/installation) the binary.

## Step 1 — Create the Configuration File

Create `deepdiffdb.config.yaml` in your working directory. This example connects two local PostgreSQL databases:

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

ignore:
  tables:
    - "audit_logs"
  columns:
    - "*.updated_at"
    - "*.created_at"

output:
  dir: "./diff-output"

conflict_resolution:
  default_strategy: "manual"
```

See [Configuration](/docs/getting-started/configuration) for the full reference including MySQL, SQLite, MSSQL, and Oracle examples.

## Step 2 — Validate Connections

```bash
deepdiffdb check --config deepdiffdb.config.yaml
```

Expected output:

```
✓ Config loaded
✓ Production database connected (postgres @ 127.0.0.1:5432/prod_db)
✓ Development database connected (postgres @ 127.0.0.1:5432/dev_db)
✓ All tables have primary keys
✓ Output directory is writable: ./diff-output
Check passed.
```

If any check fails the command exits with a non-zero status and prints an actionable error message. Fix connection issues before proceeding.

## Step 3 — Run a Full Diff

```bash
deepdiffdb diff --config deepdiffdb.config.yaml
```

DeepDiff DB will:
1. Introspect the schema of both databases and write `schema_diff.json` and `schema_diff.txt`
2. Hash every row in every non-ignored table and compare the hashes
3. Write `content_diff.json`, `conflicts.json`, and `summary.txt` to `./diff-output`

Example `summary.txt`:

```
Schema: OK
Tables scanned: 8
Added rows: 24
Updated rows: 6
Conflicts: 3

Migration pack: (not yet generated — run gen-pack)
```

## Step 4 — Generate a Migration Pack

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml
```

This produces `./diff-output/migration_pack.sql` — a transactional SQL script that applies all data differences. It also re-runs the diff internally so you always get a fresh pack.

For large databases, add streaming flags:

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml --batch-size 5000 --parallel 4
```

## Step 5 — Review the Migration Pack

Open `./diff-output/migration_pack.sql` and verify the statements look correct before applying anything to production. The file is a plain SQL script — review it with your normal code review process.

You can also generate an interactive HTML report for easier review:

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml --html
```

Then open `./diff-output/report.html` in your browser.

## Step 6 — Apply the Migration

Once satisfied with the pack:

```bash
deepdiffdb apply --pack diff-output/migration_pack.sql --config deepdiffdb.config.yaml
```

The apply command wraps every statement in a single transaction. On any error the transaction rolls back automatically, leaving production unchanged.

To validate without executing:

```bash
deepdiffdb apply --pack diff-output/migration_pack.sql --dry-run
```

Expected output on success:

```
Applying migration pack: diff-output/migration_pack.sql
Statements executed: 31
Transaction committed.
Apply complete.
```

## What's Next

- [Configuration Reference](/docs/getting-started/configuration) — all config keys explained
- [Commands Reference](/docs/commands/) — full flag documentation for every command
- [Conflict Resolution](/docs/features/conflict-resolution) — handling rows that differ in both databases
- [Streaming Large Datasets](/docs/features/streaming) — tuning for tables with millions of rows
