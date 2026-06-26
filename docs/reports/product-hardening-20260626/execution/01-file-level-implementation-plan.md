# 01 — File-Level Implementation Plan

Generated: 2026-06-26 | Based on: `00-current-code-inventory.md`

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

## 2. Implementation Order

Workstreams should be executed in this order to minimize conflicts:

```
A (naming) → B (deployment UI) → C (runtime parameters) → D (gateway) → E (regression evidence)
```

Rationale: A establishes vocabulary used by B and C. B creates the wizard that C's parameter editor integrates into. D depends on B (needs running deployments to proxy). E runs last to capture final state.

Non-conflicting workstreams: A and D's DB schema can be pre-staged. C's backend changes are independent.

---

## 3. Workstream A — Naming Debt

### Objective
Apply consistent vocabulary: BackendRuntime → "Runtime Template", NodeBackendRuntime → "Node Runtime Config", remove "ConfigSet"/"RunPlan"/"NBR" from user-facing text.

### Files to modify

| File | Action | Detail |
|---|---|---|
| `web/src/router/index.ts` | CHANGE | Rename route `RunnerConfigs` → `NodeRuntimeConfigs`, path `/runner-configs` → `/node-runtime-configs` |
| `web/src/layouts/ConsoleLayout.vue` | CHANGE | Update menu item index from `/runner-configs` to `/node-runtime-configs`, change i18n key |
| `web/src/pages/BackendRuntimesPage.vue` | CHANGE | Rename component to `RuntimeTemplatesPage.vue` (file rename); change JsonViewer title from `"ConfigSet"` to `$t('runtimes.technicalConfig')` |
| `web/src/pages/RunnerConfigsPage.vue` | CHANGE | Rename component to `NodeRuntimeConfigsPage.vue` (file rename); change page title i18n key from `runnerConfigs.title` to `nodeRuntimeConfigs.title`; remove raw `backend_runtime_id` table column — show display name; change JsonViewer title from `"ConfigSet"` to `$t('nodeRuntimeConfigs.technicalConfig')`; change table column label for NBR name |
| `web/src/pages/ModelDeploymentsPage.vue` | CHANGE | Change JsonViewer title from `"Deployment ConfigSet"` to `$t('deployments.technicalConfig')`; remove raw `source_node_backend_runtime_id` table column — resolve and show display name; change runtime selector label from `$t('deployments.runtime')` to `$t('deployments.nodeRuntimeConfig')`; add `display_name` lookup via join/map from `nodeRuntimes` |
| `web/src/pages/ModelInstancesPage.vue` | CHANGE | Replace `t('runnerConfigs.advancedJson')` with `t('common.advancedJson')` |
| `web/src/pages/BackendsPage.vue` | CHANGE | Change JsonViewer title from `"ConfigSet"` to `$t('backends.technicalConfig')` |
| `web/src/components/common/RuntimeParameterEditor.vue` | CHANGE | Change collapsible title from `"ConfigSet"` to `$t('common.parameterConfig')` |
| `web/src/locales/zh-CN.ts` | CHANGE | Rename `runnerConfigs` → `nodeRuntimeConfigs`; update all sub-keys to avoid "NBR"/"RunPlan"/"ConfigSet" raw text; change `nbrTemplateGroup` → `runtimeTemplateGroup`, `runPlanSourceNote` → `paramSourceNote`; change `deployments.existingOverrides` to remove "ConfigSet"; change `deployments.overrideHint` to remove "ConfigSet" |
| `web/src/locales/en-US.ts` | CHANGE | Mirror all zh-CN changes in English |
| `web/src/api/runtimes.ts` | CHANGE | Rename `BackendRuntime` interface to `RuntimeTemplate` (type alias kept for compat) |
| `web/src/api/deployments.ts` | MINOR | No functional change — field names align with backend JSON |
| `docs/engineering/naming-dictionary.md` | CREATE | New file with concept table per 03-workstream-a-naming-debt.md Step A2 |
| `docs/api/openapi.yaml` | CHANGE | Update description text to use "runtime template" / "node runtime config" / "run plan" vocabulary; remove "runner-config" from descriptions |
| `docs/README.md` | CHANGE | Update references to `RunnerConfigsPage` → `NodeRuntimeConfigsPage` |
| `docs/CURRENT.md` | CHANGE | Same update |

### Functions/components changed
- `BackendRuntimesPage.vue` → `RuntimeTemplatesPage.vue`: component name, route reference, i18n key namespace
- `RunnerConfigsPage.vue` → `NodeRuntimeConfigsPage.vue`: component name, table column for `backend_runtime_id` → resolved template name, route reference, i18n key namespace
- `ConsoleLayout.vue`: menu `index` attribute
- `router/index.ts`: route `path`, `name`, component import path, `meta.title`

### Data/API contract changes: NONE
No API routes or DB schema change. Purely frontend + docs.

### Tests to add/update

