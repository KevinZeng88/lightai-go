# RC3 Verification Matrix

| Area | Item | REVIEW ID | Command / Scenario | Environment | Expected Result | Actual Result | Evidence | Status |
|---|---|---|---|---|---|---|---|---|
| Security | Production defaults allow shared agent token | REVIEW-001 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Tenant Isolation | GPU detail endpoint does not enforce tenant scope | REVIEW-002 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Model Runtime | Server AgentRunSpec omits vendor and Docker driver ignores entrypoint | REVIEW-003 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Reliability | Agent task claim lacks lease/generation/idempotency | REVIEW-004 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Reliability | No complete model instance reconciliation loop | REVIEW-005 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Reliability | Stop path is not idempotent when container is missing | REVIEW-006 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| State Model | Failed task writes non-canonical actual_state='error' | REVIEW-007 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Tenant Isolation | Node transfer does not transfer existing GPU records | REVIEW-008 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Audit | Audit logs scoped by operator membership, not resource tenant | REVIEW-009 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Database | Resource tables created outside central migration and errors ignored | REVIEW-010 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Upgrade | Legacy tenant migration requires deleting DB | REVIEW-011 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Documentation | Production/runtime docs still use removed API objects | REVIEW-012 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Security | Release config exposes observability/LAN surfaces insecurely | REVIEW-013 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Security | TLS/HTTPS not implemented/documented for release exposure | REVIEW-014 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Web/Test | Web tests exist but no runnable test script/dependency | REVIEW-015 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Web/API | OpenAPI incomplete and stale | REVIEW-016 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Observability | Prometheus/Grafana supervision script-oriented and unclear | REVIEW-017 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| GPU | MetaX real hardware validation incomplete | REVIEW-018 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Runtime Security | Docker templates enable privileged mode | REVIEW-019 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Config | report_interval and metrics.advertise_addr documented as not implemented | REVIEW-020 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Data Freshness | Server overwrites GPU collected_at with receive time | REVIEW-021 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Product | Create deployment accepts weak references | REVIEW-022 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Testing | E2E/model runtime validation was not run | REVIEW-023 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Build/Web | Web build emits large chunk warning | REVIEW-024 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Documentation | Release/docs versions inconsistent | REVIEW-025 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Web/i18n | Main navigation and model/runtime pages show raw i18n keys | REVIEW-026 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Model Runtime/Web UX | Model artifact fields raw i18n keys and insufficient selectable/custom metadata inputs | REVIEW-027 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Product Acceptance | Web workflow completeness not verified against real operator tasks | REVIEW-028 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Operations/Scripts | Missing start-all.sh counterpart for stop-all.sh | REVIEW-029 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Observability/Logging | Server and Agent logs contain excessive repetitive success noise | REVIEW-030 | Implement required action and run related tests | Source + disposable env where needed | Issue closed with passing verification | TBD | TBD | Open |
| Baseline | git status | GLOBAL | `git status --short` | project root | Status recorded before and after work | TBD | TBD | Open |
| Baseline | git diff check | GLOBAL | `git diff --check` | project root | No whitespace errors | TBD | TBD | Open |
| Go | all tests | GLOBAL | `go test ./...` | project root | PASS | TBD | TBD | Open |
| Go | vet | GLOBAL | `go vet ./...` | project root | PASS | TBD | TBD | Open |
| Shell | syntax | GLOBAL | `find scripts -type f -name "*.sh" -print0 | xargs -0 -n1 sh -n` | project root | PASS | TBD | TBD | Open |
| Web | tests | REVIEW-015 | `cd web && npm test` | project root/web | PASS | TBD | TBD | Open |
| Web | build | REVIEW-024 | `cd web && npm run build` | project root/web | PASS | TBD | TBD | Open |
| Legacy cleanup | old runtime scan | REVIEW-012 | `rg '/runtime-environments|/run-templates|RuntimeEnvironment|RunTemplate' docs scripts web internal configs deploy` | project root | No active old-model use remains | TBD | TBD | Open |
| OpenAPI | route diff | REVIEW-016 | `route list vs OpenAPI check` | project root | Current routes represented | TBD | TBD | Open |
| DB | fresh DB initialization | REVIEW-010 | `start server with /tmp/lightai-go-rc3-db` | disposable | Fresh current schema created | TBD | TBD | Open |
| Tenant | GPU direct ID isolation | REVIEW-002 | `cross-tenant API test` | test DB | Unauthorized cross-tenant access denied | TBD | TBD | Open |
| Audit | tenant scoping | REVIEW-009 | `multi-tenant audit test` | test DB | Audit scoped by resource tenant | TBD | TBD | Open |
| Runtime | RunPlan to Docker options | REVIEW-003 | `unit/integration conversion test` | test | Preview/spec/options equivalent | TBD | TBD | Open |
| Task | claim race | REVIEW-004 | `race/concurrency test` | test | No duplicate claim | TBD | TBD | Open |
| Task | duplicate result | REVIEW-004 | `duplicate/stale result test` | test | No state corruption | TBD | TBD | Open |
| Runtime | stop missing container | REVIEW-006 | `stop after docker rm` | disposable Docker | Success and lease release | TBD | TBD | Open |
| Runtime | agent restart reconciliation | REVIEW-005 | `restart Agent after container change` | disposable Docker | State converges | TBD | TBD | Open |
| NVIDIA E2E | model runtime | REVIEW-023 | `start/health/stop model` | disposable Docker | PASS | TBD | TBD | Open |
| MetaX | hardware verification | REVIEW-018 | `collector/runtime smoke if hardware accessible` | MetaX host | PASS or Blocked - External Hardware | TBD | TBD | Open |
| Observability | bundled mode | REVIEW-017 | `start-all with bundled mode` | disposable | Prom/Grafana healthy | TBD | TBD | Open |
| Observability | external mode | REVIEW-017 | `config external mode smoke` | disposable | Internal bundled skipped | TBD | TBD | Open |
| Observability | disabled mode | REVIEW-017 | `config disabled mode smoke` | disposable | Observability skipped cleanly | TBD | TBD | Open |
| Logging | server noise filter | REVIEW-030 | `10-minute run with scraping` | disposable logs | No repeated /metrics INFO noise | TBD | TBD | Open |
| Logging | agent noise filter | REVIEW-030 | `10-minute stable Agent run` | disposable logs | No repeated heartbeat/task_poll/gpu_metrics INFO noise | TBD | TBD | Open |
| Logging | error visibility | REVIEW-030 | `trigger representative failure` | disposable logs | WARN/ERROR visible | TBD | TBD | Open |
| Logging | debug mode | REVIEW-030 | `enable debug/full access log` | disposable logs | Detailed logs available | TBD | TBD | Open |
| Scripts | start-all dry-run | REVIEW-029 | `scripts/start-all.sh --dry-run` | source/release | PASS | TBD | TBD | Open |
| Scripts | start-all no observability dry-run | REVIEW-029 | `scripts/start-all.sh --dry-run --no-observability` | source/release | PASS | TBD | TBD | Open |
| Scripts | start-all wait | REVIEW-029 | `scripts/start-all.sh --wait` | source/release disposable | Health checks pass | TBD | TBD | Open |
| Scripts | stop-all after start-all | REVIEW-029 | `scripts/stop-all.sh` | source/release disposable | Processes stopped | TBD | TBD | Open |
| Release | package | REVIEW-023 | `project release package command` | project root | PASS | TBD | TBD | Open |
| Release | install smoke | REVIEW-023 | `unpack and run release` | /tmp/lightai-go-rc3-release | PASS | TBD | TBD | Open |
| Patch | apply | REVIEW-023 | `patch apply command` | /tmp/lightai-go-rc3-patch | PASS | TBD | TBD | Open |
| Patch | rollback | REVIEW-023 | `patch rollback command` | /tmp/lightai-go-rc3-patch | PASS | TBD | TBD | Open |
| Web | workflow checklist | REVIEW-028 | `complete checklist` | web/e2e | No Not Verified | TBD | TBD | Open |
| Web | raw i18n scan | REVIEW-026 | `locale/raw-key tests + scan` | web | PASS | TBD | TBD | Open |
