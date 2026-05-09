# Flame Graph Actionable Context UX

## Problem

The CPU page now protects the Top Table from native/runtime pollution, but the flame graph still makes real bottleneck analysis harder than it should be. Default selection can make the graph look filtered or muted, native/runtime frames can become the selected detail even when the user is working from Java hot code, and `Self = 0 / Total > 0` Java rows lack an explanation.

## Scope

- Make table selection highlight Java frames without dimming the rest of the flame graph.
- Keep dimming only for explicit text search.
- Prefer the selected Top Table Java symbol as the selected flame graph detail.
- Add a CPU insight panel for `Self` versus `Total`.
- Add a runtime/native context note when runtime/native frames dominate or are selected.
- Improve flame graph readability by making application matches stand out without making runtime context unreadable.
- Update unit and real acceptance tests.

## Non-goals

- Do not add source-code lookup.
- Do not add a full Pyroscope sandwich view.
- Do not change profile ingestion, ClickHouse schema, or collector behavior.
- Do not hide runtime/native frames from the full flame graph.

## Files

- `web/src/features/cpu/hot-code-view.tsx`
- `web/src/features/cpu/hot-code-view.test.tsx`
- `web/src/visualization/flamegraph.tsx`
- `web/src/visualization/flamegraph.test.tsx`
- `web/src/styles.css`
- `web/tests/real-acceptance.spec.ts`
- `docs/research/pyroscope-profile-ui-study.md`

## Design

### Selection Versus Search

Selection and search must be visually distinct:

- `highlightQuery` from the Top Table highlights matching frames only.
- Manual search highlights matches and dims non-matches.
- The search input remains empty until the user types.

### Selected Detail

When a Top Table row is selected, the detail panel should prefer the first flame graph frame matching that Java symbol. Native/runtime frames may still be selected when the user clicks them directly.

### Insight Copy

For Java rows:

- High self and high total: "This method directly consumes CPU."
- Low self and high total: "CPU is observed under this Java method, mostly in callees or runtime/native frames."

For native/runtime selected frames:

- "Runtime/native frame. Use it as stack context; optimize the nearest actionable Java frame unless lock/wait profiling confirms contention."

### Readability

Do not make the full graph low-contrast just because a table row is selected. Only search should dim non-matching frames. Application matches can use border/highlight while keeping the rest of the graph readable.

## Test Scenarios

### Unit

- Table selection leaves the search input empty.
- Table selection highlights matching flame graph frames and does not dim non-matching frames.
- Search dims non-matching frames.
- Detail panel prefers the highlighted Java frame.
- CPU insight explains low self/high total.
- Native/runtime selected frame shows native-context guidance.

### Real Acceptance

- Top Table first row is Java, not native/runtime.
- `Self` and `Total` remain visible.
- Selecting `DemoHttpService.burnCpu` keeps `root` visible and search empty.
- Non-selected runtime frames are still readable after table selection.
- Manual search dims non-matches and keeps root visible.

## Verification

- `npm test -- --run hot-code flamegraph target-status`
- `npm run build`
- `go test ./backend/internal/httpapi`
- Real Playwright acceptance against `http://127.0.0.1:18181` with `KUBECONFIG=$HOME/backup/localk8s.yaml`

