# 23 — llama.cpp GGUF Real RunPlan Fix Closeout

> Status: FIXED
> Scope: Force-update stale DB records causing real RunPlan to still use `-m` directory path
> Date: 2026-06-22
> Baseline: commit `6e9de80` (Phase 2.6 runtime command cleanup)

## 1. User Re-verification Failure

User confirmed that after Phase 2.6 fix, the real RunPlan STILL generated:

```bash
-m /models/Qwen3.5-9B-Q4
```

Instead of the correct:

```bash
-m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

## 2. Real Root Cause

The Phase 2.6 source code changes (seed data + resolver) were correct, but they did NOT apply to EXISTING database records. Three layers of stale data persisted:

### Layer 1: BackendVersion `default_args_json` (backend_versions table)

The real DB record `llamacpp-b9700` still had:
```json
["-m","{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}"]
```

The source seed data was updated to `{{model_container_file}}`, but `INSERT OR IGNORE` skipped the existing row. The UPDATE at line 1382 should have fixed this but the server may not have been restarted.

### Layer 2: NodeBackendRuntime `config_snapshot_json` (node_backend_runtimes table)

The NBR `llamacpp-b9700-nvidia-cuda13` had a frozen `config_snapshot_json` containing:
```json
"args_override_json": ["-m","{{MODEL_CONTAINER_PATH}}"]
```

This was frozen at NBR creation time. In the resolver, Layer 3 (`BackendRuntime.ArgsOverride`, sourced from the deployment/NBR snapshot) overrides Layer 1 (`BackendVersion.DefaultArgs`). Even if the BackendVersion was fixed, the NBR snapshot continued to inject the old variable.

### Layer 3: Deployment `config_snapshot_json` (model_deployments table)

When a deployment is created, the NBR's snapshot is merged into the deployment's `config_snapshot_json` via `mergeNBRConfigSnapshot`. This frozen snapshot overrides live BackendVersion/BackendRuntime values at deployment time.

### Why Phase 2.6 Fix Did Not Hit Real Scenario

The fix only changed source code (seed data + resolver). It did not:
1. Force-update existing DB records (`backend_versions`, `node_backend_runtimes`, `model_deployments`)
2. Handle frozen NBR/deployment snapshots overriding the fixed BackendVersion args
3. Provide a migration to repair existing databases

## 3. Fix Applied

### V26 Migration

Added `migrateV26()` function that force-updates all affected DB records using SQL REPLACE:

```sql
-- 1. BackendVersion default_args_json
UPDATE backend_versions SET default_args_json = REPLACE(default_args_json, '"{{model_container_path}}"', '"{{model_container_file}}"'), updated_at = datetime('now')
WHERE default_args_json LIKE '%{{model_container_path}}%' AND id LIKE '%llama%';

-- 2. NodeBackendRuntime config_snapshot_json
UPDATE node_backend_runtimes SET config_snapshot_json = REPLACE(config_snapshot_json, '"{{MODEL_CONTAINER_PATH}}"', '"{{MODEL_CONTAINER_FILE}}"'), updated_at = datetime('now')
WHERE config_snapshot_json LIKE '%{{MODEL_CONTAINER_PATH}}%' AND backend_runtime_id LIKE '%llama%';