| Test file | Change |
|---|---|
| `web/tests/runtimeBoundaryUi.test.mjs` | UPDATE: change references from `RunnerConfigsPage` to `NodeRuntimeConfigsPage` |
| `web/tests/i18nKeys.test.mjs` | NO CHANGE (key count may change — update expected counts) |
| `web/tests/i18nMissingKeys.test.mjs` | NO CHANGE (auto-validates) |
| `web/tests/namingDictionary.test.mjs` | CREATE: assert `RunnerConfig` term absent from Vue templates; assert `ConfigSet` absent from user-facing labels; assert all pages use dictionary terms |
| `docs/engineering/naming-dictionary.md` | CREATE: reference document, not a test |

### Validation commands
```bash
grep -r "ConfigSet" web/src/pages/ web/src/components/ web/src/layouts/ web/src/locales/  # should return ONLY test files or intentionally retained internal uses
grep -r "RunnerConfig" web/src/router/ web/src/layouts/ web/src/pages/  # should return ZERO matches
grep -r "NBR" web/src/locales/  # should return ZERO matches in user-facing text
cd web && npm test
cd web && npm run build
```

---

## 4. Workstream B — Model Deployment UI

### Objective
Replace the thin create dialog in `ModelDeploymentsPage.vue` with a guided deployment wizard showing model facts, NBR status, service config, overrides, RunPlan preview, and blockers.

### Files to modify

| File | Action | Detail |
|---|---|---|
| `web/src/pages/ModelDeploymentsPage.vue` | REPLACE | Replace thin create dialog (lines 25-45) with wizard sections; add preview panel; keep list table and detail drawer; add NBR status column with color tags; add model name column (resolve from artifact list) |
| `web/src/components/deployments/DeploymentWizard.vue` | CREATE | Multi-section wizard component: 6 steps as defined in 04-workstream-b Step B4 |
| `web/src/components/deployments/ModelSelector.vue` | CREATE | Model selection: display name, format, task type, capabilities, location warning |
| `web/src/components/deployments/NodeRuntimeSelector.vue` | CREATE | NBR selection: display name, node, backend, version, image, status tag, status reason, block non-deployable |
| `web/src/components/deployments/DeploymentServiceEditor.vue` | CREATE | Service config: host_port, container_port, served_model_name, endpoint preview |
| `web/src/components/deployments/DeploymentOverrideEditor.vue` | CREATE | Wraps RuntimeParameterEditor for deployment override layer; shows inherited NBR values as source; marks overridden values |
| `web/src/components/deployments/DeploymentPreviewPanel.vue` | CREATE | Preview: can_run, errors/warnings, lint, resource admission, Docker command, RunPlan JSON via JsonViewer, source trace |
| `web/src/api/deployments.ts` | CHANGE | Add `previewDeployment(data)` function calling `POST /deployments/preview`; add `getDeployment(id)` type |
| `internal/server/api/router.go` | ADD | Register `POST /api/v1/deployments/preview` → `HandleDeploymentPreview` with `mdWriteChain` |
| `internal/server/api/deployment_preview_handlers.go` | CREATE | `HandleDeploymentPreview`: same logic as `preflightDeployment()` but accepts create payload without existing deployment ID; returns `preflightResult` |
| `internal/server/api/preflight_handlers.go` | REFACTOR | Extract shared `preflightCreatePayload()` function used by both `HandlePreflightDeployments` and new `HandleDeploymentPreview` |
| `docs/api/openapi.yaml` | ADD | Schema for `DeploymentPreview` request/response, `POST /deployments/preview` path |

### Functions/components changed

**New Go handler** — `HandleDeploymentPreview`:
- File: `internal/server/api/deployment_preview_handlers.go`
- Signature: `func (h *AgentHandler) HandleDeploymentPreview(w http.ResponseWriter, r *http.Request)`
- Input: same as create payload (`name`, `model_artifact_id`, `node_backend_runtime_id`, `service_json`, `placement_json`, `config_overrides`)
- Logic: validate NBR → validate artifact → check model location → check GPU → run `runplan.Resolve()` → lint → build command preview → return `preflightResult`
- Does NOT check `deployment_id` in URL, does NOT write to DB
- MUST use same resolver path as `HandleStartDeployment` (single source of truth)

**Refactored code** — `preflight_handlers.go`:
- Extract `performDeploymentPreflight(ctx, db, payload) (*preflightResult, error)` from the shared logic in `HandlePreflightDeployments`
- Both old handler and new handler call the same function

**New Vue components** — 6 deployment wizard components as listed above.

### Data/API contract changes

| Method | Path | Change | Handler |
|---|---|---|---|
| POST | `/api/v1/deployments/preview` | **ADD** | `HandleDeploymentPreview` |

Request body: `{ name, display_name?, model_artifact_id, node_backend_runtime_id, placement_json?, service_json?, config_overrides? }`

Response body: `{ can_run, run_plan, docker_preview, lint: { status, findings[] }, resource_admission: { status, findings[] }, preflight: { status, errors[], warnings[] }, source_trace }`

No existing routes changed. No routes removed.

### Tests to add/update

