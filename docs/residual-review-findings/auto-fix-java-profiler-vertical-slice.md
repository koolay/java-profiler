## Residual Review Findings

Source review: BMAD review of commit `2a2f240fcc0f21626c833b12cd94d20d9a63ad85` plus autofix pass on `auto-fix-java-profiler-vertical-slice`.

- Critical, `cmd/collector/main.go`: Collector runtime now scans processes and exports status metrics, but the full async-profiler attach, JFR collection, batching, and backend upload loop remains incomplete. This requires a broader collector execution unit.
- High, `backend/internal/clickhouse/sql_repository.go`: ClickHouse ingestion idempotency still relies on a read-then-insert pattern; a complete fix needs a deterministic concurrent dedupe design and query-level duplicate suppression strategy.
- Medium, `backend/internal/httpapi/server.go`: Thread snapshot and target status query APIs still lack collector ingestion HTTP endpoints. Add versioned collector routes before claiming those UI views are end-to-end.
- Medium, `web/src/api/client.ts` and `deploy/helm/templates/web-deployment.yaml`: UI auth now supports a server-set token cookie, but the chart still needs a production ingress/reverse-proxy or login flow to set that cookie and route `/api` consistently.
- Medium, `backend/internal/app/ingest_profile_batch.go` and `collector/runtime/runtime.go`: Backend/collector metrics were improved only at the collector scan level. Ingestion status, retention health, upload retry, and dropped-batch metrics still need end-to-end exporter wiring.
