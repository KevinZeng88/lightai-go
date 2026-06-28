# Implementation Steps

This plan is intentionally ordered. Do not jump directly to UI polish before fixing taxonomy/catalog/runtime invariants.

## Step 0 — Baseline and inventory

Run:

```bash
git status --short
grep -R "optional_devices" -n internal web docs || true
grep -R "model_runtime.port\|model_runtime.host\|model_runtime.model" -n internal web || true
grep -R "unsupported runtime_type\|runtime_type" -n internal web || true
grep -R "source_metadata\|raw config\|config_json" -n web/src/pages web/src/components || true
```

Record where these are found in the completion report.

## Step 1 — Normalize runtime device/volume taxonomy

Likely files:

```text
internal/server/configedit/taxonomy.go
internal/server/configedit/project.go
internal/server/configedit/*test.go
web/src/components/config/ConfigEditView.vue
web/src/components/config/*renderer*.vue
web/src/components/config/__tests__/ConfigEditView.render.test.ts
```

Required changes:

1. Remove `Optional devices` from normal user taxonomy.
2. Use one canonical Devices field.
3. Devices widget must use:

```text
host_device_path
container_device_path
permissions
```

4. Remove `readonly` from Devices projection/rendering.
5. Ensure Model mount and Additional volumes use separate widgets.
6. Ensure raw `launcher.devices` / `launcher.volumes` do not appear as ordinary user fields when empty or duplicated.

Implementation pattern:

- Normalize legacy fields during config projection.
- If old config has `optional_devices`, merge into `devices.items` only if the existing code needs backward cleanup for current seed data, then remove the separate user-facing field.
- Since historical DB compatibility is not required, clean catalog and rebuild DB for manual verification.

## Step 2 — Fix catalog templates

Likely files:

```text
internal/server/catalog/**/*.yaml
internal/server/catalog/**/*runtime*.yaml
internal/server/seed* or catalog seed tests
```

Search:

```bash
grep -R "optional_devices\|devices:" -n internal/server | head -80
grep -R "metax\|maca\|mxcd\|nvidia\|ascend\|huawei" -n internal/server/catalog internal || true
```

Required NVIDIA catalog behavior:

```yaml
devices:
  enabled: false
  items: []
```

Required MetaX catalog behavior:

```yaml
devices:
  enabled: true
  items:
    - host_device_path: /dev/mxcd
      container_device_path: /dev/mxcd
      permissions: rwm
    - host_device_path: /dev/dri
      container_device_path: /dev/dri
      permissions: rwm
    - host_device_path: /dev/mem
      container_device_path: /dev/mem
      permissions: rwm
```

Required MetaX docker options:

```yaml
privileged: true
cap_add:
  - SYS_PTRACE
security_options:
  - seccomp=unconfined
  - apparmor=unconfined
network_mode: host
shm_size: 100gb
ulimits:
  - name: memlock
    soft: -1
    hard: -1
group_add:
  - video
```

Do not add `/mnt:/mnt` to Model mount.

If `/mnt:/mnt` is added, it must be under Additional volumes and must be justified in the completion report. Default recommendation: leave it out.

## Step 3 — Make device existence warning-only

Likely files:

```text
internal/server/api/preflight_handlers.go
internal/server/runplan* or internal/server/*runplan*
internal/agent/*docker* or internal/server/docker spec builder
```

Search:

```bash
grep -R "device" -n internal/server internal/agent | head -120
grep -R "missing.*device\|device.*missing\|os.Stat" -n internal/server internal/agent || true
```

Required behavior:

1. If a device path can be checked and is missing, add a warning.
2. Do not set deployment/preflight `can_run=false` only because the device path is missing.
3. Continue constructing Docker run spec with the configured device entry.
4. Block only malformed entries that cannot be converted to Docker spec.

Expected test:

```text
device path missing produces warning but preflight/runplan remains deployable
```

## Step 4 — Runtime template list operations

Likely file:

```text
web/src/pages/BackendRuntimesPage.vue
web/src/pages/__tests__/BackendRuntimesPage.integration.test.ts
```

Required behavior:

1. Runtime template list action column includes:

```text
View
Edit
Copy as user config
```

2. Row click or View opens readonly detail.
3. Edit opens edit mode directly.
4. Detail can keep an Edit button.
5. System/builtin template edit follows current product rule:
   - hidden/disabled if direct edit is disallowed.
   - Copy as user config remains available.
