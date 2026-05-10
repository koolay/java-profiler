---
title: fix: Real Profiling Review Blockers
type: fix
status: completed
date: 2026-05-10
origin: docs/brainstorms/java-profiler-requirements.md
---

# fix: Real Profiling Review Blockers

## Summary

Close the review findings that can make the current Java profiler look correct while failing the real goal: collect authentic async-profiler data from the normal Kubernetes deployment and present it without fake evidence or misleading frame ownership.

## Problem Frame

The latest UI can show a Pyroscope-style CPU workflow, but code review found several correctness gaps below the surface. The normal collector image does not contain the async-profiler assets expected by the Helm chart, the runtime uploads synthetic thread/deadlock evidence, ingestion records idempotency after writing payload rows, CPU sample values are stored under a nanosecond contract as raw counts, and the UI can classify JVM adapter/runtime frames as application Java. These defects directly undermine real performance diagnosis.

## Requirements

- R1. The normal collector image and Helm chart must agree on where `asprof` and `libasyncProfiler.so` live, so production-style installs can collect real profiles.
- R2. Runtime collection must not upload fabricated thread snapshots or deadlock events. Only observed JVM evidence may be stored.
- R3. Backend ingestion must reject duplicate or conflicting batches before writing profile, status, thread, or deadlock rows, or otherwise guarantee idempotent query results.
- R4. CPU profile values must match their declared unit. `java_cpu_nanoseconds` cannot store raw sample counts without conversion or clear contract migration.
- R5. CPU Top Table must rank actionable application Java symbols and exclude JVM/native adapters such as `I2C.C2I adapters`; flame graph category styling must not mislabel dot-form JVM frames as application.
- R6. If no application Java frames are found, the CPU view must still render the flame graph so users can inspect runtime/native-heavy profiles.
- R7. Real acceptance must exercise a real profiling window with load and wait long enough for async-profiler start/stop/read cycles.

## Scope Boundaries

- Keep the project self-owned; do not add Pyroscope, Grafana, Parca, or other profile backend dependencies.
- Keep focus on Java services on Kubernetes.
- Do not introduce source-code lookup in this fix.
- Do not make Prometheus a profile storage dependency.
- Do not broaden thread/deadlock UI behavior beyond removing fake evidence and preserving honest empty states.

## Context

Relevant implementation files:

- `Dockerfile.collector`
- `deploy/helm/templates/collector-daemonset.yaml`
- `deploy/helm/values.yaml`
- `collector/runtime/runtime.go`
- `collector/internal/jfr/parser.go`
- `collector/internal/jfr/normalizer.go`
- `backend/internal/app/ingest_profile_batch.go`
- `backend/internal/app/ingest_target_status_batch.go`
- `backend/internal/app/ingest_thread_snapshot_batch.go`
- `backend/internal/clickhouse/sql_repository.go`
- `web/src/features/cpu/hot-code-view.tsx`
- `web/src/visualization/flamegraph.tsx`
- `scripts/real-acceptance.sh`

Related tests:

- `collector/runtime/runtime_test.go`
- `collector/internal/jfr/normalizer_test.go`
- `backend/internal/app/ingest_profile_batch_test.go`
- `backend/internal/app/ingest_target_status_batch_test.go`
- `backend/internal/app/ingest_thread_snapshot_batch_test.go`
- `web/src/features/cpu/hot-code-view.test.tsx`
- `web/src/visualization/flamegraph.test.tsx`
- `web/tests/real-acceptance.spec.ts`

## Key Decisions

- Package async-profiler through the same deployment path that operators use. Acceptance-only image assembly is not enough because it can verify a different artifact than Helm deploys.
- Prefer honest absence over synthetic evidence. Empty deadlock/thread views are acceptable; false positives are not.
- Use batch idempotency as an application-layer contract before writes. ClickHouse does not provide the same transactional guarantees as the in-memory stores, so handlers must avoid writing payload rows for rejected batch ids.
- Keep CPU values internally meaningful now. If the external contract says nanoseconds, convert execution samples using the configured sample interval or rename the type only with a deliberate contract migration.
- Preserve Pyroscope-inspired interaction semantics: Top Table is for ranked symbols, flame graph is sampled stack context, and self/total need to be interpretable without pretending runtime/native frames are Java owners.

## Implementation Units

### U1. Align Collector Image and Helm Assets

**Goal:** Make the standard collector image usable with chart defaults for real async-profiler collection.

**Files:**
- Modify: `Dockerfile.collector`
- Modify: `deploy/helm/values.yaml`
- Modify: `deploy/helm/templates/collector-daemonset.yaml`
- Modify as needed: `scripts/build-real-acceptance-images.sh`

**Approach:**
- Package pinned async-profiler `asprof` and `libasyncProfiler.so` into the collector image path used by Helm, or change Helm to mount/init those assets explicitly.
- Keep asset version/checksum in one source of truth where practical.
- Ensure local/CI build paths use mirrored base images where the project already requires them.

**Test scenarios:**
- `docker build` or static validation confirms the expected asset paths exist in the collector image.
- `helm template` renders env vars that point at the packaged paths.
- Acceptance image builder no longer validates a different async-profiler version/path from the chart.

### U2. Remove Synthetic Thread and Deadlock Evidence

**Goal:** Stop writing fake diagnostic data.

