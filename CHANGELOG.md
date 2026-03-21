# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0] - 2026-03-22

### Added
- Multi-stage `Dockerfile` (scratch-based image, ~15 MB, `CGO_ENABLED=0`) and `docker-compose.example.yml`
- GoReleaser configuration (`.goreleaser.yml`) — replaces manual release workflow; produces cross-platform archives, SHA256 checksums, multi-arch GHCR Docker manifests, and auto-updates Homebrew formula
- CI/CD integration examples: GitHub Actions PR diff check, GitLab CI MR check, pre-commit hook (`examples/cicd/`)
- Go benchmark suite for `HashTable` at 1k/10k rows, batched and unbatched (`tests/content/bench_test.go`)
- `MIGRATION.md` — upgrade guide from v0.x to v1.0
- `gosec` and `govulncheck` jobs added to CI workflow
- Docusaurus documentation site with Deployment section: Docker, CI/CD, Performance, Migration guide

### Changed
- Release workflow replaced with GoReleaser; archive names updated (e.g. `deepdiffdb_v1.0.0_linux_amd64.tar.gz`)
- Homebrew formula updated to inject version via ldflags; tap updated to `iamvirul/tap`
- Go minimum version bumped to **1.25.8** in `go.mod`; all CI jobs updated accordingly

### Security
- File permissions tightened: `WriteFile` `0644→0600`, `MkdirAll` `0755→0750`, log `OpenFile` `0644→0600`
- `os.Create` in HTML generator replaced with `os.OpenFile(..., 0o600)`
- Resolved **GO-2026-4603** (`html/template` URL escaping) and **GO-2026-4601** (`net/url` IPv6 parsing) by upgrading to Go 1.25.8

## [0.9] - 2026-03-21

