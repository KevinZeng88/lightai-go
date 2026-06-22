# Model Detection Runtime Plugin Implementation Plan

> Status: REVISED_FOR_REVIEW
> Based on: Design doc 28 (`28-model-detection-runtime-plugin-design.md`)
> Baseline: commit `625ac16`
> Date: 2026-06-23
> Revised: per review feedback (7 revision points)

## 1. Review Scope

This document reviews the current codebase against the design doc 28, identifies gaps, and proposes a phased implementation plan. It does NOT implement any code changes.

## 2. Current Implementation Summary

### 2.1 Agent Scanner (`internal/agent/collector/model_scanner.go`)

**ScanCandidate struct** — 9 fields:
```go
Path, PathType, Format, DetectedMetadata, Warnings, AutoSelected, SelectionReason, SizeBytes, SizeLabel
```

**Missing from design**: No `Kind`, `Task`, `Capabilities`, `DefaultTestMode`, `Deployable`, `RequiresBaseModel`, `RecommendedBackends`, `Confidence`, `Evidence`, `UnsupportedReason` fields.

**Detection coverage**: HF directory ✅, GGUF file ✅. All other model types ❌.

### 2.2 Scan API Proxy (`internal/server/api/agent_proxy_handlers.go`)

Enriches response with `root_id`, `root`, `model_root`, `scan_root`, `relative_path`, `absolute_path`. Preserves per-candidate paths. Does NOT add model type metadata.

### 2.3 Frontend Wizard (`web/src/pages/ModelArtifactsPage.vue`)

Shows format badge (HF green / GGUF orange), path basename, quantization, context length. Auto-select prioritizes HF directory. Missing: task type, deployability status, recommended backends, confidence, evidence, unsupported reason.

### 2.4 Model Persistence

**model_artifacts**: `capabilities_json`, `capability_sources_json`, `default_test_mode`, `format`, `task_type`, `architecture`.
**model_locations**: `path_type` (file/directory), `discovered_metadata_json`.
**Missing**: No scanner metadata (task, deployable, recommended_backends, evidence, confidence) is persisted.

### 2.5 Backend / BackendVersion / BackendRuntime Capabilities

**backend_versions**: Has `capabilities_json` TEXT column (JSON object). Used in seed data.
**backend_runtimes**: No format/task capability columns.
**Missing**: No `supported_formats`, `supported_tasks`, `model_path_mode`, or `test_endpoints` in any table.

### 2.6 Preflight / Compatibility

**No compatibility checks exist.** vLLM + GGUF and llama.cpp + HF directory are both silently allowed.

### 2.7 Test Dialog / Test API

Frontend: only `auto`, `chat`, `completion` modes. No `embedding` or `rerank`.
Backend: `tryInferenceWithMode()` only handles chat/completion/auto. No `/v1/embeddings` or `/v1/rerank` calls.

### 2.8 i18n

Has basic capability labels. Missing: task type labels, deployability labels, unsupported reason messages.

## 3. Design Gap Analysis

| Gap ID | Area | Current | Target | Severity | Schema? | Phase |
|--------|------|---------|--------|----------|---------|-------|
| G-01 | ScanCandidate struct | 9 fields | Full ModelCandidate with kind/task/caps/deployable/evidence/confidence | HIGH | No | A |
| G-02a | Detector: deployable types | 2 detectors | 6 core types (HF, GGUF, Embedding, Reranker, VLM, LoRA) | MEDIUM | No | B1 |
| G-02b | Detector: unsupported types | 0 detectors | 7 unsupported types (ONNX, TensorRT, OpenVINO, Diffusers, ASR, TTS, Classification) | LOW | No | B2 |
| G-03 | FileFacts abstraction | None | Shared FileFacts across detectors | LOW | No | B1 |
| G-04 | Scan API enrichment | No type metadata | Proxy preserves all candidate fields | LOW | No | A |
| G-05 | Wizard candidate display | Format badge only | Full: task, caps, deployable, backends, evidence, unsupported | MEDIUM | No | C |
| G-06 | Model persistence | caps only; no scan metadata | Persist into artifact caps + location discovered_metadata_json | MEDIUM | No | C |
| G-07 | Backend capability | No supported_formats/tasks | Express via `backend_versions.capabilities_json` structured sub-fields | MEDIUM | No | D |
| G-08 | Preflight compatibility | None | CompatibilityChecker blocks invalid combos | HIGH | No | D |
| G-09 | Test dialog modes | Only auto/chat/completion | Add embedding/rerank to frontend + backend | MEDIUM | No | E |
| G-10 | Test endpoint dispatch | No embedding/rerank calls | Add /v1/embeddings and rerank endpoint calls | MEDIUM | No | E |
| G-11 | i18n | Basic caps/labels | Task type, deployability, unsupported labels | LOW | No | C |
| G-12 | Regression tests | 2 detector types | All detector types + compat + path mode + i18n | MEDIUM | No | Each phase |

