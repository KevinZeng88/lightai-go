# ConfigEdit Object Model & Template-Driven Configuration Design

## 1. Core conclusion

The current issue is not a single missing field in the Model Deployment page. It is a class of configuration architecture problems.

The correct target model is:

```text
External ConfigEdit Component Template
        ↓ materialize
ConfigEdit Object / ConfigSet snapshot
        ↓ copy effective snapshot to child layer
Child ConfigEdit Object / ConfigSet snapshot
        ↓ compile
ResolvedRunPlan / DockerSpec
```

A page should not understand vLLM, SGLang, llama.cpp, NVIDIA, CUDA_VISIBLE_DEVICES, Docker `--gpus`, model mounts, backend command flags, or health check defaults. A page should render a ConfigEdit object and submit patches.

## 2. Distinguish two kinds of templates

There are two different template concepts and they must not be mixed.

### 2.1 Runtime/backend template

This describes how a backend runtime is launched.

Examples:

- vLLM NVIDIA Docker runtime
- SGLang NVIDIA Docker runtime
- llama.cpp CUDA server runtime
- MetaX Docker runtime
- CPU runtime

It may contain:

- image
- entrypoint
- command template
- default args
- env
- mount rules
- ports
- health check
- supported vendors

### 2.2 ConfigEdit component template

This is the higher-level template this design is about.

It describes how a configuration object is edited:

- sections
- components
- renderers
- field labels/help/tooltips
- value type and validation
- default/recommended/range
- enabled state
- editability by layer
- copy behavior to child ConfigEdit
- reset-to-parent/default behavior
- parent/child relationships
- final effects on CLI/env/Docker/mount/port/health check
- normal/advanced/developer views

In short:

```text
Runtime template = what runs and how it starts.
ConfigEdit component template = how a configuration object is shown, edited, inherited, copied, and compiled.
```

## 3. ConfigEdit should be treated as an object, not only as a form

A ConfigEdit object should be a self-contained editable configuration object.

It should include:

- object kind: model, backend_runtime, node_backend_runtime, deployment, deployment_override, template, etc.
- object id
- parent ConfigEdit snapshot reference or embedded parent summary
- child ConfigEdit initialization contract
- sections
- fields/components
- field values
- enabled/disabled state
- source/provenance
- override state
- validation rules
- render metadata
- compile effects
- patch targets
- reset behavior

It is not enough to return a flat list of `{key, value}` fields.

## 4. Parent → child chain

The configuration chain should behave like this:

```text
BackendRuntime ConfigEdit
  ↓ copy whole effective snapshot
NodeBackendRuntime ConfigEdit
  ↓ copy whole effective snapshot
Deployment ConfigEdit
  ↓ compile final snapshot
ResolvedRunPlan / DockerSpec
```

Important rules:

1. The child layer should copy the effective parent snapshot at creation time.
2. After copy, the child owns its own editable snapshot.
3. Parent changes should not silently rewrite child snapshots unless explicitly rebase/reset is requested.
4. A child field can show source/provenance, but source does not make it read-only.
5. `system_default`, `copied_from_parent`, `generated_initial_value`, and `node_inventory_default` describe initial origin only; they do not imply non-editability.
6. Each field should support reset to parent/default when applicable.

## 5. All final runtime-affecting parameters must be editable source config

Every parameter that affects final container/runtime behavior must come from a ConfigEdit item or an explicitly modeled config component.

This includes:

- Docker image
- entrypoint / command / args
- backend CLI flags
- extra args
- env and extra env
- Docker GPU option
- visible device env, e.g. `CUDA_VISIBLE_DEVICES`
- vendor-specific device mounts
- accelerator ids / count / binding mode
- model mount
- extra volumes
- ports
- host/container port
- IPC mode
- shm size
- health check
- served model name
- model path / container path
- trust remote code
- dtype
- tensor/pipeline parallelism
- memory utilization
- context length

RunPlan resolver should compile these from the final ConfigEdit snapshot. It should not silently inject hidden runtime-affecting parameters.

## 6. Derived/generated should mean initial origin only, not read-only

Do not use “derived result” to mean “not editable.”

For example:

```text
accelerator_ids = ["0"]
visible_env_key = CUDA_VISIBLE_DEVICES
visible_env_value = 0
docker_gpu_option = device=0
```

These may be initialized from node inventory and vendor rules, but after entering NodeBackendRuntime or Deployment they should be materialized into editable config fields.

Correct semantics:

