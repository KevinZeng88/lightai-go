# 02 - 验收标准与参数矩阵（Acceptance Criteria and Parameter Matrix）

> 目的：定义 E2E 改造完成后的验收标准。  
> 重点：很多验收可以通过 DryRun / RunPlan / Docker command preview 完成，不要求每项都真实启动模型。  
> 原则：先用低成本 DryRun 抓参数传播与命令错误，再用 selected runtime E2E 验证关键真实链路。

---

## 1. 总体验收标准

E2E 改造完成后，必须能发现以下历史问题：

1. vLLM `app_port=8022` 但最终 `--port 8000`。
2. vLLM positional model 与 `--model` 重复。
3. 用户参数被 default_args / ParameterDef default 覆盖。
4. Deployment start 后 list 为空。
5. 模型实例页 stop HTTP 405。
6. raw_response 非空但 parsed summary 空。
7. clone template 后修改参数没有保存。
8. GGUF 被误当目录或 format 错误。
9. test/logs 失败但 E2E 仍 PASS。
10. matrix 子项失败但总结果 PASS。
11. preview 与 create spec 不一致。
12. disabled 参数被错误渲染。
13. custom 参数进入错误位置。
14. GPU visible env 与 Docker GPU 选择不一致。

---

## 2. 运行分层验收

### 2.1 静态 / 语法验收

必须通过：

```bash
bash -n scripts/*.sh scripts/e2e/*.sh scripts/e2e/lib/*.sh
```

验收：

- 所有正式 E2E 脚本语法正确；
- helper 可被 source；
- 脚本头部有分类和依赖说明。

### 2.2 DryRun 参数传播验收

不启动容器，必须通过：

```bash
scripts/e2e-runplan-parameter-source-audit.sh
```

或等价脚本。

验收：

- API payload 保存；
- Preflight 保存；
- Deployment detail 保存；
- RunPlan 保存；
- command preview 保存；
- assertion summary 保存；
- 所有参数断言通过。

### 2.3 Clone template 验收

不要求真实启动容器，必须验证：

```text
clone builtin runtime
  -> 修改参数
  -> 保存
  -> GET clone detail
  -> DryRun
  -> command 使用 clone 参数
  -> 原 builtin 不变
```

### 2.4 Deployment visibility 验收

可通过 API 或 selected runtime 验证：

```text
create deployment
  -> list contains deployment
  -> start
  -> list still contains deployment
  -> active_instance_id/current_run_plan/status present
```

### 2.5 Instance stop 验收

需要至少一个真实运行实例，优先 llama.cpp selected runtime：

```text
running instance
  -> POST instance stop
  -> HTTP != 405
  -> state stopped
  -> container stopped/removed
  -> lease released
  -> deployment status synced
```

### 2.6 Inference response parser 验收

至少覆盖真实 llama.cpp 或 mock/raw response fixture：

```text
raw_response present
parsed summary non-empty
reasoning_content/text/content variants supported
raw_response non-empty + summary empty => FAIL
```

### 2.7 Runtime selected 验收

至少保留一个可真实运行的 selected runtime E2E：

- llama.cpp GGUF selected runtime，优先；
- vLLM/SGLang runtime 可按环境标记 SKIPPED_ENV，但 DryRun 必须通过。

---

## 3. vLLM 参数验收矩阵

### 3.1 vLLM custom port propagation

输入：

```json
{
  "service_json": {
    "host_port": 8111,
    "container_port": 8022,
    "app_port": 8022
  }
}
```

验收：

| 项 | 预期 |
|---|---|
| Docker port mapping | `-p 8111:8022` 或 create spec 等价结构 |
| app args | `--port 8022` |
| command preview | 包含 `--port 8022` |
| 反向断言 | 不包含 `--port 8000` |
| 重复断言 | `--port` exactly once |
| health target | `http://127.0.0.1:8111/...` |
| container/app 关系 | `container_port == app_port`，否则 warning/error |

