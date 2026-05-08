# Java Profiler 部署运维管理员手册

本文面向平台管理员、SRE、集群运维人员和安全管理员，说明如何部署、配置、升级、验证和排障 `java-profiler`。Java 服务 owner 的日常性能分析流程见 [Java 服务性能分析用户手册](./performance-analysis-user-manual.md)。如果问题是业务栈热点如何解释，应转交服务 owner 并引用用户手册。

## 管理员职责

管理员负责平台能力本身是否可用、安全和可运维：

- 安装和升级 collector DaemonSet、backend、Web UI 和 ClickHouse。
- 配置 collector、backend、Web 和 ClickHouse 的连接、认证、资源和 retention。
- 配置 Kubernetes RBAC、ServiceAccount、Secret 和网络访问边界。
- 确认 Web `/api/*` 代理到 backend，而不是静态资源服务器。
- 验证 collector 能发现目标 Pod、读取 Pod metadata、访问 host proc，并在权限允许时 attach JVM。
- 监控 target status、ingestion、backend、ClickHouse 和 TTL 清理状态。
- 处理上传失败、存储失败、schema 失败、权限失败和升级回滚。

管理员不负责解释业务代码热点。CPU、allocation、lock、deadlock 和线程证据的业务解释由 Java 服务 owner 完成。

## 快速导航

| 任务 | 阅读章节 |
| --- | --- |
| 第一次安装 | 安装前检查、Helm 配置基线、部署验收流程 |
| 升级或回滚 | 上线前验收清单、升级和回滚、验收通过标准 |
| Web UI 能打开但 API 不通 | Web API 代理、故障排查案例 2 |
| collector 没发现目标 | RBAC 和权限、故障排查案例 3 |
| ClickHouse 或 backend 异常 | ClickHouse 管理、故障排查案例 1 |
| 给服务团队交接 | 管理员交接清单、服务团队交接消息模板 |

## 系统边界

`java-profiler` 第一版本只覆盖 Kubernetes 中的 HotSpot 兼容 Java 服务：

- collector 以 DaemonSet 方式运行在节点本地。
- 目标启用通过 Kubernetes annotation 或 label 控制。
- CPU、allocation 和 lock profiling 基于 async-profiler。
- profile、thread、deadlock、target status 和 ingestion 数据写入 ClickHouse。
- 数据 retention 不超过 7 天；可选 raw artifact 不超过 24 小时。
- Prometheus 只负责抓取 collector/backend exporter 指标；本系统不存储 Prometheus 时序数据。

不要把它扩展成通用日志、 tracing、service map、非 Java profiling 或长期指标平台。

稳定 payload 和配置词汇以 [`contracts/profiling/`](../../contracts/profiling/) 为准；本手册解释管理员如何部署、验证和恢复这些合同。

## 组件

| 组件 | 管理员关注点 |
| --- | --- |
| collector DaemonSet | 节点覆盖、ServiceAccount、host proc、Pod metadata 访问、JVM attach 权限、上传重试和 dropped batch。 |
| backend | 认证、ClickHouse DSN、schema apply、ingestion、query API、metrics。 |
| Web UI | 静态资源、`/api/*` 反向代理、UI token、服务暴露方式。 |
| ClickHouse | database、schema、TTL、连接数、写入和查询性能、磁盘容量。 |
| Java helper | 线程快照、deadlock 诊断、JVM 兼容性。 |

## 安装前检查

部署前确认：

- 集群支持 DaemonSet。
- 目标节点允许运行 collector 需要的安全上下文。
- collector ServiceAccount 可读取本节点 Pod metadata。
- backend 可访问 ClickHouse。
- Web Pod 可访问 backend Service。
- Secret 中存在 collector token 和 UI token。
- ClickHouse retention、磁盘容量和备份策略符合平台要求。
- 服务 owner 知道 profiling 默认关闭，必须通过 metadata 显式开启。

## 上线前验收清单

每次新装、升级或回滚后，至少完成这组 smoke test：

