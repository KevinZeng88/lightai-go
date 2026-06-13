#!/bin/sh
# LightAI Go - Start Observability (Prometheus + Grafana, bundled mode)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

mkdir -p data/prometheus data/grafana data/grafana/plugins logs run

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
    --web.listen-address=0.0.0.0:9090 \
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

  # Grafana 13+: --homepath is GLOBAL flag (before 'server').
  # Pre-13:  grafana-server with GF_* env vars.
  if $GRAF_V13; then
    echo "  使用 Grafana 13+ 模式"
    GF_SECURITY_ADMIN_USER=admin \
    GF_SECURITY_ADMIN_PASSWORD="${LIGHTAI_GRAFANA_ADMIN_PASSWORD:-lightai}" \
    nohup "$GRAF_BIN" \
      --homepath "$RLS_ROOT/bin/grafana" \
      server \
      --config "$RLS_ROOT/configs/observability/grafana.ini" \
      > logs/grafana.log 2>&1 &
  else
    GF_PATHS_CONFIG=configs/observability/grafana.ini \
    GF_PATHS_DATA=data/grafana \
    GF_PATHS_LOGS=logs \
    GF_PATHS_PLUGINS=data/grafana/plugins \
    GF_PATHS_PROVISIONING=deploy/observability/grafana/provisioning \
    GF_SECURITY_ADMIN_USER=admin \
    GF_SECURITY_ADMIN_PASSWORD="${LIGHTAI_GRAFANA_ADMIN_PASSWORD:-lightai}" \
    GF_SERVER_HTTP_ADDR=0.0.0.0 \
    GF_SERVER_HTTP_PORT=3000 \
    GF_DATABASE_TYPE=sqlite3 \
    GF_DATABASE_PATH=data/grafana/grafana.db \
    GF_ANALYTICS_REPORTING_ENABLED=false \
    GF_ANALYTICS_CHECK_FOR_UPDATES=false \
    nohup "$GRAF_BIN" > logs/grafana.log 2>&1 &
  fi
  PID=$!
  echo "$PID" > run/grafana.pid
  echo "  已启动 (PID $PID)"
  if [ "${LIGHTAI_GRAFANA_ADMIN_PASSWORD:-lightai}" = "lightai" ]; then
    echo "  注意: 使用默认密码 'lightai'。生产请设置 LIGHTAI_GRAFANA_ADMIN_PASSWORD。"
  fi
fi

# --- Wait for readiness ---
echo ""
echo "等待服务就绪..."

grafana_ok=false
for i in 1 2 3 4 5 6 7 8; do
  if curl -sf http://127.0.0.1:3000/api/health >/dev/null 2>&1; then
    grafana_ok=true
    break
  fi
  sleep 3
done

prom_ok=false
for i in 1 2 3; do
  if curl -sf http://127.0.0.1:9090/-/ready >/dev/null 2>&1; then
    prom_ok=true
    break
  fi
  sleep 2
done

echo "  Prometheus: $($prom_ok && echo '就绪 (http://127.0.0.1:9090)' || echo '未就绪')"
echo "  Grafana:    $($grafana_ok && echo '就绪 (http://127.0.0.1:3000)' || echo '未就绪')"

if ! $grafana_ok; then
  echo ""
  echo "Grafana 启动失败，请检查:"
  echo "  tail -50 logs/grafana.log"
  echo "  命令: bin/grafana/bin/grafana --homepath bin/grafana server --config configs/observability/grafana.ini"
  rm -f run/grafana.pid
  exit 1
fi

echo ""
echo "Observability 已启动。"
echo "  局域网 Prometheus: http://<server-ip>:9090/"
echo "  局域网 Grafana:    http://<server-ip>:3000/"
