# Profiling Contracts

The stable profiling payload and configuration contracts live in the repository source tree under `contracts/profiling`.

Use these files as the source of truth when changing collector payloads, backend ingestion, or UI interpretation:

- [`configuration.md`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/configuration.md)
- [`payloads.md`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/payloads.md)
- [`types.go`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/types.go)

Changes to these contracts should be reflected in the requirements, operations guides, and real profiling acceptance standard when they affect scope, retention, collection, storage, or user-visible behavior.
