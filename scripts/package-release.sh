#!/bin/sh
# LightAI Go - Release Package Builder
# Generates dist/lightai-go-<version>-linux-amd64.tar.gz
# Requires: third_party/observability/ (run prepare-observability-binaries.sh first)
#
# Version management:
#   scripts/package-release.sh --version 0.1.7
#   scripts/package-release.sh --bump patch
#   scripts/package-release.sh --bump minor
#   scripts/package-release.sh --bump major
#   scripts/package-release.sh --no-bump          # re-package current version
#   scripts/package-release.sh --dry-run          # preview version only
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

# --- Parse arguments ---
EXPLICIT_VERSION=""
BUMP=""
DRY_RUN=false
NO_BUMP=false
WITH_OBSERVABILITY=true

while [ $# -gt 0 ]; do
  case "$1" in
    --version) EXPLICIT_VERSION="$2"; shift 2 ;;
    --bump) BUMP="$2"; shift 2 ;;
    --no-bump) NO_BUMP=true; shift ;;
    --dry-run) DRY_RUN=true; shift ;;
    --without-observability) WITH_OBSERVABILITY=false; shift ;;
    --help|-h)
      echo "Usage: $0 [--version <ver>] [--bump patch|minor|major] [--no-bump] [--dry-run] [--without-observability]"
      echo ""
      echo "  --version <ver>        Use explicit version (e.g. 0.1.7)"
      echo "  --bump patch|minor|major  Auto-increment version"
      echo "  --no-bump              Re-package current version without bumping"
      echo "  --dry-run              Show what version would be generated"
      echo "  --without-observability  Skip bundling Prometheus/Grafana"
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# --- Determine version ---
CURRENT_VERSION=$(head -1 VERSION 2>/dev/null | tr -d '[:space:]' || echo "0.1.0")

if [ -n "$EXPLICIT_VERSION" ]; then
  VERSION="$EXPLICIT_VERSION"
elif [ -n "$BUMP" ]; then
  # Parse semver components.
  MAJOR="${CURRENT_VERSION%%.*}"
  REST="${CURRENT_VERSION#*.}"
  MINOR="${REST%%.*}"
  PATCH="${REST#*.}"
  case "$BUMP" in
    major)
      MAJOR=$((MAJOR + 1))
      MINOR=0
      PATCH=0
      ;;
    minor)
      MINOR=$((MINOR + 1))
      PATCH=0
      ;;
    patch)
      PATCH=$((PATCH + 1))
      ;;
    *)
      echo "ERROR: invalid bump type '$BUMP'. Use patch, minor, or major." >&2
      exit 1
      ;;
  esac
  VERSION="${MAJOR}.${MINOR}.${PATCH}"
elif $NO_BUMP; then
  VERSION="$CURRENT_VERSION"
else
  # Default: bump patch.
  MAJOR="${CURRENT_VERSION%%.*}"
  REST="${CURRENT_VERSION#*.}"
  MINOR="${REST%%.*}"
  PATCH="${REST#*.}"
  PATCH=$((PATCH + 1))
  VERSION="${MAJOR}.${MINOR}.${PATCH}"
fi

# --- Dry run ---
if $DRY_RUN; then
  echo "Current version: $CURRENT_VERSION"
  echo "Would generate:  $VERSION"
  echo "Mode:            $([ -n "$EXPLICIT_VERSION" ] && echo "explicit" || [ -n "$BUMP" ] && echo "bump $BUMP" || $NO_BUMP && echo "no-bump" || echo "default (bump patch)")"
  exit 0
fi

# --- Build metadata ---
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -Iseconds)
GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' || echo "unknown")
NODE_VERSION=$(node --version 2>/dev/null || echo "unknown")

# Observability versions.
PROMETHEUS_VERSION="${PROMETHEUS_VERSION:-3.12.0}"
GRAFANA_VERSION="${GRAFANA_VERSION:-13.0.2}"

ARCH="linux-amd64"
RELEASE_NAME="lightai-go-${VERSION}-${ARCH}"
BUILD_DIR="dist/${RELEASE_NAME}"
TARBALL="dist/${RELEASE_NAME}.tar.gz"

echo "=== LightAI Go Release Builder ==="
echo "Version:     $VERSION"
echo "Commit:      $COMMIT"
echo "Build time:  $BUILD_TIME"
echo "Arch:        $ARCH"
echo "Prometheus:  $PROMETHEUS_VERSION"
echo "Grafana:     $GRAFANA_VERSION"
echo "Observability: $WITH_OBSERVABILITY"
echo ""

