# Sample 08: Foreign Key & Primary Key Support

This sample demonstrates DeepDiff DB's ability to detect and generate migration statements for foreign key constraints and primary key modifications.

## Scenario

### Production Database (port 3313)
- `users` - User accounts
- `categories` - Product categories (self-referencing FK)
- `products` - Products with FK to categories
- `orders` - Orders with FK to users (`fk_orders_user_legacy` with RESTRICT)
- `order_items` - Order items with FKs to orders and products

### Development Database (port 3314)
- `users` - User accounts (unchanged)
- `categories` - Categories (unchanged FK)
- `products` - Products (unchanged FK)
- `orders` - **Modified FK**: `fk_orders_user` with CASCADE (was RESTRICT)
- `order_items` - Order items (unchanged FKs)
- `reviews` - **New table** with FKs to users and products
- `shipping_addresses` - **New table** with FK to users

## What You'll See

When running the schema migration, DeepDiff DB will:

1. **Detect removed foreign key**:
   - `fk_orders_user_legacy` (in prod, not in dev)

2. **Detect added foreign keys**:
   - `fk_orders_user` (new FK with CASCADE)
   - `fk_reviews_user` (on new table)
   - `fk_reviews_product` (on new table)
   - `fk_shipping_user` (on new table)

3. **Generate migration SQL**:
   - DROP FOREIGN KEY statements (commented by default)
   - ADD FOREIGN KEY statements with ON DELETE/UPDATE actions

## Quick Start

### 1. Start the databases

```bash
cd samples/08-foreign-key-support
docker-compose up -d

# Wait for databases to be ready
sleep 15
```

### 2. Check connectivity

```bash
deepdiffdb check --config deepdiffdb.config.yaml
```

### 3. Run schema diff

```bash
deepdiffdb schema-diff --config deepdiffdb.config.yaml
```

### 4. Generate migration (Safe Mode - DROP commented)

```bash
deepdiffdb schema-migrate --config deepdiffdb.config.yaml
cat diff-output/schema_migration.sql
```

### 5. Generate migration (Active Mode - DROP enabled)

```bash
deepdiffdb schema-migrate --config deepdiffdb-active.config.yaml
cat diff-output/schema_migration.sql
```

### 6. Cleanup

```bash
docker-compose down -v
```

## Expected Output

### Foreign Key Changes

```
Table: orders
  - foreign key fk_orders_user_legacy missing in dev
  - foreign key fk_orders_user missing in prod

Table: reviews
  - present in dev, missing in prod
  - (includes foreign keys: fk_reviews_user, fk_reviews_product)

Table: shipping_addresses
  - present in dev, missing in prod
  - (includes foreign key: fk_shipping_user)
```

### Generated SQL (Safe Mode)

```sql
-- Table: orders
-- DROP FOREIGN KEYS (present in prod but not in dev)
-- WARNING: Dropping foreign keys removes referential integrity constraints
-- IMPORTANT: Review carefully before executing!
-- ALTER TABLE `orders` DROP FOREIGN KEY `fk_orders_user_legacy`;

-- ADD FOREIGN KEYS (present in dev but not in prod)
ALTER TABLE `orders` ADD CONSTRAINT `fk_orders_user` FOREIGN KEY (`user_id`)
  REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE;
```

### Generated SQL (Active Mode)

With `allow_drop_foreign_key: true`:

```sql
-- DROP FOREIGN KEYS (present in prod but not in dev)
ALTER TABLE `orders` DROP FOREIGN KEY `fk_orders_user_legacy`;
```

## Configuration Options

```yaml
migration:
  allow_drop_foreign_key: false    # Set to true to enable DROP FOREIGN KEY
  allow_modify_primary_key: false  # Set to true to enable PRIMARY KEY modifications
  confirm_destructive: true        # Show extra warnings
```

## Database Driver Support

| Feature | MySQL | PostgreSQL | SQLite |
|---------|-------|------------|--------|
| ADD FOREIGN KEY | Yes | Yes | Limited* |
| DROP FOREIGN KEY | Yes | Yes | Limited* |
| ON DELETE actions | Yes | Yes | Yes |
| ON UPDATE actions | Yes | Yes | Yes |
| Composite FKs | Yes | Yes | Yes |

*SQLite requires table recreation to modify foreign keys.

## SQL Syntax by Driver

### MySQL
```sql
ALTER TABLE `orders` ADD CONSTRAINT `fk_orders_user`
  FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
  ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `orders` DROP FOREIGN KEY `fk_orders_user`;
```

### PostgreSQL
```sql
ALTER TABLE "orders" ADD CONSTRAINT "fk_orders_user"
  FOREIGN KEY ("user_id") REFERENCES "users" ("id")
  ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "orders" DROP CONSTRAINT "fk_orders_user";
```

### SQLite
```sql
-- SQLite limitation: Cannot add foreign key after table creation.
-- Table recreation required.
```

## Foreign Key Actions

| Action | Description |
|--------|-------------|
| CASCADE | Delete/update child rows when parent changes |
| SET NULL | Set foreign key to NULL when parent deleted |
| RESTRICT | Prevent deletion if child rows exist |
| NO ACTION | Default, similar to RESTRICT |
| SET DEFAULT | Set to default value (MySQL only) |

## Safety Features

- **DROP FOREIGN KEY commented by default**: Prevents accidental removal
- **Warning messages**: Clear warnings about constraint removal
- **Explicit opt-in**: Requires `allow_drop_foreign_key: true`
- **Dependency ordering**: FKs dropped before related operations

## Files in This Sample

- `docker-compose.yml` - MySQL containers for prod and dev databases
- `deepdiffdb.config.yaml` - Safe mode configuration (DROP commented)
- `deepdiffdb-active.config.yaml` - Active mode (DROP enabled)
- `init-scripts/01-prod-schema.sql` - Production schema with existing FKs
- `init-scripts/02-dev-schema.sql` - Development schema with new FKs
- `diff-output/` - Directory for generated output files
