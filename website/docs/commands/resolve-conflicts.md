---
sidebar_position: 8
---

# resolve-conflicts

Resolves data conflicts interactively or automatically. A conflict is a row where the primary key exists in both databases but the column values differ. Conflicts must be resolved before a complete migration pack can be generated.

## Usage

```bash
# Interactive mode
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml

# Auto mode (applies configured strategies without prompts)
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --auto

# Resume a previous session
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --resume
```

## Flags

| Flag | Description |
|---|---|
| `--config` | Path to the configuration file (default: `deepdiffdb.config.yaml`) |
| `--auto` | Apply configured strategies automatically without interactive prompts |
| `--resume` | Load existing `resolutions.json` and continue from where the last session ended |
| `--verbose` | Enable debug-level logging |
| `--log-level` | Minimum log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `--log-format` | Log output format: `text` or `json` (default: `text`) |
| `--log-file` | Write logs to this file in addition to stdout |

## Interactive Mode Walkthrough

For each conflict the tool presents a side-by-side comparison:

```
Conflict 1 of 3  —  Table: orders  —  PK: id=1042

  Column         Production              Development
  ─────────────────────────────────────────────────
  status         'pending'             ► 'shipped'
  total          150.00                  150.00
  updated_at     2026-03-01 10:00:00   ► 2026-03-15 14:22:00

  [1] Keep production (ours)
  [2] Use development (theirs)
  [3] Skip (decide later)
  [4] Resolve all remaining in this table with: ours / theirs
  [5] Resolve all remaining conflicts with: ours / theirs
  [q] Quit and save progress

Choice:
```

Changed columns are marked with `►`. After choosing, the resolution is recorded and the next conflict is shown.

## Auto Mode

Auto mode applies the strategies defined in `conflict_resolution` in your config file. No prompts are shown.

```yaml
conflict_resolution:
  default_strategy: "theirs"
  strategies:
    - table: "config"
      strategy: "ours"
```

```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --auto
```

Output:

```
Auto-resolving conflicts...
  orders    : 2 conflicts → theirs
  config    : 1 conflict  → ours (table override)
Resolved: 3 / 3
Saved: diff-output/resolutions.json
```

## Resolution Strategies

| Strategy | Effect |
|---|---|
| `ours` | The production row value is kept; the dev change is discarded |
| `theirs` | The dev row value replaces the production row value |
| `manual` | Requires an interactive decision (only valid in interactive mode) |

## How Resolutions Are Applied

`resolve-conflicts` writes decisions to `resolutions.json`. When `gen-pack` runs next, it reads this file and honours the resolutions — generating `UPDATE` statements only for rows resolved as `theirs`, and skipping rows resolved as `ours`.

## Output Files

| File | Description |
|---|---|
| `resolutions.json` | Saved resolution decisions; read by `gen-pack` on the next run |

## Resuming a Session

Pass `--resume` to pick up where a previous session left off. Existing decisions in `resolutions.json` are loaded and those conflicts are skipped.

```bash
deepdiffdb resolve-conflicts --config deepdiffdb.config.yaml --resume
```
