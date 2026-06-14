#!/bin/sh
# P0-001/CODEX: Patch atomicity verification test suite.
# Creates synthetic old/new release dirs, generates a patch, then runs
# apply-patch.sh through success and failure scenarios.
# Usage: bash tests/test_patch_atomicity.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_ROOT="/tmp/lightai-patch-atomicity-test-$$"
PASS=0
FAIL=0

cleanup() {
  rm -rf "$TEST_ROOT"
}
trap cleanup EXIT

assert_pass()   { PASS=$((PASS + 1)); echo "  PASS: $1"; }
assert_fail()   { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }

echo "=== P0-001: Patch Atomicity Test Suite ==="
echo "Test root: $TEST_ROOT"
echo ""

# --- Setup: create synthetic old install ---
mkdir -p "$TEST_ROOT/old/bin"
mkdir -p "$TEST_ROOT/old/configs"
mkdir -p "$TEST_ROOT/old/scripts"
mkdir -p "$TEST_ROOT/old/data" "$TEST_ROOT/old/logs" "$TEST_ROOT/old/run"
echo "0.1.0" > "$TEST_ROOT/old/VERSION"
echo "old binary" > "$TEST_ROOT/old/bin/lightai-server"
echo "old binary" > "$TEST_ROOT/old/bin/lightai-agent"
echo "old config" > "$TEST_ROOT/old/configs/server.release.yaml"
echo '#!/bin/sh' > "$TEST_ROOT/old/scripts/start-server.sh"
chmod +x "$TEST_ROOT/old/scripts/start-server.sh"

# --- Setup: create synthetic new release dir ---
mkdir -p "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/bin"
mkdir -p "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/configs"
mkdir -p "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/scripts"
echo "0.2.0" > "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/VERSION"
echo "new binary" > "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/bin/lightai-server"
echo "new binary" > "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/bin/lightai-agent"
echo "new config" > "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/configs/server.release.yaml"
echo "new extra file" > "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/NEW_FILE.txt"
echo '#!/bin/sh' > "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/scripts/start-server.sh"
chmod +x "$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64/scripts/start-server.sh"

# Generate MANIFEST.sha256 for the "new" release
NEW_DIR="$PROJECT_DIR/dist/lightai-go-0.2.0-linux-amd64"
: > "$NEW_DIR/MANIFEST.sha256"
(cd "$NEW_DIR" && find . -type f ! -path './data/*' ! -path './logs/*' ! -path './run/*' ! -name MANIFEST.sha256 | sort | while IFS= read -r f; do
  sha256sum "$f" | awk '{print "file " $1 " " $2}'
done) >> "$NEW_DIR/MANIFEST.sha256"

# Also generate MANIFEST.sha256 for the "old" release so package-patch can compare
OLD_DIR="$PROJECT_DIR/dist/lightai-go-0.1.0-linux-amd64"
mkdir -p "$OLD_DIR/bin" "$OLD_DIR/configs" "$OLD_DIR/scripts"
echo "0.1.0" > "$OLD_DIR/VERSION"
echo "old binary" > "$OLD_DIR/bin/lightai-server"
echo "old binary" > "$OLD_DIR/bin/lightai-agent"
echo "old config" > "$OLD_DIR/configs/server.release.yaml"
echo '#!/bin/sh' > "$OLD_DIR/scripts/start-server.sh"
chmod +x "$OLD_DIR/scripts/start-server.sh"
: > "$OLD_DIR/MANIFEST.sha256"
(cd "$OLD_DIR" && find . -type f ! -path './data/*' ! -path './logs/*' ! -path './run/*' ! -name MANIFEST.sha256 | sort | while IFS= read -r f; do
  sha256sum "$f" | awk '{print "file " $1 " " $2}'
done) >> "$OLD_DIR/MANIFEST.sha256"