## 4. Abstraction: ModelType Plugin

### 4.1 Plugin Definition

The core abstraction is a **ModelType Plugin** — a lightweight Go struct that bundles everything needed to recognize and characterize one model type:

```go
type ModelTypePlugin struct {
    ID          string           // e.g. "embedding.sentence_transformers"
    Detect      DetectorFunc     // returns []ScanCandidate or nil
    Defaults    ModelTypeDefaults
}

type ModelTypeDefaults struct {
    Kind               string   // "directory" | "file" | "adapter" | "bundle"
    Format             string   // "huggingface" | "sentence_transformers" | "gguf" | "lora_adapter" | ...
    Task               string   // "chat" | "completion" | "embedding" | "rerank" | "vision_chat" | ...
    Capabilities       []string // ["chat","completion"] | ["embedding"] | ...
    DefaultTestMode    string   // "chat" | "embedding" | "rerank" | "auto"
    Deployable         bool     // can it run as a standalone model?
    RequiresBaseModel  bool     // adapter/lora case
    RecommendedBackends []string // ["vllm","sglang"] | ["llamacpp"] | []
    UnsupportedReason  string   // only for deployable=false
}
```

A plugin answers 10 things about a model type:
1. **Detector** — how to recognize it (DetectorFunc)
2. **Model semantic** — format, task, capabilities, default_test_mode
3. **Path semantic** — directory, file, adapter, or bundle
4. **Deployability** — can it run standalone?
5. **Recommended backends** — which backends typically support it
6. **Compatibility rule** — what backend capabilities are required (derived from Defaults.Format × Defaults.Task)
7. **Run method hint** — directory path or file path (derived from Kind)
8. **Test method** — which test mode and endpoint type (derived from DefaultTestMode)
9. **Evidence/confidence** — set by the detector, not the defaults
10. **Unsupported reason** — why it can't run (if Deployable=false)

### 4.2 Plugin Registration

Plugins are registered as a function table (not a Go interface), matching the project's existing style:

```go
var modelTypePlugins = []ModelTypePlugin{
    PluginLoRAAdapter,           // highest priority
    PluginSentenceTransformers,
    PluginReranker,
    PluginVisionLanguage,
    PluginHuggingFaceChat,
    PluginDiffusers,
    PluginASR,
    PluginTTS,
    PluginClassification,
    PluginOpenVINO,
    PluginGGUF,
    PluginONNX,
    PluginTensorRT,              // lowest priority
}
```

Adding a new model type = adding one plugin to this table. No changes to frontend pages, RunPlan resolver, preflight, or test dialog.

### 4.3 Plugin Execution

```go
func scanDirectory(absPath string, facts FileFacts) []ScanCandidate {
    var candidates []ScanCandidate
    for _, plugin := range modelTypePlugins {
        detected := plugin.Detect(facts)
        for i := range detected {
            // Apply plugin defaults to each candidate
            applyDefaults(&detected[i], plugin.Defaults)
        }
        candidates = append(candidates, detected...)
    }
    // Auto-selection: single → auto; multiple same-type → warn; mixed → user picks
    return applyAutoSelection(candidates)
}
```

