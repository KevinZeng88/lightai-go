# Manual Regression Root Cause Report

> Date: 2026-06-23
> Scope: Post-AUTORUN manual regression testing

---

## Problem 2.1: Empty Env Vars in Docker Command

**User observation**: SGLang/vLLM/llama.cpp Docker commands show empty env vars like `-e supported_formats=`, `-e blocked_architectures=`

**Root cause**: **Catalog/config bug** — `backend_versions.env_json` stores capability profile metadata (supported_formats, supported_tasks, etc.), not Docker env vars. The `buildEnv()` function (Layer 2) treats these as Docker env vars.

**Evidence**:
- DB query: `SELECT id, env_json FROM backend_versions WHERE id='llamacpp-b9700'` returns:
  ```json
  {"supported_formats":["gguf"],"supported_tasks":["chat","completion"],...}
  ```
- `resolver.go:562-570` (Layer 2) iterates `in.BackendVersion.Env` and adds all keys as Docker env vars
- Array values like `["gguf"]` are converted to empty strings via `fmt.Sprintf("%v", val)`

**Classification**: **Existing bug not covered by AUTORUN** — this was always wrong, AUTORUN didn't touch it.

**Impact**: All 3 backends (vLLM, SGLang, llama.cpp)

**Fix location**: `internal/server/db/db.go` — seed data puts capability metadata in `env_json` instead of `capabilities_json`. Also `resolver.go` should filter out non-string env values.

---

## Problem 2.2: Entrypoint Position in Docker Command

**User observation**: `--entrypoint` appears after image name in equivalent Docker command preview

**Root cause**: **Preview-only bug** — the equivalent Docker command renderer puts `--entrypoint` after the image, but Docker CLI expects it before the image. The actual Docker SDK execution uses `HostConfig.Entrypoint` field directly, so the real container starts correctly.

**Evidence**:
- Server log shows `entrypoint=[]` in the RunPlan resolve log, meaning entrypoint is empty
- The `--entrypoint vllm` in the preview comes from the args, not from the Entrypoint field
- Docker SDK `ContainerCreate` uses `cfg.Entrypoint` directly, not CLI args

**Classification**: **Preview-only bug** — real execution is unaffected.

**Impact**: All 3 backends (preview only)

**Fix location**: Equivalent Docker command renderer in Web UI or server-side preview generation.

---

## Problem 2.3: SGLang/llama.cpp Preflight Validation Failed

**User observation**: SGLang and llama.cpp now fail with `preflight validation failed` error

**Root cause**: **AUTORUN regression** — Batch 4 added required parameter validation in `mapParametersToArgs()` (line 529-532). This validation reports required parameters as missing even when they're already provided by `default_args` (Layer 1).

**Evidence**:
- Server log: `error=[{unknown required parameter "-m" missing map[]}]`
- The llama.cpp catalog defines `-m` (alias `--model`) as required in `default_args_schema`
- But `-m` is already provided in `default_args` (Layer 1) with value `{{model_container_path}}`
- `mapParametersToArgs` (Layer 4) checks if the deployment parameters contain the required param
- Deployment parameters are empty `{}`, so it reports the param as missing
- The resolver doesn't check if the param was already provided by Layer 1

**Classification**: **AUTORUN regression** — Batch 4's required parameter validation is too strict.

**Impact**: SGLang and llama.cpp (vLLM may also be affected depending on catalog)

**Fix location**: `internal/server/runplan/resolver.go:529-532` — required param check should skip if param already present in args from earlier layers.

**Requires rollback**: No — can be fixed by adjusting the validation logic.

---

## Problem 2.4: Parameter Editing Experience

**User observation**: Need parameter editing at model library, runtime config, and deployment levels

**Root cause**: **Requirement not implemented** — this is a feature request, not a bug.

**Classification**: **Requirement not implemented**

---

## Problem 2.5: GPU Memory Limit Configuration

**User observation**: No place to configure GPU memory limits

**Root cause**: **Requirement not implemented** — the catalog has `gpu-memory-utilization` for vLLM, `mem-fraction-static` for SGLang, and `n-gpu-layers` for llama.cpp, but no unified memory limit field.

**Classification**: **Requirement not implemented**

---

## Summary Table

| Problem | Classification | AUTORUN Regression? | Requires Rollback? |
|---------|---------------|--------------------|--------------------|
| 2.1 Empty env vars | Catalog/config bug | No | No |
| 2.2 Entrypoint position | Preview-only bug | No | No |
| 2.3 Preflight validation failed | AUTORUN regression (Batch 4) | **Yes** | No |
| 2.4 Parameter editing | Requirement not implemented | No | N/A |
| 2.5 GPU memory limit | Requirement not implemented | No | N/A |

---

## AUTORUN Regression Analysis

**Only 1 regression confirmed**: Problem 2.3 (preflight validation failed)

**Root cause**: Batch 4 added `mapParametersToArgs` error reporting for missing required parameters, but didn't account for parameters already provided by `default_args` (Layer 1).

**Fix**: In `mapParametersToArgs`, skip required param check if the param is already present in the args list from earlier layers. Or: pass the existing args list to `mapParametersToArgs` and check against it.

**Impact**: SGLang and llama.cpp deployments fail at preflight. vLLM may also be affected.

**Priority**: P0 — blocks deployment start for 2 of 3 backends.
