# 12 — Claude Design Review Feedback

> Status: FINAL
> Scope: Review of resource scheduling design, manual verification issues, and implementation plan
> Date: 2026-06-22
> Reviewed documents: 09, 10, 11

## 1. Overall Verdict

**ACCEPT_WITH_REVISIONS**

The design documents are thorough and well-structured. The core separation of Deployment/Instance/RunPlan, vendor-neutral accelerator model, and phased implementation are all correct. However, several items need revision before Phase 1 implementation begins. None are blocking — all are addressable through document updates or scoping clarifications.

## 2. Design Strengths

### 2.1 Deployment / Instance / RunPlan Boundary

The separation is clear and correct. Keeping Deployment as intent and Instance as execution is the right abstraction for multi-node/multi-replica growth. The design avoids the common trap of merging these concepts.

### 2.2 Vendor-Neutral Accelerator Model

The `accelerator_ids` → DeviceBinding flow (`nvidia_device_request` / `metax_device_paths` / `cpu_none`) is well-defined. The document correctly bans `MACA_VISIBLE_DEVICE` and `METAX_VISIBLE_DEVICES`, and properly distinguishes NVIDIA DeviceRequest from MetaX device paths.

### 2.3 Backend-Specific Performance Parameters

The per-backend parameter maps (vLLM gpu_memory_utilization / SGLang mem_fraction_static / llama.cpp ctx_size) are accurate and match actual backend CLI conventions. Keeping these separate from vendor binding is correct — they are backend-layer concerns, not GPU-driver concerns.

### 2.4 Scheduling Degradation Strategy

"Single-candidate scheduling + resource validation" is the right Phase 1 approach. It's honest about current capability while keeping the `replicas`/`placement` naming that enables future multi-node expansion.

### 2.5 Problem Documentation

MV-001 through MV-009 are well-catalogued with clear reproduction, expected behavior, root cause hypotheses, and fix plans.

## 3. Design Risks

### 3.1 Schema Boundary Transition Is Undocumented

**Severity: HIGH — Must address before Phase 1**

Earlier rounds (01-08) had an absolute constraint: no schema changes, no migrations. Document 11 now says "必要 schema/API 演进" is permitted. This is the correct decision — model capability persistence requires it — but the transition from "no schema" to "allowed schema" needs an explicit declaration. Without it, future implementers will be confused about what boundary applies.

**Recommendation**: Add a section to document 11 stating: "Prior rounds enforced a no-schema boundary for presentation-only work. Starting Phase 1 of this plan, targeted schema additions for model capabilities and resource parameters are explicitly permitted. All additions must be minimal and focused."

### 3.2 Model Capability Persistence: Three Options, No Decision

**Severity: MEDIUM — Must decide before Phase 2**

Document 11 offers three options (metadata JSON, new columns, new table) without selecting one. This ambiguity will stall Phase 2 implementation.

**Recommendation**: Select Option B (new columns on model_artifacts: `capabilities_json TEXT`, `capability_sources_json TEXT`, `default_test_mode TEXT`). Rationale:
- Option A (metadata JSON overload) conflates scan metadata with user overrides
- Option C (new table) is overkill for a simple list + source mapping
- Option B is minimal, queryable via SQLite JSON functions if needed, and doesn't require joins

### 3.3 Resource Policy JSON Is Not Grounded in Current Schema

**Severity: MEDIUM — Must reconcile with config_snapshot_json**

The proposed `resource_policy` JSON (section 5.1 of doc 09) is aspirational. Phase 1 doesn't implement it as a first-class object. The current implementation must instead route these parameters through existing fields (primarily `config_snapshot_json` and `parameters_json`). The design should explicitly map "resource policy parameters → existing storage fields" for Phase 1.

**Recommendation**: Add a field mapping table:
```
gpu_memory_utilization → parameters_json (already supported)
max_model_len          → parameters_json (already supported)
shm_size               → docker_json in config_snapshot_json
cpu_limit              → not yet in any field; document as P2
memory_limit_bytes     → not yet in any field; document as P2
ulimits                → docker_json in config_snapshot_json (supported)
tensor_parallel_size   → parameters_json
```

