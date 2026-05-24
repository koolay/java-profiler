# Allocation Analysis Optimization Design

## Goal

Improve the allocation analysis workflow so an operator can answer these questions from one service view:

- Is there allocation data for the selected namespace, service, Pod, and time range?
- Which Java methods and call paths allocate the most bytes or objects?
- Which allocation hotspots are probably caused by string construction, collection resizing, database query building, thread-local cleanup, or other recurring patterns?
- Is the displayed result complete enough to trust, or did query/frame limits omit important data?
- When the question is about retained heap ownership rather than allocation rate, what evidence is missing?

The design keeps the product boundary intact: allocation profiles identify allocation sources. They are not heap dumps and must not be presented as retained heap or leak-root ownership analysis.

## Observed Problems

The current UI can show real allocation data, but the diagnostic workflow has several gaps.

1. Empty states are ambiguous. Opening the page with broad default filters can show `0 B` and `scanned_samples=0`, even when data exists for a specific namespace and time range.
2. Allocation analysis relies too heavily on the flamegraph. A flamegraph is good for path context, but it is not the fastest way to rank allocation sources.
3. The UI says "Allocation profile" without clearly distinguishing sampled allocation bytes from current live heap occupancy.
4. Partial flamegraphs are visible, but the effect of `node_limit`, omitted nodes, and scanned sample count is not actionable.
5. Long Java method names are truncated in the dense flamegraph, making it hard to read paths such as `AbstractStringBuilder.ensureCapacityInternal` or `ThreadLocalUtils.getFieldValue`.
6. The UI does not summarize recurring causes such as string building, collection growth, database query construction, thread-local cleanup, or URL/config construction.
7. The scope panel can be read as contradictory when namespace, service, and Pod controls imply different scopes.

## Non-Goals

- Do not add heap dump capture in this optimization.
- Do not add retained-size analysis in this optimization.
- Do not require Pyroscope, Parca, Grafana, or any other profile backend.
- Do not duplicate Prometheus memory, GC, or allocation-rate charts. Link or preserve time context where useful.
- Do not expand beyond Java services on Kubernetes.

## Target User Workflow

1. The user opens the Java Profiler service workbench.
2. The user selects a namespace, service or Pod, and time range.
3. The UI shows whether allocation bytes and allocation objects have data for that exact scope.
4. If data exists, the UI displays:
   - top allocating call paths,
   - top self-allocating frames,
   - grouped hotspot categories,
   - selected-frame details,
   - flamegraph context.
5. If data is missing, the UI explains whether the likely cause is no matching target, disabled allocation profiling, no ingested samples, query/storage failure, or too narrow a time range.
6. If the result is partial, the UI explains what was omitted and offers ways to continue the investigation.

## Information Architecture

The Alloc view should be organized in this order:

1. **Scope and Evidence Bar**
   - Effective query scope: namespace, service, Pod, container, JVM if known, and time range.
   - Data freshness: newest profile end time and age.
   - Sample coverage: scanned samples, returned nodes, omitted nodes, and partial flag.
   - Profile modes available for this scope: allocation bytes and allocation objects.

2. **Allocation Insights**
   - A short generated summary derived from deterministic rules.
   - Example: "String construction accounts for 35.8 MiB across `StringBuilder.append`, `StringBuilder.toString`, and `String.<init>`. The hottest business path is `MultiQueryBuilder.build -> BusinessDataServiceHelper.load`."
   - Each insight links to matching table rows and flamegraph focus.

3. **Top Allocating Paths**
   - Ranked by total allocated bytes or allocated objects.
   - Each row shows total value, percent of returned profile, leaf frame, shortest representative call path, category, and sample count when available.
   - This table answers "where is most allocation coming from?"

4. **Top Self Allocating Frames**
   - Ranked by self allocated bytes or objects.
   - This table answers "which frame directly allocates?"
   - It should include JDK frames such as `Arrays.copyOf`, `Arrays.copyOfRange`, `String.<init>`, `HashMap.resize`, and business frames when available.

5. **Flame Graph**
   - Keeps the current interaction model: search, focus, back, reset, native filtering, selected-frame details.
   - Adds stronger hover/click detail for full method names and full call paths.

