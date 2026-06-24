#!/usr/bin/env bash
set -euo pipefail
TRACE_BACKEND="${TRACE_BACKEND:-vllm}"
E2E_PASSWORD="${E2E_PASSWORD:-test1234}"
RUN_ID="${RUN_ID:-$(date +%Y%m%d%H%M%S)}"
EVIDENCE="docs/reports/phase-5/evidence/param-trace/${TRACE_BACKEND}"
mkdir -p "$EVIDENCE"
COOKIE_JAR="$(mktemp -t pt-cookies-XXXXXX)"
log() { printf '[%s] [%s] %s\n' "$(date '+%H:%M:%S')" "$TRACE_BACKEND" "$*"; }
json_get() { python3 -c "import json,sys;d=json.load(sys.stdin);print(d.get('$1',''))" 2>/dev/null; }
api() {
  local m="$1" p="$2" d="${3:-}"
  local a=(-sS -X "$m" "http://127.0.0.1:18080$p" -b "$COOKIE_JAR" -c "$COOKIE_JAR" -H "Origin: http://127.0.0.1:18080" -H "Content-Type: application/json")
  [ -n "$CSRF" ] && [ "$m" != "GET" ] && a+=(-H "X-CSRF-Token: $CSRF")
  [ -n "$d" ] && a+=(-d "$d")
  curl "${a[@]}" 2>/dev/null
}

