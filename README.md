# java-profiler

Java Profiler is a focused performance profiling system for Java services running on Kubernetes. It helps service owners and platform engineers answer one question with real data: where is this Java service spending CPU time, allocating memory, or waiting on locks?

The project is intentionally narrower than a full observability platform. It provides node-local profiling, bounded profile storage, and a service-centric UI for profile analysis, target status, and ingestion health.

## What It Analyzes

- CPU hotspots: high-cost Java methods, self time, total time, and full sampled stack context.
- Allocation hotspots: methods and call paths that create allocation pressure.
- Lock delay: synchronized or monitor-related paths that block under contention.
- Thread evidence: thread snapshots that help cross-check CPU, lock, sleep, and blocked states.
- Deadlock evidence: deadlock events when the target JVM reports them.
- Profiling health: whether a JVM was accepted, disabled, unsupported, failed attach, conflicted with another profiler, or produced rejected/dropped ingestion data.

## How It Works

The first version assumes:

- Java workloads run in Kubernetes.
- Profiling is opt-in through Kubernetes annotations or labels.
- A DaemonSet collector runs node-local and discovers Java processes through node/pod metadata.
- HotSpot-compatible JVMs are supported first.
- async-profiler collects CPU, allocation, and lock profiles.
- The backend stores profile data in ClickHouse with retention bounded to 7 days or less.
- The Web UI provides a compact service-diagnosis workflow for status, CPU, memory allocation, locks, deadlocks, and ingestion.

## Current State

The repository has moved beyond documentation-only planning. The implementation is split across:

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

Release delivery is automated from `vX.Y.Z` tag pushes. The workflow publishes backend, collector, and web images to GHCR, emits SBOM/provenance attestations, packages the Helm chart, and creates the matching GitHub Release with image digests.

## Quick Verification

Run local checks before changing profiling, ingestion, backend APIs, or UI behavior:

```bash
go test ./...
javac --release 11 java-helper/thread-diagnostics/src/main/java/com/ebpfjava/threads/*.java
cd examples/jdk17-http-demo && mvn test
cd web && npm ci && npm test && npm run build
```

Optional local ClickHouse-compatible smoke check using chDB:

```bash
scripts/verify-chdb-local.sh
```

The script skips cleanly when `libchdb` is not installed. Use `CHDB_REQUIRED=1` to make missing chDB fail automation.

## Real Kubernetes Acceptance

Real acceptance is required for changes that affect collector profiling, ingestion, ClickHouse storage, backend query APIs, Kubernetes deployment, the JDK17 demo service, or the profile UI.

Use a real cluster:

```bash
export KUBECONFIG=$HOME/backup/localk8s.yaml
```

Build backend, collector, and web images from the current workspace:

```bash
export BACKEND_IMAGE=java-profiler-backend:qa-$(date +%Y%m%d%H%M%S)
export COLLECTOR_IMAGE=java-profiler-collector:qa-$(date +%Y%m%d%H%M%S)
export WEB_IMAGE=java-profiler-web:qa-$(date +%Y%m%d%H%M%S)

bash scripts/build-real-acceptance-images.sh
```

Run strict real profiling acceptance against the JDK17 demo target:

```bash
scripts/real-acceptance.sh \
  --service jdk17-http-demo \
  --configure-profiler \
  --require-full-profiling \
  --high-volume \
  --artifact-dir /tmp/java-profiler-real-acceptance-$(date +%Y%m%d%H%M%S)
```

Passing real acceptance means proving all of the following from the current Kubernetes run window:

- target status has an accepted Java target
- CPU profile is non-empty
- allocation profile is non-empty
- lock-delay profile is non-empty
- ClickHouse has profile sample and stack rows
- ingestion has accepted profile batches without unexplained rejected/dropped/truncated data
- profile TTL remains bounded to 7 days or less
- Browser UI acceptance passes against real backend data
- target workload restart count does not increase

See `docs/operations/real-profiling-acceptance-standard.md` for the full standard.

## Documentation

- `docs/brainstorms/java-profiler-requirements.md`: product requirements, actors, flows, acceptance examples, and scope boundaries.
- `docs/architecture/java-profiler-architecture.md`: collector, backend, ClickHouse, query, and UI architecture.
- `docs/architecture/performance-ingestion-architecture-review.md`: ingestion hardening, batch limits, OOM risk, and ClickHouse query pressure.
- `docs/research/coroot-node-agent-java-agent.md`: research notes on Coroot's Java agent and async-profiler behavior.
- `docs/operations/performance-analysis-user-manual.md`: service owner workflow for CPU, allocation, lock, deadlock, status, and ingestion analysis.
- `docs/operations/java-profiling-runbook.md`: operator workflow for enabling profiling, reading statuses, retention, and troubleshooting.
- `docs/operations/deployment-operations-admin-manual.md`: deployment, operations, security, storage, upgrade, and platform troubleshooting.
- `docs/operations/e2e-automation-test-guide.md`: real E2E and browser automation guide.
- `docs/operations/real-profiling-acceptance-standard.md`: mandatory real Kubernetes acceptance standard.
- `docs/index.md`: full documentation directory index.

## Scope Boundaries

The first version does not include:

- Pyroscope, Parca, Grafana, or another required profile backend.
- Non-Java profiling.
- OpenJ9 support.
- Distributed ClickHouse.
- Heap dump analysis or retained-heap dominator analysis.
- General-purpose tracing, log analysis, service maps, dashboarding, or alerting.
- Prometheus metrics storage or dashboard replacement.

Metrics may be exposed by collector/backend exporters, but Prometheus-series systems own metric storage, dashboards, alerting, and retention.

## Working Rule

Keep implementation and documentation aligned with `docs/brainstorms/java-profiler-requirements.md`. If a change affects scope, retention, collection behavior, storage, or user workflow, update the relevant operation manual and acceptance standard in the same change.
