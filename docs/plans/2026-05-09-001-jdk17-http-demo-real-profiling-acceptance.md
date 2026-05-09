---
date: 2026-05-09
type: validation
origin: docs/brainstorms/java-profiler-requirements.md
status: active
---

# Mservice Real Profiling Acceptance

## Problem Frame

The async-profiler integration now works against an isolated Java acceptance workload: the collector packages `asprof` and `libasyncProfiler.so`, attaches through host PID and `/proc`, captures CPU, allocation, and lock JFR output, parses it into ClickHouse, and the UI can show non-empty profile, thread, deadlock, target-status, and ingestion surfaces. The latest isolated evidence is under `/tmp/java-profiler-ap-real-20260509-021744`.

The remaining production-shaped proof is to run the same target filter against the real workload `java-profiler-qa/jdk17-http-demo` with a short, bounded collection window and durable evidence. This has elevated risk because earlier direct attach attempts caused a JVM crash. The work must preserve the new Coroot-style "write only if changed" async-profiler asset behavior, avoid broadening the product beyond Java-on-Kubernetes profiling, and capture enough before/after state to prove whether the attach path is safe for `jdk17-http-demo`.

The screenshots from the isolated run also showed two product-experience problems that should be fixed before or during this acceptance pass: the flamegraph rows are too tall/dense for real stack labels, and the target-status table has column width and truncation issues.

## Scope Boundaries

In scope:

- Parameterize or harden the real acceptance path so it can target an existing workload in namespace `java-profiler-qa` with service/workload filter `jdk17-http-demo`.
- Keep profiling bounded to a short CPU/allocation/lock window, using existing temporary profiling metadata and chart target filters.
- Capture pre-run and post-run Kubernetes/JVM safety evidence, including Pod status, restart counts, collector/backend logs, ClickHouse counts, UI screenshots, and Playwright video.
- Improve flamegraph and target-status UI readability for real stack and Pod names without changing backend contracts.
- Reconcile any affected runbook or user-manual wording.

Out of scope:

- New profiling modes, non-Java profiling, non-HotSpot support, or retained-heap analysis.
- Replacing async-profiler, ClickHouse, or the self-owned UI.
- Prometheus dashboard storage, alerting, tracing, logs, or broad observability features.
- Long-running continuous profiling of `jdk17-http-demo`.
- Directly editing or restarting `jdk17-http-demo` unless the existing acceptance workflow already requires temporary profiling metadata and the operator context permits it.

## Requirements Trace

- R1, R2, R3, R5, R6, R27, R28: target `jdk17-http-demo` through explicit Kubernetes metadata/filtering and keep the run bounded.
- R7, R8, R9, R11, R12: validate node-local HotSpot attach, profiler conflict handling, and status reporting on the real workload.
- R13, R14, R15, R16, R17, R20: prove profile samples/stacks and related diagnostics persist into ClickHouse with bounded retention.
- R21, R22, R23, R24, R25, R26, R30, R32, R33, R34, R35: make the UI surfaces usable enough to inspect the captured Java stacks, target status, deadlocks, and ingestion health.
- R37, R38: preserve authenticated backend access and visible ingestion/drop status while collecting evidence.
- AE2, AE3, AE5, AE8, AE9, AE10, AE11, AE13, AE15: carry forward the isolated acceptance coverage and repeat the key profile/status UI proof against `jdk17-http-demo` where safe.

## Existing Patterns

- Real acceptance orchestration lives in `scripts/real-acceptance.sh` and `web/tests/real-acceptance.spec.ts`.
- Helm target filters are set through `deploy/helm/values.yaml` and `profiling.targetNamespace` / `profiling.targetService`.
- Async-profiler execution lives under `collector/internal/profiler` and is called from `collector/runtime/runtime.go`.
- Upload sizing and batching live in `collector/internal/pipeline/profile_batcher.go`, `collector/internal/pipeline/backend_client.go`, and `backend/internal/httpapi/server.go`.
- Flamegraph rendering lives in `web/src/visualization/flamegraph.tsx` with styling in `web/src/styles.css`.
- Target status rendering lives in `web/src/features/status/target-status-view.tsx` with table styling in `web/src/styles.css`.
- Operational docs to reconcile, if behavior changes, are `docs/operations/java-profiling-runbook.md`, `docs/operations/performance-analysis-user-manual.md`, and `docs/operations/deployment-operations-admin-manual.md`.

## Key Technical Decisions

- Use the existing acceptance harness instead of creating a second ad hoc script. The harness already records ClickHouse counts, API payloads, screenshots, and video, and it keeps the evidence format consistent with the isolated run.
- Treat `jdk17-http-demo` acceptance as a validation mode for an existing workload, not an install mode for a synthetic BusyApp. The script should be able to skip workload creation and still apply/check profiler deployment filters.
- Keep the profiler collection interval short and explicit for this run. The validation goal is proof of real stacks and attach safety, not a long performance study.
- Before attaching, record baseline Pod status and restart counts; after profiling, record the same state and fail or clearly flag any restart increase for `jdk17-http-demo`.
- Improve UI readability with layout/styling changes only: compact flamegraph row height, better text truncation/tooltips, stable row widths, and status-table columns that preserve important identifiers.

## Implementation Units

### U1. Existing Workload Acceptance Mode

**Files**

