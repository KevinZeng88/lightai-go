#!/bin/sh
# LightAI Go - Local Verification
# Checks that Server and Agent are running correctly.
set -e

SERVER_URL="${LIGHTAI_SERVER_URL:-http://127.0.0.1:8080}"
AGENT_METRICS_URL="${LIGHTAI_AGENT_METRICS_URL:-http://127.0.0.1:19091/metrics}"

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

echo "=== LightAI Go Verification ==="
echo "Server: $SERVER_URL"
echo "Agent:  $AGENT_METRICS_URL"
echo ""

echo "--- Health ---"
check "server healthz"       curl -sf "$SERVER_URL/healthz"
check "agent metrics reachable" curl -sf "$AGENT_METRICS_URL"

echo ""
echo "--- API ---"
check "GET /api/nodes"       curl -sf "$SERVER_URL/api/nodes"
check "GET /api/gpus"        curl -sf "$SERVER_URL/api/gpus"
check "GET /metrics/targets" curl -sf "$SERVER_URL/metrics/targets"

echo ""
echo "--- Server Metrics ---"
check "lightai_server_nodes_total"  curl -sf "$SERVER_URL/metrics" | grep -q 'lightai_server_nodes_total'
check "lightai_server_gpus_total"   curl -sf "$SERVER_URL/metrics" | grep -q 'lightai_server_gpus_total'

echo ""
echo "--- Agent Metrics ---"
check "lightai_gpu_memory_total_bytes" curl -sf "$AGENT_METRICS_URL" | grep -q 'lightai_gpu_memory_total_bytes'
check "lightai_node_online"           curl -sf "$AGENT_METRICS_URL" | grep -q 'lightai_node_online'

echo ""
echo "--- Web ---"
check "Web index page" curl -sf "$SERVER_URL/" | grep -q '<html'

echo ""
echo "--- Result ---"
echo "Passed: $pass  Failed: $fail"
echo ""
echo "Web Console: $SERVER_URL/"
echo "Default login: admin / <LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD>"
if [ "$fail" -gt 0 ]; then
  echo "Some checks failed. Review logs/ and run scripts/collect-logs.sh"
  exit 1
fi
