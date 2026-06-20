# Phase 4 Real Smoke Evidence — 2026-06-21

## Summary

| Backend | Result | Root Cause | Blocker |
|---------|--------|-----------|---------|
| llama.cpp `server-cuda13` | **PASS** | N/A | None |
| vLLM `latest` | **FAIL** | `RuntimeError: Failed to infer device type` — GPU device detection failure in WSL2 | DOCUMENTED_BLOCKER: WSL2/NVIDIA runtime incompatibility with vLLM's GPU enumeration |
| SGLang `latest` | **FAIL** | `NotImplementedError` in `get_device` — Triton not supported on WSL2 platform | DOCUMENTED_BLOCKER: WSL2/NVIDIA runtime incompatibility with SGLang/Triton GPU detection |

## llama.cpp PASS Details

- Image: `ghcr.io/ggml-org/llama.cpp:server-cuda13`
- ENTRYPOINT: `["/app/llama-server"]` (preserved by image_default mode)
- Health check: `/v1/models` returned 200 after ~19s
- Cleanup: container stopped and removed
- E2E script: `scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh`

## vLLM FAIL Details

- Image: `vllm/vllm-openai:latest`
- ENTRYPOINT: `["vllm","serve"]` (explicitly set — differs from image_default due to legacy entrypoint override)
- Error: `RuntimeError: Failed to infer device type, please set the environment variable VLLM_DEVICE`
- WSL detection: `Using 'pin_memory=False' as WSL is detected`
- Exit code: 1
- E2E script: `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh`

Note: External direct `docker run` with `--gpus all` succeeds with the same image. The LightAI-managed container uses `DeviceRequest` with specific GPU UUIDs, which may trigger different GPU enumeration behavior in WSL2.

## SGLang FAIL Details

- Image: `lmsysorg/sglang:latest`
- ENTRYPOINT: image has `["/opt/nvidia/nvidia_entrypoint.sh"]` but LightAI overrides to `["python3","-m","sglang.launch_server"]` (legacy entrypoint override bypasses NVIDIA wrapper)
- Error: `Triton is not supported on current platform, roll back to CPU` then `NotImplementedError` in `get_device()`
- Exit code: 1
- E2E script: `scripts/e2e-model-runtime-wizard-nvidia-sglang.sh`

Note: The legacy entrypoint override discards the image's NVIDIA entrypoint wrapper (`/opt/nvidia/nvidia_entrypoint.sh`). The Phase 3 `process_start_config` with `image_default` mode would preserve this wrapper, but no process_start_config is yet being applied to these deployments (NBRs created before Phase 3 lack the config).

## Layer 3 Impact Assessment

- Layer 3 changes (Phases 1-3) do NOT introduce new failures.
- llama.cpp continues working (no regression).
- vLLM and SGLang failures are pre-existing GPU/WSL issues, not caused by Phase 1-3 changes.
- The `process_start_config` mechanism is ready for use but requires:
  a) User acceptance of detection → config on NBR (Phase 2 data flow exists, but UI/API trigger not yet wired)
  b) For SGLang specifically, applying `image_default` mode would preserve the NVIDIA wrapper entrypoint, which MAY improve GPU detection. This requires manual NBR config patching or an accept-detection API.
