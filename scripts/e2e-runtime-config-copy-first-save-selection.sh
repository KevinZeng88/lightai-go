#!/bin/bash
# e2e-runtime-config-copy-first-save-selection.sh
# Verifies: clone first-save override persistence, NBR auto-creation, wizard selector visibility
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/e2e-assert.sh"

SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-Commvault!234}"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-/tmp/lightai-e2e-rt-copy-$RUN_ID}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/lightai-e2e-cookies-$RUN_ID.txt}"
PREFIX="e2e-rtcopy"
mkdir -p "$ARTIFACT_DIR"

log() { printf '[%s] [rt-copy] %s\n' "$(date '+%H:%M:%S')" "$*"; }

api_get() { curl -sS -b "$COOKIE_JAR" -H "Origin: $SERVER_URL" -X GET "$SERVER_URL/api/v1/$1"; }
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

# Login + discover
resp="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" -H "Origin: $SERVER_URL" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR")"
CSRF_TOKEN="$(echo "$resp" | json_field csrf_token)"
[ -n "$CSRF_TOKEN" ] || { log "FATAL: Login failed"; exit 1; }
NODE_ID=$(api_get "nodes" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)
[ -n "$NODE_ID" ] || { log "FATAL: No online node"; exit 1; }
log "Node: $NODE_ID"

# Setup: need model root + artifacts + locations for DryRun
curl -sS -b "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF_TOKEN" -X POST "$SERVER_URL/api/v1/nodes/$NODE_ID/model-roots" -d '{"path":"/home/kzeng/models","label":"models"}' > /dev/null 2>&1
HF_ART=$(api_get "model-artifacts" | python3 -c "import json,sys; [print(a['id']) for a in json.load(sys.stdin) if a.get('format')=='huggingface']" 2>/dev/null | head -1)
if [ -z "$HF_ART" ]; then
  HF_RESP=$(api_post "model-artifacts" '{"name":"Qwen3-0.6B-Instruct-2512","path":"/home/kzeng/models/Qwen3-0.6B-Instruct-2512","format":"huggingface","task_type":"chat"}')
  HF_ART=$(echo "$HF_RESP" | json_field id)
  api_post "model-artifacts/$HF_ART/locations" "{\"node_id\":\"$NODE_ID\",\"absolute_path\":\"/home/kzeng/models/Qwen3-0.6B-Instruct-2512\"}" > /dev/null
fi
GGUF_ART=$(api_get "model-artifacts" | python3 -c "import json,sys; [print(a['id']) for a in json.load(sys.stdin) if a.get('format')=='gguf']" 2>/dev/null | head -1)
if [ -z "$GGUF_ART" ]; then
  GGUF_RESP=$(api_post "model-artifacts" '{"name":"Qwen3.5-9B-Q4_K_M.gguf","path":"/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf","format":"gguf","task_type":"chat"}')
  GGUF_ART=$(echo "$GGUF_RESP" | json_field id)
  api_post "model-artifacts/$GGUF_ART/locations" "{\"node_id\":\"$NODE_ID\",\"absolute_path\":\"/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf\"}" > /dev/null
fi
log "Artifacts: HF=$HF_ART GGUF=$GGUF_ART"

# Get source template runtimes
RTS=$(api_get "backend-runtimes")
echo "$RTS" > "$ARTIFACT_DIR/runtimes-initial.json"
VLLM_SRC=$(echo "$RTS" | python3 -c "import json,sys; [print(r['id']) for r in json.load(sys.stdin) if 'vllm' in r.get('id','') and r.get('vendor','')=='nvidia']" 2>/dev/null | head -1)
SGLANG_SRC=$(echo "$RTS" | python3 -c "import json,sys; [print(r['id']) for r in json.load(sys.stdin) if 'sglang' in r.get('id','') and r.get('vendor','')=='nvidia']" 2>/dev/null | head -1)
LLAMACPP_SRC=$(echo "$RTS" | python3 -c "import json,sys; [print(r['id']) for r in json.load(sys.stdin) if 'llamacpp' in r.get('id','') and r.get('vendor','')=='nvidia']" 2>/dev/null | head -1)
log "Sources: vllm=$VLLM_SRC sglang=$SGLANG_SRC llamacpp=$LLAMACPP_SRC"

