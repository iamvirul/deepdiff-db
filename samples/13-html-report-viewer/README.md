# Sample 13: HTML Report Viewer

This sample demonstrates the interactive HTML report generation feature in DeepDiff DB. The `--html` flag generates a self-contained HTML file with visual diff viewers, data filters, conflict highlighting, and SQL preview with syntax highlighting.

## What You'll Learn

- How to generate interactive HTML reports with `--html` flag
- Understanding the HTML report components:
  - Schema diff viewer with collapsible sections
  - Data diff visualization with filtering
  - Conflict highlighting with resolution status
  - SQL migration preview with syntax highlighting
- How to use the PDF export feature

## Features

### Schema Diff Viewer

- Collapsible table sections showing all changes
- Color-coded change types (added, removed, modified)
- Column-level details with type information
- Index and foreign key change tracking

### Data Diff Visualization

- Table-based view of all data changes
- Filter by table name
- Filter by change type (added, removed, updated)
- Expandable key details per table

### Conflict Highlighting

- Visual list of all conflicts
- Hash comparison display
- Resolution status badges
- Filter by table

### SQL Migration Preview

- Syntax-highlighted SQL code
- Copy to clipboard button
- Full migration script display

### PDF Export

- Export via browser print (Ctrl+P / Cmd+P)
- Print-optimized CSS styling
- All sections visible in print view

## Scenario

This sample uses an e-commerce database to demonstrate various types of changes:

| Change Type | Description |
|------------|-------------|
| Schema - Added Column | `products.discount_percent` column added |
| Schema - Modified Column | `customers.email` changed from varchar(100) to varchar(255) |
| Schema - Added Index | Index on `orders.customer_id` |
| Data - Added Rows | New products and orders in dev |
| Data - Removed Rows | Some audit logs removed |
| Data - Updated Rows | Price and inventory changes |
| Conflicts | Customer tier upgrades conflict |

## Prerequisites

- Docker and Docker Compose installed
- DeepDiff DB installed locally

## Setup

### 1. Start the Databases

```bash
cd samples/13-html-report-viewer
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

## Usage Examples

### Example 1: Generate HTML Report with Diff

Run a full diff and generate an HTML report:

```bash
deepdiffdb diff --config deepdiffdb.config.yaml --html
```

**Output:**

```
Schema OK. Data diff complete.
Changes detected. See ./diff-output/content_diff.json, ./diff-output/conflicts.json, and ./diff-output/summary.txt
Warning: 5 conflicts detected. Review ./diff-output/conflicts.json
HTML report generated: ./diff-output/report.html
```

Open `diff-output/report.html` in your browser to view the interactive report.

### Example 2: Generate HTML Report with Migration Pack

Generate a migration pack with an HTML report:

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml --html
```

This generates the same HTML report but includes the SQL migration preview tab with syntax highlighting.

### Example 3: View the Report

Open the HTML report in your browser:

```bash
# macOS
open diff-output/report.html

# Linux
xdg-open diff-output/report.html

# Windows
start diff-output/report.html
```

## Report Navigation

### Header Section

The header displays:
- Report generation timestamp
- Production database info
- Development database info
- DeepDiff DB version

### Summary Cards

Six summary cards show key metrics:
- Schema Status (OK or DRIFT)
- Tables Scanned
- Rows Added
- Rows Removed
- Rows Updated
- Conflicts

### Tab Navigation

Four tabs organize the content:

1. **Schema Diff** - Shows all schema changes
2. **Data Diff** - Shows row-level changes with filtering
3. **Conflicts** - Shows conflicting rows
4. **SQL Migration** - Shows the generated SQL (gen-pack only)

### Interactive Features

- **Collapsible Sections**: Click table headers to expand/collapse details
- **Filters**: Use dropdowns to filter by table or change type
- **View Keys**: Click "View Keys" button to see affected primary keys
- **Copy SQL**: Click "Copy" to copy migration SQL to clipboard
- **Export PDF**: Click "Export to PDF" to open print dialog

## Expected Report Contents

### Schema Diff Tab

```
[added_table] new_audit_log
  Table 'new_audit_log' exists in dev but not in prod

[modified] products
  Column Changes:
    + discount_percent (decimal(5,2) NULL)

[modified] customers
  Column Changes:
    ~ email: varchar(100) -> varchar(255)

[modified] orders
  Index Changes:
    + idx_customer_id (columns: [customer_id])
```

### Data Diff Tab

| Table | Added | Removed | Updated |
|-------|-------|---------|---------|
| products | +3 | - | ~2 |
| orders | +5 | - | ~3 |
| customers | - | - | ~4 |
| audit_log | - | -10 | - |

### Conflicts Tab

```
customers | Key: 1 | abc123... vs xyz789... | [Pending]
customers | Key: 2 | def456... vs uvw012... | [Pending]
products  | Key: 5 | ghi789... vs rst345... | [Keep Prod]
```

### SQL Migration Tab

```sql
-- DeepDiff DB Migration Pack
-- Generated: 2026-01-03 10:00:00
-- Driver: mysql

BEGIN;

-- Schema changes
ALTER TABLE `products` ADD COLUMN `discount_percent` decimal(5,2) NULL;
ALTER TABLE `customers` MODIFY COLUMN `email` varchar(255) NOT NULL;
CREATE INDEX `idx_customer_id` ON `orders` (`customer_id`);

-- Data changes
INSERT INTO `products` (`id`, `name`, `price`) VALUES
  (101, 'New Widget', 29.99),
  (102, 'Super Gadget', 49.99);

UPDATE `products` SET `price` = 24.99 WHERE `id` = 1;

COMMIT;
```

## PDF Export

To export the report as PDF:

1. Click the "Export to PDF" button in the report
2. Or press Ctrl+P (Windows/Linux) or Cmd+P (macOS)
3. Select "Save as PDF" as the destination
4. Click Save

The print stylesheet ensures:
- All tabs are visible (not just the active one)
- Tab headers are hidden
- Section titles are added for clarity
- SQL code is not truncated

## Cleanup

```bash
docker-compose down -v
```

## Files in This Sample

- `README.md` - This documentation
- `docker-compose.yml` - Database containers configuration
- `deepdiffdb.config.yaml` - DeepDiff DB configuration
- `init-scripts/01-prod-schema.sql` - Production database schema and data
- `init-scripts/02-dev-schema.sql` - Development database schema and data
- `diff-output/` - Generated reports (created when you run the tool)
  - `report.html` - Interactive HTML report (when using `--html`)

## Related Samples

- [Sample 02: Advanced Migrations](../02-advanced-migrations/) - Full workflow example
- [Sample 12: Interactive Resolution](../12-interactive-resolution/) - Conflict resolution
- [Sample 03: Schema Migrations](../03-schema-migrations/) - Schema change handling

---

**DeepDiff DB - Smart Database Comparison Tool**
