#!/usr/bin/env bash
# LightAI E2E API test — safe PID mgmt, E2E-only runtime, response capture.
set -euo pipefail
SELF_PID="$$"; PARENT_PID="${PPID:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTDIR="$PROJECT_DIR/docs/reports/phase-3/verification"
mkdir -p "$OUTDIR"

SERVER_PORT="${E2E_SERVER_PORT:-19990}"
AGENT_TOKEN="e2e-test-token"
SERVER_URL="http://127.0.0.1:${SERVER_PORT}"
# E2E uses its own runtime directory — never touches user's runtime/
E2E_DIR="$PROJECT_DIR/runtime/e2e"
E2E_RUNTIME="$E2E_DIR/runtime"
E2E_DB="$E2E_RUNTIME/lightai.db"
E2E_CREDS="$E2E_RUNTIME/initial-credentials.txt"
COOKIE_JAR="$E2E_DIR/cookies.txt"
CSRF_FILE="$E2E_DIR/csrf_token.txt"
SERVER_PID_FILE="$E2E_DIR/server.pid"
AGENT_PID_FILE="$E2E_DIR/agent.pid"
SERVER_OWNED="$E2E_DIR/server.owned"
AGENT_OWNED="$E2E_DIR/agent.owned"
TIMEOUT="${E2E_TIMEOUT:-600}"

log()  { echo "[$(date '+%H:%M:%S')] $*"; }
pass() { echo "[$(date '+%H:%M:%S')] PASS: $*"; }
fail() { echo "[$(date '+%H:%M:%S')] FAIL: $*"; return 1; }

get_json() {
  python3 -c "
import json, sys
d = json.load(sys.stdin)
if isinstance(d, list) and len(d) > 0:
    d = d[0]
if isinstance(d, dict):
    v = d.get('$1', '')
    if v is not None:
        print(v)
" 2>/dev/null
}

safe_kill() {
  local pid="$1" label="$2"; [ -z "$pid" ] && return 0
  [ "$pid" = "$SELF_PID" ] && { log "WARN: refuse self pid=$pid"; return 0; }
  [ "$pid" = "$PARENT_PID" ] && { log "WARN: refuse parent pid=$pid"; return 0; }
  if kill -0 "$pid" 2>/dev/null; then kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null; log "Stopped $label (pid=$pid)"; fi
}
cleanup_e2e_procs() {
  if [ -f "$SERVER_OWNED" ] && [ -f "$SERVER_PID_FILE" ]; then
    safe_kill "$(cat "$SERVER_PID_FILE" 2>/dev/null)" "server"; rm -f "$SERVER_PID_FILE" "$SERVER_OWNED"
  fi
  if [ -f "$AGENT_OWNED" ] && [ -f "$AGENT_PID_FILE" ]; then
    safe_kill "$(cat "$AGENT_PID_FILE" 2>/dev/null)" "agent"; rm -f "$AGENT_PID_FILE" "$AGENT_OWNED"
  fi
}
trap cleanup_e2e_procs EXIT

ensure_login() {
  mkdir -p "$E2E_DIR" "$E2E_RUNTIME"
  local pass=""
  # Only read E2E credentials, never touch user's runtime/
  if [ -f "$E2E_CREDS" ]; then
    pass=$(grep "Password:" "$E2E_CREDS" 2>/dev/null | tail -1 | awk '{print $NF}')
  elif [ -f "$PROJECT_DIR/runtime/initial-credentials.txt" ]; then
    pass=$(grep "Password:" "$PROJECT_DIR/runtime/initial-credentials.txt" 2>/dev/null | tail -1 | awk '{print $NF}')
  fi
  [ -z "$pass" ] && { fail "Cannot read admin password"; return 1; }
  local resp
  resp=$(curl -s -X POST "$SERVER_URL/api/v1/auth/login" -H "Content-Type: application/json" -H "Origin: $SERVER_URL" -d "{\"username\":\"admin\",\"password\":\"$pass\"}" -c "$COOKIE_JAR" 2>/dev/null)
  CSRF_TOKEN=$(echo "$resp" | get_json csrf_token)
  [ -z "$CSRF_TOKEN" ] && { fail "Login failed: $resp"; return 1; }
  echo "$CSRF_TOKEN" > "$CSRF_FILE"
}

