# Proposed ConfigEdit Component Template Contract

This file defines a proposed external template contract. It is not final API; Codex should audit the current codebase and propose the smallest coherent implementation path.

## 1. Top-level template

```yaml
template_id: vllm-nvidia-docker-configedit-v1
kind: config_edit_template
version: 1
applies_to:
  backend: vllm
  backend_versions:
    - latest
  runtime_kind: docker
  vendors:
    - nvidia

metadata:
  display_name: vLLM NVIDIA Docker
  description: Config editor template for vLLM OpenAI-compatible server on NVIDIA Docker.
  source: built_in

views:
  default_view: normal
  supported_views:
    - normal
    - advanced
    - developer

layers:
  backend_runtime:
    editable: true
  node_backend_runtime:
    editable: true
    copy_from_parent: copy_effective_snapshot
  deployment:
    editable: true
    copy_from_parent: copy_effective_snapshot

sections:
  - key: basic
    label: Basic
    order: 10
    view: normal
    components: []

  - key: resources
    label: Resources
    order: 20
    view: normal
    components: []

  - key: service
    label: Service
    order: 30
    view: normal
    components: []

  - key: runtime
    label: Runtime Parameters
    order: 40
    view: normal
    components: []

  - key: storage
    label: Storage & Mounts
    order: 50
    view: advanced
    components: []

  - key: advanced
    label: Advanced
    order: 90
    view: advanced
    collapsed: true
    components: []

  - key: developer
    label: Developer / Debug
    order: 100
    view: developer
    collapsed: true
    components: []
```

## 2. Generic component fields

Each component should support:

```yaml
key: model_runtime.gpu_memory_utilization
component: number
label: GPU Memory Utilization
description: Fraction of GPU memory reserved for model execution.
section: resources
view: normal
order: 20

value:
  type: number
  default: 0.9
  recommended: "0.85 - 0.95"
  min: 0.1
  max: 1.0
  step: 0.01

enabled:
  default: true
  editable: true

editability:
  backend_runtime: true
  node_backend_runtime: true
  deployment: true

copy:
  to_child: true
  strategy: copy_effective_value

reset:
  allow_reset_to_parent: true
  allow_reset_to_default: true

validation:
  - type: range
    min: 0.1
    max: 1.0
    severity: error

help:
  default_text: "Default: 0.9"
  recommended_text: "Recommended: 0.85 - 0.95"
  effect_text: "Effect: --gpu-memory-utilization"

effects:
  cli:
    flag: --gpu-memory-utilization
    value_from: self.value
    omit_if_disabled: true
    omit_if_empty: true
```

## 3. Device binding component

```yaml
key: runtime.device_binding
component: accelerator_binding
label: Device Binding
description: Controls accelerator selection and container device visibility.
section: resources
view: normal
order: 10

value:
  type: object
  fields:
    enabled:
      type: boolean
      default: true
      label: Enable device binding

    mode:
      type: select
      default: auto
      options:
        - auto
        - manual
        - disabled
        - inherited
      label: Binding mode

    vendor:
      type: select
      default: nvidia
      options:
        - nvidia
        - metax
        - huawei
        - cpu
        - none
      label: Vendor

    accelerator_ids:
      type: string_array
      default: ["0"]
      label: Accelerator IDs
      description: IDs visible to the runtime, such as 0 or 0,1.

    accelerator_count:
      type: number
      default: 1
      min: 0
      step: 1
      label: Accelerator count

    docker_gpu_option:
      type: string
      default_template: "device={{ accelerator_ids | join(',') }}"
      label: Docker GPU option
      description: Value passed to Docker --gpus for NVIDIA.

    visible_env_key:
      type: string
      default: CUDA_VISIBLE_DEVICES
      label: Visible devices env key

    visible_env_value:
      type: string
      default_template: "{{ accelerator_ids | join(',') }}"
      label: Visible devices env value

editability:
  backend_runtime: true
  node_backend_runtime: true
  deployment: true

copy:
  to_child: true
  strategy: copy_effective_object

validation:
  - type: required_if
    when: "enabled == true and mode == manual"
    field: accelerator_ids
    severity: error

views:
  normal_fields:
    - enabled
    - mode
    - vendor
    - accelerator_ids
  advanced_fields:
    - accelerator_count
    - docker_gpu_option
    - visible_env_key
    - visible_env_value

effects:
  docker:
    gpus:
      when: "enabled == true and vendor == 'nvidia' and mode != 'disabled'"
      value_from: docker_gpu_option
  env:
    - when: "enabled == true and visible_env_key != ''"
      key_from: visible_env_key
      value_from: visible_env_value
```

