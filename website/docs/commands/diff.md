---
sidebar_position: 5
---

# diff

Runs a full comparison of both schema and data between the production and development databases. This is the read-only analysis step — it writes reports but makes no changes to either database.

## Usage

```bash
deepdiffdb diff --config deepdiffdb.config.yaml
```

## What It Does

1. Introspects the schema of both databases (same logic as `schema-diff`).
2. For every table with a matching schema in both databases: hashes each row using SHA-256 (excluding ignored columns) and compares the hash maps.
3. Classifies each row difference as: **added** (in dev only), **removed** (in prod only), or **updated** (exists in both, different values).
4. Identifies **conflicts** — rows where the primary key exists in both databases but the values differ in a way that requires a resolution decision.
5. Writes all reports to `output.dir`.

Tables with schema drift are skipped for data diffing and flagged in the report.

## Flags

| Flag | Description |
|---|---|
| `--config` | Path to the configuration file (default: `deepdiffdb.config.yaml`) |
| `--html` | Generate an interactive HTML report (`report.html`) in addition to JSON/text outputs |
| `--batch-size N` | Rows per keyset-paginated query (overrides `performance.hash_batch_size`) |
| `--parallel N` | Max tables hashed concurrently (overrides `performance.max_parallel_tables`) |
| `--verbose` | Enable debug-level logging |
| `--log-level` | Minimum log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `--log-format` | Log output format: `text` or `json` (default: `text`) |
| `--log-file` | Write logs to this file in addition to stdout |

## Output Files

All files are written to `output.dir` (default: `./diff-output`).

| File | Description |
|---|---|
| `schema_diff.json` | Machine-readable schema differences |
| `schema_diff.txt` | Human-readable schema diff |
| `content_diff.json` | Row-level differences: added, removed, updated rows per table |
| `conflicts.json` | Rows that exist in both databases with conflicting values |
| `summary.txt` | High-level statistics: tables scanned, rows added/updated/removed, conflict count |
| `report.html` | Interactive HTML report (only when `--html` is used) |

## Example

```bash
# Basic diff
deepdiffdb diff --config deepdiffdb.config.yaml

# With HTML report and streaming flags for large tables
deepdiffdb diff --config deepdiffdb.config.yaml --html --batch-size 5000 --parallel 4
```

Example `summary.txt`:

```
Schema: 1 table modified (orders), 1 table added (invoices)
Tables scanned: 8
  Tables with data diff: 3
  Tables skipped (schema drift): 1

Row differences:
  Added  : 24
  Removed: 2
  Updated: 11

Conflicts: 3
```

## Performance Notes

For tables with millions of rows, use `--batch-size` and `--parallel` to control memory usage and throughput. See [Streaming Large Datasets](/docs/features/streaming) for tuning guidance.
