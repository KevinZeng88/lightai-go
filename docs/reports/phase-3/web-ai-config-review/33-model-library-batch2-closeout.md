# 33 — Model Library Persistence and UI Display/Edit (Batch 2) Closeout

> Status: FIXED
> Scope: Batch 2 = Phase C — Model library persistence and UI display/edit
> Baseline: commit `6a313f7`
> Date: 2026-06-23

## 1. Batch 2 Scope

Phase C only — persist scanner metadata and display it in model detail/edit UI. No compatibility, no test endpoints, no production smoke.

## 2. Data Authority (Implemented)

### ModelArtifact — Model Semantic Authority

| Field | Source | Written by |
|-------|--------|-----------|
| format | Wizard candidate | HandleCreateArtifact |
| task_type | Wizard candidate + edit dialog | HandleCreateArtifact / HandlePatchArtifact |
| capabilities_json | Wizard candidate + edit checkboxes | HandlePatchArtifact |
| capability_sources_json | Auto-computed on edit | HandlePatchArtifact |
| default_test_mode | Wizard candidate + edit select | HandlePatchArtifact |

### ModelLocation — Location & Evidence Authority

Saved into `discovered_metadata_json`:
```json
{
  "kind": "directory",
  "deployable": true,
  "requires_base_model": false,
  "recommended_backends": ["vllm","sglang"],
  "confidence": "high",
  "evidence": ["modules.json","1_Pooling/config.json"],
  "unsupported_reason": "",
  "scan_root": "/home/kzeng/models/bge-small-zh-v1.5"
}
```

## 3. Backend Changes

- Added `allowedTaskTypes` map (chat, completion, embedding, rerank, vision_chat, adapter, unknown)
- `HandleCreateArtifact`: validates task_type before insert
- `HandlePatchArtifact`: validates task_type in PATCH fields loop
- task_type defaults to "chat" if not provided (backward compatible)

## 4. Frontend Detail Page

Added "扫描识别信息" (Scan Recognition) section showing:
- Storage kind (directory/file/adapter) with color-coded tag
- Deployable status (green/red)
- Unsupported reason (red text when not deployable)
- Requires base model (warning)
- Confidence
- Recommended backends (tags)
- Evidence (tags)
- Scan root

Also added task_type display in basic info with primary tag.

## 5. Frontend Edit Page

Added task_type editing:
- Select with 7 options: chat/completion/embedding/rerank/vision_chat/adapter/unknown
- Saved to backend on PATCH
- Refreshes correctly

## 6. i18n Keys Added (18 keys)

| Key | zh-CN | en-US |
|-----|-------|-------|
| task_chat | 对话 (Chat) | Chat |
| task_completion | 文本补全 (Completion) | Text Completion |
| task_embedding | 向量 (Embedding) | Embedding |
| task_rerank | 重排 (Reranker) | Reranker |
| task_visionChat | 视觉对话 (VLM) | Vision Chat (VLM) |
| task_adapter | 适配器 (Adapter) | Adapter |
| task_unknown | 未知 | Unknown |
| kind_directory | 目录 | Directory |
| kind_file | 文件 | File |
| kind_adapter | 适配器 | Adapter |
| scanRecognition | 扫描识别信息 | Scan Recognition |
| kind | 存储形态 | Storage Kind |
| deployable | 可独立部署 | Standalone Deployable |
| requiresBaseModel | 需要基础模型 | Requires Base Model |
| recommendedBackends | 推荐后端 | Recommended Backends |
| confidence | 识别置信度 | Confidence |
| evidence | 识别证据 | Evidence |
| scanRoot | 扫描根目录 | Scan Root |

No `task.xxx` / `format.xxx` / `capability.xxx` / `[object Object]` / `undefined` / `null` leaks.

## 7. Excluded from Batch 2

- No backend compatibility checker (Phase D)
- No preflight compatibility blocking (Phase D)
- No embedding/rerank test endpoints (Phase E)
- No production smoke (Phase P)
- No B2 unsupported detectors
- No schema changes
- No new migrations
- No backward compatibility fallback
- No RunPlan generation changes

## 8. Batch 1 Regression

All Batch 1 detectors still produce correct results:
- HF Chat → task=chat ✅
- GGUF → concrete .gguf file ✅
- Embedding → task=embedding (not chat) ✅
- Reranker → task=rerank (not chat) ✅
- VLM → task=vision_chat with vision capability ✅
- LoRA → deployable=false ✅

## 9. Test Results

```bash
gofmt -w cmd/ internal/                                       → CLEAN
go test lightai-go/internal/agent/collector/...                → ALL PASS
go test lightai-go/internal/server/api/...                     → ALL PASS
go test lightai-go/internal/server/runplan/...                 → ALL PASS
go vet ./...                                                   → CLEAN
npm test                                                       → ALL PASS (all i18n checks pass)
npm run build                                                  → ✓ built
git diff --check                                               → CLEAN
```

## 10. Modified Files

| File | Change |
|------|--------|
| `internal/server/api/artifact_handlers.go` | task_type validation, allowedTaskTypes map |
| `web/src/pages/ModelArtifactsPage.vue` | Scanner info section in detail, task_type editing, scanMeta computed, kindText/taskTypeText helpers |
| `web/src/locales/zh-CN.ts` | 18 new i18n keys |
| `web/src/locales/en-US.ts` | 18 new i18n keys |
| `docs/.../33-model-library-batch2-closeout.md` | This closeout |

## 11. Final Status

PASS — Batch 2 (Phase C) complete. Model library now persists and displays scanner metadata. Ready for Phase D.
