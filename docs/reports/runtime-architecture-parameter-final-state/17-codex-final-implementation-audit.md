# Codex Final Implementation Audit — Runtime Architecture and ConfigSetBundle Final State

CODEX_FINAL_IMPLEMENTATION_AUDIT_COMPLETED

## 1. Verdict

REJECT_WITH_BLOCKERS

Reason: current `main` contains substantial final-state work and all required local test commands passed, but production RunPlan preparation still includes a flat-shape semantic normalization path, and NBR creation can still persist caller-provided flat `config_set` payloads. Under the audit rules, a production runtime path that still reads old flat `ConfigItem` fields is a blocker.

## 2. Audited HEAD

Commit: `5d8cd37`

Initial `git status --short`: clean.

`git show --stat --oneline HEAD`:

```text
5d8cd37 docs: final closeout and test hygiene cleanup
 .../final-closeout.md                              | 22 +++---
 internal/server/catalog/tiered_roundtrip_test.go   | 84 ++++++++++++----------
 2 files changed, 57 insertions(+), 49 deletions(-)
```

`git show --name-only --oneline HEAD`:

```text
5d8cd37 docs: final closeout and test hygiene cleanup
docs/reports/runtime-architecture-parameter-final-state/final-closeout.md
internal/server/catalog/tiered_roundtrip_test.go
```

## 3. Documents Read

- `00-index.md`
- `09-implementation-plan.md`
- `10-claude-execution-prompt.md`
- `11-final-closeout-template.md`
- `16-codex-second-review.md`
- `final-closeout.md`
- `evidence/batch-0-inventory.txt`
- `evidence/batch-5-e2e-test-results.txt`
- `evidence/final-repair-self-audit-before.txt`
- `evidence/final-repair-self-audit-after.txt`
- `evidence/web-test-output.txt`
- `evidence/web-build-output.txt`

## 4. Commands Run and Results

`git status --short`: clean before report generation.

`git log --oneline -20`: HEAD was `5d8cd37`, with the latest closeout/repair commits present:

```text
5d8cd37 docs: final closeout and test hygiene cleanup
393c891 fix: final repair redo — remove all flat fallbacks, fix tiered value structure
8f3f86e fix: final repair — remove flat fallbacks, fix setConfigValueTiered, strengthen SourceMap
95156ce docs: update final closeout — OI-10 fully resolved
c082d49 feat: OI-10 add node_backend_runtime_id column to model_deployments
05671a5 docs: final closeout — configset-bundle final-state implementation complete
```

Flat fallback grep:

```text
internal/server/api/configset_helpers.go:289:		item["value"] = valueTier
internal/server/configedit/tiered_helpers.go:13:		item["value"] = valueTier
```

Assessment: these two hits are tiered map initialization, not scalar `item["value"]` overwrite.

Legacy web editor grep:

```text
web/tests/runtimeBoundaryUi.test.mjs:60:// RuntimeParameterEditor was intentionally removed (OI-07) — replaced by ConfigEditView.
web/tests/runtimeBoundaryUi.test.mjs:61:// HumanRuntimeParameterForm was intentionally removed — replaced by ConfigEditView.
web/tests/runtimeBoundaryUi.test.mjs:62:check('Backend runtime page no longer imports hardcoded human form', !sources['src/pages/BackendRuntimesPage.vue'].includes('HumanRuntimeParameterForm'))
web/tests/runtimeBoundaryUi.test.mjs:63:check('Node runtime wizard no longer imports hardcoded human form', !sources['src/components/deployments/NodeRuntimeConfigWizard.vue'].includes('HumanRuntimeParameterForm'))
web/tests/runtimeBoundaryUi.test.mjs:71:check('Model deployment page does not import RuntimeParameterEditor', !sources['src/pages/ModelDeploymentsPage.vue'].includes('RuntimeParameterEditor'))
web/tests/runtimeBoundaryUi.test.mjs:79:check('BackendRuntimesPage uses ConfigEditView', sources['src/pages/BackendRuntimesPage.vue'].includes('ConfigEditView') && !sources['src/pages/BackendRuntimesPage.vue'].includes('RuntimeParameterEditor'))
web/tests/runtimeBoundaryUi.test.mjs:81:check('NodeRuntimeConfigWizard uses ConfigEditView', sources['src/components/deployments/NodeRuntimeConfigWizard.vue'].includes('ConfigEditView') && !sources['src/components/deployments/NodeRuntimeConfigWizard.vue'].includes('RuntimeParameterEditor'))
web/tests/runtimeBoundaryUi.test.mjs:123:// h. Config edit scope: RunnerConfigsPage migrated from RuntimeParameterEditor to ConfigEditView.
web/tests/runtimeBoundaryUi.test.mjs:126:check('RunnerConfigsPage does NOT use RuntimeParameterEditor', !rcpSrc.includes('RuntimeParameterEditor'))
```

Assessment: production web code no longer imports the legacy editors; hits are test assertions and comments.

