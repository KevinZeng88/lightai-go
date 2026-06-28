# Runtime Config Display & Probe Evidence Fix — Closeout

## Commit

`<latest>` (closeout patch on top of `ee35b5b`)

## Root Causes

| Problem | Root Cause |
|---------|-----------|
| **P0-1**: 运行模板详情页参数不显示 | `getConfigEditView()` returned backend envelope `{config_edit_view, config_view}` without unwrapping; `ConfigEditView.vue` read `localView.sections` from the envelope, which has no `sections` field. |
| **P0-2**: 复制后 display_name / name / version 展示错误 | Four-part root cause: (a) catalog runtime YAMLs missing `display_name`, loader fallback to tech ID; (b) clone dialog used raw `row.display_name \|\| row.name` bypassing display adapter; (c) clone backend generated tech `name` from display name via `sourceName + "-copy"`; (d) `extractVersion()` returned concrete version numbers for user configs. |
| **P0-3**: 节点运行配置 probe JSON 默认直出 | `RunnerConfigsPage.vue` rendered raw `probe_results_json` (including `NVIDIA_REQUIRE_CUDA`, `PATH`, `LD_LIBRARY_PATH`); `level4` had dev-language `skip_reason`. |
| **Deployments raw ID**: 部署列表/详情显示 raw `source_node_backend_runtime_id` | Deployment API SQL did not JOIN `node_backend_runtimes` to get NBR `display_name`. |

## Changes

### Batch 1 (`ee35b5b`)

| File | Change |
|------|--------|
| `web/src/api/configEdit.ts` | Unwrap `resp.config_edit_view ?? resp` |
| `configs/backend-catalog/runtimes/vllm/nvidia-docker.yaml` | Add `name` / `display_name: vLLM NVIDIA Docker` |
| `configs/backend-catalog/runtimes/sglang/nvidia-docker.yaml` | Add `name` / `display_name: SGLang NVIDIA Docker` |
| `configs/backend-catalog/runtimes/llamacpp/nvidia-docker.yaml` | Add `name` / `display_name: llama.cpp NVIDIA Docker` |
| `web/src/pages/BackendRuntimesPage.vue` | Clone dialog uses `toRuntimeTemplateDisplay().displayName` |
| `internal/server/api/node_runtime_handlers.go` | Clone tech name: `runtime.<backend>.<vendor>.user.<shortid>` |
| `web/src/utils/runtimeDisplay.ts` | `extractVersion()` always returns `*` |
| `web/src/pages/RunnerConfigsPage.vue` | Probe summary + raw JSON collapsed |
| `internal/server/api/runtime_handlers.go` | level4 structured fields, no dev language |
| `internal/server/api/runtime_boundary_test.go` | 2 clone tests added |
| `internal/server/api/ui_persistence_runplan_test.go` | Updated clone name assertion |

### Batch 2 (closeout)

| File | Change |
|------|--------|
| `web/tests/runtimeBoundaryUi.test.mjs` | Update `extractVersion` tests for new unconditional `*` behavior |
| `internal/server/api/deployment_lifecycle_handlers.go` | `deploymentSelectSQL()` LEFT JOINs `node_backend_runtimes` on `source_node_backend_runtime_id`; adds `source_node_backend_runtime_display_name` to response |
| `web/src/pages/ModelDeploymentsPage.vue` | List column and detail drawer use `source_node_backend_runtime_display_name` with fallback to raw ID |
| `internal/server/api/runtime_handlers.go` | level4 `message` changed from Chinese to English for API consistency |
| `internal/server/api/runtime_boundary_test.go` | `TestCatalogSeedProducersUserVisibleDisplayNames` added |
| `docs/reports/phase-3/runtime-config-display-probe-fix/05-closeout.md` | This document |

## DB Rebuild / Reseed

After catalog YAML changes, the DB must be reseeded for new `display_name` values to take effect.

### Procedure

```bash
# Delete old DB
rm -f /tmp/lightai/data/lightai.db

# Rebuild and restart the server (triggers SeedCatalog)
go build -o lightai-server ./cmd/server
./lightai-server

# Or: run Go tests which auto-seed in setupTestDB
go test ./internal/server/api/ -run TestCatalogSeedProd -v
```

### Verification

`TestCatalogSeedProducersUserVisibleDisplayNames` confirms:

| Runtime ID | display_name |
|-----------|-------------|
| `runtime.vllm.nvidia-docker` | `vLLM NVIDIA Docker` |
| `runtime.sglang.nvidia-docker` | `SGLang NVIDIA Docker` |
| `runtime.llamacpp.nvidia-docker` | `llama.cpp NVIDIA Docker` |

## Test Results

### Go Tests
```
ok   lightai-go/internal/server/agentclient
ok   lightai-go/internal/server/api
ok   lightai-go/internal/server/auth
ok   lightai-go/internal/server/authz
ok   lightai-go/internal/server/catalog
ok   lightai-go/internal/server/configedit
ok   lightai-go/internal/server/runplan
ok   lightai-go/internal/server/semanticconfig
```

### Frontend Tests
```
All tests PASSED (100/100)
npm run build: ✓ built in 3.55s
```

### New Tests Added
- `TestCloneBackendRuntimeWithUserVisibleDisplayName` — clone uses user-visible display_name, stable technical name
- `TestCloneBackendRuntimeNoDisplayNameUsesGeneratedName` — clone without explicit display_name defaults to technical name
- `TestCatalogSeedProducersUserVisibleDisplayNames` — catalog seed produces correct display_names

## Same-Category Check Results

1. **Parameter fields**: All 5 pages using `getConfigEditView()` now receive unwrapped data ✅
2. **display_name / name mixing**: ModelDeploymentsPage now shows NBR display_name from JOIN ✅
3. **Raw evidence**: RunnerConfigsPage shows probe summary by default, raw JSON collapsed ✅
4. **level4 product language**: No `deferred to future design` / `not yet implemented` in user-visible data; message in English ✅

## Remaining Risks

- Non-seeded databases (existing deployments) must be manually deleted and reseeded for catalog display_name changes
- `VERSION` file remains modified (pre-existing, not from this fix)
- Version probe endpoint implementation is still deferred (by design)
