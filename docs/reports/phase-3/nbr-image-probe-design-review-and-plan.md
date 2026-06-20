# NBR Image Probe — Design Review & Phased Development Plan

> Status: DESIGN REVIEW
> Date: 2026-06-20
> For: `node-backend-runtime-image-probe-design.md` + `image-capability-probe.md`
> Baseline commit: c529f7d (blocker fix)

## 1. Current Implementation Status

### 1.1 What Is Implemented (Phase 0 Blocker Fix)

| Component | Status | Detail |
|-----------|--------|--------|
| Agent `/docker-image-inspect` | DONE | Returns full Docker inspect JSON |
| Server `check-request` handler | DONE | 4-level probe, ImageInspect authoritative |
| `matchBackendType` function | DONE | Hardcoded patterns, vendor-blind, lenient |
| `probe_results_json` column | DONE | V24 migration, on `node_backend_runtimes` |
| `evaluateProbeStatus` function | DONE | 8-status model, ImageInspect-based `missing_image` |
| Web wizard step 4 probe display | DONE | Collapsible panels in wizard + detail drawer |
| `status.ts` mappings | DONE | New statuses mapped |
| 12 check-request tests | DONE | Real HTTP router, fixture agents |
| `/version-probe` endpoint | REMOVED | Security deferred |

### 1.2 What Is NOT Yet Implemented

| Design Element | Status | Gap |
|---------------|--------|-----|
| Independent probe table | NOT STARTED | `probe_results_json` is a JSON blob on NBR row |
| NBR probe/recheck API formalization | NOT STARTED | Uses `check-request`, not `probe` |
| NBR list page aggregated status | PARTIAL | Binary ready/non-ready tag, no probe-level detail |
| NBR detail page per-node probe | PARTIAL | Drawer shows probe, but no node selector |
| Backend match catalog integration | NOT STARTED | Hardcoded `patterns` map in handler |
| Script probe | NOT STARTED | No agent endpoint, no server logic |
| Version probe | DEFERRED | Stub function returns nil; endpoint removed |
| Start Wizard probe integration | NOT STARTED | Wizard step 3 shows only status tag |
| Preflight probe snapshot usage | NOT STARTED | Preflight queries NBR status, not probe details |
| i18n for probe statuses/panels | PARTIAL | Missing `status.ready_with_warnings`, probe panel labels hardcoded in English |

## 2. Gap Analysis: Design vs Code

### 2.1 Data Model

**Design says**: Probe snapshot must be node-scoped. Recommended independent table `node_backend_runtime_probe_snapshots` with rich fields.

**Code reality**: `probe_results_json` TEXT column on `node_backend_runtimes`. It IS node-scoped because NBR is `UNIQUE(node_id, backend_runtime_id)`. However:
- JSON blob makes query-by-field impossible (e.g., "find all NBRs with `image_id=sha256:xxx`")
- No history/versioning of probe results
- No "stale" detection without re-probing
- Single latest snapshot, no re-check trail

**Assessment**: Current JSON blob is acceptable for single-node, single-snapshot use. But for `node_id + image_ref` indexing, multi-node aggregation, stale detection, and Start Wizard filtering, an independent table would be superior.

**Recommendation**: Keep current JSON blob for Phase 1-2. Plan independent table for Phase 4+.

### 2.2 Multi-Node Capability

**Design says**: If one logical NBR covers multiple nodes, each node must have independent probe snapshot. Aggregated status for list summary.

**Code reality**: NBR is strictly `UNIQUE(node_id, backend_runtime_id)`. One NBR = one node. No concept of "logical NBR covering multiple nodes". `run_plan_groups` has `mode='single'`; `replicas` field exists but unused.

**Assessment**: No immediate gap — the design's concern about multi-node probe collision is already prevented by the schema. If multi-node NBR is introduced later, the independent probe table will become necessary.

### 2.3 Backend Match Rules

**Design says**: Match rules should come from Backend catalog/BackendVersion config, not hardcoded. `vendor=nvidia` must not derive `backend=vllm`.

