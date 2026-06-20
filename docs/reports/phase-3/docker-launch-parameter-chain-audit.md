# Docker Launch Parameter Chain Audit

> Status: AUDIT_REPORT
> Date: 2026-06-21
> Scope: BackendRuntime → NBR → RunPlan → Agent Docker Create full parameter chain
> Purpose: Document findings; no implementation decisions made yet.

## 1. Background

vLLM and SGLang real smoke tests initially failed inside LightAI-managed Docker containers, but external `docker run` with the same images and GPU passed. The root cause is likely in LightAI's RunPlan / Agent Docker Create parameter chain, which warranted a full-chain audit.

This document records audit findings only. No design decisions, no implementation, and no code changes are included.

## 2. External Baseline (PASS)

Both external direct `docker run` commands succeed on the local WSL2/NVIDIA Docker runtime, confirming GPU availability and image correctness.

### vLLM
```bash
docker run --rm --name lightai-ext-vllm-test \
  --gpus all \
  --ipc=host \
  -p 18004:8000 \
  -v /home/kzeng/models/Qwen3-0.6B-Instruct-2512:/models/qwen:ro \
  vllm/vllm-openai:latest \
  --model /models/qwen \
  --host 0.0.0.0 \
  --port 8000
```

### SGLang
```bash
docker run --rm --name lightai-ext-sglang-test \
  --gpus all \
  --ipc=host \
  -p 18003:30000 \
  -v /home/kzeng/models/Qwen3-0.6B-Instruct-2512:/models/qwen:ro \
  lmsysorg/sglang:latest \
  python3 -m sglang.launch_server \
  --model-path /models/qwen \
  --host 0.0.0.0 \
  --port 30000
```

Key observations from external baselines:
- Both use `--gpus all` (no device filtering).
- vLLM entrypoint is NOT overridden — Docker uses the image's built-in ENTRYPOINT.
- SGLang passes `python3 -m sglang.launch_server` as CMD (NOT as ENTRYPOINT override).
- Neither sets `CUDA_VISIBLE_DEVICES` or vendor-specific env vars.

## 3. LightAI Real Smoke (E2E Evidence)

### 3.1 Matrix Postfix (2026-06-19) — ALL PASS

| Backend | Result | Evidence |
|---------|--------|----------|
| vLLM default | PASS | `/v1/models` returned 200 after ~86s startup |
| vLLM modified (max_model_len=4096) | PASS | |
| SGLang default | PASS | `/v1/models` returned 200 |
| SGLang modified (--tp 1) | PASS | |
| llama.cpp default | PASS | |
| llama.cpp modified (--ctx-size 2048) | PASS | |

Evidence: `docs/reports/model-runtime-node-wizard/e2e-matrix-matrix-postfix-20260619032917/`

### 3.2 Real Smoke (2026-06-20) — vLLM & SGLang FAIL

| Backend | Error |
|---------|-------|
| vLLM | `RuntimeError: Failed to infer device type`, `No CUDA runtime is found` |
| SGLang | SGLang/Triton platform/device detection failure |
| llama.cpp | PASS |

Evidence: `docs/reports/phase-3/open-issues-closeout.md` (E2E-002, E2E-003)

### 3.3 Time Gap Analysis

The matrix postfix (June 19) used a different GPU UUID (`a16cf913-8923-46e8-b76f-ab8d5bd34379`) and potentially different BackendRuntime configuration than the June 20 real smoke tests (GPU `be899678-7a6b-4391-9001-b56172d1f505`). This may explain the pass/fail discrepancy.

## 4. Architecture: Two Coexisting Catalog Systems

| System | Location | Status |
|--------|----------|--------|
| `seedBuiltInBackends` | `db.go:1208` | Deprecated at line 1386 |
| `seedTargetBackendCatalog` | `db.go:1331` | Current authoritative |

