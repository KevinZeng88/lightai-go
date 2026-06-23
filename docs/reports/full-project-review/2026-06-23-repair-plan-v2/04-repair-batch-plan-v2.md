# Repair Batch Plan V2 — Converged

> Date: 2026-06-23
> Purpose: Converged batch plan — lightweight, product-stage-appropriate, with golden path non-regression gates

---

## Batch Overview (Converged)

| Batch | Name | Strategy | Effort |
|-------|------|----------|--------|
| 1A | Access Control / Tenant Ownership / RBAC Boundary | Execute | 2-3 days |
| 1B | Server → Agent Client / SSRF / Address Policy | Execute | 2-3 days |
| 1C | Agent Endpoint Protection / NBR Execution Boundary | Execute, lightweight | 1-2 days |
| 2 | Docker Lifecycle / Cleanup / Concurrency | Execute | 2-3 days |
| 3 | Input / Output / Audit / Log Safety | Execute | 1-2 days |
| 4 | RunPlan / Runtime Config / Catalog Correctness | Execute, focus bugs | 2-3 days |
| 5 | Gateway / API Key / Usage / Billing | **PAUSED** — future constraint only | 0 days |
| 6 | Web / API Contract / i18n / Permission UX | Execute | 1-2 days |
| 7 | Test Infrastructure | Per-batch + final | 2-3 days |

**Total: ~12-19 days** (reduced from 21-32)

**Execution order**: 1A/1B parallel → 1C → 2 → 3 → 4/6 parallel → 7

---

## Batch 1A: Access Control / Tenant Ownership / RBAC Boundary

### Goal
Add tenant scope checks to 13 missing endpoints. Fix rate limiter, CSRF, observability auth.

### Covered Findings
- 5.4 (P0): 13 endpoints missing tenant scope — **real cross-tenant data access risk**
- 5.7 (P1): Rate limiter XFF spoofing
- 5.10 (P1): Credentials file path
- 5.11 (P2): Observability no auth
- 5.12 (P2): CSRF rotation

### Files to Create
- `internal/server/authz/authorizer.go` — simple tenant scope check helper
- `internal/server/authz/authorizer_test.go`

### Files to Modify
- `internal/server/api/agent_proxy_handlers.go`
- `internal/server/api/agent_handlers.go`
- `internal/server/api/runtime_handlers.go`
- `internal/server/api/node_runtime_handlers.go`
- `internal/server/api/artifact_handlers.go`
- `internal/server/api/model_browser_handlers.go`
- `internal/server/auth/ratelimit.go`
- `internal/server/auth/handlers.go`
- `internal/server/api/observability_handler.go`

### Commit Boundary
1. `internal/server/authz/` package
2. Apply tenant checks to all 13 endpoints
3. Rate limiter, CSRF, observability auth

### Current Flow Non-Regression Checks
| Check | Method |
|-------|--------|
| Same-tenant user accesses own nodes | `GET /api/nodes` returns tenant's nodes |
| Same-tenant user browses files | `GET /api/nodes/{id}/files` |
| Same-tenant user scans models | `POST /api/nodes/{id}/model-paths/scan` |
| Same-tenant user lists Docker images | `GET /api/nodes/{id}/docker-images` |
| Platform admin accesses all | Admin session → all resources |
| Default tenant flow works | Login → nodes → files → scan |
| Rate limiter still works | Multiple failed logins → rate limited |
| CSRF only rotates on login | Multiple `/me` calls → no token invalidation |

### Acceptance Criteria
- [ ] All 13 endpoints have tenant checks
- [ ] Cross-tenant returns 404
- [ ] Golden path flows pass
- [ ] `go test ./internal/server/authz/...` passes
- [ ] `go test ./internal/server/api/... -run Tenant` passes

---

## Batch 1B: Server → Agent Client / SSRF / Address Policy

### Goal
Replace all bare `http.Get()`/`http.Post()` with AgentClient that has timeout, URL encoding, and basic address validation.

