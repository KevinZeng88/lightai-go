# ConfigEdit Object Model and External Template Architecture Audit

> Date: 2026-06-30  
> Scope: read-only architecture audit and implementation plan  
> Code changes: none

## Executive Summary

Current LightAI Go is not yet the target ConfigEdit object model. It has a useful tiered `ConfigSet` and a `ConfigEditView` renderer, but the active API path is still primarily "project fields from `config_set_json` and patch values." It does not yet expose a full ConfigEdit object with parent snapshot reference, child initialization contract, reset-to-parent/default semantics, component-level effects, or normal/advanced/developer view contract.

The current chain does support important snapshot boundaries:

```text
BackendVersion -> BackendRuntime -> NodeBackendRuntime -> Deployment
```

BackendRuntime copies a version config set at creation, NodeBackendRuntime copies the BackendRuntime config set at enable time, and Deployment copies the NBR config set at create time. However, this is not yet "copy whole effective ConfigEdit object" because `placement_json`, `service_json`, GPU assignment, model location, probe/process-start evidence, and several Docker effects remain outside the ConfigEdit object.

The most important gap is device binding. Final Docker preview can contain:

```text
--gpus "device=0"
-e CUDA_VISIBLE_DEVICES=0
```

but these values are generated in `internal/server/runplan/resolver.go` from deployment placement, assigned GPUs, vendor defaults, and hardcoded resolver logic. They are not materialized as an editable `runtime.device_binding` ConfigEdit component, and source map records them as derived/system-generated.

The catalog externalizes some backend/version/runtime facts for vLLM, SGLang, and llama.cpp, but it is not an external ConfigEdit component template. Runtime templates define how to run. ConfigEdit component templates must separately define how to edit, inherit, validate, group, and compile fields and components. That second template layer does not exist yet.

## Current Architecture Map

### Persisted Objects

The database has independent `config_set_json` columns for:

- `backend_versions`
- `backend_runtimes`
- `node_backend_runtimes`
- `model_deployments`

Deployment additionally stores `placement_json`, `service_json`, and `config_overrides_json`. These are runtime-affecting but not fully folded into the ConfigEdit object.

### Catalog and Runtime Templates

Current catalog inputs:

- Backend versions: `configs/backend-catalog/versions/{vllm,sglang,llamacpp}/...`
- Runtime templates: `configs/backend-catalog/runtimes/{vllm,sglang,llamacpp}/...`
- Generic registry: `configs/config-registry/items.yaml`
- Help snippets: `configs/backend-catalog/help/...`

`internal/server/catalog/loader.go` materializes BackendVersion and BackendRuntime YAML into tiered ConfigSet items. This covers launcher image, entrypoint, command, Docker options, env, model mount, health, ports, and CLI args. It does not load a separate ConfigEdit component template with sections/components/effects/copy policy.

### ConfigEdit API

`POST /api/v1/config-edit/view` loads an object by kind and calls `configedit.ProjectConfigSetToEditView`. `POST /api/v1/config-edit/apply` patches the object's `config_set_json`.

The returned view contains sections and flat fields, not a full object graph. It includes useful field metadata such as source, inherited value, copy behavior, patch target, and constraints when present, but lacks first-class parent snapshot, child copy contract, reset actions, effect definitions, and nested component identity.

### RunPlan

Preview/start use the server RunPlan resolver, which is correct. The resolver reads the NBR config snapshot and Deployment inputs, builds args/env/Docker/mount/port/health/device binding, and then `EquivalentCommandPreview` renders the human-readable Docker command.

Key current resolver behavior:

- Args primarily come from NBR `launcher.command` and ConfigSet parameter values.
- Deployment service overrides `--port` and `--host` through `applyServiceArgs`.
- Env comes from NBR env, NBR env parameters, node override env, and deployment env overrides.
- Mounts are generated from `ModelLocation` and runtime model mount.
- Health check is resolved from runtime override or backend version with code defaults for status/timeouts.
- Device binding is generated in resolver from placement/GPU assignment/vendor defaults.

