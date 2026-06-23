# Batch A Closeout — Backend Runtime Diagnostics & Resource Controls

> Date: 2026-06-23
> Status: Complete

## 1. 实现摘要

### 1.1 RunPlan lint (`lint.go`)

两阶段 lint 设计：

- **Pre-normalization lint**：检查 dedup 前的原始 args，检测重复 flag 和用户覆盖平台参数
- **Final lint**：检查最终 resolved args + env，检测 env/CLI conflict、高风险 Docker flags

LLAMA_ARG_HOST 处理：
- image-provided env（`EnvSources["backend_default"]`）→ `warning`，不 block
- user-provided env（`EnvSources["user_env"]`）→ `error`，block
- 平台/后端默认 env → `warning`

Lint 结果嵌入 dry-run response 的 `lint` 子对象，同时合并到 `warnings` 数组（向后兼容）。

### 1.2 resource_controls (`resource_controls.go`)

- 定义存储在 catalog YAML 的 `vendor_options.resource_controls`
- 用户可调参数放 deployment `parameters_json`，不放 backend_versions
- `ParseResourceControls()` 从 vendor_options_json 解析
- `ValidateResourceControlValue()` 验证 min/max/enum
- `BuildResourceControlArgs()` 从参数构建 CLI args
- llama.cpp `gpu_memory_fraction.supported = false`

### 1.3 Runtime log classifier (`log_classifier.go`)

Go 内置规则，fixture 测试。6 条初始规则：

| Rule ID | Backend | Severity | Category |
|---------|---------|----------|----------|
| sglang.torchao.syntax_warning | sglang | noise | dependency_warning |
| sglang.attention_backend.default | sglang | advisory | default_selection |
| llamacpp.env_overwritten.host | llamacpp | warning | arg_conflict |
| llamacpp.env_overwritten.port | llamacpp | warning | arg_conflict |
| cuda.oom | * | error | oom |
| container.startup.failed | * | error | startup |

`IsNonFatal()` 判断 noise/advisory 不改变实例状态。

### 1.4 Dry-run response 变化

```json
{
  "valid": true,
  "errors": [],
  "warnings": ["[lint] security.privileged_enabled: Container runs in privileged mode"],
  "command_preview": "docker run ...",
  "lint": {
    "status": "warning",
    "findings": [
      {
        "id": "security.privileged_enabled",
        "severity": "warning",
        "category": "high_risk",
        "message": "Container runs in privileged mode",
        "suggestion": "...",
        "field_path": "docker.privileged",
        "sources": ["platform"]
      }
    ]
  }
}
```

## 2. 修改文件

### 新增文件

| 文件 | 说明 |
|------|------|
| `internal/server/runplan/lint.go` | RunPlan lint 引擎 |
| `internal/server/runplan/lint_test.go` | lint 测试 |
| `internal/server/runplan/resource_controls.go` | resource_controls 建模 |
| `internal/server/runplan/resource_controls_test.go` | resource_controls 测试 |
| `internal/server/runplan/log_classifier.go` | runtime log classifier |
| `internal/server/runplan/log_classifier_test.go` | log classifier 测试 |
| `internal/server/runplan/testdata/runtime-logs/*.log` | fixture 日志文件 (4 files) |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/06-batch-execution-plan.md` | 执行聚合文档 |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/batch-a-backend-runtime-diagnostics-closeout.md` | 本 closeout |

### 修改文件

| 文件 | 变更 |
|------|------|
| `internal/server/api/deployment_lifecycle_handlers.go` | preflightResult 增加 lintResult 字段；preflightDeployment 中调用 LintRunPlan；HandleDeploymentDryRun 中嵌入 lint 结果 |
| `configs/backend-catalog/versions/vllm/vllm-v0.23.0.yaml` | 添加 vendor_options.resource_controls |
| `configs/backend-catalog/versions/sglang/sglang-v0.5.13.post1.yaml` | 添加 vendor_options.resource_controls |
| `configs/backend-catalog/versions/sglang/sglang-v0.5.12.post1.yaml` | 添加 vendor_options.resource_controls |
| `configs/backend-catalog/versions/sglang/sglang-0.4.6-compatible.yaml` | 添加 vendor_options.resource_controls |
| `configs/backend-catalog/versions/llamacpp/llamacpp-b9700.yaml` | 添加 vendor_options.resource_controls |