### Covered Findings
- 5.1 (P0): SSRF via agent proxy endpoints

### Files to Create
- `internal/server/agentclient/client.go`
- `internal/server/agentclient/client_test.go`

### Files to Modify
- `internal/server/api/agent_proxy_handlers.go`
- `internal/server/api/agent_handlers.go`
- `internal/server/api/runtime_handlers.go`
- `cmd/server/main.go`

### Address Policy (Simplified)
- Deny: 169.254.0.0/16, 0.0.0.0, ::, multicast
- Allow: everything else (localhost, private IPs, registered agents)
- No mode complexity for now — just deny metadata/link-local

### Commit Boundary
1. `internal/server/agentclient/` package
2. Replace all `http.Get()`/`http.Post()` in proxy handlers
3. SSRF tests

### Current Flow Non-Regression Checks
| Check | Method |
|-------|--------|
| Server reaches local agent | `GET /api/nodes/{id}/files` returns data |
| File browse proxy works | `GET /api/nodes/{id}/files?path=/models` |
| Model scan proxy works | `POST /api/nodes/{id}/model-paths/scan` |
| Docker images proxy works | `GET /api/nodes/{id}/docker-images` |
| Docker inspect proxy works | `GET /api/nodes/{id}/docker-image-inspect?ref=...` |
| URL encode fix doesn't break normal queries | Existing queries still work |
| Timeout doesn't fail on large dirs | Scan large directory completes |

### Acceptance Criteria
- [ ] All proxy endpoints use AgentClient
- [ ] Metadata endpoint (169.254.169.254) blocked
- [ ] localhost/private IPs allowed
- [ ] Timeout 30s default
- [ ] Golden path flows pass
- [ ] `go test ./internal/server/agentclient/...` passes

---

## Batch 1C: Agent Endpoint Protection / NBR Execution Boundary

### Goal
Protect agent management endpoints with auth. Ensure NBR-defined runtime parameters flow through without agent-side blocking. Provide traceability for high-risk parameters.

### Covered Findings
- 5.2 (P1): High-risk NBR runtime parameters — **audit/traceability, NOT blocking**
- 5.5 (P1): Agent management endpoints exposure
- 5.6 (P1): AllowRuntimeRootAdd bypass
- 7.1 (P1): Collector command validation

### What This Batch DOES
- Add auth middleware to agent HTTP server for `/docker-images`, `/docker-image-inspect`, `/files`, `/model-paths/scan`
- `AllowRuntimeRootAdd` only works on authenticated requests
- `/healthz` stays unauthenticated
- `/metrics` stays unauthenticated (Prometheus compatible)
- Log high-risk Docker params (privileged, ipc, devices, security-opt) in audit detail
- RunPlan preview displays high-risk parameters for visibility
- Collector command path validation

### What This Batch Does NOT Do
- Do NOT build vendor policy engine
- Do NOT block NBR-defined privileged/ipc/devices/security-opt/group-add parameters
- Do NOT add privileged approval workflow
- Do NOT add MetaX/Huawei/NVIDIA independent allowlist
- Do NOT change current NBR → RunPlan → AgentRunSpec flow
- Do NOT add agent-side deny for admin-configured runtime parameters

### Files to Modify
- `cmd/agent/main.go` — add auth middleware
- `internal/agent/collector/external.go` — validate collector command path

### Commit Boundary
1. Agent HTTP auth middleware
2. Collector command validation

### Current Flow Non-Regression Checks
| Check | Method |
|-------|--------|
| Server proxy still works | `GET /api/nodes/{id}/files` (server→agent) |
| Prometheus `/metrics` works | `curl http://agent:19091/metrics` |
| `/healthz` works | `curl http://agent:19091/healthz` |
| NBR params not blocked | RunPlan preview shows all NBR params |
| MetaX `/dev/mxcd` in NBR → AgentRunSpec | If NBR defines it, flows through |
| NVIDIA `--gpus` in NBR → AgentRunSpec | If NBR defines it, flows through |
| `--privileged` in NBR → AgentRunSpec | If NBR defines it, flows through |
| High-risk params in audit log | Audit detail includes privileged/ipc/devices |

