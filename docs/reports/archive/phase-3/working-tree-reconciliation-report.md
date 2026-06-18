> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# Working Tree Reconciliation Report — Phase 3 Final

> Date: 2026-06-17
> Build: ✅ | Tests: ✅ | npm: ✅ | Scripts: ✅

## 1. Summary

| Metric | Count |
|--------|-------|
| Modified tracked files | 27 |
| Deleted tracked files | 18 |
| New untracked files | 79 |
| Lines added | 1,843 |
| Lines deleted | 6,898 |

Change categories:
- **A. Logging/Observability** — 12 files (helpers, middleware, Docker, health, auth)
- **B. Runtime/RunPlan/Backend refactor** — 28 files (configs, runplan package, handlers, models)
- **C. Web UI refactor** — 16 files (pages, APIs, router, i18n)
- **D. Deleted old code** — 18 files (replaced by refactored equivalents)
- **E. Scripts** — 7 files (e2e, diagnose, smoke, package)
- **F. Docs** — 22 files (phase plans, reports, testing docs)

## 2. VERSION / logs / runtime / dist

| Check | Status |
|-------|--------|
| VERSION | ✅ Clean (no diff) |
| logs/ | ✅ Not in git |
| runtime/ | ✅ Not in git |
| dist/ | ✅ Not in git |
| build artifacts | ✅ None tracked |

## 3. Deleted Files — Replacement Verification

### Go (11 files)

| Deleted | Replacement | Safe? |
|---------|-------------|-------|
| `deployment_lifecycle.go` | `deployment_lifecycle_handlers.go` | ✅ |
| `instance_state.go` | Constants in `constants.go` | ✅ |
| `lease.go` | Inline in deployment/agent handlers | ✅ |
| `model_handlers.go` | Split: `artifact_handlers.go`, `backend_handlers.go`, `runtime_handlers.go` | ✅ |
| `model_handlers_test.go` | `phase3_rbac_test.go` | ✅ |
| `rbac_phase2f_test.go` | `phase3_rbac_test.go` | ✅ |
| `resolve_helper.go` | `runplan/resolver.go` | ✅ |
| `sweep.go` | `sweepExpiredTasks` in `agent_handlers.go` | ✅ |
| `task_constants.go` | `constants.go` | ✅ |
| `task_handlers.go` | Task result in `agent_handlers.go` | ✅ |
| `resolver/resolver.go` | `runplan/resolver.go` | ✅ |

Symbol check: No deleted symbols referenced in remaining Go code. Build passes.

### Web (7 files)

| Deleted | Replacement | Safe? |
|---------|-------------|-------|
| `modelArtifacts.ts` | Inline in page components | ✅ |
| `modelDeployments.ts` | Inline in page components | ✅ |
| `modelInstances.ts` | Inline in page components | ✅ |
| `runTemplates.ts` | `runtimes.ts` | ✅ |
| `runtimeEnvironments.ts` | `backends.ts` | ✅ |
| `RunTemplatesPage.vue` | `BackendRuntimesPage.vue` | ✅ |
| `RuntimeEnvironmentsPage.vue` | `BackendsPage.vue` | ✅ |

## 4. New Untracked Files

### Logging (this round's additions)

| File | Purpose |
|------|---------|
| `internal/common/log/helpers.go` | Operation lifecycle wrapper |
| `internal/common/log/redact.go` | Unified redaction |
| `internal/common/log/summary.go` | High-frequency summary/sampling |
| `internal/server/api/middleware_logging.go` | Request logging middleware |
| `internal/agent/runtime/health.go` | Endpoint health check |
| `internal/agent/runtime/health_test.go` | Health check tests (5 pass) |

### Runtime / RunPlan / Backend

| File | Purpose |
|------|---------|
| `configs/model-runtime/` (15 YAML) | Backend/runtime/template configs |
| `internal/server/runplan/` (8 Go) | RunPlan resolver, types, preview, templates |
| `internal/agent/runtime/runplan_adapter.go` | RunPlan→AgentRunSpec converter |
| `internal/server/api/artifact_handlers.go` | ModelArtifact CRUD |
| `internal/server/api/backend_handlers.go` | Backend/version list |
| `internal/server/api/constants.go` | Unified constants |
| `internal/server/api/deployment_lifecycle_handlers.go` | Deployment lifecycle |
| `internal/server/api/helpers.go` | HTTP helpers, redaction |
| `internal/server/api/runtime_handlers.go` | BackendRuntime CRUD |
| `internal/server/api/patch_validator.go` | Patch validation |
| `internal/server/api/phase3_rbac_test.go` | RBAC tests |
| `internal/server/models/` (6 Go) | Model structs |

