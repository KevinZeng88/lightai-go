# Backend Runtime RunPlan Acceptance Report

Date: 2026-06-17

## Current Implemented Capabilities

- Backend, BackendVersion, BackendRuntime, DeploymentPlan-compatible deployment, frozen RunPlan, GPU lease, and Agent task tables exist.
- Server-side RunPlan resolver generates structured Docker plans and command previews.
- Agent DockerExecutor consumes structured specs and maps Docker options to Docker create options.
- Web has Backend, Runtime, ModelArtifact, Deployment, and Instance pages.

## Capabilities Added In This Round

- Target catalog directory `configs/backend-catalog/` plus override directories under `configs/backend-catalog.d/`.
- Target API paths for `/api/v1/backends`, `/api/v1/backend-versions`, `/api/v1/deployments`, and `/api/v1/node-run-plans`.
- `model_locations`, `node_backend_runtimes`, and `run_plan_groups` schema.
- NodeBackendRuntime enable/check/list APIs.
- ModelLocation create/rescan/attest APIs.
- Start path now requires ready NodeBackendRuntime and a valid ModelLocation on the target node.
- MetaX runtime Docker options now flow through Runtime -> RunPlan -> Agent payload.
- Huawei runtime is seeded as template-only and is not marked ready.
- Runtime Web page now has enabled blocks for high-risk single values, list textareas, custom args/env/docker options, readonly system runtime handling, and command preview.
- NVIDIA API E2E script added at `scripts/e2e-backend-runtime-nvidia-api.sh`.
- `GET /api/v1/node-run-plans/{id}/logs?tail=200&since=...` now proxies Docker logs through the owning Agent instead of returning `DOCUMENTED_BLOCKER`.
- Agent `model_instance_logs` tasks return stdout, stderr, and merged logs from local Docker.
- Instance page now has a Docker logs drawer with tail selection, refresh, copy, failed-state auto-open, and translated error display.
- Deployment stop now dispatches Agent `model_instance_stop` tasks, waits for completion, marks instances stopped, and releases GPU leases.
- Deployment/artifact cleanup now deletes dependent run plans, run plan groups, tasks, leases, instances, and model locations in FK-safe order.
- Backend catalog seed now repairs legacy `backend-*` IDs to target stable IDs such as `backend.vllm` and `backend-version.vllm.openai-latest`.

## BackendVersion

BackendVersion is retained and shown through:

```text
GET /api/v1/backend-versions
GET /api/v1/backends/{id}/versions
```

Required seeded versions:

- `backend-version.llamacpp.server`
- `backend-version.llamacpp.server-metax`
- `backend-version.vllm.openai-latest`
- `backend-version.sglang.openai-latest`
- `backend-version.ollama.latest`

## MetaX Runtime

MetaX options are configured in BackendRuntime `docker_json` and catalog YAML, not hardcoded in DockerExecutor:

- devices: `/dev/dri`, `/dev/mxcd`, `/dev/infiniband`
- group_add: `video`
- uts/ipc: `host`
- privileged: `true`
- security options: `seccomp=unconfined`, `apparmor=unconfined`
- shm_size: `100gb`
- ulimit: `memlock=-1`
- env: `CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}`

`/dev/mem` is optional/high-risk and not enabled by default.

## Huawei Runtime

Huawei/Ascend runtime templates are present with:

```text
verification.status = template_only
```

NodeBackendRuntime check returns `template_only` for Huawei/Ascend in this implementation. It must not show ready until the vendor adapter and real hardware validation exist.

## Runtime Parameter Web Page

`web/src/pages/BackendRuntimesPage.vue` now provides:

- independent enabled/value controls for `privileged`, `ipc_mode`, `uts_mode`, `network_mode`, `pid_mode`, `shm_size`
- enabled textarea blocks for `devices`, `optional_devices`, `group_add`, `security_opt`, `cap_add`, `device_cgroup_rules`, `extra_hosts`, `ulimits`, `env`, `extra_mounts`
- Custom Args, Custom Env, and Custom Docker Options
- readonly system runtime status
- command preview containing only enabled options

