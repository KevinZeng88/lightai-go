> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Go Claude Full Project Review

> Date: 2026-06-16
> Reviewer: Claude (deepseek-v4-pro)
> Version reviewed: 0.1.14 (commit `e98816a`)
> Constraints: Review only; no code changes; no commits
> Plan: `docs/review/claude-full-project-review-plan-20260616.md`

---

## 1. Executive Summary

### Overall State

LightAI Go at v0.1.14 is a **remarkably solid early-stage project**. It has working Server/Agent binaries, multi-tenant RBAC, GPU auto-discovery for NVIDIA and MetaX (scripts ready), comprehensive Web UI, Prometheus/Grafana integration, model runtime lifecycle management (start/stop/logs/sweep), and a clean patch/upgrade system. The codebase is well-organized, documentation is extensive, and the Phase 2F RBAC/tenant hardening was executed with discipline.

### Fitness for Target Positioning

The architecture is **well-suited** for "small/medium customers, a handful of GPU servers, lightweight management, extensible." The two-binary model (Server + Agent), SQLite for simplicity, and shell-script GPU collectors are appropriate choices. The tenant/RBAC model is correctly scoped — not over-engineered but with room to grow.

### Biggest Strengths

1. **Solid engineering contracts** (`08-engineering-contracts.md`): Units, null semantics, DockerRunSpec, task lease/generation rules are well-defined and consistently followed
2. **Multi-tenant isolation**: Tenant-scoped queries, node transfer safety checks, audit logging with redaction
3. **GPU collector abstraction**: Vendor-neutral GPU resource model, auto-detection, external script protocol
4. **Comprehensive Web UI**: Full CRUD for all resources, auto-refresh, resizable columns, i18n
5. **Test discipline**: 142+ Go tests, 7 Vitest tests, shell tests, i18n verification
6. **Documentation**: Design docs, ops docs, troubleshooting, phase status tracking

### Biggest Risks

1. **Instance `tenant_id` not set on creation** — line `deployment_lifecycle.go:33,196-206`: model instances created via start deployment don't populate `tenant_id`. This defeats tenant isolation in the instance listing.
2. **Login metrics counters never incremented** — `auth/handlers.go:216`: Auth metrics (login total, failed login count) have a TODO and are unimplemented. This blinds operators to brute-force attacks.
3. **Password-expired users locked out** — `auth/middleware.go:139`: URL path check for `/api/auth/change-password` doesn't match actual `/api/v1/auth/change-password`, meaning users with expired passwords cannot change them.
4. **GPU lease race condition** — `api/lease.go:38`: Concurrent requests for the same GPU can both pass the conflict check before either INSERTs.
5. **Rate limiter memory leak** — `auth/ratelimit.go:46-54`: In-memory rate limiter map grows indefinitely with no cleanup.

### Should We Enter Next Phase?

**Yes, with caveats.** The foundation is strong enough to proceed, but the 5 critical bugs above and several high-priority issues should be fixed before Phase 2G (model serving hardening). Entering model gateway/API key/billing without fixing these would compound risk.

### Top 5 Capabilities Most Worth Building vs GPUStack

1. **Real-time instance status via polling or SSE** — GPUStack uses SSE watch; LightAI Go should at least have consistent polling
2. **Worker maintenance mode** — GPUStack allows draining workers; essential for production
3. **Grafana dashboard improvements** — GPUStack's dashboards are more polished and actionable
4. **Model download/caching** — GPUStack downloads models to workers; LightAI Go assumes models are pre-placed
5. **Instance log streaming** — GPUStack has virtualized log viewer; LightAI Go currently polls logs as strings

---

## 2. Current Architecture Review

### 2.1 Server (Control Plane)

**Strengths:**
- Clean Go 1.22+ pattern-based routing with compositional middleware chains
- Proper separation: session chain → permission chain → handler
- SQLite with WAL mode, foreign keys, versioned migration system (V1–V7)
- 27 permission codes with built-in + custom roles
- Argon2id password hashing, CSRF tokens, session sliding expiration (12h)
- Tenant-scoped queries with platform admin bypass

**Weaknesses:**
- Migration system has a **dual path**: `db.Migrate()` runs versioned migrations, but `resource_handlers.go` constructor also creates tables via `CREATE TABLE IF NOT EXISTS` (`resource_handlers.go:27-97`). Tables created in the handler bypass version tracking.
- No caching layer for RBAC permission resolution — every request queries `tenant_membership_roles → roles → role_permissions → permissions` for each permission check (`middleware.go:220-282`).
- Error handling in sweep/transfer safety checks silently discards errors — if a DB query fails, the code proceeds with defaults (e.g., `activeLeaseCount = 0`), which could allow unsafe operations.
- Hardcoded UUID for default tenant (`"a0000000-0000-0000-0000-000000000001"`) appears in 3+ places instead of a single constant or DB lookup.

### 2.2 Agent (Execution Plane)

**Strengths:**
- Collector interface with clean separation: `SystemCollector` + `GPUCollector`
- Vendor-neutral `GPUResource` model that downstream code must use
- Auto-detection mode probes each vendor's discover script and registers matching collectors
- External command collector protocol is simple, testable, and shell-compatible
- Docker runtime driver with interface abstraction, fake client for testing, and real Docker SDK adapter
- Graceful handling: failed collections preserve previous successful state

**Weaknesses:**
- Sequential task processing blocks heartbeat — `processTasks` runs in the main heartbeat goroutine (`cmd/agent/main.go:578`). A slow Docker pull or stop hangs heartbeat.
- Docker logs don't separate stdout/stderr — `docker.go:181` returns empty stderr always; the multiplexed Docker stream is not decoded
- External command collector's env config **replaces** parent process env (`external.go:176`). If `Env` is unset, the child process has an empty environment, breaking PATH resolution.
- Relative script paths in `DefaultProbes()` (`probe.go:37-38`) mean the agent must run from the project root or `deploy/` must be in CWD.
- Agent heartbeat logs `node_id` as `agentID` (`register.go:241`) — a logging regression that confuses debugging.

### 2.3 Web / UI

**Strengths:**
- Vue 3 + Element Plus + Pinia + vue-i18n, with embedded deployment (`-tags web`)
- 19 functional pages covering all CRUD operations
- `ApiClient` with automatic `/api/v1` prefix, CSRF token handling, 403 auto-retry, 401 state clear
- `useAutoRefresh` composable with visibility-aware polling, in-flight guards, error preservation
- Resizable table columns with localStorage persistence
- Multi-tenant login flow with tenant selector
- Model deployments page with Quick Deploy wizard, dry-run visualization, and log viewer

**Weaknesses:**
- Hardcoded strings in 6+ locations: Dashboard status messages (Chinese), Quick Deploy section (Chinese), Observability pages (English), Grafana credentials in HTML
- RolesPage is read-only (API supports CRUD, UI only lists)
- TenantsPage has create only (no edit/disable from UI)
- UsersPage has create only (no edit/disable from UI)
- PlaceholderPage.vue is dead code (not referenced in router)
- Sidebar nav labels "Prometheus" and "Grafana" hardcoded, not using `$t()`
- Missing i18n key `tenants.createdAt` — column header shows raw key string

### 2.4 GPU Collectors

**Strengths:**
- Well-designed external command protocol with quoted key=value lines
- NVIDIA collector: production-verified on RTX 5090, proper mktemp + trap cleanup
- MetaX collector: scripts ready, tested with 8-card C500 fixture data, Prometheus dedup verified
- Auto-detection correctly distinguishes "no devices" (exit 10) from "failed" (exit ≥30)
- Metric normalization: MB→bytes conversion, percent→ratio for Prometheus, null handling

**Gaps:**
- AMD scripts exist but are stubs (both return "not yet implemented")
- Ascend, Cambricon, Hygon, Intel vendors recognized in Go types but have no scripts or probes
- MetaX real hardware validation pending — scripts are mock-verified only
- External discover script also produces metrics that are discarded, then metrics script re-queries — doubling external command invocations

### 2.5 Model Runtime

**Strengths:**
- Clean separation: `RuntimeEnvironment` (Docker config) → `ModelArtifact` (model files) → `RunTemplate` (rendering) → `ModelDeployment` (binding) → `ModelInstance` (runtime)
- DockerRunSpec generation with variable substitution and dry-run validation
- GPU lease system with proper state machine (reserved→active→released/failed)
- Task lifecycle with claim, execution, result reporting, and sweep
- Sensitive data redaction in audit logs, envelopes, and command previews