6. **Selected Frame Details**
   - Full frame name.
   - Package/class/method/line parsed from the frame.
   - Total and self allocation values.
   - Percent of returned profile.
   - Children and parent path.
   - Category tag.
   - Copy full path and copy frame actions.

## Backend API Changes

### Existing Endpoints To Preserve

- `GET /api/ui/v1/flamegraph`
- `GET /api/ui/v1/top-stacks`
- `GET /api/ui/v1/service-selectors`

Existing response shapes should remain backward compatible.

The new allocation workflow should reuse the existing query concepts instead of building a parallel profiling stack. The current backend already has:

- `GET /api/ui/v1/flamegraph` for stack-tree rendering and partial metadata.
- `GET /api/ui/v1/top-stacks` for self and total frame ranking.
- `GET /api/ui/v1/service-summary` for target-level profile coverage.
- `GET /api/ui/v1/target-status` and ingestion health APIs for missing-data evidence.

`allocation-summary` should compose these concerns behind one allocation-specific contract where that improves UX, while keeping the core aggregation logic testable in the app/domain layer.

### Add Allocation Summary Endpoint

Add:

```text
GET /api/ui/v1/allocation-summary
```

Required query parameters:

- `namespace`
- `profile_type`
- `start`
- `end`

Optional query parameters:

- `service`
- `pod`
- `container`
- `jvm`
- `path_limit`
- `self_frame_limit`
- `node_limit`

Scope semantics:

- Missing optional scope parameters mean "all values allowed within the authorized namespace."
- Empty-string optional scope parameters are normalized the same way as missing parameters.
- The literal string `all` is accepted from the UI for readability, but normalized to missing before query execution.
- The response must include both `requested_scope` and `effective_scope` so users can see what the backend actually queried.
- `namespace` is required for the summary endpoint. Namespace-only summary is allowed only for bounded short windows and lower limits.

`profile_type` must support:

- `java_allocation_bytes`
- `java_allocation_objects`

Response:

```json
{
  "schema_version": 1,
  "requested_scope": {
    "namespace": "kd-cosmic-xk",
    "service": "all",
    "pod": "all",
    "container": "all",
    "jvm": "all",
    "start": "2026-05-24T12:00:00Z",
    "end": "2026-05-24T12:30:00Z"
  },
  "effective_scope": {
    "namespace": "kd-cosmic-xk",
    "service": "",
    "pod": "",
    "container": "",
    "jvm": "",
    "start": "2026-05-24T12:00:00Z",
    "end": "2026-05-24T12:30:00Z"
  },
  "coverage": {
    "has_data": true,
    "profile_type": "java_allocation_bytes",
    "total_value": 81054924,
    "value_unit": "bytes",
    "scanned_samples": 184,
    "returned_paths": 50,
    "returned_self_frames": 50,
    "omitted_paths_lower_bound": 1,
    "omitted_nodes_lower_bound": 62,
    "partial": true,
    "partial_reasons": ["path_limit", "node_limit"],
    "newest_profile_end": "2026-05-24T12:29:58Z"
  },
  "top_paths": [
    {
      "rank": 1,
      "leaf_frame": "java/lang/StringBuilder.append:136",
      "total_value": 17406361,
      "self_value": 0,
      "percent": 21.5,
      "category": "string_construction",
      "sample_count": 17,
      "path": [
        "java/util/Arrays.copyOf:3332",
        "java/lang/AbstractStringBuilder.ensureCapacityInternal:124",
        "java/lang/AbstractStringBuilder.append:448",
        "java/lang/StringBuilder.append:136"
      ]
    }
  ],
  "top_self_frames": [
    {
      "rank": 1,
      "frame": "java/util/Arrays.copyOf:3332",
      "self_value": 19713216,
      "percent": 24.3,
      "category": "array_copy"
    }
  ],
  "insights": [
    {
      "severity": "info",
      "category": "string_construction",
      "message_code": "allocation.string_construction.dominant",
      "evidence_frame": "java/lang/StringBuilder.append:136",
      "evidence_value": 17406361
    }
  ],
  "limitations": [
    {
      "code": "partial_result",
      "message_code": "allocation.partial.path_or_node_limit"
    }
  ]
}
```

