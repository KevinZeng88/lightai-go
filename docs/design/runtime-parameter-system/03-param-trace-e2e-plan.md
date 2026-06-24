# 03 - Parameter Trace E2E Plan

Status: discussion draft  
Target repo path: `docs/design/runtime-parameter-system/03-param-trace-e2e-plan.md`

## 1. Purpose

Normal E2E tests prove that a backend can start. They do not prove that the parameter system is correct.

This E2E plan proves:

1. each layer can modify the parameters it owns;
2. save and GET round-trip preserve `enabled/value`;
3. next layer inherits or clones values correctly;
4. later layers override earlier layers according to priority;
5. disabled values are preserved but excluded from final RunPlan/Docker spec;
6. vendor-specific defaults do not cross vendor boundaries;
7. final RunPlan and Docker inspect are exactly as expected.

## 2. Proposed script

Add:

```text
scripts/e2e-model-runtime-param-trace.sh
```

The script should reuse existing helpers instead of reinventing API login/node/model/runtime logic:

- `scripts/start-all.sh`
- `scripts/reset-password.sh`
- `scripts/e2e/lib/model-runtime-common.sh`
- existing model runtime wizard E2E helper functions

The script should support:

```bash
# default: dry-run/RunPlan trace only
bash scripts/e2e-model-runtime-param-trace.sh

# optional: create/start container and inspect
RUN_REAL_CONTAINER=1 bash scripts/e2e-model-runtime-param-trace.sh

# optional: backend selection
TRACE_BACKEND=vllm bash scripts/e2e-model-runtime-param-trace.sh
TRACE_BACKEND=sglang bash scripts/e2e-model-runtime-param-trace.sh
TRACE_BACKEND=llamacpp bash scripts/e2e-model-runtime-param-trace.sh
```

## 3. Evidence layout

Save evidence under:

```text
docs/reports/phase-3/evidence/runtime-param-layering/param-trace/
```

Required files:

```text
00-backend-version.json
01-model-created.json
02-model-modified.json
03-backend-runtime-created-or-cloned.json
04-backend-runtime-modified.json
05-nbr-created.json
06-nbr-modified.json
07-deployment-created.json
08-deployment-override-modified.json
09-preflight.json
10-runplan.json
11-equivalent-command.txt
12-docker-inspect.json
13-final-assertions.txt
14-param-source-table.md
15-docker-ps.txt
16-curl-v1-models.txt
```

If the script covers multiple backends, use subdirectories:

```text
param-trace/vllm/
param-trace/sglang/
param-trace/llamacpp/
```

## 4. Trace matrix

### 4.1 Full vLLM trace

vLLM should be the full trace target because it has model name, host/port, memory/performance args, and OpenAI-compatible serving behavior.

### 4.2 Lightweight SGLang trace

SGLang should verify:

- model path;
- host;
- container_port;
- startup timeout;
- core memory options if present;
- no NVIDIA/MetaX/Huawei cross-vendor pollution;
- final command and health behavior.

### 4.3 Lightweight llama.cpp trace

llama.cpp should verify:

- GGUF model path;
- host;
- container_port;
- ctx-size / n-gpu-layers if enabled;
- no platform-injected `LLAMA_ARG_HOST`;
- image-default `LLAMA_ARG_HOST` warning classified as benign;
- final command and health behavior.

## 5. Layer-by-layer flow

### Step 0 - Start services and reset credentials

```bash
bash scripts/start-all.sh --no-observability --wait
bash scripts/reset-password.sh --password test1234
```

Save:

```text
00-startup.log
```

### Step 1 - Read BackendVersion baseline

Read BackendVersion for the selected backend.

Save:

```text
00-backend-version.json
```

Assert:

1. BackendVersion contains schema/defaults, not node-specific values.
2. Required startup fields exist:
   - `model_container_path` or backend-specific equivalent;
   - `host`;
   - `container_port`.
3. Required fields are marked required.
4. Optional fields use enabled/value.
5. Backend-specific fields match selected backend.
6. Vendor-specific device defaults are not in BackendVersion common schema.
7. Health check defaults are present.
8. `startup_timeout_seconds` for SGLang is at least 120.

### Step 2 - Create model artifact and model location

Create model artifact and location using the existing local model path.

For vLLM/SGLang:

```text
/home/kzeng/models/Qwen3-0.6B-Instruct-2512
```

For llama.cpp:

```text
/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

Save:

```text
01-model-created.json
```

Assert:

1. Model format is correct.
2. ModelLocation node/path is correct.
3. Model layer does not contain Docker runtime args.
4. Model layer does not contain backend serve args.
5. Model layer does not contain vendor devices/env.

### Step 3 - Modify model-layer parameters

Modify model-layer fields that are allowed by API/UI, using sentinel values.

Example sentinel values:

```text
model display name = param-trace-model-modified
description = param trace model description
recommended_context = 2048, if supported
tags/capabilities = param-trace-tag, if supported
ModelLocation display metadata = param-trace-location, if supported
```

Save:

```text
02-model-modified.json
```

Assert:

1. Modified model fields GET round-trip correctly.
2. Modification does not add runtime args.
3. Modification does not add vendor devices/env.
4. Later RunPlan may use model name/path/metadata, but not treat model layer as runtime config source.

### Step 4 - Clone or create BackendRuntime

Clone the selected BackendRuntime from system catalog or create one using BackendVersion.

Save:

```text
03-backend-runtime-created-or-cloned.json
```

Assert:

1. Runtime schema is derived from BackendVersion.
2. Parameter values are built from schema.
3. Required values are enabled.
4. Optional disabled values are present but not active.
5. Vendor profile is selected or clearly absent.
6. No duplicate editing surfaces are involved.

### Step 5 - Modify BackendRuntime template parameters

Modify BackendRuntime-level template values. Use sentinel values different from later layers.

For vLLM full trace:

```text
host = 0.0.0.0
container_port = 18100
served_model_name.enabled = true
served_model_name.value = lightai-runtime-name
extra_args.enabled = true
extra_args.value = --max-model-len 2048
shm_size = 16gb
privileged = false
CUDA_VISIBLE_DEVICES.enabled = false
CUDA_VISIBLE_DEVICES.value = should-not-appear-runtime
devices.enabled = false
devices.value = /dev/fuse:/dev/fuse
```

Save:

```text
04-backend-runtime-modified.json
```

Assert:

1. GET returns exact enabled/value state.
2. Disabled values are preserved.
3. Required fields cannot be disabled.
4. BackendRuntime modifications do not mutate BackendVersion.
5. NBR created after this step should inherit these modified values unless NBR overrides them.

### Step 6 - Create NodeBackendRuntime

Create NBR from the modified BackendRuntime on the selected node.

Save:

```text
05-nbr-created.json
```

Assert:

1. NBR inherited BackendRuntime modified values.
2. NBR did not fall back to stale BackendVersion defaults.
3. NBR contains no unrelated vendor parameters.
4. NVIDIA NBR contains no `/dev/dri`, `/dev/mxcd`, `/dev/infiniband` by default.
5. Disabled values remain disabled but preserved.

### Step 7 - Modify NBR parameters

Modify NBR-level values using a second set of sentinel values.

For vLLM full trace:

```text
container_port = 18101
served_model_name.enabled = true
served_model_name.value = lightai-nbr-name
extra_args.enabled = true
extra_args.value = --max-model-len 1536
shm_size = 24gb
ipc_mode = host
CUDA_VISIBLE_DEVICES.enabled = false
CUDA_VISIBLE_DEVICES.value = should-not-appear-nbr
devices.enabled = false
devices.value = /dev/fuse:/dev/fuse
```

Save:

```text
06-nbr-modified.json
```

Assert:

1. GET returns exact enabled/value state.
2. NBR modifications do not mutate BackendRuntime.
3. NBR values are the default source for Deployment/RunPlan unless Deployment overrides them.
4. Disabled env/devices will not enter final RunPlan.
5. High-risk options are present in exactly one authoritative path.

### Step 8 - Create deployment

Create a ModelDeployment using the modified model and modified NBR.

Save:

```text
07-deployment-created.json
```

Assert:

1. Deployment sees the correct model and runtime candidate.
2. Deployment inherits or references NBR values according to design.
3. No vendor pollution appears in deployment parameters.
4. Required fields are present and cannot be disabled.

### Step 9 - Modify Deployment override

Modify Deployment override with highest-priority sentinel values.

For vLLM full trace:

```text
container_port = 18102, if Deployment is allowed to override container_port
served_model_name.enabled = true
served_model_name.value = lightai-deploy-name
extra_args.enabled = true
extra_args.value = --max-model-len 1024
CUDA_VISIBLE_DEVICES.enabled = false
CUDA_VISIBLE_DEVICES.value = should-not-appear-deploy
```

Save:

```text
08-deployment-override-modified.json
```

Assert:

1. Checkbox enable immediately makes input editable in UI/component test.
2. API save and GET round-trip exact enabled/value state.
3. Deployment override does not mutate NBR.
4. Disabled values are preserved but should not enter final config.
5. Deployment override has highest priority in final RunPlan.

### Step 10 - Preflight

Run preflight using the modified deployment/runtime/model.

Save:

```text
09-preflight.json
```

Assert:

1. Preflight passes for valid parameters.
2. Errors are structured when intentionally testing missing required fields.
3. No generic `unknown` error.
4. No backend/vendor mismatch.
5. No extra_args conflict.
6. No disabled env/device entering final config.

### Step 11 - RunPlan and equivalent command

Generate RunPlan / command preview / equivalent docker command.

Save:

```text
10-runplan.json
11-equivalent-command.txt
14-param-source-table.md
```

Assert final priority:

```text
Deployment override > NBR > BackendRuntime > BackendVersion default
```

For vLLM full trace, final RunPlan should include:

```text
--host 0.0.0.0
--port 18102, if Deployment can override container_port; otherwise 18101
--served-model-name lightai-deploy-name
--max-model-len 1024
```

Final RunPlan must not include:

```text
lightai-runtime-name
lightai-nbr-name, if Deployment override exists
--max-model-len 2048
--max-model-len 1536
should-not-appear-runtime
should-not-appear-nbr
should-not-appear-deploy
CUDA_VISIBLE_DEVICES, if enabled=false
/dev/dri
/dev/mxcd
/dev/infiniband
duplicate --host
duplicate --port
duplicate --served-model-name
duplicate --model / --model-path / -m
```

### Step 12 - Optional real container start and Docker inspect

If `RUN_REAL_CONTAINER=1`, start the deployment and inspect the container.

Save:

```text
12-docker-inspect.json
15-docker-ps.txt
16-curl-v1-models.txt
```

Inspect fields:

```text
Config.Cmd
Config.Env
HostConfig.PortBindings
HostConfig.DeviceRequests
HostConfig.Devices
HostConfig.Privileged
HostConfig.IpcMode
HostConfig.ShmSize
HostConfig.SecurityOpt
HostConfig.GroupAdd
HostConfig.Ulimits
```

Assert:

1. Docker inspect command matches RunPlan args.
2. Docker env excludes disabled env.
3. Docker devices exclude disabled devices and wrong-vendor devices.
4. Docker HostConfig high-risk options match RunPlan.
5. `/v1/models` returns 200 for real run.
6. Platform state becomes ready/running.
7. Logs are readable.

## 6. Extra negative tests

The script or unit tests should include negative cases:

1. Remove `host` from required args -> preflight reports missing `host`.
2. Remove `container_port` -> preflight reports missing `container_port`.
3. Put `--host` in `extra_args` while structured host exists -> conflict warning or error.
4. Put `/dev/dri` in NVIDIA defaults -> vendor mismatch error.
5. Disable CUDA_VISIBLE_DEVICES but verify value remains in stored parameter values and not in final Env.
6. Try to disable required host -> rejected or ignored with locked-on state.
7. Set Deployment override and confirm NBR is unchanged.

## 7. UI/component test requirements

API trace is not enough. UI tests must cover:

1. Required fields have no normal enable checkbox.
2. Optional fields have enabled/value behavior.
3. Enabling a Deployment override makes the input editable immediately.
4. Watchers do not reset input values after enable.
5. Saving and reloading preserves override values.
6. Re-disabling does not clear value.
7. High-risk fields appear in exactly one editing section.
8. MetaX fields are hidden for NVIDIA runtime.
9. Help `?` is available for parameter fields and resolves external help text.

## 8. Final test commands

After implementation:

```bash
gofmt -w cmd internal
git diff --check

bash -n scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
bash -n scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash -n scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
bash -n scripts/e2e-model-runtime-param-trace.sh

npm run build
npm test
go test ./internal/...
go build ./cmd/server/...
go build ./cmd/agent/...
```

Then run:

```bash
bash scripts/e2e-model-runtime-param-trace.sh
RUN_REAL_CONTAINER=1 TRACE_BACKEND=vllm bash scripts/e2e-model-runtime-param-trace.sh
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

## 9. Final report requirements

Final report must include:

1. param trace script path;
2. evidence directory path;
3. each layer snapshot path;
4. each layer modified parameters;
5. GET round-trip assertions;
6. inheritance and override table;
7. final RunPlan args/env/devices/ports/high-risk fields;
8. equivalent docker command;
9. Docker inspect fields if real run was used;
10. disabled parameter exclusion evidence;
11. required parameter lock evidence;
12. vendor pollution check;
13. vLLM modified real run result;
14. SGLang E2E result;
15. llama.cpp E2E result;
16. build/test results;
17. git diff stat;
18. git status short.
