# 01 - 正式 E2E 要求与能力基线（Formal E2E Requirements）

> 目的：定义 LightAI Go 正式 E2E 的最低质量要求、能力要求、脚本安全要求和 artifact 要求。  
> 适用范围：所有新增或被纳入正式验收链路的 E2E 脚本。  
> 重要说明：不满足本文档要求的脚本可以保留为 smoke / legacy / debug 脚本，但不得作为主验收 PASS 的依据。

---

## 1. E2E 分类

所有 E2E 脚本必须标注类别。一个脚本可以属于多个类别，但必须明确主要用途。

### 1.1 Smoke E2E

验证基础链路是否可用，例如：

- server/agent 能启动；
- API 能登录；
- 能创建基础对象；
- 默认链路能跑通。

Smoke E2E 不能替代参数传播、页面入口、响应语义验收。

### 1.2 DryRun E2E

不启动容器，不真实加载模型，只验证：

- API payload；
- DB/detail API；
- Deployment snapshot；
- Preflight；
- RunPlan；
- Docker command preview；
- Docker create spec；
- 参数来源优先级；
- 反向断言。

DryRun E2E 是日常回归中最重要、最高性价比的 E2E 类型。

### 1.3 Runtime E2E

真实启动容器，验证：

- Agent 接收任务；
- Docker create/start；
- health check；
- model instance 状态；
- container_id；
- GPU lease；
- stop/cleanup。

### 1.4 Inference E2E

在 Runtime E2E 基础上真实请求模型，验证：

- `/v1/models`；
- `/v1/chat/completions`；
- instance test API；
- raw_response；
- parsed summary；
- 多种响应字段解析。

### 1.5 Failed-state E2E

构造失败场景，验证：

- failed 状态；
- container_id 保留；
- exit_code；
- failure_reason_code；
- logs tail；
- last_error；
- failed 状态下日志可获取。

### 1.6 Matrix Verifier

不是简单 runner，而是对多个 backend/default/modified 组合进行强断言：

- vLLM；
- SGLang；
- llama.cpp；
- NVIDIA；
- MetaX；
- Huawei；
- default；
- modified；
- custom；
- disabled；
- conflict。

### 1.7 Legacy / Local E2E

历史脚本或本地调试脚本可以保留，但必须标注：

- legacy；
- 不作为当前主验收依据；
- 依赖旧 API 或直接 DB 的部分要说明；
- 可迁移的断言应迁移到当前正式 E2E。

---

## 2. 脚本安全要求

正式 E2E 脚本必须：

1. 使用 `set -euo pipefail`。
2. 在脚本头部说明：
   - 目的；
   - 分类；
   - 是否启动服务；
   - 是否启动容器；
   - 是否需要 GPU；
   - 是否修改 DB；
   - 是否 destructive cleanup；
   - 依赖的模型路径/镜像；
   - 预计 artifact 目录。
3. 支持环境变量配置：
   - `LIGHTAI_SERVER_URL` 或 `SERVER_URL`；
   - `LIGHTAI_E2E_USERNAME`；
   - `LIGHTAI_E2E_PASSWORD`；
   - `LIGHTAI_E2E_ARTIFACT_DIR`；
   - `LIGHTAI_E2E_RUN_ID`；
   - `LIGHTAI_E2E_SKIP_REAL_RUNTIME`；
   - backend-specific image/model path/port 变量。
4. artifact 目录必须唯一，不能覆盖历史结果。
5. cleanup 必须只清理本轮创建的资源。
6. 不得误删用户已有 model root / deployment / runtime / container。
7. 启动 server/agent 的脚本必须有安全 PID 管理。
8. 不得使用危险的 `pkill` 或宽泛 docker rm。
9. 如果复用已有 server/agent，必须标注。
10. 环境不满足时必须返回 `SKIPPED_ENV`，不能伪造 PASS。

---

## 3. 断言要求

正式 E2E 必须有明确断言函数或等价机制：

- `assert_eq`
- `assert_not_eq`
- `assert_nonempty`
- `assert_empty`
- `assert_contains`
- `assert_not_contains`
- `assert_json_field`
- `assert_json_eq`
- `assert_http_status`
- `assert_exactly_one_flag`
- `assert_flag_value`
- `assert_no_duplicate_flag`
- `assert_command_contains`
- `assert_command_not_contains`
- `assert_artifact_exists`

关键步骤失败必须 `exit 1`。

### 3.1 不允许的 false pass

正式 E2E 中不允许：

1. 关键步骤 `|| true`。
2. curl 不检查 HTTP code。
3. grep 失败但脚本继续。
4. jq/python 取空但脚本继续。
5. test/logs 失败只打印 `skipped`。
6. 最后打印 FAIL 但 exit 0。
7. matrix 子项失败但总结果 PASS。
8. 只保存 response，不检查语义。
9. 只检查某个数字存在，不检查字段含义。
10. start 失败但仍写 summary PASS。

cleanup 阶段可以 `|| true`，但必须注释说明，并且不能影响正式断言结果。

---

## 4. Artifact 要求

正式 E2E 必须保存 artifact。至少包括：

1. request payload；
2. response body；
3. HTTP status；
4. preflight response；
5. deployment detail；
6. model instance detail；
7. RunPlan JSON；
8. Docker command preview；
9. Docker create spec；
10. Docker inspect ports/env/mounts/devices，如运行容器；
11. `/v1/models` response；
12. `/v1/chat/completions` raw response；
13. instance test raw_response；
14. parsed summary；
15. logs tail；
16. assertion summary；
17. cleanup result；
18. final summary。

建议 artifact 目录：

```text
docs/reports/model-runtime-node-wizard/e2e-artifacts/<script-name>-<run-id>/
```

