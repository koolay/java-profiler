---
date: 2026-05-08
topic: java-k8s-performance-profiling
---

# Java Kubernetes Performance Profiling

## Summary

Build a focused performance analysis system for Java services running on Kubernetes. The first version uses a node-level agent, Kubernetes annotation controls, async-profiler-based Java profiling, ClickHouse storage, and a minimal self-owned UI for trend-to-flamegraph investigation.

---

## Problem Frame

The current reference system, Coroot node-agent, is powerful but broader than needed. The target need is narrower: help operators diagnose production Java service issues such as CPU hotspots, allocation pressure, GC pressure, and lock contention without restarting applications or adding application code changes.

The deployment environment is Kubernetes, and Java services run as Pods. Production safety matters more than complete fleet-wide coverage, so the system must default to controlled enablement rather than profiling everything automatically.

---

## Actors

- A1. Platform operator: installs the agent and configures cluster-level policies.
- A2. Java service owner: enables or disables profiling for a service using Kubernetes metadata.
- A3. Incident responder: temporarily enables profiling during an online performance incident and reads the result.
- A4. Profiling agent: runs on Kubernetes nodes, discovers local JVMs, executes profiling, and reports data.
- A5. Profiling backend: receives profiles and metrics, persists them in ClickHouse, and serves query results.

---

## Key Flows

- F1. Whitelist continuous profiling
  - **Trigger:** A service owner adds the agreed Kubernetes annotation or label to a Java workload.
  - **Actors:** A2, A4, A5
  - **Steps:** The node agent notices matching Pods on its node, discovers HotSpot JVMs, starts profiling after a safety delay, periodically uploads profiles, and reports profiler status.
  - **Outcome:** The service has continuous Java CPU, allocation, and lock profiling data available for review.
  - **Covered by:** R1, R2, R3, R4, R8

- F2. Temporary incident profiling
  - **Trigger:** An incident responder enables a temporary profiling annotation or label for a service or Pod.
  - **Actors:** A3, A4, A5
  - **Steps:** The node agent starts profiling only for the requested target, captures data for the configured time window, uploads results, and stops automatically when the window expires.
  - **Outcome:** The responder can inspect a bounded profile result without leaving profiling on indefinitely.
  - **Covered by:** R2, R3, R5, R6, R8

- F3. Trend-to-flamegraph investigation
  - **Trigger:** A user sees a spike in CPU, allocation rate, GC pressure, or lock wait.
  - **Actors:** A2, A3, A5
  - **Steps:** The user selects a service, time range, and profile type, sees related JVM trend data, and opens a flamegraph for the matching profile data.
  - **Outcome:** The user can connect a production symptom to the Java method stack responsible for it.
  - **Covered by:** R7, R8, R9, R10

- F4. Profiling shutdown
  - **Trigger:** A service owner removes the profiling annotation, sets an explicit disabled state, or the temporary window expires.
  - **Actors:** A2, A3, A4
  - **Steps:** The node agent stops async-profiler for matching JVMs and records the stopped status.
  - **Outcome:** Profiling no longer adds runtime overhead to the target JVMs.
  - **Covered by:** R5, R6, R11

- F5. Thread diagnosis snapshot
  - **Trigger:** A user investigates deadlock, slow requests, thread pool saturation, or high CPU in a Java service.
  - **Actors:** A2, A3, A4, A5
  - **Steps:** The user opens the service thread view or enables temporary profiling; the agent captures bounded JVM thread snapshots with thread state, stack trace, lock ownership, lock wait information, and deadlock detection output.
  - **Outcome:** The user can identify deadlocked threads, blocked threads, busy RUNNABLE threads, and their current Java stacks.
  - **Covered by:** R30, R31, R32, R33, R34

- F6. Memory pressure investigation
  - **Trigger:** A user sees allocation rate, heap usage, or GC time increase for a Java service.
  - **Actors:** A2, A3, A5
  - **Steps:** The user opens the service memory view, checks heap and GC trends, and drills into allocation bytes or allocation objects flamegraphs for the selected time range.
  - **Outcome:** The user can distinguish high allocation pressure from retained-heap analysis needs and identify the code path allocating memory.
  - **Covered by:** R9, R10, R24, R30

---

## Requirements

