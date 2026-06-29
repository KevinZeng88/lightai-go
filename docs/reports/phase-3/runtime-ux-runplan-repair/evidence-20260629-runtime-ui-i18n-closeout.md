# Runtime UI i18n Closeout Evidence - 2026-06-29

## Scope

Runtime UI recheck covered the shared vLLM / SGLang / llama.cpp flow:

- Runtime templates / backend runtimes
- Node runtime configs and check-request status
- Model deployment wizard
- RunPlan preview and Docker command preview
- Config edit parameter labels and tooltips
- API error code to frontend i18n rendering

## Root Cause

1. API errors returned only raw English `error` strings. Frontend `ApiError.message` propagated that string into toasts.
2. Runtime status UI rendered `status` and `status_reason` values directly or fell back to raw values when translations were missing.
3. Deployment wizard step transition regressed when `nextStep()` called `doPreview()` from service step `2`; `doPreview()` only advanced when the active step was already `3`, so the click completed without visible navigation.
4. Config edit fields preferred backend-provided English labels/help instead of stable i18n keys plus technical metadata.

## Verification Matrix

| Area | vLLM | SGLang | llama.cpp | Evidence |
| --- | --- | --- | --- | --- |
| Short parameter labels | `显存比例`, `最大上下文` | `静态显存比例`, `上下文长度` | `GPU 层数`, `上下文长度` | `web/tests/runtimeBoundaryUi.test.mjs`; locale keys in `web/src/locales/zh-CN.ts` / `en-US.ts` |
| Tooltip technical metadata | `--gpu-memory-utilization`, `--max-model-len` | `--mem-fraction-static`, `--context-length` | `-ngl`, `--ctx-size` | `internal/server/configedit/project.go`; `ConfigField.vue` tooltip |
| Status localization | `ready`, `needs_check`, `ready_with_warnings`, `missing_image`, `failed` translated via `status.*` | Same shared helper | Same shared helper | `web/src/utils/status.ts`; `runtimeBoundaryUi.test.mjs` |
| Check result localization | Success toast uses localized “节点运行配置检测通过，镜像：{image}” | Same shared wizard/page path | Same shared wizard/page path | `NodeRuntimeConfigWizard.vue`; `RunnerConfigsPage.vue` |
| Duplicate display name error | Backend returns `code=display_name_exists`; UI renders `apiErrors.display_name_exists` | Same shared API helper | Same shared API helper | `internal/server/api/error_code_test.go`; `web/src/utils/__tests__/apiErrors.test.ts` |
| Deployment wizard service next | Step 2 validates ports and advances to step 3 | Same shared component | Same shared component | `web/src/components/deployments/__tests__/DeploymentWizard.steps.test.ts` |

## Test Results

- `go test ./internal/server/api ./internal/server/configedit ./internal/server/semanticconfig`: PASS
- `go test ./...`: PASS
- `npm test`: PASS, including i18n missing key audit and runtime raw string leak audit
- `npm run test:unit`: PASS, 10 files / 62 tests
- `npm run build`: PASS; Vite reported existing Rollup pure annotation and chunk size warnings only

## Closeout

No unresolved problems remain from this recheck. RUR-002 is recorded as `FIXED` in `open-issues-closeout.md`.
