<template>
  <div>
    <h2>{{ t('nav.observability') }}</h2>
    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :span="8" v-for="item in cards" :key="item.title">
        <el-card>
          <template #header><span>{{ item.title }}</span></template>
          <div :class="['status-dot', item.status]"></div>
          <p>{{ item.desc }}</p>
          <el-button v-if="item.url" type="primary" size="small" @click="openUrl(item.url)">
            {{ item.btnText }}
          </el-button>
          <p v-if="!item.url" class="help-text">{{ item.help }}</p>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :span="24">
        <el-card>
          <template #header>{{ t('observability.dashboardShortcuts') }}</template>
          <el-space wrap>
            <el-button v-for="d in dashboards" :key="d.uid" @click="openUrl(d.url)" size="small">
              {{ d.name }}
            </el-button>
          </el-space>
        </el-card>
      </el-col>
    </el-row>

    <div class="last-check" v-if="lastCheck">{{ t('common.lastUpdated') }}: {{ lastCheck }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { fetchMetricsTargets } from '@/api/metrics'

const { t } = useI18n()
const targetsCount = ref(0)
const lastCheck = ref('')

const promUrl = `http://${window.location.hostname}:9090`
const grafanaUrl = `http://${window.location.hostname}:3000`

const checkProm = async () => {
  try { await fetch(promUrl + '/-/healthy'); return true } catch { return false }
}
const checkGrafana = async () => {
  try { await fetch(grafanaUrl + '/api/health'); return true } catch { return false }
}

const cards = ref([
  { title: 'Prometheus', status: 'unknown', desc: '', url: promUrl, btnText: 'Open Prometheus', help: '' },
  { title: 'Grafana', status: 'unknown', desc: '', url: grafanaUrl, btnText: 'Open Grafana', help: '' },
  { title: t('observability.targets'), status: 'ok', desc: '', url: '', btnText: '', help: '' },
])

const dashboards = [
  { name: 'LightAI Overview', uid: 'lightai-overview', url: grafanaUrl + '/d/lightai-overview' },
  { name: 'GPU Resources', uid: 'lightai-gpu-resources', url: grafanaUrl + '/d/lightai-gpu-resources' },
  { name: 'Agent Health', uid: 'lightai-agent-health', url: grafanaUrl + '/d/lightai-agent-health' },
]

onMounted(async () => {
  const [promOk, grafOk] = await Promise.all([checkProm(), checkGrafana()])
  cards.value[0].status = promOk ? 'ok' : 'down'
  cards.value[0].desc = promOk ? 'Running' : 'Not running. Start: bash scripts/observability-up.sh'
  cards.value[1].status = grafOk ? 'ok' : 'down'
  cards.value[1].desc = grafOk ? 'Running (admin/lightai)' : 'Not running. Start: bash scripts/observability-up.sh'
  try {
    const tgt = await fetchMetricsTargets()
    targetsCount.value = tgt.length
    cards.value[2].desc = `${tgt.length} targets`
  } catch { cards.value[2].desc = 'N/A' }
  lastCheck.value = new Date().toLocaleString()
})

function openUrl(url: string) { window.open(url, '_blank') }
</script>

<style scoped>
.status-dot { width: 12px; height: 12px; border-radius: 50%; display: inline-block; margin-right: 8px; }
.status-dot.ok { background: #67c23a; }
.status-dot.down { background: #f56c6c; }
.status-dot.unknown { background: #909399; }
.help-text { color: var(--el-text-color-secondary); font-size: 12px; margin-top: 8px; }
.last-check { margin-top: 12px; font-size: 12px; color: var(--el-text-color-placeholder); }
</style>