**Backend tests** — `internal/server/api/deployment_preview_test.go` (CREATE):
```go
// Test cases:
// 1. preview_rejects_legacy_backend_runtime_id
// 2. preview_rejects_non_ready_nbr
// 3. preview_accepts_ready_with_warnings
// 4. preview_blocks_missing_model_location
// 5. preview_blocks_host_port_conflict
// 6. preview_and_start_use_same_resolver_path
// 7. preview_disabled_parameter_not_applied
// 8. preview_deployment_override_wins_over_nbr
```

**Backend tests** — `internal/server/api/preflight_handlers_test.go` (UPDATE):
- Add test: `preflight_rejects_standalone_preview_without_nbr` (ensure preflight still works for existing deployments)

**Frontend tests** — `web/tests/deploymentWizard.test.mjs` (CREATE):
```javascript
// Assertions:
// 1. create payload uses node_backend_runtime_id
// 2. non-deployable NBR cannot be selected for start
// 3. ready_with_warnings shows warning and is selectable
// 4. preview button sends full payload to /deployments/preview
// 5. preview panel shows Docker command
// 6. preview panel shows lint/preflight findings
// 7. served_model_name in service_json or config_overrides
// 8. raw UUID not shown as primary label (display_name required)
```

**Frontend tests** — `web/tests/runtimeBoundaryUi.test.mjs` (UPDATE):
- Add assertion: `DeploymentWizard` imports `RuntimeParameterEditor`

### Validation commands
```bash
# Backend
go test ./internal/server/api/... -run 'Deployment|Preflight|Preview|RunPlan'
go test ./internal/server/runplan/...

# Frontend
cd web && npm test
cd web && npm run build

# API smoke (manual or via script)
curl -X POST http://localhost:18080/api/v1/deployments/preview \
  -H 'Content-Type: application/json' \
  -d '{"name":"test","model_artifact_id":"...","node_backend_runtime_id":"..."}'
# Expect: {... "can_run": ..., "docker_preview": "...", "lint": {...} ...}
```

---

## 5. Workstream C — Runtime Parameter Completeness

### Objective
Surface ALL backend parameters from catalog YAMLs in the RuntimeParameterEditor, integrate editor into BackendRuntime, NodeBackendRuntime, and Deployment pages at correct layers. Clean model page of runtime args.

### Files to modify

| File | Action | Detail |
|---|---|---|
| `web/src/pages/BackendRuntimesPage.vue` | CHANGE | Add `RuntimeParameterEditor` component in detail drawer for user-managed templates; read-only for system templates; add "Clone" button for system templates |
| `web/src/pages/RunnerConfigsPage.vue` | CHANGE (becomes NodeRuntimeConfigsPage after A) | Add `RuntimeParameterEditor` in create/edit: show inherited values from BackendRuntime; editable for node-specific overrides; show source/diff from template |
| `web/src/pages/ModelDeploymentsPage.vue` | CHANGE | Wire `DeploymentOverrideEditor` (from Workstream B) which wraps `RuntimeParameterEditor`; show NBR values as base, deployment overrides as diff |
| `web/src/pages/ModelArtifactsPage.vue` | CHANGE | Keep `parameter_defaults` textarea but relabel to "Model Facts and Hints" (remove serving-related placeholder text); remove `--max-model-len`, `--served-model-name`, `--gpu-memory-utilization` from placeholder; add explicit hint: "Model facts only — Docker/runtime parameters belong in Runtime Template or Deployment configuration" |
| `web/src/components/common/RuntimeParameterEditor.vue` | CHANGE | Add props: `layer` (backend_runtime/node_backend_runtime/deployment), `baseValues` (inherited values from parent layer), `showSource`, `showAdvanced`; add emits: `validate`; add behavior: required fields locked enabled, optional fields have enable/disable toggle, disabled values retained in model but excluded from final args, validation errors shown inline, advanced groups collapsible, source/diff from base visible; add backend applicability check (`backend`/`vendor` fields); filter parameters based on enabled backends |
| `internal/server/api/runtime_handlers.go` | CHANGE | Ensure `PATCH /api/v1/backend-runtimes/{id}` saves `config_set_json` with parameter values; ensure `PATCH /api/v1/nodes/{id}/backend-runtimes/{nbr_id}` saves node-specific parameter values |
| `internal/server/api/deployment_lifecycle_handlers.go` | CHANGE | Ensure `HandlePatchDeployment` properly processes `config_overrides` with full parameter_values/disabl_parameters/env arrays |
| `internal/server/runplan/lint.go` | CHANGE | Add lint checks: duplicate CLI flag across layers, env/CLI conflict, user extra_arg overrides platform-owned arg, unsupported backend param, disabled field applied, missing required field, vendor-incompatible field |
| `internal/server/runplan/resolver.go` | MINOR | Ensure lint results are returned in Resolve output (add `LintResult` to return or make available via separate call) |
| `configs/backend-catalog/versions/vllm/vllm-v0.23.0.yaml` | MINOR | Verify all 17 args_schema items have correct `group`, `type`, `advanced` fields |
| `configs/backend-catalog/versions/sglang/sglang-v0.5.13.post1.yaml` | MINOR | Add `--chunked-prefill-size` and `--attention-backend` to args_schema (currently only in resource_controls) |
| `configs/backend-catalog/versions/llamacpp/llamacpp-b9700.yaml` | MINOR | Verify no fake `gpu_memory_fraction` surfaces (already correct: `supported: false`) |