api() {
  local method="$1" path="$2" data="${3:-}" step="${4:-api}"
  local csrf=""; [ -f "$CSRF_FILE" ] && csrf="-H X-CSRF-Token:$(cat "$CSRF_FILE")"
  local origin="-H Origin:$SERVER_URL"
  local resp code body
  if [ -n "$data" ]; then
    resp=$(curl -s -X "$method" "$SERVER_URL$path" -b "$COOKIE_JAR" -c "$COOKIE_JAR" $csrf $origin -H "Content-Type: application/json" -d "$data" -w "\nHTTP:%{http_code}" 2>/dev/null)
  else
    resp=$(curl -s -X "$method" "$SERVER_URL$path" -b "$COOKIE_JAR" -c "$COOKIE_JAR" $origin -w "\nHTTP:%{http_code}" 2>/dev/null)
  fi
  code=$(echo "$resp" | grep -o 'HTTP:[0-9]*' | tail -1 | grep -o '[0-9]*')
  body=$(echo "$resp" | sed '/^HTTP:[0-9]*$/d')
  if [ "$code" = "401" ] || [ "$code" = "403" ]; then
    ensure_login || return 1; [ -f "$CSRF_FILE" ] && csrf="-H X-CSRF-Token:$(cat "$CSRF_FILE")"
    if [ -n "$data" ]; then
      resp=$(curl -s -X "$method" "$SERVER_URL$path" -b "$COOKIE_JAR" -c "$COOKIE_JAR" $csrf $origin -H "Content-Type: application/json" -d "$data" -w "\nHTTP:%{http_code}" 2>/dev/null)
    else
      resp=$(curl -s -X "$method" "$SERVER_URL$path" -b "$COOKIE_JAR" -c "$COOKIE_JAR" $origin -w "\nHTTP:%{http_code}" 2>/dev/null)
    fi
    code=$(echo "$resp" | grep -o 'HTTP:[0-9]*' | tail -1 | grep -o '[0-9]*'); body=$(echo "$resp" | sed '/^HTTP:[0-9]*$/d')
  fi
  if [ "$code" != "200" ] && [ "$code" != "201" ]; then
    echo "[$(date '+%H:%M:%S')] [FAIL] $step $method $path http=$code body=${body:0:200}" >&2
    echo "$body"; return 1
  fi
  echo "$body"; return 0
}

# ---- Commands ----
cmd_env() {
  docker version >/dev/null 2>&1 || { fail "no docker"; return 1; }
  nvidia-smi >/dev/null 2>&1 || { fail "no nvidia-smi"; return 1; }
  pass "env OK"
}

