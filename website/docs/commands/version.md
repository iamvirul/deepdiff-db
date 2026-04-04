---
sidebar_position: 9
---

# version

Git-like versioning for database diffs. Stores schema and data diff snapshots as commits so you can browse history, compare any two points in time, generate rollback SQL, and manage parallel lines of schema evolution through branches — all without a live database connection after the commit is taken.

## Overview

```bash
deepdiffdb version <subcommand> [flags]
```

| Subcommand | Description |
|---|---|
| [`version init`](#version-init) | Initialise a `.deepdiffdb/` repository in the current directory |
| [`version commit`](#version-commit) | Run a full diff and store the result as a new commit |
| [`version log`](#version-log) | Show commit history from newest to oldest |
| [`version diff`](#version-diff) | Compare schema evolution between two commits |
| [`version rollback`](#version-rollback) | Generate rollback SQL to undo the changes in a commit |
| [`version branch`](#version-branch) | List branches, or create a new one |
| [`version checkout`](#version-checkout) | Switch to a branch |
| [`version tree`](#version-tree) | Show ASCII commit graph for all branches |

---

## version init

Creates the `.deepdiffdb/` directory structure in the current directory. Idempotent — safe to run on an existing repo.

### Usage

```bash
deepdiffdb version init
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Directory to initialise |
| `--skip-auth` | `false` | Skip the GitHub authentication prompt |

### GitHub Authentication

After initialising the repo, `version init` prompts you to authenticate with GitHub using the **device flow** (no browser redirect required):

```
Authenticate with GitHub to verify commit authorship? [Y/n]:

  Open:  https://github.com/login/device
  Code:  ABCD-1234

  Waiting for authorization ........ ✓
Authenticated as github:iamvirul — your commits will be signed automatically.
```

The verified username is stored in `.deepdiffdb/config` as `{"github_user": "iamvirul"}`. The access token is used only to confirm identity and is **never written to disk**.

Use `--skip-auth` to bypass the prompt in CI pipelines or scripted environments — and use `--author` on `version commit` instead.

:::info Setting up the OAuth App
`version init` requires `DEEPDIFFDB_GITHUB_CLIENT_ID` to be set (or baked into the binary at build time). See the [GitHub OAuth setup guide](#github-oauth-setup) below.
:::

### What it creates

```
.deepdiffdb/
  HEAD                  ← symbolic ref: "ref: refs/heads/main"
  config                ← verified identity: {"github_user": "..."}  (0o600)
  objects/              ← zlib-compressed commit objects (Git fanout layout)
  refs/
    heads/
      main              ← tip hash for the main branch (created on first commit)
```

### Example

```bash
cd my-project
deepdiffdb version init
# Initialised version repository in ./.deepdiffdb
# Authenticate with GitHub to verify commit authorship? [Y/n]: y
# ...
# Authenticated as github:iamvirul

# Skip auth (CI)
deepdiffdb version init --skip-auth
```

---

## version commit

Connects to both databases, runs a full schema and data diff (identical to the `diff` command), and stores the result as a new commit object on the **currently checked-out branch**.

Each commit records:
- Author, message, timestamp
- The full `schema.DiffResult` and `content.DataDiff`
- Both `ProdSchema` and `DevSchema` snapshots (used for offline rollback generation)
- A SHA-256 hash derived from the content (Git-style `"commit <size>\x00<json>"` header)

### Usage

```bash
deepdiffdb version commit --message "describe the change" [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--config` | `deepdiffdb.config.yaml` | Path to configuration file |
| `--message` | _(required)_ | Commit message |
| `--author` | `$USER` | Author name — **ignored** when GitHub identity is stored in `.deepdiffdb/config` |
| `--verbose` | `false` | Enable debug-level logging |
| `--log-file` | _(none)_ | Write logs to file |
| `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | `text` | Log format: `text` or `json` |

### Author resolution

The author is resolved in this order:

1. **Verified GitHub identity** — if `.deepdiffdb/config` contains a `github_user`, the author is set to `github:<username>` automatically. Passing `--author` alongside a stored identity prints a warning and the flag is ignored.
2. **`--author` flag** — used when no config identity exists.
3. **`$USER` environment variable** — fallback when neither is set.
4. **Error** — if none of the above are available, `version commit` exits with an error asking you to authenticate or pass `--author`.

### Example

```bash
# With GitHub authentication (author set automatically)
deepdiffdb version commit \
  --config deepdiffdb.config.yaml \
  --message "V2: add category_id FK and customer_email column"

# [e35c16c9] V2: add category_id FK and customer_email column
#   Author: github:iamvirul
#   Schema drift detected.
#   Data differences detected.

# Without authentication (explicit author)
deepdiffdb version commit \
  --config deepdiffdb.config.yaml \
  --message "V2: add category_id FK and customer_email column" \
  --author "Alice"
```

### Commit object location

Objects are stored using Git's 2-character fanout layout to avoid directory entry limits:

```
.deepdiffdb/objects/<first-2-chars-of-hash>/<remaining-62-chars>
```

---

## version log

Walks the commit chain from `HEAD` backwards, printing each commit's hash, author, date, and message. Schema drift and data change markers are appended when present.

### Usage

```bash
deepdiffdb version log [--limit N]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--limit` | `20` | Maximum number of commits to display |

### Example output

```
commit 142b1305b83fa114aca634ae15b5d807552a443b94462154f470ba6011bc15ed
Author: Bob
Date:   2026-03-31 14:33:29 +0000

    V3: add reviews table and avg_rating column [schema drift] [data changes]

commit e35c16c95a306e52b2044f4beb3fd7de90ce5f46a5fd46487d8fbf11796020f1
Author: Alice
Date:   2026-03-31 14:33:22 +0000

    V2: add category_id FK and customer_email column [schema drift] [data changes]

commit cad39fda5cdefb3e18047e8615b542a6e39a7d3261432cc17cf535f59fa3591a
Author: Alice
Date:   2026-03-31 14:33:05 +0000

    V1: baseline e-commerce schema
```

---

## version diff

Compares the dev schema snapshots of two commits and reports the schema evolution between them — what tables and columns were added, removed, or modified.

### Usage

```bash
deepdiffdb version diff <hash1> <hash2>
```

Full hashes and short hashes (first 8+ characters) are both accepted.

### Example

```bash
deepdiffdb version diff cad39fda 142b1305
```

```
Schema evolution from cad39fda (V1: baseline e-commerce schema) → 142b1305 (V3: add reviews table and avg_rating column)

  + TABLE reviews (added)
  ~ TABLE orders
      + COLUMN customer_email varchar(255)
  ~ TABLE products
      + COLUMN avg_rating decimal(3,2)
      + COLUMN category_id int
      + INDEX fk_product_category
```

Legend:
- `+` — added (new table, column, or index)
- `-` — removed
- `~` — modified (table has changes inside)

---

## version rollback

Generates a SQL script that undoes the changes recorded in a commit. It inverts the stored diff (dev → prod direction) and passes it through the same migration generator used by `schema-migrate`, so all safety defaults apply:

- Destructive operations (`DROP TABLE`, `DROP COLUMN`, `DROP INDEX`) are **commented out** by default
- Output is wrapped in a `BEGIN` / `COMMIT` transaction
- Proper FK dependency ordering is applied

Requires no live database connection.

### Usage

```bash
# Print rollback SQL to stdout
deepdiffdb version rollback <hash>

# Write to file
deepdiffdb version rollback --out rollback.sql <hash>

# Override the SQL dialect
deepdiffdb version rollback --driver postgres <hash>
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--out` | _(stdout)_ | Write SQL to this file instead of printing |
| `--driver` | _(from commit)_ | Override the database driver (`mysql`, `postgres`, `sqlite`, `mssql`, `oracle`) |

### Example

```bash
deepdiffdb version rollback --out rollback_v3.sql 142b1305
# Rollback SQL written to rollback_v3.sql
```

```sql
BEGIN;

-- Schema Migration Script
-- Generated by DeepDiff DB for mysql

-- DROP TABLES (present in prod but not in dev)
-- WARNING: These operations will delete data!
-- DROP TABLE `reviews`;

-- Table: orders
-- DROP COLUMNS (present in prod but not in dev)
-- ALTER TABLE `orders` DROP COLUMN `customer_email`;

-- Table: products
-- ALTER TABLE `products` DROP COLUMN `avg_rating`;
-- ALTER TABLE `products` DROP COLUMN `category_id`;

COMMIT;
```

Uncomment the statements you want to apply, review, and execute against the target database.

---

## version branch

Lists all branches in the repository, or creates a new one.

### Usage

```bash
# List branches
deepdiffdb version branch

# Create a new branch from current HEAD
deepdiffdb version branch <name>

# Create a new branch from a specific commit
deepdiffdb version branch <name> --from <hash>
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--from` | _(current HEAD)_ | Hash to branch from |

### Example

```bash
# After committing V1 on main, create a feature branch
deepdiffdb version branch feature

# Branch list shows current branch with *
deepdiffdb version branch
#   feature 18bf631b
# * main    18bf631b

# Create a hotfix branch branching off an older commit
deepdiffdb version branch hotfix --from cad39fda
```

---

## version checkout

Switches HEAD to point to the named branch. All subsequent `version commit` calls advance only the checked-out branch tip — other branches remain at their last commit.

### Usage

```bash
deepdiffdb version checkout <branch>
```

### Example

```bash
deepdiffdb version checkout feature
# Switched to branch "feature".

# Commits now go to feature branch
deepdiffdb version commit --message "V2: experimental index" --author "Bob"

# Switch back to main
deepdiffdb version checkout main
# Switched to branch "main".
```

:::note
Returns an error if the branch does not exist. Create it first with `version branch <name>`.
:::

---

## version tree

Renders an ASCII commit graph showing all branches, their lane columns, HEAD decoration, short hash, date, and message — newest commit at the top.

### Usage

```bash
deepdiffdb version tree
```

### Example output

```
| * e2a3d002 (feature)        2026-03-31  V2: category link + customer email
* | 761e26c5 (HEAD -> main)   2026-03-31  V3: reviews + avg_rating
* | 18bf631b                  2026-03-31  V1: baseline schema
```

Each column in the graph represents a branch lane:
- `*` — the commit belongs to this lane's branch
- `|` — this lane is active but the commit is on another branch
- `(HEAD -> main)` — the current branch and its latest commit
- `(feature)` — a non-current branch tip

---

## Storage Layout

```
.deepdiffdb/
  HEAD                          ← symbolic ref: "ref: refs/heads/<branch>"
  config                        ← verified identity {"github_user": "..."} (0o600)
  objects/
    <2-hex>/
      <62-hex>                  ← zlib-compressed commit object (Git fanout)
  refs/
    heads/
      main                      ← tip hash for the main branch
      feature                   ← tip hash for the feature branch
      <branch>                  ← one file per branch
```

Each commit object is self-contained — it includes all diff results and schema snapshots, so the entire history is portable. Commit the `.deepdiffdb/` directory to share history with your team, or add it to `.gitignore` to keep it local.

:::tip
Add `.deepdiffdb/config` to your `.gitignore` — it contains your personal identity and should not be shared.
:::

---

## GitHub OAuth Setup

To enable author verification, register a GitHub OAuth App:

1. Go to [github.com/settings/applications/new](https://github.com/settings/applications/new)
2. Fill in:
   - **Application name:** `DeepDiff DB`
   - **Homepage URL:** `https://github.com/iamvirul/deepdiff-db`
   - **Authorization callback URL:** `http://localhost` _(not used by device flow)_
3. Tick **Enable Device Flow**
4. Click **Register application** and copy the **Client ID**

Then set the client ID in one of two ways:

**Environment variable (personal use):**
```bash
# ~/.zshrc or ~/.bashrc
export DEEPDIFFDB_GITHUB_CLIENT_ID="your_client_id"
```

**Baked into released binaries (build-time injection):**
```yaml
# In .github/workflows/release.yml
- name: Build
  run: |
    go build \
      -ldflags "-X github.com/iamvirul/deepdiff-db/internal/version.GitHubClientID=${{ secrets.DEEPDIFFDB_GITHUB_CLIENT_ID }}" \
      -o deepdiffdb \
      ./cmd/deepdiffdb
```

Add `DEEPDIFFDB_GITHUB_CLIENT_ID` as a repository secret in **Settings → Secrets and variables → Actions**.

---

## Typical Workflow

```bash
# 1. Initialise once — authenticate with GitHub for verified authorship
deepdiffdb version init
# → prompts for GitHub device flow auth
# → stores github:iamvirul in .deepdiffdb/config

# 2. Commit baseline (prod == dev) — author resolved automatically
deepdiffdb version commit \
  --config deepdiffdb.config.yaml \
  --message "V1: baseline e-commerce schema"

# 3. Create a feature branch for experimental schema work
deepdiffdb version branch feature
deepdiffdb version checkout feature

# 4. Developer applies schema changes to dev database
#    (ALTER TABLE, CREATE TABLE, etc.)

# 5. Commit the drift on the feature branch
deepdiffdb version commit \
  --config deepdiffdb.config.yaml \
  --message "V2: add reviews table and avg_rating column"

# 6. Switch back to main for a production hotfix
deepdiffdb version checkout main
deepdiffdb version commit \
  --config deepdiffdb.config.yaml \
  --message "V2: hotfix — drop unused index"

# 7. Visualise the full branch history
deepdiffdb version tree

# 8. See exactly what changed between two commits
deepdiffdb version diff <hash_v1> <hash_v2>

# 9. Generate rollback SQL for a commit
deepdiffdb version rollback --out rollback_v2.sql <hash_v2>
```

---

## See Also

- [Sample 17: Git-like Versioning](https://github.com/iamvirul/deepdiff-db/tree/main/samples/17-git-like-versioning) — full end-to-end demo with MySQL containers, branches, and ASCII tree
- [`diff`](./diff.md) — the underlying diff command used by `version commit`
- [`schema-migrate`](./schema-migrate.md) — standalone schema migration (same SQL generator as `version rollback`)
