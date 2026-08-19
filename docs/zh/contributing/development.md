# 贡献者指南

这个项目最怕的是 UI 看起来正常，但 collector 实际没有产生 profile 数据。因此，影响 profiling 链路的改动不能只做页面检查，还要用真实数据验收。

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

发布文档前先构建文档站：

```bash
cd docs
npm run docs:build
```

文档站支持中英文。英文是源语言，中文覆盖核心用户和贡献者路径。新增或移动公开文档前，先看 [本地化策略](./localization.md)。

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

通过意味着当前运行窗口里有 accepted target status、非空 CPU/allocation/lock profiles、ClickHouse rows、ingestion evidence、受限 retention 和真实浏览器 UI 结果，并且目标 workload restart count 没有增加。

## 截图证据

文档截图必须来自连接真实 backend 的真实 UI。

```bash
export REAL_ACCEPTANCE_BASE_URL=http://127.0.0.1:18081
export REAL_ACCEPTANCE_NAMESPACE=java-profiler-qa
export REAL_ACCEPTANCE_SERVICE=jdk17-http-demo
node scripts/capture-doc-screenshots.mjs
```
