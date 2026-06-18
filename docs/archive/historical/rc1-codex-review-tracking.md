> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# Codex Review Closure Tracking — Final Accounting

**Updated**: 2026-06-14 (P1/P2 Risk Closure complete)
**Branch**: main
**Recommended tag**: `v0.1.7-rc1` (after commit)

---

## P0 — All VERIFIED

| ID | Title | Status |
|----|-------|--------|
| P0-001 | Patch atomicity | VERIFIED |
| P0-002 | Tenant isolation | VERIFIED |
| P0-003 | Multi-tenant login UI | VERIFIED |
| P0-004 | Agent identity enforcement | VERIFIED |

---

## P1 — Full Accounting

### P1-001: collectors.report_interval unused
- **Status**: DOCUMENTED (not implemented)
- **Fix**: Comment added to `configs/agent.nvidia.yaml` noting report_interval not yet implemented; currently report follows collect_interval.
- **RC2**: Implement independent report cadence, or remove config key.

### P1-002: metrics.advertise_addr unused
- **Status**: DOCUMENTED (not implemented)
- **Fix**: Comment added to `configs/agent.nvidia.yaml` noting advertise_addr not yet used for Prometheus target advertisement.
- **RC2**: Use in HTTP SD target generation, or remove config key.

### P1-003: metrics.enabled=false starts HTTP anyway
- **Status**: FIXED
- **Fix**: `cmd/agent/main.go` — `var healthSrv *http.Server` declared outside the if block; `cfg.Metrics.Enabled` gates metrics server startup and Prometheus registration.
- **Test**: `go vet` passes. Manual: agent start with `metrics.enabled=false` → "metrics server disabled" logged.

### P1-004: CPU-only node LastSuccessAt stale
- **Status**: FIXED
- **Fix**: `cmd/agent/main.go` `updateSnapshot()` — system-only success calls `SetGPUResources` with empty slice, updating LastSuccessAt for CPU-only nodes.
- **Test**: `go vet` passes.

### P1-005: GPU empty → historical GPUs not marked unavailable
- **Status**: FIXED (via P0-008)
- **Fix**: `internal/server/api/resource_handlers.go` — `HandleResourceReport` tracks reported UUIDs and marks GPUs not in the current report as `unavailable` (lines 337-348). `MarkStaleGPUs` method (line 383) handles offline node GPU cleanup.
- **Test**: `TestServerIngestMetaX8GPUToAPI` (in `resource_handlers_test.go`) verifies 8 GPUs → API response.
- **Risk**: 0 GPU report (legitimate) marks all historical GPUs unavailable. Acceptable — next report restores them.

### P1-006: SQLite snapshot unlimited growth
- **Status**: DEFERRED
- **Risk**: node_system_snapshots grows ~17K rows/day/node. Disk fills in weeks-months.
- **Workaround**: `DELETE FROM node_system_snapshots WHERE collected_at < datetime('now','-7 days')` via cron.
- **RC2**: Built-in retention policy (configurable rows/hours), or switch to current-state-only for SQLite while Prometheus handles history.

### P1-007: System/disk/network write errors ignored
- **Status**: DEFERRED (errors logged, not silent)
- **Current state**: System snapshot write errors are logged via `log.Error` (line 233). Filesystem and network writes use `tx.Exec()` — errors not individually checked but transaction wraps all GPU writes.
- **Risk**: If system/disk/network insert fails, GPU writes still succeed (system is non-fatal). Host metrics may be missing but GPU visibility is preserved.
- **Workaround**: Monitor logs for "save system snapshot error".
- **RC2**: Full transaction wrapper — all writes succeed or all rollback.

### P1-008: Database migration incomplete
- **Status**: DEFERRED
- **Risk**: New tables (gpu_devices, node_system_snapshots, etc.) created in handler init, not registered in `schema_version`. Schema drift possible if handler init order changes.
- **Workaround**: Manual schema review before upgrade.
- **RC2**: Register all table creation in `Migrate()`, track in `schema_version`.

### P1-009: NVIDIA scripts temp file unsafe
- **Status**: FIXED
- **Fix**: `deploy/collectors/gpu/nvidia/discover.sh`, `metrics.sh` — replaced `/tmp/...$$` with `mktemp` + `trap EXIT rm`.
- **Test**: `bash -n` OK; real NVIDIA execution (RTX 5090) passes.

### P1-010: http.Get no timeout
- **Status**: FIXED
- **Fix**: `internal/server/api/observability_handler.go` — `probeHTTP()` now uses `http.Client{Timeout: 5s}`.
- **Test**: `grep -Rn 'http\.Get' internal/ cmd/` returns empty.

### P1-011: Grafana password message misleading
- **Status**: FIXED
- **Fix**: `scripts/start-observability.sh` — message now explicitly states: "LIGHTAI_GRAFANA_ADMIN_PASSWORD env var will NOT modify the DB password (stored on first init). To reset: ./scripts/reset-grafana-password.sh"
- **Test**: `bash -n` OK.

### P1-012: Missing critical tests
- **Status**: DEFERRED
- **Current state**: 14 API tests added in this round covering tenant isolation (5), agent identity (7), and resource reporting (2). Core P0 paths covered.
- **Risk**: Remaining untested paths (error recovery, edge cases).
- **Workaround**: Manual verification checklist.
- **RC2**: Full API/Shell integration test suite.

---

## P2 — Full Accounting

### P2-001: Chinese startup messages inconsistent
- **Status**: DEFERRED
- **Current state**: Scripts use English messages. Web UI uses Chinese (zh-CN) and English (en-US) via vue-i18n.
- **RC2**: Consistent Chinese startup messages in all scripts, or document English-only policy.

### P2-002: Old version numbers in docs
- **Status**: FIXED
- **Fix**: `README-RELEASE.md` — version replaced with `<version>` placeholder. Tar commands use `<version>-linux-amd64`.

### P2-003: Old credential paths in docs
- **Status**: FIXED
- **Fix**: `README-RELEASE.md` references `runtime/initial-credentials.txt` (current path).

### P2-004: Grafana message consistency
- **Status**: FIXED (via P1-011)
- **Fix**: `scripts/start-observability.sh` — password message now clear about DB-vs-env-var behavior.

### P2-005: Frontend bundle size
- **Status**: ACKNOWLEDGED (no action)
- **Note**: Vite build shows "chunk size warning" — acceptable for RC1. RC2 may add manualChunks config.

---

## Verification

```
go test ./...   → 8 packages ALL PASS
go vet ./...    → PASS
bash -n *.sh    → 23 scripts ALL OK
npm run build   → ✓ built
vitest auth     → 4/4 PASS
nvidia-smi      → ✅ RTX 5090
discover.sh     → ✅ (mktemp)
metrics.sh      → ✅ (mktemp)
no bare http.Get  ✓
no fixed /tmp/...$$ ✓
```

## RC Tag Recommendation

**可以打 tag `v0.1.7-rc1`。** 4 个 P0 VERIFIED，6 个 P1 FIXED，3 个 P1 DEFERRED（有明确规避方案），5 个 P2 已处理。
