---
sidebar_position: 3
---

# Conflict Resolution

Added in **v0.4**. A conflict occurs when the same primary key exists in both the production and development databases but the row values differ. Unlike added or removed rows, conflicts require a decision: keep production, use development, or decide manually.

## What Is a Conflict?

When DeepDiff DB compares two databases:

- A row that exists in dev but not prod is an **addition** — safe to insert into prod.
- A row that exists in prod but not dev is a **removal** — may be deleted from prod.
- A row that exists in both with the same values is **unchanged** — nothing to do.
- A row that exists in both but with **different values** is a **conflict**.

Conflicts matter because applying the dev version overwrites the prod version. If production has received updates since the dev backup was taken, blindly overwriting would lose those updates.

## Resolution Strategies

Three strategies are available:

| Strategy | Effect |
|---|---|
| `ours` | Keep the production row; discard the dev change. The row is not included in the migration pack. |
| `theirs` | Use the dev row; replace the production row. Generates an `UPDATE` or `DELETE` + `INSERT` in the migration pack. |
| `manual` | The conflict is flagged for interactive resolution via `resolve-conflicts`. |

## Config-Based Auto-Resolution

Define default and per-table strategies in `deepdiffdb.config.yaml`:

```yaml
conflict_resolution:
  default_strategy: "manual"   # fallback for any table not listed below

  strategies:
    - table: "feature_flags"
      strategy: "theirs"       # always accept dev feature flags

    - table: "audit_logs"
      strategy: "ours"         # never overwrite production audit data

    - table: "orders"
      strategy: "manual"       # require review for order data
```

When `gen-pack` runs, it reads the config strategies and automatically resolves conflicts that match. Only rows with `manual` strategy (or no matching strategy when default is `manual`) remain in `conflicts.json` for interactive resolution.

## Interactive Resolution

Run `resolve-conflicts` to work through pending conflicts one by one:

```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml
```

The tool shows each conflict as a side-by-side table:

```
Conflict 1 of 3  —  Table: orders  —  PK: id=1042

  Column         Production              Development
  ─────────────────────────────────────────────────
  status         'pending'             ► 'shipped'
  updated_at     2026-03-01 10:00:00   ► 2026-03-15 14:22:00

  [1] Keep production (ours)
  [2] Use development (theirs)
  [3] Skip
  [q] Save and quit
```

Changed columns are marked with `►`. Choose an option and the decision is saved.

## Auto Mode for CI/CD

When running non-interactively (e.g. in a CI pipeline), use `--auto` to apply configured strategies without prompts:

```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --auto
```

Any conflict with no matching strategy in config (and a default of `manual`) is left unresolved and will appear in `conflicts.json`.

## Persistence: resolutions.json

All decisions are written to `{output.dir}/resolutions.json`. This file is read by `gen-pack` on the next run — resolved conflicts are either included (if `theirs`) or excluded (if `ours`) from the migration pack automatically.

The file format is a JSON array:

```json
[
  {
    "table": "orders",
    "pk": {"id": 1042},
    "strategy": "theirs",
    "resolved_at": "2026-03-21T10:00:00Z"
  }
]
```

## Resuming a Session

Pass `--resume` to continue a partially-completed resolution session:

```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --resume
```

Previously resolved conflicts are skipped; only unresolved ones are presented.
