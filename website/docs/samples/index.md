---
sidebar_position: 1
---

# Samples

The `samples/` directory in the repository contains 17 self-contained example projects. Each sample demonstrates a specific feature or database workflow with a working configuration, seed data, and ready-to-run scripts.

## All Samples

| # | Name | Description | Databases | Docker |
|---|---|---|---|---|
| 01 | [basic-diff](https://github.com/iamvirul/deepdiff-db/tree/main/samples/01-basic-schema-drift) | Basic schema drift detection and data diff with MySQL | MySQL | Yes |
| 02 | [postgres-diff](https://github.com/iamvirul/deepdiff-db/tree/main/samples/02-advanced-migrations) | PostgreSQL diff with advanced migration scenarios | PostgreSQL | Yes |
| 03 | [sqlite-diff](https://github.com/iamvirul/deepdiff-db/tree/main/samples/03-schema-migrations) | SQLite diff with full schema migration workflow | SQLite | No |
| 04 | [drop-column-safety](https://github.com/iamvirul/deepdiff-db/tree/main/samples/04-drop-column-safety) | DROP COLUMN with safety controls and allow flags | SQLite | No |
| 05 | [modify-column](https://github.com/iamvirul/deepdiff-db/tree/main/samples/05-modify-column) | MODIFY COLUMN: type changes, nullability, defaults | SQLite | No |
| 06 | [index-support](https://github.com/iamvirul/deepdiff-db/tree/main/samples/06-index-support) | Index operations: create, drop, unique indexes | SQLite | No |
| 07 | [table-operations](https://github.com/iamvirul/deepdiff-db/tree/main/samples/07-table-operations) | CREATE TABLE and DROP TABLE detection | SQLite | No |
| 08 | [foreign-key-support](https://github.com/iamvirul/deepdiff-db/tree/main/samples/08-foreign-key-support) | Foreign key add, drop, and constraint handling | SQLite | No |
| 09 | [dependency-ordering](https://github.com/iamvirul/deepdiff-db/tree/main/samples/09-dependency-ordering) | Correct migration ordering for FK dependencies | SQLite | No |
| 10 | [conflict-detection](https://github.com/iamvirul/deepdiff-db/tree/main/samples/10-conflict-resolution) | Conflict detection: same PK, different values | SQLite | No |
| 11 | [resolution-engine](https://github.com/iamvirul/deepdiff-db/tree/main/samples/11-resolution-engine) | Conflict resolution engine with ours/theirs strategies | SQLite | No |
| 12 | [interactive-resolution](https://github.com/iamvirul/deepdiff-db/tree/main/samples/12-interactive-resolution) | Interactive `resolve-conflicts` command walkthrough | SQLite | No |
| 13 | [html-report-viewer](https://github.com/iamvirul/deepdiff-db/tree/main/samples/13-html-report-viewer) | HTML report generation with `--html` flag | SQLite | No |
| 14 | [streaming-large-datasets](https://github.com/iamvirul/deepdiff-db/tree/main/samples/14-streaming-large-datasets) | Large dataset streaming with batch-size and parallel flags | SQLite | No |
| 15 | [mssql-support](https://github.com/iamvirul/deepdiff-db/tree/main/samples/15-mssql-support) | Full MSSQL workflow: schema diff, data diff, gen-pack, apply | MSSQL | Yes |
| 16 | [oracle-support](https://github.com/iamvirul/deepdiff-db/tree/main/samples/16-oracle-support) | Full Oracle workflow: schema drift, data diff, gen-pack, apply | Oracle | Yes |
| 17 | [git-like-versioning](https://github.com/iamvirul/deepdiff-db/tree/main/samples/17-git-like-versioning) | Git-like versioning: commit snapshots, browse history, compare versions, generate rollback SQL | MySQL | Yes |

## Running a SQLite Sample

SQLite samples have no external dependencies. From the project root:

```bash
cd samples/03-schema-migrations
make diff
make gen-pack
```

## Running a Docker Sample

MySQL, PostgreSQL, MSSQL, and Oracle samples require Docker and Docker Compose.

```bash
# MySQL example
cd samples/01-basic-schema-drift
make up        # start containers
make seed      # load schema and data
make diff      # run deepdiffdb diff
make gen-pack  # generate migration pack
make down      # stop and remove containers
```

## Sample 14: Streaming Large Datasets

Sample 14 includes a Go seed script that generates 500,000 orders, 100,000 products, and 200,000 audit log rows in SQLite to demonstrate the performance impact of batch size and parallelism settings:

```bash
cd samples/14-streaming-large-datasets
make seed           # generate ~800k rows
make diff           # hash with default settings (batch=10000, parallel=1)
make diff-fast      # hash with batch=5000, parallel=4
make diff-sequential # hash with batch=0 (full scan, pre-v0.7 behaviour)
```

## Sample 16: Oracle Support

Sample 16 starts two Oracle XE 21c containers (prod on port 1521, dev on port 1522) using the `gvenzl/oracle-xe:21-slim-faststart` image and seeds them with schema and data drift:

```bash
cd samples/16-oracle-support
make up            # start Oracle containers (takes ~60s for XE to initialise)
make wait-healthy  # wait for both containers to be ready
make seed          # run SQLPlus init scripts
make diff          # run deepdiffdb diff
make gen-pack      # generate migration pack
make down          # stop containers
```

## Sample 17: Git-like Versioning

Sample 17 demonstrates the full `version` command workflow using two MySQL 8 containers. It walks through three sprint cycles — baseline, category linking, and a reviews feature — and shows how to inspect history, compare schema evolution between any two commits, and generate rollback SQL.

```bash
cd samples/17-git-like-versioning

# Automated end-to-end demo (starts containers, commits 3 versions, shows log/diff/rollback)
bash scripts/demo.sh

# Or run manually step by step:
docker-compose up -d
deepdiffdb version init
deepdiffdb version commit --config deepdiffdb.config.yaml --message "V1: baseline"
# ... apply migration scripts, commit again ...
deepdiffdb version log
deepdiffdb version diff <hash_v1> <hash_v3>
deepdiffdb version rollback --out diff-output/rollback_v3.sql <hash_v3>
docker-compose down -v
```
