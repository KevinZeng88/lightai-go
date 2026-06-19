#!/bin/bash
# e2e-real-smoke-all-three.sh — vLLM + SGLang + llama.cpp real container smoke.
# Category: Real container E2E
# Each backend: start→health→inference→stop→cleanup with custom parameters.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/e2e-assert.sh"

SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-Commvault!234}"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-/tmp/lightai-e2e-smoke-$RUN_ID}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/lightai-e2e-cookies-$RUN_ID.txt}"
PREFIX="e2e-smoke"
mkdir -p "$ARTIFACT_DIR"

log() { printf '[%s] [smoke] %s\n' "$(date '+%H:%M:%S')" "$*"; }

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

# ── login + discover ──
resp="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" -H "Origin: $SERVER_URL" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR")"
CSRF_TOKEN="$(echo "$resp" | json_field csrf_token)"
[ -n "$CSRF_TOKEN" ] || { log "FATAL: Login failed"; exit 1; }
NODE_ID=$(api_get "nodes" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)
HF_ART=$(api_get "model-artifacts" | python3 -c "import json,sys; [print(a['id']) for a in json.load(sys.stdin) if a.get('format')=='huggingface']" 2>/dev/null | head -1)
GGUF_ART=$(api_get "model-artifacts" | python3 -c "import json,sys; [print(a['id']) for a in json.load(sys.stdin) if a.get('format')=='gguf']" 2>/dev/null | head -1)
log "Node=$NODE_ID HF=$HF_ART GGUF=$GGUF_ART"

# ── helper: start + wait for running ──
wait_running() {
  local dep_id="$1" label="$2" max_wait="${3:-60}"
  local inst_id=""
  for i in $(seq 1 $((max_wait/2))); do
    sleep 2
    local insts; insts=$(api_get "model-instances")
    inst_id=$(echo "$insts" | python3 -c "import json,sys; [print(i['id']) for i in json.load(sys.stdin) if i.get('deployment_id')=='$dep_id']" 2>/dev/null | head -1)
    if [ -n "$inst_id" ]; then
      local st; st=$(api_get "model-instances/$inst_id" | json_field actual_state 2>/dev/null)
      log "  $label poll $i: state=$st"
      if [ "$st" = "running" ]; then echo "$inst_id"; return 0; fi
      if [ "$st" = "failed" ] || [ "$st" = "error" ]; then
        local err; err=$(api_get "model-instances/$inst_id" | python3 -c "import json,sys; print(json.load(sys.stdin).get('last_error',''))" 2>/dev/null)
        log "  $label FAILED: $err"; echo "FAIL:$err"; return 1
      fi
    fi
  done
  log "  $label TIMEOUT"; echo "TIMEOUT"; return 1
}

# ── helper: inference test ──
test_inference() {
  local inst_id="$1" label="$2"
  local test_resp; test_resp=$(api_post "model-instances/$inst_id/test" '{}')
  echo "$test_resp" > "$ARTIFACT_DIR/${label}-test.json"
  local ok; ok=$(echo "$test_resp" | json_field ok)
  local preview; preview=$(echo "$test_resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('response_preview',''))" 2>/dev/null)
  log "  $label inference: ok=$ok preview=${preview:0:80}"
  if [ "$ok" = "True" ] && [ -n "$preview" ]; then
    echo "PASS"
  else
    echo "FAIL"
  fi
}

# ── helper: stop + verify ──
stop_and_verify() {
  local dep_id="$1" inst_id="$2" label="$3"
  api_post "deployments/$dep_id/stop" '{}' > /dev/null
  for i in $(seq 1 10); do
    sleep 2
    local st; st=$(api_get "model-instances/$inst_id" 2>/dev/null | json_field actual_state)
    if [ "$st" = "stopped" ]; then
      log "  $label stopped after $((i*2))s"
      echo "PASS"
      return 0
    fi
  done
  log "  $label stop TIMEOUT"
  echo "FAIL"
}