# --- LDFLAGS for version injection ---
LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X lightai-go/internal/common/version.Version=$VERSION"
LDFLAGS="$LDFLAGS -X lightai-go/internal/common/version.GitCommit=$COMMIT"
LDFLAGS="$LDFLAGS -X lightai-go/internal/common/version.BuildTime=$BUILD_TIME"

# Clean previous build artifacts to avoid stale binaries.
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/bin"
mkdir -p "$BUILD_DIR/configs"
mkdir -p "$BUILD_DIR/configs/observability"
mkdir -p "$BUILD_DIR/deploy"
mkdir -p "$BUILD_DIR/deploy/observability"
mkdir -p "$BUILD_DIR/deploy/collectors"
mkdir -p "$BUILD_DIR/scripts"
mkdir -p "$BUILD_DIR/logs"
mkdir -p "$BUILD_DIR/data/prometheus"
mkdir -p "$BUILD_DIR/data/grafana"
mkdir -p "$BUILD_DIR/run"
mkdir -p "$BUILD_DIR/runtime"

# --- Step 1: Check observability binaries (fail fast) ---
echo "[1/8] Checking observability binaries..."
THIRD="$PROJECT_DIR/third_party/observability"
if $WITH_OBSERVABILITY; then
  if [ ! -x "$THIRD/prometheus/prometheus" ]; then
    echo "  ERROR: Prometheus binary not found at $THIRD/prometheus/prometheus"
    echo "  Run: ./scripts/prepare-observability-binaries.sh --download"
    echo "  Or:  ./scripts/package-release.sh --without-observability"
    exit 1
  fi
  if [ ! -d "$THIRD/grafana/bin" ]; then
    echo "  ERROR: Grafana not found at $THIRD/grafana/bin"
    echo "  Run: ./scripts/prepare-observability-binaries.sh --download"
    exit 1
  fi
  echo "  Prometheus: $THIRD/prometheus/prometheus"
  echo "  Grafana:    $THIRD/grafana/bin/"
  echo "  OK"
else
  echo "  Skipped (--without-observability)"
fi

# --- Step 2: Build Web FIRST (P0-002 fix) ---
echo "[2/8] Building Web assets..."
if [ -d web ]; then
  # Clean old web/dist to ensure fresh build.
  rm -rf web/dist
  (cd web && npm ci --silent 2>/dev/null || npm install --silent 2>/dev/null)
  (cd web && npm run build 2>/dev/null)
  if [ ! -d web/dist ]; then
    echo "  ERROR: web/dist not created after build" >&2
    exit 1
  fi
fi
echo "  OK"

# --- Step 3: Build Go binaries (AFTER Web) ---
echo "[3/8] Building Go binaries..."
# Clean any host-compiled binaries to avoid stale artifacts.
rm -f bin/lightai-server bin/lightai-agent
go build -tags web -ldflags "$LDFLAGS" -o "$BUILD_DIR/bin/lightai-server" ./cmd/server
go build -ldflags "$LDFLAGS" -o "$BUILD_DIR/bin/lightai-agent" ./cmd/agent
echo "  OK"

# --- Step 4: Bundled observability binaries ---
PROMETHEUS_BIN="$THIRD/prometheus/prometheus"
GRAFANA_DIR="$THIRD/grafana"
if $WITH_OBSERVABILITY; then
  echo "[4/8] Bundling observability binaries..."
  echo "  Observability input: $THIRD"
  if [ ! -x "$PROMETHEUS_BIN" ]; then
    echo "  ERROR: Prometheus binary not found at $PROMETHEUS_BIN"
    exit 1
  fi
  if [ ! -d "$GRAFANA_DIR/bin" ]; then
    echo "  ERROR: Grafana not found at $GRAFANA_DIR/bin"
    exit 1
  fi
  install -m 0755 "$PROMETHEUS_BIN" "$BUILD_DIR/bin/prometheus"
  cp -a "$GRAFANA_DIR" "$BUILD_DIR/bin/grafana"
  echo "  Observability bundled: bin/prometheus bin/grafana/"
  echo "  OK"
else
  echo "[4/8] Skipping observability binaries (--without-observability)"
fi

# --- Step 5: Copy configs, collectors, scripts ---
echo "[5/8] Copying configs and scripts..."
for d in "$BUILD_DIR/configs" "$BUILD_DIR/configs/observability" "$BUILD_DIR/deploy" "$BUILD_DIR/scripts"; do
  if [ ! -d "$d" ]; then
    echo "  ERROR: directory missing: $d" >&2
    exit 1
  fi
