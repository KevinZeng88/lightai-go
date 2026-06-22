# Model Detection Runtime Plugin Implementation Plan

> Status: DRAFT_FOR_REVIEW
> Based on: Design doc 28 (`28-model-detection-runtime-plugin-design.md`)
> Baseline: commit `625ac16`
> Date: 2026-06-23

## 1. Review Scope

This document reviews the current codebase against the design doc 28, identifies gaps, and proposes a phased implementation plan. It does NOT implement any code changes.

## 2. Current Implementation Summary

### 2.1 Agent Scanner (`internal/agent/collector/model_scanner.go`)

**ScanCandidate struct** — 9 fields:
```go
Path, PathType, Format, DetectedMetadata, Warnings, AutoSelected, SelectionReason, SizeBytes, SizeLabel
```

**Missing from design**: No `Kind`, `Task`, `Capabilities`, `DefaultTestMode`, `Deployable`, `RequiresBaseModel`, `RecommendedBackends`, `Confidence`, `Evidence`, `UnsupportedReason` fields.

**Detection coverage**:
- HF directory: ✅ (checks `config.json` existence)
- GGUF file: ✅ (globs `*.gguf`)
- Embedding (SentenceTransformers): ❌
- Reranker/CrossEncoder: ❌
- Vision-Language: ❌
- LoRA/Adapter: ❌
- ONNX: ❌
- TensorRT Engine: ❌
- OpenVINO: ❌
- Diffusers: ❌
- ASR: ❌
- TTS: ❌
- Classification: ❌

### 2.2 Scan API Proxy (`internal/server/api/agent_proxy_handlers.go`)

Enriches response with `root_id`, `root`, `model_root`, `scan_root`, `relative_path`, `absolute_path`. Preserves per-candidate paths. Does NOT add model type metadata.

### 2.3 Frontend Wizard (`web/src/pages/ModelArtifactsPage.vue`)

**Candidate display**: Shows format badge (HF green / GGUF orange), path basename, quantization, context length. Auto-select prioritizes HF directory over GGUF.

**Missing**: No task type, deployability status, recommended backends, confidence, evidence, or unsupported reason display.

### 2.4 Model Persistence

**model_artifacts table columns**: `capabilities_json`, `capability_sources_json`, `default_test_mode`, `format`, `task_type`, `architecture`, etc.

**model_locations table columns**: `path_type` (file/directory), `discovered_metadata_json`.

**Missing**: No `metadata_json` column on either table. No `kind` column on locations. No scanner metadata (task, deployable, recommended_backends, evidence, confidence) is persisted anywhere.

### 2.5 Backend / BackendVersion / BackendRuntime Capabilities

**backend_versions**: Has `capabilities_json` TEXT column (JSON object). Used in seed data.

**backend_runtimes**: No format/task capability columns.

**Missing**: No `supported_formats`, `supported_tasks`, `model_path_mode`, or `test_endpoints` columns or JSON fields in any table.

### 2.6 Preflight / Compatibility

**No compatibility checks exist.** Preflight validates node readiness, model location existence, GPU availability — but never checks whether the selected backend supports the model's format. vLLM + GGUF and llama.cpp + HF directory are both silently allowed.

### 2.7 Test Dialog / Test API

**Frontend** (`ModelInstancesPage.vue`): Test mode selector has only `auto`, `chat`, `completion`. No `embedding` or `rerank` options.

**Backend** (`deployment_lifecycle_handlers.go`): `tryInferenceWithMode()` only handles chat/completion/auto. Embedding/rerank modes fall through to auto (chat→completion fallback). No `/v1/embeddings` or `/v1/rerank` endpoint calls exist.

### 2.8 i18n

Has basic capability labels. Missing: task type labels (Embedding, Reranker, Vision-Language, etc.), deployability labels, unsupported reason messages.

## 3. Design Gap Analysis

