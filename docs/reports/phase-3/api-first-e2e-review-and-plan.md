# API-First E2E Audit & Improvement Plan

> Status: AUDIT
> Date: 2026-06-20
> Scope: All existing tests, E2E scripts, and test coverage gaps
> Principle: API-level E2E is the primary correctness verification; UI smoke is display-layer only

## 0. Relationship To Follow-Up Strategy Documents

This document is the pre-implementation audit input for the API-first E2E redesign. It remains useful as the inventory and gap analysis record.

Follow-up strategy and execution guidance live in:

- `docs/reports/phase-3/e2e-harness-and-api-workflow-strategy.md`
- `docs/reports/phase-3/e2e-implementation-roadmap.md`
- `docs/reports/phase-3/e2e-claude-handoff.md`

Where this audit proposes direct-handler workflow tests or a standalone shell workflow script as immediate options, the follow-up documents refine the implementation order:

1. First add a reusable Go API Workflow harness using the real `SetupRoutes` mux, real login/session/CSRF, test DB, and fake Agent.
2. Then implement the NBR Probe Chain as the first vertical API Workflow slice.
3. Then standardize Shell E2E helpers and migrate existing scripts by tier.

No business code, test implementation, shell helper, API, DB migration, CI, browser automation, Script Probe, Version Probe, Backend Match catalog, or real model container run is part of this design-only documentation round.

## 1. Existing Test Files Inventory

### 1.1 Go Test Files (27 files, ~215 test functions)

| # | File | Tests | Category |
|---|------|-------|----------|
| 1 | `internal/server/api/runtime_boundary_test.go` | 48 | handler/unit + fake-agent HTTP |
| 2 | `internal/server/api/ui_persistence_runplan_test.go` | 22 | handler/unit |
| 3 | `internal/server/api/agent_identity_test.go` | 13 | handler/unit |
| 4 | `internal/server/api/phase3_rbac_test.go` | 6 | handler/unit |
| 5 | `internal/server/api/tenant_isolation_test.go` | 6 | handler/unit |
| 6 | `internal/server/api/resource_handlers_test.go` | 7 | handler/unit |
| 7 | `internal/server/api/agent_task_result_test.go` | 3 | handler/unit |
| 8 | `internal/server/api/model_root_policy_test.go` | 2 | handler/unit |
| 9 | `internal/server/api/node_run_plan_logs_test.go` | 2 | handler/unit |
| 10 | `internal/server/runplan/resolver_test.go` | 21 | pure unit |
| 11 | `internal/server/runplan/vllm_sglang_nvidia_test.go` | 12 | pure unit |
| 12 | `internal/server/runplan/llamacpp_nvidia_test.go` | 2 | pure unit |
| 13 | `internal/server/runplan/metax_huawei_test.go` | 2 | pure unit |
| 14 | `internal/agent/runtime/docker_test.go` | 25 | unit + real container (opt-in) |
| 15 | `internal/agent/runtime/health_test.go` | 7 | pure unit |
| 16 | `internal/agent/runtime/runplan_adapter_test.go` | 2 | pure unit |
| 17 | `internal/agent/collector/nvidia_test.go` | 7 | pure unit |
| 18 | `internal/agent/collector/probe_test.go` | 6 | pure unit (exec scripts) |
| 19 | `internal/agent/collector/protocol_test.go` | 9 | pure unit |
| 20 | `internal/agent/collector/gguf_reader_test.go` | 4 | pure unit |
| 21 | `internal/agent/metrics/metrics_test.go` | 9 | pure unit |
| 22 | `internal/agent/register/register_test.go` | 7 | unit + mock server |
| 23 | `internal/agent/state/state_test.go` | 10 | pure unit |
| 24 | `internal/common/errors/errors_test.go` | 4 | pure unit |
| 25 | `internal/common/token/bootstrap_test.go` | 6 | pure unit |
| 26 | `internal/common/version/version_test.go` | 2 | pure unit |
| 27 | `cmd/agent/main_test.go` | 3 | pure unit |

