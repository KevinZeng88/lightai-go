# 01 — File-Level Implementation Plan

Revised: 2026-06-26 | Based on: `00-current-code-inventory.md`
Scope: 模型运行管理闭环 — BackendRuntime / NBR / Deployment / Instance / RunPlan / preflight / smoke
Excluded: OpenAI Gateway / API Key / Usage Metering (deferred to `future-openai-gateway-notes.md`)

## 1. Baseline

```bash
git rev-parse --short HEAD  # c13f91f
git status --short          # ?? docs/reports/product-hardening-20260626/  (only untracked)

go test ./...               # ALL PASS
go build ./cmd/server/...   # PASS
go build ./cmd/agent/...    # PASS
cd web && npm test          # ALL PASS (37 tests)
cd web && npm run build     # PASS (3.29s)
git diff --check            # PASS
```

**Pre-existing failures: NONE.** All tests pass at baseline. Any failure after implementation is a regression.

## 2. Revised Implementation Order

```
A (inventory, done) → B (NBR / runtime parameters) → C (deployment UI + preview) → D (start/stop/status/logs) → E (E2E regression) → F (naming cleanup, safe only) → G (gateway future notes, document only)
```

Rationale:
- **A** already exists (`00-current-code-inventory.md`).
- **B** (runtime parameter completeness) comes before **C** (deployment wizard) — the wizard's override editor depends on the parameter editor working correctly.
- **C** (deployment UI) adds preview endpoint + wizard — the wizard integrates parameter editor from B.
- **D** (start/stop/status/logs stability) validates the full instance lifecycle with the new preview/wizard contract.
- **E** (E2E regression) runs last — captures final state of all code changes with vLLM/SGLang/llama.cpp smoke.
- **F** (naming cleanup) runs last among code changes — safe label/i18n fixes only, no route renames.
- **G** (gateway notes) is document-only, no code.

OpenAI Gateway (API keys, usage, `/v1/models`, `/v1/chat/completions`, billing) is **deferred** — see `future-openai-gateway-notes.md`.

**⚠️ Before implementation, Claude must read and follow `03-implementation-guardrails.md`. These guardrails override any conflicting wording in this plan.**

---

## 3. Workstream B — Runtime Config / NBR Parameter Completeness

### Objective
Surface all backend runtime parameters from catalog YAMLs in the `RuntimeParameterEditor`, integrate into BackendRuntime template and NodeBackendRuntime pages at correct layers. Model page must not hold runtime serving args. Lint rules for duplicate/conflict/incompatible parameters.

### Files to modify

| File | Action | Detail |
|---|---|---|
| `web/src/components/common/RuntimeParameterEditor.vue` | CHANGE | Add props: `layer` (`'backend_runtime'` / `'node_backend_runtime'` / `'deployment'`), `baseValues` (inherited values from parent layer), `showSource`, `showAdvanced`; add emit: `validate`; add behavior: required fields locked enabled, optional fields have enable/disable toggle, disabled values retained in model but excluded from final args output, validation errors shown inline, advanced groups collapsible, source/diff from base visible; add backend/vendor applicability filter (only show params for current backend) |
| `web/src/pages/BackendRuntimesPage.vue` | CHANGE | Install `RuntimeParameterEditor` in detail drawer: system-managed templates read-only, user-managed templates editable; add "Clone from system template" button → creates new user-managed template with copied ConfigSet |
| `web/src/pages/RunnerConfigsPage.vue` | CHANGE | Install `RuntimeParameterEditor` in create dialog and detail drawer: show inherited BackendRuntime values as read-only `baseValues`, NBR-specific overrides editable as diff; save emits `PATCH` with NBR-specific parameter values |
| `web/src/pages/ModelArtifactsPage.vue` | CHANGE | Keep `parameter_defaults` textarea but relabel to "Model Facts and Hints"; remove `--max-model-len`, `--served-model-name`, `--gpu-memory-utilization` from placeholder text; add explicit hint: "Model facts only — Docker/runtime parameters belong in Runtime Template or Deployment configuration"; forbid backend serve args in model edit surface |
| `web/src/pages/ModelDeploymentsPage.vue` | CHANGE | Wire `RuntimeParameterEditor` (via `DeploymentOverrideEditor` wrapper from Workstream C): show NBR param values as base, deployment overrides as diff |
| `internal/server/api/runtime_handlers.go` | CHANGE | Ensure `PATCH /api/v1/backend-runtimes/{id}` saves `config_set_json` with full parameter values; ensure `PATCH /api/v1/nodes/{id}/backend-runtimes/{nbr_id}` saves node-specific parameter values; read-only guard for system-managed (managed_by != "user") |
| `internal/server/api/deployment_lifecycle_handlers.go` | CHANGE | Ensure `HandlePatchDeployment` properly processes `config_overrides` with full `parameter_values`, `disabled_parameters`, `env` arrays |
| `internal/server/runplan/lint.go` | CHANGE | Add lint rules: `LintRuleDuplicateArg` (same CLI flag in extra_args + parameter_values), `LintRuleEnvCLIConflict` (env override conflicts with CLI arg), `LintRulePlatformArgOverridden` (user overriding `--host`/`--port`/`--model`), `LintRuleUnsupportedParam` (param not in backend args_schema), `LintRuleDisabledFieldApplied` (disabled value appearing in final args), `LintRuleMissingRequired` (required param without value), `LintRuleVendorIncompatible` (CUDA param on non-NVIDIA vendor) |
| `internal/server/runplan/resolver.go` | MINOR | Ensure lint results accessible from Resolve output (add `LintResult` to return struct or expose via separate method) |
| `configs/backend-catalog/versions/vllm/vllm-v0.23.0.yaml` | MINOR | Verify all 17 `default_args_schema` entries have `group`, `type`, `advanced` fields; verify `vendor_options.resource_controls` entries complete |
| `configs/backend-catalog/versions/sglang/sglang-v0.5.13.post1.yaml` | MINOR | Add `--chunked-prefill-size` and `--attention-backend` to `default_args_schema` (currently only in resource_controls — needed for UI rendering) |
| `configs/backend-catalog/versions/llamacpp/llamacpp-b9700.yaml` | MINOR | Verify no fake `gpu_memory_fraction` surfaces (already correct: `supported: false` with reason) |