The backend returns raw values, units, stable category codes, and stable message codes. The UI owns localized copy and display-unit formatting. This keeps the API useful for Chinese and English docs without baking English prose into the contract.

### Authorization and Guardrails

The endpoint must be registered under `RequireUIAuth`, the same UI authentication boundary as the existing flamegraph and top-stacks endpoints. Authorization must be enforced before query execution; the UI selector list is not an authorization boundary.

Input constraints:

- `profile_type` is an enum and only accepts `java_allocation_bytes` or `java_allocation_objects`.
- `start` and `end` must parse as RFC3339 or Unix seconds and must produce `start < end`.
- Query range must not exceed the 7-day retention boundary. Default UI windows should stay at 30 minutes or less.
- Namespace-only summary must use a smaller default window and lower limits than service or Pod scope.
- `path_limit` default 50, hard maximum 200.
- `self_frame_limit` default 50, hard maximum 200.
- `node_limit` default 500 for summary details, hard maximum 2000.
- Stack depth considered by the summary should be capped at the ingested stack depth, with 128 as the expected operational ceiling.
- Backend requests should use context cancellation, a query timeout, bounded response size, and metrics for request count, latency, returned rows, partial count, and errors.

Compatibility:

- Add `schema_version` to the response.
- UI should treat a missing `allocation-summary` endpoint as a soft capability miss and fall back to the existing flamegraph/top-stacks path.
- Unknown category or limitation codes must render as generic allocation evidence, not fail the page.

### Empty-State Classification

The backend should classify empty allocation responses into one of these states:

- `no_matching_target`: no target exists for the selected Kubernetes scope.
- `profiling_disabled`: target exists but allocation profiling is not enabled.
- `no_samples_in_range`: profiling is enabled but no allocation samples match the selected time range.
- `ingestion_gap`: collector reported uploads or drops that explain missing data.
- `query_error`: storage query failed.
- `unsupported_runtime`: target JVM is not HotSpot-compatible or cannot support the requested profile type.

The UI should not infer these states from `root.value=0` alone.

Classification data sources and precedence:

1. `query_error`: storage or backend query failed.
2. `unsupported_runtime`: target status says the runtime cannot support the requested profile type.
3. `profiling_disabled`: target status says the target exists but profiling is disabled by Kubernetes metadata or expired temporary profiling.
4. `ingestion_gap`: ingestion health or target status reports dropped, retryable, rejected, or truncated profile batches in the requested window.
5. `no_matching_target`: service selectors and target status have no target matching the effective scope.
6. `no_samples_in_range`: a matching target exists and profiling can run, but profile samples are absent for the selected range.

If the evidence conflicts, prefer the earliest state in this list. That makes operational failures visible before generic empty results.

## ClickHouse Query Design

The summary endpoint can be implemented on top of the normalized profile sample table already used for flamegraphs, but it must stay bounded. Do not build the summary by joining every raw sample to every stack frame and then `arrayJoin`ing the full stack. That creates `samples * stack_depth` intermediate rows and can overload a single-node ClickHouse installation.

Recommended bounded query pipeline:

```text
HTTP request
  |
  v
validate + normalize scope
  |
  v
coverage query on profile_samples only
  |  sum(sample_value), count(), max(ended_at)
  |
  +--> if empty: classify empty state using target status + ingestion evidence
  |
  v
top stack_id query on profile_samples
  |  GROUP BY stack_id
  |  ORDER BY sum(sample_value) DESC
  |  LIMIT path_limit + 1
  |
  v
join only returned stack_ids to profile_stacks
  |
  v
application aggregation
  |  top_paths, top_self_frames, categories, insights
  |
  v
bounded response with partial metadata
```

Required query outputs:

- total sample value for the selected profile type and scope,
- top stack paths by total value,
- top frames by self value,
- newest profile end timestamp,
- sample count or row count,
- omitted-node metadata when the result is truncated.

The query layer should preserve the current bounded-retention model. No new data type may exceed the 7-day retention limit.

Aggregation rules:

