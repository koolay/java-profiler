# E2E 自动化测试说明

本文说明如何按照 [Java 服务性能分析用户手册](./performance-analysis-user-manual.md) 执行 `java-profiler` 的端到端自动化测试。目标是验证服务负责人在 UI 中能完成真实诊断流程：先确认目标状态，再查看 CPU、memory、locks、deadlocks、线程证据和 ingestion 证据。

本文只覆盖自动化测试执行和验收证据。部署、权限、ClickHouse、collector DaemonSet、token 和 Web 代理问题按 [部署运维管理员手册](./deployment-operations-admin-manual.md) 处理。

## 测试目标

E2E 测试应证明：

- Web UI 可以加载并完成服务诊断入口流程。
- 用户可以选择 namespace、service 和时间范围。
- `status` 能解释目标是否可采集。
- `cpu`、`memory`、`locks`、`deadlocks` 和线程证据能展示对应诊断证据，或给出可解释的空状态。
- `ingestion` 能说明 collector 上传、backend 接受、存储拒绝、重试或丢弃情况。
- 真实 Kubernetes 环境中，目标 Java workload 启用 profiling 后，UI、backend、ClickHouse 和 collector 数据链路一致。
- 测试过程保存截图、视频、浏览器 console、ClickHouse 查询结果、collector/backend 日志和目标 Pod restart count。

## 测试层级

| 层级 | 命令 | 用途 |
| --- | --- | --- |
| UI smoke | `cd web && npm run test:browser` | 使用本地 Vite UI 验证诊断页面能加载。 |
| 真实集群验收 | `scripts/real-acceptance.sh` | 在 Kubernetes 中验证 collector、backend、ClickHouse、Web UI 和 Java workload 的完整链路。 |
| 严格真实验收 | `scripts/real-acceptance.sh --require-full-profiling` | 要求 status、profile、thread 和 deadlock 数据在当前运行中非空。 |

UI smoke 不能证明真实 async-profiler attach 或 ClickHouse 数据链路可用；它只验证 UI 基础流程。发布前至少运行一次真实集群验收。

## 前置条件

本地工具：

```bash
kubectl version --client
helm version
curl --version
jq --version
cd web && npm install
cd web && npx playwright install --with-deps chromium
```

真实集群：

- `KUBECONFIG` 指向可用 Kubernetes 集群。
- 集群允许部署 collector DaemonSet、backend、Web UI 和 ClickHouse。
- 有一个 HotSpot 兼容 Java workload，或允许脚本创建测试 workload。
- 目标 workload 已通过 `java-profiler.io/*` metadata 启用 profiling。
- 测试 namespace 和 profiler namespace 权限已配置。
- 如使用本仓库新增 demo 服务，可先构建并导入 `examples/jdk17-http-demo` 镜像。

## 用户手册到自动化场景映射

| 用户手册流程 | 自动化场景 | 主要断言 |
| --- | --- | --- |
| 使用前确认 | 环境预检 | `kubectl` 当前 context、namespace、Pod、Service、Helm release 可访问。 |
| 启用 profiling | 安装或配置测试 workload | Pod template 存在 `java-profiler.io/profile-mode`，目标 Pod 无新增 restart。 |
| UI 使用顺序 | Playwright service diagnosis flow | 页面加载，左侧主视图导航可见，namespace/service/range 可填写，tab 可切换。 |
| 理解目标状态 | `status` 视图 | 至少出现 `accepted`、`disabled_by_metadata`、`temporary_expired` 或其他稳定 reason。 |
| CPU 升高 | `cpu` 视图 | flamegraph 搜索框可见，root frame 和非 root 栈可见。 |
| GC 压力或 allocation rate | `memory` 视图 | allocation flamegraph 或可解释空状态可见。 |
| 锁竞争导致请求慢 | `locks` 视图 | lock flamegraph、contention 证据或可解释空状态可见。 |
| 疑似死锁 | `deadlocks` 视图 | deadlock cycle 可见；若为空，必须结合 status/ingestion 解释。 |
| 线程池耗尽或忙线程 | 线程证据 | busy/slow/thread state 证据可见，或可解释空状态可见。 |
| UI 没有数据 | `ingestion` 视图 | accepted、retryable、dropped 或 rejected 证据可见。 |
| 真实 profile 数据链路确认 | strict real acceptance | 当前运行窗口内 profile batch 被接受，UI 展示真实方法栈。 |

## 快速 UI Smoke

运行：

```bash
cd web
npm install
npm run test:browser
```

预期：