### Functions/components changed

**RuntimeParameterEditor.vue** — major enhancement (currently dead code, 0 imports):
- New prop `layer: string` — `'backend_runtime'` / `'node_backend_runtime'` / `'deployment'`
- New prop `baseValues: ParameterValue[]` — inherited values from parent layer
- New prop `showSource: boolean` — toggle source/diff column
- New prop `showAdvanced: boolean` — toggle advanced group visibility
- New emit `validate: (errors: ValidationError[]) => void`
- New behavior: `backend`/`vendor` applicability filter — only params for current backend
- New behavior: required fields locked (lock icon, toggle disabled)
- New behavior: disabled optional field — value retained in editor state, excluded from `config_overrides.parameter_values` output
- New behavior: source trace per parameter (BackendVersion → BackendRuntime → NBR → Deployment)

**BackendRuntimesPage.vue** — interactive parameter editing:
- Detail drawer: install `RuntimeParameterEditor`
- System-managed (`managed_by != 'user'`): editor in read-only mode
- User-managed: editor editable
- "Clone" action on system template row → `POST /api/v1/backend-versions/{id}/clone` then redirect to new user-managed runtime

**RunnerConfigsPage.vue** — interactive parameter editing:
- Create dialog: after selecting node + runtime template, show `RuntimeParameterEditor` with template values as `baseValues` (read-only)
- Detail drawer: show `RuntimeParameterEditor` with inherited + overridden values
- NBR-specific overrides shown as diff from template

**lint.go** — 7 new lint rules:
- `LintRuleDuplicateArg`, `LintRuleEnvCLIConflict`, `LintRulePlatformArgOverridden`, `LintRuleUnsupportedParam`, `LintRuleDisabledFieldApplied`, `LintRuleMissingRequired`, `LintRuleVendorIncompatible`

### Data/API contract changes

| Method | Path | Change |
|---|---|---|
| PATCH | `/api/v1/backend-runtimes/{id}` | ENHANCE: accept full `config_set` parameter values; reject if system-managed |
| PATCH | `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}` | ENHANCE: accept node-specific `config_set` overrides |
| PATCH | `/api/v1/deployments/{id}` | ENHANCE: accept full `config_overrides` with `parameter_values`, `disabled_parameters`, `env` |

No new routes. No routes removed. No DB schema changes. Existing routes already store these payloads — frontend must send them.

### Tests to add/update

**Go tests** — `internal/server/api/runtime_parameter_layering_test.go` (CREATE):
```
Test cases:
1. nbr_value_wins_over_backend_runtime
2. deployment_override_wins_over_nbr
3. disabled_optional_value_retained_not_applied
4. required_field_cannot_be_disabled
5. vllm_gpu_memory_fraction_validates_0.1_to_0.95
6. sglang_mem_fraction_static_validates
7. llamacpp_no_fake_memory_fraction
8. vendor_incompatible_param_linted
9. system_managed_runtime_rejects_patch
```

**Go tests** — `internal/server/api/model_artifact_boundary_test.go` (CREATE):
```
Test cases:
1. model_page_parameter_defaults_not_used_as_runtime_args
2. deployment_does_not_mutate_nbr
```

**Go tests** — `internal/server/runplan/lint_test.go` (UPDATE existing):
```
Add cases for 7 new lint rules
```

**Frontend tests** — `web/tests/runtimeParameterEditor.test.mjs` (CREATE):
```javascript
// Assertions:
// 1. required fields rendered as locked enabled (toggle disabled state)
// 2. optional fields have enable checkbox
// 3. disabled field value retained in internal model
// 4. disabled field value NOT emitted in config_overrides output
// 5. vLLM memory fraction renders with range 0.1-0.95 validation
// 6. SGLang memory fraction renders
// 7. llama.cpp does NOT render memory fraction (supported: false)
// 8. source/diff shows inherited vs override values from baseValues prop
// 9. validation errors shown inline for out-of-range values
// 10. backend/vendor filter excludes inapplicable params
```

**Frontend tests** — `web/tests/modelCapabilities.test.mjs` (UPDATE existing):
```
Add assertion: ModelArtifactsPage placeholder does not reference --max-model-len, --gpu-memory-utilization, --served-model-name
```

### Validation commands
```bash
# Backend
go test ./internal/server/api/... -run 'Parameter|ConfigSet|Lint|Runtime'
go test ./internal/server/runplan/... -run 'vllm|sglang|llamacpp|Resource|Lint'

# Frontend
cd web && npm test
cd web && npm run build

# Verify model page placeholder has no runtime args
grep -n "max-model-len\|gpu-memory-utilization\|served-model-name" web/src/pages/ModelArtifactsPage.vue
```

---

## 4. Workstream C — Model Deployment UI + RunPlan Preview