### 1.2 Web Test Files (9 files)

| # | File | Type | Checks |
|---|------|------|--------|
| 28 | `web/tests/i18nKeys.test.mjs` | UI static | zh-CN ↔ en-US key parity |
| 29 | `web/tests/i18nMissingKeys.test.mjs` | UI static | All `$t()` references resolve |
| 30 | `web/tests/formatters.test.mjs` | UI static | Time formatting (zh-CN + en-US) |
| 31 | `web/tests/runtimeBoundaryUi.test.mjs` | UI static | Component existence, labels, actions |
| 32 | `web/tests/apiClientPaths.test.mjs` | UI static | API paths use relative format |
| 33 | `web/tests/noHardcodedCredentials.test.mjs` | UI static | Security scan |
| 34 | `web/src/pages/__tests__/dashboard.test.ts` | UI unit | GPU aggregation functions |
| 35 | `web/src/composables/__tests__/useAutoRefresh.test.ts` | UI unit | Composable logic |
| 36 | `web/src/stores/__tests__/auth.test.ts` | UI unit | Auth store with mocked API |

### 1.3 E2E Shell Scripts (22 scripts)

| # | Script | Tests |
|---|--------|-------|
| 37 | `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh` | Full vLLM wizard → container |
| 38 | `scripts/e2e-model-runtime-wizard-nvidia-sglang.sh` | Full SGLang wizard → container |
| 39 | `scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh` | Full llama.cpp wizard → container |
| 40 | `scripts/e2e-model-runtime-wizard-nvidia-matrix.sh` | Three-backend matrix |
| 41 | `scripts/e2e-model-runtime-wizard-nvidia-api.sh` | API-only wizard E2E |
| 42 | `scripts/e2e-model-runtime-api.sh` | Generic model runtime API |
| 43 | `scripts/e2e-model-runtime-local.sh` | Full local lifecycle |
| 44 | `scripts/e2e-model-runtime-failed-instance-logs.sh` | Failed instance logs |
| 45 | `scripts/e2e-real-smoke-all-three.sh` | Real container smoke (3 backends) |
| 46 | `scripts/e2e-instance-stop-real-llamacpp.sh` | Real llama.cpp start/stop |
| 47 | `scripts/e2e-dryrun-parameter-matrix-enhanced.sh` | DryRun parameter matrix |
| 48 | `scripts/e2e-matrix-verifier.sh` | Cross-backend param matrix |
| 49 | `scripts/e2e-inference-parser-llamacpp.sh` | Inference parser |
| 50 | `scripts/e2e-runplan-parameter-source-audit.sh` | RunPlan parameter propagation |
| 51 | `scripts/e2e-backend-runtime-nvidia-api.sh` | BackendRuntime NVIDIA API |
| 52 | `scripts/e2e-clone-template-parameter-persistence.sh` | Clone template persistence |
| 53 | `scripts/e2e-deployment-visibility-selected.sh` | Deployment lifecycle API |
| 54 | `scripts/e2e-runtime-config-copy-first-save-selection.sh` | Clone first-save |
| 55 | `scripts/e2e-runtime-config-web-check-flow.sh` | Check-request flow |
| 56 | `scripts/e2e-ui-persistence-runplan-selected.sh` | UI persistence RunPlan |
| 57 | `scripts/smoke-model-backends.sh` | Backend smoke |
| 58 | `scripts/verify-local.sh` | Local verification |

## 2. Test Category Classification

### 2.1 handler/unit (direct handler calls, in-memory SQLite)

**Coverage**: All `internal/server/api/*_test.go` files except as noted.

**Characteristics**:
- Uses `httptest.NewRecorder` + `newReq()` with `SetPathValue`
- In-memory SQLite (`db.Open(":memory:")`)
- No real HTTP router (no `http.ServeMux`)
- No real network calls (except `httptest.NewServer` fake agents)

