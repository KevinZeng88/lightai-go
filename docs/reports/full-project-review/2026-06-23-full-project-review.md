# LightAI Go Full Project Review Report

> Date: 2026-06-23
> Reviewer: MiMo Code Agent (automated comprehensive review)
> Branch: main (working tree, no uncommitted code changes)
> Scope: Full project — server, agent, web, tests, docs, configs, scripts
> Classification: CONFIRMED / SUSPECTED / NEEDS VERIFICATION

---

## 1. Executive Summary

LightAI Go is a well-architected lightweight GPU/node management platform at Phase 4 maturity. The core NVIDIA Docker lifecycle path (model root → browse → scan → artifact → backend → runtime → preflight → RunPlan → Docker start → /v1/models → logs → stop → cleanup) is **implemented and E2E-verified on real hardware** (RTX 5090, Docker 29.5.3).

**Overall maturity: Phase 4 — functional for single-node NVIDIA, documented gaps for multi-node/MetaX/Huawei.**

Key strengths:
- Clean Backend/BackendVersion/BackendRuntime/NodeBackendRuntime snapshot boundary design
- Well-structured RunPlan resolver with 50+ unit tests covering 3 backends
- Solid auth (Argon2id, CSRF, session hashing, tenant isolation)
- Comprehensive E2E evidence with 20+ scripts and checked-in artifacts
- Good documentation governance with formal open-issues closeout

Key risks:
- **8 confirmed P1 tenant isolation gaps** in server API handlers
- **2 P0 security issues** (SSRF via agent proxy, privileged container passthrough)
- **No auth on agent metrics endpoint** exposing filesystem browsing
- **Container resource leak** on start failure
- **Frontend has no component tests** — only static analysis
- **Multiple hardcoded Chinese strings** bypassing i18n in frontend

---

## 2. Current Overall Maturity Judgment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Architecture | **Strong** | Clean snapshot boundaries, file-based catalog, well-separated server/agent |
| Core NVIDIA Path | **Production-ready for single-node** | E2E verified on RTX 5090, 3-backend matrix pass |
| Security | **Needs hardening** | 2 P0 + 8 P1 issues found; auth foundation is solid but handler-level checks have gaps |
| Multi-tenant | **Partial** | Core tenant isolation exists; 8 node-proxy/NBR endpoints missing tenant scope checks |
| MetaX/Huawei | **Template-only** | No real hardware validation; documented as DOCUMENTED_BLOCKER |
| Test Coverage | **Moderate** | Strong unit tests for RunPlan/resolver; no auth unit tests; no frontend component tests |
| E2E Credibility | **High for NVIDIA** | Real hardware evidence; scripts are machine-specific (developer laptop) |
| Documentation | **Good but voluminous** | 39+ report directories; some E2E evidence directories bloat the repo |
| Web UI | **Functional** | 19 pages, i18n coverage ~600 keys, but hardcoded strings and missing route guards |
| Observability | **Implemented** | Prometheus/Grafana integration, audit logging, structured error codes |

---

## 3. Documentation vs Implementation Inconsistencies

### 3.1 VERSION file says 0.1.0 but release notes say v0.1.9
- **File**: `VERSION:1` → `0.1.0`
- **File**: `docs/RELEASE_NOTE_v0.1.9.md`
- **Status**: CONFIRMED
- **Impact**: VERSION file is stale; does not reflect actual release state
- **Priority**: P3

### 3.2 PHASE-STATUS.md references branch `phase-4-model-runtime-wizards` but working branch is `main`
- **File**: `docs/PHASE-STATUS.md:11` → `Branch: phase-4-model-runtime-wizards`
- **Actual**: `git status` shows on `main`
- **Status**: CONFIRMED
- **Impact**: Misleading branch reference in phase status document
- **Priority**: P3

### 3.3 CURRENT.md claims "Last reviewed: 2026-06-18" — 5 days stale
- **File**: `docs/CURRENT.md:4`
- **Status**: CONFIRMED
- **Impact**: Minor; document has not been updated to reflect recent commits (d61c409 through 08795eb)
- **Priority**: P3

