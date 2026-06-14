#!/bin/sh
# LightAI Go - Start Observability (Prometheus + Grafana, bundled mode)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

mkdir -p data/prometheus data/grafana data/grafana/plugins logs run
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

GRAFANA_ADMIN_USER="${GRAFANA_ADMIN_USER:-admin}"
GRAFANA_DB="data/grafana/grafana.db"
CRED_FILE="runtime/initial-credentials.txt"
GRAFANA_PASSWORD_PROVIDED=true

# If Grafana DB already exists, the env var won't take effect —
# the password is already stored in the DB.
if [ -f "$GRAFANA_DB" ]; then
  # Grafana already initialized. Use whatever the DB has.
  # Look up saved password from credentials file if available.
  GRAFANA_ADMIN_PASSWORD="${GRAFANA_ADMIN_PASSWORD:-}"
  if [ -z "${GRAFANA_ADMIN_PASSWORD:-}" ]; then
    # Try to read from credentials file (set by previous run).
    if [ -f "$CRED_FILE" ]; then
      SAVED_PASS=$(grep -A1 '\[Grafana\]' "$CRED_FILE" 2>/dev/null | grep 'Password:' | sed 's/Password: //')
      [ -n "$SAVED_PASS" ] && GRAFANA_ADMIN_PASSWORD="$SAVED_PASS"
    fi
  fi
  # If still empty, use a placeholder — Grafana will use its DB.
  GRAFANA_ADMIN_PASSWORD="${GRAFANA_ADMIN_PASSWORD:-<stored-in-grafana-db>}"
else
  # First time Grafana init. Generate password if not provided.
  if [ -z "${GRAFANA_ADMIN_PASSWORD:-}" ]; then
    GRAFANA_ADMIN_PASSWORD=$(head -c 16 /dev/urandom 2>/dev/null | base64 2>/dev/null | tr -dc 'A-Za-z0-9' | head -c 20 || echo "")
    if [ -z "$GRAFANA_ADMIN_PASSWORD" ]; then
      GRAFANA_ADMIN_PASSWORD=$(date +%s | sha256sum 2>/dev/null | head -c 20 || echo "LightAI@$(date +%s)")
    fi
    GRAFANA_PASSWORD_PROVIDED=false
  fi

  # Write/append Grafana credentials.
  mkdir -p runtime
  if [ -f "$CRED_FILE" ]; then
    # Append Grafana section to existing credentials file.
    if ! grep -q '^\[Grafana\]$' "$CRED_FILE" 2>/dev/null; then
      cat >> "$CRED_FILE" << CREDEOF

[Grafana]
Username: $GRAFANA_ADMIN_USER
Password: $GRAFANA_ADMIN_PASSWORD
Note: Grafana admin password. Change after first login.
Written: $(date -Iseconds)
CREDEOF
    fi
  else
    # Create standalone credentials file (shouldn't normally happen —
    # server bootstrap creates it first, but handle edge case).
    cat > "$CRED_FILE" << CREDEOF
============================================
LightAI Go - Initial Credentials
Generated: $(date -Iseconds)
============================================

[Grafana]
Username: $GRAFANA_ADMIN_USER
Password: $GRAFANA_ADMIN_PASSWORD
Note: Grafana admin password. Change after first login.
CREDEOF
  fi
  chmod 0600 "$CRED_FILE" 2>/dev/null || true
fi

[ -f configs/observability/grafana.env ] && . configs/observability/grafana.env

echo "=== LightAI Observability ==="

# --- Prometheus ---
echo ""
echo "[Prometheus]"
if [ -f run/prometheus.pid ]; then
  PID=$(cat run/prometheus.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "  运行中 (PID $PID)"
  else
    rm -f run/prometheus.pid
    echo "  未运行 (已清理残留 PID)"
  fi
else
  echo "  未运行"
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
  echo "  已启动 (PID $PID)"
fi

# --- Grafana ---
echo ""
echo "[Grafana]"
if [ -f run/grafana.pid ]; then
  PID=$(cat run/grafana.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "  运行中 (PID $PID)"
  else
    rm -f run/grafana.pid
    echo "  未运行 (已清理残留 PID)"
  fi
else
  echo "  未运行"
fi

if [ ! -f run/grafana.pid ]; then
  if [ ! -x "$GRAF_BIN" ]; then
    echo "  错误: Grafana 二进制不存在 (bin/grafana/bin/grafana)"
    echo "  运行: ./scripts/prepare-observability-binaries.sh --download"
    exit 1
  fi

  # Grafana 13+: server subcommand with --homepath and --config.
  # Pre-13:  grafana-server with GF_* env vars.
  if $GRAF_V13; then
    echo "  使用 Grafana 13+ 模式"
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
    GF_SECURITY_ADMIN_USER=admin \
    GF_SECURITY_ADMIN_PASSWORD="${GRAFANA_ADMIN_PASSWORD:-lightai}" \
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
    GF_SECURITY_ADMIN_USER=admin \
    GF_SECURITY_ADMIN_PASSWORD="${GRAFANA_ADMIN_PASSWORD:-lightai}" \
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
  echo "  已启动 (PID $PID)"
	  if $GRAFANA_PASSWORD_PROVIDED; then
	    echo "  Grafana 使用环境变量指定的管理员密码。"
	  else
	    echo "  Grafana 已生成随机管理员密码（首次初始化）。"
	    echo "  凭据已保存至: $CRED_FILE"
	  fi
fi

# --- Wait for readiness ---
echo ""
echo "等待服务就绪..."

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

echo "  Prometheus: $($prom_ok && echo '就绪 (http://127.0.0.1:19090)' || echo '未就绪')"
echo "  Grafana:    $($grafana_ok && echo '就绪 (http://127.0.0.1:13000)' || echo '未就绪')"

if ! $grafana_ok; then
  echo ""
  echo "Grafana 启动失败，请检查:"
  echo "  tail -50 logs/grafana.log"
  echo "  命令: bin/grafana/bin/grafana server --homepath bin/grafana --config configs/observability/grafana.ini"
  rm -f run/grafana.pid
  exit 1
fi

echo ""
echo "Observability 已启动。"
echo ""
echo "=== Prometheus 常用查询 ==="
echo "  up                                     # 所有 target 状态"
echo "  lightai_host_cpu_usage_ratio           # CPU 使用率"
echo "  lightai_host_memory_used_ratio         # 内存使用率"
echo "  lightai_host_filesystem_used_ratio     # 磁盘使用率"
echo "  lightai_gpu_memory_total_bytes         # GPU 显存总量"
echo "  lightai_gpu_memory_used_bytes          # GPU 显存已用"
echo ""
echo "Prometheus 首页显示 "No data queried yet" 是正常状态。"
echo "在上方输入框输入查询表达式后即可看到数据。"

echo "  局域网 Prometheus: http://<server-ip>:19090/"
echo "  局域网 Grafana:    http://<server-ip>:13000/"

if [ -f "$CRED_FILE" ]; then
  echo ""
  echo "  Initial credentials: $CRED_FILE"
  echo "  登录后请立即修改默认密码。"
fi
