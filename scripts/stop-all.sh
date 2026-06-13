#!/bin/sh
# LightAI Go - Stop All Services
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

echo "=== LightAI Go - 停止所有服务 ==="
echo ""

# Stop in reverse order: observability -> agent -> server.

echo "[1/3] 停止 Observability..."
if [ -x scripts/stop-observability.sh ]; then
  sh scripts/stop-observability.sh
else
  # Inline fallback.
  for svc in Grafana Prometheus; do
    pf=""
    case "$svc" in
      Grafana) pf="run/grafana.pid" ;;
      Prometheus) pf="run/prometheus.pid" ;;
    esac
    if [ -f "$pf" ]; then
      PID=$(cat "$pf")
      if kill -0 "$PID" 2>/dev/null; then
        echo "  停止 $svc (PID $PID)..."
        kill "$PID" 2>/dev/null || true
        sleep 2
        kill -0 "$PID" 2>/dev/null && kill -9 "$PID" 2>/dev/null || true
      fi
      rm -f "$pf"
    fi
  done
fi

echo ""
echo "[2/3] 停止 Agent..."
if [ -x scripts/stop-agent.sh ]; then
  sh scripts/stop-agent.sh
fi

echo ""
echo "[3/3] 停止 Server..."
if [ -x scripts/stop-server.sh ]; then
  sh scripts/stop-server.sh
fi

echo ""
echo "所有服务已停止。"
