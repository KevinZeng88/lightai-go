# LightAI Go v0.1.9 RC — Tenant Model Fix

## Core Fix

- **Tenant ID**: `tenant_id` fields now use UUIDs (not the literal string `'default'`).
- **Tenants table**: Added `slug` column. Default tenant has `slug='default'`, deterministic UUID `a0000000-0000-0000-0000-000000000001`.
- **Agent registration**: New nodes get `tenant_id = default tenant UUID`. Re-registration does not overwrite existing tenant_id.
- **Node list**: Platform admin sees all nodes. Regular users see only their tenant's nodes.
- **Node transfer**: `PATCH /api/nodes/{id}/tenant` — platform admin can transfer any node; tenant users with `node:transfer` permission can transfer nodes within their tenant.
- **API fallback**: Unregistered `/api/*` paths return JSON 404, not SPA index.html.
- **Permissions**: Added `node:transfer` permission. Built-in admin role includes it.

## Breaking Change

If your database has `nodes.tenant_id = 'default'` or `gpu_devices.tenant_id = 'default'` from a previous build, delete `data/lightai.db` and restart. This release does not include legacy migration.

## Future

- GPUDevice independent tenant assignment (RC2)
- API Key, usage tracking, billing (RC3+)
