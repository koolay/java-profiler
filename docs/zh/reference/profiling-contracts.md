# Profiling 合同

稳定的 profiling payload 和配置合同位于仓库的 `contracts/profiling` 目录。

修改 collector payload、backend ingestion 或 UI 解释逻辑时，先查看这些文件：

- [`configuration.md`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/configuration.md)
- [`payloads.md`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/payloads.md)
- [`types.go`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/types.go)

如果合同改动影响 scope、retention、collection、storage 或用户可见行为，也要同步更新 requirements、operations guides 和 real profiling acceptance standard。

## 当前 UI 查询合同

Web UI 当前消费 `/api/ui/v1` 下的 product-shaped backend routes：

- `/flamegraph`：flamegraph tree、partial metadata 和 profile value semantics。
- `/top-stacks`：Top Table 工作流使用的 Self/Total ranked rows。
- `/allocation-summary`：`java_allocation_bytes` 的 sampled allocation summary，包含 requested/effective scope、coverage、top allocating paths、top self allocating frames、insights、limitations、partial reasons 和 empty-state reason。
- `/service-summary` 与 `/service-selectors`：service 和 target selectors。
- `/target-status`：JVM eligibility 和 collection status evidence。
- `/ingestion`：aggregate profile batch acceptance、retry、rejection、drop 和 truncation evidence。
- `/jvm-events`、`/thread-diagnosis` 与 `/deadlocks`：GC、thread 和 deadlock diagnosis evidence。

空 profile 状态是用户可见合同的一部分。UI 会区分 disabled profiling、expired temporary windows、unmatched targets、ingestion gaps、query errors，以及 selected range 内没有 samples。这些状态帮助排查缺失结果，但不能替代 strict acceptance 对 CPU、Wall Clock、allocation、I/O 和 lock 非空 profile data 的要求。
