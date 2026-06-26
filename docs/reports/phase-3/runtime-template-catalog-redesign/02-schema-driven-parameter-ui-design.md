# 02 - Schema-driven 参数 UI 设计

## 1. 目标

实现：

```text
在 BackendVersion 增加一个参数 schema 后，不修改前端代码，界面自动多出输入项。
```

例如在 BackendVersion 的 `config_set.items` 中新增：

```json
{
  "backend.arg.enable_prefix_caching": {
    "code": "backend.arg.enable_prefix_caching",
    "category": "model_runtime",
    "kind": "cli_arg",
    "type": "boolean",
    "enabled": false,
    "value": false,
    "default_value": false,
    "required": false,
    "render": {
      "label": "启用 Prefix Cache",
      "help": "启用前缀缓存以提升重复前缀场景下的推理效率",
      "flag": "--enable-prefix-caching",
      "style": "flag_if_true",
      "target": "cli",
      "group": "性能参数"
    },
    "order": 340,
    "support_level": "documented"
  }
}
```

则以下界面自动出现该参数：

- BackendVersion 版本参数编辑页
- BackendRuntimeTemplate / BackendRuntime 参数页
- NodeBackendRuntime 参数页
- Deployment override 页
- Dry-run / RunPlan preview

---

## 2. 当前代码现状

### 2.1 已经具备的基础

`RuntimeParameterEditor.vue` 已经按 `config_set.items` 渲染参数：

- `boolean` → switch
- `array / lines / object` → textarea
- 其他 → input
- 每个参数保留 `enabled + value`
- 输出 `config_set` 和 `config_overrides.parameter_values`

这是正确基础。

### 2.2 当前阻断点

`HumanRuntimeParameterForm.vue` 和 `runtimeParameterViewModel.ts` 仍然硬编码了人类友好字段：

```ts
const HUMAN_FIELDS: HumanRuntimeField[] = [...]
```

这会导致：

- 新参数只能出现在高级 ConfigSet 编辑器里。
- 友好表单不会自动新增。
- 每个后端新增参数都要改前端代码。

---

## 3. 推荐实现

### 3.1 用 SchemaDrivenParameterForm 替换 HumanRuntimeParameterForm

新增组件：

```text
web/src/components/runtime/SchemaDrivenParameterForm.vue
```

职责：

- 接收 `configSet`
- 自动过滤内部项
- 自动分组
- 自动排序
- 自动按 type 渲染控件
- 自动校验
- 输出 `config_set` 或 `config_overrides`

建议 props：

```ts
defineProps<{
  configSet: Record<string, any> | null
  readonly?: boolean
  mode: 'edit_snapshot' | 'override'
  layer: 'backend' | 'backend_version' | 'backend_runtime' | 'node_backend_runtime' | 'deployment'
  showAdvanced?: boolean
  showInternal?: boolean
  baseConfigSet?: Record<string, any> | null
}>()
```

建议 emits：

```ts
'update:config-set'
'update:overrides'
'validate'
```

### 3.2 保留 RuntimeParameterEditor，先增强再统一

也可以不新增组件，直接增强 `RuntimeParameterEditor.vue`：

1. 读取 `extensions.label`。
2. 读取 `extensions.group`。
3. 支持 `order`。
4. 支持 top-level `constraints`。
5. 支持 `required`。
6. 支持 `select` / `multi_select`。
7. 支持 `readonly` / `advanced` / `visibility`。
8. 支持 `render.help`。
9. 支持 `placeholder`。
10. 支持 `render.options`。

如果希望少改文件，优先增强 `RuntimeParameterEditor.vue`，再删除 `HumanRuntimeParameterForm.vue` 的使用。

---

## 4. 统一 ConfigItem schema

建议最终 ConfigItem 支持：

```yaml
code: backend.arg.gpu_memory_utilization
category: model_runtime
kind: cli_arg
type: number
required: false
enabled: true
value: 0.9
default_value: 0.9
order: 310
support_level: documented
visibility: visible
readonly: false
advanced: false
constraints:
  min: 0.1
  max: 1.0
  step: 0.01
render:
  label: GPU 显存利用率
  help: 控制 vLLM/SGLang 可使用的 GPU 显存比例
  flag: --gpu-memory-utilization
  style: flag_space_value
  target: cli
  group: 资源控制
  placeholder: "0.90"
  unit: ratio
source:
  layer: BackendVersion
  ref: backend-version.vllm.compat
  reason: default_args_schema
```

字段说明：

| 字段 | 用途 |
|---|---|
| code | ConfigSet 内部唯一 key |
| category | 大类：launcher / runtime_env / model_runtime |
| kind | cli_arg / cli_args / env / port / volume / device / health_check / launcher_option |
| type | string / integer / number / boolean / select / multi_select / array / object / lines / path / file |
| required | 是否必填 |
| enabled | 是否参与最终 RunPlan |
| value | 当前层快照值 |
| default_value | 创建该层时复制得到的默认值 |
| order | UI 排序 |
| support_level | verified / documented / experimental |
| visibility | visible / hidden / internal |
| readonly | 当前层是否只读 |
| advanced | 是否默认放高级区 |
| constraints | min/max/step/regex/required/options |
| render | UI 和 RunPlan 渲染信息 |
| source | 参数来源快照信息 |

