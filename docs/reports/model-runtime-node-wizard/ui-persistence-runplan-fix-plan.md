# UI Persistence RunPlan Fix Plan

> Status: CLOSED
> Date: 2026-06-19
> Scope: UI editable field persistence, Deployment/Instance responsibility split, RunPlan/Docker consistency, run idempotency

## Objective

Ensure editable UI fields persist through API/DB/list/detail/edit flows and are reflected in Deployment, RunPlan, Docker command preview, and Agent Docker create spec. Enforce backend run idempotency for single-active-instance deployments and document any externally blocked validation.

## Object Responsibilities

- ModelArtifact: model file metadata, user-facing display name, internal artifact name, format, path metadata, checksum/quantization/capability tags.
- ModelLocation: node-specific model availability and source path/root/relative path verification.
- Backend / BackendVersion: system catalog capability and version defaults. Built-ins are system-owned; user changes should happen through cloned/user runtime config, not direct catalog mutation.
- BackendRuntime: reusable runtime template/config: image, default args/env/volumes/devices/health-check template.
- NodeBackendRuntime: node-scoped runtime overrides: image presence/ref, devices/security/runtime parameters, node naming.
- Deployment: service-level saved config: model + runtime + placement + service ports + deployment parameters.
- RunPlan / NodeRunPlan: immutable execution plan generated before a run. Historical plans must not change after Deployment/Runtime edits.
- ModelInstance: concrete run instance: container, Agent task, health/log/error state and troubleshooting details.

## Field Semantics

- `display_name`: UI-facing model/runtime/deployment label.
- `name` / `artifact_name`: stable platform/internal name where required by uniqueness rules.
- `source_path`: real node filesystem path.
- `mount_path`: container path used by runtime args and volume mounts.
- `served_model_name`: optional API model name exposed by OpenAI-compatible endpoints.
- `backend_model_arg`: backend-specific model argument, normally mount_path for local model backends.
- `host_port`: host access port.
- `container_port`: Docker container port.
- `app_port`: process listen port, normally equal to container_port.
- `health_port`: host-side health probe port, normally host_port.
- `api_test_port`: host-side model test port, normally host_port.

## Parameter Precedence

1. Deployment explicit parameters.
2. NodeBackendRuntime overrides.
3. BackendRuntime config.
4. BackendVersion defaults.
5. Backend defaults.
6. System fallback defaults.

## Execution Phases

1. Repository audit: identify editable UI fields, API payloads, handlers, DB columns, RunPlan resolver, Docker spec mapper, and current tests.
2. Backend tests first: add failing tests for runtime naming, model display_name persistence, deployment save-only/start/idempotency/restart, RunPlan immutability, port/parameter precedence, Docker preview/spec semantics, and empty model-test response.
3. Backend implementation: persist missing fields, protect built-ins, enforce run idempotency, add save-only/preview if absent or repair existing endpoints, fix model test validation.
4. Frontend tests first: add/extend tests for runtime/model/deployment UI labels, action buttons, port labels, empty response warning, status panels, and i18n coverage.
5. Frontend implementation: update forms, payloads, list/detail/edit views, action labels, deployment status panel, instance timeline/troubleshooting links.
6. Selected E2E: add llama.cpp UI/API E2E covering custom display/runtime names, save-only, preview, host_port=8005, run, duplicate-run guard, non-empty model test, and artifacts.
7. Documentation: update design/testing docs, open issues, and final fix report with evidence.
8. Final verification: run required Go/Web/script/E2E checks, then commit and push.

## Risk Controls

- Keep built-in catalog immutable unless a local existing endpoint already supports clone semantics.
- Preserve historical RunPlan rows and compare JSON snapshots in tests.
- If real GPU/Docker E2E is unavailable, save exact failure evidence and reproduction command in the report/open issues.
- Do not leave a UI-editable field without a persistence/list/detail verification path.

## Closure Notes

- Backend fixes implemented for runtime clone/name persistence, NodeBackendRuntime display name persistence, model artifact display name persistence, Deployment save-only/update fields, single-active-instance start guard, service port propagation, and empty model-test response handling.
- Frontend fixes implemented for runtime/model/deployment naming, deployment save/preview/run actions, port labels, and empty response failure rendering.
- Selected E2E script added at `scripts/e2e-ui-persistence-runplan-selected.sh`.
- Final verification results are recorded in `ui-persistence-runplan-fix-report.md`.
