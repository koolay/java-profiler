# Directory Index

## Files

- **[index.md](./index.md)** - Documentation directory index

## Subdirectories

### architecture/

- **[java-profiler-architecture.md](./architecture/java-profiler-architecture.md)** - Java profiling system architecture
- **[performance-ingestion-architecture-review.md](./architecture/performance-ingestion-architecture-review.md)** - Performance architecture review for OOM, batch upload, ingestion limits, and query pressure

### brainstorms/

- **[java-profiler-requirements.md](./brainstorms/java-profiler-requirements.md)** - Java profiling product requirements

### plans/

- **[2026-05-08-001-feat-java-profiler-implementation-plan.md](./plans/2026-05-08-001-feat-java-profiler-implementation-plan.md)** - Java profiling implementation plan
- **[2026-05-08-002-fix-feature-slice-review-findings.md](./plans/2026-05-08-002-fix-feature-slice-review-findings.md)** - Fix plan for review findings in the vertical slice
- **[2026-05-10-004-flamegraph-actionable-context-ux.md](./plans/2026-05-10-004-flamegraph-actionable-context-ux.md)** - Flame graph actionable-context UX fixes for Java CPU bottleneck analysis
- **[2026-05-10-005-pyroscope-study-alignment-polish.md](./plans/2026-05-10-005-pyroscope-study-alignment-polish.md)** - Final Pyroscope study alignment polish for CPU table labels, insight copy, frame categories, and focus state
- **[2026-05-10-006-fix-pyroscope-analysis-ux-plan.md](./plans/2026-05-10-006-fix-pyroscope-analysis-ux-plan.md)** - Plan to close Pyroscope analysis UX gaps with tooltip metrics, combined workflow, visual hierarchy, and focus navigation

### operations/

- **[deployment-operations-admin-manual.md](./operations/deployment-operations-admin-manual.md)** - Deployment, operations, security, storage, upgrade, and platform troubleshooting manual
- **[e2e-automation-test-guide.md](./operations/e2e-automation-test-guide.md)** - End-to-end automation test guide aligned with the Java profiling user manual
- **[java-profiling-runbook.md](./operations/java-profiling-runbook.md)** - Operator enablement, status, retention, and troubleshooting runbook
- **[performance-analysis-user-manual.md](./operations/performance-analysis-user-manual.md)** - Java service performance analysis manual for service owners and incident responders
- **[real-profiling-acceptance-standard.md](./operations/real-profiling-acceptance-standard.md)** - Mandatory real Kubernetes acceptance standard for profile data and UI workflow changes

### research/

- **[coroot-node-agent-java-agent.md](./research/coroot-node-agent-java-agent.md)** - Coroot Java agent research
- **[chdb-go.md](./research/chdb-go.md)** - chDB Go binding research for local embedded ClickHouse-compatible validation
- **[pyroscope-profile-ui-study.md](./research/pyroscope-profile-ui-study.md)** - Pyroscope-style profile UI research and product review notes for bottleneck analysis

### implementation/

- `cmd/backend` - backend entrypoint scaffold
- `cmd/collector` - collector entrypoint scaffold
- `backend/internal` - backend domain, ClickHouse, HTTP, and metrics packages
- `collector/internal` - collector policy, discovery, attach, and pipeline packages
- `contracts/profiling` - stable payload and configuration contracts
- `java-helper/thread-diagnostics` - Java helper for thread snapshots and deadlocks
- `web` - React/Vite UI
- `deploy` - Helm and manifest packaging
- `tools/chdb-smoke` - optional local chDB smoke verification for schema and query behavior