**Files:**
- Modify: `collector/runtime/runtime.go`
- Modify: `collector/runtime/runtime_test.go`
- Modify as needed: thread/deadlock docs or acceptance strictness

**Approach:**
- Remove `threadEvidence` from the production runtime loop.
- Upload thread snapshot batches only when a real snapshot collector provides observed JVM data.
- Adjust strict acceptance so deadlock data is required only when the workload intentionally creates a real deadlock and the collector observes it.

**Test scenarios:**
- Runtime profile collection does not call the thread snapshot endpoint when no real thread collector output exists.
- No synthetic `worker-blocked` or `deadlock-candidate` rows are produced.
- UI thread/deadlock views handle empty observed data honestly.

### U3. Fix Ingestion Idempotency Before Writes

**Goal:** Prevent duplicate or conflicting batches from polluting profile, status, and thread stores.

**Files:**
- Modify: `backend/internal/app/ingest_profile_batch.go`
- Modify: `backend/internal/app/ingest_target_status_batch.go`
- Modify: `backend/internal/app/ingest_thread_snapshot_batch.go`
- Modify as needed: `backend/internal/clickhouse/ingestion_repository.go`
- Modify as needed: `backend/internal/clickhouse/sql_repository.go`
- Modify tests in `backend/internal/app/*_test.go`

**Approach:**
- Check or claim `batch_id` and payload hash before payload inserts.
- Return duplicate for same hash and rejected for conflicting hash before any write.
- Normalize or validate child row `BatchID` fields against the envelope `BatchID`.
- Keep in-memory and SQL-backed behavior aligned.

**Test scenarios:**
- First batch writes payload rows and records accepted ingestion.
- Retrying the same batch returns duplicate and does not write additional payload rows.
- Reusing a batch id with different payload returns rejected and writes no payload rows.
- Child rows with empty or conflicting batch ids are normalized or rejected consistently.

### U4. Correct CPU Profile Unit Semantics

**Goal:** Make CPU values consistent with `java_cpu_nanoseconds`.

**Files:**
- Modify: `collector/internal/jfr/parser.go`
- Modify as needed: `collector/internal/jfr/normalizer.go`
- Modify: `collector/internal/jfr/normalizer_test.go`
- Modify as needed: contracts docs

**Approach:**
- Convert execution sample events to nanoseconds using the configured async-profiler sample interval, or route raw counts to a count-typed contract if a migration is chosen.
- Keep UI percent math stable; percentages should remain based on relative values.

**Test scenarios:**
- CPU execution sample normalization emits non-count nanosecond values for the default 10ms interval.
- Allocation bytes/object and lock values retain their existing semantics.
- Contract docs match emitted values.

### U5. Fix CPU Analysis Frame Classification and Empty State

**Goal:** Keep Hot Code actionable while preserving flame graph context.

**Files:**
- Modify: `web/src/features/cpu/hot-code-view.tsx`
- Modify: `web/src/visualization/flamegraph.tsx`
- Modify: `web/src/features/cpu/hot-code-view.test.tsx`
- Modify: `web/src/visualization/flamegraph.test.tsx`

**Approach:**
- Exclude JVM adapters, runtime stubs, whitespace-heavy VM pseudo symbols, and native symbols from the Java Top Table.
- Normalize slash and dot package forms before classifying runtime frames.
- Render flame graph even when Top Table has no application Java frames.

**Test scenarios:**
- `I2C.C2I adapters` and similar adapter frames do not appear in Hot Code.
- `java.lang.Thread.run`, `jdk.internal.*`, and `sun.*` classify as runtime, not application.
- Runtime/native-only profiles still show flame graph with an honest empty Top Table state.

### U6. Make Real Acceptance Wait for Real Data

**Goal:** Ensure acceptance proves a real profiling cycle under load.

**Files:**
- Modify: `scripts/real-acceptance.sh`
- Modify as needed: `scripts/deploy-jdk17-demo.sh`
- Modify: `web/tests/real-acceptance.spec.ts`

**Approach:**
- Drive demo CPU/allocation/lock load during the profiling window.
- Poll for accepted target status and non-empty profile rows/flamegraph values for at least two collector intervals.
- Align strict `--require-full-profiling` expectations with enabled profile types.

**Test scenarios:**
- Strict acceptance does not assert profile presence before a stop/read cycle can occur.
- CPU-only strict mode does not require disabled allocation/lock data.
- Full profiling mode enables allocation/lock collection before requiring those profiles.

## Verification Plan

- `go test ./backend/internal/app ./backend/internal/httpapi ./backend/internal/clickhouse ./collector/internal/jfr ./collector/runtime ./collector/internal/profiler`
- `npm test -- --run src/features/cpu/hot-code-view.test.tsx src/visualization/flamegraph.test.tsx`
- `npm run build`
- `helm template java-profiler deploy/helm`
- Real Kubernetes acceptance using `export KUBECONFIG=$HOME/backup/localk8s.yaml`, demo load, and browser E2E against the deployed web UI.

## Risks

- Packaging async-profiler in the collector image may require replacing the distroless runtime stage or adding only executable/native assets carefully.
- Pre-write idempotency for ClickHouse needs a practical design because inserts are not transactional in the same way as in-memory tests.
- Removing synthetic deadlock data may require updating strict acceptance expectations and documentation so empty deadlock state is accepted unless a real deadlock workload is present.
