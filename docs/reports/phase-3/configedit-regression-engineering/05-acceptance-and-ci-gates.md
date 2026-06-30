# 验收标准与 CI 门禁

## 1. 功能验收

### 1.1 enabled 分组

必须满足：

```text
enabled=true 的任何字段都在 已启用参数：
- normal
- advanced
- expert
- high-risk/security
- raw
- diagnostic
```

必须满足：

```text
enabled=false 的字段按原始性质分组：
- normal -> 常用参数
- advanced -> 高级参数
- expert/security/high-risk/raw/diagnostic -> 专家参数
```

### 1.2 编辑中稳定性

必须满足：

```text
1. original_enabled=false 的字段，编辑中勾选后不跳到已启用参数。
2. original_enabled=true 的字段，编辑中取消勾选后不离开已启用参数。
3. 保存刷新后按新的 original_enabled 重新分组。
```

### 1.3 风险可见性

必须满足：

```text
enabled high-risk 参数进入已启用参数。
risk/tier/view/section/diagnostic 元数据不丢失。
```

推荐满足：

```text
UI 显示 高风险 / 专家 / 诊断 tag。
```

### 1.4 enabled/value 分离

必须满足：

```text
1. enabled=false 不禁用 value 控件。
2. 取消 enabled 不清空 value。
3. patch 同时保留 value 和 enabled。
4. readonly 才禁用 checkbox/value。
5. required 字段不可禁用。
```

### 1.5 raw JSON

必须满足：

```text
1. 结构化字段不显示父对象 raw JSON。
2. raw_json 只用于专家/诊断字段或明确 diagnostics 区域。
3. mount/health/env/docker 子字段保持结构化 widget。
```

### 1.6 i18n

必须满足：

```text
ConfigEdit 普通 UI 不显示 raw English section label。
display group 显示：
- 已启用参数
- 常用参数
- 高级参数
- 专家参数
```

## 2. 测试文件验收

必须新增或修改：

```text
web/src/utils/__tests__/configEditView.test.ts
web/src/components/config/__tests__/ConfigEditView.render.test.ts
web/src/components/config/__tests__/ConfigField.enabled-state.test.ts
web/tests/configEditRegressionBoundary.test.mjs
web/package.json
```

如实现风险 tag，则还应修改：

```text
web/src/components/config/ConfigField.vue
web/src/locales/zh-CN.ts
web/src/locales/en-US.ts
```

## 3. 测试命令验收

必须全部通过：

```bash
go test ./...

cd web
npm run test:unit
npm test
npm run build
```

如果新增 e2e：

```bash
cd web
npm run test:e2e -- --grep "ConfigEdit"
```

## 4. CI 门禁建议

### 4.1 npm test 必须包含 ConfigEdit 专项边界测试

`web/package.json` 的 `test` script 必须包含：

```text
node tests/configEditRegressionBoundary.test.mjs
```

### 4.2 static boundary 不允许过度依赖页面文案

静态测试用于防架构退化，不用于替代组件测试。

允许静态检查：

```text
1. 是否导入 ConfigEditView
2. 是否使用 shared display options
3. 是否存在稳定 selector
4. 是否使用旧 RuntimeParameterEditor
5. displayGroupForField 判断顺序
```

不建议静态检查：

```text
1. 完整 HTML 顺序
2. 每个字段的最终 DOM 位置
3. Element Plus 组件内部结构
```

### 4.3 组件测试不应依赖 Element Plus 内部 class

优先使用：

```text
data-testid
data-section-key
data-field-key
data-internal-key
```

少用：

```text
.el-collapse-item
.is-active
```

如果必须用，应限制在折叠行为测试中。

## 5. 禁止通过方式

不允许：

```text
1. 只改测试，不改 displayGroupForField 错误顺序。
2. 通过隐藏 high-risk 字段让测试通过。
3. 通过把 high-risk enabled 保留在 expert 来规避风险显示。
4. 在页面里复制一套分组逻辑。
5. 删除 ConfigEdit Templates 或 ConfigEditView 测试。
6. 把 value 控件因为 field.enabled=false 禁用。
7. closeout 留 TBD/TODO/pending/recorded later。
```

## 6. 最终输出要求

Codex 完成后必须输出：

```text
1. Root cause / testing gap
2. 修改文件
3. 新增/修改测试名
4. 每个测试防止什么回归
5. 测试命令和 PASS 结果
6. commit id
7. push result
8. git status --short
9. 是否还有未解决问题
```

## 7. 推荐最终状态

如果全部完成：

```text
CONFIGEDIT_REGRESSION_ENGINEERING_PHASE_1_CLOSED
Final status: PASS
Unresolved problems: none
```

如果没有做 Playwright：

```text
Playwright ConfigEdit smoke: deferred, not required for Phase 1
```

这个 deferred 必须是明确不属于 Phase 1，而不是功能问题遗留。
