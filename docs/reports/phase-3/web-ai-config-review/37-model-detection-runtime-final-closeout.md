# 37 — Model Detection / Runtime Compatibility Final Closeout

> Status: **CLOSED_WITH_VLM_BLOCKER**
> Scope: Batch 1-6 complete — model detection, runtime compatibility, test modes, production E2E, unsupported detectors
> Date: 2026-06-23

## 1. Final Status

**CLOSED_WITH_VLM_BLOCKER** — NOT FULL_PASS.

5 production E2Es pass (real containers + endpoints). 1 VLM runtime blocked at preflight via backend capability declaration. 7 unsupported model types detected but non-deployable. All compatibility gates enforced.

## 2. Commit Timeline

| Batch | Commits | Scope |
|-------|---------|-------|
| 1 | `b0456f6`, `6a313f7` | Detection Core: 6 model type plugins, ScanCandidate enrichment |
| 2 | `9885a28` | Model Library: persistence + UI display/edit |
| 3 | `e2073c9` | Runtime Compatibility: backend capabilities, CompatibilityChecker, test modes |
| 4 | `5cb70cf`, `621e610`, `be56e10`, `143021a` | Production Full-Flow E2E: real containers, VLM triage |
| 5 | `40ca815` | B2: 7 unsupported model type detectors |
| 6 | (this commit) | Final closeout |

## 3. Batch Results

| Batch | Status | Key Deliverable |
|-------|--------|-----------------|
| 1 | FIXED | 6 detectors (HF Chat, GGUF, Embedding, Reranker, VLM, LoRA); 10-field ScanCandidate; plugin registry |
| 2 | FIXED | Model persistence to artifact/location; detail page scanner info; edit page task_type |
| 3 | FIXED | backend_versions.capabilities_json contract; CompatibilityChecker (format/path/deployable/task/architecture); embedding/rerank test modes |
| 4 | PARTIAL_PASS | 5/6 runtime E2E PASS (vLLM/SGLang/llama.cpp/embedding/reranker); 1 VLM BLOCKED (InternVL, preflight now blocks) |
| 5 | FIXED | 7 unsupported detectors (ONNX, TensorRT, OpenVINO, Diffusers, ASR, TTS, Classification); all deployable=false |
| 6 | CLOSED_WITH_VLM_BLOCKER | Final capability matrix, release-readiness recommendation |

## 4. Production Capability Matrix

