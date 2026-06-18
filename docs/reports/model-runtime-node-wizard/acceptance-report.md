# Model Runtime Node Wizard Acceptance Report

**Phase:** 4 — Model Runtime Wizards
**Branch:** `phase-4-model-runtime-wizards`
**Date:** 2026-06-18

**Final Conclusion: ACCEPTED_WITH_GAPS**

The single-node NVIDIA Docker wizard flow is verified end to end. A user can add a persisted node model root, browse and scan a model, create ModelArtifact and ModelLocation, select Backend/BackendVersion/BackendRuntime, preflight, start Docker, query `/v1/models`, inspect Docker logs through API, stop, and clean up. Remaining P2 product-depth items are recorded as formal `DOCUMENTED_BLOCKER` entries in `docs/reports/model-runtime-node-wizard/open-issues-closeout.md`.

---

## 1. Implemented Flow

```text
Node model root policy
  -> Remote file browse
  -> Model scan
  -> ModelArtifact + ModelLocation
  -> Backend / BackendVersion / BackendRuntime
  -> NodeBackendRuntime readiness
  -> Deployment preflight
  -> Server RunPlan command preview
  -> Agent Docker start
  -> /v1/models health probe
  -> Docker logs API
  -> Stop and cleanup
```

---

## 2. Scheme B: Node Model Root Policy

This round implements scheme B:

```text
Server persists NodeModelRoot in DB.
Agent keeps denied_roots and path containment as final protection.
Server passes only an authorized root to Agent browse/scan requests.
```

Default allowed roots:

```text
allowed_model_roots default to empty.
The user must explicitly add a model directory for a node before browsing, scanning, or saving a ModelLocation.
```

Denied roots default:

```text
/
/etc
/root
/boot
/proc
/sys
/dev
/run
/var/run
/var/lib/docker
```

Database:

```text
V15: node_model_roots
```

Root mutation rules:

```text
POST/PATCH/DELETE require node_model_root:write.
File browse and scan require node_file:read.
Root add/delete/disable write audit logs.
DELETE checks active ModelLocation references before removing a root.
Agent rejects denied_roots, path traversal, and symlink escapes even when Server authorizes a root.
```

---

## 3. Root Not Allowed

Root cause:

```text
RemoteFileBrowser previously allowed adding a front-end temporary root, but that root was not persisted into a shared Agent/Server allowed model root policy. Browse and scan/save could therefore use different authorization sources, producing root not allowed after the user selected a directory.
```

Fix:

```text
RemoteFileBrowser now loads /api/v1/nodes/{node_id}/model-roots.
Adding a model directory calls POST /api/v1/nodes/{node_id}/model-roots.
Browse uses /api/v1/nodes/{node_id}/files?root_id=...&path=...
Scan posts root_id/root + relative_path.
ModelLocation save stores model_root + relative_path after Server validation.
Server resolves root_id against node_model_roots and sends the authorized root to Agent.
Agent performs final denied_roots and containment checks.
```

Unified path payload:

```json
{
  "root_id": "node-root-id",
  "root": "/home/kzeng/models",
  "relative_path": "Qwen3-0.6B-Instruct-2512",
  "path_type": "directory"
}
```

---

## 4. Server API

| Endpoint | Method | File |
| --- | --- | --- |
| `/api/v1/nodes/{id}/model-roots` | GET/POST | `internal/server/api/model_browser_handlers.go` |
| `/api/v1/nodes/{id}/model-roots/{root_id}` | PATCH/DELETE | `internal/server/api/model_browser_handlers.go` |
| `/api/v1/nodes/{id}/files?root_id=...&path=...` | GET | `internal/server/api/agent_proxy_handlers.go` |
| `/api/v1/nodes/{id}/model-paths/scan` | POST | `internal/server/api/agent_proxy_handlers.go` |
| `/api/v1/model-artifacts/{id}/locations` | POST | `internal/server/api/artifact_handlers.go` |
| `/api/v1/nodes/{id}/docker-images` | GET | `internal/server/api/agent_handlers.go` |
| `/api/v1/deployments/preflight` | POST | `internal/server/api/preflight_handlers.go` |
| `/api/v1/deployments/{id}/start` | POST | `internal/server/api/deployment_lifecycle_handlers.go` |
| `/api/v1/node-run-plans/{id}/logs` | GET | `internal/server/api/node_run_plan_handlers.go` |

Legacy `/model-browser/roots` endpoints remain as compatibility wrappers, but the wizard flow uses `/model-roots`.

---

## 5. Agent Capabilities

| Capability | Endpoint | Details |
| --- | --- | --- |
| Docker images | `GET /docker-images` | repository, tag, image_id, digest, created_at, size, search, pagination |
| File browser | `GET /files` | Server-authorized root, denied_roots final protection, path traversal prevention |
| Model scanner | `POST /model-paths/scan` | Same root policy as browser; HuggingFace config.json, safetensors, GGUF detection |
| Docker start/logs/stop | task channel | Start container, return logs, stop and cleanup |

---

## 6. Web Pages And Components

