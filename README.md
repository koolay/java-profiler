# java-profiler

This repository currently captures the product and technical direction for a focused Java performance profiling system on Kubernetes.

The target problem is narrower than a general observability platform:

- profile Java services running in Kubernetes
- use node-local collection
- control enablement through Kubernetes metadata
- store results in ClickHouse
- present a small, service-centric service-diagnosis UI for profile, target status, and ingestion investigation

## Current State

The repository is transitioning from documentation-only into implementation. The source of truth remains the documentation under `docs/`, and the first code scaffolding now lives under:

- `cmd/backend`
- `cmd/collector`
- `backend/internal`
- `collector/internal`
- `contracts/profiling`
- `java-helper/thread-diagnostics`
- `examples/jdk17-http-demo`
- `web`
- `deploy`

## Core Documents

- `docs/brainstorms/java-profiler-requirements.md`
  - primary requirements draft
  - problem frame, actors, flows, acceptance examples, scope boundaries
- `docs/architecture/java-profiler-architecture.md`
  - software architecture
  - collector, backend, ClickHouse, query, and UI boundaries
- `docs/architecture/performance-ingestion-architecture-review.md`
  - performance architecture review for OOM, batch upload, ingestion limits, and ClickHouse query pressure
- `docs/research/coroot-node-agent-java-agent.md`
  - research notes on Coroot's Java agent and async-profiler-related behavior
- `docs/operations/java-profiling-runbook.md`
  - install-time and incident-time operator workflow
- `docs/operations/deployment-operations-admin-manual.md`
  - deployment, operations, security, storage, upgrade, and platform troubleshooting manual
- `docs/operations/performance-analysis-user-manual.md`
  - Java service owner workflow for the service-diagnosis page, including CPU, memory allocation, lock, deadlock, target status, and ingestion analysis
- `docs/operations/real-profiling-acceptance-standard.md`
  - mandatory real Kubernetes acceptance standard for collector, ingestion, profile storage, query API, and UI changes

## Product Direction

The current design assumes:

- Kubernetes DaemonSet collection
- opt-in profiling through annotations or labels
- HotSpot-compatible JVMs in the first version
- async-profiler for CPU, allocation, and lock profiling
- bounded retention with no collected data older than 7 days
- ClickHouse as the primary query and storage layer
- metrics exposed through collector/backend exporters only, with Prometheus-series services owning metric storage and dashboards
- a lightweight, self-owned UI rather than a broad observability workspace
- collector and backend Go container images built from `ghcr.io/koolay/library/golang:1.26.0`
- Kubernetes deployment artifacts under `deploy/helm`

## Explicit Scope Boundaries

The first version does not include:

- Pyroscope, Parca, Grafana, or other incompatible profile backends
- non-Java profiling
- OpenJ9 support
- distributed ClickHouse
- heap dump analysis or retained-heap dominator analysis
- general-purpose tracing, log analysis, or service map features
- Prometheus metrics storage or dashboard replacement

## Repository Layout

```text
cmd/
  backend/
  collector/
backend/
  internal/
collector/
  internal/
contracts/
  profiling/
java-helper/
  thread-diagnostics/
examples/
  jdk17-http-demo/
web/
  src/
deploy/
  helm/
docs/
  architecture/
  brainstorms/
  operations/
  research/
  plans/
```

## Local Verification

```bash
go test ./...
javac --release 11 java-helper/thread-diagnostics/src/main/java/com/ebpfjava/threads/*.java
cd examples/jdk17-http-demo && mvn test
cd web && npm install && npm test && npm run build
```

Optional local ClickHouse-compatible smoke check using chDB:

```bash
scripts/verify-chdb-local.sh
```

The script skips cleanly when `libchdb` is not installed. Use `CHDB_REQUIRED=1` to make missing chDB fail automation.

Real Kubernetes acceptance, including screenshots/video and target restart-count evidence, is handled by:

```bash
scripts/real-acceptance.sh --help
```

For profiling or UI changes, passing real acceptance means proving non-empty CPU, allocation, and lock-delay profile data from the current Kubernetes run window, plus browser UI acceptance against that real backend data. See `docs/operations/real-profiling-acceptance-standard.md`.

## Working Rule

When adding implementation or additional docs, keep them aligned with the requirements document. If a new assumption changes the product shape, update the docs first or in the same change.
