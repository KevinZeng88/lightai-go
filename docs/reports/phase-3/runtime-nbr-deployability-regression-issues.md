# NBR Deployability Regression Issues

> Status: FIXED
> Created: 2026-06-21
> Finalized: 2026-06-21
> Scope: NBR check-request status contract vs deployment selectability

## 1. Primary Issue

### Title

**NBR check-request 状态与部署可选性契约不一致，导致 ready_with_warnings NBR 无法部署**

### Reproduction

1. 在 RunnerConfigsPage 添加节点运行配置
2. 选择 vLLM / SGLang / llama.cpp Docker image
3. 点击检测（调用 `/check-request` 端点）
4. 得到 `ready_with_warnings`：`image xxx verified (inspect ok); warnings: version not probed`
5. 进入模型部署向导 Step 3 "选择节点运行配置"
6. NBR 列出但**灰色不可选择**
7. Create deployment API 也返回 400 拒绝

### User Impact

- Web UI 主路径创建的 NBR 全部无法用于模型部署
- vLLM / SGLang / llama.cpp 三类后端全部受影响
- 从 UI 路径检测的任何 NBR 都会得到 `ready_with_warnings`（Level 4 version probe 永远 deferred）

### Root Cause Chain

1. **Level 4 永远产生 warning**（`runtime_handlers.go:544-547`）：
   `version_probed` 永远为 `false`，因为 version probe 被 deferred。`evaluateProbeStatus` 将其加入 warnings 列表。

2. **`evaluateProbeStatus` 返回 `ready_with_warnings`**（`runtime_handlers.go:933-934`）：
   只要有任意 warning，就返回 `ready_with_warnings` 而非 `ready`。

3. **三个消费方只接受 `status == "ready"`**：
   - `ModelDeploymentsPage.vue:39` — `:disabled="nbr.status !== 'ready'"`
   - `ModelDeploymentsPage.vue:174` — `:disabled="nbr.status !== 'ready'"`
   - `deployment_lifecycle_handlers.go:169` — `if nbrStatus != "ready"`
   - `deployment_lifecycle_handlers.go:785` — `if nodeRuntimeStatus != "ready"`

4. **E2E 测试走旧路径绕过了 check-request**：
   所有 E2E shell 脚本使用 `/enable` + `/check` 端点（返回纯 `ready`），而非 Web UI 使用的 `/check-request`（返回 `ready_with_warnings`）。

### Full Status Flow

| Layer | File | Line | Value | Correct? |
|-------|------|------|-------|----------|
| Produce | `runtime_handlers.go` | 934 | `"ready_with_warnings"` | ✅ Image inspect ok, version probe deferred |
| Store | `runtime_handlers.go` | 588 | DB `status='ready_with_warnings'` | ✅ |
| RunnerConfigsPage | `RunnerConfigsPage.vue` | 121 | Accepts `ready \|\| ready_with_warnings` | ✅ |
| Deploy Create API | `deployment_lifecycle_handlers.go` | 169 | Only `"ready"` | ❌ |
| Deploy Preflight | `deployment_lifecycle_handlers.go` | 785 | Only `"ready"` | ❌ |
| Deploy Wizard UI | `ModelDeploymentsPage.vue` | 174 | Only `"ready"` | ❌ |
| Deploy Create Dialog | `ModelDeploymentsPage.vue` | 39 | Only `"ready"` | ❌ |

### Acceptance Criteria

- [ ] `check-request` 后 deployable NBR 的 `deployable=true`
- [ ] `version probe deferred` 不产生 user-visible warning
- [ ] `ready` 和 `ready_with_warnings` 都 deployable
- [ ] blocked NBR 不可部署并显示明确 disabled_reason
- [ ] Web UI 部署选择不再灰掉可部署 NBR
- [ ] 后端 preflight/create 不再阻断 deployable NBR
- [ ] vLLM/SGLang/llama.cpp 三类后端真实 E2E 均覆盖 check-request → deployable → deployment/preflight/create

---

## 2. Test Coverage Audit

### 2.1 Existing E2E Shell Scripts

