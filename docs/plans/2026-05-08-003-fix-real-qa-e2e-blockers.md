---
date: 2026-05-08
type: fix
origin: docs/operations/performance-analysis-user-manual.md
status: active
---

# Fix Real QA E2E Blockers

## Problem Frame

Real Kubernetes acceptance against a local OrbStack cluster exposed that the current implementation does not satisfy the performance-analysis user manual. The deployment can render manifests and start some pods, but the backend crashes before serving APIs, the Web UI cannot reach backend APIs from the deployed static server, collector discovery is not tied to Kubernetes profiling metadata, and the UI cannot distinguish target or ingestion failure states as the manual requires.

This plan fixes the smallest end-to-end slice needed for truthful acceptance:

- backend starts against real ClickHouse and applies schema
- Web deployment routes `/api/*` to backend instead of returning static-server `404`
- collector evaluates real Pod metadata before reporting accepted targets
- collector uploads target status and ingestion-relevant batches to backend
- backend persists/query target status through ClickHouse
- UI shows actionable no-data/backend/target-state distinctions

## Scope Boundaries

In scope:

- ClickHouse schema compatibility for real ClickHouse.
- Helm/Web runtime routing for backend APIs.
- Target status ingestion and query backed by SQL storage.
- Collector Pod metadata policy evaluation sufficient for annotated Pod/workload-template acceptance.
- UI status/no-data states aligned with the user manual.
- Real cluster/browser validation with screenshots/video when possible.

Out of scope for this fix slice:

- Full async-profiler attach/JFR lifecycle.
- Full thread snapshot ingestion persistence.
- Real flamegraph samples from async-profiler.
- Prometheus dashboard integration.
- Production-grade RBAC design beyond existing collector watch permissions and documented authorization requirements.

## Requirements Trace

- R1. Backend must start with real ClickHouse and create tables with TTL at or below 7 days.
- R2. Deployed Web UI must route API requests to backend service or be configurable without rebuild.
- R3. Collector must not mark every HotSpot process accepted merely because it is HotSpot; it must apply profiling metadata policy.
- R4. Target status must be uploaded, persisted, and queryable by namespace/service so the `status` view can explain missing data.
- R5. UI must distinguish backend unavailable, no matching target, disabled-by-metadata, unsupported JVM, profiler conflict, attach/upload/storage errors, and retention/no-data states where backend data allows.
- R6. Existing local unit/build/browser tests must keep passing.

## Existing Patterns

- Backend query/ingest handlers live under `backend/internal/httpapi`.
- Profile ingestion uses `backend/internal/app/ingest_profile_batch.go`, `backend/internal/clickhouse/sql_repository.go`, and contract types in `contracts/profiling`.
- Collector runtime currently scans `/host/proc` and exposes metrics from `collector/runtime/runtime.go`.
- Policy evaluation already exists in `collector/internal/policy/policy.go`.
- Web API client and views live under `web/src/api`, `web/src/features`, and `web/src/routes`.
- Helm chart resources live under `deploy/helm/templates`.

## Key Technical Decisions

- Use ClickHouse-compatible TTL column types rather than relying on permissive embedded or mocked stores.
- Add target-status ingestion as a first-class backend route instead of only in-memory status storage.
- Keep Web static assets simple, but ship an nginx config that proxies `/api/` to backend in-cluster.
- Treat collector's Pod metadata mapping as best-effort for this slice: resolve process container identity where available, then evaluate policy before accepted/disabled status.
- Make UI empty/error states explicit and user-actionable without pretending profile data exists.

## Implementation Units

### U1. ClickHouse Schema and SQL Target Status

**Files**

- `backend/internal/clickhouse/001_initial_profile_schema.sql`
- `db/clickhouse/001_initial_profile_schema.sql`
- `backend/internal/clickhouse/sql_repository.go`
- `backend/internal/clickhouse/status_repository.go`
- `backend/internal/clickhouse/retention_repository_test.go`
- `backend/internal/clickhouse/profile_repository_test.go`

**Approach**

- Change TTL helper columns to a ClickHouse-supported `DateTime` or TTL expression shape.
- Add SQL insert/query support for target status.
- Keep in-memory repository for unit tests where appropriate, but ensure SQL repository is covered by real schema validation tests where available.

**Test Scenarios**

- Real ClickHouse schema application succeeds.
- Target status rows insert and query by namespace/service.
- TTL remains seven days or less.

