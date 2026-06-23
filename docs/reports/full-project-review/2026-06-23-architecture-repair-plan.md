# Architecture Repair Plan — Full Project Review

> Date: 2026-06-23
> Source: Evidence Triage + Issue Family Analysis
> Purpose: 6-batch repair plan with refactoring design, testing strategy, and rollback plan
> Status: PLAN ONLY — no code changes in this phase

---

## Batch Overview

| Batch | Name | Findings | Families | Refactoring Type | Effort |
|-------|------|----------|----------|-----------------|--------|
| 1 | Security Boundary & Tenant Isolation | 5.1-5.6, 5.8, 5.9, 5.13 | A, B, C | middleware + client + policy | Large |
| 2 | Runtime Stability & Docker Lifecycle | 6.1-6.4, 7.2-7.5 | D, E | state machine + targeted | Large |
| 3 | Input/Output Hardening & Audit Safety | 5.8, 5.9, 7.2, 7.3, 10.3 | F | middleware + helpers | Medium |
| 4 | RunPlan / Runtime Config / Catalog | 4.1-4.8, 10.1-10.2 | G | pipeline refactor | Large |
| 5 | Web Contract / i18n / Permission UX | 8.1-8.8 | H | targeted fixes | Small |
| 6 | Test Infrastructure & Evidence Quality | 9.1-9.7 | I | framework | Large |

**Execution order**: Batch 1 → 2 → 3 → 4 → 5 → 6

Rationale: Security first, stability second, correctness third, quality last.

---

## Batch 1: Security Boundary & Tenant Isolation

### Goal
Establish unified security boundaries for tenant scope, SSRF protection, agent security, and input validation.

### Findings Covered
- 5.1 (SSRF), 5.2 (privileged), 5.3 (agent token), 5.4 (tenant scope), 5.5 (agent auth), 5.6 (runtime root), 5.8 (body limit), 5.9 (JSON injection), 5.13 (constant-time compare)

### Families: A, B, C

### Refactoring Types
- **Middleware abstraction** (tenant scope, body limit)
- **Client abstraction** (agent client)
- **Policy layer** (agent security)

---

### 1.1 Tenant Scope Middleware

**Files to create:**
- `internal/server/api/tenant_scope.go`

**Files to modify:**
- `internal/server/api/agent_proxy_handlers.go`
- `internal/server/api/agent_handlers.go`
- `internal/server/api/runtime_handlers.go`
- `internal/server/api/node_runtime_handlers.go`
- `internal/server/api/artifact_handlers.go`
- `internal/server/api/model_browser_handlers.go`
- `internal/server/api/router.go`

**Design:**
```go
// tenantOwnershipCheck verifies that the resource identified by resourceID
// belongs to the caller's tenant. Returns (tenantID, error).
// Platform admins bypass the check.
type OwnershipLookup func(db *db.DB, resourceType, resourceID string) (tenantID string, err error)

var ownershipLookups = map[string]OwnershipLookup{
    "node": func(db *db.DB, _, id string) (string, error) {
        var tid string
        err := db.QueryRow("SELECT tenant_id FROM nodes WHERE id = ?", id).Scan(&tid)
        return tid, err
    },
    "node_backend_runtime": func(db *db.DB, _, id string) (string, error) {
        var tid string
        err := db.QueryRow("SELECT tenant_id FROM node_backend_runtimes WHERE id = ?", id).Scan(&tid)
        return tid, err
    },
    "model_root": func(db *db.DB, _, id string) (string, error) {
        var tid string
        err := db.QueryRow("SELECT tenant_id FROM node_model_roots WHERE id = ?", id).Scan(&tid)
        return tid, err
    },
    "model_location": func(db *db.DB, _, id string) (string, error) {
        var tid string
        err := db.QueryRow("SELECT tenant_id FROM model_locations WHERE id = ?", id).Scan(&tid)
        return tid, err
    },
}

func requireTenantOwnership(resourceType string, getResourceID func(*http.Request) string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            info := auth.SessionInfoFromContext(r.Context())
            if info == nil {
                writeError(w, http.StatusUnauthorized, "unauthorized")
                return
            }
            if info.IsPlatformAdmin {
                next.ServeHTTP(w, r)
                return
            }
            resourceID := getResourceID(r)
            lookup, ok := ownershipLookups[resourceType]
            if !ok {
                next.ServeHTTP(w, r)
                return
            }
            ownerTenant, err := lookup(h.DB, resourceType, resourceID)
            if err != nil {
                writeError(w, http.StatusNotFound, "resource not found")
                return
            }
            if ownerTenant != info.TenantID {
                writeError(w, http.StatusForbidden, "access denied: resource belongs to another tenant")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

**Usage in router:**
```go
// Before (no tenant check):
mux.HandleFunc("GET /api/v1/nodes/{id}/files", h.HandleProxyNodeFiles)

