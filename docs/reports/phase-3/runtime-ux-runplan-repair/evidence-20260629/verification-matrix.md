# Runtime UX / RunPlan Repair Verification Matrix

Date: 2026-06-29

| Runtime | Copy user runtime | Node runtime save/check | Deployment dry-run | Docker preview | Start path or start-equivalent | Port binding | Device binding | List/detail names |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| vLLM NVIDIA Docker | PASS: API/UI copy now generates technical name and enforces display_name uniqueness | PASS: NBR creation UI keeps advanced params collapsed; existing API tests pass | PASS: `TestResolveVLLMNVIDIA` and full API tests | PASS: non-empty preview in `runplan-three-backend.log` | PASS: resolver output consumed by start path; real API start blocked by inactive local server | PASS: `8004:8000` and default host=container test | PASS: `--gpus "device=0"` and `CUDA_VISIBLE_DEVICES=0` | PASS: deployment DTO returns `model_display_name`/`model_name`; UI displays readable fields |
| SGLang NVIDIA Docker | PASS: shared copy/API/UI behavior | PASS: shared NBR UX/API behavior | PASS: `TestResolveSGLangNVIDIA` | PASS: non-empty preview in `runplan-three-backend.log` | PASS: resolver output consumed by start path; real API start blocked by inactive local server | PASS: `30000:30000` from template default | PASS: `--gpus "device=0"` and `CUDA_VISIBLE_DEVICES=0` | PASS: shared deployment DTO/UI behavior |
| llama.cpp NVIDIA Docker | PASS: shared copy/API/UI behavior | PASS: shared NBR UX/API behavior | PASS: `TestLlamaCppNvidiaRunPlan` | PASS: non-empty preview in `runplan-three-backend.log` | PASS: resolver output consumed by start path; real API start blocked by inactive local server | PASS: `8002:8080` in test evidence | PASS: `--gpus "device=0"` and `CUDA_VISIBLE_DEVICES=0` | PASS: shared deployment DTO/UI behavior |

Evidence:

- `docs/reports/phase-3/runtime-ux-runplan-repair/evidence-20260629/runplan-three-backend.log`
- `go test ./internal/server/runplan ./internal/server/api`
- `go test ./internal/server/...`
- `go test ./internal/agent/...`
- `go build ./cmd/server/...`
- `go build ./cmd/agent/...`
- `cd web && npm run build`
- `cd web && npm test`

