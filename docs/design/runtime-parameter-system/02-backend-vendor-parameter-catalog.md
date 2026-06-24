# 02 - Backend and Vendor Parameter Catalog

Status: discussion draft  
Target repo path: `docs/design/runtime-parameter-system/02-backend-vendor-parameter-catalog.md`

## 1. Purpose and caution

This document proposes a parameter catalog and defaulting policy for vLLM, SGLang, llama.cpp, NVIDIA, MetaX, and Huawei Ascend.

It is not a replacement for target-image verification. Before implementation, MiMo must verify parameters against the actual deployed image help output and current project catalog.

Required verification commands:

```bash
# vLLM

docker run --rm vllm/vllm-openai:latest vllm serve --help || true
docker run --rm vllm/vllm-openai:latest python -m vllm.entrypoints.openai.api_server --help || true

# SGLang

docker run --rm lmsysorg/sglang:latest python3 -m sglang.launch_server --help || true
docker run --rm lmsysorg/sglang:latest sglang serve --help || true

# llama.cpp

docker run --rm ghcr.io/ggml-org/llama.cpp:server-cuda13 --help || true
docker run --rm ghcr.io/ggml-org/llama.cpp:server-cuda13 llama-server --help || true
```

## 2. Parameter categories

| Category | UI default | Enablement | Examples |
|---|---|---|---|
| Core startup | visible | required, locked | model path, host, container_port |
| Service identity | visible | optional/recommended | served_model_name, alias |
| Performance/capacity | visible or advanced | optional unless project requires | max_model_len, gpu_memory_utilization, max_num_seqs |
| Parallelism | advanced | optional | tensor_parallel_size, data_parallel_size, tp |
| API/security | advanced | optional, redacted if secret | API key, HF token |
| Docker runtime | dedicated Docker/runtime groups | required/optional by profile | shm_size, ipc_mode, devices |
| Vendor runtime | vendor-specific groups | only matching vendor | MetaX devices, Ascend devices |
| Custom | advanced | optional | extra_args, extra_env, extra_docker_options |

## 3. Common startup fields

These should be schema fields across all backends where applicable.

| Name | Required | Editable | Default/recommendation | Notes |
|---|---:|---:|---|---|
| `model_container_path` | yes | no / read-only | system generated | Derived from ModelLocation + mount resolver. |
| `host` | yes | yes | `0.0.0.0` | Required for containerized service. |
| `container_port` | yes | yes | backend internal default | Distinct from host port mapping. |
| `served_model_name` | backend-dependent | yes | `lightai-{model_slug}` | vLLM supports it; SGLang only when current image supports it; llama.cpp should hide if unsupported. |
| `extra_args` | no | yes | disabled / empty | Long-tail only; must not duplicate structured core args. |

Suggested defaults:

- vLLM container port: `8000`.
- SGLang container port: `30000`.
- llama.cpp container port: verify with target image; current project uses `8000`.

## 4. vLLM parameter catalog

Official docs expose a large number of engine/server args. LightAI should cover common parameters as structured schema fields, and leave long-tail parameters to `extra_args` with conflict detection.

### 4.1 vLLM core and model fields

| Parameter | Flag | Type | Default policy | Recommendation |
|---|---|---|---|---|
| model path | positional or `--model` depending entrypoint | path | required, system generated | Do not expose as normal editable field. |
| served model name | `--served-model-name` | string | optional/recommended | Use `lightai-{model_slug}`; allow deployment override. |
| tokenizer | `--tokenizer` | string/path | optional disabled | Useful when tokenizer differs from model. |
| trust remote code | `--trust-remote-code` | bool | optional disabled | High caution; explain supply-chain risk. |
| dtype | `--dtype` | enum/string | optional disabled | Prefer backend/model auto unless user needs override. |
| max model length | `--max-model-len` | int/string | optional disabled | Limit when memory pressure or product policy requires. |
| quantization | `--quantization` | enum/string | optional disabled | Enable only when model/runtime supports it. |
| load format | `--load-format` | enum/string | optional disabled | Usually auto. |
| config format | `--config-format` | enum/string | optional disabled | Usually auto. |

