> Status: CURRENT_REPORT
> Last reviewed: 2026-06-18
> Scope: Current report or evidence summary
> Read order: See `docs/CURRENT.md`

# Backend Runtime RunPlan Review

**Date:** 2026-06-18
**Reviewer:** Claude (independent review)
**Branch:** `phase-3-runtime-observability-closeout`
**Commits Reviewed:** `d99e7ee` → `89b90e0`
**Review Type:** Code review only (no code modifications)
**Status:** Findings addressed in follow-up commit (see below).

---

## 1. Review Scope

This review covers the BackendRuntime / RunPlan / Docker logs / Web UI work delivered in commits `d99e7ee` and `89b90e0` on branch `phase-3-runtime-observability-closeout`.

**Files in scope (51 files changed in d99e7ee, 14 in 89b90e0):**
- `configs/backend-catalog/**` — Backend catalog YAML (backends, versions, runtimes)
- `configs/backend-catalog.d/**` — Custom override directories
- `internal/server/db/db.go` — Schema migration V13, catalog seed, legacy ID normalization
- `internal/server/api/router.go` — Target API paths
- `internal/server/api/deployment_lifecycle_handlers.go` — Start/stop/logs/cleanup
- `internal/server/api/runtime_handlers.go` — NodeBackendRuntime enable/check
- `internal/server/api/artifact_handlers.go` — ModelLocation CRUD
- `internal/server/api/agent_handlers.go` — Task result processing
- `internal/server/runplan/resolver.go` — Run plan resolution
- `internal/server/runplan/metax_huawei_test.go` — MetaX/Huawei resolver tests
- `internal/agent/runtime/docker.go` — Docker container lifecycle
- `internal/agent/runtime/driver.go` — Runtime driver interface
- `internal/agent/runtime/runplan_adapter.go` — Server→Agent spec conversion
- `web/src/pages/BackendRuntimesPage.vue` — Runtime parameter editor
- `web/src/pages/ModelInstancesPage.vue` — Docker logs drawer
- `web/src/locales/zh-CN.ts`, `web/src/locales/en-US.ts` — i18n keys
- `scripts/e2e-backend-runtime-nvidia-api.sh` — NVIDIA API E2E
- `docs/backend-catalog-vendor-extension.md` — Vendor extension docs

---

## 2. Executive Summary

**Overall Judgment: ACCEPTED_WITH_MINOR_ISSUES**

The implementation is substantially complete and correct for the current phase scope. The critical E2E path — Backend → BackendVersion → BackendRuntime → NodeBackendRuntime → ModelLocation → Deployment → RunPlan → Agent Docker start → health check `/v1/models` → Docker logs → stop → cleanup — is implemented and tested.

Key claims in the acceptance report are verified:
- NVIDIA BackendRuntime API E2E: **PASS** (script covers full lifecycle)
- Docker logs blocker: **FIXED** (agent-proxied with redaction)
- Web i18n: **PASS** (407 consistent keys, no leaks detected)
- Cleanup: **VERIFIED** (FK-safe order, tx-protected)
- Huawei template_only: **CORRECT** (blocked at NodeBackendRuntime gate)
- MetaX: Template correct, readiness gate conservative (requires_hardware_validation)

No critical or high-severity issues found. Four medium-severity findings and several low/info items documented below. None block acceptance.

**Recommendation:** Accept `89b90e0` as the phase completion point. Proceed to MetaX hardware validation when hardware is available.

---

## 3. What Appears Correct

### 3.1 Backend Catalog
- Stable target IDs (`backend.vllm`, `backend-version.vllm.openai-latest`, `runtime.vllm.nvidia-docker`) properly seeded
- Legacy ID normalization (`backend-vllm` → `backend.vllm`) via `normalizeLegacyBackendCatalogIDs()`
- Ollama backend/version/runtime present
- Catalog metadata columns (`slug`, `managed_by`, `source`, `catalog_version`, `checksum`, `status`, `verification_json`) added
- System-managed runtimes have `is_editable=0`, `managed_by=system`

