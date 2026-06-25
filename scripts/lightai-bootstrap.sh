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
  local xh=()
  if [[ -f "$CSRF_FILE" && -s "$CSRF_FILE" ]]; then
    xh=(-H "X-CSRF-Token: $(tr -d '\n' < "$CSRF_FILE")")
  fi
  curl -sS -o "$output_file" -w '%{http_code}' -X POST "$FINAL_BASE_URL$path" \
    -H "Origin: $FINAL_BASE_URL" -H "Content-Type: application/json" \
    -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
    "${xh[@]}" -d "$body" 2>/dev/null || echo "000"
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
      must_change=$(python3 -c "import json; d=json.load(open('$resp_file')); v=d.get('must_change_password','false'); print('true' if v == True or str(v).lower() == 'true' else 'false')" 2>/dev/null || echo "true")
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
    must_change=$(python3 -c "import json; d=json.load(open('$resp_file')); v=d.get('must_change_password','false'); print('true' if v == True or str(v).lower() == 'true' else 'false')" 2>/dev/null || echo "true")

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
  local xh=()
  if [[ -f "$CSRF_FILE" && -s "$CSRF_FILE" ]]; then
    xh=(-H "X-CSRF-Token: $(tr -d '\n' < "$CSRF_FILE")")
  fi
  curl -sS -X POST "$FINAL_BASE_URL$path" \
    -H "Origin: $FINAL_BASE_URL" -H "Content-Type: application/json" \
    -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
    "${xh[@]}" -d "$body" -o "$output_file" -w '%{http_code}' 2>/dev/null || echo "000"
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
    local status=""
    # Retry with backoff for rate limiting
    for attempt in 1 2 3; do
      status=$(curl_server_post "/api/v1/auth/login" "{\"username\":\"$PROFILE_AUTH_USERNAME\",\"password\":\"$final_pw\"}" "$resp_file")
      if [[ "$status" == "429" ]]; then
        log_warn "rate limited, retrying in ${attempt}s..."
        sleep $((attempt * 2))
      else
        break
      fi
    done
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
  local status=""
  for attempt in 1 2 3; do
    status=$(curl_server_post "/api/v1/auth/login" "{\"username\":\"$PROFILE_AUTH_USERNAME\",\"password\":\"$init_pw\"}" "$resp_file")
    if [[ "$status" == "429" ]]; then log_warn "rate limited, retrying in ${attempt}s..."; sleep $((attempt * 2)); else break; fi
  done
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
  curl_api_get "/api/v1/model-artifacts" "$artifacts_resp" >/dev/null 2>&1 || true

  local models_result="{}" locations_result="{}" overall_status="PASS"

  while IFS= read -r model_key; do
    [[ -z "$model_key" ]] && continue
    log_info "processing model: $model_key"

    local display_name kind path
    display_name=$(yaml_get_nested "$PROFILE_FILE" "models" "$model_key.display_name" 2>/dev/null || echo "$model_key")
    kind=$(yaml_get_nested "$PROFILE_FILE" "models" "$model_key.kind" 2>/dev/null || echo "huggingface")
    path=$(yaml_get_nested "$PROFILE_FILE" "models" "$model_key.path" 2>/dev/null || echo "")

    # Determine format for API
    local format="huggingface" task_type="chat" path_type="directory"
    if [[ "$kind" == "gguf" ]]; then format="gguf"; task_type="completion"; path_type="file"; fi

    # Check path exists
    local path_exists="true"
    if [[ ! -e "$path" ]]; then
      log_error "model path not found: $path"
      add_error "MODEL_PATH_MISSING" "model $model_key path not found: $path"
      path_exists="false"
      :
    fi

    # Find existing artifact by name
    local artifact_id=""
    if [[ -f "$artifacts_resp" ]]; then
      artifact_id=$(python3 -c "
import json
data=json.load(open('$artifacts_resp'))
arr=data if isinstance(data,list) else data.get('data',data.get('items',[]))
found=None
for a in arr:
  if a.get('name','')=='$model_key':
    found=a.get('id',''); break
if not found:
  for a in arr:
    if a.get('path','')=='$path':
      found=a.get('id',''); break
print(found or '')
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
          :
        fi
      fi
    else
      log_info "reusing existing artifact: $model_key (id=$artifact_id)"
    fi

    update_bootstrap_state "model_artifact_ids" "$model_key" "$artifact_id"

    # Create model location — register model root first if needed
    local location_id="" loc_action="REUSE"
    if [[ "$action" != "FAIL" && -n "$artifact_id" ]]; then
      # Ensure model root exists for the path (needed for location creation)
      local model_root
      model_root=$(dirname "$path")
      if [[ "$kind" == "gguf" ]]; then model_root=$(dirname "$model_root"); fi
      local root_body="{\"path\":\"$model_root\",\"name\":\"default-model-root\"}"
      local root_status
      root_status=$(curl_api_post "/api/v1/nodes/$node_id/model-roots" "$root_body" /dev/null)
      log_info "model root check for $model_root: HTTP $root_status (may already exist)"
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
  log_info "===== runtimes-only mode ====="

  # Step 1: Run models-only first
  if ! run_models_only; then
    log_error "models-only failed, cannot proceed to runtimes"
    return 1
  fi

  local node_id
  node_id=$(get_node_id)
  if [[ -z "$node_id" ]]; then
    add_error "RUNTIME_NO_NODE" "no node available for NBR creation"
    return 1
  fi

  # Load IDs from bootstrap-state.json
  local backend_vllm_id backend_sglang_id backend_llamacpp_id
  local version_vllm_id version_sglang_id version_llamacpp_id
  local model_hf_id model_gguf_id
  backend_vllm_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['backend_ids']['vllm'])" 2>/dev/null || echo "")
  backend_sglang_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['backend_ids']['sglang'])" 2>/dev/null || echo "")
  backend_llamacpp_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['backend_ids']['llamacpp'])" 2>/dev/null || echo "")
  version_vllm_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['backend_version_ids']['vllm'])" 2>/dev/null || echo "")
  version_sglang_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['backend_version_ids']['sglang'])" 2>/dev/null || echo "")
  version_llamacpp_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['backend_version_ids']['llamacpp'])" 2>/dev/null || echo "")
  model_hf_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['model_artifact_ids']['qwen3_small'])" 2>/dev/null || echo "")
  model_gguf_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['model_artifact_ids']['qwen35_gguf'])" 2>/dev/null || echo "")

  log_info "backend IDs: vllm=$backend_vllm_id sglang=$backend_sglang_id llamacpp=$backend_llamacpp_id"
  log_info "version IDs: vllm=$version_vllm_id sglang=$version_sglang_id llamacpp=$version_llamacpp_id"

  # List existing runtimes for idempotency
  local existing_resp="$FINAL_OUTPUT_DIR/responses/backend-runtimes-list.json"
  mkdir -p "$FINAL_OUTPUT_DIR/responses"
  curl_api_get "/api/v1/backend-runtimes" "$existing_resp" >/dev/null 2>&1 || true

  # List existing NBRs for idempotency
  local existing_nbr_resp="$FINAL_OUTPUT_DIR/responses/node-backend-runtimes-list.json"
  curl_api_get "/api/v1/nodes/$node_id/backend-runtimes" "$existing_nbr_resp" >/dev/null 2>&1 || true

  local runtimes_result="{}" nbrs_result="{}" overall_status="PASS"

  # Helper: find existing runtime by name
  find_br_id() {
    local name="$1"
    if [[ -f "$existing_resp" ]]; then
      python3 -c "
import json;d=json.load(open('$existing_resp'))
arr=d if isinstance(d,list) else d.get('data',d.get('items',[]))
for r in arr:
  if r.get('name','')=='$name' or r.get('display_name','')=='$name':
    print(r.get('id',''));break
" 2>/dev/null
    fi
  }

  find_nbr_id() {
    local br_id="$1"
    if [[ -f "$existing_nbr_resp" ]]; then
      python3 -c "
import json;d=json.load(open('$existing_nbr_resp'))
arr=d if isinstance(d,list) else d.get('data',d.get('items',[]))
for r in arr:
  if r.get('backend_runtime_id','')=='$br_id':
    print(r.get('id',''));break
" 2>/dev/null
    fi
  }

  # Define runtimes to create: name key, backend, version, image, model_key
  declare -A RT_BACKEND RT_VERSION RT_IMAGE RT_MODEL RT_PARAMS
  RT_BACKEND[vllm]="vllm"; RT_BACKEND[sglang]="sglang"; RT_BACKEND[llamacpp]="llamacpp"
  for rt_key in vllm sglang llamacpp; do
    local rt_backend="${RT_BACKEND[$rt_key]}"
    local rt_version="" rt_image="" rt_model=""
    case "$rt_key" in
      vllm) rt_version="$version_vllm_id"; rt_image="vllm/vllm-openai:latest"; rt_model="$model_hf_id" ;;
      sglang) rt_version="$version_sglang_id"; rt_image="lmsysorg/sglang:latest"; rt_model="$model_hf_id" ;;
      llamacpp) rt_version="$version_llamacpp_id"; rt_image="ghcr.io/ggml-org/llama.cpp:server-cuda13"; rt_model="$model_gguf_id" ;;
    esac

    # Read from profile if available
    local pf_image pf_params_json
    pf_image=$(yaml_get_nested "$PROFILE_FILE" "runtimes" "$rt_key.image" 2>/dev/null || echo "")
    [[ -n "$pf_image" ]] && rt_image="$pf_image"

    log_info "processing runtime: $rt_key (backend=$rt_backend, image=$rt_image)"

    # Find existing BR
    local br_id
    br_id=$(find_br_id "$rt_key")
    local br_action="REUSE"
    if [[ -z "$br_id" ]]; then
      local br_body="{\"name\":\"$rt_key\",\"display_name\":\"$rt_key\",\"backend_id\":\"$backend_vllm_id\",\"backend_version_id\":\"$rt_version\",\"image_name\":\"$rt_image\",\"vendor\":\"nvidia\",\"template_name\":\"\",\"health_check_override_json\":\"{}\",\"args_override_json\":\"[]\",\"default_env_json\":\"{}\",\"entrypoint_override_json\":\"[]\",\"image_pull_policy\":\"if_not_present\"}"
      # Use the correct backend_id for each runtime
      case "$rt_key" in
        vllm) br_body="{\"name\":\"$rt_key\",\"display_name\":\"$rt_key\",\"backend_id\":\"$backend_vllm_id\",\"backend_version_id\":\"$rt_version\",\"image_name\":\"$rt_image\",\"vendor\":\"nvidia\",\"template_name\":\"\",\"health_check_override_json\":\"{}\",\"args_override_json\":\"[]\",\"default_env_json\":\"{}\",\"entrypoint_override_json\":\"[]\",\"image_pull_policy\":\"if_not_present\"}" ;;
        sglang) br_body="{\"name\":\"$rt_key\",\"display_name\":\"$rt_key\",\"backend_id\":\"$backend_sglang_id\",\"backend_version_id\":\"$rt_version\",\"image_name\":\"$rt_image\",\"vendor\":\"nvidia\",\"template_name\":\"\",\"health_check_override_json\":\"{}\",\"args_override_json\":\"[]\",\"default_env_json\":\"{}\",\"entrypoint_override_json\":\"[]\",\"image_pull_policy\":\"if_not_present\"}" ;;
        llamacpp) br_body="{\"name\":\"$rt_key\",\"display_name\":\"$rt_key\",\"backend_id\":\"$backend_llamacpp_id\",\"backend_version_id\":\"$rt_version\",\"image_name\":\"$rt_image\",\"vendor\":\"nvidia\",\"template_name\":\"\",\"health_check_override_json\":\"{}\",\"args_override_json\":\"[]\",\"default_env_json\":\"{}\",\"entrypoint_override_json\":\"[]\",\"image_pull_policy\":\"if_not_present\"}" ;;
      esac
      local br_resp="$FINAL_OUTPUT_DIR/responses/br-create-$rt_key.json"
      local br_status
      br_status=$(curl_api_post "/api/v1/backend-runtimes" "$br_body" "$br_resp")
      if [[ "$br_status" == "201" ]]; then
        br_id=$(python3 -c "import json;print(json.load(open('$br_resp')).get('id',''))" 2>/dev/null || echo "")
        br_action="CREATE"
        log_info "created BackendRuntime: $rt_key (id=$br_id)"
      else
        log_error "failed to create BackendRuntime $rt_key (HTTP $br_status)"
        add_error "BR_CREATE_FAILED" "create BackendRuntime $rt_key returned HTTP $br_status"
        br_action="FAIL"; :
      fi
    else
      log_info "reusing existing BackendRuntime: $rt_key (id=$br_id)"
    fi

    update_bootstrap_state "backend_runtime_ids" "$rt_key" "$br_id"

    # Create/Reuse NodeBackendRuntime
    local nbr_id="" nbr_action="REUSE" nbr_status="unknown"
    if [[ "$br_action" != "FAIL" && -n "$br_id" ]]; then
      nbr_id=$(find_nbr_id "$br_id")
      if [[ -z "$nbr_id" ]]; then
        local enable_body="{\"backend_runtime_id\":\"$br_id\",\"image_ref\":\"$rt_image\"}"
        local enable_resp="$FINAL_OUTPUT_DIR/responses/nbr-create-$rt_key.json"
        local enable_status
        enable_status=$(curl_api_post "/api/v1/nodes/$node_id/backend-runtimes/enable" "$enable_body" "$enable_resp")
        if [[ "$enable_status" == "200" ]]; then
          nbr_id=$(python3 -c "import json;d=json.load(open('$enable_resp'));print(d.get('id',d.get('node_backend_runtime_id','')))" 2>/dev/null || echo "")
          nbr_status=$(python3 -c "import json;d=json.load(open('$enable_resp'));print(d.get('status','unknown'))" 2>/dev/null || echo "unknown")
          nbr_action="CREATE"
          log_info "created NBR: $rt_key (id=$nbr_id, status=$nbr_status)"
        else
          log_error "failed to create NBR $rt_key (HTTP $enable_status)"
          add_error "NBR_CREATE_FAILED" "enable NBR $rt_key returned HTTP $enable_status"
          nbr_action="FAIL"; :
        fi
      else
        log_info "reusing existing NBR: $rt_key (id=$nbr_id)"
        # Get NBR status
        nbr_status=$(python3 -c "
import json;d=json.load(open('$existing_nbr_resp'))
arr=d if isinstance(d,list) else d.get('data',d.get('items',[]))
for r in arr:
  if r.get('id','')=='$nbr_id': print(r.get('status','unknown'));break
" 2>/dev/null || echo "unknown")
      fi
    fi

    update_bootstrap_state "node_backend_runtime_ids" "$rt_key" "${nbr_id:-}"

    # Update results
    python3 -c "
import json,os
# Runtimes result
rf='$FINAL_OUTPUT_DIR/backend-runtimes.json'
d=json.load(open(rf)) if os.path.exists(rf) else {'status':'PASS','runtimes':{},'checked_at':'$TIMESTAMP'}
d.setdefault('runtimes',{})['$rt_key']={'backend':'$rt_backend','backend_id':'$backend_vllm_id','backend_version_id':'$rt_version','image':'$rt_image','model':'$rt_model','backend_runtime_id':'$br_id','action':'$br_action'}
d['status']='$overall_status' if d.get('status')!='FAIL' else 'FAIL'
json.dump(d,open(rf,'w'),indent=2)
# NBR result
nf='$FINAL_OUTPUT_DIR/node-backend-runtimes.json'
d2=json.load(open(nf)) if os.path.exists(nf) else {'status':'PASS','node_id':'$node_id','node_backend_runtimes':{},'checked_at':'$TIMESTAMP'}
d2.setdefault('node_backend_runtimes',{})['$rt_key']={'backend_runtime_id':'$br_id','node_backend_runtime_id':'${nbr_id:-}','status':'$nbr_status','action':'$nbr_action'}
d2['status']='$overall_status' if d2.get('status')!='FAIL' else 'FAIL'
json.dump(d2,open(nf,'w'),indent=2)
" 2>/dev/null

  done

  log_info "backend-runtimes.json written (status=$overall_status)"
  log_info "node-backend-runtimes.json written (status=$overall_status)"
  if [[ "$overall_status" == "FAIL" ]]; then
    add_error "RUNTIMES_FAILED" "one or more runtimes failed to configure"
    return 1
  fi
  return 0
}

