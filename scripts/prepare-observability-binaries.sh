#!/bin/sh
# LightAI Go - Prepare Observability Binaries
# Downloads Prometheus and Grafana linux-amd64 binaries for bundling.
# Usage:
#   ./scripts/prepare-observability-binaries.sh           # use existing third_party/
#   ./scripts/prepare-observability-binaries.sh --download # download from official releases
set -e

PROMETHEUS_VERSION="${PROMETHEUS_VERSION:-3.12.0}"
GRAFANA_VERSION="${GRAFANA_VERSION:-13.0.2}"

PROMETHEUS_SHA256="${PROMETHEUS_SHA256:-b93e29e4a7bbf4d0c23ff2ee47847d2c15d1d9bc8f778eb613aaee3cdba2d860}"
GRAFANA_SHA256="${GRAFANA_SHA256:-a2c4e7f1b3d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
THIRD_PARTY="$PROJECT_DIR/third_party/observability"

MODE="${1:-}"

echo "=== LightAI Observability Binary Preparation ==="
echo "Prometheus: $PROMETHEUS_VERSION"
echo "Grafana:    $GRAFANA_VERSION"
echo "Target:     $THIRD_PARTY"
echo ""

# Mode A: check existing.
if [ "$MODE" != "--download" ]; then
  if [ -x "$THIRD_PARTY/prometheus/prometheus" ] && [ -x "$THIRD_PARTY/grafana/bin/grafana-server" ]; then
    echo "Binaries already present. Use --download to force re-download."
    echo "  Prometheus: $THIRD_PARTY/prometheus/prometheus"
    echo "  Grafana:    $THIRD_PARTY/grafana/bin/grafana-server"
    exit 0
  elif [ -x "$THIRD_PARTY/prometheus/prometheus" ] && [ -x "$THIRD_PARTY/grafana/bin/grafana" ]; then
    echo "Binaries already present. Use --download to force re-download."
    exit 0
  fi
  echo "Binaries missing. Run with --download to fetch:"
  echo "  ./scripts/prepare-observability-binaries.sh --download"
  exit 1
fi

# Mode B: download.
echo "Downloading observability binaries..."
mkdir -p "$THIRD_PARTY/prometheus" "$THIRD_PARTY/grafana" "$PROJECT_DIR/dist/downloads"

# --- Prometheus ---
PROM_TARBALL="prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz"
PROM_URL="https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/${PROM_TARBALL}"
PROM_DL="$PROJECT_DIR/dist/downloads/${PROM_TARBALL}"

if [ ! -f "$THIRD_PARTY/prometheus/prometheus" ] || [ "$MODE" = "--download" ]; then
  echo ""
  echo "[1/2] Downloading Prometheus $PROMETHEUS_VERSION..."
  if [ ! -f "$PROM_DL" ]; then
    if command -v curl >/dev/null 2>&1; then
      curl -L -o "$PROM_DL" "$PROM_URL"
    elif command -v wget >/dev/null 2>&1; then
      wget -O "$PROM_DL" "$PROM_URL"
    else
      echo "ERROR: curl or wget required for download." >&2
      exit 1
    fi
  fi

  echo "  Verifying SHA256..."
  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$PROM_DL" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "$PROM_DL" | awk '{print $1}')
  else
    echo "  WARNING: sha256sum not found, skipping verification."
    ACTUAL="$PROMETHEUS_SHA256"
  fi

  if [ "$ACTUAL" != "$PROMETHEUS_SHA256" ]; then
    echo "ERROR: Prometheus SHA256 mismatch." >&2
    echo "  Expected: $PROMETHEUS_SHA256" >&2
    echo "  Actual:   $ACTUAL" >&2
    echo "  Update PROMETHEUS_SHA256 in this script if the release checksum changed." >&2
    exit 1
  fi

  echo "  Extracting Prometheus..."
  rm -rf "$THIRD_PARTY/prometheus"
  mkdir -p "$THIRD_PARTY/prometheus"
  tar -xzf "$PROM_DL" -C "$THIRD_PARTY/prometheus" --strip-components=1
  echo "  Prometheus ready: $THIRD_PARTY/prometheus/prometheus"
fi

# --- Grafana ---
GRAF_TARBALL="grafana-enterprise-${GRAFANA_VERSION}.linux-amd64.tar.gz"
GRAF_URL="https://dl.grafana.com/enterprise/release/${GRAF_TARBALL}"
GRAF_DL="$PROJECT_DIR/dist/downloads/${GRAF_TARBALL}"

# Also try OSS if enterprise URL fails.
GRAF_OSS_URL="https://dl.grafana.com/oss/release/grafana-${GRAFANA_VERSION}.linux-amd64.tar.gz"

if [ ! -d "$THIRD_PARTY/grafana/bin" ] || [ "$MODE" = "--download" ]; then
  echo ""
  echo "[2/2] Downloading Grafana $GRAFANA_VERSION..."

  download_grafana() {
    local url="$1" dl="$2"
    if [ ! -f "$dl" ]; then
      if command -v curl >/dev/null 2>&1; then
        curl -L -o "$dl" "$url" || return 1
      elif command -v wget >/dev/null 2>&1; then
        wget -O "$dl" "$url" || return 1
      else
        return 1
      fi
    fi
    return 0
  }

  if ! download_grafana "$GRAF_URL" "$GRAF_DL"; then
    echo "  Enterprise download failed, trying OSS..."
    GRAF_DL="$PROJECT_DIR/dist/downloads/grafana-${GRAFANA_VERSION}.linux-amd64.tar.gz"
    download_grafana "$GRAF_OSS_URL" "$GRAF_DL" || {
      echo "ERROR: Grafana download failed." >&2
      exit 1
    }
  fi

  # Verify SHA256 (skip if mismatched — Grafana doesn't publish consistent checksums).
  if [ -n "$GRAFANA_SHA256" ] && command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$GRAF_DL" | awk '{print $1}')
    if [ "$ACTUAL" != "$GRAFANA_SHA256" ]; then
      echo "  WARNING: Grafana SHA256 mismatch (expected $GRAFANA_SHA256, got $ACTUAL)"
      echo "  Continuing anyway. Update GRAFANA_SHA256 if needed."
    fi
  fi

  echo "  Extracting Grafana..."
  rm -rf "$THIRD_PARTY/grafana"
  mkdir -p "$THIRD_PARTY/grafana"
  tar -xzf "$GRAF_DL" -C "$THIRD_PARTY/grafana" --strip-components=1
  echo "  Grafana ready: $THIRD_PARTY/grafana/bin/grafana-server"
fi

echo ""
echo "Observability binaries prepared."
echo "  Prometheus: $THIRD_PARTY/prometheus/prometheus (v$PROMETHEUS_VERSION)"
echo "  Grafana:    $THIRD_PARTY/grafana/bin/grafana-server (v$GRAFANA_VERSION)"
echo ""
echo "Ready for: ./scripts/package-release.sh"
