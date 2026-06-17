#!/usr/bin/env bash
set -euo pipefail

SERVER_URL="${LIGHTAI_SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-}"
PREFIX="e2e-nvidia"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)}"
COOKIE_JAR="$(mktemp)"
CSRF_TOKEN=""
DEPLOYMENT_ID=""
INSTANCE_ID=""
RUN_PLAN_ID=""
ARTIFACT_ID=""

VLLM_IMAGE="${VLLM_IMAGE:-vllm/vllm-openai:latest}"
VLLM_MODEL="${VLLM_MODEL:-/home/kzeng/models/Qwen3-0.6B-Instruct-2512}"
VLLM_PORT="${VLLM_PORT:-8004}"

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }
skip() { log "SKIP: $*"; exit 0; }
fail() { log "FAIL: $*"; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || skip "$1 is not installed"
}

port_owner() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltnp "sport = :$port" 2>/dev/null | tail -n +2 || true
  elif command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null || true
  fi
}

ensure_services() {
  if curl -fsS "$SERVER_URL/healthz" >/dev/null 2>&1; then
    return 0
  fi

  local server_port
  server_port="$(printf '%s' "$SERVER_URL" | python3 -c 'import sys,urllib.parse as u; p=u.urlparse(sys.stdin.read().strip()); print(p.port or (443 if p.scheme=="https" else 80))')"
  local owner
  owner="$(port_owner "$server_port")"
  if [ -n "$owner" ] && ! printf '%s' "$owner" | grep -qiE 'lightai|go-build|lightai-server'; then
    log "Port $server_port is occupied by a non-LightAI process:"
    printf '%s\n' "$owner"
    skip "LightAI Server is not running and port $server_port is occupied"
  fi

  log "LightAI Server is not running; building local binaries and starting services"
  mkdir -p bin
  go build -o bin/lightai-server ./cmd/server
  go build -o bin/lightai-agent ./cmd/agent
  bash scripts/start-all.sh --no-observability --wait
  curl -fsS "$SERVER_URL/healthz" >/dev/null 2>&1 || fail "LightAI Server failed to start at $SERVER_URL"
}

json_get() {
  python3 -c 'import json,sys
data=json.load(sys.stdin)
path=sys.argv[1].split(".")
for p in path:
    if isinstance(data, list):
        data=data[0] if data else {}
    data=data.get(p, "") if isinstance(data, dict) else ""
print(data if data is not None else "")' "$1"
}

