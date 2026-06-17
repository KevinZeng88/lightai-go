# Log Observability Review

**Date:** 2026-06-18 00:43 UTC
**Verification Dir:** `docs/reports/backend-runtime-runplan/final-verification-20260618-004001/`

## E2E Output

- **E2E result:** FAIL (exit code 1) — health check timeout due to port mismatch (see below)
- **`/v1/models` result:** NOT REACHED — health check used wrong port (8080 instead of 8004)
- **Docker logs API result:** PASS (returned full vLLM startup logs, 12787 bytes stdout, 2960 bytes stderr)
- **Stop / cleanup result:** PASS (docker stop completed in 1839ms, container removed, no residual containers)
- **Stage output complete:** 13/14 stages PASS (`health_check` stage ran but timed out due to port mismatch)
- **`duration_ms` visible:** YES — 14 stage timings in E2E output, 64 in server log, 12 in agent log

### Root cause of E2E failure

The Agent health check used `http://127.0.0.1:8080/v1/models`, but:
- Container port is `8000` (vLLM default)
- Host port is `8004` (from E2E `service_json.host_port`)
- Port `8080` is the llama.cpp default (`backend_versions` seeded with `default_container_port=8080` for llama.cpp, but vLLM uses `8000`)
- The health check `port` in the agent payload appears to default to 8080 rather than using the resolved host_port

This is a code bug, not a test infrastructure issue. The full Docker lifecycle (create, start, logs, stop) worked correctly — only the health check URL was wrong.

## Server Log

- **Covered stages:**
  - `operation_started` / `operation_completed` for `model_artifact.create`, `deployment.start`, `deployment.stop`
  - `stage=preflight` with `duration_ms` (from `log.StageCompleted`) — deployment preflight validation
  - `stage=state_transition` for GPU lease ("" → "reserved"), instance ("pending" → "failed", "failed" → "stopped")
  - `stage=slow_operation` for slow API requests (>1000ms)
  - `agent_task.claimed` (task claimed by agent)
  - `agent_task.created` with task_id, instance_id, deployment_id, generation

- **Missing stages:** Individual preflight sub-steps (resolve_artifact, select_node, fetch_runtime_chain, validate_node_runtime, validate_model_location, resolve_run_plan) are not individually logged with `log.StageCompleted`. They are covered by the single `stage=preflight` entry. Acceptable for this phase.

- **Can correlate deployment_id / instance_id / run_plan_id / node_id / task_id / container_id:** YES
  - `deployment_id=495a17f3-d631-4822-b526-6fd8d036aae5`
  - `instance_id=2fbb7c63-f59d-4303-a525-2957cfaf0641`
  - `run_plan_id=f584f8aa-5aee-4649-b651-e90f3fd30453`
  - `node_id=node-70894186-093c-403d-87d1-08f17a690521`
  - `task_id=ed2e5a4f-d623-4b00-bbd2-b902a380b4b8` (start), `a7b39683-...` (logs), `e3891623-...` (stop)
  - `container_id=c590a4dda73da57910843563913d7e11e34a80259cddfed571d9601e70ab419b`

- **`duration_ms` visible:** YES — 64 instances in server log
- **Slow stage visible:** YES — `stage=slow_operation` for logs API (1003ms > 1000ms threshold)
- **Sensitive information leakage:** NONE — `agent_token=<redacted>` in server log

## Agent Log

- **Agent task claim visible:** YES — `agent_task.execution.started` with task_id, task_type, instance_id
- **Docker create visible:** YES — `docker.create.started` (duration_ms=35) and `docker.create.completed`
- **Docker start visible:** YES — `docker.start.started` and `docker.start.completed` (duration_ms=367)
- **Health check visible:** YES — `health_check.started` with endpoint_url, timeout; `health_check.timeout` with error details (61 attempts, 120s elapsed)
- **Docker logs visible:** YES — `logs task completed` with stdout_bytes=12787, stderr_bytes=2960
- **Docker stop visible:** YES — `docker.stop.started` and `docker.stop.completed` (stop_duration_ms=1839)
- **Container inspect visible:** YES — `docker.container.exited` (state=running at the time of health check failure; container was actually running)
- **`duration_ms` visible:** YES — 12 instances in agent log (docker create/start/stop, task execution)
- **Sensitive information leakage:** NONE — `env_keys` shown but values redacted

## Cleanup Evidence

- **e2e-nvidia-* resources residual:** NONE — deployments/artifacts/instances cleaned up
- **lightai- E2E container residual:** NONE — `docker ps -a --filter name=lightai-` returns "(none)"
- **Stop dispatched:** YES — via `model_instance_stop` task (task_id=e3891623-...)
- **GPU lease released:** Implicit through instance stop

## Overall Chain Visibility

| Stage | Server Log | Agent Log | E2E Output | Correlated IDs |
|-------|-----------|-----------|------------|----------------|
| Deployment create | operation_completed (model_artifact.create) | — | stage=create_deployment duration_ms=35 | deployment_id |
| Deployment start | stage=preflight operation_completed | — | stage=start_deployment duration_ms=37 | deployment_id, instance_id, run_plan_id, task_id |
| GPU lease reserve | state_transition ("", "reserved") | — | — | lease_id, gpu_id, instance_id |
| Agent task create | agent_task.created | — | — | task_id, instance_id |
| Agent task claim | agent_task.claimed | agent_task.execution.started | — | task_id, agent_id |
| Docker create | — | docker.create.started/completed duration_ms=35 | — | container_id, image |
| Docker start | — | docker.start.started/completed duration_ms=367 | — | container_id |
| Health check | — | health_check.started, wait_started, timeout (120s) | stage=health_check (FAILED - port mismatch) | instance_id, container_id, endpoint_url |
| Instance state transition | state_transition (pending→failed) | — | — | instance_id, task_id |
| Docker logs request | operation_started (deployment logs) | logs task completed (12787+2960 bytes) | stage=logs_api | run_plan_id, instance_id, container_id |
| Deployment stop | state_transition (failed→stopped) | docker.stop.started/completed duration_ms=1839 | stage=stop_deployment | instance_id, container_id |
| Cleanup | — | — | stage=cleanup_resources | — |

## Conclusion

**ACCEPT_WITH_LOG_GAPS**

The full Docker lifecycle chain is observable end-to-end with correlated IDs, duration tracking, and state transitions. The E2E script shows stage timing for all 14 stages. The Agent logs are rich and detailed.

**Gap:** The E2E health check failed due to a port mismatch (health check targets port 8080; container serves on port 8000, host-mapped to 8004). This prevents `/v1/models` verification but does not indicate a logging gap — it is a health check configuration bug.

**Gap:** Individual preflight sub-steps within `preflightDeployment` are logged as a single `stage=preflight` rather than individual stages (resolve_artifact, select_node, validate_node_runtime, etc.). This is acceptable for current scale but would aid MetaX debugging if expanded.