### 3.4 MV-008 (Qwen3 404) Root Cause Is Unconfirmed

**Severity: MEDIUM — Needs diagnosis before code fix**

The Qwen3 Chat Completion 404 has multiple possible causes (wrong port, wrong endpoint, model ID mismatch, container entrypoint). The fix plan lists diagnostic commands but the repair step says "若根因可修，直接修复" — this is correct but means Phase 1 may discover the root cause requires changes beyond the planned scope.

**Recommendation**: Make Phase 1 "Qwen3 404 diagnosis" a separate step from "Qwen3 404 fix". Diagnosis output should be recorded in the closeout document even if the fix requires Phase 2+ work.

### 3.5 Huawei Vendor Is a Design Placeholder

**Severity: LOW — Acceptable for Phase 1**

The design says "通过 runtime template/catalog 定义，不凭空硬编码未知参数". This is correct given no Huawei hardware access, but the document should acknowledge this means Huawei is NOT validated even at the dry-run level.

## 4. Must-Fix Issues Before Implementation

### 4.1 Schema Boundary Declaration (doc 11)

Add an explicit statement: "Starting Phase 1, targeted schema additions for model capability persistence and resource parameter storage are permitted. The absolute no-schema boundary from earlier presentation-only rounds is lifted for these specific needs."

### 4.2 Model Capability Storage Decision (doc 11)

Select Option B and document the fields:
```sql
ALTER TABLE model_artifacts ADD COLUMN capabilities_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_artifacts ADD COLUMN capability_sources_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE model_artifacts ADD COLUMN default_test_mode TEXT NOT NULL DEFAULT 'auto';
```

### 4.3 Resource Parameter → Existing Field Mapping (doc 09)

Add the mapping table from section 3.3 above. Phase 1 implementers need to know where each parameter goes without guessing.

### 4.4 MV-008 Diagnosis Phase (doc 10)

Split MV-008 into MV-008a (diagnosis — Phase 1) and MV-008b (fix — Phase 2+ if root cause requires non-trivial changes). Diagnosis output is a Phase 1 deliverable regardless of whether fix completes.

### 4.5 Remove Test-Diagnostics Route Cleanup Detail (doc 10/11)

Clarify: removing the "诊断与测试" menu item does NOT delete the route or page file. The route `/models/test-diagnostics` should remain accessible via direct URL. Only the sidebar menu entry is removed. This prevents broken bookmarks.

## 5. Suggested Improvements

### 5.1 Add GPU Count / Accelerator Count Display

The deployment page should display accelerator count (e.g., "RTX 5090 × 1") in addition to or instead of raw accelerator IDs. This is a Phase 1 UI polish item.

### 5.2 Add "First Deployment" Empty State

When deployments list is empty, show a guided "创建第一个部署" call-to-action instead of an empty table. This is a Phase 1 UX item.

### 5.3 ParameterDef Schema Should Be Documented

The existing `parameter_defs_json` on BackendVersion already supports parameter metadata (name, type, default, cli_name). The design documents should reference this as the source of truth for parameter definitions, not duplicate them.

### 5.4 Frontend Model Name Lookup Should Be Centralized

MV-007's fix ("不要在前端多个页面散落重复 lookup") should be implemented as a shared composable (`useModelNames`) or a frontend model-name cache, not copy-pasted across pages.

## 6. Phase 1 Scope Boundaries

### What Phase 1 MUST include

1. Remove duplicate test-diagnostics sidebar menu entry (keep route)
2. Deployment list shows model name (not UUID)
3. Qwen3 404 diagnosis (not necessarily fix)
4. Schema boundary declaration document update
5. Model capability storage decision documented

### What Phase 1 MAY include (if time permits)

1. Model name display as shared composable
2. Empty-state guidance on deployment page
3. Accelerator count in deployment display

### What MUST be deferred to Phase 2+

