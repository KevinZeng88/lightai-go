# Backend Runtime RunPlan Current State Audit

Date: 2026-06-17
Branch: `phase-3-runtime-observability-closeout`

## Scope And Method

This audit compares the current repository with `docs/lightai-backend-runtime-runplan-docker-design.md`.

Commands used before audit:

```bash
pwd
git branch --show-current
git status --short
git diff --stat
rg -n "BackendRuntime|BackendVersion|InferenceBackend|RuntimeEnvironment|RunTemplate|ModelArtifact|ModelLocation|DeploymentPlan|RunPlan|ResolvedRunPlan|NodeBackendRuntime|NodeRunPlan|DockerExecutor|gpu_leases|agent_tasks" internal cmd web configs docs -g '!web/node_modules'
rg --files internal cmd web configs scripts | sort
find web -iname '*i18n*' -o -iname '*locale*'
find web -iname '*test*' | grep -i i18n || true
```

Initial working tree:

```text
?? docs/lightai-backend-runtime-runplan-docker-design.md
```

The only pre-existing untracked file is the target design document supplied for this task. It is included as input and must not be overwritten.

## 1. Current Database Tables

The active migration entrypoint is `internal/server/db/db.go:Migrate`.

Current model runtime tables are created by `migrateV10` and later adjusted by `migrateV11`/`migrateV12`:

| Table | Current Status | File / Function |
| --- | --- | --- |
| `inference_backends` | Exists. Represents current Backend family object. | `internal/server/db/db.go:migrateV10`, `seedBuiltInBackends` |
| `backend_versions` | Exists. Holds backend version defaults, parameter defs, health check, recommended images. | `internal/server/db/db.go:migrateV10`, `seedBuiltInBackends` |
| `backend_runtimes` | Exists. Holds tenant/user runtime config with vendor, image, docker JSON, model mount JSON. | `internal/server/db/db.go:migrateV10` |
| `node_runtime_overrides` | Exists. Node-specific override table, but not exposed as NodeBackendRuntime readiness API. | `internal/server/db/db.go:migrateV10` |
| `model_artifacts` | Exists from old V3 and is preserved. It still stores a single `path` field. | `internal/server/db/db.go:migrateV3`, preserved by `migrateV10`; handlers in `internal/server/api/artifact_handlers.go` |
| `model_locations` | Missing. Target design requires node-scoped ModelLocation records. | Missing |
| `model_deployments` | Exists as current deployment intent object. API path uses `/model-deployments`, not `/deployments`. | `internal/server/db/db.go:migrateV10`; `internal/server/api/deployment_lifecycle_handlers.go` |
| `run_plan_groups` | Missing. Current implementation stores only `resolved_run_plans`. | Missing |
| `node_run_plans` | Missing by name. Current equivalent is `resolved_run_plans` plus `model_instances.current_run_plan_id`. | `internal/server/db/db.go:migrateV10` |
| `resolved_run_plans` | Exists. Stores frozen `plan_json`, `docker_preview`, input hash, plan hash. | `internal/server/db/db.go:migrateV10` |
| `gpu_leases` | Exists. V8 partial unique index prevents concurrent active/reserved lease per GPU. | `internal/server/db/db.go:migrateV8`, `migrateV10` |
| `agent_tasks` | Exists. V11 adds lease/idempotency fields. | `internal/server/db/db.go:migrateV10`, `migrateV11` |

Old tables `runtime_environment_docker_specs`, `runtime_environments`, and `run_templates` are dropped by `migrateV10`.

## 2. Current API Implementation

Implemented current endpoints in `internal/server/api/router.go`:

| Target Concept | Current API | Handler |
| --- | --- | --- |
| Backend list | `GET /api/v1/inference-backends` | `HandleListBackends` |
| Backend detail | `GET /api/v1/inference-backends/{id}` | `HandleGetBackend` |
| BackendVersion list | `GET /api/v1/inference-backends/{id}/versions` | `HandleListBackendVersions` |
| BackendRuntimeTemplate list/detail | `GET /api/v1/backend-runtime-templates`, `GET /api/v1/backend-runtime-templates/{name}` | `HandleListRuntimeTemplates`, `HandleGetRuntimeTemplate` |
| BackendRuntime CRUD | `GET/POST/PATCH/DELETE /api/v1/backend-runtimes...` | `runtime_handlers.go` |
| ModelArtifact CRUD | `GET/POST/PATCH/DELETE /api/v1/model-artifacts...` | `artifact_handlers.go` |
| Deployment CRUD/lifecycle | `GET/POST/PATCH/DELETE/POST start/stop /api/v1/model-deployments...` | `deployment_lifecycle_handlers.go` |
| ModelInstance read | `GET /api/v1/model-instances`, `GET /api/v1/model-instances/{id}` | `deployment_lifecycle_handlers.go` |
| Agent task result | `POST /api/v1/agent/tasks/{id}/result` | `agent_handlers.go` |

Target design endpoints not implemented or not using requested path:

| Requested Endpoint | Current Gap |
| --- | --- |
| `GET /api/v1/backends` | Missing target path; current path is `/inference-backends`. |
| `GET /api/v1/backend-versions` | Missing global list endpoint. |
| `GET /api/v1/nodes/{node_id}/backend-runtimes` | Missing NodeBackendRuntime readiness endpoint. |
| `POST /api/v1/nodes/{node_id}/backend-runtimes/enable` | Missing. |
| `POST /api/v1/nodes/{node_id}/backend-runtimes/check` | Missing. |
| `POST /api/v1/model-artifacts/discover` | Missing. |
| `POST /api/v1/model-artifacts/{id}/locations` | Missing. |
| `POST /api/v1/model-artifacts/{id}/locations/{location_id}/rescan` | Missing. |
| `POST /api/v1/model-artifacts/{id}/locations/{location_id}/attest` | Missing. |
| `/api/v1/deployments...` | Missing target path; current path is `/model-deployments...`. |
| `GET /api/v1/deployments/{id}/run-plan-groups` | Missing; no run plan group table. |
| `GET /api/v1/node-run-plans/{id}` | Missing target path; current plan rows are `resolved_run_plans`. |
| `GET /api/v1/node-run-plans/{id}/command-preview` | Missing target path; preview stored in `resolved_run_plans.docker_preview`. |
| `GET /api/v1/node-run-plans/{id}/logs?tail=200` | Missing. Agent runtime has `Logs`, but API endpoint is absent. |

## 3. Current Backend / BackendVersion / BackendRuntime

Current code uses `InferenceBackend` naming in Go/API but it maps to target `Backend`.

Files:

- `internal/server/models/backend.go`
- `internal/server/api/backend_handlers.go`
- `internal/server/api/runtime_handlers.go`
- `internal/server/db/db.go:seedBuiltInBackends`
- `configs/model-runtime/backends/*.yaml`
- `configs/model-runtime/backend-versions/*/*.yaml`
- `configs/model-runtime/backend-runtime-templates/*.yaml`

Implemented:

- Backend family exists for `vllm`, `sglang`, `llamacpp`.
- BackendVersion exists for vLLM `0.8.5`, vLLM `0.10.0`, SGLang `0.4.6`, SGLang `0.5.0`, llama.cpp `b4817`.
- BackendRuntime exists as user-editable DB table.
- Runtime templates exist as YAML files under `configs/model-runtime/backend-runtime-templates`.

Gaps:

- Target IDs such as `backend-version.vllm.openai-latest` and `runtime.vllm.nvidia-docker` are not used.
- `ollama` backend/version/runtime is missing.
- Target `configs/backend-catalog/` and `configs/backend-catalog.d/` directories are missing.
- `catalog_version`, `checksum`, `managed_by`, `source`, and non-destructive system seed metadata are missing in DB.
- Current RuntimeTemplate API reads YAML file content directly and does not seed runtime templates to DB as system-managed BackendRuntime rows.
- `managed_by=system` readonly behavior is represented as `is_builtin` and `is_editable`, but handler does not prevent editing built-in rows.

## 4. Current ModelArtifact / ModelLocation

Files:

- `internal/server/api/artifact_handlers.go`
- `internal/server/models/artifact.go`
- `web/src/pages/ModelArtifactsPage.vue`

Implemented:

- ModelArtifact CRUD exists.
- Fields include `format`, `task_type`, `architecture`, `quantization`, `estimated_vram_bytes`, `required_gpu_count`, and single `path`.

Gaps:

- No `model_locations` table.
- No node-scoped model root, relative path, absolute path, checksum, match status, verification status, manual attestation fields.
- Resolver uses `ModelArtifact.path` as the host model path and derives `/models/<basename>` directly.
- No Agent scan/rescan/attest flow.

## 5. Current Deployment / RunPlan / DockerSpec

Files:

- `internal/server/api/deployment_lifecycle_handlers.go`
- `internal/server/runplan/types.go`
- `internal/server/runplan/resolver.go`
- `internal/server/runplan/preview.go`
- `internal/server/runplan/*_test.go`

Implemented:

- `model_deployments` represents current deployment intent.
- `resolved_run_plans` freezes `plan_json`, `docker_preview`, `input_hash`, and `plan_hash`.
- Resolver merges Backend, BackendVersion, BackendRuntime, optional NodeRuntimeOverride, ModelArtifact, Deployment, Node, and GPU info.
- Docker preview is generated by the same server runplan package.

Gaps:

- No `RunPlanGroup` or `NodeRunPlan` table names.
- No NodeBackendRuntime readiness gate before resolving.
- NodeRuntimeOverride is not read in `HandleStartDeployment`.
- `GPUVisibleEnvKey` is hardcoded to `CUDA_VISIBLE_DEVICES`.
- `UTSMode` and `GroupAdd` are present in types but not fully assigned into server plan payload.
- MetaX visible-device placeholder `{{vendor_visible_devices}}` is not supported by resolver.
- Huawei/Ascend runtime should be `template_only` and never ready, but no status model exists.

## 6. Current Agent Docker Start Chain

Files:

- `internal/agent/runtime/driver.go`
- `internal/agent/runtime/docker.go`
- `internal/agent/runtime/docker_client.go`
- `internal/agent/runtime/runplan_adapter.go`
- `internal/server/api/agent_handlers.go`

Implemented:

- Agent consumes structured `AgentRunSpec`.
- `DockerRuntimeDriver.Start` builds `ContainerCreateOptions`; it does not query Server DB or re-resolve business objects.
- Docker options supported include `privileged`, `ipc_mode`, `uts_mode`, `network_mode`, `shm_size`, `group_add`, `security_options`, `ulimits`, raw devices, bind mounts, ports, and NVIDIA DeviceRequests.
- Logs can be fetched through `DockerRuntimeDriver.Logs`.

Gaps:

- Server task payload omits `devices`, `group_add`, and some fields that Agent already supports.
- No public API for NodeRunPlan logs.
- `StopDeployment` marks state/release leases but does not enqueue a structured Agent stop task for every running container.

## 7. Current Web Pages And Routes

Routes in `web/src/router/index.ts`:

- `/backends` -> `BackendsPage.vue`
- `/runtimes` -> `BackendRuntimesPage.vue`
- `/models/artifacts` -> `ModelArtifactsPage.vue`
- `/models/deployments` -> `ModelDeploymentsPage.vue`
- `/models/instances` -> `ModelInstancesPage.vue`
- Resource, observability, and system pages are present.

Web API files:

- `web/src/api/backends.ts`
- `web/src/api/runtimes.ts`

Gaps:

- Runtime parameter editor is basic; it edits only display name, image, and vendor.
- No enabled-block UI for `privileged`, `ipc_mode`, `uts_mode`, `network_mode`, `pid_mode`, `shm_size`.
- No textarea enabled-block UI for devices, optional_devices, group_add, security_opt, cap_add, device_cgroup_rules, extra_hosts, ulimits, env, extra_mounts.
- No Custom Args / Custom Env / Custom Docker Options productized area.
- Detail dialog displays raw field names and raw JSON keys.
- Success/error messages in `BackendRuntimesPage.vue` include hardcoded English text.

## 8. Backend Catalog Existence

Current catalog-like configuration exists under:

```text
configs/model-runtime/
  backends/
  backend-versions/
  backend-runtime-templates/
  profiles/
```

Target catalog path is missing:

```text
configs/backend-catalog/
configs/backend-catalog.d/
```

