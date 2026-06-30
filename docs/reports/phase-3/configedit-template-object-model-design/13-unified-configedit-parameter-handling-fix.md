# Codex Task — Unified ConfigEdit Parameter Handling Regression Fix

You are working in:

`/home/kzeng/projects/ai-platform-study/lightai-go`

Use the current branch. Do not create a new branch.

## Purpose

Fix the regression discovered after the ConfigEdit Object Model / External ConfigEdit Component Template implementation.

The visible symptom is on the **运行模板 / BackendRuntime** page: after the recent change, the structured editor shows only a small number of fields, while many runtime-affecting parameters still exist in the underlying config data.

This must not be treated as a single-page or single-field bug.

This is a generic ConfigEdit contract regression:

> All parameters must be handled through one unified ConfigEdit Item/Component model. There must not be one class of parameters shown through structured ConfigEdit and another class that is only visible or editable through raw JSON, page-specific code, or resolver-specific hardcoding.

## Core principle

All configuration parameters are equal from the ConfigEdit engine perspective.

Every parameter that belongs to a ConfigSet / ConfigEdit Object and affects configuration, inheritance, validation, or runtime behavior must be materialized as a ConfigEdit Item or ConfigEdit Component.

Differences are allowed only in:

- section/group
- view level
- renderer
- editability
- readonly reason
- warning/risk level
- validation rule
- effect target
- copy/reset policy

Differences are not allowed in the underlying mechanism.

Do not implement two paths such as:

```text
normal parameters -> ConfigEdit UI
advanced/vendor/security parameters -> raw JSON only
```

That is explicitly forbidden.

Raw JSON is only a developer/debug representation of the same ConfigEdit Object. It is not a fallback configuration mechanism and must not be the only place where a real parameter appears.

## Definitions

### ConfigEdit Item

A simple editable field, for example:

- `model_runtime.gpu_memory_utilization`
- `model_runtime.max_model_len`
- `launcher.docker_options.shm_size`
- `launcher.docker_options.ipc_mode`
- `runtime.health.expected_status`

Each item should have:

- key
- label
- description/help
- value
- default value
- enabled state
- source/provenance
- editable flag
- readonly reason if readonly
- validation
- renderer
- view level
- effect
- copy-to-child policy
- reset-to-parent/default behavior

### ConfigEdit Component

A structured parameter group, for example:

- `runtime.device_binding`
- `service.port_binding`
- `runtime.model_mount`
- `runtime.health`
- `launcher.devices`
- `launcher.docker_options`
- `runtime.env`
- `backend.args`

Each component may contain multiple fields, but it follows the same rules as ConfigEdit Items.

### Raw JSON

Raw JSON is only a developer representation of the ConfigEdit Object. It can be used for diagnostics, debugging, and inspection.

Raw JSON must not be the only UI/API surface for a real parameter.

## Regression symptom

The current 运行模板 page appears to show only a small subset of fields, such as:

- Image ref
- Model mount
- Environment variables
- Container listen host
- Container listen port
- Health check

But the underlying configuration still contains many important parameters, for example:

- `model_runtime.tensor_parallel_size`
- `model_runtime.pipeline_parallel_size`
- `model_runtime.max_model_len`
- `model_runtime.gpu_memory_utilization`
- `model_runtime.max_num_seqs`
- `model_runtime.max_num_batched_tokens`
- `model_runtime.kv_cache_dtype`
- `model_runtime.dtype`
- `model_runtime.swap_space`
- `model_runtime.cpu_offload_gb`
- `model_runtime.enforce_eager`
- `model_runtime.safetensors_load_strategy`
- `model_runtime.trust_remote_code`
- `model_runtime.download_dir`
- `backend.extra_args`
- `launcher.docker_options.shm_size`
- `launcher.docker_options.ipc_mode`
- `launcher.docker_options.network_mode`
- `launcher.docker_options.group_add`
- `launcher.docker_options.ulimits`
- `launcher.docker_options.security_options`
- `launcher.docker_options.privileged`
- vendor devices such as `/dev/mxcd`, `/dev/dri`, `/dev/mem`
- vendor runtime env variables

This indicates that the ConfigEdit projection/template/view layer is not materializing all parameters as structured items/components.

## Main goal

Implement generic, unified parameter materialization.

Every non-hidden ConfigSet item must appear in structured ConfigEdit output as a ConfigEdit Item or Component, regardless of whether a component template explicitly maps it.

A component template may enhance a parameter:

- better grouping
- better renderer
- better label/help
- default/recommended/range
- specific validation
- effects
- risk warning

A component template must not make unmapped parameters disappear.

## Prohibited approaches

Do not fix by adding page-level special cases such as:

```text
if vendor == metax then show /dev/mxcd
if backend == vllm then show gpu_memory_utilization
if key == launcher.docker_options.shm_size then render ...
```