### Objective
Replace thin create dialog in `ModelDeploymentsPage.vue` with guided deployment wizard showing model facts, NBR status, service config, parameter overrides, RunPlan preview, Docker command preview, lint/preflight findings, and start blockers. Add `POST /api/v1/deployments/preview` endpoint.

### Files to modify

| File | Action | Detail |
|---|---|---|
| `web/src/pages/ModelDeploymentsPage.vue` | REPLACE | Replace thin create dialog (lines 25–45) with `DeploymentWizard` component; keep list table and detail drawer; add NBR status column with colored `StatusTag`; resolve and show model/artifact display names (not raw UUIDs) |
| `web/src/components/deployments/DeploymentWizard.vue` | CREATE | 6-section wizard: (1) Model, (2) Node Runtime Config, (3) Service, (4) Resource/Placement, (5) Overrides, (6) Preview; stepper or tab navigation; validate each step before proceeding |
| `web/src/components/deployments/ModelSelector.vue` | CREATE | Model card: display_name, format, task_type, capabilities, location node/path, verification status; warning if no model location on selected node; filter/search |
| `web/src/components/deployments/NodeRuntimeSelector.vue` | CREATE | NBR card: display_name, node label, backend + version, vendor, image ref, deployable status tag with reason text; block `needs_check`, `missing_image`, `error`, `unknown`; allow `ready` and `ready_with_warnings`; show last check time |
| `web/src/components/deployments/DeploymentServiceEditor.vue` | CREATE | Fields: host_port (number, 1–65535), container_port (number, default from backend catalog), served_model_name (text), endpoint preview (read-only derived URL) |
| `web/src/components/deployments/DeploymentOverrideEditor.vue` | CREATE | Wraps `RuntimeParameterEditor` with `layer="deployment"` and `baseValues` from selected NBR; shows inherited vs overridden diff; saves only deployment-level changes |
| `web/src/components/deployments/DeploymentPreviewPanel.vue` | CREATE | Calls `POST /api/v1/deployments/preview`; shows: can_run (boolean + reason), lint findings (status + list), resource_admission (status + list), preflight errors/warnings, Docker command preview (code block), RunPlan JSON (JsonViewer, collapsible), source trace; "Save" and "Save & Start" buttons (disabled if !can_run) |
| `web/src/api/deployments.ts` | CHANGE | Add `previewDeployment(data)` function: `POST /deployments/preview`, returns `PreviewResult`; add `PreviewResult` interface |
| `internal/server/api/router.go` | ADD | Register `POST /api/v1/deployments/preview` → `HandleDeploymentPreview` with `mdWriteChain` |
| `internal/server/api/deployment_preview_handlers.go` | CREATE | `HandleDeploymentPreview`: accepts same payload as create (without requiring deployment ID), runs shared `preflightDeployment()` resolver, returns full `preflightResult` including RunPlan + lint + command preview; does NOT write to DB |
| `internal/server/api/preflight_handlers.go` | REFACTOR | Extract `performDeploymentPreflight(ctx, db, payload) (*preflightResult, error)` from `HandlePreflightDeployments` — shared by both preflight and new preview handlers |
| `docs/api/openapi.yaml` | ADD | Schema for `DeploymentPreview` request/response, `POST /deployments/preview` path |

### Functions/components changed

**New Go handler — `HandleDeploymentPreview`:**
- File: `internal/server/api/deployment_preview_handlers.go`
- Signature: `func (h *AgentHandler) HandleDeploymentPreview(w http.ResponseWriter, r *http.Request)`
- Input: `{ name, display_name?, model_artifact_id, node_backend_runtime_id, placement_json?, service_json?, config_overrides? }`
- Logic: validate NBR deployable → validate artifact exists → check model location on node → check GPU availability → run `runplan.Resolve()` → run `LintRunPlan()` → build `EquivalentCommandPreview()` → return result
- MUST use same resolver path as `HandleStartDeployment` (single source of truth — verified by test)
- Does NOT require deployment ID in URL, does NOT write to DB

**Refactored — `preflight_handlers.go`:**
- Extract `performDeploymentPreflight(ctx, db, payload) (*preflightResult, error)`
- Called by: `HandlePreflightDeployments` (existing), `HandleDeploymentPreview` (new), `HandleStartDeployment` (existing, already calls `preflightDeployment()` internally — verify shared path)

### Data/API contract changes

| Method | Path | Change | Handler |
|---|---|---|---|
| POST | `/api/v1/deployments/preview` | **ADD** | `HandleDeploymentPreview` |

Request: `{ name, display_name?, model_artifact_id, node_backend_runtime_id, placement_json?, service_json?, config_overrides? }`

Response: `{ can_run: bool, run_plan: object, docker_preview: string, lint: { status, findings[] }, resource_admission: { status, findings[] }, preflight: { status, errors[], warnings[] }, source_trace: object }`

No routes removed. No breaking changes.

### Tests to add/update

**Backend — `internal/server/api/deployment_preview_test.go` (CREATE):**
```go
// 1. preview_rejects_legacy_backend_runtime_id
// 2. preview_rejects_non_ready_nbr (needs_check, missing_image, error)
// 3. preview_accepts_ready_with_warnings (with warnings in response)
// 4. preview_blocks_missing_model_location
// 5. preview_blocks_host_port_conflict
// 6. preview_and_start_use_same_resolver_path (same Resolve() call, same result)
// 7. preview_disabled_parameter_not_applied (disabled value absent from run_plan args)
// 8. preview_deployment_override_wins_over_nbr
// 9. preview_no_deployment_id_in_url (endpoint does not read {id} from path)
// 10. preview_does_not_write_to_db (no rows in model_deployments after call)
```

