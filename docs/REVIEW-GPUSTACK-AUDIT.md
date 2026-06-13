# GPUStack Backend Architecture Audit

> 审查日期：2026-06-13
> 审查范围：`~/projects/ai-platform-study/gpustack-reference`
> 审查目的：架构参考，禁止复制代码

## 1. GPUStack 后端架构摘要

### 1.1 进程模型

GPUStack 是一个 **Python** 项目，入口为 `gpustack start`。支持三种角色：

| 角色 | 说明 |
|------|------|
| `SERVER` | 纯控制面（API、调度、控制器） |
| `WORKER` | 纯执行面（GPU 发现、推理服务） |
| `BOTH` | Server 内嵌 Worker（通过 `multiprocessing.Process`） |

LightAI Go 对应：Server 和 Agent 是**两个独立 Go 二进制**（`cmd/server`、`cmd/agent`），不内嵌。

### 1.2 Server 职责

- 数据库迁移（Alembic）
- 初始化数据（admin、default cluster）
- FastAPI 应用（路由、认证、中间件、CORS）
- Coordinator（单机 LocalCoordinator 或分布式插件）
- Leader-only 任务：Scheduler、Controllers、Collectors
- 嵌入式 Gateway（Higress）
- WebSocket Proxy（隧道代理模式）

### 1.3 Worker 职责

- 向 Server 注册
- 心跳（POST `/worker-heartbeat`）
- 同步节点状态（GPU、CPU、内存、文件系统）
- 管理模型实例（ServeManager）
- 运行推理后端（vLLM/SGLang/Custom）
- 运行基准测试
- 暴露 Worker API（日志流、文件系统、代理）

### 1.4 GPU 发现

GPUStack 使用**检测器模式**：
- `Runtime` 检测器：通过原生 GPU API（`gpustack-runtime`）检测 CUDA/ROCm/CANN
- `Fastfetch` 检测器：系统信息（非 GPU）
- `Custom` 检测器：静态配置

LightAI Go 采用**Collector 接口模式**（更轻量）：
- `SystemCollector`：gopsutil
- `GPUCollector` 接口：NvidiaCollector、MetaXCollector、MockGPUCollector

### 1.5 模型部署与实例管理

GPUStack 有完整的状态机：
- `PENDING -> SCHEDULED -> STARTING -> RUNNING（或 ERROR）`
- 调度器：多级过滤器 + 候选选择器 + 评分器
- 资源计算：GGUF 解析、transformers.PretrainedConfig、diffusers index
- Worker 端：ServeManager 监控实例，HTTP 健康检查

LightAI Go Phase 1 不实现调度；Phase 2 只做资源采集；Phase 5 才实现实例生命周期。

### 1.6 推理引擎集成

GPUStack 支持：
- vLLM（~42KB 代码，含分布式多节点、LoRA、推测解码）
- SGLang（~28KB 代码，16+ 版本分支）
- Ascend MindIE（~90KB 代码，Ascend NPU）
- VoxBox（音频模型）
- Custom（用户自定义镜像和命令）

全部通过 `gpustack-runtime` 的 deployer 模块执行（子进程或容器）。

LightAI Go 对应：Phase 5-6 实现 Docker 启停，远期才考虑推理引擎。

### 1.7 Prometheus/Grafana/Metrics

- Worker 端：`prometheus-client` 导出 Worker 指标 + Runtime 指标
- Server 端：MetricExporter 导出集群级指标
- 内置 Prometheus（端口 19090）
- 内置 Grafana（端口 13000，预配 Dashboard）
- Gateway 指标：模型使用中间件，缓冲刷新到 DB

LightAI Go 对应：`04-observability-design.md` 的三种模式，Prometheus/Grafana 通过 Docker Compose 部署，不编译进 Go 二进制。

### 1.8 Gateway/API Key/用量

- Gateway 模式：auto、embedded、incluster、external、disabled
- 使用 Higress（K8s Ingress/Gateway API）
- API Key：access_key + hashed_secret_key，支持过期、模型范围、scope
- 用量跟踪：ModelUsage、MeteredUsage、ModelUsageDetails（含归档）
- 认证：JWT，支持 OIDC/SAML/本地密码

LightAI Go 当前窗口不实现 Gateway、API Key、用量统计、计费。

### 1.9 容器参数

- Container 对象：image、name、profile、restart_policy、execution、envs、mounts、resources、ports
- WorkloadPlan：name、host_network、shm_size、containers
- GPU 固定：CUDA_VISIBLE_DEVICES
- 端口分配：40000-40063（服务）、41000-41999（Ray）

LightAI Go 对应：`05-runtime-environment-design.md` 的 DockerRunSpec，更简洁。

---

## 2. 值得 LightAI Go 借鉴的设计

### 2.1 检测器/Collector 接口抽象

GPUStack 的 `GPUDetector` 抽象基类 → LightAI Go 的 `GPUCollector` 接口。两者都支持多厂商扩展。

**借鉴点**：Collector 失败时保留旧状态、返回诊断信息（`CollectorDiagnosis`），这些 LightAI Go 已经设计好了。

### 2.2 Worker 注册 upsert 逻辑

GPUStack 的注册返回 per-worker token + PredefinedConfig。LightAI Go 的注册返回 `agent_token`，语义类似但更简单。

**借鉴点**：注册幂等（同一 agent_id 重复注册不创建重复 Node），LightAI Go 已经设计好了。

### 2.3 心跳使用 Server 接收时间

GPUStack 和 LightAI Go 都使用 Server 接收时间作为心跳时间戳，客户端时间仅用于诊断。这是正确的做法。

