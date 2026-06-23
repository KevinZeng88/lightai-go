# Golden Path Before Baseline

> Date: 2026-06-23
> Purpose: Baseline state before Batch 1A/1B execution

---

## 1. Current Git State

```
Commit: d61c409f010e8ba6edaf1b4249a3c33a159a1d2f
Branch: main
```

**Working tree status**:
- `M VERSION` — stale from previous session (0.1.0 vs v0.1.9)
- `?? .mimocode/skills/` — untracked skills directory
- `?? docs/reports/full-project-review/` — untracked review documents

**Code modified this phase**: NONE — only documents created/modified.

---

## 2. Golden Path Verification Status

### 2A. Console Login & Navigation

| Step | Flow | Status | Notes |
|------|------|--------|-------|
| 1 | Login with admin credentials | NOT VERIFIED | Requires running server |
| 2 | Dashboard loads | NOT VERIFIED | Requires running server + browser |
| 3 | Nodes page lists nodes | NOT VERIFIED | Requires running server |
| 4 | GPUs page shows status | NOT VERIFIED | Requires running server |
| 5 | Backend/Runtime/NBR pages | NOT VERIFIED | Requires running server |
| 6 | Model Artifact/Location pages | NOT VERIFIED | Requires running server |
| 7 | Deployment/Instance pages | NOT VERIFIED | Requires running server |
| 8 | Logs/diagnostics | NOT VERIFIED | Requires running server |
| 9 | Observability pages | NOT VERIFIED | Requires running server + Prometheus/Grafana |

### 2B. Model File Flow

| Step | Flow | Status | Notes |
|------|------|--------|-------|
| 1 | Browse model roots/files | NOT VERIFIED | Requires running server + agent |
| 2 | Scan model paths | NOT VERIFIED | Requires running server + agent |
| 3 | Identify HF/GGUF models | NOT VERIFIED | Requires model files |
| 4 | Create/update ModelArtifact | NOT VERIFIED | Requires running server |
| 5 | Create/update ModelLocation | NOT VERIFIED | Requires running server |

### 2C. Runtime / NBR Flow

| Step | Flow | Status | Notes |
|------|------|--------|-------|
| 1 | Backend/BackendVersion catalog loads | NOT VERIFIED | Requires running server |
| 2 | BackendRuntime clone/edit | NOT VERIFIED | Requires running server |
| 3 | NBR create/patch/enable | NOT VERIFIED | Requires running server + agent |
| 4 | NBR check/check-request | NOT VERIFIED | Requires running server + agent |
| 5 | Docker image list via proxy | NOT VERIFIED | Requires running server + agent |
| 6 | Docker image inspect via proxy | NOT VERIFIED | Requires running server + agent |
| 7 | NBR params flow into RunPlan | NOT VERIFIED | Requires running server |

### 2D. Preflight / RunPlan Flow

| Step | Flow | Status | Notes |
|------|------|--------|-------|
| 1 | Preflight returns candidates | NOT VERIFIED | Requires running server + agent |
| 2 | RunPlan preview shows params | NOT VERIFIED | Requires running server |
| 3 | Equivalent Docker command | NOT VERIFIED | Requires running server |
| 4 | High-risk params not blocked | NOT VERIFIED | Requires NBR with high-risk params |

### 2E. Instance Start / Logs / Stop

| Step | Flow | Status | Notes |
|------|------|--------|-------|
| 1 | Deployment start creates instance | NOT VERIFIED | Requires running server + agent + Docker |
| 2 | Agent claims task | NOT VERIFIED | Requires running server + agent |
| 3 | Docker container starts | NOT VERIFIED | Requires Docker + GPU |
| 4 | Health check succeeds | NOT VERIFIED | Requires Docker + model |
| 5 | /v1/models accessible | NOT VERIFIED | Requires Docker + model |
| 6 | Chat/completion smoke | NOT VERIFIED | Requires GPU + model |
| 7 | Instance logs viewable | NOT VERIFIED | Requires running instance |
| 8 | Stop removes container | NOT VERIFIED | Requires running instance |
| 9 | Restart no name conflict | NOT VERIFIED | Requires running instance |
| 10 | Failed start cleanup | NOT VERIFIED | Requires simulated failure |

### 2F. Three Backend Smoke

