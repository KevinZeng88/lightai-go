# LightAI Bootstrap — Password Environment Variable Contract

> Status: CLOSED — Implementation complete (2026-06-25)
> Version: 0.1.23
> Scope: Server, scripts, E2E, bootstrap tool, documentation
> Final closeout: `docs/engineering/bootstrap/bootstrap-final-closeout.md`

---

## 1. Canonical Password Variables

| Variable | Purpose | Used By |
|----------|---------|---------|
| `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` | Initial admin password for clean-DB first start. Server reads this when creating the admin user. | Server (clean DB), bootstrap (first login attempt) |
| `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` | Target/final admin password. Bootstrap changes admin password to this value after initial login. Subsequent runs use this for login directly. | Bootstrap tool, E2E scripts, start scripts (backward compat) |

**They are NOT the same variable.** `INITIAL_PASSWORD` is for server initialization. `ADMIN_PASSWORD` is for the bootstrap tool's target state.

---

## 2. Why Two Variables?

1. **Clean DB startup**: Server creates admin user with `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD`. If unset, falls back to `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` (legacy compat). If both unset, auto-generates and writes to `runtime/initial-credentials.txt`.

2. **Bootstrap tool flow**:
   - First run: tries `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` (may already be set). If login fails, tries `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` or reads `runtime/initial-credentials.txt`.
   - Detects `must_change_password: true` → calls change-password API with `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD`.
   - Subsequent runs: logs in directly with `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD`.

3. **E2E tests**: Set `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` for clean-DB fixture. Set `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` for existing-env mode login.

---

## 3. Server Password Resolution (at InitBootstrap)

```
1. cfg.Password (hardcoded, usually "")
2. env: LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD  ← canonical
3. env: LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD    ← legacy backward compat (with WARN log)
4. existing runtime/initial-credentials.txt  ← reuse, don't regenerate
5. auto-generate random 32-char hex + write to credentials file
```

Implementation: `internal/server/auth/bootstrap.go` `InitBootstrap()`.
Steps 1-3 are env-var checks. Step 4 calls `readPasswordFromCredentialsFile()`.
Step 5 only triggers if all previous steps yield no password.
Credentials file is created with 0600 and NOT overwritten on subsequent restarts.

---

## 4. Bootstrap Password Resolution Priority

### Server-Side Password Resolution (at InitBootstrap)

```
1. cfg.Password (hardcoded, usually empty)
2. env: LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD  ← canonical
3. env: LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD    ← legacy WARN
4. existing runtime/initial-credentials.txt  ← reuse
5. auto-generate + write credentials file
```

### Bootstrap Initial Password (for first login on fresh server)

```
1. --initial-password CLI flag
2. env: LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD
3. profile.auth.initial_password_env → env var
4. --initial-password-file
5. profile.auth.initial_password_file
6. runtime/initial-credentials.txt (server-generated or auto-generated on first start)
7. profile.auth.initial_password
8. auto-generate (fallback, creates credentials file)
```

### Target Password (for change-password after initial login)

```
1. --admin-password CLI flag
2. env: LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD
3. profile.auth.final_password_env → env var
4. --admin-password-file
5. profile.auth.final_password_file
```

---

## 5. Profile Auth Fields

```yaml
auth:
  username: admin
  initial_password_env: LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD
  initial_password: ""        # hardcoded fallback (NOT recommended)
  initial_password_file: ""
  initial_password_runtime_files:
    - auto                    # runtime/initial-credentials.txt
  final_password_env: LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD
  final_password_file: ""
```

---

## 6. Server / Bootstrap / E2E Behavior Contract

