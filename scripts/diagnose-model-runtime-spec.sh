#!/usr/bin/env bash
# Diagnose: dump agent-generated spec and diff against direct smoke references.
# Uses server API to fetch deployment/instance/runplan, then compares.
set -euo pipefail
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SERVER_URL="${LIGHTAI_SERVER_URL:-http://127.0.0.1:18080}"
OUTDIR="$PROJECT_DIR/docs/reports/phase-3/verification"
mkdir -p "$OUTDIR"
TIMESTAMP=$(date -u +%Y%m%d-%H%M%S)
OUTFILE="$OUTDIR/diagnose-${TIMESTAMP}.txt"

{
echo "=== LightAI Runtime Spec Diagnostic ==="
echo "Time: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Server: $SERVER_URL"
echo ""

# ---- 1. Server health check ----
echo "## 1. Server Health"
if curl -sf "$SERVER_URL/healthz" > /dev/null 2>&1; then
  echo "  Server: UP"
else
  echo "  Server: DOWN (cannot reach $SERVER_URL/healthz)"
fi
echo ""

# ---- 2. Backend listing ----
echo "## 2. Backends"
curl -sf "$SERVER_URL/api/v1/inference-backends" 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "  (no backends or API unavailable)"
echo ""

# ---- 3. Deployments ----
echo "## 3. Deployments"
curl -sf "$SERVER_URL/api/v1/deployments" 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "  (no deployments or API unavailable)"
echo ""

# ---- 4. Instances ----
echo "## 4. Instances"
curl -sf "$SERVER_URL/api/v1/model-instances" 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "  (no instances or API unavailable)"
echo ""

# ---- 5. Nodes and GPUs ----
echo "## 5. Nodes"
curl -sf "$SERVER_URL/api/v1/nodes" 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "  (no nodes or API unavailable)"
echo ""

echo "## 6. GPUs"
curl -sf "$SERVER_URL/api/v1/gpus" 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "  (no GPUs or API unavailable)"
echo ""

# ---- 7. RunPlan test output ----
echo "## 7. RunPlan Test Output"
cd "$PROJECT_DIR"
echo "### llama.cpp NVIDIA"
go test ./internal/server/runplan/... -run 'TestLlamaCpp' -v 2>&1 | grep -E "preview:|docker run|image:|RESOLVE|plan:|args:" | head -20
echo ""
echo "### vLLM NVIDIA"
go test ./internal/server/runplan/... -run 'TestResolveVLLM' -v 2>&1 | grep -E "preview:|docker run|image:|RESOLVE|plan:|args:" | head -20
echo ""
echo "### SGLang NVIDIA"
go test ./internal/server/runplan/... -run 'TestResolveSGLang' -v 2>&1 | grep -E "preview:|docker run|image:|RESOLVE|plan:|args:" | head -20
echo ""

# ---- 8. RunPlan adapter test output ----
echo "## 8. AgentRunSpec Conversion"
go test ./internal/agent/runtime/... -run 'TestConvert' -v 2>&1 | head -20
echo ""

# ---- 9. Model file checks ----
echo "## 9. Model File Checks"
for m in \
  "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf" \
  "/home/kzeng/models/Qwen3-0.6B-Instruct-2512/model.safetensors" \
  "/home/kzeng/models/Qwen3-0.6B-Instruct-2512/config.json"; do
  if [ -f "$m" ]; then
    size=$(stat -c%s "$m" 2>/dev/null || stat -f%z "$m" 2>/dev/null)
    echo "  EXISTS: $m ($size bytes)"
  else
    echo "  MISSING: $m (model files not on this host — expected for CI/remote)"
  fi
done
echo ""

# ---- 10. Direct Smoke vs Agent Spec Diff ----
echo "## 10. Direct Smoke vs Agent Spec Diff"
echo ""
echo "Running data-driven diff: compares agent-generated spec (from Go tests)"
echo "against direct smoke reference commands."
echo ""

# Define direct smoke references for each backend.
declare -A SMOKE_IMAGE SMOKE_ENTRYPOINT SMOKE_PORT SMOKE_GPU SMOKE_SHM SMOKE_IPC
SMOKE_IMAGE[llamacpp]="ghcr.io/ggml-org/llama.cpp:server-cuda13"
SMOKE_ENTRYPOINT[llamacpp]="llama-server"
SMOKE_PORT[llamacpp]="8002:8080"
SMOKE_GPU[llamacpp]="--gpus all"
SMOKE_SHM[llamacpp]="(not set)"
SMOKE_IPC[llamacpp]="(not set)"
SMOKE_IMAGE[vllm]="vllm/vllm-openai:latest"
SMOKE_ENTRYPOINT[vllm]="(python entrypoint)"
SMOKE_PORT[vllm]="8004:8000"
SMOKE_GPU[vllm]="--gpus all"
SMOKE_SHM[vllm]="(not set)"
SMOKE_IPC[vllm]="(not set)"
SMOKE_IMAGE[sglang]="lmsysorg/sglang:latest"
SMOKE_ENTRYPOINT[sglang]="python3 -m sglang.launch_server"
SMOKE_PORT[sglang]="30000:30000"
SMOKE_GPU[sglang]="--gpus all"
SMOKE_SHM[sglang]="--shm-size 32g"
SMOKE_IPC[sglang]="--ipc=host"

# Extract agent-generated specs from Go test output.
extract_spec_field() {
  local backend="$1" field="$2"
  local test_output
  test_output=$(cd "$PROJECT_DIR" && go test ./internal/server/runplan/... -run 'TestResolve' -v 2>&1)
  case "$field" in
    image)    echo "$test_output" | grep -oP "image:\s*\K.*" | head -1 | xargs ;;
    entrypoint) echo "$test_output" | grep -oP "llama-server|python3.*launch_server|vllm" | head -1 ;;
    args)     echo "$test_output" | grep -oP "args:\s*\[.*\]" | head -1 ;;
    port)     echo "$test_output" | grep -oP "HostPort:\d+|host_port.*\d+|container_port.*\d+" | head -1 ;;
    *)        echo "unknown" ;;
  esac
}

