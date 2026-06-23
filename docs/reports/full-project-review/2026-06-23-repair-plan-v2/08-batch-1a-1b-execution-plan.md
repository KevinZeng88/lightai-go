# Batch 1A + 1B Execution Plan

> Date: 2026-06-23
> Purpose: Precise execution preparation for Batch 1A (Tenant Ownership) and Batch 1B (AgentClient/SSRF)
> **This is preparation only. No code changes.**

---

## Batch 1A: Access Control / Tenant Ownership

### 1. Endpoint → Resource → Tenant Check Matrix

The explore agent found **16 handler entry points** missing tenant scope. The original review cited 13; 3 additional were found (HandleListNodeModelRoots, HandleAddNodeModelRoot, HandleAttestModelLocation).

#### Domain A: Node Proxy (4 endpoints)

These look up `nodes` table by `node_id` then proxy to agent. Tenant check: verify `nodes.tenant_id` matches caller.

| # | Handler | File:Line | URL Pattern | Path Param | Lookup | Tenant Field | Fix |
|---|---------|-----------|-------------|------------|--------|-------------|-----|
| 1 | `HandleProxyNodeFiles` | agent_proxy_handlers.go:13 | `GET /nodes/{id}/files` | `id` | `SELECT primary_ip, metrics_port FROM nodes WHERE id=?` | `nodes.tenant_id` | Check node tenant before proxy |
| 2 | `HandleProxyNodeModelScan` | agent_proxy_handlers.go:79 | `POST /nodes/{id}/model-paths/scan` | `id` | `SELECT primary_ip, metrics_port FROM nodes WHERE id=?` | `nodes.tenant_id` | Check node tenant before proxy |
| 3 | `HandleGetNodeDockerImages` | agent_handlers.go:594 | `GET /nodes/{id}/docker-images` | `id` | `SELECT advertised_address, metrics_port FROM nodes WHERE id=?` | `nodes.tenant_id` | Check node tenant before proxy |
| 4 | `HandleGetNodeDockerImageInspect` | agent_handlers.go:629 | `GET /nodes/{id}/docker-image-inspect` | `id` | `SELECT advertised_address, metrics_port FROM nodes WHERE id=?` | `nodes.tenant_id` | Check node tenant before proxy |

**Fix pattern**: Before the existing DB query, add:
```go
var nodeTenant string
h.DB.QueryRow("SELECT tenant_id FROM nodes WHERE id=?", nodeID).Scan(&nodeTenant)
if !tenantScopeCheck(r, nodeTenant) {
    http.Error(w, "not found", http.StatusNotFound)
    return
}
```

#### Domain B: Node Model Roots (4 endpoints)

These look up `node_model_roots` by `node_id` and/or `root_id`. Tenant check: verify `node_model_roots.tenant_id` matches caller.

| # | Handler | File:Line | URL Pattern | Path Params | Lookup | Tenant Field | Fix |
|---|---------|-----------|-------------|-------------|--------|-------------|-----|
| 5 | `HandleListNodeModelRoots` | model_browser_handlers.go:162 | `GET /nodes/{id}/model-roots` | `id` | `nodeTenant(nodeID)` then list | `node_model_roots.tenant_id` | Check node tenant |
| 6 | `HandleAddNodeModelRoot` | model_browser_handlers.go:182 | `POST /nodes/{id}/model-roots` | `id` | `nodeTenant(nodeID)` then INSERT | `node_model_roots.tenant_id` | Check node tenant |
| 7 | `HandlePatchNodeModelRoot` | model_browser_handlers.go:232 | `PATCH /nodes/{id}/model-roots/{root_id}` | `id`, `root_id` | `resolveNodeModelRoot(nodeID, rootID, "")` | `node_model_roots.tenant_id` | Check root tenant |
| 8 | `HandleDeleteNodeModelRoot` | model_browser_handlers.go:261 | `DELETE /nodes/{id}/model-roots/{root_id}` | `id`, `root_id` | `resolveNodeModelRoot(nodeID, rootID, "")` | `node_model_roots.tenant_id` | Check root tenant |

**Fix pattern**: For list/add, check node tenant. For patch/delete, check root tenant via `resolveNodeModelRoot` return.

#### Domain C: Node Backend Runtimes (7 endpoints)

These look up `node_backend_runtimes` by `node_id` and/or `nbr_id`. Tenant check: verify `node_backend_runtimes.tenant_id` matches caller.

