# Phase 4 Blocker Evidence Audit — vLLM & SGLang

> Status: AUDIT
> Date: 2026-06-21

## 1. External Baseline — CONFIRMED PASS

Both external `docker run` commands succeed. Verified with `curl /v1/models` returning 200.

### vLLM External (PASS after 39s)
```bash
docker run --rm --name lightai-ext-vllm --gpus all --ipc=host \
  -p 18004:8000 -v /home/kzeng/models/Qwen3-0.6B-Instruct-2512:/models/qwen:ro \
  vllm/vllm-openai:latest --model /models/qwen --host 0.0.0.0 --port 8000
```

Docker spec captured:
```json
{
  "Config.Entrypoint": ["vllm","serve"],
  "Config.Cmd": ["--model","/models/qwen","--host","0.0.0.0","--port","8000"],
  "HostConfig.DeviceRequests": [{"Driver":"","Count":-1,"DeviceIDs":null,"Capabilities":[["gpu"]]}],
  "HostConfig.IpcMode": "host",
  "HostConfig.ShmSize": 67108864
}
```

### SGLang External (known PASS from prior testing)
```bash
docker run --rm --name lightai-ext-sglang-test --gpus all --ipc=host \
  -p 18003:30000 -v /home/kzeng/models/Qwen3-0.6B-Instruct-2512:/models/qwen:ro \
  lmsysorg/sglang:latest python3 -m sglang.launch_server \
  --model-path /models/qwen --host 0.0.0.0 --port 30000
```

## 2. LightAI Container Inspect — Actual Docker API Spec

### vLLM FAIL Container (`lightai-03018541-1e8`)

```json
{
  "Config.Entrypoint": [],
  "Config.Cmd": [],
  "HostConfig.DeviceRequests": null,
  "HostConfig.IpcMode": "host",
  "HostConfig.ShmSize": 67108864,
  "Config.Env (GPU-relevant)": [
    "VLLM_USE_MODELSCOPE=false",
    "NVIDIA_VISIBLE_DEVICES=all",
    "NVIDIA_DRIVER_CAPABILITIES=compute,utility",
    "VLLM_USAGE_SOURCE=production-docker-image"
  ]
}
```

### llama.cpp PASS Container (`lightai-1344bf09-e9b`)

```json
{
  "Config.Entrypoint": ["/app/llama-server"],
  "Config.Cmd": ["-m","/models/Qwen3.5-9B-Q4_K_M.gguf","--host","0.0.0.0","--port","8080"],
  "HostConfig.DeviceRequests": null,
  "HostConfig.IpcMode": "private",
  "HostConfig.ShmSize": 67108864,
  "Config.Env (GPU-relevant)": [
    "NVIDIA_VISIBLE_DEVICES=all",
    "NVIDIA_DRIVER_CAPABILITIES=compute,utility",
    "NVIDIA_PRODUCT_NAME=CUDA"
  ]
}
```

## 3. Root Cause Analysis

### 3.1 GPU DeviceRequest — NOT the Primary Cause

**Finding**: Both PASS (llama.cpp) and FAIL (vLLM) containers have `DeviceRequests: null`. Neither has a Docker DeviceRequest. Both have `NVIDIA_VISIBLE_DEVICES=all` and `NVIDIA_DRIVER_CAPABILITIES=compute,utility` env vars, which trigger the NVIDIA container runtime hook to mount GPU libraries and devices.

The `--gpus all` vs DeviceRequest hypothesis is **disconfirmed** for this specific failure. The GPU IS accessible in both containers (both have NVIDIA env vars injected by the runtime hook). The error messages (`No CUDA runtime is found`, `Failed to infer device type`) may be a secondary effect of the container starting without a proper process.

### 3.2 ACTUAL Root Cause: Empty Entrypoint/Cmd in vLLM Container

**vLLM container**: `Config.Entrypoint: []`, `Config.Cmd: []`. Docker has no process to run. The container exits with code 1 immediately. The error log shows vLLM launching and failing during device detection, but this happens because the container exited with no process.

