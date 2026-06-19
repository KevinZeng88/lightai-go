# UI Persistence / RunPlan Fix Report

Date: 2026-06-19

## Issue Closure

| ID | Area | Result | Evidence |
| -- | ---- | ------ | -------- |
| 1 | UI editable field persistence | FIXED | Backend tests cover Runtime, NodeBackendRuntime, ModelArtifact, and Deployment saved fields. |
| 2 | Deployment / Instance responsibilities | FIXED | Deployment save-only status is `saved`; ModelInstance remains per-run. UI separates deployment actions from instance troubleshooting. |
| 3 | Run idempotency | FIXED | Server blocks active `pending/starting/provisioning/running/healthy/stopping` deployments with HTTP 409. Failed deployments remain rerunnable. |
| 4 | Runtime clone/name mismatch | FIXED | Clone generates independent `name/display_name` and stores source template name. |
| 5 | Runtime config name input | FIXED | Runtime and NodeBackendRuntime UI/API now carry user-visible names. |
| 6 | Model display name vs path | FIXED | `display_name` is editable and displayed separately from artifact `name/path`; tests confirm path is unchanged. |
| 7 | Deployment save / save-and-run / preview | FIXED | Deployment wizard exposes save config, save and run, and preview. Preview saves a deployment first because backend dry-run is deployment-id based. |
| 8 | Port semantics | FIXED | `host_port/container_port/app_port/health_port/api_test_port` documented; RunPlan accepts host/container/app ports. |
| 9 | Empty model-test response | FIXED | HTTP 2xx with empty chat/completion content returns `empty_model_response` and UI renders failure. Real non-empty inference E2E remains environment-gated; see `MRW-UPR-007` in `open-issues-closeout.md`. |

## Changed Files

See final git diff for the authoritative list. Main areas:

- `internal/server/api/*`
- `internal/server/db/db.go`
- `internal/server/runplan/*`
- `web/src/pages/*`
- `web/src/locales/*`
- `web/tests/runtimeBoundaryUi.test.mjs`
- `scripts/e2e-ui-persistence-runplan-selected.sh`
- `docs/*`

## Tests Added / Updated

- `internal/server/api/ui_persistence_runplan_test.go`
- `internal/server/runplan/resolver_test.go`
- `web/tests/runtimeBoundaryUi.test.mjs`

## E2E Artifacts

Selected script output from final verification:

```text
/tmp/lightai-ui-persistence-runplan-selected-final
```

The script records health, request payloads, model artifact JSON, runtime JSON, deployment JSON, RunPlan preview, start response, RunPlan JSON when available, and repeated-start response.

## Verification Results

| Command | Result |
| ------- | ------ |
| `go test ./internal/server/api ./internal/server/runplan` | PASS |
| `git diff --check` | PASS |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `npm --prefix web test -- --runInBand` | PASS |
| `npm --prefix web run build` | PASS |
| `bash -n scripts/*.sh scripts/e2e/lib/*.sh` | PASS |
| `ARTIFACT_DIR=/tmp/lightai-ui-persistence-runplan-selected-final bash scripts/e2e-ui-persistence-runplan-selected.sh` | PASS |

Local validation server was started with:

```bash
go build -tags web -o bin/lightai-server ./cmd/server
go build -o bin/lightai-agent ./cmd/agent
LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='test1234' ./bin/lightai-server --config configs/server.release.yaml
./bin/lightai-agent --config configs/agent.yaml
```

The server and agent validation sessions were stopped after the selected E2E run.

## Open Issues

One environment-gated verification item remains formally tracked as `DOCUMENTED_BLOCKER` in `docs/reports/model-runtime-node-wizard/open-issues-closeout.md`:

- `MRW-UPR-007`: real non-empty model inference E2E requires a running backend with a loadable model/GPU. The implementation and empty-response guard are covered by unit/UI tests; the selected E2E records deployment/runplan/start/idempotency artifacts but does not claim a real non-empty inference pass.

No problems from this round remain only in chat history.
