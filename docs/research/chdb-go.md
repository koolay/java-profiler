# chDB Go Binding Research

date: 2026-05-08
topic: chdb-go

## Sources

- [chDB overview](https://clickhouse.com/chdb)
- [chDB for Go installation guide](https://clickhouse.com/docs/chdb/install/go)
- [chdb-go high-level API docs](https://github.com/chdb-io/chdb-go/blob/main/chdb.md)
- [chdb-go low-level API docs](https://github.com/chdb-io/chdb-go/blob/main/lowApi.md)

## What chDB is

ClickHouse describes chDB as an in-process SQL engine powered by ClickHouse. The positioning is embedded analytics, not a separate ClickHouse server process.

The product page highlights:

- in-process SQL execution
- support for 80+ data formats
- zero-copy DataFrame exchange
- language bindings including Go
- no need to run ClickHouse as an external service

## Go installation path

The official Go install flow has two parts:

1. Install the native `libchdb` library:

   ```bash
   curl -sL https://lib.chdb.io | bash
   ```

2. Install the Go package:

   ```bash
   go install github.com/chdb-io/chdb-go@latest
   ```

   or add it to a module with:

   ```bash
   go get github.com/chdb-io/chdb-go
   ```

The ClickHouse docs list the supported runtime baseline as Go 1.21+ on Linux and macOS.

## High-level Go API

The generated `chdb` package exposes a small surface area:

- `Query(queryStr, outputFormats...)`
- `QueryStream(queryStr, outputFormats...)`
- `NewSession(paths...)`
- `Session.Query(...)`
- `Session.QueryStream(...)`
- `Session.Cleanup()`
- `Session.Close()`

The docs show three primary usage patterns:

- stateless one-off queries with `chdb.Query(...)`
- stateful sessions backed by a path on disk
- `database/sql` access through the `chdb/driver` import

The docs also show streaming queries for large result sets, with chunked reads and explicit cleanup.

## Low-level Go API

The low-level package `chdb-purego` exposes:

- `NewConnection(argc, argv)`
- `NewConnectionFromConnString(conn_string)`
- `ChdbConn.Query(...)`
- `ChdbConn.QueryStreaming(...)`
- `ChdbConn.Ready()`
- `ChdbConn.Close()`

The low-level docs mark `NewConnection` as deprecated and recommend `NewConnectionFromConnString` instead.

Connection strings can represent:

- in-memory databases, for example `:memory:`
- relative or absolute file-backed databases
- query parameters forwarded as ClickHouse startup args

One documented special case is `mode=ro`, which maps to read-only mode.

## Practical observations

- The binding is intentionally small and close to the embedded-database use case.
- Session-backed usage gives stateful SQL behavior without a separate server deployment.
- Streaming support is a fit for large reads that should not be buffered all at once.
- The `database/sql` driver path exists, but the native API is still the more direct surface.

## API naming mismatch to verify in code

The ClickHouse Go install page uses `session.QueryStreaming(...)` in its streaming example, while the generated docs in `chdb.md` and `lowApi.md` use `QueryStream(...)` / `QueryStreaming(...)` depending on package level.

That naming drift should be checked against the actual Go package before this repo relies on a specific method name in documentation or implementation notes.

## Relevance for this repository

This binding is a plausible fit only if we want an embedded ClickHouse path inside a Go component.

For the current Java-profiler design, the key takeaways are:

- chDB is embedded and does not require a separate ClickHouse service
- Go support is official enough to document
- the API surface is narrow enough to keep integration simple
- the query model is SQL-first, with explicit session and streaming support

