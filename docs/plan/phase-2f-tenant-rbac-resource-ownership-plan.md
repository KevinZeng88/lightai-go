# Phase 2F: Tenant, User, Role, RBAC & Resource Ownership Productization

## 1. Current Baseline

Phase 0.5 delivered auth, tenant, RBAC skeleton. Phase 2E delivered model runtime operations hardening.

**Already in place:**
- `tenants` table with id/slug/name/status
- `users` table with password_hash, is_platform_admin, status
- `roles` table with built_in flag, tenant_id (NULL = global)
- `permissions` table with 27 codes
- `role_permissions`, `tenant_memberships`, `tenant_membership_roles`
- `sessions` table with current_tenant_id
- `audit_logs` table (write-only, no read API)
- RBAC middleware: SessionMiddleware, RequirePermission, RequirePlatformAdmin, AgentAuthMiddleware
- Platform admin APIs: GET/POST/PUT users, GET/POST/PUT tenants
- Tenant admin APIs: memberships, roles, permissions
- Model runtime APIs with permission checks (model:read/write, runtime:read/write, deployment:read/write, instance:read)

**Gaps identified:**
- No tenant `type` field (infrastructure/business)
- No `owner_tenant_id` / `operator_tenant_id` on resources
- No `ResourcePool` concept
- `model_instances` handler lacks tenant isolation
- No audit log read API or Web page
- No Web pages for tenant/user/role management
- No active tenant switching in Web
- GPU devices hardcode default tenant UUID
- No node/GPU transfer API beyond basic PATCH

## 2. Phase Goals

1. Add tenant type field and infrastructure/business tenant model
2. Add ResourcePool schema (minimal implementation)
3. Add audit log read API and Web page
4. Add Web management pages: tenants, users, roles
5. Add active tenant switching in Web
6. Enhance node/GPU transfer with safety checks
7. Fix model_instances tenant isolation
8. Comprehensive RBAC tests
9. Document the enterprise AIDC permission model

## 3. Non-Goals

Gateway, API Key, Usage Metering, Billing, OpenAI Proxy, Rate Limit, Quota, Multi-replica scheduling, Kubernetes.

## 4. Step-by-step Execution

| Step | Description | Key Files | Migration? | Web Build? |
|------|-------------|-----------|------------|------------|
| 0 | Baseline verification | - | No | Yes |
| 1 | Create plan + audit | docs/plan/ | No | No |
| 2 | Schema: tenant type, ResourcePool | db/db.go V7 | Yes | No |
| 3 | Audit log API + Web page | api/audit_handlers.go | No | Yes |
| 4 | Web tenant management page | pages/TenantsPage.vue | No | Yes |
| 5 | Web user management page | pages/UsersPage.vue | No | Yes |
| 6 | Web role management page | pages/RolesPage.vue | No | Yes |
| 7 | Active tenant switching | Layout, router, store | No | Yes |
| 8 | Node/GPU transfer hardening | agent_handlers.go, resource_handlers.go | No | Yes |
| 9 | Fix model_instances tenant isolation | model_handlers.go | No | No |
| 10 | RBAC tests | *_test.go | No | No |
| 11 | Docs: design + operations | docs/design/, docs/ops/ | No | No |
| 12 | Regression: build/test/E2E/package | - | No | Yes |

## 5. Verification

After each step: `go test ./...`, `go build ./cmd/server && go build ./cmd/agent`.
After Web steps: `cd web && npm run build`.
Final: `scripts/e2e-model-runtime-local.sh && scripts/package-release-docker.sh --no-bump`.
