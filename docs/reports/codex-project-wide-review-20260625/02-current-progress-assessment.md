# Current Progress Assessment

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| Server / Agent skeleton | Completed and relatively reliable | `cmd/server/main.go`, `cmd/agent/main.go`, `go build` PASS | Production hardening still limited by TLS and shared agent token. |
| Auth / tenant / RBAC | Completed but tests insufficient | `internal/server/auth`, `internal/server/rbac`, `internal/server/authz`; `go test` PASS but auth coverage 3.3%, authz 17.6% | Session/CSRF/Origin exist. More negative permission tests needed. |
| Agent registration / heartbeat | Completed | `HandleRegister`, `HandleHeartbeat`, `claimAndReturnTasks` | Node identity binding exists. Agent auth still global token. |
| GPU discovery / metrics | Completed for NVIDIA/mock, MetaX not real-verified | `internal/agent/collector/nvidia.go`, vendor sample tests | MetaX remains documented blocker. |
| ModelArtifact / ModelLocation | Completed but model consistency deep comparison incomplete | `artifact_handlers.go`, `model_location_handlers.go`, `model_scanner.go` | Model root policy is implemented. Capability persistence still partly inferred/UI-only. |
| Backend / BackendVersion catalog | Completed | `configs/backend-catalog/**`, `backend_handlers.go` | System/user catalog projection exists. OpenAPI stale. |
| BackendRuntime | Completed but cleanliness issues remain | `runtime_handlers.go`, `backend_runtimes` schema | Template layer exists; old template fallback remains. |
| NodeBackendRuntime | Partially reliable | `runtime_handlers.go` NBR list/enable/check/probe | `/check-request` is sounder; `/check` trusts client evidence. |
| RunPlan / NodeRunPlan | Completed but boundary inconsistent | `internal/server/runplan`, `preflightDeployment`, `resolved_run_plans` | Dry-run/start share resolver; `/deployments/preflight` does not. |
| Model deployment | Completed but edit/change semantics incomplete | `HandleCreateDeployment`, `HandlePatchDeployment`, Web wizard | Create uses NBR; edit UI cannot safely change NBR/runtime. |
| Model instance lifecycle | Completed but single-instance focused | `HandleStartDeployment`, `HandleStopDeployment`, task result handling | Multi-replica/distributed are placeholders. |
| Agent DockerRuntimeDriver | Completed and reasonably tested | `internal/agent/runtime/docker.go`, `docker_real.go`, tests coverage 66.3% | Real smoke still environment-dependent. |
| GPU / accelerator binding | Partially completed | RunPlan GPU env and Docker DeviceRequest support | Auto-picks first available GPU; no robust scheduler/lease conflict precheck for scale. |
| NVIDIA abstraction | Completed for local path | configs and tests | Real evidence exists in prior reports. |
| MetaX abstraction | Design/template/mock only | `metax_device_binding_test.go`, configs | Real hardware validation absent. |
| Parameter editing | Partially completed | `RuntimeParameterEditor.vue`, `parameter_values_json` | Old `parameters_json` references remain in scripts/docs; deployment Docker override scope incomplete. |
| Runtime template editing | Completed but snapshot implications need clarity | BackendRuntimesPage and handlers | Template sync exists for deployment, but explicit NBR reapply/change is not fully productized. |
| Deployment parameter override | Partially completed | `parameter_values_json`, `disabled_parameters_json` | Old payloads may silently drop `parameters_json`; API should reject unknown legacy fields. |
| Logs / diagnostics / instance status | Completed but real failure coverage limited | failed logs reports, `node_run_plan_logs_test.go` | Good handler-level evidence; real Docker smoke should remain a gate. |
| Prometheus / Grafana / metrics | Partially completed | `/metrics`, `/metrics/targets`, observability scripts | Server-managed Prom/Grafana supervision not fully in Go. |
| OpenAI-compatible API | Currently limited | Instance test probes `/v1/models`, chat/completions, completions fallback | No full gateway, API key, rate limit, usage accounting. |
| Usage / audit / billing | Mostly missing | audit logs exist; no API key/usage/billing model | Audit logs cover operations; usage/billing intentionally future. |
