---
layout: home

hero:
  name: Java Profiler
  text: Low-overhead Java profiling for Kubernetes incidents
  tagline: "Find the Java stack behind a Kubernetes performance problem with opt-in profiling, real async-profiler/JFR evidence, and a focused diagnosis UI."
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
  - title: Opt in when you need it
    details: Enable profiling with Kubernetes metadata. The node-local collector keeps collection controlled and retention at seven days or less.
  - title: Follow the Java stack
    details: CPU, Wall Clock, I/O wait, GC, allocation, lock, thread, deadlock, status, and ingestion data stay tied to one service and time range.
  - title: Keep the stack self-owned
    details: async-profiler data lands in ClickHouse and the project UI; Pyroscope, Parca, and Grafana are not required backend dependencies.
  - title: Start with the useful question
    details: Check target status, then use CPU or allocation. Move to Wall Clock, I/O, GC, locks, or ingestion when the first view does not explain the incident.
---

[![Docs](https://img.shields.io/badge/docs-online-blue?style=flat-square)](https://koolay.github.io/java-profiler/) [![中文文档](https://img.shields.io/badge/docs-中文文档-2b90d9?style=flat-square)](https://koolay.github.io/java-profiler/zh/) [![GitHub stars](https://img.shields.io/github/stars/koolay/java-profiler?style=flat-square)](https://github.com/koolay/java-profiler)

![Real allocation profile analysis from the acceptance environment](./assets/screenshots/real-allocation-analysis.png)

## Start here

1. Open [Quickstart](./getting-started/quickstart.md) and enable profiling with Kubernetes metadata.
2. Open the target service and check `status` to confirm that the JVM was accepted.
3. Start with `cpu` or `memory`, then move to `wall`, `io`, `gc`, `locks`, or `ingestion` as the incident requires.

## If you own a service

- [Quickstart](./getting-started/quickstart.md): enable profiling and read your first service profile.
- [Performance Analysis Manual](./operations/performance-analysis-user-manual.md): read CPU, Wall Clock, Java I/O wait, GC, allocation summary, lock, deadlock, target status, profile evidence guidance, and ingestion evidence.
- [Java Profiling Runbook](./operations/java-profiling-runbook.md): enable temporary or continuous profiling for a Kubernetes workload.

## If you run the platform

- [Deployment and Operations](./operations/deployment-operations-admin-manual.md): install, secure, operate, upgrade, and troubleshoot the profiler.
- [Real Profiling Acceptance](./operations/real-profiling-acceptance-standard.md): prove CPU, Wall Clock, Java I/O wait, GC, allocation, lock, ClickHouse, UI, and ingestion behavior before shipping changes.

## If you are contributing

- [Contributing](./contributing/development.md): run local checks, build docs, and execute real acceptance.
- [Architecture](./architecture/java-profiler-architecture.md): understand the collector, backend, ClickHouse store, contracts, and web UI.
- [E2E Automation Guide](./operations/e2e-automation-test-guide.md): run browser and real Kubernetes acceptance flows.
- [Profiling Contracts](./reference/profiling-contracts.md): inspect stable payload and configuration contracts.
- [GitHub Issues](https://github.com/koolay/java-profiler/issues): report bugs and request changes.

## Language and evidence

- Use the English / 简体中文 switch in the top bar to change languages.
- The screenshots on this page come from a real acceptance environment. The allocation screenshot shows the current Allocation Summary and flamegraph workflow; it is not mocked UI state.

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
