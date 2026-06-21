# NBR Deployability Regression Issues

> Status: FIXED
> Created: 2026-06-21
> Finalized: 2026-06-21
> Scope: NBR check-request status + GPU assignment JSON tag fix

## 1. Primary Issues

### Issue A: NBR check-request deployability blocked by version probe warning

**Root cause**: Level 4 version probe always deferred; `evaluateProbeStatus` treated ALL non-probed results as warnings, producing `ready_with_warnings` for every `check-request` call. Three consumers (HandleCreateDeployment, preflightDeployment, ModelDeploymentsPage.vue) only accepted `status == "ready"`, blocking all NBRs.

**Fix**: Level 4 `probe_skipped: true`; `isNBRDeployable()` helper accepts ready + ready_with_warnings; frontend uses `nbr.deployable === true`.

### Issue B: GPU IDs silently dropped — `placement.GPUIds` struct missing json tag

**Root cause**: `preflightResult.placement` struct had `GPUIds []string` with NO `json:"gpu_ids"` tag. JSON unmarshaling from `{"gpu_ids":["..."]}` silently dropped GPU IDs, resulting in:
- `gpu_device_ids: null` in RunPlan
- `CUDA_VISIBLE_DEVICES: ""` (empty, hides all GPUs)
- vLLM: `AssertionError: DP adjusted local rank 0 is out of bounds`
- SGLang: `exec: --: invalid option` (entrypoint wrapper with no sub-command)

**Fix**: Added `json:"gpu_ids"` tag to `GPUIds` field and `json:"node_id"` tag to `NodeID` field.

This bug was invisible before because:
1. E2E tests previously used mock GPU or no-GPU paths
2. The auto-GPU-assign fallback (`len(GPUIds) == 0`) silently selected a GPU, masking the missing explicit GPU IDs
3. GPU DeviceRequest still worked (Count=-1 for all GPUs) but CUDA_VISIBLE_DEVICES was empty

---

## 2. Deployability Contract

| Status | deployable |
|--------|-----------|
| `ready` | true |
| `ready_with_warnings` | true |
| `missing_image`, `needs_check`, `runtime_image_mismatch`, `inspect_failed`, `agent_unreachable`, `docker_error`, `unsupported_device`, `disabled`, `evidence_missing`, `failed`, `unknown`, `error` | false |

---

## 3. Three-Backend E2E Results (Real NVIDIA GPU)

Environment: RTX 5090 (24GB), NVIDIA driver 610.43.02, CUDA 13.3, Docker 29.5.3, WSL2

### vLLM

| Stage | Result |
|-------|--------|
| check-request | **PASS** — `status=ready deployable=True warnings=none` |
| preflight | **PASS** — clean, no GPU warnings |
| create deployment | **PASS** |
| container start | **PASS** — vLLM 0.20.1 started successfully |
| `/v1/models` | **PASS** — 200 OK (~85s startup) |
| logs API | **PASS** |
| stop/cleanup | **PASS** |
| GPU params | **PASS** — `gpu_device_ids: ['0']`, `CUDA_VISIBLE_DEVICES=0`, `--gpus "device=0"` |
| Default + modified params | **PASS** both |

### SGLang

| Stage | Result |
|-------|--------|
| check-request | **PASS** — `status=ready deployable=True` |
| preflight | **PASS** |
| create deployment | **PASS** |
| GPU params | **PASS** — `gpu_device_ids: ['0']`, `CUDA_VISIBLE_DEVICES=0` |
| container start | **BLOCKED** — SGLang `nvidia_entrypoint.sh` wrapper expects sub-command; process start config needs `python3 -m sglang.launch_server` prefix (separate issue, not GPU/deployability) |

### llama.cpp

| Stage | Result |
|-------|--------|
| check-request | **PASS** — `status=ready deployable=True` |
| preflight | **PASS** |
| create deployment | **PASS** |
| container start | **PASS** |
| `/v1/models` | **PASS** — 200 OK |
| instance test (chat) | **PASS** |
| logs API | **PASS** |
| stop/cleanup | **PASS** |

---

## 4. Files Modified

| File | Change |
|------|--------|
| `internal/server/api/runtime_handlers.go` | `probe_skipped: true`, `isNBRDeployable()`, `extractProbeWarnings()`, `nbrDisabledReason()`, deployable/warnings/disabled_reason in API responses, ready_count + deployable_count |
| `internal/server/api/deployment_lifecycle_handlers.go` | **`placement.GPUIds` json tag fix**, `isNBRDeployable()` in HandleCreateDeployment + preflightDeployment |
| `internal/server/api/nbr_deployable_test.go` | NEW — 6 deployable contract tests |
| `web/src/pages/ModelDeploymentsPage.vue` | `isNBRDeployable()` uses `nbr.deployable === true`, disabled reason display |
| `scripts/e2e/lib/model-runtime-common.sh` | `e2e_check_nbr()` uses check-request |
| `scripts/e2e-model-runtime-wizard-nvidia-*.sh` | Runtime ID fixes, GPU resilience, check-request |
| `scripts/e2e-real-smoke-all-three.sh` | check-request for all three backends |
| `.gitignore` | Added `web/runtime/` |
| `docs/reports/phase-3/runtime-nbr-deployability-regression-issues.md` | This document |

---

## 5. Verification Commands

```
go test lightai-go/internal/server/api/... → ALL PASS
go test lightai-go/internal/server/runplan/... → ALL PASS
go vet → CLEAN
npm test → 20 PASS, 784 i18n keys
npm run build → ✓
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh → vLLM default PASS, modified PASS
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh → GPU chain PASS (container: process_start_config)
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh → FULL PASS
git diff --check → CLEAN
git status --short → CLEAN
```

---

## 6. E2E Evidence

```
docs/reports/model-runtime-node-wizard/e2e-matrix-20260621115854/vllm/
docs/reports/model-runtime-node-wizard/e2e-matrix-20260621115854/vllm-modified/
docs/reports/model-runtime-node-wizard/e2e-sglang-*/
docs/reports/model-runtime-node-wizard/e2e-llamacpp-*/
```
