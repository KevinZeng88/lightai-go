# ConfigEdit Object Model Full Implementation Closeout

## Final status

- Status: PASS_WITH_MVP_LIMITATIONS
- Date: 2026-06-30
- Branch: main
- Final implementation commit: `5a3e0ba feat(configedit): add object model templates and runplan effects`
- Closeout commit: pending at document creation
- Push result: pending at document creation
- Final git status: pending after closeout commit and push

## Root cause

The original problem class was architectural, not a single vLLM parameter defect:

- ConfigEdit was primarily a projected editable field list, not a complete object contract.
- Runtime-affecting parameters were split across ConfigSet, `service_json`, `placement_json`, RunPlan resolver code, semantic adapters, and page-specific UI.
- Device binding, service ports, mounts, health checks, CLI args, env, and Docker options could appear in the final Docker command without a complete editable ConfigEdit component/effect chain.
- ConfigEdit component templates did not exist as an externalizable layer separate from runtime templates.

## Implementation summary by work package

### Work Package A — Contract Tests + ConfigEdit Object Foundation

- Completed:
  - Added ConfigEdit object metadata: `template_id`, `snapshot_id`, `parent`, `child_init`, `view_level`, top-level `fields`, `components`, and `effects_preview`.
  - Preserved legacy `sections` compatibility.
  - Added reset-to-parent/default backend helpers.
  - Materialized Deployment `service_json` and `placement_json` compatibility fields into ConfigEdit components.
- Files changed:
  - `internal/server/configedit/types.go`
  - `internal/server/configedit/project.go`
  - `internal/server/configedit/apply.go`
  - `internal/server/api/config_edit_handlers.go`
  - `internal/server/configedit/object_model_test.go`
- Tests:
  - `TestProjectConfigSetToEditViewReturnsObjectContract`
  - `TestResetFieldToDefaultAndParent`
- Evidence:
  - `go test ./internal/server/configedit/... ./internal/server/runplan/... ./internal/server/api/...` PASS
- Commit:
  - `5a3e0ba`

### Work Package B — External ConfigEdit Component Template + Effect Engine

- Completed:
  - Added external ConfigEdit component template loader and validator.
  - Added built-in/local roots with local precedence.
  - Added built-in vLLM, SGLang, and llama.cpp NVIDIA Docker ConfigEdit templates.
  - Added template list/get/validate/clone API.
- Files changed:
  - `internal/server/configedit/templates.go`
  - `internal/server/configedit/templates_test.go`
  - `internal/server/api/configedit_template_handlers.go`
  - `configs/configedit-templates/builtin/*.yaml`
- Tests:
  - `TestLoadComponentTemplatesLocalOverridePrecedence`
  - `TestValidateComponentTemplateRejectsUnsafeUnknowns`
- Evidence:
  - Local override wins by `template_id`.
  - Unknown renderers/effects and unsafe expressions fail validation.
- Commit:
  - `5a3e0ba`

### Work Package C — Runtime Effects Components + RunPlan Compiler Cleanup

- Completed:
  - Added `runtime.device_binding` and `service.port_binding` materialized inputs to RunPlan snapshots.
  - RunPlan device binding now resolves from ConfigEdit component first.
  - Docker preview renders `DeviceBinding.DockerGPUOption` from the resolved spec.
  - Source map marks GPU/env effects as `configedit_effect` with `runtime.device_binding` patch target.
  - Disabled device binding removes inherited visible-device env.
- Files changed:
  - `internal/server/runplan/resolver.go`
  - `internal/server/runplan/preview.go`
  - `internal/server/runplan/resolve_with_sourcemap.go`
  - `internal/server/runplan/source_map.go`
  - `internal/server/api/configset_helpers.go`
  - `internal/server/api/deployment_preview_handlers.go`
  - `internal/server/api/deployment_lifecycle_handlers.go`
- Tests:
  - Updated RunPlan device binding and source map tests.