| Capability | Status | Evidence |
|------------|--------|----------|
| HF Chat + vLLM (scan/create/preflight/RunPlan/container/endpoint/stop) | ✅ PASS | e2e-1-* (7 files) |
| HF Chat + SGLang (scan/create/preflight/RunPlan/container/endpoint/stop) | ✅ PASS | e2e-2-* (5 files) |
| GGUF + llama.cpp (scan/create/preflight/RunPlan/-m→.gguf/container/endpoint/stop) | ✅ PASS | e2e-3-* (7 files) |
| Embedding + vLLM (scan/create/preflight/container//v1/embeddings/stop) | ✅ PASS | e2e-4-* (5 files) |
| Reranker + vLLM (scan/create/preflight/container//v1/rerank/stop) | ✅ PASS | e2e-5-* (5 files) |
| VLM / InternVL2_5-1B (scan/create/preflight) | BACKEND_CAPABILITY_BLOCKED | VLM-RUNTIME-001; e2e-6-* (4 files) |
| Wrong combo blocking (vLLM+GGUF, llama.cpp+HF, LoRA alone, deployable=false, InternVL architecture) | ✅ PASS (20 tests) | e2e-7-*; compat_test.go |
| ONNX recognition | PASS_NON_DEPLOYABLE | model_scanner_test.go |
| TensorRT recognition | PASS_NON_DEPLOYABLE | model_scanner_test.go |
| OpenVINO recognition | PASS_NON_DEPLOYABLE | model_scanner_test.go |
| Diffusers recognition | PASS_NON_DEPLOYABLE | model_scanner_test.go |
| ASR recognition | PASS_NON_DEPLOYABLE | model_scanner_test.go |
| TTS recognition | PASS_NON_DEPLOYABLE | model_scanner_test.go |
| Classification recognition | PASS_NON_DEPLOYABLE | model_scanner_test.go |
| LoRA recognition | PASS_NON_DEPLOYABLE (detector test) | model_scanner_test.go |

## 5. VLM-RUNTIME-001 Blocker

| Field | Value |
|-------|-------|
| ID | VLM-RUNTIME-001 |
| Issue | InternVL2_5-1B blocked on vLLM/SGLang backend architecture support |
| Root Cause | vLLM v0.20.1 cannot load InternVLChatModel tokenizer; sentencepiece present, --trust-remote-code provided; same image loads Qwen3/bge-small/bge-reranker. True architecture incompatibility. |
| Current Mitigation | `blocked_architectures.InternVLChatModel` in vLLM/SGLang capabilities_json; CompatibilityChecker blocks pre-deployment |
| Future Unlock | Validated backend runtime/image supporting InternVL2.5 |
| Status | BACKEND_CAPABILITY_BLOCKED |

## 6. Model Type Detectors (Complete Registry)

| Detector | Task | Deployable | Priority |
|----------|------|------------|----------|
| DetectLoRAAdapter | adapter | false | 1 |
| DetectSentenceTransformers | embedding | true | 2 |
| DetectReranker | rerank | true | 3 |
| DetectVisionLanguage | vision_chat | true | 4 |
| DetectDiffusers | image_generation | false | 5 |
| DetectASR | asr | false | 6 |
| DetectTTS | tts | false | 7 |
| DetectClassification | classification | false | 8 |
| DetectHuggingFaceChat | chat | true | 9 |
| DetectOpenVINO | unknown | false | 10 |
| DetectTensorRT | unknown | false | 11 |
| DetectONNX | unknown | false | 12 |
| DetectGGUF | chat | true | 13 |

## 7. Backend Capability Contract

| Backend | Formats | Tasks | Path Mode | InternVL? |
|---------|---------|-------|-----------|-----------|
| vLLM (v0.23.0) | huggingface, sentence_transformers | chat, completion, embedding, rerank, (vision_chat blocked_arch) | directory | BLOCKED |
| SGLang (v0.5.13) | huggingface, sentence_transformers | chat, completion, embedding, rerank, (vision_chat blocked_arch) | directory | BLOCKED |
| llama.cpp (b9700) | gguf | chat, completion | file | N/A |

## 8. CompatibilityChecker Final Rules

Check order (all BLOCK, never warn):
1. Backend capability declared → missing → BLOCK
2. Deployable → false → BLOCK
3. Format → mismatch → BLOCK
4. Path mode → mismatch → BLOCK
5. Architecture → blocked → BLOCK
6. Task → mismatch → BLOCK

20 unit tests, all pass.

## 9. Test Method Abstraction

| Mode | Endpoint | Used By |
|------|----------|---------|
| chat | backend-declared (/v1/chat/completions) | HF Chat, GGUF Chat, VLM |
| completion | backend-declared (/v1/completions) | fallback |
| embedding | backend-declared (/v1/embeddings) | Embedding models |
| rerank | backend-declared (/v1/rerank, /rerank) | Reranker models |
| auto | chat→completion fallback | default |

## 10. Evidence Index

```
evidence/batch4-full-flow-e2e/
├── e2e-1-* (7 files)  — HF Chat + vLLM: scan, docker cmd, /v1/models, chat, logs, cleanup
├── e2e-2-* (5 files)  — HF Chat + SGLang: docker cmd, /v1/models, chat, logs, cleanup
├── e2e-3-* (7 files)  — GGUF + llama.cpp: scan, docker cmd, /v1/models, chat, logs, cleanup
├── e2e-4-* (5 files)  — Embedding + vLLM: scan, docker cmd, /v1/embeddings, logs, cleanup
├── e2e-5-* (5 files)  — Reranker + vLLM: scan, docker cmd, /v1/rerank, logs, cleanup
├── e2e-6-* (4 files)  — VLM blocker: scan, logs, blocker evidence, cleanup
└── e2e-7-* (1 file)   — Wrong combos: 11 compat results
```

## 11. Cannot Claim

- ❌ All MUST RUN types runtime PASS (VLM is BLOCKED)
- ❌ VLM runtime verified
- ❌ ONNX/TensorRT/OpenVINO/Diffusers/ASR/TTS/Classification serving
- ❌ LoRA standalone deployment
- ❌ Batch 4 FULL_PASS
- ❌ Production Full-Flow E2E complete for all model types

## 12. Can Claim

- ✅ Common text Chat (vLLM/SGLang), GGUF (llama.cpp), Embedding (vLLM), Reranker (vLLM) — real production E2E verified
- ✅ Invalid model/backend combinations blocked before runtime
- ✅ Unsupported model types detected, cataloged, non-deployable by default
- ✅ VLM recognized, entered into library, blocked at preflight with clear reason
- ✅ Embedding and rerank test modes implemented with backend-declared endpoints
- ✅ Backend capability declaration drives compatibility enforcement
- ✅ 13 model type detectors with clean priority-based plugin registry

## 13. Test Results (Final)

```bash
gofmt -w cmd/ internal/                                       → CLEAN
go test lightai-go/internal/agent/collector/...                → ALL PASS (33 tests)
go test lightai-go/internal/server/runplan/...                 → ALL PASS (20 compat tests)
go test lightai-go/internal/server/api/...                     → ALL PASS
go vet ./...                                                   → CLEAN
npm test                                                       → ALL PASS
npm run build                                                  → ✓ built
git diff --check                                               → CLEAN
git status --short                                             → CLEAN
```

## 14. Release-Readiness Recommendation

The model detection / runtime compatibility workstream is ready for the next milestone with a documented VLM backend blocker.

Production-ready now:
- text chat on vLLM/SGLang
- GGUF chat on llama.cpp
- embedding on vLLM
- rerank on vLLM
- invalid combinations blocked before runtime
- unsupported assets recognized but non-deployable

Not production-ready yet:
- VLM runtime for InternVL2.5 (blocked at preflight)
- ONNX/TensorRT/OpenVINO/Diffusers/ASR/TTS/Classification serving (no backends)
- LoRA standalone deployment (requires base model composition)

## 15. Modified Files Summary

```
internal/agent/collector/model_scanner.go       — 13 detectors, plugin registry, FileFacts
internal/agent/collector/model_scanner_test.go  — 35 tests
internal/agent/collector/gguf_reader.go         — GGUF metadata parsing
internal/server/runplan/compat.go               — CompatibilityChecker
internal/server/runplan/compat_test.go          — 20 compat tests
internal/server/runplan/resolver.go             — MODEL_CONTAINER_FILE variable
internal/server/runplan/llamacpp_nvidia_test.go — GGUF file path tests
internal/server/db/db.go                        — V25-V27 migrations, seed capabilities
internal/server/api/artifact_handlers.go        — Task/caps/GGUF validation, location metadata
internal/server/api/deployment_lifecycle_handlers.go — Preflight compat, embedding/rerank test
internal/server/api/agent_proxy_handlers.go     — Scan proxy path preservation
internal/server/api/runtime_handlers.go         — NBR check status fix
web/src/pages/ModelArtifactsPage.vue            — Wizard scan flow, detail/edit pages, scanner info
web/src/pages/ModelInstancesPage.vue            — Test modes, logs auto-refresh
web/src/pages/RunnerConfigsPage.vue             — NBR check status fix
web/src/locales/zh-CN.ts                        — ~50 new i18n keys
web/src/locales/en-US.ts                        — ~50 new i18n keys
web/src/utils/modelCapabilities.js              — Persistent capabilities logic
web/tests/modelCapabilities.test.mjs            — Capability tests
docs/reports/.../open-issues-closeout.md        — VLM-RUNTIME-001 blocker
docs/reports/.../16-37*.md                      — Plan, closeout, review documents
docs/reports/.../evidence/batch4-full-flow-e2e/ — 33 evidence files
```
