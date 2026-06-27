# Final Parameter Contract

## 1. 参数体系目标

参数体系需要同时满足：

1. 后端默认参数可表达；
2. 运行模板可编辑；
3. 节点运行配置可覆盖；
4. 部署可覆盖；
5. UI 能清晰显示 enabled/value/default/required；
6. API 能完整 round-trip；
7. RunPlan 能生成最终 Docker spec；
8. E2E 能断言最终结果。

## 2. 参数分层

### 2.1 BackendVersion 参数

BackendVersion 提供参数 schema 和推荐默认值。

用途：

1. 定义后端支持哪些参数；
2. 定义参数类型；
3. 定义参数进入 args/env/mounts/ports 的方式；
4. 定义默认值；
5. 定义 required/optional；
6. 定义校验规则；
7. 定义 UI 渲染提示。

BackendVersion 不保存用户实际选择。

### 2.2 BackendRuntime 参数

BackendRuntime 保存运行模板级参数值。

用途：

1. 表示某个运行模板的默认运行方式；
2. 可被 clone；
3. 可作为 NodeBackendRuntime 的初始值；
4. 可作为 Deployment 的上游 snapshot。

### 2.3 NodeBackendRuntime 参数

NodeBackendRuntime 保存节点级参数值。

用途：

1. 表示某个节点上运行该 runtime 的参数；
2. 适配节点设备、runtime、路径、env；
3. check-request 后记录节点 evidence；
4. 作为 Deployment 的唯一上游 runtime 输入。

### 2.4 Deployment 参数

Deployment 保存部署级覆盖值。

用途：

1. 针对当前模型和当前服务进行覆盖；
2. 覆盖端口、资源、上下文、显存比例等；
3. 保存 copy-on-create snapshot；
4. 生成最终 RunPlan。

### 2.5 ResolvedRunPlan 参数

ResolvedRunPlan 保存最终执行结果。

用途：

1. 明确 Docker args；
2. 明确 env；
3. 明确 mounts；
4. 明确 ports；
5. 明确 devices；
6. 明确 health check；
7. 明确 source map；
8. 明确 warnings/errors。

## 3. 参数 schema 结构

推荐 schema 结构：

```json
{
  "schema_version": "runtime-parameter/v1",
  "parameters": [
    {
      "key": "gpu_memory_utilization",
      "display_name": "GPU memory utilization",
      "description": "Fraction of GPU memory used by vLLM.",
      "type": "number",
      "default": 0.9,
      "required": false,
      "min": 0.1,
      "max": 1.0,
      "ui": {
        "group": "resource_controls",
        "order": 10,
        "advanced": false
      },
      "binding": {
        "target": "args",
        "arg_name": "--gpu-memory-utilization",
        "style": "flag_value"
      },
      "applies_to": {
        "backends": ["vllm"],
        "model_formats": ["huggingface"]
      }
    }
  ]
}
```

## 4. 参数 value 结构

推荐 value 结构：

```json
{
  "schema_version": "runtime-parameter-values/v1",
  "values": {
    "gpu_memory_utilization": {
      "enabled": true,
      "value": 0.85,
      "source": "deployment",
      "updated_at": "2026-06-27T00:00:00Z"
    },
    "max_model_len": {
      "enabled": false,
      "value": 8192,
      "source": "backend_runtime"
    }
  }
}
```

## 5. enabled/value 语义

### 5.1 enabled

`enabled` 表示该参数是否由当前层显式参与覆盖。

规则：

1. enabled=true：当前层显式覆盖上游值；
2. enabled=false：当前层不覆盖上游值；
3. required 参数可以在最终 RunPlan 中使用 default；
4. optional 且未 enabled 的参数不进入最终 args；
5. 系统生成参数不依赖 UI enabled，例如 `--model` 来自 ModelLocation。

### 5.2 value

`value` 表示当前层保存的候选值或显式值。

规则：

1. disabled input 也显示 value；
2. disabled 状态下 value 可保存；
3. 勾选 enabled 后使用已有 value；
4. value 不因 enabled=false 丢失；
5. clone 必须保留 enabled + value；
6. refresh 后必须完整 round-trip。

## 6. 参数合并顺序

最终合并顺序：

