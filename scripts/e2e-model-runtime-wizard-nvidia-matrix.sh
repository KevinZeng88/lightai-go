#!/usr/bin/env bash
set -euo pipefail
# Matrix wrapper: runs llama.cpp, vLLM, SGLang with default + modified params.
# Uses existing proven E2E scripts with parameter overrides.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_ID="${RUN_ID:-$(date +%Y%m%d-%H%M%S)}"
VERIFY_BASE="${VERIFY_BASE:-docs/reports/model-runtime-node-wizard/e2e-matrix-${RUN_ID}}"
mkdir -p "$VERIFY_BASE"
MATRIX_EXIT=0

log() { printf '[%s] [matrix] %s\n' "$(date '+%H:%M:%S')" "$*"; }

run_one() {
  local label="$1" script="$2" deploy_params="${3:-}"
  log "===== $label start ====="
  export DEPLOY_PARAMS="$deploy_params"
  if bash "$script" 2>&1 | tee "$VERIFY_BASE/${label}.log"; then
    log "$label: PASS"
    echo "$label: PASS"
  else
    log "$label: FAIL"
    echo "$label: FAIL"
    MATRIX_EXIT=1
  fi
}

# ── llama.cpp ──
run_one "llamacpp-default"     "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-llamacpp.sh" ""
run_one "llamacpp-modified"    "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-llamacpp.sh" '"--ctx-size":"2048","--n-gpu-layers":"-1"'

# ── vLLM ──
run_one "vllm-default"         "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-api.sh" ""
run_one "vllm-modified"        "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-api.sh" '"--max-model-len":"2048","--gpu-memory-utilization":"0.80"'

# ── SGLang ──
run_one "sglang-default"       "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-sglang.sh" ""
run_one "sglang-modified"      "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-sglang.sh" '"--tp":"1"'

# ── Save logs ──
tail -n 5000 logs/lightai-server.log > "$VERIFY_BASE/server-this-run.log" 2>/dev/null || true
tail -n 5000 logs/lightai-agent.log > "$VERIFY_BASE/agent-this-run.log" 2>/dev/null || true
docker ps -a --format 'table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}' > "$VERIFY_BASE/docker-ps-after.txt" 2>/dev/null || true

echo ""
echo "========== Matrix Summary =========="
grep -E ': (PASS|FAIL)$' "$VERIFY_BASE"/*.log 2>/dev/null || true
echo "Matrix exit=$MATRIX_EXIT"
exit $MATRIX_EXIT
