# Validation and Tests

## Required command set

Run all:

```bash
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm test
cd web && npm run build
```

## Backend tests

### Runtime catalog / taxonomy

Add or update tests to assert:

1. No normal user-facing `Optional devices` field.
2. Devices widget uses `host_device_path`, `container_device_path`, `permissions`.
3. Devices does not expose `readonly`.
4. NVIDIA catalog devices default disabled/empty.
5. MetaX catalog devices default enabled and includes:

```text
/dev/mxcd
/dev/dri
/dev/mem
```

6. MetaX catalog includes:

```text
privileged=true
cap_add SYS_PTRACE
security_options seccomp=unconfined, apparmor=unconfined
network_mode=host
shm_size=100gb
ulimit memlock=-1
group_add video
```

7. Model mount remains readonly by default.
8. Additional volumes and Model mount remain separate.

### Device warning-only behavior

Add test:

```text
configured device path missing -> warning exists -> preflight/runplan remains deployable
```

The test should fail if missing device makes `can_run=false` or returns blocking error.

### Runtime type resolution

Add tests for:

1. Creating/preflighting a Docker deployment resolves `runtime_type=docker`.
2. Empty `config_overrides.runtime_type` cannot override the snapshot runtime_type.
3. Deployment snapshot retains runtime_type from selected NBR/runtime template.
4. RunPlan preview and start resolver use the same runtime_type source.

Expected failure to prevent:

```text
[resolve_error] unsupported runtime_type: (only docker is supported)
```

### Port canonicalization

Add tests for:

1. `service.container_port` is visible/canonical.
2. `model_runtime.port` is not a normal required deployment field.
3. Host network displays no Docker host-port requirement.
4. Bridge/default network displays configured/auto/unconfigured host port state.

### Parameter taxonomy

Add tests for:

1. `model_runtime.model`, `model_runtime.host`, `model_runtime.port`, `model_runtime.download_dir` do not appear in ordinary deployment override section.
2. Advanced/expert params are not required by default.
3. Custom args are available and included in RunPlan preview.

## Frontend Vitest tests

### ConfigEditView render tests

Update:

```text
web/src/components/config/__tests__/ConfigEditView.render.test.ts
```

Assert:

1. Devices shows device labels.
2. Devices does not show readonly.
3. Additional volumes shows readonly.
4. Model mount shows readonly and help text.
5. Optional devices is absent.
6. MetaX device defaults render correctly when supplied by config view.
7. Raw duplicated device/volume JSON does not show as normal fields.

### BackendRuntimesPage integration tests

Update:

```text
web/src/pages/__tests__/BackendRuntimesPage.integration.test.ts
```

Assert:

1. Runtime list has View / Edit / Copy as user config actions.
2. View opens readonly detail.
3. Edit opens edit mode directly.
4. Copy as user config still works.
5. Row click opens readonly detail.
6. Builtin direct edit follows current disabled/hidden rule.

### ModelDeploymentsPage integration tests

Add or update:

```text
web/src/pages/__tests__/ModelDeploymentsPage.integration.test.ts
```

Assert:

1. New deployment opens at first step with empty draft.
2. Cancel resets draft.
3. Save success resets draft.
4. Drawer close resets draft.
5. Save failure does not pollute next New after close/cancel.
6. Existing deployment detail default is structured.
7. Edit entry exists.
8. Raw config/source metadata sections are collapsed by default.
9. `service.container_port` appears.
10. `model_runtime.port required empty` does not appear.
11. Host network shows host port as not applicable.
12. Advanced/expert fields are collapsed or custom args, not ordinary required fields.

## Manual verification after DB rebuild

Because catalog seed changes affect visible templates, manual verification should use a clean DB:

```bash
# stop server/agent first
rm -f /tmp/lightai/data/lightai.db
```

Then verify:

1. Runtime templates list actions show View / Edit / Copy as user config.
2. NVIDIA copied runtime config has Devices disabled/empty.
3. MetaX copied runtime config has Devices enabled with `/dev/mxcd`, `/dev/dri`, `/dev/mem`.
4. Devices UI has permissions `rwm`, no readonly.
5. Model mount remains readonly.
6. Additional volumes are separate.
7. New deployment always starts clean after save/cancel/close.
8. Existing deployment detail is structured and has Edit.
9. Raw JSON sections are collapsed.
10. vLLM Docker deployment RunPlan preview resolves runtime_type=docker.
11. Container listen port / host port display is explicit.
12. Ordinary deployment form does not show `model_runtime.port required empty`.

## Completion report format

Claude must output:

```text
1. Root causes
2. Modified files
3. Runtime Devices / Model mount / Additional volumes final semantics
4. MetaX catalog final summary
5. Runtime template list action behavior
6. Model deployment wizard/detail behavior
7. runtime_type fix evidence
8. Port and parameter taxonomy fix summary
9. Tests added/updated
10. Command results
11. Commit id
12. Push result
13. git status --short
```
