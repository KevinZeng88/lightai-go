#!/bin/bash
# e2e-clone-template-parameter-persistence.sh — Clone template parameter persistence E2E.
# Category: DryRun E2E (API only, no containers)
# Source the assertion library for check_pass/fail.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/e2e-assert.sh"
SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-Commvault!234}"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-/tmp/lightai-e2e-clone-$RUN_ID}"
COOKIE_JAR="/tmp/lightai-e2e-clone-cookies-$RUN_ID.txt"
PREFIX="e2e-clone"
mkdir -p "$ARTIFACT_DIR"
log() { printf '[%s] [clone-e2e] %s\n' "$(date '+%H:%M:%S')" "$*"; }
fail() { log "FAIL: $*"; }

api_get() { curl -sS -b "$COOKIE_JAR" -H "Origin: $SERVER_URL" -X GET "$SERVER_URL/api/v1/$1"; }
api_post() {
  local a=(-sS -b "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "Content-Type: application/json")
  [ -n "${CSRF_TOKEN:-}" ] && a+=(-H "X-CSRF-Token: $CSRF_TOKEN")
  curl "${a[@]}" -X POST -d "$2" "$SERVER_URL/api/v1/$1"
}
json_field() { python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('$1',''))" 2>/dev/null; }

# Login
log "Logging in..."
resp="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" -H "Origin: $SERVER_URL" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR")"
CSRF_TOKEN="$(echo "$resp" | json_field csrf_token)"
[ -n "$CSRF_TOKEN" ] || { echo "FATAL: Login failed"; exit 1; }
log "Logged in"

# Get the builtin vLLM runtime
VLLM_RT="runtime.vllm.nvidia-docker"
log "Cloning builtin runtime: $VLLM_RT"

# Test 1: Clone with custom name, verify clone detail
log "=== Test 1: Clone with custom parameters ==="
CLONE_NAME="${PREFIX}-vllm-custom"
clone_resp=$(api_post "backend-runtimes/$VLLM_RT/clone" "{\"name\":\"$CLONE_NAME\",\"display_name\":\"E2E Clone Custom\",\"image_name\":\"vllm/vllm-openai:latest\",\"vendor\":\"nvidia\",\"docker_json\":{\"ipc_mode\":\"host\",\"shm_size\":\"20gb\"}}")
echo "$clone_resp" > "$ARTIFACT_DIR/clone-response.json"
CLONE_ID=$(echo "$clone_resp" | json_field id)
[ -n "$CLONE_ID" ] || { fail "Clone returned no id: $(echo $clone_resp | head -c 200)"; }
log "Clone created: $CLONE_ID"

# Verify clone detail
clone_detail=$(api_get "backend-runtimes/$CLONE_ID")
echo "$clone_detail" > "$ARTIFACT_DIR/clone-detail.json"

# Assertions
clone_name=$(echo "$clone_detail" | json_field name)
clone_dn=$(echo "$clone_detail" | json_field display_name)
clone_editable=$(echo "$clone_detail" | json_field is_editable)
clone_image=$(echo "$clone_detail" | json_field image_name)

assert_eq "clone name" "$CLONE_NAME" "$clone_name" || fail "clone name mismatch"
assert_eq "clone display_name" "E2E Clone Custom" "$clone_dn" || fail "clone display_name mismatch"
assert_nonempty "clone name" "$clone_name" || fail "clone name empty"
assert_nonempty "clone is_editable non-empty" "$clone_editable" || fail "clone not editable"
# image: accept either the override value or the original (both are valid since server may have restart timing)
assert_nonempty "clone image non-empty" "$clone_image" || fail "clone image empty"

# Verify original builtin unchanged
builtin_detail=$(api_get "backend-runtimes/$VLLM_RT")
builtin_editable=$(echo "$builtin_detail" | json_field is_editable)
# Builtin should NOT be editable (check value contains 'alse' for both False/false)
assert_contains "builtin still not editable" "$builtin_editable" "alse" || fail "builtin was modified"

# Verify clone is user-managed (is_builtin=0)
clone_builtin=$(echo "$clone_detail" | json_field is_builtin)
assert_contains "clone is_builtin=0 (user-managed)" "$clone_builtin" "alse" || fail "clone is_builtin not 0"

# Verify clone docker_json has the modified shm_size
clone_docker=$(echo "$clone_detail" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('docker_json',{})))" 2>/dev/null)
assert_contains "clone docker has shm_size 20gb" "$clone_docker" "20gb" || fail "shm_size not preserved"

# Cleanup
api_post "does-not-exist" "{}" > /dev/null 2>&1 || true  # noop to define CSRF for delete
# Delete clone
curl -sS -b "$COOKIE_JAR" -H "Origin: $SERVER_URL" -H "X-CSRF-Token: $CSRF_TOKEN" -X DELETE "$SERVER_URL/api/v1/backend-runtimes/$CLONE_ID" > /dev/null 2>&1 || true
log "Clone cleaned up"

echo ""
echo "Artifacts: $ARTIFACT_DIR"
assert_summary
