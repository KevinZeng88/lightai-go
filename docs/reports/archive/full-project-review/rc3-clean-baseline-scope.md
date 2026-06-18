> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# RC3 Clean Baseline Scope

## 1. Purpose

This document defines the only supported current product baseline for LightAI Go RC3.

RC3 intentionally removes compatibility with old runtime model objects, old API paths, old database structures, old Web flows, and old operational instructions when they conflict with the current product model.

## 2. Only Supported Runtime Model

Current model chain:

```text
ModelArtifact
  -> BackendRuntime
  -> ModelDeployment
  -> ResolvedRunPlan
  -> AgentTask
  -> Agent DockerRuntimeDriver
  -> Docker container
  -> Health Check
  -> ModelInstance / GPU lease state
```

The old model chain is removed from current product scope:

```text
RuntimeEnvironment
  -> RunTemplate
  -> ModelDeployment
```

## 3. Only Supported API Surface

- Current API prefix: `/api/v1`
- Current runtime objects:
  - ModelArtifact
  - BackendRuntime
  - ModelDeployment
  - ModelInstance
  - AgentTask
  - GPU lease
  - Audit log
  - Node/GPU resources
  - Tenant/RBAC resources

Removed from current API:
- `/runtime-environments`
- `/run-templates`
- Any route that exposes RuntimeEnvironment or RunTemplate as active product functionality.

Historical references may remain only when marked obsolete and never used as current operation steps.

## 4. Only Supported Web Scope

Web navigation must expose the current product model:

- Dashboard
- Nodes
- GPUs
- Models / Model Artifacts
- Backend Runtimes
- Model Deployments
- Model Instances
- Observability
- Users / Tenants / Roles / Audit

Web must not expose active old pages for:
- Runtime Environments
- Run Templates

## 5. Only Supported Database Scope

Fresh RC3 database must contain only current schema tables required by the current product model.

Core table categories:
- users / sessions / auth
- tenants / memberships / roles / permissions
- nodes
- gpu_devices
- host/system metrics if supported
- model_artifacts
- backend_runtimes
- model_deployments
- model_instances
- agent_tasks
- gpu_leases
- audit_logs with tenant scoping
- schema/version metadata

Fresh RC3 database must not create obsolete runtime-environment/run-template tables.

## 6. Only Supported Configuration Scope

Configurations must reflect implemented behavior only.

Remove or implement/warn clearly for fields such as:
- `report_interval`
- `metrics.advertise_addr`
- any old runtime environment or run template config

Release defaults must be secure:
- no default shared Agent token in non-dev mode
- observability not exposed insecurely by default
- privileged runtime profiles clearly labeled

## 7. Only Supported E2E Flow

Current E2E must validate:

```text
fresh install
  -> initial credentials
  -> Web login
  -> Server health
  -> start-all
  -> Agent registration
  -> GPU discovery
  -> ModelArtifact create
  -> BackendRuntime create
  -> ModelDeployment create
  -> dry-run
  -> start instance
  -> endpoint health check
  -> stop instance
  -> lease release
  -> reconciliation
  -> observability
  -> stop-all
```

## 8. Required Legacy Scan

Run:

```bash
rg "/runtime-environments|/run-templates|RuntimeEnvironment|RunTemplate|runtime environment|run template" .
```

Every result must be classified as:
- Removed
- Rewritten for current model
- Obsolete historical reference
- Test fixture still valid for migration/removal behavior

No result may remain as current operator guidance, current API contract, active Web route, active config, or active E2E path.
