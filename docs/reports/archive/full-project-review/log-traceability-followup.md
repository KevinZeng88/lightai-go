> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# Log Traceability Audit

**Date:** 2026-06-17
**Branch:** `phase-3-runtime-observability-closeout`

## Audit Summary

Code audit of all key logging points in the deployment lifecycle chain. Each link evaluated for: log existence, field completeness, noise level.

## Per-Link Audit

| Link | Current Log Exists | Log Message | Missing Fields | Required Fix |
|------|-------------------|-------------|----------------|-------------|
| Deployment dry-run | **NO** | (none) | request_id, tenant_id, actor_id, deployment_id, errors | Add `deployment.dry_run.succeeded` / `deployment.dry_run.failed` |
| Deployment create | **PARTIAL** | Only ERROR on failure | Success log, tenant_id, actor_id | Add `deployment.created` INFO + operation context |
| Instance start request | **PARTIAL** | `operation_started` (generic) | instance_id, node_id, agent_id, gpu_ids, tenant_id, actor_id | Add `instance.start.requested` with full context |
| Task created | **NO** | (none) | operation_id, task_id, task_type, deployment_id, instance_id, agent_id, node_id, generation | Add `agent_task.created` after INSERT |
| Task claimed by Agent | **NO** | Only generic `claim: claimed tasks` (aggregated) | Per-task: operation_id, task_id, agent_id, node_id, deployment_id, instance_id, generation, attempt | Add per-task `agent_task.claimed` in claimAndReturnTasks |
| Docker create | **YES** | `docker.create.started` / `docker.create.completed` | deployment_id (has instance_id, operation_id) | Add deployment_id |
| Docker start | **YES** | `docker.start.started` / `docker.start.completed` | deployment_id | Add deployment_id |
| Health check success | **YES** | `health_check.completed` | operation_id propagation from ctx | Rename to `runtime.health_check.succeeded` |
| /v1/models smoke | **N/A** | External endpoint — not logged by LightAI | — | No fix needed (external inference endpoint) |
| Instance stop | **PARTIAL** | `operation_started` (generic) | instance_id, task_id, container_id, actor_id | Add `instance.stop.requested` / `instance.stop.completed` |
| Docker stop/remove | **PARTIAL** | `docker.stop.started/completed` | Remove log | Add `docker.container.removed` |
| Lease release | **YES** | `gpu_lease.released` with state_transition | — | ✅ Complete |
| Task result reported | **YES** | `task_result.report_completed` | container_id, endpoint on success | Add container_id, endpoint_url |
| Task result processed | **YES** | `task.result.processed` / `task.result.failed` | — | ✅ Complete |
| Instance state updated | **NO** | (none) | operation_id, instance_id, old_state, new_state, container_id, endpoint | Add `instance.state.updated` |
| Audit log recorded | **NO** | No audit log writer exists | tenant_id, actor_id, action, resource_type, resource_id, result | Create audit log writer + insert calls |
| HTTP access log | **NO** | No middleware-level access log | method, path, status, duration, request_id | Add lightweight access log middleware |

## Noise Assessment

| Source | Current Behavior | Verdict |
|--------|-----------------|---------|
| Heartbeat success | Summary every ~60s at INFO; failures at WARN (rate-limited) | ✅ Low noise |
| Task poll (no task) | Summary every ~60s; individual polls at DEBUG | ✅ Low noise |
| GPU metrics | Summary every ~60s | ✅ Low noise |
| /metrics endpoint | No INFO-level /metrics access logs | ✅ No noise |
| Reconcile | At startup + every 60s; "no containers" at DEBUG | ✅ Low noise |
| Model instance list | Only INFO when results > 0; DEBUG otherwise | ✅ Good |
| Docker operations | One create + one start per task = 4 INFO lines | ✅ Acceptable |

## Fields Completeness

| ID Field | Server Logs | Agent Logs | DB |
|----------|------------|------------|-----|
| request_id | ✅ (via ctx) | N/A | N/A |
| operation_id | ✅ (StartOperation) | ✅ (via AgentRunSpec) | ✅ (agent_tasks.operation_id) |
| deployment_id | ✅ (in handler args) | ✅ (in AgentRunSpec) | ✅ |
| instance_id | ✅ (in handler args) | ✅ (in AgentRunSpec) | ✅ |
| task_id | ✅ (in handler) | ✅ (in task struct) | ✅ |
| agent_id | ❌ (not always) | ✅ | ✅ |
| node_id | ✅ (in handler args) | ✅ | ✅ |
| container_id | ❌ (not always) | ✅ | ✅ |
| lease_id | ✅ (in lease ops) | N/A | N/A |
| tenant_id | ✅ (via ctx/request) | ✅ (in task) | ✅ |
| actor_id | ❌ (missing) | N/A | ❌ (not in all audit entries) |

## Implementation Plan

### Phase 1: Missing Logs (Server)

1. **HandleDeploymentDryRun** — add INFO/WARN logs with context
2. **HandleCreateDeployment** — add `deployment.created` INFO
3. **HandleStartDeployment** — add `agent_task.created` log, add `instance.start.requested`
4. **HandleStopDeployment** — add `instance.stop.requested` / `instance.stop.completed`
5. **HandleTaskResult** — add `instance.state.updated`
6. **claimAndReturnTasks** — add per-task `agent_task.claimed` log

### Phase 2: Missing Logs (Agent)

7. **processTask** — add `agent_task.execution.started`
8. **DockerRuntimeDriver** — add `docker.container.removed`, add deployment_id to existing logs
9. **Health check** — rename `health_check.completed` → `runtime.health_check.succeeded`

### Phase 3: Audit Log Writer

10. Create `internal/server/api/audit_writer.go` with a reusable audit log function
11. Add audit calls to key handlers (deployment create, dry-run, start, stop, artifact create, runtime create)

### Phase 4: Access Log Middleware

12. Add lightweight request logging middleware with request_id, method, path, status, duration

### Phase 5: Verify

13. Run real Docker E2E with full log capture
14. Verify correlation chain
15. Update reports
