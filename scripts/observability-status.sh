#!/bin/bash
# LightAI Go - Observability Stack Status Check (bundled mode)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$PROJECT_DIR/run"

PROMETHEUS_BIN="${PROMETHEUS_BIN:-prometheus}"
GRAFANA_BIN="${GRAFANA_BIN:-grafana-server}"
PROMETHEUS_LISTEN="${PROMETHEUS_LISTEN:-127.0.0.1:19090}"
GRAFANA_LISTEN="${GRAFANA_LISTEN:-127.0.0.1:13000}"

echo "=== LightAI Observability Stack Status ==="
echo ""

check_binary() {
  local name="$1" path="$2"
  if command -v "$path" >/dev/null 2>&1; then
    echo "  $name binary: $(command -v "$path")"
    return 0
  elif [ -x "$path" ]; then
    echo "  $name binary: $path"
    return 0
  else
    echo "  $name binary: MISSING ($path)"
    return 1
  fi
}

check_process() {
  local name="$1" pid_file="$2" health_url="$3"
  if [ -f "$pid_file" ]; then
    PID=$(cat "$pid_file")
    if kill -0 "$PID" 2>/dev/null; then
      echo "  $name: RUNNING (PID $PID)"
      if [ -n "$health_url" ]; then
        if curl -sf "$health_url" > /dev/null 2>&1; then
          echo "    Health: OK ($health_url)"
        else
          echo "    Health: NOT RESPONDING"
        fi
      fi
      return 0
    else
      echo "  $name: STOPPED (stale PID file)"
      return 1
    fi
  else
    echo "  $name: STOPPED"
    return 1
  fi
}

echo "--- Binary Detection ---"
PROM_BIN_OK=false
GRAF_BIN_OK=false
check_binary "prometheus" "$PROMETHEUS_BIN" && PROM_BIN_OK=true
check_binary "grafana-server" "$GRAFANA_BIN" && GRAF_BIN_OK=true
echo ""

if ! $PROM_BIN_OK; then
  echo "  DIAGNOSIS: Prometheus binary not found."
  echo "  Install prometheus or set PROMETHEUS_BIN env var."
  echo "  Or switch to observability.mode=external or disabled."
  echo ""
fi
if ! $GRAF_BIN_OK; then
  echo "  DIAGNOSIS: Grafana binary not found."
  echo "  Install grafana or set GRAFANA_BIN env var."
  echo "  Or switch to observability.mode=external or disabled."
  echo ""
fi

echo "--- Process Status ---"
check_process "Prometheus" "$RUN_DIR/prometheus.pid" "http://$PROMETHEUS_LISTEN/-/healthy"
echo ""
check_process "Grafana" "$RUN_DIR/grafana.pid" "http://$GRAFANA_LISTEN/api/health"
echo ""

echo "--- LightAI Server ---"
if curl -sf http://127.0.0.1:18080/healthz > /dev/null 2>&1; then
  echo "  Server: RUNNING (http://127.0.0.1:18080)"
else
  echo "  Server: NOT REACHABLE"
fi

echo ""
echo "--- Metrics ---"
if curl -sf http://127.0.0.1:18080/metrics/targets > /dev/null 2>&1; then
  TARGET_COUNT=$(curl -sf http://127.0.0.1:18080/metrics/targets | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
  echo "  /metrics/targets: $TARGET_COUNT targets"
else
  echo "  /metrics/targets: NOT REACHABLE"
fi

if curl -sf http://127.0.0.1:19091/metrics > /dev/null 2>&1; then
  echo "  Agent /metrics: OK (http://127.0.0.1:19091)"
else
  echo "  Agent /metrics: NOT REACHABLE"
fi

echo ""
echo "Quick links:"
echo "  Server:    http://127.0.0.1:18080"
echo "  Prometheus: http://$PROMETHEUS_LISTEN"
echo "  Grafana:   http://$GRAFANA_LISTEN"
