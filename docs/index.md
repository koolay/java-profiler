---
layout: home

hero:
  name: Java Profiler
  text: Low-overhead Java profiling for Kubernetes incidents
  tagline: "A focused profiler for HotSpot services on Kubernetes: opt-in collection, async-profiler/JFR-derived evidence, ClickHouse storage, and a UI built for Java incident diagnosis."
  actions:
    - theme: brand
      text: Quickstart
      link: /getting-started/quickstart
    - theme: alt
      text: Analyze a Service
      link: /operations/performance-analysis-user-manual
    - theme: alt
      text: GitHub
      link: https://github.com/koolay/java-profiler

features:
  - title: Opt-in by default
    details: Profiling is opt-in through Kubernetes metadata, collected node-locally, and retained for 7 days or less.
  - title: Real Java evidence
    details: CPU, Wall Clock, Java I/O wait, GC, allocation summaries, lock delay, thread, deadlock, status, and ingestion evidence stay tied to one service and time range.
  - title: Own the profiling stack
    details: No required Pyroscope, Parca, or Grafana backend. async-profiler data lands in ClickHouse and a self-owned UI.
  - title: Fast time to first signal
    details: The first usable path is status, then CPU or allocation summary, then Wall Clock, I/O, GC, locks, or ingestion when the incident needs deeper evidence.
---

[![Docs](https://img.shields.io/badge/docs-online-blue?style=flat-square)](https://koolay.github.io/java-profiler/) [![中文文档](https://img.shields.io/badge/docs-中文文档-2b90d9?style=flat-square)](https://koolay.github.io/java-profiler/zh/) [![GitHub stars](https://img.shields.io/github/stars/koolay/java-profiler?style=flat-square)](https://github.com/koolay/java-profiler)

![Real allocation profile analysis from the acceptance environment](./assets/screenshots/real-allocation-analysis.png)

## 3-minute path

1. Open [Quickstart](./getting-started/quickstart.md) and enable profiling with Kubernetes metadata.
2. Open the target service, then check `status` first to confirm the JVM was accepted.
3. Move from `cpu` or `memory` to `wall`, `io`, `gc`, `locks`, and `ingestion` to get from symptom to evidence.

## For service owners

- [Quickstart](./getting-started/quickstart.md): enable profiling and read your first service profile.
- [Performance Analysis Manual](./operations/performance-analysis-user-manual.md): read CPU, Wall Clock, Java I/O wait, GC, allocation summary, lock, deadlock, target status, profile evidence guidance, and ingestion evidence.
- [Java Profiling Runbook](./operations/java-profiling-runbook.md): enable temporary or continuous profiling for a Kubernetes workload.

## For platform operators

- [Deployment and Operations](./operations/deployment-operations-admin-manual.md): install, secure, operate, upgrade, and troubleshoot the profiler.
- [Real Profiling Acceptance](./operations/real-profiling-acceptance-standard.md): prove CPU, Wall Clock, Java I/O wait, GC, allocation, lock, ClickHouse, UI, and ingestion behavior before shipping changes.

## For contributors

- [Contributing](./contributing/development.md): run local checks, build docs, and execute real acceptance.
- [Architecture](./architecture/java-profiler-architecture.md): understand the collector, backend, ClickHouse store, contracts, and web UI.
- [E2E Automation Guide](./operations/e2e-automation-test-guide.md): run browser and real Kubernetes acceptance flows.
- [Profiling Contracts](./reference/profiling-contracts.md): inspect stable payload and configuration contracts.
- [GitHub Issues](https://github.com/koolay/java-profiler/issues): report bugs and request changes.

## Trust and localization

- Use the English / 简体中文 switch in the top bar when you want the localized path.
- The real screenshots on this page come from the acceptance environment, not mocked UI state. The allocation screenshot shows the current Allocation Summary and flamegraph workflow.

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
