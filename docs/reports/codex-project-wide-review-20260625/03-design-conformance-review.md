# Design Conformance Review

## Conforms

- BackendVersion is largely kept as catalog/software capability data. Runtime/vendor/device fields are mostly in BackendRuntime, NodeBackendRuntime, and RunPlan.
- Deployment creation rejects direct `backend_runtime_id` and requires `node_backend_runtime_id`: `HandleCreateDeployment`.
- Deployment start requires `source_node_backend_runtime_id` and fails old deployments without it: `preflightDeployment`.
- `ready_with_warnings` is deployable through `isNBRDeployable` in create/start paths.
- Dry-run and start share `preflightDeployment`, so their final plan path is mostly consistent.
- NBR and deployment snapshots exist: `node_backend_runtimes.config_snapshot_json`, `model_deployments.config_snapshot_json`, `backend_runtimes.version_snapshot_json`.

## Deviations

| Finding | Evidence | Impact |
| --- | --- | --- |
| Old NBR check path trusts client evidence. | `runtime_handlers.go` `HandleCheckNodeBackendRuntime` calls `upsertNodeBackendRuntime(checkOnly=true)`, which accepts request `image_present` and `docker_available`. | A session user with runtime write permission can mark an image ready without Agent Docker inspect evidence. |
| `/deployments/preflight` is not final RunPlan preflight and has different deployability semantics. | `preflight_handlers.go` only checks NBR status, ModelLocation, tenant, and GPU count; it requires `status == "ready"` instead of `isNBRDeployable`. | UI preflight can say can_run while dry-run/start later fail on compatibility, context, mount, Docker spec, or resolver lint; it can also block `ready_with_warnings` even though start accepts it. |
| Legacy compatibility is still active despite project rule. | `runtime_handlers.go` has legacy snapshot rebuild branch; `db.go` has migrations mutating snapshots; `cmd/server/main.go` keeps legacy password env. | Adds branches that future work must reason about; snapshot immutability is not absolute for upgraded DBs. |
| API scripts still use old deployment selectors. | Several scripts use `backend_runtime_id` in deployment create/preflight. | API-first evidence can be stale or fail unexpectedly. |
| OpenAPI documents old model runtime concepts. | `docs/api/openapi.yaml` uses `/runtime-environments`, `/run-templates`, `/model-deployments`. | External clients and future agents get an incorrect contract. |

## Snapshot boundary assessment

The implemented intent is good: BR copies BackendVersion, NBR copies BR, Deployment copies NBR/BR, and RunPlan reads the deployment snapshot. The concern is not absence of snapshots, but residual mutation/rebuild paths and template sync. The acceptance rule should be:

- New NBR/deployment snapshots are immutable unless user explicitly edits them.
- Any migration that mutates snapshots must be treated as a release-time repair with evidence, not a normal runtime behavior.
- Runtime template sync must be explicit, diffed, and tested for running-instance behavior.