1. Model capability persistence implementation (schema change + API + UI)
2. Model edit page with capability editor
3. Resource & performance parameter entry
4. Any new API endpoints for resource policy
5. Playwright spec implementation
6. Multi-replica scheduling
7. Cross-node placement
8. Auto-failover / retry

## 7. Items to Defer Past Phase 5

1. Full multi-replica scheduling with spread/pack/affinity
2. Quota and priority scheduling
3. Auto-failover and cross-node retry
4. Complete Playwright UI E2E (Layer 2/3)
5. API Gateway / API Key infrastructure
6. Independent diagnostics dashboard (separate from instance detail)
7. Huawei vendor adapter implementation
8. MetaX real hardware validation

## 8. Schema/API Evolution Recommendations

### 8.1 Add (Phase 2)

```text
model_artifacts:
  + capabilities_json TEXT DEFAULT '[]'
  + capability_sources_json TEXT DEFAULT '{}'
  + default_test_mode TEXT DEFAULT 'auto'

model_artifacts PATCH:
  + accept capabilities_json, capability_sources_json, default_test_mode

model_instances/{id}/test:
  + return endpoint_probe with /v1/models, /health results
  + return diagnostic_hints for common failure patterns (404, ECONNREFUSED, empty /v1/models)
```

### 8.2 Do NOT Add (any phase until designed)

```text
- resource_policy as first-class column (use existing config_snapshot_json)
- placement_policy as first-class column (use existing placement_json)
- deployment summary DTO until performance data justifies it
- ModelCapability table (Option C — overengineered for current needs)
```

## 9. Test and Acceptance Recommendations

### 9.1 Phase 1 Acceptance Gates

Before Phase 1 closeout, these must all pass:
```bash
go test ./internal/server/api/...        # existing tests must not regress
go test ./internal/server/runplan/...     # RunPlan tests must not regress
go vet ./...                               # no new warnings
npm --prefix web test                     # all frontend tests pass
npm --prefix web run build                # production build succeeds
```

### 9.2 Phase 2+ Acceptance Gates

Add:
```bash
# New tests for capability persistence
go test ./internal/server/api/ -run "TestModelCapability" -v

# New tests for resource parameter mapping
go test ./internal/server/runplan/ -run "TestResourceParam" -v

# Frontend tests for model edit page
npm --prefix web test
```

### 9.3 Manual Verification Checklist (Phase 1)

```text
[ ] Sidebar: no duplicate "诊断与测试"
[ ] Deployment list: model name visible (not UUID)
[ ] Deployment list: no undefined/null/[object Object]
[ ] Qwen3 test: diagnostic info richer than "HTTP 404"
[ ] Navigation: all links work, no broken routes
[ ] i18n: no leaked keys (status.xxx, nav.xxx)
[ ] git status --short: clean
```

## 10. Implementation Readiness

### Recommendation: PROCEED to Phase 1 with revisions

The design is solid. After addressing the 5 must-fix items in section 4, Phase 1 can begin immediately. The P0 items (menu dedup, model name display, Qwen3 diagnosis) are well-scoped and low-risk.

### Pre-Implementation Lock Items

Before writing Phase 1 code, these must be confirmed:
1. Schema boundary declaration added to doc 11
2. Model capability storage decision (Option B) documented
3. Resource→field mapping added to doc 09
4. MV-008 split into diagnosis (Phase 1) and fix (Phase 2+)
5. Test-diagnostics route preservation confirmed in doc 10

## 11. Document Quality Assessment

| Document | Clarity | Completeness | Actionability |
|----------|---------|-------------|---------------|
| 09-resource-scheduling | High | High | Medium — needs field mapping |
| 10-manual-verification | High | High | High — P0/P1/P2 clear |
| 11-implementation-plan | Medium | Medium | Medium — needs schema decision |

## 12. Modified Files This Round

- `docs/reports/phase-3/web-ai-config-review/12-claude-design-review-feedback.md` — this document (NEW)

No code, schema, migration, or test changes in this round.
