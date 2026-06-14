#!/bin/sh
# LightAI Go - Reset Grafana Admin Password
# Usage:
#   sh scripts/reset-grafana-password.sh                    # auto-generate
#   sh scripts/reset-grafana-password.sh 'NewPassword123!'  # specify
#   sh scripts/reset-grafana-password.sh --interactive      # prompt
set -e

INTERACTIVE=false
NEW_PASS=""

case "${1:-}" in
  --interactive)
    INTERACTIVE=true ;;
  --help|-h)
    echo "Usage: $0 [<new-password>] [--interactive]"
    echo "  (no args)          Auto-generate secure random password."
    echo "  '<new-password>'   Use the specified password."
    echo "  --interactive      Prompt for password (avoids shell history)."
    exit 0 ;;
  "")
    # Auto-generate below.
    ;;
  *)
    NEW_PASS="$1" ;;
esac

if $INTERACTIVE; then
  printf "Enter new Grafana admin password: "
  stty -echo 2>/dev/null || true
  read -r INPUT_PASS
  stty echo 2>/dev/null || true
  echo ""
  if [ -z "$INPUT_PASS" ]; then
    echo "ERROR: empty password" >&2
    exit 1
  fi
  NEW_PASS="$INPUT_PASS"
fi

if [ -z "$NEW_PASS" ]; then
  NEW_PASS=$(head -c 16 /dev/urandom 2>/dev/null | base64 2>/dev/null | tr -dc 'A-Za-z0-9' | head -c 20 || echo "")
  if [ -z "$NEW_PASS" ]; then
    NEW_PASS="LightAI@$(date +%s | tail -c 9)"
  fi
  echo "Auto-generated password: $NEW_PASS"
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

GRAFANA_WAS_RUNNING=false
if [ -f run/grafana.pid ]; then
  PID=$(cat run/grafana.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "Grafana is running (PID $PID). Stopping..."
    GRAFANA_WAS_RUNNING=true
    kill "$PID" 2>/dev/null || true
    sleep 2
    rm -f run/grafana.pid
  fi
fi

echo "Resetting Grafana admin password..."
"$GRAF_BIN" cli \
  --homepath "$RLS_ROOT/bin/grafana" \
  --config "$RLS_ROOT/configs/observability/grafana.ini" \
  admin reset-admin-password "$NEW_PASS"

# Write credentials record.
mkdir -p runtime
CRED_FILE="runtime/reset-credentials.txt"
TIMESTAMP=$(date -Iseconds)
{
  echo "============================================"
  echo "LightAI Go - Grafana Password Reset"
  echo "Reset time: $TIMESTAMP"
  echo "============================================"
  echo ""
  echo "[Grafana]"
  echo "Username: admin"
  echo "Password: $NEW_PASS"
  echo "Service restart required: yes"
  echo "Next step: ./scripts/start-observability.sh"
} > "$CRED_FILE"
chmod 0600 "$CRED_FILE" 2>/dev/null || true

echo "Password reset complete."
echo "Credentials saved: $CRED_FILE"

if $GRAFANA_WAS_RUNNING; then
  echo "Restarting Grafana..."
  sh "$SCRIPT_DIR/start-observability.sh" 2>/dev/null || \
    echo "Start manually: ./scripts/start-observability.sh"
else
  echo "Start Grafana: ./scripts/start-observability.sh"
fi
