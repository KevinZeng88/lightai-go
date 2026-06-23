# Stop Conditions and Recovery

---

## Stop Conditions

MiMo MUST stop and report when ANY of these occur:

### 1. Golden Path Broken
Golden Path critical flow broken AND auto-recovery failed twice.
**Action**: Revert last commit, write failure closeout, list broken flows.

### 2. NBR Parameters Blocked
Repair causes NBR-defined parameters (privileged, ipc, devices, security-opt, group-add) to be blocked or dropped.
**Action**: Revert last commit, write failure closeout.

### 3. Default Tenant/Admin Flow Blocked
Default tenant user or platform admin cannot access resources they should.
**Action**: Revert last commit, write failure closeout.

### 4. Server Cannot Reach Agent
After AgentClient changes, server proxy to agent fails for localhost/private IP.
**Action**: Revert last commit, write failure closeout.

### 5. Unrelated Working Tree Changes
`git status` shows changes to files not in current batch scope.
**Action**: Investigate origin, stash or revert unrelated changes.

### 6. Tests Fail After Two Fix Attempts
Test failure persists after two fix cycles.
**Action**: Write failure closeout with evidence, mark as STOPPED.

### 7. Environment Dependency Missing
Need Docker/GPU/model but not available, and no mock substitute.
**Action**: Skip that verification, note in closeout, continue with available verifications.

### 8. Out-of-Scope Change Needed
Fix requires changing files in a different batch's scope.
**Action**: Note in closeout, either defer or adjust batch boundary.

### 9. Contradicts Decisions
Implementation would contradict a decided item (NBR boundary, no vendor policy, etc.).
**Action**: Stop, write failure closeout, flag contradiction.

### 10. Security Level Error
Finding priority明显错误, needs plan revision before proceeding.
**Action**: Stop, write report, flag for user review.

---

## Recovery Procedure

### Step 1: Revert if needed
```bash
git log --oneline -3  # identify commit to revert
git revert HEAD --no-edit  # or specific SHA
```

### Step 2: Preserve evidence
- Copy test output
- Copy git diff
- Copy error messages

### Step 3: Write failure closeout
Use closeout template with Status: STOPPED.

### Step 4: List failed commands
```bash
# What was run
# What failed
# Error output
```

### Step 5: Propose next safe step
- If revertable: suggest different approach
- If environment: suggest mock/dry-run alternative
- If contradiction: suggest plan revision

### Step 6: Do not continue
If P0 batch (1A, 1B, 2) fails, do NOT proceed to later batches.

---

## Batch Failure Impact

| Failed Batch | Blocks |
|-------------|--------|
| 1A (Tenant) | Cannot proceed to any batch (security foundation) |
| 1B (AgentClient) | Cannot proceed to 1C, 2, 3 (proxy broken) |
| 1C (Agent Endpoint) | Can proceed to 2, 3 (agent auth optional) |
| 2 (Docker Lifecycle) | Cannot proceed to 7 (test infra depends on clean lifecycle) |
| 3 (I/O Safety) | Can proceed (independent) |
| 4 (RunPlan) | Can proceed (independent) |
| 6 (Web UI) | Can proceed (independent) |
| 7 (Tests) | Can proceed (final batch) |
