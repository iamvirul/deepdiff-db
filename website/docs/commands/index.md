---
sidebar_position: 1
---

# Commands

DeepDiff DB exposes a set of focused commands. Each command does one thing; chain them in the order below for the standard workflow.

## Command Overview

| Command | Purpose |
|---|---|
| `check` | Validate config file and test connections to both databases |
| `schema-diff` | Detect schema drift between production and development |
| `schema-migrate` | Generate a standalone schema migration SQL script |
| `diff` | Full diff: schema introspection + row-level data comparison |
| `gen-pack` | Generate a complete SQL migration pack (schema + data) |
| `apply` | Apply a migration pack to the production database |
| `resolve-conflicts` | Interactively (or automatically) resolve data conflicts |
| `version` | Git-like versioning: commit diff snapshots, browse history, compare versions, generate rollback SQL |

## Global Flags

The following flags are accepted by all commands:

| Flag | Default | Description |
|---|---|---|
| `--config` | `deepdiffdb.config.yaml` | Path to the configuration file |
| `--verbose` | `false` | Enable debug-level logging with source location |
| `--log-file` | _(none)_ | Write logs to this file in addition to stdout |
| `--log-level` | `info` | Minimum log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | `text` | Log format: `text` or `json` |
| `--batch-size` | _(from config)_ | Rows per keyset-paginated query (overrides `performance.hash_batch_size`) |
| `--parallel` | _(from config)_ | Max tables hashed concurrently (overrides `performance.max_parallel_tables`) |
| `--resume` | `false` | Resume from the last saved checkpoint |

Not every flag applies to every command. Each command page lists the flags it accepts.

## Recommended Workflow

```bash
# 1. Validate config and connections
deepdiffdb check --config deepdiffdb.config.yaml

# 2. (Optional) Check for schema drift first
deepdiffdb schema-diff --config deepdiffdb.config.yaml

# 3. (Optional) Generate schema migration SQL if there is drift
deepdiffdb schema-migrate --config deepdiffdb.config.yaml

# 4. Run a full diff (schema + data)
deepdiffdb diff --config deepdiffdb.config.yaml

# 5. (Optional) Resolve conflicts interactively
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml

# 6. Generate the migration pack
deepdiffdb gen-pack --config deepdiffdb.config.yaml

# 7. Review diff-output/migration_pack.sql, then apply
deepdiffdb apply --pack diff-output/migration_pack.sql --config deepdiffdb.config.yaml
```
