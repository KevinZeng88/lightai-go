# LightAI Go 运行模板 UX 与 ConfigEditView 展示修复计划（给 Claude 执行）

日期：2026-06-26  
仓库：`/home/kzeng/projects/ai-platform-study/lightai-go`  
目标分支：当前分支 / main  
相关阶段：`docs/reports/phase-3/runtime-template-catalog-redesign/`

---

## 0. 背景与总目标

前面已经完成了：

1. Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / Deployment 的 `config_set_json` 快照化。
2. RunPlan snapshot-only。
3. `ConfigEditView / ConfigEditPatch` 抽象。
4. 运行模板、节点运行配置、部署 override 开始接入 `ConfigEditView`。

但手工验证运行模板页面时仍发现明显产品化问题：

- 复制为用户配置的弹窗仍有英文硬编码。
- 输入显示名称或技术名称后，列表仍显示旧名称或技术组合名。
- 复制出来的用户配置没有明显修改、删除操作。
- 系统模板详情的高级诊断仍直接展示 raw JSON。
- 用户配置详情仍显示不可理解字段：`Backend capabilities`、`Backend supported config items`、大量 `backend.*` 字段、`{{container_port}}`、`[object Object]`、整块 Model mount JSON。
- 同类问题可能出现在运行模板页、节点运行配置向导、部署 override、BackendVersion 编辑等所有使用 `ConfigEditView` 的位置。

本轮目标不是重做底层架构，而是把 **ConfigEditView 的展示、投影、组件、i18n、操作流程** 补完整。

最终应达到：

```text
系统内部：
  config_set_json / launcher.xxx / runtime.xxx / backend.arg.xxx / source_metadata_json

用户界面：
  运行模板名称、来源、镜像、模型服务参数、容器资源、设备与挂载、环境变量、服务入口、健康检查、只读诊断摘要

禁止：
  普通区域直接出现 launcher.xxx / runtime.xxx / source_metadata / [object Object] / 大段 JSON
```

---

## 1. 修复复制为用户配置的命名与 i18n

### 1.1 当前问题

运行模板页点击“复制为用户配置”后，弹窗仍显示英文：

```text
Clone runtime
Display Name
Name
```

输入 `display_name` 或 `name` 后，确认复制，列表里仍可能显示旧名称或技术组合名。

### 1.2 根因

`web/src/pages/BackendRuntimesPage.vue` 里 clone dialog 仍有英文硬编码。  
`web/src/utils/runtimeDisplay.ts` 当前使用 `${vendor}.${backendId}` 作为 `displayName`，没有优先使用 `row.display_name`。  
复制完成后前端只 `await load()`，没有自动选中新复制对象。

### 1.3 修改方法

#### 前端：`web/src/utils/runtimeDisplay.ts`

重写 `toRuntimeTemplateDisplay(row)`：

```ts
displayName 优先级：
1. row.display_name 非空
2. 产品化名称：`${backendDisplay} / ${vendorDisplay}`
3. row.name
4. row.id
```

增加字段：

```ts
rawName
rawId
sourceType: 'builtin' | 'user'
sourceLabel
backendDisplay
vendorDisplay
versionDisplay
```

映射建议：

```ts
backend.vllm       -> vLLM
backend.sglang     -> SGLang
backend.llamacpp   -> llama.cpp

nvidia             -> NVIDIA
metax              -> MetaX
huawei / ascend    -> Huawei Ascend
cpu                -> CPU
```

内置通用运行模板的 Backend Version 显示 `*`。

#### 前端：`web/src/pages/BackendRuntimesPage.vue`

1. clone dialog 全部接入 i18n。
2. clone 成功后使用后端返回对象：
   - 如果 response 有 `id`，`await load()` 后查找该 id。
   - 自动 `selected.value = newRow`。
   - 打开详情 drawer。
3. 不再把用户输入的 `display_name` 覆盖为 `${vendor}.${backendId}`。

#### 后端：`internal/server/api/runtime_handlers.go`

确认 `HandleCloneBackendRuntime`：

1. 接收 `display_name` 和 `name`。
2. 如果 `name` 为空，自动生成唯一技术名称。
3. 如果 `name` 冲突，自动唯一化或返回明确 409。
4. 返回完整 `publicBackendRuntimeJSON(newRuntime)`，包含新 id/name/display_name/is_editable/config_set/source_metadata。

### 1.4 验收

