# Java Profiling Runbook

## Enable Continuous Profiling

Add one of these metadata keys to a Java workload or Pod:

```yaml
metadata:
  annotations:
    java-profiler.io/profile-mode: continuous
    java-profiler.io/startup-delay: 30s
```

The collector waits for the startup delay, verifies HotSpot compatibility, checks for async-profiler conflicts, then uploads normalized profiles.

## Enable Temporary Profiling

```yaml
metadata:
  annotations:
    java-profiler.io/profile-mode: temporary
    java-profiler.io/profile-duration: 10m
    java-profiler.io/snapshot-interval: 10s
```

Temporary profiling stops automatically when the duration expires. High-frequency thread snapshots are only intended for temporary windows.

## Disable Profiling

```yaml
metadata:
  annotations:
    java-profiler.io/profile-disabled: "true"
```

Explicit disable wins over continuous and temporary enablement.

## Failure Statuses

- `disabled_by_metadata`: workload has no opt-in metadata or has explicit disable
- `unsupported_jvm`: process is not HotSpot-compatible
- `profiler_conflict`: another async-profiler user is detected
- `attach_failed`: collector could not attach to the JVM
- `upload_retryable`: backend or network failure can be retried
- `upload_dropped`: collector buffer overflow dropped old batches
- `storage_rejected`: backend rejected invalid or conflicting data

## Retention

Profile samples, stacks, thread snapshots, deadlock events, target status, and ingestion health have seven-day TTLs. Optional artifact index rows are retained for 24 hours maximum.

## Metrics

Collector and backend expose Prometheus-compatible metrics. Prometheus owns metric storage, dashboards, alerting, and retention. This system does not store Prometheus-style time series in ClickHouse.

Expected metric groups:

- target discovery and status counters
- profiler active, disabled, skipped, and failed counters
- upload success, retry, duplicate, and dropped-batch counters
- backend ingestion success and failure counters
- ClickHouse latency, table size, and TTL lag gauges
