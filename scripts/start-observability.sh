#!/bin/sh
# LightAI Go - Start Observability (Prometheus + Grafana, bundled mode)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

mkdir -p data/prometheus data/grafana data/grafana/plugins logs run

PROM_BIN="bin/prometheus"
GRAF_BIN=""
for candidate in bin/grafana/bin/grafana-server bin/grafana/bin/grafana; do
  [ -x "$candidate" ] && { GRAF_BIN="$candidate"; break; }
done

echo "=== LightAI Observability ==="

# --- Prometheus ---
echo ""
echo "[Prometheus]"
if [ -f run/prometheus.pid ]; then
  PID=$(cat run/prometheus.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "  状态: 运行中 (PID $PID)"
  else
    rm -f run/prometheus.pid
    echo "  状态: 已停止 (残留 PID)"
  fi
else
  echo "  状态: 未运行"
fi

if [ ! -f run/prometheus.pid ]; then
  if [ ! -x "$PROM_BIN" ]; then
    echo "  错误: Prometheus 二进制不存在 ($PROM_BIN)"
    echo "  请运行: ./scripts/prepare-observability-binaries.sh --download"
    exit 1
  fi
  nohup "$PROM_BIN" \
    --config.file=configs/observability/prometheus.yml \
    --storage.tsdb.path=data/prometheus \
    --storage.tsdb.retention.time=15d \
    --web.listen-address=0.0.0.0:9090 \
    --web.enable-lifecycle \
    > logs/prometheus.log 2>&1 &
  PID=$!
  echo "$PID" > run/prometheus.pid
  echo "  状态: 已启动 (PID $PID)"
fi

# --- Grafana ---
echo ""
echo "[Grafana]"
if [ -f run/grafana.pid ]; then
  PID=$(cat run/grafana.pid)
  if kill -0 "$PID" 2>/dev/null; then
    echo "  状态: 运行中 (PID $PID)"
  else
    rm -f run/grafana.pid
    echo "  状态: 已停止 (残留 PID)"
  fi
else
  echo "  状态: 未运行"
fi

if [ ! -f run/grafana.pid ]; then
  if [ ! -x "$GRAF_BIN" ]; then
    echo "  错误: Grafana 二进制不存在"
    echo "  请运行: ./scripts/prepare-observability-binaries.sh --download"
    exit 1
  fi
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
  nohup "$GRAF_BIN" \
    > logs/grafana.log 2>&1 &
  PID=$!
  echo "$PID" > run/grafana.pid
  echo "  状态: 已启动 (PID $PID)"
  if [ "${LIGHTAI_GRAFANA_ADMIN_PASSWORD:-lightai}" = "lightai" ]; then
    echo "  注意: 使用默认开发密码 'lightai'。生产环境请设置 LIGHTAI_GRAFANA_ADMIN_PASSWORD。"
  fi
fi

# --- Wait for readiness ---
echo ""
echo "等待服务就绪..."
sleep 2

for i in 1 2 3 4 5; do
  if curl -sf http://127.0.0.1:9090/-/ready >/dev/null 2>&1; then
    echo "  Prometheus: 就绪 (http://127.0.0.1:9090)"
    break
  fi
  [ "$i" = "5" ] && echo "  Prometheus: 未就绪，请检查 logs/prometheus.log"
  sleep 2
done

for i in 1 2 3 4 5; do
  if curl -sf http://127.0.0.1:3000/api/health >/dev/null 2>&1; then
    echo "  Grafana:    就绪 (http://127.0.0.1:3000, admin/lightai)"
    break
  fi
  [ "$i" = "5" ] && echo "  Grafana: 未就绪，请检查 logs/grafana.log"
  sleep 2
done

echo ""
echo "Observability 已启动。"
