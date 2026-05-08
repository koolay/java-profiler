# Profiling Configuration

This document records the first implementation's configuration vocabulary.

## Collector configuration

- `JAVA_PROFILER_BACKEND_URL`
- `JAVA_PROFILER_BACKEND_TOKEN`
- `JAVA_PROFILER_NODE_NAME`
- `JAVA_PROFILER_CLUSTER`
- `JAVA_PROFILER_NAMESPACE_SCOPE`
- `JAVA_PROFILER_UPLOAD_BATCH_LIMIT`
- `JAVA_PROFILER_UPLOAD_MAX_BYTES`
- `JAVA_PROFILER_UPLOAD_MAX_AGE`
- `JAVA_PROFILER_STARTUP_DELAY`
- `JAVA_PROFILER_SNAPSHOT_INTERVAL`

## Backend configuration

- `JAVA_PROFILER_CLICKHOUSE_DSN`
- `JAVA_PROFILER_LISTEN_ADDR`
- `JAVA_PROFILER_AUTH_TOKEN`
- `JAVA_PROFILER_QUERY_TIMEOUT`
- `JAVA_PROFILER_MAX_STACK_DEPTH`
- `JAVA_PROFILER_MAX_RESULT_ROWS`

## Operating defaults

- profiling is disabled unless Kubernetes metadata enables it
- temporary profiling is bounded
- export metrics are exposed through exporter endpoints only
- retention stays at or below seven days