**Code reality**: `matchBackendType` is hardcoded with 4 patterns (`vllm`, `sglang`, `llamacpp`, `ollama`). Vendor IS correctly ignored. However:
- Adding a new backend requires code change
- No catalog-driven pattern configuration
- MetaX/Huawei images correctly get `declared_match_unverified`
- `confirmed_mismatch` status exists in enum but is NEVER returned (matchBackendType always returns `declared_match_unverified` on no-match, never `confirmed_mismatch`)

**Assessment**: Functional for current 4 backends. Catalog integration should be planned but is not blocking.

### 2.4 Agent Capabilities

**Design says**: Agent needs `/docker-image-inspect` (exists check + metadata), `/docker-image-script-probe` (static script read), and version probe (deferred).

**Code reality**: `/docker-image-inspect` exists and works. Script probe endpoint does not exist. Version probe endpoint was removed for security review.

**Assessment**: Core capability (`/docker-image-inspect`) is sufficient for Phase 1-2. Script probe needs design + implementation before Phase 5+.

### 2.5 Server API

**Design says**: `POST /api/v1/nodes/{node_id}/backend-runtimes/{nbr_id}/probe`, `GET /api/v1/nodes/{node_id}/backend-runtimes/{nbr_id}/probe`, `POST /api/v1/backend-runtimes/{nbr_id}/probe-all-nodes`.

**Code reality**: `POST .../{nbr_id}/check-request` exists and functions as the probe endpoint. No dedicated probe GET endpoint. No probe-all-nodes endpoint (not needed — NBR is 1:1 with node).

**Assessment**: Current `check-request` endpoint serves the probe function. Formalizing it as `probe` (semantic rename) could be done but is low priority.

### 2.6 Web Display

**Design says**: Six areas — wizard last step ("校验与运行预览"), list page (aggregated status), detail page (per-node tabs: overview, node list, image & probe, run params, RunPlan preview), entrypoint shell wrapper notice, vendor image notice, warning/error layering.

**Code reality**:
- Wizard step 4: Has probe collapsible panels (image metadata + backend match + version probe)
- List page: Binary `ready` success / non-ready warning tag — no probe-level differentiation
- Detail drawer: Has probe collapsible panels, no "select node" concept (NBR is single-node)
- Probe panel labels: Hardcoded English ("Image ID", "Architecture", etc.)
- `formatBytes`: Hardcoded English units ("B", "KB", "MB", "GB")
- Missing i18n: `ready_with_warnings`, `inspect_failed`, `docker_error`, `agent_unreachable`
- Deploy wizard: No probe inline display

**Assessment**: Wizard step 4 display is functional but needs i18n. List page needs status tag differentiation. Deploy wizard step 3 needs probe status badges.

### 2.7 Script Probe

**Design says**: Agent uses `docker create` + `docker cp` + `docker rm` to read startup scripts. No container execution. Content truncated. Failure = warning only.

**Code reality**: Not implemented. Agent has no script probe endpoint.

**Assessment**: This is a Phase 5+ feature. Can be designed now, implemented later.

### 2.8 Version Probe

**Design says**: `--pull=never --network=none --cap-drop=ALL --security-opt no-new-privileges`, no GPU, no mounts, 5-10s timeout, stdout truncated. Best-effort, non-blocking. Command from catalog config.

**Code reality**: Endpoint removed. Stub function `getVersionProbeConfig` returns nil. Catalog YAML has no `version_probe` field.

**Assessment**: This is a Phase 5+ feature. Security review must precede implementation.

## 3. Data Model Recommendations

### 3.1 Short-Term (Phase 1-2): Keep Current JSON Blob

`probe_results_json` on `node_backend_runtimes` is functionally correct for single-node, single-snapshot use.

### 3.2 Long-Term (Phase 4+): Independent Probe Table

