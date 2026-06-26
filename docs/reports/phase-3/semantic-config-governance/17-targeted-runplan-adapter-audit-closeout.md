# 17. Targeted RunPlan Adapter Audit Closeout

> Status: PASS
> Scope: Targeted audit/fix for semantic RunPlan adapter mapping boundaries.
> Date: 2026-06-27

## Findings

| Area | Finding | Decision |
| --- | --- | --- |
| `service.listen_host` | Before this audit, semantic snapshot values were not explicitly carried into `ServiceInfo`; resolver relied on catalog command templates containing `--host 0.0.0.0`. | `ApplySemanticSnapshot()` now maps `service.listen_host` to `ServiceInfo.ListenHost`, and `applyServiceArgs()` applies it to the final `--host` flag. |
| `service.container_port` | Semantic snapshot already mapped to `ServiceInfo.ContainerPort` and `AppPort`; resolver `applyServiceArgs()` updates final `--port`. | Kept and covered by vLLM/SGLang/llama.cpp semantic adapter tests. |
| llama.cpp `deployment.served_model_name` | The adapter incorrectly mapped `deployment.served_model_name` to `--model`, which can replace the model path with an API service name. | Removed llama.cpp served-name CLI mapping. llama.cpp model path remains owned by model location / mount resolution through `-m {{model_container_file}}`. |
| vLLM/SGLang served model name | Both backends support served model name as API naming metadata. | Keep mapping to `--served-model-name`. |

## Test Evidence

Focused test:

```bash
go test ./internal/server/runplan -run 'TestSemanticAdapter'
```

Covered assertions:

- vLLM semantic snapshot produces `--host`, `--port`, `--max-model-len`, and `--served-model-name`.
- SGLang semantic snapshot produces `--host`, `--port`, `--context-length`, and `--served-model-name`.
- llama.cpp semantic snapshot produces `-m /models/llama.gguf`, `--host`, `--port`, and `--ctx-size`.
- llama.cpp does not place `deployment.served_model_name` in `--model`, `-m`, or any final args.

## File Notes

The requested audit item `internal/server/api/semantic_snapshot_helpers.go` does not exist in the current repository. The current semantic deployment helper is `internal/server/api/deployment_semantic.go`, and deployment preview/start call it before `runplan.ApplySemanticSnapshot()`.

## Closeout State

No blocker remains for this targeted audit.
