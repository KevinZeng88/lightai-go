# 03 - 实施步骤建议（Implementation Steps Recommendation）

> 目的：给 Claude 生成正式实施计划前的步骤建议。  
> 原则：先计划、后执行；先低风险 DryRun，再真实 runtime；先修 false pass，再扩展矩阵；每一阶段都应独立验证，避免一次性大改。  
> 重要说明：本文档不是最终实施计划。Claude 应基于 `00-known-issues-and-evidence.md`、`01-formal-e2e-requirements.md`、`02-acceptance-criteria-and-parameter-matrix.md` 生成 `04-claude-review-and-implementation-plan.md`，供人工审核批准后再执行。

---

## 1. 总体流程

推荐流程：

```text
Step A：固化问题、要求、验收标准
Step B：Claude 审核这 4 个文件并补充意见
Step C：Claude 生成正式实施计划
Step D：人工审核实施计划
Step E：批准 Phase 0/1/2 优先实施
Step F：每个 Phase 完成后验证、提交、push
Step G：再进入下一 Phase
```

当前只允许 Claude 做到 Step C，不允许直接进入代码改造。

---

## 2. 对 Claude 的当前任务

Claude 当前应做：

1. 阅读本目录下文件：
   - `00-known-issues-and-evidence.md`
   - `01-formal-e2e-requirements.md`
   - `02-acceptance-criteria-and-parameter-matrix.md`
   - `03-implementation-steps-recommendation.md`
2. 审核这些文件是否有遗漏、矛盾、不可执行之处。
3. 继续独立检查现有 E2E 脚本。
4. 补充新发现的问题。
5. 生成正式实施计划：

```text
docs/reports/model-runtime-node-wizard/e2e-improvement/04-claude-review-and-implementation-plan.md
```

6. 等待人工审核批准。

Claude 当前不得：

- 修改产品代码；
- 修改 E2E 脚本；
- 运行真实 E2E；
- 启动 server/agent/container；
- 清理 DB/容器/模型资源；
- commit；
- push。

---

## 3. 允许的只读命令

Claude 可执行：

```bash
git status --short
git log --oneline --decorate -10
find scripts -maxdepth 3 -type f -name "*e2e*.sh" | sort
find scripts/e2e -maxdepth 3 -type f | sort
rg "e2e|E2E|DryRun|RunPlan|command-preview|model-instances|deployments|stop|force|test|raw_response|host_port|container_port|app_port|--port|--model|served_model|gpu_memory|llama|vllm|sglang" scripts docs internal web
bash -n scripts/*.sh scripts/e2e/*.sh scripts/e2e/lib/*.sh
```

不得执行：

```bash
scripts/e2e-*.sh
scripts/start-all.sh
docker run ...
docker rm ...
go test ./...   # 当前计划阶段可以不跑，除非人工批准
npm --prefix web test ...
```

如果 Claude 认为需要执行超出只读范围的命令，必须写入计划，等待人工批准。

---

## 4. Phase 0：脚本分级与安全策略

### 目标

建立 E2E 脚本分级体系，明确哪些脚本可以日常运行，哪些脚本需要 GPU/容器，哪些脚本只能作为 legacy/debug。

### 计划输出

- 更新或新增 E2E 脚本清单文档；
- 标记每个脚本：
  - smoke；
  - dry-run；
  - runtime；
  - inference；
  - failed-state；
  - matrix verifier；
  - legacy/local；
  - helper。

### 需检查

- 是否需要 GPU；
- 是否启动容器；
- 是否启动 server/agent；
- 是否修改 DB；
- 是否 destructive cleanup；
- 是否依赖硬编码路径；
- 是否支持 SKIPPED_ENV；
- 是否保存 artifact。

### 风险

低。主要是文档和注释。

### 建议先实施

是。Phase 0 应该最先做。

---

## 5. Phase 1：修复 false pass

### 目标

让现有 E2E 至少不会“失败却 PASS”。

### 计划工作

