# Executable Repair Plan

## Phase 0 — Baseline and repository understanding

### Goal

Confirm the current implementation of the ConfigEdit template object model before modifying code.

### Steps

1. Confirm current git state:

```bash
git status --short
git log --oneline -10
```

2. Read the existing closeout/design documents:

```bash
sed -n '1,220p' docs/reports/phase-3/configedit-template-object-model-design/07-unified-configedit-parameter-handling-closeout.md || true
find docs/reports/phase-3 -iname '*configedit*' -o -iname '*parameter*' | sort
```

3. Search for ConfigEdit template code paths:

```bash
grep -R "ConfigEdit" -n internal web docs | head -200
grep -R "Normal\|Advanced\|Developer\|configedit" -n web internal | head -200
grep -R "ConfigEdit Templates\|Select a template\|Template\|Source" -n web internal | head -200
```

4. Identify:

- frontend page/component responsible for ConfigEdit Templates
- API endpoint used by the page
- backend handler/service for listing templates
- registry/catalog/materialization code producing ConfigEdit fields
- existing i18n keys for this feature
- current field tier/section/order/enabled representation

### Required phase output

Create or update a short working note under the target report directory:

```text
phase-0-baseline-findings.md
```

It must include:

- current page route and component
- current API endpoint
- current backend source or missing source
- reason the list is empty
- where Normal/Advanced/Developer is defined
- current test coverage gaps

Do not proceed to implementation until this is clear from code evidence.

## Phase 1 — Define/confirm the shared object model contract

### Goal

Make the current code contract explicit so frontend and backend agree on template fields, tier, section, source, and enabled state.

### Steps

1. Locate existing DTOs/types. Prefer extending current types over creating parallel types.
2. Ensure each template or field can carry the following conceptual properties, using existing names where possible:

```text
field key/path
display label or label_i18n_key
help text or help_i18n_key
section
tier/view level
risk level
display order
source/source kind
value type/input type
enabled state when values are attached
```

3. If backend DTOs lack needed fields, add them cleanly.
4. If frontend types lack needed fields, align them with backend DTOs.
5. Document the contract in code comments or report notes.

### Required phase output

```text
phase-1-object-model-contract.md
```

Include:

- final DTO/type names
- tier enum values and Chinese labels
- section ordering
- source values
- value/enabled representation

## Phase 2 — Fix ConfigEdit Templates data source

### Goal

The ConfigEdit Templates page must list real templates from the object model registry/materialization source.

### Steps

1. Trace the frontend request when the page loads.
2. Trace the backend handler used by that request.
3. Connect the handler to the real ConfigEdit registry/catalog/materialization source.
4. Ensure at least the following template categories are visible when present in the project catalog:

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

5. If templates are backend/version specific, show backend information.
6. If templates are scope specific, show scope information.
7. If templates are fallback materialized, show `fallback_materialized` or equivalent source.
8. The page must not show fake data. It must show registry data or a clear actionable error state.

### Required phase output

```text
phase-2-template-list-data-source.md
```

Include:

- root cause of empty page
- endpoint fixed or added
- data source connected
- example API response summary
- screenshots or API output path if available

## Phase 3 — i18n and product wording

### Goal

Remove English leakage from the ConfigEdit Templates page and runtime parameter level controls.

### Steps

1. Add/complete zh-CN and en-US i18n keys for:

```text
参数模板 / ConfigEdit Templates
刷新 / Refresh
模板 / Template
后端 / Backend
来源 / Source
请选择一个参数模板 / Select a template
常用 / Normal
高级 / Advanced
专家 / Developer
已启用参数 / Enabled parameters
常用参数 / Common parameters
高级参数 / Advanced parameters
专家参数 / Expert parameters
```

2. Update components/pages to use i18n keys.
3. Add tooltip/help text for display levels.
4. Run existing i18n audit. If there is no specific audit command, add or extend tests to catch these strings in zh-CN mode.

### Required phase output

