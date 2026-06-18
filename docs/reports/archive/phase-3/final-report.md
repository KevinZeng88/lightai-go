> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# Phase 3 Final Review Report

> Last updated: 2026-06-17

## 1. Latest Verification Result: PARTIAL

| Test | Latest Result | Notes |
|------|--------------|-------|
| api-only (3 backends) | PASS | Instance/runplan/task creation verified |
| quick | PASS | Backends listed, no Docker |
| llamacpp-only (long) | Historical PASS (422s on first successful run) | Container running verified via DB |
| llamacpp-only (bounded 120s) | FAIL — container exited(1) | Agent Docker command unreliable |
| vllm-only | Not run | Pending agent fix |
| sglang-only | Not run | Pending agent fix |
| full | Not run | Pending agent fix |
| smoke-model-backends.sh all | PASS | Direct Docker, 3/3 backends |

**Agent Docker lifecycle is NOT fully verified.** Direct Docker smoke works. Agent-driven Docker start has intermittent exit code 1.

## 2. Why Long E2E Is Paused

1. `cmd_single_backend` hardcodes `TIMEOUT=600`, overriding `E2E_TIMEOUT` env var
2. Agent `processStartTask` Docker commands cause containers to exit(1) intermittently
3. Running full (3 backends × model loading) takes ~30 min, wastes time without fixing root cause
4. Server/agent logs at `log_level: error` don't capture INFO-level lifecycle diagnostics

## 3. Completed

| Item | Status | Evidence |
|------|--------|----------|
| Old chain deletion | Completed | 20 files, -6857 lines |
| 10 new tables V10 | Completed | Clean DB migration verified |
| Backend/Version seed | Completed | 3 backends, 5 versions |
| RBAC (20 permissions) | Completed | viewer/operator/admin mapped |
| RunPlan Resolver | Completed | 20 tests, triple-backend, 0 failures |
| RunPlan→AgentRunSpec adapter | Completed | 2 tests |
| API handlers (CRUD+lifecycle) | Completed | Backend/Runtime/Artifact/Deployment/Instance |
| Failure-state handling | Completed | Idempotent stop, delete with cleanup |
| Web 5 pages | Completed | Real APIs, zh-CN/en-US i18n |
| Docker direct smoke | Completed | 3/3 backends on RTX 5090 |
| Vendor profiles | Completed | nvidia.yaml, metax.yaml |
| E2E scripts | Completed | Safe PID mgmt, credential isolation |
| Logging coverage audit | Completed | docs/reports/phase-3/logging-coverage-audit.md |
| Server lifecycle logging | Completed | db.go, runplan/resolver.go, deployment_lifecycle_handlers.go |
| Agent lifecycle logging | Completed | processStartTask begin/completed/error |

## 4. Diagnostic Logging Added

| File | Logs Added |
|------|-----------|
| `internal/server/db/db.go` | migrate begin/completed + `duration_ms` |
| `internal/server/runplan/resolver.go` | resolve begin/completed + `backend`/`vendor`/`errors`/`duration_ms` |
| `internal/server/api/deployment_lifecycle_handlers.go` | start/stop/delete begin/completed + `duration_ms`/`instance_id`/`task_id` |
| `internal/server/api/agent_handlers.go` | task result success/failure + `instance_id`/`container_id` |
| `cmd/agent/main.go` | processStartTask begin/completed/error + `task_id`/`duration_ms` |
| `scripts/e2e-model-runtime-api.sh` | `[TIMING]` markers at instance_created/running |

## 5. Container exited(1) Investigation

**Status**: Under investigation. Direct Docker smoke works. Agent-generated Docker commands differ — need spec comparison.

### Direct smoke (working):
```
docker run -d --name qwen35-9b-q4-llama --gpus all -p 8002:8080
  -v /home/kzeng/models/Qwen3.5-9B-Q4:/models:ro
  ghcr.io/ggml-org/llama.cpp:server-cuda13
  -m /models/Qwen3.5-9B-Q4_K_M.gguf --host 0.0.0.0 --port 8080 --ctx-size 4096 --n-gpu-layers 999
```

### Agent-generated (from RunPlan response, exited 1):
```
docker run -d --name lightai-<instance-id>
  -v /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf:/models/Qwen3.5-9B-Q4_K_M.gguf:ro
  ghcr.io/ggml-org/llama.cpp:server-cuda13
  llama-server -m /models/Qwen3.5-9B-Q4_K_M.gguf --host 0.0.0.0 --port 8080 -ngl 1 ...
```

**Suspected issues**: GPU flag missing (`--gpus all` vs CUDA_VISIBLE_DEVICES), mount path mismatch (file vs directory), args differences.

## 6. Direct Smoke vs Agent Spec Diff (to be completed)

Pending static comparison via diagnostic script. Key items to compare: image, entrypoint, command, args, model path, volume mount, port mapping, GPU visibility, health check.

## 7. Remaining Risks

1. Agent Docker command generation may not match direct smoke working commands
2. `log_level: error` in E2E config suppresses INFO lifecycle logs
3. No cross-component correlation_id — manual tracing via instance_id/task_id
4. Container exit code 1 not captured with docker logs in E2E

## 8. Next Steps

1. Run static diagnostic spec dump (no container start)
2. Compare direct smoke vs agent spec
3. Fix any args/mount/GPU differences found
4. One short bounded test (120s, no model load wait)
5. If container still exits(1): capture docker logs + inspect
6. Fix identified root cause
7. Then re-run llamacpp-only as final verification

## 9. Build & Test

```
go test ./... → 9 OK | go build server/agent → OK | npm build → 3.07s
bash -n scripts/e2e-model-runtime-api.sh → OK
bash -n scripts/smoke-model-backends.sh → OK
git diff --check → OK
```
