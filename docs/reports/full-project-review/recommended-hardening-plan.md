# LightAI Go Full Project Review - Recommended Hardening Plan

## P0 - Must Fix Before Pilot / Release Candidate

| Goal | Modules | Direction | Verification | Risk |
|---|---|---|---|---|
| Enforce non-default Agent token | `cmd/server`, `cmd/agent`, `configs/*`, scripts | Refuse non-dev startup with empty/default token; generate install token; document rotation. | Start server/agent with default token in release config must fail; dev config must require explicit insecure flag if allowed. | Without this, fake agents can join a deployment. |
| Fix direct GPU tenant isolation | `internal/server/api/resource_handlers.go` | Add tenant check to `HandleGetGPU`; test direct ID from another tenant returns 404. | New Go test for cross-tenant GPU detail. | Direct data leak across tenants. |
| Correct Server -> Agent runtime payload and Docker execution mapping | `deployment_lifecycle_handlers.go`, `internal/agent/runtime/docker.go`, `driver.go`, tests | Include vendor, command/entrypoint, ports, mounts, devices; map Docker Entrypoint and Cmd correctly; ensure NVIDIA DeviceRequests are set. | Unit test from real resolved plan to fake Docker create options; NVIDIA model E2E. | Model runtime can fail while dry-run looks correct. |
| Implement task lease/generation/idempotency baseline | `agent_handlers.go`, DB schema, Agent task reporting | Add lease owner/expires, operation_id, generation/attempt, max attempts; condition updates by status+lease owner. | Race tests for double heartbeat claim and duplicate result. | Duplicate/stale task results can corrupt instance/lease state. |
| Add runtime reconciliation | Agent runtime, Server task/result/status APIs | Agent scans managed containers at startup and periodically reports status. Server rejects stale generation/status. | Kill container after start; DB moves to failed/unknown. Restart Agent; DB reconciles. | Runtime state can drift indefinitely. |

## P1 - Must Fix For Next Development Stage

| Goal | Modules | Direction | Verification | Risk |
|---|---|---|---|---|
| Normalize lifecycle states | `agent_handlers.go`, Web status tags, docs | Use documented `failed`, `unknown`, `stopped`; remove `error` state or migrate it. | State transition tests and UI display smoke. | Inconsistent status handling. |
| Make stop idempotent | Agent Docker driver, Server stop handler | Missing managed container should be accepted as stopped; release leases. | Stop after manual `docker rm`; expected stopped/released. | Cleanup failures and lease leaks. |
| Synchronize GPU tenant on node transfer | `HandlePatchNodeTenant`, GPU list/detail tests | Update node and GPUs in one transaction; record audit tenant. | Transfer node with GPUs; old tenant no longer sees GPU, new tenant does. | Resource ownership inconsistency. |
| Add tenant_id to audit logs | DB migration, audit writes, audit list | Scope audit rows by resource/action tenant, not operator membership. | Multi-tenant user audit query test. | Cross-tenant audit leakage. |
| Centralize resource schema migration | `internal/server/db`, tests | Move GPU/system/filesystem/network tables into migrations and fail on errors. | Fresh DB and upgraded DB schema tests. | Schema drift and runtime startup surprises. |
| Repair ops docs/scripts for new runtime model | `docs/ops`, `docs/testing`, scripts | Replace old RuntimeEnvironment/RunTemplate paths with BackendRuntime/RunPlan paths. | Execute docs as smoke script in disposable env. | Customer runbooks fail. |

## P2 - Recommended Optimization

| Goal | Modules | Direction | Verification | Risk |
|---|---|---|---|---|
| Wire frontend tests | `web/package.json`, Web tests | Add Vitest or remove stale tests; include `npm test` in CI. | `cd web && npm test` passes. | UI regressions not caught. |
| Update OpenAPI | `docs/api/openapi.yaml`, route/DTO definitions | Regenerate or manually update all current `/api/v1` endpoints. | Diff route list against OpenAPI paths. | Integrator confusion. |
| Secure observability defaults | configs, scripts, docs | Default to localhost or documented reverse proxy; enforce Grafana password setup. | Release smoke checks bound addresses and passwords. | Metrics/Grafana exposure. |
| Clarify ignored config keys | config loader, docs | Implement `report_interval` and `advertise_addr`, or fail/warn clearly when set. | Config tests for intervals and targets. | Operator misconfiguration. |
| Separate collected_at and reported_at | Agent report, DB/API/Web | Store collector timestamp and server receive timestamp separately. | Stale collector report shows stale data age. | Misleading freshness indicators. |
| Reduce Web bundle chunk | Vite config | Manual chunks for Element Plus and major pages. | Build without chunk warning; smoke load. | Slow first load. |

## P3 - Longer-Term Evolution

| Goal | Modules | Direction | Verification | Risk |
|---|---|---|---|---|
| API key / gateway foundation | Server API, models, future proxy | Add after runtime stability; keep Agent token/User session/API key strictly separated. | Gateway/API key integration tests. | Premature gateway can destabilize core resource platform. |
| Scheduling | Server scheduler, leases, resource pools | Start with manual placement plus validations; add binpack/spread later. | Simulated multi-node scheduling tests. | Overbuilding too early. |
| GPU provider evolution | Agent collector | Keep script protocol; add provider implementations only after field need. | Vendor sample and parser tests. | Vendor-specific complexity. |
| High availability | Server DB/deployment | Document single-node control plane limits first; consider external DB later. | Failover design review. | Complexity exceeds lightweight target. |