```text
phase-3-i18n-and-copy.md
```

Include changed keys and UI locations.

## Phase 4 — Implement view-level semantics

### Goal

Make Normal/Advanced/Developer behave as hierarchical display filters.

### Required behavior

```text
常用: tier normal
高级: tier normal + advanced
专家: tier normal + advanced + developer
```

### Steps

1. Locate current filtering logic.
2. Replace any ambiguous or independent filter behavior with hierarchical filtering.
3. Keep current selected view level stable across local component state as currently designed. If there is existing persistence, preserve it.
4. Add tooltip/help text near the control.
5. Verify all ConfigEdit consumers use the same filtering utility or equivalent shared logic.

### Required phase output

```text
phase-4-view-level-semantics.md
```

Include files changed and test names.

## Phase 5 — Implement shared parameter grouping and ordering

### Goal

Make enabled/common/advanced/expert ordering shared across ConfigEdit consumers.

### Required load-time grouping

```text
1. 已启用参数
2. 常用参数
3. 高级参数
4. 专家参数
```

### Required section order within groups

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

### Steps

1. Implement or update a shared utility for grouping/sorting ConfigEdit fields.
2. Sorting input must use field metadata: enabled, tier, section, display_order, key/path.
3. Use stable deterministic fallback order when metadata is missing.
4. Apply the shared utility to:

```text
runtime templates
node backend runtimes
deployments
other ConfigEdit/RuntimeParameterEditor consumers
```

5. Ensure known high-risk Docker fields appear under expert/security/high-risk grouping:

```text
privileged
security_options
cap_add
cap_drop
```

6. Ensure ordinary backend args and resource controls do not fall into raw JSON when structured metadata exists or can be materialized.

### Required phase output

```text
phase-5-grouping-and-ordering.md
```

Include the shared utility path and all consumers updated.

## Phase 6 — Implement stable enabled/checked placement behavior

### Goal

Enabled parameters should load at the front without causing live editing jumps.

### Required behavior

- On page load, `enabled=true` appears in `已启用参数`.
- During editing, checking/unchecking does not reorder immediately.
- After save and reload, enabled fields move to `已启用参数`.
- After save and reload, disabled fields return to original tier/section.

### Implementation guidance

A safe approach is to compute a load-time grouping snapshot when the editor is initialized or when saved data is reloaded. During local edits, update values/enabled state but do not recompute group membership until save/reload.

Possible conceptual approach:

```text
initialGroupKey = deriveGroupKey(field.enabled, field.tier, field.section) at load time
renderGroupKey = initialGroupKey during current edit session
onSaveSuccess/reload -> recompute initialGroupKey from persisted state
```

Use existing state management conventions in the frontend.

### Required phase output

```text
phase-6-enabled-placement.md
```

Include how live reordering was prevented and how reload behavior was tested.

## Phase 7 — Tests, regression, closeout, commit, push

### Goal

Validate the fix across backend, frontend, and product behavior.

### Steps

1. Run backend tests:

```bash
go test ./...
```

2. Run frontend tests and build:

```bash
cd web
npm test
npm run test:unit || true
npm run build
```

Use the project’s actual test scripts if names differ. Do not ignore failing required tests; update this command list only with evidence.

3. Run or add focused tests for:

```text
ConfigEdit Templates list is backed by real registry data
zh-CN i18n does not leak listed English strings
view level hierarchical filtering
shared parameter grouping/sorting
enabled load-time front placement
no live reorder while editing
unchecked reload returns to original group
raw JSON restricted to expert/diagnostic view
```

4. Commit and push:

```bash
git status --short
git add <changed-files>
git commit -m "fix(configedit): repair template registry ui and parameter grouping"
git push
```

5. Write final closeout:

```text
configedit-template-object-model-ui-repair-closeout.md
```

Required closeout contents:

- root cause
- implementation summary
- changed files
- tests and results
- evidence paths
- commit id
- push result
- final git status
