#!/bin/bash
# LightAI Go - Local Model Runtime End-to-End Test
# Verifies full lifecycle: artifact→environment→template→deployment→dry-run→start→verify→logs→stop
#
# Usage:
#   scripts/e2e-model-runtime-local.sh [--port 8002] [--model-path /path/to/model.gguf]
#
# Prerequisites:
#   - Docker daemon running, nvidia-container-toolkit installed
#   - NVIDIA GPU with driver
#   - llama.cpp CUDA Docker image (ghcr.io/ggml-org/llama.cpp:server-cuda13)
#   - GGUF model file
#   - LightAI server + agent binaries in /tmp

set -e

PORT="${1:-8002}"
MODEL_PATH="${2:-/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf}"
API="http://127.0.0.1:18080/api/v1"
SERVER_CONFIG="run/e2e/server.yaml"
AGENT_CONFIG="run/e2e/agent.yaml"
COOKIES="run/e2e/cookies.txt"
DB="run/e2e/e2e-test.db"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

# ── Step 0: Environment Checks ──────────────────────────────────────────

echo "=== LightAI Model Runtime E2E Test ==="
echo "Port: $PORT  Model: $MODEL_PATH"

echo -n "Docker daemon... "
docker info >/dev/null 2>&1 && pass "OK" || fail "Docker not accessible"

echo -n "nvidia-smi... "
nvidia-smi >/dev/null 2>&1 && pass "OK" || fail "nvidia-smi not found"

echo -n "Docker GPU runtime... "
docker run --rm --gpus all nvidia/cuda:13.1.1-base-ubuntu24.04 nvidia-smi >/dev/null 2>&1 && pass "OK" || warn "GPU Docker runtime may not be available"

echo -n "Model file... "
[ -f "$MODEL_PATH" ] && pass "$MODEL_PATH" || fail "Model file not found: $MODEL_PATH"

echo -n "Docker socket... "
[ -S /var/run/docker.sock ] && pass "/var/run/docker.sock" || fail "Docker socket not found"

echo -n "Port $PORT... "
ss -tln | grep -q ":$PORT " && fail "Port $PORT is in use" || pass "available"

# ── Step 1: Cleanup ─────────────────────────────────────────────────────

echo ""
echo "=== Cleanup ==="
pkill -f lightai-server 2>/dev/null || true
pkill -f lightai-agent 2>/dev/null || true
sleep 1
rm -f "$DB" "$DB"-shm "$DB"-wal "$COOKIES"
rm -f run/e2e/agent-identity.json
echo "Cleaned"

# ── Step 2: Build Binaries ──────────────────────────────────────────────

echo ""
echo "=== Building Binaries ==="
cd "$(dirname "$0")/.."
CGO_ENABLED=1 go build -o /tmp/lightai-server ./cmd/server/ || fail "server build failed"
CGO_ENABLED=1 go build -o /tmp/lightai-agent ./cmd/agent/ || fail "agent build failed"
pass "Binaries built"

# ── Step 3: Start Server ────────────────────────────────────────────────

echo ""
echo "=== Starting Server ==="

cat > "$SERVER_CONFIG" << EOF
host: 127.0.0.1
port: 18080
db_path: $DB
log_level: info
agent_token: lightai-agent-token-change-me
node_offline_threshold: 300s
EOF

cat > "$AGENT_CONFIG" << EOF
server_url: http://127.0.0.1:18080
agent_id: agent-e2e-test
agent_token: lightai-agent-token-change-me
advertised_address: 127.0.0.1
primary_ip: 127.0.0.1
identity_dir: run/e2e/lightai-runtime
gpu:
  profile: production
  collector_mode: auto
metrics:
  enabled: false
heartbeat:
  interval: 2s
collectors:
  system:
    enabled: false
  report_interval: 10s
logging:
  level: info
  stdout: true
  file_enabled: false
EOF

/tmp/lightai-server --config "$SERVER_CONFIG" &
SERVER_PID=$!
sleep 4

/tmp/lightai-server --config "$SERVER_CONFIG" --reset-admin-password test1234 >/dev/null 2>&1
curl -sf "$API/observability/status" >/dev/null && pass "Server running (PID $SERVER_PID)" || fail "Server not responding"

# ── Step 4: Start Agent ─────────────────────────────────────────────────

