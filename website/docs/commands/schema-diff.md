---
sidebar_position: 3
---

# schema-diff

Detects and reports schema differences between the production and development databases without touching any data.

## Usage

```bash
deepdiffdb schema-diff --config deepdiffdb.config.yaml
```

## What It Checks

**Tables**
- **Added tables** — tables that exist in dev but not in prod
- **Removed tables** — tables that exist in prod but not in dev
- **Column changes** — per-column: data type, nullability, default value, column order
- **New columns** — columns added in dev that are absent in prod
- **Dropped columns** — columns present in prod that are absent in dev
- **Index changes** — added, removed, or modified indexes (unique/non-unique)
- **Foreign key changes** — added or removed FK constraints

**Schema objects**
- **Views** — added, removed, and modified views; materialised view flag changes (PostgreSQL)
- **Routines** — added, removed, and modified stored procedures and functions; detects definition, kind, return type, language, and parameter changes
- **Triggers** — added, removed, and modified triggers; detects timing (BEFORE/AFTER/INSTEAD OF), event (INSERT/UPDATE/DELETE), and definition changes
- **Sequences** — added, removed, and modified sequences (PostgreSQL only); tracks start value, increment, min/max value, cache size, and cycle flag

Tables listed in `ignore.tables` are excluded entirely. Use `ignore.views`, `ignore.routines`, `ignore.triggers`, and `ignore.sequences` to exclude specific schema objects by name.

## Flags

| Flag | Description |
|---|---|
| `--config` | Path to the configuration file (default: `deepdiffdb.config.yaml`) |
| `--output-dir` | Override `output.dir` from config at runtime (e.g. point to a temp dir in CI or pre-commit hooks) |
| `--quiet` | Suppress progress bars, metrics summary, and informational output. Automatically raises log level to `warn` so the tool is fully silent on success. Explicit `--log-level debug` overrides this. |
| `--verbose` | Enable debug-level logging |
| `--log-level` | Minimum log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `--log-format` | Log output format: `text` or `json` (default: `text`) |
| `--log-file` | Write logs to this file in addition to stdout |

## Output Files

Both files are written to `output.dir` (default: `./diff-output`).

| File | Description |
|---|---|
| `schema_diff.json` | Machine-readable schema differences, suitable for programmatic processing or CI checks |
| `schema_diff.txt` | Human-readable schema diff report |

## Examples

```bash
# Standard usage
deepdiffdb schema-diff --config deepdiffdb.config.yaml --log-level warn

# CI / pre-commit: fully silent on success, non-zero exit on drift
deepdiffdb schema-diff --config deepdiffdb.config.yaml --quiet

# Redirect output to a temp dir (pre-commit hooks, ephemeral CI runners)
deepdiffdb schema-diff --config deepdiffdb.config.yaml \
  --output-dir /tmp/deepdiff-$$ \
  --quiet
```

Example `schema_diff.txt` output:

```
Schema Diff Report
==================

Table: orders
  MODIFIED column: status
    prod: VARCHAR(20) NOT NULL DEFAULT 'pending'
    dev : VARCHAR(50) NOT NULL DEFAULT 'new'

  ADDED column: shipped_at
    dev : TIMESTAMP NULL

Table: products
  ADDED index: idx_products_category (non-unique)
    columns: category_id

Table: invoices
  STATUS: ADDED (exists in dev, missing in prod)

Summary: 1 table added, 2 tables modified, 0 tables removed
```

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | No schema drift detected |
| `1` | Schema drift detected or an error occurred |

The non-zero exit on drift makes `schema-diff` useful as a CI/CD gate to block deployments when databases have diverged.
