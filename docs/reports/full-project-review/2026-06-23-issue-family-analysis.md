# Issue Family Analysis — Full Project Review

> Date: 2026-06-23
> Source: Evidence triage (`2026-06-23-evidence-triage.md`) + source code analysis
> Purpose: Group 87 findings into problem families, assess systemic nature, determine refactoring need

---

## Overview

The 87 findings are not 87 independent bugs. They fall into 9 problem families, most of which reveal **systemic design gaps** rather than isolated oversights. The key insight: many findings share root causes in missing abstractions, missing middleware, or incomplete lifecycle design.

| Family | Findings | Systemic? | Needs Refactoring? |
|--------|----------|-----------|-------------------|
| A. Access Control / Tenant Scope | 5.4, 5.6, 8.3, 8.4, 8.8 | YES | YES — unified tenant scope middleware |
| B. Server→Agent Proxy / SSRF | 5.1, 5.11 | YES | YES — unified agent client |
| C. Agent Security Boundary | 5.2, 5.3, 5.5, 5.6, 7.1 | YES | YES — agent security policy |
| D. Docker Lifecycle / State Machine | 6.1, 6.3, 7.5 | YES | YES — lifecycle state machine |
| E. Concurrency / Race / Task Dispatch | 6.2, 6.4, 7.4 | PARTIAL | PARTIAL — targeted fixes + task registry |
| F. Input / Output / Log Safety | 5.8, 5.9, 7.2, 7.3, 10.3 | YES | YES — unified body/stream helpers |
| G. RunPlan / Runtime Config | 4.1, 4.2, 4.3, 4.4, 4.5, 4.6-4.8, 10.1-10.2 | YES | YES — resolver refactor |
| H. Web UI / API Contract / i18n | 8.1, 8.2, 8.3, 8.4, 8.5-8.8 | PARTIAL | PARTIAL — route guard + i18n cleanup |
| I. Test / E2E Credibility | 9.1-9.7 | YES | YES — test infrastructure |

---

## Family A: Access Control / Tenant Scope / RBAC

### Findings
- 5.4: 12+ endpoints missing tenant scope checks
- 5.6: AllowRuntimeRootAdd bypasses allowed roots
- 8.3: No global route guard for authentication (frontend)
- 8.4: RolesPage permission dialog resets existing permissions
- 8.8: Auth store permissions/roles loaded but never checked

### Pattern Analysis

**Backend**: The server has 71 handler functions across 8 handler files. Tenant scope checking is inconsistent:

- Some endpoints (HandleListNodes, HandleListDeployments) filter by `tenant_id` in the SQL query — correct.
- Some endpoints (HandleGetNodeDockerImages, HandleProxyNodeFiles, HandleListNodeBackendRuntimes) query by `node_id` without verifying the node belongs to the caller's tenant — **cross-tenant access**.
- Some endpoints (HandlePatchNodeBackendRuntime, HandleDeleteNodeBackendRuntime) look up NBR by ID without tenant check — **cross-tenant mutation**.

There is no unified `tenantScopeCheck` middleware or helper. Each handler is responsible for its own tenant check, and 12+ handlers simply don't do it.

**Frontend**: The router (`web/src/router/index.ts`) has zero `beforeEach` guards. Auth is checked reactively inside `ConsoleLayout.vue` after components mount. The auth store loads permissions/roles but never uses them — all admin-gated UI uses only `is_platform_admin`.

### Why It Exists

Tenant isolation was added incrementally. Early handlers (nodes, GPUs) got tenant filtering. Later handlers (node-proxy, NBR, model roots) were added without the same pattern. There's no shared middleware or helper that enforces "does this resource belong to the caller's tenant?"

### Systemic Assessment

**YES — systemic**. 12+ endpoints share the same gap: query by resource ID without tenant ownership check. This is not a series of oversights; it's a missing abstraction.

### Refactoring Recommendation

