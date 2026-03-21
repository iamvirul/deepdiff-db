---
sidebar_position: 6
---

# gen-pack

Generates a complete SQL migration pack that includes both schema changes (DDL) and data changes (DML). This is the main output artifact of DeepDiff DB — a single, reviewable SQL file ready to apply to production.

## Usage

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml
```

## What It Generates

The migration pack is a single transactional SQL file containing:

- **Schema DDL** — `ALTER TABLE`, `CREATE TABLE`, `CREATE INDEX`, `ADD CONSTRAINT` statements for schema differences
- **Data DML** — `DELETE` + `INSERT` pairs (or `UPDATE`) for each changed row, batched in groups of 1000
- **Foreign key management** — Driver-specific statements to disable FK checks before data changes and re-enable them after (e.g., `SET FOREIGN_KEY_CHECKS=0` for MySQL, `DISABLE CONSTRAINT ALL` for Oracle)
- Everything wrapped in a single `BEGIN` / `COMMIT` transaction

`gen-pack` internally runs the same diff logic as `diff`, so you always get a fresh pack based on the current state of both databases.

## Flags

| Flag | Description |
|---|---|
| `--config` | Path to the configuration file (default: `deepdiffdb.config.yaml`) |
| `--html` | Generate an interactive HTML report (`report.html`) |
| `--batch-size N` | Rows per keyset-paginated query (overrides `performance.hash_batch_size`) |
| `--parallel N` | Max tables hashed concurrently (overrides `performance.max_parallel_tables`) |
| `--resume` | Resume from the last saved checkpoint |
| `--verbose` | Enable debug-level logging |
| `--log-level` | Minimum log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `--log-format` | Log output format: `text` or `json` (default: `text`) |
| `--log-file` | Write logs to this file in addition to stdout |

## Output Files

| File | Description |
|---|---|
| `schema_diff.json` | Schema differences (informational; schema drift emits warnings but does not block) |
| `content_diff.json` | Row-level differences |
| `conflicts.json` | Conflict details |
| `summary.txt` | Summary statistics |
| `resolutions_summary.json` | Resolution statistics (when conflict resolutions exist) |
| `migration_pack.sql` | The complete, transactional migration script |
| `report.html` | Interactive HTML report (only when `--html` is used) |

## Example

```bash
# Standard pack generation
deepdiffdb gen-pack --config deepdiffdb.config.yaml

# With HTML report and parallelism for large databases
deepdiffdb gen-pack --config deepdiffdb.config.yaml --html --batch-size 5000 --parallel 4

# Resume a previously interrupted gen-pack
deepdiffdb gen-pack --config deepdiffdb.config.yaml --resume
```

## Schema Drift Behaviour

Unlike `diff`, `gen-pack` **does not abort on schema drift**. It emits warnings for tables with schema differences and skips data diffing on those tables. Tables with matching schemas are processed normally. This lets you generate a pack that handles both the schema migration and data migration in a single file.

## Resuming Interrupted Runs

If a `gen-pack` run is interrupted (network failure, process kill), restart with `--resume`:

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml --resume
```

DeepDiff DB loads the last checkpoint and continues from where it left off. Checkpoints expire after 24 hours. See [Checkpoint and Resume](/docs/features/checkpoint-resume) for details.
