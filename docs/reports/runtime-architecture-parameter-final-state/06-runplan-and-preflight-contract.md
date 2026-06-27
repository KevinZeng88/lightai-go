# RunPlan and Preflight Contract

## 1. 目标

RunPlan 和 Preflight 需要形成统一执行契约：

1. Preflight 判断是否能运行；
2. RunPlan 描述将如何运行；
3. Agent 按 RunPlan 创建 Docker spec；
4. evidence 证明实际执行与 RunPlan 一致；
5. UI/API 以同一套 errors/warnings/source map 展示。

## 2. Preflight 输入

Preflight 输入应包含：

1. deployment intent；
2. model artifact；
3. model location；
4. node backend runtime；
5. backend runtime snapshot；
6. backend capability profile；
7. runtime requirements；
8. parameter schema；
9. parameter values；
10. node evidence；
11. accelerator evidence；
12. port override；
13. mount override；
14. health check override。

## 3. Preflight 输出

推荐结构：

```json
{
  "status": "ok",
  "deployable": true,
  "errors": [],
  "warnings": [
    {
      "code": "version_probe_failed",
      "message": "Version probe failed but runtime can still be deployed.",
      "source": "node_backend_runtime.check"
    }
  ],
  "evidence": {
    "image": {
      "checked": true,
      "present": true,
      "source": "agent.image_inspect"
    },
    "model_path": {
      "checked": true,
      "exists": true,
      "source": "agent.files"
    },
    "parameters": {
      "checked": true,
      "valid": true
    },
    "ports": {
      "checked": true,
      "available": true
    },
    "devices": {
      "checked": true,
      "available": true
    }
  }
}
```

## 4. Preflight errors

Blocking errors 示例：

```text
image_missing
model_path_missing
model_format_unsupported
invalid_parameter
port_conflict
device_unavailable
mount_invalid
health_check_invalid
node_backend_runtime_not_ready
backend_runtime_snapshot_invalid
```

规则：

1. blocking error 禁止部署；
2. error 必须可读；
3. error 必须有 code；
4. error 必须有 source；
5. error 必须在 UI 展示；
6. error 必须可被 E2E 断言。

## 5. Preflight warnings

Warning 示例：

```text
version_probe_failed
health_check_slow
optional_endpoint_unverified
unknown_backend_version
ready_with_warnings
```

规则：

1. warning 不阻断 ready_with_warnings；
2. warning 必须展示；
3. warning 必须进入 evidence；
4. warning 不允许掩盖 blocking error。

## 6. RunPlan 输入

RunPlan 输入与 Preflight 一致，但会额外使用：

1. deployment resolved overrides；
2. system generated values；
3. resolved host port；
4. resolved device binding；
5. resolved mount path；
6. resolved health endpoint；
7. generated container name；
8. generated labels。

## 7. RunPlan 输出

推荐结构：

```json
{
  "schema_version": "resolved-runplan/v1",
  "image": "vllm/vllm-openai:latest",
  "command": null,
  "args": [
    "--model",
    "/models/Qwen3-0.6B-Instruct-2512",
    "--host",
    "0.0.0.0",
    "--port",
    "8000",
    "--gpu-memory-utilization",
    "0.85"
  ],
  "env": {
    "CUDA_VISIBLE_DEVICES": "0"
  },
  "ports": [
    {
      "host_port": 8004,
      "container_port": 8000,
      "protocol": "tcp"
    }
  ],
  "mounts": [
    {
      "source": "/home/kzeng/models/Qwen3-0.6B-Instruct-2512",
      "target": "/models/Qwen3-0.6B-Instruct-2512",
      "type": "bind",
      "read_only": true
    }
  ],
  "devices": [],
  "device_binding": {
    "vendor": "nvidia",
    "accelerator_ids": ["0"],
    "env": {
      "CUDA_VISIBLE_DEVICES": "0"
    }
  },
  "health_check": {
    "url": "http://127.0.0.1:8004/v1/models",
    "method": "GET",
    "success_status": [200],
    "timeout_seconds": 120
  },
  "source_map": {
    "image": "node_backend_runtime.snapshot.image",
    "args.--model": "model_location.path",
    "args.--gpu-memory-utilization": "deployment.parameter_values",
    "env.CUDA_VISIBLE_DEVICES": "device_binding",
    "ports.8000": "deployment.port_override"
  },
  "warnings": [],
  "errors": []
}
```

## 8. RunPlan 与 Docker spec 一致性

Agent Docker create spec 必须可从 RunPlan 机械转换。

需要 E2E 对比：

1. image；
2. command；
3. args；
4. env；
5. host config；
6. port bindings；
7. mounts；
8. devices；
9. gpus；
10. labels；
11. health check。

证据文件：

```text
runplan-preview.json
docker-create-spec.json
runplan-docker-diff.json
```

diff 要求：

1. 一致时 diff 为空或标记 PASS；
2. 不一致时列出字段；
3. 不一致导致 E2E 失败；
4. 允许 Docker SDK 自动补齐字段，但需明确忽略规则。

## 9. 合并顺序

RunPlan 合并顺序：

```text
BackendVersion defaults
→ BackendRuntime snapshot
→ NodeBackendRuntime snapshot
→ Deployment override
→ ModelLocation system values
→ DeviceBinding resolved values
→ Port resolved values
→ HealthCheck resolved values
→ Safety validation
```

## 10. 参数冲突处理

冲突规则：

1. 下游覆盖上游；
2. 同层重复为 error；
3. 类型不匹配为 error；
4. 非法范围为 error；
5. 未启用 optional 参数不进入 RunPlan；
6. required 参数缺失时使用 default；
7. required 且无 default 为 error；
8. 系统生成参数优先于用户覆盖。

## 11. HealthCheck 契约

HealthCheck 必须包含：

1. method；
2. path；
3. host；
4. host port；
5. timeout；
6. interval；
7. success status；
8. failure threshold；
9. source；
10. evidence。

OpenAI compatible backend 推荐使用：

```text
GET /v1/models
```

## 12. DeviceBinding 契约

DeviceBinding 推荐结构：

```json
{
  "vendor": "nvidia",
  "accelerator_ids": ["gpu-uuid-or-index"],
  "visible_device_ids": ["0"],
  "env": {
    "CUDA_VISIBLE_DEVICES": "0"
  },
  "docker": {
    "gpus": "device=0"
  },
  "devices": [],
  "warnings": []
}
```

要求：

1. 使用 vendor-neutral accelerator ids 作为内部对象；
2. Docker spec 使用 vendor runtime 需要的具体形式；
3. NVIDIA 支持 `CUDA_VISIBLE_DEVICES` / docker gpus；
4. MetaX native Docker 支持 `/dev/mxcd` 和 `/dev/dri`；
5. 不把 GPU vendor 写入 Backend / BackendVersion。

## 13. 禁止事项

1. preview 使用一套逻辑，Agent 使用另一套逻辑；
2. env 中混入 capabilities_json；
3. args 重复；
4. 模型路径写死在 RuntimeRequirements；
5. Preflight 通过但 RunPlan 缺必要字段；
6. RunPlan 通过但 Docker create 失败且无结构化错误；
7. health check 端口与 Docker port binding 不一致；
8. container id 与 instance id 混用。