The old seed has hardcoded legacy versions (vLLM 0.8.5/0.10.0, SGLang 0.4.6/0.5.0, llama.cpp b4817). The target catalog has current versions (vLLM v0.23.0, SGLang v0.5.12.post1/v0.5.13.post1, llama.cpp b9700) and deprecates the old versions.

## 5. BackendRuntime Fields (Target Catalog)

**Table**: `backend_runtimes`
**Struct**: `internal/server/models/runtime.go:5` (`BackendRuntime`)

| Field | Type | vLLM Value | SGLang Value | llama.cpp Value |
|-------|------|------------|-------------|-----------------|
| `image_name` | string | `vllm/vllm-openai:v0.23.0` | `lmsysorg/sglang:v0.5.13.post1-cu129-runtime` | `ghcr.io/ggml-org/llama.cpp:server-cuda13` |
| `entrypoint_override_json` | JSON array | `[]` | `[]` | `[]` |
| `args_override_json` | JSON array | `[]` | `[]` | `[]` |
| `default_env_json` | JSON object | `{"CUDA_VISIBLE_DEVICES":"{{vendor_visible_devices}}"}` | same | same |
| `docker_json` | JSON object | `{"gpu_visible_env_key":"CUDA_VISIBLE_DEVICES","ipc_mode":"host","shm_size":"16gb"}` | `{"ipc_mode":"host","shm_size":"32gb"}` | `{"ipc_mode":"host","shm_size":"8gb"}` |
| `model_mount_json` | JSON object | `{"container_path":"/models","readonly":true}` | same | same |

**Finding**: All three catalog runtimes have empty `entrypoint_override_json` and `args_override_json`. Entrypoint and args decisions flow from BackendVersion, not BackendRuntime.

## 6. BackendVersion Defaults (Entrypoint/Args Source)

**Table**: `backend_versions`
**Source**: `db.go:1355-1367` (target catalog seed)

### vLLM v0.23.0
```
default_entrypoint_json:     ["vllm","serve"]                       ← explicitly overrides image ENTRYPOINT
default_args_json:           ["{{model_container_path}}"]           ← BARE positional model path (no --model flag)
default_backend_params_json: []
parameter_defs_json:         [
  {"name":"--model","value":"{{MODEL_CONTAINER_PATH}}"},            ← VALUE silently dropped by Go struct
  {"name":"--host","default":"0.0.0.0"},
  {"name":"--port","default":"8000"},
  {"name":"--served-model-name","optional":true},
  {"name":"--tensor-parallel-size","optional":true},
  {"name":"--max-model-len","optional":true},
  {"name":"--gpu-memory-utilization","optional":true},
  {"name":"--enforce-eager","optional":true},
  {"name":"--trust-remote-code","optional":true}
]
default_container_port:      8000
```

### SGLang v0.5.13.post1
```
default_entrypoint_json:     ["python3","-m","sglang.launch_server"] ← note: python3 (old seed used python)
default_args_json:           ["--model-path","{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}"]
default_backend_params_json: []
default_container_port:      30000
```

### llama.cpp b9700
```
default_entrypoint_json:     []                                      ← EMPTY — Docker preserves image ENTRYPOINT
default_args_json:           ["-m","{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}"]
default_backend_params_json: []
default_container_port:      8080
```

## 7. ParameterDef Struct — Value Field Gap

**File**: `internal/server/runplan/resolver.go:49-56`
```go
type ParameterDef struct {
    Name     string      `json:"name"`
    CliName  string      `json:"cli_name"`
    Alias    string      `json:"alias"`
    Type     string      `json:"type"`
    Default  interface{} `json:"default"`
    Required bool        `json:"required"`
}
```

The struct has no `Value` field. The target catalog vLLM `--model` parameter_def uses `"value":"{{MODEL_CONTAINER_PATH}}"` (not `"default"`), which is silently dropped during `json.Unmarshal`. Consequently, `mapParametersToArgs()` never auto-generates `--model` from parameter_defs.

## 8. RunPlan Args Generation (4 Layers)

`buildArgs()` in `resolver.go:326-371`:

```
Layer 1: BackendVersion.DefaultArgs        ← template-substituted via {{var}}
Layer 2: BackendVersion.DefaultBackendParams ← template-substituted
Layer 3: BackendRuntime.ArgsOverride       ← template-substituted, append only
Layer 4: Deployment.Parameters → mapParametersToArgs ← via ParameterDefs
↓
deduplicateArgs()  ← removes duplicate --flag value pairs (keeps last)
↓
applyServiceArgs() ← overrides --port with AppPort
```

### vLLM Args Example

With catalog vLLM v0.23.0:
- Layer 1: `["/models/Qwen3-0.6B-Instruct-2512"]` (bare positional path)
- Layer 2: `[]`
- Layer 3: `[]`
- Layer 4: `["--host","0.0.0.0","--port","8000"]` (from ParameterDef defaults — `--model` skipped because Value is dropped)
- Combined: `["/models/Qwen3-0.6B-Instruct-2512","--host","0.0.0.0","--port","8000"]`
- Dedup: no duplicates to remove
- Service args: `--port` overridden if AppPort is set

### Deprecated Seed vs Target Catalog

The deprecated vLLM v0.8.5 `default_args_json` is:
```json
["{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}","--served-model-name","{{served_model_name}}","--max-model-len","{{max_model_len}}","--gpu-memory-utilization","{{gpu_memory_utilization}}"]
```

This includes ALL flags inline (no dependency on parameter_defs for --host/--port), unlike the target catalog version where only the bare model path is in default_args.

## 9. Entrypoint Resolution

`resolver.go:192-196`:
```go
entrypoint := in.BackendVersion.DefaultEntrypoint
if len(in.BackendRuntime.EntrypointOverride) > 0 {
    entrypoint = in.BackendRuntime.EntrypointOverride
}
```

| Backend | BackendVersion DefaultEntrypoint | BackendRuntime Override | Final Entrypoint |
|---------|----------------------------------|------------------------|-----------------|
| vLLM | `["vllm","serve"]` | `[]` (empty) | `["vllm","serve"]` |
| SGLang | `["python3","-m","sglang.launch_server"]` | `[]` (empty) | `["python3","-m","sglang.launch_server"]` |
| llama.cpp | `[]` (empty) | `[]` (empty) | `[]` (empty) |

### Docker API Behavior

`docker_real.go:77-87`:
```go
if len(opts.Entrypoint) > 0 {
    cfg.Entrypoint = strslice.StrSlice(opts.Entrypoint)
}
```

- llama.cpp: `entrypoint=[]` → `len==0` → Docker preserves image ENTRYPOINT `["/app/llama-server"]`
- vLLM: `entrypoint=["vllm","serve"]` → Docker overrides ENTRYPOINT
- SGLang: `entrypoint=["python3","-m","sglang.launch_server"]` → Docker overrides ENTRYPOINT

## 10. Agent Docker Create Flow

Chain: `ResolvedRunPlan` → `AgentRunSpec` → `ContainerCreateOptions` → Docker API

```
ResolvedRunPlan.Entrypoint → AgentRunSpec.Docker.Command → ContainerCreateOptions.Entrypoint → cfg.Entrypoint
ResolvedRunPlan.Args        → AgentRunSpec.Docker.Args    → ContainerCreateOptions.Command    → cfg.Cmd
```

Key files:
- `internal/server/runplan/resolver.go` — Resolve() builds ResolvedRunPlan
- `internal/agent/runtime/runplan_adapter.go` — ConvertRunplanToAgentSpec
- `internal/agent/runtime/docker.go` — buildCreateOptions
- `internal/agent/runtime/docker_real.go` — ContainerCreate (actual Docker API call)

## 11. GPU Assignment Chain

### Server-side resolver (`resolver.go:229-233`):
```go
gpuIDs = append(gpuIDs, fmt.Sprintf("%d", g.Index))  // GPU INDEX numbers
env[gpuVisibleKey] = strings.Join(gpuIDs, ",")       // e.g., CUDA_VISIBLE_DEVICES=0
```

