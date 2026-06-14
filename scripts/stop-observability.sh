#!/bin/sh
# LightAI Go - Stop Observability (Prometheus + Grafana)
# P1-011/P1-012: PID validation + graceful shutdown
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

GRACEFUL_TIMEOUT="${LIGHTAI_STOP_TIMEOUT:-10}"

echo "=== LightAI Observability - Stop ==="

stop_proc() {
  local name="$1" pidfile="$2" proc_match="$3"
  if [ ! -f "$pidfile" ]; then
    echo "  $name: not running (no PID file)."
    return 0
  fi

  PID=$(cat "$pidfile")

  if ! kill -0 "$PID" 2>/dev/null; then
    echo "  $name: not running (stale PID $PID)."
    rm -f "$pidfile"
    return 0
  fi

  # P1-011: Validate process identity.
  if command -v ps >/dev/null 2>&1; then
    PROC_CMD=$(ps -p "$PID" -o comm= 2>/dev/null || true)
    case "$PROC_CMD" in
      *"$proc_match"*) ;;
      *)
        echo "  $name: PID $PID is '$PROC_CMD', not $proc_match. Refusing to kill." >&2
        echo "  Remove $pidfile manually if the process is gone." >&2
        return 1
        ;;
    esac
  fi

  echo "Stopping $name (PID $PID)..."
  kill "$PID" 2>/dev/null || true

  # P1-012: Graceful wait before SIGKILL.
  waited=0
  while [ "$waited" -lt "$GRACEFUL_TIMEOUT" ]; do
    if ! kill -0 "$PID" 2>/dev/null; then
      echo "  $name stopped gracefully after ${waited}s."
      rm -f "$pidfile"
      return 0
    fi
    sleep 1
    waited=$((waited + 1))
  done

  echo "  $name did not stop after ${GRACEFUL_TIMEOUT}s, sending SIGKILL..."
  kill -9 "$PID" 2>/dev/null || true
  sleep 1
  if kill -0 "$PID" 2>/dev/null; then
    echo "  WARNING: Failed to kill $name (PID $PID)." >&2
    return 1
  fi
  echo "  $name force-stopped."
  rm -f "$pidfile"
}

stop_proc "Grafana" run/grafana.pid "grafana"
stop_proc "Prometheus" run/prometheus.pid "prometheus"
echo "Done."