api() {
  local method="$1" path="$2" data="${3:-}"
  local args=(-sS -X "$method" "$SERVER_URL$path" -b "$COOKIE_JAR" -c "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "Content-Type: application/json")
  if [ -n "$CSRF_TOKEN" ] && [ "$method" != "GET" ]; then
    args+=(-H "X-CSRF-Token: $CSRF_TOKEN")
  fi
  if [ -n "$data" ]; then
    args+=(-d "$data")
  fi
  local resp code body
  resp="$(curl "${args[@]}" -w $'\nHTTP:%{http_code}')" || return 1
  code="$(printf '%s\n' "$resp" | awk -F: '/^HTTP:/ {print $2}' | tail -1)"
  body="$(printf '%s\n' "$resp" | sed '/^HTTP:/d')"
  if [ "$code" != "200" ] && [ "$code" != "201" ] && [ "$code" != "202" ]; then
    printf '%s\n' "$body" >&2
    return 1
  fi
  printf '%s\n' "$body"
}

cleanup() {
  rm -f "$COOKIE_JAR"
}
trap cleanup EXIT

cleanup_e2e_resources() {
  if [ -n "$DEPLOYMENT_ID" ]; then
    api POST "/api/v1/deployments/$DEPLOYMENT_ID/stop" '{}' >/tmp/lightai-e2e-stop.json 2>/dev/null || true
    api DELETE "/api/v1/deployments/$DEPLOYMENT_ID" >/tmp/lightai-e2e-delete.json 2>/dev/null || true
  fi
  if [ -n "$ARTIFACT_ID" ]; then
    api DELETE "/api/v1/model-artifacts/$ARTIFACT_ID" >/tmp/lightai-e2e-delete-artifact.json 2>/dev/null || true
  fi
  if [ -n "$INSTANCE_ID" ]; then
    docker rm -f "lightai-${INSTANCE_ID:0:12}" >/dev/null 2>&1 || true
  fi
}

fail_with_diagnostics() {
  local msg="$1"
  log "Diagnostics: $msg"
  if [ -n "$RUN_PLAN_ID" ]; then
    api GET "/api/v1/node-run-plans/$RUN_PLAN_ID/command-preview" >/tmp/lightai-e2e-command-preview.json 2>/dev/null || true
    api GET "/api/v1/node-run-plans/$RUN_PLAN_ID/logs?tail=200" >/tmp/lightai-e2e-node-run-plan-logs.json 2>/dev/null || true
    log "command preview: $(head -c 800 /tmp/lightai-e2e-command-preview.json 2>/dev/null || true)"
    log "logs API tail: $(head -c 1200 /tmp/lightai-e2e-node-run-plan-logs.json 2>/dev/null || true)"
  fi
  if [ -n "$INSTANCE_ID" ]; then
    log "docker logs tail:"
    docker logs --tail=80 "lightai-${INSTANCE_ID:0:12}" 2>&1 || true
  fi
  cleanup_e2e_resources
  fail "$msg"
}

need curl
need python3
need go

ensure_services
command -v docker >/dev/null 2>&1 || skip "docker is not installed"
docker image inspect "$VLLM_IMAGE" >/dev/null 2>&1 || skip "required image missing: $VLLM_IMAGE"
[ -e "$VLLM_MODEL" ] || skip "required model path missing: $VLLM_MODEL"

if [ -z "$PASSWORD" ] && [ -f runtime/initial-credentials.txt ]; then
  PASSWORD="$(awk '/Password:/ {print $NF}' runtime/initial-credentials.txt | tail -1)"
fi
[ -n "$PASSWORD" ] || skip "admin password unavailable; set LIGHTAI_E2E_PASSWORD"

login_resp="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" -H "Origin: $SERVER_URL" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR")"
CSRF_TOKEN="$(printf '%s' "$login_resp" | json_get csrf_token)"
[ -n "$CSRF_TOKEN" ] || fail "login failed"

node_json="$(api GET /api/v1/nodes)"
node_id="$(printf '%s' "$node_json" | json_get id)"
[ -n "$node_id" ] || skip "no registered node found"
log "node_id=$node_id"

gpu_json="$(api GET /api/v1/gpus)"
gpu_id="$(printf '%s' "$gpu_json" | json_get id)"
[ -n "$gpu_id" ] || skip "no GPU found"
log "gpu_id=$gpu_id"

backend_json="$(api GET /api/v1/backends)"
printf '%s' "$backend_json" | grep -q 'backend.vllm' || fail "backend catalog missing backend.vllm"
version_json="$(api GET '/api/v1/backend-versions?backend_id=backend.vllm')"
printf '%s' "$version_json" | grep -q 'backend-version.vllm.openai-latest' || fail "backend version missing vLLM latest"

runtime_id="runtime.vllm.nvidia-docker"
api POST "/api/v1/nodes/$node_id/backend-runtimes/enable" "{\"backend_runtime_id\":\"$runtime_id\",\"image_ref\":\"$VLLM_IMAGE\",\"image_present\":true,\"docker_available\":true}" >/tmp/lightai-e2e-node-runtime.json
grep -q '"status":"ready"' /tmp/lightai-e2e-node-runtime.json || fail "node runtime is not ready: $(cat /tmp/lightai-e2e-node-runtime.json)"

artifact_json="$(api POST /api/v1/model-artifacts "{\"name\":\"$PREFIX-$RUN_ID-vllm-model\",\"display_name\":\"$PREFIX $RUN_ID vLLM model\",\"path\":\"$VLLM_MODEL\",\"format\":\"huggingface\",\"task_type\":\"chat\"}")"
artifact_id="$(printf '%s' "$artifact_json" | json_get id)"
[ -n "$artifact_id" ] || fail "artifact create failed"
ARTIFACT_ID="$artifact_id"
api POST "/api/v1/model-artifacts/$artifact_id/locations" "{\"node_id\":\"$node_id\",\"absolute_path\":\"$VLLM_MODEL\",\"path_type\":\"directory\",\"verification_status\":\"verified\",\"match_status\":\"exact_match\"}" >/dev/null

deployment_payload="$(python3 - <<PY
import json
print(json.dumps({
  "name": "$PREFIX-$RUN_ID-vllm-deployment",
  "model_artifact_id": "$artifact_id",
  "backend_runtime_id": "$runtime_id",
  "placement_json": {"node_id": "$node_id", "gpu_ids": ["$gpu_id"]},
  "service_json": {"host_port": int("$VLLM_PORT")},
  "parameters_json": {"served_model_name": "$PREFIX-$RUN_ID-vllm", "max_model_len": 4096, "gpu_memory_utilization": 0.85},
  "env_overrides_json": {}
}))
PY
)"
deployment_json="$(api POST /api/v1/deployments "$deployment_payload")"
deployment_id="$(printf '%s' "$deployment_json" | json_get id)"
[ -n "$deployment_id" ] || fail "deployment create failed"
DEPLOYMENT_ID="$deployment_id"