### 4.2 vLLM performance/capacity fields

| Parameter | Flag | Type | Default policy | Recommendation |
|---|---|---|---|---|
| GPU memory utilization | `--gpu-memory-utilization` | float | optional with suggested value | Official default has commonly been 0.9; LightAI may suggest 0.85-0.92 for single instance and lower for multi-instance. Verify image help. |
| KV cache dtype | `--kv-cache-dtype` | string | optional disabled | Only set when model/backend supports it. |
| max num sequences | `--max-num-seqs` | int | optional disabled | Control concurrency; useful in constrained GPUs. |
| max batched tokens | `--max-num-batched-tokens` | int | optional disabled | Tune throughput/memory. |
| prefix caching | `--enable-prefix-caching` | bool | optional | Enable when repeated prefixes/workload benefits. |
| chunked prefill | `--enable-chunked-prefill` | bool | optional | Enable when workload/engine version benefits; verify defaults. |
| enforce eager | `--enforce-eager` | bool | optional disabled | Debug/compatibility; may reduce performance. |
| CPU offload | `--cpu-offload-gb` | float | optional disabled | Only for memory-limited setups. |
| swap space | `--swap-space` | float/int | optional disabled | Verify target version. |

### 4.3 vLLM parallelism fields

| Parameter | Flag | Default | Recommendation |
|---|---|---|---|
| tensor parallel size | `--tensor-parallel-size` | 1 | Enable only when GPU lease count matches. |
| pipeline parallel size | `--pipeline-parallel-size` | 1 | Advanced multi-GPU. |
| data parallel size | `--data-parallel-size` if supported | 1 | Verify target image/version. |

### 4.4 vLLM service/API fields

| Parameter | Flag | Recommendation |
|---|---|---|
| host | `--host` | required, default `0.0.0.0`. |
| port | `--port` | required, container port default `8000`. |
| API key | version-specific | optional secret, redacted. |
| metrics | version-specific | optional, verify target image. |

## 5. SGLang parameter catalog

SGLang documents server arguments and recommends tuning memory-related values such as chunked prefill and memory fraction for OOM scenarios. The current image must be verified with `python3 -m sglang.launch_server --help`.

### 5.1 SGLang core/model fields

| Parameter | Flag | Type | Default policy | Recommendation |
|---|---|---|---|---|
| model path | `--model-path` | path/string | required, system generated | Use container model path. |
| model | `--model` if supported | string | omit unless target image uses it | Avoid duplicate model path semantics. |
| tokenizer path | `--tokenizer-path` | string/path | optional disabled | Use when tokenizer differs. |
| tokenizer mode | `--tokenizer-mode` | enum/string | optional disabled | Verify target version. |
| trust remote code | `--trust-remote-code` | bool | optional disabled | High caution. |
| load format | `--load-format` | enum/string | optional disabled | Usually auto. |
| context length | `--context-length` | int | optional disabled | Use when limiting memory or model config is wrong. |
| served model name | `--served-model-name` if supported | string | optional/recommended | Show only if target image supports it. |
| chat template | `--chat-template` | string/path | optional disabled | Backend/model-specific. |
| HF chat template name | `--hf-chat-template-name` | string | optional disabled | Verify target version. |

### 5.2 SGLang service fields

| Parameter | Flag | Recommendation |
|---|---|---|
| host | `--host` | required, default `0.0.0.0`. |
| port | `--port` | required, default `30000`. |
| API key | `--api-key` if supported | optional secret, redacted. |
| metrics | `--enable-metrics` if supported | optional. |
| request logs | `--log-requests` if supported | optional; beware log noise. |

### 5.3 SGLang performance/capacity fields

