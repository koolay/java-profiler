# Pyroscope Flame Graph UX Fix

## Problem

The current CPU profile UI has the right major pieces (`Top Table`, `Flame Graph`, `Both`) but still fails the real bottleneck-analysis workflow. A user trying to find slow Java code can be led to native/runtime symbols such as `so.6`, and selecting a top-table row currently turns the flame graph into grouped search results instead of preserving the sampled stack hierarchy.

Origin notes:

- `docs/research/pyroscope-profile-ui-study.md`
- User review screenshot showing `so.6` as the top table row and the flame graph filtered to native matches.

## Scope

- Fix the CPU hot-code table so it lists actionable Java frames instead of native/runtime frames.
- Keep `Self` and `Total` visible in `Both` mode.
- Preserve the full flame graph when a top-table row is selected.
- Change search behavior to highlight/dim matching frames in the existing hierarchy instead of replacing it with grouped result rows.
- Keep Focus/Back/Reset semantics for explicit flame graph drill-down.
- Update unit and real browser acceptance coverage.

## Non-goals

- Do not add source-code lookup.
- Do not add Pyroscope, Grafana, Parca, or any other required backend dependency.
- Do not change collector ingestion, ClickHouse schema, or profile payload contracts.
- Do not implement full Pyroscope sandwich view in this iteration.

## Files

- `web/src/features/cpu/hot-code-view.tsx`
- `web/src/features/cpu/hot-code-view.test.tsx`
- `web/src/visualization/flamegraph.tsx`
- `web/src/visualization/flamegraph.test.tsx`
- `web/src/styles.css`
- `web/tests/real-acceptance.spec.ts`
- `docs/research/pyroscope-profile-ui-study.md`

## Design Decisions

### Hot Code Filtering

The top table is for actionable Java symbols. It should exclude non-actionable runtime/native frames by default:

- `.so`, `lib*.so`, `[vdso]`
- `pthread`, `clock_gettime`, `read`, `write`, `open`, `close`
- `java.*`, `javax.*`, `jdk.*`, `sun.*`, `com.sun.*`
- frames that do not parse as Java class + method

These frames remain visible in the flame graph because they explain stack context.

### Self And Total

The table should always show `Symbol`, `Self`, and `Total`. Layout must reserve enough width for both numeric columns in `Both` mode. Default sorting should prioritize `Total` for first-pass bottleneck discovery while allowing `Self` sorting for direct CPU burners.

### Selection Versus Search

Top-table row selection is not search. It should:

- mark the selected row,
- highlight matching flame graph frames,
- update selected symbol details,
- preserve the full flame graph hierarchy.

Search remains a separate user action. It should highlight matches and dim non-matches inside the current flame graph. It must not rebuild the graph as grouped search rows.

### Focus

Explicit focus should change the current flame graph root and enable Back/Reset. Label it as `Focus selected` or `Focus block`, not `Show stack context`.

## Test Scenarios

### `web/src/features/cpu/hot-code-view.test.tsx`

- Aggregates Java method rows by symbol and computes `Self` and `Total`.
- Excludes `so.6`, `.so`, `pthread`, JDK/runtime, and no-class native frames from the top table.
- Renders `Symbol`, `Self`, and `Total` in `Both` mode.
- Selecting a top-table row does not write into the search input.
- Default selected row is an actionable Java method, not a native/runtime frame.
- Sorting by `Self` and `Total` changes row order predictably.

### `web/src/visualization/flamegraph.test.tsx`

- Search highlights matching frames and keeps non-matching context visible but dimmed.
- Selected-highlight input highlights matching frames without changing the search value.
- Focus selected changes the current root; Back restores the previous root.
- Empty profile still renders an empty state.

### `web/tests/real-acceptance.spec.ts`

- Against `java-profiler-qa/jdk17-http-demo`, verify the first top-table symbol is not `so.6` or another native/runtime frame.
- Verify `Self` and `Total` headers are visible.
- Select `DemoHttpService.burnCpu` and verify the flame graph still contains `root` and non-selected context.
- Verify manual search keeps the flame graph region visible and highlights the target frame.

## Verification

- `npm test -- --run hot-code flamegraph target-status`
- `npm run build`
- `go test ./backend/internal/httpapi`
- Real Playwright acceptance against `http://127.0.0.1:18181` using `KUBECONFIG=$HOME/backup/localk8s.yaml`

