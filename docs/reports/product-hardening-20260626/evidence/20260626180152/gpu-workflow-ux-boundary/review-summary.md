# GPU Workflow UX Boundary — Review Summary

Timestamp: 2026-06-26 18:01 UTC | Evidence dir: `20260626180152/gpu-workflow-ux-boundary/`

## Validation Results

| Check | Result |
|---|---|
| `go test ./...` | ALL PASS (14 packages) |
| `go build ./cmd/server/...` | PASS |
| `go build ./cmd/agent/...` | PASS |
| `cd web && npm test` | ALL PASS (37 tests) |
| `cd web && npm run build` | PASS |
| `git diff --check` | PASS |
| `git status --short` | CLEAN |

## User-Facing Changes

### Runtime Templates Page
- Table shows `nvidia.vllm`, `nvidia.sglang`, `nvidia.llama.cpp b9700` format names
- ConfigSet and Source Metadata moved to collapsed "Advanced Diagnostics"
- Raw IDs replaced with resolved names

### Node Runtime Configs Page
- New wizard always starts from Step 1 (destroy-on-close)
- Config name field with auto-generated default
- HumanRuntimeParameterForm replaces raw ConfigSet editor
- Save/check errors stay open with error display
- Non-ready check results keep wizard open

### Model Library Page
- New model wizard uses NodeSelectorTable instead of dropdown
- Label: "选择模型所在节点" (Select Model File Node)

### Model Deployments Page
- NBR deployability checks remain enforced
- All previously fixed invariants maintained

## Commit Range

```
9ce7ced NodeSelectorTable + wizard reset + config naming
9383031 Runtime template presentation adapter
a535774 HumanRuntimeParameterForm
717cc04 Model library NodeSelectorTable
```

## Guardrails Confirmed

- No Gateway/API Key/Usage/Billing code added
- Model line: only file location and facts
- Runtime line: only runtime environment configuration
- ConfigSet internal keys hidden from normal forms
- RunPlan remains visible as product concept
