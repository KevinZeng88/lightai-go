# LightAI Go E2E Implementation Roadmap

> Status: ROADMAP
> Date: 2026-06-20
> Scope: Step-by-step implementation plan for API-first E2E and controllable local shell E2E
> Companion strategy: `docs/reports/phase-3/e2e-harness-and-api-workflow-strategy.md`

## Step 0: Confirm API-First + Controllable Local Shell E2E Principles

**Goal**

Confirm that LightAI's long-term correctness gate is API Workflow E2E first, then controlled local shell E2E for real environment validation.

**Input**

- Current NBR Image Probe Phase 0-4 state.
- Current shell E2E scripts.
- Current Go handler tests.
- Current Web static/component tests.

**Output**

- Agreement that browser/UI smoke is not the main business correctness gate.
- Agreement on tier naming:
  - A: API-only local E2E.
  - B: Real Agent / Docker E2E.
  - C: Real Model Container E2E.

**Files**

- Read: `docs/reports/phase-3/e2e-harness-and-api-workflow-strategy.md`
- Read: `docs/reports/phase-3/api-first-e2e-review-and-plan.md`
- No code changes.

**Acceptance**

- The team approves API-first E2E as the primary direction.
- The team agrees that shell E2E remains valid only as controlled real-environment verification.

**Risk**

- If this is not confirmed, subsequent harness work may optimize for the wrong acceptance layer.

**Business code change**

No.

**New API**

No.

**Real Agent/Docker/GPU**

No.

**Verification commands**

```bash
git status --short
```

## Step 1: Add Go API Workflow Test Harness

**Goal**

Create reusable test infrastructure for real HTTP router API workflow tests.

**Input**

- `internal/server/api/router.go` with `SetupRoutes`.
- Existing handler tests under `internal/server/api/*_test.go`.
- Auth/session/bootstrap code.
- DB migration/seed behavior.

**Output**

- Test-only helper for in-process API app.
- Test client with real login, cookie, and CSRF.
- Fixture helpers for admin, tenant, node, GPU, runtime, artifact, location, and fake Agent.
- No business workflow test yet unless approved as part of Step 3.

**Files**

Likely create:

```text
internal/server/api/api_workflow_test_helper_test.go
internal/server/api/fake_agent_test.go
```

Likely reuse:

```text
internal/server/api/router.go
internal/server/api/runtime_boundary_test.go
internal/server/api/phase3_rbac_test.go
internal/server/auth/session.go
internal/server/auth/bootstrap.go
internal/server/db/db.go
```

**Need real router**

Yes. Use `http.NewServeMux()` and `SetupRoutes(mux, cfg)`.

**Auth / test tenant / test admin**

- Bootstrap admin with `admin` / `test1234`.
- Use `auth.InitBootstrap` with `ForceChangePassword=false`.
- Use `database.DefaultTenantID()` for tenant fixtures.
- Login via `POST /api/v1/auth/login`.
- Store cookie and `csrf_token`.

**Fake Agent design**

- Use `httptest.NewServer`.
- Provide scenario-driven endpoint responses.
- Keep schema aligned with real Agent endpoints:
  - `/docker-images`
  - `/docker-image-inspect`
  - `/files`
  - `/model-paths/scan`
  - lifecycle/task endpoints during the lifecycle workflow step.

**Acceptance**

- A minimal router contract test can login and call `GET /api/v1/nodes`.
- Mutating request without CSRF fails.
- Mutating request with CSRF succeeds where fixture permissions allow it.
- `go test ./internal/server/api/... -count=1` passes.

**Risk**

- Auth bootstrap may require extra setup beyond DB migration.
- Some handlers assume direct context sessions; router tests will expose middleware requirements.
- Fake Agent schema drift can make tests misleading if not kept close to real Agent.

**Business code change**

No expected business code change. If missing constructor/testability issue appears, first prefer test-only helper composition.

**New API**

No.

**Real Agent/Docker/GPU**

No.

**Verification commands**

```bash
go test ./internal/server/api/... -run 'TestRouter|TestWorkflowHarness' -count=1 -v
go test ./internal/server/api/... -count=1
```

## Step 2: Add Shell E2E Harness

**Goal**

Standardize local shell E2E without changing the business product path.

**Input**

- Existing top-level `scripts/e2e*.sh`.
- Existing `scripts/e2e/lib/e2e-assert.sh`.
- Existing `scripts/e2e/lib/model-runtime-common.sh`.

**Output**

Future helper structure:

```text
scripts/e2e/lib/env.sh
scripts/e2e/lib/api-client.sh
scripts/e2e/lib/assert.sh
scripts/e2e/lib/resources.sh
scripts/e2e/lib/docker.sh
scripts/e2e/lib/report.sh
scripts/e2e/lib/cleanup.sh
```

Do not rewrite every script in this step. Add the minimal helper layer and one tiny self-test or one converted low-risk script only after approval.

**Files**

Likely create:

```text
scripts/e2e/lib/env.sh
scripts/e2e/lib/api-client.sh
scripts/e2e/lib/assert.sh
scripts/e2e/lib/resources.sh
scripts/e2e/lib/report.sh
scripts/e2e/lib/cleanup.sh
```

Subsequent conversion candidates:

```text
scripts/e2e-clone-template-parameter-persistence.sh
scripts/e2e-deployment-visibility-selected.sh
scripts/e2e-runtime-config-web-check-flow.sh
```

**Login / cookie / CSRF**

- `e2e_login` logs in once per run.
- Cookie jar is per-run.
- CSRF token is added to POST/PATCH/PUT/DELETE.
- Failed login is always FAIL, not skip.

**Readiness**

- `e2e_wait_server`.
- `e2e_wait_agent`.
- `e2e_require_docker`.
- `e2e_require_gpu`.
- `e2e_require_image`.
- `e2e_require_model_path`.

**Cleanup**

- Use resource IDs captured in `resources.env`.
- No broad `pkill`.
- No broad `docker rm`.
- Fixture mode only kills PIDs it started.
- Success cleans resources.
- Failure preserves evidence by default.

**Evidence**

Every run writes:

```text
summary.json
summary.md
resources.env
environment.txt
requests/
responses/
logs/
failure-reason.txt
```

**Acceptance**

- Helper self-test passes.
- One API-only script can use shared login/API/assert/report helpers.
- Existing scripts are not broken.

**Risk**

- Rewriting too many shell scripts at once can hide regressions.
- Existing scripts use different password defaults (`test1234` and `Commvault!234`); standardization requires explicit migration.
- Existing-env and fixture modes can conflict if not separated.

**Business code change**

No.

**New API**

No.

**Real Agent/Docker/GPU**

No for helper self-test. Later converted tier B/C scripts will require them.

**Verification commands**

```bash
bash scripts/e2e/lib/e2e-assert-selftest.sh
bash -n scripts/e2e/lib/*.sh
```

## Step 3: Implement NBR Probe Chain API Workflow

**Goal**

Implement the first vertical slice of Go API Workflow E2E.

**Input**

- Step 1 harness.
- Fake Agent scenarios for Docker image list and inspect.
- Existing NBR probe handlers.
- NBR design docs:
  - `docs/design/image-capability-probe.md`
  - `docs/design/node-backend-runtime-image-probe-design.md`

**Output**

New workflow test, likely:

```text
internal/server/api/workflow_nbr_probe_test.go
```

**Workflow**

1. Login.
2. Register or fixture a node.
3. List nodes.
4. List node Docker images.
5. Create/update NBR.
6. `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe`.
7. `GET /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe`.
8. List NBRs.
9. GET NBR detail if the API exposes detail through current route set; otherwise list item is the detail source and document that boundary.
10. Assert `probe_results_json`.
11. Negative path for missing image.
12. Negative path for agent/inspect errors not mapping to `missing_image`.
13. Cleanup.

**Acceptance**

- Positive inspect success does not produce `missing_image`.
- Missing image only comes from inspect not-found.
- Agent unreachable / Docker error / inspect error do not produce `missing_image`.
- `probe_results_json` survives POST -> GET -> list/detail.
- `check-request` compatibility still passes.
- Cleanup makes created NBR invisible.

**Risk**

- Current direct handler tests already cover pieces; avoid duplicating all cases instead of workflow coverage.
- NBR detail API shape may be list-only; if so, assert list item mapping and document the API boundary.

**Business code change**

No expected. If a real bug appears, fix minimally with a test.

**New API**

No.

**Real Agent/Docker/GPU**

No.

**Verification commands**

```bash
go test ./internal/server/api/... -run 'TestWorkflowNBRProbe' -count=1 -v
go test ./internal/server/api/... -count=1
```

## Step 4: Implement BackendRuntime CRUD Chain

**Goal**

Verify Runtime Wizard API flow and snapshot/patch preservation.

**Input**

- Step 1 harness.
- Existing backend/runtime catalog seed.
- Existing clone and patch handlers.

**Output**

```text
internal/server/api/workflow_backend_runtime_test.go
```

