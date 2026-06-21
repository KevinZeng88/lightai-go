# LightAI Go Web AI 手工验证问题与修复审议

建议仓库路径：

```text
docs/reports/phase-3/web-ai-config-review/10-manual-verification-issues-and-fix-plan.md
```

状态：

```text
Status: REVISED_FOR_PHASE_1_IMPLEMENTATION
```

## 1. 背景

用户完成 Web AI 手工验证后，发现当前页面展示优化仍未解决若干产品主线问题。本文件记录现有问题、影响、根因假设、修复设计、实施阶段和验收标准。

本文件与资源调度设计文档配套：

```text
09-resource-scheduling-and-runtime-policy-design.md
```

## 2. 问题清单

### WEB-AI-MV-001：模型编辑能力不足

#### 现象

模型库添加模型后，点击“编辑”，不能修改模型详情页展示的主要可配置信息，尤其不能人工修正模型能力。

#### 期望

编辑页应允许修改：

```text
模型显示名称
描述
标签
模型能力
默认测试方式
```

扫描事实只读：

```text
大小
checksum
format
architecture
quantization
parameter count
context length
模型路径/location
```

#### 影响

模型自动扫描可能不准确，尤其能力识别可能错误。如果不能人工修正，会导致：

```text
测试方式错误
部署匹配错误
后端能力判断错误
用户无法纠正平台判断
```

#### 修复方案

1. 检查现有 ModelArtifact / metadata schema。
2. 设计模型能力持久化字段或表。
3. 模型编辑页分为“可编辑信息”和“扫描事实”。
4. 保存后刷新页面仍保留能力。
5. 测试入口使用持久化能力决定默认 mode。

#### 验收

```text
Qwen3-0.6B-Instruct-2512 可设置为 Chat
刷新后能力仍存在
测试默认 Chat Completion
扫描事实不可编辑或只读
```

---

### WEB-AI-MV-002：模型能力人工修正不能持久化

#### 现象

当前能力展示主要来自推断或前端展示，没有形成明确可持久化配置。

#### 期望

模型能力必须可持久化。

能力至少包括：

```text
chat
completion
embedding
rerank
vision
tool_calling
structured_output
```

能力来源：

```text
scan
inferred
user_override
backend_probe
```

默认测试方式：

```text
auto
chat
completion
embedding
rerank
```

#### 修复方案

建议语义：

```json
{
  "capabilities": ["chat", "completion"],
  "capability_sources": {
    "chat": "user_override",
    "completion": "scan"
  },
  "default_test_mode": "chat"
}
```

具体实现由当前 schema 决定：

```text
若已有 metadata JSON 字段合适，可使用
若没有合适字段，新增干净字段或表
不做旧数据复杂兼容
```

#### 验收

```text
能力可编辑
能力可保存
能力刷新后仍存在
测试入口读持久化能力
API 返回能力
```

---

### WEB-AI-MV-003：缺少显存/资源限制配置入口

#### 现象

用户没有看到显存大小或使用率配置入口。

#### 期望

NBR 或 Deployment 页面应有“资源与性能”配置入口。

至少包括：

```text
GPU 卡选择 / accelerator_ids
GPU 数量
显存使用率
最大上下文长度
并行参数
CPU limit
Memory limit
shm-size
ulimits
```

#### 修复方案

按层次设计：

```text
Placement / Accelerator 选择
Backend performance params
Vendor runtime binding
Container resource limits
```

vLLM 至少支持：

```text
gpu_memory_utilization
max_model_len
tensor_parallel_size
dtype
max_num_seqs
```

SGLang 支持：

```text
mem_fraction_static
tp_size
context_length
max_running_requests
```

llama.cpp 支持：

```text
ctx_size
n_gpu_layers
batch_size
threads
```

#### 验收

```text
NBR 页面出现“资源与性能”
vLLM 可配置显存使用率
RunPlan/equivalent docker command 体现参数
不同 backend 显示不同参数
```

---

### WEB-AI-MV-004：资源限制未考虑不同 GPU vendor

#### 现象

资源限制容易被写成 NVIDIA-only 参数。

#### 期望

UI 统一，但底层按 vendor/backend 映射。

#### 设计要求

```text
NVIDIA:
- DeviceRequest
- CUDA_VISIBLE_DEVICES

MetaX:
- /dev/mxcd
- /dev/dri/cardX
- /dev/dri/renderDXXX
- CUDA_VISIBLE_DEVICES

Huawei:
- 通过 runtime template/catalog 定义
- 不凭空硬编码未知参数

CPU:
- cpu_none
```

禁止重新引入：

```text
MACA_VISIBLE_DEVICE
METAX_VISIBLE_DEVICES
gpu_ids
```

#### 验收

```text
accelerator_ids 全链路保持
MetaX 不出现 MACA_VISIBLE_DEVICE
NVIDIA/MetaX/Huawei 逻辑分层清楚
```

---

### WEB-AI-MV-005：多服务器、多实例、资源调度设计边界不足

#### 现象

现阶段实现容易变成单机单实例强绑定。

#### 期望

第一阶段可单机单副本，但设计必须支持未来：

