# Coroot `java-agent` Research

Date: 2026-05-08
Source: <https://github.com/coroot/coroot-node-agent/tree/main/java-agent>

## Scope

This note covers the `java-agent/` subtree in `coroot/coroot-node-agent`.

## What this subtree does

`java-agent/` is a Java instrumentation agent whose job is to hook JVM TLS I/O and forward payloads into a native bridge.

The mechanism is:

1. A Java agent is attached via `-javaagent` or dynamic attach.
2. `SslTransformer` rewrites selected JDK SSL stream classes using ASM.
3. The rewritten methods call JNI methods in `NativeBridge`.
4. The JNI implementation in `native/coroot_java_tls.c` copies bytes into a thread-local buffer and calls C hooks.

In practice, this subtree is the JVM-side TLS interception layer for Coroot.

## Directory layout

- [`java-agent/pom.xml`](https://raw.githubusercontent.com/coroot/coroot-node-agent/main/java-agent/pom.xml)
- [`java-agent/Dockerfile`](https://raw.githubusercontent.com/coroot/coroot-node-agent/main/java-agent/Dockerfile)
- [`java-agent/Makefile`](https://raw.githubusercontent.com/coroot/coroot-node-agent/main/java-agent/Makefile)
- [`java-agent/src/main/java/io/coroot/agent/TlsAgent.java`](https://raw.githubusercontent.com/coroot/coroot-node-agent/main/java-agent/src/main/java/io/coroot/agent/TlsAgent.java)
- [`java-agent/src/main/java/io/coroot/agent/SslTransformer.java`](https://raw.githubusercontent.com/coroot/coroot-node-agent/main/java-agent/src/main/java/io/coroot/agent/SslTransformer.java)
- [`java-agent/src/main/java/io/coroot/agent/NativeBridge.java`](https://raw.githubusercontent.com/coroot/coroot-node-agent/main/java-agent/src/main/java/io/coroot/agent/NativeBridge.java)
- [`java-agent/native/coroot_java_tls.c`](https://raw.githubusercontent.com/coroot/coroot-node-agent/main/java-agent/native/coroot_java_tls.c)

## Build and packaging

### Maven

The Maven module is a plain JAR project targeting Java 8.

Notable `pom.xml` details:

- `groupId`: `io.coroot`
- `artifactId`: `coroot-java-tls-agent`
- `version`: `0.1.0`
- dependency: ASM `9.7`
- shaded ASM package relocation to `io.coroot.agent.shaded.asm`
- manifest entries for:
  - `Premain-Class`
  - `Agent-Class`
  - `Can-Retransform-Classes`
  - `Can-Redefine-Classes`

### Docker build

The `Dockerfile` builds both the native shared library and the agent JAR inside `debian:bullseye`:

- installs `default-jdk`, `maven`, `gcc`, and `gcc-aarch64-linux-gnu`
- compiles `native/coroot_java_tls.c` into:
  - `libcoroot_java_tls_amd64.so`
  - `libcoroot_java_tls_arm64.so`
- runs `mvn package -q -DskipTests`

### Makefile output

The `Makefile` copies build outputs into `../jvm/assets`:

- `libcoroot_java_tls_amd64.so`
- `libcoroot_java_tls_arm64.so`
- `coroot-java-tls-agent.jar`

That means this subtree is not only a standalone agent, it also feeds artifacts into the sibling `jvm/` component.

## Runtime behavior

### Entry points

`TlsAgent` defines both Java agent entry points:

- `premain(String args, Instrumentation inst)`
- `agentmain(String args, Instrumentation inst)`

Both delegate to `initialize(...)`.

### Initialization flow

`TlsAgent.initialize(...)` does the following:

1. Guards against double initialization with a `volatile` flag and synchronization.
2. Prints an initialization log line.
3. Requires a non-empty native library path argument.
4. Loads the native `.so` through `NativeBridge.load(...)`.
5. Registers `SslTransformer` with retransformation enabled.
6. Scans already loaded classes and retransforms any target SSL stream classes that are modifiable.

### Native bridge

`NativeBridge` contains two JNI methods:

- `tlsWriteEnter(byte[] data, int offset, int length)`
- `tlsReadExit(byte[] data, int offset, int length)`

`load(String path)` wraps `System.load(...)` and treats "already loaded" as success.

### Bytecode transformation

`SslTransformer` targets internal JDK SSL stream classes:

- output path:
  - `sun/security/ssl/SSLSocketImpl$AppOutputStream`
  - `sun/security/ssl/AppOutputStream`
- input path:
  - `sun/security/ssl/SSLSocketImpl$AppInputStream`
  - `sun/security/ssl/AppInputStream`

The transformer injects calls into:

- `write([BII)V` for output streams
- `read([BII)I` for input streams

The injected calls are:

- `NativeBridge.tlsWriteEnter(...)` before write logic
- `NativeBridge.tlsReadExit(...)` after a successful read returns a positive byte count

## Native implementation

`native/coroot_java_tls.c` is a very small JNI shim:

- uses a thread-local buffer sized to `1024` bytes
- copies up to `MAX_PAYLOAD_SIZE` from the Java byte array into the buffer
- calls internal C helpers:
  - `coroot_java_tls_write_enter(...)`
  - `coroot_java_tls_read_exit(...)`

The C helpers currently do not implement visible processing; they only contain a memory barrier and return the input length.

## Key observations

- This subtree is tightly coupled to OpenJDK internals under `sun.security.ssl`.
- It is likely sensitive to JDK implementation changes because it depends on internal class names and method descriptors.
- The Java code is intentionally small; the interesting work is in class retransformation and JNI handoff.
- The native layer currently looks like a placeholder or minimal interception stub rather than a fully featured parser.
- Java 8 compatibility is explicit in the Maven compiler settings.

## Practical summary

If you only need the essence:

- `java-agent/` is Coroot's Java TLS interception agent.
- It instruments JVM SSL stream classes with ASM.
- It bridges intercepted reads and writes into native code.
- Build outputs are exported into `jvm/assets` for the larger Coroot node-agent build.

## Data flow and persistence

The Java agent itself does **not** persist TLS payloads to disk.

What happens instead is:

1. The JNI layer copies bytes into a thread-local native buffer as a short-lived handoff.
2. The node agent loads the Java agent into the target JVM and attaches eBPF uprobes to the exported native symbols.
3. The eBPF side records a small amount of state in kernel maps, then emits an `L7Request` event to userspace.
4. Userspace parses the payload into protocol-specific data and updates in-memory container state.
5. Final persistence happens only at the agent/exporter boundary:
   - metrics are spooled to disk under `WAL_DIR` before remote write
- traces are exported over OTLP to the configured traces endpoint

So the TLS bytes are treated as transient observability input, not as a durable local data store.

## Relation to JVM inspections and async-profiler

The Coroot JVM inspection page uses two different Java-related data sources:

- JVM metrics collected by the node agent
- optional async-profiler data for richer profiling

These are separate from the Java TLS agent itself.

### JVM inspections

The JVM inspection doc says the basic checks and charts come from JVM metrics that the node agent collects automatically:

- JVM availability
- JVM safepoints
- heap size
- GC time
- safepoint time

The same page also notes that when async-profiler is enabled, additional allocation and lock-contention charts appear.

### async-profiler integration

The async-profiler path is implemented in the Go node agent, not in `java-agent/`.

The flow is:

1. `profiling.Init(...)` creates a profiling session when `--profiles-endpoint` is configured.
2. The node agent discovers HotSpot JVMs and, when `--enable-java-async-profiler` is set, calls into `jvm.DeployAndStartAsyncProfiler(...)`.
3. `jvm/tls.go` is unrelated here; async-profiler uses `jvm/async_profiler.go` and the attach API.
4. Every 60 seconds the agent:
   - stops async-profiler
   - reads the generated JFR file
   - parses it into pprof profiles
   - uploads the profiles to Coroot
   - restarts async-profiler quickly, with a small collection gap
5. The same parsed data also updates in-memory container stats, which drive metrics like:
   - `container_jvm_alloc_bytes_total`
   - `container_jvm_alloc_objects_total`
   - `container_jvm_lock_contentions_total`
   - `container_jvm_lock_time_seconds_total`
   - `container_jvm_profiling_status`

### Practical distinction

- `java-agent/` is for TLS interception and L7 payload extraction.
- async-profiler is for CPU, allocation, and lock profiling.
- Both can be enabled for the same JVM, but they solve different observability problems.
- The JVM inspection UI combines them by showing baseline JVM metrics always, and adding profiling charts when async-profiler is enabled.

### Use of `grafana/jfr-parser`

The project uses [`github.com/grafana/jfr-parser`](https://github.com/grafana/jfr-parser), but only in the async-profiler path.

In `coroot-node-agent`, the dependency is declared as `github.com/grafana/jfr-parser v0.15.0`. The direct usage is in `jvm/jfr_parse.go`:

- `parser.NewParser(jfrData, parser.Options{SymbolProcessor: parser.ProcessSymbols})`
- `p.ParseEvent()`
- `p.GetStacktrace(...)`
- `p.GetMethod(...)`
- `p.GetClass(...)`
- `p.GetSymbolString(...)`

The parser reads JFR data generated by async-profiler and extracts these event types:

- `T_EXECUTION_SAMPLE` for Java CPU samples
- `T_ALLOC_SAMPLE`
- `T_ALLOC_IN_NEW_TLAB`
- `T_ALLOC_OUTSIDE_TLAB`
- `T_MONITOR_ENTER` for lock contention

Coroot then converts those JFR events into `github.com/google/pprof/profile` profiles:

- `java:cpu:nanoseconds`
- `java:heap_alloc_objects:count`
- `java:heap_alloc_space:bytes`
- `java:lock_contentions:count`
- `java:lock_delay:nanoseconds`

This means `jfr-parser` is the bridge between async-profiler's JFR output and Coroot's pprof upload pipeline. It is not involved in Java TLS interception.

### Article-backed async-profiler analysis

The article "Profiling Java apps: breaking things to prove it works" matches the implementation in `coroot-node-agent`.

The problem statement is important: Coroot already had eBPF CPU profiling for Java, but eBPF CPU samples alone cannot explain GC pressure or monitor lock contention. async-profiler fills that gap because it can observe JVM-level allocation and lock events through JVMTI.

The implementation follows the same deployment pattern as the Java TLS agent:

- detect Java processes
- confirm HotSpot by scanning `/proc/<pid>/maps`
- copy a native asset into the target container under `/tmp/coroot`
- use the JVM Attach API to load code into the running JVM

But the loaded artifact and data path are different:

- Java TLS loads `coroot-java-tls-agent.jar` plus `libcoroot_java_tls.so`, then eBPF uprobes read TLS plaintext from native symbols.
- Java profiling loads only `libasyncProfiler.so`, starts async-profiler, and later reads finalized JFR files.

The async-profiler command in code is:

```text
start,event=itimer,interval=10ms,alloc,lock,jfr,file=/tmp/coroot/ap_<pid>.jfr
```

That means one recording session captures:

- CPU samples from `itimer`
- allocation samples from `alloc`
- monitor contention samples from `lock`
- output in JFR format

Collection is windowed. Every minute, `profiling.collectAsyncProfilerProfiles()` calls `jvm.CollectAsyncProfiler(...)`, which:

- sends `stop`
- reads `/proc/<pid>/root/tmp/coroot/ap_<ns-pid>.jfr`
- removes the consumed file
- sends a fresh `start,...,jfr,file=...`

This stop/read/start design matters because JFR needs finalized metadata and chunks before `jfr-parser` can parse it reliably. The article says they considered `dump`, but rejected it because incomplete JFR metadata breaks parsers; the current code reflects that decision.

After reading the JFR bytes, Coroot uses `jfr-parser` to resolve stack traces, methods, classes, symbols, and event payloads. Then it builds pprof profiles grouped by stack:

- execution samples become `java:cpu:nanoseconds`
- allocation events become object-count and allocated-byte profiles
- monitor-enter events become lock-contention-count and lock-delay profiles

Those profiles serve two consumers:

- The full pprof profiles are uploaded to the configured profiles endpoint for flamegraphs.
- Aggregated totals are sent through `ProfilingUpdate` and exported as Prometheus counters for the JVM report.

The exported JVM profiling counters are:

- `container_jvm_alloc_bytes_total`
- `container_jvm_alloc_objects_total`
- `container_jvm_lock_contentions_total`
- `container_jvm_lock_time_seconds_total`
- `container_jvm_profiling_status`

The article's demo scenarios map directly to these outputs:

- lock contention makes `container_jvm_lock_contentions_total` and `container_jvm_lock_time_seconds_total` rates spike, and the lock-delay flamegraph points to the contended monitor path
- allocation pressure makes `container_jvm_alloc_bytes_total` and GC time spike together, and the allocation-space flamegraph points to the allocating method

One small naming detail: the actual environment variable in code is `ENABLE_JAVA_ASYNC_PROFILER`, matching the YAML example in the article.

### Production overhead assessment

The async-profiler integration is designed for production use, but it is not zero-cost.

Expected cost profile:

- attach cost is short-lived and paid when a JVM is first instrumented
- continuous CPU sampling uses `itimer` every 10 ms, so the sample rate is about 100 Hz
- allocation and lock profiling add JVM event cost proportional to allocation rate and monitor contention rate
- once per minute, the agent stops async-profiler, reads a finalized JFR file, and starts a new recording
- JFR parsing and pprof conversion run in the node-agent process, not inside the application JVM

The article reports two concrete operational numbers:

- each Attach API command is about 2 ms
- the stop/start collection gap is about 4 ms

The code confirms the design:

- `apStartArgs(...)` uses `event=itimer,interval=10ms,alloc,lock,jfr`
- `CollectAsyncProfiler(...)` sends `stop`, reads `/proc/<pid>/root/tmp/coroot/ap_<pid>.jfr`, removes the file, then sends `start`
- `IsAsyncProfilerAlreadyLoaded(...)` skips JVMs where another async-profiler instance is already present
- `JAVA_ASYNC_PROFILER_DELAY` defaults to 30 seconds, avoiding immediate startup-period instrumentation

Practical risk assessment:

- CPU-only sampling is generally low overhead.
- Allocation profiling is usually acceptable because it is sampled, but very high allocation-rate services can see higher overhead and larger JFR files.
- Lock profiling is usually quiet when there is little contention, but cost rises when many threads frequently block on monitors.
- The node-agent can spend additional CPU parsing JFR and building pprof profiles if many JVMs are enabled on the same node.
- The application will not store raw JFR data long-term; files are read and removed each collection cycle.

For production rollout:

- enable it gradually by node or namespace
- watch request latency, CPU, GC pause time, and node-agent CPU after enabling
- use `JAVA_ASYNC_PROFILER_DELAY` to avoid profiling JVM warmup
- keep `ENABLE_JAVA_ASYNC_PROFILER=false` for very latency-sensitive JVMs until validated under real traffic
- avoid running another async-profiler-based tool in the same JVM; Coroot tries to detect this and skip, but operational ownership should still be clear

### Coroot server-side integration and `stats.go`

On the Coroot server side, Java async-profiler data is split into three paths:

1. profile storage and flamegraph query
2. JVM inspection metrics
3. anonymous usage statistics

`stats/stats.go` only participates in the third path.

#### 1. Profile storage and flamegraphs

The node agent uploads parsed pprof profiles to Coroot's `/v1/profiles` endpoint. Coroot parses the pprof payload and batches rows into ClickHouse:

- `collector/profiles.go` accepts the profile upload
- each pprof sample type becomes a profile type row
- stacks are stored separately from samples by stack hash
- Java async-profiler profile types are registered in `model/profile.go`

The Java profile types known to Coroot are:

- `java:cpu:nanoseconds`
- `java:heap_alloc_objects:count`
- `java:heap_alloc_space:bytes`
- `java:lock_contentions:count`
- `java:lock_delay:nanoseconds`

The profiling view asks ClickHouse for available profile types and then queries stack samples to build a flamegraph.

#### 2. JVM inspection metrics

The node agent also exports Prometheus metrics derived from parsed async-profiler profiles:

- `container_jvm_alloc_bytes_total`
- `container_jvm_alloc_objects_total`
- `container_jvm_lock_contentions_total`
- `container_jvm_lock_time_seconds_total`
- `container_jvm_profiling_status`

In Coroot, `constructor/queries.go` fetches these metrics, using `rate(...)` for the monotonic counters. Then `constructor/jvm.go` stores them in `model.Jvm`:

- `AllocBytes`
- `AllocObjects`
- `LockContentions`
- `LockTime`
- `ProfilingEnabled`

The JVM inspection report uses those fields to add allocation and lock-contention charts. If `ProfilingEnabled` is false for all JVMs in the app, the auditor shows a configuration hint telling the user to enable async-profiler.

#### 3. Anonymous usage statistics

`stats/stats.go` has a `Stats.Integration.JavaAsyncProfiler` boolean field serialized as `java_async_profiler`.

It is set by loading the current world model and scanning applications:

```go
for _, j := range i.Jvms {
    if j.ProfilingEnabled {
        stats.Integration.JavaAsyncProfiler = true
        break
    }
}
```

This means `stats.go` does not ingest JFR, parse profiles, store samples, or render charts. It only reports whether the Coroot instance has at least one JVM where async-profiler is active according to `container_jvm_profiling_status`.

#### End-to-end server-side chain

The full Coroot integration is:

```text
node-agent async-profiler
  -> JFR
  -> jfr-parser
  -> pprof profiles
  -> /v1/profiles
  -> ClickHouse profiling_* tables
  -> Profiling API
  -> flamegraph UI

node-agent async-profiler
  -> aggregated profiling counters
  -> Prometheus metrics
  -> constructor/jvm.go
  -> model.Jvm
  -> JVM inspection charts
  -> stats.go java_async_profiler usage flag
```

### JVM auditor integration

`coroot/auditor/jvm.go` is the report-building layer for the JVM inspection page. It does not collect metrics and does not parse profiles. It consumes `model.Jvm`, which has already been populated by the constructor from Prometheus metrics.

The auditor creates the JVM report only for applications detected as JVM applications:

```go
if !a.app.IsJvm() {
    return
}
```

It creates two checks:

- `JvmAvailability`
- `JvmSafepointTime`

The availability check is based on whether `j.HeapUsed.Last() > 0`. The safepoint check compares `j.SafepointTime.Last()` against the check threshold, whose default is `0.05` seconds/second.

The report widgets are:

- JVM instance table with status and Java version
- heap size charts
- GC time charts
- safepoint time chart
- allocation rate chart group
- lock contention chart group

The async-profiler-specific widgets are the last two:

- `Allocation rate <selector>`
  - `bytes/second` from `j.AllocBytes`
  - `objects/second` from `j.AllocObjects`
- `Lock contention <selector>`
  - `contentions/second` from `j.LockContentions`
  - `delay, seconds/second` from `j.LockTime`

These series only become meaningful when async-profiler metrics exist. The auditor always creates the chart groups in detailed reports, but the data comes from `container_jvm_alloc_*`, `container_jvm_lock_*`, and `container_jvm_profiling_status`.

The auditor also wires the charts to the Profiling report:

- allocation charts link to `AuditReportProfiling` with `query=memory`
- lock charts link to `AuditReportProfiling` with `query=lock`

That is how the UI supports the workflow described in the article:

```text
JVM report chart spike
  -> click chart profile/drill-down
  -> Profiling report opens memory or lock category
  -> featured Java profile type is selected
  -> ClickHouse flamegraph is rendered
```

If no JVM in the application has `ProfilingEnabled=true`, `auditor/jvm.go` adds a configuration hint:

```text
Enable async-profiler to get Java CPU, memory allocation, and lock contention profiles and metrics.
```

So the auditor is the UX bridge between metrics and profiles. It shows the time-series symptom in the JVM report and points the user to the pprof flamegraph that explains the code path.
