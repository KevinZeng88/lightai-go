# Runtime 配置展示与 Probe Evidence 修复边界与验收标准

## 1. 修复目标

本次只解决当前实际测试暴露的问题，并做有限同类排查。优先级固定为：

1. 修复已发现问题。
2. 检查同类页面、同类字段是否还有相同问题。
3. 补最小必要测试，防止本次问题回归。

当前发现的问题：

- P0-1：运行模板详情页参数不显示。
- P0-2：复制为用户配置后，`display_name` / `name` / version 展示语义错误。
- P0-3：节点运行配置探测 JSON 默认展示异常，raw Docker image metadata 直接暴露。

## 2. 范围边界

### 2.1 本次范围内

- BackendRuntime / 运行模板详情。
- BackendRuntime / 用户配置 clone。
- NodeBackendRuntime / 节点运行配置详情。
- `probe_results_json` / probe evidence 展示。
- RunPlan env 污染检查。
- `display_name` / `name` 展示语义。
- 版本适配字段展示。
- 最小必要测试。

### 2.2 本次不扩大处理

- 不重做 Runtime / Backend / NBR 整体架构。
- 不引入复杂版本管理体系。
- 不实现完整 `NVIDIA_REQUIRE_CUDA` 兼容性解析。
- 不新增多版本 catalog 设计。
- 不做大范围 UI 重构。
- 不做历史兼容迁移。

如果字段或 schema 已经污染，允许清理数据库并重建；不需要兼容旧脏数据。

## 3. P0-1：运行模板详情页参数不显示

### 3.1 现象

进入“运行模板”，点击“详情”，参数区域为空或不显示。

### 3.2 代码事实

Codex 静态核查后确认，当前主链路不是旧字段 `parameter_schema_json` / `parameter_values_json` / `resource_controls_json`，而是：

```text
config_set_json
→ /api/v1/config-edit/view
→ response envelope { config_edit_view, config_view }
→ web/src/api/configEdit.ts getConfigEditView()
→ ConfigEditView.vue localView.sections
```

真实根因：

- 后端 `/api/v1/config-edit/view` 返回 `{ config_edit_view, config_view }`。
- 前端 `getConfigEditView()` 直接返回整个响应对象，并声明为 `ConfigEditView`。
- `ConfigEditView.vue` 读取 `localView.sections`，但实际拿到外层对象，没有 `sections`，所以参数区为空。

涉及文件：

- `internal/server/api/config_edit_handlers.go`
- `web/src/api/configEdit.ts`
- `web/src/components/config/ConfigEditView.vue`
- `web/src/pages/BackendRuntimesPage.vue`

### 3.3 修复要求

首要修复：

- `web/src/api/configEdit.ts` 兼容后端 envelope，返回 `resp.config_edit_view ?? resp`。
- 保证 `ConfigEditView.vue` 接收到的对象包含 `sections`。
- 验证所有调用 `getConfigEditView()` 的页面都能正常显示 sections。

辅助检查：

- 运行模板详情页虽然使用列表行 `row.raw`，但参数编辑实际依赖 `/config-edit/view` 二次接口；不要把“不 fetch detail”误判为主因。
- 旧字段 `parameter_schema_json` / `parameter_values_json` / `resource_controls_json` 只作为历史兼容或残留风险检查，不作为本次主修复目标。

### 3.4 验收标准

- vLLM / SGLang / llama.cpp 运行模板详情页参数不为空。
- 使用 `ConfigEditView` 的运行模板详情、节点运行配置详情、节点配置向导等页面不再因 envelope 未解包导致 sections 为空。
- 复制后的用户配置详情页参数不为空。
- 刷新页面后参数仍显示。
- 无 RuntimeParameterEditor / ConfigEditView watch-emit 循环或 OOM。

## 4. P0-2：复制用户配置后的 display_name / name / version 展示错误

### 4.1 现象

复制运行模板为用户配置后显示类似：

```text
runtime.vllm.nvidia-docker - 用户配置
```

页面主列只有“名称”，实际主显示字段像是技术 `name`。复制后的用户配置还显示 `v0.23.0` 等具体 version，造成“模板已按具体软件版本差异化”的误解。

### 4.2 代码事实

Codex 核查确认根因分四段：

1. 系统 runtime YAML 中 `runtime.vllm.nvidia-docker` 等模板缺少用户可见 `display_name` / `name`，catalog loader 的 `displayRuntimeName()` 回退到 runtime ID，导致 DB `display_name` 可能是技术 ID。
2. clone 弹窗默认值直接用 `row.display_name || row.name` 拼接“用户配置”，绕过 display adapter。
3. clone 后端在没有显式 `name` 时用 `sourceName + "-copy"` 生成技术 `name`，`sourceName` 也可能来自 display_name/name，继续带技术名。
4. 前端 `extractVersion()` 只对内置模板返回 `*`；用户配置会从 `backend_version_id` 提取或返回版本 ID，所以 clone 后用户配置会显示 `v0.23.0` / `sglang-v0.5.13.post1` / `llamacpp-b9700`。