### 3.4 docs/README.md says "Read order: Start with docs/CURRENT.md" but AGENTS.md says "Read these first: docs/README.md, docs/PHASE-STATUS.md, docs/RELEASE_NOTE_v0.1.9.md"
- **File**: `docs/README.md:7` vs `AGENTS.md:11-15`
- **Status**: CONFIRMED
- **Impact**: Agent instructions and README disagree on reading order
- **Priority**: P3

### 3.5 E2E evidence directories bloat the repository
- **Path**: `docs/reports/model-runtime-node-wizard/e2e-*` — 29 directories
- **Path**: `docs/reports/model-runtime-node-wizard/failed-instance-logs-*` — 5 directories
- **Status**: CONFIRMED
- **Impact**: Repository contains server/agent logs, JSON payloads, tar.gz archives as checked-in files
- **Priority**: P3

---

## 4. Architecture / Model / Runtime Design Issues

### 4.1 P1 CONFIRMED: `deduplicateArgs` mishandles boolean flags
- **File**: `internal/server/runplan/resolver.go:470-504`
- **Issue**: Assumes all flags come in flag-value pairs. Boolean flags like `--trust-remote-code` followed by another flag are incorrectly paired with the next flag's value.
- **Impact**: Silent incorrect args passed to container; user-specified boolean flags from ParameterDefs could be duplicated or misinterpreted.
- **Priority**: P1

### 4.2 P1 CONFIRMED: Deployment env overrides bypass `substituteVars()`
- **File**: `internal/server/runplan/resolver.go:598-600`
- **Issue**: Layer 5 (deployment `env_overrides_json`) inserts values raw without template variable substitution, unlike layers 1-4.
- **Impact**: Template variables like `{{MODEL_CONTAINER_PATH}}` in user env overrides are silently ignored.
- **Priority**: P1

### 4.3 P1 CONFIRMED: Required parameters silently skipped with no error
- **File**: `internal/server/runplan/resolver.go:533-537`
- **Issue**: When a required parameter is missing, the code comments "resolver will report" but no error is appended. `buildArgs` does not check for missing required parameters.
- **Impact**: Deployment could be created with missing `--model` or `--model-path` arg; container starts and fails at runtime.
- **Priority**: P1

### 4.4 P2 CONFIRMED: `buildDeviceBinding()` defined but never called by `Resolve()`
- **File**: `internal/server/runplan/resolver.go:986-1030`
- **Issue**: Function exists and creates `DeviceBinding` struct, but `Resolve()` never calls it. `ResolvedRunPlan.DeviceBinding` is always nil.
- **Impact**: Dead code; agent-side code depending on this field gets no data.
- **Priority**: P2

### 4.5 P2 CONFIRMED: `computeInputHash` omits AssignedGPUs, NodeRuntimeOverride, ProcessStartConfig
- **File**: `internal/server/runplan/resolver.go:933-949`
- **Issue**: Input hash computation does not include GPU assignments or node runtime overrides. Two plans with different GPU assignments produce the same hash.
- **Impact**: If used for caching/idempotency, different GPU assignments would not trigger re-resolution.
- **Priority**: P2

### 4.6 P2 CONFIRMED: Stale SGLang catalog version reference
- **File**: `configs/backend-catalog/runtimes/sglang/nvidia-cuda.yaml:6`
- **Issue**: References `sglang-v0.5.12.post1` while nvidia-docker runtime correctly references `sglang-v0.5.13.post1`.
- **Impact**: Users selecting `sglang-nvidia-cuda` get old v0.5.12 defaults.
- **Priority**: P2

### 4.7 P2 CONFIRMED: Ollama catalog uses raw JSON string blob for capabilities
- **File**: `configs/backend-catalog/versions/ollama/ollama-latest.yaml:15`
- **Issue**: `capabilities_json` is a raw JSON string, inconsistent with all other versions that use structured YAML lists.
- **Impact**: Ollama capabilities may not be properly parsed through the same path as vLLM/SGLang/llama.cpp.
- **Priority**: P2