Do not add vLLM/SGLang/llama.cpp/NVIDIA/MetaX/Huawei parameter dictionaries inside Vue pages.

Do not make RunPlan resolver or Docker preview the only place where a runtime-affecting parameter is known.

Do not treat raw JSON as an acceptable editing surface for any real parameter.

## Required generic behavior

### 1. Unified materialization

The ConfigEdit projection layer must ensure:

```text
ConfigSet item
  -> ConfigEdit Item/Component
  -> section/group
  -> renderer
  -> validation
  -> view level
  -> effect
  -> copy/reset policy
```

This applies even when no explicit ConfigEdit component template exists for that key.

### 2. Fallback classification

When a parameter does not have an explicit component template mapping, classify it generically.

Suggested generic classification:

```text
model_runtime.*                       -> Model Runtime Parameters
backend.*                             -> Backend Parameters
backend.extra_args                    -> Backend Arguments
launcher.image                        -> Runtime Launch
launcher.entrypoint                   -> Runtime Launch
launcher.command                      -> Runtime Launch
launcher.docker_options.*             -> Container Options
launcher.docker_options.privileged    -> Security / High Risk
launcher.docker_options.security_*    -> Security / High Risk
launcher.devices                      -> Vendor Device Mounts
launcher.volumes                      -> Volume Mounts
runtime.env                           -> Environment
runtime.extra_env                     -> Environment
runtime.model_mount                   -> Model Mount
runtime.health                        -> Health Check
service.*                             -> Service
runtime.device_binding                -> Device Binding
unknown non-internal keys             -> Advanced Parameters
```

This classification must be implemented in the common ConfigEdit layer, not in pages.

### 3. View levels are display policy, not separate mechanisms

Supported view levels:

- normal
- advanced
- security/high-risk
- developer

A parameter may be normal, advanced, or security/high-risk. It still remains a ConfigEdit Item/Component.

Developer view is for metadata and diagnostics, not a separate parameter model.

### 4. Technical keys

Normal/advanced/security views must not show labels such as:

```text
技术键: model_runtime.pipeline_parallel_size
```

Human-friendly labels are required.

English is acceptable for technical runtime parameters.

Technical keys may appear only in developer view.

### 5. Built-in and readonly templates

Built-in runtime templates may be readonly. However, readonly must not hide fields.

For built-in templates:

- show complete effective fields
- mark readonly if appropriate
- explain "Built-in template. Clone to edit." or "Editable in downstream override."
- keep all parameters visible in their appropriate view level

### 6. `shm_size`

`shm_size` is a Runtime Container Option.

It must be a first-class ConfigEdit Item or part of a Docker Options component.

Expected behavior:

```text
BackendRuntime default
  -> copied to NodeBackendRuntime
  -> copied to Deployment
  -> editable or readonly based on layer policy
  -> compiled to Docker option: --shm-size <value>
```

Default values may vary by backend/vendor/image, for example:

- NVIDIA vLLM may use `8gb`
- MetaX vLLM may use `100gb`
- CPU/llama.cpp may use smaller or empty default

The value must not be available only through raw JSON.

### 7. MetaX / vendor-specific devices and security options

MetaX-specific requirements must be structured ConfigEdit components/items.

Examples:

```text
Vendor Device Mounts
- /dev/mxcd
- /dev/dri
- /dev/mem

Container Options
- ipc_mode
- network_mode
- shm_size
- group_add
- ulimits

Security / High Risk
- privileged
- seccomp=unconfined
- apparmor=unconfined
- /dev/mem

Environment
- MACA/MX runtime env
```

These may be advanced/security view and may be warning-marked, but they must not disappear.

### 8. All downstream layers must preserve the same model

The unified model must apply through the chain:

```text
BackendRuntime
  -> NodeBackendRuntime
  -> Deployment
  -> RunPlan / DockerSpec
```

If a parameter is in the BackendRuntime effective ConfigEdit snapshot, then the downstream layer must copy it unless explicit copy policy says otherwise.

Source/provenance records where the value came from. It must not change the mechanism or make the field hidden.

## Required investigation

Inspect:

- `internal/server/configedit/project.go`
- `internal/server/configedit/types.go`
- `internal/server/configedit/templates.go`
- `internal/server/configedit/apply.go`
- `internal/server/catalog/loader.go`
- `internal/server/catalog/types.go`
- `internal/server/runplan/*`
- `internal/server/api/config_edit_handlers.go`
- `configs/configedit-templates/builtin/*.yaml`
- `configs/backend-catalog/runtimes/**`
- `configs/backend-catalog/versions/**`
- `configs/config-registry/items.yaml`
- `web/src/components/config/ConfigEditView.vue`
- `web/src/components/config/ConfigField.vue`
- `web/src/utils/configEditView.ts`
- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/pages/RunnerConfigsPage.vue`
- `web/src/components/deployments/DeploymentWizard.vue`
- `web/src/components/deployments/DeploymentOverrideEditor.vue`
- `web/src/pages/ModelDeploymentsPage.vue`

Answer through code/tests:

1. Are ConfigSet items dropped when a component template exists?
2. Are unmapped parameters excluded instead of fallback materialized?
3. Are normal/advanced/security view filters hiding valid parameters?
4. Are built-in/readonly templates hiding fields?
5. Are runtime/backend parameters present in raw data but missing from ConfigEdit Items/Components?
6. Are vendor-specific Docker/device/security parameters treated differently from ordinary parameters?
7. Are pages still interpreting or hiding parameters outside the common ConfigEdit engine?

## Implementation requirements

### A. Fix ConfigEdit projection

Implement a generic materialization path:

```text
explicit component template mapping
  else semantic/config registry metadata
  else ConfigSet item metadata
  else generic fallback classifier
```

Every non-hidden item must become a ConfigEdit field/component.

### B. Fix view filtering

View filtering must not hide parameters unless explicitly marked:

- hidden
- developer-only
- secret/internal platform field

Advanced/security fields should remain accessible in advanced/security view.

### C. Fix templates where needed

Update built-in templates to improve grouping/renderer/effect metadata, but do not rely on templates being exhaustive.

Templates should improve the UI, not define the only visible fields.

### D. Fix frontend rendering if needed

Ensure ConfigEditView and ConfigField render fallback fields/components.

Frontend should not require a special template component for a field to appear.

### E. Keep raw JSON as developer representation only

Raw JSON can remain as developer view, but all real parameters must also appear as structured fields/components.

## Test requirements

Add backend tests:

1. Partial ConfigEdit template does not drop unmapped ConfigSet items.
2. Every non-hidden ConfigSet item is materialized as ConfigEdit field/component.
3. Built-in readonly runtime templates expose complete effective fields.
4. vLLM MetaX runtime exposes:
   - model runtime params
   - `shm_size`
   - ipc/network/group/ulimit/security options
   - MetaX device mounts
   - MetaX env
   - backend extra args
5. vLLM NVIDIA runtime exposes model runtime params, device binding, Docker options.
6. SGLang NVIDIA runtime exposes model runtime params, device binding, Docker options.
7. llama.cpp NVIDIA runtime exposes model/runtime args and Docker options.
8. `shm_size` maps to Docker effect and copies downstream.
9. High-risk fields are advanced/security, not hidden.
10. Raw JSON is not the only representation of any non-hidden runtime-affecting item.

Add frontend tests:

1. BackendRuntime page renders more than the minimal image/mount/env/port/health fields.
2. Model Runtime Parameters section appears.
3. Container Options section includes SHM size.
4. Vendor Device Mounts section includes MetaX devices when configured.
5. Security / High Risk section shows privileged/security options when configured.
6. Raw Config JSON is hidden in normal view and visible only in developer view.
7. Normal/advanced labels do not show technical key text.
8. Built-in readonly templates show fields readonly rather than hiding them.

## Manual/API evidence

In closeout, include evidence for:

- vLLM MetaX runtime ConfigEdit field/component list
- vLLM NVIDIA runtime ConfigEdit field/component list
- SGLang NVIDIA runtime ConfigEdit field/component list
- llama.cpp NVIDIA runtime ConfigEdit field/component list
- one downstream NBR copy check
- one downstream Deployment copy check
- one RunPlan compile check for `shm_size`
- one RunPlan compile check for MetaX devices/security/env if runnable as preview

## Verification commands

Run:

```bash
git status --short
go test ./...
cd web
npm test
npm run test:unit
npm run build
```

Fix failures and rerun.

## Closeout

Write:

```text
docs/reports/phase-3/configedit-template-object-model-design/07-unified-configedit-parameter-handling-closeout.md
```

Closeout must include:

- root cause
- what was wrong in the previous implementation
- how unified parameter materialization now works
- affected pages
- affected backend/vendor/runtime variants
- restored field categories
- evidence that raw JSON is developer representation only
- `shm_size` ownership and evidence
- MetaX devices/security/env evidence
- downstream copy evidence
- RunPlan compile evidence
- tests added/updated
- verification command results
- commit id
- push result
- final git status
- remaining limitations

Do not claim closure if any real runtime-affecting parameter exists only in raw JSON or page-specific code.

## Commit and push

Commit and push after verification:

```bash
git status --short
git add .
git commit -m "fix(configedit): unify runtime parameter materialization"
git push
```

Final output should include:

- status
- closeout path
- commit id
- tests
- push result
- git status
