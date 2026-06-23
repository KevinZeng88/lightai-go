# Full Code Research Map

> Date: 2026-06-23
> Purpose: Complete code-level map for all repair batches
> Based on: Actual source code grep/read analysis

---

## Batch 1A: Tenant Scope

### Findings: 5.4 (P0), 5.7 (P1), 5.10 (P1), 5.11 (P2), 5.12 (P2)

### Files to Create
| File | Purpose |
|------|---------|
| `internal/server/authz/checks.go` | Tenant ownership check helpers |
| `internal/server/authz/checks_test.go` | Unit tests |

### Files to Modify (16 endpoints across 6 files)
| File | Endpoints | Change |
|------|-----------|--------|
| `agent_proxy_handlers.go` | HandleProxyNodeFiles (:13), HandleProxyNodeModelScan (:79) | Add node tenant check before proxy |
| `agent_handlers.go` | HandleGetNodeDockerImages (:594), HandleGetNodeDockerImageInspect (:629) | Add node tenant check before proxy |
| `runtime_handlers.go` | HandleListNodeBackendRuntimes (:248), HandleEnableNodeBackendRuntime (:299), HandleRequestNodeBackendRuntimeCheck (:319), HandleGetNodeBackendRuntimeProbe (:645) | Add node/NBR tenant check |
| `node_runtime_handlers.go` | HandlePatchNodeBackendRuntime (:98), HandleDeleteNodeBackendRuntime (:170) | Add NBR tenant check |
| `artifact_handlers.go` | HandleRescanModelLocation (:549), HandleAttestModelLocation (:559) | Add location tenant check |
| `model_browser_handlers.go` | HandleListNodeModelRoots (:162), HandleAddNodeModelRoot (:182), HandlePatchNodeModelRoot (:232), HandleDeleteNodeModelRoot (:261) | Add node/root tenant check |
| `auth/ratelimit.go` | clientIP (:81) | Fix XFF spoofing |
| `auth/handlers.go` | HandleMe (:357) | Fix CSRF rotation |
| `api/observability_handler.go` | HandleObservabilityStatus (:12) | Add auth |

### Existing Tenant Check Pattern (from helpers.go:59-64)
```go
func tenantScopeCheck(r *http.Request, resourceTenantID string) bool {
    if isPlatformAdmin(r) { return true }
    return resourceTenantID == tenantID(r)
}
```

### Endpoint Matrix (Verified by Code)