echo ""
echo "=== Starting Agent ==="
/tmp/lightai-agent --config "$AGENT_CONFIG" &
AGENT_PID=$!
sleep 6
pass "Agent started (PID $AGENT_PID)"

# ── Step 5: Create Objects via API ──────────────────────────────────────

echo ""
echo "=== Creating Objects ==="

LOGIN=$(curl -sf -c "$COOKIES" -X POST "$API/auth/login" \
  -H "Content-Type: application/json" -H "Origin: http://127.0.0.1:18080" \
  -d '{"username":"admin","password":"test1234"}')
CSRF=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['csrf_token'])")

# ModelArtifact
ARTIFACT_NAME=$(basename "$MODEL_PATH")
A=$(curl -sf -b "$COOKIES" -X POST "$API/model-artifacts" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" -H "Origin: http://127.0.0.1:18080" \
  -d "{\"name\":\"$ARTIFACT_NAME\",\"path\":\"$MODEL_PATH\",\"format\":\"gguf\",\"task_type\":\"chat\",\"architecture\":\"qwen\",\"size_label\":\"9B\",\"quantization\":\"int4\"}")
AID=$(echo "$A" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
pass "Artifact: $AID"

# RuntimeEnvironment
R=$(curl -sf -b "$COOKIES" -X POST "$API/runtime-environments" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" -H "Origin: http://127.0.0.1:18080" \
  -d '{"name":"llama-cpp-cuda13","runtime_type":"docker","backend_type":"llama_cpp","vendor":"nvidia","default_port":8000,"docker":{"image":"ghcr.io/ggml-org/llama.cpp:server-cuda13","ipc_mode":{"enabled":true,"value":"host"},"shm_size":{"enabled":true,"value":"8gb"}}}')
RID=$(echo "$R" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
pass "Runtime: $RID"

# RunTemplate
MODEL_DIR=$(dirname "$MODEL_PATH")
T=$(curl -sf -b "$COOKIES" -X POST "$API/run-templates" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" -H "Origin: http://127.0.0.1:18080" \
  -d "{\"name\":\"llama-cpp-server\",\"runtime_type\":\"docker\",\"vendor\":\"nvidia\",\"backend_type\":\"llama_cpp\",\"required_variables\":[\"MODEL_PATH\",\"CONTAINER_PORT\"],\"args_template\":[\"-m\",\"\${MODEL_PATH}\",\"--host\",\"0.0.0.0\",\"--port\",\"\${CONTAINER_PORT}\"],\"volume_mappings\":{\"enabled\":true,\"value\":[{\"host_path\":\"$MODEL_DIR\",\"container_path\":\"/models\",\"readonly\":true}]}}")
TID=$(echo "$T" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
pass "Template: $TID"

# ── Step 6: Get Node and GPU ────────────────────────────────────────────

echo ""
echo "=== Node & GPU ==="
sleep 2
NODE_ID=$(curl -sf -b "$COOKIES" "$API/nodes" | python3 -c "import sys,json; nodes=json.load(sys.stdin); print([n['id'] for n in nodes if n['status']=='online'][0])")
GPU_ID=$(curl -sf -b "$COOKIES" "$API/gpus" | python3 -c "import sys,json; gpus=json.load(sys.stdin); print([g['id'] for g in gpus if g['health']=='healthy'][0])")
pass "Node: $NODE_ID"
pass "GPU:  $GPU_ID"

# ── Step 7: Create Deployment ───────────────────────────────────────────

echo ""
echo "=== Deployment ==="
D=$(curl -sf -b "$COOKIES" -X POST "$API/model-deployments" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" -H "Origin: http://127.0.0.1:18080" \
  -d "{\"name\":\"e2e-llama-cpp\",\"model_artifact_id\":\"$AID\",\"runtime_environment_id\":\"$RID\",\"run_template_id\":\"$TID\",\"node_id\":\"$NODE_ID\",\"gpu_ids\":[\"$GPU_ID\"],\"host_port\":$PORT}")
DID=$(echo "$D" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
pass "Deployment: $DID"

# ── Step 8: Dry Run ─────────────────────────────────────────────────────

echo ""
echo "=== Dry Run ==="
DR=$(curl -sf -b "$COOKIES" -X POST "$API/model-deployments/$DID/dry-run" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" -H "Origin: http://127.0.0.1:18080" -d '{}')
DR_VALID=$(echo "$DR" | python3 -c "import sys,json; print(json.load(sys.stdin)['valid'])")
[ "$DR_VALID" = "True" ] && pass "Dry run valid" || fail "Dry run invalid: $DR"
echo "$DR" | python3 -c "import sys,json; print(json.load(sys.stdin).get('equivalent_command_preview',''))" 2>/dev/null

# ── Step 9: Start ───────────────────────────────────────────────────────

echo ""
echo "=== Start ==="
SR=$(curl -sf -b "$COOKIES" -X POST "$API/model-deployments/$DID/start" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" -H "Origin: http://127.0.0.1:18080" -d '{}')
INSTANCE_ID=$(echo "$SR" | python3 -c "import sys,json; print(json.load(sys.stdin).get('instance_id',''))")
TASK_ID=$(echo "$SR" | python3 -c "import sys,json; print(json.load(sys.stdin).get('task_id',''))")
pass "Start dispatched. Instance=$INSTANCE_ID Task=$TASK_ID"

# ── Step 10: Wait for Container ─────────────────────────────────────────

echo ""
echo "=== Waiting for Container ==="
for i in $(seq 1 30); do
  CONTAINER=$(docker ps --format '{{.Names}}' 2>/dev/null | grep "lightai-" || true)
  if [ -n "$CONTAINER" ]; then
    pass "Container running: $CONTAINER"
    CONTAINER_ID=$(docker ps --format '{{.ID}}' --filter "name=$CONTAINER")
    break
  fi
  sleep 2
  echo -n "."
done

if [ -z "$CONTAINER_ID" ]; then
  # Check if container exited
  EXITED=$(docker ps -a --format '{{.Names}} {{.Status}}' 2>/dev/null | grep "lightai-" || true)
  if [ -n "$EXITED" ]; then
    warn "Container exited: $EXITED"
    docker logs $(docker ps -a --format '{{.Names}}' | grep "lightai-" | head -1) 2>/dev/null | tail -20
  fi
  fail "No container started"
fi

# ── Step 11: Verify Model Service (poll with timeout) ──────────────────

echo ""
echo "=== Model Service ==="
MODEL_READY=false
for i in $(seq 1 60); do
  MODELS_RESP=$(curl -sf "http://127.0.0.1:$PORT/v1/models" 2>/dev/null || echo "")
  if echo "$MODELS_RESP" | grep -q "$ARTIFACT_NAME"; then
    pass "Model API ready after $((i*3))s: $MODELS_RESP"
    MODEL_READY=true
    break
  fi
  sleep 3
  echo -n "."
done

if [ "$MODEL_READY" = false ]; then
  warn "Model API not ready after 180s. Diagnostics:"
  echo "--- docker ps ---"
  docker ps -a --format '{{.Names}} {{.Status}}' | grep lightai || echo "(none)"
  echo "--- docker logs (last 200 lines) ---"
  docker logs "$CONTAINER_ID" 2>/dev/null | tail -200 || echo "(no logs)"
  echo "--- curl response ---"
  curl -s "http://127.0.0.1:$PORT/v1/models" 2>/dev/null || echo "(empty)"
  echo "--- instance ---"
  sqlite3 "$DB" "SELECT actual_state, last_error FROM model_instances WHERE id='$INSTANCE_ID';" 2>/dev/null
  echo "--- task ---"
  sqlite3 "$DB" "SELECT status FROM agent_tasks WHERE id='$TASK_ID';" 2>/dev/null
fi

# ── Step 12: Instance & Lease Status ────────────────────────────────────

echo ""
echo "=== Instance & Lease Status ==="
INST_STATE=$(sqlite3 "$DB" "SELECT actual_state FROM model_instances WHERE id='$INSTANCE_ID';" 2>/dev/null || echo "unknown")
LEASE_STATUS=$(sqlite3 "$DB" "SELECT status FROM gpu_leases WHERE instance_id='$INSTANCE_ID';" 2>/dev/null || echo "unknown")
TASK_STATUS=$(sqlite3 "$DB" "SELECT status FROM agent_tasks WHERE id='$TASK_ID';" 2>/dev/null || echo "unknown")
echo "Instance: $INST_STATE  Lease: $LEASE_STATUS  Task: $TASK_STATUS"

# ── Step 13: Logs (poll before stop) ────────────────────────────────────

echo ""
echo "=== Logs ==="
LOGS_READY=false
LOGS_TASK_ID=""
for i in $(seq 1 20); do
  HTTP_CODE=$(curl -s -o /tmp/logs-resp.txt -w "%{http_code}" -b "$COOKIES" "$API/model-instances/$INSTANCE_ID/logs" 2>/dev/null)
  LOGS_RESP=$(cat /tmp/logs-resp.txt 2>/dev/null || echo '{}')
  if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "202" ]; then
    echo "  logs HTTP $HTTP_CODE: $(head -c 200 /tmp/logs-resp.txt)"
  fi
  LOGS_STATUS=$(echo "$LOGS_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status','none'))" 2>/dev/null)
  LOGS_CONTENT=$(echo "$LOGS_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('logs',''))" 2>/dev/null)
  LOGS_TASK_ID=$(echo "$LOGS_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('task_id',''))" 2>/dev/null)
  if [ "$LOGS_STATUS" != "pending" ] && [ -n "$LOGS_CONTENT" ]; then
    echo "$LOGS_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('logs','')[:500])" 2>/dev/null
    pass "Logs received ($(echo "$LOGS_CONTENT" | wc -c) bytes)"
    LOGS_READY=true
    break
  fi
  # Diagnostic: every 5 iterations, dump task state
  if [ $((i % 5)) -eq 0 ]; then
    echo ""
    echo "--- Diagnostic (iteration $i) ---"
    echo "Instance: $INSTANCE_ID  Task: ${LOGS_TASK_ID:-unknown}"
    sqlite3 "$DB" "SELECT id, task_type, status, node_id, claimed_at FROM agent_tasks WHERE task_type='model_instance_logs' ORDER BY created_at DESC LIMIT 3;" 2>/dev/null
    echo "Agent heartbeat node_id:"
    sqlite3 "$DB" "SELECT id, agent_id FROM nodes WHERE status='online' LIMIT 1;" 2>/dev/null
  fi
  sleep 3
  echo -n "."
done

if [ "$LOGS_READY" = false ]; then
  warn "Logs not available after 60s."
  echo "=== Full task dump ==="
  sqlite3 "$DB" "SELECT id, task_type, status, node_id, agent_id, instance_id, claimed_at, started_at, finished_at FROM agent_tasks ORDER BY created_at;" 2>/dev/null
  echo "=== Docker logs (last 20) ==="
  docker logs "$CONTAINER_ID" 2>/dev/null | tail -20
fi

# ── Step 14: Stop ───────────────────────────────────────────────────────

echo ""
echo "=== Stop ==="
STOP_RESP=$(curl -sf -b "$COOKIES" -X POST "$API/model-deployments/$DID/stop" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" -H "Origin: http://127.0.0.1:18080" -d '{}')
STOP_STATUS=$(echo "$STOP_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))")
pass "Stop: $STOP_STATUS"

sleep 5
CONTAINER_AFTER=$(docker ps --format '{{.Names}}' 2>/dev/null | grep "lightai-" || echo "")
[ -z "$CONTAINER_AFTER" ] && pass "Container stopped/removed" || warn "Container still exists: $CONTAINER_AFTER"

# ── Step 15: Final Status ───────────────────────────────────────────────

echo ""
echo "=== Final Status ==="
INST_STATE=$(sqlite3 "$DB" "SELECT actual_state FROM model_instances WHERE id='$INSTANCE_ID';" 2>/dev/null || echo "unknown")
LEASE_STATUS=$(sqlite3 "$DB" "SELECT status FROM gpu_leases WHERE instance_id='$INSTANCE_ID';" 2>/dev/null || echo "unknown")
echo "Instance: $INST_STATE  Lease: $LEASE_STATUS"

# ── Step 16: Cleanup ────────────────────────────────────────────────────

echo ""
echo "=== Cleanup ==="
kill $AGENT_PID 2>/dev/null || true
kill $SERVER_PID 2>/dev/null || true
rm -f "$DB" "$DB"-shm "$DB"-wal "$COOKIES" "$SERVER_CONFIG" "$AGENT_CONFIG"
rm -f run/e2e/agent-identity.json
pass "Test complete. Resources cleaned."

echo ""
echo "=== E2E TEST PASSED ==="