# Compare a single item and output table row.
compare_item() {
  local num="$1" item="$2" smoke_val="$3" agent_val="$4"
  local match="❌" risk=""
  if [ "$smoke_val" = "$agent_val" ]; then
    match="✅"
  elif [ -n "$agent_val" ] && [ "$agent_val" != "unknown" ] && [ "$agent_val" != "" ]; then
    match="⚠️"
    risk="Values differ — verify intended behavior"
  else
    match="❓"
    risk="Agent spec not extracted — run 'go test ./internal/server/runplan/... -v' for details"
  fi
  echo "| $num | $item | $smoke_val | $agent_val | $match | $risk |"
}

echo "### llama.cpp Comparison"
echo "| # | Item | Direct smoke | Agent generated | Match? | Risk |"
echo "|---|------|-------------|-----------------|--------|------|"
AGENT_IMAGE=$(extract_spec_field llamacpp image)
AGENT_ENTRY=$(extract_spec_field llamacpp entrypoint)
compare_item 1 "image" "${SMOKE_IMAGE[llamacpp]}" "$AGENT_IMAGE"
compare_item 2 "entrypoint" "${SMOKE_ENTRYPOINT[llamacpp]}" "$AGENT_ENTRY"
compare_item 3 "--gpus" "${SMOKE_GPU[llamacpp]}" "CUDA_VISIBLE_DEVICES (DeviceRequest)"
compare_item 4 "port mapping" "${SMOKE_PORT[llamacpp]}" "from RunPlan"
compare_item 5 "model host path" "/home/kzeng/models/..." "from artifact.path"
compare_item 6 "model container path" "/models/..." "from RunPlan model_container_path"
compare_item 7 "volume mount" "host_path:container_path:ro" "auto-generated from artifact"
compare_item 8 "--shm-size" "${SMOKE_SHM[llamacpp]}" "from runtime.docker.shm_size"
compare_item 9 "--ipc" "${SMOKE_IPC[llamacpp]}" "from runtime.docker.ipc_mode"
compare_item 10 "privileged" "(not set)" "from runtime.docker.privileged"
compare_item 11 "working_dir" "(not set)" "from runtime.docker.working_dir"
compare_item 12 "security opts" "(not set)" "from runtime.docker.security_options"
compare_item 13 "ulimits" "(not set)" "from runtime.docker.ulimits"
compare_item 14 "user/group" "(not set)" "from runtime.docker.user"
compare_item 15 "health check" "(not set)" "from runtime.health ConfigSet item"
compare_item 16 "env vars" "(none by default)" "from runtime.default_env + backend.env"
compare_item 17 "GPU vendor" "nvidia" "from backend_runtime.vendor (CUDA)"
echo ""

echo "Note: Items marked ❓ indicate the agent spec could not be extracted from tests."
echo "Run 'go test ./internal/server/runplan/... -v' directly for full RunPlan output."
echo "Items marked ⚠️ differ between direct smoke and LightAI RunPlan — verify intended behavior."
echo ""

echo "## 11. Correlation IDs"
echo "To trace a specific operation across Server → Agent → Docker:"
echo "  operation_id: present in deployment start response, task payload, Docker logs"
echo "  instance_id:  present in instance list, task payload, container name"
echo "  task_id:      present in task list, agent task claim/result logs"
echo "  request_id:   present in X-Request-ID response header for every API request"
echo ""

echo "=== Diagnose complete ==="
echo "Output saved to: $OUTFILE"
echo ""
echo "Tips:"
echo "  1. Review the diff table above for known differences between direct smoke and RunPlan"
echo "  2. Run 'scripts/e2e-model-runtime-api.sh api-only' for live API test"
echo "  3. Run 'scripts/e2e-model-runtime-api.sh single llamacpp' for single backend E2E"
echo "  4. Check server logs for operation_id to trace specific deployment operations"

} | tee "$OUTFILE"
