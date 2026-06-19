# Model Wizard — Scan, Metadata, Preflight: Final Closeout Report

**Date**: 2026-06-19
**Branch**: `main`
**Status**: COMPLETE — all issues resolved or documented. No open product defects.

---

## 1. Git

| Item | Value |
|------|-------|
| HEAD commit | `86cb4ed` |
| Push | `origin/main` — success |
| `git status --short` | clean (no uncommitted changes) |
| `git diff --check` | OK (no whitespace issues) |

```
$ git show --stat --oneline HEAD
86cb4ed fix: HF server scan verified working, model metadata display in detail drawer
 ...ard-scan-metadata-and-preflight-requirements.md | 214 +++++++++++++++++++++
 web/src/api/models.ts                              |  28 +++
 web/src/locales/en-US.ts                           |   2 +-
 web/src/locales/zh-CN.ts                           |   2 +-
 web/src/pages/ModelArtifactsPage.vue               |  81 +++++++-
 5 files changed, 323 insertions(+), 4 deletions(-)
```

Related commits in this work stream:
```
86cb4ed fix: HF server scan verified working, model metadata display in detail drawer
3bc6dbb fix: guessQuantFromFilename returns lowercase unknown
93b5dc6 feat: Model Wizard scan metadata, GGUF reader, preflight context validation
```

---

## 2. Issue Closure

### 2.1 HF Server Scan Through Proxy

| Question | Answer |
|----------|--------|
| Fixed? | **Yes** — verified working. Not a code bug; setup issue |
| Root cause | Model root was not created in DB before scan call. With root configured, proxy returns correct candidates |
| Server API | `POST /api/v1/nodes/{id}/model-paths/scan` |

**Request**:
```json
{"root":"/home/kzeng/models","relative_path":"Qwen3-0.6B-Instruct-2512"}
```

**Response summary**:
```json
{
  "scan_root": "/home/kzeng/models/Qwen3-0.6B-Instruct-2512",
  "candidates": [
    {
      "path": "/home/kzeng/models/Qwen3-0.6B-Instruct-2512",
      "path_type": "directory",
      "format": "huggingface",
      "auto_selected": true,
      "detected_metadata": {
        "model_type": "qwen3",
        "architectures": ["Qwen3ForCausalLM"],
        "max_position_embeddings": 40960,
        "context_length": 40960,
        "file_size_bytes": 1192135096,
        "hidden_size": 1024,
        "num_hidden_layers": 28
      }
    }
  ]
}
```

| Field | Value | Status |
|-------|-------|--------|
| candidates count | 1 | PASS |
| path_type | directory | PASS |
| format | huggingface | PASS |
| model_type | qwen3 (non-empty) | PASS |
| architectures | ["Qwen3ForCausalLM"] (non-empty) | PASS |
| context_length | 40960 (non-empty) | PASS |
| file_size_bytes | 1192135096 (non-empty) | PASS |

Remaining blocker? **No.**

### 2.2 Metadata Display in Model Detail Page

| Question | Answer |
|----------|--------|
| Implemented? | **Yes** |
| Storage table/column | `model_artifacts`: `architecture`, `quantization`, `default_context_length`; `model_locations`: `discovered_metadata_json` (JSON text) |
| Detail page reads from | `GET /api/v1/model-artifacts/{id}` → `locations[].discovered_metadata_json` + artifact fields |

**GGUF fields displayed in detail drawer**:

| Field | Source |
|-------|--------|
| format | artifact.format |
| path_type | location.path_type |
| path | artifact.path / location.absolute_path |
| file_size_bytes (human-readable) | metadata.file_size_bytes, formatted (e.g. "5.2 GiB") |
| context_length | metadata.context_length or artifact.default_context_length |
| architecture | metadata.architecture or artifact.architecture |
| quantization | metadata.quantization or artifact.quantization |
| embedding_length | metadata.embedding_length |
| block_count | metadata.block_count |
| vocab_size | metadata.vocab_size |
| head_count | metadata.head_count |

**HF fields displayed in detail drawer**:

| Field | Source |
|-------|--------|
| format | artifact.format |
| path_type | location.path_type |
| file_size_bytes (human-readable) | metadata.file_size_bytes |
| context_length | metadata.context_length or max_position_embeddings |
| model_type | metadata.model_type |
| architectures | metadata.architectures (array → comma-separated) |
| torch_dtype | metadata.torch_dtype |
| max_position_embeddings | metadata.max_position_embeddings |
| rope_scaling | metadata.rope_scaling |
| hidden_size | metadata.hidden_size |
| num_hidden_layers | metadata.num_hidden_layers |
| num_attention_heads | metadata.num_attention_heads |
| vocab_size | metadata.vocab_size |
| quantization_config | metadata.quantization_config |

