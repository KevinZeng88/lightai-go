#!/bin/sh
# LightAI Go - Apply Cumulative Patch
# Shell-native.  No python3 / jq / external parsers required.
# Primary manifest: patch-files.tsv (tab-separated, shell-parsed).
# Optional:         patch-manifest.json (enhanced display if python3 available).
# Usage: sh apply-patch.sh [--root <path>] [--force]

# --- semver compare (pure shell, returns 0/1/2 like strcmp) ---
semver_compare() {
  a_major="${1%%.*}"; a_rest="${1#*.}"
  a_minor="${a_rest%%.*}"; a_patch="${a_rest#*.}"
  b_major="${2%%.*}"; b_rest="${2#*.}"
  b_minor="${b_rest%%.*}"; b_patch="${b_rest#*.}"
  for part in "$a_major" "$a_minor" "$a_patch" "$b_major" "$b_minor" "$b_patch"; do
    case "$part" in ''|*[!0-9]*) echo "ERROR: invalid semver: $1 or $2" >&2; exit 1 ;; esac
  done
  if [ "$a_major" -gt "$b_major" ]; then return 1; fi
  if [ "$a_major" -lt "$b_major" ]; then return 2; fi
  if [ "$a_minor" -gt "$b_minor" ]; then return 1; fi
  if [ "$a_minor" -lt "$b_minor" ]; then return 2; fi
  if [ "$a_patch" -gt "$b_patch" ]; then return 1; fi
  if [ "$a_patch" -lt "$b_patch" ]; then return 2; fi
  return 0
}

# --- Argument parsing ---
ROOT="$(pwd)"
FORCE=false

while [ $# -gt 0 ]; do
  case "$1" in
    --root) ROOT="$2"; shift 2 ;;
    --force) FORCE=true; shift ;;
    --help|-h)
      echo "Usage: $0 [--root <path>] [--force]"
      echo "Apply a LightAI Go cumulative patch to a deployed release."
      echo ""
      echo "  --root <path>   Target deployment directory (default: current dir)."
      echo "  --force         Skip version checks (dangerous)."
      exit 0
      ;;
    *) echo "Usage: $0 [--root <path>] [--force]" >&2; exit 1 ;;
  esac
done

# --- Locate patch directory and manifests ---
PATCH_DIR="$(cd "$(dirname "$0")" && pwd)"
TSV="$PATCH_DIR/patch-files.tsv"
JSON="$PATCH_DIR/patch-manifest.json"

if [ ! -f "$TSV" ]; then
  echo "ERROR: patch-files.tsv not found in $PATCH_DIR" >&2
  exit 1
fi

# --- Read metadata from TSV header comments (# key=value) ---
FROM_VERSION=""
TO_VERSION=""
PATCH_MODE="cumulative"
while IFS= read -r line; do
  case "$line" in
    "# from_version="*) FROM_VERSION="${line#*=}" ;;
    "# to_version="*)   TO_VERSION="${line#*=}" ;;
    "# patch_mode="*)   PATCH_MODE="${line#*=}" ;;
    "# action"*)        break ;;  # end of header, start of data
  esac
done < "$TSV"

if [ -z "$FROM_VERSION" ] || [ -z "$TO_VERSION" ]; then
  echo "ERROR: could not read version metadata from patch-files.tsv" >&2
  exit 1
fi

# --- Enhanced display (optional: python3 for nicer JSON parsing) ---
echo "=== LightAI Go Patch ==="
echo "Patch:   $FROM_VERSION -> $TO_VERSION (mode: $PATCH_MODE)"

if [ -f "$JSON" ] && command -v python3 >/dev/null 2>&1; then
  CREATED=$(python3 -c "import json;print(json.load(open('$JSON')).get('created_at',''))" 2>/dev/null || echo "")
  CHANGED=$(python3 -c "import json;print(json.load(open('$JSON')).get('changed_files',''))" 2>/dev/null || echo "?")
  REMOVED=$(python3 -c "import json;print(json.load(open('$JSON')).get('removed_files',''))" 2>/dev/null || echo "?")
  [ -n "$CREATED" ] && echo "Created:  $CREATED"
  echo "Files:    $CHANGED changed, $REMOVED removed"
elif [ -f "$JSON" ]; then
  echo "Info:     install python3 for enhanced patch display"
fi

# --- Version check ---
if ! $FORCE; then
  if [ ! -f "$ROOT/VERSION" ]; then
    echo "ERROR: VERSION file not found in $ROOT (use --force to skip)" >&2
    exit 1
  fi
  CURRENT_VERSION=$(head -1 "$ROOT/VERSION" | tr -d '[:space:]')
  echo "Current:  $CURRENT_VERSION"
  echo ""

  if [ "$PATCH_MODE" = "cumulative" ]; then
    semver_compare "$CURRENT_VERSION" "$FROM_VERSION"
    cmp_from=$?
    if [ $cmp_from -eq 2 ]; then
      echo "ERROR: Current version ($CURRENT_VERSION) is BELOW from_version ($FROM_VERSION)." >&2
      echo "Upgrade to at least $FROM_VERSION first." >&2
      exit 1
    fi

    semver_compare "$CURRENT_VERSION" "$TO_VERSION"
    cmp_to=$?
    if [ $cmp_to -eq 1 ] || [ $cmp_to -eq 0 ]; then
      echo "ERROR: Current version ($CURRENT_VERSION) >= to_version ($TO_VERSION)." >&2
      echo "Already up to date or newer. Patch not needed." >&2
      exit 1
    fi
  fi