```sql
CREATE TABLE node_backend_runtime_probe_snapshots (
    id TEXT PRIMARY KEY,
    node_backend_runtime_id TEXT NOT NULL REFERENCES node_backend_runtimes(id),
    node_id TEXT NOT NULL REFERENCES nodes(id),
    agent_id TEXT NOT NULL DEFAULT '',
    image_ref TEXT NOT NULL DEFAULT '',
    image_id TEXT NOT NULL DEFAULT '',
    repotags_json TEXT NOT NULL DEFAULT '[]',
    repodigests_json TEXT NOT NULL DEFAULT '[]',
    os TEXT NOT NULL DEFAULT '',
    architecture TEXT NOT NULL DEFAULT '',
    size_bytes INTEGER NOT NULL DEFAULT 0,
    image_created_at TEXT NOT NULL DEFAULT '',
    entrypoint_json TEXT NOT NULL DEFAULT '[]',
    cmd_json TEXT NOT NULL DEFAULT '[]',
    env_json TEXT NOT NULL DEFAULT '[]',
    exposed_ports_json TEXT NOT NULL DEFAULT '{}',
    labels_json TEXT NOT NULL DEFAULT '{}',
    inspect_json TEXT NOT NULL DEFAULT '{}',
    backend_match_status TEXT NOT NULL DEFAULT 'not_checked',
    backend_match_method TEXT NOT NULL DEFAULT '',
    backend_match_detail TEXT NOT NULL DEFAULT '',
    warnings_json TEXT NOT NULL DEFAULT '[]',
    errors_json TEXT NOT NULL DEFAULT '[]',
    final_status TEXT NOT NULL DEFAULT 'evidence_missing',
    checked_at TEXT NOT NULL DEFAULT '',
    operation_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_nbr_probe_nbr ON node_backend_runtime_probe_snapshots(node_backend_runtime_id, node_id);
CREATE INDEX idx_nbr_probe_node_image ON node_backend_runtime_probe_snapshots(node_id, image_ref);
CREATE INDEX idx_nbr_probe_node_image_id ON node_backend_runtime_probe_snapshots(node_id, image_ref, image_id);
CREATE INDEX idx_nbr_probe_final_status ON node_backend_runtime_probe_snapshots(node_backend_runtime_id, final_status);
```

This enables:
- Query by node + image_ref (cross-NBR image consistency check)
- History of probe results per NBR (keep last N snapshots)
- Efficient aggregation for list page
- Start Wizard filtering ("find NBRs on node X with probe status ready")

## 4. API Recommendations

### 4.1 Current State

| Route | Method | Function |
|-------|--------|----------|
| `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check-request` | POST | Probe NBR (creates snapshot) |

### 4.2 Recommended Future State

| Route | Method | Function |
|-------|--------|----------|
| `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe` | POST | Probe NBR (recheck) — replaces check-request |
| `GET /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe` | GET | Get latest probe snapshot for this NBR |
| `GET /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe-history` | GET | Get probe history (if independent table) |

**Note**: No `probe-all-nodes` needed — NBR is 1:1 with node.

**Route path check**: Current `check-request` route uses `{id}` for node_id and `{nbr_id}` for NBR ID. Handler reads `r.PathValue("id")` and `r.PathValue("nbr_id")`. This is already verified by `TestCheckRequestEndpointPathValuesCorrect`.

## 5. Agent Capability Recommendations

### 5.1 Current

| Endpoint | Status |
|----------|--------|
| `GET /docker-images` | EXISTS — UI list source |
| `GET /docker-image-inspect` | EXISTS — authoritative check |

### 5.2 Recommended Future

| Endpoint | Method | Phase | Description |
|----------|--------|-------|-------------|
| `GET /docker-image-inspect` | GET | Existing | Keep as-is, enhance error classification |
| `POST /docker-image-script-probe` | POST | Phase 5 | Static script read via `docker create` + `docker cp` + `docker rm` |
| `GET /version-probe` | GET | Phase 6 | Deferred until security review complete |

## 6. Server Orchestration Recommendations

### Current Flow
```
POST check-request
  → resolve node → agent
  → Level 1: GET /docker-images (evidence only)
  → Level 2: GET /docker-image-inspect (authoritative)
  → Level 3: matchBackendType() (hardcoded)
  → Level 4: getVersionProbeConfig() (stub, returns nil)
  → evaluateProbeStatus() → status + reason
  → UPDATE node_backend_runtimes SET probe_results_json, status, ...
```

### Recommended Future Flow
```
POST probe
  → resolve node → agent
  → Level 1: GET /docker-images (evidence)
  → Level 2: GET /docker-image-inspect (authoritative)
  → Level 3: matchBackendType() from catalog config
  → Level 4: (Phase 5) POST /docker-image-script-probe
  → Level 5: (Phase 6) GET /version-probe (catalog-driven, secured)
  → evaluateProbeStatus() → status + reason
  → INSERT INTO node_backend_runtime_probe_snapshots (Phase 4)
  → UPDATE node_backend_runtimes SET status, probe_results_json (summary only)
```

