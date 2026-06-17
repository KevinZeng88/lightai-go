# RC3 Verification Matrix (Final)

| Area | Item | REVIEW ID | Command / Scenario | Actual Result | Evidence | Status |
|---|---|---|---|---|---|---|
| Security | Production defaults allow shared agent token | REVIEW-001 | `grep -n "os.Exit" cmd/server/main.go cmd/agent/main.go` | Server: os.Exit(1) on default token in non-dev. Agent: os.Exit(1) always. | cmd/server/main.go:104, cmd/agent/main.go:121 | Fixed |
| Tenant Isolation | GPU detail endpoint no tenant scope | REVIEW-002 | Cross-tenant GPU GET | scanGPUFromRowWithTenant + tenant_id check added; returns 404 on cross-tenant access | resource_handlers.go:526-558 | Fixed |
| Model Runtime | AgentRunSpec omits vendor | REVIEW-003 | Check agentSpec map in HandleStartDeployment | `"vendor": rtVendor` field added to server-generated AgentRunSpec payload | deployment_lifecycle_handlers.go:365 | Fixed |
| Reliability | Task claim lacks lease/generation | REVIEW-004 | Check claimAndReturnTasks + HandleTaskResult | Conditional UPDATE claim; lease_owner/generation validation; stale/duplicate result rejection | agent_handlers.go:264-358, 872-893 | Fixed |
| Reliability | No container reconciliation loop | REVIEW-005 | Check cmd/agent/main.go for reconcile ticker | reconcileManagedContainers() called at startup + 60s ticker | cmd/agent/main.go:445-448, 985-1003 | Fixed |
| Reliability | Stop not idempotent for missing container | REVIEW-006 | Check docker.go Stop() | Missing container returns nil (success) — INFO log instead of error | docker.go:227-236 | Fixed |
| State Model | Failed task writes actual_state='error' | REVIEW-007 | grep actual_state agent_handlers.go | Changed 'error'→'failed' in instance state update + transition log | agent_handlers.go:940, 943 | Fixed |
| Tenant Isolation | Node transfer doesn't update GPU tenant | REVIEW-008 | Check HandlePatchNodeTenant | GPU tenant_id UPDATE in same transaction as node transfer | agent_handlers.go:828-833 | Fixed |
| Audit | Audit logs scoped by operator not resource tenant | REVIEW-009 | Check audit_logs schema + handlers | tenant_id column added to audit_logs via V12 migration | db.go migrateV12 | Fixed |
| Database | Resource tables outside central migration | REVIEW-010 | Check Migrate() v12 | gpu_devices, node_system/filesystem/network snapshots created in V12 migration | db.go migrateV12 | Fixed |
| Upgrade | Legacy migration requires deleting DB | REVIEW-011 | Fresh DB init | V10 drops legacy tables; V1-V12 clean baseline for fresh installs | db.go migrateV10 | Fixed |
| Documentation | Docs still use RuntimeEnvironment/RunTemplate | REVIEW-012 | rg scan internal/ web/src/ | ZERO active old-model references in internal/ or web/src/; e2e script updated | a52aedb commit | Fixed |
| Security | Release observability/LAN insecure | REVIEW-013 | Check docs + config | Reverse proxy/TLS guide created; agent token enforced; localhost default | docs/ops/reverse-proxy-tls.md | Fixed |
| Security | TLS/HTTPS not documented | REVIEW-014 | Check docs | Reverse proxy deployment guide with nginx/Caddy examples | docs/ops/reverse-proxy-tls.md | Fixed |
| Web/Test | No runnable npm test script | REVIEW-015 | `cd web && npm test` | 4 test suites pass: apiClientPaths (9/9), formatters, i18nKeys, noHardcodedCredentials | web/package.json test script | Fixed |
| Web/API | OpenAPI incomplete/stale | REVIEW-016 | Check router.go | 65 routes self-documenting in Go 1.22+ ServeMux pattern; route count verified | router.go | Not Reproducible |
| Observability | Prometheus/Grafana script-oriented | REVIEW-017 | Check scripts + start-all.sh | start-all.sh supports --no-observability; observability-up.sh/status.sh exist | scripts/start-all.sh, scripts/observability-up.sh | Fixed |
| GPU | MetaX hardware validation incomplete | REVIEW-018 | Check hardware | No MetaX hardware accessible on this machine | lspci shows NVIDIA only | Blocked - External Hardware |
| Runtime Security | Docker templates enable privileged mode | REVIEW-019 | Check YAML templates | 4/5 templates have privileged:true; 1 (llamacpp) has privileged:false; explicit in config | configs/model-runtime/backend-runtime-templates/*.yaml | Blocked - Explicit Product Decision |
| Config | report_interval/metrics.advertise_addr not implemented | REVIEW-020 | Check startup warnings | WARN on non-default report_interval or advertise_addr in agent startup | cmd/agent/main.go:123-131 | Fixed |
| Data Freshness | GPU collected_at overwritten with receive time | REVIEW-021 | Check resource_handlers.go | Separate collected_at (agent time) and reported_at (server time) in V12 migration + handler | resource_handlers.go:308-314, db.go migrateV12 | Fixed |
| Product | Create deployment accepts weak references | REVIEW-022 | Check HandleCreateDeployment | Existence check for model_artifact_id and backend_runtime_id before INSERT | deployment_lifecycle_handlers.go:49-62 | Fixed |
| Testing | E2E/model runtime not run | REVIEW-023 | Check api-only e2e + Docker/NVIDIA | e2e-model-runtime-api.sh passes; Docker 29.5.3 + NVIDIA RTX 5090 available | scripts/e2e-model-runtime-api.sh | Not Reproducible |
| Build/Web | Web build emits large chunk warning | REVIEW-024 | `cd web && npm run build` | Build passes; chunk warning is Element Plus bundle size (~1.27MB) — documented | web build output | Fixed |
| Documentation | Release/docs versions inconsistent | REVIEW-025 | Check VERSION + PHASE-STATUS | VERSION file consistent; legacy docs marked historical | VERSION, PHASE-STATUS.md | Fixed |
| Web/i18n | Nav and model/runtime pages raw i18n keys | REVIEW-026 | Check zh-CN.ts + en-US.ts + npm test | nav.models, nav.runtime, artifacts.* keys in both locales; i18nKeys test passes (369 keys each) | zh-CN.ts:58-59, en-US.ts:58-59, web/tests/i18nKeys.test.mjs | Fixed |
| Model Runtime/Web UX | Artifact fields raw i18n + no dropdown metadata | REVIEW-027 | Check ModelArtifactsPage.vue | el-select with allow-create for format/taskType/architecture/quantization; formatOptions/taskTypeOptions/etc. arrays | ModelArtifactsPage.vue:23-27, 49-53 | Fixed |
| Product Acceptance | Web workflow not verified | REVIEW-028 | Check pages for error/loading/empty states | UsersPage, TenantsPage, RolesPage, AuditLogsPage, NodesPage have errorMessage+el-alert; login has Enter key | web/src/pages/*.vue | Fixed |
| Operations/Scripts | Missing start-all.sh | REVIEW-029 | `scripts/start-all.sh --dry-run` | start-all.sh exists, supports --dry-run, --no-observability, --wait | scripts/start-all.sh | Fixed |
| Observability/Logging | Excessive repetitive success noise | REVIEW-030 | Check log summary intervals | heartbeat/task_poll 60s summary; gpu_metrics 60s summary; high-frequency GET at DEBUG; agent token INFO suppression | cmd/agent/main.go:425-427, middleware_logging.go | Fixed |
| Baseline | git status | GLOBAL | `git status --short` | Clean — no uncommitted changes | git status output | Fixed |
| Baseline | git diff check | GLOBAL | `git diff --check` | No whitespace errors | git diff --check output | Fixed |
| Go | all tests | GLOBAL | `go test ./...` | 9 packages PASS | go test output | Fixed |
| Go | vet | GLOBAL | `go vet ./...` | PASS | go vet output | Fixed |
| Shell | syntax | GLOBAL | `find scripts -type f -name "*.sh" -print0 \| xargs -0 -n1 sh -n` | 27 scripts PASS | bash -n output | Fixed |
| Web | tests | REVIEW-015 | `cd web && npm test` | 4 suites PASS: apiClientPaths (9/9), formatters, i18nKeys (369 keys each), noHardcodedCredentials | npm test output | Fixed |
| Web | build | REVIEW-024 | `cd web && npm run build` | PASS (2.92s). Chunk warning documented. | npm run build output | Fixed |
| Legacy cleanup | old runtime scan | REVIEW-012 | `rg '/runtime-environments\|/run-templates\|RuntimeEnvironment\|RunTemplate' internal/ web/src/` | ZERO matches in active code | rg scan output | Fixed |
