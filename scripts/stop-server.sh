#!/bin/sh
# LightAI Go - Stop Server
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RELEASE_ROOT"

if [ ! -f run/server.pid ]; then
  echo "Server not running (no PID file)."
  exit 0
fi

PID=$(cat run/server.pid)
if kill -0 "$PID" 2>/dev/null; then
  echo "Stopping server (PID $PID)..."
  kill "$PID"
  sleep 2
  kill -0 "$PID" 2>/dev/null && kill -9 "$PID" 2>/dev/null || true
  echo "Server stopped."
else
  echo "Server not running (stale PID file)."
fi
rm -f run/server.pid
