# Acceptance Criteria and Test Requirements

## 1. Functional acceptance

### ConfigEdit Templates page

Pass conditions:

- The page is reachable from its intended route.
- The page title and menu label are localized in zh-CN.
- The table/list is populated from the real ConfigEdit template registry/materialization source when templates exist.
- The page shows template identity, backend/scope, and source.
- Selecting a template shows its fields/sections/details.
- Empty state, if possible in a fresh environment, is actionable and localized.

Fail conditions:

- The page shows only `暂无数据` while runtime parameter templates exist elsewhere.
- The page is fixed by frontend dummy data.
- The page is hidden from navigation without proving the underlying registry/API is working.
- The page still displays `ConfigEdit Templates`, `Refresh`, `Template`, `Backend`, `Source`, or `Select a template` in zh-CN.

### Runtime parameter level labels

Pass conditions:

- `Normal`, `Advanced`, `Developer` no longer leak in zh-CN.
- Labels are `常用`, `高级`, `专家`.
- Tooltip/help text explains the levels.
- Filtering is hierarchical:

```text
常用 -> normal only
高级 -> normal + advanced
专家 -> normal + advanced + developer
```

Fail conditions:

- The labels are translated but behavior is still ambiguous or inconsistent.
- The levels are treated as independent modes/profiles.

### Parameter ordering

Pass conditions:

- On page load, enabled parameters are shown first under `已启用参数`.
- Then normal/common parameters.
- Then advanced parameters.
- Then expert/developer parameters.
- Each group uses section + display_order + key/path for deterministic ordering.
- Sorting logic is shared across ConfigEdit consumers.

Fail conditions:

- Sorting is implemented only in one page.
- Sorting relies on page-specific hard-coded field names.
- High-risk Docker/security fields appear in common view.

### Checked/enabled placement

Pass conditions:

- Checked/enabled fields move to front after save/reload.
- Unchecked/disabled fields return to their original tier/section after save/reload.
- Checking/unchecking during editing does not immediately move the field.

Fail conditions:

- Row/card jumps immediately when a checkbox is toggled.
- Enabled state is lost after save/reload.
- Disabled values are deleted when they should only become inactive, unless the existing product contract explicitly deletes them.

### Raw JSON containment

Pass conditions:

- Known or materializable fields render structurally.
- Raw JSON appears only in expert/diagnostic view or explicit raw section.
- Fallback-materialized fields carry source metadata.

Fail conditions:

- Ordinary backend args or Docker options disappear from structured UI.
- Ordinary fields are only visible through raw JSON.

## 2. Required focused tests

Codex should add or update tests based on the existing stack. Names below are suggested and may be adapted.

### Frontend tests

Suggested coverage:

```text
ConfigEditTemplatesPage renders localized labels
ConfigEditTemplatesPage renders registry-backed templates
RuntimeParameterEditor renders 常用/高级/专家 labels
RuntimeParameterEditor filters tiers hierarchically
RuntimeParameterEditor groups enabled fields first on initial load
RuntimeParameterEditor does not reorder immediately on checkbox toggle
RuntimeParameterEditor returns unchecked field to original group after reload
RuntimeParameterEditor hides raw JSON outside expert view
```

### Backend/API tests

Suggested coverage:

```text
ConfigEdit template list endpoint returns registered templates
ConfigEdit template field DTO includes tier/section/source/order metadata
fallback materialization produces structured fields for unlisted known prefixes
Docker options materialize security/high-risk fields correctly
```

### i18n leak tests

Search/audit should fail if these leak in zh-CN UI snapshots or component output:

```text
ConfigEdit Templates
Refresh
Template
Backend
Source
Select a template
Normal
Advanced
Developer
```

Exact command depends on the project’s existing i18n audit. If absent, add focused unit tests.

## 3. Manual/API verification suggestions

Use the app’s existing dev startup scripts and authenticated API flow. Codex should adapt commands to the repo’s current scripts.

Useful checks:

```bash
# discover routes and endpoints
grep -R "configedit\|config-edit\|templates" -n internal web | head -200

# check frontend raw English leaks
grep -R "ConfigEdit Templates\|Select a template\|Normal\|Advanced\|Developer" -n web/src || true

# check backend template endpoint once server is running
curl -sS <server>/api/v1/<actual-configedit-template-endpoint> | jq .
```

Do not invent endpoint paths. Inspect the code and use the actual route.

## 4. Final evidence required

The closeout must include:

```text
Root cause:
- why ConfigEdit Templates was empty
- why English labels leaked
- why view levels/order were unclear or inconsistent

Changed files:
- backend files
- frontend files
- i18n files
- tests
- docs

Validation:
- go test ./...
- frontend test commands
- frontend build
- i18n audit or focused leak tests
- API/UI evidence for non-empty ConfigEdit Templates

Git:
- commit id
- push result
- git status --short clean output
```