| Gap ID | Area | Current | Target | Severity | Schema Change? | Phase |
|--------|------|---------|--------|----------|---------------|-------|
| G-01 | ScanCandidate struct | 9 fields, no task/caps/deployable | 17-field ModelCandidate with kind/task/caps/deployable/evidence/confidence | HIGH | No (struct only) | A |
| G-02 | Detector coverage | 2 detectors (HF, GGUF) | 13 detectors (embedding, reranker, VLM, LoRA, ONNX, etc.) | MEDIUM | No | B |
| G-03 | FileFacts abstraction | No pre-collected facts | FileFacts struct shared across detectors | LOW | No | B |
| G-04 | Scan API metadata enrichment | No type metadata added | Proxy preserves all candidate fields | LOW | No | A |
| G-05 | Wizard candidate display | Format badge only | Full: task, caps, deployable, backends, evidence, confidence, unsupported | MEDIUM | No | C |
| G-06 | Model persistence | capabilities_json only; no scan metadata | Persist task/deployable/recommended_backends/evidence in metadata_json or discovered_metadata_json | MEDIUM | No (use existing) | C |
| G-07 | Backend capability | No supported_formats/tasks columns | Express via capabilities_json or new JSON columns | MEDIUM | TBD (prefer existing columns) | D |
| G-08 | Preflight compatibility | None | Backend-model format/task compatibility checks; fail incompatible combinations | HIGH | No | D |
| G-09 | Test dialog modes | Only auto/chat/completion | Add embedding/rerank to frontend selector and backend handler | MEDIUM | No | E |
| G-10 | Test endpoint dispatch | No embedding/rerank API calls | Add /v1/embeddings and rerank endpoint calls | MEDIUM | No | E |
| G-11 | i18n coverage | Basic caps/labels | Task type labels, deployability labels, unsupported reason i18n | LOW | No | C |
| G-12 | Test coverage | 2 detector types in tests | Tests for all 13 detector types + compatibility checks | MEDIUM | No | F |

## 4. Schema Decision

**Default position: no new schema columns.** All new metadata (task, deployable, recommended_backends, evidence, confidence, unsupported_reason) will be stored in existing JSON columns:

| Data | Where stored |
|------|-------------|
| Scanner metadata (task, confidence, evidence, deployable, recommended_backends, unsupported_reason) | `model_locations.discovered_metadata_json` (already exists) |
| Model capabilities | `model_artifacts.capabilities_json` (already exists) |
| Capability sources | `model_artifacts.capability_sources_json` (already exists) |
| Default test mode | `model_artifacts.default_test_mode` (already exists) |
| Backend supported formats/tasks | `backend_versions.capabilities_json` (already exists as JSON object) |

**If schema change proves necessary** (only if existing JSON columns prove insufficient during implementation):
- Add `model_artifacts.metadata_json TEXT NOT NULL DEFAULT '{}'` to mirror `model_locations.discovered_metadata_json`
- No new migration needed for V25+ databases; add column in V27 if needed

## 5. Proposed Architecture (Refinement of Design Doc)

The design doc's 5-layer architecture is sound. For LightAI Go's current Go-based implementation, the practical refinement is:

```
Layer 1: ScanCandidate (enriched struct, immediate)
    ↓  user selects one
Layer 2: ModelArtifact + ModelLocation (persist candidate metadata via discovered_metadata_json)
    ↓  preflight runs
Layer 3: CompatibilityChecker (pure function: ModelDescriptor × BackendDescriptor → CompatResult)
    ↓  only if compatible
Layer 4: RunPlanResolver + TestMethodResolver (already exist, minor additions)
```

For the Detector Registry: use a Go function table (`[]DetectorFunc`) rather than a Go interface with shared struct, given the project's existing style (no heavy interface abstractions in the current scanner).

## 6. Phased Implementation Plan

### Phase A: Data Contract — ModelCandidate Enrichment

**Goal**: Expand ScanCandidate to carry all fields from the design doc without changing detection logic.

**Changes**:
1. Add fields to `ScanCandidate` struct in `model_scanner.go`:
   - `Kind string` (derived from PathType + context)
   - `Task string` (derived from format + metadata inference)
   - `Capabilities []string`
   - `DefaultTestMode string`
   - `Deployable bool`
   - `RequiresBaseModel bool`
   - `RecommendedBackends []string`
   - `Confidence string`
   - `Evidence []string`
   - `UnsupportedReason string`

2. Populate these fields for existing detectors:
   - HF directory: kind="directory", task="chat", deployable=true, confidence="medium", evidence=["config.json"]
   - GGUF: kind="file", task="chat", deployable=true, confidence="high", evidence=["*.gguf"]

3. Update `toMap()` to include new fields in API response

4. Update scan proxy to preserve new candidate fields

5. Update frontend wizard `doWizardSave` to read new fields (capabilities, default_test_mode, task, recommended_backends, evidence) from candidate and pass them to create API

**No**: new detectors, new UI sections, compatibility checks

**Acceptance**:
- Scan API returns candidates with kind, task, capabilities, deployable, recommended_backends, confidence, evidence
- HF and GGUF behavior unchanged
- Tests: go test ./internal/server/api/... , go test ./internal/server/runplan/... ALL PASS
- npm test + npm run build ALL PASS

**Risk**: Low — additive changes only, no logic changes to detection

---

### Phase B: Detector Registration

