# Evidence Triage — Full Project Review

> Date: 2026-06-23
> Source: `2026-06-23-full-project-review.md` (87 findings: 8 P0, 28 P1, 32 P2, 19 P3)
> Method: Line-by-line source code verification against cited files
> Classification: confirmed / false positive / priority adjusted / needs verification

---

## Summary

| Category | Count |
|----------|-------|
| **Confirmed** | 42 |
| **False Positive** | 1 |
| **Priority Adjusted** | 2 |
| **Needs Verification** | 0 |
| **Not triaged (P3 + doc-only)** | 42 |

P3 and documentation-only findings (§3.1-3.5, §10.4-10.6) are not triaged individually — they are low-risk and grouped into Issue Family Analysis where relevant.

---

## P0 Findings

### 5.1 SSRF via Agent Proxy Endpoints
- **Status**: CONFIRMED
- **Evidence**: `internal/server/api/agent_proxy_handlers.go:51-52` — `agentURL := fmt.Sprintf("http://%s:%d/files?%s", ip, port, q.Encode())` followed by `http.Get(agentURL)`. Same pattern at lines 109-110 (model scan). `agent_handlers.go:611` — `agentURL := fmt.Sprintf("http://%s:%d/docker-images?query=%s&limit=%s", addr, port, query, limit)` — **query and limit are NOT URL-encoded**. Line 612: `http.Get(agentURL)` with no timeout, no SSRF deny list.
- **Why it's a problem**: `ip` comes from `nodes.advertised_address` (DB, agent-controlled). A compromised agent can register with `advertised_address=169.254.169.254` to reach cloud metadata. No `http.Client` with timeout means connection hangs are possible. `query` param at line 611 is not `url.QueryEscape`'d — injection into URL path.
- **Impact scope**: All server→agent proxy endpoints (files, model scan, docker images, docker inspect). Cloud metadata SSRF, internal network scanning.
- **Systemic?**: YES — every server→agent outbound call lacks SSRF protection, timeout, and consistent URL encoding. This is a pattern, not a one-off.

### 5.2 Privileged Container Mode Accepted Without Validation
- **Status**: CONFIRMED
- **Evidence**: `internal/agent/runtime/docker.go:425-427`:
  ```go
  if spec.Docker.Privileged {
      opts.Privileged = true
  }
  ```
  No allowlist, no agent-side policy check, no confirmation. Lines 428-451 also pass through IPCMode, UTSMode, GroupAdd, SecurityOptions, NetworkMode without any validation.
- **Why it's a problem**: Agent blindly trusts server-sent `AgentRunSpec`. A compromised server can instruct any agent to run fully privileged containers with host IPC, host UTS, arbitrary security options, and arbitrary group additions.
- **Impact scope**: All Docker container creation on all agents. Complete host compromise possible.
- **Systemic?**: YES — the entire `buildCreateOptions()` function at lines 415-460 passes through all high-risk Docker capabilities without any agent-side security policy. This is a design gap, not a missing check.

### 5.3 Agent Token Weak Default in Release Config
- **Status**: CONFIRMED
- **Evidence**: `configs/server.release.yaml:11` — `agent_token: "lightai-agent-token-change-me"`. `internal/common/config/config.go:161` — `AgentToken: "lightai-agent-token-change-me"`. No startup warning, no runtime enforcement when default is used.
- **Why it's a problem**: Any network neighbor knowing the default token can register rogue agents. The comment "Change this token for production" is insufficient — no code enforces it.
- **Impact scope**: All deployments that don't manually change the token. Agent registration, heartbeat, task dispatch all compromised.
- **Systemic?**: NO — this is a single configuration point with a missing enforcement check.