## Gap Analysis Against Target Design

| Target capability | Current status | Finding |
| --- | --- | --- |
| ConfigEdit as object | Partial | `ConfigEditView` is a rendered field list. `catalog.ConfigSetBundle` models more, but active API does not use it as the authoritative object chain. |
| Parent snapshot reference | Partial | Copy metadata exists in `source_metadata_json` and per-field provenance, but view response has no parent object/snapshot contract. |
| Child initialization/copy behavior | Partial | BR->NBR and NBR->Deployment copy snapshots exist, but not as ConfigEdit object contract and not for `placement_json`/`service_json`. |
| Copy whole effective snapshot | Partial | `config_set_json` is copied; runtime-affecting fields outside ConfigSet are not. |
| Field provenance/override/reset | Partial | Provenance fields exist; reset-to-parent/default is not exposed as an API/UI action. |
| Normal/advanced/developer views | Partial | `advanced_raw` exists and diagnostics are sometimes collapsed, but there is no template-level view policy. Some raw details still appear in ordinary preview/detail flows. |
| Nested components | Partial | Vue has generic widgets for env/mount/health/ports/devices, but no component template contract for `runtime.device_binding`, port binding, health check, etc. |
| Effects preview | Partial | Source map and preview explain effects after resolution. Effects are not defined by ConfigEdit component templates. |
| Compile final snapshot only | Not met | Resolver still injects runtime-affecting device/env/Docker behavior and service args outside final ConfigEdit snapshot. |
| External ConfigEdit component template | Missing | Existing YAML is runtime/backend catalog, not ConfigEdit component template. |

## Hidden and Hardcoded Parameter Inventory

### Should Move to ConfigEdit Component Template

| Parameter/effect | Current source | Why it should move |
| --- | --- | --- |
| NVIDIA Docker GPU option `--gpus "device=..."` | `buildDeviceBinding` and `EquivalentCommandPreview` | Runtime-affecting, user-visible, must be editable/disableable as `runtime.device_binding`. |
| `CUDA_VISIBLE_DEVICES` / `ASCEND_VISIBLE_DEVICES` env key/value | `defaultVisibleEnvKey`, resolver env injection | Vendor-specific defaults should be template metadata/effect, not resolver hardcoding. |
| Device binding mode/count/accelerator IDs | `placement_json`, assigned GPU lease, resolver | Needs materialized editable component copied into Deployment. |
| Port binding host/container/app port effect | `service_json`, `ApplySemanticSnapshot`, `applyServiceArgs`, `EquivalentCommandPreview` | Should be `service.port_binding` component with CLI/Docker effects. |
| Health check defaults/status/timeouts | `buildHealthCheck` code defaults | Defaults can be generic fallback, but intended values and editability should be template fields. |
| Model mount component | Resolver computes host/container mount from ModelLocation plus runtime mount | Generic compiler may compute safe host path, but editable mount policy/container path/read-only should be component fields/effects. |
| Extra env / extra volumes / extra args | Mixed ConfigSet fields and page-specific handling | Need normal/advanced/developer placement and effect definitions. |

### Should Move to Runtime/Backend Catalog

| Parameter | Current source | Note |
| --- | --- | --- |
| Backend software default args | BackendVersion YAML | Already mostly correct as runtime/backend template, not ConfigEdit component template. |
| Runtime image, Docker base options, vendor device mounts | Runtime YAML | Runtime template should keep default run facts; ConfigEdit template should define edit UI/effects. |
| Backend version parameter list | BackendVersion YAML | Needs richer metadata or linked ConfigEdit template for display/help/validation/effects. |
| Process start profiles | Go constants in `runplan/profiles.go` | Should move to catalog or ConfigEdit/runtime template once stable. |

### Acceptable Generic Engine Code