每个脚本必须输出 artifact 目录路径。

---

## 5. 参数传播要求

正式 E2E 必须覆盖参数从用户输入到最终执行的传播链：

```text
用户输入 / API payload
  -> DB/detail API
  -> Deployment snapshot
  -> Preflight
  -> RunPlan
  -> Docker command preview
  -> Docker create spec
  -> Runtime health/test target
```

必须确认：

1. 用户显式值优先于模板默认值。
2. 默认值只能作为 fallback。
3. disabled 参数不渲染。
4. custom 参数按类型渲染到正确位置。
5. 同一 flag 最终只有一个值。
6. 默认错误值不得残留。
7. preview 与 create spec 同源或一致。
8. health/test 使用正确 host_port。
9. app args 使用 app_port。
10. container_port 默认等于 app_port；如果不等，必须 warning/error，除非产品显式支持 sidecar/proxy。

---

## 6. 必须覆盖的用户真实链路

正式 E2E 至少应覆盖以下链路中的关键路径。

### 6.1 运行模板复制与编辑

```text
内置模板
  -> clone
  -> 修改 image/env/devices/volumes/app args/ports
  -> 保存
  -> 重新 GET
  -> 部署选择
  -> DryRun
  -> RunPlan 使用 clone 后值
  -> 原内置模板不变
```

### 6.2 模型文件与模型位置

```text
添加/选择 model root
  -> browse files
  -> scan model path
  -> create model artifact
  -> create model location
  -> validate file/directory/format
```

GGUF 必须按单文件处理。HF/safetensors 通常按目录处理。

### 6.3 Deployment 创建与可见性

```text
create deployment
  -> list 可见
  -> detail 可读
  -> dry-run
  -> start
  -> start 后 list 仍可见
  -> status/active_instance/current_run_plan 存在
```

### 6.4 Runtime 启动与健康检查

```text
start
  -> instance created
  -> run_plan created
  -> docker create/start
  -> health check
  -> running
```

### 6.5 Inference 测试

```text
/v1/models
  -> /v1/chat/completions
  -> instance test API
  -> raw_response
  -> parsed summary
```

### 6.6 停止与清理

必须分别覆盖：

- deployment stop；
- instance stop；
- force stop，如已实现。

---

## 7. Backend-specific 能力要求

### 7.1 vLLM

必须验证：

- positional model；
- 不默认使用 `--model`；
- 不同时出现 positional model 和 `--model`；
- `--host`；
- `--port`；
- `--served-model-name`；
- `--tensor-parallel-size`；
- `--gpu-memory-utilization`；
- `--max-model-len`；
- `--trust-remote-code` disabled/enabled；
- `--dtype`；
- `--enforce-eager`；
- startup_timeout_seconds；
- health check target；
- Docker port mapping；
- visible device env。

### 7.2 SGLang

必须验证：

- `--model-path`；
- `--host`；
- `--port`；
- `--tp-size` 或当前版本参数；
- `--mem-fraction-static`；
- `--context-length`；
- `--served-model-name`；
- `--trust-remote-code` disabled/enabled；
- reasoning/tool parser，如支持；
- startup timeout；
- health check target。

### 7.3 llama.cpp

必须验证：

- GGUF 单文件；
- `-m` 或 `--model` 指向容器内 `.gguf` 文件；
- `--host`；
- `--port`；
- `-ngl` / `--n-gpu-layers`；
- `--ctx-size`；
- `--parallel`；
- `--threads`；
- LLAMA_ARG_* 与 CLI args 不冲突；
- OpenAI-compatible API。

---

## 8. Vendor-specific 能力要求

### 8.1 NVIDIA

必须验证：

- GPU ID / index 解析正确；
- Docker `--gpus device=...` 或 DeviceRequest 正确；
- `CUDA_VISIBLE_DEVICES` 与实际 GPU 选择一致；
- 不出现内部 DB UUID 被传给 NVIDIA runtime 的问题。

### 8.2 MetaX

必须验证：

- devices 包含 `/dev/dri`、`/dev/mxcd`、`/dev/infiniband` 等；
- devices 不使用 volume-style `/dev/dri:/dev/dri`，除非结构化 device mapping 明确支持；
- 使用 `MACA_VISIBLE_DEVICE`；
- 不将 `CUDA_VISIBLE_DEVICES` 作为唯一默认可见设备机制；
- `ipc/privileged/security_opt/shm_size/ulimits/group_add` 等参数分类正确。

### 8.3 Huawei

必须验证：

- Ascend visible device env；
- devices；
- Docker options；
- 不混用 NVIDIA/MetaX 默认参数。

---

## 9. Command preview / create spec 一致性要求

正式 E2E 必须验证：

1. command preview 与 RunPlan 一致。
2. command preview 与 Docker create spec 一致。
3. image 一致。
4. ports 一致。
5. env 一致。
6. volumes 一致。
7. devices 一致。
8. docker options 一致。
9. app args 一致。
10. disabled/custom/high-risk 参数处理一致。

---

## 10. 状态与错误诊断要求

正式 E2E 必须验证：

1. start 失败时保留 last_error。
2. failed instance 保留 container_id。
3. failed instance 保留 failure_reason_code。
4. logs tail 可获取。
5. health timeout 与 runtime error 可区分。
6. 平台主动 terminated 容器时，诊断中说明 startup/health timeout。
7. deployment 与 instance 状态同步。
8. GPU lease 释放。
9. audit/log 可追踪关键操作。

---

## 11. 输出状态要求

正式 E2E 输出状态必须区分：

- `PASS`
- `FAIL`
- `SKIPPED_ENV`
- `WEAK_PASS`

`WEAK_PASS` 表示脚本跑完但断言不足，不得作为主验收 PASS。
