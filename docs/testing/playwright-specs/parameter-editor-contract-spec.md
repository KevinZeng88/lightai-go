# Parameter Editor Contract Spec

This spec defines the reusable contract for LightAI Go configuration/parameter editing.

## 1. Contract Name

`ConfigEditParameterContract`

## 2. Contract Scope

Applies to UI surfaces that use:

```text
ConfigEditView -> ConfigField -> buildConfigEditPatch -> /api/v1/config-edit/apply
```

Known surfaces:

- BackendRuntime template/user config
- NodeBackendRuntime config
- NodeRuntimeConfigWizard
- DeploymentOverrideEditor
- BackendVersion edit view, where readonly rules permit

## 3. Core Invariants

### Invariant 1: enabled and value are independent

Changing `enabled` must not clear `value`.

Changing `value` must not force `enabled=true`, unless the field is required by schema.

### Invariant 2: defaults are not activation

`default`, `default_value`, or prefilled `value` may populate the UI value, but must not automatically enable an optional field.

### Invariant 3: required fields are forced enabled

If `required=true`, the UI should not expose a user toggle and the patch/apply path must force `enabled=true`.

### Invariant 4: save/reload is authoritative

After save, refresh, and reopen, UI state must match API state.

### Invariant 5: structured fields preserve subfield state

Structured parent objects such as `launcher.docker_options` must preserve per-subfield enabled state. The parent value alone is not enough.

### Invariant 6: layer rules are enforced

Fields hidden/protected at a layer cannot be patched at that layer.

Examples:

- Docker launcher fields are not deployment override fields.
- Backend model-serving args are editable at NBR/deployment layers according to current taxonomy, not blindly everywhere.
- Internal/debug fields should not appear in ordinary edit flow.

### Invariant 7: clone/snapshot isolation

Editing a clone or child snapshot must not mutate the source template/config.

## 4. UI Assertions

For a selected editable field:

```text
field row exists
field has stable data-field-key
field has stable data-internal-key
field label is user-facing, not raw internal key unless shown in advanced/raw diagnostics
field enabled control exists only when has_enable=true and required=false
field value control exists and remains populated when enabled=false
```

## 5. API Assertions

For each field under test:

```text
ConfigEditView field.original_enabled matches persisted state before edit
ConfigEditView field.original_value matches persisted state before edit
ConfigEditPatch includes enabled when enabled changes
ConfigEditPatch includes value when value changes
After apply, stored config_set_json contains expected enabled and value
After refetch view, field.enabled and field.value match stored config
```

For Docker subfields:

```text
launcher.docker_options.value[path] stores value
enabled metadata stores path-specific enabled state
projection reads path-specific enabled state
RunPlan emits only enabled path-specific fields
```

## 6. Contract Runner Cases

### Case A: enabled true round-trip

```text
Given editable field F with has_enable=true
When enabled is set to true
And saved
And page is refreshed/reopened
Then UI shows enabled=true
And API view shows enabled=true
And persisted config contains enabled=true
```

### Case B: enabled false round-trip

```text
Given editable field F with has_enable=true
When enabled is set to false
And saved
And page is refreshed/reopened
Then UI shows enabled=false
And API view shows enabled=false
And persisted config contains enabled=false
```

### Case C: disabled value preservation

```text
Given field F has value V
When enabled=false is saved
Then value remains V in UI and API
```

### Case D: value-only change

```text
Given field F has enabled=false and value V1
When value is changed to V2
And enabled remains false
Then API stores value=V2 and enabled=false
```

### Case E: clone isolation

```text
Given source config S and cloned config C
When field F is edited in C
Then S field F remains unchanged
```

### Case F: no raw labels in ordinary view

```text
Given ordinary runtime config view
Then labels such as launcher.docker_options, runtime.env, backend.arg.* raw key should not appear as ordinary field labels
```

Raw keys may be available in advanced diagnostics only if intentionally designed.

## 7. Failure Evidence

Every contract failure should print/log:

```text
surface name
object kind
layer
subject id
source id if applicable
field key
internal key
before UI state
before API state
patch payload if accessible
after UI state
after API state
page URL
trace path
```

## 8. Minimum Surfaces Required Before Closeout

Batch 1 closeout requires:

- BackendRuntime surface passing.
- NodeBackendRuntime surface passing or failing with documented product bug.
- Deployment override surface at least validates layer/protection behavior.

Batch 2 closeout requires:

- Full workflow coverage from NBR creation/check to deployment/preflight.
