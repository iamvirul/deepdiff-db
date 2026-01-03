# DeepDiff DB Roadmap

This roadmap outlines planned features and improvements for DeepDiff DB, organized by weekly Saturday releases leading to production readiness.

## Release Schedule

We release a new version every **Saturday**. Each release includes one or more features from this roadmap, prioritized by impact and production readiness requirements.

---

## Current Status: v0.4

**Last Release:** 2026-01-03

**Current Features:**
- Schema drift detection
- Row-level data comparison
- Migration pack generation
- Transactional apply mode
- MySQL, PostgreSQL, SQLite support
- Conflict detection
- JSON and text reports
- Standalone schema migration command (`schema-migrate`)
- DROP COLUMN support with safety controls
- MODIFY COLUMN support (type changes, nullable changes)
- CREATE TABLE and DROP TABLE support
- Index support (CREATE INDEX, DROP INDEX)
- Foreign key constraint handling (ADD/DROP FOREIGN KEY)
- Primary key modification support
- Dependency-aware migration ordering
- **NEW:** Interactive `resolve-conflicts` command with `--auto` and `--resume` flags
- **NEW:** Conflict resolution configuration (`ours`, `theirs`, `manual` strategies)
- **NEW:** Per-table conflict resolution strategies
- **NEW:** Resolution persistence with `resolutions.json`
- **NEW:** Enhanced conflict reports with resolution statistics

---

## Completed Releases

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

## Upcoming Releases

### Week 1 - v0.5: HTML Report Viewer
**Target Date:** Next Saturday

**Features:**
- Interactive HTML report generation
- Visual schema diff viewer
- Data diff visualization with filters
- Conflict highlighting and navigation
- Export to PDF option
- Embedded SQL preview in reports

**Impact:** Improves developer experience and makes reports more accessible

---

### Week 2 - v0.6: Enhanced Error Handling & Logging
**Target Date:** Week 2 Saturday

**Features:**
- Structured logging with levels (DEBUG, INFO, WARN, ERROR)
- Progress indicators for long-running operations
- Detailed error messages with context
- Error recovery suggestions
- Log file output option
- Verbose mode for debugging

**Impact:** Better observability and debugging capabilities

---

### Week 3 - v0.7: Streaming Support for Large Datasets
**Target Date:** Week 3 Saturday

**Features:**
- Streaming diff for tables > 1M rows
- Memory-efficient hash computation
- Chunked processing with progress tracking
- Configurable batch sizes
- Resume capability for interrupted operations
- Performance optimizations for large databases

**Impact:** Enables comparison of very large production databases

---

### Week 4 - v0.8: MSSQL Support
**Target Date:** Week 4 Saturday

**Features:**
- Microsoft SQL Server driver support
- MSSQL-specific schema introspection
- MSSQL-specific SQL generation
- Transaction handling for MSSQL
- Testing with MSSQL 2019+ versions

**Impact:** Expands database support to enterprise SQL Server users

---

### Week 5 - v0.9: Oracle Support
**Target Date:** Week 5 Saturday

**Features:**
- Oracle Database driver support
- Oracle-specific schema introspection
- Oracle-specific SQL generation
- Transaction handling for Oracle
- Testing with Oracle 12c+ versions

**Impact:** Expands database support to enterprise Oracle users

---

### Week 6 - v1.0: Production Ready Release
**Target Date:** Week 6 Saturday

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
- Enhanced Error Handling & Logging
- Documentation & Production Readiness

### Medium Priority (Should Have)
- HTML Report Viewer
- Streaming Support for Large Datasets
- MSSQL Support

### Low Priority (Nice to Have)
- Oracle Support (can be post-v1.0)
- Advanced features (post-v1.0)

---

## Success Criteria for v1.0

- [x] Conflict Resolution Strategies (v0.4)
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

**Last Updated:** 2026-01-03