| # | Handler | File:Line | Route | Path Param | Lookup SQL | Tenant Field | Has Check |
|---|---------|-----------|-------|------------|-----------|-------------|-----------|
| 1 | HandleProxyNodeFiles | agent_proxy_handlers.go:13 | GET /nodes/{id}/files | id | SELECT primary_ip,metrics_port FROM nodes WHERE id=? | nodes.tenant_id | NO |
| 2 | HandleProxyNodeModelScan | agent_proxy_handlers.go:79 | POST /nodes/{id}/model-paths/scan | id | SELECT primary_ip,metrics_port FROM nodes WHERE id=? | nodes.tenant_id | NO |
| 3 | HandleGetNodeDockerImages | agent_handlers.go:594 | GET /nodes/{id}/docker-images | id | SELECT advertised_address,metrics_port FROM nodes WHERE id=? | nodes.tenant_id | NO |
| 4 | HandleGetNodeDockerImageInspect | agent_handlers.go:629 | GET /nodes/{id}/docker-image-inspect | id | SELECT advertised_address,metrics_port FROM nodes WHERE id=? | nodes.tenant_id | NO |
| 5 | HandleListNodeModelRoots | model_browser_handlers.go:162 | GET /nodes/{id}/model-roots | id | nodeTenant(nodeID) then list | node_model_roots.tenant_id | NO |
| 6 | HandleAddNodeModelRoot | model_browser_handlers.go:182 | POST /nodes/{id}/model-roots | id | nodeTenant(nodeID) then INSERT | node_model_roots.tenant_id | NO |
| 7 | HandlePatchNodeModelRoot | model_browser_handlers.go:232 | PATCH /nodes/{id}/model-roots/{root_id} | id,root_id | resolveNodeModelRoot | node_model_roots.tenant_id | NO |
| 8 | HandleDeleteNodeModelRoot | model_browser_handlers.go:261 | DELETE /nodes/{id}/model-roots/{root_id} | id,root_id | resolveNodeModelRoot | node_model_roots.tenant_id | NO |
| 9 | HandleListNodeBackendRuntimes | runtime_handlers.go:248 | GET /nodes/{id}/backend-runtimes | id | SELECT ... FROM node_backend_runtimes WHERE node_id=? | node_backend_runtimes.tenant_id | NO |
| 10 | HandleEnableNodeBackendRuntime | runtime_handlers.go:299 | POST /nodes/{id}/backend-runtimes/enable | id | upsertNodeBackendRuntime | node_backend_runtimes.tenant_id | NO |
| 11 | HandleRequestNodeBackendRuntimeCheck | runtime_handlers.go:319 | POST /nodes/{id}/backend-runtimes/{nbr_id}/check-request | id,nbr_id | SELECT ... WHERE id=? AND node_id=? | node_backend_runtimes.tenant_id | NO |
| 12 | HandleGetNodeBackendRuntimeProbe | runtime_handlers.go:645 | GET /nodes/{id}/backend-runtimes/{nbr_id}/probe | id,nbr_id | SELECT ... WHERE id=? AND node_id=? | node_backend_runtimes.tenant_id | NO |
| 13 | HandlePatchNodeBackendRuntime | node_runtime_handlers.go:98 | PATCH /nodes/{id}/backend-runtimes/{nbr_id} | id,nbr_id | UPDATE ... WHERE id=? | node_backend_runtimes.tenant_id | NO |
| 14 | HandleDeleteNodeBackendRuntime | node_runtime_handlers.go:170 | DELETE /nodes/{id}/backend-runtimes/{nbr_id} | id,nbr_id | SELECT ... WHERE id=? | node_backend_runtimes.tenant_id | NO |
| 15 | HandleRescanModelLocation | artifact_handlers.go:549 | POST /model-artifacts/{id}/locations/{location_id}/rescan | id,location_id | UPDATE model_locations ... WHERE id=? | model_locations.tenant_id | NO |
| 16 | HandleAttestModelLocation | artifact_handlers.go:559 | POST /model-artifacts/{id}/locations/{location_id}/attest | id,location_id | UPDATE model_locations ... WHERE id=? | model_locations.tenant_id | NO |

### Tests to Add
- `internal/server/authz/checks_test.go` — 10 tests
- Extend `internal/server/api/tenant_isolation_test.go` — 10+ tests

### Golden Path Risk
- Same-tenant access must not be blocked
- Platform admin must bypass
- Files/scan/docker-images/docker-inspect via server proxy must work

---

## Batch 1B: AgentClient / SSRF

### Findings: 5.1 (P0)

### Files to Create
| File | Purpose |
|------|---------|
| `internal/server/agentclient/client.go` | AgentClient with SSRF protection |
| `internal/server/agentclient/client_test.go` | Unit tests |

### Files to Modify
| File | Call Sites | Change |
|------|-----------|--------|
| `agent_proxy_handlers.go` | :52 (http.Get), :110 (http.Post) | Replace with agentClient |
| `agent_handlers.go` | :612 (http.Get), :650 (http.Get) | Replace with agentClient |
| `runtime_handlers.go` | :380 (http.Get), :455 (http.Get) | Replace with agentClient |
| `cmd/server/main.go` | init | Initialize AgentClient |

### HTTP Call Replacement Matrix (Verified by Code)

| # | File:Line | Function | Current URL | Endpoint | URL Encoded | Timeout |
|---|-----------|----------|-------------|----------|-------------|---------|
| 1 | agent_proxy_handlers.go:52 | HandleProxyNodeFiles | `http://{ip}:{port}/files?{q.Encode()}` | /files | YES | NO |
| 2 | agent_proxy_handlers.go:110 | HandleProxyNodeModelScan | `http://{ip}:{port}/model-paths/scan` | /model-paths/scan | YES (body) | NO |
| 3 | agent_handlers.go:612 | HandleGetNodeDockerImages | `http://{addr}:{port}/docker-images?query={query}&limit={limit}` | /docker-images | **NO** | NO |
| 4 | agent_handlers.go:650 | HandleGetNodeDockerImageInspect | `http://{addr}:{port}/docker-image-inspect?ref={QueryEscape(ref)}` | /docker-image-inspect | YES | NO |
| 5 | runtime_handlers.go:380 | HandleRequestNodeBackendRuntimeCheck | `http://{agentID}/docker-images?limit=1000` | /docker-images | N/A | NO |
| 6 | runtime_handlers.go:455 | HandleRequestNodeBackendRuntimeCheck | `http://{agentID}/docker-image-inspect?ref={QueryEscape(ref)}` | /docker-image-inspect | YES | NO |