### Separation of Concerns

- `probe_results_json` on NBR = summary (latest status, key warnings, checked_at)
- `node_backend_runtime_probe_snapshots` = detailed evidence (inspect JSON, match details, probe history)

## 7. Web Page Recommendations

### 7.1 NBR Wizard Final Step ("校验与运行预览")

**Current**: Shows alert banner + collapsible panels (Image Metadata, Backend Match, Version Probe). Enable for `ready` and `ready_with_warnings`.

**Recommended**:
- Add i18n labels for all panel titles and field names (currently hardcoded English)
- Show `formatBytes` with locale-aware units
- Add shell wrapper notice when entrypoint is `bash`/`sh`/`python`:
  > "入口类型：Shell wrapper — 真实服务参数可能在 Cmd 或启动脚本中"
- Add vendor image notice when `backend_match_status == declared_match_unverified`:
  > "该镜像可能是厂商自定义封装，未能从镜像名/labels 确认 backend 类型。已按用户声明的 backend 接受。"
- Current panel structure is adequate for Phase 1-3

### 7.2 NBR List Page

**Current**: Binary `ready` (success) vs non-ready (warning) status tag.

**Recommended**:
- Status tag differentiation:
  - `ready` → `success` (green)
  - `ready_with_warnings` → `warning` (orange)
  - `missing_image` → `danger` (red)
  - `inspect_failed` → `danger` (red)
  - `agent_unreachable` → `danger` (red)
  - `docker_error` → `danger` (red)
  - `runtime_image_mismatch` → `danger` (red)
  - `needs_check` → `info` (grey)
  - `evidence_missing` → `info` (grey)
- Add i18n keys for all statuses
- Show `last_checked_at` and probe-level summary in tooltip

### 7.3 NBR Detail Page

**Current**: Drawer with descriptions + collapsible probe panels.

**Recommended**: Convert to full page (or larger drawer) with tabs:
1. **概览 (Overview)**: name, node, backend, vendor, image_ref, status, last_checked
2. **镜像与探测 (Image & Probe)**: ImageInspect metadata, backend match, probe warnings/errors
3. **运行参数 (Run Parameters)**: config_snapshot_json display, docker command preview
4. **RunPlan 预览 (RunPlan Preview)**: Generated docker run command, mounts, ports, env

Node selector not needed — NBR is single-node.

### 7.4 Deploy Wizard Step 3 (NBR Selection)

**Current**: Status tag only.

**Recommended**: Add probe status badge per NBR row:
- `ready` → green checkmark
- `ready_with_warnings` → orange checkmark with tooltip listing warnings
- Blocking statuses → red X, row disabled

## 8. Start Wizard / Preflight Integration

### Current
- Preflight checks NBR `status` field only
- Does not read `probe_results_json`
- Error codes: `nbr_not_ready`, `docker_image_missing`

### Recommended
- Preflight should also read `probe_results_json` for additional context:
  - `image_id` consistency check against last probe
  - `backend_match_status` warning relay
  - Probe staleness check (`checked_at` age)
- The existing `status` field is sufficient as the primary gate — probe details augment with warnings

## 9. Test Strategy

### 9.1 Status Mapping Matrix (Unit Tests)

Must test every input → output pair:

```
Input                                  Expected Status
─────────────────────────────────────────────────────
ImageInspect success, match confirmed → ready
ImageInspect success, match unverified → ready_with_warnings
ImageInspect "no such image"          → missing_image
ImageInspect "docker daemon error"    → inspect_failed
Agent HTTP connection refused         → agent_unreachable
Agent HTTP 500                        → docker_error
No image_ref in NBR                   → evidence_missing
List misses, Inspect succeeds         → NOT missing_image (ready or ready_with_warnings)
List decode error, Inspect succeeds   → NOT missing_image
```

### 9.2 Real HTTP Router Tests

Already implemented (12 tests in `runtime_boundary_test.go`). Maintain and extend.

### 9.3 List-to-Probe Consistency Test

