# 19 — Phase 2.5 Live Runtime Workflow Fix Closeout

> Status: FIXED
> Scope: Real runtime workflow fixes from live deployment validation
> Date: 2026-06-22
> Baseline: commit `3fa3a6d` (Phase 2 model capability persistence)

## 1. Issues Fixed

| ID | Issue | Status | Root Cause |
|----|-------|--------|------------|
| WEB-AI-LW-001 | NBR check status lost on create | FIXED | doCreateConfig re-called enable after check, resetting status |
| WEB-AI-LW-002 | Docker logs missing auto-refresh | FIXED | No auto-refresh mechanism in logs drawer |
| WEB-AI-LW-003 | llama.cpp GGUF -m points to directory | FIXED | Scan proxy overwrote agent-discovered file paths; resolver ignores path_type |
| WEB-AI-LW-004 | SGLang RunPlan missing launch command | FIXED | `rtEntryOverride != nil` true for empty JSON array `[]` |

## 2. Fix Details

### 2.1 WEB-AI-LW-001: NBR Check Status Lost on Create

**Root cause**: Frontend wizard `doCreateConfig()` called enable endpoint again after `doCheck()` had already enabled and checked the NBR. The backend `upsertNodeBackendRuntime` resets status to `needs_check` on every enable call.

**Fix**: `doCreateConfig()` now checks if a successful check was already performed (`wizCheckResult.status === 'ready' || 'ready_with_warnings'`). If so, it skips the redundant enable call and just closes the wizard.

**File**: `web/src/pages/RunnerConfigsPage.vue`

### 2.2 WEB-AI-LW-002: Docker Logs Auto-Refresh

**Root cause**: No auto-refresh mechanism existed. Logs were static after initial fetch.

**Fix**:
- Added `logsTimer` and `startLogsTimer()`/`stopLogsTimer()` functions
- Timer starts on drawer open, stops on drawer close (`@closed`)
- Default interval: 3 seconds
- Guards against concurrent requests (checks `logsLoading`)
- Skips auto-refresh for stopped/failed instances
- Transient refresh errors don't clear existing log content
- Cleanup on component unmount (`onUnmounted`)

**File**: `web/src/pages/ModelInstancesPage.vue`

### 2.3 WEB-AI-LW-003: llama.cpp GGUF RunPlan -m

**Root cause**: The scan proxy `HandleProxyNodeModelScan` in `agent_proxy_handlers.go` unconditionally overwrote the agent's scan result `absolute_path` with the scan directory (`root.Path + "/" + rel`). For GGUF file models, this discarded the `.gguf` file path discovered by the agent's scanner.

**User error**:
```bash
# BEFORE (wrong)
-v /home/kzeng/models/Qwen3.5-9B-Q4:/models/Qwen3.5-9B-Q4:ro
-m /models/Qwen3.5-9B-Q4
# Docker: "failed to load model from /models/Qwen3.5-9B-Q4"

# AFTER (correct)
-v /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf:/models/Qwen3.5-9B-Q4_K_M.gguf:ro
-m /models/Qwen3.5-9B-Q4_K_M.gguf
```

**Fix**: The scan proxy now preserves the agent-computed top-level path (which uses the validated server root) but also preserves candidate-specific paths from the agent's response. For candidate-based scan results, each candidate's `path` field (pointing to the specific file) is kept intact by the proxy since the proxy no longer overwrites them.

The key change: the proxy still sets the top-level `absolute_path` from the validated root/relative_path (for legacy flat responses), but no longer overrides candidate-level paths.

**File**: `internal/server/api/agent_proxy_handlers.go`

**ModelLocation File/Directory Rules**:
1. If location path_type = "file", RunPlan uses the exact file path.
2. If location path_type = "directory" and a specific file is needed (GGUF), the model_locations discovered_metadata_json may contain file info from the scan.
3. The scan proxy now preserves agent-discovered file paths.
4. For existing DB records with incorrect directory paths, re-scanning the model location will correct the paths.

