---
sidebar_position: 9
---

# version

Git-like versioning for database diffs. Stores schema and data diff snapshots as commits so you can browse history, compare any two points in time, and generate rollback SQL — all without a live database connection after the commit is taken.

## Overview

```bash
deepdiffdb version <subcommand> [flags]
```

| Subcommand | Description |
|---|---|
| [`version init`](#version-init) | Initialise a `.deepdiffdb/` repository in the current directory |
| [`version commit`](#version-commit) | Run a full diff and store the result as a new commit |
| [`version log`](#version-log) | Show commit history from newest to oldest |
| [`version diff`](#version-diff) | Compare schema evolution between two commits |
| [`version rollback`](#version-rollback) | Generate rollback SQL to undo the changes in a commit |

---

## version init

Creates the `.deepdiffdb/` directory structure in the current directory. Idempotent — safe to run on an existing repo.

### Usage

```bash
deepdiffdb version init
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Directory to initialise |

### What it creates

```
.deepdiffdb/
  HEAD              ← hash of the latest commit (empty on a fresh repo)
  objects/          ← one JSON file per commit
```

### Example

```bash
cd my-project
deepdiffdb version init
# Initialised version repository in ./.deepdiffdb
```

---

## version commit

Connects to both databases, runs a full schema and data diff (identical to the `diff` command), and stores the result as a new commit object.

Each commit records:
- Author, message, timestamp
- The full `schema.DiffResult` and `content.DataDiff`
- Both `ProdSchema` and `DevSchema` snapshots (used for offline rollback generation)
- A SHA-256 hash derived from the metadata and diff content

### Usage

```bash
deepdiffdb version commit --message "describe the change" [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--config` | `deepdiffdb.config.yaml` | Path to configuration file |
| `--message` | _(required)_ | Commit message |
| `--author` | `$USER` | Author name |
| `--verbose` | `false` | Enable debug-level logging |
| `--log-file` | _(none)_ | Write logs to file |
| `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | `text` | Log format: `text` or `json` |

### Example

```bash
deepdiffdb version commit \
  --config deepdiffdb.config.yaml \
  --message "V2: add category_id FK and customer_email column" \
  --author "Alice"

# [e35c16c9] V2: add category_id FK and customer_email column
#   Schema drift detected.
#   Data differences detected.
```

### Commit object location

```
.deepdiffdb/objects/<sha256>.json
```

---

## version log

Walks the commit chain from `HEAD` backwards, printing each commit's hash, author, date, and message. Schema drift and data change markers are appended when present.

### Usage

```bash
deepdiffdb version log [--limit N]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--limit` | `20` | Maximum number of commits to display |

### Example output

```
commit 142b1305b83fa114aca634ae15b5d807552a443b94462154f470ba6011bc15ed
Author: Bob
Date:   2026-03-31 14:33:29 +0000

    V3: add reviews table and avg_rating column [schema drift] [data changes]

commit e35c16c95a306e52b2044f4beb3fd7de90ce5f46a5fd46487d8fbf11796020f1
Author: Alice
Date:   2026-03-31 14:33:22 +0000

    V2: add category_id FK and customer_email column [schema drift] [data changes]

commit cad39fda5cdefb3e18047e8615b542a6e39a7d3261432cc17cf535f59fa3591a
Author: Alice
Date:   2026-03-31 14:33:05 +0000

    V1: baseline e-commerce schema
```

---

## version diff

Compares the dev schema snapshots of two commits and reports the schema evolution between them — what tables and columns were added, removed, or modified.

### Usage

```bash
deepdiffdb version diff <hash1> <hash2>
```

Full hashes and short hashes (first 8+ characters) are both accepted.

### Example

```bash
deepdiffdb version diff cad39fda 142b1305
```

```
Schema evolution from cad39fda (V1: baseline e-commerce schema) → 142b1305 (V3: add reviews table and avg_rating column)

  + TABLE reviews (added)
  ~ TABLE orders
      + COLUMN customer_email varchar(255)
  ~ TABLE products
      + COLUMN avg_rating decimal(3,2)
      + COLUMN category_id int
      + INDEX fk_product_category
```

Legend:
- `+` — added (new table, column, or index)
- `-` — removed
- `~` — modified (table has changes inside)

---

## version rollback

Generates a SQL script that undoes the changes recorded in a commit. It inverts the stored diff (dev → prod direction) and passes it through the same migration generator used by `schema-migrate`, so all safety defaults apply:

- Destructive operations (`DROP TABLE`, `DROP COLUMN`, `DROP INDEX`) are **commented out** by default
- Output is wrapped in a `BEGIN` / `COMMIT` transaction
- Proper FK dependency ordering is applied

Requires no live database connection.

### Usage

```bash
# Print rollback SQL to stdout
deepdiffdb version rollback <hash>

# Write to file (flags must come before the hash)
deepdiffdb version rollback --out rollback.sql <hash>

# Override the SQL dialect
deepdiffdb version rollback --driver postgres <hash>
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--out` | _(stdout)_ | Write SQL to this file instead of printing |
| `--driver` | _(from commit)_ | Override the database driver (`mysql`, `postgres`, `sqlite`, `mssql`, `oracle`) |

:::note Flag ordering
Due to Go's standard flag parser, `--out` and `--driver` must appear **before** the positional hash argument.
:::

### Example

```bash
deepdiffdb version rollback --out rollback_v3.sql 142b1305
# Rollback SQL written to rollback_v3.sql
```

```sql
BEGIN;

-- Schema Migration Script
-- Generated by DeepDiff DB for mysql

-- DROP TABLES (present in prod but not in dev)
-- WARNING: These operations will delete data!
-- DROP TABLE `reviews`;

-- Table: orders
-- DROP COLUMNS (present in prod but not in dev)
-- ALTER TABLE `orders` DROP COLUMN `customer_email`;

-- Table: products
-- ALTER TABLE `products` DROP COLUMN `avg_rating`;
-- ALTER TABLE `products` DROP COLUMN `category_id`;

COMMIT;
```

Uncomment the statements you want to apply, review, and execute against the target database.

---

## Storage Layout

```
.deepdiffdb/
  HEAD                          ← current commit hash (or empty string)
  objects/
    <sha256>.json               ← one file per commit
```

Each commit file is self-contained JSON — it includes all diff results and schema snapshots, so the entire history is portable (commit the `.deepdiffdb/` directory to share with your team, or add it to `.gitignore` to keep it local).

---

## Typical Workflow

```bash
# 1. Initialise once
deepdiffdb version init

# 2. Commit baseline (prod == dev)
deepdiffdb version commit --message "V1: baseline" --author "Alice"

# 3. Developer applies schema changes to dev database
#    (ALTER TABLE, CREATE TABLE, etc.)

# 4. Commit the drift
deepdiffdb version commit --message "V2: add reviews table" --author "Bob"

# 5. Review history
deepdiffdb version log

# 6. See exactly what changed sprint-over-sprint
deepdiffdb version diff <hash_v1> <hash_v2>

# 7. If V2 needs to be rolled back, generate the SQL
deepdiffdb version rollback --out rollback_v2.sql <hash_v2>
```

---

## See Also

- [Sample 17: Git-like Versioning](https://github.com/iamvirul/deepdiff-db/tree/main/samples/17-git-like-versioning) — full end-to-end demo with MySQL containers
- [`diff`](./diff.md) — the underlying diff command used by `version commit`
- [`schema-migrate`](./schema-migrate.md) — standalone schema migration (same SQL generator as `version rollback`)