---

## 5. UI 渲染规则

### 5.1 是否展示

默认展示：

```text
visibility != internal
kind in cli_arg / cli_args / env / launcher_option / health_check / port / volume
```

默认隐藏：

```text
launcher.command
launcher.args
launcher.entrypoint
launcher.docker_options
runtime.env
internal.*
resolver.*
source_metadata.*
MODEL_CONTAINER_PATH
MODEL_CONTAINER_DIR
```

隐藏项仍在 ConfigSet 中保留，不允许丢失。

### 5.2 label

解析顺序：

```ts
item.render?.label
|| item.extensions?.label
|| item.label
|| item.code
```

### 5.3 help

解析顺序：

```ts
item.render?.help
|| item.extensions?.help
|| item.description
|| ''
```

### 5.4 group

解析顺序：

```ts
item.render?.group
|| item.extensions?.group
|| item.category
```

### 5.5 constraints

解析顺序：

```ts
item.constraints
|| item.render?.constraints
|| {}
```

### 5.6 sorting

排序规则：

```ts
categoryOrder(category)
groupOrder(group)
item.order ?? 9999
item.code
```

---

## 6. 各层页面如何使用

### 6.1 Backend 页面

`BackendsPage.vue` 增加：

- Backend 列表
- 点击 Backend 后显示：
  - Backend 基础信息
  - Backend ConfigSet
  - BackendVersion 列表

Backend 本身系统内置可只读，后续可支持 clone/custom backend。

### 6.2 BackendVersion 页面

新增或嵌入：

```text
/backends/:backendId/versions
```

功能：

- 列出版本
- 新增版本
- 复制系统版本
- 编辑用户版本
- 删除未被 Runtime 使用的用户版本
- 编辑参数 schema
- 新增参数
- 预览表单
- 预览 CLI 参数

BackendVersion 参数编辑直接操作 `config_set.items`。

### 6.3 BackendRuntime 页面

当前 `BackendRuntimesPage.vue` 应把 `HumanRuntimeParameterForm` 替换为 schema-driven form。

系统 runtime 只读；用户 clone 后可编辑。

### 6.4 NodeBackendRuntime 页面 / Wizard

当前 `NodeRuntimeConfigWizard.vue` Step 2 应替换为 schema-driven form。创建 NBR 时，前端可以带覆盖值，但真正的 copy-on-create 必须由后端完成。

### 6.5 Deployment override 页面

当前 `DeploymentOverrideEditor.vue` 已经使用 `RuntimeParameterEditor`。增强 editor 后即可满足目标。

---

## 7. 新增参数的推荐交互

BackendVersion 详情页提供“新增参数”表单：

| 字段 | 示例 |
|---|---|
| 参数 code | backend.arg.max_model_len |
| 显示名 | 最大上下文长度 |
| 类型 | integer |
| CLI flag | --max-model-len |
| 默认值 | 空或 8192 |
| 默认启用 | false |
| 分组 | 模型参数 |
| 排序 | 320 |
| 最小值 | 1 |
| 最大值 | 131072 |
| 渲染方式 | flag_space_value |

保存后写入：

```json
config_set.items["backend.arg.max_model_len"]
```

不需要新增数据库列。

---

## 8. RunPlan 生成规则

只有满足以下条件的参数进入最终命令：

```text
kind in cli_arg / cli_args
enabled == true
value 非空，或 boolean + flag_if_true
```

参数转 CLI：

| style | 输出 |
|---|---|
| flag_space_value | `--flag value` |
| flag_equals_value | `--flag=value` |
| flag_if_true | `--flag` |
| repeat_flag | `--flag v1 --flag v2` |
| positional | `value` |
| raw_lines | 按行/空格展开 |

Deployment override：

- enabled=true 覆盖当前 Deployment snapshot 中同 code 的值。
- enabled=false 形成 disabled tombstone。
- 不允许 Deployment override `--host`、`--port` 这类受 service 控制的参数。

---

## 9. 必须新增的前端测试

建议新增：

```text
web/tests/schemaDrivenRuntimeParameters.test.mjs
```

覆盖：

1. 给 config_set 动态加入 fake 参数，不改 `HUMAN_FIELDS`，表单自动出现。
2. label 来自 `render.label`。
3. label 来自 `extensions.label`。
4. order 生效。
5. constraints.min/max 生效。
6. boolean + flag_if_true 生效。
7. disabled 参数不进入 overrides。
8. object/array/lines 不丢失。
9. hidden/internal 字段不在普通表单展示，但保存后仍保留。
10. BackendRuntime / NBR / Deployment 三处使用同一 schema-driven 逻辑。
