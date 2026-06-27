# Parameter Editor Test and Hardening Execution Plan

This plan is written for Codex or any code-generation/review agent. It assumes the Playwright baseline is already working.

## 1. Read First

Read these documents before editing code:

```text
docs/testing/playwright-ui-automation-baseline.md
docs/testing/playwright-auth-and-origin-runbook.md
docs/testing/playwright-next-ui-regression-plan.md
docs/testing/playwright-specs/parameter-editor-test-architecture.md
docs/testing/playwright-specs/parameter-editor-contract-spec.md
```

## 2. Constraints

- Do not create a new branch unless explicitly requested.
- Do not add compatibility fallback for old DBs.
- Do not bypass CSRF/Origin checks.
- Do not use direct backend URLs in Playwright business tests.
- Do not implement all surfaces in one patch.
- Prefer shared ConfigEdit test IDs and shared contract runner.

## 3. Batch A: Testability Hooks

### Objective

Make the shared ConfigEdit editor testable once, across all pages.

### Files to inspect/edit

```text
web/src/components/config/ConfigEditView.vue
web/src/components/config/ConfigSection.vue
web/src/components/config/ConfigField.vue
```

### Add selectors

Add stable test IDs:

```text
config-edit-view
config-edit-section
config-field
config-field-enabled
config-field-value
```

Add data attributes:

```text
data-object-kind
data-layer
data-object-id
data-section-key
data-field-key
data-internal-key
```

### Validation

```bash
cd web
npm test
npm run build
npm run test:e2e -- --project=chrome-local tests/e2e/auth/login.spec.ts
```

## 4. Batch B: Backend Contract Tests

### Objective

Prove the server configedit path correctly preserves enabled/value semantics.

### Files to edit

```text
internal/server/configedit/configedit_test.go
internal/server/api/config_edit_handlers_test.go
internal/server/catalog/catalog_seed_drift_test.go
```

### Required tests

1. Docker subfield enabled true persists through apply + reproject.
2. Docker subfield enabled false persists through apply + reproject.
3. Optional backend args with default values remain disabled unless explicitly required/enabled.
4. Missing `enabled` does not default to true in semantic normalization.
5. NBR apply marks status `needs_check` and reloads with expected enabled/value.

### Expected initial failures

The Docker subfield enabled round-trip may fail because `projectDockerOptions()` currently sets subfields to `enabled=false` every time.

### Validation

```bash
go test ./internal/server/configedit/...
go test ./internal/server/api/...
```

## 5. Batch C: Fix Contract Failures

### Expected fix areas

```text
internal/server/configedit/project.go
internal/server/configedit/apply.go
internal/server/catalog/loader.go
internal/server/semanticconfig/normalizer.go
web/src/pages/RunnerConfigsPage.vue
```

### Fix principles

- Store per-subfield enabled metadata for `launcher.docker_options`.
- Do not derive enabled from value/default.
- Required fields force enabled.
- Missing enabled should be false unless explicitly required.
- Saving NBR config should reload the edit view or reselect updated row.

## 6. Batch D: Frontend Unit Tests

### Objective

Prove frontend patch generation is correct.

### Files to add/edit

```text
web/src/utils/__tests__/configEditView.test.ts
web/src/components/config/__tests__/ConfigField.contract.test.ts
```

### Required assertions

- Enabled-only changes are emitted in patch.
- Value-only changes are emitted in patch.
- Disabled field values are not cleared.
- Required fields emit enabled true.
- Unchanged fields do not emit.

### Validation

```bash
cd web
npm test
npm run build
```

## 7. Batch E: First Playwright Contract Surface

### Objective

Use browser automation to verify BackendRuntime ConfigEdit integration.

### Files to add

```text
web/tests/e2e/helpers/api.ts
web/tests/e2e/helpers/config-edit.ts
web/tests/e2e/contracts/config-edit.contract.ts
web/tests/e2e/surfaces/backend-runtime.surface.ts
web/tests/e2e/runtime-configs/config-edit-backend-runtime.spec.ts
```

### Scope

Only BackendRuntime surface in this batch.

### Contract checks

- Create/clone editable backend runtime config.
- Open ConfigEditView.
- Select a field with `has_enable=true`.
- Toggle enabled true and save.
- Reload/reopen; UI/API show true.
- Toggle enabled false and save.
- Reload/reopen; UI/API show false.
- Value remains preserved.
- Source runtime remains unchanged where clone source is available.

### Validation

```bash
cd web
npm run test:e2e -- --project=chrome-local tests/e2e/runtime-configs/config-edit-backend-runtime.spec.ts
```

## 8. Batch F: Add More Surfaces

After BackendRuntime is stable:

```text
web/tests/e2e/surfaces/node-backend-runtime.surface.ts
web/tests/e2e/surfaces/deployment-override.surface.ts
```

Add thin specs:

```text
web/tests/e2e/runtime-configs/config-edit-node-backend-runtime.spec.ts
web/tests/e2e/deployments/config-edit-deployment-override.spec.ts
```

## 9. Final Regression

```bash
go test ./internal/server/...
go test ./internal/agent/...
cd web
npm test
npm run build
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/app-load.spec.ts
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/fullstack-health.spec.ts
npm run test:e2e -- --project=chrome-local tests/e2e/auth/login.spec.ts
npm run test:e2e -- --project=chrome-local tests/e2e/runtime-configs
```

## 10. Closeout Output

Report:

```text
modified files
test files added
which contract each test covers
known failures or fixed failures
trace/report paths for failures
all command results
git status --short
commit id
push result
```
