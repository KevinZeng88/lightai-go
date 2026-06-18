> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# RC2 Audit Verification Matrix (Final)

| ID | Report Status | Verified | Final Status | Evidence / Fix | Risk | Notes |
| -- | ------------- | -------- | ------------ | -------------- | ---- | ----- |
| AUD-001 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | Server: `log.RedactValue("agent_token", cfg.AgentToken)` in Warn/Error; stderr prints "value redacted" | P1 | Only fires for known defaults |
| AUD-002 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | Agent: `log.RedactValue("agent_token", cfg.AgentToken)` | P1 | Only fires for known defaults |
| AUD-003 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | HandlePatchDeployment: checks `_, err := h.DB.Exec(...); err != nil` → 500 | P0 | Was always returning 200 |
| AUD-004 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | HandleStartDeployment: `tx, _ := h.DB.Begin()` + per-statement error checks + rollback | P0 | GPU lease errors were already checked (audit overstated) |
| AUD-005 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | HandleDeleteDeployment: transaction + 6 per-statement error checks + rollback | P0 | Was 6 silent Exec calls |
| AUD-006 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | Resource report: filesystem/network/node-update tx.Exec errors now logged as non-fatal | P1 | GPU insert/update were already checked |
| AUD-007 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | sweepExpiredTasks: 4 tx.Exec + 1 tx.Query errors now logged | P1 | Within tx in claimAndReturnTasks |
| AUD-008 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | HandleGetNodeDockerImages: returns `writeError(w, 502, "agent unreachable")` | P0 | Was returning 200+[] |
| AUD-009 | DOCUMENTED_BLOCKER | ALREADY_FIXED | **ALREADY_FIXED** | SQL SELECT includes all 20 columns; list response intentionally minimal | P1 | Detail endpoint has full data |
| AUD-010 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | Audit logs: separate `SELECT COUNT(*)` query → `total` is real count | P1 | Was `len(entries)` = page size |
| AUD-011 | INVALID | ACCEPTED_RISK | **ACCEPTED_RISK** | `Source string \`json:"source"\`` in dead struct; handlers use map[string]interface{} | P2 | Zero runtime impact |
| AUD-012 | DOCUMENTED_BLOCKER | ACCEPTED_RISK | **ACCEPTED_RISK** | Array.isArray is now defense-in-depth; client.ts throws ApiError before reaching it | P2 | Mitigated by client.ts hardening |
| AUD-013 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | client.ts: captures `resp.text()` on JSON parse failure → better ApiError.message | P1 | Was silently becoming `{}` |
| AUD-014 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | 5 pages: `errorMessage` ref + `<el-alert>` instead of `catch { items=[] }` | P1 | UsersPage, TenantsPage, RolesPage, AuditLogsPage, NodesPage |
| AUD-015 | DOCUMENTED_BLOCKER | TRUE_POSITIVE | **FIXED** | Node auto-select: `AND tenant_id = ?` for non-admin; admin bypasses filter | P0 | Cross-tenant isolation |
| AUD-016 | DOCUMENTED_BLOCKER | ACCEPTED_RISK | **ACCEPTED_RISK** | EstimatedVRAMBytes int64 → 0 means unknown; needs DB migration for *int64 | P2 | Migration required |
| AUD-017 | DOCUMENTED_BLOCKER | ACCEPTED_RISK | **ACCEPTED_RISK** | Secure=false; TLS not implemented per AGENTS.md | P2 | Setting true would break cookies without TLS |
| AUD-018 | DOCUMENTED_BLOCKER | ACCEPTED_RISK | **ACCEPTED_RISK** | X-Forwarded-For trust; acceptable for single-instance deployment | P2 | Production deploy docs should note |
| AUD-019 | INVALID | ACCEPTED_RISK | **ACCEPTED_RISK** | Non-constant-time token compare; negligible risk for local network | P2 | Report's INVALID was correct |
| AUD-020 | DOCUMENTED_BLOCKER | ACCEPTED_RISK | **ACCEPTED_RISK** | Redact helpers exist but not automatic; manual call pattern adequate | P2 | AUD-001/002 demonstrate the pattern |
| AUD-021 | INVALID | ACCEPTED_RISK | **ACCEPTED_RISK** | log dir 0755; acceptable for dev/local deployment | P2 | Report's INVALID was correct |
| AUD-022 | INVALID | ACCEPTED_RISK | **ACCEPTED_RISK** | log files 0644; acceptable for dev/local deployment | P2 | Report's INVALID was correct |
| AUD-023 | INVALID | ACCEPTED_RISK | **ACCEPTED_RISK** | parseUintOrZero for N/A; mitigated by GPU memory always having real values | P2 | Report's INVALID was correct |
| AUD-024 | INVALID | ACCEPTED_RISK | **ACCEPTED_RISK** | System metrics as TEXT; pre-existing design tradeoff | P2 | Report's INVALID was correct |
| AUD-025 | INVALID | ACCEPTED_RISK | **ACCEPTED_RISK** | Some model structs (ModelArtifact) are dead; Tenant/User ARE used | P2 | Report partially overstated |
| AUD-026 | INVALID | ACCEPTED_RISK | **ACCEPTED_RISK** | SwitchTenant direct DB Exec; style inconsistency only | P2 | Report's INVALID was correct |
| AUD-027 | INVALID | ACCEPTED_RISK | **ACCEPTED_RISK** | Default token hardcoded; startup warnings per P0-011 detect it | P2 | Report's INVALID was correct |
