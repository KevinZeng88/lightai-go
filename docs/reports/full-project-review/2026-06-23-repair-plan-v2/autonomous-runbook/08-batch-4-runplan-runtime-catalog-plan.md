# Batch 4: RunPlan / Runtime Config / Catalog — Detailed Plan

---

## Goal
Fix RunPlan resolver bugs. Focus on current bugs only.

## Key Functions (resolver.go)

### deduplicateArgs (line 470)
**Bug**: Boolean flags consumed as next flag's value.
**Fix**: Detect standalone flags (next token starts with `-` or is last).

### substituteVars (line 557-594)
**Bug**: Layer 5 (env_overrides, line 597-600) skips substituteVars.
**Fix**: Apply substituteVars to layer 5 values.

### mapParametersToArgs (line 509)
**Bug**: Required params silently skipped (line 533-537).
**Fix**: Return errors for missing required params.

### computeInputHash (line 933)
**Bug**: Missing AssignedGPUs, NodeRuntimeOverride, ProcessStartConfig.
**Fix**: Add missing fields.

### buildDeviceBinding (line 986)
**Status**: Dead code, never called by Resolve().
**Fix**: Remove function.

## Catalog YAML Issues

| File | Issue | Fix |
|------|-------|-----|
| configs/backend-catalog/runtimes/sglang/nvidia-cuda.yaml:6 | Stale version ref | Update to v0.5.13.post1 |
| configs/backend-catalog/versions/ollama/ollama-latest.yaml:15 | Raw JSON blob | Convert to structured YAML |
| configs/backend-catalog/runtimes/vllm/nvidia-cuda.yaml:24-27 | Dead keys (gpus, runtime) | Remove |

## Commits

1. `fix: deduplicateArgs handles boolean flags`
2. `fix: layer 5 env_overrides applies substituteVars`
3. `fix: required param validation returns errors`
4. `fix: computeInputHash includes all fields, remove dead buildDeviceBinding`
5. `fix: catalog YAML cleanup`

## Non-Regression

| Check | Method |
|-------|--------|
| vLLM RunPlan tests pass | go test ./internal/server/runplan/... |
| SGLang RunPlan tests pass | Same |
| llama.cpp RunPlan tests pass | Same |
| Boolean flags preserved | --trust-remote-code in args |
| Value flags preserved | --model /path in args |
| Required param error | Missing --model → error |
| Env substitution | {{MODEL_CONTAINER_PATH}} substituted |
