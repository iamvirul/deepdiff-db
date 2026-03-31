# DeepDiff DB

[![CI](https://github.com/iamvirul/deepdiff-db/actions/workflows/ci.yml/badge.svg)](https://github.com/iamvirul/deepdiff-db/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/iamvirul/deepdiff-db?color=6366f1)](https://github.com/iamvirul/deepdiff-db/releases/latest)
[![codecov](https://codecov.io/gh/iamvirul/deepdiff-db/branch/main/graph/badge.svg?token=Y9IORTUBAH)](https://codecov.io/gh/iamvirul/deepdiff-db)
[![Go Report Card](https://goreportcard.com/badge/github.com/iamvirul/deepdiff-db)](https://goreportcard.com/report/github.com/iamvirul/deepdiff-db)
[![Go Version](https://img.shields.io/badge/go-1.25.8-00ADD8?logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/docker-ghcr.io-2496ED?logo=docker)](https://github.com/iamvirul/deepdiff-db/pkgs/container/deepdiff-db)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![Docs](https://img.shields.io/badge/docs-iamvirul.github.io-6366f1)](https://iamvirul.github.io/deepdiff-db/)

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
- **Multi-database support** - MySQL, PostgreSQL, SQLite, Microsoft SQL Server, and Oracle Database
- **Schema drift detection** - Identifies structural differences between databases
- **Row-level comparison** - SHA-256 hashing for accurate change detection
- **Conflict detection** - Identifies rows that exist in both databases but differ
- **Auto-generated SQL migration packs** - Production-ready migration scripts
- **Dry-run mode** - Validate migrations without executing them
- **Fully transactional apply mode** - All changes applied atomically
- **Comprehensive reporting** - JSON and human-readable text reports
- **Configurable ignore lists** - Exclude tables and columns from comparison
- **Flexible input sources** - Works with database connections or dump files
- **Structured logging** - JSON/text formats with configurable levels and file output
- **Progress tracking** - Visual progress bars and spinners for long-running operations
- **Checkpoint/resume** - Resume interrupted operations from saved checkpoints
- **Enhanced error handling** - Rich error messages with actionable suggestions
- **Streaming large datasets** - Keyset-paginated batch hashing keeps memory bounded at any table size
- **Parallel table hashing** - Hash multiple tables concurrently with configurable worker pool
- **Git-like versioning** - Commit diff snapshots, browse history, compare versions, generate rollback SQL

### Safety Features

- Primary key validation for all tables
- Transaction-wrapped migrations
- Destructive operation warnings
- Schema drift blocking (configurable)
- Conflict reporting before application
- Dry-run validation mode

## Installation

### Option 1: Homebrew (macOS and Linux - Recommended)

Install using Homebrew tap:

```bash
# Tap the repository
brew tap iamvirul/deepdiff-db

# Install deepdiff-db
brew install deepdiff-db
```

Or install in one command:

```bash
brew install iamvirul/deepdiff-db/deepdiff-db
```

**Upgrade to latest version:**
```bash
brew upgrade deepdiff-db
```

**Install development version:**
```bash
brew install --HEAD iamvirul/deepdiff-db/deepdiff-db
```

### Option 2: Download Precompiled Binaries

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

### Option 3: Build from Source

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

**Note:** For Homebrew tap maintainers, see [HOMEBREW_TAP.md](HOMEBREW_TAP.md) for instructions on updating the formula.

### Option 4: Build All Platform Binaries

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
  driver: "mysql"          # mysql, postgres, postgresql, sqlite, mssql, or oracle
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

conflict_resolution:
  default_strategy: "manual"   # ours, theirs, or manual
  strategies:
    - table: "logs"
      strategy: "theirs"       # Always use dev version
    - table: "config"
      strategy: "ours"         # Always keep prod version
```

### Configuration Options

**Database Configuration:**
- `driver`: Database driver (`mysql`, `postgres`, `postgresql`, `sqlite`, `mssql`, or `oracle`)
- `host`: Database hostname or IP address
- `port`: Database port number (not required for SQLite; defaults to 1433 for MSSQL, 1521 for Oracle)
- `user`: Database username
- `password`: Database password
- `database`: Database name (for Oracle: the Oracle service name, e.g. `XEPDB1`)

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

**Conflict Resolution Configuration:**
- `conflict_resolution.default_strategy`: Default strategy for all tables (`ours`, `theirs`, or `manual`)
- `conflict_resolution.strategies`: Per-table strategy overrides (array of `{table, strategy}` objects)

Resolution strategies:
- `ours`: Keep production values (reject dev changes)
- `theirs`: Use development values (accept dev changes)
- `manual`: Require interactive decision for each conflict

**Performance Configuration (v0.7+):**
- `performance.hash_batch_size`: Rows per keyset-paginated query during table hashing. `0` disables batching (loads all rows in one query). Default: `10000`
- `performance.max_parallel_tables`: Maximum number of tables hashed concurrently. Default: `1`

```yaml
performance:
  hash_batch_size: 10000      # ~1–2 MB per page; keeps heap bounded on any table size
  max_parallel_tables: 2      # hash prod tables in parallel; raises throughput ~2× on dual-core
```

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
- Driver-specific SQL syntax (MySQL, PostgreSQL, SQLite, MSSQL, Oracle)
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

**Large Dataset Options (v0.7+):**
```bash
# Keyset-paginated hashing, 2 tables in parallel
deepdiffdb diff --config deepdiffdb.config.yaml --batch-size 10000 --parallel 2

# Disable batching (pre-v0.7 behaviour, loads all rows in memory)
deepdiffdb diff --config deepdiffdb.config.yaml --batch-size 0 --parallel 1
```

- `--batch-size N`: Rows per page when hashing large tables. Overrides `performance.hash_batch_size`. `0` = no pagination.
- `--parallel N`: Max tables hashed concurrently. Overrides `performance.max_parallel_tables`.

**Generate Interactive HTML Report:**
```bash
deepdiffdb diff --config deepdiffdb.config.yaml --html
```

The `--html` flag generates an interactive `report.html` file with:
- Professional minimal UI design with tab-based navigation
- Visual schema diff viewer with collapsible sections (columns, indexes, foreign keys)
- Data diff visualization with filtering and expandable row keys
- Conflict highlighting with strategy badges and resolution status
- Export to PDF functionality (via browser print)

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

**Large Dataset Options (v0.7+):**
```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml --batch-size 5000 --parallel 4
```

- `--batch-size N`: Rows per page when hashing large tables. Overrides `performance.hash_batch_size`.
- `--parallel N`: Max tables hashed concurrently. Overrides `performance.max_parallel_tables`.

**Resume from Checkpoint:**
```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml --resume
```

Resumes a previously interrupted operation from the last checkpoint.

**Generate Interactive HTML Report:**
```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml --html
```

The `--html` flag generates an interactive `report.html` file with:
- All diff information (schema, data, conflicts)
- Resolution strategy breakdown (auto-resolved vs pending)
- Per-table strategy table showing conflict counts
- SQL migration preview with syntax highlighting
- Expandable row keys for detailed inspection

**Output Files:**
- `schema_diff.json` - Schema differences (warnings only)
- `content_diff.json` - Data differences
- `conflicts.json` - Conflict details
- `summary.txt` - Summary statistics with resolution breakdown
- `resolutions_summary.json` - Detailed resolution statistics (when resolutions exist)
- `migration_pack.sql` - Combined migration script
- `report.html` - Interactive HTML report (when `--html` flag is used)

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

**Resume from Checkpoint:**
```bash
deepdiffdb apply --pack migration_pack.sql --config deepdiffdb.config.yaml --resume
```

Resumes a previously interrupted application from the last checkpoint.

**Features:**
- Fully transactional execution
- Atomic application (all or nothing)
- Automatic rollback on error
- Applies to the production database specified in config
- Checkpoint support for resuming interrupted operations

**Dry Run Mode:**
```bash
deepdiffdb apply --pack migration_pack.sql --dry-run
```

Validates the SQL without executing it.

### resolve-conflicts

Interactively resolve data conflicts between production and development databases.

```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml
```

**Features:**
- Side-by-side row comparison with visual difference markers
- Per-conflict resolution choices (keep prod, use dev, skip)
- Bulk operations (resolve all in table, resolve all remaining)
- Session persistence (save progress and resume later)
- Auto mode for CI/CD pipelines

**Auto Mode (for CI/CD):**
```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --auto
```

Applies configured strategies automatically without prompts.

**Resume Mode:**
```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --resume
```

Continues from a previous session by loading existing resolutions.

**Output Files:**
- `resolutions.json` - Saved resolution decisions

### version

Git-like versioning for database diffs. Stores schema and data diff snapshots as commits, enabling history browsing, inter-version comparison, and rollback SQL generation — all without a live database connection.

```bash
deepdiffdb version <subcommand> [flags]
```

#### Subcommands

| Subcommand | Description |
|---|---|
| `version init` | Initialise a `.deepdiffdb/` repository in the current directory |
| `version commit` | Run a full diff and store the result as a new commit |
| `version log` | Show the commit history from most recent to oldest |
| `version diff <h1> <h2>` | Compare schema evolution between two commits |
| `version rollback <hash>` | Generate rollback SQL to undo the changes in a commit |

#### Example Workflow

```bash
# Initialise once per project
deepdiffdb version init

# Commit a snapshot whenever prod/dev diverges
deepdiffdb version commit --config deepdiffdb.config.yaml --message "V1: baseline" --author "Alice"

# After applying schema changes to dev:
deepdiffdb version commit --config deepdiffdb.config.yaml --message "V2: add reviews table" --author "Bob"

# Browse history
deepdiffdb version log

# See what changed between two commits
deepdiffdb version diff <hash_v1> <hash_v2>

# Generate rollback SQL for a commit (flags before positional hash)
deepdiffdb version rollback --out rollback.sql <hash_v2>
```

**Storage:** Commits are stored as JSON files in `.deepdiffdb/objects/` (content-addressable by SHA-256 hash). Add `.deepdiffdb/` to your `.gitignore` or commit it to share history with your team.

**Rollback SQL:** Each commit stores full schema snapshots, so rollback SQL can be generated at any time without a live database. Destructive operations (DROP TABLE, DROP COLUMN) are commented out by default — identical safety behaviour to `schema-migrate`.

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

Resolution Summary:
  Auto-resolved: 1
  Pending review: 1

By Decision:
  Keep production (ours): 1
  Pending manual review: 1

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

The tool processes data using **keyset-paginated batching** for large tables — each page fetches a bounded number of rows (`WHERE pk > lastVal ORDER BY pk LIMIT N`), keeping heap usage flat regardless of table size. Multiple tables can be hashed concurrently using a bounded goroutine pool. Progress bars show throughput (rows/second) and estimated time remaining. Checkpoints are automatically saved during long-running operations, allowing you to resume from interruptions.

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

## Logging and Progress

DeepDiff DB provides comprehensive logging and progress tracking:

**Logging Options:**
- `--log-format`: Choose between `text` (default) or `json` format
- `--log-level`: Set minimum log level (`debug`, `info`, `warn`, `error`)
- `--log-file`: Write logs to a file in addition to stdout
- `--verbose`: Enable debug-level logging and source location tracking

**Progress Indicators:**
- Progress bars for operations with known totals (e.g., hashing tables)
- Spinners for operations with unknown duration (e.g., database connections)
- Throughput metrics (rows/second) displayed in real-time
- Performance metrics summary at operation completion

**Checkpoint/Resume:**
- Automatic checkpoint saving during long-running operations
- `--resume` flag to continue from the last checkpoint
- Configuration validation to ensure consistency
- 24-hour checkpoint expiration for safety

**Example:**
```bash
# Generate pack with JSON logging and progress tracking
deepdiffdb gen-pack --config deepdiffdb.config.yaml \
  --log-format json \
  --log-level info \
  --log-file operation.log

# Resume interrupted operation
deepdiffdb gen-pack --config deepdiffdb.config.yaml --resume
```

## Limitations

Current limitations and known constraints:

- **Schema Auto-merge** - Schema differences must be resolved manually
- **Primary Key Requirement** - All tables must have primary keys (unless explicitly ignored)
- **Large Database Performance** - Very large tables are handled with keyset-paginated batching (v0.7+); diff output files may still be large for tables with many changed rows
- **Conflict Resolution** - Complex merge strategies (e.g., column-level merging) are not supported
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

Integration tests use testcontainers and automatically spin up MySQL, PostgreSQL, MSSQL, and Oracle XE containers for full workflow validation.

## Contributing

Contributions are welcome. Please:

1. Review existing issues and discussions
2. Fork the repository
3. Create a feature branch
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the [Apache License 2.0](LICENSE).

```text
Copyright 2026 Virul Nirmala

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## Support

For issues, questions, or feature requests, please use the [GitHub Issues](https://github.com/iamvirul/deepdiff-db/issues) page.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for detailed information about planned features, release schedule, and development priorities.