**Middleware abstraction**: Create a `tenantScopeMiddleware` or `requireTenantOwnership(resourceType, lookupFn)` helper that:
1. Extracts resource ID from request
2. Looks up resource's tenant_id
3. Compares with caller's tenant_id (unless platform admin)
4. Returns 403 if mismatch

This eliminates the per-handler pattern of "remember to check tenant."

### Impact

- Security: Cross-tenant data access across all node-proxy, NBR, model root, and artifact endpoints
- Scope: All multi-tenant deployments

---

## Family B: Server → Agent Proxy / SSRF / Outbound HTTP Safety

### Findings
- 5.1: SSRF via agent proxy endpoints (P0)
- 5.11: Observability status endpoint no auth (P2)

### Pattern Analysis

Every server→agent outbound call follows the same pattern:
```go
agentURL := fmt.Sprintf("http://%s:%d/endpoint?%s", ip, port, queryString)
resp, err := http.Get(agentURL)  // no timeout, no SSRF protection
```

Verified in:
- `agent_proxy_handlers.go:51` (files)
- `agent_proxy_handlers.go:109` (model scan)
- `agent_handlers.go:611` (docker images — query NOT URL-encoded)
- `agent_handlers.go:649` (docker image inspect — ref IS URL-encoded, inconsistent)

There is no unified agent client. Each handler constructs its own URL and makes its own HTTP call. No shared timeout, no SSRF deny list, no consistent URL encoding.

### Why It Exists

The proxy pattern was implemented endpoint-by-endpoint. Each handler needed to call the agent, so each one built its own `http.Get` call. No shared abstraction was created.

### Systemic Assessment

**YES — systemic**. All 4+ proxy handlers have the same vulnerability pattern. The fix is a shared agent client.

### Refactoring Recommendation

**Unified agent client abstraction**: Create an `AgentClient` that:
1. Validates agent address against SSRF deny list (169.254.x.x, 10.x, 172.16-31.x, 127.x, metadata IPs)
2. Uses `http.Client` with configurable timeout
3. Properly URL-encodes all query parameters
4. Validates address scheme (http/https only)
5. Optionally adds agent auth token

```go
type AgentClient struct {
    httpClient *http.Client
    denyList   []*net.IPNet
}

func (c *AgentClient) Get(addr string, port int, path string, params url.Values) (*http.Response, error) {
    if err := c.validateAddress(addr); err != nil {
        return nil, err
    }
    u := fmt.Sprintf("http://%s:%d%s?%s", addr, port, path, params.Encode())
    return c.httpClient.Get(u)
}
```

### Impact

- Security: SSRF to cloud metadata, internal network scanning
- Scope: All server→agent communication

---

## Family C: Agent Security Boundary

### Findings
- 5.2: Privileged container passthrough (P0)
- 5.3: Agent token weak default (P0)
- 5.5: Agent metrics endpoint no auth (P1)
- 5.6: AllowRuntimeRootAdd bypass (P1)
- 7.1: Collector sh -c injection (P1)

### Pattern Analysis

The agent has **no security policy layer**. It trusts everything from the server:

1. **Container creation**: `buildCreateOptions()` at `docker.go:415-460` passes through privileged, IPCMode, UTSMode, NetworkMode, GroupAdd, SecurityOptions, Ulimits — all without validation.
2. **Agent token**: Default is `"lightai-agent-token-change-me"`. No startup warning. No runtime enforcement.
3. **Metrics server**: 6 endpoints (healthz, metrics, docker-images, docker-image-inspect, files, model-paths/scan) have zero authentication. Any network client can browse filesystem and inspect Docker images.
4. **Runtime root add**: When enabled, any HTTP client can add arbitrary filesystem roots.
5. **Collector execution**: External commands run via `sh -c` with config-provided strings.

### Why It Exists

The agent was designed as an extension of the server, not as an independent security boundary. The implicit trust model is: "server is trusted, agent obeys." This works in single-node development but breaks in production where the network is untrusted.

### Systemic Assessment

