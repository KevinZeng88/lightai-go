# Web Workflow Acceptance Checklist

Every row must end in one of:

- Supported
- Partially Supported
- Not Supported
- Not Verified

Before final closeout, no row may remain `Not Verified`. Any `Partially Supported` or `Not Supported` item must generate REVIEW-031+ or be merged into REVIEW-026/027/028 and then fixed.

| Area | Workflow / Check | Expected Result | Actual Result | Evidence | Status | Linked Review |
|---|---|---|---|---|---|---|
| Auth | Login | User can log in with initial/admin credentials | TBD | TBD | Not Verified | REVIEW-028 |
| Auth | Logout | User can log out cleanly | TBD | TBD | Not Verified | REVIEW-028 |
| Auth | Change password | User can change password and re-login | TBD | TBD | Not Verified | REVIEW-028 |
| Dashboard | Node/GPU/instance/health summary | Dashboard displays current state and errors clearly | TBD | TBD | Not Verified | REVIEW-028 |
| Nodes | List | Node list loads with status and tenant | TBD | TBD | Not Verified | REVIEW-028 |
| Nodes | Detail | Node detail shows host/address/version/driver/resource info | TBD | TBD | Not Verified | REVIEW-028 |
| Nodes | Tenant transfer | Authorized user can transfer node and GPUs consistently | TBD | TBD | Not Verified | REVIEW-008 |
| GPUs | List | GPU list shows memory/util/temp/health/tenant | TBD | TBD | Not Verified | REVIEW-028 |
| GPUs | Detail | GPU direct detail respects tenant and displays full metadata | TBD | TBD | Not Verified | REVIEW-002 |
| GPUs | Long name display | Long GPU names are readable and do not break layout | TBD | TBD | Not Verified | REVIEW-028 |
| Model Artifacts | Create | User can create artifact with validated metadata | TBD | TBD | Not Verified | REVIEW-027 |
| Model Artifacts | View | User can view artifact metadata | TBD | TBD | Not Verified | REVIEW-027 |
| Model Artifacts | Edit | User can edit supported metadata | TBD | TBD | Not Verified | REVIEW-027 |
| Model Artifacts | Delete | User can delete where allowed with safe confirmation | TBD | TBD | Not Verified | REVIEW-027 |
| Model Artifacts | i18n keys | No raw artifacts.* keys appear | TBD | TBD | Not Verified | REVIEW-027 |
| Model Artifacts | Format options + custom | gguf/safetensors/pt/onnx/other + custom supported | TBD | TBD | Not Verified | REVIEW-027 |
| Model Artifacts | Task type options + custom | chat/completion/embedding/rerank/image/audio/other + custom supported | TBD | TBD | Not Verified | REVIEW-027 |
| Model Artifacts | Architecture options + custom | qwen/llama/glm/deepseek/baichuan/mistral/other + custom supported | TBD | TBD | Not Verified | REVIEW-027 |
| Model Artifacts | Quantization options + custom | Q4_K_M/Q5_K_M/Q8_0/FP16/BF16/FP8/INT8/INT4/none/other + custom supported | TBD | TBD | Not Verified | REVIEW-027 |
| Backend Runtime | Create | User can create BackendRuntime | TBD | TBD | Not Verified | REVIEW-028 |
| Backend Runtime | View/Edit/Delete | User can manage BackendRuntime lifecycle | TBD | TBD | Not Verified | REVIEW-028 |
| Deployment | Create | User can create deployment with validated references | TBD | TBD | Not Verified | REVIEW-022 |
| Deployment | Dry-run | Preview matches actual runtime spec | TBD | TBD | Not Verified | REVIEW-003 |
| Deployment | Start | User can start an instance | TBD | TBD | Not Verified | REVIEW-023 |
| Deployment | Stop | User can stop an instance, including missing-container case | TBD | TBD | Not Verified | REVIEW-006 |
| Deployment | Error display | Runtime/start/health errors are visible and actionable | TBD | TBD | Not Verified | REVIEW-028 |
| Instances | Status | Canonical state displayed | TBD | TBD | Not Verified | REVIEW-007 |
| Instances | Endpoint | Endpoint shown after successful start | TBD | TBD | Not Verified | REVIEW-023 |
| Instances | Error state | Failed/unknown/stopped states are meaningful | TBD | TBD | Not Verified | REVIEW-028 |
| Observability | Links | Prometheus/Grafana links respect mode | TBD | TBD | Not Verified | REVIEW-017 |
| Observability | No data | No-data state provides troubleshooting guidance | TBD | TBD | Not Verified | REVIEW-017 |
| Admin | Users | Basic user operations work | TBD | TBD | Not Verified | REVIEW-028 |
| Admin | Tenants | Basic tenant operations work | TBD | TBD | Not Verified | REVIEW-028 |
| Admin | Roles/RBAC | Roles/permissions visible and consistent | TBD | TBD | Not Verified | REVIEW-028 |
| Audit | Audit list | Audit logs scoped by tenant | TBD | TBD | Not Verified | REVIEW-009 |
| i18n | Navigation keys | nav.models/nav.runtime render in zh-CN and en-US | TBD | TBD | Not Verified | REVIEW-026 |
| i18n | Raw key scan | No raw key in core pages | TBD | TBD | Not Verified | REVIEW-026 |
| UX | Loading states | Core pages have loading state | TBD | TBD | Not Verified | REVIEW-028 |
| UX | Empty states | Core pages have empty state | TBD | TBD | Not Verified | REVIEW-028 |
| UX | Error states | Core pages have error state | TBD | TBD | Not Verified | REVIEW-028 |
| Build | Web tests | npm test passes | TBD | TBD | Not Verified | REVIEW-015 |
| Build | Web build | npm run build passes and chunk issue handled | TBD | TBD | Not Verified | REVIEW-024 |