// After (with tenant check):
mux.Handle("GET /api/v1/nodes/{id}/files",
    requireTenantOwnership("node", func(r *http.Request) string { return r.PathValue("id") })(http.HandlerFunc(h.HandleProxyNodeFiles)))
```

**Endpoints to add tenant scope (12+):**
1. `GET /api/v1/nodes/{id}/files` → node ownership
2. `POST /api/v1/nodes/{id}/model-scan` → node ownership
3. `GET /api/v1/nodes/{id}/docker-images` → node ownership
4. `GET /api/v1/nodes/{id}/docker-image-inspect` → node ownership
5. `GET /api/v1/nodes/{id}/backend-runtimes` → node ownership
6. `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/enable` → node ownership
7. `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check` → node ownership
8. `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check-request` → node ownership
9. `PATCH /api/v1/node-backend-runtimes/{nbr_id}` → NBR ownership
10. `DELETE /api/v1/node-backend-runtimes/{nbr_id}` → NBR ownership
11. `POST /api/v1/model-locations/{location_id}/rescan` → location ownership
12. `PATCH /api/v1/nodes/{id}/model-roots/{root_id}` → root ownership
13. `DELETE /api/v1/nodes/{id}/model-roots/{root_id}` → root ownership

**Testing:**
- Unit test: `requireTenantOwnership` with mock DB
- Integration test: 13 endpoints × (same-tenant=200, cross-tenant=403, admin-bypass=200)

---

### 1.2 Unified Agent Client

**Files to create:**
- `internal/server/api/agent_client.go`

**Files to modify:**
- `internal/server/api/agent_proxy_handlers.go`
- `internal/server/api/agent_handlers.go`

**Design:**
```go
type AgentClient struct {
    httpClient *http.Client
    denyList   []*net.IPNet
}

func NewAgentClient(timeout time.Duration) *AgentClient {
    denyList := parseDenyList([]string{
        "169.254.0.0/16",  // link-local / cloud metadata
        "10.0.0.0/8",      // private
        "172.16.0.0/12",   // private
        "127.0.0.0/8",     // loopback
        "0.0.0.0/32",      // unspecified
    })
    return &AgentClient{
        httpClient: &http.Client{Timeout: timeout},
        denyList:   denyList,
    }
}

func (c *AgentClient) Get(addr string, port int, path string, params url.Values) (*http.Response, error) {
    if err := c.validateAddress(addr); err != nil {
        return nil, fmt.Errorf("agent address rejected: %w", err)
    }
    u := fmt.Sprintf("http://%s:%d%s?%s", addr, port, path, params.Encode())
    return c.httpClient.Get(u)
}

func (c *AgentClient) Post(addr string, port int, path string, params url.Values, body io.Reader) (*http.Response, error) {
    if err := c.validateAddress(addr); err != nil {
        return nil, fmt.Errorf("agent address rejected: %w", err)
    }
    u := fmt.Sprintf("http://%s:%d%s?%s", addr, port, path, params.Encode())
    return c.httpClient.Post(u, "application/json", body)
}

func (c *AgentClient) validateAddress(addr string) error {
    ip := net.ParseIP(addr)
    if ip == nil {
        return fmt.Errorf("invalid IP: %s", addr)
    }
    for _, deny := range c.denyList {
        if deny.Contains(ip) {
            return fmt.Errorf("denied range: %s in %s", ip, deny)
        }
    }
    return nil
}
```

**Migration**: Replace all `http.Get(agentURL)` calls with `agentClient.Get(addr, port, path, params)`.

**Testing:**
- Unit test: validateAddress with denied IPs (169.254.169.254, 10.0.0.1, 127.0.0.1)
- Unit test: Get/Post with valid addresses
- Integration test: SSRF attempt via agent registration with metadata IP

---

### 1.3 Agent Security Policy

**Files to create:**
- `internal/agent/security/policy.go`

**Files to modify:**
- `internal/agent/runtime/docker.go` (buildCreateOptions)
- `cmd/agent/main.go` (metrics server, token validation)
- `internal/common/config/config.go`

**Design:**
```go
type AgentSecurityPolicy struct {
    AllowPrivileged     bool
    AllowedIPCModes     []string
    AllowedNetworkModes []string
    AllowedSecurityOpts []string
    MetricsAuthToken    string
    CollectorAllowList  []string
}

func DefaultPolicy() AgentSecurityPolicy {
    return AgentSecurityPolicy{
        AllowPrivileged:     false,
        AllowedIPCModes:     []string{"", "none", "private"},
        AllowedNetworkModes: []string{"", "bridge", "none", "host"},
        AllowedSecurityOpts: []string{},
        MetricsAuthToken:    "",
        CollectorAllowList:  []string{},
    }
}