## 5. Data Authority (Where Each Fact Lives)

**Critical: do not conflate artifact-level semantics with location-level evidence.**

### ModelArtifact — Model Semantic Authority

```
format                     ← persistent (already exists)
task_type                  ← persistent (already exists; currently defaults to "chat")
capabilities_json          ← persistent (Phase 2)
capability_sources_json    ← persistent (Phase 2)
default_test_mode          ← persistent (Phase 2)
```

These fields describe WHAT the model IS. They travel with the artifact, independent of any specific node location.

### ModelLocation — Location & Scan Evidence Authority

```
path                       ← persistent (already exists)
path_type                  ← persistent (already exists: "file" | "directory")
discovered_metadata_json   ← persistent (already exists; enriched in Phase C):
    kind                   ← "directory" | "file" | "adapter"
    evidence               ← ["config.json", "tokenizer_config.json", ...]
    confidence             ← "high" | "medium" | "low"
    detector_id            ← which plugin identified this
    scan_root              ← the directory that was scanned
    unsupported_reason     ← only if location-specific (rare; usually on artifact)
```

These fields describe HOW and WHERE the model was found. They are node-specific and may differ between nodes.

### BackendVersion — Runtime Capability Authority

```
capabilities_json (structured sub-fields; Phase D):
    supported_formats       ← ["huggingface","sentence_transformers"] | ["gguf"]
    supported_tasks         ← ["chat","completion","embedding","rerank","vision_chat"]
    supported_capabilities  ← ["chat","completion","embedding","rerank","vision"]
    model_path_modes        ← ["directory"] | ["file"]
    test_endpoints          ← {"chat":"/v1/chat/completions","embedding":"/v1/embeddings",...}
```

These fields describe what a backend CAN run. They are backend-version-specific.

### Why Not New Schema Columns

All new data fits into existing JSON columns. The `discovered_metadata_json` on model_locations already exists and is already read/written by the API. The `capabilities_json` on both model_artifacts and backend_versions already exists. Adding first-class columns would duplicate storage without adding query capability (SQLite JSON functions can query into these columns if needed).

## 6. Schema Decision

**No new schema columns.** All new metadata uses existing JSON columns. No migration needed.

| Data | Where stored | Column exists? |
|------|-------------|----------------|
| Model capabilities, sources, default_test_mode | `model_artifacts.capabilities_json`, `.capability_sources_json`, `.default_test_mode` | ✅ Phase 2 |
| Scanner evidence, confidence, detector_id, scan_root | `model_locations.discovered_metadata_json` | ✅ V13 |
| Backend supported formats, tasks, path modes, test endpoints | `backend_versions.capabilities_json` (structured sub-fields) | ✅ V17 |

If during implementation existing columns prove insufficient, fallback option:
- Add `model_artifacts.metadata_json TEXT NOT NULL DEFAULT '{}'` via V27 migration
- This would only happen if `discovered_metadata_json` on locations can't carry artifact-level metadata cleanly

## 7. Phased Implementation Plan

### Phase A: Candidate Contract — Field Plumbing Only

**Goal**: Expand ScanCandidate struct with all design fields, populate them for existing HF/GGUF detectors, and ensure the entire pipeline (scan → proxy → wizard → create) preserves them. No new detectors. No UI changes.

**Changes**:
1. Add fields to `ScanCandidate` in `model_scanner.go`:
   - `Kind string`, `Task string`, `Capabilities []string`, `DefaultTestMode string`
   - `Deployable bool`, `RequiresBaseModel bool`, `RecommendedBackends []string`
   - `Confidence string`, `Evidence []string`, `UnsupportedReason string`

2. Populate for existing detectors:
   - HF directory: kind="directory", task="chat", capabilities=["chat","completion"], default_test_mode="chat", deployable=true, confidence="medium", evidence=["config.json"], recommended_backends=["vllm","sglang"]
   - GGUF file: kind="file", task="chat", capabilities=["chat","completion"], default_test_mode="chat", deployable=true, confidence="high", evidence=["*.gguf"], recommended_backends=["llamacpp"]

