# Open Decisions

> Date: 2026-06-23
> Purpose: Decisions requiring user confirmation before implementation

---

## 1. Parameter Value Storage Format

**Question**: Should parameter values be stored as a flat map (`{"--max-model-len": 4096}`) or structured array (`[{"key": "max-model-len", "enabled": true, "value": 4096}]`)?

**Decision**: Structured array. All parameter JSON fields use `[]` default, not `{}`.

**Rationale**: Required for enabled/disabled state, source tracking, and tombstone semantics.

**SQL defaults**:
- `parameter_schema_json TEXT NOT NULL DEFAULT '[]'`
- `parameter_values_json TEXT NOT NULL DEFAULT '[]'`
- `disabled_parameters_json TEXT NOT NULL DEFAULT '[]'`
- `parameter_defaults_json TEXT NOT NULL DEFAULT '[]'`

---

## 1b. BackendRuntime Schema Snapshot

**Decision**: BackendRuntime saves BOTH `parameter_schema_json` AND `parameter_values_json`.

**Rationale**: BackendRuntime is the complete template for creating NBR. NBR deep-copies schema + values from BackendRuntime at creation time. RunPlan still does NOT query BackendRuntime at resolution time — it only reads from NBR snapshot.

---

## 2. NBR Snapshot Scope

**Question**: When creating NBR from BackendRuntime, should the snapshot include parameter schema or just parameter values?

**Options**:
- A) **Values only** — smaller snapshot, but needs schema lookup for validation
- B) **Schema + values** — self-contained, validation works offline

**Recommendation**: B — schema + values. NBR should be fully self-contained.

**Decision**: NBR MUST save both parameter schema snapshot AND parameter values snapshot. RunPlan validation must NOT query BackendVersion/BackendRuntime at resolution time. Implementation: add `parameter_schema_json` column to `node_backend_runtimes`.

---

## 2b. Deployment Disabled Override / Tombstone

**Question**: How should Deployment express "user explicitly disabled this upstream parameter"?

**Decision**: Deployment must save explicit disabled overrides. "Absent" cannot distinguish between:
- Upstream has no parameter
- User explicitly disabled parameter
- User never set parameter

**Implementation**: Add `disabled_parameters_json` column to `model_deployments`. Structure:
```json
{
  "key": "max-model-len",
  "enabled": false,
  "override_state": "disabled",
  "source": "deployment",
  "copied_from": "node_backend_runtime:xxx"
}
```
- Disabled parameters do NOT enter final args/env
- Disabled ≠ empty value
- Re-enable can restore copied value or user re-enters

---

## 3. RunPlan Resolver Migration

**Question**: How to migrate from current 5-layer resolver to NBR-only resolver?

**Options**:
- A) **Big bang** — rewrite resolver to only read NBR snapshot
- B) **Gradual** — add NBR snapshot path, keep old path as fallback, migrate later
- C) **Compatibility shim** — resolver reads NBR snapshot if available, falls back to old path

**Recommendation**: A — big bang. No backward compatibility needed. Rebuild DB after migration.

---

## 4. Deployment Parameter Merge Strategy

**Question**: When deployment overrides a parameter, should it replace the NBR value or merge with it?

**Options**:
- A) **Replace** — deployment value wins entirely
- B) **Merge** — deployment value merged with NBR defaults

**Recommendation**: A — replace. Simpler, clearer semantics. Deployment override has highest priority.

---

## 5. Disabled Parameter Persistence

**Question**: When a parameter is disabled, should it be stored as `{key, enabled: false}` or simply absent?

**Options**:
- A) **Store disabled** — explicit disabled state preserved
- B) **Absent** — disabled = not in the list

**Recommendation**: A — store disabled. Needed for "re-enable" and "reset to default" features.

---

## 6. Web Component Library

**Question**: Should `RuntimeParameterEditor` use Element Plus components or custom implementation?

**Options**:
- A) **Element Plus** — consistent with existing UI, use el-form, el-checkbox, el-input-number
- B) **Custom** — more control, but more work

**Recommendation**: A — Element Plus. Consistent with existing codebase.

---

## 7. Parameter Grouping Strategy

**Question**: How should parameters be grouped in the UI?

**Options**:
- A) **By target** — args, env, container config
- B) **By function** — GPU Memory, Context, Concurrency, Container Options
- C) **By risk** — safe, advanced, dangerous

**Recommendation**: B — by function. Most intuitive for users. Dangerous options (privileged, ipc) in separate group.

---

## 8. DB Rebuild Timing

**Question**: When should the DB be rebuilt with new schema?

**Options**:
- A) **With Batch B** — rebuild early, all subsequent batches use new schema
- B) **After Batch F** — rebuild at end, all changes in one DB rebuild

**Recommendation**: A — with Batch B. Cleaner, avoids migration complexity.

---

## 9. Old `parameters_json` Field

**Question**: What to do with existing `parameters_json` field on model_deployments?

**Options**:
- A) **Keep** — maintain backward compatibility
- B) **Replace** — migrate to `parameter_values_json` with structured format
- C) **Deprecate** — keep old field, add new field, migrate gradually

**Recommendation**: B — replace. No backward compatibility needed.

---

## 10. Template Sync UX

**Question**: How should "re-sync from upstream template" work?

**Options**:
- A) **Button + diff preview** — user clicks "Re-sync", sees diff, confirms
- B) **Automatic detection** — system detects template changes, prompts user
- C) **Manual only** — user explicitly re-creates from template

**Recommendation**: A — button + diff preview. First implementation can be simple button. Automatic detection is future enhancement.

---

## Deferred Items

These are NOT in scope for current implementation:

1. **Automatic template change detection** — future enhancement
2. **Multi-replica parameter consistency** — future when multi-replica is implemented
3. **Parameter versioning** — future enhancement
4. **Parameter inheritance rules** (e.g., "always use latest from template") — future enhancement
5. **Bulk parameter operations** (e.g., "apply to all NBRs") — future enhancement