`backend_runtime_id` grep: production deployment create/preview/preflight rejects `backend_runtime_id` as a deployment entry field through `rejectLegacyDeploymentPayload`. Remaining hits are internal DB/source references, NBR enable input, tests, and deployment response/back-reference fields. This does not violate the deployment NBR-only entry rule.

`go test ./... -count=1`: PASS for all packages, including `internal/server/api`, `internal/server/catalog`, `internal/server/configedit`, `internal/server/runplan`, and `internal/server/semanticconfig`.

`cd web && npm test -- --run`: PASS. Output included `Passed: 12, Failed: 0` and `All tests PASSED`.

`cd web && npm run build`: PASS. `vue-tsc --noEmit && vite build` completed; only dependency PURE annotation warnings and chunk-size warnings were emitted.

## 5. Code Reality Checks

### ConfigSetBundle / ConfigItem Tiered Model

Mostly implemented in catalog and ConfigEdit layers. `internal/server/catalog/types.go` defines tiered `ConfigItem` structures, copy-on-create helpers enforce schema/snapshot immutability, and ConfigEdit helpers read `schema`, `value`, and `state` tiers.

### Flat Fallback Removal

Partial. The audited grep did not find the exact old fallback phrases or scalar `item["value"] = value` overwrite. However, `internal/server/semanticconfig/normalizer.go` still consumes flat `ConfigItem` fields directly:

```text
normalizer.go:65 values, _ := item["value"].(map[string]any)
normalizer.go:66 enabledFields, _ := item["enabled_fields"].(map[string]any)
normalizer.go:87 value := item["value"]
normalizer.go:88 defaultValue := item["default_value"]
normalizer.go:92 enabled := boolFromAny(item["enabled"], false)
normalizer.go:93 if boolFromAny(item["required"], false) { ... }
```

This is not merely a test fixture. It is called by `semanticDeploymentSnapshot`, which is used by deployment preview and lifecycle RunPlan paths before `ResolveWithSourceMap`.

### ConfigEdit / API Wiring

ConfigEdit helpers are tiered:

- `configValue` reads `value.effective_value` then `value.default_value`.
- `configItemEnabled` reads `state.enabled`.
- `configItemSchemaField` reads `schema`.
- `defaultValueFromItem` reads `value.default_value`.

Config edit API returns a generated `config_view` when `config_set_json` can unmarshal into `catalog.ConfigSet`.

### RunPlan / SourceMap Integration

`runplan.ResolveWithSourceMap` calls `Resolve`, then assigns `plan.ParameterSourceMap = buildSourceMap(in, plan)`. Deployment preview and lifecycle start paths call `ResolveWithSourceMap`, so the runtime source map is populated when a plan is produced.

Remaining weakness: the shared final-state entry point is implemented as a shared resolver wrapper, while preview/lifecycle still assemble `ResolveInput` in their handlers. This is not the primary blocker, but it should be clarified or tightened after the flat semantic path is removed.

### Docker Subfield Handling

Source map generation includes Docker options such as `docker.shm_size`, `docker.ipc_mode`, `docker.network_mode`, `docker.privileged`, `docker.group_add`, devices, mounts, ports, and health checks.

Blocking caveat: Docker subfield semantic normalization still depends on old `launcher.docker_options.value` plus `enabled_fields` in `semanticconfig.NormalizeConfigSet`. If the final domain model requires Docker subfields as tiered ConfigItems, this path is not final-state compliant.

### Deployment NBR-only Entry

Deployment create, preview, and preflight reject legacy payload keys including `backend_runtime_id`. NBR enable still requires `backend_runtime_id`, which is expected for enabling a BackendRuntime on a node and is not the deployment create/preview entry.

### Web Legacy Cleanup

Production web code no longer references `RuntimeParameterEditor`, `HumanRuntimeParameterForm`, or `runtimeParameterViewModel`; grep hits are only test assertions that enforce removal.

### DB / Catalog Tiered Shape

Catalog and seeded runtime data have tiered model support. The DB schema still stores JSON in `config_set_json`, which is acceptable if the JSON content is tiered. The remaining blocker is that some API paths can still persist caller-provided flat `config_set` data without rejecting or converting it:

```text
internal/server/api/runtime_handlers.go:945 if requestSet := mapFromAny(req["config_set"]); len(requestSet) > 0 {
internal/server/api/runtime_handlers.go:946     set = copyConfigSet(jsonString(requestSet))
internal/server/api/runtime_handlers.go:947 } else if requestSet := mapFromAny(req["config_set_json"]); len(requestSet) > 0 {
internal/server/api/runtime_handlers.go:948     set = copyConfigSet(jsonString(requestSet))
```

`internal/server/api/runtime_boundary_test.go` still creates flat ConfigItems and asserts flat values are preserved, for example `item["value"] != "node-local-value" || item["enabled"] != true`.

### Evidence Consistency

Evidence files are mostly credible and the required fresh verification commands passed. However, `final-closeout.md` materially overstates final-state completion because it says old flat fallbacks and legacy paths were removed while the production semantic normalizer and NBR request replacement path still consume/persist flat-shaped items. The closeout commit list is also stale: it does not include the audited HEAD `5d8cd37`.

