# 32 — Detection Core Batch 1 Post-Implementation Review

> Status: REVIEW_COMPLETE (1 fix applied)
> Scope: Review of commit `b0456f6` + fix `(pending)`
> Date: 2026-06-23

## 1. Commit Scope Audit

Commit `b0456f6` files changed (5):

| File | Type | Scope check |
|------|------|-------------|
| `internal/agent/collector/model_scanner.go` | Scanner rewrite | ✅ A+B1 (Detection Core) |
| `internal/agent/collector/model_scanner_test.go` | New tests | ✅ A+B1 |
| `web/src/pages/ModelArtifactsPage.vue` | Wizard save fields | ✅ A+B1 field passthrough |
| `docs/.../31-detection-core-batch1-closeout.md` | Closeout doc | ✅ Documentation |
| `docs/.../28-model-detection-runtime-plugin-design.md` | Design doc (pre-existing, untracked → tracked) | ✅ Documentation |

**Scope creep check**:
- No schema changes ✅
- No migration ✅
- No Phase C (detail/edit page changes) ✅
- No Phase D (compatibility checker, preflight blocking) ✅
- No Phase E (embedding/rerank test endpoints) ✅
- No Phase P (production runtime smoke) ✅
- No B2 detectors (ONNX/TensorRT/OpenVINO/Diffusers/ASR/TTS/Classification) ✅
- No backward compatibility fallback ✅
- No i18n key additions ✅

## 2. ScanCandidate Field Count Correction

Closeout reported "11 new fields". **Correction: 10 new fields.**

Original ScanCandidate had: `Path, PathType, Format, DetectedMetadata, Warnings, AutoSelected, SelectionReason, SizeBytes, SizeLabel` (9 fields).

New fields added (10):
```go
Kind               string   // 1
Task               string   // 2
Capabilities       []string // 3
DefaultTestMode    string   // 4
Deployable         bool     // 5
RequiresBaseModel  bool     // 6
RecommendedBackends []string // 7
Confidence         string   // 8
Evidence           []string // 9
UnsupportedReason  string   // 10
```

`Format` was already present in the original struct (enriched with plugin defaults, not a new field).

## 3. Real Model Scan Verification

All scans run against actual disk paths at `/home/kzeng/models/`:

### HF Chat (`Qwen3-0.6B-Instruct-2512`)
```json
{"format":"huggingface","task":"chat","capabilities":["chat","completion"],"default_test_mode":"chat","deployable":true,"recommended_backends":["vllm","sglang"],"path_type":"directory","kind":"directory","auto_selected":true}
```
✅ Single candidate, correct task/caps/deployable/recommended

### GGUF (`Qwen3.5-9B-Q4`)
```json
{"format":"gguf","task":"chat","capabilities":["chat","completion"],"default_test_mode":"chat","deployable":true,"recommended_backends":["llamacpp"],"path_type":"file","kind":"file","path":".../Qwen3.5-9B-Q4_K_M.gguf","auto_selected":true}
```
✅ Single candidate, path points to concrete .gguf file, not directory

### Embedding (`bge-small-zh-v1.5`)
```json
{"format":"sentence_transformers","task":"embedding","capabilities":["embedding"],"default_test_mode":"embedding","deployable":true,"recommended_backends":["vllm","sglang"],"path_type":"directory"}
```
✅ Single candidate (HF Chat suppressed), correctly identified as embedding, NOT chat

### Reranker (`bge-reranker-base`)
```json
{"format":"huggingface","task":"rerank","capabilities":["rerank"],"default_test_mode":"rerank","deployable":true,"recommended_backends":["vllm","sglang"],"path_type":"directory"}
```
✅ Single candidate (ST + HF Chat suppressed), correctly identified as reranker, NOT chat

### VLM (`InternVL2_5-1B`)
```json
{"format":"huggingface","task":"vision_chat","capabilities":["chat","vision"],"default_test_mode":"chat","deployable":true,"recommended_backends":["vllm","sglang"],"path_type":"directory"}
```
✅ Single candidate (HF Chat suppressed), vision capability preserved, NOT downgraded to plain chat

### GGUF Direct File
```json
{"format":"gguf","task":"chat","path_type":"file","deployable":true,"auto_selected":true}
```
✅ Direct file selection works correctly

## 4. Issues Found and Fixed

### Issue 1: Duplicate candidates from multi-plugin matching

**Symptom**: ST/Reranker/VLM plugins produced candidates, but HF Chat also fired because `config.json` exists. Example: bge-reranker-base returned 3 candidates (ST=embedding, Reranker=rerank, HF=chat).

**Fix**: Added de-duplication after plugin collection. When a more specific plugin matches (sentence_transformers, lora_adapter, rerank task, vision_chat task), suppress the generic HF Chat candidate (format=huggingface, task=chat).

### Issue 2: ST detector false positive on reranker names

**Symptom**: bge-reranker-base matched the ST detector because "bge" is in both ST and reranker keyword lists. ST detector fired, creating a duplicate "embedding" candidate.

**Fix**: Added exclusion check in ST detector — if directory name matches reranker keywords, disable ST name-based detection.

### After fixes: All 5 model types return exactly 1 correct candidate.

## 5. Wizard Save Field Preservation

Fields passed from candidate to create API in `doWizardSave`:

| Field | Preserved to ModelArtifact | Preserved to discovered_metadata_json |
|-------|---------------------------|--------------------------------------|
| task (as task_type) | ✅ | ✅ |
| capabilities | ✅ | — |
| default_test_mode | ✅ | — |
| format | ✅ | — |
| path_type | ✅ | ✅ |
| kind | — | ✅ |
| deployable | — | ✅ |
| requires_base_model | — | ✅ |
| recommended_backends | — | ✅ |
| confidence | — | ✅ |
| evidence | — | ✅ |
| unsupported_reason | — | ✅ |

Fields like `kind`, `evidence`, `confidence` are stored in `discovered_metadata_json` on ModelLocation (the correct authority per the plan). They are NOT lost — they are available for Phase C to display in the UI.

## 6. Regression Verification

| Gate | Status |
|------|--------|
| R1: Direct GGUF -m → .gguf file | ✅ |
| R2: Directory scan GGUF -m → .gguf file | ✅ |
| R3: HF directory path mode | ✅ |
| Embedding ≠ Chat | ✅ |
| Reranker ≠ Chat | ✅ |
| VLM preserves vision | ✅ |
| LoRA deployable=false | ✅ |
| R8: All tests pass | ✅ |

## 7. Test Results (Post-fix)

```bash
gofmt -w cmd/ internal/                                       → CLEAN
go test lightai-go/internal/agent/collector/...                → ALL PASS (26 tests)
go test lightai-go/internal/agent/...                          → ALL PASS
go test lightai-go/internal/server/api/...                     → ALL PASS
go test lightai-go/internal/server/runplan/...                 → ALL PASS
go vet ./...                                                   → CLEAN
npm test                                                       → ALL PASS
npm run build                                                  → ✓ built
git diff --check                                               → CLEAN
```

## 8. Final Status

REVIEW_COMPLETE — 1 fix applied (duplicate candidate suppression + ST reranker exclusion). Batch 1 scope boundaries confirmed clean. Ready for Phase C.