**Workflow**

1. List backend.
2. List backend version.
3. List system runtime template.
4. Clone system template to user runtime.
5. PATCH image/env/ports/volumes/devices/extra_args and supported Docker/security fields.
6. GET detail verify.
7. List verify.
8. DELETE cleanup.

**Acceptance**

- PATCH does not lose unmodified fields.
- `config_snapshot_json` and Docker/config fields remain intact.
- System templates remain read-only.
- User clone is not confused with system runtime.
- Deleted runtime is not visible.

**Risk**

- Current catalog IDs may differ from older shell scripts. Use discovered IDs unless testing a specific known system ID.

**Business code change**

No expected.

**New API**

No.

**Real Agent/Docker/GPU**

No.

**Verification commands**

```bash
go test ./internal/server/api/... -run 'TestWorkflowBackendRuntime' -count=1 -v
go test ./internal/server/api/... -count=1
```

## Step 5: Implement Model Wizard Chain

**Goal**

Verify ModelArtifact and ModelLocation API workflow using the current field contract.

Current contract clarification:

- `ModelArtifact` is the logical model object.
- `ModelLocation` is the node/path/file-level evidence object.
- The API does not currently expose artifact-level `metadata_json`, `capabilities_json`, or `locations_json`; this is not a bug.
- Scan metadata/capabilities are currently stored on `ModelLocation.discovered_metadata_json`.
- Artifact detail exposes ModelLocation records through `locations[]`.
- API workflow tests should assert location-level scan metadata preservation, not force artifact-level canonical metadata.

**Input**

- Step 1 harness.
- Fake Agent for `/files` and `/model-paths/scan`.
- Current model root policy.

**Output**

```text
internal/server/api/workflow_model_wizard_test.go
```

**Workflow**

1. List nodes.
2. Add model root.
3. Browse node files through fake Agent.
4. Scan model path through fake Agent.
5. Create `ModelArtifact`.
6. Add `ModelLocation`.
7. List artifacts.
8. GET detail.
9. Cleanup.

**Acceptance**

- `ModelLocation.discovered_metadata_json` preserves scan metadata/capabilities with object/array types.
- Artifact detail returns `locations[]` with the created locations.
- Each location preserves `node_id`, path, checksum, size, format, arch, capabilities, and discovered metadata.
- Multiple locations do not mix node IDs.
- Cleanup removes created artifact/location/root.
- No artifact-level canonical metadata/capabilities redesign is required.

If future requirements need model-level filtering, model-level capability, or cross-location consistency checks, add a separate design for artifact-level canonical metadata/capabilities. That is outside the Step 5/6 E2E harness scope.

**Risk**

- File browse/scan handlers are less covered today; workflow tests may expose missing unit coverage.

**Business code change**

No expected. Fix discovered bugs minimally.

**New API**

No.

**Real Agent/Docker/GPU**

No.

**Verification commands**

```bash
go test ./internal/server/api/... -run 'TestWorkflowModelWizard' -count=1 -v
go test ./internal/server/api/... -count=1
```

## Step 6: Implement Deployment Preflight / RunPlan Chain

**Goal**

Verify deployment save, preflight, dry-run, and run plan JSON/preview preservation without real container start.

**Input**

- BackendRuntime workflow fixture or reusable fixture helper.
- ModelArtifact/Location workflow fixture.
- NBR fixture.

**Output**

```text
internal/server/api/workflow_deployment_runplan_test.go
```

**Workflow**

1. Create/select model artifact and location.
2. Create/select NBR.
3. Create deployment.
4. Preflight.
5. Dry-run / runplan preview.
6. Verify `run_plan_json`.
7. Cleanup.

**Acceptance**

- RunPlan includes image, model path, ports, volumes, devices, env, health check, command, and args.
- NBR snapshot is frozen.
- List/detail deployment fields are consistent.
- Cleanup hides deployment.

**Risk**

- Existing resolver tests cover pure logic; this workflow should avoid becoming another exhaustive resolver matrix.

**Business code change**

No expected.

**New API**

No.

**Real Agent/Docker/GPU**

No.

**Verification commands**

```bash
go test ./internal/server/api/... -run 'TestWorkflowDeploymentRunPlan' -count=1 -v
go test ./internal/server/api/... -count=1
```

## Step 7: Implement Start / Logs / Stop Chain

**Goal**

Verify lifecycle control plane behavior through fake Agent task claim/result and logs path.

**Input**

- Step 6 deployment fixture.
- Fake Agent/task result helper.
- Existing lifecycle handlers and task result handlers.

