---
sidebar_position: 8
---

# Troubleshooting

Common issues and how to resolve them.

## Connection refused / connection timeout

**Symptom:** `check` or any diff command fails with `connection refused` or `dial tcp ... i/o timeout`.

**Causes and fixes:**
- The database server is not running. Start it and retry.
- The `host` or `port` in the config is wrong. Verify with a direct connection using `psql`, `mysql`, `sqlplus`, or `sqlcmd`.
- A firewall is blocking the port. Check security group / firewall rules.
- For Docker-based databases: the container has not finished initialising. Wait a few seconds and retry, or use `docker ps` to check health status.

## "table has no primary key" error

**Symptom:** `check` exits with `table 'xyz' has no primary key`.

**Cause:** DeepDiff DB requires every compared table to have a primary key for keyset pagination and row identification.

**Fix:** Either add a primary key to the table, or add it to `ignore.tables` in your config:

```yaml
ignore:
  tables:
    - "xyz"
```

## Missing rows in diff output

**Symptom:** You know a row changed but it does not appear in `content_diff.json`.

**Cause:** The changed column is in `ignore.columns`. Changes to ignored columns are deliberately excluded from hashing.

**Fix:** Remove the column from `ignore.columns`, or check for wildcard patterns like `*.updated_at` that may be matching the column you care about.

## Schema diff shows changes that already exist in production

**Symptom:** `schema-diff` reports a column or index as added/missing, but it clearly exists in both databases.

**Cause:** The introspection query is fetching from a different schema or database than expected. This is most common with PostgreSQL (non-`public` schemas) or Oracle (connecting as a user that sees objects from a different schema via `ALL_TAB_COLUMNS`).

**Fix:** Verify that the `user` configured has `SELECT` on the correct objects. For Oracle, the schema owner prefix in `ALL_TAB_COLUMNS.OWNER` must match the connected user's schema. For PostgreSQL, check the `search_path` of the connecting user.

## Out of memory on large tables

**Symptom:** `diff` or `gen-pack` is killed by the OS with an out-of-memory error.

**Fix:** Enable batched hashing with `--batch-size`:

```bash
deepdiffdb diff --config deepdiffdb.config.yaml --batch-size 5000
```

See [Streaming Large Datasets](/docs/features/streaming) for detailed tuning guidance. Start with `--batch-size 5000` and lower further if needed.

## Oracle ORA-01017: invalid username/password

**Symptom:** Oracle connection fails with `ORA-01017`.

**Cause:** The username or password is incorrect, or the password contains special characters that are misinterpreted in the DSN.

**Fix:**
1. Verify the credentials directly with `sqlplus user/password@host:port/service`.
2. If the password contains special characters (`@`, `/`, `#`, `%`), try URL-encoding them or wrapping the password value in single quotes in the YAML.
3. Oracle passwords are case-sensitive when created with double-quote syntax. Verify the exact case.
4. The service name must be the Oracle **service name** (e.g. `XEPDB1`), not the SID.

## MSSQL login failed

**Symptom:** MSSQL connection fails with `login failed for user`.

**Cause:** The SA account may be disabled (SQL Server defaults to Windows Authentication mode). Or the password is wrong.

**Fix:**
1. Ensure SQL Server is running in **Mixed Mode** (SQL Server and Windows Authentication). In SSMS: right-click server → Properties → Security → select Mixed Mode.
2. Verify the password. For Docker-based SQL Server the SA password must meet complexity requirements (uppercase, lowercase, digit, symbol, 8+ characters).
3. After changing the authentication mode, restart the SQL Server service.

## SQLite "database is locked"

**Symptom:** SQLite operations fail with `database is locked`.

**Cause:** Another process has the SQLite file open in write mode. SQLite allows only one writer at a time.

**Fix:** Stop any other processes that have the database open (application server, DB browser, etc.) before running `deepdiffdb`. For `apply`, this is especially important because the apply command opens the production database for writing.

## Pack apply fails mid-way

**Symptom:** `apply` prints an error partway through and the operation stops.

**Cause:** A SQL statement in the pack failed (constraint violation, type mismatch, etc.). The transaction was rolled back.

**Fix:**
1. Inspect the error message — it includes the failing statement.
2. Edit `migration_pack.sql` to fix or remove the problematic statement.
3. Re-run `apply`. If the pack is large and you had partial progress, use `--resume` if checkpointing was enabled.

If the issue is a FK constraint violation, verify that FK checks are being disabled correctly for your database driver (see the [database pages](/docs/databases/mysql) for driver-specific FK handling).

## Diff output files are missing after running gen-pack

**Symptom:** `gen-pack` completes but `schema_diff.json` or `content_diff.json` are absent.

**Cause:** The `output.dir` directory did not exist and could not be created, or a previous run's files were deleted.

**Fix:** Check that the `output.dir` path is writable by the user running `deepdiffdb`. Run `deepdiffdb check` to verify output directory access.
