# 快速开始

当 `java-profiler` 已经部署好，并且你想分析一个 Java 服务时，走这条路径。

## 1. 启用 profiling

在目标 workload 的 pod template 上添加 metadata：

```yaml
metadata:
  annotations:
    java-profiler.io/profile-mode: temporary
    java-profiler.io/profile-duration: 15m
```

线上事件优先用 `temporary`。只有经过批准需要长期采集的核心服务，才使用 `continuous`。

## 2. 打开服务

在 Web UI 中设置：

- `Namespace`：Kubernetes namespace。
- `Service`：服务或 workload 名称。
- `Range`：包含 profiling 运行时间的窗口。

如果 UI 没有数据，先看 [Target status](../../operations/java-profiling-runbook.md#validate-an-existing-workload)。它会说明 JVM 是 accepted、disabled、unsupported、expired，还是 attach failed。

如果某个 profile 视图为空，先读 Profile evidence status 提示。它会把 target status 和 aggregate ingestion evidence 合在一起，解释 profiling disabled、temporary window expired、target 不匹配、ingestion gap，或当前时间范围没有 samples。

## 3. 分析 profile

排查高 CPU 时，先打开 [性能分析用户手册](../operations/performance-analysis-user-manual.md) 对应的 CPU workflow。

使用：

- Top Table 找最贵的 Java 方法。
- Both mode 同时保留 Top Table 和 Flame Graph，方便从排名最高的 symbol 追到调用栈上下文。
- Flame Graph 看完整 sampled stack context。
- Selected frame details 对比 Self CPU 和 Total CPU。
- Search 和 Focus 隔离真正重要的调用路径。

allocation pressure 看 Allocation profiles。先读 Allocation Summary，再对比 Top allocating paths、Top self allocating frames 和 flamegraph context。sampled allocation evidence 说明对象创建压力，不代表 retained heap ownership。

CPU 解释不了的延迟看 Wall Clock。Socket 或文件阻塞看 I/O wait。暂停时间看 GC pauses 和 allocation correlation。锁竞争看 lock diagnosis。

## 4. 检查 ingestion health

在相信“没有 profile”之前，先检查 [Ingestion health](../operations/performance-analysis-user-manual.md#ingestion-health)。有用的诊断需要看到当前服务和时间范围内的 accepted profile batches。Ingestion health 是 backend aggregate evidence，所以仍要同时确认 target status、namespace、service、Pod 和 time range。

## 5. 关闭 profiling

临时 profiling 会自动过期。持续 profiling 不再需要时，移除或禁用 metadata。