**Output**

```text
internal/server/api/workflow_lifecycle_test.go
```

**Workflow**

1. Start deployment.
2. Claim or inspect created task.
3. Fake Agent returns success.
4. Verify status.
5. Fetch logs.
6. Stop.
7. Fake Agent returns stop success.
8. Failure path: fake Agent returns start failure.
9. Cleanup.

**Acceptance**

- `operation_id`, generation, task ID, run plan ID, instance state, logs, diagnostics, and audit records remain correlated.
- Failed state preserves diagnostics.
- Logs API can be called for failed instance when run plan exists.
- Cleanup removes resources.

**Risk**

- Full task claim behavior may require lower-level DB assertions if current Agent polling API is not exposed in a workflow-friendly way.

**Business code change**

No expected.

**New API**

No.

**Real Agent/Docker/GPU**

No.

**Verification commands**

```bash
go test ./internal/server/api/... -run 'TestWorkflowLifecycle' -count=1 -v
go test ./internal/server/api/... -count=1
```

## Step 8: Reorganize Existing Shell E2E

**Goal**

Convert the existing 22 scripts into tiered, helper-based shell E2E without changing their business intent.

**Input**

- Step 2 shell harness.
- Script review table in strategy document.

**Output**

- Tier A scripts use API client/assert/report helpers.
- Tier B scripts use readiness and Docker helpers.
- Tier C scripts use real container helpers and failure evidence.
- Mixed scripts are split or clearly marked.

**Files**

Candidates for first conversion:

```text
scripts/e2e-clone-template-parameter-persistence.sh
scripts/e2e-deployment-visibility-selected.sh
scripts/e2e-runtime-config-web-check-flow.sh
```

**Acceptance**

- Converted scripts pass `bash -n`.
- Converted API-only script runs against existing server.
- Evidence output has `summary.json`, `resources.env`, and saved responses.
- Existing unconverted scripts still run as before.

**Risk**

- Touching many shell scripts at once will create noise. Convert one script at a time.

**Business code change**

No.

**New API**

No.

**Real Agent/Docker/GPU**

Depends on converted script:

- Tier A: no.
- Tier B: Agent/Docker yes.
- Tier C: Agent/Docker/GPU/model yes.

**Verification commands**

```bash
bash -n scripts/e2e*.sh scripts/e2e/lib/*.sh scripts/smoke-model-backends.sh scripts/verify-local.sh
bash scripts/e2e/lib/e2e-assert-selftest.sh
```

For a converted API-only script:

```bash
LIGHTAI_E2E_PASSWORD=test1234 bash scripts/e2e-clone-template-parameter-persistence.sh
```

## Step 9: Run Local Real vLLM / SGLang / llama.cpp Smoke

**Goal**

Validate the product path on the known local NVIDIA/Docker/model environment.

**Input**

- Tier C scripts after harness standardization.
- Local images:
  - `vllm/vllm-openai:latest`
  - `lmsysorg/sglang:latest`
  - `ghcr.io/ggml-org/llama.cpp:server-cuda13`
- Local models.

**Output**

- Evidence directory with one run per backend.
- Logs, run plans, Docker specs, `/v1/models`, optional chat responses.
- Cleanup on success; preserved evidence on failure.

**Acceptance**

- llama.cpp starts, `/v1/models` succeeds, logs fetch succeeds, stop succeeds.
- vLLM starts, `/v1/models` succeeds, logs fetch succeeds, stop succeeds.
- SGLang starts, health or `/v1/models` succeeds, logs fetch succeeds, stop succeeds.

**Risk**

- Image versions may change.
- GPU memory pressure may cause OOM.
- Ports may be occupied.
- Model load time may exceed short timeout.

**Business code change**

No expected. If product-path bug appears, fix minimally after recording evidence.

**New API**

No.

**Real Agent/Docker/GPU**

Yes: real Agent, Docker, GPU, and model files are required.

**Verification commands**

```bash
LIGHTAI_E2E_PASSWORD=test1234 bash scripts/e2e-real-smoke-all-three.sh
```

Or backend-specific after conversion:

```bash
LIGHTAI_E2E_PASSWORD=test1234 bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
LIGHTAI_E2E_PASSWORD=test1234 bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
LIGHTAI_E2E_PASSWORD=test1234 bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
```

## Step 10: Consider CI

**Goal**

Add only deterministic tests to CI.

**Input**

- Passing Go API Workflow E2E with fake Agent.
- Passing UI static/component tests.
- Passing shell helper self-tests.

