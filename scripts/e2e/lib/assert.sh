#!/usr/bin/env bash
# Shared assertions for LightAI E2E scripts.

if [ "${LIGHTAI_E2E_ASSERT_SH:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
LIGHTAI_E2E_ASSERT_SH=1

set -euo pipefail

E2E_ASSERT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Keep the existing assertion helpers available for older scripts.
source "$E2E_ASSERT_DIR/e2e-assert.sh"

e2e_json_get() {
  local expr="$1"
  python3 -c '
import json
import sys

expr = sys.argv[1]
try:
    data = json.load(sys.stdin)
    cur = data
    for part in expr.split("."):
        if part == "":
            continue
        if isinstance(cur, list):
            cur = cur[int(part)]
        elif isinstance(cur, dict):
            cur = cur[part]
        else:
            raise KeyError(part)
    if cur is None:
        sys.exit(1)
    if isinstance(cur, (dict, list)):
        print(json.dumps(cur, ensure_ascii=False, sort_keys=True))
    else:
        print(cur)
except Exception:
    sys.exit(1)
' "$expr"
}

e2e_json_type() {
  local expr="$1"
  python3 -c '
import json
import sys

expr = sys.argv[1]
try:
    data = json.load(sys.stdin)
    cur = data
    for part in expr.split("."):
        if part == "":
            continue
        if isinstance(cur, list):
            cur = cur[int(part)]
        elif isinstance(cur, dict):
            cur = cur[part]
        else:
            raise KeyError(part)
    print(type(cur).__name__)
except Exception:
    sys.exit(1)
' "$expr"
}

e2e_assert_json_eq() {
  local msg="$1" json="$2" expr="$3" expected="$4"
  local actual
  actual="$(printf '%s' "$json" | e2e_json_get "$expr")" || actual=""
  assert_eq "$msg" "$expected" "$actual"
}

e2e_assert_json_nonempty() {
  local msg="$1" json="$2" expr="$3"
  local actual
  actual="$(printf '%s' "$json" | e2e_json_get "$expr")" || actual=""
  assert_nonempty "$msg" "$actual"
}

e2e_assert_json_type() {
  local msg="$1" json="$2" expr="$3" expected="$4"
  local actual
  actual="$(printf '%s' "$json" | e2e_json_type "$expr")" || actual=""
  assert_eq "$msg" "$expected" "$actual"
}