### Agent spec builder (`deployment_lifecycle_handlers.go:1103`):
Uses `pf.placement.GPUIds` (UUIDs from placement).

### Agent Docker (`docker.go:457-463`):
```go
if spec.Vendor == "nvidia" && len(spec.GPUDeviceIDs) > 0 {
    dr := DeviceRequest{
        Driver:       "nvidia",
        Capabilities: [][]string{{"gpu"}},
        DeviceIDs:    spec.GPUDeviceIDs,  // GPU UUIDs → Docker DeviceRequest
    }
}
```

### External baseline uses:
- `--gpus all` — all GPUs visible, no filtering

### LightAI uses:
- `DeviceRequest` with specific GPU UUIDs + `CUDA_VISIBLE_DEVICES=<index>`

**This is the most significant structural difference between LightAI and the external baseline.**

## 12. RunPlan vs External Baseline: Key Differences

### vLLM
| Aspect | LightAI RunPlan | External Baseline | Gap |
|--------|----------------|-------------------|-----|
| Image | `vllm/vllm-openai:latest` | `vllm/vllm-openai:latest` | None |
| Entrypoint | `["vllm","serve"]` (override) | Image default (preserve) | LightAI explicitly overrides |
| Args | `["/path","--host","0.0.0.0","--port","8000"]` | `--model /path --host 0.0.0.0 --port 8000` | Bare positional vs `--model` flag |
| GPU | DeviceRequest(specific UUIDs) | `--gpus all` | **Different** |
| Env | `CUDA_VISIBLE_DEVICES=0, VLLM_USE_MODELSCOPE=false` | None | Extra env vars |
| SHM | `16gb` | Docker default (64mb) | LightAI sets explicitly |

### SGLang
| Aspect | LightAI RunPlan | External Baseline | Gap |
|--------|----------------|-------------------|-----|
| Entrypoint | `["python3","-m","sglang.launch_server"]` (override) | Image default + CMD override | LightAI uses ENTRYPOINT override |
| GPU | DeviceRequest(specific UUIDs) | `--gpus all` | **Different** |

### llama.cpp
| Aspect | LightAI RunPlan | Notes |
|--------|----------------|-------|
| Entrypoint | `[]` (empty → preserve) | Docker uses image ENTRYPOINT `["/app/llama-server"]` |
| Args | `["-m","/path","--host","0.0.0.0","--port","8080"]` | Well-formed, matches external pattern |
| GPU | DeviceRequest(specific UUIDs) | Same GPU assignment as vLLM/SGLang |

## 13. llama.cpp Success Factors

1. **Empty entrypoint `[]`** — preserves image's native ENTRYPOINT; no override risk.
2. **Well-formed args** — `-m`, `--host`, `--port` all in `default_args_json`.
3. **Static C++ binary** — simpler runtime, less sensitive to CUDA enumeration issues.
4. **Explicit `-ngl` parameter** — controls GPU offloading with graceful fallback.

## 14. Potential Issues (For Design Discussion)

### 14.1 vLLM `default_args_json` Bare Path
The target catalog vLLM v0.23.0 `default_args_json` is `["{{model_container_path}}"]` — bare positional path. External baseline uses `--model /path`. The deprecated v0.8.5 seed includes `--host`, `--port`, `--served-model-name`, `--max-model-len`, `--gpu-memory-utilization` in `default_args_json`. The target catalog relies on `parameter_defs` for these.

**Status**: Design discussion point. Not decided.

### 14.2 Entrypoint Policy Not Unified
- vLLM: explicitly overrides ENTRYPOINT with `["vllm","serve"]`
- SGLang: explicitly overrides ENTRYPOINT with `["python3","-m","sglang.launch_server"]`
- llama.cpp: empty entrypoint → Docker preserves image ENTRYPOINT

There is no mechanism to say "preserve the image's ENTRYPOINT" uniformly across backends.

