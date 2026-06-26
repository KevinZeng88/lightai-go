# 02 — Current UX Problems and Root Causes

## 1. Summary

The current UI exposes implementation structures rather than product workflows. Users are being asked to operate on internal catalog/config objects instead of choosing a model, choosing a runtime environment, checking readiness, and deploying.

The root problem is not just missing labels. The issue is a mismatch between system object design and user-facing workflow design.

---

## 2. Runtime Templates page problems

### Observed problems

- Too many templates are shown.
- The page exposes backend/runtime/catalog records directly.
- The main table shows fields such as backend ID, backend version ID, vendor, image, and management state.
- Details expose `ConfigSet` and `Source Metadata` as primary information.
- The display does not match the user's desired mental model: `nvidia.sglang`, `nvidia.vllm`, `nvidia.llama.cpp b9700`.

### Root cause

The page lists raw `BackendRuntime` records instead of a user-facing runtime template view model.

### Required direction

Introduce a presentation model that groups or names runtime options as:

```text
<vendor>.<backend> [version]
```

Hide internal IDs and internal ConfigSet keys from the primary UI.

---

## 3. Node Runtime Config wizard problems

### Observed problem A: wizard state persists after cancel

When a user cancels after reaching later steps, reopening the wizard resumes from the old step.

#### Root cause

Wizard state is local component state and is not reset on dialog close/open.

#### Required direction

Every new create flow must start clean:

```text
activeStep = 0
selectedNode = null
selectedRuntime = null
form cleared
check result cleared
errors cleared
```

Use `destroy-on-close` and/or an exposed `reset()` method.

---

### Observed problem B: no node runtime config name

Users cannot enter a meaningful name for a NodeBackendRuntime.

#### Root cause

The wizard focuses on node/runtime selection but omits the user-facing identity of the resulting config.

#### Required direction

Add config name as a required user-visible field with auto-generated default:

```text
<node hostname> / <vendor> / <backend>
```

Example:

```text
KZ-LAPTOP / NVIDIA / SGLang
```

Save it as `display_name`.

---

### Observed problem C: internal ConfigSet keys shown as normal fields

Examples:

```text
launcher.command
launcher.args
{{MODEL_CONTAINER_PATH}}
launcher.*
runtime_env.*
```

#### Root cause

The generic `RuntimeParameterEditor` renders all `config_set.items` without a user-facing filtering/presentation layer.

#### Required direction

Do not render internal ConfigSet items directly in ordinary forms. Create a human-facing runtime parameter form that maps internal ConfigSet items into product fields.

---

### Observed problem D: save/check failure closes the wizard

The user reports the window flashes and closes, without adding the config and without showing a useful error.

#### Root cause

The flow treats enable/check as a single success path and emits `saved` too early. The parent closes the dialog when `saved` is emitted.

#### Required direction

Separate outcomes:

```text
enable failed       → stay on current step, show error
check-request failed → stay on check step, show error
enable succeeded but check pending/not-ready → stay on check step, show status and refresh action
ready/ready_with_warnings → allow Finish/Close
```

Do not close automatically on non-ready status.

---

## 4. Model Library wizard problems

### Observed problem

Model Library uses a node dropdown, while Node Runtime Configs use a node table.

### Root cause

Node selection is implemented per page rather than as a shared component.

### Required direction

Create a shared `NodeSelectorTable` and use it in both places. The business labels should differ:

```text
Model Library: select model file node
Node Runtime Config: select runtime node
```

---

## 5. Model Deployment wizard risks

### Current corrected behavior

The deployment wizard now allows only `ready` and `ready_with_warnings` NodeBackendRuntime selections, while still allowing all statuses to be visible when the user toggles show all.

### Remaining requirements

Keep these invariants:

- Non-deployable NBRs remain visible but not selectable.
- Preview uses `/deployments/preview`.
- Payload uses `node_backend_runtime_id` only.
- Preview/start resolver paths remain aligned.
- Errors do not close the dialog.

---

## 6. Cross-cutting root cause pattern

The same problem appears in multiple places:

```text
Internal representation leaks into user workflow.
```

Examples:

- ConfigSet keys shown as ordinary settings
- Template placeholders shown to users
- Raw IDs as primary labels
- Catalog records shown as user choices
- JSON diagnostics presented too prominently

Required correction:

```text
Create user-facing view models and reserve raw internals for advanced diagnostics.
```

