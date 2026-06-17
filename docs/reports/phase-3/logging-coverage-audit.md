# Logging Coverage Audit — Full Platform (with High-Frequency Strategy)

> Updated: 2026-06-17 — Phase 3 hardening round

## A. Server Lifecycle (low frequency, all INFO)

| Operation | Started | Completed | Failed | Duration | Summary | This Round |
|-----------|---------|-----------|--------|----------|---------|------------|
| server start | ✅ INFO | ✅ INFO | ✅ Fatal | ❌ | ✅ | — |
| config load | ❌ | ❌ | ✅ Fatal | ❌ | ❌ | Gap: no config summary |
| DB open | ✅ INFO | ✅ INFO | ✅ Fatal | ✅ ms | ✅ | **Added duration** |
| DB migrate | ✅ INFO | ✅ INFO | ✅ ERROR | ✅ ms | ✅ | **Added duration** |
| seed/init | ✅ INFO | ✅ INFO | ✅ ERROR | ✅ ms | ✅ | **Added duration** |
| HTTP listen | ✅ INFO | ✅ INFO | ✅ Fatal | ❌ | ✅ | — |
| graceful shutdown | ✅ INFO | ✅ INFO | ✅ ERROR | ❌ | ✅ | — |

## B. API Common Path

| Operation | HF? | Strategy | Started | Completed | Failed | Duration | Summary | Gap |
|-----------|-----|----------|---------|-----------|--------|----------|---------|-----|
| request received | No | DEBUG | ✅ DEBUG | N/A | N/A | N/A | ✅ | **Added middleware** |
| request completed | No | INFO | N/A | ✅ INFO | ✅ WARN/ERROR | ✅ ms | ✅ | **Added middleware** |
| auth check | No | DEBUG | ❌ | ❌ | ✅ WARN | ❌ | ❌ | Added agent auth WARN |
| session validation | No | INFO | ❌ | ❌ | ✅ WARN | ❌ | ❌ | |
| CSRF validation | No | INFO | ❌ | ❌ | ✅ 403 | ❌ | ❌ | |
| tenant resolve | No | DEBUG | ❌ | ❌ | ❌ | ❌ | ❌ | |
| RBAC check | No | INFO | ❌ | ❌ | ✅ WARN | ❌ | ✅ | **Added permission.denied WARN** |
| validation failure | No | WARN | ❌ | N/A | ⚠️ | ❌ | ❌ | Inconsistent |
| API error response | No | ERROR | ❌ | N/A | ❌ | ❌ | ❌ | No centralized |
| request duration | No | INFO | ❌ | ✅ INFO | ✅ INFO | ✅ ms | ✅ | **Added middleware** |

## C. Auth / Tenant / RBAC (low frequency)

| Operation | Started | Completed | Failed | Gap |
|-----------|---------|-----------|--------|-----|
| login | ❌ | ❌ | ✅ | No success log, no duration |
| unauthorized (401) | ❌ | ❌ | ⚠️ | No request context |
| forbidden (403) | ❌ | ❌ | ⚠️ | No permission detail |

## D. Node / Agent

| Operation | HF? | Strategy | Started | Completed | Failed | Duration | Summary | Gap |
|-----------|-----|----------|---------|-----------|--------|----------|---------|-----|
| node register | No | INFO | ✅ | ✅ | ❌ | ❌ | ✅ | |
| node heartbeat | **YES** | **Sampled INFO + 60s summary** | ✅ DEBUG | ✅ DEBUG | ✅ WARN | ✅ ms | ✅ **Added** | Summary + state-change logging |
| node online/offline | No | INFO | ✅ | N/A | ❌ | ❌ | ✅ | |
| agent start | No | INFO | ✅ | ✅ | ✅ | ❌ | ✅ | |
| agent task polling | **YES** | **60s summary** | ✅ DEBUG | ✅ DEBUG | ❌ | ❌ | ✅ **Added** | Task poll summary |
| agent task claim | No | INFO | ⚠️ DEBUG | ❌ | ❌ | ❌ | ⚠️ | |
| agent task result | No | INFO | ❌ | ❌ | ❌ | ❌ | ❌ | |

