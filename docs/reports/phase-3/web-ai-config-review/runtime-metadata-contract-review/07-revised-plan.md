# Runtime Metadata Contract — Final Implementation Plan

> **本轮只生成计划，不修改代码。先更新此文档，确认后再执行。**
>
> 基于 `06-review-report.md` 复核 + 用户决策后最终版本。

---

## 0. 总体决策摘要

| 决策点 | 结论 |
|--------|------|
| 核心诊断 | 五源漂移 → 建立 Go constants → DB seed → API → Frontend 单一可信源链 |
| Schema change | 否。`model_locations.path_type` 已存在 |
| 数据修复 | 是。Ollama `capabilities_json` 裸数组→结构化对象 |
| 旧配置删除 | 是。`configs/model-runtime/` 整个目录 |
| Constants 包位置 | **待确认。推荐方案 B（新建 `internal/runtimecontract/`），见 §2** |
| 16-point matching | 不做 |
| RuntimeRequirements 完整 struct | 不做 |
| ResolvedBackendCapability | 不做 |
| arg abstraction | 不做 |
| ModelTypeProfile config-driven | 不做 |
| Device binding 抽象 | 不做。接受 switch。Ascend 用 template_only 不伪造绑定 |

### 5 个待确认问题 → 已决策

| # | 问题 | 决策 |
|---|------|------|
| 1 | 前端 inference | 保留，仅 wizard 临时预览。已保存模型 persisted 为空时显示未配置，不静默推断。加测试 |
| 2 | Catalog YAML vs DB | 保留 YAML 但改为与 DB 一致的结构化格式。Go constants = 编译时 canonical，DB = 运行时 canonical，YAML = human-readable，Frontend = 从 API 获取。加 catalog consistency 测试 |
| 3 | Ascend device binding | 不补真实绑定。`defaultVisibleEnvKey` 改为读 `docker_json.gpu_visible_env_key`。`buildDeviceBinding` 对 huawei/ascend 处理为 `template_only`。等硬件验证后再启用 |
| 4 | Backend family matching | 保留 map + 注释（user input normalization，非 capability source）。加测试覆盖归一化 |
| 5 | API endpoint 权限 | 需要认证（登录态/RBAC）。前端 API 失败时 fallback 本地默认 options + warning 日志。测试覆盖成功和 fallback 两条路径 |

---

## 1. 实施路线（5 个阶段）

| 阶段 | 内容 | 改代码 |
|------|------|--------|
| A | Go 常量 + 统一校验（`internal/runtimecontract/`） | 是 |
| B | 数据一致性修复（Ollama、path_type、docker_json、YAML 清理） | 是 |
| C | 前端收敛（API endpoint + options 迁移 + inference 限定） | 是 |
| D | 正式设计文档 | 否 |
| E | 测试 + closeout | 是 |

---

## 2. Phase A 包位置：Import Graph 分析与推荐 ⚠️ 需确认

### 当前 import 关系

```
internal/agent/collector/  →  internal/common/log  (仅此一个外部依赖)
internal/server/api/        →  internal/agent/collector  (测试文件)
internal/server/runplan/    →  internal/common/log
```

**Agent 不依赖 Server。Server 测试文件依赖 Agent。这是干净的架构分离。**

### 如果 constants 放在 `internal/server/runplan/`

- `internal/agent/collector/model_scanner.go` 需要使用 `runplan.FormatHuggingFace` 等常量
- 必须新增 `import "lightai-go/internal/server/runplan"`
- 后果：**agent → server 反向依赖**，破坏当前架构

### 推荐方案 B：新建 `internal/runtimecontract/`

```
internal/runtimecontract/
├── constants.go          # 所有 enum 常量
├── validation.go         # IsValidFormat/IsValidTask/... 校验函数
└── constants_test.go     # 常量 + 校验测试
```

- 零外部依赖（不 import server 或 agent 的任何包）
- `internal/server/api/` → `internal/runtimecontract/`（正向）
- `internal/server/runplan/` → `internal/runtimecontract/`（正向）
- `internal/server/db/` → `internal/runtimecontract/`（正向）
- `internal/agent/collector/` → `internal/runtimecontract/`（正向，中性包）

