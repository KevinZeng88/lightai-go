# Codex Autonomous Execution Prompt

You are working in the LightAI Go repository:

```bash
/home/kzeng/projects/ai-platform-study/lightai-go
```

Execute a focused repair for ConfigEdit Templates and ConfigEdit parameter presentation.

## Background

LightAI Go has a ConfigEdit template object model. This is a real architecture component, not a disposable debug page. Its purpose is to ensure runtime parameters, backend args, Docker launcher options, health checks, mounts, env, resource controls, and fallback-materialized fields are represented as structured template fields and rendered through shared ConfigEdit/RuntimeParameterEditor UI logic.

The design intent is:

```text
Catalog / BackendVersion / BackendRuntime / NBR / Deployment data
        -> ConfigEdit template registry and materialization
        -> ConfigEditTemplate objects with fields, sections, tiers, labels, source metadata
        -> reusable ConfigEdit / RuntimeParameterEditor projection
        -> runtime template, NBR, deployment, and template inspection pages
```

The existing closeout/design document to read first:

```bash
docs/reports/phase-3/configedit-template-object-model-design/07-unified-configedit-parameter-handling-closeout.md
```

## Current symptoms

From the current UI export:

```text
Menu: ConfigEdit Templates
Title: ConfigEdit Templates
Button: Refresh
Headers: Template / Backend / Source
Body: 暂无数据
Right panel: Select a template
Runtime template level buttons: Normal / Advanced / Developer
```

Problems:

1. ConfigEdit Templates page is empty even though ConfigEdit parameter templates/materialized fields should exist.
2. zh-CN UI leaks English strings.
3. Normal / Advanced / Developer is unclear and should be productized as display levels.
4. Parameter ordering should put active and common fields before advanced/expert fields.
5. Enabled/checked fields should appear first on the next page load, but toggling during editing must not cause the row/card to jump.

## Required behavior

### ConfigEdit Templates page

Fix the data source. The page must list real templates from the ConfigEdit template registry/materialization source. It should show template identity, backend/scope, and source. Selecting a template should show field details when supported by the existing UI.

Expected template categories when present:

```text
vLLM backend args
SGLang backend args
llama.cpp backend args
Docker launcher options
resource controls
health check
mount/volume mapping
environment variables
fallback-materialized fields
```

Do not fix this with dummy frontend data. Do not hide the page to avoid fixing the model/API.

### i18n

Use zh-CN labels:

```text
ConfigEdit Templates -> 参数模板
Refresh -> 刷新
Template -> 模板
Backend -> 后端
Source -> 来源
Select a template -> 请选择一个参数模板
Normal -> 常用
Advanced -> 高级
Developer -> 专家
Enabled parameters -> 已启用参数
Common parameters -> 常用参数
Advanced parameters -> 高级参数
Expert parameters -> 专家参数
```

### View levels

Treat the level selector as a hierarchical display filter:

```text
常用 = normal fields only
高级 = normal + advanced fields
专家 = normal + advanced + developer fields
```

Add tooltip/help text explaining:

```text
常用参数适合日常部署；高级参数用于性能、资源和兼容性调优；专家参数包含底层运行、Docker、安全和诊断选项，请谨慎修改。
```

### Grouping and sorting

Implement shared sorting/grouping for ConfigEdit consumers:

```text
1. 已启用参数
2. 常用参数
3. 高级参数
4. 专家参数
```

Within each group, sort by section, display_order, then field key/path.

Recommended section order:

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

High-risk Docker/security fields such as `privileged`, `security_options`, `cap_add`, and `cap_drop` should appear in expert/security/high-risk grouping.

### Enabled/checked placement

Required behavior:

```text
Initial page load: enabled=true fields are grouped under 已启用参数.
During editing: checking/unchecking does not immediately reorder fields.
After save/reload: enabled=true fields move to 已启用参数.
After save/reload: enabled=false fields return to original tier/section group.
```

