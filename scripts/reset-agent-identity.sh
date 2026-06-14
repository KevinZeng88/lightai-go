#!/bin/sh
# LightAI Go - Reset Agent Node Identity
# Usage: sh scripts/reset-agent-identity.sh
# Must be run while the agent is stopped.
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

IDENTITY_FILE="runtime/agent-identity.json"

echo "=== LightAI Go Agent Identity Reset ==="
echo ""

# Check agent is not running.
if [ -f run/agent.pid ]; then
  PID=$(cat run/agent.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "ERROR: Agent is running (PID $PID). Stop it first: ./scripts/stop-agent.sh" >&2
    exit 1
  fi
  rm -f run/agent.pid
fi

if [ ! -f "$IDENTITY_FILE" ]; then
  echo "No identity file found at $IDENTITY_FILE"
  echo "A new node_id will be generated on next agent start."
  exit 0
fi

# Show current identity.
echo "Current identity:"
cat "$IDENTITY_FILE"
echo ""

# Backup old identity.
BACKUP_DIR="runtime/identity-backups"
mkdir -p "$BACKUP_DIR"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="$BACKUP_DIR/agent-identity-$TIMESTAMP.json"
cp "$IDENTITY_FILE" "$BACKUP_FILE"
chmod 0600 "$BACKUP_FILE" 2>/dev/null || true

# Remove identity file.
rm -f "$IDENTITY_FILE"

echo "Identity file removed."
echo "Backup saved: $BACKUP_FILE"
echo ""
echo "=== Next Steps ==="
echo "1. A new node_id will be generated on next agent start."
echo "2. The Server will treat this as a NEW node."
echo "3. The OLD node record in Server must be cleaned manually via Web/API."
echo "4. If this was a mistake, restore the backup:"
echo "   cp $BACKUP_FILE $IDENTITY_FILE"
echo ""
echo "Start agent: ./scripts/start-agent.sh"
