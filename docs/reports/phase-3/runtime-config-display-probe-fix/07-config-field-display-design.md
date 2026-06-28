# Runtime Config Field Display Fix — Design and Acceptance

## 1. Background

The previous fix made the runtime template detail page show parameter sections again. The uploaded MHTML snapshot shows a second-stage bug: several fields under Docker options render the entire parent object instead of their own child value.

Example from the rendered page:

```text
UTS mode = {"gpu_capabilities":[["gpu"]],"gpu_driver":"","ipc_mode":"host","shm_size":"16gb"}
Network mode = {"gpu_capabilities":[["gpu"]],"gpu_driver":"","ipc_mode":"host","shm_size":"16gb"}
```

That value is the whole `launcher.docker_options` object. It is not the value of `uts_mode` or `network_mode`.

## 2. Goal

Fix the current visible runtime template detail issues and the same class of object/list display issues in the config detail view.

The goal is not to redesign the runtime/config architecture. The goal is to make the current ConfigEditView render the existing config model correctly and readably.

## 3. Scope

### In scope

- `ConfigEditView` value resolution for nested/object fields.
- Runtime template detail display.
- User runtime config detail display if it uses the same component.
- Docker options field mapping.
- Display renderers for common object/list values in read-only/detail mode.
- Raw Config Set JSON diagnostic presentation.
- Frontend tests for object child path rendering and structured display.
- Minimal backend/config-edit view adjustment only if the frontend cannot reliably infer child value paths.

### Out of scope

- No runtime/config architecture redesign.
- No new multi-version backend catalog design.
- No full Docker options management system.
- No manual UI smoke as the only validation.
- No compatibility branch for old databases.
- No unrelated runtime deployment changes.
- No broad refactor of all forms/components unless required by this bug.

## 4. Root problem

The current rendered fields show this pattern:

```text
field_key = launcher.docker_options.uts_mode
internal_key = launcher.docker_options
rendered value = full launcher.docker_options object
```

A projected child field must have a child value path. It must not render the parent object.

Correct behavior:

```text
launcher.docker_options.shm_size -> "16gb"
launcher.docker_options.ipc_mode -> "host"
launcher.docker_options.uts_mode -> absent -> hide or 未配置
launcher.docker_options.network_mode -> absent -> hide or 未配置
```

## 5. Design rules

### Rule 1: Separate storage key, value path, and display key

A config field may have three concepts:

```text
storage_key: the ConfigSet item key, for example launcher.docker_options
value_path: the path used to read a value, for example launcher.docker_options.ipc_mode
display_key: the stable UI key/label, for example Docker IPC mode
```

If a field represents an object child, it must have a `value_path` or equivalent mapping. It cannot use only the parent `storage_key` as the rendered value.

### Rule 2: Parent object rendering is allowed only for parent-level summary fields

Rendering the whole object is acceptable only if the UI field itself is a parent summary field such as:

```text
Docker options summary
Runtime environment summary
Health check summary
```

It is not acceptable for child fields such as:

```text
UTS mode
Network mode
IPC mode
Shared memory
Security options
Ulimits
Devices
Group add
```

### Rule 3: Canonical Docker options path

Use the canonical value path namespace:

```text
launcher.docker_options.shm_size
launcher.docker_options.privileged
launcher.docker_options.ipc_mode
launcher.docker_options.network_mode
launcher.docker_options.uts_mode
launcher.docker_options.security_options
launcher.docker_options.ulimits
launcher.docker_options.devices
launcher.docker_options.optional_devices
launcher.docker_options.group_add
launcher.docker_options.gpu_driver
launcher.docker_options.gpu_capabilities
```

If existing UI aliases such as `docker.shm_size` or `docker.network_mode` are kept, they must be explicit aliases to the canonical value paths above.

Do not create another naming convention.

### Rule 4: Missing optional advanced fields must not show the parent object

For optional advanced Docker fields with no value:

```text
uts_mode
network_mode
security_options
ulimits
devices
optional_devices
group_add
```

Accepted behavior:

- Hide the field in normal detail view, or
- Show `未配置` / `Not configured`.

Rejected behavior:

- Show the parent `launcher.docker_options` object.
- Show `{}` or `[]` as raw JSON in normal detail view.
- Show broken empty controls.

### Rule 5: Read-only detail mode should use structured display renderers

Normal details should not show raw JSON for common runtime config objects.

Required compact display behavior:

| Config key | Desired normal detail display |
|---|---|
| `runtime.model_mount` | `/models (read-only)` or equivalent structured text |
| `runtime.env` | key-value rows, e.g. `CUDA_VISIBLE_DEVICES = {{vendor_visible_devices}}` |
| `runtime.health` | `HTTP /v1/models`, timeout `120s`, success statuses `200` |
| `launcher.entrypoint` | joined command, e.g. `vllm serve` |
| `launcher.command` | joined args, e.g. `--model {{model_container_path}}` |
| `launcher.ports` | compact list, or `未配置` when empty |
| `launcher.volumes` | compact list, or `未配置` when empty |
| `launcher.devices` | compact list, or `未配置` when empty |
| `launcher.docker_options.ulimits` | compact list, or `未配置` when empty |
| `launcher.docker_options.security_options` | compact list, or `未配置` when empty |