### 3.2 ModelArtifact / ModelLocation
- `model_locations` table exists with node-scoped fields (`node_id`, `path_type`, `model_root`, `relative_path`, `absolute_path`, `match_status`, `verification_status`)
- `HandleCreateModelLocation`, `HandleRescanModelLocation`, `HandleAttestModelLocation` API endpoints implemented
- Resolver uses `ArtifactInfo.ModelRoot` + `RelativePath` from ModelLocation, not old `ArtifactInfo.Path`
- `HandleDeleteArtifact` checks deployment references before deletion, cleans up model_locations in transaction

### 3.3 NodeBackendRuntime
- `node_backend_runtimes` table with proper status/reason fields
- `HandleListNodeBackendRuntimes`, `HandleEnableNodeBackendRuntime`, `HandleCheckNodeBackendRuntime` API endpoints
- `evaluateNodeBackendRuntime()` correctly:
  - Returns `template_only` for `vendor == "huawei"` or `vendor == "ascend"`
  - Checks GPU device existence per vendor
  - Returns `missing_image` when image is not present
  - Returns `ready` only when Docker and image evidence are confirmed
- `HandleStartDeployment` gates on `node_runtime_status == "ready"`

### 3.4 RunPlan Resolver
- `vendor_visible_devices` template variable correctly maps to `ASSIGNED_GPU_INDEXES`
- `defaultVisibleEnvKey()` returns `CUDA_VISIBLE_DEVICES` for NVIDIA/MetaX, `ASCEND_VISIBLE_DEVICES` for Huawei/Ascend
- MetaX Docker options (privileged, ipc=host, uts=host, shm_size=100gb, security_opt, group_add, ulimits) flow through Runtime → RunPlan → Agent payload
- GPU device IDs use index-based values (not DB UUIDs)
- `deduplicateArgs()` removes duplicate flag-value pairs
- Input hash and plan hash computed
- MetaX/Huawei tests pass with verified option coverage

### 3.5 Agent DockerExecutor
- `DockerRuntimeDriver.buildCreateOptions()` maps all Docker options correctly
- NVIDIA GPUs use `DeviceRequest` with `driver="nvidia"` (proper Docker API, not raw devices)
- MetaX/non-NVIDIA GPUs use raw device passthrough
- Container name deterministic: `lightai-{first_12_chars_of_instance_id}`
- Post-start container state verification catches exited(1) containers
- Health check integration via `CheckEndpointReady`
- Stop is idempotent (missing container treated as already stopped)
- `logContainerFailure` provides diagnostic logs on failure

### 3.6 Docker Logs
- `HandleGetNodeRunPlanLogs` resolves run plan → instance/node/container, validates node status, dispatches `model_instance_logs` Agent task, waits for result, redacts sensitive env values
- Offline node returns `503 Service Unavailable` with clear message
- Tenant scope check on logs endpoint
- Tail parameter validated (positive integer, max 5000)
- `since` parameter supported
- Redaction regex catches `TOKEN|SECRET|PASSWORD|PASSWD|API_KEY|SESSION|CSRF` env patterns
- Test `TestNodeRunPlanLogsProxiesThroughAgentTask` verifies end-to-end: agent task completion → logs with redaction
- Test `TestNodeRunPlanLogsRejectsOfflineNode` verifies offline node rejection

### 3.7 Web UI
- `ModelInstancesPage.vue`: Docker logs drawer with tail selector, refresh, copy, auto-open for failed instances
- `BackendRuntimesPage.vue`: Enabled-block design for high-risk scalar options and list textareas
- Custom Args / Custom Env / Custom Docker Options areas
- System-managed runtime readonly enforcement (save button disabled, edit rejection in handler)
- Command preview computed from enabled options only

### 3.8 i18n
- 407 leaf keys in both zh-CN and en-US — exactly matched
- 360 i18n key references in Vue/TS templates all resolve to displayable strings
- Object-value key detection works (parent key `t('nav')` → detected as object)
- `dockerLogs.*` and `runtimes.*` keys present in both locales
- No hardcoded English/Chinese strings found in templates

---

## 4. High-Risk Findings

**None.** No critical or high-severity issues found.

---

## 5. Medium / Low Findings

### Finding BRR-RV-001: Dry-run is shallow, doesn't resolve actual RunPlan
- **Severity:** Medium
- **Evidence:** `internal/server/api/deployment_lifecycle_handlers.go:1084-1151` (`HandleDeploymentDryRun`)
  - Only checks if `model_artifact_id` and `backend_runtime_id` are non-empty
  - Does not call `runplan.Resolve()`, does not check NodeBackendRuntime readiness, does not check ModelLocation existence