1. 点击复制弹窗无英文硬编码。
2. 输入“我的 vLLM MetaX 配置”，复制后列表显示这个名称。
3. 复制后自动打开这个用户配置详情。
4. 技术 name 为空时后端自动生成唯一名称。
5. 技术 name 冲突时行为明确，有测试覆盖。

---

## 2. 给用户配置增加编辑、删除、重命名操作

### 2.1 当前问题

复制出来的用户配置没有明显“修改 / 删除”按钮。

### 2.2 根因

`BackendRuntimesPage.vue` 的操作列当前只对 system 模板显示 clone；对 user config 没有动作。

### 2.3 修改方法

#### 前端：运行模板表格操作列

按类型显示：

```text
system builtin:
  - 复制为用户配置

user config / is_editable=true:
  - 打开 / 编辑
  - 重命名
  - 删除
  - 复制
```

可以先用以下按钮组合：

```vue
<el-button @click.stop="openRuntime(row.raw)">详情</el-button>
<el-button v-if="row.managedBy === 'system'" @click.stop="cloneRuntime(row.raw)">复制为用户配置</el-button>
<el-button v-if="row.managedBy === 'user'" @click.stop="renameRuntime(row.raw)">重命名</el-button>
<el-button v-if="row.managedBy === 'user'" type="danger" @click.stop="deleteRuntime(row.raw)">删除</el-button>
```

#### 重命名

重命名可复用详情页基础信息，也可以独立 dialog。最低要求：

- 可修改 `display_name`。
- 可选修改 `name`。
- 调用 `PATCH /backend-runtimes/{id}`。
- 保存后刷新并保持选中。

#### 删除

使用现有：

```http
DELETE /api/v1/backend-runtimes/{id}
```

后端已有 system runtime 删除拒绝逻辑。需要前端确认弹窗：

```text
确认删除用户配置“xxx”？删除后不会影响已经创建的 NBR / Deployment 快照。
```

### 2.4 验收

1. 系统模板只显示“复制为用户配置”。
2. 用户配置显示“详情/编辑、重命名、删除、复制”。
3. 删除用户配置成功。
4. 删除系统模板被拒绝，前端不显示删除按钮。
5. 删除用户配置不影响既有 NBR/Deployment snapshot。

---

## 3. 改造运行模板详情：普通只读参数 + 诊断摘要 + 原始 JSON 折叠

### 3.1 当前问题

系统模板详情中“高级诊断”直接显示：

```text
技术配置: selected.config_set JSON
Source Metadata: selected.source_metadata JSON
```

用户无法理解这些 JSON。

### 3.2 这两个 JSON 的实际含义

- `config_set_json`：内部运行配置快照，用于生成运行计划，包括镜像、Docker 参数、环境变量、启动参数、模型挂载、健康检查等。
- `source_metadata_json`：来源信息，表示该配置从哪个 BackendVersion / BackendRuntime / catalog 模板复制而来，是否 copy-on-create、checksum、加载路径等。

### 3.3 修改方法

#### 新增前端组件

建议新增：

```text
web/src/components/config/SourceMetadataSummary.vue
web/src/components/config/ConfigDiagnosticsSummary.vue
```

`SourceMetadataSummary` 显示：

```text
来源类型
来源后端
来源版本
来源运行模板
复制语义：copy-on-create
复制边界：detached_after_create
来源 checksum
loaded_from
loaded_at
updated_at
```

没有值则不显示，不要直接 dump JSON。

`ConfigDiagnosticsSummary` 显示：

```text
配置项数量
普通配置项数量
高级/内部配置项数量
镜像
Docker shm_size
设备数量
环境变量数量
健康检查路径
```

#### `BackendRuntimesPage.vue`

系统模板详情：

1. 仍加载 `ConfigEditView`，但只读。
2. 标题使用“配置参数（只读）”。
3. 显示来源摘要。
4. 原始 JSON 放到折叠区：

```text
开发诊断 / 原始 JSON
  - 原始配置 JSON
  - 来源元数据 JSON
```

默认折叠，中文标题，不再直接显示 `Source Metadata`。

### 3.4 验收

1. 系统模板详情默认不显示 raw JSON。
2. 可以看到结构化只读参数。
3. 可以看到来源摘要。
4. 原始 JSON 仍可在“开发诊断 / 原始 JSON”中查看。
5. 页面不出现英文 `Source Metadata`。

---

## 4. 修复 ConfigField 显示 `[object Object]`

