#!/bin/sh
# LightAI Go - GLIBC Compatibility Checker
# Scans all ELF binaries in a release directory and reports
# the highest GLIBC version required by each.
# Fails if any binary requires GLIBC >= 2.29.
# Usage: scripts/check-glibc-compat.sh <dist-dir>
set -e

DIR="${1:-dist}"
if [ ! -d "$DIR" ]; then
  echo "ERROR: directory not found: $DIR" >&2
  echo "Usage: $0 <release-dir>" >&2
  exit 1
fi

if ! command -v readelf >/dev/null 2>&1; then
  echo "ERROR: readelf not found. Install binutils." >&2
  exit 1
fi

echo "=== GLIBC Compatibility Check ==="
echo "Max allowed: glibc 2.28"
echo "Scanning:    $DIR"
echo ""

# Use temp file for violation tracking to avoid subshell issues.
TMP_VIOLATIONS="/tmp/glibc-check-violations-$$"
TMP_ELFS="/tmp/glibc-check-elfs-$$"
: > "$TMP_VIOLATIONS"
echo "0" > "$TMP_ELFS"

# Use 'file' command to find ELF binaries, process in a while loop.
find "$DIR" -type f -print | while IFS= read -r f; do
  # Skip non-ELF files.
  if ! file "$f" 2>/dev/null | grep -q 'ELF'; then
    continue
  fi

  # Increment counter.
  count=$(cat "$TMP_ELFS")
  echo $((count + 1)) > "$TMP_ELFS"

  relpath="${f#"$DIR"/}"

  # Extract all GLIBC version requirements and find the highest.
  highest=$(readelf -s "$f" 2>/dev/null | grep -o 'GLIBC_[0-9][0-9]*\.[0-9][0-9]*' | sort -t. -k1,1n -k2,2n -u | tail -1)

  if [ -z "$highest" ]; then
    echo "  OK    $relpath (no GLIBC symbols)"
    continue
  fi

  # Parse version number.
  ver_num="${highest#GLIBC_}"
  major="${ver_num%%.*}"
  minor="${ver_num#*.}"

  if [ "$major" -gt 2 ] 2>/dev/null || { [ "$major" -eq 2 ] 2>/dev/null && [ "$minor" -gt 28 ] 2>/dev/null; }; then
    echo "  FAIL  $relpath  requires $highest (max allowed: GLIBC_2.28)"
    echo "VIOLATION" >> "$TMP_VIOLATIONS"
  else
    echo "  OK    $relpath  (highest: $highest)"
  fi
done

ELFS_CHECKED=$(cat "$TMP_ELFS")
VIOLATIONS=$(wc -l < "$TMP_VIOLATIONS" 2>/dev/null || printf "0")
rm -f "$TMP_VIOLATIONS" "$TMP_ELFS"

echo ""
echo "ELFs checked: $ELFS_CHECKED"
echo "Violations:   $VIOLATIONS"

if [ "$VIOLATIONS" -gt 0 ]; then
  echo ""
  echo "=== RESULT: FAIL ($VIOLATIONS ELF(s) require GLIBC >= 2.29) ==="
  echo "Rebuild in the glibc 2.28 container:"
  echo "  ./scripts/package-release-docker.sh"
  exit 1
else
  echo ""
  echo "=== RESULT: PASS (all ELF binaries compatible with glibc <= 2.28) ==="
fi
