# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/iamvirul/deepdiff-db/compare/v0.2...HEAD
[0.2]: https://github.com/iamvirul/deepdiff-db/compare/v0.1...v0.2
[0.1]: https://github.com/iamvirul/deepdiff-db/releases/tag/v0.1