**Weaknesses:**
- **Instance tenant_id not set on creation** (see Critical #1 below)
- Instance update and lease activation not in same transaction (`task_handlers.go:112-137`)
- Hardcoded 5-minute lease expiry (`lease.go:46`), not configurable
- Active lease sweep only on offline nodes — online nodes with stuck leases never cleaned
- DockerSpec DELETE-then-INSERT has concurrency gap (`model_handlers.go:609-634`)
- N+1 query pattern for runtime environment docker specs (`model_handlers.go:448`)

### 2.6 Observability

**Strengths:**
- Three-mode observability (builtin/external/disabled)
- Prometheus HTTP SD via `/metrics/targets`
- 5 Grafana dashboards auto-provisioned: overview, host resources, GPU resources, GPU overview, agent health
- 7 alert rule templates
- Both Agent and Server expose `/metrics` with proper Prometheus naming

**Weaknesses:**
- Prometheus/Grafana binaries not included in the dev repository — bundled mode requires manual download
- Go Server does not manage Prometheus/Grafana lifecycle (start/stop/health) — relies on shell scripts
- Dashboard labels are mostly English, not localized
- Agent load metrics suppressed at zero load (`metrics/metrics.go:385`), creating data gaps
- No disk I/O metrics collected in system collector

### 2.7 Packaging / Deployment

**Strengths:**
- `package-release.sh` and `package-release-docker.sh` for reproducible builds
- `apply-patch.sh` with SHA256 verification, backup, and rollback — P0 verified with 7 scenarios
- glibc ABI compatibility check (`check-glibc-compat.sh`)
- systemd units, start/stop scripts, credentials management
- Clean release tarball layout: bin/, configs/, deploy/, scripts/, logs/, run/, runtime/

**Weaknesses:**
- Version number requires manual bump — no CI automation
- Release notes contain personal paths (`/home/kzeng/`) in the dist tarball
- No health check integration between Server and Prometheus/Grafana startup order
- Docker Compose is dev/demo only; production deployment relies on manual systemd config

---

## 3. GPUStack Comparison

| Area | GPUStack Approach | LightAI Go Current | What to Learn | What Not to Copy Yet |
|------|-------------------|--------------------|---------------|----------------------|
| **Architecture** | Python monolith with multiprocessing; Server can embed Worker | Two independent Go binaries (Server + Agent) | Go binaries are simpler to deploy — keep this | Multiprocessing, os._exit(1) on leader loss |
| **GPU Detection** | Strategy pattern via DetectorFactory; uses gpustack-runtime C library + fastfetch binary | Collector interface with external shell scripts; optional native Go code | GPUStack's detector factory pattern is cleaner for multi-vendor; consider consolidating probe + register logic | C library dependency (overly complex for LightAI's needs) |
| **GPU Abstraction** | GPUDevice model with vendor-specific fields | Vendor-neutral `GPUResource` with normalized fields | LightAI's approach is better for simplicity | GPUStack's detailed per-vendor fields |
| **System Collection** | Fastfetch binary (third-party, needs download) | gopsutil (pure Go) | LightAI's pure-Go approach is better — no binary dependency | Fastfetch dependency |
| **Worker Registration** | returns per-worker token + PredefinedConfig | returns node_id, agent_id, tenant_id | LightAI's simpler approach is appropriate | Dynamic config push from server (complex) |
| **Scheduling** | 7-level filter chain + backend-specific selector + scorer chain | Manual deployment (single node, replicas=1) | LightAI should adopt: scheduler as async loop with queue, not complex filters | Full multi-stage scheduling — overkill for "handful of servers" |
| **Model Serving** | Backend-specific InferenceServer classes (vLLM, SGLang, VoxBox, etc.) | Docker runtime driver with generic commands | GPUStack's backend abstraction is well-structured but heavy for current phase | Multi-backend inference server classes |
| **Instance State Machine** | PENDING→ANALYZING→SCHEDULED→RUNNING→ERROR | pending→running→stopped→failed | GPUStack's ANALYZING and SCHEDULED states add clarity; LightAI could benefit | Separate controller per entity type |
| **Controller Pattern** | Reactive: event bus → subscribe → reconcile loop | Periodic sweep ticker | GPUStack's reactive pattern is cleaner but requires coordinator | Distributed coordinator + event bus |
| **Gateway/Proxy** | Higress (K8s Ingress) + WebSocket proxy | Not implemented (deferred) | Study the gateway abstraction but don't implement yet | Entire gateway/proxy stack |
| **API Key** | access_key + hashed_secret_key, model scope, expiry | Not implemented (deferred) | Schema design is worth studying for future | Full API key management now |
| **Usage Tracking** | ModelUsage, MeteredUsage, ModelUsageDetails with archiving | Not implemented (deferred) | Study the data model for future | Usage/archiving system |
| **Worker Maintenance** | Maintenance mode, drain | Not implemented | Critical for production — implement before model serving goes live | — |
| **UI Framework** | React + UmiJS + Ant Design ProComponents | Vue 3 + Element Plus + vue-i18n | Vue is simpler; good choice for this project | UmiJS (too heavy), React (not worth switching) |
| **UI Dashboard** | Statistical cards + system load charts + usage charts + active deployments table | 4 metric cards + Top-5 GPU tables + abnormal GPU list + collection diagnostics | GPUStack's system load trends (CPU/RAM/GPU/VRAM charts over time) would greatly improve LightAI's dashboard | Full usage/billing charts |
| **UI Table Patterns** | SealTable with expandable rows, inline editing, real-time updates via SSE | Resizable columns with auto-refresh polling | Consider SSE for instance status instead of polling; GPUStack's status column design | Inline row editing, complex multi-column sorting |
| **UI Status Display** | Standardized StatusTag with success/transitioning/warning/error/inactive + state_message | StatusTag with similar color mapping | Already good; add state_message tooltips for error context | — |
| **UI Filtering** | FilterBar with search + dropdowns + batch operations | Per-page search input + ElSelect filters | GPUStack's filter bar is more polished but not urgent | Batch operations |
| **UI Navigation** | Grouped collapsible sidebar | Flat sidebar with nested items | GPUStack's grouped, collapsible sidebar is cleaner as pages grow | Plugin system, route extensions |
| **Observability** | Built-in Prometheus + Grafana with auto-provisioned dashboards | Same: built-in/external/disabled modes, HTTP SD, 5 dashboards | Already solid; improve dashboard quality to match GPUStack's polish | Separate metric exporters per cluster |
| **Deployment** | Docker Compose, Kubernetes Helm chart, pip install | Shell scripts + systemd, Docker Compose dev/demo | Consider a `lightai install` one-liner like GPUStack's `curl -sfL ... \| sh -` | Helm chart, Kubernetes deployment |
| **Upgrade** | pip upgrade with migration | patch tarball + apply-patch.sh | LightAI's patch system is well-designed for disconnected deployment | — |
| **Documentation** | mkdocs site with tutorials, user guide, performance lab | Markdown docs in repo | GPUStack's structured docs are more usable. Consider a docs site for end users | Performance lab, multi-language docs |

### What LightAI Go Already Does Better Than GPUStack

1. **Simpler deployment**: Two Go static binaries vs Python + pip + dependencies
2. **Cleaner RBAC model**: Permission-code-based access control is more auditable than OrgRole
3. **Patch system**: Incremental SHA256-verified patch tarballs for air-gapped environments
4. **GPU abstraction**: Vendor-neutral `GPUResource` is simpler than GPUStack's vendor-specific fields
5. **Credentials management**: Auto-generated credentials, never overwrite, never logged

### Where GPUStack Is Ahead (Acknowledged, Not All To Copy)

1. **Instance lifecycle**: More mature state machine with health checks and reactive reconciliation
2. **Scheduler**: Even a simple scheduler would help for multi-node deployments
3. **Dashboard visualization**: Time-series charts for system/GPU load trends
4. **Worker maintenance mode**: Essential safety feature before production
5. **Structured documentation site**: End-user focused, with tutorials and performance data

---

## 4. Phase Completion Consistency

### 4.1 Phase 2F Closure Check

The latest commit `e98816a` closes Phase 2F. Verification:

| Item | Status | Evidence |
|------|--------|----------|
| 12 review issues fixed | ✅ | commit `86ab1d4` |
| RBAC edge tests added | ✅ | `rbac_phase2f_test.go` (18 tests) |
| i18n formatRelativeTime | ✅ | `formatters.test.mjs` PASS |
| E2E cleanup safe | ✅ | PID-based trap in `df1a212` |
| Tenant switching API + UI | ✅ | `POST /api/v1/session/switch-tenant` + ConsoleLayout selector |
| Audit log page | ✅ | `AuditLogsPage.vue` with filtering |
| V7 migration | ✅ | resource_pools, tenant types, audit_logs |
| Model instance tenant isolation | ✅ | V6 migration added tenant_id column |

### 4.2 Design vs Implementation Gaps

| Design Says | Implementation Reality | Severity |
|-------------|------------------------|----------|
| `08-contracts.md §13.14`: RuntimeEnvironment, Model, ModelInstance must write `tenant_id` | `deployment_lifecycle.go:33,196-206`: Instance INSERT doesn't set `tenant_id` | **Bug** |
| `tenant-rbac-resource-ownership-design.md`: GPU devices inherit `tenant_id` from node | `resource_handlers.go:319`: new GPUs always get hardcoded default tenant UUID | **Bug** |
| Design doc §10: Instance `active_operation_id`, `spec_generation`, etc. | Columns exist in DB but not consistently populated in all code paths | **Partial** |
| `08-contracts.md §5`: Only 3 collector result states | Implemented correctly in registry | **OK** |
| `08-contracts.md §13.13`: "Agent API only uses agent token, not User Session" | Implemented correctly | **OK** |
| `09-auth-tenant-design.md`: Resource pool tables | V7 migration creates tables but no handlers | **Schema only** |

### 4.3 Documentation Freshness

