#!/usr/bin/env bash
# Common helpers for model/runtime wizard E2E tests.
# Source this file from backend-specific E2E scripts.
# Do NOT call Docker run directly — use the product API chain.

# ── configuration (override before sourcing) ──────────────────────────
SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${E2E_USERNAME:-admin}"
PASSWORD="${E2E_PASSWORD:-test1234}"
BACKEND_NAME="${BACKEND_NAME:-unknown}"
BACKEND_RUNTIME_ID="${BACKEND_RUNTIME_ID:-}"
IMAGE_REF="${IMAGE_REF:-}"
MODEL_PATH="${MODEL_PATH:-}"
HOST_PORT="${HOST_PORT:-8000}"
DEPLOY_PARAMS="${DEPLOY_PARAMS:-}"
DEPLOY_ENV="${DEPLOY_ENV:-}"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)}"
ARTIFACT_DIR="${ARTIFACT_DIR:-/tmp/e2e-artifacts-$E2E_RUN_ID-$BACKEND_NAME}"
COOKIE_JAR="${COOKIE_JAR:-$(mktemp)}"
CSRF_TOKEN=""; INSTANCE_ID=""; DEPLOYMENT_ID=""; ARTIFACT_ID=""; ROOT_ID=""; NODE_ID=""; GPU_ID=""

log()   { printf '[%s] [%s] %s\n' "$(date '+%H:%M:%S')" "$BACKEND_NAME" "$*"; }
fail()  { log "FAIL: $*"; return 1; }
abort() { log "FATAL: $*"; exit 1; }
now_ms(){ date +%s%3N; }

json_get() {
  python3 -c "
import json,sys
d=json.load(sys.stdin)
keys='$1'.split('.')
for k in keys:
    if isinstance(d, list): d=d[0] if d else {}
    elif isinstance(d, dict): d=d.get(k, '')
    else: d=''
print(d if d is not None else '')
" 2>/dev/null
}

api() {
  local m="$1" p="$2" d="${3:-}"
  local a=(-sS -X "$m" "$SERVER_URL$p" -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
    -H "Origin: $SERVER_URL" -H "Content-Type: application/json")
  [ -n "$CSRF_TOKEN" ] && [ "$m" != "GET" ] && a+=(-H "X-CSRF-Token: $CSRF_TOKEN")
  [ -n "$d" ] && a+=(-d "$d")
  curl "${a[@]}" -w $'\nHTTP:%{http_code}' 2>/dev/null || return 1
}

api_ok() {
  local r; r="$(api "$@")" || return 1
  local code; code="$(echo "$r" | awk -F: '/^HTTP:/{print $2}' | tail -1)"
  local body; body="$(echo "$r" | sed '/^HTTP:/d')"
  [ "$code" = "200" ] || [ "$code" = "201" ] || { echo "$body" >&2; return 1; }
  echo "$body"
}

api_body() {
  local r; r="$(api "$@")" || return 1
  echo "$r" | sed '/^HTTP:/d'
}

# ── stage helpers ──────────────────────────────────────────────────────────
e2e_login() {
  log "stage=login start"
  local resp; resp="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" \
    -H "Origin: $SERVER_URL" -H "Content-Type: application/json" \
    -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR")"
  CSRF_TOKEN="$(echo "$resp" | json_get csrf_token)"
  [ -n "$CSRF_TOKEN" ] || { fail "login failed"; return 1; }
  log "stage=login done"
}

e2e_query_node() {
  NODE_ID=""
  set +e
  for i in $(seq 1 30); do
    local resp; resp="$(api_body GET /api/v1/nodes 2>/dev/null)" || true
    NODE_ID="$(echo "$resp" | json_get 0.id 2>/dev/null)" || true
    [ -n "$NODE_ID" ] && break
    sleep 1
  done
  set -e
  [ -n "$NODE_ID" ] || { fail "no node found"; return 1; }
  log "node_id=$NODE_ID"
}