- `helm lint` 或等价 chart 检查通过。
- backend Pod Ready，日志显示 ClickHouse ping 和 schema apply 成功。
- ClickHouse 中 profile、thread、deadlock、target status、ingestion 表存在，TTL 不超过 7 天。
- Web 根页面可访问。
- Web `/api/*` 能代理到 backend，而不是返回静态服务器 404。
- 无 token 或错误 token 请求 collector/UI API 会被拒绝。
- collector DaemonSet 覆盖目标节点。
- collector metrics 显示发现 Java 进程。
- 带 profiling annotation 的测试 workload 在 UI `status` 中出现。
- target status 至少能显示 accepted、disabled 或 unsupported 这类明确状态。
- collector upload success 增长，upload retryable 和 dropped batch 没有持续增长。
- rollback 后重复 backend、Web API proxy、collector upload 和 UI target status 检查。

## 验收通过标准

| 检查项 | 通过标准 | 失败后先查 |
| --- | --- | --- |
| backend 启动 | Pod Ready，日志出现 backend listening，无持续 CrashLoop。 | ClickHouse DSN、schema、Secret、网络。 |
| Web API proxy | `/api/ui/v1/target-status` 返回 backend JSON 或认证错误，不是静态 404。 | nginx template、backend URL、Service。 |
| collector 发现 | 测试节点上有 Java 进程时 discovered processes 大于 0；没有 Java 进程时要明确记录测试前提。 | DaemonSet 节点覆盖、host proc、RBAC。 |
| target status | 测试 workload 出现 accepted、disabled、unsupported 或其他明确 reason。 | Pod metadata、JVM 类型、collector 日志。 |
| 上传链路 | upload success 增长；retryable 和 dropped batch 不持续增长。 | backend、网络、token、collector buffer。 |
| retention | schema TTL 不超过 7 天；raw artifact 不超过 24 小时。 | ClickHouse schema、TTL 表达式、清理任务。 |

## Helm 配置基线

管理员应至少确认这些配置项：

```yaml
clusterName: local

image:
  backend: java-profiler-backend:tag
  collector: java-profiler-collector:tag
  web: java-profiler-web:tag

auth:
  existingSecret: java-profiler-auth
  collectorTokenKey: collector-token
  uiTokenKey: ui-token

clickhouse:
  dsn: tcp://default:password@clickhouse:9000/java_profiler

service:
  port: 8080
```

`clusterName` 会进入 target identity。多集群环境必须使用稳定且可区分的集群名，避免不同集群的 namespace/service/Pod 混在一起。

## Secret 管理

建议把 collector token 和 UI token 分开：

- collector token 只用于 `/api/collector/v1/*` ingestion API。
- UI token 只用于 `/api/ui/v1/*` query API。
- token 不应出现在日志、截图、PR 描述或 issue 中。
- token 轮换后，应滚动重启相关 Pod，确认新旧 token 行为符合预期。

无 token 或错误 token 请求必须被拒绝。

## RBAC 和权限

collector 至少需要：

- 读取本节点 Pod 列表和 Pod metadata。
- 访问节点上的进程信息，用于发现 JVM 和映射 Pod。
- 在安全策略允许时 attach 目标 JVM。

建议按最小权限配置：

- collector 只拿到发现和 attach 所需权限。
- Web/UI 查询按 namespace、service 或 owner 授权。
- 修改 `java-profiler.io/*` metadata 的权限只授予服务 owner、平台管理员或事件响应人员。
- ClickHouse profile、thread、deadlock 和 raw artifact 访问权限按敏感生产数据处理。

## Profiling Metadata 变更治理

`java-profiler.io/*` metadata 会直接改变生产采集行为，管理员应把它作为受控变更管理：

- 明确谁可以添加、修改或删除 `java-profiler.io/*` annotation 和 label。
- `continuous` 必须有服务 owner、适用范围和定期复审。
- `temporary` 必须有明确 `profile-duration`。
- 事件结束后必须移除 temporary metadata，或设置 `profile-disabled: "true"`。
- metadata 变更应进入审计日志、工单或 GitOps PR。
- 禁止普通用户跨 namespace 批量开启 profiling。
- 安全敏感服务启用前应确认 profile 数据外发和访问规则。

## Web API 代理

