<template>
  <div class="dashboard">
    <div class="page-header">
      <h2>{{ t('dashboard.title') }}</h2>
      <div class="header-actions">
        <span class="last-update" v-if="lastUpdate">{{ t('common.lastUpdated') }}: {{ formatDateTime(lastUpdate) }}</span>
        <el-button size="small" :icon="RefreshRight" @click="refresh" :loading="loading">{{ t('common.refresh') }}</el-button>
      </div>
    </div>

    <!-- refresh error banner -->
    <el-alert v-if="refreshError" :title="t('dashboard.refreshError')" type="warning" show-icon closable @close="refreshError = false" style="margin-bottom: 16px" />

    <!-- ====== 1. Resource overview cards ====== -->
    <el-row :gutter="16" class="metric-row">
      <el-col :xs="12" :sm="8" :md="4" v-for="card in cards" :key="card.label">
        <MetricCard :title="card.label" :value="card.value" :unit="card.unit" :description="card.desc" />
      </el-col>
    </el-row>

    <!-- ====== 2. Node overview ====== -->
    <el-card class="section-card">
      <template #header>
        <div class="section-header">
          <span>{{ t('dashboard.nodeSummary') }}</span>
          <el-button text type="primary" size="small" @click="$router.push('/nodes')">{{ t('dashboard.viewAllNodes') }} →</el-button>
        </div>
      </template>
      <el-table :data="nodeList" v-loading="loading" size="small" empty-text="" style="width: 100%">
        <el-table-column prop="hostname" :label="t('nodes.hostname')" show-overflow-tooltip />
        <el-table-column prop="primary_ip" :label="t('nodes.primaryIp')" width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ row.primary_ip || '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('nodes.status')" width="80">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column :label="t('nodes.gpuCount')" width="100" align="center">
          <template #default="{ row }">{{ nodeGpuCount(row.id) }}</template>
        </el-table-column>
        <el-table-column :label="t('nodes.gpuMemory')" width="180">
          <template #default="{ row }">{{ nodeGpuMem(row.id) }}</template>
        </el-table-column>
        <el-table-column :label="t('nodes.lastHeartbeat')" width="100">
          <template #default="{ row }">{{ formatRelativeTime(row.last_heartbeat_at, locale) }}</template>
        </el-table-column>
        <template #empty>{{ t('dashboard.noNodes') }}</template>
      </el-table>
    </el-card>

    <!-- ====== 3. GPU resource overview ====== -->
    <el-card class="section-card">
      <template #header>
        <div class="section-header">
          <span>{{ t('dashboard.gpuSummary') }}</span>
          <el-button text type="primary" size="small" @click="$router.push('/gpus')">{{ t('dashboard.viewAllGpus') }} →</el-button>
        </div>
      </template>

      <!-- 3a. Top 5 GPU utilization -->
      <h4 class="sub-title">{{ t('dashboard.topUtilization') }}</h4>
      <el-table :data="topUtilGpus" v-loading="loading" size="small" empty-text="" style="width: 100%">
        <el-table-column prop="vendor" :label="t('gpus.vendor')" width="80" />
        <el-table-column prop="name" :label="t('gpus.name')" show-overflow-tooltip />
        <el-table-column prop="index" :label="t('gpus.index')" width="60" align="center" />
        <el-table-column :label="t('gpus.gpuUtilization')" width="160">
          <template #default="{ row }">
            <el-progress :percentage="row.gpu_utilization_percent ?? 0" :stroke-width="8" :show-text="true" />
          </template>
        </el-table-column>
        <el-table-column :label="t('gpus.health')" width="90">
          <template #default="{ row }"><StatusTag :status="row.health" /></template>
        </el-table-column>
        <template #empty>—</template>
      </el-table>

      <!-- 3b. Top 5 memory usage -->
      <h4 class="sub-title">{{ t('dashboard.topMemory') }}</h4>
      <el-table :data="topMemGpus" v-loading="loading" size="small" empty-text="" style="width: 100%">
        <el-table-column prop="vendor" :label="t('gpus.vendor')" width="80" />
        <el-table-column prop="name" :label="t('gpus.name')" show-overflow-tooltip />
        <el-table-column prop="index" :label="t('gpus.index')" width="60" align="center" />
        <el-table-column :label="t('gpus.memory')" width="200">
          <template #default="{ row }">
            <el-progress :percentage="memPercent(row)" :stroke-width="8" :show-text="true">
              <span>{{ formatGB(row.memory_used_bytes) }} / {{ formatGB(row.memory_total_bytes) }}</span>
            </el-progress>
          </template>
        </el-table-column>
        <el-table-column :label="t('gpus.health')" width="90">
          <template #default="{ row }"><StatusTag :status="row.health" /></template>
        </el-table-column>
        <template #empty>—</template>
      </el-table>

      <!-- 3c. Abnormal GPUs -->
      <h4 class="sub-title">{{ t('dashboard.abnormalGpuList') }}</h4>
      <div v-if="abnormalGpus.length === 0" class="empty-hint">{{ t('dashboard.noAbnormalGpu') }}</div>
      <el-table v-else :data="abnormalGpus" v-loading="loading" size="small" empty-text="" style="width: 100%">
        <el-table-column prop="vendor" :label="t('gpus.vendor')" width="80" />
        <el-table-column prop="name" :label="t('gpus.name')" show-overflow-tooltip />
        <el-table-column prop="index" :label="t('gpus.index')" width="60" align="center" />
        <el-table-column :label="t('gpus.health')" width="100">
          <template #default="{ row }"><StatusTag :status="row.health" /></template>
        </el-table-column>
        <el-table-column :label="t('gpus.status')" width="100">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <template #empty>—</template>
      </el-table>
    </el-card>

    <!-- ====== 4. Collection / diagnostics summary ====== -->
    <el-card class="section-card">
      <template #header>
        <span>{{ t('dashboard.collectionStatus') }}</span>
      </template>
      <el-descriptions :column="3" border size="small">
        <el-descriptions-item :label="t('dashboard.agentLastReport')">
          {{ latestHeartbeat ? formatRelativeTime(latestHeartbeat, locale) : '—' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.latestCollection')">
          {{ latestCollected ? formatRelativeTime(latestCollected, locale) : '—' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.heartbeatTimeout')">
          <StatusTag :status="heartbeatOk ? 'healthy' : 'warning'" />
          {{ heartbeatOk ? '正常' : nodesWithStaleHeartbeat + ' 个节点超时' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.collectorError')">
          <StatusTag :status="hasUnhealthyGpus ? 'warning' : 'healthy'" />
          {{ hasUnhealthyGpus ? unhealthyGpuCount + ' 个 GPU 异常' : '正常' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.staleData')">
          <StatusTag :status="hasStaleGpus ? 'warning' : 'healthy'" />
          {{ hasStaleGpus ? staleGpuCount + ' 个 GPU 数据过期' : '正常' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.collectorStatus')">
          {{ t('dashboard.collectorStatusUnavailable') }}
        </el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { fetchNodes, type Node } from '@/api/nodes'
import { fetchGPUs, type GPU } from '@/api/gpus'
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { formatBytes, formatDateTime, formatRelativeTime, formatGB } from '@/utils/format'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const { t, locale } = useI18n()

const nodes = ref<Node[]>([])
const gpus = ref<GPU[]>([])

const { loading, lastUpdate, refreshError, refresh } = useAutoRefresh(async () => {
  const [n, g] = await Promise.all([fetchNodes(), fetchGPUs()])
  nodes.value = n
  gpus.value = g
})

// ---- Derived: all GPUs ----
const allGpus = computed(() => gpus.value)
const allNodes = computed(() => nodes.value)

// ---- Top 5 GPU views ----
const topUtilGpus = computed(() =>
  [...allGpus.value]
    .filter(g => g.gpu_utilization_percent != null)
    .sort((a, b) => (b.gpu_utilization_percent ?? 0) - (a.gpu_utilization_percent ?? 0))
    .slice(0, 5)
)

const topMemGpus = computed(() =>
  [...allGpus.value]
    .filter(g => g.memory_total_bytes > 0)
    .sort((a, b) => {
      const pa = a.memory_total_bytes > 0 ? a.memory_used_bytes / a.memory_total_bytes : 0
      const pb = b.memory_total_bytes > 0 ? b.memory_used_bytes / b.memory_total_bytes : 0
      return pb - pa
    })
    .slice(0, 5)
)

const abnormalGpus = computed(() =>
  allGpus.value.filter(g => g.health !== 'healthy' || g.status === 'unavailable')
)

// ---- Overview cards ----
const cards = computed(() => {
  const n = allNodes.value
  const g = allGpus.value
  const onlineCount = n.filter(x => x.status === 'online').length
  const healthyCount = g.filter(x => x.health === 'healthy').length
  const availableCount = g.filter(x => x.status === 'available').length
  const abnormalCount = g.filter(x => x.health !== 'healthy' || x.status === 'unavailable').length
  const totalMem = g.reduce((s, x) => s + (x.memory_total_bytes || 0), 0)
  const usedMem = g.reduce((s, x) => s + (x.memory_used_bytes || 0), 0)
  const freeMem = totalMem - usedMem
  const avgUtil = g.length > 0
    ? g.reduce((s, x) => s + (x.gpu_utilization_percent || 0), 0) / g.length
    : 0
  const maxUtil = g.length > 0
    ? Math.max(...g.map(x => x.gpu_utilization_percent || 0))
    : 0
  const maxMemPct = g.length > 0
    ? Math.max(...g.map(x => x.memory_total_bytes > 0 ? ((x.memory_used_bytes / x.memory_total_bytes) * 100) : 0))
    : 0
  const maxTemp = g.length > 0
    ? Math.max(...g.map(x => x.temperature_celsius ?? 0))
    : 0

  return [
    { label: t('dashboard.nodesTotal'), value: String(n.length), unit: '', desc: t('dashboard.onlineNodes') + ': ' + onlineCount },
    { label: t('dashboard.gpusTotal'), value: String(g.length), unit: '', desc: t('dashboard.healthyGpus') + ': ' + healthyCount + ' / ' + t('dashboard.abnormalGpus') + ': ' + abnormalCount },
    { label: t('dashboard.totalGpuMemory'), value: formatBytes(totalMem), unit: '', desc: t('dashboard.usedGpuMemory') + ': ' + formatBytes(usedMem) + ' / ' + t('dashboard.freeGpuMemory') + ': ' + formatBytes(freeMem) },
    { label: t('dashboard.avgGpuUtilization'), value: avgUtil.toFixed(1), unit: '%', desc: t('dashboard.maxGpuUtilization') + ': ' + maxUtil.toFixed(1) + '%' },
    { label: t('dashboard.maxMemUtilization'), value: maxMemPct.toFixed(1), unit: '%', desc: t('dashboard.availableGpus') + ': ' + availableCount },
    { label: t('dashboard.maxTemperature'), value: maxTemp > 0 ? maxTemp.toFixed(1) : '—', unit: maxTemp > 0 ? '°C' : '', desc: g.length > 0 ? '' : t('dashboard.noGpus') },
  ]
})

// ---- Node helpers ----
function nodeGpuCount(nodeId: string): string {
  const count = allGpus.value.filter(g => g.node_id === nodeId).length
  return String(count)
}

function nodeGpuMem(nodeId: string): string {
  const nodeGpus = allGpus.value.filter(g => g.node_id === nodeId)
  if (nodeGpus.length === 0) return '—'
  const used = nodeGpus.reduce((s, g) => s + (g.memory_used_bytes || 0), 0)
  const total = nodeGpus.reduce((s, g) => s + (g.memory_total_bytes || 0), 0)
  return formatGB(used) + ' / ' + formatGB(total)
}

// Top 5 nodes by GPU count
const nodeList = computed(() =>
  [...allNodes.value]
    .sort((a, b) => {
      const ga = allGpus.value.filter(g => g.node_id === a.id).length
      const gb = allGpus.value.filter(g => g.node_id === b.id).length
      return gb - ga
    })
    .slice(0, 5)
)

// ---- Diagnostic helpers ----
const latestHeartbeat = computed(() => {
  const onlineNodes = allNodes.value.filter(n => n.last_heartbeat_at)
  if (onlineNodes.length === 0) return null
  return onlineNodes.reduce((max, n) => (n.last_heartbeat_at! > max ? n.last_heartbeat_at! : max), '')
})

const latestCollected = computed(() => {
  const gpusWithTime = allGpus.value.filter(g => g.collected_at)
  if (gpusWithTime.length === 0) return null
  return gpusWithTime.reduce((max, g) => (g.collected_at! > max ? g.collected_at! : max), '')
})

// Nodes with heartbeat older than 20 seconds
const STALE_HEARTBEAT_MS = 20000
const nodesWithStaleHeartbeat = computed(() => {
  const now = Date.now()
  return allNodes.value.filter(n => {
    if (!n.last_heartbeat_at) return false
    return now - new Date(n.last_heartbeat_at).getTime() > STALE_HEARTBEAT_MS
  }).length
})
const heartbeatOk = computed(() => nodesWithStaleHeartbeat.value === 0)

// GPU health/status diagnostics
const unhealthyGpuCount = computed(() => allGpus.value.filter(g => g.health !== 'healthy').length)
const hasUnhealthyGpus = computed(() => unhealthyGpuCount.value > 0)

const staleGpuCount = computed(() => {
  const now = Date.now()
  // Stale if collected_at is older than 30 seconds
  return allGpus.value.filter(g => {
    if (!g.collected_at) return true // never collected
    return now - new Date(g.collected_at).getTime() > 30000
  }).length
})
const hasStaleGpus = computed(() => staleGpuCount.value > 0)

// ---- Helpers ----
function memPercent(gpu: GPU): number {
  if (!gpu.memory_total_bytes || gpu.memory_total_bytes === 0) return 0
  return Math.round((gpu.memory_used_bytes / gpu.memory_total_bytes) * 100)
}
</script>

<style scoped>
.dashboard {
  max-width: 1400px;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.page-header h2 {
  margin: 0;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.last-update {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
.metric-row {
  margin-bottom: 16px;
}
.section-card {
  margin-bottom: 16px;
}
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.sub-title {
  margin: 12px 0 8px;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}
.sub-title:first-child {
  margin-top: 0;
}
.empty-hint {
  padding: 24px;
  text-align: center;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}
</style>
