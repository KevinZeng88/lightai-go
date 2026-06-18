> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference or historical compatibility document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Go Changelog

## v0.1.14 (2026-06-16)

### Security
- HMAC-SHA256 session hashing (was plain SHA-256)
- CSRF origin check uses exact URL host comparison
- Rate limiter stale entry cleanup (prevents memory leak)
- Login metrics (success/failure counters) now increment
- Password-expired users can now access change-password endpoint
- Hardcoded credentials removed from Grafana/observability pages

### Tenant Isolation
- Model instances now have tenant_id set on creation
- GPU devices inherit tenant_id from their node
- Dynamic tenant UUID lookup (was hardcoded default UUID)
- Node transfer + audit log are atomic (same transaction)

### Agent
- Task processing runs in goroutines (no longer blocks heartbeat)
- Docker log stream properly decoded (stdout/stderr separated)
- External collector inherits parent environment
- Load metrics always emitted (even at zero)

### Web
- Roles page: create, delete, edit permissions
- Tenants page: edit, disable
- Users page: edit, disable, reset password
- Observability pages fully i18n'd
- 284 i18n keys (zh-CN + en-US)
- Removed dead PlaceholderPage, dead code

### Database
- V8 migration: partial unique index on gpu_leases for reserved/active
- Sweep uses portable strftime() (was SQLite-specific julianday())

## v0.1.13
- Phase 2F: tenant switching API and Web UI, audit log page, V7 migration

## v0.1.12
- Phase 2F: tenant RBAC hardening, 12 review fixes

## v0.1.11
- Phase runtime: conservative sweep, quick deploy web wizard

## v0.1.10
- Phase runtime: local model runtime E2E with logs verification
- Agent Docker runtime driver

## v0.1.9
- Tenant model fix: UUID tenant_id, slug='default', node transfer API
- Credentials, password reset, file logging, patch tooling

## v0.1.8 and earlier
- Phase 0: Server/Agent skeleton
- Phase 0.5: Auth, tenant, RBAC
- Phase 1: Agent register & heartbeat
- Phase 2: GPU collectors (NVIDIA, MetaX), resource reporting
- Phase 3W: Web Console MVP
