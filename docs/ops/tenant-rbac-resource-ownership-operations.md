# Tenant, RBAC & Resource Ownership — Operations Guide

## Creating Infrastructure/Business Tenants

```bash
# Infrastructure tenant (platform admin only)
curl -X POST /api/v1/tenants -d '{"name":"AI Infrastructure","slug":"ai-infra","type":"infrastructure"}'

# Business tenant
curl -X POST /api/v1/tenants -d '{"name":"NLP Team","slug":"nlp-team","type":"business"}'
```

## Managing Users

```bash
# Create user (platform admin)
curl -X POST /api/v1/users -d '{"username":"alice","display_name":"Alice","password":"..."}'

# Add user to tenant
curl -X POST /api/v1/tenant-memberships -d '{"tenant_id":"...","user_id":"...","role_ids":["operator-role-id"]}'
```

## Switching Active Tenant

Web: Use dropdown in top bar next to username.

API: `POST /api/v1/session/switch-tenant {"tenant_id":"..."}`

Requirements: User must be active member of target tenant (platform admin exempt).

## Viewing Audit Logs

Web: System → Audit Logs

API: `GET /api/v1/audit-logs?action=start&entity_type=model_deployment&limit=50`

Requires `audit:read` permission. Tenant-scoped for non-admin users.

## Transferring Nodes/GPUs

```bash
# Transfer node to another tenant
curl -X PATCH /api/v1/nodes/{id}/tenant -d '{"tenant_id":"target-tenant-id"}'
```

**Before transfer:** Ensure no active deployments or GPU leases on the resource. Platform admin can force-transfer.

## Common Issues

### Cannot see nodes/GPUs
Check: Are you in the correct active tenant? Switch via top bar dropdown.

### Audit logs empty
Check: Do you have `audit:read` permission? Non-admin users only see logs for their tenant.

### Cannot switch tenant
Check: Are you an active member of the target tenant? Platform admin can switch to any.

### Model instances not visible
Check: Instances are scoped to your active tenant. Switch to the tenant that owns the deployment.

### Transfer rejected
Check: No active deployment/lease on the resource. Target tenant must exist and be active.

## Security Notes

- Never expose password_hash, token, secret, api_key, cookie, or authorization headers
- Check for active deployments/leases before transferring GPU resources
- Platform admin should only be used for emergency/break-glass operations
- Regular tenant management should be done by tenant admins
