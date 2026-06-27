# UI and API Contract

## 1. 目标

UI 和 API 必须围绕同一套领域模型、参数体系、Preflight、RunPlan 和状态机工作。前端展示不能形成第二套业务规则。

## 2. 页面职责

### 2.1 模型页面

职责：

1. 管理 ModelArtifact；
2. 管理 ModelLocation；
3. 展示模型 metadata；
4. 展示模型格式；
5. 展示模型能力；
6. 展示扫描结果；
7. 展示路径检查结果。

不展示：

1. Docker image；
2. Docker args；
3. GPU memory utilization；
4. backend runtime args；
5. NodeBackendRuntime check 状态；
6. deployment 端口。

### 2.2 运行配置页面

职责：

1. 管理 BackendRuntime；
2. 编辑运行模板参数；
3. 显示 BackendVersion capability；
4. 显示 RuntimeRequirements；
5. 显示默认 command/args/env/ports/mounts/health check；
6. 支持 clone；
7. 支持参数 schema/value round-trip。

### 2.3 节点运行配置页面

职责：

1. 管理 NodeBackendRuntime；
2. enable / disable；
3. check-request；
4. 显示 image inspect；
5. 显示节点设备；
6. 显示 ready / ready_with_warnings / missing_image / failed；
7. 编辑节点级参数；
8. 显示 check evidence；
9. 作为部署入口候选。

### 2.4 部署页面

职责：

1. 选择 ModelArtifact / ModelLocation；
2. 选择 ready 或 ready_with_warnings 的 NodeBackendRuntime；
3. 编辑部署级覆盖参数；
4. 编辑端口；
5. 编辑附加 mounts；
6. 编辑 health check；
7. 运行 preflight；
8. 显示 RunPlan preview；
9. 创建部署；
10. 启动部署。

### 2.5 实例页面

职责：

1. 显示 ModelInstance；
2. 显示运行状态；
3. 显示 endpoint；
4. 显示 health；
5. 显示 last error；
6. 显示 operation_id；
7. 显示 logs；
8. 支持 stop；
9. 自动刷新。

## 3. RuntimeParameterEditor 契约

### 3.1 输入

RuntimeParameterEditor 接收：

1. parameter schema；
2. parameter values；
3. layer；
4. validation result；
5. disabled/read-only flag；
6. source map；
7. group order。

### 3.2 输出

RuntimeParameterEditor emit：

1. values；
2. dirty state；
3. validation errors；
4. enabled changes；
5. value changes。

### 3.3 行为

1. disabled input 仍显示 value；
2. enabled checkbox 控制当前层覆盖；
3. value 不因 enabled=false 丢失；
4. required 参数显示 required；
5. invalid value 即时显示；
6. 保存后刷新不丢 schema/value；
7. clone 后保留 enabled/value；
8. watch 不得形成 emit 循环；
9. 深拷贝避免修改 props；
10. 只在用户编辑或明确初始化时 emit。

## 4. API 契约

### 4.1 BackendRuntime API

必须返回：

1. id；
2. backend_version_id；
3. image；
4. parameter_schema_json；
5. parameter_values_json；
6. runtime_requirements_json；
7. capability_profile_json；
8. health_check_json；
9. created_at / updated_at。

### 4.2 NodeBackendRuntime API

必须返回：

1. id；
2. node_id；
3. backend_runtime_id；
4. status；
5. status_reason；
6. parameter_schema_json；
7. parameter_values_json；
8. check_evidence_json；
9. warnings；
10. errors；
11. last_checked_at。

### 4.3 Deployment API

必须接受：

1. model_artifact_id；
2. model_location_id；
3. node_backend_runtime_id；
4. parameter_values_json；
5. port overrides；
6. mount overrides；
7. health check overrides。

必须拒绝：

1. backend_runtime_id；
2. missing node_backend_runtime_id；
3. NBR 不可部署状态；
4. invalid parameter；
5. missing model location。

### 4.4 Preflight API

必须返回：

1. deployable；
2. errors；
3. warnings；
4. evidence；
5. optional RunPlan preview。

### 4.5 RunPlan preview API

必须返回：

1. full ResolvedRunPlan；
2. source map；
3. warnings；
4. errors；
5. normalized Docker spec preview；
6. diff evidence if available。

## 5. 状态展示

NodeBackendRuntime 状态显示：

```text
disabled
needs_check
checking
ready
ready_with_warnings
missing_image
failed
```

Deployment / Instance 状态显示：

```text
created
preflight_failed
task_created
agent_claimed
container_created
container_started
health_checking
running
failed
stopping
stopped
exited
```

UI 必须避免显示“未知”作为最终错误。缺少 i18n key 时需要补齐。

## 6. ready_with_warnings

规则：

1. 可部署；
2. UI 显示 warning；
3. preflight 继续检查；
4. start 可继续；
5. warning 进入 evidence；
6. API response 明确 warnings。

## 7. needs_check / missing_image / failed

规则：

1. 不可部署；
2. UI 禁用选择；
3. 提供 check 或修复入口；
4. 显示具体原因；
5. API 返回 blocking error。

## 8. 日志契约

日志 API 必须使用正确 container id。

要求：

1. instance 记录 container_id；
2. logs endpoint 使用 container_id；
3. 不把 instance_id 当 container_id；
4. 容器不存在时返回结构化错误；
5. UI 显示可读错误；
6. 日志支持刷新；
7. 日志保留 operation_id。

## 9. 验收点

必须通过：

1. Web build；
2. Web test；
3. 参数编辑器单测；
4. API schema/value round-trip 测试；
5. Deployment 创建测试；
6. Preflight 测试；
7. RunPlan preview 测试；
8. ready_with_warnings UI/API 测试；
9. logs container id 测试。
