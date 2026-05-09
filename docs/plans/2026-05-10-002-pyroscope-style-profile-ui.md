# Pyroscope-style Profile UI

## Goal

Replace the demo-specific source-context experiment with a generic profiling analysis UI modeled after Grafana Pyroscope's table and flame graph workflow.

## Scope

- Remove source-context backend API, Helm source mounts, and frontend source fetch code.
- Make the CPU view present `Top Table`, `Flame Graph`, and `Both` display modes.
- In the table, show `Symbol`, `Self`, and `Total` metrics.
- Use the table to select/search a symbol in the flame graph without implying source-code lookup.
- Keep the existing Kubernetes/JDK17 real acceptance flow.

## Non-goals

- Do not implement source JAR, GitHub, SCM, or artifact source integration.
- Do not introduce Pyroscope as a backend dependency.
- Do not change profile ingestion or ClickHouse schema in this iteration.

## Verification

- Go HTTP API tests pass after source-context removal.
- Frontend unit tests cover self/total aggregation and table/flame graph modes.
- Web production build succeeds.
- Real Playwright acceptance passes against `java-profiler-qa/jdk17-http-demo` through `http://127.0.0.1:18181`.
