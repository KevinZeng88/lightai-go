# 05 - Implementation Review Findings

> Status: CURRENT
> Scope: Web AI presentation-only implementation review
> Date: 2026-06-21

## 1. Current Page, Route, API, And i18n State

Current Web AI routes are:

| Area | Route | Page | Current entry |
| --- | --- | --- | --- |
| Model library | `/models/artifacts` | `web/src/pages/ModelArtifactsPage.vue` | Model menu |
| Deployments | `/models/deployments` | `web/src/pages/ModelDeploymentsPage.vue` | Model menu |
| Instances | `/models/instances` | `web/src/pages/ModelInstancesPage.vue` | Model menu |
| Node runtime configs | `/runner-configs` | `web/src/pages/RunnerConfigsPage.vue` | Runtime menu |
| Inference backend catalog | `/backends` | `web/src/pages/BackendsPage.vue` | Runtime menu |
| Runtime templates | `/runtimes` | `web/src/pages/BackendRuntimesPage.vue` | Runtime menu |

Current API usage:

- Model library uses `/api/v1/model-artifacts`, `/api/v1/model-artifacts/{id}`, `/api/v1/model-artifacts/{id}/locations`, and `/api/v1/nodes/{node_id}/model-paths/scan`.
- Deployment flow uses `/api/v1/backends`, `/api/v1/backends/{id}/versions`, `/api/v1/backend-runtimes`, `/api/v1/nodes/{id}/backend-runtimes`, `/api/v1/deployments`, `/api/v1/deployments/preflight`, `/api/v1/deployments/{id}/dry-run`, `/api/v1/deployments/{id}/start`, and `/api/v1/deployments/{id}/stop`.
- Node runtime config flow uses `/api/v1/nodes/{id}/backend-runtimes`, `/api/v1/nodes/{id}/backend-runtimes/enable`, `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}`, and `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check-request`.
- Instance flow uses `/api/v1/model-instances`, `/api/v1/model-instances/{id}/test`, `/api/v1/deployments/{id}/stop`, and `/api/v1/node-run-plans/{id}/logs`.

Current i18n state:

- `zh-CN.ts` and `en-US.ts` already include most Web AI page keys.
- Several visible labels are hardcoded in English, for example GGUF/HuggingFace metadata headings, `Head count`, `Quantization`, `View raw JSON`, deployment dry-run JSON titles, and raw instance detail keys.
- The existing `i18nMissingKeys` test checks referenced keys and object-valued keys, but it does not yet enforce the new Web AI workflow key set or page-level leakage patterns.

## 2. Existing Field Support

### ModelArtifact / ModelLocation

Available fields include name, display name, path, format, task type, architecture, size label, quantization, default context length, estimated VRAM, required GPU count, and per-location `discovered_metadata_json`.

There is no dedicated persisted model capabilities field in `model_artifacts` or `model_locations`. Capability display can be inferred from:

- artifact `task_type`;
- artifact name/display name/path;
- location `discovered_metadata_json`, including architecture/model type/tokenizer metadata when present.

### Backend / BackendVersion / BackendRuntime

Backend versions already expose `capabilities_json`, `default_endpoints_json`, health check, docker options, args schema, env schema, and model mount JSON. Runtime templates expose image, docker JSON, args override, default env, entrypoint override, health check override, and usage references.

These objects are low-frequency configuration objects. They can be moved under a configuration menu without deleting functionality.

### NodeBackendRuntime

NodeBackendRuntime exposes display name, node id, backend runtime id, image ref, status, status reason, `config_snapshot_json`, and `probe_results_json`.

`PATCH /api/v1/nodes/{node_id}/backend-runtimes/{nbr_id}` accepts `display_name`, `image_ref`, and `config_snapshot_json`. Editing `config_snapshot_json` invalidates ready status and marks the config as `needs_check`.

Therefore NBR can support structured editing of existing snapshot fields without schema changes, as long as the page writes the same `config_snapshot_json` field.

### ModelDeployment

Deployment stores `placement_json`, `service_json`, `parameters_json`, `env_overrides_json`, `config_snapshot_json`, and `source_node_backend_runtime_id`.

Supported presentation-level changes:

- show model/runtime/node/image/status/endpoint/error context by joining data already fetched in the frontend;
- show deployment-level existing JSON fields as structured sections;
- save existing `service_json`, `parameters_json`, and `env_overrides_json`.

Not supported without schema/API changes:

- first-class deployment extra volume schema;
- first-class deployment extra port list beyond current service port fields;
- first-class endpoint alias/test profile schema.