Simulate complete user flow:
1. Agent `/docker-images` returns image list
2. Select image from list
3. Submit probe
4. Assert: NOT `missing_image`

### 9.4 Vendor Image Test

Simulate MetaX/Huawei self-built image:
- `image_ref=registry.local/metax/runtime:latest`
- `entrypoint=["/bin/bash"]`, `cmd=["/opt/start.sh"]`
- `labels={}`
- `backend_id=vllm`
- Expected: `ready_with_warnings`, `backend_match_status=declared_match_unverified`

### 9.5 Multi-Node Image ID Drift Test

Simulate:
- `node-A: image_id=sha256:aaa`
- `node-B: image_id=sha256:bbb` (same `image_ref`)
- Expected: probe snapshots are independent per node

### 9.6 Web E2E Test

`list → select → probe → save → detail` complete UI chain.

### 9.7 Frontend Component Tests

- Status tag renders correct color per status
- Probe panels render correct metadata
- Shell wrapper notice when entrypoint is bash
- Vendor image notice when match unverified

## 10. Risk Points

| Risk | Severity | Mitigation |
|------|----------|-----------|
| Hardcoded match patterns miss new backends | Low | Catalog-driven design in Phase 5 |
| JSON blob unqueryable for cross-NBR checks | Medium | Independent probe table in Phase 4 |
| Version probe security surface | High | Deferred; strict security checklist before implementation |
| i18n drift (hardcoded English in probe panels) | Low | Add i18n keys in Phase 1 |
| Agent binary not available in some envs | Low | `/docker-image-inspect` uses same `execCmd` pattern as `/docker-images` |
| Probe snapshot not used by Start Wizard | Medium | Integrate in Phase 6 |

## 11. Implementation Readiness

**Status**: READY for Phase 1-3 implementation. Phase 4+ requires additional design review.

## 12. Questions Requiring Confirmation

1. **Probe table vs JSON blob**: Should Phase 4 implement an independent `node_backend_runtime_probe_snapshots` table, or keep JSON blob indefinitely? The independent table adds complexity but enables cross-NBR queries.

2. **NBR detail page scope**: Full page with tabs, or larger drawer? Full page is better UX but requires routing changes.

3. **Multi-node NBR timeline**: Is there a planned timeline for "logical NBR covering multiple nodes"? If near-term, probe table design should account for this.

4. **Script probe priority**: Should Phase 5 (script probe) be accelerated? It significantly improves the "entrypoint is bash" diagnostic case.

5. **Version probe re-enablement**: When should version probe security review happen? Before or after Phase 5?

6. **Backend catalog matching fields**: Can we add `match_patterns` and `version_probe` fields to BackendVersion YAML schema? This requires updating the catalog loader and all existing YAML files.

---

## Phased Development Plan

### Phase 1: Probe Status Model & API Response Schema Consolidation

**Goal**: Formalize the status model, i18n, and response schema. No behavioral changes.

**Modifications**:
- `web/src/locales/zh-CN.ts`: Add `status.ready_with_warnings`, `status.inspect_failed`, `status.docker_error`, `status.agent_unreachable`, `status.runtime_image_mismatch`, `status.evidence_missing`, probe panel labels, `formatBytes` locale units
- `web/src/locales/en-US.ts`: Same keys in English
- `web/src/utils/status.ts`: Add all new statuses to `getStatusType()`, add reason mappings
- `web/src/pages/RunnerConfigsPage.vue`: Replace hardcoded English labels with `$t()` calls; replace `formatBytes` with `$t()`-based locale units
- `internal/server/api/runtime_handlers.go`: No changes (status enum is already defined)

**API Changes**: None

**DB Migration**: None

**Tests**:
- `npm test`: i18n keys consistent, no key leaks
- Status tag renders correct Element Plus type per status

**Acceptance Criteria**:
- `ready_with_warnings` displays as "就绪（有警告）" in zh-CN
- Probe panel labels display in current locale
- `formatBytes` uses locale-aware units
- All 606+ i18n key references resolve

**Risk**: Low

**Rollback**: Safe — i18n changes are additive

---

### Phase 2: NBR List Page Aggregated Status

**Goal**: List page status tag differentiates probe-level statuses.

