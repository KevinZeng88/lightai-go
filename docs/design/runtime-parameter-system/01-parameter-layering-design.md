# 01 - Runtime Parameter Layering Design

Status: discussion draft  
Target repo path: `docs/design/runtime-parameter-system/01-parameter-layering-design.md`

## 1. Design objective

LightAI Go needs a clean parameter system that can support many backends, vendors, and deployment styles without mixing responsibilities.

The runtime path must be traceable:

```text
BackendVersion schema/defaults
  -> BackendRuntime template
  -> NodeBackendRuntime / RunnerConfig
  -> ModelDeployment override
  -> Resolver final config
  -> RunPlan
  -> equivalent docker command
  -> Docker Config / HostConfig
```

The system must prove not only that the final container can run, but that every parameter at every layer is saved, inherited, overridden, disabled, or filtered according to design.

## 2. Core principles

### 2.1 Command templates are not editing surfaces

`command_template` / `default_command` defines how the final command is generated. It is not a user-editable parameter block.

Bad model:

```text
Startup Args textarea:
{{model_container_path}}
--host
0.0.0.0
--port
{{container_port}}
--served-model-name
{{served_model_name}}
```

Correct model:

```text
host = 0.0.0.0
container_port = 8000
served_model_name.enabled = true
served_model_name.value = lightai-qwen3-0.6b
extra_args.enabled = true
extra_args.value = --max-model-len 2048
```

The resolver uses those structured fields to generate:

```text
--host 0.0.0.0 --port 8000 --served-model-name lightai-qwen3-0.6b --max-model-len 2048
```

### 2.2 Required parameters are not optional toggles

Required parameters:

- are automatically enabled;
- have no normal enable checkbox, or a locked-on indicator;
- must have a value from static default, template value, system generation, or user input;
- produce a structured preflight error if missing;
- cannot be disabled by Deployment override.

Optional parameters:

- use `enabled/value`;
- keep `value` even when disabled;
- only enter final args/env/Docker spec when enabled;
- can be copied/cloned with exact enabled/value state.

### 2.3 Best practice defaults are not always enabled

A recommendation does not mean forced enablement.

Recommended policy:

| Category | Enablement | Example |
|---|---|---|
| Required core | enabled, locked | host, container_port, model path |
| Safe recommendation | enabled when project policy requires it, otherwise optional | served_model_name |
| Performance tuning | usually optional disabled with suggested value | gpu_memory_utilization, max_model_len |
| High-risk Docker | disabled unless vendor profile requires it | privileged, host IPC, unconfined seccomp |
| Vendor required | enabled only for matching vendor | Ascend devices, MetaX devices |
| User custom | optional advanced | extra_args, extra_env, extra_docker_options |

### 2.4 Vendor-specific defaults must be orthogonal to backend args

Backend args and vendor runtime settings are different dimensions.

Examples:

- vLLM/SGLang/llama.cpp args: model path, host, port, context length, GPU memory utilization, tensor parallel size.
- NVIDIA vendor settings: Docker DeviceRequests / `--gpus`, optional `NVIDIA_VISIBLE_DEVICES`.
- MetaX vendor settings: `/dev/dri`, `/dev/mxcd`, `/dev/infiniband`, `group_add video`, `ipc=host`, MACA/MCCL env.
- Huawei Ascend vendor settings: `/dev/davinci*`, `/dev/davinci_manager`, `/dev/devmm_svm`, `/dev/hisi_hdc`, `/usr/local/dcmi`, CANN/Ascend env.

A vLLM BackendVersion should not carry `/dev/dri`; a MetaX VendorRuntimeProfile may.

### 2.5 Every final value should be source-traceable

For debugging and evidence, RunPlan should ideally include a debug-only source map, for example:

```json
{
  "args": {
    "host": {
      "value": "0.0.0.0",
      "source": "BackendVersion.default_args_schema",
      "layer": "BackendVersion"
    },
    "served_model_name": {
      "value": "lightai-deploy-name",
      "source": "ModelDeployment.override",
      "layer": "Deployment"
    },
    "max_model_len": {
      "value": "1024",
      "source": "ModelDeployment.override.extra_args",
      "layer": "Deployment"
    }
  }
}
```

This source map should be safe for internal debug/evidence. Sensitive values must be redacted in API/UI/logs.

## 3. Layer definitions

### 3.1 BackendVersion

