# Quickstart

Use this guide when `java-profiler` is already deployed and you want to investigate one Java service.

## 1. Enable profiling

Add profiling metadata to the target workload pod template:

```yaml
metadata:
  annotations:
    java-profiler.io/profile-mode: temporary
    java-profiler.io/profile-duration: 15m
```

Use `temporary` for an incident or a short investigation. Use `continuous` only for services that have been approved for ongoing collection.

## 2. Open the service

In the Web UI, set:

- `Namespace`: the Kubernetes namespace.
- `Service`: the service or workload name.
- `Range`: the time window that includes the profiling run.

If the UI looks empty, start with [Target status](../operations/java-profiling-runbook.md#validate-an-existing-workload). It tells you whether the JVM was accepted, disabled, unsupported, expired, or rejected because attach failed.

If a profile view has no samples, read the Profile evidence status banner before widening the query. It combines target status with aggregate ingestion information and usually points to disabled profiling, an expired temporary window, an unmatched target, an ingestion gap, or a time range with no samples.

## 3. Analyze the profile

Open [CPU profile analysis](../operations/performance-analysis-user-manual.md) first when investigating high CPU.

Use:

- Top Table to find the most expensive Java methods.
- Both mode to keep the Top Table and Flame Graph visible while you move from ranked symbols to stack context.
- Flame Graph to see full sampled stack context.
- Selected frame details to compare Self CPU and Total CPU.
- Search and Focus to isolate the stack path that matters.

For allocation pressure, open Allocation profiles. Start with Allocation Summary, then compare Top allocating paths, Top self allocating frames, and the flamegraph context. This is sampled object-creation data, not retained-heap ownership.

For latency that is not explained by CPU, switch to Wall Clock. For socket or file blocking, switch to I/O wait. For pause-time incidents, switch to GC pauses and allocation correlation. For contention, switch to lock diagnosis.

## 4. Check ingestion health

Before trusting a missing profile, check [Ingestion health](../operations/performance-analysis-user-manual.md#check-ingestion-health). A useful diagnosis needs accepted profile batches for the selected service and time range. Ingestion health is aggregate backend evidence, so still confirm target status and the selected namespace, service, Pod, and time range.

## 5. Turn profiling off

Temporary profiling expires automatically. For continuous profiling, remove or disable the metadata when the service no longer needs ongoing collection.
