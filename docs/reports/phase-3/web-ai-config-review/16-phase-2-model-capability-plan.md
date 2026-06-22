# 16 — Phase 2 Model Capability Persistence and Model Edit Page Plan

> Status: DRAFT
> Scope: Phase 2 — model capability persistence + model edit page
> Date: 2026-06-22
> Baseline: commit `212791b` (frontend test suite restored)

## 1. Phase 2 Scope

### MUST

1. Model capabilities persisted in `model_artifacts` table (Option B).
2. Model edit page supports editing configurable model info.
3. Model detail and edit page fields are aligned.
4. Test entry reads persisted capabilities and `default_test_mode`.
5. Qwen3-0.6B-Instruct-2512 can be manually set to Chat and persists after refresh.
6. Update docs and closeout.
7. Tests, commit, push, clean `git status`.

### MUST NOT

1. No resource parameter editor.
2. No `gpu_memory_utilization` configuration UI.
3. No multi-replica scheduling.
4. No cross-node scheduling.
5. No auto failover/retry.
6. No Playwright specs.
7. No API Gateway / API Key.
8. No refactor of entire model deployment flow.
9. No legacy compatibility hacks.

## 2. Why Only Model Capabilities This Round (Not Resource Parameters)

Resource parameter editing (Phase 3) depends on:

- A clear mapping from backend-specific parameter names to UI fields
- Proper integration with RunPlan's `parameters_json` and `config_snapshot_json`
- The deployment edit page being restructured to show backend-specific sections

None of this work is blocked by model capability persistence. Capabilities are a model-level concern (editing model metadata); resource parameters are a deployment-level concern (editing runtime config). They are independent axes.

Phase 2 focuses entirely on making model capabilities real, persisted data so that the test entry point and future scheduling logic have a reliable source of truth about what a model can do.

## 3. Schema Change Plan

### Selected Option: Option B — New columns on `model_artifacts`

```sql
ALTER TABLE model_artifacts ADD COLUMN capabilities_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_artifacts ADD COLUMN capability_sources_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE model_artifacts ADD COLUMN default_test_mode TEXT NOT NULL DEFAULT 'auto';
```

### Rationale

- Option A (metadata JSON overloading): conflates scan metadata with user overrides.
- Option C (separate ModelCapability table): overengineered for a simple list + source map.
- Option B: minimal, clear, queryable via SQLite JSON functions, no joins needed.

### Capability Enums

```text
chat
completion
embedding
rerank
vision
tool_calling
structured_output
```

### Capability Source Enums

```text
scan          — detected during model file scan (GGUF metadata, HF config.json)
inferred      — inferred from model name, task_type, architecture patterns
user_override — manually set by user in edit page
backend_probe — detected by probing running backend (future)
```

### Default Test Mode Enums

```text
auto        — let the system decide based on capabilities
chat        — Chat Completion (/v1/chat/completions)
completion  — Text Completion (/v1/completions)
embedding   — Embedding (/v1/embeddings)
rerank      — Rerank (/v1/rerank)
```

### Migration Strategy

- Add a new migration V25 that ALTER TABLE model_artifacts.
- Use `INSERT OR IGNORE` for idempotency.
- Update the V3 table creation to include these columns for fresh databases.
- No historical data fallback — old artifacts get empty defaults.

## 4. API Change Plan

### 4.1 List API (`GET /api/v1/model-artifacts`)

Add to response per artifact:
- `capabilities_json` (parsed as JSON array)
- `capability_sources_json` (parsed as JSON object)
- `default_test_mode`

### 4.2 Detail API (`GET /api/v1/model-artifacts/{id}`)

Same additions as List API.

### 4.3 Update API (`PATCH /api/v1/model-artifacts/{id}`)

Accept new fields:
- `capabilities_json` — JSON array of capability strings
- `capability_sources_json` — JSON object mapping capability → source
- `default_test_mode` — string enum

