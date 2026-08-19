# End-to-End Automation Guide

This page is the English entry point for automated UI and Kubernetes acceptance. The detailed commands, workload examples, browser flow, and failure diagnosis are in the [Chinese E2E guide](/zh/operations/e2e-automation-test-guide).

## What the end-to-end run checks

The run must exercise the real Java profiling path, not only load a healthy page:

- Kubernetes metadata accepts the target workload;
- CPU, allocation, and lock-delay profiles contain data;
- ClickHouse contains profile rows;
- ingestion reports accepted batches and any rejection, retry, drop, or truncation;
- the browser can complete the analysis workflow;
- the target workload has no unexpected additional restart.

For changes touching collector profiling, ingestion, ClickHouse, deployment, the demo workload, or the profile UI, follow the [Real Profiling Acceptance Standard](./real-profiling-acceptance-standard) and [Contributing](../contributing/development).

## Test layers

- UI smoke tests validate page loading and basic interactions.
- The real acceptance script validates collector, backend, ClickHouse, Web, and the JDK 17 workload together.
- Strict profiling acceptance requires target status, profile rows, ingestion evidence, browser evidence, bounded retention, and the restart invariant.

UI smoke tests alone cannot prove async-profiler attach or ClickHouse ingestion.

## Detailed guide

Use the [Chinese E2E automation guide](/zh/operations/e2e-automation-test-guide) for the current commands, Kubernetes examples, Playwright workflow, artifact layout, and failure matrix.