### 备选方案 A：constants 放 `internal/server/runplan/`，agent 保留本地字符串

- Agent scanner 继续用局部变量/字符串，不 import server
- 通过 consistency test 保证 server 和 agent 的值一致
- 风险：新增 model type 时需改两处

**等待用户确认方案 B 还是方案 A。如未回复，默认方案 B。**

---

## 3. Phase A：Go 常量 + 统一校验

**前提：** 方案 B 确认后，在 `internal/runtimecontract/` 下实施。

### 3.1 新建文件

**`internal/runtimecontract/constants.go`：**

```go
package runtimecontract

// ── Format constants ──
const (
    FormatHuggingFace          = "huggingface"
    FormatSentenceTransformers = "sentence_transformers"
    FormatGGUF                 = "gguf"
    FormatLoRAAdapter          = "lora_adapter"
    FormatDiffusers            = "diffusers"
    FormatONNX                 = "onnx"
    FormatTensorRT             = "tensorrt_engine"
    FormatOpenVINO             = "openvino"
    FormatOllama               = "ollama"
)

// ── Task constants ──
const (
    TaskChat           = "chat"
    TaskCompletion     = "completion"
    TaskEmbedding      = "embedding"
    TaskRerank         = "rerank"
    TaskVisionChat     = "vision_chat"
    TaskAdapter        = "adapter"
    TaskUnknown        = "unknown"
    TaskImageGeneration = "image_generation"
    TaskASR            = "asr"
    TaskTTS            = "tts"
    TaskClassification = "classification"
)

// ── Capability constants ──
const (
    CapabilityChat              = "chat"
    CapabilityCompletion        = "completion"
    CapabilityEmbedding         = "embedding"
    CapabilityRerank            = "rerank"
    CapabilityVision            = "vision"
    CapabilityImageGeneration   = "image_generation"
    CapabilityASR               = "asr"
    CapabilityTTS               = "tts"
    CapabilityClassification    = "classification"
    CapabilityToolCalling       = "tool_calling"
    CapabilityStructuredOutput  = "structured_output"
)

// ── PathMode constants ──
const (
    PathModeDirectory     = "directory"
    PathModeFile          = "file"
    PathModeOllamaManaged = "ollama_managed"
)

// ── CapabilitySource constants ──
const (
    CapabilitySourceScan          = "scan"
    CapabilitySourceInferred      = "inferred"
    CapabilitySourceUserOverride  = "user_override"
    CapabilitySourceBackendProbe  = "backend_probe"
)

// ── TestMode constants ──
const (
    TestModeAuto       = "auto"
    TestModeChat       = "chat"
    TestModeCompletion = "completion"
    TestModeEmbedding  = "embedding"
    TestModeRerank     = "rerank"
)

// ── ServingProtocol constants ──
const (
    ServingProtocolOpenAICompatible = "openai-compatible"
    ServingProtocolOllama           = "ollama"
)
```

**`internal/runtimecontract/validation.go`：**

```go
package runtimecontract

func IsValidFormat(s string) bool { ... }
func IsValidTask(s string) bool { ... }
func IsValidCapability(s string) bool { ... }
func IsValidPathMode(s string) bool { ... }
func IsValidCapabilitySource(s string) bool { ... }
func IsValidTestMode(s string) bool { ... }
func IsValidServingProtocol(s string) bool { ... }

// AllFormats returns the canonical format list for API use.
func AllFormats() []string { ... }
// AllTasks returns the canonical task list for API use.
func AllTasks() []string { ... }
// ... etc
```

### 3.2 修改文件

