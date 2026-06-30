# 可执行开发计划

## Phase 0 — Baseline 确认

### 目标

Codex 开始前先确认当前仓库状态和现有测试，不直接改代码。

### 命令

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go

git status --short
git log --oneline -8

sed -n '1,320p' web/src/utils/configEditView.ts
sed -n '1,220p' web/src/utils/__tests__/configEditView.test.ts
sed -n '1,340p' web/src/components/config/__tests__/ConfigEditView.render.test.ts
sed -n '1,260p' web/tests/configEditContract.test.mjs
sed -n '1,260p' web/tests/runtimeBoundaryUi.test.mjs
sed -n '1,160p' web/package.json
sed -n '1,120p' web/vitest.config.ts
```

### 输出

在报告目录新增：

```text
phase-0-baseline.md
```

必须记录：

```text
1. 当前 displayGroupForField 的判断顺序
2. 当前 configEditView.test.ts 是否仍有 enabled high-risk 留 expert 的错误断言
3. 当前 ConfigEditView.render.test.ts 已有覆盖点
4. 当前 configEditContract.test.mjs 覆盖点
5. 当前 runtimeBoundaryUi.test.mjs 中 ConfigEdit 相关扫描项
```

## Phase 1 — 修正 enabled 分组契约

### 目标

修正核心行为：enabled=true 优先进入“已启用参数”。

### 修改文件

```text
web/src/utils/configEditView.ts
web/src/utils/__tests__/configEditView.test.ts
```

### 修改要求

`displayGroupForField()` 改为：

```ts
export function displayGroupForField(field: ConfigEditField): ConfigEditDisplayGroup {
  const enabledAtLoad = field.original_enabled ?? field.enabled
  if (enabledAtLoad) return 'enabled'
  if (isExpertField(field)) return 'expert'
  if (field.advanced || field.view === 'advanced' || field.tier === 'advanced') return 'advanced'
  return 'common'
}
```

当前测试中如果存在：

```text
keeps enabled high-risk and diagnostic fields in the expert group
```

必须删除或改为：

```text
puts enabled high-risk and diagnostic fields in the enabled group
```

### 必须新增单测

```text
enabled=true normal -> enabled
enabled=true advanced -> enabled
enabled=true expert -> enabled
enabled=true high-risk/security -> enabled
enabled=true raw/diagnostic -> enabled
enabled=false high-risk/security -> expert
enabled=false raw/diagnostic -> expert
```

## Phase 2 — 补齐 ConfigEditView 渲染矩阵

### 目标

用真实组件渲染测试锁定 display group 与字段归属。

### 修改文件

```text
web/src/components/config/__tests__/ConfigEditView.render.test.ts
```

### 新增测试

```text
renders all enabled fields in enabled group including high-risk and raw diagnostics
renders disabled fields in common advanced expert groups
renders zh-CN display group labels
keeps enabled group expanded and expert group collapsed
```

### 注意

当前 `tests/setup/vitest.setup.ts` 的 i18n messages 是空对象。要测试中文 label，应在测试文件中局部创建 i18n：

```ts
import { createI18n } from 'vue-i18n'
import zhCN from '@/locales/zh-CN'

const zhI18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages: { 'zh-CN': zhCN },
})
```

mount 时使用：

```ts
global: { plugins: [ElementPlus, zhI18n] }
```

## Phase 3 — 新增 ConfigField enabled/value 状态测试

### 目标

锁定 ConfigField 的 enabled 与 value 分离契约。

### 新增文件

```text
web/src/components/config/__tests__/ConfigField.enabled-state.test.ts
```

### 必须覆盖

```text
1. optional field enabled=false 时 value 控件仍可编辑
2. 取消 enabled 不清空 value
3. readonly=true 时 checkbox/value 控件不可编辑
4. required=true 不显示 enabled checkbox
5. checkbox change 会 emit change
6. value change 会 emit change
7. raw_json 合法 JSON 输入输出 object
8. raw_json 非法 JSON 不崩溃
```

### 可选增强

如果同时增加风险 tag/data 属性：

```text
high-risk 字段显示 高风险 / 专家 tag
DOM 保留 data-field-risk / data-field-tier / data-field-view
```

## Phase 4 — 新增 ConfigEdit 专项静态边界测试

### 目标

把 ConfigEdit 专项边界从 `runtimeBoundaryUi.test.mjs` 的大杂烩中独立出来。

### 新增文件

```text
web/tests/configEditRegressionBoundary.test.mjs
```

### 同步修改

```text
web/package.json
```

把它加入 `npm test`，建议放在 `configEditContract.test.mjs` 后面：

```json
node tests/configEditContract.test.mjs && node tests/configEditRegressionBoundary.test.mjs && ...
```

### 必须检查

```text
1. displayGroupForField 中 enabledAtLoad 判断在 isExpertField 之前
2. ConfigEditView / ConfigSection / ConfigField 稳定 selector 存在
3. ConfigField 不因 field.enabled=false 禁用 value 控件
4. BackendRuntimesPage / RunnerConfigsPage / ModelDeploymentsPage 使用 shared ConfigEdit level options/help
5. 三个页面不硬编码 Normal / Advanced / Developer
6. 三个页面在 view level 切换时有重新 getConfigEditView 或等价 guard
7. raw diagnostics 只在 developer 模式显示
8. RuntimeParameterEditor / HumanRuntimeParameterForm 不在活动页面使用
```

### 静态测试不要过脆

如变量名不一致，可以引入显式注释标记：

```text
// CONFIGEDIT_LEVEL_RELOAD_GUARD
```

但不要用注释替代真实逻辑。

## Phase 5 — 可选 UI 风险标识增强

### 背景

enabled high-risk 参数进入“已启用参数”后，需要用户能识别风险。

### 推荐增强

修改：

```text
web/src/components/config/ConfigField.vue
web/src/locales/zh-CN.ts
web/src/locales/en-US.ts
```

增加：

```text
risk=high -> 高风险 / High risk
tier=expert 或 view=developer/security -> 专家 / Expert
diagnostic=true -> 诊断 / Diagnostic
```

并在 ConfigField DOM 上增加：

```text
:data-field-tier="field.tier || ''"
:data-field-view="field.view || ''"
:data-field-risk="field.risk || ''"
:data-field-diagnostic="field.diagnostic ? 'true' : 'false'"
```

如果 `ConfigEditField` 类型当前没有 risk 字段，需要补：

```ts
risk?: string
```

对应后端 `TemplateField` 已有 risk，但 ConfigEditView field 是否有 risk 需要 Codex 查代码确认。

### 验收

```text
enabled high-risk 字段在“已启用参数”里，同时显示 高风险 标识。
```

如果本阶段风险较大，可先只加 data 属性和测试，视觉 tag 留后续。

## Phase 6 — 文档与 closeout

新增：

```text
docs/reports/phase-3/configedit-regression-engineering/
  phase-0-baseline.md
  configedit-regression-engineering-closeout.md
```

closeout 必须包含：

```text
Root Cause / Gap
Changed Files
New Tests
What each test prevents
Verification Commands
Commit
Push
git status
Final status
```

禁止留下：

```text
TBD
TODO
recorded in final report
pending commit
pending push
```

## Phase 7 — 验证

必须执行：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go

go test ./...

cd web
npm run test:unit
npm test
npm run build
```

如果本轮新增 Playwright，再执行：

```bash
cd web
npm run test:e2e -- --grep "ConfigEdit"
```

但第一阶段不强制新增 Playwright。
