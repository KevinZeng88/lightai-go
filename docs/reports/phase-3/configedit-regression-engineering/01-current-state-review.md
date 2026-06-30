# 当前测试现状审查

## 1. 现有测试入口

当前 `web/package.json` 已经有完整测试入口：

```json
"test": "node tests/apiClientPaths.test.mjs && node tests/formatters.test.mjs && node tests/i18nKeys.test.mjs && node tests/i18nMissingKeys.test.mjs && node tests/noHardcodedCredentials.test.mjs && node tests/runtimeBoundaryUi.test.mjs && node tests/modelCapabilities.test.mjs && node tests/configEditContract.test.mjs && node tests/configEditUnwrap.test.mjs && npm run test:unit",
"test:unit": "vitest run",
"test:e2e": "playwright test"
```

说明项目已经具备三类测试能力：

```text
1. node tests/*.mjs 静态/契约检查
2. vitest 组件与工具测试
3. playwright e2e 测试
```

本轮不需要重新搭测试框架，应在现有框架上补强。

## 2. Vitest 当前覆盖范围

`web/vitest.config.ts` 当前包含：

```text
src/components/**/*.render.test.ts
src/components/**/__tests__/*.test.ts
src/pages/**/*.integration.test.ts
src/composables/__tests__/*.test.ts
src/stores/__tests__/*.test.ts
src/pages/__tests__/*.test.ts
src/utils/__tests__/*.test.ts
```

这说明新增的 ConfigEdit 测试应优先放在：

```text
web/src/utils/__tests__/configEditView.test.ts
web/src/components/config/__tests__/ConfigEditView.render.test.ts
web/src/components/config/__tests__/ConfigField.enabled-state.test.ts
web/src/components/config/__tests__/ConfigEditView.behavior.test.ts
```

## 3. 当前 ConfigEdit 共享工具状态

当前 `web/src/utils/configEditView.ts` 已经有这些核心能力：

```text
ConfigEditView / ConfigEditSection / ConfigEditField 类型
buildConfigEditPatch()
sortedSections()
sortedFields()
displayGroupForField()
```

当前 `sortedSections()` 会把所有字段收集起来，按 display group 分成：

```text
enabled
common
advanced
expert
```

并映射成：

```text
enabled_parameters
common_parameters
advanced_parameters_group
expert_parameters_group
```

这是对的，说明分组逻辑已经集中到共享工具，而不是散落在页面里。

## 4. 当前最大错误：enabled 与 expert 的优先级写反

当前 `displayGroupForField()` 的逻辑是：

```ts
export function displayGroupForField(field: ConfigEditField): ConfigEditDisplayGroup {
  if (isExpertField(field)) return 'expert'
  const enabledAtLoad = field.original_enabled ?? field.enabled
  if (enabledAtLoad) return 'enabled'
  if (field.advanced || field.view === 'advanced' || field.tier === 'advanced') return 'advanced'
  return 'common'
}
```

这会导致：

```text
enabled=true + high-risk/security -> expert
enabled=true + raw/diagnostic -> expert
```

这是错误产品行为。

正确行为应为：

```ts
export function displayGroupForField(field: ConfigEditField): ConfigEditDisplayGroup {
  const enabledAtLoad = field.original_enabled ?? field.enabled
  if (enabledAtLoad) return 'enabled'
  if (isExpertField(field)) return 'expert'
  if (field.advanced || field.view === 'advanced' || field.tier === 'advanced') return 'advanced'
  return 'common'
}
```

## 5. 当前单测也锁错了规则

`web/src/utils/__tests__/configEditView.test.ts` 当前有测试：

```text
keeps enabled high-risk and diagnostic fields in the expert group
```

这个测试现在应该删除或改反。

新规则应锁定：

```text
enabled high-risk/security/raw/diagnostic fields must be in enabled group
disabled high-risk/security/raw/diagnostic fields must be in expert group
```

## 6. 当前渲染测试有基础，但用例不够精确

`web/src/components/config/__tests__/ConfigEditView.render.test.ts` 当前已经测试：

