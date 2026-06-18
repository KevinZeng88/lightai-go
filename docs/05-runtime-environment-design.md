> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference or historical compatibility document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Go 运行环境设计

## 1. 设计目标

运行环境用于描述模型实例启动时需要的执行环境、容器镜像、启动命令、环境变量、端口、挂载目录、GPU 设备映射和 Docker 高级参数。

第一阶段运行环境的目标是：

1. 可以创建 Docker 运行环境模板；
2. 可以编辑运行环境；
3. 可以删除未被引用的运行环境；
4. 可以在模型实例创建时选择运行环境；
5. 可以预览最终 Docker 启动命令；
6. 所有高级参数都必须显式启用；
7. 未启用的参数不出现在最终命令中；
8. 启动失败时可以从运行环境、实例参数和 Docker 命令快照定位问题。

运行环境不是模型实例。
运行环境只是模板，真正执行发生在模型实例启动时。

---

## 2. 设计原则

1. 第一阶段只支持 Docker 运行方式；
2. 后续可以扩展为 Docker、裸进程、Kubernetes、其他 runtime；
3. 不把 vLLM、SGLang、Ollama 等框架写死在平台逻辑中；
4. 不把 GPU 厂商参数写死在统一逻辑中；
5. 所有可选参数必须有启用开关；
6. Web 页面中未开启的参数，不允许出现在最终 Docker 命令；
7. Docker 命令必须可预览；
8. 实例启动时必须保存 Docker 命令快照；
9. 运行环境被模型或实例引用时，不允许直接删除；
10. 运行环境修改后，不自动影响已经运行的实例。

---

## 3. 运行环境类型

第一阶段只实现：

```text
docker
```

后续预留：

```text
process
kubernetes
custom
```

运行环境类型字段：

```go
RuntimeType string
```

建议枚举：

```text
docker
process
kubernetes
custom
```

第一阶段只允许创建 `docker` 类型。

---

## 4. RuntimeEnvironment 数据结构