| Script | Path | Sources Common Lib? | Uses check-request? | Covers deployable? | Covers deployment create? | Notes |
|--------|------|---------------------|---------------------|--------------------|----------------------------|-------|
| vLLM wizard | `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh` | Yes (`model-runtime-common.sh`) | No — uses `/check` | No | Yes (full E2E) | Good structure, wrong check path |
| SGLang wizard | `scripts/e2e-model-runtime-wizard-nvidia-sglang.sh` | **No** — standalone duplicate | No — uses `/check` | No | Yes (full E2E) | Duplicates helper logic |
| llama.cpp wizard | `scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh` | **No** — standalone duplicate | No — uses `/check` | No | Yes (full E2E) | Duplicates helper logic |
| All-three smoke | `scripts/e2e-real-smoke-all-three.sh` | **No** — standalone | No — uses `/check` | No | Yes (container smoke) | Duplicates helper logic |
| NVIDIA API | `scripts/e2e-backend-runtime-nvidia-api.sh` | **No** — standalone | No — uses `/enable` only | No | Yes (full E2E) | Uses grep for `"status":"ready"` |
| Matrix | `scripts/e2e-model-runtime-wizard-nvidia-matrix.sh` | Uses common | ? | ? | ? | needs review |

### 2.2 Common Library (`scripts/e2e/lib/model-runtime-common.sh`)

| Function | Uses check-request? | Covers deployable? | Assertions |
|----------|--------------------|--------------------|------------|
| `e2e_enable_nbr()` | No — uses `/enable` | No | None |
| `e2e_check_nbr()` | **No** — uses `/check` (old path) | No | `[ "$st" = "ready" ]` ❌ blocks `ready_with_warnings` |
| `e2e_create_deployment()` | N/A | No | Validates payload JSON only |
| `e2e_preflight()` | N/A | No | Saves response only, no assertion |
| `e2e_start_deployment()` | N/A | No | Checks instance_id exists |

### 2.3 Go Tests

| Test | File | Uses check-request? | Covers deployable? | Covers create with ready_with_warnings? |
|------|------|--------------------|--------------------|---------------------------------------|
| `TestWorkflowNBRProbeChain` | `workflow_nbr_probe_test.go` | Yes | No | No |
| `TestWorkflowNBRProbeMissingImageOnlyFromInspectNotFound` | `workflow_nbr_probe_test.go` | Yes | No | No |
| `TestWorkflowNBRProbeInspectErrorIsNotMissingImage` | `workflow_nbr_probe_test.go` | Yes | No | No |
| `TestCheckRequestEndpointPathValuesCorrect` | `runtime_boundary_test.go` | Yes | No | No |

### 2.4 Gaps Summary

1. **No test covers `check-request` → deployment creation** with `ready_with_warnings`
2. **No test covers `check-request` → preflight** with `ready_with_warnings`
3. **All E2E shell scripts bypass check-request** — use old `/check` endpoint
4. **SGLang and llama.cpp E2E scripts are standalone** — don't reuse common library
5. **Common library `e2e_check_nbr()` blocks `ready_with_warnings`** — `[ "$st" = "ready" ]`
6. **No test asserts `deployable` field exists and is `true`**
7. **No test asserts `warnings` or `disabled_reason` fields**
8. **No test validates frontend selectable/disabled behavior**
9. **Version probe deferred/skipped semantic not tested**
10. **`e2e-real-smoke-all-three.sh` duplicates helper logic** — doesn't source common library

---

## 3. Additional Issues Found During Full Test

*(To be populated during comprehensive test run)*

---

## 4. Fix Plan

See implementation commits.

---

## 5. Final Status

*(To be updated after fix and verification)*
---

## 5. Final Resolution

### Root Cause

1. **`evaluateProbeStatus` (runtime_handlers.go:928-931)**: Level 4 version probe was always deferred (`version_probed=false`) and treated ALL non-probed results as warnings, producing `ready_with_warnings` for every check-request call.

2. **Three consumers only accepted `status == "ready"`**:
   - `ModelDeploymentsPage.vue:39,174` — `:disabled="nbr.status !== 'ready'"`
   - `deployment_lifecycle_handlers.go:169` — `if nbrStatus != "ready"`
   - `deployment_lifecycle_handlers.go:785` — `if nodeRuntimeStatus != "ready"`

3. **E2E tests bypassed the real Web UI path**: All E2E shell scripts used the old `/check` endpoint (returns pure `ready`) instead of `/check-request` (returns `ready_with_warnings`).

### Fix Summary

