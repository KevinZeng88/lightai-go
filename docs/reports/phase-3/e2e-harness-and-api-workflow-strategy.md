# LightAI Go API-First E2E Harness and Workflow Strategy

> Status: STRATEGY
> Date: 2026-06-20
> Scope: Long-term E2E test architecture, API workflow harness, shell E2E standardization, and existing shell script review
> Non-goals: Implement tests, change business code, add API, add DB migration, add shell helpers, add browser automation, start real model containers

## 1. Background

LightAI Go has already completed the NBR Image Probe Phase 0-4 work:

1. `ImageInspect` is the authoritative source for image existence.
2. `/docker-images` list is evidence for UI selection, not the authority for `missing_image`.
3. `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe` and `GET /probe` are formal APIs.
4. Web has i18n, status mapping, and probe display improvements.
5. L1 backend tests, L2 frontend static tests, and L3 real Agent Docker endpoint smoke have passed.
6. The old L4 browser UI smoke is no longer the main correctness gate because login/session/CSRF/browser setup makes it expensive and brittle.

The current gap is not one missing NBR test. The gap is a durable E2E system that can keep LightAI's model/runtime/deployment workflow stable as the product grows.

Current repository facts found during this review:

- `internal/server/api/router.go` already exposes `SetupRoutes(mux, RouterConfig)`, so real HTTP router tests can be built without adding a production router entrypoint.
- Most `internal/server/api/*_test.go` tests call handlers directly with `httptest.NewRecorder()` and `req.SetPathValue(...)`.
- Existing tests cover many isolated handler cases, including NBR probe error mapping, but do not yet cover full API workflow chains through the real mux.
- Existing shell E2E scripts already prove real local Docker/GPU/model flows, but helper code, login, readiness, naming, cleanup, evidence, and skip/fail rules are inconsistent.
- Web tests are strong for static checks and i18n, but browser smoke should not be treated as the main business correctness layer.

## 2. API-First E2E Principle

The durable testing model should be:

```text
Unit / handler correctness
-> real HTTP router contract
-> API workflow E2E
-> fake Agent integration
-> local shell API-only E2E
-> real Agent / Docker smoke
-> real model container smoke
-> UI static/component checks
-> manual UI checks only when needed
```

Core rules:

1. Web/UI is a display and interaction layer.
2. Every important frontend operation must have an API path.
3. Core business correctness is verified by API Workflow E2E.
4. API Workflow E2E should follow the same API order used by the Web, not simulate browser clicks.
5. UI tests cover build, i18n, status mapping, field binding, and component presence.
6. Browser smoke is not the primary business correctness gate.
7. Shell E2E can depend on the real local machine, but only as controlled real-environment verification.
8. Run Go API Workflow E2E before local shell real-environment E2E.
9. Avoid one huge full test. Use layers, tiers, and workflow-specific entrypoints.

## 3. Why Move from UI Smoke to API Workflow E2E

Browser smoke remains useful for final human confidence, but it is a poor primary correctness layer for LightAI's current risk profile:

| Issue | UI smoke problem | API Workflow E2E advantage |
| --- | --- | --- |
| Auth/session/CSRF | Browser setup is noisy and slow | Test helper can use real login or direct test session deterministically |
| Route `PathValue` | Browser failures are late and indirect | Real mux requests catch route/path mismatches directly |
| JSON field loss | UI may mask missing fields until a drawer opens | API tests can assert list/detail/create/patch payloads exactly |
| Agent/Docker errors | UI labels are secondary | API tests can assert exact error status mapping |
| Reproducibility | Depends on browser and environment | Fake Agent + test DB makes workflows deterministic |
| Debuggability | Screenshot/log correlation is slow | Request/response fixtures are direct evidence |

UI smoke should be used manually for visual questions: drawer panels, color tags, layout, and human-facing wording. It should not be the acceptance source for `missing_image`, run plan correctness, field persistence, lifecycle state, or cleanup semantics.

## 4. Test Layering

