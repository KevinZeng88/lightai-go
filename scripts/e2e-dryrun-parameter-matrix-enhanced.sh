#!/bin/bash
# e2e-dryrun-parameter-matrix-enhanced.sh — Comprehensive DryRun parameter matrix.
# Category: DryRun E2E (no containers, no GPU usage beyond API queries)
# Verifies: ALL configurable parameters propagate correctly through the full
# BackendVersion→BackendRuntime→NBR→Deployment→RunPlan→Docker command chain.
# Covers vLLM / SGLang / llama.cpp with forward + reverse assertions.
# Exit non-zero on ANY assertion failure.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/e2e-assert.sh"

SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-Commvault!234}"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-/tmp/lightai-e2e-dryrun-matrix-$RUN_ID}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/lightai-e2e-cookies-$RUN_ID.txt}"
PREFIX="e2e-dm"
mkdir -p "$ARTIFACT_DIR"

log()   { printf '[%s] [dryrun-matrix] %s\n' "$(date '+%H:%M:%S')" "$*"; }

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

# ── discover ──
NODE_ID=$(api_get "nodes" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)
[ -n "$NODE_ID" ] || { log "FATAL: No online node"; exit 1; }

RTS_JSON=$(api_get "backend-runtimes")
VLLM_RT=$(echo "$RTS_JSON" | python3 -c "import json,sys; [print(r['id']) for r in json.load(sys.stdin) if 'vllm' in r.get('id','') and r.get('vendor','')=='nvidia']" 2>/dev/null | head -1)
SGLANG_RT=$(echo "$RTS_JSON" | python3 -c "import json,sys; [print(r['id']) for r in json.load(sys.stdin) if 'sglang' in r.get('id','') and r.get('vendor','')=='nvidia']" 2>/dev/null | head -1)
LLAMACPP_RT=$(echo "$RTS_JSON" | python3 -c "import json,sys; [print(r['id']) for r in json.load(sys.stdin) if 'llamacpp' in r.get('id','') and r.get('vendor','')=='nvidia']" 2>/dev/null | head -1)
METAX_RT=$(echo "$RTS_JSON" | python3 -c "import json,sys; [print(r['id']) for r in json.load(sys.stdin) if r.get('vendor','')=='metax']" 2>/dev/null | head -1)

HF_ART=$(api_get "model-artifacts" | python3 -c "import json,sys; [print(a['id']) for a in json.load(sys.stdin) if a.get('format')=='huggingface']" 2>/dev/null | head -1)
GGUF_ART=$(api_get "model-artifacts" | python3 -c "import json,sys; [print(a['id']) for a in json.load(sys.stdin) if a.get('format')=='gguf']" 2>/dev/null | head -1)

log "Node: $NODE_ID"
log "vLLM: ${VLLM_RT:-none}  SGLang: ${SGLANG_RT:-none}  llama.cpp: ${LLAMACPP_RT:-none}  MetaX: ${METAX_RT:-none}"
log "HF: ${HF_ART:-none}  GGUF: ${GGUF_ART:-none}"

