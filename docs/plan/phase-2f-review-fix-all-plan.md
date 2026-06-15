# Phase 2F Review Fix-All Plan

Fix all 12 issues from docs/review/full-project-review-20260616.md.

| # | Issue | Priority | Status |
|---|-------|----------|--------|
| 1 | Node transfer no active gpu_lease check | P0 | pending |
| 2 | Node transfer no active deployment_instance check | P0 | pending |
| 3 | audit:read not assigned to any built-in role | P0 | pending |
| 4 | model_instances handlers lack tenant_id scope | P0 | pending |
| 5 | 9 entire i18n key sections missing | P1 | pending |
| 6 | API URL prefix inconsistency | P1 | pending |
| 7 | Personal paths hardcoded in production code | P1 | pending |
| 8 | Prometheus/Grafana pages hardcoded English | P1 | pending |
| 9 | Empty UsersPage create stub | P2 | pending |
| 10 | Format utilities not localized | P2 | pending |
| 11 | Orphan gpuLeases.ts API client | P2 | pending |
| 12 | GPU vendor filter hardcoded | P2 | pending |

## Per-Issue Detail

### 1. Transfer: active gpu_lease check (P0)
- File: internal/server/api/agent_handlers.go HandlePatchNodeTenant
- Fix: Add query for active/reserved gpu_leases on node before transfer
- Test: TestNodeTransferBlockedByActiveGpuLease
- Migration: No
- Docs: Update transfer safety rules

### 2. Transfer: active deployment_instance check (P0)
- File: internal/server/api/agent_handlers.go HandlePatchNodeTenant
- Fix: Add query for running/starting/pending/stopping instances on node before transfer
- Test: TestNodeTransferBlockedByActiveDeploymentInstance
- Migration: No
- Docs: Update transfer safety rules

### 3. audit:read role assignment (P0)
- File: internal/server/auth/bootstrap.go
- Fix: Add audit:read to admin role in BuiltinRoles
- Test: TestBuiltInAdminHasAuditRead, TestAuditReadSeedIdempotent
- Migration: No (seed in bootstrap, idempotent)
- Docs: Update role matrix

### 4. model_instances tenant isolation (P0)
- File: internal/server/api/model_handlers.go
- Fix: Add tenant_id filter to HandleListModelInstances, HandleGetModelInstance
- Test: TestModelInstancesListTenantScoped, TestModelInstanceGetOtherTenantDenied
- Migration: No
- Docs: Update tenant isolation docs

### 5. i18n key sections missing (P1)
- Files: web/src/locales/zh-CN.ts, en-US.ts
- Fix: Add all missing top-level keys for audit, tenants, users, roles, modelArtifacts, modelDeployments, modelInstances, runTemplates, runtimeEnvs, plus nav keys
- Test: i18n key consistency check via grep
- Migration: No

### 6. API URL prefix inconsistency (P1)
- Files: All web/src/api/*.ts files
- Fix: Standardize on relative paths; apiClient auto-prepends /api/v1 if needed
- Test: web build passes
- Migration: No

### 7. Personal paths hardcoded (P1)
- Files: web/src/pages/ModelDeploymentsPage.vue, scripts/e2e-model-runtime-local.sh
- Fix: Remove /home/kzeng hardcodes; use env vars in scripts, examples in Web
- Test: grep -r "/home/kzeng" web/src/ returns empty
- Migration: No

### 8. Prometheus/Grafana i18n (P1)
- Files: web/src/pages/PrometheusPage.vue, GrafanaPage.vue
- Fix: Replace hardcoded strings with t() calls; add i18n keys
- Test: web build passes
- Migration: No

### 9. UsersPage create stub (P2)
- File: web/src/pages/UsersPage.vue
- Fix: Implement create user dialog (username, display_name, password, is_platform_admin)
- Test: web build passes; dialog opens and submits
- Migration: No

### 10. Format utilities localization (P2)
- File: web/src/utils/format.ts
- Fix: Add locale-aware formatting or accept technical units
- Test: web build passes
- Migration: No

### 11. Orphan gpuLeases.ts (P2)
- File: web/src/api/gpuLeases.ts
- Fix: Delete if unused, or wire into page
- Test: web build passes; no import errors
- Migration: No

### 12. GPU vendor filter hardcoded (P2)
- File: web/src/pages/GpusPage.vue
- Fix: Dynamic vendor list from GPU data
- Test: web build passes
- Migration: No

## Verification

```
go test ./...
go build ./cmd/server && go build ./cmd/agent
cd web && npm run build
git diff --check
scripts/e2e-model-runtime-local.sh
scripts/package-release-docker.sh --no-bump
git status --short
```
