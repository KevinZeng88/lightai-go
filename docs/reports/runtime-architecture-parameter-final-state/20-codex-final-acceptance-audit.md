# Codex Final Acceptance Audit — Runtime Architecture and ConfigSetBundle Final State

CODEX_FINAL_ACCEPTANCE_AUDIT_COMPLETED

## 1. Verdict

ACCEPT

At audited HEAD `8523bd7`, the targeted ConfigSetBundle final-state blockers are closed. No reviewed public API path can persist raw flat ConfigSet into BackendVersion, BackendRuntime, or NBR snapshots. The production semanticconfig/runtime path consumes tiered ConfigItem semantics. Fresh Go tests, web tests, and web build all pass. `final-closeout.md` and the evidence files are consistent with the audited implementation state; OI-06 remains the only `DOCUMENTED_BLOCKER` and is external hardware validation.

## 2. Audited HEAD

Commit: `8523bd7`

Initial `git status --short`: clean.

`git log --oneline -25`:

```text
8523bd7 chore: final closeout cleanup — remove dead code, update commit list
6b60108 fix: reject raw config_set in BackendRuntime PATCH handler
50c576a docs: add final targeted acceptance audit
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
c97de2a feat(configset): batch-2 copy-on-create and local edits with owner preservation
48ecda3 feat(configset): batch-1 final ConfigSetBundle domain model with field-tier ConfigItem
```

`git show --stat --oneline HEAD`:

```text
8523bd7 chore: final closeout cleanup — remove dead code, update commit list
 .../runtime-architecture-parameter-final-state/final-closeout.md   | 7 +++++++
 internal/server/api/runtime_handlers.go                            | 5 -----
 2 files changed, 7 insertions(+), 5 deletions(-)
```

`git show --name-only --oneline HEAD`:

```text
8523bd7 chore: final closeout cleanup — remove dead code, update commit list
docs/reports/runtime-architecture-parameter-final-state/final-closeout.md
internal/server/api/runtime_handlers.go
```

## 3. Commands Run and Results

Dead-code / old-acceptance grep:

```text
$ git grep -n "old raw config_set acceptance\|if false\|configSet = incoming" -- internal/server/api/runtime_handlers.go internal/server/api/backend_handlers.go || true
(no output)
```

Raw ConfigSet ingress grep:

```text
internal/server/api/backend_handlers.go:379:	configSet := mapFromAny(req["config_set"])
internal/server/api/backend_handlers.go:382:		configSet = mapFromAny(req["config_set_json"])
internal/server/api/runtime_handlers.go:239:	if _, ok := req["config_set"]; ok {
internal/server/api/runtime_handlers.go:243:	if _, ok := req["config_set_json"]; ok {
internal/server/api/runtime_handlers.go:950:	if _, ok := req["config_set"]; ok {
internal/server/api/runtime_handlers.go:953:	if _, ok := req["config_set_json"]; ok {
```

Assessment: BackendVersion reads caller ConfigSet only to validate strict tiered shape before persistence. BackendRuntime PATCH and NBR enable reject raw `config_set` / `config_set_json`.

Flat ConfigItem grep:

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
internal/server/configedit/project.go:212:		// enabled_fields to avoid inferring activation from a prefilled value.
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

Assessment: production semantic/runtime path reads `item["value"]` as the tiered wrapper only. No reviewed production semantic/runtime path consumes flat `item["enabled"]`, `item["default_value"]`, `item["required"]`, or `enabled_fields` as final runtime model fields. Remaining `enabled_fields` hits are ConfigEdit UI/edit metadata paths and a stale test error string.

Verification:

```text
go test ./... -count=1
Result: PASS for all packages, including internal/server/api, internal/server/runplan, and internal/server/semanticconfig.
```

```text
cd web && npm test -- --run
Result: PASS. Output included "Passed: 12, Failed: 0", "All tests PASSED", and "ConfigEdit contract tests PASSED".
```

```text
cd web && npm run build
Result: PASS. Build completed in 3.64s with existing dependency PURE annotation warnings and chunk-size warning.
```

## 4. Public API Raw ConfigSet Ingress Assessment

### BackendVersion

ACCEPT.