### Acceptance Criteria
- [ ] Agent management endpoints require auth
- [ ] `/healthz` and `/metrics` unauthenticated
- [ ] NBR params flow through unblocked
- [ ] High-risk params logged in audit
- [ ] Golden path flows pass

---

## Batch 2: Docker Lifecycle / Cleanup / Concurrency

### Goal
Fix container cleanup, stop/remove semantics, race conditions, task dedup.

### Covered Findings
- 6.1 (P0): Container not cleaned up after start failure
- 6.2 (P0): Race on logsTaskState map
- 6.3 (P1): No container removal on stop
- 6.4 (P2): Race on reconcileState
- 7.4 (P2): No task dedup

### Cleanup Semantics (Decided)
- **Stop**: capture logs → stop → remove → release resources
- **Failed start**: capture logs → remove
- **Restart policy**: NOT platform recovery mechanism

### Files to Modify
- `internal/agent/runtime/docker_client.go` — add ContainerRemove
- `internal/agent/runtime/docker.go` — cleanup in failure paths, remove on stop
- `internal/agent/runtime/docker_real.go` — implement ContainerRemove
- `internal/agent/runtime/docker_fake.go` — implement ContainerRemove
- `cmd/agent/main.go` — race fixes, task dedup

### Commit Boundary
1. Add ContainerRemove to interface
2. Cleanup in Start() failure paths
3. Remove in Stop()
4. Race condition fixes
5. Task dedup

### Current Flow Non-Regression Checks
| Check | Method |
|-------|--------|
| Normal start succeeds | Deploy → instance running → `/v1/models` |
| Normal stop removes container | Stop → `docker ps -a` no container |
| Restart after stop works | Stop → Start → no name conflict |
| Failed start cleanup works | Failure → container removed → logs captured |
| Logs viewable before cleanup | Failed instance → logs accessible |
| `go test -race` passes | `go test -race ./internal/agent/...` |

### Acceptance Criteria
- [ ] ContainerRemove in DockerClient
- [ ] Start() cleans up on failure
- [ ] Stop() removes container
- [ ] Race conditions fixed
- [ ] Task dedup works
- [ ] Golden path flows pass
- [ ] `go test -race ./internal/agent/... ./cmd/agent/...` passes

---

## Batch 3: Input / Output / Audit / Log Safety

### Goal
Add body size limits, fix audit log JSON, add stream/task size limits, fix redaction.

### Covered Findings
- 5.8 (P1): No body size limit
- 5.9 (P1): JSON injection in audit log
- 7.2 (P2): Docker stream payload limit
- 7.3 (P2): Task result size limit
- 10.3 (P2): Redaction logic incorrect

### Files to Modify
- `cmd/server/main.go` — body limit middleware
- `internal/server/api/agent_handlers.go` — audit JSON
- `internal/server/api/helpers.go` — redaction
- `internal/agent/runtime/docker_real.go` — stream limit
- `cmd/agent/main.go` — task result truncation

### Commit Boundary
1. Body limit middleware
2. Audit log JSON fix
3. Stream/task limits
4. Redaction rewrite

### Current Flow Non-Regression Checks
| Check | Method |
|-------|--------|
| Normal API requests work | Standard CRUD operations |
| Audit log is valid JSON | `GET /api/audit-logs` → parse detail |
| `PASSWORD_CHANGED` not corrupted | Audit action name preserved |
| Normal logs not over-truncated | Instance logs viewable |
| Large log has truncation marker | `... [truncated]` appears |
| Docker stream parsing works | Container logs readable |

