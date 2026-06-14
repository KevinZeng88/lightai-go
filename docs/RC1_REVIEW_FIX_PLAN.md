# LightAI Go RC1 — 主台账

**Last Update**: 2026-06-14 (P1/P2 Risk Closure complete)
**Current Version**: 0.1.7 (VERSION file; builds produce 0.1.6 with --no-bump, 0.1.7 with next)
**Recommended Tag**: `v0.1.7-rc1`
**Branch**: main

---

## 当前 RC1 状态

| Item | Status |
|------|--------|
| P0 (Codex Review) | 4/4 **VERIFIED** |
| P1 critical fixes | 6/12 **FIXED** |
| P1 deferred | 4/12 **DEFERRED** with risk/mitigation/RC2 plan |
| P1 documented (not implemented) | 2/12 **DOCUMENTED** |
| P2 | 3/5 FIXED, 1 DEFERRED, 1 ACKNOWLEDGED |
| RC tag | **可以打 `v0.1.7-rc1`** |
| MetaX 8-card field validation | **待现场执行** |
| glibc 2.28 compatibility | **VERIFIED** (12 ELF, 0 violations) |

**交付口径**: P0 全部 VERIFIED。关键 P1 已修复，剩余 P1 已记录风险、规避方式和 RC2 计划。可以进入 v0.1.7-rc1 tag。该版本仍需在 MetaX 现场验证 Web 是否显示 8 张 MetaX C500。

---

## P0 Status — Codex Review

| ID | Title | Status | Root Cause | Tests |
|----|-------|--------|------------|-------|
| P0-001 | Patch atomicity | **VERIFIED** | No `set -e`; cp failures ignored; VERSION written unconditionally | `tests/test_patch_atomicity.sh` (7 scenarios including cp failure) |
| P0-002 | Tenant isolation | **VERIFIED** | Hardcoded `WHERE tenant_id='default'` | 6 Go tests in `tenant_isolation_test.go` |
| P0-003 | Multi-tenant login UI | **VERIFIED** | `selectedTenantId` never sent to backend | 4 Vitest tests in `auth.test.ts` |
| P0-004 | Agent identity enforcement | **VERIFIED** | Default token only warned; no agent_id binding | 7 Go tests in `agent_identity_test.go` |

### P0-001 Detail

- **Fix**: `set -e`, VERSION skipped in commit loop, cp/mkdir errors trigger rollback, semver wrappers for `set -e` safety.
- **Files**: `scripts/apply-patch.sh`
- **Verification**: Successful apply, dry-run, SHA mismatch, path traversal, cp failure + rollback, all PASS.

### P0-002 Detail

- **Fix**: HandleListNodes uses session tenant; HandleGetNode returns 404 on cross-tenant; HandleListGPUs joins nodes for tenant scoping; HandleGetNodeSystem tenant check added; `NewContextWithSessionInfo` exported for tests.
- **Files**: `internal/server/api/agent_handlers.go`, `internal/server/api/resource_handlers.go`, `internal/server/auth/middleware.go`
- **Tests**: `TestTenantNodesScopedToList`, `TestTenantBBlockedFromTenantANode`, `TestTenantScopedGPUList`, `TestSystemQueryRespectsTenant`, `TestNodesNoHardcodedDefaultTenant`

### P0-003 Detail

- **Fix**: `LoginPage.doLogin()` passes `selectedTenantId` to `auth.login()`; `auth.login()` accepts optional `tenantId` and sends `tenant_id` in request body.
- **Files**: `web/src/pages/LoginPage.vue`, `web/src/stores/auth.ts`
- **Tests**: 4 Vitest tests (`sends tenant_id`, `does not send when not provided`, `sets isLoggedIn`, `does not set on failure`)

### P0-004 Detail

- **Fix**: Agent exits on default token (`LIGHTAI_ALLOW_INSECURE_DEFAULT_TOKEN=1` bypasses); Server validates agent_id binding on re-registration (409) and heartbeat (403).
- **Files**: `cmd/agent/main.go`, `internal/server/api/agent_handlers.go`
- **Tests**: `TestAgentRegistrationWithGoodToken`, `TestNodeIDAgentIDBindingOnReRegistration`, `TestHeartbeatAgentIDMismatchRejected`, `TestHeartbeatUnregisteredNodeRequestsReregister`, `TestResourceReportAgentIDBinding`, `TestNodeIDAgentIDBindingOnReRegisterSameAgentOK`

---

## P1 Status