3. Update `toMap()` to include new fields in API response

4. Scan proxy: no changes needed (already preserves candidate fields)

5. Frontend `doWizardSave`: read `capabilities`, `default_test_mode`, `task`, `recommended_backends` from candidate; pass to create APIs; ensure `path_type` is correctly set

**Explicitly NOT in Phase A**:
- New detectors
- New detail/edit UI sections
- Compatibility checker
- Embedding/rerank test modes
- Unsupported type recognition
- i18n additions beyond what's needed for field flow

**Acceptance**:
- Scan API returns candidates with kind, task, capabilities, deployable, recommended_backends, confidence, evidence
- HF/GGUF existing behavior unchanged: direct file selection RunPlan correct, directory scan RunPlan correct
- Create model from wizard preserves capabilities and default_test_mode (no regression from Phase 2)
- Tests: `go test ./internal/server/api/...`, `go test ./internal/server/runplan/...`, `npm test`, `npm run build` ALL PASS
- **Regression gate**: GGUF RunPlan `-m` still points to `.gguf` file; HF RunPlan still uses directory path

**Risk**: Low — additive struct fields only, no logic changes to detection

---

### Phase B1: Core Deployable Model Type Plugins

**Goal**: Add detectors for model types that have known deployable backends. These immediately add value because they enable correct capabilities, task types, and recommended backends.

**Detectors added** (6 new, 2 regression):

| # | Detector | Detection evidence | Format | Task | Deployable | Recommended Backends |
|---|----------|-------------------|--------|------|------------|---------------------|
| 1 | HF Chat/Completion (regression) | config.json exists | huggingface | chat | true | vllm, sglang |
| 2 | GGUF file (regression) | *.gguf | gguf | chat | true | llamacpp |
| 3 | SentenceTransformers / Embedding | modules.json, sentence_bert_config.json, name patterns | sentence_transformers | embedding | true | vllm, sglang |
| 4 | Reranker / CrossEncoder | name patterns (reranker, cross-encoder, bge-reranker), config hints | huggingface | rerank | true | vllm, sglang |
| 5 | Vision-Language / Multimodal | name patterns (qwen-vl, llava, internvl), preprocessor_config, image_processor_config | huggingface | vision_chat | true | vllm, sglang |
| 6 | LoRA / Adapter | adapter_config.json, adapter_model.safetensors | lora_adapter | adapter | false | [] |

**Implementation**:
1. Introduce `FileFacts` struct (collects directory listing, key JSON files, glob results once)
2. Define `ModelTypePlugin` struct with `Detect` and `Defaults`
3. Migrate existing HF/GGUF detection into plugin functions (wrapping existing code, not rewriting)
4. Add 4 new plugin functions (Embedding, Reranker, VLM, LoRA)
5. Detection priority: LoRA → SentenceTransformers → Reranker → VLM → HF Chat → GGUF

**Explicitly NOT in Phase B1**:
- Unsupported detectors (ONNX, TensorRT, etc.) — deferred to Phase B2
- UI display changes — deferred to Phase C
- Compatibility checks — deferred to Phase D
- Test method changes — deferred to Phase E

**Acceptance**:
- Embedding models detected with task=embedding, capabilities=["embedding"], default_test_mode="embedding"
- Reranker models detected with task=rerank, capabilities=["rerank"], default_test_mode="rerank"
- Vision-Language detected with task=vision_chat, capabilities=["chat","vision"]
- LoRA detected with deployable=false, requires_base_model=true, unsupported_reason set
- HF/GGUF behavior unchanged (regression gate)
- Tests: at minimum 1 positive case per detector, covering all 6 types
- `go test`, `go vet`, `npm test`, `npm build` ALL PASS

**Risk**: Medium — refactoring detection into plugin functions. Mitigation: wrap existing code paths in plugin functions first, validate with existing tests, then add new plugins.

---

### Phase C: Model Persistence and UI Display / Edit

**Goal**: Persist scanner metadata and display it in model detail and edit pages.

**Changes**:

**Backend — persist scanner metadata**:
1. In `HandleCreateModelLocation`: write scanner metadata into `discovered_metadata_json`:
   ```json
   {
     "kind": "directory",
     "evidence": ["config.json", "tokenizer_config.json"],
     "confidence": "medium",
     "detector_id": "hf_chat",
     "scan_root": "/home/kzeng/models/Qwen3-0.6B-Instruct-2512"
   }
   ```
2. Populate `capabilities_json`, `capability_sources_json`, `default_test_mode` on `model_artifacts` from candidate (Phase A already ensures this)
3. Expose `discovered_metadata_json` through artifact detail API (already returned in location list)

**Frontend detail page**:
1. Show: task type, model format, capabilities, default test mode (already partially done)
2. Show scanner info section: evidence, confidence, detector, scan root
3. Show deployability status badge: "可部署" (green) / "不可独立部署" (orange)
4. Show recommended backends as tags
5. For non-deployable: show unsupported_reason
6. Add collapsible "扫描识别信息" section

**Frontend edit page**:
1. Already supports editing capabilities and default_test_mode (Phase 2)
2. Add task type editing: select with options matching detected task types

**i18n**: Add keys for task types (chat/completion/embedding/rerank/vision_chat/adapter/classification/unknown), deployability, evidence labels

**Explicitly NOT in Phase C**:
- Compatibility checks (Phase D)
- New test modes (Phase E)
- Unsupported model type display (Phase B2)

**Acceptance**:
- Scanner metadata persisted in `discovered_metadata_json`
- Detail page shows task, deployable, recommended backends, evidence
- Embedding/Reranker/Vision labels appear correctly in UI (no "Unknown" for recognized types)
- LoRA shows "不可独立部署" with reason
- No `undefined`/`null`/`[object Object]`/`task.xxx`/`format.xxx` leaks
- Tests: `go test`, `npm test`, `npm build` ALL PASS

**Risk**: Low — additive UI, existing JSON columns

---

### Phase D: Backend Capability and Compatibility Checker

**Goal**: Express backend capabilities and block invalid model-backend combinations at preflight.

**Blocking rules — all must BLOCK** (not warn):
- **format mismatch**: model format ∉ backend supported_formats → FAIL
- **path mode mismatch**: model path_type ≠ backend model_path_mode requirement → FAIL
- **deployable=false**: model is not deployable → FAIL
- **task mismatch**: model task ∉ backend supported_tasks → FAIL (unless backend explicitly declares compatibility alias or fallback)

**Changes**:

**Backend capability** (seed data only, no schema change):
1. Enrich `backend_versions.capabilities_json` for each backend:
   ```json
   // vLLM / SGLang
   {
     "supported_formats": ["huggingface", "sentence_transformers"],
     "supported_tasks": ["chat", "completion", "embedding", "rerank", "vision_chat"],
     "supported_capabilities": ["chat", "completion", "embedding", "rerank", "vision"],
     "model_path_modes": ["directory"]
   }
   // llama.cpp
   {
     "supported_formats": ["gguf"],
     "supported_tasks": ["chat", "completion"],
     "supported_capabilities": ["chat", "completion"],
     "model_path_modes": ["file"]
   }
   ```

**CompatibilityChecker** (`internal/server/runplan/compat.go` — new file):
1. Input: ModelDescriptor (format, task, deployable, path_type) + BackendDescriptor (capabilities_json)
2. Output: CompatResult{Compatible bool, Severity string, Reason string}
3. Check order: deployable → format → path_mode → task
4. On any failure: return structured error with Chinese message

**Preflight integration** (`deployment_lifecycle_handlers.go`):
1. Call CompatibilityChecker after model location validation, before RunPlan resolution
2. On failure: add structured preflight error, **block deployment**
3. Frontend preflight UI shows compatibility error

