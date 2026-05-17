# Quickstart

Use this path when you already have `java-profiler` deployed and want to profile one Java service.

## 1. Enable profiling

Add profiling metadata to the target workload pod template:

```yaml
metadata:
  annotations:
    java-profiler.io/profile-mode: temporary
    java-profiler.io/profile-duration: 15m
```

Use `temporary` for incident work. Use `continuous` only for services approved for ongoing collection.

## 2. Open the service

In the Web UI, set:

- `Namespace`: the Kubernetes namespace.
- `Service`: the service or workload name.
- `Range`: the time window that includes the profiling run.

Start with [Target status](../operations/java-profiling-runbook.md#validate-an-existing-workload) if the UI looks empty. It explains whether the JVM was accepted, disabled, unsupported, expired, or failed attach.

## 3. Analyze the profile

Open [CPU profile analysis](../operations/performance-analysis-user-manual.md) first when investigating high CPU.

Use:

- Top Table to find the most expensive Java methods.
- Flame Graph to see full sampled stack context.
- Selected frame details to compare Self CPU and Total CPU.
- Search and Focus to isolate the stack path that matters.

For latency that is not explained by CPU, switch to Wall Clock. For socket or file blocking, switch to I/O wait. For pause-time or allocation-pressure incidents, switch to GC pauses and allocation correlation. For contention, switch to lock diagnosis.

## 4. Check ingestion health

Before trusting a missing profile, check [Ingestion health](../operations/performance-analysis-user-manual.md#ingestion-health). A useful diagnosis needs accepted profile batches for the selected service and time range.

## 5. Turn profiling off

Temporary profiling expires automatically. For continuous profiling, remove or disable the metadata when the service no longer needs ongoing collection.
