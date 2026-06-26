# Architecture Cleanliness Review

## Findings

1. Legacy deployment payloads remain in scripts and reports.
   Evidence: `scripts/e2e-matrix-verifier.sh`, `scripts/e2e-dryrun-parameter-matrix-enhanced.sh`, `scripts/e2e-inference-parser-llamacpp.sh`, and older evidence files send `backend_runtime_id` and `parameters_json`.
   Impact: old scripts can no longer validate current code and may mislead future work.

2. Old model types remain in code.
   Evidence: `internal/server/models/runtime.go` defines `NodeRuntimeOverride`, with fields like `EnvJSON`, `DockerOverrideJSON`, `ModelRootHostPath`.
   Impact: duplicate concepts confuse the boundary between NBR, node override, and RunPlan.

3. API route naming has old and new tracks.
   Evidence: current router exposes `/deployments`, `/backend-runtimes`, NBR routes; OpenAPI exposes `/model-deployments`, `/runtime-environments`, `/run-templates`.
   Impact: docs and clients can target wrong routes.

4. Repeated schema creation paths exist.
   Evidence: `db.Migrate()` creates `gpu_devices`; `NewResourceHandler()` also executes `CREATE TABLE IF NOT EXISTS gpu_devices` with `tenant_id DEFAULT 'default'`.
   Impact: nonstandard initialization can create v0.1.9-incompatible tenant defaults.

5. Frontend has concept drift in deployment edit.
   Evidence: `ModelDeploymentsPage.vue` edit form has `backend_runtime_id`, but `doEdit()` does not submit it; selected value can be a NBR ID or BR ID.
   Impact: users can believe runtime changes are saved when they are ignored.

6. Template fallback path remains.
   Evidence: `HandleCreateBackendRuntimeFromTemplate` comments mention backward compatibility; `resolveTemplatePath()` has fallback behavior.
   Impact: conflicts with the stated principle that old config/template paths should be removed rather than preserved.

## Cleanup priority

1. Remove or disable stale E2E scripts that cannot pass against current APIs.
2. Delete unused models/types after confirming no imports.
3. Update OpenAPI or mark it archived/reference.
4. Remove duplicate `CREATE TABLE` paths from handlers.
5. Rename frontend state fields to `node_backend_runtime_id` where they represent NBR.