**Acceptance**:
- vLLM/SGLang + GGUF file → preflight FAIL: "模型为 GGUF 文件，vLLM/SGLang 不支持。请使用 llama.cpp。"
- llama.cpp + HF directory → preflight FAIL: "模型为 HuggingFace 目录，llama.cpp 不支持。请使用 vLLM/SGLang。"
- LoRA standalone deploy → preflight FAIL
- deployable=false models → preflight FAIL
- Embedding + vLLM → preflight PASS
- Reranker + vLLM → preflight PASS
- GGUF + llama.cpp → preflight PASS
- Tests: 8+ compatibility unit tests (all pass/fail scenarios)
- `go test`, `npm test`, `npm build` ALL PASS

**Risk**: Medium — preflight logic change. Mitigation: add as new validation step, not replacing existing.

---

### Phase E: Test Method Abstraction

**Goal**: Support embedding and rerank test modes.

**Changes**:

**Frontend** (`ModelInstancesPage.vue`):
1. Add `embedding` and `rerank` to test mode selector
2. `recommendedTestMode()` already handles these

**Backend** (`deployment_lifecycle_handlers.go`):
1. `tryEmbeddingInference()`: POST `/v1/embeddings` with `{"input": "hello world", "model": "..."}`
2. `tryRerankInference()`: POST to declared rerank endpoint with `{"query": "what is gpu", "documents": [...]}`
3. Read endpoint from `backend_versions.capabilities_json.test_endpoints` (from Phase D)
4. Rerank endpoint not declared → return diagnostic: "该模型识别为 Reranker，但当前运行后端未声明 Rerank 测试端点。"
5. Update `tryInferenceWithMode()` to dispatch embedding/rerank

**No**: Schema changes, endpoint probing, new backend columns

**Acceptance**:
- Test selector: auto/chat/completion/embedding/rerank (5 modes)
- Embedding model defaults to embedding test mode
- Reranker model defaults to rerank test mode
- Embedding test calls `/v1/embeddings`
- Rerank test calls declared endpoint; undelcared → clear diagnostic
- Chat/completion test unchanged (regression gate)
- Tests: go test, npm test, npm build ALL PASS

**Risk**: Low — additive, no existing behavior changes

---

### Phase B2: Recognized-but-Unsupported Model Type Plugins

**Goal**: Add detectors for model types that the platform cannot currently run. These models enter the model library for visibility but are marked as unsupported.

**Detectors added** (7):

| # | Detector | Evidence | deployable | unsupported_reason |
|---|----------|----------|------------|-------------------|
| 7 | ONNX | *.onnx | false | "当前平台尚未配置 ONNX Runtime 后端。" |
| 8 | TensorRT Engine | *.engine, rank*.engine | false | "当前平台尚未配置 TensorRT-LLM 后端。" |
| 9 | OpenVINO | *.xml + *.bin | false | "当前平台尚未配置 OpenVINO 后端。" |
| 10 | Diffusers | model_index.json, unet/ | false | "当前平台尚未配置 Diffusers/Image Generation 后端。" |
| 11 | ASR | name patterns (whisper, funasr, paraformer) | false | "当前平台尚未配置 ASR 后端。" |
| 12 | TTS | name patterns (cosyvoice, chattts, gpt-sovits) | false | "当前平台尚未配置 TTS 后端。" |
| 13 | Classification | config architectures: SequenceClassification/TokenClassification | false | "当前平台尚未配置分类模型服务后端。" |

**These plugins are registered in the same table as B1**, with priority between LoRA (adapter, needs base model) and GGUF (file scan). All have `Deployable=false` with clear `UnsupportedReason`.

**Acceptance**:
- All 7 types detected and create valid candidates
- All have deployable=false with clear unsupported_reason
- Models enter model library (Phase C already handles display)
- Preflight blocks deployment (Phase D already blocks deployable=false)
- Tests: 1 positive case per detector
- `go test`, `npm test`, `npm build` ALL PASS

**Risk**: Low — purely additive, no existing behavior changes

---

### Phase F: Hardening and Closeout

**Goal**: Complete tests, docs, regression verification, clean commit.

**Changes**:
1. Full regression suite:
   - `go test ./internal/server/api/...`, `go test ./internal/server/runplan/...`, `go vet ./...`
   - `npm test`, `npm run build`
