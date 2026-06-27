# LightAI Go Code Review and Gap Analysis

Date: 2026-06-27
Scope: uploaded LightAI Go source archive, with emphasis on UI automation, parameter editing, runtime configuration, deployment readiness, security, and current product-goal fit.

## 1. Executive Summary

The project has already moved in the right direction: runtime configuration is no longer page-private editing logic; the current main editing path is a shared `ConfigEditView` / `ConfigField` frontend stack backed by server-side `configedit` projection and patch application. This is the right foundation for scalable UI testing.

The next testing architecture should therefore target the **ConfigEdit contract**, not individual pages one by one. Playwright should verify that each page correctly connects to the shared editor, while most enabled/value/default/snapshot rules should be covered by unit and API tests.

The most important findings are:

1. The effective shared editor is `ConfigEditView` / `ConfigField`, not the older `RuntimeParameterEditor`.
2. Several pages already reuse the shared editor: backend versions, backend runtimes, node backend runtimes, deployment overrides, and node runtime creation wizard.
3. Backend tests already encode many intended parameter rules, but the browser UI still lacks stable `data-testid` hooks.
4. Docker sub-field enabled state is likely not fully persistent because `projectDockerOptions()` always projects subfields as `enabled=false`.
5. Catalog loading still appears to enable backend arguments when defaults exist, which conflicts with the product rule that default values must not imply enabled state.
6. NBR save flow reloads the list but does not reselect/reload the edit view, which can make saved changes look reverted or stale.
7. Vite proxy Origin was correctly identified as a blocker for browser-driven POST/PATCH/DELETE; the current proxy-level fix is appropriate for local UI automation, but should be made configurable.
8. Current Phase-1 product goals are broadly covered at backend/API level, but UI regression, user-facing diagnostics, OpenAI-compatible audit/billing, and cross-page configuration consistency still require systematic work.

## 2. Current Parameter Editing Architecture

### 2.1 Shared frontend editor

The active editing stack is:

```text
web/src/components/config/ConfigEditView.vue
web/src/components/config/ConfigSection.vue
web/src/components/config/ConfigField.vue
web/src/utils/configEditView.ts
web/src/api/configEdit.ts
```

`ConfigEditView.vue` clones the server-projected edit view, emits a patch built by `buildConfigEditPatch()`, and delegates field rendering to `ConfigField.vue`.

`ConfigField.vue` displays an enable checkbox when `field.has_enable && !field.required`, binds it to `field.enabled`, and renders the field value separately. This is directionally correct because enabled and value are separate UI concepts.

`buildConfigEditPatch()` sends both `value` and `enabled` when either changed. This is also directionally correct.

### 2.2 Reused pages/surfaces

Found shared ConfigEdit usage in:

```text
web/src/pages/BackendsPage.vue
web/src/pages/BackendRuntimesPage.vue
web/src/pages/RunnerConfigsPage.vue
web/src/components/deployments/NodeRuntimeConfigWizard.vue
web/src/components/deployments/DeploymentOverrideEditor.vue
```

The older `RuntimeParameterEditor.vue` is explicitly marked as diagnostic/dev-only legacy, with a comment stating normal flows use `ConfigEditView` and semantic projection. `HumanRuntimeParameterForm.vue` and `runtimeParameterViewModel` appear to be secondary/legacy or currently less central.

### 2.3 Backend projection and apply path

The server edit path is:

```text
internal/server/api/config_edit_handlers.go
internal/server/configedit/project.go
internal/server/configedit/apply.go
internal/server/configedit/validate.go
internal/server/configedit/taxonomy.go
```

`HandleConfigEditView` reads `config_set_json`, projects it through `ProjectConfigSetToEditView`, and returns a UI-friendly edit view.

`HandleConfigEditApply` applies `ConfigEditPatch` through `ApplyEditPatchToConfigSet`, then writes back `config_set_json` to the relevant table:

```text
backend_versions
backend_runtimes
node_backend_runtimes
model_deployments
```

For `node_backend_runtimes`, apply also marks the row as `needs_check`, which is appropriate because runtime configuration changed.

## 3. Confirmed Strengths

### 3.1 Correct strategic abstraction

The project has moved to a generalized `ConfigSet -> ConfigEditView -> ConfigEditPatch -> ConfigSet` model. This is a much better fit than page-specific parameter editing because it supports:

- BackendVersion catalog defaults
- BackendRuntime templates/user configs
- NodeBackendRuntime snapshots
- Deployment overrides
- RunPlan projection
- Future multi-vendor runtime configuration

### 3.2 Unit/API tests already cover part of the desired contract

Existing tests in `internal/server/configedit/configedit_test.go` already check important rules:

- Internal keys should not appear as ordinary labels.
- Docker options are projected as structured fields.
- Defaults/values should not imply enabled state.
- Disabled values should be preserved separately from enabled.
- Hidden/internal existing items should be preserved.
- Deployment layer should reject protected fields.

Existing API tests in `internal/server/api/config_edit_handlers_test.go` cover:

- Config edit view projection through API.
- NodeBackendRuntime enable applies editable config patch.
- Deployment creation applies editable config patch to snapshot.

