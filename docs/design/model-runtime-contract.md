# Model Runtime Contract — Design Document

> **Status:** Current implementation baseline. Covers Phases 0–3 (Batch 4).
>
> Defines the canonical source architecture, compatibility check contract, and layered backend capability design for LightAI Go.

## 1. Canonical Source Architecture

LightAI Go's model runtime metadata has multiple representations. The canonical source chain prevents drift:

```
runtimecontract Go constants (internal/runtimecontract/)
    │  Compile-time truth — defines all valid enum values
    │
    ├──→ API validation (artifact_handlers.go)
    │       Consumes IsValidFormat / IsValidTask / etc. for input validation
    │
    ├──→ Scanner plugins (model_scanner.go)
    │       Consumes Format*/Task*/Capability* constants for defaults
    │
    ├──→ DB seed + V27 repair (db.go)
    │       Runtime truth — backend_versions.capabilities_json
    │       Preflight CheckCompatibility reads this
    │
    ├──→ API endpoint (GET /api/v1/model-capabilities)
    │       Consumes AllFormats / AllTasks / AllCapabilities for response
    │       │
    │       └──→ Frontend (ModelArtifactsPage.vue)
    │               Reads from API. Falls back to local defaults on error.
    │
    └──→ Backend catalog YAML (configs/backend-catalog/versions/*.yaml)
            Human-readable mirror. Must match DB seed.
```

**Rules:**
1. Go constants define the enum vocabulary. Add a value → add a constant.
2. DB seed (`backend_versions.capabilities_json`) is the runtime canonical source.
3. Catalog YAML mirrors DB seed. Drift is a bug.
4. Frontend reads from API. Local defaults are fallback only.

## 2. Model Descriptor (Model-Side Contract)

`ModelDescriptor` (in `internal/server/runplan/compat.go`) captures model-level facts for compatibility:

| Field | Type | Source |
|-------|------|--------|
| `Format` | string | Scanner plugin default → `model_artifacts.format` |
| `Task` | string | Scanner plugin default → `model_artifacts.task_type` |
| `Deployable` | bool | Scanner plugin default → `discovered_metadata_json.deployable` |
| `PathType` | string | Scanner or manual → `model_locations.path_type` |
| `Architecture` | string | Scanner → `discovered_metadata_json.architecture` |

**Key constraint:** `PathType` MUST be read from `model_locations.path_type` (persisted column), not inferred from format string. This column exists since schema V13.

## 3. Backend Descriptor (Backend-Side Contract)

`BackendDescriptor` (in `internal/server/runplan/compat.go`) captures what a backend version provides:

| Field | Source |
|-------|--------|
| `SupportedFormats` | `backend_versions.capabilities_json` → `supported_formats` |
| `SupportedTasks` | `backend_versions.capabilities_json` → `supported_tasks` |
| `SupportedCapabilities` | `backend_versions.capabilities_json` → `supported_capabilities` |
| `ModelPathModes` | `backend_versions.capabilities_json` → `model_path_modes` |
| `ServingProtocols` | `backend_versions.capabilities_json` → `serving_protocols` |
| `TestEndpoints` | `backend_versions.capabilities_json` → `test_endpoints` |
| `BlockedArchitectures` | `backend_versions.capabilities_json` → `blocked_architectures` |

All fields parsed by `ParseBackendCapabilities()`. V27 repair (`repairBackendCapabilitiesV27`) ensures all built-in backend versions have the current structured contract.

## 4. Compatibility Check (6-Point Contract)

`CheckCompatibility(model ModelDescriptor, backend BackendDescriptor) → CompatResult`

| # | Check | Fail Code |
|---|-------|-----------|
| 1 | Backend capability declared (`len(SupportedFormats) > 0`) | `backend_capability_missing` |
| 2 | Model is deployable | `not_deployable` |
| 3 | Format matches (`model.Format ∈ backend.SupportedFormats`) | `format_mismatch` |
| 4 | Path type matches (`model.PathType ∈ backend.ModelPathModes`) | `path_mode_mismatch` |
| 5 | Architecture not blocked (`model.Architecture ∉ backend.BlockedArchitectures`) | `architecture_blocked` |
| 6 | Task matches (`model.Task ∈ backend.SupportedTasks`) | `task_mismatch` |

**Current verified combinations:**
- HF Chat + vLLM / SGLang: PASS
- GGUF Chat + llama.cpp: PASS
- Embedding + vLLM / SGLang: PASS
- Reranker + vLLM / SGLang: PASS
- Wrong combinations: BLOCKED
- Unsupported assets (ONNX, TensorRT, etc.): NON_DEPLOYABLE
- Ollama + Ollama: STRUCTURED (configured, pending E2E verification)

## 5. Deferred (Future Extension Points)

The full design proposes 16 compatibility checks. 10 are deferred:

| Check | Deferred Because |
|-------|-----------------|
| Modality compatibility | All current models use text modality |
| Serving protocol compatibility | Implicitly openai-compatible or ollama |
| Runtime features (required/optional/forbidden) | No consumer |
| Arg requirements compatibility | Template variable system already handles this |
| Environment compatibility | Not yet modeled |
| ... (5 more) | No current use case |

**When to add a check:** When the first model type requiring it gets backend support deployed in production (e.g., ASR model needs `audio` input modality).

## 6. Layered Backend Capability Design

```
BackendVersion (backend_versions table)
    │  Generic capability: supported_formats, supported_tasks, test_endpoints,
    │  blocked_architectures, serving_protocols
    │
    └──→ BackendRuntime (backend_runtimes table)
            │  Docker image, args, env, ports, health check, vendor, docker_json
            │  (currently carries docker/env overrides only — no capability overrides)
            │
            └──→ NodeBackendRuntime (node_backend_runtimes table)
                    Node-specific config: image override, devices, vendor-specific flags
```

**Current state:** Only BackendVersion carries capability data. BackendRuntime and NodeBackendRuntime carry deployment configuration only. Capability override at runtime/node level is deferred.

## 7. Ollama Special Modeling

Ollama is not a filesystem model backend:
- `supported_formats: ["ollama"]` — not gguf, even if Ollama serves GGUF internally
- `model_path_modes: ["ollama_managed"]` — models referenced by name/tag, not filesystem path
- `serving_protocols: ["ollama"]` — native API (`/api/chat`, `/api/generate`), not openai-compatible
- `supported_tasks: ["chat", "completion"]` — conservative; embedding deferred until TestDispatcher supports `/api/embeddings`

## 8. Enum Vocabulary

Full vocabulary defined in `internal/runtimecontract/constants.go`:

- **Formats (9):** huggingface, sentence_transformers, gguf, lora_adapter, diffusers, onnx, tensorrt_engine, openvino, ollama
- **Tasks (11):** chat, completion, embedding, rerank, vision_chat, adapter, unknown, image_generation, asr, tts, classification
- **Capabilities (11):** chat, completion, embedding, rerank, vision, image_generation, asr, tts, classification, tool_calling, structured_output
- **PathModes (3):** directory, file, ollama_managed
- **ServingProtocols (2):** openai-compatible, ollama
- **TestModes (5):** auto, chat, completion, embedding, rerank
- **CapabilitySources (4):** scan, inferred, user_override, backend_probe

## 9. Formal Blockers

| ID | Area | Status |
|----|------|--------|
| BLOCKER-001 | RunPlan Arg Abstraction Layer | DEFERRED |
| BLOCKER-004 | ModelTypeProfile Config-Driven Detection | DEFERRED |