**Kubernetes control**
- R1. The system must support opt-in profiling using Kubernetes annotations or labels, rather than profiling every Java Pod by default.
- R2. The system must support both service-level and Pod-level targeting, so operators can profile a broad workload or a single problematic instance.
- R3. The system must distinguish continuous whitelist profiling from temporary incident profiling.
- R4. The system must support a startup delay before profiling newly discovered JVMs to avoid profiling JVM warmup by default.
- R5. The system must support explicit disable controls that stop profiling even when a broader selector would otherwise include the target.
- R6. Temporary profiling must have a bounded duration and automatically stop when the duration expires.

**Collection**
- R7. The collection agent must run as a Kubernetes DaemonSet so it can discover and attach to Java processes on its own node.
- R8. The first version must focus on HotSpot-compatible JVMs and report unsupported JVMs as skipped rather than failed.
- R9. The collected profile types must include Java CPU, allocation bytes, allocation objects, lock contention count, and lock wait time.
- R10. The system must also expose JVM trend signals needed to interpret profiles, including heap usage, GC time, safepoint time, profiling status, allocation rate, and lock wait rate.
- R11. The agent must detect and avoid profiling a JVM already using another async-profiler-based tool.
- R12. The agent must surface per-target status and failure reasons, including unsupported JVM, attach failure, conflict, disabled target, and successful profiling.

**Receiving and storage**
- R13. The backend must receive profile data from agents without depending on Pyroscope, Parca, Grafana, or other AGPL-incompatible profile backends.
- R14. ClickHouse must be the primary storage system for profile query data because the target environment already has a single-node ClickHouse deployment.
- R15. The storage model must preserve service, namespace, Pod, container, node, JVM identity, profile type, time range, labels, stack, and sample value.
- R16. The backend must retain enough structured profile data to render flamegraphs without re-attaching to the application JVM.
- R17. The system must define profile, thread snapshot, JVM metric, and optional artifact retention policies so production storage growth is bounded.
- R18. The system must clean expired ClickHouse data automatically through retention policy, without requiring manual table maintenance for normal operation.
- R19. The system must expose storage usage and cleanup health so operators can tell whether retention is working.
- R20. No collected data type may be retained for more than 7 days, including profile samples, JVM metrics, thread snapshots, deadlock events, target status, and optional raw artifacts.

**Viewing**
- R21. The UI must provide a service-centric view for Java performance analysis, not a general-purpose observability workspace.
- R22. The UI must show profiling status per service, Pod, and JVM.
- R23. The UI must show trend charts for CPU/profile-related symptoms and let users navigate from a spike to the relevant flamegraph.
- R24. The UI must render flamegraphs for Java CPU, allocation, and lock profile types.
- R25. The UI must support selecting a time range and narrowing to a service, Pod, or JVM where data exists.
- R26. The UI must help answer memory allocation, deadlock, slow-thread, and busy-thread questions from the same service context.

**Production safety**
- R27. Profiling must be off by default for all workloads unless explicitly enabled by Kubernetes metadata.
- R28. The system must make it easy to stop profiling quickly during an incident or after an investigation.
- R29. The system must expose agent and backend health signals sufficient to tell whether missing data is caused by no target, disabled profiling, collection failure, or storage/query failure.

**Diagnostic analysis**
- R30. The memory analysis view must combine heap usage, GC time, allocation rate, allocation bytes flamegraph, allocation objects flamegraph, and top allocating Java stack data for the selected service, Pod, or JVM.
- R31. The system must collect bounded thread snapshots that include timestamp, service identity, Pod identity, JVM identity, thread id, thread name, daemon flag when available, thread state, Java stack, native thread id when available, lock owner, blocked lock, waited lock, and deadlock cycle membership when detected.
- R32. The system must detect and display Java deadlocks from thread snapshot data, including every involved thread, the lock each thread waits for, the lock owner, and the stack frame where the thread is blocked.
- R33. The slow-thread view must identify threads spending time in BLOCKED, WAITING, TIMED_WAITING, park, monitor enter, or lock contention paths, and correlate those findings with lock wait profiles when profile data exists for the same time range.
- R34. The busy-thread view must identify threads and Java stacks consuming CPU or staying RUNNABLE during the selected time range, and correlate those findings with CPU profiles when profile data exists for the same time range.
- R35. The UI must make clear when a question cannot be fully answered by the collected data, such as retained object ownership without a heap dump or historical deadlock evidence after retention has expired.
- R36. Temporary profiling must be able to request higher-frequency thread snapshots for a bounded duration, while continuous whitelist profiling should use a lower default snapshot frequency or on-demand snapshots to reduce overhead.

---

## Default Retention Policy

All defaults are bounded at or below 7 days. The system may allow shorter retention per environment, but must not allow longer retention without changing the product requirement.

