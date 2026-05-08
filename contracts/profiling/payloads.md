# Profiling Contracts

This document defines the stable payload vocabulary used by the first Java-profiler implementation.

## Target identity

Every target payload carries a stable identity with:

- `cluster`
- `namespace`
- `workload`
- `pod`
- `container`
- `node`
- `pod_uid`
- `process_id`
- `jvm_start_time`
- `runtime_vendor`
- `runtime_version`
- `service`

`jvm_start_time` is part of the identity because process ids can be reused.

## Enablement vocabulary

Enablement is expressed through Kubernetes annotations or labels.

- `java-profiler.io/profile-mode`: `disabled`, `continuous`, or `temporary`
- `java-profiler.io/profile-disabled`: boolean truthy value that forces disabled mode
- `java-profiler.io/profile-duration`: a Go-style duration string for temporary mode
- `java-profiler.io/startup-delay`: delay before profiling newly discovered JVMs
- `java-profiler.io/snapshot-interval`: thread snapshot interval for the collector

Explicit disable wins over broader enablement. Temporary enablement wins over continuous enablement when both are present.

## Profile types

Stable v1 profile types:

- `java_cpu_nanoseconds`
- `java_allocation_bytes`
- `java_allocation_objects`
- `java_lock_contention_count`
- `java_lock_delay_nanoseconds`

## Batch types

Stable batch types:

- `profile`
- `thread_snapshot`
- `target_status`
- `collector_heartbeat`
- `ingestion`
- `retention`
- `artifact_index`

## Status reasons

Stable status reasons used by collector and backend:

- `disabled_by_metadata`
- `temporary_expired`
- `invalid_duration`
- `unsupported_jvm`
- `profiler_conflict`
- `attach_failed`
- `upload_retryable`
- `upload_dropped`
- `storage_rejected`
- `accepted`

## Retention policy

Defaults stay at or under seven days.

- profile data: 7 days
- thread data: 7 days
- deadlock data: 7 days
- target status data: 7 days
- ingestion data: 7 days
- artifact index data: 24 hours maximum