### Functions/components changed

**RuntimeParameterEditor.vue** — major enhancement:
- New prop `layer: string` — sets context for source display
- New prop `baseValues: ParameterValue[]` — inherited values from parent layer
- New prop `showSource: boolean` — toggles source/diff column
- New prop `showAdvanced: boolean` — toggles advanced group visibility
- New emit `validate: (errors: ValidationError[]) => void`
- New behavior: `backend`/`vendor` filter — only show params applicable to current backend
- New behavior: required fields have lock icon, disabled toggle
- New behavior: source trace per parameter (BackendVersion → BackendRuntime → NBR → Deployment)

**BackendRuntimesPage.vue** — add interactive editing:
- Install `RuntimeParameterEditor` in drawer
- System-managed (readonly) runtimes: editor in read-only mode
- User-managed runtimes: editor editable
- "Clone from system template" button → creates new user-managed runtime with copied values

**NodeRuntimeConfigsPage.vue** (renamed in Workstream A) — add interactive editing:
- Install `RuntimeParameterEditor` in create and edit
- Show inherited BackendRuntime values as `baseValues` (read-only)
- NBR-specific overrides shown as diff
- Save emits `PATCH` with NBR-specific values only

**ModelDeploymentsPage.vue** — wire parameter editing:
- `DeploymentOverrideEditor` wraps `RuntimeParameterEditor`
- NBR values as base, deployment values as overrides
- Preview panel resolves effective RunPlan with deployment overrides applied

**lint.go** — new lint rules:
- `LintRuleDuplicateArg`: detect same CLI flag in extra_args and parameter_values
- `LintRuleEnvCLIConflict`: detect env override conflicting with CLI arg
- `LintRulePlatformArgOverridden`: detect user overriding `--host`/`--port`/`--model`
- `LintRuleUnsupportedParam`: detect param not in backend's args_schema
- `LintRuleVendorIncompatible`: detect param not supported by vendor (e.g., CUDA param on MetaX)

### Data/API contract changes

| Method | Path | Change |
|---|---|---|
| PATCH | `/api/v1/backend-runtimes/{id}` | ENHANCE: accept full `config_set` parameter values |
| PATCH | `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}` | ENHANCE: accept node-specific `config_set` overrides |
| PATCH | `/api/v1/deployments/{id}` | ENHANCE: accept full `config_overrides` with `parameter_values`, `disabled_parameters`, `env` |

No new routes. No routes removed. Existing routes already support these payloads — frontend needs to send them.

### Tests to add/update

**Go tests:**
```bash
# New file: internal/server/api/runtime_parameter_layering_test.go
# Test cases:
# 1. nbr_value_wins_over_backend_runtime
# 2. deployment_override_wins_over_nbr
# 3. disabled_optional_value_retained_not_applied
# 4. required_field_cannot_be_disabled
# 5. vllm_gpu_memory_fraction_validates_0.1_to_0.95
# 6. sglang_mem_fraction_static_validates
# 7. llamacpp_no_fake_memory_fraction
# 8. vendor_incompatible_param_linted

# New file: internal/server/api/model_artifact_boundary_test.go
# Test cases:
# 1. model_page_parameter_defaults_not_used_as_runtime_args
# 2. deployment_does_not_mutate_nbr
```

**Frontend tests:**
```javascript
// New file: web/tests/runtimeParameterEditor.test.mjs
// Assertions:
// 1. required fields rendered as locked enabled
// 2. optional fields have enable checkbox
// 3. disabled field value retained in model
// 4. disabled field value NOT emitted in config_overrides
// 5. vLLM memory fraction renders with range 0.1-0.95
// 6. SGLang memory fraction renders
// 7. llama.cpp does NOT render memory fraction
// 8. source/diff shows inherited vs override values
// 9. validation errors shown inline for out-of-range values

// New file: web/tests/modelCapabilities.test.mjs (UPDATE existing)
// Assertion: model page does not expose backend runtime args
```

### Validation commands
```bash
# Backend: parameter layering
go test ./internal/server/api/... -run 'Parameter|ConfigSet|Lint'

# Backend: runplan resolution with all parameters
go test ./internal/server/runplan/... -run 'vllm|sglang|llamacpp|Resource|Lint'

# Frontend
cd web && npm test
cd web && npm run build

# Verify model page has no runtime args
grep -n "max-model-len\|gpu-memory-utilization\|served-model-name" web/src/pages/ModelArtifactsPage.vue
# Expect: ZERO matches (placeholders only; labels removed)
```

---

## 6. Workstream D — OpenAI Gateway, Audit, Metering

