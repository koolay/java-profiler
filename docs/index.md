# Directory Index

## Files

- **[index.md](./index.md)** - Documentation directory index

## Subdirectories

### architecture/

- **[java-profiler-architecture.md](./architecture/java-profiler-architecture.md)** - Java profiling system architecture

### brainstorms/

- **[java-profiler-requirements.md](./brainstorms/java-profiler-requirements.md)** - Java profiling product requirements

### plans/

- **[2026-05-08-001-feat-java-profiler-implementation-plan.md](./plans/2026-05-08-001-feat-java-profiler-implementation-plan.md)** - Java profiling implementation plan

### operations/

- **[java-profiling-runbook.md](./operations/java-profiling-runbook.md)** - Operator enablement, status, retention, and troubleshooting runbook

### research/

- **[coroot-node-agent-java-agent.md](./research/coroot-node-agent-java-agent.md)** - Coroot Java agent research

### implementation/

- `cmd/backend` - backend entrypoint scaffold
- `cmd/collector` - collector entrypoint scaffold
- `backend/internal` - backend domain, ClickHouse, HTTP, and metrics packages
- `collector/internal` - collector policy, discovery, attach, and pipeline packages
- `contracts/profiling` - stable payload and configuration contracts
- `java-helper/thread-diagnostics` - Java helper for thread snapshots and deadlocks
- `web` - React/Vite UI
- `deploy` - Helm and manifest packaging