| Capability | Why it stays in binary |
| --- | --- |
| Template parser/validator and schema lint | Generic infrastructure. |
| Snapshot copy/override/reset engine | Generic object-model infrastructure. |
| DockerSpec compiler and command preview renderer | Generic effect executor, but effects should come from template/component values. |
| Path containment, mount safety, and tenant/RBAC | Security-sensitive platform rules. |
| GPU lease allocation and DB/device inventory lookup | Runtime state, not editable template data. |
| Validation execution framework | Generic; specific rules/ranges should be template data. |
| Source map generation | Explanation/audit infrastructure, not source of truth. |

## Device Binding Findings

Device binding is the clearest architecture mismatch.

Current generation path:

1. Deployment preview/create/start pass `placement_json`.
2. Preview/start select GPU IDs from placement or auto-pick available GPU.
3. Resolver builds `gpuIDs` from assigned GPUs.
4. `defaultVisibleEnvKey` returns `CUDA_VISIBLE_DEVICES` for NVIDIA/MetaX and `ASCEND_VISIBLE_DEVICES` for Huawei/Ascend.
5. Resolver writes env `CUDA_VISIBLE_DEVICES=...`.
6. `buildDeviceBinding` sets NVIDIA `DockerGPUOption = "device=" + value`.
7. `EquivalentCommandPreview` independently renders `--gpus "device=..."` from `plan.GPUDeviceIDs`.

Current limitations:

- There is no ConfigEdit field/component named `runtime.device_binding`.
- Final `--gpus` and visible-device env can appear without editable source fields.
- Source map marks device binding and Docker GPU option as `system_generated`/`derived` with patch target `deployment.placement_json`, not a ConfigEdit component.
- NVIDIA logic is in Go. MetaX reuses `CUDA_VISIBLE_DEVICES` by resolver default and relies on runtime YAML for device mounts; Huawei defaults `ASCEND_VISIBLE_DEVICES` but runtime is `template_only`.
- Deployment wizard does not expose GPU selection or device binding controls; auto selection can happen in preview/resolver.
- Disabling device binding is possible only by raw/API `placement_json.device_binding_enabled=false` or selection mode `disabled`; normal UI does not provide it as an editable component.

Target design:

```text
runtime.device_binding
  enabled
  mode: auto | manual | disabled | inherited
  vendor
  accelerator_ids
  accelerator_count
  docker_gpu_option
  visible_env_key
  visible_env_value
  device_mounts
  effects:
    docker.gpus
    env visible devices
    device mounts
```

The resolver should compile Docker/env effects from this component, not invent them.

## Other Final Docker Parameters That Can Still Be Hidden or Not Editable

| Final Docker/runtime field | Current status | Gap |
| --- | --- | --- |
| image | ConfigSet `launcher.image` plus NBR `image_ref` | NBR wizard uses separate DockerImagePicker and writes image outside ConfigEdit patch. |
| entrypoint | ConfigSet `launcher.entrypoint`, plus process profile | Process profiles are Go constants and not ConfigEdit components. |
| command/args | ConfigSet + resolver variable substitution + semantic adapter | Backend-specific semantic mapping remains hardcoded for vLLM/SGLang/llama.cpp. |
| extra args | `backend.extra_args` exists, but renderer/effect is generic/raw | Needs template-driven advanced component and validation. |
| env/extra env | ConfigSet `runtime.env` plus resolver env injection | Device env is injected outside ConfigEdit. |
| model mount | Runtime ConfigSet plus ModelLocation-derived host/container path | Final mount path can be generated from ModelLocation; editable component lacks full provenance/effect. |
| extra volumes | `launcher.volumes` exists in registry | Resolver currently builds only model mount in audited path; volume effect coverage is incomplete. |
| ports | Deployment `service_json` plus semantic snapshot plus preview renderer | Not a single ConfigEdit component; service editor bypasses ConfigEdit. |
| IPC/shm/security/ulimits/group_add | ConfigSet `launcher.docker_options` | Exposed via hardcoded `dockerFieldSpecs`, not template components. |
| health check | Runtime/version config plus code defaults | Defaults and fields not fully template-driven; UI widget fields do not match all server fields exactly. |
| served model name | `service_json` and semantic adapter | Deployment wizard service editor bypasses ConfigEdit. |
| model path/container path | Resolver variables from ModelLocation/model mount | Platform-owned, but should appear as readonly/generated ConfigEdit fields/effects. |
| vLLM/SGLang/llama.cpp params | Catalog + Go registry + semantic adapter | CLI mapping for common semantic fields still hardcoded in Go. |