2. Run regression gates (see §8)
3. Create closeout document
4. Final commit and push

## 8. Regression Test Gates

Every phase must pass these regression assertions — they are non-negotiable:

| # | Assertion | Introduced |
|---|-----------|-----------|
| R1 | Direct GGUF file selection RunPlan: `-m` points to `.gguf` file | RC-001/003/005/006 |
| R2 | Directory scan GGUF selection RunPlan: `-m` points to selected `.gguf` file | RC-006 |
| R3 | HF directory with vLLM/SGLang: path mode is directory | baseline |
| R4 | llama.cpp + HF directory → preflight FAIL | (new in Phase D) |
| R5 | vLLM/SGLang + GGUF file → preflight FAIL | (new in Phase D) |
| R6 | deployable=false → cannot generate RunPlan | (new in Phase D) |
| R7 | i18n: no `task.xxx` / `format.xxx` / `capability.xxx` / `status.xxx` leaks | MV-007 |
| R8 | `go test`, `go vet`, `npm test`, `npm build`, `git diff --check` ALL PASS | always |

## 9. Phase Acceptance Criteria Summary

| Phase | New Detectors | UI Changes | Compat Checks | Test Modes | Risk |
|-------|--------------|------------|---------------|------------|------|
| A | 0 (enrich existing) | Wizard field passthrough only | No | No | Low |
| B1 | 6 (HF, GGUF, Embedding, Reranker, VLM, LoRA) | No | No | No | Medium |
| C | 0 | Detail: task/evidence/deployable; Edit: task type | No | No | Low |
| D | 0 | Preflight error display | Yes (block, not warn) | No | Medium |
| E | 0 | Test selector: +embedding/rerank | No | Yes (embedding/rerank) | Low |
| B2 | 7 (ONNX, TRT, OpenVINO, Diffusers, ASR, TTS, Classification) | No (reuses C) | No (reuses D) | No | Low |
| F | 0 | 0 | Regression gates | Regression gates | Low |

## 10. Decisions (Formerly Open Questions)

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | **Plugin style: lightweight struct + function table** (`[]ModelTypePlugin`), not Go interface | Matches project's existing plain-function style; no interface abstraction overhead |
| D2 | **Backend capability schema: reuse `backend_versions.capabilities_json`** with structured sub-fields (`supported_formats`, `supported_tasks`, `supported_capabilities`, `model_path_modes`, `test_endpoints`). No new columns. | Existing column is already a JSON object; structured sub-fields are backward-compatible |
| D3 | **LoRA/Adapter: enter model library, deployment UI grays out with "需要基础模型，不能独立部署"** | Provides visibility; prevents accidental deployment; consistent with other unsupported types |
| D4 | **Unsupported models: enter model library, marked "当前不支持运行", display unsupported_reason** | Provides visibility of what was scanned; enables future backend additions to automatically unlock these models |
| D5 | **Rerank endpoint: use backend-declared endpoint from `capabilities_json.test_endpoints.rerank`; if not declared, return clear diagnostic** | Deterministic; no blind probing; backend declaration is the single source of truth |
| D6 | **Phase ordering: A → B1 → C → D → E → B2 → F** | B1 delivers immediate value (embedding/reranker/VLM recognition); C provides UI visibility before D enforces constraints; B2 is low-priority (unsupported types); E enables test UX |
| D7 | **Compatibility: format/path_mode/deployable/task mismatch → BLOCK (never warn)** | Prevent silent wrong behavior; user must explicitly choose compatible combination |

## 11. Test Plan

### New Tests Required Per Phase

**Phase A**: No new tests (struct changes only; existing tests verify regression)

**Phase B1**: ~8 tests — 1 per detector + mixed + empty
- `TestDetectHuggingFaceChat`, `TestDetectGGUFFile` (regression)
- `TestDetectSentenceTransformers`, `TestDetectReranker`, `TestDetectVisionLanguage`
- `TestDetectLoRAAdapter`
- `TestEmptyDirectory`, `TestMixedHFAndGGUF`

