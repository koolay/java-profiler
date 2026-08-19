# Kubernetes Installation Manifests

The Helm chart in `deploy/helm` is the authoritative installation definition.

By default:

- profiling is disabled unless workload metadata opts in
- collector runs as a node-local DaemonSet
- backend and web are deployed as separate services
- exporter metrics are exposed for Prometheus scraping
- ClickHouse retention remains at or below seven days

Before installing, provide:

- collector and UI auth secret
- TLS secret or cluster service mesh policy
- pinned async-profiler and thread-helper artifact checksums
- ClickHouse DSN for the existing single-node deployment
