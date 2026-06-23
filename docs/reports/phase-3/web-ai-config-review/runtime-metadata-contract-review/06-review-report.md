# Runtime Metadata Contract Review — Review Report & Staged Implementation Plan

> **Date:** 2026-06-23
> **Status:** Design review complete. No code modified.
> **User decisions:** Trim enums to current+near-term only. Constants live in `internal/server/runplan/`. Frontend inference kept as fallback.

## Context

LightAI Go currently has a working model detection → backend compatibility → deployment pipeline with 7 verified capability combinations (HF+vLLM, HF+SGLang, GGUF+llama.cpp, Embedding+vLLM, Reranker+vLLM, wrong-combination blocking, unsupported-asset recognition). The design documents in this directory propose a unified, structured contract system with 5 key types (`ModelTypeProfile`, `DiscoveredMetadata`, `RuntimeRequirements`, `BackendCapabilityProfile`, `ResolvedBackendCapability`) and 16-point computable matching.

This report reviews the design against the current implementation, identifies hardcodes, and recommends a staged approach.

---

## 1. Documentation Design Understanding

The 5-document set (01–05) proposes a layered contract architecture:

| Layer | Purpose | Current Code Parallel |
|-------|---------|----------------------|
| `ModelTypeProfile` | Type-level detection rules (files, config keys, defaults) | `ModelTypePlugin` + `ModelTypeDefaults` in `internal/agent/collector/model_scanner.go` |
| `DiscoveredMetadata` | Per-location scan result against a profile | `ScanCandidate` struct (same file) |
| `RuntimeRequirements` | Model-side demand: what a model needs to run | Distributed across `ModelDescriptor` (runplan/compat.go), plugin defaults, and implicit conventions |
| `BackendCapabilityProfile` | Backend-side supply: what a version/runtime provides | `BackendDescriptor` (runplan/compat.go) + `capabilities_json` in DB seed (db/db.go:1378-1383) |
| `ResolvedBackendCapability` | Merged node+runtime capability | No explicit struct; merged implicitly in `buildRunPlan()` (runplan/resolver.go) |

Core matching: `RuntimeRequirements × ResolvedBackendCapability → CompatibilityResult`. Current implementation does 6 checks in `CheckCompatibility()` (compat.go:33-91); the document proposes 16.

---

## 2. Design Reasonableness Assessment

**The design direction is correct.** The separation of "model needs" from "backend provides" is conceptually sound and aligns with how the current code already works (ModelDescriptor vs BackendDescriptor). The layered resolution (BackendVersion → BackendRuntime → NodeBackendRuntime → Resolved) matches the existing DB schema and catalog design.

**Key strengths:**
- Clear "don't put X in Y" boundary rules (10 design bottom lines in doc 01)
- Unified enum vocabulary for formats/tasks/capabilities — reduces string duplication
- Explicit arg_support / arg_mappings concept addresses a real gap
- BlockedArchitectures pattern already proven in production (InternVL case)

**Key weaknesses:**
- 18 RuntimeFeatures, 18 AbstractArgs, 21 CompatibilityStatuses, 8 Modalities, 9 ServingProtocols — for a system with 4 active backends and ~7 model types, ~60% of these enum values will never be used
- The 16-point match adds 10 checks that have zero current use cases
- `VerificationRecord` / `EvidenceIndex` as Go types is premature — catalog `verification_json` status field is sufficient

---

## 3. Boundary Correctness Review

### ModelTypeProfile ↔ DiscoveredMetadata: **Correct**

The document correctly separates "how to detect" (profile) from "what was found" (discovered). Current implementation mirrors this: `ModelTypePlugin.Detect()` is the profile rule, `ScanCandidate` is the discovered result.

**Gap:** The document envisions ModelTypeProfile as declarative config (required files, config key/value matches, exclude patterns). Current implementation has detection logic hardcoded in Go functions (`DetectSentenceTransformers()`, `DetectReranker()`, etc.). Name-based keyword matching (e.g., `"bge", "e5", "gte"`) lives in code, not config. This is a legitimate design choice for a Go binary but differs from the document's vision. Tracked as **BLOCKER-004**.

### RuntimeRequirements ↔ BackendCapabilityProfile: **Correct boundary, partial matchability**

