#!/usr/bin/env bash
# Clone template parameter persistence E2E.
# Tier: API-only local E2E. Requires an existing LightAI server.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export LIGHTAI_E2E_PREFIX="${LIGHTAI_E2E_PREFIX:-e2e-clone-$(date +%Y%m%d-%H%M%S)-$$}"
export LIGHTAI_E2E_ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-${ARTIFACT_DIR:-$SCRIPT_DIR/../tmp/e2e-clone-$(date +%Y%m%d-%H%M%S)-$$}}"

source "$SCRIPT_DIR/e2e/lib/env.sh"
source "$SCRIPT_DIR/e2e/lib/api-client.sh"
source "$SCRIPT_DIR/e2e/lib/assert.sh"
source "$SCRIPT_DIR/e2e/lib/resources.sh"
source "$SCRIPT_DIR/e2e/lib/cleanup.sh"

e2e_with_cleanup_trap

log() { printf '[%s] [clone-e2e] %s\n' "$(date '+%H:%M:%S')" "$*"; }

json_field() {
  python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('$1',''))" 2>/dev/null
}

e2e_wait_server_ready 30
e2e_login

VLLM_RT="runtime.vllm.nvidia-docker"
CLONE_NAME="$(e2e_resource_name "vllm-custom")"

log "cloning builtin runtime: $VLLM_RT"
clone_resp="$(e2e_api_post "backend-runtimes/$VLLM_RT/clone" "{\"name\":\"$CLONE_NAME\",\"display_name\":\"E2E Clone Custom\",\"image_name\":\"vllm/vllm-openai:latest\",\"vendor\":\"nvidia\",\"docker_json\":{\"ipc_mode\":\"host\",\"shm_size\":\"20gb\"}}" 201)"
printf '%s\n' "$clone_resp" > "$LIGHTAI_E2E_ARTIFACT_DIR/clone-response.json"
CLONE_ID="$(printf '%s' "$clone_resp" | json_field id)"
[ -n "$CLONE_ID" ] || e2e_die "clone returned no id"
e2e_register_resource backend_runtime "$CLONE_ID" "/api/v1/backend-runtimes/$CLONE_ID"
e2e_cleanup_add "curl -sS -b '$LIGHTAI_E2E_COOKIE_JAR' -H 'Origin: $LIGHTAI_SERVER_URL' -H 'X-CSRF-Token: $E2E_CSRF_TOKEN' -X DELETE '$LIGHTAI_SERVER_URL/api/v1/backend-runtimes/$CLONE_ID' >/dev/null 2>&1 || true"

clone_detail="$(e2e_api_get "backend-runtimes/$CLONE_ID")"
printf '%s\n' "$clone_detail" > "$LIGHTAI_E2E_ARTIFACT_DIR/clone-detail.json"

clone_name="$(printf '%s' "$clone_detail" | json_field name)"
clone_dn="$(printf '%s' "$clone_detail" | json_field display_name)"
clone_editable="$(printf '%s' "$clone_detail" | json_field is_editable)"
clone_image="$(printf '%s' "$clone_detail" | json_field image_name)"

assert_eq "clone name" "$CLONE_NAME" "$clone_name"
assert_eq "clone display_name" "E2E Clone Custom" "$clone_dn"
assert_nonempty "clone is_editable non-empty" "$clone_editable"
assert_nonempty "clone image non-empty" "$clone_image"

builtin_detail="$(e2e_api_get "backend-runtimes/$VLLM_RT")"
builtin_editable="$(printf '%s' "$builtin_detail" | json_field is_editable)"
assert_contains "builtin still not editable" "$builtin_editable" "alse"

clone_builtin="$(printf '%s' "$clone_detail" | json_field is_builtin)"
assert_contains "clone is_builtin=0 (user-managed)" "$clone_builtin" "alse"

clone_docker="$(printf '%s' "$clone_detail" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('docker_json',{})))" 2>/dev/null)"
assert_contains "clone docker has shm_size 20gb" "$clone_docker" "20gb"

log "Artifacts: $LIGHTAI_E2E_ARTIFACT_DIR"
assert_summary
