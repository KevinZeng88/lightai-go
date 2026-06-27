# Final Parameter Contract

## 1. 目标

本文定义参数体系最终契约。执行端必须据此修复数据结构、API、UI、RunPlan、Preflight、E2E。

关键原则：

1. 参数单一属主；
2. 参数单一定义；
3. 分层 copy-on-create；
4. 每层只叠加自己拥有的数据或 override；
5. 每个页面只展示自己拥有或允许覆盖的内容；
6. 只有 ResolvedRunPlan 阶段合成全部参数；
7. RunPlan preview 显示最终值和来源。

## 2. 参数对象

### 2.1 ParameterDefinition

ParameterDefinition 是 schema。它定义参数是什么、如何展示、如何校验、如何进入最终运行 spec。

字段应包含或等价表达：

```json
{
  "owner_type": "backend_version",
  "owner_key": "vllm.openai.latest",
  "key": "gpu_memory_utilization",
  "label": "GPU Memory Utilization",
  "description": "Fraction of GPU memory reserved for vLLM.",
  "category": "resource_controls",
  "scope": "runtime",
  "target": "args",
  "type": "number",
  "required": false,
  "default_value": 0.9,
  "default_enabled": false,
  "editable_at": ["backend_runtime", "node_backend_runtime", "deployment"],
  "visibility": "basic",
  "advanced": false,
  "order": 100,
  "constraints": {"min": 0.1, "max": 1.0},
  "arg_name": "--gpu-memory-utilization",
  "help_text": "Use lower value when multiple workloads share a GPU."
}
```

同一个参数只能由一个 owner 定义。唯一性应由 `owner_type + owner_key + key` 或稳定 definition id 保证。

### 2.2 ParameterValue

ParameterValue 是某层保存的值。它可以是默认值、继承值、当前层显式值或系统生成值。

ParameterValue 不能重新定义 schema。

### 2.3 ParameterOverride

ParameterOverride 表示当前层级覆盖某个已有定义。

必须保存：

1. definition reference；
2. override owner；
3. enabled；
4. value；
5. source；
6. timestamps；
7. optional reason。

Override 不能保存 schema 副本。

### 2.4 ResolvedParameter

ResolvedParameter 是 RunPlan 合成后的最终参数。

必须包含：

1. key；
2. definition reference；
3. final value；
4. enabled/effective；
5. source；
6. source chain；
7. target；
8. rendered output；
9. validation result。

## 3. owner 边界

### 3.1 Model / ModelArtifact owner

拥有模型自身参数和 metadata：

1. model family；
2. model format；
3. quantization；
4. context capability；
5. modality；
6. embedding/chat/rerank capability；
7. tokenizer / architecture metadata；
8. required model files。

不拥有 Docker image、Docker args、GPU runtime、容器端口。

### 3.2 Backend / BackendVersion owner

拥有后端能力和后端版本参数定义：

1. supported model formats；
2. endpoint capability；
3. backend-specific args schema；
4. resource control parameter schema；
5. health check capability schema；
6. protocol capability。

不拥有节点运行状态、部署 override、本机模型路径。

### 3.3 BackendRuntime owner

拥有运行模板自己的配置：

1. image default；
2. command default；
3. template args/env/mounts/ports default；
4. template-level override；
5. default health check；
6. template resource control defaults。

它可以 override BackendVersion 定义的参数，但不能重定义 BackendVersion 的 schema。

### 3.4 NodeBackendRuntime owner

拥有节点运行环境配置：

1. node-specific image evidence；
2. Docker runtime；
3. device binding selection；
4. node env；
5. node path/mount mapping；
6. check-request evidence；
7. node-level override。

它可以 override 可在 NBR 层编辑的参数，但不能重定义 schema。

### 3.5 Deployment owner

拥有部署意图和部署级 override：

1. resource override；
2. port override；
3. mount override；
4. health check override；
5. runtime parameter override；
6. desired state。

Deployment 可以调整最终运行相关参数，但保存的是 override。Deployment 不能创建同名 schema。

### 3.6 ResolvedRunPlan owner

拥有最终合成结果。它不定义业务 schema；它记录 final value、source、rendered Docker spec。

### 3.7 Instance owner

拥有运行事实。它不编辑参数。

## 4. copy-on-create 契约

每一层创建时执行 copy-on-create：

```text
上层有效视图 + 当前层 owner 数据 + 当前层 override = 当前层有效视图
```

层级链：

```text
BackendVersion / ModelArtifact
→ BackendRuntime
→ NodeBackendRuntime
→ Deployment
→ ResolvedRunPlan
→ Instance
```

规则：

1. BackendRuntime 创建时拷贝 BackendVersion 当时有效视图；
2. NodeBackendRuntime 创建时拷贝 BackendRuntime 当时有效视图；
3. Deployment 创建时拷贝 NodeBackendRuntime 当时有效视图；
4. ResolvedRunPlan 由 Deployment snapshot + 当前部署输入解析生成；
5. Instance 记录 ResolvedRunPlan 执行后的事实；
6. 上层后续修改不影响已有下层；
7. 下层后续修改不影响上层；
8. clone 保留 owner/key/value/enabled/source，不扩大 checked；
9. copy-on-create 保存快照，不改变 schema owner。

## 5. enabled / checked / default / required 语义

### 5.1 enabled

`enabled=true` 表示用户在当前层级显式启用或覆盖该参数。