- `coverage.total_value` is `sum(sample_value)` over every sample matching the effective scope, profile type, and time range.
- `top_paths` are ranked by `sum(sample_value)` grouped by `stack_id` first, then resolved to the full stack frames for only the returned `stack_id` set.
- `top_paths.path_key` is the exact ordered stack frame list after backend stack normalization and depth cap. Do not merge two different paths only because they have the same leaf frame.
- `top_paths.percent` uses `coverage.total_value` as the denominator, not only returned paths.
- `top_self_frames` means leaf-frame allocation unless future ingestion adds an explicit self-frame field. Do not use `arrayJoin(frames)` for self frames because that computes inclusive contribution.
- `top_self_frames` may be derived from the returned top stack set for v1. If full global self-frame ranking is required later, add a separate bounded query with its own limit and partial reason.
- Category matching runs in Go/TypeScript over returned top rows. Do not group by category in ClickHouse.
- `omitted_paths_lower_bound` is based on `LIMIT N+1`, not an expensive exact full count.
- `omitted_nodes_lower_bound` can reuse the flamegraph builder's bounded node accounting for returned paths. It is acceptable to report a lower bound rather than exact omitted nodes.

Partial reasons:

- `path_limit`: more path rows existed than `path_limit`.
- `self_frame_limit`: more self-frame rows existed than `self_frame_limit`.
- `node_limit`: returned paths could not all be represented inside the response node budget.
- `sample_limit`: a repository-level sample cap was hit before all matching samples were processed.
- `depth_limit`: one or more stacks exceeded the configured max depth and were truncated.
- `timeout`: query context deadline expired and a partial response could be returned.
- `memory_limit`: ClickHouse or backend memory guard stopped the query.
- `missing_stack`: profile samples referenced stack ids whose frames were unavailable.

Caching:

- Add a short backend cache only after correctness tests pass.
- Cache key must include auth tenant or equivalent authorization scope, effective scope, profile type, start, end, limits, and schema version.
- TTL should be 15 to 60 seconds.
- Cache successful and partial-success responses. Do not cache authorization failures.
- Negative-cache query errors for at most 1 to 5 seconds to prevent refresh storms.

## Hotspot Categorization Rules

Categorization must be deterministic and explainable. It should not require an LLM at query time.

Initial rules:

- `string_construction`
  - Frames containing `java/lang/StringBuilder`, `java/lang/AbstractStringBuilder`, `java/lang/String.<init>`, `java/lang/StringConcatFactory`, or `java/util/Formatter`.
- `array_copy`
  - Frames containing `java/util/Arrays.copyOf`, `java/util/Arrays.copyOfRange`, or `java/lang/System.arraycopy`.
- `collection_growth`
  - Frames containing `java/util/HashMap.resize`, `java/util/ArrayList.grow`, `java/util/Arrays.copyOf` under common collection frames.
- `thread_local_cleanup`
  - Frames containing `ThreadLocal`, `ThreadLocalUtils`, or known project thread lifecycle cleanup packages.
- `database_query_building`
  - Frames containing `DB.query`, `executeSqlBuilder`, `MultiQueryBuilder`, `BaseDB`, `Repository`, or `BusinessDataReader`.
- `url_or_config_building`
  - Frames containing `URIBuilder`, `URLCreator`, `DBConfig`, or connection URL construction helpers.
- `serialization_or_json`
  - Frames containing JSON, Jackson, Gson, protobuf, or serialization package names.
- `native_or_runtime`
  - Native/system frames and JVM runtime frames.
- `application_other`
  - Application Java frames not matched above.

The summary should show category confidence as rule-based, not probabilistic.

## UI Behavior Details

### Effective Scope Display

Replace the compact ambiguous scope text with an explicit query sentence:

```text
namespace=kd-cosmic-xk, service=all, pod=all, profile=allocation bytes, range=30m
```

If the UI resolved a single Pod from suggestions, show:

```text
namespace=kd-cosmic-xk, service=all, pod=<pod-name>, profile=allocation bytes, range=30m
```

### Empty State

For `no_samples_in_range`:

```text
No allocation samples matched this scope and time range.
Try a wider time range or choose one of the namespaces, services, or Pods with recent allocation data.
```

For `profiling_disabled`:

```text
This target is visible, but allocation profiling is not enabled for it.
Enable allocation profiling through the Kubernetes profiling annotation or temporary incident profile control.
```