### Objective
Add tenant-scoped OpenAI-compatible gateway (`GET /v1/models`, `POST /v1/chat/completions`), API key management, usage recording, and audit logging. Minimal product boundary for future billing.

### Files to modify/create

#### DB schema

| File | Action | Detail |
|---|---|---|
| `internal/server/db/db.go` | CHANGE | ADD `CREATE TABLE api_keys (...)` and `CREATE TABLE gateway_usage_records (...)` with indexes per 06-workstream-d Step D3 |

#### Backend — new files

| File | Action | Detail |
|---|---|---|
| `internal/server/api/gateway_auth.go` | CREATE | `GatewayAuthMiddleware`: parse Bearer token → hash → lookup active key → verify tenant/expiry/status → attach to context → update `last_used_at` |
| `internal/server/gateway/model_resolver.go` | CREATE | `ResolveGatewayTarget(tenantID, requestedModel, keyScopes, db) (*GatewayTarget, error)`: tenant-owned deployments only, running healthy instances only, unambiguous match, return `GatewayTarget` struct |
| `internal/server/api/gateway_handlers.go` | CREATE | `HandleGatewayListModels` (GET /v1/models) and `HandleGatewayChatCompletions` (POST /v1/chat/completions): resolve target → proxy request → parse usage → write audit + usage record → return response |
| `internal/server/api/api_key_handlers.go` | CREATE | `HandleCreateAPIKey`, `HandleListAPIKeys`, `HandleDisableAPIKey`, `HandleDeleteAPIKey`: CRUD for API keys with hash storage, prefix exposure, full key shown once |
| `internal/server/api/gateway_usage_handlers.go` | CREATE | `HandleListGatewayUsage` (GET /api/v1/gateway/usage): paginated query with filters (time range, deployment_id, model_artifact_id, api_key_id, success, route), summary stats |
| `internal/server/models/api_key.go` | CREATE | `APIKey` struct + `GatewayUsageRecord` struct |
| `internal/common/types/gateway.go` | CREATE | `GatewayTarget` struct: DeploymentID, InstanceID, ModelArtifactID, RequestedModel, ResolvedModel, BackendURL, Route |

#### Backend — modified files

| File | Action | Detail |
|---|---|---|
| `internal/server/api/router.go` | CHANGE | ADD routes: `POST /api/v1/api-keys` → `HandleCreateAPIKey`, `GET /api/v1/api-keys` → `HandleListAPIKeys`, `POST /api/v1/api-keys/{id}/disable` → `HandleDisableAPIKey`, `DELETE /api/v1/api-keys/{id}` → `HandleDeleteAPIKey`, `GET /api/v1/gateway/usage` → `HandleListGatewayUsage`; ADD external routes: `GET /v1/models` → `HandleGatewayListModels`, `POST /v1/chat/completions` → `HandleGatewayChatCompletions` (these go OUTSIDE `/api/v1` prefix, possibly at server root level) |
| `internal/server/auth/bootstrap.go` | CHANGE | ADD permissions: `api_key:read`, `api_key:write` to permission catalog |
| `internal/server/api/audit_writer.go` | MINOR | No change needed — gateway handlers call existing `WriteAudit()` |
| `docs/api/openapi.yaml` | CHANGE | ADD gateway routes, API key management schemas, usage query schema |

#### Frontend — new files

| File | Action | Detail |
|---|---|---|
| `web/src/pages/ApiKeysPage.vue` | CREATE | API key management: create key → show full key once with copy button → list with prefix/name/status/last_used → disable button → delete button |
| `web/src/pages/GatewayUsagePage.vue` | CREATE | Usage table: time, deployment, model, API key, route, HTTP status, latency, tokens; summary bar: requests, success, errors, total tokens, unknown tokens, avg latency |
| `web/src/api/apiKeys.ts` | CREATE | API client functions: `createApiKey`, `listApiKeys`, `disableApiKey`, `deleteApiKey` |
| `web/src/api/gatewayUsage.ts` | CREATE | API client functions: `listGatewayUsage` |

#### Frontend — modified files

| File | Action | Detail |
|---|---|---|
| `web/src/router/index.ts` | CHANGE | ADD routes: `/system/api-keys` → `ApiKeysPage`, `/observability/gateway-usage` → `GatewayUsagePage` |
| `web/src/layouts/ConsoleLayout.vue` | CHANGE | ADD menu items: "API Keys" under System group, "Gateway Usage" under Observability group |
| `web/src/locales/zh-CN.ts` | CHANGE | ADD i18n groups: `apiKeys.*`, `gatewayUsage.*` |
| `web/src/locales/en-US.ts` | CHANGE | ADD mirror i18n groups |

### Functions/components

