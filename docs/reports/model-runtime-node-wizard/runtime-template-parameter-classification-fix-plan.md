# Runtime Template Parameter Classification Fix Plan

Date: 2026-06-19
Branch: main
Starting commit: 2118897

## 1. Current Data Flow

```
BackendVersion (catalog YAML + DB seed)
  → HandleCreateBackendRuntimeFromTemplate (copy + freeze)
    → BackendRuntime (version_snapshot_json + own config)
      → upsertNodeBackendRuntime (enable/check)
        → NodeBackendRuntime (config_snapshot_json)
          → HandleCreateDeployment (capture BR+NBR snapshot)
            → Deployment (config_snapshot_json + service/params/env_overrides)
              → preflightDeployment (snapshot-only, no live re-read)
                → runplan.Resolve()
                  → ResolvedRunPlan (plan_json, frozen)
                    → Agent ConvertRunplanToAgentSpec()
                      → Docker buildCreateOptions()
                        → docker run
```

Key files:
- Catalog YAML: `configs/backend-catalog/runtimes/{backend}/{vendor}-docker.yaml`
- Catalog loader: `internal/server/api/backend_handlers.go:480-633` (upsertBackendRuntimeProjection)
- DB seed: `internal/server/db/db.go:1326-1408` (seedTargetBackendCatalog)
- BackendRuntime CRUD: `internal/server/api/runtime_handlers.go`
- NodeBackendRuntime: `internal/server/api/runtime_handlers.go:297-410`
- Deployment create: `internal/server/api/deployment_lifecycle_handlers.go:83-157`
- Preflight: `internal/server/api/deployment_lifecycle_handlers.go:504-729`
- RunPlan: `internal/server/runplan/resolver.go:156-280`
- Preview: `internal/server/runplan/preview.go:10-72`
- Agent Docker: `internal/agent/runtime/docker.go:415-504`
- Agent adapter: `internal/agent/runtime/runplan_adapter.go:8-68`

Frontend key files:
- Deploy wizard: `web/src/pages/ModelDeploymentsPage.vue` (runtime select at step 3, filteredRuntimes)
- Runtime list: `web/src/pages/BackendRuntimesPage.vue` (builtin/user runtimes)
- Runner config: `web/src/pages/RunnerConfigsPage.vue` (NodeBackendRuntimes)
- API: `web/src/api/runtimes.ts` (`GET /api/v1/backend-runtimes`)

## 2. Fix Scope

### Problem 1: Runtime detail/edit shows raw "snapshot" instead of structured config
- Root: UI shows `config_snapshot_json` as raw JSON. Should show classified params.
- API: `HandleListBackendRuntimes` returns runtimes. Detail endpoint returns full record.
- Need: UI restructure to classify and display params.

### Problem 2: User-created runtimes not appearing in deployment wizard
- Root: `filteredRuntimes` in `ModelDeploymentsPage.vue:331` only filters by `backend_version_id`.
- API: `GET /api/v1/backend-runtimes` — admin sees all; tenant users see own tenant + global.
- Need: Verify that user-created runtimes ARE returned by API. Likely a frontend filtering issue.

### Problem 3: MetaX/vLLM/SGLang documentation verification
- Need: Web search for official docs to verify parameter defaults.

### Problem 4: Built-in templates readable but not editable
- Root: `HandlePatchBackendRuntime` (line 172) checks `is_editable`. Frontend shows edit button for system runtimes.
- Need: Frontend to show read-only detail, disable edit for builtins, enable clone.

### Problem 5: Clone flow should allow pre-edit
- Root: `HandleCloneBackendRuntime` copies immediately without user interaction.
- Need: Frontend clone dialog with pre-filled editable fields, save-on-confirm.

### Cross-cutting: Parameter classification
- devices should NOT contain volume-style `host:container` paths
- MetaX `CUDA_VISIBLE_DEVICES` should be `MACA_VISIBLE_DEVICE`
- enabled/disabled semantics for params
- custom args/env/devices/volumes/options sections
- high-risk parameter flagging

## 3. Phase Execution

### Phase 1: Documentation Verification & Parameter Matrix

**Actions:**
1. Web-search vLLM official Docker/server args documentation
2. Web-search SGLang official Docker/server args documentation
3. Web-search llama.cpp official Docker/server args documentation
4. Web-search MetaX/MacaRT SGLang and vLLM documentation
5. Build parameter matrix for each backend × vendor combination
6. Write findings to `docs/reports/model-runtime-node-wizard/runtime-template-parameter-classification-fix-plan.md`