**Not replaced** (different pattern):
- deployment_lifecycle_handlers.go:2225 — already has 30s http.Client, targets instance endpoint
- observability_handler.go:36 — probes localhost Prometheus/Grafana

### AgentClient Interface
```go
type Client struct { httpClient *http.Client; agentToken string }
func New(agentToken string, timeout time.Duration) *Client
func (c *Client) GetJSON(ctx, addr, port, path, params) ([]byte, error)
func (c *Client) PostJSON(ctx, addr, port, path, body) ([]byte, error)
func ValidateAgentAddress(addr string) error
```

### Address Policy
- Deny: 169.254.0.0/16, 0.0.0.0, ::, multicast
- Allow: localhost, private IPs, hostnames
- No strict mode this cycle

### Tests to Add
- `internal/server/agentclient/client_test.go` — 13 tests

### Golden Path Risk
- localhost/private agent must remain reachable
- File browse/scan/docker-images/inspect proxy must work
- URL encoding fix must not break normal queries

---

## Batch 1C: Agent Endpoint Protection / NBR Boundary

### Findings: 5.2 (P1), 5.5 (P1), 5.6 (P1), 7.1 (P1)

### Files to Modify
| File | Change |
|------|--------|
| `cmd/agent/main.go` | Add auth middleware to healthMux handlers |
| `internal/agent/collector/external.go` | Validate collector command path |

### Agent Endpoints (from cmd/agent/main.go:291-476)
| Endpoint | Current Auth | Target Auth |
|----------|-------------|-------------|
| /healthz | None | None (keep) |
| /metrics | None | None (keep, Prometheus) |
| /docker-images | None | **Require token** |
| /docker-image-inspect | None | **Require token** |
| /files | None | **Require token** |
| /model-paths/scan | None | **Require token** |

### NBR Boundary
- Do NOT block NBR-defined params (privileged, ipc, devices, security-opt)
- High-risk params audit/preview only
- No vendor policy engine

### Tests to Add
- Agent endpoint auth tests

---

## Batch 2: Docker Lifecycle / Cleanup / Concurrency

### Findings: 6.1 (P0), 6.2 (P0), 6.3 (P1), 6.4 (P2), 7.4 (P2)

### Files to Modify
| File | Change |
|------|--------|
| `internal/agent/runtime/docker_client.go` | Add ContainerRemove to interface |
| `internal/agent/runtime/docker.go` | Cleanup in Start() failure paths, Remove in Stop() |
| `internal/agent/runtime/docker_real.go` | Implement ContainerRemove |
| `internal/agent/runtime/docker_fake.go` | Implement ContainerRemove |
| `cmd/agent/main.go` | Fix lastStderrBytes race, reconcileState race, task dedup |

### Current Interface (docker_client.go:13-30)
```go
type DockerClient interface {
    ContainerCreate(ctx, opts) (string, error)
    ContainerStart(ctx, containerID) error
    ContainerStop(ctx, containerID, timeoutSeconds) error
    ContainerInspect(ctx, containerID) (*InspectResult, error)
    ContainerLogs(ctx, containerID, opts) (string, string, error)
    // MISSING: ContainerRemove
}
```

### Race Points (Verified by Code)
| Location | Issue | Fix |
|----------|-------|-----|
| cmd/agent/main.go:1121,1197-1199 | `lastStderrBytes` map concurrent R/W | sync.Mutex |
| cmd/agent/main.go:1330-1337,1370-1385 | `reconcileState.unloggedCount` concurrent R/W | atomic.Int32 |
| cmd/agent/main.go:706-715 | Task dispatch no dedup | In-flight task map |

### Cleanup Semantics (Decided)
- Stop: capture logs → stop → remove → release
- Failed start: capture logs → remove
- Restart policy: NOT platform recovery

### Tests to Add
- Start failure cleanup test
- Stop removes container test
- Race detection tests
- Task dedup test

---

## Batch 3: I/O / Audit / Log Safety

### Findings: 5.8 (P1), 5.9 (P1), 7.2 (P2), 7.3 (P2), 10.3 (P2)

### JSON Body Decode Paths (26 locations found)
All use `json.NewDecoder(r.Body).Decode()` without `http.MaxBytesReader`.

Key files: agent_handlers.go (4), deployment_lifecycle_handlers.go (4), artifact_handlers.go (5), runtime_handlers.go (3), backend_handlers.go (2), node_runtime_handlers.go (2), model_browser_handlers.go (2), others.

