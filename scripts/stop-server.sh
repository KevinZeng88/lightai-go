#!/bin/sh
# LightAI Go - Stop Server (P1-011/P1-012: PID validation + graceful shutdown)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RELEASE_ROOT"

# Graceful shutdown timeout in seconds (configurable via env var).
GRACEFUL_TIMEOUT="${LIGHTAI_STOP_TIMEOUT:-10}"

if [ ! -f run/server.pid ]; then
  echo "Server not running (no PID file)."
  exit 0
fi

PID=$(cat run/server.pid)

# P1-011: Validate PID still belongs to lightai-server.
if ! kill -0 "$PID" 2>/dev/null; then
  echo "Server not running (stale PID $PID)."
  rm -f run/server.pid
  exit 0
fi

# Verify the process is actually lightai-server (not a reused PID).
if command -v ps >/dev/null 2>&1; then
  PROC_CMD=$(ps -p "$PID" -o comm= 2>/dev/null || true)
  case "$PROC_CMD" in
    *lightai-server*) ;;
    *)
      echo "WARNING: PID $PID is '$PROC_CMD', not lightai-server. Refusing to kill." >&2
      echo "Remove run/server.pid manually if the process is gone." >&2
      exit 1
      ;;
  esac
fi

echo "Stopping server (PID $PID)..."
kill "$PID" 2>/dev/null || true

# P1-012: Wait gracefully for process to exit before SIGKILL.
waited=0
while [ "$waited" -lt "$GRACEFUL_TIMEOUT" ]; do
  if ! kill -0 "$PID" 2>/dev/null; then
    echo "Server stopped gracefully after ${waited}s."
    rm -f run/server.pid
    exit 0
  fi
  sleep 1
  waited=$((waited + 1))
done

# Graceful timeout exceeded — force kill.
echo "Server did not stop after ${GRACEFUL_TIMEOUT}s, sending SIGKILL..."
kill -9 "$PID" 2>/dev/null || true
sleep 1
if kill -0 "$PID" 2>/dev/null; then
  echo "WARNING: Failed to kill server (PID $PID)." >&2
  exit 1
fi
echo "Server force-stopped."
rm -f run/server.pid
