# 贡献者指南

这个项目有一条底线：profiling 相关改动必须证明真实 profile 数据存在，不能只证明 UI 不报错。

## 开发环境检查

从仓库根目录运行：

```bash
go test ./...
javac --release 11 java-helper/thread-diagnostics/src/main/java/com/ebpfjava/threads/*.java
cd examples/jdk17-http-demo && mvn test
cd ../../web && npm ci && npm test && npm run build
```

## 文档站

```bash
cd docs
npm install
npm run docs:dev
```

发布文档前先构建：

```bash
cd docs
npm run docs:build
```

文档站支持中英文。英文是 source language，中文覆盖核心用户路径和贡献者路径。新增或移动公开文档前，先看 [本地化策略](./localization.md)。

## 真实验收

如果改动影响 collector profiling、ingestion、ClickHouse 存储、backend query API、部署、demo service 或 profile UI，需要跑真实 Kubernetes 验收。

```bash
export KUBECONFIG=$HOME/backup/localk8s.yaml

scripts/real-acceptance.sh \
  --service jdk17-http-demo \
  --configure-profiler \
  --require-full-profiling \
  --high-volume \
  --artifact-dir /tmp/java-profiler-real-acceptance-$(date +%Y%m%d%H%M%S)
```

通过意味着当前运行窗口里有 accepted target status、非空 CPU/allocation/lock profiles、ClickHouse rows、ingestion evidence、受限 retention、浏览器 UI 证据，并且目标 workload restart count 没有增加。

## 截图证据

文档截图必须来自连接真实 backend 的真实 UI。

```bash
export REAL_ACCEPTANCE_BASE_URL=http://127.0.0.1:18081
export REAL_ACCEPTANCE_NAMESPACE=java-profiler-qa
export REAL_ACCEPTANCE_SERVICE=jdk17-http-demo
node scripts/capture-doc-screenshots.mjs
```
