#!/bin/sh
# LightAI Go - Start Agent (release mode)
# Usage: ./scripts/start-agent.sh [metax|nvidia] [config]
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VENDOR="${1:-metax}"

cd "$RELEASE_ROOT"

# Select config based on vendor.
case "$VENDOR" in
  metax)   CONFIG="${2:-$RELEASE_ROOT/configs/agent.metax.yaml}" ;;
  nvidia)  CONFIG="${2:-$RELEASE_ROOT/configs/agent.nvidia.yaml}" ;;
  *)
    echo "Usage: $0 [metax|nvidia] [config]" >&2
    echo "  metax  - MetaX GPU collector" >&2
    echo "  nvidia - NVIDIA GPU collector" >&2
    exit 1
    ;;
esac

mkdir -p logs data run

echo "=== LightAI Go Agent ==="
echo "Vendor: $VENDOR"
echo "Config: $CONFIG"
echo "Root:   $RELEASE_ROOT"
echo ""

if [ -f run/agent.pid ]; then
  PID=$(cat run/agent.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "Agent already running (PID $PID). Stop it first: ./scripts/stop-agent.sh"
    exit 1
  fi
  rm -f run/agent.pid
fi

nohup bin/lightai-agent --config "$CONFIG" \
  > logs/agent-stdout.log 2>&1 &
PID=$!
echo "$PID" > run/agent.pid

sleep 3
if kill -0 "$PID" 2>/dev/null; then
  echo "Agent started (PID $PID)."
  echo "  Metrics: http://127.0.0.1:19091/metrics"
  echo "  Logs:    logs/agent-stdout.log"
else
  echo "Agent failed to start. Check logs/agent-stdout.log"
  rm -f run/agent.pid
  exit 1
fi
