# Runtime RunPlan Source Visibility and Preview Diagnostics Closeout

Date: 2026-06-30

## Root Cause

Deployment preview mixed final RunPlan resolution with diagnostic data. `ResolveWithSourceMap` could return a usable `ResolvedRunPlan` while also returning non-final resolver diagnostics, and the preview handler converted every resolver diagnostic into a blocking `resolve_error`. The frontend then ignored the backend-provided message/reason and translated only the code; because `resolve_error` had no locale entry, users saw repeated `[resolve_error] 未知错误`.

The second root cause was incomplete self-description in `parameter_source_map` and `device_binding`. Final Docker effects such as GPU binding, visible devices env, model mount, Docker options, ports, health check, and image were present in the final command but did not consistently carry `source`, `patch_target`, `docker_effect`, `blocking`, and editability metadata.

## Problem Categories

- Configuration was explainable in final Docker command but not explainable in preview/source map.
- System-derived injections were hidden or only visible as final command text.
- Non-blocking diagnostics incorrectly affected `can_run`.
- Frontend preview fallback hid useful backend messages behind `未知错误`.
- Page-level detail view still contained legacy GPU-specific rendering instead of consuming the backend contract.

## Repair Scope

- Backend RunPlan contract:
  - `ResolvedRunPlan.device_binding`
  - `parameter_source_map`
  - health check defaulting
  - placement/device binding mode
- Deployment preview API:
  - structured issue entries
  - deduplication
  - `can_run` from final blocking errors only
  - placement parsing for auto/manual/disabled binding
- Frontend preview:
  - structured error display
  - source map display
  - device binding injection display from contract
  - no page-side NVIDIA/CUDA/`--gpus` derivation

## Modified Files

- `internal/server/runplan/types.go`
- `internal/server/runplan/source_map.go`
- `internal/server/runplan/resolve_with_sourcemap.go`
- `internal/server/runplan/resolver.go`
- `internal/server/api/deployment_preview_handlers.go`
- `internal/server/api/preflight_handlers.go`
- `internal/server/api/helpers.go`
- `web/src/components/deployments/DeploymentPreviewPanel.vue`
- `web/src/pages/ModelDeploymentsPage.vue`
- Tests:
  - `internal/server/runplan/resolver_test.go`
  - `internal/server/runplan/source_map_test.go`
  - `internal/server/runplan/source_visibility_matrix_test.go`
  - `internal/server/api/deployment_preflight_contract_test.go`
  - `web/src/components/deployments/__tests__/DeploymentPreviewPanel.render.test.ts`

## Backend Coverage

- vLLM NVIDIA Docker: covered by RunPlan source visibility matrix and API preview tests.
- SGLang NVIDIA Docker: covered by RunPlan source visibility matrix.
- llama.cpp NVIDIA Docker: covered by RunPlan source visibility matrix.
- BackendRuntime / NodeBackendRuntime / Deployment override: source map entries now expose layer/source/patch target across image, args, env, Docker options, model mount, ports, health check, and device binding.
- RunPlan preview: Docker command preview is generated from the final `ResolvedRunPlan`.

## GPU / Device Binding Design

`PlacementInfo` now carries vendor-neutral device binding intent:

- `device_binding_enabled`
- `accelerator_selection_mode`: `auto`, `manual`, `disabled`
- `accelerator_ids`
- `accelerator_count`
- `allow_auto_select`

`ResolvedRunPlan.device_binding` now carries:

- enabled state
- selection mode
- vendor
- accelerator ids/count
- source
- patch target
- final Docker/env effects
- injection preview entries

For NVIDIA, the resolver derives:

- Docker effect: `--gpus "device=<ids>"`
- env effect: visible devices env with the selected indexes

When binding is disabled, the resolver removes visible-devices env and does not emit GPU ids or Docker GPU options. MetaX/Huawei/CPU remain vendor-neutral through the same structure; vendor-specific effects must be emitted by the resolver contract rather than page logic.

## Hidden Injection Review

| Field | Final command effect | Source now visible | Editable / patch target | Status |
| --- | --- | --- | --- | --- |
| image | Docker image argument | `parameter_source_map.image` | NBR/runtime config | FIXED |
| args / extra_args | command after image | `parameter_source_map.args` | NBR or deployment override | FIXED |
| env / extra_env | `-e KEY=value` | `parameter_source_map.env` | NBR/deployment override or derived | FIXED |
| model mount | `-v host:container:ro` | `parameter_source_map.mounts` | model location/runtime mount | FIXED |
| ports | `-p host:container/tcp` | `parameter_source_map.ports` | deployment service config | FIXED |
| ipc_mode | `--ipc` | `parameter_source_map.docker_options` | NBR/runtime Docker options | FIXED |
| shm_size | `--shm-size` | `parameter_source_map.docker_options` | NBR/runtime Docker options | FIXED |
| devices | `--device` | `parameter_source_map.devices` | NBR/runtime device config | FIXED |
| GPU binding | `--gpus`, visible-devices env | `device_binding` and `system_generated` source entries | deployment placement | FIXED |
| health_check | container health config | `parameter_source_map.health_check` | NBR/runtime health config | FIXED |
| host / port / model path | backend args | final args and source map | deployment service/runtime config/model location | FIXED |
| served_model_name | backend args | args/source map when present | deployment override/service metadata | FIXED |

