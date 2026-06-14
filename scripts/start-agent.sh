#!/bin/sh
# LightAI Go - Start Agent (release mode)
# Usage: ./scripts/start-agent.sh [config]
# Default: configs/agent.nvidia.yaml
# Override: LIGHTAI_AGENT_CONFIG env var or first argument
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RELEASE_ROOT"

# Priority: command-line arg > LIGHTAI_AGENT_CONFIG > default NVIDIA
if [ -n "${1:-}" ]; then
  CONFIG="$1"
elif [ -n "${LIGHTAI_AGENT_CONFIG:-}" ]; then
  CONFIG="$LIGHTAI_AGENT_CONFIG"
else
  CONFIG="$RELEASE_ROOT/configs/agent.nvidia.yaml"
fi

if [ ! -f "$CONFIG" ]; then
  echo "Agent config not found: $CONFIG" >&2
  exit 1
fi

mkdir -p logs data run runtime

echo "=== LightAI Go Agent ==="
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
  echo ""
  echo "  Agent stdout log: logs/agent-stdout.log"
  echo "  Agent main log:   logs/agent.log"
else
  echo "Agent failed to start. Check logs/agent-stdout.log"
  rm -f run/agent.pid
  exit 1
fi
