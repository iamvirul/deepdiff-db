# Sample 14: Streaming Support for Large Datasets

This sample demonstrates the **keyset-paginated batch hashing** and **parallel table hashing** features introduced in v0.7. It shows how to compare databases with hundreds of thousands of rows while keeping memory usage bounded and wall-clock time short.

## What You'll Learn

- How to configure `performance.hash_batch_size` and `performance.max_parallel_tables` in `deepdiffdb.config.yaml`
- How to override those settings at runtime with `--batch-size` and `--parallel` CLI flags
- The memory and throughput difference between unbatched (full-scan) and batched (keyset-paginated) hashing
- How to ignore high-volume ancillary tables (e.g. `audit_logs`) and noisy timestamp columns

## Scenario

- **`orders`** — 500 000 rows. 10 rows have a different `amount` in dev (simulating post-checkout adjustments).
- **`products`** — 100 000 rows. 5 rows have a different `price` in dev (simulating a price update rollout).
- **`audit_logs`** — 200 000 rows. Identical in both databases; the `audit_logs` table is intentionally **not** in the ignore list so you can see the full row count being hashed.

The `orders.updated_at` column is ignored so that timestamp noise does not generate false positives.

## Prerequisites

- Go 1.21+ (to run the seed script)
- `deepdiffdb` installed and on your `$PATH`
- No Docker required — the sample uses SQLite file databases

## Quick Start

```bash
cd samples/14-streaming-large-datasets

# 1. Generate the databases (full: ~800k rows, takes ~60s)
make seed

# Or use the small mode for a faster demo (17k rows, takes ~5s)
make seed-small

# 2. Run the diff with config defaults (batch=10000, parallel=2)
make diff

# 3. Inspect the output
cat diff-output/diff_report.txt
```

## Configuration

`deepdiffdb.config.yaml` in this directory sets the performance section explicitly:

```yaml
performance:
  hash_batch_size: 10000     # rows per keyset-paginated query
  max_parallel_tables: 2     # hash orders + products concurrently
```

Key points:
- **`hash_batch_size: 10000`** — each page fetches at most 10 000 rows. Raw row data (~1–2 MB per batch) is freed by the GC between pages, keeping heap flat regardless of table size.
- **`max_parallel_tables: 2`** — prod's `orders` and `products` tables are hashed in separate goroutines, roughly halving wall-clock time on a dual-core host.
- `orders.updated_at` is excluded via `ignore.columns` so timestamp mutations don't inflate the diff.

## CLI Flag Overrides

Config defaults can be overridden per-run without editing the YAML file:

```bash
# Use smaller batches and more parallelism
deepdiffdb diff --config deepdiffdb.config.yaml --batch-size 5000 --parallel 4

# Baseline: disable batching and parallelism (pre-v0.7 behaviour)
deepdiffdb diff --config deepdiffdb.config.yaml --batch-size 0 --parallel 1

# Generate a migration pack using the same flags
deepdiffdb gen-pack --config deepdiffdb.config.yaml --batch-size 5000 --parallel 4
```

`--batch-size 0` disables keyset pagination and falls back to a single full-table query, exactly matching pre-v0.7 behaviour. This is useful as a baseline comparison.

## Expected Output

After running `make diff` you should see output similar to:

```
Comparing prod.db ↔ dev.db
  Hashing orders    [prod] ████████████████████ 500000/500000
  Hashing products  [prod] ████████████████████ 100000/100000
  Hashing audit_logs[prod] ████████████████████ 200000/200000
  Hashing orders    [dev]  ████████████████████ 500003/500003
  Hashing products  [dev]  ████████████████████ 100000/100000
  Hashing audit_logs[dev]  ████████████████████ 200000/200000

Content diff complete.
  Modified rows : 15
  Added rows    : 3
  Removed rows  : 0

Reports written to ./diff-output/
```

The diff report identifies:
- 10 modified orders (different `amount`)
- 5 modified products (different `price`)
- 3 new orders present only in dev

## Memory Behaviour

With `hash_batch_size: 10000` and the full 800k-row dataset, peak heap stays around **~150–200 MB** during hashing. Without batching (`--batch-size 0`), peak heap can reach **~700–900 MB** because all raw row data must be resident before the first hash is written.

You can observe per-batch memory telemetry by enabling debug logging:

```bash
DEEPDIFFDB_LOG_LEVEL=debug deepdiffdb diff --config deepdiffdb.config.yaml 2>&1 | grep alloc_mb
```

Example debug output:
```json
{"level":"debug","table":"orders","batch":1,"total_rows_hashed":10000,"alloc_mb":"148.2"}
{"level":"debug","table":"orders","batch":2,"total_rows_hashed":20000,"alloc_mb":"151.7"}
```

## Tuning Guide

| Scenario | Recommended settings |
|---|---|
| Low-memory host (< 2 GB) | `hash_batch_size: 2000`, `max_parallel_tables: 1` |
| Standard server (4–8 GB) | `hash_batch_size: 10000`, `max_parallel_tables: 2` |
| High-memory host (16+ GB) | `hash_batch_size: 50000`, `max_parallel_tables: 4` |
| Many small tables | `hash_batch_size: 0` (no pagination overhead), `max_parallel_tables: 4` |
| CI/CD pipeline | `hash_batch_size: 5000`, `max_parallel_tables: 1` (predictable, safe) |

## Files in This Sample

```
14-streaming-large-datasets/
  README.md                  — this file
  deepdiffdb.config.yaml     — config with performance section
  Makefile                   — convenience targets
  seed/
    main.go                  — Go script to generate prod.db and dev.db
  diff-output/               — generated reports (created on first diff run)
  prod.db                    — generated by 'make seed' (gitignored)
  dev.db                     — generated by 'make seed' (gitignored)
```

## Cleanup

```bash
make clean   # removes prod.db, dev.db, and generated reports
```

## Learn More

- [Sample 01: Basic Schema Drift](../01-basic-schema-drift/) — start here if you're new to DeepDiff DB
- [Sample 10: Conflict Resolution](../10-conflict-resolution/) — configuring conflict strategies
- [Sample 12: Interactive Resolution](../12-interactive-resolution/) — resolving conflicts interactively
- [DeepDiff DB README](../../README.md) — full documentation
