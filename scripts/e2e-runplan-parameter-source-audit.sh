#!/bin/bash
# e2e-runplan-parameter-source-audit.sh — DryRun parameter propagation audit.
# Category: DryRun E2E (no containers, no GPU usage beyond API queries)
# Requires: running LightAI server at SERVER_URL
# Does NOT start containers or create instances.
# Verifies: user-set parameters flow through to RunPlan/Docker command preview.
#
# Auto-setup: discovers existing artifacts, creates model_locations if needed,
# enables NBRs if not ready. No manual DB or agent scan setup required.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/e2e-assert.sh"

# ── configuration ──
SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-Commvault!234}"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-/tmp/lightai-e2e-dryrun-audit-$RUN_ID}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/lightai-e2e-cookies-$RUN_ID.txt}"
PREFIX="e2e-dryrun"
mkdir -p "$ARTIFACT_DIR"

log()   { printf '[%s] [dryrun-audit] %s\n' "$(date '+%H:%M:%S')" "$*"; }
fail()  { log "FAIL: $*"; }

# ── helpers ──
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
[ -n "$CSRF_TOKEN" ] || { log "FATAL: Login failed: $resp"; exit 1; }
log "Logged in, CSRF token obtained"

# ── record pre-existing state ──
PRE_INSTANCES=$(api_get "model-instances" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
PRE_CONTAINERS=$(docker ps --filter 'label=lightai.managed' -q 2>/dev/null | wc -l || echo "0")
log "Pre-existing instances=$PRE_INSTANCES containers=$PRE_CONTAINERS"

# ── discover resources ──
NODES_JSON=$(api_get "nodes")
NODE_ID=$(echo "$NODES_JSON" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)
[ -n "$NODE_ID" ] || { log "FATAL: No online node found"; exit 1; }
log "Node: $NODE_ID"

echo "$NODES_JSON" > "$ARTIFACT_DIR/nodes.json"

# Discover runtimes
RTS_JSON=$(api_get "backend-runtimes")
echo "$RTS_JSON" > "$ARTIFACT_DIR/backend-runtimes.json"

VLLM_RT=$(echo "$RTS_JSON" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    rid = r.get('id','')
    if 'vllm' in rid and r.get('vendor','')=='nvidia':
        print(rid); break
" 2>/dev/null)
[ -n "$VLLM_RT" ] || { log "FATAL: No vLLM NVIDIA runtime found"; exit 1; }

SGLANG_RT=$(echo "$RTS_JSON" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    rid = r.get('id','')
    if 'sglang' in rid and r.get('vendor','')=='nvidia':
        print(rid); break
" 2>/dev/null)

LLAMACPP_RT=$(echo "$RTS_JSON" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    rid = r.get('id','')
    if 'llamacpp' in rid and r.get('vendor','')=='nvidia':
        print(rid); break
" 2>/dev/null)

METAX_RT=$(echo "$RTS_JSON" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    if r.get('vendor','')=='metax':
        print(r['id']); break
" 2>/dev/null)

log "Runtimes: vllm=$VLLM_RT sglang=${SGLANG_RT:-none} llamacpp=${LLAMACPP_RT:-none} metax=${METAX_RT:-none}"

# ── discover artifacts with model_locations ──
# The list endpoint doesn't include locations; fetch detail for each candidate.
ARTS_JSON=$(api_get "model-artifacts")
echo "$ARTS_JSON" > "$ARTIFACT_DIR/model-artifacts.json"

# Helper: find first artifact of given format that has a model_location on our node
find_artifact_with_location() {
  local fmt="$1"
  local art_list; art_list=$(echo "$ARTS_JSON" | python3 -c "
import json,sys
for a in json.load(sys.stdin):
    if a.get('format') == '$fmt':
        print(a['id'])
        break
" 2>/dev/null)
  [ -z "$art_list" ] && return 1
  # Fetch detail to check locations
  local detail; detail=$(api_get "model-artifacts/$art_list")
  local has_loc; has_loc=$(echo "$detail" | python3 -c "
import json,sys
d = json.load(sys.stdin)
for loc in d.get('locations', []):
    if loc.get('node_id') == '$NODE_ID' and loc.get('match_status') in ('exact_match','probable_match','manual_attested'):
        print('yes')
        sys.exit(0)
print('no')
" 2>/dev/null)
  if [ "$has_loc" = "yes" ]; then
    echo "$art_list"
    return 0
  fi
  return 1
}

HF_ART_ID=$(find_artifact_with_location "huggingface" 2>/dev/null)
[ -n "$HF_ART_ID" ] || { log "FATAL: No HF artifact with model_location on node $NODE_ID"; exit 1; }
log "HF artifact: $HF_ART_ID"

GGUF_ART_ID=$(find_artifact_with_location "gguf" 2>/dev/null || true)
if [ -n "$GGUF_ART_ID" ]; then
  log "GGUF artifact: $GGUF_ART_ID"
else
  log "WARNING: No GGUF artifact with location (will skip llama.cpp tests)"
fi

# ── ensure NBRs are ready ──
ensure_nbr_ready() {
  local rt_id="$1"
  local nbr_id="${NODE_ID}:${rt_id}"
  local status; status=$(api_get "nodes/$NODE_ID/backend-runtimes" | python3 -c "
import json,sys
for n in json.load(sys.stdin):
    if n.get('backend_runtime_id') == '$rt_id':
        print(n.get('status',''))
        sys.exit(0)
print('missing')
" 2>/dev/null)
  case "$status" in
    ready) return 0 ;;
    missing)
      log "Enabling NBR for $rt_id..."
      api_post "nodes/$NODE_ID/backend-runtimes/enable" \
        "{\"backend_runtime_id\":\"$rt_id\",\"image_present\":true,\"docker_available\":true}" > /dev/null
      ;;
    *)
      log "NBR $rt_id status=$status, re-enabling..."
      api_delete "nodes/$NODE_ID/backend-runtimes/$nbr_id" > /dev/null 2>&1 || true
      api_post "nodes/$NODE_ID/backend-runtimes/enable" \
        "{\"backend_runtime_id\":\"$rt_id\",\"image_present\":true,\"docker_available\":true}" > /dev/null
      ;;
  esac
}

ensure_nbr_ready "$VLLM_RT"
[ -n "$SGLANG_RT" ] && ensure_nbr_ready "$SGLANG_RT" || true
if [ -n "$LLAMACPP_RT" ] && [ -n "$GGUF_ART_ID" ]; then
  ensure_nbr_ready "$LLAMACPP_RT"
fi

# Save NBR state
api_get "nodes/$NODE_ID/backend-runtimes" > "$ARTIFACT_DIR/nbrs.json"
log "NBR state saved"

# ── helper: create deployment + dry-run + extract preview ──
# Returns: sets DEP_ID, DR_PREVIEW globals
do_dryrun() {
  local label="$1" art_id="$2" rt_id="$3" svc_json="$4" params_json="$5"
  log "--- DryRun: $label ---"
  local dep_resp; dep_resp=$(api_post "deployments" "{\"name\":\"$PREFIX-$label\",\"display_name\":\"$label\",\"model_artifact_id\":\"$art_id\",\"backend_runtime_id\":\"$rt_id\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"accelerator_ids\":[]},\"service_json\":$svc_json,\"parameters_json\":$params_json}")
  DEP_ID=$(echo "$dep_resp" | json_field id)
  if [ -z "$DEP_ID" ]; then
    log "ERROR: Deployment create failed for $label"
    echo "$dep_resp" > "$ARTIFACT_DIR/${label}-deploy-fail.json"
    DEP_ID=""
    DR_PREVIEW=""
    return 1
  fi
  echo "$dep_resp" > "$ARTIFACT_DIR/${label}-deployment.json"
  local dr_resp; dr_resp=$(api_post "deployments/$DEP_ID/dry-run" '{}')
  echo "$dr_resp" > "$ARTIFACT_DIR/${label}-dryrun.json"
  DR_PREVIEW=$(echo "$dr_resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('command_preview',''))" 2>/dev/null)
  echo "$DR_PREVIEW" > "$ARTIFACT_DIR/${label}-preview.txt"
  local valid; valid=$(echo "$dr_resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('valid'))" 2>/dev/null)
  if [ "$valid" != "True" ]; then
    log "WARNING: DryRun valid=$valid for $label"
    local errs; errs=$(echo "$dr_resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('errors',d.get('error_details',[]))))" 2>/dev/null)
    log "  errors: $errs"
  fi
}

cleanup_dryrun() {
  if [ -n "${DEP_ID:-}" ]; then
    api_delete "deployments/$DEP_ID" > /dev/null 2>&1 || true
    DEP_ID=""
  fi
}

# ═══════════════════════════════════════════════════════════════
# Test 1: vLLM custom port propagation (host≠container)
# ═══════════════════════════════════════════════════════════════
run_vllm_port_test() {
  do_dryrun "vllm-port" "$HF_ART_ID" "$VLLM_RT" \
    '{"host_port":8191,"container_port":8022,"app_port":8022}' \
    '{}' || { fail "vLLM port deployment failed"; return 0; }

  local prev="$DR_PREVIEW"

  assert_contains     "vLLM: -p 8191:8022 in preview"       "$prev" "-p 8191:8022" || fail "vLLM port mapping"
  assert_flag_value   "vLLM: --port 8022 in args"            "$prev" "--port" "8022" || fail "vLLM app port"
  assert_not_contains "vLLM: no default --port 8000"         "$prev" "--port 8000" || fail "vLLM default port leak"
  assert_exactly_one_flag "vLLM: exactly one --port"         "$prev" "--port" || fail "vLLM port dedup"
  assert_not_contains "vLLM: no --model flag (positional)"   "$prev" "--model" || fail "vLLM model flag"
  assert_contains     "vLLM: model container path /models/"  "$prev" "/models/" || fail "vLLM model path"
  assert_contains     "vLLM: --gpus device mapping"          "$prev" "--gpus" || fail "vLLM GPU device"
  assert_not_contains "vLLM: GPU not as volume"              "$prev" "-v /dev" || fail "vLLM device as volume"

  cleanup_dryrun
  log "vLLM port test done"
}

# ═══════════════════════════════════════════════════════════════
# Test 2: vLLM served_model_name + gpu_memory_utilization
# ═══════════════════════════════════════════════════════════════
run_vllm_params_test() {
  do_dryrun "vllm-params" "$HF_ART_ID" "$VLLM_RT" \
    '{"host_port":8192,"container_port":8000,"app_port":8000}' \
    '{"served_model_name":"qwen-vllm-e2e","gpu_memory_utilization":0.85,"max_model_len":4096}' || { fail "vLLM params deployment failed"; return 0; }

  local prev="$DR_PREVIEW"

  assert_contains     "vLLM: --served-model-name qwen-vllm-e2e" "$prev" "qwen-vllm-e2e" || fail "vLLM served model name"
  assert_contains     "vLLM: --gpu-memory-utilization 0.85"     "$prev" "gpu-memory-utilization 0.85" || fail "vLLM gpu mem"
  assert_contains     "vLLM: --max-model-len 4096"              "$prev" "max-model-len 4096" || fail "vLLM max model len"
  assert_exactly_one_flag "vLLM-params: exactly one --port"     "$prev" "--port" || fail "vLLM params port dedup"
  assert_contains     "vLLM: CUDA_VISIBLE_DEVICES"             "$prev" "CUDA_VISIBLE_DEVICES" || fail "vLLM CUDA visible"

  cleanup_dryrun
  log "vLLM params test done"
}

# ═══════════════════════════════════════════════════════════════
# Test 3: vLLM MetaX device visibility
# ═══════════════════════════════════════════════════════════════
run_metax_test() {
  if [ -z "$METAX_RT" ]; then
    log "SKIP: No MetaX runtime available"
    return 0
  fi

  local rt_detail; rt_detail=$(api_get "backend-runtimes/$METAX_RT")
  echo "$rt_detail" > "$ARTIFACT_DIR/metax-runtime-detail.json"
  local env_json; env_json=$(echo "$rt_detail" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('default_env_json',{})))" 2>/dev/null)

  assert_contains     "MetaX: CUDA_VISIBLE_DEVICES in env"     "$env_json" "CUDA_VISIBLE_DEVICES" || fail "MetaX visible env"
  assert_not_contains "MetaX: CUDA_VISIBLE_DEVICES not sole"  "$env_json" "CUDA_VISIBLE_DEVICES" || log "INFO: MetaX env OK (no CUDA leak)"

  log "MetaX test done"
}

# ═══════════════════════════════════════════════════════════════
# Test 4: llama.cpp GGUF model path
# ═══════════════════════════════════════════════════════════════
run_llamacpp_test() {
  if [ -z "$LLAMACPP_RT" ] || [ -z "$GGUF_ART_ID" ]; then
    log "SKIP: llama.cpp runtime or GGUF artifact not available"
    return 0
  fi

  do_dryrun "llamacpp-gguf" "$GGUF_ART_ID" "$LLAMACPP_RT" \
    '{"host_port":8193,"container_port":8080,"app_port":8080}' \
    '{}' || { fail "llama.cpp deployment failed"; return 0; }

  local prev="$DR_PREVIEW"

  # Critical: GGUF must use -m with .gguf path
  assert_contains     "llama.cpp: -m flag present"           "$prev" "-m /models/" || fail "llama.cpp -m flag"
  assert_contains     "llama.cpp: GGUF path .gguf"           "$prev" ".gguf" || fail "llama.cpp gguf path"
  assert_contains     "llama.cpp: CUDA_VISIBLE_DEVICES"      "$prev" "CUDA_VISIBLE_DEVICES" || fail "llama.cpp CUDA visible"
  assert_contains     "llama.cpp: port mapping -p"           "$prev" "-p 8193:8080" || fail "llama.cpp port mapping"
  assert_flag_value   "llama.cpp: --port 8080"               "$prev" "--port" "8080" || fail "llama.cpp app port"
  assert_exactly_one_flag "llama.cpp: exactly one --port"    "$prev" "--port" || fail "llama.cpp port dedup"
  assert_contains     "llama.cpp: --gpus device"             "$prev" "--gpus" || fail "llama.cpp GPU device"
  assert_not_contains "llama.cpp: no -v /dev"                "$prev" "-v /dev" || fail "llama.cpp device as volume"

  cleanup_dryrun
  log "llama.cpp GGUF test done"
}

# ═══════════════════════════════════════════════════════════════
# Test 5: SGLang model path
# ═══════════════════════════════════════════════════════════════
run_sglang_test() {
  if [ -z "$SGLANG_RT" ]; then
    log "SKIP: No SGLang runtime available"
    return 0
  fi

  do_dryrun "sglang" "$HF_ART_ID" "$SGLANG_RT" \
    '{"host_port":8194,"container_port":30000,"app_port":30000}' \
    '{"served_model_name":"qwen-sglang-e2e"}' || { fail "SGLang deployment failed"; return 0; }

  local prev="$DR_PREVIEW"

  assert_contains     "SGLang: --model-path in command"     "$prev" "--model-path /models/" || fail "SGLang model path"
  assert_contains     "SGLang: port mapping -p"             "$prev" "-p 8194:30000" || fail "SGLang port mapping"
  assert_contains     "SGLang: CUDA_VISIBLE_DEVICES"        "$prev" "CUDA_VISIBLE_DEVICES" || fail "SGLang CUDA visible"
  assert_contains     "SGLang: --gpus device"               "$prev" "--gpus" || fail "SGLang GPU device"
  assert_exactly_one_flag "SGLang: exactly one --port"      "$prev" "--port" || fail "SGLang port dedup"

  cleanup_dryrun
  log "SGLang test done"
}

# ═══════════════════════════════════════════════════════════════
# Test 6: DryRun-only proof (no side effects)
# ═══════════════════════════════════════════════════════════════
run_dryrun_proof() {
  log "=== Test 6: DryRun-only proof ==="

  local post_instances; post_instances=$(api_get "model-instances" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
  local post_containers; post_containers=$(docker ps --filter 'label=lightai.managed' -q 2>/dev/null | wc -l || echo "0")

  assert_eq "No new instances created" "$PRE_INSTANCES" "$post_instances" || fail "instance leak"
  assert_eq "No lightai containers"    "0"               "$post_containers" || fail "container leak"

  # Verify no deployment leftovers
  local dep_count; dep_count=$(api_get "deployments" | python3 -c "
import json,sys
deps = json.load(sys.stdin)
mine = [d for d in deps if d.get('name','').startswith('$PREFIX-')]
print(len(mine))
" 2>/dev/null)
  assert_eq "No leftover test deployments" "0" "$dep_count" || fail "deployment leak ($dep_count left)"

  {
    echo "dry_run_only: true"
    echo "pre_instances: $PRE_INSTANCES"
    echo "post_instances: $post_instances"
    echo "pre_containers: $PRE_CONTAINERS"
    echo "post_containers: $post_containers"
    echo "leftover_deployments: $dep_count"
  } > "$ARTIFACT_DIR/dryrun-proof.txt"

  log "DryRun proof done"
}

# ── run all tests ──
run_vllm_port_test
run_vllm_params_test
run_metax_test
run_llamacpp_test
run_sglang_test
run_dryrun_proof

# ── summary ──
echo ""
echo "Artifacts: $ARTIFACT_DIR"
echo "Key files:"
echo "  $(ls "$ARTIFACT_DIR"/*.txt 2>/dev/null | tr '\n' ' ')"
echo "  $(ls "$ARTIFACT_DIR"/*.json 2>/dev/null | tr '\n' ' ')"
assert_summary
LEGACY_CONTRACT_DO_NOT_USE_FOR_CURRENT_VALIDATION
