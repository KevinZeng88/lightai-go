<template>
  <div class="dashboard">
    <h2>{{ t('dashboard.title') }}</h2>

    <el-row :gutter="16" class="metric-row">
      <el-col :xs="12" :sm="6" :md="4" v-for="card in cards" :key="card.title">
        <MetricCard :title="card.title" :value="card.value" :unit="card.unit" :description="card.desc" />
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :span="12">
        <el-card>
          <template #header>{{ t('dashboard.nodeSummary') }}</template>
          <el-table :data="nodes" v-loading="loading" size="small" empty-text="">
            <el-table-column prop="hostname" :label="t('nodes.hostname')" />
            <el-table-column prop="status" :label="t('nodes.status')" width="100">
              <template #default="{ row }"><StatusTag :status="row.status" /></template>
            </el-table-column>
            <el-table-column prop="advertised_address" :label="t('nodes.address')" />
            <el-table-column :label="t('nodes.lastHeartbeat')" width="160">
              <template #default="{ row }">{{ formatDateTime(row.last_heartbeat_at) }}</template>
            </el-table-column>
            <template #empty>{{ t('nodes.noNodes') }}</template>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>{{ t('dashboard.gpuSummary') }}</template>
          <el-table :data="gpus" v-loading="loading" size="small" empty-text="">
            <el-table-column prop="vendor" :label="t('gpus.vendor')" width="80" />
            <el-table-column prop="name" :label="t('gpus.name')" show-overflow-tooltip />
            <el-table-column :label="t('gpus.memoryUsed')" width="140">
              <template #default="{ row }">
                <div class="memory-cell">
                  <el-progress :percentage="memPercent(row)" :stroke-width="8" :show-text="false" />
                  <span class="memory-text">{{ formatBytes(row.memory_used_bytes) }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column :label="t('gpus.gpuUtilization')" width="120">
              <template #default="{ row }">
                <span v-if="row.gpu_utilization_percent != null">
                  <el-progress :percentage="row.gpu_utilization_percent" :stroke-width="8" />
                </span>
                <span v-else>N/A</span>
              </template>
            </el-table-column>
            <el-table-column :label="t('gpus.health')" width="90">
              <template #default="{ row }"><StatusTag :status="row.health" /></template>
            </el-table-column>
            <template #empty>{{ t('gpus.noGpus') }}</template>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <div class="last-update" v-if="lastUpdate">
      {{ t('common.lastUpdated') }}: {{ formatDateTime(lastUpdate) }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { fetchNodes, type Node } from '@/api/nodes'
import { fetchGPUs, type GPU } from '@/api/gpus'
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { formatBytes, formatDateTime } from '@/utils/format'

const { t } = useI18n()

const nodes = ref<Node[]>([])
const gpus = ref<GPU[]>([])
const loading = ref(false)
const lastUpdate = ref('')

onMounted(async () => {
  loading.value = true
  try {
    const [n, g] = await Promise.all([fetchNodes(), fetchGPUs()])
    nodes.value = n
    gpus.value = g
    if (g.length > 0) {
      const latest = (g || []).reduce((max, gpu) => {
        if (gpu.collected_at && gpu.collected_at > max) return gpu.collected_at
        return max
      }, '')
      lastUpdate.value = latest
    }
  } catch { /* handled by component state */ }
  loading.value = false
})

const cards = computed(() => {
  const n = nodes.value
  const g = gpus.value
  const onlineCount = n.filter(x => x.status === 'online').length
  const healthyCount = g.filter(x => x.health === 'healthy').length
  const totalMem = (g || []).reduce((s, x) => s + (x.memory_total_bytes || 0), 0)
  const usedMem = (g || []).reduce((s, x) => s + (x.memory_used_bytes || 0), 0)
  const freeMem = totalMem - usedMem
  const avgGpuUtil = g.length > 0
    ? (g || []).reduce((s, x) => s + (x.gpu_utilization_percent || 0), 0) / (g || []).length || 0
    : 0
  const avgMemUtil = g.length > 0
    ? (g || []).reduce((s, x) => s + (x.memory_utilization_percent || 0), 0) / (g || []).length || 0
    : 0

  return [
    { title: t('dashboard.nodesTotal'), value: n.length, unit: '', desc: '' },
    { title: t('dashboard.onlineNodes'), value: onlineCount, unit: '', desc: t('dashboard.offlineNodes') + ': ' + (n.length - onlineCount) },
    { title: t('dashboard.gpusTotal'), value: g.length, unit: '', desc: t('dashboard.healthyGpus') + ': ' + healthyCount },
    { title: t('dashboard.totalGpuMemory'), value: formatBytes(totalMem), unit: '', desc: t('dashboard.usedGpuMemory') + ': ' + formatBytes(usedMem) },
    { title: t('dashboard.avgGpuUtilization'), value: avgGpuUtil.toFixed(1), unit: '%', desc: '' },
    { title: t('dashboard.avgMemoryUtilization'), value: avgMemUtil.toFixed(1), unit: '%', desc: '' },
  ]
})

function memPercent(gpu: GPU): number {
  if (!gpu.memory_total_bytes) return 0
  return Math.round((gpu.memory_used_bytes / gpu.memory_total_bytes) * 100)
}
</script>

<style scoped>
.dashboard h2 {
  margin-bottom: 16px;
}
.metric-row {
  margin-bottom: 8px;
}
.last-update {
  margin-top: 12px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
.memory-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
.memory-text {
  font-size: 12px;
  white-space: nowrap;
}
</style>