Validation:
- `capabilities_json` values must be in allowed enum.
- `capability_sources_json` keys must be valid capabilities, values must be valid sources.
- On save, user-provided capabilities with `inferred`/`scan` source should be changed to `user_override`.
- `default_test_mode` must be in allowed enum.
- If `default_test_mode` is not `auto`, soft-check that the corresponding capability exists (warn but don't reject — documented in closeout).

### 4.4 Create/Discover API

Initialize:
- `capabilities_json = '[]'`
- `capability_sources_json = '{}'`
- `default_test_mode = 'auto'`

## 5. Scan/Import Capability Initialization

When scanning models:

1. If the scan result contains capability hints (e.g., chat_template in tokenizer_config), initialize capabilities with source `scan`.
2. If the model name/path suggests capabilities (e.g., "Instruct", "chat"), set source to `inferred`.
3. Otherwise, leave as empty defaults.
4. The existing frontend `inferModelCapabilities()` will be enhanced to also read `capabilities_json` as the primary source, falling back to inference for legacy artifacts.

## 6. Frontend Change Plan

### 6.1 Model Edit Page

Replace the current simple dialog with a proper edit page (could be a separate route or an expanded dialog) that includes:

**Editable:**
- Display name
- Description (if available)
- Tags (if available)
- Capabilities (checkbox group)
- Default test mode (select/radio)

**Read-only (scan facts):**
- File size
- Checksum
- Format
- Architecture
- Quantization
- Parameter count
- Context length
- Model path/location

### 6.2 Model Detail Page

1. Show persistent capabilities with source badges.
2. Show default test mode.
3. Remove or update the "this is inferred" warning banner.
4. Show scan facts as read-only.

### 6.3 Test Entry

1. Read `model.default_test_mode` from the API response.
2. If `auto`, fall back to `inferModelCapabilities()` for the recommendation.
3. If explicitly set (e.g., `chat`), use that mode directly.

### 6.4 Capability Source Display

| Source | Chinese Label | English Label |
|--------|-------------|---------------|
| `scan` | 自动扫描 | Auto Scan |
| `inferred` | 自动推断 | Inferred |
| `user_override` | 人工修正 | User Override |
| `backend_probe` | 后端探测 | Backend Probe |

## 7. Detail/Edit Field Alignment

| Field | Detail Page | Edit Page |
|-------|-----------|----------|
| Name (identifier) | Show | Read-only |
| Display Name | Show | Editable |
| Description | Show (if exists) | Editable |
| Tags | Show (if exists) | Editable |
| Capabilities | Show | Editable (checkboxes) |
| Default Test Mode | Show | Editable (select) |
| Format | Show | Read-only |
| Architecture | Show | Read-only |
| Size | Show | Read-only |
| Quantization | Show | Read-only |
| Parameter Count | Show | Read-only |
| Context Length | Show | Read-only |
| Path | Show | Read-only |
| File Size | Show | Read-only |
| Checksum | Show | Read-only |

## 8. Acceptance Criteria

1. `capabilities_json`, `capability_sources_json`, `default_test_mode` exist in `model_artifacts` table.
2. List and detail APIs return these fields.
3. PATCH API accepts and validates these fields.
4. New artifacts created via scan/wizard get default values.
5. Model edit page allows editing capabilities and default test mode.
6. Model detail page shows persistent capabilities and sources.
7. Test entry reads `default_test_mode` from artifact (falling back to inference for `auto`).
8. Qwen3-0.6B-Instruct-2512 can be set to Chat and persists after refresh.
9. All existing tests pass; new tests cover the new fields.
10. `git status` clean after commit.

## 9. Database Rebuild Notes

- New databases created from scratch get the columns via the updated V3 CREATE TABLE.
- Existing databases get the columns via V25 ALTER TABLE migration.
- If rebuilding, delete `server.db` and restart — migrations run from V1 to V25.
- No seed data for model_artifacts exists — all artifacts are user-created.

## 10. Items Deferred to Phase 3+

1. Resource parameter editor (gpu_memory_utilization, max_model_len, etc.)
2. Multi-replica scheduling
3. Cross-node scheduling
4. Auto failover/retry
5. Playwright spec implementation
6. API Gateway / API Key
7. Capability `backend_probe` source (requires running instance probe)