run_dry_run() {
  log_info "===== dry-run mode ====="

  # Step 1: Run runtimes-only first
  if ! run_runtimes_only; then
    log_error "runtimes-only failed, cannot proceed to dry-run"
    return 1
  fi

  local node_id
  node_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['ids']['node_id'])" 2>/dev/null || echo "")
  if [[ -z "$node_id" ]]; then add_error "DRYRUN_NO_NODE" "no node_id in bootstrap state"; return 1; fi

  # Load NBR IDs from state
  local nbr_vllm nbr_sglang nbr_llamacpp
  nbr_vllm=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['node_backend_runtime_ids']['vllm'])" 2>/dev/null || echo "")
  nbr_sglang=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['node_backend_runtime_ids']['sglang'])" 2>/dev/null || echo "")
  nbr_llamacpp=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['node_backend_runtime_ids']['llamacpp'])" 2>/dev/null || echo "")

  # Load model IDs
  local model_hf model_gguf
  model_hf=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['model_artifact_ids']['qwen3_small'])" 2>/dev/null || echo "")
  model_gguf=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['model_artifact_ids']['qwen35_gguf'])" 2>/dev/null || echo "")
  log_info "NBR IDs: vllm=$nbr_vllm sglang=$nbr_sglang llamacpp=$nbr_llamacpp"

  local check_results="{}" preflight_results="{}" runplan_results="{}" deployment_ids="{}"
  local overall_status="PASS"

  # Helper: check-request + poll until ready
  do_check_request() {
    local rt_key="$1" nbr_id="$2"
    log_info "check-request: $rt_key (nbr=$nbr_id)"
    local ck_resp="$FINAL_OUTPUT_DIR/responses/check-request-$rt_key.json"
    mkdir -p "$FINAL_OUTPUT_DIR/responses"
    local ck_status
    ck_status=$(curl_api_post "/api/v1/nodes/$node_id/backend-runtimes/$nbr_id/check-request" "{}" "$ck_resp" 2>/dev/null) || ck_status="000"
    log_info "check-request $rt_key: HTTP $ck_status"
    if [[ "$ck_status" != "200" ]]; then
      python3 -c "import json;d=json.load(open('$ck_resp'));print('status:',d.get('status','?'));print('reason:',d.get('status_reason',''))" 2>/dev/null
      echo "FAIL"
      return 1
    fi
    # Poll for ready status (up to 10 attempts = 50s)
    local nbr_status="" attempt=0 max_attempts=10
    while [[ $attempt -lt $max_attempts ]]; do
      local nbr_resp="$FINAL_OUTPUT_DIR/responses/nbr-status-$rt_key.json"
      curl_api_get "/api/v1/nodes/$node_id/backend-runtimes" "$nbr_resp" >/dev/null 2>&1 || true
      nbr_status=$(python3 -c "
import json;d=json.load(open('$nbr_resp'))
arr=d if isinstance(d,list) else d.get('data',d.get('items',[]))
for r in arr:
  if r.get('id','')=='$nbr_id': print(r.get('status',''));break
" 2>/dev/null || echo "")
      case "$nbr_status" in
        ready|ready_with_warnings) echo "$nbr_status"; return 0 ;;
        failed|missing_image|inspect_failed|docker_error|agent_unreachable) echo "FAIL:$nbr_status"; return 1 ;;
        *) sleep 5; attempt=$((attempt+1)) ;;
      esac
    done
    echo "FAIL:timeout_${nbr_status:-unknown}"
    return 1
  }

  # Helper: preflight + dry-run for a runtime
  do_preflight_and_dryrun() {
    local rt_key="$1" nbr_id="$2" model_id="$3" host_port="$4"
    # Preflight
    local pf_body="{\"model_artifact_id\":\"$model_id\",\"node_backend_runtime_id\":\"$nbr_id\",\"node_id\":\"$node_id\",\"host_port\":$host_port,\"accelerator_ids\":[\"0\"]}"
    local pf_resp="$FINAL_OUTPUT_DIR/responses/preflight-$rt_key.json"
    local pf_status
    pf_status=$(curl_api_post "/api/v1/deployments/preflight" "$pf_body" "$pf_resp")
    log_info "preflight $rt_key: HTTP $pf_status"
    local pf_pass="FAIL" pf_errors="[]" pf_warnings="[]"
    if [[ "$pf_status" == "200" ]]; then
      pf_pass=$(python3 -c "import json;d=json.load(open('$pf_resp'));print('PASS' if d.get('can_run',False) else 'FAIL')" 2>/dev/null || echo "FAIL")
      pf_errors=$(python3 -c "import json;d=json.load(open('$pf_resp'));print(json.dumps(d.get('errors',[])))" 2>/dev/null || echo "[]")
      pf_warnings=$(python3 -c "import json;d=json.load(open('$pf_resp'));print(json.dumps(d.get('warnings',[])))" 2>/dev/null || echo "[]")
    fi
    if [[ "$pf_pass" != "PASS" ]]; then
      log_error "preflight $rt_key FAILED: $(python3 -c "import json;print(json.load(open('$pf_resp')).get('errors',[])[:3])" 2>/dev/null)"
    fi

    # Create deployment for dry-run
    local depl_name="bootstrap-${rt_key}-dryrun"
    # Check for existing deployment
    local existing_depl_resp="$FINAL_OUTPUT_DIR/responses/deployments-list.json"
    curl_api_get "/api/v1/deployments" "$existing_depl_resp" >/dev/null 2>&1 || true
    local depl_id=""
    depl_id=$(python3 -c "
import json;d=json.load(open('$existing_depl_resp'))
arr=d if isinstance(d,list) else d.get('data',d.get('items',[]))
for r in arr:
  if r.get('name','')=='$depl_name': print(r.get('id',''));break
" 2>/dev/null || echo "")
    local depl_action="REUSE"
    if [[ -z "$depl_id" ]]; then
      local svc_json="{\"host_port\":$host_port,\"container_port\":0,\"app_port\":0}"
      local depl_body="{\"name\":\"$depl_name\",\"model_artifact_id\":\"$model_id\",\"node_backend_runtime_id\":\"$nbr_id\",\"service_json\":$svc_json,\"env_overrides_json\":\"{}\",\"parameter_values_json\":[],\"disabled_parameters_json\":[],\"placement_json\":\"{}\"}"
      local depl_resp="$FINAL_OUTPUT_DIR/responses/deployment-create-$rt_key.json"
      local depl_status
      depl_status=$(curl_api_post "/api/v1/deployments" "$depl_body" "$depl_resp")
      if [[ "$depl_status" == "200" || "$depl_status" == "201" ]]; then
        depl_id=$(python3 -c "import json;d=json.load(open('$depl_resp'));print(d.get('id',''))" 2>/dev/null || echo "")
        depl_action="CREATE"
        log_info "created deployment: $rt_key (id=$depl_id)"
      else
        log_error "failed to create deployment $rt_key (HTTP $depl_status)"
        python3 -c "import json;d=json.load(open('$depl_resp'));print(d.get('error','?'))" 2>/dev/null
      fi
    else
      log_info "reusing existing deployment: $rt_key (id=$depl_id)"
    fi
    update_bootstrap_state "deployment_ids" "$rt_key" "${depl_id:-}"

    # Dry-run
    local dr_pass="FAIL" dr_image="" dr_model="" dr_ports="[]" dr_args="[]" dr_env="false" dr_params="false"
    if [[ -n "$depl_id" ]]; then
      local dr_resp="$FINAL_OUTPUT_DIR/responses/dryrun-$rt_key.json"
      local dr_status
      dr_status=$(curl_api_post "/api/v1/deployments/$depl_id/dry-run" "{}" "$dr_resp")
      log_info "dry-run $rt_key: HTTP $dr_status"
      if [[ "$dr_status" == "200" ]]; then
        dr_pass="PASS"
        dr_image=$(python3 -c "import json;d=json.load(open('$dr_resp'));rp=d.get('resolved_run_plan',d);print(rp.get('image','?'))" 2>/dev/null || echo "?")
        dr_model=$(python3 -c "import json;d=json.load(open('$dr_resp'));rp=d.get('resolved_run_plan',d);print(rp.get('model_path',rp.get('model_location','?')))" 2>/dev/null || echo "?")
        dr_ports=$(python3 -c "import json;d=json.load(open('$dr_resp'));rp=d.get('resolved_run_plan',d);print(json.dumps(rp.get('ports',[])))" 2>/dev/null || echo "[]")
        dr_args=$(python3 -c "import json;d=json.load(open('$dr_resp'));rp=d.get('resolved_run_plan',d);print(json.dumps(rp.get('args',rp.get('cmd',[]))))" 2>/dev/null || echo "[]")
        dr_env=$(python3 -c "import json;d=json.load(open('$dr_resp'));rp=d.get('resolved_run_plan',d);print('true' if rp.get('env') else 'false')" 2>/dev/null || echo "false")
        dr_params=$(python3 -c "import json;d=json.load(open('$dr_resp'));rp=d.get('resolved_run_plan',d);print('true' if rp.get('resource_controls') or rp.get('parameters') else 'false')" 2>/dev/null || echo "false")
      fi
    fi

    python3 -c "
import json,os
# Preflight result
pf='$FINAL_OUTPUT_DIR/preflight-results.json'
d=json.load(open(pf)) if os.path.exists(pf) else {'status':'PASS','check_results':{},'preflight_results':{},'runplan_results':{},'deployment_ids':{},'checked_at':'$TIMESTAMP'}
d['preflight_results']['$rt_key']={'status':'$pf_pass','errors':$pf_errors,'warnings':$pf_warnings,'deployment_id':'${depl_id:-}','node_backend_runtime_id':'$nbr_id'}
d['runplan_results']['$rt_key']={'status':'$dr_pass','image':'$dr_image','model_path':'$dr_model','ports':${dr_ports:-[]},'args':${dr_args:-[]},'env_present':$([[ "$dr_env" == "true" ]] && echo "true" || echo "false"),'resource_parameters_present':$([[ "$dr_params" == "true" ]] && echo "true" || echo "false")}
d['deployment_ids']['$rt_key']='${depl_id:-}'
fails=0
for k in 'vllm' 'sglang' 'llamacpp':
  if d['preflight_results'].get(k,{}).get('status')=='FAIL': fails+=1
  if d['runplan_results'].get(k,{}).get('status')=='FAIL': fails+=1
d['status']='FAIL' if fails>0 else 'PASS'
json.dump(d,open(pf,'w'),indent=2)
" 2>/dev/null
    echo "$pf_pass"
  }

  # Process each runtime
  for rt_key in vllm sglang llamacpp; do
    local nbr_id="" model_id="" host_port=""
    case "$rt_key" in
      vllm) nbr_id="$nbr_vllm"; model_id="$model_hf"; host_port=8004 ;;
      sglang) nbr_id="$nbr_sglang"; model_id="$model_hf"; host_port=30000 ;;
      llamacpp) nbr_id="$nbr_llamacpp"; model_id="$model_gguf"; host_port=8002 ;;
    esac
    if [[ -z "$nbr_id" || -z "$model_id" ]]; then
      log_error "missing NBR or model ID for $rt_key"; :; continue
    fi

    # Check-request
    local ck_result
    ck_result=$(do_check_request "$rt_key" "$nbr_id")
    log_info "check $rt_key: $ck_result"
    local ck_entry="{\"node_backend_runtime_id\":\"$nbr_id\",\"status\":\"$ck_result\",\"action\":\"CHECK\",\"warnings\":[]}"
    if [[ "$ck_result" == FAIL:* ]]; then ck_entry="{\"node_backend_runtime_id\":\"$nbr_id\",\"status\":\"${ck_result#FAIL:}\",\"action\":\"CHECK\",\"warnings\":[]}"; :; fi

    python3 -c "
