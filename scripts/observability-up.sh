#!/bin/bash
# LightAI Go - Start Observability Stack (bundled mode)
# Starts Prometheus + Grafana as managed subprocesses.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$PROJECT_DIR/run"

PROMETHEUS_BIN="${PROMETHEUS_BIN:-prometheus}"
GRAFANA_BIN="${GRAFANA_BIN:-grafana-server}"

PROMETHEUS_CONFIG="$PROJECT_DIR/deploy/observability/prometheus/prometheus.yml"
PROMETHEUS_DATA="$PROJECT_DIR/data/prometheus"
PROMETHEUS_LOG="$PROJECT_DIR/logs/prometheus.log"
PROMETHEUS_LISTEN="${PROMETHEUS_LISTEN:-127.0.0.1:19090}"

GRAFANA_HOME="${GRAFANA_HOME:-/usr/share/grafana}"
GRAFANA_DATA="$PROJECT_DIR/data/grafana"
GRAFANA_LOG="$PROJECT_DIR/logs/grafana.log"
GRAFANA_LISTEN="${GRAFANA_LISTEN:-127.0.0.1:13000}"
GRAFANA_PROVISIONING="$PROJECT_DIR/deploy/observability/grafana/provisioning"
GRAFANA_PROV_TARGET="/var/lib/grafana"

mkdir -p "$RUN_DIR" "$PROMETHEUS_DATA" "$GRAFANA_DATA" "$PROJECT_DIR/logs"

echo "=== LightAI Observability Stack ==="
echo ""

# --- Prometheus ---
echo "[1/2] Starting Prometheus..."
if [ -f "$RUN_DIR/prometheus.pid" ]; then
  if kill -0 "$(cat "$RUN_DIR/prometheus.pid")" 2>/dev/null; then
    echo "  Prometheus already running (PID $(cat "$RUN_DIR/prometheus.pid"))"
  else
    rm -f "$RUN_DIR/prometheus.pid"
    echo "  Stale PID file removed"
  fi
fi

if [ ! -f "$RUN_DIR/prometheus.pid" ]; then
  if ! command -v "$PROMETHEUS_BIN" &>/dev/null; then
    echo "  ERROR: prometheus binary not found at '$PROMETHEUS_BIN'"
    echo "  Set PROMETHEUS_BIN env var or install prometheus."
    exit 1
  fi
  "$PROMETHEUS_BIN" \
    --config.file="$PROMETHEUS_CONFIG" \
    --storage.tsdb.path="$PROMETHEUS_DATA" \
    --web.listen-address="$PROMETHEUS_LISTEN" \
    --storage.tsdb.retention.time=15d \
    >> "$PROMETHEUS_LOG" 2>&1 &
  PROM_PID=$!
  echo "$PROM_PID" > "$RUN_DIR/prometheus.pid"
  echo "  Prometheus started (PID $PROM_PID, listen $PROMETHEUS_LISTEN)"
else
  PROM_PID=$(cat "$RUN_DIR/prometheus.pid")
  echo "  Prometheus PID $PROM_PID"
fi

# --- Grafana ---
echo "[2/2] Starting Grafana..."
if [ -f "$RUN_DIR/grafana.pid" ]; then
  if kill -0 "$(cat "$RUN_DIR/grafana.pid")" 2>/dev/null; then
    echo "  Grafana already running (PID $(cat "$RUN_DIR/grafana.pid"))"
  else
    rm -f "$RUN_DIR/grafana.pid"
    echo "  Stale PID file removed"
  fi
fi

if [ ! -f "$RUN_DIR/grafana.pid" ]; then
  if ! command -v "$GRAFANA_BIN" &>/dev/null; then
    echo "  WARNING: grafana-server binary not found at '$GRAFANA_BIN'"
    echo "  Set GRAFANA_BIN env var or install grafana."
    echo "  Grafana will NOT be started."
  else
    # Grafana provisioning: copy to temporary location and set cfg paths.
    GF_PATHS_PROVISIONING="$PROJECT_DIR/deploy/observability/grafana/provisioning" \
    GF_PATHS_DATA="$GRAFANA_DATA" \
    GF_SERVER_HTTP_ADDR=$(echo "$GRAFANA_LISTEN" | cut -d: -f1) \
    GF_SERVER_HTTP_PORT=$(echo "$GRAFANA_LISTEN" | cut -d: -f2) \
    GF_SECURITY_ADMIN_USER="${LIGHTAI_GRAFANA_ADMIN_USER:-admin}" \
    GF_SECURITY_ADMIN_PASSWORD="${LIGHTAI_GRAFANA_ADMIN_PASSWORD:-lightai}" \
    "$GRAFANA_BIN" \
      --homepath="$GRAFANA_HOME" \
      >> "$GRAFANA_LOG" 2>&1 &
    GRAF_PID=$!
    echo "$GRAF_PID" > "$RUN_DIR/grafana.pid"
    echo "  Grafana started (PID $GRAF_PID, listen $GRAFANA_LISTEN)"
    if [ "${LIGHTAI_GRAFANA_ADMIN_PASSWORD:-lightai}" = "lightai" ]; then
      echo "  NOTE: Using default dev password 'lightai'. Set LIGHTAI_GRAFANA_ADMIN_PASSWORD for production."
    fi
  fi
else
  GRAF_PID=$(cat "$RUN_DIR/grafana.pid")
  echo "  Grafana PID $GRAF_PID"
fi

echo ""
echo "Observability stack started."
echo "  Prometheus: http://$PROMETHEUS_LISTEN"
echo "  Grafana:    http://$GRAFANA_LISTEN (admin / lightai)"
echo ""
echo "Dashboards:"
echo "  LightAI Overview:     http://$GRAFANA_LISTEN/d/lightai-overview"
echo "  LightAI GPU Resources: http://$GRAFANA_LISTEN/d/lightai-gpu-resources"
echo "  LightAI Agent Health:  http://$GRAFANA_LISTEN/d/lightai-agent-health"