| 文件 | 变更 |
|------|------|
| `internal/runtimecontract/constants.go` | **新建** |
| `internal/runtimecontract/validation.go` | **新建** |
| `internal/runtimecontract/constants_test.go` | **新建** — 常量唯一性、IsValid* 函数测试 |
| `internal/server/api/artifact_handlers.go:17-39` | 替换 `allowed*` map → `runtimecontract.IsValid*()` |
| `internal/agent/collector/model_scanner.go:132-218` | plugin defaults 使用 `runtimecontract.Format*` / `runtimecontract.Task*` 等常量 |
| `internal/server/db/db.go:1378-1383` | capabilities_json seed 加注释引用 `runtimecontract` 常量名 |

### 3.3 注意：不在 Phase A 删除旧 validation maps

`artifact_handlers.go` 中的 `allowedCapabilities` 等 map 改为调用 `runtimecontract.IsValidCapability()` 等函数，但保留原有的 map 变量声明（如果存在引用）。确保所有调用方通过新函数校验。

---

## 4. Phase B：数据一致性修复

### B1. Ollama capabilities_json 结构化

**修改：**

`db.go:1383`（seed 中 Ollama `backend_versions` 行的 `caps` 字段）：
```json
{
  "supported_formats": ["ollama"],
  "supported_tasks": ["chat", "completion"],
  "supported_capabilities": ["chat", "completion"],
  "model_path_modes": ["ollama_managed"],
  "serving_protocols": ["ollama"],
  "test_endpoints": {
    "chat": "/api/chat",
    "completion": "/api/generate"
  }
}
```

`db.go:1591-1610`（`repairBackendCapabilitiesV27`）：
- 新增 `"backend-version.ollama.latest"` 条目，值为上述结构化 JSON

**配套修改：**
- `runplan/compat.go:94-117`：`ParseBackendCapabilities` 匿名 struct 新增 `ServingProtocols []string \`json:"serving_protocols"\``
- `runplan/compat.go:15-23`：`BackendDescriptor` 新增 `ServingProtocols []string`

**验证：**
- `TestParseBackendCapabilitiesOllama` — 解析成功，SupportedFormats/ServingProtocols 非空
- `TestCompatOllamaWithOllamaModelPasses` — ModelDescriptor `{Format:"ollama", Task:"chat", PathType:"ollama_managed"}` + Ollama BackendDescriptor → PASS
- 确认裸数组 `["ollama"]` 不再存在于任何 capabilities_json 数据中

### B2. path_type 从 `model_locations.path_type` 列读取

**文件：** `internal/server/api/deployment_lifecycle_handlers.go:882-888`

**当前代码：**
```go
modelPathType := "directory"
if modelFormat == "gguf" {
    modelPathType = "file"
}
```

**修改为：**
```go
modelPathType := "directory" // fallback
if loc := h.getModelLocationJSON(pf.locationID); loc != nil {
    if pt, ok := loc["path_type"].(string); ok && pt != "" {
        modelPathType = pt
    }
}
```

**审计：** 确认 scanner 正确写入 `path_type`（`ScanCandidate.PathType` → `model_locations.path_type`）

### B3. `defaultVisibleEnvKey()` 从 docker_json 读取

**文件：** `internal/server/runplan/resolver.go:901-910`

**修改：** 删除 switch，改为从 `DockerSpecInfo.GpuVisibleEnvKey` 读取。如果 `GpuVisibleEnvKey` 为空，fallback 到 `"CUDA_VISIBLE_DEVICES"`。

**注意：** 确认 `DockerSpecInfo` struct 已定义 `GpuVisibleEnvKey` 字段，并从 `docker_json` 解析。

### B4. `buildDeviceBinding()` ascend/huawei 处理

**文件：** `internal/server/runplan/resolver.go:983-1002`

**修改：** 新增：
```go
case "huawei", "ascend":
    binding.Mode = "template_only"
    // 不设置设备绑定参数。待硬件 + smoke evidence 后再启用真实绑定。
```

`template_only` 模式下，preflight/runplan 不应暗示可以真实启动 Ascend runtime。

### B5. Catalog YAML 清理

**删除（旧格式，已被 `configs/backend-catalog/` 取代）：**
```
configs/model-runtime/  （整个目录，含 backends/、backend-versions/、backend-runtime-templates/、profiles/）
```