Both sides use structured string lists. The current 6-point check (format, task, path mode, deployable, architecture block, capability declared) covers all verified production cases. The remaining 10 document-proposed checks (modalities, serving protocols, runtime features, arg requirements, etc.) have **zero current use cases** — all currently supported model types use `text` modality and `openai-compatible` protocol implicitly.

**Recommendation:** Keep the 6-point check. Add the remaining checks incrementally as new model types (ASR, TTS, Diffusers) get backend support. Do NOT front-load all 16 checks now.

### BackendCapabilityProfile ↔ ResolvedBackendCapability: **Correct layering, over-designed implementation**

The three-layer resolution (BackendVersion → BackendRuntime → NodeBackendRuntime) is correct. Current code does this merge implicitly in the resolver. Creating an explicit `ResolvedBackendCapability` struct would help with testability but is not urgent. Tracked as **BLOCKER-003**.

### DiscoveredMetadata boundary leak: **Minor**

The document says "actual absolute paths live in `model_locations.path`, not duplicated here." Current `ScanCandidate` has its own `Path` field. This is fine for the scanner's internal use — it's the upstream consumer's responsibility to store the path in the location record. No leak in practice.

---

## 4. Hardcode Audit — Complete Results

### Table: All hardcodes found, classified by decision

| # | Area | File:Line | Pattern | Risk | Decision |
|---|------|-----------|---------|------|----------|
| H1 | Format mismatch messages | `runplan/compat.go:119-131` | `formatMismatchMsg()` — hardcoded format→backend recommendation strings | Medium: new format/backend combos need code changes | **Refactor: data-driven from BackendCapabilityProfile** |
| H2 | Scanner name-based detection | `collector/model_scanner.go:374-613` | Keyword lists in `DetectSentenceTransformers`, `DetectReranker`, `DetectVisionLanguage`, `DetectASR`, `DetectTTS`, `DetectClassification` | Medium: new model families need code changes | **Accepted: catalog seed (Go plugin registry is the catalog)** |
| H3 | Frontend capability inference | `web/src/utils/modelCapabilities.js:68-123` | Regex-based capability guessing from model name/metadata | High: duplicates server logic, can diverge | **Refactor: prefer persisted capabilities, inference as fallback only** |
| H4 | Vendor GPU env key | `runplan/resolver.go:901-910` | `defaultVisibleEnvKey()` switch on vendor string | Low: stable mapping, covered by catalog seed | **Validated: constant — but should read from BackendRuntime.docker_json.gpu_visible_env_key** |
| H5 | Device binding vendor switch | `runplan/resolver.go:983-1002` | `buildDeviceBinding()` switch on vendor for mode selection | Medium: new vendors need code changes | **Formal blocker: RUNTIME-CONTRACT-BLOCKER-005** |
| H6 | Path type deduction | `api/deployment_lifecycle_handlers.go:886-888` | `if modelFormat == "gguf" { modelPathType = "file" }` | High: fragile, should use stored path_type | **Refactor: use model artifact's stored path_type** |
| H7 | Backend family matching | `api/runtime_handlers.go:997-1001` | `matchBackendType()` patterns map | Low: covers user input normalization | **Accepted: catalog seed — patterns should derive from backend catalog** |
| H8 | Server allowed-values maps | `api/artifact_handlers.go:17-39` | `allowedCapabilities`, `allowedTaskTypes`, `allowedTestModes`, `allowedCapabilitySources` | Low: validation enums | **Refactor: extract to shared constants package** |
| H9 | Frontend hardcoded option lists | `web/src/pages/ModelArtifactsPage.vue:368-399` | `formatOptions`, `TASK_TYPE_OPTIONS`, `CAPABILITY_OPTIONS`, `TEST_MODE_OPTIONS` | Medium: UI diverges from server validation | **Refactor: derive from API or shared constants** |
| H10 | Capabilities JSON in DB seed | `db/db.go:1378-1383` | Inline JSON strings for vLLM/SGLang/llama.cpp/Ollama capabilities | Low: this IS the catalog seed | **Validated: catalog seed** |
| H11 | HF Chat dedup logic | `collector/model_scanner.go:688-709` | Hardcoded format/task strings in dedup filter | Medium: new model types need dedup rule updates | **Accepted: catalog seed — dedup logic is inherent to priority ordering** |
| H12 | Entrypoint shape classification | `runplan/detection.go:13-46` | `ClassifyEntrypointShape()` — string heuristics for entrypoint type | Low: stable heuristic | **Validated: constant** |
| H13 | Frontend `formatTestFailure()` | `web/src/utils/modelCapabilities.js:149-192` | Hardcoded Chinese error messages with endpoint/status formatting | Medium: error messages should come from API, not frontend | **Refactor: API returns structured error, frontend formats** |
| H14 | Capability labels map | `web/src/utils/modelCapabilities.js:1-9` | `CAPABILITY_LABELS` — hardcoded zh/en labels | Low: covered by i18n system | **Validated: constant — prefer i18n locale files** |

