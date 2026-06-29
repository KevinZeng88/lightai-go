# Codex AUTOFIX Prompt — LightAI Go Runtime UX / RunPlan Cross-Backend Repair

You are working in the LightAI Go repository.

Repository path:

```bash
/home/kzeng/projects/ai-platform-study/lightai-go
```

Work on the current branch. Do not create a new branch. Implement clean fixes, run tests, commit, push, and provide a final report.

## 1. Context

The user manually verified the product and found a group of issues across these screens:

- 运行模板 / BackendRuntime / runtime template copy and edit.
- 节点运行配置 / NodeBackendRuntime create, check, detail, edit.
- 模型部署 / deployment wizard, parameter override, dry-run, final RunPlan, list, detail, edit.
- Docker command preview and actual start command.
- Device binding visibility for GPU selection.

The observed case used vLLM, but the repair must cover the shared architecture and verify vLLM, SGLang, and llama.cpp. Treat this as a product-level Runtime UX + RunPlan consistency repair.

Architecture expectations already established in this project:

```text
Backend / BackendVersion
  = backend capability and documented parameters.
  = hardware-neutral.

BackendRuntime
  = runtime template / launcher definition.
  = image, launcher kind, docker options, default service port, model mount, health check, command template.
  = node-independent.

NodeBackendRuntime
  = node-specific enabled runtime.
  = selected image evidence, check status, node-specific runtime override.
  = copy-on-create from BackendRuntime.

Deployment
  = model + selected NodeBackendRuntime + deployment overrides.
  = currently single-node first, with later room for nodes/replicas.

ResolvedRunPlan
  = authoritative final execution object.
  = combines BackendRuntime snapshot + NodeBackendRuntime snapshot + Deployment override + model location + device binding.
```

Important project rules:

- No historical compatibility cleanup burden is required. If old runtime_type/backend_runtime_id paths are dirty and obsolete, remove or centralize away from them.
- Backend/BackendVersion must stay hardware-neutral.
- GPU/vendor/device binding belongs to placement/resource/device binding, NodeBackendRuntime/Deployment/RunPlan, not backend-specific model parameters.
- NodeBackendRuntime is the deployable runtime object; deployment APIs should use `node_backend_runtime_id`.
- Do not implement vLLM-only fixes. Verify all three runtime families: vLLM, SGLang, llama.cpp.
- Prefer API-first tests and E2E. Add minimal frontend tests where needed for UX regression.

## 2. User-reported defects

### 2.1 Runtime template copy flow

Screen: “运行模板”.

Problems:

1. When clicking “复制为用户配置”, the dialog asks for “技术名称” and “显示名称”. Their meanings are unclear.
2. Duplicates appear possible. Product expectation: uniqueness should be enforced where ambiguity would result.
3. After clicking “复制为用户配置”, the system navigates into a parameter/detail page. The desired behavior is to save directly and return to the runtime template list.
4. The page after copy has only an “编辑” button located far below the visible header area. Users do not know what to do.
5. If the user needs to modify the copy, they can open it and click edit later.

Required behavior:

- In the copy dialog, ask for display name only unless there is a strong technical reason to expose internal name.
- Treat technical name as an internal unique key/slug used by API/catalog/runtime identity.
- Generate technical name automatically when copying catalog runtime into a user runtime.
- Show technical name only in advanced/detail/debug contexts, preferably read-only.
- Treat display name as the user-facing label in lists, selectors, headers, and breadcrumbs.
- Enforce display-name uniqueness in the relevant UI/API scope to avoid ambiguous selectors.
- After successful copy, save and return to the runtime template list.
- Show a clear success message; highlight or select the new user configuration if feasible.

Recommended uniqueness scope:

- `name` / technical name: unique globally or at least unique for runtime config records.
- `display_name`: unique within tenant + owner/user scope + backend + vendor + launcher kind. If the current product has a simpler tenancy model, enforce uniqueness within the runtime template list shown to the user.

### 2.2 Runtime template edit flow

Problems:

1. Save/edit buttons are too low on the page and hard to find.
2. The title is visible, e.g. `vLLM NVIDIA Docker - 用户配置`, but primary actions are far below.
3. After clicking “保存”, the expected behavior is to persist changes and exit to the previous list/detail route.

Required behavior:

- Add a sticky page/drawer header across runtime config edit pages.
- Header layout:

```text
[Title / object name / status]                         [保存] [取消/返回]
[Backend / vendor / launcher / source/catalog/user summary]
```