- **Impact:** Users get `valid: true` from dry-run but actual `start` may fail with "node backend runtime is not ready" or "model location is not available"
- **Recommendation:** Either run the full resolver in dry-run and return detailed errors/warnings, or rename the endpoint to `validate` and clearly document its limited scope.

### Finding BRR-RV-002: GPU index mapping may not match Docker-visible indices
- **Severity:** Medium
- **Evidence:** `internal/server/api/deployment_lifecycle_handlers.go:469-472` uses `gpu_device_ids` built from `gpu_devices.gpu_index` queries. The `gpu_index` field comes from Agent collector reports, which typically report consecutive indices starting from 0. If the GPU ordering in the database differs from the Docker/NVIDIA toolkit ordering, the wrong GPU could be assigned to the container.
- **Impact:** On multi-GPU nodes with non-standard PCIe ordering, a deployment could be assigned the wrong physical GPU. This is standard NVIDIA index behavior and typically correct, but no validation exists to confirm the index matches what `nvidia-smi` reports.
- **Recommendation:** Add a smoke check comment in the code noting the assumption that `gpu_devices.gpu_index` matches NVIDIA toolkit index ordering. In a future phase, consider Agent-side validation of GPU index mapping before container create.

### Finding BRR-RV-003: E2E script cleanup is not guaranteed on early failure
- **Severity:** Medium
- **Evidence:** `scripts/e2e-backend-runtime-nvidia-api.sh:91-107`
  - `cleanup()` trap only removes the cookie jar
  - `cleanup_e2e_resources()` is called inline at lines 218-221 and by `fail_with_diagnostics()`
  - If the script exits via `fail()` (which does NOT call `cleanup_e2e_resources`), Docker containers and DB resources are NOT cleaned up
  - `fail()` at line 21 calls `exit 1` directly, bypassing cleanup
- **Impact:** Failed E2E runs can leave Docker containers and DB artifacts with `e2e-nvidia-*` prefix
- **Recommendation:** Register a `trap cleanup_e2e_resources EXIT` that calls the cleanup function regardless of exit path. Or unify `fail_with_diagnostics` to be the only failure path.

### Finding BRR-RV-004: Template creation reads from old config path
- **Severity:** Low
- **Evidence:** `internal/server/api/runtime_handlers.go:51` reads from `configs/model-runtime/backend-runtime-templates/` but the catalog is now at `configs/backend-catalog/runtimes/`
- **Impact:** Template-based runtime creation may not find templates that only exist in the new catalog path. Currently the old path may still have files, making this a latent issue.
- **Recommendation:** Update the template reading path to use `configs/backend-catalog/runtimes/` or support both paths.

### Finding BRR-RV-005: DiscoverArtifact doesn't use transaction
- **Severity:** Low
- **Evidence:** `internal/server/api/artifact_handlers.go:216-255` (`HandleDiscoverArtifact`)
  - Inserts into `model_artifacts` and then `model_locations` without a transaction
  - If the location insert fails, an orphan artifact record remains
- **Impact:** Minimal — the orphan artifact is harmless and would be cleaned up on next delete cycle
- **Recommendation:** Wrap both inserts in a transaction.

### Finding BRR-RV-006: NodeBackendRuntime synthetic ID format
- **Severity:** Low
- **Evidence:** `internal/server/api/runtime_handlers.go:243` generates `id := nodeID + ":" + runtimeID`
  - Does not follow UUID convention used by all other tables
  - The UNIQUE constraint on `(node_id, backend_runtime_id)` makes the `id` field purely synthetic
- **Impact:** Inconsistent ID format may confuse API consumers expecting UUIDs
- **Recommendation:** Use `uuid.NewString()` for consistency, or document the synthetic ID format explicitly.

### Finding BRR-RV-007: StartDeployment blocks HTTP handler during agent task wait
- **Severity:** Info
- **Evidence:** `internal/server/api/deployment_lifecycle_handlers.go:623` comment says "Agent will claim this task on next heartbeat" but the start handler returns synchronously after creating the task
  - The stop handler (`HandleStopDeployment`) DOES block via `waitForAgentTaskResult` (up to 90s per instance)
  - The logs handler ALSO blocks via `waitForAgentTaskResult` (up to 30s)
