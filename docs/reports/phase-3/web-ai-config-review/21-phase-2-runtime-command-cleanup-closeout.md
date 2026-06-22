# 21 — Phase 2.6 Runtime Command Cleanup Closeout

> Status: FIXED
> Scope: Runtime command generation cleanup from live deployment validation
> Date: 2026-06-22
> Baseline: commit `36b2ee9` (Phase 2.5 live workflow fixes)

## 1. Issues Fixed

| ID | Issue | Status | Root Cause |
|----|-------|--------|------------|
| WEB-AI-RC-001 | llama.cpp GGUF -m still points to directory | FIXED | Resolver used `model_container_path` (always directory from DB); no GGUF-aware file path variable |
| WEB-AI-RC-002 | llama.cpp LLAMA_ARG_HOST env + --host duplicate | DOCUMENTED | Image provides LLAMA_ARG_HOST; platform's --host correctly overrides; benign |
| WEB-AI-RC-003 | Model test prompt too complex | FIXED | Default prompt was "ping" — models interpreted it as network tool question |
| WEB-AI-RC-004 | SGLang entrypoint deprecated warning | FIXED | Switched from `python3 -m sglang.launch_server` to `sglang serve` |

## 2. Fix Details

### 2.1 WEB-AI-RC-001: llama.cpp GGUF -m Points to Directory

**Root cause**: The Phase 2.5 fix only changed the scan proxy (preventing new bad data). But:
1. Existing DB records still had `model_locations.relative_path = "Qwen3.5-9B-Q4"` (directory name)
2. The resolver's `modelRelativePath()` returned `Artifact.RelativePath` verbatim
3. The `model_container_path` template variable used for `-m` always equaled the directory path
4. No GGUF-specific logic existed in the resolver to compute a file-level path

**Fix**:
1. Added `MODEL_CONTAINER_FILE` variable in `buildVarMap` (resolver.go): when `Artifact.Path` ends with `.gguf` but `modelBase` doesn't, appends the artifact's GGUF filename to produce a file-level container path
2. Updated llama.cpp `default_args_json` in seed data (both legacy b4817 and new b9700 catalog): changed `-m {{model_container_path}}` to `-m {{model_container_file}}`
3. Added `TestLlamaCppGGUFFileInDirectory` test covering the production scenario: directory `relative_path` + GGUF artifact path

**Before (wrong)**:
```bash
-v /home/kzeng/models/Qwen3.5-9B-Q4:/models/Qwen3.5-9B-Q4:ro
-m /models/Qwen3.5-9B-Q4                          # directory, llama.cpp fails
```

**After (correct)**:
```bash
-v /home/kzeng/models/Qwen3.5-9B-Q4:/models/Qwen3.5-9B-Q4:ro
-m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf  # specific GGUF file
```

**Files**: `internal/server/runplan/resolver.go`, `internal/server/db/db.go`, `internal/server/runplan/llamacpp_nvidia_test.go`

### 2.2 WEB-AI-RC-002: LLAMA_ARG_HOST Warning

**Root cause**: The llama.cpp Docker image (`ghcr.io/ggml-org/llama.cpp:server-cuda13`) has a built-in `LLAMA_ARG_HOST` environment variable. The platform adds `--host 0.0.0.0` which correctly overrides it. The warning is cosmetic — the image emits it, not the platform.

No code in LightAI sets `LLAMA_ARG_HOST`. No code change needed.

**Status**: DOCUMENTED — benign warning from Docker image, not platform-injected.

### 2.3 WEB-AI-RC-003: Test Prompt Simplification

**Root cause**: Default test prompt was `"ping"` which models interpret as a question about the network tool "Ping", producing verbose explanations.

**Fix**:
1. Changed default prompt from `"ping"` to `"Reply with exactly one word: pong"`
2. Chat mode: added system message `"Reply with exactly one word: pong"` + user message with the prompt
3. Updated both backend (handler + inference functions) and frontend (ModelInstancesPage.vue)
4. Kept `max_tokens: 8`, `temperature: 0`

**Before (wrong response)**:
```
Ping 是一种网络测试工具，用于...
```

**After (expected response)**:
```
pong
```

**Files**: `internal/server/api/deployment_lifecycle_handlers.go`, `web/src/pages/ModelInstancesPage.vue`

### 2.4 WEB-AI-RC-004: SGLang Recommended Entrypoint

**Root cause**: SGLang seed data used `["python3", "-m", "sglang.launch_server"]` which is deprecated in favor of `sglang serve`.

**Fix**: Changed all 3 SGLang BackendVersion entries in `seedTargetBackendCatalog` from `["python3","-m","sglang.launch_server"]` to `["sglang","serve"]`. Updated `TestResolveSGLangNVIDIA` accordingly.

SGLang was already confirmed running. This is a low-risk change because:
- SGLang v0.5.12+ images include the `sglang` CLI
- `sglang serve` is the officially documented recommended entrypoint
- Existing deployments use frozen config snapshots and won't be affected

**Before**:
```bash
docker run ... lmsysorg/sglang:latest --entrypoint python3 -m sglang.launch_server --model-path ...
```

**After**:
```bash
docker run ... lmsysorg/sglang:latest --entrypoint sglang serve --model-path ...
```

**Files**: `internal/server/db/db.go`, `internal/server/runplan/vllm_sglang_nvidia_test.go`

## 3. Schema / Migration

- No schema changes in this round.
- No new migrations.
- Seed data updated (llama.cpp `default_args_json`, SGLang `default_entrypoint_json`).
- Existing deployments use frozen config snapshots and won't be affected.

## 4. Items NOT Done (per scope)

- No resource parameter editor (Phase 3)
- No multi-replica/cross-node scheduling
- No Playwright specs
- No API Gateway/API Key
- No Phase 3 scope creep

## 5. Test Results

```bash
gofmt -w cmd/ internal/                     → CLEAN
go test lightai-go/internal/server/api/...    → ALL PASS (6.670s)
go test lightai-go/internal/server/runplan/... → ALL PASS
  TestLlamaCppNvidiaRunPlan                   → PASS (-m points to .gguf file)
  TestLlamaCppGGUFFileInDirectory             → PASS (directory mount, file -m)
  TestLlamaCppRunPlanNoGPU                    → PASS
  TestResolveSGLangNVIDIA                     → PASS (entrypoint: sglang serve)
go vet ./...                                   → CLEAN
npm test                                       → ALL PASS
npm run build                                  → ✓ built
git diff --check                                → CLEAN
```

## 6. Modified Files

| File | Change |
|------|--------|
| `internal/server/runplan/resolver.go` | Add MODEL_CONTAINER_FILE variable for GGUF file path |
| `internal/server/db/db.go` | Update llama.cpp -m to `{{model_container_file}}`; SGLang entrypoint to `sglang serve` |
| `internal/server/runplan/llamacpp_nvidia_test.go` | Update to `model_container_file`; add GGUF-in-directory test |
| `internal/server/runplan/vllm_sglang_nvidia_test.go` | Update SGLang entrypoint to `sglang serve` |
| `internal/server/api/deployment_lifecycle_handlers.go` | Change test prompt to "pong"; add system message for chat |
| `web/src/pages/ModelInstancesPage.vue` | Change test prompt to "Reply with exactly one word: pong" |
| `docs/reports/phase-3/web-ai-config-review/20-phase-2-runtime-command-cleanup-issues.md` | Issue tracking |
| `docs/reports/phase-3/web-ai-config-review/21-phase-2-runtime-command-cleanup-closeout.md` | This closeout |

## 7. Final Status

PASS — all 4 issues resolved, all tests pass, git status clean.
