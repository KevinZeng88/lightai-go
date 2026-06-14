#!/bin/sh
# LightAI Go - Start Observability (Prometheus + Grafana, bundled mode)
#
# Credential hierarchy (P0-004):
#   1. LIGHTAI_GRAFANA_ADMIN_PASSWORD env var (user-provided)
#   2. runtime/observability/grafana.credentials (persisted from previous run)
#   3. Auto-generated random password (first run only)
#
# The configs/observability/grafana.env file is a TEMPLATE ONLY and
# must NOT override runtime credentials.
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

mkdir -p data/prometheus data/grafana data/grafana/plugins logs run
mkdir -p runtime/observability
mkdir -p deploy/observability/grafana/provisioning/plugins
mkdir -p deploy/observability/grafana/provisioning/alerting

PROM_BIN="bin/prometheus"
GRAF_BIN=""
GRAF_V13=false
for candidate in bin/grafana/bin/grafana-server bin/grafana/bin/grafana; do
  if [ -x "$candidate" ]; then
    GRAF_BIN="$candidate"
    case "$(basename "$candidate")" in
      grafana) GRAF_V13=true ;;
    esac
    break
  fi
done

# --- Resolve Grafana credentials (P0-004) ---
GRAFANA_ADMIN_USER="${LIGHTAI_GRAFANA_ADMIN_USER:-admin}"
GRAFANA_DB="data/grafana/grafana.db"
CRED_FILE="runtime/observability/grafana.credentials"
GRAFANA_PASSWORD_PROVIDED=false

# Check if user explicitly provided a password via env var.
if [ -n "${LIGHTAI_GRAFANA_ADMIN_PASSWORD:-}" ]; then
  GRAFANA_ADMIN_PASSWORD="$LIGHTAI_GRAFANA_ADMIN_PASSWORD"
  GRAFANA_PASSWORD_PROVIDED=true
fi

if [ -f "$GRAFANA_DB" ]; then
  # P1-011: Grafana DB already exists. LIGHTAI_GRAFANA_ADMIN_PASSWORD env var
  # will NOT modify the DB password (stored on first init). To reset: ./scripts/reset-grafana-password.sh
  if [ -z "${GRAFANA_ADMIN_PASSWORD:-}" ]; then
    # Try to read from persisted credentials file.
    if [ -f "$CRED_FILE" ]; then
      SAVED_PASS=$(grep '^PASSWORD=' "$CRED_FILE" 2>/dev/null | cut -d= -f2-)
      if [ -n "$SAVED_PASS" ]; then
        GRAFANA_ADMIN_PASSWORD="$SAVED_PASS"
        GRAFANA_PASSWORD_PROVIDED=true
      fi
    fi
  fi
  if [ -z "${GRAFANA_ADMIN_PASSWORD:-}" ]; then
    echo "WARNING: Grafana DB exists but no credentials found."
    echo "  Use LIGHTAI_GRAFANA_ADMIN_PASSWORD to set the password,"
    echo "  or run: ./scripts/reset-grafana-password.sh"
    GRAFANA_ADMIN_PASSWORD="<stored-in-grafana-db>"
  fi
else
  # First time Grafana init.
  if [ -z "${GRAFANA_ADMIN_PASSWORD:-}" ]; then
    # Generate random password (20 alphanumeric characters).
    GRAFANA_ADMIN_PASSWORD=$(head -c 24 /dev/urandom 2>/dev/null | base64 2>/dev/null | tr -dc 'A-Za-z0-9' | head -c 20 || echo "")
    if [ -z "$GRAFANA_ADMIN_PASSWORD" ]; then
      GRAFANA_ADMIN_PASSWORD=$(date +%s | sha256sum 2>/dev/null | head -c 20 || echo "LightAI@$(date +%s)")
    fi
    GRAFANA_PASSWORD_PROVIDED=false
  fi

  # Persist credentials to runtime file (P0-004).
  mkdir -p runtime/observability
  cat > "$CRED_FILE" << CREDEOF
# LightAI Go - Grafana Admin Credentials
# Generated: $(date -Iseconds)
# DO NOT edit manually. Use LIGHTAI_GRAFANA_ADMIN_PASSWORD env var.
USERNAME=$GRAFANA_ADMIN_USER
PASSWORD=$GRAFANA_ADMIN_PASSWORD
CREDEOF
  chmod 0600 "$CRED_FILE" 2>/dev/null || true
fi

# DO NOT source configs/observability/grafana.env here — it would override
# the runtime credentials we just resolved (P0-004 fix).
# The grafana.env file is a deployment template only.

echo "=== LightAI Observability ==="

