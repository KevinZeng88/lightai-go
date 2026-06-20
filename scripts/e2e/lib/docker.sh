#!/usr/bin/env bash
# Docker/GPU readiness helpers for controlled real-environment E2E.

if [ "${LIGHTAI_E2E_DOCKER_SH:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
LIGHTAI_E2E_DOCKER_SH=1

set -euo pipefail

E2E_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$E2E_DOCKER_DIR/env.sh"
source "$E2E_DOCKER_DIR/report.sh"

e2e_docker_ready() {
  docker info >/dev/null 2>&1
}

e2e_require_docker() {
  e2e_require_cmd docker
  e2e_docker_ready || e2e_die "Docker is not ready"
  e2e_report_event PASS "docker_ready" "$(docker version --format '{{.Server.Version}}' 2>/dev/null || true)"
}

e2e_require_gpu() {
  e2e_require_cmd nvidia-smi
  nvidia-smi >/dev/null 2>&1 || e2e_die "NVIDIA GPU is not ready"
  e2e_report_event PASS "gpu_ready" "$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)"
}

e2e_require_image() {
  local image="$1"
  docker image inspect "$image" >/dev/null 2>&1 || e2e_die "Docker image not found: $image"
  e2e_report_event PASS "image_ready" "$image"
}

e2e_require_path() {
  local path="$1"
  [ -e "$path" ] || e2e_die "model path not found: $path"
  e2e_report_event PASS "path_ready" "$path"
}