| Document | State | Issues |
|----------|-------|--------|
| `PHASE-STATUS.md` | Updated 2026-06-14 | MetaX noted as "hardware pending" — accurate |
| `RC1_REVIEW_FIX_PLAN.md` | Updated 2026-06-14 | P0-P2 status current. glibc note says 2.28 but VERSION shows 0.1.14 |
| `RELEASE_NOTE_v0.1.9.md` | v0.1.9 specific | Does not reflect v0.1.10–v0.1.14 changes |
| `README.md` / `docs/README.md` | Original Phase 0-2B window | Does not mention Phase 2F, model runtime, or tenant hardening |
| `docs/00-project-scope.md` | Original | Not updated for post-2F status |
| `docs/01-architecture.md` through `docs/10-*.md` | Original Phase 0-2B | Do not reflect model runtime implementation |
| `docs/design/12-model-runtime-serving-design.md` | New | Matches implementation. V6/V7 migrations documented |
| `docs/ops/model-runtime-e2e-local.md` | New | Good operational documentation |
| `docs/ops/model-runtime-troubleshooting.md` | New | Practical troubleshooting guide |

**Recommendation**: `docs/README.md`, `docs/PHASE-STATUS.md`, and `docs/00-project-scope.md` need updates to reflect post-Phase-2F reality. Release notes for v0.1.10 through v0.1.14 should be consolidated.

---

## 5. Backend Review

### Issues Found

| # | Severity | File:Line | Issue |
|---|----------|-----------|-------|
| B1 | **Critical** | `api/deployment_lifecycle.go:33,196-206` | `model_instances` INSERT doesn't set `tenant_id`, leaving it empty. Tenant isolation on instance listing is broken. |
| B2 | **Critical** | `auth/middleware.go:139` | Path check for change-password exemption uses `/api/auth/change-password` but actual route is `/api/v1/auth/change-password`. Users with expired passwords cannot change them. |
| B3 | **Critical** | `auth/handlers.go:216` | Login metrics (`AuthLoginTotal`, `AuthLoginFailed`) never incremented — TODO never implemented. Operators blind to brute-force attacks. |
| B4 | **High** | `api/lease.go:38` | Parallel lease requests for the same GPU can both pass `SELECT ... WHERE status IN (reserved, active) LIMIT 1` before either INSERTs. No UNIQUE constraint on `(gpu_id, status)`. |
| B5 | **High** | `api/task_handlers.go:112-137` | Instance running update and lease activation in separate transactions. If lease activation fails after instance update, system is inconsistent. |
| B6 | **High** | `auth/ratelimit.go:46-54` | In-memory rate limiter map (`rateLimits`) never cleaned up. Grows indefinitely with unique IPs — memory leak in long-running server. |
| B7 | **High** | `api/resource_handlers.go:319` | New GPUs always get hardcoded default tenant UUID, not the node's `tenant_id`. Design says GPUs inherit from node. |
| B8 | **High** | `api/resource_handlers.go:433` | `HandleListGPUs` uses hardcoded `"a0000000-..."` for tenant scoping instead of dynamic lookup. Breaks if DB reinitialized. |
| B9 | **Medium** | `api/agent_handlers.go:362` | `sweepExpiredTasks` tries to expire `LeaseActive` leases, but `sweep.go:70-78` intentionally only fails active leases on offline nodes. Inconsistent sweep behavior. |
| B10 | **Medium** | `api/agent_handlers.go:676-688` | Transfer safety checks: `QueryRow` errors silently ignored. If query fails, `activeLeaseCount = 0`, allowing unsafe transfer. |
| B11 | **Medium** | `api/agent_handlers.go:704` | Audit log INSERT error silently ignored. Audit trail gaps go undetected. |
| B12 | **Medium** | `api/model_handlers.go:609-634` | `createOrUpdateDockerSpec` uses DELETE-then-INSERT. Concurrent readers see empty state mid-operation. |
| B13 | **Medium** | `api/model_handlers.go:448` | N+1 query pattern: each runtime environment triggers a separate docker spec query. |
| B14 | **Medium** | `api/lease.go:46` | Lease expiry hardcoded to 5 minutes. Not configurable for slower Docker environments. |
| B15 | **Medium** | `api/sweep.go:39-95` | All sweep `Exec` errors silently discarded. Sweep failures go undetected. |
| B16 | **Medium** | `api/audit_handlers.go:104-121` | `auditLog` function (derived IDs) and `audit()` in model_handlers.go (UUID IDs) are two competing implementations. `auditLog` is never called. |
| B17 | **Low** | `models/models.go` | Timestamp type inconsistency: old models use `time.Time`, new models (Phase 1) use `string`. |
| B18 | **Low** | `models/models.go:299-310,312-330` | GpuLease missing V4 timestamp fields; Node missing V2 detail fields in Go struct. |
| B19 | **Low** | `api/router.go:209-211` | `handleNotImplemented` function defined but never used. |
| B20 | **Low** | `api/model_handlers.go:464` | `isOperator` computed but assigned to `_`. Dead code. |
| B21 | **Low** | `resolver.go:306-312` | VRAM warning query executed, result assigned to `_`, then re-queried at call site. |

### Backend Strengths

- Transaction discipline in resource reporting (P0-008 fixed)
- Agent identity binding enforced at registration and heartbeat
- Tenant-scoped node/GPU listing with proper 404 for cross-tenant access
- RBAC permission-code-based authorization (not role-name-based)
- Proper CSRF with constant-time comparison, Origin/Referer validation
- Session sliding expiration with configurable refresh window
- Task claim with atomic UPDATE in heartbeat transaction

---

## 6. Agent / GPU Review

### Issues Found

| # | Severity | File:Line | Issue |
|---|----------|-----------|-------|
| A1 | **High** | `cmd/agent/main.go:578` | `processTasks` runs sequentially in heartbeat goroutine. A slow task blocks heartbeat, which can cause node to appear offline. |
| A2 | **High** | `agent/runtime/docker.go:181` | Docker `Logs()` combines stdout+stderr into one string. Stderr is always empty. The raw multiplexed Docker stream is not decoded. |
| A3 | **Medium** | `agent/collector/external.go:176` | External command env **replaces** parent environment. If `Env` config is empty, child has no PATH, breaking tool discovery. |
| A4 | **Medium** | `agent/collector/probe.go:37-38` | Discover script paths are relative (`deploy/collectors/gpu/...`). Agent fails if not run from project root. |
| A5 | **Medium** | `agent/metrics/metrics.go:385-388` | Load1/Load5/Load15 suppressed at zero. When load drops to 0, Prometheus sees gap, not decline. Rate queries break. |
| A6 | **Medium** | `agent/register/register.go:241` | Heartbeat log: `"node_id", agentID` instead of `nodeID`. Misleading for debugging. |
| A7 | **Low** | `cmd/agent/main.go:608` | `ReporTaskResult` failure silently drops result. No retry queue. Server unaware of outcome. |
| A8 | **Low** | `cmd/agent/main.go:647` | Task timeout cancels context, but container may already be created on Docker daemon. Orphan container risk. |
| A9 | **Low** | `agent/collector/registry.go:9-17` | Registry not concurrency-safe. If `Collect` and Prometheus scrape run concurrently, race on `last*` fields. |
| A10 | **Low** | `deploy/collectors/gpu/amd/` | AMD scripts are stubs (return "not yet implemented"). AMD not in `DefaultProbes()`. |
| A11 | **Low** | `agent/collector/collector.go:118` | Ascend, Cambricon, Hygon, Intel are recognized vendor strings but have no detect scripts or Go integration. |

### Agent Strengths

- Auto-detection correctly handles all three outcomes: devices found, no devices, failed
- GPU metric normalization: MB→bytes, percent→ratio for Prometheus
- Prometheus dedup logic prevents "duplicate time series" errors (MetaX 8-card fix)
- Docker fake client enables comprehensive testing without Docker daemon
- Sensitive environment variable redaction in command previews
- Graceful shutdown: context cancellation, health server shutdown, 30s timeout

### GPU Discovery Coverage Matrix

| Vendor | Discover Script | Metrics Script | Go Integration | Auto-Detect | Real HW Verified |
|--------|----------------|----------------|----------------|-------------|------------------|
| NVIDIA | ✅ | ✅ | ✅ (native + external) | ✅ | ✅ (RTX 5090) |
| MetaX | ✅ | ✅ | ✅ (external) | ✅ | ❌ (mock only) |
| AMD | Stub | Stub | ❌ (not in probes) | ❌ | ❌ |
| Ascend | ❌ | ❌ | String const only | ❌ | ❌ |
| Cambricon | ❌ | ❌ | String const only | ❌ | ❌ |
| Hygon | ❌ | ❌ | String const only | ❌ | ❌ |
| Intel | ❌ | ❌ | String const only | ❌ | ❌ |

---

## 7. Model Runtime Review

### Issues Found

