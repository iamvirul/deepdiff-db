---
sidebar_position: 2
---

# HTML Reports

DeepDiff DB can generate a self-contained, interactive HTML report alongside the JSON and text outputs.

## Live Sample

**[→ View Sample Report](pathname:///samples/report.html)**

The sample report is generated from two real MySQL databases with intentional schema drift: added/removed/modified views, routines, and triggers, plus table-level column and index changes with data conflicts.

## Generating a Report

Pass `--html` to either `diff` or `gen-pack`:

```bash
# Diff with HTML report
deepdiffdb diff --config deepdiffdb.config.yaml --html

# Gen-pack with HTML report
deepdiffdb gen-pack --config deepdiffdb.config.yaml --html
```

The report is written to `{output.dir}/report.html`. It is a single self-contained file — no internet connection or external assets are needed to open it.

## What the Report Contains

The report is organised into four tabs.

### Schema Diff Tab

Lists every schema object that changed between prod and dev.

**Tables**
- Expandable sections per table for column, index, and foreign key changes
- Each change shows the production value and development value side-by-side

**Views**
- Added, removed, and modified views
- For modified views: shows which property changed (definition, materialized flag)

**Routines**
- Added, removed, and modified stored procedures and functions
- Kind badge (FUNCTION / PROCEDURE) on each entry
- For modified routines: highlights definition, return type, language, or parameter changes

**Triggers**
- Added, removed, and modified triggers
- Shows the associated table for each trigger
- For modified triggers: highlights definition, timing, or event changes

**Sequences** *(PostgreSQL only)*
- Added, removed, and modified sequences
- For modified sequences: shows which numeric properties differ (increment, min/max value, cache, cycle)

### Data Changes Tab

- Summary counts per table: rows added, rows removed, rows updated
- Click any table to expand and view both primary key listings and full **side-by-side column-level row differences** for every modified, added, or removed row
- Table filter dropdown for quick navigation

### Row Data Tab

- Dedicated card view for every differing row across the entire database
- Displays side-by-side columns: **Column name**, **Production value**, and **Development value**
- Asterisks (`*`) and highlighted background colors immediately draw attention to columns that differ
- Filter by table name using the real-time filter dropdown

### Conflicts Tab

- Lists all conflicts grouped by table with SHA-256 hash comparison (`prod_hash → dev_hash`)
- Embedded column comparison table showing exactly which fields differ
- Resolution status badge (`Keep Source`, `Use Target`, or `Pending`) when resolutions are applied
- Per-table resolution strategy breakdown table

### SQL Migration Preview Tab

- The full content of `migration_pack.sql` rendered with SQL syntax highlighting
- A copy-to-clipboard button
- Statement count summary at the top

## Exporting to PDF

Use your browser's built-in print dialog (`Cmd+P` on macOS, `Ctrl+P` on Windows/Linux) and select **Save as PDF**. The report is styled with print-friendly CSS that hides the navigation and renders each tab section sequentially.

## Notes

- The HTML file is self-contained and can be safely attached to a PR, emailed, or stored in an artifact registry.
- For very large diffs (tens of thousands of changed rows), the Data Diff tab may render slowly in some browsers due to the volume of DOM nodes. Consider using the JSON output files for programmatic processing at that scale.
