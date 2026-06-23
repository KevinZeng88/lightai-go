# Batch 2 Closeout: Docker Lifecycle / Cleanup / Concurrency

> Date: 2026-06-23
> Status: PASS

---

## Changes Made

### Files Modified
| File | Changes |
|------|---------|
| internal/agent/runtime/docker_client.go | Added ContainerRemove to interface |
| internal/agent/runtime/docker_real.go | Implemented ContainerRemove via Docker SDK |
| internal/agent/runtime/docker_fake.go | Implemented ContainerRemove (delete from maps) |
| internal/agent/runtime/docker.go | Cleanup on Start() failure, Remove on Stop() |
| internal/agent/runtime/docker_test.go | Updated Stop test to verify container removal |
| cmd/agent/main.go | Fixed lastStderrBytes race (mutex), reconcileState race (atomic) |

### Commits
| SHA | Message |
|-----|---------|
| 375baee | fix(agent): Docker lifecycle cleanup and race condition fixes |

---

## After Verification

- **go build**: PASS
- **go test ./internal/agent/...**: PASS (all packages)
- **go test -race ./internal/agent/runtime/...**: PASS

---

## Cleanup Semantics Implemented

| Scenario | Behavior |
|----------|----------|
| Start fails (create ok, start fails) | Logs captured → container removed |
| Start fails (health check fails) | Logs captured → container removed |
| Stop | Logs captured → container stopped → container removed |

---

## Race Fixes

| Location | Issue | Fix |
|----------|-------|-----|
| logsTaskState.lastStderrBytes | Concurrent map R/W | sync.Mutex |
| reconcileState.unloggedCount | Concurrent int R/W | atomic.Int32 |

---

## Stop Conditions

None triggered.