| Parameter | Flag | Default policy | Recommendation |
|---|---|---|---|
| tensor parallel | `--tp` / `--tensor-parallel-size` | optional | Must match GPU lease count. |
| data parallel | `--dp` / `--data-parallel-size` | optional | Multi-GPU advanced. |
| static memory fraction | `--mem-fraction-static` | optional | Lower if decoding OOM. Do not blindly set across all models. |
| max total tokens | `--max-total-tokens` | optional | Tune for memory. |
| max prefill tokens | `--max-prefill-tokens` | optional | Tune for long prompts. |
| chunked prefill size | `--chunked-prefill-size` | optional | If prefill OOM, reduce to 4096 or 2048 as SGLang tuning docs suggest. |
| schedule policy | `--schedule-policy` | optional | Advanced. |
| priority scheduling | `--enable-priority-scheduling` | optional | Advanced. |
| KV cache dtype | `--kv-cache-dtype` | optional | Compatibility/performance tuning. |
| quantization | `--quantization` | optional | Only when supported. |
| skip server warmup | `--skip-server-warmup` | optional | Useful for debugging; may affect readiness/performance. |
| disable CUDA graph | `--disable-cuda-graph` if supported | optional | Debug/compatibility; may reduce performance. |

### 5.4 SGLang health policy

For the current LightAI environment, SGLang became ready after roughly 50 seconds because initialization included model load, KV cache allocation, CUDA graph capture, piecewise CUDA graph capture/compilation, and warmup. BackendVersion health check should set `startup_timeout_seconds` to at least `120` unless target image/version evidence suggests otherwise.

## 6. llama.cpp parameter catalog

llama.cpp server options are image/version-dependent. Validate with the current `server-cuda13` image help.

### 6.1 llama.cpp core/model fields

| Parameter | Flag | Type | Default policy | Recommendation |
|---|---|---|---|---|
| model | `-m` / `--model` | path | required, system generated | Use GGUF container path. |
| host | `--host` | string | required | Default `0.0.0.0`. |
| port | `--port` | int/string | required | Verify image default; project commonly maps to container `8000`. |
| alias | `--alias` if supported | string | optional | Use as served name if supported. |
| chat template | `--chat-template` / `--jinja` if supported | string | optional | Model-specific. |
| HF token | `--hf-token` or env if supported | secret | optional | Must be redacted. |

### 6.2 llama.cpp performance/capacity fields

| Parameter | Flag | Default policy | Recommendation |
|---|---|---|---|
| context size | `-c` / `--ctx-size` | optional | Prefer model/default or auto; avoid blindly large context. |
| GPU layers | `-ngl` / `--n-gpu-layers` | optional | Use auto/all if image supports; otherwise tune by memory. |
| threads | `-t` / `--threads` | optional | CPU-bound tuning. |
| batch threads | `-tb` / `--threads-batch` | optional | Advanced. |
| batch size | `-b` / `--batch-size` | optional | Use image default unless tuning. |
| micro batch size | `-ub` / `--ubatch-size` | optional | Tune memory/perf. |
| parallel slots | `-np` / `--parallel` | optional | Multi-request serving. |
| flash attention | `--flash-attn` if supported | optional | Enable when CUDA/image/model path supports. |
| cache type K/V | `--cache-type-k`, `--cache-type-v` | optional | Advanced memory/perf. |
| mmap/mlock | `--mmap`, `--no-mmap`, `--mlock` | optional | Host memory behavior; explain risk. |

### 6.3 llama.cpp env warning policy

If image default env includes `LLAMA_ARG_HOST=0.0.0.0` and platform also passes `--host 0.0.0.0`, the CLI arg correctly takes precedence. This should be classified as a benign image-default warning when:

- platform did not inject `LLAMA_ARG_HOST`;
- CLI `--host` value matches intended value;
- container becomes ready.

The platform should not inject `LLAMA_ARG_HOST` or `LLAMA_ARG_PORT` when structured CLI args are used.

## 7. Docker runtime common fields

