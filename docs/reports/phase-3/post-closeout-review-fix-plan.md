# Post-Closeout Review Fix Plan

> Status: COMPLETE
> Date: 2026-06-19
> Scope: Fix post-closeout review findings P1-001, P1-002, P2-001, P2-002, P2-003, P3-001

## Goal

Move the runtime observability closeout from NOT_PASS to an acceptable verified state by fixing real runtime failure reporting, strict E2E behavior, audit semantics, script JSON validation, evidence handling, and handler-level tests.

## Constraints

- Work on the current branch.
- Do not mask code defects with documentation-only changes.
- Do not run long E2E repeatedly; use unit, handler, and script negative checks first.
- Commit and push only after code, tests, selected E2E, and documentation are verified.

## Execution Phases

### Phase 1: Failure Diagnostics Data Path

Files:
- `internal/agent/runtime/docker.go`
- `internal/agent/runtime/driver.go`
- `cmd/agent/main.go`
- `internal/agent/register/register.go`
- `internal/server/api/agent_handlers.go`
- Tests under `internal/agent/runtime`, `cmd/agent`, and `internal/server/api`

Steps:
1. Add failing tests for post-start-exited and health-check-failed paths retaining `container_id`.
2. Add failing tests for `processStartTask` returning failed `TaskResult` diagnostics.
3. Add handler-level tests for `HandleTaskResult` storing `container_id`, `failure_reason_code`, `exit_code`, and log previews.
4. Implement a minimal diagnostic error type or equivalent that carries container ID, reason code, exit code, stdout/stderr previews.
5. Verify targeted Go tests before moving on.

### Phase 2: Strict Failed-Instance E2E

Files:
- `scripts/e2e-model-runtime-failed-instance-logs.sh`

Steps:
1. Add strict assertion helpers.
2. Make missing `failed` state, empty `last_error`, missing `container_id`, missing run plan, or non-200 logs API fatal.
3. Add a negative self-test mode so missing run plan/logs API conditions exit non-zero.
4. Run `bash -n` and negative script validation.

### Phase 3: Audit Semantics

Files:
- `internal/server/api/agent_handlers.go`
- `internal/server/api/*_test.go`
- Documentation closeout files

Steps:
1. Add audit write on start task success: `instance.start.succeeded`.
2. Add audit write on start task failure: `instance.start.failed`.
3. Include instance, deployment, run plan, node, agent/container, and failure reason details.
4. Add handler tests or query-based validation.

### Phase 4: vLLM Standalone Script JSON

Files:
- `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh`

Steps:
1. Fix modified parameter quoting.
2. Add script-side JSON validation for deployment payloads.
3. Add a lightweight payload-only validation mode to avoid unnecessary long runtime checks.
4. Run the payload validation mode and save artifact.

### Phase 5: Evidence and Documentation Closeout

Files:
- `docs/CURRENT.md`
- `docs/testing/README.md`
- `docs/testing/backend-runtime-e2e-matrix-and-param-propagation.md`
- `docs/reports/model-runtime-node-wizard/open-issues-closeout.md`
- `docs/reports/model-runtime-node-wizard/acceptance-report.md`
- New evidence summary under `docs/reports/model-runtime-node-wizard/`

Steps:
1. Update issue closeout rows for all six review findings with FIXED or verified evidence.
2. Document local/generated artifact policy if full runtime artifacts are not checked in.
3. Save selected E2E summary artifacts and assertion results.
4. Ensure no references point only to missing checked-in paths without a reproduction path.

### Phase 6: Final Verification, Commit, Push

Run:
- `git diff --check`
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- `npm --prefix web run build`
- `npm --prefix web test -- --runInBand`
- `bash -n scripts/*.sh scripts/e2e/lib/*.sh`
- failed-instance logs E2E strict PASS
- vLLM standalone payload/deployment creation validation artifact
- selected matrix/parameter propagation E2E summary artifact
- audit logs query validation for success/failure events

Final gate:
- Update this plan to `Status: COMPLETE`: done.
- Commit with a focused message.
- Push current branch.
- Confirm `git status --short` is clean.

## Completion Evidence

Artifacts:

```text
docs/reports/model-runtime-node-wizard/failed-instance-logs-postfix-20260619032823/
docs/reports/model-runtime-node-wizard/e2e-vllm-standalone-vllm-payload-20260619032852/
docs/reports/model-runtime-node-wizard/e2e-matrix-matrix-postfix-20260619032917/
docs/reports/model-runtime-node-wizard/audit-logs-postfix-20260619033633/
```

All six review findings are recorded as `FIXED` in:

```text
docs/reports/model-runtime-node-wizard/open-issues-closeout.md
```