### Audit Log JSON Issue (agent_handlers.go:882-883)
```go
detail := fmt.Sprintf(`{"from_tenant_id":"%s","to_tenant_id":"%s","reason":"%s"}`,
    currentTenant, req.TenantID, req.Reason)
// Should use json.Marshal
```

### Redaction Issue (helpers.go:225-234)
`redactDetailString` does substring replacement, corrupts `PASSWORD_CHANGED` → `<redacted>_CHANGED`.

### Files to Modify
| File | Change |
|------|--------|
| `cmd/server/main.go` | Add body limit middleware |
| `api/agent_handlers.go` | Fix audit log JSON |
| `api/helpers.go` | Fix redaction logic |
| `agent/runtime/docker_real.go` | Add stream payload limit |
| `cmd/agent/main.go` | Add task result truncation |

### Tests to Add
- Large body → 413
- Audit detail valid JSON
- Redaction preserves action names
- Stream payload limit
- Task truncation marker

---

## Batch 4: RunPlan / Runtime Config / Catalog

### Findings: 4.1 (P1), 4.2 (P1), 4.3 (P1), 4.4 (P2), 4.5 (P2), 4.6-4.8 (P2)

### Key Functions (resolver.go)
| Function | Line | Issue |
|----------|------|-------|
| `deduplicateArgs` | :470 | Boolean flags consumed as values |
| `substituteVars` calls | :557-594 | Layer 5 (env_overrides) not substituted |
| `mapParametersToArgs` | :509 | Required params silently skipped |
| `computeInputHash` | :933 | Missing AssignedGPUs, NodeRuntimeOverride |
| `buildDeviceBinding` | :986 | Dead code, never called |

### Catalog Issues
- `configs/backend-catalog/runtimes/sglang/nvidia-cuda.yaml:6` — stale version ref
- `configs/backend-catalog/versions/ollama/ollama-latest.yaml:15` — raw JSON blob
- `configs/backend-catalog/runtimes/vllm/nvidia-cuda.yaml:24-27` — dead config keys

### Tests to Add
- Boolean flag preservation
- Value flag preservation
- Required param error
- Env substitution
- Hash difference on GPU change

---

## Batch 6: Web / i18n / Permission UX

### Findings: 8.1 (P1), 8.2 (P1), 8.3 (P1), 8.4 (P1), 8.5-8.8 (P2)

### Key Locations (Verified by Code)
| Issue | File:Line | Fix |
|-------|-----------|-----|
| No route guard | router/index.ts | Add beforeEach |
| Hardcoded Chinese | DashboardPage.vue:126,130,134 | i18n keys |
| Hardcoded Chinese | modelCapabilities.js:164-235 | i18n keys |
| Grafana credentials | GrafanaPage.vue:7, locales:322 | Remove or admin-only |
| Permission reset | RolesPage.vue:81-90 | Load existing perms |
| is_platform_admin only | TenantsPage:41, RolesPage:46, UsersPage:54 | Wire hasPermission |

### Tests to Add
- Route guard test
- i18n leakage test
- Component tests

---

## Batch 7: Test Infrastructure

### Current Test State
| Area | Test Files | Coverage |
|------|-----------|----------|
| auth package | 0 test files | Zero unit tests |
| tenant isolation | 1 file (5 tests) | Nodes/GPUs only |
| RunPlan resolver | 1 file (50+ tests) | Strong |
| Docker lifecycle | 1 file | Partial |
| Frontend | 7 files (static analysis) | No component tests |
| E2E | 20 scripts | All require real GPU |

### Tests to Add
- Auth package unit tests (5 files)
- Extend tenant isolation tests
- Fix assertion bugs (9.4, 9.5)
- Mock E2E framework
- Frontend component tests
- Race condition tests

---

## Cross-Batch Conflict Map

| File | Batches | Resolution |
|------|---------|-----------|
| agent_proxy_handlers.go | 1A, 1B | 1A first (tenant check), then 1B (replace http.Get) |
| agent_handlers.go | 1A, 1B, 3 | 1A → 1B → 3 |
| runtime_handlers.go | 1A, 1B | 1A first, then 1B |
| cmd/server/main.go | 1B, 3 | 1B (init AgentClient), then 3 (body limit) |
| cmd/agent/main.go | 1C, 2 | 1C (endpoint auth), then 2 (race fixes) |
| docker_client.go | 2 only | No conflict |
| resolver.go | 4 only | No conflict |
