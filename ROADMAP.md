# DeepDiff DB Roadmap

This roadmap outlines planned features and improvements for DeepDiff DB, organized by weekly Saturday releases leading to production readiness.

## Release Schedule

We release a new version every **Saturday**. Each release includes one or more features from this roadmap, prioritized by impact and production readiness requirements.

---

## Current Status: v0.8

**Last Release:** 2026-03-14

**Current Features:**
- Schema drift detection and standalone schema migration (`schema-migrate`)
- Row-level data comparison with SHA-256 hashing
- Migration pack generation and transactional apply mode
- MySQL, PostgreSQL, SQLite, and **Microsoft SQL Server** support
- Conflict detection with `ours`/`theirs`/`manual` resolution strategies
- Interactive `resolve-conflicts` command with `--auto` and `--resume` flags
- Per-table conflict resolution strategies with `resolutions.json` persistence
- DROP/MODIFY COLUMN, CREATE/DROP TABLE, CREATE/DROP INDEX, ADD/DROP FOREIGN KEY
- Primary key modification and dependency-aware migration ordering
- Interactive HTML report with schema diff viewer, data diff, conflict highlighting, and SQL preview
- Structured JSON/text logging with configurable levels and file output
- Visual progress bars and throughput metrics
- Checkpoint/resume system for long-running operations
- Enhanced error handling with actionable suggestions and retry logic
- **NEW:** Keyset-paginated batch hashing — `--batch-size N` / `performance.hash_batch_size`
- **NEW:** Parallel table hashing — `--parallel N` / `performance.max_parallel_tables`
- **NEW:** Bounded O(batch_size) memory during hashing regardless of table size
- **NEW:** Per-batch memory telemetry at DEBUG log level (`alloc_mb`, `batch`)

---

## Completed Releases

### v0.6: Enhanced Error Handling & Logging (Released 2026-01-06)

**Features Delivered:**
- Structured logging system with JSON/text formats
- Log levels (DEBUG, INFO, WARN, ERROR) with configurable output
- File output support for log persistence
- Operation metrics collection with timing information
- Enhanced error handling with error codes and categories
- Rich error context with actionable suggestions
- Optional stack trace capture for debugging
- Progress tracking with bars and spinners
- Throughput calculation and performance metrics
- Checkpoint/resume system for long-running operations
- `--resume` flag for `gen-pack` and `apply` commands
- Configuration hash validation for checkpoint safety
- Retry logic with exponential backoff for transient errors
- Comprehensive test coverage improvements (logger: 97.6%, errors: 78.6%, progress: 98.4%)

**Impact:** Significantly improved observability, debugging capabilities, and user experience with progress indicators and better error messages

---

### v0.5: HTML Report Viewer (Released 2026-01-03)

**Features Delivered:**
- Interactive HTML report generation with `--html` flag
- Professional minimal UI design (GitHub/Linear inspired)
- Tab-based navigation (Schema, Data, Conflicts, Migration)
- Visual schema diff viewer:
  - Collapsible sections with +/−/~ indicators
  - Column and index change display
  - Foreign key changes with ON DELETE/UPDATE actions
- Data diff visualization:
  - Table filtering dropdown
  - Expandable row keys (click to see affected primary keys)
- Conflict management:
  - Resolution strategy breakdown (auto-resolved ours/theirs vs pending)
  - Per-table strategy table with conflict/resolved/pending counts
  - Strategy badges on each conflict (ours/theirs/manual)
- SQL migration preview with syntax highlighting
- Export to PDF functionality (via browser print)
- Self-contained HTML with embedded CSS and JavaScript

**Impact:** Provides comprehensive visual analysis of database differences with professional UI

---

### v0.4: Conflict Resolution Strategies (Released 2026-01-03)

**Features Delivered:**
- Merge strategies: ours (prod), theirs (dev), manual
- Interactive conflict resolution CLI (`resolve-conflicts` command)
- Conflict resolution configuration in YAML
- Per-table conflict resolution strategies
- Automatic conflict resolution with `--auto` flag
- Resolution persistence with `--resume` flag
- Enhanced conflict reports with resolution statistics

**Impact:** Reduces manual intervention for conflict resolution

---

## Completed Releases (continued)

### v0.7: Streaming Support for Large Datasets (Released 2026-03-14)