### Summary counts

| Decision | Count |
|----------|-------|
| **Refactor now** | H1, H3, H6, H8, H9, H13 (6 items) |
| **Formal blocker** | H5 (1 item) |
| **Validated: constant/seed** | H2, H4, H7, H10, H11, H12, H14 (7 items) |

---

## 5. RuntimeRequirements × BackendCapabilityProfile Landing Risks

1. **Vocabulary mismatch risk (HIGH):** Current `capabilities_json` uses endpoint-like keys (`chat_completions`, `embeddings`) in some places and task-like keys (`chat`, `embedding`) in others. The document proposes a unified enum. The migration must reconcile both vocabularies without breaking existing deployments.

2. **Empty capabilities handling (MEDIUM):** Ollama's `capabilities_json` is `["ollama"]` — a bare list, not a structured object. `ParseBackendCapabilities()` would return empty SupportedFormats/SupportedTasks for this, triggering the "backend_capability_missing" error. This is actually correct behavior (Ollama doesn't use the structured contract) but must be explicitly handled.

3. **Version skew (MEDIUM):** Two SGLang versions exist in the catalog: `0.4.6-compatible` (no embedding/rerank in capabilities) and `v0.5.13.post1` (embedding/rerank supported). The capability JSON in the DB seed already reflects this correctly, but the YAML version files on disk (`configs/backend-catalog/versions/`) have inconsistent capability formats — some use endpoint names, some use structured JSON. The canonical source is the DB seed, not the YAML files.

4. **Test endpoints as capability proxy (LOW):** The `test_endpoints` field in capabilities_json is the only place where concrete HTTP paths are stored. Changing these paths would require updating both the DB seed and the TestDispatcher. The document's proposed `required_test_endpoints` in RuntimeRequirements + `test_endpoints` in BackendCapabilityProfile correctly models this as a two-sided contract.

---

## 6. ResolvedBackendCapability Implementation Difficulties

1. **Merge semantics are underspecified (MEDIUM):** When BackendVersion says `supported_tasks: [chat, completion]` and BackendRuntime overrides nothing, and NodeBackendRuntime adds a GPU-specific limitation — what's the merge rule? Union? Intersection? Override? The document says "merged" but doesn't specify the algorithm.

2. **Currently no runtime-level capability overrides (LOW):** All current BackendRuntime records use BackendVersion's capabilities as-is. No merge is actually needed today. This abstraction would be used when, e.g., a MetaX-specific runtime can't support `vision` even though the base vLLM version can.

3. **Node-level capabilities don't exist yet (LOW):** NodeBackendRuntime currently carries docker/image/env overrides, not capability overrides. Adding capability overrides at the node level would require DB schema changes.

---

## 7. Over-Design, Missing Items, Unreasonable Boundaries

### Over-designed

| Item | Why | Recommendation |
|------|-----|----------------|
| 18 RuntimeFeatures enum values | Only `tensor_parallel` has a current use case | Reduce to 5–6: `tensor_parallel`, `pipeline_parallel`, `flash_attention`, `quantized_cache`, `speculative_decoding` |
| 18 AbstractArgs enum values | Current code uses backend-specific CLI args directly; abstract arg layer is premature | Start with 4: `model_path`, `host`, `port`, `served_model_name` |
| 21 CompatibilityStatuses | 6 statuses (`ok`, `format_mismatch`, `task_mismatch`, `path_mode_mismatch`, `architecture_blocked`, `not_deployable`, `backend_capability_missing`) cover all current cases | Keep current codes; add more when needed |
| `VerificationRecord` / `EvidenceIndex` types | Production E2E evidence is tracked in closeout docs, not needed as code types | Defer until automated E2E test harness exists |
| 16-point compatibility matching | 6-point check covers all current production cases | Keep 6-point; add new checks only when the corresponding model type gets backend support |

