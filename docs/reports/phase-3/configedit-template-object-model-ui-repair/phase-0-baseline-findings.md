# Phase 0 Baseline Findings

Date: 2026-07-01

## Commands Run

```bash
git status --short
git log --oneline -10
sed -n '1,220p' docs/reports/phase-3/configedit-template-object-model-design/07-unified-configedit-parameter-handling-closeout.md || true
find docs/reports/phase-3 -iname '*configedit*' -o -iname '*parameter*' | sort
grep -R "ConfigEdit" -n internal web docs | head -200
grep -R "Normal\|Advanced\|Developer\|ConfigEdit Templates\|Select a template" -n web internal | head -200
```

## Current Route And Component

- Route: `/config-edit/templates`
- Router entry: `web/src/router/index.ts`
- Main component: `web/src/pages/ConfigEditTemplatesPage.vue`
- Navigation label: hard-coded in `web/src/layouts/ConsoleLayout.vue` as `ConfigEdit Templates`

## Current API Endpoint

- List: `GET /api/v1/config-edit/templates`
- Get: `GET /api/v1/config-edit/templates/{id}`
- Validate: `POST /api/v1/config-edit/templates/validate`
- Clone: `POST /api/v1/config-edit/templates/{id}/clone`
- Backend handler: `internal/server/api/configedit_template_handlers.go`

## Current Data Source

The handler currently loads only YAML component templates from:

```text
configs/configedit-templates/builtin
configs/configedit-templates/local
```

It does not include catalog/materialized ConfigEdit fields from BackendVersion/BackendRuntime ConfigSets.

## Cause Of Empty Or Incomplete List

The page depends only on explicit component-template YAML files. That makes the page empty or incomplete whenever explicit YAML templates are missing, invalid, or not exhaustive, even though the runtime catalog can materialize real ConfigEdit fields for vLLM, SGLang, llama.cpp, Docker options, env, mounts, health checks, and fallback fields.

The page also does not surface template-load issues in a productized way.

## Level Labels

Hard-coded English labels are in:

- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/pages/RunnerConfigsPage.vue`
- `web/src/pages/ModelDeploymentsPage.vue`
- `web/src/pages/ConfigEditTemplatesPage.vue`
- `web/src/layouts/ConsoleLayout.vue`

Current labels:

```text
Normal / Advanced / Developer
ConfigEdit Templates / Refresh / Template / Backend / Source / Select a template
```

## Test Gaps

- No backend/API test proves `GET /api/v1/config-edit/templates` returns registry plus materialized templates.
- No frontend test proves ConfigEdit Templates uses real API data and renders non-empty rows.
- No focused test proves zh-CN does not leak the requested English strings.
- No shared frontend test covers hierarchical view levels.
- No shared frontend test covers load-time enabled-first grouping with stable placement during editing.