| # | Handler | File:Line | URL Pattern | Path Params | Lookup | Tenant Field | Fix |
|---|---------|-----------|-------------|-------------|--------|-------------|-----|
| 9 | `HandleListNodeBackendRuntimes` | runtime_handlers.go:248 | `GET /nodes/{id}/backend-runtimes` | `id` | `SELECT ... FROM node_backend_runtimes WHERE node_id=?` | `node_backend_runtimes.tenant_id` | Check node tenant |
| 10 | `HandleEnableNodeBackendRuntime` | runtime_handlers.go:299 | `POST /nodes/{id}/backend-runtimes/enable` | `id` | `upsertNodeBackendRuntime` (implicit node) | `node_backend_runtimes.tenant_id` | Check node tenant |
| 11 | `HandleRequestNodeBackendRuntimeCheck` | runtime_handlers.go:319 | `POST /nodes/{id}/backend-runtimes/{nbr_id}/check-request` | `id`, `nbr_id` | `SELECT ... WHERE id=? AND node_id=?` | `node_backend_runtimes.tenant_id` | Check NBR tenant |
| 12 | `HandleGetNodeBackendRuntimeProbe` | runtime_handlers.go:645 | `GET /nodes/{id}/backend-runtimes/{nbr_id}/probe` | `id`, `nbr_id` | `SELECT ... WHERE id=? AND node_id=?` | `node_backend_runtimes.tenant_id` | Check NBR tenant |
| 13 | `HandlePatchNodeBackendRuntime` | node_runtime_handlers.go:98 | `PATCH /nodes/{id}/backend-runtimes/{nbr_id}` | `id`, `nbr_id` | `UPDATE ... WHERE id=?` (by nbr_id only) | `node_backend_runtimes.tenant_id` | Check NBR tenant |
| 14 | `HandleDeleteNodeBackendRuntime` | node_runtime_handlers.go:170 | `DELETE /nodes/{id}/backend-runtimes/{nbr_id}` | `id`, `nbr_id` | `SELECT ... WHERE id=?` | `node_backend_runtimes.tenant_id` | Check NBR tenant |

**Fix pattern**: For list/enable, check node tenant. For check/probe/patch/delete, check NBR tenant.

#### Domain D: Model Location (2 endpoints)

These look up `model_locations` by `location_id`. Tenant check: verify `model_locations.tenant_id` matches caller.

| # | Handler | File:Line | URL Pattern | Path Params | Lookup | Tenant Field | Fix |
|---|---------|-----------|-------------|-------------|--------|-------------|-----|
| 15 | `HandleRescanModelLocation` | artifact_handlers.go:549 | `POST /model-artifacts/{id}/locations/{location_id}/rescan` | `id`, `location_id` | `UPDATE model_locations ... WHERE id=?` | `model_locations.tenant_id` | Check location tenant |
| 16 | `HandleAttestModelLocation` | artifact_handlers.go:559 | `POST /model-artifacts/{id}/locations/{location_id}/attest` | `id`, `location_id` | `UPDATE model_locations ... WHERE id=?` | `model_locations.tenant_id` | Check location tenant |

**Fix pattern**: Check `model_locations.tenant_id` before update.

### 2. Authz Helper Design

**Approach**: Lightweight helpers, not a framework.

#### New package: `internal/server/authz/`

