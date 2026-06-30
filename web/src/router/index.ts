import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/pages/LoginPage.vue'),
    },
    {
      path: '/change-password',
      name: 'ChangePassword',
      component: () => import('@/pages/ChangePasswordPage.vue'),
    },
    {
      path: '/',
      component: () => import('@/layouts/ConsoleLayout.vue'),
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: () => import('@/pages/DashboardPage.vue'),
        },
        {
          path: 'nodes',
          name: 'Nodes',
          component: () => import('@/pages/NodesPage.vue'),
        },
        {
          path: 'gpus',
          name: 'GPUs',
          component: () => import('@/pages/GpusPage.vue'),
        },
        // [Phase1] old model runtime pages removed — new pages added in Phase 5
        {
          path: 'backends',
          name: 'Backends',
          component: () => import('@/pages/BackendsPage.vue'),
        },
        {
          path: 'runtimes',
          name: 'BackendRuntimes',
          component: () => import('@/pages/BackendRuntimesPage.vue'),
          meta: { title: 'runtimes.title' },
        },
        {
          path: 'config-edit/templates',
          name: 'ConfigEditTemplates',
          component: () => import('@/pages/ConfigEditTemplatesPage.vue'),
        },
        {
          path: 'runner-configs',
          name: 'RunnerConfigs',
          component: () => import('@/pages/RunnerConfigsPage.vue'),
          meta: { title: 'runnerConfigs.title' },
        },
        {
          path: 'models/artifacts',
          name: 'ModelArtifacts',
          component: () => import('@/pages/ModelArtifactsPage.vue'),
        },
        {
          path: 'models/deployments',
          name: 'ModelDeployments',
          component: () => import('@/pages/ModelDeploymentsPage.vue'),
        },
        {
          path: 'models/instances',
          name: 'ModelInstances',
          component: () => import('@/pages/ModelInstancesPage.vue'),
        },
        {
          path: 'models/test-diagnostics',
          name: 'ModelTestDiagnostics',
          component: () => import('@/pages/ModelInstancesPage.vue'),
        },
        {
          path: 'observability/overview',
          name: 'ObservabilityOverview',
          component: () => import('@/pages/ObservabilityOverviewPage.vue'),
        },
        {
          path: 'observability/targets',
          name: 'MetricsTargets',
          component: () => import('@/pages/ObservabilityTargetsPage.vue'),
        },
        {
          path: 'observability/prometheus',
          name: 'Prometheus',
          component: () => import('@/pages/PrometheusPage.vue'),
        },
        {
          path: 'system/tenants', name: 'Tenants', component: () => import('@/pages/TenantsPage.vue') }, { path: 'system/users', name: 'Users', component: () => import('@/pages/UsersPage.vue') }, { path: 'system/roles', name: 'Roles', component: () => import('@/pages/RolesPage.vue') }, { path: 'system/audit-logs', name: 'AuditLogs', component: () => import('@/pages/AuditLogsPage.vue') }, { path: 'observability/grafana',
          name: 'Grafana',
          component: () => import('@/pages/GrafanaPage.vue'),
        },
      ],
    },
  ],
})

router.beforeEach(async (to, _from, next) => {
  const auth = useAuthStore()
  if (!auth.isLoggedIn) {
    try {
      await auth.fetchMe()
    } catch {
      // ignore fetch errors
    }
  }
  if (!auth.isLoggedIn && to.path !== '/login') {
    next('/login')
  } else if (auth.isLoggedIn && auth.mustChangePassword && to.path !== '/change-password') {
    next('/change-password')
  } else {
    next()
  }
})

export default router
