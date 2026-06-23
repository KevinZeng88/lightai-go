# Golden Path Non-Regression Gate — Repair Plan V2

> Date: 2026-06-23
> Purpose: Define currently working flows that must remain working through all repairs
> **This is a HARD GATE. If a repair breaks a golden path flow, the repair is wrong.**

---

## 1. Principle

All repairs, refactors, cleanup, schema changes, and security fixes MUST preserve the currently working golden path.

**"No backward compatibility" ≠ "Break current flows"**

- Removing old fallbacks: OK — but current flows must work without them
- Changing schema: OK — but current data must be migrated or rebuilt
- Changing config: OK — but current catalog/seed must be updated
- Adding security checks: OK — but current authorized flows must not be blocked
- Adding tenant scope: OK — but current default-tenant flows must work

**Priority order:**
1. Current mainline flows continue working
2. Real bug fixes
3. Critical boundary hardening
4. Test coverage of current behavior
5. Future extensibility (don't write down)

---

## 2. Golden Path Definition

### 2A. Console Login & Navigation

| Step | Flow | Must Work |
|------|------|-----------|
| 1 | Login with admin credentials | YES |
| 2 | Dashboard loads with node/GPU status | YES |
| 3 | Nodes page lists registered nodes | YES |
| 4 | GPUs page shows GPU status | YES |
| 5 | Backend/Runtime/NBR pages accessible | YES |
| 6 | Model Artifact/Location pages accessible | YES |
| 7 | Deployment/Instance pages accessible | YES |
| 8 | Logs/diagnostics accessible | YES |
| 9 | Observability pages load (Grafana/Prometheus) | YES |

### 2B. Model File Flow

| Step | Flow | Must Work |
|------|------|-----------|
| 1 | Select agent/node for file browsing | YES |
| 2 | Browse model roots and files | YES |
| 3 | Scan model paths | YES |
| 4 | Identify HF/GGUF models | YES |
| 5 | Create/update ModelArtifact | YES |
| 6 | Create/update ModelLocation | YES |
| 7 | Multi-node path consistency check doesn't break single-node flow | YES |

### 2C. Runtime / NBR Flow

| Step | Flow | Must Work |
|------|------|-----------|
| 1 | Backend/BackendVersion catalog loads | YES |
| 2 | BackendRuntime can be cloned/edited | YES |
| 3 | NodeBackendRuntime can be created/patched/enabled | YES |
| 4 | NBR check/check-request works | YES |
| 5 | Docker image list via server proxy works | YES |
| 6 | Docker image inspect via server proxy works | YES |
| 7 | NBR devices/volumes/env/ports/extra_args/privileged/ipc/security-opt params flow into RunPlan | YES |

### 2D. Preflight / RunPlan Flow

| Step | Flow | Must Work |
|------|------|-----------|
| 1 | Model location + NBR form candidate intersection | YES |
| 2 | Preflight returns runnable candidates | YES |
| 3 | RunPlan preview shows image/args/env/ports/volumes/devices/health check | YES |
| 4 | Equivalent Docker command generates correctly | YES |
| 5 | High-risk params can be marked/audited but NOT blocked | YES |
| 6 | Boolean flag fix doesn't break existing value flags | YES |
| 7 | Required param error doesn't break catalog defaults | YES |
| 8 | Env substitution fix preserves original env | YES |

### 2E. Instance Start / Logs / Stop Flow

| Step | Flow | Must Work |
|------|------|-----------|
| 1 | Deployment start creates instance/task | YES |
| 2 | Agent claims task | YES |
| 3 | Docker container starts per RunPlan | YES |
| 4 | Health check succeeds | YES |
| 5 | `/v1/models` endpoint accessible | YES |
| 6 | Chat/completion smoke succeeds | YES |
| 7 | Instance logs viewable | YES |
| 8 | Stop removes container | YES |
| 9 | Restart doesn't hit container name conflict | YES |
| 10 | Failed start captures evidence before cleanup | YES |

### 2F. Three Backend Smoke

| Backend | Real GPU | Mock/Dry-Run |
|---------|----------|--------------|
| llama.cpp CUDA/GGUF | Must work if hardware available | Must have dry-run |
| vLLM OpenAI-compatible | Must work if hardware available | Must have dry-run |
| SGLang OpenAI-compatible | Must work if hardware available | Must have dry-run |

**If real GPU/model/image unavailable**: Must retain RunPlan dry-run, equivalent Docker command preview, Docker image/model path check, or mock/api-only smoke.

**Closeout must state**: Which flows were verified with real hardware, which with dry-run/mock.

---

## 3. Per-Batch Non-Regression Checks

### Batch 1A: Tenant Ownership

After adding tenant scope checks:

| Check | Command/Method |
|-------|---------------|
| Same-tenant user can access own nodes | API: `GET /api/nodes` returns tenant's nodes |
| Same-tenant user can access own NBRs | API: `GET /api/node-backend-runtimes?node_id=X` |
| Same-tenant user can browse files | API: `GET /api/nodes/{id}/files?path=...` |
| Same-tenant user can scan models | API: `POST /api/nodes/{id}/model-paths/scan` |
| Same-tenant user can list Docker images | API: `GET /api/nodes/{id}/docker-images` |
| Platform admin can access all resources | API with admin session |
| Default tenant/admin flow works | Login → nodes → files → scan |
| Web file browse not blocked by tenant check | UI: node detail → files tab |
| Web model scan not blocked by tenant check | UI: node detail → scan |

### Batch 1B: AgentClient / SSRF

After replacing bare `http.Get()`:

| Check | Command/Method |
|-------|---------------|
| Server can reach local agent | `GET /api/nodes/{id}/files` returns data |
| Server can reach LAN agent | If multi-node: `GET /api/nodes/{id}/files` on remote node |
| Dev/LAN mode allows localhost | Config: `address_policy.mode=dev` |
| Dev/LAN mode allows private IP | Config: `address_policy.mode=lan` |
| File browse proxy works | `GET /api/nodes/{id}/files?path=/models` |
| Model scan proxy works | `POST /api/nodes/{id}/model-paths/scan` |
| Docker images proxy works | `GET /api/nodes/{id}/docker-images` |
| Docker inspect proxy works | `GET /api/nodes/{id}/docker-image-inspect?ref=...` |
| URL encode fix doesn't change normal queries | Existing queries still work |
| Timeout doesn't fail on large directories | Scan large model directory completes |

### Batch 1C: Agent Endpoint Protection / NBR Execution Boundary

After adding agent endpoint auth:

| Check | Command/Method |
|-------|---------------|
| Server proxy still works | `GET /api/nodes/{id}/files` (server→agent) |
| Prometheus `/metrics` still works | `curl http://agent:19091/metrics` (or auth configured) |
| `/healthz` still works | `curl http://agent:19091/healthz` |
| NBR-defined params not blocked | RunPlan preview shows all NBR params |
| MetaX `/dev/mxcd` in NBR → appears in AgentRunSpec | If NBR defines it, it flows through |
| NVIDIA `--gpus` in NBR → appears in AgentRunSpec | If NBR defines it, it flows through |
| `--privileged` in NBR → appears in AgentRunSpec | If NBR defines it, it flows through |
| High-risk params logged in audit | Audit detail includes privileged/ipc/devices |

### Batch 2: Docker Lifecycle / Cleanup

After adding ContainerRemove and fixing races:

| Check | Command/Method |
|-------|---------------|
| Normal start succeeds | Deploy → instance running → `/v1/models` |
| Normal stop removes container | Stop → `docker ps -a` shows no container |
| Restart after stop works | Stop → Start → no name conflict |
| Failed start cleanup works | Simulate failure → container removed → logs captured |
| Logs viewable before cleanup | Failed instance → logs still accessible |
| `go test -race` passes | `go test -race ./internal/agent/... ./cmd/agent/...` |
| Concurrent log tasks don't panic | Multiple instances with concurrent log collection |

### Batch 3: Input / Output / Audit Safety

After adding body limit and fixing redaction:

| Check | Command/Method |
|-------|---------------|
| Normal API requests not rejected | Standard CRUD operations work |
| 10MB+ body returns 413 | `curl -X POST -d @100mb.json` → 413 |
| Audit log detail is valid JSON | `GET /api/audit-logs` → parse detail field |
| `PASSWORD_CHANGED` not corrupted | Audit log with action name preserved |
| Normal logs not over-truncated | Instance logs viewable with reasonable content |
| Large log truncation has marker | Truncated logs show `... [truncated]` |
| Docker stream parsing unaffected | Container logs still readable |

### Batch 4: RunPlan / Runtime Config

After fixing resolver bugs:

| Check | Command/Method |
|-------|---------------|
| vLLM RunPlan tests pass | `go test ./internal/server/runplan/...` |
| SGLang RunPlan tests pass | Same |
| llama.cpp RunPlan tests pass | Same |
| Boolean flags preserved | `--trust-remote-code` appears in args |
| Value flags preserved | `--model /path/to/model` appears in args |
| Required param error works | Missing `--model` → error returned |
| Env substitution works | `{{MODEL_CONTAINER_PATH}}` substituted |
| Original env preserved | Non-template vars unchanged |
| Equivalent Docker command matches AgentRunSpec | Preview matches actual spec |
| Input hash changes on GPU reassignment | Different GPUs → different hash |

### Batch 6: Web / i18n / Permission UX

After frontend fixes:

| Check | Command/Method |
|-------|---------------|
| Login works | Browser: login page → console |
| Route guard doesn't cause login loop | Navigate to `/dashboard` → loads |
| Dashboard accessible | Browser: `/dashboard` |
| Nodes page accessible | Browser: `/nodes` |
| GPUs page accessible | Browser: `/gpus` |
| Runtimes pages accessible | Browser: `/backend-runtimes`, `/node-backend-runtimes` |
| Deployments page accessible | Browser: `/deployments` |
| i18n no key leakage | Switch to en-US → no raw keys |
| No hardcoded Chinese in dashboard | en-US locale → all English |
| Grafana page loads | Browser: `/observability/grafana` |
| Grafana credentials removed or admin-only | No `admin/admin` shown to regular users |
| Destructive confirmation works | Stop deployment → confirmation dialog → proceeds |

### Batch 7: Test Infrastructure

Final test verification:

| Check | Command/Method |
|-------|---------------|
| Auth unit tests pass | `go test ./internal/server/auth/...` |
| Tenant isolation tests pass | `go test ./internal/server/api/... -run Tenant` |
| RunPlan tests pass | `go test ./internal/server/runplan/...` |
| Docker lifecycle tests pass | `go test ./internal/agent/runtime/...` |
| `go test -race` passes | `go test -race ./...` |
| Mock E2E passes | `scripts/e2e-mock-smoke.sh` |
| Frontend tests pass | `cd web && npm test` |
| Real GPU E2E runnable | `scripts/e2e-real-smoke-all-three.sh` (manual) |

---

## 4. Baseline / After Comparison Requirements

Each batch closeout MUST include:

### Before Baseline

- Current git commit SHA
- Which golden path flows currently work
- Key commands/scripts that verify them
- Evidence paths (logs, screenshots, curl outputs)
- What is NOT verified in current environment

### After Verification

- Same flows still work (with same or new commands)
- Which scripts/commands passed
- If old script deleted: what replaces it
- If old config deleted: how new config is generated
- If flow can't verify in current environment: why + dry-run/mock evidence

### Prohibited

- ❌ "go test pass" alone = completion
- ❌ Fix code without running golden path
- ❌ Delete old flow without providing new flow
- ❌ Security fix blocks NBR-defined parameters
- ❌ New abstraction breaks existing Web wizard / API flow

---

## 5. Evidence Retention

Failed container evidence must be captured BEFORE remove:

| Evidence | Where | Retention |
|----------|-------|-----------|
| Container logs (stdout/stderr) | Sent to server via task result | In DB |
| Container inspect state | Sent to server via task result | In DB |
| Docker error message | Sent to server via task result | In DB |
| Health check failure reason | Sent to server via task result | In DB |
| Container ID | In instance record | In DB |

**After evidence captured**: Container is removed. Evidence is NOT lost.

**Debug retain**: Future dev/admin explicit config option. NOT default behavior.

---

## 6. Golden Path Verification Scripts

Suggested verification scripts (to be created or updated):

| Script | Purpose |
|--------|---------|
| `scripts/verify-golden-path.sh` | Runs through core flows via API |
| `scripts/e2e-mock-smoke.sh` | Mock E2E (no real GPU) |
| `scripts/e2e-real-smoke-all-three.sh` | Real GPU E2E (manual) |

`verify-golden-path.sh` should:
1. Login
2. List nodes
3. List GPUs
4. List backends/runtimes/NBRs
5. List model artifacts/locations
6. Browse files on a node
7. Scan model paths
8. List Docker images
9. Create a deployment (dry-run or real)
10. Check RunPlan preview
11. Start/stop instance (if real GPU available) or verify RunPlan only
12. Check logs
13. Verify cleanup

Returns: PASS/FAIL per step + evidence paths.
