# Triage Corrections — Repair Plan V2 (Final)

> Date: 2026-06-23
> Purpose: Final corrected statistics and finding status after convergence review

---

## 1. Final Finding Statistics

### Counting Rules

1. **Confirmed**: Finding verified by source code examination, real issue exists
2. **False Positive**: Finding claim is incorrect (e.g., 5.3 — runtime enforcement exists)
3. **Partial False Positive**: Part of finding is false (e.g., 8.2 ObservabilityOverviewPage)
4. **Priority Adjusted**: Finding real but severity changed based on threat model
5. **Downgraded**: Finding real but not a priority for current product stage

### Final Counts

| Category | Count | Notes |
|----------|-------|-------|
| **Confirmed (P0 — must fix)** | 5 | SSRF, tenant scope, container leak, race panic, auth tests gap |
| **Confirmed (P1 — should fix)** | 16 | Agent auth, high-risk NBR traceability, body limit, RunPlan bugs, route guard |
| **Confirmed (P2 — fix for quality)** | 17 | Catalog cleanup, CSRF, constant-time, stale cache, etc. |
| **Total Confirmed** | 38 | |
| **False Positive** | 1 | 5.3 (agent token — bootstrap exists) |
| **Partial False Positive** | 1 | 8.2 ObservabilityOverviewPage (no credentials) |
| **Priority Adjusted** | 4 | 5.2 P0→P1, 5.4 P1→P0, 6.4 P1→P2, 8.1 P0→P1 |
| **Downgraded to Record** | 1 | 5.3 residual risk only |
| **Not Trialed (P3/doc)** | 42 | Documentation, stale config, minor issues |
| **Total Findings** | 87 | |

### What Changed from V1

| Finding | V1 Status | Final Status | Reason |
|---------|-----------|--------------|--------|
| 5.2 | CONFIRMED P0 | CONFIRMED P1 | NBR is source of truth; audit/traceability, not agent-side blocking |
| 5.3 | CONFIRMED P0 | FALSE POSITIVE → Residual Risk | Bootstrap mechanism replaces default token at runtime |
| 5.4 | CONFIRMED P1 | CONFIRMED P0 | Real cross-tenant access risk on 13 critical endpoints |
| 8.2 (GrafanaPage) | CONFIRMED P0 | CONFIRMED P1 | Operational security/i18n issue, not same tier as SSRF |
| 8.2 (ObservabilityOverviewPage) | CONFIRMED P0 | FALSE POSITIVE (partial) | File has no credentials |
| 6.4 | CONFIRMED P1 | CONFIRMED P2 | Data race on int field, no panic risk |
| 8.1 | CONFIRMED P0 | CONFIRMED P1 | i18n quality issue, not security |

### P0 Findings (Must Fix)

| ID | Issue | Fix Batch |
|----|-------|-----------|
| 5.1 | SSRF via agent proxy endpoints (no timeout, no URL encoding, no address validation) | 1B |
| 5.4 | 13 endpoints missing tenant scope checks (cross-tenant data access) | 1A |
| 6.1 | Container not cleaned up after start failure (resource leak, name collision) | 2 |
| 6.2 | Race condition on logsTaskState map (concurrent map panic) | 2 |
| 9.1 | No unit tests for auth/RBAC core logic (security regressions undetected) | 7 |

### P1 Findings (Should Fix)

| ID | Issue | Fix Batch |
|----|-------|-----------|
| 4.1 | Boolean flag dedup | 4 |
| 4.2 | Env overrides bypass substitution | 4 |
| 4.3 | Required params silently skipped | 4 |
| 5.2 | High-risk NBR runtime parameters require traceability and permission boundary | 1C |
| 5.5 | Agent metrics endpoint no auth | 1C |
| 5.6 | AllowRuntimeRootAdd bypass | 1C |
| 5.7 | Rate limiter trusts spoofable XFF | 1A |
| 5.8 | No request body size limit | 3 |
| 5.9 | JSON injection in audit log | 3 |
| 5.10 | Credentials file predictable path | 1A |
| 6.3 | No container removal on stop | 2 |
| 7.1 | Collector sh -c injection | 1C |
| 8.1 | Hardcoded Chinese | 6 |
| 8.2 | Default credentials in UI | 6 |
| 8.3 | No route guard | 6 |
| 8.4 | RolesPage permission reset | 6 |

### P2 Findings (Quality Improvement)

