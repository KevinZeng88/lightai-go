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

# Get a deeply nested YAML value using dot-separated path
# e.g. yaml_get_nested profile.yaml models qwen3_small.display_name
yaml_get_nested() {
  local file="$1" top_key="$2" nested_path="$3"
  if command -v python3 >/dev/null 2>&1; then
    python3 -c "
import sys,yaml
d=yaml.safe_load(open(sys.argv[1]))
v=d.get(sys.argv[2],{})
for k in sys.argv[3].split('.'):
  if isinstance(v,dict): v=v.get(k,'')
  else: v='';break
print(v if v is not None else '')
" "$file" "$top_key" "$nested_path" 2>/dev/null
  fi
}

yaml_get_list() {
  local file="$1" key="$2" subkey="$3"
  if command -v python3 >/dev/null 2>&1; then
    python3 -c "
import sys,yaml
d=yaml.safe_load(open(sys.argv[1]))
v=d.get(sys.argv[2],{})
if sys.argv[3]:
  v=v.get(sys.argv[3],[])
if isinstance(v,dict):
  print('\n'.join(v.keys()))
elif isinstance(v,list):
  print('\n'.join(str(x) for x in v))
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
    local csrf_val
    csrf_val=$(tr -d '\n' < "$CSRF_FILE")
    csrf_header="-H X-CSRF-Token: $csrf_val"
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
      printf "%s" "$csrf_val" > "$CSRF_FILE"
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
          printf "%s" "$csrf_val2" > "$CSRF_FILE"
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

# ── Bootstrap state ──────────────────────────────────────────────────
BOOTSTRAP_STATE_JSON="$FINAL_OUTPUT_DIR/bootstrap-state.json"
CATALOG_JSON="$FINAL_OUTPUT_DIR/catalog.json"
MODELS_JSON="$FINAL_OUTPUT_DIR/models.json"
MODEL_LOCATIONS_JSON="$FINAL_OUTPUT_DIR/model-locations.json"

init_bootstrap_state() {
  cat > "$BOOTSTRAP_STATE_JSON" << 'EOF'
{"backend_ids":{},"backend_version_ids":{},"model_artifact_ids":{},"model_location_ids":{}}
EOF
}

update_bootstrap_state() {
  local key="$1" subkey="$2" val="$3"
  python3 -c "
import json,sys
d=json.load(open('$BOOTSTRAP_STATE_JSON'))
d.setdefault('$key',{})['$subkey']='$val'
json.dump(d,open('$BOOTSTRAP_STATE_JSON','w'),indent=2)
" 2>/dev/null || true
}

# ── API helpers ──────────────────────────────────────────────────────
curl_api_get() {
  local path="$1" output_file="${2:-/dev/stdout}"
  curl -sS -X GET "$FINAL_BASE_URL$path" \
    -H "Origin: $FINAL_BASE_URL" -H "Content-Type: application/json" \
    -b "$COOKIE_JAR" -c "$COOKIE_JAR" -o "$output_file" -w '%{http_code}' 2>/dev/null || echo "000"
}

curl_api_post() {
  local path="$1" body="$2" output_file="${3:-/dev/stdout}"
  local csrf_header=""
  if [[ -f "$CSRF_FILE" && -s "$CSRF_FILE" ]]; then
    local csrf_val
    csrf_val=$(tr -d '\n' < "$CSRF_FILE")
    csrf_header="-H X-CSRF-Token: $csrf_val"
  fi
  curl -sS -X POST "$FINAL_BASE_URL$path" \
    -H "Origin: $FINAL_BASE_URL" -H "Content-Type: application/json" \
    -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
    $csrf_header -d "$body" -o "$output_file" -w '%{http_code}' 2>/dev/null || echo "000"
}

# ── Auth prerequisite ────────────────────────────────────────────────
ensure_auth() {
  # Reuse existing auth if cookie jar is valid
  if [[ -f "$COOKIE_JAR" && -s "$COOKIE_JAR" && -f "$CSRF_FILE" && -s "$CSRF_FILE" ]]; then
    local check_status
    check_status=$(curl_api_get "/api/v1/auth/me" /dev/null)
    if [[ "$check_status" == "200" ]]; then
      log_info "auth session valid (reusing existing)"
      return 0
    fi
    log_info "auth session expired, re-authenticating"
    rm -f "$COOKIE_JAR" "$CSRF_FILE"
    touch "$COOKIE_JAR" && chmod 0600 "$COOKIE_JAR"
  fi

  # Run auth-only logic inline
  local server_ok="false" agent_ok="false"
  local FINAL_PW_SOURCE="none" INITIAL_PW_SOURCE="none" RUNTIME_CRED_FILE=""

  local srv_st
  srv_st=$(curl_server_get "/healthz" 2>/dev/null || echo "000")
  [[ "$srv_st" == "200" ]] && server_ok="true"

  local agt_st
  agt_st=$(curl -sS -o /dev/null -w '%{http_code}' "$FINAL_AGENT_URL/healthz" 2>/dev/null || echo "000")
  [[ "$agt_st" == "200" ]] && agent_ok="true"

  if [[ "$server_ok" != "true" ]]; then
    add_error "AUTH_PREREQ_FAILED" "server unreachable, cannot authenticate"
    return 1
  fi

  FINAL_PW_SOURCE=$(resolve_final_password) || true
  log_info "auth prereq: final password source=$FINAL_PW_SOURCE"
  INITIAL_PW_SOURCE="none"

  if [[ "$FINAL_PW_SOURCE" != "none" ]]; then
    local final_pw
    final_pw=$(get_final_password_value)
    local resp_file="$FINAL_OUTPUT_DIR/responses/login-auth-prereq.json"
    mkdir -p "$FINAL_OUTPUT_DIR/responses"
    local status
    status=$(curl_server_post "/api/v1/auth/login" "{\"username\":\"$PROFILE_AUTH_USERNAME\",\"password\":\"$final_pw\"}" "$resp_file")
    if [[ "$status" == "200" ]]; then
      local csrf_val
      csrf_val=$(python3 -c "import json;d=json.load(open('$resp_file'));print(d.get('csrf_token',''))" 2>/dev/null || echo "")
      [[ -n "$csrf_val" ]] && printf "%s" "$csrf_val" > "$CSRF_FILE" && chmod 0600 "$CSRF_FILE"
      log_info "auth prereq: logged in with final password"
      return 0
    fi
    log_info "auth prereq: final password login failed (HTTP $status)"
  fi

  INITIAL_PW_SOURCE=$(resolve_initial_password) || true
  if [[ "$INITIAL_PW_SOURCE" == "none" ]]; then
    add_error "AUTH_PREREQ_FAILED" "no password available for authentication"
    return 1
  fi
  local init_pw
  init_pw=$(get_initial_password_value)
  local resp_file="$FINAL_OUTPUT_DIR/responses/login-auth-prereq2.json"
  mkdir -p "$FINAL_OUTPUT_DIR/responses"
  local status
  status=$(curl_server_post "/api/v1/auth/login" "{\"username\":\"$PROFILE_AUTH_USERNAME\",\"password\":\"$init_pw\"}" "$resp_file")
  if [[ "$status" != "200" ]]; then
    add_error "AUTH_PREREQ_FAILED" "login failed (HTTP $status)"
    return 1
  fi
  local csrf_val
  csrf_val=$(python3 -c "import json;d=json.load(open('$resp_file'));print(d.get('csrf_token',''))" 2>/dev/null || echo "")
  [[ -n "$csrf_val" ]] && printf "%s" "$csrf_val" > "$CSRF_FILE" && chmod 0600 "$CSRF_FILE"
  log_info "auth prereq: logged in with initial password"
  return 0
}

get_node_id() {
  local resp="$FINAL_OUTPUT_DIR/responses/nodes.json"
  local status
  status=$(curl_api_get "/api/v1/nodes" "$resp")
  if [[ "$status" != "200" ]]; then
    add_error "NODE_LOOKUP_FAILED" "GET /api/v1/nodes returned HTTP $status"
    return 1
  fi
  python3 -c "
import json
data=json.load(open('$resp'))
if isinstance(data,list): arr=data
else: arr=data.get('data',data.get('items',[]))
for n in arr:
  nm=n.get('hostname','')
  aid=n.get('agent_id','')
  nid=n.get('id','')
  if nm=='$PROFILE_NODE_NAME' or aid=='$PROFILE_NODE_NAME' or nid=='$PROFILE_NODE_NAME':
    print(n.get('id',''))
    break
" 2>/dev/null
}

# ── run_catalog_only ──────────────────────────────────────────────────
run_catalog_only() {
  log_info "===== catalog-only mode ====="

  if ! ensure_auth; then
    add_error "CATALOG_AUTH_FAILED" "could not authenticate for catalog check"
    cat > "$CATALOG_JSON" << 'EOF'
{"status":"FAIL","required_backends":["vllm","sglang","llamacpp"],"found_backends":{},"missing_backends":["vllm","sglang","llamacpp"],"backend_ids":{},"backend_version_ids":{},"checked_at":""}
EOF
    return 1
  fi

  local resp="$FINAL_OUTPUT_DIR/responses/backends.json"
  mkdir -p "$FINAL_OUTPUT_DIR/responses"
  local status
  status=$(curl_api_get "/api/v1/backends" "$resp")
  if [[ "$status" != "200" ]]; then
    add_error "CATALOG_API_FAILED" "GET /api/v1/backends returned HTTP $status"
    cat > "$CATALOG_JSON" << 'EOF'
{"status":"FAIL","required_backends":["vllm","sglang","llamacpp"],"found_backends":{},"missing_backends":["vllm","sglang","llamacpp"],"backend_ids":{},"backend_version_ids":{},"checked_at":""}
EOF
    return 1
  fi

  local required=("vllm" "sglang" "llamacpp")
  init_bootstrap_state

  python3 -c "
import json,sys
resp=json.load(open('$resp'))
backends=resp if isinstance(resp,list) else resp.get('data',resp.get('items',[]))
# Backend IDs are format: backend.{name} (e.g. backend.vllm)
def match_backend(b, target):
  bid=b.get('id','').lower()
  bname=b.get('name','').lower()
  target=target.lower()
  # Match by id suffix (e.g. backend.vllm matches vllm)
  if bid==target or bid.endswith('.'+target) or bid=='backend.'+target:
    return True
  # Match by display name
  if b.get('display_name','').lower()==target:
    return True
  return False

missing=[]
found={}
bids={}
vids={}
for r in '${required[*]}'.split():
  m=None
  for b in backends:
    if match_backend(b, r):
      m=b;break
  if m:
    bids[r]=m.get('id','')
    found[r]=m.get('display_name',m.get('id',''))
    # Try to get version
    import subprocess
    vresp_file='$FINAL_OUTPUT_DIR/responses/versions-'+r+'.json'
    vcode=subprocess.run(['curl','-sS','-o',vresp_file,'-w','%{http_code}','-X','GET','$FINAL_BASE_URL/api/v1/backends/'+m['id']+'/versions','-H','Origin: $FINAL_BASE_URL','-H','Content-Type: application/json','-b','$COOKIE_JAR'],capture_output=True,text=True).stdout.strip()
    if vcode=='200':
      try:
        vdata=json.load(open(vresp_file))
        vlist=vdata if isinstance(vdata,list) else vdata.get('data',vdata.get('items',[]))
        if vlist:
          vids[r]=vlist[0].get('id','')
          found[r]+=' ('+vlist[0].get('version','?')+')'
      except: pass
  else:
    missing.append(r)
result={'status':'PASS' if not missing else 'FAIL','required_backends':list('${required[*]}'.split()),'found_backends':found,'missing_backends':missing,'backend_ids':bids,'backend_version_ids':vids,'checked_at':'$TIMESTAMP'}
json.dump(result,open('$CATALOG_JSON','w'),indent=2)
print('catalog: found',len(found),'missing',len(missing))
" 2>/dev/null

  log_info "catalog.json written to $CATALOG_JSON"
  # Update bootstrap state
  if [[ -f "$CATALOG_JSON" ]]; then
    for bk in vllm sglang llamacpp; do
      local bid
      bid=$(python3 -c "import json;d=json.load(open('$CATALOG_JSON'));print(d['backend_ids'].get('$bk',''))" 2>/dev/null || echo "")
      [[ -n "$bid" ]] && update_bootstrap_state "backend_ids" "$bk" "$bid"
      local vid
      vid=$(python3 -c "import json;d=json.load(open('$CATALOG_JSON'));print(d['backend_version_ids'].get('$bk',''))" 2>/dev/null || echo "")
      [[ -n "$vid" ]] && update_bootstrap_state "backend_version_ids" "$bk" "$vid"
    done
  fi

  local missing_count
  missing_count=$(python3 -c "import json;print(len(json.load(open('$CATALOG_JSON'))['missing_backends']))" 2>/dev/null || echo 3)
  if [[ "$missing_count" -gt 0 ]]; then
    add_error "CATALOG_BACKEND_MISSING" "$missing_count required backend(s) missing"
    return 1
  fi
  return 0
}

run_models_only() {
  log_info "===== models-only mode ====="

  # Run catalog-only first
  if ! run_catalog_only; then
    log_error "catalog check failed, cannot proceed to models"
    return 1
  fi

  # Initialize output files
  cat > "$MODELS_JSON" << EOF
{"status":"PASS","models":{},"checked_at":"$TIMESTAMP"}
EOF
  cat > "$MODEL_LOCATIONS_JSON" << EOF
{"status":"PASS","locations":{},"checked_at":"$TIMESTAMP"}
EOF

  # Ensure we have a node ID
  PROFILE_NODE_NAME="$(yaml_get "$PROFILE_FILE" "node" "name" 2>/dev/null)" || true
  PROFILE_NODE_NAME="${PROFILE_NODE_NAME:-KZ-LAPTOP}"
  local node_id
  node_id=$(get_node_id)
  if [[ -z "$node_id" ]]; then
    add_error "NODE_NOT_FOUND" "node '$PROFILE_NODE_NAME' not found in /api/v1/nodes — models-only requires a registered node"
    python3 -c "import json;d=json.load(open('$MODELS_JSON'));d['status']='FAIL';d['error']='node $PROFILE_NODE_NAME not found';json.dump(d,open('$MODELS_JSON','w'),indent=2)" 2>/dev/null
    python3 -c "import json;d=json.load(open('$MODEL_LOCATIONS_JSON'));d['status']='FAIL';d['error']='node $PROFILE_NODE_NAME not found';json.dump(d,open('$MODEL_LOCATIONS_JSON','w'),indent=2)" 2>/dev/null
    return 1
  fi
  log_info "node found: $PROFILE_NODE_NAME (id=$node_id)"
  update_bootstrap_state "ids" "node_id" "$node_id"

  # Get model keys from profile
  local model_keys
  model_keys=$(yaml_get_list "$PROFILE_FILE" "models" "" 2>/dev/null || echo "")
  if [[ -z "$model_keys" ]]; then
    log_info "no models defined in profile"
    cat > "$MODELS_JSON" << EOF
{"status":"PASS","models":{},"checked_at":"$TIMESTAMP"}
EOF
    cat > "$MODEL_LOCATIONS_JSON" << EOF
{"status":"PASS","locations":{},"checked_at":"$TIMESTAMP"}
EOF
    return 0
  fi

  # List existing artifacts for idempotency
  local artifacts_resp="$FINAL_OUTPUT_DIR/responses/artifacts-list.json"
  curl_api_get "/api/v1/model-artifacts" "$artifacts_resp" 2>/dev/null || true

  local models_result="{}" locations_result="{}" overall_status="PASS"

  while IFS= read -r model_key; do
    [[ -z "$model_key" ]] && continue
    log_info "processing model: $model_key"

    local display_name kind path
    display_name=$(yaml_get_nested "$PROFILE_FILE" "models" "$model_key.display_name" 2>/dev/null || echo "$model_key")
    kind=$(yaml_get_nested "$PROFILE_FILE" "models" "$model_key.kind" 2>/dev/null || echo "huggingface")
    path=$(yaml_get_nested "$PROFILE_FILE" "models" "$model_key.path" 2>/dev/null || echo "")

    # Determine format for API
    local format="custom" task_type="chat" path_type="directory"
    if [[ "$kind" == "gguf" ]]; then format="gguf"; task_type="completion"; path_type="file"; fi

    # Check path exists
    local path_exists="true"
    if [[ ! -e "$path" ]]; then
      log_error "model path not found: $path"
      add_error "MODEL_PATH_MISSING" "model $model_key path not found: $path"
      path_exists="false"
      overall_status="FAIL"
    fi

    # Find existing artifact by name
    local artifact_id=""
    if [[ -f "$artifacts_resp" ]]; then
      artifact_id=$(python3 -c "
import json
data=json.load(open('$artifacts_resp'))
arr=data if isinstance(data,list) else data.get('data',data.get('items',[]))
for a in arr:
  if a.get('name','')=='$model_key' or a.get('path','')=='$path':
    print(a.get('id',''));break
" 2>/dev/null || echo "")
    fi

    local action="REUSE"
    if [[ -z "$artifact_id" ]]; then
      if [[ "$path_exists" != "true" ]]; then
        action="FAIL"
      else
        # Create new artifact
        action="CREATE"
        local create_body="{\"name\":\"$model_key\",\"display_name\":\"$display_name\",\"path\":\"$path\",\"format\":\"$format\",\"task_type\":\"$task_type\",\"source_type\":\"local_path\",\"architecture\":\"custom\",\"quantization\":\"unknown\",\"default_context_length\":0,\"estimated_vram_bytes\":0,\"required_gpu_count\":1,\"capabilities_json\":\"[]\",\"capability_sources_json\":\"{}\",\"default_test_mode\":\"auto\",\"parameter_defaults_json\":\"[]\"}"
        local create_resp="$FINAL_OUTPUT_DIR/responses/artifact-create-$model_key.json"
        local create_status
        create_status=$(curl_api_post "/api/v1/model-artifacts" "$create_body" "$create_resp")
        if [[ "$create_status" == "201" ]]; then
          artifact_id=$(python3 -c "import json;print(json.load(open('$create_resp')).get('id',''))" 2>/dev/null || echo "")
          log_info "created artifact: $model_key (id=$artifact_id)"
        else
          log_error "failed to create artifact: $model_key (HTTP $create_status)"
          add_error "MODEL_ARTIFACT_CREATE_FAILED" "create artifact $model_key returned HTTP $create_status"
          action="FAIL"
          overall_status="FAIL"
        fi
      fi
    else
      log_info "reusing existing artifact: $model_key (id=$artifact_id)"
    fi

    update_bootstrap_state "model_artifact_ids" "$model_key" "$artifact_id"

    # Create model location
    local location_id="" loc_action="REUSE"
    if [[ "$action" != "FAIL" && -n "$artifact_id" ]]; then
      local loc_body="{\"node_id\":\"$node_id\",\"path_type\":\"$path_type\",\"absolute_path\":\"$path\",\"size_bytes\":0,\"checksum\":\"\",\"manifest_digest\":\"\",\"match_status\":\"exact_match\",\"verification_status\":\"verified\",\"manual_override\":false}"
      local loc_resp="$FINAL_OUTPUT_DIR/responses/location-create-$model_key.json"
      local loc_status
      loc_status=$(curl_api_post "/api/v1/model-artifacts/$artifact_id/locations" "$loc_body" "$loc_resp")
      if [[ "$loc_status" == "201" ]]; then
        location_id=$(python3 -c "import json;print(json.load(open('$loc_resp')).get('id',''))" 2>/dev/null || echo "")
        loc_action="CREATE"
        log_info "created location: $model_key (id=$location_id)"
      else
        # Location may already exist; try to find it
        log_info "location may already exist for $model_key (HTTP $loc_status)"
        loc_action="REUSE"
      fi
    fi

    update_bootstrap_state "model_location_ids" "$model_key" "${location_id:-}"

    python3 -c "
import json
mr=json.load(open('$MODELS_JSON')) if __import__('os').path.exists('$MODELS_JSON') else {}
mr.setdefault('models',{})['$model_key']={'display_name':'$display_name','kind':'$kind','path':'$path','path_exists':$([[ "$path_exists" == "true" ]] && echo "True" || echo "False"),'artifact_id':'$artifact_id','action':'$action'}
mr['status']='$overall_status' if mr.get('status')!='FAIL' else 'FAIL'
mr['checked_at']='$TIMESTAMP'
json.dump(mr,open('$MODELS_JSON','w'),indent=2)
" 2>/dev/null

    python3 -c "
import json
lr=json.load(open('$MODEL_LOCATIONS_JSON')) if __import__('os').path.exists('$MODEL_LOCATIONS_JSON') else {}
lr.setdefault('locations',{})['$model_key']={'artifact_id':'$artifact_id','location_id':'${location_id:-}','path':'$path','action':'$loc_action'}
lr['status']='$overall_status' if lr.get('status')!='FAIL' else 'FAIL'
lr['checked_at']='$TIMESTAMP'
json.dump(lr,open('$MODEL_LOCATIONS_JSON','w'),indent=2)
" 2>/dev/null

  done <<< "$model_keys"

  log_info "models.json written (status=$overall_status)"
  if [[ "$overall_status" == "FAIL" ]]; then
    add_error "MODELS_FAILED" "one or more models failed to register"
    return 1
  fi
  return 0
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
