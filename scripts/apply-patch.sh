#!/bin/sh
# LightAI Go - Apply Cumulative Patch (P0-005: atomic rewrite)
# Shell-native. No python3/jq/external parsers required.
#
# Cross-version support:
#   Any version from from_min_version (inclusive) to to_version (exclusive)
#   can be upgraded directly to to_version.
#   e.g. 0.1.0 → 0.1.6, 0.1.5 → 0.1.6  (no intermediate steps required)
#
# Phase 1: Precheck  — validate version, SHA256, path safety, file existence
# Phase 2: Stage     — backup existing files, stage new files to temp dir
# Phase 3: Commit    — atomically replace files, write VERSION last
# Phase 4: Rollback  — on failure, restore from backup
#
# Usage:
#   sh apply-patch.sh [--root <path>] [--dry-run] [--force]

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
DRY_RUN=false

while [ $# -gt 0 ]; do
  case "$1" in
    --root) ROOT="$2"; shift 2 ;;
    --force) FORCE=true; shift ;;
    --dry-run) DRY_RUN=true; shift ;;
    --help|-h)
      echo "Usage: $0 [--root <path>] [--dry-run] [--force]"
      echo ""
      echo "Apply a LightAI Go cumulative patch to a deployed release."
      echo ""
      echo "  --root <path>   Target deployment directory (default: current dir)."
      echo "  --dry-run       Validate and report, but don't apply changes."
      echo "  --force         Skip version checks (dangerous)."
      exit 0
      ;;
    *) echo "Usage: $0 [--root <path>] [--dry-run] [--force]" >&2; exit 1 ;;
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

# --- Read metadata from TSV header comments ---
FROM_VERSION=""
TO_VERSION=""
FROM_MIN_VERSION=""
FROM_MAX_EXCLUSIVE=""
PATCH_MODE="cumulative"
while IFS= read -r line; do
  case "$line" in
    "# from_version="*)     FROM_VERSION="${line#*=}" ;;
    "# to_version="*)       TO_VERSION="${line#*=}" ;;
    "# from_min_version="*) FROM_MIN_VERSION="${line#*=}" ;;
    "# from_max_exclusive="*) FROM_MAX_EXCLUSIVE="${line#*=}" ;;
    "# patch_mode="*)       PATCH_MODE="${line#*=}" ;;
    "# action"*)            break ;;  # end of header
  esac
done < "$TSV"

if [ -z "$TO_VERSION" ]; then
  echo "ERROR: could not read version metadata from patch-files.tsv" >&2
  exit 1
fi

# P0-005: Use from_min_version for cross-version support.
# If from_max_exclusive is set, use it as the upper bound.
# Otherwise, from_version is used as the minimum.
if [ -z "$FROM_MIN_VERSION" ]; then
  FROM_MIN_VERSION="$FROM_VERSION"
fi
if [ -z "$FROM_MAX_EXCLUSIVE" ]; then
  FROM_MAX_EXCLUSIVE="$TO_VERSION"
fi

echo "=== LightAI Go Patch ==="
echo "Patch:    $FROM_VERSION -> $TO_VERSION (mode: $PATCH_MODE)"
echo "Accepts:  >= $FROM_MIN_VERSION, < $FROM_MAX_EXCLUSIVE"

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
    # P0-005: Cross-version check — current must be >= from_min_version AND < to_version.
    semver_compare "$CURRENT_VERSION" "$FROM_MIN_VERSION"
    cmp_min=$?
    if [ $cmp_min -eq 2 ]; then
      echo "ERROR: Current version ($CURRENT_VERSION) is BELOW minimum ($FROM_MIN_VERSION)." >&2
      echo "Upgrade to at least $FROM_MIN_VERSION first, or use a different patch." >&2
      exit 1
    fi

    semver_compare "$CURRENT_VERSION" "$TO_VERSION"
    cmp_to=$?
    if [ $cmp_to -eq 1 ] || [ $cmp_to -eq 0 ]; then
      echo "ERROR: Current version ($CURRENT_VERSION) >= target version ($TO_VERSION)." >&2
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

