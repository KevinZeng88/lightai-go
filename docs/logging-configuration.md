# Logging Configuration

LightAI Go uses structured logging via Go's standard `log/slog` library with custom handlers for human-readable output.

## Log Format

### Text Format (Default)

Human-readable format suitable for local development:

```
2026-06-23 13:57:33.391 INFO  message key=value key=value
```

- Timestamp: `2006-01-02 15:04:05.000`
- Level: Fixed 5-character width (`DEBUG`, `INFO `, `WARN `, `ERROR`)
- Message: Event description
- Attributes: Space-separated `key=value` pairs

### JSON Format

Structured JSON format suitable for production and log aggregation tools (ELK, Loki, Datadog):

```json
{"time":"2026-06-23T13:57:33.391Z","level":"INFO","msg":"message","key":"value"}
```

## Configuration

### Server Configuration

In the server config file or environment:

```yaml
log:
  level: info          # debug, info, warn, error
  format: text         # text or json
  dir: logs            # log directory
  file: lightai-server.log
  stdout: true         # also write to stdout
  file_enabled: true   # enable file logging
  append: true         # append to existing file
  max_size_mb: 100     # rotate when file exceeds this size
  max_files: 5         # keep this many rotated files
  retention_days: 30   # delete logs older than this
```

### Agent Configuration

```yaml
log:
  level: info
  format: text
  dir: logs
  file: lightai-agent.log
  stdout: true
  file_enabled: true
  append: true
  max_size_mb: 50
  max_files: 3
  retention_days: 7
```

## Log Levels

| Level | Usage |
|-------|-------|
| `DEBUG` | Detailed diagnostic information. Enable for troubleshooting. |
| `INFO` | Normal operational events. Default level. |
| `WARN` | Unexpected but recoverable conditions. |
| `ERROR` | Failures that require attention. |

## Output Destinations

LightAI Go supports dual-write logging:

- **File**: Writes to the configured log file with rotation
- **Stdout**: Writes to standard output (useful for containerized environments)
- **Fallback**: If neither is configured, writes to stderr

## Log Rotation

Log files are rotated automatically when they exceed `max_size_mb`:

- Current file: `lightai-server.log`
- Rotated files: `lightai-server.log.1`, `lightai-server.log.2`, etc.
- Maximum rotated files: `max_files`
- Old files are deleted at startup if older than `retention_days`

## Special Log Behaviors

### High-Frequency Endpoint Suppression

The following endpoints log at `DEBUG` level for successful (2xx) requests to reduce noise:

- `/api/v1/agent/heartbeat`
- `/api/v1/agent/resources/report`
- `/api/v1/agent/tasks/`
- `/metrics`
- GET requests to `/api/v1/model-instances`, `/api/v1/nodes`, `/api/v1/gpus`

### Static Asset Suppression

Static web assets (`/assets/*`, `/favicon.*`, `/manifest.json`, `/robots.txt`) log at `DEBUG` for 2xx responses. Errors (4xx/5xx) are still logged at their normal level.

### Slow Operation Detection

Requests exceeding the slow threshold are logged as `WARN` with `slow_operation`:

| Route | Threshold |
|-------|-----------|
| `/api/v1/node-run-plans/{id}/logs` | 3000ms |
| All other API routes | 1000ms |

### Periodic Summary Logs

High-frequency operations use periodic summaries to reduce log volume:

- **Heartbeat**: Summary every 60 seconds
- **Task Poll**: Summary every 60 seconds
- **GPU Metrics**: Summary every 60 seconds

### Container Reconciliation

The agent's container reconciliation logs:

- `INFO` when container state changes
- `DEBUG` when state is unchanged
- Summary `INFO` every ~5 minutes if unchanged

### Panic Recovery

Server-side panics are caught by a recovery middleware that:

- Logs the panic message and stack trace at `ERROR` level
- Returns HTTP 500 to the client
- Prevents the server process from crashing

## Request Context

Each API request is assigned a `request_id` (UUID) that is:

- Returned in the `X-Request-ID` response header
- Included in all log entries for that request
- Propagated to downstream operations

## Production Recommendations

1. **Use JSON format** for log aggregation:
   ```yaml
   log:
     format: json
   ```

2. **Enable file logging with rotation**:
   ```yaml
   log:
     file_enabled: true
     max_size_mb: 100
     max_files: 10
     retention_days: 30
   ```

3. **Set appropriate log level**:
   - Production: `info`
   - Troubleshooting: `debug`

4. **Monitor for `ERROR` and `WARN` entries** in log aggregation tools

5. **Set up alerts** for:
   - `handler.panic.recovered` — indicates a bug
   - `slow_operation` — indicates performance issues
   - `operation_timeout` — indicates timeouts

## Local Development Recommendations

1. **Use text format** for readability:
   ```yaml
   log:
     format: text
   ```

2. **Enable stdout** for immediate feedback:
   ```yaml
   log:
     stdout: true
   ```

3. **Use `debug` level** when developing new features:
   ```yaml
   log:
     level: debug
   ```

## Log File Locations

Default log file locations:

- Server: `logs/lightai-server.log`
- Agent: `logs/lightai-agent.log`
- Stdout logs: `logs/server-stdout.log`, `logs/agent-stdout.log`

## Structured Log Fields

### API Request Logs

All API request completions include:

| Field | Description |
|-------|-------------|
| `request_id` | Unique request identifier |
| `method` | HTTP method (GET, POST, etc.) |
| `path` | Request path (parameters stripped) |
| `status` | HTTP status code |
| `duration_ms` | Request duration in milliseconds |
| `client_ip` | Client IP address |
| `user_agent` | Client user agent (truncated to 200 chars) |

### Operation Logs

Operation lifecycle events include:

| Field | Description |
|-------|-------------|
| `operation` | Operation name |
| `stage` | Lifecycle stage (started, completed, failed) |
| `duration_ms` | Operation duration |
| `request_id` | Parent request ID |
| `operation_id` | Unique operation identifier |

### Task Logs

Agent task execution includes:

| Field | Description |
|-------|-------------|
| `task_id` | Task identifier |
| `task_type` | Task type (model_instance_start, model_instance_logs, etc.) |
| `instance_id` | Model instance ID |
| `container_id` | Docker container ID (or `container_id_status=not_allocated`) |
| `deployment_id` | Deployment ID |
| `node_id` | Node ID |
