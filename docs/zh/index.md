---
layout: home

hero:
  name: Java Profiler
  text: 面向 Kubernetes 性能问题的 Java Profiler
  tagline: "通过 Kubernetes opt-in、真实 async-profiler/JFR 数据和面向服务的 UI，找到性能问题背后的 Java 调用栈。"
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
  - title: 需要时再开启
    details: 通过 Kubernetes annotation 或 label 启用，节点本地采集，数据保留七天或更短。
  - title: 直接看 Java 调用栈
    details: CPU、Wall Clock、I/O wait、GC、allocation、lock、线程、死锁、status 和 ingestion 数据都绑定到同一个服务和时间范围。
  - title: 自己掌控 Profiling 栈
    details: async-profiler 数据写入 ClickHouse，并由项目自己的 UI 查询；不强制依赖 Pyroscope、Parca 或 Grafana backend。
  - title: 从最有用的问题开始
    details: 先看 status，再看 CPU 或 allocation；如果还解释不了问题，再切到 Wall Clock、I/O、GC、locks 或 ingestion。
---

[![Docs](https://img.shields.io/badge/docs-online-blue?style=flat-square)](https://koolay.github.io/java-profiler/) [![中文文档](https://img.shields.io/badge/docs-中文文档-2b90d9?style=flat-square)](https://koolay.github.io/java-profiler/zh/) [![GitHub stars](https://img.shields.io/github/stars/koolay/java-profiler?style=flat-square)](https://github.com/koolay/java-profiler)

![接受环境中的真实 allocation profile 分析](../assets/screenshots/real-allocation-analysis.png)

## 从这里开始

1. 打开 [快速开始](./getting-started/quickstart.md)，用 Kubernetes metadata 启用 profiling。
2. 打开目标服务，先看 `status`，确认 JVM 已被接受。
3. 从 `cpu` 或 `memory` 开始，再按问题需要查看 `wall`、`io`、`gc`、`locks` 或 `ingestion`。

## 面向服务负责人

- [快速开始](./getting-started/quickstart.md)：启用 profiling 并读懂第一个服务 profile。
- [性能分析用户手册](./operations/performance-analysis-user-manual.md)：分析 CPU、Wall Clock、Java I/O wait、GC、allocation summary、lock、deadlock、target status、profile evidence guidance 和 ingestion evidence。
- [Java Profiling Runbook](../operations/java-profiling-runbook.md)：为 Kubernetes workload 启用临时或持续 profiling。

## 面向平台运维

- [部署运维手册](./operations/deployment-operations-admin-manual.md)：安装、安全、存储、升级和故障处理。
- [真实 Profiling 验收](../operations/real-profiling-acceptance-standard.md)：在发布前证明 CPU、Wall Clock、Java I/O wait、GC、allocation、lock、ClickHouse、UI 和 ingestion 行为。

## 面向贡献者

- [开发设置](./contributing/development.md)：运行本地检查、构建文档、执行真实验收。
- [系统架构](../architecture/java-profiler-architecture.md)：理解 collector、backend、ClickHouse、contracts 和 Web UI。
- [Profiling 合同](./reference/profiling-contracts.md)：查看稳定 payload 和配置合同。
- [GitHub Issues](https://github.com/koolay/java-profiler/issues)：提交 bug 和改动请求。

## 语言和截图

- 需要切换语言时，使用页眉里的英文/中文入口。
- 本页截图来自真实验收环境，不是模拟 UI 状态。allocation 截图展示当前的 Allocation Summary 和 flamegraph 工作流。
