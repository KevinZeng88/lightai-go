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

# Check password. LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD is canonical for clean-DB first start.
# LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD is supported as legacy fallback with backward compat.
if [ -z "${LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD:-}" ] && [ -z "${LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD:-}" ]; then
  echo "WARNING: Neither LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD nor LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD is set." >&2
  echo "A random password will be generated and written to runtime/initial-credentials.txt." >&2
  echo "Set it via: export LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD='...'" >&2
  echo "  (LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD is also accepted for backward compat)" >&2
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
  echo "  Server stdout log: logs/server-stdout.log"
  echo "  Server main log:   logs/lightai-server.log"
  if [ -f runtime/initial-credentials.txt ]; then
    echo "  Initial credentials: runtime/initial-credentials.txt"
  fi
else
	echo "Server failed to start."
	echo ""
	echo "--- Diagnostic information ---"
	echo "Working directory: $(pwd)"
	echo "Config path:       $CONFIG"
	echo "Server binary:     $(test -x bin/lightai-server && echo bin/lightai-server || echo "not found")"
	echo "Configs tree:"
	find configs -maxdepth 3 -type d 2>/dev/null | sort | sed "s/^/  /" || echo "  (no configs directory)"
	echo ""
	echo "--- Last 80 lines of logs/server-stdout.log ---"
	tail -80 logs/server-stdout.log 2>/dev/null | grep -viE "password|token|cookie|csrf|secret|key" || echo "  (empty or not found)"
	echo ""
	echo "--- Last 80 lines of logs/lightai-server.log ---"
	tail -80 logs/lightai-server.log 2>/dev/null | grep -viE "password|token|cookie|csrf|secret|key" || echo "  (empty or not found)"
  rm -f run/server.pid
  exit 1
fi