cmd_start_server() {
  if curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/api/v1/inference-backends" -H "Origin: $SERVER_URL" 2>/dev/null | grep -qE "200|401"; then
    log "Server reuse"; return 0
  fi
  log "Starting server..."
  local td=$(mktemp -d); mkdir -p "$td/data" "$td/logs" "$E2E_RUNTIME"
  cat > "$td/config.yaml" << YAML
host: "127.0.0.1"
port: $SERVER_PORT
db_path: "$td/data/lightai.db"
log_level: "error"
log_dir: "$td/logs"
agent_token: "$AGENT_TOKEN"
YAML
  go build -o "$td/srv" "$PROJECT_DIR/cmd/server" 2>&1
  # Save existing creds, start server from project root (needed for config file access),
  # then copy fresh creds to E2E runtime
  local saved_creds=""
  [ -f "$PROJECT_DIR/runtime/initial-credentials.txt" ] && saved_creds=$(cat "$PROJECT_DIR/runtime/initial-credentials.txt")
  rm -f "$PROJECT_DIR/runtime/initial-credentials.txt"
  cd "$PROJECT_DIR"
  "$td/srv" --config "$td/config.yaml" > "$OUTDIR/e2e-server.log" 2>&1 &
  local pid=$!; sleep 3
  if ! kill -0 "$pid" 2>/dev/null; then fail "server start failed"; cat "$OUTDIR/e2e-server.log"|tail -10; return 1; fi
  # Copy fresh credentials to E2E runtime, restore original if needed
  [ -f "$PROJECT_DIR/runtime/initial-credentials.txt" ] && cp "$PROJECT_DIR/runtime/initial-credentials.txt" "$E2E_CREDS" 2>/dev/null
  if [ -n "$saved_creds" ]; then
    echo "$saved_creds" > "$PROJECT_DIR/runtime/initial-credentials.txt"
  fi
  mkdir -p "$E2E_DIR"; echo "$pid" > "$SERVER_PID_FILE"; touch "$SERVER_OWNED"
  pass "server PID=$pid"
}

cmd_start_agent() {
  log "Starting agent..."
  local td=$(mktemp -d); mkdir -p "$td/logs"
  cat > "$td/agent.yaml" << YAML
server_url: "$SERVER_URL"
agent_token: "$AGENT_TOKEN"
log_level: "debug"
log_dir: "$td/logs"
heartbeat: {interval: 2s}
metrics: {enabled: false}
YAML
  go build -o "$td/agent" "$PROJECT_DIR/cmd/agent" 2>&1
  "$td/agent" --config "$td/agent.yaml" > "$OUTDIR/e2e-agent.log" 2>&1 &
  local pid=$!; sleep 5
  if ! kill -0 "$pid" 2>/dev/null; then fail "agent start failed"; cat "$OUTDIR/e2e-agent.log"|tail -10; return 1; fi
  mkdir -p "$E2E_DIR"; echo "$pid" > "$AGENT_PID_FILE"; touch "$AGENT_OWNED"
  pass "agent PID=$pid"
}

cmd_login() { log "Login..."; ensure_login && pass "login OK" || fail "login failed"; }
get_ver() { case "$1" in vllm) echo "0.8.5";; sglang) echo "0.4.6";; llamacpp) echo "b4817";; esac; }

seed_backend() {
  local b="$1" img="$2" mp="$3" mn="$4" hp="$5" cp="$6" t="$7"
  log "Seed $b..."
  local rd="$E2E_DIR/responses"; mkdir -p "$rd"

  local rj=$(api POST /api/v1/backend-runtimes/from-template "{\"template_name\":\"$t\",\"name\":\"${b}-e2e\",\"vendor\":\"nvidia\",\"backend_name\":\"$b\",\"backend_version\":\"$(get_ver $b)\",\"image_name\":\"$img\"}" "seed-$b-rt") || { fail "seed-$b: rt failed"; return 1; }
  echo "$rj" > "$rd/seed-${b}-rt.json"
  local ri=$(echo "$rj" | get_json id); [ -z "$ri" ] && { fail "seed-$b: no rt id"; return 1; }
  mkdir -p "$E2E_DIR"; echo "$ri" > "$E2E_DIR/${b}_rt_id"

  local aj=$(api POST /api/v1/model-artifacts "{\"name\":\"${b}-e2e-model\",\"path\":\"$mp\",\"format\":\"custom\",\"task_type\":\"chat\"}" "seed-$b-artifact") || { fail "seed-$b: artifact failed"; return 1; }
  echo "$aj" > "$rd/seed-${b}-artifact.json"
  local ai=$(echo "$aj" | get_json id); [ -z "$ai" ] && { fail "seed-$b: no artifact id"; return 1; }
  echo "$ai" > "$E2E_DIR/${b}_art_id"

  local dj=$(api POST /api/v1/model-deployments "{\"name\":\"${b}-e2e-deploy\",\"model_artifact_id\":\"$ai\",\"backend_runtime_id\":\"$ri\",\"placement_json\":\"{}\",\"service_json\":\"{\\\"host_port\\\":$hp}\",\"parameters_json\":\"{\\\"served_model_name\\\":\\\"$mn\\\"}\",\"env_overrides_json\":\"{}\"}" "seed-$b-deploy") || { fail "seed-$b: deploy failed"; return 1; }
  echo "$dj" > "$rd/seed-${b}-deploy.json"
  local di=$(echo "$dj" | get_json id); [ -z "$di" ] && { fail "seed-$b: no deploy id"; return 1; }
  echo "$di" > "$E2E_DIR/${b}_dep_id"; echo "$hp" > "$E2E_DIR/${b}_port"; echo "$mn" > "$E2E_DIR/${b}_model"
  pass "seed-$b"
}

