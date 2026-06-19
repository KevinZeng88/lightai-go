#!/bin/bash
# e2e-assert.sh — shared assertion library for LightAI E2E scripts.
# Source this file in E2E scripts: source "$(dirname "$0")/e2e-assert.sh"
#
# All assert functions print "PASS: <msg>" or "FAIL: <msg>" and return 1 on failure.
# Callers should use `|| return 1` or `|| exit 1` to propagate failures.
# In cleanup sections, use `|| true` explicitly with a comment.

set -euo pipefail

_FAIL_COUNT=0
_PASS_COUNT=0

_assert_pass() { _PASS_COUNT=$((_PASS_COUNT + 1)); echo "PASS: $1"; }
_assert_fail() { _FAIL_COUNT=$((_FAIL_COUNT + 1)); echo "FAIL: $1"; }

assert_eq() {
  local msg="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    _assert_pass "$msg"
  else
    _assert_fail "$msg (expected='$expected' actual='$actual')"
    return 1
  fi
}

assert_not_eq() {
  local msg="$1" unexpected="$2" actual="$3"
  if [ "$unexpected" != "$actual" ]; then
    _assert_pass "$msg"
  else
    _assert_fail "$msg (unexpected='$unexpected' actual='$actual')"
    return 1
  fi
}

assert_nonempty() {
  local msg="$1" value="$2"
  if [ -n "$value" ]; then
    _assert_pass "$msg"
  else
    _assert_fail "$msg (value is empty)"
    return 1
  fi
}

assert_empty() {
  local msg="$1" value="$2"
  if [ -z "$value" ]; then
    _assert_pass "$msg"
  else
    _assert_fail "$msg (expected empty, got='$value')"
    return 1
  fi
}

assert_contains() {
  local msg="$1" haystack="$2" needle="$3"
  if echo "$haystack" | grep -qF -- "$needle"; then
    _assert_pass "$msg"
  else
    _assert_fail "$msg (needle='$needle' not found in haystack)"
    return 1
  fi
}

assert_not_contains() {
  local msg="$1" haystack="$2" needle="$3"
  if echo "$haystack" | grep -qF -- "$needle"; then
    _assert_fail "$msg (found unexpected='$needle' in haystack)"
    return 1
  else
    _assert_pass "$msg"
  fi
}

assert_http_ok() {
  local msg="$1" code="$2"
  if [ "$code" -ge 200 ] 2>/dev/null && [ "$code" -lt 300 ] 2>/dev/null; then
    _assert_pass "$msg (HTTP $code)"
  else
    _assert_fail "$msg (HTTP $code not 2xx)"
    return 1
  fi
}

assert_exactly_one_flag() {
  local msg="$1" haystack="$2" flag="$3"
  local count; count=$(echo "$haystack" | grep -oF -- "$flag" | wc -l)
  if [ "$count" -eq 1 ]; then
    _assert_pass "$msg"
  else
    _assert_fail "$msg ($flag appears $count times, expected 1)"
    return 1
  fi
}

assert_no_flag() {
  local msg="$1" haystack="$2" flag="$3"
  if echo "$haystack" | grep -qF -- "$flag"; then
    _assert_fail "$msg ($flag found but should be absent)"
    return 1
  else
    _assert_pass "$msg"
  fi
}

assert_flag_value() {
  local msg="$1" haystack="$2" flag="$3" expected_val="$4"
  # Extract value after flag: grep for "flag value" pattern
  local actual; actual=$(echo "$haystack" | grep -oP -- "${flag}\s+\K[^\s]+" | tail -1)
  if [ "$actual" = "$expected_val" ]; then
    _assert_pass "$msg"
  else
    _assert_fail "$msg ($flag value='$actual' expected='$expected_val')"
    return 1
  fi
}

assert_json_field_nonempty() {
  local msg="$1" json="$2" field="$3"
  local val; val=$(echo "$json" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    parts = '$field'.split('.')
    for p in parts:
        if isinstance(d, list):
            d = d[int(p)]
        else:
            d = d[p]
    if d is None or d == '':
        sys.exit(1)
    print(d)
except (KeyError, IndexError, TypeError, ValueError, json.JSONDecodeError) as e:
    sys.exit(1)
" 2>/dev/null) || true
  if [ -n "$val" ]; then
    _assert_pass "$msg (value=$val)"
  else
    _assert_fail "$msg (field '$field' missing, null, empty, or parse error in: ${json:0:200})"
    return 1
  fi
}

# Strict JSON helpers — fail on missing/null/empty/parse-error
json_require() {
  local json="$1" field="$2"
  local val; val=$(echo "$json" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    parts = '$field'.split('.')
    for p in parts:
        if isinstance(d, list):
            d = d[int(p)]
        else:
            d = d[p]
    if d is None or d == '':
        sys.exit(1)
    print(d)
except Exception:
    sys.exit(1)
" 2>/dev/null) || { echo "FAIL: json_require field '$field' missing/null/empty/parse-error"; return 1; }
  echo "$val"
}

json_get_optional() {
  local json="$1" field="$2" default="${3:-}"
  local val; val=$(echo "$json" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    parts = '$field'.split('.')
    for p in parts:
        if isinstance(d, list):
            d = d[int(p)]
        else:
            d = d[p]
    if d is None:
        print('')
    else:
        print(d)
except Exception:
    print('')
" 2>/dev/null) || true
  if [ -n "$val" ]; then
    echo "$val"
  else
    echo "$default"
  fi
}

assert_summary() {
  local pass=$_PASS_COUNT fail=$_FAIL_COUNT
  echo ""
  echo "=== ASSERTION SUMMARY ==="
  echo "PASS: $pass"
  echo "FAIL: $fail"
  if [ "$fail" -gt 0 ]; then
    echo "RESULT: FAIL"
    return 1
  else
    echo "RESULT: PASS"
    return 0
  fi
}
