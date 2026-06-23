# Execution Scope Reduction and Decisions — Repair Plan V2 Convergence

> Date: 2026-06-23
> Purpose: Converge V2 documents into a lightweight, product-stage-appropriate execution plan

---

## 1. Convergence Principles

### 1.1 No Over-Engineering

LightAI Go is a **lightweight GPU/model management platform for internal network environments**. It is NOT:
- A public cloud multi-tenant security platform
- An internet SaaS
- A Kubernetes orchestrator

Therefore:
- Do not amplify every security finding into P0
- Do not design heavy policy engines for theoretical attack surfaces
- Do not introduce manual approval workflows
- Do not build complex vendor-specific policy systems
- Do not design complete billing systems just because they might exist someday
- Do not retain complex fallbacks for legacy configs/schemas

**Preserve future extensibility awareness, but current execution plan must be lightweight, direct, implementable, and verifiable.**

### 1.2 No Backward Compatibility Burden — BUT No Breaking Current Flows

**"No backward compatibility" means:**
- Do NOT preserve compatibility with old BackendRuntime configs
- Do NOT preserve compatibility with old NodeBackendRuntime configs
- Do NOT retain old fallbacks
- Do NOT do complex data migrations for old data
- If schema changes: rebuild tables/database per current project rules
- If old configs cause problems: clean up or rebuild, don't add compatibility branches

**"No breaking current flows" means:**
- All repairs, refactors, cleanup, schema changes MUST preserve the currently working golden path
- If old config is deleted, must update current catalog/seed/docs/tests/scripts synchronously
- Current mainline flows must migrate to new clean model and continue working
- This is NOT permission to break current capabilities — it's permission to remove legacy, not current

### 1.3 Current Golden Path Non-Regression Gate — HARD PRINCIPLE

**Every repair batch must verify: the currently working core flows still work after the change.**

This is a hard gate. If a repair breaks a current flow, the repair is wrong — not the flow.

See `07-golden-path-non-regression.md` for the complete golden path definition and per-batch verification requirements.

### 1.3 Security Issues Re-evaluated by Real Threat Model

Current primary environment: **internal network, controlled nodes, trusted admin operations**.

**Must fix**:
- Cross-tenant access that affects multi-tenant/multi-business-line isolation
- Agent management endpoints (file browse, model scan, Docker inspect) exposed to any network client
- Server→agent HTTP calls without timeout/URL encoding/basic address constraints (causes instability)
- Docker lifecycle leaks, races, task dedup (causes real runtime failures)
- RunPlan parameter errors, required param silent skip, hash errors (causes real startup failures)

**Can downgrade or record**:
- Agent token strength: bootstrap mechanism exists, internal network, not current priority
- Grafana default credentials: operational security/experience issue, not same tier as SSRF/cross-tenant
- Hardcoded Chinese: quality/i18n issue, not P0
- Gateway/Billing/Usage: future product capabilities, not current repair batch
- Theoretical public cloud SSRF rules: don't blanket-deny all private IPs

---

## 2. Decided Items

### 2.1 Agent Token — Decided

**Decision**: Not a priority fix in this cycle.

