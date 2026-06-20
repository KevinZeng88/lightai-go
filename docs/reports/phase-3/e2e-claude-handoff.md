# LightAI Go E2E Claude Handoff

> Status: HANDOFF
> Date: 2026-06-20
> Audience: Next Claude/Codex executor implementing the E2E harness

## Current State

LightAI Go is moving from browser/UI smoke as a main business correctness signal to API-first E2E.

Current facts:

- `internal/server/api/router.go` already has `SetupRoutes(mux, RouterConfig)`.
- Existing Go API tests mostly call handlers directly and manually set `PathValue`.
- Existing NBR probe tests cover important isolated cases, including:
  - `ImageInspect` success is not `missing_image`.
  - inspect not-found maps to `missing_image`.
  - agent unreachable / Docker error / inspect error do not map to `missing_image`.
  - `POST /probe` and `GET /probe` route path values are covered indirectly.
- No current Go test performs a full real-router API workflow chain.
- Existing shell E2E scripts are useful but inconsistent in login, CSRF, cleanup, evidence, naming, and tier boundaries.
- UI static tests are useful for build/i18n/status/component checks.
- Browser smoke is not the primary business correctness gate.

## Completed Audit Outputs

Read these before implementation:

```text
docs/reports/phase-3/e2e-harness-and-api-workflow-strategy.md
docs/reports/phase-3/e2e-implementation-roadmap.md
docs/reports/phase-3/api-first-e2e-review-and-plan.md
docs/reports/phase-3/nbr-image-probe-integration-validation-plan.md
docs/design/image-capability-probe.md
docs/design/node-backend-runtime-image-probe-design.md
```

The strategy document contains:

- Test layering.
- API-first E2E principles.
- Go API Workflow harness design.
- Shell E2E harness design.
- Controlled local real environment agreement.
- Cross-API stability assertions.
- Key workflow breakdown.
- Review table for 22 existing shell E2E scripts.

The roadmap document contains Step 0 through Step 10 with inputs, outputs, files, acceptance criteria, risks, and verification commands.

## Do Not Do

Do not do these unless an explicit future user request changes scope:

- Do not create a new branch.
- Do not continue Phase 5.
- Do not make UI/browser smoke the main acceptance gate.
- Do not introduce Playwright or Cypress.
- Do not add Script Probe.
- Do not add Version Probe.
- Do not add Backend Match catalog.
- Do not add DB migrations unless an explicit future task requires it.
- Do not add new business APIs unless an explicit future task requires it.
- Do not start real model containers during harness/design steps.
- Do not collapse all tests into one huge full test.
- Do not use broad `pkill`, broad `docker rm`, or destructive cleanup.
- Do not commit unrelated working tree changes.

## Development Principles

- Keep changes small and focused.
- Prefer test-only helpers over production changes.
- Use real `SetupRoutes` for router contract and workflow tests.
- Use real login/cookie/CSRF in workflow tests unless a test is explicitly handler-unit scope.
- Use fake Agent for deterministic Go API Workflow tests.
- Keep fake Agent response schema aligned with real Agent.
- Put real Docker/GPU/model work into opt-in shell E2E tiers only.
- If a precise bug is discovered, apply the smallest fix, add/adjust tests, verify, commit, and push only when the user asked for push.
- Every test strategy change must update documentation.
- Preserve user changes in the working tree.

## First Batch of Recommended Implementation Tasks

### Task 1: Go API Workflow Test Harness

Create test-only helper files:

```text
internal/server/api/api_workflow_test_helper_test.go
internal/server/api/fake_agent_test.go
```

Implement:

1. `newWorkflowTestApp(t)`:
   - `db.Open(":memory:")`
   - `Migrate()`
   - `auth.InitBootstrap` with `admin` / `test1234`
   - `auth.NewSessionStore`
   - `auth.AuthHandler`
   - `rbac.Handler`
   - `api.NewAgentHandler`
   - `api.NewResourceHandler`
   - `SetupRoutes`
2. `WorkflowClient`:
   - `LoginAsAdmin(t)`
   - `JSON(t, method, path, body, wantStatus)`
   - automatic cookie and CSRF handling
3. Fake Agent:
   - scenario-driven `/docker-images`
   - scenario-driven `/docker-image-inspect`
   - `/files` and `/model-paths/scan` during the Model Wizard workflow step

Verify:

```bash
go test ./internal/server/api/... -run 'TestRouter|TestWorkflowHarness' -count=1 -v
go test ./internal/server/api/... -count=1
```

Commit after this task if it is independently passing.

### Task 2: NBR Probe Chain API Workflow

Create:

```text
internal/server/api/workflow_nbr_probe_test.go
```

