---
layout: home

hero:
  name: Java Profiler
  text: Find the Java stack behind Kubernetes performance problems
  tagline: "A focused profiler for HotSpot services on Kubernetes: opt-in collection, async-profiler/JFR-derived evidence, ClickHouse storage, and a UI built for Java incident diagnosis."
  actions:
    - theme: brand
      text: Quickstart
      link: /getting-started/quickstart
    - theme: alt
      text: Analyze a Service
      link: /operations/performance-analysis-user-manual

features:
  - title: Production-safe by default
    details: Profiling is opt-in through Kubernetes metadata, collected node-locally, and retained for 7 days or less.
  - title: Real Java evidence
    details: CPU, Wall Clock, Java I/O wait, GC, allocation, lock delay, thread, deadlock, status, and ingestion evidence stay tied to one service and time range.
  - title: Own the profiling stack
    details: No required Pyroscope, Parca, or Grafana backend. async-profiler data lands in ClickHouse and a self-owned UI.
---

## For service owners

- [Quickstart](./getting-started/quickstart.md): enable profiling and read your first service profile.
- [Performance Analysis Manual](./operations/performance-analysis-user-manual.md): read CPU, Wall Clock, Java I/O wait, GC, allocation, lock, deadlock, target status, and ingestion evidence.
- [Java Profiling Runbook](./operations/java-profiling-runbook.md): enable temporary or continuous profiling for a Kubernetes workload.

## For platform operators

- [Deployment and Operations](./operations/deployment-operations-admin-manual.md): install, secure, operate, upgrade, and troubleshoot the profiler.
- [Real Profiling Acceptance](./operations/real-profiling-acceptance-standard.md): prove CPU, Wall Clock, Java I/O wait, GC, allocation, lock, ClickHouse, UI, and ingestion behavior before shipping changes.

## For contributors

- [Contributing](./contributing/development.md): run local checks, build docs, and execute real acceptance.
- [Architecture](./architecture/java-profiler-architecture.md): understand the collector, backend, ClickHouse store, contracts, and web UI.
- [E2E Automation Guide](./operations/e2e-automation-test-guide.md): run browser and real Kubernetes acceptance flows.
- [Profiling Contracts](./reference/profiling-contracts.md): inspect stable payload and configuration contracts.

## Local preview

Run these commands from the repository root:

```bash
cd docs
npm install
npm run docs:dev
```

Build the publishable static site:

```bash
cd docs
npm run docs:build
```
