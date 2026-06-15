#!/bin/sh
# Test: Grafana CLI password reset — CLI pattern and credentials
# Verifies the reset scripts use the same --homepath/--config pattern as start-observability.sh
# (flags AFTER the subcommand for both "server" and "cli").
set -e

TEST_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$TEST_DIR/../.." && pwd)"
TMPDIR="${TEST_DIR}/tmp-reset-test-$$"
cleanup() {
  rm -f "$PROJECT_DIR/bin/grafana/bin/grafana" 2>/dev/null || true
  rm -rf "$TMPDIR"
}
trap cleanup EXIT

mkdir -p "$TMPDIR"

# ---- Setup mock Grafana binary ----
mkdir -p "$PROJECT_DIR/bin/grafana/bin"
cat > "$PROJECT_DIR/bin/grafana/bin/grafana" << 'MOCKEOF'
#!/bin/sh
echo "$0 $*" > "$MOCK_LOG_FILE"
exit 0
MOCKEOF
chmod +x "$PROJECT_DIR/bin/grafana/bin/grafana"
export MOCK_LOG_FILE="$TMPDIR/grafana-args.txt"

# Ensure required paths exist.
mkdir -p "$PROJECT_DIR/configs/observability" "$PROJECT_DIR/data/grafana" "$PROJECT_DIR/run" "$PROJECT_DIR/logs" "$PROJECT_DIR/runtime"

if [ ! -f "$PROJECT_DIR/configs/observability/grafana.ini" ]; then
  cat > "$PROJECT_DIR/configs/observability/grafana.ini" << 'INIEOF'
[database]
type = sqlite3
path = data/grafana/grafana.db
INIEOF
fi
touch "$PROJECT_DIR/data/grafana/grafana.db" 2>/dev/null || true

echo "=== Test 1: CLI argument order (reset-grafana-password.sh) ==="
rm -f "$MOCK_LOG_FILE"
TEST_PASS="TestPassword123!"

(
  cd "$PROJECT_DIR"
  sh scripts/reset-grafana-password.sh "$TEST_PASS" 2>&1
) || true

if [ -f "$MOCK_LOG_FILE" ]; then
  ARGS=$(cat "$MOCK_LOG_FILE")
  echo "  Mock args: $ARGS"

  # Verify: cli comes BEFORE --homepath (flags after subcommand, same as server)
  if echo "$ARGS" | grep -q 'cli.*homepath'; then
    echo "  PASS: cli before --homepath (matches server pattern)"
  else
    echo "  FAIL: cli NOT before --homepath"
  fi

  if echo "$ARGS" | grep -q 'cli.*config'; then
    echo "  PASS: cli before --config (matches server pattern)"
  else
    echo "  FAIL: cli NOT before --config"
  fi

  if echo "$ARGS" | grep -q "reset-admin-password.*$TEST_PASS"; then
    echo "  PASS: password follows reset-admin-password"
  else
    echo "  FAIL: password position wrong"
  fi
else
  echo "  SKIP: mock not called"
fi

echo ""
echo "=== Test 2: Credentials files ==="
if [ -f "$PROJECT_DIR/runtime/observability/grafana.credentials" ]; then
  echo "  PASS: runtime/observability/grafana.credentials written"
else
  echo "  FAIL: runtime/observability/grafana.credentials missing"
fi
if [ -f "$PROJECT_DIR/runtime/reset-credentials.txt" ]; then
  echo "  PASS: runtime/reset-credentials.txt written"
else
  echo "  FAIL: runtime/reset-credentials.txt missing"
fi

echo ""
echo "=== Test 3: DB path does NOT nest ==="
# Verify grafana.db is expected at data/grafana/grafana.db, not nested.
DB_FILES=$(find "$PROJECT_DIR/data" -name 'grafana.db' -print 2>/dev/null)
echo "  DB files found:"
echo "$DB_FILES" | while read -r f; do echo "    $f"; done

# Check for nested path (should NOT exist).
if echo "$DB_FILES" | grep -q 'data/grafana/data/'; then
  echo "  FAIL: nested data/grafana/data/grafana.db detected"
else
  echo "  PASS: no nested grafana.db path"
fi

echo ""
echo "=== Test 4: Shell syntax ==="
for s in scripts/reset-grafana-password.sh scripts/reset-password.sh scripts/start-observability.sh; do
  if bash -n "$PROJECT_DIR/$s" 2>/dev/null; then
    echo "  PASS: bash -n $s"
  else
    echo "  FAIL: bash -n $s"
  fi
done

echo ""
echo "=== All tests complete ==="
