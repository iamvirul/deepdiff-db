#!/usr/bin/env bash
# DeepDiff DB — pre-commit hook
#
# Runs a schema diff before every commit. If unexpected schema drift is
# detected (tables added/dropped or columns changed) the commit is blocked
# until the developer either reviews the changes or updates the baseline.
#
# Installation:
#   cp examples/cicd/pre-commit-hook.sh .git/hooks/pre-commit
#   chmod +x .git/hooks/pre-commit
#
# Or with pre-commit (https://pre-commit.com/), add to .pre-commit-config.yaml:
#   repos:
#     - repo: local
#       hooks:
#         - id: deepdiff-db
#           name: DeepDiff DB schema check
#           entry: examples/cicd/pre-commit-hook.sh
#           language: script
#           pass_filenames: false
#
# Configuration:
#   Set DEEPDIFFDB_CONFIG to the path of your config file (default: deepdiffdb.config.yaml).
#   Set DEEPDIFFDB_ALLOW_DRIFT=1 to warn instead of block on schema drift.

set -euo pipefail

CONFIG="${DEEPDIFFDB_CONFIG:-deepdiffdb.config.yaml}"
ALLOW_DRIFT="${DEEPDIFFDB_ALLOW_DRIFT:-0}"
OUTPUT_DIR="$(mktemp -d)"
trap 'rm -rf "$OUTPUT_DIR"' EXIT

# Skip if no config file present (not all repos use deepdiffdb)
if [ ! -f "$CONFIG" ]; then
  exit 0
fi

# Skip if deepdiffdb is not installed
if ! command -v deepdiffdb &>/dev/null; then
  echo "[deepdiff-db] deepdiffdb not found in PATH — skipping schema check." >&2
  echo "[deepdiff-db] Install: https://iamvirul.github.io/deepdiff-db/docs/getting-started/installation" >&2
  exit 0
fi

echo "[deepdiff-db] Running schema diff..." >&2

set +e
deepdiffdb schema-diff \
  --config "$CONFIG" \
  --output-dir "$OUTPUT_DIR" \
  --quiet
EXIT_CODE=$?
set -e

if [ $EXIT_CODE -eq 0 ]; then
  echo "[deepdiff-db] No schema drift detected. ✓" >&2
  exit 0
fi

echo "" >&2
echo "┌─────────────────────────────────────────────────────┐" >&2
echo "│  DeepDiff DB: Schema drift detected!               │" >&2
echo "└─────────────────────────────────────────────────────┘" >&2
echo "" >&2

if [ -f "$OUTPUT_DIR/schema_diff.txt" ]; then
  cat "$OUTPUT_DIR/schema_diff.txt" >&2
fi

echo "" >&2

if [ "$ALLOW_DRIFT" = "1" ]; then
  echo "[deepdiff-db] DEEPDIFFDB_ALLOW_DRIFT=1 — committing anyway (warning only)." >&2
  exit 0
fi

echo "[deepdiff-db] Commit blocked. Options:" >&2
echo "  1. Review the diff above and update your migration files." >&2
echo "  2. Set DEEPDIFFDB_ALLOW_DRIFT=1 to skip this check." >&2
echo "  3. Run: deepdiffdb schema-diff --config $CONFIG  for details." >&2
exit 1
