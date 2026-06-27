# Playwright Spec Design: Runtime Template Parameter Display

## 1. Target Spec

Target Playwright test file:

```text
web/tests/e2e/runtime-configs/runtime-template-parameter-display.spec.ts
```

Design document location:

```text
docs/testing/playwright-specs/runtime-template-parameter-display.md
```

## 2. Purpose

This spec verifies that runtime template parameters are displayed with user-facing labels and that internal structural fields are not exposed as ordinary editable parameters.

The test is based on observed UI issues:

```text
Runtime template parameter labels still appear in English.
Some internal or structural fields are shown as user parameters.
Some fields appear duplicated, especially devices/device-related fields.
Several advanced fields appear enabled by default when they should not be.
```

Observed examples include:

```text
Model mount
Environment variables
Kind
Ports
Volumes
Devices
Extra env
Backend extra args
```

## 3. Product Risk Covered

This test protects the runtime configuration user experience:

```text
parameter schema
  -> semantic/runtime parameter view model
  -> i18n/display label mapping
  -> parameter editor rendering
  -> structural field separation
  -> default enabled state presentation
```

The expected model is:

```text
Runtime template UI should show meaningful user-facing labels.
Internal Docker/config fields should not be mixed into ordinary parameter editing.
Device binding should be shown once in the appropriate section.
```

## 4. Out of Scope

This spec does not verify:

- Saving parameter changes.
- Clone name persistence.
- Runtime check.
- Deployment.
- Docker command generation.
- Container start.

Those belong to other specs.

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

## 6. Runtime Templates to Check

Check at least one template from each supported backend family when available:

```text
vLLM
SGLang
llama.cpp
```

If not all are available in the current clean DB, the test should check every visible runtime template and report which backends were covered.

Recommended source records:

```text
runtime.vllm.*
runtime.sglang.*
runtime.llamacpp.*
```

## 7. Required Selectors

Recommended `data-testid` selectors:

```text
runtime-configs-page
runtime-template-list
runtime-template-row
runtime-template-name
runtime-template-detail
runtime-parameter-editor
runtime-parameter-section
runtime-parameter-row
runtime-parameter-label
runtime-parameter-enabled-switch
runtime-parameter-value-input
runtime-structured-config-section
runtime-device-binding-section
runtime-port-section
runtime-volume-section
runtime-env-section
```

Parameter rows should include parameter key:

```html
<div data-testid="runtime-parameter-row" data-parameter-key="backend.arg.max_model_len">
```

Structural sections should be separate from ordinary parameter rows:

```html
<section data-testid="runtime-structured-config-section">
```

## 8. Forbidden Ordinary Parameter Labels

The ordinary parameter editor must not show these labels as raw parameter rows:

```text
Model mount
Environment variables
Kind
Ports
Volumes
Devices
Extra env
Backend extra args
```

If these concepts are needed, they should appear in structured sections with localized labels and clear semantics.

## 9. Internal Field Rules

The following field classes must not be rendered as ordinary user parameters:

```text
kind
ports
volumes
devices
extra_env
backend_extra_args
model_mount
environment_variables
```

They should either be:

1. Hidden because they are implementation details.
2. Rendered through dedicated structured UI controls.
3. Rendered in read-only diagnostics where appropriate.

## 10. Duplicate Display Rules

The test should detect duplicate concepts in ordinary parameter UI.

Examples:

```text
Devices appears as both Devices and runtime.gpu.devices.
Device binding appears in both generic parameter editor and device section.
Ports appears as both Ports and a structured port editor.
Volumes appears as both Volumes and a structured volume editor.
```

Expected behavior:

```text
Each concept appears once in the correct section.
```

## 11. Default Enabled State Rules

Advanced structural fields must not appear enabled by default merely because defaults exist.

Fields to check:

```text
Model mount
Environment variables
Ports
Volumes
Devices
Extra env
Backend extra args
```

Expected behavior:

```text
Having a default value does not imply enabled=true.
Enabled state must reflect actual product semantics, not value presence.
```

## 12. Test Procedure

### Step 1: Open runtime templates page

Given admin is logged in.

When navigating to the runtime templates/configs page.

Then runtime templates are visible.

### Step 2: Open each target template

For each available target backend family:

```text
vLLM
SGLang
llama.cpp
```

When opening the template detail page.

Then the parameter editor and structured sections render successfully.

### Step 3: Check forbidden raw labels

When reading ordinary parameter row labels.

Then forbidden raw labels are absent from ordinary parameter rows.

### Step 4: Check localized/user-facing labels

When reading visible labels.

Then labels should be user-facing Chinese labels in zh-CN mode or known i18n labels.

Raw internal keys should not be the primary label.

Examples of raw internal keys that should not be primary labels:

```text
backend.arg.max_model_len
runtime.gpu.devices
docker.volumes
docker.ports
extra_env
```

### Step 5: Check duplicate concepts

When reading parameter rows and structured sections.

Then device, volume, port, and env concepts should not be duplicated across ordinary parameter editor and structured sections.

### Step 6: Check default enabled state

When checking advanced fields.

Then they should not all be enabled by default.

If a field is enabled by default, the test should log the field key, label, and source template for review.

## 13. Assertions

Required assertions:

```text
runtime template page loads
at least one runtime template is checked
ordinary parameter editor is visible
forbidden raw labels are not ordinary parameter labels
internal structural keys are not primary user labels
devices/device binding is not duplicated
ports are not duplicated
volumes are not duplicated
env/extra env are not duplicated
advanced structural fields are not all enabled by default
```

## 14. Evidence on Failure

Playwright should retain:

```text
/tmp/lightai/e2e/playwright/results
/tmp/lightai/e2e/playwright/report
```

The test should log:

```text
templateId
templateName
backend family
visible ordinary parameter labels
visible structured section labels
forbidden labels found
duplicate concepts found
default enabled advanced fields
current URL
```

## 15. Expected Failure Patterns

Expected current failure patterns may include:

```text
Forbidden English labels appear in the parameter editor.
Internal structural fields appear as normal parameters.
Devices appears more than once.
A field is enabled because it has a default value.
Parameter editor renders raw schema keys instead of display labels.
```

If reproduced, inspect in order:

```text
backend runtime parameter schema
semantic runtime parameter projection
runtimeParameterViewModel mapping
RuntimeParameterEditor rendering
page-level structured config sections
i18n keys
schema default -> enabled normalization
```

## 16. Run Commands

Single spec:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web

npm run test:e2e -- --project=chrome-local tests/e2e/runtime-configs/runtime-template-parameter-display.spec.ts
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

## 17. Completion Criteria

This spec is complete when:

```text
It checks at least one runtime template.
It detects forbidden raw/internal labels.
It detects duplicate structural concepts.
It validates default enabled presentation.
It produces enough evidence to fix UI mapping defects.
It does not rely on screenshot-only assertions.
It uses stable selectors or documented data-testid additions.
```
