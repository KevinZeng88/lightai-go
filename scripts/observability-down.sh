#!/bin/bash
# LightAI Go - Stop Observability Stack

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$PROJECT_DIR/run"

echo "=== LightAI Observability Stack - Shutdown ==="

if [ -f "$RUN_DIR/grafana.pid" ]; then
  PID=$(cat "$RUN_DIR/grafana.pid")
  if kill -0 "$PID" 2>/dev/null; then
    echo "Stopping Grafana (PID $PID)..."
    kill "$PID" 2>/dev/null || true
    sleep 2
    kill -9 "$PID" 2>/dev/null || true
    echo "  Grafana stopped"
  fi
  rm -f "$RUN_DIR/grafana.pid"
else
  echo "Grafana not running (no PID file)"
fi

if [ -f "$RUN_DIR/prometheus.pid" ]; then
  PID=$(cat "$RUN_DIR/prometheus.pid")
  if kill -0 "$PID" 2>/dev/null; then
    echo "Stopping Prometheus (PID $PID)..."
    kill "$PID" 2>/dev/null || true
    sleep 2
    kill -9 "$PID" 2>/dev/null || true
    echo "  Prometheus stopped"
  fi
  rm -f "$RUN_DIR/prometheus.pid"
else
  echo "Prometheus not running (no PID file)"
fi

echo "Done."
