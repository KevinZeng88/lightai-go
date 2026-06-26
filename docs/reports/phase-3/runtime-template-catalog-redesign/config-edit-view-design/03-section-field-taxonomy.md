# 03 - 分组、顺序与字段归属规范

## 1. 总体分组

统一 section 顺序：

```text
010 basic
020 model_serving
030 backend_runtime
040 container_resources
050 devices_mounts
060 environment
070 service
080 health_check
090 advanced_raw
```

## 2. 分组说明

### basic - 基础信息

字段：display_name、name/slug、backend、backend_version、vendor、image、source、status。内置通用运行模板 Backend Version 显示 `*`。

### model_serving - 模型服务参数

字段：served_model_name、max_model_len/context_length/ctx_size、gpu_memory_utilization/mem_fraction_static、tensor_parallel_size/tp_size、max_num_seqs、max_num_batched_tokens、max_running_requests、dtype、quantization、trust_remote_code、chat_template。

默认来源：`backend.arg.*`、`model_runtime.*`。

### backend_runtime - 后端进程运行参数

字段：command、entrypoint、extra_args、working_dir、host、container_port、log_level。

来源：`launcher.command`、`launcher.entrypoint`、部分 backend arg。

### container_resources - 容器资源

字段：shm_size、privileged、ipc_mode、uts_mode、network_mode、restart_policy、security_options、ulimits、memory_limit、cpu_limit。

来源：`launcher.docker_options`。

### devices_mounts - 设备与挂载

字段：GPU selection、devices、optional_devices、group_add、volumes、model_mount、cache_mount、data_mount。

来源：`launcher.docker_options.devices`、`runtime.model_mount` 等。

### environment - 环境变量

字段：CUDA_VISIBLE_DEVICES、ASCEND_VISIBLE_DEVICES、HF_HOME、HF_ENDPOINT、VLLM_USE_MODELSCOPE、PYTORCH_CUDA_ALLOC_CONF、LD_LIBRARY_PATH、自定义 env。

来源：`runtime.env`、`kind=env`。

### service - 服务入口

字段：container_port、host_port、protocol、endpoint path、OpenAI compatible base path、service URL。

来源：`deployment.service_json`、`service.*`、`runtime.ports`。

### health_check - 健康检查

字段：path、method、expected_status、timeout、interval、retries、startup_timeout、model probe path。

来源：`runtime.health`。

### advanced_raw - 高级原始配置

字段：raw ConfigSet JSON、source_metadata、raw docker options、raw env、raw args、resolved RunPlan preview。默认折叠。

## 3. internal key 映射规则

| Internal Key | Section | Field |
| --- | --- | --- |
| `launcher.image` | basic | image |
| `launcher.command` | backend_runtime | command |
| `launcher.entrypoint` | backend_runtime | entrypoint |
| `launcher.docker_options.shm_size` | container_resources | shm_size |
| `launcher.docker_options.privileged` | container_resources | privileged |
| `launcher.docker_options.ipc_mode` | container_resources | ipc_mode |
| `launcher.docker_options.uts_mode` | container_resources | uts_mode |
| `launcher.docker_options.security_options` | container_resources | security_options |
| `launcher.docker_options.ulimits` | container_resources | ulimits |
| `launcher.docker_options.devices` | devices_mounts | devices |
| `launcher.docker_options.optional_devices` | devices_mounts | optional_devices |
| `launcher.docker_options.group_add` | devices_mounts | group_add |
| `runtime.model_mount` | devices_mounts | model_mount |
| `runtime.env` | environment | env |
| `runtime.health` | health_check | health_check |
| `backend.arg.*` | model_serving | backend argument |
| `service.*` | service | service |
| `source_metadata.*` | advanced_raw | diagnostics |
| `internal.*` | advanced_raw | diagnostics |
| `resolver.*` | advanced_raw | diagnostics |

## 4. 优先级规则

字段归组优先级：

```text
1. item.render.section
2. item.extensions.section
3. taxonomy by internal key
4. item.category
5. advanced_raw
```

label 优先级：

```text
1. item.render.label
2. item.extensions.label
3. taxonomy label
4. humanize(code)
```

## 5. enabled / required 规则

### 普通参数

```text
has_enable = true
enabled = item.enabled
required = false
```

UI 显示启用 checkbox。未启用时输入框仍显示 value/default，但 disabled；启用后才参与 ConfigSet/RunPlan。

### 必填参数

```text
has_enable = false 或 checkbox disabled
enabled = true
required = true
```

不允许取消，保存时后端强制 enabled=true。

### 系统自动字段

```text
readonly = true
has_enable = false
visibility = readonly/internal
```

普通用户不可编辑。

## 6. Docker options 拆分规则

`launcher.docker_options` 不能作为普通 JSON textarea。投影时拆为：

```text
launcher.docker_options.shm_size
launcher.docker_options.privileged
launcher.docker_options.ipc_mode
launcher.docker_options.uts_mode
launcher.docker_options.group_add
launcher.docker_options.devices
launcher.docker_options.optional_devices
launcher.docker_options.security_options
launcher.docker_options.ulimits.memlock
```

回写时合并回内部 `launcher.docker_options.value`。
