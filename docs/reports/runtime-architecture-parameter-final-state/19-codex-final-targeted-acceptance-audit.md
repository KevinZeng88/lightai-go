# Codex Final Targeted Acceptance Audit — Runtime Architecture and ConfigSetBundle Final State

CODEX_FINAL_TARGETED_ACCEPTANCE_AUDIT_COMPLETED

## 1. Verdict

REJECT_WITH_BLOCKERS

BackendVersion caller-provided `config_set` / `config_set_json` now receives strict tiered validation, and `semanticconfig` remains tiered-only in the production normalization path. However, a separate public API path still accepts and persists caller-provided raw `config_set`: `PATCH /api/v1/backend-runtimes/{id}` copies `req["config_set"]` directly into `backend_runtimes.config_set_json` without `validateTieredConfigSet()`. Because BackendRuntime is the direct parent snapshot copied into NBR, this still permits a public API to persist flat ConfigSet into the final runtime snapshot chain.

## 2. Audited HEAD

Commit: `b7e05e4`

Initial `git status --short`: clean.

`git log --oneline -20`:

```text
b7e05e4 docs: update closeout with targeted reaudit blocker fix commit
d958f02 fix: targeted reaudit blocker fix — BackendVersion raw config_set rejection
4ea556a docs: add targeted codex blocker reaudit
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
90a1ff5 feat(runplan): batch-4 shared RunPlan builder and parameter source map
4c5d952 feat(configset): batch-3 ConfigView/ConfigPanel presentation and GenericConfigSetRenderer
```

`git show --stat --oneline HEAD`:

```text
b7e05e4 docs: update closeout with targeted reaudit blocker fix commit
 .../reports/runtime-architecture-parameter-final-state/final-closeout.md | 1 +
 1 file changed, 1 insertion(+)
```

`git show --name-only --oneline HEAD`:

```text
b7e05e4 docs: update closeout with targeted reaudit blocker fix commit
docs/reports/runtime-architecture-parameter-final-state/final-closeout.md
```

## 3. Commands Run and Results

Required ConfigSet grep:

```text
internal/server/api/backend_handlers.go:379:	configSet := mapFromAny(req["config_set"])
internal/server/api/backend_handlers.go:382:		configSet = mapFromAny(req["config_set_json"])
internal/server/api/backend_handlers.go:387:			if err := validateTieredConfigSet(configSet); err != nil {
internal/server/api/runtime_handlers.go:239:	if _, ok := req["config_set"]; ok {
internal/server/api/runtime_handlers.go:240:		if incoming, ok := req["config_set"].(map[string]interface{}); ok {
internal/server/api/runtime_handlers.go:241:			configSet = incoming
internal/server/api/runtime_handlers.go:244:	sets = append(sets, "config_set_json = ?", "checksum = ?")
internal/server/api/runtime_handlers.go:947:	if _, ok := req["config_set"]; ok {
internal/server/api/runtime_handlers.go:950:	if _, ok := req["config_set_json"]; ok {
```

The full grep also includes expected DB read/write references for `config_set_json`, and test assertions around snapshots.

Required flat-field grep:

```text
internal/server/api/runtime_boundary_test.go:205:		vt, _ := item["value"].(map[string]interface{})
internal/server/configedit/apply.go:57:	fields, _ := item["enabled_fields"].(map[string]any)
internal/server/configedit/apply.go:60:		item["enabled_fields"] = fields
internal/server/configedit/configset_adapter.go:47:	// Tiered: item["value"] may already be the ConfigItemValue wrapper.
internal/server/configedit/configset_adapter.go:49:	if vt, ok := item["value"].(map[string]any); ok {
internal/server/configedit/configset_adapter.go:63:	item["value"] = map[string]any{"effective_value": vt}
internal/server/configedit/configset_adapter.go:118:	if v, ok := item["value"].(map[string]any); ok {
internal/server/configedit/configset_adapter.go:126:	return item["value"]
internal/server/configedit/project.go:195:	enabledFields := nestedMap(item, "enabled_fields")
internal/server/configedit/project.go:316:		DefaultValue:    item["default_value"],
internal/server/configedit/project.go:430:	if dv := item["default_value"]; dv != nil {
internal/server/configedit/tiered_helpers.go:10:	valueTier, _ := item["value"].(map[string]any)
internal/server/configedit/tiered_helpers.go:13:		item["value"] = valueTier
internal/server/configedit/tiered_helpers.go:38:	if v, ok := item["value"].(map[string]any); ok {
internal/server/semanticconfig/normalizer.go:68:	if vt, ok := item["value"].(map[string]any); ok {
internal/server/semanticconfig/normalizer.go:117:	if vt, ok := item["value"].(map[string]any); ok {
internal/server/semanticconfig/normalizer.go:125:	vt, _ := item["value"].(map[string]any)
internal/server/semanticconfig/normalizer.go:128:		item["value"] = vt
internal/server/semanticconfig/registry_normalizer_test.go:146:		t.Fatalf("docker subfield enabled_fields metadata was not honored: %#v", out.Items["docker.shm_size"])
```

