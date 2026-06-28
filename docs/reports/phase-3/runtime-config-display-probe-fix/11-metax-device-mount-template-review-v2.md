# MetaX Runtime Template Device / Mount Review — Revised V2

## User-provided reference command

```bash
export IMAGE=cr.metax-tech.com/public-ai-release/maca/vllm:maca.ai3.1.0.7-torch2.6-py310-ubuntu22.04-amd64

sudo docker run -it --privileged --cap-add=SYS_PTRACE --security-opt seccomp=unconfined \
      --device=/dev/dri --device=/dev/mxcd --device=/dev/mem --group-add video --name vllm_metax --network=host \
      --security-opt apparmor=unconfined --shm-size '100gb' --ulimit memlock=-1 \
      -v /mnt:/mnt \
      $IMAGE \
      /bin/bash
```

Notes:

- This is a useful runtime environment reference.
- `/bin/bash` is an interactive debug command and must not become the default serving entrypoint.
- Serving templates should still generate the proper vLLM/SGLang/llama.cpp service command.

## Final interpretation

### Devices

`Devices` means Docker `--device` pass-through mappings.

One product field is enough:

```text
Devices = Docker --device list
```

Do not keep a separate `Optional devices` field. The template decides whether Devices is enabled and what default entries it contains.

### NVIDIA default

NVIDIA runtime templates usually rely on GPU runtime integration and GPU selection rather than hand-authored `--device` entries.

Expected catalog behavior:

```text
Devices.enabled = false
Devices.items = []
```

Special device pass-through is still possible through user override.

### MetaX default

MetaX runtime templates need explicit device pass-through.

Expected catalog behavior:

```text
Devices.enabled = true
Devices.items includes:
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

`/dev/infiniband` is not a separate product concept. If a specific template requires it by default, include it in Devices. If it is only useful for RDMA/IB environments, leave it out and let users add it.

### Device path existence

Device path existence is a diagnostic signal only.

Required behavior:

```text
If a configured host_device_path is missing:
- add warning
- show warning in check/preflight/UI diagnostics when available
- continue allowing ready/preflight/deploy unless the Docker spec cannot be constructed
```

Blocking behavior is allowed only for invalid configuration shape, for example:

```text
- empty host_device_path with no defaulting path
- invalid permissions that cannot be normalized to r/w/m/rwm
- malformed device entry that cannot be converted into Docker run spec
```

### Devices UI

Device mappings should use device-specific labels:

```text
Enabled
Host device path
Container device path
Permissions
```

Defaults:

```text
container_device_path defaults to host_device_path
permissions defaults to rwm
```

Do not show `readonly` for Devices. `readonly` belongs to model mount and volumes.

### Model mount

Model mount is a model artifact directory/file mount.

Expected behavior:

```text
Model mount defaults to readonly=true
```

Reason:

- Inference runtimes should read model artifacts.
- Model artifacts should not be mutated by serving containers.
- Read-write data/cache/work directories should use Additional volumes.

### Additional volumes

`-v /mnt:/mnt` is an extra bind mount. It should be represented as Additional volumes, not Model mount.

Default rule:

```text
Do not add broad /mnt:/mnt read-write volume by default unless the product template explicitly requires it.
```

If present, it should be shown as:

```text
Additional volumes:
- host_path: /mnt
  container_path: /mnt
  readonly: false
```

## MetaX Docker options mapping

| Docker flag | LightAI Go target | Expected behavior |
| --- | --- | --- |
| `--privileged` | `launcher.docker_options.privileged=true` | Advanced/security option, visible in structured summary. |
| `--cap-add=SYS_PTRACE` | `launcher.docker_options.cap_add=["SYS_PTRACE"]` | Structured list. |
| `--security-opt seccomp=unconfined` | `launcher.docker_options.security_options` | Structured list. |
| `--security-opt apparmor=unconfined` | `launcher.docker_options.security_options` | Structured list. |
| `--device=/dev/mxcd` | `launcher.docker_options.devices[]` | Device mapping, permissions `rwm`. |
| `--device=/dev/dri` | `launcher.docker_options.devices[]` | Device mapping, permissions `rwm`. |
| `--device=/dev/mem` | `launcher.docker_options.devices[]` | Device mapping, permissions `rwm`. |
| `--group-add video` | `launcher.docker_options.group_add=["video"]` | Structured list. |
| `--network=host` | `launcher.docker_options.network_mode="host"` | Host network means Docker port publishing is not applicable. |
| `--shm-size 100gb` | `launcher.docker_options.shm_size="100gb"` | MetaX default may differ from NVIDIA. |
| `--ulimit memlock=-1` | `launcher.docker_options.ulimits[]` | Structured ulimit item, soft=-1, hard=-1. |
| `-v /mnt:/mnt` | Additional volumes | Do not treat as model mount. |
| `/bin/bash` | Debug shell command | Do not use as production serving entrypoint. |

## Current regression symptoms to fix

- `Devices` and `Optional devices` are both visible.
- Device fields use mount-like UI and may show readonly.
- Empty raw `launcher.devices` and raw `launcher.volumes` appear in advanced raw config.
- Model mount is confused with broad bind mounts.
- MetaX catalog does not express all required Docker options clearly.

## Acceptance

- Only one user-facing Devices field remains.
- Optional devices is removed from normal UI and catalog taxonomy.
- Devices uses device field labels and `permissions`, not `readonly`.
- NVIDIA catalog default has Devices disabled/empty.
- MetaX catalog default has Devices enabled with `/dev/mxcd`, `/dev/dri`, `/dev/mem`.
- Device path missing warnings do not block deployment.
- Model mount remains read-only.
- Additional volumes are separate.
