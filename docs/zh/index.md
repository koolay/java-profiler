---
layout: home

hero:
  name: Java Profiler
  text: 面向 Kubernetes 事故排障的低开销 Java Profiling
  tagline: "一个聚焦 HotSpot 服务的 profiler：Kubernetes metadata opt-in、async-profiler/JFR 真实证据、ClickHouse 存储，以及面向 Java 事故排障的 UI。"
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/getting-started/quickstart
    - theme: alt
      text: 分析服务
      link: /zh/operations/performance-analysis-user-manual
    - theme: alt
      text: GitHub
      link: https://github.com/koolay/java-profiler

features:
  - title: 默认采用 opt-in
    details: 通过 Kubernetes annotation 或 label 显式启用，节点本地采集，默认保留 7 天或更短。
  - title: 真实 Java 证据
    details: CPU、Wall Clock、Java I/O wait、GC、allocation、lock delay、线程、死锁、target status 和 ingestion evidence 都绑定到同一个服务和时间范围。
  - title: 自己掌控 Profiling 栈
    details: 不强制依赖 Pyroscope、Parca 或 Grafana 后端。async-profiler 数据写入 ClickHouse，并由自有 UI 查询。
  - title: 更快拿到第一条证据
    details: 先看 status，再看 CPU；当 CPU 解释不了问题时，再切到 Wall Clock 或 I/O。
---

[![Docs](https://img.shields.io/badge/docs-online-blue?style=flat-square)](https://koolay.github.io/java-profiler/) [![中文文档](https://img.shields.io/badge/docs-中文文档-2b90d9?style=flat-square)](https://koolay.github.io/java-profiler/zh/) [![GitHub stars](https://img.shields.io/github/stars/koolay/java-profiler?style=flat-square)](https://github.com/koolay/java-profiler)

![接受环境中的真实 CPU profile 分析](../assets/screenshots/real-cpu-analysis.png)

## 3 分钟路径

1. 打开 [快速开始](./getting-started/quickstart.md)，用 Kubernetes metadata 启用 profiling。
2. 打开目标服务，先看 `status`，确认 JVM 已被接受。
3. 再按 `cpu`、`wall`、`io`、`gc`、`locks`、`ingestion` 的顺序从现象走到证据。

## 面向服务负责人

- [快速开始](./getting-started/quickstart.md)：启用 profiling 并读懂第一个服务 profile。
- [性能分析用户手册](./operations/performance-analysis-user-manual.md)：分析 CPU、Wall Clock、Java I/O wait、GC、allocation、lock、deadlock、target status 和 ingestion evidence。
- [Java Profiling Runbook](../operations/java-profiling-runbook.md)：为 Kubernetes workload 启用临时或持续 profiling。

## 面向平台运维

- [部署运维手册](../operations/deployment-operations-admin-manual.md)：安装、安全、存储、升级和故障处理。
- [真实 Profiling 验收](../operations/real-profiling-acceptance-standard.md)：在发布前证明 CPU、Wall Clock、Java I/O wait、GC、allocation、lock、ClickHouse、UI 和 ingestion 行为。

## 面向贡献者

- [开发设置](./contributing/development.md)：运行本地检查、构建文档、执行真实验收。
- [系统架构](../architecture/java-profiler-architecture.md)：理解 collector、backend、ClickHouse、contracts 和 Web UI。
- [Profiling 合同](./reference/profiling-contracts.md)：查看稳定 payload 和配置合同。
- [GitHub Issues](https://github.com/koolay/java-profiler/issues)：提交 bug 和改动请求。

## 信任与本地化

- 需要英文或中文时，直接使用页眉里的语言切换。
- 本页截图来自真实验收环境，不是模拟 UI 状态。