**YES — systemic**. The agent lacks a security policy layer entirely. Every endpoint and every Docker capability is wide open. This is an architectural gap, not a series of missing checks.

### Refactoring Recommendation

**Agent security policy**: Create an `AgentSecurityPolicy` that:
1. Defines allowed Docker capabilities (default: no privileged, no host IPC/UTS, no arbitrary security options)
2. Enforces agent token validation with constant-time comparison
3. Adds optional auth to metrics endpoints (bearer token or bind to 127.0.0.1)
4. Validates collector commands against an allowlist
5. Restricts AllowRuntimeRootAdd to authenticated requests

```go
type AgentSecurityPolicy struct {
    AllowPrivileged    bool     // default: false
    AllowedIPCModes    []string // default: ["none", "private"]
    AllowedNetworkModes []string // default: ["bridge", "none"]
    MetricsAuthToken   string   // optional, for /metrics auth
    CollectorAllowList []string // allowed collector commands
}
```

### Impact

- Security: Full host compromise via privileged containers, filesystem exposure, command injection
- Scope: All agent deployments

---

## Family D: Docker Lifecycle / Runtime State Machine / Cleanup

### Findings
- 6.1: Container not cleaned up after start failure (P0)
- 6.3: No container removal on stop (P1)
- 7.5: Stop timeout hardcoded 30s (P3)

### Pattern Analysis

The Docker lifecycle has no unified state machine:

| Phase | Create | Start | Health Check | Stop | Cleanup |
|-------|--------|-------|--------------|------|---------|
| Success | Container created | Container started | Health OK | Container stopped | — |
| Failure | — | Container leaked | Container leaked | Container not removed | No cleanup |

Three failure paths (start failure, post-start not-running, health check failure) all return error without calling `ContainerRemove`. Stop calls `ContainerStop` but never `ContainerRemove`.

The container name `lightai-{first12chars}` is deterministic. On retry after a leaked container, the name collides and the new `ContainerCreate` fails.

### Why It Exists

The lifecycle was built incrementally: first create+start, then health check, then stop. Each phase handles its own success path but doesn't coordinate cleanup on failure. There's no central lifecycle coordinator.

### Systemic Assessment

**YES — systemic**. The same missing-cleanup pattern appears in 3 failure paths. The lifecycle needs a state machine that explicitly handles transitions and cleanup.

### Refactoring Recommendation

**Lifecycle state machine**: Create a `ContainerLifecycle` that manages state transitions:

```
CREATED → STARTING → HEALTHY → RUNNING → STOPPING → STOPPED → REMOVED
    ↓         ↓          ↓
  FAILED    FAILED    FAILED → CLEANUP → REMOVED
```

Each transition:
1. Records the current state
2. Attempts the operation
3. On failure, transitions to FAILED with cleanup
4. Cleanup always calls ContainerRemove (with best-effort error handling)

```go
func (d *DockerRuntimeDriver) StartWithCleanup(ctx context.Context, opts ContainerCreateOptions, spec AgentRunSpec) (*InstanceInfo, error) {
    containerID, err := d.client.ContainerCreate(ctx, opts)
    if err != nil {
        return nil, err
    }
    
    if err := d.client.ContainerStart(ctx, containerID); err != nil {
        d.client.ContainerRemove(ctx, containerID) // cleanup
        return nil, err
    }
    
    // ... health check with cleanup on failure ...
}
```

### Impact

- Stability: Container name collisions on retry, Docker resource leaks
- Scope: All container lifecycle operations

---

## Family E: Concurrency / Race / Task Dispatch

### Findings
- 6.2: Race on lastStderrBytes map (P0)
- 6.4: Race on reconcileState global struct (P1)
- 7.4: No task deduplication (P2)

### Pattern Analysis

The agent has several concurrency issues:

1. **`logsTaskState.lastStderrBytes`**: Plain `map[string]int` accessed from multiple goroutines (each task runs in its own goroutine via `go processTask()`). No mutex.
2. **`reconcileState`**: Global struct with `unloggedCount` read/written from reconcile goroutine and ticker goroutine. No mutex.
3. **Task dispatch**: Tasks from heartbeat dispatched without checking if same task ID already in flight. No in-flight tracking.