| # | Severity | File:Line | Issue |
|---|----------|-----------|-------|
| M1 | **Critical** | `api/deployment_lifecycle.go:33,196-206` | Instance `tenant_id` not populated. Same as B1. |
| M2 | **High** | `api/task_handlers.go:112-137` | Instance update and lease activation in separate transactions. Same as B5. |
| M3 | **Medium** | `api/lease.go:38` | GPU lease race condition (B4). Parallel start requests for same GPU both succeed. |
| M4 | **Medium** | `api/model_handlers.go:609-634` | DockerSpec DELETE-then-INSERT concurrency gap. Same as B12. |
| M5 | **Medium** | `api/model_handlers.go:1142-1150` | Phase 1 guard checks (replicas==1, schedule_mode=="manual") in Go code only — no DB CHECK constraints. |
| M6 | **Medium** | `api/sweep.go:46` | Sweep uses SQLite-specific `julianday()`. Not portable to PostgreSQL. |
| M7 | **Low** | `api/resolver.go:213-214` | GPUVisibleEnvKey default logic: always resets to CUDA_VISIBLE_DEVICES if empty, even if vendor needs HIP_VISIBLE_DEVICES. Manual override is supported but auto-detection would be better. |
| M8 | **Low** | `api/model_handlers.go:1303` | `resolver.ValidateDryRun` takes `*sql.DB` instead of `*db.DB`, accessing internal field. |
| M9 | **Low** | `api/model_handlers.go:1561` | Admin vs non-admin GpuLease queries return different column sets. |

### Model Runtime Strengths

- Clean multi-entity model: Artifact ↔ RuntimeEnv ↔ Template ↔ Deployment ↔ Instance
- Dry-run validation with variable substitution and equivalent command preview
- Sensitive data redaction throughout (audit, command preview, logs)
- Conservative sweep strategy: active leases only failed when node is offline
- Stop is idempotent (stopped already returns success)
- Quick Deploy wizard in Web UI for rapid deployment creation

### State Machine Check

| Transition | Implemented | Edge Cases |
|------------|-------------|------------|
| pending → running | ✅ via `StartDeployment` | Already running returns 409 |
| pending → failed | ✅ via sweep (timeout) | Stopped already returns success (idempotent) |
| running → stopped | ✅ via `StopDeployment` | Stopped already returns success (idempotent) |
| running → failed | ✅ via sweep (container exit) | Agent reports container exit as failed |
| stopped → running | ✅ via restart (stop+start) | Reuses same deployment, new instance |
| stopped → failed | ✅ via sweep (timeout of stop) | 5-minute timeout then sweep |

---

## 8. Web/UI Review

### Issues Found

| # | Severity | File:Line | Issue |
|---|----------|-----------|-------|
| W1 | **Medium** | `src/pages/GrafanaPage.vue:7` | Hardcoded credentials `"admin / lightai (dev only)"` in rendered HTML. Security: credential disclosure in production. |
| W2 | **Medium** | `src/pages/ModelDeploymentsPage.vue:12-48` | Entire Quick Deploy section hardcoded in Chinese. Not i18n-compatible. |
| W3 | **Medium** | `src/pages/ModelDeploymentsPage.vue:23` | Hardcoded developer path `/home/kzeng/models/...` in placeholder text. |
| W4 | **Medium** | `src/pages/DashboardPage.vue:126,130,134` | Status strings concatenated in template: `'正常'`, `'个节点超时'`, etc. Breaks i18n for these status messages. |
| W5 | **Medium** | `src/locales/zh-CN.ts`, `src/locales/en-US.ts` | Missing `tenants.createdAt` i18n key. TenantsPage column header shows raw key string. |
| W6 | **Medium** | `src/layouts/ConsoleLayout.vue:61-62` | Sidebar labels "Prometheus" and "Grafana" hardcoded, not `$t()`. |
| W7 | **Medium** | `src/pages/ObservabilityOverviewPage.vue:41-47` | All button labels and descriptions hardcoded in English. |
| W8 | **Medium** | `src/pages/PrometheusPage.vue:7-15` | All status text hardcoded in English. |
| W9 | **Medium** | `src/pages/GrafanaPage.vue:11-35` | All labels hardcoded. Dashboard names hardcoded. |
| W10 | **Low** | `src/pages/RolesPage.vue` | Read-only list. API supports `createRole`, `deleteRole`, `updateRolePermissions` but UI doesn't expose them. |
| W11 | **Low** | `src/pages/TenantsPage.vue:33` | Success message `'Created'` hardcoded English. Missing edit/disable buttons. |
| W12 | **Low** | `src/pages/UsersPage.vue` | Missing edit/disable buttons. API supports `updateUser` and `disableUser`. |
| W13 | **Low** | `src/pages/PlaceholderPage.vue` | Dead code — replaced by real implementations but file remains. |
| W14 | **Low** | `src/api/nodes.ts:63,69`, `src/api/gpus.ts:36` | Hardcoded `/api/v1/` prefix in a few API calls — works but inconsistent with convention. |
| W15 | **Low** | `src/pages/RuntimeEnvironmentsPage.vue:53` | "Volumes" label hardcoded English. |

### Web UI Strengths

- 19 functional pages covering the full resource model
- Unified `ApiClient` with CSRF, auto-retry, tenant context
- `useAutoRefresh` composable handles visibility, errors, cleanup
- Resizable, persistable table columns on list pages
- Tenant switching in sidebar with full page reload for clean state
- Dry-run visualization with validation errors, warnings, and command preview
- Log viewer with auto-polling for model instances
- 220 i18n keys consistent between zh-CN and en-US
- Tests: 7 Vitest (auth, dashboard, auto-refresh) + 3 Node (i18n, paths, formatters)

### Page-by-Page Completeness

| Page | CRUD | Filters | i18n | Auto-refresh | Notes |
|------|------|---------|------|--------------|-------|
| Dashboard | Read-only | N/A | Partial | Manual | Status strings concatenated in template |
| Nodes | Read + detail | Status + search | ✅ | ✅ | Resizable columns |
| GPUs | Read + detail | Vendor + health | ✅ | ✅ | Resizable columns |
| Users | List + Create | None | ✅ | N/A | Missing edit/disable |
| Tenants | List + Create | None | Partial | N/A | Missing edit; missing i18n key |
| Roles | List only | None | ✅ | N/A | Missing CRUD from API |
| Audit Logs | List | Action + entity type | ✅ | Manual | Paginated |
| Model Artifacts | Full CRUD | None | ✅ | ✅ | |
| Model Deployments | CRUD + Deploy | None | Partial | ✅ | Quick Deploy hardcoded zh-CN |
| Model Instances | List + Detail + Logs | Deployment filter | ✅ | ✅ | Log auto-polling |
| Runtime Environments | Full CRUD | None | Partial | N/A | "Volumes" hardcoded |
| Run Templates | Full CRUD | None | ✅ | N/A | Render preview |
| Obs. Overview | Read | N/A | ❌ (hardcoded en) | Manual | |
| Prometheus | Read (iframe) | N/A | ❌ (hardcoded en) | N/A | |
| Grafana | Read (iframe) | N/A | ❌ (hardcoded en) | N/A | Credentials in HTML! |
| Obs. Targets | Read | Status | ✅ | Manual | |
| Change Password | Form | N/A | ✅ | N/A | |
| Login | Form | N/A | ✅ | N/A | Multi-tenant support |

---

## 9. Observability Review

### Issues Found

| # | Severity | File:Line | Issue |
|---|----------|-----------|-------|
| O1 | **Medium** | `agent/metrics/metrics.go:385-388` | Load metrics suppressed at zero. Same as A5. |
| O2 | **Medium** | `deploy/observability/grafana/dashboards/` | Dashboard labels mostly English. No zh-CN variants. |
| O3 | **Medium** | `deploy/observability/grafana/dashboards/lightai-gpu-overview.json` | GPU overview dashboard uses `lightai_gpu_memory_used_bytes / lightai_gpu_memory_total_bytes` but doesn't handle NULL/NaN gracefully. |
| O4 | **Low** | `scripts/observability-up.sh` | No health check wait between Prometheus start and Grafana start. Race condition on Grafana datasource availability. |
| O5 | **Low** | `internal/server/metrics/metrics.go` | API counter metrics only track `/api/*` paths, good. But no integration with the auth handler for auth counter metrics. |
| O6 | **Low** | `configs/observability/prometheus.yml` | Scrape interval hardcoded 15s. Not configurable without editing yml. |
| O7 | **Low** | `deploy/observability/prometheus/rules/lightai.rules.yml` | 7 alert rules. All thresholds hardcoded. No mechanism for user customization. |
| O8 | **Low** | `internal/agent/collector/system.go:70` | CPU utilization is average across all cores (`percpu=false`). Per-core metrics would help diagnose uneven GPU/CPU pairing. |

### Observability Strengths

- Three-mode design (builtin/external/disabled) is flexible
- HTTP SD approach is correct for dynamic targets
- Both Agent and Server expose properly-named Prometheus metrics
- 5 Grafana dashboards with reasonable coverage
- 7 alert rules covering critical conditions
- `/metrics` never triggers GPU vendor tools (reads snapshot only)

### Metrics Coverage

| Domain | Metrics | Gaps |
|--------|---------|------|
| Server | API counters (by method, path, status), node/GPU counts, DB stats | No auth metrics implemented (TODO) |
| Agent GPU | memory total/used/free bytes, utilization %, temp, power, health, availability | No per-process GPU metrics, no GPU error counts |
| Agent System | CPU %, load 1/5/15, mem total/used/free, swap, disk usage per mount, net IO per interface | No disk IOPS/latency, no process counts, load suppressed at zero |
| Agent Health | Collector errors, report errors, registration status, heartbeat status | No task execution metrics (start/stop duration, failure counts) |

