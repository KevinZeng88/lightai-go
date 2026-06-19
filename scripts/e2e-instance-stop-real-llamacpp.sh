#!/bin/bash
# e2e-instance-stop-real-llamacpp.sh — Real container start/stop E2E with llama.cpp + GGUF.
# Category: Real container E2E
# Starts a llama.cpp container, verifies instance state, stops it, verifies cleanup.
# Requires: Docker daemon, NVIDIA GPU, llama.cpp image + GGUF model available.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/e2e-assert.sh"

SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-Commvault!234}"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-/tmp/lightai-e2e-instance-stop-$RUN_ID}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/lightai-e2e-cookies-$RUN_ID.txt}"
PREFIX="e2e-instance-stop"
mkdir -p "$ARTIFACT_DIR"

log()   { printf '[%s] [instance-stop] %s\n' "$(date '+%H:%M:%S')" "$*"; }

api_get() {
  curl -sS -b "$COOKIE_JAR" -H "Origin: $SERVER_URL" -X GET "$SERVER_URL/api/v1/$1"
}
api_post() {
  local a=(-sS -b "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "Content-Type: application/json")
  [ -n "${CSRF_TOKEN:-}" ] && a+=(-H "X-CSRF-Token: $CSRF_TOKEN")
  curl "${a[@]}" -X POST -d "$2" "$SERVER_URL/api/v1/$1"
}
api_delete() {
  local a=(-sS -b "$COOKIE_JAR" -H "Origin: $SERVER_URL")
  [ -n "${CSRF_TOKEN:-}" ] && a+=(-H "X-CSRF-Token: $CSRF_TOKEN")
  curl "${a[@]}" -X DELETE "$SERVER_URL/api/v1/$1"
}

json_field() { python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('$1',''))" 2>/dev/null; }

# ── login ──
log "Logging in..."
resp="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" \
  -H "Origin: $SERVER_URL" -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR")"
CSRF_TOKEN="$(echo "$resp" | json_field csrf_token)"
[ -n "$CSRF_TOKEN" ] || { log "FATAL: Login failed"; exit 1; }
log "Logged in"

# ── discover resources ──
NODE_ID=$(api_get "nodes" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)
[ -n "$NODE_ID" ] || { log "FATAL: No online node"; exit 1; }
log "Node: $NODE_ID"

# Find GGUF artifact
ARTS_JSON=$(api_get "model-artifacts")
GGUF_ART_ID=$(echo "$ARTS_JSON" | python3 -c "
import json,sys
for a in json.load(sys.stdin):
    if a.get('format') == 'gguf':
        print(a['id']); break
" 2>/dev/null)
[ -n "$GGUF_ART_ID" ] || { log "FATAL: No GGUF artifact available"; exit 1; }
log "GGUF artifact: $GGUF_ART_ID"

# Find llama.cpp runtime
LLAMACPP_RT=$(api_get "backend-runtimes" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    rid = r.get('id','')
    if 'llamacpp' in rid and r.get('vendor','')=='nvidia':
        print(rid); break
" 2>/dev/null)
[ -n "$LLAMACPP_RT" ] || { log "FATAL: No llama.cpp NVIDIA runtime"; exit 1; }
log "llama.cpp runtime: $LLAMACPP_RT"

# ── ensure NBR is ready ──
NBR_ID="${NODE_ID}:${LLAMACPP_RT}"
NBR_STATUS=$(api_get "nodes/$NODE_ID/backend-runtimes" | python3 -c "
import json,sys
for n in json.load(sys.stdin):
    if n.get('backend_runtime_id') == '$LLAMACPP_RT':
        print(n.get('status',''))
        sys.exit(0)
" 2>/dev/null)
if [ "$NBR_STATUS" != "ready" ]; then
  log "Enabling llama.cpp NBR (status=$NBR_STATUS)..."
  api_post "nodes/$NODE_ID/backend-runtimes/enable" \
    "{\"backend_runtime_id\":\"$LLAMACPP_RT\",\"image_present\":true,\"docker_available\":true}" > /dev/null
fi

# ── pre-state ──
PRE_CONTAINERS=$(docker ps -q 2>/dev/null | wc -l)
PRE_INSTANCES=$(api_get "model-instances" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
log "Pre state: containers=$PRE_CONTAINERS instances=$PRE_INSTANCES"

# ═══════════════════════════════════════════════════════════════
# Test: Create → Start → Verify → Stop → Verify cleanup
# ═══════════════════════════════════════════════════════════════

# Step 1: Create deployment
log "=== Step 1: Create deployment ==="
DEP_NAME="${PREFIX}-llamacpp-gguf"
CREATE_RESP=$(api_post "deployments" "{\"name\":\"$DEP_NAME\",\"display_name\":\"Instance Stop Test\",\"model_artifact_id\":\"$GGUF_ART_ID\",\"backend_runtime_id\":\"$LLAMACPP_RT\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8391,\"container_port\":8080,\"app_port\":8080},\"parameters_json\":{}}")
echo "$CREATE_RESP" > "$ARTIFACT_DIR/create-response.json"
DEP_ID=$(echo "$CREATE_RESP" | json_field id)
[ -n "$DEP_ID" ] || { log "FATAL: Deployment create failed"; exit 1; }
log "Created deployment: $DEP_ID"

# Step 2: Start deployment
log "=== Step 2: Start deployment ==="
START_RESP=$(api_post "deployments/$DEP_ID/start" '{}')
echo "$START_RESP" > "$ARTIFACT_DIR/start-response.json"
log "Start response: $(head -c 200 "$ARTIFACT_DIR/start-response.json")"

# Step 3: Wait for instance to be running (poll up to 60s)
log "=== Step 3: Wait for instance running ==="
INSTANCE_ID=""
CONTAINER_ID=""
INSTANCE_STATE=""
for i in $(seq 1 30); do
  sleep 2
  INSTANCES=$(api_get "model-instances")
  echo "$INSTANCES" > "$ARTIFACT_DIR/instances-poll-$i.json"
  INSTANCE_INFO=$(echo "$INSTANCES" | python3 -c "
import json,sys
for inst in json.load(sys.stdin):
    if inst.get('deployment_id') == '$DEP_ID':
        print(json.dumps({
            'id': inst.get('id',''),
            'actual_state': inst.get('actual_state',''),
            'container_id': inst.get('container_id',''),
            'host_port': inst.get('host_port',''),
            'endpoint_url': inst.get('endpoint_url',''),
            'last_error': inst.get('last_error','')
        }))
        break
" 2>/dev/null)
  if [ -n "$INSTANCE_INFO" ]; then
    INSTANCE_ID=$(echo "$INSTANCE_INFO" | json_field id)
    INSTANCE_STATE=$(echo "$INSTANCE_INFO" | json_field actual_state)
    CONTAINER_ID=$(echo "$INSTANCE_INFO" | python3 -c "import json,sys; d=json.load(sys.stdin); c=d.get('container_id',''); print(c if c else '')" 2>/dev/null)
    LAST_ERROR=$(echo "$INSTANCE_INFO" | json_field last_error)
    log "  poll $i: state=$INSTANCE_STATE container=${CONTAINER_ID:0:12}... error=$LAST_ERROR"
    if [ "$INSTANCE_STATE" = "running" ]; then
      break
    fi
    if [ "$INSTANCE_STATE" = "failed" ] || [ "$INSTANCE_STATE" = "error" ]; then
      log "Instance failed: $INSTANCE_INFO"
      break
    fi
  else
    log "  poll $i: no instance yet"
  fi
done
echo "$INSTANCE_INFO" > "$ARTIFACT_DIR/instance-info.json"

# Step 4: Verify instance running
log "=== Step 4: Verify instance state ==="
assert_nonempty "instance ID assigned" "$INSTANCE_ID" || log "FAIL: no instance"
assert_eq "instance state is running" "running" "$INSTANCE_STATE" || log "FAIL: not running (state=$INSTANCE_STATE)"
assert_nonempty "container ID assigned" "$CONTAINER_ID" || log "FAIL: no container"

# Verify container actually exists in Docker
if [ -n "$CONTAINER_ID" ]; then
  DOCKER_RUNNING=$(docker ps --filter "id=$CONTAINER_ID" -q 2>/dev/null)
  assert_nonempty "Docker container running" "$DOCKER_RUNNING" || log "FAIL: container not in docker ps"
fi

# Check deployment status
DEP_DETAIL=$(api_get "deployments/$DEP_ID")
echo "$DEP_DETAIL" > "$ARTIFACT_DIR/deployment-detail-running.json"
DEP_STATUS=$(echo "$DEP_DETAIL" | json_field status)
assert_contains "deployment status indicates running" "$DEP_STATUS" "running" || log "deployment status=$DEP_STATUS"

# Step 5: Stop deployment
log "=== Step 5: Stop deployment ==="
STOP_RESP=$(api_post "deployments/$DEP_ID/stop" '{}')
echo "$STOP_RESP" > "$ARTIFACT_DIR/stop-response.json"
log "Stop response: $(head -c 200 "$ARTIFACT_DIR/stop-response.json")"

# Step 6: Wait for instance to stop (poll up to 30s)
log "=== Step 6: Wait for instance stop ==="
STOPPED_STATE=""
for i in $(seq 1 15); do
  sleep 2
  INST_DETAIL=$(api_get "model-instances/${INSTANCE_ID}" 2>/dev/null || echo '{}')
  echo "$INST_DETAIL" > "$ARTIFACT_DIR/instance-detail-poll-$i.json"
  STOPPED_STATE=$(echo "$INST_DETAIL" | json_field actual_state)
  log "  poll $i: state=$STOPPED_STATE"
  if [ "$STOPPED_STATE" = "stopped" ]; then
    break
  fi
done

# Step 7: Verify instance stopped
log "=== Step 7: Verify instance stopped ==="
assert_eq "instance state is stopped" "stopped" "$STOPPED_STATE" || log "FAIL: not stopped (state=$STOPPED_STATE)"

# Verify container stopped (not running). Agent cleanup of exited containers is async.
if [ -n "$CONTAINER_ID" ]; then
  sleep 3  # grace period for Docker stop
  CONTAINER_RUNNING=$(docker ps --filter "id=$CONTAINER_ID" -q 2>/dev/null)
  assert_empty "Docker container not running after stop" "$CONTAINER_RUNNING" || log "FAIL: container still running"
fi

# Check deployment status after stop
DEP_DETAIL2=$(api_get "deployments/$DEP_ID")
echo "$DEP_DETAIL2" > "$ARTIFACT_DIR/deployment-detail-stopped.json"
DEP_STATUS2=$(echo "$DEP_DETAIL2" | json_field status)
assert_eq "deployment status after stop" "stopped" "$DEP_STATUS2" || log "deployment status=$DEP_STATUS2"

# Step 8: Cleanup deployment
log "=== Step 8: Cleanup ==="
api_delete "deployments/$DEP_ID" > "$ARTIFACT_DIR/delete-response.json" 2>/dev/null || true
log "Deployment deleted"

# Verify no leftover containers
POST_CONTAINERS=$(docker ps -q 2>/dev/null | wc -l)
assert_eq "container count back to baseline" "$PRE_CONTAINERS" "$POST_CONTAINERS" || log "FAIL: container leak (pre=$PRE_CONTAINERS post=$POST_CONTAINERS)"

# ── summary ──
echo ""
echo "Artifacts: $ARTIFACT_DIR"
echo "Key files:"
echo "  create-response.json start-response.json instance-info.json"
echo "  deployment-detail-running.json stop-response.json deployment-detail-stopped.json"
assert_summary
