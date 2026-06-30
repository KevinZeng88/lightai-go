# Phase 1 - ConfigEdit Object Model Contract

Date: 2026-07-01

## Final Contract

ConfigEdit template inspection now exposes both component template data and materialized field data.

Template-level contract:

- `template_id`, `kind`, `version`
- `source`: `built_in`, `local`, or `catalog_materialized`
- `path`
- `applies_to.backend`, `applies_to.backend_versions`, `applies_to.runtime_kind`, `applies_to.vendors`
- `metadata`: display name, scope, backend/runtime IDs, vendor, source metadata, field/component counts
- `views.default_view`, `views.supported_views`
- `layers`: backend_runtime, node_backend_runtime, deployment, deployment_override
- `sections`
- `fields`
- `components`

Field-level contract:

- identity: `key`, `internal_key`, `component_key`, `path`
- presentation: `label`, `label_i18n_key`, `help_i18n_key`, `section`, `tier`, `view`, `order`
- input contract: `type`, `widget`
- safety/state: `enabled`, `risk`
- provenance/effect: `source`, `effects`

## Boundary

The runtime template remains the deployable backend/runtime object. The ConfigEdit component/template is the parameter presentation and materialization object used to render, validate, and preview structured parameters. Raw JSON remains diagnostic/developer representation, not the sole entry point for real parameters.