## ConfigEdit Object Model Findings

Current `internal/server/configedit`:

- Projects a `config_set_json` into sections and fields.
- Splits Docker options into hardcoded subfields.
- Filters some internal/debug fields.
- Builds patches by comparing field values/enabled states.
- Applies patches by writing `value.effective_value/local_value`, enabled state, and source metadata.

Missing or incomplete:

- No top-level `parent` object in view response.
- No top-level child initialization contract.
- No authoritative `effective_snapshot_id` or `snapshot_version`.
- No reset API for parent/default.
- No object-level provenance/effect map.
- No component key separate from field key for nested components.
- No view-level `normal | advanced | developer` contract; `advanced_raw` is a section heuristic.
- No patch target routing beyond item/path mutation in one `config_set_json`.
- No ConfigEdit field for deployment placement/device binding.
- No ConfigEdit object that unifies `config_set_json`, `placement_json`, `service_json`, and model-location derived fields.

There is a more complete `catalog.ConfigSetBundle` model with inherited snapshots, own sets, local edits, and generated views. It is valuable, but the active ConfigEdit API and deployment/runplan path do not use it as the primary contract.

## External Template Findings

Existing external files are not enough for the intended architecture.

What exists:

- BackendVersion YAML defines software defaults and parameter schema.
- Runtime YAML defines default run facts such as image, Docker options, env, mounts, ports, health checks.
- Help YAML contains human text for some parameters.
- `configs/config-registry/items.yaml` defines generic registry items.
- User catalog override path exists for backend catalog files.

What is missing:

- No `kind: config_edit_template`.
- No component template precedence: user-defined > local override > built-in.
- No template parser/validator for sections/components/effects/copy/view levels.
- No component effects on CLI/env/Docker/mount/port/health check.
- No editability-by-layer in external template.
- No parent/child copy policy in external template.
- No built-in structured template editor or import/export/rollback UI.
- No coverage test that every final Docker effect maps to a ConfigEdit component/field.

## Cross-Backend Coverage

### vLLM

Externalized:

- Version YAML defines `--model`, `--host`, `--port`, `--served-model-name`, tensor/pipeline parallel, `--max-model-len`, `--gpu-memory-utilization`, dtype and other vLLM parameters.
- Runtime YAML defines NVIDIA Docker image and Docker options.

Hardcoded or incomplete:

- Semantic adapter maps `model_runtime.max_model_len` to `--max-model-len`, `gpu_memory_utilization` to `--gpu-memory-utilization`, and served model name to `--served-model-name` in Go.
- Device binding NVIDIA `--gpus`/`CUDA_VISIBLE_DEVICES` is Go resolver logic.
- ConfigEdit component metadata for range/recommended/effect is not externalized as a component template.

### SGLang

Externalized:

- Version YAML defines `--model-path`, `--host`, `--port`, `--tp`, `--mem-fraction-static`, `--context-length`, `--max-running-requests`, etc.
- NVIDIA, CPU, MetaX, Huawei runtime templates exist.

Hardcoded or incomplete:

- Semantic adapter maps max model length to `--context-length` and GPU memory utilization to `--mem-fraction-static` in Go.
- Process start profile for SGLang is Go constant.
- MetaX MacaRT runtime contains env/device expectations, but ConfigEdit has no vendor-neutral device binding component to own these effects.

### llama.cpp

Externalized:

- Version YAML defines `-m`, `--ctx-size`, `--n-gpu-layers`, `--threads`, batch/cache/split options.
- Runtime YAML defines NVIDIA Docker image and Docker options.