**Goal**: Refactor detection into a detector function table without changing what is currently detected; then add new detectors.

**Changes**:
1. Create `FileFacts` struct in scanner (collects directory listing, key JSON files, glob results once)
2. Define `DetectorFunc` type: `func(facts FileFacts) []ScanCandidate`
3. Build `FileFacts` once in `scanDirectory()`, pass to all registered detectors
4. Migrate existing HF and GGUF detection into detector functions
5. Add new detectors (output only — no backend changes):
   - `DetectSentenceTransformers` — checks modules.json, sentence_bert_config.json, name patterns
   - `DetectReranker` — checks name patterns (reranker, cross-encoder, bge-reranker), config hints
   - `DetectVisionLanguage` — checks name patterns (qwen-vl, llava, internvl), preprocessor_config
   - `DetectLoRAAdapter` — checks adapter_config.json, adapter_model.safetensors
   - `DetectONNX` — checks *.onnx
   - `DetectTensorRT` — checks *.engine
   - `DetectOpenVINO` — checks *.xml + *.bin
   - `DetectDiffusers` — checks model_index.json, unet/
   - `DetectASR` — checks name patterns (whisper, funasr, paraformer)
   - `DetectTTS` — checks name patterns (cosyvoice, chattts, gpt-sovits)
   - `DetectClassification` — checks config architectures for SequenceClassification/TokenClassification
6. Detection priority order: LoRA → SentenceTransformers → Reranker → VLM → HF Chat → Diffusers → ASR → TTS → Classification → OpenVINO → GGUF → ONNX → TensorRT
7. Unsupportable types (ONNX, TensorRT, OpenVINO, Diffusers, ASR, TTS, Classification) have `deployable=false` with clear `unsupported_reason`

**No**: UI changes, compatibility checks, schema changes, test endpoint changes

**Acceptance**:
- All 13 detector types produce correct candidate output
- Existing HF/GGUF behavior unchanged
- New types show correct task, deployable, recommended_backends, unsupported_reason
- Tests: each detector has unit test coverage (at minimum: 1 positive case per detector)
- go test, go vet, npm test, npm build ALL PASS

**Risk**: Medium — refactoring existing detection into functions could introduce regressions if not careful. Mitigation: keep existing code paths as-is, wrap in detector functions, validate with existing tests first.

---

### Phase C: Model Persistence and UI Display

**Goal**: Persist scanner metadata into model records and display it in UI.

**Changes**:

**Backend**:
1. In `HandleCreateModelLocation` and `doWizardSave`: persist scanner metadata fields into `discovered_metadata_json` on `model_locations`:
   ```json
   {
     "kind": "directory",
     "task": "chat",
     "deployable": true,
     "requires_base_model": false,
     "recommended_backends": ["vllm", "sglang"],
     "confidence": "medium",
     "evidence": ["config.json", "tokenizer_config.json"],
     "unsupported_reason": "",
     "detector_id": "hf_chat"
   }
   ```
2. Populate `capabilities_json`, `capability_sources_json`, `default_test_mode` on model_artifacts from candidate data
3. In artifact detail API: read `discovered_metadata_json` from first location and merge with artifact fields

**Frontend detail page**:
1. Show task type, model format, capabilities, default test mode
2. Show scanner metadata: deployable status, recommended backends, confidence, evidence
3. Show unsupported reason for non-deployable models
4. Add section for "扫描识别信息" (Scan Recognition Info) with collapsible evidence

**Frontend edit page**:
1. Already supports editing capabilities and default_test_mode (Phase 2)
2. Add task type editing (select: chat/completion/embedding/rerank/vision_chat/classification/unknown)

**i18n**: Add keys for all task types, deployable/not-deployable labels, backend names, evidence labels

**No**: Compatibility checks, test endpoint changes, new detectors

**Acceptance**:
- Scanner metadata persists in `discovered_metadata_json`
- Model detail page shows task, deployable, recommended backends, evidence
- Embedding/Reranker/Vision models show correct task type in UI
- LoRA shows "不可独立部署" with reason
- ONNX shows "当前不支持" with reason
- No undefined/null/[object Object] leaks
- Tests: go test, npm test, npm build ALL PASS

**Risk**: Low — additive UI changes, existing JSON column usage

---

### Phase D: Backend Capability and Compatibility Checker

**Goal**: Express what each backend supports and prevent incompatible model-backend combinations at preflight.

**Changes**:

**Backend capability** (seed data only, no schema change):
1. Enrich `backend_versions.capabilities_json` in seed data:
   ```json
   {
     "supported_formats": ["huggingface", "sentence_transformers"],
     "supported_tasks": ["chat", "completion", "embedding", "rerank", "vision_chat"],
     "model_path_mode": "directory",
     "test_endpoints": {
       "chat": "/v1/chat/completions",
       "completion": "/v1/completions",
       "embedding": "/v1/embeddings",
       "rerank": ["/v1/rerank", "/rerank"]
     }
   }
   ```
2. For llama.cpp: supported_formats=["gguf"], supported_tasks=["chat","completion"], model_path_mode="file"

**CompatibilityChecker** (`internal/server/runplan/compat.go` — new file):
1. Pure function: takes ModelDescriptor (format, task, path_type) + BackendDescriptor (supported_formats, model_path_mode) → CompatResult
2. Rules:
   - vLLM/SGLang + format=gguf → FAIL: "模型为 GGUF 文件，vLLM/SGLang 不支持。请使用 llama.cpp。"
   - llama.cpp + format=huggingface → FAIL: "模型为 HuggingFace 目录，llama.cpp 不支持。请使用 vLLM/SGLang。"
   - LoRA + standalone deploy → FAIL: "这是 LoRA/Adapter，不能作为独立模型部署。"
   - model_path_mode=file + path_type=directory → FAIL: "后端需要具体模型文件路径，但当前模型位置是目录。"
   - Supported format + task mismatch → WARNING (not block)
3. Add unit tests for all pass/fail scenarios

**Preflight integration** (`deployment_lifecycle_handlers.go`):
1. In `preflightDeployment()`, after model location validation, call `CompatibilityChecker`
2. On failure: add structured preflight error, block deployment
3. Frontend shows compatibility error in preflight UI

**No**: Schema changes, new backend columns, RunPlan changes

**Acceptance**:
- vLLM + GGUF fails preflight with clear error
- llama.cpp + HF directory fails preflight with clear error
- LoRA standalone deploy fails preflight
- Compatible combinations pass preflight
- Tests: go test with new compat tests, existing tests unchanged
- npm test + npm build ALL PASS

**Risk**: Medium — preflight logic change could break deployment flow. Mitigation: add compatibility check as a new step after existing validations, not replacing any.

---

### Phase E: Test Method Abstraction

**Goal**: Support embedding and rerank test modes in both frontend and backend.

**Changes**:

**Frontend** (`ModelInstancesPage.vue`):
1. Add `embedding` and `rerank` options to test mode `<el-select>`
2. `recommendedTestMode()` already handles these (returns the persisted/default value)

**Backend** (`deployment_lifecycle_handlers.go`):
1. Add `tryEmbeddingInference()` function: POST to `/v1/embeddings` with `{"input": "hello world", "model": "..."}`
2. Add `tryRerankInference()` function: POST to rerank endpoint candidates with query+documents payload
3. Update `tryInferenceWithMode()` to dispatch to embedding/rerank functions
4. Read test endpoint from `backend_versions.capabilities_json.test_endpoints` (from Phase D)
5. On rerank endpoint not declared: return diagnostic result with clear message

**No**: Schema changes, new backend columns

**Acceptance**:
- Embedding models default to embedding test mode
- Reranker models default to rerank test mode  
- Test selector shows all 5 modes: auto/chat/completion/embedding/rerank
- Embedding test calls /v1/embeddings
- Rerank test calls declared rerank endpoint
- If no rerank endpoint declared, shows diagnostic message
- Tests: go test, npm test, npm build ALL PASS

**Risk**: Low — additive, no existing behavior changes

---

### Phase F: Final Hardening and Closeout

**Goal**: Complete documentation, tests, and clean commit.

**Changes**:
1. Run full test suite: go test, go vet, npm test, npm build
2. Verify git diff --check clean
3. Create closeout document `29-model-detection-runtime-plugin-closeout.md`
4. Final commit and push

**Acceptance**:
- All tests pass
- git status clean
- Closeout document complete

## 7. Phase Acceptance Criteria Summary

| Phase | Go Tests | Frontend Tests | Build | Key Assertion |
|-------|----------|---------------|-------|---------------|
| A | ALL PASS | ALL PASS | ✅ | Candidate has kind/task/caps/deployable |
| B | ALL PASS + detector tests | ALL PASS | ✅ | 13 detectors produce correct output |
| C | ALL PASS | ALL PASS | ✅ | Task/caps/evidence shown in detail/edit |
| D | ALL PASS + compat tests | ALL PASS | ✅ | vLLM+GGUF blocked, llama.cpp+HF blocked |
| E | ALL PASS | ALL PASS | ✅ | Embedding/rerank test modes work |
| F | ALL PASS | ALL PASS | ✅ | git status clean, closeout complete |

