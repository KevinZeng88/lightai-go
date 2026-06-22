# 36 — Unsupported Model Detectors (Batch 5) Closeout

> Status: FIXED
> Scope: Batch 5 = B2 — 7 recognized-but-unsupported model type detectors
> Baseline: commit `143021a`
> Date: 2026-06-23

## 1. Batch 5 Scope

Add 7 detectors for model types that can be recognized but cannot currently run:
- ONNX
- TensorRT / TensorRT-LLM Engine
- OpenVINO
- Diffusers / Image Generation
- ASR (Whisper/FunASR/Paraformer/SenseVoice)
- TTS (CosyVoice/ChatTTS/GPT-SoVITS/Fish-Speech/Bark)
- Classification (Sequence/Token/Image/Audio)

## 2. Detector Summary

| Detector | Format | Task | Deployable | Unsupported Reason |
|----------|--------|------|------------|-------------------|
| DetectONNX | onnx | unknown | false | 当前平台尚未配置 ONNX Runtime 后端。 |
| DetectTensorRT | tensorrt_engine | unknown | false | 当前平台尚未配置 TensorRT-LLM 后端。 |
| DetectOpenVINO | openvino | unknown | false | 当前平台尚未配置 OpenVINO 后端。 |
| DetectDiffusers | diffusers | image_generation | false | 当前平台尚未配置 Diffusers/Image Generation 后端。 |
| DetectASR | huggingface | asr | false | 当前平台尚未配置 ASR 后端。 |
| DetectTTS | huggingface | tts | false | 当前平台尚未配置 TTS 后端。 |
| DetectClassification | huggingface | classification | false | 当前平台尚未配置分类模型服务后端。 |

## 3. Plugin Registry (Updated Priority)

```
LoRA → SentenceTransformers → Reranker → VLM → Diffusers → ASR → TTS →
Classification → HF Chat → OpenVINO → TensorRT → ONNX → GGUF
```

## 4. Excluded

- No new runtime backends
- No ONNX Runtime, TensorRT-LLM, OpenVINO, Diffusers, ASR, TTS implementations
- No production runtime smoke
- No schema changes
- No new migrations
- No backward compatibility fallback

## 5. Batch 1-4 Regression

| Gate | Status |
|------|--------|
| HF Chat detector → chat | ✅ |
| GGUF detector → .gguf file | ✅ |
| Embedding detector → embedding | ✅ |
| Reranker detector → rerank | ✅ |
| VLM detector → vision_chat | ✅ |
| LoRA detector → deployable=false | ✅ |
| VLM / InternVLChatModel → blocked by preflight | ✅ |
| GGUF RunPlan -m → .gguf file | ✅ |
| deployable=false → preflight blocks | ✅ |

## 6. Test Results

```bash
gofmt -w cmd/ internal/                                       → CLEAN
go test lightai-go/internal/agent/collector/...                → ALL PASS (33 tests: 20 existing + 13 B2)
go test lightai-go/internal/server/runplan/...                 → ALL PASS (20 tests: 16 existing + 4 B2 compat)
go test lightai-go/internal/server/api/...                     → ALL PASS
go vet ./...                                                   → CLEAN
npm test                                                       → ALL PASS
npm run build                                                  → ✓ built
git diff --check                                               → CLEAN
```

New tests:
- TestDetectDiffusers, TestDetectASR, TestDetectTTS, TestDetectClassification
- TestDetectOpenVINO, TestDetectTensorRT, TestDetectONNX
- TestB2UnsupportedDeployableFalse (4 sub-tests)
- TestCompatDeployableFalseFailsOnUnsupportedTypes (4 sub-tests)

## 7. Modified Files

| File | Change |
|------|--------|
| `internal/agent/collector/model_scanner.go` | 7 new plugin defaults, 7 new detectors, updated registry + de-dup |
| `internal/agent/collector/model_scanner_test.go` | 11 new B2 tests |
| `internal/server/runplan/compat_test.go` | 4 new deployable=false compat tests |
| `web/src/locales/zh-CN.ts` | New task/format/capability keys |
| `web/src/locales/en-US.ts` | New capability keys |
| `docs/.../36-unsupported-model-detectors-batch5-closeout.md` | This closeout |

## 8. Final Status

PASS — 7 unsupported detectors added. All have deployable=false, clear unsupported_reason, preflight blocks deployment.
