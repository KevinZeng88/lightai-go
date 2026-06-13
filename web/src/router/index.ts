import { createRouter, createWebHistory } from 'vue-router'

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
        {
          path: 'models/deployments',
          name: 'ModelDeployments',
          component: () => import('@/pages/PlaceholderPage.vue'),
        },
        {
          path: 'models/instances',
          name: 'ModelInstances',
          component: () => import('@/pages/PlaceholderPage.vue'),
        },
        {
          path: 'runtime/environments',
          name: 'RuntimeEnvironments',
          component: () => import('@/pages/PlaceholderPage.vue'),
        },
        {
          path: 'observability/targets',
          name: 'MetricsTargets',
          component: () => import('@/pages/ObservabilityTargetsPage.vue'),
        },
        {
          path: 'observability/grafana',
          name: 'Grafana',
          component: () => import('@/pages/PlaceholderPage.vue'),
        },
      ],
    },
  ],
})

export default router