### Missing

| Item | Impact | Priority |
|------|--------|----------|
| Unified Go constants/enums for format, task, capability, path mode | Current string duplication across files creates drift risk | **Stage 2** |
| Arg mapping concept in resolver (abstract arg → backend CLI) | RunPlan currently constructs args directly from seed; no abstraction layer | **Formal blocker** |
| Modality concept in model detection | Not needed until ASR/TTS/Diffusers/VLM go beyond `deployable=false` | **Defer** |
| Serving protocol enum | Currently implicit (`openai-compatible` vs `ollama`); hardcoded in backend catalog | **Stage 2 — add constant, no behavioral change** |

### Unreasonable boundaries

1. **"BackendCapabilityProfile must NOT contain GPU vendor/hardware info"** — but `blocked_architectures` is hardware-adjacent (InternVL block is about tokenizer compatibility, which varies by runtime environment). **Clarification:** Architecture blocking is a BackendVersion-level concern (applies to all runtimes of that version). If a block is specific to a vendor runtime, it belongs in BackendRuntime, not BackendVersion. Current implementation puts it in capabilities_json on BackendVersion — this is correct for the InternVL case (it fails on all vLLM/SGLang runtimes).

2. **"RuntimeRequirements uses abstract args only"** — correct goal, but the abstract arg vocabulary must be small. The document lists 18 abstract args; only 4 (`model_path`, `host`, `port`, `served_model_name`) are universal. The remaining 14 are backend-specific concepts dressed as abstract names.

---

## 8. Which Code Modules Already Conform

| Module | Conformance | Notes |
|--------|-------------|-------|
| `runplan/compat.go` `CheckCompatibility()` | ✅ High | Clean 6-point check, struct-based input, no hardcoded backend names in logic |
| `runplan/compat.go` `ParseBackendCapabilities()` | ✅ High | Clean JSON→struct deserialization, nil-safe |
| `runplan/compat_test.go` | ✅ High | 15 test cases including architecture blocking |
| `collector/model_scanner.go` plugin registry | ⚠️ Medium | Plugin pattern is correct; name-based detection is catalog-embedded-in-code |
| `collector/model_scanner.go` `ScanCandidate` | ⚠️ Medium | Rich struct but DetectedMetadata is `map[string]interface{}` |
| `db/db.go` capabilities_json seed | ✅ High | Structured JSON per version, blocked_architectures included |
| `api/artifact_handlers.go` validation maps | ⚠️ Medium | Correct validation but hardcoded string sets |

---

## 9. Which Code Modules Conflict with the Design

| Module | Conflict | Severity |
|--------|----------|----------|
| `web/src/utils/modelCapabilities.js` `inferModelCapabilities()` | Duplicates server-side detection logic with regex | **Medium** — can show wrong capabilities in UI |
| `api/deployment_lifecycle_handlers.go:886-888` | Guesses path_type from format instead of using stored value | **Medium** — fragile, breaks for new formats |
| `runplan/compat.go:119-131` `formatMismatchMsg()` | Special-case format→backend mapping | **Low** — UI-only, but should be data-driven |
| `runplan/resolver.go:901-910,983-1002` | Vendor switch statements | **Medium** — new vendors need code changes |
| Frontend `formatTestFailure()` | Constructs error messages client-side from raw API errors | **Low** — UX issue, not data integrity |

---

## 10. Staged Implementation Recommendations

### Stage 0 — This Review ✅ (Complete)
- [x] Read all design documents
- [x] Audit codebase for hardcodes and boundary violations
- [x] Produce this review report
- **Output:** This document (06-review-report.md)

### Stage 1 — Contract Documentation (No Code)
1. Write `docs/design/model-runtime-contract-and-backend-capability-profile.md`
   - Document current state: what's working, what types exist, where they live
   - Map document-proposed types to current implementation types
   - Explicitly list the 6 compatibility checks
   - Mark 10 deferred checks as "future extension points"
   - Trim enum lists to current+near-term values only
2. Write `docs/design/model-runtime-mainstream-matrix.md`
   - Current 14 model types matrix with verified/unverified status
   - 4-backend runtime matrix with actual capability JSON excerpts