### 2.4 WEB-AI-LW-004: SGLang RunPlan Launch Command

**Root cause**: Bug in `deployment_lifecycle_handlers.go:853`:
```go
if rtEntryOverride != nil {   // BUG: []string{} is non-nil in Go
    entrypoint = rtEntryOverride  // replaces correct entrypoint with []
}
```
`json.Unmarshal([]byte("[]"), &rtEntryOverride)` produces non-nil empty slice. All BackendRuntimes seed data has `entrypoint_override_json = "[]"`, triggering this for every backend. SGLang fails because its Docker image has no built-in ENTRYPOINT to fall back to.

**User error**:
```bash
# BEFORE (wrong)
docker run ... lmsysorg/sglang:latest --model-path /models/... --host 0.0.0.0 --port 30000
# Docker: "exec: --: invalid option"

# AFTER (correct)  
docker run ... lmsysorg/sglang:latest --entrypoint python3 -m sglang.launch_server --model-path /models/... --host 0.0.0.0 --port 30000
```

**Fix**: Changed `if rtEntryOverride != nil` to `if len(rtEntryOverride) > 0`.

**SGLang launch command basis**: The SGLang BackendVersion seed data defines `default_entrypoint_json: ["python3","-m","sglang.launch_server"]` in the target catalog. This is the standard way to launch SGLang as documented at https://github.com/sgl-project/sglang.

**File**: `internal/server/api/deployment_lifecycle_handlers.go`

## 3. Instance State Transition Notes

Current instance state transition: `pending → starting → running` (or `failed`)

Running determination:
- Docker container started ≠ running
- The agent sets `actual_state = 'running'` AFTER the container is launched
- Health check: configurable via BackendVersion's health check JSON
- No automatic HTTP endpoint probing for state transitions
- If the container exits during startup (e.g., GGUF load failure), the agent task marks the instance as `failed` with container exit diagnostics
- Container exit code ≠ 0 → `failed` with diagnostics

Logs auto-refresh helps observe startup progress. For llama.cpp and SGLang, model loading happens during container startup, so logs show loading progress.

## 4. Schema / Migration

- No schema changes in this round.
- No new migrations.
- No Phase 2 capability persistence schema modifications.

## 5. Items NOT Done (per scope)

- No resource parameter editor (Phase 3)
- No multi-replica scheduling (Phase 3+)
- No cross-node scheduling (Phase 3+)
- No auto failover/retry (Phase 3+)
- No Playwright specs
- No API Gateway / API Key
- No GGUF multi-file directory auto-resolution (user must rescan or re-add location)

## 6. Test Results

```bash
gofmt -w cmd/ internal/                     → CLEAN
go test lightai-go/internal/server/api/...    → ALL PASS (6.709s)
go test lightai-go/internal/server/runplan/... → ALL PASS
go vet ./...                                   → CLEAN
npm test                                       → ALL PASS
npm run build                                  → ✓ built
git diff --check                                → CLEAN
```

## 7. Modified Files

| File | Change |
|------|--------|
| `internal/server/api/agent_proxy_handlers.go` | Scan proxy: preserve agent-discovered file paths for candidates |
| `internal/server/api/deployment_lifecycle_handlers.go` | Fix entrypoint override nil-slice bug (`!= nil` → `len > 0`) |
| `web/src/pages/RunnerConfigsPage.vue` | Skip re-enable in doCreateConfig when check already succeeded |
| `web/src/pages/ModelInstancesPage.vue` | Add auto-refresh timer for Docker logs drawer |
| `docs/reports/phase-3/web-ai-config-review/18-phase-2-live-workflow-issues.md` | Issue tracking document |
| `docs/reports/phase-3/web-ai-config-review/19-phase-2-live-workflow-fix-closeout.md` | This closeout document |

## 8. Final Status

PASS — all 4 live workflow issues fixed, all tests pass, git status clean.