### Web

| File | Purpose |
|------|---------|
| `web/src/api/backends.ts` | Backend API client |
| `web/src/api/runtimes.ts` | Runtime API client |
| `web/src/pages/BackendRuntimesPage.vue` | Runtime management |
| `web/src/pages/BackendsPage.vue` | Backend listing |

### Scripts

| File | Purpose |
|------|---------|
| `scripts/diagnose-model-runtime-spec.sh` | Spec diagnostic |
| `scripts/e2e-model-runtime-api.sh` | E2E test |
| `scripts/smoke-model-backends.sh` | Backend smoke test |

### Docs

| File | Purpose |
|------|---------|
| `docs/design/13-backend-runplan-runtime-design.md` | Runtime design |
| `docs/plan/phase-*` (6 md) | Phase planning |
| `docs/reports/phase-3/` (8 md + 10 verification) | Audit/reports |
| `docs/review/gpustack-runtime-reference-review.md` | GPUStack review |
| `docs/testing/` (2 md) | Test documentation |

## 5. Modified Files by Category

### A. Logging/Observability (12 files)

`cmd/agent/main.go` — Heartbeat/task_poll/metrics summaries, task result report levels
`cmd/server/main.go` — DB timing, RequestLoggingMiddleware integration
`internal/common/log/log.go` — Context-aware logging (InfoContext, WithRequestID)
`internal/agent/runtime/docker.go` — Docker lifecycle logging, post-start verify, health wiring
`internal/agent/runtime/driver.go` — HealthCheckConfig on AgentRunSpec
`internal/agent/register/register.go` — OperationID on TaskResult
`internal/server/auth/middleware.go` — Agent auth + permission WARN
`internal/server/api/agent_handlers.go` — StateTransition, lease sweep, task claim noise fix
`internal/server/api/deployment_lifecycle_handlers.go` — NULL scan fix, Operation wrapper, lease logging, noise fix
`internal/server/api/resource_handlers.go` — GPU StateTransition
`internal/server/api/middleware_logging.go` — Request logging middleware
`internal/server/rbac/handlers.go` — RBAC write duration

### B. Runtime/Backend refactor (10 files)

`internal/server/api/router.go` — Phase 3 routes
`internal/server/db/db.go` — Schema v10
`internal/server/models/models.go` — Model restructure
`internal/server/auth/bootstrap.go` — Bootstrap changes
`internal/server/api/artifact_handlers.go` — Artifact CRUD
`internal/server/api/backend_handlers.go` — Backend handlers
`internal/server/api/constants.go` — Constants
`internal/server/api/runtime_handlers.go` — Runtime CRUD
`internal/server/models/{artifact,backend,deployment,instance,runplan,runtime}.go` — Model structs

### C. Web (10 files)

`web/src/router/index.ts` — New routes for backends/runtimes
`web/src/locales/en-US.ts` — i18n updates
`web/src/locales/zh-CN.ts` — i18n updates
`web/src/pages/DashboardPage.vue` — Dashboard polish
`web/src/pages/ModelArtifactsPage.vue` — Artifact page
`web/src/pages/ModelDeploymentsPage.vue` — Deployment page
`web/src/pages/ModelInstancesPage.vue` — Instance page
`web/src/utils/format.ts` — Format helpers
`web/src/utils/status.ts` — Status helpers
`web/tests/formatters.test.mjs` — Tests

### D. Scripts (3 files)

`scripts/check-glibc-compat.sh` — GLIBC scope fix
`scripts/package-release-docker.sh` — Docker packaging
`scripts/package-release.sh` — Release packaging

## 6. Web Router / Menu Consistency

| Check | Status |
|-------|--------|
| ConsoleLayout menu paths | ✅ Fixed: `/runtime/environments` → `/backends`, `/runtime/templates` → `/runtimes` |
| i18n labels | ✅ Use existing `backends.title` / `runtimes.title` |
| Router paths | ✅ `/backends` → BackendsPage, `/runtimes` → BackendRuntimesPage |
| No undefined route links | ✅ All menu items resolve to router paths |
| npm build | ✅ Pass (2.94s) |

## 7. Verification Results

