---
sidebar_position: 4
---

# Migration Guide: v0.x → v1.0

## Breaking changes

### 1. Output file permissions tightened

Output files (diff reports, migration packs, checkpoints) are now created with `0600` permissions instead of `0644`, and output directories with `0750` instead of `0755`.

**Impact:** Files are no longer world-readable.

**Action:** If your pipeline reads output files as a different OS user, ensure both users share the same group.

### 2. Go 1.25.8 minimum (build from source only)

If you build from source, Go 1.25.8+ is required. This resolves two stdlib CVEs ([GO-2026-4603](https://pkg.go.dev/vuln/GO-2026-4603), [GO-2026-4601](https://pkg.go.dev/vuln/GO-2026-4601)).

**Action:** `go install golang.org/dl/go1.25.8@latest && go1.25.8 download`

### 3. Release archive names changed

GoReleaser now builds releases. Archive names changed:

| Platform | v0.x | v1.0 |
|----------|------|------|
| Linux x64 | `deepdiffdb-linux-amd64` | `deepdiffdb_v1.0.0_linux_amd64.tar.gz` |
| Linux ARM64 | `deepdiffdb-linux-arm64` | `deepdiffdb_v1.0.0_linux_arm64.tar.gz` |
| macOS ARM64 | `deepdiffdb-darwin-arm64` | `deepdiffdb_v1.0.0_darwin_arm64.tar.gz` |
| Windows x64 | `deepdiffdb-windows-amd64.exe` | `deepdiffdb_v1.0.0_windows_amd64.zip` |

**Action:** Update your CI download scripts.

---

## Non-breaking additions

- **Docker image** — `ghcr.io/iamvirul/deepdiff-db:v1.0.0`
- **docker-compose.example.yml** — included in repo root
- **`examples/cicd/`** — GitHub Actions, GitLab CI, and pre-commit hook examples
- **Homebrew** — `brew install iamvirul/deepdiff-db/deepdiff-db`
- **Security scanning** in CI — gosec + govulncheck on every push

---

## Config file compatibility

The `deepdiffdb.config.yaml` format is **fully compatible**. No changes required.

---

## Upgrade steps

```bash
# 1. Update binary
brew upgrade deepdiff-db
# or
curl -fsSL \
  https://github.com/iamvirul/deepdiff-db/releases/download/v1.0.0/deepdiffdb_v1.0.0_linux_amd64.tar.gz \
  | tar -xz deepdiffdb && sudo mv deepdiffdb /usr/local/bin/

# 2. Verify
deepdiffdb --version
deepdiffdb check --config deepdiffdb.config.yaml
```

---

## Need help?

- [GitHub Issues](https://github.com/iamvirul/deepdiff-db/issues)
- [Full MIGRATION.md](https://github.com/iamvirul/deepdiff-db/blob/main/MIGRATION.md)
