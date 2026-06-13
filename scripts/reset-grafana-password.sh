#!/bin/sh
# LightAI Go - Reset Grafana Admin Password
# Usage: sh scripts/reset-grafana-password.sh 'NewPassword123!'
set -e

NEW_PASS="${1:-}"
if [ -z "$NEW_PASS" ]; then
  echo "Usage: $0 '<new-password>'" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

GRAF_BIN=""
for c in bin/grafana/bin/grafana bin/grafana/bin/grafana-server; do
  [ -x "$c" ] && { GRAF_BIN="$c"; break; }
done

if [ ! -x "$GRAF_BIN" ]; then
  echo "ERROR: Grafana binary not found." >&2; exit 1
fi

if [ -f run/grafana.pid ]; then
  PID=$(cat run/grafana.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "ERROR: Grafana is running (PID $PID). Stop it first: sh scripts/stop-observability.sh" >&2
    exit 1
  fi
fi

echo "Resetting Grafana admin password..."
"$GRAF_BIN" cli \
  --homepath "$RLS_ROOT/bin/grafana" \
  --config "$RLS_ROOT/configs/observability/grafana.ini" \
  admin reset-admin-password "$NEW_PASS"

echo "Password reset complete."