func (p *AgentSecurityPolicy) ValidateDockerSpec(spec DockerSpec) error {
    if spec.Privileged && !p.AllowPrivileged {
        return fmt.Errorf("privileged containers not allowed by agent policy")
    }
    if spec.IPCMode != "" && !contains(p.AllowedIPCModes, spec.IPCMode) {
        return fmt.Errorf("IPC mode %q not allowed by agent policy", spec.IPCMode)
    }
    // ... similar for NetworkMode, SecurityOptions ...
    return nil
}
```

**Integration in buildCreateOptions:**
```go
func (d *DockerRuntimeDriver) buildCreateOptions(spec AgentRunSpec) ContainerCreateOptions {
    if err := d.securityPolicy.ValidateDockerSpec(spec.Docker); err != nil {
        log.Error("docker.security_policy.rejected", "error", err)
        // Return error to caller
    }
    // ... existing logic ...
}
```

**Agent token enforcement:**
```go
// In main.go startup:
if cfg.AgentToken == "lightai-agent-token-change-me" && !cfg.DevMode {
    log.Warn("SECURITY: Using default agent token in non-dev mode. Change agent_token in config!")
}
```

**Metrics auth:**
```go
if cfg.Metrics.AuthToken != "" {
    healthMux.Use(metricsAuthMiddleware(cfg.Metrics.AuthToken))
}
```

**Testing:**
- Unit test: ValidateDockerSpec with allowed/rejected capabilities
- Unit test: Token validation with constant-time comparison
- Integration test: Agent rejects privileged container from compromised server

---

### 1.4 Body Size Limit Middleware

**Files to create:**
- `internal/server/api/middleware_body_limit.go`

**Files to modify:**
- `internal/server/api/router.go`

**Design:**
```go
func BodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
            next.ServeHTTP(w, r)
        })
    }
}
```

**Default limit**: 10MB for API endpoints, 100MB for file upload endpoints.

**Testing:**
- Unit test: Request with body > limit returns 413
- Unit test: Request with body < limit passes through

---

### 1.5 Safe Audit Detail Construction

**Files to modify:**
- `internal/server/api/agent_handlers.go:882-883`
- `internal/server/api/helpers.go`

**Design:**
```go
func auditDetailFromMap(fields map[string]string) string {
    b, _ := json.Marshal(fields)
    return string(b)
}

