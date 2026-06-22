# 34 — Runtime Compatibility + Test Methods (Batch 3) Closeout

> Status: FIXED
> Scope: Batch 3 = Phase D (Backend Compatibility) + Phase E (Test Method Abstraction)
> Baseline: commit `9885a28`
> Date: 2026-06-23

## 1. Batch 3 Scope

Phase D+E — enforce runtime compatibility at preflight and support embedding/rerank test modes. No production smoke, no B2 detectors.

## 2. Backend Capability Contract (Phase D)

### backend_versions.capabilities_json Structure

```json
{
  "supported_formats": ["huggingface", "sentence_transformers"],
  "supported_tasks": ["chat", "completion", "embedding", "rerank", "vision_chat"],
  "supported_capabilities": ["chat", "completion", "embedding", "rerank", "vision"],
  "model_path_modes": ["directory"],
  "test_endpoints": {
    "chat": "/v1/chat/completions",
    "completion": "/v1/completions",
    "embedding": "/v1/embeddings",
    "rerank": "/v1/rerank"
  }
}
```

### Backends Configured

| Backend | supported_formats | supported_tasks | model_path_modes | test_endpoints |
|---------|-------------------|-----------------|------------------|----------------|
| vLLM (v0.23.0) | huggingface, sentence_transformers | chat, completion, embedding, rerank, vision_chat | directory | chat, completion, embedding, rerank |
| SGLang (v0.5.13/v0.5.12) | huggingface, sentence_transformers | chat, completion, embedding, rerank, vision_chat | directory | chat, completion, embedding, rerank (/rerank) |
| llama.cpp (b9700) | gguf | chat, completion | file | chat, completion |

### V27 Migration

Force-updates `backend_versions.capabilities_json` for all built-in versions. Seed data also updated. Both handle the case where `INSERT OR IGNORE` leaves existing rows with empty capabilities.

## 3. CompatibilityChecker (Phase D)

New file: `internal/server/runplan/compat.go`

Check order: backend capability declared → deployable → format → path_mode → task

| Combination | Result | Error Code |
|-------------|--------|-----------|
| vLLM + GGUF | BLOCK | format_mismatch: "模型为 GGUF 文件，vLLM/SGLang 不支持。" |
| SGLang + GGUF | BLOCK | format_mismatch |
| llama.cpp + HF | BLOCK | format_mismatch: "模型为 HuggingFace 目录，llama.cpp 不支持。" |
| llama.cpp + Embedding | BLOCK | format_mismatch |
| llama.cpp + Reranker | BLOCK | format_mismatch |
| LoRA standalone | BLOCK | not_deployable |
| deployable=false | BLOCK | not_deployable |
| Missing backend caps | BLOCK | backend_capability_missing |
| vLLM + HF Chat | PASS | ok |
| vLLM + Embedding | PASS | ok |
| vLLM + Reranker | PASS | ok |
| vLLM + VLM | PASS | ok |
| llama.cpp + GGUF | PASS | ok |

All 14 compat tests pass.

### Preflight Integration

Compatibility check runs in `preflightDeployment()` after model location validation, before RunPlan resolution. On failure: structured preflight error blocks deployment.

## 4. Test Method Abstraction (Phase E)

### Frontend

Test mode selector: `auto | chat | completion | embedding | rerank`

### Backend

New functions: `tryEmbeddingInference()` → `/v1/embeddings`, `tryRerankInference()` → `/v1/rerank`

Dispatch: mode=chat→chat endpoint, mode=embedding→embedding endpoint, mode=rerank→rerank endpoint, mode=auto→chat then completion (embedding/rerank are explicit only)

### Endpoint Selection

Uses backend-declared endpoints from `capabilities_json.test_endpoints`. No blind probing. Undeclared endpoint → clear diagnostic.

## 5. Blocking Rules (All BLOCK, Never Warn)

- format mismatch → BLOCK
- path mode mismatch → BLOCK
- deployable=false → BLOCK
- task mismatch → BLOCK
- backend capability missing → BLOCK

No backward compatibility fallbacks.

## 6. Excluded from Batch 3

- No production smoke (Phase P)
- No B2 unsupported detectors
- No schema changes (V27 is data repair only)
- No new columns
- No backward compatibility fallback
- No endpoint blind probing
- No image upload VL test UI

## 7. Test Results

```bash
go test ./internal/server/runplan/...    → ALL PASS (14 compat + existing)
go test ./internal/agent/collector/...    → ALL PASS
go test ./internal/server/api/...         → ALL PASS
go vet ./...                              → CLEAN
npm test                                  → ALL PASS
npm run build                             → ✓ built
git diff --check                          → CLEAN
```

## 8. Modified Files

| File | Change |
|------|--------|
| `internal/server/runplan/compat.go` | New: CompatibilityChecker + ParseBackendCapabilities |
| `internal/server/runplan/compat_test.go` | New: 14 compat tests |
| `internal/server/db/db.go` | V27 migration, seed capabilities_json structured data, seed struct fix |
| `internal/server/api/deployment_lifecycle_handlers.go` | Preflight compat check + embedding/rerank test dispatch + inference functions |
| `internal/server/api/phase3_rbac_test.go` | Use default vllm version (compat) |
| `internal/server/api/ui_persistence_runplan_test.go` | Test artifact format fix (gguf→huggingface) |
| `internal/server/api/workflow_deployment_runplan_test.go` | Prefer vllm runtime for HF model |
| `web/src/pages/ModelInstancesPage.vue` | Test mode selector: +embedding +rerank |
| `web/src/locales/zh-CN.ts` | testMode_embedding, compat i18n keys |
| `web/src/locales/en-US.ts` | testMode_embedding, compat i18n keys |

## 9. Final Status

PASS — Batch 3 (D+E) complete. Compatibility enforced at preflight. Embedding/rerank test modes implemented. Ready for Phase P.
