# Export Mode 设计

状态：`READY_FOR_IMPLEMENTATION`

## 目标

`export` 模式用于从当前已配置好的 LightAI 环境生成可复用 bootstrap profile。

用途：

- reset DB 后重建开发测试环境；
- 新环境安装后自动初始化；
- 迁移当前运行配置；
- 生成 Browser E2E / API E2E 测试数据 profile；
- 作为发行版安装初始化模板。

## 命令

默认：

```bash
bash scripts/lightai-bootstrap.sh --mode export
```

默认输出：

```text
configs/bootstrap/local-kz-laptop.yaml
```

指定输出：

```bash
bash scripts/lightai-bootstrap.sh \
  --mode export \
  --output-profile configs/bootstrap/exported.yaml
```

常用参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--output-profile` | `configs/bootstrap/local-kz-laptop.yaml` | 输出 YAML profile |
| `--include-secrets` | false | 是否导出敏感字段 |
| `--include-runtime-state` | false | 是否导出 deployments / instances 等运行态 |
| `--yes` | false | 高风险确认 |

## 执行原则

1. export 只读取系统状态，不修改系统状态。
2. API 优先。
3. runtime-dir / DB 只作为补充和诊断。
4. Agent 可用于补充路径扫描、镜像证据、文件检查。
5. 输出 YAML 字段顺序稳定，便于 git diff。
6. 输出 profile 默认可提交，不包含真实密码、token、cookie、CSRF。

## 数据来源

### API 优先读取

应优先通过 API 获取：

- 当前用户 / tenant；
- nodes；
- model artifacts；
- model locations；
- backends；
- backend versions；
- backend runtimes；
- node backend runtimes；
- deployments，如 `--include-runtime-state=true`。

### runtime-dir 辅助

从 runtime-dir 读取或确认：

- `<runtime-dir>/data/lightai.db`；
- `<runtime-dir>/logs/`；
- runtime initial credentials file 路径；
- 当前 runtime 目录结构。

必要时读取 DB 做补充，但不应直接改 DB。

### Agent 辅助

如 API 支持，可读取：

- node files scan；
- model-paths scan；
- docker image inspect / check evidence；
- agent 节点硬件信息。

## 输出 profile 字段

### 顶层

```yaml
profile_name: exported-local
```

### server

```yaml
server:
  base_url: http://localhost:18080
  agent_url: http://localhost:19091
  runtime_dir: /tmp/lightai
```

### auth

```yaml
auth:
  username: admin
  initial_password_env: LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD
  initial_password: ""
  initial_password_file: ""
  initial_password_runtime_files:
    - auto
  final_password_env: LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD
  final_password_file: ""
```

要求：

- 默认不输出真实密码；
- 默认写 env 名；
- 可以记录 runtime credentials file 路径，但不写密码。

### tenant

```yaml
tenant:
  name: default
  id: "..."
```

### node

```yaml
node:
  name: KZ-LAPTOP
  id: "..."
  gpu_vendor: nvidia
  gpu_ids:
    - "0"
  accelerator_ids:
    - "0"
```

### models

```yaml
models:
  qwen3_small:
    display_name: Qwen3-0.6B-Instruct-2512
    kind: huggingface
    path: /home/kzeng/models/Qwen3-0.6B-Instruct-2512
    artifact_id: "..."
    location_id: "..."
    source: export
```

key 生成规则：

- 优先使用已有 name；
- 若 name 不适合 YAML key，转换为小写 snake_case；
- 冲突时加数字后缀；
- 输出 `export-resource-map.json` 记录 key 与原始 ID 的映射。

### runtimes

```yaml
runtimes:
  vllm:
    backend: vllm
    backend_version: "..."
    backend_runtime_id: "..."
    node_backend_runtime_id: "..."
    image: vllm/vllm-openai:latest
    model: qwen3_small
    container_port: 8000
    host_port: 8004
    docker_json: {}
    args_override_json: {}
    default_env_json: {}
    parameter_values_json: {}
    health_check: {}
    status: ready
```

要求：

- 参数值必须保留 enabled + value；
- default_env_json 默认脱敏；
- status 仅供参考，不作为 bootstrap 强制结果。

### deployments

只有 `--include-runtime-state=true` 时导出：

```yaml
deployments:
  demo_vllm:
    name: demo-vllm
    model: qwen3_small
    runtime: vllm
    deployment_id: "..."
    service_config: {}
    parameter_values_json: {}
    enabled: true
```

### bootstrap

```yaml
bootstrap:
  default_mode: dry-run
  output_dir: /tmp/lightai/e2e/bootstrap
  allow_real_container_start: false
  allow_chat_completion: false
```

## 安全规则

1. 默认不导出密码、token、cookie、CSRF。
2. `auth.initial_password_env` 默认写 `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD`。
3. `auth.final_password_env` 默认写 `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD`。
4. `initial_password_runtime_files` 可以记录 credentials file 路径，但不写出明文密码。
5. env_json 中包含 token、key、secret、password 等敏感字段时默认脱敏。
6. `--include-secrets=true` 才允许导出敏感字段。
7. `--include-secrets=true` 必须同时传 `--yes`。
8. 即使 include secrets，也不导出 session token、cookie、CSRF。
9. 输出前打印明确 WARNING。

## 备份规则

如果 `--output-profile` 已存在：

1. 默认备份为 `<file>.bak.<timestamp>`；
2. 再写入新文件；
3. 备份路径记录到 `export-summary.json`。

## 输出文件

输出 profile：

```text
configs/bootstrap/local-kz-laptop.yaml
```

同时输出到 output-dir：

```text
/tmp/lightai/e2e/bootstrap/export-summary.json
/tmp/lightai/e2e/bootstrap/export-resource-map.json
/tmp/lightai/e2e/bootstrap/export-warnings.json
/tmp/lightai/e2e/bootstrap/export.log
```

### export-summary.json

示例：

```json
{
  "profile_path": "configs/bootstrap/local-kz-laptop.yaml",
  "backup_path": "configs/bootstrap/local-kz-laptop.yaml.bak.20260625T120000",
  "tenants": 1,
  "nodes": 1,
  "models": 2,
  "model_locations": 2,
  "runtimes": 3,
  "node_backend_runtimes": 3,
  "deployments": 0,
  "warnings": 1,
  "generated_at": "..."
}
```

### export-resource-map.json

记录 YAML key 与 API ID 的映射。

### export-warnings.json

记录：

- 无法导出的字段；
- 被脱敏字段；
- 需要人工确认的路径；
- 可能无法跨机器复用的绝对路径；
- 缺少 image evidence 的 runtime；
- deployments 被跳过的原因。

## 验收标准

1. `bash scripts/lightai-bootstrap.sh --mode export` 能生成 `configs/bootstrap/local-kz-laptop.yaml`。
2. 输出 profile 不包含密码、token、cookie、CSRF。
3. 输出 profile 包含 vLLM / SGLang / llama.cpp runtime。
4. 输出 profile 包含当前测试模型路径。
5. 输出 profile 包含参数 enabled + value。
6. 输出 profile 字段顺序稳定。
7. 已存在 output-profile 时会备份。
8. `export-summary.json` 记录资源数量。
9. `export-warnings.json` 记录脱敏和不确定项。
10. 用 export 生成的 profile 运行 `dry-run` 可恢复相同测试环境。
