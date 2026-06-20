#!/usr/bin/env bash
# Cleanup orchestration for LightAI E2E scripts.

if [ "${LIGHTAI_E2E_CLEANUP_SH:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
LIGHTAI_E2E_CLEANUP_SH=1

set -euo pipefail

E2E_CLEANUP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$E2E_CLEANUP_DIR/env.sh"
source "$E2E_CLEANUP_DIR/report.sh"

E2E_CLEANUP_FILE="$LIGHTAI_E2E_ARTIFACT_DIR/cleanup.sh"
: > "$E2E_CLEANUP_FILE"
chmod +x "$E2E_CLEANUP_FILE"

E2E_KEEP_EVIDENCE="${E2E_KEEP_EVIDENCE:-}"
E2E_KEEP_ON_SUCCESS="${E2E_KEEP_ON_SUCCESS:-0}"

e2e_cleanup_add() {
  local command="$*"
  printf '%s\n' "$command" >> "$E2E_CLEANUP_FILE"
}

e2e_cleanup_run() {
  local status="${1:-0}"
  if [ "$status" -eq 0 ]; then
    if [ -s "$E2E_CLEANUP_FILE" ]; then
      tac "$E2E_CLEANUP_FILE" | while IFS= read -r cleanup_cmd; do
        [ -n "$cleanup_cmd" ] || continue
        bash -c "$cleanup_cmd" || true
      done
    fi
    e2e_report_event PASS "cleanup" "completed"
    if [ "$E2E_KEEP_ON_SUCCESS" != "1" ] && [ -z "$E2E_KEEP_EVIDENCE" ]; then
      e2e_log "evidence retained at $LIGHTAI_E2E_ARTIFACT_DIR"
    fi
  else
    e2e_report_event FAIL "cleanup" "failure path: evidence retained at $LIGHTAI_E2E_ARTIFACT_DIR"
    e2e_log "failure: evidence retained at $LIGHTAI_E2E_ARTIFACT_DIR"
  fi
}

e2e_with_cleanup_trap() {
  trap 'rc=$?; e2e_cleanup_run "$rc"; e2e_report_finish "$([ "$rc" -eq 0 ] && echo PASS || echo FAIL)" "exit_code=$rc"; exit "$rc"' EXIT
}
