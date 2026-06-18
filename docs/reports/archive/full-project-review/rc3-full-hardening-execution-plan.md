> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# RC3 Full Hardening Execution Plan

Generated: 2026-06-17

## 1. Purpose

This document is the authoritative execution plan for **LightAI Go RC3 Full Hardening Closure**.

Claude must follow this document before changing code. The objective is to close all review findings from `docs/reports/full-project-review/issue-register.md`, plus user-added findings REVIEW-026 through REVIEW-030, by fixing code/config/schema/scripts/Web/docs/tests and proving each item through explicit verification.

This is not another audit. This is a full implementation and verification closure.

## 2. Non-Negotiable Rules

1. All requirements must be represented in:
   - `rc3-full-hardening-execution-plan.md`
   - `rc3-clean-baseline-scope.md`
   - `rc3-acceptance-criteria.md`
   - `rc3-verification-matrix.md`
   - `rc3-issue-closeout-register.md`
   - `web-workflow-acceptance-checklist.md`
   - `rc3-final-closeout-report.md`
2. No issue may be closed without:
   - Required action
   - Acceptance criteria
   - Verification command or scenario
   - Actual result
   - Evidence
   - Commit reference, except `Not Reproducible` or `Blocked - External Hardware` with evidence
3. Claude has highest permissions and Docker is available on the machine.
4. Docker E2E, start/stop scripts, release install, patch apply, patch rollback, Web tests, Go tests, shell syntax, and logging verification must run.
5. Destructive validation must use disposable paths or test containers, not user runtime data.
6. The only permitted final statuses are:
   - `Fixed`
   - `Not Reproducible`
   - `Blocked - External Hardware`
   - `Blocked - Explicit Product Decision`
7. Final statuses must not include:
   - `Open`
   - `In Progress`
   - `Not Verified`
   - `Deferred`
   - `Later`
   - `Accepted Risk`
8. P0/P1/P2 are ordering labels only. Every issue must be resolved or explicitly blocked by the allowed final statuses.
9. If new problems are discovered, create REVIEW-031+ in `rc3-issue-closeout-register.md` and add corresponding rows to `rc3-verification-matrix.md`.
10. Do not stop for user confirmation unless the action would affect files outside the project or non-disposable user data.

## 3. Project Paths

Project root:

```text
/home/kzeng/projects/ai-platform-study/lightai-go
```

Full project review directory:

```text
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/full-project-review
```

Reference repositories:

```text
/home/kzeng/projects/ai-platform-study/gpustack-reference
/home/kzeng/projects/ai-platform-study/gpustack-ui-reference
```

Disposable validation directories:

```text
/tmp/lightai-go-rc3-e2e
/tmp/lightai-go-rc3-release
/tmp/lightai-go-rc3-patch
/tmp/lightai-go-rc3-db
/tmp/lightai-go-rc3-docker
/tmp/lightai-go-rc3-logs
```

## 4. Baseline Commands

Run before any change:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
ls -la docs/reports/full-project-review
git status --short
git diff --stat
git branch --show-current
git log --oneline -10
```

Read these existing documents:

```text
docs/reports/full-project-review/project-understanding-summary.md
docs/reports/full-project-review/full-project-review-report.md
docs/reports/full-project-review/issue-register.md
docs/reports/full-project-review/gpustack-gap-analysis.md
docs/reports/full-project-review/verification-log.md
docs/reports/full-project-review/recommended-hardening-plan.md
```

## 5. Clean Baseline Policy

This RC3 work intentionally does **not** preserve compatibility with the old runtime model, old database, old configuration, old API routes, old Web pages, or old operational docs.

Keep only the final product model:

```text
ModelArtifact
  -> BackendRuntime
  -> ModelDeployment
  -> ResolvedRunPlan
  -> AgentTask
  -> Agent DockerRuntimeDriver
  -> Docker container
  -> Health Check
  -> ModelInstance / GPU lease state
```

Remove the old current-product model:

```text
RuntimeEnvironment
  -> RunTemplate
  -> ModelDeployment