cmd_seed_vllm()   { seed_backend vllm "vllm/vllm-openai:latest" "/home/kzeng/models/Qwen3-0.6B-Instruct-2512" "Qwen3-0.6B-Instruct-2512" 8004 8000 vllm-nvidia-docker; }
cmd_seed_sglang() { seed_backend sglang "lmsysorg/sglang:latest" "/home/kzeng/models/Qwen3-0.6B-Instruct-2512" "Qwen3-0.6B-Instruct-2512" 30000 30000 sglang-nvidia-docker; }
cmd_seed_llamacpp() { seed_backend llamacpp "ghcr.io/ggml-org/llama.cpp:server-cuda13" "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf" "Qwen3.5-9B-Q4_K_M.gguf" 8002 8080 llamacpp-nvidia-docker; }

# start_api_only: start deployment, verify DB records, do NOT wait for model loading
cmd_start_api_only() {
  local b="$1" dep_id=$(cat "$E2E_DIR/${b}_dep_id" 2>/dev/null)
  [ -z "$dep_id" ] && { fail "start-$b: not seeded"; return 1; }
  log "Start $b (api-only)..."
  local rd="$E2E_DIR/responses"; mkdir -p "$rd"

  local sr=$(api POST "/api/v1/model-deployments/$dep_id/start" "{}" "start-$b") || { fail "start-$b: API failed"; return 1; }
  echo "$sr" > "$rd/start-${b}.json"
  local si=$(echo "$sr" | get_json instance_id)
  [ -z "$si" ] && { fail "start-$b: no instance_id in response"; return 1; }
  echo "$si" > "$E2E_DIR/${b}_instance_id"
  log "  Instance: $si"
  echo "[$(date '+%H:%M:%S')] [TIMING] start-$b.instance_created ts=$(date +%s)"

  # Verify instance exists in API (not waiting for running — just check creation)
  sleep 2
  local ir=$(api GET "/api/v1/model-instances?deployment_id=$dep_id" "" "check-$b" 2>/dev/null) || { fail "start-$b: cannot list instances"; return 1; }
  echo "$ir" > "$rd/instances-${b}.json"
  local st=$(echo "$ir" | get_json actual_state)
  log "  State: $st"
  pass "start-$b: instance=$si state=$st"
}