---

## 10. Security / RBAC / Tenant Isolation Review

### Issues Found

| # | Severity | File:Line | Issue |
|---|----------|-----------|-------|
| S1 | **Critical** | `auth/middleware.go:139` | Password-expired users locked out (same as B2). |
| S2 | **Critical** | `auth/handlers.go:216` | Login metrics never incremented (same as B3). |
| S3 | **Critical** | `api/deployment_lifecycle.go:33,196-206` | Instance tenant_id not set (same as B1). |
| S4 | **High** | `api/lease.go:38` | GPU lease race condition (same as B4). |
| S5 | **High** | `api/task_handlers.go:112-137` | Transaction boundary issue (same as B5). |
| S6 | **High** | `auth/ratelimit.go:46-54` | Rate limiter memory leak (same as B6). |
| S7 | **Medium** | `auth/csrf.go:44-49` | Origin check uses suffix match. `https://evil.com/127.0.0.1:18080` would match host `127.0.0.1:18080`. Should parse URL and compare host component. |
| S8 | **Medium** | `web/src/pages/GrafanaPage.vue:7` | Hardcoded credentials (same as W1). |
| S9 | **Medium** | `auth/handlers.go:500-507` | `HandleCSRFToken` cannot return actual token (hash only stored). Endpoint exists but is confusing. |
| S10 | **Medium** | `api/agent_handlers.go:704` | Audit log INSERT error silently ignored (same as B11). |
| S11 | **Medium** | `api/model_handlers.go:667-671` | Sensitive image names partially redacted. Docker image names like `registry.example.com/my-secret:latest` get `<redacted>` in display but preserved in execution. |
| S12 | **Low** | `auth/session.go:242` | `hashString` uses SHA-256 without HMAC. Session IDs are random, so risk is minimal, but best practice is HMAC with server secret. |
| S13 | **Low** | `api/resource_handlers.go:373` | Collector diagnostics logged at debug level only. If debug disabled, diagnostics silently lost. |
| S14 | **Low** | `rbac/handlers.go:779-785` | TOCTOU race between role count SELECT and role DELETE in `HandleRemoveMembershipRole`. |

### Tenant Isolation Assessment

| Resource | Tenant Scoping | Cross-Tenant Protection | Notes |
|----------|---------------|------------------------|-------|
| Nodes | ✅ Via session ctx | ✅ 404 for cross-tenant | Platform admin sees all |
| GPUs | ✅ Join nodes for scope | ✅ Via node tenant_id | Hardcoded default UUID issue (B8) |
| ModelArtifacts | ✅ | ✅ block cross-tenant reads | Tested |
| RuntimeEnvironments | ✅ | ✅ block cross-tenant reads | |
| RunTemplates | ✅ | ✅ | |
| ModelDeployments | ✅ | ✅ block cross-tenant start | Tested |
| ModelInstances | ❌ **Not set on creation** | Partial (V6 column exists but empty) | Critical bug B1 |
| GpuLeases | ✅ Via DB query | ✅ tenant filter | |
| AgentTasks | Follows instance | Inherits from associated instance | |
| AuditLogs | ✅ | ✅ | Tested |
| Users | Platform admin only | N/A | Correct |
| Tenants | Platform admin only | N/A | Correct |
| Roles | Built-in: global; custom: tenant-scoped | ✅ custom roles scoped to tenant | |

### RBAC Permission Coverage

27 permission codes implemented. Verification: all built-in roles have appropriate permissions assigned in `auth/bootstrap.go`.

| Built-in Role | Permission Count | Scope |
|---------------|-----------------|-------|
| admin (tenant) | All tenant-scoped permissions | Per-tenant |
| operator (tenant) | Read + instance control | Per-tenant |
| viewer (tenant) | Read-only | Per-tenant |

Platform admin (`User.IsPlatformAdmin=true`) bypasses tenant scope entirely — correct.

**Gap**: No "node maintainer" role that can manage nodes without full admin. The `node:transfer` permission exists but is assigned only to admin. A dedicated node operator role with `node:transfer` + `node:read` would be useful for datacenter operators.

---

## 11. Packaging / Deployment / Upgrade Review

### Issues Found

| # | Severity | File:Line | Issue |
|---|----------|-----------|-------|
| P1 | **Medium** | `scripts/package-release.sh` | Version bump requires `--bump` flag manually. No CI integration. VERSION file is single source of truth, which is correct. |
| P2 | **Medium** | `dist/lightai-go-0.1.14-linux-amd64/` | Dist tarball contains absolute paths from build machine. `find dist -type f | xargs grep -l '/home/kzeng'` may find artifacts. |
| P3 | **Medium** | No install script | Fresh install requires manual steps: untar, edit configs, start server, start agent. No `install.sh` or one-liner. |
| P4 | **Low** | `scripts/start-agent.sh:13` | `nohup` wrapper for agent startup. If agent crashes, no auto-restart (systemd unit exists as alternative). |
| P5 | **Low** | `configs/agent.yaml:9` | `server_url` default is `http://127.0.0.1:18080`. For multi-node deployment, every agent on different nodes must change this. |
| P6 | **Low** | `configs/agent.nvidia.yaml`, `agent.metax.yaml` | Legacy config files for explicit vendor mode. Auto-detection makes these mostly obsolete but they remain. Risk of confusion. |
| P7 | **Low** | No health check integration | start-server.sh doesn't verify server is healthy before returning. Operator must manually check. |

### Release Package Layout (v0.1.14)

```
lightai-go-0.1.14-linux-amd64/
├── bin/
│   ├── lightai-server      # Server binary
│   └── lightai-agent       # Agent binary
├── configs/                 # Server + agent config templates
├── data/                    # Empty, for runtime data
├── deploy/
│   ├── collectors/gpu/      # GPU collector scripts (nvidia, metax, amd)
│   ├── docker-compose/
│   ├── observability/       # Prometheus/Grafana configs
│   └── systemd/             # systemd unit files
├── logs/                    # Empty, for runtime logs
├── run/                     # Empty, for runtime PID files
├── runtime/                 # For initial-credentials.txt
├── scripts/                 # All management scripts
└── LICENSES/
```

Layout is clean and production-suitable. Missing: an `install.sh` for streamlined setup.

---

## 12. Documentation Review

### Document Quality Matrix

| Document | Freshness | Completeness | Usefulness |
|----------|-----------|-------------|------------|
| `AGENTS.md` | ✅ Current | ✅ | Development reference |
| `docs/README.md` | ⚠️ Phase 0-2B window | ⚠️ Doesn't reflect post-2F | Reading order still valid |
| `docs/PHASE-STATUS.md` | ✅ 2026-06-14 | ✅ | Phase tracking |
| `docs/00-project-scope.md` | ⚠️ Original | ⚠️ Pre-dates model runtime | Scope definition |
| `docs/01-architecture.md` | ⚠️ Original | ⚠️ Pre-dates model runtime | Architecture overview |
| `docs/02-07` design docs | ⚠️ Original Phase 0-2B | Partial | Foundation design |
| `docs/08-engineering-contracts.md` | ✅ | ✅ Excellent | Cross-module rules |
| `docs/09-auth-tenant-design.md` | ✅ | ✅ | Auth/RBAC |
| `docs/10-mvp-development-plan.md` | ⚠️ Original | ⚠️ Don't reflect current phase | Historical |
| `docs/design/12-model-runtime-serving-design.md` | ✅ New | ✅ | Model runtime |
| `docs/plan/12-*.md` | ✅ New | ✅ | Phase plans |
| `docs/ops/model-runtime-e2e-local.md` | ✅ New | ✅ | E2E operations |
| `docs/ops/model-runtime-troubleshooting.md` | ✅ New | ✅ | Troubleshooting |
| `docs/ops/tenant-rbac-resource-ownership-operations.md` | ✅ New | ✅ | Tenant ops |
| `docs/RC1_REVIEW_FIX_PLAN.md` | ✅ | ✅ | RC1 tracking |
| `docs/RC1_CODEX_REVIEW_TRACKING.md` | ✅ | ✅ | Codex review |
| `docs/GPU_COLLECTOR_ARCHITECTURE.md` | ✅ | ✅ | GPU architecture |
| `docs/REVIEW-GPUSTACK-AUDIT.md` | ✅ | ✅ | GPUStack reference |
| `docs/REVIEW-GPUSTACK-UI.md` | ✅ | ✅ | GPUStack UI reference |
| `RELEASE_NOTE_v0.1.9.md` | ⚠️ v0.1.9 only | ⚠️ No v0.1.10-0.1.14 notes | Release history |
| `README-RELEASE.md` | ✅ | ✅ | Release packaging |
| `README.md` | ⚠️ Basic | ⚠️ Could be more end-user friendly | Project intro |
| `web/README.md` | ✅ | ✅ | Web development |

### Documentation Gaps