// Usage:
detail := auditDetailFromMap(map[string]string{
    "from_tenant_id": currentTenant,
    "to_tenant_id":   req.TenantID,
    "reason":         req.Reason,
})
```

**Testing:**
- Unit test: auditDetailFromMap with special characters (", \, newline)
- Unit test: Verify JSON is valid after marshal

---

### 1.6 Agent Token Constant-Time Comparison

**Files to modify:**
- `internal/server/auth/middleware.go:180`

**Change:**
```go
// Before:
if token != agentToken {

// After:
if subtle.ConstantTimeCompare([]byte(token), []byte(agentToken)) != 1 {
```

**Testing:**
- Unit test: Valid token accepted, invalid token rejected
- (Timing test is impractical in unit tests — the fix is a one-line change)

---

### Batch 1 Verification

**Commands:**
```bash
go test ./internal/server/api/... -v -count=1
go test ./internal/agent/... -v -count=1
go test ./internal/server/auth/... -v -count=1
go test -race ./internal/server/api/... -count=1
```

**Acceptance criteria:**
- All existing tests pass
- New tenant scope tests pass (13 endpoints × 3 scenarios)
- SSRF tests pass (denied IPs)
- Agent security policy tests pass
- Body limit tests pass

**Risks:**
- Tenant scope middleware may break endpoints that legitimately need cross-tenant access (platform admin operations). Mitigation: admin bypass.
- SSRF deny list may be too restrictive for some network topologies. Mitigation: configurable deny list.

**Rollback:**
- Remove tenant scope middleware from router
- Revert agent client to direct http.Get calls
- Remove agent security policy checks

---

## Batch 2: Runtime Stability & Docker Lifecycle

### Goal
Fix container lifecycle leaks, race conditions, and establish proper cleanup-on-failure semantics.

### Findings Covered
- 6.1 (container leak), 6.3 (no removal on stop), 6.2 (map race), 6.4 (reconcile race), 7.2 (stream size), 7.3 (result size), 7.4 (task dedup), 7.5 (stop timeout)

### Families: D, E

### Refactoring Types
- **State machine** (Docker lifecycle)
- **Targeted fixes** (races, size limits, dedup)

---

### 2.1 Container Lifecycle Cleanup

**Files to modify:**
- `internal/agent/runtime/docker.go`

**Design**: Add cleanup-on-failure to all three failure paths:

```go
// In Start(): after ContainerCreate succeeds
if err := d.client.ContainerStart(ctx, containerID); err != nil {
    log.Error("docker.start.failed", ...)
    // NEW: cleanup created container
    if rmErr := d.client.ContainerRemove(ctx, containerID); rmErr != nil {
        log.Warn("docker.start.cleanup_failed", "container_id", containerID, "error", rmErr)
    }
    return nil, fmt.Errorf("docker start: %w", err)
}

// In post-start inspect: if container not running
if info.State != "running" {
    log.Error("docker.post_start.container_not_running", ...)
    // NEW: cleanup
    if rmErr := d.client.ContainerRemove(ctx, containerID); rmErr != nil {
        log.Warn("docker.post_start.cleanup_failed", "container_id", containerID, "error", rmErr)
    }
    return nil, fmt.Errorf("container not running")
}

// In health check failure:
if err := CheckEndpointReady(...); err != nil {
    log.Error("health_check.failed", ...)
    // NEW: cleanup
    if rmErr := d.client.ContainerRemove(ctx, containerID); rmErr != nil {
        log.Warn("health_check.cleanup_failed", "container_id", containerID, "error", rmErr)
    }
    return nil, fmt.Errorf("health check failed: %w", err)
}
```

**Stop with removal:**
```go
func (d *DockerRuntimeDriver) Stop(ctx context.Context, instanceID string) error {
    // ... existing stop logic ...
    if err := d.client.ContainerStop(ctx, info.ID, 30); err != nil {
        return fmt.Errorf("docker stop: %w", err)
    }
    // NEW: remove container after stop
    if err := d.client.ContainerRemove(ctx, info.ID); err != nil {
        log.Warn("docker.stop.remove_failed", "container_id", info.ID, "error", err)
    }
    return nil
}
```

**Testing:**
- Unit test: Mock DockerClient, verify ContainerRemove called on start failure
- Unit test: Mock DockerClient, verify ContainerRemove called after stop
- Integration test: Create container, fail start, verify container is gone

---

### 2.2 Race Condition Fixes

**Files to modify:**
- `cmd/agent/main.go`

**Fix 1: logsTaskState**
```go
// Before:
var logsTaskState struct {
    lastStderrBytes map[string]int
}

// After:
var logsTaskState struct {
    mu              sync.Mutex
    lastStderrBytes map[string]int
}

// In processLogsTask:
logsTaskState.mu.Lock()
lastStderr := logsTaskState.lastStderrBytes[payload.InstanceID]
stderrChanged := stderrBytes != lastStderr && stderrBytes > 0
logsTaskState.lastStderrBytes[payload.InstanceID] = stderrBytes
logsTaskState.mu.Unlock()
```

**Fix 2: reconcileState**
```go
// Add mutex to reconcileState struct
var reconcileState struct {
    mu            sync.Mutex
    unloggedCount int
    // ... other fields ...
}

// In reconcileManagedContainers:
reconcileState.mu.Lock()
reconcileState.unloggedCount++
reconcileState.mu.Unlock()
```

**Testing:**
- `go test -race ./cmd/agent/... -count=1`
- Unit test: Concurrent log tasks don't panic

---

### 2.3 Task Deduplication

**Files to modify:**
- `cmd/agent/main.go`

**Design:**
```go
var inFlightTasks struct {
    mu    sync.Mutex
    tasks map[string]bool
}

func (a *Agent) tryClaimTask(taskID string) bool {
    inFlightTasks.mu.Lock()
    defer inFlightTasks.mu.Unlock()
    if inFlightTasks.tasks[taskID] {
        return false // already in flight
    }
    inFlightTasks.tasks[taskID] = true
    return true
}

func (a *Agent) releaseTask(taskID string) {
    inFlightTasks.mu.Lock()
    defer inFlightTasks.mu.Unlock()
    delete(inFlightTasks.tasks, taskID)
}

// In heartbeat task dispatch:
for _, task := range hbResp.Tasks {
    if !tryClaimTask(task.ID) {
        log.Debug("task already in flight, skipping", "task_id", task.ID)
        continue
    }
    // ... dispatch task ...
}
```

**Testing:**
- Unit test: Same task ID dispatched twice, second is skipped
- Unit test: Task released after completion, can be dispatched again

---

### 2.4 Size Limits

**Files to modify:**
- `internal/agent/runtime/docker_real.go` (decodeDockerStream)
- `cmd/agent/main.go` (task result)

**Docker stream limit:**
```go
const maxStreamPayload = 10 * 1024 * 1024 // 10MB

func decodeDockerStream(data []byte) (stdout, stderr string, err error) {
    // ... existing header parsing ...
    if payloadLen > maxStreamPayload {
        return "", "", fmt.Errorf("stream payload too large: %d bytes (max %d)", payloadLen, maxStreamPayload)
    }
    // ... existing allocation ...
}
```

**Task result truncation:**
```go
const maxResultBytes = 1024 * 1024 // 1MB

func truncateResult(result *TaskResult) {
    if len(result.Stdout) > maxResultBytes {
        result.Stdout = result.Stdout[:maxResultBytes] + "\n... [truncated]"
    }
    if len(result.Stderr) > maxResultBytes {
        result.Stderr = result.Stderr[:maxResultBytes] + "\n... [truncated]"
    }
}
```

**Testing:**
- Unit test: Stream with payload > 10MB returns error
- Unit test: Task result > 1MB is truncated

---

### Batch 2 Verification

**Commands:**
```bash
go test -race ./cmd/agent/... -v -count=1
go test ./internal/agent/runtime/... -v -count=1
```

**Acceptance criteria:**
- Container cleanup tests pass (3 failure paths)
- Race detector finds no races
- Task dedup tests pass
- Size limit tests pass

**Risks:**
- Container removal on stop may break workflows that expect stopped containers to persist. Mitigation: make removal configurable.
- Task dedup may cause legitimate retries to be skipped. Mitigation: release task on timeout.

**Rollback:**
- Remove ContainerRemove calls from failure paths
- Remove mutex from logsTaskState and reconcileState
- Remove inFlightTasks map

---

## Batch 3: Input/Output Hardening & Audit Safety

### Goal
Establish size limits, safe serialization, and proper redaction across all data paths.

### Findings Covered
- 5.8 (body limit — covered in Batch 1.4), 5.9 (JSON injection — covered in Batch 1.5), 7.2 (stream size — covered in Batch 2.4), 7.3 (result size — covered in Batch 2.4), 10.3 (redactDetailString)

### Families: F

### Note
Most findings in this family are already covered by Batch 1 and Batch 2. This batch focuses on the remaining item: redaction cleanup.

---

### 3.1 Fix redactDetailString

**Files to modify:**
- `internal/server/api/helpers.go:225-234`

**Design:**
```go
func redactDetailString(s string) string {
    // Parse as JSON if possible, redact key-value pairs
    var m map[string]interface{}
    if json.Unmarshal([]byte(s), &m) == nil {
        for k := range m {
            if isSensitive(k) {
                m[k] = "<redacted>"
            }
        }
        b, _ := json.Marshal(m)
        return string(b)
    }
    // Fallback: redact known key=value patterns
    result := s
    for _, sk := range sensitiveKeys() {
        // Match "KEY":"value" or "KEY": "value" or KEY=value
        re := regexp.MustCompile(fmt.Sprintf(`(?i)(%s["']?\s*[:=]\s*["']?)[^"',}\s]+`, regexp.QuoteMeta(sk)))
        result = re.ReplaceAllString(result, "${1}<redacted>")
    }
    return result
}
```

**Testing:**
- Unit test: `{"password":"secret"}` → `{"password":"<redacted>"}`
- Unit test: `PASSWORD_CHANGED` is NOT corrupted
- Unit test: `ACCESS_LOG` is NOT corrupted

---

### Batch 3 Verification

**Commands:**
```bash
go test ./internal/server/api/... -v -count=1 -run TestRedact
```

**Acceptance criteria:**
- Redaction tests pass
- No data corruption in audit logs

**Risks:**
- Regex-based redaction may miss edge cases. Mitigation: JSON-first approach with regex fallback.

**Rollback:**
- Revert to original redactDetailString

---

## Batch 4: RunPlan / Runtime Config / Catalog Correctness

### Goal
Fix the resolver pipeline's boolean flag handling, env substitution, required param validation, and catalog drift.

### Findings Covered
- 4.1 (boolean flags), 4.2 (env substitution), 4.3 (required params), 4.4 (dead code), 4.5 (hash gaps), 4.6-4.8 (catalog), 10.1-10.2 (hardcoded values)

### Families: G

### Refactoring Type
- **Pipeline refactor** (resolver)

---

### 4.1 Fix deduplicateArgs for Boolean Flags

**Files to modify:**
- `internal/server/runplan/resolver.go:470-504`

**Design:**
```go
func deduplicateArgs(args []string) []string {
    // First pass: identify boolean flags (flags followed by another flag or end of args)
    booleanFlags := make(map[string]bool)
    i := 0
    for i < len(args) {
        if strings.HasPrefix(args[i], "-") {
            if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
                booleanFlags[args[i]] = true
            }
        }
        i++
    }
    
    // Second pass: identify last occurrence of each flag
    type flagPair struct{ idx int; isBool bool }
    lastSeen := make(map[string]flagPair)
    i = 0
    for i < len(args) {
        arg := args[i]
        if strings.HasPrefix(arg, "-") {
            if booleanFlags[arg] {
                lastSeen[arg] = flagPair{idx: i, isBool: true}
                i++
            } else if i+1 < len(args) {
                lastSeen[arg] = flagPair{idx: i, isBool: false}
                i += 2
            } else {
                i++
            }
        } else {
            i++
        }
    }
    
    // Third pass: keep only last occurrence
    seen := make(map[string]bool)
    var result []string
    i = 0
    for i < len(args) {
        arg := args[i]
        if fp, ok := lastSeen[arg]; ok && fp.idx == i {
            if !seen[arg] {
                seen[arg] = true
                result = append(result, arg)
                if !fp.isBool {
                    result = append(result, args[i+1])
                }
            }
            if fp.isBool {
                i++
            } else {
                i += 2
            }
        } else if strings.HasPrefix(arg, "-") && !booleanFlags[arg] && i+1 < len(args) {
            i += 2 // skip non-last flag-value pair
        } else {
            result = append(result, arg)
            i++
        }
    }
    return result
}
```

**Testing:**
- Unit test: `["--trust-remote-code", "--model", "llama"]` → `["--trust-remote-code", "--model", "llama"]`
- Unit test: `["--model", "a", "--model", "b"]` → `["--model", "b"]` (last wins)
- Unit test: `["--verbose", "--port", "8080"]` → `["--verbose", "--port", "8080"]`

---

### 4.2 Apply substituteVars to Layer 5

**Files to modify:**
- `internal/server/runplan/resolver.go:597-600`

**Change:**
```go
// Before:
for k, v := range in.Deployment.EnvOverrides {
    env[k] = v
}

// After:
for k, v := range in.Deployment.EnvOverrides {
    resolved, err := substituteVars(v, vars)
    if err != nil {
        warnings = append(warnings, fmt.Sprintf("env %s: %v", k, err))
        continue
    }
    env[k] = resolved
}
```

**Testing:**
- Unit test: Env override with `{{MODEL_CONTAINER_PATH}}` is substituted
- Unit test: Env override without variables passes through unchanged

---

### 4.3 Add Required Parameter Validation

**Files to modify:**
- `internal/server/runplan/resolver.go:531-536`

**Change:**
```go
// Before:
} else if def.Required {
    // Required parameter missing — skip for now, resolver will report
    continue
}

