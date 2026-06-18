#!/usr/bin/env bash
set -euo pipefail
# Matrix wrapper: runs llama.cpp, vLLM, SGLang with default + modified params.
# Uses existing proven E2E scripts with parameter overrides.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_ID="${RUN_ID:-$(date +%Y%m%d-%H%M%S)}"
VERIFY_BASE="${VERIFY_BASE:-docs/reports/model-runtime-node-wizard/e2e-matrix-${RUN_ID}}"
mkdir -p "$VERIFY_BASE"
MATRIX_EXIT=0

log() { printf '[%s] [matrix] %s\n' "$(date '+%H:%M:%S')" "$*"; }

run_one() {
  local label="$1" script="$2" deploy_params="${3:-}"
  local artifact_dir="$VERIFY_BASE/$label"
  mkdir -p "$artifact_dir"
  log "===== $label start ====="
  export DEPLOY_PARAMS="$deploy_params"
  export ARTIFACT_DIR="$artifact_dir"
  if bash "$script" 2>&1 | tee "$VERIFY_BASE/${label}.log"; then
    log "$label: PASS"
    echo "$label: PASS"
  else
    log "$label: FAIL"
    echo "$label: FAIL"
    MATRIX_EXIT=1
  fi
}

write_summary() {
  python3 - "$VERIFY_BASE" <<'PY'
import json
import pathlib
import sys

base = pathlib.Path(sys.argv[1])
rows = []
for log_path in sorted(base.glob("*.log")):
    label = log_path.stem
    if label in {"server-this-run", "agent-this-run"}:
        continue
    artifact_dir = base / label
    payload_path = artifact_dir / "deployment-request-payload.json"
    runplan_path = artifact_dir / "runplan.json"
    payload = {}
    runplan = {}
    if payload_path.exists():
        try:
            payload = json.loads(payload_path.read_text())
        except Exception as exc:
            payload = {"_parse_error": str(exc)}
    if runplan_path.exists():
        try:
            runplan = json.loads(runplan_path.read_text())
        except Exception as exc:
            runplan = {"_parse_error": str(exc)}
    status = "PASS" if "PASS:" in log_path.read_text(errors="ignore") else "FAIL"
    params = payload.get("parameters_json", {})
    docker_spec = runplan.get("docker_spec") or runplan.get("docker_create_spec") or runplan.get("resolved_docker_spec") or runplan.get("run_plan_json") or {}
    rows.append({
        "label": label,
        "status": status,
        "backend_runtime_id": payload.get("backend_runtime_id", ""),
        "parameters_json": params,
        "request_payload": str(payload_path) if payload_path.exists() else "",
        "runplan": str(runplan_path) if runplan_path.exists() else "",
        "docker_spec_summary": {
            "image": docker_spec.get("image") or docker_spec.get("image_ref") or "",
            "entrypoint": docker_spec.get("entrypoint") or docker_spec.get("Entrypoint") or [],
            "cmd": docker_spec.get("cmd") or docker_spec.get("command") or docker_spec.get("args") or docker_spec.get("Cmd") or [],
            "ports": docker_spec.get("ports") or docker_spec.get("port_bindings") or {"host_port": docker_spec.get("host_port"), "container_port": docker_spec.get("container_port")},
        },
        "param_assertion": "modified" in label and bool(params) or "default" in label,
    })

summary = {"matrix_dir": str(base), "results": rows}
(base / "matrix-summary.json").write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n")
lines = ["# E2E Matrix Summary", ""]
for row in rows:
    lines.append(f"- {row['label']}: {row['status']}; runtime={row['backend_runtime_id']}; params={json.dumps(row['parameters_json'], sort_keys=True)}; payload={row['request_payload']}; runplan={row['runplan']}")
(base / "matrix-summary.md").write_text("\n".join(lines) + "\n")
PY
}

# ── llama.cpp ──
run_one "llamacpp-default"     "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-llamacpp.sh" ""
run_one "llamacpp-modified"    "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-llamacpp.sh" '"--ctx-size":"2048","--n-gpu-layers":"-1"'

# ── vLLM ──
run_one "vllm-default"         "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-api.sh" ""
run_one "vllm-modified"        "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-api.sh" '"--max-model-len":"2048","--gpu-memory-utilization":"0.80"'

# ── SGLang ──
run_one "sglang-default"       "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-sglang.sh" ""
run_one "sglang-modified"      "$SCRIPT_DIR/e2e-model-runtime-wizard-nvidia-sglang.sh" '"--tp":"1"'

# ── Save logs ──
tail -n 5000 logs/lightai-server.log > "$VERIFY_BASE/server-this-run.log" 2>/dev/null || true
tail -n 5000 logs/lightai-agent.log > "$VERIFY_BASE/agent-this-run.log" 2>/dev/null || true
docker ps -a --format 'table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}' > "$VERIFY_BASE/docker-ps-after.txt" 2>/dev/null || true
write_summary

echo ""
echo "========== Matrix Summary =========="
grep -E ': (PASS|FAIL)$' "$VERIFY_BASE"/*.log 2>/dev/null || true
cat "$VERIFY_BASE/matrix-summary.md" 2>/dev/null || true
echo "Matrix exit=$MATRIX_EXIT"
exit $MATRIX_EXIT