For retained heap questions:

```text
Allocation profiles show where objects were allocated during the selected window. They do not show which objects are still retained on the heap.
```

### Partial Result State

When `partial=true`, show:

- returned total,
- scanned samples,
- returned paths and returned self frames,
- omitted lower-bound counts,
- all partial reasons,
- impact statement.

Example:

```text
Partial result: 184 samples scanned, at least 62 nodes omitted because the path and node budgets were reached. Rankings are reliable for returned paths, but smaller branches may be missing.
```

Actions:

- "Show grouped view"
- "Focus selected branch"
- "Download returned JSON"
- "Increase frame budget" only when the deployment allows this through bounded configuration

### Method Name Readability

Every frame label should support:

- full method name on hover,
- click-to-select,
- copy full frame,
- copy full path,
- package-shortening toggle.

The flamegraph can still truncate labels visually, but details must never require guessing.

## Acceptance Criteria

- Given namespace `kd-cosmic-xk` and a 30-minute window with allocation samples, the Alloc view shows non-zero coverage, top allocating paths, top self frames, insights, and flamegraph.
- Given allocation data whose flamegraph is partial due to `node_limit`, the UI shows scanned sample count, omitted node count, reason, and a plain-language trust warning.
- Given summary data exceeds `path_limit`, the response includes `partial=true`, `path_limit` in `partial_reasons`, and `omitted_paths_lower_bound >= 1`.
- Given summary data exceeds `self_frame_limit`, the response includes `partial=true`, `self_frame_limit` in `partial_reasons`, and the UI explains that smaller self frames may be missing.
- Given default filters with no data, the UI does not imply allocation profiling is globally disabled unless backend evidence says it is disabled.
- Given a hotspot path through `StringBuilder.append`, the UI categorizes it as `string_construction` and links the insight to the table row and flamegraph branch.
- Given a hotspot path through `ThreadLocalUtils.doClearThreadLocals`, the UI categorizes it as `thread_local_cleanup`.
- Given a user asks about current heap occupancy, the Alloc view states that sampled allocation profiles cannot answer retained heap ownership.
- Given the result is not partial, no partial warning is shown.
- Given the result is partial, the warning remains visible above both tables and flamegraph.
- Given the same endpoint is queried with two different limits or time windows, cached responses must not cross-contaminate.
- Given a namespace-only query requests a range larger than the allowed window, the backend rejects or downgrades the request with a clear limitation code before ClickHouse scans broadly.

## Test Plan

### Test Placement Matrix

| Capability | Test layer | Test location |
| --- | --- | --- |
| `GET /api/ui/v1/allocation-summary` route, auth wrapper, parameter validation, JSON shape | Backend HTTP unit | `backend/internal/httpapi/query_handlers_test.go` |
| Summary aggregation: coverage, top paths, top self frames, insights, limitations | Backend app unit | `backend/internal/app/query_allocation_summary_test.go` |
| Deterministic hotspot categories | Backend domain unit | `backend/internal/domain/allocation_categorizer_test.go` |
| In-memory repository summary behavior | Backend repository unit | `backend/internal/clickhouse/profile_repository_test.go` |
| SQL query construction and bounded limits | Backend repository unit or integration where available | `backend/internal/clickhouse/*_test.go` |
| Empty-state classification | Backend app unit and UI unit | `backend/internal/app/query_allocation_summary_test.go`, `web/src/features/allocation/allocation-view.test.tsx` |
| Complete, partial, and empty UI states | Web component unit | `web/src/features/allocation/allocation-view.test.tsx` |
| Insight links, selected frame details, copy full path | Web component plus Playwright smoke | `web/src/features/allocation/allocation-view.test.tsx`, `web/tests/profiling-flow.spec.ts` |
| Real non-empty summary and UI workflow | Real acceptance | `scripts/real-acceptance.sh`, `web/tests/real-acceptance.spec.ts` |

### Unit Tests

