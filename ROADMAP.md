# DeepDiff DB Roadmap

This roadmap outlines planned features and improvements for DeepDiff DB, organized by weekly Saturday releases leading to production readiness.

## Release Schedule

We release a new version every **Saturday**. Each release includes one or more features from this roadmap, prioritized by impact and production readiness requirements.

---

## Current Status: v1.2.0 — Verified Authorship & CI/CD Hardening 🎉

**Released:** 2026-04-05

**Features:**
- GitHub OAuth author verification for `version commit` (issue #77) ✅
- Documentation & website updates throughout

---

## Previous Release: v1.1.0 — Git-like Versioning 🎉

**Released:** 2026-04-01

**Features:**
- Schema drift detection and standalone schema migration (`schema-migrate`)
- Row-level data comparison with SHA-256 hashing
- Migration pack generation and transactional apply mode
- MySQL, PostgreSQL, SQLite, Microsoft SQL Server, and **Oracle Database** support
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
- Keyset-paginated batch hashing — `--batch-size N` / `performance.hash_batch_size`
- Parallel table hashing — `--parallel N` / `performance.max_parallel_tables`
- Bounded O(batch_size) memory during hashing regardless of table size
- Per-batch memory telemetry at DEBUG log level (`alloc_mb`, `batch`)
- Oracle Database support — pure Go driver, no Instant Client required
- Git-like versioning — `version init/commit/log/diff/rollback` with SHA-256 content-addressable commit objects and offline rollback SQL generation
- **NEW in v1.2.0:** GitHub OAuth author verification — device flow auth in `version init`; verified `github:<username>` used automatically in `version commit`

---

## Completed Releases

### v1.1.0: Git-like Versioning for DB Diffs (Released 2026-04-01)

**Features Delivered:**
- `version init` — initialises `.deepdiffdb/` repository (objects store + HEAD file) in the working directory
- `version commit` — runs a full schema+data diff and stores a SHA-256 content-addressable commit object capturing both schema snapshots, drift result, and data diff
- `version log` — walks the commit chain from HEAD with author, timestamp, and drift markers
- `version diff <h1> <h2>` — compares dev schema snapshots of two commits and reports schema evolution (added/removed tables, column/index changes)
- `version rollback <hash>` — generates driver-aware rollback SQL offline by inverting the stored diff; same safety defaults as `schema-migrate` (destructive ops commented out by default); `--out` and `--driver` flags
- New `internal/version` package: `model.go`, `store.go`, `rollback.go`
- Sample 17: Git-like Versioning — two MySQL 8 containers, three-sprint demo, automated `demo.sh`

### v1.2.0: Verified Authorship & CI/CD Hardening (Released 2026-04-05)

**Scope:**
- **GitHub OAuth author verification** (issue #77) — `version init` prompts GitHub device flow; verified `github:<username>` stored in `.deepdiffdb/config` (`0o600`); used automatically by `version commit`; `--skip-auth` for CI; `DEEPDIFFDB_GITHUB_CLIENT_ID` env var or build-time `-ldflags` injection
- Documentation and website updated throughout for v1.2.0

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

### ~~v0.9: Oracle Support~~ ✅ Released 2026-03-21

**Features delivered:**
- Oracle Database driver (`sijms/go-ora/v2`) — pure Go, no Instant Client required
- Schema introspection via `ALL_TAB_COLUMNS`, `ALL_INDEXES`, `ALL_CONSTRAINTS`, `ALL_CONS_COLUMNS`
- Oracle DDL generation: `ADD "col"` (no COLUMN), `MODIFY "col"`, standalone `DROP INDEX`, `DROP CONSTRAINT`
- `OFFSET/FETCH` pagination (Oracle 12c+), double-quote identifier quoting
- Optional port (defaults to 1521), `GENERATED ALWAYS AS IDENTITY` for auto-increment
- Integration tests with Oracle XE 21c (testcontainers + `gvenzl/oracle-xe:21-slim-faststart`)
- Sample 16: Oracle Support (Docker Compose, SQLPlus init scripts, Makefile)

**Impact:** Enterprise Oracle users can now use DeepDiff DB in production

---

### ~~v1.0: Production Ready Release~~ ✅ Released 2026-03-22

**Features delivered:**
- Multi-stage `Dockerfile` and `docker-compose.example.yml`; multi-arch Docker images on GHCR
- GoReleaser-based release pipeline (cross-platform archives, checksums, Homebrew auto-update)
- CI/CD integration examples: GitHub Actions, GitLab CI, pre-commit hook
- Security hardening: file permissions tightened, Go 1.25.8 (fixes 2 stdlib CVEs), gosec + govulncheck in CI
- Go benchmark suite for `HashTable` with performance tuning guide
- `MIGRATION.md` upgrade guide from v0.x to v1.0
- Full Docusaurus documentation site hosted on GitHub Pages

**Impact:** Ready for production use with enterprise-grade reliability

---

## Future Enhancements (Post v1.0)

### Phase 2 Features

1. ~~**Git-like Versioning for DB Diffs**~~ ✅ **Released in v1.1.0 (2026-04-01)**
   - ~~Store diff history~~ → `version commit` — SHA-256 content-addressable commit objects in `.deepdiffdb/objects/`
   - ~~Diff between any two versions~~ → `version diff <h1> <h2>` — schema evolution comparison
   - ~~Rollback capabilities~~ → `version rollback <hash>` — driver-aware rollback SQL generation
   - ~~`version init`~~ — repository initialisation
   - ~~`version log`~~ — full commit history with drift markers
   - ~~`version branch` / `version checkout` / `version tree`~~ — branching and ASCII graph
   - See [Sample 17](https://github.com/iamvirul/deepdiff-db/tree/main/samples/17-git-like-versioning) for end-to-end demo

2. ~~**GitHub OAuth Author Verification**~~ ✅ **Shipping in v1.2.0 (issue #77)**
   - ~~`version init` GitHub device flow authentication~~
   - ~~`version commit` reads verified `github:<username>` from `.deepdiffdb/config`~~
   - ~~Build-time client ID injection via `-ldflags` in `release.yml`~~
   - ~~`DEEPDIFFDB_GITHUB_CLIENT_ID` env var override for local use~~

3. **Advanced Schema Features** _(v1.3.0 candidate)_
   - View and stored procedure diff
   - Trigger comparison
   - Function/procedure diff
   - Sequence comparison

4. **Performance & Scalability** _(v1.4.0 candidate)_
   - Parallel table processing
   - Distributed diff processing
   - Incremental diff (only changed tables)
   - Diff caching

5. **Developer Experience** _(v1.5.0 candidate)_
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
- Advanced features (post-v1.0)

---

## Success Criteria for v1.0

- [x] Conflict Resolution Strategies (v0.4)
- [x] HTML Report Viewer (v0.5)
- [x] Enhanced Error Handling & Logging (v0.6)
- [x] Streaming Support for Large Datasets (v0.7)
- [x] MSSQL Support (v0.8)
- [x] Oracle Support (v0.9)
- [x] All high-priority features implemented
- [x] Comprehensive documentation
- [x] Performance benchmarks documented
- [x] Security audit completed
- [x] At least 3 database types fully supported (5 total)
- [x] Production deployment guide (Docker, CI/CD)
- [x] Migration path from v0.x documented

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

**Last Updated:** 2026-04-05

