# LightAI Go Runtime Template Detail MHTML Review

Source file: `LightAI Go.mhtml`  
Page: `/runtimes`  
Visible object: `vLLM NVIDIA Docker - 用户配置`

## 1. Conclusion

The latest fix solved the blank parameter section, but the runtime template detail page still has a second class of bugs: **nested runtime config objects are being split into child fields, but child fields are still reading/rendering the parent object value**.

This is why fields such as `UTS mode` and `Network mode` both show:

```json
{"gpu_capabilities":[["gpu"]],"gpu_driver":"","ipc_mode":"host","shm_size":"16gb"}
```

That value is the whole `launcher.docker_options` object, not the value of UTS mode or network mode.

This is not a normal Docker setting and should not be accepted as correct UI behavior.

## 2. User-visible issues found in the MHTML

### 2.1 UTS mode shows the wrong value

Rendered field:

- `data-field-key="launcher.docker_options.uts_mode"`
- `data-internal-key="launcher.docker_options"`
- visible value:

```json
{"gpu_capabilities":[["gpu"]],"gpu_driver":"","ipc_mode":"host","shm_size":"16gb"}
```

`UTS mode` is a Docker/Linux namespace option controlling hostname/domainname namespace behavior. Typical values are blank/default or `host`. It should never display GPU capabilities, GPU driver, IPC mode, or shared memory as its value.

### 2.2 Network mode shows the same wrong value

Rendered field:

- `data-field-key="docker.network_mode"`
- `data-internal-key="launcher.docker_options"`
- visible value:

```json
{"gpu_capabilities":[["gpu"]],"gpu_driver":"","ipc_mode":"host","shm_size":"16gb"}
```

`Network mode` is a Docker networking option, such as `bridge`, `host`, `none`, or container-network mode. It should not show the whole Docker options object.

### 2.3 Security options and Ulimits render as broken or confusing controls

Rendered fields include:

- `launcher.docker_options.security_options`
- `launcher.docker_options.ulimits`

The MHTML shows these as empty/broken controls instead of a clear value such as `未配置`, a compact list, or a hidden optional advanced field.

### 2.4 Model mount is shown as raw JSON

Rendered field:

```json
{"container_path":"/models","readonly":true}
```

This is not as severe as the UTS/Network bug because the value belongs to `runtime.model_mount`, but the UI is still not productized. It should display something like:

```text
容器路径：/models
只读：是
```

or:

```text
/models (read-only)
```

For deployment/runtime previews, the final model mount should eventually show host/model source path -> container path.

### 2.5 Environment variables and health check are also raw JSON

Examples visible in the MHTML:

```json
{"CUDA_VISIBLE_DEVICES":"{{vendor_visible_devices}}"}
```

```json
{"path":"/v1/models","startup_timeout_seconds":120,"success_status":[200],"type":"http"}
```

These should be rendered as key-value or summary views, not raw JSON in the normal details view.

### 2.6 Command, entrypoint, ports, volumes are shown as raw arrays

Examples:

```json
["--model","{{model_container_path}}"]
["vllm","serve"]
[]
```

These are valid internal values, but the details UI should render them in a readable way. For example:

```text
Entrypoint: vllm serve
Command: --model {{model_container_path}}
Ports: 未配置
Volumes: 未配置
```

### 2.7 Advanced raw config is visible in the normal details flow

The page includes a large `Config Set JSON` block. It can be useful for diagnostics, but it should be under a clearly named diagnostic section and default collapsed, not presented as part of the normal runtime configuration detail.

## 3. Evidence from the rendered DOM

The MHTML contains 25 rendered config fields. The important pattern is:

| Field key | Internal key | Observed issue |
|---|---|---|
| `docker.shm_size` | `launcher.docker_options` | child field points to parent object |
| `docker.privileged` | `launcher.docker_options` | child field points to parent object |
| `docker.ipc_mode` | `launcher.docker_options` | child field points to parent object |
| `launcher.docker_options.uts_mode` | `launcher.docker_options` | displays entire parent object |
| `docker.network_mode` | `launcher.docker_options` | displays entire parent object |
| `launcher.docker_options.security_options` | `launcher.docker_options` | broken/empty structured control |
| `launcher.docker_options.ulimits` | `launcher.docker_options` | broken/empty structured control |
| `docker.devices` | `launcher.docker_options` | child field points to parent object |
| `docker.optional_devices` | `launcher.docker_options` | child field points to parent object |
| `docker.group_add` | `launcher.docker_options` | child field points to parent object |

The raw config set contains only this value for `launcher.docker_options`:

```json
{
  "gpu_capabilities": [["gpu"]],
  "gpu_driver": "",
  "ipc_mode": "host",
  "shm_size": "16gb"
}
```

Therefore fields such as `uts_mode`, `network_mode`, `security_options`, `ulimits`, `devices`, `optional_devices`, and `group_add` either do not have configured values or are not present in this runtime config. They should not display the whole parent object.

## 4. Technical diagnosis

### 4.1 The parameter section is no longer empty, but object-child value binding is wrong

The previous fix made `ConfigEditView` render. The current bug is one layer deeper:

```text
ConfigSet item: launcher.docker_options = object
UI projected child fields: docker.shm_size, docker.ipc_mode, docker.network_mode, launcher.docker_options.uts_mode, ...
Renderer internal key: launcher.docker_options
Actual display value: whole launcher.docker_options object
```

The renderer needs a value path or child key mapping. It should read:

```text
launcher.docker_options.shm_size
launcher.docker_options.ipc_mode
launcher.docker_options.network_mode
launcher.docker_options.uts_mode
```

not:

