# DeepDiff DB Roadmap

This roadmap outlines planned features and improvements for DeepDiff DB, organized by weekly Saturday releases leading to production readiness.

## Release Schedule

We release a new version every **Saturday**. Each release includes one or more features from this roadmap, prioritized by impact and production readiness requirements.

---

## Current Status: v0.2

**Last Release:** 2025-12-20

**Current Features:**
- Schema drift detection
- Row-level data comparison
- Migration pack generation
- Transactional apply mode
- MySQL, PostgreSQL, SQLite support
- Conflict detection
- JSON and text reports

---

## Upcoming Releases

### Week 1 - v0.3: Enhanced Schema Migration Generator
**Target Date:** Next Saturday**

**Current State (v0.2):**
- ADD COLUMN already implemented (as part of data migration packs)

**New Features:**
- Dedicated standalone schema migration command (`schema-migrate`)
- Support for DROP COLUMN (with safety checks)
- Support for MODIFY COLUMN (type changes, nullable changes)
- Support for ADD INDEX, DROP INDEX
- Support for table creation/deletion (CREATE TABLE, DROP TABLE)
- Foreign key constraint handling (ADD/DROP FOREIGN KEY)
- Primary key modifications
- Safe migration ordering (dependency analysis)
- Separate schema migrations from data migrations

**Impact:** Enables comprehensive automatic schema synchronization between environments with a dedicated schema-only migration workflow

---

### Week 2 - v0.4: Conflict Resolution Strategies
**Target Date:** Week 2 Saturday

**Features:**
- Merge strategies: ours (prod), theirs (dev), manual
- Interactive conflict resolution CLI
- Conflict resolution configuration in YAML
- Per-table conflict resolution strategies
- Automatic conflict resolution for non-critical tables
- Conflict resolution report

**Impact:** Reduces manual intervention for conflict resolution

---

### Week 3 - v0.5: HTML Report Viewer
**Target Date:** Week 3 Saturday

**Features:**
- Interactive HTML report generation
- Visual schema diff viewer
- Data diff visualization with filters
- Conflict highlighting and navigation
- Export to PDF option
- Embedded SQL preview in reports

**Impact:** Improves developer experience and makes reports more accessible

---

### Week 4 - v0.6: Enhanced Error Handling & Logging
**Target Date:** Week 4 Saturday

**Features:**
- Structured logging with levels (DEBUG, INFO, WARN, ERROR)
- Progress indicators for long-running operations
- Detailed error messages with context
- Error recovery suggestions
- Log file output option
- Verbose mode for debugging

**Impact:** Better observability and debugging capabilities

---

### Week 5 - v0.7: Streaming Support for Large Datasets
**Target Date:** Week 5 Saturday

**Features:**
- Streaming diff for tables > 1M rows
- Memory-efficient hash computation
- Chunked processing with progress tracking
- Configurable batch sizes
- Resume capability for interrupted operations
- Performance optimizations for large databases

**Impact:** Enables comparison of very large production databases

---

### Week 6 - v0.8: MSSQL Support
**Target Date:** Week 6 Saturday

**Features:**
- Microsoft SQL Server driver support
- MSSQL-specific schema introspection
- MSSQL-specific SQL generation
- Transaction handling for MSSQL
- Testing with MSSQL 2019+ versions

**Impact:** Expands database support to enterprise SQL Server users

---

### Week 7 - v0.9: Oracle Support
**Target Date:** Week 7 Saturday

**Features:**
- Oracle Database driver support
- Oracle-specific schema introspection
- Oracle-specific SQL generation
- Transaction handling for Oracle
- Testing with Oracle 12c+ versions

**Impact:** Expands database support to enterprise Oracle users

---

### Week 8 - v1.0: Production Ready Release
**Target Date:** Week 8 Saturday

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
- Schema Migration Generator
- Conflict Resolution Strategies
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

**Last Updated:** 2025-12-20