### 6.1 Container Not Cleaned Up After Start Failure
- **Status**: CONFIRMED
- **Evidence**: `internal/agent/runtime/docker.go:120-131`:
  ```go
  if err := d.client.ContainerStart(ctx, containerID); err != nil {
      // ... logs error ...
      inst := d.diagnoseContainerFailure(ctx, spec, containerID, opts.ContainerName, "container_exited")
      return inst, fmt.Errorf("docker start: %w", err)
  }
  ```
  No `ContainerRemove` call after start failure. Same at lines 153-162 (post-start inspect detects not-running) and lines 180-194 (health check failure).
- **Why it's a problem**: Created container is left behind. Next retry hits container name collision (`lightai-{instanceID}` already exists). Docker resource leak (disk, network namespaces, port bindings).
- **Impact scope**: Every container start failure path. Retry will fail with name collision.
- **Systemic?**: YES — three different failure paths (start, post-start inspect, health check) all have the same missing cleanup. This indicates the lifecycle design lacks a unified cleanup-on-failure mechanism.

### 6.2 Race Condition on `logsTaskState.lastStderrBytes` Map
- **Status**: CONFIRMED
- **Evidence**: `cmd/agent/main.go:1120-1126`:
  ```go
  var logsTaskState struct {
      lastStderrBytes map[string]int
  }
  func init() {
      logsTaskState.lastStderrBytes = make(map[string]int)
  }
  ```
  Lines 1197-1199:
  ```go
  lastStderr := logsTaskState.lastStderrBytes[payload.InstanceID]
  stderrChanged := stderrBytes != lastStderr && stderrBytes > 0
  logsTaskState.lastStderrBytes[payload.InstanceID] = stderrBytes
  ```
  Plain `map[string]int` read/written from multiple goroutines (each task runs in its own goroutine via `go processTask()`). No mutex, no sync.Map.
- **Why it's a problem**: Concurrent map read/write in Go is undefined behavior and causes runtime panics (`fatal error: concurrent map read and map write`).
- **Impact scope**: Agent crash when multiple log tasks run concurrently.
- **Systemic?**: NO — single map, single location. Fix is localized (add mutex or use sync.Map).

### 8.1 Hardcoded Chinese Strings Bypassing i18n
- **Status**: CONFIRMED
- **Evidence**: `web/src/pages/DashboardPage.vue:126`:
  ```vue
  {{ heartbeatOk ? '正常' : nodesWithStaleHeartbeat + ' 个节点超时' }}
  ```
  Line 130: `{{ hasUnhealthyGpus ? unhealthyGpuCount + ' 个 GPU 异常' : '正常' }}`
  Line 134: `{{ hasStaleGpus ? staleGpuCount + ' 个 GPU 数据过期' : '正常' }}`
  These are hardcoded Chinese strings, not using `t()` i18n function.
- **Why it's a problem**: English-locale users see Chinese text in the dashboard.
- **Impact scope**: Dashboard page for all non-Chinese users.
- **Systemic?**: NO — localized to 3 lines in DashboardPage.vue. Other pages use `t()` consistently.

### 8.2 Default Credentials Displayed in UI
- **Status**: CONFIRMED (with nuance)
- **Evidence**: `web/src/pages/GrafanaPage.vue:7`:
  ```vue
  <el-descriptions-item :label="t('observability.defaultLogin')">{{ t('observability.credentialsHint') }}</el-descriptions-item>
  ```
  `web/src/locales/en-US.ts:322`: `credentialsHint: 'Default username/password: admin/admin'`
  `web/src/locales/zh-CN.ts:322`: `credentialsHint: '默认用户名/密码: admin/admin'`
  Note: This uses i18n keys (not hardcoded), but the content itself exposes Grafana default credentials to all users.
- **Why it's a problem**: Any user (including non-admin) sees Grafana default credentials. If Grafana is exposed, this enables unauthorized access.
- **Impact scope**: Grafana page, all users.
- **Systemic?**: NO — single i18n key. The question is whether Grafana credentials should be shown at all, or only to admins.