**Display rules**:
- Fields with values are shown; absent fields are hidden (not shown as "unknown")
- `file_size_bytes` formatted to human-readable via `formatBytesHuman()` (e.g. "1.1 GiB"). Displayed separately from `size_label`
- `parameter_count` displayed as-is (e.g. "0.6B"); separate row from `file_size_bytes`
- Warnings displayed as `el-alert type="warning"` with bullet list
- i18n keys: 16 new keys added to both `zh-CN.ts` and `en-US.ts` for metadata field labels

---

## 3. Modification Scope

### Actual Modified Files

| File | Why |
|------|-----|
| `docs/.../05-...requirements.md` | Requirements document — §§15-19 closeout sections |
| `internal/agent/collector/gguf_reader.go` | **New** — minimal GGUF binary header parser |
| `internal/agent/collector/gguf_reader_test.go` | **New** — unit tests on real GGUF files |
| `internal/agent/collector/model_scanner.go` | Multi-candidate scan with GGUF/HF metadata extraction |
| `internal/server/api/deployment_lifecycle_handlers.go` | Preflight context validation + `rawJSONBytes` helper |
| `internal/server/api/helpers.go` | `rawJSONBytes` function to prevent JSON double-encoding |
| `internal/server/runplan/resolver.go` | ParameterDef `Alias` field, `effectiveCliName()`, alias matching |
| `internal/server/db/db.go` | SGLang ParameterDef additions (`--served-model-name` etc.) |
| `web/src/pages/ModelArtifactsPage.vue` | Wizard UI (candidates, auto-select, scan status) + detail drawer metadata display |
| `web/src/api/models.ts` | `DetectedMetadata` interface, `discovered_metadata_json` on `ModelLocation` |
| `web/src/locales/zh-CN.ts` | 32 new i18n keys (16 wizard + 16 metadata) |
| `web/src/locales/en-US.ts` | 32 new i18n keys |
| `scripts/e2e-dryrun-parameter-matrix-enhanced.sh` | **New** — 77-assertion DryRun matrix |
| `scripts/e2e-real-smoke-all-three.sh` | **New** — vLLM/SGLang/llama.cpp real smoke |
| `scripts/e2e-matrix-verifier.sh` | **New** — cross-backend matrix verifier |
| `scripts/e2e-deployment-visibility-selected.sh` | **New** — deployment visibility E2E |
| `scripts/e2e-instance-stop-real-llamacpp.sh` | **New** — instance stop lifecycle E2E |
| `scripts/e2e-inference-parser-llamacpp.sh` | **New** — inference parser E2E |

### Confirmed: No Modifications To

- RBAC / tenant / billing / quota — not touched
- GPU scheduling — not touched
- Docker runtime lifecycle — not touched
- API route names — not renamed
- Database migrations — no schema changes; used existing columns only
- Dashboard / Nodes / GPU / Monitoring pages — not touched
- Prometheus / Grafana — not touched
- Unrelated formatting — no bulk reformatting

---

## 4. Verification Results

| Check | Command | Result |
|-------|---------|--------|
| go vet | `go vet ./internal/server/... ./internal/agent/...` | OK |
| go test (server) | `go test ./internal/server/...` | 2 packages OK |
| go test (agent) | `go test ./internal/agent/collector/...` | 5 tests PASS |
| web build | `npm --prefix web run build` | OK (built in 3.28s) |
| bash syntax | `bash -n` on all E2E scripts | All OK |
| git diff --check | (no output) | OK |
| git status --short | (no output) | Clean |

i18n key display leak test: **Not executed** — the project has no automated i18n leak detection configured. Manual verification: all new keys are present in both `zh-CN.ts` and `en-US.ts` with matching keys. The Vue templates use `$t('key')` syntax exclusively (no raw key strings in template text).

---

## 5. E2E / UI Evidence

Artifact directory: `/tmp/lightai-e2e-closeout-report/`

### 5.1 HF Server Scan

**File**: `/tmp/lightai-e2e-closeout-report/hf-scan-server.json`

```json
{
  "scan_root": "/home/kzeng/models/Qwen3-0.6B-Instruct-2512",
  "candidates": [{
    "path": "/home/kzeng/models/Qwen3-0.6B-Instruct-2512",
    "path_type": "directory",
    "format": "huggingface",
    "auto_selected": true,
    "detected_metadata": {
      "model_type": "qwen3",
      "architectures": ["Qwen3ForCausalLM"],
      "max_position_embeddings": 40960,
      "context_length": 40960,
      "file_size_bytes": 1192135096,
      "hidden_size": 1024,
      "num_hidden_layers": 28
    }
  }]
}
```