6. User runtime config edit is available directly.
7. Do not restore row-click-to-edit.

## Step 5 — Deployment wizard reset

Likely file:

```text
web/src/pages/ModelDeploymentsPage.vue
```

Required functions:

```ts
function createEmptyDeploymentDraft(): DeploymentDraft
function resetDeploymentWizard(): void
function openCreateDeployment(): void
function closeDeploymentDrawer(): void
function cancelDeploymentWizard(): void
```

Required behavior:

- `openCreateDeployment()` calls reset before open.
- Save success calls reset after close/reload.
- Cancel calls reset.
- Drawer close calls reset.
- Route/component unmount calls reset if needed.
- Save failure may keep draft open; after close, the next New must start clean.

Avoid:

- Reusing the same nested reactive object.
- Retaining previous `preflightResult`, `runPlanPreview`, `configOverrides`, `currentStep`, or selected NBR.

## Step 6 — Deployment detail/edit and raw JSON cleanup

Likely files:

```text
web/src/pages/ModelDeploymentsPage.vue
web/src/components/config/ConfigEditView.vue
web/src/components/common/JsonViewer.vue
web/src/api/deployments.ts or equivalent
internal/server/api/deployment_lifecycle_handlers.go
```

Required behavior:

1. Deployment list/detail provides Edit entry.
2. Detail default shows structured sections.
3. Raw config JSON / source metadata JSON are diagnostic sections collapsed by default.
4. Edit uses structured ConfigEditView or equivalent deployment override form.
5. RunPlan preview action remains available.

If backend DTO lacks necessary structured fields, add them through API rather than forcing the UI to parse raw JSON.

## Step 7 — Fix `runtime_type` source of truth

Likely backend areas:

```text
internal/server/api/deployment_lifecycle_handlers.go
internal/server/api/preflight_handlers.go
internal/server/runplan* or internal/server/*runplan*
internal/server/configedit/project.go
```

Required backend invariant:

```text
Docker deployment resolves runtime_type=docker from selected NBR/runtime snapshot.
```

Fix strategy:

1. Locate the exact source of `[resolve_error] unsupported runtime_type: (only docker is supported)`.
2. Trace runtime_type from:

```text
BackendRuntime catalog -> NodeBackendRuntime snapshot -> Deployment create snapshot -> Preflight/RunPlan -> Start
```

3. Ensure deployment overrides cannot blank out runtime_type.
4. Ensure wizard draft cannot send empty runtime_type.
5. Ensure preflight/preview/start share the same resolved source.
6. Add explicit tests for Docker runtime_type.

Do not add support for non-Docker runtime type in this patch.

## Step 8 — Port canonicalization

Likely files:

```text
internal/server/configedit/taxonomy.go
internal/server/configedit/project.go
internal/server/runplan* or internal/server/*runplan*
web/src/pages/ModelDeploymentsPage.vue
web/src/components/config/__tests__/ConfigEditView.render.test.ts
web/src/pages/__tests__/ModelDeploymentsPage.integration.test.ts
```

Required behavior:

1. User-facing container listen port is `service.container_port`.
2. Hide `model_runtime.port` from ordinary deployment overrides.
3. Derive backend CLI port from service.container_port if needed.
4. Host network displays host port as Not applicable / host network.
5. Bridge/default network displays configured/auto/unconfigured state explicitly.
6. No blank host port cell without explanation.

## Step 9 — Parameter layering

Likely files:

```text
internal/server/configedit/taxonomy.go
internal/server/configedit/project.go
runtime catalog parameter schema files
web/src/components/config/ConfigEditView.vue
```

Required changes:

1. Hide from ordinary deployment form:

```text
model_runtime.model
model_runtime.host
model_runtime.port
model_runtime.download_dir
```

2. Keep common stable parameters visible:

```text
gpu_memory_utilization
max_model_len
tensor_parallel_size
served_model_name, if used
```

3. Put specialized tuning parameters into Advanced/Expert collapsed section or Custom args:

```text
cpu_offload_gb
kv_cache_dtype
max_num_batched_tokens
max_num_seqs
swap_space, if supported by selected version
safetensors_load_strategy
```

4. None of these specialized fields should be required by default.
5. Custom args must appear in RunPlan preview.

## Step 10 — Update docs and closeout

Update existing closeout or create a new one under:

```text
docs/reports/phase-3/runtime-config-display-probe-fix/
```

Include:

- final semantics
- changed files
- tests
- known manual DB rebuild requirement
- commit id
- git status
