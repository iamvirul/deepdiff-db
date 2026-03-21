---
sidebar_position: 4
---

# Checkpoint and Resume

Added in **v0.6**. For long-running operations, DeepDiff DB automatically saves progress checkpoints so that an interrupted run can be resumed rather than restarted from scratch.

## Supported Operations

| Command | Checkpoint support |
|---|---|
| `gen-pack` | Yes — saves progress after each table is processed |
| `apply` | Yes — saves progress after each statement batch is committed |
| `diff` | No — fast enough to re-run; no checkpoint needed |
| `schema-diff` | No |
| `schema-migrate` | No |
| `resolve-conflicts` | Partial — `resolutions.json` serves as the resume state |

## How to Resume

Add `--resume` to the interrupted command:

```bash
# Resume an interrupted gen-pack
deepdiffdb gen-pack --config deepdiffdb.config.yaml --resume

# Resume an interrupted apply
deepdiffdb apply \
  --pack diff-output/migration_pack.sql \
  --config deepdiffdb.config.yaml \
  --resume
```

DeepDiff DB locates the most recent checkpoint for the operation, validates that the config has not changed, and continues from the last saved position.

## Checkpoint File Location

Checkpoints are written to `{output.dir}/.checkpoint/` as JSON files named by operation type and timestamp:

```
diff-output/
  .checkpoint/
    gen-pack-2026-03-21T10-05-32Z.json
    apply-2026-03-21T10-22-11Z.json
```

Do not edit checkpoint files manually. If a checkpoint file is corrupt, delete it and re-run without `--resume`.

## Checkpoint Expiry

Checkpoint files expire after **24 hours**. An expired checkpoint is automatically deleted and the operation starts fresh. This prevents accidentally resuming a stale run against a database that has changed significantly since the original run.

To force a fresh start before the 24-hour expiry, delete the checkpoint directory:

```bash
rm -rf diff-output/.checkpoint/
```

## Config Validation on Resume

When resuming, DeepDiff DB compares the stored config hash against the current config file. If the config has changed (different databases, different ignore lists, different batch size), the resume is rejected with an error:

```
Error: checkpoint config mismatch — config file has changed since checkpoint was created.
Delete the checkpoint and re-run without --resume to start fresh.
```

This prevents silently applying a checkpoint generated against a different database.

## When to Use Checkpointing

- **Large migration packs** — packs with thousands of statements can take minutes. A network interruption mid-apply would otherwise require restarting.
- **Slow database hosts** — gen-pack on a database with slow I/O over a VPN can be interrupted by connection timeouts.
- **Scheduled maintenance windows** — start gen-pack before the window opens, then apply during the window. If gen-pack is interrupted, resume it.
- **Interactive resolution sessions** — use `resolve-conflicts --resume` to continue a multi-session conflict review across days.