```go
// internal/server/authz/checks.go

package authz

import (
    "database/sql"
    "net/http"
    "github.com/user/lightai-go/internal/server/auth"
)

// CheckNodeTenant verifies the node belongs to the caller's tenant.
// Returns true if authorized (including platform admin bypass).
func CheckNodeTenant(r *http.Request, db *sql.DB, nodeID string) bool {
    if isPlatformAdmin(r) {
        return true
    }
    var tid string
    err := db.QueryRow("SELECT tenant_id FROM nodes WHERE id=?", nodeID).Scan(&tid)
    if err != nil {
        return false // node not found
    }
    return tid == tenantID(r)
}

// CheckNBRTenant verifies the NBR belongs to the caller's tenant.
func CheckNBRTenant(r *http.Request, db *sql.DB, nbrID string) bool {
    if isPlatformAdmin(r) {
        return true
    }
    var tid string
    err := db.QueryRow("SELECT tenant_id FROM node_backend_runtimes WHERE id=?", nbrID).Scan(&tid)
    if err != nil {
        return false
    }
    return tid == tenantID(r)
}

// CheckModelRootTenant verifies the model root belongs to the caller's tenant.
func CheckModelRootTenant(r *http.Request, db *sql.DB, rootID string) bool {
    if isPlatformAdmin(r) {
        return true
    }
    var tid string
    err := db.QueryRow("SELECT tenant_id FROM node_model_roots WHERE id=?", rootID).Scan(&tid)
    if err != nil {
        return false
    }
    return tid == tenantID(r)
}

// CheckModelLocationTenant verifies the model location belongs to the caller's tenant.
func CheckModelLocationTenant(r *http.Request, db *sql.DB, locationID string) bool {
    if isPlatformAdmin(r) {
        return true
    }
    var tid string
    err := db.QueryRow("SELECT tenant_id FROM model_locations WHERE id=?", locationID).Scan(&tid)
    if err != nil {
        return false
    }
    return tid == tenantID(r)
}

func tenantID(r *http.Request) string {
    info := auth.SessionInfoFromContext(r.Context())
    if info == nil {
        return ""
    }
    return info.TenantID
}

func isPlatformAdmin(r *http.Request) bool {
    info := auth.SessionInfoFromContext(r.Context())
    return info != nil && info.IsPlatformAdmin
}
```

**Why not a unified `CheckTenantOwnership(resourceType, resourceID)`**: Each resource type has a different table and query. A unified function would need a switch/map lookup, adding complexity without benefit for 4 resource types. Explicit functions are clearer and testable.

**Why not middleware**: Some endpoints need the tenant check AFTER extracting the resource (e.g., `HandlePatchNodeModelRoot` needs rootID from path). Middleware would need to parse the path twice. Handler-level calls are simpler.

### 3. Batch 1A Test Plan

#### New test file: `internal/server/authz/checks_test.go`

Tests:
- `TestCheckNodeTenant_SameTenant` → returns true
- `TestCheckNodeTenant_CrossTenant` → returns false
- `TestCheckNodeTenant_PlatformAdmin` → returns true
- `TestCheckNodeTenant_NodeNotFound` → returns false
- `TestCheckNBRTenant_SameTenant` → returns true
- `TestCheckNBRTenant_CrossTenant` → returns false
- `TestCheckModelRootTenant_SameTenant` → returns true
- `TestCheckModelRootTenant_CrossTenant` → returns false
- `TestCheckModelLocationTenant_SameTenant` → returns true
- `TestCheckModelLocationTenant_CrossTenant` → returns false

#### Integration tests: extend `internal/server/api/tenant_isolation_test.go`

Add:
- `TestTenantProxyNodeFiles_CrossTenant` → 404
- `TestTenantProxyNodeModelScan_CrossTenant` → 404
- `TestTenantDockerImages_CrossTenant` → 404
- `TestTenantDockerImageInspect_CrossTenant` → 404
- `TestTenantNBRList_CrossTenant` → 404 (or empty list)
- `TestTenantNBRPatch_CrossTenant` → 404
- `TestTenantNBRDelete_CrossTenant` → 404
- `TestTenantModelRootPatch_CrossTenant` → 404
- `TestTenantModelRootDelete_CrossTenant` → 404
- `TestTenantModelLocationRescan_CrossTenant` → 404

#### Suggested commands

```bash
go test ./internal/server/authz/...
go test ./internal/server/api/... -run 'Tenant'
go test ./internal/server/api/... -run 'Node|Runtime|Model'
```

---

## Batch 1B: AgentClient / SSRF / Address Policy

### 1. Bare HTTP Call Replacement Matrix

