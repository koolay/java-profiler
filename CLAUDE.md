# Claude Instructions for `java-profiler`

## What this repository is

This repository is a documentation-first workspace for a Java performance profiling system on Kubernetes.

Treat the requirements document as the source of truth:

- `docs/brainstorms/java-profiler-requirements.md`

Use the research note for implementation context:

- `docs/research/coroot-node-agent-java-agent.md`

## Ground Rules

- Keep changes grounded in the existing docs.
- Do not invent a broader product scope than the requirements support.
- Keep the first version focused on Java, Kubernetes, HotSpot, async-profiler, ClickHouse, and a narrow UI.
- Preserve the retention ceiling: no collected data should be retained for more than 7 days.
- Avoid introducing AGPL-dependent profile backends such as Pyroscope, Parca, or Grafana as core dependencies.

## Research Source Rules

- For any web research, technology selection, dependency evaluation, architecture comparison, or implementation reference, search and cite only English-language international sources by default.
- Prefer primary sources: official documentation, official GitHub repositories, standards documents, release notes, license files, and reputable international engineering writeups.
- Do not use Chinese-language community sources, Chinese blogs, Chinese forums, Zhihu, Juejin, CSDN, SegmentFault, WeChat articles, Gitee mirrors, or translated summaries as research evidence unless the user explicitly asks for Chinese-community context.
- If Chinese-community context is explicitly requested, clearly separate it from the main evidence and do not use it to override primary international sources.
- When using search queries, write them in English unless the user explicitly asks otherwise.

## When Writing or Editing Docs

- Prefer concrete, testable language over marketing language.
- Keep Kubernetes control behavior explicit: opt-in profiling, service-level and Pod-level targeting, temporary versus continuous enablement, and explicit disable controls.
- Keep production safety explicit: bounded profiling windows, startup delay, skipped unsupported JVMs, and visible status/failure reasons.
- Keep diagnostic scope explicit: CPU profiles, allocation profiles, lock profiles, and thread snapshots. Metrics are exporter-only in this project; Prometheus-series services own metric storage, dashboards, alerting, and retention. Do not imply retained-heap analysis unless the docs are updated to add it.

## When Writing Code Later

- Mirror the documented architecture rather than introducing a new one.
- Keep collection node-local and bounded.
- Keep storage and retention behavior simple enough for single-node ClickHouse operation.
- Make the UI service-centric rather than general-purpose.

## Useful Shortcuts

- Read `README.md` first for the project summary.
- Read `docs/brainstorms/java-profiler-requirements.md` before making product decisions.
- Read `docs/research/coroot-node-agent-java-agent.md` when reasoning about Coroot or async-profiler behavior.

## Design System

Always read `DESIGN.md` before making any visual or UI decisions.
All font choices, colors, spacing, layout density, and aesthetic direction are defined there.
Do not deviate without explicit user approval.
In QA or review mode, flag UI code that does not match `DESIGN.md`.
