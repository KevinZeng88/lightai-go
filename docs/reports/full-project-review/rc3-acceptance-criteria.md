# RC3 Acceptance Criteria

## 1. Universal Acceptance Rules

Every issue must satisfy all of the following before closure:

1. The finding is reconfirmed or proven not reproducible.
2. Required action is implemented or explicitly blocked by a permitted final status.
3. Tests or verification scenarios are executed.
4. Actual result and evidence are written to `rc3-verification-matrix.md`.
5. Status and commit are written to `rc3-issue-closeout-register.md`.
6. Related docs are updated.

Allowed final statuses:
- Fixed
- Not Reproducible
- Blocked - External Hardware
- Blocked - Explicit Product Decision

Disallowed final statuses:
- Open
- In Progress
- Not Verified
- Deferred
- Later
- Accepted Risk

## 2. Clean Baseline Criteria

Pass only if:

- Old `/runtime-environments` and `/run-templates` are absent from active API docs and operator instructions.
- RuntimeEnvironment and RunTemplate are absent from active Web routes/pages/client methods.
- OpenAPI exposes only current active routes.
- Fresh DB does not create obsolete runtime tables.
- Current E2E uses ModelArtifact -> BackendRuntime -> ModelDeployment -> RunPlan.
- `rg` scan results are documented and all active old-model remnants are removed.

## 3. Security Criteria

Pass only if:

- Release/non-dev mode refuses empty/default Agent token.
- Install/init flow generates or requires a secure Agent token.
- Agent token rotation is documented or scripted.
- GPU detail direct-ID endpoint is tenant scoped.
- Node transfer updates GPU tenant ownership atomically.
- Audit logs store tenant_id and query by resource tenant.
- Release observability defaults are not exposed insecurely.
- TLS/reverse proxy guidance exists.
- Privileged runtime profiles are explicit and documented.

## 4. Runtime Criteria

Pass only if:

- Dry-run preview, AgentRunSpec, and Docker create options are equivalent.
- Vendor/runtime/image/entrypoint/cmd/args/env/ports/volumes/devices/GPU IDs/health check are preserved.
- NVIDIA DeviceRequests are generated when expected.
- Deployment create/update validates referenced model/runtime/node/GPU.
- Start/health/stop Docker E2E passes.
- Missing-container stop is idempotent and releases lease.
- Instance states use canonical values; `error` is not used as actual_state.

## 5. Task and Reconciliation Criteria

Pass only if:

- Agent task claim uses lease_owner, lease_expires_at, operation_id, generation, attempt, max_attempts or equivalent semantics.
- Concurrent claims cannot duplicate a task.
- Stale, duplicate, or old-generation results cannot corrupt state.
- Agent startup and periodic reconciliation detect managed containers.
- Manual container removal and Agent restart converge to canonical states.

## 6. Database Criteria

Pass only if:

- Current schema is centralized and errors are checked.
- Fresh DB initialization passes in disposable environment.
- Schema initialization is idempotent.
- Old DB compatibility is intentionally removed and documented.
- Current fresh DB contains no obsolete runtime tables.
- `audit_logs.tenant_id` exists and is used.
- GPU collected_at and reported_at semantics are separate.

## 7. Observability and Logging Criteria

Pass only if:

- Bundled/external/disabled observability mode is documented and implemented consistently.
- `report_interval` and `metrics.advertise_addr` are implemented, removed, or clearly warned/errored.
- Default logs are low-noise.
- Successful `/metrics`, `/metrics/targets`, health, ready, assets, and static requests are not repeatedly logged at INFO.
- Stable heartbeat/task-poll/GPU metrics success is not repeatedly logged at INFO.
- Errors, recoveries, task claims, runtime failures, and state changes remain visible.
- Debug/full access logging can be enabled.
- A 10-minute stable run produces acceptable log noise.
- A representative failure remains clearly visible.

## 8. start-all.sh Criteria

Pass only if:

- `scripts/start-all.sh` exists.
- It supports:
  - no args
  - `--dry-run`
  - `--no-observability`
  - `--wait`
- It works in source tree and disposable release directory, or clearly detects unsupported mode.
- It is idempotent and does not duplicate already running processes.
- It does not delete or overwrite data, credentials, DB, config, runtime, or logs.
- It prints PID, log path, and listening address.
- `--wait` health checks server, agent, and enabled observability endpoints.
- `stop-all.sh` stops processes started by `start-all.sh`.

## 9. Web/i18n Criteria

Pass only if:

- `cd web && npm test` runs and passes.
- `cd web && npm run build` passes.
- zh-CN and en-US have complete core locale coverage.
- No raw i18n key appears for core navigation and model/runtime pages.
- The following keys are fixed:
  - nav.models
  - nav.runtime
  - artifacts.name
  - artifacts.path
  - artifacts.format
  - artifacts.taskType
  - artifacts.architecture
  - artifacts.size
  - artifacts.quantization
- Model artifact fields support recommended options plus custom input.
- Long GPU names, model paths, and image names are displayed usefully.
- Loading, empty, and error states exist.
- Web workflow checklist is complete and contains no Not Verified.

## 10. Documentation / OpenAPI Criteria

Pass only if:

- OpenAPI reflects the current `/api/v1` route surface.
- Route-vs-OpenAPI check exists.
- Ops/testing docs use the current BackendRuntime/RunPlan model.
- Old docs are removed, rewritten, or marked obsolete.
- Version references are consistent.
- start-all/stop-all and logging strategy are documented.
- Commands are executable in disposable environments.

## 11. E2E / Release / Patch Criteria

Pass only if disposable validation covers:

- Release package build.
- Release install smoke.
- Fresh DB startup.
- Initial credentials.
- Web login.
- Server health.
- start-all/stop-all.
- Agent registration.
- NVIDIA GPU discovery.
- ModelArtifact create.
- BackendRuntime create.
- Deployment create.
- Dry-run preview.
- Start instance.
- Endpoint health.
- Stop instance.
- Lease release.
- Missing-container stop idempotency.
- Agent restart reconciliation.
- Prometheus/Grafana smoke.
- Patch package build.
- Patch apply.
- Patch rollback.
- Log collection.
- Password reset.
- Tenant direct-ID isolation smoke.
- Audit tenant scoping smoke.
- Web i18n smoke.
- Logging noise/recovery/debug mode smoke.