- **Impact:** For the current single-instance deployment model, this is acceptable. Multiple concurrent stop/logs requests could exhaust the HTTP handler pool (default Go `http.Server` has no explicit limit on `ServeMux`, but practical limits apply).
- **Recommendation:** Document this as a known scaling limitation. For future multi-instance deployments, consider async stop with polling from the Web client.

### Finding BRR-RV-008: catalogChecksum is not cryptographic
- **Severity:** Info
- **Evidence:** `internal/server/db/db.go:1314-1323` — uses a simple multiplicative hash
- **Impact:** Function name is misleading. The hash is not collision-resistant, but is only used for change detection (not security), so the impact is minimal.
- **Recommendation:** Rename to `catalogHash` or document that it's a non-cryptographic hash for catalog version tracking only.

---

## 6. Architecture Alignment

| Component | Design Target | Implementation | Alignment |
|-----------|-------------|----------------|-----------|
| Backend | `configs/backend-catalog/backends/` | ✅ Backends for vllm, sglang, llamacpp, ollama | **MATCH** |
| BackendVersion | Separate from BackendRuntime, hardware-independent | ✅ Schema + seed + API | **MATCH** |
| BackendRuntime | Vendor-aware, includes Docker options | ✅ NVIDIA, MetaX, Huawei, CPU variants seeded | **MATCH** |
| NodeBackendRuntime | Node-scoped readiness | ✅ Table, API, evaluation logic | **MATCH** |
| ModelArtifact | Logical model | ✅ CRUD + discover | **MATCH** |
| ModelLocation | Node-scoped model paths | ✅ Table, CRUD, attest | **MATCH** |
| DeploymentPlan | User deploy intent | ✅ `model_deployments` with placement/service/params | **ALIGNED** (table name differs from target) |
| RunPlanGroup | Orchestration group | ✅ `run_plan_groups` table, API | **MATCH** |
| NodeRunPlan | Frozen execution plan | ✅ `resolved_run_plans` with hashes | **ALIGNED** (table name differs from target) |
| Agent DockerExecutor | Structured spec consumption | ✅ `AgentRunSpec` → `ContainerCreateOptions` | **MATCH** |
| Docker logs | Agent-proxied via task | ✅ `model_instance_logs` task with redaction | **MATCH** |
| Web Runtime UI | Enabled-block design | ✅ Scalar toggles + list textareas + custom options | **MATCH** |
| i18n | zh-CN + en-US, no key leaks | ✅ 407 keys, all string-valued | **MATCH** |
| MetaX | Template only until hardware validation | ✅ `verification_status: requires_hardware_validation` | **MATCH** |
| Huawei | Template only until adapter | ✅ `evaluateNodeBackendRuntime` returns `template_only` | **MATCH** |
| GPU lease lifecycle | Reserved → Active → Released | ✅ Start reserves, task result activates, stop releases | **MATCH** |

---

## 7. API Review

### 7.1 Implemented Target APIs (all verified in `router.go`)

