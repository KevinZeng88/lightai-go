# 18 — Phase 2.5 Live Runtime Workflow Issues

> Status: DRAFT
> Scope: Real runtime workflow fixes from live deployment validation
> Date: 2026-06-22
> Baseline: commit `3fa3a6d` (Phase 2 model capability persistence)

## Issue Index

| ID | Issue | Severity | Status |
|----|-------|----------|--------|
| WEB-AI-LW-001 | NBR check status lost on create | P0 | DRAFT |
| WEB-AI-LW-002 | Docker logs missing auto-refresh | P1 | DRAFT |
| WEB-AI-LW-003 | llama.cpp GGUF RunPlan -m points to directory | P0 | DRAFT |
| WEB-AI-LW-004 | SGLang RunPlan missing launch command | P0 | DRAFT |

---

## WEB-AI-LW-001: NBR Check Status Lost on Create

### User Phenomenon

1. Add node runtime config → step through wizard
2. Last step: click "检测" → shows "就绪/已通过检测"
3. Click "创建" → NBR appears in list as "需重新检测"
4. Must click "检测" again to make it "就绪"

### Root Cause

In the frontend wizard (`RunnerConfigsPage.vue`), `doCheck()` calls enable + check-request, but then `doCreateConfig()` calls enable again. The backend `upsertNodeBackendRuntime` function always resets non-blocking status to `needs_check` when `checkOnly=false`.

So the wizard creates the NBR (enable) → checks it (check-request → sets `ready`) → creates it again (enable → resets to `needs_check`).

### Affected Code

- `web/src/pages/RunnerConfigsPage.vue`: `doCheck()` (line 494) and `doCreateConfig()` (line 511)
- `internal/server/api/runtime_handlers.go`: `upsertNodeBackendRuntime` (line 700), status reset at line 740-746

### Fix

`doCreateConfig()` should not re-enable an already-enabled NBR. It should only enable if the NBR doesn't exist yet (first-time flow without prior check). If check was already done and status is `ready`/`ready_with_warnings`, skip the second enable call.

### Acceptance

- Check → status shows `ready` → click "创建" → NBR list shows `ready` (not `needs_check`)
- Create without check → shows `needs_check`
- Modify checked config → shows `needs_check`

---

## WEB-AI-LW-002: Docker Logs Missing Auto-Refresh

### User Phenomenon

When viewing Docker logs for a starting/running instance, the log output is static — user must manually click refresh to see new output. This is inconvenient during startup when logs are scrolling.

### Root Cause

No auto-refresh mechanism exists for the Docker logs drawer. The logs are fetched once when the drawer opens. The `useAutoRefresh` composable exists but is not used in `ModelInstancesPage.vue`.

### Affected Code

- `web/src/pages/ModelInstancesPage.vue`: logs drawer (line 140), `loadLogs()` (line 281), `openLogs()` (line 275)
- `web/src/composables/useAutoRefresh.ts`: available but unused here

### Fix

Add auto-refresh timer in logs drawer:
- Start timer on drawer open, stop on drawer close
- Default interval: 3 seconds
- Guard against concurrent requests
- Stop auto-refresh for stopped/failed instances
- Keep manual refresh button

### Acceptance

- Open logs → auto-refreshes every ~3s
- Close logs → stops refreshing
- No multiple concurrent refresh requests
- Stopped/failed instances stop auto-refresh

---

## WEB-AI-LW-003: llama.cpp GGUF RunPlan -m Points to Directory

### User Phenomenon

Model location shows:
- Type: file
- Path: `/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf`

But RunPlan generates:
```bash
-v /home/kzeng/models/Qwen3.5-9B-Q4:/models/Qwen3.5-9B-Q4:ro
-m /models/Qwen3.5-9B-Q4
```