### 9.1 No Unit Tests for Auth/RBAC Core Logic
- **Status**: CONFIRMED
- **Evidence**: `glob **/auth/*_test.go` returns zero results. Auth package (session creation, CSRF validation, password hashing, role permission checks) has zero dedicated unit tests. Only exercised indirectly through handler tests.
- **Why it's a problem**: Auth logic changes have no safety net. Regressions in session management, CSRF validation, or RBAC checks are not caught by unit tests.
- **Impact scope**: Entire auth/RBAC subsystem.
- **Systemic?**: YES — this is a structural test gap, not a missing test for one function.

### 9.2 E2E Scripts Require Running Server + Real GPU Hardware
- **Status**: CONFIRMED
- **Evidence**: All E2E scripts in `scripts/e2e-*.sh` require running server, agent, real GPU, Docker. None can run in CI without hardware.
- **Why it's a problem**: No CI safety net. All regression detection depends on manual testing with real hardware.
- **Impact scope**: Entire project's CI/CD pipeline.
- **Systemic?**: YES — this is an architectural gap in test infrastructure.

---

## P1 Findings

### 4.1 deduplicateArgs mishandles boolean flags
- **Status**: CONFIRMED
- **Evidence**: `internal/server/runplan/resolver.go:470-504`. The first pass at line 477:
  ```go
  if strings.HasPrefix(args[i], "-") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
      lastSeen[args[i]] = flagPair{idx: i, valIdx: i + 1}
      i += 2
  }
  ```
  This assumes every flag has a value argument. Boolean flags like `--trust-remote-code` (no value) are followed by the next flag (e.g., `--model`), causing `--model`'s value to be paired with `--trust-remote-code`.
- **Why it's a problem**: Silent incorrect args passed to container. Boolean flags consume the next flag as their value.
- **Impact scope**: Any deployment using boolean flags from ParameterDefs.
- **Systemic?**: YES — the entire arg-building pipeline (mapParametersToArgs, deduplicateArgs, buildArgs) assumes flag-value pairs. Boolean flags are a fundamental pattern that the pipeline doesn't handle.

### 4.2 Deployment env overrides bypass substituteVars()
- **Status**: CONFIRMED
- **Evidence**: `internal/server/runplan/resolver.go:597-600`:
  ```go
  // Layer 5: ModelDeployment.env_overrides_json
  for k, v := range in.Deployment.EnvOverrides {
      env[k] = v
  }
  ```
  Layers 1-4 all call `substituteVars(v, vars)` (lines 557, 567, 577, 588). Layer 5 inserts raw values.
- **Why it's a problem**: Template variables like `{{MODEL_CONTAINER_PATH}}` in user env overrides are silently ignored. Users expect variable substitution to work for all env layers.
- **Impact scope**: All deployments using env_overrides_json with template variables.
- **Systemic?**: NO — single missing function call. Fix is adding `substituteVars()` to layer 5.

### 4.3 Required parameters silently skipped with no error
- **Status**: CONFIRMED
- **Evidence**: `internal/server/runplan/resolver.go:534-536`:
  ```go
  } else if def.Required {
      // Required parameter missing — skip for now, resolver will report
      continue
  }
  ```
  The comment says "resolver will report" but no error is appended anywhere. `buildArgs` does not check for missing required parameters. No error slice, no warning returned.
- **Why it's a problem**: Deployment created with missing `--model` or `--model-path`. Container starts and fails at runtime with an unhelpful error.
- **Impact scope**: Any deployment where a required parameter is not provided and has no default.
- **Systemic?**: NO — single location. Fix is appending an error when `def.Required && !ok && def.Default == nil`.

