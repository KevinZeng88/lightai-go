# Stability, Reliability, and Observability Review

## Strengths

- Agent task claim uses a transaction and conditional update, reducing double-claim races.
- Task result handling ignores stale terminal results and validates lease owner when present.
- Start path writes instance, run plan, GPU leases, and agent task in one transaction.
- Docker create/start failures preserve container ID and diagnostic previews.
- Stop treats missing containers as stopped, improving idempotency.
- Operation IDs are generated for deployment start/stop and flow into logs/task payload.
- `/metrics` reads snapshots rather than triggering GPU vendor tools.

## Risks

| Area | Finding | Evidence | Impact |
| --- | --- | --- | --- |
| Task timeout cleanup | Server main says sweep loop removed; timeout sweep happens inside heartbeat claim. | `cmd/server/main.go` comment; `sweepExpiredTasks()` called by `claimAndReturnTasks`. | If node stops heartbeating, cleanup depends on node health/offline path and may not cover all states uniformly. |
| GPU lease conflict | Start inserts reserved leases but does not show a robust pre-insert uniqueness conflict strategy in reviewed path. | `HandleStartDeployment` loops accelerator IDs and inserts leases. | Concurrent starts may over-reserve without stronger transactional availability check/index. |
| Preflight divergence | UI preflight can be green before dry-run/start resolver fails. | `preflight_handlers.go` vs `preflightDeployment`. | Bad UX and unreliable API-first validation. |
| Log volume | Many logs are helpful, but Docker inspect/check, heartbeat, resource reports can be frequent. | Agent loop default heartbeat 2s, collect 5s. | Larger node counts may produce noise and DB write pressure. |
| Deployment deletion | Delete stops rows and deletes tasks/instances/runplans synchronously. | `HandleDeleteDeployment`. | If a real container is still running, DB deletion can lose cleanup context. |

## Observability gaps

- Prometheus/Grafana local scripts exist, but server-managed supervision is not implemented in Go.
- Instance health is endpoint-based; no first-class runtime event stream exists.
- The current UI refresh model is mostly manual/on-load in many pages; `useAutoRefresh` exists but is not obviously adopted everywhere.

## Recommended reliability gates

1. Concurrent start test for same GPU and same host port.
2. Node-offline during in-progress start test.
3. Agent restart after claimed task test.
4. Delete deployment with live container test.
5. Stop failed/starting instance cleanup test.