```go
type RuntimeEnvironment struct {
    ID          string
    TenantID    string
    OwnerID     string
    CreatedBy   string
    UpdatedBy   string
    Name        string
    Description string
    Type        string
    Enabled     bool

    DockerImage string

    CommandEnabled bool
    Command        []string

    EntrypointEnabled bool
    Entrypoint        []string

    WorkingDirEnabled bool
    WorkingDir        string

    EnvEnabled bool
    Env        []EnvVar

    PortsEnabled bool
    Ports        []PortMapping

    VolumesEnabled bool
    Volumes        []VolumeMount

    DevicesEnabled bool
    Devices        []DeviceMapping

    GpuEnabled bool
    GpuMode    string

    NetworkEnabled bool
    NetworkMode    string

    IpcEnabled bool
    IpcMode    string

    UtsEnabled bool
    UtsMode    string

    PrivilegedEnabled bool
    Privileged        bool

    ShmSizeEnabled bool
    ShmSize        string

    UlimitEnabled bool
    Ulimits        []UlimitSpec

    SecurityOptEnabled bool
    SecurityOpts        []string

    ExtraArgsEnabled bool
    ExtraArgs        []string

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## 5. 子结构定义

### 5.1 EnvVar

```go
type EnvVar struct {
    Key   string
    Value string
}
```

示例：

```text
CUDA_VISIBLE_DEVICES=0,1
VLLM_USE_MODELSCOPE=true
```

注意：

1. 敏感环境变量后续要支持脱敏；
2. 第一阶段可以先明文保存；
3. 不允许空 key；
4. key 建议只允许字母、数字和下划线。

---

### 5.2 PortMapping

```go
type PortMapping struct {
    HostPort      int
    ContainerPort int
    Protocol      string
}
```

示例：

```text
8001:8000/tcp
```

规则：

1. `container_port` 必填；
2. `host_port` 可以为空，表示 Docker 自动分配；
3. 第一阶段建议要求显式 host_port，便于 endpoint 展示；
4. protocol 默认 `tcp`。

---

### 5.3 VolumeMount

```go
type VolumeMount struct {
    HostPath      string
    ContainerPath string
    ReadOnly      bool
}
```

示例：

```text
/data/models:/models:ro
/data/cache:/cache
```

规则：

1. host_path 和 container_path 必填；
2. container_path 必须是绝对路径；
3. host_path 是否存在由 Agent 启动时检查；
4. 缺失时启动失败，并记录清楚错误。

---

### 5.4 DeviceMapping

```go
type DeviceMapping struct {
    HostPath      string
    ContainerPath string
}
```

示例：

```text
/dev/dri:/dev/dri
/dev/mxcd:/dev/mxcd
/dev/infiniband:/dev/infiniband
```

规则：

1. 如果 container_path 为空，则默认等于 host_path；
2. 设备不存在时可以失败，也可以由配置决定是否忽略；
3. 第一阶段建议设备不存在即失败，避免模型启动后异常更难定位。

---

### 5.5 UlimitSpec

```go
type UlimitSpec struct {
    Name string
    Soft string
    Hard string
}
```

示例：

```text
memlock=-1
nofile=1048576
```

---

## 6. GPU 参数设计

GPU 参数不要直接绑定 NVIDIA。

字段：

```go
GpuEnabled bool
GpuMode    string
```

GpuMode 建议枚举：

```text
none
all
selected
vendor_specific
```

含义：

1. `none`：不传 GPU 参数；
2. `all`：使用全部 GPU；
3. `selected`：使用实例选择的 GPU；
4. `vendor_specific`：使用厂商自定义设备映射和环境变量。

第一阶段建议：

1. NVIDIA 可使用 `--gpus`；
2. 沐曦、昇腾、寒武纪等通过 `devices`、`env`、`extra_args` 表达；
3. RuntimeEnvironment 不保存具体 GPU ID，`selected` 的设备选择来自 ModelInstance；
4. 不在统一 DockerRunSpec 中硬编码所有厂商差异；
5. 厂商差异可以后续放入 RuntimePreset 或 GPUVendorAdapter。

---

## 7. DockerRunSpec

DockerRunSpec 是实例启动时生成的最终启动规格。

```go
type DockerRunSpec struct {
    Image       string
    Name        string
    Detached    bool

    Command     []string
    Entrypoint  []string
    Args        []string
    WorkingDir  string

    Env         []EnvVar
    Ports       []PortMapping
    Volumes     []VolumeMount
    Devices     []DeviceMapping

    GpuPolicy    string
    GpuDeviceIDs []string
    NetworkArgs  []string
    IpcArgs      []string
    UtsArgs      []string
    SecurityOpts []string
    Ulimits      []UlimitSpec
    ExtraArgs    []string
    Labels       map[string]string

    ShmSize     string
    Privileged  bool

    PreviewCommand string
}
```

DockerRunSpec 由以下信息组合生成：

```text
RuntimeEnvironment
Model
ModelInstance
Node
GPU selection
```

Server 使用统一生成器生成并冻结 DockerRunSpec，Preview API 和任务创建复用同一逻辑。StartInstanceTask 直接携带冻结规格，Agent 只校验和执行，不重新合并 Runtime、Model 或 Instance。

生成后必须保存到实例启动记录和任务中，便于排障。`Labels` 至少包含 `lightai.managed`、instance、node、operation 和 spec generation ownership labels。

Runtime 决定 entrypoint 和 command；Model 提供默认 `Args`；Instance 启用 args override 时整体替换 Model args。详细字段和算法见 `docs/08-engineering-contracts.md`。

---

## 8. 参数启用规则

所有可选参数必须满足：

```text
enabled=false → 不出现在 DockerRunSpec
enabled=true  → 校验字段并生成 Docker 参数
```

示例：

```text
ShmSizeEnabled=false
ShmSize="100gb"
```

最终命令中不能出现：

```text
--shm-size 100gb
```

只有当：

```text
ShmSizeEnabled=true
```

才允许出现。

这个规则适用于：

1. command；
2. entrypoint；
3. working_dir；
4. env；
5. ports；
6. volumes；
7. devices；
8. gpu；
9. network；
10. ipc；
11. uts；
12. privileged；
13. shm_size；
14. ulimit；
15. security_opt；
16. extra_args。

所有命令、args 和 extra args 必须保存为参数数组。禁止使用 shell 字符串执行，禁止把重定向、管道、命令替换等 shell 片段放入 `ExtraArgs`。

env、volume、device、port、args 和 extra args 的确定合并规则见 `docs/08-engineering-contracts.md`。

---

## 9. Docker 命令生成示例

输入：

```text
image: vllm/vllm-openai:latest
name: lightai-inst-001
ports: 8001:8000
volumes: /data/models:/models
env: CUDA_VISIBLE_DEVICES=0,1
shm-size: 100gb
ipc: host
privileged: true
```

输出：

```bash
docker run -d \
  --name lightai-inst-001 \
  -p 8001:8000 \
  -v /data/models:/models \
  -e CUDA_VISIBLE_DEVICES=0,1 \
  --shm-size 100gb \
  --ipc host \
  --privileged \
  vllm/vllm-openai:latest
