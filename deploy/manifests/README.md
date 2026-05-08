# Kubernetes Manifests

The Helm chart under `deploy/helm` is the source of truth for installation.

Default behavior:

- profiling is disabled unless workload metadata opts in
- collector runs as a node-local DaemonSet
- backend and web are deployed as separate services
- exporter metrics are exposed for Prometheus scraping
- ClickHouse retention remains at or below seven days

Required operator inputs:

- collector and UI auth secret
- TLS secret or cluster service mesh policy
- pinned async-profiler and thread-helper artifact checksums
- ClickHouse DSN for the existing single-node deployment
