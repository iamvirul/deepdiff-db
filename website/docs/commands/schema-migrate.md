---
sidebar_position: 4
---

# schema-migrate

Generates a standalone, transaction-wrapped SQL migration script based on the detected schema differences. The script is ready to review and apply manually or through your existing deployment pipeline.

## Usage

```bash
deepdiffdb schema-migrate --config deepdiffdb.config.yaml
```

## What It Generates

The output SQL covers every structural change detected between dev and prod:

- `ALTER TABLE ... ADD COLUMN` — for new columns in dev
- `ALTER TABLE ... MODIFY` / `ALTER COLUMN` — for columns whose type, nullability, or default changed
- `ALTER TABLE ... DROP COLUMN` — for columns removed in dev (blocked by default)
- `CREATE INDEX` / `CREATE UNIQUE INDEX` — for new indexes
- `DROP INDEX` — for removed indexes (blocked by default)
- `ALTER TABLE ... ADD CONSTRAINT ... FOREIGN KEY` — for new FKs
- `ALTER TABLE ... DROP FOREIGN KEY` / `DROP CONSTRAINT` — for removed FKs (blocked by default)
- `CREATE TABLE` / `DROP TABLE` — for added/removed tables (`DROP TABLE` blocked by default)

### Safety Defaults

Destructive operations are **blocked by default**. When a blocked operation is encountered, the statement is **commented out** in the generated SQL with a warning. Enable them in `deepdiffdb.config.yaml`:

```yaml
migration:
  allow_drop_column: true
  allow_drop_table: true
  allow_drop_index: true
  allow_drop_foreign_key: true
  allow_modify_primary_key: true
```

### Driver-Specific DDL

Each database driver generates the correct dialect:

| Operation | MySQL | PostgreSQL | SQLite | MSSQL | Oracle |
|---|---|---|---|---|---|
| Add column | `ADD COLUMN` | `ADD COLUMN` | `ADD COLUMN` | `ADD` | `ADD` |
| Modify column | `MODIFY COLUMN` | `ALTER COLUMN ... TYPE` | _(not supported)_ | `ALTER COLUMN` | `MODIFY` |
| Drop index | `DROP INDEX ... ON table` | `DROP INDEX` | `DROP INDEX` | `DROP INDEX ... ON table` | `DROP INDEX` (standalone) |

## Flags

| Flag | Description |
|---|---|
| `--config` | Path to the configuration file (default: `deepdiffdb.config.yaml`) |
| `--dry-run` | Print the generated SQL to stdout without writing to disk |
| `--verbose` | Enable debug-level logging |
| `--log-level` | Minimum log level (default: `info`) |
| `--log-format` | Log output format: `text` or `json` (default: `text`) |
| `--log-file` | Write logs to this file in addition to stdout |

## Output

| File | Description |
|---|---|
| `output/schema_migration.sql` | Transaction-wrapped DDL migration script |

## Example

```bash
deepdiffdb schema-migrate --config deepdiffdb.config.yaml --dry-run
```

Example output:

```sql
BEGIN;

-- Table: orders
ALTER TABLE `orders` MODIFY COLUMN `status` VARCHAR(50) NOT NULL DEFAULT 'new';
ALTER TABLE `orders` ADD COLUMN `shipped_at` TIMESTAMP NULL;

-- Table: products
CREATE INDEX `idx_products_category` ON `products` (`category_id`);

-- Table: invoices (ADDED)
CREATE TABLE `invoices` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `order_id` INT NOT NULL,
  `total` DECIMAL(10,2) NOT NULL,
  PRIMARY KEY (`id`)
);

COMMIT;
```

## Notes

- The migration script targets the **production** database — it transforms prod to match dev.
- Always review the generated SQL before applying it.
- For data changes (row-level), use `gen-pack` instead of `schema-migrate`.
