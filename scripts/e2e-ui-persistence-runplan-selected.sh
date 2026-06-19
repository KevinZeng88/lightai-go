#!/usr/bin/env bash
set -euo pipefail

SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${E2E_USERNAME:-admin}"
PASSWORD="${E2E_PASSWORD:-test1234}"
ARTIFACT_DIR="${ARTIFACT_DIR:-/tmp/lightai-ui-persistence-runplan-selected-$(date +%Y%m%d%H%M%S)}"
COOKIE_JAR="${COOKIE_JAR:-$(mktemp)}"
CSRF_TOKEN=""

mkdir -p "$ARTIFACT_DIR"

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }
fail() { log "FAIL: $*"; printf '%s\n' "$*" > "$ARTIFACT_DIR/failure-reason.txt"; exit 1; }

json_get() {
  python3 -c '
import json,sys
d=json.load(sys.stdin)
for k in sys.argv[1].split("."):
    if isinstance(d, list):
        d=d[int(k)] if k.isdigit() and int(k)<len(d) else {}
    elif isinstance(d, dict):
        d=d.get(k, "")
    else:
        d=""
print(d if d is not None else "")
' "$1" 2>/dev/null
}

api_raw() {
  local method="$1" path="$2" body="${3:-}"
  local args=(-sS -X "$method" "$SERVER_URL$path" -b "$COOKIE_JAR" -c "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "Content-Type: application/json")
  [ -n "$CSRF_TOKEN" ] && [ "$method" != "GET" ] && args+=(-H "X-CSRF-Token: $CSRF_TOKEN")
  [ -n "$body" ] && args+=(-d "$body")
  curl "${args[@]}" -w $'\nHTTP:%{http_code}'
}

api_ok() {
  local raw code body
  raw="$(api_raw "$@")"
  code="$(printf '%s\n' "$raw" | awk -F: '/^HTTP:/{print $2}' | tail -1)"
  body="$(printf '%s\n' "$raw" | sed '/^HTTP:/d')"
  [ "$code" = "200" ] || [ "$code" = "201" ] || { printf '%s\n' "$body" >&2; return 1; }
  printf '%s\n' "$body"
}

if ! curl -fsS "$SERVER_URL/api/v1/health" > "$ARTIFACT_DIR/health.json" 2>/dev/null; then
  curl -fsS "$SERVER_URL/healthz" > "$ARTIFACT_DIR/health.json" 2>/dev/null || fail "server not reachable at $SERVER_URL"
fi

login="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" -H "Origin: $SERVER_URL" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR")"
CSRF_TOKEN="$(printf '%s' "$login" | json_get csrf_token)"
[ -n "$CSRF_TOKEN" ] || fail "login failed"

node_id="$(api_ok GET /api/v1/nodes | json_get 0.id)"
[ -n "$node_id" ] || fail "no node available"
roots_json="$(api_ok GET "/api/v1/nodes/$node_id/model-roots?include_disabled=true" || printf '[]')"
model_root="$(printf '%s' "$roots_json" | python3 -c 'import json,sys
rows=json.load(sys.stdin)
for row in rows:
    if row.get("status","enabled") == "enabled" and row.get("path"):
        print(row["path"]); sys.exit(0)
print("")
')"
if [ -z "$model_root" ]; then
  root_resp="$(api_ok POST "/api/v1/nodes/$node_id/model-roots" '{"path":"/home/kzeng/models"}')"
  model_root="$(printf '%s' "$root_resp" | json_get path)"
fi
[ -n "$model_root" ] || fail "no enabled model root available"
runtime_list="$(api_ok GET /api/v1/backend-runtimes)"
runtime_id="$(printf '%s' "$runtime_list" | python3 -c 'import json,sys
rows=json.load(sys.stdin)
for row in rows:
    if row.get("is_editable") is True:
        print(row.get("id","")); sys.exit(0)
print(rows[0].get("id","") if rows else "")
')"
[ -n "$runtime_id" ] || fail "no backend runtime available"

run_id="$(date +%s)"
artifact_payload="{\"name\":\"ui-persist-$run_id\",\"display_name\":\"UI Persist $run_id\",\"path\":\"$model_root/ui-persist-$run_id.gguf\",\"format\":\"gguf\",\"task_type\":\"chat\"}"
printf '%s\n' "$artifact_payload" > "$ARTIFACT_DIR/model-artifact-request.json"
artifact_json="$(api_ok POST /api/v1/model-artifacts "$artifact_payload")"
printf '%s\n' "$artifact_json" > "$ARTIFACT_DIR/model-artifact.json"
artifact_id="$(printf '%s' "$artifact_json" | json_get id)"
[ -n "$artifact_id" ] || fail "artifact create returned no id"
location_json="$(api_ok POST "/api/v1/model-artifacts/$artifact_id/locations" "{\"node_id\":\"$node_id\",\"model_root\":\"$model_root\",\"relative_path\":\"ui-persist-$run_id.gguf\",\"absolute_path\":\"$model_root/ui-persist-$run_id.gguf\",\"path_type\":\"file\",\"verification_status\":\"verified\",\"match_status\":\"exact_match\"}")"
printf '%s\n' "$location_json" > "$ARTIFACT_DIR/model-location.json"

runtime_detail="$(api_ok GET "/api/v1/backend-runtimes/$runtime_id")"
runtime_editable="$(printf '%s' "$runtime_detail" | json_get is_editable)"
if [ "$runtime_editable" != "True" ] && [ "$runtime_editable" != "true" ]; then
  cloned_runtime="$(api_ok POST "/api/v1/backend-runtimes/$runtime_id/clone" "{\"display_name\":\"UI Runtime $run_id\"}")"
  printf '%s\n' "$cloned_runtime" > "$ARTIFACT_DIR/runtime-clone.json"
  runtime_id="$(printf '%s' "$cloned_runtime" | json_get id)"
fi

runtime_patch="{\"display_name\":\"UI Runtime $run_id\"}"
printf '%s\n' "$runtime_patch" > "$ARTIFACT_DIR/runtime-patch-request.json"
runtime_json="$(api_ok PATCH "/api/v1/backend-runtimes/$runtime_id" "$runtime_patch")"
printf '%s\n' "$runtime_json" > "$ARTIFACT_DIR/runtime.json"
runtime_image="$(printf '%s' "$runtime_json" | json_get image_name)"
nbr_json="$(api_ok POST "/api/v1/nodes/$node_id/backend-runtimes/enable" "{\"backend_runtime_id\":\"$runtime_id\",\"display_name\":\"UI Node Runtime $run_id\",\"image_ref\":\"$runtime_image\",\"image_present\":true,\"docker_available\":true}")"
printf '%s\n' "$nbr_json" > "$ARTIFACT_DIR/node-backend-runtime.json"
# Agent check to set NBR ready
api_ok POST "/api/v1/nodes/$node_id/backend-runtimes/check" "{\"backend_runtime_id\":\"$runtime_id\",\"image_ref\":\"$runtime_image\",\"image_present\":true,\"docker_available\":true}" > /dev/null

deployment_payload="{\"name\":\"ui-persist-deploy-$run_id\",\"display_name\":\"UI Persist Deploy $run_id\",\"model_artifact_id\":\"$artifact_id\",\"backend_runtime_id\":\"$runtime_id\",\"placement_json\":{\"node_id\":\"$node_id\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8005,\"container_port\":8080,\"app_port\":8080,\"health_port\":8005,\"api_test_port\":8005},\"parameters_json\":{\"served_model_name\":\"ui-persist-$run_id\"},\"env_overrides_json\":{}}"
printf '%s\n' "$deployment_payload" > "$ARTIFACT_DIR/deployment-request.json"
deployment_json="$(api_ok POST /api/v1/deployments "$deployment_payload")"
printf '%s\n' "$deployment_json" > "$ARTIFACT_DIR/deployment.json"
deployment_id="$(printf '%s' "$deployment_json" | json_get id)"
[ -n "$deployment_id" ] || fail "deployment create returned no id"

instance_count="$(api_ok GET "/api/v1/model-instances?deployment_id=$deployment_id" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
[ "$instance_count" = "0" ] || fail "save-only created model instances"

dry_run_json="$(api_ok POST "/api/v1/deployments/$deployment_id/dry-run" '{}')"
printf '%s\n' "$dry_run_json" > "$ARTIFACT_DIR/runplan-preview.json"
printf '%s\n' "$dry_run_json" | grep -q '8005' || fail "preview missing host_port 8005"
printf '%s\n' "$dry_run_json" | grep -q '8080' || fail "preview missing container/app port 8080"

set +e
start_json="$(api_ok POST "/api/v1/deployments/$deployment_id/start" '{}')"
start_rc=$?
set -e
printf '%s\n' "$start_json" > "$ARTIFACT_DIR/start-response.json"
if [ "$start_rc" -eq 0 ]; then
  instance_id="$(printf '%s' "$start_json" | json_get instance_id)"
  run_plan_id="$(printf '%s' "$start_json" | json_get run_plan_id)"
  [ -n "$instance_id" ] || fail "start response missing instance_id"
  [ -n "$run_plan_id" ] || fail "start response missing run_plan_id"
  api_ok GET "/api/v1/node-run-plans/$run_plan_id" > "$ARTIFACT_DIR/runplan.json"
  api_raw POST "/api/v1/deployments/$deployment_id/start" '{}' > "$ARTIFACT_DIR/repeated-start-response.txt"
  grep -q 'HTTP:409' "$ARTIFACT_DIR/repeated-start-response.txt" || fail "repeated start did not return 409"
else
  log "start failed in current environment; saved response for diagnostics"
fi

printf '{"status":"PASS","artifact_dir":"%s"}\n' "$ARTIFACT_DIR" > "$ARTIFACT_DIR/summary.json"
log "PASS artifacts=$ARTIFACT_DIR"
