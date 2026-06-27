# RunPlan and Preflight Contract

## 1. 目标

RunPlan 与 Preflight 是 Runtime 架构最终闭环的核心。Preflight 负责判断是否可运行；ResolvedRunPlan 负责生成唯一最终执行 spec。

## 2. Preflight 契约

### 2.1 输入

Preflight 输入包括：

1. ModelArtifact；
2. ModelLocation；
3. BackendCapabilityProfile；
4. RuntimeRequirements；
5. BackendRuntime snapshot；
6. NodeBackendRuntime snapshot；
7. NodeBackendRuntime check evidence；
8. Deployment override；
9. user intent。

### 2.2 检查项

必须检查：

1. image exists；
2. image inspect evidence；
3. model path exists；
4. model format compatible；
5. required files；
6. parameter validation；
7. device availability；
8. Docker runtime availability；
9. ports availability；
10. mounts validity；
11. env validity；
12. health check validity；
13. resource controls validity；
14. warning vs blocking error。

### 2.3 输出

Preflight 输出：

```json
{
  "deployable": true,
  "errors": [],
  "warnings": [],
  "evidence": {},
  "requirement_results": [],
  "parameter_results": []
}
```

errors 阻断部署；warnings 不阻断 ready_with_warnings 部署。

### 2.4 禁止事项

1. 不信任前端传入的 image_present；
2. 不绕过 Server → Agent evidence；
3. 不把 warning 当 blocking error；
4. 不把 blocking error 降级为 warning；
5. 不静默吞掉参数错误。

## 3. ResolvedRunPlan 契约

ResolvedRunPlan 是最终执行权威。

### 3.1 输入

1. Deployment snapshot；
2. selected ModelArtifact / ModelLocation snapshot；
3. selected NodeBackendRuntime snapshot；
4. CapabilityProfile；
5. RuntimeRequirements；
6. parameter definitions；
7. parameter overrides；
8. system generated values。

### 3.2 输出

必须输出：

1. image；
2. command；
3. args；
4. env；
5. mounts；
6. ports；
7. devices；
8. health_check；
9. resource_controls；
10. labels；
11. parameter_source_map；
12. warnings；
13. errors；
14. docker_create_spec_preview。

### 3.3 parameter_source_map

每个最终参数必须包含来源：

```json
{
  "key": "max_model_len",
  "value": 8192,
  "target": "args",
  "rendered": ["--max-model-len", "8192"],
  "definition_ref": "backend_version:vllm.openai.latest:max_model_len",
  "source": "deployment_override",
  "source_chain": [
    "backend_version_default:4096",
    "backend_runtime_default:8192",
    "deployment_override:8192"
  ]
}
```

source 至少支持：

1. default；
2. model；
3. backend_version；
4. backend_runtime；
5. node_backend_runtime；
6. deployment_override；
7. system_generated；
8. runtime_detected。

## 4. 参数合成规则

合成顺序应清晰、可测试。建议顺序：

```text
ParameterDefinition default
→ BackendVersion effective defaults
→ BackendRuntime snapshot/default/override
→ NodeBackendRuntime snapshot/override/evidence
→ ModelArtifact / ModelLocation selected values
→ Deployment override
→ system_generated values
→ ResolvedRunPlan render
```

要求：

1. 未 enabled 的 optional override 不进入当前层；
2. required/default-applied 参数按 schema 规则生效；
3. enabled override 覆盖继承值；
4. target 决定渲染位置；
5. args 去重；
6. env 不混入 capabilities_json；
7. mounts 不重复；
8. ports 与 health check 一致；
9. devices 由 DeviceBinding 生成；
10. 每个输出项都能追踪来源。

## 5. Preview 与执行一致性

RunPlan preview 与 Agent Docker create spec 必须来自同一个 ResolvedRunPlan。

验收：

1. API 获取 RunPlan preview；
2. Agent 记录 Docker create spec；
3. E2E 对比 image / command / args / env / mounts / ports / devices / health_check；
4. 不一致时测试失败。

## 6. vLLM 映射

vLLM RunPlan 至少映射：

1. model path → `--model`；
2. host → `--host`；
3. port → `--port`；
4. gpu_memory_utilization → `--gpu-memory-utilization`；
5. max_model_len → `--max-model-len`；
6. dtype；
7. quantization；
8. tensor_parallel_size；
9. served_model_name；
10. health check。

## 7. SGLang 映射

SGLang RunPlan 至少映射：

1. model path → `--model-path`；
2. host → `--host`；
3. port → `--port`；
4. mem_fraction_static → `--mem-fraction-static`；
5. context_length → `--context-length`；
6. dtype；
7. tensor parallel；
8. health check。

## 8. llama.cpp 映射

llama.cpp RunPlan 至少映射：

1. GGUF path → `--model` or `-m`；
2. host → `--host`；
3. port → `--port`；
4. ctx_size → `--ctx-size`；
5. n_gpu_layers → `--n-gpu-layers` or `-ngl`；
6. batch；
7. ubatch；
8. health check。

## 9. DeviceBinding

RunPlan 输出设备绑定：

1. vendor-neutral AcceleratorIds；
2. resolved runtime device ids；
3. env mapping；
4. Docker devices / device requests；
5. vendor-specific mounts if required；
6. warnings/errors。

NVIDIA 和 MetaX 的具体绑定逻辑在 DeviceBinding / Agent runtime 层实现，不进入 Backend / BackendVersion。

## 10. Preflight 与 RunPlan 关系

Preflight 与 RunPlan 共享输入和解析规则：

1. Preflight 校验即将生成的 RunPlan；
2. RunPlan 不能绕过 Preflight 已发现的 blocking errors；
3. Preflight warnings 应进入 RunPlan warnings；
4. RunPlan errors 应可回传给 UI/API；
5. E2E 断言两者一致。

## 11. 验收

必须覆盖：

1. missing image；
2. missing model path；
3. invalid parameter；
4. unchecked optional 不进入 args；
5. deployment override 生效；
6. source map 正确；
7. preview 与 Docker spec 一致；
8. ready_with_warnings 可继续；
9. blocking error 阻断；
10. logs/evidence 可追踪。
