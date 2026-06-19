# 05 — Model Wizard Scan, Metadata, and Preflight Requirements

**Status**: Draft — awaiting implementation
**Date**: 2026-06-19
**Scope**: Model Wizard directory selection, model scanning, metadata extraction, context validation, DryRun/RunPlan validation, E2E hardening

---

## 1. Goals and Scope

This document defines the exact scope of Model Wizard improvements. It is the **sole authority** for implementation. Any deviation must be reflected in this document first.

### In Scope

1. Model Wizard UI: directory selection feedback, scan status display, candidate list, auto-selection for single GGUF
2. Agent-side scanning: multi-candidate scan, GGUF metadata extraction, HF metadata extraction
3. Server-side: scan response with candidates/metadata/warnings, metadata persistence, context-length preflight validation
4. RunPlan/DryRun: GGUF/HF path validation, context parameter validation
5. E2E: scan metadata E2E, UI smoke, DryRun path checks, context preflight checks

### Out of Scope (DO NOT MODIFY)

- RBAC, tenant, billing, quota
- GPU scheduling, GPU lease lifecycle
- Docker runtime driver core lifecycle
- Backend catalog structure changes
- Database migration (large-scale schema changes)
- API route renaming
- Frontend layout, navigation, theme
- Dashboard, Nodes, GPU pages
- Prometheus/Grafana/monitoring
- Packaging/release scripts
- Unrelated existing E2E scripts

---

## 2. Expected Modification File List

### Docs (allowed)
- `docs/reports/model-runtime-node-wizard/e2e-improvement/05-... .md` (this file)
- `docs/reports/model-runtime-node-wizard/e2e-improvement/04-... .md` (update with execution results)

### Server/API (allowed)
- `internal/server/api/artifact_handlers.go` — scan response enrichment, discover artifact metadata
- `internal/server/api/agent_proxy_handlers.go` — pass through scan candidates
- `internal/server/api/deployment_lifecycle_handlers.go` — preflight context validation
- `internal/server/api/helpers.go` — helper functions if needed
- `internal/server/runplan/resolver.go` — path validation (GGUF vs directory)
- `internal/server/models/artifact.go` — minimal additions if needed for metadata

### Agent (allowed)
- `internal/agent/collector/model_scanner.go` — multi-candidate scan, GGUF/HF metadata extraction
- `internal/agent/collector/` — new GGUF metadata reader (new file, minimal)
- `cmd/agent/main.go` — scan endpoint changes if needed

### Web (allowed)
- `web/src/pages/ModelArtifactsPage.vue` — wizard UI improvements
- `web/src/components/RemoteFileBrowser.vue` — minimal if needed
- `web/src/api/models.ts` — TypeScript types for new scan response fields
- `web/src/locales/zh-CN.ts` — new i18n keys
- `web/src/locales/en-US.ts` — new i18n keys

### E2E/Tests (allowed)
- `scripts/e2e/` — new or enhanced E2E scripts
- `scripts/e2e/lib/` — helpers if needed
- `internal/server/runplan/vllm_sglang_nvidia_test.go` — path validation tests
- Go test files directly related to changes

---

## 3. Prohibited Modifications

Unless directly causing a failure in the scope of this work, DO NOT modify:

- RBAC/permission model
- Tenant/billing/quota
- GPU scheduling strategy
- GPU lease lifecycle
- Docker runtime core lifecycle
- Backend catalog structure (except adding ParameterDefs if needed for context)
- Database migrations (except minimal, backward-compatible additions)
- API route renaming
- Frontend global layout, navigation, theme
- Dashboard/Nodes/GPU pages
- Prometheus/Grafana/monitoring
- Packaging/release scripts
- Unrelated old E2E scripts
- Repository-wide formatting

**Out-of-scope findings**: record in §12, do not fix in this round.

---

## 4. UI Behavior Requirements

### 4.1 Directory Selection State Display

After clicking "Select Directory" in the wizard, the page MUST show a persistent, clearly visible info block showing:

| Field | Example |
|-------|---------|
| Agent / Node | `KZ-LAPTOP (node-a068be66…)` |
| Selected scan directory | `/home/kzeng/models/Qwen3.5-9B-Q4` |
| Scan status | Not scanned / Scanning… / Scan complete / Scan failed |
| Scan result count | `Found 1 GGUF file` or `Found 3 GGUF files` or `Found HF model directory` |
| Current selection | `Auto-selected: Qwen3.5-9B-Q4_K_M.gguf` or `Please select a model candidate` |
| Actions | [Rescan] [Change Directory] |