- Backend categorization rules for representative Java frames.
- Summary response construction from profile samples and stack ids.
- Empty-state classification for `no_matching_target`, `profiling_disabled`, `no_samples_in_range`, `ingestion_gap`, `query_error`, and `unsupported_runtime`.
- Percent formatting from raw values, with display strings generated in the UI.
- Parameter normalization for missing, empty-string, and `all` optional scope values.
- Guardrail validation for time range, `profile_type`, and limit caps.
- Partial metadata for `path_limit`, `self_frame_limit`, `node_limit`, `sample_limit`, `timeout`, and `missing_stack`.
- Cache key isolation for scope, profile type, time range, limits, authorization scope, and schema version if caching is implemented.
- UI rendering for complete, partial, and empty allocation states.

### Browser Tests

- Select namespace, time range, and Alloc profile.
- Verify top allocating paths table.
- Verify top self frames table.
- Verify partial warning content.
- Verify selected-frame details show full frame and full path.
- Verify category insight links focus the matching row or flamegraph branch.
- Verify retained-heap limitation copy appears when allocation evidence is shown.
- Verify a mocked 404 or capability miss for `allocation-summary` falls back to existing flamegraph/top-stacks behavior.

### Real Acceptance

Follow `docs/operations/real-profiling-acceptance-standard.md` for any implementation touching collector profiling, ingestion, ClickHouse storage, backend query APIs, Kubernetes deployment, or the profile UI.

The real acceptance run must prove:

- non-empty allocation profile for the selected workload,
- non-empty allocation summary response,
- saved `backend-allocation-summary.json` containing `coverage.has_data=true`, `coverage.total_value>0`, non-empty `top_paths`, non-empty `top_self_frames`, and non-empty `insights`,
- correct empty-state behavior for a controlled scope with no samples,
- correct partial-result behavior by using a test-only low limit or fixture, not by hoping production data naturally reaches `node_limit`,
- summary latency and ClickHouse resource usage stay below the acceptance thresholds defined for the deployment,
- no target workload restart increase.

Real acceptance should keep E2E scope tight:

- Unit tests prove deterministic logic.
- HTTP tests prove contract and errors.
- Playwright mocked tests prove user interactions.
- Real acceptance proves current workspace images, Kubernetes target, ClickHouse rows, non-empty summary response, and browser workflow with real data.

### Coverage Diagram

```text
CODE PATH COVERAGE
==================
[+] backend/internal/httpapi/query_handlers.go
    |
    +-- AllocationSummary()
        +-- [GAP] auth wrapper uses RequireUIAuth
        +-- [GAP] validates profile_type bytes/objects only
        +-- [GAP] normalizes missing/empty/all scope
        +-- [GAP] rejects invalid time range and excessive limits
        +-- [GAP] maps query errors to clear HTTP status and limitation code

[+] backend/internal/app/query_allocation_summary.go
    |
    +-- QueryAllocationSummary()
        +-- [GAP] coverage query has data
        +-- [GAP] no data classified from target status and ingestion evidence
        +-- [GAP] top paths grouped by stack_id, not leaf-only
        +-- [GAP] top self frames use leaf frame semantics
        +-- [GAP] partial reasons emitted for each limit
        +-- [GAP] categories and insights generated from returned rows

[+] backend/internal/domain/allocation_categorizer.go
    |
    +-- CategorizeAllocationPath()
        +-- [GAP] string_construction
        +-- [GAP] array_copy
        +-- [GAP] collection_growth
        +-- [GAP] thread_local_cleanup
        +-- [GAP] database_query_building
        +-- [GAP] url_or_config_building
        +-- [GAP] application_other fallback

[+] web/src/features/allocation/allocation-view.tsx
    |
    +-- AllocationView()
        +-- [GAP] complete state
        +-- [GAP] partial warning state
        +-- [GAP] empty classified state
        +-- [GAP] capability fallback state
        +-- [GAP] insight click focuses table/flamegraph
        +-- [GAP] full-path details and copy actions

USER FLOW COVERAGE
==================
[+] Real allocation investigation
    |
    +-- [GAP] [->E2E] select namespace + 30m + Alloc
    +-- [GAP] [->E2E] read top paths and top self frames
    +-- [GAP] [->E2E] click insight and inspect selected frame
    +-- [GAP] [->E2E] see partial warning when limits are reached
    +-- [GAP] [->E2E] see clear empty state for controlled no-sample scope

COVERAGE TARGET
===============
New deterministic backend logic: 100% branch coverage.
User-facing allocation workflow: mocked Playwright plus real acceptance.
```

