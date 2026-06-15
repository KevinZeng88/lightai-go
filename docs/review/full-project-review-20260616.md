# LightAI Go Full Project Review — 2026-06-16

## 1. Executive Summary

**Overall Status:** Phase 2F core deliverables (tenant type, audit API, tenant switching, Web management pages) are implemented. However, the review reveals **4 P0 gaps** that must be fixed before declaring Phase 2F closed:

1. **Node transfer does not check for active gpu_lease** — transfer proceeds even with active leases (P0)
2. **Node transfer does not check for active deployment_instance** — transfer proceeds even with running instances (P0)
3. **`audit:read` permission not assigned to any built-in role** — no user can view audit logs (P0)
4. **`model_instances` tenant isolation not implemented in handlers** — cross-tenant instance enumeration (P0)

**Recommended:** Do NOT close Phase 2F yet. Fix the 4 P0 items first. The remaining findings (Web i18n, API URL consistency) are P1 and can be addressed in a follow-up hardening phase.

**Version:** 0.1.14 — release build passes (436MB, glibc ABI OK) but untested on clean install.

**Max Risks:**
1. Transfer safety: active GPU leases and running instances are not blocked on node transfer
2. Tenant isolation: model_instances list/get handlers lack tenant scoping
3. Audit log access: `audit:read` permission exists but is unreachable by any built-in role
4. Web i18n: 9+ entire i18n key sections missing — pages show raw key strings instead of translated text

## 2. Verification Commands Executed

```
go test ./...             ✅ ALL PASS
go build ./cmd/server     ✅
go build ./cmd/agent      ✅
cd web && npm run build   ✅ 3.19s
git diff --check          ✅ CLEAN
git status --short        ✅ CLEAN
```

## 3. Phase 2F Requirement Matrix

| Step | Requirement | Implementation | Tests | Docs | Status | Risk |
|------|-------------|----------------|-------|------|--------|------|
| 0 | Baseline | git clean, build pass | - | - | Done | - |
| 1 | Plan + audit | docs/plan/phase-2f-*.md | - | plan doc | Done | - |
| 2 | Schema V7 | db.go migrateV7: tenant.type, resource_pools* | - | design doc | Done | - |
| 3A | Audit API | audit_handlers.go, router.go | - | design doc | Done | - |
| 3B | Audit Web | AuditLogsPage.vue, auditLogs.ts | - | - | Done | i18n keys missing |
| 4 | Tenant Web | TenantsPage.vue, tenants.ts | - | - | Done | i18n keys missing |
| 5 | User Web | UsersPage.vue, users.ts | - | - | Done | create stub empty, i18n missing |
| 6 | Role Web | RolesPage.vue, roles.ts | - | - | Done | i18n keys missing |
| 7 | Tenant switch | auth/handlers.go + ConsoleLayout.vue | - | design doc | Done | - |
| 8 | Transfer hardening | agent_handlers.go HandlePatchNodeTenant | rbac_phase2f_test.go | - | **Partial** | **P0: no lease/instance check** |
| 9 | Instance isolation | model_handlers.go | - | - | **GAP** | **P0: handler lacks tenant filter** |
| 10 | RBAC tests | rbac_phase2f_test.go (18 tests) | - | - | Done | Transfer tests mostly DB-level |
| 11 | Docs | design + ops docs | - | design + ops | Done | - |
| 12 | Regression | build/test pass, E2E verified | - | - | Done | E2E passes (23:51 run) |

## 4. Previously Raised Issues Matrix

