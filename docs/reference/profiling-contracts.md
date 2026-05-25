# Profiling Contracts

The stable profiling payload and configuration contracts live in the repository source tree under `contracts/profiling`.

Use these files as the source of truth when changing collector payloads, backend ingestion, or UI interpretation:

- [`configuration.md`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/configuration.md)
- [`payloads.md`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/payloads.md)
- [`types.go`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/types.go)

Changes to these contracts should be reflected in the requirements, operations guides, and real profiling acceptance standard when they affect scope, retention, collection, storage, or user-visible behavior.

## Current UI Query Contracts

The web UI currently consumes product-shaped backend routes under `/api/ui/v1`:

- `/flamegraph`: flamegraph tree, partial metadata, and profile value semantics.
- `/top-stacks`: ranked Self/Total rows for Top Table workflows.
- `/allocation-summary`: sampled allocation summary for `java_allocation_bytes`, including requested/effective scope, coverage, top allocating paths, top self allocating frames, insights, limitations, partial reasons, and empty-state reason.
- `/service-summary` and `/service-selectors`: service and target selectors.
- `/target-status`: JVM eligibility and collection status evidence.
- `/ingestion`: aggregate profile batch acceptance, retry, rejection, drop, and truncation evidence.
- `/jvm-events`, `/thread-diagnosis`, and `/deadlocks`: GC, thread, and deadlock diagnosis evidence.

Empty profile states are user-visible contract behavior. The UI distinguishes disabled profiling, expired temporary windows, unmatched targets, ingestion gaps, query errors, and no samples in the selected range. These states help diagnosis, but they do not replace strict acceptance requirements for non-empty CPU, Wall Clock, allocation, I/O, and lock profile data.