| Page/Component | Route/File | Status |
| --- | --- | --- |
| ModelArtifactsPage | `web/src/pages/ModelArtifactsPage.vue` | Model wizard uses node root, browse, scan, ModelLocation save |
| RemoteFileBrowser | `web/src/components/RemoteFileBrowser.vue` | Loads persisted roots, adds/deletes roots through Server API |
| BackendRuntimesPage | `web/src/pages/BackendRuntimesPage.vue` | Runtime template and node runtime management |
| RunnerConfigsPage | `web/src/pages/RunnerConfigsPage.vue` | Runtime configuration wizard with Docker image picker |
| DockerImagePicker | `web/src/components/DockerImagePicker.vue` | Node Docker image browser |
| ModelDeploymentsPage | `web/src/pages/ModelDeploymentsPage.vue` | Start wizard explicitly selects Backend, BackendVersion, Runtime and uses Server preflight/start |
| ModelInstancesPage | `web/src/pages/ModelInstancesPage.vue` | Instance status and Docker logs |

Wizard flows:

```text
Model wizard:
  select node -> add/select model root -> browse -> scan -> save artifact/location

Runtime wizard:
  select template/runtime -> select node -> select Docker image -> check -> save node runtime

Deployment wizard:
  select model -> select Backend -> select BackendVersion -> select Runtime -> preflight -> start
```

---

## 7. Full Run Chain Review

Report:

```text
docs/reports/model-runtime-node-wizard/full-run-chain-review.md
```

Conclusion:

```text
ACCEPTED_WITH_GAPS
```

P0 status:

```text
root not allowed: FIXED
allowed root persistence and policy consistency: FIXED
browse/scan/save path semantics: FIXED
manual internal ID in main start flow: FIXED
page-to-Docker-run path for local NVIDIA: FIXED and E2E verified
```

P1 status:

```text
BackendVersion explicit in start wizard: FIXED
Server command preview from RunPlan resolver: READY
Docker image selection in Web: READY
Runtime/NodeRuntime management: READY for validated flow
delete/disable management: READY for roots and delete flows; richer disable UX is P2
```

---

## 8. NVIDIA E2E

Command:

```bash
scripts/e2e-model-runtime-wizard-nvidia-api.sh
```

Result:

```text
PASS
```

Evidence:

```text
docs/reports/model-runtime-node-wizard/e2e-run-20260618-115214/
```

Observed stages:

```text
login PASS
negative root tests PASS
add /home/kzeng/models root PASS
browse root PASS
scan Qwen3-0.6B-Instruct-2512 PASS
create ModelArtifact + ModelLocation PASS
query Docker images PASS
enable/check NodeBackendRuntime PASS
preflight PASS
Server command preview PASS
start Docker PASS
/v1/models PASS
Docker logs API PASS
stop PASS
cleanup PASS
```

Negative tests:

```text
/ rejected
/etc rejected
/etc/lightai rejected
/tmp/../etc rejected after filepath.Clean
```

Cleanup evidence:

```text
Residual e2e-wizard deployments: 0
Residual e2e-wizard model_artifacts: 0
Residual e2e-wizard node_model_roots: 0
```

Two old `lightai-*` containers from 2026-06-17 were observed after the run and left untouched because they were not created by this E2E run.

---

## 9. i18n Verification

Added/updated namespaces:

```text
fileBrowser
modelWizard
startWizard
```

Verification in this round:

```text
npm --prefix web run build: PASS
npm --prefix web test -- --runInBand: PASS
```

The i18n test verifies zh-CN and en-US key alignment, string leaf values, and referenced keys. No dotted key display leak was observed in the added wizard strings.

---

## 10. Final Verification Status

Full final gate to run before commit:

```bash
gofmt -w cmd/ internal/ || true
go test ./...
go vet ./...
go build ./...
npm --prefix web run build
npm --prefix web test -- --runInBand
bash -n scripts/*.sh
git diff --check
```

Current E2E status:

```text
scripts/e2e-model-runtime-wizard-nvidia-api.sh: PASS
```

---

## 11. Remaining Items

Formal closeout document:

```text
docs/reports/model-runtime-node-wizard/open-issues-closeout.md
```

| Item | Priority | Status | Notes |
| --- | --- | --- | --- |
| Deep model consistency comparison | P2 | DOCUMENTED_BLOCKER | Basic scanner exists; deep checksum/tokenizer manifest comparison is outside this closure scope. |
| GPU auto/manual product controls | P2 | DOCUMENTED_BLOCKER | Backend supports selected node/preflight; richer GPU controls are tracked in the formal closeout document. |
| Health-check detail panel | P2 | DOCUMENTED_BLOCKER | Runtime status exists; Web can add detailed panel. |
| Instance detail command preview link | P2 | DOCUMENTED_BLOCKER | Server preview exists; start flow displays generated preview. |
| Non-Docker runner types | P2 | DOCUMENTED_BLOCKER | Wizard remains Docker-first for this phase. |

No P0/P1 blocker remains for the validated local NVIDIA Docker wizard path.
