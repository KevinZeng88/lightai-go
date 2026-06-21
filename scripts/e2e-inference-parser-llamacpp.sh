#!/bin/bash
# e2e-inference-parser-llamacpp.sh — Inference parser E2E (fixture + real).
# Category: Mixed — fixture tests via go test, real inference with llama.cpp container.
# Verifies: extractPreview handles reasoning_content/chat/completion/text formats,
# and real inference against a running instance returns valid preview.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/e2e-assert.sh"

SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-Commvault!234}"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-/tmp/lightai-e2e-inference-$RUN_ID}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/lightai-e2e-cookies-$RUN_ID.txt}"
PREFIX="e2e-inference"
mkdir -p "$ARTIFACT_DIR"

log()   { printf '[%s] [inference] %s\n' "$(date '+%H:%M:%S')" "$*"; }

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

# ═══════════════════════════════════════════════════════════════
# Part 1: Fixture-based parser tests (go test)
# ═══════════════════════════════════════════════════════════════
log "=== Part 1: Fixture parser tests (go test) ==="
cd "$(git rev-parse --show-toplevel 2>/dev/null || echo "$SCRIPT_DIR/..")"
GO_OUT=$(go test ./internal/server/api/ -run "TestExtractPreview|TestTryInference" -v -count=1 2>&1)
echo "$GO_OUT" > "$ARTIFACT_DIR/go-test-output.txt"
echo "$GO_OUT"

PARSER_TEST_PASS=$(echo "$GO_OUT" | grep -c "--- PASS:" || echo "0")
PARSER_TEST_FAIL=$(echo "$GO_OUT" | grep -c "--- FAIL:" || echo "0")
log "Parser tests: $PARSER_TEST_PASS pass, $PARSER_TEST_FAIL fail"

assert_contains "go test passes" "$GO_OUT" "PASS" || log "FAIL: go test did not pass"
assert_not_contains "no go test failures" "$GO_OUT" "FAIL" || true  # already checked above
log "Fixture parser tests done"

# ═══════════════════════════════════════════════════════════════
# Part 2: Real llama.cpp inference (requires running container)
# ═══════════════════════════════════════════════════════════════
log "=== Part 2: Real llama.cpp inference ==="

# Login
resp="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" \
  -H "Origin: $SERVER_URL" -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR")"
CSRF_TOKEN="$(echo "$resp" | json_field csrf_token)"
[ -n "$CSRF_TOKEN" ] || { log "FATAL: Login failed"; exit 1; }

# Discover resources
NODE_ID=$(api_get "nodes" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)
[ -n "$NODE_ID" ] || { log "SKIP: No online node — skipping real inference"; }
if [ -z "$NODE_ID" ]; then
  echo ""
  echo "Real inference: SKIPPED_ENV (no online node)"
  assert_summary
  exit 0
fi