Web 容器必须把 `/api/*` 代理到 backend Service。验收时不要只打开根页面，还要请求 API：

```bash
curl -i http://127.0.0.1:18081/api/ui/v1/target-status
```

预期：

- 返回 backend API 响应，而不是静态服务器 404。
- 缺少 UI token 时按认证策略拒绝。
- 带正确 UI token 或 cookie 时返回 JSON。

常见问题：

- Web Pod Ready，但 nginx 模板没有 envsubst。
- `JAVA_PROFILER_BACKEND_URL` 指向错误 Service。
- 本机代理环境变量拦截 localhost 验收请求；验收时可显式设置 `NO_PROXY='*'`。

## ClickHouse 管理

管理员负责 ClickHouse 可用性和 schema 兼容性：

- backend 启动时应能 ping ClickHouse。
- schema apply 应成功。
- TTL 表达式必须兼容运行中的 ClickHouse 版本。
- target status、ingestion、profile samples、profile stacks、thread snapshots、deadlock events 都应有不超过 7 天 retention。
- raw artifact index 如启用，最长 24 小时。

启动时 ClickHouse 可能已经 Pod Ready，但 TCP、database 或 schema 仍未就绪。backend 应有有限重试窗口；如果超过窗口仍失败，应明确报错，而不是静默进入不可用状态。

## 数据保留和容量

默认 retention：

| 数据 | 最大保留 |
| --- | --- |
| profile samples / stacks | 7 天 |
| thread snapshots / stacks | 7 天 |
| deadlock events | 7 天 |
| target status | 7 天 |
| ingestion health | 7 天 |
| raw artifact index | 24 小时 |

管理员应监控：

- ClickHouse 磁盘容量。
- 写入延迟和失败率。
- 查询延迟。
- TTL lag。
- 表行数和压缩效果。

超过 retention 的空结果不是漏采，应在用户沟通中明确。

## Exporter 指标

collector 和 backend 暴露 Prometheus-compatible metrics。Prometheus 负责抓取、存储、告警和仪表盘。

建议看这些指标组：

- target discovery 和 status counters。
- profiler active、disabled、skipped、failed counters。
- upload success、retry、duplicate、dropped batch counters。
- backend ingestion accepted、duplicate、retryable、rejected counters。
- ClickHouse latency、table size、TTL lag gauges。

## 部署验收流程

### 1. Helm 静态检查

```bash
helm lint ./deploy/helm --values deploy/helm/values.yaml
```

确认 chart 能渲染后再部署。

### 2. 安装或升级

部署后确认：

```bash
kubectl -n <namespace> get pods
kubectl -n <namespace> rollout status deploy/<release>-backend
kubectl -n <namespace> rollout status deploy/<release>-web
kubectl -n <namespace> rollout status ds/<release>-collector
```

所有组件应 Ready。

### 3. Backend 和 ClickHouse 验收

检查 backend 日志：

- 成功连接 ClickHouse。
- schema apply 成功。
- backend 开始监听。
- 没有持续 CrashLoop。

### 4. Web 代理验收

端口转发 Web：

```bash
kubectl -n <namespace> port-forward svc/<release>-web 18081:80
```

请求 UI API：

```bash
NO_PROXY='*' curl -i 'http://127.0.0.1:18081/api/ui/v1/target-status'
```

确认走 backend，而不是静态 404。

### 5. Collector 验收

部署一个带 annotation 的 Java 测试 workload：

```yaml
metadata:
  annotations:
    java-profiler.io/profile-mode: temporary
    java-profiler.io/profile-duration: 10m
    java-profiler.io/startup-delay: 0s
    java-profiler.io/snapshot-interval: 10s
```

确认 collector metrics：

- discovered processes 大于 0。
- target status 有 accepted、disabled、unsupported 或其他明确状态。
- upload success 增长。
- upload retryable 和 dropped batch 没有持续增长。

### 6. UI 状态验收

在 UI 中选择测试 workload 的 namespace 和 service：

- `status` 能看到 Pod、PID、Seen、State、Reason、Message、User action。
- accepted 目标能进入 CPU、memory、locks 视图。
- 没有 profile 数据时，UI 显示空状态，不误报为成功分析结论。

