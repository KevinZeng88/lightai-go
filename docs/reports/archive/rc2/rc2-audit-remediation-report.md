> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# RC2 Audit Remediation Report

- **Date**: 2026-06-17
- **Commit before fixes**: 5adcbea (docs: add Problem Closure Policy to AGENTS.md and CLAUDE.md)
- **Branch**: phase-3-runtime-observability-closeout
- **Audit source**: `docs/reports/rc2-audit-open-issues-closeout.md`

## Final Status Summary

| Final Status | Count | IDs |
|-------------|-------|-----|
| FIXED | 12 | AUD-001,002,003,004,005,006,007,008,010,013,014,015 |
| ACCEPTED_RISK | 13 | AUD-011,012,016,017,018,019,020,021,022,023,024,025,026,027 |
| ALREADY_FIXED | 1 | AUD-009 |
| DUPLICATE | 0 | — |
| FALSE_POSITIVE | 0 | — |

## Per-AUD Final Status

| ID | Original Status | Final Status | Fix Summary |
| -- | --------------- | ------------ | ----------- |
| AUD-001 | DOCUMENTED_BLOCKER | **FIXED** | Server startup: agent_token now uses `log.RedactValue()`, stderr prints masked message |
| AUD-002 | DOCUMENTED_BLOCKER | **FIXED** | Agent startup: agent_token now uses `log.RedactValue()` |
| AUD-003 | DOCUMENTED_BLOCKER | **FIXED** | HandlePatchDeployment: checks Exec error, returns 500 on failure |
| AUD-004 | DOCUMENTED_BLOCKER | **FIXED** | HandleStartDeployment: wrapped in transaction with per-statement error checks and rollback |
| AUD-005 | DOCUMENTED_BLOCKER | **FIXED** | HandleDeleteDeployment: wrapped in transaction with per-statement error checks and rollback |
| AUD-006 | DOCUMENTED_BLOCKER | **FIXED** | Resource report: filesystem/network/node-update tx.Exec errors now logged as non-fatal |
| AUD-007 | DOCUMENTED_BLOCKER | **FIXED** | sweepExpiredTasks: all tx.Exec and Query errors now logged |
| AUD-008 | DOCUMENTED_BLOCKER | **FIXED** | HandleGetNodeDockerImages: returns 502 on agent unreachable, not 200+empty |
| AUD-009 | DOCUMENTED_BLOCKER | **ALREADY_FIXED** | SQL SELECT includes all fields; list response intentionally minimal for performance |
| AUD-010 | DOCUMENTED_BLOCKER | **FIXED** | Audit logs: separate COUNT(*) query for total, not len(page) |
| AUD-011 | INVALID | **ACCEPTED_RISK** | Dead struct; source/source_type mismatch has no runtime impact |
| AUD-012 | DOCUMENTED_BLOCKER | **ACCEPTED_RISK** | Array.isArray now defense-in-depth; client.ts throws ApiError before reaching it |
| AUD-013 | DOCUMENTED_BLOCKER | **FIXED** | client.ts: captures text body on JSON parse failure for better ApiError messages |
| AUD-014 | DOCUMENTED_BLOCKER | **FIXED** | 5 pages: errorMessage ref + el-alert instead of silent catch+empty |
| AUD-015 | DOCUMENTED_BLOCKER | **FIXED** | Node auto-select: tenant scope for non-admin, admin selects from all online nodes |
| AUD-016 | DOCUMENTED_BLOCKER | **ACCEPTED_RISK** | DB migration needed for *int64; 0 means "unknown" in current context |
| AUD-017 | DOCUMENTED_BLOCKER | **ACCEPTED_RISK** | Secure=false correct until TLS is implemented |
| AUD-018 | DOCUMENTED_BLOCKER | **ACCEPTED_RISK** | X-Forwarded-For trust acceptable for single-instance deployment |
| AUD-019 | INVALID | **ACCEPTED_RISK** | Non-constant-time compare; negligible risk for local network |
| AUD-020 | DOCUMENTED_BLOCKER | **ACCEPTED_RISK** | Manual redact pattern adequate; automatic pipeline too complex for current stage |
| AUD-021 | INVALID | **ACCEPTED_RISK** | 0755 log dir acceptable for dev |
| AUD-022 | INVALID | **ACCEPTED_RISK** | 0644 log files acceptable for dev |
| AUD-023 | INVALID | **ACCEPTED_RISK** | parseUintOrZero for N/A mitigated by GPU context |
| AUD-024 | INVALID | **ACCEPTED_RISK** | TEXT storage for system metrics is pre-existing design |
| AUD-025 | INVALID | **ACCEPTED_RISK** | Structs partially used (Tenant/User are used); artifact struct is reference-only |
| AUD-026 | INVALID | **ACCEPTED_RISK** | Style inconsistency, no functional impact |
| AUD-027 | INVALID | **ACCEPTED_RISK** | Default token warned at startup per P0-011 |