- Save persists and returns to list or detail.
- Advanced parameters can remain lower in the page, but primary actions must stay visible.

### 2.3 Runtime template port fields

Observed issue:

- `Container listen port` / 容器端口 is 8000 for vLLM.
- `宿主机端口` is empty.
- Later actual start command includes `-p 8000:8000/tcp`.

Evidence from uploaded mhtml snapshot:

- Deployment detail contains `BackendRuntimeConfigSet`.
- Context includes `backend_id=backend.vllm`, `backend_runtime=runtime.vllm.nvidia-docker`, `launcher_kind=docker`, `vendor=nvidia`.
- `launcher.ports` is `[]`.
- `service.container_port` is `8000`.
- `model_runtime.port` is `{{container_port}}`.

This indicates a shared schema/semantic mismatch.

Required behavior:

- Standardize port semantics:
  - `service.container_port`: in-container service listen port used by health checks, CLI port arg, Docker container target port.
  - `service.host_port` or `launcher.ports[].host_port`: host published port.
  - `launcher.ports[]`: effective Docker port bindings derived from the resolved port values, not an unrelated empty field.
- Pick a deterministic default for single-node development. Prefer default host port equals container port when safe and available.
- Validate occupied/conflicting host port where validation exists.
- The UI must not show container port 8000 while leaving a required host port blank without explanation.
- Docker preview and final RunPlan must include effective port binding.
- Do not hard-code vLLM's 8000 into shared resolver.

Cross-backend defaults to verify from current catalog/seeds/templates:

- vLLM: expected container port 8000 in the user's environment.
- SGLang: expected default from template, commonly 30000 in this project context.
- llama.cpp: default from its template. Recent project smoke tests used host 8002 to container 8000; verify current catalog/template instead of assuming.

### 2.4 Node runtime configuration creation/edit flow

Problems:

1. Current creation flow is too parameter-heavy.
2. After selecting node and runtime template, choosing image should be enough for normal cases.
3. Most users should not need to edit parameters during initial creation.
4. If parameters need changes, they can edit later.
5. There is no obvious “编辑” button. Row click opens detail, but modification should be possible there.

Required behavior:

- NodeBackendRuntime creation minimal path:
  1. Select node.
  2. Select runtime template/user configuration.
  3. Select or confirm image.
  4. Save.
- Use template defaults for parameters.
- Keep advanced parameters collapsed by default during creation.
- After save, return to the node runtime configuration list.
- Preserve explicit enable/check readiness behavior.
- Row click opens useful detail/edit.
- Detail page exposes edit mode clearly with sticky header actions.

### 2.5 Model deployment creation / parameter override / preview

Problems:

1. Buttons should be visible at the top/sticky header area.
2. Parameters should be lower/collapsed because users often deploy with defaults.
3. During “参数覆盖” -> next step, the page shows:

```text
可运行: No
加载失败: [resolve_error] unsupported runtime_type: (only docker is supported)
Docker 命令预览: empty
最终运行计划: empty
```

Required behavior:

- Deployment wizard keeps primary actions visible at top and/or bottom.
- Basic deployment path can proceed without touching advanced parameter overrides.
- Parameter override is optional and collapsed by default.
- RunPlan preview resolves successfully for valid Docker runtimes.
- Docker command preview renders before save when required deployment inputs are present.

Likely root-cause direction:

- Some preview/resolve path still reads legacy or empty `runtime_type`.
- The current config path has `launcher.kind`, `launcher_kind`, or `BackendRuntimeConfigSet` context, where `launcher_kind=docker` exists.
- Actual start may work because it uses a different resolver path or post-save path.

Required code audit:

- Search all `runtime_type`, `launcher_kind`, `launcher.kind`, `backend_runtime_id`, `node_backend_runtime_id`, `ResolveRunPlan`, `RunPlan`, `dry-run`, `preflight`, and deployment preview call sites.
- Make preview, dry-run, save, and actual start use the same authoritative resolver.
- The authoritative launcher type should come from the resolved config/runtime context.
- Valid Docker runtimes must not fail due to empty legacy `runtime_type`.

### 2.6 Model deployment list/detail/edit flow

Problems:

1. After saving deployment:
   - “名称” column is empty.
   - “模型” shows UUID like `72112423-ae38-4c29-a660-2c038225b6eb` instead of model display name.
2. No visible edit button.
3. Row click opens a mostly blank page with only “编辑”.
4. Clicking “编辑” does not work.
5. This page should support parameter modifications and later node/replica additions.

Required behavior:

- Deployment list shows human-readable fields:
  - Deployment display name or generated readable name.
  - Model display name, with ID only as debug/detail fallback.
  - Runtime display name, node, backend, status, created time.
  - Actions: start/stop/logs/dry-run/edit as applicable.
- If deployment name is optional at creation, generate a readable default.
- Row click opens meaningful detail page/drawer.
- Detail page shows summary, effective runtime, config overrides, final RunPlan preview, instances, logs/action links.
- Edit mode loads saved deployment config snapshot/overrides and can persist changes.
- Keep structure ready for later nodes/replicas, while supporting current single-node path.

### 2.7 Device binding visibility

Observed successful start command:

```bash
docker run -d --name lightai-2294c3a7-aea --ipc host --shm-size 8gb --gpus "device=0" -v /home/kzeng/models/Qwen3-0.6B-Instruct-2512:/models/Qwen3-0.6B-Instruct-2512:ro -e CUDA_VISIBLE_DEVICES=0 -p 8000:8000/tcp vllm/vllm-openai:latest --model /models/Qwen3-0.6B-Instruct-2512 --port 8000 --host 0.0.0.0
```

User question:

- Where are `--gpus "device=0"` and `CUDA_VISIBLE_DEVICES=0` configured? They were not visible in previous pages.

Required behavior and explanation:

- These are device/resource binding outputs, not ordinary vLLM/SGLang/llama.cpp parameters.
- Source should be selected node, selected/leased accelerator IDs, placement result, and ResolvedRunPlan.DeviceBinding.
- For NVIDIA Docker, render:
  - Docker GPU option: `--gpus "device=<ids>"`
  - environment variable: `CUDA_VISIBLE_DEVICES=<ids>`
- For MetaX/other vendors, rendering may differ, but source should remain neutral DeviceBinding.

UI requirements:

- Deployment wizard/detail must show a clear “设备绑定 / GPU 选择” section.
- Show node, accelerator vendor, selected GPU IDs, and how they render for Docker where useful.
- Do not expose `CUDA_VISIBLE_DEVICES` as an ordinary backend parameter.
- Avoid duplicating GPU selection in backend-specific parameter lists.
- Final RunPlan preview includes DeviceBinding.
- Docker command preview includes rendered device binding before start.

## 3. Required implementation tasks

### 3.1 Repository audit commands

Run at least:

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go