涉及文件：

- `configs/backend-catalog/runtimes/vllm/nvidia-docker.yaml`
- `configs/backend-catalog/runtimes/sglang/nvidia-docker.yaml`
- `configs/backend-catalog/runtimes/llamacpp/nvidia-docker.yaml`
- `internal/server/catalog/loader.go`
- `internal/server/api/node_runtime_handlers.go`
- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/utils/runtimeDisplay.ts`
- `web/src/locales/zh-CN.ts`
- `web/src/locales/en-US.ts`

### 4.3 修复要求

#### 4.3.1 用户主显示字段

- 列表主列改为“显示名称”。
- 用户主显示字段使用 `display_name` 或 display adapter 后的 displayName。
- 技术 `name` 只放详情或辅助字段。

复制规则：

```text
新 display_name = 源用户可见 displayName + " - 用户配置"
新 name = 稳定唯一技术标识，不从用户显示名派生
```

示例：

```text
源 display_name：vLLM NVIDIA Docker
复制后 display_name：vLLM NVIDIA Docker - 用户配置
技术 name：runtime.vllm.nvidia-docker.user.<shortid>
```

#### 4.3.2 Catalog 显示名

至少给以下系统 runtime catalog 补用户可见 `display_name`：

- vLLM NVIDIA Docker
- SGLang NVIDIA Docker
- llama.cpp NVIDIA Docker

#### 4.3.3 Version 展示

当前 catalog 虽然有具体 `BackendVersion` 文件和 `backend_version_id`，但运行模板没有按具体软件版本形成不同参数 schema、启动命令、健康检查、capability profile 或 runtime requirements。

因此用户主 UI 不应显示具体版本号。当前规则：

- 运行模板列表 / 用户配置列表：隐藏版本列，或统一显示“适配版本：*”。
- 运行模板详情 / 用户配置详情：如需显示，统一显示“适配版本：*”。
- 具体 `backend_version_id` 仅作为技术信息或诊断信息展示，不作为用户主字段。
- 内置模板和复制后的用户配置版本展示规则一致。

### 4.4 验收标准

- 复制后列表显示类似 `vLLM NVIDIA Docker - 用户配置`。
- 列名为“显示名称”。
- 技术 `name` 不作为用户主标题。
- 详情页可以看到技术标识 `name`。
- 列表不显示 `v0.23.0`、`sglang-v0.5.13.post1`、`llamacpp-b9700` 等具体 backend version。
- 源模板和复制配置的适配版本一致，均为 `*` 或列表隐藏版本字段。
- 不出现 `runtime.vllm.nvidia-docker` 作为用户标题。
- 不出现 i18n key 泄露。

## 5. P0-3：节点运行配置探测 JSON 默认展示异常

### 5.1 现象

节点运行配置详情默认展示完整 probe JSON，其中包含 Docker image inspect 的原始 env：

```text
NVIDIA_REQUIRE_CUDA=cuda>=13.0 brand=unknown,driver>=535,driver<536 ...
PATH=...
LD_LIBRARY_PATH=...
CUDA_VERSION=13.0.2
```

这类内容很像异常配置，用户无法判断含义。

### 5.2 代码事实

Codex 核查确认：

- 当前没有看到 `.Config.Env` 写入 `config_set_json`、runtime env、Deployment snapshot env 或 `ResolvedRunPlan.env` 的代码路径。
- `.Config.Env` 被保存到 `probe_results_json.level2.env`，属于 raw evidence。
- RunPlan env 来源是 NBR/Deployment 的 ConfigSet `runtime.env`、参数 `target=env`、部署 env overrides 和 GPU visible env，不读取 `probe_results_json.level2.env`。
- 当前真实问题是 RunnerConfigs 详情默认直接展示 raw `probe_results_json`，用户会看到完整 `NVIDIA_REQUIRE_CUDA`、`PATH`、`LD_LIBRARY_PATH` 等 image metadata。
- `level4` 仍有开发口径字符串：`version probe not yet implemented; deferred to future design`。

涉及文件：

- `internal/server/api/runtime_handlers.go`
- `web/src/pages/RunnerConfigsPage.vue`
- `internal/server/api/deployment_lifecycle_handlers.go`
- `internal/server/runplan/resolver.go`

### 5.3 修复要求

#### 5.3.1 数据边界保持

继续保持三类 env 分离：

```text
image config env：Docker image inspect 返回的镜像元数据，只能作为 raw evidence
configured env：用户或系统显式配置的运行环境变量
resolved runplan env：最终启动容器时 LightAI Go 主动注入的环境变量
```

补测试确认 Docker image inspect `.Config.Env` 不进入：

- BackendRuntime / NBR `config_set_json` 的 runtime env。
- Deployment snapshot env。
- `ResolvedRunPlan.env`。
- RuntimeParameterEditor / ConfigEditView 可编辑参数。

#### 5.3.2 UI 默认展示摘要

节点运行配置详情默认展示摘要，不直接展示完整 raw JSON。

摘要至少包括：

- 镜像状态。
- 镜像引用。
- 镜像 ID / digest，截断显示。
- CUDA 版本。
- 是否存在 NVIDIA CUDA 约束。
- 后端匹配状态。
- 匹配依据。
- 启动方式。
- 启动 profile。
- 置信度。
- 兼容性检查状态。
- 是否阻断部署。

#### 5.3.3 Raw JSON 默认收起

完整 probe evidence 放到“诊断原文 / Raw probe evidence”折叠区，默认收起。

默认页面不得出现：

- 完整 `NVIDIA_REQUIRE_CUDA` 原文。
- `PATH=/usr/local/nvidia`。
- `LD_LIBRARY_PATH`。
- 完整 Docker `Config.Env`。
- `deferred to future design`。
- `not yet implemented`。

#### 5.3.4 level4 产品口径

把开发口径改成结构化产品口径，例如：

```json
{
  "compatibility_check_status": "not_run",
  "version_probe_status": "not_available",
  "blocking": false,
  "message": "当前仅完成镜像存在性与后端匹配检查。"
}
```

### 5.4 验收标准

- 节点运行配置详情默认显示摘要。
- Raw JSON 默认收起。
- 默认页面不显示完整 `NVIDIA_REQUIRE_CUDA`。
- 默认页面不显示 `PATH` / `LD_LIBRARY_PATH` 等 image `Config.Env`。
- `env_json` / ConfigSet runtime env 不包含 Docker inspect 原始 `Config.Env`。
- `ResolvedRunPlan.env` 不包含 Docker inspect 原始 `Config.Env`。
- vLLM 最终启动 env 只包含明确需要注入的变量。
- 页面不出现 `deferred to future design` / `not yet implemented`。

## 6. 同类问题有限检查

修完三个问题后，只检查下面三类。

### 6.1 参数字段是否在其他页面丢失

检查：

- 运行模板详情 / 编辑。
- 用户运行配置详情 / 编辑。
- 节点运行配置详情 / 编辑。
- 节点配置向导。
- 部署 RunPlan 预览。

重点检查 `ConfigEditView` sections 是否正常。

### 6.2 display_name / name 是否混用

检查：

- 运行模板列表 / 详情。
- 用户运行配置列表 / 详情。
- 节点运行配置列表 / 详情。
- 模型部署列表。
- 模型实例列表。

规则：用户主标题用 `display_name`；技术 `name` / `id` 放详情或辅助字段。

Codex 发现部署列表 / 详情仍可能直接显示 `source_node_backend_runtime_id`。如果已有 display_name 可用，应做最小 UI 修复；如果需要较大 DTO / 查询改造，先在执行报告中明确记录涉及链路和建议修复方案。

### 6.3 raw evidence 是否默认直出

检查：

- 节点运行配置详情。
- 镜像检测结果。
- preflight 结果。
- 部署诊断。
- 模型实例诊断。

规则：默认展示摘要，Raw JSON 默认收起，开发口径不进入用户页面。

## 7. 最小必要测试

### 7.1 Go / API 测试

- clone 后 `display_name` 是用户可见名，`name` 是稳定唯一技术名。
- version 展示语义不依赖用户配置的具体 `backend_version_id`。
- probe `.Config.Env` 只在 `probe_results_json.level2.env`，不进入 ConfigSet runtime env。
- `ResolvedRunPlan.env` 不包含 `NVIDIA_REQUIRE_CUDA`、`PATH`、`LD_LIBRARY_PATH`，除非用户显式配置。

### 7.2 Frontend 测试

- `getConfigEditView()` unwraps `config_edit_view`。
- 运行模板详情显示 `ConfigEditView.sections`。
- clone 默认显示名不含 `runtime.vllm.nvidia-docker`。
- 用户配置列表 / 详情不显示 `v0.23.0`，显示 `*` 或隐藏版本列。
- RunnerConfigs 详情默认不出现 `NVIDIA_REQUIRE_CUDA`、`PATH`、`LD_LIBRARY_PATH`。
- Raw JSON 默认收起。

### 7.3 推荐测试命令

```bash
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm test
cd web && npm run build
```

如果已有相关 API-first E2E，可补跑 runtime / NBR / RunPlan 相关脚本。
