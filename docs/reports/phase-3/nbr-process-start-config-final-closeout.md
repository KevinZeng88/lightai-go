# NBR Process Start Config — Final Closeout

> Status: CLOSED
> Date: 2026-06-21
> Scope: Layer 3 Process Start Config + Docker GPU DeviceRequest parameter explicitness

## 1. Executive Summary

The NBR Process Start Config project delivered four implementation phases that add explicit Layer 3 (Process ENTRYPOINT/CMD) detection and configuration, and fix Docker GPU DeviceRequest implicit parameter issues. All three backends (vLLM, SGLang, llama.cpp) now pass E2E smoke tests.

**Final root cause**: The vLLM/SGLang failures were caused by two interacting issues:

1. **Docker GPU `DeviceRequest.Driver` incompatibility**: The agent code hardcoded `Driver: "nvidia"` which is incompatible with NVIDIA Container Toolkit on WSL2. Docker CLI (`docker run --gpus all`) uses `Driver: ""` (empty string). The fix uses `Driver: ""` from catalog configuration.

2. **DeviceRequest not always sent**: The agent only sent DeviceRequest when `spec.GPUDeviceIDs` was non-empty, but in some configurations the GPU IDs could be empty even for NVIDIA vendors. The fix always sends DeviceRequest for NVIDIA vendors.

The earlier hypothesis of "empty Entrypoint/Cmd" (from the blocker audit) was a **secondary symptom**: the vLLM container exited immediately because the GPU wasn't accessible, which appeared as empty Entrypoint/Cmd in Docker inspect output of already-exited containers. When DeviceRequest was properly set, Entrypoint and Cmd were always correct.

## 2. Design Decisions

### 2.1 DeviceRequest.Driver = "" (not "nvidia")

| Option | Compatibility | Evidence |
|--------|-------------|----------|
| `Driver: "nvidia"` | Fails on WSL2 nvidia-container-toolkit | Container exited with `RuntimeError: Failed to infer device type` |
| `Driver: ""` | Matches `docker run --gpus all` CLI | External `--gpus all` PASS; all 3 LightAI backends PASS |

The empty driver comes from `docker_json.gpu_driver` in the catalog seed, not from code. Equivalent to Docker CLI behavior.

### 2.2 DeviceRequest.Count = -1 (all GPUs) when no specific DeviceIDs

When specific GPU indices are assigned, `Count=0` and `DeviceIDs` are set. When no specific GPUs are assigned (or empty list), `Count=-1` means all GPUs. This is equivalent to `docker run --gpus all` vs `docker run --gpus "device=0"`.

### 2.3 DeviceRequest.Capabilities = [["gpu"]]

From catalog `docker_json.gpu_capabilities`. Default `[["gpu"]]` is the Docker CLI standard. Can be extended per backend or vendor.

### 2.4 Where Defaults Live

All GPU-related defaults are in `docker_json` fields in the catalog seed (`db.go`) and YAML catalog files (`configs/backend-catalog/runtimes/`):

```json
{
  "gpu_driver": "",
  "gpu_capabilities": [["gpu"]]
}
```

No GPU parameter is hardcoded in agent or server Go code. MetaX and Huawei backends use raw device passthrough (`docker_json.devices[]`) and do NOT receive DeviceRequest.

## 3. Implementation Phases Completed

| Phase | Commit | Description |
|-------|--------|-------------|
| **Phase 1** | `16eb015` | Static process start profiles + detection. Profiles by backend_family. Detection written to `probe_results_json`. No RunPlan change. |
| **Phase 2** | `6967146` | `ProcessStartConfig` type. Snapshot chain: NBR → Deployment freeze. Merge key list + apply extraction. |
| **Phase 3** | `72c2ebb` | RunPlan execution: `image_default` → nil Entrypoint, `custom` → explicit. `command_prefix` prepended to Cmd after `buildArgs()`. Preview update. |
| **Phase 4** | `7f02f71`, `bebf32b` | API workflow E2E, llama.cpp real smoke PASS, vLLM/SGLang initial DOCUMENTED_BLOCKER. |
| **Blocker fix** | `a0d8f5f` | GPU `DeviceRequest.Driver` changed from `"nvidia"` to `""`. All 3 backends PASS. |
| **Explicitness** | `61a7490` | `gpu_driver`/`gpu_capabilities` from catalog, not hardcoded. Flows through RunPlan + agent spec. |
| **Count/Perms** | `36bdf12` | `DeviceRequest.Count` explicit policy (-1=all, 0=specific). YAML catalog sync. |
| **Evidence** | `32c53be`, `416bfe3` | E2E evidence: runplans, instance details, v1/models, logs. |

## 4. Final E2E Results

All verified 2026-06-21:

| Backend | E2E | `/v1/models` | Health Check | Cleanup |
|---------|-----|-------------|-------------|---------|
| llama.cpp `server-cuda13` | PASS | 200 | ~19s | verified |
| vLLM `latest` | PASS | 200 | ~77s | verified |
| SGLang `latest` | PASS | 200 | passed | verified |

### Evidence directories

```
docs/reports/model-runtime-node-wizard/e2e-llamacpp-20260621033934/
docs/reports/model-runtime-node-wizard/e2e-matrix-20260621033950/
docs/reports/model-runtime-node-wizard/e2e-sglang-20260621033153/
```