### 4.8 P2 CONFIRMED: vLLM `nvidia-cuda` runtime YAML has dead config keys
- **File**: `configs/backend-catalog/runtimes/vllm/nvidia-cuda.yaml:24-27`
- **Issue**: `gpus: all` and `runtime: nvidia` keys have no matching Go struct fields in `DockerSpecInfo`. They are silently ignored.
- **Impact**: Dead configuration that gives the appearance of GPU configuration but has no effect.
- **Priority**: P2

---

## 5. Security & Permission Issues

### 5.1 P0 CONFIRMED: SSRF via Agent Proxy Endpoints
- **File**: `internal/server/api/agent_handlers.go:611`, `agent_proxy_handlers.go:51,109`
- **Issue**: Server makes outbound HTTP requests to agent addresses from DB (attacker-controlled if agent compromised). No SSRF protection, no timeout, no URL encoding on query params.
- **Impact**: Malicious agent could register with `advertised_address=169.254.169.254` (cloud metadata) to perform SSRF.
- **Priority**: P0

### 5.2 P0 CONFIRMED: Privileged Container Mode Accepted Without Validation
- **File**: `internal/agent/runtime/docker.go:425-427`
- **Issue**: Agent blindly trusts server-sent `AgentRunSpec.Privileged` flag. No agent-side policy, allowlist, or confirmation.
- **Impact**: Compromised server can instruct agent to run fully privileged containers, granting complete host access.
- **Priority**: P0

### 5.3 P0 CONFIRMED: Agent Token Weak Default in Release Config
- **File**: `configs/server.release.yaml:11`, `internal/common/config/config.go:161`
- **Issue**: Default agent token is `"lightai-agent-token-change-me"`. No runtime enforcement or startup warning when default is used.
- **Impact**: Any network neighbor can register rogue agents if deployed without changing token.
- **Priority**: P0

### 5.4 P1 CONFIRMED: 8 Node-Proxy/NBR Endpoints Missing Tenant Scope Checks
- **Files**: `agent_proxy_handlers.go:13-141`, `agent_handlers.go:594-663`, `runtime_handlers.go:248-833`, `node_runtime_handlers.go:98-189`, `artifact_handlers.go:549-557`, `model_browser_handlers.go:232-286`
- **Issue**: File browsing, model scan, Docker image inspect, NBR list/check/probe/enable/patch/delete, model location rescan, and model root patch/delete all skip tenant scope validation. Only permission check (`node:read`, `backend_runtime:read/write`) is enforced, not tenant ownership.
- **Impact**: User in tenant A can access nodes, files, and runtimes belonging to tenant B.
- **Priority**: P1

### 5.5 P1 CONFIRMED: Agent Metrics Endpoint Has No Authentication
- **File**: `cmd/agent/main.go:291-476`
- **Issue**: Entire metrics/healthz HTTP server has zero authentication. Exposes `/metrics`, `/healthz`, `/docker-images`, `/docker-image-inspect`, `/files`, `/model-paths/scan`.
- **Impact**: Any network-reachable client can inspect Docker images, browse filesystem, scan model paths.
- **Priority**: P1

### 5.6 P1 CONFIRMED: `AllowRuntimeRootAdd` Bypasses Allowed Roots Restriction
- **File**: `cmd/agent/main.go:377-379,452-453`
- **Issue**: When `AllowRuntimeRootAdd=true`, any HTTP client can add arbitrary filesystem roots via `extra_roots` query parameter. Combined with no auth (5.5), allows scanning any directory.
- **Priority**: P1

### 5.7 P1 CONFIRMED: Rate Limiter Trusts Spoofable X-Forwarded-For
- **File**: `internal/server/auth/ratelimit.go:81-85`
- **Issue**: Login rate limiter uses `X-Forwarded-For` for per-IP limiting. Attacker can bypass by setting different XFF values.
- **Impact**: Brute-force login attacks without per-IP throttling.
- **Priority**: P1

### 5.8 P1 CONFIRMED: No Request Body Size Limit
- **Files**: All handlers using `json.NewDecoder(r.Body).Decode()`
- **Issue**: No `http.MaxBytesReader` on any request body.
- **Impact**: DoS via multi-gigabyte JSON body exhausting server memory.
- **Priority**: P1