| Layer | Goal | Daily `go test` / `npm test` | Real server | Real Agent | Docker | GPU | CI fit | Finds | Does not find | Boundary |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1. Unit / Handler tests | Validate small functions and direct handler behavior | Yes | No | No, except fake HTTP servers | No | No | Strong | Field validation, persistence, status mapping, direct handler regressions | Real route/middleware/session behavior | Should stay narrow and fast; direct `PathValue` is acceptable here |
| 2. Real HTTP Router Contract tests | Validate real `http.ServeMux`, route patterns, middleware, `PathValue`, auth/CSRF | Yes, targeted under `internal/server/api` | In-process mux, not external process | Fake only when needed | No | No | Strong | Route mismatch, middleware ordering, CSRF/session regressions, API contract shape | Real Agent/Docker runtime behavior | Uses `SetupRoutes` and `httptest.Server` or `mux.ServeHTTP` |
| 3. API Workflow E2E tests | Verify Web-equivalent API sequences across resources | Yes for fake-agent/test-DB workflows | In-process mux | Fake Agent | No | No | Strong | list -> select -> create -> patch -> probe/preflight -> detail/list -> cleanup regressions | Real Docker/GPU/model compatibility | Primary correctness layer for business workflows |
| 4. Fake Agent Integration tests | Verify Server-to-Agent proxy and error mapping without real Agent | Yes | In-process mux | Fake `httptest.Server` | No | No | Strong | Agent unreachable, Docker inspect error, response schema drift, proxy payload shape | Real Docker daemon behavior | Fake Agent schema must mirror real Agent endpoints |
| 5. Local Shell API-only E2E | Verify real server process + real DB/API workflows | Not daily; local opt-in | Yes | No or optional | No | No | Medium if services can run | Packaging/config/env/login/CSRF issues, real binary behavior | Agent/Docker/GPU/container behavior | Tier A: controlled local process or existing server |
| 6. Real Agent Smoke | Verify real server + agent + Docker-facing endpoints | Not daily; local opt-in | Yes | Yes | Docker endpoint access | Usually no model GPU load | Local only | `/docker-images`, `/docker-image-inspect`, `/files`, `/model-paths/scan`, lightweight Docker lifecycle | Full model-serving correctness | Tier B: requires Agent and Docker, may not need GPU model load |
| 7. Real Container Smoke | Verify full model serving containers | Not daily; local opt-in | Yes | Yes | Yes | Yes | No, unless dedicated GPU runner | vLLM/SGLang/llama.cpp start, health, `/v1/models`, chat, logs, stop | Fine-grained API contract regressions | Tier C: expensive final real-machine validation |
| 8. UI Static / Component tests | Verify build, i18n, path usage, status mapping, component presence | Yes via `npm --prefix web test` and build | No | No | No | No | Strong | i18n leaks, TS/Vue compile errors, status mapping, static binding mistakes | Real API behavior, browser rendering fidelity | Keep these fast and deterministic |
| 9. Manual UI checks | Verify visual UX and final user interaction | No | Usually yes | Optional | Optional | Optional | No | Visual regressions, drawer layout, tag color, wording | Business correctness by itself | Use after API correctness passes, not as the main gate |

## 5. Go API Workflow E2E Harness Design

### 5.1 Router and Test App

The repository already has:

```go
func SetupRoutes(mux *http.ServeMux, cfg RouterConfig)
```

The recommended harness should add a test-only helper under `internal/server/api`, for example:

```text
internal/server/api/api_workflow_test_helper_test.go
```

Responsibilities:

- Open isolated test DB, normally `db.Open(":memory:")`.
- Run `database.Migrate()`.
- Initialize bootstrap admin with deterministic credentials.
- Build `auth.SessionStore`, `auth.AuthHandler`, `rbac.Handler`, `AgentHandler`, `ResourceHandler`, and metrics as needed.
- Register routes through `SetupRoutes`.
- Return a `WorkflowTestApp` with:
  - `DB *db.DB`
  - `Mux *http.ServeMux`
  - optional `Server *httptest.Server`
  - `Client *WorkflowClient`
  - `AdminUsername`, `AdminPassword`
  - fake Agent registry handles
  - cleanup functions

The first version can use `mux.ServeHTTP` directly. Use `httptest.Server` only where URL construction, cookies, or real `http.Client` behavior is needed.

### 5.2 Auth, Session, CSRF, Tenant, Admin

Use real auth for router contract and workflow tests unless a test is explicitly a unit-level handler test.

Recommended flow:

1. Bootstrap admin:
   - username: `admin`
   - password: `test1234`
   - force password change: false
2. Login through `POST /api/v1/auth/login`.
3. Store returned session cookie in the test client cookie jar.
4. Store returned `csrf_token`.
5. Add `Origin` and `X-CSRF-Token` for mutating requests.
6. Use default tenant UUID from `database.DefaultTenantID()`.

Direct context injection with `adminSession()` should remain for narrow handler tests only. Workflow tests should exercise middleware.

### 5.3 Test Auth Helper

Add a test helper, not a production bypass:

```text
loginAsAdmin(t)
apiJSON(t, method, path, body, wantStatus)
apiRaw(t, method, path, body)
```

