# RunPlan Parameter Source Audit & Fix Plan

Date: 2026-06-19
Branch: main
Starting commit: 4894b2e

## 1. RunPlan Parameter Generation Chain

```
BackendVersion.default_args_json          (Layer 1, template-substituted)
BackendVersion.default_backend_params_json (Layer 2, appended)
BackendRuntime.args_override_json          (Layer 3, appended)
mapParametersToArgs(params, ParameterDefs) (Layer 4, from parameters_json + defaults)
     ↓
deduplicateArgs(args)                     -- keeps FIRST occurrence, drops later ones
overridePortArg(args, appPort)            -- safety net for --port only
     ↓
Final RunPlan.Args
```

## 2. Known Issues Found

### Issue 1 (CRITICAL): deduplicateArgs gives Layer 1 priority over Layer 4
- `deduplicateArgs` keeps the FIRST `--flag value` pair and drops subsequent ones.
- Layer 1 (default_args_json) comes before Layer 4 (user parameters).
- So any flag in default_args_json ALWAYS wins over user settings.
- Currently masked because most flags aren't in both layers; only `--port` was hit and patched with `overridePortArg`.

### Issue 2 (MEDIUM): Key format mismatch in buildVarMap vs mapParametersToArgs
- `buildVarMap.getParam("served_model_name")` looks up `params["served_model_name"]`
- Falls back to ParameterDefs using `d.Name == "served_model_name"`
- But seed ParameterDefs use CLI-format names: `"--served-model-name"`, `"--tensor-parallel-size"`
- The fallback NEVER matches — template variables stay empty

### Issue 3 (MEDIUM): Missing ParameterDefs in seed data
- `gpu_memory_utilization`: no ParameterDef in any backend seed
- `enforce_eager`: no ParameterDef in any backend seed
- `trust_remote_code`: only in SGLang 0.4.6-compatible, not in vLLM or SGLang latest

### Issue 4 (LOW): value field silently dropped from ParameterDef
- Seed JSON has `"value"` field but Go struct lacks `json:"value"` tag

## 3. Systematic Fix: Reverse deduplicateArgs priority

Change `deduplicateArgs` to keep the LAST occurrence (highest priority), not the first.
This is a one-line conceptual change that fixes ALL user-override issues.

Also extend `overridePortArg` pattern to a generic `overrideArgFromService` that covers
all service_json→args mappings, not just `--port`.

## 4. Fix Plan

### Fix A: Reverse dedup priority (keep last, not first)
- In `deduplicateArgs`, change to keep LAST occurrence of each flag
- This makes Layer 4 (user params) naturally override Layer 1 (defaults)

### Fix B: Generic service-to-args bridge
- After dedup, apply all service_json fields to args
- Replace `overridePortArg` with `applyServiceArgs(args, service)` covering port, host

### Fix C: Fix getParam key matching in buildVarMap
- Normalize ParameterDef names: strip leading `--` and convert `-` to `_`

### Fix D: Add missing ParameterDefs to seed data
- vLLM v0.23.0: add gpu_memory_utilization, enforce_eager, trust_remote_code
- SGLang v0.5.13: add gpu_memory_utilization

### Fix E: Tests for all parameter propagation
- Matrix of vLLM/SGLang/llama.cpp × custom parameter values

## 5. Verification

```bash
go test ./... && go vet ./... && go build ./...
npm --prefix web run build && npm --prefix web test -- --runInBand
```