- `source=system_default`: initial value came from system default.
- `source=copied_from_parent`: initial value came from parent snapshot.
- `source=node_inventory_default`: initial value came from node inventory.
- `source=generated_initial_value`: initial value came from template rule.
- `source=user_override`: user edited it in current layer.

None of those source labels alone should make the field read-only.

## 7. Device binding model

Device binding must be a first-class ConfigEdit component, not a hidden RunPlan injection.

Recommended component:

```text
runtime.device_binding
```

Fields:

- enabled: boolean
- mode: auto | manual | disabled | inherited
- vendor: nvidia | metax | huawei | cpu | none
- accelerator_ids: string[]
- accelerator_count: number
- visible_env_key: string
- visible_env_value: string
- docker_gpu_option: string
- device_mounts: string[]
- source
- overridden
- patch target
- validation

For NVIDIA, template defaults can initialize:

```text
docker_gpu_option = device={{ accelerator_ids | join(',') }}
visible_env_key = CUDA_VISIBLE_DEVICES
visible_env_value = {{ accelerator_ids | join(',') }}
```

Final Docker compile:

```text
runtime.device_binding.docker_gpu_option → docker run --gpus "device=0"
runtime.device_binding.visible_env_key/value → -e CUDA_VISIBLE_DEVICES=0
```

Users should be able to edit/disable these fields in BackendRuntime/NBR/Deployment according to template edit scope.

## 8. Normal / advanced / developer views

Current UI exposes too much raw/template/debug information to normal users.

A ConfigEdit template should define view levels:

### Normal

- image
- model location
- served model name
- device binding
- memory/context key parameters
- port
- health check summary

### Advanced

- extra args
- extra env
- extra volumes
- IPC/shm
- detailed health check
- backend-specific advanced runtime parameters

### Developer / Debug

- internal key
- raw template command
- source map raw JSON
- Config JSON
- dry-run detail
- unresolved raw placeholders
- raw DockerSpec

Technical keys such as `model_runtime.pipeline_parallel_size` should not be shown in the normal UI. They can exist in developer mode only.

## 9. Help/tooltip metadata

Field help should be useful to operators. English is acceptable and preferable for stable technical wording.

A field should be able to show:

- label
- short description
- default value
- recommended value
- range/min/max/step
- examples
- effect on CLI/env/Docker
- applicable backend/version/vendor
- risk/warning
- source/current layer
- edit scope

Example:

```text
GPU Memory Utilization
Description: Fraction of GPU memory reserved for model execution.
Default: 0.9
Recommended: 0.85 - 0.95
Range: 0.1 - 1.0
Effect: --gpu-memory-utilization
Applies to: vLLM
```

Avoid normal UI text like:

```text
技术键: model_runtime.pipeline_parallel_size
```

## 10. Externalization principle

The binary should provide generic infrastructure:

- ConfigEdit object model
- template parser/validator
- generic renderers
- generic validators
- snapshot copy/override/reset engine
- RunPlan compiler
- DockerSpec compiler
- device binding abstraction
- source/provenance engine
- API contracts
- RBAC/audit

The external template should define:

- backend/version-specific parameters
- labels/help/default/recommended/range
- renderers
- validations
- CLI/env/Docker/mount/port/health effects
- copy behavior
- editability by layer
- normal/advanced/developer grouping
- vendor-specific device binding defaults

This allows vLLM/SGLang/llama.cpp new versions to be supported by catalog/template updates without rebuilding the Go/Vue binary.

## 11. Template management UI

Eventually there should be a page to manage these ConfigEdit component templates.

Capabilities:

- list templates
- clone built-in template
- edit as structured form
- raw YAML/JSON advanced editor
- validate template
- preview generated ConfigEdit
- preview RunPlan/Docker command
- test with selected model/node
- publish/disable/rollback
- import/export
- diff versions

Built-in templates should be read-only; users should clone to create local/user-defined overrides.

Priority order:

```text
user-defined template > local override template > built-in template
```

## 12. What should change in the current implementation

The current implementation appears to have these risks:

1. Some fields are generated/inserted in RunPlan resolver rather than materialized into ConfigEdit snapshots.
2. Deployment pages can display final GPU injection but cannot edit the source config fields.
3. ConfigEditView is improving as a renderer, but ConfigEdit may not yet be a full object with parent/child snapshot semantics.
4. Source maps explain some final effects but do not replace editable fields.
5. Runtime template/config catalog likely does not yet fully define ConfigEdit components and child behavior.
6. Page-level or resolver-level special handling still exists for some runtime effects.
7. Raw template/developer fields are shown too prominently in normal UI.

The next step should be a read-only audit against this target design, followed by a staged implementation plan.
