#!/bin/sh
# LightAI Go - Stop Agent
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RELEASE_ROOT"

if [ ! -f run/agent.pid ]; then
  echo "Agent not running (no PID file)."
  exit 0
fi

PID=$(cat run/agent.pid)
if kill -0 "$PID" 2>/dev/null; then
  echo "Stopping agent (PID $PID)..."
  kill "$PID"
  sleep 2
  kill -0 "$PID" 2>/dev/null && kill -9 "$PID" 2>/dev/null || true
  echo "Agent stopped."
else
  echo "Agent not running (stale PID file)."
fi
rm -f run/agent.pid
