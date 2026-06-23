# Batch 6: Web / i18n / Permission UX — Detailed Plan

---

## Goal
Fix frontend security, i18n, and permission UX issues.

## Fix Matrix

| # | Issue | File:Line | Fix |
|---|-------|-----------|-----|
| 1 | No route guard | router/index.ts | Add `router.beforeEach` auth check |
| 2 | Hardcoded Chinese | DashboardPage.vue:126,130,134 | Add i18n keys for status values |
| 3 | Hardcoded Chinese | modelCapabilities.js:164-235 | Pass locale, add i18n keys |
| 4 | Grafana credentials | GrafanaPage.vue:7 | Remove or admin-only |
| 5 | Grafana credentials | locales/zh-CN.ts:322, en-US.ts:322 | Remove credential strings |
| 6 | Permission reset | RolesPage.vue:81-90 | Load existing perms on dialog open |
| 7 | Hardcoded ports | GrafanaPage.vue:27, PrometheusPage.vue:23 | Configurable |
| 8 | No confirmation | ModelDeploymentsPage.vue:577, ModelInstancesPage.vue:380 | Add ElMessageBox.confirm |
| 9 | Stale cache | useNodeLabels.ts:6 | Reset loaded flag or add TTL |
| 10 | Unused permissions | stores/auth.ts:28-29 | Wire hasPermission() helper |

## Route Guard Design
```typescript
router.beforeEach(async (to, from, next) => {
  const auth = useAuthStore()
  if (!auth.isLoggedIn) { await auth.fetchMe() }
  if (!auth.isLoggedIn && to.path !== '/login') { next('/login') }
  else if (auth.mustChangePassword && to.path !== '/change-password') { next('/change-password') }
  else { next() }
})
```

## Commits

1. `feat: add router.beforeEach auth guard`
2. `fix: remove Grafana default credentials`
3. `fix: move hardcoded Chinese to i18n keys`
4. `fix: RolesPage loads existing permissions`

## Non-Regression

| Check | Method |
|-------|--------|
| Login works | Browser: login → console |
| No login loop | Navigate to /dashboard → loads |
| No i18n key leakage | Switch to en-US → no raw keys |
| No credentials shown | GrafanaPage → no admin/admin |
| Confirmation dialog | Stop deployment → dialog appears |
| npm test passes | cd web && npm test |