```

Old material may remain only as clearly marked historical/obsolete documentation. It must not appear as a current API, Web workflow, configuration path, test path, operator guide, or OpenAPI contract.

## 6. Stage Plan

### Stage 0 — Baseline / Documentation / Working Tree Protection

Required outputs:
- `rc3-full-hardening-execution-plan.md`
- `rc3-clean-baseline-scope.md`
- `rc3-acceptance-criteria.md`
- `rc3-verification-matrix.md`
- `rc3-issue-closeout-register.md`
- `web-workflow-acceptance-checklist.md`
- initialized `rc3-final-closeout-report.md`

Exit criteria:
- REVIEW-001 through REVIEW-030 are registered.
- Every stage has entry/exit conditions and verification commands.
- Every required document exists in `docs/reports/full-project-review/`.
- `git diff --check` passes.

Commit:
```text
docs: add rc3 full hardening execution plan
```

### Stage 1 — Remove Legacy Runtime Model / Clean API Config DB Docs

Scope:
- REVIEW-012
- REVIEW-016
- REVIEW-025
- All legacy RuntimeEnvironment / RunTemplate remnants.

Required actions:
1. Scan all code/config/docs/scripts/Web/OpenAPI/tests for old runtime model names and paths.
2. Remove or rewrite current-product use of:
   - `/runtime-environments`
   - `/run-templates`
   - `RuntimeEnvironment`
   - `RunTemplate`
   - runtime environment
   - run template
3. Remove obsolete current routes, Web pages, API client methods, docs, E2E steps, OpenAPI paths, and tests.
4. Keep only clearly marked historical references if necessary.
5. Clean DB schema to the current baseline.

Verification:
```bash
rg "/runtime-environments|/run-templates|RuntimeEnvironment|RunTemplate|runtime environment|run template" .
go test ./...
go vet ./...
cd web && npm test || true
cd web && npm run build
find scripts -type f -name "*.sh" -print0 | xargs -0 -n1 sh -n
git diff --check
rg "/runtime-environments|/run-templates|RuntimeEnvironment|RunTemplate" docs scripts web internal configs deploy
```

Commit:
```text
refactor(runtime): remove legacy runtime model and align clean baseline
```

### Stage 2 — Security and Tenant Isolation

Scope:
- REVIEW-001
- REVIEW-002
- REVIEW-008
- REVIEW-009
- REVIEW-013
- REVIEW-014
- REVIEW-019

Required actions:
1. Refuse release/non-dev startup when Agent token is empty or default.
2. Generate or require a secure Agent token at install/init time.
3. Document and/or script Agent token rotation.
4. Add tenant scoping to GPU direct detail API.
5. Transfer GPU tenant ownership in the same transaction as node transfer.
6. Add `tenant_id` to audit logs and scope audit queries by resource tenant.
7. Secure release observability defaults.
8. Provide reverse proxy/TLS deployment guidance.
9. Mark privileged runtime profiles clearly and require explicit operator intent.

Verification:
```bash
go test ./internal/server/api/...
go test ./internal/server/auth/...
go test ./internal/server/db/...
go test ./...
go vet ./...
git diff --check
```

Required tests:
- Default token startup failure.
- Cross-tenant GPU direct-ID access denial.
- Node transfer with GPU visibility.
- Audit tenant scoping.
- Privileged runtime risk metadata/config snapshot.

Commit:
```text
fix(security): enforce agent token and tenant isolation
```

### Stage 3 — Model Runtime Spec Fidelity

Scope:
- REVIEW-003
- REVIEW-022
- REVIEW-026 runtime navigation portion
- REVIEW-027 model artifact/runtime UX portion

Required actions:
1. Stop hand-building incomplete AgentRunSpec payloads.
2. Ensure AgentRunSpec contains vendor, runtime, image, entrypoint, cmd, args, env, ports, volumes, devices, GPU IDs, health check.
3. Map Docker Entrypoint and Cmd correctly.
4. Generate NVIDIA DeviceRequests correctly.
5. Ensure dry-run preview, AgentRunSpec, and Docker create options are equivalent.
6. Validate deployment references at create/update.
7. Support model metadata recommended options plus custom input.

Recommended model metadata options:
- `format`: gguf, safetensors, pt, onnx, other + custom
- `taskType`: chat, completion, embedding, rerank, image, audio, other + custom
- `architecture`: qwen, llama, glm, deepseek, baichuan, mistral, other + custom
- `quantization`: Q4_K_M, Q5_K_M, Q8_0, FP16, BF16, FP8, INT8, INT4, none, other + custom
- `size`: clear unit and meaning

Verification:
```bash
go test ./internal/server/runplan/...
go test ./internal/server/api/...
go test ./internal/agent/runtime/...
go test ./...
go vet ./...
cd web && npm test || true
cd web && npm run build
git diff --check
```

Required tests:
- RunPlan -> AgentRunSpec conversion.
- AgentRunSpec -> Docker create options.
- Docker entrypoint/cmd mapping.
- NVIDIA DeviceRequests.
- Deployment reference validation.
- Model artifact metadata options/custom input.

Commit:
```text
fix(runtime): align runplan agent spec and docker options
```

### Stage 4 — Task Lease / Idempotency / Reconciliation

Scope:
- REVIEW-004
- REVIEW-005
- REVIEW-006
- REVIEW-007

Required actions:
1. Add task lease fields: lease_owner, lease_expires_at, operation_id, generation, attempt, max_attempts.
2. Claim tasks with conditional updates.
3. Validate task result by lease owner, operation ID, and generation.
4. Ignore/reject duplicate, stale, or old-generation results.
5. Normalize failure state; do not write `actual_state='error'`.
6. Treat missing managed container on stop as successful stop.
7. Reconcile managed containers at Agent startup and periodically.
8. Ensure container crash, manual removal, and Agent restart converge to valid states.

Verification:
```bash
go test ./internal/server/api/...
go test ./internal/agent/runtime/...
go test ./...
go vet ./...
git diff --check
```

Required tests:
- Double heartbeat cannot claim same task.
- Duplicate task result ignored.
- Stale operation result ignored.
- Stop missing container succeeds.
- Failed state normalizes to `failed`.
- Agent restart reconciliation.
- Manually removed container reconciliation.

Commit:
```text
fix(agent): add task lease and reconciliation
```

### Stage 5 — Database / Schema / Fresh Install

Scope:
- REVIEW-010
- REVIEW-011
- REVIEW-021
- REVIEW-025 database/version portion

Policy:
- No legacy DB compatibility.
- Current version uses clean baseline DB schema.
- Old tables/fields/migrations that serve only the old model must be removed.

Required actions:
1. Move resource schema into central schema initialization/migration.
2. Treat schema initialization errors as fatal.
3. Fresh DB schema test.
4. Remove legacy tenant migration requirement.
5. Replace any “delete DB to upgrade” wording with current clean baseline policy.
6. Add `tenant_id` to audit logs.
7. Split GPU `collected_at` and `reported_at`.
8. Enforce version/doc consistency.
9. Remove obsolete runtime tables/fields from fresh DB.

Verification:
```bash
go test ./internal/server/db/...
go test ./internal/server/api/...
go test ./...
go vet ./...
git diff --check
rm -rf /tmp/lightai-go-rc3-db
mkdir -p /tmp/lightai-go-rc3-db
# Start server with temp data/runtime/logs and verify fresh DB schema.
```

Required tests:
- Fresh DB initialization.
- Schema idempotency.
- Resource tables exist from central schema.
- `collected_at`/`reported_at` semantics.
- Version consistency.
- No obsolete runtime tables in fresh DB.

Commit:
```text
fix(db): rebuild clean schema baseline
```

### Stage 6 — Observability / Logging / Config / Runtime Security / start-all.sh

Scope:
- REVIEW-017
- REVIEW-020
- REVIEW-021 remaining portion
- REVIEW-013 remaining portion
- REVIEW-019 remaining portion
- REVIEW-029
- REVIEW-030

Required actions:
1. Choose and document current observability management mode.
2. Align bundled/external/disabled observability config, docs, and scripts.
3. Implement, remove, or warn/error clearly for `report_interval`.
4. Implement, remove, or warn/error clearly for `metrics.advertise_addr`.
5. Align `collected_at`/`reported_at` across API/Web/Grafana.
6. Add no-data troubleshooting.
7. Secure observability exposure defaults.
8. Align privileged runtime risk handling across API/Web/docs.
9. Add `scripts/start-all.sh`, paired with `scripts/stop-all.sh`.
10. Reduce repetitive success logging in Server and Agent.

#### start-all.sh minimum requirements

Script:

```text
scripts/start-all.sh
```

Required parameters:
```bash
scripts/start-all.sh
scripts/start-all.sh --dry-run
scripts/start-all.sh --no-observability
scripts/start-all.sh --wait
```

Required behavior:
1. Works from source tree and release directory, or explicitly detects mode.
2. Checks required scripts/configs.
3. Starts Server, bundled observability if enabled, then Agent.
4. Skips already-running processes.
5. Does not overwrite data, credentials, DB, config, runtime, or logs.
6. Prints PID, log path, and listening address.
7. `--wait` performs health checks.
8. `--dry-run` prints intended actions only.
9. Returns non-zero on failure.

Required health checks:
- Server health endpoint.
- Agent metrics endpoint.
- Prometheus endpoint if enabled.
- Grafana endpoint if enabled.

#### Logging noise reduction requirements

Server:
- Default INFO must not repeatedly log successful:
  - `GET /metrics`
  - `GET /metrics/targets`
  - `GET /health`
  - `GET /ready`
  - `/assets/*`
  - favicon/static frontend files
- Non-2xx/3xx requests must be logged.
- Slow requests must be logged.
- State-changing requests must be logged.
- Auth/security/runtime failures must be logged.
- Full access log must be configurable.

Agent:
- Stable heartbeat success must not repeatedly log at INFO.
- No-task polling must not repeatedly log at INFO.
- GPU metrics success must not repeatedly log at INFO.
- First success, failure, recovery, claimed tasks, GPU health/availability changes, abnormal latency, startup, and shutdown must remain visible.

Verification:
```bash
go test ./...
go vet ./...
find scripts -type f -name "*.sh" -print0 | xargs -0 -n1 sh -n
scripts/start-all.sh --dry-run
scripts/start-all.sh --dry-run --no-observability
scripts/start-all.sh --wait
scripts/stop-all.sh
git diff --check
```

Disposable release validation:
```bash
# In unpacked disposable release dir:
scripts/start-all.sh --dry-run
scripts/start-all.sh --wait
scripts/start-all.sh --wait
scripts/stop-all.sh
```

Required logging validation:
1. Run stable Server/Agent for 10 minutes.
2. Server log must not contain repeated `/metrics 200` noise.
3. Agent log must not repeatedly emit `heartbeat.summary`, `task_poll.summary`, `gpu_metrics.summary` at INFO.
4. Trigger a representative error and confirm WARN/ERROR remains visible.
5. Enable debug/full access log and confirm detailed logs are available.

Commit:
```text
fix(ops): harden observability logging and start orchestration
```

### Stage 7 — Web i18n / Model UX / Web Workflow Completeness

Scope:
- REVIEW-015
- REVIEW-024
- REVIEW-026
- REVIEW-027
- REVIEW-028

Required actions:
1. Add runnable Web test script/dependencies.
2. `cd web && npm test` must pass.
3. Scan for raw i18n keys.
4. Fix:
   - `nav.models`
   - `nav.runtime`
   - `artifacts.name`
   - `artifacts.path`
   - `artifacts.format`
   - `artifacts.taskType`
   - `artifacts.architecture`
   - `artifacts.size`
   - `artifacts.quantization`
5. Fix all model/runtime page raw keys, not only the listed keys.
6. Complete zh-CN and en-US locale coverage.
7. Support recommended options plus custom input for model metadata fields.
8. Improve long-field display for GPU names, model paths, image names.
9. Add loading, empty, and error states.
10. Complete Web workflow checklist.
11. Reduce Vite chunk warning as far as practical; document if any remains.

Verification:
```bash
cd web && npm test
cd web && npm run build
go test ./...
go vet ./...
git diff --check
```

Required tests:
- Core navigation i18n.
- Model artifact i18n.
- No raw i18n key smoke.
- Locale completeness.
- Artifact metadata options/custom input.
- Route/menu title.
- Basic workflow page render.

Commit:
```text
fix(web): complete i18n and model workflow UX
```

### Stage 8 — Testing Infrastructure / OpenAPI / Documentation Consistency

Scope:
- REVIEW-012
- REVIEW-015
- REVIEW-016
- REVIEW-025
- REVIEW-028 remaining portion

Required actions:
1. Update OpenAPI to current route surface.
2. Add route list vs OpenAPI path check.
3. Rewrite docs/ops and docs/testing around BackendRuntime/RunPlan.
4. Align getting-started, E2E, troubleshooting, release note, phase status.
5. Remove or obsolete old RuntimeEnvironment/RunTemplate docs.
6. Ensure docs commands are executable in disposable environments.
7. Align version references.
8. Document start-all/stop-all.
9. Document logging strategy and debug/full access mode.

Verification:
```bash
go test ./...
go vet ./...
cd web && npm test
cd web && npm run build
find scripts -type f -name "*.sh" -print0 | xargs -0 -n1 sh -n
git diff --check
rg "/runtime-environments|/run-templates|RuntimeEnvironment|RunTemplate" docs scripts web internal configs deploy
```

Historical references may remain only when clearly marked obsolete and not presented as current instructions.

Commit:
```text
docs: align runtime api openapi operations and logging guides
```

### Stage 9 — Isolated E2E / Release / Patch / Rollback Verification

Scope:
- REVIEW-018
- REVIEW-023
- All prior integrated validation.

Use only disposable directories:
```text
/tmp/lightai-go-rc3-e2e
/tmp/lightai-go-rc3-release
/tmp/lightai-go-rc3-patch
/tmp/lightai-go-rc3-docker
/tmp/lightai-go-rc3-logs
```

Required scenarios:
1. Clean release tarball install.
2. Fresh DB startup.
3. Initial credentials generation.
4. Web login.
5. Server health.
6. start-all.sh startup.
7. Agent registration.
8. NVIDIA GPU discovery and display.
9. Dashboard display.
10. ModelArtifact create.
11. BackendRuntime create.
12. Deployment create.
13. Dry-run preview.
14. Start instance.
15. Endpoint health check.
16. `/v1/models` or equivalent OpenAI-compatible smoke.
17. Stop instance.
18. Lease release.
19. Missing-container stop idempotency.
20. Agent restart reconciliation.
21. Prometheus targets.
22. Grafana/provisioning smoke.
23. Patch package build.
24. Patch apply.
25. Patch rollback.
26. Log collection.
27. Password reset.
28. Multi-tenant direct-ID isolation smoke.
29. Audit tenant scoping smoke.
30. Web i18n smoke.
31. Web workflow acceptance.
32. stop-all.sh stops all start-all.sh processes.
33. Repeated start-all.sh does not duplicate processes.
34. Server access log noise filter smoke.
35. Agent periodic summary noise filter smoke.
36. Error visibility after logging noise reduction smoke.
37. Debug/full access log smoke.

MetaX:
- If no MetaX hardware is accessible, mark only this as `Blocked - External Hardware`.
- Provide runnable MetaX validation script and acceptance criteria.
- If hardware is accessible, run discover, metrics, and runtime smoke and save sanitized evidence.

Verification:
```bash
go test ./...
go vet ./...
cd web && npm test
cd web && npm run build
find scripts -type f -name "*.sh" -print0 | xargs -0 -n1 sh -n
git diff --check
# plus project release/package/patch commands
```

Commit:
```text
test(e2e): verify rc3 release runtime logging and patch workflows
```

### Stage 10 — Final Closeout / Commit / Push

Required final state:
1. REVIEW-001 through REVIEW-030 closed.
2. All REVIEW-031+ closed.
3. No Open.
4. No In Progress.
5. No Not Verified.
6. No Deferred.
7. No Later.
8. Verification matrix filled with actual result, evidence, and status.
9. Web workflow checklist has no Not Verified.
10. Final closeout report completed.

Final verification:
```bash
git status --short
git diff --stat
git diff --check
go test ./...
go vet ./...
cd web && npm test
cd web && npm run build
find scripts -type f -name "*.sh" -print0 | xargs -0 -n1 sh -n
```

Final report must include:

```text
RC3 Full Hardening Closure Completed

Compatibility:
- Legacy DB compatibility: intentionally removed
- Legacy RuntimeEnvironment/RunTemplate model: removed from current product scope
- Current clean baseline: BackendRuntime / RunPlan / ModelDeployment / AgentTask / Docker runtime
- Old API/docs/config remnants: cleared or marked obsolete

Issues:
- Fixed: N
- Not Reproducible: N
- Blocked - External Hardware: N
- Blocked - Explicit Product Decision: N
- Open: 0
- Deferred: 0
- Not Verified: 0

Verification:
- go test ./...: PASS
- go vet ./...: PASS
- web tests: PASS
- web build: PASS
- shell syntax: PASS
- git diff --check: PASS
- legacy API/code/config/docs scan: PASS
- fresh DB initialization: PASS
- tenant direct-ID isolation: PASS
- audit tenant scoping: PASS
- RunPlan -> AgentRunSpec -> Docker options: PASS
- task lease race/idempotency: PASS
- runtime reconciliation: PASS
- NVIDIA model E2E: PASS
- observability smoke: PASS
- server access log noise filter: PASS
- agent periodic summary noise filter: PASS
- error visibility after logging noise reduction: PASS
- debug/full access log mode: PASS
- start-all.sh dry-run: PASS
- start-all.sh --wait: PASS
- stop-all.sh after start-all.sh: PASS
- release package: PASS
- release install smoke: PASS
- patch apply: PASS
- patch rollback: PASS
- Web workflow acceptance: PASS
- raw i18n key scan: PASS
- MetaX hardware: PASS / Blocked - External Hardware

Top Remaining Risks:
- None for current supported scope.
- External hardware blockers only if applicable.

Git:
- git status --short
- git diff --stat
- git log --oneline -10
- pushed branch/commit
```

Final commit:
```text
docs: add rc3 final closeout report
```

Push:
```bash
git status --short
git log --oneline -10
git push
```

Output final branch, commit hash, and push result.

## 7. Immediate Start Instruction

Start at Stage 0 now.

Do not ask for confirmation.

Do not begin code changes until all Stage 0 documents are created and populated.

After Stage 0 exits successfully, continue through Stage 10 automatically.

If any command fails, fix the cause and rerun it. Do not record failure and continue, except for true external hardware absence such as MetaX hardware not being available.
