# Model Runtime Node Wizard Acceptance Report

**Phase:** 4 — Model Runtime Wizards
**Branch:** `phase-4-model-runtime-wizards`
**Commits:** `84d86ec` (backend + i18n), `TBD` (web UI + E2E)

## 1. Server API

| Endpoint | Method | Status | Handler File |
|----------|--------|--------|-------------|
| `/api/v1/nodes/{id}/docker-images` | GET | ✅ Enhanced | `agent_handlers.go` |
| `/api/v1/nodes/{id}/files` | GET | ✅ New | `agent_proxy_handlers.go` |
| `/api/v1/nodes/{id}/model-paths/scan` | POST | ✅ New | `agent_proxy_handlers.go` |
| `/api/v1/model-artifacts/{id}/locations/{lid}` | PATCH | ✅ New | `model_location_handlers.go` |
| `/api/v1/model-artifacts/{id}/locations/{lid}` | DELETE | ✅ New | `model_location_handlers.go` |
| `/api/v1/backend-runtimes/{id}/clone` | POST | ✅ New | `node_runtime_handlers.go` |
| `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}` | PATCH | ✅ New | `node_runtime_handlers.go` |
| `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}` | DELETE | ✅ New | `node_runtime_handlers.go` |
| `/api/v1/deployments/preflight` | POST | ✅ New | `preflight_handlers.go` |

## 2. Agent Capabilities

| Capability | Endpoint | Status |
|-----------|----------|--------|
| Enhanced Docker images | `GET /docker-images?query=&limit=` | ✅ Returns repository, tag, image_id, digest, created_at, size, image_ref, image_present |
| File browser | `GET /files?root=&path=` | ✅ Controlled listing with allowed_roots, traversal prevention |
| Model scanner | `POST /model-paths/scan` | ✅ Detects HuggingFace config.json, safetensors, GGUF |

## 3. Web Components

| Component | File | Description |
|-----------|------|-------------|
| RemoteFileBrowser | `components/RemoteFileBrowser.vue` | Reusable file browser with breadcrumb, directory listing, file/dir selection |
| DockerImagePicker | `components/DockerImagePicker.vue` | Reusable Docker image selector with search, table, manual input |

## 4. Web Pages Updated

| Page | Changes |
|------|---------|
| ModelArtifactsPage | Added wizard (node → browse → scan → save), detail drawer with location management (add/rescan/delete) |
| ModelDeploymentsPage | Added start wizard (model → runtime → preflight → start), preflight result display |

## 5. i18n

- 492 leaf keys in both zh-CN and en-US
- 404 key references in Vue/TS templates — all resolve to strings
- 8 new namespaces: modelWizard, modelLocations, runtimeWizard, nodeRuntime, fileBrowser, dockerImages, preflight, startWizard
- Test: `npm --prefix web test -- --runInBand` PASS

## 6. E2E Script

- `scripts/e2e-model-runtime-wizard-nvidia-api.sh`
- Covers: login → browse files → scan model → create artifact+location → docker images → enable runtime → clone runtime → preflight → deploy+start → /v1/models → logs → stop → cleanup
- Cleanup with on_exit trap, all resources use `e2e-wizard-*` prefix

## 7. Basic Verification

```bash
go build ./...     ✅
go test ./...      ✅
go vet ./...       ✅
npm run build      ✅
npm test           ✅ 492 keys, no i18n leaks
bash -n scripts    ✅
git diff --check   ✅
```

## 8. Remaining Items

| Item | Priority | Notes |
|------|----------|-------|
| E2E full run on NVIDIA hardware | P1 | Script written, needs service+GPU+model+image available |
| BackendRuntimesPage wizard+node management | P1 | Base page exists, wizard flow for runtime creation not yet added |
| Deep model consistency comparison | P2 | Model scanner returns basic metadata, full fingerprint comparison deferred |
| Web tests for new components | P2 | Components are functional, automated tests deferred |