# start_full: full lifecycle with model loading wait (for single-backend/full mode)
cmd_start_full() {
  local b="$1" dep_id=$(cat "$E2E_DIR/${b}_dep_id" 2>/dev/null)
  [ -z "$dep_id" ] && { fail "start-$b: not seeded"; return 1; }
  log "Start $b (full)..."
  local rd="$E2E_DIR/responses"; mkdir -p "$rd"

  local sr=$(api POST "/api/v1/model-deployments/$dep_id/start" "{}" "start-$b") || { fail "start-$b: API failed"; return 1; }
  echo "$sr" > "$rd/start-${b}.json"
  local si=$(echo "$sr" | get_json instance_id)
  [ -z "$si" ] && { fail "start-$b: no instance_id"; return 1; }
  echo "$si" > "$E2E_DIR/${b}_instance_id"
  log "  Instance: $si"
  echo "[$(date '+%H:%M:%S')] [TIMING] start-$b.instance_created ts=$(date +%s)"

  log "Waiting for instance running..."
  local to=$TIMEOUT w=0
  while [ $w -lt $to ]; do
    sleep 2; w=$((w+2))
    local ir=$(api GET "/api/v1/model-instances?deployment_id=$dep_id" "" "wait-$b" 2>/dev/null) || continue
    echo "$ir" > "$rd/instances-${b}.json"
    local st=$(echo "$ir" | get_json actual_state)
    local em=$(echo "$ir" | get_json last_error)
    case "$st" in
      running) echo "[$(date '+%H:%M:%S')] [TIMING] start-$b.running ts=$(date +%s) wait=${w}s"; echo "[$(date '+%H:%M:%S')] [WAIT] wait_completed elapsed_ms=$((w*1000)) state=running"; pass "start-$b: running after ${w}s"; return 0;;
      failed|error) fail "start-$b: $st error=$em"; return 1;;
    esac
    [ $((w%30)) -eq 0 ] && log "  $b: $st (${w}s)..."
    echo "[$(date '+%H:%M:%S')] [WAIT] wait_progress elapsed_ms=$((w*1000)) state=$st"
  done
  echo "[$(date '+%H:%M:%S')] [WAIT] wait_timeout elapsed_ms=$((to*1000)) last_state=$st"; fail "start-$b: timeout ${to}s"; return 1
}

cmd_test_backend() {
  local b="$1" port=$(cat "$E2E_DIR/${b}_port" 2>/dev/null) model=$(cat "$E2E_DIR/${b}_model" 2>/dev/null)
  [ -z "$port" ] && { fail "test-$b: no port"; return 1; }
  local mc=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:$port/v1/models" 2>/dev/null)
  [ "$mc" != "200" ] && { fail "test-$b: /v1/models=$mc"; return 1; }
  local cc=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:$port/v1/chat/completions" -H "Content-Type: application/json" -d "{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"Hi\"}],\"max_tokens\":5}" 2>/dev/null)
  [ "$cc" != "200" ] && { fail "test-$b: /v1/chat=$cc"; return 1; }
  pass "test-$b: /v1/models=200 /v1/chat=200"
}

# stop_backend: saves response, verifies cleanup
cmd_stop_backend() {
  local b="$1" dep_id=$(cat "$E2E_DIR/${b}_dep_id" 2>/dev/null)
  [ -z "$dep_id" ] && return 0
  log "Stop $b..."
  local rd="$E2E_DIR/responses"; mkdir -p "$rd"
  local sr
  sr=$(api POST "/api/v1/model-deployments/$dep_id/stop" "{}" "stop-$b") || { log "stop-$b returned error: $sr"; }
  echo "$sr" > "$rd/stop-${b}.json"
  # Verify stop response
  local status=$(echo "$sr" | get_json status)
  log "  Stop response: status=$status"
  sleep 2
  # Verify instance is stopped via API
  local ir=$(api GET "/api/v1/model-instances?deployment_id=$dep_id" "" "check-stop-$b" 2>/dev/null) || true
  local st=$(echo "$ir" | get_json actual_state)
  log "  Instance state after stop: $st"
  pass "stop-$b"
}

cmd_stop_all() { for b in vllm sglang llamacpp; do cmd_stop_backend "$b"; done; }

cmd_cleanup() {
  log "Cleanup..."
  cmd_stop_all 2>/dev/null || true
  docker rm -f lightai-inst- 2>/dev/null || true
  cleanup_e2e_procs
  pass "cleanup done"
}

# ---- Layered commands ----
cmd_quick() {
  log "=== QUICK ==="
  cmd_env; cmd_start_server; cmd_login
  curl -s "$SERVER_URL/api/v1/inference-backends" -b "$COOKIE_JAR" -H "Origin: $SERVER_URL" \
    | python3 -c "import sys,json;d=json.load(sys.stdin);print(f'{len(d)} backends')" 2>/dev/null
  pass "quick OK"
}