**Phase C**: No new backend tests (UI only). Update existing wizard/model tests if needed.

**Phase D**: ~10 tests
- `TestCompatVLLMWithGGUFFails`, `TestCompatSGLangWithGGUFFails`
- `TestCompatLlamaCppWithHFFails`, `TestCompatLlamaCppWithEmbeddingFails`
- `TestCompatLoRAStandaloneFails`
- `TestCompatDeployableFalseFails` (ONNX/TensorRT/OpenVINO/Diffusers/etc.)
- `TestCompatFilePathModeWithDirectoryFails`
- `TestCompatVLLMWithHFPasses`, `TestCompatLlamaCppWithGGUFPasses`
- `TestCompatEmbeddingWithVLLMPasses`, `TestCompatRerankerWithVLLMPasses`

**Phase E**: ~4 tests
- `TestEmbeddingTestEndpoint`, `TestRerankTestEndpoint`
- `TestRerankNoEndpointDeclared`, `TestChatCompletionUnchanged`

**Phase B2**: ~7 tests — 1 per unsupported type
- `TestDetectONNX`, `TestDetectTensorRT`, `TestDetectOpenVINO`
- `TestDetectDiffusers`, `TestDetectASR`, `TestDetectTTS`, `TestDetectClassification`

### Verification Commands (all phases)

```bash
gofmt -w cmd/ internal/
go test ./internal/server/api/...
go test ./internal/server/runplan/...
go test ./internal/agent/...        # if agent scanner tests exist
go vet ./...
npm --prefix web test
npm --prefix web run build
git diff --check
git status --short
```

## 12. Risk and Rollback Notes

| Risk | Mitigation |
|------|-----------|
| Plugin refactoring breaks HF/GGUF | Wrap existing code in plugin functions; validate with existing tests before adding new plugins |
| Preflight compatibility blocks valid deployments | Feature-flag: skip compat check if `capabilities_json` lacks `supported_formats` (backward compat) |
| New candidate fields confuse existing API consumers | Add fields with zero-value defaults; existing fields unchanged |
| Embedding/rerank test not testable without running instance | Unit-test endpoint dispatch; runtime validation deferred to manual testing |
| Seed data update breaks existing DB | Use REPLACE pattern (like V26) to update `capabilities_json`; no new columns |

## 13. Explicit Non-goals

1. No resource parameter editor (Phase 3)
2. No multi-replica/cross-node scheduling
3. No Playwright specs
4. No API Gateway/API Key
5. No new backends (ONNX Runtime, TensorRT-LLM, OpenVINO, Diffusers, ASR, TTS)
6. No model conversion
7. No LoRA merge
8. No image/audio upload test UI
9. No new schema columns (default; only if existing JSON columns prove insufficient)
10. No backward compatibility for old data
11. No endpoint probing mechanism for rerank/embedding (use declared endpoints only)

## 14. Modified Files Summary (Expected)

| Phase | Files |
|-------|-------|
| A | `internal/agent/collector/model_scanner.go`, `web/src/pages/ModelArtifactsPage.vue` |
| B1 | `internal/agent/collector/model_scanner.go` (plugins + detectors), new test files |
| C | `internal/server/api/artifact_handlers.go`, `web/src/pages/ModelArtifactsPage.vue`, `web/src/locales/zh-CN.ts`, `web/src/locales/en-US.ts` |
| D | `internal/server/runplan/compat.go` (new), `internal/server/api/deployment_lifecycle_handlers.go`, `internal/server/db/db.go` (seed) |
| E | `internal/server/api/deployment_lifecycle_handlers.go`, `web/src/pages/ModelInstancesPage.vue` |
| B2 | `internal/agent/collector/model_scanner.go` (7 new plugins), new test file |
| F | Closeout doc, no code |

## 15. Recommended First Prompt After Approval

```
Proceed to Phase A: enrich ScanCandidate struct with design fields.
Only add fields and populate them for existing HF/GGUF detection.
No new detectors, no UI changes, no compatibility checks.
Verify regression gates R1-R3 and R8 before moving to Phase B1.
```
