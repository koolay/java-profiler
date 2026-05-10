# Performance Ingestion Architecture Review

Date: 2026-05-10

## Purpose

This review records the performance architecture findings from real Kubernetes profiling acceptance. The current implementation can pass strict acceptance after batch splitting and sampling tuning, but the architecture should not stop there. The next design step is to make ingestion bounded, backpressure-aware, and query-efficient under production profile volume.

This document uses Pyroscope and Coroot as learning references, not as required backend dependencies.

## What We Learned From Pyroscope

Pyroscope does not treat every profile event as an unbounded row stream into a general table.

Key architecture lessons:

- The write path separates ingest routing from durable storage. In Pyroscope v2, distributors receive profile ingest requests and route them to segment writers.
- Segment writers accumulate profiles into small blocks or segments, then durably store them and update metadata.
- Query execution is separated into query frontend and backend roles, with planning and parallel execution because a flamegraph query can require expensive post-processing.
- Storage uses block/head concepts and compaction to reduce read amplification.
- The UI is built around profile semantics: top table, flame graph, self versus total, search highlighting, and focus/drilldown.

Implication for this project:

- We should not model the long-term hot path as raw JFR event rows flowing directly from collector to ClickHouse.
- The stable query path should operate on stack-level profile aggregates for a profile window.
- Raw artifacts can be retained briefly for debugging, but they should not be the primary UI query source.

Sources:

- Grafana Pyroscope v2 architecture: https://grafana.com/docs/pyroscope/latest/reference-pyroscope-v2-architecture/about-pyroscope-v2-architecture/
- Grafana Pyroscope HTTP API: https://grafana.com/docs/pyroscope/latest/reference-server-api/
- Grafana Pyroscope disk storage: https://grafana.com/docs/pyroscope/latest/configure-server/storage/configure-disk-storage/
- Local UI study: `docs/research/pyroscope-profile-ui-study.md`

## What We Learned From Coroot

Coroot's Java profiling path is close to this project's target shape.

Key architecture lessons:

- The node agent dynamically loads async-profiler into HotSpot JVMs without app code changes.
- It collects CPU, allocation, and lock contention profiles in one async-profiler session.
- It periodically stops async-profiler, reads a finalized JFR file, parses it, uploads profiles, and quickly restarts profiling.
- It detects already-loaded async-profiler libraries by scanning `/proc/<pid>/maps` and skips conflicting JVMs.
- ClickHouse is used for profile storage, and UI analysis is flamegraph-oriented.

Implication for this project:

- The stop/read/restart model is valid because JFR output needs finalized chunks.
- Conflict detection is necessary, but our runtime needs clearer handling for orphaned local sessions after collector restart.
- Allocation and lock profiling must be bounded because they can produce much higher event volume than CPU samples.

Sources:

- Coroot profiling overview: https://docs.coroot.com/profiling/overview/
- Coroot node-agent configuration: https://docs.coroot.com/configuration/coroot-node-agent/
- Coroot Java profiling article: https://dev.to/coroot/profiling-java-apps-breaking-things-to-prove-it-works-14da
- Local Coroot research: `docs/research/coroot-node-agent-java-agent.md`

## Real Acceptance Failure Modes Observed

The real Kubernetes acceptance loop exposed production-relevant failure modes:

1. A single profile upload can exceed backend request limits.

   Initial collector uploads produced very large JSON payloads. Splitting profile batches fixed the immediate request-size failure, but splitting alone does not reduce total data volume.

2. ClickHouse can OOM under unbounded profile sample volume.

   Allocation profiling with a low sampling interval produced hundreds of thousands of samples in one run. A 2Gi ClickHouse pod was OOMKilled during ingest/query.

3. Lock profiling needs real contention.

   A single synchronized request often does not produce lock-delay events because no other thread blocks on the monitor. Acceptance must drive concurrent lock requests.

4. Collector restart can create stale async-profiler conflict state.

   The target JVM may still have `libasyncProfiler.so` loaded while the new collector process has no in-memory session record. The system can report `profiler_conflict` even though the loaded profiler may be from the same logical installation.

5. UI acceptance can pass only if backend data is meaningful.

   A non-crashing UI is insufficient. The top table and flamegraph must be backed by real accepted profile data for the current run window.

## Current Mitigations

The current implementation includes short-term stabilizers:

- collector-side profile batch splitting
- higher allocation sampling interval
- concurrent lock workload in real acceptance
- real acceptance wait for target-status rows
- larger ClickHouse memory limit for the real acceptance installation
- strict acceptance checks for non-empty CPU, allocation, and lock-delay flamegraphs

These are necessary, but they are not the final performance architecture.

## Target Architecture Direction

### 1. Aggregate Before Upload

Collector should aggregate parsed JFR events by:

- cluster
- namespace
- service
- pod
- container
- process id
- JVM start time
- profile type
- profile window start/end
- stack id

The uploaded payload should contain stack-level values, not every raw event.

Expected result:

- lower payload bytes
- lower ClickHouse row count
- lower backend memory pressure
- more stable top-table and flamegraph query latency

### 2. Make Ingestion Explicitly Bounded

Collector and backend need hard limits:

- max payload bytes per request
- max stacks per batch
- max profile events parsed per window
- max retry queue bytes
- max buffered batches
- max per-target upload concurrency

When limits are exceeded, the system must record visible state:

- truncated profile
- dropped samples
- dropped stacks
- retryable backend failure
- non-retryable rejected payload
- ClickHouse unavailable

This state must appear in target status or ingestion health so users can distinguish "no hotspot" from "data was dropped."

### 3. Keep Raw Artifacts Optional And Short-Lived

Raw JFR or pprof artifacts are valuable for debugging parser and ingestion bugs, but they should be:

- optional
- short-retention
- excluded from the normal UI query path
- bounded by bytes and time

The default UI path should use normalized stack aggregates.

### 4. Query Aggregates, Not Raw Events

Backend query APIs should be able to answer these from ClickHouse aggregates:

- flamegraph tree
- top table
- self and total values
- profile-type-specific totals
- scanned/omitted/truncated metadata

The browser should never need to process large raw row sets.

### 5. Handle Async-Profiler Session Ownership

The collector should distinguish:

- active session owned by this collector process
- known local profiler session orphaned after collector restart
- external async-profiler conflict
- unknown loaded async-profiler library

Possible next design options:

- record session ownership metadata in the target temp directory
- attempt safe stop/cleanup for known local orphaned sessions
- expose orphaned-session state in status and require target restart only as a fallback

### 6. Treat ClickHouse As A Shared Constrained Resource

The first version assumes single-node ClickHouse. Therefore:

- profile retention must remain bounded to 7 days or less
- profile inserts must be batched and aggregated
- queries must be scoped by service, profile type, and time range
- acceptance should include high-volume profile ingestion without OOM
- backend/collector metrics must expose write failures, retry pressure, dropped data, and query health

## Proposed Phased Plan

### Phase 1: Bounded Ingestion Contract

- Add explicit payload and stack limits to collector and backend.
- Add ingestion result fields for dropped/truncated/rejected state.
- Show limit and dropped-data evidence in the ingestion UI.
- Add tests for oversized payloads and retryable backend failures.

### Phase 2: Collector-Side Stack Aggregation

- Aggregate JFR events into stack-level profile samples before JSON encoding.
- Preserve profile-type-specific value units.
- Keep raw event counts as metadata.
- Validate that strict real acceptance still produces CPU, allocation, and lock-delay data with lower ClickHouse row counts.

### Phase 3: Query Path Hardening

- Add top-table query endpoint that does not require the UI to infer everything from a full flamegraph.
- Return scanned row count, omitted node count, and truncated indicators.
- Add query limits and clear empty/error states.

### Phase 4: Session Ownership And Recovery

- Persist local async-profiler session markers.
- Recover or stop known orphaned local sessions after collector restart.
- Keep external-profiler conflict behavior conservative.

### Phase 5: High-Volume Acceptance Scenario

Extend `docs/operations/real-profiling-acceptance-standard.md` with a high-volume scenario:

- allocation-heavy workload
- concurrent lock contention
- bounded ClickHouse memory
- non-empty CPU/allocation/lock-delay profiles
- no ClickHouse OOM
- ingestion UI reports any truncation/drop explicitly

## Architectural Decision

The current batch-splitting fix is accepted as a short-term stabilization. It is not the long-term architecture.

The long-term direction is:

> Collect JFR, parse locally, aggregate by stack and profile window before upload, enforce explicit ingestion limits, store bounded ClickHouse aggregates, and make every truncation/drop/retry visible to the user.

This keeps the product aligned with the project boundary: a focused Java profiler on Kubernetes with ClickHouse storage and self-owned UI, without requiring Pyroscope, Coroot, Grafana, or Parca as runtime dependencies.