## 升级和回滚

升级前：

- 确认 schema 变化是否向后兼容。
- 确认 payload contract 是否变化。
- 确认 Web `/api/*` 代理配置仍然存在。
- 记录当前 image tag、chart version 和 values。
- 保存当前 chart version。
- 保存当前 image tags。
- 保存当前 Helm values。
- 保存当前 auth Secret 名称和 key 名称，不记录 token 明文。
- 保存当前 ClickHouse DSN 目标，不记录密码。
- 保存当前 namespace、release name 和 service port。

升级后：

1. backend 无 CrashLoop。
2. collector 上传成功。
3. target status query 返回真实数据。
4. profile/query API 没有 500。
5. Web UI 可打开并能查询 `/api/ui/v1/target-status`。

回滚后重复同样验收。不要只依赖 Pod Ready。

## 故障排查案例

### 案例 1：backend 启动失败

现象：

- backend CrashLoop。
- Web API 返回 503 或 502。
- UI 显示 backend unavailable。

排查：

1. 查看 backend 日志。
2. 检查 ClickHouse DSN、Secret 和网络。
3. 检查 schema apply 错误。
4. 检查 TTL 表达式是否兼容当前 ClickHouse。
5. 确认 ClickHouse database 已存在或可创建。

处理：

- 修复 DSN、Secret 或 schema。
- 如果是 ClickHouse 慢启动，确认 backend storage readiness retry 是否足够。

### 案例 2：Web 页面能打开，但 API 404

现象：

- 根页面正常。
- `/api/ui/v1/target-status` 返回静态服务器 404。

排查：

1. 查看 Web nginx 配置是否生成。
2. 检查 `JAVA_PROFILER_BACKEND_URL`。
3. 检查 Web Pod 能否解析 backend Service。
4. 检查 Helm values 中 service port。

处理：

- 修复 nginx template、env 或 Service 名称。
- 滚动重启 Web。

### 案例 3：collector 没有发现目标

现象：

- discovered processes 为 0。
- `status` 没有目标。

排查：

1. collector 是否运行在目标 Pod 所在节点。
2. host proc mount 是否存在。
3. ServiceAccount 是否能读本节点 Pod metadata。
4. 目标服务是否真的是 Java 进程。
5. JVM 是否已经启动且未快速退出。

处理：

- 修复 DaemonSet 节点覆盖、mount、RBAC 或目标 workload。

### 案例 4：目标一直是 `disabled_by_metadata`

现象：

- collector 找到 Java 进程。
- UI 状态为 `disabled_by_metadata`。

排查：

1. Pod metadata 是否有 `java-profiler.io/profile-mode`。
2. 是否存在 `java-profiler.io/profile-disabled: "true"`。
3. annotation 和 label 是否冲突。
4. 修改是否加在 Pod template，而不是只加在 Deployment metadata。

处理：

- 修正 metadata。
- 必要时 rollout workload。

### 案例 5：目标是 `unsupported_jvm`

现象：

- Java 进程被发现，但状态为 `unsupported_jvm`。

排查：

1. JVM 是否 HotSpot 兼容。
2. 是否为 wrapper、helper、sidecar Java 进程。
3. 同一 Pod 是否还有另一个业务 JVM。

处理：

- 对非 HotSpot JVM 不启用 first-version profiling。
- 通过 PID、container、JVM start time 区分目标。

### 案例 6：目标是 `profiler_conflict`

现象：

- 目标 JVM 被其他 async-profiler 或诊断工具占用。

排查：

1. 是否有人手动运行 async-profiler。
2. 是否有其他 profiling agent。
3. 是否存在平台侧自动诊断任务。

处理：

- 停止冲突工具，或跳过该 JVM。
- 不要同时对同一个 JVM 运行多个 profiler。

### 案例 7：目标是 `attach_failed`

现象：

- metadata 和 JVM 条件满足，但 attach 失败。

排查：

1. collector 安全上下文。
2. 目标容器安全策略。
3. JVM attach 是否被禁用。
4. collector 是否能访问目标进程 namespace。

