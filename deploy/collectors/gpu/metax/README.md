# MetaX GPU Collector

Phase 2C Deferred - requires real MetaX (沐曦) GPU hardware.

When hardware is available:
1. Confirm mx-smi tool availability, version, and machine-readable output format.
2. Adapt discover.sh to convert mx-smi output to LightAI GPU Collector Protocol.
3. Adapt metrics.sh similarly.
4. Save sanitized samples to `docs/vendor-samples/metax/`.
5. Enable the metax collector in `configs/agent.dev.yaml`.

No Go Agent code changes needed - only script adaptation.