**更新（改为与 DB seed 一致的结构化格式）：**
```
configs/backend-catalog/versions/sglang/sglang-0.4.6-compatible.yaml
configs/backend-catalog/versions/ollama/ollama-latest.yaml
```

### B6. Backend family matching 加注释

**文件：** `internal/server/api/runtime_handlers.go:997-1001`

不改逻辑。在 `patterns` map 上方加注释：
```go
// patterns maps backend family names to common image name variants.
// This is for user input normalization only — NOT a backend capability source.
// Canonical capability data is in backend_versions.capabilities_json.
// Test: TestMatchBackendTypeNormalization covers all 4 families.
```

---

## 5. Phase C：前端收敛

### C1. 新增 API endpoint：`GET /api/enums/model-capabilities`

**权限：** 需要认证（登录态/RBAC，与现有 API 一致）

**响应格式：**
```json
{
  "formats": ["gguf", "huggingface", "safetensors", "sentence_transformers", "lora_adapter", "diffusers", "onnx", "openvino", "tensorrt_engine", "ollama", "pt", "other"],
  "tasks": ["chat", "completion", "embedding", "rerank", "vision_chat", "adapter", "image_generation", "asr", "tts", "classification", "unknown"],
  "capabilities": ["chat", "completion", "embedding", "rerank", "vision", "image_generation", "asr", "tts", "classification", "tool_calling", "structured_output"],
  "test_modes": ["auto", "chat", "completion", "embedding", "rerank"]
}
```

**实现：**
- `internal/server/api/` 新增 handler，调用 `runtimecontract.AllFormats()` 等函数生成响应
- `internal/server/server.go` 注册路由

**前端消费：**
- `ModelArtifactsPage.vue` 优先从 API 获取 options
- API 失败时 fallback 到本地默认 options（兜底）
- fallback 时 `console.warn` 记录
- 本地 fallback 不是 canonical source，仅保证页面可用

### C2. 前端 `inferModelCapabilities` 语义收紧

**规则：**
- Persisted `capabilities` 非空 → 只展示 persisted（不变）
- Persisted `capabilities` 为空 + wizard 临时预览场景 → 可使用 regex inference
- Persisted `capabilities` 为空 + 已保存模型正式展示 → 显示"未配置"，提示重新扫描或编辑
- 不静默 fallback 到 regex 推断

**文件：** `web/src/utils/modelCapabilities.js:68-123`

**测试：** `web/tests/modelCapabilities.test.mjs` 新增：persisted 为空时 `inferModelCapabilities` 返回空数组（或标记为 `source: 'none'`）

### C3. CAPABILITY_LABELS 补全

**文件：** `web/src/utils/modelCapabilities.js:1-9`

补全缺失的 label：
```js
image_generation: { zh: '图像生成', en: 'Image Generation' },
asr: { zh: '语音识别', en: 'ASR' },
tts: { zh: '语音合成', en: 'TTS' },
classification: { zh: '分类', en: 'Classification' },
```

**文件：** `web/src/locales/zh-CN.ts`、`web/src/locales/en-US.ts` — 同步更新 i18n

### C4. `formatTestFailure` 使用结构化错误

**文件：** `web/src/utils/modelCapabilities.js:149-192`

- 如果 API 已返回 `reason_code`，前端改为 switch on `reason_code`
- 如果 API 未返回结构化错误，本次最小改动：将现有逻辑重构为 `switch(reason_code)` + fallback

---

## 6. Phase D：正式设计文档（不改代码）

| 文件 | 内容 |
|------|------|
| `docs/design/model-runtime-contract.md` | Canonical source 架构定义、6-point check、deferred checks、BackendVersion→BackendRuntime→NodeBackendRuntime 分层 |
| `docs/design/model-runtime-mainstream-matrix.md` | 14 模型类型 × 4 后端矩阵，标注 verified/blocked/unverified/recognized_non_deployable |
| `docs/reports/phase-3/web-ai-config-review/runtime-metadata-contract-review/07-revised-plan.md` | 本文件的 docs 版本（精简） |