处理：

- 修复权限或 JVM 参数。
- 重新触发 collector 扫描。

### 案例 8：上传 retryable 或 dropped batch

现象：

- collector metrics 中 upload retryable 增长。
- 或 dropped batch 增长。

排查：

1. backend 是否 Ready。
2. 网络是否能从 collector 到 backend。
3. collector token 是否正确。
4. backend ingestion 是否返回 5xx、401、400。
5. collector buffer 是否过小或 backend 长时间不可用。

处理：

- retryable 表示可恢复，先修 backend 或网络。
- dropped batch 表示部分数据已丢失，需要告知用户该窗口证据不完整。

### 案例 9：storage rejected

现象：

- backend 拒绝 batch。
- ingestion 显示 rejected。

排查：

1. batch id 是否被复用。
2. 相同 batch id 是否带了不同 payload。
3. payload 是否缺少 target identity、时间或 profile type。
4. collector/backend 版本是否合同不一致。

处理：

- duplicate 同 payload 可以幂等处理。
- reused batch id with different payload 应拒绝。
- 修正 collector payload contract。

### 案例 10：UI 空结果

现象：

- API 成功返回，但 UI 无 profile 或 target。

排查：

1. namespace、service、Pod、时间范围是否正确。
2. 数据是否超过 retention。
3. target status 是否有 accepted。
4. ingestion 是否有对应时间窗口的 accepted profile batch。
5. profile 视图是否选择了正确 profile type。

处理：

- 空结果不是错误。需要用 status 和 ingestion 判断是没有匹配数据，还是采集链路失败。

### 案例 11：真实 profile 数据链路验收

现象：

- target status 已 accepted，但需要证明性能分析能力完整可用。

验收：

1. 部署带稳定 CPU 负载的 HotSpot Java 测试服务。
2. 启用 temporary profiling。
3. 确认 target status accepted。
4. 等待 profile batch 上传并被 backend accepted。
5. 查询 ClickHouse profile sample 和 stack 行数。
6. 打开 UI `cpu`，确认 flamegraph 有非 root 栈。
7. 如有 allocation 或 lock 负载，分别确认 `memory` 和 `locks` 有非空栈。

结论：

- 只有 target status accepted 只能证明控制面可用。
- 非空 CPU/allocation/lock profile 才能证明性能分析数据链路可用。

### 案例 12：retention 验收

现象：

- 用户询问历史事件为什么没有数据。

验收：

1. 写入或找到接近 retention 边界的数据。
2. 确认 7 天内可查询。
3. 确认超过 7 天后 TTL 或清理任务删除。
4. 确认 raw artifact 不超过 24 小时。

结论：

- retention 清理后的空结果符合设计。
- 长期趋势应由 Prometheus 或其他系统保存，不由 java-profiler 保存。

## 管理员交接清单

部署或升级后，向服务 owner 交接这些信息：

- Web UI 地址。
- 可用 namespace/service 范围。
- profiling metadata 使用方式。
- target status 和 ingestion 的解释方式。
- retention 限制。
- 敏感数据和访问控制要求。
- 已知未支持能力，例如非 HotSpot JVM、非 Java profiling、heap dump retained-heap 分析。

## 服务团队交接消息模板

```text
Java Profiler 已在 <cluster>/<namespace> 可用。

Web UI:
可用范围:
启用 temporary profiling 示例:
启用 continuous profiling 申请方式:
默认 retention: profile/thread/deadlock/status/ingestion 数据不超过 7 天
遇到 UI 没数据先看: status，然后看 ingestion
权限申请方式:
管理员联系渠道:
平台问题联系:
安全提醒: 不要外发完整 flamegraph、thread dump、stack payload、token 或未脱敏截图
用户手册: docs/operations/performance-analysis-user-manual.md
```

## 管理员事故记录模板

```text
Cluster:
Namespace:
Release:
Component: collector / backend / web / ClickHouse
Version / image:
Incident window:
Symptom:
User impact:
Target status evidence:
Ingestion evidence:
Backend logs:
ClickHouse evidence:
Root cause:
Fix:
Follow-up:
```
