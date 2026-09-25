# Sample 18: Cross-Engine Row Diff & Migration Verification

This sample demonstrates **cross-engine migration verification** between **PostgreSQL 16** (Source / Production) and **Oracle XE 21c** (Target / Development).

It replicates common real-world migration discrepancies where:
1. **Identifier case differs**: PostgreSQL stores unquoted identifiers in lowercase (`um_user`, `um_user_attribute`), whereas Oracle stores them in UPPERCASE (`UM_USER`, `UM_USER_ATTRIBUTE`).
2. **Data drift exists**: Certain rows have modified columns, missing records, or newly inserted rows.
3. **Column-level inspection is needed**: You need to see exactly *which* column drifted (e.g. `email` or `attr_value`) rather than just knowing that a row checksum failed.

---

## Architecture & Scenario

```text
┌─────────────────────────────────┐           ┌─────────────────────────────────┐
│       PostgreSQL 16 (Prod)      │           │        Oracle XE 21c (Dev)      │
│          Port: 55432            │           │            Port: 51521          │
├─────────────────────────────────┤           ├─────────────────────────────────┤
│ um_user (6 rows, lowercase)     │ ◄───────► │ UM_USER (5 rows, UPPERCASE)     │
│ um_user_attribute (7 rows)      │           │ UM_USER_ATTRIBUTE (6 rows)      │
└─────────────────────────────────┘           └─────────────────────────────────┘
                                       │
                                       ▼
                         ┌───────────────────────────┐
                         │        DeepDiff DB        │
                         │ Canonical Case Folding    │
                         │ Column-Level Row Diffing  │
                         └───────────────────────────┘
```

---

## Quick Start

### 1. Start Containers

```bash
make up
make wait-healthy
```

*Note: Oracle XE 21c takes approximately 45–90 seconds to initialize on the first launch.*

### 2. Seed Data

```bash
make seed
```

This script seeds:
- **`um_user` / `UM_USER`**:
  - Key `1`, `2`, `3`: Identical in both databases.
  - Key `4` (`dan`): Email in Postgres is `dan@corp.internal`, but in Oracle is `dan+updated@corp.internal`.
  - Keys `5`, `6` (`eve`, `frank`): Present in Postgres, missing in Oracle.
  - Key `7` (`grace`): Present in Oracle, missing in Postgres.
- **`um_user_attribute` / `UM_USER_ATTRIBUTE`**:
  - Key `104`: Attribute `department` is `'Finance'` in Postgres, but `'Accounting'` in Oracle.
  - Keys `106`, `107`: Missing in Oracle.
  - Key `108`: New in Oracle.

### 3. Run Cross-Engine Diff

```bash
make diff
```

DeepDiff DB canonically pairs `um_user` with `UM_USER` and `um_user_attribute` with `UM_USER_ATTRIBUTE`:

```text
Changes detected. See diff-output/content_diff.json, diff-output/conflicts.json, and diff-output/summary.txt
Warning: 2 conflicts detected. Review diff-output/conflicts.json
```

### 4. Inspect Column-Level Row Differences (`row-diff`)

Run `make row-diff` to view the exact column values side-by-side:

```bash
make row-diff
```

Output:

```text
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Table: um_user | Key: 4 (updated)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Column               | Production (PostgreSQL)  | Development (Oracle)    
  ---------------------+--------------------------+--------------------------
  email                | dan@corp.internal        | dan+updated@corp.internal *
  id                   | 4                        | 4                       
  status               | ACTIVE                   | ACTIVE                  
  username             | dan                      | dan                     

  * indicates columns that differ
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Table: um_user_attribute | Key: 104 (updated)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Column               | Production (PostgreSQL)  | Development (Oracle)    
  ---------------------+--------------------------+--------------------------
  attr_name            | department               | department              
  attr_value           | Finance                  | Accounting               *
  id                   | 104                      | 104                     
  user_id              | 3                        | 3                       

  * indicates columns that differ
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 5. Targeted Key Inspection

To inspect a single record directly without scanning the whole table:

```bash
../../deepdiffdb row-diff --table um_user --key 4
```

### 6. Interactive HTML Report

Generate the HTML viewer with column diffing embedded:

```bash
make html
```

The report opens in your browser at `diff-output/report.html`, featuring:
- **Data Changes Tab**: Click `um_user` or `um_user_attribute` to see the column diff tables right inside the expanded row.
- **Row Data Tab**: Browse all changed records with real-time table filtering.
- **Conflicts Tab**: Inspect conflicting hashes with side-by-side values.

### 7. Teardown

```bash
make down
make clean
```