This is good; Playwright should complement these tests rather than duplicate them fully.

### 3.3 Playwright baseline is now usable

The manually verified baseline is now adequate for business UI testing:

```text
app-load.spec.ts              PASS
fullstack-health.spec.ts      PASS
auth/login.spec.ts            PASS
```

The key browser automation blockers have been solved:

- Correct Vite port: `15173`
- Local Chrome path: `/usr/bin/google-chrome-stable`
- Playwright evidence output: `/tmp/lightai/e2e/playwright/`
- Vite proxy Origin rewrite for backend CSRF/origin validation
- First-login password change and storageState reuse

## 4. High-Priority Issues

## P0-1. Docker sub-field enabled state is likely not persistent

### Evidence

In `internal/server/configedit/project.go`, `projectDockerOptions()` splits `launcher.docker_options` into subfields such as `docker.shm_size`, `docker.devices`, and `docker.group_add`. However, each projected subfield is forced to disabled:

```go
dockerItem["enabled"] = false
```

The comment says the parent object only stores values and therefore subfields default unchecked unless future schema carries per-field enabled bit.

### Why this matters

This directly matches the user-observed behavior: a checkbox can be turned on and saved, but after refresh the checkbox returns to off. Apply may store the value in the parent object, but projection loses the per-subfield enabled state.

### Recommended fix direction

Do not infer enabled from value. Instead, extend `launcher.docker_options` storage to preserve subfield enabled metadata explicitly. Possible clean schema inside the config item:

```json
{
  "code": "launcher.docker_options",
  "value": {
    "shm_size": "16gb",
    "devices": ["/dev/nvidia0"]
  },
  "enabled": true,
  "enabled_fields": {
    "shm_size": true,
    "devices": false,
    "group_add": false
  }
}
```

Then:

- `projectDockerOptions()` reads `enabled_fields[path]`.
- `ApplyEditPatchToConfigSet()` writes `enabled_fields[path]` for path-based fields.
- RunPlan resolver should only emit docker subfields when corresponding `enabled_fields[path] == true`, except required fields.

If `enabled_fields` is not the desired name, use an equivalent explicit structure. The key point is that enabled must be stored per subfield and not derived from value.

## P0-2. Default values may still incorrectly enable backend args during catalog load

### Evidence

In `internal/server/catalog/loader.go`, `addArgConfigItems()` sets:

```go
enabled := boolFromAny(arg["required"]) || defaultValue != nil
```

This conflicts with the intended rule already tested in `configedit_test.go`: defaults should prefill display/value but must not opt a parameter into RunPlan.

### Impact

This can explain why many parameters look enabled by default, including backend args that should be optional. It also creates inconsistent semantics between catalog materialization and config edit projection.

### Recommended fix direction

Use:

```go
enabled := boolFromAny(arg["required"])
```

If a catalog schema explicitly wants a default-enabled optional argument, it should carry a separate explicit field such as `enabled: true` or `default_enabled: true`, not infer it from `default`.

## P0-3. Semantic normalization defaults missing enabled to true

### Evidence

In `internal/server/semanticconfig/normalizer.go`, normalized items use:

```go
Enabled: boolFromAny(item["enabled"], true)
```

### Impact

Missing enabled becomes enabled. This is dangerous because configuration items may accidentally enter RunPlan or capability snapshots due to omitted metadata.

### Recommended fix direction

Change fallback to false except required fields. If required metadata is not available in this function, preserve false here and let projection/validation enforce required items elsewhere.

## P0-4. NodeBackendRuntime save may leave stale edit view selected

### Evidence

In `web/src/pages/RunnerConfigsPage.vue`, `saveNBREdit()` applies the patch, shows success, and calls `await load()`, but does not reselect the updated row or reload `nbrEditView`.

By contrast, `BackendRuntimesPage.vue` reloads and reselects the updated runtime after save.

### Impact

The user may see a stale drawer/edit state immediately after saving. This can be misread as “save reverted” even if the backend persisted the change.

### Recommended fix direction

Mirror `BackendRuntimesPage.vue`:

```ts
await load()
const updated = runtimes.value.find(r => r.id === selected.value?.id)
if (updated) selected.value = updated
```

or explicitly refetch `nbrEditView` for `selected.value.id` after save.

## P0-5. Browser UI lacks stable test selectors

### Evidence

The current Vue components do not appear to consistently expose `data-testid` or `data-parameter-key` attributes for ConfigEdit rows, enable switches, and inputs.

### Impact

Playwright specs will be fragile if they rely on English/Chinese labels, deep CSS, or Element Plus implementation details.

### Recommended fix direction

Add stable selectors to the shared components once, not page by page:

```html
<div data-testid="config-edit-view" :data-object-kind="view.object_kind" :data-layer="view.layer">
<div data-testid="config-edit-section" :data-section-key="section.key">
<div data-testid="config-field" :data-field-key="field.key" :data-internal-key="field.internal_key">
<el-checkbox data-testid="config-field-enabled" ... />
<el-input data-testid="config-field-value" ... />
```