### ModelInstance

Instance list/detail exposes id, deployment id, node id, current run plan id, state, container id, endpoint URL, host/container port, last error, and timestamps. Logs remain accessible through current run plan id.

The page can default-filter `stopped` in the frontend while keeping failed/exited rows available for diagnostics.

## 3. Requirements Feasible Without Data Structure Changes

Feasible in this round:

- Reorganize navigation around model workflow and move backend/runtime template entries under configuration.
- Rename menu entries and page titles through i18n.
- Add model capability badges inferred from existing metadata/name/task type.
- Add test recommendation display and a selectable Chat/Completion test mode in the instance test dialog.
- Keep Qwen/Instruct/Chat models defaulted to Chat Completion by frontend inference.
- Improve test failure messages with endpoint/status/error summary already returned by the API.
- Replace raw instance detail key/value dump with Chinese/productized sections.
- Default-hide stopped instances and add a show-stopped filter.
- Replace NBR JSON-primary edit with structured editors that still save the existing `config_snapshot_json`.
- Show RunPlan/dry-run summaries before raw JSON.
- Add frontend tests for capability inference and Web AI workflow keys/leakage checks.

## 4. Requirements Recorded For Follow-Up

The following are not implemented in this round because they require new persisted fields, API contracts, or backend summary endpoints:

| ID | Requirement | Reason |
| --- | --- | --- |
| WEB-AI-FU-001 | Persisted model capability override/checklist | No current persisted `capabilities` field or model capability API exists for ModelArtifact. |
| WEB-AI-FU-002 | First-class deployment-level extra volumes | Current deployment API has JSON fields but no stable typed extra volume contract. |
| WEB-AI-FU-003 | First-class deployment-level extra port mappings beyond service port | Current `service_json` supports host/container/app/health/test port style fields, not a typed port list. |
| WEB-AI-FU-004 | Endpoint alias / served model alias persisted at deployment level | Existing fields can carry `parameters_json`, but no explicit alias schema/API is defined. |
| WEB-AI-FU-005 | Backend-provided deployment list summary joins | Current list API returns IDs and JSON; frontend can enrich, but a server-side summary DTO would be cleaner later. |

These items must not be closed as implemented by UI-only presentation.

## 5. Implementation Plan

1. Navigation and naming:
   - make “模型运行” the main workflow group;
   - expose model library, runtime config, deployments, instances, and test/diagnostics in that group;
   - move inference backends and runtime templates under configuration/advanced configuration.

2. Capability and test behavior:
   - add a frontend capability inference helper using model name/task type/metadata;
   - show capability badges in model list/detail and wizard scan results;
   - add test mode selection and clear endpoint-aware errors;
   - ensure `Qwen3-0.6B-Instruct-2512` defaults to Chat Completion.

3. NBR presentation:
   - parse `config_snapshot_json` into image/command/env/volumes/ports/devices/security/health sections;
   - keep JSON in an advanced collapsed area;
   - save existing `config_snapshot_json` only.

4. Deployment presentation:
   - enrich list with frontend-resolved model/runtime/NBR/image/node/endpoint/recent error where available;
   - show dry-run/RunPlan summary before raw JSON;
   - expose existing service/env/parameters JSON fields as existing override sections.

5. Instance presentation:
   - default-hide stopped rows;
   - show failed/exited for diagnostics;
   - replace raw detail dump with basic/running/resource/test/logs/diagnostic sections.

6. Documentation and verification:
   - add closeout and formal open-issues closeout;
   - run Go tests, Go vet, frontend build/test, shell syntax checks, diff check, and git status.

## 6. Test Plan

Frontend:

- add a model capability inference test covering Qwen3 Instruct default Chat Completion;
- extend i18n/workflow leakage tests for new route/menu/page keys and raw JSON placement;
- run `npm --prefix web build`;
- run `npm --prefix web test`.

Go/backend:

- run `gofmt -w cmd/ internal/`;
- run `go test ./internal/server/api/...`;
- run `go test ./internal/server/runplan/...`;
- run `go vet ./...`.

Shell/E2E syntax:

- run `bash -n scripts/e2e/*.sh scripts/e2e/lib/*.sh`.

Repository hygiene:

- run `git diff --check`;
- run `git status --short`.

## 7. Schema And Migration Boundary

This implementation plan explicitly does not modify database schema, does not add migrations, does not add new persisted data structures, and does not change Backend, BackendVersion, BackendRuntime, NodeBackendRuntime, ModelDeployment, or ModelInstance core semantics.
