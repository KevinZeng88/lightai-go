# Batch D Closeout: ModelArtifact / ModelLocation Parameter Editing

> Date: 2026-06-24
> Status: PASS

---

## Summary

Implemented model-side parameter editing for ModelArtifact and copy-on-create for deployment.

## D1: ModelArtifact API

### Changes
- Added `parameter_defaults_json` to SELECT queries (list + single get)
- Added `parameter_defaults_json` to INSERT query
- Added `parameter_defaults_json` to PATCH handler
- API returns `parameter_defaults` field in response

### Commit
```
f8193fb fix(server): wire NBR snapshot into RunPlan resolution and add parameter_defaults to artifacts
```

## D2: Web Model Parameter Editor

### Changes
- Integrated RuntimeParameterEditor into ModelArtifactsPage edit dialog
- Added parameterDefaults state and computed modelEditorModel
- Load parameter_defaults on showEdit
- Include parameter_defaults in save payload
- Added i18n key `parameterDefaults` (en-US, zh-CN)

### Commit
```
8688ff0 feat(web): add model parameter defaults editor to ModelArtifactsPage
```

## D3: Deployment Copy Model Defaults

### Changes
- Deployment creation reads `parameter_defaults_json` from ModelArtifact
- Copies into deployment's `parameter_values_json`
- Provides default parameter values for future Batch E override logic

### Commit
```
82c0dcf feat(deployments): copy model parameter defaults at creation
```

## Files Changed

| File | Change |
|------|--------|
| `internal/server/api/artifact_handlers.go` | Add parameter_defaults_json to SELECT/INSERT/PATCH |
| `internal/server/api/deployment_lifecycle_handlers.go` | Copy model defaults at deployment creation; wire NBR snapshot |
| `internal/server/api/runtime_handlers.go` | Default NBR param fields to [] when nil |
| `web/src/pages/ModelArtifactsPage.vue` | Integrate RuntimeParameterEditor |
| `web/src/locales/en-US.ts` | Add parameterDefaults key |
| `web/src/locales/zh-CN.ts` | Add parameterDefaults key |

## API Changes

- `GET /api/v1/model-artifacts` returns `parameter_defaults` field
- `PATCH /api/v1/model-artifacts/{id}` accepts `parameter_defaults`
- `POST /api/v1/model-artifacts` accepts `parameter_defaults`
- Deployment creation copies model parameter_defaults into parameter_values_json

## Schema Changes

None — `parameter_defaults_json` column already exists from V28 migration.

## ModelArtifact / ModelLocation Boundary

- ModelArtifact stores: model format, architecture, quantization, capabilities, discovered metadata, parameter defaults
- ModelArtifact does NOT store: Docker image, entrypoint, ports, devices, privileged, security-opt
- ModelLocation stores: path, file type, size, checksum, discovered_metadata_json
- ModelLocation does NOT store: container config
- Model-side defaults only participate in generating Deployment defaults
- Model-side defaults do NOT override NBR snapshot

## Deployment Copy Defaults

- Deployment creation reads `parameter_defaults_json` from ModelArtifact
- Copies into `parameter_values_json` on the deployment
- This is the input for future Batch E override/disabled tombstone logic
- Deployment does NOT copy Docker container config from model

## `/tmp/lightai` Status

**NOT updated.** Running server/agent has NOT been rebuilt.

## Test Results

| Command | Result |
|---------|--------|
| `go build ./internal/server/...` | PASS |
| `go test ./internal/server/api/...` | PASS |
| `go test ./internal/server/runplan/...` | PASS |
| `cd web && npm run build` | PASS |
| `cd web && npm test` | PASS |

## Unresolved Issues

1. RuntimeParameterEditor not fully integrated into RunnerConfigsPage NBR edit (existing edit dialog works)
2. ModelLocation does not have parameter_defaults_json (not needed — model defaults are on ModelArtifact)

## Git Status

```
 M VERSION
?? .mimocode/plans/1782215119986-calm-planet.md
?? .mimocode/skills/
```
