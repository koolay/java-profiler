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
- When validating an existing non-demo workload, set `JAVA_PROFILER_ACCEPTANCE_LOAD_PATHS` to one or more comma-separated HTTP paths that actually exist on that service, such as `/profile-load/`, so the acceptance script can drive real CPU/allocation/lock load without relying on the JDK demo endpoints.
- Prefer approved base images from the project/user-provided mirror list. For the real acceptance collector image, the default runtime base image is `ghcr.io/koolay/library/alpine:3.18.0`.

Recommended current-workspace setup:

```bash
export BACKEND_IMAGE=java-profiler-backend:qa-$(date +%Y%m%d%H%M%S)
export COLLECTOR_IMAGE=java-profiler-collector:qa-$(date +%Y%m%d%H%M%S)
export WEB_IMAGE=java-profiler-web:qa-$(date +%Y%m%d%H%M%S)

bash scripts/build-real-acceptance-images.sh

scripts/real-acceptance.sh \
  --service jdk17-http-demo \
  --configure-profiler \
  --require-full-profiling \
  --high-volume \
  --artifact-dir /tmp/java-profiler-real-acceptance-$(date +%Y%m%d%H%M%S)
```

## Required Data Evidence

Every full acceptance run must collect and verify all of the following from the current run window:

- target status contains at least one `accepted` row for the Java target
- CPU flamegraph has a non-zero root value
- Wall Clock flamegraph has a non-zero root value for the Java target
- I/O wait evidence has a non-zero root value when the demo I/O workload is enabled
- allocation flamegraph has a non-zero root value
- lock-delay flamegraph has a non-zero root value
- GC pause evidence returns at least one JVM `gc_pause` event when the demo GC workload is enabled
- ClickHouse contains profile samples and profile stacks for the target
- ClickHouse contains JVM event rows for GC evidence when GC acceptance is in scope
- backend ingestion UI API returns successfully
- profile sample TTL remains bounded to 7 days
- target workload restart count does not increase during acceptance

Thread snapshots and deadlock events are useful evidence, but they are optional unless the change explicitly targets those features. If absent, record them as gaps, not as proof of failure for unrelated profile changes.

## Required Workload Behavior

The demo workload must be driven during the async-profiler window, not before or after it.

- CPU load must execute while profiling is active.
- Wall Clock load must execute while profiling is active and must include blocked or waiting Java threads.
- I/O load must execute while profiling is active through Java socket/file/nio paths; node-level non-Java I/O is not valid evidence.
- Allocation load must execute while profiling is active.
- GC load must create allocation pressure sufficient to produce at least one JVM GC pause event in the selected window.
- Lock profiling must create real contention with concurrent lock requests. A single `synchronized` request is not enough because it can complete without blocking another Java thread.
- If a previous run loaded `libasyncProfiler.so` into the target JVM and the collector was restarted, restart the demo pod before a fresh strict run to avoid stale profiler-conflict state.

## Required UI Evidence

The profile UI must be validated with real backend data, not mocked data, and must support the core performance-analysis workflow:

- service/namespace defaults and filters select the Java demo target
- Status view shows target evidence
- CPU view supports Top Table, Flame Graph, and Both modes
- Wall Clock view returns profile evidence without replacing CPU evidence
- I/O view returns Java-owned blocking evidence or a clear no-evidence state
- GC view returns JVM event evidence and allocation correlation for the same target/time filters
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
- explicitly clear stale disable metadata with `java-profiler.io/profile-disabled: "false"` when configuring a target
- force a fresh workload rollout for acceptance by adding run-specific metadata such as `java-profiler.io/acceptance-run`
- drive CPU, Wall Clock, I/O, allocation, GC, and concurrent lock load for the full profiling wait window
- keep no-Service synthetic workload runs alive until the full profiling wait window has elapsed
- fail when CPU, Wall Clock, allocation, or lock-delay profile data is empty
- fail when required GC event evidence is empty after the demo GC workload ran
- fail when required Java I/O wait evidence is empty after the demo I/O workload ran
- support `--high-volume` for ingestion hardening changes; this mode must extend the profiling window, increase CPU/allocation/lock load parallelism, and verify bounded profile batch metadata
- fail high-volume acceptance when profile batches are rejected, when a profile batch exceeds the collector batch limit, or when ClickHouse restarts/OOMKills during the run
- verify collector/backend profile payload compatibility in tests; profile batch JSON must match the backend contract
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
- CPU, Wall Clock, allocation, or lock-delay flamegraph root value is zero
- GC pause event evidence is empty when GC acceptance is in scope
- Java I/O evidence is empty when I/O acceptance is in scope
- backend rejects profile payloads because batch size is too large
- backend rejects profile payloads because required fields such as `batch_id` or `collector_id` are missing; this means collector/backend payload versions or JSON tags do not match
- target status is `disabled_by_metadata` because a stale truthy `profile-disabled` annotation was left on the Pod template
- target status is `profiler_conflict` after a previous run; roll the target Pod before retrying strict acceptance
- no-Service synthetic workload exits before the profiling window has elapsed
- ClickHouse OOMs under the real acceptance workload
- high-volume ingestion has no accepted profile batches
- collector/backend ingestion metadata hides dropped, truncated, split, or rejected profile batches
- UI tests pass with mocked data but fail against real backend data
- search, focus, Back, Reset, or view-mode interactions do not affect the real UI state

Do not hide these as "environment issues" until the root cause is proven. Fix the product, script, workload, or deployment and rerun until the standard passes.