1. 建立统一 assert helper。
2. 修复 `model-runtime-common.sh`：
   - test 失败必须 fail；
   - raw_response/summary 空必须 fail；
   - logs 正式步骤失败必须 fail；
   - stop 正式步骤失败必须 fail；
   - cleanup 单独容错。
3. 修复各 backend-specific 脚本：
   - curl 检查 HTTP code；
   - JSON 关键字段为空 fail；
   - 子步骤失败 fail。
4. matrix 子项失败时总结果 fail。
5. 引入 `WEAK_PASS` 概念，避免弱断言脚本冒充正式 PASS。

### 风险

中。会改变历史脚本 PASS/FAIL 行为，可能暴露已有产品或脚本问题。

### 建议

优先实施，但需要人工批准。实施后先跑 `bash -n` 和轻量 dry-run，不要立刻 full matrix。

---

## 6. Phase 2：新增参数传播 DryRun E2E

### 目标

低成本发现参数传播和 Docker command 错误，不启动容器。

### 建议新增脚本

```text
scripts/e2e-runplan-parameter-source-audit.sh
```

### 覆盖 backend

- vLLM；
- SGLang；
- llama.cpp。

### 核心断言

vLLM：

```text
host_port=8111
container_port=8022
app_port=8022
served_model_name=qwen-vllm-e2e
gpu_memory_utilization=0.85
max_model_len=4096
```

断言：

```text
-p 8111:8022
--port 8022
not --port 8000
exactly one --port
not default enabled --model
positional model
--served-model-name qwen-vllm-e2e
--gpu-memory-utilization 0.85
```

SGLang：

```text
custom app_port
custom model path
custom tp/mem-fraction
--model-path correct
--port uses app_port
default not overriding
```

llama.cpp：

```text
GGUF file
format=gguf
path_type=file
-m points container .gguf file
--port uses app_port
ctx-size/n-gpu-layers uses user value
no duplicate host/port/model
```

### 风险

中低。不启动容器，但会创建/删除测试对象。

### 建议

Phase 1 后立即实施。它能最快发现近期同类问题。

---

## 7. Phase 3：新增 clone template E2E

### 目标

发现“复制弹窗修改参数首次保存不生效”问题。

### 建议新增脚本

```text
scripts/e2e-clone-template-parameter-persistence.sh
```

### 核心链路

```text
builtin runtime
  -> clone with modified payload
  -> GET clone detail
  -> enable on node
  -> create deployment using clone
  -> dry-run
  -> command preview uses clone values
  -> builtin unchanged
```

### 必须修改并验证的参数

- name/display_name；
- image；
- env；
- devices；
- volumes；
- docker options；
- app args；
- ports；
- startup timeout；
- health check；
- custom args；
- high-risk options enabled/disabled 状态。

### 风险

中。主要是 API/DB 对象创建，不必真实启动容器。

### 建议

Phase 2 通过后实施。

---

## 8. Phase 4：新增 deployment visibility E2E

### 目标

防止 deployment start 后 list 不显示。

### 建议新增脚本

```text
scripts/e2e-deployment-visibility-selected.sh
```

### 核心链路

```text
create deployment
  -> list contains id
  -> dry-run
  -> start
  -> list still contains id
  -> detail has status
  -> active_instance_id/current_run_plan exists
```

### 风险

中。可能需要启动 selected runtime；也可先用 fake/dry-run 模式检测 list/detail 字段完整性。

### 建议

先做 API/dry-run 版本，再纳入 selected runtime。

---

## 9. Phase 5：新增 instance stop E2E

### 目标

防止“部署页能停、实例页不能停”。

### 建议新增脚本

```text
scripts/e2e-instance-stop-selected.sh
```

### 核心链路

```text
start selected llama.cpp instance
  -> POST /api/v1/model-instances/{id}/stop
  -> not 405
  -> state stopped
  -> container stopped
  -> GPU lease released
  -> deployment state synced
```

如果 force stop 已实现，补充：

```text
POST /api/v1/model-instances/{id}/force-stop
```

### 风险

中高。需要真实 runtime、Docker、GPU。

### 建议

优先选 llama.cpp，因为环境已有 GGUF 和镜像基础。不要一开始就 full matrix。