### U2. Backend Target Status Ingestion API

**Files**

- `backend/internal/httpapi/server.go`
- `backend/internal/httpapi/ingest_handlers.go`
- `backend/internal/httpapi/ingest_handlers_test.go`
- `backend/internal/app/ingest_target_status_batch.go`
- `contracts/profiling/payloads.md`

**Approach**

- Add collector-authenticated target-status ingestion route.
- Wire SQL-backed status repository into both ingest and query handlers.
- Preserve existing profile ingestion behavior.

**Test Scenarios**

- Missing/invalid collector token is rejected.
- Valid target status batch returns accepted.
- UI-authenticated target status query returns persisted records.

### U3. Collector Metadata Policy and Upload

**Files**

- `collector/runtime/runtime.go`
- `collector/runtime/runtime_test.go`
- `collector/internal/discovery/pod_watcher.go`
- `collector/internal/policy/policy.go`
- `collector/internal/pipeline/backend_client.go`
- `contracts/profiling/types.go`

**Approach**

- Reuse policy evaluation so unannotated JVMs are `disabled_by_metadata`.
- Upload target-status batches to backend.
- Keep profile upload separate; do not claim real async-profiler data until implemented.

**Test Scenarios**

- Annotated temporary Pod produces accepted/temporary target status.
- Unannotated HotSpot JVM produces disabled-by-metadata status.
- Backend unavailable increments upload retryable metrics.

### U4. Web API Routing in Deployed Chart

**Files**

- `Dockerfile.web`
- `deploy/helm/templates/web-deployment.yaml`
- `deploy/helm/templates/service.yaml`
- `deploy/helm/values.yaml`

**Approach**

- Add nginx config or runtime config so `/api/*` proxies to `java-profiler-backend`.
- Avoid compile-time-only `VITE_API_BASE` for in-cluster deployments.

**Test Scenarios**

- In-cluster Web port-forward `/api/ui/v1/target-status` reaches backend, not static 404.
- Static routes still serve the React app.

### U5. UI State Distinctions

**Files**

- `web/src/api/client.ts`
- `web/src/api/types.ts`
- `web/src/features/status/target-status-view.tsx`
- `web/src/features/ingestion/ingestion-health-view.tsx`
- `web/src/features/memory/memory-view.tsx`
- `web/src/features/cpu/cpu-view.tsx`
- `web/src/features/locks/locks-view.tsx`
- `web/src/features/deadlocks/deadlocks-view.tsx`
- `web/src/routes/service-overview.tsx`
- `web/tests/profiling-flow.spec.ts`

**Approach**

- Present backend unavailable separately from no matching data.
- Map known status reasons to user actions from the manual.
- Avoid rendering zero-value flamegraphs as if they were real evidence.

**Test Scenarios**

- Backend 404/503 shows backend unavailable.
- Empty successful status response shows no matching targets.
- Disabled/unsupported/conflict statuses show distinct messages.

### U6. Real Acceptance Script Notes

**Files**

- `docs/operations/java-profiling-runbook.md`
- `docs/operations/performance-analysis-user-manual.md`

**Approach**

- Update docs only if implementation changes behavior or accepted deployment steps.
- Keep acceptance instructions honest about profile data not being complete until async-profiler slice lands.

**Test Scenarios**

- Manual remains aligned with implemented status and no-data behavior.

## Sequencing

1. U1: fix schema so backend can start.
2. U2: persist/query target status.
3. U3: make collector upload real target status.
4. U4: route Web API to backend.
5. U5: improve UI diagnostic states.
6. U6: reconcile docs if behavior changed.

## Verification

- `go test ./...`
- `javac --release 11 java-helper/thread-diagnostics/src/main/java/com/ebpfjava/threads/*.java`
- `cd web && npm test -- --run`
- `cd web && npm run build`
- `helm lint ./deploy/helm --values deploy/helm/values.yaml`
- Real Kubernetes acceptance:
  - deploy ClickHouse, backend, collector, web, and annotated Java workload
  - confirm backend reaches Ready
  - confirm collector metrics show target status upload success
  - confirm Web port-forward shows status data instead of static `404`
  - capture screenshots/video for manual comparison

## Risks

- Process-to-Pod metadata mapping may require deeper container runtime inspection than this slice can finish.
- Full async-profiler profiling remains outside this fix, so flamegraphs may still legitimately show no profile samples.
- Current Web build is static; runtime backend routing must not break local Vite development.
