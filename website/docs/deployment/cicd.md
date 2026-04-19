---
sidebar_position: 2
---

# CI/CD Integration

DeepDiff DB is designed to run inside CI pipelines. Ready-to-use examples for GitHub Actions, GitLab CI, Jenkins, and pre-commit hooks are in [`examples/cicd/`](https://github.com/iamvirul/deepdiff-db/tree/main/examples/cicd).

## GitHub Actions

The example workflow runs on every pull request that touches migration files, spins up two MySQL containers, runs a full diff, and posts the result as a PR comment.

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
        run: |
          mkdir -p diff-output
          deepdiffdb diff --config deepdiffdb.config.yaml || true
          # Parse the JSON reports written to diff-output/ by the tool
          TABLES_CHANGED=$(jq '[.tables[] | select(.has_differences == true)] | length' \
            diff-output/schema_diff.json 2>/dev/null || echo 0)
          DATA_CHANGES=$(jq '.tables | length' \
            diff-output/content_diff.json 2>/dev/null || echo 0)
          echo "tables_changed=$TABLES_CHANGED" >> $GITHUB_OUTPUT
          echo "data_changes=$DATA_CHANGES"     >> $GITHUB_OUTPUT
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
    - apk add --no-cache curl tar jq
    - |
      VERSION=$(curl -fsSL https://api.github.com/repos/iamvirul/deepdiff-db/releases/latest \
        | jq -r '.tag_name')
      curl -fsSL \
        "https://github.com/iamvirul/deepdiff-db/releases/download/${VERSION}/deepdiffdb_${VERSION}_linux_amd64.tar.gz" \
        | tar -xz deepdiffdb && mv deepdiffdb /usr/local/bin/
    - deepdiffdb diff --config deepdiffdb.config.yaml || true
  after_script:
    - |
      if [ -f diff-output/schema_diff.json ]; then
        TABLES=$(jq '[.tables[] | select(.has_differences == true)] | length' \
          diff-output/schema_diff.json 2>/dev/null || echo 0)
        NOTE="**DeepDiff DB Report**\n\nSchema changes: ${TABLES}\n\nSee artifacts for full output."
        curl -s --request POST \
          --header "PRIVATE-TOKEN: ${GITLAB_API_TOKEN}" \
          --data-urlencode "body=${NOTE}" \
          "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/merge_requests/${CI_MERGE_REQUEST_IID}/notes" \
          > /dev/null || true
      fi
  artifacts:
    paths: [diff-output/]
    expire_in: 7 days
```

See [`examples/cicd/gitlab-ci.yml`](https://github.com/iamvirul/deepdiff-db/blob/main/examples/cicd/gitlab-ci.yml) for the full example.

## Jenkins

The `Jenkinsfile` example uses a Declarative Pipeline with masked credentials. It marks the build `UNSTABLE` (rather than `FAILED`) on drift, so downstream stages can still run.

```groovy
pipeline {
    agent { label 'linux' }

    stages {
        stage('Schema Diff') {
            steps {
                sh 'deepdiffdb schema-diff --config deepdiffdb.config.yaml || true'
            }
        }

        stage('Data Diff') {
            steps {
                sh 'deepdiffdb diff --config deepdiffdb.config.yaml || true'
            }
        }

        stage('Evaluate Results') {
            steps {
                script {
                    def driftCount = sh(
                        script: "jq '[.tables[] | select(.has_differences == true)] | length' diff-output/schema_diff.json",
                        returnStdout: true
                    ).trim().toInteger()

                    if (driftCount > 0) {
                        currentBuild.result = 'UNSTABLE'
                    }
                }
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: 'diff-output/**', allowEmptyArchive: true
        }
    }
}
```

See [`examples/cicd/Jenkinsfile`](https://github.com/iamvirul/deepdiff-db/blob/main/examples/cicd/Jenkinsfile) for the full example including credential injection and install step.

## Pre-commit hook

Block commits when unexpected schema drift is detected.

### Manual install

```bash
cp examples/cicd/pre-commit-hook.sh .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

The hook calls `schema-diff --quiet --output-dir <tmpdir>` so it is fully silent on success and prints a clear block message only when drift is found. Set `DEEPDIFFDB_ALLOW_DRIFT=1` to demote the block to a warning.

### With the pre-commit framework

DeepDiff DB ships a `.pre-commit-hooks.yaml` at the repo root, so you can reference it directly from `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/iamvirul/deepdiff-db
    rev: v1.3.0   # pin to a release tag
    hooks:
      - id: deepdiff-db-schema
        # Optional: override the config path
        # args: [--config, path/to/deepdiffdb.config.yaml]
```

The hook triggers on `.sql`, `.go`, `.py`, `.ts`, `.js`, `.rb`, and `.java` file changes. Set `DEEPDIFFDB_CONFIG` to point at a non-default config path.

## Using `version commit` in CI

To record a versioned snapshot on every merge to main, add a `version commit` step. Use `--skip-auth` to bypass the GitHub device flow and `--author` to identify the pipeline:

```yaml
# GitHub Actions — snapshot on merge to main
on:
  push:
    branches: [main]

jobs:
  version-commit:
    runs-on: ubuntu-latest
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

      - name: Init version repo (skip auth in CI)
        run: deepdiffdb version init --skip-auth

      - name: Commit version snapshot
        run: |
          deepdiffdb version commit \
            --config deepdiffdb.config.yaml \
            --message "CI snapshot: ${{ github.sha }}" \
            --author "ci/github-actions"
```

:::tip
`--skip-auth` must be passed on the first `version init` run. Subsequent runs on the same checkout are idempotent. If `.deepdiffdb/` is committed to the repo, `version init` is only needed once locally.
:::

---

## Tips

- **Pin the version** with `DEEPDIFFDB_VERSION: v1.3.0` to avoid unexpected upgrades.
- **Cache the binary** between runs using your CI platform's cache action.
- **Silent schema gate** — use `schema-diff --quiet` for a fully silent check; exit code `0` = no drift, non-zero = drift or error.
- **Temporary output dirs** — use `--output-dir /tmp/deepdiff-$$` so ephemeral runners don't need a writable workspace.
- **Use Docker** for a hermetic environment: `docker run ghcr.io/iamvirul/deepdiff-db:latest diff`.
- **Skip GitHub auth in CI** — always pass `--skip-auth` to `version init` in pipelines; use `--author "ci/<platform>"` on `version commit`.
