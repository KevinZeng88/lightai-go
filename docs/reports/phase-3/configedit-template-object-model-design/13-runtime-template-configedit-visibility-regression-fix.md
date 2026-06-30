# Codex Task — Fix Runtime Template ConfigEdit Parameter Visibility Regression

You are working in:

`/home/kzeng/projects/ai-platform-study/lightai-go`

Use the current branch. Do not create a new branch.

## Context

A previous large change implemented the ConfigEdit object model, external ConfigEdit component templates, runtime effects, and a ConfigEdit Template Management MVP. The final status was `PASS_WITH_MVP_LIMITATIONS`.

However, user validation found a regression on the **运行模板 / BackendRuntime** page.

Observed page: `LightAI Go - 运行模板.mhtml`

The selected runtime was a MetaX/vLLM runtime. The normal edit UI only showed a few fields, such as:

- Image ref
- Model mount
- Environment variables
- Container listen host
- Container listen port
- Health check

But the raw config still contains many important runtime-affecting fields, including but not limited to:

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
- Docker/container options such as `ipc_mode`, `shm_size`, `network_mode`, `group_add`, `ulimits`, `security_options`, `privileged`
- vendor-specific device requirements such as MetaX `/dev/mxcd`, `/dev/dri`, `/dev/mem`
- vendor/runtime env such as MetaX MACA/MX env variables

This is not acceptable. The ConfigEdit/component template implementation must not hide ConfigSet items that existed before. Normal/advanced/developer view separation must not turn useful runtime parameters into raw JSON-only content.

## Root problem to fix

The regression class is:

> Existing runtime-affecting ConfigSet items are present in data but missing from structured ConfigEdit UI because ConfigEdit component templates or projection logic only render a small subset of fields. Raw JSON still contains the data, but normal/advanced ConfigEdit does not show it as editable/viewable fields.

This must be treated as a regression from the previous ConfigEdit object/template implementation.

## Main goal

Ensure all runtime-affecting parameters present in BackendRuntime / NodeBackendRuntime / Deployment snapshots remain visible and editable through ConfigEdit unless explicitly marked hidden or readonly by template/policy.

This must cover:

- BackendRuntime / 运行模板 page
- NodeBackendRuntime / 节点运行配置
- Deployment create/edit
- Deployment override
- RunPlan preview

This must cover runtime variants, not only NVIDIA:

- vLLM NVIDIA
- vLLM MetaX
- vLLM Huawei if present
- SGLang NVIDIA
- SGLang MetaX/Huawei if present
- llama.cpp NVIDIA
- llama.cpp CPU if present

## Design rules

### 1. Component template must enhance, not replace, ConfigSet visibility

If a ConfigEdit component template explicitly maps a field/component, use that structured component.

If a ConfigSet item exists but has no explicit component template mapping, it must not disappear.

Instead, place it into a fallback structured section such as:

- `Model Runtime Parameters`
- `Container Options`
- `Vendor Device Requirements`
- `Environment`
- `Advanced Runtime Parameters`
- `Ungrouped Config Items`

Only hide a field when the field/template explicitly says:

```yaml
hidden: true
view_level: developer
```

or the field is a platform internal/secret field that should not be user-visible.

### 2. Built-in readonly templates must still show complete fields

If a runtime/template is built-in or not directly editable, the UI should still show the complete effective fields as readonly, with a clear message such as:

- "Built-in template. Clone to edit."
- "Readonly at this layer. Editable in Node runtime config or Deployment override."

Do not reduce a readonly built-in template to only a few fields.

### 3. View level separation

Normal view should show common operator fields.

Advanced view should show technical but valid runtime fields.

Developer view should show raw/internal diagnostics.

Correct classification:

#### Normal / Advanced

Show as structured fields:

