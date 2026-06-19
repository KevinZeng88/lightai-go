# Backend Runtime / RunPlan / Docker Semantics

This document records the post-closeout UI persistence and RunPlan consistency rules.

## Object Responsibilities

- ModelArtifact stores model identity and user-facing metadata. `display_name` is for UI selection. `name` is the platform artifact name. `path` is the original source path.
- ModelLocation stores the node-specific model path: `model_root`, `relative_path`, and `absolute_path`.
- Backend and BackendVersion are system/catalog capability layers and remain read-mostly.
- BackendRuntime is a user-manageable runtime template. System templates must be cloned before editing.
- NodeBackendRuntime is a node-level runtime config with its own `display_name`, image evidence, readiness, and frozen runtime snapshot.
- Deployment stores the service-level config: model artifact, runtime config, placement, service ports, deployment parameters, and env overrides.
- RunPlan/NodeRunPlan is immutable once generated. Later Deployment or Runtime edits affect only future RunPlans.
- ModelInstance stores one concrete run: task, container, health, logs, status, and last error.

## Field Semantics

- `display_name`: user-visible label for model/runtime/config selection.
- `artifact_name` / `name`: stable internal artifact or config name.
- `source_path`: node/source path used to discover the model.
- `mount_path`: path mounted inside the container.
- `served_model_name`: optional OpenAI-compatible model id.
- `backend_model_arg`: backend-specific model path/name argument, usually the container mount path.

## Port Semantics

- `host_port`: host access port, for example `8005`.
- `container_port`: Docker exposed container port, for example `8080`.
- `app_port`: backend process listening port. Current templates use `container_port`; advanced edits should keep `app_port` and `container_port` aligned.
- `health_port`: host-side health probe port. Defaults to `host_port`.
- `api_test_port`: host-side model test port. Defaults to `host_port`.

RunPlan JSON, Docker command preview, and Agent Docker create spec must all use the same effective `host_port` and `container_port`.

## Parameter Priority

Effective runtime parameters are resolved in this order:

1. Deployment explicit parameters and env overrides.
2. NodeBackendRuntime frozen snapshot and node image override.
3. BackendRuntime template settings.
4. BackendVersion defaults.
5. Backend defaults.
6. System fallback.

## Run Idempotency

The server enforces single-active-instance per deployment. `pending`, `starting`, `provisioning`, `running`, `healthy`, and `stopping` block duplicate start requests with HTTP 409. `failed`, `stopped`, and `saved` can start a new RunPlan and ModelInstance while preserving history.