**Modifications**:
- `web/src/pages/RunnerConfigsPage.vue`: Replace binary `row.status==='ready'?'success':'warning'` with `getStatusType(row.status)`
- `web/src/utils/status.ts`: Already maps all statuses in Phase 1

**API Changes**: None (list endpoint already returns `status` field)

**DB Migration**: None

**Tests**:
- Component test: each status renders correct tag type

**Acceptance Criteria**:
- `ready` → green success tag
- `ready_with_warnings` → orange warning tag
- `missing_image` → red danger tag
- `inspect_failed` → red danger tag
- All statuses have visible i18n text

**Risk**: Low

**Rollback**: Safe — purely cosmetic

---

### Phase 3: NBR Probe/Recheck API Formalization

**Goal**: Rename `check-request` to `probe`, add GET endpoint for probe results, formalize response schema.

**Modifications**:
- `internal/server/api/runtime_handlers.go`: Add `HandleProbeNodeBackendRuntime` (wraps check-request logic), add `HandleGetNodeBackendRuntimeProbe` (GET latest probe)
- `internal/server/api/router.go`: 
  - Add `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe` → `HandleProbeNodeBackendRuntime`
  - Add `GET /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe` → `HandleGetNodeBackendRuntimeProbe`
  - Keep old `check-request` route for backward compatibility (delegate to new handler)
- `internal/server/api/runtime_boundary_test.go`: Add tests for new routes, verify PathValue names

**API Changes**:
- New `POST .../probe` — semantic equivalent of `check-request`
- New `GET .../probe` — returns latest probe snapshot
- Old `POST .../check-request` — preserved, delegates to probe handler

**DB Migration**: None

**Tests**:
- `TestProbeEndpointPathValuesCorrect`: Route `{id}` → handler reads `PathValue("id")`
- `TestProbeEndpointPathValuesCorrect`: Route `{nbr_id}` → handler reads `PathValue("nbr_id")`
- `TestGetProbeReturnsSnapshot`: GET returns probe_results
- `TestCheckRequestBackwardCompat`: Old route still works

**Acceptance Criteria**:
- `POST .../probe` returns same schema as `check-request`
- `GET .../probe` returns probe snapshot (200) or empty (404 if never probed)
- Old `check-request` continues to work
- All existing tests pass

**Risk**: Low (additive, backward compatible)

**Rollback**: Safe — old route preserved

---

### Phase 4: NBR Detail Page Enhancements

**Goal**: Detail page shows probe results in a structured, i18n-complete format with entrypoint shell wrapper notice and vendor image notice.

**Modifications**:
- `web/src/pages/RunnerConfigsPage.vue`:
  - Convert drawer to tabbed layout (Overview, Image & Probe, Run Parameters)
  - Add shell wrapper notice when `entrypoint` is `bash`/`sh`/`python`
  - Add vendor image notice when `backend_match_status == declared_match_unverified`
  - All labels via `$t()`
- `web/src/locales/zh-CN.ts`: Add probe detail section labels
- `web/src/locales/en-US.ts`: Same

**API Changes**: None (already returns `probe_results_json`)

**DB Migration**: None

**Tests**:
- Component test: shell wrapper notice renders when entrypoint is bash
- Component test: vendor image notice renders when match unverified
- i18n: no key leaks

**Acceptance Criteria**:
- Entrypoint bash shows "Shell wrapper" notice
- Vendor image shows "declared match not verified" notice
- All labels in current locale

**Risk**: Low

**Rollback**: Safe — purely frontend

---

### Phase 5: Backend Match Catalog Integration

**Goal**: Move match patterns from hardcoded Go map to BackendVersion catalog YAML.

**Modifications**:
- `configs/backend-catalog/versions/*/*.yaml`: Add optional `match_patterns` field to each BackendVersion
- `internal/server/api/backend_handlers.go`: Extend `backendVersionCatalogDoc` struct with `MatchPatterns []string`
- `internal/server/api/runtime_handlers.go`: Rewrite `matchBackendType` to read patterns from BackendVersion config (via `version_snapshot_json` in BackendRuntime or direct DB query)
- `internal/server/db/db.go`: Add `match_patterns_json` column to `backend_versions` (migration V25)

**API Changes**: None

