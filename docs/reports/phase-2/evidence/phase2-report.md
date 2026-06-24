# Phase 2 Report: UI 分组、唯一编辑入口、基础可用性

> Date: 2026-06-25

## 修复内容

1. **BackendVersion schema 添加 group/label 字段**: vLLM/SGLang/llama.cpp 的 `default_args_schema` 都添加了 `group` 和 `label` 字段
2. **RuntimeParameterEditor 按 group 分组渲染**: 新增 `groupedBackendParams` computed，按 `group` 字段分组显示参数
3. **BackendParamDef 接口扩展**: 添加 `label` 和 `group` 可选字段
4. **BackendParam 接口扩展**: 添加 `label` 和 `group` 字段
5. **分组 CSS**: 添加 `param-group` 和 `param-group-title` 样式

## 分组策略

| Group | 参数示例 |
|-------|---------|
| startup | model path, host, port, served_model_name |
| performance | gpu_memory_utilization, max_model_len, context_length, swap_space |
| parallelism | tensor_parallel_size, pipeline_parallel_size, dp |
| security | trust_remote_code |
| observability | enable_metrics, log_level |
| advanced | download_dir |

## E2E 结果

| Backend | Result |
|---------|--------|
| vLLM default | PASS |
| vLLM modified | PASS |
| SGLang | PASS |
| llama.cpp | PASS |
