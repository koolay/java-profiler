---
title: "optimize: OOM and Allocation Pressure Investigation Workflow"
type: optimization
status: completed
created: 2026-06-11
origin: Coralogix research, current implementation review, and plan-eng-review
---

# optimize: OOM and Allocation Pressure Investigation Workflow

## Summary

Optimize `java-profiler` around a concrete Kubernetes Java incident path: a service has rising memory, GC pressure, or `OOMKilled` restarts, and the operator needs code-level allocation evidence without taking heap dumps. The current product already has the core pieces: opt-in Kubernetes targeting, async-profiler/JFR-derived allocation data, GC event evidence, target status, ingestion health, ClickHouse retention, and an allocation-focused UI. This plan connects those pieces into a first-class memory-pressure investigation workflow.

Recommendation: implement the complete workflow by composing existing evidence first, then add backend aggregation only after the UI proves which data shape is hard to consume. This preserves the current architecture and avoids creating a parallel incident store.

---

## Research Input

Coralogix, "Stop Guessing Why Your Pods Are Crashing" (Jonny Steiner, 2026-06-09), argues that Kubernetes metrics and CPU profiles are not enough for Java pod crash diagnosis:

- CPU metrics can look healthy while memory silently reaches the container limit and the kernel kills the process.
- Heap dumps are a weak production-first strategy because the pod may crash before the dump triggers, or dump overhead can worsen the incident.
- Allocation profiling closes the code-level gap by identifying allocation-driven pressure, object churn, and the Java call paths that create it.
- The most useful workflow pivots from infrastructure symptoms to allocation flame graphs and top allocating code paths.

Source: https://coralogix.com/blog/stop-guessing-why-your-pods-are-crashing/?ref=dailydev

---

## Current Implementation Inventory

Implemented capabilities found in the current worktree:

- Collector and backend are real Go packages under `cmd/collector`, `collector/internal/*`, `cmd/backend`, and `backend/internal/*`.
- Kubernetes opt-in policy supports annotations and labels such as `java-profiler.io/profile-mode`, `profile-disabled`, `profile-duration`, `startup-delay`, and `snapshot-interval`.
- Profile types include CPU, allocation bytes, allocation objects, lock contention count, lock delay, wall clock, and Java I/O wait.
- The async-profiler runner supports JFR collection with allocation and lock configuration, plus wall-clock profiling unless disabled.
- JFR normalization maps events into stable profile sample types and JVM events such as `gc_pause`.
- Backend ingestion accepts profile batches, JVM event batches, target status batches, thread snapshots, collector heartbeats, and ingestion health evidence.
- ClickHouse schema uses TTL deletion and stores profiles, thread snapshots, deadlocks, JVM events, target status, ingestion batches, and artifact index data.
- Query APIs expose flame graph, top stacks, allocation summary, JVM event evidence, target status, ingestion health, service selectors, thread diagnosis, and deadlocks.
- The web UI has views for allocation profiles, CPU, wall clock, I/O, GC, locks, deadlocks, target status, and ingestion health.
- The memory view already shows allocation flame graph evidence, allocation summary cards, top allocating paths, top self allocating frames, partial-result warnings, scope preflight, and 7-day retention guardrails.
- Real acceptance scripts and docs exist for Kubernetes deployment, JDK17 demo workload, real profile evidence, ClickHouse rows, browser UI workflow, and bounded retention.

---

## Engineering Review Findings

The first draft was directionally right but not implementation-ready.

1. **Architecture gap:** it said "combined evidence panel" without a concrete data flow or ownership boundary.
2. **API gap:** it did not decide when to compose existing endpoints versus create a backend aggregation endpoint.
3. **Collector gap:** it proposed OOM/restart status vocabulary before proving whether current discovery/status data already carries enough evidence.
4. **Test gap:** validation listed broad commands but not code paths, edge cases, or expected test files.
5. **Acceptance gap:** it mentioned an optional restart scenario, which conflicts with the existing strict acceptance expectation that target workload restart count must not increase unless a separate, explicitly isolated failure scenario is created.

The fixed plan below addresses these gaps.

---

## Gap Analysis

