# Closeout Template

每批 closeout 使用以下模板。

```markdown
# Batch N Closeout

## Batch Goal

## Scope Completed

## Files Changed

| File | Change |
| --- | --- |

## Risks Closed

| ID | Status | Evidence |
| --- | --- | --- |

## Questions Resolved

| ID | Decision | Evidence |
| --- | --- | --- |

## Validation Commands

| Command | Result | Summary |
| --- | --- | --- |

## Runtime Evidence

| Evidence | Path / Summary |
| --- | --- |

## Failures and Fixes

## Skips

| Item | Reason | Required Evidence to Close |
| --- | --- | --- |

## Commit

```bash
git status --short
git add ...
git commit -m "..."
git push
```

Commit ID:

Push result:

Final `git status --short`:

## Remaining Issues

Only include issues that are truly blocked by external dependency. R-001 到 R-015 不允许使用 `INTENTIONALLY_DEFERRED_WITH_OWNER_AND_ACCEPTANCE`、`future`、`follow-up`、`later`、`manual verification later`。
```

## Final Closeout Requirements

最终 closeout 必须包含：

| Risk | Final Status | Closing Commit | Evidence |
| --- | --- | --- |
| R-001 |  |  |  |
| R-002 |  |  |  |
| R-003 |  |  |  |
| R-004 |  |  |  |
| R-005 |  |  |  |
| R-006 |  |  |  |
| R-007 |  |  |  |
| R-008 |  |  |  |
| R-009 |  |  |  |
| R-010 |  |  |  |
| R-011 |  |  |  |
| R-012 |  |  |  |
| R-013 |  |  |  |
| R-014 |  |  |  |
| R-015 |  |  |  |

状态值：

- CLOSED
- CLOSED_BY_SCOPE_REDUCTION
- BLOCKED_BY_EXTERNAL_DEPENDENCY

禁止使用：

- TODO
- future
- follow-up
- later
- manual verification later
- INTENTIONALLY_DEFERRED_WITH_OWNER_AND_ACCEPTANCE
- partially done without next action

状态定义：

- `CLOSED`：代码/测试/文档/脚本全部闭环，验证通过。
- `CLOSED_BY_SCOPE_REDUCTION`：该能力不做，但 UI/API/docs 不再声称支持，相关入口已隐藏、禁用或拒绝。
- `BLOCKED_BY_EXTERNAL_DEPENDENCY`：只有外部资源确实不可用，且有命令级证据。

本项目当前不需要兼容旧 DB、旧 API、旧 payload、旧脚本、旧运行模板、旧快照。Closeout 中不能把“保留兼容路径”当成修复完成。如果某设计 item 不实现，只能通过 scope reduction 关闭，不能模糊延期。