**Go:**
- `GatewayAuthMiddleware(next http.Handler) http.Handler` — file: `internal/server/api/gateway_auth.go`
- `ResolveGatewayTarget(ctx, tenantID, requestedModel, scopes, db) (*GatewayTarget, error)` — file: `internal/server/gateway/model_resolver.go`
- `HandleGatewayListModels(w, r)` — file: `internal/server/api/gateway_handlers.go`
- `HandleGatewayChatCompletions(w, r)` — file: `internal/server/api/gateway_handlers.go`
- `HandleCreateAPIKey(w, r)` — file: `internal/server/api/api_key_handlers.go`
- `HandleListAPIKeys(w, r)` — file: `internal/server/api/api_key_handlers.go`
- `HandleDisableAPIKey(w, r)` — file: `internal/server/api/api_key_handlers.go`
- `HandleDeleteAPIKey(w, r)` — file: `internal/server/api/api_key_handlers.go`
- `HandleListGatewayUsage(w, r)` — file: `internal/server/api/gateway_usage_handlers.go`
- `hashAPIKey(key string) string` — file: `internal/server/api/api_key_handlers.go` (use bcrypt or sha256)
- `generateAPIKey() (full, prefix string)` — file: `internal/server/api/api_key_handlers.go`

**Vue:**
- `ApiKeysPage.vue`: create dialog (show full key once, copy button), table (prefix, name, status, last used), disable/delete actions
- `GatewayUsagePage.vue`: filter bar (time range, deployment, model, API key, success, route), usage table, summary bar

### Data/API contract changes

| Method | Path | Change | Permission | Handler |
|---|---|---|---|---|
| GET | `/v1/models` | **ADD** | Bearer API key | HandleGatewayListModels |
| POST | `/v1/chat/completions` | **ADD** | Bearer API key | HandleGatewayChatCompletions |
| POST | `/api/v1/api-keys` | **ADD** | api_key:write | HandleCreateAPIKey |
| GET | `/api/v1/api-keys` | **ADD** | api_key:read | HandleListAPIKeys |
| POST | `/api/v1/api-keys/{id}/disable` | **ADD** | api_key:write | HandleDisableAPIKey |
| DELETE | `/api/v1/api-keys/{id}` | **ADD** | api_key:write | HandleDeleteAPIKey |
| GET | `/api/v1/gateway/usage` | **ADD** | gateway_usage:read | HandleListGatewayUsage |

### DB schema changes (ADD only — clean DB, no migration)

**New table: `api_keys`**
```sql
CREATE TABLE api_keys (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  key_prefix TEXT NOT NULL,
  key_hash TEXT NOT NULL,
  scopes_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  last_used_at TEXT,
  expires_at TEXT,
  created_by TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(tenant_id, name)
);
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
```

**New table: `gateway_usage_records`**
```sql
CREATE TABLE gateway_usage_records (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  api_key_id TEXT NOT NULL DEFAULT '',
  deployment_id TEXT NOT NULL DEFAULT '',
  instance_id TEXT NOT NULL DEFAULT '',
  model_artifact_id TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  operation_id TEXT NOT NULL DEFAULT '',
  route TEXT NOT NULL DEFAULT '',
  requested_model TEXT NOT NULL DEFAULT '',
  resolved_model TEXT NOT NULL DEFAULT '',
  backend_url TEXT NOT NULL DEFAULT '',
  http_status INTEGER NOT NULL DEFAULT 0,
  success INTEGER NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  prompt_tokens INTEGER,
  completion_tokens INTEGER,
  total_tokens INTEGER,
  usage_source TEXT NOT NULL DEFAULT 'unknown',
  error_code TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_gateway_usage_tenant_created ON gateway_usage_records(tenant_id, created_at);
CREATE INDEX idx_gateway_usage_deployment_created ON gateway_usage_records(deployment_id, created_at);
CREATE INDEX idx_gateway_usage_key_created ON gateway_usage_records(api_key_id, created_at);
```

**New permission seeds** — in `internal/server/auth/bootstrap.go`:
```
api_key:read, api_key:write, gateway_usage:read
```

### Tests to add

**Go tests** — `internal/server/api/gateway_test.go` (CREATE):
```go
// Test cases:
// 1. missing_bearer_header_rejected
// 2. invalid_key_rejected
// 3. disabled_key_rejected
// 4. expired_key_rejected
// 5. cross_tenant_deployment_not_accessible
// 6. ambiguous_model_rejected (two deployments with same served_model_name)
// 7. no_running_instance_returns_503
// 8. successful_chat_request_proxied
// 9. backend_usage_parsed_and_recorded (prompt_tokens, completion_tokens in usage record)
// 10. missing_usage_recorded_as_unknown (usage_source='missing')
// 11. audit_record_written_on_success
// 12. audit_record_written_on_failure
// 13. full_key_never_returned_after_creation
// 14. create_key_returns_full_key_once
// 15. list_keys_shows_prefix_only
```

**Go tests** — `internal/server/api/api_key_test.go` (CREATE):
```go
// Test cases:
// 1. create_key_requires_tenant_admin
// 2. create_key_duplicate_name_rejected
// 3. disable_key_sets_status
// 4. delete_key_removes_row
// 5. disabled_key_auth_rejected
```

