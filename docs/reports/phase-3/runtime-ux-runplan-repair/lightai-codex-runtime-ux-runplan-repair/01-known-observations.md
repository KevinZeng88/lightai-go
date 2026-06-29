# Known Observations From User Report and Uploaded MHTML

## User-visible defects

- Runtime template copy asks for technical name and display name; user does not know what technical name means.
- Copy flow enters a mostly unhelpful detail/parameter page; desired flow is save and return to list.
- Runtime template edit save button is buried low in the page; desired behavior is sticky header actions and save exits.
- vLLM template shows container listen port 8000 while host port is blank.
- Node runtime configuration creation is too parameter-heavy; desired basic flow is node + template + image, with advanced params later.
- Node runtime configuration lacks obvious edit affordance.
- Deployment wizard has primary actions too low; parameters should be lower/collapsed.
- Deployment parameter override next step reports `[resolve_error] unsupported runtime_type: (only docker is supported)`.
- Docker command preview and final RunPlan preview are empty in wizard.
- Saved deployment list has blank name, model column shows UUID.
- Deployment detail opens mostly blank and edit does not work.
- Actual start succeeds and renders Docker command with GPU/device binding that was not visible in earlier UI.

## MHTML evidence extracted

Uploaded file: `LightAI Go.mhtml`, saved from `/models/deployments`.

Extracted textual indicators:

- `BackendRuntimeConfigSet` appears in deployment/config detail.
- Context includes:
  - `backend_id=backend.vllm`
  - `backend_runtime=runtime.vllm.nvidia-docker`
  - `launcher_kind=docker`
  - `vendor=nvidia`
- Port-related fields include:
  - `launcher.ports=[]`
  - `service.container_port=8000`
  - `model_runtime.port={{container_port}}`

Interpretation:

- Runtime launcher kind exists in new config context.
- Preview failure likely reads an obsolete/empty `runtime_type` field instead of the current launcher kind.
- Port state is split between service-level and launcher-level fields.
- Preview and actual start likely use different resolver/render paths.

## Actual successful start command observed by user

```bash
docker run -d --name lightai-2294c3a7-aea --ipc host --shm-size 8gb --gpus "device=0" -v /home/kzeng/models/Qwen3-0.6B-Instruct-2512:/models/Qwen3-0.6B-Instruct-2512:ro -e CUDA_VISIBLE_DEVICES=0 -p 8000:8000/tcp vllm/vllm-openai:latest --model /models/Qwen3-0.6B-Instruct-2512 --port 8000 --host 0.0.0.0
```

Interpretation:

- Device binding is already present in the start path.
- UI and preview path do not clearly expose where it comes from.
- The source should be neutral DeviceBinding / AcceleratorIds / placement / GPU lease, then rendered to NVIDIA Docker-specific syntax.
