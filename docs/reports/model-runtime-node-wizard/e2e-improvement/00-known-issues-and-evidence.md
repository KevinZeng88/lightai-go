# 00 - 已知问题与证据（Known Issues and Evidence）

> 适用范围：LightAI Go 当前模型运行 / Backend Runtime / RunPlan / Docker 命令预览 / E2E 脚本体系。  
> 目的：把已经人工发现的问题、证据和 E2E 漏检原因固化下来，供 Claude 审核、补充和生成实施计划。  
> 重要说明：本文档列出的是**已知问题与初步证据**，不是完整问题清单。Claude 必须继续独立审查现有脚本，发现本文档未列出的其它缺口。

---

## 1. 背景

近期项目已经修复或暴露出多类问题，这些问题说明现有 E2E 更偏向“默认 happy path / smoke test”，不足以发现配置传播、参数覆盖、页面入口和业务语义断言问题。

典型现象包括：

1. 运行模板复制弹窗里修改参数后，首次保存不生效；后续打开再修改才保存。
2. 用户设置 vLLM `host_port/container_port/app_port` 后，Docker command 仍使用默认 `--port 8000`。
3. Deployment 运行后模型部署页面不显示，根因是 list SELECT/Scan 列不一致。
4. 模型实例页停止报 HTTP 405，但模型部署页可以停止。
5. llama.cpp 测试请求成功，但响应摘要被误判为空。
6. vLLM positional model 与 `--model` 参数重复。
7. 默认参数覆盖用户参数，根因包括：
   - `deduplicateArgs` 保留 first，导致默认值压过用户值；
   - CLI 参数名与 snake_case 参数名匹配失败，例如 `--served-model-name` vs `served_model_name`；
   - `mapParametersToArgs` 使用 ParameterDef default，而没有正确读取 effective user config。
8. GGUF 是单文件，但部分 E2E 脚本对 file/directory/format 语义不够严谨。

---

## 2. 已知问题清单

### 2.1 vLLM 端口配置传播问题未被 E2E 发现

#### 现象

用户设置：

```text
host_port = 8111
container_port = 8022
app_port = 8022
```

但生成命令类似：

```bash
docker run ... -p 8111:8022/tcp vllm/vllm-openai:latest --model /models/Qwen3-0.6B-Instruct-2512 --host 0.0.0.0 --port 8000
```

问题在于 Docker 映射的是 `8111 -> 8022`，但 vLLM 实际监听 `8000`。平台 health check 访问 host_port 时自然失败，随后平台可能终止容器，日志出现 `KeyboardInterrupt: terminated`。

#### 已知根因

已由后续修复定位：

```text
mapParametersToArgs generates --port from ParameterDef defaults (hardcoded "8000").
The user's app_port lives in service_json, not parameters_json, so it was never consulted.
Even though buildVarMap correctly sets vars["app_port"]="8022",
the parameter mapper reads from a different source.
```

#### E2E 漏检原因

现有 vLLM 脚本多数只传 `host_port`，没有传 `container_port/app_port` 非默认组合；即便 modified case 修改了部分参数，也没有断言 Docker command / RunPlan 中的 `--port` 使用用户值。

需要 Claude 继续核实的脚本包括但不限于：

- `scripts/e2e-model-runtime-wizard-nvidia-api.sh`
- `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh`
- `scripts/e2e-ui-persistence-runplan-selected.sh`
- `scripts/e2e/lib/model-runtime-common.sh`

---

### 2.2 vLLM positional model 与 `--model` 重复或使用过时方式

#### 现象

vLLM 日志提示：

```text
With `vllm serve`, you should provide the model as a positional argument or in a config file instead of via the `--model` option.
The `--model` option will be removed in a future version.
```

同时 non-default args 出现：

```text
model_tag: /models/Qwen3-0.6B-Instruct-2512
model: /models/Qwen3-0.6B-Instruct-2512
```

说明模型路径可能同时以 positional 和 `--model` 两种方式出现，或模板仍默认使用 `--model`。

#### E2E 漏检原因

现有脚本没有强制断言：

```text
不应出现默认 enabled --model
不应同时出现 positional model 和 --model
模型路径应只出现一次
```

---

### 2.3 默认参数覆盖用户参数

#### 现象

后续清查发现：

```text
deduplicateArgs kept FIRST flag
```

这意味着当参数顺序为：

```text
default: --port 8000
user:    --port 8022
```

旧逻辑保留第一个，导致所有用户覆盖都可能失效。