Expected:
```bash
-v /home/kzeng/models/Qwen3.5-9B-Q4:/models/Qwen3.5-9B-Q4:ro
-m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

### Docker Error
```
gguf_init_from_reader: failed to read magic
failed to load model from /models/Qwen3.5-9B-Q4
llama_server: exiting due to model loading error
```

### Root Cause

The scan proxy handler (`agent_proxy_handlers.go:120-127`) overwrites the agent's discovered candidate file path with the scan request's directory path. When the user scans `/home/kzeng/models/Qwen3.5-9B-Q4/`, the agent correctly finds `Qwen3.5-9B-Q4_K_M.gguf` with path `/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf`, but the proxy replaces this with the directory path `/home/kzeng/models/Qwen3.5-9B-Q4`.

As a result, the model location stores `relative_path = "Qwen3.5-9B-Q4"` (directory) instead of `relative_path = "Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf"`. The resolver then mounts the directory and passes `-m /models/Qwen3.5-9B-Q4`.

Additionally, the resolver has no logic to handle the case where the artifact path is a directory but the backend requires a specific file (GGUF). For `path_type = "directory"` + GGUF format, the resolver should:
1. Respect the discovered file path stored in `model_locations`
2. OR look for `.gguf` files in the directory
3. OR fail preflight with a clear error

### Affected Code

- `internal/server/api/agent_proxy_handlers.go`: scan proxy overrides candidate paths (line 120-127)
- `internal/server/runplan/resolver.go`: no GGUF file discovery logic
- `internal/server/api/deployment_lifecycle_handlers.go`: resolver input assembly (line 797) doesn't read `path_type`

### Fix

1. Fix scan proxy to preserve agent's candidate file paths (don't override with scan directory)
2. When the frontend wizard creates model from scan, use the candidate's specific path
3. For the resolver: when format is `gguf` and the relative_path points to a directory, check model_locations for discovered file info
4. For existing GGUF directory models without a specific file path: handle gracefully

### Acceptance

- llama.cpp + GGUF file location → `-m` points to container file path
- llama.cpp + GGUF directory with discovered files → finds and uses the .gguf file
- Scan creates model location with correct file path, not directory path

---

## WEB-AI-LW-004: SGLang RunPlan Missing Launch Command

### User Phenomenon

SGLang RunPlan generates:
```bash
docker run ... lmsysorg/sglang:latest --model-path /models/... --host 0.0.0.0 --port 30000
```

Docker error:
```
/opt/nvidia/nvidia_entrypoint.sh: line 67: exec: --: invalid option
exec: usage: exec [-cl] [-a name] [command [argument ...]] [redirection ...]
```

### Root Cause

Bug in `deployment_lifecycle_handlers.go:853`:
```go
if rtEntryOverride != nil {  // BUG: true for empty JSON array "[]"
    entrypoint = rtEntryOverride  // replaces correct entrypoint with []
}
```

`json.Unmarshal([]byte("[]"), &rtEntryOverride)` produces non-nil empty slice `[]string{}` in Go. The condition `rtEntryOverride != nil` is true even for empty arrays. This causes the BackendVersion's correct entrypoint (`["python3", "-m", "sglang.launch_server"]`) to be replaced with an empty slice.

All BackendRuntimes have `entrypoint_override_json = "[]"` in seed data, so all backends are affected. But vLLM works because its Docker image has a built-in ENTRYPOINT that Docker falls back to. SGLang's image has no (or wrapper) ENTRYPOINT, so Docker tries to exec `--model-path` which fails.

### Expected RunPlan
```bash
docker run ... lmsysorg/sglang:latest \
  python3 -m sglang.launch_server \
  --model-path /models/Qwen3-0.6B-Instruct-2512 \
  --host 0.0.0.0 \
  --port 30000
```

### Affected Code

- `internal/server/api/deployment_lifecycle_handlers.go`: line 853 `rtEntryOverride != nil` → should be `len(rtEntryOverride) > 0`

### Fix

Change `if rtEntryOverride != nil` to `if len(rtEntryOverride) > 0` at line 853.

### Acceptance

- SGLang RunPlan first container arg is not `--model-path`
- RunPlan contains valid launch command
- No `exec: --: invalid option` error
- vLLM RunPlan continues to work (no regression)