import json,os
pf='$FINAL_OUTPUT_DIR/preflight-results.json'
d=json.load(open(pf)) if os.path.exists(pf) else {'status':'PASS','check_results':{},'preflight_results':{},'runplan_results':{},'deployment_ids':{},'checked_at':'$TIMESTAMP'}
d['check_results']['$rt_key']=json.loads('''$ck_entry''')
d['status']='FAIL' if '${ck_result}'=='FAIL:'* else d.get('status','PASS')
json.dump(d,open(pf,'w'),indent=2)
" 2>/dev/null

    # If check passed, do preflight + dry-run
    if [[ "$ck_result" == ready || "$ck_result" == ready_with_warnings ]]; then
      local pf_result
      pf_result=$(do_preflight_and_dryrun "$rt_key" "$nbr_id" "$model_id" "$host_port")
      if [[ "$pf_result" != "PASS" ]]; then :; fi
    else
      log_warn "skipping preflight for $rt_key (NBR not ready: $ck_result)"
      add_error "DRYRUN_NBR_NOT_READY" "$rt_key NBR not ready: $ck_result (image may be missing — pull image and re-run check-request)"
      :
    fi
  done

  log_info "preflight-results.json written (status=$overall_status)"
  if [[ "$overall_status" == "FAIL" ]]; then
    add_error "DRYRUN_FAILED" "one or more dry-run checks failed"
    return 1
  fi
  return 0
}

