# Batch 1B: AgentClient / SSRF — Detailed Plan

> Absorbs: `08-batch-1a-1b-execution-plan.md` §Batch 1B

---

## Goal
Replace 6 bare http.Get/Post calls with AgentClient. Add SSRF protection.

## HTTP Call Replacement Matrix (verified by code)

| # | File:Line | Function | Current | Replace With |
|---|-----------|----------|---------|-------------|
| 1 | agent_proxy_handlers.go:52 | HandleProxyNodeFiles | `http.Get(url)` | `agentClient.GetJSON(ctx, ip, port, "/files", q)` |
| 2 | agent_proxy_handlers.go:110 | HandleProxyNodeModelScan | `http.Post(url, body)` | `agentClient.PostJSON(ctx, ip, port, "/model-paths/scan", body)` |
| 3 | agent_handlers.go:612 | HandleGetNodeDockerImages | `http.Get(url)` (no URL encode!) | `agentClient.GetJSON(ctx, addr, port, "/docker-images", q)` |
| 4 | agent_handlers.go:650 | HandleGetNodeDockerImageInspect | `http.Get(url)` | `agentClient.GetJSON(ctx, addr, port, "/docker-image-inspect", q)` |
| 5 | runtime_handlers.go:380 | HandleRequestNodeBackendRuntimeCheck | `http.Get(url)` | `agentClient.GetJSON(ctx, addr, port, "/docker-images", q)` |
| 6 | runtime_handlers.go:455 | HandleRequestNodeBackendRuntimeCheck | `http.Get(url)` | `agentClient.GetJSON(ctx, addr, port, "/docker-image-inspect", q)` |

**Not replaced**: deployment_lifecycle_handlers.go:2225 (already has timeout, targets instance), observability_handler.go:36 (probes Prometheus/Grafana)

## AgentClient Design

New file: `internal/server/agentclient/client.go`

```go
package agentclient

type Client struct {
    httpClient *http.Client
    agentToken string
}

func New(agentToken string, timeout time.Duration) *Client
func (c *Client) GetJSON(ctx context.Context, addr string, port int, path string, params url.Values) ([]byte, error)
func (c *Client) PostJSON(ctx context.Context, addr string, port int, path string, body io.Reader) ([]byte, error)
func ValidateAgentAddress(addr string) error
```

### Implementation Rules
- Use `net.JoinHostPort(addr, fmt.Sprintf("%d", port))` for URL construction
- Use `url.Values.Encode()` for all query params
- Response body limit: `io.LimitReader(resp.Body, MaxResponseBytes)` then check size
- Agent token: `req.Header.Set("Authorization", "Bearer "+c.agentToken)`
- Timeout: configurable per client (30s default, 120s for file/scan)

### Address Validation
```go
var deniedCIDRs = []*net.IPNet{
    parseCIDR("169.254.0.0/16"), // metadata/link-local
    parseCIDR("0.0.0.0/32"),     // unspecified
    parseCIDR("::/128"),          // unspecified v6
    parseCIDR("224.0.0.0/4"),    // multicast
    parseCIDR("ff00::/8"),       // multicast v6
}
```
- Allow: localhost, private IPs, hostnames
- Deny: metadata, link-local, unspecified, multicast

## Commits

1. `feat: add agentclient package with SSRF protection`
2. `feat: replace bare http.Get/Post with AgentClient`
3. `test: add agentclient SSRF and integration tests`

## Tests

- `internal/server/agentclient/client_test.go` — 13 tests
  - ValidateAgentAddress: localhost allowed, private allowed, metadata denied, link-local denied, unspecified denied, multicast denied, hostname allowed
  - GetJSON: success, timeout, response limit, denied address, URL encoding
  - PostJSON: success

## Non-Regression

| Check | Method |
|-------|--------|
| File browse proxy | GET /nodes/{id}/files → 200 |
| Model scan proxy | POST /nodes/{id}/model-paths/scan → 200 |
| Docker images proxy | GET /nodes/{id}/docker-images → 200 |
| Docker inspect proxy | GET /nodes/{id}/docker-image-inspect → 200 |
| localhost agent reachable | Same-machine agent → 200 |
| URL encoding correct | Special chars in params → properly encoded |
