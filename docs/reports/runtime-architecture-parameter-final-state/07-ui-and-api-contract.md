# UI and API Contract

## 1. 目标

本文定义 Runtime 架构与参数体系最终态下 UI/API 的行为边界。UI 不应通过展示全部字段来回避领域边界；API 应返回足够结构化的信息支持分层展示、override、source map 和自动化验收。

## 2. UI 总原则

1. 每个页面只展示当前层级拥有或允许覆盖的内容。
2. 参数按 category 分组。
3. basic 参数优先展示。
4. advanced 参数默认折叠。
5. required/default/inherited/override/source 清晰标识。
6. checked/enabled 只表示当前层级显式覆盖。
7. disabled input 仍显示当前值。
8. RunPlan preview 显示最终值和来源。
9. UI 不复制 schema。
10. UI 不把 default 显示成 checked。

## 3. 页面契约

### 3.1 Model / ModelArtifact 页面

展示：

1. 模型名称；
2. 模型格式；
3. 模型家族；
4. 量化；
5. 上下文能力；
6. modality；
7. embedding/chat/rerank 能力；
8. 模型文件；
9. ModelLocation；
10. discovered metadata 中属于模型自身的内容。

不展示：

1. Docker image；
2. Docker args；
3. Docker env；
4. GPU runtime；
5. accelerator device binding；
6. container port；
7. BackendRuntime 参数编辑项。

### 3.2 Backend / BackendVersion 页面

展示：

1. 后端能力；
2. 版本能力；
3. 支持的 endpoint；
4. 支持的模型格式；
5. 参数能力定义；
6. RuntimeRequirements 摘要；
7. BackendCapabilityProfile。

不展示节点状态和部署状态。

### 3.3 BackendRuntime 页面

展示：

1. 运行模板名称；
2. 关联 BackendVersion；
3. image；
4. command；
5. 模板级 args/env/mounts/ports；
6. 模板级参数默认值或 override；
7. 模板 health check。

要求：

1. RuntimeParameterEditor 必须 populate schema/value/source；
2. legacy Docker editor 与 RuntimeParameterEditor 不能双入口冲突；
3. 保存后刷新不丢 schema/value/enabled/source；
4. clone 不扩大 checked 范围。

### 3.4 NodeBackendRuntime 页面

展示：

1. 节点；
2. 关联 BackendRuntime snapshot；
3. enable 状态；
4. check-request evidence；
5. image inspect evidence；
6. Docker runtime；
7. device binding；
8. 节点 env；
9. 节点路径/mount；
10. 节点 override。

要求：

1. ready 和 ready_with_warnings 可用于部署；
2. needs_check、missing_image、failed、disabled 禁止部署；
3. check-request evidence 可读；
4. 不自动创建 NBR。

### 3.5 Deployment 页面

展示：

1. 模型选择；
2. ModelLocation；
3. NodeBackendRuntime 选择；
4. 部署级 override；
5. 资源覆盖；
6. 端口覆盖；
7. 卷覆盖；
8. 健康检查覆盖；
9. 最终有效参数预览；
10. RunPlan preview；
11. errors/warnings。

要求：

1. Deployment 只能保存 override；
2. Deployment 不重定义 schema；
3. 可覆盖参数来自 definition 的 editable_at；
4. 未 enabled 的 optional 参数不保存为 override；
5. RunPlan preview 显示 source map；
6. preview 与实际执行一致。

### 3.6 Instance 页面

展示：

1. 实例状态；
2. deployment；
3. container id；
4. health status；
5. logs；
6. actual Docker spec summary；
7. errors；
8. operation_id。

不提供运行参数编辑入口。

## 4. RuntimeParameterEditor 契约

RuntimeParameterEditor 必须支持：

1. schema/value/source 输入；
2. category 分组；
3. required 标识；
4. default-applied 标识；
5. inherited source 标识；
6. current-layer override 标识；
7. checked/enabled 修改；
8. disabled input 显示值；
9. advanced 折叠；
10. constraints 校验；
11. depends_on / show_when；
12. no watch → emit loop；
13. 保存后刷新稳定。

## 5. API 契约

### 5.1 参数编辑 API

API 返回应支持 UI 表达：

1. definitions；
2. current_values；
3. inherited_values；
4. overrides；
5. effective_values；
6. source；
7. editable_at；
8. category；
9. validation errors；
10. warnings。

### 5.2 RunPlan preview API

必须返回：

1. final image；
2. command；
3. args；
4. env；
5. mounts；
6. ports；
7. devices；
8. health check；
9. parameter_source_map；
10. warnings/errors。

### 5.3 Preflight API

必须返回结构化：

1. deployable；
2. errors；
3. warnings；
4. evidence；
5. requirement_results；
6. parameter_results。

### 5.4 Instance Logs API

必须使用真实 container id 获取日志，不能混用 instance id。日志接口应支持自动刷新和错误展示。

## 6. UI 已知修复点

必须处理：

1. RunnerConfigsPage 双入口；
2. legacy Docker editor 与 RuntimeParameterEditor 并存；
3. RuntimeParameterEditor 未 populate；
4. watch → emit 循环导致 OOM；
5. 只显示勾选框、不显示 disabled input；
6. 所有参数默认 checked；
7. 参数没有分类展示；
8. 不同页面展示不属于本层级的参数；
9. 保存后 schema/value/enabled/source 丢失；
10. clone 后 checked 扩大；
11. Deployment 覆盖不足；
12. RunPlan preview source 不可见。

## 7. E2E 断言

必须能断言：

1. Model API 不返回 Docker 参数用于编辑；
2. BackendRuntime API 返回模板参数；
3. NBR API 返回节点配置和 evidence；
4. Deployment API 返回可覆盖参数和 preview；
5. default 不导致 enabled；
6. required 不显示为用户 checked；
7. optional 默认不进入 override；
8. Deployment override 不复制 schema；
9. RunPlan source map 完整；
10. preview 与 Docker spec 一致。