| Check | Result |
|-------|--------|
| `gofmt` | ✅ Clean (0 project files with issues) |
| `go test ./... -count=1` | ✅ 9 packages PASS |
| `go build ./cmd/server/` | ✅ |
| `go build ./cmd/agent/` | ✅ |
| `npm --prefix web run build` | ✅ 2.94s |
| `find scripts -name '*.sh' \| xargs bash -n` | ✅ 27 scripts |
| `git diff --check` | ✅ |
| VERSION | ✅ Clean (reverted) |
| logs/runtime/dist | ✅ Not in git |
| Deleted symbol references | ✅ None found |
| Web import consistency | ✅ No broken imports |

## 8. Endpoint Health Check Runtime Validation

**Status: PARTIAL**

Two single-llamacpp attempts were made (Round 4 and Round 5). Both exited(1) — the container entrypoint/command config caused immediate exit. The health check module was exercised (16 attempts over 30s, re-inspect aborted early on exited container) but never reached a successful endpoint-ready state.

The health check logging infrastructure (started, wait_progress, wait_timeout, container_exited, stderr capture) was fully verified to work correctly. What remains unverified is: a container that actually reaches ready state, with the health check confirming endpoint readiness and returning success.

## 9. Remaining Risks

| Risk | Severity | Details |
|------|----------|---------|
| Endpoint health check never confirmed "ready" | Medium | Container always exit(1). Need working entrypoint/command for llama.cpp to verify end-to-end. |
| Health check port=0 edge case | Low | When no host_port configured, falls back to 8080. Verified in unit test. |
| `llama-server` in defaultArgs removed from config | Low | Config fix applied; old binary may have been used in Round 5 test |

## 10. Suggested Commit Split (for reference — do not execute)

1. **Phase 3 RunPlan/backend model/API refactor**
   - Files: `internal/server/runplan/`, `internal/server/models/`, `internal/server/api/` (handlers, router, constants), `configs/model-runtime/`, `internal/agent/runtime/runplan_adapter.go`
   - Message: `phase-3: add RunPlan resolver, backend/runtime/artifact/deployment CRUD, model configs`
   - Verify: `go test ./internal/server/runplan/... ./internal/server/api/...`

2. **Web backend/runtime page refactor**
   - Files: `web/src/pages/BackendsPage.vue`, `web/src/pages/BackendRuntimesPage.vue`, `web/src/api/backends.ts`, `web/src/api/runtimes.ts`, `web/src/router/index.ts`, `web/src/locales/`, deleted old pages
   - Message: `web: add Backends/Runtimes pages, remove deprecated templates/environments pages`
   - Verify: `npm --prefix web run build`

3. **Logging/observability hardening**
   - Files: `internal/common/log/helpers.go`, `internal/common/log/redact.go`, `internal/common/log/summary.go`, `internal/common/log/log.go`, `internal/server/api/middleware_logging.go`, `cmd/server/main.go`, `cmd/agent/main.go`, `internal/server/auth/middleware.go`, `internal/agent/runtime/docker.go`, `internal/agent/register/register.go`, `internal/server/api/agent_handlers.go`, `internal/server/api/deployment_lifecycle_handlers.go`, `internal/server/api/resource_handlers.go`, `internal/server/rbac/handlers.go`
   - Message: `logging: add full-chain operation lifecycle logging, high-frequency summary, request_id/operation_id correlation`
   - Verify: `go test ./...`

4. **Runtime health check + diagnose/e2e scripts**
   - Files: `internal/agent/runtime/health.go`, `internal/agent/runtime/health_test.go`, `internal/agent/runtime/driver.go`, `scripts/diagnose-model-runtime-spec.sh`, `scripts/e2e-model-runtime-api.sh`, `scripts/smoke-model-backends.sh`
   - Message: `runtime: add endpoint health check with container re-inspect, diagnose/e2e scripts`
   - Verify: `go test ./internal/agent/runtime/...`, `bash -n scripts/*.sh`

5. **Docs/reports**
   - Files: `docs/reports/phase-3/`, `docs/design/`, `docs/plan/`, `docs/testing/`, `docs/review/`
   - Message: `docs: add Phase 3 logging audit, coverage audit, final reports, verification records`
   - Verify: manual review

## 11. Final git status --short Summary

```
M  — 27 tracked files modified (logging, runtime, web, scripts)
D  — 18 tracked files deleted (replaced by refactored equivalents)
?? — 79 new files (configs, handlers, models, runplan, logging, health, scripts, web, docs)
```

All changes legitimate. No build artifacts, logs, runtime files, or temp files in working tree.

**未 commit，未 push。**