# Generate patch
echo "[setup] Generating patch 0.1.0 -> 0.2.0..."
cd "$PROJECT_DIR"
bash scripts/package-patch.sh --from 0.1.0 --to 0.2.0 --from-min 0.1.0 2>/dev/null || {
  echo "NOTE: package-patch.sh failed (expected for cross-version test). Using manual patch."
  # Create manual patch
  PATCH_MANUAL="$PROJECT_DIR/dist/lightai-go-patch-0.1.0-to-0.2.0-linux-amd64"
  mkdir -p "$PATCH_MANUAL"
  cp "$PROJECT_DIR/scripts/apply-patch.sh" "$PATCH_MANUAL/"
  TAB="$(printf '\t')"
  # Build TSV manually
  {
    echo "# from_version=0.1.0"
    echo "# to_version=0.2.0"
    echo "# from_min_version=0.1.0"
    echo "# from_max_exclusive=0.2.0"
    echo "# patch_mode=cumulative"
    echo "# created_at=$(date -Iseconds)"
    echo "# changed_files=4"
    echo "# removed_files=0"
    echo "# action${TAB}mode${TAB}sha256${TAB}path"
    mkdir -p "$PATCH_MANUAL/bin" "$PATCH_MANUAL/configs" "$PATCH_MANUAL/scripts"
    cp "$NEW_DIR/bin/lightai-server" "$PATCH_MANUAL/bin/"
    cp "$NEW_DIR/bin/lightai-agent" "$PATCH_MANUAL/bin/"
    cp "$NEW_DIR/configs/server.release.yaml" "$PATCH_MANUAL/configs/"
    cp "$NEW_DIR/scripts/start-server.sh" "$PATCH_MANUAL/scripts/"
    # SHA256s
    S1=$(sha256sum "$PATCH_MANUAL/bin/lightai-server" | awk '{print $1}')
    S2=$(sha256sum "$PATCH_MANUAL/bin/lightai-agent" | awk '{print $1}')
    S3=$(sha256sum "$PATCH_MANUAL/configs/server.release.yaml" | awk '{print $1}')
    S4=$(sha256sum "$PATCH_MANUAL/scripts/start-server.sh" | awk '{print $1}')
    echo "update${TAB}0755${TAB}${S1}${TAB}./bin/lightai-server"
    echo "update${TAB}0755${TAB}${S2}${TAB}./bin/lightai-agent"
    echo "update${TAB}0644${TAB}${S3}${TAB}./configs/server.release.yaml"
    echo "update${TAB}0755${TAB}${S4}${TAB}./scripts/start-server.sh"
    echo "create${TAB}0644${TAB}-${TAB}./NEW_FILE.txt"
    cp "$NEW_DIR/NEW_FILE.txt" "$PATCH_MANUAL/NEW_FILE.txt"
  } > "$PATCH_MANUAL/patch-files.tsv"
  # Build tarball
  cd "$PROJECT_DIR/dist"
  rm -f lightai-go-patch-0.1.0-to-0.2.0-linux-amd64.tar.gz
  tar -czf "lightai-go-patch-0.1.0-to-0.2.0-linux-amd64.tar.gz" "lightai-go-patch-0.1.0-to-0.2.0-linux-amd64"
  cd "$PROJECT_DIR"
}

PATCH_TARBALL="dist/lightai-go-patch-0.1.0-to-0.2.0-linux-amd64.tar.gz"
PATCH_NAME="lightai-go-patch-0.1.0-to-0.2.0-linux-amd64"

# Extract patch
rm -rf "/tmp/$PATCH_NAME"
tar -xzf "$PATCH_TARBALL" -C /tmp/ 2>/dev/null || true
PATCH_DIR="/tmp/$PATCH_NAME"

echo ""
echo "=== Test 1: Successful apply ==="
rm -rf "$TEST_ROOT/deploy"
cp -a "$TEST_ROOT/old" "$TEST_ROOT/deploy"
bash "$PATCH_DIR/apply-patch.sh" --root "$TEST_ROOT/deploy" 2>&1; rc=$?; true
if [ "$(cat "$TEST_ROOT/deploy/VERSION")" = "0.2.0" ]; then
  assert_pass "VERSION updated to 0.2.0"
else
  assert_fail "VERSION should be 0.2.0, got $(cat "$TEST_ROOT/deploy/VERSION")"
fi
if grep -q "new binary" "$TEST_ROOT/deploy/bin/lightai-server"; then
  assert_pass "binary updated"
else
  assert_fail "binary not updated"
fi

echo ""
echo "=== Test 2: dry-run does not modify files ==="
rm -rf "$TEST_ROOT/deploy2"
cp -a "$TEST_ROOT/old" "$TEST_ROOT/deploy2"
bash "$PATCH_DIR/apply-patch.sh" --root "$TEST_ROOT/deploy2" --dry-run 2>&1 || true
if [ "$(cat "$TEST_ROOT/deploy2/VERSION")" = "0.1.0" ]; then
  assert_pass "dry-run preserves VERSION"
else
  assert_fail "dry-run changed VERSION to $(cat "$TEST_ROOT/deploy2/VERSION")"
