# MetaX Runtime Template / Device Binding Review

> Status: FIXED
> Created: 2026-06-21
> Finalized: 2026-06-21
> Scope: Correct MetaX device binding semantics per official MetaX documentation

## 1. MetaX Device Binding Semantics (Corrected)

### Official MetaX Docker Native Binding

MetaX 普通 Docker 原生命令模式下，指定卡主要依赖设备文件绑定：

- 必要设备：`/dev/mxcd`（MetaX 计算设备）
- 全量暴露：`/dev/dri`（包含所有 DRI 设备）
- 单卡绑定：`/dev/dri/cardX` + `/dev/dri/renderDXXX`
- 如果挂载整个 `/dev/dri`，则不是强约束到某张卡

### Visible Env Correction

- **没有 `METAX_VISIBLE_DEVICES`** — MetaX 论坛明确说明
- **没有 `MACA_VISIBLE_DEVICE`** — 不在 LightAI Go 中使用
- **应使用 `CUDA_VISIBLE_DEVICES`** — CUDA-compatible 框架/运行时的可见性控制
- `CUDA_VISIBLE_DEVICES` 只作为框架层可见性控制，不应描述为唯一隔离机制
- 强约束优先依赖设备文件绑定（`/dev/dri/cardX` + `/dev/dri/renderDXXX`）

### metax-docker Mode

- `metax-docker run --gpus=all / N / [ID1,ID2...]` 是另一种 runtime mode
- 当前只记录为后续可选支持方向
- 不混入当前普通 Docker device-path 模式

## 2. Changes Applied

| Change | Before | After |
|--------|--------|-------|
| `defaultVisibleEnvKey()` for metax | `"MACA_VISIBLE_DEVICE"` | `"CUDA_VISIBLE_DEVICES"` |
| MetaX YAML templates | `MACA_VISIBLE_DEVICE` | `CUDA_VISIBLE_DEVICES` |
| DB seed for MetaX runtime | `MACA_VISIBLE_DEVICE` | `CUDA_VISIBLE_DEVICES` |
| DB seed GPU visible env key | `"MACA_VISIBLE_DEVICE"` | `"CUDA_VISIBLE_DEVICES"` |
| metax_huawei_test.go | `MACA_VISIBLE_DEVICE` assertions | `CUDA_VISIBLE_DEVICES` assertions |
| metax_device_binding_test.go | `MACA_VISIBLE_DEVICE` checks | `CUDA_VISIBLE_DEVICES` checks |
| E2E scripts (3) | `MACA_VISIBLE_DEVICE` | `CUDA_VISIBLE_DEVICES` |

### Preserved MetaX Env Vars

- `MACA_SMALL_PAGESIZE_ENABLE` — legitimate MetaX env var
- `PYTORCH_ENABLE_PG_HIGH_PRIORITY_STREAM` — legitimate MetaX env var

## 3. Current MetaX Binding Capability

| Capability | Status |
|-----------|--------|
| Device file binding (`/dev/mxcd`, `/dev/dri`) | ✅ Implemented (template + resolver + Docker driver) |
| `CUDA_VISIBLE_DEVICES` for visible control | ✅ Implemented |
| Per-card device binding (`/dev/dri/cardX`, `/dev/dri/renderDXXX`) | ⚠️ Dry-run only — agent inventory lacks card/renderD mapping |
| `metax-docker --gpus` selector | 🔮 Future — not implemented |
| Strong per-card isolation | ⚠️ Requires agent inventory of MetaX device node mapping |
| MetaX real hardware validation | BLOCKED — requires MetaX GPU hardware |

## 4. Test Results

```
go test lightai-go/internal/server/api/... → ALL PASS
go test lightai-go/internal/server/runplan/... → ALL PASS
go vet → CLEAN
go build → OK
```

Key test assertions:
- `TestResolveMetaXRunPlanUsesRuntimeDockerOptions`: CUDA_VISIBLE_DEVICES=6,7, devices/group_add/privileged/security_opt ✅
- `TestNVIDIADifferentiatesFromMetaX`: NVIDIA ≠ MetaX device paths and env vars ✅
- `TestCPUModeSkipsAllGPUBinding`: CPU mode skips GPU checks ✅

## 5. Files Modified

| File | Change |
|------|--------|
| `internal/server/runplan/resolver.go` | `defaultVisibleEnvKey()` metax → "CUDA_VISIBLE_DEVICES" |
| `internal/server/db/db.go` | DB seed MACA → CUDA_VISIBLE_DEVICES |
| `configs/backend-catalog/runtimes/vllm/metax-docker.yaml` | MACA → CUDA_VISIBLE_DEVICES |
| `configs/backend-catalog/runtimes/sglang/metax-docker.yaml` | MACA → CUDA_VISIBLE_DEVICES |
| `configs/backend-catalog/runtimes/llamacpp/metax-docker.yaml` | MACA → CUDA_VISIBLE_DEVICES |
| `configs/backend-catalog/runtimes/sglang/metax-macart.yaml` | MACA → CUDA_VISIBLE_DEVICES |
| `internal/server/runplan/metax_huawei_test.go` | MACA → CUDA_VISIBLE_DEVICES assertions |
| `internal/server/api/metax_device_binding_test.go` | MACA → CUDA_VISIBLE_DEVICES checks |
| `scripts/e2e-runplan-parameter-source-audit.sh` | MACA → CUDA_VISIBLE_DEVICES |
| `scripts/e2e-matrix-verifier.sh` | MACA → CUDA_VISIBLE_DEVICES |
| `scripts/e2e-dryrun-parameter-matrix-enhanced.sh` | MACA → CUDA_VISIBLE_DEVICES |
| `docs/reports/phase-3/metax-runtime-template-device-binding-review.md` | This document |