**Status**: Design discussion point. Not decided.

### 14.3 GPU DeviceRequest Specificity
LightAI uses specific GPU UUIDs (`DeviceRequest{DeviceIDs: [...]}`). External baseline uses `--gpus all`. In WSL2/Docker Desktop, device-pinned GPU access may trigger different CUDA enumeration behavior than all-GPU access.

**Status**: Needs environmental verification. Not decided.

### 14.4 ParameterDef Value Field Gap
The `ParameterDef` Go struct (`resolver.go:49`) has no `Value` field. The catalog seed `--model` parameter_def uses `"value"` which is silently dropped.

**Status**: Design discussion point. Not decided.

### 14.5 Seed Changes Don't Propagate to Existing NBRs/Deployments
The `config_snapshot_json` on NBRs and ModelDeployments is frozen at creation time. Changes to BackendRuntime or BackendVersion seed data only affect NEW NBRs and deployments.

**Status**: Known architectural behavior. Not a bug — snapshots are intentionally frozen.

## 15. No `launch_spec` Exists

The current `docker_json` has no mechanism to express:
- Entrypoint policy (preserve image ENTRYPOINT vs override vs clear)
- Launcher command handling
- Args source preference
- Shell-mode wrapping

This is a potential design space but NOT implemented.

## 16. NodeBackendRuntime Fields

**Table**: `node_backend_runtimes` (`db.go:1153`)

| Key fields | Meaning |
|-----------|---------|
| `backend_runtime_id` | FK to BackendRuntime |
| `node_id` | Target node |
| `config_snapshot_json` | Frozen BR config at enable-on-node time |
| `image_ref` | Actual Docker image on node |
| `probe_results_json` | Image capability probe results |
| `status` | ready / needs_check / missing_image / unknown / failed |

### Clone/Snapshot Flow

```
seedTargetBackendCatalog()
  → INSERT backend_versions
  → INSERT backend_runtimes

Enable-on-node:
  upsertNodeBackendRuntime() [runtime_handlers.go:662]
    → buildRuntimeConfigSnapshot() [line 798]
    → Freeze: image_name, entrypoint_override_json, args_override_json,
              default_env_json, docker_json, model_mount_json,
              health_check_override_json, version_snapshot_json

Create Deployment:
  buildDeploymentRuntimeSnapshot() [deployment_lifecycle_handlers.go:59]
  mergeNBRConfigSnapshot() [line 88]
  → Stored as ModelDeployment.config_snapshot_json

Start/Dry-run:
  applyDeploymentConfigSnapshot() [line 922]
  → Override live BR values with frozen snapshot
  → runplan.Resolve(input)
```

## 17. Conclusion

The parameter chain is functional. The audit identifies the following areas as potential contributors to the vLLM/SGLang smoke test failures, ranked by likelihood:

1. **GPU DeviceRequest specificity** (H1) — `DeviceRequest{DeviceIDs: [...]}` vs `--gpus all`. vLLM/SGLang have complex CUDA/Triton stacks sensitive to GPU enumeration; llama.cpp tolerates it better.
2. **vLLM model arg format** (H2) — Bare positional `["/path"]` vs `--model /path`.
3. **Entrypoint override behavior** (H3) — vLLM/SGLang explicitly set entrypoint while llama.cpp preserves image default.

**None of these findings constitute confirmed bugs.** They are design discussion points requiring further investigation and design before any code change.

## 18. Implementation Status

**No code changes have been made based on this audit.** All findings are for discussion only.

Relevant evidence directories:
- `docs/reports/model-runtime-node-wizard/e2e-matrix-matrix-postfix-20260619032917/`
- `docs/reports/model-runtime-node-wizard/e2e-matrix-20260620224523/`
- `docs/reports/model-runtime-node-wizard/e2e-sglang-20260620224604/`
- `docs/reports/model-runtime-node-wizard/e2e-llamacpp-20260620224306/`
- `docs/reports/phase-3/open-issues-closeout.md`