The agent uses a semaphore (`taskSem`) for concurrency limiting, but there's no task registry to prevent duplicate execution.

### Why It Exists

Go's concurrency model makes it easy to share state across goroutines. The agent started with single-goroutine task processing and added concurrency incrementally without revisiting shared state.

### Systemic Assessment

**PARTIAL — partially systemic**. The map race (6.2) and reconcile race (6.4) are localized bugs (add mutex). Task dedup (7.4) is a missing feature (add task registry). They share a root cause (concurrent access to shared state) but the fixes are different.

### Refactoring Recommendation

**Targeted fixes**:
1. Add `sync.Mutex` to `logsTaskState` (or use `sync.Map`)
2. Add `sync.Mutex` to `reconcileState`
3. Add task registry (`map[string]bool`) with mutex to prevent duplicate task dispatch

**Optional — task executor abstraction**: If more concurrency issues emerge, create a `TaskExecutor` that manages in-flight tasks, deduplication, and error handling centrally.

### Impact

- Stability: Agent panics from concurrent map access
- Scope: Agent runtime

---

## Family F: Input / Output / Log Safety

### Findings
- 5.8: No request body size limit (P1)
- 5.9: JSON injection in audit log detail (P1)
- 7.2: Docker stream payload not limited (P2)
- 7.3: Task result stdout/stderr not limited (P2)
- 10.3: redactDetailString incorrect replacement (P2)

### Pattern Analysis

There are no size limits anywhere in the data pipeline:

1. **Request bodies**: No `http.MaxBytesReader` on any handler. Multi-GB JSON body → OOM.
2. **Docker streams**: `decodeDockerStream` allocates based on `payloadLen` header (up to 4GB).
3. **Task results**: Full container logs marshaled into JSON, no truncation.
4. **Audit detail**: `fmt.Sprintf` used for JSON construction (injection risk). `redactDetailString` does substring replacement (data corruption).

### Why It Exists

Size limits and safe marshaling were not part of the initial design. The codebase was built for correctness first, safety second.

### Systemic Assessment

**YES — systemic**. The same "no size limit" pattern appears in request handling, Docker streams, and task results. The same "unsafe string construction" pattern appears in audit logging and redaction.

### Refactoring Recommendation

**Unified helpers**:

1. **Body limit middleware**: `http.MaxBytesReader` wrapper for all JSON decode paths
2. **Stream size limit**: Max allocation check in `decodeDockerStream`
3. **Result truncation**: Truncate stdout/stderr to configurable max (e.g., 1MB) before marshaling
4. **Safe audit detail**: Use `json.Marshal` instead of `fmt.Sprintf` for audit detail construction
5. **Proper redaction**: Redact key-value pairs, not substrings

```go
// Body limit middleware
func BodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
            next.ServeHTTP(w, r)
        })
    }
}

// Safe audit detail
func auditDetail(fields map[string]string) string {
    b, _ := json.Marshal(fields)
    return string(b)
}
```

### Impact

- Stability: OOM from unbounded allocation
- Security: JSON injection in audit logs
- Scope: All data paths

---

## Family G: RunPlan / Runtime Config Consistency

### Findings
- 4.1: Boolean flag dedup mishandled (P1)
- 4.2: Env overrides bypass substituteVars (P1)
- 4.3: Required params silently skipped (P1)
- 4.4: buildDeviceBinding dead code (P2)
- 4.5: computeInputHash missing fields (P2)
- 4.6-4.8: Stale/dead catalog config (P2)
- 10.1-10.2: Hardcoded ports/paths (P2)

### Pattern Analysis

The RunPlan resolver (`resolver.go`, 1030 lines) is the most complex component. It has a 5-layer arg-building pipeline:

1. Backend.default_args_json
2. BackendVersion.default_args
3. BackendRuntime.args_override_json
4. Deployment.parameters_json (via mapParametersToArgs)
4. Deployment.env_overrides_json (via buildEnv)

Issues:
- **Boolean flags**: `deduplicateArgs` assumes all flags have values. Boolean flags like `--trust-remote-code` consume the next flag as their value.
- **Env substitution**: Layer 5 skips `substituteVars()` while layers 1-4 all call it.
- **Required params**: Missing required params silently continue with no error.
- **Dead code**: `buildDeviceBinding()` defined but never called.
- **Hash gaps**: `computeInputHash` omits GPU assignments and node overrides.
- **Catalog drift**: SGLang version reference stale, Ollama uses raw JSON blob, vLLM has dead keys.

### Why It Exists

The resolver grew organically. Layer 1-3 were built first, layer 4 added later, layer 5 added even later. Each layer was correct in isolation but the pipeline wasn't verified end-to-end. Catalog YAML files were created once and not maintained.

### Systemic Assessment

**YES — systemic**. The arg-building pipeline has 5 layers with inconsistent behavior (substitution, dedup, validation). The pipeline needs end-to-end verification and consistent treatment of all layers.

### Refactoring Recommendation

**Resolver pipeline refactor**:
1. Fix `deduplicateArgs` to handle boolean flags (flags without values)
2. Apply `substituteVars()` to layer 5
3. Add required parameter validation (append error when missing)
4. Remove or integrate `buildDeviceBinding()`
5. Fix `computeInputHash` to include all relevant fields
6. Clean up catalog YAML files

**Boolean flag fix**:
```go
func deduplicateArgs(args []string) []string {
    // Track which args are boolean flags (no value)
    booleanFlags := make(map[string]bool)
    // ... scan for flags followed by other flags or end of args ...
    
    // In dedup logic, handle boolean flags differently
}
```

### Impact

- Correctness: Silent incorrect args passed to containers
- Scope: All deployments using boolean flags, env overrides, or required params

---

## Family H: Web UI / API Contract / i18n / Auth Guard

### Findings
- 8.1: Hardcoded Chinese strings (P1, adjusted from P0)
- 8.2: Default credentials in UI (P1, adjusted from P0)
- 8.3: No auth route guard (P1)
- 8.4: RolesPage permission reset (P1)
- 8.5: Hardcoded ports (P2)
- 8.6: No confirmation dialogs (P2)
- 8.7: useNodeLabels stale cache (P2)
- 8.8: Auth store permissions unused (P2)

### Pattern Analysis

Frontend issues fall into two categories:

**Security**:
- No route guard (8.3) — brief window where unauthorized UI is visible
- Default credentials shown to all users (8.2)
- Permissions loaded but never checked (8.8)

**Quality**:
- Hardcoded Chinese in 3 lines of DashboardPage (8.1)
- Hardcoded ports in observability pages (8.5)
- No confirmation dialogs for destructive actions (8.6)
- Stale node label cache (8.7)
- RolesPage permission reset bug (8.4)

### Why It Exists

The frontend was built for functionality first. i18n was added incrementally (600+ keys) but 3 lines were missed. Route guards were deferred. Permissions were loaded but the UI only checks `is_platform_admin`.

### Systemic Assessment

**PARTIAL — partially systemic**. The hardcoded Chinese (8.1) is localized (3 lines). The route guard (8.3) is an architectural gap. The permission system (8.8) is a design gap (loaded but unused). These need different approaches.

### Refactoring Recommendation

**Route guard** (architectural):
```ts
router.beforeEach((to, from, next) => {
  const auth = useAuthStore()
  if (to.path !== '/login' && !auth.isAuthenticated) {
    next('/login')
  } else {
    next()
  }
})
```

**i18n cleanup**: Move 3 hardcoded Chinese strings to i18n keys.

**Permission wiring**: Check permissions array instead of just `is_platform_admin`.

**Credentials**: Show only to platform admins, or remove from UI entirely.

### Impact

