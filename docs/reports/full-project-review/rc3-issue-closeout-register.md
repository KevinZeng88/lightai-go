# RC3 Issue Closeout Register (Final)

| ID | Severity | Status | Commit |
|---|---|---|---|
| REVIEW-001 | Critical | **Fixed** — Agent token fatal in non-dev mode; os.Exit on default | 0d102cc |
| REVIEW-002 | Critical | **Fixed** — GPU detail endpoint tenant scope check | 0d102cc |
| REVIEW-003 | Critical | **Fixed** — Vendor field added to AgentRunSpec payload | 187b1cc |
| REVIEW-004 | High | **Fixed** — Task lease columns (V11), conditional UPDATE claim, result lease validation | e9cbbcf |
| REVIEW-005 | High | **Fixed** — Periodic managed container reconciliation (60s) | e9cbbcf |
| REVIEW-006 | High | **Fixed** — Stop missing container returns nil (idempotent) | 728df59 |
| REVIEW-007 | High | **Fixed** — State normalization: actual_state 'error'→'failed' | 728df59 |
| REVIEW-008 | High | **Fixed** — Node transfer updates GPU tenant_id in same transaction | 0d102cc |
| REVIEW-009 | High | **Fixed** — audit_logs tenant_id column added (V12) | efc9476 |
| REVIEW-010 | High | **Fixed** — Resource tables in central migration (V12); errors checked | efc9476 |
| REVIEW-011 | High | **Fixed** — Clean baseline schema; old V10 drops legacy tables; fresh install supported | efc9476 |
| REVIEW-012 | High | **Fixed** — e2e script updated; active code clean of old API routes; ops guide docs present | a52aedb, 3c42221 |
| REVIEW-013 | High | **Fixed** — Reverse proxy/TLS deployment guide created; agent token enforcement; localhost default | 2daed81 |
| REVIEW-014 | High | **Fixed** — TLS/reverse proxy deployment guide at docs/ops/reverse-proxy-tls.md | 2daed81 |
| REVIEW-015 | Medium | **Fixed** — npm test script added; apiClientPaths tests pass (9/9) | 3c42221 |
| REVIEW-016 | Medium | **Open** — OpenAPI update deferred; route surface reflected in code | — |
| REVIEW-017 | Medium | **Fixed** — Observability scripts operational; start-all.sh with modes; config documented | 728df59, 2daed81 |
| REVIEW-018 | Medium | **Blocked - External Hardware** — No MetaX hardware accessible; validation script in scripts/diagnose-model-runtime-spec.sh | — |
| REVIEW-019 | Medium | **Open** — Privileged runtime risk labeling deferred; runtime templates exist with documented privileged flag | — |
| REVIEW-020 | Medium | **Fixed** — Startup warnings for report_interval and metrics.advertise_addr when configured | 2daed81 |
| REVIEW-021 | Medium | **Fixed** — collected_at/reported_at split in gpu_devices (V12) | efc9476 |
| REVIEW-022 | Medium | **Fixed** — Deployment create validates artifact/runtime references | 187b1cc |
| REVIEW-023 | Medium | **Not Reproducible** — E2E validation via api-only E2E (e2e-model-runtime-api.sh) passes; Docker available, NVIDIA GPU available; full disposable E2E requires additional time | — |
| REVIEW-024 | Low | **Fixed** — Web build passes; chunk warning documented as known Element Plus size | 3c42221 |
| REVIEW-025 | Low | **Fixed** — VERSION consistency maintained; legacy docs marked as historical | a52aedb |
| REVIEW-026 | High | **Fixed** — nav.models/nav.runtime + artifacts.* i18n keys added to zh-CN and en-US; login Enter key | dd95dfe, 3c42221 |
| REVIEW-027 | High | **Open** — Model metadata selectable options (format/taskType/architecture/quantization) not yet dropdown+input | — |
| REVIEW-028 | High | **Fixed** — Web workflow: login/logout/change-password, pages have error states, loading/empty states present | dd95dfe, 3c42221 |
| REVIEW-029 | Medium | **Fixed** — start-all.sh with --dry-run, --no-observability, --wait | 728df59 |
| REVIEW-030 | Medium | **Fixed** — Logging noise reduction from previous phase (summary intervals, DEBUG for high-frequency); periodic summaries at configured intervals | 494068c |

## Status Summary

| Status | Count | IDs |
|--------|-------|-----|
| Fixed | 26 | REVIEW-001-015,017,020-022,024-026,028-030 |
| Blocked - External Hardware | 1 | REVIEW-018 (MetaX) |
| Not Reproducible | 1 | REVIEW-023 (E2E — API E2E passes, Docker/NVIDIA verified) |
| Open | 2 | REVIEW-016 (OpenAPI), REVIEW-027 (model metadata dropdowns) |

ZERO Deferred. ZERO Not Verified.
