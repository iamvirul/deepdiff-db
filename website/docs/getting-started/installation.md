---
sidebar_position: 1
---

# Installation

DeepDiff DB ships as a single static binary with no runtime dependencies. Choose the method that suits your environment.

## Homebrew (macOS and Linux — Recommended)

```bash
brew tap iamvirul/tap
brew install deepdiff-db
```

**Upgrade to the latest version:**

```bash
brew upgrade deepdiff-db
```

## Docker

```bash
docker pull ghcr.io/iamvirul/deepdiff-db:latest

# Run a diff
docker run --rm \
  -v $(pwd)/deepdiffdb.config.yaml:/config/deepdiffdb.config.yaml:ro \
  -v $(pwd)/diff-output:/diff-output \
  ghcr.io/iamvirul/deepdiff-db:latest diff
```

See the [Docker guide](/docs/deployment/docker) for full usage details.

## Binary Download

Pre-compiled archives are published for every release on the [GitHub Releases](https://github.com/iamvirul/deepdiff-db/releases) page.

### Linux (amd64)

```bash
curl -fsSL \
  https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb_linux_amd64.tar.gz \
  | tar -xz deepdiffdb
sudo mv deepdiffdb /usr/local/bin/deepdiffdb
```

### Linux (arm64)

```bash
curl -fsSL \
  https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb_linux_arm64.tar.gz \
  | tar -xz deepdiffdb
sudo mv deepdiffdb /usr/local/bin/deepdiffdb
```

### macOS (Apple Silicon — arm64)

```bash
curl -fsSL \
  https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb_darwin_arm64.tar.gz \
  | tar -xz deepdiffdb
sudo mv deepdiffdb /usr/local/bin/deepdiffdb
```

### macOS (Intel — amd64)

```bash
curl -fsSL \
  https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb_darwin_amd64.tar.gz \
  | tar -xz deepdiffdb
sudo mv deepdiffdb /usr/local/bin/deepdiffdb
```

### Windows (amd64)

Download `deepdiffdb_windows_amd64.zip` from the [Releases page](https://github.com/iamvirul/deepdiff-db/releases), extract `deepdiffdb.exe`, and add it to your `%PATH%`.

## Build from Source

Requires Go 1.25.8 or later.

**Using `go install`:**

```bash
go install github.com/iamvirul/deepdiff-db/cmd/deepdiffdb@latest
```

This installs the binary to `$GOPATH/bin` (usually `~/go/bin`). Make sure that directory is on your `PATH`.

**Clone and build locally:**

```bash
git clone https://github.com/iamvirul/deepdiff-db.git
cd deepdiff-db
go build -o deepdiffdb ./cmd/deepdiffdb
sudo mv deepdiffdb /usr/local/bin/deepdiffdb
```

## Verify the Installation

```bash
deepdiffdb --version
```

Expected output:

```
DeepDiff DB v1.0.0
```

## Next Step

Follow the [Quick Start](/docs/getting-started/quick-start) guide to run your first diff.
