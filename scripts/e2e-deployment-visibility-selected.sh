#!/bin/bash
# e2e-deployment-visibility-selected.sh — Deployment visibility E2E.
# Category: API-only E2E (no containers, no GPU usage beyond API queries)
# Verifies: deployment list/detail/status/visibility lifecycle.
#
# Each deployment detail must include: id, name, display_name, status,
# desired_state, model_artifact_id, backend_runtime_id, placement_json,
# service_json, config_snapshot_json, created_at, updated_at.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/e2e-assert.sh"

SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-Commvault!234}"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-/tmp/lightai-e2e-deploy-vis-$RUN_ID}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/lightai-e2e-cookies-$RUN_ID.txt}"
PREFIX="e2e-deploy-vis"
mkdir -p "$ARTIFACT_DIR"

log()   { printf '[%s] [deploy-vis] %s\n' "$(date '+%H:%M:%S')" "$*"; }

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

ART_ID=$(api_get "model-artifacts" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)
[ -n "$ART_ID" ] || { log "FATAL: No model artifacts"; exit 1; }

RUNTIME_ID=$(api_get "backend-runtimes" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    if 'vllm' in r.get('id','') and r.get('vendor','')=='nvidia':
        print(r['id']); break
" 2>/dev/null)
[ -n "$RUNTIME_ID" ] || { log "FATAL: No vLLM runtime"; exit 1; }
log "Runtime: $RUNTIME_ID artifact: $ART_ID"

# ═══════════════════════════════════════════════════════════════
# Test 1: List deployments — all rows return valid JSON
# ═══════════════════════════════════════════════════════════════
log "=== Test 1: Deployment list integrity ==="
LIST_JSON=$(api_get "deployments")
echo "$LIST_JSON" > "$ARTIFACT_DIR/deployment-list.json"

LIST_COUNT=$(echo "$LIST_JSON" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null)
assert_nonempty "deployment list returns array" "$LIST_COUNT" || log "FAIL: list not array"

# Every deployment in the list must have required fields
echo "$LIST_JSON" | python3 -c "
import json,sys
deps = json.load(sys.stdin)
required = ['id','name','status','desired_state','model_artifact_id','backend_runtime_id','placement_json','service_json','created_at','updated_at']
for i, d in enumerate(deps):
    for f in required:
        if f not in d:
            print(f'MISSING:{i}:{f}')
            sys.exit(1)
print('OK: all {} deployments have required fields'.format(len(deps)))
" > "$ARTIFACT_DIR/list-field-check.txt" 2>&1
assert_contains "all list rows have required fields" "$(cat "$ARTIFACT_DIR/list-field-check.txt")" "OK" || log "FAIL: field validation"

log "Deployment list integrity done"

# ═══════════════════════════════════════════════════════════════
# Test 2: Create deployment → verify list/detail/status
# ═══════════════════════════════════════════════════════════════
log "=== Test 2: Create + detail visibility ==="
DEP_NAME="${PREFIX}-visibility-test"

CREATE_RESP=$(api_post "deployments" "{\"name\":\"$DEP_NAME\",\"display_name\":\"Visibility Test\",\"model_artifact_id\":\"$ART_ID\",\"backend_runtime_id\":\"$RUNTIME_ID\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8291,\"container_port\":8000,\"app_port\":8000},\"parameters_json\":{}}")
echo "$CREATE_RESP" > "$ARTIFACT_DIR/create-response.json"
DEP_ID=$(echo "$CREATE_RESP" | json_field id)
[ -n "$DEP_ID" ] || { log "FATAL: Deployment create failed"; exit 1; }
log "Created deployment: $DEP_ID"

# Verify it appears in list
LIST2=$(api_get "deployments")
echo "$LIST2" > "$ARTIFACT_DIR/deployment-list2.json"
FOUND_NAME=$(echo "$LIST2" | python3 -c "
import json,sys
for d in json.load(sys.stdin):
    if d.get('id') == '$DEP_ID':
        print(d.get('name',''))
        break
" 2>/dev/null)
assert_eq "deployment appears in list" "$DEP_NAME" "$FOUND_NAME" || log "FAIL: deployment not in list"

# Verify detail fields
DETAIL=$(api_get "deployments/$DEP_ID")
echo "$DETAIL" > "$ARTIFACT_DIR/deployment-detail.json"

DETAIL_NAME=$(echo "$DETAIL" | json_field name)
DETAIL_STATUS=$(echo "$DETAIL" | json_field status)
DETAIL_DESIRED=$(echo "$DETAIL" | json_field desired_state)
DETAIL_ART=$(echo "$DETAIL" | json_field model_artifact_id)
DETAIL_RT=$(echo "$DETAIL" | json_field backend_runtime_id)
DETAIL_PLACEMENT=$(echo "$DETAIL" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('placement_json',{})))" 2>/dev/null)
DETAIL_CONFIG=$(echo "$DETAIL" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('config_snapshot_json') is not None)" 2>/dev/null)

assert_eq "detail name matches" "$DEP_NAME" "$DETAIL_NAME" || log "FAIL: detail name"
assert_eq "detail status=saved" "saved" "$DETAIL_STATUS" || log "FAIL: detail status"
assert_eq "detail desired_state=stopped" "stopped" "$DETAIL_DESIRED" || log "FAIL: detail desired_state"
assert_eq "detail artifact matches" "$ART_ID" "$DETAIL_ART" || log "FAIL: detail artifact"
assert_eq "detail runtime matches" "$RUNTIME_ID" "$DETAIL_RT" || log "FAIL: detail runtime"
assert_contains "detail placement has node" "$DETAIL_PLACEMENT" "$NODE_ID" || log "FAIL: detail placement"
assert_eq "detail has config_snapshot_json" "True" "$DETAIL_CONFIG" || log "FAIL: detail config snapshot"

# Verify placement_json is an object (not string)
PLACEMENT_TYPE=$(echo "$DETAIL" | python3 -c "import json,sys; d=json.load(sys.stdin); p=d.get('placement_json'); print(type(p).__name__)" 2>/dev/null)
assert_eq "placement_json is dict" "dict" "$PLACEMENT_TYPE" || log "FAIL: placement_json type=$PLACEMENT_TYPE"

log "Create + detail visibility done"

# ═══════════════════════════════════════════════════════════════
# Test 3: DryRun returns valid preview
# ═══════════════════════════════════════════════════════════════
log "=== Test 3: DryRun visibility ==="
DR_RESP=$(api_post "deployments/$DEP_ID/dry-run" '{}')
echo "$DR_RESP" > "$ARTIFACT_DIR/dryrun-response.json"

DR_VALID=$(echo "$DR_RESP" | json_field valid)
DR_PREVIEW=$(echo "$DR_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('command_preview',''))" 2>/dev/null)
DR_IMAGE=$(echo "$DR_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('resolved_image',''))" 2>/dev/null)
DR_NODE=$(echo "$DR_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('selected_node',''))" 2>/dev/null)
DR_LOC=$(echo "$DR_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('selected_model_location',''))" 2>/dev/null)
DR_RT=$(echo "$DR_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('selected_runtime',''))" 2>/dev/null)

assert_eq "dryrun valid=true" "True" "$DR_VALID" || log "FAIL: dryrun valid"
assert_nonempty "dryrun has command_preview" "$DR_PREVIEW" || log "FAIL: dryrun preview empty"
assert_nonempty "dryrun has resolved_image" "$DR_IMAGE" || log "FAIL: dryrun image"
assert_eq "dryrun selected_node" "$NODE_ID" "$DR_NODE" || log "FAIL: dryrun node"
assert_nonempty "dryrun selected_model_location" "$DR_LOC" || log "FAIL: dryrun model location"
assert_nonempty "dryrun selected_runtime" "$DR_RT" || log "FAIL: dryrun runtime"
assert_contains "dryrun preview has docker run" "$DR_PREVIEW" "docker run" || log "FAIL: dryrun docker run"

log "DryRun visibility done"

# ═══════════════════════════════════════════════════════════════
# Test 4: Deployment delete → verify removal from list
# ═══════════════════════════════════════════════════════════════
log "=== Test 4: Delete visibility ==="
api_delete "deployments/$DEP_ID" > "$ARTIFACT_DIR/delete-response.json"

LIST3=$(api_get "deployments")
STILL_THERE=$(echo "$LIST3" | python3 -c "
import json,sys
for d in json.load(sys.stdin):
    if d.get('id') == '$DEP_ID':
        print('yes')
        break
" 2>/dev/null)
assert_empty "deployment removed from list after delete" "$STILL_THERE" || log "FAIL: deployment still in list"

log "Delete visibility done"

# ═══════════════════════════════════════════════════════════════
# Test 5: List does NOT contain stale/non-existent deployments
# ═══════════════════════════════════════════════════════════════
log "=== Test 5: No stale entries ==="
# Verify all existing deployments have valid IDs (are fetchable)
STALE_COUNT=0
while IFS= read -r did; do
  [ -z "$did" ] && continue
  local_detail=$(api_get "deployments/$did" 2>/dev/null)
  local_check=$(echo "$local_detail" | json_field id 2>/dev/null)
  if [ "$local_check" != "$did" ]; then
    log "STALE: deployment $did not fetchable"
    STALE_COUNT=$((STALE_COUNT + 1))
  fi
done < <(echo "$LIST3" | python3 -c "import json,sys; [print(d['id']) for d in json.load(sys.stdin)]" 2>/dev/null)
assert_eq "no stale deployments" "0" "$STALE_COUNT" || log "FAIL: $STALE_COUNT stale"

log "Stale check done"

# ── summary ──
echo ""
echo "Artifacts: $ARTIFACT_DIR"
echo "Key files:"
echo "  deployment-list.json deployment-list2.json create-response.json"
echo "  deployment-detail.json dryrun-response.json delete-response.json"
assert_summary
