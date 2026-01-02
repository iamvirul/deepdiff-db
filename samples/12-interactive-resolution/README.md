# Sample 12: Interactive Conflict Resolution

This sample demonstrates the `resolve-conflicts` command in DeepDiff DB. This command provides an interactive CLI for reviewing and resolving database conflicts with full row data comparison, bulk operations, and session persistence.

## What You'll Learn

- How to interactively resolve conflicts one by one
- How to use auto mode for CI/CD pipelines
- How to resume a previous resolution session
- How to use bulk operations to resolve multiple conflicts at once
- How resolutions are persisted to `resolutions.json`

## Features

### Interactive Mode (Default)

- Side-by-side row comparison showing all column values
- Visual difference markers (`*`) for changed columns
- Per-conflict resolution choices
- Bulk operations for table or all remaining conflicts
- Save progress and resume later

### Auto Mode (`--auto`)

- Applies configured strategies without prompts
- Ideal for CI/CD pipelines
- Reports summary of auto-resolved vs pending conflicts

### Resume Mode (`--resume`)

- Loads existing `resolutions.json` and continues where you left off
- Only shows pending (unresolved) conflicts
- Preserves previous decisions

## Scenario

This sample uses an e-commerce database with 6 tables:

| Table | Strategy | Conflicts | Interactive? |
|-------|----------|-----------|--------------|
| `products` | `theirs` | 3 | No (auto-resolved) |
| `orders` | `ours` | 3 | No (auto-resolved) |
| `inventory_log` | `theirs` | 2 | No (auto-resolved) |
| `customers` | `manual` | 3 | Yes |
| `feature_flags` | `manual` | 3 | Yes |
| `audit_trail` | `theirs` | 0 | N/A |

**Total: 14 conflicts**
- 8 auto-resolved (theirs/ours strategies)
- 6 require interactive resolution (manual strategy)

## Prerequisites

- Docker and Docker Compose installed
- DeepDiff DB installed locally

## Setup

### 1. Start the Databases

```bash
cd samples/12-interactive-resolution
docker-compose up -d
```

Wait for databases to be ready:

```bash
docker-compose logs -f
# Press Ctrl+C when you see "ready for connections"
```

### 2. Verify Configuration

```bash
deepdiffdb check --config deepdiffdb.config.yaml
```

### 3. Generate Conflicts

Run a diff to detect conflicts:

```bash
deepdiffdb diff --config deepdiffdb.config.yaml
```

Then generate the initial resolutions file:

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml
```

## Usage Examples

### Example 1: Auto Mode (CI/CD)

Apply all configured strategies automatically:

```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --auto
```

**Expected Output:**

```
DeepDiff DB - Conflict Resolution
=================================
Total conflicts: 14
Pending review: 6

Applying auto-resolution strategies...

Resolution Summary
==================
Total conflicts:        14
Resolved (keep prod):    3
Resolved (use dev):      5
Pending (manual):        6

Resolutions saved to: ./diff-output/resolutions.json
```

### Example 2: Interactive Mode

Start an interactive resolution session:

```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml
```

**Interactive Display:**

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Conflict 1 of 6 | Table: customers | Key: 1
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Column          | Production        | Development
  ----------------+-------------------+-------------------
  id              | 1                 | 1
  email           | john@example.com  | john@example.com
  name            | John Doe          | John Doe
  tier            | gold              | platinum          *
  loyalty_points  | 1500              | 2500              *

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Choose resolution:
  [1] Keep production (ours)
  [2] Use development (theirs)
  [3] Skip (decide later)
  [4] Apply "ours" to all remaining in customers
  [5] Apply "theirs" to all remaining in customers
  [6] Apply "ours" to ALL remaining conflicts
  [7] Apply "theirs" to ALL remaining conflicts
  [q] Quit and save progress

Enter choice (1-7, q):
```

### Example 3: Resume Session

After quitting with `q`, resume where you left off:

```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --resume
```

The command will:
1. Load existing `resolutions.json`
2. Skip already-resolved conflicts
3. Show only pending conflicts

### Example 4: Bulk Operations

During interactive mode, use options 4-7 for bulk operations:

- **Option 4**: Apply "ours" (keep prod) to all remaining conflicts in the current table
- **Option 5**: Apply "theirs" (use dev) to all remaining conflicts in the current table
- **Option 6**: Apply "ours" to ALL remaining conflicts across all tables
- **Option 7**: Apply "theirs" to ALL remaining conflicts across all tables

Example workflow:

```
Enter choice (1-7, q): 4
Applied "ours" to 3 conflicts in table customers

Enter choice (1-7, q): 7
Applied "theirs" to 3 remaining conflicts
```

## Understanding the Output

### Resolution Summary

After completing (or quitting) a session:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Resolution Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Total conflicts:        14
  Resolved (keep prod):    6
  Resolved (use dev):      5
  Pending (manual):        3
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Resolutions saved to: ./diff-output/resolutions.json
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Resolutions File

The `resolutions.json` file persists all decisions:

```json
{
  "version": "1.0",
  "created_at": "2026-01-02T10:00:00Z",
  "updated_at": "2026-01-02T10:05:00Z",
  "resolutions": [
    {
      "conflict": {
        "table": "customers",
        "key": "1",
        "prod_hash": "abc123...",
        "dev_hash": "def456..."
      },
      "strategy": "ours",
      "decision": "keep_prod",
      "resolved": true
    }
  ]
}
```

## Command Flags

| Flag | Description |
|------|-------------|
| `--config` | Path to configuration file (required) |
| `--auto` | Apply configured strategies without prompts |
| `--resume` | Resume from existing resolutions.json |
| `--conflicts` | Custom path to conflicts.json |
| `--resolutions` | Custom path to resolutions.json |

## Workflow Integration

### Development Workflow

1. Run `diff` to detect conflicts
2. Run `gen-pack` to generate initial resolutions
3. Run `resolve-conflicts` interactively to review
4. Run `gen-pack` again to generate migration with resolved conflicts

### CI/CD Pipeline

```bash
# Auto-resolve with configured strategies
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --auto

# Check for pending manual conflicts
if grep -q '"decision": "pending"' diff-output/resolutions.json; then
  echo "Manual conflicts require review"
  exit 1
fi

# Generate migration pack
deepdiffdb gen-pack --config deepdiffdb.config.yaml
```

## Cleanup

```bash
docker-compose down -v
```

## Files in This Sample

- `README.md` - This documentation
- `docker-compose.yml` - Database containers configuration
- `deepdiffdb.config.yaml` - Configuration with mixed resolution strategies
- `init-scripts/01-prod-schema.sql` - Production database schema and data
- `init-scripts/02-dev-schema.sql` - Development database schema and data
- `diff-output/` - Generated reports (created when you run the tool)

## Related Samples

- [Sample 10: Conflict Resolution Configuration](../10-conflict-resolution/) - Configuration basics
- [Sample 11: Resolution Engine](../11-resolution-engine/) - Engine internals
- [Sample 02: Advanced Migrations](../02-advanced-migrations/) - Full workflow

---

**DeepDiff DB - Smart Database Comparison Tool**