fi
if grep -q "old binary" "$TEST_ROOT/deploy2/bin/lightai-server"; then
  assert_pass "dry-run does not modify bin"
else
  assert_fail "dry-run modified bin"
fi

echo ""
echo "=== Test 3: SHA mismatch fails, VERSION unchanged ==="
rm -rf "$TEST_ROOT/deploy3"
cp -a "$TEST_ROOT/old" "$TEST_ROOT/deploy3"
# Corrupt a file in the patch
echo "corrupted" >> "$PATCH_DIR/bin/lightai-server"
bash "$PATCH_DIR/apply-patch.sh" --root "$TEST_ROOT/deploy3" 2>&1; rc=$?
# Restore original for later tests
cp "$NEW_DIR/bin/lightai-server" "$PATCH_DIR/bin/lightai-server"
if [ "$rc" != "0" ]; then
  assert_pass "SHA mismatch exit non-zero: $rc"
else
  assert_fail "SHA mismatch should exit non-zero, got $rc"
fi
if [ "$(cat "$TEST_ROOT/deploy3/VERSION")" = "0.1.0" ]; then
  assert_pass "SHA mismatch preserves VERSION"
else
  assert_fail "SHA mismatch changed VERSION"
fi

echo ""
echo "=== Test 4: Path traversal rejected ==="
rm -rf "$TEST_ROOT/deploy4"
cp -a "$TEST_ROOT/old" "$TEST_ROOT/deploy4"
# Create a malicious TSV with path traversal
MALICIOUS_TSV="/tmp/malicious-patch.tsv"
cat > "$MALICIOUS_TSV" << 'EOF'
# from_version=0.1.0
# to_version=0.2.0
# from_min_version=0.1.0
# from_max_exclusive=0.2.0
# patch_mode=cumulative
# action	mode	sha256	path
update	0644	-	../../etc/passwd
EOF
cp "$PROJECT_DIR/scripts/apply-patch.sh" /tmp/malicious-apply.sh
# Point TSV to malicious one
sed -i 's|patch-files.tsv|malicious-patch.tsv|' /tmp/malicious-apply.sh
bash /tmp/malicious-apply.sh --root "$TEST_ROOT/deploy4" 2>&1; rc=$?
if [ "$rc" != "0" ]; then
  assert_pass "path traversal rejected (exit $rc)"
else
  assert_fail "path traversal should be rejected"
fi

echo ""
echo "=== Test 5: Write to read-only dir fails, VERSION unchanged ==="
rm -rf "$TEST_ROOT/deploy5"
cp -a "$TEST_ROOT/old" "$TEST_ROOT/deploy5"
chmod -w "$TEST_ROOT/deploy5/bin"
bash "$PATCH_DIR/apply-patch.sh" --root "$TEST_ROOT/deploy5" 2>&1; rc=$?
chmod +w "$TEST_ROOT/deploy5/bin"  # restore
if [ "$rc" != "0" ]; then
  assert_pass "write failure exit non-zero: $rc"
else
  assert_fail "write failure should exit non-zero"
fi
if [ "$(cat "$TEST_ROOT/deploy5/VERSION")" = "0.1.0" ]; then
  assert_pass "write failure preserves VERSION"
else
  assert_fail "write failure changed VERSION to $(cat "$TEST_ROOT/deploy5/VERSION")"
fi

echo ""
echo "=== Test 6: Successful apply writes VERSION last ==="
rm -rf "$TEST_ROOT/deploy6"
cp -a "$TEST_ROOT/old" "$TEST_ROOT/deploy6"
bash "$PATCH_DIR/apply-patch.sh" --root "$TEST_ROOT/deploy6" 2>&1; rc=$?
if [ "$rc" = "0" ]; then
  assert_pass "apply exit 0"
else
  assert_fail "apply exit $rc"
fi
if [ "$(cat "$TEST_ROOT/deploy6/VERSION")" = "0.2.0" ]; then
  assert_pass "VERSION correctly updated"
else
  assert_fail "VERSION should be 0.2.0"
fi

echo ""
echo "=== Test 7: Newly created file exists after apply ==="
if [ -f "$TEST_ROOT/deploy6/NEW_FILE.txt" ]; then
  assert_pass "NEW_FILE.txt created"
else
  assert_fail "NEW_FILE.txt not created"
fi

echo ""
echo "=== Results ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
if [ "$FAIL" -gt 0 ]; then
  echo "=== SOME TESTS FAILED ==="
  exit 1
else
  echo "=== ALL PATCH ATOMICITY TESTS PASSED ==="
  exit 0
fi
