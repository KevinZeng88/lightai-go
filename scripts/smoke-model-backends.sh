#!/usr/bin/env bash
# LightAI Go Model Backend Smoke Tests
# Usage: bash scripts/smoke-model-backends.sh <command>
# Commands: env, vllm, sglang, llamacpp, runplan, all, cleanup
set -euo pipefail

OUTDIR="docs/reports/phase-3/verification"
mkdir -p "$OUTDIR"

# ---- Config ----
VLLM_IMAGE="vllm/vllm-openai:latest"
VLLM_MODEL="/home/kzeng/models/Qwen3-0.6B-Instruct-2512"
VLLM_HOST_PORT=8004
VLLM_CONTAINER_PORT=8000
VLLM_CONTAINER="qwen3-06b-vllm"

SGLANG_IMAGE="lmsysorg/sglang:latest"
SGLANG_MODEL="/home/kzeng/models/Qwen3-0.6B-Instruct-2512"
SGLANG_HOST_PORT=30000
SGLANG_CONTAINER_PORT=30000
SGLANG_CONTAINER="qwen3-06b-sglang"

LLAMA_IMAGE="ghcr.io/ggml-org/llama.cpp:server-cuda13"
LLAMA_MODEL_DIR="/home/kzeng/models/Qwen3.5-9B-Q4"
LLAMA_MODEL_FILE="/models/Qwen3.5-9B-Q4_K_M.gguf"
LLAMA_HOST_PORT=8002
LLAMA_CONTAINER_PORT=8080
LLAMA_CONTAINER="qwen35-9b-q4-llama"

# ---- Helpers ----
log()  { echo "[$(date '+%H:%M:%S')] $*"; }
pass() { log "PASS: $*"; }
fail() { log "FAIL: $*"; return 1; }

wait_http() {
  local url="$1" timeout="${2:-120}" name="${3:-service}"
  log "Waiting for $name at $url (timeout=${timeout}s)..."
  local start=$(date +%s)
  while true; do
    if curl -sf -o /dev/null "$url" 2>/dev/null; then
      local elapsed=$(($(date +%s) - start))
      log "$name ready after ${elapsed}s"
      return 0
    fi
    if [ $(($(date +%s) - start)) -ge "$timeout" ]; then
      fail "$name did not become ready within ${timeout}s"
      return 1
    fi
    sleep 2
  done
}

check_port_free() {
  local port="$1" name="$2"
  if ss -tlnp 2>/dev/null | grep -q ":${port} "; then
    fail "$name port $port is already in use"
    return 1
  fi
}

cleanup_container() {
  local name="$1"
  docker rm -f "$name" 2>/dev/null || true
}

# ---- Commands ----

cmd_env() {
  log "=== Environment Check ==="
  {
    echo "=== docker version ==="
    docker version 2>&1
    echo ""
    echo "=== nvidia-smi ==="
    nvidia-smi 2>&1
    echo ""
    echo "=== Docker GPU smoke test ==="
    docker run --rm --gpus all --entrypoint nvidia-smi "$VLLM_IMAGE" 2>&1
  } | tee "$OUTDIR/06-nvidia-docker-env.txt"
  pass "Environment check"
}