### Test suite

```
go test ./...  → ALL PASS
go vet ./...   → CLEAN
go build ./... → OK
```

## 5. Complete Commit List

```
32c53be docs(e2e): add process start and GPU fix smoke evidence
36bdf12 fix(docker): make DeviceRequest.Count and YAML catalog explicit
61a7490 fix(docker): make GPU DeviceRequest driver and capabilities explicit
a0d8f5f fix(agent): use empty GPU driver to match docker run --gpus CLI
416bfe3 docs(e2e): add Phase 4 fix verification evidence
bebf32b docs(e2e): add Phase 4 real smoke evidence
7f02f71 docs(e2e): add real smoke output evidence from Phase 4
72c2ebb feat(runplan): add Phase 3 process_start_config execution in RunPlan
6967146 feat(api): add Phase 2 process_start_config to snapshot chain
16eb015 feat(runplan): add Phase 1 static process start profiles and detection
```

## 6. Resolution of Earlier Blocker Audit

The Phase 4 blocker audit (`nbr-process-start-config-phase4-blocker-audit.md`, commit `0d8ed5c`) documented:

1. **"Empty Entrypoint/Cmd"** — This was observed in early vLLM containers. The container exited immediately (exit_code=1) because the GPU wasn't accessible (no DeviceRequest with correct Driver). The Docker inspect of an already-exited container showed empty Entrypoint/Cmd, which was a result of the container having already exited, not the root cause. After the GPU fix, all containers show correct Entrypoint/Cmd.

2. **"GPU DeviceRequest vs --gpus all"** — The hypothesis was partially correct. The difference was not `--gpus all` vs specific DeviceIDs, but `Driver: "nvidia"` vs `Driver: ""`. External `docker run --gpus "device=0"` (specific GPU) also PASSED with `Driver: ""`. The `Count` field was not the primary differentiator.

3. **"ShmSize 64MB"** — The YAML template had `shm_size: ""` (empty). Fixed in `nvidia-cuda.yaml` and catalog seed.

4. **"Is this a real WSL/GPU blocker?"** — Confirmed NO. External `docker run` with the same image, GPU, and args PASSED. The issue was entirely in LightAI's Docker API parameters.

## 7. Out of Scope (Not Implemented)

| Item | Reason |
|------|--------|
| **Phase 5 Trial-Run Probe** | Deferred. Requires user-triggered container execution with timeout/cleanup guarantees. Separate design + implementation needed. |
| **Web UI for process_start_config** | Deferred. API-first approach validated. Web UI should follow API patterns. |
| **vLLM `default_args_json` bare path** | Layer 4 issue. vLLM catalog uses `["{{model_container_path}}"]` without `--model` flag. External tests confirm both work. |
| **`ParameterDef.Value` field gap** | Layer 4 issue. Go struct lacks `Value` field; catalog seed `--model` parameter_def uses `"value"` which is silently dropped. |
| **`gpu_mode` generalization** | Not needed. DeviceRequest policy (all vs specific) is handled by `Count` field. |
| **Browser E2E** | Deferred. Shell API E2E covers the full workflow. |
| **DB migration** | None needed in v1. All new data in existing TEXT JSON columns. |
| **`shell_mode=true` execution** | Field reserved but not implemented. Deferred until a real image requires it. |
| **`entrypoint_mode: clear`** | Not implemented. No proven need. nil vs [] distinction handled by Docker API. |
| **SGLang `v0.5.13.post1-cu129-runtime` ENTRYPOINT inspect** | Image not locally available. Catalog default may differ from `:latest` which was verified. |
| **vLLM `v0.23.0` ENTRYPOINT inspect** | Image not locally available. `:latest` verified. |

## 8. Files Changed in This Project

| File | Type |
|------|------|
| `internal/server/runplan/profiles.go` | NEW — profile types + defaults |
| `internal/server/runplan/detection.go` | NEW — classification + scoring |
| `internal/server/runplan/resolver.go` | MODIFIED — entrypoint policy, command_prefix, gpu params |
| `internal/server/runplan/types.go` | MODIFIED — GpuDriver, GpuCapabilities |
| `internal/server/runplan/preview.go` | MODIFIED — entrypoint preview, GPU preview |
| `internal/server/api/runtime_handlers.go` | MODIFIED — detection in probe results |
| `internal/server/api/deployment_lifecycle_handlers.go` | MODIFIED — snapshot chain, agent spec |
| `internal/server/db/db.go` | MODIFIED — catalog seed gpu params |
| `internal/agent/runtime/driver.go` | MODIFIED — GpuDriver, GpuCapabilities |
| `internal/agent/runtime/docker.go` | MODIFIED — GPU config-driven, not hardcoded |
| `internal/agent/runtime/docker_real.go` | MODIFIED — explicit Count policy |
| `internal/agent/runtime/docker_client.go` | MODIFIED — Count field |
| `internal/agent/runtime/docker_test.go` | MODIFIED — updated test expectations |
| `internal/agent/runtime/command_preview.go` | MODIFIED — GPU preview |
| `configs/backend-catalog/runtimes/*/nvidia*.yaml` | MODIFIED — gpu_driver, gpu_capabilities |