**Gap**: PathValue parameter names are set manually. A route pattern `{id}` → handler `PathValue("node_id")` mismatch would NOT be caught. The existing regression tests (`TestCheckRequestEndpointPathValuesCorrect`, `TestProbeEndpointPathValuesCorrect`) catch this by asserting the handler returns 200 (not 400), which is an indirect detection method.

### 2.2 real HTTP router (app mux with real routes)

**Coverage**: **ZERO tests.** No test instantiates the real `http.ServeMux` with `SetupRoutes()` or equivalent.

**Impact**: Route parameter name mismatches, middleware chains, and routing priority issues are only caught by:
1. Indirect PathValue tests (handler returns 400 if path param name is wrong)
2. E2E shell scripts (real server process)
3. Manual testing

**Risk**: High for new routes. Mitigated by PathValue regression tests.

### 2.3 API workflow E2E (list → select → create → probe → verify)

**Coverage**: **ZERO handler/unit tests.** No single Go test simulates a complete user workflow across multiple API calls.

**E2E script coverage**: The 22 shell scripts DO cover complete workflows, but they:
- Require running server + agent + Docker
- Require real GPU for container smoke
- Are not run in CI
- Are not idempotent (leave DB state, containers)

### 2.4 real Agent smoke (running agent, real Docker daemon)

**Coverage**: E2E scripts (`e2e-model-runtime-*.sh`, `e2e-real-smoke-*.sh`) and the opt-in `TestRealDockerRuntimeDriver` (requires `LIGHTAI_TEST_DOCKER=1`).

### 2.5 real container smoke (Docker containers with model serving)

**Coverage**: E2E scripts only. The Go test `TestRealDockerRuntimeDriver` starts `alpine:latest`, not a model-serving container.

### 2.6 UI static/component (build, i18n, component existence)

**Coverage**: All 9 web test files + `npm run build`. Comprehensive for static checks.

### 2.7 manual UI (browser interaction)

**Coverage**: The Layer 4 Web Smoke from the validation plan. Currently PENDING.

## 3. Gap Analysis by Business Flow

### A. BackendRuntime / Runtime Wizard

| API Step | Handler Test? | Real Router? | Workflow E2E? | Script? |
|----------|--------------|-------------|---------------|---------|
| List Backends | `TestBackendListReadOnly` | No | No | `smoke-model-backends.sh` |
| List BackendVersions | `TestBackendVersionList` | No | No | — |
| List BackendRuntimes | `TestBackendRuntimeCRUD` | No | No | — |
| Clone system → user | `TestCreateFromTemplate` | No | No | `e2e-clone-template-*.sh` |
| PATCH runtime fields | `TestCreateAndPatchBackendRuntimeNamePersistence` | No | No | — |
| GET runtime after PATCH | Same test | No | No | — |
| DELETE runtime | `TestBackendRuntimeCRUD` | No | No | — |
| Reload catalog | `TestBackendCatalogReload*` | No | No | — |
| config_snapshot_json frozen | `TestBackendRuntimeEditDoesNotAffectNBRConfig` | No | No | — |

**Key gap**: No single test does clone → modify → save → GET → verify-fields → delete. The existing tests verify individual steps.

### B. NodeBackendRuntime / NBR Wizard

| API Step | Handler Test? | Real Router? | Workflow E2E? |
|----------|--------------|-------------|---------------|
| List nodes | `TestNodeListReturnsNewFields` | No | No |
| List docker images | No (proxy handler, tested via fake agent in check-request tests) | No | No |
| Create NBR (enable) | `TestNodeBackendRuntimeCopiesTemplateSnapshot*` | No | No |
| POST /probe (check) | `TestPostProbeMissingImageOnlyFromInspectNotFound` | No | No |
| GET /probe | `TestGetProbeReturnsSnapshotAfterProbe` | No | No |
| List NBRs (by node) | `TestNodeBackendRuntimeDisplayNamePersistence` | No | No |
| Patch NBR | `TestPatchNodeBackendRuntimeSnapshotFieldsNeedRecheck` | No | No |
| Delete NBR | `TestDeployment*` tests exercise delete | No | No |
| check-request compat | `TestCheckRequestBackwardCompatible` | No | No |
| missing_image only from inspect not-found | 6 dedicated tests | No | No |

