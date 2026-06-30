# 验收清单：配置渲染与向导操作

## 1. 页面验收

在中文界面打开：

1. 节点运行配置 → 新增/编辑 → 镜像与参数。
2. 模型部署向导 → 参数/运行配置相关步骤。
3. 运行模板/Backend Runtime 编辑页。
4. 模型库中涉及参数编辑的页面。

逐项确认：

- 结构化参数字段显示业务 label。
- 已知字段无“配置项”。
- tooltip/help 可见。
- checkbox 取消启用后，输入框仍显示。
- 保存/预览/RunPlan 结果仍按 enabled 状态生效。
- 长表单滚动时，主要操作按钮持续可见。
- `取消 / 上一步 / 下一步 / 保存 / 保存并检测` 操作区统一。

## 2. 字段 label spot check

至少确认以下中文 label 可见：

- 镜像
- GPU 显存利用率
- 最大上下文长度
- 数据类型
- 张量并行数
- 最大批处理 Token 数
- 流水线并行数
- CPU Offload 容量
- 模型下载目录
- 强制 eager 模式
- 监听地址
- KV Cache 数据类型
- 最大并发序列数
- 模型路径
- 服务端口
- Docker 选项
- 模型挂载
- 环境变量
- 服务监听地址
- 容器端口
- 附加启动参数
- 启动命令
- 设备绑定
- 入口命令
- 附加环境变量
- 启动方式
- 端口映射
- 服务模型名
- 卷挂载
- 健康检查

## 3. 自动化测试验收

必须至少包含以下测试类别：

- schema normalizer unit tests。
- ConfigField / ConfigEditView component tests。
- RunnerConfigsPage integration tests。
- ModelDeploymentsPage 或等价部署向导 integration tests。
- i18n/label leak audit。

推荐命令：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go

git status --short

go test ./...

cd web
npm test
npm run test:unit
npm run build
```

## 4. 文档验收

修复后生成：

- 审查文档：`docs/reports/phase-3/runtime-config-rendering-wizard-usability-review.md`
- closeout 文档：`docs/reports/phase-3/runtime-config-rendering-wizard-usability-closeout.md`

closeout 必须包含：

- 根因。
- 公共链路。
- 修复范围。
- 测试结果。
- 页面/DOM evidence。
- commit id。
- push 结果。
- git status。

## 5. 拒收条件

出现任一情况即拒收：

- 已知字段仍显示“配置项”。
- 只修了节点运行配置页面，其他公共配置入口仍分叉。
- 按钮只在当前页面局部移动，未形成公共向导操作区。
- 测试未覆盖公共 normalizer。
- closeout 没有记录真实根因和验证证据。
- 未提交或未推送。