done
cp configs/server.release.yaml "$BUILD_DIR/configs/"
cp configs/agent.yaml "$BUILD_DIR/configs/"
cp configs/agent.metax.yaml "$BUILD_DIR/configs/"
cp configs/agent.nvidia.yaml "$BUILD_DIR/configs/"
cp configs/observability/prometheus.yml "$BUILD_DIR/configs/observability/" 2>/dev/null || true
cp configs/observability/grafana.ini "$BUILD_DIR/configs/observability/" 2>/dev/null || true
# Copy grafana.env as a template (no default password that works in production).
cp configs/observability/grafana.env "$BUILD_DIR/configs/observability/" 2>/dev/null || true
cp -r configs/templates "$BUILD_DIR/configs/" 2>/dev/null || true
cp -r deploy/collectors "$BUILD_DIR/deploy/" 2>/dev/null || true
cp -r deploy/observability "$BUILD_DIR/deploy/" 2>/dev/null || true
cp scripts/start-server.sh scripts/start-agent.sh "$BUILD_DIR/scripts/"
cp scripts/stop-server.sh scripts/stop-agent.sh "$BUILD_DIR/scripts/"
cp scripts/start-all.sh scripts/stop-all.sh "$BUILD_DIR/scripts/"
cp scripts/start-observability.sh scripts/stop-observability.sh "$BUILD_DIR/scripts/"
cp scripts/reset-password.sh scripts/reset-grafana-password.sh scripts/reset-agent-identity.sh "$BUILD_DIR/scripts/"
cp scripts/status.sh scripts/verify-local.sh "$BUILD_DIR/scripts/"
cp scripts/collect-logs.sh scripts/collect-debug-bundle.sh "$BUILD_DIR/scripts/"
cp scripts/apply-patch.sh scripts/diagnose-model-runtime-spec.sh "$BUILD_DIR/scripts/"
cp scripts/smoke-model-backends.sh "$BUILD_DIR/scripts/"
# Copy backend-catalog configs (backend definitions, version schemas, runtime templates, help docs).
mkdir -p "$BUILD_DIR/configs/backend-catalog"
cp -r configs/backend-catalog/* "$BUILD_DIR/configs/backend-catalog/"
# Copy optional catalog overrides if present.
mkdir -p "$BUILD_DIR/configs/backend-catalog.d"
cp -r configs/backend-catalog.d/* "$BUILD_DIR/configs/backend-catalog.d/" 2>/dev/null || true
	# Copy config-registry (config item definitions — required for seed/migration).
	cp -r configs/config-registry "$BUILD_DIR/configs/"
# Copy bootstrap tooling (scripts, profiles, export helper, docs).
cp scripts/lightai-bootstrap.sh "$BUILD_DIR/scripts/"
mkdir -p "$BUILD_DIR/scripts/lib"
cp scripts/lib/bootstrap-export.py "$BUILD_DIR/scripts/lib/"
mkdir -p "$BUILD_DIR/configs/bootstrap"
cp -r configs/bootstrap/*.yaml "$BUILD_DIR/configs/bootstrap/" 2>/dev/null || true
mkdir -p "$BUILD_DIR/docs/engineering/bootstrap"
cp docs/engineering/bootstrap/lightai-bootstrap.md "$BUILD_DIR/docs/engineering/bootstrap/" 2>/dev/null || true
chmod +x "$BUILD_DIR"/scripts/*.sh
echo "  OK"


# --- Step 5b: Fail-fast validation of release directory ---
echo "[5b/8] Validating release directory..."
FAIL_FAST=false
check_file() {
  if [ ! -e "$BUILD_DIR/$1" ]; then
    echo "  ERROR: missing required file/dir: $1" >&2
    FAIL_FAST=true
  fi
}
check_dir() {
  if [ ! -d "$BUILD_DIR/$1" ]; then
    echo "  ERROR: missing required directory: $1" >&2
    FAIL_FAST=true
  fi
}
check_file "bin/lightai-server"
check_file "bin/lightai-agent"
check_file "configs/server.release.yaml"
check_file "configs/agent.yaml"
check_dir  "configs/config-registry"
check_dir  "configs/backend-catalog"
check_file "scripts/start-all.sh"
check_file "scripts/start-server.sh"
check_file "scripts/start-agent.sh"
check_file "scripts/lightai-bootstrap.sh"
if $FAIL_FAST; then
  echo "  FAIL: release directory validation failed. Aborting." >&2
  exit 1
fi
echo "  OK"
# --- Step 6: Copy README and licenses ---
echo "[6/8] Copying docs..."
cp README-RELEASE.md "$BUILD_DIR/"
if [ -f "$THIRD/prometheus/LICENSE" ]; then
  mkdir -p "$BUILD_DIR/LICENSES/prometheus"
  cp "$THIRD/prometheus/LICENSE" "$BUILD_DIR/LICENSES/prometheus/"
fi
if [ -f "$THIRD/grafana/LICENSE" ]; then
  mkdir -p "$BUILD_DIR/LICENSES/grafana"
  cp "$THIRD/grafana/LICENSE" "$BUILD_DIR/LICENSES/grafana/"
fi
echo "  OK"

# --- Step 7: Write VERSION and manifests ---
echo "[7/8] Writing VERSION and manifests..."
echo "$VERSION" > "$BUILD_DIR/VERSION"

# release-manifest.json: full metadata.
cat > "$BUILD_DIR/release-manifest.json" << EOF
{
  "product": "lightai-go",
  "version": "$VERSION",
  "release_name": "RC1",
  "arch": "$ARCH",
  "created_at": "$BUILD_TIME",
  "package_type": "full",
  "git_commit": "$COMMIT",
  "go_version": "$GO_VERSION",
  "node_version": "$NODE_VERSION",
  "prometheus_version": "$PROMETHEUS_VERSION",
  "grafana_version": "$GRAFANA_VERSION",
  "build_os": "linux",
  "build_arch": "amd64",
  "glibc_baseline": "2.28"
}
EOF

# Generate MANIFEST.sha256 (exclude runtime dirs).
MANIFEST="$BUILD_DIR/MANIFEST.sha256"
: > "$MANIFEST"
(cd "$BUILD_DIR" && find . -type f \
  ! -path './data/*' ! -path './logs/*' ! -path './run/*' ! -path './runtime/*' \
  ! -path './data/prometheus/*' ! -path './data/grafana/*' \
  ! -name MANIFEST.sha256 \
  | sort | while IFS= read -r f; do
    [ -f "$f" ] && sha256sum "$f" 2>/dev/null | awk '{print "file " $1 " " $2}'
  done) >> "$MANIFEST"
# Record symlinks too.
(cd "$BUILD_DIR" && find . -type l \
  ! -path './data/*' ! -path './logs/*' ! -path './run/*' ! -path './runtime/*' \
  | sort | while IFS= read -r f; do
    target=$(readlink "$f" 2>/dev/null || echo "")
    echo "symlink $target $f"
  done) >> "$MANIFEST"
echo "  OK"

# --- Step 8: Build tarball and checksums ---
echo "[8/8] Creating tarball..."
mkdir -p dist
rm -f "$TARBALL"
tar -czf "$TARBALL" -C dist "$RELEASE_NAME"

# Generate tarball SHA256 checksum (P2-010).
sha256sum "$TARBALL" | awk '{print $1}' > "${TARBALL}.sha256"
echo "  Checksum: ${TARBALL}.sha256"
echo "  OK"

# --- Write VERSION back to project root only on success ---
echo "$VERSION" > VERSION
echo "  Project VERSION updated to $VERSION"

# --- Quick verification ---
echo ""
echo "--- Package Contents ---"
echo "  bin/lightai-server: $(tar -tzf "$TARBALL" | grep -c 'bin/lightai-server$')"
echo "  bin/lightai-agent:  $(tar -tzf "$TARBALL" | grep -c 'bin/lightai-agent$')"
if $WITH_OBSERVABILITY; then
  echo "  bin/prometheus:     $(tar -tzf "$TARBALL" | grep -c 'bin/prometheus$')"
  echo "  bin/grafana/:       $(tar -tzf "$TARBALL" | grep -c 'bin/grafana/')"
fi
echo "  configs:            $(tar -tzf "$TARBALL" | grep -c 'configs/')"
echo "  scripts:            $(tar -tzf "$TARBALL" | grep -c 'scripts/')"
echo "  VERSION:            $(tar -tzf "$TARBALL" | grep -c 'VERSION$')"

echo ""
echo "Release: $TARBALL"
echo "Size:    $(du -h "$TARBALL" | cut -f1)"
echo "SHA256:  $(cat "${TARBALL}.sha256")"
echo ""
echo "Deploy:"
echo "  tar -xzf ${RELEASE_NAME}.tar.gz && cd ${RELEASE_NAME}"
echo "  export LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD='...'  # initial admin password for clean DB"
echo "  export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='...'     # target admin password after bootstrap (also accepted for backward compat)"
echo "  export LIGHTAI_GRAFANA_ADMIN_PASSWORD='...'"
echo "  ./scripts/start-server.sh"
echo "  ./scripts/start-agent.sh metax   # or nvidia"
echo "  ./scripts/start-observability.sh"