Hardcoded or incomplete:

- Semantic adapter maps max model length to `--ctx-size` in Go.
- Resolver has llama.cpp-specific `MODEL_CONTAINER_FILE` handling for GGUF file path.
- NVIDIA device binding remains resolver-injected.
- No ConfigEdit template can define that `--n-gpu-layers=-1` is recommended for GPU mode or distinguish GGUF file model path display/effect.

## Cross-Page Coverage and Bypass Inventory

| Page/component | Uses ConfigEdit | Bypasses ConfigEdit / hardcoded logic |
| --- | --- | --- |
| `BackendRuntimesPage.vue` | Renders `ConfigEditView` for runtime config. | Clone/rename/delete flows are separate; source summary and raw JSON diagnostics are page logic. |
| `RunnerConfigsPage.vue` | Renders `ConfigEditView` for NBR edit. | Image/status/probe summaries are separate; raw Config JSON visible in detail, not developer view gated by ConfigEdit. |
| `NodeRuntimeConfigWizard.vue` | Uses `ConfigEditView` for parameter patch when enabling NBR. | Node selection, runtime template selection, Docker image picker, save/check flow are outside ConfigEdit; image is separate field. |
| `DeploymentWizard.vue` | Uses `DeploymentOverrideEditor` with ConfigEdit patch. | Model selection, NBR selection, service ports/served name, node compatibility, preview call, and payload assembly are page logic. |
| `DeploymentServiceEditor.vue` | No. | Directly edits host port, container port, served model name into `service_json`. |
| `DeploymentOverrideEditor.vue` | Yes, but loads NBR object with deployment layer mode. | Does not create a Deployment ConfigEdit object with parent/child metadata before create. |
| `DeploymentPreviewPanel.vue` | No. | Displays Docker preview, device binding, source map, raw final run plan directly; shows patch target/source/effect details in normal preview. |
| `ModelDeploymentsPage.vue` | Edit drawer uses `ConfigEditView`. | Create wizard bypasses ConfigEdit for service/placement; dry-run command/device binding and raw JSON are page logic. |
| `BackendsPage.vue` | BackendVersion editor uses `ConfigEditView`. | Add-parameter mini form writes raw ConfigSet-like objects in Vue; backend detail initially shows raw config JSON. |
| `ModelArtifactsPage.vue` | No. | Model facts, capabilities, parameter defaults, locations, scan metadata are custom forms. Some are model-specific and may remain outside runtime ConfigEdit, but model parameter defaults should eventually become model-layer ConfigEdit. |

## UI/UX Findings

- Normal deployment preview exposes `source`, `patch target`, `system_generated`, and raw source-map fields. These belong in developer/debug view.
- `DeploymentPreviewPanel` and `ModelDeploymentsPage` show final RunPlan JSON/dry-run detail behind a collapse or detail section, but not through a ConfigEdit view-level policy.
- `RunnerConfigsPage` displays raw Config JSON and raw Source Metadata directly in detail without an explicit developer mode gate.
- Tooltips currently combine technical keys and direct help. They do not consistently show default, recommended range, min/max/step, effect, applicability, warnings, or edit scope.
- `ConfigField.vue` has useful structured widgets, but widgets are selected by Go/Vue hardcoded mapping (`dockerFieldSpecs`, `widgetOverrides`) rather than external component template.
- Technical keys are hidden in ordinary labels unless dev mode, which is good, but source-map tables still expose technical targets/keys in normal preview.

## What Should Be Externalized vs Stay in Binary

### Externalize

- Backend/version-specific CLI parameters and aliases.
- Labels/help/default/recommended/range/min/max/step/options.
- Component renderer selection.
- Editability by layer.
- Copy behavior to child layers.
- Normal/advanced/developer grouping.
- Effects on CLI/env/Docker/mount/port/health check.
- Vendor-specific device binding defaults.
- Process-start profiles when they become user-visible/runtime-specific.
- Template warnings and risk text.

### Keep in Binary

