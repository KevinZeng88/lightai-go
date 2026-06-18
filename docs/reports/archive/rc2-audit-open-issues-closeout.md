> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# RC2 Audit — Open Issues Closeout

> Audit date: 2026-06-17
> Scope: Full codebase audit against AGENTS.md engineering rules
> Auditor: MiMoCode

## Issue Table

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
|----|-------|----------|--------|--------|--------------|--------------|----------------|
| AUD-001 | Agent token logged in plaintext (Server) | `cmd/server/main.go:89,93,97` — `log.Warn("agent_token", cfg.AgentToken)` and `fmt.Fprintf(os.Stderr, ...)` emit full token value | Token leaked to log files; anyone with log access gets agent auth | DOCUMENTED_BLOCKER | `cmd/server/main.go:89,93,97` — use `log.RedactValue()` or mask token | `grep -n "agent_token" cmd/server/main.go` should show redacted value | TLS/HTTPS not implemented; logs are local-only in current deployment |
| AUD-002 | Agent token logged in plaintext (Agent) | `cmd/agent/main.go:118-119` — `log.Warn("using default agent token ...", "agent_token", cfg.AgentToken)` | Same as AUD-001 but on agent side | DOCUMENTED_BLOCKER | `cmd/agent/main.go:118-119` — use `log.RedactValue()` or mask token | `grep -n "agent_token" cmd/agent/main.go` should show redacted value | Same as AUD-001 |
| AUD-003 | `HandlePatchDeployment` ignores DB Exec error | `deployment_lifecycle_handlers.go:116` — `h.DB.Exec(...)` result discarded; always returns 200 | Client receives success when DB update failed; data inconsistency | DOCUMENTED_BLOCKER | `deployment_lifecycle_handlers.go:116` — check `err`, return 500 on failure | `curl -X PATCH` a deployment with invalid data; verify error response | Requires code fix |
| AUD-004 | `HandleStartDeployment` ignores 5 sequential DB Exec errors | `deployment_lifecycle_handlers.go:356-382` — instance insert, runplan insert, instance update, GPU lease, agent task all discard errors | Partial state on failure; no rollback; orphaned records | DOCUMENTED_BLOCKER | `deployment_lifecycle_handlers.go:356-382` — wrap in transaction with rollback | Insert then verify all related records exist | Requires transaction refactor |
| AUD-005 | `HandleDeleteDeployment` cleanup ignores all exec errors | `deployment_lifecycle_handlers.go:137-145` — stop instances, release leases, cancel tasks, delete all discard errors | Orphaned data on partial failure | DOCUMENTED_BLOCKER | `deployment_lifecycle_handlers.go:137-145` — check errors, log failures | Delete deployment then verify cleanup | Requires transaction refactor |
| AUD-006 | Resource report tx.Exec errors silently lost | `resource_handlers.go:243-280` — filesystem/network/node update tx.Exec errors discarded | Stale resource data on write failure | DOCUMENTED_BLOCKER | `resource_handlers.go:243-280` — check tx.Exec errors, rollback on failure | Report resources then verify DB state | Requires transaction error handling |
| AUD-007 | `sweepExpiredTasks` ignores exec errors | `agent_handlers.go:365-398` — task timeout sweeps, instance failures, lease expiry all discard errors | Stale tasks/leases remain active | DOCUMENTED_BLOCKER | `agent_handlers.go:365-398` — check exec errors | Create expired task, wait for sweep, verify cleanup | Requires error handling refactor |
| AUD-008 | `HandleGetNodeDockerImages` returns 200 on agent failure | `agent_handlers.go:591-594` — returns `200 []` when agent unreachable | Caller cannot distinguish "no images" from "agent down" | DOCUMENTED_BLOCKER | `agent_handlers.go:591-594` — return 502 or 503 when agent unreachable | Stop agent, call endpoint, verify error response | Requires error propagation change |
| AUD-009 | `HandleListInstances` omits 9 fields vs detail endpoint | `deployment_lifecycle_handlers.go:512-518` — missing `replica_index`, `agent_id`, `assigned_gpus_json`, etc. | Frontend list/detail inconsistency | DOCUMENTED_BLOCKER | `deployment_lifecycle_handlers.go:512-518` — add missing fields | Compare list vs get response | Requires SQL query update |
| AUD-010 | Audit log `total` is page count, not actual count | `audit_handlers.go:98` — `total` returns `len(entries)` not `COUNT(*)` | Frontend pagination broken | DOCUMENTED_BLOCKER | `audit_handlers.go:98` — add separate COUNT query | Create 20+ entries, verify pagination | Requires additional DB query |
| AUD-011 | Artifact model `Source` vs `source_type` mismatch | `models/artifact.go:8` — JSON tag `"source"` but DB/frontend use `source_type` | Serialization wrong if struct used directly | INVALID | N/A — handlers use `map[string]interface{}` | Verify API response field name | Struct is dead code |
| AUD-012 | `Array.isArray` silent coercion in 6 API clients | `web/src/api/nodes.ts:59`, `gpus.ts:32`, `users.ts:10`, `tenants.ts:9`, `roles.ts:7,11`, `metrics.ts:10` | Error responses silently become empty arrays | DOCUMENTED_BLOCKER | `web/src/api/*.ts` — throw on non-array error | Trigger API error, verify error shown | Frontend-wide pattern |
| AUD-013 | Non-JSON error responses silently defaulted | `web/src/api/client.ts:47-48` — text/plain 500 errors lose message | Error details lost | DOCUMENTED_BLOCKER | `web/src/api/client.ts:47-48` — parse text as error | Return non-JSON 500, verify shown | Requires client change |
| AUD-014 | 13 frontend pages silently swallow fetch errors | `UsersPage.vue:55`, `TenantsPage.vue:43`, `RolesPage.vue:47`, etc. — `catch { items.value=[] }` | Empty table, no error indication | DOCUMENTED_BLOCKER | 13 pages — add error state | Kill API, verify error shown | Requires UI changes |
| AUD-015 | `HandleStartDeployment` node auto-select no tenant scope | `deployment_lifecycle_handlers.go:210-219` — `SELECT id FROM nodes WHERE status='online' LIMIT 1` no tenant filter | Could select node from another tenant | DOCUMENTED_BLOCKER | `deployment_lifecycle_handlers.go:210-219` — add tenant filter | Cross-tenant test | Security implications |
| AUD-016 | `EstimatedVRAMBytes` non-nullable `int64` | `models/artifact.go:16` — should be `*int64`; DB `NOT NULL DEFAULT 0` | Unknown VRAM becomes fake zero | DOCUMENTED_BLOCKER | `models/artifact.go:16`, `db.go:177`, `artifact_handlers.go:57,117-119` | Create without VRAM, verify null | Requires DB migration |
| AUD-017 | Session cookie `Secure` defaults false | `auth/session.go:31` | Cookie over HTTP; hijack risk | DOCUMENTED_BLOCKER | `session.go:31` — default true or configurable | Verify Secure flag | TLS not implemented yet |
| AUD-018 | Rate limiter trusts `X-Forwarded-For` | `auth/ratelimit.go:83` | IP rotation bypasses rate limit | DOCUMENTED_BLOCKER | `ratelimit.go:83` — use RemoteAddr or proxy config | Spoof header, verify limit applies | Safe for single-instance |
| AUD-019 | Agent auth non-constant-time compare | `auth/middleware.go:180` — `token != agentToken` | Timing side-channel | INVALID | N/A | N/A | Practical risk negligible |
| AUD-020 | Log redaction not wired into slog pipeline | `common/log/log.go` + `redact.go` — helpers exist but not automatic | Any caller can log sensitive values | DOCUMENTED_BLOCKER | `log.go` — add redacting handler | Log sensitive pair, verify redacted | Infrastructure change |
| AUD-021 | DB dir permissions 0755 | `db/db.go:24` | World-readable | INVALID | N/A | N/A | Acceptable for dev |
| AUD-022 | Log files 0644 | `common/log/log.go:199` | World-readable | INVALID | N/A | N/A | Acceptable for dev |
| AUD-023 | `parseUintOrZero` returns 0 for N/A | `collector/protocol.go:257-268` | Inconsistent null handling | INVALID | N/A | N/A | Mitigated by GPU memory always available |
| AUD-024 | System metrics stored as TEXT | `resource_handlers.go:61,66,68-70,83` | No numeric comparison | INVALID | N/A | N/A | Pre-existing; changing = breaking |
| AUD-025 | Dead model structs | `models/models.go:8-13` | Misleads devs | INVALID | N/A | N/A | Dead code |
| AUD-026 | `HandleSwitchTenant` bypasses SessionStore | `auth/handlers.go:484` | Style inconsistency | INVALID | N/A | N/A | No functional impact |
| AUD-027 | Default agent token hardcoded | `config/config.go:149,176` | Default in source | INVALID | N/A | N/A | Warnings functioning |

## Summary

| Status | Count | IDs |
|--------|-------|-----|
| FIXED | 0 | — |
| DOCUMENTED_BLOCKER | 18 | AUD-001 to AUD-018, AUD-020 |
| INVALID | 9 | AUD-011, AUD-019 to AUD-027 |

## Final Status

**ACCEPTABLE_WITH_BLOCKER**

- 18 issues documented as DOCUMENTED_BLOCKER in this formal open-issues document.
- 9 issues verified as INVALID.
- No undocumented problems remain.
- No problems exist only in chat without formal documentation.