| # | File:Line | Function | Current URL | Agent Endpoint | Query Params | URL Encoded | Timeout | Replace With |
|---|-----------|----------|-------------|----------------|--------------|-------------|---------|-------------|
| 1 | agent_proxy_handlers.go:51 | `HandleProxyNodeFiles` | `http://{ip}:{port}/files?{q.Encode()}` | `/files` | path, glob | YES | NO | `agentClient.GetJSON(ctx, ip, port, "/files", q)` |
| 2 | agent_proxy_handlers.go:109 | `HandleProxyNodeModelScan` | `http://{ip}:{port}/model-paths/scan` | `/model-paths/scan` | JSON body | YES (body) | NO | `agentClient.PostJSON(ctx, ip, port, "/model-paths/scan", body)` |
| 3 | agent_handlers.go:611 | `HandleGetNodeDockerImages` | `http://{addr}:{port}/docker-images?query={query}&limit={limit}` | `/docker-images` | query, limit | **NO** | NO | `agentClient.GetJSON(ctx, addr, port, "/docker-images", q)` |
| 4 | agent_handlers.go:649 | `HandleGetNodeDockerImageInspect` | `http://{addr}:{port}/docker-image-inspect?ref={QueryEscape(ref)}` | `/docker-image-inspect` | ref | YES | NO | `agentClient.GetJSON(ctx, addr, port, "/docker-image-inspect", q)` |
| 5 | runtime_handlers.go:379 | `HandleRequestNodeBackendRuntimeCheck` | `http://{agentID}/docker-images?limit=1000` | `/docker-images` | limit (static) | N/A | NO | `agentClient.GetJSON(ctx, addr, port, "/docker-images", q)` |
| 6 | runtime_handlers.go:454 | `HandleRequestNodeBackendRuntimeCheck` | `http://{agentID}/docker-image-inspect?ref={QueryEscape(ref)}` | `/docker-image-inspect` | ref | YES | NO | `agentClient.GetJSON(ctx, addr, port, "/docker-image-inspect", q)` |

**Not replaced** (different pattern):
- `deployment_lifecycle_handlers.go:2225` — already has 30s `http.Client`, targets instance endpoint (not agent)
- `observability_handler.go:25,29` — probes localhost Prometheus/Grafana (not agent)

### 2. AgentClient Lightweight Design

```go
// internal/server/agentclient/client.go

package agentclient

import (
    "context"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "time"
)

// Default timeouts
const (
    DefaultTimeout      = 30 * time.Second
    FileScanTimeout     = 120 * time.Second
    MaxResponseBytes    = 100 * 1024 * 1024 // 100MB
)

// Denied CIDRs — always blocked regardless of mode
var deniedCIDRs = []*net.IPNet{
    parseCIDR("169.254.0.0/16"), // link-local / cloud metadata
    parseCIDR("0.0.0.0/32"),     // unspecified
    parseCIDR("::/128"),          // unspecified v6
    parseCIDR("224.0.0.0/4"),    // multicast
    parseCIDR("ff00::/8"),       // multicast v6
}

type Client struct {
    httpClient *http.Client
    agentToken string
}

func New(agentToken string, timeout time.Duration) *Client {
    return &Client{
        httpClient: &http.Client{Timeout: timeout},
        agentToken: agentToken,
    }
}

// GetJSON performs a GET request and returns the response body.
func (c *Client) GetJSON(ctx context.Context, addr string, port int, path string, params url.Values) ([]byte, error) {
    if err := ValidateAgentAddress(addr); err != nil {
        return nil, fmt.Errorf("agent address rejected: %w", err)
    }
    u := fmt.Sprintf("http://%s:%d%s", addr, port, path)
    if params != nil {
        u += "?" + params.Encode()
    }
    req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
    if err != nil {
        return nil, err
    }
    if c.agentToken != "" {
        req.Header.Set("Authorization", "Bearer "+c.agentToken)
    }
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes))
    if err != nil {
        return nil, err
    }
    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("agent returned %d: %s", resp.StatusCode, string(body))
    }
    return body, nil
}

// PostJSON performs a POST request with JSON body.
func (c *Client) PostJSON(ctx context.Context, addr string, port int, path string, body io.Reader) ([]byte, error) {
    if err := ValidateAgentAddress(addr); err != nil {
        return nil, fmt.Errorf("agent address rejected: %w", err)
    }
    u := fmt.Sprintf("http://%s:%d%s", addr, port, path)
    req, err := http.NewRequestWithContext(ctx, "POST", u, body)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    if c.agentToken != "" {
        req.Header.Set("Authorization", "Bearer "+c.agentToken)
    }
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    respBody, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes))
    if err != nil {
        return nil, err
    }
    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("agent returned %d: %s", resp.StatusCode, string(respBody))
    }
    return respBody, nil
}

// ValidateAgentAddress checks if an address is allowed.
// Denied: metadata, link-local, unspecified, multicast
// Allowed: everything else (localhost, private IPs, registered agents)
func ValidateAgentAddress(addr string) error {
    ip := net.ParseIP(addr)
    if ip == nil {
        // hostname — allowed (DNS resolution happens at dial time)
        return nil
    }
    for _, cidr := range deniedCIDRs {
        if cidr.Contains(ip) {
            return fmt.Errorf("address %s is in denied range %s", addr, cidr)
        }
    }
    return nil
}

func parseCIDR(s string) *net.IPNet {
    _, n, _ := net.ParseCIDR(s)
    return n
}
```

