# Copy Commands

Assuming the zip file is placed under `/tmp`:

```bash
cd /tmp
unzip -o lightai-codex-runtime-ux-runplan-repair.zip -d /tmp/lightai-codex-runtime-ux-runplan-repair
```

Copy the repair package into the LightAI Go project docs:

```bash
mkdir -p /home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/phase-3/runtime-ux-runplan-repair
rsync -av /tmp/lightai-codex-runtime-ux-runplan-repair/ /home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/phase-3/runtime-ux-runplan-repair/
```

Paste this prompt into Codex:

```bash
cat /home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/phase-3/runtime-ux-runplan-repair/00-codex-autofix-prompt.md
```