| Field | Type | Default policy | Recommendation |
|---|---|---|---|
| image | string | required | Runtime-specific. |
| entrypoint | list/string | optional | Needed for SGLang/llama.cpp image variants. |
| cmd args | list | generated | From command template + structured schema + extra args. |
| env | map/list | optional | Structure + extra_env; secrets redacted. |
| volumes | list | required for model mount | Use model path resolver. |
| ports | list/map | required | Distinguish host port and container port. |
| devices | list | vendor-specific optional/required | Do not use MetaX devices on NVIDIA. |
| device_requests / gpus | structured | NVIDIA default | Prefer for NVIDIA GPU selection. |
| privileged | bool | high risk | Default false unless vendor profile requires. |
| ipc_mode | string | high risk | Default conservative; SGLang/MetaX may need larger shm/IPC. |
| shm_size | size | recommended | Use backend/vendor recommendation; avoid arbitrary huge default except vendor profile. |
| network_mode | string | high risk | Avoid host unless needed. |
| pid_mode | string | high risk | Avoid host unless needed. |
| uts_mode | string | high risk | Vendor-specific only. |
| security_opt | list | high risk | Vendor-specific only. |
| group_add | list | vendor-specific | MetaX may require `video`. |
| ulimits | list | vendor-specific | MetaX may need `memlock=-1`. |
| extra_hosts | list | optional | Advanced. |
| resource controls | CPU/memory | optional | Product policy. |
| log options | map | optional | Avoid noise. |

High-risk fields should have one authoritative editing location and one help explanation.

## 8. NVIDIA vendor profile

### 8.1 Default strategy

- Use Docker DeviceRequests / `--gpus` / GPU lease to control GPU visibility.
- Avoid platform-injected `CUDA_VISIBLE_DEVICES` by default.
- Optionally support `NVIDIA_VISIBLE_DEVICES` when needed.
- Do not default to privileged.
- Do not default to MetaX or Huawei device paths.

### 8.2 NVIDIA must not contain by default

NVIDIA profile must not include these unless user manually adds them:

- `/dev/dri`;
- `/dev/mxcd`;
- `/dev/infiniband` as a MetaX default;
- `/dev/davinci*`;
- `/dev/davinci_manager`;
- `/dev/devmm_svm`;
- `/dev/hisi_hdc`;
- unconfined seccomp / apparmor as a default;
- `group_add video` as a default.

## 9. MetaX / 沐曦 vendor profile

### 9.1 Public support notes

MetaX provides `vLLM-metax`, a hardware plugin enabling vLLM on MetaX GPUs through the MACA backend. This means MetaX vLLM should be modeled as vLLM backend schema plus MetaX vendor profile and MetaX image/plugin compatibility, not as NVIDIA vLLM with a few extra devices.

Public SGLang MetaX support is less clearly documented than vLLM-MetaX and Huawei Ascend SGLang. MiMo should verify available MetaX SGLang images/docs from vendor channels before enabling a system catalog entry. If no reliable source/image exists, add a planned profile with disabled/unverified status rather than a runnable preset.

### 9.2 Common MetaX Docker runtime fields to model

These fields are vendor-specific and high-risk/advanced unless vendor docs mark them required:

- devices:
  - `/dev/dri`;
  - `/dev/mxcd`;
  - `/dev/infiniband` only if IB/RDMA is needed;
- group:
  - `group_add: video`;
- IPC/UTS/security:
  - `ipc=host`;
  - `uts=host`;
  - `privileged=true` if vendor requires;
  - `security-opt seccomp=unconfined`;
  - `security-opt apparmor=unconfined`;
- shared memory:
  - `shm_size: 100gb` as vendor recommendation if confirmed;
- ulimits:
  - `memlock=-1`;
- env:
  - `CUDA_VISIBLE_DEVICES` / MACA-visible-device equivalent if required;
  - `MCCL_SOCKET_IFNAME`;
  - `GLOO_SOCKET_IFNAME`;
  - `MCCL_IB_HCA`;
  - `MACA_MPS_MODE` if required by deployment mode.

