<template>
  <div class="gpus-page">
    <div class="page-header">
      <h2>{{ t('gpus.title') }}</h2>
      <el-button @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
    </div>

    <div class="toolbar">
      <el-select v-model="vendorFilter" :placeholder="t('gpus.vendor')" clearable size="small" style="width: 120px">
        <el-option label="NVIDIA" value="nvidia" />
        <el-option label="MetaX" value="metax" />
      </el-select>
      <el-select v-model="healthFilter" :placeholder="t('gpus.health')" clearable size="small" style="width: 120px; margin-left: 8px">
        <el-option :label="t('status.healthy')" value="healthy" />
        <el-option :label="t('status.warning')" value="warning" />
        <el-option :label="t('status.unhealthy')" value="unhealthy" />
      </el-select>
      <el-input v-model="search" :placeholder="t('common.search')" clearable size="small" style="width: 240px; margin-left: 8px" />
    </div>

    <el-table :data="filteredGPUs" v-loading="loading" size="small" @row-click="openDetail">
      <el-table-column prop="health" :label="t('gpus.health')" width="90">
        <template #default="{ row }"><StatusTag :status="row.health" /></template>
      </el-table-column>
      <el-table-column prop="vendor" :label="t('gpus.vendor')" width="80" />
      <el-table-column prop="name" :label="t('gpus.name')" show-overflow-tooltip />
      <el-table-column prop="index" :label="t('gpus.index')" width="60" />
      <el-table-column :label="t('gpus.uuid')" width="100">
        <template #default="{ row }">
          <span class="mono">{{ row.uuid?.substring(0, 12) }}...</span>
          <CopyButton :text="row.uuid" />
        </template>
      </el-table-column>
      <el-table-column :label="t('gpus.memory')" width="220">
        <template #default="{ row }">
          <div class="mem-bar">
            <el-progress :percentage="memPercent(row)" :stroke-width="10" :show-text="false" />
            <span class="mem-text">{{ formatBytes(row.memory_used_bytes) }} / {{ formatBytes(row.memory_total_bytes) }}</span>
            <span class="mem-free">{{ t('gpus.free') }}: {{ formatBytes(row.memory_free_bytes) }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('gpus.gpuUtilization')" width="130">
        <template #default="{ row }">
          <span v-if="row.gpu_utilization_percent != null">
            <el-progress :percentage="row.gpu_utilization_percent" :stroke-width="8" :show-text="true" />
          </span>
          <span v-else>N/A</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('gpus.temperature')" width="100">
        <template #default="{ row }">{{ formatCelsius(row.temperature_celsius) }}</template>
      </el-table-column>
      <el-table-column :label="t('gpus.powerDraw')" width="80">
        <template #default="{ row }">{{ formatWatts(row.power_draw_watts) }}</template>
      </el-table-column>
      <el-table-column :label="t('gpus.collectedAt')" width="160">
        <template #default="{ row }">{{ formatDateTime(row.collected_at) }}</template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="drawerVisible" :title="t('gpus.detail')" size="550px">
      <template v-if="selectedGPU">
        <el-descriptions :column="1" border size="small" :title="t('gpus.deviceInfo')">
          <el-descriptions-item :label="t('gpus.name')">{{ selectedGPU.name }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.vendor')">{{ selectedGPU.vendor }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.index')">{{ selectedGPU.index }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.uuid')">
            <span class="mono">{{ selectedGPU.uuid }}</span>
            <CopyButton :text="selectedGPU.uuid" />
          </el-descriptions-item>
          <el-descriptions-item :label="t('gpus.pciBusId')">{{ selectedGPU.pci_bus_id }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.driverVersion')">{{ selectedGPU.driver_version }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.status')"><StatusTag :status="selectedGPU.health" /></el-descriptions-item>
        </el-descriptions>

        <el-descriptions :column="2" border size="small" :title="t('gpus.memoryInfo')" style="margin-top: 16px">
          <el-descriptions-item :label="t('gpus.memoryTotal')">{{ formatBytes(selectedGPU.memory_total_bytes) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.memoryUsed')">{{ formatBytes(selectedGPU.memory_used_bytes) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.memoryFree')">{{ formatBytes(selectedGPU.memory_free_bytes) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.memoryUtilization')">{{ formatPercent(selectedGPU.memory_utilization_percent) }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions :column="2" border size="small" :title="t('gpus.thermalInfo')" style="margin-top: 16px">
          <el-descriptions-item :label="t('gpus.gpuUtilization')">{{ formatPercent(selectedGPU.gpu_utilization_percent) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.temperature')">{{ formatCelsius(selectedGPU.temperature_celsius) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.powerDraw')">{{ formatWatts(selectedGPU.power_draw_watts) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.collectedAt')">{{ formatDateTime(selectedGPU.collected_at) }}</el-descriptions-item>
        </el-descriptions>

        <el-collapse style="margin-top: 16px">
          <el-collapse-item :title="t('common.rawJson')">
            <pre class="raw-json">{{ JSON.stringify(selectedGPU, null, 2) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { fetchGPUs, type GPU } from '@/api/gpus'
import StatusTag from '@/components/StatusTag.vue'
import CopyButton from '@/components/CopyButton.vue'
import { formatBytes, formatDateTime, formatPercent, formatCelsius, formatWatts } from '@/utils/format'

const { t } = useI18n()

const gpus = ref<GPU[]>([])
const loading = ref(false)
const search = ref('')
const vendorFilter = ref('')
const healthFilter = ref('')
const drawerVisible = ref(false)
const selectedGPU = ref<GPU | null>(null)

const filteredGPUs = computed(() => {
  let result = gpus.value
  if (vendorFilter.value) result = result.filter(g => g.vendor === vendorFilter.value)
  if (healthFilter.value) result = result.filter(g => g.health === healthFilter.value)
  if (search.value) {
    const q = search.value.toLowerCase()
    result = result.filter(g =>
      g.name.toLowerCase().includes(q) ||
      g.uuid.toLowerCase().includes(q) ||
      g.node_id.toLowerCase().includes(q)
    )
  }
  return result
})

async function refresh() {
  loading.value = true
  try { gpus.value = await fetchGPUs() } catch { /* */ }
  loading.value = false
}

function openDetail(row: GPU) {
  selectedGPU.value = row
  drawerVisible.value = true
}

function memPercent(gpu: GPU): number {
  if (!gpu.memory_total_bytes) return 0
  return Math.round((gpu.memory_used_bytes / gpu.memory_total_bytes) * 100)
}

onMounted(refresh)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.page-header h2 { margin: 0; }
.toolbar { margin-bottom: 12px; }
.mono { font-family: monospace; font-size: 12px; }
.mem-bar { display: flex; align-items: center; gap: 8px; }
.mem-text { font-size: 12px; white-space: nowrap; color: var(--el-text-color-secondary); }
.raw-json {
  max-height: 400px;
  overflow: auto;
  font-size: 12px;
  background: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
}
</style>