**Backend — `internal/server/api/preflight_handlers_test.go` (UPDATE):**
- Add: `preflight_rejects_legacy_fields` (ensure existing preflight still works)

**Frontend — `web/tests/deploymentWizard.test.mjs` (CREATE):**
```javascript
// 1. create payload uses node_backend_runtime_id (not backend_runtime_id)
// 2. non-deployable NBR cannot be selected for start (needs_check blocked)
// 3. ready_with_warnings shows warning badge and is selectable
// 4. preview button sends full payload to POST /deployments/preview
// 5. preview panel shows Docker command string
// 6. preview panel shows lint findings when present
// 7. preview panel shows resource_admission findings
// 8. served_model_name stored in config_overrides.parameter_values or service_json
// 9. raw UUID not shown as primary label (display_name resolved)
// 10. save button disabled when can_run is false
// 11. save+start button disabled when can_run is false
```

**Frontend — `web/tests/runtimeBoundaryUi.test.mjs` (UPDATE):**
- Add: `DeploymentWizard` imports `RuntimeParameterEditor`

### Validation commands
```bash
# Backend
go test ./internal/server/api/... -run 'Deployment|Preflight|Preview|RunPlan'
go test ./internal/server/runplan/...

# Frontend
cd web && npm test
cd web && npm run build

# API smoke
curl -X POST http://localhost:18080/api/v1/deployments/preview \
  -H 'Content-Type: application/json' \
  -d '{"name":"test","model_artifact_id":"...","node_backend_runtime_id":"..."}'
```

---

## 5. Workstream D — Start/Stop/Status/Log Stability

### Objective
Verify and harden the full instance lifecycle: start (with preflight gate), status polling, stopped instance display/cleanup, log fetching, instance list auto-refresh. No new features — verify current behavior, fix any gaps found.

### Files to inspect and potentially fix

| File | Action | Detail |
|---|---|---|
| `internal/server/api/deployment_lifecycle_handlers.go` | INSPECT + FIX | `HandleStartDeployment` (line 1061): verify preflight gate, instance creation, runplan write, lease acquire, task dispatch are atomic; `HandleStopDeployment` (line 1333): verify non-terminal instances found, stop tasks dispatched, instance state updated; `HandleListInstances` (line 1449): verify filtering + pagination; `HandleGetNodeRunPlanLogs` (line 1569): verify agent task dispatch + polling |
| `web/src/pages/ModelInstancesPage.vue` | INSPECT + FIX | Verify auto-refresh interval, stopped instance display (show stopped_at, last_error, restart_count), log viewer error handling, instance state transitions shown correctly |
| `web/src/pages/ModelDeploymentsPage.vue` | INSPECT + FIX | Verify start/stop buttons correctly gated on deployment status; dry-run result clears on close; list refreshes after start/stop actions |
| `internal/server/runplan/log_classifier.go` | INSPECT | Verify log classification rules cover common failure patterns for vLLM/SGLang/llama.cpp |
| `internal/agent/runtime/docker_test.go` | INSPECT | Verify docker lifecycle test coverage: create, start, stop, remove, logs |

### Specific checks to perform

1. **Start gate:** Does `HandleStartDeployment` always run preflight before creating instance? Is the transaction truly atomic (instance + runplan + lease + task)?
2. **Stop cleanup:** Does `HandleStopDeployment` handle instances in `running`, `pending`, `error` states? Are GPU leases released? Are agent tasks cancelled?
3. **Stopped instance display:** Does `ModelInstancesPage` show `stopped_at` time? Does it show `last_error` when stopped with error? Does it show `restart_count`?
4. **Log fetch error handling:** What happens when agent is unreachable? When container has no logs? When logs are too large?
5. **Auto-refresh:** Does instance list auto-refresh stop when all instances are terminal?
6. **Status transitions:** Are all state transitions valid (pending → running → stopped, pending → error, running → error → stopped)?

### Expected findings and fixes

Based on source inspection, the lifecycle code is already solid — this workstream is primarily **verification + gap filling**, not redesign.

Anticipated fixes (TBD after detailed inspection):
- Stopped instance row may need `stopped_at` column exposed in table
- Auto-refresh may not stop for terminal states (wasteful polling)
- Log error messages may not distinguish "container not found" vs "agent unreachable"

### Tests to add

**Go tests** — `internal/server/api/instance_lifecycle_test.go` (CREATE):
```go
// 1. start_without_preflight_rejected (if preflight was somehow skipped)
// 2. stop_cleans_up_leases
// 3. stop_cancels_pending_tasks
// 4. instance_state_transitions_valid
// 5. logs_return_error_for_unreachable_agent
// 6. stop_idempotent (stopping already-stopped instance returns success)
```

**Frontend tests** — `web/tests/instanceLifecycle.test.mjs` (CREATE):
```javascript
// 1. stopped instance shows stopped_at column
// 2. error instance shows last_error
// 3. auto-refresh stops for terminal states
// 4. log viewer shows error state when fetch fails
```

### Data/API contract changes: NONE
No new routes, no DB changes. Purely verification + gap fixes within existing contracts.

### Validation commands
```bash
# Backend lifecycle tests
go test ./internal/server/api/... -run 'Instance|Lifecycle|Start|Stop|Log'

# Full suite
go test ./...

# Frontend
cd web && npm test
cd web && npm run build

# Manual smoke: create → preview → start → poll instance → fetch logs → stop → verify stopped state
```