#### 影响范围

不仅影响 `--port`，还可能影响：

- `--host`
- `--served-model-name`
- `--gpu-memory-utilization`
- `--tensor-parallel-size`
- `--max-model-len`
- `--trust-remote-code`
- `--ctx-size`
- `--n-gpu-layers`
- `--model-path`
- `-m`
- vendor visible device env
- custom app args / docker options

#### E2E 漏检原因

现有 E2E 大多只验证“能启动”或“命令里有某些片段”，没有验证：

```text
用户值最终覆盖默认值
默认值不得残留
同一个 flag 只能出现一次
```

---

### 2.4 Matrix 脚本是 runner/summary，不是 verifier

#### 现象

`e2e-model-runtime-wizard-nvidia-matrix.sh` 会运行 default/modified 组合并生成 summary，但核心逻辑更像汇总器：

- 它收集 payload / runplan。
- 它记录 `parameters_json`。
- 但没有对 Docker command / Docker create spec 做强断言。
- `param_assertion` 这类逻辑不等价于“modified 参数已经进入最终 RunPlan”。

#### E2E 漏检原因

matrix 即使显示 PASS，也可能只是：

```text
脚本执行完了
payload 中有参数
```

但没有证明：

```text
RunPlan 使用了参数
Docker command 使用了参数
错误默认值没有残留
```

因此应将其标注为 **WEAK_PASS 风险**，后续升级为 verifier。

---

### 2.5 `model-runtime-common.sh` 存在 false pass 风险

#### 现象

公共 helper 中部分函数更偏记录，不偏断言，例如：

- `e2e_instance_test()` 保存 response 并打印 status/duration，但未强制判断 raw_response / parsed summary 非空。
- `e2e_stop_deployment()` 对 stop 失败的处理偏宽松，正式 stop 与 cleanup 错误处理边界不清。
- 部分 logs/test 步骤失败可能只 log，不导致整体 fail。

#### E2E 漏检原因

如果 test API 请求成功但 summary 为空，现有 helper 不一定失败。因此无法发现：

```text
请求成功，但模型响应摘要被误判为空
```

后续必须把正式断言与 cleanup 容错分开：

```text
正式断言失败：exit 1
cleanup 清理失败：允许记录后继续
```

---

### 2.6 模型实例页 stop 没被覆盖

#### 现象

实际产品问题：

```text
模型部署页可以停止
模型实例页停止报 HTTP 405
```

#### E2E 漏检原因

现有脚本多调用：

```text
POST /api/v1/deployments/{id}/stop
```

而不是：

```text
POST /api/v1/model-instances/{id}/stop
```

因此它们只能证明 deployment-level stop 可用，不能证明 instance-level stop 可用。

后续必须新增独立 E2E 覆盖：

```text
运行实例 -> instance stop -> 非 405 -> stopped -> container stopped -> lease released -> deployment 状态同步
```

如果 force stop 已实现，也应覆盖。

---

### 2.7 Inference response parser 没有语义断言

#### 现象

实际问题：

```text
llama.cpp 测试请求成功，但模型响应摘要被误判为空。
```

#### E2E 漏检原因

部分脚本仅断言：

- `/v1/models` HTTP 成功；
- `/v1/chat/completions` HTTP 200；
- 或调用 `/model-instances/{id}/test` 后只打印结果。

缺少以下断言：

```text
raw_response 已保存
parsed summary 非空
content / reasoning_content / text / top-level response 字段被支持
raw_response 非空但 summary 空时必须 fail
```

---

### 2.8 clone template 首次保存问题未覆盖

#### 现象

运行模板复制弹窗里修改参数后首次保存不生效，后续打开再修改才保存。

#### E2E 漏检原因

现有脚本虽可能调用 clone，但多为：

```json
{}
```

或只改 `display_name`，没有在 clone payload 里修改：

- image
- env
- devices
- volumes
- app args
- ports
- high-risk options
- custom args

也没有验证：

```text
clone 后 GET 新配置
修改值保留
部署向导选择该配置
DryRun 使用修改值
原内置模板不变
```

---

### 2.9 llama.cpp / GGUF file/directory/format 语义不严谨

#### 事实要求

GGUF 通常是单文件：

```text
source_path: /path/to/model.gguf
path_type: file
format: gguf
llama.cpp -m: 容器内 .gguf 文件路径
```

#### 已知风险

部分脚本里 MODEL 是 `.gguf` 文件，但 scan 或 artifact 创建时可能使用：

