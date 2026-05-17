# java-profiler

Java performance profiling for Kubernetes services. Find where a HotSpot JVM is spending CPU, allocating memory, waiting on locks, pausing for GC, or blocking on Java I/O, using real async-profiler/JFR-derived data and a service-focused UI.

[Docs](https://koolay.github.io/java-profiler/) · [中文文档](https://koolay.github.io/java-profiler/zh/) · [Quickstart](https://koolay.github.io/java-profiler/getting-started/quickstart) · [Analyze a service](https://koolay.github.io/java-profiler/operations/performance-analysis-user-manual) · [Contributing](https://koolay.github.io/java-profiler/contributing/development)

## Why java-profiler

Most observability stacks tell you that a Java service is slow. `java-profiler` is for the next question: which Java stack is responsible?

- **Kubernetes-native opt-in**: enable profiling with annotations or labels. No application code changes.
- **Real JVM profile data**: CPU, Wall Clock, allocation, lock-delay, Java I/O wait, and GC evidence come from async-profiler/JFR-derived collection.
- **Expert Java workbench**: Top Table, Flame Graph, selected-frame details, native-frame filtering, target status, deadlocks, and ingestion health in one workflow.
- **Ownable storage**: profile data lands in ClickHouse with retention bounded to 7 days or less.
- **Focused scope**: no required Pyroscope, Parca, or Grafana backend.
- **Built for proof**: real acceptance requires non-empty CPU, Wall Clock, Java I/O wait, GC, allocation, lock, ClickHouse, ingestion, and browser UI evidence.

## Quickstart

Enable temporary profiling on a workload pod template:

```yaml
metadata:
  annotations:
    java-profiler.io/profile-mode: temporary
    java-profiler.io/profile-duration: 15m
```

Open the Web UI, select the namespace, service, and time range, then start with:

- `status` to confirm the JVM was accepted.
- `cpu` to find expensive Java methods.
- `wall` when latency is not explained by CPU alone.
- `io` to isolate Java-owned socket or file blocking paths.
- `gc` to correlate JVM pause evidence with allocation pressure.
- `memory` to inspect allocation pressure.
- `locks` and `deadlocks` to investigate contention.
- `ingestion` to confirm profile batches were accepted.

See the [Quickstart](docs/getting-started/quickstart.md) and [Performance Analysis Manual](docs/operations/performance-analysis-user-manual.md).

## What it analyzes

- CPU hotspots: high-cost Java methods, self time, total time, and sampled stack context.
- Wall Clock latency: Java stack time spent runnable, blocked, waiting, sleeping, or doing I/O.
- Java I/O wait: socket or file blocking paths when JVM/JFR evidence preserves Java ownership.
- GC pauses: JVM GC event evidence correlated with allocation profiles and the incident window.
- Allocation hotspots: methods and call paths creating allocation pressure.
- Lock delay: synchronized or monitor paths that block under contention.
- Thread evidence: snapshots for CPU, lock, sleep, blocked, and waiting states.
- Deadlock evidence: deadlock cycles reported by the target JVM.
- Profiling health: accepted, disabled, unsupported, attach failure, profiler conflict, rejected upload, or dropped ingestion data.

## How it works

```text
Kubernetes metadata
        |
        v
Node-local collector DaemonSet
        |
        v
async-profiler/JFR + thread diagnostics
        |
        v
Backend API -> ClickHouse
        |
        v
Service diagnosis UI
```

The first version targets Java services running on Kubernetes, HotSpot-compatible JVMs first. Profiling is controlled through Kubernetes metadata, collected node-locally, stored in ClickHouse, and exposed through a compact UI for service owners and platform engineers.

## Screenshots

These screenshots come from a real Kubernetes acceptance environment, not mocked UI state.

![CPU profile analysis showing real DemoHttpService hotspots](docs/assets/screenshots/real-cpu-analysis.png)

- [Target status evidence](docs/assets/screenshots/real-target-status.png)
- [Wall Clock latency evidence](docs/assets/screenshots/real-wall-clock.png)
- [Java I/O wait evidence](docs/assets/screenshots/real-io-wait.png)
- [GC pause and allocation correlation](docs/assets/screenshots/real-gc-pauses.png)
- [Deadlock diagnosis surface](docs/assets/screenshots/real-deadlocks.png)
- [Ingestion health evidence](docs/assets/screenshots/real-ingestion-health.png)

Regenerate them from a port-forwarded real UI:

```bash
export REAL_ACCEPTANCE_BASE_URL=http://127.0.0.1:18081
export REAL_ACCEPTANCE_NAMESPACE=java-profiler-qa
export REAL_ACCEPTANCE_SERVICE=jdk17-http-demo
node scripts/capture-doc-screenshots.mjs
```

## Develop

Run local checks before changing profiling, ingestion, backend APIs, or UI behavior:

```bash
go test ./...
javac --release 11 java-helper/thread-diagnostics/src/main/java/com/ebpfjava/threads/*.java
cd examples/jdk17-http-demo && mvn test
cd ../../web && npm ci && npm test && npm run build
```

Build the docs site:

```bash
cd docs
npm install
npm run docs:build
```

For changes touching collector profiling, ingestion, ClickHouse storage, backend query APIs, deployment, the demo service, or profile UI, run real Kubernetes acceptance. See [Contributing](docs/contributing/development.md) and the [Real Profiling Acceptance Standard](docs/operations/real-profiling-acceptance-standard.md).

## Documentation

- [Online docs](https://koolay.github.io/java-profiler/)
- [中文文档](https://koolay.github.io/java-profiler/zh/)
- [Quickstart](docs/getting-started/quickstart.md)
- [Analyze a Java service](docs/operations/performance-analysis-user-manual.md)
- [Enable profiling](docs/operations/java-profiling-runbook.md)
- [Deploy and operate the platform](docs/operations/deployment-operations-admin-manual.md)
- [Development setup](docs/contributing/development.md)
- [Localization policy](docs/contributing/localization.md)
- [Architecture](docs/architecture/java-profiler-architecture.md)

## Scope

The first version does not include non-Java profiling, OpenJ9 support, heap dump analysis, distributed ClickHouse, tracing, log analysis, service maps, dashboarding, alerting, or Prometheus metric storage.

Metrics may be exposed by collector/backend exporters, but Prometheus-series systems own metric storage, dashboards, alerting, and retention.