**Key design decisions**:
- No complex mode system (dev/LAN/production) — just deny metadata/link-local
- Private IPs allowed — multi-server deployments run on LAN
- Hostnames allowed — DNS resolution happens at dial time
- Timeout configurable per client instance
- Response body limited to 100MB
- Agent token passed as Bearer header
- No mTLS this cycle

### 3. Batch 1B Test Plan

#### New test file: `internal/server/agentclient/client_test.go`

Tests:
- `TestValidateAgentAddress_Localhost` → allowed
- `TestValidateAgentAddress_PrivateIP` → allowed (10.0.0.1, 192.168.1.1)
- `TestValidateAgentAddress_Metadata` → denied (169.254.169.254)
- `TestValidateAgentAddress_LinkLocal` → denied (169.254.0.1)
- `TestValidateAgentAddress_Unspecified` → denied (0.0.0.0, ::)
- `TestValidateAgentAddress_Multicast` → denied (224.0.0.1)
- `TestValidateAgentAddress_Hostname` → allowed
- `TestGetJSON_Success` → mock server returns 200
- `TestGetJSON_Timeout` → mock server hangs → timeout error
- `TestGetJSON_ResponseLimit` → mock server returns >100MB → truncated
- `TestGetJSON_DeniedAddress` → metadata address → error
- `TestGetJSON_URLEncoding` → special chars in params → properly encoded
- `TestPostJSON_Success` → mock server returns 200

#### Suggested commands

```bash
go test ./internal/server/agentclient/...
go test ./internal/server/api/... -run 'Proxy|Files|DockerImages|ModelScan'
```

---

## Concurrency & Integration Strategy

### Can Parallel

| Work | Reason |
|------|--------|
| `internal/server/authz/` package | New file, no conflicts |
| `internal/server/agentclient/` package | New file, no conflicts |
| authz tests | Independent test file |
| agentclient tests | Independent test file |

### Must Serialize (Shared Files)

| File | Why | Resolution |
|------|-----|-----------|
| `agent_proxy_handlers.go` | 1A adds tenant check, 1B replaces http.Get | 1A first, then 1B |
| `agent_handlers.go` | 1A adds tenant check, 1B replaces http.Get | 1A first, then 1B |
| `runtime_handlers.go` | 1A adds tenant check, 1B replaces http.Get | 1A first, then 1B |
| `cmd/server/main.go` | 1B initializes AgentClient | After 1B package ready |

### Recommended Commit Sequence

**Batch 1A (3 commits)**:
1. `feat: add authz package with tenant ownership checks` — new `internal/server/authz/` files
2. `feat: add tenant scope checks to 16 endpoints` — modify 6 handler files
3. `fix: rate limiter, CSRF rotation, observability auth` — modify auth files

**Batch 1B (3 commits)**:
4. `feat: add agentclient package with SSRF protection` — new `internal/server/agentclient/` files
5. `feat: replace bare http.Get/Post with AgentClient` — modify handler files
6. `test: add agentclient SSRF and integration tests` — test files

**Conflict resolution**: Commits 2 and 5 both modify handler files. Execute commit 2 first, then commit 5 rebase on top.

---

## Risk Assessment

### Batch 1A Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Tenant check blocks same-tenant access | LOW | HIGH | Test with default tenant; golden path verification |
| Tenant check breaks admin access | LOW | HIGH | Platform admin bypass in all helpers |
| New DB query adds latency | LOW | LOW | Single indexed query per request |
| Missing tenant_id on some resources | LOW | MEDIUM | Verify all resources have tenant_id column |

### Batch 1B Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| AgentClient blocks localhost in dev | LOW | HIGH | ValidateAgentAddress allows localhost |
| AgentClient blocks private IPs | LOW | HIGH | ValidateAgentAddress allows private IPs |
| Timeout too short for large scans | MEDIUM | MEDIUM | FileScanTimeout = 120s |
| Response body limit too small | LOW | MEDIUM | 100MB default, configurable |
| URL encoding changes break queries | LOW | MEDIUM | Use url.Values.Encode() consistently |
| Agent token not available in context | LOW | HIGH | Pass token from server config |
