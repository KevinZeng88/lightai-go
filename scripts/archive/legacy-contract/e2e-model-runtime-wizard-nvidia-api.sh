#!/usr/bin/env bash
set -euo pipefail
# E2E test for Model/Runtime wizard APIs and preflight on NVIDIA hardware.

SERVER_URL="${LIGHTAI_SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-}"
PREFIX="e2e-wizard"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)}"
COOKIE_JAR="$(mktemp)"
CSRF_TOKEN=""
DEPLOYMENT_ID=""; INSTANCE_ID=""; ARTIFACT_ID=""; RUNTIME_CLONE_ID=""; NBR_ID=""; ROOT_ID=""; ROOT_CREATED="0"; RUN_PLAN_ID=""
EXIT_CODE=0; CURRENT_STAGE=""; STAGE_START_MS=""
ARTIFACT_DIR="${ARTIFACT_DIR:-docs/reports/model-runtime-node-wizard/e2e-vllm-${RUN_ID}}"

VLLM_IMAGE="${VLLM_IMAGE:-vllm/vllm-openai:latest}"
VLLM_MODEL="${VLLM_MODEL:-/home/kzeng/models/Qwen3-0.6B-Instruct-2512}"
VLLM_PORT="${VLLM_PORT:-8004}"

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }
skip() { log "SKIP: $*"; exit 0; }
fail() { log "FAIL: $*"; EXIT_CODE=1; exit 1; }
now_ms() { date +%s%3N; }
stage_start() { CURRENT_STAGE="$1"; log "stage=$1 start"; STAGE_START_MS="$(now_ms)"; }
stage_done() { local s="${1:-$CURRENT_STAGE}"; local d=$(($(now_ms) - STAGE_START_MS)); log "stage=$s done duration_ms=$d"; CURRENT_STAGE=""; }
validate_json_payload() { python3 -m json.tool "$1" >/dev/null 2>&1; }

json_get() {
  python3 -c 'import json,sys
d=json.load(sys.stdin)
for key in sys.argv[1].split("."):
    if isinstance(d, list):
        d=d[0] if d else {}
    if isinstance(d, dict):
        d=d.get(key, "")
    else:
        d=""
print(d if d is not None else "")' "$1"
}

