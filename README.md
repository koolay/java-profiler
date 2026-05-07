# java-profiler

This repository currently captures the product and technical direction for a focused Java performance profiling system on Kubernetes.

The target problem is narrower than a general observability platform:

- profile Java services running in Kubernetes
- use node-local collection
- control enablement through Kubernetes metadata
- store results in ClickHouse
- present a small, service-centric UI for profile and thread-diagnosis investigation

## Current State

There is no production code in the repository yet. The source of truth is the documentation under `docs/`.

## Core Documents

- `docs/brainstorms/java-profiler-requirements.md`
  - primary requirements draft
  - problem frame, actors, flows, acceptance examples, scope boundaries
- `docs/architecture/java-profiler-architecture.md`
  - software architecture
  - collector, backend, ClickHouse, query, and UI boundaries
- `docs/research/coroot-node-agent-java-agent.md`
  - research notes on Coroot's Java agent and async-profiler-related behavior

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
docs/
  architecture/
  brainstorms/
  research/
```

## Working Rule

When adding implementation or additional docs, keep them aligned with the requirements document. If a new assumption changes the product shape, update the docs first or in the same change.
