# DeepDiff DB

[![codecov](https://codecov.io/gh/iamvirul/deepdiff-db/branch/main/graph/badge.svg?token=Y9IORTUBAH)](https://codecov.io/gh/iamvirul/deepdiff-db)
[![Go Report Card](https://goreportcard.com/badge/github.com/iamvirul/deepdiff-db)](https://goreportcard.com/report/github.com/iamvirul/deepdiff-db)

DeepDiff DB is a high-speed Go CLI tool for comparing two databases, detecting schema drift, identifying data-level differences, and generating safe migration packs that can be applied to production without risking corruption.

It solves the real-world problem where dev backups drift away from production and accidentally overwrite or break real data. DeepDiff DB forces structural validation first, then performs controlled content merging.

---

## Why DeepDiff DB

Teams often:

* restore dev backups
* tweak schema
* modify reference tables
* change configurations
* add or remove rows

…and then try to push the same changes into production.

This usually ends badly.

DeepDiff DB makes the entire process deterministic, reviewable, and safe.

---

## Features

* Fast Go-based diff engine
* Single static binary per OS
* MySQL, PostgreSQL, SQLite support
* Schema drift detection
* Row-level comparison
* SHA-256 hashing for change detection
* Conflict detection
* Auto-generated SQL migration pack
* Dry-run mode
* Fully transactional apply mode
* JSON and text reports
* Configurable ignore lists
* Zero dependencies after download
* Works with DB connections or dump files

---

# Installation

You can install DeepDiff DB in several ways:

---

## Option 1: Local Development Build (Latest Unreleased Version)

For developers who want to test the latest changes without waiting for a release:

**macOS/Linux:**
```bash
# Build and install to ~/bin (no sudo required)
./scripts/build-local.sh --install --install-dir ~/bin

# Or install to /usr/local/bin (requires sudo)
sudo ./scripts/build-local.sh --install

# Or just build without installing (outputs to bin/deepdiffdb)
./scripts/build-local.sh --build-only
```

**Windows (PowerShell):**
```powershell
# Build and install
.\scripts\build-local.ps1 -Install

# Or just build
.\scripts\build-local.ps1 -BuildOnly
```

**Features:**
- ✅ Builds optimized binaries with version information
- ✅ Automatically includes git commit hash in version
- ✅ Handles installation permissions automatically
- ✅ Validates binary after build
- ✅ Cross-platform support (macOS, Linux, Windows)
- ✅ Cross-compilation support (build for different platforms)

**Note:** Make sure `~/bin` is in your PATH to use the binary from anywhere:
```bash
export PATH="$HOME/bin:$PATH"  # Add to ~/.zshrc or ~/.bashrc
```

See [scripts/README.md](scripts/README.md) for more options, examples, and troubleshooting.

---

## Option 2: Download Precompiled Binaries (Recommended for Users)

Each release ships with binaries for:

### Supported Platforms

* Linux (x64, ARM64)
* macOS (Intel, Apple Silicon)
* Windows (x64)

You can download them from:

```
https://github.com/iamvirul/deepdiffdb/releases
```

### Example

```bash
# Linux
wget https://github.com/iamvirul/deepdiffdb/releases/download/v1.0.0/deepdiffdb-linux-amd64
chmod +x deepdiffdb-linux-amd64
sudo mv deepdiffdb-linux-amd64 /usr/local/bin/deepdiffdb
```

```powershell
# Windows
deepdiffdb-windows-amd64.exe
```

```bash
# macOS (Apple Silicon)
wget https://github.com/iamvirul/deepdiffdb/releases/download/v1.0.0/deepdiffdb-darwin-arm64
chmod +x deepdiffdb-darwin-arm64
sudo mv deepdiffdb-darwin-arm64 /usr/local/bin/deepdiffdb
```

After that `deepdiffdb` is ready to use.

---

## Option 3: Install via `go install`

If you want to build from source using Go's install command:

```bash
go install github.com/iamvirul/deepdiff-db/cmd/deepdiffdb@latest
```

This produces a static binary in your GOPATH/bin.

---

# Building Binaries (For Maintainers)

To ship releases, you can build all binaries with Go cross-compilation.

## Build all targets:

```bash
make build-all
```

If you don’t use Make, here are direct commands:

### Linux x64

```bash
GOOS=linux GOARCH=amd64 go build -o bin/deepdiffdb-linux-amd64 ./cmd/deepdiffdb
```

### Linux ARM64

```bash
GOOS=linux GOARCH=arm64 go build -o bin/deepdiffdb-linux-arm64 ./cmd/deepdiffdb
```

### macOS Intel

```bash
GOOS=darwin GOARCH=amd64 go build -o bin/deepdiffdb-darwin-amd64 ./cmd/deepdiffdb
```

### macOS Apple Silicon

```bash
GOOS=darwin GOARCH=arm64 go build -o bin/deepdiffdb-darwin-arm64 ./cmd/deepdiffdb
```

### Windows x64

```bash
GOOS=windows GOARCH=amd64 go build -o bin/deepdiffdb-windows-amd64.exe ./cmd/deepdiffdb
```

All binaries are statically compiled and require no additional runtime or dependencies.

---

# Configuration

DeepDiff DB uses a YAML config file to define both database connections.

### deepdiffdb.config.yaml

```yaml
prod:
  driver: "mysql"
  host: "localhost"
  port: 3306
  user: "root"
  password: "password"
  database: "prod_db"

dev:
  driver: "mysql"
  host: "localhost"
  port: 3306
  user: "root"
  password: "password"
  database: "dev_db"

ignore:
  tables:
    - "logs"
    - "audit"
  columns:
    - "*.updated_at"

output:
  dir: "./diff-output"
```

You can override values using CLI flags if needed.

An example config is included at `deepdiffdb.config.yaml.example`.

---

# Commands

## Schema Diff

```bash
deepdiffdb schema-diff --config deepdiffdb.config.yaml
```

Detects structural drift and generates:

* schema_diff.json
* schema_diff.txt

Stops if mismatch detected.

---

## Full Diff (Schema + Data)

```bash
deepdiffdb diff --config deepdiffdb.config.yaml
```

Outputs:

* schema_diff.json
* content_diff.json
* migration_pack.sql
* conflicts.json
* summary.txt

---

## Generate Migration Pack

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml
```

---

## Apply Migration Pack

```bash
deepdiffdb apply --pack migration_pack.sql --config deepdiffdb.config.yaml
```

Runs fully inside a transaction.

---

## Dry Run

```bash
deepdiffdb apply --pack migration_pack.sql --dry-run
```

---

## Quick Check

```bash
deepdiffdb check --config deepdiffdb.config.yaml
```

Validates:

* connectivity to both databases
* primary keys exist on all tables (unless ignored)
* output directory is writable

Displays a simple terminal summary on success.

---

# Output Example

**summary.txt**

```
Schema: OK
Tables scanned: 12
Added rows: 18
Updated rows: 4
Conflicts: 2
Migration pack: migration_pack.sql
```

---

# Folder Structure

```txt
deepdiffdb/
  cmd/
    deepdiffdb/
  internal/
    schema/
    content/
    drivers/
    report/
  pkg/
    config/
  bin/
  go.mod
  README.md
```

---

# How It Works

* extracts metadata using information_schema
* builds a normalized schema model
* performs a deep hash-based row comparison
* streams table data in chunks
* isolates safe changes
* generates SQL patch file
* applies migration in a single transaction

Fast, safe, predictable.

---

# Current Limitations

* MSSQL/Oracle not supported yet
* schema auto-merge not implemented
* requires primary keys
* large DBs produce big diff files
* conflict resolution is manual

---

# Roadmap

* schema migration generator
* merge strategies (ours/theirs/manual)
* HTML report viewer
* CI/CD integration
* Git-like versioning for DB diffs
* big-dataset streaming
* Oracle and MSSQL support
