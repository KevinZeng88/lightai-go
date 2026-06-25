# Runtime Parameter Layering — Final Closeout

> Status: **REOPENED** (was: CLOSED)
> Date: 2026-06-25 (original: 2026-06-24)
> Branch: main
>
> ⚠️ **REOPENED 原因：**
> OOM fix 声称完成（syncing guard + try/finally + JSON.stringify shallow watch），但代码审查发现：
> - syncing guard 在 Vue 3 异步 flush 下实际无效
> - Phase 2 声称 "BackendRuntimesPage 移除重复内联参数编辑" 但 RunnerConfigsPage 仍有重复入口
>
> 修复计划详见：`docs/reports/repairs/runtime-architecture-parameter-2026-06-25/`

## 1. 问题背景

手工 Web 测试发现三个严重问题：
1. 模型编辑页显示 Docker runtime 参数（devices, network, shm 等），不符合参数分层设计
2. 运行模板/配置页 checkbox-only，非 boolean 字段缺少输入控件
3. 浏览器 Out of Memory

## 2. 参数分层结论

| 层级 | 字段范围 | 说明 |
|------|---------|------|
| Model / ModelArtifact | name, format, architecture, quantization, size, context_length, capabilities, serving parameter hints | 模型本身信息，不含 Docker 参数 |
| BackendVersion | default_args_schema, capabilities_json, health_check, default_args, default_port | 后端能力和版本定义，硬件无关 |
| BackendRuntime | image, docker_json, args_override_json, parameter_schema_json, parameter_values_json | 运行模板，给 NBR 拷贝用 |
| NodeBackendRuntime | config_snapshot_json (frozen), parameter_schema_json, parameter_values_json | 节点最终运行配置，RunPlan 主要来源 |
| Deployment | parameter_values_json, disabled_parameters_json, env_overrides_json | 部署级 override |

## 3. enabled/value 分离规则

- `enabled=false` 也必须保存 value
- 取消 enabled 不清空 value
- copy/clone 时 enabled 和 value 原样复制
- RunPlan 只使用 enabled=true 的参数

## 4. NBR 参数传播修复

**Bug**: `getBackendRuntimeJSON()` 不 SELECT `parameter_schema_json`/`parameter_values_json`，导致 NBR 创建时参数 schema 永远为 `"[]"`。

**Fix**: 补全 SELECT 字段 + fallback 到 BackendVersion `default_args_schema_json` + `buildDefaultParamValuesFromSchema()` 为 required 参数生成默认值。

## 5. default_env_json 脱敏边界

- **内部快照**: `buildRuntimeConfigSnapshot()` 直接从 DB 读取原始 `default_env_json`
- **API response**: `getBackendRuntimeJSON()` 使用 `redactRawJSON()` 脱敏
- **Web 展示**: 通过 API response，已脱敏

## 6. resolver Layer 2 参数值变量替换

**Bug**: Layer 2 (parameter values) 不调用 `substituteVars()`，`{{MODEL_CONTAINER_PATH}}` 模板变量不被替换。

**Fix**: 在 Layer 2 循环中添加 `substituteVars(valStr, vars)` 调用。

## 7. 容器标识 lifecycle 修复

**Bug**: `driver.Logs()` fallback 用 containerID (hex) 生成 `lightai-{containerID}`，但正确名称应为 `lightai-{instanceID[:12]}`。

**Fix**: `Logs()` 改为 `Logs(containerID, instanceID)` 双参数，fallback 用 instanceID 生成正确容器名。

## 8. preflight errors 展示修复

- 前端 `preflightErrorText()` 覆盖全部 12 种错误码
- 新增 `preflightErrorContext()` 展示完整 context
- i18n 新增 8 种错误码翻译

## 9. capabilities_json 结构化

**Bug**: `backendVersionCatalogDoc` 结构体缺少 `CapabilitiesJSON` 字段，YAML `capabilities_json` 不被加载。BackendVersion 只有 API capabilities 列表，缺少 `supported_formats`/`supported_tasks`/`model_path_modes`。

**Fix**: 添加 `CapabilitiesJSON` 字段 + `firstNonNil()` 优先使用 + `isEmptyJSON()` 处理 `"null"`。

## 10. SGLang timeout 根因与修复

**根因**: BackendVersion `health_check` 缺少 `startup_timeout_seconds`，平台默认 30 秒。SGLang 启动需要约 50 秒（模型加载 + CUDA graph capture + compilation）。

**Fix**: 三个 BackendVersion YAML 添加 `startup_timeout_seconds: 120`。

## 11. vLLM BackendVersion `default_args` 传播

**Bug**: `buildRuntimeConfigSnapshot()` 不合并 BackendVersion `default_args`，导致 NBR snapshot 缺少 `--host 0.0.0.0` 等默认参数。

**Fix**: snapshot 构建时若 `args_override_json` 为空，从 BackendVersion `default_args` 合并。

## 12. E2E 结果

| 后端 | 结果 | 证据 |
|------|------|------|
| vLLM default | PASS | `evidence/runtime-param-layering/vllm-*.json` |
| vLLM modified | PASS | `evidence/runtime-param-layering/vllm-*.json` |
| SGLang | PASS | `evidence/runtime-param-layering/sglang-*.json` |
| llama.cpp | PASS | `evidence/runtime-param-layering/llamacpp-*.json` |

## 13. 测试结果

```
npm run build: PASS
npm test: PASS (129 tests)
go test ./internal/...: PASS
go build ./cmd/server/...: PASS
go build ./cmd/agent/...: PASS
bash -n scripts: PASS
git diff --check: PASS
```

## 14. Evidence 路径

```
docs/reports/phase-3/evidence/runtime-param-layering/
```

## 15. OOM 修复说明

代码层修复：RuntimeParameterEditor `syncing` guard + `try/finally` + `JSON.stringify` shallow watch。浏览器长时间手工操作需人工在 UI 中验证；本轮已通过 API/代码/单测验证日志上限、polling 清理、watch 防循环。
