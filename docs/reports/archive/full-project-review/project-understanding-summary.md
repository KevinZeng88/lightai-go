> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# LightAI Go Full Project Review - Project Understanding Summary

## Project Goal

LightAI Go is a lightweight GPU/node and model-serving management platform for small and medium GPU deployments. It intentionally borrows architecture ideas from GPUStack, but the project documents explicitly require a smaller first-stage scope: local Server/Agent binaries, SQLite-backed current state, GPU discovery/metrics, basic Web Console, tenant/RBAC foundation, and Docker-based model runtime.

## Current Capability Scope

Current documentation and code indicate the project has implemented:

- Server and Agent binaries with health and metrics endpoints.
- Local auth, session, CSRF, tenant membership, built-in roles, permission catalog, and Web login.
- Agent registration, heartbeat, node online/offline state, and resource reports.
- System, NVIDIA, MetaX script-based GPU collector architecture, with MetaX still requiring real hardware validation.
- Vue 3 Web Console pages for dashboard, nodes, GPUs, observability, backend runtimes, model artifacts, model deployments, model instances, users, tenants, roles, and audit logs.
- Backend/Runtime/RunPlan-oriented model runtime objects, replacing older runtime environment/run template tables in `migrateV10`.
- Packaging and operational scripts for release, patching, start/stop, observability, password reset, log collection, and local verification.

The implementation is more advanced than the older Phase 0-2 wording in `docs/README.md`, but not yet mature enough to treat the model runtime path as production-ready.

## Core Modules

- Server control plane:
  - `cmd/server/main.go`
  - `internal/server/api/`
  - `internal/server/auth/`
  - `internal/server/db/`
  - `internal/server/runplan/`
  - `internal/server/rbac/`
- Agent execution plane:
  - `cmd/agent/main.go`
  - `internal/agent/collector/`
  - `internal/agent/register/`
  - `internal/agent/runtime/`
  - `internal/agent/metrics/`
- Web Console:
  - `web/src/api/`
  - `web/src/pages/`
  - `web/src/router/`
  - `web/src/stores/`
- Operational assets:
  - `configs/`
  - `deploy/`
  - `scripts/`
  - `docs/ops/`
  - `docs/testing/`

## Core Runtime Chain

The current intended chain is:

```text
ModelArtifact
  -> BackendRuntime
  -> ModelDeployment
  -> Resolve RunPlan
  -> ResolvedRunPlan
  -> AgentTask
  -> Agent DockerRuntimeDriver
  -> Docker container
  -> Task result
  -> ModelInstance / GPU lease state
```

The older documented/manual chain is still present in some docs and scripts:

```text
ModelArtifact
  -> RuntimeEnvironment
  -> RunTemplate
  -> ModelDeployment
```

That mismatch is a major documentation and operational risk.

## Main Documents Read

- `AGENTS.md`
- `docs/README.md`
- `docs/PHASE-STATUS.md`
- `docs/RELEASE_NOTE_v0.1.9.md`
- `docs/00-project-scope.md` through `docs/10-mvp-development-plan.md`
- `docs/08-engineering-contracts.md`
- `docs/09-auth-tenant-design.md`
- `docs/GPU_COLLECTOR_ARCHITECTURE.md`
- `docs/design/12-model-runtime-serving-design.md`
- `docs/design/13-backend-runplan-runtime-design.md`
- `docs/design/tenant-rbac-resource-ownership-design.md`
- `docs/ops/*`
- `docs/testing/*`
- `docs/api/openapi.yaml`
- RC/Phase review reports under `docs/reports/`, `docs/review/`

## GPUStack Relationship

GPUStack is a maturity reference, not source code to copy. The reference projects show a fuller platform surface: worker/client split, detector factory, scheduler policies, gateway/OpenAI proxy, API keys, usage, model routes, migrations, observability, cluster/worker pools, generated clients, and broad test directories. LightAI Go should not copy this scope wholesale, but it should preserve architectural hooks for:

- durable worker/agent identity and task leases,
- robust runtime reconciliation,
- GPU/resource allocation safety,
- schema migration discipline,
- model serving gateway/API key future path,
- operational diagnostics and supportability.

## Review Scope

This review covered architecture, product maturity, security, reliability, observability, model runtime, GPU adaptation, Web/API consistency, database/migration, packaging/scripts, test coverage, documentation, and GPUStack gap analysis. No business code, config, script, or test file was modified.