它不表示：

1. 参数存在默认值；
2. 参数是 required；
3. 参数最终一定由用户手工设置；
4. 参数 schema 可编辑。

### 5.2 default_value

`default_value` 表示 schema 提供的默认值。default value 可以在最终 RunPlan 生效，但不导致 UI checked。

### 5.3 required

`required=true` 表示该参数必须在最终运行时有有效值。required 参数可以由 default、继承值、系统生成值或 override 满足。

required 参数不显示成用户 checked，除非用户在当前层级显式覆盖。

### 5.4 optional

optional 参数默认不 checked。未 enabled 的 optional 参数不进入当前层级 override。它是否进入最终 RunPlan 由 schema default、backend requirement 和 resolver 规则决定。

### 5.5 advanced

advanced 参数默认折叠、不 checked。高级参数保留展示和编辑能力，但不干扰普通配置流程。

### 5.6 inherited

inherited 表示当前层从上层 snapshot 继承的值。UI 应显示来源，不能显示成当前层 checked。

### 5.7 override

override 表示当前层显式覆盖。UI 应显示当前层 checked/enabled，并记录 source。

## 6. 参数分类

参数至少支持以下 category：

1. `model_loading`：模型加载参数；
2. `service_api`：服务/API 参数；
3. `resource_controls`：资源控制参数；
4. `accelerator`：GPU / accelerator 参数；
5. `network_ports`：网络 / 端口参数；
6. `mounts_paths`：卷挂载 / 文件路径参数；
7. `health_check`：健康检查参数；
8. `container_runtime`：Docker / 容器参数；
9. `advanced`：高级参数。

UI 展示必须按 category 分组。basic 参数优先展示，advanced 默认折叠。

## 7. 页面展示边界

### 7.1 Model 页面

展示：

1. 模型 metadata；
2. 模型格式；
3. 模型能力；
4. 上下文能力；
5. 量化信息；
6. 模型文件信息；
7. ModelLocation。

不展示 Docker args、Docker env、容器镜像、GPU runtime、节点运行环境参数。

### 7.2 Backend / BackendVersion 页面

展示后端能力、版本能力、能力参数定义。避免展示节点状态和部署 override。

### 7.3 BackendRuntime 页面

展示运行模板参数、模板默认值、模板 override、镜像、命令、模板健康检查。

### 7.4 NodeBackendRuntime 页面

展示节点运行环境配置、节点 override、设备绑定、image check、check-request evidence。

### 7.5 Deployment 页面

展示可覆盖参数、部署 override、最终有效参数预览、RunPlan preview、端口、卷、健康检查。

### 7.6 Instance 页面

展示运行事实、状态、日志、健康检查、实际 Docker spec 摘要。不编辑参数。

## 8. 参数进入最终运行 spec 的规则

参数进入 Docker spec 必须经 RunPlan resolver。

target 可以是：

1. args；
2. env；
3. mounts；
4. ports；
5. devices；
6. health_check；
7. labels；
8. internal_only。

resolver 规则：

1. required/default-applied 参数按 schema 规则生效；
2. enabled override 覆盖继承值；
3. 未 enabled 的 optional override 不进入当前层；
4. 参数渲染由 schema target 决定；
5. args 去重；
6. env 不混入 capabilities_json；
7. ports 与 health check 保持一致；
8. 每个 rendered output 带 source。

## 9. vLLM 参数契约

至少覆盖：

1. `--model`：通常由 ModelLocation / resolver 生成；
2. `--host`：服务/API 参数，可由模板默认，部署覆盖；
3. `--port`：网络参数，可由模板默认，部署覆盖；
4. `--gpu-memory-utilization`：资源控制参数；
5. `--max-model-len`：模型加载/资源参数；
6. dtype；
7. quantization；
8. tensor parallel；
9. served model name；
10. OpenAI compatible endpoint。

每个参数必须明确 owner、可编辑层级、target、default、required、source。

## 10. SGLang 参数契约

至少覆盖：

1. `--model-path`；
2. `--host`；
3. `--port`；
4. `--mem-fraction-static`；
5. `--context-length`；
6. dtype；
7. tensor parallel；
8. OpenAI compatible endpoint。

## 11. llama.cpp 参数契约

至少覆盖：

1. `--model` / `-m`；
2. `--host`；
3. `--port`；
4. `--ctx-size`；
5. `--n-gpu-layers` / `-ngl`；
6. batch；
7. ubatch；
8. OpenAI compatible endpoint。

## 12. Anti-patterns

必须消除：

1. 所有参数默认 checked；
2. 有默认值的参数全部 checked；
3. required 显示成用户 checked；
4. 每个页面都展示全部参数；
5. 多个层级重复定义同一个参数；
6. Deployment 重新定义运行参数 schema；
7. UI 为了展示复制 schema；
8. optional 参数默认进入 args；
9. 高级参数默认展开；
10. Model 页面展示 Docker 参数；
11. Backend 页面展示节点运行环境参数；
12. NodeBackendRuntime 页面展示模型 metadata 编辑项；
13. Instance 页面允许编辑运行参数；
14. RunPlan 把未 enabled 的 optional 参数写入最终 args；
15. UI checked 状态与最终 RunPlan 生效状态混淆；
16. 参数刷新后 schema/value/enabled/source 丢失；
17. clone 后 checked 状态被错误扩大。
