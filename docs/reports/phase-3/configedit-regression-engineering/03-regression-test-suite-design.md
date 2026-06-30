# ConfigEdit 防回归测试套件设计

## 1. 测试分层

建议形成五层防线：

```text
Layer 1: 纯函数单测
Layer 2: 组件渲染测试
Layer 3: 字段交互/patch 测试
Layer 4: 静态契约测试
Layer 5: 少量 Playwright smoke
```

第一阶段只做 Layer 1-4。Playwright 放第二阶段，避免初期引入过重依赖。

## 2. Layer 1 — 纯函数单测

### 文件

```text
web/src/utils/__tests__/configEditView.test.ts
```

### 覆盖函数

```text
displayGroupForField()
sortedSections()
sortedFields()
buildConfigEditPatch()
```

### 必须测试的分组矩阵

| Case | 输入 | 期望 |
|---|---|---|
| normal enabled | enabled=true, normal | enabled |
| advanced enabled | enabled=true, advanced=true/view=advanced | enabled |
| expert enabled | enabled=true, tier=expert/view=developer | enabled |
| security enabled | enabled=true, view=security/section=security_high_risk/risk=high | enabled |
| raw enabled | enabled=true, section=advanced_raw | enabled |
| diagnostic enabled | enabled=true, diagnostic=true | enabled |
| normal disabled | enabled=false, normal | common |
| advanced disabled | enabled=false, advanced=true/view=advanced | advanced |
| expert disabled | enabled=false, tier=expert/view=developer | expert |
| security disabled | enabled=false, view=security/section=security_high_risk | expert |
| raw disabled | enabled=false, section=advanced_raw | expert |
| diagnostic disabled | enabled=false, diagnostic=true | expert |

### 编辑稳定性测试

```text
original_enabled=false, enabled=false -> common
original_enabled=false, enabled=true  -> common
original_enabled=true, enabled=true   -> enabled
original_enabled=true, enabled=false  -> enabled
reload original_enabled=false         -> original group
reload original_enabled=true          -> enabled
```

### sortedSections 测试

构造混合字段：

```text
normal enabled
advanced enabled
security/high-risk enabled
raw/diagnostic enabled
normal disabled
advanced disabled
security/high-risk disabled
raw/diagnostic disabled
```

期望 section 顺序：

```text
enabled_parameters
common_parameters
advanced_parameters_group
expert_parameters_group
```

并期望字段归属：

```text
enabled_parameters:
  normal enabled
  advanced enabled
  security/high-risk enabled
  raw/diagnostic enabled

common_parameters:
  normal disabled

advanced_parameters_group:
  advanced disabled

expert_parameters_group:
  security/high-risk disabled
  raw/diagnostic disabled
```

### sortedFields 测试

继续保留当前排序逻辑测试：

```text
section rank -> order -> field key
```

并增加覆盖：

```text
model_serving before health_check
health_check before environment
environment before docker
docker before security_high_risk
security_high_risk before advanced_raw
```

### buildConfigEditPatch 测试

当前 `web/tests/configEditContract.test.mjs` 已有部分 patch 测试，但建议在 Vitest 中补更清晰的单元测试：

```text
1. value changed -> patch includes value
2. enabled changed -> patch includes enabled
3. enabled false but value changed -> patch still includes value
4. required true and enabled false -> patch enabled=true
5. readonly field changed locally -> no patch
6. no change -> no patch
7. semantic_key preferred over key
8. path preserved
```

## 3. Layer 2 — ConfigEditView 渲染测试

### 文件

```text
web/src/components/config/__tests__/ConfigEditView.render.test.ts
```

### 当前已有测试保留

保留现有测试：

```text
1. renders sections and structured runtime fields
2. docker child fields don't leak parent object
3. structured widgets instead of raw JSON
4. readonly/editable mode
5. self-contained metadata rendering
```

### 新增测试：enabled high-risk/raw 前置

测试名建议：

```text
renders all enabled fields in enabled group including high-risk and raw diagnostics
```

输入：

```text
model_runtime.dtype:
  enabled=true, normal

model_runtime.tensor_parallel_size:
  enabled=true, advanced=true, view=advanced

docker.privileged:
  enabled=true, view=security, section=security_high_risk, risk=high, tier=expert

advanced_raw_diag:
  enabled=true, section=advanced_raw, diagnostic=true, tier=expert

model_runtime.max_model_len:
  enabled=false, normal

docker.security_options:
  enabled=false, view=security, section=security_high_risk, risk=high, tier=expert

debug.source_map:
  enabled=false, section=advanced_raw, diagnostic=true
```

断言：

```text
enabled_parameters 存在并包含前 4 个 enabled 字段
common_parameters 包含 max_model_len
expert_parameters_group 包含 security_options 和 debug.source_map
```

### 新增测试：section 中文 label

当前 vitest setup 的 i18n messages 是空对象，因此要在测试里显式创建 zh-CN i18n。

测试名建议：

```text
renders localized display group labels in zh-CN
```

断言：

```text
已启用参数
常用参数
高级参数
专家参数
```

不要只断言 section key。要确保用户看到的是中文。

### 新增测试：expert group collapsed but enabled group open

规则：

```text
已启用参数默认展开
专家参数默认折叠
```