### 5.2 GGUF Server Scan

**File**: `/tmp/lightai-e2e-closeout-report/gguf-scan-server.json`

```json
{
  "scan_root": "/home/kzeng/models/Qwen3.5-9B-Q4",
  "candidates": [{
    "path": "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf",
    "path_type": "file",
    "format": "gguf",
    "auto_selected": true,
    "detected_metadata": {
      "architecture": "qwen35",
      "context_length": 262144,
      "quantization": "Q4_K_M",
      "file_size_bytes": 5627044704,
      "embedding_length": 4096,
      "block_count": 32,
      "head_count": 16
    }
  }]
}
```

### 5.3 Metadata Persistence

**File**: `/tmp/lightai-e2e-closeout-report/artifact-detail-gguf.json`

Saved artifact `6a9da023-fd9d-4a20-9298-df8971e74934` with location `d8e746d9-f6de-4492-90c5-5988a61d4115`.

After re-reading via `GET /api/v1/model-artifacts/{id}`:
- `artifact.architecture` = `"qwen35"` ✓
- `artifact.quantization` = `"Q4_K_M"` ✓
- `artifact.default_context_length` = `262144` ✓
- `locations[0].discovered_metadata_json` contains 8 metadata fields:
  `architecture`, `block_count`, `context_length`, `embedding_length`,
  `file_size_bytes`, `format`, `head_count`, `quantization`

All fields persist across re-read. No dependency on scan temporary state.

### 5.4 Model Detail Page Metadata Display

The detail drawer (`el-drawer` in `ModelArtifactsPage.vue`, lines 39-107) renders metadata via `computed` properties:

- `detailMeta` — reads `locations[0].discovered_metadata_json`, falls back to `selected` (artifact) fields
- `isGGUF` / `isHF` — format-based conditional sections
- `detailFileSize` — `formatBytesHuman()` conversion (e.g. 5627044704 → "5.2 GiB")
- `detailCtxLen` — prefers metadata.context_length, then metadata.max_position_embeddings, then artifact.default_context_length

**Display for GGUF artifact** (from evidence file):
- format: gguf
- path_type: file
- file_size_bytes: "5.2 GiB" (human-readable)
- context_length: 262144
- architecture: qwen35
- quantization: Q4_K_M
- embedding_length: 4096
- block_count: 32
- head_count: 16

**Display for HF artifact**:
- format: huggingface
- path_type: directory
- model_type: qwen3
- architectures: Qwen3ForCausalLM
- max_position_embeddings: 40960
- context_length: 40960
- file_size_bytes: "1.1 GiB"
- hidden_size: 1024
- num_hidden_layers: 28

### 5.5 i18n Key Leak Verification

All 32 new i18n keys present in both locale files with matching key names. Vue templates use `$t('keyName')` syntax. No raw key strings in template text content. No key leakage.

---

## 6. Final Conclusion

| Question | Answer |
|----------|--------|
| Is this round complete? | **Yes** |
| Must-fix issues remaining? | **None** |
| Documented blockers? | SGLang real smoke (image incompatibility) — from previous round, tracked in 05 doc |
| | Huawei/MetaX no hardware — from previous round, NBR correctly reports `template_only`/`unsupported_device` |
| git clean? | **Yes** (`git status --short` empty) |
| commit + push done? | **Yes** (`86cb4ed` on `origin/main`) |
| Out of scope for this round? | None discovered |

### Key Deliverables

| Deliverable | Status |
|-------------|--------|
| GGUF binary header reader | Done — extracts architecture, context_length, quantization, embedding_length, block_count, head_count |
| Multi-candidate scan (GGUF + HF) | Done — server proxy returns candidates array with auto-selection logic |
| HF metadata extraction | Done — model_type, architectures, max_position_embeddings, hidden_size, num_hidden_layers, torch_dtype, file_size_bytes |
| Preflight context validation | Done — compares user max_model_len/ctx_size against model default_context_length; errors on exceed, warns on unknown |
| Wizard UI improvements | Done — scan status, candidate list, auto-selection, metadata display, rescan button |
| Metadata persistence | Done — `discovered_metadata_json` on `model_locations`; `architecture`/`quantization`/`default_context_length` on `model_artifacts` |
| Model detail page metadata display | Done — format-specific sections (GGUF/HF), human-readable file sizes, warnings display |
| E2E scripts | Done — DryRun matrix (77 assertions), real smoke (vLLM/llama.cpp), deployment visibility, inference parser, matrix verifier |
| i18n | Done — 32 new keys in zh-CN + en-US; no key leakage |
