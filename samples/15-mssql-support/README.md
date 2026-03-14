# Sample 15: MSSQL Support

This sample demonstrates **Microsoft SQL Server support** introduced in v0.8. It shows how to compare two SQL Server databases — detecting schema drift and data differences — and generate a migration pack with MSSQL-compatible SQL.

## What You'll Learn

- Configuring `driver: "mssql"` in `deepdiffdb.config.yaml`
- Connecting to SQL Server using the `sqlserver://` DSN format
- MSSQL-specific schema introspection via `INFORMATION_SCHEMA` and `sys.*` catalog views
- MSSQL square-bracket identifier quoting (`[table]`, `[column]`)
- MSSQL `OFFSET/FETCH` pagination (replaces MySQL/PostgreSQL `LIMIT`)
- How DeepDiff DB generates MSSQL-compatible `ALTER TABLE` / `DROP INDEX … ON …` SQL
- FK control using `sp_msforeachtable` during pack application

## Scenario

Two SQL Server 2022 containers are started: **prod** (port 1433) and **dev** (port 1434).

The development database has deliberate drift from production:

### Schema drift
| Table | Change |
|---|---|
| `customers` | New column `phone NVARCHAR(30) NULL` added in dev |
| `products` | `category` changed from `NULL` → `NOT NULL` in dev; new index `IX_products_category` added |
| `orders` | New column `notes NVARCHAR(500) NULL` added; `status` default changed from `'pending'` → `'new'` |

### Data drift
| Table | Change |
|---|---|
| `products` | `GADGET-001` price updated: `$49.99` → `$54.99` |
| `orders` | Order 3 status changed `'shipped'` → `'completed'`; 2 new orders added |
| `customers` | 1 new customer (`Frank Brown`) added |

## Prerequisites

- Docker and Docker Compose
- `deepdiffdb` installed and on your `$PATH`
- `sqlcmd` (optional — the Makefile uses the tool inside the container)

## Quick Start

```bash
cd samples/15-mssql-support

# 1. Start containers, wait until healthy, and seed both databases
make seed

# 2. Run the full diff (schema + content)
make diff

# 3. Inspect the output
cat diff-output/diff_report.txt

# 4. Generate a migration pack
make gen-pack
```

## Configuration

`deepdiffdb.config.yaml` uses `driver: "mssql"` for both connections:

```yaml
prod:
  driver: "mssql"
  host: "127.0.0.1"
  port: 1433
  user: "sa"
  password: "StrongP@ss1word!"
  database: "prod_db"

dev:
  driver: "mssql"
  host: "127.0.0.1"
  port: 1434
  user: "sa"
  password: "StrongP@ss1word!"
  database: "dev_db"
```

Key points:
- `driver: "mssql"` maps internally to the `sqlserver://` Go driver — no additional setup required.
- Port is optional when using the SQL Server default (1433).
- The `sa` account is used here for simplicity; use a least-privilege account in production.
- `ordered_at` and `created_at` are ignored to suppress timestamp noise from clock drift between containers.

## MSSQL-Specific SQL in the Generated Pack

DeepDiff DB generates MSSQL-compatible SQL throughout:

**Identifier quoting:**
```sql
SELECT [id], [name], [email] FROM [customers]
```

**Column addition (no `COLUMN` keyword):**
```sql
ALTER TABLE [customers] ADD [phone] NVARCHAR(30) NULL;
```

**Column type/nullability change:**
```sql
ALTER TABLE [products] ALTER COLUMN [category] NVARCHAR(100) NOT NULL;
```

**Index drop (requires table name):**
```sql
DROP INDEX [IX_products_category] ON [products];
```

**Foreign key constraint drop:**
```sql
ALTER TABLE [orders] DROP CONSTRAINT [FK_orders_customer];
```

**FK control during pack application:**
```sql
EXEC sp_msforeachtable 'ALTER TABLE ? NOCHECK CONSTRAINT ALL';
-- ... data changes ...
EXEC sp_msforeachtable 'ALTER TABLE ? WITH CHECK CHECK CONSTRAINT ALL';
```

**Pagination:**
```sql
SELECT [id], [name] FROM [customers]
ORDER BY [id]
OFFSET 0 ROWS FETCH NEXT 10000 ROWS ONLY;
```

## Expected Output

After running `make diff` you should see output similar to:

```
Comparing prod_db (mssql) ↔ dev_db (mssql)

Schema diff:
  customers  : 1 column added (phone)
  products   : 1 column modified (category: nullable → NOT NULL), 1 index added
  orders     : 2 columns added (notes, status default change), no structural removals

Content diff:
  customers  : 1 row added
  products   : 1 row modified (GADGET-001 price)
  orders     : 1 row modified, 2 rows added

Reports written to ./diff-output/
```

## Running Individual Diff Stages

```bash
# Schema only — faster, no row hashing
make diff-schema

# Content only — skips schema comparison
make diff-content

# Generate migration pack
make gen-pack
```

## Migration Pack

Running `make gen-pack` produces a ZIP archive in `diff-output/` containing:

| File | Purpose |
|---|---|
| `schema_migrate.sql` | `ALTER TABLE` / `DROP INDEX` / `ADD CONSTRAINT` statements |
| `data_inserts.sql` | `INSERT` for rows present only in dev |
| `data_updates.sql` | `UPDATE` for rows that differ between prod and dev |
| `manifest.json` | Machine-readable summary of all changes |

All SQL uses MSSQL-compatible syntax — square-bracket quoting, `OFFSET/FETCH` pagination, no MySQL `LIMIT`, and proper `ALTER COLUMN` instead of `MODIFY COLUMN`.

## Cleanup

```bash
make clean   # stops containers, removes volumes, and deletes generated reports
```

## Files in This Sample

```
15-mssql-support/
  README.md                   — this file
  deepdiffdb.config.yaml      — mssql driver config pointing to both containers
  docker-compose.yml          — two SQL Server 2022 (Express) containers
  Makefile                    — convenience targets: up, seed, diff, gen-pack, clean
  init-scripts/
    01-prod-schema.sql        — production tables + seed data
    02-dev-schema.sql         — development tables with deliberate drift + seed data
  diff-output/                — generated reports (created on first diff run)
```

## Security Note

The sample uses the `sa` account with a demo password. **Never use `sa` or hardcoded passwords in production.** Use a dedicated low-privilege SQL Server login with `SELECT`, `INSERT`, `UPDATE`, `DELETE` on the target database only.

## Learn More

- [Sample 01: Basic Schema Drift](../01-basic-schema-drift/) — introduction to DeepDiff DB
- [Sample 08: Foreign Key Support](../08-foreign-key-support/) — FK-aware migration ordering
- [Sample 14: Streaming Large Datasets](../14-streaming-large-datasets/) — keyset-paginated batch hashing
- [DeepDiff DB README](../../README.md) — full documentation
