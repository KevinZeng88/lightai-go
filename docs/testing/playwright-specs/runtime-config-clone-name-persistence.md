# Playwright Spec Design: Runtime Config Clone Name Persistence

## 1. Target Spec

Target Playwright test file:

```text
web/tests/e2e/runtime-configs/runtime-config-clone-name-persistence.spec.ts
```

Design document location:

```text
docs/testing/playwright-specs/runtime-config-clone-name-persistence.md
```

## 2. Purpose

This spec verifies that a cloned runtime configuration keeps the user-entered name after save, refresh, and API readback.

The test is based on the observed issue:

```text
When copying a user runtime config, the create page shows the new name, but after save the config name becomes the same as the source config.
```

## 3. Product Risk Covered

This test protects the clone and save chain:

```text
clone dialog/form
  -> display_name/name input
  -> save payload
  -> clone API or update API
  -> backend merge logic
  -> DB persistence
  -> list API
  -> detail API
  -> UI hydration after refresh
```

The expected model is:

```text
A clone is a new independent runtime configuration.
Its display name must come from user input.
The source config name must not overwrite the clone name after save.
```

## 4. Out of Scope

This spec does not verify:

- Parameter enabled persistence.
- Parameter value persistence.
- Runtime check.
- Deployment.
- Docker/image behavior.
- RunPlan rendering.

Those belong to separate specs.

## 5. Preconditions

Baseline tests must pass:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web

npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/app-load.spec.ts
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/fullstack-health.spec.ts
npm run test:e2e -- --project=chrome-local tests/e2e/auth/login.spec.ts
```

Use admin storage state:

```ts
import { test } from '@playwright/test';
import { adminStorageStatePath } from '../helpers/auth';

test.use({ storageState: adminStorageStatePath });
```

## 6. Test Data Strategy

Create a unique clone name:

```text
e2e-clone-name-<timestamp>
```

Recommended source config selection order:

1. vLLM NVIDIA Docker runtime config.
2. SGLang NVIDIA Docker runtime config.
3. llama.cpp NVIDIA Docker runtime config.
4. Any visible runtime config that supports clone.

The test should record:

```text
sourceConfigId
sourceDisplayName
sourceName
cloneDisplayName
cloneId
```

## 7. Required Selectors

Recommended `data-testid` selectors:

```text
runtime-configs-page
runtime-config-list
runtime-config-row
runtime-config-name
runtime-config-display-name
runtime-config-clone-button
runtime-config-clone-dialog
runtime-config-display-name-input
runtime-config-name-input
runtime-config-save-button
runtime-config-success-message
```

Rows should include config identifiers:

```html
<tr data-testid="runtime-config-row" data-config-id="...">
```

If the page has both `name` and `display_name`, the UI should clearly distinguish:

```text
name: stable generated internal name, usually not user-editable
display_name: user-facing editable label
```

## 8. Recommended Helpers

Create or extend:

```text
web/tests/e2e/helpers/api.ts
web/tests/e2e/helpers/runtime-configs.ts
web/tests/e2e/page-objects/runtime-configs.page.ts
```

API calls must go through browser context:

```ts
await page.evaluate(async () => {
  const response = await fetch('/api/v1/xxx', { credentials: 'include' });
  return await response.json();
});
```

Do not use direct backend requests that bypass Vite proxy and browser cookies.

## 9. Test Procedure

### Step 1: Open runtime configuration page

Given admin is logged in.

When navigating to the runtime configuration page.

Then the list is visible.

### Step 2: Select source config

When selecting a source config that supports clone.

Then record the source config's ID, name, and display name from API and UI.

### Step 3: Clone config

When clicking clone.

And entering `e2e-clone-name-<timestamp>` as display name.

And saving.

Then the operation succeeds.

### Step 4: Verify list persistence

When returning to or refreshing the list.

Then the list contains the clone display name.

And the list still contains the source display name.

### Step 5: Verify detail persistence

When opening the cloned config detail page.

Then the detail page displays the clone display name.

And it does not display the source name in the clone's editable display name field.

### Step 6: Verify API persistence

When reading the cloned config through API.

Then the clone API record has the new display name.

And the clone ID is different from the source ID.

And the source API record is unchanged.

### Step 7: Refresh and re-enter

When refreshing the browser.

And reopening the cloned config detail.

Then the UI still shows the new display name.

## 10. API Assertions

The test must verify equivalent of:

```json
{
  "id": "clone-id",
  "name": "generated-or-internal-name",
  "display_name": "e2e-clone-name-20260627-191500"
}
```

Required assertions:

```text
clone ID exists
clone ID != source ID
clone display_name == user-entered name
clone display_name != source display_name, unless user intentionally entered same name
source display_name unchanged
list UI shows clone display name
detail UI shows clone display name
refresh does not revert clone name
```

If the product stores a separate internal `name`, the test should not require it to equal the display name unless the product contract says so.

## 11. Expected Failure Pattern

The suspected failure mode is:

```text
Create/clone form shows the new name.
Save appears successful.
After save or refresh, the clone name becomes the source config name.
API returns source display_name or merged source snapshot value.
```

If reproduced, inspect in order:

```text
clone form state
clone request payload
save request payload
backend clone handler
backend name/display_name merge order
copy-on-create snapshot logic
list API query and DTO mapping
detail API query and DTO mapping
frontend hydration after save
```

## 12. Evidence on Failure

Playwright should retain:

```text
/tmp/lightai/e2e/playwright/results
/tmp/lightai/e2e/playwright/report
```

The test should log:

```text
sourceConfigId
sourceName
sourceDisplayName
cloneConfigId
expectedCloneDisplayName
actualCloneDisplayName
list API response
detail API response
current URL
```

## 13. Run Commands

Single spec:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web

npm run test:e2e -- --project=chrome-local tests/e2e/runtime-configs/runtime-config-clone-name-persistence.spec.ts
```

Batch 1:

```bash
npm run test:e2e -- --project=chrome-local tests/e2e/runtime-configs
```

## 14. Completion Criteria

This spec is complete when:

```text
It creates a clone from UI.
It verifies clone name through UI list, UI detail, and API.
It proves source config is unchanged.
It fails clearly if clone name is overwritten by source name.
It does not rely on manual observation.
It uses stable selectors or documented data-testid additions.
```