- Vite 启动在 `http://127.0.0.1:5173`。
- Playwright 执行 `web/tests/profiling-flow.spec.ts`。
- 页面出现 `Service diagnosis` 标题。
- 至少一个诊断 tab 按钮可见，左侧主视图导航图标可见。

失败处理：

- 端口被占用时，先停止占用 `5173` 的进程。
- 依赖缺失时重新执行 `npm install` 和 `npx playwright install chromium`。
- UI 文案调整导致 selector 失败时，优先更新测试到用户可见角色和文本，不使用脆弱 CSS selector。

## 真实集群 E2E

查看脚本参数：

```bash
scripts/real-acceptance.sh --help
```

首次安装并运行：

```bash
KUBECONFIG=/path/to/kubeconfig \
scripts/real-acceptance.sh \
  --install \
  --namespace java-profiler-qa \
  --service checkout-java \
  --artifact-dir /tmp/java-profiler-real-acceptance
```

验收当前工作区代码时，先构建本地镜像，再让验收脚本部署这些 tag：

```bash
export BACKEND_IMAGE=java-profiler-backend:qa-$(date +%Y%m%d%H%M%S)
export COLLECTOR_IMAGE=java-profiler-collector:qa-$(date +%Y%m%d%H%M%S)
export WEB_IMAGE=java-profiler-web:qa-$(date +%Y%m%d%H%M%S)

bash scripts/build-real-acceptance-images.sh
```

复用已有 Java workload，仅更新 profiler 目标过滤：

```bash
KUBECONFIG=/path/to/kubeconfig \
scripts/real-acceptance.sh \
  --configure-profiler \
  --namespace java-profiler-qa \
  --service checkout-java \
  --artifact-dir /tmp/java-profiler-real-acceptance
```

发布前严格验收：

```bash
KUBECONFIG=/path/to/kubeconfig \
scripts/real-acceptance.sh \
  --configure-profiler \
  --namespace java-profiler-qa \
  --service jdk17-http-demo \
  --require-full-profiling \
  --high-volume \
  --artifact-dir /tmp/java-profiler-real-acceptance-$(date +%Y%m%d%H%M%S)
```

严格验收推荐使用 `jdk17-http-demo`，因为它能稳定产生 CPU、allocation 和 lock contention。若复用已有 workload，必须确认该 workload 在 profiling 窗口内能持续产生对应负载。

关键环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `BACKEND_IMAGE` | `ghcr.io/koolay/java-profiler-backend:0.1.0` | backend 镜像。 |
| `COLLECTOR_IMAGE` | `ghcr.io/koolay/java-profiler-collector:0.1.0` | collector 镜像。 |
| `WEB_IMAGE` | `ghcr.io/koolay/java-profiler-web:0.1.0` | Web UI 镜像。 |
| `CLICKHOUSE_IMAGE` | `docker.m.daocloud.io/clickhouse/clickhouse-server:24.8` | ClickHouse 镜像。 |
| `JAVA_WORKLOAD_IMAGE` | `docker.m.daocloud.io/eclipse-temurin:21-jdk` | 测试 Java workload 镜像。 |
| `JAVA_WORKLOAD_PREBUILT` | `0` | 设为 `1` 表示 workload 镜像已经自带 CPU-busy Java app。 |
| `UI_TOKEN` | `qa-ui-token` | UI 调 backend 的 token。 |
| `COLLECTOR_TOKEN` | `qa-collector-token` | collector 调 backend 的 token。 |
| `JAVA_PROFILER_HIGH_VOLUME_SECONDS` | `180` | `--high-volume` 下的负载和采集窗口秒数。 |
| `JAVA_PROFILER_LOAD_PARALLELISM` | `4` | high-volume CPU/allocation 请求并发度。 |
| `JAVA_PROFILER_LOCK_PARALLELISM` | `6` | high-volume lock contention 请求并发度。 |

## 使用 JDK 17 Demo 服务作为目标

推荐用脚本部署 demo、等待 rollout、可选产生负载，并保存部署证据：

```bash
KUBECONFIG=/path/to/kubeconfig \
scripts/deploy-jdk17-demo.sh \
  --namespace java-profiler-qa \
  --image ghcr.io/koolay/java-profiler-jdk17-http-demo:latest \
  --run-load \
  --artifact-dir /tmp/java-profiler-jdk17-demo-deploy
```

demo 镜像由 GitHub workflow `.github/workflows/profile-demo-image.yml` 生成并推送到：

```text
ghcr.io/koolay/java-profiler-jdk17-http-demo:latest
ghcr.io/koolay/java-profiler-jdk17-http-demo:sha-<commit>
```

