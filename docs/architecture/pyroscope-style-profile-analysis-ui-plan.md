# Pyroscope-Style Profile Analysis UI Implementation Plan

## Goal

Make the CPU profile workflow read like a Pyroscope-style investigation surface:

- Top Table identifies actionable Java methods.
- Flame Graph preserves full stack context.
- Small flamegraph blocks remain inspectable even when their labels cannot fit.
- Frame details explain `Self CPU` versus `Total CPU`.
- The UI guides a user from a suspicious time window to a concrete performance hypothesis.

This plan keeps Pyroscope, Parca, Grafana, and other profile backends out of the required architecture. Pyroscope is only a UX reference.

## Current Problems

1. Tiny flamegraph frames can appear as blank blocks.
   - Current behavior hides labels for frames narrower than 7%.
   - This is acceptable visually, but the UI needs stronger hover, click, and tooltip affordances so blank blocks do not look like missing code.

2. Native/runtime frames compete with application code.
   - Native frames are useful flamegraph context.
   - They should not dominate the Java Top Table or look like the default optimization target.

3. Selection and search need clearer separation.
   - Selecting a table row should highlight matching frames while preserving the full flamegraph.
   - Search may dim non-matching frames, but it should not turn the graph into a filtered result list.

4. CPU interpretation is technically correct but not explicit enough.
   - `Total CPU` is subtree cost.
   - `Self CPU` is direct cost in that frame.
   - The selected-frame panel should turn that distinction into a short diagnosis.

5. Time-window performance analysis needs a first-class workflow.
   - The first implementation can guide users through one selected window.
   - Baseline comparison should be added after the single-window workflow is solid.

## Design Rules

- Keep `Both` as the primary CPU investigation mode.
- Keep native/JVM/system frames in the flamegraph.
- Exclude native/JVM/system frames from the default hot-code Top Table.
- Do not force labels into tiny flamegraph blocks.
- Every frame must be inspectable through hover, focus, click, or keyboard focus.
- A selected Java row highlights flamegraph matches without changing stack layout.
- Focus changes the current flamegraph root and must show a visible focused state.
- Percentages are relative to the returned profile or current focused root, not wall-clock time order.

## Existing Flow To Reuse

```text
ClickHouse profile rows
        |
        v
backend/internal/app/query_flamegraph.go
        |
        v
Flamegraph API root tree  -----------------------------+
        |                                               |
        v                                               v
web/src/visualization/flamegraph.tsx        backend/internal/app/query_top_stacks.go
        |                                               |
        |                                               v
        |                                  Top Stack API rows, preferred path
        |                                               |
        +------------------+----------------------------+
                           |
                           v
             web/src/features/cpu/hot-code-view.tsx
                           |
                           v
             Top Table + Flame Graph in Both mode
```

`hot-code-view.tsx` already has a flamegraph-derived Java hot-code fallback for development, tests, and backend-empty states. The implementation should reuse that fallback instead of introducing a third classifier.

## UI State Model

```text
Top Table selection
        |
        v
selectedFrameName ---------------> highlightQuery
        |                              |
        |                              v
        |                    Flamegraph highlights matching frames
        |
        +---- does not write search input

Search input
        |
        v
searchQuery ---------------------> Top Table filtering
        |
        +-------------------------> Flamegraph search match/dim state

Flamegraph click
        |
        v
focus.path ----------------------> current flamegraph root changes
        |
        +-------------------------> Back / Reset become available

Hover or keyboard focus
        |
        v
inspectedFrame ------------------> Inspector detail changes only
```

State separation is the core architecture rule: selection highlights, search filters/dims, focus changes root, and hover/focus inspects. Do not route all four behaviors through the same search string.

## Implementation Units

### Unit 1: Make Tiny Frames Inspectable

Files:

- `web/src/visualization/flamegraph.tsx`
- `web/src/visualization/flamegraph.test.tsx`
- `web/src/styles.css`

Changes:

- Add a small shared frame-detail formatter inside `flamegraph.tsx`.
  - Use it for frame `aria-label`, frame `title`, and inspector/detail values.
  - Keep the helper local to the flamegraph component unless another file needs it.
  - The formatter should include full frame name, category, total value, self value, total percentage, and self percentage.
- Add a `title` attribute to every flamegraph frame button with:
  - full frame name
  - category
  - total value
  - self value
  - total percentage
  - self percentage