- Evidence:
  - `--gpus` and visible-device env source map entries now point to `runtime.device_binding`.
  - Disabled binding test verifies GPU option/env removal.
- Commit:
  - `5a3e0ba`

### Work Package D — UI Full-Chain Integration + Template Management MVP + Final Audit

- Completed:
  - Extended Web ConfigEdit types for object/component/effect metadata.
  - Added `accelerator_binding` renderer.
  - Updated operator ConfigEdit calls to request `view_level: normal`.
  - Hid raw source map and final RunPlan JSON behind developer mode in deployment preview.
  - Added ConfigEdit Template Management MVP page.
- Files changed:
  - `web/src/utils/configEditView.ts`
  - `web/src/api/configEdit.ts`
  - `web/src/components/config/ConfigField.vue`
  - `web/src/components/deployments/DeploymentPreviewPanel.vue`
  - `web/src/pages/ConfigEditTemplatesPage.vue`
  - `web/src/router/index.ts`
  - `web/src/layouts/ConsoleLayout.vue`
- Tests:
  - Deployment preview test now covers developer-mode source map and normal-mode hiding.
- Evidence:
  - `npm run test:unit` PASS
  - `npm run build` PASS
- Commit:
  - `5a3e0ba`

## ConfigEdit object API examples

Representative object shape returned by `/api/v1/config-edit/view`:

```json
{
  "object_kind": "deployment",
  "object_id": "dep-1",
  "template_id": "vllm-nvidia-docker-configedit-v1",
  "snapshot_id": "sha256:...",
  "parent": {
    "object_kind": "node_backend_runtime",
    "object_id": "node-1:runtime.vllm.nvidia-docker"
  },
  "child_init": {
    "strategy": "copy_effective_snapshot",
    "copy_scope": "whole_effective_configedit_snapshot"
  },
  "view_level": "normal",
  "sections": [],
  "components": [],
  "fields": [],
  "effects_preview": []
}
```

BackendRuntime, NodeBackendRuntime, Deployment, and deployment override all use this same object response path.

## External ConfigEdit template examples

Built-in templates:

- `vllm-nvidia-docker-configedit-v1`
- `sglang-nvidia-docker-configedit-v1`
- `llamacpp-nvidia-docker-configedit-v1`

Each template defines:

- applicability by backend/runtime/vendor
- supported view levels
- sections
- components
- renderer names
- effects for CLI/env/Docker/mount/port/health/device binding

Template roots:

- Built-in: `configs/configedit-templates/builtin/`
- Local override: `configs/configedit-templates/local/`

## Parent/child/copy snapshot evidence

- BackendRuntime creation already copies BackendVersion ConfigSet.
- NodeBackendRuntime enable already copies BackendRuntime ConfigSet.
- Deployment create copies NodeBackendRuntime ConfigSet and now materializes service/device compatibility inputs into the Deployment ConfigEdit snapshot.
- `TestProjectConfigSetToEditViewReturnsObjectContract` verifies parent and child-init metadata.
- `TestResetFieldToDefaultAndParent` verifies reset behavior.

## Runtime effects evidence

### Device Binding

- Auto/manual/disabled behavior is represented in `runtime.device_binding`.
- Final Docker `--gpus` renders from `DeviceBinding.DockerGPUOption`.
- Visible device env renders from `DeviceBinding.VisibleEnvKey/VisibleEnvValue`.
- Source map records `configedit_effect` with patch target `runtime.device_binding`.

### Service Port

- `service.port_binding` can provide host/container port and listen host.
- RunPlan `applyServiceArgs` uses effective ConfigEdit service binding.

### Model Mount

- Existing `runtime.model_mount` remains the ConfigEdit component for container path and readonly policy.
- Platform-owned safe host path remains generated from selected ModelLocation.

### Health Check

- Existing `runtime.health` remains the ConfigEdit component for health check configuration.

### Args / Env / Docker Options