### 4.2 Visual States

**Before scan**:
```
已选择扫描目录：
/home/kzeng/models/Qwen3.5-9B-Q4

点击「扫描模型」检测模型类型和元数据。
```

**Scanning**:
```
正在扫描 /home/kzeng/models/Qwen3.5-9B-Q4 …
[spinner] 检测模型文件中…
```

**Single GGUF (auto-selected)**:
```
扫描完成。发现 1 个 GGUF 文件，已自动选择：
Qwen3.5-9B-Q4_K_M.gguf
格式: GGUF | 量化: Q4_K_M | 上下文长度: 32768
```

**Multiple GGUF (user selection required)**:
```
扫描完成。发现 3 个 GGUF 文件，请选择一个模型文件：
○ Qwen3.5-9B-Q4_K_M.gguf (Q4_K_M, 32768 ctx)
○ Qwen3.5-9B-Q5_K_M.gguf (Q5_K_M, 32768 ctx)
○ Qwen3.5-9B-Q8_0.gguf (Q8_0, 32768 ctx)
```

**HF directory**:
```
扫描完成。检测到 HuggingFace 模型目录。
格式: HuggingFace | 架构: Qwen2ForCausalLM | 上下文长度: 32768
```

**HF + GGUF mixed**:
```
扫描完成。发现多种模型类型，请选择要导入的模型候选：
○ HuggingFace 目录 (Qwen2ForCausalLM, 32768 ctx)
○ Qwen3.5-9B-Q4_K_M.gguf (GGUF, Q4_K_M, 32768 ctx)
```

**Scan failed**:
```
扫描失败：无法访问目录或目录中没有可识别的模型文件。
[重新扫描]
```

### 4.3 i18n Keys Required

| Key | zh-CN | en-US |
|-----|-------|-------|
| `modelWizard.scanDirectory` | "已选择扫描目录" | "Selected scan directory" |
| `modelWizard.scanStatus` | "扫描状态" | "Scan status" |
| `modelWizard.scanNotStarted` | "未扫描" | "Not scanned" |
| `modelWizard.scanning` | "扫描中…" | "Scanning…" |
| `modelWizard.scanComplete` | "扫描完成" | "Scan complete" |
| `modelWizard.scanFailed` | "扫描失败" | "Scan failed" |
| `modelWizard.scanResults` | "扫描结果" | "Scan results" |
| `modelWizard.autoSelected` | "已自动选择" | "Auto-selected" |
| `modelWizard.selectCandidate` | "请选择一个模型文件" | "Please select a model file" |
| `modelWizard.mixedTypes` | "发现多种模型类型，请选择要导入的模型候选" | "Multiple model types found, please select a candidate" |
| `modelWizard.noModelFound` | "未发现可识别的模型文件" | "No recognizable model files found" |
| `modelWizard.contextLength` | "上下文长度" | "Context length" |
| `modelWizard.rescan` | "重新扫描" | "Rescan" |
| `modelWizard.changeDir` | "更换目录" | "Change directory" |
| `modelWizard.ggufFound` | "发现 {n} 个 GGUF 文件" | "Found {n} GGUF file(s)" |
| `modelWizard.hfFound` | "检测到 HuggingFace 模型目录" | "HuggingFace model directory detected" |

---

## 5. Directory Scanning Rules

### 5.1 Scan Trigger

Scan is triggered by clicking "Scan Model" button AFTER selecting a directory. The selected directory is the **scan root** — the scan inspects its contents to determine what model candidates exist.

### 5.2 Candidate Generation Rules

| Scenario | Candidates Generated | Auto-Select? |
|----------|---------------------|--------------|
| Directory contains `config.json` (HF) | 1 HF directory candidate (`path_type=directory`) | Yes (only 1) |
| Directory contains 1 `.gguf` file | 1 GGUF file candidate (`path_type=file`) | Yes |
| Directory contains N>1 `.gguf` files | N GGUF file candidates (`path_type=file`) | No |
| Directory has HF + GGUF | 1 HF + N GGUF candidates | No |
| Directory has neither | 0 candidates → error: "no model found" | — |
| Selected entry is a `.gguf` file | 1 GGUF file candidate | Yes |