### Acceptance Criteria
- [ ] Body limit 10MB default
- [ ] Audit log uses json.Marshal
- [ ] Stream payload limited
- [ ] Task results truncated
- [ ] Redaction parses key-value pairs
- [ ] Golden path flows pass

---

## Batch 4: RunPlan / Runtime Config / Catalog Correctness

### Goal
Fix RunPlan resolver bugs. Focus on current bugs, not full architecture redesign.

### Covered Findings
- 4.1 (P1): Boolean flag dedup
- 4.2 (P1): Env overrides bypass substitution
- 4.3 (P1): Required params silently skipped
- 4.4 (P2): buildDeviceBinding dead code
- 4.5 (P2): computeInputHash missing fields
- 4.6-4.8 (P2): Catalog cleanup

### What This Batch Does
- Fix deduplicateArgs for boolean flags
- Apply substituteVars to layer 5
- Add required param error propagation
- Fix computeInputHash
- Remove dead buildDeviceBinding
- Clean up catalog YAML

### What This Batch Does NOT Do
- NOT full three-layer RunPlan
- NOT scheduler
- NOT APICompatibilityProfile
- NOT RuntimeRequirements redesign

### Files to Modify
- `internal/server/runplan/resolver.go`
- `configs/backend-catalog/`

### Commit Boundary
1. deduplicateArgs fix
2. Env substitution + required params
3. Hash fix + dead code removal
4. Catalog cleanup

### Current Flow Non-Regression Checks
| Check | Method |
|-------|--------|
| vLLM RunPlan tests pass | `go test ./internal/server/runplan/...` |
| SGLang RunPlan tests pass | Same |
| llama.cpp RunPlan tests pass | Same |
| Boolean flags preserved | `--trust-remote-code` in args |
| Value flags preserved | `--model /path` in args |
| Required param error works | Missing `--model` → error |
| Env substitution works | `{{MODEL_CONTAINER_PATH}}` substituted |
| Equivalent Docker command correct | Preview matches AgentRunSpec |

### Acceptance Criteria
- [ ] Boolean flags handled correctly
- [ ] Layer 5 env substitution works
- [ ] Required params produce errors
- [ ] Hash includes all fields
- [ ] Dead code removed
- [ ] Catalog consistent
- [ ] Golden path flows pass
- [ ] All RunPlan tests pass

---

## Batch 5: Unified Model API / Gateway / Audit / Usage / Billing — PAUSED

**Status**: Not an execution batch. Future constraint only.

**Why paused**: Current priority is runtime chain, security boundaries, lifecycle, RunPlan correctness.

**What to retain as future constraints** (in other batch docs where relevant):
- Current priority: OpenAI-compatible API entry
- Future: must not hardcode OpenAI-only
- Don't hardcode deployment name as API model name
- Don't hardcode actor as session user
- Don't hardcode usage as token-only

---

## Batch 6: Web / API Contract / i18n / Permission UX

### Goal
Fix frontend security, i18n, and permission UX issues.

### Covered Findings
- 8.1 (P1): Hardcoded Chinese
- 8.2 (P1): Default credentials in UI
- 8.3 (P1): No route guard
- 8.4 (P1): RolesPage permission reset
- 8.5 (P2): Hardcoded ports
- 8.6 (P2): No confirmation dialogs
- 8.7 (P2): Stale cache
- 8.8 (P2): Unused permissions

### Files to Modify
- `web/src/router/index.ts`
- `web/src/pages/DashboardPage.vue`
- `web/src/pages/GrafanaPage.vue`
- `web/src/pages/RolesPage.vue`
- `web/src/utils/modelCapabilities.js`
- `web/src/stores/auth.ts`
- `web/src/locales/zh-CN.ts`
- `web/src/locales/en-US.ts`

### Commit Boundary
1. Route guard
2. Credentials removal
3. i18n fixes
4. Permission wiring

