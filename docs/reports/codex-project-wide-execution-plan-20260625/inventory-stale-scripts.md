# Inventory — Stale Scripts

Scripts using legacy `backend_runtime_id` / `parameters_json` / `image_present` from request body:

$(cat /tmp/e2e-stale-contracts.txt 2>/dev/null || echo "None found")

## Classification
- **active-current**: scripts/e2e-model-runtime-param-trace.sh, lightai-bootstrap.sh, bootstrap-export.py
- **active-needs-repair**: scripts/e2e-matrix-verifier.sh, scripts/e2e-dryrun-parameter-matrix.sh, scripts/e2e-model-runtime-wizard-nvidia-api.sh, scripts/e2e-model-runtime-api.sh
- **hardware-only**: All E2E scripts requiring NVIDIA GPU / Docker / model files
