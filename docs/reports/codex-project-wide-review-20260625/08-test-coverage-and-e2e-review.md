# Test Coverage and E2E Review

## Verification results

- `go test ./...` passed.
- `go test ./... -cover` passed.
- `go build ./cmd/server/...` passed.
- `go build ./cmd/agent/...` passed.
- `npm test` passed.
- `npm run build` passed with Vite chunk-size warning.

## Coverage observations

Strong areas:

- `internal/server/runplan`: 70.6%.
- `internal/agent/runtime`: 66.3%.
- `internal/agent/collector`: 58.4%.
- `internal/server/api`: 56.2%.
- `internal/server/agentclient`: 84.8%.

Weak areas:

- `cmd/server`: 0%.
- `internal/server/db`: 0%.
- `internal/server/metrics`: 0%.
- `internal/server/rbac`: 0%.
- `internal/server/auth`: 3.3%.
- `internal/server/authz`: 17.6%.
- frontend Vue component tests are mostly script/static assertions, not browser workflow tests.

## E2E quality concerns

Several E2E scripts still rely on deprecated or client-trusted paths:

- Deployment create with `backend_runtime_id`: `scripts/e2e-matrix-verifier.sh`, `scripts/e2e-dryrun-parameter-matrix-enhanced.sh`, `scripts/e2e-runplan-parameter-source-audit.sh`.
- Client-trusted NBR check with `image_present=true`: multiple scripts including `scripts/e2e-backend-runtime-nvidia-api.sh`, `scripts/e2e-model-runtime-wizard-nvidia-api.sh`.
- `parameters_json` payloads remain in scripts and stored evidence even though current code stores `parameter_values_json`.

This means script presence is not equal to trustworthy current E2E coverage.

## Highest-priority tests to add or repair

1. Contract test: `/nodes/{id}/backend-runtimes/check` must not allow session caller to mark ready from request evidence.
2. Contract test: `/deployments/preflight`, `/deployments/{id}/dry-run`, and `/deployments/{id}/start` produce consistent errors for format mismatch, missing model location, NBR status including `ready_with_warnings`, context overflow, port config, and snapshot source.
3. API-first E2E script using only `node_backend_runtime_id`, `/check-request`, and `parameter_values_json`.
4. Real Docker smoke: llama.cpp GGUF start/stop/logs/cleanup.
5. Negative tenant matrix for every model/runtime/deployment/NBR route.
6. Browser smoke for model wizard, runner config check, deployment preview/start, failed logs.
7. Migration/schema test for fresh DB and no legacy `tenant_id='default'`.