### 4.1 当前问题

设备映射显示：

```text
[object Object]
[object Object]
[object Object]
```

### 4.2 根因

`web/src/components/config/ConfigField.vue` 对数组使用：

```ts
field.value.join('\n')
```

如果数组元素是 object，就会显示 `[object Object]`。

### 4.3 修改方法

#### 新增/增强 widget

在 `ConfigField.vue` 中支持以下 widget：

```text
device_table
mount_form
key_value_table
health_check_form
port_form
readonly_summary
```

最低实现要求：

##### device_table

用于：

```text
launcher.docker_options.devices
launcher.docker_options.optional_devices
```

支持数组对象：

```json
[
  {"host_path": "/dev/dri", "container_path": "/dev/dri"},
  {"host_path": "/dev/mxcd", "container_path": "/dev/mxcd"}
]
```

显示为表格：

```text
Host Path       Container Path     Readonly
/dev/dri        /dev/dri           false
/dev/mxcd       /dev/mxcd          false
```

编辑时允许新增/删除行。只读时只展示表格。

##### key_value_table

用于：

```text
runtime.env
launcher.docker_options.ulimits
labels
capabilities map
```

显示为：

```text
Key        Value
HF_HOME    /cache/hf
```

##### mount_form

用于：

```text
runtime.model_mount
```

字段：

```text
container_path
readonly
```

如果有 host_path/source_path 也展示。

##### health_check_form

用于：

```text
runtime.health
```

字段：

```text
path
port
timeout
interval
retries
model_probe_path
```

##### port_form

用于：

```text
ports
service
container_port
host_port
```

如果值为 `{{container_port}}`，普通 UI 应显示为只读说明：

```text
由部署服务端口决定
```

不要显示模板变量字符串给用户编辑。

### 4.4 验收

1. 页面不再出现 `[object Object]`。
2. Devices 显示为表格。
3. Model mount 不显示整块 JSON。
4. Health check 不显示整块 JSON。
5. Env 显示为键值表。
6. 模板变量 `{{container_port}}` 不作为普通可编辑字符串出现。

---

## 5. 扩展后端 ConfigEditView projection，避免 object 全部 raw_json

### 5.1 当前问题

`ProjectConfigSetToEditView()` 目前只特殊处理 `launcher.docker_options`。  
其他 object 类型默认 `widget=raw_json`，导致：

```text
Backend capabilities 内容是 JSON
Backend supported config items 内容是 JSON
Model mount 内容是 JSON
Health check 内容是 JSON
```

### 5.2 修改方法

#### 后端：`internal/server/configedit/project.go`

新增投影函数：

```go
projectEnv()
projectModelMount()
projectHealthCheck()
projectPorts()
projectCapabilitiesSummary()
```

处理规则：

```text
runtime.env
  -> environment / key_value_table

runtime.model_mount
  -> devices_mounts / mount_form

runtime.health
  -> health_check / health_check_form

launcher.docker_options.devices
  -> devices_mounts / device_table

launcher.docker_options.optional_devices
  -> devices_mounts / device_table 或 string_list

launcher.docker_options.ulimits
  -> container_resources / key_value_table

ports / service.* / deployment.service_json
  -> service / port_form

Backend capabilities / Backend supported config items / capabilities_detail / internal metadata
  -> advanced_raw / readonly_summary
```

#### 后端：`taxonomy.go`

扩展 widget/section 规则：

```text
runtime.env                      -> environment, key_value_table
runtime.model_mount              -> devices_mounts, mount_form
runtime.health                   -> health_check, health_check_form
launcher.docker_options.devices  -> devices_mounts, device_table
service.container_port           -> service, port_form / readonly placeholder
backend.capabilities             -> advanced_raw, readonly_summary
backend.supported_config_items   -> advanced_raw, readonly_summary
```

### 5.3 Apply 回写

`ApplyEditPatchToConfigSet()` 要支持：

```text
env key/value -> runtime.env.value
model_mount fields -> runtime.model_mount.value
health fields -> runtime.health.value
device_table rows -> launcher.docker_options.value.devices
ulimits rows -> launcher.docker_options.value.ulimits
```

已有 `Path` 机制可以复用，但需保证：

- object path 不丢失已有字段。
- required 字段仍强制 enabled=true。
- readonly/internal 字段不能被 patch。

### 5.4 验收

