# ConfigEdit Public Rendering Closeout

Date: 2026-07-01

## Status

CODE_AND_TEST_FIXED

All issues identified in this task have code/test fixes. No known code-level blocker remains. Final browser confirmation is recommended after reload with current runtime data.

## Root Cause

The public ConfigEdit projection and renderer conflated advanced/expert fields with diagnostic metadata. Backend projection set `diagnostic=true` for ordinary advanced fields, frontend grouping treated `diagnostic` as an expert-group signal, and the field renderer displayed a public diagnostic badge. Generic array/object fields without a known widget also fell back to readonly JSON-like text, so empty low-level launcher arrays appeared as useless `[]` fields.

This was not BackendRuntimesPage-specific because BackendRuntimesPage only consumes `ConfigEditView`; the same projection/rendering path is shared by BackendRuntime, NodeBackendRuntime, Deployment, parameter templates, and future ConfigEdit consumers.

## Affected Public Components

- Backend ConfigEdit projection: `internal/server/configedit/project.go`
- Backend ConfigEdit taxonomy and field type metadata: `internal/server/configedit/taxonomy.go`, `internal/server/configedit/types.go`
- Frontend public field renderer: `web/src/components/config/ConfigField.vue`
- Frontend grouping and metadata utilities: `web/src/utils/configEditView.ts`, `web/src/utils/configEditFieldMeta.ts`
- Runtime template drawer interaction: `web/src/pages/BackendRuntimesPage.vue`

## What Changed

- Diagnostic semantics now mean internal/debug/source/resolver/raw metadata only.
- Advanced, expert, model runtime, launcher, runtime, service, and Docker option fields are no longer diagnostic by default.
- High-risk Docker/security fields use `risk=high` for public high-risk presentation instead of diagnostic.
- Public ConfigField no longer renders the diagnostic badge in normal field headers.
- ConfigEdit grouping no longer uses `field.diagnostic` alone to move fields into the expert group.
- Editable unrecognized arrays/objects now get a JSON textarea fallback instead of readonly `[]` or `{}` text.
- Readonly unrecognized arrays/objects show a readable count/empty summary.
- Structured widgets remain preferred for:
  - `service.container_port` / service port: `port_form`
  - `runtime.model_mount`: `mount_form`
  - `runtime.device_binding`: `accelerator_binding`
  - Docker devices/env/options: structured table/list widgets where available
- Empty low-level `launcher.ports`, `launcher.volumes`, and `launcher.devices` are hidden from public normal/advanced views when structured equivalents exist. They remain available only in developer diagnostics.
- Visible fields without explicit descriptions now get a fallback tooltip/help string based on key, type, and section.
- Common ConfigEdit labels/descriptions were added for port, mount, device binding, Docker options, runtime command/entrypoint, GPU memory utilization, tensor/pipeline parallelism, shm/ipc/security options, and environment variables.
- BackendRuntimesPage cancel/save now exits the edit drawer state after discarding or saving changes; no field-specific rendering logic was added there.

## Fields Fixed

- `launcher.ports`, `launcher.volumes`, `launcher.devices`: no longer public useless empty-array fields when structured alternatives exist.
- `service.container_port`, `service.host_port`, `service.listen_host`: structured service/port display path.
- `runtime.model_mount`: structured model mount display path.
- `runtime.device_binding`: structured accelerator binding display path.
- `launcher.docker_options.*`: split Docker options remain structured and high-risk options are marked with risk metadata.
- `model_runtime.*` advanced/expert parameters: no longer diagnostic by default.

## API / Projection Verification Example

The projection-level regression test exercises the ConfigEdit API payload shape before HTTP serialization:

```text
go test ./internal/server/configedit
```

Sample projected field expectations verified by test:

```json
{
  "key": "model_runtime.tensor_parallel_size",
  "diagnostic": false
}
```

```json
{
  "key": "runtime.device_binding",
  "widget": "accelerator_binding",
  "section": "devices_mounts",
  "diagnostic": false
}
```

```json
{
  "key": "launcher.ports",
  "visibility": "internal",
  "readonly": true,
  "diagnostic": true
}
```

`launcher.ports` is only present in developer view in that internal diagnostic form; it is not present in public advanced view when structured service port fields exist.

## Tests Added Or Updated

- Backend:
  - Advanced model runtime fields are not diagnostic.
  - `launcher.*`, `runtime.*`, `service.*`, and Docker options are not diagnostic unless internal/debug.
  - Internal/source/debug fields remain diagnostic.
  - Empty low-level launcher arrays are not projected as public advanced fields when structured equivalents exist.
  - Service port, model mount, and device binding structured fields are projected.
- Frontend:
  - ConfigField does not render the diagnostic badge for ordinary fields.
  - ConfigEdit grouping does not place a field into expert group only because `diagnostic` is true.
  - Editable array/object fields without a specific widget use editable JSON fallback.
  - Structured port/mount/device widgets render when available.
  - Fallback help/tooltip appears for visible fields without explicit metadata.

## Verification Results

All requested commands passed:

```bash
git status --short
go test ./...
cd web
npm test
npm run test:unit
npm run build
```

Results:

- `go test ./...`: PASS
- `npm test`: PASS, including `npm run test:unit`
- `npm run test:unit`: PASS, 15 files / 115 tests
- `npm run build`: PASS

Build emitted existing Rollup/Vite chunk-size and pure-comment warnings only; the build completed successfully.

## Skipped Tests

None.

## Problem Closure

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| CE-001 | Diagnostic badge displayed on ordinary ConfigEdit fields | User report and renderer check | Mislabels normal runtime parameters as diagnostics | FIXED | `web/src/components/config/ConfigField.vue`; `internal/server/configedit/project.go` | `go test ./...`; `npm test`; `npm run test:unit` | Public diagnostic badge removed; diagnostic semantics narrowed |
| CE-002 | Empty low-level launcher arrays displayed as useless `[]` fields | User report and projection tests | Port/mount/device configuration looked readonly and unusable | FIXED | `internal/server/configedit/project.go`; `web/src/components/config/ConfigField.vue` | `go test ./internal/server/configedit`; frontend unit tests | Empty low-level arrays hidden from public views when structured fields exist; editable fallback added |
| CE-003 | Help icons could disappear for fields without explicit descriptions | Renderer metadata behavior | Visible parameters lacked parameter explanations | FIXED | `web/src/utils/configEditFieldMeta.ts`; locale files | `npm test`; `npm run test:unit` | Fallback tooltip/help added |
| CE-004 | Runtime template save/cancel could leave the user in edit context | User report | Confusing edit drawer state after save/cancel | FIXED | `web/src/pages/BackendRuntimesPage.vue` | `npm test`; `npm run build` | Cancel/save exit edit drawer state |

All known problems are FIXED. No DOCUMENTED_BLOCKER or INVALID entries were needed.

## Final Diff Review

- `git diff --stat`: reviewed; scope is limited to ConfigEdit backend projection/taxonomy/types/tests, shared ConfigEdit frontend components/utils/tests/locales, `BackendRuntimesPage.vue` edit-state behavior, and this closeout document.
- `git diff --check`: PASS.
- No page-level hardcoded avoidance for `Port mappings`, `Volume mounts`, or `Device bindings` was added to `BackendRuntimesPage.vue`.
- Diagnostic badge, array/object fallback, grouping, and tooltip fixes are in shared ConfigEdit projection/rendering/utilities.

## Commit And Push

- Fix commit id: pending.
- Closeout metadata commit id: pending.
- Push result: pending.

## Final Git Status

Final status after commit and push must be checked with:

```bash
git status --short
```