cmd_vllm() {
  log "=== vLLM Smoke Test ==="
  cleanup_container "$VLLM_CONTAINER"
  check_port_free "$VLLM_HOST_PORT" "vLLM" || return 1

  # Verify model exists
  if [ ! -f "$VLLM_MODEL/config.json" ] || [ ! -f "$VLLM_MODEL/model.safetensors" ]; then
    fail "vLLM model files missing at $VLLM_MODEL"
    return 1
  fi

  docker run -d \
    --name "$VLLM_CONTAINER" \
    --gpus all \
    -p "${VLLM_HOST_PORT}:${VLLM_CONTAINER_PORT}" \
    -v "${VLLM_MODEL}:/models/$(basename $VLLM_MODEL):ro" \
    "$VLLM_IMAGE" \
    --model "/models/$(basename $VLLM_MODEL)" \
    --served-model-name "$(basename $VLLM_MODEL)" \
    --host 0.0.0.0 \
    --port "$VLLM_CONTAINER_PORT" \
    --max-model-len 4096 \
    --gpu-memory-utilization 0.6 2>&1

  # Wait for model to load (vLLM: 60-120s)
  if ! wait_http "http://127.0.0.1:${VLLM_HOST_PORT}/v1/models" 180 "vLLM"; then
    docker logs --tail=50 "$VLLM_CONTAINER" 2>&1
    cleanup_container "$VLLM_CONTAINER"
    return 1
  fi

  # API tests
  log "GET /v1/models"
  curl -sf "http://127.0.0.1:${VLLM_HOST_PORT}/v1/models" | python3 -m json.tool > /dev/null || fail "vLLM /v1/models"

  log "POST /v1/chat/completions"
  RESP=$(curl -sf "http://127.0.0.1:${VLLM_HOST_PORT}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"$(basename $VLLM_MODEL)\",\"messages\":[{\"role\":\"user\",\"content\":\"Say hello in one word\"}],\"max_tokens\":20}")
  echo "$RESP" | python3 -m json.tool > /dev/null || fail "vLLM /v1/chat/completions"

  pass "vLLM smoke test"
  cleanup_container "$VLLM_CONTAINER"
}

cmd_sglang() {
  log "=== SGLang Smoke Test ==="
  cleanup_container "$SGLANG_CONTAINER"
  check_port_free "$SGLANG_HOST_PORT" "SGLang" || return 1

  if [ ! -f "$SGLANG_MODEL/config.json" ] || [ ! -f "$SGLANG_MODEL/model.safetensors" ]; then
    fail "SGLang model files missing at $SGLANG_MODEL"
    return 1
  fi

  docker run -d \
    --name "$SGLANG_CONTAINER" \
    --gpus all \
    --shm-size 32g \
    --ipc=host \
    -p "${SGLANG_HOST_PORT}:${SGLANG_CONTAINER_PORT}" \
    -v "${SGLANG_MODEL}:/models/$(basename $SGLANG_MODEL):ro" \
    "$SGLANG_IMAGE" \
    python3 -m sglang.launch_server \
      --model-path "/models/$(basename $SGLANG_MODEL)" \
      --host 0.0.0.0 \
      --port "$SGLANG_CONTAINER_PORT" 2>&1

  # Wait (SGLang: 30-90s, /health for newer versions)
  if ! wait_http "http://127.0.0.1:${SGLANG_HOST_PORT}/health" 180 "SGLang"; then
    # Fallback: try /v1/models
    log "SGLang /health not responding, trying /v1/models..."
    if ! wait_http "http://127.0.0.1:${SGLANG_HOST_PORT}/v1/models" 60 "SGLang-fallback"; then
      docker logs --tail=50 "$SGLANG_CONTAINER" 2>&1
      cleanup_container "$SGLANG_CONTAINER"
      return 1
    fi
  fi

  # Get actual model ID from SGLang (returns path, not served name)
  MODEL_ID=$(curl -sf "http://127.0.0.1:${SGLANG_HOST_PORT}/v1/models" | python3 -c "import sys,json; print(json.load(sys.stdin)['data'][0]['id'])" 2>/dev/null)
  log "SGLang model ID: $MODEL_ID"

  RESP=$(curl -sf "http://127.0.0.1:${SGLANG_HOST_PORT}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"$MODEL_ID\",\"messages\":[{\"role\":\"user\",\"content\":\"Say hello in one word\"}],\"max_tokens\":20}")
  echo "$RESP" | python3 -m json.tool > /dev/null || fail "SGLang /v1/chat/completions"

  pass "SGLang smoke test"
  cleanup_container "$SGLANG_CONTAINER"
}

