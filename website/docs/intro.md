---

sidebar_position: 1
---

# DeepDiff DB

DeepDiff DB is a high-performance Go CLI tool for comparing two databases, detecting schema drift, identifying row-level data differences, and generating safe, reviewable SQL migration packs. It is designed to make the process of synchronising a development database back to production deterministic, auditable, and safe — without ever writing to production until you explicitly tell it to.

## Feature Highlights

- **Multi-database support** — MySQL, PostgreSQL, SQLite, Microsoft SQL Server, and Oracle Database
- **Schema drift detection** — Detects added/removed tables, column type changes, nullability, defaults, indexes, foreign keys, views, stored procedures, functions, triggers, and sequences (PostgreSQL)
- **Row-level diffing** — SHA-256 hashing per row for accurate change detection across any table size
- **Conflict detection** — Identifies rows that exist in both databases with differing values, before any write
- **Safe migration generation** — Produces transactional SQL packs reviewable before apply; destructive operations are off by default
- **Git-like versioning** — Commit schema+data diff snapshots, browse history, compare any two commits, generate rollback SQL, and manage parallel lines of schema evolution through branches (`version init/commit/log/diff/rollback/branch/checkout/tree`)
- **Streaming large tables** — Keyset-paginated batch hashing keeps heap flat at O(batch\_size) regardless of row count (v0.7+)
- **Parallel table hashing** — Hash multiple tables concurrently with a configurable worker pool
- **Interactive conflict resolution** — Side-by-side row comparison with per-conflict decisions or auto strategies
- **HTML reports** — Tab-based interactive report with schema diff, data diff, conflict view, and SQL preview
- **Checkpoint and resume** — Survive network interruptions; resume pack generation or apply from last saved state
- **Single static binary** — Zero runtime dependencies; works on Linux, macOS, and Windows

## Supported Databases

| Database | Driver value | Go driver |
|---|---|---|
| MySQL | `mysql` | `github.com/go-sql-driver/mysql` |
| PostgreSQL | `postgres` / `postgresql` | `github.com/lib/pq` |
| SQLite | `sqlite` | `modernc.org/sqlite` (pure Go) |
| Microsoft SQL Server | `mssql` | `github.com/microsoft/go-mssqldb` |
| Oracle Database | `oracle` | `github.com/sijms/go-ora/v2` (pure Go) |

## Install

**Homebrew (macOS / Linux):**

```bash
brew tap iamvirul/deepdiff-db
brew install deepdiff-db
```

**Binary download:**

Download the latest pre-compiled binary for your platform from the [GitHub Releases](https://github.com/iamvirul/deepdiff-db/releases) page, then move it to a directory on your `PATH`.

```bash
# Linux amd64 example
wget https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb-linux-amd64
chmod +x deepdiffdb-linux-amd64
sudo mv deepdiffdb-linux-amd64 /usr/local/bin/deepdiffdb
```

**Verify:**

```bash
deepdiffdb --version
```

## Next Steps

Head to the [Quick Start](/docs/getting-started/quick-start) guide to run your first diff in under five minutes, or read the [Installation](/docs/getting-started/installation) page for all install methods.