- RBAC, tenancy, audit, secret redaction.
- ConfigEdit template parser, validator, version migration.
- Generic snapshot copy/override/reset engine.
- Generic renderer registry and allowed component types.
- Generic expression evaluator with constrained functions.
- DockerSpec compiler and preview renderer.
- GPU lease allocation and node inventory lookup.
- Security validation for filesystem paths, mounts, env sensitivity, privileged flags.
- Source/effect map generation.
- Built-in template loading and precedence implementation.

## Proposed Staged Implementation Plan

### Stage 1 - Contract Audit and Failing Tests

Add tests before behavior changes:

- For vLLM/SGLang/llama.cpp, assert every final Docker command token maps to a ConfigEdit field/component/effect.
- Assert `--gpus` and visible-device env are not allowed as resolver-only system generated fields.
- Assert deployment preview source map references ConfigEdit component patch targets for image, args, env, mounts, ports, Docker options, health check, and device binding.
- Assert Deployment creation copies NBR effective ConfigEdit snapshot plus service and placement fields into a Deployment ConfigEdit object.
- Assert `service_json` and `placement_json` direct updates are either mirrored into ConfigEdit or rejected by the new contract.
- Add UI tests that normal view hides raw source map, dry-run detail, raw command templates, and internal keys.

Acceptance for Stage 1: tests fail on current implementation for the known gaps and pass for existing snapshot boundaries that are already correct.

### Stage 2 - ConfigEdit Object Model

Introduce a versioned ConfigEdit object response:

```json
{
  "object_kind": "deployment",
  "object_id": "...",
  "template_id": "...",
  "snapshot_id": "...",
  "parent": {
    "object_kind": "node_backend_runtime",
    "object_id": "...",
    "snapshot_id": "..."
  },
  "child_init": {
    "strategy": "copy_effective_snapshot"
  },
  "sections": [],
  "components": [],
  "effects_preview": [],
  "diagnostics": {}
}
```

Implementation notes:

- Reuse tiered ConfigSet items where possible.
- Promote `catalog.ConfigSetBundle` or equivalent into the active API path rather than maintaining unused parallel abstractions.
- Add reset-to-parent/default API operations.
- Add `view_level: normal | advanced | developer`.
- Store Deployment-level ConfigEdit fields for service and placement, not only `service_json`/`placement_json`.

Acceptance: BackendRuntime, NBR, Deployment, and Deployment override can all be opened as ConfigEdit objects with parent metadata and patch/reset semantics.

### Stage 3 - External ConfigEdit Component Template

Add template files, for example:

```text
configs/configedit-templates/builtin/vllm/nvidia-docker.yaml
configs/configedit-templates/builtin/sglang/nvidia-docker.yaml
configs/configedit-templates/builtin/llamacpp/nvidia-docker.yaml
data/configedit-templates.d/user/...
```

Template must define:

- sections/components/renderers
- fields/value schema/default/recommended/range
- validation
- editability by layer
- copy behavior
- view grouping
- effects on CLI/env/Docker/mount/port/health
- version/backend/vendor applicability

Acceptance: adding a new vLLM/SGLang/llama.cpp parameter can be done by template/catalog update plus reload, without Go/Vue changes, if renderer/effect type already exists.

### Stage 4 - Device Binding Component

Add `runtime.device_binding` component and materialize it into NBR/Deployment snapshots.

Required behavior:

- `enabled`, `mode`, `vendor`, `accelerator_ids`, `accelerator_count`, `visible_env_key`, `visible_env_value`, `docker_gpu_option`, `device_mounts`.
- NVIDIA defaults produce Docker GPU option and CUDA visible env from template effects.
- MetaX/Huawei/CPU use vendor-specific template defaults without NVIDIA page logic.
- Auto selection can be a generated initial value, but it must become editable Deployment ConfigEdit state.
- Disable removes Docker GPU option and visible-device env.

Acceptance: final `--gpus` and `CUDA_VISIBLE_DEVICES` cannot appear unless `runtime.device_binding` effect is present in the final materialized snapshot.

