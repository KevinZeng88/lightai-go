@AGENTS.md

# Claude Code Specific Instructions

Use plan mode before modifying cross-cutting Agent / Server / Web behavior.

Before changing code:

1. Read `AGENTS.md`.
2. Read `docs/README.md`.
3. Read `docs/PHASE-STATUS.md`.
4. Read the task-relevant topic documents.
5. Inspect the current implementation and actual API behavior.

When the user asks for implementation:

- Keep changes minimal.
- Prefer root-cause fixes over frontend fallback hacks.
- Update tests with the change.
- Do not expand into future API Key, billing, gateway, Kubernetes, Ray, or scheduler work unless explicitly requested.

After finishing, provide:

- Root cause.
- Modified files.
- Tests run.
- Verification commands.
- Before/after behavior.
- Remaining risks.
