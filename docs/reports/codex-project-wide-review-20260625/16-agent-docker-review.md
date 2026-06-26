# Agent Docker Review

## Strengths

- Agent exposes `/metrics` from a cached snapshot and does not run GPU tools on scrape.
- DockerRuntimeDriver consumes `AgentRunSpec` and is decoupled behind `DockerClient`.
- Docker create/start/inspect/logs/stop/remove are wrapped and tested with fake client.
- Failure diagnostics preserve container ID, state, exit code, stdout/stderr tail preview.
- File browser/model scanner performs final Agent-side path validation.

## Risks

| Finding | Evidence | Impact | Recommendation |
| --- | --- | --- | --- |
| Agent management endpoints share metrics server and global token. | `cmd/agent/main.go` registers `/docker-images`, `/docker-image-inspect`, `/files`, `/model-paths/scan` on metrics server with bearer check. | Token leak exposes operational data and filesystem browsing within allowed roots. | Per-node token, TLS/mTLS, bind controls, route audit. |
| `collectors.report_interval` and `metrics.advertise_addr` are documented but not implemented. | Agent logs warnings on startup. | Config/docs mismatch can confuse operators. | Either implement or remove from default docs. |
| Docker security options pass through. | `docker_real.go` maps privileged, network, devices, security options. | Strong power with weak policy. | Add server policy gate and warnings. |
| Real Docker smoke not run in this review. | Validation log. | Cannot claim current machine runtime health from this audit. | Keep hardware smoke separate and required before release claims. |

## Agent reliability notes

The agent task loop processes tasks concurrently with a semaphore and keeps heartbeat independent from task execution. This is a good direction. Remaining hardening should focus on crash/restart reconciliation, orphan container cleanup, and idempotent task result retries.