- Keep `.flame-row-tiny` label hiding, but make selected and hovered tiny frames visually obvious.
- Ensure keyboard focus on a tiny frame updates the inspector.
- Add test coverage for a tiny frame that has no visible label but still exposes full details through accessible name, title, and inspector.

Acceptance:

- Narrow frames may be visually blank.
- A user can hover, click, or keyboard-focus a narrow frame and see the full symbol.
- Screen-reader accessible names still include the full frame name and value.

### Unit 2: Strengthen Selected Frame Details

Files:

- `web/src/visualization/flamegraph.tsx`
- `web/src/visualization/flamegraph.test.tsx`
- `web/src/styles.css`

Changes:

- Extend the inspector/detail panel with a compact interpretation line:
  - high self: optimize this frame's own work
  - high total with low self: inspect callees
  - native/runtime frame: find the owning Java caller before treating it as an optimization target
- Keep the wording profile-type aware through existing labels such as `Self CPU` and `Total CPU`.
- Avoid adding long explanatory copy to the main page body.

Acceptance:

- Selecting an application frame with high self shows direct-work guidance.
- Selecting a frame with low self and high total points the user to callees.
- Selecting a native/runtime frame tells the user to inspect the Java caller.

### Unit 3: Verify Top Table Actionability

Files:

- `web/src/features/cpu/hot-code-view.tsx`
- `web/src/features/cpu/hot-code-view.test.tsx`
- `backend/internal/app/query_top_stacks.go`
- `backend/internal/app/query_top_stacks_test.go`

Changes:

- Treat the existing backend Top Stack classifier as the primary implementation when `topRows` are available.
- Treat the existing frontend flamegraph-derived hot-code classifier as a fallback for backend-empty, test, and development states.
- Verify both classifiers agree on representative frames:
  - native/system: `[vdso]`, `lib*.so`, `pthread`, `clock_gettime`, async-profiler internals
  - JVM/runtime: `java.lang.Thread`, JVM runtime helpers, JDK infrastructure when not application-owned
  - application Java: service/package frames and accepted Java source frames
- Only change classifier code when a concrete mismatch is found.
- Keep native/runtime frames in the flamegraph API response.
- Preserve `Self CPU` and `Total CPU` columns in all Top Table modes.

Acceptance:

- A native frame such as `libasyncProfiler.so.StackWalker::walkVM` can appear in the flamegraph.
- The same frame does not appear as the primary actionable row in the CPU Top Table.
- CPU Top Table tests cover native/runtime exclusion, application Java retention, and backend/fallback consistency for the representative frame set.

### Unit 4: Separate Selection, Highlighting, Search, and Focus

Files:

- `web/src/features/cpu/hot-code-view.tsx`
- `web/src/features/cpu/hot-code-view.test.tsx`
- `web/src/visualization/flamegraph.tsx`
- `web/src/visualization/flamegraph.test.tsx`

Changes:

- Treat table selection as `selectedFrameName`.
- Treat search input as explicit user search only.
- Pass selected symbol into flamegraph as highlight state without replacing search text.
- Use dimming only for explicit search.
- Keep focus as a separate click/drilldown action on the flamegraph.

Acceptance:

- Selecting a Top Table row highlights matching frames.
- The search input does not change just because a table row was selected.
- The full flamegraph stack layout remains visible after table selection.
- Explicit search still dims non-matching frames.

### Unit 5: Add Single-Window Analysis Summary

Files:

- `web/src/features/cpu/hot-code-view.tsx`
- `web/src/features/cpu/hot-code-view.test.tsx`
- `web/src/styles.css`
- `docs/operations/performance-analysis-user-manual.md`

Changes:

- Add a compact selected-row summary near the table/flamegraph boundary:
  - selected symbol
  - `Self CPU`
  - `Total CPU`
  - diagnosis based on self/total ratio and frame category
- Keep it operational, not tutorial-like.
- Update the operations manual with the intended CPU investigation workflow.

Acceptance:

- With no selection, the summary points to the top ranked Java method.
- With selection, the summary follows the selected Java method.
- The manual tells users to start from target status, then Top Table, then Flame Graph, then focus/search.

### Unit 6: Add Baseline Comparison Later

Files:

- `web/src/features/cpu/hot-code-view.tsx`
- `web/src/api/types.ts`
- `web/src/api/client.ts`
- `backend/internal/app/query_top_stacks.go`
- `backend/internal/app/query_flamegraph.go`
- `docs/brainstorms/java-profiler-requirements.md`

Scope:

- Do not include this in the first implementation unless the single-window workflow is already accepted.
- Add an explicit comparison mode only after requirements are updated.

Future behavior:

- Compare abnormal window with a previous normal window.
- Show added, removed, and increased Java methods.
- Keep both windows bounded by the existing retention limit.
- Do not introduce a required external observability backend.

## Test Plan

Frontend unit tests:

- `npm run test -- --run web/src/visualization/flamegraph.test.tsx`
- `npm run test -- --run web/src/features/cpu/hot-code-view.test.tsx`

Required frontend unit coverage:

- `flamegraph.test.tsx`
  - Tiny frame width below the label threshold keeps the visible block compact but exposes full detail through accessible name and `title`.
  - Keyboard focus on a tiny frame updates the inspector.
  - Hovering a tiny frame updates the inspector and returns to the selected frame after hover expires.
  - Selected or hovered tiny frames have an obvious visual affordance.
  - The shared frame-detail formatter drives `aria-label`, `title`, and inspector/detail values consistently.
  - Search highlights matching frames without removing non-matching stack context.
- `hot-code-view.test.tsx`
  - Selecting a Top Table row highlights matching flamegraph frames without rewriting the search input.
  - The selected-row analysis summary follows the selected Java row.
  - Search no-match state is clear and recoverable.
  - Backend rows and flamegraph fallback rows classify the representative native/runtime/application frame set consistently.
  - Native/runtime-only profiles keep the flamegraph visible and explain that no application Java frames were found.
  - Same method or class name from different packages remains distinct.

Frontend build:

- `npm run build`

Backend unit tests:

- `go test ./backend/internal/app ./backend/internal/domain`

Required backend unit coverage:

- `query_top_stacks_test.go`
  - `Self` and `Total` remain correct when native leaf frames own self time.
  - Runtime/native frames are excluded from Top Stack rows.
  - Application Java rows are retained under runtime/native stack context.
  - Representative async-profiler, libc, pthread, JVM, JDK, and application frames are classified as expected.
  - Same symbol names from different packages remain separate rows.

Browser verification:

- Start the Vite dev server.
- Verify the CPU view in `Both` mode.
- Verify tiny frame inspectability, Top Table selection, search dimming, focus, Back, and Reset.

Required browser coverage:

- CPU investigation path: selected time window -> `Both` mode -> Top Table Java row -> flamegraph highlight -> selected detail -> focus -> Back -> Reset.
- Tiny frame path: hover, click, and keyboard focus a narrow frame and verify full details are visible.
- Search path: matching query highlights/dims correctly; no-match query is recoverable.
- Native-only path: Top Table empty explanation appears while flamegraph remains inspectable.

Real profiling acceptance:

- Required if implementation touches profile query behavior, backend ingestion semantics, deployment, or accepted UI workflow.
- Follow `docs/operations/real-profiling-acceptance-standard.md`.
- Use `export KUBECONFIG=$HOME/backup/localk8s.yaml`.
- Completion requires non-empty CPU, allocation, and lock-delay profile evidence, ClickHouse rows, bounded TTL, UI workflow acceptance, and no target restart increase.

## Recommended Build Order

1. Unit 1: tiny frame inspectability.
2. Unit 2: selected-frame diagnosis.
3. Unit 4: selection/search/focus separation.
4. Unit 3: Top Table actionability verification and gap fixes.
5. Unit 5: single-window analysis summary and manual update.
6. Unit 6: baseline comparison only after requirements are updated.

## NOT in Scope

- Required Pyroscope, Parca, Grafana, or other profile backend dependencies.
  - Rationale: this project keeps ClickHouse as the primary profile query store.
- Baseline comparison in the first implementation.
  - Rationale: it changes API and product scope; complete the single-window workflow first.
- General observability, tracing, log analysis, dashboards, alerting, or metric retention.
  - Rationale: the product boundary stays focused on Java profiling and bounded profile retention.
- Non-Java profiling.
  - Rationale: the first version remains focused on HotSpot-compatible Java services on Kubernetes.
- A third frontend/backend frame classifier.
  - Rationale: backend Top Stack classification and frontend fallback classification already exist; verify and tighten them instead of duplicating rules.

