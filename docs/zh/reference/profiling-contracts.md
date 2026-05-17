# Profiling 合同

稳定的 profiling payload 和配置合同在源码目录 `contracts/profiling` 下。

修改 collector payload、backend ingestion 或 UI 解释逻辑时，以这些文件为准：

- [`configuration.md`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/configuration.md)
- [`payloads.md`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/payloads.md)
- [`types.go`](https://github.com/koolay/java-profiler/blob/main/contracts/profiling/types.go)

如果合同改动影响 scope、retention、collection、storage 或用户可见行为，需要同步更新 requirements、operations guides 和 real profiling acceptance standard。
