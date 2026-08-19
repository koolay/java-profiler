# Java Profiler Deployment and Operations

This page is the English entry point for platform administrators, SREs, and security administrators. The [Chinese operational manual](/zh/operations/deployment-operations-admin-manual) contains the longer installation and troubleshooting procedures.

## Product boundary

The first version targets HotSpot-compatible Java services on Kubernetes:

- The collector runs as a node-local DaemonSet.
- Profiling is opt-in through Kubernetes annotations or labels.
- CPU, allocation, and lock profiling use async-profiler.
- Profiles, thread diagnostics, target status, and ingestion evidence are stored in ClickHouse.
- Collected data is retained for no more than seven days; optional raw artifacts are limited to 24 hours.
- Prometheus-compatible metrics are exposed by collector/backend exporters; this product does not store Prometheus time series.

The platform does not expand into general logging, tracing, service maps, non-Java profiling, or a long-term metrics platform.

## Administrator responsibilities

Administrators own installation, configuration, permissions, upgrades, rollback, ClickHouse retention, Web-to-backend routing, collector discovery, and operational health. Service owners own the interpretation of CPU, allocation, lock, deadlock, and thread evidence.

Use the [Profiling Contracts](../reference/profiling-contracts) for field names and status values. Use the [Real Profiling Acceptance Standard](./real-profiling-acceptance-standard) when a release changes the profiling path.

## Required operational checks

Before handing a deployment to users, verify:

1. Collector, backend, Web, and ClickHouse are healthy.
2. Kubernetes RBAC allows node-local JVM discovery and attach operations.
3. The Web `/api/*` proxy reaches the backend rather than the static asset server.
4. A target workload is accepted only when its metadata enables profiling.
5. ClickHouse TTLs are bounded and cleanup health is observable.
6. Real acceptance produces non-empty profile data, ingestion evidence, browser evidence, and no unexpected workload restart.

## Release and troubleshooting

The release scripts build and publish versioned artifacts. Treat a release as ready only after the deployment, backend, ClickHouse, collector, and UI checks pass. For detailed Helm, Secret, RBAC, proxy, retention, upgrade, rollback, and failure-case procedures, use the [full Chinese operational manual](/zh/operations/deployment-operations-admin-manual).