| Coralogix lesson | Current state | Gap |
| --- | --- | --- |
| OOM diagnosis starts from a pod crash or memory ceiling symptom. | Target status and ingestion health exist, but the UI does not present a dedicated OOM/restart investigation path. | Operators must manually combine status, GC, ingestion, and allocation views. |
| Allocation pressure must be first-class, not secondary to CPU. | Allocation profile types, allocation summary, and memory UI exist. | Navigation is profile-type oriented rather than incident oriented. |
| Heap dumps are too heavy for the default production path. | The product correctly does not require heap dumps. | Documentation should explicitly position async-profiler allocation sampling as the preferred production path. |
| Memory growth needs code-level allocation paths. | Top allocating paths and self frames exist. | UI should connect allocation rows to object churn/leak hypotheses while preserving "sampled allocation source, not retained heap ownership" semantics. |
| Infrastructure symptoms and code evidence must be correlated. | GC event evidence, target status, ingestion health, and allocation data are separate views/endpoints. | Need one workflow that shows status, freshness, GC, allocation totals, top paths, and flame graph context together. |
| Production overhead and missing data must be governed. | Profile limits, partial metadata, status reasons, ingestion health, and retention TTL exist. | Need explicit user-facing states for disabled, partial, truncated, stale, unsupported, or outside-retention allocation evidence. |

---

## Target Data Flow

The first implementation should compose existing evidence in the web app. Add backend aggregation only if the component becomes fragile or chatty after tests.

```text
Kubernetes Pod/Container status
          │
          ▼
collector discovery/status ───────┐
          │                        │
          ▼                        │
target-status batch                │
                                   │
async-profiler/JFR ───────┐        │
          │               │        │
          ▼               ▼        ▼
profile batch       JVM event batch ingestion batch
          │               │        │
          └───────────────┴────────┘
                          │
                          ▼
                      ClickHouse TTL
                          │
                          ▼
backend query APIs:
  /flamegraph
  /allocation-summary
  /jvm-events
  /target-status
  /ingestion
                          │
                          ▼
web/src/features/memory-pressure/*
  evidence strip + allocation summary + flame graph + focus links
```

Ownership boundary:

- Collector owns target discovery, profiling, and optional Kubernetes restart/OOM status observation.
- Backend owns storage, query limits, retention, and structured evidence responses.
- Web owns the first combined workflow by composing existing endpoints.

Backend aggregation decision:

- **Do not add a new endpoint in Phase 1.** Use existing endpoints through `web/src/api/client.ts`.
- Add `GET /api/query/v1/memory-pressure` only if Phase 1 needs repeated cross-endpoint derivation that causes inconsistent states or duplicated policy in multiple components.
- If a backend endpoint is added later, it must be read-only aggregation over existing ClickHouse tables, not a new write model.

---

## Optimization Plan

### Phase 1: Memory-Pressure Workflow Entry

Goal: make the product answer "why is this Java service running out of memory?" without requiring users to assemble the path from separate tabs.

Implementation targets:

- Add a `memory-pressure` view or mode in `web/src/routes/service-overview.tsx`.
- Keep the existing `memory` allocation profile view; do not remove CPU, wall, I/O, GC, lock, deadlock, status, or ingestion tabs.
- Implement a new composed component under `web/src/features/memory-pressure/` that calls:
  - `getTargetStatus`,
  - `getIngestionHealth`,
  - `getJVMEvents` with `event_type=gc_pause`,
  - `getAllocationSummary`,
  - `getFlamegraph` with `profile_type=java_allocation_bytes`.
- Render a compact evidence strip above the allocation analysis:
  - target desired state and latest status reason,
  - last accepted profile batch,
  - allocation summary total sampled bytes and sample count,
  - GC pause count and total pause duration,
  - partial/truncated/stale/outside-retention warnings,
  - restart or OOM evidence when available.
- Default the workflow to the selected service/Pod and time range. If no Pod is selected, show service-level evidence but prompt for Pod narrowing when allocation summary guardrails require it.

### Phase 2: Kubernetes Crash Context

Goal: connect Kubernetes failure symptoms to Java allocation evidence while staying inside the profiling product boundary.

Implementation targets:

- First inspect whether `collector/internal/discovery/pod_watcher.go` and target status ingestion can already observe container `restartCount`, `lastState.terminated.reason`, and `lastState.terminated.exitCode`.
- If the current collector cannot represent this, extend the target status vocabulary in `domain/types.go`, `contracts/profiling/payloads.md`, and backend/web type mappings with narrowly scoped reasons:
  - `container_restarted`,
  - `oom_killed_seen`,
  - `profiling_window_after_restart`.
- Store these as target status evidence with the existing 7-day TTL. Do not add a new incident table in Phase 2.
- Treat Kubernetes crash evidence as optional. Allocation profiling remains valid when pod status evidence is unavailable due to RBAC, retention, or timing.

### Phase 3: Allocation Interpretation

Goal: turn allocation tables into actionable Java hypotheses without pretending sampled allocation equals retained heap.

Implementation targets:

- Add deterministic insight categories derived from `AllocationSummary`:
  - high total path allocation,
  - high self allocation frame,
  - allocation concentration in a small number of paths,
  - GC pauses present with high allocation,
  - GC pauses present but allocation evidence missing or stale,
  - allocation evidence present with no GC pause evidence in the selected window.
- Add focus links from top allocating paths/self frames into the flame graph when the frame exists.
- Preserve current wording that allocation profiles identify object creation sources, not retained heap ownership.
- Surface partial-result metadata next to every insight that depends on truncated evidence.

### Phase 4: Documentation and Acceptance

Goal: make the workflow a tested product promise.

Documentation targets:

- Update `README.md` so allocation profiling is framed as the code-level answer to OOM, memory climb, and object churn.
- Update `docs/operations/performance-analysis-user-manual.md` with a "Pod OOM or memory climb" runbook:
  - select namespace/service/Pod,
  - verify target status and ingestion freshness,
  - inspect GC evidence,
  - inspect allocation summary and flame graph,
  - use CPU only when allocation pressure does not explain the incident.
- Update `docs/operations/real-profiling-acceptance-standard.md` only if the new workflow changes strict acceptance steps.

Acceptance targets:

- Extend the JDK17 demo workload or load script only enough to produce non-empty allocation and GC evidence without increasing target workload restarts during strict acceptance.
- If an actual OOM/restart scenario is added, keep it as a separate non-strict scenario with an isolated workload and explicit assertion that the restart is intentional.
- Real acceptance must still prove non-empty CPU, wall-clock, I/O, allocation, lock-delay, ClickHouse profile rows, JVM event rows for GC, ingestion evidence, bounded 7-day TTL, browser UI workflow behavior, and no unexpected target workload restart increase.

---

## NOT in Scope

- Heap dump collection as a required production path: conflicts with the low-overhead production diagnosis goal.
- Retained-object graph analysis: allocation sampling identifies creation pressure, not retained heap ownership.
- Prometheus-series storage, alerting, dashboards, tracing, or log analysis: these expand the product beyond profiling.
- Required Pyroscope, Parca, Grafana, Coralogix, or other external profile backend dependencies: ClickHouse remains the query store.
- Non-Java profiling: the current product boundary is Java services on Kubernetes.
- A new incident write model or incident table: current target status, JVM event, ingestion, and profile tables are sufficient for the first implementation.

---

## Test Coverage Plan

Detected test commands from repo docs and package scripts:

- Go: `go test ./...`
- Web: `cd web && npm test && npm run build`
- Strict real acceptance when implementation touches collector, ingestion, ClickHouse, backend query APIs, deployment, demo workload, or profile UI:
  `export KUBECONFIG=$HOME/backup/localk8s.yaml && scripts/real-acceptance.sh --require-full-profiling --artifact-dir /tmp/java-profiler-real-acceptance-$(date +%Y%m%d%H%M%S)`

Code path coverage required before implementation is complete:

```text
CODE PATH COVERAGE

ServiceOverview memory-pressure navigation
  ├── [GAP] Tab/mode selection preserves namespace/service/Pod/time range
  ├── [GAP] Share/permalink includes memory-pressure view and allocation profile type
  └── [GAP] Existing memory/cpu/wall/io/gc/locks/status/ingestion views still route

MemoryPressureView composed evidence loading
  ├── [GAP] All evidence endpoints populated
  ├── [GAP] Allocation summary preflight rejects broad/invalid scope before API call
  ├── [GAP] One endpoint fails while others render with targeted warning
  ├── [GAP] No allocation samples but status/ingestion explains disabled or stale data
  ├── [GAP] GC events present and allocation evidence present
  ├── [GAP] GC events present but allocation evidence missing
  ├── [GAP] Partial/truncated allocation summary marks dependent insights partial
  └── [GAP] Outside 7-day retention shows retention-specific guidance

Kubernetes restart/OOM target status
  ├── [GAP] No restart evidence: workflow remains useful
  ├── [GAP] restartCount increased with OOMKilled reason
  ├── [GAP] restartCount increased with non-OOM reason
  ├── [GAP] missing RBAC or missing pod status does not block allocation analysis
  └── [GAP] status reason contract round-trips collector -> backend -> web

Allocation insight derivation
  ├── [GAP] high total path allocation insight
  ├── [GAP] high self frame allocation insight
  ├── [GAP] concentrated allocation insight
  ├── [GAP] GC + allocation pressure insight
  ├── [GAP] GC present but allocation stale/missing insight
  └── [GAP] no false retained-heap ownership wording

Real acceptance
  ├── [GAP] non-empty allocation summary and flame graph
  ├── [GAP] non-empty GC/JVM event evidence
  ├── [GAP] ingestion evidence visible in workflow
  ├── [GAP] focus/search/reset/selected-frame behavior still works
  └── [GAP] target workload restart count does not unexpectedly increase
```

