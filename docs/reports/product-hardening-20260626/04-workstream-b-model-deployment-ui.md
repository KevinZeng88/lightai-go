# 04 — Workstream B: Model Deployment UI Experience

## Goal

Replace thin deployment creation with a safe, previewable deployment workflow.

## Step B1 — Inspect current flow

Run:

```bash
rg -n "createVisible|createDeployment|dryRunDeployment|preflightDeployment|startDeployment|stopDeployment|node_backend_runtime_id|source_node_backend_runtime_id|service_json|config_overrides|host_port|served_model_name" web/src internal/server docs/api
```

Inspect:

```text
web/src/pages/ModelDeploymentsPage.vue
web/src/api/deployments.ts
internal/server/api/router.go
internal/server/api/*deployment*
internal/server/runplan/*
docs/api/openapi.yaml
```

## Step B2 — Define create/update payload contract

Target create payload must be explicit:

```json
{
  "name": "qwen3-vllm-demo",
  "display_name": "Qwen3 vLLM Demo",
  "model_artifact_id": "...",
  "node_backend_runtime_id": "...",
  "service_json": {
    "host_port": 8004,
    "container_port": 8000,
    "served_model_name": "qwen3-demo"
  },
  "placement_json": {
    "mode": "single",
    "accelerator_ids": ["0"],
    "gpu_policy": "exclusive"
  },
  "config_overrides": {
    "parameter_values": [
      {
        "key": "backend.common.served_model_name",
        "value": "qwen3-demo",
        "enabled": true
      }
    ]
  }
}
```

Rules:

- request must use `node_backend_runtime_id`;
- request must reject `backend_runtime_id` as deployment selector;
- client must not provide image/docker readiness evidence;
- unknown legacy fields must return structured error;
- source NBR snapshot must be copied at deployment creation;
- deployment override must not mutate NBR.

## Step B3 — Add unsaved preview path if missing

Current dry-run may require existing deployment. Create or confirm an endpoint for unsaved deployment preview.

Preferred:

```text
POST /api/v1/deployments/preview
```

Request: same as create payload.

Response:

```json
{
  "can_run": true,
  "run_plan": {},
  "docker_preview": "docker run ...",
  "lint": {
    "status": "ok",
    "findings": []
  },
  "resource_admission": {
    "status": "ok",
    "findings": []
  },
  "preflight": {
    "status": "ok",
    "errors": [],
    "warnings": []
  },
  "source_trace": {}
}
```

If reusing `/deployments/preflight`, ensure it returns final RunPlan and does not diverge from start resolver.

Required backend tests:

- preview and start use same resolver path;
- preview rejects legacy `backend_runtime_id`;
- preview rejects non-ready NBR;
- preview accepts `ready_with_warnings` with warnings;
- missing model location blocks;
- host port conflict blocks;
- disabled deployment parameter is not applied;
- deployment override wins over NBR.

## Step B4 — Build frontend wizard

Replace or refactor deployment dialog into sections.

Suggested files:

```text
web/src/components/deployments/DeploymentWizard.vue
web/src/components/deployments/ModelSelector.vue
web/src/components/deployments/NodeRuntimeSelector.vue
web/src/components/deployments/DeploymentServiceEditor.vue
web/src/components/deployments/DeploymentOverrideEditor.vue
web/src/components/deployments/DeploymentPreviewPanel.vue
```

Keep simple if fewer files are better, but the UI must show all sections.

Required UI sections:

### 1. Model

Show:

- display name;
- format;
- task type;
- capabilities;
- location/node;
- verification status;
- path type;
- warning if no location on selected node.

### 2. Node runtime config

Show:

- display name;
- node label;
- backend;
- backend version;
- runtime template;
- vendor;
- image;
- status;
- status reason;
- last checked time;
- probe summary.

Selectable:

- `ready`;
- `ready_with_warnings`.

Blocked:

- `needs_check`;
- `missing_image`;
- `error`;
- `unknown`;
- stale check status.

### 3. Service

Fields:

- host port;
- container port;
- served model name;
- endpoint preview;
- health check profile.

### 4. Resource and placement

Fields:

- accelerator IDs;
- exclusive/shared;
- memory budget if backend supports it;
- backend-specific controls.

### 5. Overrides

Use RuntimeParameterEditor.

Show:

- inherited values from NBR;
- deployment override values;
- disabled values retained;
- source/diff.

### 6. Preview

Show:

- can_run;
- errors/warnings;
- lint;
- resource admission;
- Docker command;
- RunPlan JSON;
- source trace;
- save/start blockers.

Use existing `JsonViewer`.

## Step B5 — Improve deployment list/detail

List columns:

- deployment display name;
- model display name;
- node runtime config display name;
- backend;
- node;
- status;
- last instance state;
- endpoint;
- actions.

Detail drawer:

- basic info;
- selected model;
- source NBR snapshot;
- service config;
- overrides;
- latest RunPlan;
- latest instance;
- logs link;
- dry-run/preview.

## Step B6 — Tests

Frontend tests:

```text
web/tests/deploymentWizard.test.mjs
web/tests/runtimeBoundaryUi.test.mjs
```

Required assertions:

- create payload uses `node_backend_runtime_id`;
- non-deployable NBR cannot be selected for start;
- `ready_with_warnings` shows warning and is selectable;
- preview button sends full payload;
- preview panel shows Docker command and findings;
- served model name is stored as parameter override or service field according to final contract;
- raw UUID-only labels are avoided when display names are available.

Backend tests:

```bash
go test ./internal/server/api/... -run 'Deployment|Preflight|Preview|RunPlan'
go test ./internal/server/runplan/... 
```

## Acceptance

- Deployment can be created from UI with model/NBR/service/override/preview.
- Start is blocked by preview/preflight errors.
- RunPlan preview is visible before start.
- Equivalent Docker command is visible before start.
- Deployment details are understandable without reading raw JSON.
- Existing API-first E2E still passes.
- Browser smoke covers model -> NBR -> deployment preview -> save/start -> logs -> stop.