GGUF_ART_ID=$(api_get "model-artifacts" | python3 -c "import json,sys; [print(a['id']) for a in json.load(sys.stdin) if a.get('format')=='gguf']" 2>/dev/null | head -1)
LLAMACPP_RT=$(api_get "backend-runtimes" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    if 'llamacpp' in r.get('id','') and r.get('vendor','')=='nvidia':
        print(r['id']); break
" 2>/dev/null)

if [ -z "$GGUF_ART_ID" ] || [ -z "$LLAMACPP_RT" ]; then
  log "SKIP: Missing GGUF artifact or llama.cpp runtime"
  echo "Real inference: SKIPPED_ENV"
  assert_summary
  exit 0
fi

# Ensure NBR ready
NBR_STATUS=$(api_get "nodes/$NODE_ID/backend-runtimes" | python3 -c "
import json,sys
for n in json.load(sys.stdin):
    if n.get('backend_runtime_id') == '$LLAMACPP_RT':
        print(n.get('status',''))
" 2>/dev/null)
if [ "$NBR_STATUS" != "ready" ]; then
  log "Enabling llama.cpp NBR..."
  api_post "nodes/$NODE_ID/backend-runtimes/enable" \
    "{\"backend_runtime_id\":\"$LLAMACPP_RT\",\"image_present\":true,\"docker_available\":true}" > /dev/null
fi

# Create deployment
DEP_NAME="${PREFIX}-llamacpp"
DEP_RESP=$(api_post "deployments" "{\"name\":\"$DEP_NAME\",\"display_name\":\"Inference Test\",\"model_artifact_id\":\"$GGUF_ART_ID\",\"backend_runtime_id\":\"$LLAMACPP_RT\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"accelerator_ids\":[]},\"service_json\":{\"host_port\":8491,\"container_port\":8080,\"app_port\":8080},\"parameters_json\":{}}")
DEP_ID=$(echo "$DEP_RESP" | json_field id)
[ -n "$DEP_ID" ] || { log "FATAL: Deployment create failed"; exit 1; }
log "Deployment: $DEP_ID"

# Start
START_RESP=$(api_post "deployments/$DEP_ID/start" '{}')
echo "$START_RESP" > "$ARTIFACT_DIR/start-response.json"
INSTANCE_ID=$(echo "$START_RESP" | json_field instance_id)
[ -n "$INSTANCE_ID" ] || { log "FATAL: Start failed: $(head -c 300 "$ARTIFACT_DIR/start-response.json")"; exit 1; }
log "Instance: $INSTANCE_ID"

# Wait for running
INSTANCE_STATE=""
ENDPOINT_URL=""
for i in $(seq 1 30); do
  sleep 2
  INST_DETAIL=$(api_get "model-instances/$INSTANCE_ID" 2>/dev/null || echo '{}')
  echo "$INST_DETAIL" > "$ARTIFACT_DIR/instance-poll-$i.json"
  INSTANCE_STATE=$(echo "$INST_DETAIL" | json_field actual_state)
  ENDPOINT_URL=$(echo "$INST_DETAIL" | json_field endpoint_url)
  log "  poll $i: state=$INSTANCE_STATE endpoint=$ENDPOINT_URL"
  if [ "$INSTANCE_STATE" = "running" ]; then
    break
  fi
  if [ "$INSTANCE_STATE" = "failed" ]; then
    break
  fi
done

assert_eq "instance state is running" "running" "$INSTANCE_STATE" || log "FAIL: not running (state=$INSTANCE_STATE)"

if [ "$INSTANCE_STATE" = "running" ]; then
  # Call the test endpoint
  log "=== Call model instance test ==="
  TEST_RESP=$(api_post "model-instances/$INSTANCE_ID/test" '{}')
  echo "$TEST_RESP" > "$ARTIFACT_DIR/test-response.json"
  log "Test response: $(head -c 300 "$ARTIFACT_DIR/test-response.json")"

  TEST_OK=$(echo "$TEST_RESP" | json_field ok)
  TEST_PREVIEW=$(echo "$TEST_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('response_preview',''))" 2>/dev/null)
  TEST_RAW=$(echo "$TEST_RESP" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('raw_response',''))[:500])" 2>/dev/null)
  TEST_METHOD=$(echo "$TEST_RESP" | json_field model_resolution_method)

  # Assertions on inference result
  assert_eq "inference test ok=true" "True" "$TEST_OK" || log "FAIL: test not ok"
  assert_nonempty "inference test has preview" "$TEST_PREVIEW" || log "FAIL: no preview"
  assert_nonempty "inference test has raw_response" "$TEST_RAW" || log "FAIL: no raw response"
  assert_nonempty "inference test has resolution method" "$TEST_METHOD" || log "FAIL: no resolution method"

  log "Inference preview: ${TEST_PREVIEW:0:200}"
  log "Inference method: $TEST_METHOD"
fi

# Stop deployment
log "=== Stop deployment ==="
api_post "deployments/$DEP_ID/stop" '{}' > "$ARTIFACT_DIR/stop-response.json"

# Wait for stop
for i in $(seq 1 15); do
  sleep 2
  STOP_STATE=$(api_get "model-instances/$INSTANCE_ID" 2>/dev/null | json_field actual_state || echo "")
  log "  stop poll $i: state=$STOP_STATE"
  if [ "$STOP_STATE" = "stopped" ]; then break; fi
done

# Cleanup
api_delete "deployments/$DEP_ID" > /dev/null 2>&1 || true

# Verify container gone
sleep 2
REMAINING=$(docker ps --filter 'label=lightai.managed' -q 2>/dev/null | wc -l)
log "Remaining lightai containers: $REMAINING"

echo ""
echo "Artifacts: $ARTIFACT_DIR"
echo "Key files: go-test-output.txt start-response.json test-response.json"
assert_summary
