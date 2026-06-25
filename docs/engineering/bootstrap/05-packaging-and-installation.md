# 打包与安装初始化设计

状态：`READY_FOR_IMPLEMENTATION`

> **注意**：`configs/bootstrap/` 目录当前不存在，将在实现阶段（Batch 2）创建。本文档描述目标打包要求。

## 目标

bootstrap 工具不仅用于开发机，也应进入发行版，成为安装后初始化工具。

安装场景应支持：

1. 安装前准备 YAML profile；
2. 启动 LightAI Server / Agent；
3. 运行 `scripts/lightai-bootstrap.sh`；
4. 自动完成登录、改密、模型、运行环境、NBR、preflight 初始化；
5. 输出状态文件和日志；
6. 后续可接 Browser E2E / API E2E。

## 发行版应包含

```text
scripts/lightai-bootstrap.sh
configs/bootstrap/bootstrap-profile.example.yaml
configs/bootstrap/local-kz-laptop.yaml
docs/engineering/bootstrap/lightai-bootstrap.md
```

如有辅助脚本，也应包含：

```text
scripts/lib/
scripts/e2e/lib/
```

## 打包脚本检查

需要检查并修改项目现有打包脚本，例如：

```text
scripts/package-release.sh
scripts/package-release-docker.sh
```

确保 release artifact 中包含：

- `configs/bootstrap/`
- `scripts/lightai-bootstrap.sh`
- `docs/engineering/bootstrap/lightai-bootstrap.md`

## 安装后最短初始化

```bash
export LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD='<initial-password>'
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='<final-admin-password>'

bash scripts/lightai-bootstrap.sh
```

如果 profile 不是默认路径：

```bash
bash scripts/lightai-bootstrap.sh --profile configs/bootstrap/site.yaml
```

## 安装 profile 示例

安装方可以复制：

```text
configs/bootstrap/bootstrap-profile.example.yaml
```

修改为：

```text
configs/bootstrap/site.yaml
```

填入：

- server URL；
- agent URL；
- runtime-dir；
- node 名称；
- GPU vendor / GPU ids；
- 模型路径；
- runtime image；
- host/container port；
- runtime 参数。

密码通过环境变量或文件传入，不写入 profile。

## 初始化密码与 credentials file

安装时常见做法：

```bash
export LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD='<initial-password>'
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='<final-admin-password>'
```

server clean DB 首次初始化 admin 时，会把实际生效 initial password 写入 runtime initial credentials file。

该文件敏感，权限必须为 `0600`。

bootstrap 首次运行时：

1. 先用 final/admin password 登录；
2. 失败后读取 initial password；
3. 如系统要求改密，则改为 final/admin password；
4. 后续运行直接用 final/admin password。

## Runtime-dir

默认：

```text
/tmp/lightai
```

安装场景可根据部署目录覆盖：

```bash
bash scripts/lightai-bootstrap.sh \
  --profile configs/bootstrap/site.yaml \
  --runtime-dir /opt/lightai/runtime
```

profile 也可配置：

```yaml
server:
  runtime_dir: /opt/lightai/runtime
```

## 输出目录

默认：

```text
/tmp/lightai/e2e/bootstrap/
```

安装场景可以覆盖：

```bash
bash scripts/lightai-bootstrap.sh --output-dir /var/log/lightai/bootstrap
```

## 发行版验证

打包后执行：

```bash
tar -tzf <release>.tar.gz | grep -E 'scripts/lightai-bootstrap.sh|configs/bootstrap|docs/engineering/bootstrap/lightai-bootstrap.md'
```

必须能看到：

```text
scripts/lightai-bootstrap.sh
configs/bootstrap/bootstrap-profile.example.yaml
configs/bootstrap/local-kz-laptop.yaml
docs/engineering/bootstrap/lightai-bootstrap.md
```

## 安装场景验收

1. clean DB 首次启动后，可通过 bootstrap 完成改密。
2. site profile 可完成模型注册。
3. site profile 可完成 runtime 配置。
4. dry-run 可完成 enable/check/preflight/runplan。
5. 输出 `bootstrap-state.json`。
6. 重复运行不会产生重复资源。
7. credentials file 权限为 `0600`。
8. release artifact 包含 bootstrap 所需文件。

## 运维注意事项

1. 不要把包含真实密码的 profile 提交到仓库。
2. 不要把 runtime initial credentials file 上传到不受控系统。
3. 安装后建议尽快使用 final/admin password 完成改密。
4. 如果使用 `full` 模式，必须显式开启：

```bash
bash scripts/lightai-bootstrap.sh --mode full --allow-real-start
```

5. full 模式可能启动真实容器，占用 GPU 和端口。
