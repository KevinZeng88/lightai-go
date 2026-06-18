> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Model Runtime — Troubleshooting Guide

## Quick Diagnostics

```bash
# Check server and agent processes
ps aux | grep lightai

# Check agent tasks
sqlite3 run/e2e/e2e-test.db "SELECT id, task_type, status, node_id, claimed_at, finished_at FROM agent_tasks ORDER BY created_at;"

# Check instances
sqlite3 run/e2e/e2e-test.db "SELECT id, actual_state, container_id, last_error FROM model_instances;"

# Check leases
sqlite3 run/e2e/e2e-test.db "SELECT id, status, gpu_id, expires_at FROM gpu_leases;"

# Check server log
tail -100 run/e2e/server.log

# Check agent log
tail -100 run/e2e/agent.log
```

## Common Issues

### Docker permission denied
```
Error: permission denied while trying to connect to the Docker daemon socket
```
**Fix:** Add user to `docker` group: `sudo usermod -aG docker $USER` then log out/in.

### NVIDIA runtime not available
```
Error: could not select device driver "nvidia"
```
**Fix:** Install `nvidia-container-toolkit`, restart Docker.

### --gpus not working
```
Error: unknown flag: --gpus
```
**Fix:** Check `nvidia-container-toolkit` is installed. Verify with `docker run --rm --gpus all nvidia/cuda:13.1.1-base-ubuntu24.04 nvidia-smi`.

### Host/container model path mismatch
```
Error: File not found: /models/model.gguf
```
**Check:** Volume mapping must map host model directory to container path. Verify in dry-run preview: `-v /host/path:/container/path`.

### /v1/models returns empty
- Container may still be loading the model (large models take 30-120s).
- Check `docker logs <container>` for loading progress.
- Wait longer or increase model API polling timeout.

### Logs task pending indefinitely
- Agent may be offline. Check: `ps aux | grep lightai-agent`.
- Task node_id may not match agent heartbeat node_id.
- Check: `sqlite3 ... "SELECT node_id, status FROM agent_tasks WHERE task_type='model_instance_logs';"`
- Ensure dedup logic works: repeated GET logs should return same task_id (202), not create new tasks.

### Logs task claimed but not succeeded
- Agent may have failed silently. Check agent log for errors.
- Docker container may not exist. Verify: `docker ps -a | grep lightai`.
- The agent needs container_id or container_name to fetch logs.

### Container not found
```
Error: No such container: lightai-...
```
- Container may have been removed manually or crashed.
- Stop API should be idempotent and release leases regardless.

### Lease not released
- Stop may have failed. Check instance actual_state and task status.
- Sweep loop (every 30s) will eventually release expired reserved leases.
- Manual cleanup: `sqlite3 ... "UPDATE gpu_leases SET status='released' WHERE instance_id='...'"`.

### Stop already_stopped
This is normal. The stop API is idempotent — calling stop on an already stopped deployment returns `{"status":"already_stopped"}`.

### Port already in use
```
Error: port 8002 is in use
```
- Kill the previous container: `docker rm -f <container>`.
- Or use a different port: `--port 8003`.

### network_mode=host with port mapping
When `network_mode=host`, port mapping (`-p`) is ineffective. The dry-run validator issues a warning. Use either host network OR port mapping, not both.

### Agent offline
- Check if agent process is running: `ps aux | grep lightai-agent`.
- Check heartbeat in agent log: `grep heartbeat agent.log`.
- Server marks nodes offline after 300s without heartbeat (configurable).

### Task timed_out
- Task exceeded `timeout_seconds` (default 300s for start, 60s for stop, 30s for logs).
- Server sweep marks timed-out tasks and moves instance to `unknown`.
- Reserved leases are released. Active leases only released if node is offline.

### SQLite diagnostics
```bash
DB="run/e2e/e2e-test.db"
# All tables
sqlite3 "$DB" ".tables"
# Task timeline
sqlite3 "$DB" "SELECT id, task_type, status, created_at, claimed_at, finished_at FROM agent_tasks ORDER BY created_at;"
# Active instances
sqlite3 "$DB" "SELECT id, actual_state, container_id FROM model_instances WHERE actual_state NOT IN ('stopped','failed');"
# Active leases
sqlite3 "$DB" "SELECT id, status, gpu_id, expires_at FROM gpu_leases WHERE status IN ('reserved','active');"
```

## Log Locations

| Component | Log Path |
|-----------|----------|
| Server | `run/e2e/server.log` |
| Agent | `run/e2e/agent.log` |
| Docker container | `docker logs <container_id>` |
| SQLite DB | `run/e2e/e2e-test.db` |
