# 22 — llama.cpp GGUF Real RunPlan Regression

> Status: INVESTIGATING
> Scope: Why real RunPlan still has `-m /models/Qwen3.5-9B-Q4` despite Phase 2.6 fix
> Date: 2026-06-22
> Baseline: commit `6e9de80`

## WEB-AI-RC-005: Real RunPlan Still -m Points to Directory

### User Actual RunPlan

```bash
docker run -d --name lightai-81501e8a-b1d --ipc host --shm-size 8gb --gpus "device=0" \
  -v /home/kzeng/models/Qwen3.5-9B-Q4:/models/Qwen3.5-9B-Q4:ro \
  -e CUDA_VISIBLE_DEVICES=0 -p 8004:8080/tcp \
  ghcr.io/ggml-org/llama.cpp:server-cuda13 \
  -m /models/Qwen3.5-9B-Q4 --host 0.0.0.0 --port 8080
```

**Error**: `-m /models/Qwen3.5-9B-Q4` points to directory, not .gguf file.

### Model Detail Page Shows

```
格式: gguf
路径: /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

### Root Cause

**Three layers of stale data in the real DB:**

#### Layer 1: BackendVersion `default_args_json` (backend_versions table)

The real DB record for `llamacpp-b9700` still has:
```json
["-m","{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}"]
```

The source code was updated to `{{model_container_file}}` in the seed data, but `INSERT OR IGNORE` skipped the existing row. The UPDATE statement at line 1382 should fix this, but the server may not have been restarted to trigger the seed function re-run.

#### Layer 2: NodeBackendRuntime `config_snapshot_json` (node_backend_runtimes table)

The NBR for `llamacpp-b9700-nvidia-cuda13` has a frozen `config_snapshot_json` containing:
```json
"args_override_json": ["-m","{{MODEL_CONTAINER_PATH}}"]
```

This was frozen when the NBR was created, before the seed fix. In the resolver, this Layer 3 override (`BackendRuntime.ArgsOverride`) takes priority over Layer 1 (`BackendVersion.DefaultArgs`). Even if the BackendVersion is fixed, the NBR snapshot still provides the old args.

#### Layer 3: Deployment `config_snapshot_json` (model_deployments table)

When a deployment is created, the NBR's `config_snapshot_json` is merged into the deployment's `config_snapshot_json` (via `mergeNBRConfigSnapshot`). This frozen snapshot overrides the live BackendVersion/BackendRuntime values at deployment time.

### Why Phase 2.6 Fix Did Not Hit Real Scenario

The Phase 2.6 fix only changed:
1. Source code seed data (correct)
2. Resolver `buildVarMap` to add `MODEL_CONTAINER_FILE` (correct)
3. Runplan test fixtures (correct)

But did NOT:
1. Force-update existing DB records (`backend_versions`, `node_backend_runtimes`, `model_deployments`)
2. Handle the NBR frozen snapshot overriding the fixed BackendVersion args
3. Handle the deployment frozen snapshot overriding the fixed NBR/BackendVersion args

### Legacy Cleanup Policy

Per user directive, no backward compatibility for old runtime args. Policy:
1. All built-in `backend_versions` with old `{{model_container_path}}` for llama.cpp `-m` are invalid — force-update them.
2. All `node_backend_runtimes` with old `args_override_json` using `{{MODEL_CONTAINER_PATH}}` for llama.cpp are invalid — force-update `config_snapshot_json`.
3. All `model_deployments` with frozen snapshots using the old args are stale — mark for regeneration.
4. No fallback logic in resolver to "correct" old arg templates — fix the source data.

### Acceptance Criteria

1. Real RunPlan MUST contain `-m /models/.../Qwen3.5-9B-Q4_K_M.gguf` (file path)
2. Real RunPlan MUST NOT contain `-m /models/Qwen3.5-9B-Q4` (directory path)
3. Test covers directory relative_path + GGUF artifact path → correct file -m
4. Existing DB records are force-updated by migration
