# ConfigEdit Object Model Autonomous Execution Log

## Work Package A — Contract Tests + ConfigEdit Object Foundation

### Scope completed

- Added versioned ConfigEdit object fields on the existing view contract: `template_id`, `snapshot_id`, `parent`, `child_init`, `view_level`, top-level `fields`, `components`, and `effects_preview`.
- Kept the legacy `sections[].fields[]` shape for existing UI/API compatibility.
- Added backend reset helpers for reset-to-default and reset-to-parent.
- Added Deployment ConfigEdit compatibility materialization for `service_json` and `placement_json` into `service.port_binding` and `runtime.device_binding` ConfigSet items.

### Files changed

- `internal/server/configedit/types.go`
- `internal/server/configedit/project.go`
- `internal/server/configedit/apply.go`
- `internal/server/configedit/taxonomy.go`
- `internal/server/api/config_edit_handlers.go`
- `internal/server/configedit/object_model_test.go`

### Tests added or updated

- `TestProjectConfigSetToEditViewReturnsObjectContract`
- `TestResetFieldToDefaultAndParent`

### Tests run

- `go test ./internal/server/configedit/... ./internal/server/runplan/... ./internal/server/api/...` — PASS

### Failures found

- Existing ConfigEdit tests expected default full view; new view filtering initially treated `mode=edit` as normal.

### Fixes applied

- Preserved compatibility by making default/mode-based projection return developer/full view unless `view_level` is explicitly supplied.
- Updated Web callers to explicitly request `view_level: normal` on operator paths.

### Self-audit answers

- ConfigEdit is now an object contract layered over the previous field projection.
- BackendRuntime, NodeBackendRuntime, Deployment, and deployment override use the same `/config-edit/view` object response.
- Deployment ConfigEdit receives parent metadata and materialized service/device components.
- Deployment create copies and persists the NBR snapshot, then materializes compatibility service/device components into the Deployment snapshot.
- `service_json` and `placement_json` remain compatibility inputs, but preview/create materialize them into ConfigEdit components before RunPlan input.

### Remaining limitations

- MVP_LIMITATION: storage still keeps `service_json` and `placement_json` for API compatibility; they are mirrored into ConfigEdit instead of removed.

### Commit id

- 5a3e0ba

## Work Package B — External ConfigEdit Component Template + Effect Engine

### Scope completed

- Added external ConfigEdit component template structs, loader, validation, and local-over-built-in precedence.
- Added built-in templates for vLLM NVIDIA Docker, SGLang NVIDIA Docker, and llama.cpp NVIDIA Docker.
- Added server APIs to list/get/validate/clone templates.

### Files changed

- `internal/server/configedit/templates.go`
- `internal/server/configedit/templates_test.go`
- `internal/server/api/configedit_template_handlers.go`
- `internal/server/api/router.go`
- `configs/configedit-templates/builtin/*.yaml`

### Tests added or updated

- `TestLoadComponentTemplatesLocalOverridePrecedence`
- `TestValidateComponentTemplateRejectsUnsafeUnknowns`

### Tests run

- `go test ./internal/server/configedit/... ./internal/server/runplan/... ./internal/server/api/...` — PASS

### Failures found

- No remaining backend failures after validation and loader tests.

### Fixes applied

- Template loader uses recursive `WalkDir`.
- Unsafe expression and unknown renderer/effect validation fails closed.

### Self-audit answers

- Runtime templates and ConfigEdit component templates are separate directories and contracts.
- Built-in and local override roots are supported.
- Local override wins by `template_id`.
- vLLM/SGLang/llama.cpp built-ins validate.
- Adding parameters is data-driven when the renderer/effect type already exists.

### Remaining limitations

- MVP_LIMITATION: template save is clone-to-local YAML; full structured field-by-field authoring is limited to the MVP raw editor.

### Commit id

- 5a3e0ba

## Work Package C — Runtime Effects Components + RunPlan Compiler Cleanup

### Scope completed

- Added materialized `runtime.device_binding` and `service.port_binding` snapshot inputs to RunPlan.
- RunPlan now builds device binding from the ConfigEdit component path first; placement is compatibility initialization only.
- Docker command preview renders `DeviceBinding.DockerGPUOption` from the resolved spec.
- Source map records GPU Docker/env effects as `configedit_effect` with `runtime.device_binding` patch target.
- Service port binding can drive container/host/app port and host CLI args.

### Files changed

- `internal/server/runplan/resolver.go`
- `internal/server/runplan/preview.go`
- `internal/server/runplan/resolve_with_sourcemap.go`
- `internal/server/runplan/source_map.go`
- `internal/server/api/configset_helpers.go`
- `internal/server/api/deployment_preview_handlers.go`
- `internal/server/api/deployment_lifecycle_handlers.go`

### Tests added or updated

- Updated device binding/source-map tests to assert ConfigEdit effect source instead of resolver-only system generation.

### Tests run

- `go test ./internal/server/configedit/... ./internal/server/runplan/... ./internal/server/api/...` — PASS

### Failures found

- Disabled device binding initially left a default `CUDA_VISIBLE_DEVICES` from the NBR env snapshot.

### Fixes applied

- Disabled binding now carries the visible env key so Resolve removes inherited visible-device env from the final env map.

### Self-audit answers

- Final `--gpus` is sourced from `runtime.device_binding` effect/field in the resolved plan.
- Final visible device env is sourced from `runtime.device_binding` effect/field in the resolved plan.
- Ports are sourced from `service.port_binding` when materialized.
- Source map explains effects without using source map as source of truth.
- EquivalentCommandPreview renders the resolved spec.

### Remaining limitations

- MVP_LIMITATION: old snapshots without `runtime.device_binding` still use compatibility initialization from placement/assigned GPUs.

### Commit id

- 5a3e0ba

## Work Package D — UI Full-Chain Integration + Template Management MVP + Final Audit

### Scope completed

- Extended Web ConfigEdit types for object metadata, components, effects, and reset metadata.
- Added `accelerator_binding` renderer for device binding.
- Updated primary operator ConfigEdit API calls to request `view_level: normal`.
- Hid raw source map and final RunPlan JSON behind `developerMode` in deployment preview.
- Added ConfigEdit Template Management MVP page and navigation entry.

### Files changed

- `web/src/utils/configEditView.ts`
- `web/src/api/configEdit.ts`
- `web/src/components/config/ConfigField.vue`
- `web/src/components/deployments/DeploymentPreviewPanel.vue`
- `web/src/components/deployments/__tests__/DeploymentPreviewPanel.render.test.ts`
- `web/src/pages/ConfigEditTemplatesPage.vue`
- `web/src/router/index.ts`
- `web/src/layouts/ConsoleLayout.vue`
- ConfigEdit caller pages/components under `web/src/pages` and `web/src/components/deployments`

### Tests added or updated

- Updated deployment preview tests to assert developer-mode source map visibility and normal-mode raw/source hiding.

### Tests run

- `npm run test:unit` — PASS

### Failures found

- Existing preview panel test expected source map in normal mode.

### Fixes applied

- Test now passes `developerMode` when asserting source map and adds normal-mode hiding assertion.

### Self-audit answers

- Normal UI paths request `view_level: normal`.
- Preview raw source map and final RunPlan JSON require developer mode.
- Device binding can be edited through ConfigEdit field renderer when present in the object.
- Operators can list, view, clone, validate, and raw-edit/check templates through the MVP page.

### Remaining limitations

- MVP_LIMITATION: template preview against live model/node fixtures is not fully wired into the page; validation and object/template inspection are available.

### Commit id

- 5a3e0ba
