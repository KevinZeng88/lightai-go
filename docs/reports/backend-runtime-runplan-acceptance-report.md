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

Verification:

```bash
cd web && node tests/i18nKeys.test.mjs && node tests/i18nMissingKeys.test.mjs
```

Result:

```text
PASS: i18n keys consistent between zh-CN and en-US
PASS: all 348 i18n key references found in both locale files and resolve to strings
```

Object leaf checking is included in `web/tests/i18nMissingKeys.test.mjs`.

## Local NVIDIA E2E

Script:

```bash
scripts/e2e-backend-runtime-nvidia-api.sh
```

Behavior:

- skips if Server is not running
- skips if Docker, image, model, or credentials are unavailable
- uses `e2e-nvidia-*` resource prefix
- creates ModelArtifact + ModelLocation
- enables NodeBackendRuntime
- creates DeploymentPlan
- starts deployment
- queries RunPlanGroup, NodeRunPlan, command preview
- attempts `/v1/models`
- stops deployment

Result for this run:

```text
[22:53:38] SKIP: LightAI Server is not running at http://127.0.0.1:18080
```

This is an environment skip, not a pass. The script itself passed shell syntax validation.

## Docker Logs / Status / Cleanup

- Docker status and health are still handled by Agent runtime start/inspect code.
- Stop path releases GPU leases and cancels non-terminal tasks.
- Server-side Docker logs API is documented as `BRR-BLOCKER-001` in `docs/reports/backend-runtime-runplan/open-issues-closeout.md`.

## Problem Closure

Unresolved problems remain only as formal `DOCUMENTED_BLOCKER` entries in:

```text
docs/reports/backend-runtime-runplan/open-issues-closeout.md
```

No unresolved problem is intentionally left only in chat.
