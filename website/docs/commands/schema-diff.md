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

- **Added tables** — tables that exist in dev but not in prod
- **Removed tables** — tables that exist in prod but not in dev
- **Column changes** — per-column: data type, nullability, default value, column order
- **New columns** — columns added in dev that are absent in prod
- **Dropped columns** — columns present in prod that are absent in dev
- **Index changes** — added, removed, or modified indexes (unique/non-unique)
- **Foreign key changes** — added or removed FK constraints

Tables listed in `ignore.tables` are excluded entirely.

## Flags

| Flag | Description |
|---|---|
| `--config` | Path to the configuration file (default: `deepdiffdb.config.yaml`) |
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

## Example

```bash
deepdiffdb schema-diff --config deepdiffdb.config.yaml --log-level warn
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
