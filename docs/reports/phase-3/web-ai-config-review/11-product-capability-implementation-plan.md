# LightAI Go Web AI 产品能力实施规划与验收标准

建议仓库路径：

```text
docs/reports/phase-3/web-ai-config-review/11-product-capability-implementation-plan.md
```

状态：

```text
Status: REVISED_FOR_PHASE_1_IMPLEMENTATION
```

## 1. 实施前提

本计划基于两个设计文档：

```text
09-resource-scheduling-and-runtime-policy-design.md
10-manual-verification-issues-and-fix-plan.md
```

本轮原则：

```text
先确认设计
再依据设计开发
最后按验收标准验证
```

## 2. Schema 边界过渡声明

前几轮 Web AI 展示改造阶段（文档 01-08）坚持 no-schema / no-migration 边界，因为目标仅为展现方式调整。

从本产品能力修复计划开始，为实现模型能力持久化、资源参数存储、部署运行策略等核心产品能力，**允许有目标、最小化、干净的 schema/API 演进**。

约束仍然存在：

- 不得引入旧数据兼容脏逻辑。
- 本项目不需要为了历史数据保留复杂 fallback。
- 所有 schema/API 演进必须有文档、测试和验收。
- Phase 1 不做 schema 变更；schema 变更从 Phase 2 开始。

## 3. 本轮允许范围

允许：

```text
必要 schema/API 演进
模型能力持久化
资源与性能参数结构化
部署页 summary DTO 或后端 join
菜单去重
测试诊断增强
RunPlan 参数映射
```

不做：

```text
完整多副本调度
跨节点自动调度
quota
优先级
亲和/反亲和
自动 failover
Playwright spec 实现
API Gateway / API Key
```

## 4. 阶段规划

## Phase 0：文档确认

### 目标

确认设计文档无异议。

### 产出

```text
09-resource-scheduling-and-runtime-policy-design.md
10-manual-verification-issues-and-fix-plan.md
11-product-capability-implementation-plan.md
```

### 验收

```text
现有问题全部落文档
资源调度设计完整
第一阶段/后续阶段边界清楚
实现步骤清楚
验收标准清楚
schema 边界过渡声明已记录
模型能力持久化方案已选定（Option B）
资源字段映射已补全
```

### Phase 1 精确范围

```text
Phase 1 MUST:
1. 去掉"诊断与测试"侧边栏菜单入口（保留 route，不删除页面文件）。
2. 部署列表/详情显示模型名称，不显示 UUID 作为主名。
3. Qwen3 404 诊断增强（MV-008a），不承诺一定在 Phase 1 修复 Chat Completion。
4. 更新文档：schema 边界、能力持久化方案、资源字段映射。

Phase 1 MAY:
1. 前端模型名 lookup 集中化为共享 composable（useModelNames）。
2. 部署页 accelerator count 展示（如 "RTX 5090 × 1"）。
3. 部署页首个部署空状态引导。

Phase 1 MUST NOT:
1. 实现模型能力持久化 schema 变更（Phase 2）。
2. 实现模型编辑页能力编辑器（Phase 2）。
3. 实现资源参数编辑器（Phase 3）。
4. 新增 resource_policy first-class column。
5. 新增 placement_policy first-class column。
6. 实现 Playwright spec。
7. 实现多副本/跨节点调度。
8. 实现自动 failover / retry。
```

## Phase 1：P0 产品问题修复

### 范围

```text
去掉“诊断与测试”独立菜单
部署页显示模型名称
Qwen3 Chat 404 诊断增强
```

### 修改点

前端：

```text
导航/路由
部署列表/详情模型显示
实例/部署测试错误提示
i18n
```

后端：

```text
必要时增加 deployment summary DTO
/model-instances/{id}/test 增强 endpoint probe 和诊断返回
```

### 验收

```text
主导航无重复“诊断与测试”
部署页显示 Qwen3-0.6B-Instruct-2512，而不是 UUID
Chat 404 错误包含 endpoint/mode/backend/runtime image/probe 信息
```

## Phase 2：模型能力持久化与模型编辑页

### 范围

```text
模型能力持久化
模型编辑页
默认测试方式持久化
```