---

## 6. Workstream E — vLLM / SGLang / llama.cpp E2E Regression

### Objective
Establish authoritative current regression baseline with Go tests, frontend tests, API E2E, and runtime smoke evidence — all timestamped and reproducible. Run after B–C–D changes.

### Scope
- Full Go test suite
- Full frontend test suite + build
- API-first E2E (18-step chain without gateway steps)
- Runtime smoke: vLLM, SGLang, llama.cpp (where GPU hardware available)
- Browser smoke: manual verification of key pages (or Playwright if configured)

### Files to create

| File | Action | Detail |
|---|---|---|
| `docs/reports/product-hardening-20260626/evidence/<TS>/baseline/` | CREATE | Baseline test outputs captured before B–D changes (already done: all PASS at c13f91f) |
| `docs/reports/product-hardening-20260626/evidence/<TS>/api-e2e/` | CREATE | API-first E2E result (18 steps, see below) |
| `docs/reports/product-hardening-20260626/evidence/<TS>/runtime-smoke/` | CREATE | vLLM / SGLang / llama.cpp smoke matrix |
| `docs/reports/product-hardening-20260626/evidence/<TS>/browser-smoke/` | CREATE | Screenshots or manual verification notes |
| `docs/reports/product-hardening-20260626/execution/test-and-evidence-inventory.md` | CREATE | Script classification: keep / repair / archive |
| `docs/reports/product-hardening-20260626/execution/final-regression-report.md` | CREATE | Final closeout: commit range, test results, smoke matrix, known blocks, git status |

### API-first E2E chain (18 steps)

1. login → get session cookie
2. CSRF token
3. list nodes
4. list model artifacts
5. list runtime templates (BackendRuntimes)
6. list node runtime configs (NBRs)
7. check/probe NBR
8. create deployment preview (`POST /api/v1/deployments/preview`)
9. verify preview response (can_run, docker_preview, lint, preflight)
10. create deployment (`POST /api/v1/deployments`)
11. dry-run deployment
12. start deployment (`POST /api/v1/deployments/{id}/start`)
13. poll instance until running
14. fetch instance logs
15. test model endpoint (via `POST /api/v1/model-instances/{id}/test`)
16. stop deployment
17. verify instance stopped state
18. verify audit logs recorded

No gateway steps (no `/v1/models`, no `/v1/chat/completions` proxy).

### Runtime smoke matrix

| Backend | Check | Expected |
|---|---|---|
| vLLM | NBR probe → preview → start → health → inference → stop | PASS or DOCUMENTED_BLOCKER |
| SGLang | NBR probe → preview → start → health → inference → stop | PASS or DOCUMENTED_BLOCKER |
| llama.cpp | NBR probe → preview → start → health → inference → stop | PASS or DOCUMENTED_BLOCKER |

Each blocked backend must be classified: external dependency, catalog/config bug, code bug, or environment missing. Fix if code/config bug. Document if external.

### Script inventory (test-and-evidence-inventory.md)

Classify all scripts in `scripts/`:

| Category | Action |
|---|---|
| Current contract E2E (6 files in `scripts/e2e-current-contract-*.sh`) | KEEP — actively maintained |
| Current contract lib (10 files in `scripts/e2e/lib/`) | KEEP — shared library |
| Legacy contract E2E (15 files in `scripts/archive/legacy-contract/`) | ARCHIVE — add README with deprecation notice |
| `scripts/smoke-model-backends.sh` | ARCHIVE — bypasses product API |
| Operational scripts (start/stop/status/etc.) | KEEP |

### Data/API contract changes: NONE
Evidence collection only. No code changes in Workstream E.

### Validation commands (run after B–D)
```bash
# 1. Full Go test suite
go test ./... 2>&1 | tee docs/reports/product-hardening-20260626/evidence/<TS>/go-test.log

# 2. Build
go build ./cmd/server/... && go build ./cmd/agent/...

# 3. Frontend tests + build
cd web && npm test 2>&1 | tee ../docs/reports/product-hardening-20260626/evidence/<TS>/frontend-test.log
cd web && npm run build 2>&1 | tee ../docs/reports/product-hardening-20260626/evidence/<TS>/frontend-build.log

# 4. Diff hygiene
git diff --check

# 5. API E2E
bash scripts/e2e-current-contract-api-dryrun.sh 2>&1 | tee docs/reports/product-hardening-20260626/evidence/<TS>/api-e2e.log

# 6. Runtime smoke (GPU required)
bash scripts/e2e-real-smoke-all-three.sh 2>&1 | tee docs/reports/product-hardening-20260626/evidence/<TS>/runtime-smoke.log

# 7. Final git status
git status --short
```

---

## 7. Workstream F — Naming Cleanup (Safe Only)

### Objective
Fix user-visible labels, i18n strings, table columns, and drawer titles to use consistent vocabulary. Do NOT rename component files or route paths.

### Downgraded scope — what is NOT changed
- Route path `/runner-configs` stays as-is (no redirect, no rename)
- Component file `RunnerConfigsPage.vue` stays as-is (no file rename)
- Component file `BackendRuntimesPage.vue` stays as-is (no file rename)
- Route name `RunnerConfigs` stays as-is
- Only i18n display values, menu labels, page titles, table column labels, and hardcoded strings change

### Target vocabulary