- Profile samples and stacks: 7 days.
- JVM trend metrics persisted in ClickHouse: 7 days.
- Thread snapshots and thread stacks: 7 days.
- Deadlock events derived from thread snapshots: 7 days.
- Target status and profiling health history: 7 days.
- Raw JFR, pprof, or thread-dump artifacts: disabled by default; if enabled for debugging, 24 hours maximum.

Expired ClickHouse data must be removed by table TTL or an equivalent automatic cleanup job. Cleanup health must be visible because this deployment assumes a single-node ClickHouse shared with logs.

---

## Viewing Technology Direction

The first version should use a self-owned lightweight Web UI rather than Pyroscope, Grafana, or a bundled third-party profiling console.

- Backend query API returns service summaries, JVM trend series, flamegraph trees, thread snapshots, deadlock events, and target status from ClickHouse.
- Time-series charts can use a permissively licensed chart library after license review, or a minimal in-house chart if dependency policy requires it.
- Flamegraphs should be rendered from backend-provided stack-tree JSON using a small self-owned SVG or Canvas renderer.
- Thread diagnosis should use purpose-built views: deadlock cycle graph or list, slow-thread table, busy-thread table, and stack trace panel.
- Every view should keep the same selectors: namespace, service, Pod, container, JVM, profile type, and time range.

This keeps the viewing layer narrow: it answers Java performance questions, but does not become a general observability dashboard.

---

## Acceptance Examples

- AE1. **Covers R1, R3, R7, R9.** Given a Java Deployment without profiling annotations, when the DaemonSet agent observes its Pod, no async-profiler session is started and the target is shown as not enabled.
- AE2. **Covers R1, R4, R8, R9.** Given a HotSpot Java Pod with the continuous profiling annotation, when the startup delay has elapsed, the node agent starts collecting CPU, allocation, and lock profiles.
- AE3. **Covers R5, R6.** Given a service has temporary profiling enabled for a bounded duration, when the duration expires or an explicit disable annotation is applied, the agent stops profiling that target.
- AE4. **Covers R11, R12.** Given a JVM already has another async-profiler library loaded, when the agent evaluates the target, it skips profiling and reports a conflict status.
- AE5. **Covers R13, R14, R16, R23.** Given a completed profile upload, when a user opens the profile view for the same service and time range, the backend queries ClickHouse and returns a flamegraph without requiring the original JVM.
- AE6. **Covers R21, R23, R25, R26.** Given allocation rate spikes for one Pod, when the user opens the Java service view and selects that time range, the UI lets the user drill into the Java allocation flamegraph for that Pod.
- AE7. **Covers R17, R18, R19, R20.** Given any collected data older than 7 days, when ClickHouse retention runs, expired data is removed and the backend reports cleanup health without manual intervention.
- AE8. **Covers R30.** Given allocation bytes spike while GC time also rises, when the user opens the memory view for the affected time range, the UI shows heap and GC trends plus the allocation flamegraph and top allocating Java stacks.
- AE9. **Covers R31, R32.** Given two or more Java threads are deadlocked, when the agent captures a thread snapshot, the backend stores the deadlock event and the UI shows the deadlock cycle with involved thread names, locks, and blocked stack frames.
- AE10. **Covers R31, R33.** Given request threads are repeatedly BLOCKED on the same monitor, when the user opens the slow-thread view, the UI shows the blocked threads, the blocking stack frame, and related lock wait profile data when available.
- AE11. **Covers R31, R34.** Given one worker thread is consuming CPU, when the user opens the busy-thread view for the same time range, the UI shows the busy thread, its current stack snapshots, and the matching CPU flamegraph stack when samples exist.
- AE12. **Covers R35.** Given a user asks which objects are retaining heap after a memory leak, when only allocation profiles are available, the UI explains that retained-heap ownership requires a heap dump or future retained-heap feature and does not present allocation data as retained memory.
- AE13. **Covers R36.** Given temporary profiling is enabled for 10 minutes with high-frequency thread snapshots, when the duration expires, both async-profiler and high-frequency snapshots stop automatically.

---

## Success Criteria

- Operators can enable Java profiling for a selected production workload without restarting application Pods.
- Incident responders can temporarily profile a single Java service or Pod and stop it automatically or manually.
- Users can move from a JVM trend spike to a Java flamegraph that identifies the responsible stack.
- Users can answer the first-version diagnostic questions: what is allocating memory, where a Java deadlock is, which threads are slow or blocked, and which threads or stacks are busy.
- The first version remains narrow enough to operate with an existing single-node ClickHouse and no AGPL profile backend dependency.
- Stored data is automatically cleaned, with no retained data older than 7 days.
- A downstream implementation plan does not need to invent the collection modes, storage target, UI scope, or safety boundaries.