1. ConfigEditView 普通区不再把 object 全部变成 raw_json。
2. object/list 字段要么结构化展示，要么进入高级只读摘要。
3. Apply patch 后内部 `config_set_json` 仍保持原格式。
4. RunPlan 不受影响。

---

## 6. 隐藏或解释 Backend capabilities / supported config items 等高级字段

### 6.1 当前问题

用户配置详情中出现：

```text
Backend capabilities
Backend supported config items
backend 开头的一堆参数
```

用户不知道这些是什么。

### 6.2 判断规则

普通用户应该看到：

```text
模型服务参数
后端启动参数
容器资源
设备与挂载
环境变量
服务入口
健康检查
```

普通用户不应该直接看到：

```text
Backend capabilities
supported config items
source metadata
internal capabilities
capabilities_detail
backend capability profile
```

这些属于平台能力描述或诊断信息，不是用户日常编辑项。

### 6.3 修改方法

在 `configedit` taxonomy 中处理：

```text
kind=metadata
kind=capability
category=capabilities
category=metadata
code contains capabilities
code contains supported_config
visibility=internal/hidden
support_level=reference
```

默认放入：

```text
advanced_raw / readonly_summary
```

普通编辑区不展示。

如果确实需要展示，显示为只读摘要：

```text
支持 OpenAI API: 是
支持模型格式: HuggingFace / GGUF
支持健康检查: 是
```

不要展示原始 JSON。

### 6.4 验收

1. 普通编辑区不显示 `Backend capabilities` JSON。
2. 不显示 `Backend supported config items` 大段 JSON。
3. 相关信息如需保留，只在“高级诊断 / 能力摘要”只读显示。

---

## 7. 修复节点运行配置选择器的技术化展示

### 7.1 当前问题

节点运行配置向导仍显示：

```text
runtime.llamacpp.cpu-docker
backend.llamacpp
cpu
ghcr.io/ggml-org/llama.cpp:server
```

### 7.2 修改方法

`NodeRuntimeConfigWizard.vue` 必须使用 `toRuntimeTemplateDisplay()`，与运行模板页共用展示规则。

表格建议列：

```text
名称：vLLM / NVIDIA 或用户 display_name
来源：内置模板 / 用户配置
后端：vLLM / SGLang / llama.cpp
厂商：NVIDIA / MetaX / Huawei Ascend / CPU
后端版本：*
镜像：xxx
```

raw id 只放 tooltip 或高级信息。

### 7.3 验收

1. 主标题不显示 `runtime.xxx`。
2. Backend 不显示 `backend.xxx`。
3. 内置模板显示“内置模板”。
4. 用户配置显示“用户配置”。
5. Backend Version 显示 `*`。

---

## 8. 同步检查所有可能复现的页面

### 8.1 必须检查

```text
web/src/pages/BackendRuntimesPage.vue
web/src/components/deployments/NodeRuntimeConfigWizard.vue
web/src/components/deployments/DeploymentOverrideEditor.vue
web/src/pages/BackendsPage.vue
web/src/pages/RunnerConfigsPage.vue
web/src/pages/ModelDeploymentsPage.vue
```

### 8.2 搜索项

执行：

```bash
grep -R "RuntimeParameterEditor" -n web/src
grep -R "JsonViewer" -n web/src/pages web/src/components
grep -R "Source Metadata" -n web/src
grep -R "launcher\." -n web/src
grep -R "runtime\." -n web/src
grep -R "\[object Object\]" -n web/src web/tests
```

### 8.3 处理原则

- 普通页面不直接显示 `RuntimeParameterEditor`，除非明确是迁移兼容或开发诊断。
- 普通页面不直接显示 `config_set` JsonViewer。
- `source_metadata` 不直接用英文标题，不直接 JSON dump。
- 运行模板、NBR、Deployment override 都应通过 `ConfigEditView` 展示。

---

## 9. i18n 修复

### 9.1 必须补齐中英文 key

至少新增：