| Internal entity | zh-CN UI label | en-US UI label | Where visible |
|---|---|---|---|
| BackendRuntime | 运行模板 | Runtime Template | `/runtimes` page title, menu, table |
| NodeBackendRuntime | 节点运行配置 | Node Runtime Config | `/runner-configs` page title, menu, table, deployment selector |
| ModelDeployment | 模型部署 | Deployment | `/models/deployments` page |
| ModelInstance | 模型实例 | Instance | `/models/instances` page |
| ResolvedRunPlan | 运行计划 | Run Plan | preview/detail panels |
| ConfigSet | 技术配置 | Technical Config | drawer section title (technical label only) |

### Files to modify

| File | Action | Detail |
|---|---|---|
| `web/src/locales/zh-CN.ts` | CHANGE | Change `runnerConfigs.title` from `"运行配置"` → `"节点运行配置"`; change `runtimes.title` (if not already) → `"运行模板"`; change `deployments.existingOverrides` to remove raw "ConfigSet"; change `deployments.overrideHint` to remove raw "ConfigSet"; change `deployments.nbrTemplateGroup` → `deployments.runtimeTemplateGroup` (remove "NBR" acronym); change `deployments.runPlanSourceNote` → remove raw "NBR"; change `help.runPlanTitle` → translate "RunPlan" to "运行计划" |
| `web/src/locales/en-US.ts` | CHANGE | Mirror all zh-CN changes: `runnerConfigs.title` → `"Node Runtime Configs"`; `runtimes.title` → `"Runtime Templates"`; `deployments.nbrTemplateGroup` → `deployments.runtimeTemplateGroup`; etc. |
| `web/src/layouts/ConsoleLayout.vue` | CHANGE | Menu item label for `/runner-configs`: use `$t('runnerConfigs.title')` (already i18n — label changes via locale files); menu item label for `/runtimes`: use `$t('runtimes.title')` |
| `web/src/pages/RunnerConfigsPage.vue` | CHANGE | Table column `prop="backend_runtime_id"` → resolve and show BackendRuntime `name` or `display_name` (not raw UUID); change hardcoded `title="ConfigSet"` on JsonViewer → `$t('common.technicalConfig')`; add i18n key `common.technicalConfig` = `"技术配置"` / `"Technical Config"` |
| `web/src/pages/BackendRuntimesPage.vue` | CHANGE | Change hardcoded `title="ConfigSet"` on JsonViewer → `$t('common.technicalConfig')` |
| `web/src/pages/ModelDeploymentsPage.vue` | CHANGE | Change hardcoded `title="Deployment ConfigSet"` on JsonViewer → `$t('deployments.technicalConfig')`; table column `prop="source_node_backend_runtime_id"` → resolve and show NBR `display_name` (not raw UUID); runtime selector label → `$t('deployments.nodeRuntimeConfig')` |
| `web/src/pages/ModelInstancesPage.vue` | CHANGE | Replace `t('runnerConfigs.advancedJson')` → `t('common.advancedJson')` |
| `web/src/pages/BackendsPage.vue` | CHANGE | Change hardcoded `title="ConfigSet"` on JsonViewer → `$t('common.technicalConfig')` |
| `web/src/components/common/RuntimeParameterEditor.vue` | CHANGE | Change hardcoded `title="ConfigSet"` → `$t('common.parameterConfiguration')` |
| `docs/engineering/naming-dictionary.md` | CREATE | Concept table with owner layer, user-editability, copy semantics, preferred label, forbidden terms |
| `docs/api/openapi.yaml` | CHANGE | Update description text: "runner config" → "node runtime config", "runtime config" → "runtime template" where appropriate |
| `docs/README.md` | CHANGE | Update concept references to match dictionary |

### i18n keys to add

```javascript
// zh-CN.ts additions
common: {
  technicalConfig: "技术配置",
  parameterConfiguration: "参数配置",
  advancedJson: "高级 JSON",
}

// en-US.ts additions
common: {
  technicalConfig: "Technical Config",
  parameterConfiguration: "Parameter Configuration",
  advancedJson: "Advanced JSON",
}
```

### i18n keys to change (value only, key stays)

| Key | Old zh-CN | New zh-CN | Old en-US | New en-US |
|---|---|---|---|---|
| `runnerConfigs.title` | 运行配置 | 节点运行配置 | Runtime Configs | Node Runtime Configs |
| `runtimes.title` | (verify) | 运行模板 | (verify) | Runtime Templates |
| `deployments.nbrTemplateGroup` | NBR 静态模板预览 | 运行模板快照 | NBR Template (Static Snapshot) | Runtime Template Snapshot |
| `deployments.runPlanSourceNote` | 参数按来源分组：NBR 模板 → ... | 参数按来源分组 | Parameters grouped by source: NBR template → ... | Parameters grouped by source |
| `help.runPlanTitle` | RunPlan / Docker 预览 | 运行计划 / Docker 预览 | RunPlan / Docker Preview | Run Plan / Docker Preview |
| `deployments.existingOverrides` | 部署级 ConfigSet 覆盖 | 部署级参数覆盖 | Deployment Config Overrides | Deployment Config Overrides |
| `deployments.overrideHint` | ...物化到部署 ConfigSet。 | ...保存到部署配置。 | ...materialized into the deployment ConfigSet. | ...saved to deployment config. |

### Data/API contract changes: NONE
Purely frontend labels + i18n + docs.

### Tests to add/update

