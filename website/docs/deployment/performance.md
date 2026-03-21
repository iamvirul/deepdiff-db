---
sidebar_position: 3
---

# Performance

## Benchmark results

Benchmarks run on Apple M3 (ARM64), Go 1.25.8, SQLite in-memory database.
Run with: `go test ./tests/content/... -bench=BenchmarkHashTable -benchmem -benchtime=3s`

| Benchmark | Rows | Batch size | ns/op | Rows/sec |
|-----------|------|-----------|-------|----------|
| Unbatched | 1 000 | — | 917 187 | ~1 090 000 |
| Unbatched | 10 000 | — | 8 665 431 | ~1 154 000 |
| Batched | 1 000 | 500 | 1 820 235 | ~549 000 |
| Batched | 10 000 | 1 000 | 12 927 090 | ~773 000 |

> **Note:** These numbers reflect a local SQLite in-memory database. Real-world performance against network databases (MySQL, PostgreSQL, Oracle) will be lower due to network latency. The batched mode trades peak throughput for bounded memory — important for tables with millions of rows.

## Memory profile

| Mode | Table size | Approximate heap |
|------|-----------|-----------------|
| Unbatched | 1M rows | ~840 MB |
| Batched (batch=10 000) | 1M rows | ~84 MB |
| Batched (batch=1 000) | 1M rows | ~8 MB |

Batched mode keeps exactly `O(batch_size)` rows in memory at any time. GC runs between batches, keeping the working set flat regardless of total table size.

## Tuning

### Batch size

```yaml
# deepdiffdb.config.yaml
performance:
  hash_batch_size: 10000    # rows per keyset page (0 = no batching)
  max_parallel_tables: 2    # concurrent table comparisons
```

```bash
# CLI flags override config
deepdiffdb diff --batch-size 5000 --parallel 4
```

**Guidelines:**
- `hash_batch_size=0` (unbatched) is fastest for tables with < 100k rows where memory is not a concern
- `hash_batch_size=10000` is a good default for large tables
- Reduce to `1000–5000` if you see memory pressure
- `max_parallel_tables` helps on multi-core hosts; set to the number of available cores for maximum throughput. Default is `1` to avoid overloading the source databases

### Index requirements

Keyset pagination requires a **primary key**. Tables without a primary key fall back to full-scan mode (unbatched). Add a primary key or a unique index to unlock batched mode.

### Connection pool

Parallel hashing opens multiple connections. Ensure your database's `max_connections` limit is high enough:

```
required connections ≈ max_parallel_tables × 2  (prod + dev)
```

## Profiling

To capture a CPU profile for your own workload:

```bash
go test ./tests/content/... \
  -bench=BenchmarkHashTable_Unbatched_10k \
  -cpuprofile=cpu.prof \
  -memprofile=mem.prof \
  -benchtime=10s

go tool pprof -http=:8080 cpu.prof
```