**Expected files:** None changed (research only)
**Verification:** Documentation complete with sources

### Phase 2: Fix Catalog YAML + DB Seed Parameters

**Actions:**
1. Fix MetaX vLLM/SGLang devices — remove volume-style paths
2. Fix MetaX env — add MACA_VISIBLE_DEVICE, remove CUDA_VISIBLE_DEVICES as sole default
3. Fix GPU visible device env key per vendor in DockerSpecInfo
4. Fix default enabled/disabled states per parameter matrix
5. Add `devices_json`, `volumes_json`, `env_schema_json` columns to seed data
6. Fix `defaultVisibleEnvKey()` in resolver.go for MetaX vendor

**Expected files:**
- `configs/backend-catalog/runtimes/vllm/metax-docker.yaml`
- `configs/backend-catalog/runtimes/sglang/metax-docker.yaml`
- `configs/backend-catalog/runtimes/sglang/metax-macart.yaml`
- `configs/backend-catalog/runtimes/llamacpp/metax-docker.yaml`
- `internal/server/db/db.go` (seed data)
- `internal/server/runplan/resolver.go` (defaultVisibleEnvKey)

**Verification:** `go test ./internal/server/runplan/ -v`

### Phase 3: BackendRuntime Detail/Edit API & List

**Actions:**
1. Ensure `HandleGetBackendRuntime` returns structured parameter fields
2. Ensure `HandleListBackendRuntimes` includes user-created runtimes (audit tenant filter)
3. Add list filter by vendor/backend_id to support deployment wizard
4. Ensure `HandlePatchBackendRuntime` blocks builtins (already does)
5. Verify `HandleCloneBackendRuntime` works correctly

**Expected files:**
- `internal/server/api/runtime_handlers.go`
- `internal/server/api/node_runtime_handlers.go`

**Verification:** `go test ./internal/server/api/ -run "BackendRuntime|Runtime|Clone"`

### Phase 4: Frontend — Runtime Detail/Edit Page

**Actions:**
1. `BackendRuntimesPage.vue`: Restructure detail to show classified params (not raw snapshot)
2. Show devices, volumes, env, docker options, app args, health check, source template sections
3. Built-in: read-only, no save button, only clone button
4. User-created: editable fields, save button
5. Add "view raw JSON" toggle in advanced section
6. Add clone dialog with pre-filled editable fields
7. Update i18n keys

