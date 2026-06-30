# LightAI Go ConfigEdit Object Model — Codex Autonomous Execution Pack

This pack converts the architecture audit into an autonomous Codex execution plan.

Target repository path:

```text
/home/kzeng/projects/ai-platform-study/lightai-go
```

Target documentation directory inside the repository:

```text
docs/reports/phase-3/configedit-template-object-model-design/
```

Primary file to give Codex:

```text
docs/reports/phase-3/configedit-template-object-model-design/06-codex-autonomous-execution-master-prompt.md
```

Execution mode:

- Codex runs autonomously.
- Codex reads the existing design and audit documents.
- Codex follows the work packages in order.
- Codex self-audits each package.
- Codex fixes discovered issues immediately when they are fixable.
- Codex stops only for true blockers and documents them.
- Human review happens at the final closeout.

Recommended sync command if this pack is downloaded to `/tmp`:

```bash
cd /tmp
unzip -o lightai-configedit-object-model-codex-autonomous-execution-pack.zip

cd /home/kzeng/projects/ai-platform-study/lightai-go
mkdir -p docs/reports/phase-3/configedit-template-object-model-design
rsync -av /tmp/lightai-configedit-object-model-codex-autonomous-execution-pack/   docs/reports/phase-3/configedit-template-object-model-design/
```
