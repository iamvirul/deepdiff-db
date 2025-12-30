# Sample 07: Table Operations (CREATE TABLE & DROP TABLE)

This sample demonstrates DeepDiff DB's ability to generate full `CREATE TABLE` and `DROP TABLE` statements when comparing databases with different table structures.

## Scenario

### Production Database (port 3311)
- `users` - User accounts
- `orders` - Customer orders
- `legacy_audit_log` - **Old audit system (to be removed)**
- `old_sessions` - **Legacy session storage (to be removed)**

### Development Database (port 3312)
- `users` - User accounts (unchanged)
- `orders` - Customer orders (unchanged)
- `audit_events` - **New modern audit system with indexes**
- `user_sessions` - **New JWT-based session storage with indexes**
- `feature_flags` - **New feature flag system**

## What You'll See

When running the schema migration, DeepDiff DB will:

1. **Generate CREATE TABLE statements** for new tables:
   - `audit_events` - Complete with column definitions, primary key, and indexes
   - `user_sessions` - Complete with column definitions, primary key, and indexes
   - `feature_flags` - Complete with column definitions and unique constraint

2. **Generate DROP TABLE statements** for removed tables:
   - `legacy_audit_log` - Commented out by default for safety
   - `old_sessions` - Commented out by default for safety

## Quick Start

### 1. Start the databases

```bash
cd samples/07-table-operations
docker-compose up -d

# Wait for databases to be ready
sleep 10
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

### Safe Mode (default)

```sql
-- DeepDiff DB Schema Migration
-- Generated at: ...
-- Driver: mysql

BEGIN;

-- ================================================================
-- DROP TABLES (commented for safety - enable with allow_drop_table: true)
-- ================================================================
-- WARNING: These operations will delete data!

-- DROP TABLE `legacy_audit_log`;
-- DROP TABLE `old_sessions`;

-- ================================================================
-- CREATE TABLES
-- ================================================================

CREATE TABLE `audit_events` (
  `id` INT NOT NULL,
  `event_type` VARCHAR(50) NOT NULL,
  `entity_type` VARCHAR(50) NOT NULL,
  `entity_id` INT NOT NULL,
  `user_id` INT,
  `payload` JSON,
  `created_at` TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX `idx_audit_entity` ON `audit_events` (`entity_type`, `entity_id`);
CREATE INDEX `idx_audit_user` ON `audit_events` (`user_id`);
CREATE INDEX `idx_audit_created` ON `audit_events` (`created_at`);

CREATE TABLE `feature_flags` (
  `id` INT NOT NULL,
  `name` VARCHAR(100) NOT NULL,
  `description` TEXT,
  `enabled` TINYINT(1),
  `rollout_percentage` INT,
  `created_at` TIMESTAMP,
  `updated_at` TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `user_sessions` (
  `id` INT NOT NULL,
  `user_id` INT NOT NULL,
  `refresh_token` VARCHAR(500) NOT NULL,
  `user_agent` VARCHAR(255),
  `ip_address` VARCHAR(45),
  `expires_at` TIMESTAMP NOT NULL,
  `created_at` TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX `idx_sessions_user` ON `user_sessions` (`user_id`);
CREATE INDEX `idx_sessions_token` ON `user_sessions` (`refresh_token`);

COMMIT;
```

### Active Mode (allow_drop_table: true)

The `DROP TABLE` statements are uncommented and executable:

```sql
-- ================================================================
-- DROP TABLES
-- ================================================================
-- WARNING: These operations will delete data!

DROP TABLE `legacy_audit_log`;
DROP TABLE `old_sessions`;
```

## Key Features Demonstrated

### CREATE TABLE Generation

- **Full column definitions**: data types, NOT NULL constraints, default values
- **Primary key constraints**: single and composite keys supported
- **Index creation**: indexes are created separately after the table
- **Driver-specific syntax**: MySQL uses backticks, PostgreSQL/SQLite use double quotes
- **Table options**: MySQL tables include `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

### DROP TABLE Safety

- **Commented by default**: DROP TABLE statements are commented to prevent accidental data loss
- **Warning message**: Clear warning about destructive operations
- **Explicit opt-in**: Set `allow_drop_table: true` to enable DROP TABLE statements
- **Ordered execution**: DROP TABLE statements appear before CREATE TABLE to avoid dependency issues

## Configuration Options

```yaml
migration:
  allow_drop_table: false  # Set to true to enable DROP TABLE statements
  allow_drop_column: false # Set to true to enable DROP COLUMN statements
  allow_drop_index: false  # Set to true to enable DROP INDEX statements
  confirm_destructive: true # Show extra warnings for destructive operations
```

## Database Driver Support

| Feature | MySQL | PostgreSQL | SQLite |
|---------|-------|------------|--------|
| CREATE TABLE | Yes | Yes | Yes |
| DROP TABLE | Yes | Yes | Yes |
| Primary Key | Yes | Yes | Yes |
| Indexes | Yes | Yes | Yes |
| Table Options | ENGINE, CHARSET | - | - |
| Identifier Quoting | Backticks | Double quotes | Double quotes |

## Files in This Sample

- `docker-compose.yml` - MySQL containers for prod and dev databases
- `deepdiffdb.config.yaml` - Safe mode configuration (DROP commented)
- `deepdiffdb-active.config.yaml` - Active mode (DROP enabled)
- `init-scripts/01-prod-schema.sql` - Production schema with legacy tables
- `init-scripts/02-dev-schema.sql` - Development schema with new tables
- `diff-output/` - Directory for generated output files