## 8. Test Plan

### New Tests Required

**Phase A**: No new tests (struct-only changes, existing tests verify no regression)

**Phase B**: ~13 detector unit tests (one per detector type)
- `TestDetectHuggingFaceChat`
- `TestDetectSentenceTransformers`
- `TestDetectReranker`
- `TestDetectVisionLanguage`
- `TestDetectGGUFFile`
- `TestDetectLoRAAdapter`
- `TestDetectONNX`
- `TestDetectTensorRT`
- `TestDetectOpenVINO`
- `TestDetectDiffusers`
- `TestDetectASR`
- `TestDetectTTS`
- `TestDetectClassification`
- `TestEmptyDirectory`
- `TestMixedHFAndGGUF`

**Phase C**: No new backend tests (UI changes only). Update existing wizard tests if needed.

**Phase D**: ~8 compatibility tests
- `TestCompatVLLMWithGGUFFails`
- `TestCompatLlamaCppWithHFFails`
- `TestCompatLoRAStandaloneFails`
- `TestCompatVLLMWithHFPasses`
- `TestCompatLlamaCppWithGGUFPasses`
- `TestCompatFilePathModeWithDirectoryFails`
- `TestCompatONNXWithoutBackendFails`
- `TestCompatEmbeddingWithVLLMPasses`

**Phase E**: ~4 test mode tests
- `TestEmbeddingTestEndpoint`
- `TestRerankTestEndpoint`
- `TestRerankNoEndpointDeclared`
- `TestChatCompletionEndpointsUnchanged`

### Verification Commands (all phases)

```bash
gofmt -w cmd/ internal/
go test ./internal/server/api/...
go test ./internal/server/runplan/...
go test ./internal/agent/...   # if agent scanner tests exist
go vet ./...
npm --prefix web test
npm --prefix web run build
git diff --check
git status --short
```

## 9. Risk and Rollback Notes

| Risk | Mitigation |
|------|-----------|
| Detector refactoring breaks existing HF/GGUF | Keep existing code paths as first-pass detectors; add new detectors after |
| Preflight compatibility breaks deployment | Add as new validation step, not replacing existing checks; feature-flag if needed |
| New candidate fields break scan API consumers | Add fields with zero-value defaults; existing fields unchanged |
| Embedding/rerank test not testable without running instance | Test with unit tests for endpoint dispatch; leave runtime validation for Phase 3 |
| Seed data update breaks existing DB | Use V26 migration pattern: REPLACE old JSON, no new columns |

## 10. Explicit Non-goals

1. No resource parameter editor (Phase 3)
2. No multi-replica/cross-node scheduling
3. No Playwright specs
4. No API Gateway/API Key
5. No new backends (ONNX Runtime, TensorRT-LLM, OpenVINO, Diffusers, ASR, TTS)
6. No model conversion
7. No LoRA merge
8. No image/audio upload test UI
9. No schema changes (default position; only if existing JSON columns prove insufficient)
10. No backward compatibility for old data

## 11. Open Questions for User Review

1. **Q1: Detector registry style** — The design doc suggests either Go interface or function table. The current codebase uses plain functions (not interfaces). Should we stay with function tables (`[]DetectorFunc`) for consistency, or move to interfaces for extensibility?

2. **Q2: Schema for backend capabilities** — The plan reuses `backend_versions.capabilities_json` (existing JSON column). Is this acceptable, or should we add first-class columns (`supported_formats_json`, `supported_tasks_json`)?

3. **Q3: LoRA/Adapter handling** — Should LoRA adapters be completely hidden from deployment UI, or listed but grayed out with "不可独立部署" message? Plan proposes the latter (listed, grayed out).

4. **Q4: Unsupported models in model library** — Should ONNX/TensorRT/OpenVINO/Diffusers models enter the model library at all? Plan says yes — they should be listed but marked as "当前不支持" to provide visibility.

5. **Q5: Rerank endpoint probing** — Should the test handler probe multiple rerank endpoint candidates (`/v1/rerank`, `/rerank`, `/v2/rerank`, `/score`, `/v1/score`) or only use the declared one? Plan says use declared; probe only if none declared.

6. **Q6: Phase ordering** — Phase C (UI) before Phase D (compatibility) or Phase D before Phase C? Plan puts UI persistence first because it provides visibility before enforcing constraints.

## 12. Recommended Next Prompt After Approval

```
Proceed to Phase A: implement ModelCandidate struct enrichment.
Start with internal/agent/collector/model_scanner.go.
Do not add new detectors — only add fields and populate them for existing HF/GGUF detection.
```