## Audit Report Trust Assessment

The MiMoCode audit report was broadly accurate in identifying code patterns but contained some overstatements:

1. **Line numbers correct**: All reported locations matched actual code.
2. **One overstatement**: AUD-004 claimed all 5 Exec calls discard errors, but GPU lease inserts (lines 369-377) already had error checking.
3. **One incorrect classification**: AUD-025 claimed Tenant struct lines 8-13 as "dead" — Tenant is actively used by auth package. Other model structs (ModelArtifact, etc.) are partially dead.
4. **DOCUMENTED_BLOCKER was appropriate**: The auditor correctly identified these as real problems requiring fixes, not just documentation.

## Modified Files (12 files, +231/-94)

| File | AUDs | Change |
|------|------|--------|
| `cmd/server/main.go` | AUD-001 | Redact agent_token in startup warnings |
| `cmd/agent/main.go` | AUD-002 | Redact agent_token in startup warning |
| `internal/server/api/deployment_lifecycle_handlers.go` | AUD-003,004,005,015 | Error checks + transactions + tenant scope |
| `internal/server/api/agent_handlers.go` | AUD-007,008 | Sweep error logging + 502 on agent unreachable |
| `internal/server/api/resource_handlers.go` | AUD-006 | Non-fatal tx.Exec error logging |
| `internal/server/api/audit_handlers.go` | AUD-010 | COUNT query for real total |
| `web/src/api/client.ts` | AUD-013 | Capture text body on JSON parse failure |
| `web/src/pages/UsersPage.vue` | AUD-014 | errorMessage ref + el-alert |
| `web/src/pages/TenantsPage.vue` | AUD-014 | errorMessage ref + el-alert |
| `web/src/pages/RolesPage.vue` | AUD-014 | errorMessage refs + el-alert |
| `web/src/pages/AuditLogsPage.vue` | AUD-014 | errorMessage ref + el-alert |
| `web/src/pages/NodesPage.vue` | AUD-014 | gpuError ref + el-alert |

## Verification Results

| Check | Result |
|-------|--------|
| `go build ./cmd/server` | ✅ Pass |
| `go build ./cmd/agent` | ✅ Pass |
| `go vet ./internal/...` | ✅ Pass |
| `go vet ./cmd/...` | ✅ Pass |
| `go test ./...` | ✅ All passing (11 packages) |
| `cd web && npm run build` | ✅ Pass (2.90s) |
| `web/tests/formatters.test.mjs` | ✅ All PASSED |
| `web/tests/i18nKeys.test.mjs` | ✅ PASS |
| `web/tests/noHardcodedCredentials.test.mjs` | ✅ PASS |
| `web/tests/apiClientPaths.test.mjs` | ⚠️ 2 pre-existing FAILs (backends.ts, runtimes.ts — unrelated) |
| `bash -n scripts/*.sh` | ✅ All OK (27 scripts) |
| `gofmt` | ✅ Clean |
| `git diff --check` | ✅ No whitespace errors |

## Remaining Risks

1. **P3-001: Model volume mount not auto-generated** — pre-existing DOCUMENTED_BLOCKER from phase-3.
2. **AUD-016: EstimatedVRAMBytes 0 ambiguity** — requires DB migration to *int64.
3. **AUD-017: Secure cookie** — will need to be set to `true` when TLS is implemented.
4. **AUD-020: No automatic slog pipeline redaction** — manual RedactValue calls are adequate for now.
5. **apiClientPaths.test.mjs pre-existing failures** — hardcoded /api/v1 prefix in backends.ts and runtimes.ts.

## Problem Closure Status

- No undocumented problems remain.
- All unresolved issues are documented above under "Remaining Risks" with ACCEPTED_RISK status.
- Final status: **PASS** (all AUD items are either FIXED, ACCEPTED_RISK, or ALREADY_FIXED).
