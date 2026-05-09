# Pyroscope Study Alignment Polish

## Problem

The CPU profile UI now follows the basic Pyroscope-style top-table and flame-graph flow, but review against `docs/research/pyroscope-profile-ui-study.md` still shows four gaps:

1. Table columns are generic `Self` and `Total` instead of CPU-specific labels.
2. The insight explains `Self` versus `Total`, but does not tell the user what to do next.
3. Flame graph colors do not communicate frame category, so runtime/native frames still visually compete with actionable Java frames.
4. Focus mode lacks an explicit focused-frame status pill or breadcrumb.

## Scope

- Rename CPU table columns to `Self CPU` and `Total CPU`.
- Make CPU insight copy action-oriented and include the selected Java owner symbol.
- Add a flame graph legend for `Application Java`, `JVM/runtime`, and `Native/system`.
- Add frame category classes to flame graph rows.
- Add a focused-frame status pill when the flame graph root is focused.
- Update unit and real acceptance tests.

## Non-goals

- Do not add source-code lookup.
- Do not add a full sandwich view.
- Do not change ingestion, ClickHouse schema, or collector behavior.
- Do not hide runtime/native frames from the full flame graph.

## Files

- `web/src/features/cpu/hot-code-view.tsx`
- `web/src/features/cpu/hot-code-view.test.tsx`
- `web/src/visualization/flamegraph.tsx`
- `web/src/visualization/flamegraph.test.tsx`
- `web/src/styles.css`
- `web/tests/real-acceptance.spec.ts`
- `docs/research/pyroscope-profile-ui-study.md`
- `docs/index.md`

## Test Scenarios

- CPU top table renders `Self CPU` and `Total CPU`.
- Insight for low self/high total includes the selected Java symbol and next action.
- Flame graph renders the category legend.
- Application frames get application styling; native/system frames get native styling.
- Focusing a frame shows `Focused: <frame>` status.
- Real acceptance verifies `Self CPU`, `Total CPU`, action-oriented insight, legend, and focused status.

## Verification

- `npm test -- --run hot-code flamegraph target-status`
- `npm run build`
- `go test ./backend/internal/httpapi`
- Real Playwright acceptance against `http://127.0.0.1:18181` with `KUBECONFIG=$HOME/backup/localk8s.yaml`

