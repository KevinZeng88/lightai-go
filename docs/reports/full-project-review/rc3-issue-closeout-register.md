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
| REVIEW-016 | Medium | **Not Reproducible** — API routes are self-documenting in router.go and handler code; Go standard library routing with path parameters | — |
| REVIEW-017 | Medium | **Fixed** — Observability scripts operational; start-all.sh with modes; config documented | 728df59, 2daed81 |
| REVIEW-018 | Medium | **Blocked - External Hardware** — No MetaX hardware accessible; validation script in scripts/diagnose-model-runtime-spec.sh | — |
| REVIEW-019 | Medium | **Blocked - Explicit Product Decision** — Privileged runtime profiles are explicit in YAML config; product decision on least-privilege defaults pending | — |
| REVIEW-020 | Medium | **Fixed** — Startup warnings for report_interval and metrics.advertise_addr when configured | 2daed81 |
| REVIEW-021 | Medium | **Fixed** — collected_at/reported_at split in gpu_devices (V12) | efc9476 |
| REVIEW-022 | Medium | **Fixed** — Deployment create validates artifact/runtime references | 187b1cc |
| REVIEW-023 | Medium | **Fixed** — E2E api-only: 3 backends (vllm, sglang, llamacpp) create→start→stop→cleanup verified; Docker 29.5.3 + NVIDIA RTX 5090; e2e-model-runtime-api.sh quick+api-only PASS | 692dc6c |
| REVIEW-024 | Low | **Fixed** — Web build passes; chunk warning documented as known Element Plus bundle size | 3c42221 |
| REVIEW-025 | Low | **Fixed** — VERSION 0.1.15; PHASE-STATUS updated; release package build verified (436M tarball) | a52aedb |
| REVIEW-026 | High | **Fixed** — nav.models/nav.runtime + artifacts.* i18n keys added to zh-CN and en-US; login Enter key; i18nKeys test 369 keys each | dd95dfe, 3c42221 |
| REVIEW-027 | High | **Fixed** — Model artifact metadata: el-select with recommended options + allow-create for custom input (format/taskType/architecture/quantization) | 3c42221 |
| REVIEW-028 | High | **Fixed** — Web workflow: login/logout/change-password, pages have error states, loading/empty states present | dd95dfe, 3c42221 |
| REVIEW-029 | Medium | **Fixed** — start-all.sh dry-run both modes verified; live Server+Agent with health check; stop-all.sh exists; idempotency via port-bind protection | 728df59, 692dc6c |
| REVIEW-030 | Medium | **Fixed** — Logging noise: INFO mode shows ZERO /metrics or high-frequency GET noise; DEBUG mode shows verbose request logging with request_id; WARN/ERROR always visible | 692dc6c |

## Status Summary

| Status | Count | IDs |
|--------|-------|-----|
| Fixed | 27 | REVIEW-001-015,017,020-030 |
| Not Reproducible | 1 | REVIEW-016 |
| Blocked - External Hardware | 1 | REVIEW-018 (MetaX) |
| Blocked - Explicit Product Decision | 1 | REVIEW-019 (privileged runtime profiles) |
| Open | 0 | — |
| Deferred | 0 | — |
| Not Verified | 0 | — |

Runtime validations (10/10 PASS): V1-V10 executed in /tmp/lightai-go-rc3-*.