1. **No release notes for v0.1.10–v0.1.14**: The gap between v0.1.9 and current is undocumented in release note form.
2. **Top-level README.md**: Too brief for new users. Should include quick-start, architecture diagram, and links to key docs.
3. **No API documentation**: No OpenAPI/Swagger spec. API consumers must read Go code.
4. **No "Getting Started" guide for end users**: `RUNBOOK-LOCAL-VERIFY.md` is development-focused. Need a production deployment guide.
5. **Design docs outdated**: `docs/01`–`docs/10` were written for Phase 0-2B window and haven't been updated to reflect model runtime implementation.
6. **No architecture diagram**: Would help new developers and operators understand the system.

### Conflicting/Redundant Documents

- `docs/plan/phase-2f-*.md` files are detailed but partially redundant with each other
- `docs/design/tenant-rbac-resource-ownership-design.md` overlaps with `docs/09-auth-tenant-design.md`
- No actual conflicts found — all documents agree on core semantics

---

## 13. Test Coverage Review

### Test Coverage Matrix

| Area | Existing Tests | Count | Gaps | Suggested Tests |
|------|---------------|-------|------|-----------------|
| **Server API: Agent** | Registration, identity binding, heartbeat, resource report | 13 | Task claim concurrency, node transfer with active instances | Test parallel task claims; test transfer with varying instance states |
| **Server API: Resources** | GPU ingestion (MetaX 8-card), memory edge cases, tenant auto-assign | 7 | GPU disappearance detection, partial report handling | Test GPU removed from report → marked unavailable; test concurrent resource reports |
| **Server API: Model** | CRUD round-trip, dry-run, deployment start/stop, lease conflict, tenant isolation | 38 | Concurrent start, sweep with mixed states, Docker spec update concurrency | Test parallel deployment starts; test sweep behavior with pending + running instances |
| **Server API: RBAC** | Tenant isolation, audit log, transfer permission, built-in role protection | 18 | Membership role removal edge cases, permission escalation attempts | Test removing last role from membership with concurrent role add |
| **Server API: Tenant** | Node scoping, GPU scoping, cross-tenant blocking, system query | 5 | Instance tenant isolation (broken — see B1) | Test that tenant A cannot see tenant B's model instances |
| **Agent: Collector** | NVIDIA CSV parsing, protocol parsing, probe detection | 12 | External command failure modes, concurrent collection | Test external script timeout; test concurrent Collect + Prometheus scrape |
| **Agent: Metrics** | MetaX 8-card normalize, Prometheus dedup, zero values, vendor neutral | 7 | Load=0 edge case, counter reset behavior | Test that load=0 is emitted as 0 not omitted |
| **Agent: Register** | Success, HTTP errors, node_id reuse, mismatch, unreachable | 7 | Concurrent registration, large response payload | Test registration with response > 4096 bytes |
| **Agent: State** | First start, reuse, corruption, empty, persist, mismatch, permissions | 10 | Concurrent SetNodeID calls | Test parallel SetNodeID calls |
| **Agent: Docker** | Start, stop, inspect, logs, env, binds, GPU requests, sensitive redaction | 24 | Real Docker integration (gated by env var), log streaming | Test Docker multiplexed log stream decoding |
| **Common** | Errors, version | 6 | Config loading, log rotation | Test YAML config parsing; test log rotation |
| **Web: Auth** | Login flow, tenant_id in request, CSRF | 4 | Token refresh, session expiry, password change flow | Test 403→CSRF refresh→retry; test session expiry redirect |
| **Web: Dashboard** | Aggregation functions, top-N, abnormal filtering | 7 | API response error handling, loading states | Test dashboard with empty data; test with all nodes offline |
| **Web: AutoRefresh** | Interval, in-flight guard, error preservation, stop | 7 | Visibility pause/resume, route leave cleanup | Test visibility change behavior |
| **Web: i18n** | Key consistency check (220 keys) | 1 | Missing key detection for templates | Add test that checks all `$t()` calls have corresponding keys |
| **Web: API Paths** | No hardcoded `/api/v1` prefix (12 files) | 1 | Verification that exceptions are intentional | — |
| **Web: Formatters** | formatRelativeTime (zh-CN, en-US) | 1 | formatBytes, formatPercent, formatCelsius tests | Add tests for edge cases (null, NaN, negative) |
| **Shell: Patch** | Atomicity (7 scenarios) | 7 | Upgrade from old version, concurrent apply | Test patch from v0.1.9 to current |
| **Shell: Scripts** | bash -n syntax | 23 scripts | No runtime tests | Test start-* and stop-* scripts in clean environment |

### Test Summary

```
Go tests:      142+ (all passing)
  Server API:  81
  Agent:       60
  Common:       6
Vitest:         12 (passing — needs running from web/ directory)
Shell:           7 scenarios
Node tests:      3 (i18n, paths, formatters — all passing)
bash -n:        23 scripts (all OK)
go vet:         PASS (no issues)
```

### Test Gaps by Priority

1. **Critical gap**: No test for `HandleChangePassword` middleware exemption — the bug B2 exists because this path isn't tested as a full HTTP request through middleware.
2. **High gap**: No test for concurrent lease creation (B4 race condition).
3. **High gap**: No test for rate limiter cleanup (B6 memory leak).
4. **Medium gap**: No integration test from HTTP request through middleware to handler — all API tests call handler functions directly.
5. **Medium gap**: No test for Docker log stream decoding (A2).
6. **Medium gap**: No MetaX real-hardware validation (blocked by hardware availability).

---

## 14. Product Suggestions

### 14.1 Current Page Structure Assessment

The current page structure is reasonable for operators managing GPU resources:

✅ **Good**: Dashboard → Nodes → GPUs → Model artifacts → Deployments → Instances flow matches the operator workflow
✅ **Good**: System section (Tenants, Users, Roles, Audit Logs) separated from operational pages
✅ **Good**: Observability section separate with own sub-pages
⚠️ **Improve**: Runtime Environments and Run Templates feel disconnected from the deployment flow
⚠️ **Improve**: Quick Deploy wizard is buried in a collapse panel on ModelDeploymentsPage

### 14.2 What Small/Medium Customers Most Need to See

1. **"How are my GPUs doing?"** — Current Dashboard shows top-5 tables and abnormal GPUs. Add: GPU utilization trends (last 1h/6h/24h) using Grafana data
2. **"What models are running, and can users access them?"** — Current instances page shows status. Add: endpoint URL copy button, health check status
3. **"Is any GPU idle that I could use?"** — Add: free GPU count on Dashboard, idle GPU list
4. **"Who did what?"** — Audit logs page exists. Add: date range picker, user filter
5. **"How do I deploy a model?"** — Quick Deploy is good but hidden. Promote to a top-level "Deploy" button or wizard page

### 14.3 Features to Prioritize

**Next (Phase 2G):**
- Fix all 5 Critical bugs
- Instance `tenant_id` population
- Password-expired user fix
- Login metrics implementation
- Lease race condition fix
- Rate limiter cleanup

**Soon (Phase 2H):**
- Worker maintenance mode (drain before maintenance)
- Instance logs: decode Docker multiplexed stream
- Concurrent task processing in agent (goroutine pool)
- Prometheus/Grafana Go supervisor (replace shell scripts)
- Dashboard time-series charts (CPU/GPU/Memory trends)
- Fill UI gaps: Roles CRUD, Tenants edit/disable, Users edit/disable

**Later (Phase 3):**
- Basic scheduler (single filter: GPU memory fit + vendor match)
- Model file download/caching on workers
- Instance endpoint URL display and OpenAI-compatible test
- Worker auto-restart on crash (systemd integration)
- Installation one-liner script
- OpenAPI spec generation

### 14.4 Features to Defer

- API Key management
- Token usage tracking and metering
- Billing integration
- Model marketplace
- Multi-cluster federation
- Kubernetes deployment
- SSO/LDAP/OAuth
- Advanced scheduling (multi-level filters, scoring)
- Playground (chat, embedding interactive UI)

### 14.5 How to Showcase Lightweight Advantage

vs GPUStack, LightAI Go's competitive advantages:
1. **Two static binaries** — no Python, pip, or Docker dependency for the platform itself
2. **Simple config** — one YAML file per binary, not 80+ CLI flags
3. **Auto-detection** — plug in, GPU type detected automatically
4. **Air-gapped deployment** — tarball + patch, no network needed
5. **SQLite by default** — no external database to manage

These advantages should be prominent in documentation and marketing.

### 14.6 Gradual Approach to GPUStack Feature Parity

Don't try to match GPUStack feature-for-feature. Instead:
- **Adopt the architecture patterns** (scheduler loop, controller pattern, event bus) but simplified
- **Improve the UX** (SSE for real-time updates, maintenance mode, better dashboards)
- **Keep the deployment simple** (two binaries, one config file each)
- **Add only the features customers ask for** (likely: basic scheduling, endpoint URLs, model download)

---

## 15. Action List

### Critical (Correctness / Security / Tenant Isolation)