Current seed is hardcoded in `internal/server/db/db.go:seedBuiltInBackends`; it does not read the current YAML files or target catalog files.

## 9. Consistent Parts

- Backend and BackendVersion are separated.
- BackendRuntime exists separately from BackendVersion.
- Server generates frozen RunPlan and stores it in DB.
- Agent Docker executor consumes structured spec and does not re-derive business objects.
- GPU leases and agent task records exist.
- Web already has Backend, Runtime, ModelArtifact, Deployment, Instance pages.
- i18n tests exist: `web/tests/i18nKeys.test.mjs`, `web/tests/i18nMissingKeys.test.mjs`.

## 10. Inconsistent Parts

- API paths do not fully match target `/api/v1/backends`, `/api/v1/deployments`, and `/api/v1/node-run-plans`.
- `InferenceBackend` name is used publicly instead of target `Backend`; this can be preserved internally but should expose target aliases.
- ModelLocation is collapsed into ModelArtifact.path.
- NodeBackendRuntime is represented only as `node_runtime_overrides`, which is not the same as readiness/status.
- Catalog is hardcoded seed + template YAML, not target catalog seed.
- MetaX runtime options are incomplete in seeded/runtime catalog and are not guaranteed to flow into Agent payload.
- Huawei runtime templates are missing.
- Built-in runtime immutability is not enforced by handlers.

## 11. Missing Items

- Target catalog directory and built-in entries for vLLM, SGLang, llama.cpp, Ollama.
- MetaX runtime templates with `/dev/dri`, `/dev/mxcd`, `/dev/infiniband`, `group_add=video`, `uts=host`, `ipc=host`, `privileged=true`, `security_opt`, `shm_size=100gb`, `ulimit memlock=-1`, and visible device env placeholder.
- Huawei/Ascend template-only runtime entries and non-ready status semantics.
- NodeBackendRuntime API and DB status fields.
- ModelLocation DB/API and deployment resolver use of node-specific location.
- Runtime UI enabled blocks and command preview.
- E2E script requested as `scripts/e2e-backend-runtime-nvidia-api.sh`.
- Acceptance report and vendor extension doc.

## 12. Current Risk Items

| ID | Risk | Evidence | Status |
| --- | --- | --- | --- |
| BRR-001 | Runtime readiness is not checked before start. | `HandleStartDeployment` resolves directly from `backend_runtimes`; no NodeBackendRuntime lookup. | To fix |
| BRR-002 | Model path is not node-specific. | `runplan.ArtifactInfo.Path` is sourced from `model_artifacts.path`. | To fix |
| BRR-003 | MetaX visible devices use NVIDIA key. | Resolver hardcodes `CUDA_VISIBLE_DEVICES`. | To fix |
| BRR-004 | Agent supports options that Server payload omits. | `DockerSpec.GroupAdd` and raw devices exist in Agent but are absent from `agentSpec` in start handler. | To fix |
| BRR-005 | Runtime Web UI can leak hardcoded English and raw keys. | `BackendRuntimesPage.vue` uses `Created`, `Saved`, `Failed`, `Delete`. | To fix |

## 13. Old RuntimeEnv / RuntimeTemplate Status

Active code under `internal/` and `web/src/` has moved away from old `RuntimeEnvironment` and `RunTemplate` pages/APIs. Old terms remain in historical design docs, runbooks, and reviews.

Database migration `migrateV10` explicitly drops old tables:

```sql
DROP TABLE IF EXISTS runtime_environment_docker_specs;
DROP TABLE IF EXISTS runtime_environments;
DROP TABLE IF EXISTS run_templates;
```

## 14. Old-To-New Mapping

| Old Concept | Current / Target Mapping |
| --- | --- |
| RuntimeEnvironment.backend_type | Backend / `inference_backends.name` |
| RuntimeEnvironment Docker image/options | BackendRuntime `image_name`, `docker_json` |
| RunTemplate args/env/ports | BackendVersion defaults + BackendRuntime args/env + Deployment parameters |
| ModelArtifact.path | Temporary compatibility field; target is first/default ModelLocation absolute path |
| ModelDeployment | DeploymentPlan-compatible current object |
| ResolvedRunPlan | Target NodeRunPlan-compatible frozen execution record |

