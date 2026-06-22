# 26 — HF Directory + GGUF File Detection Closeout

> Status: FIXED
> Scope: Directory scan properly distinguishes HF directory models from GGUF file models
> Date: 2026-06-23
> Baseline: commit `1088e56`

## 1. Changes Applied

### Agent Scanner (already correct)

The agent scanner (`model_scanner.go`) already detects:
- **HF directories**: checks for `config.json` → creates `PathType: "directory"`, `Format: "huggingface"` candidate
- **GGUF files**: globs `*.gguf` → creates `PathType: "file"`, `Format: "gguf"` candidates
- **Mixed**: both HF and GGUF candidates returned; no auto-selection → user must choose
- **Empty**: returns error `"no recognizable model files found in directory"`

### Frontend Wizard (enhanced)

1. **Candidate display**: Shows format badge (HF Directory / GGUF) for each candidate
2. **Mixed type handling**: Prefers HF directory as default selection when no auto-selected candidate exists
3. **Directory model label**: Shows "目录型模型 — 适用于 vLLM/SGLang" hint for HF candidates
4. **Empty result**: Shows "目录中没有发现模型文件" warning
5. **Success message**: Shows format-specific badge for auto-selected candidates

### Scan Proxy (unchanged)

The scan proxy correctly preserves per-candidate paths (fixed in Phase 2.5).

### Backend Validation (unchanged from previous phase)

`HandleCreateModelLocation` rejects GGUF format artifacts with directory-level absolute paths.

## 2. File Discovery Rules

| Scenario | Behavior |
|----------|----------|
| HF directory (config.json present) | Auto-select as directory model; format=huggingface; path_type=directory |
| Single GGUF in non-HF directory | Auto-select as file model; format=gguf; path_type=file |
| Multiple GGUF in non-HF directory | Show candidate list; user must select |
| Mixed HF + GGUF | Show all candidates; HF pre-selected as default; user can switch |
| No recognizable models | Error: "目录中没有发现模型文件" |

## 3. RunPlan Rules

| Backend | Model Format | Argument |
|---------|-------------|----------|
| vLLM / SGLang | huggingface (directory) | `--model-path /models/<dir>` |
| llama.cpp | gguf (file) | `-m /models/<dir>/<file>.gguf` |
| vLLM / SGLang | gguf (file) | Preflight fails (backend mismatch) |
| llama.cpp | huggingface (directory) | Preflight fails (backend mismatch) |

## 4. Modified Files

| File | Change |
|------|--------|
| `web/src/pages/ModelArtifactsPage.vue` | Enhanced candidate display with format badges, HF priority auto-select, empty result warning |
| `web/src/locales/zh-CN.ts` | Added scanSummary, directoryModel, fileModel, directoryModelHint, formatHF keys |
| `web/src/locales/en-US.ts` | Added same keys in English |
| `internal/server/api/artifact_handlers.go` | (previous phase) GGUF directory path rejection |
| `docs/.../26-hf-directory-scan-closeout.md` | This closeout |

## 5. Schema / Migration

- No schema changes
- No new migrations
- Scanner and frontend only

## 6. Test Results

```bash
gofmt -w cmd/ internal/                     → CLEAN
go test lightai-go/internal/server/api/...    → ALL PASS
go test lightai-go/internal/server/runplan/... → ALL PASS
go vet ./...                                   → CLEAN
npm test                                       → ALL PASS
npm run build                                  → ✓ built
git diff --check                                → CLEAN
```

## 7. Final Status

PASS — directory scan correctly distinguishes HF directory models from GGUF file models. Both formats produce correct RunPlans for their respective backends.
