#!/bin/sh
# LightAI Go - Reset Admin Passwords
# Usage:
#   scripts/reset-password.sh                         # reset Web/Admin only (auto-generate)
#   scripts/reset-password.sh --password 'NewPwd'     # specify password
#   scripts/reset-password.sh --interactive           # interactive prompt
#   scripts/reset-password.sh --grafana-only          # reset Grafana only
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

NEW_PASSWORD=""
INTERACTIVE=false
GRAFANA_ONLY=false

while [ $# -gt 0 ]; do
  case "$1" in
    --password)
      NEW_PASSWORD="$2"; shift 2 ;;
    --interactive)
      INTERACTIVE=true; shift ;;
    --grafana-only)
      GRAFANA_ONLY=true; shift ;;
    --help|-h)
      echo "Usage: $0 [--password <pw>] [--interactive] [--grafana-only]"
      echo ""
      echo "Reset admin passwords for LightAI Go components."
      echo ""
      echo "Modes:"
      echo "  (default)          Reset LightAI Web/Admin password only."
      echo "  --password <pw>    Use the specified password."
      echo "  --interactive      Prompt for password (no shell history leak)."
      echo ""
      echo "Scope:"
      echo "  (default)          LightAI Web admin only."
      echo "  --grafana-only     Reset Grafana admin password only."
      echo ""
      echo "Examples:"
      echo "  $0                                  # auto-generate, Web only"
      echo "  $0 --password 'MyStr0ngP@ss'       # specify, Web only"
      echo "  $0 --interactive --grafana-only    # prompt, Grafana only"
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      echo "Usage: $0 [--password <pw>] [--interactive] [--grafana-only] [--help]"
      exit 1
      ;;
  esac
done

# Determine password.
if [ "$INTERACTIVE" = true ]; then
  printf "Enter new admin password: "
  stty -echo 2>/dev/null || true
  read -r INPUT_PASS
  stty echo 2>/dev/null || true
  echo ""
  if [ -z "$INPUT_PASS" ]; then
    echo "ERROR: empty password" >&2
    exit 1
  fi
  NEW_PASSWORD="$INPUT_PASS"
fi

mkdir -p runtime
TIMESTAMP=$(date -Iseconds)

if $GRAFANA_ONLY; then
  # ================================================================
  # Grafana-only mode
  # ================================================================
  echo "=== Reset Grafana Admin Password ==="

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

  # Same paths as start-observability.sh.
  GRAFANA_HOME="$RLS_ROOT/bin/grafana"
  GRAFANA_INI="$RLS_ROOT/configs/observability/grafana.ini"
  GRAFANA_DB="$RLS_ROOT/data/grafana/grafana.db"

  if [ ! -f "$GRAFANA_INI" ]; then
    echo "ERROR: Grafana config not found at $GRAFANA_INI" >&2
    exit 1
  fi
  if [ ! -f "$GRAFANA_DB" ]; then
    echo "ERROR: Grafana database not found at $GRAFANA_DB" >&2
    echo "  Grafana must be started at least once to initialize the database." >&2
    exit 1
  fi

  GRAF_PASS="$NEW_PASSWORD"
  if [ -z "$GRAF_PASS" ]; then
    GRAF_PASS=$(head -c 16 /dev/urandom 2>/dev/null | base64 2>/dev/null | tr -dc 'A-Za-z0-9' | head -c 20 || echo "")
    if [ -z "$GRAF_PASS" ]; then
      GRAF_PASS="LightAI@$(date +%s | tail -c 9)"
    fi
  fi

  echo "Grafana binary : $GRAF_BIN"
  echo "Grafana home   : $GRAFANA_HOME"
  echo "Grafana config : $GRAFANA_INI"
  echo "Grafana DB     : $GRAFANA_DB"
  echo ""

  # Stop Grafana if running.
  GRAFANA_WAS_RUNNING=false
  if [ -f run/grafana.pid ]; then
    PID=$(cat run/grafana.pid)
    if kill -0 "$PID" 2>/dev/null; then
      GRAFANA_WAS_RUNNING=true
      echo "Grafana is running (PID $PID). Stopping..."
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

  echo "Resetting Grafana admin password..."
  # Same --homepath/--config pattern as start-observability.sh (flags after cli).
  "$GRAF_BIN" cli \
    --homepath "$GRAFANA_HOME" \
    --config "$GRAFANA_INI" \
    admin reset-admin-password "$GRAF_PASS"
  echo "Grafana admin password reset successful."

  # Update runtime credentials.
  mkdir -p runtime/observability
  cat > runtime/observability/grafana.credentials << CREDEOF
# LightAI Go - Grafana Admin Credentials
# Updated by password reset: $TIMESTAMP
USERNAME=admin
PASSWORD=$GRAF_PASS
CREDEOF
  chmod 0600 runtime/observability/grafana.credentials 2>/dev/null || true

  # Write reset record.
  cat > runtime/reset-credentials.txt << EOF
============================================
LightAI Go - Grafana Password Reset
Reset time: $TIMESTAMP
============================================

[Grafana]
Username: admin
Password: $GRAF_PASS
DB path: $GRAFANA_DB
Service restart required: yes
EOF
  chmod 0600 runtime/reset-credentials.txt 2>/dev/null || true

  # Restart if was running.
  if $GRAFANA_WAS_RUNNING; then
    echo ""
    echo "Restarting Grafana..."
    sh "$SCRIPT_DIR/start-observability.sh" 2>/dev/null || \
      echo "WARNING: Grafana restart failed. Start manually: ./scripts/start-observability.sh"
  else
    echo ""
    echo "Grafana was not running. To start: ./scripts/start-observability.sh"
  fi

  echo ""
  echo "Credentials saved: runtime/reset-credentials.txt"
  echo "Runtime credentials: runtime/observability/grafana.credentials"

else
  # ================================================================
  # Default: LightAI Web/Admin only
  # ================================================================
  echo "=== Reset LightAI Web/Admin Password ==="

  SERVER_BIN=""
  for c in bin/lightai-server; do
    [ -x "$c" ] && { SERVER_BIN="$c"; break; }
  done

  if [ ! -x "$SERVER_BIN" ]; then
    echo "ERROR: lightai-server binary not found at bin/lightai-server" >&2
    echo "Build it first: go build -o bin/lightai-server ./cmd/server" >&2
    exit 1
  fi

  if [ -z "$NEW_PASSWORD" ]; then
    NEW_PASSWORD=$(head -c 16 /dev/urandom 2>/dev/null | base64 2>/dev/null | tr -dc "A-Za-z0-9" | head -c 20 || echo "")
    if [ -z "$NEW_PASSWORD" ]; then
      NEW_PASSWORD="LightAI@$(date +%s | tail -c 9)"
    fi
  fi

  "$SERVER_BIN" --reset-admin-password "$NEW_PASSWORD"
  echo "Web/Admin password reset complete."

  # Write reset record.
  cat > runtime/reset-credentials.txt << EOF
============================================
LightAI Go - Web/Admin Password Reset
Reset time: $TIMESTAMP
============================================

[Web/Admin]
Username: admin
Password: $NEW_PASSWORD
EOF
  chmod 0600 runtime/reset-credentials.txt 2>/dev/null || true

  echo ""
  echo "Credentials saved: runtime/reset-credentials.txt"
  echo ""
  echo "Note: Grafana password was NOT modified."
  echo "  To reset Grafana: $0 --grafana-only"
  echo "  Or use: ./scripts/reset-grafana-password.sh"
fi