**Go tests** — `internal/server/gateway/model_resolver_test.go` (CREATE):
```go
// Test cases:
// 1. resolve_by_served_model_name
// 2. resolve_by_deployment_name
// 3. resolve_by_model_artifact_name
// 4. ambiguous_multiple_matches_rejected
// 5. scoped_to_deployment_ids_respected
// 6. stopped_deployment_not_resolved
```

**Frontend tests** — `web/tests/apiKeys.test.mjs` (CREATE):
```javascript
// Assertions:
// 1. create dialog shows full key with copy button
// 2. key list shows prefix only (not hash, not full key)
// 3. disable button sends PATCH
// 4. delete button sends DELETE
```

**Frontend tests** — `web/tests/gatewayUsage.test.mjs` (CREATE):
```javascript
// Assertions:
// 1. usage table renders rows
// 2. summary bar shows aggregate stats
// 3. filters send correct query params
```

### Validation commands
```bash
# DB rebuild (clean DB policy)
rm -f /tmp/lightai/data/lightai.db

# Backend tests
go test ./internal/server/api/... -run 'Gateway|APIKey|Usage|Audit|Key'
go test ./internal/server/gateway/...

# Frontend
cd web && npm test
cd web && npm run build

# API smoke testing
curl -X POST http://localhost:18080/api/v1/api-keys \
  -H 'Content-Type: application/json' \
  -H 'Cookie: ...' \
  -d '{"name":"test-key"}'
# Expect: {"id":"...","key":"lak-...","key_prefix":"lak-...","name":"test-key",...}

curl http://localhost:18080/v1/models \
  -H 'Authorization: Bearer lak-...'
# Expect: {"object":"list","data":[...]}

curl -X POST http://localhost:18080/v1/chat/completions \
  -H 'Authorization: Bearer lak-...' \
  -H 'Content-Type: application/json' \
  -d '{"model":"qwen3-demo","messages":[{"role":"user","content":"hi"}]}'
# Expect: OpenAI-compatible response with usage
```

---

## 7. Workstream E — Stability Regression

### Objective
Establish authoritative current regression baseline with Go tests, frontend tests, API E2E, browser smoke, and runtime smoke evidence — all timestamped and reproducible.

### Files to create/update

| File | Action | Detail |
|---|---|---|
| `docs/reports/product-hardening-20260626/evidence/<YYYYMMDDHHMMSS>/baseline/` | CREATE | Baseline test outputs before any change |
| `docs/reports/product-hardening-20260626/evidence/<YYYYMMDDHHMMSS>/api-e2e/` | CREATE | API-first E2E result (18 steps per Step E3) |
| `docs/reports/product-hardening-20260626/evidence/<YYYYMMDDHHMMSS>/browser-smoke/` | CREATE | Browser smoke evidence (if Playwright configured) or manual screenshots |
| `docs/reports/product-hardening-20260626/evidence/<YYYYMMDDHHMMSS>/runtime-smoke/` | CREATE | vLLM/SGLang/llama.cpp runtime smoke matrix |
| `docs/reports/product-hardening-20260626/execution/test-and-evidence-inventory.md` | CREATE | Script classification table (keep/repair/archive) per Step E2 |
| `docs/reports/product-hardening-20260626/execution/final-regression-report.md` | CREATE | Final closeout document per Step E7 |
| `scripts/archive/legacy-contract/` | LABEL | Add `README.md` in directory: "ARCHIVED — These scripts use deprecated API contract (backend_runtime_id, parameters_json, image_present). Do not use for current validation. See scripts/e2e-current-contract-*.sh for current equivalents." |
| `scripts/smoke-model-backends.sh` | ARCHIVE or REPAIR | Move to `scripts/archive/` with label: "Direct Docker smoke — bypasses product API. Use e2e-real-smoke-all-three.sh for product-level validation." |

### Functions/components: NONE
No code changes in Workstream E. Evidence collection only.

### Data/API contract changes: NONE

### Tests: NONE to add
All tests are run on the final code state after Workstreams A–D complete.

### Validation commands (mandatory, run after A–D changes)

```bash
# 1. Full Go test suite
go test ./... 2>&1 | tee docs/reports/product-hardening-20260626/evidence/<TS>/go-test.log

# 2. Go build
go build ./cmd/server/... && echo "SERVER OK" || echo "SERVER FAIL"
go build ./cmd/agent/... && echo "AGENT OK" || echo "AGENT FAIL"

# 3. Frontend tests
cd web && npm test 2>&1 | tee ../docs/reports/product-hardening-20260626/evidence/<TS>/frontend-test.log

# 4. Frontend build
cd web && npm run build 2>&1 | tee ../docs/reports/product-hardening-20260626/evidence/<TS>/frontend-build.log

# 5. Diff hygiene
git diff --check

# 6. API E2E (18-step chain per Step E3)
bash scripts/e2e-current-contract-api-dryrun.sh 2>&1 | tee docs/reports/product-hardening-20260626/evidence/<TS>/api-e2e.log

# 7. Runtime smoke (requires GPU hardware)
bash scripts/e2e-real-smoke-all-three.sh 2>&1 | tee docs/reports/product-hardening-20260626/evidence/<TS>/runtime-smoke.log

# 8. Gateway smoke (if D implemented)
# curl /v1/models, /v1/chat/completions with API key

# 9. Final status
git status --short
```