run_full() {
  log_info "===== full mode ====="

  # Safety gate: double-allow required
  local profile_allow="false"
  profile_allow=$(yaml_get_nested "$PROFILE_FILE" "bootstrap" "allow_real_container_start" 2>/dev/null || echo "false")
  # Normalize Python True/true to lowercase
  profile_allow=$(echo "$profile_allow" | tr '[:upper:]' '[:lower:]')
  local cli_allow="${ALLOW_REAL_START:-false}"
  if [[ "$profile_allow" != "true" || "$cli_allow" != "true" ]]; then
    add_error "FAIL_FULL_NOT_ALLOWED" "full mode requires both profile.bootstrap.allow_real_container_start=true AND --allow-real-start flag"
    cat > "$FINAL_OUTPUT_DIR/full-results.json" << EOF
{"status":"FAIL","allow_real_start":false,"image_policy":"local_only_no_pull","containers_started":false,"model_instances_created":false,"results":{},"checked_at":"$TIMESTAMP","error":"FAIL_FULL_NOT_ALLOWED: set profile.bootstrap.allow_real_container_start=true and use --allow-real-start"}
EOF
    return 1
  fi
  log_info "full mode authorized: profile_allow=$profile_allow cli_allow=$cli_allow"

  # Run dry-run first
  if ! run_dry_run; then
    log_error "dry-run failed, cannot proceed to full"
    return 1
  fi

  local node_id profile_keep
  node_id=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['ids']['node_id'])" 2>/dev/null || echo "")
  profile_keep=$(yaml_get_nested "$PROFILE_FILE" "bootstrap" "keep_containers_after_full" 2>/dev/null || echo "false")
  profile_keep=$(echo "$profile_keep" | tr '[:upper:]' '[:lower:]')
  local containers_started="false" instances_created="false" overall_status="PASS"

  # Load NBR and deployment IDs
  local nbr_vllm nbr_sglang nbr_llamacpp
  nbr_vllm=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['node_backend_runtime_ids']['vllm'])" 2>/dev/null || echo "")
  nbr_sglang=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['node_backend_runtime_ids']['sglang'])" 2>/dev/null || echo "")
  nbr_llamacpp=$(python3 -c "import json;print(json.load(open('$BOOTSTRAP_STATE_JSON'))['node_backend_runtime_ids']['llamacpp'])" 2>/dev/null || echo "")

  declare -A RT_IMAGE RT_PORT RT_MODEL
  RT_IMAGE[vllm]="vllm/vllm-openai:latest"; RT_IMAGE[sglang]="lmsysorg/sglang:latest"; RT_IMAGE[llamacpp]="ghcr.io/ggml-org/llama.cpp:server-cuda13"
  RT_PORT[vllm]=8004; RT_PORT[sglang]=30000; RT_PORT[llamacpp]=8002
  RT_MODEL[vllm]="eb7e9fb3-f050-4f14-86c1-f743809cd3aa"; RT_MODEL[sglang]="eb7e9fb3-f050-4f14-86c1-f743809cd3aa"; RT_MODEL[llamacpp]="8f1dc586-ab25-476d-84b5-fc63d472bdc2"

  local full_json="$FINAL_OUTPUT_DIR/full-results.json"
  python3 -c "import json;json.dump({'status':'PASS','allow_real_start':True,'image_policy':'local_only_no_pull','containers_started':False,'model_instances_created':False,'results':{},'checked_at':'$TIMESTAMP'},open('$full_json','w'),indent=2)" 2>/dev/null

  # Check port conflicts
  log_info "checking port conflicts..."
  local port_conflicts=""
  for rt in vllm sglang llamacpp; do
    local hp="${RT_PORT[$rt]}"
    if ss -ltnp 2>/dev/null | grep -q ":$hp "; then
      log_warn "port $hp in use (runtime $rt)"
      port_conflicts="$port_conflicts $rt:$hp"
    fi
  done
  if [[ -n "$port_conflicts" ]]; then
    add_error "FULL_PORT_IN_USE" "ports in use:$port_conflicts — stop existing containers first"
    :
  fi

  for rt in vllm sglang llamacpp; do
    log_info "=== full mode: $rt ==="
    local rt_result="{\"image\":\"${RT_IMAGE[$rt]}\",\"image_present\":false,\"deployment_id\":\"\",\"instance_id\":\"\",\"node_run_plan_id\":\"\",\"container_id_present\":false,\"start_status\":\"SKIP\",\"instance_status\":\"not_started\",\"logs_status\":\"SKIP\",\"health_status\":\"SKIP\",\"models_endpoint_status\":\"SKIP\",\"chat_completion_status\":\"NOT_RUN\",\"stop_status\":\"SKIP\",\"action\":\"SKIP_IMAGE_MISSING\",\"recommended_manual_command\":\"docker pull ${RT_IMAGE[$rt]}\"}"

    # Check image
    if docker image inspect "${RT_IMAGE[$rt]}" >/dev/null 2>&1; then
      rt_result="{\"image\":\"${RT_IMAGE[$rt]}\",\"image_present\":true,\"deployment_id\":\"\",\"instance_id\":\"\",\"node_run_plan_id\":\"\",\"container_id_present\":false,\"start_status\":\"SKIP\",\"instance_status\":\"not_started\",\"logs_status\":\"SKIP\",\"health_status\":\"SKIP\",\"models_endpoint_status\":\"SKIP\",\"chat_completion_status\":\"NOT_RUN\",\"stop_status\":\"SKIP\",\"action\":\"START\",\"recommended_manual_command\":\"\"}"
    else
      add_error "FULL_IMAGE_MISSING" "$rt: image ${RT_IMAGE[$rt]} not found locally — docker pull required"
      python3 -c "