| ID | Title | Status | Detail |
|----|-------|--------|--------|
| P1-001 | report_interval unused | **DOCUMENTED** | Comment added to config; RC2: implement or remove |
| P1-002 | advertise_addr unused | **DOCUMENTED** | Comment added to config; RC2: implement or remove |
| P1-003 | metrics.enabled starts HTTP | **FIXED** | `cfg.Metrics.Enabled` gates server startup |
| P1-004 | CPU-only LastSuccessAt | **FIXED** | System-only success updates LastSuccessAt |
| P1-005 | GPU empty → historical not marked | **FIXED** | GPUs not in report marked `unavailable` in handler |
| P1-006 | SQLite snapshot growth | **DEFERRED** | Risk: ~17K rows/day/node. Mitigation: cron DELETE. RC2: built-in retention |
| P1-007 | Write errors ignored | **DEFERRED** | Errors logged, not silent. Full transaction hardening in RC2 |
| P1-008 | Migration incomplete | **DEFERRED** | New tables in handler init, not Migrate(). RC2: all tables registered |
| P1-009 | NVIDIA temp files unsafe | **FIXED** | `mktemp` + `trap EXIT` in discover.sh/metrics.sh |
| P1-010 | http.Get no timeout | **FIXED** | `http.Client{Timeout: 5s}` in probeHTTP |
| P1-011 | Grafana password message | **FIXED** | Explicit message: env var does NOT modify existing DB password |
| P1-012 | Missing critical tests | **DEFERRED** | 14 API tests + 4 Vitest tests added this round. Full suite in RC2 |

### P1 Deferred Details

**P1-006**: Risk: node_system_snapshots grows ~17K rows/day per node. Workaround: `DELETE FROM node_system_snapshots WHERE collected_at < datetime('now','-7 days')`. RC2: configurable retention.

**P1-007**: Risk: system/disk/network write failures don't block GPU writes. Errors are logged. Workaround: monitor logs for "save system snapshot error". RC2: full transaction wrapper.

**P1-008**: Risk: gpu_devices/node_system_snapshots created in handler init, not via Migrate(). Workaround: manual schema review before upgrade. RC2: migrate all tables through Migrate().

**P1-012**: Risk: untested edge cases. Workaround: manual verification checklist. RC2: API/Shell/Patch integration test suite.

---

## P2 Status

| ID | Title | Status |
|----|-------|--------|
| P2-001 | Chinese startup messages | **DEFERRED** |
| P2-002 | Old version numbers | **FIXED** (README-RELEASE.md uses `<version>` placeholder) |
| P2-003 | Old credential paths | **FIXED** (references current runtime/ paths) |
| P2-004 | Grafana consistency | **FIXED** (via P1-011) |
| P2-005 | Frontend bundle size | **ACKNOWLEDGED** (no action for RC1) |

---

## Verification Results

```
go test ./...                    ALL PASS (8 packages, 14 API tests)
go vet ./...                     PASS
bash -n scripts/*.sh             ALL 23 OK
tests/test_patch_atomicity.sh    7 scenarios PASS
npm run build                    ✓ built
vitest auth.test.ts              4/4 PASS
nvidia-smi                       ✅ RTX 5090
discover.sh / metrics.sh         ✅ (mktemp)
check-glibc-compat.sh            ✅ 12 ELF, 0 violations
```

---

## Release Checklist

Before tagging:

```bash
git status                     # clean working tree
git diff --stat                # review all changes
git diff --check               # no whitespace errors
go test ./...                  # PASS
go vet ./...                   # PASS
bash -n scripts/*.sh           # ALL OK
```

After tagging, build release:

```bash
./scripts/package-release-docker.sh --bump patch
scripts/check-glibc-compat.sh dist
sha256sum -c dist/*.sha256
```

---

## MetaX 8-Card Field Validation

待现场执行（开发环境无 MetaX 硬件）：

```bash
# Agent metrics
curl -s -o /tmp/agent_metrics.txt -w '%{http_code}\n' http://127.0.0.1:19091/metrics
# Expected: 200

grep -c 'collected before' /tmp/agent_metrics.txt               # Expected: 0
grep -c '^lightai_gpu_available_status' /tmp/agent_metrics.txt  # Expected: 8
grep -c '^lightai_gpu_memory_total_bytes' /tmp/agent_metrics.txt # Expected: 8
grep -c '^lightai_gpu_memory_used_bytes' /tmp/agent_metrics.txt  # Expected: 8
grep -c '^lightai_gpu_memory_free_bytes' /tmp/agent_metrics.txt  # Expected: 8

# Server API
curl -s http://127.0.0.1:18080/api/gpus | jq 'length'      # Expected: 8
curl -s http://127.0.0.1:18080/api/gpus | jq '.[0].vendor' # Expected: "metax"

# Web: Nodes page → 8 MetaX C500, GPUs page → 8 MetaX C500
```

---

## Document Index

| File | Role |
|------|------|
| `docs/RC1_REVIEW_FIX_PLAN.md` | **主台账** — 唯一权威状态 |
| `docs/RC1_CODEX_REVIEW_TRACKING.md` | Codex Review 详细问题清单和验证记录 |
| `docs/GPU_COLLECTOR_ARCHITECTURE.md` | GPU 抽象架构文档 |
| `docs/archive/RC1_PATCH_TEST.md` | 已归档 — patch 测试临时文件 |
| `docs/REVIEW-GPUSTACK-AUDIT.md` | 独立 — GPUStack 审计（非 RC1） |
| `docs/REVIEW-GPUSTACK-UI.md` | 独立 — GPUStack UI 参考（非 RC1） |