| Scenario | Server | Bootstrap | E2E |
|----------|--------|-----------|-----|
| Clean DB, INITIAL_PASSWORD set | Uses INITIAL_PASSWORD, writes credentials file | — | Uses INITIAL_PASSWORD for login |
| Clean DB, only ADMIN_PASSWORD set | Falls back to ADMIN_PASSWORD (with WARN) | — | Uses ADMIN_PASSWORD for login |
| Clean DB, neither set | Auto-generates, writes to runtime/initial-credentials.txt | Reads from credentials file | Reads from credentials file |
| Server restart (DB exists, file exists) | Reads existing runtime/initial-credentials.txt (reuse, no re-generate) | Uses ADMIN_PASSWORD for login | Uses ADMIN_PASSWORD for login |
| Existing DB (admin user exists) | No effect (INSERT OR IGNORE) | Uses ADMIN_PASSWORD for login | Uses ADMIN_PASSWORD for login |
| must_change_password=true | — | Changes to ADMIN_PASSWORD via API | — |
| must_change_password=true, ADMIN_PASSWORD missing | — | FAIL: missing final password | — |

---

## 7. Credential File Format

### `runtime/initial-credentials.txt`

```
Username: admin
Password: <hex-string>
```

- Created by server on first admin user creation (clean DB)
- 0600 permissions
- NOT overwritten on subsequent starts
- Path: `$PROJECT_ROOT/runtime/initial-credentials.txt`

### `runtime/reset-credentials.txt`

```
LightAI Server - Admin Password Reset

New password for user 'admin':
    <new-password>

Please change the password after first login if not already done.
```

- Created by `scripts/reset-password.sh`
- 0600 permissions

---

## 8. Security Rules

1. **Never log passwords** — Server, agent, and scripts MUST NOT log password values, password hashes, tokens, CSRF tokens, or session IDs.
2. **Credentials files are 0600** — Both `initial-credentials.txt` and `reset-credentials.txt` are created with restrictive permissions.
3. **No hardcoded passwords in production** — The `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` env var MUST be set for production deployments.
4. **Output files MUST NOT contain plaintext passwords** — `auth.json`, `effective-config.json`, `bootstrap-state.json`, logs MUST NOT contain password values.
5. **Grafana has its own variable** — `LIGHTAI_GRAFANA_ADMIN_PASSWORD` is separate from the LightAI admin password. Do not mix.

---

## 9. Legacy Variable Audit Results

| Old Variable | Replacement | Status |
|---|---|---|
| `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` (as initial password) | `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` | **DEPRECATED** for initial-password use. Still supported as fallback with WARN log. Retained as canonical target-password variable for bootstrap. |

**No other conflicting password variable names found** in the codebase.

Variables that are correctly scoped and unchanged:
- `LIGHTAI_GRAFANA_ADMIN_PASSWORD` — Grafana admin (separate system)
- `LIGHTAI_E2E_PASSWORD` — E2E test fixture fallback

---

## 10. Installation Examples

### Clean DB first start with explicit passwords

```bash
export LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD='MyInitPass123!'
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='MyFinalPass456!'
./scripts/start-server.sh
./scripts/lightai-bootstrap.sh  # changes admin password from init to final
```

### Clean DB first start with auto-generated password

```bash
./scripts/start-server.sh
# Server outputs: "Initial credentials written to runtime/initial-credentials.txt"
cat runtime/initial-credentials.txt
# Username: admin
# Password: 3f8a2b1c...
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='MyNewPass789!'
./scripts/lightai-bootstrap.sh  # reads initial password from file, changes to final
```

### Reset DB and re-initialize

```bash
rm -f data/lightai.db runtime/initial-credentials.txt
export LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD='ResetPass123!'
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='ResetPass123!'  # same for simplicity
./scripts/start-server.sh
./scripts/lightai-bootstrap.sh
```

### E2E test with clean DB

```bash
export LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD='test1234'
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='test1234'
bash scripts/e2e-real-smoke-all-three.sh
```

---

## 11. Related Documents

- `docs/RUNBOOK-LOCAL-VERIFY.md` — Local verification runbook
- `docs/09-auth-tenant-design.md` — Auth and tenant architecture
- `internal/server/auth/bootstrap.go` — Server bootstrap implementation
- `cmd/server/main.go` — Server entrypoint and config
- `scripts/e2e/lib/env.sh` — E2E environment setup
