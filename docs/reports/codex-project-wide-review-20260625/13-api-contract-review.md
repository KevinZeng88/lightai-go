# API Contract Review

## Current live route shape

The current router uses:

- `/api/v1/backend-runtimes`
- `/api/v1/nodes/{id}/backend-runtimes`
- `/api/v1/nodes/{id}/backend-runtimes/enable`
- `/api/v1/nodes/{id}/backend-runtimes/check`
- `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check-request`
- `/api/v1/deployments`
- `/api/v1/deployments/preflight`
- `/api/v1/deployments/{id}/dry-run`
- `/api/v1/deployments/{id}/start`
- `/api/v1/node-run-plans/{id}`
- model roots and file browse APIs under `/api/v1/nodes/{id}/model-roots`, `/files`, `/model-paths/scan`

## Contract strengths

- Create deployment rejects `backend_runtime_id`.
- Preflight rejects `backend_runtime_id`.
- Deployment start fails if `source_node_backend_runtime_id` is absent.
- NBR list returns `deployable`, `warnings`, and `disabled_reason` so frontend need not infer deployability only from status string.

## Contract problems

1. OpenAPI is stale and cannot be treated as contract.
2. Scripts still use old route and field contract.
3. Unknown legacy fields such as `parameters_json` are not clearly rejected in `HandleCreateDeployment`; they are ignored because only `parameter_values_json` is read.
4. `/deployments/preflight` response errors are strings in some paths, while dry-run/start use structured errors.
5. `/deployments/preflight` requires NBR status `ready`, while create/start use `isNBRDeployable` and accept `ready_with_warnings`.
6. `/nodes/{id}/backend-runtimes/check` and `/check-request` overlap but have different evidence trust models.

## Recommended contract decisions

- Declare `/check-request` as the only supported user-triggered NBR readiness endpoint.
- Deprecate or remove `/check`.
- Require `parameter_values_json` for structured parameters; reject `parameters_json` with HTTP 400 to force script cleanup.
- Make preflight use the same deployability helper and `PreflightError` schema as dry-run/start.
- Generate OpenAPI from router/tests or maintain a hand-written contract with sample curl payloads.
