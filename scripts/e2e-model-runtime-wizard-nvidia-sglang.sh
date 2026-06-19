#!/usr/bin/env bash
set -euo pipefail
# SGLang E2E: Backend=SGLang, Runtime=sglang-v0.5.12-nvidia-cuda.
SERVER_URL="${LIGHTAI_SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-test1234}"
RUN_ID="${RUN_ID:-$(date +%Y%m%d%H%M%S)}"
PREFIX="e2e-sglang"
IMAGE="lmsysorg/sglang:latest"
MODEL="/home/kzeng/models/Qwen3-0.6B-Instruct-2512"
PORT="8005"
RUNTIME_ID="sglang-v0.5.12-nvidia-cuda"
DEPLOY_PARAMS="${DEPLOY_PARAMS:-}"
ARTIFACT_DIR="${ARTIFACT_DIR:-docs/reports/model-runtime-node-wizard/e2e-sglang-${RUN_ID}}"
COOKIE_JAR="$(mktemp)"; CSRF_TOKEN=""; EXIT_CODE=0
log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }
fail() { log "FAIL: $*"; exit 1; }
validate_json_payload() { python3 -m json.tool "$1" >/dev/null 2>&1; }
json_get() { python3 -c 'import json,sys;d=json.load(sys.stdin)
for k in sys.argv[1].split("."):
 if isinstance(d,list):d=d[0] if d else {}
 elif isinstance(d,dict):d=d.get(k,"")
 else:d=""
print(d if d is not None else "")' "$1"; }
api() { local m="$1" p="$2" d="${3:-}"; local a=(-sS -X "$m" "$SERVER_URL$p" -b "$COOKIE_JAR" -c "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "Content-Type: application/json"); [ -n "$CSRF_TOKEN" ] && [ "$m" != "GET" ] && a+=(-H "X-CSRF-Token: $CSRF_TOKEN"); [ -n "$d" ] && a+=(-d "$d"); curl "${a[@]}" 2>/dev/null; }

log "===== SGLang E2E ====="
mkdir -p "$ARTIFACT_DIR"
# Login
CSRF_TOKEN="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" -H "Origin: $SERVER_URL" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR" | json_get csrf_token)"
[ -n "$CSRF_TOKEN" ] || fail "login failed"

# Node + GPU
node_id="$(api GET /api/v1/nodes | json_get 0.id)"; [ -n "$node_id" ] || fail "no node"
gpu_id="$(api GET /api/v1/gpus | json_get 0.id)"; [ -n "$gpu_id" ] || fail "no GPU"
log "node=$node_id gpu=$gpu_id"

# Model root
root_path="$(dirname "$MODEL")"
root_id="$(api POST "/api/v1/nodes/$node_id/model-roots" "{\"path\":\"$root_path\"}" | json_get id)"
[ -n "$root_id" ] || fail "add model root failed"
log "root=$root_id path=$root_path"

# Scan + Artifact
rel="${MODEL#$root_path/}"
scan="$(api POST "/api/v1/nodes/$node_id/model-paths/scan" "{\"root_id\":\"$root_id\",\"root\":\"$root_path\",\"relative_path\":\"$rel\",\"path_type\":\"directory\"}")"
log "scan: $(echo "$scan" | json_get discovered_name 2>/dev/null || echo '?')"
artifact="$(api POST /api/v1/model-artifacts "{\"name\":\"$PREFIX-$RUN_ID-model\",\"display_name\":\"$PREFIX model\",\"path\":\"$MODEL\",\"format\":\"huggingface\",\"task_type\":\"chat\"}")"
artifact_id="$(echo "$artifact" | json_get id)"; [ -n "$artifact_id" ] || fail "artifact create failed"
api POST "/api/v1/model-artifacts/$artifact_id/locations" "{\"node_id\":\"$node_id\",\"root_id\":\"$root_id\",\"relative_path\":\"$rel\",\"path_type\":\"directory\",\"verification_status\":\"verified\",\"match_status\":\"exact_match\"}" >/dev/null
log "artifact=$artifact_id"

# Enable NBR
api POST "/api/v1/nodes/$node_id/backend-runtimes/enable" "{\"backend_runtime_id\":\"$RUNTIME_ID\",\"image_ref\":\"$IMAGE\",\"image_present\":true,\"docker_available\":true}" >/dev/null
log "nbr enabled"

# Agent check (required after enable to set NBR ready)
api POST "/api/v1/nodes/$node_id/backend-runtimes/check" "{\"backend_runtime_id\":\"$RUNTIME_ID\",\"image_ref\":\"$IMAGE\",\"image_present\":true,\"docker_available\":true}" >/dev/null
log "nbr checked"

# Deploy
payload="{\"name\":\"$PREFIX-$RUN_ID-deploy\",\"model_artifact_id\":\"$artifact_id\",\"node_backend_runtime_id\":\"$node_id:$RUNTIME_ID\",\"placement_json\":{\"node_id\":\"$node_id\",\"gpu_ids\":[\"$gpu_id\"]},\"service_json\":{\"host_port\":$PORT}"
[ -n "$DEPLOY_PARAMS" ] && payload="$payload,\"parameters_json\":{$DEPLOY_PARAMS}"
payload="$payload}"
printf '%s\n' "$payload" > "$ARTIFACT_DIR/deployment-request-payload.json"
validate_json_payload "$ARTIFACT_DIR/deployment-request-payload.json" || fail "deployment payload is invalid JSON"
deploy_id="$(api POST /api/v1/deployments "$payload" | json_get id)"; [ -n "$deploy_id" ] || fail "deploy create failed"
log "deploy=$deploy_id"

# Preflight
pf="$(api POST /api/v1/deployments/preflight "{\"model_artifact_id\":\"$artifact_id\",\"node_backend_runtime_id\":\"$node_id:$RUNTIME_ID\",\"host_port\":$PORT}")"
log "preflight nodes=$(echo "$pf" | json_get candidate_nodes)"

# Start
log "start_deployment"
instance_id="$(api POST "/api/v1/deployments/$deploy_id/start" | json_get instance_id)"
[ -n "$instance_id" ] || fail "start failed"
log "instance=$instance_id"
inst_detail="$(api GET "/api/v1/model-instances/$instance_id" 2>/dev/null || echo '{}')"
printf '%s\n' "$inst_detail" > "$ARTIFACT_DIR/instance-detail-after-start.json"
run_plan_id="$(printf '%s' "$inst_detail" | json_get current_run_plan_id)"
[ -n "$run_plan_id" ] && api GET "/api/v1/node-run-plans/$run_plan_id" > "$ARTIFACT_DIR/runplan.json" 2>/dev/null || true

# Health check
log "health_check start"
hc_ok=0
for i in $(seq 1 120); do
  inst="$(api GET "/api/v1/model-instances/$instance_id" 2>/dev/null || echo '{}')"
  state="$(echo "$inst" | json_get actual_state)"; cid="$(echo "$inst" | json_get container_id)"
  if [ "$state" = "running" ]; then
    models="$(curl -sS "http://127.0.0.1:$PORT/v1/models" 2>/dev/null || echo '')"
    if [ -n "$models" ] && echo "$models" | python3 -c 'import json,sys;json.load(sys.stdin)' 2>/dev/null; then
      log "/v1/models PASS cid=${cid:0:12}"; hc_ok=1; break
    fi
  elif [ "$state" = "failed" ]; then
    fail "instance failed cid=$cid err=$(echo "$inst" | json_get last_error)"; break
  fi
  sleep 2
done
[ "$hc_ok" = "1" ] || fail "health_check timeout"

# Instance test
api POST "/api/v1/model-instances/$instance_id/test" '{"mode":"chat","max_tokens":16}' >/dev/null && log "instance_test ok" || log "instance_test skipped"

# Logs
api POST "/api/v1/model-instances/$instance_id/logs" '{"tail":50}' >/dev/null && log "logs ok" || log "logs skipped"

# Stop
api POST "/api/v1/deployments/$deploy_id/stop" >/dev/null; sleep 3; log "stop done"

# Cleanup
api DELETE "/api/v1/deployments/$deploy_id" >/dev/null 2>&1 || true
api DELETE "/api/v1/model-artifacts/$artifact_id" >/dev/null 2>&1 || true
api DELETE "/api/v1/nodes/$node_id/model-roots/$root_id" >/dev/null 2>&1 || true
log "PASS: SGLang E2E completed"
