---
sidebar_position: 2
---

# check

Validates the configuration file and tests connectivity to both databases. Run this first before any other command.

## Usage

```bash
deepdiffdb check --config deepdiffdb.config.yaml
```

## What It Does

1. Loads and validates the configuration file — checks for required fields, valid driver names, and valid port numbers.
2. Opens a connection to the **production** database and runs a lightweight ping query.
3. Opens a connection to the **development** database and runs a lightweight ping query.
4. Verifies that every non-ignored table in both databases has a primary key. Tables without primary keys cannot be diffed and must either be fixed or added to the `ignore.tables` list.
5. Checks that the `output.dir` directory exists and is writable (creates it if missing).
6. Prints a summary of the resolved configuration settings.

## Flags

| Flag | Description |
|---|---|
| `--config` | Path to the configuration file (default: `deepdiffdb.config.yaml`) |

## Example Output

```
✓ Config loaded
✓ Production database connected (postgres @ 127.0.0.1:5432/prod_db)
✓ Development database connected (postgres @ 127.0.0.1:5432/dev_db)
✓ All non-ignored tables have primary keys (12 tables checked)
✓ Output directory is writable: ./diff-output

Configuration Summary:
  Ignored tables : audit_logs, sessions
  Ignored columns: *.updated_at, *.created_at
  Batch size     : 10000
  Parallel tables: 1

Check passed.
```

## When to Use

- **Before starting any diff** — confirm connectivity before investing time in a long operation.
- **In CI/CD pipelines** — use as a readiness gate before running `diff` or `gen-pack`.
- **After editing the config file** — confirm the new settings are valid before running.

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | All checks passed |
| `1` | One or more checks failed (see stderr for details) |
