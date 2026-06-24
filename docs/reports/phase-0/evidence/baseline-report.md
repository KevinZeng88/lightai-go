# Phase 0 Baseline Report

> Date: 2026-06-24
> Commit: c2a2f2b

## E2E Results

| Backend | Result | Evidence |
|---------|--------|----------|
| vLLM default | PASS | vllm-e2e.log |
| vLLM modified | PASS | vllm-e2e.log |
| SGLang | PASS | sglang-e2e.log |
| llama.cpp | PASS | llamacpp-e2e.log |

## RunPlan Baseline

| Backend | Args | Port |
|---------|------|------|
| vLLM | --model /models/... --host 0.0.0.0 --port 8000 | 8004:8000 |
| SGLang | --model-path /models/... --host 0.0.0.0 --port 30000 | 8005:30000 |
| llama.cpp | -m /models/... --host 0.0.0.0 --port 8080 | 8002:8080 |

## 当前参数链路问题清单

### P1: required 参数 checkbox 可取消勾选
- **位置**: RuntimeParameterEditor.vue backendParams 渲染
- **影响**: required 参数（host, port, model）用户可取消勾选
- **Phase**: Phase 1

### P2: Deployment override 缺少 backendSchema
- **位置**: ModelDeploymentsPage.vue
- **影响**: parameter_values_json 被清空为 []
- **状态**: 已修复（上一轮），需 Phase 1 验证

### P3: BackendRuntimesPage 重复编辑入口已消除
- **状态**: 已修复（上一轮），需 Phase 2 验证

### P4: vLLM default_args 缺少 --host/--port
- **状态**: 已修复（上一轮），host/port 改由 schema parameter_values_json 提供

### P5: SGLang health timeout 不足
- **状态**: 已修复（上一轮），startup_timeout_seconds 改为 120

### P6: capabilities_json 结构化字段缺失
- **状态**: 已修复（上一轮），添加了 supported_formats/supported_tasks/model_path_modes

### P7: NBR parameter_schema_json 传播丢失
- **状态**: 已修复（上一轮），getBackendRuntimeJSON 补全 SELECT

### P8: buildRuntimeConfigSnapshot env 脱敏
- **状态**: 已修复（上一轮），快照直接从 DB 读取原始值

### P9: Layer 3 deployment override 模板替换缺失
- **位置**: resolver.go Layer 3
- **影响**: {{container_port}} 不会被替换
- **Phase**: Phase 1

### P10: 参数分组未实现
- **位置**: RuntimeParameterEditor.vue
- **影响**: 所有参数平铺显示
- **Phase**: Phase 2

### P11: extra_args 冲突检测未实现
- **位置**: resolver.go
- **影响**: extra_args 可重复 host/port/model
- **Phase**: Phase 3

### P12: vendor 隔离需确认
- **状态**: catalog 层面已隔离，需 Phase 4 验证

### P13: LLAMA_ARG_HOST benign warning
- **状态**: log_classifier 已正确分类
