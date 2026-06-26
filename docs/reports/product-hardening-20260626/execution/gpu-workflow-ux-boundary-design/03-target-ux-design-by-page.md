# 03 — Target UX Design by Page

## 1. Runtime Templates page

### Product purpose

Let users understand which runtime schemes the platform supports.

### Main table columns

Recommended columns:

```text
Display name
GPU vendor
Inference backend
Backend version
Default image
Supported model formats
Ready node configs
Managed by
Actions
```

### Display name rule

Use:

```text
<vendor>.<backend> [version]
```

Examples:

```text
nvidia.vllm
nvidia.sglang
nvidia.llama.cpp b9700
metax.vllm
metax.sglang
```

### Grouping rule

If the backend catalog contains multiple records that look like implementation variants, group or present them as variants under a single user-facing runtime template.

Example:

```text
nvidia.vllm
  default image: vllm/vllm-openai:latest
  variants: latest, v0.23.0, v0.23.0-cu129
```

### Primary details drawer

Show:

- User-facing name
- Vendor
- Backend
- Version
- Image
- Supported model formats
- Supported task types
- Default ports
- Common runtime parameters
- Associated ready node runtime configs

### Advanced diagnostics drawer section

Collapsed by default:

- ConfigSet JSON
- Source metadata
- Raw IDs
- Catalog source

### Avoid in primary UI

- raw backend_runtime_id as name
- raw backend_version_id as primary label
- `launcher.*`
- `runtime_env.*`
- `{{MODEL_CONTAINER_PATH}}`
- raw ConfigSet keys

---

## 2. Node Runtime Configs page

### Product purpose

Let users enable and verify a runtime environment on a node.

### Main table columns

```text
Config name
Node
GPU vendor
Backend
Runtime template
Image
Status
Last checked
Warnings
Actions
```

### Status semantics

```text
ready                 → deployable
ready_with_warnings   → deployable with caution
needs_check           → not deployable; run check
missing_image         → not deployable; image issue
error                 → not deployable; show error
unknown               → not deployable; refresh/check
```

### New Node Runtime Config wizard

#### Step 1 — Select node

Use shared `NodeSelectorTable`.

Show:

- Node name / hostname
- Status / agent online
- GPU vendor
- GPU model
- GPU count
- Docker/runtime capability if available

Actions:

- Refresh

#### Step 2 — Select runtime template

Use runtime template cards/table.

Show:

- `nvidia.sglang` style name
- Backend
- Version
- Default image
- Supported formats
- Description

Actions:

- Refresh

#### Step 3 — Name, image, and common parameters

Fields:

```text
Config name
Image
Shared memory
Health check timeout
GPU visible devices
Backend common fields
Backend-specific fields
```

Do not render raw ConfigSet keys as normal fields.

#### Step 4 — Save and check

Show summary:

```text
Node
Runtime template
Config name
Image
Selected GPU settings
Key backend parameters
```

Actions:

```text
Save
Save and Check
Refresh Check Result
Finish
```

Behavior:

- Save failure: stay open, show error.
- Check failure: stay open, show error.
- Check result not ready: stay open, show status and fix hints.
- Ready / ready_with_warnings: show finish action and refresh parent list.

---

## 3. Model Library page

### Product purpose

Let users register model files and model facts.

### Main table columns

```text
Model display name
Format
Task type
Quantization
Size
Context length
Location count
Capabilities
Actions
```

### Model add/scan wizard

#### Step 1 — Select model file node

Use shared `NodeSelectorTable`, but label the step:

```text
Select node where model files are stored
```

Show:

- Node name
- Online status
- Model root paths if available
- File browsing availability

#### Step 2 — Browse path

Use RemoteFileBrowser.

#### Step 3 — Scan and confirm

Show:

- Detected model format
- Architecture
- Quantization
- Context length
- Model task type
- Model path
- Warnings
- Display name

### Design rule

Model Library should never show Docker/image/runtime/GPU serve parameters as model configuration.

---

## 4. Model Deployments page

### Product purpose

Let users combine a model and a ready node runtime config to create a deployment.

### Deployment wizard

#### Step 1 — Select model

Show model facts and location availability.

#### Step 2 — Select node runtime config

Show NBRs grouped by node and backend.

Selectable:

```text
ready
ready_with_warnings
```

Visible but disabled:

```text
needs_check
missing_image
error
unknown
```

#### Step 3 — Service settings

Show:

- Host port
- Container port
- Served model name
- Optional deployment-level runtime overrides

#### Step 4 — Preview Run Plan

Call `/deployments/preview`.

Show:

- Can run
- Errors
- Warnings
- Model path mapping
- GPU device binding
- Docker image
- Ports
- Volumes
- Key backend args
- Source trace

#### Step 5 — Save / Start

Errors keep the dialog open.

### Compatibility rule

Deployment requires a compatible pair:

```text
ModelLocation.node_id == NodeBackendRuntime.node_id
```

If the selected model has no location on the NBR's node, show a clear error.

---

## 5. Model Instances page

### Product purpose

Let users observe actual runtime state.

Show:

- Instance name
- Deployment
- Model
- Node
- Backend
- Status
- Start time
- Health state
- Logs
- Stop action

Avoid:

- Raw internal IDs as primary labels
- Static JSON as main view

---

## 6. Error behavior standard

All wizards must follow:

```text
Validation error → stay on same step, mark field/section
API error        → stay open, show backend error
Async check      → show pending/not-ready/ready status
Save success     → show result; close only when action semantics say completion
```

A dialog should never flash closed when the operation failed or when the result is ambiguous.