### 5.3 Path Rules

- **GGUF final path MUST be**: `path_type=file`, `path=/absolute/path/to/model.gguf`, `format=gguf`
- **GGUF MUST NOT be**: `path_type=directory`, `path=/absolute/path/to/dir`, `format=gguf`
- **HF final path MUST be**: `path_type=directory`, `path=/absolute/path/to/model_dir`, `format=huggingface` or `safetensors`

### 5.4 Validation at Save Time

Before saving the artifact:
1. If `format=gguf` and `path` does not end with `.gguf`: REJECT (already implemented)
2. If `format=gguf` and `path_type=directory`: REJECT
3. If `format=huggingface` and `path` is a file (not directory): REJECT

---

## 6. GGUF Metadata Extraction

### 6.1 Required Fields

Extract from GGUF binary header (reading magic bytes + metadata key-value pairs):

| Field | GGUF Key | Fallback |
|-------|----------|----------|
| `format` | — | `"gguf"` |
| `architecture` | `general.architecture` | from filename heuristics |
| `context_length` | `{arch}.context_length` | `unknown` |
| `embedding_length` | `{arch}.embedding_length` | `unknown` |
| `block_count` | `{arch}.block_count` | `unknown` |
| `vocab_size` | `{arch}.vocab_size` | `unknown` |
| `quantization` | tensor type analysis OR filename | `unknown` |
| `file_size_bytes` | file stat | required |
| `parameter_count` | inferred from metadata if possible | `unknown` |

### 6.2 Quantization Detection

1. **Primary**: Analyze GGUF tensor types from metadata. Count occurrences of each quantization type (e.g., `q4_K`, `q5_K`, `q8_0`, `f16`, `f32`). The dominant weight tensor type determines quantization.
2. **Fallback**: Parse filename for known patterns: `Q2_K`, `Q3_K_S`, `Q3_K_M`, `Q3_K_L`, `Q4_K_S`, `Q4_K_M`, `Q5_K_S`, `Q5_K_M`, `Q6_K`, `Q8_0`, `F16`, `F32`, `IQ1_S`, `IQ2_XXS`, `IQ2_XS`, `IQ3_XXS`, `IQ4_XS`, `IQ4_NL`.
3. **Last resort**: `"unknown"` + warning.

### 6.3 GGUF Header Reading

Read only the header portion of the GGUF file (first ~4KB is usually sufficient for metadata). The GGUF format:
- 4-byte magic: `GGUF` (0x47475546)
- Version: uint32 (2 or 3)
- Tensor count: uint64
- Metadata KV count: uint64
- Metadata KV pairs: each has string key + value type + value

Implementation: write a minimal GGUF header reader. Do not pull in large external dependencies. A single new file `internal/agent/collector/gguf_reader.go` with ~200-300 lines is acceptable.

### 6.4 Warnings

If any field cannot be extracted, record in `warnings`:
```
"warnings": ["context_length not found in GGUF metadata"]
```

**Never fabricate metadata.** If unknown, write `"unknown"`.

---

## 7. HF / Safetensors Metadata Extraction

### 7.1 Files to Read

From the model directory, read and parse:

| File | Fields to Extract |
|------|-------------------|
| `config.json` | `architectures`, `model_type`, `torch_dtype`, `max_position_embeddings`, `hidden_size`, `num_hidden_layers`, `num_attention_heads`, `num_key_value_heads`, `vocab_size`, `rope_scaling`, `quantization_config` |
| `generation_config.json` | `max_length`, `max_new_tokens` |
| `tokenizer_config.json` | `tokenizer_class`, `model_max_length` |
| `model.safetensors.index.json` | weight map for parameter count estimation |

### 7.2 Extracted Fields

| Field | Source | Fallback |
|-------|--------|----------|
| `format` | — | `"huggingface"` or `"safetensors"` |
| `model_type` | `config.json` → `model_type` | `"unknown"` |
| `architectures` | `config.json` → `architectures` | `[]` |
| `torch_dtype` | `config.json` → `torch_dtype` | `"unknown"` |
| `max_position_embeddings` | `config.json` → `max_position_embeddings` | `"unknown"` |
| `rope_scaling` | `config.json` → `rope_scaling` | `null` |
| `hidden_size` | `config.json` → `hidden_size` | `0` |
| `num_hidden_layers` | `config.json` → `num_hidden_layers` | `0` |
| `num_attention_heads` | `config.json` → `num_attention_heads` | `0` |
| `vocab_size` | `config.json` → `vocab_size` | `0` |
| `quantization_config` | `config.json` → `quantization_config` | `null` |
| `file_size_bytes` | directory file size sum | required |
| `parameter_count` | inferred from config or index | `"unknown"` |

