---
date: 2026-05-09
type: fix
origin: docs/brainstorms/java-profiler-requirements.md
status: active
---

# Fix Real Acceptance UI Evidence

## Problem Frame

The isolated async-profiler acceptance run in `/tmp/java-profiler-ap-real-20260509-021744` produced real Java profiling data for `async-profiler-lab`: CPU, allocation, lock, thread snapshot, deadlock, ingestion, and target-status data are present, and stack frames include Java code such as `BusyApp.lambda$main$0:9`, `BusyApp.lambda$main$0:14`, and `java/lang/Thread.run`. The functional chain is proven for the isolated workload.

The evidence screenshots show that the product experience is not yet acceptable:

- `ui-02-cpu.png` renders the flamegraph as a very tall diagonal list of small buttons. Labels overlap visually, the page height is extreme, and a real Java/native stack is hard to inspect.
- `ui-01-status.png` truncates Pod names and user actions at a normal desktop width. The table technically contains the right states, but it is not scannable.
- `ui-04-ingestion.png` is readable enough, but it should remain stable after layout changes.
- The recent mservice check also exposed an acceptance-script correctness issue: full-profiling acceptance must not pass from stale data outside the current run window.

This plan fixes the UI and acceptance evidence quality using the existing real artifacts as the design target. It does not attempt to claim current `mservice` profiling success while that workload is explicitly disabled.

## Scope Boundaries

In scope:

- Replace the current vertical flamegraph row list with a bounded, scannable flamegraph rendering suitable for deep Java/native stacks.
- Improve status table layout so Pod, PID, state, reason, message, and user action remain readable on a 1280px-wide screenshot.
- Preserve search, zoom/reset, partial-result warnings, accessibility labels, and existing backend contracts.
- Add or update frontend tests for the new flamegraph/status rendering behavior.
- Keep the real acceptance script guarded against stale data by requiring current-window `accepted` status and current-window ClickHouse counts.
- Re-run local verification and, where possible, use the existing `/tmp/java-profiler-ap-real-20260509-021744` payloads as fixture-level evidence for UI behavior.

Out of scope:

- Re-running `mservice` profiling while it remains disabled by metadata.
- Adding Pyroscope, Grafana, Parca, or another profile backend.
- Adding general observability, tracing, logs, Prometheus metric charts, or non-Java profiling.
- Changing async-profiler collection semantics unless a UI fixture exposes a data-shape bug.
- Reworking the whole visual design system.

## Requirements Trace

- R21, R23: the UI remains a service-centric Java profiling console, not a general workspace.
- R22, R29: target status must explain why data exists or does not exist without unreadable truncation.
- R24, R25, R26, R30, R33, R34: flamegraphs must be usable enough to connect CPU, allocation, and lock symptoms to Java stack frames.
- R37, R38: ingestion and status surfaces remain part of the diagnostic path and should not imply success when data is stale.
- AE5, AE6, AE8, AE10, AE11, AE15: the evidence flow should prove queryable profiles, status, ingestion, and diagnostic data for the selected workload and time range.

## Evidence Inputs

- `/tmp/java-profiler-ap-real-20260509-021744/summary.md`
- `/tmp/java-profiler-ap-real-20260509-021744/backend-flamegraph-cpu.json`
- `/tmp/java-profiler-ap-real-20260509-021744/backend-flamegraph-alloc-bytes.json`
- `/tmp/java-profiler-ap-real-20260509-021744/backend-flamegraph-lock-delay.json`
- `/tmp/java-profiler-ap-real-20260509-021744/backend-target-status.json`
- `/tmp/java-profiler-ap-real-20260509-021744/ui-01-status.png`
- `/tmp/java-profiler-ap-real-20260509-021744/ui-02-cpu.png`
- `/tmp/java-profiler-ap-real-20260509-021744/ui-03-deadlocks.png`
- `/tmp/java-profiler-ap-real-20260509-021744/ui-04-ingestion.png`
- `/tmp/java-profiler-ap-real-20260509-021744/playwright-output/real-acceptance-real-clust-f6280-lock-and-ingestion-surfaces/video.webm`

## Existing Patterns

