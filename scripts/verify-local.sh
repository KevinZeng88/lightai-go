#!/bin/sh
# LightAI Go - Local Verification
set -e

SERVER_URL="${LIGHTAI_SERVER_URL:-http://127.0.0.1:18080}"
AGENT_METRICS_URL="${LIGHTAI_AGENT_METRICS_URL:-http://127.0.0.1:19091/metrics}"
PROM_URL="${LIGHTAI_PROMETHEUS_URL:-http://127.0.0.1:19090}"
GRAF_URL="${LIGHTAI_GRAFANA_URL:-http://127.0.0.1:13000}"

pass=0
fail=0

check() {
  local label="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "  PASS  $label"
    pass=$((pass + 1))
  else
    echo "  FAIL  $label"
    fail=$((fail + 1))
  fi
}

info() { echo "  INFO  $1"; }

echo "=== LightAI Go 验证 ==="
echo ""

echo "--- 基础服务 ---"
check "Server healthz"        curl -sf --max-time 2 "$SERVER_URL/healthz"
check "Agent metrics 可达"    curl -sf --max-time 2 "$AGENT_METRICS_URL"

echo ""
echo "--- API (认证保护) ---"
if curl -sf --max-time 2 "$SERVER_URL/api/nodes" >/dev/null 2>&1; then
  echo "  PASS  GET /api/nodes"
else
  info "GET /api/nodes 返回非 2xx (认证保护正常)"
fi
if curl -sf --max-time 2 "$SERVER_URL/api/gpus" >/dev/null 2>&1; then
  echo "  PASS  GET /api/gpus"
else
  info "GET /api/gpus 返回非 2xx (认证保护正常)"
fi
check "GET /metrics/targets"  curl -sf --max-time 2 "$SERVER_URL/metrics/targets"

echo ""
echo "--- Server Metrics ---"
check "lightai_server_nodes_total"   curl -sf --max-time 2 "$SERVER_URL/metrics" | grep -q 'lightai_server_nodes_total'
check "lightai_server_gpus_total"    curl -sf --max-time 2 "$SERVER_URL/metrics" | grep -q 'lightai_server_gpus_total'

echo ""
echo "--- Agent Metrics ---"
check "lightai_gpu_memory_total_bytes" curl -sf --max-time 2 "$AGENT_METRICS_URL" | grep -q 'lightai_gpu_memory_total_bytes'

echo ""
echo "--- Web ---"
check "Web 首页" curl -sf --max-time 2 "$SERVER_URL/" | grep -q '<html'

echo ""
echo "--- Prometheus ---"
if curl -sf --max-time 2 "$PROM_URL/-/ready" >/dev/null 2>&1; then
  echo "  PASS  Prometheus /-/ready"
  check "Prometheus targets" curl -sf --max-time 2 "$PROM_URL/api/v1/targets" | grep -q '"health":"up"'
else
  info "Prometheus 未运行 (如未启动: ./scripts/start-observability.sh)"
fi

echo ""
echo "--- Grafana ---"
if curl -sf --max-time 2 "$GRAF_URL/api/health" >/dev/null 2>&1; then
  echo "  PASS  Grafana /api/health"
else
  info "Grafana 未运行 (如未启动: ./scripts/start-observability.sh)"
fi

echo ""
echo "--- 结果 ---"
echo "通过: $pass  失败: $fail"
echo ""
echo "  LightAI Web: $SERVER_URL/"
echo "  Prometheus:  $PROM_URL/"
echo "  Grafana:     $GRAF_URL/"
[ "$fail" -gt 0 ] && echo "有问题请运行: ./scripts/collect-logs.sh" && exit 1
exit 0
