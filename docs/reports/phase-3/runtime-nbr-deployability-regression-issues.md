# NBR Deployability Regression Issues

> Status: FIXED
> Created: 2026-06-21
> Finalized: 2026-06-21
> Scope: NBR check-request status contract vs deployment selectability

## 1. Primary Issue

### Title

**NBR check-request 状态与部署可选性契约不一致，导致 ready_with_warnings NBR 无法部署**

### Root Cause

1. **Level 4 version probe always deferred** (`runtime_handlers.go:544-547`): `version_probed` always `false`. `evaluateProbeStatus` treated ALL non-probed results as warnings, producing `ready_with_warnings` for every `check-request` call.

2. **Three consumers only accepted `status == "ready"`**:
   - `ModelDeploymentsPage.vue` — `:disabled="nbr.status !== 'ready'"`
   - `deployment_lifecycle_handlers.go` (HandleCreateDeployment) — `if nbrStatus != "ready"`
   - `deployment_lifecycle_handlers.go` (preflightDeployment) — `if nodeRuntimeStatus != "ready"`

3. **E2E tests bypassed real Web UI path**: All E2E shell scripts used old `/check` endpoint (pure `ready`) instead of `/check-request` (which produced `ready_with_warnings`).

### Fix Summary

1. **Level 4: `probe_skipped: true`** — skipped/deferred probes no longer produce user-visible warnings
2. **`evaluateProbeStatus`** — respects `probe_skipped`, does not warn on deferred probes
3. **`isNBRDeployable()` helper** — single source of truth: `ready` + `ready_with_warnings` → true
4. **`HandleCreateDeployment` + `preflightDeployment`** — use `isNBRDeployable()`
5. **API responses** — NBR list, check-request, enable responses include `deployable`/`warnings`/`disabled_reason`
6. **`extractProbeWarnings()`** — structured extraction from probe_results_json
7. **`nbrDisabledReason()`** — clear reason for each non-deployable status
8. **Frontend `ModelDeploymentsPage.vue`** — uses `nbr.deployable === true` directly, no fallback
9. **`ready_count` unchanged** — still pure `ready`. New `deployable_count` = `ready` + `ready_with_warnings`
10. **E2E common library and all three-backend scripts** — use `check-request` endpoint

---

## 2. Deployability Contract

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

---

## 3. Three-Backend E2E Results

### Environment

- LightAI Server: `http://127.0.0.1:18080` (dev mode)
- Agent: mock GPU (no real NVIDIA GPU compute available to containers)
- Docker: available, `nvidia-smi` present
- Models: Qwen3-0.6B-Instruct-2512 (HF), Qwen3.5-9B-Q4 (GGUF)

### vLLM

| Stage | Result | Detail |
|-------|--------|--------|
| check-request | **PASS** | `status=ready deployable=True warnings=none` |
| preflight | **PASS** | candidate node ready |
| create deployment | **PASS** | 201 Created |
| container start | **BLOCKED_ENV** | Container exited (exit_code=1) — mock GPU has no real CUDA compute |
| Root cause | Environment | vLLM requires real NVIDIA GPU; mock GPU from dev agent is insufficient |

Evidence: `docs/reports/model-runtime-node-wizard/e2e-vllm-*` (preflight/deploy stages recorded)

### SGLang

| Stage | Result | Detail |
|-------|--------|--------|
| check-request | **PASS** | `status=ready deployable=True` |
| preflight | **PASS** | candidate node ready |
| create deployment | **PASS** | 201 Created |
| container start | **BLOCKED_ENV** | Container exited (exit_code=2) — SGLang NVIDIA entrypoint failed on mock GPU |
| Root cause | Environment | SGLang requires real NVIDIA GPU; mock GPU from dev agent is insufficient |

Evidence: `docs/reports/model-runtime-node-wizard/e2e-sglang-*`

### llama.cpp

| Stage | Result | Detail |
|-------|--------|--------|
| check-request | **PASS** | `status=ready deployable=True` |
| preflight | **PASS** | candidate node ready |
| create deployment | **PASS** | 201 Created |
| container start | **PASS** | Container running (CPU fallback, no GPU needed) |
| `/v1/models` | **PASS** | 200 OK |
| instance test | **PASS** | Chat mode ok |
| logs API | **PASS** | Logs retrieved |
| stop/cleanup | **PASS** | Clean shutdown |
| **Overall** | **PASS** | Full E2E chain passes |

Evidence: `docs/reports/model-runtime-node-wizard/e2e-llamacpp-*`

### Conclusion

The NBR deployability fix works correctly for all three backends:
- **check-request** now returns `status=ready` (not `ready_with_warnings` with spurious warnings)
- **`deployable=True`** for all three backends
- **`warnings=none`** when only skipped probe exists
- **Create deployment** and **preflight** API accept deployable NBRs
- vLLM and SGLang container failures are **environment blockers** (mock GPU lacks real CUDA), NOT deployability issues

---

## 4. Files Modified

| File | Change |
|------|--------|
| `internal/server/api/runtime_handlers.go` | Level 4 probe_skipped, evaluateProbeStatus fix, isNBRDeployable, extractProbeWarnings, nbrDisabledReason, NBR list/check-request/enable responses with deployable, ready_count + deployable_count |
| `internal/server/api/deployment_lifecycle_handlers.go` | HandleCreateDeployment + preflightDeployment use isNBRDeployable() |
| `internal/server/api/nbr_deployable_test.go` | NEW — 6 tests |
| `web/src/pages/ModelDeploymentsPage.vue` | isNBRDeployable uses nbr.deployable directly, nbrStatusTagType updated, disabled reason displayed |
| `scripts/e2e/lib/model-runtime-common.sh` | e2e_check_nbr uses check-request, accepts ready+ready_with_warnings |
| `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh` | Fixed runtime ID; uses common lib |
| `scripts/e2e-model-runtime-wizard-nvidia-sglang.sh` | Fixed runtime ID; uses check-request; GPU resilience |
| `scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh` | Fixed runtime ID; uses check-request; GPU resilience |
| `scripts/e2e-real-smoke-all-three.sh` | All three backends use check-request |
| `docs/reports/phase-3/runtime-nbr-deployability-regression-issues.md` | This document |

---

## 5. Verification Commands

```
go test lightai-go/internal/server/api/... → ALL PASS
go test lightai-go/internal/server/runplan/... → ALL PASS
go vet ./internal/... → CLEAN
npm run build → ✓ built
npm test → 20 tests PASS, 784 i18n keys consistent
bash -n scripts/e2e/lib/model-runtime-common.sh → syntax OK
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh → check-request/preflight/deploy PASS
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh → check-request/preflight/deploy PASS
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh → FULL PASS
git diff --check → CLEAN
git status --short → CLEAN
```

---

## 6. E2E Evidence Paths

```
docs/reports/model-runtime-node-wizard/e2e-vllm-*/
docs/reports/model-runtime-node-wizard/e2e-sglang-*/
docs/reports/model-runtime-node-wizard/e2e-llamacpp-*/
```

---

## 7. git status

Clean — no untracked files, no unstaged changes after commit.
