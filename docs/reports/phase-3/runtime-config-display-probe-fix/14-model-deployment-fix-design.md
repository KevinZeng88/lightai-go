# Model Deployment Regression Fix Design

## Problems

User reported:

1. Deployment fails with:

```text
[resolve_error] unsupported runtime_type: (only docker is supported)
```

2. Existing deployment detail has no useful edit option and mainly shows:

```text
Raw config JSON
Source metadata JSON
```

3. If creating a deployment fails and user clicks Save only, the next New deployment opens at the last saved state until refresh.

4. If user cancels New deployment, the next New deployment resumes from the cancelled step.

5. Container listen port shows:

```text
container port: 8000
host port: empty
```

without clear semantic handling.

6. Deployment parameter page shows fields such as:

```text
model_runtime.cpu_offload_gb
model_runtime.kv_cache_dtype
model_runtime.max_num_batched_tokens
model_runtime.max_num_seqs
model_runtime.port
model_runtime.host
model_runtime.model
```

without proper product layering.

## Deployment wizard state design

### Required state lifecycle

`New deployment` must create a fresh draft each time.

Required events:

| Event | Required behavior |
| --- | --- |
| Click New | reset draft, set step=0/first step, open drawer |
| Save success | close drawer, reset draft, reload list |
| Save failure | keep drawer open for correction, but next New after close must reset |
| Cancel | close drawer, reset draft |
| Drawer close / X / overlay close | reset draft |
| Route leave / component unmount | reset draft |

Draft state includes at least:

```text
current step
selected model
selected runtime / node_backend_runtime_id
service config
config overrides
preflight result
runplan preview
error state
loading state
```

### Implementation expectations

In `web/src/pages/ModelDeploymentsPage.vue` or equivalent:

1. Create a function similar to:

```ts
function createEmptyDeploymentDraft(): DeploymentDraft
```

2. Create a single reset function:

```ts
function resetDeploymentWizard(): void
```

3. Call reset before opening New:

```ts
function openCreateDeployment(): void {
  resetDeploymentWizard()
  drawerMode.value = 'create'
  drawerVisible.value = true
}
```

4. Call reset on cancel/close/save success.

5. Do not rely on Vue reactive object mutation that preserves old nested references.

6. Do not keep failed preflight/runplan preview inside the next create session.

## Existing deployment detail/edit design

### List actions

Deployment list should expose useful actions, for example:

```text
View
Edit
Preview RunPlan
Start
Stop
Delete
```

Actual action availability should follow deployment/instance status.

### Detail view

Default detail should show structured sections:

```text
Basic information
Selected model
Selected runtime / NBR display name
Service endpoint / port semantics
Runtime parameters summary
Resource/device summary
RunPlan preview entry
Status / last error
```

Raw JSON should move to diagnostic sections:

```text
Raw config JSON — collapsed by default
Source metadata JSON — collapsed by default
Resolved RunPlan JSON — collapsed by default
```

### Edit behavior

Editing an existing deployment should use structured config edit UI. It should not require editing raw JSON.

Expected behavior:

1. Click Edit from deployment list or detail.
2. Open structured edit mode/drawer.
3. Show editable fields allowed for deployment-level overrides.
4. Save patches deployment override snapshot.
5. Cancel discards unsaved changes.

If certain deployment fields are immutable after creation, show them read-only with clear reason.

## `runtime_type` resolution design

### Required invariant

A Docker runtime deployment must resolve with:

```text
runtime_type = docker
```

This value must come from the selected NBR/runtime snapshot, not from user-editable deployment overrides.

### Where to inspect

Prioritize these code areas:

```text
web/src/pages/ModelDeploymentsPage.vue
web/src/api/deployments.ts or equivalent
internal/server/api/deployment_lifecycle_handlers.go
internal/server/api/preflight_handlers.go
internal/server/runplan* or internal/server/*runplan*
internal/server/configedit/project.go
internal/server/configedit/taxonomy.go
runtime catalog YAML seed files
```

Search terms:

```bash
grep -R "unsupported runtime_type" -n internal web
grep -R "runtime_type" -n internal web
grep -R "config_overrides" -n internal web
grep -R "source_node_backend_runtime" -n internal web
grep -R "RunPlan" -n internal/server
```

### Fix requirements

1. `runtime_type` must be copy-on-create from selected NBR/runtime template snapshot.
2. Deployment `config_overrides` must not override `runtime_type` to empty.
3. Old draft state must not carry empty `runtime_type` into new create.
4. Preflight/RunPlan preview must use the same resolver path as actual start.
5. If old DB data is bad, clean rebuild is allowed; still add code guard to avoid new bad snapshots.
6. A warning can be emitted for missing/empty runtime_type before defaulting only if the source runtime is unambiguously Docker.
7. Do not silently support non-Docker runtime types in this patch.

## Port field design

### Canonical field

User-facing canonical container listen port:

```text
service.container_port
```

`model_runtime.port` should not be a normal user-visible required deployment field.

### `model_runtime.port`

Expected behavior:

1. Hide it from ordinary deployment override forms.
2. If the backend CLI needs a `--port` argument, derive it from `service.container_port` during RunPlan construction.
3. Never show `required + empty + readonly`.

### Host port semantics

If `network_mode=host`:

```text
host port display: Not applicable / host network uses container port directly
port publishing: none
```

If bridge/default network:

```text
host port explicitly configured: show mapping host:container
host port empty with auto-publish supported: show Auto assign / Docker assigned after start
host port empty with no auto-publish: show warning, not silent blank
```

RunPlan preview must display the final interpretation.

## Parameter layering design

### Hide from normal deployment form

These should not be ordinary deployment override fields:

```text
model_runtime.model
model_runtime.host
model_runtime.port
model_runtime.download_dir
```

Rationale:

- model comes from selected model/artifact/location.
- host is usually internal binding such as `0.0.0.0`.
- port is derived from service config.
- download_dir is backend-specific operational detail.

### Common user-facing parameters

Keep only stable, high-value parameters in normal/advanced UI. Suggested vLLM user-visible set:

```text
gpu_memory_utilization
max_model_len
tensor_parallel_size
served_model_name, if useful
```

### Advanced collapsible parameters

Parameters like these are real vLLM tuning options, but should be behind an Advanced/Expert section:

```text
max_num_batched_tokens
max_num_seqs
cpu_offload_gb
kv_cache_dtype
swap_space, if supported by the selected backend version
safetensors_load_strategy
```

They should not be required by default.

### Custom args / Extra args

Backend-version-specific, low-frequency, or experimental CLI flags should be available through:

```text
Custom args / Extra args
```

Rules:

1. Freeform extra args must be clearly marked expert-only.
2. They should appear after structured parameters.
3. They should be excluded from normal forms unless enabled.
4. They should be included in RunPlan preview.

## Raw JSON policy

Default UI must not display raw JSON dumps as primary content.

Allowed diagnostic sections:

```text
Raw config JSON
Source metadata JSON
Resolved RunPlan JSON
Probe raw evidence
```

Rules:

1. Collapsed by default.
2. User must explicitly expand.
3. Raw JSON must not replace structured detail/edit.
4. Tests should assert raw JSON markers are not visible by default.
