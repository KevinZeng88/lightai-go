# Phase 5 - Grouping and Ordering

Date: 2026-07-01

## Shared Ordering

`web/src/utils/configEditView.ts` now groups ConfigEdit fields into:

1. 已启用参数
2. 常用参数
3. 高级参数
4. 专家参数

Within each group fields sort by:

1. section rank
2. display order
3. field key/path

Section rank follows the requested product order:

`model`, `runtime`, `resource`, `service`, `health`, `mount`, `env`, `docker`, `security`, `raw`.

Existing local section keys are mapped into that order, including `model_serving`, `backend_runtime`, `container_resources`, `devices_mounts`, `environment`, `health_check`, `security_high_risk`, and `advanced_raw`.

Expert/security/raw fields are classified before enabled placement so diagnostic JSON and high-risk Docker/security options remain in 专家参数 even when their values are enabled.

## High-Risk Fields

Security-sensitive Docker fields such as `privileged` and `security_options` are projected into expert/high-risk grouping through `security_high_risk`, `view=security/developer`, and `risk=high`.

## Applied Consumers

Because `ConfigEditView` and `buildConfigEditPatch` use the shared `sortedSections`/`sortedFields` utilities, the ordering applies to runtime templates, node backend runtimes, deployments, deployment overrides, and other ConfigEdit consumers using the shared view.