| API | Method | Path | Auth | Status |
|-----|--------|------|------|--------|
| List Backends | GET | `/api/v1/backends` | `backend:read` | ✅ |
| Get Backend | GET | `/api/v1/backends/{id}` | `backend:read` | ✅ |
| List Backend Versions | GET | `/api/v1/backends/{id}/versions` | `backend:read` | ✅ |
| List All Versions | GET | `/api/v1/backend-versions` | `backend:read` | ✅ |
| List Runtimes | GET | `/api/v1/backend-runtimes` | `backend_runtime:read` | ✅ |
| Create Runtime | POST | `/api/v1/backend-runtimes/from-template` | `backend_runtime:write` + CSRF | ✅ |
| Get Runtime | GET | `/api/v1/backend-runtimes/{id}` | `backend_runtime:read` | ✅ |
| Patch Runtime | PATCH | `/api/v1/backend-runtimes/{id}` | `backend_runtime:write` + CSRF | ✅ |
| Delete Runtime | DELETE | `/api/v1/backend-runtimes/{id}` | `backend_runtime:write` + CSRF | ✅ |
| List Node Runtimes | GET | `/api/v1/nodes/{id}/backend-runtimes` | `backend_runtime:read` | ✅ |
| Enable Node Runtime | POST | `/api/v1/nodes/{id}/backend-runtimes/enable` | `backend_runtime:write` + CSRF | ✅ |
| Check Node Runtime | POST | `/api/v1/nodes/{id}/backend-runtimes/check` | `backend_runtime:write` + CSRF | ✅ |
| List Artifacts | GET | `/api/v1/model-artifacts` | `model_artifact:read` | ✅ |
| Create Artifact | POST | `/api/v1/model-artifacts` | `model_artifact:write` + CSRF | ✅ |
| Discover Artifact | POST | `/api/v1/model-artifacts/discover` | `model_artifact:write` + CSRF | ✅ |
| Get Artifact | GET | `/api/v1/model-artifacts/{id}` | `model_artifact:read` | ✅ |
| Patch Artifact | PATCH | `/api/v1/model-artifacts/{id}` | `model_artifact:write` + CSRF | ✅ |
| Delete Artifact | DELETE | `/api/v1/model-artifacts/{id}` | `model_artifact:write` + CSRF | ✅ |
| Create Location | POST | `/api/v1/model-artifacts/{id}/locations` | `model_artifact:write` + CSRF | ✅ |
| Rescan Location | POST | `/api/v1/model-artifacts/{id}/locations/{lid}/rescan` | `model_artifact:write` + CSRF | ✅ |
| Attest Location | POST | `/api/v1/model-artifacts/{id}/locations/{lid}/attest` | `model_artifact:write` + CSRF | ✅ |
| List Deployments | GET | `/api/v1/deployments` | `model_deployment:read` | ✅ |
| Create Deployment | POST | `/api/v1/deployments` | `model_deployment:write` + CSRF | ✅ |
| Get Deployment | GET | `/api/v1/deployments/{id}` | `model_deployment:read` | ✅ |
| Patch Deployment | PATCH | `/api/v1/deployments/{id}` | `model_deployment:write` + CSRF | ✅ |
| Delete Deployment | DELETE | `/api/v1/deployments/{id}` | `model_deployment:write` + CSRF | ✅ |
| Dry Run | POST | `/api/v1/deployments/{id}/dry-run` | `model_deployment:write` + CSRF | ✅ (shallow) |
| Start | POST | `/api/v1/deployments/{id}/start` | `model_deployment:start` + CSRF | ✅ |
| Stop | POST | `/api/v1/deployments/{id}/stop` | `model_deployment:stop` + CSRF | ✅ |
| Run Plan Groups | GET | `/api/v1/deployments/{id}/run-plan-groups` | `model_deployment:read` | ✅ |
| List Instances | GET | `/api/v1/model-instances` | `model_instance:read` | ✅ |
| Get Instance | GET | `/api/v1/model-instances/{id}` | `model_instance:read` | ✅ |
| Get Node Run Plan | GET | `/api/v1/node-run-plans/{id}` | `model_instance:read` | ✅ |
| Command Preview | GET | `/api/v1/node-run-plans/{id}/command-preview` | `model_instance:read` | ✅ |
| Logs | GET | `/api/v1/node-run-plans/{id}/logs` | `model_instance:read` | ✅ |
| Task Result | POST | `/api/v1/agent/tasks/{id}/result` | Agent token | ✅ |

### 7.2 Tenant Isolation
- `tenantID(r)` used consistently in list/create operations
- `tenantScopeCheck(r, tid)` used in get/patch/delete operations
- Platform admin can bypass tenant filters (correct for admin dashboard)
- Logs endpoint performs tenant scope check on the resolved `tenant_id`
- Agent task result uses agent token (not session auth) — correct

### 7.3 RBAC Permissions
- `backend:read`, `backend_runtime:read`, `backend_runtime:write` — for catalog/runtime operations
- `model_artifact:read`, `model_artifact:write` — for artifact CRUD
- `model_deployment:read`, `model_deployment:write`, `model_deployment:start`, `model_deployment:stop` — granular lifecycle permissions
- `model_instance:read` — for instance/run-plan/logs read access

### 7.4 CSRF Protection
- All write endpoints (POST/PATCH/DELETE) require CSRF token via `CSRFMiddleware`
- Agent token endpoints excluded from CSRF (correct — agent-auth, not session-auth)

