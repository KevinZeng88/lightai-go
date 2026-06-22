# 31 — Detection Core Batch 1 (A+B1) Closeout

> Status: FIXED
> Scope: Batch 1 = Phase A (Candidate contract) + Phase B1 (Core deployable model type plugins)
> Baseline: commit `a9fc3a6` (plan V4)
> Date: 2026-06-23

## 1. Batch 1 Scope

Completed Detection Core:
1. Expanded ScanCandidate struct with 11 new fields
2. Introduced ModelTypePlugin / DetectorFunc / FileFacts abstraction
3. Migrated existing HF/GGUF detection into plugins without regression
4. Added 4 new core detectors: SentenceTransformers/Embedding, Reranker/CrossEncoder, Vision-Language, LoRA/Adapter
5. Scan API returns complete candidate with all new fields
6. Wizard save preserves task/capabilities/default_test_mode/path_type/scanner metadata

## 2. ScanCandidate New Fields

| Field | Type | Example |
|-------|------|---------|
| Kind | string | "directory" / "file" / "adapter" |
| Format | string | "huggingface" / "sentence_transformers" / "gguf" / "lora_adapter" |
| Task | string | "chat" / "embedding" / "rerank" / "vision_chat" / "adapter" |
| Capabilities | []string | ["chat","completion"] / ["embedding"] |
| DefaultTestMode | string | "chat" / "embedding" / "rerank" / "auto" |
| Deployable | bool | true / false |
| RequiresBaseModel | bool | false except LoRA |
| RecommendedBackends | []string | ["vllm","sglang"] / ["llamacpp"] |
| Confidence | string | "high" / "medium" / "low" |
| Evidence | []string | ["config.json","tokenizer_config.json"] |
| UnsupportedReason | string | "" or LoRA message |

## 3. ModelTypePlugin / FileFacts Abstraction

**ModelTypePlugin** = lightweight struct: `ID + Detect (DetectorFunc) + Defaults (ModelTypeDefaults)`

**FileFacts** = pre-collected filesystem facts passed to all detectors. Collects JSON parsing, file globs, and presence checks once per scanned directory.

**Plugin registry** (priority-ordered):
```
LoRA → SentenceTransformers → Reranker → Vision-Language → HF Chat → GGUF
```

## 4. Detector Results

### HF Chat / Completion (existing, migrated to plugin)
- Detection: config.json present
- Defaults: task=chat, capabilities=["chat","completion"], deployable=true, recommended=["vllm","sglang"]

### GGUF file (existing, migrated to plugin)
- Detection: *.gguf glob
- Defaults: task=chat, capabilities=["chat","completion"], deployable=true, recommended=["llamacpp"]

### SentenceTransformers / Embedding (new)
- Detection: modules.json + 1_Pooling/ or config_sentence_transformers.json or name patterns
- Defaults: task=embedding, capabilities=["embedding"], default_test_mode="embedding"
- Test model: `/home/kzeng/models/bge-small-zh-v1.5`

### Reranker / CrossEncoder (new)
- Detection: name contains reranker/cross-encoder + config.json
- Defaults: task=rerank, capabilities=["rerank"], default_test_mode="rerank"
- Test model: `/home/kzeng/models/bge-reranker-base`

### Vision-Language / Multimodal (new)
- Detection: VL name patterns or VL indicators (configuration_internvl_chat.py, etc.)
- Defaults: task=vision_chat, capabilities=["chat","vision"], default_test_mode="chat"
- Test model: `/home/kzeng/models/InternVL2_5-1B`

### LoRA / Adapter (new)
- Detection: adapter_config.json or adapter_model.safetensors
- Defaults: deployable=false, requires_base_model=true, unsupported_reason set

## 5. Regression Verification

| Gate | Status |
|------|--------|
| R1: Direct GGUF file selection RunPlan -m points to .gguf | ✅ PASS |
| R2: Directory scan GGUF selection RunPlan -m points to selected .gguf | ✅ PASS |
| R3: HF directory path mode is directory | ✅ PASS |
| Embedding scan not identified as Chat | ✅ PASS |
| Reranker scan not identified as Chat | ✅ PASS |
| VLM scan preserves vision capability | ✅ PASS (capabilities=["chat","vision"]) |
| LoRA scan deployable=false | ✅ PASS |
| R8: All tests pass | ✅ PASS |

## 6. Excluded from Batch 1

- No detail/edit page changes (Phase C)
- No backend compatibility checker (Phase D)
- No preflight compatibility blocking (Phase D)
- No embedding/rerank test endpoints (Phase E)
- No production smoke (Phase P)
- No B2 unsupported detectors (ONNX/TensorRT/OpenVINO/Diffusers/ASR/TTS/Classification)
- No schema changes
- No new migrations
- No backward compatibility fallbacks

## 7. Test Results

```bash
gofmt -w cmd/ internal/                                       → CLEAN
go test lightai-go/internal/agent/collector/...                 → ALL PASS (9 new detector tests + 17 existing)
go test lightai-go/internal/server/api/...                      → ALL PASS
go test lightai-go/internal/server/runplan/...                   → ALL PASS
go vet ./...                                                    → CLEAN
npm test                                                        → ALL PASS
npm run build                                                   → ✓ built
git diff --check                                                → CLEAN
```

New tests:
- TestDetectHuggingFaceChat
- TestDetectGGUFFile
- TestDetectSentenceTransformers
- TestDetectReranker
- TestDetectVisionLanguage
- TestDetectLoRAAdapter
- TestEmptyDirectory
- TestMixedHFAndGGUF
- TestScanDirectoryFullPipeline

## 8. Modified Files

| File | Change |
|------|--------|
| `internal/agent/collector/model_scanner.go` | Full rewrite: expanded ScanCandidate, ModelTypePlugin/FileFacts, 6 detectors, plugin registry, strsToInterfaces |
| `internal/agent/collector/model_scanner_test.go` | New: 9 detector unit tests |
| `web/src/pages/ModelArtifactsPage.vue` | Wizard save: pass task/capabilities/default_test_mode from candidate; enrich discovered_metadata_json |

## 9. Final Status

PASS — Batch 1 (A+B1) complete. Detection Core delivers 6 model type plugins with full candidate fields. Ready for Phase C (Persistence + UI).
