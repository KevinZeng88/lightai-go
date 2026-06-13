#!/bin/bash
# LightAI Go - Observability Stack Status Check

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$PROJECT_DIR/run"

echo "=== LightAI Observability Stack Status ==="
echo ""

check_process() {
  local name="$1" pid_file="$2" listen_addr="$3" health_url="$4"
  if [ -f "$pid_file" ]; then
    PID=$(cat "$pid_file")
    if kill -0 "$PID" 2>/dev/null; then
      echo "  $name: RUNNING (PID $PID)"
      if [ -n "$health_url" ]; then
        if curl -sf "$health_url" > /dev/null 2>&1; then
          echo "    Health: OK ($health_url)"
        else
          echo "    Health: NOT RESPONDING ($health_url)"
        fi
      fi
      if [ -n "$listen_addr" ]; then
        echo "    Listen: $listen_addr"
      fi
      return 0
    else
      echo "  $name: STALE PID (file exists but process not running)"
      return 1
    fi
  else
    echo "  $name: NOT RUNNING"
    return 1
  fi
}

check_process "Prometheus" "$RUN_DIR/prometheus.pid" "" "http://127.0.0.1:19090/-/healthy"
echo ""
check_process "Grafana" "$RUN_DIR/grafana.pid" "" "http://127.0.0.1:13000/api/health"
echo ""

# Check LightAI Server
echo "--- LightAI Server ---"
if curl -sf http://127.0.0.1:18080/healthz > /dev/null 2>&1; then
  echo "  Server: RUNNING (http://127.0.0.1:18080)"
else
  echo "  Server: NOT REACHABLE"
fi

# Check metrics endpoints
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
echo "  Prometheus: http://127.0.0.1:19090"
echo "  Grafana:   http://127.0.0.1:13000"
