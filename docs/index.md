---
layout: home

hero:
  name: Java Profiler
  text: Kubernetes-native Java profiling documentation
  tagline: Operate, validate, and evolve a HotSpot-first profiler built around async-profiler and ClickHouse.
  actions:
    - theme: brand
      text: Start with Requirements
      link: /brainstorms/java-profiler-requirements
    - theme: alt
      text: Run the Profiler
      link: /operations/java-profiling-runbook

features:
  - title: Product Boundary
    details: Java services on Kubernetes, opt-in through annotations or labels, node-local collection, and bounded profile retention.
  - title: Real Acceptance
    details: Acceptance requires non-empty CPU, allocation, and lock profile data from a real Kubernetes workload.
  - title: Operations First
    details: Deployment, profiling, performance analysis, and E2E automation guides are grouped for operators and implementers.
---

## Start here

- [Requirements](./brainstorms/java-profiler-requirements.md) defines the product scope, actors, retention policy, and success criteria.
- [Architecture](./architecture/java-profiler-architecture.md) explains the collector, backend, ClickHouse store, and web UI.
- [Java Profiling Runbook](./operations/java-profiling-runbook.md) shows how to enable, disable, and validate profiling.
- [Real Profiling Acceptance Standard](./operations/real-profiling-acceptance-standard.md) defines the evidence required before profiling changes are complete.

## Main sections

- [Operations](./operations/java-profiling-runbook.md): deployment, profiling, performance analysis, E2E testing, and acceptance.
- [Research](./research/coroot-node-agent-java-agent.md): upstream references and technology studies.

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
