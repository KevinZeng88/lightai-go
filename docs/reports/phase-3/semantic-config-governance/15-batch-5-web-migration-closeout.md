# 15. Batch 5 Web Migration Closeout

> Status: PASS
> Scope: Web entrypoint migration away from page-private legacy semantic keys.
> Date: 2026-06-27

## Summary

Batch 5 removes the highest-impact web-side legacy key injection and tightens ordinary UI entrypoints so they no longer create `backend.common.*` or `backend.arg.*` user config keys.

## Implemented

| Requirement | Evidence |
| --- | --- |
| DeploymentWizard deletes `backend.common.served_model_name` injection | `DeploymentWizard.buildPayload()` no longer pushes `backend.common.served_model_name` into `config_overrides.parameter_values`. Served model name remains in `service_json` transition payload until API migration consumes semantic snapshot authority. |
| BackendsPage ordinary add parameter no longer defaults to `backend.arg.*` | Default parameter code changed to `model_runtime.custom_parameter`; placeholder no longer suggests `backend.arg.fake_new_param`. |
| BackendsPage rejects legacy user config keys | `addParameter()` rejects `backend.arg.*`, `backend.common.*`, `launcher.listen_host`, and `launcher.container_port`. |
| RuntimeParameterEditor leaves normal flow | Component is marked diagnostic/dev-only; active runtime/deployment pages continue to use `ConfigEditView`. Existing UI boundary tests assert active pages do not import it. |
| ConfigEditView remains common renderer | Batch 3 metadata/changed-only patch remains the shared renderer path for BackendRuntime, RunnerConfigs, NodeRuntime wizard and deployment override editor. |
| JsonViewer remains diagnostic | No raw JSON editor was introduced; existing JsonViewer usages remain read-only diagnostic display. |

## Validation

Commands run:

```bash
cd web && npm run build
cd web && npm test
```

Results:

```text
Web build completed successfully.
npm test: all test groups passed.
```

## Closeout State

No unresolved Batch 5 blocker remains.

`service_json` remains in request/response payloads as a transition carrier until Batch 6 and API closeout finish catalog/API cleanup. It is no longer used by the wizard to inject legacy backend common keys.