```text
runtimes.cloneRuntimeTitle
runtimes.cloneAsUserConfig
runtimes.displayName
runtimes.technicalName
runtimes.userConfig
runtimes.builtinTemplate
runtimes.rename
runtimes.delete
runtimes.deleteConfirm
runtimes.configParametersReadonly
runtimes.sourceSummary
runtimes.developerDiagnostics
runtimes.rawConfigJson
runtimes.rawSourceMetadataJson

configEdit.sections.basic
configEdit.sections.modelServing
configEdit.sections.backendRuntime
configEdit.sections.containerResources
configEdit.sections.devicesMounts
configEdit.sections.environment
configEdit.sections.service
configEdit.sections.healthCheck
configEdit.sections.advancedRaw

configEdit.fields.image
configEdit.fields.command
configEdit.fields.entrypoint
configEdit.fields.sharedMemory
configEdit.fields.privileged
configEdit.fields.ipcMode
configEdit.fields.utsMode
configEdit.fields.networkMode
configEdit.fields.securityOptions
configEdit.fields.ulimits
configEdit.fields.devices
configEdit.fields.optionalDevices
configEdit.fields.additionalGroups
configEdit.fields.modelMount
configEdit.fields.environmentVariables
configEdit.fields.healthCheck
configEdit.fields.containerPort
configEdit.fields.hostPort
configEdit.placeholders.deploymentContainerPort
```

### 9.2 验收

1. 中文界面不出现新增英文硬编码。
2. `Source Metadata` 不再直接出现。
3. `Clone runtime`、`Display Name`、`Name` 不再直接出现。
4. Section 标题中文化。

---

## 10. 测试要求

必须运行：

```bash
go build ./cmd/server/...
go build ./cmd/agent/...
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm run build
cd web && npm test
```

### 10.1 后端新增/更新测试

1. `ProjectConfigSetToEditView` 不在普通 section 显示 `launcher.xxx` / `runtime.xxx` label。
2. `launcher.docker_options.devices` 对象数组投影为 `device_table`。
3. `runtime.model_mount` 投影为 `mount_form`。
4. `runtime.env` 投影为 `key_value_table`。
5. `runtime.health` 投影为 `health_check_form`。
6. `{{container_port}}` 在普通 view 中是只读说明或隐藏到高级区。
7. capability/support metadata 默认进入高级只读摘要。
8. `ApplyEditPatchToConfigSet` 能正确回写 env/model_mount/health/devices/ulimits。
9. clone runtime 支持 display_name/name，返回新对象。
10. delete user runtime 成功，delete system runtime 被拒绝。

### 10.2 前端新增/更新测试

1. clone dialog 不出现英文硬编码。
2. clone 后显示用户输入的 display_name。
3. clone 后自动选中新用户配置。
4. user config 行显示编辑/删除操作。
5. system template 行只显示复制操作。
6. ConfigField 渲染 object array 不出现 `[object Object]`。
7. Device table 显示 host_path/container_path。
8. Model mount 不显示整块 JSON。
9. Source Metadata 不直接作为英文标题出现。
10. NodeRuntimeConfigWizard selector 不显示 raw runtime id 作为主标题。
11. DeploymentOverrideEditor 不出现 `[object Object]` / raw JSON 普通输入。

---

## 11. 推荐执行顺序

Claude 按以下顺序执行，避免走偏：

```text
1. 修 runtimeDisplay.ts，保证名称/来源/版本显示正确。
2. 修 BackendRuntimesPage 的 clone dialog、clone 后选中、用户配置操作按钮。
3. 修 ConfigField.vue，先消灭 [object Object]。
4. 扩展 configedit projection，结构化 env/model_mount/health/devices/ports/capabilities。
5. 修诊断区：source metadata summary + raw JSON 折叠。
6. 同步 NodeRuntimeConfigWizard 使用统一 display model。
7. 检查 DeploymentOverrideEditor / BackendsPage / RunnerConfigsPage / ModelDeploymentsPage 是否复现。
8. 补 i18n。
9. 补测试。
10. 跑全量验证。
11. 更新 final-closeout.md。
12. commit + push。
```

---

## 12. Closeout 要求

更新：

```text
docs/reports/phase-3/runtime-template-catalog-redesign/final-closeout.md
```

新增章节：

```text
Post-closeout Runtime Template UX and ConfigEditView Display Repair
```

必须包含：

```text
1. 修复的问题清单
2. Runtime display model 规则
3. clone/rename/delete 行为
4. ConfigEditView 新增 widgets
5. object/list 字段结构化说明
6. raw JSON 保留位置
7. i18n 修复说明
8. 测试命令和结果
9. commit id
10. push result
11. git status
```

---

## 13. 提交

```bash
git status --short
git add .
git commit -m "web: polish runtime template config editing ux"
git push
```

最终输出：

```text
PASS/FAIL
commit id
push result
test summary
closeout path
remaining blocked items, if any
git status
```
