# AGENTS.md for `java-profiler`

## Purpose

This repository documents the intended design of a Java performance profiling system for Kubernetes. The repo is currently documentation-first.

## Source of Truth

1. `docs/brainstorms/java-profiler-requirements.md`
2. `docs/research/coroot-node-agent-java-agent.md`
3. `README.md`

## Required Constraints

- Keep the first version focused on Java services running on Kubernetes.
- Use Kubernetes annotations or labels for opt-in control.
- Assume a DaemonSet-based node-local collector.
- Assume HotSpot-compatible JVMs first.
- Assume async-profiler for CPU, allocation, and lock profiling.
- Use ClickHouse as the primary profile query store.
- Keep data retention bounded to 7 days or less.
- Do not introduce Pyroscope, Parca, or Grafana as required backend dependencies.
- Do not expand the scope into general observability, tracing, log analysis, or non-Java profiling unless the requirements are updated.

## Research Source Rules

- For any web research, technology selection, dependency evaluation, architecture comparison, or implementation reference, search and cite only English-language international sources by default.
- Prefer primary sources: official documentation, official GitHub repositories, standards documents, release notes, license files, and reputable international engineering writeups.
- Do not use Chinese-language community sources, Chinese blogs, Chinese forums, Zhihu, Juejin, CSDN, SegmentFault, WeChat articles, Gitee mirrors, or translated summaries as research evidence unless the user explicitly asks for Chinese-community context.
- If Chinese-community context is explicitly requested, clearly separate it from the main evidence and do not use it to override primary international sources.
- When using search queries, write them in English unless the user explicitly asks otherwise.

## Editing Guidance

- Keep docs concrete and internally consistent.
- Preserve the product boundary between profiling, thread snapshots, and exporter metrics. Metrics are exposed through collector/backend exporters only; Prometheus-series services own metric storage, dashboards, alerting, and retention.
- When a change affects scope, retention, collection, or storage, update the requirements document at the same time.
- Do not rewrite unrelated content just to normalize style.

## Real Profiling Acceptance Standard

- Follow `docs/operations/real-profiling-acceptance-standard.md` for any change that touches collector profiling, ingestion, ClickHouse storage, backend query APIs, Kubernetes deployment, the JDK17 demo service, or the profile UI.
- Real acceptance must use `export KUBECONFIG=$HOME/backup/localk8s.yaml` and must deploy images built from the current workspace.
- Completion requires proof of real profile data, not just healthy pods or a non-crashing UI: accepted target status, non-empty CPU profile, non-empty allocation profile, non-empty lock-delay profile, ClickHouse profile rows, ingestion evidence, bounded 7-day TTL, browser UI acceptance, and no target workload restart increase.
- UI acceptance must validate the real performance-analysis workflow: Top Table, Flame Graph, Both, Self/Total semantics, search highlighting, selected-frame details, focus, Back, Reset, and ingestion evidence.
- If strict acceptance fails, fix the product, test workload, script, or deployment and rerun until it passes. Do not mark the task complete with empty CPU/memory/lock data.

## Suggested Working Order

1. Read the requirements document.
2. Read the research note if the change touches Java agent or async-profiler behavior.
3. Read the real profiling acceptance standard if the change touches profiling or UI behavior.
4. Edit the smallest set of files needed.
5. Reconcile any downstream wording changes in `README.md`, `CLAUDE.md`, and `AGENTS.md`.
