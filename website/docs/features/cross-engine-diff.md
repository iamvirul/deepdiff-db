---
sidebar_position: 1
---

# Cross-Engine Data Diff & Verification

DeepDiff DB supports **cross-engine data diffing and migration verification**, allowing you to compare schemas and row data between completely different database engines (such as **PostgreSQL $\leftrightarrow$ Oracle**, **MySQL $\leftrightarrow$ PostgreSQL**, or **SQLite $\leftrightarrow$ MSSQL**).

![Live Migration Pipeline Mismatch](/img/cross-engine-pipeline-mismatch.jpg)

---

## The Cross-Engine Challenge: Identifier Case Folding

When migrating data between database engines (for example, in an enterprise ETL pipeline migrating from PostgreSQL to Oracle), automated row-count and checksum verification frequently fails or reports false positives:

```text
TOTAL RECORD COUNTS (entire database)
Source (PG)            : 12,923
Oracle                 : 19,132
Delta (Oracle - Source): +6,209

MISMATCHED TABLES:
TABLE                  PG      ORACLE   DELTA
------------------------------------------------
UM_USER_ATTRIBUTE      12281   18403    +6122
UM_USER                640     727      +87
```

### Why Naive Comparisons Fail

1. **Identifier Folding Discrepancies**:
   - **PostgreSQL** folds unquoted identifiers to **lowercase** (`um_user_attribute`, `um_user`).
   - **Oracle** folds unquoted identifiers to **UPPERCASE** (`UM_USER_ATTRIBUTE`, `UM_USER`).
   - **MySQL** on Linux defaults to case-sensitive table names, while on Windows/macOS it is case-insensitive.
   - Traditional tools comparing by exact table name (`"um_user" == "UM_USER"`) find 0 matching tables and report `"No differences found"`, completely missing thousands of drifted rows!

2. **Column Quoting and Dialect Differences**:
   - Querying a PostgreSQL table requires double quotes for mixed-case or lowercase identifiers: `SELECT "id", "email" FROM "customers"`.
   - Querying Oracle requires uppercase or quoted identifiers: `SELECT "ID", "EMAIL" FROM "CUSTOMERS"`.
   - SQLite and MySQL use backticks or brackets.

---

## How DeepDiff DB Solves This

DeepDiff DB introduces **engine-aware canonical identifier folding** and **dialect-specific query drivers**:

```text
┌─────────────────────────┐               ┌─────────────────────────┐
│  PostgreSQL (Source)    │               │     Oracle (Target)     │
│  "um_user" (lowercase)  │               │  "UM_USER" (UPPERCASE)  │
└────────────┬────────────┘               └────────────┬────────────┘
             │                                         │
             ▼                                         ▼
   Canonical: "um_user"   ◄──────────────────►   Canonical: "um_user"
                          (Matched & Paired)
                                   │
                                   ▼
             ┌─────────────────────────────────────────┐
             │       Keyset-Paginated Row Hashing       │
             │  PostgreSQL driver: SELECT "id"...      │
             │  Oracle driver:     SELECT "ID"...      │
             └─────────────────────┬───────────────────┘
                                   │
                                   ▼
             ┌─────────────────────────────────────────┐
             │       Side-by-Side Column Diffing       │
             │     email: dan@example.com (*)          │
             │     email: dan+new@example.com          │
             └─────────────────────────────────────────┘
```

1. **Case-Insensitive Table Discovery**: DeepDiff matches tables across engines using case-folded canonical names (`schema.CanonicalIdent`).
2. **Preserved Dialect Execution**: Each database connection uses its own driver and dialect rules, ensuring queries against PostgreSQL use correct lowercase identifiers while queries against Oracle use uppercase identifiers.
3. **Column-Level Value Alignment**: Columns are aligned canonically, so `email` in PostgreSQL is compared directly with `EMAIL` in Oracle regardless of catalog casing.

---

## Inspecting Changed Rows (`row-diff`)

Once table differences or conflicts are detected, use the `row-diff` command (or pass `--row-diff` to `diff`) to inspect exactly what changed at the column level:

![row-diff Terminal Output](/img/row-diff-terminal.svg)

```bash
# Compare PostgreSQL against Oracle with column-level row inspection
deepdiffdb diff --config deepdiffdb.config.yaml --row-diff

# Or inspect a specific table and primary key directly
deepdiffdb row-diff --table customers --key 4
```

Terminal output highlights differing columns with an asterisk (`*`):

```text
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Table: customers | Key: 4 (updated)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Column               | Production (PostgreSQL)  | Development (Oracle)    
  ---------------------+--------------------------+--------------------------
  country              | FR                       | FR                      
  email                | dan@example.com          | dan+new@example.com      *
  id                   | 4                        | 4                       
  name                 | Dan                      | Dan                     

  * indicates columns that differ
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## Interactive HTML Report

Generating an HTML report (`deepdiffdb diff --html`) provides an interactive side-by-side inspection view:

- **Data Changes Tab**: Opens automatically when data drift exists. Clicking any table displays both key summaries and the complete column comparison table for every modified row.
- **Row Data Tab**: Provides a dedicated card view of every changed row across all tables with real-time table filtering.
- **Conflicts Tab**: Displays conflicting keys with source and target hash values, resolution strategy, and side-by-side column values.

```bash
deepdiffdb diff --config deepdiffdb.config.yaml --html
open diff-output/report.html
```

---

## Ready-to-Run Sample

Check out **[Sample 18: Cross-Engine Row Diff & Verification](https://github.com/iamvirul/deepdiff-db/tree/main/samples/18-cross-engine-row-diff)** in the repository, which sets up PostgreSQL and Oracle containers with pre-seeded data mismatches to test cross-engine verification.