## 4. Port component

```yaml
key: service.port_binding
component: port_binding
label: Service Port
section: service
view: normal

value:
  type: object
  fields:
    host_port:
      type: number
      default: 8000
      min: 1
      max: 65535
    container_port:
      type: number
      default: 8000
      min: 1
      max: 65535
    protocol:
      type: select
      default: tcp
      options: [tcp, udp]

editability:
  backend_runtime: true
  node_backend_runtime: true
  deployment: true

copy:
  to_child: true
  strategy: copy_effective_object

effects:
  docker:
    port_mapping:
      host_port_from: host_port
      container_port_from: container_port
      protocol_from: protocol
  cli:
    - flag: --port
      value_from: container_port
```

## 5. Health check component

```yaml
key: runtime.health_check
component: health_check
label: Health Check
section: service
view: advanced

value:
  type: object
  fields:
    path:
      type: string
      default: /v1/models
    expected_status:
      type: number
      default: 200
    startup_timeout_seconds:
      type: number
      default: 120
      min: 1
    interval_seconds:
      type: number
      default: 5
      min: 1
    timeout_seconds:
      type: number
      default: 3
      min: 1

editability:
  backend_runtime: true
  node_backend_runtime: true
  deployment: true

copy:
  to_child: true
  strategy: copy_effective_object
```

## 6. Developer/debug raw template component

Raw command templates and internal keys should be allowed, but they must be developer-view only by default.

```yaml
key: developer.raw_command_template
component: raw_template_viewer
label: Raw Command Template
section: developer
view: developer
readonly: true
```

## 7. ConfigEdit object materialized from template

A materialized ConfigEdit object should expose fields/components like:

```json
{
  "object_kind": "deployment",
  "object_id": "...",
  "template_id": "vllm-nvidia-docker-configedit-v1",
  "parent": {
    "object_kind": "node_backend_runtime",
    "object_id": "...",
    "snapshot_id": "..."
  },
  "sections": [],
  "components": [
    {
      "key": "runtime.device_binding",
      "component": "accelerator_binding",
      "label": "Device Binding",
      "description": "Controls accelerator selection and container device visibility.",
      "value": {
        "enabled": true,
        "mode": "manual",
        "vendor": "nvidia",
        "accelerator_ids": ["0"],
        "docker_gpu_option": "device=0",
        "visible_env_key": "CUDA_VISIBLE_DEVICES",
        "visible_env_value": "0"
      },
      "source": {
        "kind": "copied_from_parent",
        "layer": "node_backend_runtime",
        "overridden": false
      },
      "editable": true,
      "patch_target": "deployment.config.runtime.device_binding",
      "effects_preview": [
        "docker --gpus \"device=0\"",
        "env CUDA_VISIBLE_DEVICES=0"
      ]
    }
  ]
}
```

## 8. RunPlan compiler rule

RunPlan compiler should consume the final materialized ConfigEdit snapshot and effects. It should not invent hidden runtime-affecting fields.

Allowed responsibilities:

- evaluate template expressions from current snapshot values
- validate final required fields
- assemble DockerSpec
- build command preview
- produce source/effect map
- produce diagnostics

Disallowed responsibilities:

- silently add `--gpus` not present in ConfigEdit/effects
- silently add `CUDA_VISIBLE_DEVICES` not present in ConfigEdit/effects
- silently override user-edited fields
- treat source map raw values as final errors
- make page-specific assumptions about backend parameters
