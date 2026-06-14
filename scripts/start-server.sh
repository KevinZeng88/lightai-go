#!/bin/sh
# LightAI Go - Start Server (release mode)
# Usage: ./scripts/start-server.sh [config]
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG="${1:-$RELEASE_ROOT/configs/server.release.yaml}"

cd "$RELEASE_ROOT"

# Create required directories.
mkdir -p logs data run runtime

# Check password.
if [ -z "${LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD:-}" ]; then
  echo "WARNING: LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD not set." >&2
  echo "A random password will be generated and printed ONCE to stderr." >&2
  echo "Set it via: export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='...'" >&2
fi

echo "=== LightAI Go Server ==="
echo "Config: $CONFIG"
echo "Root:   $RELEASE_ROOT"
echo ""

if [ -f run/server.pid ]; then
  PID=$(cat run/server.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "Server already running (PID $PID). Stop it first: ./scripts/stop-server.sh"
    exit 1
  fi
  rm -f run/server.pid
fi

nohup bin/lightai-server --config "$CONFIG" \
  > logs/server-stdout.log 2>&1 &
PID=$!
echo "$PID" > run/server.pid

sleep 2
if kill -0 "$PID" 2>/dev/null; then
  echo "Server started (PID $PID)."
  echo "  Health:  http://127.0.0.1:18080/healthz"
  echo "  Web:     http://127.0.0.1:18080/"
  echo ""
  echo "  Server stdout log:   logs/server-stdout.log"
  echo "  Server main log:     logs/server.log"
  if [ -f runtime/initial-credentials.txt ]; then
    echo "  Initial credentials: runtime/initial-credentials.txt"
  fi
else
  echo "Server failed to start. Check logs/server-stdout.log"
  rm -f run/server.pid
  exit 1
fi
