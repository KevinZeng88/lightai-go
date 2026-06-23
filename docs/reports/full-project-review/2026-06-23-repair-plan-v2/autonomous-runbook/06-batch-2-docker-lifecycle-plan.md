# Batch 2: Docker Lifecycle / Cleanup / Concurrency — Detailed Plan

---

## Goal
Fix container cleanup, stop/remove semantics, race conditions, task dedup.

## DockerClient Interface (docker_client.go:13-30)

**Current**: No ContainerRemove method.
**Fix**: Add `ContainerRemove(ctx, containerID, force) error`

### Implementations to Update
| File | Change |
|------|--------|
| docker_client.go | Add ContainerRemove to interface |
| docker_real.go | Implement via r.cli.ContainerRemove |
| docker_fake.go | Implement in fake |

## Cleanup in Start() (docker.go:85-194)

**Current failure paths after ContainerCreate**:
- Line 120: ContainerStart fails → diagnose, return error (NO cleanup)
- Line 147-162: Post-start inspect not running → diagnose, return error (NO cleanup)
- Line 180-194: Health check fails → diagnose, return error (NO cleanup)

**Fix**: Add defer after successful ContainerCreate:
```go
containerID, err := d.client.ContainerCreate(ctx, opts)
if err != nil { return err }
defer func() {
    if retErr != nil {
        // Capture logs before remove
        stdout, stderr, _ := d.client.ContainerLogs(ctx, containerID, LogFetchOptions{Tail: 100})
        _ = stdout; _ = stderr // send to server via task result
        d.client.ContainerRemove(ctx, containerID, true)
    }
}()
```

## Stop/Remove (docker.go:230-276)

**Current**: Only ContainerStop, no ContainerRemove.
**Fix**: After ContainerStop, call ContainerRemove.

## Race Conditions (cmd/agent/main.go)

### lastStderrBytes (line 1121, 1197-1199)
**Current**: Plain `map[string]int`, concurrent R/W → panic.
**Fix**: Add `sync.Mutex`:
```go
var logsTaskState struct {
    mu             sync.Mutex
    lastStderrBytes map[string]int
}
```

### reconcileState (line 1330-1337, 1370-1385)
**Current**: `unloggedCount` int, concurrent R/W.
**Fix**: Use `atomic.Int32`.

### Task Dedup (line 706-715)
**Current**: No dedup check.
**Fix**: Add in-flight task map:
```go
var inFlightTasks struct {
    mu    sync.Mutex
    tasks map[string]bool
}
```

## Commits

1. `feat: add ContainerRemove to DockerClient interface`
2. `feat: cleanup container on Start() failure`
3. `feat: remove container on Stop()`
4. `fix: protect lastStderrBytes and reconcileState with sync`
5. `feat: add task dedup`

## Non-Regression

| Check | Method |
|-------|--------|
| Normal start succeeds | Deploy → instance running |
| Stop removes container | Stop → docker ps -a no container |
| Restart no conflict | Stop → Start → no name collision |
| Failed start cleanup | Failure → container removed |
| Race detection | go test -race ./internal/agent/... ./cmd/agent/... |