| Test file | Change |
|---|---|
| `web/tests/i18nKeys.test.mjs` | NO CODE CHANGE (keys not renamed, only values changed — key count stays same) |
| `web/tests/i18nMissingKeys.test.mjs` | NO CHANGE (auto-validates new keys) |
| `web/tests/runtimeBoundaryUi.test.mjs` | UPDATE: change any assertions that reference old hardcoded strings |
| `web/tests/namingDictionary.test.mjs` | CREATE: assert no raw "ConfigSet" in Vue template text content (exclude test files); assert no raw "NBR" in i18n zh-CN/en-US values; assert menu labels match dictionary |

### Validation commands
```bash
# No raw "ConfigSet" in user-facing labels
grep -rn "ConfigSet" web/src/pages/ web/src/components/ web/src/layouts/ web/src/locales/
# Expected: zero matches in text content (may appear in variable names, data keys)

# No raw "NBR" in i18n values
grep -rn '"NBR' web/src/locales/
# Expected: zero matches

# Raw UUIDs not in table column props
grep -rn 'prop="backend_runtime_id"\|prop="source_node_backend_runtime_id"' web/src/pages/
# Expected: zero matches (replaced with resolved name columns)

cd web && npm test
cd web && npm run build
```

---

## 8. Workstream G — OpenAI Gateway Future Notes (Document Only)

### Objective
Document design boundaries, dependency conditions, and implementation suggestions for future OpenAI-compatible gateway, API key management, usage metering, and billing. **No code is written in this workstream.**

### Deliverable

**Single file:** `docs/reports/product-hardening-20260626/execution/future-openai-gateway-notes.md`

See that file for full content. Summary of what it covers:

1. **Design boundaries:** tenant scoping, API key auth model, model routing policy, usage record schema, billing integration points
2. **Dependency conditions:** requires stable deployment/instance lifecycle (Workstreams B–D completed), requires running deployments to proxy, requires at least one healthy instance per deployment
3. **Architecture sketch:** gateway routes outside `/api/v1`, Bearer `lak-*` keys, model resolution order, proxy timeout/error handling, usage capture from backend response
4. **DB tables needed:** `api_keys`, `gateway_usage_records` (full DDL provided in notes)
5. **API routes needed:** `GET /v1/models`, `POST /v1/chat/completions`, CRUD `/api/v1/api-keys`, `GET /api/v1/gateway/usage`
6. **UI pages needed:** `ApiKeysPage.vue`, `GatewayUsagePage.vue`
7. **Security considerations:** key hashing (bcrypt), redaction in logs, prefix-only display after creation, audit on all key operations
8. **Implementation prerequisites:** all items in this document's Workstreams B–E must be complete and stable

**No code changes. No DB changes. No route additions. No frontend pages created. Document only.**

---

## 9. DB/Schema Impact Summary

| Workstream | DB Change | Type |
|---|---|---|
| B — NBR / Runtime Parameters | None (ConfigSet JSON within existing columns) | — |
| C — Deployment UI + Preview | None | — |
| D — Start/Stop/Status/Logs | None | — |
| E — E2E Regression | None | — |
| F — Naming Cleanup | None | — |
| G — Gateway Future Notes | None (document only) | — |

**DB change: NONE.** No new tables, no schema changes, no migration, no clean rebuild required for this hardening scope. Gateway tables (`api_keys`, `gateway_usage_records`) are deferred to future workstream.

---

## 10. API Contract Impact Summary

| Method | Path | Workstream | Change |
|---|---|---|---|
| POST | `/api/v1/deployments/preview` | C | **ADD** |
| PATCH | `/api/v1/backend-runtimes/{id}` | B | ENHANCE (accept full parameter values; block if system-managed) |
| PATCH | `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}` | B | ENHANCE (accept node-specific parameter values) |
| PATCH | `/api/v1/deployments/{id}` | B | ENHANCE (accept full config_overrides) |

**Routes explicitly NOT added in this scope:**
- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /api/v1/api-keys`
- `GET /api/v1/api-keys`
- `POST /api/v1/api-keys/{id}/disable`
- `DELETE /api/v1/api-keys/{id}`
- `GET /api/v1/gateway/usage`

No existing routes removed. No breaking changes to existing route contracts. 1 new route, 3 enhanced routes.

---

## 11. UI Contract Impact Summary

| Page/Component | Workstream | Change |
|---|---|---|
| `RunnerConfigsPage.vue` | B, F | Add `RuntimeParameterEditor` in create/detail; fix labels + remove raw UUID column |
| `BackendRuntimesPage.vue` | B, F | Add `RuntimeParameterEditor` in detail (read-only for system, editable for user); fix labels |
| `ModelDeploymentsPage.vue` | B, C, F | Replace dialog with wizard; wire `DeploymentOverrideEditor`; fix labels + remove raw UUID column |
| `ModelArtifactsPage.vue` | B | Relabel + remove runtime args from placeholder |
| `ModelInstancesPage.vue` | D, F | Stopped instance display; fix i18n reference |
| `BackendsPage.vue` | F | Fix hardcoded "ConfigSet" title |
| `ConsoleLayout.vue` | F | Menu label updates (via i18n) |
| `RuntimeParameterEditor.vue` | B, F | Major enhancement (props/emits/behavior); fix hardcoded "ConfigSet" title |
| `DeploymentWizard.vue` | C | **NEW** — 6-section wizard |
| `ModelSelector.vue` | C | **NEW** |
| `NodeRuntimeSelector.vue` | C | **NEW** |
| `DeploymentServiceEditor.vue` | C | **NEW** |
| `DeploymentOverrideEditor.vue` | C | **NEW** |
| `DeploymentPreviewPanel.vue` | C | **NEW** |

Pages NOT created in this scope: `ApiKeysPage.vue`, `GatewayUsagePage.vue`.

---

## 12. Commit Plan

```
Commit 1: "feat: integrate runtime parameter editor across BackendRuntime/NBR/Deployment layers (B)"
  - RuntimeParameterEditor.vue enhancement
  - Editor in BackendRuntimesPage, RunnerConfigsPage, ModelDeploymentsPage
  - Lint rules (7 new rules in lint.go)
  - ModelArtifactsPage cleanup
  - Catalog YAML verification/fixes
  - Go tests (runtime_parameter_layering_test.go, model_artifact_boundary_test.go)
  - Frontend tests (runtimeParameterEditor.test.mjs, modelCapabilities update)

