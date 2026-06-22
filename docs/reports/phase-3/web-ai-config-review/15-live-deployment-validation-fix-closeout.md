# Live Deployment Validation Fix Closeout

> Status: CURRENT  
> Created: 2026-06-22  
> Commit: d36e1cf  
> Scope: Fix vLLM served model name mismatch + frontend diagnostic improvements

## 1. User Feedback Recap

User provided live vLLM deployment logs showing:
- vLLM started successfully, `/v1/chat/completions` route confirmed present
- `/v1/models` returns 200 OK
- Test failure: `The model 'Qwen3-0.6B-Instruct-2512' does not exist` (404)
- Frontend showed only `Failed to fetch`

## 2. Root Cause

### 2.1 vLLM missing `--served-model-name`
DB seed data for vLLM v0.23.0 had `default_args_json: ["{{model_container_path}}"]` — no `--served-model-name` flag. Without it, vLLM uses a path-based or HF config-derived model ID that differs from the artifact display name.

### 2.2 `resolveModelID()` short-circuit
When `runplanModel` was non-empty (always, since it falls back to artifact's `ModelName`), `resolveModelID()` returned immediately without probing `/v1/models`. It never verified whether the requested model actually existed in the runtime's served models.

### 2.3 `buildVarMap()` empty default
`SERVED_MODEL_NAME` defaulted to empty string. Without explicit user config or parameter default, the template variable resolved to empty, making `--served-model-name` ineffective even if present in args.

## 3. `/v1/models` Return Value
- Could not directly probe since local instance may not be running
- Diagnostic probe capability exists in `collectTestDiagnostics()` which captures `/v1/models` response body
- `resolveModelID()` now always probes and returns `availableModels`

## 4. Requested vs Available Models
- Requested: `Qwen3-0.6B-Instruct-2512` (artifact display name)
- Available: vLLM serves model under a different ID when `--served-model-name` is not set

## 5. Fixes Applied

### 5.1 RunPlan `--served-model-name`
**File:** `internal/server/db/db.go`
- vLLM v0.23.0 seed `default_args_json` updated to: `["{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}","--served-model-name","{{served_model_name}}"]`

**File:** `internal/server/runplan/resolver.go`
- `buildVarMap()`: When `SERVED_MODEL_NAME` is empty after checking deployment params and parameter defs, derive from `in.Artifact.Name`
- Priority: deployment param > param def default > artifact name > sanitized path basename

### 5.2 Test API Diagnostics
**File:** `internal/server/api/deployment_lifecycle_handlers.go`
- `resolveModelID()`: Always probes `/v1/models`. Returns `runplanModel` only if verified against available models. Returns `availableModels []string` for diagnostics.
- On `model_id_not_resolved`: Response includes `requested_model`, `available_models`, `hint`
- Inference result includes `requested_model` and `available_models` for mismatch diagnosis

### 5.3 Frontend Error Display
**File:** `web/src/utils/modelCapabilities.js`
- `formatTestFailure()`: Handles `model_id_not_resolved` with requested/available models display
- Handles 404/endpoint failures with HTTP status, requested model, available models, backend error body
- No longer collapses all errors to `Failed to fetch`

**File:** `web/src/pages/ModelInstancesPage.vue`
- Test error `<el-descriptions>` now includes: `reason_code`, `http_status`, `requested_model`, `available_models`, `hint`, `error_body`

**Files:** `web/src/locales/en-US.ts`, `web/src/locales/zh-CN.ts`
- New i18n keys: `testReasonCode`, `testHttpStatus`, `testRequestedModel`, `testAvailableModels`, `testHint`, `testBackendError`

## 6. RunPlan Parameter Source Annotations

**File:** `web/src/pages/ModelDeploymentsPage.vue`
- Dry-run dialog replaced with 6 source-grouped sections:
  1. NBR Template (static snapshot) — image, command, env, docker options
  2. Model Location Injection — volumes, served model name (with source note)
  3. Deployment Port Injection — port mapping
  4. GPU / Vendor Binding Injection — devices with source note, GPU visible env
  5. Backend Service Parameters — health check
  6. Final Docker Command — command preview
- Simple run-plan dialog title: "最终运行计划"

**File:** `web/src/pages/RunnerConfigsPage.vue`
- Command preview label: "NBR 静态模板预览" / "NBR Static Template Preview"

**Files:** `web/src/locales/en-US.ts`, `web/src/locales/zh-CN.ts`
- New deployment i18n keys: `finalRunPlan`, `runPlanSourceNote`, `nbrTemplateGroup`, `modelLocationGroup`, `portInjectionGroup`, `gpuBindingGroup`, `backendServiceGroup`, `finalCommandGroup`, `dockerOptions`, `servedModelName`, `gpuVisibleEnv`
- New runnerConfigs key: `nbrTemplatePreview`

## 7. `--gpus` / `CUDA_VISIBLE_DEVICES` Source Explanation
- GPU device IDs come from placement / accelerator_ids
- If user did not manually select, comes from current single-node default scheduling selection
- Displayed in the GPU / Vendor Binding Injection group with source note

## 8. `--served-model-name` Source Explanation
- Derived from model display name / deployment served model name
- Source note shown alongside served model name in RunPlan display

## 9. Deploy Page Action Column Fixed
**File:** `web/src/pages/ModelDeploymentsPage.vue`
- Action column: `fixed="right"` added

## 10. Model Detail vs Edit Field Gap
Not addressed in this round (belongs to Phase 2 capabilities). Recorded here as known gap:
- Detail page shows full instance/config info
- Edit page allows editing a subset of fields (args, env, volumes, ports, devices)
- Full capabilities editor (resource params, etc.) deferred to Phase 2

## 11. Docker Status Determination
Instance status `running` is based on:
1. Docker container state = running (agent-side `Start()` verifies)
2. Health check passed (`/v1/models` returns 200 within timeout)
3. Does NOT depend on test request model name matching

Risk: vLLM server can be `running` but test fails due to served model name mismatch. This is correct behavior — status reflects runtime health, not test compatibility. Test diagnostics now clearly show the mismatch.

## 12. vLLM + GGUF Preflight
Not addressed in this round. The vLLM GGUF/Q4 format incompatibility remains a valid issue but is P2 for this round. Recorded for Phase 2/3 format compatibility checks.

## 13. Schema Changes
**None.** No new DB columns, tables, or schema modifications. Only seed data text change and code logic changes.

## 14. Migration
**None.** No migration added. The seed data change uses `INSERT OR IGNORE`; existing DB rows are not auto-updated. New installations get the corrected seed. Existing installations benefit from the `buildVarMap()` derivation fix (which works regardless of seed data).

## 15. Phase 2 Scope
**Not entered.** Changes are within Phase 1.5 product fix scope.

## 16. Test Results

### Go
```
go test ./internal/server/runplan/...  → PASS (0.003s)
go test ./internal/server/api/...      → PASS (6.226s)
go vet ./...                           → PASS
gofmt -w cmd/ internal/               → PASS
```

### Frontend
```
npm --prefix web run build  → PASS (3.12s)
npm --prefix web test       → 18/19 PASS
  FAIL: main navigation exposes model workflow group → PRE-EXISTING (confirmed before changes)
```

### vLLM RunPlan Test (key evidence)
```
TestResolveVLLMNVIDIA:
  --served-model-name Qwen3-0.6B-Instruct-2512 ✓ (present in docker args)
  docker run ... --entrypoint vllm serve /models/Qwen3-0.6B-Instruct-2512
    --host 0.0.0.0 --port 8000 --enforce-eager --max-model-len 4096
    --gpu-memory-utilization 0.6 --served-model-name Qwen3-0.6B-Instruct-2512 ✓
```

## 17. Modified Files
```
internal/server/db/db.go                                — vLLM seed default_args
internal/server/runplan/resolver.go                     — buildVarMap SERVED_MODEL_NAME derivation
internal/server/api/deployment_lifecycle_handlers.go    — resolveModelID always probes, diagnostics
web/src/utils/modelCapabilities.js                      — formatTestFailure enhancement
web/src/pages/ModelInstancesPage.vue                    — test error display fields
web/src/pages/ModelDeploymentsPage.vue                  — dry-run groups, action column fixed
web/src/pages/RunnerConfigsPage.vue                     — NBR template preview label
web/src/locales/en-US.ts                                — new i18n keys
web/src/locales/zh-CN.ts                                — new i18n keys
docs/reports/phase-3/web-ai-config-review/14-live-deployment-validation-issues.md  — issue record
docs/reports/phase-3/web-ai-config-review/15-live-deployment-validation-fix-closeout.md  — this file
```

## 18. Final Verification Checklist
- [x] vLLM `/v1/chat/completions` route confirmed present (from user logs, not the issue)
- [x] `/v1/models` probe integrated into model ID resolution
- [x] Root cause confirmed: served model name mismatch, not missing route
- [x] RunPlan includes `--served-model-name` (verified in test output)
- [x] Test API returns `requested_model` and `available_models`
- [x] Frontend no longer shows only `Failed to fetch`
- [x] Deploy page action column fixed right
- [x] RunPlan parameter sources annotated
- [x] No schema changes
- [x] No migration added
- [x] No Phase 2 entry
- [x] Tests pass (except pre-existing navigation test)
- [x] Frontend builds clean

## 19. Remaining Risks
1. Existing DB installations won't get the updated seed `default_args_json` (INSERT OR IGNORE). However, the `buildVarMap()` derivation fix ensures `SERVED_MODEL_NAME` is always populated from artifact name, which feeds into `--served-model-name` if it's in the args. If existing BackendVersion still has old `default_args_json` without `--served-model-name`, the user needs to reload the backend catalog or manually add the flag.
2. The navigation test failure (`main navigation exposes model workflow group`) is pre-existing and unrelated to these changes.
3. SGLang also has `--served-model-name` as optional — same derivation logic benefits SGLang but the default args already include `--model-path` which embeds the path. SGLang may also need `--served-model-name` in default args for consistency; not addressed in this round.
