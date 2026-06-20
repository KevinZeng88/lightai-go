#!/usr/bin/env bash
# Evidence report helpers for LightAI E2E scripts.

if [ "${LIGHTAI_E2E_REPORT_SH:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
LIGHTAI_E2E_REPORT_SH=1

set -euo pipefail

E2E_REPORT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$E2E_REPORT_DIR/env.sh"

E2E_REPORT_EVENTS="$LIGHTAI_E2E_ARTIFACT_DIR/events.jsonl"
E2E_REPORT_MD="$LIGHTAI_E2E_ARTIFACT_DIR/summary.md"
E2E_REPORT_JSON="$LIGHTAI_E2E_ARTIFACT_DIR/summary.json"

e2e_report_init() {
  : > "$E2E_REPORT_EVENTS"
  cat > "$E2E_REPORT_MD" <<EOF
# LightAI E2E Summary

- run_id: $LIGHTAI_E2E_RUN_ID
- mode: $LIGHTAI_E2E_MODE
- server: $LIGHTAI_SERVER_URL
- agent: $LIGHTAI_AGENT_URL
- evidence: $LIGHTAI_E2E_ARTIFACT_DIR

## Events
EOF
}

e2e_report_event() {
  local status="$1" name="$2" detail="${3:-}"
  local ts
  ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  python3 - "$E2E_REPORT_EVENTS" "$ts" "$status" "$name" "$detail" <<'PY'
import json
import sys

path, ts, status, name, detail = sys.argv[1:]
with open(path, "a", encoding="utf-8") as f:
    f.write(json.dumps({"time": ts, "status": status, "name": name, "detail": detail}, ensure_ascii=False) + "\n")
PY
  printf -- '- %s `%s` %s\n' "$status" "$name" "$detail" >> "$E2E_REPORT_MD"
}

e2e_report_finish() {
  local status="$1" detail="${2:-}"
  e2e_report_event "$status" "finish" "$detail"
  python3 - "$E2E_REPORT_EVENTS" "$E2E_REPORT_JSON" "$status" "$LIGHTAI_E2E_RUN_ID" "$LIGHTAI_E2E_ARTIFACT_DIR" <<'PY'
import json
import sys

events_path, out_path, status, run_id, evidence = sys.argv[1:]
events = []
try:
    with open(events_path, encoding="utf-8") as f:
        events = [json.loads(line) for line in f if line.strip()]
except FileNotFoundError:
    pass
with open(out_path, "w", encoding="utf-8") as f:
    json.dump({"run_id": run_id, "status": status, "evidence": evidence, "events": events}, f, ensure_ascii=False, indent=2)
PY
}

e2e_report_init