## What Already Exists

- `web/src/visualization/flamegraph.tsx`
  - Existing: frame layout, category styling, hover/focus inspector, selected frame state, search, highlight, focus, Back, Reset, and tiny-frame hiding.
  - Reuse: add shared frame detail formatting and stronger tiny-frame inspectability without replacing the component.
- `web/src/features/cpu/hot-code-view.tsx`
  - Existing: `Both` view, Top Table, `Self CPU`, `Total CPU`, Java frame extraction, table selection, search, and flamegraph highlighting.
  - Reuse: preserve the table/flamegraph split and tighten selection/search separation.
- `backend/internal/app/query_top_stacks.go`
  - Existing: Top Stack ranking, `Self`/`Total` separation, native/runtime exclusions, display values, and profile semantics.
  - Reuse: treat this as the preferred production source when backend `topRows` are available.
- `web/src/features/cpu/hot-code-view.tsx` fallback extraction
  - Existing: flamegraph-derived hot Java frame extraction for backend-empty, tests, and development states.
  - Reuse: keep it as fallback and verify it agrees with backend classification for representative frames.

## Failure Modes

| Flow | Production failure | Test coverage required | Expected user behavior |
|------|--------------------|------------------------|------------------------|
| Tiny frame inspection | Frame label is hidden and no detail is discoverable | Unit + browser | Hover, click, or keyboard focus shows full frame detail |
| Shared frame formatter | `aria-label`, `title`, and inspector show different values | Unit | All detail surfaces show the same frame name, values, and percentages |
| Top Table selection | Selection rewrites search and destroys stack context | Unit + browser | Selection highlights matching frames while search text stays unchanged |
| Explicit search | No-match search leaves user with an apparently broken view | Unit + browser | UI shows a recoverable empty/no-match state |
| Focus | Clicking a frame changes root but user cannot return | Existing + browser | Focus state, Back, and Reset are visible and functional |
| Backend Top Stack classification | Native/runtime frame becomes the top actionable row | Backend unit | Top Table keeps application Java rows as the optimization target |
| Frontend fallback classification | Backend-empty state shows different hot rows from backend path | Frontend unit | Fallback agrees with backend representative frame classification |
| Native-only profile | Top Table is empty and flamegraph disappears | Unit + browser | Explanation appears and flamegraph remains inspectable |
| Same class names | Same method/class from different packages merges incorrectly | Existing + required unit | Rows remain package-distinct |

Critical silent gaps after this plan: none, provided the required unit and browser tests are implemented.

## Parallelization Strategy

| Step | Modules touched | Depends on |
|------|-----------------|------------|
| Tiny frame inspectability and selected-frame diagnosis | `web/src/visualization`, `web/src/styles.css` | — |
| Hot Code selection/search summary | `web/src/features/cpu`, `web/src/styles.css` | Tiny frame detail formatter if summary reuses frame detail text |
| Top Table classifier verification | `backend/internal/app`, `web/src/features/cpu` | — |
| Operations manual update | `docs/operations` | Final UI behavior |

Parallel lanes:

- Lane A: tiny frame inspectability -> selected-frame diagnosis.
- Lane B: Top Table classifier verification.
- Lane C: operations manual update after Lane A and Lane B settle.

Conflict flags:

- Lane A and the Hot Code summary both touch `web/src/styles.css`; coordinate or keep them sequential.
- Lane B touches `web/src/features/cpu` if frontend fallback tests change; avoid running it in parallel with Hot Code summary unless the edits are isolated.

## Risks

- Native/runtime classification can accidentally hide useful Java infrastructure from the Top Table.
  - Mitigation: keep the filter narrow and test representative async-profiler, libc, pthread, JVM, JDK, and application frames.
- Too much explanatory UI can make the analysis surface noisy.
  - Mitigation: keep diagnosis copy short and tied to the selected frame only.
- Baseline comparison can expand backend API scope.
  - Mitigation: defer it until the single-window workflow is accepted and requirements explicitly call for comparison.

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | — |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | ISSUES RESOLVED IN PLAN | 4 findings addressed, 0 critical gaps |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | — | — |

**UNRESOLVED:** 0

**VERDICT:** ENG REVIEW COMPLETED — plan scope reduced and test coverage requirements added. Design review is recommended before implementation because this change affects user-facing profile analysis UI.