### Added
- **Oracle Database Support** (issue #14)
  - New driver: `driver: "oracle"` in `deepdiffdb.config.yaml`; uses `sijms/go-ora/v2` (pure Go, no Oracle Instant Client required)
  - DSN format: `oracle://user:pass@host:port/service_name`
  - Schema introspection via `ALL_TAB_COLUMNS`, `ALL_INDEXES`, `ALL_IND_COLUMNS`, `ALL_CONSTRAINTS`, `ALL_CONS_COLUMNS`
  - Double-quote identifier quoting: `"TABLE"`, `"COLUMN"` (same as PostgreSQL)
  - `OFFSET 0 ROWS FETCH NEXT n ROWS ONLY` pagination (Oracle 12c+; no `LIMIT` clause)
  - Oracle-compatible `ALTER TABLE` generation: `ADD "col"` (no `COLUMN` keyword), `MODIFY "col"` (not `ALTER COLUMN`), `DROP COLUMN "col"`, `DROP INDEX "name"` (standalone DDL, no table name), `DROP CONSTRAINT "name"`
  - Full type support: `NUMBER`, `VARCHAR2`, `DATE`, `TIMESTAMP`, `CLOB`, `BLOB`, `CHAR`, `FLOAT`
  - `GENERATED ALWAYS AS IDENTITY` for auto-increment columns
  - Port is optional for Oracle (defaults to 1521 when `port: 0`)
  - `CheckPrimaryKeys` extended with Oracle case querying `ALL_CONSTRAINTS` / `ALL_CONS_COLUMNS`
  - Integration test `TestIntegration_Oracle_FullWorkflow` using testcontainers + `gvenzl/oracle-xe:21-slim-faststart`
- **Sample 16: Oracle Support** (`samples/16-oracle-support/`)
  - Docker Compose with two Oracle XE 21c containers (prod on 1521, dev on 1522)
  - SQLPlus init scripts with deliberate schema drift: new columns, nullability changes, FK relationships
  - Seed data with deliberate data drift: price updates, new customer, new orders
  - Makefile targets: `up`, `wait-healthy`, `seed`, `diff`, `gen-pack`, `down`, `clean`

### Changed
- `pkg/config/config.go`: `oracle` added to valid driver list; port validation skipped for Oracle (optional, defaults to 1521)
- `internal/drivers/drivers.go`: Oracle DSN builder added (`oracle://user:pass@host:port/service`)
- `internal/content/hash.go` / `cursor.go`: Oracle identifier quoting and `OFFSET/FETCH` pagination
- `internal/content/pack.go`: Oracle FK disable/enable via `DISABLE CONSTRAINT` / `ENABLE CONSTRAINT`, Oracle-specific DDL generation
- `internal/schema/introspect.go`: Oracle catalog query path
- `internal/schema/migrate.go`: Oracle DDL generation for all migration types
- `internal/schema/primary_keys.go`: Oracle `ALL_CONSTRAINTS` path
- `deepdiffdb.config.yaml.example`: Commented Oracle connection example added

## [0.8] - 2026-03-14

### Added
- **Microsoft SQL Server Support** (issue #13)
  - New driver: `driver: "mssql"` in `deepdiffdb.config.yaml`; maps to `github.com/microsoft/go-mssqldb` (`sqlserver://` DSN)
  - Schema introspection via `INFORMATION_SCHEMA.COLUMNS`, `sys.indexes`, `sys.index_columns`, `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS`, and `sys.key_constraints`
  - Square-bracket identifier quoting: `[table]`, `[column]`, with `]]` escaping for literal `]`
  - `OFFSET 0 ROWS FETCH NEXT n ROWS ONLY` pagination (MSSQL has no `LIMIT` clause)
  - MSSQL-compatible `ALTER TABLE` generation: `ADD` (no `COLUMN` keyword), `ALTER COLUMN`, `DROP CONSTRAINT`, `DROP INDEX … ON …`
  - FK control during pack application via `sp_msforeachtable`: `NOCHECK CONSTRAINT ALL` / `WITH CHECK CHECK CONSTRAINT ALL`
  - Full type reconstruction: `varchar(max)`, `decimal(10,2)`, `nvarchar(255)` with `nchar`/`nvarchar` half-length correction (MSSQL stores byte-length)
  - `CheckPrimaryKeys` extended with MSSQL case querying `sys.tables` / `sys.key_constraints`
  - Port is optional for MSSQL (defaults to SQL Server standard 1433)
  - Integration test `TestIntegration_MSSQL_FullWorkflow` using testcontainers + `mcr.microsoft.com/mssql/server:2022-latest`
- **Sample 15: MSSQL Support** (`samples/15-mssql-support/`)
  - Docker Compose with two SQL Server 2022 Express containers (prod on 1433, dev on 1434)
  - Init SQL scripts with deliberate schema drift: new columns, nullability changes, new indexes, FK relationships
  - Seed data with deliberate data drift: price updates, status changes, new rows
  - Makefile targets: `up`, `seed`, `diff`, `diff-schema`, `diff-content`, `gen-pack`, `down`, `clean`

### Changed
- `pkg/config/config.go`: `mssql` added to valid driver list; port validation skipped for MSSQL (optional)
- `internal/drivers/drivers.go`: MSSQL DSN builder added (`sqlserver://user:pass@host:port?database=db`)
- `internal/content/hash.go` / `cursor.go`: MSSQL identifier quoting and `OFFSET/FETCH` pagination
- `internal/content/pack.go`: MSSQL FK disable/enable, column type introspection, `ALTER TABLE ADD` syntax
- `deepdiffdb.config.yaml.example`: Commented MSSQL connection example added

## [0.7] - 2026-03-14

### Added
- **Streaming Support for Large Datasets**
  - Keyset-paginated batch hashing for large tables (`WHERE pk > lastVal ORDER BY pk LIMIT N`)
  - Each page fetches a bounded number of rows, keeping heap flat regardless of table size (~150–200 MB peak vs ~700–900 MB unbatched)
  - `--batch-size N` CLI flag for `diff` and `gen-pack` — overrides `performance.hash_batch_size` in config
  - `--parallel N` CLI flag for `diff` and `gen-pack` — overrides `performance.max_parallel_tables` in config
  - `performance` configuration section in `deepdiffdb.config.yaml`:
    ```yaml
    performance:
      hash_batch_size: 10000      # rows per keyset-paginated query (0 = disabled)
      max_parallel_tables: 2      # tables hashed concurrently
    ```
  - Parallel table hashing via bounded goroutine pool (`errgroup` + `semaphore.NewWeighted`)
  - Per-batch memory telemetry at `DEBUG` level (`alloc_mb`, `batch`, `total_rows_hashed`)
  - `--batch-size 0` falls back to pre-v0.7 full-scan behaviour (full backward compatibility)
- **Shared Keyset Query Builder** (`internal/content/cursor.go`)
  - `BuildCursorQuery` and `buildCursorWhere` extracted into a shared module supporting composite primary keys
  - Used by both `hash.go` and `pack.go` — eliminates cursor logic drift between the two
- **Sample 14: Streaming Large Datasets** (`samples/14-streaming-large-datasets/`)
  - Go seed script generating 500k orders / 100k products / 200k audit_logs in SQLite
  - Makefile targets: `seed`, `seed-small`, `diff`, `diff-fast`, `diff-sequential`, `gen-pack`, `clean`
  - Memory tuning guide for low/standard/high-memory hosts
  - No Docker required

### Changed
- `HashTable` signature extended with `batchSize int` parameter; `batchSize=0` preserves original behaviour
- Sequential per-table loop in `runFullDiff` and `runGenPack` replaced with `hashTablesParallel`
- Inline cursor closure in `pack.go` replaced with `BuildCursorQuery` (DRY)
- `performance.hash_batch_size` defaults to `10000`; `performance.max_parallel_tables` defaults to `1`
- `deepdiffdb.config.yaml.example` updated with commented `performance:` section

### Performance
- **~4× throughput improvement** on multi-table databases with `--parallel 4`
- Memory during hashing reduced from O(n) unbounded growth to O(batch_size) bounded heap
- `runtime.GC()` hint issued after each batch to return memory promptly between pages

## [0.6.1] - 2026-01-08

### Added
- **Homebrew Tap Support**
  - Introduced `deepdiff-db.rb` Homebrew formula.
  - Added GitHub Actions workflow (`update-homebrew-formula.yml`) for automated formula updates on release.
  - Provided `HOMEBREW_TAP.md` with detailed usage and maintenance guidelines.
  - Updated `README.md` with Homebrew installation instructions.

## [0.6] - 2026-01-06

### Added
- **Structured Logging System** (`pkg/logger`)
  - JSON and text log formats with configurable output
  - Log levels: DEBUG, INFO, WARN, ERROR
  - File output support for log persistence
  - Source location tracking (optional, for debugging)
  - Operation metrics collection with timing information
  - Context-aware logging with structured fields
  - Convenience methods: `WithTable()`, `WithOperation()`, `WithDatabase()`
  - `LogOperation()` for automatic operation timing and error logging
  - Metrics summary printing with formatted output
- **Enhanced Error Handling** (`pkg/errors`)
  - Custom error type with error codes and categories
  - Rich error context with key-value pairs
  - Actionable suggestions for error resolution
  - Optional stack trace capture for debugging
  - Error code categorization (Connection, Schema, Data, Migration, Checkpoint, Configuration, System)
  - Retryable error detection with exponential backoff
  - Default suggestions for common error codes
  - Context-specific suggestions based on error details
  - Debug string format with full error details
  - Standard error interface compatibility
- **Progress Tracking System** (`pkg/progress`)
  - Progress bars for operations with known totals
  - Spinners for operations with unknown duration
  - Throughput calculation (rows/second)
  - Performance metrics collection (duration, rows processed, memory usage, query count)
  - Disabled mode for CI/CD environments
  - Context propagation for progress manager
  - Metrics summary with formatted output
- **Checkpoint/Resume System** (`internal/checkpoint`)
  - Automatic checkpoint saving during long-running operations
  - Resume capability with `--resume` flag for `gen-pack` and `apply` commands
  - Configuration hash validation to ensure consistency
  - Checkpoint expiration (24-hour default) for safety
  - Atomic checkpoint file writes
  - Support for hash table, pack generation, and pack application operations
  - State persistence across interruptions
- **CLI Enhancements**
  - `--log-format` flag for choosing JSON or text log format
  - `--log-level` flag for setting minimum log level
  - `--log-file` flag for writing logs to a file
  - `--resume` flag for resuming interrupted operations
  - Progress indicators for all long-running operations
  - Enhanced error messages with suggestions
- **Comprehensive Test Coverage**
  - Test coverage for logger package: 97.6%
  - Test coverage for errors package: 78.6%
  - Test coverage for progress package: 98.4%
  - Test coverage for checkpoint package: 28.5%
  - Comprehensive tests for all new packages and features

### Changed
- All commands now use structured logging instead of `log.Printf`
- Error handling improved throughout with enhanced error types
- Progress bars displayed for operations exceeding 10,000 rows
- Checkpoint system integrated into `gen-pack` and `apply` commands
- Database connection retry logic with exponential backoff
- Improved user experience with progress indicators and better error messages

### Fixed
- All linter errors resolved (errcheck, staticcheck)
- Proper error handling for all function return values
- Context handling in tests (no nil contexts)
- Type-safe context keys

## [0.5] - 2026-01-03

### Added
- Interactive HTML report generation with `--html` flag
  - Works with both `diff` and `gen-pack` commands
  - Self-contained HTML with embedded CSS and JavaScript
  - Professional minimal UI design inspired by GitHub/Linear
  - Tab-based navigation (Schema, Data, Conflicts, Migration)
- Schema diff viewer with advanced features:
  - Collapsible sections with +/−/~ indicators
  - Column changes with type and nullable diff display
  - Index changes with column information
  - **Foreign Key changes** showing add/remove/modify with ON DELETE/UPDATE actions
- Data diff visualization:
  - Table filtering dropdown
  - Row counts for added/removed/modified
  - **Expandable row keys** - click to reveal affected primary keys
- Conflict management:
  - **Resolution strategy breakdown** showing auto-resolved (ours/theirs) vs pending counts
  - **Per-table strategy table** with conflict/resolved/pending counts per table
  - **Strategy badges** on each conflict (ours/theirs/manual)
  - Resolution status indicators (Keep Source/Use Target/Pending)
- SQL migration preview with syntax highlighting
- Export to PDF functionality (via browser print)
- New package `internal/report/html` for HTML report generation
- Comprehensive test coverage for HTML report generation (13 tests)
- New sample configuration:
  - `13-html-report-viewer`: Demonstrates HTML report generation with various schema and data changes

### Changed
- Updated CLI help text to mention `--html` flag support
- `diff` and `gen-pack` commands now support `--html` flag
- `BuildReportData` now accepts resolutions for detailed breakdown display

## [0.4] - 2026-01-03

### Added
- `resolve-conflicts` command for interactive conflict resolution
  - Interactive mode with side-by-side row comparison display
  - `--auto` flag for automated resolution using configured strategies
  - `--resume` flag to continue from saved resolutions
  - Bulk operations for resolving multiple conflicts at once
  - Progress tracking and resolution summary display
- Conflict resolution configuration support in `deepdiffdb.config.yaml`
  - `conflict_resolution.default_strategy` for global default (`ours`, `theirs`, `manual`)
  - `conflict_resolution.strategies` array for per-table strategy overrides
- Resolution persistence with `resolutions.json` file
  - Save and load resolution decisions
  - Merge saved resolutions with new conflicts
  - Track resolution timestamps and decisions
- Enhanced conflict reports with resolution statistics
  - `summary.txt` now includes Resolution Summary section
  - New `resolutions_summary.json` with detailed statistics
  - Breakdown by decision type (keep_prod, use_dev, pending)
  - Breakdown by table
- New sample configurations:
  - `10-conflict-detection`: Demonstrates conflict detection and reporting
  - `11-resolution-engine`: Demonstrates conflict resolution engine
  - `12-interactive-resolution`: Demonstrates interactive resolve-conflicts command
- Comprehensive test coverage for new features:
  - CLI prompt and display utilities
  - Resolution engine (apply strategies, summary building)
  - Resolution persistence (save, load, merge)
  - Enhanced report generation with resolutions

### Changed
- `gen-pack` command now generates enhanced reports with resolution statistics when resolutions are available
- Updated `deepdiffdb.config.yaml.example` with conflict resolution configuration examples
- Improved documentation with resolve-conflicts command usage

### Fixed
- Empty branch staticcheck warning in persistence tests

## [0.3] - 2025-12-30

### Added
- `schema-migrate` command for generating standalone schema migration scripts
- Support for DROP COLUMN operations with configurable safety controls (`allow_drop_column`)
- Support for MODIFY COLUMN operations with DEFAULT value handling
- Support for CREATE TABLE and DROP TABLE operations with configurable safety controls (`allow_drop_table`)
- Index support for schema migrations (CREATE INDEX, DROP INDEX) with configurable safety controls (`allow_drop_index`)
- Foreign key support for schema migrations (ADD FOREIGN KEY, DROP FOREIGN KEY) with configurable safety controls (`allow_drop_foreign_key`)
- Primary key modification support with configurable safety controls (`allow_modify_primary_key`)
- Dependency ordering for schema migrations to ensure correct execution order
- Comprehensive test coverage for ordering logic
- Comprehensive test coverage for index support
- Comprehensive test coverage for DEFAULT value introspection
- Sample configurations demonstrating new features:
  - `04-drop-column-safety`: Demonstrates DROP COLUMN feature with safety controls
  - `05-modify-column`: Demonstrates MODIFY COLUMN feature
  - `06-index-support`: Demonstrates index operations
  - `07-table-operations`: Demonstrates CREATE/DROP TABLE operations
  - `08-foreign-key-support`: Demonstrates foreign key operations
  - `09-dependency-ordering`: Demonstrates dependency ordering for migrations
- ROADMAP.md documenting future features and release schedule
- Semantic versioning with timestamp-based dev builds in build scripts

### Changed
- Schema migration generation now uses dependency-aware ordering
- Improved identifier quoting to handle embedded quotes and schema-qualified names
- MySQL MODIFY COLUMN now always includes NULL constraint explicitly
- Build scripts now include version information with git commit hash
- PR checks workflow with branch protection integration

### Fixed
- Fixed error return value check in container.Terminate calls
- Fixed column_type vs data_type usage for MySQL introspection
- Fixed PR checks workflow permission and merge abort issues
- Updated sample documentation to use correct command syntax

## [0.2] - 2025-12-20

### Added
- Comprehensive test coverage for `pack.go` (improved from ~25% to ~61%)
- Comprehensive test coverage for `report.go` (improved from ~64% to ~68%)
- Tests for MySQL, PostgreSQL, and SQLite driver-specific behaviors
- Tests for schema diff scenarios with new columns
- Tests for composite primary keys
- Tests for UPDATE statement generation for new columns
- Tests for error handling and edge cases
- Tests for various data types (timestamps, blobs, etc.)
- Tests for ignored columns in schema diffs
- Tests for unsupported driver handling

### Changed
- Overall content package test coverage improved to 81.5%
- `GeneratePack` function coverage improved to 89.9%
- `getFullColumnType` function coverage improved to 62.5%
- `literal` function coverage improved to 44.4%

### Fixed
- Fixed noisy test that was testing incompatible driver/database combinations
- Improved test assertions to fail on unexpected behavior instead of just logging

## [0.1] - Initial Release

### Added
- Fast Go-based diff engine for database comparison
- Support for MySQL, PostgreSQL, and SQLite databases
- Schema drift detection and reporting
- Row-level data comparison using SHA-256 hashing
- Conflict detection for rows that differ between databases
- Auto-generated SQL migration pack for applying changes
- Dry-run mode for validating migration packs
- Fully transactional apply mode
- JSON and text reports for diffs and conflicts
- Configurable ignore lists for tables and columns
- Single static binary per OS with zero dependencies
- Works with DB connections or dump files
- Comprehensive test suite with integration tests
- Codecov integration for code coverage tracking
- CI/CD workflows for automated testing and releases
- Local development build scripts
- Sample configurations and examples
- Documentation and README

### Features
- High-speed database comparison
- Safe migration pack generation
- Structural validation before content merging
- Deterministic and reviewable changes
- Support for large databases with cursor-based pagination
- Batched UPDATE statements with CASE expressions for efficiency
- PostgreSQL schema-aware queries
- MySQL foreign key check handling

[Unreleased]: https://github.com/iamvirul/deepdiff-db/compare/v1.0...HEAD
[1.0]: https://github.com/iamvirul/deepdiff-db/compare/v0.9...v1.0
[0.9]: https://github.com/iamvirul/deepdiff-db/compare/v0.8...v0.9
[0.8]: https://github.com/iamvirul/deepdiff-db/compare/v0.7...v0.8
[0.7]: https://github.com/iamvirul/deepdiff-db/compare/v0.6.1...v0.7
[0.6.1]: https://github.com/iamvirul/deepdiff-db/compare/v0.6...v0.6.1
[0.6]: https://github.com/iamvirul/deepdiff-db/compare/v0.5...v0.6
[0.5]: https://github.com/iamvirul/deepdiff-db/compare/v0.4...v0.5
[0.4]: https://github.com/iamvirul/deepdiff-db/compare/v0.3...v0.4
[0.3]: https://github.com/iamvirul/deepdiff-db/compare/v0.2...v0.3
[0.2]: https://github.com/iamvirul/deepdiff-db/compare/v0.1...v0.2
[0.1]: https://github.com/iamvirul/deepdiff-db/releases/tag/v0.1