api() {
  local m="$1" p="$2" d="${3:-}"
  local a=(-sS -X "$m" "$SERVER_URL$p" -b "$COOKIE_JAR" -c "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "Content-Type: application/json")
  [ -n "$CSRF_TOKEN" ] && [ "$m" != "GET" ] && a+=(-H "X-CSRF-Token: $CSRF_TOKEN")
  [ -n "$d" ] && a+=(-d "$d")
  local r; r="$(curl "${a[@]}" -w $'\nHTTP:%{http_code}')" || return 1
  local code; code="$(printf '%s\n' "$r" | awk -F: '/^HTTP:/ {print $2}' | tail -1)"
  local body; body="$(printf '%s\n' "$r" | sed '/^HTTP:/d')"
  [ "$code" = "200" ] || [ "$code" = "201" ] || { printf '%s\n' "$body" >&2; return 1; }
  printf '%s\n' "$body"
}

api_expect_fail() {
  local m="$1" p="$2" d="${3:-}"
  local a=(-sS -X "$m" "$SERVER_URL$p" -b "$COOKIE_JAR" -c "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "Content-Type: application/json")
  [ -n "$CSRF_TOKEN" ] && [ "$m" != "GET" ] && a+=(-H "X-CSRF-Token: $CSRF_TOKEN")
  [ -n "$d" ] && a+=(-d "$d")
  local r; r="$(curl "${a[@]}" -w $'\nHTTP:%{http_code}')" || return 1
  local code; code="$(printf '%s\n' "$r" | awk -F: '/^HTTP:/ {print $2}' | tail -1)"
  [ "$code" != "200" ] && [ "$code" != "201" ]
}

json_find_root_id_by_path() {
  python3 - "$1" <<'PY'
import json, sys
target = sys.argv[1]
try:
    data = json.load(sys.stdin)
except Exception:
    data = []
for item in data if isinstance(data, list) else []:
    if item.get("path") == target:
        print(item.get("id", ""))
        break
PY
}

on_exit() {
  local rc=$?; [ "$rc" -ne 0 ] && [ "$EXIT_CODE" -eq 0 ] && EXIT_CODE=$rc
  [ -n "$CURRENT_STAGE" ] && log "failed_stage=$CURRENT_STAGE"
  if [ "${KEEP_E2E_RESOURCES:-0}" != "1" ]; then
    [ -n "$DEPLOYMENT_ID" ] && { api POST "/api/v1/deployments/$DEPLOYMENT_ID/stop" '{}' >/dev/null 2>&1 || true; api DELETE "/api/v1/deployments/$DEPLOYMENT_ID" >/dev/null 2>&1 || true; }
    [ -n "$ARTIFACT_ID" ] && api DELETE "/api/v1/model-artifacts/$ARTIFACT_ID" >/dev/null 2>&1 || true
    [ "$ROOT_CREATED" = "1" ] && [ -n "$ROOT_ID" ] && api DELETE "/api/v1/nodes/$node_id/model-roots/$ROOT_ID" >/dev/null 2>&1 || true
    [ -n "$INSTANCE_ID" ] && docker rm -f "lightai-${INSTANCE_ID:0:12}" >/dev/null 2>&1 || true
  fi
  rm -f "$COOKIE_JAR"
  exit $EXIT_CODE
}
trap on_exit EXIT

need() { command -v "$1" >/dev/null 2>&1 || skip "$1 not installed"; }
need curl; need python3; need go; need docker
mkdir -p "$ARTIFACT_DIR"
docker image inspect "$VLLM_IMAGE" >/dev/null 2>&1 || skip "image missing: $VLLM_IMAGE"
[ -e "$VLLM_MODEL" ] || skip "model path missing: $VLLM_MODEL"

# Start services
if ! curl -fsS "$SERVER_URL/healthz" >/dev/null 2>&1; then
  mkdir -p bin
  go build -o bin/lightai-server ./cmd/server && go build -o bin/lightai-agent ./cmd/agent
  bash scripts/start-all.sh --no-observability --wait
  curl -fsS "$SERVER_URL/healthz" >/dev/null 2>&1 || fail "Server failed to start"
fi

# Login
[ -z "$PASSWORD" ] && [ -f runtime/initial-credentials.txt ] && PASSWORD="$(awk '/Password:/ {print $NF}' runtime/initial-credentials.txt | tail -1)"
[ -n "$PASSWORD" ] || skip "password unavailable"

stage_start login
CSRF_TOKEN="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" -H "Origin: $SERVER_URL" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR" | json_get csrf_token)"
[ -n "$CSRF_TOKEN" ] || fail "login failed"
stage_done

# Query node and GPU
stage_start query_node
node_json="$(api GET /api/v1/nodes)"; node_id="$(printf '%s' "$node_json" | json_get id)"
[ -n "$node_id" ] || fail "no node"
log "node_id=$node_id"
stage_done

stage_start query_gpu
gpu_json="$(api GET /api/v1/gpus)"; gpu_id="$(printf '%s' "$gpu_json" | json_get id)"
[ -n "$gpu_id" ] || skip "no GPU"
log "gpu_id=$gpu_id"
stage_done

# Negative root policy checks
stage_start negative_model_roots
api_expect_fail POST "/api/v1/nodes/$node_id/model-roots" '{"path":"/"}' || fail "adding / model root should fail"
api_expect_fail POST "/api/v1/nodes/$node_id/model-roots" '{"path":"/etc"}' || fail "adding /etc model root should fail"
api_expect_fail POST "/api/v1/nodes/$node_id/model-roots" '{"path":"/etc/lightai"}' || fail "adding /etc/lightai model root should fail"
api_expect_fail POST "/api/v1/nodes/$node_id/model-roots" '{"path":"/tmp/../etc"}' || fail "adding traversal model root should fail"
stage_done

# Add allowed root
stage_start add_model_root
root_path="$(dirname "$VLLM_MODEL")"
set +e
root_json="$(api POST "/api/v1/nodes/$node_id/model-roots" "{\"path\":\"$root_path\",\"description\":\"$PREFIX-$RUN_ID\"}")"
root_create_rc=$?
set -euo pipefail
if [ "$root_create_rc" -eq 0 ]; then
  ROOT_ID="$(printf '%s' "$root_json" | json_get id)"
  ROOT_CREATED="1"
else
  roots_json="$(api GET "/api/v1/nodes/$node_id/model-roots?include_disabled=true")"
  ROOT_ID="$(printf '%s' "$roots_json" | json_find_root_id_by_path "$root_path")"
  [ -n "$ROOT_ID" ] || fail "model root create failed and existing root not found"
  api PATCH "/api/v1/nodes/$node_id/model-roots/$ROOT_ID" "{\"status\":\"enabled\",\"description\":\"$PREFIX-$RUN_ID\"}" >/dev/null
  ROOT_CREATED="0"
fi
[ -n "$ROOT_ID" ] || fail "model root create failed"
log "root_id=$ROOT_ID root_path=$root_path"
stage_done

# Browse files
stage_start browse_files
files_json="$(api GET "/api/v1/nodes/$node_id/files?root_id=$ROOT_ID&path=&limit=50")"
printf '%s' "$files_json" | grep -q "Qwen3" || fail "Qwen3 model not found in /home/kzeng/models"
stage_done

# Scan model
stage_start scan_model
scan_json="$(api POST "/api/v1/nodes/$node_id/model-paths/scan" "{\"root_id\":\"$ROOT_ID\",\"relative_path\":\"$(basename "$VLLM_MODEL")\",\"path_type\":\"directory\"}")"
log "scan result: $(printf '%s' "$scan_json" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("format","?"), d.get("discovered_name","?"))' 2>/dev/null || true)"
stage_done

# Create ModelArtifact + ModelLocation
stage_start create_model_artifact
artifact_json="$(api POST /api/v1/model-artifacts "{\"name\":\"$PREFIX-$RUN_ID-model\",\"display_name\":\"$PREFIX model\",\"path\":\"$VLLM_MODEL\",\"format\":\"huggingface\",\"task_type\":\"chat\"}")"
ARTIFACT_ID="$(printf '%s' "$artifact_json" | json_get id)"
[ -n "$ARTIFACT_ID" ] || fail "artifact create failed"
api POST "/api/v1/model-artifacts/$ARTIFACT_ID/locations" "{\"node_id\":\"$node_id\",\"root_id\":\"$ROOT_ID\",\"relative_path\":\"$(basename "$VLLM_MODEL")\",\"path_type\":\"directory\",\"verification_status\":\"verified\",\"match_status\":\"exact_match\"}" >/dev/null
stage_done

# Query Docker images
stage_start docker_images
images_json="$(api GET "/api/v1/nodes/$node_id/docker-images?query=vllm&limit=5")"
printf '%s' "$images_json" | grep -q "vllm" || fail "vllm image not found"
stage_done

# Enable runtime
stage_start enable_runtime
api POST "/api/v1/nodes/$node_id/backend-runtimes/enable" "{\"backend_runtime_id\":\"runtime.vllm.nvidia-docker\",\"image_ref\":\"$VLLM_IMAGE\",\"image_present\":true,\"docker_available\":true}" >/dev/null
# Agent check to set NBR ready (enable sets needs_check until agent verifies)
api POST "/api/v1/nodes/$node_id/backend-runtimes/$nbr_id/check-request" "{}"\"runtime.vllm.nvidia-docker\",\"image_ref\":\"$VLLM_IMAGE\",\"image_present\":true,\"docker_available\":true}" >/tmp/e2e-wiz-nbr.json
grep -q '"status":"ready"' /tmp/e2e-wiz-nbr.json || fail "runtime not ready after check"
stage_done

# Clone runtime
stage_start clone_runtime
clone_json="$(api POST /api/v1/backend-runtimes/runtime.vllm.nvidia-docker/clone '{}')"
RUNTIME_CLONE_ID="$(printf '%s' "$clone_json" | json_get id)"
[ -n "$RUNTIME_CLONE_ID" ] && log "clone_id=$RUNTIME_CLONE_ID"
stage_done

# Preflight
stage_start preflight
pf_json="$(api POST /api/v1/deployments/preflight "{\"model_artifact_id\":\"$ARTIFACT_ID\",\"backend_runtime_id\":\"runtime.vllm.nvidia-docker\",\"host_port\":$VLLM_PORT}")"
printf '%s' "$pf_json" | grep -q '"can_run":true' || fail "preflight can_run=false"
log "candidate nodes: $(printf '%s' "$pf_json" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(len(d.get("candidate_nodes",[])))' 2>/dev/null || echo 0)"
stage_done

# Create and start deployment
stage_start start_deployment
DEPLOY_PARAMS="${DEPLOY_PARAMS:-}"
if [ -n "$DEPLOY_PARAMS" ]; then
  deploy_payload="{\"name\":\"$PREFIX-$RUN_ID-deploy\",\"model_artifact_id\":\"$ARTIFACT_ID\",\"backend_runtime_id\":\"runtime.vllm.nvidia-docker\",\"placement_json\":{\"node_id\":\"$node_id\",\"accelerator_ids\":[\"$gpu_id\"]},\"service_json\":{\"host_port\":$VLLM_PORT},\"parameters_json\":{$DEPLOY_PARAMS},\"env_overrides_json\":{}}"
else
  deploy_payload="{\"name\":\"$PREFIX-$RUN_ID-deploy\",\"model_artifact_id\":\"$ARTIFACT_ID\",\"backend_runtime_id\":\"runtime.vllm.nvidia-docker\",\"placement_json\":{\"node_id\":\"$node_id\",\"accelerator_ids\":[\"$gpu_id\"]},\"service_json\":{\"host_port\":$VLLM_PORT},\"parameters_json\":{\"served_model_name\":\"$PREFIX-$RUN_ID\",\"max_model_len\":4096},\"env_overrides_json\":{}}"
fi
printf '%s\n' "$deploy_payload" > "$ARTIFACT_DIR/deployment-request-payload.json"
validate_json_payload "$ARTIFACT_DIR/deployment-request-payload.json" || fail "deployment payload is invalid JSON"
deploy_json="$(api POST /api/v1/deployments "$deploy_payload")"
DEPLOYMENT_ID="$(printf '%s' "$deploy_json" | json_get id)"
[ -n "$DEPLOYMENT_ID" ] || fail "deployment create failed"
printf '%s\n' "$deploy_json" > "$ARTIFACT_DIR/deployment-response.json"
[ "${LIGHTAI_E2E_STOP_AFTER_DEPLOYMENT_CREATE:-0}" = "1" ] && { stage_done; log "STOP_AFTER_DEPLOYMENT_CREATE deployment_id=$DEPLOYMENT_ID artifact_dir=$ARTIFACT_DIR"; exit 0; }
start_json="$(api POST "/api/v1/deployments/$DEPLOYMENT_ID/start" '{}')"
INSTANCE_ID="$(printf '%s' "$start_json" | json_get instance_id)"
[ -n "$INSTANCE_ID" ] || fail "start failed"
log "instance_id=$INSTANCE_ID"
stage_done

# Health check
stage_start health_check
deadline=$((SECONDS + 300)); ok=false
while [ "$SECONDS" -lt "$deadline" ]; do
  if curl -fsS "http://127.0.0.1:$VLLM_PORT/v1/models" >/tmp/e2e-wiz-models.json 2>/dev/null; then
    log "/v1/models PASS"; ok=true; break
  fi
  state="$(api GET "/api/v1/model-instances?deployment_id=$DEPLOYMENT_ID" 2>/dev/null | json_get actual_state || true)"
  [ "$state" = "failed" ] && fail "instance failed"
  sleep 5
done
[ "$ok" = true ] || fail "/v1/models timeout"
stage_done

# Docker logs
stage_start logs_api
RUN_PLAN_ID="$(api GET "/api/v1/model-instances?deployment_id=$DEPLOYMENT_ID" | json_get current_run_plan_id)"
[ -n "$RUN_PLAN_ID" ] || fail "node run plan not found"
api GET "/api/v1/node-run-plans/$RUN_PLAN_ID" > "$ARTIFACT_DIR/runplan.json"
api GET "/api/v1/node-run-plans/$RUN_PLAN_ID/logs?tail=50" >/tmp/e2e-wiz-logs.json
stage_done

# Stop
stage_start stop_deployment
api POST "/api/v1/deployments/$DEPLOYMENT_ID/stop" '{}' >/dev/null
stage_done

# Cleanup
stage_start cleanup_resources
api DELETE "/api/v1/deployments/$DEPLOYMENT_ID" >/dev/null
api DELETE "/api/v1/model-artifacts/$ARTIFACT_ID" >/dev/null
[ "$ROOT_CREATED" = "1" ] && api DELETE "/api/v1/nodes/$node_id/model-roots/$ROOT_ID" >/dev/null && ROOT_CREATED="0"
docker rm -f "lightai-${INSTANCE_ID:0:12}" >/dev/null 2>&1 || true
stage_done

log "PASS: model runtime wizard E2E completed"
