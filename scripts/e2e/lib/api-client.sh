#!/usr/bin/env bash
# Shared API client for LightAI local E2E scripts.

if [ "${LIGHTAI_E2E_API_CLIENT_SH:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
LIGHTAI_E2E_API_CLIENT_SH=1

set -euo pipefail

E2E_API_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$E2E_API_DIR/env.sh"
source "$E2E_API_DIR/report.sh"

E2E_CSRF_TOKEN="${E2E_CSRF_TOKEN:-}"
E2E_API_COUNTER=0
E2E_LAST_STATUS=""
E2E_LAST_BODY=""

e2e_api_url() {
  local path="$1"
  case "$path" in
    http://*|https://*) printf '%s' "$path" ;;
    /api/*) printf '%s%s' "$LIGHTAI_SERVER_URL" "$path" ;;
    /*) printf '%s/api/v1%s' "$LIGHTAI_SERVER_URL" "$path" ;;
    *) printf '%s/api/v1/%s' "$LIGHTAI_SERVER_URL" "$path" ;;
  esac
}

e2e_server_ready() {
  curl -fsS "$LIGHTAI_SERVER_URL/healthz" >/dev/null 2>&1 || \
    curl -fsS "$LIGHTAI_SERVER_URL/api/v1/observability/status" >/dev/null 2>&1
}

e2e_wait_server_ready() {
  local timeout="${1:-30}"
  local deadline=$((SECONDS + timeout))
  while [ "$SECONDS" -le "$deadline" ]; do
    if e2e_server_ready; then
      e2e_report_event PASS "server_ready" "$LIGHTAI_SERVER_URL"
      return 0
    fi
    sleep 1
  done
  e2e_report_event FAIL "server_ready" "$LIGHTAI_SERVER_URL"
  e2e_die "server not ready: $LIGHTAI_SERVER_URL"
}

e2e_agent_ready() {
  curl -fsS "$LIGHTAI_AGENT_URL/metrics" >/dev/null 2>&1 || \
    curl -fsS "$LIGHTAI_AGENT_URL/healthz" >/dev/null 2>&1
}

e2e_wait_agent_ready() {
  local timeout="${1:-30}"
  local deadline=$((SECONDS + timeout))
  while [ "$SECONDS" -le "$deadline" ]; do
    if e2e_agent_ready; then
      e2e_report_event PASS "agent_ready" "$LIGHTAI_AGENT_URL"
      return 0
    fi
    sleep 1
  done
  e2e_report_event FAIL "agent_ready" "$LIGHTAI_AGENT_URL"
  e2e_die "agent not ready: $LIGHTAI_AGENT_URL"
}

e2e_login() {
  e2e_require_cmd curl
  e2e_require_cmd python3
  rm -f "$LIGHTAI_E2E_COOKIE_JAR"
  local body response_file status_file
  body="{\"username\":\"$LIGHTAI_E2E_USERNAME\",\"password\":\"$LIGHTAI_E2E_PASSWORD\"}"
  response_file="$LIGHTAI_E2E_ARTIFACT_DIR/responses/login.json"
  status_file="$LIGHTAI_E2E_ARTIFACT_DIR/responses/login.status"
  status="$(curl -sS -o "$response_file" -w '%{http_code}' -X POST "$LIGHTAI_SERVER_URL/api/v1/auth/login" \
    -H "Origin: $LIGHTAI_SERVER_URL" -H "Content-Type: application/json" \
    -d "$body" -c "$LIGHTAI_E2E_COOKIE_JAR")"
  printf '%s\n' "$status" > "$status_file"
  if [ "$status" != "200" ]; then
    e2e_report_event FAIL "login" "HTTP $status"
    e2e_die "login failed HTTP $status; response: $response_file"
  fi
  E2E_CSRF_TOKEN="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("csrf_token",""))' "$response_file")"
  [ -n "$E2E_CSRF_TOKEN" ] || e2e_die "login response missing csrf_token"
  e2e_report_event PASS "login" "$LIGHTAI_E2E_USERNAME"
}

e2e_api() {
  local method="$1" path="$2" body="${3:-}" want_status="${4:-}"
  local url request_file response_file status_file status
  E2E_API_COUNTER=$((E2E_API_COUNTER + 1))
  url="$(e2e_api_url "$path")"
  request_file="$LIGHTAI_E2E_ARTIFACT_DIR/requests/$(printf '%03d' "$E2E_API_COUNTER")-${method}.json"
  response_file="$LIGHTAI_E2E_ARTIFACT_DIR/responses/$(printf '%03d' "$E2E_API_COUNTER")-${method}.json"
  status_file="$LIGHTAI_E2E_ARTIFACT_DIR/responses/$(printf '%03d' "$E2E_API_COUNTER")-${method}.status"
  printf '%s\n' "$body" > "$request_file"

  local args=(-sS -o "$response_file" -w '%{http_code}' -X "$method" "$url" -b "$LIGHTAI_E2E_COOKIE_JAR" -c "$LIGHTAI_E2E_COOKIE_JAR" -H "Origin: $LIGHTAI_SERVER_URL" -H "Content-Type: application/json")
  if [ -n "$E2E_CSRF_TOKEN" ] && [ "$method" != "GET" ]; then
    args+=(-H "X-CSRF-Token: $E2E_CSRF_TOKEN")
  fi
  if [ -n "$body" ]; then
    args+=(-d "$body")
  fi

  status="$(curl "${args[@]}")"
  E2E_LAST_STATUS="$status"
  E2E_LAST_BODY="$(cat "$response_file")"
  printf '%s\n' "$status" > "$status_file"

  if [ -n "$want_status" ] && [ "$status" != "$want_status" ]; then
    e2e_report_event FAIL "api_${method}_${path}" "HTTP $status want $want_status"
    e2e_die "$method $path HTTP $status want $want_status; response: $response_file"
  fi
  e2e_report_event PASS "api_${method}_${path}" "HTTP $status"
  cat "$response_file"
}

e2e_api_get() { e2e_api GET "$1" "" "${2:-200}"; }
e2e_api_post() { e2e_api POST "$1" "${2:-{}}" "${3:-200}"; }
e2e_api_patch() { e2e_api PATCH "$1" "${2:-{}}" "${3:-200}"; }
e2e_api_delete() { e2e_api DELETE "$1" "${2:-}" "${3:-200}"; }
