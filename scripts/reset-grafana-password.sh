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
  echo "Auto-generated Grafana password."
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

# ---- Locate Grafana binary ----
GRAF_BIN=""
for c in bin/grafana/bin/grafana bin/grafana/bin/grafana-server; do
  if [ -x "$c" ]; then
    GRAF_BIN="$c"
    break
  fi
done

if [ ! -x "$GRAF_BIN" ]; then
  echo "ERROR: Grafana binary not found at bin/grafana/bin/grafana or bin/grafana/bin/grafana-server" >&2
  echo "  Run: ./scripts/prepare-observability-binaries.sh --download" >&2
  exit 1
fi

# ---- Locate Grafana database (same paths as start-observability.sh) ----
GRAFANA_HOME="$RLS_ROOT/bin/grafana"
GRAFANA_INI="$RLS_ROOT/configs/observability/grafana.ini"
GRAFANA_DATA="$RLS_ROOT/data/grafana"
GRAFANA_DB="$GRAFANA_DATA/grafana.db"

if [ ! -f "$GRAFANA_INI" ]; then
  echo "ERROR: Grafana config not found at $GRAFANA_INI" >&2
  exit 1
fi

if [ ! -f "$GRAFANA_DB" ]; then
  echo "ERROR: Grafana database not found at $GRAFANA_DB" >&2
  echo "  Grafana must be started at least once to initialize the database." >&2
  echo "  Run: ./scripts/start-observability.sh" >&2
  exit 1
fi

echo "Grafana binary : $GRAF_BIN"
echo "Grafana home   : $GRAFANA_HOME"
echo "Grafana config : $GRAFANA_INI"
echo "Grafana DB     : $GRAFANA_DB"
echo ""

# ---- Stop Grafana if running ----
GRAFANA_WAS_RUNNING=false
if [ -f run/grafana.pid ]; then
  PID=$(cat run/grafana.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "Grafana is running (PID $PID). Stopping..."
    GRAFANA_WAS_RUNNING=true
    kill "$PID" 2>/dev/null || true
    sleep 2
    if kill -0 "$PID" 2>/dev/null; then
      kill -9 "$PID" 2>/dev/null || true
      sleep 1
    fi
    rm -f run/grafana.pid
  else
    rm -f run/grafana.pid
  fi
fi

# ---- Reset admin password ----
# Uses the same --homepath / --config pattern as start-observability.sh
# (flags after the subcommand, both for "server" and "cli").
echo "Resetting Grafana admin password..."
"$GRAF_BIN" cli \
  --homepath "$GRAFANA_HOME" \
  --config "$GRAFANA_INI" \
  admin reset-admin-password "$NEW_PASS"
echo "Grafana admin password reset successful."

# ---- Write credentials records ----
mkdir -p runtime runtime/observability
TIMESTAMP=$(date -Iseconds)

# 1. Update the runtime credentials file used by start-observability.sh.
cat > runtime/observability/grafana.credentials << CREDEOF
# LightAI Go - Grafana Admin Credentials
# Updated by password reset: $TIMESTAMP
# DO NOT edit manually. Use LIGHTAI_GRAFANA_ADMIN_PASSWORD env var or reset-grafana-password.sh.
USERNAME=admin
PASSWORD=$NEW_PASS
CREDEOF
chmod 0600 runtime/observability/grafana.credentials 2>/dev/null || true
echo "Credentials updated : runtime/observability/grafana.credentials"

# 2. Write human-readable reset record.
cat > runtime/reset-credentials.txt << EOF
============================================
LightAI Go - Grafana Password Reset
Reset time: $TIMESTAMP
============================================

[Grafana]
Username: admin
Password: $NEW_PASS
DB path: $GRAFANA_DB
Service restart required: yes
Next step: ./scripts/start-observability.sh
EOF
chmod 0600 runtime/reset-credentials.txt 2>/dev/null || true
echo "Reset record saved : runtime/reset-credentials.txt"

# ---- Restart Grafana if it was running ----
if $GRAFANA_WAS_RUNNING; then
  echo ""
  echo "Restarting Grafana..."
  if sh "$SCRIPT_DIR/start-observability.sh" 2>/dev/null; then
    echo "Grafana restarted. Verify login: http://127.0.0.1:13000 (admin / <new-password>)"
  else
    echo "WARNING: Grafana restart failed." >&2
    echo "  Start manually: ./scripts/start-observability.sh" >&2
  fi
else
  echo ""
  echo "Grafana was not running. To start: ./scripts/start-observability.sh"
fi
