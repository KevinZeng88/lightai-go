#!/bin/sh
# LightAI Go - Stop Observability (Prometheus + Grafana)
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

echo "=== LightAI Observability - 停止 ==="

stop_proc() {
  local name="$1" pidfile="$2"
  if [ -f "$pidfile" ]; then
    PID=$(cat "$pidfile")
    if kill -0 "$PID" 2>/dev/null; then
      echo "停止 $name (PID $PID)..."
      kill "$PID" 2>/dev/null || true
      sleep 2
      kill -0 "$PID" 2>/dev/null && kill -9 "$PID" 2>/dev/null || true
      echo "  $name 已停止。"
    else
      echo "  $name 未运行 (残留 PID)。"
    fi
    rm -f "$pidfile"
  else
    echo "  $name 未运行。"
  fi
}

stop_proc "Grafana" run/grafana.pid
stop_proc "Prometheus" run/prometheus.pid
echo "完成。"