# api-only: validate API/DB records WITHOUT waiting for model loading
cmd_api_only() {
  log "=== API-ONLY (no model load) ==="
  local count_inst=0 count_tasks=0
  cmd_env; cmd_start_server; cmd_start_agent; cmd_login
  for b in vllm sglang llamacpp; do
    "cmd_seed_$b" || continue
    cmd_start_api_only "$b" || { cmd_stop_backend "$b"; continue; }
    count_inst=$((count_inst+1))
    cmd_stop_backend "$b"
  done
  cmd_cleanup
  log "api-only summary: instances=$count_inst"
  [ $count_inst -eq 3 ] && pass "api-only: all 3 backends verified" || fail "api-only: only $count_inst/3"
}

# single_backend: full lifecycle with model loading
cmd_single_backend() {
  local b="$1"; TIMEOUT=600; log "=== SINGLE: $b ==="
  cmd_env; cmd_start_server; cmd_start_agent; cmd_login
  "cmd_seed_$b" || { cmd_cleanup; return 1; }
  cmd_start_full "$b" || { cmd_stop_backend "$b"; cmd_cleanup; return 1; }
  sleep 5; cmd_test_backend "$b" || true
  cmd_stop_backend "$b"; sleep 3; cmd_cleanup
  pass "$b-only: complete"
}

cmd_full() { TIMEOUT=400; log "=== FULL (3 backends) ==="; cmd_all; }

cmd_all() {
  local failed=0
  cmd_env || failed=1; cmd_start_server || failed=1; cmd_start_agent || failed=1; cmd_login || failed=1
  for b in vllm sglang llamacpp; do
    "cmd_seed_$b" || { failed=1; continue; }
    cmd_start_full "$b" || { failed=1; cmd_stop_backend "$b"; continue; }
    sleep 5; cmd_test_backend "$b" || failed=1
    cmd_stop_backend "$b"; sleep 3
  done
  cmd_cleanup
  [ $failed -eq 0 ] && log "ALL E2E PASSED" || log "SOME E2E FAILED"
  return $failed
}

case "${1:-help}" in
  env) cmd_env ;; start-server) cmd_start_server ;; start-agent) cmd_start_agent ;; login) cmd_login ;;
  seed-vllm) cmd_seed_vllm ;; seed-sglang) cmd_seed_sglang ;; seed-llamacpp) cmd_seed_llamacpp ;;
  start-vllm) cmd_start_full vllm ;; start-sglang) cmd_start_full sglang ;; start-llamacpp) cmd_start_full llamacpp ;;
  test-vllm) cmd_test_backend vllm ;; test-sglang) cmd_test_backend sglang ;; test-llamacpp) cmd_test_backend llamacpp ;;
  stop-all) cmd_stop_all ;; cleanup) cmd_cleanup ;;
  quick) cmd_quick 2>&1 | tee "$OUTDIR/17-api-e2e-quick.txt" ;;
  api-only) cmd_api_only 2>&1 | tee "$OUTDIR/17-api-e2e-api-only.txt" ;;
  vllm-only) cmd_single_backend vllm 2>&1 | tee "$OUTDIR/14-api-e2e-vllm.txt" ;;
  sglang-only) cmd_single_backend sglang 2>&1 | tee "$OUTDIR/15-api-e2e-sglang.txt" ;;
  llamacpp-only) cmd_single_backend llamacpp 2>&1 | tee "$OUTDIR/16-api-e2e-llamacpp.txt" ;;
  full) cmd_full 2>&1 | tee "$OUTDIR/17-api-e2e-all.txt" ;;
  all) cmd_all 2>&1 | tee "$OUTDIR/17-api-e2e-all.txt" ;;
  *) echo "Usage: $0 {quick|api-only|llamacpp-only|vllm-only|sglang-only|full|all}" ;;
esac
