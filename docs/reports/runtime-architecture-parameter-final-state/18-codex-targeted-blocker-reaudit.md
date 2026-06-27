# Codex Targeted Blocker Re-audit — Runtime Architecture and ConfigSetBundle Final State

CODEX_TARGETED_BLOCKER_REAUDIT_COMPLETED

## 1. Verdict

REJECT_WITH_BLOCKERS

The direct blocker fixes in `afcf19d`/`1428308` closed the semantic normalizer flat-field reads and the direct NBR enable `config_set`/`config_set_json` acceptance. However, raw flat `config_set` can still be accepted and persisted through BackendVersion APIs, copied into BackendRuntime snapshots, and then copied into NBR snapshots. That keeps the raw ConfigSet persistence blocker open under the targeted re-audit rules.

## 2. Audited HEAD

Commit: `afcf19d`

Initial `git status --short`: clean.

`git log --oneline -15`:

```text
afcf19d docs: update final closeout with codex audit blocker fix commit
1428308 fix: codex final audit blocker fix — semanticconfig normalizer + NBR config_set rejection
8e898f0 docs: add final codex implementation audit
5d8cd37 docs: final closeout and test hygiene cleanup
393c891 fix: final repair redo — remove all flat fallbacks, fix tiered value structure
8f3f86e fix: final repair — remove flat fallbacks, fix setConfigValueTiered, strengthen SourceMap
95156ce docs: update final closeout — OI-10 fully resolved
c082d49 feat: OI-10 add node_backend_runtime_id column to model_deployments
05671a5 docs: final closeout — configset-bundle final-state implementation complete
b6d6b6c feat: OI-02 wire ConfigView into config-edit API response
45f3d74 feat: OI-05+07 remove legacy RuntimeParameterEditor and HumanRuntimeParameterForm
3911175 feat: OI-03+04 integrate SourceMapBuilder into RunPlan resolver
fc12301 feat: OI-01+05+08+09 remove legacy flat fields, update to tiered-only ConfigItem
beff997 docs: batch-6 final closeout for configset-bundle final-state implementation
bbd43fc docs: batch-5 api-first e2e evidence — full test suite results
```

`git show --stat --oneline HEAD`:

```text
afcf19d docs: update final closeout with codex audit blocker fix commit
 .../runtime-architecture-parameter-final-state/final-closeout.md        | 2 +-
 1 file changed, 1 insertion(+), 1 deletion(-)
```

`git show --name-only --oneline HEAD`:

```text
afcf19d docs: update final closeout with codex audit blocker fix commit
docs/reports/runtime-architecture-parameter-final-state/final-closeout.md
```

## 3. Commands Run and Results

Required grep:

```text
internal/server/api/config_edit_handlers_test.go:120:		if vt, ok := item["value"].(map[string]interface{}); !ok || vt["effective_value"] != float64(4096) {
internal/server/api/configset_helpers.go:42:	if v, ok := item["value"].(map[string]interface{}); ok {
internal/server/api/configset_helpers.go:208:	if v, ok := item["value"].(map[string]interface{}); ok {
internal/server/api/configset_helpers.go:279:// item["value"] must remain {default_value, inherited_value, local_value, effective_value}.
internal/server/api/configset_helpers.go:281:// item["value"] with a scalar.
internal/server/api/configset_helpers.go:286:	valueTier, _ := item["value"].(map[string]interface{})
internal/server/api/configset_helpers.go:289:		item["value"] = valueTier
internal/server/api/configset_helpers.go:314:		item["value"] = map[string]interface{}{}
internal/server/api/configset_helpers.go:372:				if v, ok := item["value"]; ok {
internal/server/api/configset_helpers.go:375:				if enabled, ok := item["enabled"].(bool); ok {
internal/server/api/runtime_boundary_test.go:205:		vt, _ := item["value"].(map[string]interface{})
internal/server/api/workflow_deployment_runplan_test.go:403:	if v, ok := item["value"].(map[string]interface{}); ok {
internal/server/api/workflow_deployment_runplan_test.go:408:	return item["value"]
internal/server/configedit/apply.go:57:	fields, _ := item["enabled_fields"].(map[string]any)
internal/server/configedit/apply.go:60:		item["enabled_fields"] = fields
internal/server/configedit/configset_adapter.go:47:	// Tiered: item["value"] may already be the ConfigItemValue wrapper.
internal/server/configedit/configset_adapter.go:49:	if vt, ok := item["value"].(map[string]any); ok {
internal/server/configedit/configset_adapter.go:63:	item["value"] = map[string]any{"effective_value": vt}
internal/server/configedit/configset_adapter.go:118:	if v, ok := item["value"].(map[string]any); ok {
internal/server/configedit/configset_adapter.go:126:	return item["value"]
internal/server/configedit/project.go:195:	enabledFields := nestedMap(item, "enabled_fields")
internal/server/configedit/project.go:212:		// enabled_fields to avoid inferring activation from a prefilled value.
internal/server/configedit/project.go:316:		DefaultValue:    item["default_value"],
internal/server/configedit/project.go:430:	if dv := item["default_value"]; dv != nil {
internal/server/configedit/tiered_helpers.go:4:// item["value"] must remain {default_value, inherited_value, local_value, effective_value}.
internal/server/configedit/tiered_helpers.go:5:// Updates local_value and effective_value. NEVER overwrites item["value"] with a scalar.
internal/server/configedit/tiered_helpers.go:10:	valueTier, _ := item["value"].(map[string]any)
internal/server/configedit/tiered_helpers.go:13:		item["value"] = valueTier
internal/server/configedit/tiered_helpers.go:38:	if v, ok := item["value"].(map[string]any); ok {
internal/server/semanticconfig/normalizer.go:68:	if vt, ok := item["value"].(map[string]any); ok {
internal/server/semanticconfig/normalizer.go:117:	if vt, ok := item["value"].(map[string]any); ok {
internal/server/semanticconfig/normalizer.go:125:	vt, _ := item["value"].(map[string]any)
internal/server/semanticconfig/normalizer.go:128:		item["value"] = vt
internal/server/semanticconfig/registry_normalizer_test.go:146:		t.Fatalf("docker subfield enabled_fields metadata was not honored: %#v", out.Items["docker.shm_size"])
```

Assessment of grep: `semanticconfig/normalizer.go` no longer reads flat `item["default_value"]`, `item["enabled"]`, `item["required"]`, or `enabled_fields`; its `item["value"]` reads are tiered wrapper reads. Remaining `enabled_fields` hits are in ConfigEdit UI paths and one stale test failure message, not semantic runtime normalization.

`go test ./... -count=1`: PASS. All listed packages passed, including `internal/server/api`, `internal/server/runplan`, and `internal/server/semanticconfig`.

`cd web && npm test -- --run`: PASS. Output included `Passed: 12, Failed: 0` and `All tests PASSED`.

`cd web && npm run build`: PASS. Build completed in `3.61s` with existing dependency PURE annotation warnings and chunk-size warning.

## 4. Blocker 1 Assessment: semanticconfig flat-field removal

Status: FIXED.

`internal/server/semanticconfig/normalizer.go` now reads:

- key from `schema.key`
- Docker options from `value.effective_value`
- enabled state from `state.enabled`
- required state from `schema.required`
- default value from `value.default_value`

`semanticDeploymentSnapshot` still calls `semanticconfig.NormalizeConfigSet`, and deployment preview/lifecycle still pass that snapshot into `runplan.ApplySemanticSnapshot` before `ResolveWithSourceMap`. The important change is that this path now consumes tiered ConfigItem shape rather than flat runtime values.

## 5. Blocker 2 Assessment: raw config_set/config_set_json rejection

Status: DOCUMENTED_BLOCKER.

Direct NBR creation has been fixed:

```text
internal/server/api/runtime_handlers.go:947 if _, ok := req["config_set"]; ok {
internal/server/api/runtime_handlers.go:948     return "", fmt.Errorf("config_set is not accepted; use editable_config_patch to modify individual parameters")
internal/server/api/runtime_handlers.go:950 if _, ok := req["config_set_json"]; ok {
internal/server/api/runtime_handlers.go:951     return "", fmt.Errorf("config_set_json is not accepted; use editable_config_patch to modify individual parameters")
```

However, raw ConfigSet payloads can still enter the same snapshot chain upstream:

```text
internal/server/api/backend_handlers.go:379 configSet := mapFromAny(req["config_set"])
internal/server/api/backend_handlers.go:381 configSet = mapFromAny(req["config_set_json"])
internal/server/api/backend_handlers.go:402 ... configSetJSON(configSet) ...
```

BackendRuntime creation then copies the BackendVersion `config_set_json`:

```text
internal/server/api/runtime_handlers.go:107 versionSet := mapFromAny(version["config_set"])
internal/server/api/runtime_handlers.go:108 configSet := copyConfigSet(rawJSONString(version["config_set_json"], "{}"))
internal/server/api/runtime_handlers.go:110 configSet = versionSet
```

NBR creation then copies the BackendRuntime `config_set_json` as its frozen snapshot base. Therefore raw flat `config_set` can still be persisted into an eventual NBR snapshot through BackendVersion -> BackendRuntime -> NBR, even though the direct NBR endpoint now rejects raw `config_set` and `config_set_json`.

`runtime_boundary_test.go` also still contains flat BackendVersion fixtures:

```text
runtime_boundary_test.go:40  config_set item contains flat code/enabled/value/default_value/render fields
runtime_boundary_test.go:70  patch config_set item contains flat code/enabled/value/default_value/render fields
runtime_boundary_test.go:402 user BackendVersion create fixture contains flat config_set item
runtime_boundary_test.go:417 user BackendVersion patch fixture contains flat config_set item
```

This means the targeted fix did not eliminate raw flat ConfigSet acceptance across the snapshot chain.

## 6. Tests and Evidence Assessment

Tests pass, but coverage is incomplete for the raw ConfigSet rejection requirement:

- `registry_normalizer_test.go` fixture for `NormalizeConfigSetDoesNotDefaultMissingEnabledToTrue` is tiered.
- The NBR editable patch test now reads tiered `value.effective_value` and `state.enabled`.
- No dedicated test was found that asserts direct NBR `config_set` and `config_set_json` requests return 400.
- Existing `runtime_boundary_test.go` still uses flat BackendVersion `config_set` fixtures, and those tests pass.

Evidence file `evidence/codex-final-audit-blocker-fix.txt` accurately describes the intended direct fixes. `final-closeout.md` is not fully consistent with current evidence requirements:

- It lists `1428308` but does not list audited HEAD `afcf19d`.
- Its evidence list does not include `evidence/codex-final-audit-blocker-fix.txt`.
- It still states final status `PASS`, which conflicts with the remaining raw ConfigSet upstream persistence path found in this re-audit.

## 7. Remaining Blockers

- DOCUMENTED_BLOCKER: Raw flat `config_set` / `config_set_json` is still accepted by BackendVersion APIs and can flow through BackendRuntime copy-on-create into NBR snapshots.
- DOCUMENTED_BLOCKER: `runtime_boundary_test.go` still contains flat ConfigSet fixtures for BackendVersion create/patch paths, so the tests do not prove tiered-only final-state across the snapshot chain.
- DOCUMENTED_BLOCKER: `final-closeout.md` materially overstates final-state completion because it does not reflect the remaining upstream raw ConfigSet acceptance and does not register the blocker fix evidence file.

## 8. Commit ID for Audit Report

Pending at document creation time. The final terminal output records the commit that adds this report.

## 9. Push Result

Pending at document creation time. The final terminal output records the push result.

## 10. Final git status --short

Pending until after the audit report commit and push.