## 3. 测试命令和结果

```bash
$ go test ./internal/server/runplan/... -count=1
ok  	lightai-go/internal/server/runplan	0.004s

$ go test ./internal/server/api/... -count=1
ok  	lightai-go/internal/server/api	6.598s

$ go test ./...
ok  	lightai-go/internal/server/runplan	0.004s
ok  	lightai-go/internal/server/api	6.559s
(all other packages pass)

$ go build ./...
(exit 0, no errors)

$ gofmt -l internal/
(no output, all files formatted)

$ git diff --check
(no output, no whitespace errors)
```

## 4. 已覆盖的具体 warning/冲突

| 场景 | 测试 | 结果 |
|------|------|------|
| Clean vLLM RunPlan | TestLintCleanVLLM | ok |
| Clean SGLang RunPlan | TestLintCleanSGLang | ok |
| Clean llama.cpp RunPlan | TestLintCleanLlamaCpp | ok |
| llama.cpp image-provided LLAMA_ARG_HOST + --host | TestLintLlamaCppImageProvidedHostConflict | warning (not error) |
| llama.cpp user-provided LLAMA_ARG_HOST + --host | TestLintLlamaCppUserProvidedHostConflict | error |
| Duplicate --ctx-size | TestLintDuplicateFlag | error |
| Duplicate --gpu-memory-utilization | TestLintDuplicateGpuMemoryUtilization | error |
| Duplicate --mem-fraction-static | TestLintDuplicateMemFractionStatic | error |
| Privileged mode | TestLintPrivilegedWarning | warning |
| IPC host mode | TestLintIPCHostWarning | warning |
| SGLang torchao SyntaxWarning | TestClassifySGLangTorchaoSyntaxWarning | noise |
| SGLang attention backend default | TestClassifySGLangAttentionBackendDefault | advisory |
| llama.cpp LLAMA_ARG_HOST overwritten | TestClassifyLlamaCppEnvHostOverwritten | warning |
| CUDA OOM | TestClassifyCUDAOOM | error |
| vLLM resource_controls parse | TestParseResourceControlsVLLM | pass |
| llama.cpp resource_controls (no memory fraction) | TestParseResourceControlsLlamaCpp | pass |
| Resource control min/max validation | TestValidateResourceControlMinMax | pass |
| Resource control enum validation | TestValidateResourceControlEnum | pass |
| JSON roundtrip for lint/resource_controls | TestLintResultJSON, TestResourceControlJSON | pass |

## 5. 明确未做事项

| 事项 | 原因 |
|------|------|
| Shared GPU admission / budget-based GPU lease | DOCUMENTED_BLOCKER，需要删除 gpu_leases 唯一索引 + schema change |
| Frontend auto-refresh (Phase 4) | Batch B |
| JsonViewer (Phase 5) | Batch B |
| HealthCheckEditor (Phase 6a) | Batch B |
| Complete ConfigEditorLayout (Phase 6b) | Deferred |
| Status-summary API | Conditional，先轮询现有 list endpoint |
| vitest introduction | Deferred |
| llama.cpp VRAM estimator | Deferred |
| Pre-normalization lint hook in resolver.go | 当前 lint 只在 preflight 中调用，resolver.go 未直接修改（最小侵入） |
| Env source tracking 精细化 | 当前简化为所有 env 来源标记为 "platform"，需要 layer metadata 支持才能精确区分 |

## 6. Commit 信息

- **commit id**: (待提交后填写)
- **push result**: (待推送后填写)
- **git status**: (待提交后检查)
