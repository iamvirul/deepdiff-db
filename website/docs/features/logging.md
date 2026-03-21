---
sidebar_position: 5
---

# Logging

Added in **v0.6**. All DeepDiff DB commands emit structured, levelled log output. Logs can be written to stdout, a file, or both, and can be formatted as human-readable text or machine-parseable JSON.

## Log Levels

| Level | When it fires |
|---|---|
| `debug` | Detailed operational data: per-batch hashing telemetry, query parameters, cursor values |
| `info` | Normal significant events: command started, table hashing started/completed, file written |
| `warn` | Unexpected but handled events: schema drift on a table (data diff skipped), blocked DDL operation |
| `error` | Failures that abort the operation: connection failure, transaction rollback |

The default level is `info`. Only messages at the configured level and above are emitted.

## Flags

| Flag | Default | Description |
|---|---|---|
| `--log-level` | `info` | Minimum log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | `text` | Output format: `text` or `json` |
| `--log-file` | _(none)_ | Write logs to this file path in addition to stdout |
| `--verbose` | `false` | Shorthand for `--log-level debug` with source file and line number included in each log entry |

## Text Format

The default format is human-readable:

```
2026-03-21T10:05:00Z  INFO  Hashing table: orders (prod)
2026-03-21T10:05:02Z  INFO  Hashing table: orders (dev)
2026-03-21T10:05:04Z  INFO  Table orders: 42000 rows, 3 changes, 1 conflict
2026-03-21T10:05:04Z  WARN  Table invoices: schema drift detected, skipping data diff
2026-03-21T10:05:05Z  INFO  Migration pack written: diff-output/migration_pack.sql (31 statements)
```

## JSON Format

Use `--log-format json` to emit newline-delimited JSON records — suitable for log aggregation pipelines (ELK, Datadog, Splunk, Loki):

```json
{"level":"info","timestamp":"2026-03-21T10:05:00Z","service":"deepdiffdb","message":"Hashing table","table":"orders","database":"prod"}
{"level":"info","timestamp":"2026-03-21T10:05:04Z","service":"deepdiffdb","message":"Table hashed","table":"orders","rows":42000,"changes":3,"conflicts":1,"duration_ms":2103}
{"level":"warn","timestamp":"2026-03-21T10:05:04Z","service":"deepdiffdb","message":"Schema drift detected, skipping data diff","table":"invoices"}
{"level":"info","timestamp":"2026-03-21T10:05:05Z","service":"deepdiffdb","message":"Migration pack written","file":"diff-output/migration_pack.sql","statements":31}
```

Every JSON log entry includes at minimum: `level`, `timestamp`, `service`, and `message`. Additional context fields are appended depending on the operation.

## Writing to a File

```bash
deepdiffdb gen-pack \
  --config deepdiffdb.config.yaml \
  --log-format json \
  --log-level info \
  --log-file /var/log/deepdiffdb/gen-pack-2026-03-21.log
```

Log file output is in addition to stdout — both destinations receive every log entry.

## Debug Mode

Enable debug mode for maximum verbosity:

```bash
deepdiffdb diff --config deepdiffdb.config.yaml --verbose
```

Or equivalently:

```bash
deepdiffdb diff --config deepdiffdb.config.yaml --log-level debug
```

In debug mode, per-batch telemetry is emitted for every page of every table hash:

```json
{"level":"debug","timestamp":"...","service":"deepdiffdb","message":"Batch hashed","table":"orders","batch":42,"rows_in_batch":10000,"total_rows_hashed":420000,"alloc_mb":14.2}
```

## Using JSON Logs with Log Aggregators

### Datadog

Configure the Datadog agent to tail the log file and parse it as JSON. The `level` field maps to Datadog's `status` attribute automatically.

### ELK Stack (Filebeat)

Use Filebeat with the `json.keys_under_root: true` option to index all fields at the document root. Set up an index lifecycle policy on the `deepdiffdb-*` index pattern.

### Grafana Loki

Use the Promtail `json` pipeline stage. Label on `level` and `service` for efficient log stream filtering.
