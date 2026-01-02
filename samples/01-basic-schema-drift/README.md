# Sample 01: Basic Schema Drift Detection

This sample demonstrates the fundamental schema drift detection capabilities of DeepDiff DB. It shows how to identify structural differences between two databases using the `schema-diff` command.

## What You'll Learn

- How to set up and configure DeepDiff DB for schema comparison
- How to detect schema drift between two databases
- How to interpret schema diff reports
- Basic workflow for identifying structural differences

## Scenario

This sample simulates a common scenario where:

- **Production Database (db1)**: Contains a `users` table with columns: `id`, `name`, `email`, `created_at`
- **Development Database (db2)**: Contains a `users` table with an additional column: `country`

The development team has added a new column to the schema, and you need to detect this difference before applying changes to production.

## Prerequisites

- Docker and Docker Compose installed
- DeepDiff DB installed locally
- Basic understanding of database schemas

## Setup

### 1. Start the Databases

```bash
cd samples/01-basic-schema-drift
docker-compose up -d
```

This starts two MySQL databases:
- **Production DB (db1)**: Port 3308 - Contains the original schema
- **Development DB (db2)**: Port 3307 - Contains the modified schema with additional column

Wait for the databases to be ready (about 10-15 seconds):

```bash
# Check if databases are ready
docker-compose logs -f
# Press Ctrl+C when you see "ready for connections"
```

### 2. Verify Database Schemas

**Production Database (db1):**
```bash
docker exec -it mysql_db1 mysql -uroot -prootpassword -e "DESCRIBE testdb.users"
```

Expected output:
```
+------------+-------------+------+-----+-------------------+-------------------+
| Field      | Type        | Null | Key | Default           | Extra             |
+------------+-------------+------+-----+-------------------+-------------------+
| id         | int         | NO   | PRI | NULL              | auto_increment    |
| name       | varchar(50) | NO   |     | NULL              |                   |
| email      | varchar(50) | NO   | UNI | NULL              |                   |
| created_at | timestamp   | YES  |     | CURRENT_TIMESTAMP | DEFAULT_GENERATED |
+------------+-------------+------+-----+-------------------+-------------------+
```

**Development Database (db2):**
```bash
docker exec -it mysql_db2 mysql -uroot -prootpassword -e "DESCRIBE testdb.users"
```

Expected output:
```
+------------+-------------+------+-----+-------------------+-------------------+
| Field      | Type        | Null | Key | Default           | Extra             |
+------------+-------------+------+-----+-------------------+-------------------+
| id         | int         | NO   | PRI | NULL              | auto_increment    |
| name       | varchar(50) | NO   |     | NULL              |                   |
| email      | varchar(50) | NO   | UNI | NULL              |                   |
| created_at | timestamp   | YES  |     | CURRENT_TIMESTAMP | DEFAULT_GENERATED |
| country    | varchar(50) | YES  |     | NULL              |                   |
+------------+-------------+------+-----+-------------------+-------------------+
```

Notice that the development database has an additional `country` column.

## Usage Examples

### Example 1: Validate Configuration

Before running schema diff, validate your configuration:

```bash
deepdiffdb check --config deepdiffdb.config.yaml
```

Expected output:
```
Config loaded.
Prod: 127.0.0.1:3308/testdb
Dev : 127.0.0.1:3307/testdb
Output directory ready: ./diff-output
Ignore tables: []
Ignore columns: []
Connections OK. Primary keys verified.
```

### Example 2: Detect Schema Drift

Run the schema diff command to detect differences:

```bash
deepdiffdb schema-diff --config deepdiffdb.config.yaml
```

**Expected Output:**
```
Schema drift detected; see ./diff-output/schema_diff.json and ./diff-output/schema_diff.txt
```

The command will exit with an error code because schema drift was detected. This is the expected behavior - schema drift detection is designed to fail when differences are found, making it suitable for CI/CD pipelines.

### Example 3: Review Schema Diff Reports

The command generates two report files in the `diff-output/` directory:

**Text Report** (`schema_diff.txt`):
```
Table: users
  - column country missing in prod (dev type=varchar nullable=true)
```

**JSON Report** (`schema_diff.json`):
```json
{
  "tables": [
    {
      "table": "users",
      "column_diffs": [
        {
          "column": "country",
          "missing_in_prod": true,
          "dev_type": "varchar",
          "dev_nullable": true
        }
      ],
      "has_differences": true
    }
  ]
}
```