```text
path_type: directory
format: huggingface
```

后续 location 又使用 `path_type:file`，语义前后不一致。

#### E2E 漏检原因

现有脚本没有强断言：

```text
GGUF artifact format=gguf
location path_type=file
volume mount 合理
-m 指向容器内 .gguf 文件
不是目录
```

---

### 2.10 legacy/local 脚本可能不代表当前产品链路

#### 风险

部分脚本可能仍使用旧 API 路径，例如：

```text
/model-deployments
```

或直接读 sqlite DB。

这类脚本可作为历史参考或本地 smoke，但不应作为当前 wizard/runtime/runplan 主验收依据。

Claude 需要继续核实：

- 哪些脚本仍是旧 API；
- 哪些脚本绕过当前 API；
- 哪些脚本应降级为 legacy；
- 哪些有价值断言应迁移到当前 E2E。

---

## 3. 需要 Claude 继续发现的问题

本文档不是完整清单。Claude 必须继续独立审查以下方面：

1. 是否仍使用旧 API。
2. 是否直接读 sqlite DB 绕过真实 API。
3. 是否吞掉 curl/jq/python/grep 错误。
4. 是否缺少反向断言。
5. 是否缺少 artifact。
6. 是否 cleanup 过于 destructive。
7. 是否启动服务但没有安全 PID 管理。
8. 是否误删用户 runtime/model root/deployment。
9. artifact 路径是否可能覆盖。
10. 是否缺少 SKIPPED_ENV 规则。
11. full matrix 是否默认太重，不适合日常运行。
12. 是否缺少只做 DryRun、不启动容器的快速参数传播测试。
13. 是否缺少 preview 与 Docker create spec 一致性测试。
14. 是否缺少重复参数 exactly once 断言。
15. 是否缺少 disabled 参数不渲染断言。
16. 是否缺少 custom 参数分类渲染断言。
17. 是否缺少 deployment start 后 list 仍显示断言。
18. 是否缺少失败时统一诊断采集。
19. 是否缺少 vendor-specific 参数检查，例如 NVIDIA/MetaX/Huawei visible device env。
20. 是否缺少端口语义检查，例如 `container_port == app_port`。

---

## 4. 现有脚本初步分类建议

| 脚本 | 初步定位 | 主要风险 |
|---|---|---|
| `scripts/e2e-model-runtime-wizard-nvidia-api.sh` | vLLM runtime smoke | 多测 host_port/default path，缺 container/app port 与 command 反向断言 |
| `scripts/e2e-model-runtime-wizard-nvidia-vllm.sh` | vLLM wrapper | modified 参数不足，缺 RunPlan/command 强断言 |
| `scripts/e2e-model-runtime-wizard-nvidia-matrix.sh` | matrix runner/summary | 不是 verifier，PASS 可能是 WEAK_PASS |
| `scripts/e2e/lib/model-runtime-common.sh` | shared helper | test/logs/stop false pass 风险 |
| `scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh` | llama.cpp runtime smoke | GGUF 语义不严，test/logs/instance stop 断言不足 |
| `scripts/e2e-model-runtime-wizard-nvidia-sglang.sh` | SGLang runtime smoke | 参数传播断言不足 |
| `scripts/e2e-ui-persistence-runplan-selected.sh` | UI persistence / dry-run smoke | 有端口值但只 grep 数字，不断言 app args |
| `scripts/e2e-backend-runtime-nvidia-api.sh` | backend runtime API smoke | 偏真实启动，不覆盖全量 user overrides |
| `scripts/e2e-model-runtime-api.sh` | legacy/API E2E | 旧 API/本地 runtime 风险 |
| `scripts/e2e-model-runtime-local.sh` | legacy/local full lifecycle | 直接进程/DB/旧模型链路，不宜作为主验收 |
| `scripts/e2e-model-runtime-failed-instance-logs.sh` | failed-state E2E | 范围窄，但断言相对有价值 |

Claude 必须基于代码实际情况修正/补充此表。

---

## 5. 对 Claude 的审查要求

Claude 在生成实施计划前，必须：

1. 核实本文档中的问题是否准确。
2. 标注每个问题的脚本证据。
3. 列出本文档未发现的新增问题。
4. 区分：
   - 已确认问题；
   - 疑似问题；
   - 需要运行后才能确认的问题；
   - 已不适用的问题。
5. 不得把本文档当作完整清单。
6. 不得直接执行改造，必须先生成计划给人工审核。