The helper should:

- Fail tests with request path, response code, and response body.
- Preserve response JSON for assertions.
- Keep cookie/CSRF behavior real.
- Avoid duplicating auth code across workflow tests.

### 5.4 Test DB Initialization and Fixtures

Each workflow test should use a fresh DB or a fresh named fixture scope.

Default fixtures:

| Fixture | How to create | Notes |
| --- | --- | --- |
| Default tenant | `Migrate()` seed | Use UUID tenant, not literal `default` |
| Admin user | `auth.InitBootstrap` | password `test1234` |
| Node | Insert via Agent registration API or DB fixture | Prefer Agent registration API for router workflows |
| GPU | Resource report fixture or DB insert | Use bytes and nullable semantics correctly |
| Backend / BackendVersion / BackendRuntime | DB migration/catalog seed | Do not mutate system templates except clone paths |
| ModelArtifact / ModelLocation | API workflow fixture | Needed for deployment workflows |
| NodeBackendRuntime | API `enable` | Avoid direct DB insert in workflow tests unless setup-only |

Fixture names should use:

```text
e2e-<test-name>-<short-random>
```

No test should rely on resources created by another test.

### 5.5 Fake Agent Design

Use one shared fake Agent helper:

```text
internal/server/api/fake_agent_test.go
```

Recommended capabilities:

- `GET /healthz`
- `GET /docker-images`
- `GET /docker-image-inspect?ref=...`
- `GET /files`
- `POST /model-paths/scan`
- task polling/result endpoints if needed by lifecycle tests
- configurable status code, latency, and body per endpoint

The fake Agent should support named scenarios:

| Scenario | Expected server outcome |
| --- | --- |
| inspect success, list has image | `ready` or `ready_with_warnings` |
| inspect success, list misses image | not `missing_image` |
| inspect not found | `missing_image` |
| inspect 500 | `inspect_failed` or `docker_error` depending actual endpoint contract |
| agent unreachable | `agent_unreachable` |
| malformed JSON | precise decode/proxy error, not `missing_image` |

Fake responses must follow real Agent schema. For Docker image list, use the actual fields:

```json
{
  "images": [
    {
      "repository": "vllm/vllm-openai",
      "tag": "latest",
      "image_ref": "vllm/vllm-openai:latest",
      "image_id": "sha256:..."
    }
  ]
}
```

For inspect, include the real Docker-style fields the server maps into probe results:

```json
{
  "inspect": {
    "Id": "sha256:...",
    "RepoTags": ["vllm/vllm-openai:latest"],
    "Config": {
      "Entrypoint": ["python3", "-m", "vllm.entrypoints.openai.api_server"],
      "Cmd": [],
      "Env": ["PATH=/usr/local/bin"]
    },
    "Size": 123456789
  }
}
```

### 5.6 Cleanup and Isolation

For in-memory DB tests, cleanup is mostly per-test DB teardown. Still assert cleanup behavior through API:

- After `DELETE`, `GET detail` should return 404 or equivalent not found.
- After `DELETE`, list should not include the resource.
- Downstream resources should not be left visible.

For any test using temp files or fake catalog dirs, use `t.TempDir()` and `t.Cleanup(...)`.

### 5.7 Naming, Directory, and Run Commands

Recommended file layout:

```text
internal/server/api/api_workflow_test_helper_test.go
internal/server/api/fake_agent_test.go
internal/server/api/workflow_nbr_probe_test.go
internal/server/api/workflow_backend_runtime_test.go
internal/server/api/workflow_model_wizard_test.go
internal/server/api/workflow_deployment_runplan_test.go
internal/server/api/workflow_lifecycle_test.go
```

Run commands:

```bash
go test ./internal/server/api/... -count=1
go test ./internal/server/api/... -run 'TestWorkflow' -count=1 -v
go test ./... -count=1
```

Rules:

- Fake-agent API Workflow tests should be eligible for `go test ./internal/server/api/...`.
- Keep expensive or real Docker tests behind explicit env flags.
- Do not put real Docker/GPU/model container tests into default `go test ./...`.

## 6. Shell E2E Harness Design

Shell E2E remains valuable because it catches local packaging, config, process startup, Docker, GPU, and model-serving failures that Go fake-agent tests cannot catch.

It should be standardized into reusable helper files:

```text
scripts/e2e/lib/env.sh
scripts/e2e/lib/api-client.sh
scripts/e2e/lib/assert.sh
scripts/e2e/lib/resources.sh
scripts/e2e/lib/docker.sh
scripts/e2e/lib/report.sh
scripts/e2e/lib/cleanup.sh
```