## E. GPU / Resource

| Operation | HF? | Strategy | Started | Completed | Failed | Duration | Summary | Gap |
|-----------|-----|----------|---------|-----------|--------|----------|---------|-----|
| GPU discover | No | INFO | ✅ | ✅ | ❌ | ❌ | ✅ | |
| GPU metrics collect | **YES** | **DEBUG + 60s summary** | ✅ DEBUG | ✅ DEBUG | ❌ | ✅ ms | ✅ | **Added** summary + duration |
| GPU health update | No | INFO on change | ❌ | ❌ | ❌ | ❌ | ❌ | |
| GPU lease reserve | No | INFO | ❌ | ❌ | ❌ | ❌ | ❌ | |
| GPU lease activate | No | INFO | ❌ | ❌ | ❌ | ❌ | ❌ | |
| GPU lease release | No | INFO | ❌ | ❌ | ❌ | ❌ | ❌ | |

## F. Model Resource CRUD (low frequency)

| Operation | Started | Completed | Failed | Duration | Gap |
|-----------|---------|-----------|--------|----------|-----|
| backend list/get | ❌ | ❌ | ✅ | ❌ | No success log |
| runtime create/delete | ❌ | ❌ | ✅ | ❌ | |
| artifact create/delete | ❌ | ❌ | ✅ | ❌ | |
| deployment create/delete | ✅ | ✅ | ✅ | ✅ ms | Added this round |
| instance list/get | ❌ | ❌ | ❌ | ❌ | |

## G. RunPlan / Resolver (low frequency)

| Operation | Started | Completed | Failed | Duration | Summary |
|-----------|---------|-----------|--------|----------|---------|
| resolve | ✅ INFO | ✅ INFO | ❌ | ✅ ms | ✅ (backend, vendor, image, args, errors) |

## H. Deployment / Instance Lifecycle (low frequency, all INFO)

| Operation | Started | Completed | Failed | Duration | Summary | This Round |
|-----------|---------|-----------|--------|----------|---------|------------|
| start | ✅ INFO | ✅ INFO | ❌ | ✅ ms | ✅ (instance_id, task_id, operation_id) | Added operation_id |
| stop | ✅ INFO | ✅ INFO | ❌ | ✅ ms | ✅ (instances_stopped) | |
| delete | ✅ INFO | ✅ INFO | ❌ | ✅ ms | ✅ | |
| state transition | ❌ | ❌ | ❌ | ❌ | ❌ | Gap: no state_from/to |

## I. Agent Task Execution

| Operation | HF? | Strategy | Started | Completed | Failed | Duration | Gap |
|-----------|-----|----------|---------|-----------|--------|----------|-----|
| task received | No | INFO | ✅ | ❌ | ❌ | ❌ | No completed |
| task payload parsed | No | DEBUG | ❌ | ❌ | ❌ | ❌ | |
| task execution start | No | INFO | ❌ | ❌ | ❌ | ❌ | |
| task execution done | No | INFO | ❌ | ❌ | ❌ | ❌ | |
| task result reported | No | INFO | ❌ | ❌ | ❌ | ❌ | |

## J. Docker Runtime

| Operation | HF? | Started | Completed | Failed | Duration | Gap |
|-----------|-----|---------|-----------|--------|----------|-----|
| docker create | No | ✅ INFO | ✅ INFO | ✅ ERROR | ✅ ms | **Added** full lifecycle |
| docker start | No | ✅ INFO | ✅ INFO | ✅ ERROR | ✅ ms | **Added** full lifecycle |
| docker stop | No | ✅ INFO | ✅ INFO | ✅ ERROR | ✅ ms | **Added** full lifecycle |
| container exit | No | ✅ ERROR | N/A | ✅ ERROR | N/A | **Added** inspect + logs tail |
| docker spec dump | No | N/A | ✅ DEBUG | N/A | N/A | **Added** spec dump (DEBUG) |
| slow operation | No | N/A | ✅ WARN | N/A | N/A | **Added** slow_op WARN |

