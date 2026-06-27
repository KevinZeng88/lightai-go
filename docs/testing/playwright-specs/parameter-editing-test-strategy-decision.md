# Parameter Editing Test Strategy Decision

## 1. Purpose

This document freezes the testing direction for LightAI Go parameter editing so that implementation does not keep shifting between page-specific Playwright tests, component tests, API tests, and backend contract tests.

The strategy applies to parameter editing across runtime templates, runner configs, node backend runtimes, deployment overrides, and related ConfigEdit-based surfaces.

## 2. Decision Summary

Use a layered test strategy:

```text
Layer 1: Backend contract tests
  Verify config projection, apply/patch behavior, enabled/value persistence, default semantics, and snapshot isolation.

Layer 2: Frontend unit/Vitest tests
  Verify ConfigEdit view-model behavior, patch construction, enabled/value independence, and component-level rendering rules.

Layer 3: Thin Playwright surface tests
  Verify representative pages correctly connect ConfigEdit to real UI routes, real auth, real API calls, save/reload behavior, and visible user workflow.
```

Do not create a separate full Playwright test suite for every page that contains parameter editing. Page-level tests should be thin and representative.

## 3. Main Rationale

Parameter editing is a shared capability. The same rules should not be duplicated as heavy UI tests on every page.

The core rules should be tested once at the shared abstraction boundary:

```text
Backend:
  internal/server/configedit
  internal/server/semanticconfig
  internal/server/catalog
  internal/server/runplan

Frontend:
  web/src/components/config/ConfigEditView.vue
  web/src/components/config/ConfigField.vue
  web/src/utils/configEditView.ts
```

Page tests should verify integration points, such as:

```text
- The page loads the expected ConfigEdit view.
- The page renders editable fields using stable selectors.
- The page sends a save request.
- The page reloads fresh state after save.
- UI state matches API state after reload.
```

## 4. Stable Terminology

### ConfigEdit

The active shared editing abstraction for runtime/deployment-style configuration.

Expected code areas:

```text
web/src/components/config/ConfigEditView.vue
web/src/components/config/ConfigField.vue
web/src/utils/configEditView.ts
internal/server/configedit
```

### Surface

A page or workflow that hosts ConfigEdit, for example:

```text
BackendRuntime surface
RunnerConfig surface
NodeBackendRuntime surface
DeploymentOverride surface
```

### Contract

A capability-level rule that must hold regardless of page:

```text
enabled=true persists
enabled=false persists
value persists
disabled does not clear value
default value does not imply enabled
missing enabled does not imply enabled
clone/snapshot does not mutate source
```

## 5. Rules to Freeze

### Rule 1: enabled and value are independent

A parameter record must treat `enabled` and `value` as independent state.

Valid states include:

```json
{ "enabled": false, "value": "8192" }
{ "enabled": true,  "value": "8192" }
{ "enabled": false, "value": "" }
```

Disabling a parameter must not clear the value unless the user explicitly clears it.

### Rule 2: default value does not automatically enable a parameter

A default value is a hint or initial value. It does not mean the parameter should be passed into runtime args.

Required fields may be enabled by contract. Optional fields with defaults remain disabled unless explicitly enabled.

### Rule 3: missing enabled must not silently become true

When legacy or partial payloads omit `enabled`, the safe default is disabled unless schema marks the field required.

### Rule 4: projection must round-trip user state

If the user saves `enabled=true`, the next projected ConfigEdit view must return `enabled=true`.

If the user saves `enabled=false`, the next projected ConfigEdit view must return `enabled=false`.

Projection code must not overwrite saved user state with schema defaults.

### Rule 5: clone/snapshot isolation must be explicit

A cloned config can copy source values at creation time.

After creation:

```text
editing clone must not mutate source
editing source must not mutate clone
deployment snapshot must not be mutated by upstream runtime edits
```

### Rule 6: Playwright must use stable selectors

ConfigEdit shared components should expose shared `data-testid` attributes. Avoid test logic that depends on Element Plus internals, translated Chinese labels, or fragile CSS structure.

Recommended selectors:

```text
config-edit-view
config-edit-section
config-field
config-field-enabled
config-field-value
config-edit-save
```

Recommended data attributes:

```text
data-field-key
data-section-key
data-surface
data-object-id
```

## 6. What Playwright Should Cover

Playwright should cover a small number of representative flows.

Initial recommended surfaces:

```text
P0: BackendRuntime or RunnerConfig
P1: DeploymentOverride
P1: NodeBackendRuntime
```

A surface test should cover:

```text
1. Login using stored admin state.
2. Open the representative page.
3. Confirm ConfigEdit is rendered.
4. Change one representative parameter.
5. Save.
6. Reload or reopen the page.
7. Compare UI state with API state.
```

The surface test should not re-test every parameter rule already covered by backend/unit tests.

## 7. What Playwright Should Not Cover

Avoid these patterns:

```text
- One full enabled/value/default/clone suite per page.
- One UI test per parameter.
- Assertions based mainly on screenshot/manual visual review.
- Tests that rely on Chinese labels as primary selectors.
- Tests that bypass the browser context for authenticated UI flows.
- Repeating backend semantic tests through slow browser tests.
```

## 8. Existing Specific Specs

The existing specific specs remain useful as examples and acceptance references:

```text
runtime-config-parameter-enabled-persistence.md
runtime-config-clone-name-persistence.md
runtime-template-parameter-display.md
```

They should be interpreted under this decision document. Their behavior should be implemented through shared contracts/helpers where possible.

## 9. Review Gate

Before implementation, Codex should review this strategy against current code and answer:

```text
- Does the current code actually use ConfigEditView/ConfigField on the intended surfaces?
- Which pages reuse ConfigEdit?
- Which pages still have custom editing logic?
- Are the proposed backend/unit/Playwright layers aligned with current architecture?
- Which existing docs contradict this decision?
- What should be changed in docs before implementation?
```

Implementation should begin after review feedback is incorporated.