**Key gap**: No single test does list-nodes → list-images → create-NBR → POST-probe → GET-probe → verify-fields → list-verify. The pieces are all tested individually.

### C. Model Wizard / ModelArtifact / ModelLocation

| API Step | Handler Test? | Real Router? | Workflow E2E? |
|----------|--------------|-------------|---------------|
| Browse node files | No (proxy) | No | `e2e-model-runtime-*.sh` |
| Scan model path | No (proxy) | No | `e2e-model-runtime-*.sh` |
| Create ModelArtifact | `TestModelArtifactDisplayNamePersistence*` | No | `e2e-model-runtime-*.sh` |
| Add ModelLocation | No dedicated test | No | — |
| List locations | No dedicated test | No | `e2e-model-runtime-*.sh` |
| Delete location | No dedicated test | No | — |

**Key gap**: Model wizard is the least tested flow. Most coverage comes from E2E shell scripts.

### D. Deployment / Start Wizard

| API Step | Handler Test? | Real Router? | Workflow E2E? |
|----------|--------------|-------------|---------------|
| Create deployment (save) | `TestDeploymentSaveOnlyAndPatchEditableFields` | No | `e2e-deployment-visibility-*.sh` |
| Preflight | `TestPreflightDeploymentFailsWhenNoNBRExists` | No | `e2e-model-runtime-*.sh` |
| Generate RunPlan (dry-run) | `TestNBRConfigModificationDoesNotAffectDeploymentDryRun` | No | `e2e-dryrun-parameter-matrix-*.sh` |
| Start deployment | `TestDeploymentStartUsesNBRNotBackendRuntime` | No | `e2e-model-runtime-*.sh` |
| Get instance status | `TestModelInstanceFailureKeepsContainerID*` | No | `e2e-model-runtime-*.sh` |
| Get logs | `TestNodeRunPlanLogsProxiesThroughAgentTask` | No | — |
| Stop deployment | Docker test `TestDockerRuntimeDriverStop` | No | `e2e-model-runtime-*.sh` |
| Cleanup | Docker test `TestRealDockerRuntimeDriver` (opt-in) | No | `e2e-model-runtime-*.sh` |
| NBR snapshot freeze | `TestRunPlanImmutableAfterDeploymentEdit` (partial) | No | — |

**Key gap**: No single test covers create → preflight → dry-run → start → status → logs → stop → cleanup. The E2E scripts cover this but require a real environment.

### E. Agent / Docker

| API Step | Test? |
|----------|-------|
| /docker-images | Agent unit: `execCmd` parsing tested indirectly via check-request tests |
| /docker-image-inspect | Agent unit: same execCmd pattern, tested via fake agents |
| /files | No dedicated test |
| /model-paths/scan | No dedicated test |
| Docker start | `TestDockerRuntimeDriverStart*` (unit) + `TestRealDockerRuntimeDriver` (opt-in real) |
| Docker logs | `TestDockerRuntimeDriverLogs` (unit) |
| Docker stop | `TestDockerRuntimeDriverStop` (unit) + real test |
| Docker error mapping | `TestStartPostCreateContainerExitReturnsDiagnostics` |
| Health timeout | `TestStartHealthCheckFailureReturnsDiagnostics` |

**Key gap**: `/files` and `/model-paths/scan` have no dedicated handler tests. Docker error → status mapping is well covered.

### F. Audit / Logs / Observability

