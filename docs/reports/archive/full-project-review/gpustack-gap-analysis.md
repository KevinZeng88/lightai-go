> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# LightAI Go Full Project Review - GPUStack Gap Analysis

| Area | GPUStack Reference | LightAI Go Current State | Gap | Recommendation |
|---|---|---|---|---|
| Product Scope | GPUStack includes workers, scheduler, gateway, API keys, model routes, usage, model catalog, cloud/cluster features, migrations, and many operational docs. | LightAI Go targets a lightweight Server/Agent platform with manual placement and local Docker runtime. | Scope reduction is reasonable, but runtime reliability and tenant safety must not be simplified away. | Keep scope small, but harden Agent task lease, reconciliation, tenant isolation, and upgrade paths before pilot. |
| Worker/Agent Registration | GPUStack has worker routes, worker client routers, worker status/heartbeat, and system principals (`gpustack/routes/routes.py`). | LightAI Agent uses shared bearer token, node_id/agent_id binding, heartbeat task delivery. | Shared token remains default-allowed; task delivery lacks full lease/generation semantics. | Borrow the concept of strict system principal separation and durable worker lifecycle without adopting full GPUStack complexity. |
| GPU Detection | GPUStack has detector factory and detector implementations under `gpustack/detectors/`. | LightAI uses external shell collectors and a vendor-neutral protocol. | Script protocol is suitable for light field adaptation, but MetaX remains unverified and stale/receive timestamps are conflated. | Preserve external collector approach for v1; add hardware evidence, timestamp semantics, and SDK/provider extension points. |
| Scheduling | GPUStack has scheduler and policy packages (`gpustack/scheduler`, `gpustack/policies`). | LightAI is manual placement with GPU lease rows. | Manual placement is acceptable, but lease safety and reconciliation must be reliable. | Avoid full scheduler now; implement correct lease uniqueness, state cleanup, and conflict messages. |
| Runtime Model | GPUStack has inference backends, model instances, model routes, and gateway proxy. | LightAI recently moved to Backend/Runtime/RunPlan objects. | The current Server->Agent runtime payload is inconsistent with Docker execution, and old docs still reference removed APIs. | Stabilize one runtime object model, update docs/scripts/Web/API, then add gateway/API key later. |
| Gateway / API Key | GPUStack exposes OpenAI-compatible proxy routes, API keys, model routes, usage. | LightAI explicitly defers API key/gateway/usage. | Deferral is acceptable for small deployments, but endpoint health and direct endpoint security remain limited. | Keep deferred, but design endpoint metadata and auth boundary so gateway can be added without table rewrites. |
| Database Migration | GPUStack uses Alembic migrations under `gpustack/migrations/versions/`. | LightAI uses hand-written SQLite migrations in `internal/server/db/db.go`; some resource tables are created in handlers. | LightAI migration discipline is weaker and has documented delete-DB upgrade guidance for legacy tenant data. | Move all schema to migrations, add migration tests for fresh and upgraded DBs. |
| Observability | GPUStack has routes for Prometheus/Grafana and metrics config assets. | LightAI exposes Server/Agent metrics and scripts for bundled observability. | LightAI lacks a clear product-grade supervision story and secure-by-default exposure. | Choose managed-by-script/systemd or managed-by-server; secure default binds/passwords. |
| Multi-Tenancy | GPUStack has organization/tenant-aware routes and access helpers. | LightAI has tenant/RBAC foundations. | Direct GPU detail, audit logs, and node transfer/GPU tenant synchronization have gaps. | Add direct-ID isolation tests and tenant_id on all tenant-scoped audit/resource records. |
| UI Architecture | GPUStack UI has service/request layers, access config, route config, hooks, locale files, and plugin extension points. | LightAI Web is compact Vue 3 app with Element Plus and direct API modules. | LightAI UI simplicity is fine, but tests are not wired and some docs/API paths are stale. | Keep simple UI; add runnable tests, API error states, and route/API contract checks. |
| Tests | GPUStack reference contains broad `tests/` categories. | LightAI has focused Go tests and some disconnected Web test files. | Critical cross-component runtime behavior is under-tested. | Add minimal cross-boundary tests: runplan -> AgentRunSpec -> Docker options; tenant direct-ID; migration; patch; scripts. |

## Capabilities LightAI Go Should Borrow Now

- Worker/Agent task lease and idempotent state reporting.
- Reconciliation after worker/agent restart.
- Migration discipline and upgrade tests.
- Tenant-aware API filtering by resource ownership, including direct ID endpoints.
- Runtime/backend version separation, but with one stable API/documentation model.

## Capabilities LightAI Go Can Keep Lightweight

- Manual node/GPU placement instead of full scheduler.
- External command GPU collectors before SDK/NVML/ROCm integrations.
- Direct model endpoint display before full gateway/API key.
- SQLite for small deployments, if migrations and retention are controlled.

## Capabilities To Defer But Preserve

- API key and OpenAI-compatible gateway.
- Token usage/billing/quota.
- Multi-replica scheduling.
- Kubernetes/Ray/multi-cluster.
- SSO/OIDC/SAML.
- Advanced model catalog and download workflows.