### Stage 5 - RunPlan Compiler Cleanup

Change RunPlan from resolver-injection to compiler-from-effects:

- Resolver receives a final materialized ConfigEdit snapshot.
- Generic compiler evaluates component effects.
- Source map explains effect origin; it is not a source of truth.
- `EquivalentCommandPreview` only renders `DockerSpec`; it must not create new semantics that are absent from DockerSpec/effects.
- Code defaults remain only as defensive fallback with diagnostics, not ordinary behavior.

Acceptance: no runtime-affecting `system_generated` source map entries except allowed platform-owned readonly fields such as container name, instance ID, and GPU lease IDs.

### Stage 6 - UI Cleanup

- Use ConfigEdit components for service ports, served model name, deployment placement/device binding, NBR image, runtime Docker options, env, health, mount, and backend params.
- Keep raw source map, Config JSON, dry-run details, unresolved templates, and raw DockerSpec in developer/debug mode.
- Improve tooltip/help display: default, recommended, range, examples, effect, applicability, source, edit scope.
- Replace page-level field knowledge with generic component renderers where possible.

Acceptance: normal deployment create/edit/preview exposes no raw source-map or raw JSON fields and still shows all user-editable runtime-affecting controls.

### Stage 7 - Template Management UI

Add a ConfigEdit Template page separate from the runtime template page.

Capabilities:

- List built-in/local/user templates.
- Clone built-in to user template.
- Structured editor for sections/components/fields/effects.
- Raw YAML/JSON advanced editor.
- Validate and lint.
- Preview ConfigEdit object for selected backend/version/vendor/layer.
- Preview RunPlan/Docker command for selected model/node.
- Publish/disable/rollback/import/export.

Acceptance: operators can add or modify a vLLM/SGLang/llama.cpp parameter template, validate it, preview its UI/effects, publish it, and use it without rebuilding binaries.

## Risk and Migration Considerations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Existing deployments have `service_json`/`placement_json` outside ConfigEdit | Migration may change preview/start behavior | Backfill Deployment ConfigEdit components from existing JSON and keep compatibility reads during transition. |
| Current source map tests expect system-generated device binding | Tests will fail after architecture change | Replace with assertions that device binding source is ConfigEdit component/effect. |
| User catalog/runtime templates are already external | Confusing "runtime template" with "ConfigEdit template" | Keep separate file roots, API names, and UI pages. |
| Renderer explosion | Too many backend-specific widgets | Keep generic renderers; backend specifics stay in template data. |
| Security-sensitive Docker options become editable | Privileged/device/mount mistakes | Add policy validator, RBAC gates, risk warnings, developer view gating, and audit logs. |
| Template expression safety | Arbitrary execution risk | Use a constrained expression language with no filesystem/network/process access. |
| MetaX/Huawei lack hardware validation | False readiness | Keep hardware validation blockers; template support does not imply runtime readiness. |

## Acceptance Standards

Implementation should pass:

```bash
go test ./...
cd web
npm test
npm run test:unit
npm run build
```

Targeted acceptance:

- vLLM, SGLang, and llama.cpp fixtures prove final Docker parameters map to ConfigEdit fields/components/effects.
- BackendRuntime, NodeBackendRuntime, Deployment, Deployment override, and RunPlan preview use the same ConfigEdit chain.
- Parent/child copy is snapshot-based and detached after create.
- Reset to parent/default works.
- Device binding supports auto/manual/disabled and vendor-neutral values.
- Final RunPlan compiles from final materialized snapshot only.
- Normal UI hides developer/debug data.
- ConfigEdit template validation catches invalid effects, missing renderers, duplicate keys, invalid layer editability, and unsafe Docker/mount choices.

## Open Questions

