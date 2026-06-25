#!/usr/bin/env bash
# LightAI Bootstrap — Unified environment auto-initialization tool.
# Usage: bash scripts/lightai-bootstrap.sh [--mode <mode>] [--profile <path>] [...]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# ── Builtin defaults ────────────────────────────────────────────────
DEFAULT_PROFILE="configs/bootstrap/local-kz-laptop.yaml"
DEFAULT_MODE="dry-run"
DEFAULT_BASE_URL="http://localhost:18080"
DEFAULT_AGENT_URL="http://localhost:19091"
DEFAULT_RUNTIME_DIR="/tmp/lightai"
DEFAULT_OUTPUT_DIR="/tmp/lightai/e2e/bootstrap"
DEFAULT_INITIAL_PASSWORD_ENV="LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD"
DEFAULT_ADMIN_PASSWORD_ENV="LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD"

# ── CLI parsing ─────────────────────────────────────────────────────
PROFILE=""
MODE=""
BASE_URL=""
AGENT_URL=""
RUNTIME_DIR=""
OUTPUT_DIR=""
INITIAL_PASSWORD=""
INITIAL_PASSWORD_FILE=""
ADMIN_PASSWORD=""
ADMIN_PASSWORD_FILE=""
ALLOW_REAL_START=""
ALLOW_CHAT_COMPLETION=""
OUTPUT_PROFILE=""
INCLUDE_SECRETS=""
INCLUDE_RUNTIME_STATE=""
YES_FLAG=""
SHOW_HELP=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)          PROFILE="$2"; shift 2 ;;
    --mode)             MODE="$2"; shift 2 ;;
    --base-url)         BASE_URL="$2"; shift 2 ;;
    --agent-url)        AGENT_URL="$2"; shift 2 ;;
    --runtime-dir)      RUNTIME_DIR="$2"; shift 2 ;;
    --output-dir)       OUTPUT_DIR="$2"; shift 2 ;;
    --initial-password) INITIAL_PASSWORD="$2"; shift 2 ;;
    --initial-password-file) INITIAL_PASSWORD_FILE="$2"; shift 2 ;;
    --admin-password)   ADMIN_PASSWORD="$2"; shift 2 ;;
    --admin-password-file)   ADMIN_PASSWORD_FILE="$2"; shift 2 ;;
    --allow-real-start)      ALLOW_REAL_START="true"; shift ;;
    --allow-chat-completion) ALLOW_CHAT_COMPLETION="true"; shift ;;
    --output-profile)   OUTPUT_PROFILE="$2"; shift 2 ;;
    --include-secrets)  INCLUDE_SECRETS="true"; shift ;;
    --include-runtime-state) INCLUDE_RUNTIME_STATE="true"; shift ;;
    --yes)              YES_FLAG="true"; shift ;;
    --help|-h)          SHOW_HELP="true"; shift ;;
    *) echo "ERROR: unknown option: $1" >&2; exit 2 ;;
  esac
done

# ── Help ─────────────────────────────────────────────────────────────
if [[ "$SHOW_HELP" == "true" ]]; then
  cat << 'HELP'
LightAI Bootstrap — Unified environment auto-initialization.

Usage:
  bash scripts/lightai-bootstrap.sh [options]