Implement workflow:

1. Login.
2. Register or fixture node.
3. List nodes.
4. List node Docker images.
5. Enable/create NBR.
6. `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe`.
7. `GET /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe`.
8. List NBRs.
9. Assert `probe_results_json` is preserved.
10. Negative path: inspect not-found -> `missing_image`.
11. Negative path: agent unreachable / inspect error is not `missing_image`.
12. Cleanup.

Verify:

```bash
go test ./internal/server/api/... -run 'TestWorkflowNBRProbe' -count=1 -v
go test ./internal/server/api/... -count=1
```

Commit after this task if passing.

### Task 3: BackendRuntime CRUD Chain

Create:

```text
internal/server/api/workflow_backend_runtime_test.go
```

Implement:

1. list backend
2. list backend version
3. list runtime templates
4. clone system runtime/template
5. patch fields
6. get detail
7. list verify
8. delete cleanup

Verify:

```bash
go test ./internal/server/api/... -run 'TestWorkflowBackendRuntime' -count=1 -v
go test ./internal/server/api/... -count=1
```

### Task 4: Model Wizard Chain

Create:

```text
internal/server/api/workflow_model_wizard_test.go
```

Use fake Agent for browse/scan. Step 5 is expected to validate the current API contract:

- `ModelArtifact` is the logical model object.
- `ModelLocation` is the node/path/file-level evidence object.
- Artifact detail exposes locations through `locations[]`.
- Scan metadata/capabilities are stored on `ModelLocation.discovered_metadata_json`.
- Verify location-level `node_id`, path, checksum, size, format, arch, capabilities, discovered metadata, multi-location separation, and cleanup.

Do not refactor ModelArtifact merely because older strategy text mentioned artifact-level `metadata_json`, `capabilities_json`, or `locations_json`. Artifact-level canonical metadata/capabilities is a future product design topic, not part of the current E2E harness work.

### Task 5: Deployment Preflight / RunPlan Chain

Create:

```text
internal/server/api/workflow_deployment_runplan_test.go
```

Verify deployment create, preflight, dry-run, run plan fields, NBR snapshot freeze, cleanup.

### Task 6: Start / Logs / Stop Chain

Create:

```text
internal/server/api/workflow_lifecycle_test.go
```

Use fake Agent task result flow. Verify operation/task/generation/status/logs/diagnostics/audit correlation.

### Task 7: Shell E2E Harness Evaluation

Only after Go workflow slices pass, add shell helpers:

```text
scripts/e2e/lib/env.sh
scripts/e2e/lib/api-client.sh
scripts/e2e/lib/assert.sh
scripts/e2e/lib/resources.sh
scripts/e2e/lib/docker.sh
scripts/e2e/lib/report.sh
scripts/e2e/lib/cleanup.sh
```

Convert one script at a time. Start with:

```text
scripts/e2e-clone-template-parameter-persistence.sh
```

Then:

```text
scripts/e2e-deployment-visibility-selected.sh
scripts/e2e-runtime-config-web-check-flow.sh
```

### Task 8: Real Container Smoke

Run only after API workflows and shell harness are stable:

1. llama.cpp
2. vLLM
3. SGLang

Use existing local environment and preserve evidence on failure.

## Questions To Confirm Before Coding

Ask the user before starting implementation if not already confirmed:

1. Should workflow tests use real login/cookie/CSRF by default?
2. Should all fake-agent workflow tests be included in `go test ./...` immediately?
3. Should shell harness first support only existing-env mode, or also fixture mode?
4. Should shell evidence default to `/tmp` or `docs/reports/...`?
5. Should the currently untracked `docs/reports/phase-3/api-first-e2e-review-and-plan.md` be committed as part of the phase-3 test strategy docs?

## Expected Task Order

Use this order:

1. Implement Go API Workflow test harness.
2. Implement NBR Probe Chain API Workflow.
3. Verify.
4. Commit.
5. Implement BackendRuntime CRUD Chain.
6. Verify.
7. Commit.
8. Implement Model Wizard Chain.
9. Implement Deployment Preflight/RunPlan Chain.
10. Implement Start/Logs/Stop Chain.
11. Evaluate Shell E2E harness.
12. Convert API-only shell scripts first.
13. Convert Real Agent/Docker scripts.
14. Convert Real Model Container scripts.
15. Run vLLM/SGLang/llama.cpp smoke only on the controlled local machine.

## Completion Report Requirements

Every implementation turn must report:

- Root cause or purpose of the change.
- What changed.
- Modified files.
- Verification commands and results.
- Skipped tests and why.
- Whether unresolved problems remain.
- Whether unresolved problems are documented in the formal open-issues document.
- `git status --short`.
