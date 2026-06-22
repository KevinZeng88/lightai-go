# Live Deployment Validation Issues

> Status: CURRENT
> Created: 2026-06-22
> Scope: Issues discovered during live vLLM deployment validation

## WEB-AI-LV-007: vLLM HF model started but test request model name does not match served model id

### User Phenomenon
- vLLM Docker container starts successfully
- `/v1/chat/completions` route exists (confirmed in Docker logs)
- `/v1/models` returns 200 OK
- But Chat Completion test fails with `Failed to fetch`

### Docker Log Evidence
```
Starting vLLM server on http://0.0.0.0:8000
Route: /v1/models, Methods: GET
Route: /v1/chat/completions, Methods: POST
Route: /v1/completions, Methods: POST
GET /v1/models HTTP/1.1" 200 OK
```

### Backend Error
```
ErrorInfo(message='The model `Qwen3-0.6B-Instruct-2512` does not exist.', type='NotFoundError', param='model', code=404)
POST /v1/chat/completions HTTP/1.1" 404 Not Found
```

### Root Cause
1. **DB seed data (`default_args_json`)**: vLLM v0.23.0 seed had `["{{model_container_path}}"]` with NO `--served-model-name` flag. Parameter defs had `--served-model-name` as optional with no default.

2. **`resolveModelID` short-circuit**: When `runplanModel` was non-empty (always, since it falls back to artifact `ModelName`), `resolveModelID()` returned immediately WITHOUT probing `/v1/models`. So it never verified whether the requested model name actually matched what vLLM served.

3. **`buildVarMap()` no fallback**: `SERVED_MODEL_NAME` defaulted to empty string. Without explicit user configuration or parameter default, the template variable resolved to empty, so `--served-model-name` had no effect even if in args.

### Root Cause Hypothesis Confirmed
Test request used `display_name` (`Qwen3-0.6B-Instruct-2512`), but vLLM registered model id as a different value (likely path-based or HF ID from `config.json`) because `--served-model-name` was not set.

### Fix Applied
1. **DB seed data**: Added `--host 0.0.0.0 --port {{container_port}} --served-model-name {{served_model_name}}` to vLLM v0.23.0 default args.
2. **`buildVarMap()`**: When `SERVED_MODEL_NAME` is empty, derive from `in.Artifact.Name` (artifact name).
3. **`resolveModelID()`**: Always probes `/v1/models`. Returns `runplanModel` only if verified against available models. Includes `availableModels` in return for diagnostics.
4. **Test API**: Adds `requested_model`, `available_models`, `hint` to response.
5. **Frontend**: `formatTestFailure()` now shows requested_model vs available_models on 404/`model_id_not_resolved`.

### Acceptance Criteria
- vLLM RunPlan includes `--served-model-name` with the correct model name
- Test API probes `/v1/models` and verifies model name
- On mismatch, frontend shows requested vs available models, not just `Failed to fetch`
- `/v1/chat/completions` route confirmed to exist (not the issue)
