#!/usr/bin/env bash
set -euo pipefail

SERVER_URL="${LIGHTAI_SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-}"
PREFIX="e2e-nvidia"
COOKIE_JAR="$(mktemp)"
CSRF_TOKEN=""

VLLM_IMAGE="${VLLM_IMAGE:-vllm/vllm-openai:latest}"
VLLM_MODEL="${VLLM_MODEL:-/home/kzeng/models/Qwen3-0.6B-Instruct-2512}"
VLLM_PORT="${VLLM_PORT:-8004}"

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }
skip() { log "SKIP: $*"; exit 0; }
fail() { log "FAIL: $*"; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || skip "$1 is not installed"
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

need curl
need python3

curl -fsS "$SERVER_URL/healthz" >/dev/null 2>&1 || skip "LightAI Server is not running at $SERVER_URL"
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

artifact_json="$(api POST /api/v1/model-artifacts "{\"name\":\"$PREFIX-vllm-model\",\"display_name\":\"$PREFIX vLLM model\",\"path\":\"$VLLM_MODEL\",\"format\":\"huggingface\",\"task_type\":\"chat\"}")"
artifact_id="$(printf '%s' "$artifact_json" | json_get id)"
[ -n "$artifact_id" ] || fail "artifact create failed"
api POST "/api/v1/model-artifacts/$artifact_id/locations" "{\"node_id\":\"$node_id\",\"absolute_path\":\"$VLLM_MODEL\",\"path_type\":\"directory\",\"verification_status\":\"verified\",\"match_status\":\"exact_match\"}" >/dev/null

deployment_json="$(api POST /api/v1/deployments "{\"name\":\"$PREFIX-vllm-deployment\",\"model_artifact_id\":\"$artifact_id\",\"backend_runtime_id\":\"$runtime_id\",\"placement_json\":\"{\\\"node_id\\\":\\\"$node_id\\\",\\\"gpu_ids\\\":[\\\"$gpu_id\\\"]}\",\"service_json\":\"{\\\"host_port\\\":$VLLM_PORT}\",\"parameters_json\":\"{\\\"served_model_name\\\":\\\"$PREFIX-vllm\\\",\\\"max_model_len\\\":4096}\",\"env_overrides_json\":\"{}\"}")"
deployment_id="$(printf '%s' "$deployment_json" | json_get id)"
[ -n "$deployment_id" ] || fail "deployment create failed"

start_json="$(api POST "/api/v1/deployments/$deployment_id/start" '{}')"
run_plan_id="$(printf '%s' "$start_json" | json_get run_plan_id)"
instance_id="$(printf '%s' "$start_json" | json_get instance_id)"
[ -n "$run_plan_id" ] || fail "start did not return run_plan_id: $start_json"
log "instance_id=$instance_id run_plan_id=$run_plan_id"

api GET "/api/v1/deployments/$deployment_id/run-plan-groups" >/tmp/lightai-e2e-run-plan-groups.json
api GET "/api/v1/node-run-plans/$run_plan_id" >/tmp/lightai-e2e-node-run-plan.json
api GET "/api/v1/node-run-plans/$run_plan_id/command-preview" >/tmp/lightai-e2e-command-preview.json
grep -q 'vllm/vllm-openai' /tmp/lightai-e2e-command-preview.json || fail "command preview missing image"

deadline=$((SECONDS + 240))
while [ "$SECONDS" -lt "$deadline" ]; do
  if curl -fsS "http://127.0.0.1:$VLLM_PORT/v1/models" >/tmp/lightai-e2e-vllm-models.json 2>/dev/null; then
    log "/v1/models PASS"
    break
  fi
  sleep 5
done

api GET "/api/v1/node-run-plans/$run_plan_id/logs?tail=200" >/tmp/lightai-e2e-node-run-plan-logs.json || true
api POST "/api/v1/deployments/$deployment_id/stop" '{}' >/tmp/lightai-e2e-stop.json

log "PASS: backend runtime NVIDIA API E2E completed"
