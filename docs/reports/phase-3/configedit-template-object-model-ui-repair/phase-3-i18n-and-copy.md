# Phase 3 - i18n and Product Copy

Date: 2026-07-01

## Changed Keys

Added zh-CN/en-US keys under:

- `configEdit.templates.*`
- `configEdit.levels.*`
- `configEdit.sections.enabledParameters`
- `configEdit.sections.commonParameters`
- `configEdit.sections.advancedParameterGroup`
- `configEdit.sections.expertParametersGroup`
- `configEdit.sections.securityHighRisk`

## UI Changes

Localized:

- ConfigEdit Templates -> 参数模板
- Refresh -> 刷新
- Template / Backend / Source -> 模板 / 后端 / 来源
- Select a template -> 请选择一个参数模板
- Normal / Advanced / Developer -> 常用 / 高级 / 专家
- Enabled/Common/Advanced/Expert parameters -> 已启用参数 / 常用参数 / 高级参数 / 专家参数

Added shared display-level help text:

常用参数适合日常部署；高级参数用于性能、资源和兼容性调优；专家参数包含底层运行、Docker、安全和诊断选项，请谨慎修改。

## Evidence

`web/tests/runtimeBoundaryUi.test.mjs` checks the template page, side navigation, runtime pages, and shared display helper do not retain the listed hardcoded English labels.