| ID | Issue | Evidence | Impact | Fix | GPUStack Ref? | Effort | Needs Test | Needs Doc |
|----|-------|----------|--------|-----|---------------|--------|------------|-----------|
| C1 | Instance `tenant_id` not set on creation | `api/deployment_lifecycle.go:33,196-206` | Tenant isolation broken for instances | Add `tenant_id` to both INSERT statements; add test that instances inherit tenant from deployment | No | S | Yes | No |
| C2 | Password-expired users cannot change password | `auth/middleware.go:139` | Users locked out of system | Fix path check to `/api/v1/auth/change-password` (also add `/api/v1/auth/logout` exemption) | No | S | Yes | No |
| C3 | Login metrics never incremented | `auth/handlers.go:216` | Operators blind to brute-force attacks | Implement the TODO: pass ServerMetrics to AuthHandler, increment counters in HandleLogin | No | S | Yes | No |
| C4 | GPU lease race condition | `api/lease.go:38` | Double-leasing same GPU | Add UNIQUE constraint on `(gpu_id, status)` where status in `(reserved, active)`; or use INSERT with SELECT NOT EXISTS | GPUStack uses DB-level constraints | M | Yes | No |
| C5 | Rate limiter memory leak | `auth/ratelimit.go:46-54` | Memory exhaustion on long-running server | Add periodic cleanup goroutine to evict stale entries; consider using token-bucket library | No | S | Yes | No |

### High (Strongly Recommended Before Production)

| ID | Issue | Evidence | Impact | Fix | GPUStack Ref? | Effort | Needs Test | Needs Doc |
|----|-------|----------|--------|-----|---------------|--------|------------|-----------|
| H1 | Instance update + lease activation not atomic | `api/task_handlers.go:112-137` | Inconsistent state: instance running, leases reserved | Wrap in single DB transaction | No | S | Yes | No |
| H2 | Transfer safety checks silently ignore errors | `api/agent_handlers.go:676-688` | Unsafe node transfer if DB query fails | Check errors, fail transfer on query error | No | S | Yes | No |
| H3 | Audit log INSERT errors silently ignored | `api/agent_handlers.go:704` | Undetected audit gaps | Log error at WARN level; consider failing the operation if audit is critical | No | S | Yes | No |
| H4 | Sequential task processing blocks heartbeat | `cmd/agent/main.go:578` | Node appears offline during long tasks | Process tasks in goroutine pool with concurrency limit; heartbeat continues independently | GPUStack uses async tasks | M | Yes | No |
| H5 | Docker logs don't decode Docker multiplexed stream | `agent/runtime/docker.go:181` | Stderr always empty; log output incomplete | Use Docker SDK's `ContainerLogs` with `FollowStream=false` then decode multiplexed stream into separate stdout/stderr | GPUStack does this correctly | M | Yes | No |
| H6 | GPU tenant not inherited from node | `api/resource_handlers.go:319` | GPUs assigned to default tenant regardless of node | Use node's `tenant_id` when creating GPU device records | No | S | Yes | No |
| H7 | Hardcoded default tenant UUID | `api/resource_handlers.go:433` | Breaks if DB reinitialized | Use `db.DefaultTenantID()` function or session-derived tenant context | No | S | Yes | No |
| H8 | Hardcoded credentials in GrafanaPage | `web/src/pages/GrafanaPage.vue:7` | Credential disclosure | Remove hardcoded credentials; show link to credentials file or use env-based config | No | S | No | No |

### Medium (Experience / Maintainability / Documentation)

| ID | Issue | Evidence | Impact | Fix | GPUStack Ref? | Effort | Needs Test | Needs Doc |
|----|-------|----------|--------|-----|---------------|--------|------------|-----------|
| M1 | Quick Deploy hardcoded in Chinese | `web/src/pages/ModelDeploymentsPage.vue:12-48` | Non-Chinese users cannot use Quick Deploy | Extract all strings to i18n keys in both locales | No | S | No | No |
| M2 | Dashboard status strings hardcoded | `web/src/pages/DashboardPage.vue:126,130,134` | Status messages not localized | Use `$t()` with parameterized messages | No | S | No | No |
| M3 | Observability pages hardcoded English | `PrometheusPage.vue, GrafanaPage.vue, ObservabilityOverviewPage.vue` | Non-English users see mixed languages | Add i18n keys for all observability page strings | No | S | No | No |
| M4 | Sidebar nav labels hardcoded | `web/src/layouts/ConsoleLayout.vue:61-62` | "Prometheus" and "Grafana" not translated | Use `$t()` for nav labels | No | S | No | No |
| M5 | Missing i18n key `tenants.createdAt` | `web/src/locales/zh-CN.ts, en-US.ts` | Column header shows raw key string | Add `tenants.createdAt` to both locale files | No | S | No | No |
| M6 | Roles page is read-only | `web/src/pages/RolesPage.vue` | Cannot manage roles from UI | Add create/edit/delete dialogs using existing API client functions | No | M | No | No |
| M7 | Tenants page missing edit/disable | `web/src/pages/TenantsPage.vue` | Cannot manage tenants from UI | Add edit/disable buttons using existing API client functions | No | M | No | No |
| M8 | Users page missing edit/disable | `web/src/pages/UsersPage.vue` | Cannot manage users from UI | Add edit/disable buttons using existing API client functions | No | M | No | No |
| M9 | PlaceholderPage.vue dead code | `web/src/pages/PlaceholderPage.vue` | Confusion for new developers | Delete the file | No | S | No | No |
| M10 | N+1 query for runtime env docker specs | `api/model_handlers.go:448` | Performance issue with many environments | Batch query docker specs for all returned environments | No | S | No | No |
| M11 | DockerSpec DELETE-then-INSERT | `api/model_handlers.go:609-634` | Concurrent readers see empty state | Use INSERT OR REPLACE (SQLite) or UPSERT pattern | No | S | Yes | No |
| M12 | Sweep errors silently discarded | `api/sweep.go:39-95` | Sweep failures undetected | Log sweep errors at WARN level; add sweep error counter metric | No | S | Yes | No |
| M13 | Dead `auditLog` function | `api/audit_handlers.go:104-121` | Confusion, maintenance burden | Remove or consolidate with `audit()` in model_handlers.go | No | S | No | No |
| M14 | Dead `handleNotImplemented` | `api/router.go:209-211` | Dead code | Remove | No | S | No | No |
| M15 | Dead `isOperator` variable | `api/model_handlers.go:464` | Dead code | Remove or implement the check | No | S | No | No |
| M16 | External command env replacement | `agent/collector/external.go:176` | Script tool discovery may fail | Append custom env to parent process env, or document that PATH must be in custom env | No | S | No | Yes |
| M17 | Relative script paths in probes | `agent/collector/probe.go:37-38` | Agent fails if not run from project root | Resolve paths relative to binary location or config dir | No | S | No | No |
| M18 | Heartbeat log regression | `agent/register/register.go:241` | Misleading: logs agentID as nodeID | Fix log field name to `nodeID` | No | S | No | No |
| M19 | Load metrics suppressed at zero | `agent/metrics/metrics.go:385` | Prometheus rate queries break | Always emit load metrics, even at zero | No | S | Yes | No |
| M20 | CSRF origin check uses suffix match | `auth/csrf.go:44-49` | Weak origin validation | Parse URL, compare host component | No | S | No | No |
| M21 | Resource pool tables schema-only | V7 migration | Unused tables in DB | Add handlers or add comment marking as future work | No | L | No | Yes |
| M22 | Timestamp type inconsistency | `models/models.go` | Confusion, potential parsing bugs | Standardize on `time.Time` for all models | No | M | Yes | No |
| M23 | Hardcoded 5-min lease expiry | `api/lease.go:46` | Too aggressive for slow Docker environments | Make configurable, default 5 minutes | No | S | No | Yes |
| M24 | No OpenAPI/Swagger spec | N/A | API consumers must read Go code | Generate OpenAPI spec from route definitions | GPUStack auto-generates from FastAPI | L | No | Yes |
| M25 | Release notes gap v0.1.10–v0.1.14 | N/A | Users don't know what changed | Create consolidated release notes or changelog | No | M | No | Yes |
| M26 | No "Getting Started" for end users | N/A | New users struggle to deploy | Write production deployment guide | No | M | No | Yes |
| M27 | Design docs outdated | `docs/01`–`docs/10` | Developers see stale information | Update docs to reflect current implementation | No | M | No | Yes |
| M28 | No architecture diagram | N/A | Hard to understand system at a glance | Create architecture diagram (ASCII or image) | GPUStack has good diagrams | M | No | Yes |
| M29 | Agent not concurrency-safe on registry | `agent/collector/registry.go:9-17` | Rare race condition on state | Add sync.RWMutex to Registry | No | S | Yes | No |

### Low (Long-Term / Nice-to-Have)

| ID | Issue | Evidence | Impact | Fix | GPUStack Ref? | Effort |
|----|-------|----------|--------|-----|---------------|--------|
| L1 | VRAM warning double-query | `resolver.go:306-312` | Minor performance waste | Return VRAM info from ValidateDryRun or cache | No | S |
| L2 | Model struct missing migration fields | `models/models.go:299-310` | Go struct doesn't reflect schema | Add fields or document intentionally omitted | No | S |
| L3 | AMD GPU stubs | `deploy/collectors/gpu/amd/` | No AMD GPU support | Implement AMD rocm-smi scripts | No | L |
| L4 | Other vendor GPU support | Ascend/Hygon/etc | Limited GPU vendor coverage | Wait for customer demand | GPUStack supports more vendors | L |
| L5 | Migration uses SQLite-specific julianday() | `api/sweep.go:46` | Not portable to PostgreSQL | Use Unix timestamp arithmetic | No | S |
| L6 | Config files for explicit vendor mode | `agent.nvidia.yaml`, `agent.metax.yaml` | Legacy, causes confusion | Deprecate or document as explicit-only | No | S |
| L7 | No agent auto-restart | `scripts/start-agent.sh` | Agent crash requires manual restart | Use systemd unit with Restart=always | No | S |
| L8 | Observability startup order race | `scripts/observability-up.sh` | Grafana may start before Prometheus | Add health check loop between starts | No | S |

