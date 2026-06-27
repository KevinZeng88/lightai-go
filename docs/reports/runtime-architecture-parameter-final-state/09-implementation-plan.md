# Implementation Plan

## 1. 执行原则

不新建分支；不做旧 DB / 旧 API / 旧 snapshot 兼容；允许 fresh DB rebuild；旧逻辑与最终模型冲突时删除旧逻辑；每批次必须可验证、可提交、可回滚；不允许直接大 AUTORUN，必须先按本文分批执行。

## 2. Batch 0 — Baseline and inventory

确认工作区、未提交修改、现有 ConfigSet/config_set_json/ConfigEdit/RunPlan/preview/start 路径。特别保护 `deploy/observability/grafana/provisioning/dashboards/dashboards.yaml`，除非证明与本阶段相关，否则不得纳入提交。

验收：

```bash
git status --short
git log --oneline -10
grep -R "config_set_json\|config_overrides_json\|ConfigEdit\|RuntimeParameterEditor\|RunPlan" -n internal web configs docs | tee docs/reports/runtime-architecture-parameter-final-state/evidence/batch-0-inventory.txt
```

## 3. Batch 1 — Final ConfigSetBundle domain model

落地 ConfigSetBundle / ConfigSet / ConfigItem 字段分级；明确 schema/value/state/provenance/snapshot/presentation；清理旧混合语义；建立 fresh DB schema 或 clean JSON shape；更新 seed/catalog 到最终模型。

## 4. Batch 2 — Copy-on-create and local edits

实现每层 ConfigSetBundle copy-on-create、inherited snapshot 深拷贝、schema/snapshot 只读、value/state 当前层可修改、local_edits/provenance/source_chain 记录。

## 5. Batch 3 — ConfigSet presentation and renderer

实现 ConfigView / ConfigPanel、GenericConfigSetRenderer、own_sections、child_slots；清理旧 RuntimeParameterEditor / legacy editor 冲突；保留 custom renderer registry，但不得绕过 ConfigItem 规则。

## 6. Batch 4 — Shared RunPlan builder and source map

实现唯一 builder；preview/preflight/dry-run/start 共用；实现 parameter_source_map；Docker 子字段纳入 ConfigItem；清理旧 enabled_fields 长期兼容。

## 7. Batch 5 — API-first E2E

fresh DB；vLLM / SGLang / llama.cpp 至少覆盖 RunPlan preview；NVIDIA real smoke；evidence 沉淀。

## 8. Batch 6 — Final cleanup and closeout

删除旧字段、旧 UI、旧 fallback；更新文档；生成 closeout；commit + push；git status clean 或只剩明确外部文件。
