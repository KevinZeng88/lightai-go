#!/bin/sh
# LightAI Go - Docker-based Release Package Builder (pure wrapper)
#
# This script is a THIN WRAPPER around scripts/package-release.sh.
# ALL build logic (version management, Web build, Go build, manifest,
# packaging) lives in package-release.sh. This script only handles:
#   - Docker availability check
#   - Build image validation
#   - Host artifact cleanup
#   - Container execution
#   - Post-build glibc compatibility check
#   - File ownership fixup
#
# Usage:
#   scripts/package-release-docker.sh [--no-bump | --bump patch | --version X.Y.Z] [...]
#   scripts/package-release-docker.sh --no-glibc-check --no-bump
#
# Build image (set via env var or --image):
#   LIGHTAI_BUILD_IMAGE=linux-build:el8-glibc2.28
#
# Wrapper-only options (NOT passed to package-release.sh):
#   --image <image>         Override build image
#   --no-glibc-check        Skip post-build glibc ABI check
#   --container-workdir <dir>  Override container workdir (default: /workspace)

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# --- Defaults ---
BUILD_IMAGE="${LIGHTAI_BUILD_IMAGE:-linux-build:el8-glibc2.28}"
CONTAINER_WORKDIR="/workspace"
DO_GLIBC_CHECK=true

# --- Separate wrapper args from package-release.sh args ---
PACKAGE_ARGS=""

while [ $# -gt 0 ]; do
  case "$1" in
    --image)
      BUILD_IMAGE="$2"; shift 2 ;;
    --no-glibc-check)
      DO_GLIBC_CHECK=false; shift ;;
    --container-workdir)
      CONTAINER_WORKDIR="$2"; shift 2 ;;
    --help|-h)
      echo "Usage: $0 [wrapper-options] [-- <package-release.sh args>]"
      echo ""
      echo "Wrapper options:"
      echo "  --image <img>          Use specified Docker image (default: $BUILD_IMAGE)"
      echo "  --no-glibc-check       Skip post-build glibc ABI verification"
      echo "  --container-workdir <d> Override container working directory"
      echo ""
      echo "All other arguments are passed through to scripts/package-release.sh:"
      echo "  --version <ver>        Explicit version"
      echo "  --bump patch|minor|major  Auto-increment version"
      echo "  --no-bump              Re-package current version"
      echo "  --dry-run              Preview version only"
      echo "  --without-observability  Skip bundling Prometheus/Grafana"
      echo ""
      echo "Environment:"
      echo "  LIGHTAI_BUILD_IMAGE    Override default build image"
      exit 0
      ;;
    *)
      PACKAGE_ARGS="$PACKAGE_ARGS $1"; shift ;;
  esac
done

# Default: pass --no-bump if no args specified.
if [ -z "$PACKAGE_ARGS" ]; then
  PACKAGE_ARGS="--no-bump"
fi

echo "=== LightAI Go Docker Release Builder (wrapper) ==="
echo "Image:      $BUILD_IMAGE"
echo "Workdir:    $CONTAINER_WORKDIR"
echo "Args:      $PACKAGE_ARGS"
echo ""

# --- Step 1: Check Docker ---
if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker not found. Install Docker to use this wrapper." >&2
  echo "Alternatively, run scripts/package-release.sh directly on a compatible host." >&2
  exit 1
fi

# --- Step 2: Check build image exists ---
if ! docker image inspect "$BUILD_IMAGE" >/dev/null 2>&1; then
  echo "ERROR: Build image '$BUILD_IMAGE' not found." >&2
  echo "" >&2
  echo "This wrapper requires a pre-built glibc 2.28 build image." >&2
  echo "The image must contain: Go, Node.js, git, gcc, make, tar, gzip, binutils." >&2
  echo "" >&2
  echo "To prepare the image:" >&2
  echo "  1. Build or pull a Rocky Linux 8 / UBI 8 based image with Go + Node.js" >&2
  echo "  2. Tag it as: $BUILD_IMAGE" >&2
  echo "  3. Or set LIGHTAI_BUILD_IMAGE to your image name" >&2
  exit 1
fi

echo "Build image '$BUILD_IMAGE' found."

# --- Step 3: Prepare persistent Go build caches ---
echo "[pre] Preparing Go build caches..."
CACHE_DIR="$PROJECT_DIR/.cache"
mkdir -p "$CACHE_DIR/go-mod" "$CACHE_DIR/go-build"
echo "  go-mod:  $CACHE_DIR/go-mod  -> /go/pkg/mod"
echo "  go-build: $CACHE_DIR/go-build -> /go-cache"

# --- Step 4: Clean host artifacts ---
echo "[pre] Cleaning host build artifacts..."
rm -rf "$PROJECT_DIR/bin/lightai-server" "$PROJECT_DIR/bin/lightai-agent" 2>/dev/null || true
rm -rf "$PROJECT_DIR/web/dist" 2>/dev/null || true
echo "  OK"

# --- Step 5: Run package-release.sh inside container ---
echo "[run] Executing scripts/package-release.sh in container..."
echo ""

# Build the docker run command with UID/GID matching host user to avoid
# root-owned output files.
HOST_UID=$(id -u 2>/dev/null || echo "")
HOST_GID=$(id -g 2>/dev/null || echo "")
USER_ARGS=""
if [ -n "$HOST_UID" ] && [ -n "$HOST_GID" ] && [ "$HOST_UID" != "0" ]; then
  USER_ARGS="--user ${HOST_UID}:${HOST_GID}"
fi

docker run --rm \
  $USER_ARGS \
  -v "$PROJECT_DIR:$CONTAINER_WORKDIR" \
  -v "$CACHE_DIR/go-mod:/go/pkg/mod" \
  -v "$CACHE_DIR/go-build:/go-cache" \
  -w "$CONTAINER_WORKDIR" \
  -e HOME=/tmp \
  -e GOPATH=/go \
  -e GOMODCACHE=/go/pkg/mod \
  -e GOCACHE=/go-cache \
  "$BUILD_IMAGE" \
  /bin/sh -c "cd $CONTAINER_WORKDIR && ./scripts/package-release.sh $PACKAGE_ARGS"

echo ""
echo "=== Container build complete ==="
echo "Release artifacts: $PROJECT_DIR/dist/"

# --- Step 6: glibc compatibility check ---
if $DO_GLIBC_CHECK; then
  echo ""
  echo "[check] Running glibc ABI compatibility check..."
  if [ -x "$PROJECT_DIR/scripts/check-glibc-compat.sh" ]; then
    "$PROJECT_DIR/scripts/check-glibc-compat.sh" "$PROJECT_DIR/dist" || {
      echo ""
      echo "=== GLIBC CHECK FAILED ==="
      echo "The release contains binaries that require GLIBC >= 2.29."
      echo "This means the build did NOT use the glibc 2.28 container correctly."
      echo "Check that:"
      echo "  1. The build image '$BUILD_IMAGE' has glibc 2.28"
      echo "  2. No host binaries leaked into the container build"
      exit 1
    }
    echo "  glibc check passed: all ELF binaries compatible with glibc <= 2.28"
  else
    echo "  WARNING: check-glibc-compat.sh not found, skipping verification"
  fi
else
  echo ""
  echo "[skip] glibc check disabled (--no-glibc-check)"
fi

echo ""
echo "=== Done ==="
echo "Release package and checksums are in dist/"
ls -lh "$PROJECT_DIR"/dist/*.tar.gz "$PROJECT_DIR"/dist/*.sha256 2>/dev/null || true
