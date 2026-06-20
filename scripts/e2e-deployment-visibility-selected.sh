#!/usr/bin/env bash
# Deployment visibility API-only E2E.
# Verifies deployment list/detail/dry-run/delete visibility without starting containers.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export LIGHTAI_E2E_PREFIX="${LIGHTAI_E2E_PREFIX:-e2e-deploy-vis}"
export LIGHTAI_E2E_ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-${ARTIFACT_DIR:-$SCRIPT_DIR/../tmp/e2e-deploy-vis-$(date +%Y%m%d-%H%M%S)-$$}}"

source "$SCRIPT_DIR/e2e/lib/env.sh"
source "$SCRIPT_DIR/e2e/lib/api-client.sh"
source "$SCRIPT_DIR/e2e/lib/assert.sh"
source "$SCRIPT_DIR/e2e/lib/resources.sh"
source "$SCRIPT_DIR/e2e/lib/cleanup.sh"

e2e_with_cleanup_trap

log() { printf '[%s] [deploy-vis] %s\n' "$(date '+%H:%M:%S')" "$*"; }
json_field() { python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('$1',''))" 2>/dev/null; }

e2e_wait_server_ready 30
e2e_login

NODE_ID="$(e2e_api_get "nodes" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)"
[ -n "$NODE_ID" ] || e2e_die "no node found"
ART_ID="$(e2e_api_get "model-artifacts" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)"
[ -n "$ART_ID" ] || e2e_die "no model artifacts found"
RUNTIME_ID="$(e2e_api_get "backend-runtimes" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    if 'vllm' in r.get('id','') and r.get('vendor','') == 'nvidia':
        print(r['id'])
        break
" 2>/dev/null)"
[ -n "$RUNTIME_ID" ] || e2e_die "no vLLM NVIDIA runtime found"
log "node=$NODE_ID artifact=$ART_ID runtime=$RUNTIME_ID"

log "test=list integrity"
LIST_JSON="$(e2e_api_get "deployments")"
printf '%s\n' "$LIST_JSON" > "$LIGHTAI_E2E_ARTIFACT_DIR/deployment-list.json"
LIST_COUNT="$(printf '%s' "$LIST_JSON" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null)"
assert_nonempty "deployment list returns array" "$LIST_COUNT"
printf '%s' "$LIST_JSON" | python3 -c "
import json,sys
deps = json.load(sys.stdin)
required = ['id','name','status','desired_state','model_artifact_id','backend_runtime_id','placement_json','service_json','created_at','updated_at']
for i, d in enumerate(deps):
    for f in required:
        if f not in d:
            print(f'MISSING:{i}:{f}')
            sys.exit(1)
print('OK: all {} deployments have required fields'.format(len(deps)))
" > "$LIGHTAI_E2E_ARTIFACT_DIR/list-field-check.txt" 2>&1
assert_contains "all list rows have required fields" "$(cat "$LIGHTAI_E2E_ARTIFACT_DIR/list-field-check.txt")" "OK"

log "test=create and detail"
DEP_NAME="$(e2e_resource_name "visibility")"
CREATE_RESP="$(e2e_api_post "deployments" "{\"name\":\"$DEP_NAME\",\"display_name\":\"Visibility Test\",\"model_artifact_id\":\"$ART_ID\",\"backend_runtime_id\":\"$RUNTIME_ID\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8291,\"container_port\":8000,\"app_port\":8000},\"parameters_json\":{}}" 201)"
printf '%s\n' "$CREATE_RESP" > "$LIGHTAI_E2E_ARTIFACT_DIR/create-response.json"
DEP_ID="$(printf '%s' "$CREATE_RESP" | json_field id)"
[ -n "$DEP_ID" ] || e2e_die "deployment create did not return id"
e2e_register_resource deployment "$DEP_ID" "/api/v1/deployments/$DEP_ID"
e2e_cleanup_add "curl -sS -b '$LIGHTAI_E2E_COOKIE_JAR' -H 'Origin: $LIGHTAI_SERVER_URL' -H 'X-CSRF-Token: $E2E_CSRF_TOKEN' -X DELETE '$LIGHTAI_SERVER_URL/api/v1/deployments/$DEP_ID' >/dev/null 2>&1 || true"

