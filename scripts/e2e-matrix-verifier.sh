#!/bin/bash
# e2e-matrix-verifier.sh — Cross-backend parameter matrix verification.
# Category: DryRun E2E (no containers, no GPU usage beyond API queries)
# Verifies: every backend+vendor combination produces valid DryRun with correct
# conventions (model path format, visible device env, port propagation).

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e/lib/e2e-assert.sh"

SERVER_URL="${SERVER_URL:-http://127.0.0.1:18080}"
USERNAME="${LIGHTAI_E2E_USERNAME:-admin}"
PASSWORD="${LIGHTAI_E2E_PASSWORD:-Commvault!234}"
RUN_ID="${LIGHTAI_E2E_RUN_ID:-$(date +%Y%m%d-%H%M%S)-$$}"
ARTIFACT_DIR="${LIGHTAI_E2E_ARTIFACT_DIR:-/tmp/lightai-e2e-matrix-$RUN_ID}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/lightai-e2e-cookies-$RUN_ID.txt}"
PREFIX="e2e-matrix"
mkdir -p "$ARTIFACT_DIR"

log()   { printf '[%s] [matrix] %s\n' "$(date '+%H:%M:%S')" "$*"; }

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

# Login
log "Logging in..."
resp="$(curl -sS -X POST "$SERVER_URL/api/v1/auth/login" \
  -H "Origin: $SERVER_URL" -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" -c "$COOKIE_JAR")"
CSRF_TOKEN="$(echo "$resp" | json_field csrf_token)"
[ -n "$CSRF_TOKEN" ] || { log "FATAL: Login failed"; exit 1; }

# Discover
NODE_ID=$(api_get "nodes" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if len(d)>0 else '')" 2>/dev/null)
[ -n "$NODE_ID" ] || { log "FATAL: No online node"; exit 1; }
log "Node: $NODE_ID"

# Discover all runtimes
RTS_JSON=$(api_get "backend-runtimes")
echo "$RTS_JSON" > "$ARTIFACT_DIR/backend-runtimes.json"

# Build matrix from available runtimes
declare -a MATRIX_ENTRIES=()
while IFS=$'\t' read -r rt_id vendor backend_name; do
  MATRIX_ENTRIES+=("$rt_id|$vendor|$backend_name")
  log "Matrix entry: $rt_id vendor=$vendor backend=$backend_name"
