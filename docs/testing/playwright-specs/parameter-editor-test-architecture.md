# Parameter Editor Test Architecture

Date: 2026-06-27
Subject: LightAI Go ConfigEdit / parameter editor testing strategy

## 1. Key Conclusion

Do not write one heavy Playwright test per page.

The actual reusable editing stack is:

```text
ConfigSet
  -> server configedit projector
  -> ConfigEditView
  -> ConfigField
  -> buildConfigEditPatch
  -> config-edit/apply
  -> updated ConfigSet
```

Therefore the test architecture should be:

```text
Unit/Go tests:     business rules, default/enabled semantics, projection/apply consistency
Vitest tests:      frontend patch building and component-level behavior
Playwright tests:  only browser/page integration and persistence smoke for each surface
```

## 2. Surfaces

The shared ConfigEdit editor is used in multiple UI surfaces.

| Surface | Primary file | Object kind | Layer | Test depth |
|---|---|---|---|---|
| Backend Version | `web/src/pages/BackendsPage.vue` | `backend_version` | `backend_version` | Thin integration |
| Backend Runtime | `web/src/pages/BackendRuntimesPage.vue` | `backend_runtime` | `backend_runtime` | First full Playwright surface |
| Node Backend Runtime | `web/src/pages/RunnerConfigsPage.vue` | `node_backend_runtime` | `node_backend_runtime` | Full persistence + status side effect |
| Node Runtime Wizard | `web/src/components/deployments/NodeRuntimeConfigWizard.vue` | uses config-edit view/patch | `node_backend_runtime` | Wizard-specific integration |
| Deployment Override | `web/src/components/deployments/DeploymentOverrideEditor.vue` | `deployment` | `deployment` | Override + RunPlan preview |
| Model Artifact | model page | not ordinary runtime editor | model facts/hints | separate negative assertions |

## 3. Testing Pyramid

## 3.1 Go tests: canonical contract

Go tests are the best place for:

- `enabled` does not default to true.
- `default_value` does not imply enabled.
- `value` and `enabled` are independent.
- Docker subfield enabled metadata round-trips.
- Hidden/internal fields do not leak into ordinary sections.
- Layer-specific validation rejects protected fields.
- Clone/snapshot boundaries are preserved.

Recommended new/extended tests:

```text
internal/server/configedit/configedit_test.go
  TestDockerSubfieldEnabledRoundTrip
  TestApplyEditPatchPersistsDockerSubfieldEnabledMetadata
  TestOptionalDefaultArgsRemainDisabled

internal/server/api/config_edit_handlers_test.go
  TestBackendRuntimeConfigEditApplyPersistsEnabledAndReloadsView
  TestNodeBackendRuntimeConfigEditApplyPersistsEnabledAndMarksNeedsCheck
  TestDeploymentOverrideConfigEditApplyPersistsEnabledToSnapshot

internal/server/catalog/catalog_seed_drift_test.go
  TestCatalogDefaultArgsDoNotAutoEnableOptionalParams
```

## 3.2 Frontend unit tests

Frontend unit tests are the best place for:

- `buildConfigEditPatch()` includes `enabled` when only enabled changes.
- It includes `value` when only value changes.
- It includes both when both change.
- It does not emit unchanged fields.
- Required fields force enabled true.
- Disabled fields keep value editable/present.

Recommended files:

```text
web/src/utils/__tests__/configEditView.test.ts
web/src/components/config/__tests__/ConfigField.contract.test.ts
```

## 3.3 Playwright tests

Playwright should be used sparingly for browser integration:

- Can page open the shared editor?
- Are selectors stable?
- Can a user toggle enabled and save?
- Does the saved value survive refresh/reopen?
- Does API state match UI state?
- Does source/clone isolation hold?

Do not use Playwright to exhaustively test every parameter and every type. That belongs in unit/API tests.

## 4. Reusable Playwright Design

Use a contract runner plus surface adapters.

```text
web/tests/e2e/contracts/config-edit.contract.ts
web/tests/e2e/surfaces/backend-runtime.surface.ts
web/tests/e2e/surfaces/node-backend-runtime.surface.ts
web/tests/e2e/surfaces/deployment-override.surface.ts
web/tests/e2e/helpers/api.ts
web/tests/e2e/helpers/config-edit.ts
```

