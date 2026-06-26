# 03 — Workstream A: Naming Debt

## Goal

Make product concepts unambiguous in code-facing docs and user-facing UI.

## Step A1 — Inventory

Run:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
mkdir -p docs/reports/product-hardening-20260626/execution
rg -n "RunnerConfig|runner-config|runnerConfigs|BackendRuntime|backend runtime|runtime template|NodeBackendRuntime|node_backend_runtime|RunPlan|run plan|ConfigSet|config_set|deployment|instance" web internal docs configs > docs/reports/product-hardening-20260626/execution/naming-rg.txt
```

Create:

```text
docs/reports/product-hardening-20260626/execution/naming-inventory.md
```

Required columns:

| Path | Current text | Category | Target text | Action |
| --- | --- | --- | --- | --- |

Categories:

- user-facing UI;
- i18n key;
- internal code;
- API field;
- docs;
- tests;
- historical evidence.

## Step A2 — Create dictionary

Create:

```text
docs/engineering/naming-dictionary.md
```

Required content:

```text
Backend = 推理后端 / Backend
BackendVersion = 后端版本 / Backend Version
BackendRuntime = 运行模板 / Runtime Template
NodeBackendRuntime = 节点运行配置 / Node Runtime Config
ModelArtifact = 模型 / Model
ModelLocation = 模型位置 / Model Location
ModelDeployment = 模型部署 / Deployment
ModelInstance = 模型实例 / Instance
ResolvedRunPlan = 运行计划 / Run Plan
ConfigSet = 配置集 / ConfigSet
```

For each concept, document:

- owner layer;
- user-editability;
- whether it is copied to next layer;
- whether changes affect existing child objects;
- preferred UI label;
- allowed technical abbreviation;
- forbidden stale terms.

## Step A3 — Update frontend labels

Inspect and update:

```text
web/src/router/index.ts
web/src/layouts/ConsoleLayout.vue
web/src/pages/BackendRuntimesPage.vue
web/src/pages/RunnerConfigsPage.vue
web/src/pages/ModelDeploymentsPage.vue
web/src/pages/ModelInstancesPage.vue
web/src/locales/zh-CN.*
web/src/locales/en-US.*
```

Concrete target:

- `/runtimes` page title = `运行模板` / `Runtime Templates`.
- `/runner-configs` page title = `节点运行配置` / `Node Runtime Configs`.
- Deployment runtime selector label = `节点运行配置`, not generic `runtime`.
- BackendRuntime table column should say template/source backend/version/vendor/image.
- NBR table column should say node/template/image/status/last check/deployable.
- RunPlan panels should say `运行计划`, not generic diagnostic JSON.
- ConfigSet should appear only as technical JSON section title.

## Step A4 — Update docs

Update only current docs, not historical evidence unless the evidence is still active.

Suggested docs:

```text
README.md
docs/engineering/naming-dictionary.md
docs/design/catalog-configset-and-runtime-snapshot.md
docs/design/runtime-parameter-system/01-parameter-layering-design.md
docs/design/runtime-operations-ux-resource-controls.md
docs/api/openapi.yaml
```

## Step A5 — Add tests

Add or update frontend tests:

```text
web/tests/i18nMissingKeys.test.mjs
web/tests/i18nKeys.test.mjs
web/tests/runtimeBoundaryUi.test.mjs
web/tests/namingDictionary.test.mjs
```

Test requirements:

- no raw route title keys are rendered;
- required zh-CN/en-US labels exist;
- stale user-facing `RunnerConfig` is absent unless explicitly allowed;
- `Node Runtime Config` and `Runtime Template` are distinct;
- deployment page labels do not call NBR merely "runtime";
- model page labels do not imply runtime parameter ownership.

## Acceptance

- `docs/engineering/naming-dictionary.md` exists.
- UI labels match target vocabulary.
- Tests enforce vocabulary.
- Docs and OpenAPI descriptions use the same terms.
- `cd web && npm test` passes.
- `cd web && npm run build` passes.
