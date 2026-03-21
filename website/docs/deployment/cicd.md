---
sidebar_position: 2
---

# CI/CD Integration

DeepDiff DB is designed to run inside CI pipelines. Example workflows for GitHub Actions and GitLab CI are provided in [`examples/cicd/`](https://github.com/iamvirul/deepdiff-db/tree/main/examples/cicd).

## GitHub Actions

The example workflow (`.github/workflows/db-diff.yml`) runs on every pull request that touches migration files, spins up two MySQL containers, runs a diff, and posts the result as a PR comment.

```yaml
name: Database Diff Check

on:
  pull_request:
    paths:
      - 'migrations/**'
      - '**.sql'

jobs:
  db-diff:
    runs-on: ubuntu-latest
    services:
      db-prod:
        image: mysql:8.4
        env: { MYSQL_ROOT_PASSWORD: prod_secret, MYSQL_DATABASE: myapp_prod }
        options: --health-cmd="mysqladmin ping" --health-interval=5s
      db-dev:
        image: mysql:8.4
        env: { MYSQL_ROOT_PASSWORD: dev_secret, MYSQL_DATABASE: myapp_dev }
        options: --health-cmd="mysqladmin ping" --health-interval=5s

    steps:
      - uses: actions/checkout@v4

      - name: Install DeepDiff DB
        run: |
          VERSION=$(curl -s https://api.github.com/repos/iamvirul/deepdiff-db/releases/latest \
            | grep '"tag_name"' | cut -d'"' -f4)
          curl -fsSL \
            "https://github.com/iamvirul/deepdiff-db/releases/download/${VERSION}/deepdiffdb_${VERSION}_linux_amd64.tar.gz" \
            | tar -xz deepdiffdb
          sudo mv deepdiffdb /usr/local/bin/deepdiffdb

      - name: Run diff
        run: deepdiffdb diff --config deepdiffdb.config.yaml
```

See [`examples/cicd/github-actions.yml`](https://github.com/iamvirul/deepdiff-db/blob/main/examples/cicd/github-actions.yml) for the full example including PR comment posting and artifact upload.

## GitLab CI

```yaml
db-diff:
  stage: test
  image: alpine:3.20
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  script:
    - apk add --no-cache curl tar
    - |
      curl -fsSL "https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiffdb_linux_amd64.tar.gz" \
        | tar -xz deepdiffdb && mv deepdiffdb /usr/local/bin/
    - deepdiffdb diff --config deepdiffdb.config.yaml
  artifacts:
    paths: [diff-output/]
    expire_in: 7 days
```

See [`examples/cicd/gitlab-ci.yml`](https://github.com/iamvirul/deepdiff-db/blob/main/examples/cicd/gitlab-ci.yml) for the full example.

## Pre-commit hook

Block commits when unexpected schema drift is detected:

```bash
# Install
cp examples/cicd/pre-commit-hook.sh .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

The hook runs `deepdiffdb schema-diff` and blocks the commit if drift is found. Set `DEEPDIFFDB_ALLOW_DRIFT=1` to demote it to a warning.

### With pre-commit framework

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: local
    hooks:
      - id: deepdiff-db
        name: DeepDiff DB schema check
        entry: examples/cicd/pre-commit-hook.sh
        language: script
        pass_filenames: false
```

## Tips

- **Pin the version** in CI with `DEEPDIFFDB_VERSION: v1.0.0` to avoid unexpected upgrades.
- **Cache the binary** between runs using your CI platform's cache action.
- **Fail fast on schema changes** — set `--exit-code` (coming in a future release) to fail CI if any drift is detected.
- **Use Docker** in CI for a hermetic environment: `docker run ghcr.io/iamvirul/deepdiff-db:latest diff`.
