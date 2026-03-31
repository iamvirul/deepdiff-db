# Sample 17: Git-like Versioning for DB Diffs

This sample demonstrates DeepDiff DB's versioning system — a Git-like history of database schema and data differences over time.
You can commit snapshots, browse the log, compare any two versions, and generate rollback SQL, all without touching a live database.

## What You'll Learn

- How to initialise a version repository with `version init`
- How to capture a diff snapshot with `version commit`
- How to browse history with `version log`
- How to compare schema evolution between two commits with `version diff`
- How to generate rollback SQL for any commit with `version rollback`

## Scenario

An e-commerce team evolves their `shop` database across three sprints:

| Version | What Changed |
|---------|-------------|
| **V1** | Baseline — `categories`, `products`, `orders` (prod == dev) |
| **V2** | Dev adds `category_id` FK to `products`, `customer_email` to `orders` |
| **V3** | Dev adds `reviews` table and `avg_rating` column to `products` |

At each stage a `version commit` is taken.  The demo then shows how to inspect the history and generate rollback SQL.

## Prerequisites

- Docker and Docker Compose
- DeepDiff DB installed (`deepdiffdb` on PATH, or built from source)

## Quick Start (automated)

```bash
cd samples/17-git-like-versioning
bash scripts/demo.sh
```

The script handles everything: starts databases, runs all three commits, shows the log, diffs, and writes rollback SQL files.

---

## Manual Walkthrough

### 1. Start the databases

```bash
docker-compose up -d
```

Two MySQL 8.0 containers start:

| Container | Port | Role |
|-----------|------|------|
| `vcs_prod_db` | 3320 | Production (frozen at V1) |
| `vcs_dev_db`  | 3321 | Development (evolves V2 → V3) |

Wait until both are healthy (~15 s):

```bash
docker-compose ps
```

### 2. Initialise the version repository

```bash
deepdiffdb version init
```

This creates `.deepdiffdb/` in the current directory (analogous to `.git/`).

### 3. Commit V1 — clean baseline

Both databases are identical at this point.

```bash
deepdiffdb version commit \
  --config deepdiffdb.config.yaml \
  --message "V1: baseline e-commerce schema" \
  --author  "Alice"
```

Expected output:
```
[a1b2c3d4] V1: baseline e-commerce schema
  No differences found (clean snapshot).
```

### 4. Apply V2 schema changes to dev

```bash
docker exec -i vcs_dev_db mysql -uroot -prootpassword shop \
  < migrations/v2_add_category_link_and_customer.sql
```

This adds:
- `products.category_id INT FK → categories.id`
- `orders.customer_email VARCHAR(255)`

### 5. Commit V2

```bash
deepdiffdb version commit \
  --config deepdiffdb.config.yaml \
  --message "V2: link products to categories, capture customer_email on orders" \
  --author  "Alice"
```

Expected output:
```
[e5f6a7b8] V2: link products to categories, capture customer_email on orders
  Schema drift detected.
```

### 6. Apply V3 schema changes to dev

```bash
docker exec -i vcs_dev_db mysql -uroot -prootpassword shop \
  < migrations/v3_add_reviews.sql
```

This adds:
- New table `reviews(id, product_id FK, rating, comment, created_at)`
- `products.avg_rating DECIMAL(3,2)`

### 7. Commit V3

```bash
deepdiffdb version commit \
  --config deepdiffdb.config.yaml \
  --message "V3: add reviews table and avg_rating column" \
  --author  "Bob"
```

### 8. Browse the history

```bash
deepdiffdb version log
```

Sample output:
```
commit e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4
Author: Bob
Date:   2026-03-31 12:05:00 +0000

    V3: add reviews table and avg_rating column [schema drift]

commit a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
Author: Alice
Date:   2026-03-31 12:03:00 +0000

    V2: link products to categories, capture customer_email on orders [schema drift]

commit 0f1e2d3c4b5a6978...
Author: Alice
Date:   2026-03-31 12:00:00 +0000

    V1: baseline e-commerce schema
```

### 9. Compare schema evolution V1 → V3

```bash
deepdiffdb version diff <hash_v1> <hash_v3>
```

Sample output:
```
Schema evolution from a1b2c3d4 (V1: baseline…) → e5f6a7b8 (V3: add reviews…)

  + TABLE reviews (added)
  ~ TABLE orders
      + COLUMN customer_email varchar(255)
  ~ TABLE products
      + COLUMN avg_rating decimal(3,2)
      + COLUMN category_id int
```

### 10. Generate rollback SQL for V3

```bash
deepdiffdb version rollback --out diff-output/rollback_v3.sql <hash_v3>
cat diff-output/rollback_v3.sql
```

The generated script undoes every change introduced between the V3 snapshot and its baseline, wrapped in a transaction.

---

## File Structure

```
17-git-like-versioning/
  docker-compose.yml                          — two MySQL containers
  deepdiffdb.config.yaml                      — prod/dev connection config
  init-scripts/
    init_db1.sql                              — V1 schema for prod (port 3320)
    init_db2.sql                              — V1 schema for dev  (port 3321)
  migrations/
    v2_add_category_link_and_customer.sql     — ALTER statements for V2
    v3_add_reviews.sql                        — CREATE + ALTER for V3
  scripts/
    demo.sh                                   — automated end-to-end demo
  diff-output/                                — rollback SQL files land here
```

After running the demo, committed objects are stored in:
```
.deepdiffdb/
  HEAD                    — symbolic ref: "ref: refs/heads/main"
  objects/
    <2-char>/
      <62-char>           — zlib-compressed commit object (Git fanout layout)
  refs/
    heads/
      main                — tip hash for the main branch
```

## Teardown

```bash
docker-compose down -v
rm -rf .deepdiffdb diff-output/rollback_*.sql
```