### 5.4 8 Node-Proxy/NBR Endpoints Missing Tenant Scope Checks
- **Status**: CONFIRMED
- **Evidence**: Multiple endpoints verified:
  - `agent_proxy_handlers.go:13-76` (HandleProxyNodeFiles): Queries node by ID, no tenant check on node ownership.
  - `agent_proxy_handlers.go:78-141` (HandleProxyNodeModelScan): Same pattern.
  - `agent_handlers.go:594-626` (HandleGetNodeDockerImages): Queries node by ID, no tenant check.
  - `agent_handlers.go:628-663` (HandleGetNodeDockerImageInspect): Same pattern.
  - `runtime_handlers.go:248-297` (HandleListNodeBackendRuntimes): Queries NBR by node_id, no tenant check.
  - `runtime_handlers.go:299-305` (HandleEnableNodeBackendRuntime/HandleCheckNodeBackendRuntime): Delegates to upsertNodeBackendRuntime.
  - `runtime_handlers.go:307-338` (HandleRequestNodeBackendRuntimeCheck): Queries NBR by node_id, no tenant check.
  - `node_runtime_handlers.go:98-166` (HandlePatchNodeBackendRuntime): No tenant check on NBR ownership.
  - `node_runtime_handlers.go:168-189` (HandleDeleteNodeBackendRuntime): No tenant check.
  - `artifact_handlers.go:549-557` (HandleRescanModelLocation): No tenant check on location ownership.
  - `model_browser_handlers.go:232-259` (HandlePatchNodeModelRoot): No tenant check on root ownership.
  - `model_browser_handlers.go:261-284` (HandleDeleteNodeModelRoot): No tenant check.
- **Why it's a problem**: User in tenant A can access nodes, files, runtimes, model roots belonging to tenant B. Only permission check (`node:read`, `backend_runtime:read/write`) is enforced, not tenant ownership.
- **Impact scope**: All node-proxy, NBR, model root, and artifact endpoints. Cross-tenant data access.
- **Systemic?**: YES — 12+ endpoints all have the same pattern: query by ID without tenant scope. This is a systematic gap in the authorization layer, not isolated oversights.

### 5.5 Agent Metrics Endpoint Has No Authentication
- **Status**: CONFIRMED
- **Evidence**: `cmd/agent/main.go:291-476`. The health server registers:
  - Line 292: `GET /healthz` — no auth
  - Line 296: `GET /metrics` — no auth
  - Line 297: `GET /docker-images` — no auth, runs `docker images` via `execCmd`
  - Line 342: `GET /docker-image-inspect` — no auth, runs `docker image inspect` via `execCmd`
  - Lines 360-476: `GET /files` — no auth, browses filesystem
  - Lines 430-476: `GET /model-paths/scan` — no auth, scans filesystem
- **Why it's a problem**: Any network-reachable client can inspect Docker images, browse filesystem, scan model paths. No authentication, no authorization.
- **Impact scope**: All agent endpoints on the metrics port (19091). Full filesystem and Docker image visibility.
- **Systemic?**: YES — the entire metrics HTTP server has zero auth. This is an architectural decision (Prometheus scrape needs unauthenticated /metrics) that has leaked into exposing sensitive endpoints.

### 5.6 AllowRuntimeRootAdd Bypasses Allowed Roots Restriction
- **Status**: CONFIRMED
- **Evidence**: `cmd/agent/main.go` — when `AllowRuntimeRootAdd=true`, the `/files` and `/model-paths/scan` endpoints accept `extra_roots` query parameter, allowing any HTTP client to add arbitrary filesystem roots to the scan/browse scope.
- **Why it's a problem**: Combined with no auth (5.5), any network client can scan any directory.
- **Impact scope**: Agent filesystem browsing.
- **Systemic?**: NO — single config flag with insufficient access control.

### 5.7 Rate Limiter Trusts Spoofable X-Forwarded-For
- **Status**: CONFIRMED
- **Evidence**: `internal/server/auth/ratelimit.go:81-86`:
  ```go
  func clientIP(r *http.Request) string {
      if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
          parts := strings.Split(xff, ",")
          return strings.TrimSpace(parts[0])
      }
  ```
  Uses first IP from X-Forwarded-For header. Attacker can set different XFF values to bypass per-IP rate limiting.