import json;d=json.load(open('$full_json'));d['results']['$rt']=json.loads('''$rt_result''');d['status']='FAIL';json.dump(d,open('$full_json','w'),indent=2)
" 2>/dev/null
      :
      continue
    fi

    # Find deployment_id from dry-run
    local depl_id
    depl_id=$(python3 -c "
import json,subprocess,os
# Get deployment by name
resp=subprocess.run(['curl','-sS','-b','$COOKIE_JAR','-H','Origin: $FINAL_BASE_URL','$FINAL_BASE_URL/api/v1/deployments'],capture_output=True,text=True)
if resp.returncode==0:
  try:
    d=json.loads(resp.stdout)
    arr=d if isinstance(d,list) else d.get('data',d.get('items',[]))
    for r in arr:
      if r.get('name','')=='bootstrap-${rt}-dryrun':
        print(r['id']);break
  except: pass
" 2>/dev/null || echo "")
    if [[ -z "$depl_id" ]]; then
      log_error "$rt: no deployment found for bootstrap-${rt}-dryrun"
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['start_status']='FAIL';d['action']='FAIL';print(json.dumps(d))" 2>/dev/null)
      python3 -c "import json;d=json.load(open('$full_json'));d['results']['$rt']=json.loads('''$rt_result''');d['status']='FAIL';json.dump(d,open('$full_json','w'),indent=2)" 2>/dev/null
      :
      continue
    fi
    rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['deployment_id']='$depl_id';print(json.dumps(d))" 2>/dev/null)

    # Skip if port in use by unrelated process
    if [[ -n "$port_conflicts" ]]; then
      log_warn "$rt: skipping start due to port conflict"
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['start_status']='SKIP';d['action']='SKIP_PORT_IN_USE';print(json.dumps(d))" 2>/dev/null)
      python3 -c "import json;d=json.load(open('$full_json'));d['results']['$rt']=json.loads('''$rt_result''');d['status']='FAIL';json.dump(d,open('$full_json','w'),indent=2)" 2>/dev/null
      :
      continue
    fi

    # Start deployment
    local start_resp="$FINAL_OUTPUT_DIR/responses/start-$rt.json"
    local start_status
    start_status=$(curl_api_post "/api/v1/deployments/$depl_id/start" "{}" "$start_resp")
    log_info "$rt start: HTTP $start_status"
    if [[ "$start_status" == "200" || "$start_status" == "201" ]]; then
      containers_started="true"
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['start_status']='PASS';d['action']='START';print(json.dumps(d))" 2>/dev/null)
    else
      log_error "$rt start failed: HTTP $start_status"
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['start_status']='FAIL';d['action']='FAIL';d['instance_status']='failed';print(json.dumps(d))" 2>/dev/null)
      python3 -c "import json;d=json.load(open('$full_json'));d['results']['$rt']=json.loads('''$rt_result''');d['status']='FAIL';json.dump(d,open('$full_json','w'),indent=2)" 2>/dev/null
      :
      continue
    fi

    # Poll for instance
    local instance_id="" npr_id="" cid_present="false" inst_state="not_started"
    for attempt in $(seq 1 20); do
      local inst_resp="$FINAL_OUTPUT_DIR/responses/instances-$rt.json"
      curl_api_get "/api/v1/model-instances?deployment_id=$depl_id" "$inst_resp" >/dev/null 2>&1 || true
      instance_id=$(python3 -c "
import json;d=json.load(open('$inst_resp'))
arr=d if isinstance(d,list) else d.get('data',d.get('items',[]))
if arr: i=arr[0]; print(i.get('id',''))
" 2>/dev/null || echo "")
      if [[ -n "$instance_id" ]]; then
        npr_id=$(python3 -c "import json;d=json.load(open('$inst_resp'));arr=d if isinstance(d,list) else d.get('data',[]);print(arr[0].get('current_run_plan_id','')) if arr else print('')" 2>/dev/null || echo "")
        inst_state=$(python3 -c "import json;d=json.load(open('$inst_resp'));arr=d if isinstance(d,list) else d.get('data',[]);print(arr[0].get('actual_state','running')) if arr else print('running')" 2>/dev/null || echo "running")
        local cid
        cid=$(python3 -c "import json;d=json.load(open('$inst_resp'));arr=d if isinstance(d,list) else d.get('data',[]);print(arr[0].get('container_id','') or '') if arr else print('')" 2>/dev/null || echo "")
        [[ -n "$cid" ]] && cid_present="true"
        instances_created="true"
        log_info "$rt instance: id=$instance_id state=$inst_state cid_present=$cid_present"
        update_bootstrap_state "instance_ids" "$rt" "$instance_id"
        [[ -n "$npr_id" ]] && update_bootstrap_state "node_run_plan_ids" "$rt" "$npr_id"
        break
      fi
      sleep 3
    done
    rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['instance_id']='$instance_id';d['node_run_plan_id']='$npr_id';d['container_id_present']=$([[ "$cid_present" == "true" ]] && echo 'true' || echo 'false');d['instance_status']='$inst_state';print(json.dumps(d))" 2>/dev/null)
    update_bootstrap_state "container_ids_present" "$rt" "$cid_present"

    # Logs
    if [[ -n "$npr_id" ]]; then
      local log_resp="$FINAL_OUTPUT_DIR/responses/logs-$rt.json"
      local log_status
      log_status=$(curl_api_get "/api/v1/node-run-plans/$npr_id/logs" "$log_resp" 2>/dev/null || echo "000")
      if [[ "$log_status" == "200" ]]; then
        rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['logs_status']='PASS';print(json.dumps(d))" 2>/dev/null)
        log_info "$rt logs retrieved"
      else
        rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['logs_status']='FAIL';print(json.dumps(d))" 2>/dev/null)
      fi
    fi

    # Health check (local port)
    local hp="${RT_PORT[$rt]}"
    if curl -sS -o /dev/null -w '' --max-time 5 "http://localhost:$hp/health" 2>/dev/null; then
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['health_status']='PASS';print(json.dumps(d))" 2>/dev/null)
      log_info "$rt health: PASS (http://localhost:$hp/health)"
    elif curl -sS -o /dev/null -w '' --max-time 5 "http://localhost:$hp/healthz" 2>/dev/null; then
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['health_status']='PASS';print(json.dumps(d))" 2>/dev/null)
      log_info "$rt health: PASS (http://localhost:$hp/healthz)"
    else
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['health_status']='WARN';print(json.dumps(d))" 2>/dev/null)
      log_warn "$rt health: not reachable on port $hp"
    fi

    # Models endpoint
    if curl -sS -o /dev/null -w '' --max-time 5 "http://localhost:$hp/v1/models" 2>/dev/null; then
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['models_endpoint_status']='PASS';print(json.dumps(d))" 2>/dev/null)
      log_info "$rt /v1/models: PASS"
    else
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['models_endpoint_status']='WARN';print(json.dumps(d))" 2>/dev/null)
      log_warn "$rt /v1/models: not reachable"
    fi

    # Stop (unless keep)
    if [[ "$profile_keep" != "true" ]]; then
      local stop_resp="$FINAL_OUTPUT_DIR/responses/stop-$rt.json"
      local stop_status
      stop_status=$(curl_api_post "/api/v1/deployments/$depl_id/stop" "{}" "$stop_resp")
      log_info "$rt stop: HTTP $stop_status"
      if [[ "$stop_status" == "200" ]]; then
        rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['stop_status']='PASS';print(json.dumps(d))" 2>/dev/null)
      else
        rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['stop_status']='FAIL';print(json.dumps(d))" 2>/dev/null)
      fi
    else
      rt_result=$(echo "$rt_result" | python3 -c "import json,sys;d=json.load(sys.stdin);d['stop_status']='SKIP';print(json.dumps(d))" 2>/dev/null)
      log_info "$rt: keeping container (keep_containers_after_full=true)"
    fi

    python3 -c "import json;d=json.load(open('$full_json'));d['results']['$rt']=json.loads('''$rt_result''');json.dump(d,open('$full_json','w'),indent=2)" 2>/dev/null
    rm -f "$start_resp" "$inst_resp" "$log_resp" "$stop_resp"
  done

  # Finalize
  python3 -c "
import json;d=json.load(open('$full_json'))
d['containers_started']=$([[ "$containers_started" == "true" ]] && echo 'true' || echo 'false')
d['model_instances_created']=$([[ "$instances_created" == "true" ]] && echo 'true' || echo 'false')
if d['status']!='FAIL':
  has_pass=False
  has_skip=False
  for v in d.get('results',{}).values():
    a=v.get('action','')
    if a in ('START','REUSE_RUNNING_INSTANCE'): has_pass=True
    elif a=='SKIP_IMAGE_MISSING': has_skip=True
  if has_pass and has_skip: d['status']='PARTIAL'
  elif has_pass: d['status']='PASS'
json.dump(d,open('$full_json','w'),indent=2)
" 2>/dev/null

  log_info "full-results.json written (status=$overall_status)"
  if [[ "$overall_status" == "FAIL" ]]; then
    add_error "FULL_FAILED" "one or more full checks failed"
    return 1
  fi
  return 0
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
