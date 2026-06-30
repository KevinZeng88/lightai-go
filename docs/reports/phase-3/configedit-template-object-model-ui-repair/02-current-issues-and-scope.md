# Current Issues and Repair Scope

## 1. Observed issue: ConfigEdit Templates page is empty

Observed from the exported page state:

```text
Main menu: ConfigEdit Templates
Page title: ConfigEdit Templates
Button: Refresh
Table headers: Template / Backend / Source
Table body: 暂无数据
Right panel: Select a template
```

This is a product and architecture issue. The page appears to exist, but it is not populated with the ConfigEdit template registry.

## 2. Observed issue: English leaks in zh-CN UI

The following strings currently leak:

```text
ConfigEdit Templates
Refresh
Template
Backend
Source
Select a template
Normal
Advanced
Developer
```

Required Chinese labels:

```text
ConfigEdit Templates -> 参数模板 or 配置模板
Refresh -> 刷新
Template -> 模板
Backend -> 后端
Source -> 来源
Select a template -> 请选择一个参数模板
Normal -> 常用
Advanced -> 高级
Developer -> 专家
```

Use the product wording consistently. Recommended default: `参数模板`.

## 3. Observed issue: Normal / Advanced / Developer is unclear

The current labels do not explain their product semantics. Codex must not interpret them as separate runtime modes. They are display-level filters:

```text
常用 = normal fields only
高级 = normal + advanced fields
专家 = normal + advanced + developer fields
```

The UI should include tooltip/help text explaining this.

Suggested explanation:

```text
常用参数适合日常部署；高级参数用于性能、资源和兼容性调优；专家参数包含底层运行、Docker、安全和诊断选项，请谨慎修改。
```

## 4. Observed issue: parameter order is not user-oriented

Users should see the important and active configuration first. The order should be shared across ConfigEdit consumers:

```text
1. 已启用参数
2. 常用参数
3. 高级参数
4. 专家参数
```

Within each group, fields should sort by section and display_order.

Suggested section order:

```text
model
runtime
resource
service
health
mount
env
docker
security
raw
```

The exact enum names may differ in code. Preserve the existing code style and names where reasonable.

## 5. Observed issue: enabled/checked parameters need stable front placement

Required behavior:

- On initial page load, fields with `enabled=true` appear in `已启用参数`.
- While the user is editing, checking or unchecking a field must not immediately move the row/card. Avoid UI jumps.
- After save and reload, `enabled=true` fields appear in `已启用参数`.
- After save and reload, `enabled=false` fields return to their original tier/section position.

This behavior should apply to runtime templates, node backend runtimes, deployments, and other ConfigEdit consumers.

## 6. Repair scope

Codex should inspect and repair all relevant code paths, likely including but not limited to:

```text
web/src/pages/*ConfigEdit*.*
web/src/pages/*Runtime*.*
web/src/components/*ConfigEdit*.*
web/src/components/*RuntimeParameter*.*
web/src/i18n/**
internal/server/**configedit**
internal/server/**runtime**
internal/server/**backend**
internal/**/catalog**
docs/reports/phase-3/configedit-template-object-model-design/**
```

Do not assume these exact paths are complete. Use repository search.

## 7. Non-goals

- Do not remove the ConfigEdit Templates feature to make the page disappear.
- Do not solve the empty page by hard-coding dummy rows in the frontend.
- Do not rely on page-specific whitelists for parameter ordering.
- Do not expose raw JSON as the ordinary display for known or materializable fields.
- Do not add historical compatibility for obsolete snapshots.