- Existing ConfigSet-backed args/env/Docker options are exposed as ConfigEdit components and effects preview entries.
- Device visible env is no longer resolver-only in the source map.

## RunPlan hidden injection cleanup

Converted or redirected:

- `--gpus` → `runtime.device_binding` ConfigEdit effect
- visible-device env → `runtime.device_binding` ConfigEdit effect
- service port/listen host → `service.port_binding` where materialized
- Docker preview → renders resolved spec, preferring `DeviceBinding.DockerGPUOption`

Allowed platform-generated readonly fields:

- container name
- instance id
- operation/lease id
- hardware inventory evidence
- safe resolved host model path

## Docker command token mapping evidence

### vLLM

- Template: `vllm-nvidia-docker-configedit-v1`
- Covered by RunPlan NVIDIA tests and source map tests.

### SGLang

- Template: `sglang-nvidia-docker-configedit-v1`
- Covered by RunPlan source visibility matrix.

### llama.cpp

- Template: `llamacpp-nvidia-docker-configedit-v1`
- Covered by RunPlan source visibility matrix and llama.cpp tests.

## UI evidence

### Normal view

- Main operator ConfigEdit callers pass `view_level: normal`.
- Deployment preview hides raw source map and final RunPlan JSON unless `developerMode` is true.
- Test: `DeploymentPreviewPanel.render.test.ts` normal-mode hiding assertion.

### Advanced / Developer view

- Backend API supports `view_level: advanced` and `view_level: developer`.
- Developer mode can show raw source maps and final RunPlan JSON.

### Deployment create/edit

- Deployment override and edit use ConfigEdit object APIs.
- Device binding renderer supports enabled/mode/vendor/accelerator IDs/visible env/Docker GPU option when `runtime.device_binding` exists in the object.

## ConfigEdit Template Management MVP evidence

Implemented:

- list built-in/local templates
- view template metadata, sections, components, effects
- clone built-in to local YAML
- raw JSON editor
- server-side validate/lint

MVP limitation:

- Live ConfigEdit object preview and live RunPlan preview from the template management page are not fully wired.

## Tests run

```bash
git status --short
go test ./...
cd web
npm test
npm run test:unit
npm run build
```

Results:

- `git status --short` before full verification: clean
- `go test ./...`: PASS
- `npm test`: PASS
- `npm run test:unit`: PASS
- `npm run build`: PASS, with Vite chunk-size warning only
- `git status --short` after full verification and before closeout document: clean

## Commits

- `5a3e0ba feat(configedit): add object model templates and runplan effects`
- Closeout commit pending at document creation.

## Push result

Pending at document creation.

## Remaining limitations

| ID | Status | Evidence | Impact | Reason not completed | Recommended next action | Owner |
| --- | --- | --- | --- | --- | --- | --- |
| CE-MVP-001 | MVP_LIMITATION | `service_json` and `placement_json` remain persisted compatibility columns | Duplicate storage still exists, though preview/create materialize into ConfigEdit | Removing columns would require broader migration/backfill not required for MVP | Keep compatibility columns until a schema cleanup migration can safely remove or fully deprecate them | Platform |
| CE-MVP-002 | MVP_LIMITATION | Template management page supports raw edit/validate but not live RunPlan fixture preview | Operators can validate templates but cannot preview a live Docker command from that page alone | Requires fixture/live-object selection UX | Add template preview endpoint/page workflow using selected backend/node/model fixture | Platform |
| CE-MVP-003 | MVP_LIMITATION | Existing legacy snapshots may lack `runtime.device_binding` until preview/create materializes it | Old rows depend on compatibility initialization | Safe in-place full backfill was outside the MVP scope | Add idempotent migration/backfill command for existing deployments/NBRs | Platform |

No fixable issue discovered during this run remains unaddressed. No problem exists only in chat history.

## Final decision

PASS_WITH_MVP_LIMITATIONS