### Failure Modes

| Codepath | Production failure | Test required | User-visible behavior |
| --- | --- | --- | --- |
| Parameter normalization | `all` is treated as literal service name | HTTP unit | Clear invalid/normalized scope display |
| Coverage query | ClickHouse timeout | App/HTTP unit | Partial or query error with retry guidance |
| Top paths query | Full stack join explodes memory | Repository/performance test | Query rejected, timed out, or partial with limitation code |
| Top self frames | Inclusive frames mislabeled as self | App unit | Correct leaf/self definition in table |
| Empty-state classification | Disabled profiling shown as no samples | App and UI unit | Specific disabled-profiling message |
| Partial warning | `path_limit` hidden from UI | UI unit and Playwright | Plain-language partial warning |
| Cache | Results leak across namespace or limits | Unit/integration if cache exists | No cross-scope data exposure |
| Real acceptance | Fixture never triggers partial | Scripted low-limit path | Stable partial proof |

## Implementation Plan

1. Add backend summary model, scope normalization, guardrail validation, and deterministic categorizer.
2. Add bounded summary repository methods: coverage query, top stack-id query, and stack lookup for returned ids.
3. Add app-layer aggregation for top paths, leaf/self frames, partial reasons, empty-state classification, and insights.
4. Add `GET /api/ui/v1/allocation-summary` under `RequireUIAuth`.
5. Add UI API client type and capability fallback to existing flamegraph/top-stacks behavior.
6. Add Alloc view evidence bar and explicit effective scope display.
7. Add top allocating paths and top self frames tables.
8. Add Allocation Insights summary, partial warning, and classified empty states.
9. Improve selected-frame details and copy-full-path behavior.
10. Add unit, browser, and real acceptance coverage before marking the design complete.

## Worktree Parallelization Strategy

| Step | Modules touched | Depends on |
| --- | --- | --- |
| Backend summary domain and app aggregation | `backend/internal/domain`, `backend/internal/app` | none |
| ClickHouse bounded repository queries | `backend/internal/clickhouse` | summary model shape |
| HTTP route and contract | `backend/internal/httpapi` | app aggregation |
| Web API types and Alloc UI | `web/src/api`, `web/src/features`, `web/src/routes` | response contract |
| Browser and real acceptance | `web/tests`, `scripts` | HTTP route and UI |

Parallel lanes:

- Lane A: backend domain/app aggregation -> HTTP route.
- Lane B: ClickHouse repository queries.
- Lane C: Web API/UI starts after response contract is stable.
- Lane D: acceptance scripts and Playwright tests start after route and UI are available.

Launch A and B in parallel if the response types are agreed first. Run C after the contract lands. Run D last. Avoid parallel edits inside `web/src/routes/service-overview.tsx`.

## NOT in Scope

- Heap dump capture: retained heap ownership is a separate feature and has different safety/storage risks.
- Retained-size analysis: allocation profiles cannot prove live object ownership.
- New profile backend: ClickHouse remains the profile query store.
- Prometheus chart storage or dashboards: existing Prometheus-series services own metric trends.
- General observability expansion: this remains Java profiling for Kubernetes.
- Exact omitted-node counts for all possible omitted branches: lower-bound counts are enough to keep queries bounded.

## Risks

- Categorization can overfit to known package names. Keep rules transparent and allow `application_other`.
- Larger summaries can increase query cost. Use bounded stack-id-first queries, limits, timeouts, and partial metadata.
- Users may still read allocation bytes as retained memory. Repeat the sampled-allocation limitation in empty states, details, and documentation.
- Partial result warnings can become noisy. Show them only when backend metadata says returned nodes are incomplete.
- Summary cache can leak or stale data if keyed too broadly. Include authorization scope, effective scope, time range, limits, and schema version.

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | - | - |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | - | - |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 3 | CLEAR | 6 issues found, 0 critical gaps, incorporated into this plan |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | - | - |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | - | - |

**UNRESOLVED:** 0

**VERDICT:** ENG CLEARED - ready to implement.