Each surface adapter provides:

```ts
export type ConfigEditSurface = {
  name: string
  objectKind: string
  layer: string
  gotoList(page): Promise<void>
  createEditableSubject(page): Promise<ConfigEditSubject>
  openSubject(page, subject): Promise<void>
  save(page): Promise<void>
  reloadAndReopen(page, subject): Promise<void>
  getConfigEditView(page, subject): Promise<any>
  getSourceConfigEditView?(page, subject): Promise<any>
}
```

The contract runner performs shared assertions:

```text
1. Open surface.
2. Create/copy editable subject.
3. Find a field with `has_enable=true`.
4. Record original source and clone state.
5. Toggle enabled true.
6. Save.
7. Reopen.
8. Assert UI/API enabled true.
9. Toggle enabled false.
10. Save.
11. Reopen.
12. Assert UI/API enabled false.
13. Assert value was not cleared.
14. Assert source was not polluted where applicable.
```

## 5. Required Test IDs

The project should add test IDs to the shared editor once.

### ConfigEditView.vue

```html
<div
  data-testid="config-edit-view"
  :data-object-kind="localView.object_kind"
  :data-layer="localView.layer"
  :data-object-id="localView.object_id"
>
```

### ConfigSection.vue

```html
<section
  data-testid="config-edit-section"
  :data-section-key="section.key"
>
```

### ConfigField.vue

```html
<div
  data-testid="config-field"
  :data-field-key="field.key"
  :data-internal-key="field.internal_key"
  :data-section-key="field.section"
>
```

Enable checkbox:

```html
<el-checkbox
  data-testid="config-field-enabled"
  :data-field-key="field.key"
  ...
/>
```

Value input controls should expose:

```html
data-testid="config-field-value"
:data-field-key="field.key"
```

For controls where Element Plus wraps the input, place the test id on a stable wrapper and use `locator('input,textarea')` inside that wrapper.

## 6. API Helper Rule

All Playwright API validation should use browser-context fetch:

```ts
await page.evaluate(async ({ url }) => {
  const response = await fetch(url, { credentials: 'include' })
  return { status: response.status, body: await response.json() }
}, { url: '/api/v1/config-edit/view' })
```

For writes, obtain CSRF first:

```ts
const me = await fetch('/api/v1/auth/me', { credentials: 'include' }).then(r => r.json())
await fetch('/api/v1/config-edit/apply', {
  method: 'POST',
  credentials: 'include',
  headers: {
    'Content-Type': 'application/json',
    'X-CSRF-Token': me.csrf_token,
  },
  body: JSON.stringify(payload),
})
```

Do not call `http://127.0.0.1:18080` directly from Playwright business tests.

## 7. Recommended First Implementation Slice

Do not implement all surfaces at once. First slice:

```text
1. Add shared data-testid hooks to ConfigEdit components.
2. Add/extend Go test for Docker subfield enabled persistence.
3. Add frontend unit test for buildConfigEditPatch enabled-only change.
4. Add Playwright contract runner.
5. Add BackendRuntime surface adapter.
6. Add `config-edit-backend-runtime.spec.ts`.
```

Only after BackendRuntime is stable, add:

```text
7. NodeBackendRuntime surface adapter.
8. DeploymentOverride surface adapter.
9. ModelArtifact negative test for Docker params not shown on model page.
```

## 8. Avoided Anti-Patterns

Do not:

- Write a separate full enabled/value spec for every page.
- Use deep Element Plus CSS selectors.
- Use raw Chinese labels as the primary locator.
- Test all parameters through UI.
- Bypass the browser by direct backend API calls.
- Hide failing test data immediately through aggressive cleanup.
- Treat screenshot comparison as the main assertion.

## 9. Completion Definition

The parameter editor test architecture is acceptable when:

1. ConfigEdit component has stable test IDs.
2. Unit tests prove patch semantics.
3. Go tests prove configedit persistence semantics.
4. Playwright proves at least BackendRuntime, NBR, and Deployment surfaces can save enabled/value and reload consistently.
5. Future pages can be covered by adding a small surface adapter, not a new custom testing framework.