---

## 7. Phase E：测试 + closeout

### 7.1 单元测试

```bash
# Phase A
cd internal/runtimecontract && go test -v ./...
# TestIsValidFormat (valid + invalid), TestIsValidTask, TestIsValidCapability, TestIsValidPathMode
# TestIsValidServingProtocol, TestAllFormats, TestAllTasks, TestAllCapabilities

# Phase B
cd internal/server/runplan && go test -v -run "TestCompat|TestParse" ./...
# TestParseBackendCapabilitiesOllama — 验证结构化 JSON 解析
# TestCompatOllamaWithOllamaModelPasses — Ollama 兼容性通过
# TestCompatOllamaMissingCapabilitiesFails — 空 capabilities 失败
# 现有 15 个 compat 测试全部通过

# Phase C
cd web && npm test
# modelCapabilities.test.mjs — 验证 persisted 为空时不 inference
```

### 7.2 Catalog consistency 测试

```bash
cd internal/server/runplan && go test -v -run TestCatalogConsistency ./...
```

验证：
1. 所有 `backend_versions.capabilities_json` 可被 `ParseBackendCapabilities` 成功解析
2. `capabilities_json` 中的 format/task/capability/path_mode 值均为 `runtimecontract` 常量或合法值
3. Ollama 的 capabilities_json 是结构化对象（非裸数组）

### 7.3 Catalog YAML consistency

验证 `configs/backend-catalog/versions/*.yaml` 的 capabilities 字段与 DB seed 的 `capabilities_json` 一致。

### 7.4 构建验证

```bash
go build ./cmd/server/
go build ./cmd/agent/
cd web && npm run build
```

### 7.5 回归测试

```bash
cd internal && go test ./...
```

---

## 8. 修改文件总清单

### Phase A — `internal/runtimecontract/`

| 文件 | 操作 |
|------|------|
| `internal/runtimecontract/constants.go` | 新建 |
| `internal/runtimecontract/validation.go` | 新建 |
| `internal/runtimecontract/constants_test.go` | 新建 |
| `internal/server/api/artifact_handlers.go` | 修改 — 使用 `runtimecontract.IsValid*()` |
| `internal/agent/collector/model_scanner.go` | 修改 — plugin defaults 使用常量 |
| `internal/server/db/db.go` | 修改 — seed JSON 注释引用常量 |

### Phase B — 数据修复

| 文件 | 操作 |
|------|------|
| `internal/server/db/db.go:1383` | 修改 — Ollama seed caps |
| `internal/server/db/db.go:1591-1610` | 修改 — V27 repair 新增 Ollama |
| `internal/server/runplan/compat.go:94-117` | 修改 — ParseBackendCapabilities 新增 serving_protocols |
| `internal/server/runplan/compat.go:15-23` | 修改 — BackendDescriptor 新增 ServingProtocols |
| `internal/server/runplan/compat_test.go` | 修改 — 新增 Ollama 测试 |
| `internal/server/api/deployment_lifecycle_handlers.go:882-888` | 修改 — 从 location 读 path_type |
| `internal/server/runplan/resolver.go:901-910` | 修改 — defaultVisibleEnvKey 读 docker_json |
| `internal/server/runplan/resolver.go:983-1002` | 修改 — ascend template_only |
| `internal/server/api/runtime_handlers.go:997-1001` | 修改 — 加注释 |
| `configs/model-runtime/` | **删除** 整个目录 |
| `configs/backend-catalog/versions/sglang/sglang-0.4.6-compatible.yaml` | 修改 — capabilities 结构化 |
| `configs/backend-catalog/versions/ollama/ollama-latest.yaml` | 修改 — 新增 capabilities_json |

### Phase C — 前端

