# Backend Catalog Vendor Extension

LightAI Go now uses the target catalog path:

```text
configs/backend-catalog/
configs/backend-catalog.d/
```

Built-in catalog entries are system-managed. Add site-specific entries under:

```text
configs/backend-catalog.d/custom-backends/
configs/backend-catalog.d/custom-versions/
configs/backend-catalog.d/custom-runtimes/
```

## Runtime Boundary

Use `BackendVersion` for backend software version and parameter schema.

Use `BackendRuntime` for vendor/runtime-specific execution details:

- image
- command/args override
- docker options
- device policy
- mount policy
- health check override
- verification status

Use `NodeBackendRuntime` for per-node readiness:

- image present
- Docker available
- driver/toolkit evidence
- vendor adapter/device support

## MetaX / MuXi

MetaX Docker options belong in Runtime templates, not DockerExecutor code.

The built-in MetaX templates include:

- `/dev/dri`
- `/dev/mxcd`
- `/dev/infiniband`
- `group_add: video`
- `uts_mode: host`
- `ipc_mode: host`
- `privileged: true`
- `security_opt: seccomp=unconfined`
- `security_opt: apparmor=unconfined`
- `shm_size: 100gb`
- `ulimits.memlock: -1`
- `CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}`

`/dev/mem` is documented only as an optional high-risk device and is not enabled by default.

## Huawei / Ascend

Huawei runtime templates are present but marked:

```yaml
verification:
  status: template_only
```

NodeBackendRuntime checks must not mark Huawei/Ascend ready until a Huawei vendor adapter and real hardware validation are available. Expected statuses include:

- `template_only`
- `adapter_missing`
- `unsupported_device`

The runtime template reserves:

- `/dev/davinci_manager`
- `/dev/devmm_svm`
- `/dev/hisi_hdc`
- `/dev/davinci{{index}}`
- `/usr/local/dcmi`
- `/usr/local/bin/npu-smi`
- `/usr/local/Ascend/driver/lib64`
- `/usr/local/Ascend/driver/version.info`
- `/etc/ascend_install.info`
- `ASCEND_VISIBLE_DEVICES={{vendor_visible_devices}}`

## Adding A New Vendor

1. Add a runtime YAML under `configs/backend-catalog.d/custom-runtimes/`.
2. Use stable `id` and `slug`.
3. Keep node-specific image or readiness in NodeBackendRuntime, not BackendRuntime.
4. Add a vendor adapter only when device discovery, visible-device mapping, or monitoring requires vendor-specific logic.
5. Keep high-risk Docker options disabled unless the vendor runtime requires them and the risk is visible in Web.