如果要测试本地未发布镜像，或目标集群不能直接拉取 GHCR 镜像，可以构建并导入节点 containerd：

```bash
KUBECONFIG=/path/to/kubeconfig \
scripts/deploy-jdk17-demo.sh \
  --namespace java-profiler-qa \
  --image jdk17-http-demo:local \
  --build-image \
  --load-to-node \
  --run-load
```

脚本会执行：

- 使用 `--image` 指定的镜像，默认是 GitHub workflow 发布的 GHCR demo 镜像。
- 如传 `--build-image`，构建本地镜像。
- 如传 `--load-to-node`，把本地镜像导入 Kubernetes 节点 containerd。
- 创建测试 namespace。
- 应用 `examples/jdk17-http-demo/k8s.yaml`。
- 设置 Deployment 镜像。
- 等待 `deploy/jdk17-http-demo` rollout。
- 保存 Deployment、Service、Pod 和 restart count 证据。
- 如果传 `--run-load`，通过 port-forward 调用 `/health`、`/work` 和 `/threads`。
- 输出下一步 `scripts/real-acceptance.sh` 命令。

手工部署步骤如下。

构建镜像：

```bash
docker build -t jdk17-http-demo:local examples/jdk17-http-demo
```

部署 demo：

```bash
kubectl create namespace java-profiler-qa --dry-run=client -o yaml | kubectl apply -f -
kubectl -n java-profiler-qa apply -f examples/jdk17-http-demo/k8s.yaml
kubectl -n java-profiler-qa rollout status deploy/jdk17-http-demo --timeout=120s
```

demo manifest 已在 Pod template 上启用 profiling：

```yaml
metadata:
  labels:
    java-profiler.io/profile-mode: temporary
  annotations:
    java-profiler.io/profile-mode: temporary
    java-profiler.io/profile-disabled: "false"
    java-profiler.io/profile-duration: 1h
    java-profiler.io/startup-delay: 0s
    java-profiler.io/snapshot-interval: 10s
    java-profiler.io/acceptance-run: "20260516225619"
```

`profile-disabled: "false"` 用于覆盖旧的禁用标记；`acceptance-run` 应使用每次运行唯一值，强制 Deployment 滚动出新的 profiling 窗口。

产生负载：

```bash
kubectl -n java-profiler-qa port-forward svc/jdk17-http-demo 18080:8080
curl "http://127.0.0.1:18080/work?mode=cpu&durationMs=30000"
curl "http://127.0.0.1:18080/work?mode=alloc&durationMs=30000"
curl "http://127.0.0.1:18080/work?mode=lock&durationMs=30000"
curl "http://127.0.0.1:18080/threads?durationMs=30000"
```

用 demo 运行真实验收时，把 service 设置为 `jdk17-http-demo`：

```bash
KUBECONFIG=/path/to/kubeconfig \
scripts/real-acceptance.sh \
  --configure-profiler \
  --namespace java-profiler-qa \
  --service jdk17-http-demo \
  --require-full-profiling \
  --high-volume \
  --artifact-dir /tmp/java-profiler-jdk17-demo-e2e
```

如果要验证一个已有的非 demo 服务，例如 `kd-cosmic-xk/mservice`，可以保留 `--service mservice`，并通过 `JAVA_PROFILER_ACCEPTANCE_LOAD_PATHS` 指定真实可用的 HTTP 路径，例如：

```bash
KUBECONFIG=/path/to/kubeconfig \
JAVA_PROFILER_ACCEPTANCE_LOAD_PATHS=/profile-load/ \
scripts/real-acceptance.sh \
  --configure-profiler \
  --namespace kd-cosmic-xk \
  --profiler-namespace java-profiler-qa \
  --service mservice \
  --require-full-profiling \
  --artifact-dir /tmp/java-profiler-mservice-e2e
```

如果状态变成 `profiler_conflict`，通常是前一次运行已把 async-profiler 加载进同一个 JVM，而 collector/backend 又重启过。滚动目标 demo Pod 后重新运行严格验收。

## Playwright 真实 UI 验收

真实 UI 测试文件是 `web/tests/real-acceptance.spec.ts`，默认跳过。脚本会在需要时设置环境变量；也可以手动运行：

```bash
cd web
REAL_ACCEPTANCE=1 \
REAL_ACCEPTANCE_BASE_URL=http://127.0.0.1:18081 \
REAL_ACCEPTANCE_NAMESPACE=java-profiler-qa \
REAL_ACCEPTANCE_SERVICE=jdk17-http-demo \
REAL_ACCEPTANCE_ARTIFACT_DIR=/tmp/java-profiler-real-acceptance-ui \
npx playwright test --config=playwright.config.ts tests/real-acceptance.spec.ts
```