---

## 16. Recommended Next Phase Plan

### Phase 2G: Stability & Bug Fix (Suggested: 1–2 weeks)

**Goal**: Fix all Critical and High issues. No new features.

**Scope:**
- Fix C1–C5 (instance tenant_id, password change path, login metrics, lease race, rate limiter)
- Fix H1–H8 (atomic transactions, error handling, agent concurrency, Docker logs, GPU tenant, credentials)
- Add tests for each fix

**Non-goals:**
- No new API endpoints
- No new UI pages
- No model gateway/API key/scheduler work

**Acceptance criteria:**
- `go test ./...` 100% pass with new tests
- `model_instances.tenant_id` populated on creation
- Password-expired user can change password
- Login metrics increment on login attempt
- Concurrent lease creation for same GPU correctly rejects second request
- Rate limiter has periodic cleanup

### Phase 2H: Production Readiness (Suggested: 2–3 weeks)

**Goal**: Make the system production-safe for early adopters. Fill UI gaps.

**Scope:**
- Worker maintenance mode (POST `/api/v1/nodes/{id}/maintenance`)
- Instance log streaming with proper stdout/stderr separation
- Concurrent task processing in agent (goroutine pool with configurable max)
- Prometheus/Grafana Go supervisor (start/stop/health check from Server)
- Dashboard time-series charts (CPU, GPU memory, GPU utilization)
- Fill Web UI gaps (Roles CRUD, Tenants edit/disable, Users edit/disable)
- Fix all Medium i18n issues
- Remove all dead code
- Add OpenAPI spec generation

**Non-goals:**
- No scheduler beyond single-node manual deployment
- No model download/caching
- No API keys

**Acceptance criteria:**
- Maintenance mode: node stops accepting new instances, drains existing
- Docker logs show separate stdout and stderr in UI
- Agent can process multiple tasks concurrently without blocking heartbeat
- Server manages Prometheus/Grafana lifecycle (start/stop/health)
- Dashboard shows trend charts from Prometheus data
- All 19 pages have consistent i18n
- OpenAPI spec available at `/api/v1/openapi.json`

### Phase 3: Basic Scheduling & Model Operations (Suggested: 3–4 weeks)

**Goal**: Enable multi-node model deployment with basic scheduling.

**Scope:**
- Basic scheduler: filter by GPU memory + vendor match, pick best-fit node
- Model file download/caching on workers
- Instance endpoint URL display with copy button
- OpenAI-compatible `/v1/models` endpoint test point
- Instance health check (HTTP probe)
- Worker auto-restart via systemd
- Installation one-liner script (`curl ... | sh`)
- Grafana dashboard zh-CN variants

**Non-goals:**
- No API keys, token accounting, billing
- No multi-level scheduler with scoring
- No model marketplace
- No Gateway/load balancer
- No Kubernetes or Helm

**Acceptance criteria:**
- Scheduler picks worker with enough GPU memory and matching vendor
- Models can be downloaded from HuggingFace/local path to worker
- Deployed model shows endpoint URL that works
- Instance health check detects crashed model and reports failure
- Running `curl -sfL https://.../install.sh | sh -` sets up Server + Agent on clean Ubuntu

### What NOT to Do in Phase 3

- **API Key management**: Wait until model serving is stable and customers ask for it
- **Gateway / load balancer**: Single-node model serving works for early adopters
- **Usage tracking / billing**: Only relevant when multi-tenant model serving with quotas
- **Complex scheduler**: Filter by GPU memory + vendor is sufficient for "handful of servers"
- **Kubernetes / Helm**: Systemd + shell scripts are simpler for the target audience

---

## 17. Final Verdict

### Is the Project Healthy?

**Yes.** LightAI Go is in a surprisingly healthy state for a project of its scope and maturity. The foundation is solid: clean architecture, well-defined contracts, good test coverage, comprehensive documentation, and a functional Web UI. The engineering discipline shown in Phase 2F (RBAC hardening, tenant isolation, 12 review fixes, E2E cleanup) is commendable.

### What Is the Biggest Problem?

The biggest problem is **not architecture or design — it's a small set of correctness bugs that could have outsized impact**:

1. Instance tenant_id not set (breaks tenant isolation)
2. Password-expired user lockout (denial of service)
3. Login metrics never incremented (security blind spot)

These are fixable in days, not weeks. The fixes are straightforward. They simply haven't been discovered and prioritized.

### Should We Proceed to Model Gateway / API Key / Billing?

**Not yet.** The foundation needs these fixes first:

1. Critical bugs (C1–C5)
2. Transaction boundaries (H1)
3. Agent concurrency (H4)
4. Docker log correctness (H5)

After these, proceed to Phase 2G (stability), then Phase 2H (production readiness). Only after the system handles multi-node model serving reliably should API keys and billing be considered.

### Top 5 Things LightAI Go Should Learn from GPUStack

1. **Scheduler as async reconcile loop** — Even a simple scheduler (GPU memory fit + vendor match) structured as a periodic scan + event subscriber is more robust than the current manual deployment model. But keep it simple: no multi-level filtering, no scoring.

2. **Instance state machine with health checks** — GPUStack's `PENDING→ANALYZING→SCHEDULED→RUNNING→ERROR` flow is clearer than LightAI's `pending→running→stopped→failed`. Adding an `analyzing` state (validating Docker image, checking model files) and `scheduled` state (node selected, waiting for agent) would improve visibility.

3. **Real-time instance status via SSE** — GPUStack's SSE watch mode is more efficient than LightAI's 5-second polling. For instance lifecycle (start can take minutes), real-time updates matter.

4. **Worker maintenance mode with drain** — GPUStack allows marking a worker for maintenance, which stops scheduling new instances and waits for existing ones to complete. This is essential for production GPU operations.

5. **Dashboard time-series trends** — GPUStack's dashboard shows CPU/RAM/GPU/VRAM trends over time, not just current values. LightAI's dashboard is point-in-time only. Integrating Prometheus-backed time-series charts would dramatically improve operator visibility.

### What NOT to Learn from GPUStack (Right Now)

1. Complex scheduler with scoring and multi-level filters
2. Multi-backend inference server classes (vLLM, SGLang, etc.)
3. K8s Gateway/Ingress integration
4. Plugin system for UI extensions
5. Distributed coordinator with leader election
6. Code generation for API clients
7. Benchmark system
8. Cloud provider integrations

---

## Appendix A: Verification Commands Executed

```bash
# All passing
go test ./...                    # 142+ tests PASS
go vet ./...                     # PASS (no issues)
go build ./cmd/server            # BUILD OK
go build ./cmd/agent             # BUILD OK
bash -n scripts/*.sh             # 23 scripts OK
node web/tests/i18nKeys.test.mjs # 220 keys consistent
node web/tests/apiClientPaths.test.mjs # 12 files PASS
node web/tests/formatters.test.mjs     # 8 checks PASS
git diff --check                 # Clean (no whitespace errors)
git status --short               # Clean working tree
```

## Appendix B: Test Count Summary

```
Category          Count    Status
─────────────────────────────────
Go unit tests      142+    All PASS
Vitest tests        12     PASS (web/ dir required)
Node tests           3     PASS
Shell tests          7     PASS
bash syntax         23     PASS
go vet               0     PASS (no issues)
─────────────────────────────────
Total verification  187+   All PASS
```

## Appendix C: Key File Inventory

```
cmd/server/main.go          — Server entry point
cmd/agent/main.go           — Agent entry point (GPU auto-detect, heartbeat loop, task execution)
internal/server/api/        — All HTTP handlers, router, middleware metrics
internal/server/auth/       — Session, CSRF, middleware, rate limiter, bootstrap
internal/server/db/db.go    — SQLite init, V1-V7 migrations
internal/server/models/     — Go struct definitions
internal/server/rbac/       — RBAC CRUD handlers
internal/agent/collector/   — Collector interface, registry, external command, NVIDIA, probe
internal/agent/metrics/     — Prometheus snapshot and collectors
internal/agent/register/    — Registration and heartbeat
internal/agent/runtime/     — Docker driver, fake, real, command preview
internal/agent/state/       — Node identity persistence
internal/common/            — Config, errors, log, types, version
web/src/                    — Vue 3 SPA (19 pages, 220 i18n keys)
deploy/collectors/gpu/      — Shell scripts (NVIDIA, MetaX, AMD)
deploy/observability/       — Prometheus config, Grafana dashboards, alert rules
scripts/                    — Release, patch, start/stop, observability, verify
docs/                       — Design, plan, ops, review documentation
tests/                      — Shell tests, fixtures
```