A safe implementation is to compute a load-time grouping snapshot and keep it stable during the edit session. Recompute group placement only after save/reload.

### Raw JSON

Raw JSON must be limited to expert/diagnostic view or explicit raw section. Known or materializable fields must render as structured ConfigEdit fields.

## Execution phases

### Phase 0 — Baseline

Run:

```bash
git status --short
git log --oneline -10
sed -n '1,220p' docs/reports/phase-3/configedit-template-object-model-design/07-unified-configedit-parameter-handling-closeout.md || true
find docs/reports/phase-3 -iname '*configedit*' -o -iname '*parameter*' | sort
grep -R "ConfigEdit" -n internal web docs | head -200
grep -R "Normal\|Advanced\|Developer\|ConfigEdit Templates\|Select a template" -n web internal | head -200
```

Produce a short baseline note under:

```text
docs/reports/phase-3/configedit-template-object-model-ui-repair/phase-0-baseline-findings.md
```

Include route, component, endpoint, current data source, cause of empty list, location of level labels, and test gaps.

### Phase 1 — Object model contract

Confirm or extend existing types/DTOs so fields carry tier, section, source, order, labels/help keys, input/value type, risk, and enabled state where values are attached.

Document final contract in:

```text
docs/reports/phase-3/configedit-template-object-model-ui-repair/phase-1-object-model-contract.md
```

### Phase 2 — Template list data source

Fix the page/API so ConfigEdit Templates lists real registry/materialized templates. Document root cause and example response in:

```text
docs/reports/phase-3/configedit-template-object-model-ui-repair/phase-2-template-list-data-source.md
```

### Phase 3 — i18n/product copy

Localize all listed UI strings and add level tooltip/help text. Document changed keys in:

```text
docs/reports/phase-3/configedit-template-object-model-ui-repair/phase-3-i18n-and-copy.md
```

### Phase 4 — View-level semantics

Implement hierarchical filtering and shared behavior across ConfigEdit consumers. Document in:

```text
docs/reports/phase-3/configedit-template-object-model-ui-repair/phase-4-view-level-semantics.md
```

### Phase 5 — Grouping/sorting

Implement shared grouping/sorting and apply it to runtime templates, node backend runtimes, deployments, and other ConfigEdit/RuntimeParameterEditor consumers. Document in:

```text
docs/reports/phase-3/configedit-template-object-model-ui-repair/phase-5-grouping-and-ordering.md
```

### Phase 6 — Enabled placement stability

Implement load-time enabled-first grouping without live reorder on toggle. Document in:

```text
docs/reports/phase-3/configedit-template-object-model-ui-repair/phase-6-enabled-placement.md
```

### Phase 7 — Tests and closeout

Add/update tests for:

```text
ConfigEdit Templates list uses real data
zh-CN labels do not leak listed English strings
常用/高级/专家 hierarchical filtering
enabled fields load first
checking/unchecking does not immediately reorder
unchecked fields return to original group after reload
raw JSON only in expert/diagnostic view
```

Run:

```bash
go test ./...
cd web
npm test
npm run test:unit || true
npm run build
```

Use actual project scripts if these differ. Do not ignore required failures.

Write closeout:

```text
docs/reports/phase-3/configedit-template-object-model-ui-repair/configedit-template-object-model-ui-repair-closeout.md
```

Closeout must include:

```text
root cause
implementation summary
changed files
test commands and results
evidence paths
commit id
push result
git status --short
```

Commit and push:

```bash
git status --short
git add <changed-files>
git commit -m "fix(configedit): repair template registry ui and parameter grouping"
git push
git status --short
```

## Restrictions

- Work on the current branch unless explicitly instructed otherwise.
- Do not add legacy compatibility logic for old snapshots or old schemas.
- Do not hide ConfigEdit Templates instead of fixing its data source.
- Do not add fake frontend template rows.
- Do not implement sorting only in a single page.
- Do not downgrade known/materializable parameters to raw JSON-only.
- Do not leave unresolved issues outside the closeout/open-issue documentation.