// After:
} else if def.Required {
    errors = append(errors, fmt.Errorf("required parameter %q is missing and has no default", def.Name))
    continue
}
```

**Testing:**
- Unit test: Missing required parameter returns error
- Unit test: Missing optional parameter does not return error

---

### 4.4 Remove Dead buildDeviceBinding Code

**Files to modify:**
- `internal/server/runplan/resolver.go:986-1030`

**Action**: Remove `buildDeviceBinding()` function and `DeviceBinding` struct if confirmed unused.

**Testing:**
- Verify no references to `buildDeviceBinding` or `DeviceBinding` in codebase
- `go build ./...` passes

---

### 4.5 Fix computeInputHash

**Files to modify:**
- `internal/server/runplan/resolver.go:933-949`

**Change:**
```go
// Add missing fields:
data, _ := json.Marshal(map[string]interface{}{
    "backend":         in.Backend.Name,
    "version":         in.BackendVersion.Version,
    "runtime":         in.BackendRuntime.ID,
    "artifact":        in.Artifact.Path,
    "deployment":      in.Deployment.ID,
    "host_port":       in.Deployment.Service.HostPort,
    "container_port":  in.Deployment.Service.ContainerPort,
    "app_port":        in.Deployment.Service.AppPort,
    "parameters":      in.Deployment.Parameters,
    "env_overrides":   in.Deployment.EnvOverrides,
    "accelerator_ids": in.Deployment.Placement.AcceleratorIds,
    "node_id":         in.Deployment.Placement.NodeID,
    "assigned_gpus":   in.Deployment.Placement.AssignedGPUs,       // NEW
    "node_override":   in.NodeRuntimeOverride,                      // NEW
})
```

**Testing:**
- Unit test: Same inputs produce same hash
- Unit test: Different GPU assignments produce different hashes

---

### 4.6 Catalog Cleanup

**Files to modify:**
- `configs/backend-catalog/runtimes/sglang/nvidia-cuda.yaml:6`
- `configs/backend-catalog/versions/ollama/ollama-latest.yaml:15`
- `configs/backend-catalog/runtimes/vllm/nvidia-cuda.yaml:24-27`

**Changes:**
1. Update SGLang version reference from `v0.5.12.post1` to `v0.5.13.post1`
2. Convert Ollama capabilities_json from raw JSON string to structured YAML
3. Remove dead `gpus: all` and `runtime: nvidia` keys from vLLM runtime YAML

**Testing:**
- `go test ./internal/server/runplan/... -v -count=1` — catalog parsing tests pass
- Verify no dead keys in catalog YAML

---

### Batch 4 Verification

**Commands:**
```bash
go test ./internal/server/runplan/... -v -count=1
go build ./...
```

**Acceptance criteria:**
- Boolean flag dedup tests pass
- Env substitution tests pass
- Required param validation tests pass
- Hash tests pass
- Catalog tests pass
- All existing tests still pass

**Risks:**
- Boolean flag fix may change behavior for existing deployments. Mitigation: extensive test coverage.
- Required param validation may break deployments that relied on silent skip. Mitigation: this is a correctness fix — the old behavior was wrong.

**Rollback:**
- Revert deduplicateArgs to original
- Revert buildEnv layer 5
- Revert mapParametersToArgs required param handling

---

## Batch 5: Web Contract / i18n / Permission UX

### Goal
Fix frontend security gaps, i18n issues, and UX bugs.

### Findings Covered
- 8.1 (hardcoded Chinese), 8.2 (default credentials), 8.3 (route guard), 8.4 (RolesPage), 8.5 (hardcoded ports), 8.6 (confirmation dialogs), 8.7 (stale cache), 8.8 (unused permissions)

### Families: H

### Refactoring Type
- **Targeted fixes**

---

### 5.1 Add Route Guard

**Files to modify:**
- `web/src/router/index.ts`

**Design:**
```ts
import { useAuthStore } from '@/stores/auth'

