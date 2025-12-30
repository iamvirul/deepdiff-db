# Sample 10: Conflict Resolution Configuration

This sample demonstrates the conflict resolution configuration feature in DeepDiff DB. It shows how to configure default resolution strategies and per-table overrides for automated conflict handling.

## What You'll Learn

- How to configure conflict resolution strategies
- How to set a default strategy for all tables
- How to override strategies for specific tables
- Use cases for each resolution strategy

## Scenario

This sample simulates a scenario with four tables, each requiring different conflict handling:

| Table | Strategy | Reason |
|-------|----------|--------|
| `users` | `manual` | Critical user data requires careful review |
| `logs` | `theirs` | Ephemeral data, dev version is more recent |
| `config` | `ours` | Production config must be preserved |
| `settings` | `manual` | Uses default strategy (manual) |

### Data Conflicts

**Users Table (manual):**
- Row 1: Email changed in dev (conflict)
- Row 2: Status changed in dev (conflict)
- Row 4: New user in dev (added)

**Logs Table (theirs):**
- Row 1: Different details in dev (conflict - use dev)
- Row 3: Removed in dev (removed)
- Row 4-5: New logs in dev (added)

**Config Table (ours):**
- Row 1: `app_name` differs (conflict - keep prod)
- Row 2: `max_login_attempts` differs (conflict - keep prod)
- Row 4: `maintenance_mode` differs (conflict - keep prod)
- Row 5: New config in dev (added)

**Settings Table (manual - default):**
- Row 1: `theme` changed (conflict)
- Row 2: `notifications` changed (conflict)
- Row 4: New setting in dev (added)

## Prerequisites

- Docker and Docker Compose installed
- DeepDiff DB installed locally

## Setup

### 1. Start the Databases

```bash
cd samples/10-conflict-resolution
docker-compose up -d
```

This starts two MySQL databases:
- **Production DB (db1)**: Port 3318
- **Development DB (db2)**: Port 3317

Wait for databases to be ready:

```bash
docker-compose logs -f
# Press Ctrl+C when you see "ready for connections"
```

### 2. Verify Configuration

```bash
deepdiffdb check --config deepdiffdb.config.yaml
```

## Configuration

The configuration file includes a `conflict_resolution` section:

```yaml
conflict_resolution:
  # Default strategy for all tables
  default_strategy: "manual"

  # Per-table overrides
  strategies:
    - table: "logs"
      strategy: "theirs"    # Always use dev version

    - table: "config"
      strategy: "ours"      # Always keep prod version

    - table: "users"
      strategy: "manual"    # Require manual review
```

### Resolution Strategies

| Strategy | Description | Use Case |
|----------|-------------|----------|
| `ours` | Keep production version | Config tables, critical production data |
| `theirs` | Use development version | Logs, caches, temporary data |
| `manual` | Require manual review | User data, financial records |

## Usage Examples

### Example 1: View Data Differences

Run a diff to see conflicts:

```bash
deepdiffdb diff --config deepdiffdb.config.yaml
```

Review the generated reports:

```bash
cat diff-output/summary.txt
cat diff-output/conflicts.json
```

### Example 2: Generate Migration Pack

When generating a migration pack, the configured strategies will be applied:

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml
```

**Expected behavior:**
- `logs` conflicts: Dev version applied automatically
- `config` conflicts: Prod version kept (no changes)
- `users` conflicts: Reported for manual review
- `settings` conflicts: Reported for manual review (default strategy)

### Example 3: Review Conflicts

For tables with `manual` strategy, review conflicts in:

```bash
cat diff-output/conflicts.json
```

## Understanding the Output

### Summary Report

The summary shows how many conflicts were resolved automatically vs. requiring manual review:

```
Table: logs
  - Conflicts: 1 (auto-resolved: theirs)
  - Added: 2
  - Removed: 1

Table: config
  - Conflicts: 3 (auto-resolved: ours)
  - Added: 1

Table: users
  - Conflicts: 2 (manual review required)
  - Added: 1

Table: settings
  - Conflicts: 2 (manual review required)
  - Added: 1
```

### Migration Pack

The generated migration pack will:
1. Apply all `theirs` changes automatically
2. Skip all `ours` conflicts (keep prod)
3. Include comments about `manual` conflicts requiring review

## Common Use Cases

### 1. Log Tables

```yaml
conflict_resolution:
  strategies:
    - table: "logs"
      strategy: "theirs"
    - table: "audit_log"
      strategy: "theirs"
```

Log data is typically ephemeral. The development version is often more recent and relevant for testing.

### 2. Configuration Tables

```yaml
conflict_resolution:
  strategies:
    - table: "config"
      strategy: "ours"
    - table: "feature_flags"
      strategy: "ours"
```

Production configuration should rarely be overwritten by development values. This prevents accidental changes to production settings.

### 3. User Data

```yaml
conflict_resolution:
  strategies:
    - table: "users"
      strategy: "manual"
    - table: "accounts"
      strategy: "manual"
```

User data is critical and should always be reviewed manually before any changes are applied.

### 4. Development Workflow

For development environments where dev is the source of truth:

```yaml
conflict_resolution:
  default_strategy: "theirs"
  strategies:
    - table: "production_config"
      strategy: "manual"
```

## Validation

The configuration is validated at load time. Invalid strategies will cause an error:

```bash
# This would fail:
conflict_resolution:
  default_strategy: "invalid"  # Error: invalid value
```

Valid values are: `ours`, `theirs`, `manual`

## Cleanup

```bash
docker-compose down -v
```

## Files in This Sample

- `README.md` - This documentation
- `docker-compose.yml` - Database containers configuration
- `deepdiffdb.config.yaml` - DeepDiff DB configuration with conflict resolution
- `init-scripts/01-prod-schema.sql` - Production database schema and data
- `init-scripts/02-dev-schema.sql` - Development database schema and data
- `diff-output/` - Generated reports (created when you run the tool)

## Learn More

- [DeepDiff DB Documentation](../../README.md)
- [Sample 01: Basic Schema Drift](../01-basic-schema-drift/) - Getting started
- [Sample 02: Advanced Migrations](../02-advanced-migrations/) - Full workflow

---

**DeepDiff DB - Smart Database Comparison Tool**