### Current Flow Non-Regression Checks
| Check | Method |
|-------|--------|
| Login works | Browser: login → console |
| Route guard no login loop | Navigate to `/dashboard` → loads |
| Dashboard accessible | Browser: `/dashboard` |
| Nodes page accessible | Browser: `/nodes` |
| Runtimes pages accessible | Browser: `/backend-runtimes` |
| i18n no key leakage | Switch to en-US → no raw keys |
| Grafana page loads | Browser: `/observability/grafana` |
| No credentials for regular users | `admin/admin` not shown |
| Stop/restart confirmation works | Dialog appears → proceeds |

### Acceptance Criteria
- [ ] Route guard exists
- [ ] No credentials in UI
- [ ] No hardcoded Chinese
- [ ] Permission dialog pre-selects
- [ ] Golden path flows pass
- [ ] `cd web && npm test` passes

---

## Batch 7: Test Infrastructure (Per-Batch + Final)

### Goal
Add tests alongside each batch. Finalize mock E2E at end.

### Per-Batch Test Additions

| Batch | Tests Added |
|-------|-------------|
| 1A | authz unit tests, tenant isolation tests |
| 1B | AgentClient SSRF tests |
| 1C | Agent endpoint auth tests |
| 2 | Docker lifecycle mock tests, race tests |
| 3 | Body limit tests, redaction tests |
| 4 | RunPlan resolver tests (fix existing) |
| 6 | Frontend component tests |

### Final Test Infrastructure
- Mock E2E framework (`scripts/e2e-mock-smoke.sh`)
- `go test -race` as standard verification
- Golden path verification script

### Current Flow Non-Regression Checks
| Check | Method |
|-------|--------|
| Auth unit tests pass | `go test ./internal/server/auth/...` |
| Tenant tests pass | `go test ./internal/server/api/... -run Tenant` |
| RunPlan tests pass | `go test ./internal/server/runplan/...` |
| Docker tests pass | `go test ./internal/agent/runtime/...` |
| `go test -race` passes | `go test -race ./...` |
| Mock E2E passes | `scripts/e2e-mock-smoke.sh` |
| Real GPU E2E runnable | `scripts/e2e-real-smoke-all-three.sh` (manual) |

### Acceptance Criteria
- [ ] All new tests pass
- [ ] `go test -race` passes
- [ ] Mock E2E passes
- [ ] Real GPU E2E still runnable

---

## Concurrency Strategy

### Can Parallel
- New packages (`authz/`, `agentclient/`)
- Test files
- Frontend fixes
- RunPlan resolver tests
- DockerClient fake tests

### Must Serialize (Shared Files)
- Server API handlers (1A, 1B, 1C all touch)
- Agent main.go (1C, 2 both touch)
- Docker runtime driver (2 touches)
- Runplan resolver.go (4 touches)
- Router (1A touches)

### Recommended Flow
```
[1A: authz package] ──────────┐
[1B: agentclient package] ────┼──→ [Integration: server handlers]
[1C: agent endpoint auth] ────┘

[2: DockerClient interface] ──→ [Integration: docker.go, main.go]

[3: middleware + local fixes] ──→ [Independent commits]

[4: resolver tests first] ──→ [Integration: resolver.go]

[6: frontend] ──→ [Independent]

[7: tests] ──→ [Per-batch, no conflicts]
```

---

## Closeout Requirements (Per Batch)

Each batch closeout MUST contain:

1. **Before baseline**: git SHA, working flows, commands, evidence paths
2. **Changes made**: files, functions, logic
3. **After verification**: same flows still work, commands pass
4. **Golden path check**: explicit pass/fail per flow
5. **If old script/config deleted**: what replaces it
6. **If flow can't verify**: why + dry-run/mock evidence
7. **Test results**: unit, integration, race
8. **Commit SHAs**: all commits in this batch

Closeout document: `docs/reports/full-project-review/2026-06-23-repair-plan-v2/batch-{id}-closeout.md`