**DB Migration**: V25 — `ALTER TABLE backend_versions ADD COLUMN match_patterns_json TEXT NOT NULL DEFAULT '[]'`

**Tests**:
- `TestMatchBackendTypeFromCatalog`: Patterns from catalog correctly used
- `TestMatchBackendTypeFallbackHardcoded`: When catalog has no patterns, fall back to hardcoded
- `TestMatchBackendTypeVendorImage`: MetaX image with no matching pattern → declared_match_unverified

**Acceptance Criteria**:
- vLLM catalog `match_patterns: [vllm, vllm-openai]` used for matching
- New backend can be added via catalog YAML without code change
- Existing hardcoded patterns serve as fallback

**Risk**: Medium (catalog schema change)

**Rollback**: Safe — hardcoded fallback preserves existing behavior

---

### Phase 6: Start Wizard / Preflight Probe Integration

**Goal**: Start Wizard and Preflight use probe snapshot for richer validation.

**Modifications**:
- `web/src/pages/ModelDeploymentsPage.vue`: Add probe status badges to NBR selection step
- `internal/server/api/preflight_handlers.go`: Read `probe_results_json`, add warnings for stale probe, image_id drift
- `internal/server/api/deployment_lifecycle_handlers.go`: No changes (NBR `status` field is sufficient gate)

**API Changes**: Preflight response adds optional `probe_warnings` field

**DB Migration**: None

**Tests**:
- Component test: probe badges in deploy wizard
- API test: preflight returns probe warnings when applicable

**Acceptance Criteria**:
- Deploy wizard step 3 shows probe status per NBR
- `ready_with_warnings` NBR can be selected with warning notice
- Blocking status NBR cannot be selected
- Preflight warns on probe staleness (>24h since last check)

**Risk**: Low (additive)

**Rollback**: Safe — preflight field is optional

---

### Phase 7: Script Probe Design & Implementation

**Goal**: Implement static script probe for entrypoint shell wrappers.

**Design requirements** (implementation deferred to separate plan):
- Agent endpoint: `POST /docker-image-script-probe` with body `{image_ref, script_paths[]}`
- Implementation: `docker create` + `docker cp` + `docker rm` (no container execution)
- Content truncated to 32KB
- Failure → warning only, never blocking
- Best-effort extraction of known commands (`python -m vllm`, `llama-server`, etc.)

**Security boundaries**:
- No `docker run` (no container execution)
- No mounts, no GPU, no privileged
- Timeout on `docker cp`

**Acceptance Criteria**: Separate design review required before implementation.

---

### Phase 8: Full Test Suite & Regression

**Goal**: Complete test coverage and regression verification.

**Modifications**:
- `runtime_boundary_test.go`: Extend test matrix
- Web component tests: Probe panels, status tags, wizard flow
- E2E test: `list → select → probe → save → detail` UI chain

**Acceptance Criteria**:
- All existing tests pass
- New tests cover all status mappings
- Web i18n keys consistent
- `go test ./...` PASS
- `npm test` PASS
- `git status --short` clean

---

## Summary: Recommended Implementation Order

| Phase | Priority | Effort | Dependencies |
|-------|----------|--------|-------------|
| Phase 1 (i18n + schema) | HIGH | Small | None |
| Phase 2 (list status) | HIGH | Small | Phase 1 |
| Phase 3 (API formalization) | MEDIUM | Medium | Phase 1 |
| Phase 4 (detail page) | MEDIUM | Medium | Phase 1 |
| Phase 5 (catalog match) | MEDIUM | Large | Phase 3 |
| Phase 6 (Start Wizard) | MEDIUM | Medium | Phase 3 |
| Phase 7 (script probe) | LOW | Large | Separate design review |
| Phase 8 (full tests) | HIGH | Medium | All implemented phases |

**Recommended first batch**: Phases 1 + 2 (can be done together, both small, no API changes).

**Recommended second batch**: Phases 3 + 4 (API formalization + detail page, medium effort, builds on Phase 1).

**Recommended third batch**: Phase 5 (catalog integration, larger effort, requires catalog schema design).

**Defers**: Phase 6 (Start Wizard) until Phases 1-4 done; Phase 7 (script probe) until separate security design review.
