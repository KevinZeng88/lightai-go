# Tenant, RBAC & Resource Ownership Design (Phase 2F)

## Goals

- Establish infrastructure vs business tenant model for enterprise AIDC
- Define platform admin / tenant admin / operator / viewer boundaries
- Implement ResourcePool as a managed abstraction above raw GPU assignment
- Apply tenant isolation to all model runtime resources
- Enable active tenant switching with session-scoped membership validation
- Audit all sensitive operations with tenant-scoped log access

## Non-Goals

Gateway, API Key, Usage Metering, Billing, OpenAI Proxy, Rate Limit, Quota, Multi-replica scheduling, Kubernetes.

## Tenant Model

| Type | Purpose | Example |
|------|---------|---------|
| `infrastructure` | Owns nodes, GPUs, resource pools, platform-wide runtimes | IT/infra team |
| `business` | Owns models, deployments, users with limited resource access | AI team, department |
| `system` | Reserved for platform internals | LightAI system |

**Default tenant** (`slug=default`) is `infrastructure` type. New tenants default to `business`.

## Permission Model

### Built-in Roles

| Role | Scope | Key Permissions |
|------|-------|-----------------|
| `admin` | Tenant | All: user/tenant/role management, resource CRUD, deploy, audit |
| `operator` | Tenant | Resource management, deploy, start/stop/logs |
| `viewer` | Tenant | Read-only for all resources |

### Platform Admin

Set via `users.is_platform_admin=1`. Has all permissions across all tenants. Can manage users, tenants, roles globally.

### Permission Codes (27 defined)

Node: `node:read`, `node:transfer`
GPU: `gpu:read`
Model: `model:read`, `model:write`
Runtime: `runtime:read`, `runtime:write`
Deployment: `deployment:read`, `deployment:write`
Instance: `instance:read`, `instance:write`, `instance:operate`
Task: `task:read`
Membership: `membership:read`, `membership:write`
Role: `role:read`, `role:write`
Tenant: `tenant:settings:write`
Audit: `audit:read`
Platform: `platform:user:manage`, `platform:tenant:manage`, `platform:settings:write`

## Resource Ownership

| Table | Ownership Field | Notes |
|-------|----------------|-------|
| nodes | `tenant_id` | Infrastructure tenant owns hardware |
| gpu_devices | `tenant_id` | Inherits from node |
| model_artifacts | `tenant_id` | Business tenant owns models |
| runtime_environments | `tenant_id` (nullable) | NULL = global/shared |
| run_templates | `tenant_id` (nullable) | NULL = global/shared |
| model_deployments | `tenant_id` | Business tenant |
| model_instances | `tenant_id` | V6 migration added, backfilled from deployment |
| gpu_leases | `tenant_id` | Auto-assigned from deployment |
| agent_tasks | `tenant_id` | For audit and scoping |

**Future**: `owner_tenant_id` / `operator_tenant_id` distinction for shared resources.

## ResourcePool

Minimal implementation (Phase 2F):
- `resource_pools`: id, name, slug, owner_tenant_id, visibility, status
- `resource_pool_nodes`: pool_id → node_id
- `resource_pool_gpus`: pool_id → gpu_id

Pools allow a logical grouping of nodes/GPUs that can be assigned to business tenants. Full sharing policy (allowed_tenants) deferred to API Key/Gateway phase.

## Active Tenant Switching

`POST /api/v1/session/switch-tenant` validates:
1. Target tenant exists and is active
2. User is a member (platform admin can switch to any)
3. Updates `sessions.current_tenant_id`

Web UI: dropdown in top bar, page reload on switch.

## Node/GPU Transfer

`PATCH /api/v1/nodes/{id}/tenant` (existing):
- Platform admin can transfer any node
- Tenant user needs `node:transfer` permission + must own the node
- Target tenant must exist and be active
- Audit log written with old/new tenant

Safety: active gpu_lease or running deployment_instance should block transfer.

## Audit Logs

- `GET /api/v1/audit-logs` with tenant scoping
- Platform admin sees all; tenant users see their tenant's operator logs
- Sensitive fields redacted in detail
- Pagination: limit (max 200), offset
- Filters: action, entity_type, entity_id

## Tenant Isolation

Applied to: model_instances (V6 + handler update), audit_logs (API-level scoping)
Already existed: model_artifacts, model_deployments, gpu_leases, nodes, gpu_devices
