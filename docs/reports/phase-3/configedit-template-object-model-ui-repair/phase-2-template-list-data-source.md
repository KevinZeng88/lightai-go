# Phase 2 - Template List Data Source

Date: 2026-07-01

## Root Cause

`/api/v1/config-edit/templates` only loaded explicit YAML files from `configs/configedit-templates/{builtin,local}`. It did not expose templates derived from the ConfigEdit registry/materialization path, so the page could be empty or incomplete when explicit component YAML was missing, invalid, or not exhaustive.

## Fix

`HandleListConfigEditTemplates` and `HandleGetConfigEditTemplate` now use a merged registry:

- explicit built-in/local component templates
- materialized templates generated from `configs/config-registry` + `configs/backend-catalog`

The generated templates are built from `catalog.MaterializeBackendRuntime` and `configedit.ProjectConfigSetToEditView`, so vLLM, SGLang, llama.cpp, Docker options, resources, health checks, mounts, env, and fallback-materialized fields come from the same ConfigEdit object model as runtime editing.

## Example Evidence

The API test `TestConfigEditTemplatesListIncludesMaterializedRegistryData` verifies:

- `catalog_materialized` templates exist
- backends include `vllm`, `sglang`, `llamacpp`
- backend runtime args include `model_runtime.*`
- high-risk Docker field `docker.privileged` is structured with `section=security_high_risk`, `tier=expert`, `risk=high`

