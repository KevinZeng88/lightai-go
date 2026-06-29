# LightAI Go Codex Repair Package

This package contains a self-contained Codex execution prompt and supporting review notes for the runtime template / node runtime configuration / model deployment UX and RunPlan defects reported on 2026-06-29.

Recommended use:

1. Put this directory under the project docs area, for example:
   `docs/reports/phase-3/runtime-ux-runplan-repair/`
2. Paste `00-codex-autofix-prompt.md` into Codex.
3. Ask Codex to execute from the current branch, produce code fixes, run tests, commit, push, and provide the requested final report.

Primary file: `00-codex-autofix-prompt.md`.
