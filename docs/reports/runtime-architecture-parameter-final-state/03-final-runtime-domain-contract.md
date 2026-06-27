# Final Runtime Domain Contract

## 1. 总体边界

Runtime 领域模型需要形成稳定的分层结构：

```text
Backend / BackendVersion
        ↓
BackendRuntime
        ↓
NodeBackendRuntime
        ↓
Deployment
        ↓
ResolvedRunPlan
        ↓
ModelInstance
```

模型资产链路：

```text
ModelArtifact
        ↓
ModelLocation
        ↓
Deployment
        ↓
ResolvedRunPlan
```

节点与设备链路：

```text
Node
        ↓
Accelerator
        ↓
NodeBackendRuntime
        ↓
DeviceBinding
        ↓
ResolvedRunPlan
```

## 2. Backend

### 2.1 职责

Backend 表达推理后端类别，例如：

1. vLLM；
2. SGLang；
3. llama.cpp；
4. future backend。

### 2.2 可包含

1. name；
2. display_name；
3. description；
4. vendor/project 信息；
5. high-level protocol family；
6. 默认能力 profile 引用；
7. 文档链接或说明。

### 2.3 不可包含

1. GPU vendor；
2. 节点硬件；
3. Docker image 具体 tag；
4. 本机模型路径；
5. 节点 runtime 状态；
6. 部署实例参数；
7. NBR check 结果；
8. 容器 id；
9. 端口占用状态。

## 3. BackendVersion

### 3.1 职责

BackendVersion 表达某个 Backend 的版本能力和兼容范围。

### 3.2 可包含

1. backend_id；
2. version；
3. capability profile；
4. runtime requirements；
5. parameter schema；
6. recommended defaults；
7. supported model formats；
8. supported endpoints；
9. health check mode；
10. known warnings。

### 3.3 不可包含

1. 具体节点状态；
2. 某台机器的 image inspect 结果；
3. 某个部署的端口；
4. 某个本机模型路径；
5. GPU vendor 固定绑定；
6. container id；
7. instance id。

## 4. BackendRuntime

### 4.1 职责

BackendRuntime 是可复用运行模板。它绑定 BackendVersion 与运行方式，例如 Docker image、默认 args、默认 env、默认 health check、默认 resource controls。

### 4.2 可包含

1. backend_version_id；
2. image；
3. command；
4. default args；
5. default env；
6. default mounts；
7. default ports；
8. default health check；
9. parameter_schema_json；
10. parameter_values_json；
11. runtime_requirements_json；
12. capability_profile_json；
13. display_name；
14. clone metadata。

### 4.3 不可包含

1. 节点 check 状态；
2. 某个节点上的 image 是否存在；
3. 具体部署端口占用结果；
4. 具体 container id；
5. 本机模型路径；
6. NBR 专属设备选择；
7. Deployment 覆盖参数。

## 5. NodeBackendRuntime

### 5.1 职责

NodeBackendRuntime 是某个节点启用后的运行配置，也是唯一部署入口。

它负责把 BackendRuntime 落到某个 Node 上，并保存该节点的运行环境检查结果、节点级参数、设备绑定策略和状态。

### 5.2 可包含

1. node_id；
2. backend_runtime_id；
3. status；
4. check evidence；
5. image inspect evidence；
6. available accelerator evidence；
7. node-level parameter values；
8. enabled 参数；
9. node-level env override；
10. node-level device binding preference；
11. ready / warning / error；
12. copy-on-enable snapshot；
13. check timestamp。

### 5.3 状态

推荐状态：

```text
disabled
needs_check
checking
ready
ready_with_warnings
missing_image
failed
```

部署允许：

```text
ready
ready_with_warnings
```

部署禁止：

```text
disabled
needs_check
checking
missing_image
failed
```

### 5.4 不可包含

1. 自动创建逻辑；
2. deployment 专属模型路径；
3. deployment 专属端口覆盖；
4. 具体实例 container id；
5. Backend / BackendVersion 的硬件污染字段。

## 6. ModelArtifact

### 6.1 职责

ModelArtifact 表达模型资产的逻辑对象。

### 6.2 可包含

1. model name；
2. display name；
3. family；
4. architecture；
5. model type；
6. quantization；
7. format；
8. context length；
9. tokenizer metadata；
10. discovered_metadata_json；
11. user metadata。

### 6.3 discovered_metadata_json 边界

允许包含：

1. detected model family；
2. config.json 解析信息；
3. GGUF metadata；
4. tokenizer 类型；
5. embedding / generation / rerank 类型；
6. quantization；
7. architecture；
8. hidden size；
9. context length；
10. model format；
11. file list summary。

禁止包含：

1. 通用 catalog 中的本机路径；
2. BackendRuntime image；
3. NodeBackendRuntime 状态；
4. deployment port；
5. Docker args；
6. GPU device binding；
7. container id。

## 7. ModelLocation

### 7.1 职责

ModelLocation 表达模型在某个节点或存储位置的实际路径。

### 7.2 可包含

1. model_artifact_id；
2. node_id；
3. path；
4. location type；
5. existence check；
6. file count；
7. size；
8. scan evidence；
9. permission evidence；
10. last scanned time。

### 7.3 不可包含

1. BackendCapabilityProfile；
2. BackendVersion 参数；
3. Docker image；
4. deployment override；
5. runtime status。

## 8. Deployment

### 8.1 职责

Deployment 表达一次部署意图和部署级覆盖。

### 8.2 必须输入

1. model_artifact_id；
2. model_location_id；
3. node_backend_runtime_id；
4. deployment name；
5. deployment parameter overrides；
6. resource overrides；
7. port overrides；
8. mount overrides；
9. env overrides；
10. health check overrides。

### 8.3 必须快照

1. NBR snapshot；
2. BackendRuntime snapshot；
3. parameter schema snapshot；
4. parameter values snapshot；
5. model location snapshot；
6. deployment override snapshot；
7. resolved runplan snapshot。

### 8.4 API 约束

1. 只接受 `node_backend_runtime_id`；
2. 拒绝 `backend_runtime_id`；
3. 部署前必须 preflight；
4. 可选择自动运行 preflight；
5. API response 必须返回 errors/warnings；
6. 可返回 RunPlan preview。

## 9. ResolvedRunPlan

### 9.1 职责

ResolvedRunPlan 是最终执行权威。

### 9.2 必须包含

1. image；
2. command；
3. args；
4. env；
5. ports；
6. mounts；
7. devices；
8. accelerator ids；
9. device binding；
10. health check；
11. labels；
12. resource controls；
13. model path；
14. service endpoint；
15. source map；
16. warnings；
17. errors。

### 9.3 source map

RunPlan 中的关键字段必须能追踪来源：

```json
{
  "args.--gpu-memory-utilization": "deployment.parameter_values",
  "args.--model": "model_location.path",
  "env.CUDA_VISIBLE_DEVICES": "device_binding",
  "ports.8000": "deployment.port_override",
  "health_check.path": "backend_capability_profile.default_health_check"
}
```

## 10. ModelInstance

### 10.1 职责

ModelInstance 表达运行中的或曾经运行的实例状态。

### 10.2 必须包含

1. deployment_id；
2. node_id；
3. container_id；
4. status；
5. actual endpoint；
6. health status；
7. last error；
8. start time；
9. stop time；
10. logs reference；
11. operation_id。

### 10.3 状态建议

```text
pending
task_created
agent_claimed
container_created
container_started
health_checking
running
degraded
failed
stopping
stopped
exited
```

失败状态必须有结构化 reason。
