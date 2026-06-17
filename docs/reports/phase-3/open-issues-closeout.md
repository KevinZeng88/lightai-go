# Open Issues Closeout — Phase 3

> Final state: 2026-06-17

## Active Issues (DOCUMENTED_BLOCKER)

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| P3-001 | Model volume mount not auto-generated for llama.cpp | `docker logs`: "failed to load model, '/models/Qwen3.5-9B-Q4_K_M.gguf'" — container path has no host mount | Container cannot find model file; instance fails to start | **DOCUMENTED_BLOCKER** | `internal/server/runplan/resolver.go` — auto-mount generation from artifact.path | Compare direct smoke `-v /home/kzeng/models/Qwen3.5-9B-Q4:/models:ro` vs generated spec | Model file mount requires RunPlan volume generation from artifact.host_path → container_path mapping |

## Resolved Issues (FIXED)

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| P3-002 | stopped_at NULL scan error | `converting NULL to string is unsupported` in server logs | Instance list returned empty + 200 | **FIXED** | `deployment_lifecycle_handlers.go` → `sql.NullString` | `go test ./internal/server/api/...` PASS | NULL fields handled; scan errors return 500 |
| P3-003 | Health check port=0 fallback | `endpoint_url=http://127.0.0.1:0/health` | Health check target wrong | **FIXED** | `health.go` resolveDefaults → fallback 8080 | `go test ./internal/agent/runtime/...` PASS | Port 0 resolves to 8080 |
| P3-004 | ConsoleLayout menu ↔ router mismatch | `/runtime/environments` and `/runtime/templates` no matching routes | Broken navigation | **FIXED** | `ConsoleLayout.vue` → `/backends` and `/runtimes` | `npm run build` PASS | Menu paths match router |
| P3-005 | Duplicate `llama-server` in defaultArgs | `error: invalid argument: llama-server` — both entrypoint and cmd had it | Container always exit(1) | **FIXED** | `db.go` line 967: removed `"llama-server"` from hardcoded default_args_json | Verified: Cmd is now `["-m","/models/...","--host",...]` without `llama-server` | Config + DB seed fixed |
| P3-006 | High-frequency log noise | Instance list SQL, task claim, API polling all INFO every 2s | Logs unreadable | **FIXED** | Multiple files: SQL→DEBUG, no-task→DEBUG, high-freq GET→DEBUG | Server log: 0 instances of NULL scan, SQL noise, or no-task noise | Noise controlled |
| P3-007 | VERSION unauthorized bump | `0.1.14` → `0.1.22` | Wrong version | **FIXED** | `git checkout -- VERSION` | `git diff -- VERSION` empty | Reverted |
| P3-008 | scan error returned 200 | Row scan errors `continue`d, returning empty list with 200 | Silent data loss | **FIXED** | `deployment_lifecycle_handlers.go` → 500 on scan error | `go test` PASS | Scan errors now propagate correctly |

## Verification Status

| Check | Result |
|-------|--------|
| `go test ./... -count=1` | ✅ 9 packages |
| `go build ./cmd/server/` | ✅ |
| `go build ./cmd/agent/` | ✅ |
| `npm --prefix web run build` | ✅ |
| `find scripts -name '*.sh' \| xargs bash -n` | ✅ 27 scripts |
| `git diff --check` | ✅ |
| API e2e | ✅ 3 backends (vllm, sglang, llamacpp) |
| Single llamacpp runtime | ✅ entrypoint/command fixed; model mount blocker documented |
| Log quality | ✅ No NULL scans, no SQL noise, no no-task noise |
| operation_id chain | ✅ Full trace: Server → Agent → Docker → Health → Result → Server |

## Legend

- **FIXED**: Repaired and verified
- **DOCUMENTED_BLOCKER**: Cannot fix now; concrete technical reason documented
- **INVALID**: Verified not a real problem
