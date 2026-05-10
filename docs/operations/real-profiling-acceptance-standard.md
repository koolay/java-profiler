# Real Profiling Acceptance Standard

This standard is mandatory for changes that affect collector profiling, ingestion, ClickHouse storage, the backend query API, the JDK17 demo service, Kubernetes deployment, or the profile analysis UI.

## Non-Negotiable Goal

Acceptance must prove that a user can locate a real Java performance bottleneck from real profile data. A run is not acceptable just because pods are running, requests return 200, or the UI does not crash.

## Required Environment

- Run against a real Kubernetes cluster with:

  ```bash
  export KUBECONFIG=$HOME/backup/localk8s.yaml
  ```

- Build and deploy the latest local code before acceptance.
- Use the JDK HTTP demo service as the profiling target unless a task explicitly names another Java service.
- The deployed backend, collector, and web pods must run the image tags built from the current workspace.
- Prefer approved base images from the project/user-provided mirror list. For the real acceptance collector image, the default runtime base image is `ghcr.io/koolay/library/alpine:3.18.0`.

## Required Data Evidence

Every full acceptance run must collect and verify all of the following from the current run window:

- target status contains at least one `accepted` row for the Java target
- CPU flamegraph has a non-zero root value
- allocation flamegraph has a non-zero root value
- lock-delay flamegraph has a non-zero root value
- ClickHouse contains profile samples and profile stacks for the target
- backend ingestion UI API returns successfully
- profile sample TTL remains bounded to 7 days
- target workload restart count does not increase during acceptance

Thread snapshots and deadlock events are useful evidence, but they are optional unless the change explicitly targets those features. If absent, record them as gaps, not as proof of failure for unrelated profile changes.

## Required Workload Behavior

The demo workload must be driven during the async-profiler window, not before or after it.

- CPU load must execute while profiling is active.
- Allocation load must execute while profiling is active.
- Lock profiling must create real contention with concurrent lock requests. A single `synchronized` request is not enough because it can complete without blocking another Java thread.
- If a previous run loaded `libasyncProfiler.so` into the target JVM and the collector was restarted, restart the demo pod before a fresh strict run to avoid stale profiler-conflict state.

## Required UI Evidence

The profile UI must be validated with real backend data, not mocked data, and must support the core performance-analysis workflow:

- service/namespace defaults and filters select the Java demo target
- Status view shows target evidence
- CPU view supports Top Table, Flame Graph, and Both modes
- Top Table ranks application Java symbols with Self and Total CPU semantics
- Flame Graph shows full sampled stack context, not Java source call order
- Search changes flamegraph highlighting/dimming
- selecting a frame updates the selected-frame inspector
- focusing a selected frame works and Back returns to the previous root
- Reset clears search/focus state
- Ingestion view shows accepted ingestion evidence

The UI can include native/JVM frames in the flamegraph, but it must make their meaning clear. Runtime/native frames are evidence about where samples landed; application Java rows are the nearest actionable ownership signal.

## Required Automation

Use `scripts/real-acceptance.sh --require-full-profiling` for strict acceptance. The script must:

- wait for target status rows instead of assuming they exist immediately after rollout or table truncation
- drive CPU, allocation, and concurrent lock load for the full profiling wait window
- fail when CPU, allocation, or lock-delay profile data is empty
- run Playwright UI acceptance unless `--skip-browser` is explicitly justified
- write evidence under `/tmp/java-profiler-real-acceptance-*`

Run these checks before claiming completion:

```bash
go test ./collector/internal/profiler ./collector/runtime ./collector/internal/jfr ./backend/internal/app ./backend/internal/httpapi ./backend/internal/clickhouse
cd web && npm test -- --run src/features/cpu/hot-code-view.test.tsx src/visualization/flamegraph.test.tsx
bash -n scripts/real-acceptance.sh scripts/build-real-acceptance-images.sh
shellcheck scripts/real-acceptance.sh scripts/build-real-acceptance-images.sh
git diff --check
```

## Failure Interpretation

Treat these as acceptance blockers:

- no accepted target status for the current run window
- CPU, allocation, or lock-delay flamegraph root value is zero
- backend rejects profile payloads because batch size is too large
- ClickHouse OOMs under the real acceptance workload
- UI tests pass with mocked data but fail against real backend data
- search, focus, Back, Reset, or view-mode interactions do not affect the real UI state

Do not hide these as "environment issues" until the root cause is proven. Fix the product, script, workload, or deployment and rerun until the standard passes.