start_json="$(api POST "/api/v1/deployments/$deployment_id/start" '{}')"
run_plan_id="$(printf '%s' "$start_json" | json_get run_plan_id)"
instance_id="$(printf '%s' "$start_json" | json_get instance_id)"
[ -n "$run_plan_id" ] || fail "start did not return run_plan_id: $start_json"
RUN_PLAN_ID="$run_plan_id"
INSTANCE_ID="$instance_id"
log "instance_id=$instance_id run_plan_id=$run_plan_id"

api GET "/api/v1/deployments/$deployment_id/run-plan-groups" >/tmp/lightai-e2e-run-plan-groups.json
api GET "/api/v1/node-run-plans/$run_plan_id" >/tmp/lightai-e2e-node-run-plan.json
api GET "/api/v1/node-run-plans/$run_plan_id/command-preview" >/tmp/lightai-e2e-command-preview.json
grep -q 'vllm/vllm-openai' /tmp/lightai-e2e-command-preview.json || fail "command preview missing image"

deadline=$((SECONDS + 240))
models_ok=false
while [ "$SECONDS" -lt "$deadline" ]; do
  if curl -fsS "http://127.0.0.1:$VLLM_PORT/v1/models" >/tmp/lightai-e2e-vllm-models.json 2>/dev/null; then
    log "/v1/models PASS"
    models_ok=true
    break
  fi
  api GET "/api/v1/model-instances?deployment_id=$deployment_id" >/tmp/lightai-e2e-instances.json 2>/dev/null || true
  state="$(python3 -c 'import json,sys; d=json.load(open("/tmp/lightai-e2e-instances.json")); print((d[0] if d else {}).get("actual_state",""))' 2>/dev/null || true)"
  if [ "$state" = "failed" ]; then
    fail_with_diagnostics "instance entered failed state before /v1/models became healthy"
  fi
  sleep 5
done
[ "$models_ok" = true ] || fail_with_diagnostics "/v1/models did not become healthy before timeout"

api GET "/api/v1/node-run-plans/$run_plan_id/logs?tail=200" >/tmp/lightai-e2e-node-run-plan-logs.json || fail_with_diagnostics "logs API failed"
api POST "/api/v1/deployments/$deployment_id/stop" '{}' >/tmp/lightai-e2e-stop.json
api DELETE "/api/v1/deployments/$deployment_id" >/tmp/lightai-e2e-delete.json || fail_with_diagnostics "deployment cleanup failed"
api DELETE "/api/v1/model-artifacts/$ARTIFACT_ID" >/tmp/lightai-e2e-delete-artifact.json || fail_with_diagnostics "model artifact cleanup failed"
docker rm -f "lightai-${INSTANCE_ID:0:12}" >/dev/null 2>&1 || true

log "PASS: backend runtime NVIDIA API E2E completed"