BackendVersion is a system catalog concept. It defines backend capabilities and schema.

Responsibilities:

- backend identity and version;
- supported model formats;
- command template;
- core args schema;
- common optional args schema;
- backend-specific parameters;
- default internal port;
- health check defaults;
- RuntimeRequirements;
- BackendCapabilityProfile;
- supported vendor/runtime compatibility hints.

Must not contain:

- node path;
- GPU ID;
- host port;
- tenant-specific setting;
- deployment-specific override;
- vendor device defaults.

Example schema idea:

```yaml
args_schema:
  - name: model_container_path
    group: startup
    type: string
    required: true
    user_editable: false
    value: "{{model_container_path}}"
    cli_flag: "--model"
    source: system_generated

  - name: host
    group: startup
    type: string
    required: true
    user_editable: true
    default: "0.0.0.0"
    cli_flag: "--host"

  - name: container_port
    group: startup
    type: integer
    required: true
    user_editable: true
    value: "{{container_port}}"
    cli_flag: "--port"

  - name: served_model_name
    group: startup
    type: string
    required: false
    enabled: true
    default: "lightai-{{model_slug}}"
    cli_flag: "--served-model-name"
    backend: vllm
```

### 3.2 ModelArtifact and ModelLocation

Model layer contains model facts, not runtime instructions.

Allowed fields:

- display name;
- description;
- source;
- format;
- architecture;
- quantization;
- size;
- checksum;
- tokenizer metadata;
- chat template metadata;
- max context / recommended context;
- tags / capabilities;
- ModelLocation node/path metadata;
- location consistency evidence.

Must not contain:

- Docker args;
- backend serve args;
- env;
- devices;
- privileged/security/IPC/shm settings;
- GPU visibility controls.

RunPlan may use model layer for:

- host model path;
- container model path after mount resolution;
- model display name / slug;
- model format compatibility;
- model metadata defaults such as recommended context.

### 3.3 VendorRuntimeProfile

VendorRuntimeProfile is the hardware/vendor runtime dimension.

Responsibilities:

- Docker GPU visibility strategy;
- vendor-specific devices;
- vendor-specific mounts;
- vendor-specific env;
- vendor-required high-risk options;
- vendor-specific health or diagnostics hints;
- vendor runtime compatibility rules.

Examples:

- `nvidia` profile should prefer Docker DeviceRequests / GPU lease.
- `metax` profile may define `/dev/dri`, `/dev/mxcd`, MACA env, and high-risk Docker options if vendor docs require them.
- `huawei` profile may define Ascend device nodes and CANN mounts/env.

VendorRuntimeProfile must not define backend serve args such as `--max-model-len` or `--ctx-size`.

### 3.4 BackendRuntime

BackendRuntime is a reusable template combining:

- BackendVersion;
- runtime type, such as Docker;
- image;
- command style;
- vendor profile;
- default parameter values;
- default env values;
- default Docker options;
- health timeout overrides.

BackendRuntime may be cloned from system catalog to a user-managed runtime template.

BackendRuntime modifications should be inherited by subsequently created NBRs, but must not mutate BackendVersion.

### 3.5 NodeBackendRuntime / RunnerConfig

NBR is the node-specific runtime configuration.

Responsibilities:

- node identity;
- selected backend runtime template;
- node-specific image value if overridden;
- node-specific ports;
- node-specific env;
- node-specific devices;
- node-specific high-risk options;
- resource controls;
- health check override;
- GPU/vendor constraints.

NBR modifications should affect deployments using that NBR, unless overridden by Deployment. NBR must not mutate BackendRuntime.

### 3.6 ModelDeployment override

Deployment override is deployment-local and highest priority.

Responsibilities:

- deployment display name / service name;
- selected model;
- selected NBR / runtime candidate;
- deployment-level overrides for allowed fields;
- host port mapping;
- served model name;
- optional backend args;
- optional env;
- optional resource controls;
- optional high-risk controls if explicitly allowed.

Rules:

- Deployment override must not copy unrelated backend/vendor parameters.
- Deployment override must not mutate NBR.
- Required core parameters may be shown but cannot be disabled.
- Optional parameters use enabled/value.
- Disabled override value is saved but not applied.

### 3.7 Resolver and RunPlan

Resolver combines the layers by priority:

```text
system generated values
+ BackendVersion defaults
+ BackendRuntime template values
+ NBR values
+ Deployment override values
```

