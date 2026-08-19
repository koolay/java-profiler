# Java Profiling Runbook

This runbook covers the common lifecycle of a profiling target: enable it, check that the collector accepted it, investigate the result, and turn it off again.

## Enable continuous profiling

Add one of these metadata keys to a Java workload or Pod:

```yaml
metadata:
  annotations:
    java-profiler.io/profile-mode: continuous
    java-profiler.io/startup-delay: 30s
```

After the startup delay, the collector checks HotSpot compatibility and existing async-profiler use before it starts uploading normalized profiles.

## Enable temporary profiling

```yaml
metadata:
  annotations:
    java-profiler.io/profile-disabled: "false"
    java-profiler.io/profile-mode: temporary
    java-profiler.io/profile-duration: 10m
    java-profiler.io/startup-delay: 0s
    java-profiler.io/snapshot-interval: 10s
```

Temporary profiling stops when the duration expires. Use high-frequency thread snapshots only for a bounded temporary window.

The temporary window is measured against the target Pod/JVM lifecycle. Adding temporary metadata to a long-running Pod can therefore produce `temporary_expired` immediately. For a clean incident window, update the Pod template and roll the workload, or add a run-specific annotation such as `java-profiler.io/acceptance-run: "<timestamp>"` so Kubernetes creates a fresh Pod.

## Disable profiling

```yaml
metadata:
  annotations:
    java-profiler.io/profile-disabled: "true"
```

An explicit disable wins over both continuous and temporary enablement.

When re-enabling a workload that was previously disabled, remove the key or set `java-profiler.io/profile-disabled: "false"` on the Pod template. A stale truthy `profile-disabled` annotation keeps the target in `disabled_by_metadata` even when `profile-mode` is `temporary` or `continuous`.

## Validate an existing workload

For a production-shaped smoke test, keep the window short and save the before-and-after state. The real acceptance script can point the profiler Helm release at an existing workload without creating the synthetic BusyApp:

```bash
KUBECONFIG=/path/to/kubeconfig \
JAVA_PROFILER_COLLECTOR_INTERVAL=30s \
scripts/real-acceptance.sh \
  --configure-profiler \
  --namespace java-profiler-qa \
  --service jdk17-http-demo \
  --artifact-dir /tmp/java-profiler-jdk17-demo-$(date +%Y%m%d-%H%M%S) \
  --require-full-profiling
```

The script records target Pod state before and after the run and fails if the selected workload's restart count increases. With `--require-full-profiling`, only data created after the run starts counts; historical rows from earlier Pods or runs do not. Use `--skip-workload-rollout-check` only when `--service` is a label-level filter rather than a Deployment name.

After collector, backend, or Web changes, build images from the current workspace and deploy those exact tags before running full acceptance:

```bash
export BACKEND_IMAGE=java-profiler-backend:qa-$(date +%Y%m%d%H%M%S)
export COLLECTOR_IMAGE=java-profiler-collector:qa-$(date +%Y%m%d%H%M%S)
export WEB_IMAGE=java-profiler-web:qa-$(date +%Y%m%d%H%M%S)

bash scripts/build-real-acceptance-images.sh

KUBECONFIG=/path/to/kubeconfig \
BACKEND_IMAGE="$BACKEND_IMAGE" \
COLLECTOR_IMAGE="$COLLECTOR_IMAGE" \
WEB_IMAGE="$WEB_IMAGE" \
scripts/real-acceptance.sh \
  --service jdk17-http-demo \
  --configure-profiler \
  --require-full-profiling \
  --high-volume \
  --artifact-dir /tmp/java-profiler-real-acceptance-$(date +%Y%m%d%H%M%S)
```

If a previous run left async-profiler loaded in the target JVM and the status becomes `profiler_conflict`, roll the target Pod before retrying.

## Statuses and common failures

- `disabled_by_metadata`: workload has no opt-in metadata or has explicit disable
- `unsupported_jvm`: process is not HotSpot-compatible
- `profiler_conflict`: another async-profiler user is detected
- `temporary_expired`: temporary profile window has expired; roll the Pod or start a fresh temporary window
- `attach_failed`: collector could not attach to the JVM
- `upload_retryable`: backend or network failure can be retried
- `upload_dropped`: collector buffer overflow dropped old batches
- `storage_rejected`: backend rejected invalid or conflicting data

If the status is `accepted` but profiles remain empty, check backend ingestion health before changing the workload. Rejected batches often point to a collector/backend payload mismatch or ClickHouse schema drift.

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
