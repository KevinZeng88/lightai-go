# Phase 6 Report: 三后端/三厂商矩阵扩展

> Date: 2026-06-25

## 参数覆盖表

| Backend | Schema Params | Vendor Runtimes |
|---------|--------------|-----------------|
| vLLM | 18 | 6 (nvidia-docker, nvidia-cuda, cpu, metax, huawei, ascend-cann) |
| SGLang | 13 | 6 (nvidia-docker, nvidia-cuda, cpu, metax, metax-macart, huawei) |
| llama.cpp | 14 | 5 (nvidia-docker, nvidia-cuda13, cpu, metax, huawei) |

## Vendor Verification Status

| Vendor | vLLM | SGLang | llama.cpp |
|--------|------|--------|-----------|
| NVIDIA Docker | verified | verified | verified |
| NVIDIA CUDA | verified | verified | verified |
| MetaX | requires_hardware_validation | requires_hardware_validation | requires_hardware_validation |
| MetaX MacaRT | - | requires_hardware_validation | - |
| Huawei | template_only | template_only | template_only |
| Ascend CANN | template_only | - | - |
| CPU | template_only | template_only | verified |

## vLLM 参数覆盖

startup: model, host, port, served-model-name
performance: max-model-len, gpu-memory-utilization, max-num-seqs, max-num-batched-tokens, kv-cache-dtype, dtype, swap-space, cpu-offload-gb, enforce-eager, safetensors-load-strategy
parallelism: tensor-parallel-size, pipeline-parallel-size
security: trust-remote-code
advanced: download-dir

## SGLang 参数覆盖

startup: model-path, host, port, served-model-name
performance: mem-fraction-static, context-length, max-running-requests, disable-cuda-graph
parallelism: tp, tensor-parallel-size, dp
observability: enable-metrics, log-level

## llama.cpp 参数覆盖

startup: -m/--model, host, port
performance: ctx-size, n-gpu-layers, threads, threads-batch, batch-size, ubatch-size, cache-type-k, cache-type-v
parallelism: split-mode, main-gpu, tensor-split
