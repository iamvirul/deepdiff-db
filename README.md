# DeepDiff DB

[![codecov](https://codecov.io/gh/iamvirul/deepdiff-db/branch/main/graph/badge.svg?token=Y9IORTUBAH)](https://codecov.io/gh/iamvirul/deepdiff-db)
[![Go Report Card](https://goreportcard.com/badge/github.com/iamvirul/deepdiff-db)](https://goreportcard.com/report/github.com/iamvirul/deepdiff-db)

DeepDiff DB is a high-performance Go CLI tool designed for comparing two databases, detecting schema drift, identifying data-level differences, and generating safe migration packs that can be applied to production environments without risking data corruption.

## Overview

DeepDiff DB addresses a common challenge in database management: development backups that drift away from production databases. When developers restore backups, modify schemas, update reference tables, change configurations, or manipulate data, attempting to push these changes directly to production often results in data loss or corruption.

DeepDiff DB makes the entire process deterministic, reviewable, and safe by:

- Performing structural validation before any data operations
- Detecting and reporting schema differences
- Identifying row-level data changes using cryptographic hashing
- Generating reviewable SQL migration scripts
- Supporting transactional application of changes
- Providing comprehensive conflict detection and reporting

## Features

### Core Capabilities

- **Fast Go-based diff engine** - Optimized for performance with efficient memory usage
- **Single static binary** - Zero dependencies after download, works on any compatible system
- **Multi-database support** - MySQL, PostgreSQL, and SQLite
- **Schema drift detection** - Identifies structural differences between databases
- **Row-level comparison** - SHA-256 hashing for accurate change detection
- **Conflict detection** - Identifies rows that exist in both databases but differ
- **Auto-generated SQL migration packs** - Production-ready migration scripts
- **Dry-run mode** - Validate migrations without executing them
- **Fully transactional apply mode** - All changes applied atomically
- **Comprehensive reporting** - JSON and human-readable text reports
- **Configurable ignore lists** - Exclude tables and columns from comparison
- **Flexible input sources** - Works with database connections or dump files

### Safety Features

- Primary key validation for all tables
- Transaction-wrapped migrations
- Destructive operation warnings
- Schema drift blocking (configurable)
- Conflict reporting before application
- Dry-run validation mode

## Installation

### Option 1: Download Precompiled Binaries (Recommended)

Precompiled binaries are available for the following platforms:

- Linux (x64, ARM64)
- macOS (Intel, Apple Silicon)
- Windows (x64)

Download the latest release from the [GitHub Releases](https://github.com/iamvirul/deepdiff-db/releases) page.

**Linux Example:**
```bash
wget https://github.com/iamvirul/deepdiffdb/releases/download/v1.0.0/deepdiffdb-linux-amd64
chmod +x deepdiffdb-linux-amd64
sudo mv deepdiffdb-linux-amd64 /usr/local/bin/deepdiffdb
```

**macOS Example (Apple Silicon):**
```bash
wget https://github.com/iamvirul/deepdiffdb/releases/download/v1.0.0/deepdiffdb-darwin-arm64
chmod +x deepdiffdb-darwin-arm64
sudo mv deepdiffdb-darwin-arm64 /usr/local/bin/deepdiffdb
```

**Windows Example:**
```powershell
# Download deepdiffdb-windows-amd64.exe and place in your PATH
```

### Option 2: Build from Source

**Using Go install:**
```bash
go install github.com/iamvirul/deepdiff-db/cmd/deepdiffdb@latest
```

This installs the binary to `$GOPATH/bin` or `$GOBIN`.

**Local Development Build:**

For developers who want to test the latest changes:

**macOS/Linux:**
```bash
# Build and install to ~/bin (no sudo required)
./scripts/build-local.sh --install --install-dir ~/bin

# Or install to /usr/local/bin (requires sudo)
sudo ./scripts/build-local.sh --install

# Or just build without installing (outputs to bin/deepdiffdb)
./scripts/build-local.sh --build-only
```

**Windows (PowerShell):**
```powershell
# Build and install
.\scripts\build-local.ps1 -Install

# Or just build
.\scripts\build-local.ps1 -BuildOnly
```

**Note:** Ensure `~/bin` is in your PATH:
```bash
export PATH="$HOME/bin:$PATH"  # Add to ~/.zshrc or ~/.bashrc
```

See [scripts/README.md](scripts/README.md) for detailed build options, examples, and troubleshooting.

### Option 3: Build All Platform Binaries

For maintainers who need to build binaries for all platforms:

```bash
make build-all
```

This generates binaries for all supported platforms in the `bin/` directory.

## Configuration

DeepDiff DB uses a YAML configuration file to define database connections and behavior settings.

### Configuration File Structure

Create a `deepdiffdb.config.yaml` file:

```yaml
prod:
  driver: "mysql"          # mysql, postgres, postgresql, or sqlite
  host: "localhost"
  port: 3306
  user: "root"
  password: "password"
  database: "prod_db"

dev:
  driver: "mysql"
  host: "localhost"
  port: 3306
  user: "root"
  password: "password"
  database: "dev_db"

ignore:
  tables:
    - "logs"
    - "audit"
  columns:
    - "*.updated_at"       # Pattern matching supported
    - "users.last_login"

output:
  dir: "./diff-output"

migration:
  allow_drop_column: false
  allow_drop_table: false
  allow_drop_index: false
  allow_drop_foreign_key: false
  allow_modify_primary_key: false
  confirm_destructive: false
```

### Configuration Options

**Database Configuration:**
- `driver`: Database driver (`mysql`, `postgres`, `postgresql`, or `sqlite`)
- `host`: Database hostname or IP address
- `port`: Database port number (not required for SQLite)
- `user`: Database username
- `password`: Database password
- `database`: Database name

**Ignore Configuration:**
- `tables`: List of table names to exclude from comparison
- `columns`: List of column patterns to exclude (supports wildcards like `*.updated_at`)

**Output Configuration:**
- `dir`: Directory path for generated reports and migration files (default: `./diff-output`)

**Migration Configuration:**
- `allow_drop_column`: Enable DROP COLUMN statements (default: false)
- `allow_drop_table`: Enable DROP TABLE statements (default: false)
- `allow_drop_index`: Enable DROP INDEX statements (default: false)
- `allow_drop_foreign_key`: Enable DROP FOREIGN KEY statements (default: false)
- `allow_modify_primary_key`: Enable PRIMARY KEY modification statements (default: false)
- `confirm_destructive`: Require confirmation for destructive operations (default: false)

An example configuration file is included at `deepdiffdb.config.yaml.example`.

## Commands

### check

Validates configuration and database connectivity.

```bash
deepdiffdb check --config deepdiffdb.config.yaml
```

This command:
- Validates the configuration file
- Tests connectivity to both databases
- Verifies that all non-ignored tables have primary keys
- Ensures the output directory is writable
- Displays a summary of configuration settings

### schema-diff

Detects and reports schema differences between production and development databases.

```bash
deepdiffdb schema-diff --config deepdiffdb.config.yaml
```

**Output Files:**
- `schema_diff.json` - Machine-readable schema differences
- `schema_diff.txt` - Human-readable schema diff report

**Behavior:**
- Exits with an error if schema drift is detected
- Useful for CI/CD pipelines to block deployments with schema mismatches

### schema-migrate

Generates a standalone schema migration script based on detected schema differences.

```bash
deepdiffdb schema-migrate --config deepdiffdb.config.yaml
```

**Output Files:**
- `schema_migration.sql` - Transaction-wrapped SQL migration script

**Features:**
- Driver-specific SQL syntax (MySQL, PostgreSQL, SQLite)
- Safe defaults (destructive operations commented out by default)
- Proper dependency ordering for foreign keys and constraints
- Transaction-wrapped for atomic execution

**Dry Run Mode:**
```bash
deepdiffdb schema-migrate --config deepdiffdb.config.yaml --dry-run
```

Validates and displays the migration without writing to disk.

### diff

Performs a full comparison of both schema and data.

```bash
deepdiffdb diff --config deepdiffdb.config.yaml
```

**Output Files:**
- `schema_diff.json` - Schema differences
- `schema_diff.txt` - Human-readable schema report
- `content_diff.json` - Data differences with added/removed/updated rows
- `conflicts.json` - Rows that exist in both databases but differ
- `summary.txt` - High-level summary with statistics

**Behavior:**
- Stops if schema drift is detected (unless using `gen-pack`)
- Provides comprehensive analysis of all differences

### gen-pack

Generates a SQL migration pack for data differences.

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml
```

**Output Files:**
- `schema_diff.json` - Schema differences (warnings only)
- `content_diff.json` - Data differences
- `conflicts.json` - Conflict details
- `summary.txt` - Summary statistics
- `migration_pack.sql` - Combined migration script

**Features:**
- Continues even with schema drift (with warnings)
- Only processes tables with matching schemas
- Batches updates for performance (1000 rows per batch)
- Handles foreign key constraints appropriately
- Includes both schema and data changes

### apply

Applies a migration pack to the production database.

```bash
deepdiffdb apply --pack migration_pack.sql --config deepdiffdb.config.yaml
```

**Features:**
- Fully transactional execution
- Atomic application (all or nothing)
- Automatic rollback on error
- Applies to the production database specified in config

**Dry Run Mode:**
```bash
deepdiffdb apply --pack migration_pack.sql --dry-run
```

Validates the SQL without executing it.

## Usage Examples

### Basic Workflow

1. **Validate Configuration:**
   ```bash
   deepdiffdb check --config deepdiffdb.config.yaml
   ```

2. **Check for Schema Drift:**
   ```bash
   deepdiffdb schema-diff --config deepdiffdb.config.yaml
   ```

3. **Generate Schema Migration (if needed):**
   ```bash
   deepdiffdb schema-migrate --config deepdiffdb.config.yaml
   ```

4. **Compare Data:**
   ```bash
   deepdiffdb diff --config deepdiffdb.config.yaml
   ```

5. **Generate Migration Pack:**
   ```bash
   deepdiffdb gen-pack --config deepdiffdb.config.yaml
   ```

6. **Review Generated Files:**
   - Check `conflicts.json` for any conflicts
   - Review `migration_pack.sql` for the proposed changes

7. **Apply Migration (after review):**
   ```bash
   deepdiffdb apply --pack diff-output/migration_pack.sql --config deepdiffdb.config.yaml
   ```

### Sample Output

**summary.txt:**
```
Schema: OK
Tables scanned: 12
Added rows: 18
Updated rows: 4
Conflicts: 2
Migration pack: migration_pack.sql
```

## How It Works

DeepDiff DB uses a multi-stage approach to ensure safe and accurate database synchronization:

1. **Schema Introspection** - Extracts metadata using database-specific information schema queries
2. **Schema Normalization** - Builds a normalized schema model for comparison
3. **Schema Comparison** - Identifies structural differences (tables, columns, indexes, constraints)
4. **Data Hashing** - Computes SHA-256 hashes for each row (excluding ignored columns)
5. **Hash Comparison** - Compares hash maps to identify added, removed, and modified rows
6. **Conflict Detection** - Identifies rows that exist in both databases but differ
7. **Migration Generation** - Creates SQL migration scripts with proper ordering and batching
8. **Transactional Application** - Applies changes within a single transaction for atomicity

The tool processes data in chunks for large tables and provides progress indicators for operations exceeding 10,000 rows.

## Architecture

### Project Structure

```
deepdiffdb/
├── cmd/
│   └── deepdiffdb/          # CLI entry point
├── internal/
│   ├── schema/              # Schema introspection and comparison
│   ├── content/             # Data hashing, diff, and migration generation
│   ├── drivers/             # Database driver abstraction
│   └── report/              # Report generation utilities
├── pkg/
│   └── config/              # Configuration loading and validation
├── samples/                 # Example configurations and use cases
├── tests/                   # Test suite
└── scripts/                 # Build and development scripts
```

### Core Components

- **CLI Layer** (`cmd/deepdiffdb/`) - Command dispatch and argument parsing
- **Configuration Layer** (`pkg/config/`) - YAML configuration loading and validation
- **Schema Layer** (`internal/schema/`) - Schema introspection, comparison, and migration generation
- **Content Layer** (`internal/content/`) - Data hashing, diff computation, and pack generation
- **Driver Layer** (`internal/drivers/`) - Database connection management and abstraction

## Limitations

Current limitations and known constraints:

- **Database Support** - MSSQL and Oracle are not yet supported (planned for future releases)
- **Schema Auto-merge** - Schema differences must be resolved manually
- **Primary Key Requirement** - All tables must have primary keys (unless explicitly ignored)
- **Large Database Performance** - Very large databases may produce large diff files and require significant processing time
- **Conflict Resolution** - Conflict resolution is currently manual (automated strategies planned)
- **SQLite Constraints** - SQLite has limited support for ALTER TABLE operations

See [ROADMAP.md](ROADMAP.md) for planned features and improvements.

## Testing

The project includes comprehensive test coverage:

```bash
# Run all tests
go test ./tests/...

# Run tests by package
go test ./tests/config
go test ./tests/content
go test ./tests/schema
go test ./tests/drivers

# Run integration tests (requires Docker)
go test ./tests -run TestIntegration -v
```

Integration tests use testcontainers and automatically spin up MySQL and PostgreSQL containers for full workflow validation.

## Contributing

Contributions are welcome. Please:

1. Review existing issues and discussions
2. Fork the repository
3. Create a feature branch
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

[License information to be added]

## Support

For issues, questions, or feature requests, please use the [GitHub Issues](https://github.com/iamvirul/deepdiff-db/issues) page.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for detailed information about planned features, release schedule, and development priorities.