This is a future implementation plan. This strategy document does not add those files.

### 6.1 Unified Environment Variables

Use one naming convention:

| Variable | Default | Purpose |
| --- | --- | --- |
| `LIGHTAI_E2E_MODE` | `existing-env` | `existing-env` or `fixture` |
| `LIGHTAI_SERVER_URL` | `http://127.0.0.1:18080` | Server base URL |
| `LIGHTAI_AGENT_URL` | `http://127.0.0.1:19091` | Agent endpoint URL |
| `LIGHTAI_PROMETHEUS_URL` | `http://127.0.0.1:19090` | Optional Prometheus URL |
| `LIGHTAI_GRAFANA_URL` | `http://127.0.0.1:13000` | Optional Grafana URL |
| `LIGHTAI_E2E_USERNAME` | `admin` | Login user |
| `LIGHTAI_E2E_PASSWORD` | `test1234` | Login password |
| `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` | `test1234` | Fixed test password when starting fixture server |
| `LIGHTAI_E2E_RUN_ID` | timestamp + pid | Resource/evidence suffix |
| `LIGHTAI_E2E_PREFIX` | `e2e-<run-id>` | Resource name prefix |
| `LIGHTAI_E2E_ARTIFACT_DIR` | `docs/reports/.../e2e-run-<run-id>` or `/tmp/...` | Evidence output |
| `LIGHTAI_E2E_KEEP_ON_FAIL` | `1` | Preserve resources/evidence on failure |
| `LIGHTAI_E2E_CLEAN_ON_PASS` | `1` | Cleanup on success |
| `LIGHTAI_E2E_START_SERVICES` | `0` | Whether script may start server/agent |
| `LIGHTAI_E2E_REQUIRE_DOCKER` | per tier | Fail/skip rule |
| `LIGHTAI_E2E_REQUIRE_GPU` | per tier | Fail/skip rule |
| `VLLM_IMAGE` | `vllm/vllm-openai:latest` | vLLM image |
| `SGLANG_IMAGE` | `lmsysorg/sglang:latest` | SGLang image |
| `LLAMACPP_IMAGE` | `ghcr.io/ggml-org/llama.cpp:server-cuda13` | llama.cpp image |
| `HF_MODEL_PATH` | local Qwen HF path | HuggingFace model path |
| `GGUF_MODEL_PATH` | local GGUF path | GGUF file path |

### 6.2 Fixed Test Password

Fixture mode should start server with:

```bash
LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD=test1234
```

Existing-env mode should not reset or mutate credentials. It should login with `LIGHTAI_E2E_USERNAME` and `LIGHTAI_E2E_PASSWORD`; if login fails, fail with a clear message.

### 6.3 Start or Reuse Services

Two modes:

| Mode | Behavior |
| --- | --- |
| `existing-env` | Never start/kill server or agent. Verify URLs and fail/skip if missing. |
| `fixture` | Start server/agent from built binaries in temp dirs, record exact PIDs, clean only those PIDs. |

Never use broad `pkill`, broad `docker rm`, or broad file removal. Cleanup must use PIDs, container IDs, labels, and resource IDs created by the current run.

### 6.4 Readiness Checks

Standard checks:

- Server: `GET /healthz`
- Auth: `POST /api/v1/auth/login`
- Node list: `GET /api/v1/nodes`
- Agent: `GET $LIGHTAI_AGENT_URL/healthz` or known Agent endpoint
- Docker: `docker info`
- Image: `docker image inspect <image>`
- GPU: `nvidia-smi` and optional Docker GPU smoke
- Model path: `test -e <path>`
- Ports: `ss -tlnp` and only fail on ports needed by the current flow

### 6.5 Login, Cookie, and CSRF

`api-client.sh` should provide:

```bash
e2e_login
e2e_api GET /api/v1/nodes
e2e_api POST /api/v1/... '{"json":true}'
e2e_api_expect 409 POST /api/v1/... '{}'
e2e_api_save response-name GET /api/v1/...
```

Rules:

- Always send `Origin: $LIGHTAI_SERVER_URL`.
- Store cookies in a per-run cookie jar.
- Add `X-CSRF-Token` to mutating methods.
- Capture response body and status separately.
- Save failed response bodies into evidence.

### 6.6 JSON Assertions

Use Python for JSON assertions, not ad hoc grep, where structure matters:

```bash
assert_json_field response.json id
assert_json_eq response.json status ready
assert_json_array_contains response.json data id "$RESOURCE_ID"
assert_json_not_contains_status response.json missing_image
```

