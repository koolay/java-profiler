# java-profiler

[![Docs](https://img.shields.io/badge/docs-online-blue?style=flat-square)](https://koolay.github.io/java-profiler/) [![中文文档](https://img.shields.io/badge/docs-中文文档-2b90d9?style=flat-square)](https://koolay.github.io/java-profiler/zh/) [![GitHub stars](https://img.shields.io/github/stars/koolay/java-profiler?style=flat-square)](https://github.com/koolay/java-profiler)

`java-profiler` helps teams find the Java stack behind a performance problem in a Kubernetes service. It collects real async-profiler/JFR-derived data from HotSpot-compatible JVMs and presents CPU, allocation, lock, thread, GC, and I/O evidence in a service-focused UI.

## What it is for

Use it when your existing monitoring tells you that a Java service is slow, using too much CPU, approaching an OOM kill, pausing for GC, or waiting on locks—but does not show which Java code is responsible.

- Enable profiling with Kubernetes annotations or labels; application code does not need to change.
- Collect CPU, Wall Clock, allocation, lock-delay, Java I/O wait, GC, thread, and deadlock evidence from the target JVM.
- Query the data from ClickHouse with retention capped at seven days.
- Keep metric storage, dashboards, alerting, logs, and tracing in the systems that already own those concerns.

The first version is deliberately limited to Java services on Kubernetes, HotSpot-compatible JVMs, a node-local DaemonSet collector, async-profiler, ClickHouse, and a compact diagnosis UI. Pyroscope, Parca, and Grafana are not required backend dependencies.

## Quickstart

Add temporary profiling to the target workload's Pod template:

```yaml
metadata:
  annotations:
    java-profiler.io/profile-mode: temporary
    java-profiler.io/profile-duration: 15m
```

Open the Web UI, select the namespace, service, and time range, and check `status` first. Then choose the view that matches the symptom:

- `cpu` for expensive Java methods;
- `wall` for runnable, blocked, waiting, sleeping, or I/O time that CPU does not explain;
- `io` for Java-owned socket or file blocking;
- `gc` for JVM pause events and their allocation context;
- `memory` for sampled allocation pressure;
- `locks` and `deadlocks` for contention and deadlock cycles;
- `ingestion` to confirm that profile batches reached the backend.

See the [Quickstart](docs/getting-started/quickstart.md) and [Performance Analysis Manual](docs/operations/performance-analysis-user-manual.md).

## What the profiler can show

- CPU hotspots with Self CPU, Total CPU, and sampled stack context.
- Wall Clock latency broken down by runnable, blocked, waiting, sleeping, and I/O paths.
- Java I/O wait when JVM/JFR data preserves ownership of the blocking path.
- GC pause events correlated with allocation profiles in the same time window.
- Sampled allocation totals, top allocating paths, top self-allocating frames, and allocation flamegraph context.
- Lock delay caused by synchronized or monitor paths under contention.
- Thread snapshots for CPU, lock, sleep, blocked, and waiting states.
- Deadlock cycles reported by the target JVM.
- Target and ingestion status that explain disabled profiling, unsupported JVMs, attach failures, conflicts, expired temporary windows, rejected uploads, and dropped data.

Allocation profiles identify where objects are created. They do not provide retained-heap ownership, dominator trees, or a heap-leak analysis.

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

The collector discovers eligible JVMs on its node, starts bounded profiling sessions, and uploads normalized data. The backend stores query-ready profiles and diagnosis records in ClickHouse. The UI keeps the same service, Pod, JVM, profile type, and time-range context as the investigation moves between views.

## Screenshots

These screenshots come from a real Kubernetes acceptance environment, not mocked UI state.

![Real allocation profile analysis from the acceptance environment](docs/assets/screenshots/real-allocation-analysis.png)

- [Target status](docs/assets/screenshots/real-target-status.png)
- [CPU profile analysis](docs/assets/screenshots/real-cpu-analysis.png)
- [Allocation analysis](docs/assets/screenshots/real-allocation-analysis.png)
- [Wall Clock latency](docs/assets/screenshots/real-wall-clock.png)
- [Java I/O wait](docs/assets/screenshots/real-io-wait.png)
- [GC pause and allocation correlation](docs/assets/screenshots/real-gc-pauses.png)
- [Deadlock diagnosis](docs/assets/screenshots/real-deadlocks.png)
- [Ingestion health](docs/assets/screenshots/real-ingestion-health.png)

Regenerate the screenshots from a port-forwarded real UI:

```bash
export REAL_ACCEPTANCE_BASE_URL=http://127.0.0.1:18081
export REAL_ACCEPTANCE_NAMESPACE=java-profiler-qa
export REAL_ACCEPTANCE_SERVICE=jdk17-http-demo
node scripts/capture-doc-screenshots.mjs
```

## Develop

Run the relevant checks before changing profiling, ingestion, backend APIs, or UI behavior:

```bash
go test ./...
javac --release 11 java-helper/thread-diagnostics/src/main/java/com/ebpfjava/threads/*.java
cd examples/jdk17-http-demo && mvn test
cd ../../web && npm ci && npm test && npm run build
```

Build the documentation site with:

```bash
cd docs
npm install
npm run docs:build
```

Changes that touch collector profiling, ingestion, ClickHouse, backend query APIs, deployment, the demo service, or the profile UI also need real Kubernetes acceptance. Start with [Contributing](docs/contributing/development.md) and follow the [Real Profiling Acceptance Standard](docs/operations/real-profiling-acceptance-standard.md).

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

The first version does not include non-Java profiling, OpenJ9 support, heap-dump analysis, distributed ClickHouse, tracing, log analysis, service maps, dashboarding, alerting, or Prometheus metric storage.

Collectors and backends may expose operational metrics. Prometheus-series services own metric storage, dashboards, alerting, and retention.