This can be implemented as a small display-value helper rather than a large component refactor.

### Rule 6: Raw Config Set JSON is diagnostic-only

Raw Config Set JSON must remain available, but it must be under a diagnostic section and collapsed by default.

Allowed label examples:

```text
诊断原文
Raw Config Set JSON
```

Rejected behavior:

- Showing the raw Config Set JSON as part of the normal detail flow.
- Using raw JSON as the main display for common object/list fields.

## 6. Implementation guidance

### Preferred implementation approach

1. Inspect `web/src/components/config/ConfigEditView.vue` and related config field renderer/helpers.
2. Find where the displayed value is resolved from a field.
3. Add a controlled resolver with behavior similar to:

```text
resolveFieldValue(configSet, field):
  if field.value_path exists:
    return getByPath(configSet, field.value_path)
  if field.field_key is an alias:
    return getByPath(configSet, aliasMap[field.field_key])
  if field.field_key is a dotted child path:
    return getByPath(configSet, field.field_key)
  return getByPath(configSet, field.internal_key or field.field_key)
```

4. Add a small alias map for current Docker aliases only if needed:

```text
docker.shm_size -> launcher.docker_options.shm_size
docker.privileged -> launcher.docker_options.privileged
docker.ipc_mode -> launcher.docker_options.ipc_mode
docker.network_mode -> launcher.docker_options.network_mode
docker.devices -> launcher.docker_options.devices
docker.optional_devices -> launcher.docker_options.optional_devices
docker.group_add -> launcher.docker_options.group_add
```

5. Add display formatting helpers for object/list values.
6. Collapse raw Config Set JSON.
7. Add tests.

### Backend adjustment rule

Prefer frontend value resolution if the backend already provides enough metadata.

Modify backend/config-edit generation only if the frontend cannot unambiguously determine the intended value path. If backend is changed, keep it minimal: add `value_path` or equivalent metadata. Do not change the ConfigSet storage model.

## 7. Acceptance criteria

### Object child fields

1. `UTS mode` does not display `gpu_capabilities`.
2. `Network mode` does not display `gpu_capabilities`.
3. `Shared memory` displays `16gb`.
4. `IPC mode` displays `host`.
5. Missing optional Docker fields are hidden or show `未配置` / `Not configured`.
6. No Docker option child field displays the whole `launcher.docker_options` object.

### Structured display

7. `Model mount` displays `/models (read-only)` or equivalent structured text.
8. `Environment variables` displays `CUDA_VISIBLE_DEVICES = {{vendor_visible_devices}}` in key-value form.
9. `Health check` displays method/type/path/timeout/status in readable form, not raw JSON.
10. `Entrypoint` displays `vllm serve`.
11. `Command` displays `--model {{model_container_path}}`.
12. Empty `Ports`, `Volumes`, `Devices`, `Ulimits`, `Security options` show `未配置` / `Not configured` or are hidden.
13. Raw Config Set JSON is collapsed by default.

### Regression

14. Runtime template details still show sections.
15. User runtime config details still show sections.
16. No OOM/watch loop regression.
17. Existing runtime clone/display/version fixes remain intact.
18. Existing probe summary/raw evidence fix remains intact.

## 8. Required tests

### Frontend tests

Add or update tests for `ConfigEditView` or the relevant helper/component:

1. Object child fields read subkey values rather than parent object.
2. Alias `docker.shm_size` resolves to `launcher.docker_options.shm_size`.
3. Alias `docker.ipc_mode` resolves to `launcher.docker_options.ipc_mode`.
4. `UTS mode` and `Network mode` do not contain `gpu_capabilities`.
5. Missing optional child value renders `未配置` / `Not configured` or is hidden.
6. `runtime.model_mount` renders structured text.
7. `runtime.env` renders key-value rows.
8. `runtime.health` renders summary text.
9. Raw Config Set JSON is collapsed by default.

### Backend tests

Backend tests are only required if backend config-edit metadata is changed. If backend adds `value_path`, test that object child fields carry the expected value path.

### Required commands

```bash
go test ./internal/server/...
cd web && npm test
cd web && npm run build
```

## 9. Review checklist

Before commit, verify:

- Search rendered output/tests for `UTS mode` and `gpu_capabilities`; they must not be associated.
- Search rendered output/tests for `Network mode` and `gpu_capabilities`; they must not be associated.
- Normal detail view does not show raw JSON for model mount, env, health, command, entrypoint.
- Raw Config Set JSON is present only in collapsed diagnostics.
- No unrelated architecture changes.

## 10. Closeout output required

Claude must output:

1. Root cause.
2. Modified files.
3. Implementation summary.
4. Test commands and results.
5. Commit id.
6. Push result.
7. `git status --short`.
8. Any remaining issue, only if it is truly outside the defined scope.