### 7.3 Supported Context Length

Derive from HF metadata:
1. If `max_position_embeddings` is set: `supported_context_length = max_position_embeddings`
2. If `rope_scaling` is present with `factor`: `supported_context_length = max_position_embeddings * factor` (where applicable)
3. If `tokenizer_config.json` has `model_max_length`: use as secondary reference
4. Otherwise: `"unknown"`

### 7.4 Important Distinctions

- `file_size_bytes` = total size of files on disk (sum of safetensors + config files)
- `parameter_count` = estimated model parameter count (from config or index)
- These MUST NOT be confused. Label clearly.

---

## 8. Context Length Validation in Preflight

### 8.1 Backend-Specific Parameter Mapping

| Backend | User Parameter | Model Metadata Field |
|---------|---------------|---------------------|
| vLLM | `max_model_len` (from `parameters_json`) | `default_context_length` on artifact |
| SGLang | `context_length` (from `parameters_json`) | `default_context_length` on artifact |
| llama.cpp | `ctx_size` (from `parameters_json`) | `default_context_length` on artifact |

### 8.2 Validation Rules

During `preflightDeployment` (before RunPlan resolution):

1. Extract user-requested context from `pf.params` using the backend-specific parameter name
2. Read `default_context_length` from the model artifact
3. Compare:

| Condition | Action |
|-----------|--------|
| `default_context_length == 0` or unknown | Add `warning` with code `unknown_model_context_length`. Do NOT block. |
| `user_context <= default_context_length` | PASS silently |
| `user_context > default_context_length` and no rope_scaling | Add `error` with code `context_length_exceeded`. **Block start.** |
| `user_context > default_context_length` with rope_scaling evidence | Add `warning` with code `context_length_exceeded_with_rope`. Do NOT block but warn prominently. |

### 8.3 Validation Implementation

The validation runs in `preflightDeployment`, after artifact fetch but before RunPlan resolution. It adds errors/warnings to the `preflightResult` using `pf.addErr()` and `pf.warns`.

### 8.4 DryRun Visibility

The context validation result MUST appear in the DryRun response. Add a `context_validation` field:
```json
{
  "context_validation": {
    "user_context": 8192,
    "model_context": 4096,
    "status": "warning",
    "code": "context_length_exceeded_with_rope",
    "message": "user max_model_len (8192) exceeds model default_context_length (4096)"
  }
}
```

---

## 9. RunPlan / DryRun Path Validation

### 9.1 GGUF Path Validation

In `buildMounts` or RunPlan resolution:
- GGUF model: verify the `-m` argument contains a `.gguf` file path, NOT just a directory
- If the model path in the args ends with `/` or is a directory: FAIL

### 9.2 HF Path Validation

- HF model: verify the model path in the args is a directory (container path), NOT a `.gguf` file
- The container path should be `/models/<dirname>` not `/models/<file>.gguf`

### 9.3 Host Path in Container Args

- No host path (`/home/...`) should appear in the application args after the image name
- Host paths are ONLY allowed in Docker volume mounts (`-v /host:/container`)

### 9.4 Mount Consistency

- Volume mount source must match the model location's `absolute_path`
- Volume mount target must match the container path used in the application args

---

## 10. API / Data Structure Rules

### 10.1 Scan Response Format

The scan endpoint `POST /nodes/{id}/model-paths/scan` response must include:

```json
{
  "scan_root": "/home/kzeng/models/Qwen3.5-9B-Q4",
  "candidates": [
    {
      "path": "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf",
      "path_type": "file",
      "format": "gguf",
      "detected_metadata": {
        "architecture": "qwen2",
        "context_length": 32768,
        "quantization": "Q4_K_M",
        "file_size_bytes": 5580000000,
        "parameter_count": "unknown"
      },
      "warnings": ["parameter_count not determined"],
      "auto_selected": true,
      "selection_reason": "single GGUF file in directory"
    }
  ],
  "warnings": []
}
```

### 10.2 Metadata Storage