**Output**

CI candidate set:

```bash
go test ./...
go vet ./...
npm --prefix web run build
npm --prefix web test
bash scripts/e2e/lib/e2e-assert-selftest.sh
bash -n scripts/e2e*.sh scripts/e2e/lib/*.sh
```

Do not add real Docker/GPU/model container smoke to general CI unless a dedicated GPU runner exists.

**Acceptance**

- CI stays deterministic.
- No dependency on local model files.
- No dependency on Docker daemon for default CI path.

**Risk**

- Adding too much to CI will make it flaky and slow.

**Business code change**

No.

**New API**

No.

**Real Agent/Docker/GPU**

No for default CI.

**Verification commands**

```bash
go test ./...
go vet ./...
npm --prefix web run build
npm --prefix web test
```

## Recommended Execution Order for the Next Developer

1. Implement Step 1 harness only.
2. Implement Step 3 NBR Probe Chain API Workflow.
3. Run `go test ./internal/server/api/... -count=1`.
4. Commit the harness + NBR workflow.
5. Implement Step 4 BackendRuntime CRUD Chain.
6. Implement Step 5 Model Wizard Chain.
7. Implement Step 6 Deployment Preflight/RunPlan Chain.
8. Implement Step 7 Lifecycle Chain.
9. Then start Step 2/8 shell harness conversion.
10. Run Step 9 real container smoke only after API workflows pass.

## Execution Update: 2026-06-20

The roadmap has been executed through Step 9 with the following status:

| Step | Status | Output |
| --- | --- | --- |
| Step 1 | DONE | `internal/server/api/api_workflow_test_helper_test.go`, `internal/server/api/fake_agent_test.go` |
| Step 3 | DONE | `internal/server/api/workflow_nbr_probe_test.go` |
| Step 4 | DONE | `internal/server/api/workflow_backend_runtime_test.go` |
| Step 5 | DONE | `internal/server/api/workflow_model_wizard_test.go` |
| Step 6 | DONE | `internal/server/api/workflow_deployment_runplan_test.go` |
| Step 7 | DONE | `internal/server/api/workflow_lifecycle_test.go` |
| Step 2 | DONE, minimal harness | `scripts/e2e/lib/env.sh`, `api-client.sh`, `assert.sh`, `resources.sh`, `docker.sh`, `report.sh`, `cleanup.sh` |
| Step 8 | DONE, first batch | `e2e-clone-template-parameter-persistence.sh`, `e2e-deployment-visibility-selected.sh`, `e2e-runtime-config-web-check-flow.sh` |
| Step 9 | PARTIAL WITH FORMAL BLOCKERS | llama.cpp PASS; vLLM and SGLang reached real container start but failed due current Docker/GPU runtime compatibility |

Step 8 verification results:

```bash
bash -n scripts/e2e*.sh scripts/e2e/lib/*.sh scripts/smoke-model-backends.sh scripts/verify-local.sh
LIGHTAI_E2E_PASSWORD=test1234 bash scripts/e2e-clone-template-parameter-persistence.sh
LIGHTAI_E2E_PASSWORD=test1234 bash scripts/e2e-deployment-visibility-selected.sh
LIGHTAI_E2E_PASSWORD=test1234 bash scripts/e2e-runtime-config-web-check-flow.sh
```

The three converted scripts pass in the controlled local environment after setting the admin password to `test1234` and running real server/agent sessions.

Step 9 verification results:

```bash
LIGHTAI_E2E_PASSWORD=test1234 timeout 420s bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
LIGHTAI_E2E_PASSWORD=test1234 timeout 600s bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
LIGHTAI_E2E_PASSWORD=test1234 timeout 600s bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
```

Outcome:

- llama.cpp: PASS.
- vLLM: DOCUMENTED_BLOCKER in `docs/reports/phase-3/open-issues-closeout.md`.
- SGLang: DOCUMENTED_BLOCKER in `docs/reports/phase-3/open-issues-closeout.md`.

Implementation notes:

- Real server/agent were held by foreground tool sessions because the execution environment reaped `nohup` children started through `scripts/start-server.sh` and `scripts/start-agent.sh`.
- `bin/lightai-server` and `bin/lightai-agent` are local ignored build artifacts and are not committed.
- Admin password in the existing DB was reset with `scripts/reset-password.sh --password test1234` for local verification.
- A real bug was fixed: Agent `/docker-image-inspect` now preserves Docker stderr in error responses, allowing server-side `missing_image` mapping to distinguish Docker not-found from generic inspect errors.