| ID | Issue | Fix Batch |
|----|-------|-----------|
| 4.4 | buildDeviceBinding dead code | 4 |
| 4.5 | computeInputHash missing fields | 4 |
| 4.6-4.8 | Catalog cleanup | 4 |
| 5.11 | Observability no auth | 1A |
| 5.12 | CSRF rotation on /me | 1A |
| 5.13 | Non-constant-time token compare | 1A |
| 6.4 | Race on reconcileState | 2 |
| 6.5 | Task result swallows DB errors | 3 |
| 6.6-6.7 | Blocking stop/logs handlers | Future |
| 7.2-7.3 | Docker stream/task size limits | 3 |
| 7.4 | No task dedup | 2 |
| 8.5-8.8 | UI hardcoded ports, stale cache, unused permissions | 6 |
| 9.6 | No race condition tests | 7 |
| 9.7 | Missing tenant isolation edge cases | 7 |
| 10.3 | redactDetailString incorrect | 3 |

### Downgraded / Residual

| ID | Issue | Status | Notes |
|----|-------|--------|-------|
| 5.3 | Agent token weak default | RESIDUAL | Bootstrap exists; internal network; not priority |
| 5.2 | High-risk NBR params | P1 | Traceability + permission boundary, NOT agent-side blocking |

---

## 2. Finding 5.2 — Final Status and Explanation

### Original Claim
Agent blindly trusts server-sent `Privileged` flag. No agent-side policy. P0.

### Corrected Understanding
- NBR (NodeBackendRuntime) is the **source of truth** for runtime parameters
- NBR defines which Docker parameters are used (privileged, ipc, devices, security-opt, group-add)
- Agent executes what NBR specifies — this is the correct design for admin-managed infrastructure
- Admin who configures NBR is responsible for parameter safety
- Platform ensures only authorized users can modify NBR (via tenant/RBAC)

### Final Priority: P1

**Fix direction**: Audit/traceability + NBR modification permission boundary
- Log high-risk parameters in audit detail when deployment starts
- RunPlan preview shows high-risk parameters
- Ensure only authorized users can modify NBR and start deployments
- Do NOT block NBR-defined privileged/ipc/devices/security-opt parameters

**Batch**: 1C (Agent Endpoint Protection / NBR Execution Boundary)

---

## 3. Finding 5.4 — Final Status and Explanation

### Original Claim
8-13 endpoints missing tenant scope checks. P1.

### Why Upgraded to P0
- Cross-tenant data access is a **real exploitation risk** in multi-tenant/multi-business-line deployments
- Affects critical resources: node files, model scan, Docker image inspect, NBR CRUD, model root/location
- More impactful than SSRF in internal network environment
- `tenantScopeCheck()` helper exists but was not applied to these endpoints
- This is a systematic omission, not a one-off bug

### Final Priority: P0

**Fix direction**: Add `tenantScopeCheck()` to all 13 endpoints
**Batch**: 1A (Access Control / Tenant Ownership)

---

## 4. Security Re-classification

### Tier 1 — Must Fix (P0)

- Cross-tenant data access (5.4) — real multi-tenant risk
- Server→agent SSRF (5.1) — instability + potential SSRF
- Container lifecycle leaks (6.1) — resource leak on every failure
- Race condition panics (6.2) — crashes under concurrent load
- Auth test gaps (9.1) — security regressions undetected

### Tier 2 — Should Fix (P1)

- High-risk NBR traceability (5.2) — audit, not blocking
- Agent endpoint exposure (5.5, 5.6) — management endpoints need auth
- Body size DoS (5.8) — internal network but still a risk
- Audit log injection (5.9) — malformed JSON
- RunPlan parameter bugs (4.1, 4.2, 4.3) — startup failures
- Frontend auth gaps (8.3, 8.4) — route guard, permission reset
- Credentials exposure (8.2) — Grafana default creds
- i18n quality (8.1) — hardcoded Chinese

### Tier 3 — Record / Low Priority

- Agent token strength (5.3) — bootstrap exists
- CSRF rotation (5.12) — intermittent only
- Constant-time compare (5.13) — internal network

---

## 5. Threat Model Context

LightAI Go operates in **internal network, controlled nodes, trusted admin** environment.

| Threat | Likelihood | Impact | Priority |
|--------|-----------|--------|----------|
| Cross-tenant access | HIGH (multi-business-line) | HIGH | **P0** |
| SSRF via agent proxy | LOW (internal) | HIGH | **P0** (fix — instability) |
| Container leak | HIGH (normal op) | MEDIUM | **P0** |
| Race condition panic | MEDIUM (concurrent) | HIGH | **P0** |
| Auth test gaps | N/A (testing) | HIGH | **P0** |
| High-risk NBR params | LOW (admin-controlled) | MEDIUM | P1 (traceability) |
| Agent endpoint exposure | LOW (internal) | MEDIUM | P1 |
| Body size DoS | LOW (internal) | MEDIUM | P1 |
| Agent token weak | LOW (bootstrap) | LOW | Residual |
