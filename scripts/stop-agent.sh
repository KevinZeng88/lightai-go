#!/bin/sh
# LightAI Go - Stop Agent (P1-011/P1-012: PID validation + graceful shutdown)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RELEASE_ROOT"

GRACEFUL_TIMEOUT="${LIGHTAI_STOP_TIMEOUT:-10}"

if [ ! -f run/agent.pid ]; then
  echo "Agent not running (no PID file)."
  exit 0
fi

PID=$(cat run/agent.pid)

# P1-011: Validate PID belongs to lightai-agent.
if ! kill -0 "$PID" 2>/dev/null; then
  echo "Agent not running (stale PID $PID)."
  rm -f run/agent.pid
  exit 0
fi

if command -v ps >/dev/null 2>&1; then
  PROC_CMD=$(ps -p "$PID" -o comm= 2>/dev/null || true)
  case "$PROC_CMD" in
    *lightai-agent*) ;;
    *)
      echo "WARNING: PID $PID is '$PROC_CMD', not lightai-agent. Refusing to kill." >&2
      echo "Remove run/agent.pid manually if the process is gone." >&2
      exit 1
      ;;
  esac
fi

echo "Stopping agent (PID $PID)..."
kill "$PID" 2>/dev/null || true

# P1-012: Graceful wait before SIGKILL.
waited=0
while [ "$waited" -lt "$GRACEFUL_TIMEOUT" ]; do
  if ! kill -0 "$PID" 2>/dev/null; then
    echo "Agent stopped gracefully after ${waited}s."
    rm -f run/agent.pid
    exit 0
  fi
  sleep 1
  waited=$((waited + 1))
done

echo "Agent did not stop after ${GRACEFUL_TIMEOUT}s, sending SIGKILL..."
kill -9 "$PID" 2>/dev/null || true
sleep 1
if kill -0 "$PID" 2>/dev/null; then
  echo "WARNING: Failed to kill agent (PID $PID)." >&2
  exit 1
fi
echo "Agent force-stopped."
rm -f run/agent.pid
