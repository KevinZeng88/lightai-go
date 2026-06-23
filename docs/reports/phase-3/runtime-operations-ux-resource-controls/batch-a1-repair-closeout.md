# Batch A.1 Repair Closeout — Backend Runtime Diagnostics

> Date: 2026-06-23
> Status: Complete

## 1. 修复了哪些 Gap

### GAP-1: resource_controls 接入 RunPlan 生成链路

**Before**：
- `BuildResourceControlArgs()` 存在于 `resource_controls.go` 但从未被 `buildArgs()` 调用。
- deployment `parameters_json` 中的 `gpu_memory_fraction` 不会生成 `--gpu-memory-utilization`。
- resource_controls 建模是"死代码"。

**After**：
- `VersionInfo` 新增 `VendorOptionsJSON` 字段。
- `buildArgs()` 在 Layer 4（ParameterDefs）之后新增 Layer 4b：解析 `vendor_options_json` 的 `resource_controls`，调用 `BuildResourceControlArgs()`。
- 避免与 ParameterDefs 重复：检查 existingFlags，跳过已存在的 flag。
- `preflightDeployment()` 从 DB 读取 `vendor_options_json` 并传入 `VersionInfo`。

**Evidence**：
- `TestResolveVLLMResourceControlsGPUFraction`：`gpu_memory_fraction=0.7` → args 包含 `--gpu-memory-utilization 0.7` ✓
- `TestResolveSGLangResourceControlsMemFraction`：`gpu_memory_fraction=0.65` → args 包含 `--mem-fraction-static 0.65` ✓
- `TestResolveSGLangResourceControlsAttentionBackend`：`attention_backend=triton` → args 包含 `--attention-backend triton` ✓
- `TestResolveLlamaCppNoFakeMemoryFraction`：`gpu_memory_fraction=0.8` → 不生成任何 memory fraction 参数 ✓
- `TestResolveLlamaCppResourceControlsGpuLayers`：`gpu_layers=99` → args 包含 `--n-gpu-layers 99` ✓
- `TestResolveResourceControlsNoDuplicateWithParameterDefs`：max_model_len 在 ParameterDefs 和 resource_controls 中都定义 → 只生成 1 次 `--max-model-len` ✓

### GAP-3: runtime log classifier 接入 API

**Before**：
- `log_classifier.go` 只有库函数和测试，无 API 入口。
- `HandleGetNodeRunPlanLogs` 返回 logs/stdout/stderr 但不分类。

**After**：
- `HandleGetNodeRunPlanLogs` 在返回响应中新增 `classified_log_events` 字段。
- 使用 `NewRuntimeLogClassifier().ClassifyLogText()` 对 logs + stderr 进行分类。
- 返回结构：`[{rule_id, severity, category, message, suggestion, raw_line, occurrences}]`

**Evidence**：
- `TestNodeRunPlanLogsClassifiesLogEvents`：输入包含 SGLang attention backend default 和 llama.cpp LLAMA_ARG_HOST overwritten → 返回的 `classified_log_events` 包含 `sglang.attention_backend.default` 和 `llamacpp.env_overwritten.host` ✓

## 2. 修改文件

| 文件 | 变更 |
|------|------|
| `internal/server/runplan/resolver.go` | `VersionInfo` 新增 `VendorOptionsJSON`；`buildArgs()` 新增 Layer 4b resource_controls 集成 |
| `internal/server/runplan/resolver_test.go` | 新增 7 个 resource_controls 端到端 resolver 测试 |
| `internal/server/api/deployment_lifecycle_handlers.go` | preflightResult 新增 `bvVendorOptions`；DB 查询读取 `vendor_options_json`；`VersionInfo` 传入 `VendorOptionsJSON`；`HandleGetNodeRunPlanLogs` 返回 `classified_log_events` |
| `internal/server/api/node_run_plan_logs_test.go` | 新增 `TestNodeRunPlanLogsClassifiesLogEvents` |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/batch-a-backend-runtime-diagnostics-closeout.md` | 更新 commit id |

## 3. 测试命令和结果

```
$ go test ./internal/server/runplan/... -count=1
ok  	lightai-go/internal/server/runplan	0.004s

$ go test ./internal/server/api/... -count=1
ok  	lightai-go/internal/server/api	6.662s

$ go test ./...
(all packages pass)

$ go build ./...
(exit 0)

$ gofmt -l internal/
(no output)

$ git diff --check
(no output)
```

## 4. resource_controls 最终进入 RunPlan args 的 evidence

```
TestResolveVLLMResourceControlsGPUFraction:
  args = [--gpu-memory-utilization 0.7]  ← gpu_memory_fraction 从 parameters_json 映射

TestResolveSGLangResourceControlsMemFraction:
  args = [--mem-fraction-static 0.65]  ← gpu_memory_fraction 从 parameters_json 映射

TestResolveSGLangResourceControlsAttentionBackend:
  args = [--attention-backend triton]  ← attention_backend 从 parameters_json 映射

TestResolveLlamaCppNoFakeMemoryFraction:
  args 中不包含 --gpu-memory-utilization / --mem-fraction-static  ← supported=false 正确跳过
  args 包含 --ctx-size 4096  ← ctx_size 正确映射

TestResolveResourceControlsNoDuplicateWithParameterDefs:
  --max-model-len 只出现 1 次  ← existingFlags 去重生效
```

## 5. classified_log_events API 出口

**API**: `GET /api/v1/node-run-plans/{id}/logs`
**Handler**: `HandleGetNodeRunPlanLogs`
**新增字段**: `classified_log_events` (array)

```json
{
  "id": "...",
  "logs": "...",
  "classified_log_events": [
    {
      "rule_id": "sglang.attention_backend.default",
      "severity": "advisory",
      "category": "default_selection",
      "message": "SGLang used the default attention backend (flashinfer).",
      "suggestion": "...",
      "raw_line": "Attention backend not specified...",
      "occurrences": 1
    }
  ]
}
```

## 6. 仍保留的 Limitation

| Limitation | 说明 |
|------------|------|
| Pre-normalization lint 未接入 resolver | `LintInput.PreDedupArgs` 从未被设置，pre-normalization lint 在真实链路中不生效。v1 limitation。 |
| Env source tracking 简化 | 所有 env 标记为 "platform"，user-provided env conflict 可能降级为 warning。 |
| lint error 不改变 dry-run valid | 诊断优先策略，lint errors 合并到 warnings 数组。 |
| Shared GPU admission | DOCUMENTED_BLOCKER，需要 schema change。 |
| log classifier 仅在 logs API 中 | model test result API 未包含 classified_log_events。 |

## 7. 未做事项

- Shared GPU admission / budget-based GPU lease
- Frontend auto-refresh / JsonViewer / HealthCheckEditor
- Complete ConfigEditorLayout
- Status-summary API
- vitest introduction
- llama.cpp VRAM estimator
- Model test result API 的 classified_log_events（可在 Batch B 中添加）

## 8. Commit 信息

- **commit id**: (待提交后填写)
- **push result**: (待推送后填写)
- **git status**: (待提交后检查)
