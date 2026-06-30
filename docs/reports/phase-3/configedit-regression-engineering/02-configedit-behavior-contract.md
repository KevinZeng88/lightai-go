# ConfigEdit 行为契约

## 1. 适用范围

本契约适用于所有使用 ConfigEdit 的位置，包括但不限于：

```text
运行模板 / BackendRuntime
节点运行配置 / NodeBackendRuntime
模型部署 / Deployment
部署向导 override
后续模型库 / 模型位置参数编辑
参数模板 / ConfigEdit Templates 观测页
```

实现层面至少覆盖：

```text
web/src/utils/configEditView.ts
web/src/components/config/ConfigEditView.vue
web/src/components/config/ConfigSection.vue
web/src/components/config/ConfigField.vue
web/src/utils/configEditFieldMeta.ts
web/src/utils/configEditDisplay.ts
```

## 2. 分组优先级契约

### 2.1 正确分组顺序

页面加载后，字段按以下顺序分组：

```text
1. 已启用参数
2. 常用参数
3. 高级参数
4. 专家参数
```

### 2.2 enabled 优先级最高

`enabled=true` 的字段，无论原始 tier/view/section/risk 是什么，都进入“已启用参数”。

```text
enabled=true + normal      -> 已启用参数
enabled=true + advanced    -> 已启用参数
enabled=true + developer   -> 已启用参数
enabled=true + security    -> 已启用参数
enabled=true + high-risk   -> 已启用参数
enabled=true + raw         -> 已启用参数
enabled=true + diagnostic  -> 已启用参数
```

原因：

```text
已启用参数代表“当前实际生效的配置”。
高风险参数如果已经启用，更应该前置显示，不能藏在后面的专家参数里。
```

### 2.3 disabled 字段才按 tier/view/risk 分组

只有未启用字段才按常用/高级/专家归类：

```text
enabled=false + normal      -> 常用参数
enabled=false + advanced    -> 高级参数
enabled=false + developer   -> 专家参数
enabled=false + security    -> 专家参数
enabled=false + high-risk   -> 专家参数
enabled=false + raw         -> 专家参数
enabled=false + diagnostic  -> 专家参数
```

### 2.4 分组实现建议

`displayGroupForField()` 应按这个顺序判断：

```ts
const enabledAtLoad = field.original_enabled ?? field.enabled
if (enabledAtLoad) return 'enabled'
if (isExpertField(field)) return 'expert'
if (isAdvancedField(field)) return 'advanced'
return 'common'
```

禁止把 expert 判断放在 enabled 之前。

## 3. 编辑中不跳组契约

分组必须基于 **加载时 enabled 状态**，不是编辑中的实时 checkbox 状态。

```text
original_enabled=false, enabled=false -> 当前显示在原始分组
用户勾选后：original_enabled=false, enabled=true -> 当前仍留在原始分组
保存刷新后：original_enabled=true, enabled=true -> 进入已启用参数
用户取消勾选后：original_enabled=true, enabled=false -> 当前仍留在已启用参数
保存刷新后：original_enabled=false, enabled=false -> 回到原始分组
```

目的：

```text
1. 避免勾选/取消时页面跳动
2. 避免用户正在填值时字段位置改变
3. 保持一次编辑会话中的视觉稳定
```

## 4. 风险标识契约

enabled high-risk 字段进入“已启用参数”后，不能丢失风险信息。

字段元数据必须保留：

```text
risk=high
tier=expert
view=security/developer
section=security_high_risk / advanced_raw
diagnostic=true
visibility=internal/hidden
```

UI 层应能显示或至少保留可测试的风险信息，例如：

```text
[专家]
[高风险]
[诊断]
[原始配置]
```

最低要求：

```text
ConfigField DOM 上保留 data-risk / data-tier / data-view / data-section-key 等可测试属性。
```

建议增强：

```text
ConfigField.vue 在字段标题附近显示风险 tag：
- risk=high -> 高风险
- tier=expert 或 view=developer/security -> 专家
- diagnostic=true -> 诊断
```

## 5. view level 契约

显示级别不是运行模式，也不是配置 profile。它只是“未启用字段”的查看范围。

### 5.1 显示级别定义