cmd_llamacpp() {
  log "=== llama.cpp Smoke Test ==="
  cleanup_container "$LLAMA_CONTAINER"
  check_port_free "$LLAMA_HOST_PORT" "llama.cpp" || return 1

  if [ ! -f "${LLAMA_MODEL_DIR}/Qwen3.5-9B-Q4_K_M.gguf" ]; then
    fail "llama.cpp model file missing at ${LLAMA_MODEL_DIR}/Qwen3.5-9B-Q4_K_M.gguf"
    return 1
  fi

  docker run -d \
    --name "$LLAMA_CONTAINER" \
    --gpus all \
    -p "${LLAMA_HOST_PORT}:${LLAMA_CONTAINER_PORT}" \
    -v "${LLAMA_MODEL_DIR}:/models:ro" \
    "$LLAMA_IMAGE" \
    -m "$LLAMA_MODEL_FILE" \
    --host 0.0.0.0 \
    --port "$LLAMA_CONTAINER_PORT" \
    --ctx-size 4096 \
    --n-gpu-layers 999 2>&1

  if ! wait_http "http://127.0.0.1:${LLAMA_HOST_PORT}/v1/models" 60 "llama.cpp"; then
    docker logs --tail=50 "$LLAMA_CONTAINER" 2>&1
    cleanup_container "$LLAMA_CONTAINER"
    return 1
  fi

  MODEL_ID=$(curl -sf "http://127.0.0.1:${LLAMA_HOST_PORT}/v1/models" | python3 -c "import sys,json; print(json.load(sys.stdin)['data'][0]['id'])" 2>/dev/null)
  log "llama.cpp model ID: $MODEL_ID"

  RESP=$(curl -sf "http://127.0.0.1:${LLAMA_HOST_PORT}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"$MODEL_ID\",\"messages\":[{\"role\":\"user\",\"content\":\"Say hello in one word\"}],\"max_tokens\":20}")
  echo "$RESP" | python3 -m json.tool > /dev/null || fail "llama.cpp /v1/chat/completions"

  pass "llama.cpp smoke test"
  cleanup_container "$LLAMA_CONTAINER"
}

cmd_runplan() {
  log "=== LightAI RunPlan Tests ==="
  go test ./internal/server/runplan/... -v -run 'TestLlamaCpp|TestResolve' -count=1 2>&1 | tee "$OUTDIR/08-lightai-runplan-nvidia-llamacpp.txt"
  if [ ${PIPESTATUS[0]} -eq 0 ]; then
    pass "RunPlan tests"
  else
    fail "RunPlan tests"
    return 1
  fi
}

cmd_all() {
  local failed=0
  cmd_env || { failed=1; log "env check failed, continuing anyway"; }
  cmd_vllm || failed=1
  cmd_sglang || failed=1
  cmd_llamacpp || failed=1
  cmd_runplan || failed=1
  cmd_cleanup
  if [ $failed -eq 0 ]; then
    log "All smoke tests passed"
  else
    log "Some tests failed — check output above"
    return 1
  fi
}

cmd_cleanup() {
  log "Cleaning up containers..."
  cleanup_container "$VLLM_CONTAINER"
  cleanup_container "$SGLANG_CONTAINER"
  cleanup_container "$LLAMA_CONTAINER"
  log "Cleanup done"
}

# ---- Main ----
case "${1:-help}" in
  env)      cmd_env ;;
  vllm)     cmd_vllm ;;
  sglang)   cmd_sglang ;;
  llamacpp) cmd_llamacpp ;;
  runplan)  cmd_runplan ;;
  all)      cmd_all ;;
  cleanup)  cmd_cleanup ;;
  *)
    echo "Usage: $0 {env|vllm|sglang|llamacpp|runplan|all|cleanup}"
    echo ""
    echo "Commands:"
    echo "  env       Check Docker + NVIDIA environment"
    echo "  vllm      Run vLLM smoke test (port $VLLM_HOST_PORT)"
    echo "  sglang    Run SGLang smoke test (port $SGLANG_HOST_PORT)"
    echo "  llamacpp  Run llama.cpp smoke test (port $LLAMA_HOST_PORT)"
    echo "  runplan   Run LightAI RunPlan tests"
    echo "  all       Run all smoke tests sequentially"
    echo "  cleanup   Remove all smoke test containers"
    exit 1
    ;;
esac
