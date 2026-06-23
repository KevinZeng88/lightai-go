# Open Decisions and Risks — Repair Plan V2 (Converged)

> Date: 2026-06-23
> Purpose: Decided items, remaining open decisions, and risks

---

## 1. Decided Items (No Longer Open)

### 1.1 Agent Token — DECIDED

**Decision**: Not a priority fix in this cycle.

- Does NOT enter P0/P1 execution batch
- Retained as residual risk only
- Bootstrap mechanism replaces default token at runtime
- Internal network environment reduces risk

### 1.2 NBR is Source of Truth for Runtime Parameters — DECIDED

**Decision**: NodeBackendRuntime defines actual runtime parameters. Agent executes what NBR specifies.

- No vendor-specific policy engine
- No privileged manual approval
- No MetaX/Huawei/NVIDIA independent allowlist systems
- Agent does NOT block NBR-defined parameters
- High-risk params logged in audit but not blocked
- Responsibility lies with admin who configured NBR

### 1.3 Agent Endpoint Protection — DECIDED

**Decision**: Protect agent management endpoints with auth.

| Endpoint | Policy |
|----------|--------|
| `/healthz` | Unauthenticated |
| `/metrics` | Unauthenticated (Prometheus compatible) |
| `/docker-images` | Require agent token |
| `/docker-image-inspect` | Require agent token |
| `/files` | Require agent token |
| `/model-paths/scan` | Require agent token |
| `AllowRuntimeRootAdd` | Authenticated requests only |

### 1.4 Stop/Remove Semantics — DECIDED

**Decision**: Confirmed.

| Scenario | Action |
|----------|--------|
| Stop | Capture logs → stop → remove → release |
| Failed start | Capture logs → remove |
| Debug retain | Future explicit config, not default |
| Restart policy | NOT platform recovery mechanism |

### 1.5 Finding 5.2 — High-Risk NBR Parameters — DECIDED

**Decision**: P1, audit/traceability only, NOT agent-side blocking.

- NBR is source of truth for runtime parameters (privileged, ipc, devices, security-opt, group-add)
- Agent does NOT block NBR-defined parameters
- High-risk params logged in audit detail at deployment start
- RunPlan preview shows high-risk parameters
- Security boundary: tenant/RBAC/NBR modification permissions
- Batch: 1C

### 1.6 Finding 5.4 — Tenant Scope — DECIDED

**Decision**: P0, must-fix before repair closeout.

- 13 endpoints missing tenant scope checks
- Real cross-tenant data access risk in multi-tenant deployments
- Affects: node files, model scan, Docker image inspect, NBR CRUD, model root/location
- Fix: add `tenantScopeCheck()` to all 13 endpoints
- Batch: 1A

### 1.7 Batch 5 Paused — DECIDED

**Decision**: No standalone Gateway/API Key/Usage/Billing design batch.

- Future constraint only
- Current priority: runtime chain, security, lifecycle, RunPlan
- Don't hardcode OpenAI-only, session-user-only, token-only

### 1.8 Current Golden Path Non-Regression — DECIDED

**Decision**: All repairs must preserve currently working flows.

- See `07-golden-path-non-regression.md`
- Each batch has explicit non-regression checks
- Before/After baseline comparison required
- If repair breaks golden path, repair is wrong

---

## 2. Remaining Open Decisions

### 2.1 Agent Client Address Policy Default

**Question**: What should the default address policy be?

**Current recommendation**: Simple deny list (metadata, link-local, unspecified, multicast). Allow everything else including private IPs.

**Still open**: Should there be a config option for strict mode (only registered agents)?

**Recommendation**: Keep simple for now. Add strict mode later if needed.

### 2.2 `/metrics` Authentication

**Question**: Should `/metrics` require auth by default?

**Current decision**: No auth (Prometheus compatible).

**Still open**: Should there be a config option to enable auth?

**Recommendation**: Add config option `agent.metrics_auth=false` default. Document for operators.

### 2.3 Failed Container Evidence Storage

**Question**: Where is evidence stored — server DB or agent local?

**Current approach**: Agent captures logs → sends to server via task result → server stores in DB → agent removes container.

**Still open**: What if agent crashes before sending evidence?

**Recommendation**: Agent writes evidence to local file first, then sends to server. If agent crashes, evidence is in local file. Add `evidence_dir` config.

