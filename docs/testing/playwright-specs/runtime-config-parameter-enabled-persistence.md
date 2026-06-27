# Playwright Spec Design: Runtime Config Parameter Enabled Persistence

## 1. Target Spec

Target Playwright test file:

```text
web/tests/e2e/runtime-configs/runtime-config-parameter-enabled-persistence.spec.ts
```

Design document location:

```text
docs/testing/playwright-specs/runtime-config-parameter-enabled-persistence.md
```

## 2. Purpose

This spec verifies that a cloned runtime configuration can persist a parameter's `enabled` state independently from its `value`.

The test is based on the observed issue:

```text
A cloned user runtime configuration can save parameter value changes, but changing whether a parameter is enabled does not persist. After clicking save, the enabled state reverts.
```

## 3. Product Risk Covered

This test protects the runtime parameter editor and persistence chain:

```text
RuntimeParameterEditor
  -> page form state
  -> save payload
  -> API DTO
  -> backend merge/update logic
  -> DB persistence
  -> readback API
  -> UI hydration after refresh
```

The expected model is:

```text
value and enabled are independent fields.
```

A parameter with a value may still be disabled. A disabled parameter may still keep its configured value for future use.

## 4. Out of Scope

This spec does not verify:

- Docker image availability.
- Runtime check result.
- Deployment creation.
- Container start.
- RunPlan command rendering.
- OpenAI-compatible inference.
- Parameter label translation.

Those belong to separate specs.

## 5. Preconditions

The following baseline tests must pass first:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web

npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/app-load.spec.ts
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/fullstack-health.spec.ts
npm run test:e2e -- --project=chrome-local tests/e2e/auth/login.spec.ts
```

The test must use the admin storage state:

```ts
import { test } from '@playwright/test';
import { adminStorageStatePath } from '../helpers/auth';