LIST2="$(e2e_api_get "deployments")"
printf '%s\n' "$LIST2" > "$LIGHTAI_E2E_ARTIFACT_DIR/deployment-list2.json"
FOUND_NAME="$(printf '%s' "$LIST2" | python3 -c "
import json,sys
for d in json.load(sys.stdin):
    if d.get('id') == '$DEP_ID':
        print(d.get('name',''))
        break
" 2>/dev/null)"
assert_eq "deployment appears in list" "$DEP_NAME" "$FOUND_NAME"

DETAIL="$(e2e_api_get "deployments/$DEP_ID")"
printf '%s\n' "$DETAIL" > "$LIGHTAI_E2E_ARTIFACT_DIR/deployment-detail.json"
assert_eq "detail name matches" "$DEP_NAME" "$(printf '%s' "$DETAIL" | json_field name)"
assert_eq "detail status=saved" "saved" "$(printf '%s' "$DETAIL" | json_field status)"
assert_eq "detail desired_state=stopped" "stopped" "$(printf '%s' "$DETAIL" | json_field desired_state)"
assert_eq "detail artifact matches" "$ART_ID" "$(printf '%s' "$DETAIL" | json_field model_artifact_id)"
assert_eq "detail runtime matches" "$RUNTIME_ID" "$(printf '%s' "$DETAIL" | json_field backend_runtime_id)"
DETAIL_PLACEMENT="$(printf '%s' "$DETAIL" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('placement_json',{})))" 2>/dev/null)"
assert_contains "detail placement has node" "$DETAIL_PLACEMENT" "$NODE_ID"
DETAIL_CONFIG="$(printf '%s' "$DETAIL" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('config_snapshot_json') is not None)" 2>/dev/null)"
assert_eq "detail has config_snapshot_json" "True" "$DETAIL_CONFIG"
PLACEMENT_TYPE="$(printf '%s' "$DETAIL" | python3 -c "import json,sys; d=json.load(sys.stdin); print(type(d.get('placement_json')).__name__)" 2>/dev/null)"
assert_eq "placement_json is dict" "dict" "$PLACEMENT_TYPE"

log "test=dry-run visibility"
DR_RESP="$(e2e_api_post "deployments/$DEP_ID/dry-run" "{}" 200)"
printf '%s\n' "$DR_RESP" > "$LIGHTAI_E2E_ARTIFACT_DIR/dryrun-response.json"
assert_eq "dryrun valid=true" "True" "$(printf '%s' "$DR_RESP" | json_field valid)"
DR_PREVIEW="$(printf '%s' "$DR_RESP" | json_field command_preview)"
assert_nonempty "dryrun has command_preview" "$DR_PREVIEW"
assert_nonempty "dryrun has resolved_image" "$(printf '%s' "$DR_RESP" | json_field resolved_image)"
assert_eq "dryrun selected_node" "$NODE_ID" "$(printf '%s' "$DR_RESP" | json_field selected_node)"
assert_nonempty "dryrun has selected_model_location" "$(printf '%s' "$DR_RESP" | json_field selected_model_location)"
assert_nonempty "dryrun has selected_runtime" "$(printf '%s' "$DR_RESP" | json_field selected_runtime)"
assert_contains "dryrun preview has docker run" "$DR_PREVIEW" "docker run"

log "test=delete visibility"
e2e_api_delete "deployments/$DEP_ID" "" 200 > "$LIGHTAI_E2E_ARTIFACT_DIR/delete-response.json"
LIST3="$(e2e_api_get "deployments")"
STILL_THERE="$(printf '%s' "$LIST3" | python3 -c "
import json,sys
for d in json.load(sys.stdin):
    if d.get('id') == '$DEP_ID':
        print('yes')
        break
" 2>/dev/null)"
assert_empty "deployment removed from list after delete" "$STILL_THERE"

log "test=no stale entries"
STALE_COUNT=0
while IFS= read -r did; do
  [ -z "$did" ] && continue
  detail="$(e2e_api_get "deployments/$did")"
  check_id="$(printf '%s' "$detail" | json_field id)"
  if [ "$check_id" != "$did" ]; then
    log "STALE: deployment $did not fetchable"
    STALE_COUNT=$((STALE_COUNT + 1))
  fi
done < <(printf '%s' "$LIST3" | python3 -c "import json,sys; [print(d['id']) for d in json.load(sys.stdin)]" 2>/dev/null)
assert_eq "no stale deployments" "0" "$STALE_COUNT"

log "Artifacts: $LIGHTAI_E2E_ARTIFACT_DIR"
assert_summary
