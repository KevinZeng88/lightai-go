# Executive Summary

## Overall judgment

LightAI Go is beyond a skeleton and has a real single-node NVIDIA model serving path: model root, model scan, ModelArtifact/ModelLocation, Backend/BackendVersion/BackendRuntime, NBR, dry-run/start, Agent Docker start, logs, stop, cleanup, metrics, tenant/RBAC, and Web console are all represented in code.

The project is not ready for broad feature expansion yet. The highest leverage next step is architecture/test convergence around NBR verification, RunPlan contract, API contract, and stale compatibility paths. The current implementation is most trustworthy for server-side unit-tested RunPlan resolution and fake-agent API workflows. It is least trustworthy where current docs/scripts/old endpoints still exercise deprecated payloads or client-trusted evidence.

## Top risks

1. `POST /api/v1/nodes/{id}/backend-runtimes/check` remains session-accessible and can mark NBR ready using request body `image_present=true,docker_available=true`; evidence: `internal/server/api/runtime_handlers.go` `upsertNodeBackendRuntime(checkOnly=true)`. This violates the explicit project rule that NBR readiness must not trust frontend/client evidence.
2. API-first E2E scripts are inconsistent with the current contract. Several scripts still send deployment `backend_runtime_id` and `parameters_json`, while current handler rejects `backend_runtime_id`; evidence: `scripts/e2e-matrix-verifier.sh`, `scripts/e2e-dryrun-parameter-matrix-enhanced.sh`, `scripts/e2e-model-runtime-wizard-nvidia-api.sh`.
3. `/deployments/preflight` is a light candidate check, not the same final RunPlan boundary used by dry-run/start, and it checks `nbr.Status != "ready"` instead of the deployable helper that accepts `ready_with_warnings`; evidence: `internal/server/api/preflight_handlers.go` does not call `preflightDeployment` or `runplan.Resolve`.
4. Deployment edit UI exposes a runtime selector named `backend_runtime_id` but does not submit NBR changes; evidence: `web/src/pages/ModelDeploymentsPage.vue` `showEdit()` and `doEdit()`.
5. Snapshot boundaries are mostly implemented, but migrations and legacy paths still mutate or rebuild frozen snapshots for old data; evidence: `internal/server/db/db.go` migrations around `config_snapshot_json`, and `runtime_handlers.go` legacy rebuild branch.
6. OpenAPI is stale and documents old `/model-deployments`, `/runtime-environments`, and `/run-templates` paths rather than current `/deployments`, `/backend-runtimes`, NBR, RunPlan, and model-root APIs; evidence: `docs/api/openapi.yaml`.
7. Agent auth is still a shared bearer token with node identity checked in payload/database, not per-node credentials; a leaked token can reach all agent API paths.
8. Security-sensitive Docker options are configurable and previewed, but there is no policy gate around `privileged`, raw `devices`, mounts, `network_mode`, `pid_mode`, `cap_add`, or env injection.
9. Coverage is uneven: `internal/server/runplan` and `internal/agent/runtime` are healthy, but auth/authz/db/main/metrics have low or zero coverage.
10. The repository contains substantial untracked E2E evidence and modified `web/package*.json` files before this review; current status is not a clean baseline.

## Most trustworthy capabilities

- RunPlan resolver unit tests and Docker command preview generation.
- Agent Docker driver abstraction with fake Docker tests and real SDK adapter.
- Tenant-scoped basic nodes/GPUs/model/deployment APIs, with multiple handler tests.
- NBR deployability semantics for `ready` and `ready_with_warnings` in create/start paths.
- Docker failure result propagation, structured last_error, and logs endpoint at handler/test level.

## Least reliable capabilities

- NBR readiness validation through old `/check` endpoint.
- Current API-first E2E harness reliability because scripts mix old and new contracts.
- Multi-node, multi-replica, scheduling, and GPU lease behavior beyond single instance.
- OpenAI-compatible API/gateway, usage, audit billing, API keys: mostly absent or limited to instance test helper.
- MetaX/Huawei real runtime validation.
- OpenAPI and current API contract documentation.

## Priority recommendation

Do not continue broad feature expansion yet. First close the P0/P1 contract and evidence gaps:

1. Remove or hard-block client-trusted NBR check path.
2. Make preflight/dry-run/start share one final RunPlan boundary or explicitly rename preflight to candidate check and add contract tests.
3. Clean E2E scripts so current CI cannot pass through deprecated payloads.
4. Rebuild API contract documentation from live routes/tests.
5. Add real smoke gates for one NVIDIA path plus contract/dry-run matrix for all runtime families.
