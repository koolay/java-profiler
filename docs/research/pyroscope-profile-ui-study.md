# Pyroscope Profile UI Study

## Context

This note captures the product review from the real `jdk17-http-demo` CPU profile screen and records how Grafana Pyroscope presents profile data. The goal is to make the Java profiler UI useful for real performance analysis: a service owner should quickly identify which function is expensive, whether the cost is direct or in callees, and where to drill next.

This is product guidance for our UI. It does not add Pyroscope, Grafana, Parca, or any other profiling backend as a required dependency.

## Sources

- Grafana flame graph visualization documentation: https://grafana.com/docs/grafana/latest/visualizations/panels-visualizations/visualizations/flame-graph/
- Grafana Pyroscope flame graph documentation: https://grafana.com/docs/pyroscope/latest/introduction/flamegraphs/
- Grafana Pyroscope profiling types documentation: https://grafana.com/docs/pyroscope/latest/introduction/profiling-types/
- Grafana Pyroscope Self vs Total documentation: https://grafana.com/docs/pyroscope/latest/view-and-analyze-profile-data/self-vs-total/
- Grafana Pyroscope source-code integration documentation: https://grafana.com/docs/pyroscope/latest/view-and-analyze-profile-data/line-by-line/
- Grafana Pyroscope repository overview: https://github.com/grafana/pyroscope

## Current UI Review

The latest CPU profile screen is moving in the right direction because it has a top table, a flame graph, and display modes. It is not yet good enough for real performance bottleneck analysis.

### Findings

1. `so.6` appears as the top hot-code row.

   This is the highest-severity usability bug. The top table should guide users to actionable application symbols. A native/runtime frame such as `so.6`, `lib*.so`, `[vdso]`, `pthread`, `libjvm`, or other JVM/runtime symbols should not appear as the first hot-code target. Users will mistake it for a business-code hotspot.

   Recommendation: exclude native/runtime/JDK infrastructure frames from the hot-code table. Keep them visible in the flame graph because they are still useful as stack context.

2. The table shows `Symbol` and `Self`, but `Total` is not visible in the current layout.

   Pyroscope's top table model depends on three columns: symbol, self, and total. Without both `Self` and `Total`, users cannot distinguish a function that burns CPU directly from one that is expensive because of its callees.

   Recommendation: make `Self` and `Total` always visible in all table layouts. If horizontal space is limited, shrink symbol text or stack the numeric percentage under each numeric value, but do not hide `Total`.

3. Selecting a table row currently filters the flame graph by writing the symbol into the search box.

   This loses the full stack context at the moment the user needs context most. The default action should be selection and highlighting, not filtering. Searching should be a user-initiated action.

   Recommendation: selecting a top-table row should highlight matching frames in the full flame graph and populate detail state, without reducing the graph to matching rows. A separate search action can intentionally dim non-matching frames.

4. Multiple rows show `Self = 0`.

   `Self = 0` can be valid, but when several visible business methods have zero self and `Total` is hidden or visually weak, the screen suggests there is no actionable hotspot. That is misleading.

   Recommendation: default sort should be explicit and controllable. For incident-style bottleneck discovery, defaulting to `Total` can be easier to understand, while allowing one-click sorting by `Self` for direct CPU burners.

5. The explanatory text is more prominent than the decision signal.

   The warning that stack context is not source-line call order is correct, but it should not dominate the workflow. The user needs a concise decision aid.

   Recommendation: show a compact interpretation near the selected function, for example:

   - High self: optimize this function's own work.
   - High total, low self: inspect callees in the flame graph.

## How Pyroscope Presents Profiles

### Flame Graph

Grafana describes the flame graph as a hierarchical profile view. Horizontally it represents the full profile; each block width represents that function's value. Vertically it shows stack hierarchy. The root represents the total sampled work.

Implication for this project:

- The flame graph should remain the canonical stack-context view.
- Selecting a block should show value, percentage, and sample details.
- Focusing a block should make that block the 100% root and rescale children relative to it.
- Runtime/native/JVM frames may stay in the flame graph because they explain sampled execution context.

Important reading rules:

- The graph is built by sampling stack traces, aggregating repeated stacks, and rendering the aggregate hierarchy.
- The root is the selected profile's total resource usage for the time range and label filters.
- Horizontal position is not time order. Width is the value share of that frame relative to the current root.
- Vertical position is call-stack hierarchy. Frames below a node are callees observed under that node.
- A wide frame means that frame's subtree accounts for a large share of the current profile. It does not automatically mean that frame's own function body is slow; that is what `Self` helps distinguish.
- A narrow frame can still be important if it is repeated many times, appears in a critical path, or has high self after aggregation in the top table.

Current UI risk:

- Filtering the flame graph to matching rows creates a display that looks like a real flame graph but no longer shows the original aggregate hierarchy. That breaks the user's ability to read width and vertical structure correctly.
- If a search or top-table selection changes the graph from "full aggregate hierarchy" to "grouped matches", the UI must label it as search results, not a flame graph.

Design rule:

- Preserve the full flame graph by default. Use highlight, dimming, selection, focus, and breadcrumbs to guide attention without destroying stack context.
- Only show grouped search results as a separate table/list, not as the main flame graph.

### Top Table

Grafana's flame graph panel includes a top-table mode. The table has `Symbol`, `Self`, and `Total`. It is sorted by self by default and can be reordered by total or symbol. Rows aggregate values for the same function if it appears in multiple places in the profile.

Implication for this project:

- Our table should aggregate by normalized Java symbol, not by raw line-specific frame only.
- `Self` and `Total` must both be visible and sortable.
- The table should not include non-actionable runtime/native frames by default.
- Row actions should map to search/highlight and stack-context inspection.

### Search

Grafana's search finds functions by name. Matching functions remain highlighted and non-matching functions are visually de-emphasized.

Implication for this project:

- Search should highlight/dim inside the current graph, not replace the graph with a fake one-row result set.
- Search should be opt-in. Selecting a table row should not automatically become a search filter unless the user chooses a search action.

### Focus Block

Grafana's focus action sets the selected block to 100% width and rescales its children. This supports drill-down into smaller subtrees.

Implication for this project:

- The current Back/Reset model is valid, but the action label should be "Focus" or "Focus block" rather than "Show stack context" when the result is a focused flame graph.
- A status pill or breadcrumb should make the current focus clear and removable.

### Sandwich View

Grafana's sandwich view shows aggregated context for a selected function: callers above and callees below. This is important when a function appears in multiple places.

Implication for this project:

- Our current "sampled stack context" is only one sampled path. It is useful but weaker than Pyroscope's aggregated sandwich view.
- The next higher-value interaction is not source code lookup; it is an aggregated caller/callee context for the selected Java symbol.

### Self vs Total

Pyroscope defines `Self` as resource usage directly attributed to a function, excluding sub-functions. `Total` includes the function and everything it calls. For CPU profiles, high self means the function itself consumes CPU. High total with low self means the cost is mostly in callees.

Implication for this project:

- The UI must teach this distinction through layout, not long prose.
- `Self` and `Total` should be shown together in table rows, tooltips, and selected-frame details.
- The selected function detail should include an interpretation based on the relation between self and total.

### Profiling Types

Pyroscope presents profile data by the resource or blocking dimension being analyzed, not as a single generic "performance" view. The documented profile types include CPU, memory allocation and heap views, goroutines, mutex, block, lock, and exceptions.

Implication for this project:

- Each tab needs a profile-type-specific question and metric vocabulary.
- CPU should answer "which functions consume CPU time?"
- Memory should answer "which functions allocate or retain memory?"
- Locks should answer "where are threads delayed by synchronization?"
- Deadlocks should answer "which lock cycle blocks progress?"
- Status and ingestion should explain whether the profile data is trustworthy before users interpret it.

## Profile-Type UX Model For This Project

### CPU

CPU profiling measures CPU time consumed by application code. The flame graph width indicates CPU time consumed by each function.

UI target:

- Default table columns: `Symbol`, `Self CPU`, `Total CPU`.
- Primary sort: `Total CPU` for first-pass bottleneck discovery, with `Self CPU` available for direct CPU burners.
- Detail copy: "High self means this method itself burns CPU. High total with low self means inspect callees."
- Native/JVM/runtime frames stay in the flame graph but are excluded from Hot Code by default.
- Use `Self CPU` and `Total CPU` labels in CPU tables so the columns do not look reusable across allocation or lock profiles.
- When a Java row has high total and low self, tell the user to start from that Java owner and inspect highlighted frames in the full stack; native/runtime frames explain where samples landed but are not the default optimization target.
- Use a visible frame-category legend so users can distinguish application Java, JVM/runtime, and native/system context.
- When focused, show an explicit `Focused: <frame>` status so users know the current root changed.

### Memory

Memory allocation profiling tracks allocation amount and frequency. Pyroscope separates allocation object count, allocation space, heap, in-use objects, and in-use space.

UI target:

- Memory must not be a blank secondary tab. It needs the same analysis primitives as CPU: top table, flame graph, and trend/timeline evidence when available.
- Use memory-specific columns, for example `Allocated bytes`, `Objects`, `In-use bytes`, or `Total allocated`, depending on the selected profile type.
- The UI should distinguish allocation churn from retained heap. Churn can hurt GC and CPU even when heap does not grow; retained heap points closer to leak or footprint issues.
- The empty state should say whether memory profiles are unavailable, delayed, or genuinely zero.

### Locks

Lock profiling measures synchronization contention. Pyroscope documents mutex and lock profiles using count and duration dimensions.

UI target:

- Lock views need both frequency and duration. A frequently acquired lock is not necessarily the worst bottleneck if hold or wait duration is low.
- Suggested table columns: `Lock/site`, `Wait duration`, `Wait count`, `Total blocked`.
- The flame graph should make blocked call paths visible, not just CPU execution paths.
- Detail copy should guide users toward the contended monitor or synchronized method.

### Blocking And Deadlocks

Block profiling is about where execution is paused or delayed. Deadlock analysis is a discrete graph/cycle problem rather than a normal flame graph problem.

UI target:

- Blocking should use duration/count language and show where threads wait.
- Deadlocks should prioritize cycle readability: thread, held lock, waited lock, owning thread, and stack frame evidence.
- Do not mix deadlock cycles into CPU hotspot ranking.

### Exceptions

Pyroscope lists exceptions as a supported profile type. This project does not currently scope exception profiling as a required first-version Java profiler surface.

UI target:

- Do not add an Exceptions tab unless requirements are updated.
- If added later, it should answer "which exception types and throw sites are hot?" with count and stack context, not CPU samples.

### Source Code Integration

Pyroscope's source code view is not a generic local file lookup. It depends on source metadata such as repository and git reference labels, GitHub access, and for Java an explicit `.pyroscope.yaml` mapping. The feature belongs in a function-details workflow after profile analysis is already working.

Implication for this project:

- Do not reintroduce a demo-only source snippet feature.
- If source viewing is added later, treat it as an integration requiring build metadata and repository mapping, not as bundled demo source.

## Next UI Target

The next CPU view should behave like this:

1. The default screen shows `Top Table` and `Flame Graph` together.
2. The top table lists actionable Java symbols only.
3. The CPU columns are always `Symbol`, `Self CPU`, and `Total CPU`.
4. Default sort is explicit. Prefer `Total CPU` for first-time bottleneck discovery, with one-click `Self CPU` sorting for direct CPU burners.
5. Selecting a row highlights matching frames in the full flame graph.
6. Search remains separate and visually dims non-matching frames instead of replacing context.
7. "Focus block" drills into a selected block and shows a removable focus state.
8. Add an aggregated caller/callee view later to approximate Pyroscope sandwich view.

## Acceptance Criteria For The Next Iteration

- `so.6`, `.so`, `[vdso]`, `pthread`, `libjvm`, and JDK/runtime frames do not appear in Hot Code / Top Table by default.
- `Self CPU` and `Total CPU` are both visible at desktop width in `Both` mode.
- A Top Table row click does not filter away flame graph context.
- Search highlights/dims matching frames in the existing graph.
- Top Table supports sorting by `Self CPU`, `Total CPU`, and `Symbol`.
- The selected function detail explains whether the cost is mostly direct self time or downstream total time.
- Real Playwright acceptance verifies the table contains no native frame as the first actionable row and verifies `Self CPU` and `Total CPU` are visible.
- Memory and lock tabs use profile-type-specific labels and empty states; they must not reuse CPU wording.