```text
多节点
多实例
多副本
跨节点运行
调度策略
失败重试
资源池
quota
优先级
亲和/反亲和
```

#### 修复方案

文档中明确对象边界：

```text
Deployment = 部署意图
Instance = 实际运行实例
Placement = 调度候选/结果
ResolvedRunPlan = 实例最终运行计划
```

第一阶段实现：

```text
replicas = 1
scheduler 退化为单节点候选选择和资源校验
```

#### 验收

```text
页面显示副本数=1 或单副本说明
RunPlan 说明对应候选节点/实例
实例显示所属 deployment/node/accelerator
不写死 single-node/single-instance 命名
```

---

### WEB-AI-MV-006：“模型实例”和“诊断与测试”页面重复

#### 现象

“模型实例”和“诊断与测试”页面一样。

#### 期望

去掉“诊断与测试”独立菜单。

测试与诊断能力保留在：

```text
模型实例详情
模型部署详情
```

后续如果做独立诊断台，应具备不同职责：

```text
最近失败
测试历史
operation 追踪
批量健康检查
日志搜索
```

#### 验收

```text
主导航无重复页面
模型实例详情仍有测试/日志/诊断
无 i18n key 泄露
```

#### 重要说明

- 去掉"诊断与测试"独立菜单，**仅指从侧边栏/主导航移除入口**。
- **不删除已有 route**。
- **不删除页面文件**。
- `/models/test-diagnostics` 或现有测试诊断 route 仍可直接访问。
- 测试诊断能力主入口迁移到模型实例详情 / 部署详情。
- 这样避免破坏已有书签或直接链接。

---

### WEB-AI-MV-007：部署页显示模型 UUID 而不是模型名称

#### 现象

部署页面显示：

```text
633d14eb-ed29-45b4-85fe-ea1d26cc837e
```

期望显示：

```text
Qwen3-0.6B-Instruct-2512
```

#### 影响

客户无法识别部署对应的模型。

#### 修复方案

1. 部署列表、详情、RunPlan、实例关联均使用模型显示名。
2. UUID 只作为高级 ID、tooltip 或复制字段。
3. 如果 API 不返回模型名，补后端 summary DTO 或 join。
4. 不要在前端多个页面散落重复 lookup。

#### 验收

```text
部署列表显示模型名
部署详情显示模型名
RunPlan 摘要显示模型名
UUID 不作为主显示
```

---

### WEB-AI-MV-008a：Qwen3 Chat Completion 404 诊断增强（Phase 1）

#### 现象

Qwen3-0.6B-Instruct-2512 运行后测试失败：

```text
Chat Completion 请求失败：
接口 http://127.0.0.1:8004/v1/chat/completions
HTTP 状态 404
错误摘要 chat completions returned HTTP 404
```

#### Phase 1 目标

**诊断增强和信息充分**，不承诺一定在 Phase 1 修复 Chat Completion。

如果诊断发现是 RunPlan / endpoint / command 等小范围错误，可以直接修。
如果需要 schema / API / 后端模板大调整，则进入 Phase 2+。

#### 诊断必须输出

```text
endpoint
mode
backend
runtime image
/v1/models probe result
/health probe result
HTTP status
错误摘要
建议动作
operation_id（如可用）
container logs hint（如可用）
```

#### Phase 1 验收

```text
错误信息包含 endpoint/mode/backend/runtime image
错误信息包含 /v1/models probe 结果
不再只有 HTTP 404
诊断输出完整
```

---

### WEB-AI-MV-008b：Qwen3 Chat Completion 404 根因修复（Phase 2+）

#### 触发条件

MV-008a 诊断确认根因后实施。

#### 可能根因

```text
端口指向错误服务
容器未启动 OpenAI-compatible server
vLLM command/entrypoint 不正确
后端只支持 completion 不支持 chat
模型 ID 不匹配
```

#### 验收

```text
Chat Completion 成功返回 200
测试弹窗显示成功结果
/test API 返回 ok=true,mode=chat
```

---

### WEB-AI-MV-009：自动化完整流程验证应延后到产品主线修复后

#### 现象

已安装 Playwright，并已写设计文档，但当前产品能力尚未修完。

#### 决策

暂停 Playwright spec 实现，先修产品主线。

保留：

```text
@playwright/test 依赖
08-playwright-browser-smoke-design.md
```

暂不新增：

```text
web/playwright.config.ts
web/e2e/*.spec.ts
```

#### 后续

产品能力修复后，再做：

```text
mock UI smoke
live runtime UI E2E
API/runtime E2E
```

## 3. 实施优先级

### P0

```text
部署页模型名显示修复
去掉重复“诊断与测试”菜单
Qwen3 Chat 404 诊断增强
```

### P1

```text
模型能力持久化
模型编辑页
资源与性能配置入口
```

### P2

```text
Deployment resource policy
Placement candidate 表达
单副本调度校验
```

### P3

```text
Playwright 自动化
多副本调度
跨节点调度
失败自动重试
```

## 4. 验收命令

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

## 5. 文档状态

```text
Status: REVISED_FOR_PHASE_1_IMPLEMENTATION
```

本文件要求先讨论确认，再依据设计实施。