**Features Delivered:**
- Keyset-paginated batch hashing (`WHERE pk > lastVal ORDER BY pk LIMIT N`) — O(batch_size) heap at any table size
- `--batch-size N` and `--parallel N` CLI flags for `diff` and `gen-pack`
- `performance.hash_batch_size` and `performance.max_parallel_tables` config keys (defaults: 10000 / 1)
- Bounded goroutine pool via `errgroup` + `semaphore.NewWeighted` for parallel table hashing
- `BuildCursorQuery` shared module (`internal/content/cursor.go`) used by both hash and pack paths
- Per-batch memory telemetry at DEBUG level
- Sample 14: Streaming Large Datasets (SQLite, no Docker, seed script + Makefile)

**Impact:** Enables comparison of databases with millions of rows while keeping memory usage bounded and wall-clock time short

---

## Upcoming Releases

---

### ~~v0.8: MSSQL Support~~ ✅ Released 2026-03-14

**Features delivered:**
- Microsoft SQL Server driver (`github.com/microsoft/go-mssqldb`)
- Schema introspection via `INFORMATION_SCHEMA` + `sys.*` catalog views
- MSSQL-compatible SQL generation (square-bracket quoting, `ALTER COLUMN`, `DROP INDEX … ON …`)
- FK control via `sp_msforeachtable` in pack application
- `OFFSET/FETCH` pagination (no `LIMIT`)
- Integration tests with SQL Server 2022 (testcontainers)
- Sample 15: MSSQL Support

**Impact:** Enterprise SQL Server users can now use DeepDiff DB in production

---

### v0.9: Oracle Support
**Target Date:** Week 2 Saturday

**Features:**
- Oracle Database driver support
- Oracle-specific schema introspection
- Oracle-specific SQL generation
- Transaction handling for Oracle
- Testing with Oracle 12c+ versions

**Impact:** Expands database support to enterprise Oracle users

---

### v1.0: Production Ready Release
**Target Date:** Week 3 Saturday

**Features:**
- Comprehensive documentation
- Performance benchmarking and optimization
- Security audit and improvements
- CI/CD integration examples
- Docker image support
- Package manager support (Homebrew, apt, etc.)
- Migration guide from v0.x to v1.0
- Production deployment best practices

**Impact:** Ready for production use with enterprise-grade reliability

---

## Future Enhancements (Post v1.0)

### Phase 2 Features

1. **Git-like Versioning for DB Diffs**
   - Store diff history
   - Diff between any two versions
   - Rollback capabilities

2. **CI/CD Integration**
   - GitHub Actions plugin
   - GitLab CI integration
   - Jenkins plugin
   - Pre-commit hooks

3. **Advanced Schema Features**
   - View and stored procedure diff
   - Trigger comparison
   - Function/procedure diff
   - Sequence comparison

4. **Performance & Scalability**
   - Parallel table processing
   - Distributed diff processing
   - Incremental diff (only changed tables)
   - Diff caching

5. **Developer Experience**
   - VS Code extension
   - CLI autocomplete
   - Configuration wizard
   - Interactive mode

6. **Enterprise Features**
   - Audit logging
   - Role-based access control
   - API server mode
   - Web dashboard
   - Multi-tenant support

---

## Priority Matrix

### High Priority (Must Have for v1.0)
- ~~Conflict Resolution Strategies~~ (v0.4)
- ~~HTML Report Viewer~~ (v0.5)
- ~~Enhanced Error Handling & Logging~~ (v0.6)
- ~~Streaming Support for Large Datasets~~ (v0.7)
- ~~MSSQL Support~~ (v0.8)
- Documentation & Production Readiness

### Low Priority (Nice to Have)
- Oracle Support (can be post-v1.0)
- Advanced features (post-v1.0)

---

## Success Criteria for v1.0

- [x] Conflict Resolution Strategies (v0.4)
- [x] HTML Report Viewer (v0.5)
- [x] Enhanced Error Handling & Logging (v0.6)
- [x] Streaming Support for Large Datasets (v0.7)
- [ ] All high-priority features implemented
- [ ] Test coverage > 80%
- [ ] Comprehensive documentation
- [ ] Performance benchmarks documented
- [ ] Security audit completed
- [ ] At least 3 database types fully supported
- [ ] Production deployment guide
- [ ] Migration path from v0.x documented

---

## Notes

- This roadmap is flexible and may be adjusted based on feedback and priorities
- Features may be combined or split across releases as needed
- Bug fixes and improvements are included in each release
- Community feedback is welcome and will influence priorities

---

## Contributing

If you'd like to contribute to any of these features, please:
1. Check existing issues in `.github/ISSUE_TEMPLATE/`
2. Comment on the relevant feature issue
3. Submit a pull request with your implementation

---

**Last Updated:** 2026-03-14