String assertions can remain for command preview text, but JSON fields should be structural.

### 6.7 Resource Naming

All created names should begin with:

```text
e2e-<timestamp>-<flow>
```

Store resource IDs in:

```text
$LIGHTAI_E2E_ARTIFACT_DIR/resources.env
```

This enables deterministic cleanup and post-failure investigation.

### 6.8 Cleanup

Success behavior:

- Stop created deployments.
- Delete deployments, artifacts, model roots, cloned runtimes, and NBRs created by this run.
- Remove containers created by this run by exact container ID or exact label.
- Write `summary.json` with `status=PASS`.

Failure behavior:

- Preserve evidence by default.
- Preserve containers by default for real container tests unless `LIGHTAI_E2E_CLEAN_ON_FAIL=1`.
- Write `failure-reason.txt`, `summary.json`, response bodies, logs, and known resource IDs.

### 6.9 Evidence Report

Every shell E2E should write:

```text
summary.json
summary.md
resources.env
environment.txt
requests/
responses/
logs/
failure-reason.txt if failed
```

For real container smoke, also write:

```text
docker-inspect-<container>.json
docker-logs-<container>.txt
v1-models.json
chat-response.json
```

### 6.10 Skip / Fail Rules

Use these rules consistently:

| Condition | API-only | Real Agent/Docker | Real Container |
| --- | --- | --- | --- |
| Server unavailable in existing-env | FAIL | FAIL | FAIL |
| Login fails | FAIL | FAIL | FAIL |
| No node when required | FAIL or SKIP based on declared test | FAIL | FAIL |
| Docker missing | SKIP | FAIL | FAIL |
| GPU missing | SKIP | SKIP unless declared required | FAIL |
| Image missing | SKIP for optional backend | FAIL for targeted backend |
| Model path missing | SKIP for optional backend | SKIP | FAIL for targeted backend |
| Endpoint returns wrong status | FAIL | FAIL | FAIL |
| Cleanup API fails on pass | FAIL | FAIL | FAIL |

## 7. Controlled Local Real Environment

Default ports:

| Component | Port |
| --- | --- |
| Server | `18080` |
| Agent metrics/API | `19091` |
| Prometheus | `19090` |
| Grafana | `13000` |
| Vite dev server | `15173` |

Known local real environment:

- NVIDIA GPU available.
- Docker available.
- Target images available:
  - `vllm/vllm-openai:latest`
  - `lmsysorg/sglang:latest`
  - `ghcr.io/ggml-org/llama.cpp:server-cuda13`
- Model paths available.

Real container smoke should use short but realistic timeouts:

| Backend | Health endpoint | Typical timeout |
| --- | --- | --- |
| vLLM | `/v1/models` | 180s |
| SGLang | `/health`, fallback `/v1/models` | 180s |
| llama.cpp | `/v1/models` | 90s |

## 8. Shell E2E Tiers

### Tier A: API-only Local E2E

Depends on:

- Running server and DB.

Does not depend on:

- Real Agent.
- Docker.
- GPU.
- Model container.

Use for:

- Runtime Wizard API flow.
- NBR Wizard API flow when using fake/pre-seeded state.
- Model Wizard API flow when not browsing real Agent files.
- Deployment preflight/runplan API flow.

### Tier B: Real Agent / Docker E2E

Depends on:

- Server.
- Agent.
- Docker.

Use for:

- `/docker-images`
- `/docker-image-inspect`
- `/files`
- `/model-paths/scan`
- Docker inspect metadata/probe.
- Lightweight container lifecycle.

### Tier C: Real Model Container E2E

Depends on:

- Server.
- Agent.
- Docker.
- GPU.
- Model files.

Use for:

- vLLM container smoke.
- SGLang container smoke.
- llama.cpp container smoke.
- start -> health -> `/v1/models` or `/v1/chat/completions` -> logs -> stop -> cleanup.

## 9. Cross-API Stability Assertions

Use these assertion templates in Go API Workflow and shell E2E:

| Assertion | Template |
| --- | --- |
| Create response fields | `id`, `name`, `tenant_id` or ownership field, timestamps, and workflow-specific JSON fields are present |
| GET detail fields | Every create/list critical field appears in detail and keeps type |
| GET list fields | Created resource appears in list and contains enough fields for Web selection |
| PATCH preserves fields | Patch one field, then assert all unmodified fields remain unchanged |
| DELETE invisibility | After delete, detail is not found and list no longer contains the ID |
| Upstream/downstream ID use | Downstream API uses IDs returned by previous API, never hard-coded except system catalog IDs |
| List/detail consistency | A resource selected from list can be fetched by detail using the same ID |
| Check/probe/preflight usability | Result from check/probe/preflight can be used in the next API step |
| JSON field preservation | Assert `config_snapshot_json`, `probe_results_json`, `run_plan_json`, `diagnostics_json`, `metadata_json`, `locations_json`, `capabilities_json` are not dropped |
| Error status mapping | Agent unreachable, Docker error, inspect error, not found, validation error map to exact statuses |
| Operation traceability | `operation_id`, audit records, task logs, diagnostics, and run plan IDs are correlated |
| Cleanup success | Resources created by the test are deleted or confirmed absent |
| Failure evidence | Response body, request payload, logs, and resource IDs are preserved |

## 10. Key API Workflow Breakdown

### A. BackendRuntime / Runtime Wizard

Flow:

1. List backends.
2. List backend versions.
3. List system runtime templates.
4. Clone system template into user runtime.
5. PATCH image, env, ports, volumes, devices, extra args, and supported Docker/security fields.
6. GET detail.
7. List verify.
8. DELETE cleanup.

Assertions:

- `config_snapshot_json` and related Docker/config fields are not lost.
- PATCH does not drop unmodified fields.
- System template and user runtime are not confused.
- User clone is editable; system template remains read-only.

### B. NodeBackendRuntime / NBR Wizard

Flow:

1. List nodes.
2. List node Docker images.
3. Select `vllm/vllm-openai:latest`.
4. Create/update NodeBackendRuntime.
5. `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe`.
6. `GET /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe`.
7. List NBR.
8. GET NBR detail.
9. Cleanup.

Assertions:

- `probe_results_json` survives list/detail mapping.
- `image_id`, `repo_tags`, `entrypoint`, `cmd`, `size`, `backend_match_status`, and `final_status` are present.
- Only `not-exist/lightai-test:missing` or equivalent inspect-not-found maps to `missing_image`.
- Agent unreachable, Docker error, and inspect error do not map to `missing_image`.
- `check-request` remains backward compatible.

### C. Model Wizard / ModelArtifact / ModelLocation

Flow:

1. List nodes.
2. Browse node files.
3. Scan model path.
4. Create `ModelArtifact`.
5. Add `ModelLocation`.
6. List artifacts.
7. GET detail.
8. Delete cleanup.

Assertions:

- `size`, `format`, `arch`, `checksum`, `capabilities_json`, and `locations_json` are preserved.
- Multi-node locations are not mixed.
- Location deletion does not delete unrelated artifacts or locations.

### D. Deployment / Preflight / RunPlan

Flow:

1. Create or select `ModelArtifact` and `ModelLocation`.
2. Create or select `NodeBackendRuntime`.
3. Create deployment.
4. Preflight.
5. Dry-run / runplan preview.
6. Cleanup.

Assertions:

- RunPlan includes image, model path, ports, volumes, devices, env, health check, command, and args.
- NBR snapshot is frozen.
- RunPlan uses NBR config snapshot and model location, not live mutable frontend state.

### E. Start / Logs / Stop Lifecycle

Flow:

1. Start deployment.
2. Task claim.
3. Fake Agent task result success/failure.
4. Status.
5. Logs.
6. Stop.
7. Failed state.
8. Cleanup.

Assertions:

- `operation_id`, task ID, generation, status, diagnostics, and audit records remain correlated.
- Failure preserves diagnostics and logs access.
- Stop is idempotent when container already stopped/missing.

### F. Real Container Smoke

Flow:

1. vLLM real container.
2. SGLang real container.
3. llama.cpp real container.
4. Start.
5. Health check.
6. `/v1/models` or `/v1/chat/completions`.
7. Logs.
8. Stop.
9. Cleanup.

Assertions:

- Short timeout per backend.
- Failure preserves container logs and API evidence.
- Success cleans created LightAI resources and containers.

## 11. Existing Shell E2E Script Review

Legend:

- Type: A = API-only local E2E, B = Real Agent/Docker, C = Real Model Container, Mixed = spans tiers.
- Login/Cleanup/Evidence: `yes`, `partial`, or `no`.

