# Runtime Config Rendering and Wizard Actions Closeout

Date: 2026-06-30

## Root Cause

`/api/v1/config-edit/view` already powered BackendRuntime, NodeBackendRuntime, and deployment override editing, but the projected field contract was not self-contained enough. Catalog/default argument metadata, semantic defaults, and ConfigSet schema data carried labels/help/options in several places, while projection and frontend rendering did not consistently preserve or consume that metadata. When translations were missing, `ConfigField` fell through to the generic `configEdit.labels.default` label, so structured parameters rendered as `配置项`.

Long-form wizard actions were also split between dialog footer, step content, and local wizard footer implementations. That made primary actions hard to reach after scrolling and caused inconsistent cancel/previous/next/save behavior across NodeBackendRuntime, deployment, and model artifact workflows.

## Modified Files

- Backend self-contained config metadata:
  - `internal/server/catalog/types.go`
  - `internal/server/catalog/loader.go`
  - `internal/server/configedit/types.go`
  - `internal/server/configedit/project.go`
  - `internal/server/configedit/configset_adapter.go`
  - `internal/server/configedit/taxonomy.go`
  - `internal/server/semanticconfig/registry.go`
- Backend/API tests:
  - `internal/server/api/config_edit_handlers_test.go`
  - `internal/server/catalog/loader_test.go`
  - `internal/server/configedit/configedit_test.go`
  - `internal/server/semanticconfig/registry_normalizer_test.go`
- Frontend config rendering:
  - `web/src/utils/configEditView.ts`
  - `web/src/utils/configEditFieldMeta.ts`
  - `web/src/components/config/ConfigField.vue`
  - `web/src/components/config/ConfigEditView.vue`
  - `web/src/locales/zh-CN.ts`
  - `web/src/locales/en-US.ts`
- Frontend wizard actions:
  - `web/src/components/common/WizardActionBar.vue`
  - `web/src/components/deployments/NodeRuntimeConfigWizard.vue`
  - `web/src/components/deployments/DeploymentWizard.vue`
  - `web/src/pages/RunnerConfigsPage.vue`
  - `web/src/pages/ModelDeploymentsPage.vue`
  - `web/src/pages/ModelArtifactsPage.vue`
- Frontend tests:
  - `web/src/utils/__tests__/configEditFieldMeta.test.ts`
  - `web/src/components/config/__tests__/ConfigEditView.render.test.ts`
  - `web/src/components/common/__tests__/WizardActionBar.test.ts`
  - `web/src/pages/__tests__/RunnerConfigsPage.integration.test.ts`
  - `web/src/pages/__tests__/ModelDeploymentsPage.integration.test.ts`
  - `web/tests/runtimeBoundaryUi.test.mjs`

## What Changed

- Config edit projection now emits field-level self-contained metadata: stable key, internal key, patch path, value/default/effective/source details, enabled state, type/render, label/help/tooltip i18n keys and text, placeholder, required/readonly/disabled/sensitive flags, options/constraints/validation rules, section/order/basic/advanced, and next-layer copy/override/disable/patch target behavior.
- Catalog materialization preserves default argument schema metadata instead of reducing it to key/type/default.
- Semantic registry remains a materialization metadata source only; pages and the frontend resolver do not branch on backend type or maintain vLLM/SGLang-specific business dictionaries.
- `ConfigField` now resolves labels/help/tooltips from field metadata through a shared resolver. Generic labels such as `配置项` are treated as missing metadata, then humanized diagnostic fallback is used.
- Required labels and descriptions were added to zh-CN/en-US locales for current runtime fields while preserving stable parameter keys.
- `WizardActionBar` centralizes sticky wizard actions with cancel, previous, primary, secondary actions, loading state, disabled reason, and wrap-friendly layout.
- NodeBackendRuntime, deployment, and model artifact wizards now use the shared action bar for their main wizard controls.

## Tests Run

- `go test ./internal/server/configedit ./internal/server/catalog ./internal/server/semanticconfig`
  - Result: PASS
- `go test ./...`
  - Result: PASS
- `cd web && npm test`
  - Result: PASS
- `cd web && npm run test:unit`
  - Result: PASS
- `cd web && npm run build`
  - Result: PASS

## API / Manual Evidence

- `internal/server/api/config_edit_handlers_test.go::TestConfigEditViewAPIProjectsRuntimeWithoutInternalOrdinaryLabels` now verifies that the config-edit API projection includes structured fields and self-contained metadata such as `label_i18n_key`, `description_i18n_key`, `default_value`, `patch_target`, and `copy_behavior`.
- Manual API check command for a running local server:

```bash
curl -s -X POST http://localhost:18080/api/v1/config-edit/view \
  -H 'Content-Type: application/json' \
  -d '{"object_kind":"backend_runtime","object_id":"runtime.vllm.nvidia-docker","layer":"node_backend_runtime","mode":"enable"}'
```

Expected evidence: each field carries stable `key`, `internal_key`, `path`, field metadata, source/effective/default state, and next-layer patch behavior without the page knowing the backend type.

## Problem Closure

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| RC-001 | Structured config fields rendered as `配置项` | ConfigField fell through to generic default label when metadata/i18n was incomplete | Node runtime image/parameter step hid actual parameter names | FIXED | Backend config projection and `ConfigField` resolver | `go test ./...`, `cd web && npm test`, `cd web && npm run build` | Field metadata is self-contained and generic fallback is diagnostic only |
| RC-002 | Catalog/default arg metadata was partially lost during materialization/projection | Arg label/help/options/validation were not consistently copied into ConfigSet/EditView | Frontend had insufficient metadata to render unknown complete fields | FIXED | `internal/server/catalog/loader.go`, `internal/server/configedit/project.go` | Catalog/configedit tests | Config edit view carries preserved label/help/options/validation metadata |
| RC-003 | Long-form wizard actions were split across footer/content/header locations | NBR, deployment, and model artifact wizards had local action implementations | Hard to operate long forms and inconsistent loading/disabled behavior | FIXED | `WizardActionBar.vue` and wizard/page replacements | Frontend unit/source tests and build | Main wizard controls are unified through `WizardActionBar` |
| RC-004 | Vite build emitted Rollup annotation and chunk-size warnings | `npm run build` completed with exit code 0 and warnings from dependency/chunk sizing | No failing verification or task-specific regression | INVALID | N/A | `cd web && npm run build` exit code 0 | Not a task blocker; no code change required |

No unresolved problems remain outside this document. No problems exist only in chat.

## Commit / Push

- Commit id: reported in the final response after the repository commit is created.
- Push result: reported in the final response after `git push` completes.
