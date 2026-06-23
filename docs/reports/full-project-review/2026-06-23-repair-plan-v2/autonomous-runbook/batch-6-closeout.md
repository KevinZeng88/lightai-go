# Batch 6 Closeout: Web / i18n / Permission UX

> Date: 2026-06-23
> Status: PASS

---

## Changes Made

| File | Changes |
|------|---------|
| web/src/router/index.ts | Route guard (beforeEach) |
| web/src/pages/GrafanaPage.vue | Credentials admin-only |
| web/src/pages/DashboardPage.vue | i18n for status strings |
| web/src/pages/RolesPage.vue | Load existing permissions |
| web/src/locales/zh-CN.ts | New i18n keys |
| web/src/locales/en-US.ts | New i18n keys |
| web/src/api/roles.ts | fetchRolePermissions function |
| internal/server/rbac/handlers.go | GET /roles/{id}/permissions endpoint |
| internal/server/api/router.go | New route registration |

### Commits
| SHA | Message |
|-----|---------|
| c6869fd | fix(web): route guard, credentials, i18n, permission loading |

---

## After Verification

- **npm run build**: PASS
- **npm test**: PASS
- **go build**: PASS

---

## Stop Conditions

None triggered.
