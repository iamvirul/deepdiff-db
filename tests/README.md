# Integration Tests

This directory contains comprehensive test suites for the deepdiff-db project.

## Test Structure

```
tests/
├── config/          - Configuration loading and validation tests
├── content/         - Data diffing, hashing, and migration pack tests
├── drivers/         - Database driver DSN building tests
├── schema/          - Schema introspection and diffing tests
├── integration_test.go                    - Basic integration tests (SQLite)
└── integration_testcontainers_test.go    - Full integration tests with Docker
```

## Running Tests

### All Tests
```bash
go test ./tests/...
```

### Specific Test Package
```bash
go test ./tests/config
go test ./tests/content
go test ./tests/schema
go test ./tests/drivers
```

### Integration Tests (Requires Docker)

The `integration_testcontainers_test.go` file contains comprehensive integration tests that use Docker containers to test the full workflow with real MySQL and PostgreSQL databases.

**Prerequisites:**
- Docker must be installed and running
- Docker daemon must be accessible

**Run integration tests:**
```bash
# Run all integration tests
go test ./tests -run TestIntegration -v

# Run MySQL integration test only
go test ./tests -run TestIntegration_MySQL_FullWorkflow -v

# Run PostgreSQL integration test only
go test ./tests -run TestIntegration_PostgreSQL_FullWorkflow -v

# Run report generation test (no Docker required)
go test ./tests -run TestIntegration_AllReportsGenerated -v
```

## What the Integration Tests Cover

### TestIntegration_MySQL_FullWorkflow
1. **Schema Diff**: Tests schema comparison between prod and dev MySQL databases
2. **Full Diff**: Tests complete data diffing workflow
3. **Generate Pack**: Tests migration pack generation
4. **Apply Pack**: Tests migration pack application and verifies data matches

### TestIntegration_PostgreSQL_FullWorkflow
- Same workflow as MySQL but with PostgreSQL databases
- Tests PostgreSQL-specific features and SQL generation

### TestIntegration_AllReportsGenerated
- Verifies all report files are created:
  - `schema_diff.json` - Schema differences in JSON format
  - `schema_diff.txt` - Schema differences in human-readable format
  - `content_diff.json` - Data differences in JSON format
  - `conflicts.json` - Conflict detection results
  - `summary.txt` - Summary report with statistics
  - `migration_pack.sql` - SQL migration pack
- Validates JSON structure and content
- Checks that all required fields are present in reports

## Test Output

All tests verify:
- ✅ Schema diff reports are generated
- ✅ Content diff reports are generated
- ✅ Conflicts are detected and reported
- ✅ Migration packs are generated correctly
- ✅ Migration packs can be applied successfully
- ✅ Data matches after migration application
- ✅ All report files have valid structure and content

## Skipping Docker Tests

If Docker is not available, you can skip the testcontainers tests:

```bash
go test ./tests -run TestIntegration_AllReportsGenerated -v
```

Or use the basic integration test:
```bash
go test ./tests -run TestIntegration_FullWorkflow -v
```

## Troubleshooting

### Docker Issues
- Ensure Docker is running: `docker ps`
- Check Docker permissions
- For CI/CD, ensure Docker-in-Docker is configured

### Test Timeouts
- Container startup can take 30-60 seconds
- Tests have 60-second timeouts configured
- If tests fail with timeouts, check Docker performance

### Port Conflicts
- Testcontainers automatically assigns random ports
- No manual port configuration needed


