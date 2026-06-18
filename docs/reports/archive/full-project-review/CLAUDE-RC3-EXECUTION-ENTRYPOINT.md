> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# CLAUDE EXECUTION ENTRYPOINT

Read and follow these documents in order:

1. `docs/reports/full-project-review/rc3-full-hardening-execution-plan.md`
2. `docs/reports/full-project-review/rc3-clean-baseline-scope.md`
3. `docs/reports/full-project-review/rc3-acceptance-criteria.md`
4. `docs/reports/full-project-review/rc3-verification-matrix.md`
5. `docs/reports/full-project-review/rc3-issue-closeout-register.md`
6. `docs/reports/full-project-review/web-workflow-acceptance-checklist.md`
7. `docs/reports/full-project-review/rc3-final-closeout-report.md`

Start at Stage 0.

Do not ask for confirmation.

Do not begin code changes until Stage 0 documentation is complete.

After Stage 0, continue through Stage 10 automatically.

You have highest permissions and Docker is available. Do not skip Docker/E2E/release/patch/start-stop/logging/Web/Go/shell validation except for true external hardware absence such as MetaX hardware not being accessible.

If any command fails, fix the cause and rerun it. Do not record failure and continue.

All destructive validation must use `/tmp/lightai-go-rc3-*` or test containers created for this work.

Final output must include branch, commit hash, and push result.
