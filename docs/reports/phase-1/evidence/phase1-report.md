# Phase 1 Report: 参数语义正确性最小闭环

> Date: 2026-06-25

## 修复内容

1. **Layer 3 template substitution**: resolver Layer 3 (Deployment overrides) now calls `substituteVars()` for template variable replacement
2. **Required params locked**: RuntimeParameterEditor backendParams checkbox now disabled when `param.required` is true
3. **Required params always enabled**: `syncBackendParamsFromSchema` forces `enabled: true` for required params
4. **host/container_port protection**: Deployment override cannot override `--host` or `--port` (protected flags)
5. **Test fix**: `TestVLLMRunPlanRendersHostPortFlags` updated to use `Required: true` on host/port params
6. **Test helper fix**: `makeNbrSnapshotFromInput` now creates default param values from schema (matching real behavior)

## E2E Results

| Backend | Result |
|---------|--------|
| vLLM default | PASS |
| vLLM modified | PASS |
| SGLang | PASS |
| llama.cpp | PASS |
