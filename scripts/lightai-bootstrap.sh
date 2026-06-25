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

# ── Mode dispatch ────────────────────────────────────────────────────
run_auth_only() {
  log_info "mode: auth-only"
  add_error "NOT_IMPLEMENTED" "auth-only mode not yet implemented (Batch 3)"
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