### 9.3 MetaX backend combinations

Initial design target:

| Backend | MetaX status | Action |
|---|---|---|
| vLLM | public vLLM-metax exists | Add/validate MetaX vLLM profile/catalog entry. |
| SGLang | must verify vendor docs/image | Add only after confirmed help/log evidence. |
| llama.cpp | unclear | Do not add runnable preset without evidence. |

## 10. Huawei Ascend vendor profile

### 10.1 Public support notes

Huawei Ascend support is not a generic NVIDIA-compatible path. vLLM uses the `vllm-ascend` plugin/image route. SGLang also documents Ascend NPU support and provides Ascend-specific setup instructions. Therefore Huawei should be modeled as a separate vendor profile with Ascend/CANN devices, mounts, images, and health behavior.

### 10.2 Common Huawei Ascend Docker runtime fields to model

Exact values must be verified against current Ascend/vllm-ascend/SGLang Ascend docs and local node configuration. Typical fields to model include:

- devices:
  - `/dev/davinci*`;
  - `/dev/davinci_manager`;
  - `/dev/devmm_svm`;
  - `/dev/hisi_hdc`;
- mounts:
  - `/usr/local/dcmi`;
  - Ascend driver/toolkit/CANN paths if required;
- env:
  - CANN / Ascend runtime env;
  - visible device controls such as `ASCEND_RT_VISIBLE_DEVICES`, if supported/required;
  - HCCL-related env for distributed runs;
- IPC/shm/security:
  - verify exact requirements from docs; do not copy MetaX defaults.

### 10.3 Huawei backend combinations

| Backend | Huawei status | Action |
|---|---|---|
| vLLM | public vllm-ascend plugin/images exist | Add/validate Huawei vLLM profile/catalog entry. |
| SGLang | public Ascend NPU docs exist | Add/validate Huawei SGLang profile/catalog entry. |
| llama.cpp | not primary target | Do not add runnable preset without evidence. |

## 11. Help catalog proposal

Each backend/vendor parameter should have externalized help.

Example file:

```text
configs/backend-catalog/help/vllm/vllm-v0.23.0.zh-CN.yaml
```

Example entry:

```yaml
- name: max_model_len
  title: 最大模型上下文长度
  summary: 控制 vLLM 为请求和 KV cache 规划的最大上下文长度。
  official_default: "由模型配置或 vLLM 默认推导，需以当前 image --help 为准。"
  lightai_recommendation: "小显存或多实例时建议限制；单模型单卡可先使用模型默认。"
  risk: "设置过大可能导致显存不足；设置过小会限制长上下文能力。"
  required: false
  advanced: false
  source:
    - https://docs.vllm.ai/en/stable/configuration/engine_args/
```

## 12. Source references

These are seed references for design discussion. Implementation must re-check target image help and vendor docs.

- vLLM Engine Arguments: https://docs.vllm.ai/en/stable/configuration/engine_args/
- vLLM Optimization and Tuning: https://docs.vllm.ai/en/stable/configuration/optimization/
- SGLang Server Arguments: https://docs.sglang.io/advanced_features/server_arguments.html
- SGLang Hyperparameter Tuning: https://sgl-project.github.io/advanced_features/hyperparameter_tuning.html
- llama.cpp Server README: https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md
- NVIDIA Container Toolkit specialized Docker configuration: https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/docker-specialized.html
- NVIDIA Container Toolkit user guide: https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/1.10.0/user-guide.html
- MetaX vLLM plugin: https://github.com/MetaX-MACA/vLLM-metax
- vLLM Ascend installation: https://docs.vllm.ai/projects/ascend/en/latest/installation.html
- SGLang Ascend NPU support: https://github.com/sgl-project/sglang/blob/main/docs/platforms/ascend/ascend_npu.md
- SGLang hardware support: https://docs.sglang.ai/