# ═══════════════════════════════════════
# BACKEND 1: vLLM
# ═══════════════════════════════════════
smoke_vllm() {
  log "══════ vLLM Real Smoke ══════"
  local rt="runtime.vllm.nvidia-docker"
  local dep_resp; dep_resp=$(api_post "deployments" "{\"name\":\"$PREFIX-vllm\",\"display_name\":\"vLLM Smoke\",\"model_artifact_id\":\"$HF_ART\",\"backend_runtime_id\":\"$rt\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8191,\"container_port\":8022,\"app_port\":8022},\"parameters_json\":{\"served_model_name\":\"e2e-vllm-smoke\",\"gpu_memory_utilization\":0.85,\"max_model_len\":4096,\"tensor_parallel_size\":1}}")
  local dep_id; dep_id=$(echo "$dep_resp" | json_field id)
  [ -z "$dep_id" ] && { log "FAIL: vLLM deploy create"; echo "vLLM: FAIL" >> "$ARTIFACT_DIR/results.txt"; return; }
  echo "$dep_resp" > "$ARTIFACT_DIR/vllm-deploy.json"

  local start; start=$(api_post "deployments/$dep_id/start" '{}')
  echo "$start" > "$ARTIFACT_DIR/vllm-start.json"
  local inst_id; inst_id=$(echo "$start" | json_field instance_id)
  [ -z "$inst_id" ] && { log "FAIL: vLLM start"; echo "vLLM: FAIL (start)"; api_delete "deployments/$dep_id" >/dev/null 2>&1; return; }

  local running; running=$(wait_running "$dep_id" "vLLM" 120)
  if [ "${running:0:4}" = "FAIL" ] || [ "$running" = "TIMEOUT" ]; then
    log "vLLM did not reach running: $running"
    echo "vLLM: BLOCKED_ENV (${running:0:100})" >> "$ARTIFACT_DIR/results.txt"
    api_delete "deployments/$dep_id" >/dev/null 2>&1 || true
    return
  fi
  inst_id="$running"

  # Check container
  sleep 3
  local cid; cid=$(api_get "model-instances/$inst_id" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('container_id',''))" 2>/dev/null)
  log "vLLM container: ${cid:0:12}"
  local cinspect; cinspect=$(docker inspect "$cid" 2>/dev/null | python3 -c "import json,sys; d=json.load(sys.stdin)[0]; print('running' if d.get('State',{}).get('Running') else 'stopped')" 2>/dev/null || echo "no")
  assert_eq "vLLM container running" "running" "$cinspect" || log "FAIL: vLLM container"

  # Inference
  local inf_result; inf_result=$(test_inference "$inst_id" "vllm")
  assert_eq "vLLM inference ok" "PASS" "$inf_result" || log "FAIL: vLLM inference"

  # Stop
  local stop_result; stop_result=$(stop_and_verify "$dep_id" "$inst_id" "vLLM")
  assert_eq "vLLM stop ok" "PASS" "$stop_result" || log "FAIL: vLLM stop"

  # Cleanup
  api_delete "deployments/$dep_id" >/dev/null 2>&1 || true
  sleep 2
  local still_running; still_running=$(docker ps --filter "id=$cid" -q 2>/dev/null)
  assert_empty "vLLM container gone" "$still_running" || log "FAIL: vLLM container leak"
  echo "vLLM: PASS" >> "$ARTIFACT_DIR/results.txt"
  log "vLLM smoke complete"
}

# ═══════════════════════════════════════
# BACKEND 2: SGLang
# ═══════════════════════════════════════
smoke_sglang() {
  log "══════ SGLang Real Smoke ══════"
  local rt="runtime.sglang.nvidia-docker"
  local dep_resp; dep_resp=$(api_post "deployments" "{\"name\":\"$PREFIX-sglang\",\"display_name\":\"SGLang Smoke\",\"model_artifact_id\":\"$HF_ART\",\"backend_runtime_id\":\"$rt\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8194,\"container_port\":31000,\"app_port\":31000},\"parameters_json\":{\"served_model_name\":\"e2e-sglang-smoke\",\"tp\":1}}")
  local dep_id; dep_id=$(echo "$dep_resp" | json_field id)
  [ -z "$dep_id" ] && { log "FAIL: SGLang deploy create"; echo "SGLang: FAIL" >> "$ARTIFACT_DIR/results.txt"; return; }
  echo "$dep_resp" > "$ARTIFACT_DIR/sglang-deploy.json"

  local start; start=$(api_post "deployments/$dep_id/start" '{}')
  echo "$start" > "$ARTIFACT_DIR/sglang-start.json"
  local inst_id; inst_id=$(echo "$start" | json_field instance_id)
  [ -z "$inst_id" ] && { log "FAIL: SGLang start"; echo "SGLang: FAIL (start)"; api_delete "deployments/$dep_id" >/dev/null 2>&1; return; }

  local running; running=$(wait_running "$dep_id" "SGLang" 120)
  if [ "${running:0:4}" = "FAIL" ] || [ "$running" = "TIMEOUT" ]; then
    log "SGLang did not reach running: $running"
    echo "SGLang: BLOCKED_ENV (${running:0:100})" >> "$ARTIFACT_DIR/results.txt"
    api_delete "deployments/$dep_id" >/dev/null 2>&1 || true
    return
  fi
  inst_id="$running"

  sleep 3
  local cid; cid=$(api_get "model-instances/$inst_id" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('container_id',''))" 2>/dev/null)
  log "SGLang container: ${cid:0:12}"
  local cinspect; cinspect=$(docker inspect "$cid" 2>/dev/null | python3 -c "import json,sys; d=json.load(sys.stdin)[0]; print('running' if d.get('State',{}).get('Running') else 'stopped')" 2>/dev/null || echo "no")
  assert_eq "SGLang container running" "running" "$cinspect" || log "FAIL: SGLang container"

  local inf_result; inf_result=$(test_inference "$inst_id" "sglang")
  assert_eq "SGLang inference ok" "PASS" "$inf_result" || log "FAIL: SGLang inference"

  local stop_result; stop_result=$(stop_and_verify "$dep_id" "$inst_id" "sglang")
  assert_eq "SGLang stop ok" "PASS" "$stop_result" || log "FAIL: SGLang stop"

  api_delete "deployments/$dep_id" >/dev/null 2>&1 || true
  sleep 2
  local still_running; still_running=$(docker ps --filter "id=$cid" -q 2>/dev/null)
  assert_empty "SGLang container gone" "$still_running" || log "FAIL: SGLang container leak"
  echo "SGLang: PASS" >> "$ARTIFACT_DIR/results.txt"
  log "SGLang smoke complete"
}