router.beforeEach((to, from, next) => {
  const auth = useAuthStore()
  if (to.path !== '/login' && to.path !== '/change-password' && !auth.isAuthenticated) {
    next('/login')
  } else {
    next()
  }
})
```

**Testing:**
- Manual: Navigate to /nodes without login → redirected to /login
- Manual: Navigate to /nodes with login → page loads

---

### 5.2 Fix Hardcoded Chinese Strings

**Files to modify:**
- `web/src/pages/DashboardPage.vue:126,130,134`
- `web/src/locales/en-US.ts`
- `web/src/locales/zh-CN.ts`

**Changes:**
```vue
<!-- Before -->
{{ heartbeatOk ? '正常' : nodesWithStaleHeartbeat + ' 个节点超时' }}

<!-- After -->
{{ heartbeatOk ? t('dashboard.normal') : t('dashboard.nodesTimeout', { count: nodesWithStaleHeartbeat }) }}
```

Add i18n keys:
```ts
// en-US.ts
dashboard: {
  normal: 'Normal',
  nodesTimeout: '{count} nodes timed out',
  gpuAbnormal: '{count} GPUs abnormal',
  gpuDataStale: '{count} GPUs data stale',
}

// zh-CN.ts
dashboard: {
  normal: '正常',
  nodesTimeout: '{count} 个节点超时',
  gpuAbnormal: '{count} 个 GPU 异常',
  gpuDataStale: '{count} 个 GPU 数据过期',
}
```

---

### 5.3 Remove Default Credentials from UI

**Files to modify:**
- `web/src/locales/en-US.ts:322`
- `web/src/locales/zh-CN.ts:322`

**Change**: Replace `credentialsHint` with a generic message:
```ts
// Before:
credentialsHint: 'Default username/password: admin/admin',

