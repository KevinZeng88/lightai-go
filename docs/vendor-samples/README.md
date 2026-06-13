# GPU 厂商采集样例说明

本目录用于保存 NVIDIA 和 MetaX Collector 的脱敏测试夹具。不得提交客户现场或测试机器的未脱敏真实输出。

## NVIDIA

建议在真实 NVIDIA 环境采集：

```bash
nvidia-smi --query-gpu=index,name,uuid,pci.bus_id,driver_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw --format=csv,noheader,nounits
nvidia-smi --query-gpu=index,name,uuid,pci.bus_id,driver_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw --format=csv
nvidia-smi -q
nvidia-smi --version
```

建议脱敏后保存为：

```text
docs/vendor-samples/nvidia/query-success.csv
docs/vendor-samples/nvidia/query-empty.csv
docs/vendor-samples/nvidia/query-partial.csv
docs/vendor-samples/nvidia/query-error.txt
docs/vendor-samples/nvidia/version.txt
```

机器可读的 query CSV 是解析器首选输入。`nvidia-smi -q` 只用于诊断和补充确认，不作为默认解析格式。

## MetaX

MetaX 工具名称、安装路径、命令参数和机器可读格式必须在测试环境确认后记录。不要根据其他厂商工具猜测 `mx-smi` 的实际接口。

现场采样至少覆盖：

```text
工具路径和版本
设备枚举成功输出
指标采集成功输出
无设备输出
部分字段缺失输出
命令失败输出和退出码
命令超时行为
机器可读格式（如厂商工具支持）
```

建议脱敏后保存为：

```text
docs/vendor-samples/metax/device-success.<ext>
docs/vendor-samples/metax/metrics-success.<ext>
docs/vendor-samples/metax/empty.<ext>
docs/vendor-samples/metax/partial.<ext>
docs/vendor-samples/metax/error.txt
docs/vendor-samples/metax/version.txt
```

Collector 实现必须以这些真实样例为准。优先解析厂商提供的 JSON、CSV 或其他机器可读格式；只有不存在机器可读格式时才解析稳定的文本表格。

## 脱敏要求

提交前必须替换或删除：

1. 主机名、IP、MAC；
2. GPU UUID、序列号、资产编号；
3. 用户名、客户名、项目名；
4. 客户目录、挂载路径和内部域名；
5. Token、证书、账号和其他凭据。

脱敏时保留字段数量、字段顺序、分隔符、单位、空值和异常形态。解析器依赖的结构不得被简化。

缺失或不支持的指标必须映射为 unknown/nil。禁止为了让样例完整而补写不存在的值。