## K. Health Check

| Operation | HF? | Strategy | Started | Completed | Failed | Duration | Gap |
|-----------|-----|----------|---------|-----------|--------|----------|-----|
| health check start | No | INFO | ❌ | ❌ | ❌ | ❌ | |
| retry progress | **YES** | **DEBUG, not INFO** | ❌ | ❌ | ❌ | ❌ | |
| final success | No | INFO | ❌ | ❌ | ❌ | ❌ | |
| final failure | No | WARN | ❌ | ❌ | ❌ | ❌ | |
| timeout | No | ERROR | ❌ | ❌ | ❌ | ❌ | |

## L. E2E / Smoke Scripts

| Operation | Started | Completed | Failed | Duration | Wait Log | This Round |
|-----------|---------|-----------|--------|----------|----------|------------|
| quick | ✅ | ✅ | ❌ | ❌ | N/A | |
| api-only | ✅ | ✅ | ✅ | ❌ | N/A | |
| single backend | ✅ | ⚠️ | ✅ | ❌ | ✅ (all 4 types) | Added wait_started/progress/completed/timeout |
| smoke all | ✅ | ✅ | ✅ | ❌ | ❌ | |

## Summary by Domain

| Domain | Total | ≥Started | ≥Completed | ≥Failed | Duration | Wait Log | HF Strategy |
|--------|-------|----------|------------|---------|----------|----------|-------------|
| Server | 7 | 4 | 5 | 6 | 1 | N/A | N/A (low freq) |
| API Path | 10 | 0 | 0 | 4 | 0 | N/A | Planned |
| Auth | 4 | 0 | 0 | 2 | 0 | N/A | N/A (low freq) |
| Node/Agent | 9 | 2 | 2 | 2 | 0 | ❌ | Heartbeat, poll need summary |
| GPU | 6 | 1 | 1 | 0 | 0 | ❌ | Metrics need summary |
| CRUD | 6 | 1 | 1 | 2 | 1 | N/A | N/A (low freq) |
| RunPlan | 1 | 1 | 1 | 0 | 1 | N/A | |
| Deploy/Instance | 4 | 4 | 4 | 0 | 4 | ❌ | |
| Agent Task | 5 | 1 | 0 | 0 | 0 | ❌ | |
| Docker | 5 | 1 | 0 | 0 | 0 | ❌ | |
| Health | 5 | 0 | 0 | 0 | 0 | ❌ | Retries: DEBUG only |
| E2E | 5 | 3 | 3 | 2 | 0 | ✅ | |

## Top Remaining Gaps (2026-06-17)

1. **Health check module not implemented** — Technical blocker: module must be built first. No wait/retry/health check logging possible until module exists. Product gap: Docker start success treated as "running" with no post-start verification. Minimal next: `internal/agent/runtime/health.go`.
2. **StateTransition not wired for node/GPU/lease states** — Helper exists and is called for instance state changes. Node online/offline, GPU health, lease reserve/activate/release still use plain Info logs. No technical blocker. Minimal next: add `log.StateTransition` calls in `MarkOfflineNodes`, `HandleHeartbeat`, lease operations.
3. **CRUD write operations lack started/completed/duration** — BackendRuntime, ModelArtifact, RBAC handlers have error logs only. No technical blocker. Minimal next: wrap write handlers with `defer log.OperationCompleted`.
4. **Wait helpers not wired into actual wait loops** — No active polling loops exist in current code. Task claim is piggybacked on heartbeat; Docker start is synchronous. Wait helpers are available for future use.
5. **GPU lease lifecycle not logged** — Lease reserve/activate/release are inline DB operations without lifecycle logging. No technical blocker. Minimal next: add logging around lease INSERT/UPDATE statements.
6. **StartOperation/OperationCompleted not used in deployment handlers** — Existing handlers use equivalent manual logging with operation_id and duration. Mechanical refactor, not functionally blocking.
7. **Diagnose script diff table is static** — Based on known configs rather than live API query. Live diff requires extracting RunPlan from `resolved_run_plans` table (no direct API endpoint).
