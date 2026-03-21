# Migration Guide: v0.x → v1.0

This document covers all changes you need to make when upgrading DeepDiff DB from any v0.x release to v1.0.

---

## Breaking Changes

### 1. File permissions tightened (security hardening)

Output files (diff reports, migration packs, checkpoints) are now created with `0600` permissions instead of `0644`, and output directories with `0750` instead of `0755`.

**Impact:** Files are no longer world-readable. If your deployment reads output files as a different OS user (e.g., a CI agent reading files written by an app user), ensure those users are in the same group.

**Action required:** Review your file-reading pipelines if you rely on world-readable output files.

---

### 2. Go 1.25.8 minimum required

The minimum supported Go version is now **1.25.8** (up from 1.22). This resolves two standard library CVEs:
- [GO-2026-4603](https://pkg.go.dev/vuln/GO-2026-4603): `html/template` URL escaping
- [GO-2026-4601](https://pkg.go.dev/vuln/GO-2026-4601): `net/url` IPv6 parsing

**Action required:** Update your Go toolchain before building from source.

---

## Non-Breaking Changes (New Features)

### Docker support

A multi-stage `Dockerfile` is now included. Build and run with:

```bash
# Build
docker build --build-arg VERSION=v1.0.0 -t deepdiff-db:v1.0.0 .

# Run
docker run --rm \
  -v $(pwd)/deepdiffdb.config.yaml:/config/deepdiffdb.config.yaml:ro \
  -v $(pwd)/diff-output:/diff-output \
  deepdiff-db:v1.0.0 diff
```

A `docker-compose.example.yml` is also provided for projects that manage both databases via Compose.

---

### GoReleaser-based releases

Starting with v1.0, releases are built by GoReleaser. Archive names have changed:

| Platform | v0.x binary name | v1.0 archive name |
|----------|-----------------|-------------------|
| Linux x64 | `deepdiffdb-linux-amd64` | `deepdiffdb_v1.0.0_linux_amd64.tar.gz` |
| Linux ARM64 | `deepdiffdb-linux-arm64` | `deepdiffdb_v1.0.0_linux_arm64.tar.gz` |
| macOS x64 | `deepdiffdb-darwin-amd64` | `deepdiffdb_v1.0.0_darwin_amd64.tar.gz` |
| macOS ARM64 | `deepdiffdb-darwin-arm64` | `deepdiffdb_v1.0.0_darwin_arm64.tar.gz` |
| Windows x64 | `deepdiffdb-windows-amd64.exe` | `deepdiffdb_v1.0.0_windows_amd64.zip` |

**Action required:** Update your CI download scripts to use the new archive names. Each archive contains the binary plus `README.md`, `CHANGELOG.md`, and `deepdiffdb.config.yaml.example`.

Docker images are now published to GitHub Container Registry:

```bash
docker pull ghcr.io/iamvirul/deepdiff-db:v1.0.0
docker pull ghcr.io/iamvirul/deepdiff-db:latest
```

---

### Homebrew formula updated

The formula now installs the v1.0 binary and injects the version string via ldflags:

```bash
brew tap iamvirul/tap
brew install deepdiff-db
deepdiffdb --version   # DeepDiff DB v1.0.0
```

---

## Configuration File Changes

No configuration file changes are required. The v0.x `deepdiffdb.config.yaml` format is fully compatible with v1.0.

---

## Command Changes

No command-line flag or subcommand changes in v1.0. All v0.x commands work unchanged.

---

## Deprecated Features

None in v1.0.

---

## Upgrade Steps

1. **Check your Go version** (if building from source):
   ```bash
   go version  # must be >= go1.25.8
   ```

2. **Update the binary** via your preferred method:
   ```bash
   # Homebrew
   brew upgrade deepdiff-db

   # Direct download (Linux arm64)
   curl -fsSL https://github.com/iamvirul/deepdiff-db/releases/download/v1.0.0/deepdiffdb_v1.0.0_linux_arm64.tar.gz \
     | tar -xz deepdiffdb
   sudo mv deepdiffdb /usr/local/bin/deepdiffdb

   # Docker
   docker pull ghcr.io/iamvirul/deepdiff-db:v1.0.0
   ```

3. **Verify** the installation:
   ```bash
   deepdiffdb --version
   deepdiffdb check --config deepdiffdb.config.yaml
   ```

4. **(Optional)** Review output file permissions if your pipeline reads diff output as a different OS user.

---

## Getting Help

- [Documentation](https://iamvirul.github.io/deepdiff-db/)
- [GitHub Issues](https://github.com/iamvirul/deepdiff-db/issues)
- [Changelog](CHANGELOG.md)