# ═══════════════════════════════════════════════════════
# Test cluster: run for each backend
# ═══════════════════════════════════════════════════════
test_backend() {
  local label="$1" src="$2" art_id="$3"
  log "=== $label: clone + first-save + visibility ==="

  # Get source detail
  local src_detail; src_detail=$(api_get "backend-runtimes/$src")
  echo "$src_detail" > "$ARTIFACT_DIR/${label}-source.json"
  local src_shm; src_shm=$(echo "$src_detail" | python3 -c "import json,sys; d=json.load(sys.stdin); dj=d.get('docker_json',{}); print(dj.get('shm_size','') if isinstance(dj,dict) else '')" 2>/dev/null)
  log "  Source shm_size=$src_shm"

  # Clone with override: shm_size=6gb + custom display_name
  local clone_name="${PREFIX}-${label}-user"
  local clone_dn="E2E ${label} User Runtime"
  local clone_payload; clone_payload=$(python3 -c "
import json
payload = {
    'name': '$clone_name',
    'display_name': '$clone_dn',
    'image_name': '$(echo "$src_detail" | python3 -c "import json,sys; print(json.load(sys.stdin).get('image_name',''))" 2>/dev/null)',
    'vendor': 'nvidia',
    'docker_json': {'shm_size': '6gb', 'ipc_mode': 'host', 'privileged': False},
    'args_override_json': [],
    'default_env_json': {},
    'entrypoint_override_json': []
}
print(json.dumps(payload))
")
  echo "$clone_payload" > "$ARTIFACT_DIR/${label}-clone-request.json"

  local clone_resp; clone_resp=$(api_post "backend-runtimes/$src/clone" "$clone_payload")
  echo "$clone_resp" > "$ARTIFACT_DIR/${label}-clone-response.json"
  local clone_id; clone_id=$(echo "$clone_resp" | json_field id)
  [ -n "$clone_id" ] || { log "  FAIL: clone create failed"; echo "FAIL" >> "$ARTIFACT_DIR/${label}-results.txt"; return; }
  log "  Cloned: $clone_id"

  # ── Test 1: First-save override persistence ──
  log "  --- Test 1: First-save override ---"
  local detail; detail=$(api_get "backend-runtimes/$clone_id")
  echo "$detail" > "$ARTIFACT_DIR/${label}-detail-first.json"
  local saved_shm; saved_shm=$(echo "$detail" | python3 -c "import json,sys; d=json.load(sys.stdin); dj=d.get('docker_json',{}); print(dj.get('shm_size','MISSING') if isinstance(dj,dict) else 'NOT_DICT')" 2>/dev/null)
  local saved_dn; saved_dn=$(echo "$detail" | json_field display_name)
  local saved_editable; saved_editable=$(echo "$detail" | json_field is_editable)
  local saved_builtin; saved_builtin=$(echo "$detail" | json_field is_builtin)

  assert_eq "$label: first-save shm_size=6gb" "6gb" "$saved_shm" || log "FAIL: shm_size=$saved_shm (expected 6gb)"
  assert_eq "$label: display_name preserved" "$clone_dn" "$saved_dn" || log "FAIL: display_name=$saved_dn"
  assert_contains "$label: is_editable=1" "$saved_editable" "rue" || log "FAIL: not editable"
  assert_contains "$label: is_builtin=0" "$saved_builtin" "alse" || log "FAIL: builtin"

  # ── Test 2: Clone does NOT auto-create NBR ──
  log "  --- Test 2: Clone does NOT auto-create NBR ---"
  local nbr_list; nbr_list=$(api_get "nodes/$NODE_ID/backend-runtimes")
  echo "$nbr_list" > "$ARTIFACT_DIR/${label}-nbrs-before-enable.json"
  local nbr_status; nbr_status=$(echo "$nbr_list" | python3 -c "
import json,sys
for n in json.load(sys.stdin):
    if n.get('backend_runtime_id') == '$clone_id':
        print(n.get('status',''))
        break
" 2>/dev/null)
  assert_empty "$label: no auto NBR after clone" "$nbr_status" || log "FAIL: NBR auto-created (should not happen)"

  # ── Test 3: Explicit enable on node ──
  log "  --- Test 3: Explicit enable on node ---"
  local enable_resp; enable_resp=$(api_post "nodes/$NODE_ID/backend-runtimes/enable" "{\"backend_runtime_id\":\"$clone_id\",\"image_present\":true,\"docker_available\":true}")
  echo "$enable_resp" > "$ARTIFACT_DIR/${label}-enable-response.json"
  local nbr_enabled_status; nbr_enabled_status=$(echo "$enable_resp" | json_field status)
  assert_eq "$label: explicit enable creates NBR" "ready" "$nbr_enabled_status" || log "NBR status=$nbr_enabled_status (expected ready)"

  # ── Test 4: DryRun after explicit enable — uses user's shm_size ──
  log "  --- Test 4: DryRun after explicit enable ---"
  local preview; preview=""
  if [ -n "$art_id" ]; then
    local dep_payload="{\"name\":\"$PREFIX-${label}-dep\",\"model_artifact_id\":\"$art_id\",\"backend_runtime_id\":\"$clone_id\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8501,\"container_port\":8000,\"app_port\":8000},\"parameters_json\":{}}"
    local dep_resp; dep_resp=$(api_post "deployments" "$dep_payload")
    local dep_id; dep_id=$(echo "$dep_resp" | json_field id)
    if [ -n "$dep_id" ]; then
      local dr; dr=$(api_post "deployments/$dep_id/dry-run" '{}')
      echo "$dr" > "$ARTIFACT_DIR/${label}-dryrun.json"
      preview=$(echo "$dr" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('command_preview',''))" 2>/dev/null)
      echo "$preview" > "$ARTIFACT_DIR/${label}-preview.txt"
      assert_contains "$label: DryRun uses 6gb" "$preview" "6gb" || log "FAIL: 6gb not in DryRun"
      assert_not_contains "$label: DryRun does NOT use source default" "$preview" "$src_shm" || log "FAIL: source default $src_shm leaked"
      api_delete "deployments/$dep_id" > /dev/null 2>&1 || true
    fi
  fi

  # ── Test 5: Deployment wizard runtime list includes cloned runtime ──
  log "  --- Test 5: Wizard selector visibility ---"
  local rt_list; rt_list=$(api_get "backend-runtimes")
  echo "$rt_list" > "$ARTIFACT_DIR/${label}-runtimes-after.json"
  local found; found=$(echo "$rt_list" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    if r.get('id') == '$clone_id':
        print('yes: dn=' + str(r.get('display_name','')))
        break
" 2>/dev/null)
  assert_contains "$label: appears in runtime list" "$found" "yes" || log "FAIL: not in runtime list"
  assert_contains "$label: display_name in list" "$found" "$clone_dn" || log "FAIL: display_name wrong in list"

  log "  $label: DONE"
  echo "$clone_id" > "$ARTIFACT_DIR/${label}-clone-id.txt"
}

# Run for each backend
test_backend "vllm" "$VLLM_SRC" "$HF_ART"
test_backend "sglang" "$SGLANG_SRC" "$HF_ART"
test_backend "llamacpp" "$LLAMACPP_SRC" "$GGUF_ART"

# Cleanup cloned runtimes
for label in vllm sglang llamacpp; do
  cid=$(cat "$ARTIFACT_DIR/${label}-clone-id.txt" 2>/dev/null || true)
  [ -n "$cid" ] && api_delete "backend-runtimes/$cid" > /dev/null 2>&1 || true
done

echo ""
echo "Artifacts: $ARTIFACT_DIR"
assert_summary