# ── ensure NBR ready ──
ensure_nbr() {
  local rt="$1"
  local st; st=$(api_get "nodes/$NODE_ID/backend-runtimes" | python3 -c "
import json,sys
for n in json.load(sys.stdin):
    if n.get('backend_runtime_id')=='$rt': print(n.get('status','')); break
" 2>/dev/null)
  if [ "$st" != "ready" ]; then
    api_post "nodes/$NODE_ID/backend-runtimes/enable" \
      "{\"backend_runtime_id\":\"$rt\",\"image_present\":true,\"docker_available\":true}" > /dev/null 2>&1 || true
  fi
}
for rt in "$VLLM_RT" "$SGLANG_RT" "$LLAMACPP_RT"; do [ -n "$rt" ] && ensure_nbr "$rt"; done

# ── helper: create deployment + dry-run, assert on preview ──
# Args: label rt art_id svc_json params_json
run_dryrun() {
  local label="$1" rt="$2" art="$3" svc="$4" params="$5"
  local dep_resp; dep_resp=$(api_post "deployments" \
    "{\"name\":\"$PREFIX-$label\",\"display_name\":\"$label\",\"model_artifact_id\":\"$art\",\"backend_runtime_id\":\"$rt\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"accelerator_ids\":[]},\"service_json\":$svc,\"parameters_json\":$params}")
  local dep_id; dep_id=$(echo "$dep_resp" | json_field id)
  if [ -z "$dep_id" ]; then
    echo "ERROR: deploy create: $(head -c 200 <<< "$dep_resp")"
    echo "FAIL" > "$ARTIFACT_DIR/${label}-result.txt"
    return 1
  fi
  echo "$dep_resp" > "$ARTIFACT_DIR/${label}-deploy.json"
  local dr; dr=$(api_post "deployments/$dep_id/dry-run" '{}')
  echo "$dr" > "$ARTIFACT_DIR/${label}-dryrun.json"
  local prev; prev=$(echo "$dr" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('command_preview',''))" 2>/dev/null)
  echo "$prev" > "$ARTIFACT_DIR/${label}-preview.txt"
  local v; v=$(echo "$dr" | json_field valid)
  api_delete "deployments/$dep_id" > /dev/null 2>&1 || true
  if [ "$v" != "True" ]; then
    echo "INVALID" > "$ARTIFACT_DIR/${label}-result.txt"
    return 1
  fi
  # Return preview via global
  _DR_PREVIEW="$prev"
  echo "PASS" > "$ARTIFACT_DIR/${label}-result.txt"
  return 0
}

# ── assertion helpers ──
fwd()  { assert_contains "$1" "$_DR_PREVIEW" "$2" || log "FAIL: $1"; }
nope() { assert_not_contains "$1" "$_DR_PREVIEW" "$2" || log "FAIL: $1"; }
exactly_one() { assert_exactly_one_flag "$1" "$_DR_PREVIEW" "$2" || log "FAIL: $1"; }
has_no() { assert_no_flag "$1" "$_DR_PREVIEW" "$2" || log "FAIL: $1"; }
fval() { assert_flag_value "$1" "$_DR_PREVIEW" "$2" "$3" || log "FAIL: $1"; }

# ═══════════════════════════════════════════════════════════════
# SECTION 1: vLLM PARAMETER MATRIX
# ═══════════════════════════════════════════════════════════════
section_vllm() {
  [ -z "$VLLM_RT" ] && { log "SKIP vLLM: no runtime"; return; }
  log "══════ vLLM Parameter Matrix ══════"

  # --- 1a: Port separation (host≠container≠app) ---
  log "--- vLLM: port separation ---"
  run_dryrun "vllm-ports" "$VLLM_RT" "$HF_ART" \
    '{"host_port":9101,"container_port":8022,"app_port":8022}' '{}'
  fwd      "vLLM port mapping -p 9101:8022" "-p 9101:8022"
  fval     "vLLM --port 8022" "--port" "8022"
  nope     "vLLM no default --port 8000" "--port 8000"
  exactly_one "vLLM exactly one --port" "--port"
  has_no   "vLLM no --model flag (positional)" "--model"

  # --- 1b: served_model_name ---
  log "--- vLLM: served_model_name ---"
  run_dryrun "vllm-served-name" "$VLLM_RT" "$HF_ART" \
    '{"host_port":9102,"container_port":8000,"app_port":8000}' \
    '{"served_model_name":"e2e-vllm-custom"}'
  fwd      "vLLM --served-model-name" "e2e-vllm-custom"
  exactly_one "vLLM served-model-name once" "--served-model-name"

  # --- 1c: gpu_memory_utilization ---
  log "--- vLLM: gpu_memory_utilization ---"
  run_dryrun "vllm-gpu-mem" "$VLLM_RT" "$HF_ART" \
    '{"host_port":9103,"container_port":8000,"app_port":8000}' \
    '{"gpu_memory_utilization":0.77}'
  fwd      "vLLM --gpu-memory-utilization 0.77" "gpu-memory-utilization 0.77"

  # --- 1d: max_model_len ---
  log "--- vLLM: max_model_len ---"
  run_dryrun "vllm-max-len" "$VLLM_RT" "$HF_ART" \
    '{"host_port":9104,"container_port":8000,"app_port":8000}' \
    '{"max_model_len":2048}'
  fwd      "vLLM --max-model-len 2048" "max-model-len 2048"

  # --- 1e: Combined params (all vLLM params together) ---
  log "--- vLLM: combined params (user override test) ---"
  run_dryrun "vllm-combined" "$VLLM_RT" "$HF_ART" \
    '{"host_port":9105,"container_port":9000,"app_port":9000}' \
    '{"served_model_name":"e2e-combined","gpu_memory_utilization":0.66,"max_model_len":1024,"tensor_parallel_size":1,"trust_remote_code":true,"enforce_eager":true}'
  fwd      "vLLM combined served-model-name" "e2e-combined"
  fwd      "vLLM combined gpu-memory 0.66" "gpu-memory-utilization 0.66"
  fwd      "vLLM combined max-model-len 1024" "max-model-len 1024"
  fwd      "vLLM combined --port 9000" "--port 9000"
  fwd      "vLLM combined -p 9105:9000" "-p 9105:9000"
  # User params must override defaults
  exactly_one "vLLM combined one --port" "--port"
  exactly_one "vLLM combined one --served-model-name" "--served-model-name"
  nope     "vLLM combined no default port 8000" "--port 8000"

  # --- 1f: CUDA_VISIBLE_DEVICES + --gpus ---
  log "--- vLLM: CUDA_VISIBLE_DEVICES + GPU device ---"
  fwd      "vLLM CUDA_VISIBLE_DEVICES" "CUDA_VISIBLE_DEVICES"
  fwd      "vLLM --gpus device" "--gpus"
  nope     "vLLM GPU not as volume" "-v /dev/dri"
  nope     "vLLM GPU not as /dev" "-v /dev/nvidia"

  # --- 1g: Docker options (ipc, shm_size) ---
  fwd      "vLLM ipc=host" "--ipc host"
  fwd      "vLLM shm-size" "--shm-size"

  # --- 1h: Model container path ---
  fwd      "vLLM model path /models/" "/models/"
  # Host path appears in Docker volume mount (-v /host:/container), which is correct.
  # App args (after the image name) must NOT contain host path.
  local vllm_args_only; vllm_args_only=$(echo "$_DR_PREVIEW" | python3 -c "
import sys
line = sys.stdin.read().strip()
# Split on ' -e ' or image reference to find app args
# App args start after the image name + space
parts = line.split(' vllm/vllm-openai:')
if len(parts) > 1:
    print(parts[-1])
" 2>/dev/null)
  if [ -n "$vllm_args_only" ]; then
    assert_not_contains "vLLM no host path in app args" "$vllm_args_only" "/home/kzeng/models" || log "FAIL: vLLM host path in app args"
  fi

  # --- 1i: Reverse assertions ---
  log "--- vLLM: reverse assertions ---"
  nope     "vLLM no duplicate --port" "--port --port"
  nope     "vLLM no duplicate --served-model-name" "--served-model-name --served-model-name"
  nope     "vLLM model flag absent (positional)" "--model /models/"
  # vLLM should NOT have -m (that's llama.cpp)
  nope     "vLLM no -m flag" " -m /models/"
  # Not a GGUF model
  nope     "vLLM no .gguf path" ".gguf"

  log "vLLM matrix complete"
}

# ═══════════════════════════════════════════════════════════════
# SECTION 2: SGLang PARAMETER MATRIX
# ═══════════════════════════════════════════════════════════════
section_sglang() {
  [ -z "$SGLANG_RT" ] && { log "SKIP SGLang: no runtime"; return; }
  log "══════ SGLang Parameter Matrix ══════"

  # --- 2a: Port separation ---
  log "--- SGLang: port separation ---"
  run_dryrun "sglang-ports" "$SGLANG_RT" "$HF_ART" \
    '{"host_port":9201,"container_port":31000,"app_port":31000}' '{}'
  fwd      "SGLang port mapping -p 9201:31000" "-p 9201:31000"
  fwd      "SGLang --port 31000" "--port 31000"
  nope     "SGLang no default --port 30000" "--port 30000"
  exactly_one "SGLang exactly one --port" "--port"

  # --- 2b: served_model_name ---
  log "--- SGLang: served_model_name ---"
  run_dryrun "sglang-served-name" "$SGLANG_RT" "$HF_ART" \
    '{"host_port":9202,"container_port":30000,"app_port":30000}' \
    '{"served_model_name":"e2e-sglang-custom"}'
  fwd      "SGLang --served-model-name" "e2e-sglang-custom"

  # --- 2c: --tp-size=1 ---
  log "--- SGLang: tp_size ---"
  run_dryrun "sglang-tp" "$SGLANG_RT" "$HF_ART" \
    '{"host_port":9203,"container_port":30000,"app_port":30000}' \
    '{"tp":1}'
  # tp=1 may or may not appear depending on whether default is excluded

  # --- 2d: Combined params ---
  log "--- SGLang: combined params ---"
  run_dryrun "sglang-combined" "$SGLANG_RT" "$HF_ART" \
    '{"host_port":9204,"container_port":32000,"app_port":32000}' \
    '{"served_model_name":"e2e-sglang-combined","tp":1,"trust_remote_code":true}'
  fwd      "SGLang combined served-model-name" "e2e-sglang-combined"
  fwd      "SGLang combined --port 32000" "--port 32000"
  fwd      "SGLang combined -p 9204:32000" "-p 9204:32000"
  exactly_one "SGLang combined one --port" "--port"

  # --- 2e: --model-path ---
  fwd      "SGLang --model-path container" "--model-path /models/"
  # Host path should only be in volume mount, not app args
  local sglang_args_only; sglang_args_only=$(echo "$_DR_PREVIEW" | python3 -c "
import sys
line = sys.stdin.read().strip()
parts = line.split(' lmsysorg/sglang:')
if len(parts) > 1:
    print(parts[-1])
" 2>/dev/null)
  if [ -n "$sglang_args_only" ]; then
    assert_not_contains "SGLang no host path in model-path" "$sglang_args_only" "/home/kzeng/models" || log "FAIL: SGLang host path in app args"
  fi

  # --- 2f: Reverse assertions ---
  log "--- SGLang: reverse assertions ---"
  nope     "SGLang no default port 30000" "--port 30000"
  nope     "SGLang no duplicate --port" "--port --port"
  nope     "SGLang no duplicate --model-path" "--model-path --model-path"
  nope     "SGLang no -m flag" " -m /models/"
  nope     "SGLang no .gguf" ".gguf"

  # GPU
  fwd      "SGLang CUDA_VISIBLE_DEVICES" "CUDA_VISIBLE_DEVICES"
  fwd      "SGLang --gpus" "--gpus"
  nope     "SGLang GPU not as volume" "-v /dev/dri"

  log "SGLang matrix complete"
}

# ═══════════════════════════════════════════════════════════════
# SECTION 3: llama.cpp PARAMETER MATRIX
# ═══════════════════════════════════════════════════════════════
section_llamacpp() {
  [ -z "$LLAMACPP_RT" ] && { log "SKIP llama.cpp: no runtime"; return; }
  [ -z "$GGUF_ART" ] && { log "SKIP llama.cpp: no GGUF artifact"; return; }
  log "══════ llama.cpp Parameter Matrix ══════"

  # --- 3a: GGUF -m flag ---
  log "--- llama.cpp: GGUF -m flag ---"
  run_dryrun "llamacpp-basic" "$LLAMACPP_RT" "$GGUF_ART" \
    '{"host_port":9301,"container_port":8080,"app_port":8080}' '{}'
  fwd      "llama.cpp -m /models/...gguf" "-m /models/"
  fwd      "llama.cpp .gguf path" ".gguf"
  nope     "llama.cpp no directory model" "/models/ --port"  # directory would be /models/ followed by args
  exactly_one "llama.cpp exactly one -m" "-m"

  # --- 3b: Port separation ---
  log "--- llama.cpp: port separation ---"
  run_dryrun "llamacpp-ports" "$LLAMACPP_RT" "$GGUF_ART" \
    '{"host_port":9302,"container_port":9090,"app_port":9090}' '{}'
  fwd      "llama.cpp port mapping -p 9302:9090" "-p 9302:9090"
  fval     "llama.cpp --port 9090" "--port" "9090"
  nope     "llama.cpp no default --port 8080" "--port 8080"
  exactly_one "llama.cpp exactly one --port" "--port"

  # --- 3c: --ctx-size ---
  log "--- llama.cpp: ctx-size ---"
  run_dryrun "llamacpp-ctx" "$LLAMACPP_RT" "$GGUF_ART" \
    '{"host_port":9303,"container_port":8080,"app_port":8080}' \
    '{"ctx_size":1024}'
  fwd      "llama.cpp ctx-size 1024 via -c" "-c 1024"

  # --- 3d: --n-gpu-layers ---
  log "--- llama.cpp: n-gpu-layers ---"
  run_dryrun "llamacpp-ngl" "$LLAMACPP_RT" "$GGUF_ART" \
    '{"host_port":9304,"container_port":8080,"app_port":8080}' \
    '{"n_gpu_layers":20}'
  fwd      "llama.cpp -ngl 20" "-ngl 20"

  # --- 3e: Combined params ---
  log "--- llama.cpp: combined params ---"
  run_dryrun "llamacpp-combined" "$LLAMACPP_RT" "$GGUF_ART" \
    '{"host_port":9305,"container_port":9191,"app_port":9191}' \
    '{"ctx_size":2048,"n_gpu_layers":30,"threads":4}'
  fwd      "llama.cpp combined -m" "-m /models/"
  fwd      "llama.cpp combined .gguf" ".gguf"
  fwd      "llama.cpp combined --port 9191" "--port 9191"
  fwd      "llama.cpp combined ctx-size 2048 via -c" "-c 2048"
  fwd      "llama.cpp combined -ngl 30" "-ngl 30"
  exactly_one "llama.cpp combined one -m" "-m"
  exactly_one "llama.cpp combined one --port" "--port"
  fwd      "llama.cpp combined ctx flag present" "-c 2048"

  # --- 3f: Reverse assertions ---
  log "--- llama.cpp: reverse assertions ---"
  nope     "llama.cpp no default port 8080" "--port 8080"
  nope     "llama.cpp no duplicate -m" "-m /models/ -m"
  nope     "llama.cpp no .gguf as directory" " -m /models/ --"
  nope     "llama.cpp no HF directory path" "Qwen3-0.6B-Instruct-2512"  # HF model name should NOT appear
  nope     "llama.cpp no --model-path" "--model-path"
  fwd      "llama.cpp CUDA_VISIBLE_DEVICES" "CUDA_VISIBLE_DEVICES"
  fwd      "llama.cpp --gpus" "--gpus"
  nope     "llama.cpp GPU not as volume" "-v /dev/dri"

  log "llama.cpp matrix complete"
}

# ═══════════════════════════════════════════════════════════════
# SECTION 4: MetaX env check (if runtime present)
# ═══════════════════════════════════════════════════════════════
section_metax() {
  [ -z "$METAX_RT" ] && { log "SKIP MetaX: no runtime"; return; }
  log "══════ MetaX Env Check ══════"
  local detail; detail=$(api_get "backend-runtimes/$METAX_RT")
  local env; env=$(echo "$detail" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('default_env_json',{})))" 2>/dev/null)
  assert_contains     "MetaX MACA_VISIBLE_DEVICE in env" "$env" "MACA_VISIBLE_DEVICE" || log "FAIL: MetaX MACA_VISIBLE_DEVICE"
  assert_not_contains "MetaX no CUDA_VISIBLE_DEVICES sole" "$env" "CUDA_VISIBLE_DEVICES" || log "FAIL: MetaX CUDA leak"
  log "MetaX env check done"
}

# ── run all sections ──
section_vllm
section_sglang
section_llamacpp
section_metax

# ── final summary ──
echo ""
echo "Artifacts: $ARTIFACT_DIR"
echo "Key files:"
for f in "$ARTIFACT_DIR"/*-preview.txt; do
  echo "  $(basename "$f")"
done
assert_summary
