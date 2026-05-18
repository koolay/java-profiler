---
title: 'UI Layout Sidebar Tightening'
type: 'feature'
created: '2026-05-18'
status: 'done'
origin:
  - 'docs/brainstorms/java-profiler-requirements.md'
  - 'docs/plans/2026-05-17-001-feat-expert-java-profiling-plan.md'
---

# UI Layout Sidebar Tightening

## Summary

Tighten the profiler workbench layout so the main analysis area reads as a single, intentional workspace rather than a split shell with unused right-hand space. The pass keeps the existing Java-only profiling model intact while improving width allocation, button alignment, sidebar density, and the fidelity of the published screenshots in `README.md` and the docs site.

## Problem Frame

The current workbench presentation can make the CPU and allocation pages feel under-filled when the main profiling panel no longer needs a second inspector column. The sidebar and scope card also consume more width and height than necessary, which reduces the room available for the top table, flamegraph, and selected-frame details. The docs assets need to reflect the updated UI so the README and site screenshots match the real acceptance output.

## Scope

### In scope

- Reduce the workbench shell sidebar width and compress the `Scope` card density.
- Convert profile views that use the flamegraph workspace into a single-column main analysis layout when no separate right inspector is needed.
- Keep toolbar actions such as `Copy`, `Share`, `Reset view`, and `Hide Native` aligned and non-wrapping.
- Update GC summary presentation so it remains readable at the new layout density.
- Refresh the published screenshots under `docs/assets/screenshots/` and keep the README and docs pages pointed at those stable asset paths.

### Out of scope

- No backend schema changes.
- No new profiling data types.
- No new product areas beyond CPU, Wall Clock, I/O, GC, allocation, deadlocks, and ingestion evidence.
- No expansion into general observability, tracing, or metrics dashboards.

## Requirements Traceability

- `docs/brainstorms/java-profiler-requirements.md` requires a focused Java profiling workspace and a memory analysis view that presents allocation evidence as flamegraphs and top stacks, not as retained-heap analysis.
- `docs/plans/2026-05-17-001-feat-expert-java-profiling-plan.md` establishes the app-workbench direction, Top Table / Flamegraph expert workflow, and the split layout that can be tightened for views that no longer need a right-hand detail drawer.
- `docs/operations/real-profiling-acceptance-standard.md` requires real browser acceptance evidence, non-empty profile data, and current screenshots for UI-facing changes.

## Implementation Notes

- `web/src/features/cpu/hot-code-view.tsx` should treat the flamegraph workspace as a wide single-column area when the selected view does not need a second inspector column.
- `web/src/routes/service-overview.tsx` and `web/src/styles.css` should keep the top-level action buttons and left sidebar compact enough to preserve room for the main analysis workspace.
- `web/src/visualization/flamegraph.tsx` should preserve the selection/focus behavior while presenting the right-side inspector and action buttons without creating a fake empty column.
- `web/src/features/gc/gc-view.tsx` should keep GC summary cards readable at the narrower sidebar density.
- `web/tests/real-acceptance.spec.ts` should continue validating the real workflow and capture updated screenshots.
- `docs/index.md`, `docs/zh/index.md`, `docs/operations/performance-analysis-user-manual.md`, and the Chinese manual should point at the refreshed screenshot assets without changing the established doc structure.

## Verification

- `web/src/features/cpu/hot-code-view.test.tsx`
- `web/src/features/gc/gc-view.test.tsx`
- `web/src/visualization/flamegraph.test.tsx`
- `web/tests/real-acceptance.spec.ts`
- `npm run docs:build` in `docs/`
- Real acceptance with `KUBECONFIG=$HOME/backup/localk8s.yaml` and current workspace-built images

## Acceptance

- CPU, Wall Clock, I/O, GC, and allocation pages no longer leave a large unused right-hand panel when the main analysis view is active.
- The left sidebar and scope card feel denser without becoming cramped or dropping important labels.
- The top action buttons and flamegraph controls stay aligned on one row where possible, without awkward wrapping.
- The latest real acceptance screenshots are the ones referenced by the README and online docs.