### 3.2 vLLM model path

输入：

```text
model_container_path=/models/Qwen3-0.6B-Instruct-2512
```

验收：

| 项 | 预期 |
|---|---|
| positional model | 出现在 image 后第一个模型位置参数或符合模板定义 |
| `--model` | 默认不出现 |
| 重复 | 不同时出现 positional model 和 `--model` |
| host path | 不出现在容器 app args 中 |
| volume | host path mount 到 container path |

### 3.3 vLLM common app args

输入与验收：

| 参数 | 输入示例 | 预期 command |
|---|---:|---|
| `served_model_name` | `qwen-vllm-e2e` | `--served-model-name qwen-vllm-e2e` |
| `gpu_memory_utilization` | `0.85` | `--gpu-memory-utilization 0.85` |
| `max_model_len` | `4096` | `--max-model-len 4096` |
| `tensor_parallel_size` | `1` 或 `2` | `--tensor-parallel-size <value>` |
| `trust_remote_code` disabled | false | 不出现 `--trust-remote-code` |
| `trust_remote_code` enabled | true | 出现 `--trust-remote-code` |
| `enforce_eager` enabled | true | 出现 `--enforce-eager` |
| `dtype` | `float16`/`bfloat16` | `--dtype <value>` |

反向验收：

- 不被 default 覆盖；
- 不重复；
- disabled 不渲染；
- 参数名 snake_case / CLI format 均可解析。

---

## 4. SGLang 参数验收矩阵

### 4.1 SGLang custom port

输入：

```json
{
  "service_json": {
    "host_port": 30111,
    "container_port": 30022,
    "app_port": 30022
  }
}
```

验收：

| 项 | 预期 |
|---|---|
| Docker port mapping | `-p 30111:30022` |
| app args | `--port 30022` |
| 反向断言 | 不包含默认 `--port 30000` 或模板默认值 |
| 重复断言 | `--port` exactly once |

### 4.2 SGLang model path

| 项 | 预期 |
|---|---|
| model path arg | `--model-path <container_path>` |
| host path | 不进入 app args |
| HF model | 通常 path_type=directory |
| volume | host directory mount to container directory |

### 4.3 SGLang common app args

| 参数 | 输入示例 | 预期 |
|---|---:|---|
| tp-size / tensor-parallel-size | `1` 或 `2` | 使用当前模板版本正确 flag |
| mem-fraction-static | `0.80` | app args 使用用户值 |
| context-length | `4096` | app args 使用用户值 |
| served-model-name | `qwen-sglang-e2e` | app args 使用用户值 |
| trust-remote-code disabled | false | 不出现 |
| trust-remote-code enabled | true | 出现 |
| disable-cuda-graph | true/false | 按 enabled 状态渲染 |

---

## 5. llama.cpp 参数验收矩阵

### 5.1 GGUF file semantics

输入：