Assessment: `semanticconfig` hits are tiered-wrapper reads, not flat runtime reads. `configedit` hits are UI/edit projection and Docker UI metadata paths, outside the targeted production semantic/runplan path.

`go test ./... -count=1`: PASS. All packages passed, including `internal/server/api`, `internal/server/runplan`, and `internal/server/semanticconfig`.

`cd web && npm test -- --run`: PASS. Output included `Passed: 12, Failed: 0`, `All tests PASSED`, and `ConfigEdit contract tests PASSED`.

`cd web && npm run build`: PASS. Build completed in `3.58s` with existing dependency PURE annotation warnings and chunk-size warning.

## 4. BackendVersion Raw ConfigSet Assessment

Status: FIXED for BackendVersion create/patch.

`validateTieredConfigSet()` rejects:

- top-level `code`
- top-level `enabled`
- top-level `default_value`
- top-level `required`
- `enabled_fields`
- scalar `value`
- missing `schema`
- missing `value`
- missing `state`

BackendVersion create and patch call `upsertBackendVersionFromRequest()`, which tracks whether the caller supplied `config_set` or `config_set_json`; caller-provided sets are validated before persistence. Validation errors are mapped by `backendVersionUpsertStatus()` to HTTP 400 because the error text includes `tiered shape`.

Test coverage found:

- `TestBackendVersionCreateRejectsFlatConfigSet`
- `TestBackendVersionPatchRejectsFlatConfigSet`
- `TestBackendVersionCreateAcceptsTieredConfigSet`
- Existing BackendVersion success fixtures have been converted to tiered `schema` / `state` / `value` structure.

Coverage caveat: the tests explicitly cover common flat create/patch rejection and strict tiered create success. They do not individually enumerate every invalid shape listed above, but the validator code does cover those branches.

## 5. Snapshot Chain Assessment: BackendVersion -> BackendRuntime -> NBR

Status: DOCUMENTED_BLOCKER.

The BackendVersion -> BackendRuntime part is improved because caller-provided BackendVersion ConfigSets are now validated before persistence. Direct NBR creation also rejects raw `config_set` and `config_set_json`.

However, BackendRuntime patch remains a public API route:

```text
internal/server/api/router.go:166: PATCH /api/v1/backend-runtimes/{id}
```

That handler still accepts caller-provided raw ConfigSet without validation:

```text
internal/server/api/runtime_handlers.go:239 if _, ok := req["config_set"]; ok {
internal/server/api/runtime_handlers.go:240     if incoming, ok := req["config_set"].(map[string]interface{}); ok {
internal/server/api/runtime_handlers.go:241         configSet = incoming
internal/server/api/runtime_handlers.go:244 sets = append(sets, "config_set_json = ?", "checksum = ?")
```

NBR creation copies the BackendRuntime `config_set_json` as its snapshot base:

```text
internal/server/api/runtime_handlers.go:944 set := copyConfigSet(rawJSONString(rt["config_set_json"], "{}"))
```

Therefore a public API can still persist flat ConfigSet into `backend_runtimes.config_set_json`, and NBR creation can then copy that flat snapshot. This violates the acceptance rule that no public API can persist flat ConfigSet into the final snapshot chain.

## 6. semanticconfig/runtime Path Assessment

Status: FIXED for the targeted semanticconfig blocker.

`internal/server/semanticconfig/normalizer.go` reads:

- key from `schema.key`
- Docker options from `value.effective_value`
- enabled state from `state.enabled`
- required state from `schema.required`
- default value from `value.default_value`

No production semantic/runplan path was found that still consumes flat `item["enabled"]`, `item["default_value"]`, `item["required"]`, or `enabled_fields` as runtime model fields.

## 7. Tests/Evidence/Closeout Consistency

Status: DOCUMENTED_BLOCKER.

Fresh tests pass, but closeout is not consistent with this audit:

- `final-closeout.md` lists `d958f02` but does not list Codex re-audit commit `4ea556a`.
- `final-closeout.md` does not list current closeout update commit `b7e05e4` in the commit list.
- `final-closeout.md` evidence list does not include `evidence/codex-targeted-reaudit-blocker-fix.txt`.
- `final-closeout.md` says final status `PASS`, which contradicts the remaining public BackendRuntime raw ConfigSet persistence path found here.

Because there is a production/API blocker, these are not merely minor documentation issues.

## 8. Remaining Blockers

- DOCUMENTED_BLOCKER: `PATCH /api/v1/backend-runtimes/{id}` can still persist caller-provided raw `config_set` into `backend_runtimes.config_set_json`.
- DOCUMENTED_BLOCKER: NBR creation can copy that BackendRuntime snapshot into `node_backend_runtimes.config_set_json`.
- DOCUMENTED_BLOCKER: `final-closeout.md` materially contradicts code reality and does not register all required audit/fix evidence and commits.

OI-06 remains a separate external hardware `DOCUMENTED_BLOCKER`, but it is not the only blocker at this HEAD.

## 9. Audit Report Commit ID

Pending at document creation time. The final terminal output records the commit that adds this report.

## 10. Push Result

Pending at document creation time. The final terminal output records the push result.

## 11. Final git status --short

Pending until after the audit report commit and push.