### Stage 2 — Minimal Go Constants in `internal/server/runplan/` (Low Risk)
1. Add `constants.go` to `internal/server/runplan/` with:
   - Format constants (9 values)
   - Task constants (11 values)
   - Capability constants (7 values)
   - PathMode constants (2 values)
   - CapabilitySource constants (4 values)
   - TestMode constants (5 values)
   - ServingProtocol constants (3 values)
2. Add validation functions: `IsValidFormat()`, `IsValidTask()`, `IsValidCapability()`, etc.
3. Replace `allowedCapabilities`/`allowedTaskTypes`/`allowedTestModes` maps in `artifact_handlers.go` with calls to validation functions.
4. Update `model_scanner.go` plugin defaults to use string constants.

### Stage 3 — Targeted Hardcode Refactors (6 items)

| Priority | Item | File(s) | Effort |
|----------|------|---------|--------|
| **P0** | H6: Use stored path_type instead of guessing from format | `deployment_lifecycle_handlers.go:886-888` | 1 line |
| **P0** | H3: Frontend `inferModelCapabilities` — prefer persisted, inference as fallback | `web/src/utils/modelCapabilities.js:68-80` | ~5 lines |
| **P1** | H1: `formatMismatchMsg` — derive recommendation from BackendDescriptor fields | `runplan/compat.go:119-131` | ~20 lines |
| **P1** | H8: Extract validation maps to use shared constants | `artifact_handlers.go` | ~10 lines |
| **P2** | H9: Frontend option lists — derive from API or shared constants | `ModelArtifactsPage.vue` | ~15 lines |
| **P2** | H13: `formatTestFailure` — use API-structured error | `modelCapabilities.js` | ~30 lines |

### Stage 4 — Tests
1. Add tests for new validation functions (valid + invalid inputs)
2. Add test for `formatMismatchMsg` with data-driven backend descriptor
3. Update frontend tests for `inferModelCapabilities` — verify persisted capabilities take precedence
4. Add path_type round-trip test in deployment lifecycle
5. Ensure all 7 existing test suites pass

### Stage 5 — Closeout
Write `closeout.md` in this directory with final audit results, formal blocker list, verification results, and remaining risks.

---

## 11. Formal Blockers

### RUNTIME-CONTRACT-BLOCKER-001: RunPlan Arg Abstraction Layer

| Field | Value |
|-------|-------|
| **Area** | `runplan/resolver.go` — arg construction |
| **Current behavior** | BackendVersion seed data contains backend-specific CLI args (`--model`, `--model-path`, `-m`) directly in `default_args_json` and `parameter_defs_json`. The resolver substitutes template variables but does not translate abstract args to backend-specific forms. |
| **Why not fixed now** | Requires redesign of parameter_defs_json schema, arg_mappings in BackendCapabilityProfile, and resolver arg construction. Touches DB seed, all 4 backends, RunPlan resolution, and Docker command generation. |
| **Risk** | Adding a new backend with different CLI conventions requires manual seed data construction with no abstraction safety net. |
| **Required future fix** | Add `arg_mappings` to BackendCapabilityProfile (or BackendVersion). Define 4 abstract args: `model_path`, `host`, `port`, `served_model_name`. Resolver translates abstract→concrete per backend. |
| **Validation** | Deploy HF chat on vLLM and llama.cpp — verify args are correctly translated for each backend. |

### RUNTIME-CONTRACT-BLOCKER-002: Full RuntimeRequirements Struct

| Field | Value |
|-------|-------|
| **Area** | Model-side requirement expression |
| **Current behavior** | Model requirements are implicit in plugin defaults and `ModelDescriptor` (6 fields). No modalities, serving protocols, runtime features, or arg requirements. |
| **Why not fixed now** | All current model types use text modality + openai-compatible protocol implicitly. Adding explicit modality/protocol fields with no consumers is dead code. |
| **Risk** | When ASR/TTS/Diffusers/VLM get backend support, their modality/protocol requirements will need to be expressed. |
| **Required future fix** | Add `Modalities` (input/output), `ServingProtocols`, and `RuntimeFeatures` fields to ModelDescriptor when the first non-text model type gets backend support. |

### RUNTIME-CONTRACT-BLOCKER-003: ResolvedBackendCapability Formal Merge