e2e_query_gpu() {
  local resp; resp="$(api_body GET /api/v1/gpus 2>/dev/null)" || true
  GPU_ID="$(echo "$resp" | json_get 0.id 2>/dev/null)" || true
  [ -n "$GPU_ID" ] || GPU_ID="$(echo "$resp" | json_get id 2>/dev/null)" || true
  [ -n "$GPU_ID" ] || { fail "no GPU found"; return 1; }
  log "gpu_id=$GPU_ID"
}

e2e_add_model_root() {
  local root_path; root_path="$(dirname "$MODEL_PATH")"
  ROOT_ID="$(api_ok POST "/api/v1/nodes/$NODE_ID/model-roots" "{\"path\":\"$root_path\"}" | json_get id)"
  [ -n "$ROOT_ID" ] || { fail "add model root failed"; return 1; }
  log "root_id=$ROOT_ID root=$root_path"
}

e2e_scan_model() {
  local root_path; root_path="$(dirname "$MODEL_PATH")"
  local rel_path; rel_path="${MODEL_PATH#$root_path/}"
  local r; r="$(api_ok POST "/api/v1/nodes/$NODE_ID/model-paths/scan" \
    "{\"root_id\":\"$ROOT_ID\",\"root\":\"$root_path\",\"relative_path\":\"$rel_path\",\"path_type\":\"directory\"}")"
  ARTIFACT_ID="$(echo "$r" | json_get artifact_id)"
  [ -n "$ARTIFACT_ID" ] || { fail "scan model failed"; return 1; }
  log "artifact_id=$ARTIFACT_ID rel=$rel_path"
}

e2e_enable_nbr() {
  local imgs; imgs="$(api_body GET "/api/v1/nodes/$NODE_ID/docker-images" 2>/dev/null || echo '[]')"
  local ip; ip="$(echo "$imgs" | python3 -c "
import json,sys
imgs=json.load(sys.stdin)
img='${IMAGE_REF}'
# match repo:tag or just repo prefix
for i in imgs:
    tags=i.get('repotags',[]) or []
    name=i.get('name','')
    for t in tags:
        if img in str(t):
            print('true'); sys.exit(0)
print('false')
" 2>/dev/null || echo false)"
  local r; r="$(api_ok POST "/api/v1/nodes/$NODE_ID/backend-runtimes/enable" \
    "{\"backend_runtime_id\":\"$BACKEND_RUNTIME_ID\",\"image_ref\":\"$IMAGE_REF\",\"image_present\":$ip,\"docker_available\":true}")"
  log "nbr enabled status=$(echo "$r" | json_get status)"
}

e2e_create_deployment() {
  local name; name="e2e-${BACKEND_NAME}-${E2E_RUN_ID}"
  local payload; payload="{\"name\":\"$name\",\"model_artifact_id\":\"$ARTIFACT_ID\",\"backend_runtime_id\":\"$BACKEND_RUNTIME_ID\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[\"$GPU_ID\"]},\"service_json\":{\"host_port\":$HOST_PORT}"
  if [ -n "$DEPLOY_PARAMS" ]; then
    payload="$payload,\"parameters_json\":{$DEPLOY_PARAMS}"
  fi
  if [ -n "$DEPLOY_ENV" ]; then
    payload="$payload,\"env_overrides_json\":{$DEPLOY_ENV}"
  fi
  payload="$payload}"
  DEPLOYMENT_ID="$(api_ok POST /api/v1/deployments "$payload" | json_get id)"
  [ -n "$DEPLOYMENT_ID" ] || { fail "create deployment failed"; return 1; }
  log "deployment_id=$DEPLOYMENT_ID"
}

e2e_preflight() {
  local r; r="$(api_body POST /api/v1/deployments/preflight \
    "{\"model_artifact_id\":\"$ARTIFACT_ID\",\"backend_runtime_id\":\"$BACKEND_RUNTIME_ID\",\"host_port\":$HOST_PORT}")"
  log "candidate_nodes=$(echo "$r" | json_get candidate_nodes)"
  mkdir -p "$ARTIFACT_DIR"
  echo "$r" > "$ARTIFACT_DIR/preflight.json"
}

