#!/bin/bash
# e2e-runplan-parameter-source-audit.sh — DryRun parameter propagation audit.
# Category: DryRun E2E (no containers, no GPU usage beyond API queries)
# Requires: running LightAI server at SERVER_URL
# Does NOT start containers or create instances.
# Verifies: user-set parameters flow through to RunPlan/Docker command preview.

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
FAILED=0

mkdir -p "$ARTIFACT_DIR"

log()   { printf '[%s] [dryrun-audit] %s\n' "$(date '+%H:%M:%S')" "$*"; }
fail()  { log "FAIL: $*"; FAILED=1; }
abort() { log "FATAL: $*"; exit 1; }

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
[ -n "$CSRF_TOKEN" ] || abort "Login failed: $resp"
log "Logged in, CSRF token obtained"

# ── record pre-existing state ──
PRE_INSTANCES=$(api_get "model-instances" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
PRE_CONTAINERS=$(docker ps --filter 'label=lightai.managed' -q 2>/dev/null | wc -l)
log "Pre-existing instances=$PRE_INSTANCES containers=$PRE_CONTAINERS (should be 0 for clean dry-run)"

# ── query resources ──
NODE_ID=$(api_get "nodes" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)
[ -n "$NODE_ID" ] || abort "No online node found"
log "Node: $NODE_ID"

# Find vLLM runtime
VLLM_RT=$(api_get "backend-runtimes" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    if 'vllm' in r.get('backend_id','') and r.get('vendor','')=='nvidia':
        print(r['id']); break
" 2>/dev/null)
[ -n "$VLLM_RT" ] || abort "No vLLM NVIDIA runtime found"

# Find SGLang runtime
SGLANG_RT=$(api_get "backend-runtimes" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    if 'sglang' in r.get('backend_id','') and r.get('vendor','')=='nvidia':
        print(r['id']); break
" 2>/dev/null)
[ -n "$SGLANG_RT" ] || log "WARNING: No SGLang NVIDIA runtime (will skip SGLang tests)"

# Find llama.cpp runtime
LLAMACPP_RT=$(api_get "backend-runtimes" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    if 'llamacpp' in r.get('backend_id','') and r.get('vendor','')=='nvidia':
        print(r['id']); break
" 2>/dev/null)
[ -n "$LLAMACPP_RT" ] || log "WARNING: No llama.cpp NVIDIA runtime (will skip llama.cpp tests)"

log "Runtimes: vllm=$VLLM_RT sglang=${SGLANG_RT:-none} llamacpp=${LLAMACPP_RT:-none}"

# ─────────────────────────────────────────────────────────────
# Test 1: vLLM custom port propagation
# ─────────────────────────────────────────────────────────────
run_vllm_port_test() {
  log "=== Test 1: vLLM custom port propagation ==="

  # Create artifact
  local art_resp; art_resp=$(api_post "model-artifacts" "{\"name\":\"$PREFIX-vllm-port\",\"path\":\"/home/kzeng/models/Qwen3-0.6B-Instruct-2512\",\"format\":\"huggingface\",\"task_type\":\"chat\"}")
  local art_id; art_id=$(echo "$art_resp" | json_field id)
  [ -n "$art_id" ] || { fail "vLLM artifact create failed"; return 1; }
  echo "$art_resp" > "$ARTIFACT_DIR/vllm-port-artifact.json"

  # Create deployment with custom ports
  local dep_resp; dep_resp=$(api_post "deployments" "{\"name\":\"$PREFIX-vllm-port\",\"display_name\":\"VLLM Port Test\",\"model_artifact_id\":\"$art_id\",\"backend_runtime_id\":\"$VLLM_RT\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8111,\"container_port\":8022,\"app_port\":8022},\"parameters_json\":{}}")
  local dep_id; dep_id=$(echo "$dep_resp" | json_field id)
  [ -n "$dep_id" ] || { fail "vLLM deployment create failed"; return 1; }
  echo "$dep_resp" > "$ARTIFACT_DIR/vllm-port-deployment.json"

  # DryRun
  local dr_resp; dr_resp=$(api_post  "deployments/$dep_id/dry-run" '{}')
  echo "$dr_resp" > "$ARTIFACT_DIR/vllm-port-dryrun.json"
  local preview; preview=$(echo "$dr_resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('docker_preview',''))" 2>/dev/null)
  echo "$preview" > "$ARTIFACT_DIR/vllm-port-preview.txt"

  # Assertions
  assert_contains     "vLLM: -p 8111:8022 in preview"        "$preview" "-p 8111:8022" || fail "port mapping"
  assert_flag_value   "vLLM: --port 8022 in args"             "$preview" "--port" "8022" || fail "app port"
  assert_not_contains "vLLM: no default --port 8000"          "$preview" "--port 8000" || fail "default port leak"
  assert_exactly_one_flag "vLLM: exactly one --port"          "$preview" "--port" || fail "port dedup"
  assert_not_contains "vLLM: no --model flag (positional)"    "$preview" "--model" || fail "model flag"
  assert_contains     "vLLM: model path in command"           "$preview" "/models/" || fail "model path"

  # Cleanup
  api_delete "deployments/$dep_id" > /dev/null 2>&1 || true
  api_delete "model-artifacts/$art_id" > /dev/null 2>&1 || true
  log "vLLM port test done"
}

# ─────────────────────────────────────────────────────────────
# Test 2: vLLM served_model_name + gpu_memory_utilization
# ─────────────────────────────────────────────────────────────
run_vllm_params_test() {
  log "=== Test 2: vLLM custom parameters ==="

  local art_resp; art_resp=$(api_post "model-artifacts" "{\"name\":\"$PREFIX-vllm-params\",\"path\":\"/home/kzeng/models/Qwen3-0.6B-Instruct-2512\",\"format\":\"huggingface\",\"task_type\":\"chat\"}")
  local art_id; art_id=$(echo "$art_resp" | json_field id)
  [ -n "$art_id" ] || { fail "vLLM params artifact create failed"; return 1; }
  echo "$art_resp" > "$ARTIFACT_DIR/vllm-params-artifact.json"

  local dep_resp; dep_resp=$(api_post "deployments" "{\"name\":\"$PREFIX-vllm-params\",\"display_name\":\"VLLM Params Test\",\"model_artifact_id\":\"$art_id\",\"backend_runtime_id\":\"$VLLM_RT\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8005,\"container_port\":8000,\"app_port\":8000},\"parameters_json\":{\"served_model_name\":\"qwen-vllm-e2e\",\"gpu_memory_utilization\":0.85,\"max_model_len\":4096}}")
  local dep_id; dep_id=$(echo "$dep_resp" | json_field id)
  [ -n "$dep_id" ] || { fail "vLLM params deployment create failed"; return 1; }
  echo "$dep_resp" > "$ARTIFACT_DIR/vllm-params-deployment.json"

  local dr_resp; dr_resp=$(api_post  "deployments/$dep_id/dry-run" '{}')
  echo "$dr_resp" > "$ARTIFACT_DIR/vllm-params-dryrun.json"
  local preview; preview=$(echo "$dr_resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('docker_preview',''))" 2>/dev/null)
  echo "$preview" > "$ARTIFACT_DIR/vllm-params-preview.txt"

  assert_contains     "vLLM: --served-model-name qwen-vllm-e2e" "$preview" "qwen-vllm-e2e" || fail "served model name"
  assert_contains     "vLLM: --gpu-memory-utilization 0.85"     "$preview" "gpu-memory-utilization 0.85" || fail "gpu mem"
  assert_contains     "vLLM: --max-model-len 4096"              "$preview" "max-model-len 4096" || fail "max model len"
  assert_exactly_one_flag "vLLM: exactly one --port"            "$preview" "--port" || fail "port dedup"

  api_delete "deployments/$dep_id" > /dev/null 2>&1 || true
  api_delete "model-artifacts/$art_id" > /dev/null 2>&1 || true
  log "vLLM params test done"
}

# ─────────────────────────────────────────────────────────────
# Test 3: vLLM MetaX device visibility
# ─────────────────────────────────────────────────────────────
run_metax_test() {
  log "=== Test 3: MetaX device visibility ==="

  # Find MetaX runtime
  local metax_rt; metax_rt=$(api_get "backend-runtimes" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    if r.get('vendor','')=='metax':
        print(r['id']); break
" 2>/dev/null)
  if [ -z "$metax_rt" ]; then
    log "SKIP: No MetaX runtime available"
    return 0
  fi

  # Get runtime detail to check env
  local rt_detail; rt_detail=$(api_get "backend-runtimes/$metax_rt")
  echo "$rt_detail" > "$ARTIFACT_DIR/metax-runtime-detail.json"
  local env_json; env_json=$(echo "$rt_detail" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('default_env_json',{})))" 2>/dev/null)

  assert_contains     "MetaX: MACA_VISIBLE_DEVICE in env"     "$env_json" "MACA_VISIBLE_DEVICE" || fail "MetaX visible env"
  assert_not_contains "MetaX: CUDA_VISIBLE_DEVICES not sole"  "$env_json" "CUDA_VISIBLE_DEVICES" || log "INFO: check MetaX env (may have CUDA as secondary)"

  log "MetaX test done"
}

# ─────────────────────────────────────────────────────────────
# Test 4: DryRun-only proof
# ─────────────────────────────────────────────────────────────
run_dryrun_proof() {
  log "=== Test 4: DryRun-only proof ==="

  local post_instances; post_instances=$(api_get "model-instances" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
  local post_containers; post_containers=$(docker ps --filter 'label=lightai.managed' -q 2>/dev/null | wc -l)

  assert_eq "No new instances created" "$PRE_INSTANCES" "$post_instances" || fail "instance leak"
  assert_eq "No lightai containers"    "0"               "$post_containers" || fail "container leak"

  echo "dry_run_only: true" > "$ARTIFACT_DIR/dryrun-proof.txt"
  echo "pre_instances: $PRE_INSTANCES" >> "$ARTIFACT_DIR/dryrun-proof.txt"
  echo "post_instances: $post_instances" >> "$ARTIFACT_DIR/dryrun-proof.txt"
  echo "pre_containers: $PRE_CONTAINERS" >> "$ARTIFACT_DIR/dryrun-proof.txt"
  echo "post_containers: $post_containers" >> "$ARTIFACT_DIR/dryrun-proof.txt"

  log "DryRun proof done"
}

# ── run all tests ──
run_vllm_port_test
run_vllm_params_test
run_metax_test
run_dryrun_proof

# ── summary ──
echo ""
echo "Artifacts: $ARTIFACT_DIR"
assert_summary
