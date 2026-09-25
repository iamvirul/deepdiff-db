---
sidebar_position: 6
---

# row-diff

:::info Added in v1.5.0
The `row-diff` command and `--row-diff` flag were added in DeepDiff DB v1.5.0.
:::

Inspects column-level differences for modified, added, or removed rows between production and development databases. Displays a side-by-side terminal comparison with differing columns highlighted, and writes machine-readable reports.

![row-diff Terminal Inspection](/img/row-diff-terminal.svg)

## Usage

```bash
# Inspect all updated rows across all tables
deepdiffdb row-diff --config deepdiffdb.config.yaml

# Inspect a specific table and primary key directly
deepdiffdb row-diff --table customers --key 4

# Inspect removed rows in JSON format
deepdiffdb row-diff --status removed --format json

# Run as part of a full diff
deepdiffdb diff --config deepdiffdb.config.yaml --row-diff
```

## What It Does

1. **Hashes & Discovers Differences**: Hashes tables across databases using keyset pagination and identifies modified, removed, or added primary keys.
2. **Cross-Engine Identifier Resolution**: Pairs tables and columns case-insensitively across different database engines (for example, PostgreSQL `customers.email` $\leftrightarrow$ Oracle `CUSTOMERS.EMAIL`).
3. **Selective Row Fetching**: Queries the actual raw column values directly using parameterized queries and dialect-specific identifier quoting. When `--table` and `--key` are supplied, it executes a targeted single-row lookup for instant results.
4. **Side-by-Side Column Comparison**: Compares column values across both engines, flags differences with `*`, and formats NULL and type variations cleanly.
5. **Generates Reports**: Outputs results to stdout and saves `row_diff.json` and `row_diff.txt` in the configured output directory.

## Flags

| Flag | Description | Default |
|---|---|---|
| `--config` | Path to configuration file | `deepdiffdb.config.yaml` |
| `--table` | Filter inspection to a specific table name | `""` (all tables) |
| `--key` | Filter inspection to a specific primary key value | `""` (all keys) |
| `--status` | Row status to inspect: `updated`, `added`, `removed`, or `all` | `updated` |
| `--limit` | Maximum differing rows to inspect per table (`0` = unlimited) | `0` |
| `--format` | Output format: `table` (terminal ASCII) or `json` | `table` |
| `--output` | Output directory to save reports | Config output directory |
| `--batch-size N` | Rows per keyset-paginated query | Config default |
| `--parallel N` | Max tables hashed concurrently | Config default |
| `--verbose` | Enable debug logging | `false` |

## Terminal Output Example

```text
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Table: customers | Key: 4 (updated)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Column               | Production               | Development             
  ---------------------+--------------------------+--------------------------
  country              | FR                       | FR                      
  email                | dan@example.com          | dan+new@example.com      *
  id                   | 4                        | 4                       
  name                 | Dan                      | Dan                     

  * indicates columns that differ
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total differing rows inspected: 1
Row diff report written to diff-output/row_diff.json and diff-output/row_diff.txt
```

## JSON Output Example

Passing `--format json` outputs structured data ideal for automated CI/CD assertion steps:

```json
{
  "total_rows": 1,
  "tables": [
    {
      "table": "customers",
      "rows": [
        {
          "table": "customers",
          "key": "4",
          "status": "updated",
          "columns": [
            {
              "column": "country",
              "prod_val": "FR",
              "dev_val": "FR",
              "differs": false
            },
            {
              "column": "email",
              "prod_val": "dan@example.com",
              "dev_val": "dan+new@example.com",
              "differs": true
            }
          ],
          "differing_columns": ["email"]
        }
      ]
    }
  ]
}
```

## Cross-Engine Migrations (PostgreSQL ↔ Oracle / MySQL)

Different database engines enforce different default identifier casing:
- **PostgreSQL**: folds unquoted identifiers to **lowercase** (`um_user_attribute`)
- **Oracle**: folds unquoted identifiers to **UPPERCASE** (`UM_USER_ATTRIBUTE`)

`deepdiffdb row-diff` seamlessly bridges this gap:
- Tables are discovered using canonical name folding.
- Each database query preserves the correct engine-specific quoting (`"customers"` in PostgreSQL, `"CUSTOMERS"` in Oracle, `` `customers` `` in MySQL, `[customers]` in MSSQL).
- Column values are paired by canonical name, allowing you to instantly identify which exact columns drifted during an ETL or replication pipeline.
