# 20 — Phase 2.6 Runtime Command Cleanup Issues

> Status: DRAFT
> Scope: Runtime command generation cleanup from live deployment validation
> Date: 2026-06-22
> Baseline: commit `36b2ee9` (Phase 2.5 live workflow fixes)

## Issue Index

| ID | Issue | Severity | Status |
|----|-------|----------|--------|
| WEB-AI-RC-001 | llama.cpp GGUF actual RunPlan still -m points to directory | P0 | DRAFT |
| WEB-AI-RC-002 | llama.cpp LLAMA_ARG_HOST env + --host arg duplicate warning | P2 | DRAFT |
| WEB-AI-RC-003 | Model test prompt too complex, models explain "Ping" | P1 | DRAFT |
| WEB-AI-RC-004 | SGLang entrypoint uses deprecated python -m sglang.launch_server | P2 | DRAFT |

---

## WEB-AI-RC-001: llama.cpp GGUF Actual RunPlan Still -m Points to Directory

### User Phenomenon

Docker log still shows:
```
load_model: loading model '/models/Qwen3.5-9B-Q4'
gguf_init_from_reader: failed to read magic
failed to load model from /models/Qwen3.5-9B-Q4
```

The actual RunPlan still generates `-m /models/Qwen3.5-9B-Q4` (directory) instead of `-m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf` (file).

### Root Cause

The Phase 2.5 fix only changed the scan proxy to preserve candidate file paths for NEW scans. It did NOT fix:

1. **Existing DB records**: `model_locations.relative_path` still stores the directory name (e.g., `Qwen3.5-9B-Q4`) from old scan proxy behavior
2. **Resolver uses relative_path directly**: `modelRelativePath()` returns `Artifact.RelativePath` verbatim, which is the directory name from DB
3. **No GGUF-specific logic**: The resolver doesn't check if the model format is "gguf" and adjust the container model path to include the .gguf filename

### Fix Plan

In the resolver's `buildVarMap`, for GGUF format models:
1. Compute `MODEL_CONTAINER_FILE` that includes the .gguf filename from `ArtifactInfo.Path`
2. If artifact path ends with `.gguf` but relative_path doesn't, append the artifact's filename to the container file path
3. Update llama.cpp `default_args_json` to use `{{model_container_file}}` for the `-m` flag
4. Keep existing mount logic unchanged (directory mount)

### Acceptance

- llama.cpp + GGUF file location → `-m` points to container file path (not directory)
- Mount still uses directory
- Test covers: file location → -m points to .gguf file

---

## WEB-AI-RC-002: llama.cpp LLAMA_ARG_HOST + --host Duplicate

### User Phenomenon

Docker log shows:
```
warn: LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host
```

### Root Cause

The llama.cpp Docker image (`ghcr.io/ggml-org/llama.cpp:server-cuda13`) has a built-in `LLAMA_ARG_HOST` environment variable. The platform also adds `--host 0.0.0.0` in `default_args_json`. The command line `--host` correctly overrides the env, but the image emits a warning.

No code in the LightAI platform sets `LLAMA_ARG_HOST` — it comes from the Docker image.

### Fix

This is a benign warning (cosmetic only). Document in closeout that the image provides `LLAMA_ARG_HOST` as default env, and the platform's `--host 0.0.0.0` correctly overrides it. No code change needed.

### Acceptance

- Warning documented as image-originated, not platform-injected
- No code change required

---

## WEB-AI-RC-003: Model Test Prompt Too Complex

### User Phenomenon

Model responds to test prompt "ping" with:
```
Ping 是一种网络测试工具，用于...
```

The model explains what Ping is instead of giving a simple response. The test prompt is too ambiguous.

### Root Cause

The test prompt is just `"ping"` which models interpret as a question about the network tool. The prompt should be directive, asking for a specific short response.

### Fix

Change the default test prompt to request a simple "pong" response:
- Chat mode: system message "Reply with exactly one word: pong" + user message "ping"
- Completion mode: "Reply with exactly one word: pong"
- Keep max_tokens=8, temperature=0

Update backend HandlerModelInstanceTest and frontend ModelInstancesPage.vue.

### Acceptance

- Test prompt is shorter and more directive
- Default max_tokens is small (8)
- Models no longer tend to explain "Ping"
- Test success condition remains API HTTP success

---

## WEB-AI-RC-004: SGLang Entrypoint Deprecated Warning

### User Phenomenon

SGLang now runs successfully, but logs show:
```
'python -m sglang.launch_server' is still supported, but 'sglang serve' is the recommended entrypoint.
```

### Root Cause

The SGLang BackendVersion seed data uses `["python3", "-m", "sglang.launch_server"]` as the entrypoint. The SGLang project now recommends `sglang serve` as the standard entrypoint.

### Fix

Change the SGLang BackendVersion `default_entrypoint_json` from `["python3","-m","sglang.launch_server"]` to `["sglang","serve"]` in the target catalog seed data. This is low-risk because:
- SGLang images v0.5.12+ include the `sglang` CLI
- `sglang serve` is the documented recommended way to start SGLang
- The change only affects new deployments (existing deployments use frozen config snapshots)

Update the SGLang runplan test accordingly.

### Acceptance

- SGLang RunPlan uses `sglang serve` as entrypoint
- Test updated and passes
- First container arg is still not `--model-path`