```text
常用 normal:
  显示 已启用参数 + 未启用常用参数

高级 advanced:
  显示 已启用参数 + 未启用常用参数 + 未启用高级参数

专家 developer:
  显示 已启用参数 + 未启用常用参数 + 未启用高级参数 + 未启用专家参数
```

### 5.2 enabled 永远可见

任何已经启用的参数，无论属于专家/高风险/raw，都应在所有 view level 中可见。

这条规则非常重要：

```text
用户在“常用”视图下也必须知道当前配置里已经启用了哪些危险或专家参数。
```

### 5.3 实现位置

当前系统的 view level 过滤主要由后端 `/config-edit/view` 或页面重新拉取 view 完成。ConfigEdit 组件本身目前不负责本地 view level 过滤。

因此第一阶段测试要求：

```text
1. ConfigEdit shared component 对传入字段执行正确分组。
2. Consumer page 切换 view level 时重新请求 ConfigEdit view。
3. 后续如引入前端本地过滤，必须按本契约实现。
```

## 6. 排序契约

在每个 display group 内部，字段按以下顺序排序：

```text
1. section rank
2. display order
3. stable field key
```

建议 section rank：

```text
model / model_serving
runtime / backend_runtime
resource / container_resources
service
health / health_check
mount / devices_mounts
env / environment
docker
security / security_high_risk
raw / advanced_raw
unknown
```

## 7. enabled 与 value 分离契约

ConfigField 必须把“启用状态”和“参数值”分开处理：

```text
field.enabled 控制是否写入运行计划 / patch 生效
field.value 保存参数值
```

必须满足：

```text
1. enabled=false 时 value 仍可见。
2. enabled=false 时 value 仍可编辑，除非 readonly/disabled。
3. 用户取消 enabled 不应清空 value。
4. patch 同时携带 value 和 enabled。
5. required=true 的字段不能被禁用，patch 应强制 enabled=true。
```

## 8. readonly 契约

readonly=true 时：

```text
1. enabled checkbox 不可编辑。
2. value 控件不可编辑。
3. 字段仍应显示。
4. 结构化 widget 仍以只读方式显示，不退回 raw JSON。
```

## 9. raw JSON 契约

普通字段不能退回 raw JSON-only。

允许 raw JSON 的情况：

```text
1. widget=raw_json 的专家/诊断字段
2. developer diagnostics 区域
3. ConfigEdit Templates 的 diagnostics JSON tab
```

禁止：

```text
1. 已知结构化字段显示为一整个父对象 JSON。
2. model_runtime.* / launcher.docker_options.* / runtime.env / runtime.health / mount 等结构化字段丢失后退成 raw JSON-only。
```

## 10. i18n 契约

ConfigEdit 普通 UI 不应显示后端原始英文 section label。

应使用：

```text
section.key -> configEdit.sections.*
field.label_i18n_key / configEdit.labels.* / fallback humanize
help_i18n_key / configEdit.descriptions.*
```

ConfigSection 当前已有 `SECTION_I18N_MAP`。测试应锁住：

```text
enabled_parameters -> 已启用参数
common_parameters -> 常用参数
advanced_parameters_group -> 高级参数
expert_parameters_group -> 专家参数
```

组件测试如果要断言中文，必须显式创建 zh-CN i18n，而不是依赖当前 vitest.setup.ts 的空 messages。

## 11. patch 契约

`buildConfigEditPatch()` 应只输出变化字段：

```text
1. value 变化 -> 输出 patch
2. enabled 变化 -> 输出 patch
3. value 和 enabled 都没变 -> 不输出
4. readonly 字段 -> 不输出
5. required 字段即使 enabled=false，也按 enabled=true 处理
```

## 12. 稳定 selector 契约

所有 ConfigEdit 测试依赖稳定 selector。必须保留：

```text
ConfigEditView:
  data-testid="config-edit-view"
  data-layer
  data-object-kind
  data-object-id

ConfigSection:
  data-testid="config-edit-section"
  data-section-key

ConfigField:
  data-testid="config-field"
  data-field-key
  data-internal-key

ConfigField enabled:
  data-testid="config-field-enabled"

ConfigField value:
  data-testid="config-field-value"
```

建议新增：

```text
data-field-tier
data-field-view
data-field-risk
data-field-diagnostic
```
