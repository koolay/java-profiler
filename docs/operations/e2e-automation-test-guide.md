# End-to-End Automation Guide

This page is the English entry point for automated UI and Kubernetes acceptance. The detailed execution guide, workload examples, evidence checklist, and failure diagnosis are maintained in Chinese at [E2E 自动化测试说明](/zh/operations/e2e-automation-test-guide).

## What E2E acceptance must prove

The test must exercise the real Java profiling path, not only a healthy UI:

- the target workload is accepted through Kubernetes metadata;
- CPU, allocation, and lock-delay profile data is non-empty;
- ClickHouse contains profile rows;
- ingestion reports accepted batches and any rejection, retry, drop, or truncation;
- the browser can perform the real analysis workflow;
- the target workload does not gain an unexpected restart.

For changes touching collector profiling, ingestion, ClickHouse, deployment, the demo workload, or the profile UI, follow the [Real Profiling Acceptance Standard](./real-profiling-acceptance-standard) and [Contributing](../contributing/development).

## Test layers

- UI smoke tests validate page loading and basic interactions.
- The real acceptance script validates collector, backend, ClickHouse, Web, and the JDK 17 workload together.
- Strict profiling acceptance requires target status, profile rows, ingestion evidence, browser evidence, bounded retention, and the restart invariant.

UI smoke alone cannot prove async-profiler attach or the ClickHouse ingestion path.

## Detailed guide

Use the [Chinese E2E automation guide](/zh/operations/e2e-automation-test-guide) for the current commands, Kubernetes examples, Playwright workflow, artifact layout, and failure matrix.