```

如果 `privileged_enabled=false`，即使 `privileged=true`，最终命令也不允许出现 `--privileged`。

---

## 10. 运行环境 CRUD API

权限和租户规则：

1. 所有查询默认按 `session.current_tenant_id` 过滤；
2. 查询要求 `runtime:read`；
3. 创建、修改和删除要求 `runtime:write`；
4. 创建时 Server 写入 `tenant_id`、`owner_id`、`created_by`、`updated_by`，不信任客户端同名字段；
5. 更新时 Server 写入 `updated_by=session.current_user_id`；
6. 不允许查询、引用、修改或删除其他 tenant 的运行环境；
7. API 不比较 built-in 或 custom Role 名称，只检查实时解析的 permission code；
8. 第一阶段只有 default tenant 时，使用方式与单租户一致。

创建时固定写入：

```text
tenant_id = session.current_tenant_id
owner_id = session.current_user_id
created_by = session.current_user_id
updated_by = session.current_user_id
```

### 10.1 创建运行环境

```http
POST /api/runtime-environments
```

### 10.2 查询运行环境列表

```http
GET /api/runtime-environments
```

### 10.3 查询运行环境详情

```http
GET /api/runtime-environments/{id}
```

### 10.4 更新运行环境

```http
PUT /api/runtime-environments/{id}
```

### 10.5 删除运行环境

```http
DELETE /api/runtime-environments/{id}
```

删除规则：

1. 未被模型或实例引用，可以删除；
2. 已被模型默认引用，不允许删除；
3. 已被实例引用，不允许删除；
4. 可以先禁用，再迁移引用后删除；
5. 目标资源必须属于 `session.current_tenant_id`；
6. owner_id 是可转移业务所有者，不等于 created_by，也不直接决定删除权限；删除仍以 tenant scope 和 `runtime:write` 为准。

---

## 11. Docker 命令预览 API

```http
POST /api/runtime-environments/preview-docker-command
```

用途：

1. 创建运行环境时预览；
2. 创建实例前预览；
3. 排查参数是否正确；
4. 确认未启用参数不会出现在命令里；
5. 确认预览结果与实际任务中的冻结 DockerRunSpec 来自同一生成逻辑。

输入：

```json
{
  "runtime_environment": {},
  "model": {},
  "instance": {},
  "gpu_ids": ["0", "1"]
}
```

输出：

```json
{
  "command": "docker run ...",
  "warnings": []
}
```

---

## 12. 运行环境预设

后续可以提供 Runtime Preset。

第一阶段可以先不做，但文档预留：

```text
vLLM
SGLang
Ollama
Xinference
Custom Docker
MetaX vLLM
Ascend MindIE
```

Preset 只用于快速填充模板，保存后仍然是普通 RuntimeEnvironment。

---

## 13. Web 页面要求

运行环境页面应支持：

1. 列表；
2. 创建；
3. 编辑；
4. 启用 / 禁用；
5. 删除；
6. Docker 命令预览；
7. 参数分组展示。

参数分组：

```text
基础信息
镜像与命令
环境变量
端口
目录挂载
设备映射
GPU
网络
高级 Docker 参数
```

高级 Docker 参数默认折叠，避免普通用户误配置。

---

## 14. 校验规则

创建或更新运行环境时需要校验：

1. name 必填且唯一；
2. type 必须为 docker；
3. docker_image 必填；
4. enabled 字段必须明确；
5. env key 不允许为空；
6. port 不允许重复；
7. volume container_path 不允许重复；
8. device host_path 不允许为空；
9. shm_size 格式需要合法；
10. extra_args 必须是拆分后的参数数组，不允许 shell 运算符或 shell 片段；
11. ownership labels 不允许由用户覆盖；
12. 同一输入必须生成稳定的规范 DockerRunSpec。

第一阶段不做复杂安全沙箱，但要记录风险。

---

## 15. 日志与审计

运行环境变更需要记录：

1. 创建；
2. 修改；
3. 删除；
4. 禁用；
5. 启用；
6. Docker 命令预览；
7. 校验失败原因。

结构化日志同时记录：

```text
user_id
tenant_id
required_permission
runtime_environment_id
result
```

第一阶段可以先记录到 Server 日志。
后续再做操作审计表。

---

## 16. 测试要求

至少包含：

1. RuntimeEnvironment 校验测试；
2. DockerRunSpec 生成测试；
3. enabled=false 参数不生成测试；
4. env 生成测试；
5. port 生成测试；
6. volume 生成测试；
7. device 生成测试；
8. shm-size 生成测试；
9. privileged 开关测试；
10. 删除引用保护测试；
11. Preview 与任务冻结规格一致性测试；
12. ownership labels 测试；
13. shell 片段拒绝测试；
14. 参数合并确定性测试；
15. tenant scope 查询过滤测试；
16. 跨 tenant CRUD 拒绝测试；
17. 创建时归属字段由 Session 写入测试；
18. 缺少 `runtime:write` 时写操作拒绝、具有该 permission 时允许测试；
19. custom Role 的 runtime permission 生效测试。

---

## 17. MVP 完成标准

运行环境模块完成后，应达到：

1. 可以创建 Docker 运行环境；
2. 可以编辑 Docker 运行环境；
3. 可以禁用运行环境；
4. 可以删除未被引用的运行环境；
5. 可以预览 Docker 命令；
6. 未启用参数不出现在命令中；
7. 模型和实例可以引用运行环境；
8. Server 启动实例时可以生成并冻结 DockerRunSpec；
9. Agent 只执行冻结规格。