### 7.5 Sensitive Data Handling
- `default_env_json` is redacted in `queryBackendRuntimes` output via `redactRawJSON`
- Docker logs redact sensitive env patterns via `redactDockerLogText`
- Agent task payloads do not log env values (uses `log.RedactEnvKeys`)
- System-managed runtimes protected from direct edit/delete

---

## 8. E2E Review

### 8.1 Script Coverage

The E2E script (`scripts/e2e-backend-runtime-nvidia-api.sh`) covers:
1. ✅ Service startup (builds binaries if needed, starts via `start-all.sh`)
2. ✅ Login + CSRF token acquisition
3. ✅ Node/GPU discovery via API
4. ✅ Backend catalog verification (checks for `backend.vllm`, `backend-version.vllm.openai-latest`)
5. ✅ NodeBackendRuntime enable (`POST /api/v1/nodes/{id}/backend-runtimes/enable`)
6. ✅ ModelArtifact create (`POST /api/v1/model-artifacts`)
7. ✅ ModelLocation create (`POST /api/v1/model-artifacts/{id}/locations`)
8. ✅ Deployment create with placement/service/params (`POST /api/v1/deployments`)
9. ✅ Deployment start (`POST /api/v1/deployments/{id}/start`)
10. ✅ RunPlanGroup query (`GET /api/v1/deployments/{id}/run-plan-groups`)
11. ✅ NodeRunPlan query (`GET /api/v1/node-run-plans/{id}`)
12. ✅ Command preview query (`GET /api/v1/node-run-plans/{id}/command-preview`)
13. ✅ `/v1/models` health check (polling loop with 240s deadline)
14. ✅ Docker logs API (`GET /api/v1/node-run-plans/{id}/logs?tail=200`)
15. ✅ Stop deployment (`POST /api/v1/deployments/{id}/stop`)
16. ✅ Delete deployment (`DELETE /api/v1/deployments/{id}`)
17. ✅ Delete artifact (`DELETE /api/v1/model-artifacts/{id}`)
18. ✅ Docker container cleanup (`docker rm -f`)

### 8.2 Safety Checks
- Port conflict detection: checks if port is occupied by non-LightAI process
- Image/model pre-check: skips if vLLM image or model path is missing
- Diagnostics on failure: `fail_with_diagnostics` collects command preview, logs API output, and Docker logs
- Resource namespacing: uses `e2e-nvidia-{RUN_ID}` prefix

### 8.3 Gap: Cleanup trap
As noted in Finding BRR-RV-003, the `fail()` function doesn't call cleanup. However, the acceptance report's cleanup verification showed 0 residual resources after the E2E run, confirming that the script's happy path cleanups work.

### 8.4 Overall Assessment
The E2E script is a credible verification of the NVIDIA path. It goes through the complete lifecycle via the API (not bypassing the platform), validates key checkpoints, and cleans up resources. **Trustworthy: YES.**

---

## 9. Web i18n Review

### 9.1 Key Metrics
- zh-CN leaf keys: 407
- en-US leaf keys: 407
- Key references found in templates: 360
- All references resolve to displayable strings: ✅
- Object-value key detection: Active (would catch `t('nav')` returning `[object Object]`)
- Hardcoded key-like patterns: 0

### 9.2 New Keys (this phase)
- `runtimes.*` — 40+ keys for runtime editor UI (titles, options, risk warnings, messages)
- `dockerLogs.*` — 8 keys for logs drawer (title, refresh, copy, errors, meta labels)
- All keys present in both locales with proper translations
- Risk warning strings localized (e.g., `runtimes.privilegedRisk` in both zh-CN and en-US)

### 9.3 Verification
```
npm --prefix web test -- --runInBand
→ 9 tests passed, 0 failed
→ i18n keys consistent: PASS
→ all 360 i18n key references resolve to strings: PASS
→ No hardcoded credentials found: PASS
```

### 9.4 Assessment
No i18n key display leaks detected. The `i18nMissingKeys.test.mjs` test correctly identifies object-valued parent keys and would flag any new template reference that resolves to a non-string value. **Risk of key display leak: LOW.**

---

## 10. Security Review

### 10.1 Docker Privileged / Security Options
- `privileged` option is an explicit toggle in the Runtime editor, marked with risk warning (`runtimes.privilegedRisk`)
- MetaX runtimes have `privileged: true` by default (correct — MetaX requires it)
- NVIDIA runtimes default to `privileged: false`
- `security_opt` (seccomp, apparmor) configurable per runtime
- `/dev/mem` is explicitly excluded from default MetaX devices and flagged as `optional_high_risk_devices`