- Image
- Runtime kind/vendor
- Device binding
- GPU/accelerator IDs
- visible devices env
- model mount
- service host/container port/host port
- health check
- common backend runtime parameters
- `shm_size`
- `ipc_mode`
- extra args/env
- model runtime parameters such as max length, dtype, tensor parallel, GPU memory utilization

#### Advanced / Security

Show but warn:

- privileged
- security options
- `/dev/mem`
- host network
- raw device mounts
- ulimits
- group_add

#### Developer

Only developer/debug view should show:

- technical keys such as `model_runtime.pipeline_parallel_size`
- raw Config JSON
- raw source map
- patch target internals
- unresolved command template
- dry-run detail

### 4. `--shm-size` ownership

`shm_size` belongs to Runtime Container Options.

It should be:

- defaulted at BackendRuntime / runtime template layer, because it depends on backend/vendor/image requirements;
- copied to NodeBackendRuntime;
- copied to Deployment;
- editable/overridable at each layer unless policy marks readonly;
- compiled to Docker option `--shm-size <value>`.

For example:

- vLLM NVIDIA may default to `8gb`;
- vLLM MetaX may need a larger value such as `100gb`;
- llama.cpp CPU may not need a large value.

The field must not be raw JSON-only.

### 5. MetaX / vendor-specific devices

MetaX-specific requirements must be visible as structured fields/components, not hidden raw JSON.

At minimum, expose:

- device mounts such as `/dev/mxcd`, `/dev/dri`, `/dev/mem`
- `group_add`, for example `video`
- `ipc_mode`
- `network_mode`
- `privileged`
- `security_options`
- `ulimits`
- MetaX runtime env variables

Recommended grouping:

- `Device Binding`
  - accelerator IDs
  - binding mode
  - visible devices env

- `Vendor Device Mounts`
  - `/dev/mxcd`
  - `/dev/dri`
  - `/dev/mem`

- `Container Options`
  - ipc mode
  - shm size
  - network mode
  - ulimits
  - group add

- `Security / High Risk`
  - privileged
  - seccomp
  - apparmor
  - `/dev/mem`

Security/high-risk items may be collapsed and warning-marked, but must not vanish.

### 6. Page logic must not hardcode vendor/backend semantics

Do not fix this by adding page-level code like:

```text
if vendor == metax then show /dev/mxcd
if backend == vllm then show gpu_memory_utilization
```

The fix must be in the ConfigEdit object/template/projection layer.

Allowed places:

- ConfigEdit template metadata
- ConfigEdit projection
- renderer registry
- generic fallback classification
- template validation
- backend/catalog data

Not allowed:

- page-specific parameter dictionaries
- page-specific vLLM/SGLang/llama.cpp/MetaX/NVIDIA logic
- raw JSON as the only configuration surface

## Required investigation

Inspect these areas:

- `internal/server/configedit/project.go`
- `internal/server/configedit/types.go`
- `internal/server/configedit/templates.go`
- `internal/server/catalog/loader.go`
- `internal/server/catalog/types.go`
- `internal/server/runplan/*`
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

Find:

1. Are unmapped ConfigSet items dropped when component templates exist?
2. Does normal/advanced/developer view filtering hide too much?
3. Do built-in templates only exist for NVIDIA, causing MetaX/Huawei/CPU fallback to become too small?
4. Are runtime fields still present in raw JSON but missing from `sections/components/fields`?
5. Are Docker options split into too few subfields?
6. Are MetaX devices/env/security options classified as developer-only instead of advanced/security?
7. Are backend version args not merged into runtime template ConfigEdit object?

## Implementation requirements

### A. Add or fix fallback projection

Add a generic fallback so every non-hidden ConfigSet item appears in structured ConfigEdit output.

Fallback classification examples:

- `model_runtime.*` -> `Model Runtime Parameters`
- `backend.*` -> `Backend Parameters`
- `launcher.image`, `launcher.entrypoint`, `launcher.command` -> `Runtime Launch`
- `launcher.docker_options.*` -> `Container Options` or `Security / High Risk`
- `launcher.devices`, `launcher.volumes` -> `Device & Volume Mounts`
- `runtime.env`, `runtime.extra_env` -> `Environment`
- `runtime.model_mount` -> `Model Mount`
- `runtime.health` -> `Health Check`
- `service.*` -> `Service`
- unknown non-internal keys -> `Advanced Runtime Parameters`

Fallback labels should be human-friendly. Do not show `技术键: xxx` in normal/advanced view.

### B. Fill template coverage gaps

Extend or add ConfigEdit templates so runtime variants do not shrink.

At minimum, ensure meaningful coverage for:

- vLLM NVIDIA
- vLLM MetaX
- vLLM Huawei if current runtime exists
- SGLang NVIDIA
- SGLang MetaX/Huawei if current runtime exists
- llama.cpp NVIDIA
- llama.cpp CPU if current runtime exists

A template can share a common backend section plus vendor-specific container/device sections.

Do not duplicate massive templates unnecessarily. Prefer includes/shared fragments if supported; otherwise keep YAML clear and documented.

### C. Show complete BackendRuntime fields

On the 运行模板 / BackendRuntime page:

- Show complete effective ConfigEdit fields.
- If built-in readonly, show fields readonly rather than hiding them.
- If edit requires clone, show "Clone to edit" guidance.
- Keep raw Config JSON in developer view only.

### D. Preserve downstream copy behavior

Make sure fields visible at BackendRuntime flow through:

```text
BackendRuntime
  -> NodeBackendRuntime
  -> Deployment
  -> RunPlan
```

Fields must remain visible/editable downstream unless layer policy says otherwise.

### E. Add tests

Add backend tests:

1. Runtime template ConfigEdit projection does not drop existing ConfigSet items.
2. When component template is partial, unmapped items appear in fallback advanced sections.
3. Built-in readonly runtime templates still expose complete fields.
4. vLLM MetaX runtime exposes:
   - model runtime params
   - Docker/container options
   - MetaX device mounts
   - MetaX env
   - security options
   - shm_size
5. vLLM/SGLang/llama.cpp NVIDIA still expose model runtime params and device binding.
6. `shm_size` maps to Docker option effect and remains editable/copyable.
7. High-risk fields are classified advanced/security, not hidden.

Add frontend tests:

1. BackendRuntime page renders more than the minimal image/mount/env/port/health set for vLLM MetaX.
2. Model Runtime Parameters section is visible.
3. Container Options section includes SHM size.
4. Vendor Device Mounts or Device & Volume Mounts include MetaX devices.
5. Security / High Risk section can show privileged/security options.
6. Raw config JSON is hidden in normal view and visible only in developer view.
7. No normal/advanced label displays technical key text.

### F. Manual/API evidence

In closeout, include evidence for at least:

- vLLM MetaX runtime ConfigEdit view field list.
- vLLM NVIDIA runtime ConfigEdit view field list.
- SGLang NVIDIA runtime ConfigEdit view field list.
- llama.cpp NVIDIA runtime ConfigEdit view field list.
- BackendRuntime page screenshots or textual DOM evidence proving fields are visible.
- RunPlan preview evidence showing `shm_size`/device/options still compile.

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
docs/reports/phase-3/configedit-template-object-model-design/07-runtime-template-configedit-visibility-regression-closeout.md
```

Closeout must include:

- root cause
- affected pages
- affected backends/vendors
- field categories restored
- MetaX-specific evidence
- `--shm-size` ownership decision and evidence
- tests added/updated
- verification command results
- commit id
- push result
- final git status
- remaining limitations

Do not claim full closure if any runtime-affecting ConfigSet item remains raw JSON-only without explicit hidden/developer policy.

## Commit and push

Commit and push after verification:

```bash
git status --short
git add .
git commit -m "fix(configedit): restore runtime template parameter visibility"
git push
```

Final output should include:

- status
- closeout path
- commit id
- tests
- push result
- git status
