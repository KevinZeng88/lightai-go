#!/bin/sh
# LightAI Go - Reset Admin Passwords
# Usage:
#   scripts/reset-password.sh                         # auto-generate all passwords
#   scripts/reset-password.sh --password 'NewPwd'     # specify password
#   scripts/reset-password.sh --interactive           # interactive prompt
#   scripts/reset-password.sh --web-only              # reset Web/Admin only
#   scripts/reset-password.sh --grafana-only          # reset Grafana only
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

NEW_PASSWORD=""
INTERACTIVE=false
WEB_ONLY=false
GRAFANA_ONLY=false

while [ $# -gt 0 ]; do
  case "$1" in
    --password)
      NEW_PASSWORD="$2"; shift 2 ;;
    --interactive)
      INTERACTIVE=true; shift ;;
    --web-only)
      WEB_ONLY=true; shift ;;
    --grafana-only)
      GRAFANA_ONLY=true; shift ;;
    --help|-h)
      echo "Usage: $0 [--password <pw>] [--interactive] [--web-only] [--grafana-only]"
      echo ""
      echo "Reset admin passwords for LightAI Go components."
      echo ""
      echo "Modes:"
      echo "  (default)          Auto-generate secure random passwords for all components."
      echo "  --password <pw>    Use the specified password."
      echo "  --interactive      Prompt for password (no shell history leak)."
      echo ""
      echo "Scope (default: both):"
      echo "  --web-only         Reset Web/Admin password only."
      echo "  --grafana-only     Reset Grafana admin password only."
      echo ""
      echo "Examples:"
      echo "  $0                                  # auto-generate"
      echo "  $0 --password 'MyStr0ngP@ss'       # specify"
      echo "  $0 --interactive                   # prompt"
      echo "  $0 --interactive --grafana-only    # prompt, Grafana only"
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      echo "Usage: $0 [--password <pw>] [--interactive] [--web-only] [--grafana-only] [--help]"
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
CRED_FILE="runtime/reset-credentials.txt"
TIMESTAMP=$(date -Iseconds)

# Header for credentials file.
cat > "$CRED_FILE" << EOF
============================================
LightAI Go - Password Reset
Reset time: $TIMESTAMP
============================================

EOF

DO_WEB=false
DO_GRAF=false
if $WEB_ONLY && ! $GRAFANA_ONLY; then
  DO_WEB=true
elif $GRAFANA_ONLY && ! $WEB_ONLY; then
  DO_GRAF=true
else
  DO_WEB=true
  DO_GRAF=true
fi

# --- Reset Web/Admin ---
if $DO_WEB; then
  echo "=== Reset Web/Admin Password ==="

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
            # Auto-generate before calling server binary (flag requires non-empty value).
            NEW_PASSWORD=$(head -c 16 /dev/urandom 2>/dev/null | base64 2>/dev/null | tr -dc "A-Za-z0-9" | head -c 20 || echo "")
            if [ -z "$NEW_PASSWORD" ]; then
              NEW_PASSWORD="LightAI@$(date +%s | tail -c 9)"
            fi
          fi
          "$SERVER_BIN" --reset-admin-password "$NEW_PASSWORD"

  echo "Web/Admin password reset complete."
  echo ""
fi

# --- Reset Grafana ---
if $DO_GRAF; then
  echo "=== Reset Grafana Admin Password ==="

  GRAF_BIN=""
  for c in bin/grafana/bin/grafana bin/grafana/bin/grafana-server; do
    [ -x "$c" ] && { GRAF_BIN="$c"; break; }
  done

  if [ ! -x "$GRAF_BIN" ]; then
    echo "WARNING: Grafana binary not found. Skipping Grafana reset." >&2
    echo "Install observability binaries: ./scripts/prepare-observability-binaries.sh --download" >&2
  else
    GRAFANA_RUNNING=false
    if [ -f run/grafana.pid ]; then
      PID=$(cat run/grafana.pid)
      if kill -0 "$PID" 2>/dev/null; then
        GRAFANA_RUNNING=true
        echo "Grafana is running (PID $PID). Stopping it first..."
        kill "$PID" 2>/dev/null || true
        sleep 2
        if kill -0 "$PID" 2>/dev/null; then
          echo "ERROR: Failed to stop Grafana." >&2
          exit 1
        fi
        rm -f run/grafana.pid
      fi
    fi

    GRAF_PASS="$NEW_PASSWORD"
    if [ -z "$GRAF_PASS" ]; then
      GRAF_PASS=$(head -c 16 /dev/urandom 2>/dev/null | base64 2>/dev/null | tr -dc 'A-Za-z0-9' | head -c 20 || echo "")
      if [ -z "$GRAF_PASS" ]; then
        GRAF_PASS="LightAI@$(date +%s | tail -c 9)"
      fi
    fi

    GF_PATHS_CONFIG="$RLS_ROOT/configs/observability/grafana.ini" \
    GF_PATHS_DATA="$RLS_ROOT/data/grafana" \
    GF_PATHS_PROVISIONING="$RLS_ROOT/deploy/observability/grafana/provisioning" \
    "$GRAF_BIN" cli \
      --homepath "$RLS_ROOT/bin/grafana" \
      --config "$RLS_ROOT/configs/observability/grafana.ini" \
      admin reset-admin-password "$GRAF_PASS"

    # Append to credentials file.
    cat >> "$CRED_FILE" << EOF
[Grafana]
Username: admin
Password: $GRAF_PASS
Note: Grafana admin password has been reset.
Service restart required: yes
Next step: Run ./scripts/start-observability.sh to restart Grafana.
EOF

    # Restart Grafana if it was running.
    if $GRAFANA_RUNNING; then
      echo "Restarting Grafana..."
      sh "$SCRIPT_DIR/start-observability.sh" 2>/dev/null || echo "WARNING: Grafana restart failed. Start manually: ./scripts/start-observability.sh"
    else
      echo "Grafana password reset. Start Grafana: ./scripts/start-observability.sh"
    fi

    echo "Grafana admin password reset complete."
  fi
fi

chmod 0600 "$CRED_FILE" 2>/dev/null || true

echo ""
echo "============================================"
echo "Password reset summary"
echo "============================================"
echo "Credentials saved to: $CRED_FILE"
echo ""
if [ -f "$CRED_FILE" ]; then
  cat "$CRED_FILE"
fi
echo ""
echo "Next steps:"
echo "  1. Save the new password(s) in a secure location."
echo "  2. If Grafana was reset, restart: ./scripts/start-observability.sh"
echo "  3. Login and change password(s) immediately."
