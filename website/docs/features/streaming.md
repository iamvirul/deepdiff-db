---
sidebar_position: 1
---

# Streaming Large Datasets

Added in **v0.7**. DeepDiff DB uses keyset-paginated batch hashing to keep memory usage flat regardless of table size, and a bounded goroutine pool for parallel table processing.

## How Keyset Pagination Works

Before v0.7, DeepDiff DB loaded an entire table into memory before computing hashes. For tables with millions of rows this could use 700–900 MB of heap per table.

Starting from v0.7, each table is read in pages using a keyset (cursor) query:

```sql
-- First page
SELECT pk, col1, col2 FROM orders
ORDER BY id
LIMIT 10000;

-- Subsequent pages
SELECT pk, col1, col2 FROM orders
WHERE id > :lastSeenId
ORDER BY id
LIMIT 10000;
```

Each page is hashed and discarded before the next page is fetched. Peak heap usage is proportional to the batch size — approximately 1–2 MB per 10,000 rows for typical schemas — so a 50 million row table uses the same ~15 MB peak as a 50,000 row table.

The cursor field is the table's primary key (or the first column of a composite primary key).

## Configuring Batch Size

### Via configuration file

```yaml
performance:
  hash_batch_size: 10000       # rows per page; 0 = full-table scan (pre-v0.7 behaviour)
  max_parallel_tables: 2       # tables hashed concurrently
```

### Via CLI flags (overrides config)

```bash
deepdiffdb diff --config deepdiffdb.config.yaml --batch-size 5000 --parallel 4
deepdiffdb gen-pack --config deepdiffdb.config.yaml --batch-size 5000 --parallel 4
```

Setting `--batch-size 0` disables pagination entirely and reverts to pre-v0.7 full-table-scan behaviour. Use this only if you have confirmed that available memory is sufficient for the largest table.

## Parallel Table Hashing

The `--parallel` flag (or `performance.max_parallel_tables` in config) controls how many tables are hashed concurrently. DeepDiff DB uses a bounded goroutine pool backed by `errgroup` and `semaphore.NewWeighted`.

**Example — 4 tables in parallel:**

```bash
deepdiffdb gen-pack --config deepdiffdb.config.yaml --batch-size 10000 --parallel 4
```

Each parallel table consumes its own batch of database connections, so the effective connection count is `parallel * 2` (one connection per database per table). Make sure your database's `max_connections` limit accommodates this.

## Memory Characteristics

| Scenario | Peak heap |
|---|---|
| Pre-v0.7 (no batching) | ~700–900 MB for a 5M row table |
| v0.7+ default (batch=10000, parallel=1) | ~15–20 MB |
| batch=5000, parallel=4 | ~30–40 MB total across 4 tables |

Memory figures are approximate and depend on row width.

## When to Tune These Settings

| Situation | Recommendation |
|---|---|
| Tables with `>` 500k rows | Use `--batch-size 10000` (default) |
| Very wide rows (many large text columns) | Reduce `--batch-size` to 2000–5000 |
| Fast database with many CPU cores | Increase `--parallel` to 2–4 |
| Slow network or rate-limited database | Reduce `--parallel` to 1 |
| Available heap is under 512 MB | Keep `--batch-size` at 5000 or less |
| Debugging or reproducing pre-v0.7 behaviour | Set `--batch-size 0` |

## Debug Telemetry

At `--log-level debug`, DeepDiff DB emits per-batch memory telemetry:

```json
{"level":"debug","table":"orders","batch":42,"total_rows_hashed":420000,"alloc_mb":14.2}
```

This is useful for observing memory behaviour in production-like environments before committing to a batch size configuration.