| File | Current purpose | Type | Server | Agent | Docker | GPU | Model files | Unified login | Cleanup | Negative path | Evidence report | Keep | Refactor | Target tier |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh` | vLLM default + modified model-runtime wizard to container via product API | C | yes | yes | yes | yes | yes | partial via shared helper | partial | no | partial | yes | yes, move to tier C harness | C |
| `scripts/e2e-model-runtime-wizard-nvidia-sglang.sh` | SGLang model-runtime wizard to container | C | yes | yes | yes | yes | yes | local duplicate | partial | no | partial | yes | yes, replace duplicate helper code | C |
| `scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh` | llama.cpp model-runtime wizard to container | C | yes | yes | yes | yes | yes | local duplicate | partial | no | partial | yes | yes, replace duplicate helper code | C |
| `scripts/e2e-model-runtime-wizard-nvidia-matrix.sh` | Wrapper for backend matrix default/modified runs | C | yes | yes | yes | yes | yes | inherited | partial | no | yes | yes | yes, become tier C orchestrator | C |
| `scripts/e2e-model-runtime-wizard-nvidia-api.sh` | vLLM product API chain with build/start fallback and `/v1/models` | C | yes | yes | yes | yes | yes | local duplicate | yes | no | partial | yes | yes, split service management from flow | C |
| `scripts/e2e-model-runtime-api.sh` | Generic model runtime API flow | Mixed | yes | likely | likely | likely | likely | local duplicate | partial | partial | partial | yes | yes, classify exact cases into A/B/C | Mixed |
| `scripts/e2e-model-runtime-local.sh` | Fixture-mode local full lifecycle; builds server/agent and runs real Docker/GPU path | C | starts fixture | starts fixture | yes | yes | yes | fixture-specific | yes | partial | partial | yes | yes, preserve fixture-mode ideas | C |
| `scripts/e2e-model-runtime-failed-instance-logs.sh` | Forces failure and verifies failed instance logs/diagnostics | C | yes | yes | yes | yes | yes | local duplicate | yes | yes | yes | yes | yes, move to lifecycle/diagnostics tier C | C |
| `scripts/e2e-real-smoke-all-three.sh` | Real container smoke for vLLM/SGLang/llama.cpp | C | yes | yes | yes | yes | yes | local duplicate | partial | no | partial | yes | yes, align with C harness and preserve logs on fail | C |
| `scripts/e2e-instance-stop-real-llamacpp.sh` | Real llama.cpp start/stop cleanup behavior | C | yes | yes | yes | yes | yes | local duplicate | partial | partial | partial | yes | yes | C |
| `scripts/e2e-dryrun-parameter-matrix-enhanced.sh` | Dry-run parameter matrix, no container start | A | yes | no direct | no direct | no | existing artifacts | local duplicate | partial | partial | yes | yes | yes, move under API-only harness | A |
| `scripts/e2e-matrix-verifier.sh` | Cross-backend dry-run matrix over available runtimes | A | yes | no direct | no direct | no | existing artifacts | local duplicate | partial | partial | yes | yes | yes | A |
| `scripts/e2e-inference-parser-llamacpp.sh` | Mixed fixture/unit plus real llama.cpp inference parser checks | Mixed | yes | maybe | yes for real part | yes for real part | GGUF | local duplicate | partial | partial | yes | yes | yes, split fixture and real-container portions | Mixed -> A/C |
| `scripts/e2e-runplan-parameter-source-audit.sh` | Dry-run parameter propagation audit | A | yes | no direct | only counts managed containers | no | existing artifacts | local duplicate | partial | yes | yes | yes | yes | A |
| `scripts/e2e-backend-runtime-nvidia-api.sh` | vLLM full API/container chain with service startup fallback | C | yes | yes | yes | yes | yes | local duplicate | yes | partial diagnostics | partial | yes | yes, split real container from API-only assertions | C |
| `scripts/e2e-clone-template-parameter-persistence.sh` | Clone template persistence API check | A | yes | no | no | no | no | local duplicate | yes | no | partial | yes | yes, move to BackendRuntime CRUD Chain | A |
| `scripts/e2e-deployment-visibility-selected.sh` | Deployment list/detail/dry-run/delete visibility | A | yes | no direct | no | no | existing artifact | local duplicate | yes | yes | yes | yes | yes | A |
| `scripts/e2e-runtime-config-copy-first-save-selection.sh` | Runtime clone/copy first-save selection behavior | A/B mixed due NBR check | yes | maybe | maybe via check | no | existing artifacts | local duplicate | yes | partial | yes | yes | yes, split API-only clone from real check | A/B |
| `scripts/e2e-runtime-config-web-check-flow.sh` | Real Docker image check-request regression | B | yes | yes | yes | no model load | no | shared model-runtime helper | no explicit full cleanup | yes | yes | yes | yes, convert to NBR probe tier B | B |
| `scripts/e2e-ui-persistence-runplan-selected.sh` | API flow for UI persistence/runplan selected state | Mixed | yes | maybe | start path may create tasks | maybe | synthetic/existing | local duplicate | no full cleanup visible | yes | yes | yes | yes, rename away from UI and classify by dependency | A/B |
| `scripts/smoke-model-backends.sh` | Direct Docker smoke for vLLM/SGLang/llama.cpp outside LightAI API | C, external baseline | no | no | yes | yes | yes | no | yes | no | partial | yes | yes, keep as environment baseline not product E2E | C baseline |
| `scripts/verify-local.sh` | Local service/metrics/web readiness verification | B readiness | yes | yes metrics | no | no | no | no auth | no | yes for protected API expectation | no | yes | yes, make readiness helper/report | B readiness |

Summary:

- Keep the scripts, but stop treating them as one flat E2E set.
- Convert API-only scripts to tier A after the harness exists.
- Convert Docker/probe scripts to tier B after the harness exists.
- Convert full model container scripts to tier C after the harness exists.
- Split mixed scripts where possible instead of growing them.
- Preserve `scripts/e2e/lib/e2e-assert.sh` ideas, but replace duplicate login/API code with unified helpers.

## 12. Phased Route

High-level route:

1. Step 0: Confirm API-first + controllable local Shell E2E principle.
2. Step 1: Add Go API Workflow test harness.
3. Step 2: Add Shell E2E harness design/minimal helpers.
4. Step 3: Implement first Go API Workflow vertical slice: NBR Probe Chain.
5. Step 4: Implement BackendRuntime CRUD Chain.
6. Step 5: Implement Model Wizard Chain.
7. Step 6: Implement Deployment Preflight/RunPlan Chain.
8. Step 7: Implement Start/Logs/Stop Chain.
9. Step 8: Reorganize existing Shell E2E into API-only / real-agent / real-container.
10. Step 9: Run local real vLLM / SGLang / llama.cpp smoke.
11. Step 10: Consider CI for tests that do not require real Docker/GPU.

The detailed implementation roadmap is in:

```text
docs/reports/phase-3/e2e-implementation-roadmap.md
```

## 13. Recommended First Vertical Slice

Start with Go API Workflow E2E for NBR Probe Chain.

Why:

- It directly covers the recent NBR Image Probe regression class.
- It exercises real route patterns for the new `/probe` endpoints.
- It proves `ImageInspect` authority and `missing_image` mapping.
- It validates list/detail `probe_results_json` preservation.
- It is deterministic with a fake Agent and does not require Docker/GPU.

Minimum acceptance:

```bash
go test ./internal/server/api/... -run 'TestWorkflowNBRProbe' -count=1 -v
go test ./internal/server/api/... -count=1
```

## 14. Real Container Smoke Plan

After API workflow tests and shell harness cleanup, run tier C real container smoke in this order:

1. llama.cpp: usually fastest startup and GGUF path coverage.
2. vLLM: validates OpenAI-compatible HF serving and GPU allocation.
3. SGLang: validates launch module, shared memory, and health fallback behavior.

Each backend should:

- Use product API chain, not direct `docker run`, for LightAI E2E.
- Save run plan and Docker spec evidence.
- Wait for health endpoint.
- Call `/v1/models`.
- Call a small `/v1/chat/completions` request when supported.
- Fetch logs through LightAI API.
- Stop deployment.
- Cleanup on success and preserve evidence on failure.

`scripts/smoke-model-backends.sh` can remain as an external baseline that proves the host can run the images outside LightAI. It should not replace product-path E2E.

## 15. What Not To Do Now

Do not do the following in this E2E harness phase:

- Add new business APIs.
- Add DB migrations.
- Add browser automation.
- Add Playwright or Cypress.
- Continue Phase 5.
- Add Backend Match catalog.
- Add Script Probe.
- Add Version Probe.
- Start real model containers as part of the design-only phase.
- Make UI/browser smoke the main acceptance gate.
- Collapse all tests into one full test.

## 16. Questions To Confirm

1. Should Go API Workflow tests use real login by default, with direct session context only for legacy handler tests?
2. Should `go test ./...` include all fake-agent API Workflow tests, or should some workflow tests be run only by `go test ./internal/server/api/...` first?
3. Should shell fixture mode be allowed to start server/agent automatically, or should the first harness only support existing-env mode?
4. Should evidence for local shell E2E be written under `docs/reports/...` by default, or under `/tmp` unless explicitly promoting a run?
5. Should the existing untracked `docs/reports/phase-3/api-first-e2e-review-and-plan.md` be committed as an input audit artifact in the same docs series?
