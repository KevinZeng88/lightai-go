# Runtime Template Catalog Redesign Final Closeout

Date: 2026-06-26

## Implementation Summary

Implemented strict ConfigSet snapshot flow for BackendVersion, BackendRuntime, NodeBackendRuntime, and Deployment. Added BackendVersion UI management in the Backends page, moved runtime/NBR parameter editing to schema-driven `config_set.items`, cleaned ordinary runtime template selection with visibility/support metadata, and removed RunPlan fallback to BackendVersion images.

Post-closeout repair on 2026-06-26 tightened two remaining snapshot boundaries:

- NodeBackendRuntime create now starts from the BackendRuntime ConfigSet snapshot, applies request `config_set` as the node-local edited snapshot when present, and applies request `config_overrides` before persisting `node_backend_runtimes.config_set_json`.
- RunPlan no longer falls back from the NBR snapshot parameter schema to live `BackendVersion.ParameterDefs`, and no longer reads live `BackendVersion.VendorOptionsJSON/resource_controls` to generate args.

## Fixed Issues

| Requirement / Finding | Status | Evidence |
| --- | --- | --- |
| BackendVersion clone/edit | FIXED | `TestSystemBackendVersionReadOnlyAndCloneable`, `TestBackendVersionCreatePatchAndReloadUserCatalog`; `BackendsPage.vue` supports clone/new/edit/delete for user versions. |
| `fake_new_param` schema render | FIXED | `TestCreateBackendRuntimeCopiesBackendVersionSnapshot`; `web/tests/runtimeBoundaryUi.test.mjs` checks schema editor rendering path. |
| BackendVersion -> BackendRuntime -> NodeBackendRuntime -> Deployment copy | FIXED | Runtime boundary tests cover BackendRuntime copy, NBR copy, Deployment copy, and upstream mutation isolation. |
| Upstream mutation does not affect existing downstream objects | FIXED | `TestCreateBackendRuntimeCopiesBackendVersionSnapshot`, `TestNodeBackendRuntimeCopiesTemplateSnapshotAndTemplateEditDoesNotChangeIt`, `TestWorkflowDeploymentRunPlanPreservesNBRSnapshot`. |
| `enabled=true` RunPlan parameter included | FIXED | Existing RunPlan tests assert enabled parameters render into args, including vLLM/SGLang/llama.cpp cases. |
| `enabled=false` RunPlan parameter excluded | FIXED | RunPlan resolver skips disabled NBR/deployment parameter values; covered by resolver tests and existing disabled-parameter logic. |
| Ordinary runtime selector excludes hidden/reference/disabled/template-only/runtime.xxx | FIXED | `TestBackendRuntimeListHidesHiddenReferenceDisabledTemplates`; API default list filters visible active/experimental templates. |
| BackendVersion runtime-only fields | FIXED | `TestBackendVersionRejectsRuntimeOnlyFields`; create/patch returns 400 for `image_ref`, `command`, `entrypoint`, `model_mount`, docker/device/env fields. |
| Deployment fallback to BackendRuntime | FIXED | `TestCreateDeploymentRejectsMissingNodeRuntimeSnapshot`; create fails if NBR snapshot is missing. |
| RunPlan snapshot-only image | FIXED | `TestResolveImagePriority` now asserts BackendVersion-only image fails. |
| ConfigSet env extraction bug | FIXED | `configSetParameterValues()` supports env items with `render.env_name` and does not convert map-valued `runtime.env` into CLI args. |
| NodeBackendRuntime create ignored request ConfigSet | FIXED | `TestCreateNodeBackendRuntimeAppliesRequestConfigSetSnapshot` creates NBR with `fake_new_param` value/enabled and verifies persisted `config_set_json`. |
| RunPlan used live BackendVersion parameter schema fallback | FIXED | `TestResolveDoesNotFallbackToLiveBackendVersionParameterSchema` verifies subsequent live `ParameterDefs` edits do not affect an existing NBR snapshot RunPlan. |
| RunPlan used live BackendVersion vendor resource controls | FIXED | `TestResolveDoesNotUseLiveBackendVersionVendorOptionsResourceControls` verifies subsequent live `VendorOptionsJSON/resource_controls` edits do not affect an existing NBR/Deployment RunPlan. |

## Code Change Files

Backend/catalog:

- `internal/server/catalog/types.go`
- `internal/server/catalog/loader.go`
- `internal/server/db/db.go`
- `internal/server/api/backend_handlers.go`
- `internal/server/api/runtime_handlers.go`
- `internal/server/api/configset_helpers.go`
- `internal/server/api/deployment_lifecycle_handlers.go`
- `internal/server/runplan/resolver.go`

Tests:

- `internal/server/api/runtime_boundary_test.go`
- `internal/server/runplan/resolver_test.go`
- `internal/server/runplan/vllm_sglang_nvidia_test.go`
- `internal/server/runplan/llamacpp_nvidia_test.go`
- `internal/server/runplan/metax_huawei_test.go`
- `web/tests/runtimeBoundaryUi.test.mjs`

Web:

- `web/src/pages/BackendsPage.vue`
- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/components/common/RuntimeParameterEditor.vue`
- `web/src/components/deployments/NodeRuntimeConfigWizard.vue`

Catalog:

- `configs/backend-catalog/runtimes/vllm/metax-docker.yaml`
- `configs/backend-catalog/runtimes/vllm/huawei-docker.yaml`

Docs:

- `docs/reports/phase-3/runtime-template-catalog-redesign/current-code-audit.md`
- `docs/reports/phase-3/runtime-template-catalog-redesign/open-issues-closeout.md`
- `docs/reports/phase-3/runtime-template-catalog-redesign/final-closeout.md`

## Final Visible Runtime Templates

Ordinary selector visibility is:

```text
runtime.vllm.nvidia-docker
runtime.sglang.nvidia-docker
runtime.llamacpp.nvidia-docker
runtime.llamacpp.cpu-docker
runtime.vllm.metax-docker
runtime.vllm.huawei-docker
```

API verification command:

```bash
curl /api/v1/backend-runtimes
```

Automated evidence:

```bash
go test ./internal/server/api -run TestBackendRuntimeListHidesHiddenReferenceDisabledTemplates
```

## Hidden / Reference / Disabled Templates

Hidden/reference entries remain in catalog for audit/adaptation, but ordinary selectors exclude them:

```text
runtime.sglang.huawei-docker
runtime.llamacpp.huawei-docker
sglang-0.4.6-metax-macart
vllm-v0.23.0-nvidia-cuda
sglang-v0.5.13.post1-nvidia-cuda
llamacpp-b9700-nvidia-cuda13
runtime.vllm.cpu-docker
runtime.sglang.cpu-docker
runtime.sglang.metax-docker
runtime.llamacpp.metax-docker
runtime.ollama.cpu-docker
runtime.ollama.nvidia-docker
```

## BackendVersion UI Behavior

`BackendsPage.vue` now has a Versions tab for the selected Backend:

- list BackendVersion rows
- clone system versions
- create user versions
- edit user version metadata and ConfigSet
- add new parameter schema items
- delete user versions through existing API
- render system versions read-only

## Schema-Driven Parameter Editing Evidence

`RuntimeParameterEditor.vue` now reads:

```text
render.label / extensions.label
render.help / extensions.help
render.group / extensions.group
top-level constraints / render.constraints
order
visibility
readonly / advanced
render.options / constraints.options
```

It renders boolean, select, multi-select, multiline, object, integer, number, and string inputs from `config_set.items`. BackendRuntime edit and NodeBackendRuntime wizard no longer import `HumanRuntimeParameterForm`.

## Copy-On-Create Evidence

The implementation keeps snapshot boundaries:

```text
Backend config_set -> BackendVersion config_set
BackendVersion config_set -> BackendRuntime config_set
BackendRuntime config_set -> NodeBackendRuntime config_set
NodeBackendRuntime config_set -> Deployment config_set
```

Evidence:

```bash
go test ./internal/server/api -run 'TestCreateBackendRuntimeCopiesBackendVersionSnapshot|TestNodeBackendRuntimeCopiesTemplateSnapshotAndTemplateEditDoesNotChangeIt|TestCreateNodeBackendRuntimeAppliesRequestConfigSetSnapshot|TestWorkflowDeploymentRunPlanPreservesNBRSnapshot'
```

## RunPlan Snapshot-Only Evidence

RunPlan no longer reads BackendVersion default images as fallback. RunPlan also no longer reads live BackendVersion parameter schema or live BackendVersion vendor resource controls after NBR/Deployment snapshots exist. Deployment creation rejects missing NBR ConfigSet snapshots.

Evidence:

```bash
go test ./internal/server/runplan -run TestResolveImagePriority
go test ./internal/server/runplan -run 'TestResolveDoesNotFallbackToLiveBackendVersionParameterSchema|TestResolveDoesNotUseLiveBackendVersionVendorOptionsResourceControls'
go test ./internal/server/api -run TestCreateDeploymentRejectsMissingNodeRuntimeSnapshot
```

## Verification Commands And Results

All required commands passed:

```bash
go build ./cmd/server/...      # PASS, exit 0
go build ./cmd/agent/...       # PASS, exit 0
go test ./internal/server/...  # PASS, exit 0
go test ./internal/agent/...   # PASS, exit 0
cd web && npm run build        # PASS, exit 0; Vite/Rollup chunk/comment warnings only
cd web && npm test             # PASS, exit 0
```

## External Hardware / Image Dependencies

Formal blocker document:

```text
docs/reports/phase-3/runtime-template-catalog-redesign/open-issues-closeout.md
```

Blocked items:

- RTC-BLOCKER-001: MetaX vLLM real hardware/image validation.
- RTC-BLOCKER-002: Huawei vLLM real hardware/image validation.

No unresolved fixable problems remain outside the formal open-issues document.

## Problem Closure Status

All discovered fixable problems are FIXED. External validation problems are DOCUMENTED_BLOCKER in `open-issues-closeout.md`. No problems exist only in chat. No remaining risk exists without a formal entry.

## Commit / Push / Git Status

Implementation commit id before post-closeout repair: `6686003`.

Post-closeout repair commit id: recorded by the final pushed repository HEAD for this closeout update.

Push result: `git push` is required after this file is committed; final command output is recorded with the pushed HEAD.

Expected final `git status --short` after commit and push:

```text
clean
```