### 建议数据设计

根据当前 schema 选择：

**已选定方案：Option B — 在 model_artifacts 表新增 TEXT 字段。**（Phase 2 实施）

```sql
ALTER TABLE model_artifacts ADD COLUMN capabilities_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_artifacts ADD COLUMN capability_sources_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE model_artifacts ADD COLUMN default_test_mode TEXT NOT NULL DEFAULT 'auto';
```

示例存储：

```json
// capabilities_json
["chat", "completion"]

// capability_sources_json
{"chat": "user_override", "completion": "scan"}

// default_test_mode
"chat"
```

选择理由：

- Option A（metadata JSON 字段复用）：会混淆扫描 metadata 与用户 override，语义不清。
- Option C（独立 ModelCapability 表）：对当前阶段过度设计，增加 join 复杂度。
- Option B：最小、清晰，可通过 SQLite JSON 函数查询，不需要额外 join。
- Phase 1 不做 schema 变更；Phase 2 实施此方案。

### UI

模型编辑页：

```text
可编辑：
- 显示名称
- 描述
- 标签
- 能力
- 默认测试方式

只读：
- size
- checksum
- format
- architecture
- quantization
- parameter count
- context length
- location/path
```

### 验收

```text
能力可编辑
保存后刷新仍存在
测试入口使用持久化能力
Qwen3 默认 Chat
扫描事实只读
```

## Phase 3：资源与性能参数

### 范围

```text
NBR 页面资源与性能分区
后端性能参数映射
容器资源限制展示/编辑
RunPlan/equivalent docker command 生效
```

### 后端参数

vLLM：

```text
gpu_memory_utilization
max_model_len
tensor_parallel_size
max_num_seqs
dtype
```

SGLang：

```text
mem_fraction_static
tp_size
context_length
max_running_requests
```

llama.cpp：

```text
ctx_size
n_gpu_layers
batch_size
threads
```

容器资源：

```text
cpu_limit
memory_limit
shm_size
ulimits
```

### 验收

```text
vLLM 可配置 gpu_memory_utilization
RunPlan args 包含对应参数
equivalent docker command 包含对应参数
不同 backend 显示不同字段
不出现 MACA_VISIBLE_DEVICE
不出现 gpu_ids
```

## Phase 4：Deployment resource policy 与 placement 退化实现

### 范围

第一阶段只实现单副本/单节点，但预留调度边界。

```text
replicas = 1
placement candidate
resource policy
selected node/model location/NBR/accelerator_ids
```

### UI

部署页展示：

```text
副本数：1（当前版本）
资源策略
候选节点
模型位置
NBR
accelerator_ids
RunPlan 预览
```

### 验收

```text
Deployment 和 Instance 概念清楚
Instance 显示所属 Deployment/Node/accelerator_ids
RunPlan 说明对应候选节点/实例
不写死单机命名
```

## Phase 5：Qwen3 endpoint 诊断和修复

### 诊断项

```text
/v1/models
/v1/chat/completions
/v1/completions
/health
RunPlan command
equivalent docker command
Docker logs
runtime image
model id / served model name
```

### 验收

```text
如果根因可修，Chat Completion 成功
如果根因不可修，错误提示可诊断
错误提示不再只有 HTTP 404
```

## Phase 6：测试与回归

### 命令

```bash
gofmt -w cmd/ internal/
go test ./internal/server/api/...
go test ./internal/server/runplan/...
go vet ./...
npm --prefix web test
npm --prefix web run build
git diff --check
git status --short
```

### 可选

```text
server/API smoke
Qwen3 真实 endpoint probe
```

## 5. Closeout 要求

新增：

```text
docs/reports/phase-3/web-ai-config-review/12-product-capability-fix-closeout.md
```

必须包含：

```text
设计文档路径
现有问题文档路径
实施计划文档路径
每个 issue 的处理结果
模型能力持久化实现
资源与性能实现
调度边界落地
P0/P1/P2 完成情况
未做事项
测试命令和结果
schema/migration 说明
commit id
push 结果
final git status
```

## 6. 状态

```text
Status: REVISED_FOR_PHASE_1_IMPLEMENTATION
```