```text
launcher.docker_options
```

for every child field.

### 4.2 Field key naming is inconsistent

The same logical group uses both:

```text
docker.shm_size
docker.ipc_mode
docker.network_mode
```

and:

```text
launcher.docker_options.uts_mode
launcher.docker_options.security_options
launcher.docker_options.ulimits
```

This inconsistency is likely contributing to incorrect mapping and should be normalized.

Recommended canonical naming:

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
```

If user-facing aliases like `docker.shm_size` are kept, they must map cleanly to canonical value paths.

### 4.3 Optional empty Docker options should not appear as broken fields

Most Docker options are advanced. In template details, if a field is not configured and not enabled, it should either be hidden or shown as `未配置`, not as an empty/broken input/table.

### 4.4 Some values belong in dedicated display components

The following internal values should not be raw JSON in normal detail view:

- `runtime.model_mount`
- `runtime.env`
- `runtime.health`
- `launcher.command`
- `launcher.entrypoint`
- `launcher.ports`
- `launcher.volumes`
- `launcher.devices`

They need compact renderers.

## 5. Required fix scope

This should be treated as a follow-up bug batch after commit `7671a3e`.

### P0: Fix object child value resolution

1. Audit `ConfigEditView.vue` and the config edit field renderer.
2. Add explicit support for child value paths when a projected UI field maps to an object item.
3. A child field must not render the whole parent object as its value.
4. If the child key is absent, render blank/未配置 or hide according to presentation rules.

### P0: Normalize Docker options field mapping

1. Pick one canonical namespace for Docker options, preferably `launcher.docker_options.<field>`.
2. Ensure each child field maps to the correct object subkey.
3. Remove or map inconsistent aliases such as `docker.network_mode` if they are only UI aliases.

### P0: Hide or summarize empty advanced Docker fields

Fields such as UTS mode, network mode, security options, ulimits, optional devices, and additional groups should not clutter the default template detail page when not configured.

### P1: Productize object/list display

Add compact display renderers for:

- model mount
- env
- health check
- command
- entrypoint
- ports
- volumes
- devices
- ulimits
- security options

### P1: Keep raw Config Set JSON diagnostic-only

Raw Config Set JSON should remain available for debugging, but default collapsed under a diagnostic label.

## 6. Acceptance criteria

1. `UTS mode` does not display `{"gpu_capabilities":...}`.
2. `Network mode` does not display `{"gpu_capabilities":...}`.
3. `Shared memory` displays `16gb`.
4. `IPC mode` displays `host`.
5. Optional empty fields such as UTS mode, network mode, security options, ulimits, devices, optional devices, and additional groups are hidden or displayed as `未配置`.
6. `Model mount` displays `/models (read-only)` or equivalent structured text, not raw JSON.
7. `Environment variables` displays `CUDA_VISIBLE_DEVICES = {{vendor_visible_devices}}` in key-value form.
8. `Health check` displays `HTTP /v1/models`, timeout `120s`, success status `200`.
9. `Entrypoint` displays `vllm serve`.
10. `Command` displays `--model {{model_container_path}}`.
11. `Ports`, `Volumes`, `Devices` display `未配置` when empty, not `[]` as raw JSON.
12. Raw Config Set JSON is default collapsed under diagnostics.
13. Frontend tests assert that UTS/Network do not show the parent Docker options object.
14. Existing Go tests and frontend tests pass.

## 7. Suggested tests

### Frontend tests

Add or update tests for `ConfigEditView`:

- projected object child fields read subkey values, not parent object
- absent child value renders blank/未配置 or is hidden
- `docker.shm_size`/`launcher.docker_options.shm_size` displays `16gb`
- `docker.ipc_mode`/`launcher.docker_options.ipc_mode` displays `host`
- `UTS mode` and `Network mode` do not contain `gpu_capabilities`
- `runtime.model_mount` renders structured display
- `runtime.env` renders key-value display
- `runtime.health` renders summary display
- raw Config Set JSON is collapsed by default

### API/config tests

Add tests around config edit view generation:

- child fields from object config items have explicit value paths
- field keys are canonical or have explicit aliases
- optional absent Docker option fields are not presented as enabled values

## 8. Short Claude execution prompt

```text
请修复 runtime 模板详情页第二阶段问题：ConfigEditView 已能显示参数，但 Docker options 等 object 子字段取值/展示错误。

先阅读：
docs/reports/phase-3/runtime-config-display-probe-fix/05-closeout.md
本次 mhtml 复核报告：docs/reports/phase-3/runtime-config-display-probe-fix/06-mhtml-config-field-review.md

目标：
1. 修复 object 子字段值解析，避免 UTS mode / Network mode 显示整个 launcher.docker_options 对象。
2. 统一 Docker options 子字段映射，优先使用 launcher.docker_options.<field>。
3. 空的高级 Docker 字段默认隐藏或显示“未配置”，不能显示父对象。
4. Model mount、Environment variables、Health check、Command、Entrypoint、Ports、Volumes 等正常详情页不要裸 JSON，改为结构化摘要。
5. Raw Config Set JSON 保留在诊断区，默认收起。
6. 补前端测试覆盖 UTS/Network 不显示 gpu_capabilities，shm_size 显示 16gb，ipc_mode 显示 host，model mount/env/health 有结构化展示。

限制：
- 不新建分支。
- 不重做 runtime/config 架构。
- 不做完整 Docker 参数体系重构。
- 先修当前可见问题和同类 object/list 展示问题。

完成后运行：
go test ./internal/server/...
cd web && npm test
cd web && npm run build

输出根因、修改文件、测试结果、commit id、push 结果、git status --short。
```
