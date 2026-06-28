# LightAI Go 模型部署页面 MHTML 回归复核

来源：`LightAI Go1.mhtml`，页面 URL：`/models/deployments`，快照时间约 2026-06-28 19:08 +0800。

## 一、MHTML 可确认的问题

### 1. 部署向导状态停留在中间步骤

页面显示：

- `创建部署`
- 步骤：`选择模型`、`选择运行配置`、`服务配置`、`参数覆盖`、`预览运行计划`
- 当前可见内容位于 `参数覆盖 / 服务配置` 之后，底部只有 `上一步 / 下一步 / 取消`

这与用户观察一致：新建/取消/保存后，下次再点新建可能复用上一次 wizard 状态。  
MHTML 只能确认当前快照处在中间步骤，无法单独证明保存/取消后的状态复用根因。需要检查 `ModelDeploymentsPage.vue` 里的 drawer/form state 初始化、关闭、取消、保存成功后的 reset 逻辑。

建议规则：

- 点击“新建”必须创建全新的 wizard draft state。
- 保存成功后关闭并 reset draft。
- 取消后关闭并 reset draft。
- drawer 关闭事件也必须 reset draft。
- 不得复用上一次保存/取消时的 step、selected model、selected runtime、config_overrides。

### 2. 参数覆盖页暴露了不适合普通用户的 model_runtime.* 高级参数

MHTML 可见：

- `Model runtime cpu offload gb`
- `Model runtime download dir`
- `Model runtime host`
- `Model runtime kv cache dtype`
- `Model runtime max num batched tokens`
- `Model runtime max num seqs`
- `Model runtime model`
- `Model runtime pipeline parallel size`
- `Model runtime port`
- `Model runtime safetensors load strategy`
- `Model runtime swap space`

其中至少 `model_runtime.model`、`model_runtime.port`、`model_runtime.host` 不应作为普通部署覆盖项直接暴露：
- model 路径应来自模型选择 / model mount / artifact location。
- port 应以 `service.container_port` 为 canonical 字段。
- host 多数情况下属于容器内部服务绑定，默认即可，不应 required+readonly+empty。

`cpu_offload_gb`、`kv_cache_dtype`、`max_num_batched_tokens`、`max_num_seqs`、`swap_space` 等是 vLLM 高级调优参数，是否保留应取决于 backend 参数 catalog：
- 常用、稳定、用户理解成本低的参数可以放“高级参数”。
- 不常用/实验/后端版本强相关的参数建议归入 `custom args / extra args`，或隐藏到“专家参数”默认收起。
- 不应显示 required 但不可填写的字段。

### 3. `Model runtime port` 仍显示 required，但没有用户可编辑值

MHTML 可见：

- `Model runtime port`
- `required`
- 同页又有 `Container listen port`
- `容器端口:`
- `宿主机端口:`

这说明端口语义仍然重复：

- `service.container_port` 应作为用户可见 canonical 容器端口。
- `model_runtime.port` 如为内部 CLI 参数映射，应从用户普通表单隐藏，或由 `service.container_port` 派生。
- 不允许显示 `required + readonly/empty`。

### 4. Container listen port 显示容器端口 8000，但宿主机端口为空

MHTML 可见服务入口区域：

- `Container listen host`
- `Container listen port`
- `容器端口:`
- `宿主机端口:`

用户补充说容器端口为 8000、宿主机端口为空。宿主机端口为空不一定错误，取决于当前网络模式：

- 若 `network_mode=host`，宿主机端口与容器端口共享网络命名空间，host port 可以隐藏或显示“不适用（host network）”。
- 若 bridge/default network，host port 为空意味着不能形成明确端口映射，应该提示“自动分配/未配置/不可访问”，不能静默空白。
- RunPlan preview 必须能清楚说明最终端口映射。

### 5. 高级原始配置仍然可见，并列出 raw JSON/array 字段

MHTML 可见 `高级原始配置` 下列字段：