e2e_start_deployment() {
  log "stage=start_deployment start"
  local r; r="$(api_ok POST "/api/v1/deployments/$DEPLOYMENT_ID/start")"
  INSTANCE_ID="$(echo "$r" | json_get instance_id)"
  [ -n "$INSTANCE_ID" ] || { fail "start failed"; return 1; }
  log "instance_id=$INSTANCE_ID"
}

e2e_wait_health() {
  log "stage=health_check start"
  local hc_start; hc_start="$(now_ms)"
  for i in $(seq 1 120); do
    local inst; inst="$(api_body GET "/api/v1/model-instances/$INSTANCE_ID" 2>/dev/null || echo '{}')"
    local state cid; state="$(echo "$inst" | json_get actual_state)"; cid="$(echo "$inst" | json_get container_id)"
    case "$state" in
      running)
        local models; models="$(curl -sS "http://127.0.0.1:$HOST_PORT/v1/models" 2>/dev/null || echo '')"
        if [ -n "$models" ] && echo "$models" | python3 -c 'import json,sys; json.load(sys.stdin)' 2>/dev/null; then
          log "/v1/models PASS cid=${cid:0:12} state=$state"
          echo "$models" > "$ARTIFACT_DIR/v1-models.json"
          local hc_dur; hc_dur=$(($(now_ms) - hc_start))
          log "stage=health_check done duration_ms=$hc_dur"
          return 0
        fi
        ;;
      failed)
        fail "instance failed cid=$cid err=$(echo "$inst" | json_get last_error)"
        return 1
        ;;
    esac
    sleep 2
  done
  fail "/v1/models timeout"
  return 1
}

e2e_instance_test() {
  local r; r="$(api_body POST "/api/v1/model-instances/$INSTANCE_ID/test" \
    '{"mode":"chat","max_tokens":16}' 2>/dev/null || echo '{}')"
  echo "$r" > "$ARTIFACT_DIR/instance-test.json"
  log "instance_test: $(echo "$r" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("status","?"),d.get("duration_ms","?"))' 2>/dev/null || echo 'parse_error')"
}

e2e_docker_logs() {
  local r; r="$(api_body POST "/api/v1/model-instances/$INSTANCE_ID/logs" \
    '{"tail":50}' 2>/dev/null || echo '{}')"
  echo "$r" > "$ARTIFACT_DIR/logs.json"
  log "logs_api: $(echo "$r" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("ok" if d.get("logs") else "empty")' 2>/dev/null || echo 'parse_error')"
}

e2e_stop_deployment() {
  log "stage=stop start"
  api_body POST "/api/v1/deployments/$DEPLOYMENT_ID/stop" > /dev/null 2>&1 || true
  sleep 3
  log "stage=stop done"
}

e2e_cleanup() {
  log "stage=cleanup start"
  api_body DELETE "/api/v1/deployments/$DEPLOYMENT_ID" > /dev/null 2>&1 || true
  api_body DELETE "/api/v1/model-artifacts/$ARTIFACT_ID" > /dev/null 2>&1 || true
  api_body DELETE "/api/v1/nodes/$NODE_ID/model-roots/$ROOT_ID" > /dev/null 2>&1 || true
  log "stage=cleanup done"
}

e2e_save_artifacts() {
  mkdir -p "$ARTIFACT_DIR"
  log "artifacts saved to $ARTIFACT_DIR"
}

# ── full default E2E pipeline ──────────────────────────────────────────────
e2e_run_default() {
  set +e  # allow individual stages to handle errors
  e2e_login        || return 1
  e2e_query_node   || return 1
  e2e_query_gpu    || return 1
  e2e_add_model_root   || return 1
  e2e_scan_model       || return 1
  e2e_enable_nbr       || return 1
  e2e_create_deployment || return 1
  e2e_preflight        || return 1
  e2e_start_deployment || return 1
  e2e_wait_health      || return 1
  e2e_instance_test    || return 1
  e2e_docker_logs      || return 1
  e2e_stop_deployment  || return 1
  e2e_cleanup          || return 1
  e2e_save_artifacts
  return 0
}