## Understanding the Output

### Schema Diff Text Report

The text report provides a human-readable summary:
- Lists each table with differences
- Shows which columns are missing or different
- Indicates the column type and nullable status from the development database

### Schema Diff JSON Report

The JSON report provides machine-readable details:
- Structured format suitable for programmatic processing
- Detailed information about each difference
- Can be integrated into CI/CD pipelines for automated checks

## Common Use Cases

### CI/CD Integration

Schema drift detection is ideal for continuous integration:

```bash
# In your CI pipeline
deepdiffdb schema-diff --config deepdiffdb.config.yaml

# If exit code is 0, schemas match
# If exit code is non-zero, schemas differ (block deployment)
```

### Pre-Deployment Validation

Check for schema drift before deploying:

```bash
# Before deploying to production
deepdiffdb schema-diff --config deepdiffdb.config.yaml

# Review the reports
cat diff-output/schema_diff.txt

# If drift detected, investigate and resolve before proceeding
```

### Development Workflow

Regularly check for schema differences:

```bash
# Daily or weekly schema drift check
deepdiffdb schema-diff --config deepdiffdb.config.yaml

# Review changes
cat diff-output/schema_diff.txt
```

## Configuration

The configuration file (`deepdiffdb.config.yaml`) defines:

```yaml
prod:
  driver: "mysql"
  host: "127.0.0.1"
  port: 3308          # Production database port
  user: "root"
  password: "rootpassword"
  database: "testdb"

dev:
  driver: "mysql"
  host: "127.0.0.1"
  port: 3307          # Development database port
  user: "root"
  password: "rootpassword"
  database: "testdb"

ignore:
  tables: []          # No tables ignored
  columns: []         # No columns ignored

output:
  dir: "./diff-output"  # Output directory for reports
```

## What Gets Detected

The `schema-diff` command detects:

- **Added columns** - Columns present in dev but not in prod
- **Removed columns** - Columns present in prod but not in dev
- **Modified columns** - Columns with different types or constraints
- **Added tables** - Tables present in dev but not in prod
- **Removed tables** - Tables present in prod but not in dev

In this sample, we're detecting an **added column** (`country` in the `users` table).

## Next Steps

After detecting schema drift, you typically want to:

1. **Review the differences** - Understand what changed
2. **Generate migration script** - Use `schema-migrate` to create SQL migration
3. **Test the migration** - Apply to a test database first
4. **Apply to production** - After validation, apply the migration

See [Sample 03: Schema Migrations](../03-schema-migrations/) for generating and applying schema migrations.

## Troubleshooting

**Problem**: Command fails with "connection refused"

**Solution**: Ensure Docker containers are running:
```bash
docker-compose ps
docker-compose up -d
```

**Problem**: "Primary keys verified" error

**Solution**: Ensure all tables have primary keys. This sample includes primary keys, so this shouldn't occur.

**Problem**: No differences detected when differences exist

**Solution**: 
- Verify database connections are correct
- Check that you're comparing the right databases
- Ensure tables aren't in the ignore list

## Cleanup

When you're done experimenting:

```bash
# Stop and remove containers
docker-compose down

# Remove volumes (WARNING: This deletes all data)
docker-compose down -v
```

## Files in This Sample

- `README.md` - This file
- `docker-compose.yml` - Database containers configuration
- `deepdiffdb.config.yaml` - DeepDiff DB configuration file
- `init-scripts/init_db1.sql` - Production database schema and data
- `init-scripts/init_db2.sql` - Development database schema and data
- `diff-output/` - Generated schema diff reports (created when you run the tool)

## Learn More

- [DeepDiff DB Documentation](../../README.md)
- [Sample 02: Advanced Migrations](../02-advanced-migrations/) - Full workflow with data diffing
- [Sample 03: Schema Migrations](../03-schema-migrations/) - Generating schema migration scripts
- [Sample 04: DROP COLUMN Safety](../04-drop-column-safety/) - Safe column removal
- [Sample 10: Conflict Resolution](../10-conflict-resolution/) - Configuring conflict resolution strategies
- [Sample 12: Interactive Resolution](../12-interactive-resolution/) - Interactive CLI for conflict resolution

---

**DeepDiff DB - Smart Database Comparison Tool**