| Backend | Real GPU | Dry-Run/Mock |
|---------|----------|-------------|
| llama.cpp CUDA/GGUF | NOT VERIFIED | NOT VERIFIED |
| vLLM OpenAI-compatible | NOT VERIFIED | NOT VERIFIED |
| SGLang OpenAI-compatible | NOT VERIFIED | NOT VERIFIED |

---

## 3. Baseline Commands

### Quick Verification (current environment)

```bash
# Compilation check
go build ./cmd/server/...
go build ./cmd/agent/...

# Unit tests
go test ./internal/server/...
go test ./internal/agent/...
go test ./internal/server/runplan/...

# Frontend build
cd web && npm run build

# Frontend tests
cd web && npm test
```

### Full Verification (requires running server + agent)

```bash
# Start server
go run ./cmd/server/...

# Start agent (separate terminal)
go run ./cmd/agent/...

# Login
curl -X POST http://127.0.0.1:18080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"..."}'

# List nodes
curl -H "Authorization: Bearer ..." http://127.0.0.1:18080/api/v1/nodes

# Browse files
curl -H "Authorization: Bearer ..." \
  "http://127.0.0.1:18080/api/v1/nodes/{id}/files?path=/models"

# Scan models
curl -X POST -H "Authorization: Bearer ..." \
  http://127.0.0.1:18080/api/v1/nodes/{id}/model-paths/scan

# Docker images
curl -H "Authorization: Bearer ..." \
  http://127.0.0.1:18080/api/v1/nodes/{id}/docker-images
```

### E2E Scripts (requires real GPU)

```bash
scripts/e2e-real-smoke-all-three.sh
scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
```

### Mock E2E (if available)

```bash
scripts/e2e-mock-smoke.sh
```

---

## 4. Before/After Comparison Template

For each batch closeout:

```markdown
## Batch {X} Closeout

### Before
- **Commit**: {SHA}
- **Working flows**: {list}
- **Key commands**: {list}
- **Evidence paths**: {list}
- **Not verified**: {list with reasons}

### Changes
- **Files created**: {list}
- **Files modified**: {list}
- **Logic changes**: {description}

### After
- **Same flows still work**: YES/NO per flow
- **Commands run**: {list with output}
- **If old script deleted**: {replacement}
- **If old config deleted**: {new config}
- **If flow can't verify**: {reason + dry-run evidence}

### Test Results
- `go test ./internal/server/authz/...`: PASS/FAIL
- `go test ./internal/server/agentclient/...`: PASS/FAIL
- `go test ./internal/server/api/... -run Tenant`: PASS/FAIL
- `go test -race ./internal/server/...`: PASS/FAIL
- `cd web && npm test`: PASS/FAIL

### Commit SHAs
- {commit 1}
- {commit 2}
- {commit 3}

### Golden Path Status
- {flow}: PASS/FAIL/SKIP (reason)

### Known Issues
- {list}

### Not Verified
- {list with reasons}
```

---

## 5. Current Environment Assessment

| Requirement | Available | Notes |
|------------|-----------|-------|
| Go compiler | YES | Can build and test |
| SQLite | YES | Server uses SQLite |
| Docker | UNKNOWN | Need to check |
| NVIDIA GPU | UNKNOWN | Need to check |
| Real models | UNKNOWN | Need to check |
| Browser | UNKNOWN | For Web UI verification |
| Prometheus | UNKNOWN | For observability |
| Grafana | UNKNOWN | For observability |

**Implication**: Batch 1A/1B unit tests can run without Docker/GPU. Integration tests (cross-tenant HTTP) need running server. Golden path verification needs full stack.

---

## 6. What Can Be Verified Without Full Stack

| Verification | Method | Requires |
|-------------|--------|----------|
| Code compiles | `go build ./...` | Go compiler |
| Unit tests pass | `go test ./internal/server/authz/...` | Go compiler |
| Unit tests pass | `go test ./internal/server/agentclient/...` | Go compiler |
| Existing tests pass | `go test ./internal/server/...` | Go compiler |
| Race detection | `go test -race ./internal/server/...` | Go compiler |
| Frontend builds | `cd web && npm run build` | Node.js |
| Frontend tests | `cd web && npm test` | Node.js |
| Cross-tenant HTTP | Integration test with test DB | Go compiler |
| SSRF blocked | Unit test with mock server | Go compiler |
| Agent proxy works | Need running server + agent | Full stack |
| Golden path full | Need full stack | Everything |