Options:
  --profile <path>           Bootstrap YAML profile (default: configs/bootstrap/local-kz-laptop.yaml)
  --mode <mode>              Execution mode: auth-only, catalog-only, models-only, runtimes-only, dry-run, full, export
  --base-url <url>           Server URL (default: http://localhost:18080)
  --agent-url <url>          Agent URL (default: http://localhost:19091)
  --runtime-dir <dir>        Runtime directory (default: /tmp/lightai)
  --output-dir <dir>         Output directory (default: /tmp/lightai/e2e/bootstrap)
  --initial-password <val>   Initial admin password (NOT logged)
  --initial-password-file <path> File containing initial password
  --admin-password <val>     Target admin password (NOT logged)
  --admin-password-file <path> File containing target password
  --allow-real-start         Allow full mode to start real containers
  --allow-chat-completion    Allow full mode to execute chat completion
  --output-profile <path>    Export output profile path
  --include-secrets          Export include sensitive fields (requires --yes)
  --include-runtime-state    Export include runtime state
  --yes                      Confirm high-risk operations
  --help, -h                 Show this help

Modes:
  auth-only       Server/agent check, login, initial password change
  catalog-only    auth-only + verify catalog backends
  models-only     catalog-only + register model artifacts and locations
  runtimes-only   models-only + configure BackendRuntime and NBR
  dry-run         runtimes-only + enable/check/preflight
  full            dry-run + deploy, start, health check (requires --allow-real-start)
  export          Export current environment to a reusable profile

Default command (no args):
  bash scripts/lightai-bootstrap.sh
  Equivalent to: --profile configs/bootstrap/local-kz-laptop.yaml --mode dry-run

Password variables:
  LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD  Initial admin password (canonical)
  LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD    Target admin password
HELP
  exit 0
fi

# ── Resolve project-relative paths ──────────────────────────────────
resolve_path() {
  local p="$1"
  if [[ "$p" == /* ]]; then echo "$p"; else echo "$PROJECT_DIR/$p"; fi
}

# ── Profile loading ─────────────────────────────────────────────────
DEFAULT_PROFILE_PATH="$(resolve_path "$DEFAULT_PROFILE")"
PROFILE_PATH="${PROFILE:-$DEFAULT_PROFILE_PATH}"
# If profile is relative, resolve from project dir
if [[ -n "$PROFILE" && "$PROFILE" != /* ]]; then
  PROFILE_PATH="$(resolve_path "$PROFILE")"
fi

parse_profile_field() {
  # Simple YAML scalar field parser using grep + awk.
  # Only handles top-level keys and simple nested patterns.
  # Full YAML parsing requires python3 fallback (see below).
  local file="$1" key="$2" subkey="$3"
  if [[ ! -f "$file" ]]; then return 1; fi
  if [[ -n "$subkey" ]]; then
    awk -v k="$key" -v sk="$subkey" '
      $0 ~ "^"k":" { in_section=1; next }
      in_section && $0 ~ "^  "sk":" { gsub(/^  [^:]*: ?/, ""); gsub(/^"/, ""); gsub(/"$/, ""); print; exit }
      in_section && $0 ~ "^[a-z]" { exit }
    ' "$file" 2>/dev/null
  else
    awk -v k="$key" '$0 ~ "^"k":" { gsub(/^[^:]*: ?/, ""); gsub(/^"/, ""); gsub(/"$/, ""); print; exit }' "$file" 2>/dev/null
  fi
}

# Try python3 for reliable YAML parsing; fall back to grep-based.
yaml_get() {
  local file="$1" key="$2" subkey="${3:-}"
  if command -v python3 >/dev/null 2>&1; then
    local py_expr="import sys,yaml;d=yaml.safe_load(open(sys.argv[1]));k=sys.argv[2];sk=sys.argv[3] if len(sys.argv)>3 else ''"
    py_expr="$py_expr;v=d.get(k,{}) if sk else d.get(k,'')"
    py_expr="$py_expr;v=v.get(sk,'') if sk and isinstance(v,dict) else v"
    py_expr="$py_expr;print(v if v is not None else '')"
    python3 -c "$py_expr" "$file" "$key" "$subkey" 2>/dev/null
  else
    parse_profile_field "$file" "$key" "$subkey"
  fi
}

yaml_get_list() {
  local file="$1" key="$2" subkey="$3"
  if command -v python3 >/dev/null 2>&1; then
    python3 -c "
import sys,yaml
d=yaml.safe_load(open(sys.argv[1]))
v=d.get(sys.argv[2],{}).get(sys.argv[3],[]) if sys.argv[3] else d.get(sys.argv[2],[])
if isinstance(v,list): print('\n'.join(str(x) for x in v))
" "$file" "$key" "$subkey" 2>/dev/null
  else
    return 1
  fi
}

# ── Parameter merging: CLI > env > profile > builtin defaults ───────
merge_val() {
  # Priority: CLI arg > env var > profile > default
  local cli="$1" env_name="$2" profile_file="$3" profile_key="$4" profile_subkey="$5" default_val="$6"
  if [[ -n "$cli" ]]; then echo "$cli"; return; fi
  if [[ -n "$env_name" ]]; then
    local env_val="${!env_name:-}"
    if [[ -n "$env_val" ]]; then echo "$env_val"; return; fi
  fi
  if [[ -f "$profile_file" ]]; then
    local pf_val
    pf_val=$(yaml_get "$profile_file" "$profile_key" "$profile_subkey" 2>/dev/null) || true
    if [[ -n "${pf_val:-}" ]]; then echo "$pf_val"; return; fi
  fi
  echo "$default_val"
}

PROFILE_FILE="$PROFILE_PATH"
FINAL_MODE="$(merge_val "$MODE" "" "$PROFILE_FILE" "bootstrap" "default_mode" "$DEFAULT_MODE")"
FINAL_BASE_URL="$(merge_val "$BASE_URL" "" "$PROFILE_FILE" "server" "base_url" "$DEFAULT_BASE_URL")"
FINAL_AGENT_URL="$(merge_val "$AGENT_URL" "" "$PROFILE_FILE" "agent_url" "agent_url" "$DEFAULT_AGENT_URL")"
FINAL_RUNTIME_DIR="$(merge_val "$RUNTIME_DIR" "" "$PROFILE_FILE" "server" "runtime_dir" "$DEFAULT_RUNTIME_DIR")"
FINAL_OUTPUT_DIR="$(merge_val "$OUTPUT_DIR" "" "$PROFILE_FILE" "bootstrap" "output_dir" "$DEFAULT_OUTPUT_DIR")"
FINAL_INITIAL_PASSWORD_ENV="${DEFAULT_INITIAL_PASSWORD_ENV}"
FINAL_ADMIN_PASSWORD_ENV="${DEFAULT_ADMIN_PASSWORD_ENV}"
FINAL_ALLOW_REAL_START="${ALLOW_REAL_START:-false}"
FINAL_ALLOW_CHAT_COMPLETION="${ALLOW_CHAT_COMPLETION:-false}"
FINAL_INCLUDE_SECRETS="${INCLUDE_SECRETS:-false}"
FINAL_INCLUDE_RUNTIME_STATE="${INCLUDE_RUNTIME_STATE:-false}"
FINAL_OUTPUT_PROFILE="${OUTPUT_PROFILE:-}"

# Profile auth fields
PROFILE_AUTH_USERNAME="$(yaml_get "$PROFILE_FILE" "auth" "username" 2>/dev/null)" || true
PROFILE_AUTH_USERNAME="${PROFILE_AUTH_USERNAME:-admin}"
PROFILE_INITIAL_PASSWORD_FILE="$(yaml_get "$PROFILE_FILE" "auth" "initial_password_file" 2>/dev/null)" || true
PROFILE_FINAL_PASSWORD_FILE="$(yaml_get "$PROFILE_FILE" "auth" "final_password_file" 2>/dev/null)" || true

# ── Setup output directory ──────────────────────────────────────────
mkdir -p "$FINAL_OUTPUT_DIR"

BOOTSTRAP_LOG="$FINAL_OUTPUT_DIR/bootstrap.log"
ERRORS_JSON="$FINAL_OUTPUT_DIR/errors.json"
EFFECTIVE_CONFIG_JSON="$FINAL_OUTPUT_DIR/effective-config.json"

# ── Sanitization helpers ────────────────────────────────────────────
# Patterns that MUST NOT appear in output files
SENSITIVE_PATTERNS=("password" "token" "csrf" "cookie" "secret" "key")

is_sensitive_key() {
  local key="$1"
  for pat in "${SENSITIVE_PATTERNS[@]}"; do
    if echo "${key,,}" | grep -q "$pat"; then return 0; fi
  done
  return 1
}

sanitize_for_json() {
  # Replace sensitive values with "[REDACTED]"
  local raw="$1"
  if [[ -z "$raw" ]]; then echo '""'; return; fi
  # If it looks like a hex string (auto-generated password), redact
  if echo "$raw" | grep -qEx '[0-9a-fA-F]{32,}'; then
    echo '"[REDACTED-HEX]"'
  elif [[ ${#raw} -gt 3 ]]; then
    echo '"[REDACTED]"'
  else
    echo '""'
  fi
}

# ── Logging ──────────────────────────────────────────────────────────
log_msg() {
  local level="$1" msg="$2"
  printf '[%s] [%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "$msg" >> "$BOOTSTRAP_LOG"
}

log_info()  { log_msg "INFO"  "$*"; }
log_warn()  { log_msg "WARN"  "$*"; echo "WARN: $*" >&2; }
log_error() { log_msg "ERROR" "$*"; echo "ERROR: $*" >&2; }

# ── Errors file ─────────────────────────────────────────────────────
ERRORS_LIST="[]"
add_error() {
  local code="$1" msg="$2"
  log_error "[$code] $msg"
  ERRORS_LIST=$(echo "$ERRORS_LIST" | python3 -c "
import sys,json
arr=json.load(sys.stdin)
arr.append({'code':'$code','message':'$msg','timestamp':'$TIMESTAMP'})
print(json.dumps(arr,indent=2))
" 2>/dev/null || echo "$ERRORS_LIST")
}

flush_errors() {
  echo "$ERRORS_LIST" > "$ERRORS_JSON"
}

# ── Effective config ────────────────────────────────────────────────
write_effective_config() {
  local initial_src="none"
  if [[ -n "${INITIAL_PASSWORD:-}" ]]; then initial_src="cli"; elif [[ -n "${!FINAL_INITIAL_PASSWORD_ENV:-}" ]]; then initial_src="env:${FINAL_INITIAL_PASSWORD_ENV}"; elif [[ -f "$FINAL_RUNTIME_DIR/runtime/initial-credentials.txt" ]]; then initial_src="runtime-file"; fi
  local final_src="none"
  if [[ -n "${ADMIN_PASSWORD:-}" ]]; then final_src="cli"; elif [[ -n "${!FINAL_ADMIN_PASSWORD_ENV:-}" ]]; then final_src="env:${FINAL_ADMIN_PASSWORD_ENV}"; fi

  cat > "$EFFECTIVE_CONFIG_JSON" << EFFECTIVE_EOF
{
  "profile_path": "$(realpath "$PROFILE_FILE" 2>/dev/null || echo "$PROFILE_FILE")",
  "mode": "$FINAL_MODE",
  "base_url": "$FINAL_BASE_URL",
  "agent_url": "$FINAL_AGENT_URL",
  "runtime_dir": "$FINAL_RUNTIME_DIR",
  "output_dir": "$FINAL_OUTPUT_DIR",
  "auth_username": "$PROFILE_AUTH_USERNAME",
  "initial_password_source": "$initial_src",
  "final_password_source": "$final_src",
  "allow_real_start": "$FINAL_ALLOW_REAL_START",
  "allow_chat_completion": "${FINAL_ALLOW_CHAT_COMPLETION:-false}",
  "include_runtime_state": "$FINAL_INCLUDE_RUNTIME_STATE",
  "include_secrets": "$FINAL_INCLUDE_SECRETS",
  "generated_at": "$TIMESTAMP",
  "bootstrap_version": "0.1.0"
}
EFFECTIVE_EOF
  log_info "effective-config written to $EFFECTIVE_CONFIG_JSON"
}

# ── Runtime-dir check ────────────────────────────────────────────────
check_runtime_dir() {
  local dir="$1"
  local issues=""
  if [[ ! -d "$dir" ]]; then
    issues="${issues}missing-dir "
    log_warn "runtime-dir does not exist: $dir"
  else
    if [[ ! -f "$dir/data/lightai.db" ]]; then
      issues="${issues}missing-db "
      log_warn "DB not found: $dir/data/lightai.db (clean DB expected on first run)"
    fi
    if [[ ! -d "$dir/logs" ]]; then
      issues="${issues}missing-logs "
      log_warn "logs directory not found: $dir/logs"
    fi
  fi
  if [[ -n "$issues" ]]; then
    add_error "RUNTIME_DIR_CHECK" "runtime-dir issues: $issues"
  else
    log_info "runtime-dir check OK: $dir"
  fi
}

# ── Auth helpers ─────────────────────────────────────────────────────

COOKIE_JAR="$FINAL_OUTPUT_DIR/.cookie-jar"
CSRF_FILE="$FINAL_OUTPUT_DIR/.csrf-token"
AUTH_JSON="$FINAL_OUTPUT_DIR/auth.json"

# Ensure cookie jar starts fresh
rm -f "$COOKIE_JAR" "$CSRF_FILE"
touch "$COOKIE_JAR" && chmod 0600 "$COOKIE_JAR"

CURL_OPTS=(-sS -o /dev/null -w '%{http_code}')

curl_server_get() {
  local path="$1" output_file="${2:-/dev/null}"
  curl -sS -o "$output_file" -w '%{http_code}' -X GET "$FINAL_BASE_URL$path" \
    -H "Origin: $FINAL_BASE_URL" -H "Content-Type: application/json" \
    -b "$COOKIE_JAR" -c "$COOKIE_JAR" 2>/dev/null || echo "000"
}

curl_server_post() {
  local path="$1" body="$2" output_file="${3:-/dev/null}"
  local csrf_header=""
  if [[ -f "$CSRF_FILE" && -s "$CSRF_FILE" ]]; then
    csrf_header="-H X-CSRF-Token: $(cat "$CSRF_FILE")"
  fi
  curl -sS -o "$output_file" -w '%{http_code}' -X POST "$FINAL_BASE_URL$path" \
    -H "Origin: $FINAL_BASE_URL" -H "Content-Type: application/json" \
    -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
    $csrf_header -d "$body" 2>/dev/null || echo "000"
}

read_credentials_file_password() {
  local path="$1"
  if [[ ! -f "$path" ]]; then echo ""; return; fi
  awk '/^Password:/ {print $2; exit}' "$path" 2>/dev/null || echo ""
}

find_credentials_file() {
  # Try multiple locations relative to runtime_dir
  local candidates=(
    "$FINAL_RUNTIME_DIR/runtime/initial-credentials.txt"
    "$FINAL_RUNTIME_DIR/initial-credentials.txt"
    "$FINAL_RUNTIME_DIR/data/initial-credentials.txt"
  )
  for c in "${candidates[@]}"; do
    if [[ -f "$c" ]]; then echo "$c"; return 0; fi
  done
  return 1
}

resolve_final_password() {
  # 1. --admin-password
  if [[ -n "${ADMIN_PASSWORD:-}" ]]; then
    echo "cli:--admin-password"
    return 0
  fi
  # 2. LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD
  if [[ -n "${!FINAL_ADMIN_PASSWORD_ENV:-}" ]]; then
    echo "env:${FINAL_ADMIN_PASSWORD_ENV}"
    return 0
  fi
  # 3. profile.auth.final_password_env
  local indirect_env
  indirect_env="$(yaml_get "$PROFILE_FILE" "auth" "final_password_env" 2>/dev/null)" || true
  if [[ -n "${indirect_env:-}" && -n "${!indirect_env:-}" ]]; then
    echo "env:${indirect_env}"
    return 0
  fi
  # 4. --admin-password-file
  if [[ -n "${ADMIN_PASSWORD_FILE:-}" && -f "$ADMIN_PASSWORD_FILE" ]]; then
    echo "file:${ADMIN_PASSWORD_FILE}"
    return 0
  fi
  # 5. profile.auth.final_password_file
  local pf_file
  pf_file="$(yaml_get "$PROFILE_FILE" "auth" "final_password_file" 2>/dev/null)" || true
  if [[ -n "${pf_file:-}" && -f "$pf_file" ]]; then
    echo "file:${pf_file}"
    return 0
  fi
  echo "none"
  return 1
}

get_final_password_value() {
  # Returns the actual password value from the resolved source
  if [[ -n "${ADMIN_PASSWORD:-}" ]]; then echo "$ADMIN_PASSWORD"; return 0; fi
  if [[ -n "${!FINAL_ADMIN_PASSWORD_ENV:-}" ]]; then echo "${!FINAL_ADMIN_PASSWORD_ENV}"; return 0; fi
  local indirect_env
  indirect_env="$(yaml_get "$PROFILE_FILE" "auth" "final_password_env" 2>/dev/null)" || true
  if [[ -n "${indirect_env:-}" && -n "${!indirect_env:-}" ]]; then echo "${!indirect_env}"; return 0; fi
  if [[ -n "${ADMIN_PASSWORD_FILE:-}" && -f "$ADMIN_PASSWORD_FILE" ]]; then head -1 "$ADMIN_PASSWORD_FILE"; return 0; fi
  local pf_file
  pf_file="$(yaml_get "$PROFILE_FILE" "auth" "final_password_file" 2>/dev/null)" || true
  if [[ -n "${pf_file:-}" && -f "$pf_file" ]]; then head -1 "$pf_file"; return 0; fi
  echo ""
  return 1
}

resolve_initial_password() {
  # 1. --initial-password
  if [[ -n "${INITIAL_PASSWORD:-}" ]]; then
    echo "cli:--initial-password"
    return 0
  fi
  # 2. LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD
  if [[ -n "${!FINAL_INITIAL_PASSWORD_ENV:-}" ]]; then
    echo "env:${FINAL_INITIAL_PASSWORD_ENV}"
    return 0
  fi
  # 3. profile.auth.initial_password_env
  local indirect_env
  indirect_env="$(yaml_get "$PROFILE_FILE" "auth" "initial_password_env" 2>/dev/null)" || true
  if [[ -n "${indirect_env:-}" && -n "${!indirect_env:-}" ]]; then
    echo "env:${indirect_env}"
    return 0
  fi
  # 4. --initial-password-file
  if [[ -n "${INITIAL_PASSWORD_FILE:-}" && -f "$INITIAL_PASSWORD_FILE" ]]; then
    echo "file:${INITIAL_PASSWORD_FILE}"
    return 0
  fi
  # 5. profile.auth.initial_password_file
  local pf_file
  pf_file="$(yaml_get "$PROFILE_FILE" "auth" "initial_password_file" 2>/dev/null)" || true
  if [[ -n "${pf_file:-}" && -f "$pf_file" ]]; then
    echo "file:${pf_file}"
    return 0
  fi
  # 6. runtime credentials file
  local cred_file
  if cred_file=$(find_credentials_file 2>/dev/null) && [[ -n "$cred_file" ]]; then
    echo "runtime:${cred_file}"
    return 0
  fi
  # 7. profile.auth.initial_password
  local pf_pw
  pf_pw="$(yaml_get "$PROFILE_FILE" "auth" "initial_password" 2>/dev/null)" || true
  if [[ -n "${pf_pw:-}" && "${pf_pw:-}" != '""' ]]; then
    echo "profile:initial_password"
    return 0
  fi
  echo "none"
  return 1
}

get_initial_password_value() {
  if [[ -n "${INITIAL_PASSWORD:-}" ]]; then echo "$INITIAL_PASSWORD"; return 0; fi
  if [[ -n "${!FINAL_INITIAL_PASSWORD_ENV:-}" ]]; then echo "${!FINAL_INITIAL_PASSWORD_ENV}"; return 0; fi
  local indirect_env
  indirect_env="$(yaml_get "$PROFILE_FILE" "auth" "initial_password_env" 2>/dev/null)" || true
  if [[ -n "${indirect_env:-}" && -n "${!indirect_env:-}" ]]; then echo "${!indirect_env}"; return 0; fi
  if [[ -n "${INITIAL_PASSWORD_FILE:-}" && -f "$INITIAL_PASSWORD_FILE" ]]; then head -1 "$INITIAL_PASSWORD_FILE"; return 0; fi
  local pf_file
  pf_file="$(yaml_get "$PROFILE_FILE" "auth" "initial_password_file" 2>/dev/null)" || true
  if [[ -n "${pf_file:-}" && -f "$pf_file" ]]; then head -1 "$pf_file"; return 0; fi
  local cred_file
  if cred_file=$(find_credentials_file 2>/dev/null) && [[ -n "$cred_file" ]]; then
    read_credentials_file_password "$cred_file"
    return 0
  fi
  local pf_pw
  pf_pw="$(yaml_get "$PROFILE_FILE" "auth" "initial_password" 2>/dev/null)" || true
  if [[ -n "${pf_pw:-}" && "${pf_pw:-}" != '""' ]]; then echo "$pf_pw"; return 0; fi
  echo ""
  return 1
}

write_auth_json() {
  local status="$1" method="$2" server_ok="$3" agent_ok="$4"
  local csrf_present="false" cookie_present="false"
  [[ -f "$CSRF_FILE" && -s "$CSRF_FILE" ]] && csrf_present="true"
  [[ -f "$COOKIE_JAR" && -s "$COOKIE_JAR" ]] && cookie_present="true"
  cat > "$AUTH_JSON" << AUTH_EOF
{
  "login_status": "$status",
  "username": "$PROFILE_AUTH_USERNAME",
  "auth_method": "$method",
  "server_reachable": $server_ok,
  "agent_reachable": $agent_ok,
  "token_present": false,
  "cookie_present": $cookie_present,
  "csrf_present": $csrf_present,
  "password_changed": $([[ "$method" == *"changed"* ]] && echo "true" || echo "false"),
  "must_change_password_initial": $([[ "$status" == "PASS" && "$method" == "initial_password_without_change_required" ]] && echo "false" || echo "true"),
  "must_change_password_final": false,
  "initial_password_source": "$INITIAL_PW_SOURCE",
  "final_password_source": "$FINAL_PW_SOURCE",
  "runtime_initial_credentials_file": "${RUNTIME_CRED_FILE:-}",
  "timestamp": "$TIMESTAMP"
}
AUTH_EOF
  log_info "auth.json written to $AUTH_JSON"
}

# ── Mode dispatch ────────────────────────────────────────────────────

run_auth_only() {
  log_info "===== auth-only mode ====="

  local server_ok="false" agent_ok="false"
  local FINAL_PW_SOURCE="none" INITIAL_PW_SOURCE="none" RUNTIME_CRED_FILE=""

  # Step 1: Check server health
  log_info "checking server: $FINAL_BASE_URL"
  local server_status
  server_status=$(curl_server_get "/healthz" 2>/dev/null || echo "000")
  if [[ "$server_status" == "200" ]]; then
    server_ok="true"
    log_info "server reachable: $FINAL_BASE_URL (HTTP $server_status)"
  else
    log_error "server unreachable: $FINAL_BASE_URL (HTTP $server_status)"
    write_auth_json "FAIL" "none" "false" "false"
    add_error "SERVER_UNREACHABLE" "server $FINAL_BASE_URL returned HTTP $server_status"
    return 1
  fi

  # Step 2: Check agent health
  log_info "checking agent: $FINAL_AGENT_URL"
  local agent_status
  agent_status=$(curl -sS -o /dev/null -w '%{http_code}' "$FINAL_AGENT_URL/healthz" 2>/dev/null || echo "000")
  if [[ "$agent_status" == "200" ]]; then
    agent_ok="true"
    log_info "agent reachable: $FINAL_AGENT_URL (HTTP $agent_status)"
  else
    log_warn "agent unreachable: $FINAL_AGENT_URL (HTTP $agent_status)"
    # Non-fatal: agent may not be running, continue with auth
  fi

  # Step 3: Runtime-dir check already done in main

  # Step 4 & 5: Resolve passwords
  FINAL_PW_SOURCE=$(resolve_final_password) || true
  log_info "final password source: $FINAL_PW_SOURCE"

  if [[ "$FINAL_PW_SOURCE" == "none" ]]; then
    log_info "final password not set, will try initial password"
  fi

  # Step 6: Try login with final/admin password first
  local login_status="" login_resp="" login_body=""
  local auth_method="none"
  INITIAL_PW_SOURCE="none"
  RUNTIME_CRED_FILE=""

  if [[ "$FINAL_PW_SOURCE" != "none" ]]; then
    local final_pw
    final_pw=$(get_final_password_value)
    login_body="{\"username\":\"$PROFILE_AUTH_USERNAME\",\"password\":\"$final_pw\"}"
    local resp_file="$FINAL_OUTPUT_DIR/responses/login-final.json"
    mkdir -p "$FINAL_OUTPUT_DIR/responses"
    login_status=$(curl_server_post "/api/v1/auth/login" "$login_body" "$resp_file")
    log_info "login attempt with final password: HTTP $login_status"

    if [[ "$login_status" == "200" ]]; then
      local must_change
      must_change=$(python3 -c "import json; d=json.load(open('$resp_file')); print(d.get('must_change_password','false'))" 2>/dev/null || echo "true")
      if [[ "$must_change" == "false" ]]; then
        auth_method="final_password"
        log_info "login succeeded with final password (no change required)"
      else
        # Final password works but still requires change — edge case
        log_warn "final password login succeeded but must_change_password=true"
        auth_method="final_password"
      fi
    else
      log_info "final password login failed (HTTP $login_status), falling back to initial password"
    fi
  fi

  # Step 7: Fall back to initial password
  if [[ "$auth_method" == "none" ]]; then
    INITIAL_PW_SOURCE=$(resolve_initial_password) || true
    log_info "initial password source: $INITIAL_PW_SOURCE"

    if [[ "$INITIAL_PW_SOURCE" == "none" ]]; then
      log_error "no initial password source available"
      write_auth_json "FAIL" "none" "$server_ok" "$agent_ok"
      add_error "AUTH_NO_PASSWORD" "no initial or final password available (set LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD or LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD)"
      return 1
    fi

    # Record credentials file path if used
    if [[ "$INITIAL_PW_SOURCE" == runtime:* ]]; then
      RUNTIME_CRED_FILE="${INITIAL_PW_SOURCE#runtime:}"
    fi

    local init_pw
    init_pw=$(get_initial_password_value)
    login_body="{\"username\":\"$PROFILE_AUTH_USERNAME\",\"password\":\"$init_pw\"}"
    local resp_file="$FINAL_OUTPUT_DIR/responses/login-initial.json"
    mkdir -p "$FINAL_OUTPUT_DIR/responses"
    login_status=$(curl_server_post "/api/v1/auth/login" "$login_body" "$resp_file")
    log_info "login attempt with initial password: HTTP $login_status"

    if [[ "$login_status" != "200" ]]; then
      log_error "initial password login failed (HTTP $login_status)"
      write_auth_json "FAIL" "none" "$server_ok" "$agent_ok"
      add_error "AUTH_LOGIN_FAILED" "login failed with both final and initial password (HTTP $login_status)"
      return 1
    fi

    # Extract CSRF token from successful login
    local csrf_val
    csrf_val=$(python3 -c "import json; d=json.load(open('$resp_file')); print(d.get('csrf_token',''))" 2>/dev/null || echo "")
    if [[ -n "$csrf_val" ]]; then
      echo "$csrf_val" > "$CSRF_FILE"
      chmod 0600 "$CSRF_FILE"
      log_info "CSRF token saved"
    else
      log_warn "login response missing csrf_token"
    fi

    local must_change
    must_change=$(python3 -c "import json; d=json.load(open('$resp_file')); print(d.get('must_change_password','false'))" 2>/dev/null || echo "true")

    if [[ "$must_change" == "true" ]]; then
      # Step 8: Change password required
      log_info "must_change_password=true, attempting password change"

      if [[ "$FINAL_PW_SOURCE" == "none" ]]; then
        log_error "must_change_password=true but no final/admin password set — cannot change"
        write_auth_json "FAIL" "initial_password" "$server_ok" "$agent_ok"
        add_error "FAIL_MISSING_FINAL_PASSWORD" "system requires password change but LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD is not set"
        return 1
      fi

      local final_pw
      final_pw=$(get_final_password_value)
      local change_body="{\"current_password\":\"$init_pw\",\"new_password\":\"$final_pw\"}"
      local change_resp="$FINAL_OUTPUT_DIR/responses/change-password.json"
      local change_status
      change_status=$(curl_server_post "/api/v1/auth/change-password" "$change_body" "$change_resp")
      log_info "change-password: HTTP $change_status"

      if [[ "$change_status" == "200" ]]; then
        log_info "password changed successfully"
        auth_method="initial_password_changed"
      else
        log_error "change-password failed (HTTP $change_status)"
        write_auth_json "FAIL" "initial_password" "$server_ok" "$agent_ok"
        add_error "AUTH_CHANGE_PASSWORD_FAILED" "change-password returned HTTP $change_status"
        return 1
      fi

      # Step 9: Re-login with final password
      rm -f "$COOKIE_JAR" "$CSRF_FILE"
      touch "$COOKIE_JAR" && chmod 0600 "$COOKIE_JAR"
      login_body="{\"username\":\"$PROFILE_AUTH_USERNAME\",\"password\":\"$final_pw\"}"
      local relogin_resp="$FINAL_OUTPUT_DIR/responses/login-after-change.json"
      login_status=$(curl_server_post "/api/v1/auth/login" "$login_body" "$relogin_resp")
      log_info "re-login with final password: HTTP $login_status"

      if [[ "$login_status" == "200" ]]; then
        local csrf_val2
        csrf_val2=$(python3 -c "import json; d=json.load(open('$relogin_resp')); print(d.get('csrf_token',''))" 2>/dev/null || echo "")
        if [[ -n "$csrf_val2" ]]; then
          echo "$csrf_val2" > "$CSRF_FILE"
          chmod 0600 "$CSRF_FILE"
        fi
        auth_method="initial_password_changed"
        log_info "re-login with final password succeeded"
      else
        log_error "re-login after password change failed (HTTP $login_status)"
        write_auth_json "FAIL" "initial_password_changed" "$server_ok" "$agent_ok"
        add_error "AUTH_RELOGIN_FAILED" "re-login after password change returned HTTP $login_status"
        return 1
      fi
    else
      # No change required
      auth_method="initial_password_without_change_required"
      log_info "login succeeded with initial password (no change required)"
    fi
  fi

  # Step 11: Write auth.json
  log_info "auth completed: method=$auth_method"
  write_auth_json "PASS" "$auth_method" "$server_ok" "$agent_ok"
  log_info "cookie jar: $COOKIE_JAR, csrf: $CSRF_FILE"
  return 0
}

run_catalog_only() {
  log_info "mode: catalog-only"
  add_error "NOT_IMPLEMENTED" "catalog-only mode not yet implemented (Batch 4)"
}

run_models_only() {
  log_info "mode: models-only"
  add_error "NOT_IMPLEMENTED" "models-only mode not yet implemented (Batch 4)"
}

run_runtimes_only() {
  log_info "mode: runtimes-only"
  add_error "NOT_IMPLEMENTED" "runtimes-only mode not yet implemented (Batch 5)"
}

run_dry_run() {
  log_info "mode: dry-run"
  add_error "NOT_IMPLEMENTED" "dry-run mode not yet implemented (Batch 6)"
}

run_full() {
  log_info "mode: full"
  add_error "NOT_IMPLEMENTED" "full mode not yet implemented (Batch 7)"
}

run_export() {
  log_info "mode: export"
  add_error "NOT_IMPLEMENTED" "export mode not yet implemented (Batch 8)"
}

# ── Main ─────────────────────────────────────────────────────────────
log_info "LightAI Bootstrap starting"
log_info "  profile: $PROFILE_FILE"
log_info "  mode:    $FINAL_MODE"
log_info "  base:    $FINAL_BASE_URL"
log_info "  agent:   $FINAL_AGENT_URL"
log_info "  runtime: $FINAL_RUNTIME_DIR"
log_info "  output:  $FINAL_OUTPUT_DIR"

# Validate profile exists
if [[ ! -f "$PROFILE_FILE" ]]; then
  log_error "profile not found: $PROFILE_FILE"
  add_error "PROFILE_MISSING" "bootstrap profile not found: $PROFILE_FILE"
  flush_errors
  exit 1
fi

# Check runtime-dir
check_runtime_dir "$FINAL_RUNTIME_DIR"

# Write config
write_effective_config

# Dispatch mode
case "$FINAL_MODE" in
  auth-only)     run_auth_only ;;
  catalog-only)  run_catalog_only ;;
  models-only)   run_models_only ;;
  runtimes-only) run_runtimes_only ;;
  dry-run)       run_dry_run ;;
  full)          run_full ;;
  export)        run_export ;;
  *)
    log_error "unknown mode: $FINAL_MODE"
    add_error "UNKNOWN_MODE" "unknown bootstrap mode: $FINAL_MODE (valid: auth-only, catalog-only, models-only, runtimes-only, dry-run, full, export)"
    flush_errors
    exit 1
    ;;
esac

# Always flush errors
flush_errors

# Check for errors in output
ERROR_COUNT=$(echo "$ERRORS_LIST" | python3 -c "import sys,json;print(len(json.load(sys.stdin)))" 2>/dev/null || echo 0)
if [[ "$ERROR_COUNT" -gt 0 ]]; then
  log_warn "completed with $ERROR_COUNT error(s) — see $ERRORS_JSON"
else
  log_info "completed successfully"
fi
log_info "all output in: $FINAL_OUTPUT_DIR"

# Ensure no sensitive data leaked
sanitize_check() {
  for f in "$EFFECTIVE_CONFIG_JSON" "$BOOTSTRAP_LOG"; do
    if [[ -f "$f" ]]; then
      # Only flag if a value (after colon+space) looks like a real password/token,
      # not just a key name containing 'password' or 'secret'.
      if grep -qiE '"[^"]*":\s*"[a-zA-Z0-9+/=_-]{8,}"' "$f" 2>/dev/null; then
        # Check if any value looks like a plausible secret (long random string)
        local suspects
        suspects=$(grep -oiE '"[^"]*(password|secret|token|key)[^"]*":\s*"[^"]{4,}"' "$f" 2>/dev/null || true)
        if [[ -n "$suspects" ]]; then
          log_warn "potential sensitive data found in $f — review manually (keys matching password/secret/token/key with non-empty values)"
        fi
      fi
    fi
  done
}
# Note: full password/token/csrf detection requires value-level inspection.
# This heuristic catches the most common cases without false-flagging key names.
sanitize_check

echo ""
echo "=== Bootstrap complete ==="
echo "  Mode:   $FINAL_MODE"
echo "  Output: $FINAL_OUTPUT_DIR"
if [[ "$ERROR_COUNT" -gt 0 ]]; then
  echo "  Errors: $ERROR_COUNT (see $ERRORS_JSON)"
fi