Expected tests to add or update:

- `web/src/routes/service-overview.test.tsx`: memory-pressure route selection, permalink/share params, existing tabs unchanged.
- `web/src/features/memory-pressure/memory-pressure-view.test.tsx`: populated, empty, partial, stale, endpoint-error, retention, and mixed-evidence states.
- `web/src/features/memory-pressure/memory-pressure-insights.test.ts`: deterministic insight category derivation and wording.
- `web/src/api/types.ts` type tests or compile coverage for any new target status reasons.
- `backend/internal/app/query_status_test.go` if target status query semantics change.
- `backend/internal/httpapi/query_handlers_test.go` only if a backend aggregation endpoint is added.
- `collector/internal/discovery/pod_watcher_test.go` or nearby discovery tests if restart/OOM evidence is collected from Kubernetes pod status.
- `domain/types_test.go` for any new `StatusReason` values and JSON contract behavior.
- `scripts/real-acceptance.sh` assertions if the browser acceptance path gains a distinct memory-pressure workflow.

---

## Failure Modes and Handling

| Failure mode | User-visible behavior | Required coverage |
| --- | --- | --- |
| Allocation profiling disabled by metadata or unsupported JVM. | Evidence strip shows target status reason and does not imply no allocation pressure. | Web component test with target status reason. |
| Allocation summary rejected by scope guardrails. | UI preflight prevents the invalid call and asks user to narrow service/Pod or time range. | Web test for namespace-only and retention range. |
| JVM GC events missing while allocation data exists. | Workflow says no GC pause evidence in this window, not "no GC happened." | Web test with allocation data and empty events. |
| GC events exist while allocation data is stale/missing. | Workflow points to ingestion/status evidence before recommending code interpretation. | Web test with GC events plus stale ingestion or no samples. |
| One evidence endpoint times out or fails. | Other panels render; failed panel has a scoped warning. | Web test with one rejected promise. |
| Pod restart evidence is unavailable due to RBAC or retention. | Restart/OOM card is marked unavailable; allocation analysis remains available. | Collector/backend test if status reason added; web unavailable-state test. |
| Partial allocation query truncates top paths or nodes. | Insights and tables show partial warning tied to `partial_reasons`. | Web test with partial summary metadata. |
| Real acceptance workload restarts unexpectedly. | Acceptance fails unless the scenario is explicitly isolated and intentional. | Script assertion retained or extended. |

---

## Implementation Order

1. Add `memory-pressure` navigation and composed UI using existing APIs only.
2. Add insight derivation and tests in a pure web module.
3. Add focused UI tests for populated, mixed, empty, partial, stale, and retention states.
4. Inspect collector pod status data. Add target status reasons only if current data cannot prove restart/OOM context.
5. If status reasons are added, update `domain`, `contracts`, collector, backend, web types, and tests in one patch.
6. Update README, user manual, and acceptance standard language.
7. Extend real acceptance browser checks for the memory-pressure workflow without weakening the no-unexpected-restart invariant.

---

## Validation Checklist

- `go test ./...`
- `cd web && npm test && npm run build`
- Any new backend query/status contract has Go tests covering accepted, missing, invalid, and stale data paths.
- Any new UI branch has component tests covering success, empty, partial, endpoint failure, broad scope, retention, and mixed-evidence states.
- Strict real acceptance passes when implementation touches collector profiling, ingestion, ClickHouse, backend query APIs, deployment, demo workload, or profile UI.
- The workflow never requires heap dumps, Prometheus storage, tracing, logs, Pyroscope, Parca, Grafana, or Coralogix.

---

## Implementation Notes

Completed in this implementation:

- Added a composed `memory-pressure` workbench view that reuses target status, ingestion health, JVM GC events, allocation summary, and allocation flamegraph APIs.
- Added deterministic memory-pressure insight derivation for high total allocation, high self-frame allocation, allocation concentration, GC plus allocation pressure, and missing allocation evidence when GC pauses exist.
- Added Kubernetes restart/OOM crash-context status reasons through the collector target-status path: `container_restarted`, `oom_killed_seen`, and `profiling_window_after_restart`.
- Updated contracts, README, performance-analysis manual, and real browser acceptance coverage for the OOM/memory-pressure workflow.
- Kept heap dumps, new incident tables, and external profiling backends out of scope.