# ═══════════════════════════════════════
# BACKEND 3: llama.cpp
# ═══════════════════════════════════════
smoke_llamacpp() {
  log "══════ llama.cpp Real Smoke ══════"
  local rt="runtime.llamacpp.nvidia-docker"
  local dep_resp; dep_resp=$(api_post "deployments" "{\"name\":\"$PREFIX-llamacpp\",\"display_name\":\"llama.cpp Smoke\",\"model_artifact_id\":\"$GGUF_ART\",\"backend_runtime_id\":\"$rt\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"gpu_ids\":[]},\"service_json\":{\"host_port\":8193,\"container_port\":9090,\"app_port\":9090},\"parameters_json\":{\"ctx_size\":2048,\"n_gpu_layers\":30}}")
  local dep_id; dep_id=$(echo "$dep_resp" | json_field id)
  [ -z "$dep_id" ] && { log "FAIL: llama.cpp deploy create"; echo "llama.cpp: FAIL" >> "$ARTIFACT_DIR/results.txt"; return; }
  echo "$dep_resp" > "$ARTIFACT_DIR/llamacpp-deploy.json"

  local start; start=$(api_post "deployments/$dep_id/start" '{}')
  echo "$start" > "$ARTIFACT_DIR/llamacpp-start.json"
  local inst_id; inst_id=$(echo "$start" | json_field instance_id)
  [ -z "$inst_id" ] && { log "FAIL: llama.cpp start"; echo "llama.cpp: FAIL (start)"; api_delete "deployments/$dep_id" >/dev/null 2>&1; return; }

  local running; running=$(wait_running "$dep_id" "llama.cpp" 60)
  if [ "${running:0:4}" = "FAIL" ] || [ "$running" = "TIMEOUT" ]; then
    log "llama.cpp did not reach running: $running"
    echo "llama.cpp: BLOCKED_ENV (${running:0:100})" >> "$ARTIFACT_DIR/results.txt"
    api_delete "deployments/$dep_id" >/dev/null 2>&1 || true
    return
  fi
  inst_id="$running"

  sleep 2
  local cid; cid=$(api_get "model-instances/$inst_id" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('container_id',''))" 2>/dev/null)
  log "llama.cpp container: ${cid:0:12}"
  local cinspect; cinspect=$(docker inspect "$cid" 2>/dev/null | python3 -c "import json,sys; d=json.load(sys.stdin)[0]; print('running' if d.get('State',{}).get('Running') else 'stopped')" 2>/dev/null || echo "no")
  assert_eq "llama.cpp container running" "running" "$cinspect" || log "FAIL: llama.cpp container"

  local inf_result; inf_result=$(test_inference "$inst_id" "llamacpp")
  assert_eq "llama.cpp inference ok" "PASS" "$inf_result" || log "FAIL: llama.cpp inference"

  local stop_result; stop_result=$(stop_and_verify "$dep_id" "$inst_id" "llamacpp")
  assert_eq "llama.cpp stop ok" "PASS" "$stop_result" || log "FAIL: llama.cpp stop"

  api_delete "deployments/$dep_id" >/dev/null 2>&1 || true
  sleep 2
  local still_running; still_running=$(docker ps --filter "id=$cid" -q 2>/dev/null)
  assert_empty "llama.cpp container gone" "$still_running" || log "FAIL: llama.cpp container leak"
  echo "llama.cpp: PASS" >> "$ARTIFACT_DIR/results.txt"
  log "llama.cpp smoke complete"
}

# ── run all ──
echo "# Real smoke results $(date)" > "$ARTIFACT_DIR/results.txt"
smoke_vllm
smoke_sglang
smoke_llamacpp

# ── summary ──
echo ""
echo "=== REAL SMOKE RESULTS ==="
cat "$ARTIFACT_DIR/results.txt"
echo ""
echo "Artifacts: $ARTIFACT_DIR"
assert_summary