注意：如果 high-risk enabled 进入 enabled group，则它不应因为 expert 属性导致整组折叠。

## 4. Layer 3 — ConfigField enabled/value 状态测试

### 文件

```text
web/src/components/config/__tests__/ConfigField.enabled-state.test.ts
```

### 必测场景

#### 4.1 disabled 参数仍可编辑 value

输入：

```text
enabled=false
readonly=false
disabled=false
widget=number
value=4096
```

断言：

```text
enabled checkbox unchecked
value input 不 disabled
修改 value 后 emit change
```

#### 4.2 取消 enabled 不清空 value

流程：

```text
enabled=true, value=4096
取消 checkbox
value 仍为 4096
emit change
```

#### 4.3 readonly 禁止编辑

输入：

```text
readonly prop=true
```

断言：

```text
enabled checkbox disabled
value input disabled
字段仍显示
```

#### 4.4 required 字段不显示 enabled checkbox

输入：

```text
required=true
has_enable=true
enabled=false
```

断言：

```text
没有 config-field-enabled checkbox
显示 required tag
```

#### 4.5 风险元数据可测试

如果本轮增加风险 tag / data 属性，断言：

```text
data-field-risk="high"
data-field-tier="expert"
显示 高风险 tag
```

## 5. Layer 4 — 静态契约测试

### 新文件

```text
web/tests/configEditRegressionBoundary.test.mjs
```

不要继续把所有 ConfigEdit 规则塞进 `runtimeBoundaryUi.test.mjs`。

### 测试内容

#### 5.1 displayGroupForField 顺序门禁

源码级检查：

```text
displayGroupForField 中 enabledAtLoad 判断必须出现在 isExpertField 判断之前
```

这可以防止未来有人又把 expert 优先级调回去。

#### 5.2 ConfigEdit 稳定 selector

迁移或保留当前 `configEditContract.test.mjs` 中 selector 断言：

```text
data-testid="config-edit-view"
data-testid="config-edit-section"
data-testid="config-field"
data-testid="config-field-enabled"
data-testid="config-field-value"
```

#### 5.3 ConfigEdit 不可回退旧组件

断言：

```text
BackendRuntimesPage / RunnerConfigsPage / ModelDeploymentsPage 不导入 RuntimeParameterEditor
不导入 HumanRuntimeParameterForm
不使用 getHumanFieldsForBackend
```

#### 5.4 Consumer page view-level conformance

检查这些文件：

```text
web/src/pages/BackendRuntimesPage.vue
web/src/pages/RunnerConfigsPage.vue
web/src/pages/ModelDeploymentsPage.vue
```

每个应满足：

```text
使用 configEditViewLevelOptions(t)
使用 configEditViewLevelHelp(t)
不硬编码 Normal / Advanced / Developer
切换 configViewLevel 时重新 getConfigEditView，或有明确等价逻辑
切换时清空对应 patch
raw JsonViewer 只在 configViewLevel === 'developer' 时显示
```

注意：静态测试不要太脆弱，不要依赖完全相同的变量名。可以允许页面提供一个注释标记，例如：

```text
CONFIGEDIT_LEVEL_RELOAD_GUARD
```

或者封装成共享 composable 后测试 composable。

## 6. Layer 5 — Playwright smoke，第二阶段

### 文件建议

```text
web/tests/e2e/configedit-smoke.spec.ts
```

### 最小 smoke

```text
1. 登录
2. 打开 /config-edit/templates
3. 页面显示“参数模板”
4. 模板列表非空
5. 打开 /runtimes
6. 打开某个运行模板
7. 切换 常用 / 高级 / 专家
8. 不出现 Normal / Advanced / Developer
9. 打开 /runner-configs
10. 打开某个 NBR
11. 切换 常用 / 高级 / 专家
12. 打开 /models/deployments
13. 编辑部署，切换 常用 / 高级 / 专家
```

E2E 需要依赖测试数据，不建议第一阶段强制作为 CI blocker。

## 7. 测试命名建议

### configEditView.test.ts

```text
groups every enabled field before tier-specific buckets
keeps disabled expert/security/raw/diagnostic fields in expert group
keeps placement stable during edit session using original_enabled
moves fields after reload when original_enabled changes
sorts fields by section rank order and stable key
builds patches for enabled and value changes independently
does not patch readonly fields
forces required fields enabled in patch
```

### ConfigEditView.render.test.ts

```text
renders all enabled fields in enabled group including high-risk and raw diagnostics
renders disabled fields in common advanced expert groups
renders zh-CN display group labels
keeps enabled group expanded and expert group collapsed
does not render structured fields as raw parent JSON
```

### ConfigField.enabled-state.test.ts

```text
keeps value editable when optional field is disabled
does not clear value when enabled checkbox is unchecked
disables checkbox and value controls in readonly mode
hides enabled checkbox for required fields
emits change when enabled toggles
emits change when value changes
renders risk metadata for high-risk enabled fields
```

## 8. 不建议做的测试

```text
1. 不要只 grep 某个中文词来代表行为正确。
2. 不要只测试 HTTP 200。
3. 不要为每个页面复制一套 ConfigEdit 分组测试。
4. 不要把页面业务流程测试当成 ConfigEdit 组件测试的替代。
5. 不要把所有新规则塞进 runtimeBoundaryUi.test.mjs。
```