---

## Scope Boundaries

- No Pyroscope, Parca, Grafana profile backend, or other incompatible profile server dependency.
- No general-purpose Coroot replacement.
- No log analysis, distributed tracing, service map, network tracing, TLS L7 extraction, or cloud topology.
- No non-Java language profiling in the first version.
- No OpenJ9 support in the first version.
- No distributed ClickHouse requirement in the first version.
- No full alerting product in the first version.
- No heap dump analysis, retained heap dominator tree, object reference graph, or leak suspect engine in the first version.
- No promise to reconstruct exact historical thread execution between snapshots; thread snapshots answer sampled current state, while async-profiler answers sampled CPU, allocation, and lock activity over time.
- No retention period longer than 7 days for any collected data.
- No requirement to support arbitrary profile formats beyond the Java async-profiler-derived profile types needed for the product.
- No permanent raw artifact archive unless explicitly enabled by retention configuration.

---

## Key Decisions

- Use a DaemonSet collector as the primary collection shape: node-local discovery and JVM attach are a better production fit than ad hoc cross-Pod jobs.
- Use Kubernetes annotations or labels for control: this keeps rollout GitOps-friendly and avoids requiring a custom control UI before the profiling loop works.
- Use ClickHouse as the primary profile query store: the environment already has a single-node ClickHouse deployment, and profile samples are naturally suited to columnar query patterns.
- Keep Prometheus-compatible metrics in the design: trend charts and health signals should remain easy to scrape and reason about, even if profile samples live in ClickHouse.
- Start with a service-centric Java UI: the product should solve Java performance diagnosis, not become a broad observability workbench.
- Use a self-owned flamegraph renderer in v1 unless planning proves a small permissively licensed dependency is safer and faster.
- Combine async-profiler data with JVM thread snapshots: profiles explain sampled CPU, allocation, and lock cost over a time range; snapshots explain current thread state, deadlock cycles, and blocking relationships.
- Treat retained-heap analysis as out of scope for v1: allocation profiling can identify allocation sources, but it must not be marketed as heap-retention or leak-root ownership analysis.
- Treat data cleanup as part of the product, not an operational afterthought: single-node ClickHouse must have bounded retention and visible cleanup status from the first version.

---

## Dependencies / Assumptions

- The target Kubernetes environment allows a DaemonSet with the permissions needed to inspect local processes and attach to target JVMs.
- The existing ClickHouse instance has enough retention, disk, and operational headroom for profile data in addition to logs.
- Java workloads use HotSpot-compatible JVMs such as OpenJDK, Oracle JDK, Amazon Corretto, or similar.
- async-profiler and JFR parsing remain acceptable dependencies for Java profile capture and conversion.
- JVM attach or an equivalent safe JVM management path is available for bounded thread snapshot capture.
- Prometheus or a Prometheus-compatible scraper is available or acceptable for JVM trend metrics.

---

## Outstanding Questions

### Deferred to Planning

- [Affects R1-R6][Technical] Define the exact Kubernetes annotation and label vocabulary for continuous enablement, temporary enablement, duration, and explicit disable.
- [Affects R13-R17][Technical] Decide whether ClickHouse stores only normalized profile samples or also raw profile/JFR artifacts for replay and debugging.
- [Affects R15-R17][Technical] Define the initial ClickHouse schema, retention, partitioning, and compaction strategy for single-node operation.
- [Affects R17-R20][Technical] Define exact retention windows, all at or below 7 days, for profile samples, thread snapshots, JVM metrics, deadlock events, target status, and optional raw artifacts.
- [Affects R21-R26][Technical] Define the exact flamegraph interaction model: search, compare, zoom, stack reversal, and top-table behavior.
- [Affects R30-R36][Technical] Define thread snapshot trigger behavior, default continuous snapshot frequency, temporary profiling snapshot frequency, and per-target safeguards.
- [Affects R31-R34][Technical] Decide whether thread snapshots are captured through JVM Attach diagnostic commands, JMX-style APIs, or a small in-process helper loaded only when profiling is enabled.
- [Affects R30-R35][Product] Define exact UI wording for unsupported questions, especially retained heap versus allocation source, so users do not over-interpret the data.
- [Affects R27-R29][Technical] Define agent permission requirements and the minimum health/status metrics.
