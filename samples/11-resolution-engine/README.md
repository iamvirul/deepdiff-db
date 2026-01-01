# Sample 11: Resolution Engine

This sample demonstrates the conflict resolution engine in DeepDiff DB. The resolution engine is the core component that applies configured strategies to detected conflicts, determining which should be auto-resolved and which require manual review.

## What You'll Learn

- How the resolution engine processes conflicts
- Understanding resolution decisions (keep_prod, use_dev, pending)
- How conflicts are categorized as resolved vs unresolved
- Practical e-commerce scenario with mixed resolution strategies

## Resolution Engine Overview

The resolution engine takes detected conflicts and applies the configured strategy for each table:

| Strategy | Decision | Resolved | Action |
|----------|----------|----------|--------|
| `ours` | `keep_prod` | `true` | Keep production version, skip in migration |
| `theirs` | `use_dev` | `true` | Use development version, include in migration |
| `manual` | `pending` | `false` | Requires human review before migration |

## Scenario

This sample simulates an e-commerce application with six tables, each requiring different conflict handling:

### Table Strategy Summary

| Table | Strategy | Reason | Expected Conflicts |
|-------|----------|--------|-------------------|
| `products` | `theirs` | Price/stock updates from dev | 3 auto-resolved |
| `orders` | `ours` | Production orders are authoritative | 3 kept in prod |
| `inventory_log` | `theirs` | Logs should sync from dev | 2 auto-resolved |
| `customers` | `manual` | Critical data needs review | 3 pending review |
| `feature_flags` | `ours` | Prod flags must be preserved | 3 kept in prod |
| `audit_trail` | `theirs` | Audit logs sync from dev | 0 (same data) |

### Detailed Conflict Breakdown

**Products (theirs - use_dev):**
```
Row 1: Laptop Pro    - Price: $1299.99 -> $1199.99 (sale price)
Row 2: Wireless Mouse - Stock: 200 -> 180 (sales occurred)
Row 4: Monitor 27"   - Price: $399.99 -> $349.99 (promotion)
```

**Orders (ours - keep_prod):**
```
Row 1: Order #1 - Status: completed -> pending (test reversion)
Row 2: Order #2 - Quantity: 3 -> 5 (test modification)
Row 4: Order #4 - Status: processing -> pending (test data)
```

**Customers (manual - pending):**
```
Row 1: John Doe   - Tier: gold -> platinum, Points: 1500 -> 2500
Row 2: Jane Smith - Points: 200 -> 350
Row 4: Alice Brown - Tier: silver -> gold, Points: 800 -> 1200
```

## Prerequisites

- Docker and Docker Compose installed
- DeepDiff DB installed locally

## Setup

### 1. Start the Databases

```bash
cd samples/11-resolution-engine
docker-compose up -d
```

This starts two MySQL databases:
- **Production DB (db1)**: Port 3320
- **Development DB (db2)**: Port 3319

Wait for databases to be ready:

```bash
docker-compose logs -f
# Press Ctrl+C when you see "ready for connections"
```

### 2. Verify Configuration

```bash
deepdiffdb check --config deepdiffdb.config.yaml
```

## Usage Examples

### Example 1: Detect Conflicts

Run a diff to identify all conflicts:

```bash
deepdiffdb diff --config deepdiffdb.config.yaml
```

Review the conflicts report:

```bash
cat diff-output/conflicts.json
```

### Example 2: Generate Migration Pack with Resolution

Generate a migration pack with the resolution engine applied:

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml
```

**Expected Output:**

```
Conflict Resolution Summary:
  Total conflicts: 14
  Auto-resolved (theirs -> use dev): 5
  Auto-resolved (ours -> keep prod): 6
  Pending manual review: 3

Schema OK. Data diff complete.
Changes detected. Pack written to ./diff-output/migration_pack.sql
  6 conflicts excluded (ours strategy - keeping prod values)
  Warning: 3 conflicts excluded (manual review required)
