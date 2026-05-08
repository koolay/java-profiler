---
title: "fix: address review findings in java profiler vertical slice"
type: fix
status: active
date: 2026-05-08
origin: commit:2a2f240
---

# fix: address review findings in java profiler vertical slice

## Summary

Tighten the first implementation slice so it matches the documented product boundary and does not silently degrade into a partial demo. The fix set focuses on three areas:

- wire the collector executable into the actual discovery, profiling, and upload flow instead of only exposing metrics
- fail fast when ClickHouse initialization or schema application cannot complete
- use a stable JVM/process start time in target identity and align the UI/query path with the intended time-range behavior

---

## Problem Frame

The feature commit introduces most of the domain model, repositories, and UI surfaces, but three behaviors are not production-safe yet:

1. the collector binary does not execute the collector pipeline that the rest of the codebase defines
2. backend startup can fall back to an in-memory repository or continue with a partially initialized SQL repository when ClickHouse setup fails
3. process identity and UI query expectations do not yet line up with stable JVM start times and explicit time-range selection
4. ingestion idempotency can mark a failed write as accepted, making a later retry skip missing samples
5. deployed service wiring is inconsistent: collector uses HTTPS against an HTTP backend, and UI calls need a viable authenticated path
6. temporary profiling can be enabled without a bounded duration, which violates the v1 safety contract

These issues do not affect the happy-path tests, but they do affect whether the system can operate correctly in a real cluster.

---

## Fix Plan

### F1. Wire the collector executable into the actual collection pipeline

Goal: make `cmd/collector` do real work, not just serve `/metrics`.

Work items:

- create a collector runtime entrypoint that owns the discovery loop, policy evaluation, process scanning, HotSpot detection, attach plan, JFR parsing, thread snapshot capture, batching, and backend upload
- connect the existing `collector/internal/*` packages into that runtime instead of leaving them as unused scaffolding
- preserve the metrics endpoint, but expose collection health and failure counters from the running pipeline
- add a focused test that proves the collector runtime instantiates the pipeline components and emits status/health output for at least one discovered target

Acceptance criteria:

- the collector binary starts the collection pipeline
- discovered JVMs produce a target status and can be batched for upload
- metrics still serve from the collector process

### F2. Fail fast on ClickHouse initialization errors

Goal: avoid silent fallback to in-memory storage or partially initialized SQL access.

Work items:

- change backend startup so ClickHouse connection or schema application errors are surfaced instead of being ignored
- remove the implicit in-memory fallback for production startup paths, or gate it behind an explicit test-only configuration
- add a startup test that proves an invalid ClickHouse DSN or a schema application failure does not leave the backend serving a deceptively healthy API
- expose a degraded startup state if the service is ever allowed to run without a real ClickHouse backend

Acceptance criteria:

- invalid ClickHouse configuration causes a visible startup failure or an explicit degraded state
- schema application failures are not swallowed
- the backend does not silently accept data into an ephemeral in-memory repository in production mode
- schema application works from the compiled container image, not only from a source checkout
- profile batch idempotency does not convert retryable storage failure into a permanent duplicate skip

### F3. Stabilize JVM identity and align query/time-range behavior

Goal: make process identity deterministic and match the UI to the intended query contract.

Work items:

- replace the synthetic process start time in `collector/internal/discovery/process_scanner.go` with a real process start timestamp derived from procfs or another stable source
- verify that `domain.TargetIdentity.Key()` continues to distinguish pid reuse correctly
- add or update UI controls so the time-range shown to the user is actually passed into the query parameters, or remove the misleading static label until the control exists
- add a backend test that confirms empty time parameters and explicit time parameters are treated intentionally, not by accident

Acceptance criteria:

- target identity changes when a JVM restarts, even if the pid is reused
- the UI does not advertise a time range that it is not sending
- flamegraph and diagnosis queries can be reasoned about using the same window that the UI shows

### F4. Reconcile deployment and auth wiring

Goal: make the shipped Helm slice internally consistent.

Work items:

- align the collector backend URL scheme with the backend listener actually shipped by the chart
- keep backend stack-data endpoints authenticated by default, while allowing browser calls through a server-set UI token cookie or equivalent proxy/session path
- document any remaining UI auth proxy requirement as residual work if a full login flow is outside this fix slice

Acceptance criteria:

- collector-to-backend traffic does not fail TLS handshake by default
- UI API calls have a viable authenticated same-origin/proxy path
- no static UI bearer token is exposed in browser JavaScript

### F5. Enforce bounded temporary profiling and query windows

Goal: keep production safety and query semantics aligned with requirements.

Work items:

- reject temporary profiling metadata unless it includes a positive duration
- carry collection windows into normalized profile samples and reject samples missing time bounds at ingest
- ensure ClickHouse profile queries apply the requested time range, matching in-memory query behavior

Acceptance criteria:

- temporary profiling without duration fails closed
- stored profile samples have non-zero started/ended timestamps
- ClickHouse flamegraph queries cannot return samples outside the selected overlap window

---

## Verification

- Run the existing Go test suite after the wiring and startup changes land.
- Add one targeted test for each fix area so regressions are caught locally.
- Keep the review-specific behavior changes documented in `README.md` and `docs/index.md` if the visible workflow changes.

---

## Follow-Up

- If any of the acceptance criteria force a behavior choice, prefer the safer failure mode over silent degradation.
- Keep the plan narrow: do not expand into new profiling types, new backends, or broader observability features while fixing these review items.