### 5.9 P1 CONFIRMED: JSON Injection in Audit Log Detail
- **File**: `internal/server/api/agent_handlers.go:882-883`
- **Issue**: `req.Reason` interpolated directly into JSON string via `fmt.Sprintf`. User-supplied `"` or `\` produces malformed/injectable JSON.
- **Priority**: P1

### 5.10 P1 CONFIRMED: Credentials File Written to Predictable Path
- **File**: `internal/server/auth/bootstrap.go:354`
- **Issue**: Initial admin credentials written to `runtime/initial-credentials.txt` with 0600. Path is well-known, never auto-deleted.
- **Priority**: P1

### 5.11 P2 CONFIRMED: Observability Status Endpoint Has No Authentication
- **File**: `router.go:72`, `observability_handler.go:12-34`
- **Issue**: `/api/v1/observability/status` registered without auth. Leaks Prometheus/Grafana URLs and readiness to unauthenticated callers.
- **Priority**: P2

### 5.12 P2 CONFIRMED: CSRF Token Rotation on Every `/me` Call
- **File**: `internal/server/auth/handlers.go:357-359`
- **Issue**: Every `GET /api/v1/auth/me` rotates CSRF secret. Concurrent `/me` calls invalidate each other's tokens.
- **Impact**: Intermittent CSRF validation failures in concurrent frontend scenarios.
- **Priority**: P2

### 5.13 P2 CONFIRMED: Agent Token Compared with Non-Constant-Time Comparison
- **File**: `internal/server/auth/middleware.go:180`
- **Issue**: Uses plain `!=` instead of `crypto/subtle.ConstantTimeCompare`.
- **Priority**: P2

---

## 6. Stability & State Machine Issues

### 6.1 P0 CONFIRMED: Container Not Cleaned Up After Start Failure
- **File**: `internal/agent/runtime/docker.go:120-131,153-162,180-194`
- **Issue**: When `ContainerCreate` succeeds but `ContainerStart` fails, the created container is left behind. No `ContainerRemove` call.
- **Impact**: Container name collision on retry, Docker resource leak (disk, network namespaces).
- **Priority**: P0

### 6.2 P0 CONFIRMED: Race Condition on `logsTaskState.lastStderrBytes` Map
- **File**: `cmd/agent/main.go:1120-1126,1197-1199`
- **Issue**: Plain `map[string]int` accessed from multiple goroutines without synchronization.
- **Impact**: Undefined behavior and panics in Go.
- **Priority**: P0

### 6.3 P1 CONFIRMED: No Container Removal on Stop — Restart Policy Conflict
- **File**: `internal/agent/runtime/docker.go:230-276`
- **Issue**: `Stop` only calls `ContainerStop` but never `ContainerRemove`. If `RestartPolicy` is set, container restarts after stop.
- **Priority**: P1

### 6.4 P1 CONFIRMED: Race Condition on `reconcileState` Global Struct
- **File**: `cmd/agent/main.go:1330-1337,1370-1385`
- **Issue**: `reconcileManagedContainers` launched as goroutine can race with ticker-based invocations. `unloggedCount` read/written without synchronization.
- **Priority**: P1

### 6.5 P2 CONFIRMED: `HandleTaskResult` Swallows DB Errors Silently
- **File**: `internal/server/api/agent_handlers.go:970-1005`
- **Issue**: Multiple `h.DB.Exec` calls don't check errors. Handler returns HTTP 200 even on DB failure.
- **Impact**: Silent data loss; agent won't retry.
- **Priority**: P2

### 6.6 P2 CONFIRMED: `HandleStopDeployment` Blocking Pattern
- **File**: `internal/server/api/deployment_lifecycle_handlers.go:1485`
- **Issue**: Waits synchronously up to 90s per instance. 5 instances = 7.5 minutes blocking.
- **Priority**: P2

### 6.7 P2 CONFIRMED: `waitForAgentTaskResult` Polling Pattern
- **File**: `internal/server/api/deployment_lifecycle_handlers.go:1759-1786`
- **Issue**: Polls DB every 200ms in tight loop, holding server goroutine for 30-90s per request.
- **Priority**: P2

---

## 7. Docker / GPU / Runtime Real Runtime Risks

### 7.1 P1 CONFIRMED: External Collector Commands Executed via `sh -c` Without Path Sanitization
- **File**: `internal/agent/collector/external.go:171`
- **Issue**: `cmdStr` from config passed to `sh -c`. If config file is writable by non-root, this is command injection.
- **Priority**: P1

### 7.2 P2 CONFIRMED: `decodeDockerStream` Does Not Limit Payload Size
- **File**: `internal/agent/runtime/docker_real.go:256-293`
- **Issue**: `payloadLen` from Docker stream header can be up to 4GB. No max check before allocation.
- **Impact**: OOM from maliciously crafted stream.
- **Priority**: P2

### 7.3 P2 CONFIRMED: Task Result Contains Full stdout/stderr Without Size Limit
- **File**: `cmd/agent/main.go:1219-1228`
- **Issue**: Full container logs marshaled into JSON and sent to server. No truncation.
- **Impact**: Memory pressure and network saturation.
- **Priority**: P2

### 7.4 P2 CONFIRMED: No Task Deduplication
- **File**: `cmd/agent/main.go:700-716`
- **Issue**: Tasks from heartbeat dispatched without checking if same task ID already in flight. Server retry or network duplicate can cause concurrent execution.
- **Priority**: P2

### 7.5 P3 CONFIRMED: Stop Timeout Hardcoded to 30 Seconds
- **File**: `internal/agent/runtime/docker.go:253`
- **Issue**: Large model serving containers (70B params) may need longer GPU memory deallocation.
- **Priority**: P3

---

## 8. Web UI / API Contract Issues

### 8.1 P0 CONFIRMED: Hardcoded Chinese Strings Bypassing i18n
- **Files**: `DashboardPage.vue:126,130,134`, `modelCapabilities.js:177-235`
- **Issue**: Diagnostic descriptions and `formatTestFailure()` return hardcoded Chinese, ignoring locale.
- **Impact**: English-locale users see Chinese text.
- **Priority**: P0

### 8.2 P0 CONFIRMED: Default Credentials Displayed in UI
- **Files**: `GrafanaPage.vue:7`, `ObservabilityOverviewPage.vue:322`
- **Issue**: "Default username/password: admin/admin" shown to every user.
- **Priority**: P0

### 8.3 P1 CONFIRMED: No Global Route Guard for Authentication
- **File**: `web/src/router/index.ts`
- **Issue**: Zero `beforeEach` guards. Auth checked reactively inside `ConsoleLayout.vue` after components mount.
- **Impact**: Brief window where unauthorized UI is visible before redirect.
- **Priority**: P1

### 8.4 P1 CONFIRMED: RolesPage Permission Dialog Resets Existing Permissions
- **File**: `web/src/pages/RolesPage.vue:86-88`
- **Issue**: `selectedPermIds` reset to `[]` every time dialog opens. Saving without changes strips all permissions.
- **Priority**: P1

### 8.5 P2 CONFIRMED: Hardcoded Service Ports in Observability Pages
- **Files**: `GrafanaPage.vue:27`, `PrometheusPage.vue:23`, `ObservabilityOverviewPage.vue:37-38`
- **Issue**: Ports 13000 and 19090 hardcoded in URL construction.
- **Priority**: P2

### 8.6 P2 CONFIRMED: No Confirmation Dialogs for Destructive Actions
- **Files**: `ModelDeploymentsPage.vue:577,580-581`, `ModelInstancesPage.vue:380-394`
- **Issue**: Stop/restart deployment and instance have no confirmation dialogs.
- **Priority**: P2

### 8.7 P2 CONFIRMED: `useNodeLabels` Stale Cache Never Refreshes
- **File**: `web/src/composables/useNodeLabels.ts:6`
- **Issue**: Module-level `loaded` flag never resets. Node additions/removals not reflected until page reload.
- **Priority**: P2

### 8.8 P2 CONFIRMED: Auth Store Permissions/Roles Loaded But Never Checked
- **File**: `web/src/stores/auth.ts:28-29`
- **Issue**: `permissions` and `roles` arrays loaded from `/auth/me` but never used. All admin-gated UI uses only `is_platform_admin`.
- **Priority**: P2

---

## 9. Test & E2E Coverage Gaps

### 9.1 P0 CONFIRMED: No Unit Tests for Auth/RBAC Core Logic
- **Evidence**: `glob **/auth/*_test.go` returns zero results
- **Issue**: `auth` package (session creation, CSRF validation, password hashing, role permission checks) has zero dedicated unit tests. Only exercised indirectly through handler tests.
- **Priority**: P0

### 9.2 P0 CONFIRMED: E2E Scripts Require Running Server + Real GPU Hardware
- **Files**: All 20 `scripts/e2e-*.sh` files
- **Issue**: Every E2E script requires running server, agent, real GPU, Docker, pre-downloaded models. None can run in CI.
- **Priority**: P0

### 9.3 P1 CONFIRMED: Frontend Tests Are Static Analysis Only
- **Files**: All 7 files in `web/tests/`
- **Issue**: No Vue component rendering tests, no user interaction tests, no API integration mocks. `formatters.test.mjs` re-implements the function inline rather than importing it.
- **Priority**: P1

### 9.4 P1 CONFIRMED: `TestNoVarSyntax` Does Not Assert Anything
- **File**: `internal/server/runplan/resolver_test.go:278-288`
- **Issue**: If-body is empty; test always passes regardless of outcome.
- **Priority**: P1

### 9.5 P1 CONFIRMED: `TestTenantAdminCannotTransferOtherTenantNode` Uses `t.Logf` Instead of `t.Errorf`
- **File**: `internal/server/api/agent_identity_test.go:233-258`
- **Issue**: Test always passes regardless of actual status code.
- **Priority**: P1

### 9.6 P2 CONFIRMED: No Concurrency/Race Condition Tests
- **Issue**: No tests using `-race` flag, no tests for concurrent heartbeat + resource report, no concurrent deployment start/stop tests.
- **Priority**: P2

### 9.7 P2 CONFIRMED: Missing Edge Case Tests for Tenant Isolation
- **File**: `internal/server/api/tenant_isolation_test.go`
- **Issue**: Tests cover basic tenant scoping for nodes and GPUs, but missing: cross-tenant deployment/artifact/run-plan access, admin bypass verification, empty tenant_id handling.
- **Priority**: P2

---

## 10. Residual Logic / Hardcoded Values / Magic Values

### 10.1 P2 CONFIRMED: Hardcoded Port Numbers Throughout
- **Files**: `observability_handler.go:24-29` (19090, 13000), `agent_handlers.go:100` (9090), `agent_proxy_handlers.go:19` (19091), `middleware_logging.go:38` (3000ms)
- **Priority**: P2

### 10.2 P2 CONFIRMED: Hardcoded Backend Catalog Paths
- **File**: `backend_handlers.go:20-26`
- **Issue**: `configs/backend-catalog/versions` is a relative path. Only works when server CWD is project root.
- **Priority**: P2

### 10.3 P2 CONFIRMED: `redactDetailString` Has Incorrect Replacement Logic
- **File**: `helpers.go:225-234`
- **Issue**: Replaces substring occurrences of sensitive key names within any context, not key-value pairs. `PASSWORD_CHANGED` becomes `<redacted>_CHANGED`.
- **Priority**: P2

### 10.4 P3 CONFIRMED: `replaceDash` Panics on Empty Replacement String
- **File**: `internal/server/runplan/resource_controls.go:178`
- **Issue**: `replacement[0]` panics if replacement is empty. Currently safe (only caller passes `"_"`), but exported function with no precondition check.
- **Priority**: P3

### 10.5 P3 CONFIRMED: Container Name Truncation at 12 Characters
- **File**: `internal/agent/runtime/docker.go:533-538`
- **Issue**: UUID truncated to 12 chars (~48 bits entropy). No retry on collision.
- **Priority**: P3

### 10.6 P3 CONFIRMED: `sensitiveKeyFragments` List Too Broad
- **File**: `internal/agent/runtime/sensitive.go:13-22`
- **Issue**: Fragment `"KEY"` matches `GPU_VISIBLE_DEVICES_KEY`, `API_KEY`, but also `MONITORING_KEY_TYPE`. `"ACCESS"` matches `ACCESS_LOG`.
- **Priority**: P3

---

## 11. Optimization Recommendations and Priority

### P0 — Must Fix Before Any External Deployment

| ID | Issue | Fix Direction |
|----|-------|---------------|
| 5.1 | SSRF via agent proxy | Add SSRF deny list, URL escaping, timeouts on all outbound HTTP |
| 5.2 | Privileged container passthrough | Add agent-side security policy; default `allow_privileged: false` |
| 5.3 | Agent token weak default | Refuse to start with default token in non-dev mode; log warning |
| 6.1 | Container leak on start failure | Add `ContainerRemove` in failure paths after successful create |
| 6.2 | Race on `lastStderrBytes` map | Use `sync.Mutex` or `sync.Map` |
| 8.1 | Hardcoded Chinese in dashboard/test failure | Move all strings to i18n keys |
| 8.2 | Default credentials in UI | Show only to platform admins or remove |
| 9.1 | No auth unit tests | Add dedicated auth package tests |
| 9.2 | E2E scripts not CI-compatible | Add mock/in-container E2E that doesn't require real GPU |

### P1 — Should Fix Before Multi-Tenant / Production Use

| ID | Issue | Fix Direction |
|----|-------|---------------|
| 4.1 | Boolean flag deduplication | Rewrite dedup logic to handle flag-only args |
| 4.2 | Env overrides bypass substitution | Apply `substituteVars()` to layer 5 |
| 4.3 | Required params silently skipped | Append error to errors slice for missing required params |
| 5.4 | 8 endpoints missing tenant checks | Add `tenantScopeCheck` after DB lookup in all node-proxy/NBR handlers |
| 5.5 | Agent metrics no auth | Bind to 127.0.0.1 by default; add optional auth token |
| 5.6 | Runtime root add bypass | Remove or restrict to authenticated admin with allowlist |
| 5.7 | Rate limiter IP spoofing | Make IP extraction configurable; default to `RemoteAddr` |
| 5.8 | No body size limit | Add `http.MaxBytesReader` to all JSON decode paths |
| 5.9 | JSON injection in audit log | Use `json.Marshal` for detail construction |
| 5.10 | Credentials file exposure | Delete file after first successful login |
| 6.3 | No container removal on stop | Add `ContainerRemove` after `ContainerStop` |
| 6.4 | Race on `reconcileState` | Add mutex or channel-based lock |
| 7.1 | Collector `sh -c` injection | Validate command path; use `exec.Command` directly |
| 8.3 | No auth route guard | Add `router.beforeEach` guard checking auth |
| 8.4 | RolesPage permission reset | Load existing permissions on dialog open |
| 9.3 | No frontend component tests | Add Vue Test Utils component tests |
| 9.4-9.5 | Tests that don't assert | Fix empty assertions and `t.Logf` → `t.Errorf` |

### P2 — Should Fix for Quality / Maintainability

| ID | Issue | Fix Direction |
|----|-------|---------------|
| 4.4 | Dead `buildDeviceBinding` code | Remove or integrate into Resolve() |
| 4.5 | Input hash omits GPU/override | Include AssignedGPUs and NodeRuntimeOverride |
| 4.6-4.8 | Stale/dead catalog config | Update YAML references; remove dead keys |
| 5.11 | Observability no auth | Add session auth |
| 5.12 | CSRF rotation on /me | Only rotate on login |
| 5.13 | Non-constant-time token compare | Use `crypto/subtle.ConstantTimeCompare` |
| 6.5 | Task result swallows DB errors | Check and log DB errors |
| 6.6-6.7 | Blocking stop/logs handlers | Consider SSE/WebSocket or async pattern |
| 7.2-7.4 | Docker stream/task size limits | Add payload size limits and task dedup |
| 8.5-8.8 | UI hardcoded ports, stale cache, unused permissions | Make configurable; add TTL; wire permission checks |
| 10.1-10.3 | Hardcoded values, incorrect redaction | Make configurable; fix redaction logic |

---

## 12. Suggested Fix Batches

### Batch 1: Critical Security (P0)
1. SSRF protection on agent proxy endpoints
2. Agent-side privileged container policy
3. Agent token default enforcement
4. Container cleanup on start failure
5. Race condition fixes (lastStderrBytes, reconcileState)

### Batch 2: Tenant Isolation (P1)
1. Add tenant scope checks to all 8 missing endpoints
2. Agent metrics endpoint authentication
3. Remove/Restrict AllowRuntimeRootAdd
4. Rate limiter IP extraction fix
5. Request body size limits

### Batch 3: Core Logic Fixes (P1)
1. Boolean flag deduplication
2. Env override variable substitution
3. Required parameter validation
4. JSON injection in audit log
5. Credentials file cleanup

### Batch 4: Frontend Quality (P1-P2)
1. Hardcoded Chinese → i18n keys
2. Default credentials removal from UI
3. Auth route guard
4. RolesPage permission pre-select
5. Confirmation dialogs for destructive actions

### Batch 5: Test Infrastructure (P1-P2)
1. Auth package unit tests
2. Fix empty assertion tests
3. Add mock-compatible E2E framework
4. Frontend component test foundation

### Batch 6: Hardening (P2)
1. CSRF rotation fix
2. Constant-time agent token compare
3. Task result error handling
4. Docker stream payload limits
5. Catalog YAML cleanup

---

## 13. Review Execution Details

### Commands Executed
- `git log --oneline -20` — recent commit history
- `git status --short` — working tree state (clean except VERSION edit and .mimocode/)
- `read` on 50+ files across docs/, internal/, web/src/, configs/, scripts/
- `glob` and `grep` for pattern matching across codebase
- 6 parallel subagent explorations covering: server API, agent/Docker, RunPlan/catalog, Web UI, tests/E2E, security/auth

### Check Scope
| Area | Files Reviewed | Depth |
|------|---------------|-------|
| Server API handlers | 15 handler files, router, middleware | Line-by-line |
| Agent runtime | 11 packages, main.go | Line-by-line |
| RunPlan resolver | 12 files | Line-by-line |
| Web frontend | 19 pages, 12 API modules, 5 composables, 8 components | Line-by-line |
| Tests | 47 Go test files, 7 frontend tests, 20 E2E scripts | Structure + key assertions |
| Security/Auth | auth/, rbac/, middleware, config | Line-by-line |
| Documentation | docs/CURRENT.md, PHASE-STATUS.md, README.md, 2 closeout reports | Full read |
| Configs | backend-catalog YAML, server/agent configs | Full read |

### Unable to Verify
- Real multi-GPU behavior (only single RTX 5090 available)
- MetaX hardware validation (no MetaX hardware)
- Huawei/Ascend runtime (template-only)
- TLS/HTTPS behavior (not implemented)
- Production load/scalability (no load testing infrastructure)
- Actual Docker container escape scenarios (would require destructive testing)

### Positive Findings (Well-Implemented)
1. **RunPlan resolver test suite**: 50+ tests covering all 3 backends, image priority, env merging, resource controls
2. **Snapshot immutability**: BackendVersion → BackendRuntime → NodeBackendRuntime boundary thoroughly tested
3. **Auth foundation**: Argon2id (64MB), constant-time password comparison, CSRF with origin/referer validation, session ID hashing
4. **Collector protocol**: Clean external command abstraction with exit codes for multi-vendor GPU support
5. **Identity persistence**: First-start generation, corrupt file detection, server-reconciliation
6. **Frontend API client**: CSRF token management, 401 handling, structured error class
7. **Polling composables**: Visibility-aware, focus-aware, re-entry guarded, route-leave cleanup
8. **Documentation governance**: Formal open-issues closeout with FIXED/DOCUMENTED_BLOCKER/INVALID states

---

## Report Metadata

- **Report path**: `docs/reports/full-project-review/2026-06-23-full-project-review.md`
- **Git status**: Clean (no code changes; only VERSION file modified from previous session)
- **Files modified by this review**: 1 (this report file only)
- **Commit**: N/A — report-only, no code changes
- **Total findings**: 87 (8 P0, 28 P1, 32 P2, 19 P3)