---

## 8. DB/Schema Impact Summary

| Workstream | DB Change | Type |
|---|---|---|
| A — Naming | None | — |
| B — Deployment UI | None | — |
| C — Runtime Parameters | None (ConfigSet JSON within existing columns) | — |
| D — Gateway | ADD `api_keys` table, ADD `gateway_usage_records` table, ADD 4 indexes | Clean fresh schema only |
| E — Regression | None | — |

**Breaking DB change: YES — Workstream D adds tables.** Clean DB rebuild required:
```bash
rm -f /tmp/lightai/data/lightai.db
# Restart server to recreate with new schema
```

No data migration needed (clean DB policy). No legacy compatibility required.

---

## 9. API Contract Impact Summary

| Method | Path | Workstream | Change |
|---|---|---|---|
| POST | `/api/v1/deployments/preview` | B | ADD |
| GET | `/v1/models` | D | ADD |
| POST | `/v1/chat/completions` | D | ADD |
| POST | `/api/v1/api-keys` | D | ADD |
| GET | `/api/v1/api-keys` | D | ADD |
| POST | `/api/v1/api-keys/{id}/disable` | D | ADD |
| DELETE | `/api/v1/api-keys/{id}` | D | ADD |
| GET | `/api/v1/gateway/usage` | D | ADD |
| PATCH | `/api/v1/backend-runtimes/{id}` | C | ENHANCE (accept parameter values) |
| PATCH | `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}` | C | ENHANCE (accept parameter values) |
| PATCH | `/api/v1/deployments/{id}` | C | ENHANCE (accept full config_overrides) |

No existing routes removed. No breaking changes to existing route contracts.

---

## 10. UI Contract Impact Summary

| Page/Component | Workstream | Change |
|---|---|---|
| `RuntimeTemplatesPage.vue` (renamed) | A | Name change, add parameter editor (C) |
| `NodeRuntimeConfigsPage.vue` (renamed) | A | Name change, add parameter editor (C) |
| `ModelDeploymentsPage.vue` | A, B, C | Naming fixes, wizard replacement, override editor |
| `ModelArtifactsPage.vue` | C | Remove runtime args from placeholder/label |
| `BackendsPage.vue` | A | "ConfigSet" → i18n label |
| `ModelInstancesPage.vue` | A | "runnerConfigs" → "common" i18n reference |
| `ConsoleLayout.vue` | A, D | Menu item name/route changes, add API Keys + Gateway Usage |
| `RuntimeParameterEditor.vue` | A, C | Title fix + major feature expansion |
| `DeploymentWizard.vue` | B | NEW |
| `ApiKeysPage.vue` | D | NEW |
| `GatewayUsagePage.vue` | D | NEW |

---

## 11. Commit Plan

Proposed commit sequence (one commit per workstream after validation):

```
Commit 1: "fix: apply naming dictionary (Workstream A)"
  - All naming changes
  - docs/engineering/naming-dictionary.md
  - Test updates

Commit 2: "feat: add deployment preview endpoint and wizard UI (Workstream B)"
  - POST /api/v1/deployments/preview
  - 6 new Vue components
  - Backend + frontend tests

Commit 3: "feat: integrate runtime parameter editor across layers (Workstream C)"
  - RuntimeParameterEditor enhancement
  - Editor integration in BackendRuntime/NBR/Deployment pages
  - Lint.go new rules
  - ModelArtifactsPage cleanup
  - Tests

Commit 4: "feat: add OpenAI-compatible gateway with API keys and usage (Workstream D)"
  - api_keys + gateway_usage_records tables
  - /v1/models, /v1/chat/completions handlers
  - API key CRUD + UI
  - Gateway usage query + UI
  - Auth middleware + model resolver
  - Tests

Commit 5: "chore: final regression evidence and closeout (Workstream E)"
  - Evidence directories
  - test-and-evidence-inventory.md
  - final-regression-report.md
  - Legacy script archival labels
```

---

## 12. File Count Summary

| Workstream | Files CREATE | Files MODIFY | Files RENAME | Total |
|---|---|---|---|---|
| A — Naming | 1 (naming-dictionary.md) + 1 (test) | 10 (pages, layout, router, locales, api, docs) | 2 (page renames) | 14 |
| B — Deployment UI | 7 (1 handler + 6 Vue components) | 4 (router, deployments.ts, preflight handler, openapi) | 0 | 11 |
| C — Runtime Parameters | 0 | 8 (components, pages, lint, catalog YAMLs, handlers) | 0 | 8 |
| D — Gateway | 11 (4 handlers + 2 models + 2 Vue pages + 2 API clients + 1 resolver) | 5 (db.go, router.go, bootstrap.go, layout, router, locales, openapi) | 0 | 16 |
| E — Regression | 5 (evidence dirs + inventory + report) | 0 | 0 | 5 |
| **Total** | **26** | **27** | **2** | **~55** |