test.use({ storageState: adminStorageStatePath });
```

## 6. Test Data Strategy

The test should create a unique cloned runtime configuration:

```text
e2e-enabled-persistence-<timestamp>
```

Recommended source configuration selection order:

1. vLLM NVIDIA Docker runtime config.
2. SGLang NVIDIA Docker runtime config.
3. llama.cpp NVIDIA Docker runtime config.
4. Any runtime config with at least one parameter that supports `enabled`.

Recommended parameter candidates:

```text
backend.arg.gpu_memory_utilization
backend.arg.max_model_len
backend.arg.max_num_seqs
backend.arg.mem_fraction_static
backend.arg.ctx_size
backend.arg.n_gpu_layers
```

Selection rules:

1. Prefer a parameter that is currently `enabled=false`.
2. If every candidate is enabled, select one and test false -> true -> false.
3. If no parameter exposes an enabled switch, fail the test with UI and API evidence.

## 7. Required Selectors

Prefer stable `data-testid` selectors. If not present, add them before writing the final test.

Recommended selectors:

```text
runtime-configs-page
runtime-config-list
runtime-config-row
runtime-config-name
runtime-config-clone-button
runtime-config-edit-form
runtime-config-display-name-input
runtime-config-save-button
runtime-parameter-editor
runtime-parameter-row
runtime-parameter-label
runtime-parameter-enabled-switch
runtime-parameter-value-input
runtime-config-success-message
```

Parameter rows should expose the internal parameter key:

```html
<div data-testid="runtime-parameter-row" data-parameter-key="backend.arg.max_model_len">
```

Enabled switch:

```html
<button data-testid="runtime-parameter-enabled-switch" data-parameter-key="backend.arg.max_model_len">
```

Value input:

```html
<input data-testid="runtime-parameter-value-input" data-parameter-key="backend.arg.max_model_len" />
```

## 8. Recommended Helpers

Create or extend:

```text
web/tests/e2e/helpers/api.ts
web/tests/e2e/helpers/runtime-configs.ts
web/tests/e2e/page-objects/runtime-configs.page.ts
```

The API helper must call APIs through the browser context:

```ts
await page.evaluate(async () => {
  const response = await fetch('/api/v1/xxx', {
    credentials: 'include',
  });
  return await response.json();
});
```

Do not use direct backend requests such as:

```ts
request.get('http://127.0.0.1:18080/api/v1/xxx')
```

UI E2E must use the same origin and cookies as the real browser.

## 9. Test Procedure

### Step 1: Open runtime configuration page

Given the admin user is logged in.

When the test navigates to the runtime configuration page.

Then the page loads and the runtime config list is visible.

### Step 2: Clone a runtime config

When the test selects a source runtime config.

And clicks clone.

And enters a unique cloned config name.

And saves.

Then the cloned config appears in the list.

And the cloned config can be read from API.

### Step 3: Select a parameter

When the test opens the cloned config detail page.

Then it locates one parameter with an enabled switch.

The test records:

```text
sourceConfigId
clonedConfigId
parameterKey
beforeEnabled
beforeValue
```

### Step 4: Enable the parameter

When the test sets the parameter enabled state to true.

And saves.

And reloads the page.

And reopens the cloned config.

Then the UI still shows enabled=true.

And API still returns enabled=true.

And the parameter value is preserved.

### Step 5: Disable the parameter

When the test sets the same parameter enabled state to false.

And saves.

And reloads the page.

And reopens the cloned config.

Then the UI still shows enabled=false.

And API still returns enabled=false.

And the parameter value is still preserved.

### Step 6: Verify source config isolation

When the source config is read through API.

Then the source config's same parameter enabled/value state is unchanged from the baseline.

## 10. API Assertions

The actual API structure may vary, but the test must verify an equivalent of:

```json
{
  "parameter_values": {
    "backend.arg.max_model_len": {
      "enabled": true,
      "value": 8192
    }
  }
}
```

If the API response does not include an enabled field, this is a product/API gap and the test should fail with a clear error.

Required assertions:

```text
cloned config exists
cloned config name is correct
selected parameter exists
enabled=true persists in UI
enabled=true persists in API
enabled=false persists in UI
enabled=false persists in API
value is not cleared by enabled toggle
source config is not modified
```

## 11. Expected Failure Pattern

The currently suspected failure mode is:

```text
UI switch toggles successfully.
Save action reports success.
After refresh, the switch reverts to its old state.
API enabled field is unchanged or absent.
```

If reproduced, inspect in order:

```text
RuntimeParameterEditor emit payload
page-level save payload
API request body
backend DTO
backend merge/update logic
DB persistence
API readback hydration
schema default overlay logic
clone snapshot boundary
```

## 12. Evidence on Failure

Playwright should retain:

```text
/tmp/lightai/e2e/playwright/results
/tmp/lightai/e2e/playwright/report
```

The test should additionally log:

```text
sourceConfigId
clonedConfigId
parameterKey
beforeEnabled
beforeValue
afterEnabled
afterValue
API response body
current URL
```

## 13. Run Commands

Single spec:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web

npm run test:e2e -- --project=chrome-local tests/e2e/runtime-configs/runtime-config-parameter-enabled-persistence.spec.ts
```

Batch 1:

```bash
npm run test:e2e -- --project=chrome-local tests/e2e/runtime-configs
```

Baseline + Batch 1:

```bash
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/app-load.spec.ts
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/fullstack-health.spec.ts
npm run test:e2e -- --project=chrome-local tests/e2e/auth/login.spec.ts
npm run test:e2e -- --project=chrome-local tests/e2e/runtime-configs
```

## 14. Completion Criteria

This spec is complete when:

```text
It runs without manual interaction.
It can reproduce the enabled persistence bug before fix, or passes after fix.
It validates both UI and API.
It verifies source/clone isolation.
It preserves failure evidence.
It uses stable selectors or documented data-testid additions.
```
