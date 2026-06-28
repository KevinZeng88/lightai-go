请在当前 main 分支执行 Runtime Config Field Display follow-up 修复，不新建分支。

先阅读：

- `docs/reports/phase-3/runtime-config-display-probe-fix/05-closeout.md`
- `docs/reports/phase-3/runtime-config-display-probe-fix/06-mhtml-config-field-review.md`
- `docs/reports/phase-3/runtime-config-display-probe-fix/07-config-field-display-design.md`

目标：

1. 修复 ConfigEditView object 子字段取值错误：UTS mode、Network mode 等字段不得显示整个 `launcher.docker_options` 父对象。
2. Docker options 统一按 `launcher.docker_options.<field>` 作为 canonical value path；旧 `docker.*` UI alias 必须显式映射到 canonical path。
3. 缺失的高级 Docker 字段隐藏或显示“未配置”，不得显示父对象、`{}`、`[]` 或破碎控件。
4. 正常详情页不要裸 JSON 展示常见对象/数组：model mount、env、health、entrypoint、command、ports、volumes、devices、ulimits、security options 要结构化显示。
5. Raw Config Set JSON 只保留在诊断区，默认收起。
6. 补测试覆盖 object child value resolution、Docker alias mapping、structured display、raw JSON collapsed。

边界：

- 不重做 runtime/config 架构。
- 不引入完整 Docker 参数管理体系。
- 不改部署/RunPlan 语义，除非发现当前详情展示依赖的字段明显错误。
- 优先前端 resolver/display fix；只有当前端无法判断 child value path 时，才最小修改后端 config-edit metadata。
- 不处理与本次 runtime details 展示无关的问题。

必须验收：

- UTS mode 不显示 `gpu_capabilities`。
- Network mode 不显示 `gpu_capabilities`。
- Shared memory 显示 `16gb`。
- IPC mode 显示 `host`。
- Model mount 显示 `/models (read-only)` 或等价结构化文本。
- Env 显示 `CUDA_VISIBLE_DEVICES = {{vendor_visible_devices}}`。
- Health check 显示 HTTP `/v1/models`、timeout `120s`、status `200`。
- Entrypoint 显示 `vllm serve`。
- Command 显示 `--model {{model_container_path}}`。
- Raw Config Set JSON 默认收起。

运行测试：

```bash
go test ./internal/server/...
cd web && npm test
cd web && npm run build
```

完成后提交并推送，输出：根因、修改文件、测试结果、commit id、push 结果、`git status --short`。