- `scripts/real-acceptance.sh`
- `deploy/helm/values.yaml`
- `docs/operations/java-profiling-runbook.md`

**Approach**

- Add or verify a mode that targets an existing Kubernetes workload without creating the BusyApp deployment.
- Make namespace, profiler namespace, release, service name, collector interval, and artifact directory explicit in the summary.
- Ensure Helm target filters can be set to `java-profiler-qa/jdk17-http-demo` without accidentally profiling unrelated Java Pods.
- Record the exact metadata/filter state used for the run.

**Test Scenarios**

- Running the script against an already deployed workload does not apply the synthetic BusyApp manifest.
- The generated summary shows namespace `java-profiler-qa`, service `jdk17-http-demo`, profiler namespace, release, and artifact directory.
- The collector target-status query returns rows only for the selected namespace/service when filters are supplied.

### U2. Mservice Safety Evidence

**Files**

- `scripts/real-acceptance.sh`
- `docs/operations/performance-analysis-user-manual.md`

**Approach**

- Capture pre-run `kubectl get pods -o wide`, deployment state, restart counts, and relevant collector/backend logs before the profiling window.
- Capture the same data after the profiling window and compare restart counts for `jdk17-http-demo`.
- Preserve the existing non-empty profile/thread/deadlock/ingestion assertions, but add a clear safety failure if `jdk17-http-demo` restarts during the run.
- Keep the evidence directory layout consistent with `/tmp/java-profiler-ap-real-20260509-021744`.

**Test Scenarios**

- A successful run writes before/after Kubernetes state and restart-count files.
- If restart count increases for the target workload, the script fails or emits a hard safety finding in `summary.md`.
- If profile samples, stacks, thread snapshots, and ingestion rows are non-empty, the summary reports the counts for `jdk17-http-demo`.

### U3. Flamegraph Readability

**Files**

- `web/src/visualization/flamegraph.tsx`
- `web/src/visualization/flamegraph.test.tsx`
- `web/src/styles.css`

**Approach**

- Reduce vertical density and stabilize row sizing so real Java/native frame labels remain readable.
- Truncate long frame names inside rows while preserving the full frame name in `title`.
- Keep zoom/reset/search behavior intact.
- Avoid introducing a third-party renderer for this pass.

**Test Scenarios**

- Flamegraph still renders root and child frames and still shows partial-result warnings.
- Long frame names do not overflow their row container.
- Search and reset controls remain accessible by label/text used in Playwright tests.

### U4. Target Status Table Readability

**Files**

- `web/src/features/status/target-status-view.tsx`
- `web/src/features/status/target-status-view.test.tsx`
- `web/src/styles.css`

**Approach**

- Adjust table widths so Pod, reason, message, and user-action cells remain scannable with real Kubernetes identifiers.
- Use truncation with tooltips for long identifiers and wrapping for message/action text.
- Preserve explicit backend-unavailable and no-target empty states.

**Test Scenarios**

- Status reasons such as `accepted`, `disabled_by_metadata`, `profiler_conflict`, and `attach_failed` remain visible.
- Long Pod names are truncated visually but available in `title`.
- The table remains horizontally scrollable on narrow viewports without overlapping text.

### U5. Acceptance Documentation and Evidence Pointers

**Files**

- `docs/operations/java-profiling-runbook.md`
- `docs/operations/performance-analysis-user-manual.md`
- `README.md`

**Approach**

- Document the jdk17-http-demo validation command and the expected evidence artifacts only if the script or behavior changes.
- Keep documentation honest about production safety: short window first, watch restarts, and preserve opt-in controls.
- Do not turn the docs into a general observability guide.

**Test Scenarios**

- Docs mention bounded profiling and restart-count checks when describing real workload acceptance.
- Docs still state that metric storage/dashboards remain outside this product.

## Sequencing

1. Update the acceptance harness for existing-workload targeting and restart-count evidence.
2. Compact flamegraph and target-status UI rendering.
3. Run unit/build/lint checks locally.
4. Run real acceptance against the isolated workload if script behavior changed materially.
5. Run the short-window `java-profiler-qa/jdk17-http-demo` acceptance, saving screenshots/video and summary under a timestamped `/tmp/java-profiler-jdk17-demo-*` directory.
6. Reconcile docs and evidence pointers.

## Verification

- `go test ./...`
- `helm lint ./deploy/helm --values deploy/helm/values.yaml`
- `cd web && npm test -- --run`
- `cd web && npm run build`
- Existing-workload dry check against the current cluster state, without creating BusyApp.
- Short-window real acceptance against `java-profiler-qa/jdk17-http-demo` with:
  - non-empty target status for `java-profiler-qa/jdk17-http-demo`
  - non-empty CPU, allocation, and lock profile evidence when the workload is active enough
  - collector/backend ingestion rows
  - before/after restart counts showing no new `jdk17-http-demo` restart
  - Playwright screenshots for status, CPU flamegraph, deadlocks, and ingestion
  - Playwright video saved under the run artifact directory

## Assumptions

- The current Kubernetes context is the intended cluster for `java-profiler-qa/jdk17-http-demo` validation.
- The operator context allows applying or using temporary profiling metadata/filters for `jdk17-http-demo`.
- `jdk17-http-demo` is HotSpot-compatible and active enough during the short window to produce meaningful CPU/allocation/lock samples.
- The existing dirty working tree contains the already-run async-profiler slice and should be preserved rather than reverted.