Commit 2: "feat: add deployment preview endpoint and wizard UI (C)"
  - POST /api/v1/deployments/preview (deployment_preview_handlers.go)
  - Refactor preflight_handlers.go (extract shared function)
  - 6 new Vue components (DeploymentWizard + 5 sub-components)
  - Go tests (deployment_preview_test.go)
  - Frontend tests (deploymentWizard.test.mjs)

Commit 3: "fix: harden instance lifecycle display and edge cases (D)"
  - Instance stopped state display fixes
  - Auto-refresh terminal state optimization
  - Log fetch error handling
  - Go tests (instance_lifecycle_test.go)
  - Frontend tests (instanceLifecycle.test.mjs)

Commit 4: "chore: final regression evidence and naming cleanup (E + F)"
  - Evidence directories (baseline, api-e2e, runtime-smoke, browser-smoke)
  - test-and-evidence-inventory.md, final-regression-report.md
  - i18n label fixes (runnerConfigs, deployments, common keys)
  - Hardcoded "ConfigSet"/"RunPlan"/"NBR" removal from UI
  - docs/engineering/naming-dictionary.md
  - Legacy script archival labels
  - Frontend tests (namingDictionary.test.mjs)

Commit 5: "docs: add OpenAI gateway future design notes (G)"
  - future-openai-gateway-notes.md
  - No code changes
```

---

## 13. File Count Summary

| Workstream | Files CREATE | Files MODIFY | Files RENAME | Total |
|---|---|---|---|---|
| B — Runtime Parameters | 0 | 11 (editor, 4 pages, 3 handlers, lint, resolver, 3 catalog YAMLs) + 3 test files | 0 | 14 |
| C — Deployment UI | 7 (1 handler + 6 Vue components) + 2 test files | 4 (ModelDeploymentsPage, deployments.ts, router, preflight handler, openapi) | 0 | 11 |
| D — Start/Stop/Logs | 0 | 2 (lifecycle handler, instances page) + 2 test files | 0 | 4 |
| E — E2E Regression | 5 (evidence dirs + inventory + report) | 0 | 0 | 5 |
| F — Naming Cleanup | 1 (dictionary) + 1 test | 8 (2 locales, layout, 4 pages, editor, openapi, docs) | 0 | 10 |
| G — Gateway Notes | 1 (future notes doc) | 0 | 0 | 1 |
| **Total** | **17** | **25** | **0** | **~42** |

---

## 14. Revised Top 10 Concrete Code Changes

1. **Enhance `RuntimeParameterEditor.vue`** with 4 new props (`layer`, `baseValues`, `showSource`, `showAdvanced`), 1 new emit (`validate`), required-field lock, optional enable/disable toggle with value retention, backend/vendor filter, source trace (Workstream B)
2. **Wire `RuntimeParameterEditor.vue`** into `BackendRuntimesPage.vue` (system templates read-only, user templates editable with clone), `RunnerConfigsPage.vue` (inherited + NBR override diff), and `ModelDeploymentsPage.vue` via `DeploymentOverrideEditor` wrapper (Workstream B)
3. **Add 7 lint rules** to `runplan/lint.go`: duplicate args, env/CLI conflict, platform arg override, unsupported param, disabled field applied, missing required, vendor incompatibility (Workstream B)
4. **Add `POST /api/v1/deployments/preview`** — new handler in `deployment_preview_handlers.go`, shares `preflightDeployment()` resolver with start, returns `{can_run, run_plan, docker_preview, lint, resource_admission, preflight, source_trace}` (Workstream C)
5. **Refactor `preflight_handlers.go`** — extract `performDeploymentPreflight()` shared by `HandlePreflightDeployments`, new `HandleDeploymentPreview`, and `HandleStartDeployment` (Workstream C)
6. **Create `DeploymentWizard.vue`** — 6-section guided workflow replacing thin create dialog in `ModelDeploymentsPage.vue`: Model → Node Runtime Config → Service → Resource/Placement → Overrides → Preview (Workstream C)
7. **Harden instance lifecycle display** — stopped instance shows `stopped_at` + `last_error` + `restart_count`; auto-refresh stops for terminal states; log errors distinguish "container not found" vs "agent unreachable" (Workstream D)
8. **Strip "ConfigSet"/"RunPlan"/"NBR" from user-facing labels** — replace hardcoded `title="ConfigSet"` on 4 pages + 1 component with i18n keys; fix i18n string values to remove raw "NBR"/"RunPlan"; resolve raw UUID table columns to display names (Workstream F)
9. **Create `docs/engineering/naming-dictionary.md`** — concept table with owner layer, editability, copy semantics, preferred zh-CN/en-US labels, forbidden stale terms (Workstream F)
10. **Run full E2E regression** — Go test suite, frontend test suite, API E2E (18 steps), vLLM/SGLang/llama.cpp runtime smoke, evidence collection, final closeout report (Workstream E)