else
  echo "Current:  (forced)"
  echo ""
fi

# --- Safety: validate target root is a LightAI deployment ---
if [ ! -d "$ROOT/bin" ] && [ ! -d "$ROOT/scripts" ]; then
  echo "WARNING: $ROOT does not look like a LightAI deployment (no bin/ or scripts/)." >&2
  $FORCE || { echo "Use --force to override." >&2; exit 1; }
fi

# --- Apply patch ---
echo "Applying patch..."

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="$ROOT/backups/patch-$TIMESTAMP"
mkdir -p "$BACKUP_DIR"
echo "Backup: $BACKUP_DIR"

APPLIED=0
SKIPPED=0
FAILED=0

# Excluded path prefixes (runtime data that must never be overwritten).
EXCLUDES="data/ logs/ run/ runtime/ data/prometheus/ data/grafana/ backups/ .git/"

# Read TSV data lines (skip headers).
# Use printf-based tab for POSIX sh compatibility (dash does not support $'\t').
TAB="$(printf '\t')"
while IFS="$TAB" read -r action mode sha256 relpath; do
  # Skip header/comment lines.
  case "$action" in ""|\#*) continue ;; esac

  # Security: refuse absolute paths and path traversal.
  case "$relpath" in
    /*)  echo "SKIP: absolute path rejected: $relpath" >&2; SKIPPED=$((SKIPPED + 1)); continue ;;
    *..*) echo "SKIP: path traversal rejected: $relpath" >&2; SKIPPED=$((SKIPPED + 1)); continue ;;
  esac

  # Strip leading "./" if present.
  relpath="${relpath#./}"

  # Check excludes.
  skip=false
  for ex in $EXCLUDES; do
    case "$relpath" in $ex*) skip=true; break ;; esac
  done
  if $skip; then
    echo "SKIP: excluded: $relpath"
    SKIPPED=$((SKIPPED + 1))
    continue
  fi

  dst="$ROOT/$relpath"
  src="$PATCH_DIR/$relpath"

  case "$action" in
    update|create)
      if [ ! -f "$src" ]; then
        echo "FAIL: source missing: $relpath" >&2
        FAILED=$((FAILED + 1))
        continue
      fi

      # Verify SHA256 if provided.
      if [ -n "$sha256" ] && [ "$sha256" != "-" ]; then
        actual_sha=$(sha256sum "$src" | awk '{print $1}')
        if [ "$actual_sha" != "$sha256" ]; then
          echo "FAIL: SHA256 mismatch: $relpath (expected $sha256, got $actual_sha)" >&2
          FAILED=$((FAILED + 1))
          continue
        fi
      fi

      # Backup existing file.
      if [ -f "$dst" ]; then
        mkdir -p "$(dirname "$BACKUP_DIR/$relpath")" 2>/dev/null || true
        cp "$dst" "$BACKUP_DIR/$relpath" 2>/dev/null || true
      fi

      # Ensure target directory exists.
      mkdir -p "$(dirname "$dst")" 2>/dev/null || true

      # Copy file.
      cp "$src" "$dst"

      # Restore permissions (mode is octal, e.g. 0755).
      if [ -n "$mode" ] && [ "$mode" != "0000" ]; then
        chmod "$mode" "$dst" 2>/dev/null || true
      fi

      echo "  $action  $relpath"
      APPLIED=$((APPLIED + 1))
      ;;

    delete)
      if [ -f "$dst" ] || [ -L "$dst" ]; then
        # Backup before removal.
        mkdir -p "$(dirname "$BACKUP_DIR/$relpath")" 2>/dev/null || true
        cp "$dst" "$BACKUP_DIR/$relpath" 2>/dev/null || true
        rm -f "$dst"
        echo "  delete  $relpath"
        APPLIED=$((APPLIED + 1))
      else
        echo "  skip    $relpath (already absent)"
        SKIPPED=$((SKIPPED + 1))
      fi
      ;;

    *)
      echo "SKIP: unknown action '$action' for $relpath" >&2
      SKIPPED=$((SKIPPED + 1))
      ;;
  esac
done < "$TSV"

# --- Update VERSION ---
if [ -f "$ROOT/VERSION" ]; then
  cp "$ROOT/VERSION" "$BACKUP_DIR/VERSION" 2>/dev/null || true
fi
echo "$TO_VERSION" > "$ROOT/VERSION"

# --- Summary ---
echo ""
echo "=== Patch Applied ==="
echo "Version: $FROM_VERSION -> $TO_VERSION"
echo "Applied: $APPLIED items"
echo "Skipped: $SKIPPED items"
if [ $FAILED -gt 0 ]; then
  echo "Failed:  $FAILED items"
fi
echo "Backup:  $BACKUP_DIR"
echo ""
echo "Rollback: cp -r $BACKUP_DIR/* $ROOT/"
echo "Restart:  cd $ROOT && sh scripts/start-server.sh && sh scripts/start-agent.sh && sh scripts/start-observability.sh"

if [ $FAILED -gt 0 ]; then
  exit 1
fi