---

## 10. Phase 6：新增 inference response parser E2E

### 目标

防止 raw response 有内容但 summary 被误判空。

### 建议新增脚本

```text
scripts/e2e-inference-response-parser-selected.sh
```

或拆分为：

- API fixture parser test；
- selected runtime inference E2E。

### 核心断言

- raw_response 保存；
- parsed summary 非空；
- content/reasoning_content/text/top-level response 均覆盖；
- raw_response 非空但 summary 空时 fail。

### 风险

中。fixture 低风险，真实 runtime 中风险。

### 建议

先用 fixture 或 API-level test，再做 selected runtime。

---

## 11. Phase 7：升级 matrix wrapper 为 verifier

### 目标

matrix 不再只是“跑多个脚本”，而是强断言矩阵。

### 计划工作

1. matrix 输出 PASS/FAIL/SKIPPED_ENV/WEAK_PASS。
2. 每个 modified case 必须有 assertions。
3. 子项失败导致总失败。
4. summary JSON 包含每项断言。
5. 支持只跑 dry-run matrix。
6. full runtime matrix 默认不自动运行。

### 风险

中。可能让原本看似 PASS 的脚本变成 WEAK_PASS/FAIL。

### 建议

放在 Phase 1-6 后做。

---

## 12. Phase 8：治理 legacy/local 脚本

### 目标

避免旧脚本误导验收结果。

### 计划工作

1. 标注 legacy/local。
2. 将有价值断言迁移到当前 API E2E。
3. 修正 README/文档，不再把 legacy 作为主 PASS。
4. 保留 debug 用途。

### 风险

低到中。

### 建议

最后做。

---

## 13. 建议优先级

最高优先级：

1. Phase 1：修 false pass。
2. Phase 2：参数传播 DryRun E2E。
3. Phase 3：clone template 参数保存 E2E。
4. Phase 5：instance stop E2E。

原因：

- Phase 1 防止“假 PASS”；
- Phase 2 能最快抓住参数覆盖类问题；
- Phase 3 对应复制模板实际 bug；
- Phase 5 对应实例页 stop 实际 bug。

---

## 14. 建议运行顺序

实施后建议按以下顺序运行：

```bash
git diff --check
bash -n scripts/*.sh scripts/e2e/*.sh scripts/e2e/lib/*.sh

# 低风险
scripts/e2e-runplan-parameter-source-audit.sh
scripts/e2e-clone-template-parameter-persistence.sh
scripts/e2e-deployment-visibility-selected.sh

# 中风险
scripts/e2e-inference-response-parser-selected.sh

# 需要真实 runtime
scripts/e2e-instance-stop-selected.sh
scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh

# 环境满足后再跑
scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
scripts/e2e-model-runtime-wizard-nvidia-sglang.sh

# 最后
scripts/e2e-model-runtime-wizard-nvidia-matrix.sh
```

---

## 15. Claude 正式计划输出要求

Claude 审核本文档后，应输出：

```text
docs/reports/model-runtime-node-wizard/e2e-improvement/04-claude-review-and-implementation-plan.md
```

该计划必须包含：

1. 当前分支和起始 commit。
2. 已阅读的文件。
3. 对 `00/01/02/03` 四个文件的审核意见。
4. 确认的问题。
5. Claude 额外发现的问题。
6. 分阶段实施计划。
7. 每阶段修改文件清单。
8. 每阶段风险。
9. 每阶段运行命令。
10. 每阶段验收标准。
11. 哪些阶段需要人工批准后才能运行。
12. 建议首批执行 Phase。
13. 本轮未执行修改/运行/提交/push 的确认。

---

## 16. 人工审批策略

建议人工先批准：

```text
Phase 0 + Phase 1 + Phase 2
```

暂缓：

```text
full runtime matrix
vLLM/SGLang full runtime
destructive cleanup 相关改动
legacy 脚本删除
```

每个 Phase 完成后必须：

```text
验证 -> 文档 -> commit -> push -> clean status
```

不要把所有 Phase 一次性做完。