# --- Prometheus ---
echo ""
echo "[Prometheus]"
if [ -f run/prometheus.pid ]; then
  PID=$(cat run/prometheus.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "  Running (PID $PID)"
  else
    rm -f run/prometheus.pid
    echo "  Not running (stale PID cleaned)"
  fi
else
  echo "  Not running"
fi

if [ ! -f run/prometheus.pid ]; then
  nohup "$PROM_BIN" \
    --config.file=configs/observability/prometheus.yml \
    --storage.tsdb.path=data/prometheus \
    --storage.tsdb.retention.time=15d \
    --web.listen-address=0.0.0.0:19090 \
    --web.enable-lifecycle \
    > logs/prometheus.log 2>&1 &
  PID=$!
  echo "$PID" > run/prometheus.pid
  echo "  Started (PID $PID)"
fi

# --- Grafana ---
echo ""
echo "[Grafana]"
if [ -f run/grafana.pid ]; then
  PID=$(cat run/grafana.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "  Running (PID $PID)"
  else
    rm -f run/grafana.pid
    echo "  Not running (stale PID cleaned)"
  fi
else
  echo "  Not running"
fi

if [ ! -f run/grafana.pid ]; then
  if [ ! -x "$GRAF_BIN" ]; then
    echo "  ERROR: Grafana binary not found (bin/grafana/bin/grafana)"
    echo "  Run: ./scripts/prepare-observability-binaries.sh --download"
    exit 1
  fi

  # Grafana 13+: server subcommand with --homepath and --config.
  # Pre-13:  grafana-server with GF_* env vars.
  if $GRAF_V13; then
    echo "  Using Grafana 13+ mode"
    # Write dashboards.yaml with absolute path at startup time.
    mkdir -p "$RLS_ROOT/deploy/observability/grafana/provisioning/dashboards"
    cat > "$RLS_ROOT/deploy/observability/grafana/provisioning/dashboards/dashboards.yaml" << YAMLEOF
apiVersion: 1
providers:
  - name: LightAI
    orgId: 1
    folder: ''
    type: file
    disableDeletion: true
    editable: true
    options:
      path: $RLS_ROOT/deploy/observability/grafana/dashboards
YAMLEOF
    GF_PATHS_PROVISIONING="$RLS_ROOT/deploy/observability/grafana/provisioning" \
    GF_SECURITY_ADMIN_USER="${GRAFANA_ADMIN_USER}" \
    GF_SECURITY_ADMIN_PASSWORD="${GRAFANA_ADMIN_PASSWORD}" \
    nohup "$GRAF_BIN" server \
      --homepath "$RLS_ROOT/bin/grafana" \
      --config "$RLS_ROOT/configs/observability/grafana.ini" \
      > logs/grafana.log 2>&1 &
  else
    GF_PATHS_CONFIG=configs/observability/grafana.ini \
    GF_PATHS_DATA=data/grafana \
    GF_PATHS_LOGS=logs \
    GF_PATHS_PLUGINS=data/grafana/plugins \
    GF_PATHS_PROVISIONING=deploy/observability/grafana/provisioning \
    GF_SECURITY_ADMIN_USER="${GRAFANA_ADMIN_USER}" \
    GF_SECURITY_ADMIN_PASSWORD="${GRAFANA_ADMIN_PASSWORD}" \
    GF_SERVER_HTTP_ADDR=0.0.0.0 \
    GF_SERVER_HTTP_PORT=13000 \
    GF_DATABASE_TYPE=sqlite3 \
    GF_DATABASE_PATH=data/grafana/grafana.db \
    GF_ANALYTICS_REPORTING_ENABLED=false \
    GF_ANALYTICS_CHECK_FOR_UPDATES=false \
    nohup "$GRAF_BIN" > logs/grafana.log 2>&1 &
  fi
  PID=$!
  echo "$PID" > run/grafana.pid
  echo "  Started (PID $PID)"
  if $GRAFANA_PASSWORD_PROVIDED; then
    echo "  Grafana using user-provided admin password."
  else
    echo "  Grafana generated random admin password (first init)."
    echo "  Credentials saved to: $CRED_FILE"
  fi
fi

# --- Wait for readiness ---
echo ""
echo "Waiting for services to be ready..."

grafana_ok=false
for i in 1 2 3 4 5 6 7 8; do
  if curl -sf http://127.0.0.1:13000/api/health >/dev/null 2>&1; then
    grafana_ok=true
    break
  fi
  sleep 3
done

prom_ok=false
for i in 1 2 3; do
  if curl -sf http://127.0.0.1:19090/-/ready >/dev/null 2>&1; then
    prom_ok=true
    break
  fi
  sleep 2
done

echo "  Prometheus: $($prom_ok && echo 'Ready (http://127.0.0.1:19090)' || echo 'Not ready')"
echo "  Grafana:    $($grafana_ok && echo 'Ready (http://127.0.0.1:13000)' || echo 'Not ready')"

if ! $grafana_ok; then
  echo ""
  echo "Grafana failed to start. Check:"
  echo "  tail -50 logs/grafana.log"
  echo "  Command: bin/grafana/bin/grafana server --homepath bin/grafana --config configs/observability/grafana.ini"
  rm -f run/grafana.pid
  exit 1
fi

echo ""
echo "Observability started."
echo ""
echo "=== Prometheus Queries ==="
echo "  up                                     # All target status"
echo "  lightai_host_cpu_usage_ratio           # CPU usage"
echo "  lightai_host_memory_used_ratio         # Memory usage"
echo "  lightai_host_filesystem_used_ratio     # Disk usage"
echo "  lightai_gpu_memory_total_bytes         # GPU memory total"
echo "  lightai_gpu_memory_used_bytes          # GPU memory used"
echo ""
echo "Prometheus showing 'No data queried yet' is normal."
echo "Enter a query expression above to see data."
echo ""
echo "  LAN Prometheus: http://<server-ip>:19090/"
echo "  LAN Grafana:    http://<server-ip>:13000/"

if [ -f "$CRED_FILE" ]; then
  echo ""
  echo "  Credentials file: $CRED_FILE"
  echo "  Username: $(grep '^USERNAME=' "$CRED_FILE" 2>/dev/null | cut -d= -f2)"
  echo "  Login and change the default password if not already done."
fi
