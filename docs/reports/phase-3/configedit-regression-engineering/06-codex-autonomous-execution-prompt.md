# Codex Autonomous Execution Prompt — ConfigEdit Regression Engineering

请在 LightAI Go 仓库中执行 ConfigEdit 防回归工程 Phase 1。

## 背景

ConfigEdit 是 LightAI Go 的共享参数编辑底座，运行模板、节点运行配置、模型部署、部署 override 等多个地方都依赖它。近期多次问题说明需要把 ConfigEdit 的核心行为沉淀成稳定测试，而不是只在具体页面发现问题后修补。

本轮重点是 ConfigEdit 共享组件和共享工具：

```text
web/src/utils/configEditView.ts
web/src/components/config/ConfigEditView.vue
web/src/components/config/ConfigSection.vue
web/src/components/config/ConfigField.vue
web/src/utils/configEditFieldMeta.ts
web/src/utils/configEditDisplay.ts
```

## 必须先读文档

请先阅读：

```text
docs/reports/phase-3/configedit-regression-engineering/01-current-state-review.md
docs/reports/phase-3/configedit-regression-engineering/02-configedit-behavior-contract.md
docs/reports/phase-3/configedit-regression-engineering/03-regression-test-suite-design.md
docs/reports/phase-3/configedit-regression-engineering/04-executable-implementation-plan.md
docs/reports/phase-3/configedit-regression-engineering/05-acceptance-and-ci-gates.md
```

如果这些文档尚未复制到仓库，请先让用户复制，或按当前 prompt 中的同等要求执行。

## 当前需要修正的关键规则

现在的正确产品规则是：

```text
enabled=true 的参数全部进入“已启用参数”，包括 expert / high-risk / raw / diagnostic 参数。
enabled=false 的 high-risk / raw / diagnostic 参数才进入“专家参数”。
```

原因：

```text
已启用参数代表当前实际生效配置。
已经启用的高风险配置必须前置显示，让用户第一眼看到。
风险控制应通过字段风险标识和告警完成，而不是把已启用高风险字段藏在专家参数后面。
```

## Phase 0 — Baseline

执行：

```bash
git status --short
git log --oneline -8

sed -n '1,320p' web/src/utils/configEditView.ts
sed -n '1,220p' web/src/utils/__tests__/configEditView.test.ts
sed -n '1,340p' web/src/components/config/__tests__/ConfigEditView.render.test.ts
sed -n '1,260p' web/tests/configEditContract.test.mjs
sed -n '1,260p' web/tests/runtimeBoundaryUi.test.mjs
```

新增或更新：

```text
docs/reports/phase-3/configedit-regression-engineering/phase-0-baseline.md
```

记录当前测试缺口和需要修改的测试。

## Phase 1 — 修正 displayGroupForField

修改：

```text
web/src/utils/configEditView.ts
```

要求 `displayGroupForField()` 先判断 enabledAtLoad：

```ts
export function displayGroupForField(field: ConfigEditField): ConfigEditDisplayGroup {
  const enabledAtLoad = field.original_enabled ?? field.enabled
  if (enabledAtLoad) return 'enabled'
  if (isExpertField(field)) return 'expert'
  if (field.advanced || field.view === 'advanced' || field.tier === 'advanced') return 'advanced'
  return 'common'
}
```

禁止 expert 判断优先于 enabled。

## Phase 2 — 增强 configEditView 单测

修改：

```text
web/src/utils/__tests__/configEditView.test.ts
```

删除或改正当前错误断言：

```text
keeps enabled high-risk and diagnostic fields in the expert group
```

新增/覆盖测试：

```text
enabled=true normal -> enabled
enabled=true advanced -> enabled
enabled=true expert -> enabled
enabled=true high-risk/security -> enabled
enabled=true raw/diagnostic -> enabled
enabled=false high-risk/security -> expert
enabled=false raw/diagnostic -> expert
enabled=false advanced -> advanced
enabled=false normal -> common
编辑中 original_enabled=false, enabled=true 不跳到 enabled
编辑中 original_enabled=true, enabled=false 仍留 enabled
保存刷新后 original_enabled 更新，分组随之改变
section 内部按 section rank -> order -> key 排序
buildConfigEditPatch value/enabled/required/readonly 行为
```

## Phase 3 — 增强 ConfigEditView 渲染测试

修改：

```text
web/src/components/config/__tests__/ConfigEditView.render.test.ts
```

新增测试：

```text
renders all enabled fields in enabled group including high-risk and raw diagnostics
renders disabled fields in common advanced expert groups
renders zh-CN display group labels
keeps enabled group expanded and expert group collapsed
```

测试输入必须同时包含：

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

期望：

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

中文 label 测试要显式创建 zh-CN i18n，不能依赖全局空 messages。

## Phase 4 — 新增 ConfigField enabled/value 测试

新增：

```text
web/src/components/config/__tests__/ConfigField.enabled-state.test.ts
```

测试：

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

## Phase 5 — 新增 ConfigEdit 专项静态边界测试

新增：

```text
web/tests/configEditRegressionBoundary.test.mjs
```

并修改：

```text
web/package.json
```

把它加入 `npm test`。

测试要求：

```text
1. displayGroupForField 中 enabledAtLoad 判断在 isExpertField 之前
2. ConfigEditView / ConfigSection / ConfigField 稳定 selector 存在
3. ConfigField 不因 field.enabled=false 禁用 value 控件
4. BackendRuntimesPage / RunnerConfigsPage / ModelDeploymentsPage 使用 configEditViewLevelOptions(t)
5. 三个页面使用 configEditViewLevelHelp(t)
6. 三个页面不硬编码 Normal / Advanced / Developer
7. 三个页面切换 configViewLevel 时重新 getConfigEditView 或有明确等价逻辑
8. raw diagnostics 只在 developer 模式显示
9. RuntimeParameterEditor / HumanRuntimeParameterForm 不在活动页面使用
```

如果静态规则因为变量名不同过脆，可以先抽象 helper 或添加真实逻辑旁的注释标记，但不能用注释替代真实 reload/patch 清空逻辑。

## Phase 6 — 风险标识增强

如果当前 ConfigField 没有风险可见性，请至少增加 DOM metadata：

```text
:data-field-tier="field.tier || ''"
:data-field-view="field.view || ''"
:data-field-risk="field.risk || ''"
:data-field-diagnostic="field.diagnostic ? 'true' : 'false'"
```

如果 `ConfigEditField` 类型缺少 `risk?: string`，请补上。

推荐进一步显示 tag：

```text
risk=high -> 高风险 / High risk
tier=expert 或 view=developer/security -> 专家 / Expert
diagnostic=true -> 诊断 / Diagnostic
```

新增 i18n key：

```text
configEdit.badges.highRisk
configEdit.badges.expert
configEdit.badges.diagnostic
```

## Phase 7 — 文档 closeout

新增：

```text
docs/reports/phase-3/configedit-regression-engineering/configedit-regression-engineering-closeout.md
```

必须写：

```text
Root cause / testing gap
Changed files
New/updated tests
What each test prevents
Verification commands
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

## 验证命令

必须全部执行并 PASS：

```bash
go test ./...

cd web
npm run test:unit
npm test
npm run build
```

## 最终输出

请输出：

```text
1. 修改文件
2. 新增/修改测试名
3. 每个测试防什么回归
4. 测试命令和结果
5. commit id
6. push result
7. git status --short
8. 是否还有未解决问题
```

完成后 commit 并 push 到当前分支。
