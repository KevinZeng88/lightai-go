#!/bin/bash
# e2e-assert-selftest.sh — self-test for e2e-assert.sh functions.
# Verifies that each assert function correctly passes and fails.
# Must be run to validate assertion library before any E2E depends on it.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/e2e-assert.sh"

PASSED=0
FAILED=0

check_pass() {
  local desc="$1" fn="$2"
  if eval "$fn"; then
    PASSED=$((PASSED + 1))
    echo "  OK: $desc -> PASS (expected)"
  else
    FAILED=$((FAILED + 1))
    echo "  XX: $desc -> FAIL (expected PASS)"
  fi
}

check_fail() {
  local desc="$1" fn="$2"
  # Run in subshell so return 1 doesn't exit the self-test
  if (set +e; eval "$fn" 2>/dev/null); then
    FAILED=$((FAILED + 1))
    echo "  XX: $desc -> PASS (expected FAIL)"
  else
    PASSED=$((PASSED + 1))
    echo "  OK: $desc -> FAIL (expected)"
  fi
}

echo "=== e2e-assert self-test ==="

# ── assert_nonempty ──
check_pass "nonempty hello"         'assert_nonempty test "hello"'
check_fail  "nonempty empty string" 'assert_nonempty test ""'

# ── assert_contains ──
check_pass "contains world"         'assert_contains test "hello world" "world"'
check_fail  "contains missing"      'assert_contains test "hello world" "foo"'

# ── assert_not_contains ──
check_pass "not_contains missing"   'assert_not_contains test "hello world" "foo"'
check_fail  "not_contains present"  'assert_not_contains test "hello world" "world"'

# ── assert_exactly_one_flag ──
check_pass "exactly_one 1 occ"      'assert_exactly_one_flag test "--port 8000" "--port"'
check_fail  "exactly_one 0 occ"     'assert_exactly_one_flag test "" "--port"'
check_fail  "exactly_one 2 occ"     'assert_exactly_one_flag test "--port 1 --port 2" "--port"'

# ── assert_no_flag ──
check_pass "no_flag absent"         'assert_no_flag test "--port 8000" "--model"'
check_fail  "no_flag present"       'assert_no_flag test "--port 8000" "--port"'

# ── assert_http_ok ──
check_pass "http 200"               'assert_http_ok test 200'
check_pass "http 204"               'assert_http_ok test 204'
check_fail  "http 400"              'assert_http_ok test 400'
check_fail  "http 500"              'assert_http_ok test 500'

# ── assert_eq ──
check_pass "eq match"               'assert_eq test "hello" "hello"'
check_fail  "eq mismatch"           'assert_eq test "hello" "world"'

# ── assert_empty ──
check_pass "empty true"             'assert_empty test ""'
check_fail  "empty nonempty"        'assert_empty test "hello"'

# ── assert_json_field_nonempty ──
check_pass "json field exists"      'assert_json_field_nonempty test "{\"key\":\"value\"}" "key"'
check_fail  "json field empty"      'assert_json_field_nonempty test "{\"key\":\"\"}" "key"'
check_fail  "json field null"       'assert_json_field_nonempty test "{\"key\":null}" "key"'
check_fail  "json field missing"    'assert_json_field_nonempty test "{}" "key"'
check_fail  "json parse error"      'assert_json_field_nonempty test "invalid" "key"'

# ── json_require ──
check_pass "require field exists"   'json_require "{\"key\":\"v\"}" "key" > /dev/null'
check_fail  "require field null"    'json_require "{\"key\":null}" "key" > /dev/null'
check_fail  "require field missing" 'json_require "{}" "key" > /dev/null'

# ── assert_flag_value ──
check_pass "flag_value correct"     'assert_flag_value test "--port 8022" --port 8022'
check_fail  "flag_value wrong"      'assert_flag_value test "--port 8022" --port 8000'

echo ""
echo "=== SELF-TEST SUMMARY ==="
echo "Passed: $PASSED"
echo "Failed: $FAILED"
if [ "$FAILED" -gt 0 ]; then
  echo "RESULT: FAIL"
  exit 1
else
  echo "RESULT: PASS"
  exit 0
fi