1. Should Deployment continue storing `service_json` and `placement_json` as compatibility mirrors, or should they become derived columns from Deployment ConfigEdit?
2. Should `catalog.ConfigSetBundle` be promoted as the active implementation, or should a new `configedit.Object` wrap existing ConfigSet with less migration risk?
3. What RBAC permission is required to edit high-risk Docker effects such as privileged, host network, device mounts, and security options?
4. Should ConfigEdit templates live under the backend catalog reload path or a separate reload endpoint?
5. How should UI expose template view levels: per-user preference, per-page mode, or explicit developer toggle?
6. What fields are allowed as platform-generated readonly effects: container name, instance ID, lease IDs, and absolute host model path likely should remain binary-owned.

## Formal Closeout Status for This Audit

This pass intentionally did not fix implementation gaps. The gaps above are audit findings and are captured in this report as the formal implementation plan.

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| --- | --- | --- | --- | --- | --- | --- | --- |
| CE-001 | ConfigEdit is not yet a full object model | `configedit.ProjectConfigSetToEditView` returns field sections from one `config_set_json`; no parent/child/reset/effects contract | Pages cannot consistently edit/copy/compile all runtime-affecting settings | DOCUMENTED_BLOCKER | `internal/server/configedit`, `internal/server/api/config_edit_handlers.go`, DB/API contracts | Stage 2 tests and API snapshots | Track in this report |
| CE-002 | Device binding is generated by resolver | `runplan/resolver.go` derives visible env and Docker GPU option; source map records system-generated values | Final Docker command can contain non-ConfigEdit editable effects | DOCUMENTED_BLOCKER | `internal/server/runplan`, ConfigEdit templates, deployment object model | Stage 4 tests for auto/manual/disabled | Track in this report |
| CE-003 | Backend-specific semantic CLI mapping remains in Go | `runplan/semantic_adapter.go` maps vLLM/SGLang/llama.cpp semantic fields to flags | New backend parameter/version can require binary changes | DOCUMENTED_BLOCKER | ConfigEdit template effects and semantic registry | Stage 3 cross-backend fixtures | Track in this report |
| CE-004 | Deployment service/placement bypass ConfigEdit | `DeploymentServiceEditor.vue`, `DeploymentWizard.vue`, preview handler use `service_json`/`placement_json` directly | Deployment create/edit is not one ConfigEdit chain | DOCUMENTED_BLOCKER | Deployment ConfigEdit object and UI | Stage 2/6 UI/API tests | Track in this report |
| CE-005 | Runtime and ConfigEdit templates are not separated | Runtime/backend catalog exists; no `config_edit_template` parser/storage/UI | Operator cannot externalize edit/display/validation/effects fully | DOCUMENTED_BLOCKER | New configedit-template loader/API/UI | Stage 3/7 validation | Track in this report |

Unresolved problems remain: yes, by design for this audit-only pass.  
All unresolved problems are documented in the table above.  
Problems existing only in chat: none.  
Final status: ACCEPTABLE_WITH_BLOCKER.

## Verification

Read-only commands used:

```bash
sed -n ... docs/reports/phase-3/configedit-template-object-model-design/03-codex-audit-prompt.md
sed -n ... docs/README.md docs/PHASE-STATUS.md docs/RELEASE_NOTE_v0.1.9.md docs/CURRENT.md
sed -n ... docs/design/backend-runtime-runplan-docker.md docs/design/runtime-template-node-runtime-snapshot.md
rg --files internal/server web/src configs/backend-catalog
rg -n "ConfigEdit|configedit|RunPlan|CUDA_VISIBLE_DEVICES|--gpus|device_binding|service_json|placement_json" internal/server web/src configs
sed -n ... internal/server/configedit/*.go
sed -n ... internal/server/catalog/*.go
sed -n ... internal/server/runplan/*.go
sed -n ... internal/server/api/*deployment* internal/server/api/*runtime* internal/server/api/config_edit_handlers.go
sed -n ... web/src/components/config/*.vue web/src/components/deployments/*.vue web/src/pages/*.vue
sed -n ... configs/backend-catalog/** configs/config-registry/items.yaml
```

No build/test commands were run because the requested pass was read-only architecture audit and planning.