### 10.2 Logs Sensitive Data Redaction
- `redactDockerLogText` uses regex `(?i)([A-Z0-9_]*(TOKEN|SECRET|PASSWORD|PASSWD|API_KEY|SESSION|CSRF)[A-Z0-9_]*=)[^\s]+` to redact sensitive env values
- Test `TestNodeRunPlanLogsProxiesThroughAgentTask` verifies that `API_KEY=super-secret` is redacted
- `default_env_json` is redacted in `getBackendRuntimeJSON` output
- Agent logs redact env keys via `log.RedactEnvKeys`

### 10.3 Tenant Isolation
- All list endpoints filter by `tenant_id` for non-admin users
- Get/Patch/Delete endpoints verify `tenantScopeCheck(r, existing["tenant_id"])`
- Logs endpoint resolves `tenant_id` from the run plan and checks scope
- Agent task result endpoint uses agent token (not session auth)

### 10.4 RBAC Granularity
- `model_deployment:start` and `model_deployment:stop` are separate permissions
- `model_instance:read` controls access to instance details, run plans, command previews, and logs
- Write operations require CSRF token verification

### 10.5 Resource Cleanup Safety
- Deployment delete uses FK-safe order: stop instances → release leases → cancel tasks → delete run plans → delete run plan groups → delete leases → delete instances → delete deployment
- All operations wrapped in a single transaction
- Artifact delete checks for deployment references before proceeding

---

## 11. Operational Review

### 11.1 Failed State Diagnostics
- Instance state transitions: `pending → running | failed`
- Agent task result updates instance state with container_id, endpoint_url, last_error
- `logContainerFailure` captures container state, exit code, and tail logs on Docker start failure
- Web instance page auto-opens logs drawer for failed instances
- E2E script `fail_with_diagnostics` collects command preview + logs API + Docker logs

### 11.2 Docker Logs Availability
- Must be fetched through running Agent (not Server local Docker)
- Requires node to be `online`
- Timeout: 30 seconds for logs task completion
- Returns stdout, stderr, and merged logs with redaction

### 11.3 Cleanup Reliability
- Stop: dispatches Agent stop task per instance, waits up to 90s, marks instance stopped, releases GPU leases
- Delete: transactional FK-safe cleanup of all dependent records
- GPU lease lifecycle: reserved → active (on start success) or failed (on start failure) → released (on stop)
- Idempotent stop: agent treats missing container as already stopped

### 11.4 Command Preview
- Generated server-side via `EquivalentCommandPreview(plan)` from the resolved RunPlan
- Web Runtime page shows client-side preview (simpler, based on enabled options only)
- E2E verifies server-generated preview contains expected image

### 11.5 Observability
- Structured logging with operation IDs throughout start/stop/logs flow
- State transition logging for instances and GPU leases
- Audit log entries for deployment create/start/stop
- Slow operation detection for Docker create/start/stop

---

## 12. Recommended Next Steps

### P0 (Before next phase merge)
None. No blocking issues found.

### P1 (Recommended before wider use)
1. **Fix dry-run to actually run the resolver** (BRR-RV-001). Currently misleading.
2. **Fix E2E cleanup trap** (BRR-RV-003). Register `trap cleanup_e2e_resources EXIT` instead of relying on inline calls.

### P2 (Nice to have)
1. **Update template reading path** (BRR-RV-004). Read from `configs/backend-catalog/runtimes/`.
2. **Add transaction to HandleDiscoverArtifact** (BRR-RV-005). Wrap artifact + location inserts.
3. **Use UUID for node_backend_runtimes.id** (BRR-RV-006). Consistency with other tables.
4. **Document async task polling limitation** (BRR-RV-007). Note that stop/logs block HTTP handlers.

### MetaX Hardware Validation (when hardware available)
- Enable MetaX runtime on a real MetaX node
- Verify `NodeBackendRuntime` transitions to `ready`
- Start a vLLM deployment with MetaX runtime
- Verify `/v1/models` health check
- Verify Docker logs
- Verify stop + cleanup + lease release

---

## 13. Verification Results

