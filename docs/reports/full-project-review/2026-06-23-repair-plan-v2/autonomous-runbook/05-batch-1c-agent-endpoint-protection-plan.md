# Batch 1C: Agent Endpoint Protection / NBR Boundary — Detailed Plan

---

## Goal
Protect agent management endpoints with auth. NBR-defined params flow through unblocked.

## Agent Endpoint Matrix (from cmd/agent/main.go:291-476)

| Endpoint | Handler | Current Auth | Target Auth | Notes |
|----------|---------|-------------|-------------|-------|
| GET /healthz | healthMux.HandleFunc | None | **None** | Load balancer needs |
| GET /metrics | healthMux.Handle(promhttp) | None | **None** | Prometheus scrape |
| GET /docker-images | healthMux.HandleFunc | None | **Bearer token** | Management endpoint |
| GET /docker-image-inspect | healthMux.HandleFunc | None | **Bearer token** | Management endpoint |
| GET /files | healthMux.HandleFunc | None | **Bearer token** | Filesystem browse |
| POST /model-paths/scan | healthMux.HandleFunc | None | **Bearer token** | Model scan |

## Implementation

### Auth Middleware (cmd/agent/main.go)
```go
func requireAgentToken(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != agentToken {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}
```

### Apply to Endpoints
```go
healthMux.HandleFunc("GET /docker-images", requireAgentToken(handleDockerImages))
healthMux.HandleFunc("GET /docker-image-inspect", requireAgentToken(handleDockerImageInspect))
healthMux.HandleFunc("GET /files", requireAgentToken(handleFiles))
healthMux.HandleFunc("POST /model-paths/scan", requireAgentToken(handleModelPathsScan))
```

### Server Proxy Token Passing
AgentClient (Batch 1B) already sends `Authorization: Bearer {agentToken}` header. No additional changes needed.

## NBR Boundary (Do NOT Do)

- Do NOT block NBR-defined privileged/ipc/devices/security-opt/group-add
- Do NOT add vendor policy engine
- Do NOT add privileged approval
- Do NOT add MetaX/Huawei/NVIDIA allowlist

## Collector Validation (internal/agent/collector/external.go:171)

Current: `cmdStr` from config passed to `sh -c`.
Fix: Validate command path exists and is executable before `sh -c`.

## Commits

1. `feat: add auth middleware to agent HTTP endpoints`
2. `fix: validate collector command path`

## Non-Regression

| Check | Method |
|-------|--------|
| /healthz unauthenticated | curl http://agent:19091/healthz → 200 |
| /metrics unauthenticated | curl http://agent:19091/metrics → 200 |
| /docker-images requires auth | curl http://agent:19091/docker-images → 401 |
| /docker-images with token | curl -H "Authorization: Bearer {token}" → 200 |
| Server proxy still works | GET /api/v1/nodes/{id}/docker-images → 200 |
| NBR params not blocked | RunPlan preview shows all NBR params |