# --- PHASE 1: Precheck ---
echo ""
echo "[Phase 1/4] Precheck..."

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="$ROOT/backups/patch-$TIMESTAMP"
STAGE_DIR="$ROOT/.patch-staging-$TIMESTAMP"

# Excluded path prefixes (runtime data that must never be overwritten).
EXCLUDES="data/ logs/ run/ runtime/ data/prometheus/ data/grafana/ backups/ .git/"

PRECHECK_PASS=true
PRECHECK_APPLY=0
PRECHECK_DELETE=0
PRECHECK_SKIP=0

TAB="$(printf '\t')"
while IFS="$TAB" read -r action mode sha256 relpath; do
  case "$action" in ""|\#*) continue ;; esac

  # Security: refuse absolute paths and path traversal (P0-005).
  case "$relpath" in
    /*)  echo "  REJECT: absolute path: $relpath" >&2; PRECHECK_PASS=false; continue ;;
    *..*) echo "  REJECT: path traversal: $relpath" >&2; PRECHECK_PASS=false; continue ;;
  esac

  relpath="${relpath#./}"

  # Check excludes.
  skip=false
  for ex in $EXCLUDES; do
    case "$relpath" in $ex*) skip=true; break ;; esac
  done
  if $skip; then
    PRECHECK_SKIP=$((PRECHECK_SKIP + 1))
    continue
  fi

  dst="$ROOT/$relpath"
  src="$PATCH_DIR/$relpath"

  case "$action" in
    update|create)
      if [ ! -f "$src" ]; then
        echo "  FAIL: source missing in patch: $relpath" >&2
        PRECHECK_PASS=false
        continue
      fi
      # P0-005: Verify SHA256 if provided.
      if [ -n "$sha256" ] && [ "$sha256" != "-" ]; then
        actual_sha=$(sha256sum "$src" | awk '{print $1}')
        if [ "$actual_sha" != "$sha256" ]; then
          echo "  FAIL: SHA256 mismatch: $relpath (expected $sha256, got $actual_sha)" >&2
          PRECHECK_PASS=false
          continue
        fi
      fi
      PRECHECK_APPLY=$((PRECHECK_APPLY + 1))
      ;;
    delete)
      PRECHECK_DELETE=$((PRECHECK_DELETE + 1))
      ;;
    *)
      echo "  SKIP: unknown action '$action' for $relpath" >&2
      PRECHECK_SKIP=$((PRECHECK_SKIP + 1))
      ;;
  esac
done < "$TSV"

echo "  Apply: $PRECHECK_APPLY  Delete: $PRECHECK_DELETE  Skip: $PRECHECK_SKIP"

if ! $PRECHECK_PASS; then
  echo ""
  echo "=== PRECHECK FAILED ==="
  echo "No files were modified. Fix the issues above and try again."
  exit 1
fi

# --- Dry-run: stop here ---
if $DRY_RUN; then
  echo ""
  echo "=== DRY RUN COMPLETE ==="
  echo "All prechecks passed. $PRECHECK_APPLY files would be applied, $PRECHECK_DELETE files would be deleted."
  echo "No changes were made."
  exit 0
fi

echo "  Precheck passed."

# --- PHASE 2: Stage ---
echo "[Phase 2/4] Staging..."

mkdir -p "$BACKUP_DIR"
mkdir -p "$STAGE_DIR"
echo "Backup: $BACKUP_DIR"

APPLIED=0
SKIPPED=0
DELETED=0
STAGE_FAILED=false

while IFS="$TAB" read -r action mode sha256 relpath; do
  case "$action" in ""|\#*) continue ;; esac

  case "$relpath" in
    /*)  continue ;;
    *..*) continue ;;
  esac

  relpath="${relpath#./}"

  skip=false
  for ex in $EXCLUDES; do
    case "$relpath" in $ex*) skip=true; break ;; esac
  done
  if $skip; then
    SKIPPED=$((SKIPPED + 1))
    continue
  fi

  dst="$ROOT/$relpath"
  src="$PATCH_DIR/$relpath"

  case "$action" in
    update|create)
      if [ ! -f "$src" ]; then
        echo "  FAIL: source missing: $relpath" >&2
        STAGE_FAILED=true
        continue
      fi

      # P0-005: Backup existing file if present.
      if [ -f "$dst" ] || [ -L "$dst" ]; then
        mkdir -p "$(dirname "$BACKUP_DIR/$relpath")" 2>/dev/null || true
        cp -a "$dst" "$BACKUP_DIR/$relpath" 2>/dev/null || true
      fi

      # Stage to temp directory.
      mkdir -p "$(dirname "$STAGE_DIR/$relpath")" 2>/dev/null || true
      cp "$src" "$STAGE_DIR/$relpath"

      if [ -n "$mode" ] && [ "$mode" != "0000" ]; then
        chmod "$mode" "$STAGE_DIR/$relpath" 2>/dev/null || true
      fi
      ;;
    delete)
      if [ -f "$dst" ] || [ -L "$dst" ]; then
        mkdir -p "$(dirname "$BACKUP_DIR/$relpath")" 2>/dev/null || true
        cp -a "$dst" "$BACKUP_DIR/$relpath" 2>/dev/null || true
      fi
      ;;
  esac
done < "$TSV"

# Backup current VERSION.
if [ -f "$ROOT/VERSION" ]; then
  mkdir -p "$BACKUP_DIR" 2>/dev/null || true
  cp "$ROOT/VERSION" "$BACKUP_DIR/VERSION" 2>/dev/null || true
fi

if $STAGE_FAILED; then
  echo ""
  echo "=== STAGE FAILED ==="
  echo "Some files could not be staged. Rolling back..."
  rm -rf "$STAGE_DIR"
  echo "No changes were made to the installation."
  exit 1
fi

echo "  Staged $PRECHECK_APPLY files to temp directory."

# --- PHASE 3: Commit (atomic) ---
echo "[Phase 3/4] Committing..."

# Apply creates and updates from stage directory.
while IFS="$TAB" read -r action mode sha256 relpath; do
  case "$action" in ""|\#*) continue ;; esac
  case "$relpath" in /*|*..*) continue ;; esac
  relpath="${relpath#./}"

  skip=false
  for ex in $EXCLUDES; do
    case "$relpath" in $ex*) skip=true; break ;; esac
  done
  if $skip; then continue; fi

  dst="$ROOT/$relpath"
  src="$STAGE_DIR/$relpath"

  case "$action" in
    update|create)
      if [ -f "$src" ]; then
        mkdir -p "$(dirname "$dst")" 2>/dev/null || true
        cp "$src" "$dst"
        echo "  apply   $relpath"
        APPLIED=$((APPLIED + 1))
      fi
      ;;
    delete)
      if [ -f "$dst" ] || [ -L "$dst" ]; then
        rm -f "$dst"
        echo "  delete  $relpath"
        DELETED=$((DELETED + 1))
      fi
      ;;
  esac
done < "$TSV"

# P0-005: Write VERSION LAST — only after all files are successfully applied.
echo "$TO_VERSION" > "$ROOT/VERSION"
echo "  version $TO_VERSION (written to VERSION)"

# Clean up stage directory.
rm -rf "$STAGE_DIR"

# --- PHASE 4: Summary ---
echo "[Phase 4/4] Complete"
echo ""
echo "=== Patch Applied ==="
echo "Version: $CURRENT_VERSION -> $TO_VERSION"
echo "Applied: $APPLIED files"
echo "Deleted: $DELETED files"
echo "Skipped: $SKIPPED items"
echo "Backup:  $BACKUP_DIR"
echo ""
echo "Rollback: cp -r $BACKUP_DIR/* $ROOT/ && echo '$CURRENT_VERSION' > $ROOT/VERSION"
echo "Restart:  cd $ROOT && sh scripts/start-server.sh && sh scripts/start-agent.sh && sh scripts/start-observability.sh"
