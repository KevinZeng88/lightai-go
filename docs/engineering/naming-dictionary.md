# Naming Dictionary

Canonical vocabulary for LightAI Go product concepts. Use these terms in all user-facing UI, i18n, docs, and API descriptions.

## Core Concepts

| Internal Entity | zh-CN UI Label | en-US UI Label | Owner Layer | User-Editable | Copied to Next Layer |
|---|---|---|---|---|---|
| Backend | 推理后端 | Backend | System catalog | No | — |
| BackendVersion | 后端版本 | Backend Version | System catalog | No | — |
| **BackendRuntime** | **运行模板** | **Runtime Template** | User / system catalog | Yes (user-managed only) | Copied to NBR at enable time |
| **NodeBackendRuntime** | **节点运行配置** | **Node Runtime Config** | Node-specific | Yes (node overrides) | Copied to Deployment at creation |
| **ModelArtifact** | 模型 | Model | Model library | Yes | Referenced by Deployment |
| ModelLocation | 模型位置 | Model Location | Node-specific | System-managed | — |
| **ModelDeployment** | 模型部署 | Deployment | Deployment | Yes | — |
| **ModelInstance** | 模型实例 | Instance | Runtime | System-managed (lifecycle) | — |
| **ResolvedRunPlan** | **运行计划** | **Run Plan** | System-generated | No (read-only) | Frozen at deployment start |
| ConfigSet | 配置集 | ConfigSet | Internal | System-managed | Technical JSON label only |

## Forbidden Stale Terms

| Forbidden | Replacement |
|---|---|
| RunnerConfig (user-facing) | Node Runtime Config / 节点运行配置 |
| NBR (user-facing i18n) | Runtime template name or "运行模板快照" |
| RunPlan (raw English in zh-CN UI) | 运行计划 |
| ConfigSet (in UI section titles) | 技术配置 / Technical Config |
| backend_runtime_id (as displayed column value) | Resolved BackendRuntime display_name |

## Allowed Technical Abbreviations

| Abbreviation | Where Allowed |
|---|---|
| NBR | Internal code comments, Go variable names, DB column names, test files |
| RunPlan | Internal code, Go package name, DB table names |
| ConfigSet | Internal code, JSON keys, DB column names, advanced/debug panels |

## Usage Rules

1. **UI menus and page titles** must use the zh-CN/en-US labels from this dictionary.
2. **Table columns** must show display names (not raw UUIDs). Resolve IDs to display names via join or map.
3. **i18n strings** must not contain raw "NBR" or "RunPlan" (untranslated) in user-facing text.
4. **ConfigSet** may appear as a technical JSON label in debug/diagnostic panels but NOT as a primary UI section title.
5. **API descriptions** in OpenAPI docs must use the vocabulary from this dictionary.
6. **Historical evidence** documents (in `docs/reports/archive/`) are exempt from renaming.