### 2.4 RunPlan Hash Granularity

**Question**: How fine-grained should the input hash be?

**Current approach**: Include AssignedGPUs, NodeRuntimeOverride, ProcessStartConfig.

**Still open**: Should hash include every env var, every arg? Or just deployment-level inputs?

**Recommendation**: Hash deployment-level inputs only. Instance-level resolution is deterministic given same deployment + node + NBR.

### 2.5 Batch 4 Schema Changes

**Question**: Does RunPlan fix require schema changes?

**Current assessment**: No. RunPlan is computed at runtime, not stored in DB.

**Still open**: If catalog YAML structure changes, does it require DB migration?

**Recommendation**: Catalog YAML is file-based, not DB-based. No migration needed. Just update YAML files.

### 2.6 Collector Command Validation

**Question**: How strict should collector command validation be?

**Current approach**: Validate command path exists and is executable.

**Still open**: Should we allowlist specific commands?

**Recommendation**: For now, validate path only. Add allowlist later if needed.

---

## 3. Risks

### 3.1 Tenant Check False Positives

**Risk**: New tenant scope checks block legitimate same-tenant access.

**Mitigation**:
- Use existing `tenantScopeCheck()` helper pattern
- Test with default tenant and admin user
- Golden path includes same-tenant access verification

### 3.2 AgentClient Timeout Too Short

**Risk**: 30s timeout too short for large model directory scans.

**Mitigation**:
- Make timeout configurable per endpoint type
- Default 30s for most, 120s for file/model scan
- Golden path includes large directory scan test

### 3.3 ContainerRemove Breaks Debug Workflow

**Risk**: Removing containers on stop/failure makes debugging harder.

**Mitigation**:
- Evidence captured BEFORE remove
- Logs sent to server before remove
- Container inspect state captured
- Future: debug retain config option

### 3.4 RunPlan Fix Breaks Existing Parameters

**Risk**: Boolean flag fix, env substitution fix, required param fix change existing behavior.

**Mitigation**:
- All existing RunPlan tests must pass
- Golden path includes vLLM/SGLang/llama.cpp RunPlan verification
- Fix is backward-compatible (only adds missing behavior)

### 3.5 Frontend Route Guard Login Loop

**Risk**: Route guard redirects to login, login redirects to guard, infinite loop.

**Mitigation**:
- Guard checks `to.path !== '/login'` before redirect
- Guard checks auth state before redirect
- Golden path includes login flow verification

### 3.6 Agent Auth Breaks Prometheus Scrape

**Risk**: Adding auth to agent endpoints breaks Prometheus.

**Mitigation**:
- `/metrics` stays unauthenticated by default
- `/healthz` stays unauthenticated
- Only management endpoints require auth
- Golden path includes Prometheus scrape verification

---

## 4. Items Explicitly NOT in Scope

| Item | Why Not |
|------|---------|
| Agent-side blocking of NBR params | NBR is source of truth; admin responsibility |
| Vendor-specific policy engine | NBR is source of truth |
| Privileged manual approval | Admin responsibility via NBR |
| Gateway implementation | Future capability |
| API Key implementation | Future capability |
| Usage metering implementation | Future capability |
| Billing implementation | Future capability |
| Full scheduler | Future capability |
| Three-layer RunPlan | Future capability |
| Multi-replica support | Future capability |
| Multi-node scheduling | Future capability |
| mTLS for agent communication | Future capability |
| Token strength validation | Residual risk, not priority |
| Secret manager integration | Future capability |

---

## 5. Decision Record Summary

| Decision | Status | Date |
|----------|--------|------|
| Agent token not priority | DECIDED | 2026-06-23 |
| NBR is runtime param source | DECIDED | 2026-06-23 |
| Agent endpoint auth | DECIDED | 2026-06-23 |
| Stop/remove semantics | DECIDED | 2026-06-23 |
| 5.2: audit/traceability, not blocking | DECIDED | 2026-06-23 |
| 5.4: P0 must-fix tenant scope | DECIDED | 2026-06-23 |
| Batch 5 paused | DECIDED | 2026-06-23 |
| Golden path non-regression | DECIDED | 2026-06-23 |
| Address policy default | OPEN | — |
| `/metrics` auth config | OPEN | — |
| Evidence storage location | OPEN | — |
| Hash granularity | OPEN | — |
| Collector command allowlist | OPEN | — |