```

**Resolution Engine Behavior:**

1. **Auto-resolved (theirs)**: Products, inventory_log, audit_trail
   - Conflicts **included** in migration pack
   - Dev version will be applied to prod

2. **Auto-resolved (ours)**: Orders, feature_flags
   - Conflicts **excluded** from migration pack
   - Prod version preserved (no changes applied)

3. **Unresolved (manual)**: Customers
   - Conflicts **excluded** from migration pack
   - Reported for manual review before applying

### Example 3: Review Pending Conflicts

For tables with `manual` strategy, conflicts are marked as pending:

```bash
cat diff-output/conflicts.json | grep -A5 '"table": "customers"'
```

These conflicts require manual review before deciding whether to:
- Keep the production version (customer tier/points)
- Use the development version (updated tiers/points)

## Understanding Resolution Output

### Resolution Summary

When the resolution engine processes conflicts, it produces:

```
Resolution Summary:
==================
Total Conflicts: 14

Auto-Resolved (theirs): 5
  - products: 3 conflicts -> use_dev
  - inventory_log: 2 conflicts -> use_dev

Auto-Resolved (ours): 6
  - orders: 3 conflicts -> keep_prod
  - feature_flags: 3 conflicts -> keep_prod

Pending Manual Review: 3
  - customers: 3 conflicts -> pending
```

### Resolution Decisions

Each conflict gets a resolution with:

```json
{
  "conflict": {
    "table": "products",
    "key": "1",
    "prod_hash": "abc123...",
    "dev_hash": "def456..."
  },
  "strategy": "theirs",
  "decision": "use_dev",
  "resolved": true
}
```

## Resolution Engine API

The resolution engine provides these core functions:

```go
// Resolve all conflicts based on config
resolutions := resolve.ResolveConflicts(conflicts, config)

// Get strategy for a specific table
strategy := resolve.GetStrategyForTable("products", config)

// Apply strategy to a single conflict
resolution := resolve.ApplyStrategy(conflict, strategy)

// Filter conflicts by resolution status
unresolved := resolve.FilterResolved(conflicts, resolutions)
resolved := resolve.FilterUnresolved(conflicts, resolutions)

// Get counts by decision type
counts := resolve.CountByDecision(resolutions)
// {keep_prod: 6, use_dev: 5, pending: 3}

// Filter DataDiff for pack generation (integrated with gen-pack)
filteredDiff, excludedCounts := resolve.FilterDataDiffByResolutions(dataDiff, resolutions)

// Build resolution summary for reporting
summary := resolve.BuildResolutionSummary(resolutions)
// summary.TotalConflicts, summary.ResolvedCount, summary.UnresolvedCount
```

## Use Cases

### 1. E-Commerce Product Updates

```yaml
conflict_resolution:
  strategies:
    - table: "products"
      strategy: "theirs"
    - table: "prices"
      strategy: "theirs"
```

Product catalog changes (prices, descriptions, stock) flow from dev to prod.

### 2. Preserve Production State

```yaml
conflict_resolution:
  strategies:
    - table: "orders"
      strategy: "ours"
    - table: "transactions"
      strategy: "ours"
```

Financial and order data in production is never overwritten.

### 3. Critical Data Review

```yaml
conflict_resolution:
  strategies:
    - table: "customers"
      strategy: "manual"
    - table: "accounts"
      strategy: "manual"
```

Customer account changes require human approval.

## Cleanup

```bash
docker-compose down -v
```

## Files in This Sample

- `README.md` - This documentation
- `docker-compose.yml` - Database containers configuration
- `deepdiffdb.config.yaml` - DeepDiff DB configuration with resolution strategies
- `init-scripts/01-prod-schema.sql` - Production database schema and data
- `init-scripts/02-dev-schema.sql` - Development database schema and data
- `diff-output/` - Generated reports (created when you run the tool)

## Related Samples

- [Sample 10: Conflict Resolution Configuration](../10-conflict-resolution/) - Configuration basics
- [Sample 02: Advanced Migrations](../02-advanced-migrations/) - Full workflow
- [Sample 01: Basic Schema Drift](../01-basic-schema-drift/) - Getting started

---

**DeepDiff DB - Smart Database Comparison Tool**