// After:
credentialsHint: 'Check Grafana admin settings for credentials',
```

Or show only to platform admins:
```vue
<el-descriptions-item v-if="isPlatformAdmin" :label="t('observability.defaultLogin')">
  {{ t('observability.credentialsHint') }}
</el-descriptions-item>
```

---

### 5.4 Fix RolesPage Permission Reset

**Files to modify:**
- `web/src/pages/RolesPage.vue:81-89`

**Change:**
```ts
async function openPermissions(row: Role) {
  editingRole.value = row
  permVisible.value = true
  loadingPerms.value = true
  try {
    allPermissions.value = await fetchPermissions()
    // NEW: load existing permissions for this role
    const rolePerms = await fetchRolePermissions(row.id)
    selectedPermIds.value = rolePerms.map(p => p.id)
  } catch (e: any) { ... }
  finally { loadingPerms.value = false }
}
```

---

### 5.5 Configurable Observability Ports

**Files to modify:**
- `web/src/pages/GrafanaPage.vue:27`
- `web/src/pages/PrometheusPage.vue:23`
- `web/src/pages/ObservabilityOverviewPage.vue:37-38`

**Change**: Fetch ports from `/api/v1/observability/status` instead of hardcoding.

---

### 5.6 Add Confirmation Dialogs

**Files to modify:**
- `web/src/pages/ModelDeploymentsPage.vue`
- `web/src/pages/ModelInstancesPage.vue`

**Change**: Add `ElMessageBox.confirm()` before stop/restart actions.

---

### Batch 5 Verification

**Commands:**
```bash
cd web && npm run build
npm run lint
```

**Acceptance criteria:**
- Build passes
- Lint passes
- Route guard redirects unauthenticated users
- i18n keys work in both locales
- RolesPage loads existing permissions
- Confirmation dialogs appear before destructive actions

**Risks:**
- Route guard may cause flash of login page. Mitigation: check auth state before route change.
- i18n key changes may break existing translations. Mitigation: update both locales simultaneously.

**Rollback:**
- Remove router.beforeEach
- Revert i18n keys
- Revert RolesPage changes

---

## Batch 6: Test Infrastructure & Evidence Quality

### Goal
Establish unit tests for auth, fix broken tests, add CI-compatible E2E, and add frontend component tests.

### Findings Covered
- 9.1 (no auth tests), 9.2 (E2E requires GPU), 9.3 (no frontend tests), 9.4 (empty assertion), 9.5 (t.Logf), 9.6 (no race tests), 9.7 (missing tenant edge cases)

### Families: I

### Refactoring Type
- **Test infrastructure**

---

### 6.1 Auth Unit Tests

**Files to create:**
- `internal/server/auth/session_test.go`
- `internal/server/auth/csrf_test.go`
- `internal/server/auth/password_test.go`
- `internal/server/auth/rbac_test.go`

**Tests to write:**
- Session creation and retrieval
- CSRF token generation and validation
- Password hashing (Argon2id) and verification
- Role permission checks
- Session expiry
- Concurrent session access

---

### 6.2 Fix Broken Tests

**Files to modify:**
- `internal/server/runplan/resolver_test.go:278-288` (TestNoVarSyntax)
- `internal/server/api/agent_identity_test.go:233-258` (TestTenantAdminCannotTransfer)

**Fix TestNoVarSyntax:**
```go
func TestNoVarSyntax(t *testing.T) {
    in := makeTestInput()
    in.BackendVersion.DefaultArgs = []string{"${MAX_MODEL_LEN}"}
    plan, _, _ := Resolve(in)
    joined := strings.Join(plan.Args, " ")
    if !strings.Contains(joined, "${MAX_MODEL_LEN}") {
        t.Errorf("${MAX_MODEL_LEN} should be preserved as literal, got: %s", joined)
    }
}
```

**Fix TestTenantAdminCannotTransfer:**
```go
func TestTenantAdminCannotTransferOtherTenantNode(t *testing.T) {
    // ... existing setup ...
    handler.HandlePatchNodeTenant(w, req.WithContext(ctx))
    if w.Code != http.StatusForbidden {
        t.Errorf("cross-tenant transfer: expected 403, got %d", w.Code)
    }
}
```

---

### 6.3 Mock-Compatible E2E Framework

**Files to create:**
- `scripts/e2e-mock.sh`
- `internal/testutil/mock_gpu.go`

**Design**: Create E2E scripts that use mock GPU data instead of real hardware:
1. Mock GPU collector returns predefined GPU data
2. Mock Docker client returns predefined container states
3. E2E script tests the full API flow without real GPU

---

### 6.4 Frontend Component Test Foundation

**Files to create:**
- `web/tests/components/DashboardPage.test.ts`
- `web/tests/components/LoginPage.test.ts`

**Design**: Use Vue Test Utils to test component rendering and user interactions:
```ts
import { mount } from '@vue/test-utils'
import DashboardPage from '@/pages/DashboardPage.vue'

