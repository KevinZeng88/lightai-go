#!/bin/sh
# LightAI Go - GLIBC Compatibility Checker
# Scans ELF binaries in given directories and reports
# the highest GLIBC version required by each.
# Fails if any binary requires GLIBC >= 2.29.
#
# Usage: scripts/check-glibc-compat.sh <dir> [<dir> ...]
#
# Each <dir> should be a specific release or patch staging directory.
# The script does NOT scan a top-level dist/ that contains all historical
# versions — those were checked when they were built.
set -e

if [ $# -eq 0 ]; then
  echo "ERROR: no target directory specified." >&2
  echo "Usage: $0 <release-dir> [<release-dir> ...]" >&2
  echo "  e.g.: $0 dist/lightai-go-0.1.18-linux-amd64" >&2
  echo "        $0 dist/lightai-go-0.1.18-linux-amd64 dist/lightai-go-patch-0.1.17-to-0.1.18-linux-amd64" >&2
  exit 1
fi

if ! command -v readelf >/dev/null 2>&1; then
  echo "ERROR: readelf not found. Install binutils." >&2
  exit 1
fi

echo "=== GLIBC Compatibility Check ==="
echo "Max allowed: glibc 2.28"

# Collect all ELF files first (deduplicate across directories).
TMP_FILELIST="/tmp/glibc-check-files-$$"
: > "$TMP_FILELIST"

for dir in "$@"; do
  if [ ! -d "$dir" ]; then
    echo "ERROR: directory not found: $dir" >&2
    rm -f "$TMP_FILELIST"
    exit 1
  fi
  echo "Scanning:    $dir"
  find "$dir" -type f -print >> "$TMP_FILELIST"
done
echo ""

# Deduplicate file list (in case a file appears under multiple target dirs — unlikely but safe).
sort -u "$TMP_FILELIST" > "${TMP_FILELIST}.uniq"
mv "${TMP_FILELIST}.uniq" "$TMP_FILELIST"

# Use temp file for violation tracking to avoid subshell issues.
TMP_VIOLATIONS="/tmp/glibc-check-violations-$$"
: > "$TMP_VIOLATIONS"
ELF_CHECKED=0

while IFS= read -r f; do
  [ -z "$f" ] && continue

  # Skip non-ELF files.
  if ! file "$f" 2>/dev/null | grep -q 'ELF'; then
    continue
  fi

  ELF_CHECKED=$((ELF_CHECKED + 1))

  # Show path relative to the first matching target dir for readability.
  relpath="$f"
  for dir in "$@"; do
    case "$f" in
      "$dir/"*) relpath="${f#"$dir"/}"; break ;;
    esac
  done

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
done < "$TMP_FILELIST"

rm -f "$TMP_FILELIST"

VIOLATIONS=$(wc -l < "$TMP_VIOLATIONS" 2>/dev/null || printf "0")
rm -f "$TMP_VIOLATIONS"

echo ""
echo "ELFs checked: $ELF_CHECKED"
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