`backend_handlers.go` still reads `req["config_set"]` and `req["config_set_json"]`, but only as caller-provided input to validate. `upsertBackendVersionFromRequest()` sets `callerProvided`, calls `validateTieredConfigSet(configSet)`, and returns before DB write on validation failure.

`validateTieredConfigSet()` rejects the required invalid shapes:

- top-level `code`
- top-level `enabled`
- top-level `default_value`
- top-level `required`
- `enabled_fields`
- scalar `value`
- missing `schema`
- missing `value`
- missing `state`

`backendVersionUpsertStatus()` maps tiered-shape validation failures to HTTP 400.

### BackendRuntime

ACCEPT.

`PATCH /api/v1/backend-runtimes/{id}` now rejects both raw inputs:

```text
runtime_handlers.go:239 if _, ok := req["config_set"]; ok { writeError(...400...) }
runtime_handlers.go:243 if _, ok := req["config_set_json"]; ok { writeError(...400...) }
```

Allowed field patches continue through `setConfigValue()` for fields such as `image_ref`, `docker_options`, `env`, `model_mount`, `health_check`, `entrypoint`, and `command`, preserving tiered shape.

### NBR

ACCEPT.

NBR create/enable snapshot creation rejects both raw inputs:

```text
runtime_handlers.go:950 if _, ok := req["config_set"]; ok { return error }
runtime_handlers.go:953 if _, ok := req["config_set_json"]; ok { return error }
```

It uses the BackendRuntime tiered `config_set_json` as snapshot base and supports targeted edits via `editable_config_patch`.

## 5. semanticconfig/runtime Path Assessment

ACCEPT.

`internal/server/semanticconfig/normalizer.go` reads the final tiered shape:

- `schema.key`
- `value.effective_value`
- `value.default_value`
- `state.enabled`
- `schema.required`

The targeted production path no longer consumes flat `item["enabled"]`, `item["default_value"]`, `item["required"]`, or `enabled_fields` as runtime model fields.

## 6. Snapshot Chain Assessment: BackendVersion -> BackendRuntime -> NBR

ACCEPT.

The public API snapshot chain is closed against raw flat ConfigSet persistence:

1. BackendVersion caller-provided ConfigSets must pass strict tiered validation before `backend_versions.config_set_json` write.
2. BackendRuntime create copies BackendVersion snapshot and allowed field changes are applied via `setConfigValue()`.
3. BackendRuntime patch rejects raw `config_set` and `config_set_json`, while allowed field patches preserve tiered shape.
4. NBR create/enable rejects raw `config_set` and `config_set_json`, then copy-on-create snapshots the BackendRuntime tiered ConfigSet.

No reviewed public API ingress remains that can persist raw flat ConfigSet into BackendVersion, BackendRuntime, or NBR snapshots.

## 7. Tests/Evidence/Closeout Consistency

ACCEPT.

Fresh verification commands passed. `final-closeout.md` now records:

- `6b60108 fix: reject raw config_set in BackendRuntime PATCH handler`
- `50c576a docs: add final targeted acceptance audit`
- `d958f02 fix: targeted reaudit blocker fix — BackendVersion raw config_set rejection`
- `4ea556a docs: add targeted codex blocker reaudit`
- `1428308 fix: codex final audit blocker fix — semanticconfig normalizer + NBR config_set rejection`
- `8e898f0 docs: add final codex implementation audit`

The evidence list includes:

- `codex-final-audit-blocker-fix.txt`
- `codex-targeted-reaudit-blocker-fix.txt`
- `backend-runtime-raw-configset-blocker-fix.txt`
- web test/build evidence

`final-closeout.md` states OI-06 is the only remaining `DOCUMENTED_BLOCKER`, and this matches the code/evidence reviewed in this final acceptance audit.

## 8. Remaining Blockers

Only OI-06 remains:

- DOCUMENTED_BLOCKER: NVIDIA real smoke and MetaX hardware validation require physical hardware unavailable in the dev environment.

No implementation blocker was found in this targeted acceptance audit.

## 9. Audit Report Commit ID

Pending at document creation time. The final terminal output records the commit that adds this report.

## 10. Push Result

Pending at document creation time. The final terminal output records the push result.

## 11. Final git status --short

Pending until after the audit report commit and push.