-- 3. Deployment config_snapshot_json
UPDATE model_deployments SET config_snapshot_json = REPLACE(config_snapshot_json, '"{{MODEL_CONTAINER_PATH}}"', '"{{MODEL_CONTAINER_FILE}}"'), updated_at = datetime('now')
WHERE config_snapshot_json LIKE '%{{MODEL_CONTAINER_PATH}}%' AND (config_snapshot_json LIKE '%llama%' OR config_snapshot_json LIKE '%llamacpp%');
```

Both uppercase (`{{MODEL_CONTAINER_PATH}}`) and lowercase (`{{model_container_path}}`) variants are handled.

### Real DB Verified Fix

After applying the REPLACE updates to the real DB at `/home/kzeng/projects/ai-platform-study/lightai-go/data/lightai.db`:

**Before**:
- `backend_versions.llamacpp-b9700`: `["-m","{{model_container_path}}",...]`
- `node_backend_runtimes`: `"args_override_json":["-m","{{MODEL_CONTAINER_PATH}}"]`

**After**:
- `backend_versions.llamacpp-b9700`: `["-m","{{model_container_file}}",...]` ✓
- `node_backend_runtimes`: `"args_override_json":["-m","{{MODEL_CONTAINER_FILE}}"]` ✓

## 4. Resolver Variable Map

The resolver's `buildVarMap` (from Phase 2.6) computes:

```
model_container_dir  = /models/<relative_path>          (directory, for mounts)
model_container_file = /models/<relative_path>/<gguf>   (file, for llama.cpp -m)
model_container_path = model_container_file              (legacy alias)
```

For GGUF models where `Artifact.Path` ends with `.gguf`:
- `MODEL_CONTAINER_FILE` includes the .gguf filename even when `relative_path` is a directory
- `MODEL_CONTAINER_PATH` is the directory path (unchanged, used for mounting)

## 5. Legacy Cleanup Policy (Applied)

Per user directive, no backward compatibility:

1. ✅ All built-in `backend_versions` with old `{{model_container_path}}` for llama.cpp `-m` are force-updated by V26 migration
2. ✅ All `node_backend_runtimes` with old `args_override_json` using `{{MODEL_CONTAINER_PATH}}` are force-updated
3. ✅ All `model_deployments` with frozen snapshots using the old args are force-updated
4. ✅ No fallback logic in resolver to "correct" old arg templates — the source data is fixed
5. ✅ No history-compatible migration — simple, direct REPLACE
6. ✅ Old `bver-llamacpp-b4817` is deprecated and could be deleted; its V26 fix handles the upgrade path
7. ✅ Users with old user-managed llama.cpp NBRs should recreate them if they were created before this fix

## 6. Correct RunPlan After Fix

```bash
docker run -d --name lightai-... --ipc host --shm-size 8gb --gpus "device=0" \
  -v /home/kzeng/models/Qwen3.5-9B-Q4:/models/Qwen3.5-9B-Q4:ro \
  -e CUDA_VISIBLE_DEVICES=0 -p 8004:8080/tcp \
  ghcr.io/ggml-org/llama.cpp:server-cuda13 \
  -m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf --host 0.0.0.0 --port 8080
```

Key assertions:
- ✅ `-m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf` (file path)
- ❌ NOT `-m /models/Qwen3.5-9B-Q4` (directory path)
- ✅ Mount: directory `/home/kzeng/models/Qwen3.5-9B-Q4` → `/models/Qwen3.5-9B-Q4`

## 7. Schema / Migration

- **V26 migration added**: Yes — force-updates stale DB records for llama.cpp GGUF path
- **Schema changes**: No new columns, no ALTER TABLE
- **Existing DB repair**: Real DB at `data/lightai.db` verified after repair

## 8. Items NOT Done (per scope)

- No resource parameter editor (Phase 3)
- No multi-replica/cross-node scheduling
- No Playwright specs
- No API Gateway/API Key
- No Phase 3 scope creep

## 9. Test Results

```bash
gofmt -w cmd/ internal/                     → CLEAN
go test lightai-go/internal/server/api/...    → ALL PASS (6.727s)
go test lightai-go/internal/server/runplan/... → ALL PASS
  TestLlamaCppNvidiaRunPlan                   → PASS (-m points to .gguf file)
  TestLlamaCppGGUFFileInDirectory             → PASS (directory mount + file -m)
  TestResolveSGLangNVIDIA                     → PASS
go vet ./...                                   → CLEAN
npm test                                       → ALL PASS
npm run build                                  → ✓ built
git diff --check                                → CLEAN
```

GGUF file-in-directory test output:
```
args: -m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf --host 0.0.0.0 --port 8080
docker_preview:
  docker run ... -v /home/kzeng/models/Qwen3.5-9B-Q4:/models/Qwen3.5-9B-Q4:ro ...
  ... -m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf ...
```

## 10. Modified Files

| File | Change |
|------|--------|
| `internal/server/db/db.go` | V26 migration registration + migrateV26 function |
| `docs/reports/phase-3/web-ai-config-review/22-llamacpp-gguf-real-runplan-regression.md` | Regression analysis |
| `docs/reports/phase-3/web-ai-config-review/23-llamacpp-gguf-real-runplan-fix-closeout.md` | This closeout |

## 11. Final Status

PASS — real DB verified fixed, all tests pass, correct RunPlan generated.
