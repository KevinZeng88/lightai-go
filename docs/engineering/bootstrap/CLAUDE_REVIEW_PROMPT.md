# Claude Review Prompt

请审核 `docs/engineering/bootstrap/` 下的 LightAI Bootstrap 工程设计文档。

本轮只审核文档，不实现代码。

重点审核：

1. 密码变量契约是否与当前代码一致。
2. `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` 和 `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` 的语义是否清晰、无冲突。
3. clean DB 初始化时写入 runtime initial credentials file 的规则是否安全、正确、可测试。
4. credentials file 的路径、格式、权限是否需要按当前代码修正文档。
5. `scripts/lightai-bootstrap.sh` 的 mode 设计是否能用现有 API 实现。
6. `auth-only / catalog-only / models-only / runtimes-only / dry-run / full / export` 的前置关系是否合理。
7. `full` 模式的双重允许保护是否足够。
8. profile schema 是否覆盖当前开发机 vLLM / SGLang / llama.cpp 和两个测试模型。
9. export mode 是否足够安全，是否会误导用户提交敏感信息。
10. packaging 要求是否与现有打包脚本一致。
11. acceptance criteria 是否完整且可执行。
12. execution plan 是否能按 batch 安全推进。

请输出：

1. 总体结论：PROCEED / PROCEED_WITH_CHANGES / PAUSE。
2. 必须修改的问题。
3. 建议修改的问题。
4. 当前代码/API 可能不支持的点。
5. 需要用户确认的问题。
6. 建议执行顺序调整。
7. 是否建议进入实现阶段。

不要直接实现代码。