## 6. Findings

### Finding 1 — Critical

Severity: Critical

File: `internal/server/semanticconfig/normalizer.go`

Evidence: `NormalizeConfigSet` reads `item["value"]`, `item["default_value"]`, `item["enabled"]`, `item["required"]`, and `launcher.docker_options.enabled_fields` as flat item fields. It is called by `semanticDeploymentSnapshot`, which feeds deployment preview and lifecycle RunPlan resolution.

Impact: Final RunPlan preparation is not tiered-only. A tiered `ConfigItem.value` map can be treated as the runtime value itself, and old flat shape remains a production runtime path.

Required action: Replace semantic normalization with tier-aware reads from `schema`, `value`, `state`, `provenance`, and Docker subfield ConfigItems. Add tests proving tiered values, enabled state, defaults, required rules, and Docker subfields flow through preview/start RunPlan correctly without flat fields.

### Finding 2 — Critical

Severity: Critical

File: `internal/server/api/runtime_handlers.go`

Evidence: `buildRuntimeConfigSnapshot` accepts request `config_set` or `config_set_json` and directly replaces the copied BackendRuntime snapshot with `copyConfigSet(jsonString(requestSet))`. Existing API tests still send flat items and assert flat `item["value"]` and `item["enabled"]` are preserved.

Impact: The API can persist non-final flat `config_set_json` as a NodeBackendRuntime snapshot, which violates the no-compatibility/tiered-only final-state policy and can feed the semantic normalizer/runtime path.

Required action: Either reject raw flat `config_set` input for this endpoint or validate/convert it into the final tiered ConfigSetBundle shape before persistence. Update tests so they fail on flat input and cover tiered input only.

### Finding 3 — Major

Severity: Major

File: `docs/reports/runtime-architecture-parameter-final-state/final-closeout.md`

Evidence: The closeout claims OI-01/OI-03/OI-04/OI-08/OI-09 are fully resolved and that old flat fallbacks were removed. Code reality shows flat semantic normalization and direct raw request ConfigSet persistence remain. The closeout commit list also stops at `393c891` and omits HEAD `5d8cd37`.

Impact: The closeout is not a reliable final acceptance artifact for current HEAD.

Required action: After implementation blockers are fixed, regenerate closeout evidence from current HEAD and include the actual final commit list.

### Finding 4 — Minor

Severity: Minor

File: `internal/server/runplan/resolve_with_sourcemap.go`, deployment handlers

Evidence: `ResolveWithSourceMap` is the shared resolver/source-map entry and is called by preview and lifecycle paths. However, `ResolveInput` assembly remains handler-local rather than clearly centralized in a single RunPlan builder function.

Impact: Lower immediate risk than the flat path because source maps are populated and tests pass, but the implementation is weaker than the strict "shared RunPlan builder" wording.

Required action: Clarify whether `ResolveWithSourceMap` is the intended shared builder or centralize input assembly after the blocking flat paths are fixed.

## 7. Final OI Status

- OI-01: DOCUMENTED_BLOCKER. Legacy flat fields are removed in ConfigEdit/API helper paths, but `semanticconfig.NormalizeConfigSet` still reads flat fields in a production runtime path.
- OI-02: FIXED. ConfigEdit API includes generated `config_view`.
- OI-03: DOCUMENTED_BLOCKER. `ParameterSourceMap` is populated by `ResolveWithSourceMap`, but the shared builder contract is weaker than documented because input assembly remains split.
- OI-04: DOCUMENTED_BLOCKER. Docker source map entries exist, but Docker semantic handling still relies on `enabled_fields` and flat `launcher.docker_options.value`.
- OI-05: FIXED. Web tests pass and legacy UI test checks pass.
- OI-06: DOCUMENTED_BLOCKER. This still appears to be an external hardware validation blocker for real Huawei/Ascend hardware, not an implementation blocker discovered by this audit.
- OI-07: FIXED. Production web legacy editor references are removed.
- OI-08: DOCUMENTED_BLOCKER. The DB column can store tiered JSON, but API paths can still persist raw flat ConfigSet JSON.
- OI-09: DOCUMENTED_BLOCKER. Catalog tiered shape exists, but semantic normalization remains flat-shape.
- OI-10: FIXED. Deployment create/preview/preflight reject `backend_runtime_id` as a legacy deployment entry.

All unresolved implementation issues found in this audit are recorded above as `DOCUMENTED_BLOCKER` entries.

## 8. Acceptance Decision

REJECT_WITH_BLOCKERS.

The implementation is close in several visible/API/UI areas and tests pass, but the final-state contract is not met while production runtime preparation still consumes flat ConfigItem fields and an API path can persist flat ConfigSet snapshots.

## 9. Commit ID for Audit Report

Pending at document creation time. The final terminal output for this audit records the commit that adds this report.

## 10. Push Result

Pending at document creation time. The final terminal output records the push result.

## 11. Final git status --short

Pending until after the audit report commit and push.