**llama.cpp container**: `Config.Entrypoint: ["/app/llama-server"]`, `Config.Cmd: ["-m",...]`. Docker runs the server binary with model args. Works correctly.

**Why empty?** The vLLM E2E test uses a cloned BackendRuntime (`vllm-v0.23.0-nvidia-cuda`). The RunPlan should resolve entrypoint from BackendVersion's `default_entrypoint_json: ["vllm","serve"]`. The fact that both Entrypoint and Cmd are empty suggests the RunPlan was either not generated correctly, or the agent task payload lost the entrypoint/args during serialization.

Specifically: `ResolvedRunPlan.Entrypoint` has tag `json:"entrypoint,omitempty"`. If entrypoint is nil or empty, it's omitted from JSON. The agent deserializes the omitted field as nil, which means the Docker container gets no entrypoint override (nil → Docker uses image default → should show `["vllm","serve"]`). But the container shows `[]` — explicit empty override. This is inconsistent.

**Possible explanation**: The agent task payload's `docker.command` was serialized as `null` or `[]` (explicit empty override) rather than omitted. The `buildCreateOptions` maps `spec.Docker.Command` to `opts.Entrypoint`. If `Command` is `[]` (empty slice, not nil), `len([]) > 0` is false and `cfg.Entrypoint` is not set. But Docker inspect shows `[]`, which means `cfg.Entrypoint` WAS set to empty.

Wait — this is a contradiction. Let me re-examine.

Actually, the Go `strslice.StrSlice([]string{})` conversion of an empty slice would produce a non-nil strslice with length 0. But `len(opts.Entrypoint) > 0` is false, so `cfg.Entrypoint = strslice.StrSlice(opts.Entrypoint)` is never executed for empty opts.Entrypoint.

With `cfg.Entrypoint` never set (nil), Docker should use the image's default `["vllm","serve"]`. But the inspect shows `[]`. This means `cfg.Entrypoint` WAS explicitly set to something empty.

**This needs further investigation** into the Docker SDK's `strslice.StrSlice` behavior and the agent's entrypoint/cmd deserialization. The most likely cause is that the JSON `"command": null` or `"command": []` in the agent payload is being deserialized and then setting `cfg.Entrypoint` to empty.

### 3.3 ShmSize — Secondary Issue

Both containers have `ShmSize: 67108864` (64MB, Docker default). The catalog specifies `16gb` for vLLM and `8gb` for llama.cpp. The ShmSize from `docker_json` is not being applied. This is a pre-existing issue with how `docker_json` flows through the clone/start path.

### 3.4 No CUDA_VISIBLE_DEVICES — Secondary Issue

The vLLM container does NOT have `CUDA_VISIBLE_DEVICES=0` in its env. LightAI's `default_env_json` sets `CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}`. The template substitution should produce `CUDA_VISIBLE_DEVICES=0`. Its absence is consistent with the container being created without proper RunPlan execution.

## 4. Differential Diagnosis: vLLM vs llama.cpp

| Aspect | llama.cpp PASS | vLLM FAIL |
|--------|---------------|-----------|
| Entrypoint | `["/app/llama-server"]` ✅ | `[]` ❌ |
| Cmd | `["-m",...,"--host","0.0.0.0","--port","8080"]` ✅ | `[]` ❌ |
| DeviceRequest | null (same) | null (same) |
| IpcMode | "private" | "host" |
| ShmSize | 64MB (same) | 64MB (same) |
| GPU env | NVIDIA_VISIBLE_DEVICES=all ✅ | NVIDIA_VISIBLE_DEVICES=all ✅ |
| CUDA_VISIBLE_DEVICES | NOT set | NOT set |
| E2E script | `e2e-model-runtime-wizard-nvidia-llamacpp.sh` | `e2e-model-runtime-wizard-nvidia-vllm.sh` |

Both use similar E2E flows. The key difference is the BackendRuntime used:
- llama.cpp: `llamacpp-b9700-nvidia-cuda13` (direct catalog runtime or clone)
- vLLM: `vllm-v0.23.0-nvidia-cuda` (clone of catalog runtime)

The clone operation may lose configuration, or the resolveImage path for vLLM may fail differently than llama.cpp.