```text
1. ConfigEditView 能渲染 display group
2. docker.shm_size 等字段不泄露 parent object
3. mount_form / health_check_form / key_value_table 是结构化渲染
4. runtime.env 在 enabled group
5. advanced raw 默认隐藏
6. readonly/editable 模式基本差异
7. 自包含字段 metadata 可以渲染
```

这些测试有价值，应保留。

缺口是：

```text
1. 没有明确测试 enabled high-risk/raw/diagnostic 必须进入 enabled group
2. 没有测试 enabled group 内风险元数据仍保留
3. 没有测试 checked/unchecked 编辑中不跳组的真实组件行为
4. 没有测试 ConfigField 的 enabled 与 value 分离保存行为
5. 没有测试 required 字段 patch 行为和 readonly 行为的组件层表现
6. 没有用 zh-CN i18n messages 验证 section label 中文化
```

## 7. 当前 ConfigField 能力与测试缺口

`ConfigField.vue` 当前特点：

```text
1. field.enabled 由独立 el-checkbox 控制
2. value 控件不因为 field.enabled=false 自动禁用
3. readonly / field.readonly / field.disabled 才会禁用控件
4. 支持 boolean/select/number/raw_json/accelerator_binding/key_value_table/device_table/mount_form/health_check_form/port_form/readonly_summary/textarea/default input
5. 使用 resolveConfigFieldLabel / Help / Tooltip
```

这说明 ConfigField 已经具备“enabled 与 value 分离”的基础，但需要更强的组件测试锁死：

```text
1. disabled parameter value input 仍可编辑
2. readonly mode 下 enabled checkbox/value input 不可编辑
3. required field 不显示 enabled checkbox，patch 强制 enabled=true
4. checkbox change 会触发 ConfigEditView patch
5. value change 会触发 ConfigEditView patch
6. raw_json 输入合法 JSON 时 value 为 object；非法 JSON 时不崩溃
7. key_value_table/device_table/mount_form/health_check_form 会输出结构化对象
```

## 8. 当前 static contract 测试已存在，但可拆分

`web/tests/configEditContract.test.mjs` 当前测试：

```text
1. buildConfigEditPatch enabled 变化
2. buildConfigEditPatch value 变化
3. required field 强制 enabled=true
4. ConfigField 稳定 selector
5. ConfigEditView 稳定 selector
6. ConfigSection 稳定 selector
```

这个文件适合作为“最低静态门禁”，但不应承载过多业务行为。复杂行为应转移到 Vitest 单测和组件测试。

## 9. 当前 runtimeBoundaryUi.test.mjs 已经过载

`web/tests/runtimeBoundaryUi.test.mjs` 目前包含大量跨主题源码扫描，包括 ConfigEdit、i18n、runtime、deployment、probe、wizard、display 等。它已经能挡住一些英文泄露和硬编码问题，但长期会变成难维护的大杂烩。

建议：

```text
1. 保留 runtimeBoundaryUi.test.mjs 作为大边界扫描。
2. 新增 web/tests/configEditRegressionBoundary.test.mjs，把 ConfigEdit 专项静态规则迁出去。
3. 后续逐步减少 runtimeBoundaryUi.test.mjs 中 ConfigEdit 细节。
```

## 10. Playwright 当前具备，但不应作为第一优先级

`web/playwright.config.ts` 已存在，能启动 Vite dev server 并运行 e2e。防回归工程第一阶段应先做单元/组件/静态测试，因为：

```text
1. ConfigEdit 是共享组件，组件测试能覆盖更多输入组合
2. Playwright 对本地服务、登录、数据准备依赖更重
3. 当前问题多是组件/契约层问题，而不是浏览器兼容问题
```

Playwright 适合作为第二阶段补一条核心 smoke：

```text
打开参数模板 / 运行模板 / 节点运行配置 / 模型部署，确认页面可见、非空、无英文泄露、切换显示级别不崩溃。
```

## 11. 现状结论

当前不是没有测试，而是测试缺少一套明确的 ConfigEdit 行为契约。尤其是：

```text
1. enabled 优先级规则写反
2. 组件测试没有覆盖 high-risk enabled 前置
3. ConfigField enabled/value 分离缺少强断言
4. view level 与 enabled visibility 的规则没有明确沉淀
5. runtimeBoundaryUi.test.mjs 过载，不适合作为唯一防线
```