- **ModelArtifact**: Use existing columns (`architecture`, `quantization`, `default_context_length`, `size_label`, `estimated_vram_bytes`). Populate from scan metadata.
- **ModelLocation**: Store full metadata in existing `discovered_metadata_json` column. This is a catch-all JSON field that can store any additional metadata not fitting in dedicated columns.

### 10.3 No Schema Migration

No new database columns. Use existing schema:
- `model_artifacts.architecture` — already exists, store HF architectures[0] or GGUF architecture
- `model_artifacts.quantization` — already exists, store quantization string
- `model_artifacts.default_context_length` — already exists, store context_length
- `model_locations.discovered_metadata_json` — already exists, store full metadata

If these fields prove insufficient during implementation, flag as DOCUMENTED_BLOCKER and propose minimal migration.

---

## 11. E2E Acceptance Criteria

### 11.1 Required E2E Scenarios

Each scenario below must have an E2E script (new or enhanced):

| # | Scenario | Key Assertions |
|---|----------|---------------|
| 1 | **Single GGUF scan** | auto-selected, path_type=file, path ends with .gguf, -m /models/<file>.gguf in DryRun |
| 2 | **Multi GGUF scan** | N candidates, none auto-selected, user picks one |
| 3 | **HF directory scan** | format=huggingface, path_type=directory, context_length extracted, container path used |
| 4 | **HF + GGUF mixed** | multiple candidate types, user selects |
| 5 | **GGUF metadata** | quantization detected, architecture extracted, context_length from GGUF header |
| 6 | **HF metadata** | model_type, architectures, torch_dtype, max_position_embeddings extracted |
| 7 | **Context preflight: PASS** | user ctx <= model ctx → no error |
| 8 | **Context preflight: WARNING** | user ctx > model ctx → warning in preflight response |
| 9 | **Context preflight: unknown** | model ctx unknown → warning, not blocked |
| 10 | **DryRun GGUF path** | -m /models/<file>.gguf present, no -m /models/ |
| 11 | **DryRun HF path** | container directory path, no host path in app args |
| 12 | **Mount consistency** | volume source = artifact path, volume target = container path in args |

### 11.2 E2E Output Requirements

Each E2E script must save to a unique artifact directory:
- `request_payload.json`
- `response.json`
- `http_status.txt`
- `scan_root.txt`
- `candidates.json`
- `selected_candidate.json`
- `saved_artifact.json`
- `saved_location.json`
- `detected_metadata.json`
- `warnings.json`
- `runplan.json` or `docker_command.txt`
- `assertion_report.txt`
- `cleanup_result.txt`
- `final_summary.txt`

### 11.3 UI Smoke

For UI smoke tests (if implemented):
- Save operation steps
- Save page key text evidence (DOM snapshots or text extraction)
- i18n keys must not leak into screenshots (use `en-US` or `zh-CN` consistently)

---

## 12. Out-of-Scope Findings

Record here any issues discovered during implementation that are outside this scope:

| # | Finding | Location | Why Out of Scope |
|---|---------|----------|-----------------|
|   |         |          |                  |

---

## 13. Implementation Order

1. **Agent: GGUF reader** — `internal/agent/collector/gguf_reader.go` (new file, minimal GGUF header parser)
2. **Agent: Enhanced scanner** — update `model_scanner.go` to produce multi-candidate response with metadata
3. **Server: Scan response pass-through** — update `agent_proxy_handlers.go` to pass through enhanced scan response
4. **Server: Preflight context validation** — add context validation to `preflightDeployment` in `deployment_lifecycle_handlers.go`
5. **Server: RunPlan path validation** — add GGUF/HF path checks in resolver
6. **Web: Wizard UI** — update `ModelArtifactsPage.vue` with scan status, candidate list, auto-selection
7. **Web: i18n** — add new keys to locale files
8. **E2E: Scan metadata scripts** — new E2E scripts for each acceptance scenario
9. **Documentation** — update 04 and 05 docs with final results

---

## 14. Verification

After implementation, verify:

```bash
# Syntax
bash -n scripts/e2e/*.sh scripts/e2e/lib/*.sh

# Go tests
go test ./internal/server/...
go test ./internal/agent/...

# Web build
npm --prefix web run build

# E2E
bash scripts/e2e/e2e-model-scan-metadata.sh
bash scripts/e2e-dryrun-parameter-matrix-enhanced.sh

# Governance
git diff --check
git status --short
```
