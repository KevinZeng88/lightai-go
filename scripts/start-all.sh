#!/usr/bin/env bash
# start-all.sh — LightAI Go unified launcher
# Start Server, optional bundled observability, and Agent in correct order.
# Idempotent: detects already-running services and skips them.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

DRY_RUN=false
NO_OBSERVABILITY=false
WAIT=false

usage() {
  echo "Usage: $0 [--dry-run] [--no-observability] [--wait]"
  echo ""
  echo "  --dry-run          Print intended actions without starting anything."
  echo "  --no-observability Skip bundled Prometheus/Grafana startup."
  echo "  --wait             Health-check each service after startup."
  exit 1
}

for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=true ;;
    --no-observability) NO_OBSERVABILITY=true ;;
    --wait) WAIT=true ;;
    --help|-h) usage ;;
    *) echo "Unknown option: $arg"; usage ;;
  esac
done

# Detect source tree vs release directory.
if [ -f "$PROJECT_DIR/cmd/server/main.go" ]; then
  MODE="source"
else
  MODE="release"
fi

log()  { echo "[start-all] $*"; }
dryn() { if $DRY_RUN; then echo "[DRY-RUN] $*"; fi }

# Health check helper — returns 0 if endpoint responds.
is_healthy() {
  local url="$1"
  curl -sf -o /dev/null "$url" 2>/dev/null
}

SERVER_PORT=${LIGHTAI_SERVER_PORT:-18080}
AGENT_METRICS_PORT=${LIGHTAI_AGENT_METRICS_PORT:-19091}
PROM_PORT=${LIGHTAI_PROMETHEUS_PORT:-19090}
GRAFANA_PORT=${LIGHTAI_GRAFANA_PORT:-13000}

log "Mode: $MODE"
log "Observability: $($NO_OBSERVABILITY && echo 'skipped' || echo 'enabled')"

# ── Start Server (idempotent) ──
if is_healthy "http://127.0.0.1:${SERVER_PORT}/healthz"; then
  log "Server: already running (health check OK on port $SERVER_PORT) — skipping"
elif $DRY_RUN; then
  dryn "$SCRIPT_DIR/start-server.sh"
else
  log "Starting server..."
  bash "$SCRIPT_DIR/start-server.sh" || {
    log "ERROR: Server failed to start"
    exit 1
  }
fi

# ── Start observability (idempotent) ──
if ! $NO_OBSERVABILITY; then
  OBS_HEALTHY=false
  if is_healthy "http://127.0.0.1:${PROM_PORT}/-/healthy" 2>/dev/null; then
    OBS_HEALTHY=true
  fi
  if $OBS_HEALTHY; then
    log "Observability: already running (Prometheus health OK on port $PROM_PORT) — skipping"
  elif $DRY_RUN; then
    if [ -f "$SCRIPT_DIR/observability-up.sh" ]; then
      dryn "$SCRIPT_DIR/observability-up.sh"
    fi
  else
    log "Starting bundled observability..."
    if [ -f "$SCRIPT_DIR/observability-up.sh" ]; then
      bash "$SCRIPT_DIR/observability-up.sh" || log "WARNING: observability startup may have issues — continuing"
    elif [ -f "$SCRIPT_DIR/start-observability.sh" ]; then
      bash "$SCRIPT_DIR/start-observability.sh" || log "WARNING: observability startup may have issues — continuing"
    else
      log "WARNING: observability scripts not found — skipping"
    fi
  fi
fi

# ── Start Agent (idempotent) ──
if is_healthy "http://127.0.0.1:${AGENT_METRICS_PORT}/healthz"; then
  log "Agent: already running (health check OK on port $AGENT_METRICS_PORT) — skipping"
elif $DRY_RUN; then
  dryn "$SCRIPT_DIR/start-agent.sh"
else
  log "Starting agent..."
  bash "$SCRIPT_DIR/start-agent.sh" || {
    log "ERROR: Agent failed to start"
    exit 1
  }
fi

# ── Health checks (--wait) ──
health_check() {
  local url="$1"
  local label="$2"
  log "Health check: $label ($url)..."
  for i in $(seq 1 30); do
    if curl -sf -o /dev/null "$url" 2>/dev/null; then
      log "  $label: OK"
      return 0
    fi
    sleep 1
  done
  log "  $label: FAILED (timeout after 30s)"
  return 1
}

if $WAIT && ! $DRY_RUN; then
  health_check "http://127.0.0.1:${SERVER_PORT}/healthz" "Server"
  health_check "http://127.0.0.1:${AGENT_METRICS_PORT}/healthz" "Agent"

  if ! $NO_OBSERVABILITY; then
    curl -sf -o /dev/null "http://127.0.0.1:${PROM_PORT}/-/healthy" 2>/dev/null && log "  Prometheus: OK" || log "  Prometheus: not responding"
    curl -sf -o /dev/null "http://127.0.0.1:${GRAFANA_PORT}/api/health" 2>/dev/null && log "  Grafana: OK" || log "  Grafana: not responding"
  fi
fi

log "Done."
