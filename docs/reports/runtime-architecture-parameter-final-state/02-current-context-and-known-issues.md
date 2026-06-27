# Current Context and Known Issues

## 1. 项目背景

LightAI Go 是面向中小客户 GPU 服务器场景的轻量化 GPU 与模型运行管理平台。平台需要统一管理 GPU 资源、模型运行配置、后端运行模板、节点运行配置、模型部署、实例生命周期、日志和 API 入口。

当前 Runtime 主线已经形成以下核心对象：

1. Backend；
2. BackendVersion；
3. BackendRuntime；
4. NodeBackendRuntime；
5. ModelArtifact；
6. ModelLocation；
7. Deployment；
8. ModelInstance；
9. RunPlan / NodeRunPlan；
10. Agent Docker runtime；
11. Preflight；
12. HealthCheck；
13. RuntimeParameterEditor。

本阶段目标是继续收敛这些对象的边界和实现。

## 2. 需要读取的已有材料

如果仓库中存在以下文档，Claude 必须先读取并核对：

```text
docs/reports/phase-3/runtime-architecture-and-parameter-current-gap-review.md
docs/reports/phase-3/runtime-architecture-and-parameter-repair-plan.md
```

这些文档只作为历史输入材料。本阶段新输出进入：

```text
docs/reports/runtime-architecture-parameter-final-state/
```

还应检索以下关键词，寻找相关设计与修复记录：

```bash
grep -R "RuntimeParameterEditor" -n docs internal web cmd || true
grep -R "RuntimeRequirements" -n docs internal web cmd || true
grep -R "BackendCapabilityProfile" -n docs internal web cmd || true
grep -R "discovered_metadata_json" -n docs internal web cmd || true
grep -R "parameter_schema_json" -n docs internal web cmd || true
grep -R "parameter_values_json" -n docs internal web cmd || true
grep -R "RunPlan" -n docs internal web cmd || true
grep -R "Preflight" -n docs internal web cmd || true
```

## 3. 已知重点问题

### 3.1 discovered_metadata_json 边界错误

`discovered_metadata_json` 应描述模型类别、模型格式、模型家族、模型能力等稳定信息。它不应包含某个本机模型路径。

需要检查是否存在如下路径进入通用 catalog 或模板：

```text
/home/kzeng/models/bge-small-zh-v1.5
/home/kzeng/models/Qwen3-0.6B-Instruct-2512
/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

如果存在，需要判断字段归属：

1. 模型路径归 ModelLocation；
2. 模型实例扫描结果归 ModelArtifact / ModelLocation；
3. 模型类别能力归 model metadata；
4. 后端能力归 BackendCapabilityProfile；
5. 运行需求归 RuntimeRequirements；
6. 最终执行归 ResolvedRunPlan。

### 3.2 RuntimeRequirements 定义不清

RuntimeRequirements 需要成为可执行契约，不能只作为说明性 JSON。

需要检查它是否能驱动：

1. UI 提示；
2. API 校验；
3. Preflight；
4. Agent environment check；
5. RunPlan；
6. E2E 断言。

必须覆盖：

1. image；
2. docker runtime；
3. accelerator；
4. device binding；
5. model path；
6. model format；
7. required files；
8. ports；
9. mounts；
10. env；
11. args；
12. health check；
13. resource controls；
14. endpoint protocol；
15. warning/blocking error。

### 3.3 BackendCapabilityProfile 定义不清

BackendCapabilityProfile 应描述后端能力，不能承载节点状态或部署实例状态。

需要检查并定义：

1. supported model formats；
2. supported protocols；
3. endpoint paths；
4. parameter schema；
5. resource controls；
6. health check modes；
7. device binding modes；
8. accelerator abstraction；
9. runtime limitations；
10. warning capabilities。

### 3.4 参数体系边界不清

需要检查参数在以下层级是否混乱：

1. Model / ModelArtifact；
2. Backend；
3. BackendVersion；
4. BackendRuntime；
5. NodeBackendRuntime；
6. Deployment；
7. ResolvedRunPlan。

已知风险：

1. schema/value 未完整保存；
2. clone 后参数丢失；
3. refresh 后参数丢失；
4. enabled 和 value 混在一起；
5. disabled input 不显示；
6. 未 enabled 参数错误进入 Docker args；
7. required/default 参数没有统一规则；
8. vLLM/SGLang/llama.cpp 参数覆盖不完整；
9. resource_controls 与 args 没有一致映射。

### 3.5 RuntimeParameterEditor / RunnerConfigsPage 问题

需要重点检查：

1. RunnerConfigsPage 是否存在双入口；
2. legacy Docker editor 是否仍与 RuntimeParameterEditor 并存；
3. RuntimeParameterEditor 是否未 populate 数据；
4. watch → emit 是否可能循环；
5. 页面是否只显示勾选框；
6. disabled input 是否保留值展示；
7. enabled 后是否可编辑；
8. 保存后刷新是否造成 OOM；
9. schema/value 是否能 round-trip；
10. clone 是否保留 enabled + value。

### 3.6 RunPlan preview 与实际执行不一致

需要检查：

1. preview command 与 Agent Docker create spec 是否一致；
2. env 是否混入 capabilities_json；
3. args 是否重复；
4. GPU/device binding 是否一致；
5. ports 与 health check 是否一致；
6. mounts 是否一致；
7. resource controls 是否进入 args；
8. deployment override 是否生效；
9. NBR snapshot 与 deployment snapshot 边界是否清楚。

### 3.7 NodeBackendRuntime 部署入口约束

需要确认：

1. Deployment 只接受 `node_backend_runtime_id`；
2. Deployment 拒绝 `backend_runtime_id`；
3. UI 只选择 ready / ready_with_warnings 的 NBR；
4. needs_check、missing_image、failed、disabled 不可部署；
5. check-request 由 Server 代理 Agent；
6. image inspect 为权威；
7. 不自动创建 NBR。

### 3.8 日志与状态问题

需要检查：

1. instance id 与 container id 是否混用；
2. container logs 是否使用正确 container id；
3. container 退出后状态是否更新；
4. health check 是否驱动 running；
5. stop 后实例列表策略是否明确；
6. 前端是否自动刷新；
7. API 是否返回可读错误；
8. operation_id 是否贯通。

## 4. 必须产出的复核文档

Claude 执行 Batch 0 后必须生成：

```text
docs/reports/runtime-architecture-parameter-final-state/00-existing-docs-and-code-reconciliation.md
```

该文档必须包含：

1. 已读取的历史文档；
2. 历史文档结论是否仍成立；
3. 当前代码已解决的问题；
4. 当前代码仍存在的问题；
5. 历史文档遗漏的问题；
6. 当前新增问题；
7. 需要修复的 P0/P1/P2 清单；
8. 每个问题的代码路径；
9. 每个问题的验证方式；
10. 无法验证项及原因。