| Command | Result | Notes |
|---------|--------|-------|
| `go test ./...` | ✅ PASS | All packages: ok |
| `go vet ./...` | ✅ PASS | No issues |
| `npm --prefix web run build` | ✅ PASS | Built in 2.87s |
| `npm --prefix web test -- --runInBand` | ✅ PASS | 9 tests, all passed |
| `bash -n scripts/*.sh` | ✅ PASS | All 28 scripts have valid syntax |
| `git diff --check` | ✅ PASS | No whitespace issues |
| `git status --short` | ✅ CLEAN | No uncommitted changes |

---

## 14. Open Questions

1. **GPU index mapping validation:** Is there any scenario where `gpu_devices.gpu_index` differs from the NVIDIA toolkit device index? If so, `gpu_device_ids` in the Agent payload would be incorrect.

2. **Runtime template path migration:** `HandleCreateBackendRuntimeFromTemplate` reads from `configs/model-runtime/backend-runtime-templates/`. Should this be migrated to `configs/backend-catalog/runtimes/`, or is the old path intentionally preserved for backward compatibility?

3. **Multi-GPU deployment:** The current implementation creates a single instance per deployment with auto-assigned single GPU. Is the `replicas` field intended to be used in this phase, or is it reserved for future multi-instance deployments?

4. **Container port conflict:** If two deployments use the same `host_port`, the second Docker start will fail with port conflict. Is there a port allocation mechanism planned, or is manual port selection sufficient for the current phase?

5. **MetaX image digest:** The seeded MetaX runtime image is `0d307f1665d3` (a Docker image ID, not a tag). How is this image built/provided? Is there a documented process?

---

## 15. Final Conclusion

**Recommendation: ACCEPT `89b90e0` as the phase-3-runtime-observability-closeout completion point.**

The implementation is well-structured, test-covered, and follows the target design faithfully. The critical NVIDIA E2E path is verified end-to-end. The MetaX and Huawei paths are correctly templated with conservative readiness gating. The Web UI has proper enabled-block controls, and i18n is leak-free.

No blocking issues found. The four medium-severity findings (shallow dry-run, GPU index assumption, E2E cleanup gap, template path mismatch) are all fixable in follow-up work and do not affect the correctness of the primary NVIDIA deployment path.

**Should enter MetaX real hardware validation:** YES — when MetaX hardware is available. The template, resolver, and readiness gate are all ready for testing.

**Must-fix before next major integration:** None identified.

---

## Follow-up Resolution (2026-06-18)

The following findings were addressed in a follow-up commit:

| Finding | Action | Status |
|---------|--------|--------|
| BRR-RV-001 (shallow dry-run) | Extracted `preflightDeployment()` shared between `HandleDeploymentDryRun` and `HandleStartDeployment`. Dry-run now calls the real RunPlan resolver with full validation. | **FIXED** |
| BRR-RV-003 (E2E cleanup trap) | Added `on_exit` EXIT trap with `EXIT_CODE` tracking, `KEEP_E2E_RESOURCES=1` support, and `failed_stage` output on any failure path. | **FIXED** |
| BRR-RV-004 (template path) | Added `resolveTemplatePath()` that tries `configs/backend-catalog/runtimes/{backend}/{vendor}-docker.yaml` first, falls back to old `configs/model-runtime/` path. | **FIXED** |
| BRR-RV-005 (DiscoverArtifact transaction) | Wrapped artifact + location inserts in `tx.Begin()`/`tx.Commit()`/`defer tx.Rollback()`. | **FIXED** |
| BRR-OBS-001 (full-chain observability) | Added stage timing to `HandleStartDeployment` via `log.StageCompleted`/`log.StageFailed`. E2E script now outputs `stage=login/health_check/cleanup` with `duration_ms`. Agent already had detailed stage logging. | **FIXED** |
| BRR-RV-002 (GPU index mapping) | Retained as documented assumption; needs hardware validation. | DOCUMENTED |
| BRR-RV-006 (synthetic ID) | Retained as compatibility note. | DOCUMENTED |
| BRR-RV-007 (async task polling) | Retained as future scaling note. | DOCUMENTED |

All fixes verified: `go test ./...` passes, `go vet` clean, `npm run build` success, `npm test` all pass (407 i18n keys consistent), `bash -n` valid.
