# Runtime Devices / Model Mount / Additional Volumes — Final Design

## Product model

### Devices

Meaning:

```text
Docker --device pass-through list.
```

Fields:

```yaml
enabled: boolean
items:
  - host_device_path: string
    container_device_path: string optional
    permissions: string optional, default rwm
```

Rules:

1. `container_device_path` defaults to `host_device_path`.
2. `permissions` defaults to `rwm`.
3. Device path existence is diagnostic-only.
4. Missing path produces warning if it can be checked.
5. Missing path must not block ready/preflight/deploy.
6. Invalid device entry shape can block only when Docker run spec cannot be constructed.
7. No `readonly` field.
8. No `Optional devices` field.

### Model mount

Meaning:

```text
Model artifact path mounted into the runtime container.
```

Fields:

```yaml
host_path: string
container_path: string
readonly: boolean default true
```

Rules:

1. Default `readonly=true`.
2. It should be the primary way model files reach the serving container.
3. It should not be used for broad work/cache/data mounts.
4. It should not default to `/mnt:/mnt`.

### Additional volumes

Meaning:

```text
Extra user or template-defined bind mounts.
```

Fields:

```yaml
enabled: boolean
items:
  - host_path: string
    container_path: string
    readonly: boolean
```

Rules:

1. Use it for `/mnt:/mnt`, cache directories, work directories, or vendor-specific extra filesystem paths.
2. It can be read-write or read-only.
3. It must stay separate from Model mount.

## Catalog defaults

### NVIDIA

Expected defaults:

```yaml
devices:
  enabled: false
  items: []
```

Rationale:

- NVIDIA GPU selection is handled through the existing GPU allocation/device binding logic and NVIDIA container runtime integration.
- Manual `--device` is an advanced override.

### MetaX / 沐曦

Expected defaults:

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

Expected Docker options:

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

Additional volume `/mnt:/mnt`:

```text
Do not enable by default unless the template intentionally needs it for production serving.
If added, put it under Additional volumes, not Model mount.
```

### Huawei / Ascend

Expected behavior:

1. Follow actual vendor runtime requirements.
2. If explicit device pass-through is required, use the same single Devices field.
3. Do not introduce Optional devices.
4. Do not show device entries as volumes.

## Config taxonomy and UI mapping

Canonical user-facing paths should be consistent. Prefer paths such as:

```text
launcher.docker_options.devices
runtime.model_mount
launcher.docker_options.volumes or runtime.additional_volumes
launcher.docker_options.privileged
launcher.docker_options.cap_add
launcher.docker_options.security_options
launcher.docker_options.ulimits
launcher.docker_options.group_add
launcher.docker_options.network_mode
launcher.docker_options.shm_size
```

Avoid showing both:

```text
docker.devices
launcher.devices
launcher.docker_options.devices
```

as separate user-facing fields. If legacy paths exist, normalize them into one canonical field before projection.

## UI widgets

### device_mapping_table

Columns:

```text
Enabled
Host device path
Container device path
Permissions
Warning/status, optional
```

Help text:

```text
Devices are passed to Docker as --device entries. They are hardware/device nodes, not filesystem mounts.
```

### model_mount_form

Fields:

```text
Host model path
Container model path
Readonly
```

Help text:

```text
Model files are mounted read-only by default to protect model artifacts. Use Additional volumes for read/write data, cache, or work directories.
```

### volume_mapping_table

Columns:

```text
Host path
Container path
Readonly
```

Help text:

```text
Additional volumes are filesystem bind mounts. They are separate from Devices and Model mount.
```

## RunPlan behavior

The resolved Docker run spec must include:

- `HostConfig.Devices` or equivalent for Devices entries when enabled.
- Bind mounts for Model mount and Additional volumes.
- `readonly` only for mounts.
- Device permissions for Devices.

Warnings:

- Device missing warnings can be included in RunPlan diagnostics.
- Missing device warnings must not change `can_run` to false.
- Only invalid spec construction should produce a blocking resolve error.

## Backward cleanup

Because historical compatibility is not required for this project:

1. Remove legacy `optional_devices` catalog fields.
2. Remove or normalize legacy device paths that create duplicate normal UI fields.
3. Update seed/catalog tests.
4. Rebuild local DB for manual verification after catalog changes.