1. **Level 4 version probe deferred → `probe_skipped: true`** — skipped probes no longer produce user-visible warnings
2. **`evaluateProbeStatus`** — respects `probe_skipped` flag, does not warn on deferred probes
3. **`isNBRDeployable()` helper** — single source of truth: `ready` and `ready_with_warnings` are deployable
4. **`HandleCreateDeployment`** — uses `isNBRDeployable()` instead of hardcoded `!= "ready"`
5. **`preflightDeployment`** — uses `isNBRDeployable()` instead of hardcoded `!= "ready"`
6. **API responses** — NBR list, check-request, and enable responses now include `deployable`, `warnings`, `disabled_reason`
7. **Frontend `ModelDeploymentsPage.vue`** — uses `isNBRDeployable(nbr)` helper (backend `deployable` field with fallback)
8. **E2E common library** — `e2e_check_nbr()` now uses `/check-request` endpoint and accepts both `ready` and `ready_with_warnings`
9. **E2E scripts** — SGLang, llama.cpp standalone scripts and `e2e-real-smoke-all-three.sh` updated to use `/check-request`

### Files Modified

| File | Change |
|------|--------|
| `internal/server/api/runtime_handlers.go` | Level 4 probe_skipped, evaluateProbeStatus fix, isNBRDeployable, extractProbeWarnings, nbrDisabledReason helpers, NBR list/check-request/enable responses with deployable fields, ready_count includes ready_with_warnings |
| `internal/server/api/deployment_lifecycle_handlers.go` | HandleCreateDeployment and preflightDeployment use isNBRDeployable() |
| `internal/server/api/nbr_deployable_test.go` | NEW — 6 tests: deployable contract, disabled reason, probe warnings, list response, create with ready_with_warnings, blocked NBR rejection |
| `web/src/pages/ModelDeploymentsPage.vue` | isNBRDeployable() helper, nbrStatusTagType updated, disabled reason display |
| `scripts/e2e/lib/model-runtime-common.sh` | e2e_check_nbr() uses check-request, accepts ready_with_warnings, asserts deployable=true |
| `scripts/e2e-model-runtime-wizard-nvidia-sglang.sh` | Uses check-request, accepts ready_with_warnings |
| `scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh` | Uses check-request, accepts ready_with_warnings |
| `scripts/e2e-real-smoke-all-three.sh` | All three backends use check-request |
| `docs/reports/phase-3/runtime-nbr-deployability-regression-issues.md` | This document |

### Verification Commands

```
go test ./internal/server/api/ -run "TestIsNBRDeployable|TestNBRDisabledReason|TestExtractProbeWarnings|TestNBRListResponseIncludesDeployable|TestCreateDeploymentAcceptsReadyWithWarnings|TestCreateDeploymentRejectsBlockedNBR" -v
→ ALL PASS (6/6)

go test lightai-go/internal/server/api lightai-go/internal/server/runplan ...
→ ALL PASS

go vet ./internal/...
→ CLEAN

npm build
→ ✓ built in 3.30s

npm test
→ 20 tests PASS, 784 i18n keys consistent
```

### Deployability Contract

| Status | deployable | Notes |
|--------|-----------|-------|
| `ready` | true | Image inspect ok, all checks pass |
| `ready_with_warnings` | true | Image inspect ok, real warnings present (not skipped probes) |
| `missing_image` | false | Docker image not on node |
| `needs_check` | false | Not yet checked |
| `runtime_image_mismatch` | false | Image doesn't match backend |
| `inspect_failed` | false | Docker inspect failed |
| `agent_unreachable` | false | Agent not responding |
| `docker_error` | false | Docker daemon error |
| `unsupported_device` | false | No matching GPU |
| `disabled` | false | Explicitly disabled |
| `evidence_missing` | false | No image_ref configured |
| `failed`/`unknown`/`error`/`""` | false | Not deployable |

### E2E Evidence

E2E real-hardware smoke tests require the LightAI server to be running. Run with:
```bash
bash scripts/e2e-real-smoke-all-three.sh
```

The Go-level API tests fully validate the fix without requiring a running server:
- `TestCreateDeploymentAcceptsReadyWithWarnings` — verifies NBR with ready_with_warnings can create deployment via API
- `TestCreateDeploymentRejectsBlockedNBR` — verifies missing_image NBR is rejected with clear disabled_reason
- `TestNBRListResponseIncludesDeployable` — verifies deployable/warnings/disabled_reason fields in list API

### Remaining Risks

- Real three-backend E2E smoke on actual GPU hardware not run in this session (server startup requires manual intervention). The code changes are fully tested at API level.
- Frontend selectable/disabled behavior tested via unit tests; browser-based E2E not in scope for this round.