done < <(echo "$RTS_JSON" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    rid = r.get('id','')
    vendor = r.get('vendor','?')
    parts = rid.split('.')
    backend = parts[1] if len(parts) > 1 else rid
    print(f'{rid}\t{vendor}\t{backend}')
" 2>/dev/null)

log "Matrix size: ${#MATRIX_ENTRIES[@]} combinations"

# Find artifacts
HF_ART_ID=$(api_get "model-artifacts" | python3 -c "import json,sys; [print(a['id']) for a in json.load(sys.stdin) if a.get('format')=='huggingface']" 2>/dev/null | head -1)
GGUF_ART_ID=$(api_get "model-artifacts" | python3 -c "import json,sys; [print(a['id']) for a in json.load(sys.stdin) if a.get('format')=='gguf']" 2>/dev/null | head -1)

log "HF artifact: ${HF_ART_ID:-none}"
log "GGUF artifact: ${GGUF_ART_ID:-none}"

# ── test one combination ──
test_combination() {
  local rt_id="$1" vendor="$2" backend="$3"
  local label="${backend}-${vendor}"

  # Pick artifact: GGUF for llama.cpp/ollama, HF for others
  local art_id
  if echo "$backend" | grep -q "llamacpp\|ollama"; then
    art_id="$GGUF_ART_ID"
  else
    art_id="$HF_ART_ID"
  fi
  if [ -z "$art_id" ]; then
    echo "SKIP:$label:no_artifact" >> "$ARTIFACT_DIR/matrix-results.txt"
    log "  SKIP $label: no artifact"
    return
  fi

  # Ensure NBR ready
  local nbr_status; nbr_status=$(api_get "nodes/$NODE_ID/backend-runtimes" | python3 -c "
import json,sys
for n in json.load(sys.stdin):
    if n.get('backend_runtime_id') == '$rt_id':
        print(n.get('status',''))
        sys.exit(0)
" 2>/dev/null)
  if [ "$nbr_status" != "ready" ]; then
    api_post "nodes/$NODE_ID/backend-runtimes/enable" \
      "{\"backend_runtime_id\":\"$rt_id\",\"image_present\":true,\"docker_available\":true}" > /dev/null 2>&1 || true
  fi

  # Create deployment
  local dep_name="${PREFIX}-${label}"
  local port=$((8500 + RANDOM % 500))
  local dep_resp; dep_resp=$(api_post "deployments" "{\"name\":\"$dep_name\",\"display_name\":\"Matrix $label\",\"model_artifact_id\":\"$art_id\",\"backend_runtime_id\":\"$rt_id\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"accelerator_ids\":[]},\"service_json\":{\"host_port\":$port,\"container_port\":8000,\"app_port\":8000},\"parameters_json\":{}}")
  local dep_id; dep_id=$(echo "$dep_resp" | json_field id)
  if [ -z "$dep_id" ]; then
    echo "FAIL:$label:deploy_create" >> "$ARTIFACT_DIR/matrix-results.txt"
    log "  FAIL $label: deploy create"
    return
  fi
  echo "$dep_resp" > "$ARTIFACT_DIR/${label}-deployment.json"

  # DryRun
  local dr_resp; dr_resp=$(api_post "deployments/$dep_id/dry-run" '{}')
  echo "$dr_resp" > "$ARTIFACT_DIR/${label}-dryrun.json"
  local valid; valid=$(echo "$dr_resp" | json_field valid)
  local preview; preview=$(echo "$dr_resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('command_preview',''))" 2>/dev/null)

  if [ "$valid" != "True" ]; then
    local errs; errs=$(echo "$dr_resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get('errors',[])))" 2>/dev/null)
    # Hardware-blocked failures are SKIPPED_ENV, not product bugs
    if echo "$errs" | grep -q "template_only\|unsupported_device"; then
      echo "SKIP:$label:hardware_blocked" >> "$ARTIFACT_DIR/matrix-results.txt"
      log "  SKIP $label: hardware not available (status in NBR)"
    else
      echo "FAIL:$label:dryrun_invalid:$errs" >> "$ARTIFACT_DIR/matrix-results.txt"
      log "  FAIL $label: dryrun invalid -- $errs"
    fi
  else
    # Backend-specific conventions check
    local issues=""
    case "$backend" in
      llamacpp)
        echo "$preview" | grep -qF -- "-m /models/" || issues="$issues,no_m_flag"
        echo "$preview" | grep -qF -- ".gguf" || issues="$issues,no_gguf_path"
        ;;
      vllm)
        echo "$preview" | grep -qF -- "/models/" || issues="$issues,no_model_path"
        echo "$preview" | grep -qF -- "--model" && issues="$issues,has_model_flag" || true
        ;;
      sglang)
        echo "$preview" | grep -qF -- "--model-path /models/" || issues="$issues,no_model_path_flag"
        ;;
    esac
    # Port mapping
    echo "$preview" | grep -qF -- "-p $port" || issues="$issues,no_port_map"
    # GPU device (NVIDIA only)
    if [ "$vendor" = "nvidia" ]; then
      echo "$preview" | grep -qF -- "CUDA_VISIBLE_DEVICES" || issues="$issues,no_cuda_visible"
    fi
    # Visible device env for MetaX
    if [ "$vendor" = "metax" ]; then
      echo "$preview" | grep -qF -- "CUDA_VISIBLE_DEVICES" || issues="$issues,no_maca_visible"
    fi

    if [ -z "$issues" ]; then
      echo "PASS:$label" >> "$ARTIFACT_DIR/matrix-results.txt"
      log "  PASS $label"
    else
      echo "FAIL:$label$issues" >> "$ARTIFACT_DIR/matrix-results.txt"
      log "  FAIL $label: $issues"
    fi
  fi

  # Cleanup
  api_delete "deployments/$dep_id" > /dev/null 2>&1 || true
}

# ── run matrix ──
log "=== Running matrix ==="
echo "# Matrix results $(date)" > "$ARTIFACT_DIR/matrix-results.txt"
for entry in "${MATRIX_ENTRIES[@]}"; do
  IFS='|' read -r rt_id vendor backend <<< "$entry"
  test_combination "$rt_id" "$vendor" "$backend"
done

# ── summary ──
log "=== Matrix summary ==="
cat "$ARTIFACT_DIR/matrix-results.txt"

PASS_COUNT=$(grep -c "^PASS:" "$ARTIFACT_DIR/matrix-results.txt" 2>/dev/null || echo 0)
FAIL_COUNT=$(grep -c "^FAIL:" "$ARTIFACT_DIR/matrix-results.txt" 2>/dev/null || echo 0)
SKIP_COUNT=$(grep -c "^SKIP:" "$ARTIFACT_DIR/matrix-results.txt" 2>/dev/null || echo 0)

echo ""
echo "Matrix results: $PASS_COUNT pass, $FAIL_COUNT fail, $SKIP_COUNT skip"

# Assert: all non-skipped entries should pass
if [ "$FAIL_COUNT" -gt 0 ]; then
  echo "RESULT: FAIL (unexpected failures)"
  exit 1
fi

echo ""
echo "Artifacts: $ARTIFACT_DIR"
echo "Matrix report: $ARTIFACT_DIR/matrix-results.txt"
echo "RESULT: PASS (PASS=$PASS_COUNT SKIP=$SKIP_COUNT FAIL=0)"
