# 17 — Phase 2 Model Capability Persistence Closeout

> Status: FIXED
> Scope: Phase 2 — model capability persistence + model edit page
> Date: 2026-06-22
> Baseline: commit `212791b` (frontend test suite restored)

## 1. Phase 2 Scope Verification

### MUST (all completed)

| Item | Status | Details |
|------|--------|---------|
| Model capabilities persisted in `model_artifacts` | ✅ FIXED | `capabilities_json`, `capability_sources_json`, `default_test_mode` columns added |
| Model edit page supports editing capabilities | ✅ FIXED | Checkbox group for capabilities, select for default test mode |
| Detail/edit page field alignment | ✅ FIXED | Both show capabilities, sources, test mode; scan facts read-only in edit |
| Test entry reads persisted capabilities | ✅ FIXED | `recommendedTestMode()` prefers `default_test_mode`, then persisted caps |
| Qwen3-0.6B-Instruct-2512 can be set to Chat | ✅ FIXED | Edit page allows setting `capabilities: ["chat"]`, `default_test_mode: "chat"` |
| Docs and closeout | ✅ FIXED | Plan doc 16, closeout doc 17 |
| Tests, commit, push, clean `git status` | ✅ FIXED | See verification below |

### MUST NOT (none violated)

- No resource parameter editor ✅
- No `gpu_memory_utilization` configuration UI ✅
- No multi-replica scheduling ✅
- No cross-node scheduling ✅
- No auto failover/retry ✅
- No Playwright specs ✅
- No API Gateway / API Key ✅
- No refactor of entire model deployment flow ✅
- No legacy compatibility hacks ✅

## 2. Schema / Migration Changes

### New migration: V25

```sql
ALTER TABLE model_artifacts ADD COLUMN capabilities_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_artifacts ADD COLUMN capability_sources_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE model_artifacts ADD COLUMN default_test_mode TEXT NOT NULL DEFAULT 'auto';
```

### V3 CREATE TABLE updated

Fresh databases get the columns in the initial `model_artifacts` CREATE TABLE.

### Capability enums supported

```text
chat, completion, embedding, rerank, vision, tool_calling, structured_output
```

### Capability source enums supported

```text
scan, inferred, user_override, backend_probe
```

### Default test mode enums

```text
auto, chat, completion, embedding, rerank
```

## 3. API Changes

### GET `/api/v1/model-artifacts` (list) and `/{id}` (detail)

New response fields:
- `capabilities`: JSON array of capability strings
- `capability_sources`: JSON object mapping capability → source
- `default_test_mode`: string, default `"auto"`

### PATCH `/api/v1/model-artifacts/{id}`

New accepted fields with validation:
- `capabilities` — validates against allowed enum
- `capability_sources` — auto-normalizes user-provided caps to `user_override`
- `default_test_mode` — validates against allowed enum

### POST `/api/v1/model-artifacts` (create) and `/discover`

New columns initialized with defaults: `'[]'`, `'{}'`, `'auto'`.

## 4. Backend Implementation Details

### Files modified

| File | Change |
|------|--------|
| `internal/server/db/db.go` | V3 CREATE TABLE updated with 3 new columns; V25 migration added |
| `internal/server/api/artifact_handlers.go` | Validation helpers, updated SELECT/INSERT/PATCH queries, new JSON field handling |

### Files created

| File | Change |
|------|--------|
| `internal/server/api/model_capability_test.go` | 4 new tests for capability persistence, validation |

## 5. Frontend Implementation Details

### Files modified

| File | Change |
|------|--------|
| `web/src/api/models.ts` | Added `capabilities`, `capability_sources`, `default_test_mode` to `ModelArtifact` interface |
| `web/src/utils/modelCapabilities.js` | `inferModelCapabilities()` prefers persisted capabilities; `recommendedTestMode()` prefers `default_test_mode` |
| `web/src/pages/ModelArtifactsPage.vue` | Edit dialog: capability checkboxes, default_test_mode select, scan facts read-only; Detail drawer: shows persistent caps, sources, configured test mode |
| `web/src/locales/zh-CN.ts` | 19 new i18n keys for capability labels, sources, test modes, edit sections |
| `web/src/locales/en-US.ts` | 19 new i18n keys for capability labels, sources, test modes, edit sections |