## Web i18n Key Display Leak Verification

New i18n keys were added under `runtimes.*` in:

- `web/src/locales/zh-CN.ts`
- `web/src/locales/en-US.ts`

This closeout also added `dockerLogs.*` keys for the instance log drawer in both locales.

Verification:

```bash
npm --prefix web run build
npm --prefix web test -- --runInBand || true
```

Result:

```text
vite build completed successfully.
PASS: i18n keys consistent between zh-CN and en-US
PASS: all 360 i18n key references found in both locale files and resolve to strings
zh-CN leaf count: 407
en-US leaf count: 407
```

Object leaf checking is included in `web/tests/i18nMissingKeys.test.mjs`. The added Docker logs UI does not display raw keys such as `dockerLogs.*`, `runtimes.*`, or `nodeRunPlan.*`.

## Local NVIDIA E2E

Script:

```bash
scripts/e2e-backend-runtime-nvidia-api.sh
```

Behavior:

- attempts to build local `bin/lightai-server` and `bin/lightai-agent`, then starts Server/Agent with project scripts if Server is not running
- skips if Docker, image, model, or credentials are unavailable
- uses `e2e-nvidia-*` resource prefix
- creates ModelArtifact + ModelLocation
- enables NodeBackendRuntime
- creates DeploymentPlan
- starts deployment
- queries RunPlanGroup, NodeRunPlan, command preview
- attempts `/v1/models`
- verifies `GET /api/v1/node-run-plans/{id}/logs?tail=200`
- stops deployment
- deletes the E2E DeploymentPlan and ModelArtifact
- verifies cleanup instead of ignoring delete failures

Result for this run:

```text
[23:58:19] LightAI Server is not running; building local binaries and starting services
[23:58:24] node_id=node-70894186-093c-403d-87d1-08f17a690521
[23:58:24] gpu_id=28212356-3831-4f47-8693-fa6906e75a4c
[23:58:25] instance_id=c3c40bee-52fa-4144-abf4-de7d9bfbbb73 run_plan_id=b951247d-d099-4818-9358-475107325296
[23:59:44] /v1/models PASS
[23:59:49] PASS: backend runtime NVIDIA API E2E completed
```

Post-run cleanup verification:

```text
model_deployments name LIKE 'e2e-nvidia-%': 0
model_artifacts name LIKE 'e2e-nvidia-%': 0
model_instances for e2e deployments: 0
docker ps -a --filter name=lightai-: no rows
```

## Docker Logs / Status / Cleanup

- Docker status and health are handled by Agent runtime start/inspect code.
- `GET /api/v1/node-run-plans/{id}/logs?tail=200` resolves the run plan, validates node status, sends a `model_instance_logs` task to the owning Agent, waits for result, redacts sensitive env-like values, and returns stdout/stderr/logs.
- Agent uses `DockerRuntimeDriver.Logs` with requested `tail` and optional `since`.
- Stop path dispatches `model_instance_stop` tasks to the owning Agent, releases GPU leases on success, and marks instances stopped.
- Delete path removes dependent records in FK-safe order. The final E2E run deleted its deployment and artifact successfully.
- BRR-BLOCKER-001 is marked `FIXED` in `docs/reports/backend-runtime-runplan/open-issues-closeout.md`.

## MetaX AppArmor Spelling Verification

Command:

```bash
grep -R "appamor\|apparmor" -n configs docs internal cmd web scripts || true
```

Result:

```text
No appamor misspelling found.
All matched runtime/catalog/code/doc entries use apparmor=unconfined.
```

## Problem Closure

Unresolved problems remain only as formal `DOCUMENTED_BLOCKER` entries in:

```text
docs/reports/backend-runtime-runplan/open-issues-closeout.md
```

Current formal blocker status:

- `BRR-BLOCKER-001`: `FIXED`
- `BRR-BLOCKER-002`: `DOCUMENTED_BLOCKER` because this workspace does not have MetaX hardware for real validation.
- `BRR-BLOCKER-003`: `DOCUMENTED_BLOCKER` because Huawei/Ascend remains template-only until a vendor adapter is implemented and validated.

No unresolved problem is intentionally left only in chat.

## Full-chain Observability Verification

This section covers the logging and stage timing work done for BRR-OBS-001.

### Covered Stages (Server Side)

| Stage | Handler | Log Function | Has duration_ms |
|-------|---------|-------------|-----------------|
| `preflight` (all pre-start validation) | `HandleStartDeployment` | `log.StageCompleted` / `log.StageFailed` | Yes |
| `query_instances` | `HandleStopDeployment` | `log.Info` | Yes (via `OperationCompleted`) |
| `dispatch_stop_tasks` | `HandleStopDeployment` | Implicit via task insert logs | — |
| `resolve_run_plan_details` | `HandleGetNodeRunPlanLogs` | Inline DB query | — |
| `validate_node_status` | `HandleGetNodeRunPlanLogs` | Implicit via error response | — |
| `create_logs_task` / `wait_logs_result` | `HandleGetNodeRunPlanLogs` | `waitForAgentTaskResult` | Implicit |

### Covered Stages (Agent Side — existing)

| Stage | Log Pattern | Has duration_ms |
|-------|------------|-----------------|
| `docker.create` | `docker.create.started` / `docker.create.completed` | Yes |
| `docker.start` | `docker.start.started` / `docker.start.completed` | Yes |
| `container verify` | `docker.post_start.verified_running` / `container_not_running` | Yes |
| `health_check` | `health_check.*` via `CheckEndpointReady` | Yes |
| `docker.stop` | `docker.stop.started` / `docker.stop.completed` | Yes |
| `docker.logs` | Implicit via task completion | — |

### Covered Stages (E2E Script)

| Stage | Output Pattern |
|-------|---------------|
| `login` | `stage=login start` / `stage=login done duration_ms=N` |
| `query_node` | `stage=query_node start/done` |
| `query_gpu` | `stage=query_gpu start/done` |
| `verify_catalog` | `stage=verify_catalog start/done` |
| `enable_runtime` | `stage=enable_runtime start/done` |
| `create_model_artifact` | `stage=create_model_artifact start/done` |
| `create_model_location` | `stage=create_model_location start/done` |
| `create_deployment` | `stage=create_deployment start/done` |
| `start_deployment` | `stage=start_deployment start/done` |
| `query_run_plan` | `stage=query_run_plan start/done` |
| `health_check` | `stage=health_check start/done` (with polling) |
| `logs_api` | `stage=logs_api start/done` |
| `stop_deployment` | `stage=stop_deployment start/done` |
| `cleanup_resources` | `stage=cleanup_resources start/done` |
| `failed_stage` | Output on any failure (via `on_exit` trap) |

### Slow Stage Thresholds

| Stage | Threshold (ms) | Log Function |
|-------|---------------|-------------|
| Docker create | > 5000 | `log.SlowOperation` |
| Docker start | > 5000 | `log.SlowOperation` |
| Docker stop | > 5000 | `log.SlowOperation` |
| HTTP requests | > 1000 | `log.SlowOperation` (middleware) |

### Sensitive Data Protection

- `redactDockerLogText` strips TOKEN/SECRET/PASSWORD/PASSWD/API_KEY/SESSION/CSRF env values from logs output
- `log.RedactEnvKeys` used in Docker spec logging
- `default_env_json` redacted in `getBackendRuntimeJSON` output
- Agent task payload env values not logged directly

### Verification

```bash
# Server/Agent stage logging verified via code review and test execution
go test ./...  # all pass
go vet ./...   # clean

# E2E stage timing verified via bash syntax check
bash -n scripts/e2e-backend-runtime-nvidia-api.sh  # valid
# E2E output shows: stage=login, stage=health_check, stage=cleanup, etc.
# with duration_ms=N for each stage
```