- `Backend extra args`
- `Command ["--model","{{model_container_path}}"]`
- `Devices []`
- `Entrypoint ["vllm","serve"]`
- `Extra env`
- `Kind`
- `Ports []`
- `Served model name`
- `Volumes []`

这些不应作为普通部署配置页默认可见内容。建议：

- 普通页展示结构化摘要和可编辑字段。
- Raw config / source metadata / command / entrypoint / raw arrays 放到诊断区，默认收起。
- 用户需要修改不常用 CLI 参数时，通过 `Custom args / Extra args` 明确添加。

## 二、用户手工观察但 MHTML 不足以完全证明的问题

### 1. 模型部署失败：unsupported runtime_type

用户观察：

`[resolve_error] unsupported runtime_type: (only docker is supported)`

MHTML 快照里没有该字符串，但问题严重，必须继续查：

- 是否旧 deployment snapshot / old DB 残留 runtime_type 空值。
- 是否某些部署创建路径没有写入 runtime_type。
- 是否保存 draft / cancel draft 后复用旧 config_overrides，导致 runtime_type 被空值覆盖。
- 是否 RunPlan preview、dry-run、start 使用的 deployment snapshot 不是当前 NBR snapshot。

验收必须包括：
- 新建 vLLM Docker 部署 preflight/RunPlan preview 显示 runtime_type=docker。
- 旧失败 deployment 重建后不再出现 unsupported runtime_type。
- 保存/取消后的 wizard reset 不会复用旧错误状态。

### 2. 模型部署没有修改选项，打开只有原始配置 JSON、来源元数据 JSON

MHTML 只显示当前新建 drawer，未显示打开已有部署详情。用户观察应纳入修复：

- 已有部署详情应有“查看 / 编辑 / RunPlan 预览 / Dry Run / 启动 / 停止”等明确操作。
- 默认不展示 raw config JSON / source metadata JSON。
- raw JSON 只放诊断区，默认收起。
- 编辑部署应进入结构化 ConfigEditView，而不是只给 raw JSON。

## 三、建议修复边界

本轮应合并两类问题：

### A. Runtime template / device-mount 语义

- 只保留 `Devices`，不引入 Optional devices。
- Devices = Docker `--device` 设备透传列表。
- NVIDIA 默认 devices disabled/empty。
- MetaX 默认 devices enabled，包含 `/dev/mxcd`、`/dev/dri`、`/dev/mem`。
- Devices 缺失只 warning，不阻断部署。
- Devices UI 不出现 readonly。
- Model mount 默认 readonly；Additional volumes 与 Model mount 分离。

### B. Model deployment wizard/detail 回归

- 新建/保存/取消/关闭必须 reset wizard state。
- 已有部署详情应有编辑入口，不只 raw JSON。
- Raw config/source metadata 默认收起。
- 修复 runtime_type 仍可能为空的路径或旧状态污染。
- 端口字段以 `service.container_port` 为 canonical。
- `model_runtime.port/host/model` 不作为普通 required 字段展示。
- vLLM 高级参数分层：常用参数保留，高风险/不常用/版本相关参数归入 Custom args 或专家区默认收起。

## 四、建议测试

- `ModelDeploymentsPage.integration.test.ts`
  - 新建后总是从 step 1/空 draft 开始。
  - 保存成功 reset draft。
  - 取消 reset draft。
  - 打开已有 deployment 默认结构化详情，不直接展开 raw JSON。
  - 有编辑入口。
  - `service.container_port` 显示 canonical 端口。
  - 不显示 `model_runtime.port required + empty`。
  - 默认不显示 raw config/source metadata JSON。

- RunPlan/API test
  - 新建 vLLM Docker deployment preflight resolves `runtime_type=docker`。
  - 旧/空 config_overrides 不得覆盖 runtime_type。
  - Deployment snapshot retains runtime_type from NBR/runtime template.

- Config taxonomy test
  - `model_runtime.port/host/model` 不在普通用户部署覆盖字段中。
  - 高级参数默认收起；不常用参数进入 custom/extra args 或专家区。
