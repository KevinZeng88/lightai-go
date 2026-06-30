<template>
  <el-container class="console-layout">
    <el-aside width="220px" class="sidebar">
      <div class="logo">LightAI Go</div>
      <el-menu
        :default-active="activeMenu"
        router
        :collapse="false"
        background-color="#001529"
        text-color="#ffffffb3"
        active-text-color="#fff"
      >
        <el-menu-item index="/">
          <el-icon><Odometer /></el-icon>
          <span>{{ t('nav.dashboard') }}</span>
        </el-menu-item>

        <el-sub-menu index="resources">
          <template #title>
            <el-icon><Monitor /></el-icon>
            <span>{{ t('nav.resources') }}</span>
          </template>
          <el-menu-item index="/nodes">{{ t('nav.nodes') }}</el-menu-item>
          <el-menu-item index="/gpus">{{ t('nav.gpus') }}</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="models">
          <template #title>
            <el-icon><Box /></el-icon>
            <span>{{ t('nav.aiWorkflow') }}</span>
          </template>
          <el-menu-item index="/models/artifacts">{{ t('nav.modelLibrary') }}</el-menu-item>
          <el-menu-item index="/runner-configs">{{ t('runnerConfigs.title') }}</el-menu-item>
          <el-menu-item index="/models/deployments">{{ t('deployments.title') }}</el-menu-item>
          <el-menu-item index="/models/instances">{{ t('instances.title') }}</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="config">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>{{ t('nav.config') }}</span>
          </template>
          <el-menu-item index="/backends">{{ t('backends.title') }}</el-menu-item>
          <el-menu-item index="/runtimes">{{ t('runtimes.title') }}</el-menu-item>
          <el-menu-item index="/config-edit/templates">ConfigEdit Templates</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="system">
          <template #title><el-icon><Monitor /></el-icon><span>{{ t('nav.system') }}</span></template>
          <el-menu-item index="/system/tenants">{{ t('nav.tenants') }}</el-menu-item>
          <el-menu-item index="/system/users">{{ t('nav.users') }}</el-menu-item>
          <el-menu-item index="/system/roles">{{ t('nav.roles') }}</el-menu-item>
          <el-menu-item index="/system/audit-logs">{{ t('nav.auditLogs') }}</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="observability">
          <template #title>
            <el-icon><TrendCharts /></el-icon>
            <span>{{ t('nav.observability') }}</span>
          </template>
          <el-menu-item index="/observability/overview">{{ t('nav.overview') }}</el-menu-item>
          <el-menu-item index="/observability/targets">{{ t('nav.metricsTargets') }}</el-menu-item>
          <el-menu-item index="/observability/prometheus">{{ t('observability.prometheusTitle') }}</el-menu-item>
          <el-menu-item index="/observability/grafana">{{ t('observability.grafanaTitle') }}</el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="topbar">
        <div class="topbar-left">
          <span class="user-info" v-if="auth.user">
            <el-select v-model="currentTenantId" @change="doSwitchTenant" size="small" style="width:180px;margin-right:8px" v-if="tenantOpts.length > 1">
              <el-option v-for="t in tenantOpts" :key="t.id" :label="t.name" :value="t.id" />
            </el-select>
            {{ auth.user.display_name || auth.user.username }}
            <span v-if="auth.tenant">@ {{ auth.tenant.name }}</span>
          </span>
        </div>
        <div class="topbar-right">
          <LanguageSwitcher />
          <el-button text @click="doLogout" style="margin-left: 12px">
            {{ t('auth.logout') }}
          </el-button>
        </div>
      </el-header>

      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { apiClient } from '@/api/client'
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const currentTenantId = ref(auth.tenant?.id || '')
const tenantOpts = ref<{id:string,name:string}[]>([])

async function loadTenants() {
  try { const data: any = await apiClient.get('/tenants'); tenantOpts.value = Array.isArray(data) ? data.filter((t:any) => t.status === 'active') : [] }
  catch { tenantOpts.value = [] }
}
async function doSwitchTenant(tid: string) {
  try { await apiClient.post('/session/switch-tenant', { tenant_id: tid }); window.location.reload() }
  catch(e:any) { alert('Switch failed: ' + (e?.message || 'unknown')) }
}
loadTenants()

const activeMenu = computed(() => route.path)

onMounted(async () => {
  await auth.fetchMe()
  if (!auth.isLoggedIn) {
    router.replace('/login')
  } else if (auth.mustChangePassword && route.path !== '/change-password') {
    router.replace('/change-password')
  }
})

async function doLogout() {
  await auth.logout()
  router.replace('/login')
}
</script>

<style scoped>
.console-layout {
  min-height: 100vh;
}
.sidebar {
  background: #001529;
  overflow-y: auto;
}
.logo {
  height: 56px;
  line-height: 56px;
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  text-align: center;
  border-bottom: 1px solid #ffffff1a;
}
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--el-border-color-light);
  background: #fff;
}
.topbar-left {
  font-size: 14px;
}
.topbar-right {
  display: flex;
  align-items: center;
}
.user-info {
  color: var(--el-text-color-regular);
}
</style>