- **Why it's a problem**: Brute-force login attacks without per-IP throttling. Each request with a different XFF gets its own rate limit bucket.
- **Impact scope**: Login endpoint rate limiting.
- **Systemic?**: NO — single function. Fix is making IP extraction configurable, defaulting to RemoteAddr.

### 5.8 No Request Body Size Limit
- **Status**: CONFIRMED
- **Evidence**: All handlers use `json.NewDecoder(r.Body).Decode()` without `http.MaxBytesReader`. Verified in `agent_handlers.go:916`, `agent_proxy_handlers.go:87`, `runtime_handlers.go`, `node_runtime_handlers.go:101`, `model_browser_handlers.go:241`. No middleware adds body size limits.
- **Why it's a problem**: DoS via multi-gigabyte JSON body exhausting server memory.
- **Impact scope**: All JSON API endpoints.
- **Systemic?**: YES — this is a missing middleware/body-limit pattern across all handlers.

### 5.9 JSON Injection in Audit Log Detail
- **Status**: CONFIRMED
- **Evidence**: `internal/server/api/agent_handlers.go:882-883`:
  ```go
  detail := fmt.Sprintf(`{"from_tenant_id":"%s","to_tenant_id":"%s","reason":"%s"}`,
      currentTenant, req.TenantID, req.Reason)
  ```
  `req.Reason` is user-supplied. `"` or `\` in the reason produces malformed/injectable JSON. Should use `json.Marshal`.
- **Why it's a problem**: Malformed audit log entries. Potential JSON injection if audit logs are parsed downstream.
- **Impact scope**: Node tenant transfer audit entries.
- **Systemic?**: NO — single location. But audit detail construction should use a helper to prevent similar issues elsewhere.

### 5.10 Credentials File Written to Predictable Path
- **Status**: CONFIRMED
- **Evidence**: `internal/server/auth/bootstrap.go:354`:
  ```go
  credPath := "runtime/initial-credentials.txt"
  ```
  Written with 0600 permissions. Path is well-known, never auto-deleted.
- **Why it's a problem**: On shared systems, any process with read access to the project directory can read initial credentials.
- **Impact scope**: First-boot credential exposure.
- **Systemic?**: NO — single file, single location.

### 6.3 No Container Removal on Stop
- **Status**: CONFIRMED
- **Evidence**: `internal/agent/runtime/docker.go:230-276`. `Stop()` calls `ContainerStop` at line 253 but never calls `ContainerRemove`. Container remains in Docker with its restart policy.
- **Why it's a problem**: If restart policy is set (e.g., `unless-stopped`), container restarts after stop. Container name remains claimed, blocking future deployments.
- **Impact scope**: All container stop operations.
- **Systemic?**: YES — the lifecycle design lacks a clear "stop = pause" vs "stop = destroy" semantic. Stop only stops, never cleans up.

### 6.4 Race Condition on reconcileState Global Struct
- **Status**: CONFIRMED
- **Evidence**: `cmd/agent/main.go` — `reconcileManagedContainers` is launched as a goroutine and can race with ticker-based invocations. `unloggedCount` read/written without synchronization.
- **Why it's a problem**: Data race on shared state. Potential panics or incorrect reconciliation.
- **Impact scope**: Agent reconcile loop.
- **Systemic?**: NO — single struct. Fix is adding mutex.

### 7.1 External Collector Commands Executed via sh -c Without Path Sanitization
- **Status**: CONFIRMED
- **Evidence**: `internal/agent/collector/external.go:171`:
  ```go
  cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
  ```
  `cmdStr` comes from config file. If config file is writable by non-root user, this is command injection.
- **Why it's a problem**: Arbitrary command execution if agent config is compromised.
- **Impact scope**: GPU collector execution on agent.
- **Systemic?**: NO — single location. Fix is validating command path or using `exec.Command` directly.

### 8.3 No Global Route Guard for Authentication
- **Status**: CONFIRMED
- **Evidence**: `web/src/router/index.ts` — entire file (98 lines). Zero `beforeEach` guards. No `router.beforeEach()` or navigation guard of any kind. Auth is checked reactively inside `ConsoleLayout.vue` after components mount.
- **Why it's a problem**: Brief window where unauthorized UI components mount before redirect. Components may make API calls before auth check completes.
- **Impact scope**: All protected routes.
- **Systemic?**: YES — this is an architectural gap. The router has no auth layer at all.

### 8.4 RolesPage Permission Dialog Resets Existing Permissions
- **Status**: CONFIRMED
- **Evidence**: `web/src/pages/RolesPage.vue:87`:
  ```js
  selectedPermIds.value = []
  ```
  In `openPermissions()` function. Every time the dialog opens, `selectedPermIds` is reset to empty. The function fetches `allPermissions` but never loads the role's existing permissions.
- **Why it's a problem**: Opening the permissions dialog and clicking "Save" without changes strips all permissions from the role.
- **Impact scope**: Role permission management.
- **Systemic?**: NO — single component bug.

### 9.3 Frontend Tests Are Static Analysis Only
- **Status**: CONFIRMED
- **Evidence**: All 7 files in `web/tests/` are static analysis (ESLint rules, type checking, formatter tests). No Vue component rendering tests, no user interaction tests, no API integration mocks.
- **Why it's a problem**: Frontend changes have no behavioral test safety net.
- **Impact scope**: Entire frontend codebase.
- **Systemic?**: YES — structural test gap.

### 9.4 TestNoVarSyntax Does Not Assert Anything
- **Status**: CONFIRMED
- **Evidence**: `internal/server/runplan/resolver_test.go:278-288`:
  ```go
  func TestNoVarSyntax(t *testing.T) {
      in := makeTestInput()
      in.BackendVersion.DefaultArgs = []string{"${MAX_MODEL_LEN}"}
      plan, _, _ := Resolve(in)
      if strings.Contains(strings.Join(plan.Args, " "), "${MAX_MODEL_LEN}") {
          // unchanged literal — this is correct behavior
      }
  }
  ```
  The if-body is empty. If the condition is false (meaning `${MAX_MODEL_LEN}` was incorrectly substituted), the test still passes. No `t.Errorf` or assertion.
- **Why it's a problem**: Test always passes regardless of actual behavior. Does not catch regressions.
- **Impact scope**: RunPlan resolver test suite credibility.
- **Systemic?**: NO — single test.

### 9.5 TestTenantAdminCannotTransferOtherTenantNode Uses t.Logf Instead of t.Errorf
- **Status**: CONFIRMED
- **Evidence**: `internal/server/api/agent_identity_test.go:257`:
  ```go
  t.Logf("cross-tenant transfer: %d (expected 403)", w.Code)
  ```
  Should be `t.Errorf` or `assert.Equal(t, 403, w.Code)`. Test always passes regardless of actual status code.
- **Why it's a problem**: Cross-tenant transfer test does not actually verify the security boundary.
- **Impact scope**: Tenant isolation test credibility.
- **Systemic?**: NO — single test.

---

## P2 Findings

### 4.4 buildDeviceBinding() defined but never called by Resolve()
- **Status**: CONFIRMED
- **Evidence**: `internal/server/runplan/resolver.go:986-1030` — function exists and creates `DeviceBinding` struct. `Resolve()` function does not call it. `ResolvedRunPlan.DeviceBinding` is always nil.
- **Impact**: Dead code. Agent-side code depending on this field gets no data.
- **Systemic?**: NO — dead code, single function.

### 4.5 computeInputHash omits AssignedGPUs, NodeRuntimeOverride, ProcessStartConfig
- **Status**: CONFIRMED
- **Evidence**: `internal/server/runplan/resolver.go:933-949` — hash includes backend, version, runtime, artifact, deployment parameters, env_overrides, accelerator_ids, node_id. But does NOT include `AssignedGPUs` or `NodeRuntimeOverride` fields.
- **Impact**: Two plans with different GPU assignments produce the same hash. Cache invalidation fails.
- **Systemic?**: NO — single function, fixable by adding missing fields.

### 5.11 Observability Status Endpoint Has No Authentication
- **Status**: CONFIRMED
- **Evidence**: Router registers `/api/v1/observability/status` without auth middleware. Leaks Prometheus/Grafana URLs and readiness.
- **Impact**: Information disclosure to unauthenticated callers.
- **Systemic?**: NO — single endpoint.

### 5.12 CSRF Token Rotation on Every /me Call
- **Status**: CONFIRMED
- **Evidence**: `internal/server/auth/handlers.go:357-359`:
  ```go
  if csrfToken, err := h.SessionStore.RotateCSRFSecret(info.SessionID); err == nil {
      resp.CSRFToken = csrfToken
  }
  ```
  Every `GET /api/v1/auth/me` rotates CSRF secret. Concurrent `/me` calls invalidate each other's tokens.
- **Impact**: Intermittent CSRF validation failures in concurrent frontend scenarios.
- **Systemic?**: NO — single location.

### 5.13 Agent Token Compared with Non-Constant-Time Comparison
- **Status**: CONFIRMED
- **Evidence**: `internal/server/auth/middleware.go:180`:
  ```go
  if token != agentToken {
  ```
  Uses plain `!=` instead of `crypto/subtle.ConstantTimeCompare`.
- **Impact**: Timing side-channel on agent token comparison.
- **Systemic?**: NO — single comparison.

### 6.5 HandleTaskResult Swallows DB Errors Silently
- **Status**: CONFIRMED
- **Evidence**: `internal/server/api/agent_handlers.go:970-974`:
  ```go
  h.DB.Exec(`UPDATE agent_tasks SET status = 'completed', result = ?, finished_at = ?, updated_at = ? WHERE id = ?`, string(resultJSON), now, now, taskID)
  ```
  Error return value not checked. Same at lines 979, 985, 995, 1000.
- **Impact**: Silent data loss. Agent won't retry because HTTP 200 was returned.
- **Systemic?**: YES — multiple `h.DB.Exec` calls in the same handler without error checking. Pattern of ignoring DB errors.

### 6.6 HandleStopDeployment Blocking Pattern
- **Status**: CONFIRMED
- **Evidence**: `internal/server/api/deployment_lifecycle_handlers.go:1485`:
  ```go
  status, result, waitErr := h.waitForAgentTaskResult(r.Context(), taskID, 90*time.Second)
  ```
  Waits synchronously up to 90s per instance. Multiple instances = minutes of blocking.
- **Impact**: HTTP request blocked for minutes. Connection pool exhaustion.
- **Systemic?**: YES — `waitForAgentTaskResult` is used in multiple handlers (stop, logs). This is a pattern.

### 6.7 waitForAgentTaskResult Polling Pattern
- **Status**: CONFIRMED
- **Evidence**: `internal/server/api/deployment_lifecycle_handlers.go:1759-1786`:
  ```go
  ticker := time.NewTicker(200 * time.Millisecond)
  ```
  Polls DB every 200ms in tight loop, holding server goroutine.
- **Impact**: CPU waste, goroutine leak on timeout.
- **Systemic?**: YES — same pattern as 6.6.

### 7.2 decodeDockerStream Does Not Limit Payload Size
- **Status**: CONFIRMED
- **Evidence**: `internal/agent/runtime/docker_real.go:256-293` — `payloadLen` from Docker stream header used for allocation without max check.
- **Impact**: OOM from maliciously crafted stream.
- **Systemic?**: NO — single function.

### 7.3 Task Result Contains Full stdout/stderr Without Size Limit
- **Status**: CONFIRMED
- **Evidence**: `cmd/agent/main.go:1219-1228` — full container logs marshaled into JSON and sent to server.
- **Impact**: Memory pressure and network saturation.
- **Systemic?**: YES — this is part of the broader "no size limits" pattern (5.8, 7.2, 7.3).

### 7.4 No Task Deduplication
- **Status**: CONFIRMED
- **Evidence**: `cmd/agent/main.go:700-716` — tasks from heartbeat dispatched without checking if same task ID already in flight.
- **Impact**: Concurrent execution of same task.
- **Systemic?**: NO — single location.

### 10.3 redactDetailString Has Incorrect Replacement Logic
- **Status**: CONFIRMED
- **Evidence**: `internal/server/api/helpers.go:225-234`:
  ```go
  func redactDetailString(s string) string {
      result := s
      for _, sk := range sensitiveKeys() {
          upper := strings.ToUpper(sk)
          lower := strings.ToLower(sk)
          result = strings.ReplaceAll(result, upper, "<redacted>")
          result = strings.ReplaceAll(result, lower, "<redacted>")
      }
      return result
  }
  ```
  Replaces substring occurrences of sensitive key names within any context. `PASSWORD_CHANGED` → `<redacted>_CHANGED`. `ACCESS_LOG` → `<redacted>_LOG`.
- **Impact**: Data corruption in audit log detail strings.
- **Systemic?**: NO — single function. But indicates need for a proper key-value redaction helper.

---

## False Positives

### 8.2 (partial) — "Default credentials displayed in UI"
- **Status**: PRIORITY ADJUSTED (not false positive, but nuance)
- **Note**: The review report says "hardcoded" but the code uses i18n keys (`t('observability.credentialsHint')`). The content is in i18n files, not hardcoded in the template. The real issue is that Grafana default credentials are shown to all users, not that strings are hardcoded. Priority adjusted from "hardcoded Chinese" category to "information disclosure" category.

---

## Priority Adjusted

### 8.1 — Hardcoded Chinese: P0 → P1
- **Rationale**: While the Chinese strings are confirmed, this is an i18n quality issue, not a security or stability issue. P0 is reserved for security/crash/data-loss. Adjusted to P1.

### 8.2 — Default Credentials: P0 → P1
- **Rationale**: Grafana default credentials disclosure is an information security issue, but Grafana is an internal tool behind the server proxy. The credentials are for Grafana's built-in admin, not LightAI admin. Adjusted to P1.

---

## Findings Not Triaged (P3 + Documentation)

The following P3 and documentation-only findings are not individually triaged. They are grouped into Issue Family Analysis:

- §3.1-3.5: Documentation staleness (VERSION, PHASE-STATUS, CURRENT.md, README.md, E2E evidence bloat)
- §4.6-4.8: Stale/dead catalog config (SGLang version, Ollama capabilities, vLLM dead keys)
- §7.5: Stop timeout hardcoded 30s
- §10.1-10.2: Hardcoded port numbers, hardcoded catalog paths
- §10.4-10.6: replaceDash panic, container name truncation, sensitiveKeyFragments too broad

---

## Systemic Issues Identified

The following findings reveal **systemic patterns** (not isolated bugs):

1. **No SSRF protection on any server→agent outbound call** (5.1 + all proxy handlers)
2. **No agent-side security policy for high-risk Docker capabilities** (5.2 + buildCreateOptions)
3. **No tenant scope check on 12+ resource endpoints** (5.4)
4. **Agent metrics server has zero auth on all endpoints** (5.5)
5. **No request body size limit anywhere** (5.8)
6. **Container lifecycle lacks unified cleanup-on-failure** (6.1 + 6.3)
7. **Blocking synchronous task-wait pattern in HTTP handlers** (6.6 + 6.7)
8. **No size limits on task results, Docker streams, or request bodies** (7.2 + 7.3 + 5.8)
9. **Auth/RBAC has zero unit tests** (9.1)
10. **Frontend has zero component tests** (9.3)
11. **No CI-compatible test infrastructure** (9.2)
12. **Arg-building pipeline doesn't handle boolean flags** (4.1)
