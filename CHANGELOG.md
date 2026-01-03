# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/iamvirul/deepdiff-db/compare/v0.5...HEAD
[0.5]: https://github.com/iamvirul/deepdiff-db/compare/v0.4...v0.5
[0.4]: https://github.com/iamvirul/deepdiff-db/compare/v0.3...v0.4
[0.3]: https://github.com/iamvirul/deepdiff-db/compare/v0.2...v0.3
[0.2]: https://github.com/iamvirul/deepdiff-db/compare/v0.1...v0.2
[0.1]: https://github.com/iamvirul/deepdiff-db/releases/tag/v0.1