### 2.4 状态机与 operation/generation

GPUStack 的 ModelInstanceStateEnum 和 LightAI Go 的 `operation`/`generation` 机制都解决了并发操作冲突问题。

**借鉴点**：LightAI Go 的 lease 语义（`LeaseOwner`、`LeaseExpiresAt`）比 GPUStack 的简单心跳更严谨。

### 2.5 指标命名规范

GPUStack 使用 `prometheus-client` 库，LightAI Go 使用 `promhttp`。两者都遵循 Prometheus 命名最佳实践（`_bytes`、`_ratio`、snake_case）。

### 2.6 HTTP Pull 模式

两者都使用 Agent/Worker 主动向 Server 发起 HTTP 请求（注册、心跳、上报），而不是 Server 向 Agent 发起连接。这是 NAT/防火墙友好的设计。

---

## 3. LightAI Go 已经覆盖的点

| 能力 | GPUStack | LightAI Go |
|------|----------|------------|
| Server/Agent 架构 | ✅ | ✅ |
| Agent 注册/心跳 | ✅ | ✅ |
| GPU 发现接口 | ✅（Detector） | ✅（Collector） |
| 系统信息采集 | ✅（Fastfetch） | ✅（gopsutil） |
| Prometheus metrics | ✅ | ✅ |
| 节点在线/离线 | ✅（WorkerSyncer） | ✅（heartbeat timeout） |
| RBAC 认证 | ✅（JWT + OrgRole） | ✅（Permission code + Session） |
| 多租户 | ✅（Organization） | ✅（Tenant） |
| CSRF 保护 | ✅ | ✅ |
| SQLite 存储 | ✅ | ✅ |
| 容器执行 | ✅（gpustack-runtime） | 远期（Phase 6） |
| 模型调度 | ✅（多级过滤+评分） | 远期（Phase 5+） |
| 推理引擎 | ✅（vLLM/SGLang/etc） | 远期 |
| API Key | ✅ | 远期 |
| Gateway | ✅（Higress） | 远期 |
| 用量/计费 | ✅ | 远期 |

---

## 4. LightAI Go 第一阶段不应采用的重能力

以下 GPUStack 能力对 LightAI Go Phase 0-2B 来说**过重**，不采用：

1. **分布式 Coordinator**（Redis 选举、Leader 切换、`os._exit(1)`）：Phase 1 单实例即可
2. **多级调度器**（7 级过滤器 + 4 级评分器）：Phase 5+ 才需要
3. **推理引擎集成**（vLLM/SGLang/AscendMindIE）：Phase 5-6 只做 Docker 启停
4. **Gateway/Higress**：远期
5. **WebSocket Proxy**（隧道代理）：远期
6. **内置 Prometheus/Grafana**：LightAI Go 用 Docker Compose 外部部署
7. **用量归档**（TableArchiver）：远期
8. **Kubernetes 部署**：远期
9. **多进程架构**（Server 内嵌 Worker）：LightAI Go 是独立二进制
10. **Fastfetch 二进制**：LightAI Go 用 gopsutil 纯 Go 库

---

## 5. 对 Phase 0 到 Phase 2B 的开发启发

### Phase 0（基础骨架）

- GPUStack 的 `main.py` → `cmd/start.py` → `server/server.py` 启动流程：先配置、再日志、再 DB、再 HTTP
- LightAI Go 对应：main → config → log → DB → HTTP router → start

### Phase 0.5（认证/RBAC）

- GPUStack 使用 JWT + OrgRole（简单模型）
- LightAI Go 使用 Permission code + RolePermission + TenantMembershipRole（更细粒度）
- 关键差异：LightAI Go 的 RBAC 模型更复杂，但第一阶段只需要最小可用

### Phase 1（Agent 注册/心跳）

- GPUStack 的注册返回 token → LightAI Go 应该类似
- GPUStack 的心跳是简单 POST → LightAI Go 也是
- GPUStack 的 WorkerSyncer 判断在线/离线 → LightAI Go 用 `last_heartbeat_at` + timeout

### Phase 2A（System/Registry/Mock）

- GPUStack 使用 Fastfetch 二进制 → LightAI Go 用 gopsutil（纯 Go，更轻）
- GPUStack 的 DetectorFactory → LightAI Go 的 CollectorRegistry
- GPUStack 的 Custom detector → LightAI Go 的 MockGPUCollector

### Phase 2B（NVIDIA Collector）

- GPUStack 使用 `gpustack-runtime` 原生 API → LightAI Go 用 `nvidia-smi` CLI
- 两者都需要解析 GPU 设备信息和指标
- LightAI Go 的 bytes/percent/ratio 规则已在 `08-engineering-contracts.md` 明确定义

---

## 6. 不得复制 GPUStack 代码的说明

**禁止行为**：
- 禁止逐行翻译 GPUStack Python 代码为 Go
- 禁止复制 GPUStack 的类结构、函数名、变量名
- 禁止复制 GPUStack 的 SQL 查询
- 禁止复制 GPUStack 的配置键名
- 禁止复制 GPUStack 的 API 路径设计（除非是行业标准如 `/healthz`）
- 禁止复制 GPUStack 的 HTML 模板、CSS、JS
- 禁止复制 GPUStack 的测试用例

**允许行为**：
- 学习架构模式（如 Server/Agent 分离、Pull 模式、Collector 接口）
- 学习行业标准实践（如 Prometheus 命名规范、HTTP SD 格式）
- 学习故障处理思路（如心跳超时、注册幂等）
- 学习安全实践（如 Argon2id、CSRF、Session Cookie）
- 引用 GPUStack 作为设计决策的参考理由