- Security: Brief auth bypass window, credential exposure
- Quality: i18n gaps, UX issues
- Scope: All frontend users

---

## Family I: Test / E2E Credibility

### Findings
- 9.1: No auth unit tests (P0)
- 9.2: E2E requires real GPU (P0)
- 9.3: Frontend tests are static analysis only (P1)
- 9.4: TestNoVarSyntax empty assertion (P1)
- 9.5: TestTenantAdminCannotTransfer uses t.Logf (P1)
- 9.6: No concurrency tests (P2)
- 9.7: Missing tenant isolation edge cases (P2)

### Pattern Analysis

**Structural gaps**:
- Auth package has zero unit tests (9.1)
- All E2E requires real hardware (9.2)
- Frontend has zero component tests (9.3)

**Broken tests**:
- `TestNoVarSyntax` has empty if-body — always passes (9.4)
- `TestTenantAdminCannotTransferOtherTenantNode` uses `t.Logf` instead of `t.Errorf` — always passes (9.5)

**Missing coverage**:
- No concurrency/race tests (9.6)
- Tenant isolation tests missing edge cases (9.7)

### Why It Exists

Tests were written for the happy path. Auth was tested indirectly through handler tests. E2E was built for real hardware validation. Frontend tests were static analysis only (ESLint, TypeScript). Some tests were written but never completed (empty assertions).

### Systemic Assessment

**YES — systemic**. The test infrastructure has fundamental gaps:
1. No auth unit tests → auth changes have no safety net
2. No CI-compatible E2E → all regression detection depends on manual testing
3. No frontend component tests → UI changes have no behavioral verification
4. Broken tests → false confidence

### Refactoring Recommendation

**Test infrastructure overhaul**:

1. **Auth unit tests**: Test session creation, CSRF validation, password hashing, role permission checks in isolation
2. **Mock E2E**: Create in-container E2E that uses mock GPU data (no real hardware)
3. **Frontend component tests**: Add Vue Test Utils tests for key components
4. **Fix broken tests**: Add actual assertions to TestNoVarSyntax and TestTenantAdminCannotTransfer
5. **Race tests**: Add `go test -race` to CI
6. **Tenant isolation tests**: Add edge cases (cross-tenant deployment/artifact access, admin bypass, empty tenant_id)

### Impact

- Confidence: No safety net for auth, frontend, or concurrency changes
- Scope: Entire project

---

## Cross-Family Dependencies

Several families interact:

1. **A + B**: Tenant scope check (A) should happen before SSRF proxy call (B). If tenant check is missing, SSRF is exploitable cross-tenant.
2. **C + D**: Agent security policy (C) should reject privileged containers. Docker lifecycle (D) should clean up on failure. Both are needed.
3. **F + G**: Body size limit (F) prevents OOM. Resolver correctness (G) prevents bad args. Both affect deployment reliability.
4. **I + A**: Auth unit tests (I) would have caught tenant scope gaps (A). Test infrastructure enables security verification.

---

## Refactoring Priority Matrix

| Family | Systemic? | Refactoring Type | Effort | Impact |
|--------|-----------|-----------------|--------|--------|
| A. Tenant Scope | YES | Middleware | Medium | High (security) |
| B. SSRF | YES | Agent client | Medium | High (security) |
| C. Agent Security | YES | Policy layer | Large | High (security) |
| D. Docker Lifecycle | YES | State machine | Large | High (stability) |
| E. Concurrency | PARTIAL | Targeted fixes | Small | Medium (stability) |
| F. I/O Safety | YES | Middleware + helpers | Medium | Medium (stability) |
| G. RunPlan Config | YES | Pipeline refactor | Large | Medium (correctness) |
| H. Web UI | PARTIAL | Targeted fixes | Small | Low (quality) |
| I. Test Infrastructure | YES | Framework | Large | High (confidence) |

**Recommended order**: A → B → C → D → F → G → I → E → H

Rationale: Security first (A, B, C), then stability (D, F), then correctness (G), then confidence (I), then targeted fixes (E, H).