| Field | Value |
|-------|-------|
| **Area** | Capability resolution across BackendVersion → BackendRuntime → NodeBackendRuntime |
| **Current behavior** | No explicit merge — BackendVersion's capabilities_json is used directly. BackendRuntime and NodeBackendRuntime carry docker/env overrides but no capability overrides. |
| **Why not fixed now** | No current use case for runtime-level or node-level capability overrides. All runtime variants of the same backend version share capabilities. |
| **Risk** | When a vendor-specific runtime can't support a capability (e.g., MetaX vLLM can't do vision), there's no mechanism to express this. |
| **Required future fix** | Add `capability_overrides_json` to BackendRuntime and NodeBackendRuntime. Define merge semantics: intersection for supported_*, override for blocked_*. |

### RUNTIME-CONTRACT-BLOCKER-004: ModelTypeProfile Config-Driven Detection

| Field | Value |
|-------|-------|
| **Area** | `collector/model_scanner.go` — model type detection |
| **Current behavior** | Detection rules (required files, name patterns, config key checks) are hardcoded in Go `Detect*()` functions. |
| **Why not fixed now** | Moving detection rules to YAML/JSON config would require a config loader, validation, and migration of 13 detector functions. The current Go plugin pattern is working and testable. |
| **Risk** | Adding a new model type requires Go code changes and recompilation. |
| **Required future fix** | Define ModelTypeProfile as a config struct. Each profile specifies: required_files[], config_key_matches{}, exclude_patterns[], defaults{}. Load from embedded config or DB. |

### RUNTIME-CONTRACT-BLOCKER-005: Vendor Device Binding Abstraction

| Field | Value |
|-------|-------|
| **Area** | `runplan/resolver.go:983-1002` — `buildDeviceBinding()` |
| **Current behavior** | Switch on vendor string to select binding mode. NVIDIA → `nvidia_device_request`, MetaX → `metax_device_paths`, CPU → `cpu_none`. |
| **Why not fixed now** | The binding modes are fundamentally different (GPU driver API vs device path mounts vs none). Abstracting this requires a plugin-like device binding interface. For 3 vendors, the switch is maintainable. |
| **Risk** | Adding a new vendor (e.g., AMD ROCm) requires code changes. |
| **Required future fix** | Define `DeviceBindingStrategy` interface. Each vendor registers its strategy. Switch becomes a registry lookup. |

---

## 12. Resolved Design Decisions

| # | Question | Decision |
|---|----------|----------|
| 1 | Enum scope | **Trim to current+near-term only** (~6 features, ~4 abstract args, ~4 modalities, ~3 protocols). Add more when needed. |
| 2 | Scanner detection rules | **Keep in code** (accepted as catalog seed). Documented as BLOCKER-004. |
| 3 | Constants package location | **`internal/server/runplan/`** — co-located with existing types. |
| 4 | Frontend `inferModelCapabilities` | **Keep as fallback** — prefer persisted capabilities from server; use regex inference only when capabilities are empty. |
| 5 | Stage 1 documentation | **Pending** — write canonical design doc; relationship to review docs TBD. |

---

## 13. Verification Plan (Post-Implementation)

After Stages 2–4 are implemented:

```bash
# Go tests
cd internal/server/runplan && go test -v ./...
cd internal/agent/collector && go test -v ./...
cd internal/server/api && go test -v ./...

# Frontend tests
cd web && npm test

# Format check
gofmt -l internal/

# Git status — should be clean except intentional changes
git status
git diff --stat
```

---

## Summary

| Aspect | Verdict |
|--------|---------|
| Design direction | ✅ Correct — layered contracts, clear boundaries |
| Boundary definitions | ✅ Correct with minor clarifications needed |
| Computable matching | ✅ 6-point check covers current needs; 16-point is future work |
| Hardcodes needing refactor | 6 items (H1, H3, H6, H8, H9, H13) — all low-risk, ~80 lines total |
| Hardcodes accepted as seed/constant | 7 items (H2, H4, H7, H10, H11, H12, H14) |
| Formal blockers | 5 items (BLOCKER-001 through BLOCKER-005) |
| Over-design | Enum counts too high; defer VerificationRecord/EvidenceIndex types; defer 10 of 16 match checks |
| Missing | Unified Go constants, arg abstraction (blocked), modality concept (deferred) |
| Implementation risk | Low — all recommended refactors are surgical, no schema changes, no new backends |