- Flamegraph component: `web/src/visualization/flamegraph.tsx`
- Flamegraph tests: `web/src/visualization/flamegraph.test.tsx`
- Status view: `web/src/features/status/target-status-view.tsx`
- Status tests: `web/src/features/status/target-status-view.test.tsx`
- Shared styling: `web/src/styles.css`
- Real browser acceptance: `web/tests/real-acceptance.spec.ts`
- Real acceptance script: `scripts/real-acceptance.sh`

## Key Technical Decisions

- Render flamegraph levels as fixed-height rows with absolute-positioned frames rather than one DOM row per flattened stack item. This bounds page height by stack depth, not by total frame count, and matches how users expect flamegraphs to read.
- Compute each child frame's width and x-offset from sibling sample totals. Preserve full names in `title` while showing ellipsized labels in-frame.
- Keep controls simple: search highlights matching frames and reset returns to root. Zoom remains click-to-zoom on a frame.
- Use CSS table layout changes for status rather than changing API shape: widen the diagnostic panel, use a responsive grid that gives the main panel more space, and wrap message/action text while truncating only identifiers.
- Treat acceptance as current-run evidence. Historical profile rows may remain useful for manual investigation but cannot satisfy `--require-full-profiling`.

## Implementation Units

### U1. Flamegraph Rendering

**Files**

- `web/src/visualization/flamegraph.tsx`
- `web/src/visualization/flamegraph.test.tsx`
- `web/src/styles.css`

**Approach**

- Convert the renderer from a flattened diagonal list to row-based flamegraph levels.
- Each visual frame should carry `title="<frame>: <value>"`, a stable accessible name, and a bounded label area.
- Limit the rendered height with a scrollable flamegraph viewport for very deep stacks.
- Keep search and reset behavior accessible through the existing labels used by tests.

**Test Scenarios**

- Renders root and child frames from a nested flamegraph tree.
- Long labels are present for accessibility/title even when visually truncated.
- Search highlights or filters matching frames without making the layout unbounded.
- Clicking a child zooms the view and Reset returns to root.

### U2. Status Table Readability

**Files**

- `web/src/features/status/target-status-view.tsx`
- `web/src/features/status/target-status-view.test.tsx`
- `web/src/styles.css`

**Approach**

- Make the service layout allocate more width to the diagnosis panel.
- Keep Pod names single-line with tooltip, but ensure enough width for Kubernetes identifiers.
- Let message and user-action cells wrap naturally.
- Preserve existing empty/error status messages.

**Test Scenarios**

- Accepted, unsupported, and disabled statuses remain visible.
- Long Pod names retain a `title` attribute.
- User action text is present in the DOM and not collapsed into an inaccessible abbreviation.

### U3. Acceptance Freshness Guard

**Files**

- `scripts/real-acceptance.sh`
- `docs/operations/java-profiling-runbook.md`

**Approach**

- Ensure the script records acceptance start time.
- Query UI APIs and ClickHouse counts from the current run window by default.
- Require at least one `accepted` target status in the current window when `--require-full-profiling` is used.
- Document that historical data does not satisfy real acceptance.

**Test Scenarios**

- With a disabled target, `--require-full-profiling` fails before claiming profile success.
- With current-window profile data, the script can still pass.
- The summary records the acceptance start time used for queries.

### U4. Evidence Review and Verification

**Files**

- `web/tests/real-acceptance.spec.ts`
- `README.md`

**Approach**

- Keep Playwright assertions focused on real surfaces: status, CPU flamegraph, deadlocks, ingestion.
- If practical, add a fixture or test helper using the archived CPU flamegraph payload to protect against the tall-list regression.
- Update README only if command behavior or evidence expectations changed.

**Test Scenarios**

- `cd web && npm test -- --run`
- `cd web && npm run build`
- `go test ./...`
- `helm lint ./deploy/helm --values deploy/helm/values.yaml`
- `bash -n scripts/real-acceptance.sh`
- Disabled `mservice` proof remains a failing acceptance with no stale-data pass.

## Sequencing

1. Implement U1 flamegraph rendering and tests.
2. Implement U2 status table layout and tests.
3. Verify U3 freshness guard is present and document it if needed.
4. Run local verification.
5. Run review/autofix, browser pipeline where available, commit, push, and update/open PR.

## Assumptions

- The archived isolated acceptance data is valid evidence for `async-profiler-lab`; it is the correct fixture for UI polish.
- Current `mservice` remains disabled by metadata and should not be represented as a successful profiling run.
- The existing self-owned UI remains the right target for first-version profiling investigation.
