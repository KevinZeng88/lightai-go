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

## 2026-06-19 Post-Codex Manual Retest Round

### Issues Found After Codex Commit 52603ce

Manual retesting after the Codex round revealed persistent issues despite the previous round claiming ACCEPTABLE_WITH_BLOCKER:

| ID | Issue | Root Cause | Fix |
| -- | ----- | ---------- | --- |
| MRW-UPR-008 | Model artifact edit shows "制品名称" field not saved | Backend PATCH ignores `name` field; frontend showed it as editable | Frontend: `name` field now shows as readonly with hint tag; wizard step 3 clarifies display_name vs internal name with separate fields |
| MRW-UPR-009 | Runtime config name overwritten by template selection | `onWizTemplateSelected()` unconditionally overwrote user-entered `wizConfigName` | Guard: only auto-generate if `wizConfigName` is empty |
| MRW-UPR-010 | Deployment not decoupled from runtime template | `model_deployments` had no config snapshot; preflight read live BackendRuntime values | DB V22: added `config_snapshot_json` to `model_deployments`; capture at create; preflight applies: NBR snapshot > Deployment snapshot > Version snapshot > Live BackendRuntime |
| MRW-UPR-011 | Container/app port defaults display 0 in wizard | Wizard initialized ports to 0; backend resolved correctly but UI was misleading | Fetch BackendVersion `default_container_port` on version selection; pre-fill wizard fields |
| MRW-UPR-012 | No deployment edit entry | Frontend deployment list had no edit button despite existing PATCH API | Added edit button and dialog with fields: display_name, model_artifact_id, backend_runtime_id, host_port, container_port, app_port |

### Changed Files

- `internal/server/db/db.go` — V22 migration: `config_snapshot_json` column on `model_deployments`
- `internal/server/api/deployment_lifecycle_handlers.go` — `buildDeploymentRuntimeSnapshot`, capture at create, return in get/list, `applyDeploymentConfigSnapshot` in preflight
- `internal/server/models/deployment.go` — added `ConfigSnapshotJSON` field
- `web/src/pages/ModelArtifactsPage.vue` — removed name from edit form; wizard display name field and name hint
- `web/src/pages/RunnerConfigsPage.vue` — `onWizTemplateSelected` guards against overwrite
- `web/src/pages/ModelDeploymentsPage.vue` — edit button, edit dialog, port pre-fill from version
- `web/src/locales/zh-CN.ts` — new keys: `modelWizard.displayName`, `modelWizard.nameHint`, `deployments.displayName`, `deployments.editDeployment`, `common.readonly`
- `web/src/locales/en-US.ts` — same new keys
- `internal/server/api/ui_persistence_runplan_test.go` — 3 new tests

### Tests Added

- `TestDeploymentCapturesConfigSnapshotAtCreate` — snapshot non-empty after create, contains source_runtime_id
- `TestDeploymentPatchPortsAndDisplayName` — edit round-trip for ports and display name
- `TestModelArtifactNameFieldNotSavedOnPatch` — confirms name is read-only on PATCH

### Verification Results

| Command | Result |
| ------- | ------ |
| `go test ./...` | PASS (10 packages) |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `npm --prefix web run build` | PASS |
| `npm --prefix web test -- --runInBand` | PASS (668 i18n keys, 556 references) |
| `bash -n scripts/*.sh scripts/e2e/lib/*.sh` | PASS |

### Open Issues

None. All discovered issues are FIXED.
