# Golden Path Baseline Plan

> Date: 2026-06-23
> Absorbs and corrects: `09-golden-path-before-baseline.md`

---

## 1. Environment Capability Check

Run BEFORE starting any batch:

```bash
go version
node --version 2>/dev/null || echo "Node.js not available"
npm --version 2>/dev/null || echo "npm not available"
docker version 2>/dev/null || echo "Docker not available"
nvidia-smi 2>/dev/null || echo "nvidia-smi not available"
ls scripts/e2e-*.sh scripts/*smoke*.sh 2>/dev/null || echo "No E2E scripts"
ls configs/backend-catalog/ 2>/dev/null || echo "No catalog"
```

---

## 2. Quick Baseline Commands

Run BEFORE each batch. All must pass (or fail for known/environment reasons):

```bash
# Compilation
go build ./cmd/server/...
go build ./cmd/agent/...

# Unit tests
go test ./internal/server/...
go test ./internal/agent/...
go test ./internal/server/runplan/...

# Frontend
cd web && npm run build
cd web && npm test
```

**Failure triage**:
- Compilation error → code issue, must fix before proceeding
- Test failure in target batch code → expected, will fix in batch
- Test failure in unrelated code → STOP, investigate
- Frontend build failure → environment or code issue

---

## 3. Full Golden Path Verification

### 3A. Can Verify Without Full Stack

| Flow | Command | Requires |
|------|---------|----------|
| Compilation | `go build ./...` | Go |
| Unit tests | `go test ./internal/...` | Go |
| RunPlan tests | `go test ./internal/server/runplan/...` | Go |
| Frontend build | `cd web && npm run build` | Node.js |
| Frontend tests | `cd web && npm test` | Node.js |
| Race detection | `go test -race ./internal/...` | Go |

### 3B. Requires Running Server + Agent

| Flow | Command | Requires |
|------|---------|----------|
| Login | `curl -X POST http://127.0.0.1:18080/api/v1/auth/login` | Server |
| Nodes list | `curl http://127.0.0.1:18080/api/v1/nodes` | Server |
| File browse | `curl http://127.0.0.1:18080/api/v1/nodes/{id}/files?path=/` | Server+Agent |
| Model scan | `curl -X POST http://127.0.0.1:18080/api/v1/nodes/{id}/model-paths/scan` | Server+Agent |
| Docker images | `curl http://127.0.0.1:18080/api/v1/nodes/{id}/docker-images` | Server+Agent |
| NBR list | `curl http://127.0.0.1:18080/api/v1/nodes/{id}/backend-runtimes` | Server |

### 3C. Requires Real GPU + Docker

| Flow | Script | Requires |
|------|--------|----------|
| llama.cpp smoke | `scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh` | GPU+Docker+Model |
| vLLM smoke | `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh` | GPU+Docker+Model |
| SGLang smoke | `scripts/e2e-model-runtime-wizard-nvidia-sglang.sh` | GPU+Docker+Model |
| Full smoke | `scripts/e2e-real-smoke-all-three.sh` | GPU+Docker+Model |

### 3D. Mock/Dry-Run Alternatives

When real GPU not available:
- RunPlan dry-run: `go test ./internal/server/runplan/...`
- Equivalent Docker command: RunPlan preview in API
- Docker image check: `curl http://127.0.0.1:18080/api/v1/nodes/{id}/docker-images`
- Model path check: `curl -X POST http://127.0.0.1:18080/api/v1/nodes/{id}/model-paths/scan`

---

## 4. Before/After Template

```markdown
## Batch {X} Closeout

### Before
- Commit: {SHA}
- go build: PASS/FAIL
- go test ./internal/server/...: PASS/FAIL ({N} tests)
- go test ./internal/agent/...: PASS/FAIL ({N} tests)
- cd web && npm run build: PASS/FAIL
- cd web && npm test: PASS/FAIL
- Golden path: {list status}

### After
- Commit: {SHA}
- go build: PASS/FAIL
- go test ./internal/server/...: PASS/FAIL ({N} tests)
- go test ./internal/agent/...: PASS/FAIL ({N} tests)
- cd web && npm run build: PASS/FAIL
- cd web && npm test: PASS/FAIL
- Golden path: {list status}
- New tests: {count}

### Not Verified
- {list with reason}
```