## Resolve Error Fix

- `preview.can_run` now requires a non-nil final plan and no blocking issue.
- Resolver diagnostics with a valid final plan are warnings, not blocking errors.
- All issue entries include `code`, `message`, `key`, `path`, `reason`, `source`, `severity`, and `blocking`.
- Preview response errors/warnings are deduplicated.
- Frontend displays backend message/reason first and never collapses unknown `resolve_error` into repeated `未知错误`.

## Health Check Review

HTTP health checks with a path now materialize safe defaults:

- `expected_status=0` becomes `200`
- `startup_timeout_seconds=0` becomes `120`
- `interval_seconds=0` becomes `5`
- `timeout_seconds=0` becomes `3`

Zero health-check defaults no longer generate resolver errors. If a future backend intentionally uses `0` to mean “do not check status”, that must be represented explicitly in the health-check contract instead of overloading a missing default.

## Tests Run

- `go test ./internal/server/runplan ./internal/server/api`
  - Result: PASS
- `go test ./...`
  - Result: PASS
- `cd web && npm test`
  - Result: PASS
- `cd web && npm run test:unit`
  - Result: PASS
- `cd web && npm run build`
  - Result: PASS

## API / Manual Evidence

Automated API evidence:

- `TestDeploymentPreviewCanRunUsesResolvedPlanAndExplainsSources`
  - verifies `can_run=true` when a final plan exists
  - verifies non-empty Docker preview
  - verifies `device_binding`, `parameter_source_map`, `patch_target`, `docker_effect`, and `blocking`
- `TestDeploymentPreviewDeviceBindingDisabledOmitsGPUInjection`
  - verifies disabled binding omits GPU Docker/env injection

Manual API check for a running local server:

```bash
curl -s -X POST http://localhost:18080/api/v1/deployments/preview \
  -H 'Content-Type: application/json' \
  -d '{"model_artifact_id":"<artifact>","node_backend_runtime_id":"<nbr>","service_json":{"host_port":8000,"container_port":8000,"app_port":8000},"placement_json":{"accelerator_selection_mode":"auto","allow_auto_select":true}}'
```

Expected evidence: `can_run` reflects final blocking errors only, `docker_preview` is generated from `run_plan`, and `run_plan.parameter_source_map` plus `run_plan.device_binding` explain final Docker effects.

## Problem Closure

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| RP-001 | Final resolved plan could be marked not runnable due to non-final diagnostics | Preview handler used all resolver errors as blocking | False `可运行：否` despite runnable Docker command | FIXED | `deployment_preview_handlers.go` | API preview tests, `go test ./...` | `can_run` is now based on final plan plus blocking issues |
| RP-002 | Repeated `[resolve_error] 未知错误` in UI | Frontend ignored backend message/reason and only translated code | Users could not diagnose key/path/source | FIXED | `DeploymentPreviewPanel.vue` | frontend render test, `npm test` | Structured diagnostics display message/key/path/source/blocking |
| RP-003 | GPU injection hidden as system-generated side effect | Device binding lacked mode/source/patch target/injection preview | Users could not explain or disable `--gpus` / visible devices env | FIXED | `resolver.go`, `types.go`, `resolve_with_sourcemap.go` | resolver/API tests | Auto/manual/disabled contract is explicit |
| RP-004 | Key Docker effects lacked self-contained source entries | Source map entries lacked patch target/effect/source kind | Preview/UI needed page knowledge | FIXED | `source_map.go` | source map and matrix tests | Source map entries are self-contained |
| RP-005 | Health check zero defaults could be ambiguous | `expected_status`, interval, timeout could remain `0` | Could cause confusing diagnostics | FIXED | `resolver.go` | resolver test | HTTP health defaults materialize safe values |
| RP-006 | Vite build emitted dependency/chunk warnings | `npm run build` exited 0 with Rollup annotation/chunk-size warnings | No task-specific failure | INVALID | N/A | build exit code 0 | Not a blocking problem |

## Unresolved Problems

No unresolved problems remain from this implementation. No problems exist only in chat.

## Commit / Push

- Commit id: reported in final response after commit creation.
- Push result: reported in final response after `git push`.
- Git status: recorded in final response after push.