Final priority for overlapping fields:

```text
Deployment override > NBR > BackendRuntime > BackendVersion default
```

Resolver responsibilities:

- substitute template values such as `{{container_port}}`, `{{model_container_path}}`, `{{model_slug}}`;
- enforce required parameters;
- filter disabled parameters;
- filter non-matching backend/vendor parameters;
- detect duplicate/conflicting args;
- generate final args/env/ports/devices/high-risk Docker fields;
- generate command preview and equivalent docker command;
- generate structured preflight errors;
- optionally generate source map.

## 4. Parameter schema contract

Each parameter should support these fields, even if the initial implementation uses a subset:

```yaml
name: string
label: string
group: startup | model | performance | capacity | api | env | docker | vendor | advanced
type: string | integer | float | boolean | enum | string_list | key_value | path | size
required: boolean
default: any
value: any
enabled: boolean
user_editable: boolean
cli_flag: string
env_name: string
docker_field: string
backend: vllm | sglang | llamacpp | any
vendor: nvidia | metax | huawei | any
layer: backend_version | backend_runtime | nbr | deployment
advanced: boolean
risk_level: low | medium | high
source: official_default | image_help | lightai_recommended | vendor_profile | user_custom
allow_override: boolean
visible_when: expression
applies_to: list
help_ref: string
placeholder: string
validation: object
conflict_keys: list
redaction: none | secret | token | path_sensitive
copy_policy: copy_exact | derive_on_copy | never_copy
```

## 5. Help text model

Parameter help text should be externalized to versioned files, not hardcoded in Vue components.

Suggested location:

```text
configs/backend-catalog/help/
  vllm/
    vllm-v0.23.0.zh-CN.yaml
    vllm-v0.23.0.en-US.yaml
  sglang/
    sglang-v0.5.13.post1.zh-CN.yaml
    sglang-v0.5.13.post1.en-US.yaml
  llamacpp/
    llamacpp-b9700.zh-CN.yaml
    llamacpp-b9700.en-US.yaml
  vendors/
    nvidia.zh-CN.yaml
    metax.zh-CN.yaml
    huawei.zh-CN.yaml
```

Help item example:

```yaml
- name: gpu_memory_utilization
  title: GPU 显存利用率
  summary: 控制 vLLM 可使用的 GPU 显存比例。
  official_default: "0.9, verify with image help"
  lightai_recommendation: "单实例可考虑 0.85-0.92；多实例建议降低。"
  risk: "设置过高可能导致 OOM；设置过低可能降低吞吐。"
  applies_to:
    backend: vllm
    vendor: any
  source:
    - https://docs.vllm.ai/en/stable/configuration/engine_args/
```

UI behavior:

- each parameter label shows a `?` icon;
- click opens a popover/drawer;
- help shows meaning, official default, image default, LightAI recommendation, risk, backend/vendor applicability, layer, and source;
- help content can be updated without editing UI code.

## 6. Custom parameter model

Custom parameters are required for long-tail support:

- `extra_args` for backend CLI flags;
- `extra_env` for environment variables;
- `extra_docker_options` for advanced Docker runtime options.

But custom parameters must not bypass structured fields.

Conflict detection must catch:

- duplicate `--host`;
- duplicate `--port`;
- duplicate model path flags such as `--model`, `--model-path`, `-m`;
- duplicate `--served-model-name` when structured field is enabled;
- extra env duplicating structured env;
- extra Docker options duplicating structured Docker fields;
- vendor-incompatible devices;
- disabled fields entering final spec.

## 7. Clean DB policy

No legacy compatibility is required for old parameter schemas or old polluted values.

If old DB data pollutes the clean model:

1. document the issue;
2. rebuild the DB;
3. do not add fallback branches to keep old dirty data working.

## 8. Acceptance criteria

Implementation is acceptable only when:

1. required startup args are independent schema fields;
2. raw startup textarea no longer carries required core args;
3. each editable field has one authority;
4. each layer can be modified and GET-verified;
5. each next layer inherits/copies according to design;
6. Deployment override has highest priority;
7. disabled optional values are retained but filtered from final config;
8. vendor-specific defaults do not cross vendors;
9. final RunPlan and Docker inspect match;
10. vLLM/SGLang/llama.cpp E2E still pass;
11. evidence and closeout docs are updated.
