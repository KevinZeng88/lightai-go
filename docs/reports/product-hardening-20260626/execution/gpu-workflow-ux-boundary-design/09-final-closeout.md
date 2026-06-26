# 09 — GPU Workflow UX Boundary Repair — Final Closeout

Date: 2026-06-26 | Status: CLOSED

## 1. Scope

Repair the GPU/model runtime management UX boundaries so users configure a runtime environment rather than editing internal ConfigSet structures. Establish three clear product lines (Model, Runtime, Deployment) with well-defined boundaries and consistent UX patterns.

### Three Lines

| Line | Objects | User Goal | Pages |
|---|---|---|---|
| Model | ModelArtifact, ModelLocation | Where are model files and what facts do they have? | ModelArtifactsPage |
| Runtime | Backend, BackendVersion, BackendRuntime, NodeBackendRuntime | Can this node run models with this GPU/backend? | BackendRuntimesPage, RunnerConfigsPage |
| Deployment | ModelDeployment, ResolvedRunPlan, ModelInstance | Combine model + ready runtime → running service | ModelDeploymentsPage, ModelInstancesPage |

## 2. Completed User-Flow Improvements

### Runtime Templates Page
- Display names use `nvidia.vllm` / `nvidia.sglang` / `nvidia.llama.cpp b9700` format via `runtimeDisplay.ts` adapter
- Main table hides raw `backend_runtime_id` / `backend_version_id`; shows vendor, backend, version, image, ready count
- ConfigSet + Source Metadata moved to collapsed "Advanced Diagnostics" section
- System templates read-only with clone action; user-managed templates editable via `HumanRuntimeParameterForm`

### Node Runtime Configs Page
- `NodeRuntimeConfigWizard`: 4-step wizard (node → template → image+params → save & check)
- Wizard resets to step 1 on every open (`destroy-on-close`)
- Config name field with auto-generated default (`<hostname> / <vendor> / <backend>`)
- Default name saved even when user leaves field empty
- Node selection via shared `NodeSelectorTable`
- Parameters via `HumanRuntimeParameterForm` — hides internal ConfigSet keys
- Save/check failure stays on current step with error display
- Non-ready check result keeps wizard open; only `ready`/`ready_with_warnings` enables Finish

### Model Library Page
- Node selection uses shared `NodeSelectorTable` instead of dropdown
- Label: "选择模型所在节点" (Select Model File Node)
- No Docker/runtime/GPU parameters exposed

### Model Deployments Page
- `DeploymentWizard`: 5-step wizard with preview
- Only `ready`/`ready_with_warnings` NBRs selectable
- `needs_check`/`missing_image`/`error`/`unknown` visible but disabled
- Payload uses `node_backend_runtime_id` only (never `backend_runtime_id`)
- Preview calls `POST /api/v1/deployments/preview`
- Frontend compatibility check: model location must exist on NBR's node
- `load()` derives model locations from artifact `.locations` arrays
- Backend `HandleCreateDeployment` enforces model location on NBR node

## 3. Key Blockers and Fixes

| Blocker | Fix Commit | Resolution |
|---|---|---|
| Design documents not committed | `3cd3be0` | 8 design docs committed |
| Claude understanding check | `6869c1e` | Confirmation document |
| Default config name not saved | `ee00c25` | `form.display_name \|\| defaultConfigName.value` |
| DeploymentWizard no node compatibility | `ee00c25` | `checkNodeCompatibility()` added |
| shm_size mapped to wrong key | `ee00c25` | Flat `shm_size` key, merged into `launcher.docker_options` |
| RuntimeParameterEditor on system templates | `ee00c25` | Moved to Advanced Diagnostics; `HumanRuntimeParameterForm` for editable |
| `/model-locations` API not found | `20b7bcf` | Derive from artifact `.locations` arrays |
| `HandleCreateDeployment` no location check | `2c27bb4` | Added `model_locations` check before INSERT |
| Test fixtures missing model_locations | `e9e71df` | Added `snapshotInsertModelLocation` to affected tests |

## 4. Backend Contract Confirmation

| Check | Status |
|---|---|
| `POST /api/v1/deployments` rejects `backend_runtime_id` | ✅ `rejectLegacyDeploymentPayload` + explicit checks |
| `POST /api/v1/deployments` only accepts `node_backend_runtime_id` | ✅ |
| `ready` / `ready_with_warnings` NBR deployable | ✅ `isNBRDeployable()` |
| `missing_image` / `needs_check` / `error` NBR not deployable | ✅ |
| `POST /api/v1/deployments` validates `model_locations` at create | ✅ `2c27bb4` |
| `POST /api/v1/deployments/preview` validates model location | ✅ deployment_preview_handlers.go:79-90 |
| dry-run/start validate model location via `preflightDeployment` | ✅ deployment_lifecycle_handlers.go:810-814 |
| Missing location returns `model_location_missing` | ✅ |

## 5. Test Results

```bash
go test ./...          # ALL PASS (18 packages, 0 failures)
go build ./cmd/server/... ./cmd/agent/...  # PASS
cd web && npm test     # ALL PASS (37 tests, 991 i18n keys)
cd web && npm run build  # PASS (3.42s)
git diff --check       # PASS
git status --short     # CLEAN
```

New Go tests added:
- `TestCreateDeploymentRejectsModelLocationMissing` — expects 400
- `TestCreateDeploymentAcceptsWithModelLocation` — expects 201

## 6. Evidence

```
docs/reports/product-hardening-20260626/evidence/20260626180152/gpu-workflow-ux-boundary/
  review-summary.md
  go-test.log
  go-build.log
  npm-test.log
  npm-build.log
  git-diff-check.log
  final-closeout-summary.md
```

## 7. Out of Scope (Confirmed NOT Implemented)

- ❌ OpenAI Gateway (`/v1/models`, `/v1/chat/completions`)
- ❌ API Key management
- ❌ Usage Metering / Billing
- ❌ DB schema migration
- ❌ Kubernetes / Ray scheduler
- ❌ MetaX real hardware validation (external dependency, carried from RC1)

## 8. Remaining Risks

**NONE.**

Audit results:
- No user-facing page exposes `launcher.*` / `runtime_env.*` / `{{MODEL_CONTAINER_PATH}}` as primary form fields ✅
- No deployment payload uses `backend_runtime_id` ✅
- No page calls a non-existent API endpoint ✅
- No save failure silently closes a dialog ✅
- All test suites pass; evidence captured ✅

## 9. Commit Range

```
3cd3be0  docs: add gpu workflow ux boundary design package
6869c1e  docs: confirm gpu workflow ux boundary understanding
e95f54b  docs: audit gpu workflow ux boundaries
9ce7ced  fix: reset runtime config wizard and add config naming
9383031  fix: simplify runtime template presentation
a535774  fix: add human runtime parameter form
717cc04  fix: align model library node selector and deployment compatibility ux
3812bac  test: add gpu workflow ux regression evidence
ee00c25  fix: close gpu workflow ux boundary blockers
20b7bcf  fix: derive deployment model locations from artifacts
2c27bb4  fix: enforce deployment create model location compatibility
e9e71df  fix: add model location fixtures to deployment create tests
fa309e4  docs: close gpu workflow ux boundary repair
```

## 10. Final Status

GPU_WORKFLOW_UX_BOUNDARY_REPAIR_CLOSED
