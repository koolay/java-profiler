# Claude Instructions for `ebpf-java`

## What this repository is

This repository is a documentation-first workspace for a Java performance profiling system on Kubernetes.

Treat the requirements document as the source of truth:

- `docs/brainstorms/java-k8s-performance-profiling-requirements.md`

Use the research note for implementation context:

- `docs/research/coroot-node-agent-java-agent.md`

## Ground Rules

- Keep changes grounded in the existing docs.
- Do not invent a broader product scope than the requirements support.
- Keep the first version focused on Java, Kubernetes, HotSpot, async-profiler, ClickHouse, and a narrow UI.
- Preserve the retention ceiling: no collected data should be retained for more than 7 days.
- Avoid introducing AGPL-dependent profile backends such as Pyroscope, Parca, or Grafana as core dependencies.

## When Writing or Editing Docs

- Prefer concrete, testable language over marketing language.
- Keep Kubernetes control behavior explicit: opt-in profiling, service-level and Pod-level targeting, temporary versus continuous enablement, and explicit disable controls.
- Keep production safety explicit: bounded profiling windows, startup delay, skipped unsupported JVMs, and visible status/failure reasons.
- Keep diagnostic scope explicit: CPU, allocation, lock contention, JVM trends, and thread snapshots. Do not imply retained-heap analysis unless the docs are updated to add it.

## When Writing Code Later

- Mirror the documented architecture rather than introducing a new one.
- Keep collection node-local and bounded.
- Keep storage and retention behavior simple enough for single-node ClickHouse operation.
- Make the UI service-centric rather than general-purpose.

## Useful Shortcuts

- Read `README.md` first for the project summary.
- Read `docs/brainstorms/java-k8s-performance-profiling-requirements.md` before making product decisions.
- Read `docs/research/coroot-node-agent-java-agent.md` when reasoning about Coroot or async-profiler behavior.

