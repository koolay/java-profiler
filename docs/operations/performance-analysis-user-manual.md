# Performance Analysis Manual

Use this workflow when a Java service is already being profiled and you need to find the stack responsible for CPU, Wall Clock latency, Java I/O wait, GC pauses, allocation, lock, deadlock, or ingestion problems.

The screenshots below come from a real Kubernetes acceptance environment, not mocked UI state.

## Start with target status

Target status tells you whether the selected JVM was accepted, rejected, disabled, unsupported, expired, or failed attach. Check this before assuming a service has no performance data.

![Real target status evidence](../assets/screenshots/real-target-status.png)

## Analyze CPU profiles

Use the CPU view when a service has high CPU or latency that may be caused by expensive Java code.

- Top Table shows the most expensive Java symbols.
- Self CPU is time spent directly in that frame.
- Total CPU includes callees under that frame.
- Flame Graph shows sampled stack context.
- Search highlights matching frames without hiding the rest of the stack.
- Focus narrows the graph to the selected stack path.

![Real CPU profile analysis](../assets/screenshots/real-cpu-analysis.png)

## Analyze Wall Clock latency

Use Wall Clock when latency is high but CPU does not explain the full incident. It shows Java stack time that may be runnable, blocked, waiting, sleeping, or doing I/O.

- Compare Wall Clock with CPU before declaring a method CPU-bound.
- Use the same Top Table and Flame Graph controls as CPU.
- Treat Wall Clock as latency evidence, not as CPU utilization.

![Real Wall Clock latency evidence](../assets/screenshots/real-wall-clock.png)

## Analyze Java I/O wait

Use I/O wait when a service is blocked on sockets, files, or Java-owned network clients. The view only claims Java ownership when the collected stack preserves JVM/JFR evidence for the blocking path.

- Start with top Java I/O symbols.
- Drill into Flame Graph context to separate business code from runtime or native frames.
- Cross-check remote dependency latency outside this profiler when the stack points to RPC or storage clients.

![Real Java I/O wait evidence](../assets/screenshots/real-io-wait.png)

## Correlate GC pauses and allocation pressure

Use GC pauses when request latency or throughput changes line up with JVM pause activity. The GC view shows JVM GC event evidence beside allocation profile context for the same service, Pod, and time range.

- Confirm that GC events exist for the selected time window.
- Use allocation correlation to find Java paths creating pressure.
- Do not treat allocation profile size as retained heap ownership.

![Real GC pause and allocation correlation](../assets/screenshots/real-gc-pauses.png)

## Analyze allocation pressure

Use allocation profiles when heap usage, allocation rate, or GC pressure rises. Start with the largest allocation symbols, then inspect the stack paths that lead to those allocations.

## Analyze lock contention

Use lock diagnosis when request latency or thread state suggests blocking. Lock-delay profiles point to synchronized or monitor paths where threads spend time waiting.

## Check deadlocks

Deadlock diagnosis shows cycles reported by the target JVM. A real run may show cycle evidence or a verified empty state for the selected time range.

![Real deadlock diagnosis surface](../assets/screenshots/real-deadlocks.png)

## Check ingestion health

Ingestion health closes the loop. It shows whether profile batches were accepted, rejected, dropped, or truncated by the backend ingestion path.

![Real ingestion health evidence](../assets/screenshots/real-ingestion-health.png)

## When the UI is empty

Check these in order:

1. Target status: is the JVM accepted?
2. Service and namespace: do they match the workload metadata?
3. Time range: does it include the profiling window?
4. Ingestion health: were profile batches accepted?
5. Runbook: follow [Enable Profiling](./java-profiling-runbook.md) to verify metadata and collector behavior.