```text
MODEL=/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

验收：

| 项 | 预期 |
|---|---|
| artifact format | `gguf` |
| location path_type | `file` |
| scan file/directory | 若选择文件则 file；若选择目录则必须选具体 GGUF |
| volume | mount file 或 parent directory，语义清晰 |
| `-m` / `--model` | 指向容器内 `.gguf` 文件 |
| 反向断言 | 不指向目录 |

### 5.2 llama.cpp custom port

输入：

```json
{
  "service_json": {
    "host_port": 18081,
    "container_port": 18082,
    "app_port": 18082
  }
}
```

验收：

| 项 | 预期 |
|---|---|
| Docker port mapping | `-p 18081:18082` |
| app args | `--port 18082` |
| 反向断言 | 不包含默认 `--port 8080` 或旧默认 |
| 重复断言 | `--port` exactly once |

### 5.3 llama.cpp common app args

| 参数 | 输入示例 | 预期 |
|---|---:|---|
| ctx-size | `4096` | `--ctx-size 4096` |
| n-gpu-layers | `99` 或 `-1` | `-ngl 99` / `--n-gpu-layers 99` 按模板 |
| parallel | `2` | `--parallel 2` |
| threads | `8` | `--threads 8` |
| host | `0.0.0.0` | `--host 0.0.0.0` exactly once |

反向验收：

- `LLAMA_ARG_HOST` 与 `--host` 不冲突；
- `LLAMA_ARG_PORT` 与 `--port` 不冲突；
- `LLAMA_ARG_MODEL` 与 `-m/--model` 不冲突。

---

## 6. Docker 参数分类验收矩阵

### 6.1 Devices

| 输入 | 预期 |
|---|---|
| device `/dev/dri` | 渲染为 `--device /dev/dri` |
| device `/dev/mxcd` | 渲染为 `--device /dev/mxcd` |
| device `/dev/infiniband` | 渲染为 `--device /dev/infiniband` |
| `/dev/dri:/dev/dri` | 默认不得作为 simple device 出现，除非结构化 mapping |

### 6.2 Volumes

| 输入 | 预期 |
|---|---|
| host model path -> container path | `-v host:container:ro` |
| custom volume | `-v` |
| disabled volume | 不渲染 |

### 6.3 Env

| 输入 | 预期 |
|---|---|
| `CUDA_VISIBLE_DEVICES=0` | NVIDIA 场景可出现，且与 GPU 选择一致 |
| `MACA_VISIBLE_DEVICE=0` | MetaX 场景使用 |
| `ASCEND...` | Huawei 场景使用 |
| custom env | `-e KEY=value` |
| disabled env | 不渲染 |

### 6.4 Docker options

| 参数 | 位置 |
|---|---|
| `--ipc=host` | image 前 |
| `--shm-size` | image 前 |
| `--privileged` | image 前 |
| `--security-opt` | image 前 |
| `--group-add` | image 前 |
| `--ulimit` | image 前 |
| `--network` | image 前 |

### 6.5 App args

| 参数 | 位置 |
|---|---|
| vLLM/SGLang/llama.cpp app args | image 后 |
| custom app args | image 后 |
| disabled app args | 不渲染 |

---

## 7. GPU / Vendor 验收矩阵

### 7.1 NVIDIA

输入：

```text
gpu index = 0
```

验收：

| 项 | 预期 |
|---|---|
| Docker GPU | `--gpus device=0` 或 DeviceRequest 等价 |
| CUDA_VISIBLE_DEVICES | `0` |
| 一致性 | Docker GPU 与 env 一致 |
| 反向断言 | 不传内部 DB UUID 给 NVIDIA runtime |

### 7.2 MetaX

验收：

| 项 | 预期 |
|---|---|
| visible env | `MACA_VISIBLE_DEVICE` |
| devices | `/dev/dri`、`/dev/mxcd`、`/dev/infiniband` |
| docker options | 按模板分类 |
| 反向断言 | 不把 CUDA_VISIBLE_DEVICES 作为唯一 visible device 机制 |
| device style | 不出现 volume-style simple devices |

### 7.3 Huawei

验收：

| 项 | 预期 |
|---|---|
| visible env | 项目定义的 Ascend env |
| devices | 项目定义的 Ascend devices |
| 反向断言 | 不混入 NVIDIA/MetaX 不适用参数 |

---

## 8. Clone template 验收矩阵

输入：clone 内置模板，并在 clone payload 中修改：

- display/template name；
- image；
- env；
- devices；
- volumes；
- docker options；
- app args；
- ports；
- startup timeout；
- health check；
- custom args。

验收：

| 项 | 预期 |
|---|---|
| clone response | 有新 runtime id |
| GET clone detail | 修改值保留 |
| original builtin | 不变 |
| deployment selection | clone 可选 |
| DryRun | 使用 clone 修改值 |
| command preview | 使用 clone 修改值 |
| disabled 参数 | 不渲染 |
| high-risk 参数 | 状态保留 |

---

## 9. Deployment visibility 验收

| 步骤 | 预期 |
|---|---|
| create deployment | 返回 id |
| list deployments | 包含该 id |
| detail deployment | 字段完整 |
| dry-run | 有 runplan/preview |
| start | 返回 instance_id/run_plan_id |
| start 后 list | 仍包含该 deployment |
| status | starting/running/stopped/failed 合理 |
| active_instance | 存在 |
| current_run_plan | 存在 |

此验收必须能抓住 SELECT/Scan 列不一致导致 list 返回空的问题。

---

## 10. Instance stop / force stop 验收

### 10.1 Instance stop

| 步骤 | 预期 |
|---|---|
| running instance | 有 instance_id |
| POST instance stop | HTTP 200/202，不能 405 |
| instance state | stopped/stopping -> stopped |
| container | stopped/removed |
| GPU lease | released |
| deployment status | 同步 |
| audit/log | 有 stop 记录 |

### 10.2 Force stop

如已实现：

| 步骤 | 预期 |
|---|---|
| force stop confirmation | UI/API 有二次确认或明确接口 |
| POST force stop | 成功 |
| container | kill/remove |
| GPU lease | released |
| status | stopped/failed_force_stopped 等项目定义 |
| audit/log | force=true/reason/operator |

---

## 11. Inference response parser 验收

必须覆盖以下 raw response fixture 或真实响应：

### 11.1 OpenAI message content

```json
{
  "choices": [
    {
      "message": {
        "content": "hello"
      }
    }
  ]
}
```

预期：summary = `hello`

### 11.2 reasoning_content

```json
{
  "choices": [
    {
      "message": {
        "reasoning_content": "thinking..."
      }
    }
  ]
}
```

预期：summary 非空，不判失败。

### 11.3 choices text

```json
{
  "choices": [
    {
      "text": "hello"
    }
  ]
}
```

预期：summary = `hello`

### 11.4 delta content

```json
{
  "choices": [
    {
      "delta": {
        "content": "hello"
      }
    }
  ]
}
```

预期：summary 非空。

### 11.5 top-level response

```json
{
  "response": "hello"
}
```

预期：summary = `hello`

### 11.6 raw_response 非空但 summary 空

预期：E2E 必须 FAIL，并保存 raw_response。

---

## 12. Matrix verifier 验收

matrix 不再只是跑脚本。必须输出：

| backend | scenario | status | assertions | artifact |
|---|---|---|---|---|

要求：

- default scenario 有最小断言；
- modified scenario 必须验证用户参数进入 RunPlan/command；
- failed 子项导致总结果 FAIL；
- 环境缺失标记 SKIPPED_ENV；
- 断言弱但脚本跑通标记 WEAK_PASS；
- WEAK_PASS 不等于 PASS；
- summary JSON/MD 都保存。

---

## 13. 文档验收

改造完成后必须更新：

1. `docs/testing/backend-runtime-e2e-matrix-and-param-propagation.md`
2. `docs/testing/model-runtime-gpu-smoke-tests.md`
3. `docs/reports/model-runtime-node-wizard/e2e-improvement/`
4. 新增 artifact README 或执行说明，如需要。

---

## 14. 最终验收命令建议

按风险递进：

```bash
git diff --check
bash -n scripts/*.sh scripts/e2e/*.sh scripts/e2e/lib/*.sh
go test ./...
go vet ./...
go build ./...
npm --prefix web run build
npm --prefix web test -- --runInBand
scripts/e2e-runplan-parameter-source-audit.sh
scripts/e2e-clone-template-parameter-persistence.sh
scripts/e2e-deployment-visibility-selected.sh
scripts/e2e-instance-stop-selected.sh
scripts/e2e-inference-response-parser-selected.sh
scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

vLLM/SGLang 真实 runtime E2E 可根据环境执行；如果环境不满足，必须 SKIPPED_ENV，不得伪造 PASS。