**Expected files:**
- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/locales/zh-CN.ts`
- `web/src/locales/en-US.ts`

**Verification:** `npm --prefix web run build && npm --prefix web test -- --runInBand`

### Phase 5: Deployment Wizard Runtime Selection

**Actions:**
1. Verify `GET /api/v1/backend-runtimes` returns all runtimes (builtin + user + global)
2. Fix frontend `filteredRuntimes` to show user-created runtimes
3. Add labels: "系统内置" / "用户配置" / "节点配置"
4. Ensure user-created runtime selected in wizard → deployment captures its config

**Expected files:**
- `web/src/pages/ModelDeploymentsPage.vue` (minor filter/tag fixes)
- `web/src/locales/zh-CN.ts`
- `web/src/locales/en-US.ts`

**Verification:** Wizard shows user-created runtimes; selecting one creates deployment with correct snapshot

### Phase 6: Tests, E2E, Documentation

**Actions:**
1. Backend tests for parameter classification, device normalization
2. Frontend tests for clone dialog, readonly detail, wizard selection
3. Update design docs
4. Selected E2E for MetaX template details + clone flow
5. Full verification suite
6. Commit and push

**Expected files:**
- `internal/server/api/ui_persistence_runplan_test.go` (new tests)
- `internal/server/runplan/resolver_test.go` (new tests)
- `web/tests/` (new tests)
- `docs/lightai-backend-runtime-runplan-docker-design.md`
- `docs/reports/model-runtime-node-wizard/*.md`

## 4. Risk Points

| Risk | Mitigation |
|------|-----------|
| MetaX real hardware unavailable | Document as DOCUMENTED_BLOCKER; mock tests cover classification |
| SGLang v0.5.13 API changed from `launch_server` to `serve` | Verify via web search; if so, update entrypoint |
| Catalog YAML format changes break reload | Test `go test ./internal/server/api/ -run "Catalog\|Reload"` |
| Frontend i18n key leaks | `npm test` catches key mismatches |
| Changing `defaultVisibleEnvKey` breaks existing deployments | Only affects new RunPlans; existing RunPlans are immutable |
| User runtime not showing in wizard | Verify API response; may be frontend filter or tenant scope |

## 5. No-Change Zones

- Do NOT refactor `runplan.Resolve()` architecture
- Do NOT change the 5-layer snapshot inheritance model
- Do NOT modify `preflightDeployment` snapshot application logic
- Do NOT change Docker create spec generation
- Do NOT create new DB migrations unless absolutely necessary
- Do NOT create new branches

## 6. Acceptance Criteria

1. Built-in runtime detail page shows classified params (devices/volumes/env/options/args/health/source), not raw JSON
2. Built-in runtime: read-only detail, no save, clone button available
3. User-created runtime: editable, save persists
4. Clone built-in → opens dialog with pre-filled fields → save creates user config
5. Cancel clone → nothing created
6. User-created runtime appears in deployment wizard runtime select
7. Selecting user-created runtime → deployment captures correct config snapshot
8. MetaX devices: `/dev/dri` (not `/dev/dri:/dev/dri`) in devices section
9. MetaX env: `MACA_VISIBLE_DEVICE` (not `CUDA_VISIBLE_DEVICES`)
10. All tests pass, git status clean


## Implementation Summary (2026-06-19)

### Phase 1: Documentation Verification
- Web-searched vLLM (docs.vllm.ai), SGLang (docs.sglang.io), llama.cpp (github/ggml-org) official docs
- Confirmed: vLLM entrypoint `["vllm","serve"]`, SGLang `["python3","-m","sglang.launch_server"]`, llama.cpp empty entrypoint (binary has built-in)
- Confirmed: SGLang default port 30000, vLLM 8000, llama.cpp 8080
- Confirmed: MetaX uses `MACA_VISIBLE_DEVICE` (not `CUDA_VISIBLE_DEVICES`), via MacaRT-SGLang/MacaRT-vLLM distributions
- Confirmed: MetaX Docker requires privileged, host IPC, seccomp/apparmor unconfined, 100gb shm

### Phase 2: Catalog YAML + DB Seed + Resolver Fixes
- Added "metax" case to `defaultVisibleEnvKey()` → returns "MACA_VISIBLE_DEVICE"
- Fixed MetaX YAML: vllm/metax-docker.yaml, sglang/metax-docker.yaml, llamacpp/metax-docker.yaml
  - Changed `CUDA_VISIBLE_DEVICES` → `MACA_VISIBLE_DEVICE`
  - Added `MACA_SMALL_PAGESIZE_ENABLE`, `PYTORCH_ENABLE_PG_HIGH_PRIORITY_STREAM`
- Fixed DB seed (db.go): `runtime.vllm.metax-docker` → MACA_VISIBLE_DEVICE
- Updated MetaX test (metax_huawei_test.go) to verify MACA_VISIBLE_DEVICE=6,7
- Devices in YAML remain correct single-path format: `/dev/dri`, `/dev/mxcd`, `/dev/infiniband`

### Phase 3: BackendRuntime API
- Verified `HandleListBackendRuntimes` returns both builtin and user-created runtimes
- Tenant filtering: admin sees all, tenant users see own tenant + global

### Phase 4: Frontend Runtime Detail Page
- Added clone-to-user dialog with pre-filled name/display_name
- Enhanced detail drawer with classified params: Docker config table, app args, env table
- Builtin templates: readonly alert, no save button, clone button
- User templates: editable with save button
- Raw JSON collapsible section in detail view
- Added i18n keys: cloneToUserConfig, cloneSourceTemplate, systemBuiltin, dockerConfig, appArgs, detailEnv, viewRawJSON, rawJSON

### Phase 5: Deployment Wizard Runtime Selection
- Fixed runtime label to show display_name with source tag (System Built-in)
- Added startWizard.systemBuiltin i18n key
- API returns all runtimes (verified); frontend filter by backend_version_id works for user runtimes

### Phase 6: Verification
- go test ./... (10 packages) PASS
- go vet ./... PASS
- go build ./... PASS
- npm build PASS
- npm test PASS (688 i18n keys, 577 references, 15 boundary tests)
- git diff --check PASS