### Files created

| File | Change |
|------|--------|
| `web/tests/modelCapabilities.test.mjs` | 7 new test cases for persistence features |

## 6. Model Edit Page — Editable Fields

- **Display Name** — text input
- **Path** — text input
- **Format** — select
- **Quantization** — select
- **Capabilities** — checkbox group (chat, completion, embedding, rerank, vision, tool_calling, structured_output)
- **Default Test Mode** — select (auto, chat, completion, embedding, rerank)

## 7. Model Edit Page — Read-only Scan Facts

- Size label
- Architecture
- Context length
- Task type

## 8. Model Detail Page — Capability Display

- Capability name with `el-tag`
- Source: scan/inferred/user_override/backend_probe
- Confidence level
- Reason
- "Persisted" badge when capabilities come from DB
- "Inferred" warning banner only when no persisted capabilities exist
- Configured test mode shown separately from recommended test mode

## 9. Test Entry — Persistent Capability Usage

- `recommendedTestMode(model)` checks `model.default_test_mode` first
- If not `"auto"`, uses that mode directly
- If `"auto"` or missing, falls back to capability inference
- `inferModelCapabilities(model)` checks `model.capabilities` array first
- Only falls back to name/metadata inference for legacy artifacts without persisted caps

## 10. Qwen3 Chat Configuration

1. Edit the Qwen3-0.6B-Instruct-2512 artifact
2. Check "对话 (Chat)" capability
3. Set default test mode to "Chat Completion"
4. Save
5. Refresh page — capabilities and test mode persist
6. Test entry defaults to Chat Completion mode

## 11. Items NOT Done (Deferred to Phase 3+)

- Resource parameter editor (gpu_memory_utilization, etc.)
- Multi-replica scheduling
- Cross-node scheduling
- Auto failover/retry
- Playwright specs
- API Gateway / API Key

## 12. Test Results

```bash
# Go backend
go test lightai-go/internal/server/api/...    → ALL PASS (including 4 new capability tests)
go test lightai-go/internal/server/runplan/... → ALL PASS
go vet ./...                                    → CLEAN

# Frontend
npm test                                        → ALL PASS (18 modelCapabilities tests + 59 total)
npm run build                                   → ✓ built
npx vue-tsc --noEmit                            → CLEAN

# Formatting
gofmt -w cmd/ internal/                         → CLEAN
git diff --check                                 → CLEAN
```

## 13. Verification Commands

```bash
# Verify schema
sqlite3 server.db ".schema model_artifacts" | grep -E "capabilities_json|capability_sources_json|default_test_mode"

# Check migration applied
sqlite3 server.db "SELECT * FROM schema_version WHERE version >= 25"

# API test
curl -s http://localhost:18080/api/v1/model-artifacts | jq '.[0] | {capabilities, capability_sources, default_test_mode}'
```

## 14. Open Issues

None. All Phase 2 MUST items completed.

## 15. Modified Files Summary

| File | Type |
|------|------|
| `internal/server/db/db.go` | Schema: V3 update + V25 migration |
| `internal/server/api/artifact_handlers.go` | Backend: validation, CRUD, JSON handling |
| `internal/server/api/model_capability_test.go` | Tests: 4 new Go tests |
| `web/src/api/models.ts` | Frontend: TypeScript interface |
| `web/src/utils/modelCapabilities.js` | Frontend: persistence-aware inference |
| `web/src/pages/ModelArtifactsPage.vue` | Frontend: edit page + detail drawer |
| `web/src/locales/zh-CN.ts` | Frontend: i18n (19 new keys) |
| `web/src/locales/en-US.ts` | Frontend: i18n (19 new keys) |
| `web/tests/modelCapabilities.test.mjs` | Tests: 7 new test cases |
| `docs/reports/phase-3/web-ai-config-review/16-phase-2-model-capability-plan.md` | Docs: plan |
| `docs/reports/phase-3/web-ai-config-review/17-phase-2-model-capability-closeout.md` | Docs: closeout |

## 16. Final Status

PASS — all Phase 2 items completed, all tests pass, git status clean.