| 文件 | 操作 |
|------|------|
| `internal/server/api/` + `internal/server/server.go` | 新建 endpoint + 注册路由 |
| `web/src/pages/ModelArtifactsPage.vue:368-399` | 修改 — API 获取 options + fallback |
| `web/src/utils/modelCapabilities.js:1-9` | 修改 — 补全 CAPABILITY_LABELS |
| `web/src/utils/modelCapabilities.js:68-123` | 修改 — inference 仅 wizard 使用 |
| `web/src/utils/modelCapabilities.js:149-192` | 修改 — 结构化错误 |
| `web/src/locales/zh-CN.ts` | 修改 — 新增 i18n key |
| `web/src/locales/en-US.ts` | 修改 — 新增 i18n key |
| `web/tests/modelCapabilities.test.mjs` | 修改 — 新增测试 |

### Phase D — 文档

| 文件 | 操作 |
|------|------|
| `docs/design/model-runtime-contract.md` | 新建 |
| `docs/design/model-runtime-mainstream-matrix.md` | 新建 |
| `docs/reports/phase-3/web-ai-config-review/runtime-metadata-contract-review/07-revised-plan.md` | 新建 |

### Phase E — closeout

| 文件 | 操作 |
|------|------|
| `docs/reports/phase-3/web-ai-config-review/runtime-metadata-contract-review/closeout.md` | 新建 |

---

## 9. 不做（本批明确排除）

| 项目 | 原因 |
|------|------|
| 16-point compatibility matching | 6-point 覆盖所有生产用例 |
| RuntimeRequirements 完整 struct | 无当前用例（modalities/serving_protocols/runtime_features） |
| ResolvedBackendCapability 正式合并 | 无 runtime-level capability override 用例 |
| arg abstraction 层 | template variable 系统已有部分抽象，完整抽象触及 resolver + DB seed |
| ModelTypeProfile config-driven detection | Go plugin 方案工作正常 |
| 真实 Ascend/Huawei device binding | 无硬件验证。template_only 处理 |
| Backend family matching 改为 DB 派生 | 4 个 backend 稳定，map + 注释足够 |
| 前端 inference 完全移除 | wizard 临时预览场景仍需 |

---

## 10. Formal Blockers（已削减）

仅保留 2 个：

### BLOCKER-001: RunPlan Arg Abstraction Layer
与 06-review-report 相同。触及 resolver + DB seed + 4 个 backend。

### BLOCKER-004: ModelTypeProfile Config-Driven Detection
与 06-review-report 相同。Go plugin 方案工作正常，配置化未来做。

BLOCKER-002（RuntimeRequirements）、BLOCKER-003（ResolvedBackendCapability）、BLOCKER-005（Device binding abstraction）降级为 deferred，不再作为 formal blocker 追踪。

---

## 11. 风险与回滚

| 阶段 | 风险 | 回滚 |
|------|------|------|
| A | 常量值与现有 string 不一致 | `git revert`（值完全相同，风险极低） |
| B1 | Ollama 结构化 JSON 导致行为变化 | 回滚单列 UPDATE |
| B2 | path_type 为空 → preflight 失败 | fallback → format 推断 + WARNING 日志 |
| B5 | 删除 YAML 后代码引用 | `grep -r "model-runtime"` 确认无引用 |
| C1 | API endpoint 失败 → 前端无 options | fallback 到本地 hardcode + console.warn |
| C2 | 关闭 inference 后模型显示空 | 渐进：wizard 保留 inference，正式页关闭 |

---

## 12. 验证命令汇总

```bash
# Phase A
cd internal/runtimecontract && go test -v ./...

# Phase B
cd internal/server/runplan && go test -v -run "TestCompat|TestParse|TestCatalog" ./...
cd internal/server/api && go test -v ./...

# Phase C
cd web && npm test

# Full regression
cd internal && go test ./...
go build ./cmd/server/ && go build ./cmd/agent/
cd web && npm run build

# Git
git status --short   # 必须为空（closeout 前提交所有变更）
```

---

## 13. 已确认：方案 B — `internal/runtimecontract/`

**决策：** 新建 `internal/runtimecontract/` 中性包，零外部依赖。Agent 和 Server 均通过 `import "lightai-go/internal/runtimecontract"` 引用。

与现有 `internal/common/log` 的模式一致：中性包，被所有模块引用，不引入反向依赖。