该测试按照用户手册顺序执行：

1. 打开 UI。
2. 填写 namespace 和 service。
3. 选择最近 60 分钟。
4. 打开 `status` 并截图。
5. 打开 `cpu`，切换 Top Table、Flame Graph、Both，并截图。
6. 搜索应用符号，确认 flamegraph 高亮/弱化命中项。
7. 选中 Top Table 行和 flamegraph frame，确认详情、Focus、Back、Reset。
8. 打开 `deadlocks` 并截图。
9. 打开 `ingestion`，确认有 accepted 上传证据并截图。
10. 附加浏览器 console 信息。

## 验收证据

每次真实 E2E 应保留 artifact 目录。至少包含：

- `summary.md`：脚本执行摘要、PASS/GAP/FAIL。
- UI 截图：status、cpu、deadlocks、ingestion。
- Playwright video 和 trace，失败时用于复盘。
- 浏览器 console 输出。
- collector 日志。
- backend 日志。
- ClickHouse profile、thread、deadlock、target status、ingestion 计数。
- 目标 workload profiling 前后的 restart count。
- `kubectl get pods -o wide` 和相关 Deployment/Service 描述。

验收结论必须区分：

- `PASS`：当前运行窗口内数据链路完整，UI 展示可解释证据。
- `GAP`：测试环境、负载或目标状态导致部分证据为空；未使用 `--require-full-profiling` 时允许记录但不能当作完整通过。
- `FAIL`：脚本或断言失败，需要修复后重跑。

## 通过标准

完整通过需要满足：

- UI smoke 通过。
- 真实集群 E2E 脚本退出码为 `0`。
- `status` 至少有一个当前目标显示 `accepted` 或等价可采状态。
- `ingestion` 显示当前运行窗口内有 accepted 数据。
- CPU profile 有非 root 栈，或严格记录为什么测试负载没有产生样本。
- 如本次目标包含 allocation、lock 或 deadlock 负载，对应视图有非空证据。
- 目标 Pod 测试前后 restart count 没有增加。
- 没有未解释的 browser console error、backend panic、collector crash 或 ClickHouse 写入拒绝。

## 常见失败定位

| 现象 | 优先检查 |
| --- | --- |
| UI 空白或 401 | Web 代理、UI token、backend URL。 |
| `status` 为空 | namespace/service 过滤、collector target filter、workload metadata。 |
| reason 是 `disabled_by_metadata` | Pod template 是否使用 `java-profiler.io/profile-mode` 或 `profile-disabled`。 |
| reason 是 `temporary_expired` | 临时窗口是否已过期，是否需要滚动重启或重新开启窗口。 |
| reason 是 `unsupported_jvm` | JVM 是否 HotSpot 兼容，是否选中了 sidecar/helper Java 进程。 |
| reason 是 `attach_failed` | collector 权限、容器安全策略、JVM attach 参数。 |
| reason 是 `profiler_conflict` | 上一次运行留下 async-profiler 状态，先滚动目标 Pod，再重跑严格验收。 |
| profile 为空但 status accepted | 负载不足、时间范围错误、profile batch 未上传、retention 过期。 |
| ingestion rejected | backend 合同、ClickHouse schema、payload 版本；若提示 `batch_id` / `collector_id` 缺失，重新构建并部署当前工作区 collector/backend 镜像。 |
| no-Service 合成 workload 很快结束 | 使用更新后的脚本；验收脚本必须等待完整 profiling 窗口。 |
| Playwright strict locator 报重复 frame | 重复栈帧是合法数据；测试应按用户可见角色/文本定位，并在必要时选择第一个匹配项。 |
| 目标 Pod 重启 | 先停止扩大采集范围，检查 attach 安全性和资源压力。 |

## CI 建议

PR 级别：

```bash
go test ./...
cd web && npm test && npm run build && npm run test:browser
```

合并前或夜间环境：

```bash
KUBECONFIG=/path/to/kubeconfig \
scripts/real-acceptance.sh \
  --configure-profiler \
  --namespace java-profiler-qa \
  --service jdk17-http-demo \
  --require-full-profiling \
  --artifact-dir "$CI_ARTIFACTS/java-profiler-e2e"
```

CI 应上传 artifact 目录，并在失败时保留 Playwright trace、video、截图和集群日志。
