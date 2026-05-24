# TODOs

## Add baseline comparison for profile analysis

- **What:** Add an explicit comparison mode for abnormal profile windows versus normal baseline windows.
- **Why:** Single-window CPU analysis can identify current hotspots, but it cannot prove whether a method is newly expensive, removed, or significantly worse than normal behavior.
- **Pros:** Helps incident analysis move from "this is hot now" to "this changed during the incident"; reduces false positives from methods that are always hot.
- **Cons:** Expands backend API/query scope, adds UI state for two windows, and requires careful retention-bound query handling.
- **Context:** The first Pyroscope-style CPU UI implementation should focus on tiny frame inspectability, Top Table actionability, selected-frame diagnosis, and single-window workflow. Baseline comparison is intentionally deferred until the single-window workflow is accepted.
- **Depends on / blocked by:** Requires updated requirements and accepted single-window CPU investigation workflow. Must preserve the 7-day retention boundary and must not add Pyroscope, Parca, or Grafana as required backend dependencies.