grep -R "runtime_type" -n . --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=dist --exclude-dir=build
grep -R "launcher_kind" -n . --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=dist --exclude-dir=build
grep -R "launcher.kind" -n . --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=dist --exclude-dir=build
grep -R "node_backend_runtime_id" -n internal cmd web/src --exclude-dir=node_modules
grep -R "backend_runtime_id" -n internal cmd web/src --exclude-dir=node_modules
grep -R "ResolveRunPlan\|RunPlan\|Dry Run\|dry-run\|preflight" -n internal cmd web/src --exclude-dir=node_modules
```

Audit preview, dry-run, preflight, save, and start source-of-truth consistency.

### 3.2 Runtime kind resolution

Implement one centralized function/code path that resolves launcher kind from current config:

- Prefer resolved config `launcher.kind` or context `launcher_kind`.
- Resolve current supported Docker runtimes to `docker`.
- Reject truly unsupported launchers with clear errors.
- Route deployment preview, dry-run, preflight, and actual start through the same code path.

### 3.3 Port resolution

Implement one centralized function/code path for service and host port resolution:

- Container service port from `service.container_port` or equivalent resolved template value.
- CLI port arg from backend-specific command mapping.
- Docker host binding from `service.host_port` or `launcher.ports`.
- If host port is empty and product policy applies, materialize `host_port=container_port` into effective plan.
- Substitute variables such as `{{container_port}}` before command rendering.
- Preview and actual start command should materially match.

### 3.4 Device binding rendering

Centralize device binding render logic:

- Neutral source: DeviceBinding / AcceleratorIds / placement or GPU lease result.
- NVIDIA Docker render: `--gpus "device=<ids>"` and `CUDA_VISIBLE_DEVICES=<ids>`.
- UI summary and RunPlan JSON expose the binding.
- Backend-specific parameter editors do not duplicate GPU selection.

### 3.5 Deployment DTO/list/detail/edit

Fix API and frontend so:

- Deployment list returns and displays deployment display name.
- Deployment list returns and displays model display name.
- Deployment detail loads saved config snapshot, parameter overrides, and effective RunPlan.
- Edit action is visible and functional.
- Save exits to detail/list according to existing navigation conventions.

### 3.6 Frontend UX consistency

Apply consistent sticky action headers to:

- Runtime template copy/edit/detail.
- Node runtime configuration create/detail/edit.
- Model deployment wizard/detail/edit.

Apply “default-simple, advanced-later” behavior:

- Node runtime create: node + runtime template + image are enough.
- Deployment create: model + node runtime + basic resource/device selection are enough.
- Advanced params/overrides collapsed by default.

## 4. Cross-backend verification matrix

Verify all rows for vLLM, SGLang, and llama.cpp:

| Runtime | Copy user runtime | Node runtime save/check | Deployment dry-run | Docker preview | Start path or start-equivalent | Port binding | Device binding | List/detail names |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| vLLM NVIDIA Docker | required | required | required | required | required | required | required | required |
| SGLang NVIDIA Docker | required | required | required | required | required | required | required | required |
| llama.cpp NVIDIA Docker | required | required | required | required | required | required | required | required |

For each runtime, assert:

- Runtime copy returns to list.
- Technical name is generated/read-only; display name is user-facing and unique.
- Node runtime creation can complete with node + template + image.
- Deployment can be created with default parameters.
- Parameter override step is optional.
- Dry-run does not report `unsupported runtime_type`.
- Docker preview is non-empty.
- Final RunPlan is non-empty.
- Docker preview and actual start command use the same resolver.
- Container and host ports resolve correctly for that backend.
- Device binding is visible in RunPlan and rendered into Docker command.
- Deployment list shows human-readable deployment name and model display name.
- Deployment detail/edit loads meaningful content and edit works.

## 5. Required tests

Prefer API-first tests. Add frontend tests where the regression is UI-specific.

Backend/API:

```bash
go test ./internal/server/...
go test ./internal/agent/...
go build ./cmd/server/...
go build ./cmd/agent/...
```

Frontend:

```bash
cd web && npm run build
cd web && npm test
```

Add/update tests for:

- RunPlan resolver uses current launcher kind, not empty legacy runtime type.
- Deployment preview and actual start share resolver path.
- vLLM/SGLang/llama.cpp resolve Docker RunPlan.
- Port values are materialized and variable templates substituted.
- NVIDIA device binding renders `--gpus` and `CUDA_VISIBLE_DEVICES`.
- Deployment list API returns deployment display name and model display name.
- Runtime copy rejects duplicate display name in selected uniqueness scope.
- Runtime copy returns to list.
- Node runtime create can save with node + template + image only.
- Deployment wizard can proceed without parameter overrides.
- Row click opens detail; edit is visible and functional.

E2E/smoke:

- Use existing API-first harness if present.
- Execute dry-run/command-render for all three backend families.
- Execute real start/logs/stop for available local images/models.
- Where real execution is unavailable, document the missing external prerequisite and keep dry-run coverage.

## 6. Acceptance criteria

The repair is complete only when:

1. Valid Docker deployment preview resolves for vLLM, SGLang, and llama.cpp.
2. Docker command preview and final RunPlan are non-empty for all three backend families.
3. Preview and actual start use the same resolver path.
4. Runtime template copy saves and returns to list.
5. Technical name is generated/read-only; display name is user-facing and uniqueness is enforced.
6. Runtime config edit has visible sticky header actions; save exits.
7. Node runtime create works with node + runtime template + image only.
8. Deployment create works without touching parameter overrides.
9. Deployment list has no blank primary name and no UUID-only model primary label.
10. Deployment detail is meaningful; edit works.
11. Device binding is visible as a deployment/resource/RunPlan concept.
12. NVIDIA Docker rendering includes `--gpus "device=<ids>"` and `CUDA_VISIBLE_DEVICES=<ids>` when GPU is selected.
13. Shared fixes preserve SGLang and llama.cpp behavior.
14. Tests and evidence are recorded.
15. Changes are committed and pushed.
16. Final `git status --short` is clean except explicitly documented pre-existing local files.

## 7. Final report required from Codex

Return a concise final report with:

1. Root causes found, grouped by Runtime Template / Node Runtime / Deployment / RunPlan / Device Binding.
2. Files changed.
3. Behavior after fix for vLLM, SGLang, llama.cpp.
4. Test commands and outputs.
5. E2E/dry-run evidence paths.
6. Commit ID and push result.
7. Final `git status --short`.
8. Any remaining issue that is blocked only by unavailable external images/models/hardware, with exact command/evidence showing the blocker.