| Issue | Status | Evidence |
|-------|--------|----------|
| Web table column width/layout | Partially resolved | NodesPage/GpusPage have resizable columns; other pages lack column width handling |
| GPU name display (# symbol) | Not resolved | GpusPage.vue line 40 shows raw `name` prop with no sanitization |
| Node info completeness | Partially resolved | hostname/ip/version shown but os/arch/kernel info only in drawer |
| Agent auto GPU discovery | Resolved | collector_mode: auto probes NVIDIA + MetaX |
| MetaX metrics alignment | Unknown | No real MetaX hardware to verify |
| 0.1.14 clean install verification | Not done | Package builds but never installed fresh and verified |
| Grafana/Prometheus i18n | Not resolved | Hardcoded English in PrometheusPage.vue, GrafanaPage.vue |
| /v1/models test knowledge in docs | Not resolved | E2E doc mentions /v1/models but no dedicated test doc |

## 5. Test Coverage Matrix

| Feature | Explicit Test | Positive | Negative | Risk |
|---------|--------------|----------|----------|------|
| Audit log tenant isolation | TestAuditLogTenantIsolation | ✅ | ✅ | - |
| Audit:read permission gate | TestAuditLogRequiresAuditReadPermission | ✅ | - | Permission not assigned to any role |
| Instance tenant isolation | TestModelInstanceTenantIsolation | ✅ | - | Handler lacks tenant filter |
| Switch tenant membership | TestSwitchTenantMembershipValidation | ✅ | ✅ | - |
| Switch to inactive tenant | TestSwitchTenantToInactiveTenantFails | - | ✅ | - |
| Non-admin cannot create tenant | TestNonAdminCannotCreateTenant | - | ✅ | - |
| Built-in role protected | TestBuiltInRoleCannotBeDeleted | ✅ | - | - |
| Transfer to nonexistent tenant | TestNodeTransferToNonExistentTenantFails | - | ✅ | DB-only, not handler test |
| Transfer to inactive tenant | TestNodeTransferToInactiveTenantFails | - | ✅ | Handler-level ✅ |
| Transfer permission required | TestNodeTransferRequiresPermission | - | ✅ | DB-only, not handler test |
| Transfer writes audit | TestNodeTransferWritesAuditLog | ✅ | - | DB-only, not handler test |
| Active lease blocks transfer | TestGpuLeaseActiveBlocksTransfer | - | ✅ | **DB-only, handler lacks check** |
| Active deployment blocks transfer | **NO TEST** | - | - | **P0: No test, no handler check** |
| Tenant user can transfer own node | TestTenantUserWithTransferPermissionCanTransferNode | ✅ | - | Handler-level ✅ |
| GPU transfer to inactive tenant | **NO TEST** | - | - | No independent GPU transfer API |

## 6. Documentation Consistency Review

| Issue | Doc | Detail |
|-------|-----|--------|
| Continuation-steps Step 8 "done" but safety checks missing | phase-2f-continuation-steps.md:20 | Active lease/instance checks not implemented |
| Design doc says "should block transfer" for leases/instances | tenant-rbac-resource-ownership-design.md:97 | Code does not implement this |
| Ops guide says "Check for active deployments/leases before transferring" | tenant-rbac-resource-ownership-operations.md:46 | Code does not do this |
| Default tenant type should be 'infrastructure' per design | design doc:24 | Bootstrap creates it as 'business' (column default) |

## 7. Security / RBAC Review

| Issue | Severity | Detail |
|-------|----------|--------|
| model_instances handler lacks tenant filter | **P0** | Any user with instance:read can enumerate all tenants' instances |
| HandlePatchNodeTenant lacks active gpu_lease check | **P0** | Transfer breaks active GPU leases |
| HandlePatchNodeTenant lacks active deployment_instance check | **P0** | Transfer breaks running model instances |
| audit:read not assigned to any built-in role | **P0** | No user can access audit logs |
| API URL prefix inconsistency | P1 | 10+ API modules use wrong URL prefix in dev mode |
| i18n keys missing for 9+ entire sections | P1 | Pages show raw key strings instead of translations |
| Personal paths in production code | P1 | /home/kzeng/models hardcoded in ModelDeploymentsPage.vue |
| Prometheus/Grafana pages lack i18n | P1 | Hardcoded English throughout |
| Empty create user stub | P2 | UsersPage "Create" button does nothing |

## 8. Web/UI Review

| Area | Finding |
|------|---------|
| Pages | All 20 page files exist and compile |
| Router | All 18 routes registered |
| Navigation | 6 menu groups, all items present |
| i18n | **9 entire top-level key sections missing** (audit, tenants, users, roles, modelArtifacts, modelDeployments, modelInstances, runTemplates, runtimeEnvs) |
| API clients | 15 files, but URL prefix inconsistencies across 10+ modules |
| Tables | GpusPage/NodesPage have resizable columns; other pages lack width handling |
| Tenant switcher | Implemented with full page reload |

## 9. Agent/GPU/Monitoring Review

| Area | Status |
|------|--------|
| NVIDIA collector | Working (verified in E2E) |
| MetaX collector | Scripts ready, real hardware not verified |
| GPU auto-detection | Working (probes NVIDIA + MetaX) |
| Prometheus metrics | agent_lightai_gpu_* exported |
| Grafana dashboards | 3 dashboards auto-provisioned |
| Prometheus/Grafana i18n | Hardcoded English |

## 10. Release Readiness

- Package builds (436MB, glibc ABI OK)
- Tarball is clean (no temp files)
- **NOT ready for release** — 4 P0 items must be fixed first

## 11. Action List

### P0 — Must Fix Now

| # | Issue | Impact | Fix |
|---|-------|--------|-----|
| 1 | HandlePatchNodeTenant: no active gpu_lease check | Transfer while GPU leased corrupts state | Add query for active leases before transfer |
| 2 | HandlePatchNodeTenant: no active deployment_instance check | Transfer while instance running corrupts state | Add query for running instances before transfer |
| 3 | audit:read not assigned to any built-in role | No user can view audit logs | Add audit:read to admin role in bootstrap |
| 4 | model_instances handler lacks tenant_id filter | Cross-tenant instance enumeration | Add tenant filter to HandleListModelInstances, HandleGetModelInstance |

### P1 — Fix Before Release

| # | Issue | Impact | Fix |
|---|-------|--------|-----|
| 5 | 9+ entire i18n key sections missing | Pages show raw key strings | Add all missing i18n keys to both locales |
| 6 | API URL prefix inconsistency | API calls break in dev mode | Unify URL prefix handling across all API modules |
| 7 | Personal paths in production code | Code not portable | Remove hardcoded /home/kzeng paths |
| 8 | Prometheus/Grafana hardcoded English | Poor UX for Chinese users | Add i18n keys, use t() |

### P2 — Follow-up

| # | Issue | Impact | Fix |
|---|-------|--------|-----|
| 9 | Empty UsersPage create stub | Create button does nothing | Implement user creation form |
| 10 | Format utilities not localized | English suffixes in Chinese UI | Add locale-aware format functions |
| 11 | Download gpuLeases.ts | Unused API client | Either wire up or remove |
| 12 | GPU vendor filter hardcoded | New vendors not shown | Make vendor list dynamic |

---

## Phase 2F Final Closure

### Final Status

Phase 2F is **closed** after all 12 review findings were fixed and validated.

### Review Fix Commits

| Commit | Description |
|--------|-------------|
| `86ab1d4` | phase-2f: fix all 12 review issues |
| `3109999` | phase-2f: complete review fix validation coverage |
| `1c26622` | phase-2f: localize formatRelativeTime with zh-CN/en-US i18n |
| `df1a212` | test: make model runtime E2E cleanup safe with PID-based trap |

### E2E Exit 144 Root Cause and Fix

The original `scripts/e2e-model-runtime-local.sh` used:

```bash
pkill -f lightai-server 2>/dev/null || true
pkill -f lightai-agent 2>/dev/null || true
```

In the sandbox environment, `pkill -f` matched the parent shell process, causing the script to receive SIGUSR1 (exit code 144).

**Fix:** Replaced with PID-based trap cleanup:

```bash
trap cleanup EXIT INT TERM

cleanup() {
  set +e
  if [ -n "$AGENT_PID" ] && kill -0 "$AGENT_PID" 2>/dev/null; then
    kill "$AGENT_PID" 2>/dev/null
    sleep 1
    kill -0 "$AGENT_PID" 2>/dev/null && kill -9 "$AGENT_PID" 2>/dev/null
  fi
  if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null
    sleep 1
    kill -0 "$SERVER_PID" 2>/dev/null && kill -9 "$SERVER_PID" 2>/dev/null
  fi
  if [ -n "$CONTAINER_ID" ]; then
    docker rm -f "$CONTAINER_ID" 2>/dev/null || true
  fi
  ...
}
```

**E2E cleanup rules:**
- Only kill PIDs that this script started (SERVER_PID, AGENT_PID)
- Docker cleanup by explicit container_id, not wildcard name/pattern
- No `pkill`, `pkill -f`, or `killall` anywhere in the script
- `trap EXIT INT TERM` ensures cleanup runs on any exit
- Cleanup is idempotent — failed cleanup does not mask test results (uses `set +e`)

### Final E2E Result (2026-06-16 00:40)

```
deployment_id:  5665def4-09bc-4729-a98c-dadd95a40ff1
instance_id:    133466ae-5f0e-4214-86ce-14b8a981a1b4
start_task_id:  9fe683a1-32b2-413c-8445-7be43fe64469
logs_task_id:   96eced2e-db89-4e74-970e-0cf9b8cac821
stop_task_id:   2ed8b3ad-1e75-4ccf-9191-33bfe9486268
container_id:   2c993f58f7839b6ad19621958f476c0b969c2289a585af4f3827573847be810a
container_name: lightai-133466ae-5f0

Dry-run:         --gpus "device=0" ✅
/v1/models:      Qwen3.5-9B-Q4_K_M.gguf (8.95B params, 262K ctx) ✅
Logs:            3142 bytes (CUDA device info) ✅
Instance:        running → stopped ✅
Lease:           active → released ✅
Stop idempotent: ✅
Cleanup:         trap handler (no pkill) ✅
Exit code:       0 ✅
```

### Final Verification

```
go test ./...                              ✅ ALL PASS
go build ./cmd/server                      ✅
go build ./cmd/agent                       ✅
cd web && npm run build                    ✅
node web/tests/i18nKeys.test.mjs           ✅ PASS (220/220)
node web/tests/apiClientPaths.test.mjs     ✅ PASS (12/12)
node web/tests/formatters.test.mjs         ✅ PASS (8/8)
git diff --check                           ✅ CLEAN
git status --short                         ✅ CLEAN
scripts/e2e-model-runtime-local.sh         ✅ PASS
scripts/package-release-docker.sh --no-bump ✅ PASS (436MB, glibc ABI OK)
```

### Closed Review Issues (All 12)

| # | Issue | Fix | Test | Status |
|---|-------|-----|------|--------|
| 1 | Transfer no gpu_lease check | Active lease query in HandlePatchNodeTenant | TestNodeTransferBlockedByActiveGpuLease | ✅ |
| 2 | Transfer no instance check | Running instance query in HandlePatchNodeTenant | TestNodeTransferBlockedByActiveDeploymentInstance | ✅ |
| 3 | audit:read not assigned | Added to BuiltinRoles admin (tenant admin) | TestBuiltInAdminHasAuditReadPermission | ✅ |
| 4 | Instance no tenant filter | List: tenant WHERE; Get: tenantScopeCheck | TestModelInstanceGetOtherTenantDeniedForNonAdmin | ✅ |
| 5 | i18n sections missing | 220/220 keys zh=EN; i18nKeys.test.mjs | PASS | ✅ |
| 6 | API URL prefix inconsistent | apiClient auto-prepends /api/v1; 12 modules verified | apiClientPaths.test.mjs PASS | ✅ |
| 7 | Personal paths hardcoded | Replaced with empty defaults + example comments | grep clean | ✅ |
| 8 | Prometheus/Grafana English | i18n keys + t() calls | web build | ✅ |
| 9 | UsersPage empty stub | Create dialog: username/display_name/password | web build | ✅ |
| 10 | Format utilities | formatRelativeTime locale-aware (zh-CN/en-US) | formatters.test.mjs 8/8 PASS | ✅ |
| 11 | Orphan gpuLeases.ts | Deleted | web build | ✅ |
| 12 | GPU vendor hardcoded | Dynamic vendorOptions from data | web build | ✅ |