This should be done in `ConfigEditView.vue`, `ConfigSection.vue`, and `ConfigField.vue`. After that, all surfaces become testable through the same contract runner.

## 5. Medium-Priority Issues

## P1-1. Legacy parameter editors remain in tree and may confuse future work

`RuntimeParameterEditor.vue` is marked diagnostic/dev-only legacy. `HumanRuntimeParameterForm.vue` and `runtimeParameterViewModel` appear less central. Keeping them is acceptable if they are still used for diagnostics, but they should be documented as non-primary or removed if truly unused.

Recommendation:

- Add a short architecture note in `docs/testing/playwright-specs/parameter-editor-surfaces-matrix.md`.
- Add a code comment in the legacy files.
- Avoid building new tests around legacy editor unless still used by a route.

## P1-2. Login error mapping hides security/origin errors

`LoginPage.vue` maps 401/403 to invalid credentials. During Playwright setup, `invalid origin` appeared to the user as username/password error. This slows diagnosis.

Recommendation:

- For 403 with `error != invalid credentials`, display the server error or a clearer generic security message.
- Keep user-friendly wording, but do not misclassify origin/CSRF failures as bad credentials.

## P1-3. Vite backend target is hardcoded

`web/vite.config.ts` uses:

```ts
const backendTarget = 'http://127.0.0.1:18080'
```

This works locally but should be configurable for different dev/test environments.

Recommendation:

```ts
const backendTarget = process.env.LIGHTAI_BACKEND_URL || 'http://127.0.0.1:18080'
```

Use the same value in documentation and Playwright setup.

## P1-4. ConfigEdit API helper for Playwright must handle CSRF

The frontend `apiClient` can refresh CSRF on 403 by calling `/auth/me` and retrying. A custom Playwright API helper using raw `fetch` must either:

- rely only on GET for validation, or
- fetch `/api/v1/auth/me`, extract `csrf_token`, and send `X-CSRF-Token` for POST/PATCH/DELETE.

Otherwise Playwright API cross-checks may fail even when UI is correct.

## 6. Gap Against Existing Product Goals

## 6.1 Covered reasonably well

Based on the code and docs, the following are substantially present:

- Node and GPU discovery/monitoring.
- BackendRuntime and NodeBackendRuntime separation.
- ConfigSet-based runtime configuration.
- Deployment snapshot concept.
- RunPlan resolver and preview direction.
- Tenant/RBAC base model.
- Docker/agent runtime integration.
- Basic Playwright smoke/auth baseline.

## 6.2 Partially covered / needs hardening

- UI parameter editor consistency across pages.
- Clone/snapshot isolation across BackendRuntime, NBR, Deployment.
- User-facing diagnostics for config/check/preflight errors.
- Browser automation coverage of actual product workflows.
- Stable selector/testability strategy.
- Config default/enabled semantics across catalog, projection, normalization, RunPlan.

## 6.3 Known future or currently out of scope

- API Key lifecycle, usage accounting, token stats, billing.
- Sophisticated multi-node scheduling, quotas, fair-share, priority scheduling.
- Full OpenAI-compatible gateway audit and metering.
- Multi-tenant operational UI completeness.

These can remain later phases, but should be documented so product maturity expectations are clear.

## 7. Recommended Priority Plan

## Batch A — ConfigEdit contract hardening

1. Add `data-testid` to shared ConfigEdit components.
2. Add/extend Go tests for Docker subfield `enabled_fields` persistence.
3. Fix Docker subfield enabled storage/projection.
4. Fix catalog default-enabled semantics.
5. Fix semantic normalizer missing-enabled fallback.
6. Fix RunnerConfigsPage save reload behavior.

## Batch B — Unit/API tests

1. `configedit` tests for enabled true/false round-trip.
2. API tests for backend_runtime apply + reload.
3. API tests for node_backend_runtime apply + reload + needs_check.
4. API tests for deployment override patch + RunPlan preview consistency.
5. Catalog seed tests ensuring optional defaults are not enabled.

## Batch C — Playwright surface contract

1. Implement reusable ConfigEdit Playwright contract runner.
2. Implement BackendRuntime surface adapter.
3. Implement NodeBackendRuntime surface adapter.
4. Implement DeploymentOverride surface adapter.
5. Keep model artifact page separate because it should not expose Docker runtime params.

## Batch D — Full workflow UI automation

1. Add/enable node runtime configuration.
2. Check runtime image/status.
3. Create model artifact/location.
4. Create deployment with override.
5. Preflight.
6. Start instance.
7. Logs.
8. Stop instance.

## 8. Acceptance Criteria

Before merging a parameter-editor fix:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go

go test ./internal/server/configedit/...
go test ./internal/server/api/...
cd web
npm test
npm run build
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/app-load.spec.ts
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/fullstack-health.spec.ts
npm run test:e2e -- --project=chrome-local tests/e2e/auth/login.spec.ts
npm run test:e2e -- --project=chrome-local tests/e2e/runtime-configs
```

For early Batch C, only the first surface may exist. The suite can initially run only `BackendRuntime` until NBR/Deployment adapters are implemented.