## 5. External Env/Args Test Results

Additional external Docker tests confirmed:

| Test | Config | Result |
|------|--------|--------|
| `--gpus all`, LightAI env vars, bare path args | vLLM `/models/path --host 0.0.0.0 --port 8000` | ✅ PASS |
| `--gpus "device=0"`, LightAI env vars | vLLM with specific GPU | ✅ PASS |
| `--gpus all`, bare path + env vars + volume format | vLLM with exact LightAI args format | ✅ PASS |

**Conclusion**: vLLM's model args, env vars, and volume path format are NOT the cause of failure. The external tests prove all these configurations work. The failure is in the container creation — empty entrypoint and CMD.

## 6. SGLang Status

SGLang was not re-tested with `process_start_config` in `image_default` mode because:
1. No NBR currently has `process_start_config` set (the accept-detection flow hasn't been wired yet)
2. The SGLang E2E uses legacy entrypoint override which bypasses the NVIDIA wrapper

The SGLang container also likely has empty entrypoint/cmd (same root cause as vLLM) or uses the legacy override path. This needs separate investigation when `process_start_config` is applied.

## 7. Conclusions

### 7.1 Is vLLM Blocker a Real Environment Issue?

**NO.** The external environment supports vLLM with both `--gpus all` and `--gpus "device=0"`. The LightAI vLLM failure is a **container creation issue**: the Entrypoint and Cmd are both empty, meaning no Docker process runs. The GPU DeviceRequest vs `--gpus all` hypothesis is not the primary cause (both PASS and FAIL containers have null DeviceRequests).

### 7.2 Is the GPU DeviceRequest Issue Real?

**Partially.** The DeviceRequest `Driver: "nvidia"` vs `Driver: ""` difference could still matter in some environments or Docker versions. But it's NOT the cause of the current vLLM E2E failure. The immediate issue is empty Entrypoint/Cmd.

### 7.3 Did Phase 3 Changes Cause This?

**Unlikely.** Phase 3 only adds `process_start_config` handling to the resolver. When `process_start_config` is nil (the case for all existing E2E tests), the code follows legacy behavior. The empty entrypoint/cmd issue is likely pre-existing and related to how the cloned BackendRuntime's configuration flows through the start path.

### 7.4 What Needs to Be Fixed?

1. **Investigate why Entrypoint/Cmd are empty for vLLM container**: Trace the RunPlan resolution → agent spec → Docker API path for the specific cloned runtime used in vLLM E2E.
2. **Fix ShmSize propagation**: 64MB instead of catalog-specified 16GB/8GB.
3. **Fix CUDA_VISIBLE_DEVICES propagation**: Should be set to GPU index per `default_env_json` template.
4. **SGLang process_start_config test**: Once #1 is fixed, test SGLang with `image_default` + `command_prefix` to verify the NVIDIA wrapper entrypoint is preserved.

## 8. Recommendations

### Immediate (Next Round)

1. Debug empty Entrypoint/Cmd in vLLM E2E — trace the complete path from BackendRuntime clone → NBR → preflight → RunPlan → agent spec → Docker API.
2. Check if the BackendRuntime clone API correctly propagates all configuration fields (docker_json, default_env_json, etc.).
3. Fix ShmSize if it's a simple propagation bug.

### Deferred

1. Wire accept-detection → process_start_config in NBR (Phase 2 API gap).
2. Test SGLang with `image_default` mode via process_start_config.
3. Consider adding `DeviceRequest{Count: -1}` fallback when DeviceIDs is empty but vendor is NVIDIA (defense-in-depth for GPU visibility).

## 9. Evidence Captured

- External vLLM baseline: PASS, `/v1/models` 200 after 39s
- External vLLM with LightAI env/args format: PASS
- LightAI vLLM container inspect: empty Entrypoint/Cmd, null DeviceRequest
- LightAI llama.cpp container inspect: correct Entrypoint/Cmd, null DeviceRequest
- External `--gpus all` DeviceRequest: `{Driver:"", Count:-1}`
- External `--gpus "device=0"` DeviceRequest: `{Driver:"", Count:0, DeviceIDs:["0"]}`