# Login
CSRF=$(curl -sS -X POST http://127.0.0.1:18080/api/v1/auth/login -H "Origin: http://127.0.0.1:18080" -H "Content-Type: application/json" -d "{\"username\":\"admin\",\"password\":\"${E2E_PASSWORD}\"}" -c "$COOKIE_JAR" | json_get csrf_token)
[ -n "$CSRF" ] || { log "FAIL: login"; exit 1; }
log "login ok"

# Get node + GPU
NODE_ID=$(api GET /api/v1/nodes | python3 -c "import json,sys;print(json.load(sys.stdin)[0]['id'])")
GPU_ID=$(api GET /api/v1/gpus | python3 -c "import json,sys;d=json.load(sys.stdin);print(d[0]['id'] if d else '')" 2>/dev/null || echo "")
log "node=$NODE_ID gpu=$GPU_ID"

# Step 1: Read BackendVersion schema
BV_ID="vllm-v0.23.0"
[ "$TRACE_BACKEND" = "sglang" ] && BV_ID="sglang-v0.5.13.post1"
[ "$TRACE_BACKEND" = "llamacpp" ] && BV_ID="llamacpp-b9700"
api GET "/api/v1/backends/backend.${TRACE_BACKEND}/versions" > "$EVIDENCE/00-backend-versions.json"
log "step 1: backend versions saved"

# Step 2: Create model + location
MODEL_PATH="/home/kzeng/models/Qwen3-0.6B-Instruct-2512"
FORMAT="huggingface"
[ "$TRACE_BACKEND" = "llamacpp" ] && MODEL_PATH="/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf"
[ "$TRACE_BACKEND" = "llamacpp" ] && FORMAT="gguf"
ARTIFACT=$(api POST /api/v1/model-artifacts "{\"name\":\"trace-${TRACE_BACKEND}-${RUN_ID}\",\"display_name\":\"Trace ${TRACE_BACKEND} ${RUN_ID}\",\"path\":\"${MODEL_PATH}\",\"format\":\"${FORMAT}\",\"task_type\":\"chat\"}")
ARTIFACT_ID=$(echo "$ARTIFACT" | json_get id)
echo "$ARTIFACT" > "$EVIDENCE/01-model-created.json"
ROOT_PATH=$(dirname "$MODEL_PATH")
ROOT_ID=$(api POST "/api/v1/nodes/$NODE_ID/model-roots" "{\"path\":\"$ROOT_PATH\"}" | json_get id)
REL_PATH="${MODEL_PATH#$ROOT_PATH/}"
api POST "/api/v1/model-artifacts/$ARTIFACT_ID/locations" "{\"node_id\":\"$NODE_ID\",\"root_id\":\"$ROOT_ID\",\"relative_path\":\"$REL_PATH\",\"path_type\":\"$([ "$FORMAT" = "gguf" ] && echo file || echo directory)\",\"verification_status\":\"verified\",\"match_status\":\"exact_match\"}" >/dev/null
log "step 2: artifact=$ARTIFACT_ID location added"

# Step 3: Read BackendRuntime
RT_ID="runtime.${TRACE_BACKEND}.nvidia-docker"
[ "$TRACE_BACKEND" = "llamacpp" ] && RT_ID="runtime.llamacpp.nvidia-docker"
api GET "/api/v1/backend-runtimes/$RT_ID" > "$EVIDENCE/03-backend-runtime.json"
log "step 3: backend runtime saved"

# Step 4: Read NBR
NBR_ID="${NODE_ID}:${RT_ID}"
api GET "/api/v1/nodes/$NODE_ID/backend-runtimes/$NBR_ID" > "$EVIDENCE/05-nbr.json"
log "step 4: NBR saved: $NBR_ID"

# Step 5: Create deployment + preflight
HOST_PORT=8099
[ "$TRACE_BACKEND" = "sglang" ] && HOST_PORT=8098
[ "$TRACE_BACKEND" = "llamacpp" ] && HOST_PORT=8097
DEPLOY=$(api POST /api/v1/deployments "{\"name\":\"trace-${TRACE_BACKEND}-${RUN_ID}\",\"model_artifact_id\":\"${ARTIFACT_ID}\",\"node_backend_runtime_id\":\"${NBR_ID}\",\"placement_json\":{\"node_id\":\"${NODE_ID}\",\"accelerator_ids\":[\"${GPU_ID}\"]},\"service_json\":{\"host_port\":${HOST_PORT}}}")
DEPLOY_ID=$(echo "$DEPLOY" | json_get id)
echo "$DEPLOY" > "$EVIDENCE/07-deployment-created.json"
log "step 5: deployment=$DEPLOY_ID"

# Step 6: Preflight
PREFLIGHT=$(api POST "/api/v1/deployments/$DEPLOY_ID/start" "{}")
echo "$PREFLIGHT" > "$EVIDENCE/09-preflight.json"
PREFLIGHT_STATUS=$(echo "$PREFLIGHT" | python3 -c "import json,sys;d=json.load(sys.stdin);print(d.get('status','') or d.get('ok',''))" 2>/dev/null)
log "step 6: preflight status=$PREFLIGHT_STATUS"

# Step 7: RunPlan (dry-run)
DRYRUN=$(api POST "/api/v1/deployments/$DEPLOY_ID/dry-run" "{}")
echo "$DRYRUN" > "$EVIDENCE/10-runplan.json"
CMD=$(echo "$DRYRUN" | python3 -c "import json,sys;print(json.load(sys.stdin).get('command_preview',''))" 2>/dev/null)
echo "$CMD" > "$EVIDENCE/11-equivalent-command.txt"
log "step 7: command=$(echo "$CMD" | head -c 200)"

# Step 8: Assertions
log "step 8: assertions"
FAIL=0
echo "$CMD" | grep -q -- '--host' || { log "FAIL: --host missing"; FAIL=1; }
echo "$CMD" | grep -q -- '--port' || { log "FAIL: --port missing"; FAIL=1; }
echo "$CMD" | grep -q '/dev/dri' && { log "FAIL: /dev/dri in NVIDIA"; FAIL=1; }
echo "$CMD" | grep -q '/dev/mxcd' && { log "FAIL: /dev/mxcd in NVIDIA"; FAIL=1; }
[ "$PREFLIGHT_STATUS" = "started" ] || [ "$PREFLIGHT_STATUS" = "True" ] || { log "FAIL: preflight not ok (status=$PREFLIGHT_STATUS)"; FAIL=1; }

[ $FAIL -eq 0 ] && log "PASS" || log "FAIL"
rm -f "$COOKIE_JAR"
exit $FAIL
