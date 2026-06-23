# Batch 3: I/O / Audit / Log Safety — Detailed Plan

---

## Goal
Add body size limits, fix audit log JSON, fix redaction, add stream/task limits.

## Body Size Limit

**26 `json.NewDecoder(r.Body).Decode()` paths found** across:
- agent_handlers.go (4), deployment_lifecycle_handlers.go (4), artifact_handlers.go (5)
- runtime_handlers.go (3), backend_handlers.go (2), node_runtime_handlers.go (2)
- model_browser_handlers.go (2), preflight_handlers.go (1), resource_handlers.go (1)
- model_location_handlers.go (1), agent_proxy_handlers.go (1 io.ReadAll)

**Fix**: Add middleware in cmd/server/main.go:
```go
func bodyLimit(maxBytes int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
            next.ServeHTTP(w, r)
        })
    }
}
```
Default: 10MB. Apply to router.

## Audit Log JSON (agent_handlers.go:882-883)

**Current**:
```go
detail := fmt.Sprintf(`{"from_tenant_id":"%s","to_tenant_id":"%s","reason":"%s"}`, ...)
```

**Fix**:
```go
detailMap := map[string]string{"from_tenant_id": currentTenant, "to_tenant_id": req.TenantID, "reason": req.Reason}
detailBytes, _ := json.Marshal(detailMap)
detail := string(detailBytes)
```

## Redaction (helpers.go:225-234)

**Current**: Substring replacement corrupts `PASSWORD_CHANGED`.
**Fix**: Parse JSON key-value pairs, redact only values of sensitive keys.

## Docker Stream Limit (docker_real.go:256-293)

**Current**: `payloadLen` up to 4GB.
**Fix**: Add `maxStreamPayload = 100MB` check.

## Task Result Truncation (cmd/agent/main.go:1219-1228)

**Current**: Full stdout/stderr marshaled.
**Fix**: Truncate at 10MB with `... [truncated]` marker.

## Commits

1. `feat: add body limit middleware`
2. `fix: audit log uses json.Marshal`
3. `fix: redaction parses JSON key-value pairs`
4. `fix: docker stream and task result size limits`

## Non-Regression

| Check | Method |
|-------|--------|
| Normal API not rejected | Standard CRUD → 200 |
| Large body → 413 | 20MB body → 413 |
| Audit detail valid JSON | GET /api/audit-logs → parse detail |
| PASSWORD_CHANGED preserved | Audit log action name intact |
| Normal logs not truncated | Instance logs readable |
