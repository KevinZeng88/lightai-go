#!/usr/bin/env bash
# start-all.sh — LightAI Go unified launcher
# Start Server, optional bundled observability, and Agent in correct order.
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
  SERVER_BIN="go run ./cmd/server"
else
  MODE="release"
  if [ -x "$PROJECT_DIR/lightai-server" ]; then
    SERVER_BIN="$PROJECT_DIR/lightai-server"
  elif [ -x "$PROJECT_DIR/bin/lightai-server" ]; then
    SERVER_BIN="$PROJECT_DIR/bin/lightai-server"
  else
    echo "ERROR: Cannot find lightai-server binary. Run from source tree or release directory." >&2
    exit 1
  fi
fi

log()  { echo "[start-all] $*"; }
dryn() { if $DRY_RUN; then echo "[DRY-RUN] $*"; fi }

# ── Check prerequisites ──
check_script() {
  local s="$SCRIPT_DIR/$1"
  if [ ! -f "$s" ]; then
    echo "ERROR: Required script not found: $s" >&2
    exit 1
  fi
}

check_script start-server.sh
check_script start-agent.sh

log "Mode: $MODE"
log "Observability: $($NO_OBSERVABILITY && echo 'skipped' || echo 'enabled')"

# ── Start Server ──
log "Starting server..."
dryn "$SCRIPT_DIR/start-server.sh"
if ! $DRY_RUN; then
  bash "$SCRIPT_DIR/start-server.sh"
fi

# ── Start observability (if enabled) ──
if ! $NO_OBSERVABILITY; then
  if [ -f "$SCRIPT_DIR/observability-up.sh" ]; then
    log "Starting bundled observability..."
    dryn "$SCRIPT_DIR/observability-up.sh"
    if ! $DRY_RUN; then
      bash "$SCRIPT_DIR/observability-up.sh" || log "WARNING: observability startup may have issues — continuing"
    fi
  elif [ -f "$SCRIPT_DIR/start-observability.sh" ]; then
    log "Starting bundled observability..."
    dryn "$SCRIPT_DIR/start-observability.sh"
    if ! $DRY_RUN; then
      bash "$SCRIPT_DIR/start-observability.sh" || log "WARNING: observability startup may have issues — continuing"
    fi
  else
    log "WARNING: observability scripts not found — skipping"
  fi
fi

# ── Start Agent ──
log "Starting agent..."
dryn "$SCRIPT_DIR/start-agent.sh"
if ! $DRY_RUN; then
  bash "$SCRIPT_DIR/start-agent.sh"
fi

# ── Health checks ──
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
  SERVER_PORT=${LIGHTAI_SERVER_PORT:-18080}
  AGENT_METRICS_PORT=${LIGHTAI_AGENT_METRICS_PORT:-19091}

  health_check "http://127.0.0.1:${SERVER_PORT}/healthz" "Server"
  health_check "http://127.0.0.1:${AGENT_METRICS_PORT}/healthz" "Agent"

  if ! $NO_OBSERVABILITY; then
    PROM_PORT=${LIGHTAI_PROMETHEUS_PORT:-19090}
    GRAFANA_PORT=${LIGHTAI_GRAFANA_PORT:-13000}
    curl -sf -o /dev/null "http://127.0.0.1:${PROM_PORT}/-/healthy" 2>/dev/null && log "  Prometheus: OK" || log "  Prometheus: not responding"
    curl -sf -o /dev/null "http://127.0.0.1:${GRAFANA_PORT}/api/health" 2>/dev/null && log "  Grafana: OK" || log "  Grafana: not responding"
  fi
fi

log "Done."
