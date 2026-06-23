# Staged Implementation and Acceptance Plan

## 1. 原则

当前阶段先讨论、审查、生成计划，不直接开发。设计通过后，再分阶段开发。

开发原则：

```text
1. 不新建分支，除非用户明确要求；
2. 不新增 schema/migration；
3. 不新增 production runtime backend；
4. 不改变 Batch 4 PARTIAL_PASS；
5. 不删除 VLM-RUNTIME-001 blocker；
6. 不做历史兼容 fallback；
7. 不把本机路径写进类型规则或能力定义；
8. 能修且可验证的问题本批解决；不能解决的写 formal blocker。
```

## 2. 推荐阶段

### Stage 0：Review Only

目标：让 Claude 理解文档、审查代码、提出问题和实施计划。

输出：

```text
1. 设计理解摘要；
2. 对设计的质疑和建议；
3. hardcode 初步审查结果；
4. 是否有过度设计；
5. 是否有遗漏对象；
6. 分阶段开发计划；
7. 风险和阻断点；
8. 暂不修改代码。
```

验收：

```text
1. Claude 未修改代码；
2. 提出的问题具体到文件/模块；
3. 对 RuntimeRequirements × BackendCapabilityProfile 匹配关系理解正确；
4. 能区分 ModelTypeProfile / DiscoveredMetadata / VerificationRecord；
5. 能指出哪些 hardcode 必须改、哪些可作为 enum/seed 保留。
```

### Stage 1：Contract Documentation

目标：把设计沉淀到正式 docs/design 文档。

产物：

```text
docs/design/model-runtime-contract-and-backend-capability-profile.md
docs/design/model-runtime-mainstream-matrix.md
```

验收：

```text
1. 文档明确对象边界；
2. RuntimeRequirements 是可执行契约；
3. BackendCapabilityProfile 与 RuntimeRequirements 有明确匹配规则；
4. 明确 ResolvedBackendCapability；
5. 明确 VerificationRecord 不混入能力契约；
6. 主流模型和 runtime 矩阵覆盖完整；
7. 没有绝对路径污染类型定义。
```

### Stage 2：Go Types and Validation

目标：新增统一 types、enum、validation、normalization。

建议位置：

```text
internal/modelmeta/
  constants.go
  types.go
  validate.go
  normalize.go
  compat.go
```

函数：

```go
func ValidateModelTypeProfile(p ModelTypeProfile) error
func ValidateDiscoveredMetadata(m DiscoveredMetadata) error
func ValidateRuntimeRequirements(r RuntimeRequirements) error
func ValidateBackendCapabilityProfile(p BackendCapabilityProfile) error
func ValidateResolvedBackendCapability(p ResolvedBackendCapability) error
func ValidateArtifactCapabilitySet(c []string) error
func CheckRuntimeRequirementsCompatibility(req RuntimeRequirements, backend ResolvedBackendCapability) CompatResult
```

验收：

```text
1. enum 非法必须报错；
2. required fields 缺失必须报错；
3. 绝对路径必须报错；
4. RuntimeRequirements 里 backend-specific CLI 必须报错；
5. BackendCapabilityProfile 里 evidence path/container id 必须报错；
6. compatibility result 是结构化状态，不只是 bool。
```

### Stage 3：Hardcode Audit and Minimal Refactor

目标：根据 audit 结果做低风险接入和改造。

最低改造：

```text
1. backend_versions.capabilities_json validation；
2. scanner metadata normalization；
3. CompatibilityChecker 从 BackendCapabilityProfile / ResolvedBackendCapability 读能力；
4. TestDispatcher 使用 test_endpoints，不猜 endpoint；
5. seed/catalog 能通过 validation；
6. frontend 不出现 i18n key 泄露。
```

RunPlan arg mapping 如果改动太大，可以列 formal blocker，但必须文档化。

### Stage 4：Tests

新增测试范围：

```text
1. ModelTypeProfile validation；
2. DiscoveredMetadata validation；
3. RuntimeRequirements validation；
4. BackendCapabilityProfile validation；
5. RuntimeRequirements × BackendCapabilityProfile compatibility；
6. Seed/catalog conformance；
7. Scanner metadata conformance；
8. Test endpoint dispatch conformance；
9. No backend-specific args in RuntimeRequirements；
10. No absolute paths in type-level contracts。
```

必须保留现有测试：

```text
1. detector tests；
2. compatibility tests；
3. API tests；
4. RunPlan tests；
5. frontend i18n tests；
6. npm test；
7. npm build。
```

### Stage 5：Closeout

新增：

```text
docs/reports/phase-3/web-ai-config-review/39-runtime-metadata-contract-backend-capability-refactor-closeout.md
```

必须包含：

```text
1. 范围；
2. 文档路径；
3. 代码实现路径；
4. hardcode audit 路径；
5. 改造清单；
6. formal blockers；
7. 测试结果；
8. 是否未新增 schema/migration；
9. 是否未新增 backend/runtime；
10. 是否未改变 Batch 4 PARTIAL_PASS；
11. commit id；
12. push result；
13. git status clean。
```

## 3. 验证命令

开发阶段完成后执行：

```bash
gofmt -w cmd/ internal/
go test ./internal/modelmeta/...
go test ./internal/agent/collector/...
go test ./internal/agent/...
go test ./internal/server/api/...
go test ./internal/server/runplan/...
go vet ./...
npm --prefix web test
npm --prefix web run build
git diff --check
git status --short
```

路径按实际项目调整。

## 4. 本批不应做的事

```text
1. 不重新跑生产 E2E；
2. 不新增 VLM runtime enablement；
3. 不新增 ONNX/TensorRT/OpenVINO serving；
4. 不新增 DB schema；
5. 不创建兼容旧 JSON 的多分支 fallback；
6. 不把所有 hardcode 一次性大重构到不可控。
```

## 5. 开发前讨论问题

Claude 审查后需要回答：

```text
1. 当前代码中 BackendVersion / BackendRuntime / NodeBackendRuntime 的边界是否足够承载 ResolvedBackendCapability？
2. capabilities_json 当前有哪些字段已经接近 BackendCapabilityProfile？
3. discovered_metadata_json 当前是否含本机绝对路径或非契约字段？
4. CompatibilityChecker 当前是否已经可迁移到 RuntimeRequirements × BackendCapabilityProfile？
5. TestDispatcher 是否仍硬编码 endpoint？
6. RunPlan 的 backend-specific args 是否能低风险迁移到 arg_support？
7. 哪些硬编码属于合法 enum，哪些必须改？
8. 哪些改造适合本批，哪些必须成为 formal blocker？
```
