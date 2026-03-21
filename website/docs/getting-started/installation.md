---
sidebar_position: 1
---

# Installation

DeepDiff DB ships as a single static binary with no runtime dependencies. Choose the method that suits your environment.

## Homebrew (macOS and Linux — Recommended)

```bash
brew tap iamvirul/deepdiff-db
brew install deepdiff-db
```

One-liner alternative:

```bash
brew install iamvirul/deepdiff-db/deepdiff-db
```

**Upgrade to the latest version:**

```bash
brew upgrade deepdiff-db
```

## Binary Download

Pre-compiled binaries are published for every release on the [GitHub Releases](https://github.com/iamvirul/deepdiff-db/releases) page.

### Linux (amd64)

```bash
wget https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb-linux-amd64
chmod +x deepdiffdb-linux-amd64
sudo mv deepdiffdb-linux-amd64 /usr/local/bin/deepdiffdb
```

### Linux (arm64)

```bash
wget https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb-linux-arm64
chmod +x deepdiffdb-linux-arm64
sudo mv deepdiffdb-linux-arm64 /usr/local/bin/deepdiffdb
```

### macOS (Apple Silicon — arm64)

```bash
wget https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb-darwin-arm64
chmod +x deepdiffdb-darwin-arm64
sudo mv deepdiffdb-darwin-arm64 /usr/local/bin/deepdiffdb
```

### macOS (Intel — amd64)

```bash
wget https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb-darwin-amd64
chmod +x deepdiffdb-darwin-amd64
sudo mv deepdiffdb-darwin-amd64 /usr/local/bin/deepdiffdb
```

### Windows (amd64)

Download `deepdiffdb-windows-amd64.exe` from the Releases page and place it somewhere on your `%PATH%`, for example `C:\Windows\System32\deepdiffdb.exe`, or add its directory to the `PATH` environment variable.

## Build from Source

Requires Go 1.21 or later.

**Using `go install`:**

```bash
go install github.com/iamvirul/deepdiff-db/cmd/deepdiffdb@latest
```

This installs the binary to `$GOPATH/bin` (usually `~/go/bin`). Make sure that directory is on your `PATH`.

**Clone and build locally:**

```bash
git clone https://github.com/iamvirul/deepdiff-db.git
cd deepdiff-db
./scripts/build-local.sh --install --install-dir ~/bin
```

## Verify the Installation

```bash
deepdiffdb version
```

Expected output:

```
deepdiffdb version 0.9.0
```

## Next Step

Follow the [Quick Start](/docs/getting-started/quick-start) guide to run your first diff.