describe('DashboardPage', () => {
  it('renders node status', () => {
    const wrapper = mount(DashboardPage, { ... })
    expect(wrapper.find('.status-dot').exists()).toBe(true)
  })
})
```

---

### 6.5 Race Condition Tests

**Files to modify:**
- Add `-race` flag to CI test commands

**Command:**
```bash
go test -race ./... -count=1
```

---

### 6.6 Tenant Isolation Edge Case Tests

**Files to modify:**
- `internal/server/api/tenant_isolation_test.go`

**Tests to add:**
- Cross-tenant deployment access
- Cross-tenant artifact access
- Cross-tenant run-plan access
- Admin bypass verification
- Empty tenant_id handling

---

### Batch 6 Verification

**Commands:**
```bash
go test ./internal/server/auth/... -v -count=1
go test -race ./... -count=1
cd web && npm test
```

**Acceptance criteria:**
- Auth unit tests pass
- Broken tests fixed and passing
- Race detector finds no races
- Frontend component tests pass
- Tenant isolation edge cases pass

**Risks:**
- Mock E2E may not catch real hardware issues. Mitigation: keep real-hardware E2E as manual validation.
- Frontend tests may require significant setup. Mitigation: start with simple rendering tests.

**Rollback:**
- Remove new test files
- Revert broken test fixes

---

## Execution Summary

| Batch | Priority | Effort | Dependencies |
|-------|----------|--------|-------------|
| 1. Security Boundary | P0 | Large | None |
| 2. Docker Lifecycle | P0 | Large | None |
| 3. I/O Hardening | P1 | Medium | Batch 1, 2 |
| 4. RunPlan Config | P1 | Large | None |
| 5. Web UI | P1 | Small | None |
| 6. Test Infrastructure | P1 | Large | Batch 1-5 |

**Total estimated effort**: 6 batches, ~3-4 weeks for a single developer.

**Critical path**: Batch 1 (security) → Batch 2 (stability) → Batch 6 (tests verify all fixes)