- Does NOT enter P0/P1 execution batch
- Retained as residual risk only
- Unless review confirms default token is actually used at runtime (it's not — bootstrap replaces it), no complex token strength/rotation/secret manager/mTLS design

### 2.2 NBR is the Source of Truth for Runtime Parameters — Decided

**Decision**: NodeBackendRuntime defines actual runtime parameters. Agent executes what NBR specifies.

- NBR controls which Docker parameters are used
- RunPlan and AgentRunSpec are generated from NBR
- Agent should NOT add complex vendor policy to block NBR-defined parameters

If MetaX official images require `/dev/mxcd`, `/dev/dri`, `--ipc=host`, `--security-opt`, `--group-add`, `--privileged`:
- NBR explicitly configures and executes them
- Responsibility lies with the admin who configured the NBR

**This cycle does NOT do**:
- Vendor-specific policy engine
- Privileged manual approval
- MetaX/Huawei/NVIDIA independent allowlist systems
- Agent complex secondary blocking of NBR parameters

**This cycle DOES**:
- RunPlan preview shows high-risk parameters
- Audit log records which high-risk parameters were used at startup
- Documentation提醒 these parameters are admin-configured
- Ensure only authorized people can modify NBR and start deployments

### 2.2a Finding 5.2 Priority — Decided

**Decision**: P1 (audit/traceability), NOT P0 (blocking).

- 5.2 is NOT "privileged passthrough" that must be blocked
- 5.2 IS "high-risk NBR runtime parameters require traceability and permission boundary"
- Batch: 1C
- Fix direction: audit log + RunPlan preview + NBR modification permission
- Explicitly: Do NOT block NBR-defined privileged/ipc/devices/security-opt params

### 2.2b Finding 5.4 Priority — Decided

**Decision**: P0, must-fix.

- 13 endpoints missing tenant scope is a real cross-tenant data access risk
- More impactful than SSRF in internal network environment
- Batch: 1A
- Fix: add `tenantScopeCheck()` to all 13 endpoints

### 2.3 Agent Endpoint Protection — Decided

**Decision**: Protect agent management endpoints, not restrict official runtime parameters.

| Endpoint | Policy |
|----------|--------|
| `/healthz` | Allow unauthenticated (for load balancers) |
| `/metrics` | Default Prometheus-compatible, auth configurable |
| `/docker-images` | Require agent token |
| `/docker-image-inspect` | Require agent token |
| `/files` | Require agent token |
| `/model-paths/scan` | Require agent token |
| `AllowRuntimeRootAdd` | Only effective on authenticated requests |

Batch 1C renamed to: **Agent Endpoint Protection / NBR Execution Boundary**

### 2.4 Stop/Remove Semantics — Decided

**Decision**: Confirmed, no longer open decision.

| Scenario | Action | Evidence |
|----------|--------|----------|
| Stop | Capture evidence/log tail → stop container → remove container → release resources | Logs sent to server |
| Failed start | Capture evidence/log tail → remove container | Logs captured before remove |
| Debug retain | Future dev/admin explicit config, NOT default behavior | — |
| Restart policy | Docker restart policy NOT used as platform recovery mechanism; platform uses controller/reconciler | — |

### 2.5 Batch 5 Paused — Decided

**Decision**: No standalone Gateway/API Key/Usage/Billing design batch.

Reasons:
- Current priority: runtime chain, security boundaries, lifecycle, RunPlan correctness
- Gateway/API Key/Usage/Billing are future capabilities
- Current fixes only need to not block future capabilities
- If a repair impacts these capabilities, note the impact and reservation in that batch's document

Batch 5 changed to: **Future Constraint Only: Unified Model API / Gateway / Audit / Usage / Billing**

Retained principles:
- Current priority: OpenAI-compatible API entry
- Future: must not hardcode OpenAI-only
- Don't hardcode deployment name as API model name
- Don't hardcode actor as session user
- Don't hardcode usage as token-only
- Don't expand billing/audit/usage into full design now

---

## 3. Revised Execution Batches

| Batch | Name | Strategy |
|-------|------|----------|
| 1A | Access Control / Tenant Ownership / RBAC Boundary | Execute |
| 1B | Server → Agent Client / SSRF / Address Policy | Execute |
| 1C | Agent Endpoint Protection / NBR Execution Boundary | Execute, lightweight |
| 2 | Docker Lifecycle / Cleanup / Concurrency | Execute |
| 3 | Input / Output / Audit / Log Safety | Execute |
| 4 | RunPlan / Runtime Config / Catalog Correctness | Execute, focus on current bugs |
| 5 | Unified Model API / Gateway / Audit / Usage / Billing | Paused — future constraint only |
| 6 | Web / API Contract / i18n / Permission UX | Execute |
| 7 | Test Infrastructure | Add tests per batch, finalize at end |

### Batch 1A — Execute
- Tenant scope for 13 endpoints
- Rate limiter IP fix
- CSRF rotation fix
- Observability endpoint auth

### Batch 1B — Execute
- AgentClient with timeout/URL encoding/response limit
- Address policy (deny metadata/link-local/unspecified/multicast; private IPs allowed in dev/LAN)
- Replace all bare http.Get/Post calls

### Batch 1C — Execute (Lightweight)
- Agent endpoint auth (require token for management endpoints)
- `AllowRuntimeRootAdd` requires auth
- No vendor policy engine
- No privileged approval
- High-risk parameter logging in audit

### Batch 2 — Execute
- Add ContainerRemove to DockerClient interface
- Cleanup on failed start
- Remove on stop
- Race condition fixes (logsTaskState, reconcileState)
- Task dedup

### Batch 3 — Execute
- Body size limit middleware
- Audit log json.Marshal
- Docker stream payload limit
- Task result truncation
- Redaction rewrite

### Batch 4 — Execute (Focus on Current Bugs)
- Boolean flag dedup fix
- Env override substitution fix
- Required param validation
- Input hash fix
- Dead code removal
- Catalog YAML cleanup
- NOT: full three-layer RunPlan, NOT scheduler, NOT APICompatibilityProfile

### Batch 5 — Future Constraint Only
- No standalone design batch
- Document future constraints in existing batch docs where relevant
- Retain: OpenAI-compatible first, protocol-agnostic future
- Retain: actor model reservation, usage model reservation

### Batch 6 — Execute
- Route guard
- Credential removal
- i18n fixes
- Permission wiring
- Confirmation dialogs

### Batch 7 — Test Infrastructure (Per-Batch + Final)
- Auth unit tests (with Batch 1A)
- Tenant isolation tests (with Batch 1A)
- AgentClient SSRF tests (with Batch 1B)
- Agent endpoint auth tests (with Batch 1C)
- Docker lifecycle mock tests (with Batch 2)
- RunPlan resolver tests (with Batch 4)
- Race condition tests (with Batch 2)
- Mock E2E framework (final)
- Frontend component tests (with Batch 6)

---

## 4. Concurrency Strategy

### Can Run in Parallel

| Work | Reason |
|------|--------|
| New packages (`authz/`, `agentclient/`) | Independent files, no shared state |
| Test files | Independent, no production code conflicts |
| Frontend local fixes | Different file tree |
| RunPlan resolver tests | Independent test files |
| DockerClient fake tests | Independent test files |

### Must Serialize (Shared Files)

| Work | Reason |
|------|--------|
| Server API handlers | Multiple batches touch same files |
| Agent main.go | Multiple fixes in same file |
| Docker runtime driver | Interface change + lifecycle fixes |
| Runplan resolver.go | Multiple bug fixes in same file |
| Router | Auth middleware changes |
| Shared config | Config structure changes |

### Recommended Parallel Execution

```
Batch 1A (authz package) ──────┐
Batch 1B (agentclient package) ┼──→ Integration: server handlers
Batch 1C (agent endpoint auth) ┘

Batch 2 (DockerClient interface) ──→ Integration: docker.go, main.go

Batch 3 (middleware + local fixes) ──→ Independent commits

Batch 4 (resolver tests first) ──→ Integration: resolver.go

Batch 6 (frontend) ──→ Independent, can run anytime

Batch 7 (tests) ──→ Per-batch, no production code conflicts
```

---

## 5. Automation Principles

- Execute per document decisions by default
- No manual confirmation prompts
- No manual approval workflows
- Edge cases recorded in closeout documents
- If problem is confirmed and fixable: fix it
- Don't retain fallbacks for old configs/old versions

---

## 6. Summary of Changes from V2 Original

| Aspect | V2 Original | V2 Converged |
|--------|-------------|--------------|
| Batch 1C | Agent Security Policy (vendor policy engine) | Agent Endpoint Protection (lightweight) |
| Batch 5 | 3-5 days full design | Paused, future constraint only |
| Batch 7 | 3-5 days one-shot | Per-batch tests + final |
| Agent token | P0 fix | Residual risk, not fixed |
| Vendor policy | Complex allowlist system | NBR is source of truth |
| Gateway/Billing | Full design batch | Future constraint only |
| Total effort | 21-32 days | ~14-20 days |
| Priority | Everything P0 | Real threat model based |
