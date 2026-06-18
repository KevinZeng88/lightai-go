# LightAI Go Testing Documentation

> Status: CURRENT
> Last updated: 2026-06-19

## Test Infrastructure

### Go unit tests
```bash
go test ./...          # 10 packages, all PASS
go vet ./...           # no errors
```

### Frontend tests
```bash
npm --prefix web run build    # Vite production build
npm --prefix web test         # i18n keys, formatters, boundary UI checks
```

### i18n
- zh-CN: 640 keys, en-US: 640 keys — consistent
- 538 key references in templates — all resolve to strings
- No hardcoded credentials, no dotted key leaks

### E2E Test Scripts

#### Matrix E2E
```
scripts/e2e-model-runtime-wizard-nvidia-matrix.sh        — wrapper; runs all 3 backends
scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh      — llama.cpp default + modified
scripts/e2e-model-runtime-wizard-nvidia-vllm.sh          — vLLM default + modified
scripts/e2e-model-runtime-wizard-nvidia-sglang.sh        — SGLang default + modified
scripts/e2e/lib/model-runtime-common.sh                  — shared API helpers and pipeline
```

#### Failed Instance E2E
```
scripts/e2e-model-runtime-failed-instance-logs.sh
```

Constructs a port-conflict failure to verify:
- Instance state = failed
- container_id preserved
- last_error has structured JSON {failure_reason_code, exit_code, container_id, error}
- current_run_plan_id available from GET /api/v1/model-instances/{id}
- Real logs API: GET /api/v1/node-run-plans/{run_plan_id}/logs
- docker-logs-response.json saved as real API response
- stdout/stderr preview single-line escaped

### Logs API
- Endpoint: `GET /api/v1/node-run-plans/{run_plan_id}/logs`
- run_plan_id obtained from `GET /api/v1/model-instances/{id}` → `current_run_plan_id`
- Web: ModelInstancesPage log button enabled when `current_run_plan_id` exists (any actual_state)

### Evidence
- Matrix: `docs/reports/model-runtime-node-wizard/e2e-matrix-*/`
- Failed instance: `docs/reports/model-runtime-node-wizard/failed-instance-logs-20260619024025/`

### Observability Status (2026-06-19): CLOSED

All gaps closed — see `docs/reports/model-runtime-node-wizard/open-issues-closeout.md`.

## Key Test Design Documents
- `docs/testing/backend-runtime-e2e-matrix-and-param-propagation.md` — E2E matrix specification
- `docs/design/backend-runtime-layered-catalog-design.md` — Backend/Version/Runtime catalog layering
- `docs/design/backend-runtime-runplan-docker.md` — RunPlan/Docker spec design
- `docs/design/model-runtime-node-wizard.md` — Model/runtime wizard product design