| Check | Test? |
|-------|-------|
| operation_id in logs | No dedicated test (present in handler code) |
| node_id/agent_id in log | Partially — `TestHandleTaskResultStartSuccessStoresStateAndAudit` |
| audit_logs records | `TestHandleTaskResultStartSuccessStoresStateAndAudit` |
| instance_logs readable | `TestModelInstanceFailedStateAllowsLogAccess` |
| stop/failed/cleanup tracked | `TestHandleTaskResultStartFailureStoresDiagnosticsAndAudit` |

**Key gap**: operation_id propagation across the full probe chain is not verified.

## 4. Critical Test Pattern Gaps

### 4.1 No Real HTTP Router Tests

**Gap**: All 215+ Go tests call handlers directly. No test wires up `SetupRoutes()` and sends HTTP requests through the real mux.

**Severity**: MEDIUM. Mitigated by PathValue regression tests + E2E shell scripts. However, middleware chain errors (auth, CSRF, CORS, rate limiting) are completely untested.

### 4.2 No Client-Trusted Payload Detection

**Gap**: Handler/unit tests bypass auth middleware by setting `adminSession()` in the request context. Tests can pass `image_present=true` in the request body without the handler rejecting it.

**Severity**: LOW for current code (the check-request handler correctly ignores client-provided image_present). But no test verifies this rejection behavior.

### 4.3 No List → Select → Save → Detail Data Consistency Test

**Gap**: No test verifies that data survives a round-trip through the full API chain: create → list (find it) → GET (read it) → PATCH (modify) → GET (verify change) → list (still there) → DELETE (gone) → list (not there).

**Severity**: MEDIUM. Individual CRUD tests cover the steps separately. The `refresh()` bug (dropping `probe_results_json`) would have been caught by such a test.

### 4.4 No API-Level E2E for NBR Probe Chain

**Gap**: The most critical chain for the current phase — list docker images → create NBR → POST probe → GET probe → verify → list NBRs → detail drawer data — has no single end-to-end test.

**Severity**: HIGH. This is the exact chain where the Phase 0 root-cause bug lived (list saw image but check-request reported missing). A combined API workflow test would catch regressions.

### 4.5 No Negative Path for Evidence/Agent/Docker/Inspect Errors

**Coverage is GOOD**. The check-request probe tests explicitly verify 6+ error types never map to `missing_image`.

## 5. Recommendations

### 5.1 UI Smoke Tests That Can Be De-Prioritized

The following manual/browser-based checks are adequately covered by automated layers:

| Manual Check | Equivalent Automated Coverage |
|-------------|-------------------------------|
| "No i18n key leaks" | `i18nMissingKeys.test.mjs` — static scan catches all `$t()` references |
| "Status tag color correct" | `getStatusType()` mapping verified by code review + TypeScript compilation |
| "Probe panels show fields" | Data flow verified by handler tests + TypeScript type checking |
| "Create button enabled for ready/ready_with_warnings" | Code logic in `RunnerConfigsPage.vue` template verified by `runtimeBoundaryUi.test.mjs` |
| "formatBytes shows correct units" | `formatters.test.mjs` tests time formatting; `$t('nodeRuntimeProbe.bytes')` verified by i18n key check |

**Remaining manual-only checks** (require visual DOM verification):
- Drawer actually renders probe panels (not just data flows)
- Diagnostic notices (shell wrapper, vendor image) appear correctly
- Collapsible panels expand/collapse correctly

### 5.2 UI Static/Component Tests That Must Be Retained

All current web tests must be retained — they provide essential safety nets:
- **i18n key consistency** — prevents zh-CN/en-US drift
- **i18n key leaks** — prevents raw key display in UI
- **No hardcoded credentials** — security gate
- **Component existence** — prevents accidental removal of critical UI elements
- **API client paths** — prevents hardcoded absolute URLs

### 5.3 First Batch of API Workflow E2E Tests to Add

Priority order:

**P0 — NBR Probe Chain** (covers the current phase's critical path):
```
TestNBRProbeWorkflowE2E:
  1. List nodes → get node_id
  2. List docker images (fake agent returns vllm/vllm-openai:latest)
  3. Create NBR (enable) with image_ref
  4. POST /probe
  5. GET /probe → verify probe_results_json has level1-4
  6. List NBRs → verify probe_results_json in list item
  7. GET NBR detail → verify same data
  8. POST /probe with non-existent image → verify missing_image
  9. DELETE NBR → verify cleanup
```

**P0 — BackendRuntime CRUD Chain**:
```
TestBackendRuntimeWorkflowE2E:
  1. List backends → get backend_id
  2. List versions → get version_id
  3. Clone system runtime → user runtime
  4. GET runtime → verify fields
  5. PATCH runtime (image_name, args) → verify
  6. GET runtime → verify changes persisted
  7. DELETE runtime → verify gone
```

**P1 — Deployment Lifecycle Chain**:
```
TestDeploymentLifecycleWorkflowE2E:
  1. Create ModelArtifact
  2. Add ModelLocation
  3. Create NBR (ready)
  4. Create deployment (save only)
  5. Preflight → verify can_run
  6. Dry-run → verify RunPlan fields
  7. Start deployment (with fake agent that returns success)
  8. Get instance status → running
  9. Get logs (fake agent returns log lines)
  10. Stop deployment
  11. Verify cleanup
```

**P1 — Model Wizard Chain**:
```
TestModelWizardWorkflowE2E:
  1. List nodes → get node_id
  2. Browse files (fake agent returns directory listing)
  3. Scan model path (fake agent returns GGUF metadata)
  4. Create ModelArtifact
  5. Add ModelLocation
  6. List artifacts → verify
  7. GET artifact → verify metadata fields
  8. Delete artifact → cleanup
```

### 5.4 Test Implementation Pattern

New API workflow tests should follow this pattern:

```go
func TestNBRProbeWorkflowE2E(t *testing.T) {
    db := setupTestDB(t)
    h := NewAgentHandler(db, nil)

    // 1. Setup: node + runtime + fake agent
    nodeID := "node-e2e-001"
    runtimeBoundaryInsertOnlineNode(t, db, nodeID)
    insertGPU(t, db, nodeID, "nvidia")
    insertRuntime(t, db, "rt-e2e-001", "vLLM Test", "")

    fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Returns docker images list + inspect
    }))
    defer fakeAgent.Close()
    setNodeAgentAddress(t, db, nodeID, fakeAgent)

    // 2. Step 1: List nodes → verify node exists
    w1 := httptest.NewRecorder()
    h.HandleListNodes(w1, newReq("GET", "/x", "", adminSession(), nil))
    assertCode(t, w1, 200)

    // 3. Step 2: Create NBR (enable)
    w2 := httptest.NewRecorder()
    h.HandleCheckNodeBackendRuntime(w2, newReq("POST", "/x",
        `{"backend_runtime_id":"rt-e2e-001","image_ref":"vllm/vllm-openai:latest","image_present":true,"docker_available":true}`,
        adminSession(), map[string]string{"id": nodeID}))
    assertCode(t, w2, 200)
    nbrID := parseResponse(t, w2)["id"].(string)

    // 4. Step 3: POST /probe
    w3 := httptest.NewRecorder()
    h.HandleProbeNodeBackendRuntime(w3, newReq("POST", "/x", `{}`,
        adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
    assertCode(t, w3, 200)
    probeResp := parseResponse(t, w3)
    assertNotEqual(t, probeResp["status"], "missing_image", "probe must not return missing_image")

    // 5. Step 4: GET /probe → verify snapshot
    w4 := httptest.NewRecorder()
    h.HandleGetNodeBackendRuntimeProbe(w4, newReq("GET", "/x", "",
        adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
    assertCode(t, w4, 200)
    snapshot := parseResponse(t, w4)
    assertNotEmpty(t, snapshot["probe_results_json"], "snapshot must not be empty")

    // 6. Step 5: List NBRs → verify probe_results_json in item
    w5 := httptest.NewRecorder()
    h.HandleListNodeBackendRuntimes(w5, newReq("GET", "/x", "",
        adminSession(), map[string]string{"id": nodeID}))
    assertCode(t, w5, 200)
    items := parseResponseArray(t, w5)
    found := findItemByID(items, nbrID)
    assertNotNil(t, found, "NBR must appear in list")
    assertNotEmpty(t, found["probe_results_json"], "list item must have probe_results_json")
}
```

**Key characteristics**:
- Exercises the full API workflow sequence
- Uses fake agents (httptest.NewServer) for external dependencies
- Uses in-memory SQLite for fast, isolated execution
- Calls handlers directly (no real router) — acceptable because PathValue regression tests cover route matching
- Verifies data consistency across multiple API calls
- Includes cleanup

## 6. E2E Script Assessment

### 6.1 Existing Script Coverage

The 22 shell E2E scripts provide excellent coverage for real-environment integration testing. They test:
- Full wizard → container lifecycle (vLLM, SGLang, llama.cpp)
- Parameter matrix verification (dry-run)
- Clone template persistence
- Failed instance logs
- RunPlan parameter propagation
- Deployment visibility

### 6.2 Script Limitations

- Require running server + agent + Docker + GPU
- Not run in CI (no `.github/workflows/`)
- Not idempotent (accumulate DB state)
- Not parallelizable (single server instance)
- Output in docs/reports/ subdirectories (scattered evidence)

### 6.3 Recommendation

Retain all E2E scripts as real-environment verification. Add a lightweight API-level E2E script that does NOT require Docker/GPU:

```
scripts/e2e-nbr-probe-api-workflow.sh
```

This script would:
- Not start containers
- Not require GPU
- Use curl against a running server
- Verify: list nodes → enable NBR → POST /probe → GET /probe → verify not missing_image
- Accept env vars for SERVER_URL, NODE_ID, NBR_ID
- Output PASS/FAIL per step

This is the smoke script previously discussed in the validation plan, now framed as an API workflow E2E rather than a "smoke test."

## 7. Questions Requiring Confirmation

1. **Real HTTP router tests**: Should we add a `SetupRoutes()` test helper that instantiates the real mux and sends HTTP requests through it? This would catch middleware chain and route ordering issues but adds complexity.

2. **API workflow E2E scope**: The recommended first batch (NBR probe chain + BackendRuntime CRUD) is ~200 lines of test code. Should this be implemented now (before Phase 5) or deferred?

3. **E2E script for NBR probe API**: Should `scripts/e2e-nbr-probe-api-workflow.sh` be implemented now? It depends on server auth which is currently blocking.

4. **CI integration**: There are no GitHub Actions workflows. Should one be created to run `go test ./...` + `npm test` on push?

## 8. Summary

| Metric | Current | Target |
|--------|---------|--------|
| Handler/unit tests | 215+ ✅ | Maintain |
| Real HTTP router tests | 0 ❌ | Add 2-3 PathValue regression tests per new route |
| API workflow E2E (Go) | 0 ❌ | Add 4 key workflow tests (P0: NBR probe + BackendRuntime CRUD; P1: Deployment lifecycle + Model wizard) |
| Real Agent smoke | E2E scripts ✅ | Add API-level agent smoke (no Docker/GPU) |
| Real container smoke | E2E scripts ✅ | Maintain opt-in only |
| UI static/component | 9 test files ✅ | Maintain all |
| Manual UI smoke | PENDING | Reduce to visual-only checks (drawer render, notices) |

**Immediate action**: Implement P0 API workflow E2E tests (NBR probe chain + BackendRuntime CRUD). These are ~150 lines of test code, require no new infrastructure, and would have caught the Phase 0 root-cause bug.
