# Current E2E Contract

## Required Payload Fields
- `node_backend_runtime_id` — deployment creation/preflight
- `parameter_values_json` — structured parameter format
- `/nodes/{id}/backend-runtimes/{nbr}/check-request` — server-verified readiness

## Forbidden Fields
- `backend_runtime_id` — use `node_backend_runtime_id`
- `parameters_json` — use `parameter_values_json`
- `image_present` / `docker_available` in request body

## Hardware Skip Standard
- No NVIDIA GPU → SKIP with exit code 0, message "SKIP: no GPU"
- No Docker → SKIP with exit code 0, message "SKIP: docker unavailable"

## Routes
- POST /api/v1/nodes/{id}/backend-runtimes/enable — NBR enable
- POST /api/v1/nodes/{id}/backend-runtimes/{nbr}/check-request — Agent-proxied check
- POST /api/v1/deployments — Create deployment
- POST /api/v1/deployments/preflight — Preflight validation
- POST /api/v1/deployments/{id}/dry-run — RunPlan dry-run
- POST /api/v1/deployments/{id}/start — Start deployment
- POST /api/v1/deployments/{id}/stop — Stop deployment
