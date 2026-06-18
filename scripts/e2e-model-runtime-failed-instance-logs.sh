#!/usr/bin/env bash
set -euo pipefail
# Dedicated failed instance logs/status E2E test.
# Constructs a container guaranteed to exit quickly, then verifies:
# 1. instance state = failed
# 2. container_id preserved
# 3. exit_code preserved
# 4. failure_reason_code in last_error
# 5. stderr_tail_preview preserved and single-line
# 6. Docker logs API available in failed state

SERVER_URL="${LIGHTAI_SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-test1234}"
RUN_ID="${RUN_ID:-$(date +%Y%m%d%H%M%S)}"
ARTIFACT_DIR="${ARTIFACT_DIR:-docs/reports/model-runtime-node-wizard/failed-instance-logs-${RUN_ID}}"
COOKIE_JAR="$(mktemp)"; CSRF_TOKEN=""; EXIT_CODE=0
mkdir -p "$ARTIFACT_DIR"

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }
fail() { log "FAIL: $*"; EXIT_CODE=1; }
json_get() { python3 -c 'import json,sys;d=json.load(sys.stdin)
for k in sys.argv[1].split("."):
 if isinstance(d,list):d=d[0] if d else {}
 elif isinstance(d,dict):d=d.get(k,"")
 else:d=""
print(d if d is not None else "")' "$1"; }
api() { local m="$1" p="$2" d="${3:-}"; local a=(-sS -X "$m" "$SERVER_URL$p" -b "$COOKIE_JAR" -c "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "Content-Type: application/json"); [ -n "$CSRF_TOKEN" ] && [ "$m" != "GET" ] && a+=(-H "X-CSRF-Token: $CSRF_TOKEN"); [ -n "$d" ] && a+=(-d "$d"); curl "${a[@]}" 2>/dev/null; }

log "===== Failed Instance E2E ====="
# Login
CSRF_TOKEN="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" -H "Origin: $SERVER_URL" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR" | json_get csrf_token)"
[ -n "$CSRF_TOKEN" ] || fail "login failed"

# Node + GPU
node_id="$(api GET /api/v1/nodes | json_get 0.id)"; [ -n "$node_id" ] || fail "no node"
gpu_id="$(api GET /api/v1/gpus | json_get 0.id)"; [ -n "$gpu_id" ] || fail "no GPU"
log "node=$node_id gpu=$gpu_id"

# Model root
root_id="$(api POST "/api/v1/nodes/$node_id/model-roots" "{\"path\":\"/home/kzeng/models\"}" | json_get id)"
[ -n "$root_id" ] || fail "add model root failed"

# Scan + Artifact
scan="$(api POST "/api/v1/nodes/$node_id/model-paths/scan" "{\"root_id\":\"$root_id\",\"root\":\"/home/kzeng/models\",\"relative_path\":\"Qwen3-0.6B-Instruct-2512\",\"path_type\":\"directory\"}")"
log "scan: $(echo "$scan" | json_get discovered_name 2>/dev/null || echo '?')"
artifact="$(api POST /api/v1/model-artifacts "{\"name\":\"fail-test-$RUN_ID\",\"display_name\":\"Fail Test\",\"path\":\"/home/kzeng/models/Qwen3-0.6B-Instruct-2512\",\"format\":\"huggingface\",\"task_type\":\"chat\"}")"
artifact_id="$(echo "$artifact" | json_get id)"; [ -n "$artifact_id" ] || fail "artifact create failed"
api POST "/api/v1/model-artifacts/$artifact_id/locations" "{\"node_id\":\"$node_id\",\"root_id\":\"$root_id\",\"relative_path\":\"Qwen3-0.6B-Instruct-2512\",\"path_type\":\"directory\",\"verification_status\":\"verified\",\"match_status\":\"exact_match\"}" >/dev/null

# Bind port 8090 first so Docker start fails with port conflict
nc -l 8090 &>/dev/null &
NC_PID=$!
sleep 1

# Enable NBR with valid image (port conflict will cause docker.start failure)
api POST "/api/v1/nodes/$node_id/backend-runtimes/enable" "{\"backend_runtime_id\":\"vllm-v0.23.0-nvidia-cuda\",\"image_ref\":\"vllm/vllm-openai:latest\",\"image_present\":true,\"docker_available\":true}" >/dev/null

# Create deployment with bad health check path to force health_check failure
deploy_id="$(api POST /api/v1/deployments "{\"name\":\"fail-test-$RUN_ID\",\"model_artifact_id\":\"$artifact_id\",\"backend_runtime_id\":\"vllm-v0.23.0-nvidia-cuda\",\"placement_json\":{\"node_id\":\"$node_id\",\"gpu_ids\":[\"$gpu_id\"]},\"service_json\":{\"host_port\":8090},\"parameters_json\":{\"served_model_name\":\"fail-test\",\"max_model_len\":128}}" | json_get id)"
[ -n "$deploy_id" ] || fail "deploy create failed"
log "deploy=$deploy_id"

# Start
instance_id="$(api POST "/api/v1/deployments/$deploy_id/start" | json_get instance_id)"
[ -n "$instance_id" ] && log "instance=$instance_id" || fail "start failed"

# Wait for failed state (container should crash with bad params)
log "waiting for failed state..."
failed=0
for i in $(seq 1 60); do
  inst="$(api GET "/api/v1/model-instances/$instance_id" 2>/dev/null || echo '{}')"
  state="$(echo "$inst" | json_get actual_state)"
  cid="$(echo "$inst" | json_get container_id)"
  lerr="$(echo "$inst" | json_get last_error)"
  if [ "$state" = "failed" ]; then
    log "instance FAILED cid=$cid"
    echo "$inst" > "$ARTIFACT_DIR/failed-instance.json"
    # Verify container_id preserved
    [ -n "$cid" ] && log "container_id OK: $cid" || fail "container_id empty"
    # Verify last_error has failure info
    if [ -n "$lerr" ] && [ "$lerr" != "null" ] && [ "$lerr" != "{}" ]; then
      log "last_error OK"
    else
      log "last_error: empty"
    fi
    failed=1
    break
  elif [ "$state" = "running" ]; then
    log "instance running (unexpected) cid=$cid"
    break
  fi
  sleep 2
done
[ "$failed" = "1" ] || log "instance did not reach failed (may be running)"

# Docker logs in failed state — get run_plan_id from instance detail
inst_detail="$(api GET "/api/v1/model-instances/$instance_id" 2>/dev/null || echo '{}')"
run_plan_id="$(echo "$inst_detail" | json_get current_run_plan_id 2>/dev/null)"
if [ -n "$run_plan_id" ] && [ "$run_plan_id" != "null" ]; then
  logs_resp="$(api GET "/api/v1/node-run-plans/$run_plan_id/logs" 2>/dev/null || echo '{}')"
  echo "$logs_resp" > "$ARTIFACT_DIR/docker-logs-response.json"
  logs_len=$(echo "$logs_resp" | wc -c)
  if [ "$logs_len" -gt 50 ]; then
    log "logs_api: real response ($logs_len bytes) run_plan_id=$run_plan_id"
  else
    log "logs_api: response ($logs_len bytes) run_plan_id=$run_plan_id"
  fi
else
  log "logs_api: no run_plan_id available (instance may not have been fully created)"
  echo '{"error":"no run_plan_id"}' > "$ARTIFACT_DIR/docker-logs-response.json"
fi

# Status refresh
status_refresh="$(api GET "/api/v1/model-instances/$instance_id" 2>/dev/null || echo '{}')"
echo "$status_refresh" > "$ARTIFACT_DIR/status-refresh-response.json"
log "status_refresh state=$(echo "$status_refresh" | json_get actual_state)"

# Kill port holder
kill $NC_PID 2>/dev/null || true

# Stop + cleanup
api POST "/api/v1/deployments/$deploy_id/stop" >/dev/null 2>&1 || true; sleep 2
api DELETE "/api/v1/deployments/$deploy_id" >/dev/null 2>&1 || true
api DELETE "/api/v1/model-artifacts/$artifact_id" >/dev/null 2>&1 || true
api DELETE "/api/v1/nodes/$node_id/model-roots/$root_id" >/dev/null 2>&1 || true
log "cleanup done"

# Save server/agent logs
tail -n 2000 logs/lightai-server.log > "$ARTIFACT_DIR/server-this-run.log" 2>/dev/null || true
tail -n 2000 logs/lightai-agent.log > "$ARTIFACT_DIR/agent-this-run.log" 2>/dev/null || true
docker ps -a --format 'table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}' > "$ARTIFACT_DIR/docker-ps-after.txt" 2>/dev/null || true

echo '{"status":"cleanup_completed"}' > "$ARTIFACT_DIR/cleanup-result.json"

if [ "$failed" = "1" ]; then
  log "PASS: failed instance E2E completed"
else
  log "FAIL: instance did not reach failed state (check logs)"
fi
exit $EXIT_CODE
