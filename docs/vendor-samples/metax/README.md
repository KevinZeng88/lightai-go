> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# MetaX Vendor Samples

Sanitized mx-smi output samples for MetaX GPU Collector testing.

## Files

- `mx-smi-list.txt` — `mx-smi -L` output showing 8 GPUs.
- `mx-smi-default.txt` — `mx-smi` default summary table.

## Format Support

Current scripts target mx-smi 2.3.1 output format.

## Desensitization

UUIDs and PCI bus IDs are from real hardware but are hardware identifiers,
not personal data. They are preserved for format validation.

## Scripts

- `deploy/collectors/gpu/metax/discover.sh` — device discovery.
- `deploy/collectors/gpu/metax/metrics.sh` — real-time metrics.