```text
BackendVersion schema/defaults
→ BackendRuntime values
→ NodeBackendRuntime values
→ Deployment values
→ System-resolved values
→ Runtime safety validation
```

优先级：

1. System-resolved values 最高，用于模型路径、端口绑定结果、device binding 等系统事实；
2. Deployment values 覆盖 NBR；
3. NBR values 覆盖 BackendRuntime；
4. BackendRuntime values 覆盖 BackendVersion defaults；
5. schema 负责校验和绑定规则。

## 7. 参数进入 Docker spec 的规则

### 7.1 args

binding 示例：

```json
{
  "target": "args",
  "arg_name": "--max-model-len",
  "style": "flag_value"
}
```

生成：

```text
--max-model-len 8192
```

### 7.2 env

binding 示例：

```json
{
  "target": "env",
  "env_name": "CUDA_VISIBLE_DEVICES"
}
```

生成：

```text
CUDA_VISIBLE_DEVICES=0
```

### 7.3 mounts

binding 示例：

```json
{
  "target": "mounts",
  "mount_type": "bind",
  "container_path": "/models"
}
```

### 7.4 ports

binding 示例：

```json
{
  "target": "ports",
  "container_port": 8000,
  "protocol": "tcp"
}
```

### 7.5 health check

binding 示例：

```json
{
  "target": "health_check",
  "field": "path"
}
```

## 8. 参数去重

RunPlan 生成必须去重：

1. 同一 flag 不重复；
2. 同一 env key 不重复；
3. 同一 container port 不重复；
4. 同一 mount target 不冲突；
5. 同一 device path 不重复。

冲突处理：

1. 同层重复为 validation error；
2. 下游覆盖上游；
3. 系统事实覆盖用户输入；
4. 类型不匹配为 blocking error；
5. warning 不得掩盖实际冲突。

## 9. 后端参数示例

### 9.1 vLLM

必备系统参数：

1. `--model`：来自 ModelLocation；
2. `--host`：默认 `0.0.0.0`；
3. `--port`：容器端口，默认 `8000`。

常用可编辑参数：

1. `--gpu-memory-utilization`；
2. `--max-model-len`；
3. `--dtype`；
4. `--quantization`；
5. `--tensor-parallel-size`；
6. `--served-model-name`；
7. `--trust-remote-code`。

### 9.2 SGLang

必备系统参数：

1. `--model-path`：来自 ModelLocation；
2. `--host`：默认 `0.0.0.0`；
3. `--port`：容器端口，默认 `30000`。

常用可编辑参数：

1. `--mem-fraction-static`；
2. `--context-length`；
3. `--dtype`；
4. `--tp-size`；
5. `--served-model-name`；
6. `--trust-remote-code`。

### 9.3 llama.cpp

必备系统参数：

1. `--model` 或 `-m`：来自 ModelLocation；
2. `--host`：默认 `0.0.0.0`；
3. `--port`：容器端口，默认 `8000`。

常用可编辑参数：

1. `--ctx-size`；
2. `--n-gpu-layers` 或 `-ngl`；
3. `--batch-size`；
4. `--ubatch-size`；
5. `--parallel`；
6. `--cont-batching`。

## 10. UI 渲染规则

1. 参数按 group 分组；
2. 每个参数显示 label、description、当前值、默认值；
3. disabled 状态仍显示输入框；
4. enabled checkbox 控制当前层是否覆盖；
5. required 参数显示 required 标识；
6. invalid value 立即显示错误；
7. 保存后刷新不丢 schema；
8. 保存后刷新不丢 value；
9. clone 后保留 enabled + value；
10. preview 显示最终结果和 source map。

## 11. API 规则

API response 必须能支持前端完整渲染：

1. schema；
2. values；
3. merged values；
4. source map；
5. validation errors；
6. warnings；
7. effective run args；
8. effective env；
9. effective ports；
10. effective mounts。

## 12. 测试要求

必须覆盖：

1. enabled=false value round-trip；
2. enabled=true override；
3. required default；
4. deployment override；
5. clone；
6. refresh；
7. invalid type；
8. invalid range；
9. args 去重；
10. env 去重；
11. preview 与 Docker spec 一致。
